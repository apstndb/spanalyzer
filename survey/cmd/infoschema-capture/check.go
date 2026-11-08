package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/apstndb/spanalyzer/survey/internal/runtimepins"
)

// v0alpha2 adds repeated observations and structured surface deltas.
const comparisonSchemaVersion = "v0alpha2"

type comparisonReport struct {
	SchemaVersion  string                     `json:"schema_version"`
	Status         string                     `json:"status"`
	MaterialChange bool                       `json:"material_change"`
	Current        comparisonObservation      `json:"current"`
	Retained       comparisonObservation      `json:"retained"`
	ProducerChange *comparisonProducerDelta   `json:"producer_change,omitempty"`
	SurfaceDelta   *surfaceDelta              `json:"surface_delta,omitempty"`
	Observations   []comparisonObservationRef `json:"observations,omitempty"`
}

type comparisonObservationRef struct {
	ObservedAt    string `json:"observed_at"`
	SurfaceSHA256 string `json:"surface_sha256"`
	ColumnCount   int    `json:"column_count"`
}

type surfaceDelta struct {
	AddedColumns               []surfaceColumnRef          `json:"added_columns,omitempty"`
	RemovedColumns             []surfaceColumnRef          `json:"removed_columns,omitempty"`
	ChangedColumns             []surfaceColumnChange       `json:"changed_columns,omitempty"`
	RollingQueryabilityChanges []rollingQueryabilityChange `json:"rolling_queryability_changes,omitempty"`
}

type surfaceColumnRef struct {
	TableName       string `json:"table_name"`
	ColumnName      string `json:"column_name"`
	SpannerType     string `json:"spanner_type"`
	OrdinalPosition int    `json:"ordinal_position"`
}

type surfaceColumnChange struct {
	TableName  string              `json:"table_name"`
	ColumnName string              `json:"column_name"`
	Retained   surfaceColumnFields `json:"retained"`
	Current    surfaceColumnFields `json:"current"`
}

type surfaceColumnFields struct {
	SpannerType     string `json:"spanner_type"`
	OrdinalPosition int    `json:"ordinal_position"`
}

type rollingQueryabilityChange struct {
	TableName      string `json:"table_name"`
	ColumnName     string `json:"column_name"`
	RetainedStatus string `json:"retained_status"`
	CurrentStatus  string `json:"current_status"`
}

type columnIdentity struct {
	table  string
	column string
}

type comparisonObservation struct {
	Path                 string                          `json:"path,omitempty"`
	Target               infoschem.CaptureTarget         `json:"target"`
	ObservedAt           string                          `json:"observed_at"`
	SurfaceSHA256        string                          `json:"surface_sha256"`
	ColumnCount          int                             `json:"column_count"`
	RollingQueryability  []infoschem.RollingQueryability `json:"rolling_queryability"`
	ProducerSourceSHA256 string                          `json:"producer_source_sha256"`
	InvocationSHA256     string                          `json:"invocation_sha256"`
}

type comparisonProducerDelta struct {
	SourceChanged     bool `json:"source_changed"`
	InvocationChanged bool `json:"invocation_changed"`
}

func compareRetainedCapture(repoRoot string, current *infoschem.CaptureDocument) (*comparisonReport, error) {
	return compareRetainedObservations(repoRoot, []*infoschem.CaptureDocument{current})
}

func compareRetainedObservations(repoRoot string, observations []*infoschem.CaptureDocument) (*comparisonReport, error) {
	if err := validateObservations(observations); err != nil {
		return nil, err
	}
	retainedPath, retained, targetChanged, err := retainedBaseline(repoRoot, observations[0])
	if err != nil {
		return nil, err
	}
	return aggregateCheckObservations(retainedPath, retained, targetChanged, observations), nil
}

