//go:build integration && omni

package main

import (
	"os"
	"testing"
)

func TestIntegrationSQLChoicesOnOmni(t *testing.T) {
	if os.Getenv("SPANEMUBOOST_ENABLE_OMNI_TESTS") == "" {
		t.Skip("set SPANEMUBOOST_ENABLE_OMNI_TESTS=1 to run Spanner Omni tests")
	}
	querygenOmniIntegrationRequireRuntime(t)
	querygenIntegrationSQLChoices(t, querygenOmniRuntime(t), true)
}
