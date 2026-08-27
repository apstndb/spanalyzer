package spannersys

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/apstndb/spanalyzer/survey/internal/capturemeta"
	"github.com/apstndb/spanalyzer/survey/internal/strictjson"
)

const (
	// SurfaceCaptureSchemaVersion identifies the provenance-complete capture
	// contract. The legacy v0alpha1 sidecars remain inputs to the v0alpha1
	// exported manifest and are intentionally not reinterpreted as this format.
	SurfaceCaptureSchemaVersion = "v0alpha2"
	surfaceCaptureCatalog       = "SPANNER_SYS"
	surfaceCaptureDialect       = "googlesql"
)

var surfaceSHA256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// SurfaceCapture is a redacted observation of one target's advertised
// SPANNER_SYS column surface with closed producer provenance.
type SurfaceCapture struct {
	SchemaVersion        string                 `json:"schema_version"`
	Catalog              string                 `json:"catalog"`
	Dialect              string                 `json:"dialect"`
	Target               capturemeta.Target     `json:"target"`
	ObservedAt           string                 `json:"observed_at"`
	ProducerSourceSHA256 string                 `json:"producer_source_sha256"`
	InvocationSHA256     string                 `json:"invocation_sha256"`
	Query                string                 `json:"query"`
	SurfaceSHA256        string                 `json:"surface_sha256"`
	Columns              []SurfaceCaptureColumn `json:"columns"`
}

// SurfaceCaptureColumn is one complete SPANNER_SYS column-metadata tuple.
type SurfaceCaptureColumn struct {
	TableName       string `json:"table_name"`
	ColumnName      string `json:"column_name"`
	SpannerType     string `json:"spanner_type"`
	OrdinalPosition int    `json:"ordinal_position"`
}

// SurfaceCaptureProducerIdentity binds a capture to its closed producer input
// set without claiming that hashes attest execution.
type SurfaceCaptureProducerIdentity struct {
	SourceSHA256     string
	InvocationSHA256 string
}

// LegacySurfaceBaseline exposes only the redacted comparison fields from one
// embedded v0alpha1 capture. It does not promote the legacy document to the
// provenance-complete v0alpha2 contract.
type LegacySurfaceBaseline struct {
	SchemaVersion string
	Target        string
	ObservedAt    string
	RuntimeTag    string
	SurfaceSHA256 string
	ColumnCount   int
}

type surfaceCaptureDefinition struct {
	SchemaVersion  string                  `json:"schema_version"`
	InvocationID   string                  `json:"invocation_id"`
	Query          string                  `json:"query"`
	Execution      surfaceCaptureExecution `json:"execution"`
	ProducerInputs []string                `json:"producer_inputs"`
}

type surfaceCaptureExecution struct {
	GoWork     string `json:"go_work"`
	ModuleMode string `json:"module_mode"`
}

type surfaceCaptureInputHash struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// CaptureSurfaceFromTransaction observes the complete SPANNER_SYS column
// surface and read timestamp in one read-only transaction.
func CaptureSurfaceFromTransaction(
	ctx context.Context,
	txn *spanner.ReadOnlyTransaction,
	target capturemeta.Target,
	producer SurfaceCaptureProducerIdentity,
) (*SurfaceCapture, error) {
	columns, err := discoverSurfaceCaptureColumns(ctx, txn)
	if err != nil {
		return nil, fmt.Errorf("discover SPANNER_SYS columns: %w", err)
	}
	observedAt, err := txn.Timestamp()
	if err != nil {
		return nil, fmt.Errorf("read SPANNER_SYS transaction timestamp: %w", err)
	}
	return BuildSurfaceCapture(columns, observedAt, target, producer)
}

