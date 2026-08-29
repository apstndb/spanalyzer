package infoschem

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
	"slices"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/apstndb/spanalyzer/survey/internal/capturemeta"
	"github.com/apstndb/spanalyzer/survey/internal/strictjson"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	captureSchemaVersion = "v0alpha1"
	captureCatalog       = "INFORMATION_SCHEMA"
	captureDialect       = "googlesql"
	closedTransactionErr = "cannot use a closed transaction"
)

var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// CaptureDocument is a redacted observation of one target's advertised
// INFORMATION_SCHEMA definition surface.
type CaptureDocument struct {
	SchemaVersion        string                `json:"schema_version"`
	Catalog              string                `json:"catalog"`
	Dialect              string                `json:"dialect"`
	Target               CaptureTarget         `json:"target"`
	ObservedAt           string                `json:"observed_at"`
	ProducerSourceSHA256 string                `json:"producer_source_sha256"`
	InvocationSHA256     string                `json:"invocation_sha256"`
	Query                string                `json:"query"`
	SurfaceSHA256        string                `json:"surface_sha256"`
	Columns              []CaptureColumn       `json:"columns"`
	RollingQueryability  []RollingQueryability `json:"rolling_queryability"`
}

// CaptureTarget identifies either a rolling managed observation or a
// retrospectively reproducible container release.
type CaptureTarget = capturemeta.Target

// ImageIdentity identifies the platform-specific OCI manifest actually run.
type ImageIdentity = capturemeta.ImageIdentity

// CaptureColumn is one complete INFORMATION_SCHEMA.COLUMNS tuple.
type CaptureColumn struct {
	TableName       string `json:"table_name"`
	ColumnName      string `json:"column_name"`
	SpannerType     string `json:"spanner_type"`
	OrdinalPosition int    `json:"ordinal_position"`
}

// RollingQueryability records the bounded result of resolving one registry
// column whose availability may roll out separately from its advertisement.
type RollingQueryability struct {
	TableName  string `json:"table_name"`
	ColumnName string `json:"column_name"`
	Status     string `json:"status"`
	StatusCode string `json:"status_code,omitempty"`
}

// ProducerIdentity binds a capture to the closed producer input set without
// claiming that hashes attest execution.
type ProducerIdentity struct {
	SourceSHA256     string
	InvocationSHA256 string
}

type captureSurface struct {
	Columns             []CaptureColumn       `json:"columns"`
	RollingQueryability []RollingQueryability `json:"rolling_queryability"`
}

type captureDefinition struct {
	SchemaVersion               string                  `json:"schema_version"`
	InvocationID                string                  `json:"invocation_id"`
	Query                       string                  `json:"query"`
	Execution                   captureExecution        `json:"execution"`
	RollingColumns              []rollingRegistryColumn `json:"rolling_columns"`
	RollingQueryabilityStatuses []string                `json:"rolling_queryability_statuses"`
	ProducerInputs              []string                `json:"producer_inputs"`
}

type captureExecution struct {
	GoWork     string `json:"go_work"`
	ModuleMode string `json:"module_mode"`
}

type sourceInputHash struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// CaptureFromTransaction observes the complete INFORMATION_SCHEMA surface and
// the bounded queryability of every registry column marked Rolling in one
// read-only transaction.
func CaptureFromTransaction(
	ctx context.Context,
	txn *spanner.ReadOnlyTransaction,
	target CaptureTarget,
	producer ProducerIdentity,
) (*CaptureDocument, error) {
	metadata, err := DiscoverColumnMetadataWithTxn(ctx, txn)
	if err != nil {
		return nil, fmt.Errorf("discover INFORMATION_SCHEMA columns: %w", err)
	}
	queryability, err := observeRollingQueryability(ctx, txn, metadata)
	if err != nil {
		return nil, err
	}
	observedAt, err := txn.Timestamp()
	if err != nil {
		return nil, fmt.Errorf("read INFORMATION_SCHEMA transaction timestamp: %w", err)
	}
	return BuildCapture(metadata, queryability, observedAt, target, producer)
}

