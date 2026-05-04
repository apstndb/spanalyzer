package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	spanalyzer "github.com/apstndb/go-googlesql-spanner-poc"
	googlesql "github.com/goccy/go-googlesql"
	"github.com/goccy/go-yaml"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func main() {
	ddlPath := flag.String("ddl", "", "optional path to a dialect-specific GoogleSQL DDL file")
	sql := flag.String("sql", "", "SQL query or expression to analyze")
	dialect := flag.String("dialect", "spanner", "SQL dialect/catalog mode: spanner or bigquery")
	mode := flag.String("mode", "", "comma-separated modes: spanner_type, bigquery_type, parse, analyze, unparse")
	sqlMode := flag.String("sql-mode", "query", "how to interpret --sql: query or expression")
	output := flag.String("output", defaultOutputFormat, "output format: yaml, json, or textproto")
	productMode := flag.String("product-mode", "external", "GoogleSQL product mode: internal or external")
	strictNameResolution := flag.Bool("strict-name-resolution", false, "enable strict name resolution")
	foldLiteralCast := flag.Bool("fold-literal-cast", true, "set AnalyzerOptions fold_literal_cast")
	pruneUnusedColumns := flag.Bool("prune-unused-columns", true, "set AnalyzerOptions prune_unused_columns")
	parseLocationRecordType := flag.String("parse-location-record-type", "none", "parse location recording: none, full_node_scope, or code_search")
	enableMaximumDevelopmentLanguageFeatures := flag.Bool("enable-maximum-development-language-features", false, "use raw GoogleSQL EnableMaximumLanguageFeaturesForDevelopment without the dialect feature blacklist")
	var protoDescriptorFiles repeatedStringFlag
	var externalDDLs repeatedStringFlag
	var externalProtoDescriptorFiles repeatedStringFlag
	var params repeatedStringFlag
	var positionalParams repeatedStringFlag
	flag.Var(&protoDescriptorFiles, "proto-descriptors-file", "path to a FileDescriptorSet used by CREATE/ALTER PROTO BUNDLE; repeatable")
	flag.Var(&externalDDLs, "external-ddl", "Spanner DDL path for a BigQuery EXTERNAL_QUERY connection as CONNECTION=PATH; repeatable")
	flag.Var(&externalProtoDescriptorFiles, "external-proto-descriptors-file", "FileDescriptorSet for a BigQuery EXTERNAL_QUERY connection as CONNECTION=PATH; repeatable")
	flag.Var(&params, "param", "named query parameter type as name=TYPE; repeatable")
	flag.Var(&positionalParams, "positional-param", "positional query parameter TYPE; repeatable")
	flag.Parse()

	if *sql == "" {
		flag.Usage()
		os.Exit(2)
	}

	ddlPathForAnalyzer, ddl, err := readDDL(*ddlPath)
	if err != nil {
		exitErr(err)
	}
	externalSchemas, err := readExternalSchemas(externalDDLs, externalProtoDescriptorFiles)
	if err != nil {
		exitErr(err)
	}
	options, err := analyzerOptionsFromFlags(*productMode, *strictNameResolution, *foldLiteralCast, *pruneUnusedColumns, *parseLocationRecordType, *enableMaximumDevelopmentLanguageFeatures)
	if err != nil {
		exitErr(err)
	}
	result, err := runDialect(
		strings.ToLower(*dialect),
		ddlPathForAnalyzer,
		ddl,
		*sql,
		splitModes(*mode),
		strings.ToLower(*sqlMode),
		*output,
		externalSchemas,
		protoDescriptorFiles,
		params,
		positionalParams,
		options,
	)
	if err != nil {
		exitErr(err)
	}
	fmt.Print(result)
}

const defaultOutputFormat = "yaml"

func readDDL(path string) (string, string, error) {
	if path == "" {
		return "schema.sql", "", nil
	}
	ddl, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	return path, string(ddl), nil
}

type anyProto interface {
	ProtoReflect() protoreflect.Message
}

type externalSchema struct {
	DDLPath              string
	DDL                  string
	ProtoDescriptorFiles []string
}

