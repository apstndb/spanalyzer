package astconv

import (
	"slices"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func TestToGraphsDDL_PropertyGraphMetadata(t *testing.T) {
	s := propertyGraphTestSchema()
	s.PropertyGraphs = []*infoschem.PropertyGraph{
		{
			PropertyGraphName: "FinGraph",
			PropertyGraphMetadataJSON: spanner.NullJSON{
				Valid: true,
				Value: &propertyGraphMetadata{
					Name: "FinGraph",
					NodeTables: []*propertyGraphElementTable{
						{
							Name:          "Account",
							Kind:          "NODE",
							BaseTableName: "Account",
							KeyColumns:    []string{"id"},
							LabelNames:    []string{"Account"},
							PropertyDefinitions: []*propertyGraphPropertyDef{
								{PropertyDeclarationName: "id", ValueExpressionSQL: "id"},
								{PropertyDeclarationName: "nick", ValueExpressionSQL: "nick"},
							},
						},
						{
							Name:          "Person",
							Kind:          "NODE",
							BaseTableName: "Person",
							KeyColumns:    []string{"id"},
							LabelNames:    []string{"Client"},
							PropertyDefinitions: []*propertyGraphPropertyDef{
								{PropertyDeclarationName: "id", ValueExpressionSQL: "id"},
								{PropertyDeclarationName: "person_name", ValueExpressionSQL: "name"},
							},
						},
					},
					EdgeTables: []*propertyGraphElementTable{
						{
							Name:          "Own",
							Kind:          "EDGE",
							BaseTableName: "Own",
							KeyColumns:    []string{"id", "account_id"},
							LabelNames:    []string{"Ownership"},
							SourceNodeTable: &propertyGraphNodeTableReference{
								NodeTableName:    "Person",
								EdgeTableColumns: []string{"id"},
								NodeTableColumns: []string{"id"},
							},
							DestinationNodeTable: &propertyGraphNodeTableReference{
								NodeTableName:    "Account",
								EdgeTableColumns: []string{"account_id"},
								NodeTableColumns: []string{"id"},
							},
						},
					},
					Labels: []*propertyGraphElementLabel{
						{Name: "Account", PropertyDeclarationNames: []string{"id", "nick"}},
						{Name: "Client", PropertyDeclarationNames: []string{"id", "person_name"}},
						{Name: "Ownership"},
					},
					PropertyDeclarations: []*propertyGraphPropertyDecl{
						{Name: "id", Type: "INT64"},
						{Name: "nick", Type: "STRING(MAX)"},
						{Name: "person_name"},
					},
				},
			},
		},
	}

	ddls, err := s.toGraphsDDL()
	if err != nil {
		t.Fatalf("toGraphsDDL: %v", err)
	}
	if len(ddls) != 1 {
		t.Fatalf("got %d DDLs, want 1", len(ddls))
	}

	cpg, ok := ddls[0].(*ast.CreatePropertyGraph)
	if !ok {
		t.Fatalf("DDL type = %T, want *ast.CreatePropertyGraph", ddls[0])
	}
	got := cpg.SQL()
	want := "CREATE PROPERTY GRAPH FinGraph NODE TABLES (Account, Person LABEL Client PROPERTIES (id, name AS person_name)) EDGE TABLES (Own SOURCE KEY (id) REFERENCES Person DESTINATION KEY (account_id) REFERENCES Account LABEL Ownership NO PROPERTIES)"
	if got != want {
		t.Fatalf("graph SQL = %q, want %q", got, want)
	}
}

func TestToGraphsDDL_EmptyLabelPropertyListMeansNoProperties(t *testing.T) {
	s := propertyGraphTestSchema()
	s.PropertyGraphs = []*infoschem.PropertyGraph{
		{
			PropertyGraphName: "FinGraph",
			PropertyGraphMetadataJSON: spanner.NullJSON{
				Valid: true,
				Value: &propertyGraphMetadata{
					Name: "FinGraph",
					NodeTables: []*propertyGraphElementTable{
						{
							Name:          "Account",
							Kind:          "NODE",
							BaseTableName: "Account",
							KeyColumns:    []string{"id"},
							LabelNames:    []string{"Public"},
							PropertyDefinitions: []*propertyGraphPropertyDef{
								{PropertyDeclarationName: "id", ValueExpressionSQL: "id"},
								{PropertyDeclarationName: "nick", ValueExpressionSQL: "nick"},
							},
						},
					},
					Labels: []*propertyGraphElementLabel{{Name: "Public"}},
				},
			},
		},
	}

	ddls, err := s.toGraphsDDL()
	if err != nil {
		t.Fatalf("toGraphsDDL: %v", err)
	}
	got := ddls[0].SQL()
	want := "CREATE PROPERTY GRAPH FinGraph NODE TABLES (Account LABEL Public NO PROPERTIES)"
	if got != want {
		t.Fatalf("graph SQL = %q, want %q", got, want)
	}
}

func TestRoundtrip_PropertyGraphDynamicMetadata(t *testing.T) {
	ddls := []string{
		`CREATE TABLE GraphNode (
  id INT64 NOT NULL,
  label STRING(MAX) NOT NULL,
  properties JSON,
) PRIMARY KEY (id)`,
		`CREATE TABLE GraphEdge (
  id INT64 NOT NULL,
  dest_id INT64 NOT NULL,
  label STRING(MAX) NOT NULL,
  properties JSON,
) PRIMARY KEY (id, dest_id)`,
		`CREATE PROPERTY GRAPH FinGraph
  NODE TABLES (
    GraphNode DYNAMIC LABEL (label) DYNAMIC PROPERTIES (properties)
  )
  EDGE TABLES (
    GraphEdge
      SOURCE KEY (id) REFERENCES GraphNode
      DESTINATION KEY (dest_id) REFERENCES GraphNode
      DYNAMIC LABEL (label)
      DYNAMIC PROPERTIES (properties)
  )`,
	}

	var stmts []ast.DDL
	for _, ddl := range ddls {
		stmt, err := memefish.ParseDDL("", ddl)
		if err != nil {
			t.Fatalf("ParseDDL(%q): %v", ddl, err)
		}
		stmts = append(stmts, stmt)
	}

	schema, err := FromDDLStatements(stmts)
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}
	if len(schema.PropertyGraphs) != 1 {
		t.Fatalf("property graphs = %d, want 1", len(schema.PropertyGraphs))
	}
	if !schema.PropertyGraphs[0].PropertyGraphMetadataJSON.Valid {
		t.Fatal("PropertyGraphMetadataJSON should be valid")
	}

	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var found string
	for _, d := range reconDDLs {
		if cpg, ok := d.(*ast.CreatePropertyGraph); ok {
			found = cpg.SQL()
			break
		}
	}
	if found == "" {
		t.Fatal("reconstructed property graph not found")
	}

	want := "CREATE PROPERTY GRAPH FinGraph NODE TABLES (GraphNode DYNAMIC LABEL (label) DYNAMIC PROPERTIES (properties)) EDGE TABLES (GraphEdge SOURCE KEY (id) REFERENCES GraphNode DESTINATION KEY (dest_id) REFERENCES GraphNode DYNAMIC LABEL (label) DYNAMIC PROPERTIES (properties))"
	if found != want {
		t.Fatalf("graph SQL = %q, want %q", found, want)
	}
}

