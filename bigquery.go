package spanalyzer

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	googlesql "github.com/goccy/go-googlesql"
)

// BigQueryGoogleSQLCatalog contains a GoogleSQL frontend catalog built from
// BigQuery DDL.
type BigQueryGoogleSQLCatalog struct {
	SimpleCatalog   *googlesql.SimpleCatalog
	simpleCatalogs  map[string]*googlesql.SimpleCatalog
	AnalyzerOptions *googlesql.AnalyzerOptions
	TypeFactory     *googlesql.TypeFactory
}

// BigQueryAnalyzer analyzes BigQuery GoogleSQL statements and returns BigQuery
// REST TableSchema-shaped metadata.
type BigQueryAnalyzer struct {
	googleSQL              *BigQueryGoogleSQLCatalog
	helper                 *GoogleSQLHelper
	externalQueryAnalyzers map[string]*Analyzer
}

// BigQueryTableSchema mirrors the BigQuery REST TableSchema JSON shape.
type BigQueryTableSchema struct {
	Fields []*BigQueryTableFieldSchema `json:"fields,omitempty"`
}

// BigQueryTableFieldSchema mirrors the BigQuery REST TableFieldSchema JSON
// shape for fields this project can infer from GoogleSQL analyzer types.
type BigQueryTableFieldSchema struct {
	Name             string                      `json:"name,omitempty"`
	Type             string                      `json:"type,omitempty"`
	Mode             string                      `json:"mode,omitempty"`
	Fields           []*BigQueryTableFieldSchema `json:"fields,omitempty"`
	RangeElementType *BigQueryFieldElementType   `json:"rangeElementType,omitempty"`
	Description      string                      `json:"description,omitempty"`
	MaxLength        string                      `json:"maxLength,omitempty"`
	Precision        string                      `json:"precision,omitempty"`
	Scale            string                      `json:"scale,omitempty"`
	Collation        string                      `json:"collation,omitempty"`
	DefaultValueExpr string                      `json:"defaultValueExpression,omitempty"`
	PolicyTags       *BigQueryPolicyTags         `json:"policyTags,omitempty"`
	DataPolicies     []*BigQueryDataPolicyOption `json:"dataPolicies,omitempty"`
	RoundingMode     string                      `json:"roundingMode,omitempty"`
}

// BigQueryFieldElementType mirrors the REST fieldElementType object used by
// RANGE fields.
type BigQueryFieldElementType struct {
	Type string `json:"type,omitempty"`
}

// BigQueryPolicyTags mirrors the REST policyTags object.
type BigQueryPolicyTags struct {
	Names []string `json:"names,omitempty"`
}

// BigQueryDataPolicyOption mirrors the REST dataPolicies entry shape.
type BigQueryDataPolicyOption struct {
	Name string `json:"name,omitempty"`
}

// BuildBigQueryGoogleSQLCatalogFromDDL analyzes BigQuery DDL and registers its
// tables and views in a GoogleSQL frontend catalog.
func BuildBigQueryGoogleSQLCatalogFromDDL(path, ddlSQL string, options ...AnalyzerOption) (*BigQueryGoogleSQLCatalog, error) {
	config := defaultAnalyzerConfig()
	for _, option := range options {
		option(&config)
	}
	if err := InitGoogleSQL(); err != nil {
		return nil, err
	}
	tf, err := googlesql.NewTypeFactory()
	if err != nil {
		return nil, err
	}
	simpleCatalog, opts, err := newGoogleSQLAnalyzerObjects("bigquery", tf, config)
	if err != nil {
		return nil, err
	}
	catalog := &BigQueryGoogleSQLCatalog{
		SimpleCatalog:   simpleCatalog,
		simpleCatalogs:  map[string]*googlesql.SimpleCatalog{"": simpleCatalog},
		AnalyzerOptions: opts,
		TypeFactory:     tf,
	}
	if strings.TrimSpace(ddlSQL) == "" {
		return catalog, nil
	}
	if err := catalog.applyDDL(path, ddlSQL); err != nil {
		return nil, err
	}
	return catalog, nil
}

// NewBigQueryAnalyzerFromDDL builds a BigQuery analyzer from BigQuery DDL.
func NewBigQueryAnalyzerFromDDL(path, ddlSQL string, options ...AnalyzerOption) (*BigQueryAnalyzer, error) {
	catalog, err := BuildBigQueryGoogleSQLCatalogFromDDL(path, ddlSQL, options...)
	if err != nil {
		return nil, err
	}
	return NewBigQueryAnalyzerFromGoogleSQLCatalog(catalog)
}