// BuildCapture constructs and validates a canonical capture from already
// observed metadata. It is exported so capture tooling can remain thin and
// tests can avoid a live service.
func BuildCapture(
	metadata DiscoveredColumnMetadata,
	queryability []RollingQueryability,
	observedAt time.Time,
	target CaptureTarget,
	producer ProducerIdentity,
) (*CaptureDocument, error) {
	columns := make([]CaptureColumn, 0)
	for tableName, tableColumns := range metadata {
		for columnName, column := range tableColumns {
			columns = append(columns, CaptureColumn{
				TableName:       tableName,
				ColumnName:      columnName,
				SpannerType:     column.SpannerType,
				OrdinalPosition: column.OrdinalPosition,
			})
		}
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
	queryability = append([]RollingQueryability(nil), queryability...)
	sort.Slice(queryability, func(i, j int) bool {
		if queryability[i].TableName != queryability[j].TableName {
			return queryability[i].TableName < queryability[j].TableName
		}
		return queryability[i].ColumnName < queryability[j].ColumnName
	})

	surfaceHash, err := hashJSON(captureSurface{
		Columns:             columns,
		RollingQueryability: queryability,
	})
	if err != nil {
		return nil, fmt.Errorf("hash INFORMATION_SCHEMA capture surface: %w", err)
	}
	document := &CaptureDocument{
		SchemaVersion:        captureSchemaVersion,
		Catalog:              captureCatalog,
		Dialect:              captureDialect,
		Target:               target,
		ObservedAt:           observedAt.UTC().Format(time.RFC3339Nano),
		ProducerSourceSHA256: producer.SourceSHA256,
		InvocationSHA256:     producer.InvocationSHA256,
		Query:                ColumnDiscoveryQuery,
		SurfaceSHA256:        surfaceHash,
		Columns:              columns,
		RollingQueryability:  queryability,
	}
	if err := ValidateCapture(document); err != nil {
		return nil, err
	}
	return document, nil
}

func observeRollingQueryability(
	ctx context.Context,
	txn *spanner.ReadOnlyTransaction,
	metadata DiscoveredColumnMetadata,
) ([]RollingQueryability, error) {
	rolling := v0alpha1RollingColumns
	results := make([]RollingQueryability, 0, len(rolling))
	for _, column := range rolling {
		result := RollingQueryability{TableName: column.TableName, ColumnName: column.ColumnName}
		if _, advertised := metadata[column.TableName][column.ColumnName]; !advertised {
			result.Status = "not_advertised"
			results = append(results, result)
			continue
		}

		query := fmt.Sprintf(
			"SELECT `%s` FROM INFORMATION_SCHEMA.%s LIMIT 0",
			column.ColumnName,
			column.TableName,
		)
		iter := txn.Query(ctx, spanner.NewStatement(query))
		_, err := iter.Next()
		iter.Stop()
		if isClientClosedTransactionError(err) {
			return nil, fmt.Errorf("probe rolling column %s.%s: %w", column.TableName, column.ColumnName, err)
		}
		switch {
		case errors.Is(err, iterator.Done):
			result.Status = "queryable"
		case spanner.ErrCode(err) == codes.InvalidArgument:
			result.Status = "advertised_not_queryable"
			result.StatusCode = "INVALID_ARGUMENT"
		case spanner.ErrCode(err) == codes.Unimplemented:
			result.Status = "advertised_not_queryable"
			result.StatusCode = "UNIMPLEMENTED"
		default:
			return nil, fmt.Errorf("probe rolling column %s.%s: %w", column.TableName, column.ColumnName, err)
		}
		results = append(results, result)
	}
	return results, nil
}

// isClientClosedTransactionError prevents a local transaction-lifecycle error
// from being mistaken for a backend rollout where a column is advertised
// before its owning view accepts it. The exact description is part of the
// pinned Cloud Spanner Go client behavior used by this capture contract.
func isClientClosedTransactionError(err error) bool {
	return spanner.ErrCode(err) == codes.InvalidArgument && status.Convert(err).Message() == closedTransactionErr
}

type rollingRegistryColumn struct {
	TableName  string `json:"table_name"`
	ColumnName string `json:"column_name"`
}

var v0alpha1RollingColumns = []rollingRegistryColumn{
	{TableName: "INDEXES", ColumnName: "SEARCH_UNNEST"},
	{TableName: "INDEX_COLUMNS", ColumnName: "EXPRESSION"},
}

func rollingRegistryColumns() []rollingRegistryColumn {
	var columns []rollingRegistryColumn
	for _, table := range informationSchemaTables {
		for _, column := range table.Columns {
			if column.Rolling {
				columns = append(columns, rollingRegistryColumn{
					TableName:  table.Name,
					ColumnName: column.Name,
				})
			}
		}
	}
	sort.Slice(columns, func(i, j int) bool {
		if columns[i].TableName != columns[j].TableName {
			return columns[i].TableName < columns[j].TableName
		}
		return columns[i].ColumnName < columns[j].ColumnName
	})
	return columns
}

// DecodeCapture decodes and validates one strict capture document.
func DecodeCapture(data []byte) (*CaptureDocument, error) {
	var document CaptureDocument
	if err := strictjson.Decode(data, &document); err != nil {
		return nil, fmt.Errorf("decode INFORMATION_SCHEMA capture: %w", err)
	}
	if err := rejectDestinationIdentifiers(&document); err != nil {
		return nil, err
	}
	if err := ValidateCapture(&document); err != nil {
		return nil, err
	}
	return &document, nil
}

// EncodeCapture returns canonical indented JSON after validating the document.
func EncodeCapture(document *CaptureDocument) ([]byte, error) {
	if err := ValidateCapture(document); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode INFORMATION_SCHEMA capture: %w", err)
	}
	data = append(data, '\n')
	if _, err := DecodeCapture(data); err != nil {
		return nil, fmt.Errorf("validate encoded INFORMATION_SCHEMA capture: %w", err)
	}
	return data, nil
}

