package astconv

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
)

type propertyGraphMetadata struct {
	Catalog              string                       `json:"catalog,omitempty"`
	Schema               string                       `json:"schema,omitempty"`
	Name                 string                       `json:"name"`
	NodeTables           []*propertyGraphElementTable `json:"nodeTables,omitempty"`
	EdgeTables           []*propertyGraphElementTable `json:"edgeTables,omitempty"`
	Labels               []*propertyGraphElementLabel `json:"labels,omitempty"`
	PropertyDeclarations []*propertyGraphPropertyDecl `json:"propertyDeclarations,omitempty"`
}

type propertyGraphElementTable struct {
	Name                 string                           `json:"name"`
	Kind                 string                           `json:"kind"`
	BaseCatalogName      string                           `json:"baseCatalogName,omitempty"`
	BaseSchemaName       string                           `json:"baseSchemaName,omitempty"`
	BaseTableName        string                           `json:"baseTableName"`
	KeyColumns           []string                         `json:"keyColumns,omitempty"`
	LabelNames           []string                         `json:"labelNames,omitempty"`
	PropertyDefinitions  []*propertyGraphPropertyDef      `json:"propertyDefinitions,omitempty"`
	DynamicLabelExpr     string                           `json:"dynamicLabelExpr,omitempty"`
	DynamicPropertyExpr  string                           `json:"dynamicPropertyExpr,omitempty"`
	SourceNodeTable      *propertyGraphNodeTableReference `json:"sourceNodeTable,omitempty"`
	DestinationNodeTable *propertyGraphNodeTableReference `json:"destinationNodeTable,omitempty"`
}

type propertyGraphNodeTableReference struct {
	NodeTableName    string   `json:"nodeTableName"`
	EdgeTableColumns []string `json:"edgeTableColumns,omitempty"`
	NodeTableColumns []string `json:"nodeTableColumns,omitempty"`
}

type propertyGraphElementLabel struct {
	Name                     string   `json:"name"`
	PropertyDeclarationNames []string `json:"propertyDeclarationNames,omitempty"`
}

type propertyGraphPropertyDecl struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

type propertyGraphPropertyDef struct {
	PropertyDeclarationName string `json:"propertyDeclarationName"`
	ValueExpressionSQL      string `json:"valueExpressionSql"`
}

func decodePropertyGraphMetadata(j spanner.NullJSON) (*propertyGraphMetadata, error) {
	if !j.Valid || j.Value == nil {
		return nil, fmt.Errorf("PROPERTY_GRAPH_METADATA_JSON is NULL")
	}

	var data []byte
	switch v := j.Value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	case json.RawMessage:
		data = v
	default:
		var err error
		data, err = json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal property graph metadata: %w", err)
		}
	}

	var meta propertyGraphMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("unmarshal property graph metadata: %w", err)
	}
	if meta.Name == "" {
		return nil, fmt.Errorf("property graph metadata is missing name")
	}
	return &meta, nil
}

func buildPropertyGraphDDL(s *Schema, meta *propertyGraphMetadata) (string, error) {
	if meta.Schema != "" {
		return "", fmt.Errorf("unsupported named-schema property graph %q: memefish cannot reconstruct it yet", tableDisplayName(meta.Schema, meta.Name))
	}

	labelProps := make(map[string][]string, len(meta.Labels))
	for _, label := range meta.Labels {
		labelProps[label.Name] = append([]string(nil), label.PropertyDeclarationNames...)
	}

	nodeKeys := make(map[string][]string, len(meta.NodeTables))
	for _, node := range meta.NodeTables {
		nodeKeys[node.Name] = effectivePropertyGraphElementKey(s, node)
	}

	nodeSQLs := make([]string, 0, len(meta.NodeTables))
	for _, node := range meta.NodeTables {
		sql, err := buildPropertyGraphElementSQL(s, node, labelProps, nodeKeys)
		if err != nil {
			return "", err
		}
		nodeSQLs = append(nodeSQLs, sql)
	}

	parts := []string{
		"CREATE PROPERTY GRAPH " + ident(meta.Name).SQL(),
		"  NODE TABLES (",
		"    " + strings.Join(nodeSQLs, ",\n    "),
		"  )",
	}

	if len(meta.EdgeTables) > 0 {
		edgeSQLs := make([]string, 0, len(meta.EdgeTables))
		for _, edge := range meta.EdgeTables {
			sql, err := buildPropertyGraphElementSQL(s, edge, labelProps, nodeKeys)
			if err != nil {
				return "", err
			}
			edgeSQLs = append(edgeSQLs, sql)
		}
		parts = append(
			parts,
			"  EDGE TABLES (",
			"    "+strings.Join(edgeSQLs, ",\n    "),
			"  )",
		)
	}

	return strings.Join(parts, "\n"), nil
}