// BuildSurfaceCapture constructs and validates one canonical capture from
// already observed columns.
func BuildSurfaceCapture(
	observations []ColumnObservation,
	observedAt time.Time,
	target capturemeta.Target,
	producer SurfaceCaptureProducerIdentity,
) (*SurfaceCapture, error) {
	columns := make([]SurfaceCaptureColumn, 0, len(observations))
	for _, observation := range observations {
		columns = append(columns, SurfaceCaptureColumn{
			TableName: observation.TableName, ColumnName: observation.ColumnName,
			SpannerType: observation.SpannerType, OrdinalPosition: observation.OrdinalPosition,
		})
	}
	sort.Slice(columns, func(i, j int) bool {
		if columns[i].TableName != columns[j].TableName {
			return columns[i].TableName < columns[j].TableName
		}
		if columns[i].OrdinalPosition != columns[j].OrdinalPosition {
			return columns[i].OrdinalPosition < columns[j].OrdinalPosition
		}
		return columns[i].ColumnName < columns[j].ColumnName
	})
	surfaceHash, err := hashSurfaceCaptureJSON(columns)
	if err != nil {
		return nil, fmt.Errorf("hash SPANNER_SYS capture surface: %w", err)
	}
	document := &SurfaceCapture{
		SchemaVersion: SurfaceCaptureSchemaVersion, Catalog: surfaceCaptureCatalog,
		Dialect: surfaceCaptureDialect, Target: target,
		ObservedAt:           observedAt.UTC().Format(time.RFC3339Nano),
		ProducerSourceSHA256: producer.SourceSHA256, InvocationSHA256: producer.InvocationSHA256,
		Query: columnDiscoveryQuery, SurfaceSHA256: surfaceHash, Columns: columns,
	}
	if err := ValidateSurfaceCapture(document); err != nil {
		return nil, err
	}
	return document, nil
}

func discoverSurfaceCaptureColumns(ctx context.Context, txn *spanner.ReadOnlyTransaction) ([]ColumnObservation, error) {
	type row struct {
		TableName       string `spanner:"TABLE_NAME"`
		ColumnName      string `spanner:"COLUMN_NAME"`
		SpannerType     string `spanner:"SPANNER_TYPE"`
		OrdinalPosition int64  `spanner:"ORDINAL_POSITION"`
	}
	iter := txn.Query(ctx, spanner.NewStatement(columnDiscoveryQuery))
	defer iter.Stop()
	var rows []row
	if err := spanner.SelectAll(iter, &rows); err != nil {
		return nil, err
	}
	columns := make([]ColumnObservation, 0, len(rows))
	for _, row := range rows {
		columns = append(columns, ColumnObservation{
			TableName: row.TableName, ColumnName: row.ColumnName,
			SpannerType: row.SpannerType, OrdinalPosition: int(row.OrdinalPosition),
		})
	}
	return columns, nil
}

// DecodeSurfaceCapture strictly decodes and validates one v0alpha2 capture.
func DecodeSurfaceCapture(data []byte) (*SurfaceCapture, error) {
	var document SurfaceCapture
	if err := strictjson.Decode(data, &document); err != nil {
		return nil, fmt.Errorf("decode SPANNER_SYS capture: %w", err)
	}
	if err := rejectSurfaceCaptureDestinationIdentifiers(&document); err != nil {
		return nil, err
	}
	if err := ValidateSurfaceCapture(&document); err != nil {
		return nil, err
	}
	return &document, nil
}

// EncodeSurfaceCapture returns canonical indented JSON after validation.
func EncodeSurfaceCapture(document *SurfaceCapture) ([]byte, error) {
	if err := ValidateSurfaceCapture(document); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode SPANNER_SYS capture: %w", err)
	}
	data = append(data, '\n')
	if _, err := DecodeSurfaceCapture(data); err != nil {
		return nil, fmt.Errorf("validate encoded SPANNER_SYS capture: %w", err)
	}
	return data, nil
}

