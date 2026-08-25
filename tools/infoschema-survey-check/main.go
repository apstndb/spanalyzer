// Command infoschema-survey-check regenerates and validates the generated
// INFORMATION_SCHEMA analyzer projection from one explicitly selected managed
// observation and the retained survey registry.
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
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var managedCapturePathPattern = regexp.MustCompile(`^survey/infoschem/evidence/managed/[0-9]{8}T[0-9]{6}Z-[0-9a-f]{12}\.json$`)

const (
	defaultManifestPath        = "information_schema_manifest.json"
	defaultProjectionSource    = "information_schema_projection_source.json"
	manifestSchemaVersion      = "v0alpha2"
	projectionSchemaVersion    = "v0alpha1"
	projectionMode             = "managed_live_primary"
	rollingAdvertisementPolicy = "runtime_filtered"
	exporterSource             = `package main

import (
	"encoding/json"
	"os"

	"github.com/apstndb/spanalyzer/survey/infoschem"
)

type output struct {
	Registry            []*infoschem.TableMeta      ` + "`json:\"registry\"`" + `
	Capture             *infoschem.CaptureDocument ` + "`json:\"capture\"`" + `
	ExpectedCapturePath string                      ` + "`json:\"expected_capture_path\"`" + `
}

func main() {
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	capture, err := infoschem.DecodeCapture(data)
	if err != nil {
		panic(err)
	}
	expected, err := infoschem.ExpectedCapturePath(capture)
	if err != nil {
		panic(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(output{
		Registry: infoschem.AllTableMetas(),
		Capture: capture,
		ExpectedCapturePath: expected,
	}); err != nil {
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
	Repository                    string `json:"repository"`
	RegistryPath                  string `json:"registry_path"`
	RegistryExportSHA256          string `json:"registry_export_sha256"`
	SelectedObservationPath       string `json:"selected_observation_path"`
	SelectedObservationFileSHA256 string `json:"selected_observation_file_sha256"`
	ObservedAt                    string `json:"observed_at"`
	SurfaceSHA256                 string `json:"surface_sha256"`
	ProducerSourceSHA256          string `json:"producer_source_sha256"`
	InvocationSHA256              string `json:"invocation_sha256"`
	ProjectionSourcePath          string `json:"projection_source_path"`
	ProjectionSourceSHA256        string `json:"projection_source_sha256"`
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

type projectionSource struct {
	SchemaVersion              string                    `json:"schema_version"`
	Mode                       string                    `json:"mode"`
	SelectedObservation        selectedObservation       `json:"selected_observation"`
	RollingAdvertisementPolicy string                    `json:"rolling_advertisement_policy"`
	Documentation              documentation             `json:"documentation"`
	DocumentationOnlyAbsent    []documentationOnlyColumn `json:"documentation_only_absent"`
	ProjectionOverrides        []projectionOverride      `json:"projection_overrides"`
}

type selectedObservation struct {
	Path       string `json:"path"`
	FileSHA256 string `json:"file_sha256"`
}

type documentationOnlyColumn struct {
	TableName  string `json:"table_name"`
	ColumnName string `json:"column_name"`
	RawType    string `json:"raw_type"`
}

type projectionOverride struct {
	TableName     string `json:"table_name"`
	ColumnName    string `json:"column_name"`
	ProjectedType string `json:"projected_type"`
}

type surveyExport struct {
	Registry            []surveyTable   `json:"registry"`
	Capture             captureDocument `json:"capture"`
	ExpectedCapturePath string          `json:"expected_capture_path"`
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

type captureDocument struct {
	SchemaVersion        string                `json:"schema_version"`
	Catalog              string                `json:"catalog"`
	Dialect              string                `json:"dialect"`
	Target               captureTarget         `json:"target"`
	ObservedAt           string                `json:"observed_at"`
	ProducerSourceSHA256 string                `json:"producer_source_sha256"`
	InvocationSHA256     string                `json:"invocation_sha256"`
	Query                string                `json:"query"`
	SurfaceSHA256        string                `json:"surface_sha256"`
	Columns              []captureColumn       `json:"columns"`
	RollingQueryability  []rollingQueryability `json:"rolling_queryability"`
}

type captureTarget struct {
	Kind             string `json:"kind"`
	ObservationScope string `json:"observation_scope,omitempty"`
}

type captureColumn struct {
	TableName       string `json:"table_name"`
	ColumnName      string `json:"column_name"`
	SpannerType     string `json:"spanner_type"`
	OrdinalPosition int    `json:"ordinal_position"`
}

type rollingQueryability struct {
	TableName  string `json:"table_name"`
	ColumnName string `json:"column_name"`
	Status     string `json:"status"`
	StatusCode string `json:"status_code,omitempty"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("infoschema-survey-check", flag.ContinueOnError)
	repoRoot := flags.String("repo-root", "", "spanalyzer repository root (auto-detected when omitted)")
	manifestPath := flags.String("manifest", defaultManifestPath, "manifest path relative to the spanalyzer repository root")
	write := flags.Bool("write", false, "write the deterministic generated manifest instead of checking it")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}

	root, err := resolveRepoRoot(*repoRoot)
	if err != nil {
		return err
	}
	projectionFile := filepath.Join(root, defaultProjectionSource)
	policyData, err := os.ReadFile(projectionFile)
	if err != nil {
		return fmt.Errorf("read INFORMATION_SCHEMA projection source: %w", err)
	}
	policy, err := decodeProjectionSource(policyData)
	if err != nil {
		return err
	}
	if err := validateProjectionSource(policy); err != nil {
		return err
	}

	capturePath, err := safeRepositoryPath(root, policy.SelectedObservation.Path)
	if err != nil {
		return fmt.Errorf("resolve selected INFORMATION_SCHEMA observation: %w", err)
	}
	captureData, err := os.ReadFile(capturePath)
	if err != nil {
		return fmt.Errorf("read selected INFORMATION_SCHEMA observation: %w", err)
	}
	if got := hashBytes(captureData); got != policy.SelectedObservation.FileSHA256 {
		return fmt.Errorf("selected observation SHA-256 = %q, projection source pins %q", got, policy.SelectedObservation.FileSHA256)
	}

	producerRoot := filepath.Join(root, "survey")
	exported, err := exportSurveyModule(producerRoot, capturePath)
	if err != nil {
		return err
	}
	if err := validateExportedCapture(policy, exported, captureData); err != nil {
		return err
	}
	document, err := buildManifest(policy, hashBytes(policyData), exported)
	if err != nil {
		return err
	}
	generated, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode INFORMATION_SCHEMA manifest: %w", err)
	}
	generated = append(generated, '\n')
	manifestFile := filepath.Join(root, filepath.FromSlash(*manifestPath))
	if *write {
		if err := os.WriteFile(manifestFile, generated, 0o644); err != nil {
			return fmt.Errorf("write INFORMATION_SCHEMA manifest: %w", err)
		}
		return nil
	}
	committed, err := os.ReadFile(manifestFile)
	if err != nil {
		return fmt.Errorf("read INFORMATION_SCHEMA manifest: %w", err)
	}
	var decoded manifest
	if err := decodeStrict(committed, &decoded, "INFORMATION_SCHEMA manifest"); err != nil {
		return err
	}
	if err := validateManifest(&decoded); err != nil {
		return err
	}
	if !bytes.Equal(committed, generated) {
		return errors.New("INFORMATION_SCHEMA manifest is stale; run infoschema-survey-check --write")
	}
	return nil
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
	directory, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	for {
		if regularFile(filepath.Join(directory, defaultManifestPath)) && regularFile(filepath.Join(directory, "tools", "infoschema-survey-check", "main.go")) {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("spanalyzer repository root not found from %q; pass --repo-root", start)
		}
		directory = parent
	}
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func decodeProjectionSource(data []byte) (*projectionSource, error) {
	var policy projectionSource
	if err := decodeStrict(data, &policy, "INFORMATION_SCHEMA projection source"); err != nil {
		return nil, err
	}
	return &policy, nil
}

