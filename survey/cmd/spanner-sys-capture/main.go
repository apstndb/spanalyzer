// Command spanner-sys-capture records or compares a provenance-complete,
// redacted SPANNER_SYS column-surface observation.
package main

import (
	"bytes"
	"context"
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

	"github.com/apstndb/spanalyzer/survey/internal/captureenv"
	"github.com/apstndb/spanalyzer/survey/internal/capturemeta"
	"github.com/apstndb/spanalyzer/survey/internal/runtimepins"
	"github.com/apstndb/spanalyzer/survey/spannersys"
)

const captureLabel = "io.github.apstndb.spanalyzer.spanner-sys-capture"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	checkMode := len(args) > 0 && args[0] == "check"
	if checkMode {
		args = args[1:]
	}
	flags := flag.NewFlagSet("spanner-sys-capture", flag.ContinueOnError)
	flags.SetOutput(stderr)
	target := flags.String("target", "", "capture target: managed or omni")
	database := flags.String("database", os.Getenv("TEST_REAL_SPANNER_DATABASE"), "managed Spanner database resource (defaults to TEST_REAL_SPANNER_DATABASE; never retained)")
	image := flags.String("image", "", "Omni image override with a descriptive tag and optional @sha256 digest; defaults to runtime_targets.json")
	repoRoot := flags.String("repo-root", "", "spanalyzer repository root (auto-detected when omitted)")
	output := flags.String("output", "", "canonical repository output path (stdout when omitted)")
	write := flags.Bool("write", false, "write to the canonical repository path")
	timeout := flags.Duration("timeout", 15*time.Minute, "overall capture timeout")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		writeDiagnostic(stderr, "unexpected positional arguments: %v\n", flags.Args())
		return 2
	}
	if *write && *output != "" {
		writeDiagnostic(stderr, "--write and --output are mutually exclusive\n")
		return 2
	}
	if checkMode && (*write || *output != "") {
		writeDiagnostic(stderr, "check does not accept --write or --output\n")
		return 2
	}
	if *target != "managed" && *target != "omni" {
		writeDiagnostic(stderr, "--target must be managed or omni\n")
		return 2
	}
	if *target == "managed" && *database == "" {
		writeDiagnostic(stderr, "--database or TEST_REAL_SPANNER_DATABASE is required for target managed\n")
		return 2
	}
	if err := captureenv.ValidateExecutionEnvironment(); err != nil {
		writeDiagnostic(stderr, "%v\n", err)
		return 2
	}
	root, err := resolveRepoRoot(*repoRoot)
	if err != nil {
		writeDiagnostic(stderr, "%v\n", err)
		return 1
	}
	if *target == "omni" && *image == "" {
		*image, err = runtimepins.ImageForHost(root, "omni")
		if err != nil {
			writeDiagnostic(stderr, "resolve pinned Omni image: %v\n", err)
			return 1
		}
	}
	producer, err := spannersys.ComputeSurfaceCaptureProducerIdentity(root)
	if err != nil {
		writeDiagnostic(stderr, "compute SPANNER_SYS producer identity: %v\n", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	document, err := capture(ctx, *target, *database, *image, producer)
	if err != nil {
		writeDiagnostic(stderr, "capture SPANNER_SYS: %v\n", err)
		return 1
	}
	data, err := spannersys.EncodeSurfaceCapture(document)
	if err != nil {
		writeDiagnostic(stderr, "encode SPANNER_SYS capture: %v\n", err)
		return 1
	}
	relative, err := spannersys.ExpectedSurfaceCapturePath(document)
	if err != nil {
		writeDiagnostic(stderr, "derive SPANNER_SYS capture path: %v\n", err)
		return 1
	}
	canonicalOutput := filepath.Join(root, "survey", "spannersys", filepath.FromSlash(relative))
	if checkMode {
		report, err := compareRetainedCapture(root, document)
		if err != nil {
			writeDiagnostic(stderr, "compare retained SPANNER_SYS capture: %v\n", err)
			return 1
		}
		if err := json.NewEncoder(stdout).Encode(report); err != nil {
			writeDiagnostic(stderr, "write SPANNER_SYS comparison: %v\n", err)
			return 1
		}
		if report.MaterialChange {
			return 1
		}
		return 0
	}
	if !*write && *output == "" {
		if _, err := stdout.Write(data); err != nil {
			writeDiagnostic(stderr, "write SPANNER_SYS capture: %v\n", err)
			return 1
		}
		writeDiagnostic(stderr, "canonical_path=%s\n", filepath.ToSlash(filepath.Join("survey", "spannersys", relative)))
		return 0
	}
	outputPath := canonicalOutput
	if *output != "" {
		outputPath, err = filepath.Abs(*output)
		if err != nil {
			writeDiagnostic(stderr, "resolve --output: %v\n", err)
			return 1
		}
	}
	wantPath, err := filepath.Abs(canonicalOutput)
	if err != nil {
		writeDiagnostic(stderr, "resolve canonical output: %v\n", err)
		return 1
	}
	if outputPath != wantPath {
		writeDiagnostic(stderr, "--output = %q, want canonical path %q\n", outputPath, wantPath)
		return 2
	}
	wrote, err := writeCapture(outputPath, data, document)
	if err != nil {
		writeDiagnostic(stderr, "write SPANNER_SYS capture: %v\n", err)
		return 1
	}
	if wrote {
		writeDiagnostic(stderr, "wrote %s\n", filepath.ToSlash(filepath.Join("survey", "spannersys", relative)))
	} else {
		writeDiagnostic(stderr, "already retained equivalent observation %s\n", filepath.ToSlash(filepath.Join("survey", "spannersys", relative)))
	}
	return 0
}

func capture(ctx context.Context, targetKind, database, image string, producer spannersys.SurfaceCaptureProducerIdentity) (*spannersys.SurfaceCapture, error) {
	environment, err := captureenv.Open(ctx, targetKind, database, image, captureLabel)
	if err != nil {
		return nil, err
	}
	defer func() { _ = environment.Close() }()
	txn := environment.Client.ReadOnlyTransaction()
	defer txn.Close()
	return spannersys.CaptureSurfaceFromTransaction(ctx, txn, environment.Target, producer)
}

type comparisonReport struct {
	SchemaVersion  string                `json:"schema_version"`
	Status         string                `json:"status"`
	MaterialChange bool                  `json:"material_change"`
	Current        comparisonObservation `json:"current"`
	Retained       comparisonObservation `json:"retained"`
	SurfaceDelta   *surfaceDelta         `json:"surface_delta,omitempty"`
}

// surfaceDelta lists column-level differences for a surface_changed report.
// SPANNER_SYS captures have no rolling-queryability probes, so unlike the
// INFORMATION_SCHEMA report there is no rolling_queryability_changes field.
type surfaceDelta struct {
	AddedColumns   []surfaceColumnRef    `json:"added_columns,omitempty"`
	RemovedColumns []surfaceColumnRef    `json:"removed_columns,omitempty"`
	ChangedColumns []surfaceColumnChange `json:"changed_columns,omitempty"`
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

type columnIdentity struct {
	table  string
	column string
}

type comparisonObservation struct {
	Path                 string           `json:"path,omitempty"`
	SchemaVersion        string           `json:"schema_version"`
	Target               comparisonTarget `json:"target"`
	ObservedAt           string           `json:"observed_at"`
	SurfaceSHA256        string           `json:"surface_sha256"`
	ColumnCount          int              `json:"column_count"`
	ProducerSourceSHA256 string           `json:"producer_source_sha256,omitempty"`
	InvocationSHA256     string           `json:"invocation_sha256,omitempty"`
}

type comparisonTarget struct {
	Kind             string                     `json:"kind"`
	ObservationScope string                     `json:"observation_scope,omitempty"`
	Image            *capturemeta.ImageIdentity `json:"image,omitempty"`
	LegacyRuntimeTag string                     `json:"legacy_runtime_tag,omitempty"`
}

func comparisonTargetFromCapture(target capturemeta.Target) comparisonTarget {
	return comparisonTarget{
		Kind: target.Kind, ObservationScope: target.ObservationScope, Image: target.Image,
	}
}

func comparisonTargetFromLegacy(baseline spannersys.LegacySurfaceBaseline) comparisonTarget {
	target := comparisonTarget{Kind: baseline.Target, LegacyRuntimeTag: baseline.RuntimeTag}
	if baseline.Target == "managed" {
		target.ObservationScope = "single_database"
	}
	return target
}

func compareRetainedCapture(repoRoot string, current *spannersys.SurfaceCapture) (*comparisonReport, error) {
	baselineTarget := current.Target
	targetChanged := false
	if current.Target.Kind == "omni" {
		if current.Target.Image == nil {
			return nil, errors.New("current Omni capture has no image identity")
		}
		pinnedReference, err := runtimepins.ImageForPlatform(repoRoot, "omni", current.Target.Image.Platform)
		if err != nil {
			return nil, err
		}
		family, tag, digest, err := captureenv.SplitTaggedImage(pinnedReference)
		if err != nil {
			return nil, err
		}
		pinnedImage := capturemeta.ImageIdentity{
			Family: family, Tag: tag, Digest: digest, Platform: current.Target.Image.Platform,
		}
		targetChanged = *current.Target.Image != pinnedImage
		baselineTarget.Image = &pinnedImage
	}
	report := &comparisonReport{SchemaVersion: "v0alpha2", Current: comparisonObservation{
		SchemaVersion: current.SchemaVersion, Target: comparisonTargetFromCapture(current.Target), ObservedAt: current.ObservedAt,
		SurfaceSHA256: current.SurfaceSHA256, ColumnCount: len(current.Columns),
		ProducerSourceSHA256: current.ProducerSourceSHA256, InvocationSHA256: current.InvocationSHA256,
	}}
	path, retained, err := findV0Alpha2Baseline(repoRoot, baselineTarget)
	if err == nil {
		report.Retained = comparisonObservation{
			Path: path, SchemaVersion: retained.SchemaVersion, Target: comparisonTargetFromCapture(retained.Target),
			ObservedAt: retained.ObservedAt, SurfaceSHA256: retained.SurfaceSHA256,
			ColumnCount: len(retained.Columns), ProducerSourceSHA256: retained.ProducerSourceSHA256,
			InvocationSHA256: retained.InvocationSHA256,
		}
		classifyV0Alpha2Comparison(report, current, retained, targetChanged)
		return report, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	legacy, err := spannersys.LoadLegacySurfaceBaseline(current.Target.Kind)
	if err != nil {
		return nil, err
	}
	report.Retained = comparisonObservation{
		Path:          filepath.ToSlash(filepath.Join("survey", "spannersys", "evidence", legacy.Target+legacyPathSuffix(legacy))),
		SchemaVersion: legacy.SchemaVersion,
		Target:        comparisonTargetFromLegacy(legacy),
		ObservedAt:    legacy.ObservedAt,
		SurfaceSHA256: legacy.SurfaceSHA256, ColumnCount: legacy.ColumnCount,
	}
	if targetChanged || current.Target.Kind == "omni" && current.Target.Image.Tag != legacy.RuntimeTag {
		report.Status, report.MaterialChange = "target_identity_changed", true
	} else if current.SurfaceSHA256 != legacy.SurfaceSHA256 {
		report.Status, report.MaterialChange = "surface_changed", true
	} else {
		report.Status = "unchanged_legacy_baseline"
	}
	return report, nil
}

func classifyV0Alpha2Comparison(report *comparisonReport, current, retained *spannersys.SurfaceCapture, targetChanged bool) {
	switch {
	case targetChanged:
		report.Status, report.MaterialChange = "target_identity_changed", true
	case retained.SurfaceSHA256 != current.SurfaceSHA256:
		report.Status, report.MaterialChange = "surface_changed", true
		report.SurfaceDelta = computeSurfaceDelta(retained.Columns, current.Columns)
	case retained.ProducerSourceSHA256 != current.ProducerSourceSHA256 || retained.InvocationSHA256 != current.InvocationSHA256:
		report.Status = "producer_changed_only"
	default:
		report.Status = "unchanged"
	}
}

func computeSurfaceDelta(retained, current []spannersys.SurfaceCaptureColumn) *surfaceDelta {
	delta := &surfaceDelta{}
	retainedColumns := indexSurfaceColumns(retained)
	currentColumns := indexSurfaceColumns(current)
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
	return delta
}

func surfaceColumnRefFrom(column spannersys.SurfaceCaptureColumn) surfaceColumnRef {
	return surfaceColumnRef{
		TableName:       column.TableName,
		ColumnName:      column.ColumnName,
		SpannerType:     column.SpannerType,
		OrdinalPosition: column.OrdinalPosition,
	}
}

func indexSurfaceColumns(columns []spannersys.SurfaceCaptureColumn) map[columnIdentity]spannersys.SurfaceCaptureColumn {
	index := make(map[columnIdentity]spannersys.SurfaceCaptureColumn, len(columns))
	for _, column := range columns {
		index[columnIdentity{table: column.TableName, column: column.ColumnName}] = column
	}
	return index
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

func legacyPathSuffix(baseline spannersys.LegacySurfaceBaseline) string {
	if baseline.Target == "omni" {
		return "-" + baseline.RuntimeTag + ".json"
	}
	return ".json"
}

func findV0Alpha2Baseline(repoRoot string, target capturemeta.Target) (string, *spannersys.SurfaceCapture, error) {
	root := filepath.Join(repoRoot, "survey", "spannersys", "evidence", target.Kind)
	type candidate struct {
		path     string
		document *spannersys.SurfaceCapture
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
		document, err := spannersys.DecodeSurfaceCapture(data)
		if err != nil {
			return err
		}
		if document.Target.Kind != target.Kind {
			return nil
		}
		if target.Kind == "omni" && (document.Target.Image == nil || target.Image == nil || *document.Target.Image != *target.Image) {
			return nil
		}
		observed, err := time.Parse(time.RFC3339Nano, document.ObservedAt)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		candidates = append(candidates, candidate{path: filepath.ToSlash(relative), document: document, observed: observed})
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil, fs.ErrNotExist
	}
	if err != nil {
		return "", nil, err
	}
	if len(candidates) == 0 {
		return "", nil, fs.ErrNotExist
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].observed.After(candidates[j].observed) })
	return candidates[0].path, candidates[0].document, nil
}

func resolveRepoRoot(explicit string) (string, error) {
	if explicit != "" {
		root, err := filepath.Abs(explicit)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(filepath.Join(root, "runtime_targets.json")); err != nil {
			return "", fmt.Errorf("invalid repository root %q: %w", root, err)
		}
		return root, nil
	}
	return runtimepins.FindRepositoryRoot(".")
}

func writeCapture(path string, data []byte, document *spannersys.SurfaceCapture) (bool, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".spanner-sys-capture-*.tmp")
	if err != nil {
		return false, err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return false, err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return false, err
	}
	if err := temporary.Close(); err != nil {
		return false, err
	}
	if err := os.Link(temporaryPath, path); err == nil {
		return true, nil
	} else if !errors.Is(err, os.ErrExist) {
		return false, err
	}
	existing, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if bytes.Equal(existing, data) {
		return false, nil
	}
	previous, err := spannersys.DecodeSurfaceCapture(existing)
	if err != nil {
		return false, err
	}
	previousPath, previousErr := spannersys.ExpectedSurfaceCapturePath(previous)
	currentPath, currentErr := spannersys.ExpectedSurfaceCapturePath(document)
	if previousErr == nil && currentErr == nil && previousPath == currentPath {
		if previous.ProducerSourceSHA256 != document.ProducerSourceSHA256 || previous.InvocationSHA256 != document.InvocationSHA256 {
			return false, errors.New("canonical capture path already exists with a different producer identity")
		}
		return false, nil
	}
	return false, errors.New("canonical capture path already exists with conflicting identity or surface")
}

func writeDiagnostic(writer io.Writer, format string, arguments ...any) {
	_, _ = fmt.Fprintf(writer, format, arguments...)
}
