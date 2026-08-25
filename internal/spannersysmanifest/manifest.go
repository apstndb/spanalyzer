// Package spannersysmanifest decodes and validates the repository's pinned
// SPANNER_SYS survey manifest. It is internal so the prerelease evidence
// format does not become part of spanalyzer's public API.
package spannersysmanifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

const (
	SchemaVersion    = "v0alpha1"
	SourceRepository = "github.com/apstndb/spanner-emulator-survey"
	SourcePath       = "spannersys"
	CaptureQuery     = "SELECT TABLE_NAME, COLUMN_NAME, SPANNER_TYPE, ORDINAL_POSITION\n" +
		"FROM INFORMATION_SCHEMA.COLUMNS\n" +
		"WHERE TABLE_SCHEMA = 'SPANNER_SYS'\n" +
		"ORDER BY TABLE_NAME, ORDINAL_POSITION"
	maxTypeDepth = 32
)

var requiredTargets = [...]string{"managed", "omni"}

type Document struct {
	SchemaVersion   string                  `json:"schema_version"`
	Source          Source                  `json:"source"`
	RequiredTargets []string                `json:"required_targets"`
	Captures        []Capture               `json:"captures"`
	Documentation   []DocumentationEvidence `json:"documentation"`
	ContentSHA256   string                  `json:"content_sha256"`
	Tables          []Table                 `json:"tables"`
}

type Source struct {
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
	Path       string `json:"path"`
}

type Capture struct {
	Target         string `json:"target"`
	CapturedAt     string `json:"captured_at"`
	RuntimeVersion string `json:"runtime_version,omitempty"`
	Query          string `json:"query"`
	ContentSHA256  string `json:"content_sha256"`
}

type Table struct {
	Name           string   `json:"name"`
	EvidenceStatus string   `json:"evidence_status"`
	Project        bool     `json:"project"`
	Columns        []Column `json:"columns"`
}

type Column struct {
	Name                 string         `json:"name"`
	Type                 TypeDescriptor `json:"type"`
	DecoderNullable      bool           `json:"decoder_nullable"`
	CanonicalSpannerType string         `json:"canonical_spanner_type"`
	Observations         []Observation  `json:"observations"`
	EvidenceStatus       string         `json:"evidence_status"`
	Project              bool           `json:"project"`
}

type Observation struct {
	Target          string `json:"target"`
	SpannerType     string `json:"spanner_type"`
	OrdinalPosition int    `json:"ordinal_position"`
}

type TypeDescriptor struct {
	Kind                   string          `json:"kind"`
	Scalar                 string          `json:"scalar,omitempty"`
	Element                *TypeDescriptor `json:"element,omitempty"`
	ElementDecoderNullable bool            `json:"element_decoder_nullable,omitempty"`
	Fields                 []StructField   `json:"fields,omitempty"`
}

type StructField struct {
	Name            string         `json:"name"`
	Type            TypeDescriptor `json:"type"`
	DecoderNullable bool           `json:"decoder_nullable"`
}

type DocumentationEvidence struct {
	TableName             string `json:"table_name"`
	ColumnName            string `json:"column_name"`
	URL                   string `json:"url"`
	DocumentUpdatedAt     string `json:"document_updated_at"`
	RetrievedAt           string `json:"retrieved_at"`
	DocumentedName        string `json:"documented_name,omitempty"`
	DocumentedSpannerType string `json:"documented_spanner_type"`
	ObservedSpannerType   string `json:"observed_spanner_type"`
	ConflictsWithLive     bool   `json:"conflicts_with_live"`
}

type manifestContent struct {
	RequiredTargets []string `json:"required_targets"`
	Tables          []Table  `json:"tables"`
}

// Decode strictly decodes and validates one SPANNER_SYS manifest.
func Decode(data []byte) (*Document, error) {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return nil, fmt.Errorf("validate SPANNER_SYS manifest JSON keys: %w", err)
	}
	var document Document
	if err := decodeStrictJSON(data, &document); err != nil {
		return nil, fmt.Errorf("decode SPANNER_SYS manifest: %w", err)
	}
	if err := Validate(&document); err != nil {
		return nil, err
	}
	return &document, nil
}

