package spanalyzer

import (
	"fmt"
	"strconv"
	"strings"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
)

type Catalog struct {
	Tables           map[string]*Table
	Indexes          map[string]*Index
	Views            map[string]*View
	ViewOrder        []string
	PropertyGraphs   map[string]*PropertyGraph
	Sequences        map[string]*Sequence
	Models           map[string]*Model
	ProtoTypes       map[string]bool
	ProtoDescriptors *ProtoDescriptorSet
}

type ObjectName struct {
	Parts []string
}

func (n ObjectName) String() string {
	return strings.Join(n.Parts, ".")
}

type Table struct {
	Name       ObjectName
	Columns    []*Column
	PrimaryKey []*KeyPart
	Synonyms   []string
}

type Index struct {
	Name          ObjectName
	TableName     ObjectName
	Keys          []*KeyPart
	StoredColumns []string
	NullFiltered  bool
}

func (t *Table) Column(name string) (*Column, int) {
	for i, c := range t.Columns {
		if strings.EqualFold(c.Name, name) {
			return c, i
		}
	}
	return nil, -1
}

type View struct {
	Name  ObjectName
	Query string
}

type PropertyGraph struct {
	Name       string
	RawSQL     string
	NodeTables []*GraphElement
	EdgeTables []*GraphElement
}

type Sequence struct {
	Name ObjectName
}

type Model struct {
	Name    string
	Inputs  []*ModelColumn
	Outputs []*ModelColumn
}

type ModelColumn struct {
	Name string
	Type *TypeSpec
}

type GraphElement struct {
	Name              string
	Alias             string
	KeyColumns        []string
	Source            *GraphEndpoint
	Destination       *GraphEndpoint
	Labels            []*GraphLabelProperties
	Properties        *GraphProperties
	PropertiesSQL     string
	DynamicLabel      string
	DynamicProperties string
}

type GraphEndpoint struct {
	KeyColumns       []string
	ElementReference string
	ReferenceColumns []string
}

type GraphLabelProperties struct {
	Name          string
	Default       bool
	Properties    *GraphProperties
	PropertiesSQL string
}

type GraphProperties struct {
	NoProperties      bool
	AllColumns        bool
	ExceptColumns     []string
	DerivedProperties []*GraphDerivedProperty
}

type GraphDerivedProperty struct {
	Name string
	SQL  string
}

type Column struct {
	Name                 string
	Type                 *TypeSpec
	NotNull              bool
	Hidden               bool
	PrimaryKey           bool
	DefaultSQL           string
	GeneratedSQL         string
	OnUpdateSQL          string
	AllowCommitTimestamp bool
}

type KeyPart struct {
	Name string
	Desc bool
}

// TypeSpec is this package's internal schema type representation.
//
// It intentionally sits between Spanner DDL, the GoogleSQL frontend type
// system, and spannerpb.Type. Spanner DDL contains details that either are not
// representable in ResultSet Type metadata or are needed before descriptor
// resolution, such as TOKENLIST columns, STRING/BYTES length modifiers,
// STRING(MAX), and named proto or enum references from PROTO BUNDLE. Keeping
// those details here lets the catalog build GoogleSQL analyzer types first and
// convert only final analyzer results to spannerpb.Type.
type TypeSpec struct {
	Code         spannerpb.TypeCode
	Tokenlist    bool
	ArrayElement *TypeSpec
	StructFields []StructField
	ProtoTypeFQN string
	Length       *int64
	Max          bool
}

type StructField struct {
	Name string
	Type *TypeSpec
}

func BuildSchemaCatalog(path, ddlSQL string) (*Catalog, error) {
	ddls, err := memefish.ParseDDLs(path, ddlSQL)
	if err != nil {
		return nil, err
	}
	catalog := &Catalog{
		Tables:         map[string]*Table{},
		Indexes:        map[string]*Index{},
		Views:          map[string]*View{},
		PropertyGraphs: map[string]*PropertyGraph{},
		Sequences:      map[string]*Sequence{},
		Models:         map[string]*Model{},
		ProtoTypes:     map[string]bool{},
	}
	for _, ddl := range ddls {
		if err := catalog.ApplyDDL(ddl); err != nil {
			return nil, err
		}
	}
	catalog.addInformationSchemaTables()
	catalog.addSpannerSysTables()
	return catalog, nil
}