func TestRoundtrip_PropertyGraphBackedByView(t *testing.T) {
	ddls := []string{
		`CREATE TABLE GraphNode (
  id INT64 NOT NULL,
  name STRING(MAX),
) PRIMARY KEY (id)`,
		`CREATE VIEW GraphView SQL SECURITY INVOKER AS
  SELECT id, name AS display_name FROM GraphNode`,
		`CREATE PROPERTY GRAPH FinGraph
  NODE TABLES (GraphView KEY (id))`,
	}

	var stmts []ast.DDL
	for _, ddl := range ddls {
		stmt, err := memefish.ParseDDL("", ddl)
		if err != nil {
			t.Fatalf("ParseDDL(%q): %v", ddl, err)
		}
		stmts = append(stmts, stmt)
	}

	schema, err := FromDDLStatements(stmts)
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}
	if got := tableColumnNames(schema, "GraphView"); !slices.Equal(got, []string{"id", "display_name"}) {
		t.Fatalf("GraphView columns = %v", got)
	}

	reconstructed, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	var graphSQL string
	for _, ddl := range reconstructed {
		if graph, ok := ddl.(*ast.CreatePropertyGraph); ok {
			graphSQL = graph.SQL()
		}
	}
	if got, want := graphSQL, "CREATE PROPERTY GRAPH FinGraph NODE TABLES (GraphView KEY (id))"; got != want {
		t.Fatalf("graph SQL = %q, want %q", got, want)
	}
}

func TestExplicitViewColumnNamesFailsClosed(t *testing.T) {
	for _, ddl := range []string{
		"CREATE VIEW StarView SQL SECURITY INVOKER AS SELECT * FROM GraphNode",
		"CREATE VIEW ExprView SQL SECURITY INVOKER AS SELECT id + 1 FROM GraphNode",
	} {
		stmt, err := memefish.ParseDDL("", ddl)
		if err != nil {
			t.Fatalf("ParseDDL(%q): %v", ddl, err)
		}
		view := stmt.(*ast.CreateView)
		if names, ok := explicitViewColumnNames(view.Query); ok || names != nil {
			t.Errorf("explicitViewColumnNames(%q) = %v, %v; want nil, false", ddl, names, ok)
		}
	}
}

func propertyGraphTestSchema() *Schema {
	return &Schema{
		Columns: []*infoschem.Column{
			{TableName: "Account", ColumnName: "id", OrdinalPosition: 1, SpannerType: "INT64"},
			{TableName: "Account", ColumnName: "nick", OrdinalPosition: 2, SpannerType: "STRING(MAX)"},
			{TableName: "Person", ColumnName: "id", OrdinalPosition: 1, SpannerType: "INT64"},
			{TableName: "Person", ColumnName: "name", OrdinalPosition: 2, SpannerType: "STRING(MAX)"},
			{TableName: "Own", ColumnName: "id", OrdinalPosition: 1, SpannerType: "INT64"},
			{TableName: "Own", ColumnName: "account_id", OrdinalPosition: 2, SpannerType: "INT64"},
		},
		IndexColumns: []*infoschem.IndexColumn{
			{TableName: "Account", IndexName: "PRIMARY_KEY", OrdinalPosition: int64Ptr(1), ColumnName: "id"},
			{TableName: "Person", IndexName: "PRIMARY_KEY", OrdinalPosition: int64Ptr(1), ColumnName: "id"},
			{TableName: "Own", IndexName: "PRIMARY_KEY", OrdinalPosition: int64Ptr(1), ColumnName: "id"},
			{TableName: "Own", IndexName: "PRIMARY_KEY", OrdinalPosition: int64Ptr(2), ColumnName: "account_id"},
		},
	}
}
