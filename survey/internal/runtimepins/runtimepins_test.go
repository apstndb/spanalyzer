package runtimepins

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

func TestImageForPlatform(t *testing.T) {
	root, err := FindRepositoryRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		kind     string
		platform string
		contains string
	}{
		{name: "emulator amd64", kind: "emulator", platform: "linux/amd64", contains: "emulator:1.5.56@sha256:24c921"},
		{name: "emulator arm64", kind: "emulator", platform: "linux/arm64", contains: "emulator:1.5.56@sha256:5b1e36"},
		{name: "omni amd64", kind: "omni", platform: "linux/amd64", contains: "spanner-omni:2026.r2.1-beta@sha256:48631b"},
		{name: "omni arm64", kind: "omni", platform: "linux/arm64", contains: "spanner-omni:2026.r2.1-beta@sha256:ed31d9"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ImageForPlatform(root, test.kind, test.platform)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got, test.contains) {
				t.Fatalf("ImageForPlatform() = %q, want substring %q", got, test.contains)
			}
		})
	}
}

func TestRuntimeTargetsSatisfyJSONSchema(t *testing.T) {
	root, err := FindRepositoryRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	schemaData, err := os.ReadFile(filepath.Join(root, "schemas", "runtime-targets.v0alpha1.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	documentData, err := os.ReadFile(filepath.Join(root, "runtime_targets.json"))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	const schemaURL = "file:///runtime-targets.v0alpha1.schema.json"
	if err := compiler.AddResource(schemaURL, bytes.NewReader(schemaData)); err != nil {
		t.Fatal(err)
	}
	compiled, err := compiler.Compile(schemaURL)
	if err != nil {
		t.Fatal(err)
	}
	var document any
	if err := json.Unmarshal(documentData, &document); err != nil {
		t.Fatal(err)
	}
	if err := compiled.Validate(document); err != nil {
		t.Fatal(err)
	}
}

func TestImageForPlatformRejectsUnknownTarget(t *testing.T) {
	root, err := FindRepositoryRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ImageForPlatform(root, "unknown", "linux/arm64"); err == nil {
		t.Fatal("ImageForPlatform() error = nil")
	}
}
