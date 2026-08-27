package main

import (
	"encoding/json"
	"errors"
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

const comparisonSchemaVersion = "v0alpha1"

type comparisonReport struct {
	SchemaVersion  string                   `json:"schema_version"`
	Status         string                   `json:"status"`
	MaterialChange bool                     `json:"material_change"`
	Current        comparisonObservation    `json:"current"`
	Retained       comparisonObservation    `json:"retained"`
	ProducerChange *comparisonProducerDelta `json:"producer_change,omitempty"`
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
	if current == nil {
		return nil, errors.New("current capture is nil")
	}
	retainedPath, retained, targetChanged, err := retainedBaseline(repoRoot, current)
	if err != nil {
		return nil, err
	}
	report := &comparisonReport{
		SchemaVersion: comparisonSchemaVersion,
		Current:       comparisonObservationFromCapture("", current),
		Retained:      comparisonObservationFromCapture(retainedPath, retained),
	}
	if targetChanged {
		report.Status = "target_identity_changed"
		report.MaterialChange = true
		return report, nil
	}
	if current.SurfaceSHA256 != retained.SurfaceSHA256 {
		report.Status = "surface_changed"
		report.MaterialChange = true
		return report, nil
	}
	sourceChanged := current.ProducerSourceSHA256 != retained.ProducerSourceSHA256
	invocationChanged := current.InvocationSHA256 != retained.InvocationSHA256
	if sourceChanged || invocationChanged {
		report.Status = "producer_changed_only"
		report.ProducerChange = &comparisonProducerDelta{
			SourceChanged:     sourceChanged,
			InvocationChanged: invocationChanged,
		}
		return report, nil
	}
	report.Status = "unchanged"
	return report, nil
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