func runDialect(dialect, ddlPath, ddl, sql string, modes []string, sqlMode, output string, externalSchemas map[string]externalSchema, protoDescriptorFiles, params, positionalParams []string, options []spanalyzer.AnalyzerOption) (string, error) {
	switch dialect {
	case "spanner":
		if len(externalSchemas) > 0 {
			return "", fmt.Errorf("--external-ddl is only supported with --dialect=bigquery")
		}
		analyzer, err := spanalyzer.NewAnalyzerFromDDLWithProtoDescriptorFilesAndOptions(ddlPath, ddl, protoDescriptorFiles, options...)
		if err != nil {
			return "", err
		}
		if err := addQueryParameters(analyzer, params, positionalParams); err != nil {
			return "", err
		}
		return runModes(analyzer, modes, sqlMode, sql, output)
	case "bigquery":
		if len(params) > 0 || len(positionalParams) > 0 {
			return "", fmt.Errorf("--param and --positional-param are not supported with --dialect=bigquery yet")
		}
		analyzer, err := spanalyzer.NewBigQueryAnalyzerFromDDL(ddlPath, ddl, options...)
		if err != nil {
			return "", err
		}
		if len(protoDescriptorFiles) > 0 {
			return "", fmt.Errorf("--proto-descriptors-file is only supported with --dialect=spanner; use --external-proto-descriptors-file for BigQuery EXTERNAL_QUERY connections")
		}
		externalAnalyzers, err := buildExternalQueryAnalyzers(externalSchemas, options)
		if err != nil {
			return "", err
		}
		analyzer.SetExternalQueryAnalyzers(externalAnalyzers)
		return runBigQueryModes(analyzer, modes, sqlMode, sql, output)
	default:
		return "", fmt.Errorf("unsupported --dialect %q", dialect)
	}
}

func readExternalSchemas(ddlValues, protoValues []string) (map[string]externalSchema, error) {
	schemas := map[string]externalSchema{}
	for _, value := range ddlValues {
		connectionID, path, err := parseConnectionFileFlag("--external-ddl", value)
		if err != nil {
			return nil, err
		}
		if _, ok := schemas[connectionID]; ok {
			return nil, fmt.Errorf("duplicate --external-ddl for connection %q", connectionID)
		}
		ddlPath, ddl, err := readDDL(path)
		if err != nil {
			return nil, fmt.Errorf("--external-ddl %s: %w", connectionID, err)
		}
		schemas[connectionID] = externalSchema{DDLPath: ddlPath, DDL: ddl}
	}
	for _, value := range protoValues {
		connectionID, path, err := parseConnectionFileFlag("--external-proto-descriptors-file", value)
		if err != nil {
			return nil, err
		}
		schema := schemas[connectionID]
		schema.ProtoDescriptorFiles = append(schema.ProtoDescriptorFiles, path)
		schemas[connectionID] = schema
	}
	for connectionID, schema := range schemas {
		if schema.DDLPath == "" {
			return nil, fmt.Errorf("--external-proto-descriptors-file for connection %q requires --external-ddl", connectionID)
		}
	}
	return schemas, nil
}

func parseConnectionFileFlag(flagName, value string) (string, string, error) {
	connectionID, path, ok := strings.Cut(value, "=")
	if !ok || connectionID == "" || path == "" {
		return "", "", fmt.Errorf("invalid %s %q, want CONNECTION=PATH", flagName, value)
	}
	return connectionID, path, nil
}

func buildExternalQueryAnalyzers(schemas map[string]externalSchema, options []spanalyzer.AnalyzerOption) (map[string]*spanalyzer.Analyzer, error) {
	analyzers := make(map[string]*spanalyzer.Analyzer, len(schemas))
	for connectionID, schema := range schemas {
		analyzer, err := spanalyzer.NewAnalyzerFromDDLWithProtoDescriptorFilesAndOptions(schema.DDLPath, schema.DDL, schema.ProtoDescriptorFiles, options...)
		if err != nil {
			return nil, fmt.Errorf("EXTERNAL_QUERY connection %s: %w", connectionID, err)
		}
		analyzers[connectionID] = analyzer
	}
	return analyzers, nil
}

func marshalProto(message anyProto, output string) ([]byte, error) {
	switch strings.ToLower(output) {
	case "json":
		return marshalProtoJSON(message)
	case "yaml":
		jsonBytes, err := marshalProtoJSON(message)
		if err != nil {
			return nil, err
		}
		return yaml.JSONToYAML(jsonBytes)
	case "textproto":
		return prototext.MarshalOptions{
			Multiline: true,
			Indent:    "  ",
		}.Marshal(message)
	default:
		return nil, fmt.Errorf("unsupported --output %q", output)
	}
}

