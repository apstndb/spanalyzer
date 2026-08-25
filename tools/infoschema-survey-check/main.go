// Command infoschema-survey-check validates the committed INFORMATION_SCHEMA
// manifest and optionally compares it with a pinned spanner-emulator-survey
// checkout.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	defaultManifestPath = "information_schema_manifest.json"
	exporterSource      = `package main

import (
	"encoding/json"
	"os"

	"github.com/apstndb/spanner-emulator-survey/infoschem"
)

func main() {
	if err := json.NewEncoder(os.Stdout).Encode(infoschem.AllTableMetas()); err != nil {
		panic(err)
	}
}
`
)

type manifest struct {
	SchemaVersion string        `json:"schema_version"`
	Source        source        `json:"source"`
	Documentation documentation `json:"documentation"`
	ContentSHA256 string        `json:"content_sha256"`
	Tables        []table       `json:"tables"`
}

type source struct {
	Repository   string `json:"repository"`
	Commit       string `json:"commit"`
	Path         string `json:"path"`
	ExportSHA256 string `json:"export_sha256"`
	ExporterNote string `json:"exporter_note"`
}

type documentation struct {
	URL         string `json:"url"`
	LastUpdated string `json:"last_updated"`
}

type table struct {
	Name    string   `json:"name"`
	Columns []column `json:"columns"`
}

type column struct {
	Name           string `json:"name"`
	Ordinal        int    `json:"ordinal,omitempty"`
	RawType        string `json:"raw_type"`
	EvidenceStatus string `json:"evidence_status"`
	Project        bool   `json:"project"`
	ProjectedType  string `json:"projected_type,omitempty"`
}

type surveyTable struct {
	Schema  string         `json:"Schema"`
	Name    string         `json:"Name"`
	Columns []surveyColumn `json:"Columns"`
}