// ValidateCapture enforces the closed semantic contract independently of JSON
// Schema tooling.
func ValidateCapture(document *CaptureDocument) error {
	if document == nil {
		return errors.New("INFORMATION_SCHEMA capture is nil")
	}
	if document.SchemaVersion != captureSchemaVersion {
		return fmt.Errorf("capture schema_version = %q, want %q", document.SchemaVersion, captureSchemaVersion)
	}
	if document.Catalog != captureCatalog || document.Dialect != captureDialect {
		return fmt.Errorf("capture catalog/dialect = %q/%q, want %q/%q", document.Catalog, document.Dialect, captureCatalog, captureDialect)
	}
	if err := capturemeta.ValidateTarget(document.Target); err != nil {
		return err
	}
	observedAt, err := time.Parse(time.RFC3339Nano, document.ObservedAt)
	if err != nil || observedAt.Location() != time.UTC || observedAt.Format(time.RFC3339Nano) != document.ObservedAt {
		return fmt.Errorf("capture observed_at = %q, want canonical RFC3339 UTC", document.ObservedAt)
	}
	if err := validateSHA256("producer_source_sha256", document.ProducerSourceSHA256); err != nil {
		return err
	}
	if err := validateSHA256("invocation_sha256", document.InvocationSHA256); err != nil {
		return err
	}
	if document.Query != ColumnDiscoveryQuery {
		return errors.New("capture query does not match the canonical INFORMATION_SCHEMA discovery query")
	}
	if err := validateColumns(document.Columns); err != nil {
		return err
	}
	if err := validateRollingQueryability(document.RollingQueryability); err != nil {
		return err
	}
	advertised := make(map[string]bool, len(document.Columns))
	for _, column := range document.Columns {
		advertised[column.TableName+"\x00"+column.ColumnName] = true
	}
	for _, result := range document.RollingQueryability {
		present := advertised[result.TableName+"\x00"+result.ColumnName]
		if (result.Status == "not_advertised") == present {
			return fmt.Errorf(
				"capture rolling queryability %s.%s status %s disagrees with column advertisement",
				result.TableName,
				result.ColumnName,
				result.Status,
			)
		}
	}
	wantHash, err := hashJSON(captureSurface{
		Columns:             document.Columns,
		RollingQueryability: document.RollingQueryability,
	})
	if err != nil {
		return fmt.Errorf("hash INFORMATION_SCHEMA capture surface: %w", err)
	}
	if document.SurfaceSHA256 != wantHash {
		return fmt.Errorf("capture surface_sha256 = %q, want %q", document.SurfaceSHA256, wantHash)
	}
	return nil
}