func (c *Catalog) ApplyDDL(ddl ast.DDL) error {
	switch ddl := ddl.(type) {
	case *ast.CreateTable:
		return c.applyCreateTable(ddl)
	case *ast.AlterTable:
		return c.applyAlterTable(ddl)
	case *ast.DropTable:
		delete(c.Tables, objectNameFromPath(ddl.Name).String())
		return nil
	case *ast.CreateView:
		name := objectNameFromPath(ddl.Name)
		return c.applyCreateView(ddl, name)
	case *ast.DropView:
		c.dropView(objectNameFromPath(ddl.Name).String())
		return nil
	case *ast.CreatePropertyGraph:
		return c.applyCreatePropertyGraph(ddl)
	case *ast.DropPropertyGraph:
		delete(c.PropertyGraphs, ddl.Name.Name)
		return nil
	case *ast.CreateSequence:
		return c.applyCreateSequence(ddl)
	case *ast.AlterSequence:
		// Sequence options do not affect query result row types. The sequence
		// object itself is enough for GET_NEXT_SEQUENCE_VALUE analysis.
		return nil
	case *ast.DropSequence:
		delete(c.Sequences, objectNameFromPath(ddl.Name).String())
		return nil
	case *ast.CreateModel:
		return c.applyCreateModel(ddl)
	case *ast.AlterModel:
		// Model options do not affect query row types. Keep the registered
		// model input/output schema unchanged.
		return nil
	case *ast.DropModel:
		delete(c.Models, ddl.Name.Name)
		return nil
	case *ast.CreateProtoBundle:
		return c.applyCreateProtoBundle(ddl)
	case *ast.AlterProtoBundle:
		return c.applyAlterProtoBundle(ddl)
	case *ast.DropProtoBundle:
		clear(c.ProtoTypes)
		return nil
	case *ast.CreateIndex:
		return c.applyCreateIndex(ddl)
	case *ast.AlterIndex:
		return c.applyAlterIndex(ddl)
	case *ast.DropIndex:
		delete(c.Indexes, objectNameFromPath(ddl.Name).String())
		return nil
	case *ast.CreateVectorIndex, *ast.DropVectorIndex:
		// Vector indexes do not affect logical query result row types. Regular
		// index metadata is retained for query code generation.
		return nil
	case *ast.CreateSearchIndex, *ast.AlterSearchIndex, *ast.DropSearchIndex:
		// Search indexes accelerate full-text search execution and do not
		// affect logical query result row types.
		return nil
	case *ast.CreateSchema, *ast.DropSchema:
		// Schemas only scope object names; individual objects carry their full
		// paths in this catalog.
		return nil
	case *ast.CreateChangeStream, *ast.AlterChangeStream, *ast.DropChangeStream:
		// Change streams are not visible to ordinary queries.
		return nil
	case *ast.AlterDatabase:
		// Database options do not affect query row types.
		return nil
	case *ast.CreateRole, *ast.DropRole, *ast.Grant, *ast.Revoke:
		// IAM objects and permissions are out of scope for schema analysis.
		return nil
	case *ast.RenameTable:
		return c.applyRenameTable(ddl)
	default:
		return fmt.Errorf("unsupported DDL %T", ddl)
	}
}

func (c *Catalog) applyCreateIndex(ddl *ast.CreateIndex) error {
	name := objectNameFromPath(ddl.Name)
	key := name.String()
	if _, ok := c.Indexes[key]; ok {
		if ddl.IfNotExists {
			return nil
		}
		return fmt.Errorf("index %s already exists", name)
	}
	index := &Index{
		Name:         name,
		TableName:    objectNameFromPath(ddl.TableName),
		Keys:         make([]*KeyPart, 0, len(ddl.Keys)),
		NullFiltered: ddl.NullFiltered,
	}
	for _, key := range ddl.Keys {
		index.Keys = append(index.Keys, keyPartFromIndexKey(key))
	}
	if ddl.Storing != nil {
		index.StoredColumns = identNames(ddl.Storing.Columns)
	}
	c.Indexes[key] = index
	return nil
}

