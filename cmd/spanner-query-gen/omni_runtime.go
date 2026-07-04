//go:build integration && omni

package main

import (
	"testing"

	"github.com/apstndb/spanemuboost"
)

// querygenOmniIntegrationRequireRuntime skips the Docker health check when
// Omni integration tests attach to a long-lived spanemuboost serve endpoint.
func querygenOmniIntegrationRequireRuntime(tb testing.TB) {
	tb.Helper()
	if spanemuboost.EndpointConfiguredForBackend(spanemuboost.BackendOmni) {
		return
	}
	querygenIntegrationRequireContainerRuntime(tb)
}

// querygenOmniRuntime returns a Spanner Omni runtime handle. When
// SPANEMUBOOST_ENDPOINT_FILE or SPANEMUBOOST_OMNI_URI is set, it attaches to the
// long-lived runtime started by `spanemuboost serve` instead of booting a new
// testcontainers instance.
func querygenOmniRuntime(tb testing.TB) spanemuboost.RuntimeHandle {
	tb.Helper()
	runtime, err := spanemuboost.NewLazyRuntimeFromEnvOrStart(spanemuboost.BackendOmni)
	if err != nil {
		tb.Fatalf("NewLazyRuntimeFromEnvOrStart: %v", err)
	}
	tb.Cleanup(func() {
		if err := runtime.Close(); err != nil {
			tb.Errorf("failed to close Spanner Omni runtime: %v", err)
		}
	})
	return runtime
}
