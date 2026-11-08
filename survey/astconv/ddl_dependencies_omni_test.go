//go:build integration && omni

package astconv

import (
	"os"
	"testing"

	"github.com/apstndb/spanalyzer/survey/internal/runtimepins"
	"github.com/apstndb/spanemuboost"
)

func TestIntegrationDDLDependenciesOnOmni(t *testing.T) {
	if os.Getenv("SPANEMUBOOST_ENABLE_OMNI_TESTS") == "" {
		t.Skip("set SPANEMUBOOST_ENABLE_OMNI_TESTS=1")
	}
	root, err := runtimepins.FindRepositoryRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	image, err := runtimepins.ImageForHost(root, "omni")
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := spanemuboost.NewLazyRuntimeFromEnvOrStart(spanemuboost.BackendOmni, spanemuboost.WithContainerImage(image))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := runtime.Close(); err != nil {
			t.Error(err)
		}
	})
	assertDDLDependencyReapplication(t, runtime, true)
}
