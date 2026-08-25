package infoschem

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/apstndb/spanalyzer/survey/internal/strictjson"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsClientClosedTransactionError(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "closed transaction", err: status.Error(codes.InvalidArgument, closedTransactionErr), want: true},
		{name: "backend invalid argument", err: status.Error(codes.InvalidArgument, "Name not found inside table"), want: false},
		{name: "different code", err: status.Error(codes.FailedPrecondition, closedTransactionErr), want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := isClientClosedTransactionError(test.err); got != test.want {
				t.Fatalf("isClientClosedTransactionError() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestBuildCaptureManaged(t *testing.T) {
	document := validTestCapture(t, CaptureTarget{
		Kind:             "managed",
		ObservationScope: "single_database",
	})
	if document.Target.Image != nil {
		t.Fatal("managed capture unexpectedly has image identity")
	}
	if document.ObservedAt != "2026-08-25T08:30:12.123456789Z" {
		t.Fatalf("ObservedAt = %q", document.ObservedAt)
	}
	path, err := ExpectedCapturePath(document)
	if err != nil {
		t.Fatal(err)
	}
	want := "evidence/managed/20260825T083012Z-" + document.SurfaceSHA256[:12] + ".json"
	if path != want {
		t.Fatalf("ExpectedCapturePath() = %q, want %q", path, want)
	}

	data, err := EncodeCapture(document)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeCapture(data)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.SurfaceSHA256 != document.SurfaceSHA256 {
		t.Fatalf("decoded surface hash = %q, want %q", decoded.SurfaceSHA256, document.SurfaceSHA256)
	}
}

func TestBuildCaptureContainerPathUsesDigestAndPlatform(t *testing.T) {
	digest := "sha256:" + strings.Repeat("d", 64)
	document := validTestCapture(t, CaptureTarget{
		Kind: "omni",
		Image: &ImageIdentity{
			Family:   "us-docker.pkg.dev/spanner-omni/images/spanner-omni",
			Tag:      "2026.r2.1-beta",
			Digest:   digest,
			Platform: "linux/arm64/v8",
		},
	})
	path, err := ExpectedCapturePath(document)
	if err != nil {
		t.Fatal(err)
	}
	want := "evidence/omni/" + strings.Repeat("d", 64) + "/linux-arm64-v8-" + document.SurfaceSHA256[:12] + ".json"
	if path != want {
		t.Fatalf("ExpectedCapturePath() = %q, want %q", path, want)
	}
}

func TestDecodeCaptureRejectsDestinationIdentifier(t *testing.T) {
	document := validTestCapture(t, CaptureTarget{
		Kind: "emulator",
		Image: &ImageIdentity{
			Family:   "gcr.io/cloud-spanner-emulator/emulator",
			Tag:      "1.5.56",
			Digest:   "sha256:" + strings.Repeat("e", 64),
			Platform: "linux/amd64",
		},
	})
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
			_, err := DecodeCapture(input)
			if err == nil || !strings.Contains(err.Error(), "forbidden destination identifier") {
				t.Fatalf("DecodeCapture() error = %v, want redaction failure", err)
			}
		})
	}
}

func TestValidateCaptureRejectsStaleSurfaceHash(t *testing.T) {
	document := validTestCapture(t, CaptureTarget{
		Kind:             "managed",
		ObservationScope: "single_database",
	})
	document.Columns[0].ColumnName += "_changed"
	err := ValidateCapture(document)
	if err == nil || !strings.Contains(err.Error(), "surface_sha256") {
		t.Fatalf("ValidateCapture() error = %v, want surface hash failure", err)
	}
}

func TestValidateCaptureRejectsRawRollingError(t *testing.T) {
	document := validTestCapture(t, CaptureTarget{
		Kind:             "managed",
		ObservationScope: "single_database",
	})
	document.RollingQueryability[0].Status = "advertised_not_queryable"
	document.RollingQueryability[0].StatusCode = "backend said database projects/p failed"
	refreshTestSurfaceHash(t, document)
	err := ValidateCapture(document)
	if err == nil || !strings.Contains(err.Error(), "unsupported status_code") {
		t.Fatalf("ValidateCapture() error = %v, want bounded status-code failure", err)
	}
}

func TestValidateCaptureRejectsQueryabilityAdvertisementMismatch(t *testing.T) {
	document := validTestCapture(t, CaptureTarget{
		Kind:             "managed",
		ObservationScope: "single_database",
	})
	document.RollingQueryability[0].Status = "queryable"
	refreshTestSurfaceHash(t, document)
	err := ValidateCapture(document)
	if err == nil || !strings.Contains(err.Error(), "disagrees with column advertisement") {
		t.Fatalf("ValidateCapture() error = %v, want advertisement mismatch", err)
	}
}