type surveyColumn struct {
	Name            string `json:"Name"`
	SpannerType     string `json:"SpannerType"`
	OrdinalPosition int    `json:"OrdinalPosition"`
	Rolling         bool   `json:"Rolling"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("infoschema-survey-check", flag.ContinueOnError)
	repoRoot := fs.String("repo-root", "", "spanalyzer repository root (auto-detected when omitted)")
	manifestPath := fs.String("manifest", defaultManifestPath, "manifest path relative to the spanalyzer repository root")
	surveyRoot := fs.String("survey-root", "", "optional spanner-emulator-survey checkout to compare")
	if err := fs.Parse(args); err != nil {
		return err
	}

	root, err := resolveRepoRoot(*repoRoot)
	if err != nil {
		return err
	}
	doc, err := readManifest(filepath.Join(root, filepath.FromSlash(*manifestPath)))
	if err != nil {
		return err
	}
	if err := validateManifest(doc); err != nil {
		return err
	}
	if *surveyRoot == "" {
		return nil
	}
	return compareSurveyCheckout(doc, *surveyRoot)
}

func resolveRepoRoot(explicit string) (string, error) {
	if explicit != "" {
		root, err := filepath.Abs(explicit)
		if err != nil {
			return "", fmt.Errorf("resolve repository root: %w", err)
		}
		return root, nil
	}
	start, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	return findRepoRoot(start)
}

func findRepoRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	for {
		if regularFile(filepath.Join(dir, defaultManifestPath)) && regularFile(filepath.Join(dir, "tools", "infoschema-survey-check", "main.go")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("spanalyzer repository root not found from %q; pass --repo-root", start)
		}
		dir = parent
	}
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func readManifest(path string) (*manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read INFORMATION_SCHEMA manifest: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var doc manifest
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode INFORMATION_SCHEMA manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("decode INFORMATION_SCHEMA manifest: multiple JSON values")
		}
		return nil, fmt.Errorf("decode INFORMATION_SCHEMA manifest trailer: %w", err)
	}
	return &doc, nil
}

func validateManifest(doc *manifest) error {
	if doc.SchemaVersion != "v0alpha1" {
		return fmt.Errorf("manifest schema_version = %q, want v0alpha1", doc.SchemaVersion)
	}
	if doc.Source.Commit == "" || doc.Source.ExportSHA256 == "" {
		return errors.New("manifest source provenance is incomplete")
	}
	want, err := hashJSON(doc.Tables)
	if err != nil {
		return fmt.Errorf("hash manifest tables: %w", err)
	}
	if doc.ContentSHA256 != want {
		return fmt.Errorf("manifest content_sha256 = %q, want %q", doc.ContentSHA256, want)
	}
	return nil
}

func compareSurveyCheckout(doc *manifest, surveyRoot string) error {
	root, err := filepath.Abs(surveyRoot)
	if err != nil {
		return fmt.Errorf("resolve survey root: %w", err)
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		if err == nil {
			err = errors.New("not a directory")
		}
		return fmt.Errorf("invalid --survey-root %q: %w", root, err)
	}

	commitCommand := exec.Command("git", "rev-parse", "HEAD")
	commitCommand.Dir = root
	commitOutput, err := commitCommand.Output()
	if err != nil {
		return fmt.Errorf("resolve survey commit in %q: %w", root, err)
	}
	commit := strings.TrimSpace(string(commitOutput))
	if commit != doc.Source.Commit {
		return fmt.Errorf("survey checkout commit = %q, manifest pins %q", commit, doc.Source.Commit)
	}

	temporary, err := os.CreateTemp("", "spanalyzer-infoschema-survey-export-*.go")
	if err != nil {
		return fmt.Errorf("create temporary survey exporter: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		// Cleanup is best-effort after the exporter has already been closed and
		// its output has either been consumed or reported.
		_ = os.Remove(temporaryPath)
	}()
	if _, writeErr := temporary.WriteString(exporterSource); writeErr != nil {
		if closeErr := temporary.Close(); closeErr != nil {
			return fmt.Errorf("write and close temporary survey exporter: %w", errors.Join(writeErr, closeErr))
		}
		return fmt.Errorf("write temporary survey exporter: %w", writeErr)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary survey exporter: %w", err)
	}

	exportCommand := exec.Command("go", "run", temporaryPath)
	exportCommand.Dir = root
	exportOutput, err := exportCommand.Output()
	if err != nil {
		return fmt.Errorf("run survey exporter in %q: %w", root, err)
	}
	survey, err := decodeSurveyExport(exportOutput)
	if err != nil {
		return err
	}
	return compareManifestToSurvey(doc, survey)
}

func decodeSurveyExport(data []byte) ([]surveyTable, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var survey []surveyTable
	if err := decoder.Decode(&survey); err != nil {
		return nil, fmt.Errorf("decode survey export: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("decode survey export: multiple JSON values")
		}
		return nil, fmt.Errorf("decode survey export trailer: %w", err)
	}
	return survey, nil
}

func compareManifestToSurvey(doc *manifest, survey []surveyTable) error {
	exportHash, err := hashJSON(survey)
	if err != nil {
		return fmt.Errorf("hash survey export: %w", err)
	}
	if exportHash != doc.Source.ExportSHA256 {
		return fmt.Errorf("survey export SHA-256 = %q, manifest pins %q", exportHash, doc.Source.ExportSHA256)
	}
	if len(doc.Tables) != len(survey) {
		return fmt.Errorf("manifest has %d tables, survey exports %d", len(doc.Tables), len(survey))
	}
	for tableIndex, surveyTable := range survey {
		manifestTable := doc.Tables[tableIndex]
		if surveyTable.Schema != "INFORMATION_SCHEMA" {
			return fmt.Errorf("survey table %q schema = %q", surveyTable.Name, surveyTable.Schema)
		}
		if manifestTable.Name != surveyTable.Name {
			return fmt.Errorf("table %d manifest name = %q, survey name = %q", tableIndex, manifestTable.Name, surveyTable.Name)
		}

		surveyByName := make(map[string]surveyColumn, len(surveyTable.Columns))
		for _, surveyColumn := range surveyTable.Columns {
			surveyByName[surveyColumn.Name] = surveyColumn
		}
		manifestLive := make([]column, 0, len(manifestTable.Columns))
		for _, manifestColumn := range manifestTable.Columns {
			if manifestColumn.EvidenceStatus == "docs_only_absent" {
				if _, exists := surveyByName[manifestColumn.Name]; exists {
					return fmt.Errorf("manifest marks live survey column %s.%s as docs_only_absent", manifestTable.Name, manifestColumn.Name)
				}
				continue
			}
			manifestLive = append(manifestLive, manifestColumn)
		}
		if len(manifestLive) != len(surveyTable.Columns) {
			return fmt.Errorf("manifest table %s has %d live columns, survey exports %d", manifestTable.Name, len(manifestLive), len(surveyTable.Columns))
		}
		for columnIndex, surveyColumn := range surveyTable.Columns {
			manifestColumn := manifestLive[columnIndex]
			wantStatus := "live_observed"
			if surveyColumn.Rolling {
				wantStatus = "rolling"
			}
			if manifestColumn.Name != surveyColumn.Name || manifestColumn.Ordinal != surveyColumn.OrdinalPosition || manifestColumn.RawType != surveyColumn.SpannerType || manifestColumn.EvidenceStatus != wantStatus || !manifestColumn.Project {
				return fmt.Errorf("manifest column %s[%d] = {%s %d %s %s project=%t}, survey = {%s %d %s %s project=true}", manifestTable.Name, columnIndex, manifestColumn.Name, manifestColumn.Ordinal, manifestColumn.RawType, manifestColumn.EvidenceStatus, manifestColumn.Project, surveyColumn.Name, surveyColumn.OrdinalPosition, surveyColumn.SpannerType, wantStatus)
			}
		}
	}
	return nil
}

func hashJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