func validateColumns(columns []CaptureColumn) error {
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

func validateRollingQueryability(results []RollingQueryability) error {
	want := v0alpha1RollingColumns
	if len(results) != len(want) {
		return fmt.Errorf("capture rolling_queryability count = %d, want %d", len(results), len(want))
	}
	for index, result := range results {
		if result.TableName != want[index].TableName || result.ColumnName != want[index].ColumnName {
			return fmt.Errorf("capture rolling_queryability[%d] = %s.%s, want %s.%s", index, result.TableName, result.ColumnName, want[index].TableName, want[index].ColumnName)
		}
		switch result.Status {
		case "queryable", "not_advertised":
			if result.StatusCode != "" {
				return fmt.Errorf("capture rolling queryability %s.%s status %s must omit status_code", result.TableName, result.ColumnName, result.Status)
			}
		case "advertised_not_queryable":
			if result.StatusCode != "INVALID_ARGUMENT" && result.StatusCode != "UNIMPLEMENTED" {
				return fmt.Errorf("capture rolling queryability %s.%s has unsupported status_code %q", result.TableName, result.ColumnName, result.StatusCode)
			}
		default:
			return fmt.Errorf("capture rolling queryability %s.%s has unsupported status %q", result.TableName, result.ColumnName, result.Status)
		}
	}
	return nil
}

// ExpectedCapturePath returns the canonical path relative to survey/infoschem.
func ExpectedCapturePath(document *CaptureDocument) (string, error) {
	if err := ValidateCapture(document); err != nil {
		return "", err
	}
	return capturemeta.ExpectedPath(document.Target, document.ObservedAt, document.SurfaceSHA256)
}

// ComputeProducerIdentity hashes the closed producer input set declared by the
// capture definition. Capture output files are intentionally not inputs.
func ComputeProducerIdentity(repoRoot string) (ProducerIdentity, error) {
	definitionPath := filepath.Join(repoRoot, "survey", "infoschem", "capture-definition.v0alpha1.json")
	definitionData, err := os.ReadFile(definitionPath)
	if err != nil {
		return ProducerIdentity{}, fmt.Errorf("read capture definition: %w", err)
	}
	var definition captureDefinition
	if err := strictjson.Decode(definitionData, &definition); err != nil {
		return ProducerIdentity{}, fmt.Errorf("decode capture definition: %w", err)
	}
	if definition.SchemaVersion != captureSchemaVersion || definition.InvocationID == "" || definition.Query != ColumnDiscoveryQuery {
		return ProducerIdentity{}, errors.New("capture definition identity or query is invalid")
	}
	if definition.Execution.GoWork != "off" || definition.Execution.ModuleMode != "readonly" {
		return ProducerIdentity{}, fmt.Errorf("capture definition execution = %#v, want GOWORK=off and module_mode=readonly", definition.Execution)
	}
	if !slices.Equal(definition.RollingColumns, v0alpha1RollingColumns) {
		return ProducerIdentity{}, fmt.Errorf("capture definition rolling columns = %v, want %v", definition.RollingColumns, v0alpha1RollingColumns)
	}
	if current := rollingRegistryColumns(); !slices.Equal(current, v0alpha1RollingColumns) {
		return ProducerIdentity{}, fmt.Errorf("current rolling registry columns = %v, v0alpha1 capture contract = %v; introduce a new capture contract version", current, v0alpha1RollingColumns)
	}
	wantStatuses := []string{"advertised_not_queryable", "not_advertised", "queryable"}
	if !slices.Equal(definition.RollingQueryabilityStatuses, wantStatuses) {
		return ProducerIdentity{}, fmt.Errorf("capture definition rolling statuses = %v, want %v", definition.RollingQueryabilityStatuses, wantStatuses)
	}
	if len(definition.ProducerInputs) == 0 {
		return ProducerIdentity{}, errors.New("capture definition has no producer_inputs")
	}

	inputPaths := make(map[string]bool)
	for _, pattern := range definition.ProducerInputs {
		if filepath.IsAbs(pattern) || strings.Contains(filepath.ToSlash(pattern), "../") {
			return ProducerIdentity{}, fmt.Errorf("capture definition producer input %q is not repository-relative", pattern)
		}
		matches, err := filepath.Glob(filepath.Join(repoRoot, filepath.FromSlash(pattern)))
		if err != nil {
			return ProducerIdentity{}, fmt.Errorf("expand producer input %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			return ProducerIdentity{}, fmt.Errorf("capture definition producer input %q matched no files", pattern)
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				return ProducerIdentity{}, fmt.Errorf("stat producer input %q: %w", match, err)
			}
			if info.Mode().IsRegular() && !strings.HasSuffix(match, "_test.go") {
				relative, err := filepath.Rel(repoRoot, match)
				if err != nil {
					return ProducerIdentity{}, fmt.Errorf("relativize producer input %q: %w", match, err)
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
	inputs := make([]sourceInputHash, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(path)))
		if err != nil {
			return ProducerIdentity{}, fmt.Errorf("read producer input %q: %w", path, err)
		}
		inputs = append(inputs, sourceInputHash{Path: path, SHA256: sumSHA256(data)})
	}
	sourceHash, err := hashJSON(inputs)
	if err != nil {
		return ProducerIdentity{}, fmt.Errorf("hash producer inputs: %w", err)
	}
	invocationHash, err := hashJSON(definition)
	if err != nil {
		return ProducerIdentity{}, fmt.Errorf("hash capture definition: %w", err)
	}
	return ProducerIdentity{SourceSHA256: sourceHash, InvocationSHA256: invocationHash}, nil
}

// FileSHA256 returns the lowercase SHA-256 of an encoded capture or selector.
func FileSHA256(data []byte) string {
	return sumSHA256(data)
}

func validateSHA256(label, value string) error {
	if !sha256Pattern.MatchString(value) {
		return fmt.Errorf("capture %s = %q, want 64 lowercase hexadecimal characters", label, value)
	}
	return nil
}

func hashJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return sumSHA256(data), nil
}

func sumSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func rejectDestinationIdentifiers(document *CaptureDocument) error {
	data, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("encode capture for destination-identifier validation: %w", err)
	}
	lower := bytes.ToLower(data)
	for _, marker := range [][]byte{
		[]byte("projects/"),
		[]byte("/instances/"),
		[]byte("/databases/"),
	} {
		if bytes.Contains(lower, marker) {
			return fmt.Errorf("capture contains forbidden destination identifier marker %q", marker)
		}
	}
	return nil
}
