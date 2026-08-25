package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/apstndb/spanalyzer/survey/infoschem"
)

func TestRunRejectsMissingTarget(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if got := run(nil, &stdout, &stderr); got != 2 {
		t.Fatalf("run() = %d, want 2; stderr=%s", got, stderr.String())
	}
}

func TestRunRejectsWriteAndOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	got := run([]string{"--target", "managed", "--write", "--output", "capture.json"}, &stdout, &stderr)
	if got != 2 || !strings.Contains(stderr.String(), "mutually exclusive") {
		t.Fatalf("run() = %d, stderr = %q", got, stderr.String())
	}
}

func TestSplitTaggedImage(t *testing.T) {
	family, tag, digest, err := splitTaggedImage("gcr.io/cloud-spanner-emulator/emulator:1.5.56")
	if err != nil {
		t.Fatal(err)
	}
	if family != "gcr.io/cloud-spanner-emulator/emulator" || tag != "1.5.56" || digest != "" {
		t.Fatalf("splitTaggedImage() = %q, %q, %q", family, tag, digest)
	}
	pinned := "sha256:" + strings.Repeat("a", 64)
	_, _, digest, err = splitTaggedImage("gcr.io/cloud-spanner-emulator/emulator:1.5.56@" + pinned)
	if err != nil || digest != pinned {
		t.Fatalf("splitTaggedImage(tag@digest) = %q, %v", digest, err)
	}
	for _, input := range []string{
		"gcr.io/cloud-spanner-emulator/emulator",
		"gcr.io/cloud-spanner-emulator/emulator@" + pinned,
	} {
		if _, _, _, err := splitTaggedImage(input); err == nil {
			t.Fatalf("splitTaggedImage(%q) error = nil", input)
		}
	}
}

func TestCreateAtomicallyDoesNotReplace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "capture.json")
	wrote, err := createAtomically(path, []byte("first\n"))
	if err != nil || !wrote {
		t.Fatalf("first create = %t, %v", wrote, err)
	}
	wrote, err = createAtomically(path, []byte("second\n"))
	if err != nil || wrote {
		t.Fatalf("second create = %t, %v", wrote, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "first\n" {
		t.Fatalf("written bytes = %q", data)
	}
}

func TestCreateAtomicallyConcurrentWriters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capture.json")
	start := make(chan struct{})
	type result struct {
		wrote bool
		err   error
	}
	results := make(chan result, 2)
	var wait sync.WaitGroup
	for _, data := range [][]byte{[]byte("first\n"), []byte("second\n")} {
		wait.Add(1)
		go func(data []byte) {
			defer wait.Done()
			<-start
			wrote, err := createAtomically(path, data)
			results <- result{wrote: wrote, err: err}
		}(data)
	}
	close(start)
	wait.Wait()
	close(results)
	writes := 0
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		if result.wrote {
			writes++
		}
	}
	if writes != 1 {
		t.Fatalf("successful creates = %d, want 1", writes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "first\n" && string(data) != "second\n" {
		t.Fatalf("retained partial or unexpected data = %q", data)
	}
}

func TestWriteCaptureContainerKeepsFirstEquivalentObservation(t *testing.T) {
	target := func(tag string) infoschem.CaptureTarget {
		return infoschem.CaptureTarget{Kind: "emulator", Image: &infoschem.ImageIdentity{
			Family:   "gcr.io/cloud-spanner-emulator/emulator",
			Tag:      tag,
			Digest:   "sha256:" + strings.Repeat("d", 64),
			Platform: "linux/arm64",
		}}
	}
	metadata := infoschem.DiscoveredColumnMetadata{
		"EXAMPLE": {
			"NAME": {Name: "NAME", SpannerType: "STRING(MAX)", OrdinalPosition: 1},
		},
	}
	queryability := []infoschem.RollingQueryability{
		{TableName: "INDEXES", ColumnName: "SEARCH_UNNEST", Status: "not_advertised"},
		{TableName: "INDEX_COLUMNS", ColumnName: "EXPRESSION", Status: "not_advertised"},
	}
	producer := infoschem.ProducerIdentity{
		SourceSHA256:     strings.Repeat("a", 64),
		InvocationSHA256: strings.Repeat("b", 64),
	}
	first, err := infoschem.BuildCapture(metadata, queryability, time.Date(2026, 8, 25, 1, 0, 0, 0, time.UTC), target("1.5.56"), producer)
	if err != nil {
		t.Fatal(err)
	}
	second, err := infoschem.BuildCapture(metadata, queryability, time.Date(2026, 8, 26, 1, 0, 0, 0, time.UTC), target("stable-alias"), producer)
	if err != nil {
		t.Fatal(err)
	}
	firstData, err := infoschem.EncodeCapture(first)
	if err != nil {
		t.Fatal(err)
	}
	secondData, err := infoschem.EncodeCapture(second)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "capture.json")
	wrote, err := writeCapture(path, firstData, first)
	if err != nil || !wrote {
		t.Fatalf("first write = %t, %v", wrote, err)
	}
	wrote, err = writeCapture(path, secondData, second)
	if err != nil || wrote {
		t.Fatalf("equivalent rerun write = %t, %v", wrote, err)
	}
	retained, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(retained, firstData) {
		t.Fatal("equivalent rerun replaced the first retained observation")
	}
}

