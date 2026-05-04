package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	spanalyzer "github.com/apstndb/go-googlesql-spanner-poc"
	googlesql "github.com/goccy/go-googlesql"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func main() {
	ddlPath := flag.String("ddl", "", "optional path to a Spanner GoogleSQL DDL file")
	sql := flag.String("sql", "", "SQL query or expression to analyze")
	mode := flag.String("mode", "spanner_type", "comma-separated modes: spanner_type, parse, analyze, unparse")
	sqlMode := flag.String("sql-mode", "query", "how to interpret --sql: query or expression")
	output := flag.String("output", "json", "output format: json or textproto")
	productMode := flag.String("product-mode", "", "GoogleSQL product mode: internal or external")
	strictNameResolution := flag.Bool("strict-name-resolution", false, "enable strict name resolution")
	foldLiteralCast := flag.Bool("fold-literal-cast", true, "set AnalyzerOptions fold_literal_cast")
	pruneUnusedColumns := flag.Bool("prune-unused-columns", true, "set AnalyzerOptions prune_unused_columns")
	parseLocationRecordType := flag.String("parse-location-record-type", "none", "parse location recording: none, full_node_scope, or code_search")
	var protoDescriptorFiles repeatedStringFlag
	var params repeatedStringFlag
	var positionalParams repeatedStringFlag
	flag.Var(&protoDescriptorFiles, "proto-descriptors-file", "path to a FileDescriptorSet used by CREATE/ALTER PROTO BUNDLE; repeatable")
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
	options, err := analyzerOptionsFromFlags(*productMode, *strictNameResolution, *foldLiteralCast, *pruneUnusedColumns, *parseLocationRecordType)
	if err != nil {
		exitErr(err)
	}
	analyzer, err := spanalyzer.NewAnalyzerFromDDLWithProtoDescriptorFilesAndOptions(ddlPathForAnalyzer, ddl, protoDescriptorFiles, options...)
	if err != nil {
		exitErr(err)
	}
	if err := addQueryParameters(analyzer, params, positionalParams); err != nil {
		exitErr(err)
	}
	result, err := runModes(analyzer, splitModes(*mode), strings.ToLower(*sqlMode), *sql, *output)
	if err != nil {
		exitErr(err)
	}
	fmt.Print(result)
}

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

func marshalProto(message anyProto, output string) ([]byte, error) {
	switch strings.ToLower(output) {
	case "json":
		return protojson.MarshalOptions{
			Multiline:     true,
			Indent:        "  ",
			UseProtoNames: true,
		}.Marshal(message)
	case "textproto":
		return prototext.MarshalOptions{
			Multiline: true,
			Indent:    "  ",
		}.Marshal(message)
	default:
		return nil, fmt.Errorf("unsupported --output %q", output)
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

func analyzerOptionsFromFlags(productMode string, strictNameResolution, foldLiteralCast, pruneUnusedColumns bool, parseLocationRecordType string) ([]spanalyzer.AnalyzerOption, error) {
	options := []spanalyzer.AnalyzerOption{
		spanalyzer.WithStrictNameResolution(strictNameResolution),
		spanalyzer.WithFoldLiteralCast(foldLiteralCast),
		spanalyzer.WithPruneUnusedColumns(pruneUnusedColumns),
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
