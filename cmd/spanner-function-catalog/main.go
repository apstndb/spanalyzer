package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	spanalyzer "github.com/apstndb/go-googlesql-spanner-poc"
	googlesql "github.com/goccy/go-googlesql"
)

func main() {
	ddlPath := flag.String("ddl", "", "path to a Spanner GoogleSQL DDL file")
	verbose := flag.Bool("verbose", true, "include function signatures")
	productMode := flag.String("product-mode", "external", "GoogleSQL product mode: internal or external")
	strictNameResolution := flag.Bool("strict-name-resolution", false, "enable strict name resolution")
	var protoDescriptorFiles repeatedStringFlag
	flag.Var(&protoDescriptorFiles, "proto-descriptors-file", "path to a FileDescriptorSet used by CREATE/ALTER PROTO BUNDLE; repeatable")
	flag.Parse()

	ddlPathForAnalyzer := *ddlPath
	var ddl string
	if ddlPathForAnalyzer != "" {
		ddlBytes, err := os.ReadFile(ddlPathForAnalyzer)
		if err != nil {
			exitErr(err)
		}
		ddl = string(ddlBytes)
	} else {
		ddlPathForAnalyzer = "schema.sql"
	}

	options, err := analyzerOptionsFromFlags(*productMode, *strictNameResolution)
	if err != nil {
		exitErr(err)
	}
	analyzer, err := spanalyzer.NewAnalyzerFromDDLWithProtoDescriptorFilesAndOptions(ddlPathForAnalyzer, ddl, protoDescriptorFiles, options...)
	if err != nil {
		exitErr(err)
	}
	dump, err := analyzer.FunctionCatalogDebugString(*verbose)
	if err != nil {
		exitErr(err)
	}
	fmt.Print(dump)
}

func analyzerOptionsFromFlags(productMode string, strictNameResolution bool) ([]spanalyzer.AnalyzerOption, error) {
	options := []spanalyzer.AnalyzerOption{
		spanalyzer.WithStrictNameResolution(strictNameResolution),
	}
	if productMode != "" {
		mode, err := parseProductMode(productMode)
		if err != nil {
			return nil, err
		}
		options = append(options, spanalyzer.WithProductMode(mode))
	}
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
