package spanalyzer

import (
	"fmt"
	"sort"
	"strings"

	googlesql "github.com/goccy/go-googlesql"
)

type GoogleSQLCatalog struct {
	SpannerCatalog  *Catalog
	SimpleCatalog   *googlesql.SimpleCatalog
	simpleCatalogs  map[string]*googlesql.SimpleCatalog
	AnalyzerOptions *googlesql.AnalyzerOptions
	TypeFactory     *googlesql.TypeFactory
}

func BuildGoogleSQLCatalogFromDDL(path, ddlSQL string, protoDescriptorPaths []string, options ...AnalyzerOption) (*GoogleSQLCatalog, error) {
	schema, err := BuildSchemaCatalog(path, ddlSQL)
	if err != nil {
		return nil, err
	}
	if err := schema.LoadProtoDescriptorSetFiles(protoDescriptorPaths); err != nil {
		return nil, err
	}
	return BuildGoogleSQLCatalogFromSpannerCatalog(schema, options...)
}

func BuildGoogleSQLCatalogFromSpannerCatalog(schema *Catalog, options ...AnalyzerOption) (*GoogleSQLCatalog, error) {
	if schema == nil {
		return nil, fmt.Errorf("nil schema catalog")
	}
	schema.addInformationSchemaTables()
	schema.addSpannerSysTables()
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
	simpleCatalog, opts, err := newGoogleSQLAnalyzerObjects("spanner", tf, config)
	if err != nil {
		return nil, err
	}
	out := &GoogleSQLCatalog{
		SpannerCatalog:  schema,
		SimpleCatalog:   simpleCatalog,
		simpleCatalogs:  map[string]*googlesql.SimpleCatalog{"": simpleCatalog},
		AnalyzerOptions: opts,
		TypeFactory:     tf,
	}
	if err := out.addTables(); err != nil {
		return nil, err
	}
	if err := out.addSequences(); err != nil {
		return nil, err
	}
	if err := out.addModels(); err != nil {
		return nil, err
	}
	if err := out.addSpannerFunctions(); err != nil {
		return nil, err
	}
	if err := out.addViews(); err != nil {
		return nil, err
	}
	if err := out.addPropertyGraphs(); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *GoogleSQLCatalog) Helper() *GoogleSQLHelper {
	return &GoogleSQLHelper{
		Catalog:     c.SimpleCatalog,
		Options:     c.AnalyzerOptions,
		TypeFactory: c.TypeFactory,
	}
}