// NewBigQueryAnalyzerFromGoogleSQLCatalog wraps an existing BigQuery
// GoogleSQL catalog.
func NewBigQueryAnalyzerFromGoogleSQLCatalog(catalog *BigQueryGoogleSQLCatalog) (*BigQueryAnalyzer, error) {
	if catalog == nil {
		return nil, fmt.Errorf("nil BigQuery GoogleSQL catalog")
	}
	return &BigQueryAnalyzer{
		googleSQL: catalog,
		helper:    catalog.Helper(),
	}, nil
}

// SetExternalQueryAnalyzers sets Spanner analyzers keyed by BigQuery
// connection ID for EXTERNAL_QUERY result schema inference.
func (a *BigQueryAnalyzer) SetExternalQueryAnalyzers(analyzers map[string]*Analyzer) {
	a.externalQueryAnalyzers = analyzers
}

// Helper returns a reusable GoogleSQL helper for this catalog.
func (c *BigQueryGoogleSQLCatalog) Helper() *GoogleSQLHelper {
	return &GoogleSQLHelper{
		Catalog:     c.SimpleCatalog,
		Options:     c.AnalyzerOptions,
		TypeFactory: c.TypeFactory,
	}
}

func (c *BigQueryGoogleSQLCatalog) applyDDL(path, ddlSQL string) error {
	location, err := googlesql.NewParseResumeLocationFromString2(path, ddlSQL)
	if err != nil {
		return err
	}
	for {
		if atEnd, err := parseResumeLocationAtEnd(location); err != nil {
			return err
		} else if atEnd {
			return nil
		}
		out, err := googlesql.AnalyzeNextStatement(location, c.AnalyzerOptions, c.SimpleCatalog, c.TypeFactory)
		if err != nil {
			return err
		}
		if out == nil {
			return nil
		}
		if err := c.applyDDLAnalyzerOutput(out); err != nil {
			return err
		}
	}
}

func parseResumeLocationAtEnd(location *googlesql.ParseResumeLocation) (bool, error) {
	input, err := location.Input()
	if err != nil {
		return false, err
	}
	position, err := location.BytePosition()
	if err != nil {
		return false, err
	}
	if position < 0 || int(position) > len(input) {
		return false, fmt.Errorf("invalid parse resume byte position %d", position)
	}
	remaining := strings.TrimSpace(input[position:])
	return remaining == "" || remaining == ";", nil
}

func (c *BigQueryGoogleSQLCatalog) applyDDLAnalyzerOutput(out *googlesql.AnalyzerOutput) error {
	stmt, err := out.ResolvedStatement()
	if err != nil {
		return err
	}
	switch stmt := stmt.(type) {
	case *googlesql.ResolvedCreateTableStmt:
		return c.addResolvedTable(stmt)
	case *googlesql.ResolvedCreateTableAsSelectStmt:
		return c.addResolvedTable(stmt)
	case *googlesql.ResolvedCreateExternalTableStmt:
		return c.addResolvedTable(stmt)
	case *googlesql.ResolvedCreateViewStmt:
		return c.addResolvedView(stmt)
	case *googlesql.ResolvedCreateSchemaStmt:
		return nil
	case *googlesql.ResolvedCreateIndexStmt:
		return nil
	case *googlesql.ResolvedDropStmt:
		return nil
	default:
		kind, _ := stmt.NodeKindString()
		if kind == "" {
			kind = fmt.Sprintf("%T", stmt)
		}
		return fmt.Errorf("unsupported BigQuery DDL statement %s", kind)
	}
}

type resolvedCreateTable interface {
	NamePath() ([]string, error)
	ColumnDefinitionList() ([]*googlesql.ResolvedColumnDefinition, error)
}

func (c *BigQueryGoogleSQLCatalog) addResolvedTable(stmt resolvedCreateTable) error {
	namePath, err := stmt.NamePath()
	if err != nil {
		return err
	}
	columns, err := stmt.ColumnDefinitionList()
	if err != nil {
		return err
	}
	return c.addTableFromColumnDefinitions(ObjectName{Parts: namePath}, columns)
}