func decodeStrict(data []byte, destination any, label string) error {
	if err := rejectDuplicateKeys(data); err != nil {
		return fmt.Errorf("decode %s: %w", label, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode %s: %w", label, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("decode %s: multiple JSON values", label)
		}
		return fmt.Errorf("decode %s trailer: %w", label, err)
	}
	return nil
}

func rejectDuplicateKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := scanJSONValue(decoder, "$"); err != nil {
		return fmt.Errorf("validate JSON keys: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("validate JSON keys: multiple JSON values")
		}
		return fmt.Errorf("validate JSON trailer: %w", err)
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
		for index := 0; decoder.More(); index++ {
			if err := scanJSONValue(decoder, fmt.Sprintf("%s[%d]", path, index)); err != nil {
				return err
			}
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

func validateProjectionSource(policy *projectionSource) error {
	if policy.SchemaVersion != projectionSchemaVersion || policy.Mode != projectionMode {
		return fmt.Errorf("projection source identity = %q/%q, want %q/%q", policy.SchemaVersion, policy.Mode, projectionSchemaVersion, projectionMode)
	}
	if policy.RollingAdvertisementPolicy != rollingAdvertisementPolicy {
		return fmt.Errorf("projection source rolling_advertisement_policy = %q, want %q", policy.RollingAdvertisementPolicy, rollingAdvertisementPolicy)
	}
	if !managedCapturePathPattern.MatchString(policy.SelectedObservation.Path) {
		return fmt.Errorf("projection source selected observation path %q is not a managed capture", policy.SelectedObservation.Path)
	}
	if err := validateSHA256("projection source selected observation file_sha256", policy.SelectedObservation.FileSHA256); err != nil {
		return err
	}
	if err := validateDocumentation(policy.Documentation); err != nil {
		return err
	}
	seen := make(map[string]string)
	for _, documented := range policy.DocumentationOnlyAbsent {
		key := documented.TableName + "." + documented.ColumnName
		if documented.TableName == "" || documented.ColumnName == "" || documented.RawType == "" {
			return fmt.Errorf("projection source documentation-only column %q is incomplete", key)
		}
		if previous := seen[key]; previous != "" {
			return fmt.Errorf("projection source repeats column %s as %s and documentation_only_absent", key, previous)
		}
		seen[key] = "documentation_only_absent"
	}
	for _, override := range policy.ProjectionOverrides {
		key := override.TableName + "." + override.ColumnName
		if override.TableName == "" || override.ColumnName == "" || override.ProjectedType == "" {
			return fmt.Errorf("projection source override %q is incomplete", key)
		}
		if previous := seen[key]; previous != "" {
			return fmt.Errorf("projection source repeats column %s as %s and projection_override", key, previous)
		}
		seen[key] = "projection_override"
	}
	return nil
}

func safeRepositoryPath(root, slashPath string) (string, error) {
	if filepath.IsAbs(slashPath) || strings.Contains(slashPath, "\\") {
		return "", fmt.Errorf("path %q is not a canonical repository-relative slash path", slashPath)
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(slashPath)))
	if cleaned != slashPath || cleaned == "." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("path %q escapes or is not canonical", slashPath)
	}
	return filepath.Join(root, filepath.FromSlash(slashPath)), nil
}