func (c *Catalog) applyAlterIndex(ddl *ast.AlterIndex) error {
	name := objectNameFromPath(ddl.Name)
	index, ok := c.Indexes[name.String()]
	if !ok {
		return fmt.Errorf("index %s does not exist", name)
	}
	switch alt := ddl.IndexAlteration.(type) {
	case *ast.AddStoredColumn:
		if !containsName(index.StoredColumns, alt.Name.Name) {
			index.StoredColumns = append(index.StoredColumns, alt.Name.Name)
		}
		return nil
	case *ast.DropStoredColumn:
		index.StoredColumns = filterNames(index.StoredColumns, alt.Name.Name)
		return nil
	default:
		return fmt.Errorf("unsupported ALTER INDEX %s alteration %T", name, alt)
	}
}

func (c *Catalog) applyCreateModel(ddl *ast.CreateModel) error {
	name := ddl.Name.Name
	if _, ok := c.Models[name]; ok {
		if ddl.IfNotExists {
			return nil
		}
		if !ddl.OrReplace {
			return fmt.Errorf("model %s already exists", name)
		}
	}
	model := &Model{Name: name}
	if ddl.InputOutput != nil {
		inputs, err := modelColumnsFromDDL(ddl.InputOutput.InputColumns)
		if err != nil {
			return err
		}
		outputs, err := modelColumnsFromDDL(ddl.InputOutput.OutputColumns)
		if err != nil {
			return err
		}
		model.Inputs = inputs
		model.Outputs = outputs
	}
	c.Models[name] = model
	return nil
}

func modelColumnsFromDDL(columns []*ast.CreateModelColumn) ([]*ModelColumn, error) {
	out := make([]*ModelColumn, 0, len(columns))
	for _, column := range columns {
		spec, err := schemaTypeToTypeSpec(column.DataType)
		if err != nil {
			return nil, fmt.Errorf("model column %s: %w", column.Name.Name, err)
		}
		out = append(out, &ModelColumn{Name: column.Name.Name, Type: spec})
	}
	return out, nil
}

func (c *Catalog) applyCreateSequence(ddl *ast.CreateSequence) error {
	name := objectNameFromPath(ddl.Name)
	key := name.String()
	if _, ok := c.Sequences[key]; ok {
		if ddl.IfNotExists {
			return nil
		}
		return fmt.Errorf("sequence %s already exists", name)
	}
	c.Sequences[key] = &Sequence{Name: name}
	return nil
}

func (c *Catalog) applyCreateProtoBundle(ddl *ast.CreateProtoBundle) error {
	clear(c.ProtoTypes)
	c.addProtoBundleTypes(ddl.Types)
	return nil
}

func (c *Catalog) applyAlterProtoBundle(ddl *ast.AlterProtoBundle) error {
	if ddl.Insert != nil {
		c.addProtoBundleTypes(ddl.Insert.Types)
	}
	if ddl.Update != nil {
		c.addProtoBundleTypes(ddl.Update.Types)
	}
	if ddl.Delete != nil {
		for _, name := range protoBundleTypeNames(ddl.Delete.Types) {
			delete(c.ProtoTypes, name)
		}
	}
	return nil
}

func (c *Catalog) addProtoBundleTypes(types *ast.ProtoBundleTypes) {
	for _, name := range protoBundleTypeNames(types) {
		c.ProtoTypes[name] = true
	}
}

func protoBundleTypeNames(types *ast.ProtoBundleTypes) []string {
	if types == nil {
		return nil
	}
	names := make([]string, 0, len(types.Types))
	for _, typ := range types.Types {
		names = append(names, normalizeProtoTypeName(identPathString(typ.Path)))
	}
	return names
}

func (c *Catalog) applyCreateView(ddl *ast.CreateView, name ObjectName) error {
	key := name.String()
	if _, ok := c.Tables[key]; ok {
		return fmt.Errorf("view %s conflicts with an existing table", name)
	}
	if _, ok := c.Views[key]; ok && !ddl.OrReplace {
		return fmt.Errorf("view %s already exists", name)
	}
	if _, ok := c.Views[key]; !ok {
		c.ViewOrder = append(c.ViewOrder, key)
	}
	c.Views[key] = &View{Name: name, Query: ddl.Query.SQL()}
	return nil
}

func (c *Catalog) dropView(name string) {
	delete(c.Views, name)
	for i, viewName := range c.ViewOrder {
		if viewName == name {
			c.ViewOrder = append(c.ViewOrder[:i], c.ViewOrder[i+1:]...)
			return
		}
	}
}

