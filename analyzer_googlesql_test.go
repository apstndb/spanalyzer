package spanalyzer

import (
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

func TestAnalyzerRowTypeForStatement(t *testing.T) {
	const ddl = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  Active BOOL,
) PRIMARY KEY (SingerId);
`
	analyzer, err := NewAnalyzerFromDDL("schema.sql", ddl)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement("SELECT SingerId, FirstName AS name, Active FROM Singers")
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 3; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "SingerId", spannerpb.TypeCode_INT64)
	assertField(t, rowType.Fields[1], "name", spannerpb.TypeCode_STRING)
	assertField(t, rowType.Fields[2], "Active", spannerpb.TypeCode_BOOL)
}

func TestComposableGoogleSQLCatalogHelperAndResultConversion(t *testing.T) {
	const ddl = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
) PRIMARY KEY (SingerId);
`
	googleSQLCatalog, err := BuildGoogleSQLCatalogFromDDL("schema.sql", ddl, nil)
	if err != nil {
		t.Fatalf("BuildGoogleSQLCatalogFromDDL() error = %v", err)
	}
	helper := googleSQLCatalog.Helper()
	out, err := helper.AnalyzeStatement("SELECT COUNT(*) AS singer_count, MIN(FirstName) AS first_name FROM Singers")
	if err != nil {
		t.Fatalf("AnalyzeStatement() error = %v", err)
	}
	rowType, err := RowTypeFromAnalyzerOutput(out, googleSQLCatalog.SpannerCatalog)
	if err != nil {
		t.Fatalf("RowTypeFromAnalyzerOutput() error = %v", err)
	}
	if got, want := len(rowType.Fields), 2; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "singer_count", spannerpb.TypeCode_INT64)
	assertField(t, rowType.Fields[1], "first_name", spannerpb.TypeCode_STRING)

	exprOut, err := helper.AnalyzeExpression("ARRAY_FIRST([1, 2, 3])")
	if err != nil {
		t.Fatalf("AnalyzeExpression() error = %v", err)
	}
	typ, err := TypeFromAnalyzerOutput(exprOut)
	if err != nil {
		t.Fatalf("TypeFromAnalyzerOutput() error = %v", err)
	}
	if got, want := typ.Code, spannerpb.TypeCode_INT64; got != want {
		t.Fatalf("typ.Code = %s, want %s", got, want)
	}
}

func TestAnalyzerRowTypeForView(t *testing.T) {
	const ddl = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName STRING(MAX),
) PRIMARY KEY (SingerId);

CREATE VIEW SingerNames SQL SECURITY INVOKER AS
SELECT SingerId, FirstName AS Name FROM Singers;
`
	analyzer, err := NewAnalyzerFromDDL("schema.sql", ddl)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement("SELECT * FROM SingerNames")
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 2; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "SingerId", spannerpb.TypeCode_INT64)
	assertField(t, rowType.Fields[1], "Name", spannerpb.TypeCode_STRING)
}

func TestAnalyzerRowTypeForTableSynonym(t *testing.T) {
	const ddl = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  SYNONYM (SingerAlias)
) PRIMARY KEY (SingerId);
`
	analyzer, err := NewAnalyzerFromDDL("schema.sql", ddl)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement("SELECT SingerId, FirstName FROM SingerAlias")
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 2; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "SingerId", spannerpb.TypeCode_INT64)
	assertField(t, rowType.Fields[1], "FirstName", spannerpb.TypeCode_STRING)
}

func TestAnalyzerRowTypeForSpannerRegisteredFunctions(t *testing.T) {
	const ddl = `
CREATE SEQUENCE MySequence OPTIONS (sequence_kind = 'bit_reversed_positive');
`
	analyzer, err := NewAnalyzerFromDDL("schema.sql", ddl)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement(`
SELECT
  PENDING_COMMIT_TIMESTAMP() AS commit_ts,
  BIT_REVERSE(1) AS reversed,
  BIT_REVERSE(1, TRUE) AS reversed_with_sign,
  GET_NEXT_SEQUENCE_VALUE(SEQUENCE MySequence) AS next_value,
  GET_NEXT_SEQUENCE_VALUE('MySequence') AS next_value_by_name,
  GET_INTERNAL_SEQUENCE_STATE(SEQUENCE MySequence) AS sequence_state,
  GET_INTERNAL_SEQUENCE_STATE('MySequence') AS sequence_state_by_name,
  GET_TABLE_COLUMN_IDENTITY_STATE('Singers.SingerId') AS identity_state
`)
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 8; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "commit_ts", spannerpb.TypeCode_TIMESTAMP)
	assertField(t, rowType.Fields[1], "reversed", spannerpb.TypeCode_INT64)
	assertField(t, rowType.Fields[2], "reversed_with_sign", spannerpb.TypeCode_INT64)
	assertField(t, rowType.Fields[3], "next_value", spannerpb.TypeCode_INT64)
	assertField(t, rowType.Fields[4], "next_value_by_name", spannerpb.TypeCode_INT64)
	assertField(t, rowType.Fields[5], "sequence_state", spannerpb.TypeCode_INT64)
	assertField(t, rowType.Fields[6], "sequence_state_by_name", spannerpb.TypeCode_INT64)
	assertField(t, rowType.Fields[7], "identity_state", spannerpb.TypeCode_INT64)
}

