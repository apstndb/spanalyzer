//go:build integration && omni

package main

import (
	"os"
	"strings"
	"testing"

	"github.com/apstndb/spanemuboost"
)

func TestIntegrationHintPositionAuditOnOmni(t *testing.T) {
	if os.Getenv("SPANEMUBOOST_ENABLE_OMNI_TESTS") == "" {
		t.Skip("set SPANEMUBOOST_ENABLE_OMNI_TESTS=1 to run Spanner Omni tests")
	}

	image := strings.TrimSpace(os.Getenv("SPANALYZER_OMNI_IMAGE"))
	if image == "" {
		t.Fatal("set SPANALYZER_OMNI_IMAGE to the pinned Spanner Omni image under test")
	}
	runtime := spanemuboost.NewLazyRuntime(
		spanemuboost.BackendOmni,
		spanemuboost.WithContainerImage(image),
	)
	runHintPositionAudit(t, runtime)
}
