package spanalyzer

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

const (
	informationSchemaName                  = "INFORMATION_SCHEMA"
	informationSchemaManifestSchemaVersion = "v0alpha2"
)

type informationSchemaColumn struct {
	name string
	typ  *TypeSpec
}

type informationSchemaTable struct {
	name    string
	columns []informationSchemaColumn
}

type informationSchemaManifest struct {
	SchemaVersion string                                 `json:"schema_version"`
	Source        informationSchemaManifestSource        `json:"source"`
	Documentation informationSchemaManifestDocumentation `json:"documentation"`
	ContentSHA256 string                                 `json:"content_sha256"`
	Tables        []informationSchemaManifestTable       `json:"tables"`
}

type informationSchemaManifestSource struct {
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

type informationSchemaManifestDocumentation struct {
	URL         string `json:"url"`
	LastUpdated string `json:"last_updated"`
}

type informationSchemaManifestTable struct {
	Name    string                            `json:"name"`
	Columns []informationSchemaManifestColumn `json:"columns"`
}

type informationSchemaManifestColumn struct {
	Name           string `json:"name"`
	Ordinal        int    `json:"ordinal,omitempty"`
	RawType        string `json:"raw_type"`
	EvidenceStatus string `json:"evidence_status"`
	Project        bool   `json:"project"`
	ProjectedType  string `json:"projected_type,omitempty"`
}

//go:embed information_schema_manifest.json
var embeddedInformationSchemaManifest []byte

var (
	informationSchemaOnce   sync.Once
	informationSchemaTables []informationSchemaTable
	informationSchemaErr    error
)

func (c *Catalog) addInformationSchemaTables() error {
	tables, err := loadInformationSchemaTables()
	if err != nil {
		return err
	}
	for _, def := range tables {
		table := &Table{
			Name:    ObjectName{Parts: []string{informationSchemaName, def.name}},
			Columns: make([]*Column, 0, len(def.columns)),
		}
		for _, column := range def.columns {
			table.Columns = append(table.Columns, &Column{
				Name: column.name,
				Type: column.typ,
			})
		}
		c.Tables[table.Name.String()] = table
	}
	return nil
}

func loadInformationSchemaTables() ([]informationSchemaTable, error) {
	informationSchemaOnce.Do(func() {
		_, informationSchemaTables, informationSchemaErr = parseInformationSchemaManifest(embeddedInformationSchemaManifest)
	})
	if informationSchemaErr != nil {
		return nil, informationSchemaErr
	}
	return informationSchemaTables, nil
}

func parseInformationSchemaManifest(data []byte) (*informationSchemaManifest, []informationSchemaTable, error) {
	if err := rejectDuplicateInformationSchemaJSONKeys(data); err != nil {
		return nil, nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest informationSchemaManifest
	if err := decoder.Decode(&manifest); err != nil {
		return nil, nil, fmt.Errorf("decode embedded INFORMATION_SCHEMA manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, nil, errors.New("decode embedded INFORMATION_SCHEMA manifest: multiple JSON values")
		}
		return nil, nil, fmt.Errorf("decode embedded INFORMATION_SCHEMA manifest trailer: %w", err)
	}

	if manifest.SchemaVersion != informationSchemaManifestSchemaVersion {
		return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest schema_version = %q, want %q", manifest.SchemaVersion, informationSchemaManifestSchemaVersion)
	}
	if manifest.Source.Repository != "github.com/apstndb/spanalyzer" || manifest.Source.RegistryPath != "survey/infoschem" {
		return nil, nil, errors.New("INFORMATION_SCHEMA manifest source repository or registry path is invalid")
	}
	if manifest.Source.ProjectionSourcePath != "information_schema_projection_source.json" {
		return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest projection source path = %q", manifest.Source.ProjectionSourcePath)
	}
	for label, value := range map[string]string{
		"source.registry_export_sha256":           manifest.Source.RegistryExportSHA256,
		"source.selected_observation_file_sha256": manifest.Source.SelectedObservationFileSHA256,
		"source.surface_sha256":                   manifest.Source.SurfaceSHA256,
		"source.producer_source_sha256":           manifest.Source.ProducerSourceSHA256,
		"source.invocation_sha256":                manifest.Source.InvocationSHA256,
		"source.projection_source_sha256":         manifest.Source.ProjectionSourceSHA256,
	} {
		if err := validateSHA256(label, value); err != nil {
			return nil, nil, err
		}
	}
	observedAt, err := time.Parse(time.RFC3339Nano, manifest.Source.ObservedAt)
	if err != nil || observedAt.Location() != time.UTC || observedAt.Format(time.RFC3339Nano) != manifest.Source.ObservedAt {
		return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest source observed_at = %q, want canonical RFC3339 UTC", manifest.Source.ObservedAt)
	}
	expectedObservationPath := fmt.Sprintf(
		"survey/infoschem/evidence/managed/%s-%s.json",
		observedAt.UTC().Format("20060102T150405Z"),
		manifest.Source.SurfaceSHA256[:12],
	)
	if manifest.Source.SelectedObservationPath != expectedObservationPath {
		return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest selected observation path = %q, want %q from observed_at and surface_sha256", manifest.Source.SelectedObservationPath, expectedObservationPath)
	}
	if manifest.Documentation.URL == "" || manifest.Documentation.LastUpdated == "" {
		return nil, nil, errors.New("INFORMATION_SCHEMA manifest documentation provenance is incomplete")
	}
	documentationURL, err := url.ParseRequestURI(manifest.Documentation.URL)
	if err != nil || documentationURL.Scheme == "" || documentationURL.Host == "" {
		return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest documentation.url = %q, want an absolute URI", manifest.Documentation.URL)
	}
	if _, err := time.Parse(time.DateOnly, manifest.Documentation.LastUpdated); err != nil {
		return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest documentation.last_updated = %q, want YYYY-MM-DD", manifest.Documentation.LastUpdated)
	}
	if len(manifest.Tables) == 0 {
		return nil, nil, errors.New("INFORMATION_SCHEMA manifest has no tables")
	}

	wantHash, err := hashInformationSchemaManifestTables(manifest.Tables)
	if err != nil {
		return nil, nil, err
	}
	if err := validateSHA256("content_sha256", manifest.ContentSHA256); err != nil {
		return nil, nil, err
	}
	if manifest.ContentSHA256 != wantHash {
		return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest content_sha256 = %q, want %q", manifest.ContentSHA256, wantHash)
	}

	tableNames := make(map[string]struct{}, len(manifest.Tables))
	tables := make([]informationSchemaTable, 0, len(manifest.Tables))
	for _, manifestTable := range manifest.Tables {
		if manifestTable.Name == "" {
			return nil, nil, errors.New("INFORMATION_SCHEMA manifest contains a table with an empty name")
		}
		if _, exists := tableNames[manifestTable.Name]; exists {
			return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest contains duplicate table %q", manifestTable.Name)
		}
		tableNames[manifestTable.Name] = struct{}{}

		columnNames := make(map[string]struct{}, len(manifestTable.Columns))
		projectedOrdinals := make(map[int]string, len(manifestTable.Columns))
		lastProjectedOrdinal := 0
		table := informationSchemaTable{name: manifestTable.Name}
		for _, manifestColumn := range manifestTable.Columns {
			path := manifestTable.Name + "." + manifestColumn.Name
			if manifestColumn.Name == "" {
				return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest table %q contains a column with an empty name", manifestTable.Name)
			}
			if _, exists := columnNames[manifestColumn.Name]; exists {
				return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest contains duplicate column %q", path)
			}
			columnNames[manifestColumn.Name] = struct{}{}
			if manifestColumn.RawType == "" {
				return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest column %q has an empty raw_type", path)
			}

			switch manifestColumn.EvidenceStatus {
			case "live_observed", "rolling":
				if !manifestColumn.Project {
					return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest column %q is %s but project is false", path, manifestColumn.EvidenceStatus)
				}
				if manifestColumn.Ordinal <= 0 {
					return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest projected column %q has ordinal %d", path, manifestColumn.Ordinal)
				}
			case "docs_only_absent":
				if manifestColumn.Project {
					return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest docs-only column %q must not be projected", path)
				}
				if manifestColumn.Ordinal != 0 {
					return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest docs-only column %q must not declare an ordinal", path)
				}
				if manifestColumn.ProjectedType != "" {
					return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest docs-only column %q must not declare projected_type", path)
				}
				continue
			default:
				return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest column %q has unknown evidence_status %q", path, manifestColumn.EvidenceStatus)
			}

			if existing, duplicate := projectedOrdinals[manifestColumn.Ordinal]; duplicate {
				return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest columns %q and %q share ordinal %d", existing, path, manifestColumn.Ordinal)
			}
			if manifestColumn.Ordinal <= lastProjectedOrdinal {
				return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest projected column %q is out of ordinal order", path)
			}
			projectedOrdinals[manifestColumn.Ordinal] = path
			lastProjectedOrdinal = manifestColumn.Ordinal

			typ, err := informationSchemaColumnType(manifestColumn)
			if err != nil {
				return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest column %q: %w", path, err)
			}
			table.columns = append(table.columns, informationSchemaColumn{name: manifestColumn.Name, typ: typ})
		}
		if len(table.columns) == 0 {
			return nil, nil, fmt.Errorf("INFORMATION_SCHEMA manifest table %q has no projected columns", manifestTable.Name)
		}
		tables = append(tables, table)
	}
	return &manifest, tables, nil
}

func informationSchemaColumnType(column informationSchemaManifestColumn) (*TypeSpec, error) {
	typeSQL := column.RawType
	if strings.HasPrefix(column.RawType, "PROTO<") {
		if column.ProjectedType == "" {
			return nil, fmt.Errorf("raw type %q requires an explicit projected_type override", column.RawType)
		}
		typeSQL = column.ProjectedType
	} else if column.ProjectedType != "" {
		return nil, fmt.Errorf("raw type %q does not permit projected_type override %q", column.RawType, column.ProjectedType)
	}

	typ, err := ParseTypeSpec("information_schema_manifest.json", typeSQL)
	if err != nil {
		return nil, fmt.Errorf("parse projected type %q: %w", typeSQL, err)
	}
	if typ.Code == spannerpb.TypeCode_PROTO || typ.Code == spannerpb.TypeCode_ENUM {
		return nil, fmt.Errorf("projected type %q requires descriptor policy not provided by the manifest", typeSQL)
	}
	return typ, nil
}

func hashInformationSchemaManifestTables(tables []informationSchemaManifestTable) (string, error) {
	data, err := json.Marshal(tables)
	if err != nil {
		return "", fmt.Errorf("marshal INFORMATION_SCHEMA manifest tables for hashing: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func validateSHA256(name, value string) error {
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != sha256.Size {
		return fmt.Errorf("INFORMATION_SCHEMA manifest %s = %q, want a lowercase SHA-256 digest", name, value)
	}
	if value != strings.ToLower(value) {
		return fmt.Errorf("INFORMATION_SCHEMA manifest %s = %q, want lowercase hexadecimal", name, value)
	}
	return nil
}

func rejectDuplicateInformationSchemaJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := scanInformationSchemaJSONValue(decoder, "$"); err != nil {
		return fmt.Errorf("validate embedded INFORMATION_SCHEMA manifest keys: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("validate embedded INFORMATION_SCHEMA manifest keys: multiple JSON values")
		}
		return fmt.Errorf("validate embedded INFORMATION_SCHEMA manifest trailer: %w", err)
	}
	return nil
}

func scanInformationSchemaJSONValue(decoder *json.Decoder, path string) error {
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
			if err := scanInformationSchemaJSONValue(decoder, path+"."+key); err != nil {
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
			if err := scanInformationSchemaJSONValue(decoder, fmt.Sprintf("%s[%d]", path, index)); err != nil {
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