func TestAnalyzerRowTypeForNamedQueryParameters(t *testing.T) {
	analyzer, err := NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	idType, err := ParseTypeSpec("param", "INT64")
	if err != nil {
		t.Fatalf("ParseTypeSpec() error = %v", err)
	}
	nameType, err := ParseTypeSpec("param", "STRING(MAX)")
	if err != nil {
		t.Fatalf("ParseTypeSpec() error = %v", err)
	}
	if err := analyzer.AddQueryParameter("id", idType); err != nil {
		t.Fatalf("AddQueryParameter(id) error = %v", err)
	}
	if err := analyzer.AddQueryParameter("name", nameType); err != nil {
		t.Fatalf("AddQueryParameter(name) error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement("SELECT @id AS id, @name AS name")
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 2; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "id", spannerpb.TypeCode_INT64)
	assertField(t, rowType.Fields[1], "name", spannerpb.TypeCode_STRING)
}

func TestAnalyzerRowTypeForExpression(t *testing.T) {
	analyzer, err := NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	paramType, err := ParseTypeSpec("param", "STRING(MAX)")
	if err != nil {
		t.Fatalf("ParseTypeSpec() error = %v", err)
	}
	if err := analyzer.AddQueryParameter("prompt", paramType); err != nil {
		t.Fatalf("AddQueryParameter() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForExpression("AI.SCORE(@prompt)")
	if err != nil {
		t.Fatalf("RowTypeForExpression() error = %v", err)
	}
	if got, want := len(rowType.Fields), 1; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "expression", spannerpb.TypeCode_FLOAT64)
}

func TestAnalyzerParseUnparseAndResolvedASTDebugString(t *testing.T) {
	analyzer, err := NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	parseTree, err := analyzer.ParseDebugString("query", "SELECT 1 AS n")
	if err != nil {
		t.Fatalf("ParseDebugString() error = %v", err)
	}
	if !strings.Contains(parseTree, "QueryStatement") {
		t.Fatalf("parse tree does not contain QueryStatement:\n%s", parseTree)
	}
	unparsed, err := analyzer.Unparse("query", "SELECT 1 AS n")
	if err != nil {
		t.Fatalf("Unparse() error = %v", err)
	}
	if !strings.Contains(unparsed, "SELECT") {
		t.Fatalf("unparse does not contain SELECT: %s", unparsed)
	}
	resolved, err := analyzer.ResolvedASTDebugString("query", "SELECT 1 AS n")
	if err != nil {
		t.Fatalf("ResolvedASTDebugString() error = %v", err)
	}
	if !strings.Contains(resolved, "QueryStmt") {
		t.Fatalf("resolved AST does not contain QueryStmt:\n%s", resolved)
	}
}

func TestAnalyzerFunctionCatalogDebugString(t *testing.T) {
	analyzer, err := NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	dump, err := analyzer.FunctionCatalogDebugString(true)
	if err != nil {
		t.Fatalf("FunctionCatalogDebugString() error = %v", err)
	}
	for _, want := range []string{
		"GoogleSQL:sum",
		"Spanner:BIT_REVERSE",
		"(INT64, BOOL) -> INT64",
	} {
		if !strings.Contains(dump, want) {
			t.Fatalf("FunctionCatalogDebugString() does not contain %q:\n%s", want, dump)
		}
	}
}

func TestAnalyzerRowTypeForInformationSchema(t *testing.T) {
	analyzer, err := NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement(`
SELECT
  TABLE_NAME,
  COLUMN_NAME,
  ORDINAL_POSITION,
  SPANNER_TYPE
FROM INFORMATION_SCHEMA.COLUMNS
`)
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 4; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "TABLE_NAME", spannerpb.TypeCode_STRING)
	assertField(t, rowType.Fields[1], "COLUMN_NAME", spannerpb.TypeCode_STRING)
	assertField(t, rowType.Fields[2], "ORDINAL_POSITION", spannerpb.TypeCode_INT64)
	assertField(t, rowType.Fields[3], "SPANNER_TYPE", spannerpb.TypeCode_STRING)
}

func TestAnalyzerRowTypeForSpannerSys(t *testing.T) {
	analyzer, err := NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement(`
SELECT
  INTERVAL_END,
  TABLE_NAME,
  READ_QUERY_COUNT
FROM SPANNER_SYS.TABLE_OPERATIONS_STATS_MINUTE
`)
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 3; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "INTERVAL_END", spannerpb.TypeCode_TIMESTAMP)
	assertField(t, rowType.Fields[1], "TABLE_NAME", spannerpb.TypeCode_STRING)
	assertField(t, rowType.Fields[2], "READ_QUERY_COUNT", spannerpb.TypeCode_INT64)
}

