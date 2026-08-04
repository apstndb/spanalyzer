//go:build integration

package main

import (
	"context"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/apstndb/spanemuboost"
)

const hintPositionEmulatorImage = "gcr.io/cloud-spanner-emulator/emulator:1.5.55"

func TestIntegrationHintPositionAuditOnEmulator(t *testing.T) {
	runtime := spanemuboost.NewLazyRuntime(
		spanemuboost.BackendEmulator,
		spanemuboost.EnableInstanceAutoConfigOnly(),
		spanemuboost.WithContainerImage(hintPositionEmulatorImage),
	)
	runHintPositionAudit(t, runtime)
}

func runHintPositionAudit(t *testing.T, runtime *spanemuboost.LazyRuntime) {
	t.Helper()
	ddls, err := parseBuiltInDDLs("hint-position-schema.sql", docsDDL)
	if err != nil {
		t.Fatalf("parseBuiltInDDLs() error = %v", err)
	}
	t.Cleanup(func() { _ = runtime.Close() })
	clients, err := spanemuboost.OpenClients(t.Context(), runtime,
		spanemuboost.WithRandomDatabaseID(),
		spanemuboost.WithSetupDDLs(ddls),
	)
	if err != nil {
		t.Fatalf("OpenClients() error = %v", err)
	}
	t.Cleanup(func() { _ = clients.Close() })

	for _, auditCase := range hintPositionAuditCases {
		t.Run(auditCase.Query.Label, func(t *testing.T) {
			err := executeHintPositionProbe(t, clients.Client, auditCase.Query)
			got, detail := classifyHintPositionResult(err)
			t.Logf("classification=%s detail=%s", got, detail)
			if got != auditCase.Expectation {
				t.Fatalf("syntax classification = %q, want %q: %s\nSQL: %s", got, auditCase.Expectation, detail, auditCase.Query.SQL)
			}
		})
	}
}

func executeHintPositionProbe(t *testing.T, client *spanner.Client, query queryCase) error {
	t.Helper()
	statement := spanner.NewStatement(query.SQL)
	if query.effectivePlanMode() == planModeReadWrite {
		_, err := client.ReadWriteTransaction(t.Context(), func(ctx context.Context, transaction *spanner.ReadWriteTransaction) error {
			_, err := transaction.Update(ctx, statement)
			return err
		})
		return err
	}
	return client.Single().Query(t.Context(), statement).Do(func(*spanner.Row) error { return nil })
}
