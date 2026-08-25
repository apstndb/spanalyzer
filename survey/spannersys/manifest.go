package spannersys

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	manifestSchemaVersion = "v0alpha1"
	surveyRepository      = "github.com/apstndb/spanner-emulator-survey"
	surveySourcePath      = "spannersys"
)

type manifestDocument struct {
	SchemaVersion   string                  `json:"schema_version"`
	Source          manifestSource          `json:"source"`
	RequiredTargets []string                `json:"required_targets"`
	Captures        []manifestCapture       `json:"captures"`
	Documentation   []documentationEvidence `json:"documentation"`
	ContentSHA256   string                  `json:"content_sha256"`
	Tables          []manifestTable         `json:"tables"`
}

type manifestSource struct {
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
	Path       string `json:"path"`
}

type manifestCapture struct {
	Target         string `json:"target"`
	CapturedAt     string `json:"captured_at"`
	RuntimeVersion string `json:"runtime_version,omitempty"`
	Query          string `json:"query"`
	ContentSHA256  string `json:"content_sha256"`
}

type manifestTable struct {
	Name           string           `json:"name"`
	EvidenceStatus string           `json:"evidence_status"`
	Project        bool             `json:"project"`
	Columns        []manifestColumn `json:"columns"`
}

type manifestColumn struct {
	Name                 string                `json:"name"`
	Type                 typeDescriptor        `json:"type"`
	DecoderNullable      bool                  `json:"decoder_nullable"`
	CanonicalSpannerType string                `json:"canonical_spanner_type"`
	Observations         []manifestObservation `json:"observations"`
	EvidenceStatus       string                `json:"evidence_status"`
	Project              bool                  `json:"project"`
}

type manifestObservation struct {
	Target          string `json:"target"`
	SpannerType     string `json:"spanner_type"`
	OrdinalPosition int    `json:"ordinal_position"`
}

type documentationEvidence struct {
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
	RequiredTargets []string        `json:"required_targets"`
	Tables          []manifestTable `json:"tables"`
}

type decodedCapture struct {
	Document *captureDocument
}

// ExportManifest returns a deterministic prerelease SPANNER_SYS manifest
// derived only from the package registry and the embedded redacted captures.
// sourceCommit must identify the exact survey commit whose exporter is being
// run. ExportManifest never reads Git state and never contacts Spanner.
func ExportManifest(sourceCommit string) ([]byte, error) {
	embedded, err := loadEmbeddedCaptures()
	if err != nil {
		return nil, err
	}
	return exportManifest(sourceCommit, embedded)
}

func exportManifest(sourceCommit string, inputs []embeddedCapture) ([]byte, error) {
	if err := validateSourceCommit(sourceCommit); err != nil {
		return nil, err
	}

	captures, err := decodeRequiredCaptures(inputs)
	if err != nil {
		return nil, err
	}
	descriptors, err := registryDescriptors()
	if err != nil {
		return nil, fmt.Errorf("describe SPANNER_SYS registry: %w", err)
	}
	tables, err := buildManifestTables(descriptors, captures)
	if err != nil {
		return nil, err
	}
	documentation := manifestDocumentationEvidence()
	if err := validateDocumentationEvidence(documentation, tables); err != nil {
		return nil, fmt.Errorf("validate SPANNER_SYS documentation evidence: %w", err)
	}

	requiredTargets := append([]string(nil), requiredCaptureTargets...)
	contentHash, err := hashJSON(manifestContent{
		RequiredTargets: requiredTargets,
		Tables:          tables,
	})
	if err != nil {
		return nil, fmt.Errorf("hash SPANNER_SYS manifest content: %w", err)
	}

	document := manifestDocument{
		SchemaVersion: manifestSchemaVersion,
		Source: manifestSource{
			Repository: surveyRepository,
			Commit:     sourceCommit,
			Path:       surveySourcePath,
		},
		RequiredTargets: requiredTargets,
		Captures:        manifestCaptureSidecars(captures),
		Documentation:   documentation,
		ContentSHA256:   contentHash,
		Tables:          tables,
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode SPANNER_SYS manifest: %w", err)
	}
	return append(data, '\n'), nil
}

