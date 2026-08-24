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

func TestAnalyzerRowTypeForTableWithoutPrimaryKey(t *testing.T) {
	const ddl = `
CREATE TABLE Singers (
  Name STRING(MAX),
  Rank INT64
);
`
	analyzer, err := NewAnalyzerFromDDL("schema.sql", ddl)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}

	starType, err := analyzer.RowTypeForStatement("SELECT * FROM Singers")
	if err != nil {
		t.Fatalf("RowTypeForStatement(SELECT *) error = %v", err)
	}
	if got, want := len(starType.Fields), 2; got != want {
		t.Fatalf("len(SELECT * fields) = %d, want %d", got, want)
	}
	assertField(t, starType.Fields[0], "Name", spannerpb.TypeCode_STRING)
	assertField(t, starType.Fields[1], "Rank", spannerpb.TypeCode_INT64)

	rowIDType, err := analyzer.RowTypeForStatement("SELECT rowid FROM Singers")
	if err != nil {
		t.Fatalf("RowTypeForStatement(SELECT rowid) error = %v", err)
	}
	if got, want := len(rowIDType.Fields), 1; got != want {
		t.Fatalf("len(SELECT rowid fields) = %d, want %d", got, want)
	}
	assertField(t, rowIDType.Fields[0], "rowid", spannerpb.TypeCode_INT64)
}

func TestAnalyzerRowTypeForUUID(t *testing.T) {
	paramType, err := ParseTypeSpec("param", "UUID")
	if err != nil {
		t.Fatalf("ParseTypeSpec(UUID) error = %v", err)
	}
	if got, want := paramType.Code, spannerpb.TypeCode_UUID; got != want {
		t.Fatalf("ParseTypeSpec(UUID).Code = %v, want %v", got, want)
	}

	const ddl = `
CREATE TABLE Fans (
  FanId UUID DEFAULT (NEW_UUID()),
) PRIMARY KEY (FanId);
`
	analyzer, err := NewAnalyzerFromDDL("schema.sql", ddl)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement("SELECT FanId, NEW_UUID() AS GeneratedId FROM Fans")
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 2; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "FanId", spannerpb.TypeCode_UUID)
	assertField(t, rowType.Fields[1], "GeneratedId", spannerpb.TypeCode_UUID)
}

func TestAnalyzerRowTypeForRemoteFunction(t *testing.T) {
	const ddl = `
CREATE SCHEMA spanalyzer_remote;
CREATE FUNCTION spanalyzer_remote.remote_add(x INT64, y INT64) RETURNS INT64
NOT DETERMINISTIC LANGUAGE REMOTE
OPTIONS (endpoint = "https://spanalyzer-remote-test-uc.a.run.app", max_batching_rows = 10);
`
	analyzer, err := NewAnalyzerFromDDL("schema.sql", ddl)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement("SELECT spanalyzer_remote.remote_add(1, 2) AS total")
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 1; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "total", spannerpb.TypeCode_INT64)
}

func TestAnalyzerRowTypeForZSTDCompressionFunctions(t *testing.T) {
	analyzer, err := NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement(`SELECT
  ZSTD_COMPRESS("hello") AS compressed_string,
  ZSTD_COMPRESS(b"hello", level => 1) AS compressed_bytes,
  ZSTD_DECOMPRESS_TO_BYTES(ZSTD_COMPRESS(b"hello")) AS decompressed_bytes,
  ZSTD_DECOMPRESS_TO_BYTES(ZSTD_COMPRESS(b"hello"), size_limit => 1024) AS limited_bytes,
  ZSTD_DECOMPRESS_TO_STRING(ZSTD_COMPRESS("hello")) AS decompressed_string,
  ZSTD_DECOMPRESS_TO_STRING(ZSTD_COMPRESS("hello"), size_limit => 1024) AS limited_string`)
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	want := []struct {
		name string
		code spannerpb.TypeCode
	}{
		{name: "compressed_string", code: spannerpb.TypeCode_BYTES},
		{name: "compressed_bytes", code: spannerpb.TypeCode_BYTES},
		{name: "decompressed_bytes", code: spannerpb.TypeCode_BYTES},
		{name: "limited_bytes", code: spannerpb.TypeCode_BYTES},
		{name: "decompressed_string", code: spannerpb.TypeCode_STRING},
		{name: "limited_string", code: spannerpb.TypeCode_STRING},
	}
	if got := len(rowType.Fields); got != len(want) {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, len(want))
	}
	for i, field := range want {
		assertField(t, rowType.Fields[i], field.name, field.code)
	}
}