func buildPropertyGraphElementSQL(
	s *Schema,
	element *propertyGraphElementTable,
	labelProps map[string][]string,
	nodeKeys map[string][]string,
) (string, error) {
	if element.BaseSchemaName != "" {
		return "", fmt.Errorf("unsupported named-schema property graph element table %q: memefish cannot reconstruct it yet", tableDisplayName(element.BaseSchemaName, element.BaseTableName))
	}
	if element.Name == "" || element.BaseTableName == "" {
		return "", fmt.Errorf("property graph element metadata is missing name or baseTableName")
	}

	var parts []string
	parts = append(parts, ident(element.BaseTableName).SQL())
	if element.Name != element.BaseTableName {
		parts = append(parts, "AS "+ident(element.Name).SQL())
	}

	basePK := tablePrimaryKeyColumns(s, element.BaseTableName)
	effectiveKey := effectivePropertyGraphElementKey(s, element)
	if len(effectiveKey) > 0 && !slices.Equal(effectiveKey, basePK) {
		parts = append(parts, "KEY "+propertyGraphColumnListSQL(effectiveKey))
	}

	if strings.EqualFold(element.Kind, "EDGE") {
		if element.SourceNodeTable == nil || element.DestinationNodeTable == nil {
			return "", fmt.Errorf("edge element %s is missing source or destination node metadata", element.Name)
		}
		sourceSQL, err := buildPropertyGraphReferenceSQL("SOURCE", element.SourceNodeTable, nodeKeys)
		if err != nil {
			return "", fmt.Errorf("edge %s source: %w", element.Name, err)
		}
		destSQL, err := buildPropertyGraphReferenceSQL("DESTINATION", element.DestinationNodeTable, nodeKeys)
		if err != nil {
			return "", fmt.Errorf("edge %s destination: %w", element.Name, err)
		}
		parts = append(parts, sourceSQL, destSQL)
	}

	staticSQL, err := buildPropertyGraphStaticSQL(s, element, labelProps)
	if err != nil {
		return "", err
	}
	if staticSQL != "" {
		parts = append(parts, staticSQL)
	}
	if element.DynamicLabelExpr != "" {
		parts = append(parts, fmt.Sprintf("DYNAMIC LABEL (%s)", ident(element.DynamicLabelExpr).SQL()))
	}
	if element.DynamicPropertyExpr != "" {
		parts = append(parts, fmt.Sprintf("DYNAMIC PROPERTIES (%s)", ident(element.DynamicPropertyExpr).SQL()))
	}

	return strings.Join(parts, " "), nil
}

func buildPropertyGraphReferenceSQL(kind string, ref *propertyGraphNodeTableReference, nodeKeys map[string][]string) (string, error) {
	if ref.NodeTableName == "" {
		return "", fmt.Errorf("%s reference is missing nodeTableName", kind)
	}
	if len(ref.EdgeTableColumns) == 0 {
		return "", fmt.Errorf("%s reference %s is missing edgeTableColumns", kind, ref.NodeTableName)
	}
	sql := fmt.Sprintf("%s KEY %s REFERENCES %s", kind, propertyGraphColumnListSQL(ref.EdgeTableColumns), ident(ref.NodeTableName).SQL())
	if len(ref.NodeTableColumns) > 0 && !slices.Equal(ref.NodeTableColumns, nodeKeys[ref.NodeTableName]) {
		sql += " " + propertyGraphColumnListSQL(ref.NodeTableColumns)
	}
	return sql, nil
}