// ValidateSurfaceCapture enforces the semantic contract independently of JSON
// Schema tooling.
func ValidateSurfaceCapture(document *SurfaceCapture) error {
	if document == nil {
		return errors.New("SPANNER_SYS capture is nil")
	}
	if document.SchemaVersion != SurfaceCaptureSchemaVersion {
		return fmt.Errorf("capture schema_version = %q, want %q", document.SchemaVersion, SurfaceCaptureSchemaVersion)
	}
	if document.Catalog != surfaceCaptureCatalog || document.Dialect != surfaceCaptureDialect {
		return fmt.Errorf("capture catalog/dialect = %q/%q, want %q/%q", document.Catalog, document.Dialect, surfaceCaptureCatalog, surfaceCaptureDialect)
	}
	if document.Target.Kind != "managed" && document.Target.Kind != "omni" {
		return fmt.Errorf("SPANNER_SYS capture target = %q, want managed or omni", document.Target.Kind)
	}
	if err := capturemeta.ValidateTarget(document.Target); err != nil {
		return err
	}
	observedAt, err := time.Parse(time.RFC3339Nano, document.ObservedAt)
	if err != nil || observedAt.Location() != time.UTC || observedAt.Format(time.RFC3339Nano) != document.ObservedAt {
		return fmt.Errorf("capture observed_at = %q, want canonical RFC3339 UTC", document.ObservedAt)
	}
	if err := validateSurfaceCaptureSHA256("producer_source_sha256", document.ProducerSourceSHA256); err != nil {
		return err
	}
	if err := validateSurfaceCaptureSHA256("invocation_sha256", document.InvocationSHA256); err != nil {
		return err
	}
	if document.Query != columnDiscoveryQuery {
		return errors.New("capture query does not match the canonical SPANNER_SYS discovery query")
	}
	if err := validateSurfaceCaptureColumns(document.Columns); err != nil {
		return err
	}
	wantHash, err := hashSurfaceCaptureJSON(document.Columns)
	if err != nil {
		return fmt.Errorf("hash SPANNER_SYS capture surface: %w", err)
	}
	if document.SurfaceSHA256 != wantHash {
		return fmt.Errorf("capture surface_sha256 = %q, want %q", document.SurfaceSHA256, wantHash)
	}
	return nil
}

func validateSurfaceCaptureColumns(columns []SurfaceCaptureColumn) error {
	if len(columns) == 0 {
		return errors.New("capture has no columns")
	}
	seen := make(map[string]bool, len(columns))
	previousTable := ""
	previousOrdinal := 0
	for index, column := range columns {
		if column.TableName == "" || column.ColumnName == "" || column.SpannerType == "" {
			return fmt.Errorf("capture column %d has an empty table, column, or type", index)
		}
		if column.OrdinalPosition < 1 {
			return fmt.Errorf("capture column %s.%s ordinal_position = %d, want at least 1", column.TableName, column.ColumnName, column.OrdinalPosition)
		}
		key := column.TableName + "\x00" + column.ColumnName
		if seen[key] {
			return fmt.Errorf("capture has duplicate column %s.%s", column.TableName, column.ColumnName)
		}
		seen[key] = true
		switch {
		case previousTable == "" || column.TableName > previousTable:
			if column.OrdinalPosition != 1 {
				return fmt.Errorf("capture table %s starts at ordinal_position %d, want 1", column.TableName, column.OrdinalPosition)
			}
		case column.TableName < previousTable:
			return fmt.Errorf("capture columns are not sorted at %s.%s", column.TableName, column.ColumnName)
		case column.OrdinalPosition != previousOrdinal+1:
			return fmt.Errorf("capture column %s.%s ordinal_position = %d, want %d", column.TableName, column.ColumnName, column.OrdinalPosition, previousOrdinal+1)
		}
		previousTable = column.TableName
		previousOrdinal = column.OrdinalPosition
	}
	return nil
}

// ExpectedSurfaceCapturePath returns the canonical path relative to
// survey/spannersys.
func ExpectedSurfaceCapturePath(document *SurfaceCapture) (string, error) {
	if err := ValidateSurfaceCapture(document); err != nil {
		return "", err
	}
	return capturemeta.ExpectedPath(document.Target, document.ObservedAt, document.SurfaceSHA256)
}

