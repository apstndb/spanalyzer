package spannersys

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
)

const (
	captureSchemaVersion = "v0alpha1"
	columnDiscoveryQuery = `SELECT TABLE_NAME, COLUMN_NAME, SPANNER_TYPE, ORDINAL_POSITION
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_SCHEMA = 'SPANNER_SYS'
ORDER BY TABLE_NAME, ORDINAL_POSITION`
)

var (
	requiredCaptureTargets = []string{"managed", "omni"}

	//go:embed evidence/*.json
	embeddedCaptureFiles embed.FS
)

type captureDocument struct {
	SchemaVersion  string          `json:"schema_version"`
	Target         string          `json:"target"`
	CapturedAt     string          `json:"captured_at"`
	RuntimeVersion string          `json:"runtime_version,omitempty"`
	Query          string          `json:"query"`
	ContentSHA256  string          `json:"content_sha256"`
	Columns        []captureColumn `json:"columns"`
}

type captureColumn struct {
	TableName       string `json:"table_name"`
	ColumnName      string `json:"column_name"`
	SpannerType     string `json:"spanner_type"`
	OrdinalPosition int    `json:"ordinal_position"`
}

type embeddedCapture struct {
	Target                 string
	ExpectedRuntimeVersion string
	Path                   string
	Data                   []byte
}

func loadEmbeddedCaptures() ([]embeddedCapture, error) {
	files := []struct {
		target                 string
		expectedRuntimeVersion string
		path                   string
	}{
		{target: "managed", path: "evidence/managed.json"},
		{
			target:                 "omni",
			expectedRuntimeVersion: "2026.r2.1-beta",
			path:                   "evidence/omni-2026.r2.1-beta.json",
		},
	}

	captures := make([]embeddedCapture, 0, len(files))
	for _, file := range files {
		data, err := embeddedCaptureFiles.ReadFile(file.path)
		if err != nil {
			return nil, fmt.Errorf("read embedded %s capture %q: %w", file.target, file.path, err)
		}
		captures = append(captures, embeddedCapture{
			Target:                 file.target,
			ExpectedRuntimeVersion: file.expectedRuntimeVersion,
			Path:                   file.path,
			Data:                   data,
		})
	}
	return captures, nil
}

func decodeCapture(data []byte) (*captureDocument, error) {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return nil, err
	}
	if err := rejectDestinationIdentifiers(data); err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var capture captureDocument
	if err := decoder.Decode(&capture); err != nil {
		return nil, fmt.Errorf("decode capture: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("decode capture: multiple JSON values")
		}
		return nil, fmt.Errorf("decode capture trailer: %w", err)
	}
	if err := validateCapture(&capture); err != nil {
		return nil, err
	}
	return &capture, nil
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := scanJSONValue(decoder, "$"); err != nil {
		return fmt.Errorf("validate capture JSON keys: %w", err)
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
		objectKeys := make(map[string]bool)
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object %s contains a non-string key", path)
			}
			if objectKeys[key] {
				return fmt.Errorf("object %s contains duplicate key %q", path, key)
			}
			objectKeys[key] = true
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

func validateCapture(capture *captureDocument) error {
	if capture.SchemaVersion != captureSchemaVersion {
		return fmt.Errorf(
			"capture schema_version = %q, want %q",
			capture.SchemaVersion,
			captureSchemaVersion,
		)
	}
	if !slices.Contains(requiredCaptureTargets, capture.Target) {
		return fmt.Errorf("capture target = %q, want one of %v", capture.Target, requiredCaptureTargets)
	}
	switch capture.Target {
	case "managed":
		if capture.RuntimeVersion != "" {
			return errors.New("managed capture must not set runtime_version")
		}
	case "omni":
		if capture.RuntimeVersion == "" {
			return errors.New("omni capture must set runtime_version")
		}
	}
	if err := validateDate("capture captured_at", capture.CapturedAt); err != nil {
		return err
	}
	if capture.Query != columnDiscoveryQuery {
		return errors.New("capture query does not match the canonical SPANNER_SYS discovery query")
	}
	if err := validateSHA256("capture content_sha256", capture.ContentSHA256); err != nil {
		return err
	}
	if len(capture.Columns) == 0 {
		return errors.New("capture has no columns")
	}

	seenColumns := make(map[string]bool, len(capture.Columns))
	previousTable := ""
	previousOrdinal := 0
	for index, column := range capture.Columns {
		if column.TableName == "" || column.ColumnName == "" || column.SpannerType == "" {
			return fmt.Errorf("capture column %d has an empty table, column, or type", index)
		}
		if column.OrdinalPosition < 1 {
			return fmt.Errorf(
				"capture column %s.%s ordinal_position = %d, want at least 1",
				column.TableName,
				column.ColumnName,
				column.OrdinalPosition,
			)
		}

		key := column.TableName + "\x00" + column.ColumnName
		if seenColumns[key] {
			return fmt.Errorf("capture has duplicate column %s.%s", column.TableName, column.ColumnName)
		}
		seenColumns[key] = true

		switch {
		case previousTable == "" || column.TableName > previousTable:
			if column.OrdinalPosition != 1 {
				return fmt.Errorf(
					"capture table %s starts at ordinal_position %d, want 1",
					column.TableName,
					column.OrdinalPosition,
				)
			}
			previousOrdinal = column.OrdinalPosition
		case column.TableName < previousTable:
			return fmt.Errorf("capture columns are not sorted by table and ordinal at %s.%s", column.TableName, column.ColumnName)
		case column.OrdinalPosition != previousOrdinal+1:
			return fmt.Errorf(
				"capture column %s.%s ordinal_position = %d, want %d",
				column.TableName,
				column.ColumnName,
				column.OrdinalPosition,
				previousOrdinal+1,
			)
		default:
			previousOrdinal = column.OrdinalPosition
		}
		previousTable = column.TableName
	}

	wantHash, err := hashJSON(capture.Columns)
	if err != nil {
		return fmt.Errorf("hash capture columns: %w", err)
	}
	if capture.ContentSHA256 != wantHash {
		return fmt.Errorf("capture content_sha256 = %q, want %q", capture.ContentSHA256, wantHash)
	}
	return nil
}

func validateDate(label, value string) error {
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil || parsed.Format(time.DateOnly) != value {
		return fmt.Errorf("%s = %q, want YYYY-MM-DD", label, value)
	}
	return nil
}

func validateSHA256(label, value string) error {
	if len(value) != sha256.Size*2 || value != strings.ToLower(value) {
		return fmt.Errorf("%s = %q, want 64 lowercase hexadecimal characters", label, value)
	}
	if _, err := hex.DecodeString(value); err != nil {
		return fmt.Errorf("%s = %q, want 64 lowercase hexadecimal characters", label, value)
	}
	return nil
}

func rejectDestinationIdentifiers(data []byte) error {
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

func hashJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