// Validate checks provenance, evidence, projection, type, and hash invariants.
func Validate(document *Document) error {
	if document == nil {
		return errors.New("SPANNER_SYS manifest is nil")
	}
	if document.SchemaVersion != SchemaVersion {
		return fmt.Errorf("SPANNER_SYS manifest schema_version = %q, want %q", document.SchemaVersion, SchemaVersion)
	}
	if document.Source.Repository != SourceRepository {
		return fmt.Errorf("SPANNER_SYS manifest source.repository = %q, want %q", document.Source.Repository, SourceRepository)
	}
	if document.Source.Path != SourcePath {
		return fmt.Errorf("SPANNER_SYS manifest source.path = %q, want %q", document.Source.Path, SourcePath)
	}
	if err := validateLowerHex("source.commit", document.Source.Commit, 20); err != nil {
		return err
	}
	if err := validateRequiredTargets(document.RequiredTargets); err != nil {
		return err
	}
	if err := validateCaptures(document.Captures); err != nil {
		return err
	}
	columns, err := validateTables(document.Tables)
	if err != nil {
		return err
	}
	if err := validateDocumentation(document.Documentation, columns); err != nil {
		return err
	}
	if err := validateLowerHex("content_sha256", document.ContentSHA256, sha256.Size); err != nil {
		return err
	}
	wantHash, err := contentSHA256(document.RequiredTargets, document.Tables)
	if err != nil {
		return err
	}
	if document.ContentSHA256 != wantHash {
		return fmt.Errorf("SPANNER_SYS manifest content_sha256 = %q, want %q", document.ContentSHA256, wantHash)
	}
	return nil
}

// CanonicalSpannerType renders a validated structural descriptor using the
// spelling returned by INFORMATION_SCHEMA.COLUMNS.SPANNER_TYPE.
func CanonicalSpannerType(descriptor TypeDescriptor) (string, error) {
	return canonicalSpannerType(descriptor, 0)
}

func (descriptor *TypeDescriptor) UnmarshalJSON(data []byte) error {
	var header struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return err
	}
	switch header.Kind {
	case "scalar":
		var scalar struct {
			Kind   string `json:"kind"`
			Scalar string `json:"scalar"`
		}
		if err := decodeStrictJSON(data, &scalar); err != nil {
			return err
		}
		*descriptor = TypeDescriptor{Kind: scalar.Kind, Scalar: scalar.Scalar}
	case "array":
		var array struct {
			Kind                   string          `json:"kind"`
			Element                *TypeDescriptor `json:"element"`
			ElementDecoderNullable bool            `json:"element_decoder_nullable,omitempty"`
		}
		if err := decodeStrictJSON(data, &array); err != nil {
			return err
		}
		*descriptor = TypeDescriptor{
			Kind:                   array.Kind,
			Element:                array.Element,
			ElementDecoderNullable: array.ElementDecoderNullable,
		}
	case "struct":
		var structure struct {
			Kind   string        `json:"kind"`
			Fields []StructField `json:"fields"`
		}
		if err := decodeStrictJSON(data, &structure); err != nil {
			return err
		}
		*descriptor = TypeDescriptor{Kind: structure.Kind, Fields: structure.Fields}
	default:
		return fmt.Errorf("unsupported descriptor kind %q", header.Kind)
	}
	return nil
}

func validateRequiredTargets(targets []string) error {
	if len(targets) != len(requiredTargets) {
		return fmt.Errorf("SPANNER_SYS manifest required_targets has %d entries, want %d", len(targets), len(requiredTargets))
	}
	for index, target := range requiredTargets {
		if targets[index] != target {
			return fmt.Errorf("SPANNER_SYS manifest required_targets[%d] = %q, want %q", index, targets[index], target)
		}
	}
	return nil
}