func validateSourceCommit(sourceCommit string) error {
	if len(sourceCommit) != 40 {
		return fmt.Errorf("source commit = %q, want 40 lowercase hexadecimal characters", sourceCommit)
	}
	for _, character := range sourceCommit {
		isDigit := character >= '0' && character <= '9'
		isLowerHex := character >= 'a' && character <= 'f'
		if !isDigit && !isLowerHex {
			return fmt.Errorf("source commit = %q, want 40 lowercase hexadecimal characters", sourceCommit)
		}
	}
	return nil
}

func decodeRequiredCaptures(inputs []embeddedCapture) (map[string]decodedCapture, error) {
	if len(inputs) != len(requiredCaptureTargets) {
		return nil, fmt.Errorf(
			"capture input count = %d, want %d required targets",
			len(inputs),
			len(requiredCaptureTargets),
		)
	}

	captures := make(map[string]decodedCapture, len(inputs))
	for _, input := range inputs {
		if input.Target == "" || input.Path == "" {
			return nil, errors.New("capture input has an empty target or path")
		}
		if _, ok := captures[input.Target]; ok {
			return nil, fmt.Errorf("duplicate capture input target %q", input.Target)
		}
		capture, err := decodeCapture(input.Data)
		if err != nil {
			return nil, fmt.Errorf("decode %s capture %q: %w", input.Target, input.Path, err)
		}
		if capture.Target != input.Target {
			return nil, fmt.Errorf(
				"capture input %q target = %q, want %q",
				input.Path,
				capture.Target,
				input.Target,
			)
		}
		if capture.RuntimeVersion != input.ExpectedRuntimeVersion {
			return nil, fmt.Errorf(
				"capture input %q runtime_version = %q, want %q",
				input.Path,
				capture.RuntimeVersion,
				input.ExpectedRuntimeVersion,
			)
		}
		captures[input.Target] = decodedCapture{Document: capture}
	}
	for _, target := range requiredCaptureTargets {
		if _, ok := captures[target]; !ok {
			return nil, fmt.Errorf("missing required capture target %q", target)
		}
	}
	return captures, nil
}

func manifestCaptureSidecars(captures map[string]decodedCapture) []manifestCapture {
	sidecars := make([]manifestCapture, 0, len(requiredCaptureTargets))
	for _, target := range requiredCaptureTargets {
		capture := captures[target].Document
		sidecars = append(sidecars, manifestCapture{
			Target:         capture.Target,
			CapturedAt:     capture.CapturedAt,
			RuntimeVersion: capture.RuntimeVersion,
			Query:          capture.Query,
			ContentSHA256:  capture.ContentSHA256,
		})
	}
	return sidecars
}

func buildManifestTables(
	descriptors []tableDescriptor,
	captures map[string]decodedCapture,
) ([]manifestTable, error) {
	descriptorColumns := make(map[string]map[string]bool, len(descriptors))
	for _, table := range descriptors {
		descriptorColumns[table.Name] = make(map[string]bool, len(table.Columns))
		for _, column := range table.Columns {
			descriptorColumns[table.Name][column.Name] = true
		}
	}

	observations := make(map[string]map[string]map[string]captureColumn, len(captures))
	for _, target := range requiredCaptureTargets {
		observations[target] = make(map[string]map[string]captureColumn)
		for _, column := range captures[target].Document.Columns {
			knownColumns, ok := descriptorColumns[column.TableName]
			if !ok {
				return nil, fmt.Errorf("%s capture contains unknown table %s", target, column.TableName)
			}
			if !knownColumns[column.ColumnName] {
				return nil, fmt.Errorf(
					"%s capture contains unknown column %s.%s",
					target,
					column.TableName,
					column.ColumnName,
				)
			}
			if observations[target][column.TableName] == nil {
				observations[target][column.TableName] = make(map[string]captureColumn)
			}
			observations[target][column.TableName][column.ColumnName] = column
		}
	}

	tables := make([]manifestTable, 0, len(descriptors))
	for _, table := range descriptors {
		columns := make([]manifestColumn, 0, len(table.Columns))
		projectedColumns := 0
		for _, descriptor := range table.Columns {
			column, err := buildManifestColumn(table.Name, descriptor, observations)
			if err != nil {
				return nil, err
			}
			if column.Project {
				projectedColumns++
			}
			columns = append(columns, column)
		}

		evidenceStatus := "observed_both"
		project := true
		if projectedColumns == 0 {
			evidenceStatus = "absent_both"
			project = false
		}
		tables = append(tables, manifestTable{
			Name:           table.Name,
			EvidenceStatus: evidenceStatus,
			Project:        project,
			Columns:        columns,
		})
	}
	return tables, nil
}