func buildPropertyGraphStaticSQL(s *Schema, element *propertyGraphElementTable, labelProps map[string][]string) (string, error) {
	propDefs := make(map[string]*propertyGraphPropertyDef, len(element.PropertyDefinitions))
	for _, def := range element.PropertyDefinitions {
		propDefs[def.PropertyDeclarationName] = def
	}

	labelNames := append([]string(nil), element.LabelNames...)
	if len(labelNames) == 0 && len(propDefs) > 0 {
		labelNames = []string{element.Name}
	}

	baseCols := tableColumnNames(s, element.BaseTableName)

	if len(labelNames) == 1 && labelNames[0] == element.Name {
		propNames, declared := labelProps[element.Name]
		if !declared && len(propDefs) > 0 {
			propNames = propertyGraphPropertyNamesInDefinitionOrder(element.PropertyDefinitions)
		}
		propSQL, err := buildPropertyGraphPropertiesSQL(baseCols, propNames, propDefs)
		if err != nil {
			return "", err
		}
		if propSQL == "PROPERTIES ARE ALL COLUMNS" {
			return "", nil
		}
		return propSQL, nil
	}

	var clauses []string
	for _, labelName := range labelNames {
		propNames, declared := labelProps[labelName]
		if !declared && len(propDefs) > 0 {
			return "", fmt.Errorf("label %s is missing property declaration metadata", labelName)
		}
		propSQL, err := buildPropertyGraphPropertiesSQL(baseCols, propNames, propDefs)
		if err != nil {
			return "", err
		}

		labelSQL := "LABEL " + ident(labelName).SQL()
		if labelName == element.Name {
			labelSQL = "DEFAULT LABEL"
		}
		if propSQL != "" {
			labelSQL += " " + propSQL
		}
		clauses = append(clauses, labelSQL)
	}
	return strings.Join(clauses, " "), nil
}

func buildPropertyGraphPropertiesSQL(baseCols, propNames []string, propDefs map[string]*propertyGraphPropertyDef) (string, error) {
	if len(propNames) == 0 {
		return "NO PROPERTIES", nil
	}
	if isIdentityPropertyList(propNames, baseCols, propDefs) {
		return "PROPERTIES ARE ALL COLUMNS", nil
	}
	if exceptCols, ok := identityPropertyExceptColumns(propNames, baseCols, propDefs); ok {
		if len(exceptCols) == 0 {
			return "PROPERTIES ARE ALL COLUMNS", nil
		}
		return "PROPERTIES ARE ALL COLUMNS EXCEPT " + propertyGraphColumnListSQL(exceptCols), nil
	}

	items := make([]string, 0, len(propNames))
	for _, propName := range propNames {
		def, ok := propDefs[propName]
		if !ok {
			return "", fmt.Errorf("missing property definition for %s", propName)
		}
		exprSQL, err := normalizePropertyGraphExprSQL(def.ValueExpressionSQL)
		if err != nil {
			return "", fmt.Errorf("property %s: %w", propName, err)
		}
		if exprSQL == ident(propName).SQL() {
			items = append(items, exprSQL)
			continue
		}
		items = append(items, exprSQL+" AS "+ident(propName).SQL())
	}
	return "PROPERTIES (" + strings.Join(items, ", ") + ")", nil
}

func normalizePropertyGraphExprSQL(expr string) (string, error) {
	parsed, err := memefish.ParseExpr("", expr)
	if err != nil {
		return "", err
	}
	return parsed.SQL(), nil
}

func effectivePropertyGraphElementKey(s *Schema, element *propertyGraphElementTable) []string {
	if len(element.KeyColumns) > 0 {
		return append([]string(nil), element.KeyColumns...)
	}
	return tablePrimaryKeyColumns(s, element.BaseTableName)
}

