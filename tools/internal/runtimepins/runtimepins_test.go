package runtimepins

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImageForPlatform(t *testing.T) {
	root, err := FindRepositoryRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	image, err := ImageForPlatform(root, "omni", "linux/arm64")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(image, "spanner-omni:2026.r2.1-beta@sha256:ed31d9") {
		t.Fatalf("ImageForPlatform() = %q", image)
	}
}

func TestModuleLocalLoadersStayIdentical(t *testing.T) {
	root, err := FindRepositoryRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{
		filepath.Join(root, "tools", "internal", "runtimepins", "runtimepins.go"),
		filepath.Join(root, "survey", "internal", "runtimepins", "runtimepins.go"),
		filepath.Join(root, "cmd", "spanner-query-gen", "internal", "runtimepins", "runtimepins.go"),
	}
	var baseline []byte
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if baseline == nil {
			baseline = data
			continue
		}
		if !bytes.Equal(data, baseline) {
			t.Fatalf("module-local runtime pin loader %q diverged from %q", path, paths[0])
		}
	}
}
