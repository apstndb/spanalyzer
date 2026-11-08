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
	if !strings.Contains(image, "spanner-omni:2026.r4-lts.2@sha256:b607b1") {
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

func TestBundledCLIPinsMatchCanonicalRegistry(t *testing.T) {
	root, err := FindRepositoryRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := os.ReadFile(filepath.Join(root, "runtime_targets.json"))
	if err != nil {
		t.Fatal(err)
	}
	bundled, err := os.ReadFile(filepath.Join(root, "cmd/spanner-query-gen/internal/runtimepins/pins.generated.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(canonical, bundled) {
		t.Fatal("bundled CLI runtime pins are stale; run go generate ./internal/runtimepins from cmd/spanner-query-gen")
	}
}

func TestImageForVersionKeepsDefaultAndAlternativeSeparate(t *testing.T) {
	root, err := FindRepositoryRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, platform := range []string{"linux/amd64", "linux/arm64"} {
		alternative, err := ImageForVersion(root, "omni", "2026.r5.1", platform)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(alternative, ":2026.r5.1@sha256:") {
			t.Fatalf("alternative = %q", alternative)
		}
		defaultImage, err := ImageForPlatform(root, "omni", platform)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(defaultImage, ":2026.r4-lts.2@sha256:") {
			t.Fatalf("default after alternative lookup = %q", defaultImage)
		}
	}
	if _, err := ImageForVersion(root, "omni", "unregistered", "linux/arm64"); err == nil {
		t.Fatal("unregistered version must not fall back to the default")
	}
}

func TestInvalidAlternativeCannotBeIgnoredByDefaultLookup(t *testing.T) {
	root, err := FindRepositoryRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, registryFile))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ name, old, replacement string }{
		{"duplicate version", `"2026.r5.1"`, `"2026.r4-lts.2"`},
		{"invalid alternative digest", "sha256:b918acd3c785bd13733d7dca6e6c029ef4ab8d7b63a0cd28800210a9bf976516", "sha256:invalid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if strings.Count(string(data), test.old) != 1 {
				t.Fatal("test replacement is not unique")
			}
			dir := t.TempDir()
			invalid := strings.Replace(string(data), test.old, test.replacement, 1)
			if err := os.WriteFile(filepath.Join(dir, registryFile), []byte(invalid), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := ImageForPlatform(dir, "omni", "linux/arm64"); err == nil {
				t.Fatal("invalid alternative must fail the registry validation")
			}
		})
	}
}