func propertyGraphColumnListSQL(cols []string) string {
	items := make([]string, len(cols))
	for i, col := range cols {
		items[i] = ident(col).SQL()
	}
	return "(" + strings.Join(items, ", ") + ")"
}

func propertyGraphPropertyNamesInDefinitionOrder(defs []*propertyGraphPropertyDef) []string {
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		names = append(names, def.PropertyDeclarationName)
	}
	return names
}

func isIdentityPropertyList(propNames, baseCols []string, propDefs map[string]*propertyGraphPropertyDef) bool {
	if len(baseCols) == 0 || !slices.Equal(propNames, baseCols) {
		return false
	}
	for _, name := range propNames {
		def, ok := propDefs[name]
		if !ok || def.ValueExpressionSQL != name {
			return false
		}
	}
	return true
}

func identityPropertyExceptColumns(propNames, baseCols []string, propDefs map[string]*propertyGraphPropertyDef) ([]string, bool) {
	if len(baseCols) == 0 {
		return nil, false
	}
	included := make([]string, 0, len(baseCols))
	includedSet := make(map[string]struct{}, len(propNames))
	for _, name := range propNames {
		def, ok := propDefs[name]
		if !ok || def.ValueExpressionSQL != name {
			return nil, false
		}
		included = append(included, name)
		includedSet[name] = struct{}{}
	}
	expectedIncluded := make([]string, 0, len(baseCols))
	var except []string
	for _, baseCol := range baseCols {
		if _, ok := includedSet[baseCol]; ok {
			expectedIncluded = append(expectedIncluded, baseCol)
			continue
		}
		except = append(except, baseCol)
	}
	if !slices.Equal(included, expectedIncluded) {
		return nil, false
	}
	return except, true
}

func tableColumnNames(s *Schema, tableName string) []string {
	type columnPos struct {
		name string
		pos  int64
	}
	var cols []columnPos
	for _, col := range s.Columns {
		if col.TableSchema == "" && col.TableName == tableName {
			cols = append(cols, columnPos{name: col.ColumnName, pos: col.OrdinalPosition})
		}
	}
	slices.SortFunc(cols, func(a, b columnPos) int {
		switch {
		case a.pos < b.pos:
			return -1
		case a.pos > b.pos:
			return 1
		default:
			return strings.Compare(a.name, b.name)
		}
	})
	names := make([]string, 0, len(cols))
	for _, col := range cols {
		names = append(names, col.name)
	}
	return names
}

func tablePrimaryKeyColumns(s *Schema, tableName string) []string {
	type keyPos struct {
		name string
		pos  int64
	}
	var cols []keyPos
	for _, col := range s.IndexColumns {
		if col.TableSchema == "" && col.TableName == tableName && col.IndexName == "PRIMARY_KEY" && col.OrdinalPosition != nil {
			cols = append(cols, keyPos{name: col.ColumnName, pos: *col.OrdinalPosition})
		}
	}
	slices.SortFunc(cols, func(a, b keyPos) int {
		switch {
		case a.pos < b.pos:
			return -1
		case a.pos > b.pos:
			return 1
		default:
			return strings.Compare(a.name, b.name)
		}
	})
	names := make([]string, 0, len(cols))
	for _, col := range cols {
		names = append(names, col.name)
	}
	return names
}

type propertyGraphMetadataBuilder struct {
	s             *Schema
	labelOrder    []string
	labelProps    map[string][]string
	propertyOrder []string
	propertyTypes map[string]string
}

func newPropertyGraphMetadataBuilder(s *Schema) *propertyGraphMetadataBuilder {
	return &propertyGraphMetadataBuilder{
		s:             s,
		labelProps:    make(map[string][]string),
		propertyTypes: make(map[string]string),
	}
}