func (c *BigQueryGoogleSQLCatalog) addResolvedView(stmt *googlesql.ResolvedCreateViewStmt) error {
	namePath, err := stmt.NamePath()
	if err != nil {
		return err
	}
	columns, err := stmt.ColumnDefinitionList()
	if err != nil {
		return err
	}
	return c.addTableFromColumnDefinitions(ObjectName{Parts: namePath}, columns)
}

func (c *BigQueryGoogleSQLCatalog) addTableFromColumnDefinitions(name ObjectName, columns []*googlesql.ResolvedColumnDefinition) error {
	tableName := name.String()
	parentCatalog, leafName, err := simpleCatalogForObjectName(c.SimpleCatalog, c.simpleCatalogs, name)
	if err != nil {
		return err
	}
	gsTable, err := googlesql.NewSimpleTable(leafName, 0)
	if err != nil {
		return err
	}
	if tableName != leafName {
		if err := gsTable.SetFullName(tableName); err != nil {
			return err
		}
	}
	if err := gsTable.SetAllowDuplicateColumnNames(true); err != nil {
		return err
	}
	if err := gsTable.SetAllowAnonymousColumnName(true); err != nil {
		return err
	}
	for _, column := range columns {
		columnName, err := column.Name()
		if err != nil {
			return err
		}
		columnType, err := column.Type()
		if err != nil {
			return err
		}
		hidden, err := column.IsHidden()
		if err != nil {
			return err
		}
		gsCol, err := googlesql.NewSimpleColumn(tableName, columnName, columnType, hidden, true)
		if err != nil {
			return err
		}
		if err := gsTable.AddColumn2(gsCol, false); err != nil {
			return err
		}
	}
	return parentCatalog.AddTable(gsTable)
}

// TableSchemaForStatement analyzes a query statement and returns a BigQuery
// REST TableSchema-shaped result schema.
func (a *BigQueryAnalyzer) TableSchemaForStatement(sql string) (*BigQueryTableSchema, error) {
	out, err := a.analyzeStatement(sql)
	if err != nil {
		return nil, err
	}
	return BigQueryTableSchemaFromAnalyzerOutput(out)
}

// TableSchemaForExpression analyzes a single expression and returns a
// one-field BigQuery TableSchema-shaped result schema.
func (a *BigQueryAnalyzer) TableSchemaForExpression(sql string) (*BigQueryTableSchema, error) {
	out, err := a.helper.AnalyzeExpression(sql)
	if err != nil {
		return nil, err
	}
	field, err := BigQueryTableFieldSchemaFromExpressionAnalyzerOutput("expression", out)
	if err != nil {
		return nil, err
	}
	return &BigQueryTableSchema{Fields: []*BigQueryTableFieldSchema{field}}, nil
}

// ParseDebugString returns the parser AST debug string.
func (a *BigQueryAnalyzer) ParseDebugString(sqlMode, sql string) (string, error) {
	if sqlMode == "query" {
		var err error
		sql, err = a.rewriteExternalQueries(sql)
		if err != nil {
			return "", err
		}
	}
	return a.helper.ParseDebugString(sqlMode, sql)
}

// Unparse returns SQL generated from the parser AST.
func (a *BigQueryAnalyzer) Unparse(sqlMode, sql string) (string, error) {
	if sqlMode == "query" {
		var err error
		sql, err = a.rewriteExternalQueries(sql)
		if err != nil {
			return "", err
		}
	}
	return a.helper.Unparse(sqlMode, sql)
}

// ResolvedASTDebugString returns the resolved AST debug string.
func (a *BigQueryAnalyzer) ResolvedASTDebugString(sqlMode, sql string) (string, error) {
	if sqlMode == "query" {
		var err error
		sql, err = a.rewriteExternalQueries(sql)
		if err != nil {
			return "", err
		}
	}
	return a.helper.ResolvedASTDebugString(sqlMode, sql)
}

func (a *BigQueryAnalyzer) analyzeStatement(sql string) (*googlesql.AnalyzerOutput, error) {
	rewritten, err := a.rewriteExternalQueries(sql)
	if err != nil {
		return nil, err
	}
	return a.helper.AnalyzeStatement(rewritten)
}