func validateCaptures(captures []Capture) error {
	if len(captures) != len(requiredTargets) {
		return fmt.Errorf("SPANNER_SYS manifest captures has %d entries, want %d", len(captures), len(requiredTargets))
	}
	for index, target := range requiredTargets {
		capture := captures[index]
		path := fmt.Sprintf("captures[%d]", index)
		if capture.Target != target {
			return fmt.Errorf("SPANNER_SYS manifest %s.target = %q, want %q", path, capture.Target, target)
		}
		if _, err := time.Parse(time.DateOnly, capture.CapturedAt); err != nil {
			return fmt.Errorf("SPANNER_SYS manifest %s.captured_at = %q, want YYYY-MM-DD", path, capture.CapturedAt)
		}
		if capture.Query != CaptureQuery {
			return fmt.Errorf("SPANNER_SYS manifest %s.query differs from the required SPANNER_SYS column query", path)
		}
		if err := validateLowerHex(path+".content_sha256", capture.ContentSHA256, sha256.Size); err != nil {
			return err
		}
		switch target {
		case "managed":
			if capture.RuntimeVersion != "" {
				return fmt.Errorf("SPANNER_SYS manifest %s.runtime_version = %q, want omitted for managed", path, capture.RuntimeVersion)
			}
		case "omni":
			if capture.RuntimeVersion == "" {
				return fmt.Errorf("SPANNER_SYS manifest %s.runtime_version is empty for Omni", path)
			}
		}
	}
	return nil
}

func validateTables(tables []Table) (map[string]Column, error) {
	if len(tables) == 0 {
		return nil, errors.New("SPANNER_SYS manifest has no tables")
	}
	allColumns := make(map[string]Column)
	seenTables := make(map[string]bool, len(tables))
	previousTable := ""
	for tableIndex, table := range tables {
		if err := validateName("table name", table.Name); err != nil {
			return nil, err
		}
		normalizedTableName := strings.ToUpper(table.Name)
		if seenTables[normalizedTableName] {
			return nil, fmt.Errorf("SPANNER_SYS manifest contains duplicate table %q", table.Name)
		}
		seenTables[normalizedTableName] = true
		if tableIndex > 0 && table.Name <= previousTable {
			return nil, fmt.Errorf("SPANNER_SYS manifest tables are not strictly sorted: %q follows %q", table.Name, previousTable)
		}
		previousTable = table.Name
		if len(table.Columns) == 0 {
			return nil, fmt.Errorf("SPANNER_SYS manifest table %q has no columns", table.Name)
		}

		seenColumns := make(map[string]bool, len(table.Columns))
		projectedColumns := 0
		for _, column := range table.Columns {
			if err := validateName("column name", column.Name); err != nil {
				return nil, fmt.Errorf("SPANNER_SYS manifest table %q: %w", table.Name, err)
			}
			normalizedColumnName := strings.ToUpper(column.Name)
			if seenColumns[normalizedColumnName] {
				return nil, fmt.Errorf("SPANNER_SYS manifest table %q contains duplicate column %q", table.Name, column.Name)
			}
			seenColumns[normalizedColumnName] = true
			path := table.Name + "." + column.Name
			canonical, err := CanonicalSpannerType(column.Type)
			if err != nil {
				return nil, fmt.Errorf("SPANNER_SYS manifest column %s type: %w", path, err)
			}
			if column.CanonicalSpannerType != canonical {
				return nil, fmt.Errorf("SPANNER_SYS manifest column %s canonical_spanner_type = %q, descriptor renders %q", path, column.CanonicalSpannerType, canonical)
			}
			if err := validateColumnEvidence(path, column, projectedColumns+1); err != nil {
				return nil, err
			}
			if column.Project {
				projectedColumns++
			}
			allColumns[path] = column
		}

		wantProject := projectedColumns > 0
		wantStatus := "absent_both"
		if wantProject {
			wantStatus = "observed_both"
		}
		if table.Project != wantProject || table.EvidenceStatus != wantStatus {
			return nil, fmt.Errorf("SPANNER_SYS manifest table %q evidence = {%s project=%t}, want {%s project=%t}", table.Name, table.EvidenceStatus, table.Project, wantStatus, wantProject)
		}
	}
	return allColumns, nil
}