func (b *propertyGraphMetadataBuilder) fromAST(cpg *ast.CreatePropertyGraph) (*propertyGraphMetadata, error) {
	if cpg.Name == nil || cpg.Content == nil || cpg.Content.NodeTables == nil || cpg.Content.NodeTables.Tables == nil {
		return nil, fmt.Errorf("invalid CREATE PROPERTY GRAPH AST")
	}

	meta := &propertyGraphMetadata{
		Catalog: "",
		Schema:  "",
		Name:    cpg.Name.Name,
	}

	nodeKeys := make(map[string][]string)
	for _, element := range cpg.Content.NodeTables.Tables.Elements {
		node, err := b.fromASTElement(element, "NODE")
		if err != nil {
			return nil, err
		}
		if len(node.KeyColumns) == 0 {
			node.KeyColumns = tablePrimaryKeyColumns(b.s, node.BaseTableName)
		}
		meta.NodeTables = append(meta.NodeTables, node)
		nodeKeys[node.Name] = append([]string(nil), node.KeyColumns...)
	}

	if cpg.Content.EdgeTables != nil && cpg.Content.EdgeTables.Tables != nil {
		for _, element := range cpg.Content.EdgeTables.Tables.Elements {
			edge, err := b.fromASTElement(element, "EDGE")
			if err != nil {
				return nil, err
			}
			if len(edge.KeyColumns) == 0 {
				edge.KeyColumns = tablePrimaryKeyColumns(b.s, edge.BaseTableName)
			}
			if edge.SourceNodeTable != nil && len(edge.SourceNodeTable.NodeTableColumns) == 0 {
				edge.SourceNodeTable.NodeTableColumns = append([]string(nil), nodeKeys[edge.SourceNodeTable.NodeTableName]...)
			}
			if edge.DestinationNodeTable != nil && len(edge.DestinationNodeTable.NodeTableColumns) == 0 {
				edge.DestinationNodeTable.NodeTableColumns = append([]string(nil), nodeKeys[edge.DestinationNodeTable.NodeTableName]...)
			}
			meta.EdgeTables = append(meta.EdgeTables, edge)
		}
	}

	for _, labelName := range b.labelOrder {
		meta.Labels = append(meta.Labels, &propertyGraphElementLabel{
			Name:                     labelName,
			PropertyDeclarationNames: append([]string(nil), b.labelProps[labelName]...),
		})
	}
	for _, propertyName := range b.propertyOrder {
		meta.PropertyDeclarations = append(meta.PropertyDeclarations, &propertyGraphPropertyDecl{
			Name: propertyName,
			Type: b.propertyTypes[propertyName],
		})
	}

	return meta, nil
}

func (b *propertyGraphMetadataBuilder) fromASTElement(element *ast.PropertyGraphElement, kind string) (*propertyGraphElementTable, error) {
	graphName := element.Name.Name
	if element.Alias != nil {
		graphName = element.Alias.Name
	}
	meta := &propertyGraphElementTable{
		Name:          graphName,
		Kind:          kind,
		BaseTableName: element.Name.Name,
	}

	switch keys := element.Keys.(type) {
	case *ast.PropertyGraphNodeElementKey:
		if keys != nil && keys.Key != nil && keys.Key.Keys != nil {
			meta.KeyColumns = identNames(keys.Key.Keys.ColumnNameList)
		}
	case *ast.PropertyGraphEdgeElementKeys:
		if keys.Element != nil && keys.Element.Keys != nil {
			meta.KeyColumns = identNames(keys.Element.Keys.ColumnNameList)
		}
		if keys.Source != nil {
			meta.SourceNodeTable = &propertyGraphNodeTableReference{
				NodeTableName:    keys.Source.ElementReference.Name,
				EdgeTableColumns: identNames(keys.Source.Keys.ColumnNameList),
			}
			if keys.Source.ReferenceColumns != nil {
				meta.SourceNodeTable.NodeTableColumns = identNames(keys.Source.ReferenceColumns.ColumnNameList)
			}
		}
		if keys.Destination != nil {
			meta.DestinationNodeTable = &propertyGraphNodeTableReference{
				NodeTableName:    keys.Destination.ElementReference.Name,
				EdgeTableColumns: identNames(keys.Destination.Keys.ColumnNameList),
			}
			if keys.Destination.ReferenceColumns != nil {
				meta.DestinationNodeTable.NodeTableColumns = identNames(keys.Destination.ReferenceColumns.ColumnNameList)
			}
		}
	}

	if element.DynamicLabel != nil {
		meta.DynamicLabelExpr = element.DynamicLabel.ColumnName.Name
	}
	if element.DynamicProperties != nil {
		meta.DynamicPropertyExpr = element.DynamicProperties.ColumnName.Name
	}

	labelNames, labelProps, propDefs, err := b.extractElementProperties(meta.Name, meta.BaseTableName, element.Properties)
	if err != nil {
		return nil, fmt.Errorf("property graph element %s: %w", meta.Name, err)
	}
	meta.LabelNames = labelNames
	meta.PropertyDefinitions = propDefs

	for _, labelName := range labelNames {
		if err := b.recordLabel(labelName, labelProps[labelName]); err != nil {
			return nil, err
		}
	}
	for _, def := range propDefs {
		if err := b.recordProperty(def.PropertyDeclarationName, b.inferPropertyType(meta.BaseTableName, def)); err != nil {
			return nil, err
		}
	}

	return meta, nil
}