func TestWriteCaptureManagedKeepsFirstEquivalentObservationInSameSecond(t *testing.T) {
	metadata := infoschem.DiscoveredColumnMetadata{
		"EXAMPLE": {
			"NAME": {Name: "NAME", SpannerType: "STRING(MAX)", OrdinalPosition: 1},
		},
	}
	queryability := []infoschem.RollingQueryability{
		{TableName: "INDEXES", ColumnName: "SEARCH_UNNEST", Status: "not_advertised"},
		{TableName: "INDEX_COLUMNS", ColumnName: "EXPRESSION", Status: "not_advertised"},
	}
	target := infoschem.CaptureTarget{Kind: "managed", ObservationScope: "single_database"}
	producer := infoschem.ProducerIdentity{
		SourceSHA256:     strings.Repeat("a", 64),
		InvocationSHA256: strings.Repeat("b", 64),
	}
	first, err := infoschem.BuildCapture(metadata, queryability, time.Date(2026, 8, 25, 1, 0, 0, 100, time.UTC), target, producer)
	if err != nil {
		t.Fatal(err)
	}
	second, err := infoschem.BuildCapture(metadata, queryability, time.Date(2026, 8, 25, 1, 0, 0, 900, time.UTC), target, producer)
	if err != nil {
		t.Fatal(err)
	}
	firstData, err := infoschem.EncodeCapture(first)
	if err != nil {
		t.Fatal(err)
	}
	secondData, err := infoschem.EncodeCapture(second)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "capture.json")
	wrote, err := writeCapture(path, firstData, first)
	if err != nil || !wrote {
		t.Fatalf("first write = %t, %v", wrote, err)
	}
	wrote, err = writeCapture(path, secondData, second)
	if err != nil || wrote {
		t.Fatalf("equivalent rerun write = %t, %v", wrote, err)
	}
	retained, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(retained, firstData) {
		t.Fatal("same-second equivalent rerun replaced the first retained observation")
	}
}

func TestValidateExecutionEnvironment(t *testing.T) {
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "-mod=readonly")
	if err := validateExecutionEnvironment(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOWORK", "")
	if err := validateExecutionEnvironment(); err == nil {
		t.Fatal("validateExecutionEnvironment() accepted workspace mode")
	}
}

func TestResolveRepoRoot(t *testing.T) {
	root, err := resolveRepoRoot("")
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	if root != want {
		t.Fatalf("resolveRepoRoot() = %q, want %q", root, want)
	}
}
