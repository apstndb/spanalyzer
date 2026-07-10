package main

import (
	"path/filepath"
	"testing"

	"github.com/apstndb/spanalyzer/internal/querygen"
)

func TestPlanReportProtoDescriptorSet(t *testing.T) {
	t.Parallel()

	if got, err := planReportProtoDescriptorSet(querygen.QueryCodegenSchema{}, t.TempDir()); err != nil || got != nil {
		t.Fatalf("planReportProtoDescriptorSet(empty) = (%v, %v), want (nil, nil)", got, err)
	}

	baseDir := filepath.Join("..", "..")
	loaded, err := planReportProtoDescriptorSet(querygen.QueryCodegenSchema{
		ProtoDescriptorFiles: []string{"testdata/protos/order_descriptors.pb"},
	}, baseDir)
	if err != nil {
		t.Fatalf("planReportProtoDescriptorSet() error = %v", err)
	}
	if loaded == nil || len(loaded.FileDescriptorSet().GetFile()) == 0 {
		t.Fatal("planReportProtoDescriptorSet() returned no descriptor files")
	}
	digest, err := planReportProtoDescriptorDigest(loaded)
	if err != nil {
		t.Fatalf("planReportProtoDescriptorDigest() error = %v", err)
	}
	if len(digest) != 64 {
		t.Fatalf("planReportProtoDescriptorDigest() = %q, want a SHA-256 digest", digest)
	}
}