func TestAnalyzerRowTypeForSpannerSysDistributionPercentile(t *testing.T) {
	analyzer, err := NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement(`
SELECT
  INTERVAL_END,
  AVG_LATENCY_SECONDS,
  SPANNER_SYS.DISTRIBUTION_PERCENTILE(LATENCY_DISTRIBUTION[OFFSET(0)], 99.0) AS percentile_latency,
  SPANNER_SYS.DISTRIBUTION_PERCENTILE(LATENCY_DISTRIBUTION_JSON_STRING, 99.0) AS percentile_latency_json
FROM SPANNER_SYS.QUERY_STATS_TOTAL_10MINUTE
WHERE INTERVAL_END = (
  SELECT MAX(INTERVAL_END)
  FROM SPANNER_SYS.QUERY_STATS_TOTAL_10MINUTE
)
ORDER BY INTERVAL_END
`)
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 4; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "INTERVAL_END", spannerpb.TypeCode_TIMESTAMP)
	assertField(t, rowType.Fields[1], "AVG_LATENCY_SECONDS", spannerpb.TypeCode_FLOAT64)
	assertField(t, rowType.Fields[2], "percentile_latency", spannerpb.TypeCode_FLOAT64)
	assertField(t, rowType.Fields[3], "percentile_latency_json", spannerpb.TypeCode_FLOAT64)
}

func TestAnalyzerRejectsRawSpannerSysDistributionArrayPercentile(t *testing.T) {
	analyzer, err := NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	_, err = analyzer.RowTypeForStatement(`
SELECT SPANNER_SYS.DISTRIBUTION_PERCENTILE(LATENCY_DISTRIBUTION, 99.0) AS percentile_latency
FROM SPANNER_SYS.QUERY_STATS_TOTAL_10MINUTE
`)
	if err == nil {
		t.Fatal("RowTypeForStatement() error = nil, want raw ARRAY<STRUCT> percentile argument to be rejected")
	}
	if !strings.Contains(err.Error(), "No matching signature for function") {
		t.Fatalf("RowTypeForStatement() error = %v, want signature mismatch", err)
	}
}

func TestAnalyzerRowTypeForSpannerSearchFunctions(t *testing.T) {
	const ddl = `
CREATE TABLE Albums (
  AlbumId INT64 NOT NULL,
  Description STRING(MAX),
  DescriptionTokens TOKENLIST AS (TOKENIZE_FULLTEXT(Description)) HIDDEN,
  DescriptionNgramTokens TOKENLIST AS (TOKENIZE_NGRAMS(Description)) HIDDEN,
  DescriptionSubstrTokens TOKENLIST AS (TOKENIZE_SUBSTRING(Description)) HIDDEN
) PRIMARY KEY (AlbumId);
`
	analyzer, err := NewAnalyzerFromDDL("schema.sql", ddl)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement(`
SELECT
  SEARCH(DescriptionTokens, 'classical') AS fulltext_hit,
  SEARCH_NGRAMS(DescriptionNgramTokens, 'clasic') AS ngram_hit,
  SEARCH_SUBSTRING(DescriptionSubstrTokens, 'ssic') AS substring_hit,
  SCORE(DescriptionTokens, 'classical') AS fulltext_score,
  SCORE_NGRAMS(DescriptionNgramTokens, 'clasic') AS ngram_score,
  SNIPPET(Description, 'classical') AS snippet,
  DEBUG_TOKENLIST(DescriptionTokens) AS debug_tokens
FROM Albums
`)
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 7; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "fulltext_hit", spannerpb.TypeCode_BOOL)
	assertField(t, rowType.Fields[1], "ngram_hit", spannerpb.TypeCode_BOOL)
	assertField(t, rowType.Fields[2], "substring_hit", spannerpb.TypeCode_BOOL)
	assertField(t, rowType.Fields[3], "fulltext_score", spannerpb.TypeCode_FLOAT64)
	assertField(t, rowType.Fields[4], "ngram_score", spannerpb.TypeCode_FLOAT64)
	assertField(t, rowType.Fields[5], "snippet", spannerpb.TypeCode_JSON)
	assertField(t, rowType.Fields[6], "debug_tokens", spannerpb.TypeCode_STRING)
}

