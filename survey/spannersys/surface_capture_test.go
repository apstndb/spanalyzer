package spannersys

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/apstndb/spanalyzer/survey/internal/capturemeta"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

func validSurfaceCapture(t *testing.T, target capturemeta.Target) *SurfaceCapture {
	t.Helper()
	document, err := BuildSurfaceCapture([]ColumnObservation{
		{TableName: "EXAMPLE", ColumnName: "ID", SpannerType: "INT64", OrdinalPosition: 1},
		{TableName: "EXAMPLE", ColumnName: "NAME", SpannerType: "STRING(MAX)", OrdinalPosition: 2},
	}, time.Date(2026, 8, 27, 1, 2, 3, 123456789, time.UTC), target, SurfaceCaptureProducerIdentity{
		SourceSHA256: strings.Repeat("a", 64), InvocationSHA256: strings.Repeat("b", 64),
	})
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func TestSurfaceCaptureRoundTripAndPath(t *testing.T) {
	document := validSurfaceCapture(t, capturemeta.Target{Kind: "managed", ObservationScope: "single_database"})
	data, err := EncodeSurfaceCapture(document)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeSurfaceCapture(data)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.SurfaceSHA256 != document.SurfaceSHA256 {
		t.Fatalf("surface hash = %q, want %q", decoded.SurfaceSHA256, document.SurfaceSHA256)
	}
	path, err := ExpectedSurfaceCapturePath(document)
	if err != nil {
		t.Fatal(err)
	}
	want := "evidence/managed/20260827T010203Z-" + document.SurfaceSHA256[:12] + ".json"
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestDecodeSurfaceCaptureRejectsDestinationIdentifier(t *testing.T) {
	document := validSurfaceCapture(t, capturemeta.Target{Kind: "managed", ObservationScope: "single_database"})
	document.Columns[0].ColumnName = "projects/p/instances/i/databases/d"
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	for name, input := range map[string][]byte{
		"literal": data,
		"escaped": bytes.ReplaceAll(data, []byte("/"), []byte(`\u002f`)),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := DecodeSurfaceCapture(input)
			if err == nil || !strings.Contains(err.Error(), "forbidden destination identifier") {
				t.Fatalf("DecodeSurfaceCapture() error = %v, want redaction failure", err)
			}
		})
	}
}

func TestComputeSurfaceCaptureProducerIdentity(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	first, err := ComputeSurfaceCaptureProducerIdentity(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ComputeSurfaceCaptureProducerIdentity(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || !surfaceSHA256Pattern.MatchString(first.SourceSHA256) || !surfaceSHA256Pattern.MatchString(first.InvocationSHA256) {
		t.Fatalf("producer identity = %#v then %#v", first, second)
	}
}

func TestSurfaceCaptureSatisfiesJSONSchema(t *testing.T) {
	schemaData, err := os.ReadFile("../schemas/spanner-sys-capture.v0alpha2.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat = true
	const schemaURL = "file:///spanner-sys-capture.v0alpha2.schema.json"
	if err := compiler.AddResource(schemaURL, bytes.NewReader(schemaData)); err != nil {
		t.Fatal(err)
	}
	compiled, err := compiler.Compile(schemaURL)
	if err != nil {
		t.Fatal(err)
	}
	document := validSurfaceCapture(t, capturemeta.Target{Kind: "omni", Image: &capturemeta.ImageIdentity{
		Family: "example.invalid/omni", Tag: "v1", Digest: "sha256:" + strings.Repeat("c", 64), Platform: "linux/amd64",
	}})
	data, err := EncodeSurfaceCapture(document)
	if err != nil {
		t.Fatal(err)
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	if err := compiled.Validate(value); err != nil {
		t.Fatal(err)
	}
}

func TestLoadLegacySurfaceBaseline(t *testing.T) {
	managed, err := LoadLegacySurfaceBaseline("managed")
	if err != nil {
		t.Fatal(err)
	}
	if managed.SchemaVersion != "v0alpha1" || managed.Target != "managed" || managed.ColumnCount != 539 {
		t.Fatalf("legacy managed baseline = %#v", managed)
	}
}
