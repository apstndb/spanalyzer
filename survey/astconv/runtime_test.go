package astconv

import (
	"testing"

	"github.com/apstndb/spanalyzer/survey/internal/runtimepins"
)

func pinnedEmulatorImage(t *testing.T) string {
	t.Helper()
	repoRoot, err := runtimepins.FindRepositoryRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	image, err := runtimepins.ImageForHost(repoRoot, "emulator")
	if err != nil {
		t.Fatal(err)
	}
	return image
}