func (b *propertyGraphMetadataBuilder) extractElementProperties(
	elementName, baseTableName string,
	props ast.PropertyGraphLabelsOrProperties,
) ([]string, map[string][]string, []*propertyGraphPropertyDef, error) {
	switch p := props.(type) {
	case nil:
		propNames, defs, err := b.propertyDefinitionsFor(baseTableName, nil)
		if err != nil {
			return nil, nil, nil, err
		}
		return []string{elementName}, map[string][]string{elementName: propNames}, defs, nil
	case *ast.PropertyGraphSingleProperties:
		propNames, defs, err := b.propertyDefinitionsFor(baseTableName, p.Properties)
		if err != nil {
			return nil, nil, nil, err
		}
		return []string{elementName}, map[string][]string{elementName: propNames}, defs, nil
	case *ast.PropertyGraphLabelAndPropertiesList:
		labelProps := make(map[string][]string, len(p.LabelAndProperties))
		var labelNames []string
		var defs []*propertyGraphPropertyDef
		defByName := make(map[string]string)
		for _, item := range p.LabelAndProperties {
			labelName := propertyGraphLabelName(item.Label, elementName)
			labelNames = append(labelNames, labelName)
			propNames, itemDefs, err := b.propertyDefinitionsFor(baseTableName, item.Properties)
			if err != nil {
				return nil, nil, nil, err
			}
			labelProps[labelName] = propNames
			for _, def := range itemDefs {
				if prev, ok := defByName[def.PropertyDeclarationName]; ok {
					if prev != def.ValueExpressionSQL {
						return nil, nil, nil, fmt.Errorf("property %s is defined with conflicting expressions", def.PropertyDeclarationName)
					}
					continue
				}
				defByName[def.PropertyDeclarationName] = def.ValueExpressionSQL
				defs = append(defs, def)
			}
		}
		return labelNames, labelProps, defs, nil
	default:
		return nil, nil, nil, fmt.Errorf("unsupported property graph properties type %T", props)
	}
}

