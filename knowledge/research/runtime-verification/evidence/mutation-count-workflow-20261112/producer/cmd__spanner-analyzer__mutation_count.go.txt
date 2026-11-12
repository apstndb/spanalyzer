package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	spanalyzer "github.com/apstndb/spanalyzer"
)

func runMutationCountCLI(ddl, sql, output, mutations, facts, indexes, observation, calibration string) (string, error) {
	if !strings.EqualFold(output, "json") && !strings.EqualFold(output, "yaml") {
		return "", fmt.Errorf("mutation_count output must be json or yaml")
	}
	if (mutations == "") == (sql == "") {
		return "", fmt.Errorf("mutation_count requires exactly one of --mutations-file or --sql DML")
	}
	a, err := spanalyzer.NewMutationCountAnalyzer(ddl)
	if err != nil {
		return "", err
	}
	options := spanalyzer.MutationCountOptions{Calibration: calibration}
	if facts != "" {
		if err := readMutationJSON(facts, &options.Facts); err != nil {
			return "", err
		}
	}
	if indexes != "" {
		var snapshot struct {
			SchemaSHA256 string                                     `json:"schema_sha256"`
			Indexes      []spanalyzer.MutationCountIndexObservation `json:"indexes"`
		}
		if err := readMutationJSON(indexes, &snapshot); err != nil {
			return "", err
		}
		a, err = a.WithObservedIndexes(snapshot.SchemaSHA256, snapshot.Indexes)
		if err != nil {
			return "", err
		}
	}
	if observation != "" {
		options.Observation = &spanalyzer.MutationCountObservation{}
		if err := readMutationJSON(observation, options.Observation); err != nil {
			return "", err
		}
	}
	var report spanalyzer.MutationCountReport
	if mutations != "" {
		b, err := os.ReadFile(mutations)
		if err != nil {
			return "", err
		}
		report, err = a.AnalyzeMutationsJSON(b, options)
		if err != nil {
			return "", err
		}
	} else {
		report, err = a.AnalyzeDML([]string{sql}, options)
		if err != nil {
			queueReport, queueErr := a.AnalyzeQueueRead(sql)
			if queueErr != nil {
				return "", err
			}
			if len(options.Facts) > 0 || options.Observation != nil || options.Calibration != "" {
				return "", fmt.Errorf("queue TVFs have no caller CommitStats or ordinary row-price refinement")
			}
			report = queueReport
		}
	}
	b, err := marshalJSONYAML(report, output)
	if err != nil {
		return "", err
	}
	return string(b) + "\n", nil
}
func readMutationJSON(path string, v interface{}) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	// Refinements are caller assertions. A misspelled field must not silently
	// drop old-row or measurement evidence and leave a wider-looking success.
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := decoder.Decode(new(interface{})); err != io.EOF {
		return fmt.Errorf("%s: expected exactly one JSON document", path)
	}
	return nil
}