func buildManifestColumn(
	tableName string,
	descriptor columnDescriptor,
	observations map[string]map[string]map[string]captureColumn,
) (manifestColumn, error) {
	canonical, err := canonicalSpannerType(descriptor.Type)
	if err != nil {
		return manifestColumn{}, fmt.Errorf("render %s.%s descriptor: %w", tableName, descriptor.Name, err)
	}

	columnObservations := make([]manifestObservation, 0, len(requiredCaptureTargets))
	for _, target := range requiredCaptureTargets {
		observation, ok := observations[target][tableName][descriptor.Name]
		if !ok {
			continue
		}
		columnObservations = append(columnObservations, manifestObservation{
			Target:          target,
			SpannerType:     observation.SpannerType,
			OrdinalPosition: observation.OrdinalPosition,
		})
	}

	column := manifestColumn{
		Name:                 descriptor.Name,
		Type:                 descriptor.Type,
		DecoderNullable:      descriptor.DecoderNullable,
		CanonicalSpannerType: canonical,
		Observations:         columnObservations,
		EvidenceStatus:       "absent_both",
		Project:              false,
	}
	if len(columnObservations) == 0 {
		return column, nil
	}
	if len(columnObservations) != len(requiredCaptureTargets) {
		return manifestColumn{}, fmt.Errorf(
			"required targets disagree on presence of %s.%s",
			tableName,
			descriptor.Name,
		)
	}
	first := columnObservations[0]
	for _, observation := range columnObservations[1:] {
		if observation.SpannerType != first.SpannerType || observation.OrdinalPosition != first.OrdinalPosition {
			return manifestColumn{}, fmt.Errorf(
				"required targets disagree on type or ordinal of %s.%s: %s has %s at %d; %s has %s at %d",
				tableName,
				descriptor.Name,
				first.Target,
				first.SpannerType,
				first.OrdinalPosition,
				observation.Target,
				observation.SpannerType,
				observation.OrdinalPosition,
			)
		}
	}
	if first.SpannerType != canonical {
		return manifestColumn{}, fmt.Errorf(
			"canonical descriptor for %s.%s = %q, required targets report %q",
			tableName,
			descriptor.Name,
			canonical,
			first.SpannerType,
		)
	}
	column.EvidenceStatus = "observed_both"
	column.Project = true
	return column, nil
}

