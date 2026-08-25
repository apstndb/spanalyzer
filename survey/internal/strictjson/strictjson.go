// Package strictjson decodes closed JSON contracts without accepting duplicate
// keys, unknown fields, trailing values, or multiple top-level values.
package strictjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Decode decodes one JSON value into destination using closed-contract rules.
func Decode(data []byte, destination any) error {
	if err := rejectDuplicateKeys(data); err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("decode JSON: multiple JSON values")
		}
		return fmt.Errorf("decode JSON trailer: %w", err)
	}
	return nil
}

func rejectDuplicateKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := scanValue(decoder, "$"); err != nil {
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

func scanValue(decoder *json.Decoder, path string) error {
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
			if err := scanValue(decoder, path+"."+key); err != nil {
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
			if err := scanValue(decoder, fmt.Sprintf("%s[%d]", path, index)); err != nil {
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