func TestAnalyzerRowTypeForSpannerTokenlistFunctions(t *testing.T) {
	analyzer, err := NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement(`
SELECT
  DEBUG_TOKENLIST(TOKEN('exact')) AS exact_tokens,
  DEBUG_TOKENLIST(TOKENIZE_BOOL(TRUE)) AS bool_tokens,
  DEBUG_TOKENLIST(TOKENIZE_JSON(JSON '{"format":"vinyl"}')) AS json_tokens,
  SEARCH(TOKENLIST_CONCAT([TOKENIZE_FULLTEXT('classic'), TOKENIZE_FULLTEXT('album')]), 'classic') AS concat_hit
`)
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 4; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "exact_tokens", spannerpb.TypeCode_STRING)
	assertField(t, rowType.Fields[1], "bool_tokens", spannerpb.TypeCode_STRING)
	assertField(t, rowType.Fields[2], "json_tokens", spannerpb.TypeCode_STRING)
	assertField(t, rowType.Fields[3], "concat_hit", spannerpb.TypeCode_BOOL)
}

func TestAnalyzerRowTypeForSpannerFunctionNamedArgs(t *testing.T) {
	analyzer, err := NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement(`
SELECT
  SEARCH(TOKENIZE_FULLTEXT('classic', token_category => 'content'), 'classic', enhance_query => TRUE) AS search_hit,
  DEBUG_TOKENLIST(TOKENIZE_NUMBER(10, comparison_type => 'all', min => 0, max => 100, tree_base => 2)) AS number_tokens
`)
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 2; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "search_hit", spannerpb.TypeCode_BOOL)
	assertField(t, rowType.Fields[1], "number_tokens", spannerpb.TypeCode_STRING)
}

func TestAnalyzerRowTypeForSpannerAIFunctions(t *testing.T) {
	analyzer, err := NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement(`
SELECT
  AI.CLASSIFY('apple', categories => ['fruit', 'device']) AS category,
  AI.IF(prompt => 'Is Seattle a US city?') AS decision,
  AI.SCORE('Rate this on a scale from 1 to 10') AS score
`)
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 3; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "category", spannerpb.TypeCode_STRING)
	assertField(t, rowType.Fields[1], "decision", spannerpb.TypeCode_BOOL)
	assertField(t, rowType.Fields[2], "score", spannerpb.TypeCode_FLOAT64)
}

func TestAnalyzerRowTypeForViewUsingSpannerFunctions(t *testing.T) {
	const ddl = `
CREATE TABLE Albums (
  AlbumId INT64 NOT NULL,
  Description STRING(MAX),
  DescriptionTokens TOKENLIST AS (TOKENIZE_FULLTEXT(Description)) HIDDEN
) PRIMARY KEY (AlbumId);

CREATE VIEW AlbumHits SQL SECURITY INVOKER AS
SELECT AlbumId, SEARCH(DescriptionTokens, 'classic', enhance_query => TRUE) AS Hit FROM Albums;
`
	analyzer, err := NewAnalyzerFromDDL("schema.sql", ddl)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement("SELECT AlbumId, Hit FROM AlbumHits")
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 2; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "AlbumId", spannerpb.TypeCode_INT64)
	assertField(t, rowType.Fields[1], "Hit", spannerpb.TypeCode_BOOL)
}

