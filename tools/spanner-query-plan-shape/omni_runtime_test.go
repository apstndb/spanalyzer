//go:build integration && omni

package main

import (
	"os"
	"strings"
	"testing"

	"github.com/apstndb/spanemuboost"
)

var planProbeOmniRuntime = spanemuboost.NewLazyRuntime(
	spanemuboost.BackendOmni,
	spanemuboost.WithContainerImage(strings.TrimSpace(os.Getenv("SPANALYZER_OMNI_IMAGE"))),
)

func TestMain(m *testing.M) {
	planProbeOmniRuntime.TestMain(m)
}

func openOmniClients(t *testing.T, ddls []string, options ...spanemuboost.Option) *spanemuboost.Clients {
	t.Helper()
	if os.Getenv("SPANEMUBOOST_ENABLE_OMNI_TESTS") == "" {
		t.Skip("set SPANEMUBOOST_ENABLE_OMNI_TESTS=1 to run Spanner Omni tests")
	}
	if strings.TrimSpace(os.Getenv("SPANALYZER_OMNI_IMAGE")) == "" {
		t.Fatal("set SPANALYZER_OMNI_IMAGE to the pinned Spanner Omni image under test")
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