func exportSurveyModule(surveyRoot, capturePath string) (*surveyExport, error) {
	root, err := filepath.Abs(surveyRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve survey root: %w", err)
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		if err == nil {
			err = errors.New("not a directory")
		}
		return nil, fmt.Errorf("invalid --survey-root %q: %w", root, err)
	}

	temporary, err := os.CreateTemp("", "spanalyzer-infoschema-survey-export-*.go")
	if err != nil {
		return nil, fmt.Errorf("create temporary survey exporter: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if _, writeErr := temporary.WriteString(exporterSource); writeErr != nil {
		_ = temporary.Close()
		return nil, fmt.Errorf("write temporary survey exporter: %w", writeErr)
	}
	if err := temporary.Close(); err != nil {
		return nil, fmt.Errorf("close temporary survey exporter: %w", err)
	}

	command := exec.Command("go", "run", temporaryPath, capturePath)
	command.Dir = root
	command.Env = append(os.Environ(), "GOWORK=off")
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("run survey exporter in %q: %w\n%s", root, err, output)
	}
	var exported surveyExport
	if err := decodeStrict(output, &exported, "survey export"); err != nil {
		return nil, err
	}
	return &exported, nil
}

func validateExportedCapture(policy *projectionSource, exported *surveyExport, captureData []byte) error {
	if exported.Capture.SchemaVersion != "v0alpha1" || exported.Capture.Catalog != "INFORMATION_SCHEMA" || exported.Capture.Dialect != "googlesql" {
		return errors.New("selected observation has an unexpected capture contract identity")
	}
	if exported.Capture.Target.Kind != "managed" || exported.Capture.Target.ObservationScope != "single_database" {
		return errors.New("selected observation is not a single-database managed capture")
	}
	wantRelative := strings.TrimPrefix(policy.SelectedObservation.Path, "survey/infoschem/")
	if wantRelative == policy.SelectedObservation.Path || exported.ExpectedCapturePath != wantRelative {
		return fmt.Errorf("selected observation path = %q, capture identity requires %q", policy.SelectedObservation.Path, filepath.ToSlash(filepath.Join("survey", "infoschem", exported.ExpectedCapturePath)))
	}
	fileHash := hashBytes(captureData)
	if fileHash != policy.SelectedObservation.FileSHA256 {
		return fmt.Errorf("selected observation file SHA-256 = %q, projection source pins %q", fileHash, policy.SelectedObservation.FileSHA256)
	}
	return nil
}

