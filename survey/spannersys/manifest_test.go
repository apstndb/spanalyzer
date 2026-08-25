package spannersys

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

const syntheticSourceCommit = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestExportManifest(t *testing.T) {
	t.Parallel()

	first, err := ExportManifest(syntheticSourceCommit)
	if err != nil {
		t.Fatalf("ExportManifest(): %v", err)
	}
	second, err := ExportManifest(syntheticSourceCommit)
	if err != nil {
		t.Fatalf("second ExportManifest(): %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("repeated ExportManifest calls returned different bytes")
	}
	if !bytes.HasSuffix(first, []byte("\n")) {
		t.Error("ExportManifest result has no trailing newline")
	}
	if err := rejectDestinationIdentifiers(first); err != nil {
		t.Fatalf("ExportManifest result contains destination identifier: %v", err)
	}

	document := decodeManifestForTest(t, first)
	if document.SchemaVersion != manifestSchemaVersion {
		t.Errorf("schema_version = %q, want %q", document.SchemaVersion, manifestSchemaVersion)
	}
	if document.Source != (manifestSource{
		Repository: surveyRepository,
		Commit:     syntheticSourceCommit,
		Path:       surveySourcePath,
	}) {
		t.Errorf("source = %+v", document.Source)
	}
	if !slices.Equal(document.RequiredTargets, []string{"managed", "omni"}) {
		t.Errorf("required_targets = %v", document.RequiredTargets)
	}
	if len(document.Captures) != 2 {
		t.Fatalf("captures = %d, want 2", len(document.Captures))
	}
	if document.Captures[0].Target != "managed" || document.Captures[0].RuntimeVersion != "" {
		t.Errorf("managed capture sidecar = %+v", document.Captures[0])
	}
	if document.Captures[1].Target != "omni" || document.Captures[1].RuntimeVersion != "2026.r2.1-beta" {
		t.Errorf("Omni capture sidecar = %+v", document.Captures[1])
	}
	if document.Captures[0].ContentSHA256 != document.Captures[1].ContentSHA256 {
		t.Errorf("required-target capture hashes differ: %+v", document.Captures)
	}

	projectedTables := 0
	projectedColumns := 0
	totalColumns := 0
	absentColumns := make([]string, 0, 8)
	previousTable := ""
	for _, table := range document.Tables {
		if table.Name <= previousTable {
			t.Errorf("tables are not sorted: %q after %q", table.Name, previousTable)
		}
		previousTable = table.Name
		if table.Project {
			projectedTables++
		}
		for _, column := range table.Columns {
			totalColumns++
			if column.Project {
				projectedColumns++
				if column.EvidenceStatus != "observed_both" || len(column.Observations) != 2 {
					t.Errorf("projected %s.%s evidence = %q observations = %d", table.Name, column.Name, column.EvidenceStatus, len(column.Observations))
				}
				if column.Observations[0].Target != "managed" || column.Observations[1].Target != "omni" {
					t.Errorf("%s.%s observation order = %+v", table.Name, column.Name, column.Observations)
				}
				continue
			}
			if column.EvidenceStatus != "absent_both" || len(column.Observations) != 0 {
				t.Errorf("non-projecting %s.%s evidence = %q observations = %d", table.Name, column.Name, column.EvidenceStatus, len(column.Observations))
			}
			absentColumns = append(absentColumns, table.Name+"."+column.Name)
		}
	}
	if len(document.Tables) != 51 || totalColumns != 547 {
		t.Errorf("manifest surface = %d tables / %d columns, want 51 / 547", len(document.Tables), totalColumns)
	}
	if projectedTables != 50 || projectedColumns != 539 {
		t.Errorf("projected surface = %d tables / %d columns, want 50 / 539", projectedTables, projectedColumns)
	}
	wantAbsent := []string{
		"TABLE_SIZES_STATS_PER_LOCALITY_GROUP_1HOUR.USED_BYTES",
		"VECTOR_INDEX_STATS.START_TIME",
		"VECTOR_INDEX_STATS.VECTOR_INDEX_NAME",
		"VECTOR_INDEX_STATS.NUM_LEAVES",
		"VECTOR_INDEX_STATS.NUM_CLUSTERS_SAMPLED",
		"VECTOR_INDEX_STATS.NUM_ZERO_SIZE_CLUSTERS_SAMPLED",
		"VECTOR_INDEX_STATS.CLUSTER_SIZE_PERCENTILES",
		"VECTOR_INDEX_STATS.CLUSTER_AVERAGE_DISTANCE_TO_CENTROID_PERCENTILES",
	}
	if !slices.Equal(absentColumns, wantAbsent) {
		t.Errorf("absent_both columns =\n%v\nwant\n%v", absentColumns, wantAbsent)
	}

	wantContentHash, err := hashJSON(manifestContent{
		RequiredTargets: document.RequiredTargets,
		Tables:          document.Tables,
	})
	if err != nil {
		t.Fatalf("hash manifest content: %v", err)
	}
	if document.ContentSHA256 != wantContentHash {
		t.Errorf("content_sha256 = %q, want %q", document.ContentSHA256, wantContentHash)
	}
	if len(document.Documentation) != 11 {
		t.Errorf("documentation conflict rows = %d, want 11", len(document.Documentation))
	}
	for _, item := range document.Documentation {
		if !item.ConflictsWithLive {
			t.Errorf("documentation row does not preserve conflict: %+v", item)
		}
	}
}

func TestExportManifestFailsClosed(t *testing.T) {
	t.Parallel()

	baseline, err := loadEmbeddedCaptures()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		mutate    func(t *testing.T, captures []embeddedCapture) []embeddedCapture
		wantError string
	}{
		{
			name: "missing required target",
			mutate: func(t *testing.T, captures []embeddedCapture) []embeddedCapture {
				t.Helper()
				return captures[:1]
			},
			wantError: "capture input count",
		},
		{
			name: "duplicate required target",
			mutate: func(t *testing.T, captures []embeddedCapture) []embeddedCapture {
				t.Helper()
				captures[1].Target = "managed"
				return captures
			},
			wantError: "duplicate capture input target",
		},
		{
			name: "unknown advertised column",
			mutate: func(t *testing.T, captures []embeddedCapture) []embeddedCapture {
				t.Helper()
				capture := decodeCaptureForTest(t, captures[1].Data)
				capture.Columns[0].ColumnName = "UNKNOWN_COLUMN"
				captures[1].Data = encodeCaptureForTest(t, capture)
				return captures
			},
			wantError: "unknown column",
		},
		{
			name: "runtime version does not match evidence path",
			mutate: func(t *testing.T, captures []embeddedCapture) []embeddedCapture {
				t.Helper()
				capture := decodeCaptureForTest(t, captures[1].Data)
				capture.RuntimeVersion = "2026.r3-beta"
				captures[1].Data = encodeCaptureForTest(t, capture)
				return captures
			},
			wantError: "runtime_version",
		},
		{
			name: "target presence disagreement",
			mutate: func(t *testing.T, captures []embeddedCapture) []embeddedCapture {
				t.Helper()
				capture := decodeCaptureForTest(t, captures[1].Data)
				capture.Columns = capture.Columns[:len(capture.Columns)-1]
				captures[1].Data = encodeCaptureForTest(t, capture)
				return captures
			},
			wantError: "disagree on presence",
		},
		{
			name: "target type disagreement",
			mutate: func(t *testing.T, captures []embeddedCapture) []embeddedCapture {
				t.Helper()
				capture := decodeCaptureForTest(t, captures[1].Data)
				capture.Columns[0].SpannerType = "INT64"
				captures[1].Data = encodeCaptureForTest(t, capture)
				return captures
			},
			wantError: "disagree on type or ordinal",
		},
		{
			name: "target ordinal disagreement",
			mutate: func(t *testing.T, captures []embeddedCapture) []embeddedCapture {
				t.Helper()
				capture := decodeCaptureForTest(t, captures[1].Data)
				capture.Columns[0], capture.Columns[1] = capture.Columns[1], capture.Columns[0]
				capture.Columns[0].OrdinalPosition = 1
				capture.Columns[1].OrdinalPosition = 2
				captures[1].Data = encodeCaptureForTest(t, capture)
				return captures
			},
			wantError: "disagree on type or ordinal",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			captures := cloneEmbeddedCaptures(baseline)
			_, err := exportManifest(syntheticSourceCommit, test.mutate(t, captures))
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("exportManifest() error = %v, want substring %q", err, test.wantError)
			}
		})
	}
}

