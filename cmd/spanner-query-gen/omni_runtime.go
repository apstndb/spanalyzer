//go:build integration && omni

package main

import (
	"os"
	"strings"
	"testing"

	"github.com/apstndb/spanemuboost"
)

const querygenOmniImageEnv = "SPANALYZER_OMNI_IMAGE"

// querygenOmniIntegrationRequireRuntime skips the Docker health check when
// Omni integration tests attach to a long-lived spanemuboost serve endpoint.
func querygenOmniIntegrationRequireRuntime(tb testing.TB) {
	tb.Helper()
	if spanemuboost.EndpointConfiguredForBackend(spanemuboost.BackendOmni) {
		return
	}
	querygenIntegrationRequireContainerRuntime(tb)
}

// querygenOmniRuntime returns a Spanner Omni runtime handle. SPANALYZER_OMNI_IMAGE
// pins the image used for a cold start. When
// SPANEMUBOOST_ENDPOINT_FILE or SPANEMUBOOST_OMNI_URI is set, it attaches to the
// long-lived runtime started by `spanemuboost serve` instead of booting a new
// testcontainers instance.
func querygenOmniRuntime(tb testing.TB) spanemuboost.RuntimeHandle {
	tb.Helper()
	var options []spanemuboost.Option
	if image := strings.TrimSpace(os.Getenv(querygenOmniImageEnv)); image != "" {
		options = append(options, spanemuboost.WithContainerImage(image))
	}
	runtime, err := spanemuboost.NewLazyRuntimeFromEnvOrStart(spanemuboost.BackendOmni, options...)
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
