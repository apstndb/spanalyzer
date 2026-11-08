//go:build integration && omni

package main

import (
	"os"
	"testing"

	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"github.com/apstndb/spanemuboost"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIntegrationPlanReportRetentionOnOmni(t *testing.T) {
	if os.Getenv("SPANEMUBOOST_ENABLE_OMNI_TESTS") == "" {
		t.Skip("set SPANEMUBOOST_ENABLE_OMNI_TESTS=1")
	}
	querygenOmniIntegrationRequireRuntime(t)
	runtime := querygenOmniRuntime(t)
	seed := spanemuboost.SetupClients(t, runtime, spanemuboost.EnableDatabaseAutoConfigOnly(), spanemuboost.WithRandomDatabaseID(), spanemuboost.ForceSchemaTeardown())
	attached, err := spanemuboost.NewAttachedRuntime(spanemuboost.Endpoint{Backend: spanemuboost.BackendOmni, URI: seed.URI(), ProjectID: seed.ProjectID, InstanceID: seed.InstanceID})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := attached.Close(); err != nil {
			t.Error(err)
		}
	}()
	config, dir, opts := feedbackReportFixture(t)
	for _, drop := range []bool{false, true} {
		clients, err := openPlanReportClients(t.Context(), attached, config.Schemas[0], dir, nil, planReportResolvedOptimizer(opts.Optimizer), drop)
		if err != nil {
			t.Fatal(err)
		}
		path := clients.DatabasePath()
		if path == seed.DatabasePath() {
			t.Fatal("random database reused the endpoint seed")
		}
		if err := clients.Close(); err != nil {
			t.Fatal(err)
		}
		_, err = seed.DatabaseClient.GetDatabase(t.Context(), &databasepb.GetDatabaseRequest{Name: path})
		if drop {
			if status.Code(err) != codes.NotFound {
				t.Fatalf("opt-in cleanup GetDatabase = %v", err)
			}
		} else {
			if err != nil {
				t.Fatalf("default retention GetDatabase = %v", err)
			}
			// This test owns this exact retained database. Production never
			// enumerates or evicts databases left by previous invocations.
			if err := seed.DatabaseClient.DropDatabase(t.Context(), &databasepb.DropDatabaseRequest{Database: path}); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := seed.DatabaseClient.GetDatabase(t.Context(), &databasepb.GetDatabaseRequest{Name: seed.DatabasePath()}); err != nil {
			t.Fatalf("endpoint seed was affected: %v", err)
		}
	}
}
