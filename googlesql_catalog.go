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
	tables          map[string]*googlesql.SimpleTable
	models          map[string]*registeredGoogleSQLModel
	AnalyzerOptions *googlesql.AnalyzerOptions
	TypeFactory     *googlesql.TypeFactory
	DescriptorPool  *googlesql.DescriptorPool
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
	descriptorPool, err := buildGoogleSQLDescriptorPool(schema.ProtoDescriptors)
	if err != nil {
		return nil, err
	}
	if descriptorPool != nil {
		if err := simpleCatalog.SetDescriptorPool(descriptorPool); err != nil {
			return nil, err
		}
	}
	out := &GoogleSQLCatalog{
		SpannerCatalog:  schema,
		SimpleCatalog:   simpleCatalog,
		simpleCatalogs:  map[string]*googlesql.SimpleCatalog{"": simpleCatalog},
		tables:          map[string]*googlesql.SimpleTable{},
		models:          map[string]*registeredGoogleSQLModel{},
		AnalyzerOptions: opts,
		TypeFactory:     tf,
		DescriptorPool:  descriptorPool,
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
	if err := out.addMLPredict(); err != nil {
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
	return typeSpecToGoogleSQLTypeWithProto(c.TypeFactory, spec, c.SpannerCatalog, c.DescriptorPool)
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
	gsTable, err := googlesql.NewSimpleTable(leafName, -1)
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
		gsCol, err := googlesql.NewSimpleColumn(tableName, col.Name, gsType, col.Hidden, false)
		if err != nil {
			return err
		}
		if err := gsTable.AddColumn2(gsCol, true); err != nil {
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
	needsBorrowedTable := len(c.SpannerCatalog.PropertyGraphs) > 0 || len(table.Synonyms) > 0
	if needsBorrowedTable {
		if err := parentCatalog.AddTable(gsTable); err != nil {
			return err
		}
		c.tables[tableName] = gsTable
		c.tables[leafName] = gsTable
		for _, synonym := range table.Synonyms {
			if err := c.SimpleCatalog.AddTable2(synonym, gsTable); err != nil {
				return err
			}
		}
		return nil
	}
	return parentCatalog.AddOwnedTable(gsTable)
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
		graph, err := c.newGoogleSQLPropertyGraph(c.SpannerCatalog.PropertyGraphs[name])
		if err != nil {
			return err
		}
		if err := c.SimpleCatalog.AddOwnedPropertyGraph(graph); err != nil {
			return err
		}
	}
	return nil
}

func (c *GoogleSQLCatalog) newGoogleSQLPropertyGraph(graph *PropertyGraph) (*googlesql.SimplePropertyGraph, error) {
	if graph == nil {
		return nil, fmt.Errorf("nil property graph")
	}
	namePath := []string{graph.Name}
	out, err := googlesql.NewSimplePropertyGraph(namePath)
	if err != nil {
		return nil, err
	}
	state := &propertyGraphBuildState{
		catalog:              c,
		graphNamePath:        namePath,
		graph:                out,
		propertyDeclarations: map[string]*graphPropertyDeclaration{},
		labels:               map[string]googlesql.GraphElementLabelNode{},
		addedDeclarations:    map[string]bool{},
		addedLabels:          map[string]bool{},
		nodeTables:           map[string]googlesql.GraphNodeTableNode{},
	}
	for _, elem := range graph.NodeTables {
		node, err := state.addNodeTable(elem)
		if err != nil {
			return nil, fmt.Errorf("property graph %s node table %s: %w", graph.Name, elem.Name, err)
		}
		state.nodeTables[graphElementName(elem)] = node
		state.nodeTables[elem.Name] = node
		if err := state.addGraphDeclarationsAndLabels(); err != nil {
			return nil, err
		}
		if err := out.AddNodeTable(node); err != nil {
			return nil, err
		}
	}
	for _, elem := range graph.EdgeTables {
		edge, err := state.addEdgeTable(elem)
		if err != nil {
			return nil, fmt.Errorf("property graph %s edge table %s: %w", graph.Name, elem.Name, err)
		}
		if err := state.addGraphDeclarationsAndLabels(); err != nil {
			return nil, err
		}
		if err := out.AddEdgeTable(edge); err != nil {
			return nil, err
		}
	}
	return out, nil
}

type propertyGraphBuildState struct {
	catalog              *GoogleSQLCatalog
	graphNamePath        []string
	graph                *googlesql.SimplePropertyGraph
	propertyDeclarations map[string]*graphPropertyDeclaration
	labels               map[string]googlesql.GraphElementLabelNode
	addedDeclarations    map[string]bool
	addedLabels          map[string]bool
	nodeTables           map[string]googlesql.GraphNodeTableNode
}

type graphPropertyDeclaration struct {
	typ  googlesql.Googlesql_TypeNode
	decl googlesql.GraphPropertyDeclarationNode
}

type graphPropertyBinding struct {
	name string
	expr string
	typ  googlesql.Googlesql_TypeNode
}

func (s *propertyGraphBuildState) addNodeTable(elem *GraphElement) (*googlesql.SimpleGraphNodeTable, error) {
	inputTable, table, err := s.inputTable(elem)
	if err != nil {
		return nil, err
	}
	keyCols, err := tableColumnIndexes(table, graphKeyColumns(table, elem.KeyColumns))
	if err != nil {
		return nil, err
	}
	labels, definitions, err := s.labelsAndDefinitions(table, elem)
	if err != nil {
		return nil, err
	}
	dynamicLabel, err := newGraphDynamicLabel(elem.DynamicLabel)
	if err != nil {
		return nil, err
	}
	dynamicProperties, err := newGraphDynamicProperties(elem.DynamicProperties)
	if err != nil {
		return nil, err
	}
	return googlesql.NewSimpleGraphNodeTable(
		graphElementName(elem),
		s.graphNamePath,
		inputTable,
		keyCols,
		labels,
		definitions,
		dynamicLabel,
		dynamicProperties,
	)
}

func (s *propertyGraphBuildState) addEdgeTable(elem *GraphElement) (*googlesql.SimpleGraphEdgeTable, error) {
	inputTable, table, err := s.inputTable(elem)
	if err != nil {
		return nil, err
	}
	keyCols, err := tableColumnIndexes(table, graphKeyColumns(table, elem.KeyColumns))
	if err != nil {
		return nil, err
	}
	labels, definitions, err := s.labelsAndDefinitions(table, elem)
	if err != nil {
		return nil, err
	}
	source, err := s.nodeTableReference(table, elem.Source)
	if err != nil {
		return nil, fmt.Errorf("source: %w", err)
	}
	destination, err := s.nodeTableReference(table, elem.Destination)
	if err != nil {
		return nil, fmt.Errorf("destination: %w", err)
	}
	dynamicLabel, err := newGraphDynamicLabel(elem.DynamicLabel)
	if err != nil {
		return nil, err
	}
	dynamicProperties, err := newGraphDynamicProperties(elem.DynamicProperties)
	if err != nil {
		return nil, err
	}
	return googlesql.NewSimpleGraphEdgeTable(
		graphElementName(elem),
		s.graphNamePath,
		inputTable,
		keyCols,
		labels,
		definitions,
		source,
		destination,
		dynamicLabel,
		dynamicProperties,
	)
}

func (s *propertyGraphBuildState) inputTable(elem *GraphElement) (*googlesql.SimpleTable, *Table, error) {
	table := s.catalog.SpannerCatalog.Tables[elem.Name]
	if table == nil {
		return nil, nil, fmt.Errorf("backing table %s not found", elem.Name)
	}
	inputTable := s.catalog.tables[elem.Name]
	if inputTable == nil {
		return nil, nil, fmt.Errorf("GoogleSQL table %s not found", elem.Name)
	}
	return inputTable, table, nil
}

func (s *propertyGraphBuildState) labelsAndDefinitions(table *Table, elem *GraphElement) ([]googlesql.GraphElementLabelNode, []googlesql.GraphPropertyDefinitionNode, error) {
	labels := elem.Labels
	if len(labels) == 0 {
		labels = []*GraphLabelProperties{{Default: true, Properties: elem.Properties}}
	}
	outLabels := make([]googlesql.GraphElementLabelNode, 0, len(labels))
	definitions := []googlesql.GraphPropertyDefinitionNode{}
	defined := map[string]bool{}
	for _, label := range labels {
		bindings, err := s.propertyBindings(table, label.Properties)
		if err != nil {
			return nil, nil, err
		}
		propertyDecls := make([]googlesql.GraphPropertyDeclarationNode, 0, len(bindings))
		for _, binding := range bindings {
			decl, err := s.propertyDeclaration(binding.name, binding.typ)
			if err != nil {
				return nil, nil, err
			}
			propertyDecls = append(propertyDecls, decl)
			if !defined[strings.ToLower(binding.name)] {
				definition, err := googlesql.NewSimpleGraphPropertyDefinition(decl, binding.expr)
				if err != nil {
					return nil, nil, err
				}
				definitions = append(definitions, definition)
				defined[strings.ToLower(binding.name)] = true
			}
		}
		labelName := label.Name
		if label.Default || labelName == "" {
			labelName = graphElementName(elem)
		}
		gsLabel, err := s.label(labelName, propertyDecls)
		if err != nil {
			return nil, nil, err
		}
		outLabels = append(outLabels, gsLabel)
	}
	return outLabels, definitions, nil
}

func (s *propertyGraphBuildState) propertyBindings(table *Table, props *GraphProperties) ([]graphPropertyBinding, error) {
	if props == nil || props.NoProperties {
		return nil, nil
	}
	if props.AllColumns {
		except := stringSetFromSlice(props.ExceptColumns)
		bindings := make([]graphPropertyBinding, 0, len(table.Columns))
		for _, col := range table.Columns {
			if col.Hidden || except[strings.ToLower(col.Name)] {
				continue
			}
			typ, err := s.catalog.TypeSpecToGoogleSQLType(col.Type)
			if err != nil {
				return nil, fmt.Errorf("property %s: %w", col.Name, err)
			}
			bindings = append(bindings, graphPropertyBinding{name: col.Name, expr: col.Name, typ: typ})
		}
		return bindings, nil
	}
	if len(props.DerivedProperties) == 0 {
		return nil, nil
	}
	bindings := make([]graphPropertyBinding, 0, len(props.DerivedProperties))
	for _, prop := range props.DerivedProperties {
		col, _ := table.Column(prop.SQL)
		if col == nil {
			return nil, fmt.Errorf("derived property %s uses expression %q; only direct column-derived properties are currently supported", prop.Name, prop.SQL)
		}
		typ, err := s.catalog.TypeSpecToGoogleSQLType(col.Type)
		if err != nil {
			return nil, fmt.Errorf("property %s: %w", prop.Name, err)
		}
		bindings = append(bindings, graphPropertyBinding{name: prop.Name, expr: prop.SQL, typ: typ})
	}
	return bindings, nil
}

func (s *propertyGraphBuildState) propertyDeclaration(name string, typ googlesql.Googlesql_TypeNode) (googlesql.GraphPropertyDeclarationNode, error) {
	key := strings.ToLower(name)
	if existing := s.propertyDeclarations[key]; existing != nil {
		return existing.decl, nil
	}
	decl, err := googlesql.NewSimpleGraphPropertyDeclaration(name, s.graphNamePath, typ)
	if err != nil {
		return nil, err
	}
	s.propertyDeclarations[key] = &graphPropertyDeclaration{typ: typ, decl: decl}
	return decl, nil
}

func (s *propertyGraphBuildState) label(name string, propertyDeclarations []googlesql.GraphPropertyDeclarationNode) (googlesql.GraphElementLabelNode, error) {
	key := strings.ToLower(name)
	if label := s.labels[key]; label != nil {
		return label, nil
	}
	label, err := googlesql.NewSimpleGraphElementLabel(name, s.graphNamePath, propertyDeclarations)
	if err != nil {
		return nil, err
	}
	s.labels[key] = label
	return label, nil
}

func (s *propertyGraphBuildState) addGraphDeclarationsAndLabels() error {
	declarationKeys := make([]string, 0, len(s.propertyDeclarations))
	for key := range s.propertyDeclarations {
		declarationKeys = append(declarationKeys, key)
	}
	sort.Strings(declarationKeys)
	for _, key := range declarationKeys {
		if s.addedDeclarations[key] {
			continue
		}
		if err := s.graph.AddPropertyDeclaration(s.propertyDeclarations[key].decl); err != nil {
			return err
		}
		s.addedDeclarations[key] = true
	}

	labelKeys := make([]string, 0, len(s.labels))
	for key := range s.labels {
		labelKeys = append(labelKeys, key)
	}
	sort.Strings(labelKeys)
	for _, key := range labelKeys {
		if s.addedLabels[key] {
			continue
		}
		if err := s.graph.AddLabel(s.labels[key]); err != nil {
			return err
		}
		s.addedLabels[key] = true
	}
	return nil
}

func (s *propertyGraphBuildState) nodeTableReference(edgeTable *Table, endpoint *GraphEndpoint) (googlesql.GraphNodeTableReferenceNode, error) {
	if endpoint == nil {
		return nil, nil
	}
	node := s.nodeTables[endpoint.ElementReference]
	if node == nil {
		return nil, fmt.Errorf("node table %s not found", endpoint.ElementReference)
	}
	nodeTable := s.catalog.SpannerCatalog.Tables[endpoint.ElementReference]
	if nodeTable == nil {
		return nil, fmt.Errorf("backing node table %s not found", endpoint.ElementReference)
	}
	edgeCols, err := tableColumnIndexes(edgeTable, endpoint.KeyColumns)
	if err != nil {
		return nil, err
	}
	refCols := endpoint.ReferenceColumns
	if len(refCols) == 0 {
		refCols = graphKeyColumns(nodeTable, nil)
	}
	nodeCols, err := tableColumnIndexes(nodeTable, refCols)
	if err != nil {
		return nil, err
	}
	return googlesql.NewSimpleGraphNodeTableReference(node, edgeCols, nodeCols)
}

func graphElementName(elem *GraphElement) string {
	if elem != nil && elem.Alias != "" {
		return elem.Alias
	}
	if elem == nil {
		return ""
	}
	return elem.Name
}

func graphKeyColumns(table *Table, explicit []string) []string {
	if len(explicit) > 0 {
		return explicit
	}
	out := make([]string, 0, len(table.PrimaryKey))
	for _, key := range table.PrimaryKey {
		out = append(out, key.Name)
	}
	return out
}

func tableColumnIndexes(table *Table, names []string) ([]int32, error) {
	indexes := make([]int32, 0, len(names))
	for _, name := range names {
		_, index := table.Column(name)
		if index < 0 {
			return nil, fmt.Errorf("column %s not found in table %s", name, table.Name)
		}
		indexes = append(indexes, int32(index))
	}
	return indexes, nil
}

func newGraphDynamicLabel(column string) (googlesql.GraphDynamicLabelNode, error) {
	if column == "" {
		return nil, nil
	}
	return googlesql.NewSimpleGraphDynamicLabel(column)
}

func newGraphDynamicProperties(column string) (googlesql.GraphDynamicPropertiesNode, error) {
	if column == "" {
		return nil, nil
	}
	return googlesql.NewSimpleGraphDynamicProperties(column)
}

func stringSetFromSlice(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[strings.ToLower(value)] = true
	}
	return out
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