func TestAnalyzerRowTypeForDistinctPredicates(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{
			name: "not distinct",
			sql:  "SELECT NULL IS NOT DISTINCT FROM NULL AS not_distinct",
			want: "not_distinct",
		},
		{
			name: "distinct",
			sql:  "SELECT NULL IS DISTINCT FROM NULL AS distinct_value",
			want: "distinct_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer, err := NewAnalyzerFromDDL("schema.sql", "")
			if err != nil {
				t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
			}
			rowType, err := analyzer.RowTypeForStatement(tt.sql)
			if err != nil {
				t.Fatalf("RowTypeForStatement() error = %v", err)
			}
			if got, want := len(rowType.Fields), 1; got != want {
				t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
			}
			assertField(t, rowType.Fields[0], tt.want, spannerpb.TypeCode_BOOL)
		})
	}
}

func TestAnalyzerRowTypeForBroaderGoogleSQLGrammarSurface(t *testing.T) {
	const ddl = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
) PRIMARY KEY (SingerId);

CREATE TABLE Albums (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  AlbumTitle STRING(MAX),
) PRIMARY KEY (SingerId, AlbumId);

CREATE TABLE Concerts (
  VenueId INT64 NOT NULL,
  TicketPrices ARRAY<INT64>,
) PRIMARY KEY (VenueId);
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
			name: "explicit correlated unnest",
			sql:  `SELECT c.VenueId, price FROM Concerts AS c CROSS JOIN UNNEST(c.TicketPrices) AS price`,
			wantFields: []struct {
				name string
				code spannerpb.TypeCode
			}{{"VenueId", spannerpb.TypeCode_INT64}, {"price", spannerpb.TypeCode_INT64}},
		},
		{
			name: "implicit correlated unnest",
			sql:  `SELECT c.VenueId, price FROM Concerts AS c, c.TicketPrices AS price`,
			wantFields: []struct {
				name string
				code spannerpb.TypeCode
			}{{"VenueId", spannerpb.TypeCode_INT64}, {"price", spannerpb.TypeCode_INT64}},
		},
		{
			name: "left correlated unnest",
			sql:  `SELECT c.VenueId, price FROM Concerts AS c LEFT JOIN UNNEST(c.TicketPrices) AS price ON TRUE`,
			wantFields: []struct {
				name string
				code spannerpb.TypeCode
			}{{"VenueId", spannerpb.TypeCode_INT64}, {"price", spannerpb.TypeCode_INT64}},
		},
		{
			name: "in unnested array subquery",
			sql:  `SELECT SingerId FROM Singers WHERE SingerId IN UNNEST(ARRAY(SELECT SingerId FROM Albums))`,
			wantFields: []struct {
				name string
				code spannerpb.TypeCode
			}{{"SingerId", spannerpb.TypeCode_INT64}},
		},
		{
			name: "group by ordinal",
			sql:  `SELECT SingerId, COUNT(*) AS album_count FROM Albums GROUP BY 1`,
			wantFields: []struct {
				name string
				code spannerpb.TypeCode
			}{{"SingerId", spannerpb.TypeCode_INT64}, {"album_count", spannerpb.TypeCode_INT64}},
		},
		{
			name: "aggregate having min",
			sql:  `SELECT SingerId, ANY_VALUE(AlbumTitle HAVING MIN AlbumId) AS earliest_title FROM Albums GROUP BY SingerId`,
			wantFields: []struct {
				name string
				code spannerpb.TypeCode
			}{{"SingerId", spannerpb.TypeCode_INT64}, {"earliest_title", spannerpb.TypeCode_STRING}},
		},
		{
			name: "tablesample on subquery",
			sql:  `SELECT SingerId FROM (SELECT SingerId FROM Singers WHERE SingerId > 0) TABLESAMPLE BERNOULLI (50 PERCENT)`,
			wantFields: []struct {
				name string
				code spannerpb.TypeCode
			}{{"SingerId", spannerpb.TypeCode_INT64}},
		},
		{
			name: "struct expression star",
			sql:  `SELECT row_value.* FROM (SELECT STRUCT(1 AS x, "a" AS y) AS row_value)`,
			wantFields: []struct {
				name string
				code spannerpb.TypeCode
			}{{"x", spannerpb.TypeCode_INT64}, {"y", spannerpb.TypeCode_STRING}},
		},
		{
			name: "unnest array of struct",
			sql:  `SELECT * FROM UNNEST(ARRAY<STRUCT<x INT64, y STRING>>[(1, "a"), (2, "b")])`,
			wantFields: []struct {
				name string
				code spannerpb.TypeCode
			}{{"x", spannerpb.TypeCode_INT64}, {"y", spannerpb.TypeCode_STRING}},
		},
		{
			name: "set operation regular first",
			sql:  `SELECT SingerId FROM Singers UNION ALL SELECT AS VALUE SingerId FROM Albums`,
			wantFields: []struct {
				name string
				code spannerpb.TypeCode
			}{{"SingerId", spannerpb.TypeCode_INT64}},
		},
		{
			name: "set operation value first",
			sql:  `SELECT AS VALUE SingerId FROM Singers UNION ALL SELECT SingerId FROM Albums`,
			wantFields: []struct {
				name string
				code spannerpb.TypeCode
			}{{"$value_column", spannerpb.TypeCode_INT64}},
		},
		{
			name: "lateral",
			sql:  `SELECT s.SingerId, a.AlbumId FROM Singers AS s CROSS JOIN LATERAL (SELECT AlbumId FROM Albums WHERE Albums.SingerId = s.SingerId) AS a`,
			wantFields: []struct {
				name string
				code spannerpb.TypeCode
			}{{"SingerId", spannerpb.TypeCode_INT64}, {"AlbumId", spannerpb.TypeCode_INT64}},
		},
		{
			name: "set operation by name",
			sql:  `SELECT SingerId AS id, FirstName AS name FROM Singers UNION ALL BY NAME SELECT AlbumId AS id, AlbumTitle AS name FROM Albums`,
			wantFields: []struct {
				name string
				code spannerpb.TypeCode
			}{{"id", spannerpb.TypeCode_INT64}, {"name", spannerpb.TypeCode_STRING}},
		},
		{
			name: "set operation corresponding",
			sql:  `SELECT SingerId AS id, FirstName AS name FROM Singers UNION ALL CORRESPONDING SELECT AlbumId AS id, AlbumTitle AS name FROM Albums`,
			wantFields: []struct {
				name string
				code spannerpb.TypeCode
			}{{"id", spannerpb.TypeCode_INT64}, {"name", spannerpb.TypeCode_STRING}},
		},
		{
			name: "group by all",
			sql:  `SELECT SingerId, COUNT(*) AS album_count FROM Albums GROUP BY ALL`,
			wantFields: []struct {
				name string
				code spannerpb.TypeCode
			}{{"SingerId", spannerpb.TypeCode_INT64}, {"album_count", spannerpb.TypeCode_INT64}},
		},
		{
			name: "match recognize",
			sql:  `SELECT * FROM Albums MATCH_RECOGNIZE (PARTITION BY SingerId ORDER BY AlbumId MEASURES MATCH_NUMBER() AS match_num PATTERN (A) DEFINE A AS TRUE)`,
			wantFields: []struct {
				name string
				code spannerpb.TypeCode
			}{{"SingerId", spannerpb.TypeCode_INT64}, {"match_num", spannerpb.TypeCode_INT64}},
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

	for _, expr := range []string{
		"ARRAY_FIRST([1, 2, 3])",
		"ARRAY_LAST([1, 2, 3])",
	} {
		exprOut, err := helper.AnalyzeExpression(expr)
		if err != nil {
			t.Fatalf("AnalyzeExpression(%q) error = %v", expr, err)
		}
		typ, err := TypeFromAnalyzerOutput(exprOut)
		if err != nil {
			t.Fatalf("TypeFromAnalyzerOutput(%q) error = %v", expr, err)
		}
		if got, want := typ.Code, spannerpb.TypeCode_INT64; got != want {
			t.Fatalf("%s type = %s, want %s", expr, got, want)
		}
	}
}