func validateObservations(observations []*infoschem.CaptureDocument) error {
	if len(observations) == 0 {
		return errors.New("no observations")
	}
	for index, observation := range observations {
		if observation == nil {
			return fmt.Errorf("observation %d is nil", index)
		}
	}
	return nil
}

func validateObservationsFlag(checkMode bool, observations int, observationsSet bool) error {
	if observationsSet && !checkMode {
		return errors.New("--observations is only valid in check mode")
	}
	if checkMode && (observations < 1 || observations > 10) {
		return fmt.Errorf("--observations must be between 1 and 10, got %d", observations)
	}
	return nil
}

func flagWasSet(flags *flag.FlagSet, name string) bool {
	found := false
	flags.Visit(func(item *flag.Flag) {
		if item.Name == name {
			found = true
		}
	})
	return found
}

// aggregateCheckObservations classifies one or more current observations
// against a retained baseline. It performs no I/O.
func aggregateCheckObservations(retainedPath string, retained *infoschem.CaptureDocument, targetChanged bool, observations []*infoschem.CaptureDocument) *comparisonReport {
	current := observations[0]
	report := &comparisonReport{
		SchemaVersion: comparisonSchemaVersion,
		Current:       comparisonObservationFromCapture("", current),
		Retained:      comparisonObservationFromCapture(retainedPath, retained),
	}
	if !observationSurfacesAgree(observations) {
		report.Status = "surface_inconsistent"
		report.MaterialChange = true
		report.Observations = observationSummaries(observations)
		if differing := firstObservationDifferingFrom(retained.SurfaceSHA256, observations); differing != nil {
			report.SurfaceDelta = computeSurfaceDelta(retained, differing)
		}
		return report
	}
	if targetChanged {
		report.Status = "target_identity_changed"
		report.MaterialChange = true
		return report
	}
	if current.SurfaceSHA256 != retained.SurfaceSHA256 {
		report.Status = "surface_changed"
		report.MaterialChange = true
		report.SurfaceDelta = computeSurfaceDelta(retained, current)
		return report
	}
	sourceChanged := current.ProducerSourceSHA256 != retained.ProducerSourceSHA256
	invocationChanged := current.InvocationSHA256 != retained.InvocationSHA256
	if sourceChanged || invocationChanged {
		report.Status = "producer_changed_only"
		report.ProducerChange = &comparisonProducerDelta{
			SourceChanged:     sourceChanged,
			InvocationChanged: invocationChanged,
		}
		return report
	}
	report.Status = "unchanged"
	return report
}

func observationSurfacesAgree(observations []*infoschem.CaptureDocument) bool {
	first := observations[0].SurfaceSHA256
	for _, observation := range observations[1:] {
		if observation.SurfaceSHA256 != first {
			return false
		}
	}
	return true
}

func firstObservationDifferingFrom(surfaceSHA256 string, observations []*infoschem.CaptureDocument) *infoschem.CaptureDocument {
	for _, observation := range observations {
		if observation.SurfaceSHA256 != surfaceSHA256 {
			return observation
		}
	}
	return nil
}

func observationSummaries(observations []*infoschem.CaptureDocument) []comparisonObservationRef {
	summaries := make([]comparisonObservationRef, 0, len(observations))
	for _, observation := range observations {
		summaries = append(summaries, comparisonObservationRef{
			ObservedAt:    observation.ObservedAt,
			SurfaceSHA256: observation.SurfaceSHA256,
			ColumnCount:   len(observation.Columns),
		})
	}
	return summaries
}

