package main

import (
	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	dbpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
	"strings"
	"time"
)

// Resume never calls CreateDatabase. The trusted private journal identifies
// the single DB already owned by this authorized task.
func resumeManagedFixture(ctx context.Context, journal string, interceptor grpc.UnaryClientInterceptor, r *result) (*managedClients, error) {
	parent, err := managedTarget()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(journal)
	if err != nil {
		return nil, err
	}
	var j struct{ Database string }
	if err := json.Unmarshal(b, &j); err != nil {
		return nil, err
	}
	if !strings.HasPrefix(j.Database, parent+"/databases/mutation1110-") || strings.Contains(strings.TrimPrefix(j.Database, parent+"/databases/"), "/") {
		return nil, fmt.Errorf("ownership journal target mismatch")
	}
	admin, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := admin.GetDatabase(ctx, &dbpb.GetDatabaseRequest{Name: j.Database}); err != nil {
		_ = admin.Close()
		return nil, err
	}
	c := &managedClients{DatabaseClient: admin, name: j.Database}
	c.close = func() error {
		if c.Client != nil {
			c.Client.Close()
		}
		defer func() { _ = admin.Close() }()
		cx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		err := admin.DropDatabase(cx, &dbpb.DropDatabaseRequest{Database: c.name})
		r.Lifecycle["drop"] = status.Code(err).String()
		if err != nil {
			return err
		}
		_, err = admin.GetDatabase(cx, &dbpb.GetDatabaseRequest{Name: c.name})
		r.Lifecycle["absence"] = status.Code(err).String()
		if status.Code(err) != codes.NotFound {
			return fmt.Errorf("owned DB absence: %s", status.Code(err))
		}
		return nil
	}
	c.Client, err = spanner.NewClient(ctx, c.name, option.WithGRPCDialOption(grpc.WithChainUnaryInterceptor(interceptor)))
	if err != nil {
		_ = admin.Close()
		return nil, err
	}
	r.Lifecycle["resume_existing_owned_database"] = "OK"
	// Remove only this fixture's optional queues, then reset its known rows in
	// FK dependency order. A fresh DB/instance is never provisioned on resume.
	ddl, err := admin.GetDatabaseDdl(ctx, &dbpb.GetDatabaseDdlRequest{Database: c.name})
	if err != nil {
		return nil, err
	}
	for _, name := range []string{"SharedForeign", "AliasForeign", "InfoForeign"} {
		found := false
		for _, sql := range ddl.Statements {
			if strings.HasPrefix(strings.ToUpper(sql), "CREATE TABLE "+strings.ToUpper(name)+" ") {
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("owned resume schema is missing %s", name)
		}
	}
	for _, name := range []string{"ChildMessages", "ReceiveMessages", "JsonMessages", "TextMessages", "Messages"} {
		exists := false
		for _, sql := range ddl.Statements {
			if strings.HasPrefix(strings.ToUpper(sql), "CREATE QUEUE "+strings.ToUpper(name)+" ") || strings.HasPrefix(strings.ToUpper(sql), "CREATE QUEUE "+strings.ToUpper(name)+"(") {
				exists = true
			}
		}
		if exists {
			op, err := admin.UpdateDatabaseDdl(ctx, &dbpb.UpdateDatabaseDdlRequest{Database: c.name, Statements: []string{"DROP QUEUE " + name}})
			if err == nil {
				err = op.Wait(ctx)
			}
			if err != nil {
				return nil, err
			}
		}
	}
	for _, name := range []string{"SharedForeign", "AliasForeign", "InfoForeign", "ForeignEvents", "ChildRows", "ParentRows", "Owners", "Bare", "OneIndex", "TwoIndexes", "StoreIndex", "NullIndex", "CompositeIndex", "DefaultsOnly", "GeneratedOnly", "BothAuto"} {
		cx, cancel := context.WithTimeout(ctx, 45*time.Second)
		_, err := c.Client.Apply(cx, []*spanner.Mutation{spanner.Delete(name, spanner.AllKeys())})
		cancel()
		if err != nil {
			return nil, fmt.Errorf("reset owned fixture %s: %s", name, status.Code(err))
		}
	}
	return c, nil
}