// ComputeSurfaceCaptureProducerIdentity hashes the closed producer input set
// declared by the v0alpha2 capture definition.
func ComputeSurfaceCaptureProducerIdentity(repoRoot string) (SurfaceCaptureProducerIdentity, error) {
	definitionPath := filepath.Join(repoRoot, "survey", "spannersys", "capture-definition.v0alpha2.json")
	definitionData, err := os.ReadFile(definitionPath)
	if err != nil {
		return SurfaceCaptureProducerIdentity{}, fmt.Errorf("read SPANNER_SYS capture definition: %w", err)
	}
	var definition surfaceCaptureDefinition
	if err := strictjson.Decode(definitionData, &definition); err != nil {
		return SurfaceCaptureProducerIdentity{}, fmt.Errorf("decode SPANNER_SYS capture definition: %w", err)
	}
	if definition.SchemaVersion != SurfaceCaptureSchemaVersion || definition.InvocationID == "" || definition.Query != columnDiscoveryQuery {
		return SurfaceCaptureProducerIdentity{}, errors.New("SPANNER_SYS capture definition identity or query is invalid")
	}
	if definition.Execution.GoWork != "off" || definition.Execution.ModuleMode != "readonly" {
		return SurfaceCaptureProducerIdentity{}, fmt.Errorf("SPANNER_SYS capture definition execution = %#v, want GOWORK=off and module_mode=readonly", definition.Execution)
	}
	if len(definition.ProducerInputs) == 0 {
		return SurfaceCaptureProducerIdentity{}, errors.New("SPANNER_SYS capture definition has no producer_inputs")
	}
	inputPaths := make(map[string]bool)
	for _, pattern := range definition.ProducerInputs {
		if filepath.IsAbs(pattern) || strings.Contains(filepath.ToSlash(pattern), "../") {
			return SurfaceCaptureProducerIdentity{}, fmt.Errorf("SPANNER_SYS producer input %q is not repository-relative", pattern)
		}
		matches, err := filepath.Glob(filepath.Join(repoRoot, filepath.FromSlash(pattern)))
		if err != nil {
			return SurfaceCaptureProducerIdentity{}, fmt.Errorf("expand SPANNER_SYS producer input %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			return SurfaceCaptureProducerIdentity{}, fmt.Errorf("SPANNER_SYS producer input %q matched no files", pattern)
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				return SurfaceCaptureProducerIdentity{}, err
			}
			if info.Mode().IsRegular() && !strings.HasSuffix(match, "_test.go") {
				relative, err := filepath.Rel(repoRoot, match)
				if err != nil {
					return SurfaceCaptureProducerIdentity{}, err
				}
				inputPaths[filepath.ToSlash(relative)] = true
			}
		}
	}
	paths := make([]string, 0, len(inputPaths))
	for path := range inputPaths {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	inputs := make([]surfaceCaptureInputHash, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(path)))
		if err != nil {
			return SurfaceCaptureProducerIdentity{}, fmt.Errorf("read SPANNER_SYS producer input %q: %w", path, err)
		}
		inputs = append(inputs, surfaceCaptureInputHash{Path: path, SHA256: surfaceCaptureSum(data)})
	}
	sourceHash, err := hashSurfaceCaptureJSON(inputs)
	if err != nil {
		return SurfaceCaptureProducerIdentity{}, err
	}
	invocationHash, err := hashSurfaceCaptureJSON(definition)
	if err != nil {
		return SurfaceCaptureProducerIdentity{}, err
	}
	return SurfaceCaptureProducerIdentity{SourceSHA256: sourceHash, InvocationSHA256: invocationHash}, nil
}

// LoadLegacySurfaceBaseline returns the retained v0alpha1 baseline for target.
func LoadLegacySurfaceBaseline(target string) (LegacySurfaceBaseline, error) {
	inputs, err := loadEmbeddedCaptures()
	if err != nil {
		return LegacySurfaceBaseline{}, err
	}
	for _, input := range inputs {
		if input.Target != target {
			continue
		}
		document, err := decodeCapture(input.Data)
		if err != nil {
			return LegacySurfaceBaseline{}, err
		}
		return LegacySurfaceBaseline{
			SchemaVersion: document.SchemaVersion,
			Target:        document.Target,
			ObservedAt:    document.CapturedAt,
			RuntimeTag:    document.RuntimeVersion,
			SurfaceSHA256: document.ContentSHA256,
			ColumnCount:   len(document.Columns),
		}, nil
	}
	return LegacySurfaceBaseline{}, fmt.Errorf("no legacy SPANNER_SYS baseline for target %q", target)
}

func validateSurfaceCaptureSHA256(label, value string) error {
	if !surfaceSHA256Pattern.MatchString(value) {
		return fmt.Errorf("capture %s = %q, want 64 lowercase hexadecimal characters", label, value)
	}
	return nil
}

func hashSurfaceCaptureJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return surfaceCaptureSum(data), nil
}

func surfaceCaptureSum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func rejectSurfaceCaptureDestinationIdentifiers(document *SurfaceCapture) error {
	data, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("encode capture for destination-identifier validation: %w", err)
	}
	lower := bytes.ToLower(data)
	for _, marker := range [][]byte{[]byte("projects/"), []byte("/instances/"), []byte("/databases/")} {
		if bytes.Contains(lower, marker) {
			return fmt.Errorf("capture contains forbidden destination identifier marker %q", marker)
		}
	}
	return nil
}
