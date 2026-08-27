package runtimepins

import (
	"strings"
	"testing"
)

func TestImageForPlatform(t *testing.T) {
	root, err := FindRepositoryRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	image, err := ImageForPlatform(root, "emulator", "linux/amd64")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(image, "emulator:1.5.56@sha256:24c921") {
		t.Fatalf("ImageForPlatform() = %q", image)
	}
}
