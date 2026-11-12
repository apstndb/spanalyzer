package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	spanalyzer "github.com/apstndb/spanalyzer"
	"github.com/apstndb/spanalyzer/internal/querygen"
	"github.com/apstndb/spanalyzer/plancontract"
)

func readQuerygenMutationInputs(baseDir, factsPath, indexesPath string) (querygen.MutationCountInputs, map[string]string, error) {
	inputs := querygen.MutationCountInputs{}
	digests := map[string]string{}
	for _, file := range []struct{ name, path string }{{"facts", factsPath}, {"indexes", indexesPath}} {
		if file.path == "" {
			continue
		}
		data, err := os.ReadFile(resolveOutputPath(baseDir, file.path))
		if err != nil {
			return inputs, nil, err
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		var destination interface{}
		facts := struct {
			Writes map[string]map[string]querygen.MutationCountWriteInput `json:"writes"`
		}{}
		indexes := struct {
			Catalogs map[string]spanalyzer.MutationCountIndexSnapshot `json:"catalogs"`
		}{}
		if file.name == "facts" {
			destination = &facts
		} else {
			destination = &indexes
		}
		if err := decoder.Decode(destination); err != nil {
			return inputs, nil, fmt.Errorf("%s: %w", file.path, err)
		}
		if err := decoder.Decode(new(interface{})); err != io.EOF {
			return inputs, nil, fmt.Errorf("%s: expected exactly one JSON document", file.path)
		}
		if file.name == "facts" {
			inputs.Writes = facts.Writes
		} else {
			inputs.Catalogs = indexes.Catalogs
		}
		digests[file.name] = planReportDigest(string(data))
	}
	return inputs, digests, nil
}

func planContractMutationCounts(counts map[string]spanalyzer.MutationCountReport) map[string]plancontract.MutationCountEvidence {
	out := make(map[string]plancontract.MutationCountEvidence, len(counts))
	for method, count := range counts {
		out[method] = plancontract.MutationCountEvidence{Effective: plancontract.MutationCountInterval{Lower: count.Effective.Lower, Upper: count.Effective.Upper, Overflow: count.Effective.Overflow}, EffectiveBasis: count.EffectiveBasis, Disagreements: count.Disagreements}
	}
	return out
}

func mutationCountsJSONSchema() map[string]interface{} {
	interval := objectSchema([]interface{}{"lower", "upper"}, map[string]interface{}{
		"lower":    nonNegativeIntegerSchema("Conditional lower bound."),
		"upper":    map[string]interface{}{"type": []interface{}{"integer", "null"}, "minimum": 0, "description": "Conditional upper bound; null means unknown, never zero."},
		"overflow": boolSchema("Whether interval arithmetic overflowed."),
	})
	// The root library owns this evolving preview report. Validate the fields
	// consumed by contracts while allowing its separately versioned detail.
	report := map[string]interface{}{"type": "object", "required": []interface{}{"version", "effective", "effective_basis"}, "properties": map[string]interface{}{
		"version":         map[string]interface{}{"const": "spanalyzer.mutation-count/v0alpha1"},
		"effective":       interval,
		"effective_basis": stringSchema("Facts or same-execution observation on which this conditional interval depends."),
	}, "additionalProperties": true}
	return objectSchema(nil, map[string]interface{}{"mutation": report, "dml": report})
}

func writePlanReportMutationCountsMarkdown(b *strings.Builder, query planReportQuery) {
	if query.MutationCountError != "" {
		fmt.Fprintf(b, "Mutation count error: %s\n\n", query.MutationCountError)
	}
	if len(query.MutationCounts) == 0 {
		return
	}
	b.WriteString("| API | Conditional interval | Basis | Observed count |\n| --- | --- | --- | --- |\n")
	for _, method := range []string{"mutation", "dml"} {
		count, ok := query.MutationCounts[method]
		if !ok {
			continue
		}
		upper := "unknown"
		if count.Effective.Upper != nil {
			upper = fmt.Sprint(*count.Effective.Upper)
		}
		observed := "unknown"
		if count.Observation != nil && count.Observation.Successful && count.Observation.StatsPresent && count.Observation.Count != nil {
			observed = fmt.Sprint(*count.Observation.Count)
		}
		fmt.Fprintf(b, "| %s | %d..%s | %s | %s |\n", method, count.Effective.Lower, upper, count.EffectiveBasis, observed)
	}
	b.WriteByte('\n')
}
