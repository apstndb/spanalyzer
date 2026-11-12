package querygen

import (
	"strings"

	spanalyzer "github.com/apstndb/spanalyzer"
)

func writeMutationCounts(analyzer *spanalyzer.MutationCountAnalyzer, spec resolvedWriteSpec) (map[string]spanalyzer.MutationCountReport, error) {
	return AnalyzeWriteMutationCounts(analyzer, newMutationCountPlanWrite(spec, nil, ""), nil)
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

func newMutationCountPlanWrite(spec resolvedWriteSpec, counts map[string]spanalyzer.MutationCountReport, countError string) QueryCodegenPlanWrite {
	return QueryCodegenPlanWrite{
		MutationCounts:          counts,
		MutationCountError:      countError,
		Name:                    spec.Name,
		Catalog:                 spec.Catalog,
		Table:                   spec.Table,
		Operation:               spec.Operation,
		InputStruct:             spec.InputStruct,
		Keys:                    columnNamesFromColumns(spec.Keys),
		InsertColumns:           columnNamesFromColumns(spec.InsertColumns),
		UpdateColumns:           columnNamesFromColumns(spec.UpdateColumns),
		Methods:                 spec.Methods,
		ColumnCapabilities:      planColumnCapabilities(spec.AllColumns),
		ServerSideUpdateEffects: planServerSideUpdateEffects(spec),
		VetSuppressions:         spec.VetSuppressions,
	}
}
