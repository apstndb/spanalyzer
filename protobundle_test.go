package spanalyzer

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestLoadProtoDescriptorSetFilesDeduplicatesIdenticalFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("example.proto"),
		Package: proto.String("example"),
		Syntax:  proto.String("proto3"),
	}
	first := writeDescriptorSet(t, dir, "first.pb", file)
	withSourceInfo := proto.Clone(file).(*descriptorpb.FileDescriptorProto)
	withSourceInfo.SourceCodeInfo = &descriptorpb.SourceCodeInfo{
		Location: []*descriptorpb.SourceCodeInfo_Location{{
			LeadingComments: proto.String("generated documentation"),
		}},
	}
	second := writeDescriptorSet(t, dir, "second.pb", withSourceInfo)

	loaded, err := LoadProtoDescriptorSetFiles([]string{first, second})
	if err != nil {
		t.Fatalf("LoadProtoDescriptorSetFiles() error = %v", err)
	}
	got := loaded.FileDescriptorSet()
	if len(got.GetFile()) != 1 || got.GetFile()[0].GetName() != "example.proto" {
		t.Fatalf("merged descriptor files = %v, want one example.proto", got.GetFile())
	}
	if got.GetFile()[0].GetSourceCodeInfo() != nil {
		t.Fatal("canonical merged descriptor retained SourceCodeInfo")
	}

	// The accessor must not expose mutable catalog state to callers.
	got.File[0].Name = proto.String("mutated.proto")
	if name := loaded.FileDescriptorSet().GetFile()[0].GetName(); name != "example.proto" {
		t.Fatalf("FileDescriptorSet() returned shared state; name = %q", name)
	}
}

func TestLoadProtoDescriptorSetFilesRejectsConflictingDuplicate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	first := writeDescriptorSet(t, dir, "first.pb", &descriptorpb.FileDescriptorProto{
		Name:    proto.String("example.proto"),
		Package: proto.String("example.v1"),
		Syntax:  proto.String("proto3"),
	})
	second := writeDescriptorSet(t, dir, "second.pb", &descriptorpb.FileDescriptorProto{
		Name:    proto.String("example.proto"),
		Package: proto.String("example.v2"),
		Syntax:  proto.String("proto3"),
	})

	_, err := LoadProtoDescriptorSetFiles([]string{first, second})
	if err == nil ||
		!strings.Contains(err.Error(), `conflicting proto descriptor file "example.proto"`) ||
		!strings.Contains(err.Error(), "first.pb") ||
		!strings.Contains(err.Error(), "second.pb") {
		t.Fatalf("LoadProtoDescriptorSetFiles() error = %v, want conflicting duplicate", err)
	}
}

func TestFileDescriptorSetCanonicalOrderIsInputOrderIndependent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	a := writeDescriptorSet(t, dir, "a.pb", &descriptorpb.FileDescriptorProto{
		Name:   proto.String("a.proto"),
		Syntax: proto.String("proto3"),
	})
	z := writeDescriptorSet(t, dir, "z.pb", &descriptorpb.FileDescriptorProto{
		Name:   proto.String("z.proto"),
		Syntax: proto.String("proto3"),
	})
	forward, err := LoadProtoDescriptorSetFiles([]string{a, z})
	if err != nil {
		t.Fatalf("LoadProtoDescriptorSetFiles(forward) error = %v", err)
	}
	reverse, err := LoadProtoDescriptorSetFiles([]string{z, a})
	if err != nil {
		t.Fatalf("LoadProtoDescriptorSetFiles(reverse) error = %v", err)
	}
	marshal := proto.MarshalOptions{Deterministic: true}
	forwardBytes, err := marshal.Marshal(forward.FileDescriptorSet())
	if err != nil {
		t.Fatalf("Marshal(forward) error = %v", err)
	}
	reverseBytes, err := marshal.Marshal(reverse.FileDescriptorSet())
	if err != nil {
		t.Fatalf("Marshal(reverse) error = %v", err)
	}
	if !slices.Equal(forwardBytes, reverseBytes) {
		t.Fatal("canonical descriptor encoding changed with input path order")
	}
	files := forward.FileDescriptorSet().GetFile()
	if len(files) != 2 {
		t.Fatalf("canonical descriptor file count = %d, want 2", len(files))
	}
	if got := []string{files[0].GetName(), files[1].GetName()}; !slices.Equal(got, []string{"a.proto", "z.proto"}) {
		t.Fatalf("canonical file order = %v, want [a.proto z.proto]", got)
	}
}

func writeDescriptorSet(t *testing.T, dir, name string, files ...*descriptorpb.FileDescriptorProto) string {
	t.Helper()
	data, err := proto.Marshal(&descriptorpb.FileDescriptorSet{File: files})
	if err != nil {
		t.Fatalf("proto.Marshal() error = %v", err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return path
}