func (a *BigQueryAnalyzer) rewriteExternalQueries(sql string) (string, error) {
	if len(a.externalQueryAnalyzers) == 0 {
		return sql, nil
	}
	calls, err := findExternalQueryCalls(sql)
	if err != nil {
		return "", err
	}
	if len(calls) == 0 {
		return sql, nil
	}
	var b strings.Builder
	last := 0
	for _, call := range calls {
		b.WriteString(sql[last:call.start])
		replacement, err := a.externalQueryReplacement(call.args)
		if err != nil {
			return "", err
		}
		b.WriteString(replacement)
		last = call.end
	}
	b.WriteString(sql[last:])
	return b.String(), nil
}

func (a *BigQueryAnalyzer) externalQueryReplacement(args []string) (string, error) {
	if len(args) < 2 || len(args) > 3 {
		return "", fmt.Errorf("EXTERNAL_QUERY requires 2 or 3 arguments, got %d", len(args))
	}
	connectionID, err := decodeGoogleSQLStringLiteral(strings.TrimSpace(args[0]))
	if err != nil {
		return "", fmt.Errorf("EXTERNAL_QUERY connection argument: %w", err)
	}
	spannerSQL, err := decodeGoogleSQLStringLiteral(strings.TrimSpace(args[1]))
	if err != nil {
		return "", fmt.Errorf("EXTERNAL_QUERY SQL argument: %w", err)
	}
	analyzer, err := a.externalQueryAnalyzerForConnection(connectionID)
	if err != nil {
		return "", err
	}
	rowType, err := analyzer.RowTypeForStatement(spannerSQL)
	if err != nil {
		return "", fmt.Errorf("EXTERNAL_QUERY Spanner SQL: %w", err)
	}
	return bigQueryTypedEmptySubquery(rowType)
}

func (a *BigQueryAnalyzer) externalQueryAnalyzerForConnection(connectionID string) (*Analyzer, error) {
	if analyzer := a.externalQueryAnalyzers[connectionID]; analyzer != nil {
		return analyzer, nil
	}
	return nil, fmt.Errorf("no Spanner schema configured for EXTERNAL_QUERY connection %q", connectionID)
}

// BigQueryTableSchemaFromAnalyzerOutput converts a resolved query statement to
// a BigQuery REST TableSchema-shaped schema.
func BigQueryTableSchemaFromAnalyzerOutput(out *googlesql.AnalyzerOutput) (*BigQueryTableSchema, error) {
	stmt, err := out.ResolvedStatement()
	if err != nil {
		return nil, err
	}
	query, ok := stmt.(*googlesql.ResolvedQueryStmt)
	if !ok {
		return nil, ErrStatementHasNoRowType
	}
	return BigQueryTableSchemaFromResolvedQuery(query)
}

// BigQueryTableSchemaFromResolvedQuery converts a resolved query to a BigQuery
// REST TableSchema-shaped schema.
func BigQueryTableSchemaFromResolvedQuery(query *googlesql.ResolvedQueryStmt) (*BigQueryTableSchema, error) {
	n, err := query.OutputColumnListSize()
	if err != nil {
		return nil, err
	}
	fields := make([]*BigQueryTableFieldSchema, 0, n)
	for i := int32(0); i < n; i++ {
		outCol, err := query.OutputColumnList2(i)
		if err != nil {
			return nil, err
		}
		name, err := outCol.Name()
		if err != nil {
			return nil, err
		}
		resolvedCol, err := outCol.Column()
		if err != nil {
			return nil, err
		}
		typ, err := resolvedCol.Type()
		if err != nil {
			return nil, err
		}
		field, err := BigQueryTableFieldSchemaFromGoogleSQLType(name, typ)
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}
	return &BigQueryTableSchema{Fields: fields}, nil
}

// BigQueryTableFieldSchemaFromExpressionAnalyzerOutput converts a resolved
// expression to a BigQuery REST TableFieldSchema-shaped field.
func BigQueryTableFieldSchemaFromExpressionAnalyzerOutput(name string, out *googlesql.AnalyzerOutput) (*BigQueryTableFieldSchema, error) {
	expr, err := out.ResolvedExpr()
	if err != nil {
		return nil, err
	}
	typ, err := expr.Type()
	if err != nil {
		return nil, err
	}
	return BigQueryTableFieldSchemaFromGoogleSQLType(name, typ)
}

