package querygen

import (
	"fmt"
	"slices"

	spanalyzer "github.com/apstndb/spanalyzer"
)

// MutationCountWriteInput supplies an explicit one-row scenario. Values and Old
// use Spanner wire scalar encodings. SeedDML is used only by an explicitly
// requested runtime measurement in a command-created fixture, never by analysis.
type MutationCountWriteInput struct {
	Values      map[string]interface{}               `json:"values,omitempty"`
	Facts       []spanalyzer.MutationCountRowFacts   `json:"facts,omitempty"`
	Calibration string                               `json:"calibration,omitempty"`
	Observation *spanalyzer.MutationCountObservation `json:"observation,omitempty"`
	SeedDML     []string                             `json:"seed_dml,omitempty"`
}

// MutationCountInputs names each configured write and API independently. Index
// snapshots are keyed by catalog, so a snapshot cannot silently cross schemas.
type MutationCountInputs struct {
	Writes   map[string]map[string]MutationCountWriteInput    `json:"writes,omitempty"`
	Catalogs map[string]spanalyzer.MutationCountIndexSnapshot `json:"catalogs,omitempty"`
}

// MutationCountWriteShape derives the same one-row shape used by explain-plan.
// Values remain unknown until explicitly supplied. DML update/delete may match
// zero rows, while a Mutation API update's success requires its target to exist.
func MutationCountWriteShape(write QueryCodegenPlanWrite, channel string, values map[string]interface{}) (spanalyzer.MutationWriteShape, error) {
	if !slices.Contains(write.Methods, channel) || (channel != "mutation" && channel != "dml") {
		return spanalyzer.MutationWriteShape{}, fmt.Errorf("write %s has no %s method", write.Name, channel)
	}
	columns := slices.Clone(write.InsertColumns)
	if write.Operation == "update" {
		columns = append(slices.Clone(write.Keys), write.UpdateColumns...)
	}
	if write.Operation == "delete" {
		columns = nil
	}
	shape := spanalyzer.MutationWriteShape{Operation: write.Operation, Table: write.Table, Columns: columns, Channel: channel, Values: values}
	if channel == "dml" && (write.Operation == "update" || write.Operation == "delete") {
		one := int64(1)
		shape.RowBounds = &spanalyzer.MutationCountInterval{Lower: 0, Upper: &one}
	}
	return shape, nil
}

// AnalyzeWriteMutationCounts refines each API separately, retaining schema
// bounds and any old-row or same-execution observation alongside the result.
func AnalyzeWriteMutationCounts(a *spanalyzer.MutationCountAnalyzer, write QueryCodegenPlanWrite, inputs map[string]MutationCountWriteInput) (map[string]spanalyzer.MutationCountReport, error) {
	for method := range inputs {
		if !slices.Contains(write.Methods, method) {
			return nil, fmt.Errorf("write %s has no %s method", write.Name, method)
		}
	}
	result := map[string]spanalyzer.MutationCountReport{}
	for _, method := range write.Methods {
		if method != "mutation" && method != "dml" {
			continue
		}
		input := inputs[method]
		shape, err := MutationCountWriteShape(write, method, input.Values)
		if err != nil {
			return nil, err
		}
		r, err := a.AnalyzeWriteShape(shape, spanalyzer.MutationCountOptions{Facts: input.Facts, Calibration: input.Calibration, Observation: input.Observation})
		if err != nil {
			return nil, fmt.Errorf("write %s %s accounting: %w", write.Name, method, err)
		}
		result[method] = r
	}
	return result, nil
}

// RefinePlanMutationCounts applies caller facts without database access. Unknown
// names, incompatible snapshots and malformed facts fail instead of silently
// returning the original wider interval.
func RefinePlanMutationCounts(plan *QueryCodegenPlan, config QueryCodegenConfig, baseDir string, inputs MutationCountInputs) error {
	schemas := map[string]QueryCodegenSchema{}
	for _, schema := range config.Schemas {
		schemas[schema.Name] = schema
	}
	models := map[string]*spanalyzer.MutationCountAnalyzer{}
	for name, snapshot := range inputs.Catalogs {
		schema, ok := schemas[name]
		if !ok {
			return fmt.Errorf("index snapshot names unknown catalog %q", name)
		}
		a, err := mutationCountCatalog(schema, baseDir)
		if err != nil {
			return err
		}
		if a == nil {
			return fmt.Errorf("catalog %s does not support mutation accounting", name)
		}
		a, err = a.WithObservedIndexes(snapshot.SchemaSHA256, snapshot.Indexes)
		if err != nil {
			return fmt.Errorf("catalog %s: %w", name, err)
		}
		models[name] = a
	}
	seen := map[string]bool{}
	for i := range plan.Writes {
		write := &plan.Writes[i]
		seen[write.Name] = true
		facts, supplied := inputs.Writes[write.Name]
		a := models[write.Catalog]
		if !supplied && a == nil {
			continue
		}
		if a == nil {
			var err error
			a, err = mutationCountCatalog(schemas[write.Catalog], baseDir)
			if err != nil {
				return err
			}
			if a == nil {
				return fmt.Errorf("write %s does not support mutation accounting", write.Name)
			}
			models[write.Catalog] = a
		}
		counts, err := AnalyzeWriteMutationCounts(a, *write, facts)
		if err != nil {
			return err
		}
		write.MutationCounts, write.MutationCountError = counts, ""
	}
	for name := range inputs.Writes {
		if !seen[name] {
			return fmt.Errorf("mutation facts name unknown or invalid write %q", name)
		}
	}
	return nil
}

// MutationCountAnalyzerForSchema constructs the non-executing accounting model
// with exactly the same DDL byte identity used by generated write reports.
func MutationCountAnalyzerForSchema(schema QueryCodegenSchema, baseDir string) (*spanalyzer.MutationCountAnalyzer, error) {
	return mutationCountCatalog(schema, baseDir)
}
