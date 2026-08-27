package captureenv

import (
	"strings"
	"testing"
)

func TestSplitTaggedImage(t *testing.T) {
	family, tag, digest, err := SplitTaggedImage("gcr.io/cloud-spanner-emulator/emulator:1.5.56")
	if err != nil {
		t.Fatal(err)
	}
	if family != "gcr.io/cloud-spanner-emulator/emulator" || tag != "1.5.56" || digest != "" {
		t.Fatalf("SplitTaggedImage() = %q, %q, %q", family, tag, digest)
	}
	pinned := "sha256:" + strings.Repeat("a", 64)
	_, _, digest, err = SplitTaggedImage("gcr.io/cloud-spanner-emulator/emulator:1.5.56@" + pinned)
	if err != nil || digest != pinned {
		t.Fatalf("SplitTaggedImage(tag@digest) = %q, %v", digest, err)
	}
	for _, input := range []string{
		"gcr.io/cloud-spanner-emulator/emulator",
		"gcr.io/cloud-spanner-emulator/emulator@" + pinned,
	} {
		if _, _, _, err := SplitTaggedImage(input); err == nil {
			t.Fatalf("SplitTaggedImage(%q) error = nil", input)
		}
	}
}

func TestValidateExecutionEnvironment(t *testing.T) {
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "-mod=readonly")
	if err := ValidateExecutionEnvironment(); err != nil {
		t.Fatal(err)
	}
}