func validateColumnEvidence(path string, column Column, projectedOrdinal int) error {
	switch column.EvidenceStatus {
	case "observed_both":
		if !column.Project {
			return fmt.Errorf("SPANNER_SYS manifest column %s is observed_both but not projected", path)
		}
		if len(column.Observations) != len(requiredTargets) {
			return fmt.Errorf("SPANNER_SYS manifest column %s has %d observations, want %d", path, len(column.Observations), len(requiredTargets))
		}
		for index, target := range requiredTargets {
			observation := column.Observations[index]
			if observation.Target != target {
				return fmt.Errorf("SPANNER_SYS manifest column %s observation[%d].target = %q, want %q", path, index, observation.Target, target)
			}
			if observation.SpannerType != column.CanonicalSpannerType {
				return fmt.Errorf("SPANNER_SYS manifest column %s target %s type = %q, want %q", path, target, observation.SpannerType, column.CanonicalSpannerType)
			}
			if observation.OrdinalPosition != projectedOrdinal {
				return fmt.Errorf("SPANNER_SYS manifest column %s target %s ordinal = %d, want %d", path, target, observation.OrdinalPosition, projectedOrdinal)
			}
		}
	case "absent_both":
		if column.Project {
			return fmt.Errorf("SPANNER_SYS manifest column %s is absent_both but projected", path)
		}
		if len(column.Observations) != 0 {
			return fmt.Errorf("SPANNER_SYS manifest column %s is absent_both but has observations", path)
		}
	default:
		return fmt.Errorf("SPANNER_SYS manifest column %s has unsupported evidence_status %q", path, column.EvidenceStatus)
	}
	return nil
}

func validateDocumentation(documentation []DocumentationEvidence, columns map[string]Column) error {
	seen := make(map[string]bool, len(documentation))
	for _, item := range documentation {
		if err := validateName("documentation table_name", item.TableName); err != nil {
			return err
		}
		if err := validateName("documentation column_name", item.ColumnName); err != nil {
			return err
		}
		if item.DocumentedSpannerType == "" || item.ObservedSpannerType == "" {
			return errors.New("SPANNER_SYS manifest documentation contains an empty type")
		}
		if item.DocumentedName != "" {
			if err := validateName("documentation documented_name", item.DocumentedName); err != nil {
				return err
			}
		}
		parsedURL, err := url.ParseRequestURI(item.URL)
		if err != nil || parsedURL.Scheme != "https" || parsedURL.Host != "docs.cloud.google.com" || !strings.HasPrefix(parsedURL.Path, "/spanner/") {
			return fmt.Errorf("SPANNER_SYS manifest documentation URL %q is not an official Spanner documentation URL", item.URL)
		}
		if _, err := time.Parse(time.RFC3339, item.DocumentUpdatedAt); err != nil {
			return fmt.Errorf("SPANNER_SYS manifest documentation document_updated_at = %q, want RFC3339", item.DocumentUpdatedAt)
		}
		if _, err := time.Parse(time.DateOnly, item.RetrievedAt); err != nil {
			return fmt.Errorf("SPANNER_SYS manifest documentation retrieved_at = %q, want YYYY-MM-DD", item.RetrievedAt)
		}
		path := item.TableName + "." + item.ColumnName
		if seen[path] {
			return fmt.Errorf("SPANNER_SYS manifest contains duplicate documentation evidence for %s", path)
		}
		seen[path] = true
		column, ok := columns[path]
		if !ok {
			return fmt.Errorf("SPANNER_SYS manifest documentation references unknown column %s", path)
		}
		if !column.Project {
			return fmt.Errorf("SPANNER_SYS manifest documentation references non-projecting column %s", path)
		}
		if item.ObservedSpannerType != column.CanonicalSpannerType {
			return fmt.Errorf("SPANNER_SYS manifest documentation observed type for %s = %q, want %q", path, item.ObservedSpannerType, column.CanonicalSpannerType)
		}
		if !item.ConflictsWithLive || item.DocumentedSpannerType == item.ObservedSpannerType {
			return fmt.Errorf("SPANNER_SYS manifest documentation for %s does not describe a live conflict", path)
		}
	}
	return nil
}