func buildManifest(policy *projectionSource, policyHash string, exported *surveyExport) (*manifest, error) {
	registryHash, err := hashJSON(exported.Registry)
	if err != nil {
		return nil, fmt.Errorf("hash survey registry export: %w", err)
	}
	observed := make(map[string]captureColumn, len(exported.Capture.Columns))
	for _, captured := range exported.Capture.Columns {
		key := captured.TableName + "." + captured.ColumnName
		observed[key] = captured
	}
	known := make(map[string]bool)
	overrides := make(map[string]string, len(policy.ProjectionOverrides))
	for _, override := range policy.ProjectionOverrides {
		overrides[override.TableName+"."+override.ColumnName] = override.ProjectedType
	}
	usedOverrides := make(map[string]bool)

	tables := make([]table, 0, len(exported.Registry))
	tableIndexes := make(map[string]int, len(exported.Registry))
	for _, registeredTable := range exported.Registry {
		if registeredTable.Schema != "INFORMATION_SCHEMA" {
			return nil, fmt.Errorf("survey table %q schema = %q", registeredTable.Name, registeredTable.Schema)
		}
		manifestTable := table{Name: registeredTable.Name, Columns: make([]column, 0, len(registeredTable.Columns))}
		for _, registeredColumn := range registeredTable.Columns {
			key := registeredTable.Name + "." + registeredColumn.Name
			known[key] = true
			captured, exists := observed[key]
			if !exists && !registeredColumn.Rolling {
				return nil, fmt.Errorf("selected managed observation is missing stable registry column %s", key)
			}
			if exists && (captured.SpannerType != registeredColumn.SpannerType || captured.OrdinalPosition != registeredColumn.OrdinalPosition) {
				return nil, fmt.Errorf("selected managed observation column %s = %s ordinal %d, registry = %s ordinal %d", key, captured.SpannerType, captured.OrdinalPosition, registeredColumn.SpannerType, registeredColumn.OrdinalPosition)
			}
			status := "live_observed"
			if registeredColumn.Rolling {
				status = "rolling"
			}
			manifestColumn := column{
				Name:           registeredColumn.Name,
				Ordinal:        registeredColumn.OrdinalPosition,
				RawType:        registeredColumn.SpannerType,
				EvidenceStatus: status,
				Project:        true,
			}
			if projectedType := overrides[key]; projectedType != "" {
				manifestColumn.ProjectedType = projectedType
				usedOverrides[key] = true
			}
			manifestTable.Columns = append(manifestTable.Columns, manifestColumn)
		}
		tableIndexes[manifestTable.Name] = len(tables)
		tables = append(tables, manifestTable)
	}
	for key := range observed {
		if !known[key] {
			return nil, fmt.Errorf("selected managed observation contains unknown registry column %s", key)
		}
	}
	for key := range overrides {
		if !usedOverrides[key] {
			return nil, fmt.Errorf("projection source override %s does not identify a projected registry column", key)
		}
	}
	for _, documented := range policy.DocumentationOnlyAbsent {
		index, ok := tableIndexes[documented.TableName]
		if !ok {
			return nil, fmt.Errorf("documentation-only column %s.%s names an unknown table", documented.TableName, documented.ColumnName)
		}
		key := documented.TableName + "." + documented.ColumnName
		if known[key] || observed[key].ColumnName != "" {
			return nil, fmt.Errorf("documentation-only column %s is present in registry or selected observation", key)
		}
		tables[index].Columns = append(tables[index].Columns, column{
			Name:           documented.ColumnName,
			RawType:        documented.RawType,
			EvidenceStatus: "docs_only_absent",
			Project:        false,
		})
	}
	contentHash, err := hashJSON(tables)
	if err != nil {
		return nil, fmt.Errorf("hash INFORMATION_SCHEMA manifest tables: %w", err)
	}
	return &manifest{
		SchemaVersion: manifestSchemaVersion,
		Source: source{
			Repository:                    "github.com/apstndb/spanalyzer",
			RegistryPath:                  "survey/infoschem",
			RegistryExportSHA256:          registryHash,
			SelectedObservationPath:       policy.SelectedObservation.Path,
			SelectedObservationFileSHA256: policy.SelectedObservation.FileSHA256,
			ObservedAt:                    exported.Capture.ObservedAt,
			SurfaceSHA256:                 exported.Capture.SurfaceSHA256,
			ProducerSourceSHA256:          exported.Capture.ProducerSourceSHA256,
			InvocationSHA256:              exported.Capture.InvocationSHA256,
			ProjectionSourcePath:          defaultProjectionSource,
			ProjectionSourceSHA256:        policyHash,
		},
		Documentation: policy.Documentation,
		ContentSHA256: contentHash,
		Tables:        tables,
	}, nil
}

