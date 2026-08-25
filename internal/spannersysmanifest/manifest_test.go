package spannersysmanifest

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestDecodeRepositoryManifest(t *testing.T) {
	document, err := Decode(repositoryManifest(t))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got, want := document.Source.Commit, "91908d001349f844aac070cc6518119c0e3c36c0"; got != want {
		t.Fatalf("source commit = %q, want %q", got, want)
	}
	if got, want := document.ContentSHA256, "2b337e1dc3d935c39249c139a1847200c0959dd280d41a286d76d80fb5b3e3a2"; got != want {
		t.Fatalf("content SHA-256 = %q, want %q", got, want)
	}

	tableCount := len(document.Tables)
	columnCount := 0
	projectedTables := 0
	projectedColumns := 0
	absentColumns := 0
	for _, table := range document.Tables {
		columnCount += len(table.Columns)
		if table.Project {
			projectedTables++
		}
		for _, column := range table.Columns {
			if column.Project {
				projectedColumns++
			} else {
				absentColumns++
			}
		}
	}
	if got, want := tableCount, 51; got != want {
		t.Errorf("table count = %d, want %d", got, want)
	}
	if got, want := columnCount, 547; got != want {
		t.Errorf("column count = %d, want %d", got, want)
	}
	if got, want := projectedTables, 50; got != want {
		t.Errorf("projected table count = %d, want %d", got, want)
	}
	if got, want := projectedColumns, 539; got != want {
		t.Errorf("projected column count = %d, want %d", got, want)
	}
	if got, want := absentColumns, 8; got != want {
		t.Errorf("absent column count = %d, want %d", got, want)
	}
	if got, want := len(document.Documentation), 11; got != want {
		t.Errorf("documentation conflict count = %d, want %d", got, want)
	}
}

func TestDecodeRejectsMalformedManifest(t *testing.T) {
	original := repositoryManifest(t)
	tests := []struct {
		name      string
		data      func(t *testing.T) []byte
		wantError string
	}{
		{
			name: "unknown field",
			data: func(t *testing.T) []byte {
				return []byte(strings.Replace(string(original), `"schema_version":`, `"unknown": true, "schema_version":`, 1))
			},
			wantError: "unknown field",
		},
		{
			name: "duplicate key",
			data: func(t *testing.T) []byte {
				return []byte(strings.Replace(string(original), `"schema_version":`, `"schema_version": "v0alpha1", "schema_version":`, 1))
			},
			wantError: "duplicate key",
		},
		{
			name: "trailing value",
			data: func(t *testing.T) []byte {
				return append(append([]byte(nil), original...), []byte("{}")...)
			},
			wantError: "after root value",
		},
		{
			name: "stale content hash",
			data: func(t *testing.T) []byte {
				return []byte(strings.Replace(string(original), `"name": "TEXT"`, `"name": "TEXT_CHANGED"`, 1))
			},
			wantError: "content_sha256",
		},
		{
			name: "unknown scalar",
			data: func(t *testing.T) []byte {
				return []byte(strings.Replace(string(original), `"scalar": "STRING(MAX)"`, `"scalar": "NUMERIC"`, 1))
			},
			wantError: "unsupported scalar code",
		},
		{
			name: "target disagreement",
			data: func(t *testing.T) []byte {
				document := decodedWithoutValidation(t, original)
				document.Tables[0].Columns[0].Observations[1].SpannerType = "BYTES(MAX)"
				return encodeWithContentHash(t, document)
			},
			wantError: "target omni type",
		},
		{
			name: "noncontiguous ordinal",
			data: func(t *testing.T) []byte {
				document := decodedWithoutValidation(t, original)
				for index := range document.Tables[0].Columns[1].Observations {
					document.Tables[0].Columns[1].Observations[index].OrdinalPosition = 1
				}
				return encodeWithContentHash(t, document)
			},
			wantError: "ordinal = 1, want 2",
		},
		{
			name: "absent column projected",
			data: func(t *testing.T) []byte {
				document := decodedWithoutValidation(t, original)
				column := findColumn(t, document, "TABLE_SIZES_STATS_PER_LOCALITY_GROUP_1HOUR", "USED_BYTES")
				column.Project = true
				return encodeWithContentHash(t, document)
			},
			wantError: "absent_both but projected",
		},
		{
			name: "array without element",
			data: func(t *testing.T) []byte {
				document := decodedWithoutValidation(t, original)
				column := findColumn(t, document, "LOCK_STATS_TOP_MINUTE", "SAMPLE_LOCK_REQUESTS")
				column.Type.Element = nil
				return encodeWithContentHash(t, document)
			},
			wantError: "malformed array descriptor",
		},
		{
			name: "unknown descriptor field",
			data: func(t *testing.T) []byte {
				return []byte(strings.Replace(string(original), `"kind": "scalar",`, `"kind": "scalar", "element_decoder_nullable": false,`, 1))
			},
			wantError: "unknown field",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Decode(test.data(t))
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("Decode() error = %v, want substring %q", err, test.wantError)
			}
		})
	}
}

func repositoryManifest(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("../../spanner_sys_manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func decodedWithoutValidation(t *testing.T, data []byte) *Document {
	t.Helper()
	var document Document
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	return &document
}

func encodeWithContentHash(t *testing.T, document *Document) []byte {
	t.Helper()
	hash, err := contentSHA256(document.RequiredTargets, document.Tables)
	if err != nil {
		t.Fatal(err)
	}
	document.ContentSHA256 = hash
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(data, '\n')
}

func findColumn(t *testing.T, document *Document, tableName, columnName string) *Column {
	t.Helper()
	for tableIndex := range document.Tables {
		if document.Tables[tableIndex].Name != tableName {
			continue
		}
		for columnIndex := range document.Tables[tableIndex].Columns {
			if document.Tables[tableIndex].Columns[columnIndex].Name == columnName {
				return &document.Tables[tableIndex].Columns[columnIndex]
			}
		}
	}
	t.Fatalf("column %s.%s not found", tableName, columnName)
	return nil
}
