package querygen

import (
	"fmt"
	"strings"

	spanalyzer "github.com/apstndb/spanalyzer"
)

func writeMutationCounts(analyzer *spanalyzer.MutationCountAnalyzer, spec resolvedWriteSpec) (map[string]spanalyzer.MutationCountReport, error) {
	result := map[string]spanalyzer.MutationCountReport{}
	columns := columnNamesFromColumns(spec.InsertColumns)
	if spec.Operation == "update" {
		columns = append(columnNamesFromColumns(spec.Keys), columnNamesFromColumns(spec.UpdateColumns)...)
	}
	if spec.Operation == "delete" {
		columns = nil
	}
	for _, method := range spec.Methods {
		if method != "mutation" && method != "dml" {
			continue
		}
		shape := spanalyzer.MutationWriteShape{Operation: spec.Operation, Table: spec.Table, Columns: columns, Channel: method}
		if method == "dml" && (spec.Operation == "update" || spec.Operation == "delete") {
			one := int64(1)
			shape.RowBounds = &spanalyzer.MutationCountInterval{Lower: 0, Upper: &one}
		}
		r, err := analyzer.AnalyzeWriteShape(shape, spanalyzer.MutationCountOptions{})
		if err != nil {
			return nil, fmt.Errorf("write %s mutation accounting: %w", spec.Name, err)
		}
		result[method] = r
	}
	return result, nil
}

func mutationCountCatalog(schema QueryCodegenSchema, baseDir string) (*spanalyzer.MutationCountAnalyzer, error) {
	if !strings.EqualFold(emptyDefault(schema.Dialect, "spanner"), "spanner") {
		return nil, nil
	}
	_, ddl, err := readCodegenDDL(schema, baseDir)
	if err != nil {
		return nil, err
	}
	return spanalyzer.NewMutationCountAnalyzer(ddl)
}
