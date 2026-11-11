package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	dbpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	instancepb "cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type queueRead struct {
	ID, SQL, Status string
	Rows            int
	StatsScope      string
	Error           string   `json:",omitempty"`
	Renewed         bool     `json:",omitempty"`
	QueryStatsKeys  []string `json:",omitempty"`
}
type managedClients struct {
	Client         *spanner.Client
	DatabaseClient *database.DatabaseAdminClient
	name           string
	close          func() error
}

func (c *managedClients) DatabasePath() string { return c.name }
func (c *managedClients) Close() error         { return c.close() }

func managedTarget() (string, error) {
	if os.Getenv("SPANNER_EMULATOR_HOST") != "" {
		return "", fmt.Errorf("managed probe refuses emulator override")
	}
	p, i := os.Getenv("SPANNER_PROJECT_ID"), os.Getenv("SPANNER_INSTANCE_ID")
	if p == "" || i == "" {
		return "", fmt.Errorf("dedicated project and instance environment are required")
	}
	return "projects/" + p + "/instances/" + i, nil
}
func managedPreflight() error {
	parent, err := managedTarget()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := instance.NewInstanceAdminClient(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	i, err := c.GetInstance(ctx, &instancepb.GetInstanceRequest{Name: parent})
	if err != nil {
		return fmt.Errorf("managed instance preflight: %s", status.Code(err))
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "OK", "instance_state": i.State.String(), "instance_type": i.InstanceType.String(), "operation": "read_only_preflight"})
}

// Create is never retried as a new lifecycle. Only the successfully created
// random database is owned; no existing database or instance is dropped.
func openManagedFixture(ctx context.Context, ddl []string, interceptor grpc.UnaryClientInterceptor, r *result, journalPath string, descriptors []byte) (*managedClients, error) {
	parent, err := managedTarget()
	if err != nil {
		return nil, err
	}
	admin, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		return nil, err
	}
	var nonce [8]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		_ = admin.Close()
		return nil, err
	}
	id := "mutation1110-" + hex.EncodeToString(nonce[:])
	db := parent + "/databases/" + id
	// The private exclusive journal survives an interrupted create/wait. A
	// rerun using this output path cannot accidentally start a second DB.
	j, err := os.OpenFile(journalPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		_ = admin.Close()
		return nil, err
	}
	err = json.NewEncoder(j).Encode(map[string]string{"database": db, "purpose": "one authorized temporary mutation-count fixture", "created_at": time.Now().UTC().Format(time.RFC3339Nano)})
	closeErr := j.Close()
	if err != nil || closeErr != nil {
		_ = admin.Close()
		return nil, fmt.Errorf("write private ownership journal: %v %v", err, closeErr)
	}
	op, err := admin.CreateDatabase(ctx, &dbpb.CreateDatabaseRequest{Parent: parent, CreateStatement: "CREATE DATABASE `" + id + "`", DatabaseDialect: dbpb.DatabaseDialect_GOOGLE_STANDARD_SQL})
	r.Lifecycle["create_request"] = status.Code(err).String()
	if err != nil {
		_ = admin.Close()
		return nil, fmt.Errorf("create managed fixture: %s", status.Code(err))
	}
	_, err = op.Wait(ctx)
	r.Lifecycle["create_wait"] = status.Code(err).String()
	if err != nil {
		check, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if _, checkErr := admin.GetDatabase(check, &dbpb.GetDatabaseRequest{Name: db}); checkErr == nil {
			dropErr := admin.DropDatabase(check, &dbpb.DropDatabaseRequest{Database: db})
			r.Lifecycle["drop_after_failed_wait"] = status.Code(dropErr).String()
			if dropErr != nil {
				r.CleanupErrors = append(r.CleanupErrors, "failed-wait drop: "+status.Code(dropErr).String())
			}
		} else {
			r.CleanupErrors = append(r.CleanupErrors, "create wait ownership unresolved; private journal retained")
		}
		_ = admin.Close()
		return nil, fmt.Errorf("create managed fixture wait: %s", status.Code(err))
	}
	owned := &managedClients{DatabaseClient: admin, name: db}
	owned.close = func() error {
		if owned.Client != nil {
			owned.Client.Close()
		}
		defer func() { _ = admin.Close() }()
		cleanup, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		err := admin.DropDatabase(cleanup, &dbpb.DropDatabaseRequest{Database: db})
		r.Lifecycle["drop"] = status.Code(err).String()
		if err != nil {
			return fmt.Errorf("drop owned fixture: %s", status.Code(err))
		}
		_, err = admin.GetDatabase(cleanup, &dbpb.GetDatabaseRequest{Name: db})
		r.Lifecycle["absence"] = status.Code(err).String()
		if status.Code(err) != codes.NotFound {
			return fmt.Errorf("owned fixture absence check: %s", status.Code(err))
		}
		return nil
	}
	ddlOp, err := admin.UpdateDatabaseDdl(ctx, &dbpb.UpdateDatabaseDdlRequest{Database: db, Statements: ddl, ProtoDescriptors: descriptors})
	if err == nil {
		err = ddlOp.Wait(ctx)
	}
	if err != nil {
		if cleanupErr := owned.Close(); cleanupErr != nil {
			r.CleanupErrors = append(r.CleanupErrors, cleanupErr.Error())
		}
		return nil, fmt.Errorf("fixture schema: %s", status.Code(err))
	}
	owned.Client, err = spanner.NewClient(ctx, db, option.WithGRPCDialOption(grpc.WithChainUnaryInterceptor(interceptor)))
	if err != nil {
		if cleanupErr := owned.Close(); cleanupErr != nil {
			r.CleanupErrors = append(r.CleanupErrors, cleanupErr.Error())
		}
		return nil, err
	}
	return owned, nil
}

var resourceID = regexp.MustCompile(`projects/[^/\s"']+/instances/[^/\s"']+(?:/databases/[^/\s"']+)?`)
var requestID = regexp.MustCompile(`requestID = "[^"]*"`)
var ownedDatabaseID = regexp.MustCompile(`mutation1110-[a-f0-9]+`)

func redact(s string) string {
	s = resourceID.ReplaceAllString(s, "<managed-resource>")
	for _, k := range []string{"SPANNER_PROJECT_ID", "SPANNER_INSTANCE_ID", "SPANNER_DATABASE_ID"} {
		if v := os.Getenv(k); v != "" {
			s = strings.ReplaceAll(s, v, "<redacted>")
		}
	}
	s = requestID.ReplaceAllString(s, `requestID = "<redacted>"`)
	s = ownedDatabaseID.ReplaceAllString(s, "<owned-database>")
	return s
}
