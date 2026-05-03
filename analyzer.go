package spanalyzer

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	googlesql "github.com/goccy/go-googlesql"
)

var ErrStatementHasNoRowType = errors.New("statement has no row type")

var initGoogleSQLOnce sync.Once
var initGoogleSQLErr error

func InitGoogleSQL() error {
	initGoogleSQLOnce.Do(func() {
		cacheDir, err := googleSQLCompilationCacheDir()
		if err != nil {
			initGoogleSQLErr = err
			return
		}
		initGoogleSQLErr = googlesql.Init(
			googlesql.WithCompilationMode(googlesql.CompilationModeCompiler),
			googlesql.WithCompilationCache(cacheDir),
		)
	})
	return initGoogleSQLErr
}

func googleSQLCompilationCacheDir() (string, error) {
	if dir := os.Getenv("SPANNER_ANALYZER_GOOGLESQL_CACHE_DIR"); dir != "" {
		return dir, nil
	}
	const cacheSubdir = "spanner-analyzer/go-googlesql"
	if dir, err := os.UserCacheDir(); err == nil {
		cacheDir := filepath.Join(dir, cacheSubdir)
		if err := os.MkdirAll(cacheDir, 0o755); err == nil {
			return cacheDir, nil
		}
	}
	cacheDir := filepath.Join(os.TempDir(), cacheSubdir)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("create GoogleSQL compilation cache: %w", err)
	}
	return cacheDir, nil
}

type Analyzer struct {
	catalog     *Catalog
	gsCatalog   *googlesql.SimpleCatalog
	opts        *googlesql.AnalyzerOptions
	typeFactory *googlesql.TypeFactory
}

func NewAnalyzer(schema *Catalog) (*Analyzer, error) {
	return NewAnalyzerWithOptions(schema)
}

func NewAnalyzerWithOptions(schema *Catalog, options ...AnalyzerOption) (*Analyzer, error) {
	if schema == nil {
		return nil, fmt.Errorf("nil schema catalog")
	}
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
	gsCatalog, opts, err := newGoogleSQLAnalyzerObjects(tf, config)
	if err != nil {
		return nil, err
	}
	a := &Analyzer{
		catalog:     schema,
		gsCatalog:   gsCatalog,
		opts:        opts,
		typeFactory: tf,
	}
	if err := a.addTablesToGoogleSQLCatalog(); err != nil {
		return nil, err
	}
	if err := a.addSequencesToGoogleSQLCatalog(); err != nil {
		return nil, err
	}
	if err := a.addModelsToGoogleSQLCatalog(); err != nil {
		return nil, err
	}
	if err := a.addSpannerFunctionsToGoogleSQLCatalog(); err != nil {
		return nil, err
	}
	if err := a.addViewsToGoogleSQLCatalog(); err != nil {
		return nil, err
	}
	if err := a.addPropertyGraphsToGoogleSQLCatalog(); err != nil {
		return nil, err
	}
	return a, nil
}

func NewAnalyzerFromDDL(path, ddlSQL string) (*Analyzer, error) {
	return NewAnalyzerFromDDLWithOptions(path, ddlSQL)
}

func NewAnalyzerFromDDLWithOptions(path, ddlSQL string, options ...AnalyzerOption) (*Analyzer, error) {
	catalog, err := BuildSchemaCatalog(path, ddlSQL)
	if err != nil {
		return nil, err
	}
	return NewAnalyzerWithOptions(catalog, options...)
}

func NewAnalyzerFromDDLWithProtoDescriptorFiles(path, ddlSQL string, protoDescriptorPaths []string) (*Analyzer, error) {
	return NewAnalyzerFromDDLWithProtoDescriptorFilesAndOptions(path, ddlSQL, protoDescriptorPaths)
}

func NewAnalyzerFromDDLWithProtoDescriptorFilesAndOptions(path, ddlSQL string, protoDescriptorPaths []string, options ...AnalyzerOption) (*Analyzer, error) {
	catalog, err := BuildSchemaCatalog(path, ddlSQL)
	if err != nil {
		return nil, err
	}
	if err := catalog.LoadProtoDescriptorSetFiles(protoDescriptorPaths); err != nil {
		return nil, err
	}
	return NewAnalyzerWithOptions(catalog, options...)
}

