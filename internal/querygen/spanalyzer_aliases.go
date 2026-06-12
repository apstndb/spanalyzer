package querygen

import spanalyzer "github.com/apstndb/spanalyzer"

type Analyzer = spanalyzer.Analyzer
type BigQueryAnalyzer = spanalyzer.BigQueryAnalyzer
type BigQuerySpannerExternalDatasetBinding = spanalyzer.BigQuerySpannerExternalDatasetBinding
type BigQuerySpannerExternalDatasetOptions = spanalyzer.BigQuerySpannerExternalDatasetOptions
type BigQuerySpannerExternalDatasetTable = spanalyzer.BigQuerySpannerExternalDatasetTable
type BigQuerySpannerExternalDatasetColumn = spanalyzer.BigQuerySpannerExternalDatasetColumn
type BigQuerySpannerExternalDatasetVerification = spanalyzer.BigQuerySpannerExternalDatasetVerification
type BigQueryTableSchema = spanalyzer.BigQueryTableSchema
type BigQueryTableFieldSchema = spanalyzer.BigQueryTableFieldSchema
type Catalog = spanalyzer.Catalog
type Column = spanalyzer.Column
type Index = spanalyzer.Index
type QueryCodegenDiagnosticError = spanalyzer.QueryCodegenDiagnosticError
type QueryCodegenPlanVetSuppression = spanalyzer.QueryCodegenPlanVetSuppression
type QueryCodegenPlanWarning = spanalyzer.QueryCodegenPlanWarning
type Table = spanalyzer.Table
type TypeSpec = spanalyzer.TypeSpec

var (
	BuildBigQueryGoogleSQLCatalogFromDDL       = spanalyzer.BuildBigQueryGoogleSQLCatalogFromDDL
	BuildSchemaCatalog                         = spanalyzer.BuildSchemaCatalog
	NewAnalyzerFromDDLWithProtoDescriptorFiles = spanalyzer.NewAnalyzerFromDDLWithProtoDescriptorFiles
	NewBigQueryAnalyzerFromDDL                 = spanalyzer.NewBigQueryAnalyzerFromDDL
	ParseTypeSpec                              = spanalyzer.ParseTypeSpec
)

func queryCodegenDiagnosticError(id, stage, subject, message string) error {
	return spanalyzer.QueryCodegenDiagnosticError{
		ID:      id,
		Stage:   stage,
		Subject: subject,
		Message: message,
	}
}