func TestComputeProducerIdentity(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	first, err := ComputeProducerIdentity(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ComputeProducerIdentity(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("ComputeProducerIdentity() is nondeterministic: %#v != %#v", first, second)
	}
	if !sha256Pattern.MatchString(first.SourceSHA256) || !sha256Pattern.MatchString(first.InvocationSHA256) {
		t.Fatalf("ComputeProducerIdentity() = %#v, want SHA-256 values", first)
	}
}

func TestCommittedCaptures(t *testing.T) {
	var paths []string
	for _, pattern := range []string{
		"evidence/managed/*.json",
		"evidence/omni/*/*.json",
		"evidence/emulator/*/*.json",
	} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, matches...)
	}
	if len(paths) < 3 {
		t.Fatalf("committed capture count = %d, want at least one managed, Omni, and Emulator capture", len(paths))
	}
	targets := make(map[string]bool)
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		document, err := DecodeCapture(data)
		if err != nil {
			t.Fatalf("DecodeCapture(%q): %v", path, err)
		}
		expected, err := ExpectedCapturePath(document)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.ToSlash(path) != expected {
			t.Fatalf("capture path = %q, identity requires %q", path, expected)
		}
		targets[document.Target.Kind] = true
	}
	for _, target := range []string{"managed", "omni", "emulator"} {
		if !targets[target] {
			t.Errorf("committed captures have no %s target", target)
		}
	}
}

func TestCommittedCapturesSatisfyJSONSchema(t *testing.T) {
	schemaData, err := os.ReadFile("../schemas/information-schema-capture.v0alpha1.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat = true
	const schemaURL = "file:///information-schema-capture.v0alpha1.schema.json"
	if err := compiler.AddResource(schemaURL, bytes.NewReader(schemaData)); err != nil {
		t.Fatal(err)
	}
	compiled, err := compiler.Compile(schemaURL)
	if err != nil {
		t.Fatal(err)
	}
	for _, pattern := range []string{
		"evidence/managed/*.json",
		"evidence/omni/*/*.json",
		"evidence/emulator/*/*.json",
	} {
		paths, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var instance any
			if err := strictjson.Decode(data, &instance); err != nil {
				t.Fatal(err)
			}
			if err := compiled.Validate(instance); err != nil {
				t.Errorf("capture %q does not satisfy JSON Schema: %v", path, err)
			}
		}
	}
}

func TestInformationSchemaProjectionContracts(t *testing.T) {
	tests := []struct {
		name       string
		schemaPath string
		dataPath   string
	}{
		{
			name:       "projection source",
			schemaPath: "../../schemas/spanalyzer.information-schema-projection-source.v0alpha1.schema.json",
			dataPath:   "../../information_schema_projection_source.json",
		},
		{
			name:       "generated manifest",
			schemaPath: "../../schemas/spanalyzer.information-schema-manifest.v0alpha2.schema.json",
			dataPath:   "../../information_schema_manifest.json",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			schemaData, err := os.ReadFile(test.schemaPath)
			if err != nil {
				t.Fatal(err)
			}
			compiler := jsonschema.NewCompiler()
			compiler.AssertFormat = true
			const schemaURL = "file:///contract.schema.json"
			if err := compiler.AddResource(schemaURL, bytes.NewReader(schemaData)); err != nil {
				t.Fatal(err)
			}
			compiled, err := compiler.Compile(schemaURL)
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(test.dataPath)
			if err != nil {
				t.Fatal(err)
			}
			var instance any
			if err := strictjson.Decode(data, &instance); err != nil {
				t.Fatal(err)
			}
			if err := compiled.Validate(instance); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func validTestCapture(t *testing.T, target CaptureTarget) *CaptureDocument {
	t.Helper()
	metadata := DiscoveredColumnMetadata{
		"COLUMNS": {
			"TABLE_NAME":  {Name: "TABLE_NAME", SpannerType: "STRING(MAX)", OrdinalPosition: 1},
			"COLUMN_NAME": {Name: "COLUMN_NAME", SpannerType: "STRING(MAX)", OrdinalPosition: 2},
		},
	}
	queryability := []RollingQueryability{
		{TableName: "INDEXES", ColumnName: "SEARCH_UNNEST", Status: "not_advertised"},
		{TableName: "INDEX_COLUMNS", ColumnName: "EXPRESSION", Status: "not_advertised"},
	}
	document, err := BuildCapture(
		metadata,
		queryability,
		time.Date(2026, 8, 25, 8, 30, 12, 123456789, time.UTC),
		target,
		ProducerIdentity{
			SourceSHA256:     strings.Repeat("a", 64),
			InvocationSHA256: strings.Repeat("b", 64),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func refreshTestSurfaceHash(t *testing.T, document *CaptureDocument) {
	t.Helper()
	hash, err := hashJSON(captureSurface{
		Columns:             document.Columns,
		RollingQueryability: document.RollingQueryability,
	})
	if err != nil {
		t.Fatal(err)
	}
	document.SurfaceSHA256 = hash
}