func TestAnalyzerArrayLambdaFunctionResultTypes(t *testing.T) {
	analyzer, err := NewAnalyzerFromDDL("schema.sql", "")
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	for _, tt := range []struct {
		name string
		sql  string
	}{
		{name: "transform", sql: "SELECT ARRAY_TRANSFORM([1, 2, 3], e -> e * 2) AS values"},
		{name: "filter", sql: "SELECT ARRAY_FILTER([1, 2, 3], e -> e != 2) AS values"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rowType, err := analyzer.RowTypeForStatement(tt.sql)
			if err != nil {
				t.Fatalf("RowTypeForStatement() error = %v", err)
			}
			if got, want := len(rowType.Fields), 1; got != want {
				t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
			}
			field := rowType.Fields[0]
			assertField(t, field, "values", spannerpb.TypeCode_ARRAY)
			if got, want := field.Type.GetArrayElementType().GetCode(), spannerpb.TypeCode_INT64; got != want {
				t.Fatalf("array element type = %s, want %s", got, want)
			}
		})
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

func TestAnalyzerRowTypeForDefinerView(t *testing.T) {
	const ddl = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
) PRIMARY KEY (SingerId);

CREATE VIEW SingerNames SQL SECURITY DEFINER AS
SELECT SingerId, FirstName AS Name FROM Singers;
`
	analyzer, err := NewAnalyzerFromDDL("definer_view.sql", ddl)
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

func TestAnalyzerRowTypeForPropertyGraphWithDynamicMetadata(t *testing.T) {
	const ddl = `
CREATE TABLE GraphNode (
  id INT64 NOT NULL,
  label STRING(MAX) NOT NULL,
  properties JSON,
) PRIMARY KEY (id);

CREATE PROPERTY GRAPH DynamicGraph
  NODE TABLES (
    GraphNode
      DYNAMIC LABEL (label)
      DYNAMIC PROPERTIES (properties)
  );
`
	analyzer, err := NewAnalyzerFromDDL("dynamic_graph.sql", ddl)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement("GRAPH DynamicGraph MATCH (n) RETURN n.id AS id")
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 1; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	// Because id is not declared as a static graph property, the frontend
	// resolves it through DYNAMIC PROPERTIES and reports JSON.
	assertField(t, rowType.Fields[0], "id", spannerpb.TypeCode_JSON)
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

	replacedRowType, err := analyzer.RowTypeForStatement("SELECT order_value.order_number, order_value.shipping_address.country FROM (SELECT REPLACE_FIELDS(NEW `examples.shipping.Order` { order_number: \"A-1\" shipping_address { country: \"US\" } }, \"B-2\" AS order_number, \"CA\" AS shipping_address.country) AS order_value)")
	if err != nil {
		t.Fatalf("RowTypeForStatement(REPLACE_FIELDS) error = %v", err)
	}
	if got, want := len(replacedRowType.Fields), 2; got != want {
		t.Fatalf("len(REPLACE_FIELDS row fields) = %d, want %d", got, want)
	}
	assertField(t, replacedRowType.Fields[0], "order_number", spannerpb.TypeCode_STRING)
	assertField(t, replacedRowType.Fields[1], "country", spannerpb.TypeCode_STRING)

	for _, tt := range []struct {
		name     string
		sql      string
		wantName string
		wantCode spannerpb.TypeCode
	}{
		{
			name:     "new-map-constructor-field-access",
			sql:      "SELECT order_value.order_number FROM (SELECT NEW `examples.shipping.Order` { order_number: \"A-1\" date: 123 } AS order_value)",
			wantName: "order_number",
			wantCode: spannerpb.TypeCode_STRING,
		},
		{
			name:     "new-parenthesized-constructor-field-access",
			sql:      "SELECT order_value.order_number FROM (SELECT NEW `examples.shipping.Order`(\"A-1\" AS order_number, 123 AS date) AS order_value)",
			wantName: "order_number",
			wantCode: spannerpb.TypeCode_STRING,
		},
		{
			name:     "cast-string-to-proto-field-access",
			sql:      "SELECT order_value.order_number FROM (SELECT CAST('order_number: \"A-1\" date: 123' AS `examples.shipping.Order`) AS order_value)",
			wantName: "order_number",
			wantCode: spannerpb.TypeCode_STRING,
		},
		{
			name:     "select-as-proto-nested",
			sql:      "SELECT order_value.order_number FROM (SELECT AS `examples.shipping.Order` \"A-1\" AS order_number, 123 AS date) AS order_value",
			wantName: "order_number",
			wantCode: spannerpb.TypeCode_STRING,
		},
		{
			name:     "select-as-proto-distinct-nested",
			sql:      "SELECT order_value.order_number FROM (SELECT DISTINCT AS `examples.shipping.Order` CAST(Id AS STRING) AS order_number, Id AS date FROM Orders) AS order_value",
			wantName: "order_number",
			wantCode: spannerpb.TypeCode_STRING,
		},
		{
			name:     "nested-proto-field-access",
			sql:      "SELECT OrderInfo.shipping_address.country FROM Orders",
			wantName: "country",
			wantCode: spannerpb.TypeCode_STRING,
		},
		{
			name:     "proto-presence-field-access",
			sql:      "SELECT OrderInfo.has_order_number FROM Orders",
			wantName: "has_order_number",
			wantCode: spannerpb.TypeCode_BOOL,
		},
		{
			name:     "upstream-extract-proto-field",
			sql:      "SELECT EXTRACT(FIELD(order_number) FROM OrderInfo) AS order_number FROM Orders",
			wantName: "order_number",
			wantCode: spannerpb.TypeCode_STRING,
		},
		{
			name:     "upstream-extract-proto-presence",
			sql:      "SELECT EXTRACT(HAS(order_number) FROM OrderInfo) AS has_order_number FROM Orders",
			wantName: "has_order_number",
			wantCode: spannerpb.TypeCode_BOOL,
		},
		{
			name:     "upstream-extract-raw-proto-field",
			sql:      "SELECT EXTRACT(RAW(order_number) FROM OrderInfo) AS raw_order_number FROM Orders",
			wantName: "raw_order_number",
			wantCode: spannerpb.TypeCode_STRING,
		},
		{
			name:     "upstream-filter-proto-fields",
			sql:      "SELECT filtered.order_number FROM (SELECT FILTER_FIELDS(OrderInfo, +order_number) AS filtered FROM Orders)",
			wantName: "order_number",
			wantCode: spannerpb.TypeCode_STRING,
		},
		{
			name:     "upstream-extract-proto-oneof-case",
			sql:      "SELECT EXTRACT(ONEOF_CASE(fulfillment) FROM OrderInfo) AS fulfillment_case FROM Orders",
			wantName: "fulfillment_case",
			wantCode: spannerpb.TypeCode_STRING,
		},
		{
			name:     "repeated-proto-field-unnest",
			sql:      "SELECT item.product_name FROM Orders AS o, UNNEST(o.OrderInfo.line_item) AS item",
			wantName: "product_name",
			wantCode: spannerpb.TypeCode_STRING,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rowType, err := analyzer.RowTypeForStatement(tt.sql)
			if err != nil {
				t.Fatalf("RowTypeForStatement() error = %v", err)
			}
			if got, want := len(rowType.Fields), 1; got != want {
				t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
			}
			assertField(t, rowType.Fields[0], tt.wantName, tt.wantCode)
		})
	}

	_, err = analyzer.RowTypeForStatement("SELECT PROTO_DEFAULT_IF_NULL(OrderInfo.order_number) AS order_number FROM Orders")
	if err == nil || !strings.Contains(err.Error(), "Function not found: PROTO_DEFAULT_IF_NULL") {
		t.Fatalf("RowTypeForStatement(PROTO_DEFAULT_IF_NULL) error = %v, want missing-function error", err)
	}

	for _, tt := range []struct {
		name string
		sql  string
		want []string
	}{
		{
			name: "proto construction and access",
			sql:  "SELECT order_value.order_number FROM (SELECT NEW `examples.shipping.Order` { order_number: \"A-1\" date: 123 } AS order_value)",
			want: []string{"MakeProto", "GetProtoField"},
		},
		{
			name: "proto field replacement",
			sql:  "SELECT order_value.order_number FROM (SELECT REPLACE_FIELDS(NEW `examples.shipping.Order` { order_number: \"A-1\" }, \"B-2\" AS order_number) AS order_value)",
			want: []string{"MakeProto", "ReplaceField", "GetProtoField"},
		},
		{
			name: "proto field filtering",
			sql:  "SELECT filtered.order_number FROM (SELECT FILTER_FIELDS(OrderInfo, +order_number) AS filtered FROM Orders)",
			want: []string{"FilterField", "GetProtoField"},
		},
		{
			name: "proto oneof case extraction",
			sql:  "SELECT EXTRACT(ONEOF_CASE(fulfillment) FROM OrderInfo) AS fulfillment_case FROM Orders",
			want: []string{"GetProtoOneof"},
		},
	} {
		t.Run(tt.name+" resolved AST", func(t *testing.T) {
			debug, err := analyzer.ResolvedASTDebugString("query", tt.sql)
			if err != nil {
				t.Fatalf("ResolvedASTDebugString() error = %v", err)
			}
			for _, want := range tt.want {
				if !strings.Contains(debug, want) {
					t.Errorf("ResolvedASTDebugString() missing %q in:\n%s", want, debug)
				}
			}
		})
	}
}

func TestAnalyzerRowTypeForPropertyGraphWithExpressions(t *testing.T) {
	const ddl = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName STRING(MAX),
) PRIMARY KEY (SingerId);

CREATE PROPERTY GRAPH MyGraph
  NODE TABLES (
    Singers
      LABEL Singer
      PROPERTIES (
        SingerId,
        CONCAT(FirstName, ' ', LastName) AS FullName
      )
  );
`
	analyzer, err := NewAnalyzerFromDDL("graph_expr.sql", ddl)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}

	sql := "GRAPH MyGraph MATCH (n:Singer) RETURN n.SingerId, n.FullName"
	rowType, err := analyzer.RowTypeForStatement(sql)
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 2; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "SingerId", spannerpb.TypeCode_INT64)
	// Derived properties are JSON.
	assertField(t, rowType.Fields[1], "FullName", spannerpb.TypeCode_JSON)
}