func (a *Analyzer) RowTypeForStatement(sql string) (*spannerpb.StructType, error) {
	sql = normalizeSpannerFunctionNamedArgs(sql)
	out, err := googlesql.AnalyzeStatement(sql, a.opts, a.gsCatalog, a.typeFactory)
	if err != nil {
		return nil, err
	}
	stmt, err := out.ResolvedStatement()
	if err != nil {
		return nil, err
	}
	query, ok := stmt.(*googlesql.ResolvedQueryStmt)
	if !ok {
		return nil, ErrStatementHasNoRowType
	}
	return a.rowTypeFromResolvedQuery(query)
}

func (a *Analyzer) RowTypeForExpression(sql string) (*spannerpb.StructType, error) {
	typ, err := a.TypeForExpression(sql)
	if err != nil {
		return nil, err
	}
	return &spannerpb.StructType{
		Fields: []*spannerpb.StructType_Field{
			{Name: "expression", Type: typ},
		},
	}, nil
}

func (a *Analyzer) TypeForExpression(sql string) (*spannerpb.Type, error) {
	sql = normalizeSpannerFunctionNamedArgs(sql)
	out, err := googlesql.AnalyzeExpression(sql, a.opts, a.gsCatalog, a.typeFactory)
	if err != nil {
		return nil, err
	}
	expr, err := out.ResolvedExpr()
	if err != nil {
		return nil, err
	}
	typ, err := expr.Type()
	if err != nil {
		return nil, err
	}
	return googleSQLTypeToSpannerPB(typ)
}