// BigQueryTableFieldSchemaFromGoogleSQLType converts a GoogleSQL frontend type
// to a BigQuery REST TableFieldSchema-shaped field.
func BigQueryTableFieldSchemaFromGoogleSQLType(name string, typ googlesql.Googlesql_TypeNode) (*BigQueryTableFieldSchema, error) {
	field, err := bigQueryTableFieldSchemaFromGoogleSQLType(typ)
	if err != nil {
		return nil, err
	}
	field.Name = name
	return field, nil
}

func bigQueryTableFieldSchemaFromGoogleSQLType(typ googlesql.Googlesql_TypeNode) (*BigQueryTableFieldSchema, error) {
	if typ == nil {
		return nil, fmt.Errorf("nil GoogleSQL type")
	}
	kind, err := typ.Kind()
	if err != nil {
		return nil, err
	}
	switch kind {
	case googlesql.TypeKindTypeBool:
		return nullableBigQueryField("BOOLEAN"), nil
	case googlesql.TypeKindTypeInt64:
		return nullableBigQueryField("INTEGER"), nil
	case googlesql.TypeKindTypeFloat, googlesql.TypeKindTypeDouble:
		return nullableBigQueryField("FLOAT"), nil
	case googlesql.TypeKindTypeString:
		return nullableBigQueryField("STRING"), nil
	case googlesql.TypeKindTypeBytes:
		return nullableBigQueryField("BYTES"), nil
	case googlesql.TypeKindTypeTimestamp:
		return nullableBigQueryField("TIMESTAMP"), nil
	case googlesql.TypeKindTypeDate:
		return nullableBigQueryField("DATE"), nil
	case googlesql.TypeKindTypeTime:
		return nullableBigQueryField("TIME"), nil
	case googlesql.TypeKindTypeDatetime:
		return nullableBigQueryField("DATETIME"), nil
	case googlesql.TypeKindTypeGeography:
		return nullableBigQueryField("GEOGRAPHY"), nil
	case googlesql.TypeKindTypeNumeric:
		return nullableBigQueryField("NUMERIC"), nil
	case googlesql.TypeKindTypeBignumeric:
		return nullableBigQueryField("BIGNUMERIC"), nil
	case googlesql.TypeKindTypeJson:
		return nullableBigQueryField("JSON"), nil
	case googlesql.TypeKindTypeArray:
		arrayType, err := typ.AsArray()
		if err != nil {
			return nil, err
		}
		elemType, err := arrayType.ElementType()
		if err != nil {
			return nil, err
		}
		field, err := bigQueryTableFieldSchemaFromGoogleSQLType(elemType)
		if err != nil {
			return nil, err
		}
		field.Mode = "REPEATED"
		return field, nil
	case googlesql.TypeKindTypeStruct:
		structType, err := typ.AsStruct()
		if err != nil {
			return nil, err
		}
		structFields, err := structType.Fields()
		if err != nil {
			return nil, err
		}
		fields := make([]*BigQueryTableFieldSchema, 0, len(structFields))
		for _, structField := range structFields {
			field, err := BigQueryTableFieldSchemaFromGoogleSQLType(structField.Name, structField.Type_)
			if err != nil {
				return nil, err
			}
			fields = append(fields, field)
		}
		return &BigQueryTableFieldSchema{Type: "RECORD", Mode: "NULLABLE", Fields: fields}, nil
	case googlesql.TypeKindTypeRange:
		rangeType, err := typ.AsRange()
		if err != nil {
			return nil, err
		}
		elemType, err := rangeType.ElementType()
		if err != nil {
			return nil, err
		}
		elemField, err := bigQueryTableFieldSchemaFromGoogleSQLType(elemType)
		if err != nil {
			return nil, err
		}
		if elemField.Type != "DATE" && elemField.Type != "DATETIME" && elemField.Type != "TIMESTAMP" {
			return nil, fmt.Errorf("unsupported BigQuery RANGE element type %s", elemField.Type)
		}
		return &BigQueryTableFieldSchema{
			Type: "RANGE",
			Mode: "NULLABLE",
			RangeElementType: &BigQueryFieldElementType{
				Type: elemField.Type,
			},
		}, nil
	default:
		debug, _ := typ.DebugString(false)
		return nil, fmt.Errorf("unsupported BigQuery TableSchema type kind %s (%s)", kind, debug)
	}
}

func nullableBigQueryField(typ string) *BigQueryTableFieldSchema {
	return &BigQueryTableFieldSchema{Type: typ, Mode: "NULLABLE"}
}

