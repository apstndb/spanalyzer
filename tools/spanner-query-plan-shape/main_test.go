package main

import (
	"io"
	"strings"
	"testing"
)

func TestIsRawPlanOutput(t *testing.T) {
	for _, output := range []string{"json", "yaml"} {
		if !isRawPlanOutput(output) {
			t.Fatalf("isRawPlanOutput(%q) = false, want true", output)
		}
	}
	for _, output := range []string{"nodes", "reference", "textproto"} {
		if isRawPlanOutput(output) {
			t.Fatalf("isRawPlanOutput(%q) = true, want false", output)
		}
	}
}

func TestRunGoogleSQLProtoSurfaceRequiresDescriptors(t *testing.T) {
	err := run([]string{"--case", "google_sql_proto_surface"}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "requires --proto-descriptors-file") {
		t.Fatalf("run() error = %v, want missing proto descriptor error", err)
	}
}
