//go:build integration

package main

import (
	"context"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/apstndb/spanemuboost"
)

func TestIntegrationHintPositionAuditOnEmulator(t *testing.T) {
	emulatorImage, err := repositoryPinnedImage("emulator")
	if err != nil {
		t.Fatal(err)
	}
	runtime := spanemuboost.NewLazyRuntime(
		spanemuboost.BackendEmulator,
		spanemuboost.EnableInstanceAutoConfigOnly(),
		spanemuboost.WithContainerImage(emulatorImage),
	)
	runHintPositionAuditWithOverrides(t, runtime, map[string]hintPositionExpectation{
		"hint-position/accepted/pipe-log-unsupported": hintPositionUnavailable,
		"hint-position/versioned/pipe-finish":         hintPositionUnavailable,
	})
}

func runHintPositionAudit(t *testing.T, runtime *spanemuboost.LazyRuntime) {
	runHintPositionAuditWithOverrides(t, runtime, nil)
}

func runHintPositionAuditWithOverrides(t *testing.T, runtime *spanemuboost.LazyRuntime, overrides map[string]hintPositionExpectation) {
	t.Helper()
	t.Cleanup(func() { _ = runtime.Close() })
	clients, err := spanemuboost.OpenClients(t.Context(), runtime,
		spanemuboost.WithRandomDatabaseID(),
		spanemuboost.WithSetupDDLs(hintPositionDDLs(t)),
	)
	if err != nil {
		t.Fatalf("OpenClients() error = %v", err)
	}
	t.Cleanup(func() { _ = clients.Close() })
	runHintPositionAuditCasesWithOverrides(t, clients.Client, overrides)
}

func hintPositionDDLs(t *testing.T) []string {
	t.Helper()
	ddls, err := parseBuiltInDDLs("hint-position-schema.sql", docsDDL)
	if err != nil {
		t.Fatalf("parseBuiltInDDLs() error = %v", err)
	}
	return ddls
}

func runHintPositionAuditCases(t *testing.T, client *spanner.Client) {
	runHintPositionAuditCasesWithOverrides(t, client, nil)
}

func runHintPositionAuditCasesWithOverrides(t *testing.T, client *spanner.Client, overrides map[string]hintPositionExpectation) {
	t.Helper()
	for _, auditCase := range hintPositionAuditCases {
		t.Run(auditCase.Query.Label, func(t *testing.T) {
			err := executeHintPositionProbe(t, client, auditCase.Query)
			got, detail := classifyHintPositionResult(err)
			t.Logf("classification=%s detail=%s", got, detail)
			want := auditCase.Expectation
			if override, ok := overrides[auditCase.Query.Label]; ok {
				want = override
			}
			if got != want {
				t.Fatalf("syntax classification = %q, want %q: %s\nSQL: %s", got, want, detail, auditCase.Query.SQL)
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