type externalQueryCall struct {
	start int
	end   int
	args  []string
}

func findExternalQueryCalls(sql string) ([]externalQueryCall, error) {
	var calls []externalQueryCall
	for i := 0; i < len(sql); {
		if next := skipGoogleSQLTrivia(sql, i); next != i {
			i = next
			continue
		}
		if sql[i] == '\'' || sql[i] == '"' {
			next, err := scanGoogleSQLString(sql, i)
			if err != nil {
				return nil, err
			}
			i = next
			continue
		}
		if !strings.HasPrefix(strings.ToUpper(sql[i:]), "EXTERNAL_QUERY") || !externalQueryBoundary(sql, i, i+len("EXTERNAL_QUERY")) {
			_, size := utf8.DecodeRuneInString(sql[i:])
			if size == 0 {
				size = 1
			}
			i += size
			continue
		}
		open := i + len("EXTERNAL_QUERY")
		for open < len(sql) && unicode.IsSpace(rune(sql[open])) {
			open++
		}
		if open >= len(sql) || sql[open] != '(' {
			i = open
			continue
		}
		close, args, err := parseExternalQueryArguments(sql, open)
		if err != nil {
			return nil, err
		}
		calls = append(calls, externalQueryCall{start: i, end: close + 1, args: args})
		i = close + 1
	}
	return calls, nil
}

func externalQueryBoundary(sql string, start, end int) bool {
	if start > 0 && isGoogleSQLIdentRune(rune(sql[start-1])) {
		return false
	}
	return end >= len(sql) || !isGoogleSQLIdentRune(rune(sql[end]))
}

func isGoogleSQLIdentRune(r rune) bool {
	return r == '_' || r == '$' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func parseExternalQueryArguments(sql string, open int) (int, []string, error) {
	depth := 1
	argStart := open + 1
	var args []string
	for i := open + 1; i < len(sql); {
		if next := skipGoogleSQLTrivia(sql, i); next != i {
			i = next
			continue
		}
		switch sql[i] {
		case '\'', '"':
			next, err := scanGoogleSQLString(sql, i)
			if err != nil {
				return 0, nil, err
			}
			i = next
			continue
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				args = append(args, strings.TrimSpace(sql[argStart:i]))
				return i, args, nil
			}
		case ',':
			if depth == 1 {
				args = append(args, strings.TrimSpace(sql[argStart:i]))
				argStart = i + 1
			}
		}
		i++
	}
	return 0, nil, fmt.Errorf("unterminated EXTERNAL_QUERY call")
}

func skipGoogleSQLTrivia(sql string, i int) int {
	if strings.HasPrefix(sql[i:], "--") {
		if end := strings.IndexByte(sql[i:], '\n'); end >= 0 {
			return i + end + 1
		}
		return len(sql)
	}
	if strings.HasPrefix(sql[i:], "/*") {
		if end := strings.Index(sql[i+2:], "*/"); end >= 0 {
			return i + 2 + end + 2
		}
		return len(sql)
	}
	return i
}

func scanGoogleSQLString(sql string, start int) (int, error) {
	quote := sql[start]
	triple := strings.HasPrefix(sql[start:], strings.Repeat(string(quote), 3))
	if triple {
		endToken := strings.Repeat(string(quote), 3)
		if end := strings.Index(sql[start+3:], endToken); end >= 0 {
			return start + 3 + end + 3, nil
		}
		return 0, fmt.Errorf("unterminated triple-quoted string literal")
	}
	for i := start + 1; i < len(sql); i++ {
		if sql[i] == '\\' {
			i++
			continue
		}
		if sql[i] == quote {
			if i+1 < len(sql) && sql[i+1] == quote {
				i++
				continue
			}
			return i + 1, nil
		}
	}
	return 0, fmt.Errorf("unterminated string literal")
}