func TestDecodeCaptureStrictness(t *testing.T) {
	t.Parallel()

	embedded, err := loadEmbeddedCaptures()
	if err != nil {
		t.Fatal(err)
	}
	baseline := embedded[0].Data

	tests := []struct {
		name      string
		mutate    func(t *testing.T, data []byte) []byte
		wantError string
	}{
		{
			name: "unknown field",
			mutate: func(t *testing.T, data []byte) []byte {
				t.Helper()
				var document map[string]any
				if err := json.Unmarshal(data, &document); err != nil {
					t.Fatal(err)
				}
				document["database"] = "redacted"
				return marshalJSONForTest(t, document)
			},
			wantError: "unknown field",
		},
		{
			name: "duplicate key",
			mutate: func(t *testing.T, data []byte) []byte {
				t.Helper()
				return bytes.Replace(
					data,
					[]byte(`"target": "managed",`),
					[]byte(`"target": "managed", "target": "managed",`),
					1,
				)
			},
			wantError: "duplicate key",
		},
		{
			name: "trailing value",
			mutate: func(t *testing.T, data []byte) []byte {
				t.Helper()
				return append(append([]byte(nil), data...), []byte("{}")...)
			},
			wantError: "multiple JSON values",
		},
		{
			name: "destination identifier",
			mutate: func(t *testing.T, data []byte) []byte {
				t.Helper()
				capture := decodeCaptureForTest(t, data)
				capture.RuntimeVersion = "projects/example/instances/example/databases/example"
				return encodeCaptureForTest(t, capture)
			},
			wantError: "forbidden destination identifier",
		},
		{
			name: "escaped destination identifier",
			mutate: func(t *testing.T, data []byte) []byte {
				t.Helper()
				capture := decodeCaptureForTest(t, data)
				capture.RuntimeVersion = "projects/example/instances/example/databases/example"
				return bytes.ReplaceAll(encodeCaptureForTest(t, capture), []byte("/"), []byte(`\u002f`))
			},
			wantError: "forbidden destination identifier",
		},
		{
			name: "stale hash",
			mutate: func(t *testing.T, data []byte) []byte {
				t.Helper()
				capture := decodeCaptureForTest(t, data)
				capture.ContentSHA256 = strings.Repeat("0", 64)
				return marshalJSONForTest(t, capture)
			},
			wantError: "content_sha256",
		},
		{
			name: "noncanonical query",
			mutate: func(t *testing.T, data []byte) []byte {
				t.Helper()
				capture := decodeCaptureForTest(t, data)
				capture.Query += " "
				return marshalJSONForTest(t, capture)
			},
			wantError: "canonical SPANNER_SYS discovery query",
		},
		{
			name: "ordinal gap",
			mutate: func(t *testing.T, data []byte) []byte {
				t.Helper()
				capture := decodeCaptureForTest(t, data)
				capture.Columns[1].OrdinalPosition = 3
				return encodeCaptureForTest(t, capture)
			},
			wantError: "want 2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := decodeCapture(test.mutate(t, baseline))
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("decodeCapture() error = %v, want substring %q", err, test.wantError)
			}
		})
	}
}

