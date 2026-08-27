package capturemeta

import (
	"strings"
	"testing"
)

func TestExpectedPath(t *testing.T) {
	surface := strings.Repeat("a", 64)
	managed, err := ExpectedPath(Target{Kind: "managed", ObservationScope: "single_database"}, "2026-08-27T01:02:03.123Z", surface)
	if err != nil {
		t.Fatal(err)
	}
	if managed != "evidence/managed/20260827T010203Z-aaaaaaaaaaaa.json" {
		t.Fatalf("managed path = %q", managed)
	}
	container, err := ExpectedPath(Target{Kind: "omni", Image: &ImageIdentity{
		Family: "example.invalid/omni", Tag: "v1", Digest: "sha256:" + strings.Repeat("b", 64), Platform: "linux/arm64/v8",
	}}, "2026-08-27T01:02:03Z", surface)
	if err != nil {
		t.Fatal(err)
	}
	want := "evidence/omni/" + strings.Repeat("b", 64) + "/linux-arm64-v8-aaaaaaaaaaaa.json"
	if container != want {
		t.Fatalf("container path = %q, want %q", container, want)
	}
}

func TestValidateTargetRejectsMutableContainerIdentity(t *testing.T) {
	err := ValidateTarget(Target{Kind: "emulator", Image: &ImageIdentity{
		Family: "example.invalid/emulator", Tag: "latest", Platform: "linux/amd64",
	}})
	if err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("ValidateTarget() error = %v, want digest failure", err)
	}
}
