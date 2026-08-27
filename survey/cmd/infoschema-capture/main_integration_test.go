package main

import (
	"context"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/apstndb/spanalyzer/survey/internal/runtimepins"
	"github.com/apstndb/spanemuboost"
)

func TestCaptureTransactionSupportsMultipleQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Emulator-backed transaction test in short mode")
	}
	ctx := context.Background()
	repoRoot, err := runtimepins.FindRepositoryRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	emulatorImage, err := runtimepins.ImageForHost(repoRoot, "emulator")
	if err != nil {
		t.Fatal(err)
	}
	env, err := spanemuboost.RunWithClients(
		ctx,
		spanemuboost.BackendEmulator,
		spanemuboost.WithContainerImage(emulatorImage),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = env.Close() }()

	txn := newCaptureTransaction(env.Client)
	defer txn.Close()
	for index := range 2 {
		iter := txn.Query(ctx, spanner.NewStatement("SELECT 1"))
		if _, err := iter.Next(); err != nil {
			iter.Stop()
			t.Fatalf("query %d: %v", index+1, err)
		}
		iter.Stop()
	}
	if _, err := txn.Timestamp(); err != nil {
		t.Fatalf("transaction timestamp: %v", err)
	}
}