func TestValidateSourceCommit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		commit string
		valid  bool
	}{
		{name: "lowercase hex", commit: syntheticSourceCommit, valid: true},
		{name: "empty", commit: ""},
		{name: "short", commit: strings.Repeat("a", 39)},
		{name: "uppercase", commit: strings.Repeat("A", 40)},
		{name: "nonhex", commit: strings.Repeat("g", 40)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := validateSourceCommit(test.commit)
			if (err == nil) != test.valid {
				t.Errorf("validateSourceCommit(%q) error = %v, valid = %t", test.commit, err, test.valid)
			}
		})
	}
}

func TestManifestSchemasValidateEvidenceAndExport(t *testing.T) {
	t.Parallel()

	embedded, err := loadEmbeddedCaptures()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := ExportManifest(syntheticSourceCommit)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		schemaPath string
		instances  [][]byte
	}{
		{
			name:       "capture schema",
			schemaPath: "../schemas/spanner-sys-capture.v0alpha1.schema.json",
			instances:  [][]byte{embedded[0].Data, embedded[1].Data},
		},
		{
			name:       "manifest schema",
			schemaPath: "../schemas/spanner-sys-manifest.v0alpha1.schema.json",
			instances:  [][]byte{manifest},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			schemaData, err := os.ReadFile(test.schemaPath)
			if err != nil {
				t.Fatal(err)
			}
			compiler := jsonschema.NewCompiler()
			compiler.AssertFormat = true
			const schemaURL = "file:///contract.schema.json"
			if err := compiler.AddResource(schemaURL, bytes.NewReader(schemaData)); err != nil {
				t.Fatalf("add schema resource: %v", err)
			}
			compiled, err := compiler.Compile(schemaURL)
			if err != nil {
				t.Fatalf("compile %s: %v", test.schemaPath, err)
			}
			for index, data := range test.instances {
				var instance any
				if err := json.Unmarshal(data, &instance); err != nil {
					t.Fatalf("decode instance %d: %v", index, err)
				}
				if err := compiled.Validate(instance); err != nil {
					t.Errorf("instance %d does not satisfy %s: %v", index, test.schemaPath, err)
				}
			}
		})
	}
}

func decodeManifestForTest(t *testing.T, data []byte) manifestDocument {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var document manifestDocument
	if err := decoder.Decode(&document); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		t.Fatalf("manifest trailer: %v", err)
	}
	return document
}

func decodeCaptureForTest(t *testing.T, data []byte) *captureDocument {
	t.Helper()
	capture, err := decodeCapture(data)
	if err != nil {
		t.Fatal(err)
	}
	return capture
}

func encodeCaptureForTest(t *testing.T, capture *captureDocument) []byte {
	t.Helper()
	hash, err := hashJSON(capture.Columns)
	if err != nil {
		t.Fatal(err)
	}
	capture.ContentSHA256 = hash
	return marshalJSONForTest(t, capture)
}

func marshalJSONForTest(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func cloneEmbeddedCaptures(captures []embeddedCapture) []embeddedCapture {
	cloned := make([]embeddedCapture, len(captures))
	for index, capture := range captures {
		cloned[index] = capture
		cloned[index].Data = append([]byte(nil), capture.Data...)
	}
	return cloned
}
