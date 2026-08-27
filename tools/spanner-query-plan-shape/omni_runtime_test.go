//go:build integration && omni

package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/apstndb/spanemuboost"
)

var planProbeOmniRuntime spanemuboost.RuntimeHandle

func TestMain(m *testing.M) {
	if os.Getenv("SPANEMUBOOST_ENABLE_OMNI_TESTS") == "" {
		os.Exit(m.Run())
	}
	image := strings.TrimSpace(os.Getenv("SPANALYZER_OMNI_IMAGE"))
	if image == "" {
		var err error
		image, err = repositoryPinnedImage("omni")
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "resolve pinned Spanner Omni image: %v\n", err)
			os.Exit(1)
		}
	}
	runtime, err := spanemuboost.NewLazyRuntimeFromEnvOrStart(
		spanemuboost.BackendOmni,
		spanemuboost.WithContainerImage(image),
	)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "open Spanner Omni runtime: %v\n", err)
		os.Exit(1)
	}
	planProbeOmniRuntime = runtime
	code := m.Run()
	if err := runtime.Close(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "close Spanner Omni runtime: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

func openOmniClients(t *testing.T, ddls []string, options ...spanemuboost.Option) *spanemuboost.Clients {
	t.Helper()
	if os.Getenv("SPANEMUBOOST_ENABLE_OMNI_TESTS") == "" {
		t.Skip("set SPANEMUBOOST_ENABLE_OMNI_TESTS=1 to run Spanner Omni tests")
	}
	// Reopening one LazyRuntime disables database auto-configuration by
	// default, while random databases are not normally dropped on Close.
	// Re-enable creation and force per-test teardown so suites share only the
	// container, never database state.
	setupOptions := []spanemuboost.Option{
		spanemuboost.EnableDatabaseAutoConfigOnly(),
		spanemuboost.WithRandomDatabaseID(),
		spanemuboost.ForceSchemaTeardown(),
		spanemuboost.WithSetupDDLs(ddls),
	}
	setupOptions = append(setupOptions, options...)
	return spanemuboost.SetupClients(t, planProbeOmniRuntime, setupOptions...)
}