func decodeGoogleSQLStringLiteral(lit string) (string, error) {
	lit = strings.TrimSpace(lit)
	if len(lit) == 0 {
		return "", fmt.Errorf("empty string literal")
	}
	raw := false
	if len(lit) >= 2 && (lit[0] == 'r' || lit[0] == 'R') && (lit[1] == '\'' || lit[1] == '"') {
		raw = true
		lit = lit[1:]
	}
	if strings.HasPrefix(lit, "'''") || strings.HasPrefix(lit, `"""`) {
		quote := lit[:3]
		if !strings.HasSuffix(lit, quote) || len(lit) < 6 {
			return "", fmt.Errorf("invalid triple-quoted string literal")
		}
		body := lit[3 : len(lit)-3]
		if raw {
			return body, nil
		}
		return unescapeGoogleSQLString(body), nil
	}
	if lit[0] != '\'' && lit[0] != '"' {
		return "", fmt.Errorf("want string literal, got %q", lit)
	}
	if len(lit) < 2 || lit[len(lit)-1] != lit[0] {
		return "", fmt.Errorf("invalid string literal")
	}
	body := lit[1 : len(lit)-1]
	if raw {
		return body, nil
	}
	return unescapeGoogleSQLString(body), nil
}

var googleSQLSimpleEscapes = map[byte]byte{
	'a':  '\a',
	'b':  '\b',
	'f':  '\f',
	'n':  '\n',
	'r':  '\r',
	't':  '\t',
	'v':  '\v',
	'\\': '\\',
	'?':  '?',
	'\'': '\'',
	'"':  '"',
	'`':  '`',
}

func unescapeGoogleSQLString(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' && i+1 < len(s) && s[i+1] == '\'' {
			b.WriteByte('\'')
			i++
			continue
		}
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		if repl, ok := googleSQLSimpleEscapes[s[i]]; ok {
			b.WriteByte(repl)
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func bigQueryTypedEmptySubquery(rowType *spannerpb.StructType) (string, error) {
	if rowType == nil {
		return "", fmt.Errorf("nil Spanner row type")
	}
	if len(rowType.Fields) == 0 {
		return "(SELECT 1 WHERE FALSE)", nil
	}
	cols := make([]string, 0, len(rowType.Fields))
	for _, field := range rowType.Fields {
		typ, err := bigQuerySQLTypeFromSpannerType(field.Type)
		if err != nil {
			return "", fmt.Errorf("field %s: %w", field.Name, err)
		}
		cols = append(cols, fmt.Sprintf("CAST(NULL AS %s) AS %s", typ, quoteBigQueryIdent(field.Name)))
	}
	return "(SELECT " + strings.Join(cols, ", ") + " LIMIT 0)", nil
}

func bigQuerySQLTypeFromSpannerType(t *spannerpb.Type) (string, error) {
	if t == nil {
		return "", fmt.Errorf("nil Spanner type")
	}
	switch t.Code {
	case spannerpb.TypeCode_BOOL:
		return "BOOL", nil
	case spannerpb.TypeCode_INT64:
		return "INT64", nil
	case spannerpb.TypeCode_FLOAT32, spannerpb.TypeCode_FLOAT64:
		return "FLOAT64", nil
	case spannerpb.TypeCode_TIMESTAMP:
		return "TIMESTAMP", nil
	case spannerpb.TypeCode_DATE:
		return "DATE", nil
	case spannerpb.TypeCode_STRING:
		return "STRING", nil
	case spannerpb.TypeCode_BYTES:
		return "BYTES", nil
	case spannerpb.TypeCode_NUMERIC:
		return "NUMERIC", nil
	case spannerpb.TypeCode_JSON:
		return "JSON", nil
	case spannerpb.TypeCode_ARRAY:
		elem, err := bigQuerySQLTypeFromSpannerType(t.ArrayElementType)
		if err != nil {
			return "", err
		}
		return "ARRAY<" + elem + ">", nil
	case spannerpb.TypeCode_STRUCT:
		fields := t.GetStructType().GetFields()
		parts := make([]string, 0, len(fields))
		for _, field := range fields {
			fieldType, err := bigQuerySQLTypeFromSpannerType(field.Type)
			if err != nil {
				return "", fmt.Errorf("struct field %s: %w", field.Name, err)
			}
			parts = append(parts, quoteBigQueryIdent(field.Name)+" "+fieldType)
		}
		return "STRUCT<" + strings.Join(parts, ", ") + ">", nil
	default:
		return "", fmt.Errorf("unsupported Spanner type %s for BigQuery EXTERNAL_QUERY rewrite", t.Code)
	}
}

var simpleBigQueryIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func quoteBigQueryIdent(name string) string {
	if simpleBigQueryIdentifier.MatchString(name) {
		return name
	}
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}