func TestAnalyzerRowTypeForDMLReturning(t *testing.T) {
	const ddl = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX)
) PRIMARY KEY (SingerId);
`
	analyzer, err := NewAnalyzerFromDDL("dml_returning.sql", ddl)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}

	sql := "DELETE FROM Singers WHERE SingerId = 1 THEN RETURN SingerId, FirstName"
	rowType, err := analyzer.RowTypeForStatement(sql)
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 2; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "SingerId", spannerpb.TypeCode_INT64)
	assertField(t, rowType.Fields[1], "FirstName", spannerpb.TypeCode_STRING)
}

func TestAnalyzerRowTypeForComplexProtoBundle(t *testing.T) {
	const ddl = `
CREATE PROTO BUNDLE (
  examples.user.User,
  examples.order.OrderExt,
);
CREATE TABLE Orders (
  OrderId STRING(MAX) NOT NULL,
  OrderInfo examples.order.OrderExt,
) PRIMARY KEY (OrderId);
`
	analyzer, err := NewAnalyzerFromDDLWithProtoDescriptorFiles("complex_proto.sql", ddl, []string{"testdata/protos/complex/complex_descriptors.pb"})
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDLWithProtoDescriptorFiles() error = %v", err)
	}

	sql := "SELECT OrderInfo.order_id, OrderInfo.customer.name, OrderInfo.customer.type FROM Orders"
	rowType, err := analyzer.RowTypeForStatement(sql)
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 3; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "order_id", spannerpb.TypeCode_STRING)
	assertField(t, rowType.Fields[1], "name", spannerpb.TypeCode_STRING)
	// Native MakeEnumType results in ENUM type.
	assertEnumField(t, rowType.Fields[2], "type", "examples.user.User.UserType")
}

func assertEnumField(t *testing.T, field *spannerpb.StructType_Field, name, fqn string) {
	t.Helper()
	if field.Name != name || field.Type.GetCode() != spannerpb.TypeCode_ENUM || field.Type.GetProtoTypeFqn() != fqn {
		t.Fatalf("field = (%q, %s, %q), want (%q, ENUM, %q)", field.Name, field.Type.GetCode(), field.Type.GetProtoTypeFqn(), name, fqn)
	}
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