func newGoogleSQLAnalyzerObjects(rootName string, tf *googlesql.TypeFactory, config analyzerConfig) (*googlesql.SimpleCatalog, *googlesql.AnalyzerOptions, error) {
	catalog, err := googlesql.NewSimpleCatalog(rootName, tf)
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
	if !config.rawMaximumDevelopmentLanguageFeatures {
		if err := applyDialectLanguageFeaturePreset(rootName, lang); err != nil {
			return nil, nil, err
		}
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

func applyDialectLanguageFeaturePreset(dialect string, lang *googlesql.LanguageOptions) error {
	switch dialect {
	case "spanner":
		return disableLanguageFeatures(lang, pipeLanguageFeatures())
	case "bigquery":
		return nil
	default:
		return fmt.Errorf("unknown GoogleSQL dialect preset %q", dialect)
	}
}

func disableLanguageFeatures(lang *googlesql.LanguageOptions, features []googlesql.LanguageFeature) error {
	for _, feature := range features {
		if err := lang.DisableLanguageFeature(feature); err != nil {
			return err
		}
	}
	return nil
}

func pipeLanguageFeatures() []googlesql.LanguageFeature {
	return []googlesql.LanguageFeature{
		googlesql.LanguageFeatureFeaturePipes,
		googlesql.LanguageFeatureFeaturePipeStaticDescribe,
		googlesql.LanguageFeatureFeaturePipeAssert,
		googlesql.LanguageFeatureFeaturePipeLog,
		googlesql.LanguageFeatureFeaturePipeIf,
		googlesql.LanguageFeatureFeaturePipeFork,
		googlesql.LanguageFeatureFeaturePipeExportData,
		googlesql.LanguageFeatureFeaturePipeCreateTable,
		googlesql.LanguageFeatureFeaturePipeTee,
		googlesql.LanguageFeatureFeaturePipeInsert,
		googlesql.LanguageFeatureFeaturePipeWith,
		googlesql.LanguageFeatureFeaturePipeAggregateWithDifferentialPrivacy,
		googlesql.LanguageFeatureFeatureStatementWithPipeOperators,
	}
}

func (c *GoogleSQLCatalog) TypeSpecToGoogleSQLType(spec *TypeSpec) (googlesql.Googlesql_TypeNode, error) {
	return typeSpecToGoogleSQLTypeWithProto(c.TypeFactory, spec, c.SpannerCatalog)
}

func (c *GoogleSQLCatalog) addTables() error {
	for _, table := range c.SpannerCatalog.Tables {
		if err := c.addTable(table); err != nil {
			return err
		}
	}
	return nil
}

func (c *GoogleSQLCatalog) addTable(table *Table) error {
	tableName := table.Name.String()
	parentCatalog, leafName, err := simpleCatalogForObjectName(c.SimpleCatalog, c.simpleCatalogs, table.Name)
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
	primaryKey := make([]int32, 0, len(table.PrimaryKey))
	for i, col := range table.Columns {
		gsType, err := c.TypeSpecToGoogleSQLType(col.Type)
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
	if err := parentCatalog.AddTable(gsTable); err != nil {
		return err
	}
	for _, synonym := range table.Synonyms {
		if err := c.SimpleCatalog.AddTable2(synonym, gsTable); err != nil {
			return err
		}
	}
	return nil
}

func simpleCatalogForObjectName(root *googlesql.SimpleCatalog, catalogs map[string]*googlesql.SimpleCatalog, name ObjectName) (*googlesql.SimpleCatalog, string, error) {
	if len(name.Parts) <= 1 {
		return root, name.String(), nil
	}
	parent := root
	for i, part := range name.Parts[:len(name.Parts)-1] {
		key := strings.Join(name.Parts[:i+1], ".")
		if catalog, ok := catalogs[key]; ok {
			parent = catalog
			continue
		}
		child, err := parent.MakeOwnedSimpleCatalog(part)
		if err != nil {
			return nil, "", err
		}
		catalogs[key] = child
		parent = child
	}
	return parent, name.Parts[len(name.Parts)-1], nil
}

func (c *GoogleSQLCatalog) addViews() error {
	pending := c.orderedViews()
	done := map[string]bool{}
	var lastErr error
	for len(done) < len(pending) {
		progress := false
		for _, view := range pending {
			key := view.Name.String()
			if done[key] {
				continue
			}
			if err := c.addView(view); err != nil {
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

func (c *GoogleSQLCatalog) orderedViews() []*View {
	keys := append([]string(nil), c.SpannerCatalog.ViewOrder...)
	if len(keys) == 0 && len(c.SpannerCatalog.Views) > 0 {
		for key := range c.SpannerCatalog.Views {
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
		if view := c.SpannerCatalog.Views[key]; view != nil {
			views = append(views, view)
		}
	}
	return views
}

func (c *GoogleSQLCatalog) addView(view *View) error {
	helper := c.Helper()
	out, err := helper.AnalyzeStatement(view.Query)
	if err != nil {
		return fmt.Errorf("analyze view %s: %w", view.Name, err)
	}
	rowType, err := RowTypeFromAnalyzerOutput(out, c.SpannerCatalog)
	if err != nil {
		return fmt.Errorf("analyze view %s: %w", view.Name, err)
	}
	table := &Table{Name: view.Name}
	for _, field := range rowType.Fields {
		spec, err := TypeSpecFromSpannerPB(field.Type)
		if err != nil {
			return fmt.Errorf("view %s field %s: %w", view.Name, field.Name, err)
		}
		table.Columns = append(table.Columns, &Column{Name: field.Name, Type: spec})
	}
	return c.addTable(table)
}

func (c *GoogleSQLCatalog) addPropertyGraphs() error {
	names := make([]string, 0, len(c.SpannerCatalog.PropertyGraphs))
	for name := range c.SpannerCatalog.PropertyGraphs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		graph, err := googlesql.NewSimplePropertyGraph([]string{name})
		if err != nil {
			return err
		}
		if err := c.SimpleCatalog.AddPropertyGraph(graph); err != nil {
			return err
		}
	}
	return nil
}

func (c *GoogleSQLCatalog) addSequences() error {
	names := make([]string, 0, len(c.SpannerCatalog.Sequences))
	for name := range c.SpannerCatalog.Sequences {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		sequence, err := googlesql.NewSimpleSequence(name)
		if err != nil {
			return err
		}
		if err := c.SimpleCatalog.AddSequence(sequence); err != nil {
			return err
		}
	}
	return nil
}