func TestAnalyzerBuildsCreateModelDDL(t *testing.T) {
	const ddl = `
CREATE MODEL RatingModel
INPUT (score FLOAT64)
OUTPUT (label STRING(MAX))
REMOTE OPTIONS (endpoint = '//aiplatform.googleapis.com/projects/p/locations/us-central1/endpoints/e');
`
	if _, err := NewAnalyzerFromDDL("schema.sql", ddl); err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
}

func TestAnalyzerRowTypeForMLPredictGeminiPro(t *testing.T) {
	const ddl = `
CREATE MODEL GeminiPro
INPUT (prompt STRING(MAX))
OUTPUT (content STRING(MAX))
REMOTE OPTIONS (
  endpoint = '//aiplatform.googleapis.com/projects/p/locations/us-central1/publishers/google/models/gemini-pro',
  default_batch_size = 1
);
`
	analyzer, err := NewAnalyzerFromDDL("schema.sql", ddl)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement(`
SELECT content
FROM ML.PREDICT(
  MODEL GeminiPro,
  (SELECT "Is 7 a prime number?" AS prompt),
  STRUCT(256 AS maxOutputTokens, 0.2 AS temperature, 40 AS topK, 0.95 AS topP)
)
`)
	if err != nil {
		t.Fatalf("RowTypeForStatement(content) error = %v", err)
	}
	if got, want := len(rowType.Fields), 1; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "content", spannerpb.TypeCode_STRING)

	rowType, err = analyzer.RowTypeForStatement(`
SELECT *
FROM ML.PREDICT(
  MODEL GeminiPro,
  (SELECT "Is 7 a prime number?" AS prompt),
  STRUCT(256 AS maxOutputTokens, 0.2 AS temperature, 40 AS topK, 0.95 AS topP)
)
`)
	if err != nil {
		t.Fatalf("RowTypeForStatement(*) error = %v", err)
	}
	if got, want := len(rowType.Fields), 2; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "content", spannerpb.TypeCode_STRING)
	assertField(t, rowType.Fields[1], "prompt", spannerpb.TypeCode_STRING)
}

func TestAnalyzerRowTypeForMLPredictEmulatorCompatibility(t *testing.T) {
	const ddl = `
CREATE MODEL FraudDetection
INPUT (Amount INT64, Name STRING(MAX))
OUTPUT (Outcome BOOL)
REMOTE OPTIONS (
  endpoint = '//aiplatform.googleapis.com/projects/p/locations/us-central1/endpoints/e'
);
`
	tests := []struct {
		name       string
		sql        string
		wantFields []struct {
			name string
			code spannerpb.TypeCode
		}
	}{
		{
			name: "model outputs followed by pass-through input columns",
			sql: `
SELECT *
FROM ML.PREDICT(
  MODEL FraudDetection,
  (SELECT 1000 AS Amount, "John Smith" AS Name)
)
`,
			wantFields: []struct {
				name string
				code spannerpb.TypeCode
			}{
				{name: "Outcome", code: spannerpb.TypeCode_BOOL},
				{name: "Amount", code: spannerpb.TypeCode_INT64},
				{name: "Name", code: spannerpb.TypeCode_STRING},
			},
		},
		{
			name: "safe namespace",
			sql: `
SELECT *
FROM SAFE.ML.PREDICT(
  MODEL FraudDetection,
  (SELECT 1000 AS Amount, "John Smith" AS Name)
)
`,
			wantFields: []struct {
				name string
				code spannerpb.TypeCode
			}{
				{name: "Outcome", code: spannerpb.TypeCode_BOOL},
				{name: "Amount", code: spannerpb.TypeCode_INT64},
				{name: "Name", code: spannerpb.TypeCode_STRING},
			},
		},
		{
			name: "input column matching model output is not duplicated",
			sql: `
SELECT *
FROM ML.PREDICT(
  MODEL FraudDetection,
  (SELECT TRUE AS Outcome, 1000 AS Amount, "John Smith" AS Name)
)
`,
			wantFields: []struct {
				name string
				code spannerpb.TypeCode
			}{
				{name: "Outcome", code: spannerpb.TypeCode_BOOL},
				{name: "Amount", code: spannerpb.TypeCode_INT64},
				{name: "Name", code: spannerpb.TypeCode_STRING},
			},
		},
	}

	analyzer, err := NewAnalyzerFromDDL("schema.sql", ddl)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rowType, err := analyzer.RowTypeForStatement(tt.sql)
			if err != nil {
				t.Fatalf("RowTypeForStatement() error = %v", err)
			}
			if got, want := len(rowType.Fields), len(tt.wantFields); got != want {
				t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
			}
			for i, want := range tt.wantFields {
				assertField(t, rowType.Fields[i], want.name, want.code)
			}
		})
	}
}