func computeSurfaceDelta(retained, current *infoschem.CaptureDocument) *surfaceDelta {
	delta := &surfaceDelta{}
	retainedColumns := indexCaptureColumns(retained.Columns)
	currentColumns := indexCaptureColumns(current.Columns)
	for _, key := range sortedColumnIdentities(retainedColumns, currentColumns) {
		retainedColumn, retainedOK := retainedColumns[key]
		currentColumn, currentOK := currentColumns[key]
		switch {
		case !retainedOK:
			delta.AddedColumns = append(delta.AddedColumns, surfaceColumnRefFrom(currentColumn))
		case !currentOK:
			delta.RemovedColumns = append(delta.RemovedColumns, surfaceColumnRefFrom(retainedColumn))
		case retainedColumn.SpannerType != currentColumn.SpannerType || retainedColumn.OrdinalPosition != currentColumn.OrdinalPosition:
			delta.ChangedColumns = append(delta.ChangedColumns, surfaceColumnChange{
				TableName:  key.table,
				ColumnName: key.column,
				Retained: surfaceColumnFields{
					SpannerType:     retainedColumn.SpannerType,
					OrdinalPosition: retainedColumn.OrdinalPosition,
				},
				Current: surfaceColumnFields{
					SpannerType:     currentColumn.SpannerType,
					OrdinalPosition: currentColumn.OrdinalPosition,
				},
			})
		}
	}
	retainedRolling := indexRollingQueryability(retained.RollingQueryability)
	currentRolling := indexRollingQueryability(current.RollingQueryability)
	for _, key := range sortedColumnIdentities(retainedRolling, currentRolling) {
		retainedStatus := rollingStatus(retainedRolling, key)
		currentStatus := rollingStatus(currentRolling, key)
		if retainedStatus == currentStatus {
			continue
		}
		delta.RollingQueryabilityChanges = append(delta.RollingQueryabilityChanges, rollingQueryabilityChange{
			TableName:      key.table,
			ColumnName:     key.column,
			RetainedStatus: retainedStatus,
			CurrentStatus:  currentStatus,
		})
	}
	return delta
}

func surfaceColumnRefFrom(column infoschem.CaptureColumn) surfaceColumnRef {
	return surfaceColumnRef{
		TableName:       column.TableName,
		ColumnName:      column.ColumnName,
		SpannerType:     column.SpannerType,
		OrdinalPosition: column.OrdinalPosition,
	}
}

func indexCaptureColumns(columns []infoschem.CaptureColumn) map[columnIdentity]infoschem.CaptureColumn {
	index := make(map[columnIdentity]infoschem.CaptureColumn, len(columns))
	for _, column := range columns {
		index[columnIdentity{table: column.TableName, column: column.ColumnName}] = column
	}
	return index
}

func indexRollingQueryability(results []infoschem.RollingQueryability) map[columnIdentity]infoschem.RollingQueryability {
	index := make(map[columnIdentity]infoschem.RollingQueryability, len(results))
	for _, result := range results {
		index[columnIdentity{table: result.TableName, column: result.ColumnName}] = result
	}
	return index
}

func rollingStatus(index map[columnIdentity]infoschem.RollingQueryability, key columnIdentity) string {
	result, ok := index[key]
	if !ok {
		return ""
	}
	return result.Status
}

func sortedColumnIdentities[T any](left, right map[columnIdentity]T) []columnIdentity {
	seen := make(map[columnIdentity]bool, len(left)+len(right))
	keys := make([]columnIdentity, 0, len(left)+len(right))
	for key := range left {
		seen[key] = true
		keys = append(keys, key)
	}
	for key := range right {
		if seen[key] {
			continue
		}
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].table != keys[j].table {
			return keys[i].table < keys[j].table
		}
		return keys[i].column < keys[j].column
	})
	return keys
}

func comparisonObservationFromCapture(path string, capture *infoschem.CaptureDocument) comparisonObservation {
	return comparisonObservation{
		Path:                 path,
		Target:               capture.Target,
		ObservedAt:           capture.ObservedAt,
		SurfaceSHA256:        capture.SurfaceSHA256,
		ColumnCount:          len(capture.Columns),
		RollingQueryability:  append([]infoschem.RollingQueryability(nil), capture.RollingQueryability...),
		ProducerSourceSHA256: capture.ProducerSourceSHA256,
		InvocationSHA256:     capture.InvocationSHA256,
	}
}