func (c *Catalog) applyCreateTable(ddl *ast.CreateTable) error {
	name := objectNameFromPath(ddl.Name)
	if _, ok := c.Tables[name.String()]; ok && !ddl.IfNotExists {
		return fmt.Errorf("table %s already exists", name)
	}
	if _, ok := c.Tables[name.String()]; ok && ddl.IfNotExists {
		return nil
	}
	table := &Table{Name: name}
	for _, syn := range ddl.Synonyms {
		table.Synonyms = append(table.Synonyms, syn.Name.Name)
	}
	for _, def := range ddl.Columns {
		col, err := columnFromDef(def)
		if err != nil {
			return fmt.Errorf("table %s column %s: %w", name, def.Name.Name, err)
		}
		table.Columns = append(table.Columns, col)
		if col.PrimaryKey {
			table.PrimaryKey = append(table.PrimaryKey, &KeyPart{Name: col.Name})
		}
	}
	for _, key := range ddl.PrimaryKeys {
		table.PrimaryKey = append(table.PrimaryKey, keyPartFromIndexKey(key))
		if col, _ := table.Column(key.Name.Name); col != nil {
			col.PrimaryKey = true
		}
	}
	c.Tables[name.String()] = table
	return nil
}

func (c *Catalog) applyAlterTable(ddl *ast.AlterTable) error {
	name := objectNameFromPath(ddl.Name)
	table, ok := c.Tables[name.String()]
	if !ok {
		return fmt.Errorf("table %s does not exist", name)
	}
	switch alt := ddl.TableAlteration.(type) {
	case *ast.AddColumn:
		if existing, _ := table.Column(alt.Column.Name.Name); existing != nil {
			if alt.IfNotExists {
				return nil
			}
			return fmt.Errorf("column %s.%s already exists", name, alt.Column.Name.Name)
		}
		col, err := columnFromDef(alt.Column)
		if err != nil {
			return fmt.Errorf("table %s column %s: %w", name, alt.Column.Name.Name, err)
		}
		table.Columns = append(table.Columns, col)
		return nil
	case *ast.DropColumn:
		_, idx := table.Column(alt.Name.Name)
		if idx < 0 {
			return fmt.Errorf("column %s.%s does not exist", name, alt.Name.Name)
		}
		table.Columns = append(table.Columns[:idx], table.Columns[idx+1:]...)
		table.PrimaryKey = filterPrimaryKey(table.PrimaryKey, alt.Name.Name)
		return nil
	case *ast.AlterColumn:
		col, _ := table.Column(alt.Name.Name)
		if col == nil {
			return fmt.Errorf("column %s.%s does not exist", name, alt.Name.Name)
		}
		return applyColumnAlteration(col, alt.Alteration)
	case *ast.AddSynonym:
		return table.addSynonym(alt.Name.Name)
	case *ast.DropSynonym:
		return table.dropSynonym(alt.Name.Name)
	case *ast.RenameTo:
		return c.renameTable(name.String(), renamedObjectName(name, alt.Name.Name), alt.AddSynonym)
	default:
		return fmt.Errorf("unsupported ALTER TABLE %s alteration %T", name, alt)
	}
}

func (c *Catalog) applyRenameTable(ddl *ast.RenameTable) error {
	for _, rename := range ddl.Tos {
		if err := c.renameTable(rename.Old.Name, ObjectName{Parts: []string{rename.New.Name}}, nil); err != nil {
			return err
		}
	}
	return nil
}

func (c *Catalog) renameTable(oldKey string, newName ObjectName, addSynonym *ast.AddSynonym) error {
	table, ok := c.Tables[oldKey]
	if !ok {
		return fmt.Errorf("table %s does not exist", oldKey)
	}
	newKey := newName.String()
	if oldKey != newKey {
		if _, exists := c.Tables[newKey]; exists {
			return fmt.Errorf("table %s already exists", newName)
		}
		delete(c.Tables, oldKey)
		table.Name = newName
		c.Tables[newKey] = table
	}
	if addSynonym != nil {
		table.Synonyms = []string{addSynonym.Name.Name}
	}
	return nil
}

func (t *Table) addSynonym(name string) error {
	for _, syn := range t.Synonyms {
		if strings.EqualFold(syn, name) {
			return nil
		}
	}
	if len(t.Synonyms) > 0 {
		return fmt.Errorf("table %s already has synonym %s", t.Name, t.Synonyms[0])
	}
	t.Synonyms = append(t.Synonyms, name)
	return nil
}