func manifestDocumentationEvidence() []documentationEvidence {
	const (
		documentUpdatedAt = "2026-08-15T02:44:18Z"
		retrievedAt       = "2026-08-25"
		activePDMURL      = "https://docs.cloud.google.com/spanner/docs/introspection/active-partitioned-dmls"
		lockStatsURL      = "https://docs.cloud.google.com/spanner/docs/introspection/lock-statistics"
		transactionURL    = "https://docs.cloud.google.com/spanner/docs/introspection/transaction-statistics"
	)

	evidence := []documentationEvidence{
		{
			TableName:             "ACTIVE_PARTITIONED_DMLS",
			ColumnName:            "TEXT_FINGERPRINT",
			URL:                   activePDMURL,
			DocumentUpdatedAt:     documentUpdatedAt,
			RetrievedAt:           retrievedAt,
			DocumentedSpannerType: "INT64",
			ObservedSpannerType:   "STRING(MAX)",
			ConflictsWithLive:     true,
		},
		{
			TableName:             "ACTIVE_PARTITIONED_DMLS",
			ColumnName:            "PROGRESS",
			URL:                   activePDMURL,
			DocumentUpdatedAt:     documentUpdatedAt,
			RetrievedAt:           retrievedAt,
			DocumentedSpannerType: "DOUBLE",
			ObservedSpannerType:   "STRING(MAX)",
			ConflictsWithLive:     true,
		},
	}

	lockDocumentedType := "ARRAY<STRUCT<column STRING, lock_mode STRING, transaction_tag STRING>>"
	lockObservedType := "ARRAY<STRUCT<COLUMN STRING(MAX), LOCK_MODE STRING(MAX), TRANSACTION_TAG STRING(MAX)>>"
	for _, suffix := range []string{"MINUTE", "10MINUTE", "HOUR"} {
		evidence = append(evidence, documentationEvidence{
			TableName:             "LOCK_STATS_TOP_" + suffix,
			ColumnName:            "SAMPLE_LOCK_REQUESTS",
			URL:                   lockStatsURL,
			DocumentUpdatedAt:     documentUpdatedAt,
			RetrievedAt:           retrievedAt,
			DocumentedSpannerType: lockDocumentedType,
			ObservedSpannerType:   lockObservedType,
			ConflictsWithLive:     true,
		})
	}

	operationsDocumentedType := "ARRAY<STRUCT<TABLE STRING(MAX), INSERT_OR_UPDATE_COUNT INT64, INSERT_OR_UPDATE_BYTES INT64>>"
	operationsObservedType := "ARRAY<STRUCT<TABLE_NAME STRING(MAX), INSERT_OR_UPDATE_COUNT INT64, INSERT_OR_UPDATE_BYTES INT64>>"
	for _, family := range []string{"TXN_STATS_TOP_", "TXN_STATS_TOTAL_"} {
		for _, suffix := range []string{"MINUTE", "10MINUTE", "HOUR"} {
			evidence = append(evidence, documentationEvidence{
				TableName:             family + suffix,
				ColumnName:            "OPERATIONS_BY_TABLE",
				URL:                   transactionURL,
				DocumentUpdatedAt:     documentUpdatedAt,
				RetrievedAt:           retrievedAt,
				DocumentedName:        "TABLE",
				DocumentedSpannerType: operationsDocumentedType,
				ObservedSpannerType:   operationsObservedType,
				ConflictsWithLive:     true,
			})
		}
	}
	return evidence
}

func validateDocumentationEvidence(
	documentation []documentationEvidence,
	tables []manifestTable,
) error {
	columns := make(map[string]manifestColumn)
	for _, table := range tables {
		for _, column := range table.Columns {
			columns[table.Name+"\x00"+column.Name] = column
		}
	}
	seen := make(map[string]bool, len(documentation))
	for _, item := range documentation {
		if item.TableName == "" || item.ColumnName == "" || item.URL == "" ||
			item.DocumentUpdatedAt == "" || item.RetrievedAt == "" ||
			item.DocumentedSpannerType == "" || item.ObservedSpannerType == "" {
			return errors.New("documentation evidence contains an empty required field")
		}
		if !strings.HasPrefix(item.URL, "https://docs.cloud.google.com/spanner/") {
			return fmt.Errorf("documentation evidence URL %q is not an official Spanner documentation URL", item.URL)
		}
		if err := validateDate("documentation retrieved_at", item.RetrievedAt); err != nil {
			return err
		}
		if _, err := time.Parse(time.RFC3339, item.DocumentUpdatedAt); err != nil {
			return fmt.Errorf("documentation document_updated_at = %q, want RFC3339: %w", item.DocumentUpdatedAt, err)
		}
		key := item.TableName + "\x00" + item.ColumnName
		if seen[key] {
			return fmt.Errorf("duplicate documentation evidence for %s.%s", item.TableName, item.ColumnName)
		}
		seen[key] = true
		column, ok := columns[key]
		if !ok {
			return fmt.Errorf("documentation evidence references unknown column %s.%s", item.TableName, item.ColumnName)
		}
		if !column.Project {
			return fmt.Errorf("documentation live-conflict evidence references non-projecting column %s.%s", item.TableName, item.ColumnName)
		}
		if column.CanonicalSpannerType != item.ObservedSpannerType {
			return fmt.Errorf(
				"documentation evidence observed type for %s.%s = %q, descriptor renders %q",
				item.TableName,
				item.ColumnName,
				item.ObservedSpannerType,
				column.CanonicalSpannerType,
			)
		}
		if !item.ConflictsWithLive || item.DocumentedSpannerType == item.ObservedSpannerType {
			return fmt.Errorf("documentation evidence for %s.%s does not describe a live conflict", item.TableName, item.ColumnName)
		}
	}
	return nil
}