func (b *propertyGraphMetadataBuilder) propertyDefinitionsFor(baseTableName string, props ast.PropertyGraphElementProperties) ([]string, []*propertyGraphPropertyDef, error) {
	baseCols := tableColumnNames(b.s, baseTableName)
	switch p := props.(type) {
	case nil:
		if len(baseCols) == 0 {
			return nil, nil, fmt.Errorf("table %s columns are required to materialize implicit graph properties", baseTableName)
		}
		return propertyGraphIdentityDefinitions(baseCols), propertyGraphIdentityPropertyDefs(baseCols), nil
	case *ast.PropertyGraphNoProperties:
		return nil, nil, nil
	case *ast.PropertyGraphPropertiesAre:
		if len(baseCols) == 0 {
			return nil, nil, fmt.Errorf("table %s columns are required to materialize graph properties", baseTableName)
		}
		except := map[string]struct{}{}
		if p.ExceptColumns != nil {
			for _, col := range p.ExceptColumns.ColumnNameList {
				except[col.Name] = struct{}{}
			}
		}
		var included []string
		for _, baseCol := range baseCols {
			if _, ok := except[baseCol]; ok {
				continue
			}
			included = append(included, baseCol)
		}
		return propertyGraphIdentityDefinitions(included), propertyGraphIdentityPropertyDefs(included), nil
	case *ast.PropertyGraphDerivedPropertyList:
		var names []string
		var defs []*propertyGraphPropertyDef
		for _, dp := range p.DerivedProperties {
			name, err := propertyGraphDerivedPropertyName(dp)
			if err != nil {
				return nil, nil, err
			}
			names = append(names, name)
			defs = append(defs, &propertyGraphPropertyDef{
				PropertyDeclarationName: name,
				ValueExpressionSQL:      dp.Expr.SQL(),
			})
		}
		return names, defs, nil
	default:
		return nil, nil, fmt.Errorf("unsupported property graph element properties type %T", props)
	}
}

func propertyGraphIdentityDefinitions(cols []string) []string {
	return append([]string(nil), cols...)
}

func propertyGraphIdentityPropertyDefs(cols []string) []*propertyGraphPropertyDef {
	defs := make([]*propertyGraphPropertyDef, 0, len(cols))
	for _, col := range cols {
		defs = append(defs, &propertyGraphPropertyDef{
			PropertyDeclarationName: col,
			ValueExpressionSQL:      col,
		})
	}
	return defs
}

func propertyGraphDerivedPropertyName(dp *ast.PropertyGraphDerivedProperty) (string, error) {
	if dp.Alias != nil {
		return dp.Alias.Name, nil
	}
	switch expr := dp.Expr.(type) {
	case *ast.Path:
		if len(expr.Idents) > 0 {
			return leafName(expr), nil
		}
	case *ast.Ident:
		return expr.Name, nil
	}
	return "", fmt.Errorf("derived property %s requires AS alias", dp.Expr.SQL())
}

func propertyGraphLabelName(label ast.PropertyGraphElementLabel, defaultName string) string {
	switch l := label.(type) {
	case *ast.PropertyGraphElementLabelDefaultLabel:
		return defaultName
	case *ast.PropertyGraphElementLabelLabelName:
		return l.Name.Name
	default:
		return defaultName
	}
}

func identNames(idents []*ast.Ident) []string {
	names := make([]string, 0, len(idents))
	for _, ident := range idents {
		names = append(names, ident.Name)
	}
	return names
}

func (b *propertyGraphMetadataBuilder) recordLabel(name string, props []string) error {
	if existing, ok := b.labelProps[name]; ok {
		if !slices.Equal(existing, props) {
			return fmt.Errorf("label %s exposes inconsistent property sets", name)
		}
		return nil
	}
	b.labelOrder = append(b.labelOrder, name)
	b.labelProps[name] = append([]string(nil), props...)
	return nil
}

func (b *propertyGraphMetadataBuilder) recordProperty(name, typ string) error {
	if existing, ok := b.propertyTypes[name]; ok {
		if existing != "" && typ != "" && existing != typ {
			return fmt.Errorf("property %s has inconsistent types %s and %s", name, existing, typ)
		}
		if existing == "" && typ != "" {
			b.propertyTypes[name] = typ
		}
		return nil
	}
	b.propertyOrder = append(b.propertyOrder, name)
	b.propertyTypes[name] = typ
	return nil
}

func (b *propertyGraphMetadataBuilder) inferPropertyType(baseTableName string, def *propertyGraphPropertyDef) string {
	if def.ValueExpressionSQL != def.PropertyDeclarationName {
		return ""
	}
	for _, col := range b.s.Columns {
		if col.TableSchema == "" && col.TableName == baseTableName && col.ColumnName == def.PropertyDeclarationName {
			return col.SpannerType
		}
	}
	return ""
}