func (t *Table) dropSynonym(name string) error {
	for i, syn := range t.Synonyms {
		if strings.EqualFold(syn, name) {
			t.Synonyms = append(t.Synonyms[:i], t.Synonyms[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("table %s synonym %s does not exist", t.Name, name)
}

func renamedObjectName(old ObjectName, newLeaf string) ObjectName {
	if len(old.Parts) == 0 {
		return ObjectName{Parts: []string{newLeaf}}
	}
	parts := append([]string(nil), old.Parts...)
	parts[len(parts)-1] = newLeaf
	return ObjectName{Parts: parts}
}

func columnFromDef(def *ast.ColumnDef) (*Column, error) {
	spec, err := schemaTypeToTypeSpec(def.Type)
	if err != nil {
		return nil, err
	}
	col := &Column{
		Name:       def.Name.Name,
		Type:       spec,
		NotNull:    def.NotNull,
		Hidden:     !def.Hidden.Invalid(),
		PrimaryKey: def.PrimaryKey,
	}
	switch sem := def.DefaultSemantics.(type) {
	case nil:
	case *ast.ColumnDefaultExpr:
		col.DefaultSQL = sem.Expr.SQL()
		if sem.OnUpdate != nil {
			col.OnUpdateSQL = sem.OnUpdate.Expr.SQL()
		}
	case *ast.GeneratedColumnExpr:
		col.GeneratedSQL = sem.Expr.SQL()
	case *ast.IdentityColumn:
		col.GeneratedSQL = sem.SQL()
	default:
		return nil, fmt.Errorf("unsupported column default semantics %T", sem)
	}
	applyColumnOptions(col, def.Options)
	return col, nil
}

func applyColumnAlteration(col *Column, alt ast.ColumnAlteration) error {
	switch alt := alt.(type) {
	case *ast.AlterColumnType:
		spec, err := schemaTypeToTypeSpec(alt.Type)
		if err != nil {
			return err
		}
		col.Type = spec
		col.NotNull = alt.NotNull
		if alt.DefaultExpr != nil {
			col.DefaultSQL = alt.DefaultExpr.Expr.SQL()
			if alt.DefaultExpr.OnUpdate != nil {
				col.OnUpdateSQL = alt.DefaultExpr.OnUpdate.Expr.SQL()
			}
		}
		return nil
	case *ast.AlterColumnSetDefault:
		col.DefaultSQL = alt.DefaultExpr.Expr.SQL()
		if alt.DefaultExpr.OnUpdate != nil {
			col.OnUpdateSQL = alt.DefaultExpr.OnUpdate.Expr.SQL()
		}
		return nil
	case *ast.AlterColumnDropDefault:
		col.DefaultSQL = ""
		return nil
	case *ast.AlterColumnSetOnUpdate:
		col.OnUpdateSQL = alt.OnUpdate.Expr.SQL()
		return nil
	case *ast.AlterColumnDropOnUpdate:
		col.OnUpdateSQL = ""
		return nil
	case *ast.AlterColumnSetOptions:
		applyColumnOptions(col, alt.Options)
		return nil
	default:
		return fmt.Errorf("unsupported ALTER COLUMN alteration %T", alt)
	}
}

func applyColumnOptions(col *Column, options *ast.Options) {
	if options == nil {
		return
	}
	for _, opt := range options.Records {
		if strings.EqualFold(opt.Name.Name, "allow_commit_timestamp") {
			col.AllowCommitTimestamp = strings.EqualFold(opt.Value.SQL(), "TRUE")
		}
	}
}

func schemaTypeToTypeSpec(t ast.SchemaType) (*TypeSpec, error) {
	switch t := t.(type) {
	case *ast.ScalarSchemaType:
		return scalarTypeToTypeSpec(t.Name, nil, false)
	case *ast.SizedSchemaType:
		var length *int64
		if !t.Max {
			v, err := intValue(t.Size)
			if err != nil {
				return nil, err
			}
			length = &v
		}
		return scalarTypeToTypeSpec(t.Name, length, t.Max)
	case *ast.ArraySchemaType:
		elem, err := schemaTypeToTypeSpec(t.Item)
		if err != nil {
			return nil, err
		}
		return &TypeSpec{Code: spannerpb.TypeCode_ARRAY, ArrayElement: elem}, nil
	case *ast.StructType:
		fields := make([]StructField, 0, len(t.Fields))
		for _, field := range t.Fields {
			spec, err := astTypeToTypeSpec(field.Type)
			if err != nil {
				return nil, err
			}
			name := ""
			if field.Ident != nil {
				name = field.Ident.Name
			}
			fields = append(fields, StructField{Name: name, Type: spec})
		}
		return &TypeSpec{Code: spannerpb.TypeCode_STRUCT, StructFields: fields}, nil
	case *ast.NamedType:
		return &TypeSpec{Code: spannerpb.TypeCode_PROTO, ProtoTypeFQN: normalizeProtoTypeName(identPathString(t.Path))}, nil
	default:
		return nil, fmt.Errorf("unsupported schema type %T", t)
	}
}

func astTypeToTypeSpec(t any) (*TypeSpec, error) {
	switch t := t.(type) {
	case ast.SchemaType:
		return schemaTypeToTypeSpec(t)
	case ast.Type:
		return typeToTypeSpec(t)
	default:
		return nil, fmt.Errorf("unsupported schema type %T", t)
	}
}

func scalarTypeToTypeSpec(name ast.ScalarTypeName, length *int64, max bool) (*TypeSpec, error) {
	if name == ast.TokenListTypeName {
		return &TypeSpec{Tokenlist: true}, nil
	}
	code, err := scalarTypeCode(name)
	if err != nil {
		return nil, err
	}
	return &TypeSpec{Code: code, Length: length, Max: max}, nil
}

func scalarTypeCode(name ast.ScalarTypeName) (spannerpb.TypeCode, error) {
	switch name {
	case ast.BoolTypeName:
		return spannerpb.TypeCode_BOOL, nil
	case ast.Int64TypeName:
		return spannerpb.TypeCode_INT64, nil
	case ast.Float32TypeName:
		return spannerpb.TypeCode_FLOAT32, nil
	case ast.Float64TypeName:
		return spannerpb.TypeCode_FLOAT64, nil
	case ast.StringTypeName:
		return spannerpb.TypeCode_STRING, nil
	case ast.BytesTypeName:
		return spannerpb.TypeCode_BYTES, nil
	case ast.DateTypeName:
		return spannerpb.TypeCode_DATE, nil
	case ast.TimestampTypeName:
		return spannerpb.TypeCode_TIMESTAMP, nil
	case ast.NumericTypeName:
		return spannerpb.TypeCode_NUMERIC, nil
	case ast.JSONTypeName:
		return spannerpb.TypeCode_JSON, nil
	case ast.IntervalTypeName:
		return spannerpb.TypeCode_INTERVAL, nil
	default:
		return spannerpb.TypeCode_TYPE_CODE_UNSPECIFIED, fmt.Errorf("unsupported scalar type %s", name)
	}
}

func intValue(v ast.IntValue) (int64, error) {
	switch v := v.(type) {
	case *ast.IntLiteral:
		return strconv.ParseInt(v.Value, v.Base, 64)
	default:
		return 0, fmt.Errorf("unsupported size expression %T", v)
	}
}

func filterPrimaryKey(keys []*KeyPart, name string) []*KeyPart {
	out := keys[:0]
	for _, key := range keys {
		if !strings.EqualFold(key.Name, name) {
			out = append(out, key)
		}
	}
	return out
}

func filterNames(names []string, name string) []string {
	out := names[:0]
	for _, value := range names {
		if !strings.EqualFold(value, name) {
			out = append(out, value)
		}
	}
	return out
}

func containsName(names []string, name string) bool {
	for _, value := range names {
		if strings.EqualFold(value, name) {
			return true
		}
	}
	return false
}

func identNames(idents []*ast.Ident) []string {
	names := make([]string, 0, len(idents))
	for _, ident := range idents {
		names = append(names, ident.Name)
	}
	return names
}

func keyPartFromIndexKey(key *ast.IndexKey) *KeyPart {
	return &KeyPart{
		Name: key.Name.Name,
		Desc: strings.EqualFold(string(key.Dir), "DESC"),
	}
}

func objectNameFromPath(path *ast.Path) ObjectName {
	parts := make([]string, 0, len(path.Idents))
	for _, ident := range path.Idents {
		parts = append(parts, ident.Name)
	}
	return ObjectName{Parts: parts}
}

func identPathString(path []*ast.Ident) string {
	parts := make([]string, 0, len(path))
	for _, ident := range path {
		parts = append(parts, ident.Name)
	}
	return strings.Join(parts, ".")
}