func canonicalSpannerType(descriptor TypeDescriptor, depth int) (string, error) {
	if depth > maxTypeDepth {
		return "", fmt.Errorf("type nesting exceeds %d levels", maxTypeDepth)
	}
	switch descriptor.Kind {
	case "scalar":
		if descriptor.Element != nil || len(descriptor.Fields) != 0 || descriptor.ElementDecoderNullable {
			return "", errors.New("scalar descriptor contains non-scalar fields")
		}
		switch descriptor.Scalar {
		case "BOOL", "INT64", "FLOAT64", "STRING(MAX)", "BYTES(MAX)", "DATE", "TIMESTAMP":
			return descriptor.Scalar, nil
		default:
			return "", fmt.Errorf("unsupported scalar code %q", descriptor.Scalar)
		}
	case "array":
		if descriptor.Scalar != "" || descriptor.Element == nil || len(descriptor.Fields) != 0 {
			return "", errors.New("malformed array descriptor")
		}
		if descriptor.Element.Kind == "array" {
			return "", errors.New("nested arrays are not supported")
		}
		element, err := canonicalSpannerType(*descriptor.Element, depth+1)
		if err != nil {
			return "", fmt.Errorf("array element: %w", err)
		}
		return "ARRAY<" + element + ">", nil
	case "struct":
		if descriptor.Scalar != "" || descriptor.Element != nil || descriptor.ElementDecoderNullable || len(descriptor.Fields) == 0 {
			return "", errors.New("malformed struct descriptor")
		}
		fields := make([]string, 0, len(descriptor.Fields))
		seen := make(map[string]bool, len(descriptor.Fields))
		for _, field := range descriptor.Fields {
			if err := validateName("struct field", field.Name); err != nil {
				return "", err
			}
			normalizedFieldName := strings.ToUpper(field.Name)
			if seen[normalizedFieldName] {
				return "", fmt.Errorf("duplicate struct field %q", field.Name)
			}
			seen[normalizedFieldName] = true
			fieldType, err := canonicalSpannerType(field.Type, depth+1)
			if err != nil {
				return "", fmt.Errorf("struct field %s: %w", field.Name, err)
			}
			fields = append(fields, field.Name+" "+fieldType)
		}
		return "STRUCT<" + strings.Join(fields, ", ") + ">", nil
	default:
		return "", fmt.Errorf("unsupported descriptor kind %q", descriptor.Kind)
	}
}

func contentSHA256(targets []string, tables []Table) (string, error) {
	data, err := json.Marshal(manifestContent{RequiredTargets: targets, Tables: tables})
	if err != nil {
		return "", fmt.Errorf("marshal SPANNER_SYS manifest content for hashing: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func validateLowerHex(name, value string, byteLength int) error {
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != byteLength || value != strings.ToLower(value) {
		return fmt.Errorf("SPANNER_SYS manifest %s = %q, want %d lowercase hexadecimal characters", name, value, byteLength*2)
	}
	return nil
}

func validateName(label, value string) error {
	if value == "" || strings.TrimSpace(value) != value || strings.ContainsRune(value, '\x00') {
		return fmt.Errorf("SPANNER_SYS manifest %s = %q, want a non-empty trimmed value", label, value)
	}
	return nil
}

func decodeStrictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return fmt.Errorf("JSON trailer: %w", err)
	}
	return nil
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := scanJSONValue(decoder, "$"); err != nil {
		return err
	}
	if token, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("unexpected JSON token %v after root value", token)
		}
		return err
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder, path string) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		keys := make(map[string]bool)
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object %s contains a non-string key", path)
			}
			if keys[key] {
				return fmt.Errorf("object %s contains duplicate key %q", path, key)
			}
			keys[key] = true
			if err := scanJSONValue(decoder, path+"."+key); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim('}') {
			return fmt.Errorf("object %s has invalid closing token %v", path, closing)
		}
	case '[':
		index := 0
		for decoder.More() {
			if err := scanJSONValue(decoder, fmt.Sprintf("%s[%d]", path, index)); err != nil {
				return err
			}
			index++
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim(']') {
			return fmt.Errorf("array %s has invalid closing token %v", path, closing)
		}
	default:
		return fmt.Errorf("value %s starts with unexpected delimiter %q", path, delimiter)
	}
	return nil
}