func marshalProtoJSON(message anyProto) ([]byte, error) {
	return protojson.MarshalOptions{
		Multiline:     true,
		Indent:        "  ",
		UseProtoNames: true,
	}.Marshal(message)
}

func marshalJSONYAML(value any, output string) ([]byte, error) {
	jsonBytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(output) {
	case "json":
		return jsonBytes, nil
	case "yaml":
		return yaml.JSONToYAML(jsonBytes)
	default:
		return nil, fmt.Errorf("unsupported --output %q for BigQuery TableSchema", output)
	}
}

func runModes(analyzer *spanalyzer.Analyzer, modes []string, sqlMode, sql, output string) (string, error) {
	if len(modes) == 0 {
		modes = []string{"spanner_type"}
	}
	sections := make([]modeResult, 0, len(modes))
	for _, mode := range modes {
		body, err := runMode(analyzer, mode, sqlMode, sql, output)
		if err != nil {
			return "", err
		}
		sections = append(sections, modeResult{name: mode, body: body})
	}
	if len(sections) == 1 {
		return sections[0].body, nil
	}
	var b strings.Builder
	for i, section := range sections {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("== ")
		b.WriteString(section.name)
		b.WriteString(" ==\n")
		b.WriteString(section.body)
		if !strings.HasSuffix(section.body, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String(), nil
}

func runBigQueryModes(analyzer *spanalyzer.BigQueryAnalyzer, modes []string, sqlMode, sql, output string) (string, error) {
	if len(modes) == 0 {
		modes = []string{"bigquery_type"}
	}
	sections := make([]modeResult, 0, len(modes))
	for _, mode := range modes {
		body, err := runBigQueryMode(analyzer, mode, sqlMode, sql, output)
		if err != nil {
			return "", err
		}
		sections = append(sections, modeResult{name: mode, body: body})
	}
	if len(sections) == 1 {
		return sections[0].body, nil
	}
	var b strings.Builder
	for i, section := range sections {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("== ")
		b.WriteString(section.name)
		b.WriteString(" ==\n")
		b.WriteString(section.body)
		if !strings.HasSuffix(section.body, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String(), nil
}

type modeResult struct {
	name string
	body string
}

func runMode(analyzer *spanalyzer.Analyzer, mode, sqlMode, sql, output string) (string, error) {
	switch mode {
	case "spanner_type", "rowtype":
		message, err := spannerTypeProtoForSQLMode(analyzer, sqlMode, sql)
		if err != nil {
			return "", err
		}
		out, err := marshalProto(message, output)
		if err != nil {
			return "", err
		}
		return string(out) + "\n", nil
	case "parse":
		return analyzer.ParseDebugString(sqlMode, sql)
	case "unparse":
		unparsed, err := analyzer.Unparse(sqlMode, sql)
		if err != nil {
			return "", err
		}
		return unparsed + "\n", nil
	case "analyze", "resolved_ast":
		return analyzer.ResolvedASTDebugString(sqlMode, sql)
	default:
		return "", fmt.Errorf("unsupported --mode %q", mode)
	}
}

func runBigQueryMode(analyzer *spanalyzer.BigQueryAnalyzer, mode, sqlMode, sql, output string) (string, error) {
	switch mode {
	case "bigquery_type":
		schema, err := bigQueryTableSchemaForSQLMode(analyzer, sqlMode, sql)
		if err != nil {
			return "", err
		}
		out, err := marshalJSONYAML(schema, output)
		if err != nil {
			return "", err
		}
		return string(out) + "\n", nil
	case "parse":
		return analyzer.ParseDebugString(sqlMode, sql)
	case "unparse":
		unparsed, err := analyzer.Unparse(sqlMode, sql)
		if err != nil {
			return "", err
		}
		return unparsed + "\n", nil
	case "analyze", "resolved_ast":
		return analyzer.ResolvedASTDebugString(sqlMode, sql)
	default:
		return "", fmt.Errorf("unsupported BigQuery --mode %q", mode)
	}
}

func spannerTypeProtoForSQLMode(analyzer *spanalyzer.Analyzer, sqlMode, sql string) (anyProto, error) {
	switch sqlMode {
	case "query":
		return analyzer.RowTypeForStatement(sql)
	case "expression":
		return analyzer.TypeForExpression(sql)
	default:
		return nil, fmt.Errorf("unsupported --sql-mode %q", sqlMode)
	}
}

func bigQueryTableSchemaForSQLMode(analyzer *spanalyzer.BigQueryAnalyzer, sqlMode, sql string) (*spanalyzer.BigQueryTableSchema, error) {
	switch sqlMode {
	case "query":
		return analyzer.TableSchemaForStatement(sql)
	case "expression":
		return analyzer.TableSchemaForExpression(sql)
	default:
		return nil, fmt.Errorf("unsupported --sql-mode %q", sqlMode)
	}
}

func splitModes(value string) []string {
	parts := strings.Split(value, ",")
	modes := make([]string, 0, len(parts))
	for _, part := range parts {
		mode := strings.ToLower(strings.TrimSpace(part))
		if mode != "" {
			modes = append(modes, mode)
		}
	}
	return modes
}

func analyzerOptionsFromFlags(productMode string, strictNameResolution, foldLiteralCast, pruneUnusedColumns bool, parseLocationRecordType string, enableMaximumDevelopmentLanguageFeatures bool) ([]spanalyzer.AnalyzerOption, error) {
	options := []spanalyzer.AnalyzerOption{
		spanalyzer.WithStrictNameResolution(strictNameResolution),
		spanalyzer.WithFoldLiteralCast(foldLiteralCast),
		spanalyzer.WithPruneUnusedColumns(pruneUnusedColumns),
		spanalyzer.WithMaximumDevelopmentLanguageFeatures(enableMaximumDevelopmentLanguageFeatures),
	}
	if productMode != "" {
		mode, err := parseProductMode(productMode)
		if err != nil {
			return nil, err
		}
		options = append(options, spanalyzer.WithProductMode(mode))
	}
	recordType, err := parseLocationRecordTypeValue(parseLocationRecordType)
	if err != nil {
		return nil, err
	}
	options = append(options, spanalyzer.WithParseLocationRecordType(recordType))
	return options, nil
}

func parseProductMode(value string) (googlesql.ProductMode, error) {
	switch strings.ToLower(value) {
	case "internal":
		return googlesql.ProductModeProductInternal, nil
	case "external":
		return googlesql.ProductModeProductExternal, nil
	default:
		return 0, fmt.Errorf("unsupported --product-mode %q", value)
	}
}

func parseLocationRecordTypeValue(value string) (googlesql.ParseLocationRecordType, error) {
	switch strings.ToLower(value) {
	case "none":
		return googlesql.ParseLocationRecordTypeParseLocationRecordNone, nil
	case "full_node_scope":
		return googlesql.ParseLocationRecordTypeParseLocationRecordFullNodeScope, nil
	case "code_search":
		return googlesql.ParseLocationRecordTypeParseLocationRecordCodeSearch, nil
	default:
		return 0, fmt.Errorf("unsupported --parse-location-record-type %q", value)
	}
}

func addQueryParameters(analyzer *spanalyzer.Analyzer, params, positionalParams []string) error {
	if len(params) > 0 && len(positionalParams) > 0 {
		return fmt.Errorf("--param and --positional-param cannot be mixed")
	}
	for _, param := range params {
		name, typeSQL, ok := strings.Cut(param, "=")
		if !ok || name == "" || typeSQL == "" {
			return fmt.Errorf("invalid --param %q, want name=TYPE", param)
		}
		spec, err := spanalyzer.ParseTypeSpec("param", typeSQL)
		if err != nil {
			return fmt.Errorf("parameter %s: %w", name, err)
		}
		if err := analyzer.AddQueryParameter(name, spec); err != nil {
			return fmt.Errorf("parameter %s: %w", name, err)
		}
	}
	for i, typeSQL := range positionalParams {
		spec, err := spanalyzer.ParseTypeSpec("positional-param", typeSQL)
		if err != nil {
			return fmt.Errorf("positional parameter %d: %w", i+1, err)
		}
		if err := analyzer.AddPositionalQueryParameter(spec); err != nil {
			return fmt.Errorf("positional parameter %d: %w", i+1, err)
		}
	}
	return nil
}

func exitErr(err error) {
	_ = json.NewEncoder(os.Stderr).Encode(map[string]string{"error": err.Error()})
	os.Exit(1)
}

type repeatedStringFlag []string

func (f *repeatedStringFlag) String() string {
	return fmt.Sprint([]string(*f))
}

func (f *repeatedStringFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}