func validateManifest(document *manifest) error {
	if document.SchemaVersion != manifestSchemaVersion {
		return fmt.Errorf("manifest schema_version = %q, want %q", document.SchemaVersion, manifestSchemaVersion)
	}
	for label, value := range map[string]string{
		"source.registry_export_sha256":           document.Source.RegistryExportSHA256,
		"source.selected_observation_file_sha256": document.Source.SelectedObservationFileSHA256,
		"source.surface_sha256":                   document.Source.SurfaceSHA256,
		"source.producer_source_sha256":           document.Source.ProducerSourceSHA256,
		"source.invocation_sha256":                document.Source.InvocationSHA256,
		"source.projection_source_sha256":         document.Source.ProjectionSourceSHA256,
		"content_sha256":                          document.ContentSHA256,
	} {
		if err := validateSHA256(label, value); err != nil {
			return err
		}
	}
	want, err := hashJSON(document.Tables)
	if err != nil {
		return fmt.Errorf("hash manifest tables: %w", err)
	}
	if document.ContentSHA256 != want {
		return fmt.Errorf("manifest content_sha256 = %q, want %q", document.ContentSHA256, want)
	}
	return validateDocumentation(document.Documentation)
}

func validateDocumentation(value documentation) error {
	parsed, err := url.ParseRequestURI(value.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("documentation URL = %q, want an absolute URI", value.URL)
	}
	if _, err := time.Parse(time.DateOnly, value.LastUpdated); err != nil {
		return fmt.Errorf("documentation last_updated = %q, want YYYY-MM-DD", value.LastUpdated)
	}
	return nil
}

func validateSHA256(label, value string) error {
	if len(value) != 64 {
		return fmt.Errorf("%s = %q, want 64 lowercase hexadecimal characters", label, value)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || hex.EncodeToString(decoded) != value {
		return fmt.Errorf("%s = %q, want 64 lowercase hexadecimal characters", label, value)
	}
	return nil
}

func hashJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return hashBytes(data), nil
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