func TestAnalyzerRowTypeForPropertyGraph(t *testing.T) {
	const ddl = `
CREATE TABLE Person (
  id INT64 NOT NULL,
) PRIMARY KEY (id);

CREATE PROPERTY GRAPH g
  NODE TABLES (
    Person LABEL Person PROPERTIES ARE ALL COLUMNS
  );
`
	analyzer, err := NewAnalyzerFromDDL("schema.sql", ddl)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement("SELECT * FROM GRAPH_TABLE(g MATCH (p:Person) RETURN p.id AS id)")
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 1; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "id", spannerpb.TypeCode_INT64)
}

func TestAnalyzerRowTypeForNestedView(t *testing.T) {
	const ddl = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
) PRIMARY KEY (SingerId);

CREATE VIEW SingerNameStrings SQL SECURITY INVOKER AS
SELECT CAST(SingerId AS STRING) AS SingerIdText, Name FROM SingerNames;

CREATE VIEW SingerNames SQL SECURITY INVOKER AS
SELECT SingerId, FirstName AS Name FROM Singers;
`
	analyzer, err := NewAnalyzerFromDDL("schema.sql", ddl)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement("SELECT SingerIdText, Name FROM SingerNameStrings")
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 2; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "SingerIdText", spannerpb.TypeCode_STRING)
	assertField(t, rowType.Fields[1], "Name", spannerpb.TypeCode_STRING)
}

func TestAnalyzerRowTypeForProtoBundleFieldAccess(t *testing.T) {
	const protoDescriptorPath = "testdata/protos/order_descriptors.pb"
	const ddl = `
CREATE PROTO BUNDLE (
  ` + "`examples.shipping.Order`" + `,
  ` + "`examples.shipping.Order.Address`" + `,
  ` + "`examples.shipping.Order.Item`" + `
);
CREATE TABLE Orders (
  Id INT64 NOT NULL,
  OrderInfo ` + "`examples.shipping.Order`" + `,
) PRIMARY KEY(Id);
`
	analyzer, err := NewAnalyzerFromDDLWithProtoDescriptorFiles("schema.sql", ddl, []string{protoDescriptorPath})
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDLWithProtoDescriptorFiles() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement("SELECT OrderInfo.order_number, OrderInfo.shipping_address.country, OrderInfo.shipping_address FROM Orders")
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 3; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "order_number", spannerpb.TypeCode_STRING)
	assertField(t, rowType.Fields[1], "country", spannerpb.TypeCode_STRING)
	assertProtoField(t, rowType.Fields[2], "shipping_address", "examples.shipping.Order.Address")
}

func TestAnalyzerRowTypeForProtoColumn(t *testing.T) {
	const protoDescriptorPath = "testdata/protos/order_descriptors.pb"
	const ddl = `
CREATE PROTO BUNDLE (
  ` + "`examples.shipping.Order`" + `,
  ` + "`examples.shipping.Order.Address`" + `,
  ` + "`examples.shipping.Order.Item`" + `
);
CREATE TABLE Orders (
  Id INT64 NOT NULL,
  OrderInfo ` + "`examples.shipping.Order`" + `,
) PRIMARY KEY(Id);
`
	analyzer, err := NewAnalyzerFromDDLWithProtoDescriptorFiles("schema.sql", ddl, []string{protoDescriptorPath})
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDLWithProtoDescriptorFiles() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement("SELECT OrderInfo FROM Orders")
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 1; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertProtoField(t, rowType.Fields[0], "OrderInfo", "examples.shipping.Order")
}

func assertField(t *testing.T, field *spannerpb.StructType_Field, name string, code spannerpb.TypeCode) {
	t.Helper()
	if field.Name != name || field.Type.GetCode() != code {
		t.Fatalf("field = (%q, %s), want (%q, %s)", field.Name, field.Type.GetCode(), name, code)
	}
}

func assertProtoField(t *testing.T, field *spannerpb.StructType_Field, name, fqn string) {
	t.Helper()
	if field.Name != name || field.Type.GetCode() != spannerpb.TypeCode_PROTO || field.Type.GetProtoTypeFqn() != fqn {
		t.Fatalf("field = (%q, %s, %q), want (%q, PROTO, %q)", field.Name, field.Type.GetCode(), field.Type.GetProtoTypeFqn(), name, fqn)
	}
}