func retainedBaseline(repoRoot string, current *infoschem.CaptureDocument) (string, *infoschem.CaptureDocument, bool, error) {
	if current.Target.Kind == "managed" {
		path, err := selectedManagedObservationPath(repoRoot)
		if err != nil {
			return "", nil, false, err
		}
		capture, err := readRetainedCapture(repoRoot, path)
		return path, capture, false, err
	}
	if current.Target.Image == nil {
		return "", nil, false, fmt.Errorf("%s capture has no image identity", current.Target.Kind)
	}
	pinnedReference, err := runtimepins.ImageForPlatform(repoRoot, current.Target.Kind, current.Target.Image.Platform)
	if err != nil {
		return "", nil, false, err
	}
	family, tag, digest, err := splitTaggedImage(pinnedReference)
	if err != nil {
		return "", nil, false, err
	}
	pinned := infoschem.ImageIdentity{
		Family:   family,
		Tag:      tag,
		Digest:   digest,
		Platform: current.Target.Image.Platform,
	}
	path, capture, err := findContainerBaseline(repoRoot, current.Target.Kind, pinned)
	if err != nil {
		return "", nil, false, err
	}
	return path, capture, *current.Target.Image != pinned, nil
}

func selectedManagedObservationPath(repoRoot string) (string, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "information_schema_projection_source.json"))
	if err != nil {
		return "", fmt.Errorf("read selected managed observation: %w", err)
	}
	var source struct {
		SelectedObservation struct {
			Path string `json:"path"`
		} `json:"selected_observation"`
	}
	if err := json.Unmarshal(data, &source); err != nil {
		return "", fmt.Errorf("decode selected managed observation: %w", err)
	}
	path := filepath.ToSlash(filepath.Clean(source.SelectedObservation.Path))
	if path == "." || filepath.IsAbs(path) || path == ".." || len(path) >= 3 && path[:3] == "../" {
		return "", fmt.Errorf("selected managed observation path %q is not repository-relative", path)
	}
	return path, nil
}

func findContainerBaseline(repoRoot, kind string, image infoschem.ImageIdentity) (string, *infoschem.CaptureDocument, error) {
	root := filepath.Join(repoRoot, "survey", "infoschem", "evidence", kind)
	type candidate struct {
		path     string
		capture  *infoschem.CaptureDocument
		observed time.Time
	}
	var candidates []candidate
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		capture, err := infoschem.DecodeCapture(data)
		if err != nil {
			return fmt.Errorf("decode retained capture %q: %w", path, err)
		}
		if capture.Target.Kind != kind || capture.Target.Image == nil || *capture.Target.Image != image {
			return nil
		}
		observed, err := time.Parse(time.RFC3339Nano, capture.ObservedAt)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		candidates = append(candidates, candidate{path: filepath.ToSlash(relative), capture: capture, observed: observed})
		return nil
	})
	if err != nil {
		return "", nil, fmt.Errorf("scan retained %s captures: %w", kind, err)
	}
	if len(candidates) == 0 {
		return "", nil, fmt.Errorf("no retained %s capture matches pinned image %+v", kind, image)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].observed.After(candidates[j].observed) })
	return candidates[0].path, candidates[0].capture, nil
}

func readRetainedCapture(repoRoot, relative string) (*infoschem.CaptureDocument, error) {
	path := filepath.Join(repoRoot, filepath.FromSlash(relative))
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil || rel == ".." || len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator) {
		return nil, fmt.Errorf("retained capture path %q escapes repository root", relative)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read retained capture %q: %w", relative, err)
	}
	capture, err := infoschem.DecodeCapture(data)
	if err != nil {
		return nil, fmt.Errorf("decode retained capture %q: %w", relative, err)
	}
	return capture, nil
}

func writeComparisonReport(writer io.Writer, report *comparisonReport) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