func (a *Analyzer) ParseDebugString(sqlMode, sql string) (string, error) {
	out, err := a.parse(sqlMode, sql)
	if err != nil {
		return "", err
	}
	node, err := out.Node()
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := writeASTDebugString(&buf, node, 0); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (a *Analyzer) Unparse(sqlMode, sql string) (string, error) {
	out, err := a.parse(sqlMode, sql)
	if err != nil {
		return "", err
	}
	node, err := out.Node()
	if err != nil {
		return "", err
	}
	return googlesql.Unparse(node)
}

func (a *Analyzer) ResolvedASTDebugString(sqlMode, sql string) (string, error) {
	sql = normalizeSpannerFunctionNamedArgs(sql)
	var out *googlesql.AnalyzerOutput
	var err error
	switch sqlMode {
	case "query":
		out, err = googlesql.AnalyzeStatement(sql, a.opts, a.gsCatalog, a.typeFactory)
	case "expression":
		out, err = googlesql.AnalyzeExpression(sql, a.opts, a.gsCatalog, a.typeFactory)
	default:
		return "", fmt.Errorf("unsupported sql mode %q", sqlMode)
	}
	if err != nil {
		return "", err
	}
	node, err := out.ResolvedNode()
	if err != nil {
		return "", err
	}
	return node.DebugString()
}

func (a *Analyzer) parse(sqlMode, sql string) (*googlesql.ParserOutput, error) {
	parserOptions, err := a.opts.GetParserOptions()
	if err != nil {
		return nil, err
	}
	switch sqlMode {
	case "query":
		return googlesql.ParseStatement(sql, parserOptions)
	case "expression":
		return googlesql.ParseExpression(sql, parserOptions)
	default:
		return nil, fmt.Errorf("unsupported sql mode %q", sqlMode)
	}
}

func writeASTDebugString(buf *bytes.Buffer, node googlesql.ASTNode, depth int) error {
	if node == nil {
		return nil
	}
	for i := 0; i < depth; i++ {
		buf.WriteString("  ")
	}
	line, err := node.SingleNodeDebugString()
	if err != nil {
		return err
	}
	buf.WriteString(line)
	buf.WriteByte('\n')
	n, err := node.NumChildren()
	if err != nil {
		return err
	}
	for i := int32(0); i < n; i++ {
		child, err := node.Child(i)
		if err != nil {
			return err
		}
		if err := writeASTDebugString(buf, child, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func (a *Analyzer) AddQueryParameter(name string, spec *TypeSpec) error {
	typ, err := a.typeSpecToGoogleSQLType(spec)
	if err != nil {
		return err
	}
	if err := a.opts.SetParameterMode(googlesql.ParameterModeParameterNamed); err != nil {
		return err
	}
	return a.opts.AddQueryParameter(name, typ)
}

func (a *Analyzer) AddPositionalQueryParameter(spec *TypeSpec) error {
	typ, err := a.typeSpecToGoogleSQLType(spec)
	if err != nil {
		return err
	}
	if err := a.opts.SetParameterMode(googlesql.ParameterModeParameterPositional); err != nil {
		return err
	}
	return a.opts.AddPositionalQueryParameter(typ)
}

func newGoogleSQLAnalyzerObjects(tf *googlesql.TypeFactory, config analyzerConfig) (*googlesql.SimpleCatalog, *googlesql.AnalyzerOptions, error) {
	catalog, err := googlesql.NewSimpleCatalog("spanner", tf)
	if err != nil {
		return nil, nil, err
	}
	lang, err := googlesql.NewLanguageOptions()
	if err != nil {
		return nil, nil, err
	}
	if err := lang.EnableMaximumLanguageFeaturesForDevelopment(); err != nil {
		return nil, nil, err
	}
	if err := lang.SetSupportsAllStatementKinds(); err != nil {
		return nil, nil, err
	}
	if config.productMode != nil {
		if err := lang.SetProductMode(*config.productMode); err != nil {
			return nil, nil, err
		}
	}
	if config.strictNameResolution {
		if err := lang.SetNameResolutionMode(googlesql.NameResolutionModeNameResolutionStrict); err != nil {
			return nil, nil, err
		}
	}
	if err := catalog.AddBuiltinFunctionsAndTypes(&googlesql.BuiltinFunctionOptions{LanguageOptions: lang}); err != nil {
		return nil, nil, err
	}
	opts, err := googlesql.NewAnalyzerOptions2()
	if err != nil {
		return nil, nil, err
	}
	if err := opts.SetLanguage(lang); err != nil {
		return nil, nil, err
	}
	if config.foldLiteralCast != nil {
		if err := opts.SetFoldLiteralCast(*config.foldLiteralCast); err != nil {
			return nil, nil, err
		}
	}
	if config.pruneUnusedColumns != nil {
		if err := opts.SetPruneUnusedColumns(*config.pruneUnusedColumns); err != nil {
			return nil, nil, err
		}
	}
	if config.parseLocationRecordType != nil {
		if err := opts.SetParseLocationRecordType(*config.parseLocationRecordType); err != nil {
			return nil, nil, err
		}
	}
	return catalog, opts, nil
}

func (a *Analyzer) addTablesToGoogleSQLCatalog() error {
	for _, table := range a.catalog.Tables {
		if err := a.addTableToGoogleSQLCatalog(table); err != nil {
			return err
		}
	}
	return nil
}

func (a *Analyzer) addTableToGoogleSQLCatalog(table *Table) error {
	tableName := table.Name.String()
	gsTable, err := googlesql.NewSimpleTable(tableName, 0)
	if err != nil {
		return err
	}
	if err := gsTable.SetAllowDuplicateColumnNames(true); err != nil {
		return err
	}
	if err := gsTable.SetAllowAnonymousColumnName(true); err != nil {
		return err
	}
	primaryKey := make([]int32, 0, len(table.PrimaryKey))
	for i, col := range table.Columns {
		gsType, err := a.typeSpecToGoogleSQLType(col.Type)
		if err != nil {
			return fmt.Errorf("column %s.%s: %w", tableName, col.Name, err)
		}
		gsCol, err := googlesql.NewSimpleColumn(tableName, col.Name, gsType, col.Hidden, true)
		if err != nil {
			return err
		}
		if err := gsTable.AddColumn2(gsCol, false); err != nil {
			return err
		}
		if col.PrimaryKey {
			primaryKey = append(primaryKey, int32(i))
		}
	}
	if len(primaryKey) > 0 {
		if err := gsTable.SetPrimaryKey(primaryKey); err != nil {
			return err
		}
	}
	if err := a.gsCatalog.AddTable(gsTable); err != nil {
		return err
	}
	for _, synonym := range table.Synonyms {
		if err := a.gsCatalog.AddTable2(synonym, gsTable); err != nil {
			return err
		}
	}
	return nil
}

func (a *Analyzer) addViewsToGoogleSQLCatalog() error {
	pending := a.orderedViews()
	done := map[string]bool{}
	var lastErr error
	for len(done) < len(pending) {
		progress := false
		for _, view := range pending {
			key := view.Name.String()
			if done[key] {
				continue
			}
			if err := a.addViewToGoogleSQLCatalog(view); err != nil {
				lastErr = err
				continue
			}
			done[key] = true
			progress = true
		}
		if !progress {
			if lastErr != nil {
				return lastErr
			}
			return fmt.Errorf("could not resolve view dependencies")
		}
	}
	return nil
}

func (a *Analyzer) orderedViews() []*View {
	keys := append([]string(nil), a.catalog.ViewOrder...)
	if len(keys) == 0 && len(a.catalog.Views) > 0 {
		for key := range a.catalog.Views {
			keys = append(keys, key)
		}
		sort.Strings(keys)
	}
	views := make([]*View, 0, len(keys))
	seen := map[string]bool{}
	for _, key := range keys {
		if seen[key] {
			continue
		}
		seen[key] = true
		if view := a.catalog.Views[key]; view != nil {
			views = append(views, view)
		}
	}
	return views
}

func (a *Analyzer) addViewToGoogleSQLCatalog(view *View) error {
	rowType, err := a.RowTypeForStatement(view.Query)
	if err != nil {
		return fmt.Errorf("analyze view %s: %w", view.Name, err)
	}
	table := &Table{Name: view.Name}
	for _, field := range rowType.Fields {
		spec, err := typeSpecFromSpannerPB(field.Type)
		if err != nil {
			return fmt.Errorf("view %s field %s: %w", view.Name, field.Name, err)
		}
		table.Columns = append(table.Columns, &Column{Name: field.Name, Type: spec})
	}
	return a.addTableToGoogleSQLCatalog(table)
}

func (a *Analyzer) addPropertyGraphsToGoogleSQLCatalog() error {
	names := make([]string, 0, len(a.catalog.PropertyGraphs))
	for name := range a.catalog.PropertyGraphs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		graph, err := googlesql.NewSimplePropertyGraph([]string{name})
		if err != nil {
			return err
		}
		if err := a.gsCatalog.AddPropertyGraph(graph); err != nil {
			return err
		}
	}
	return nil
}

func (a *Analyzer) addSequencesToGoogleSQLCatalog() error {
	names := make([]string, 0, len(a.catalog.Sequences))
	for name := range a.catalog.Sequences {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		sequence, err := googlesql.NewSimpleSequence(name)
		if err != nil {
			return err
		}
		if err := a.gsCatalog.AddSequence(sequence); err != nil {
			return err
		}
	}
	return nil
}

func (a *Analyzer) rowTypeFromResolvedQuery(query *googlesql.ResolvedQueryStmt) (*spannerpb.StructType, error) {
	n, err := query.OutputColumnListSize()
	if err != nil {
		return nil, err
	}
	fields := make([]*spannerpb.StructType_Field, 0, n)
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
		gsType, err := resolvedCol.Type()
		if err != nil {
			return nil, err
		}
		pbType, ok, err := a.directProtoColumnType(resolvedCol)
		if err != nil {
			return nil, err
		}
		if !ok {
			pbType, err = googleSQLTypeToSpannerPB(gsType)
		}
		if err != nil {
			return nil, err
		}
		fields = append(fields, &spannerpb.StructType_Field{Name: name, Type: pbType})
	}
	return &spannerpb.StructType{Fields: fields}, nil
}

func (a *Analyzer) directProtoColumnType(col *googlesql.ResolvedColumn) (*spannerpb.Type, bool, error) {
	tableName, err := col.TableName()
	if err != nil {
		return nil, false, err
	}
	columnName, err := col.Name()
	if err != nil {
		return nil, false, err
	}
	table := a.catalog.Tables[tableName]
	if table == nil {
		return nil, false, nil
	}
	c, _ := table.Column(columnName)
	if c == nil || c.Type == nil {
		return nil, false, nil
	}
	switch c.Type.Code {
	case spannerpb.TypeCode_PROTO, spannerpb.TypeCode_ENUM:
		t, err := c.Type.SpannerPB()
		return t, err == nil, err
	default:
		return nil, false, nil
	}
}

func typeSpecFromSpannerPB(t *spannerpb.Type) (*TypeSpec, error) {
	if t == nil {
		return nil, fmt.Errorf("nil Spanner protobuf type")
	}
	switch t.Code {
	case spannerpb.TypeCode_ARRAY:
		elem, err := typeSpecFromSpannerPB(t.ArrayElementType)
		if err != nil {
			return nil, err
		}
		return &TypeSpec{Code: spannerpb.TypeCode_ARRAY, ArrayElement: elem}, nil
	case spannerpb.TypeCode_STRUCT:
		fields := make([]StructField, 0, len(t.GetStructType().GetFields()))
		for _, field := range t.GetStructType().GetFields() {
			spec, err := typeSpecFromSpannerPB(field.Type)
			if err != nil {
				return nil, err
			}
			fields = append(fields, StructField{Name: field.Name, Type: spec})
		}
		return &TypeSpec{Code: spannerpb.TypeCode_STRUCT, StructFields: fields}, nil
	case spannerpb.TypeCode_PROTO, spannerpb.TypeCode_ENUM:
		return &TypeSpec{Code: t.Code, ProtoTypeFQN: t.ProtoTypeFqn}, nil
	default:
		return &TypeSpec{Code: t.Code}, nil
	}
}
