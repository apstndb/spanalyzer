package astconv

import (
	"strings"
	"testing"

	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func TestRoundtripVectorIndexFilter(t *testing.T) {
	stmt, err := memefish.ParseDDL("", `CREATE VECTOR INDEX TechDocEmbeddingIndex
  ON Documents(DocEmbedding)
  STORING (NullIfFiltered)
  WHERE DocEmbedding IS NOT NULL AND NullIfFiltered IS NOT NULL
  OPTIONS (distance_type = 'COSINE')`)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}

	schema, err := FromDDLStatements([]ast.DDL{stmt})
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}
	if got := schema.Indexes[0].Filter; got == nil || *got != "DocEmbedding IS NOT NULL AND NullIfFiltered IS NOT NULL" {
		t.Fatalf("Filter = %v", got)
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	vectorIndex, ok := ddls[0].(*ast.CreateVectorIndex)
	if !ok {
		t.Fatalf("DDL type = %T, want *ast.CreateVectorIndex", ddls[0])
	}
	if vectorIndex.Where == nil || vectorIndex.Where.Expr.SQL() != "DocEmbedding IS NOT NULL AND NullIfFiltered IS NOT NULL" {
		t.Fatalf("Where = %#v", vectorIndex.Where)
	}
}

func TestVectorIndexDDLFiltersSystemOptimizedOptions(t *testing.T) {
	for _, tc := range []struct{ name, options string }{
		{"automatic", "distance_type = 'COSINE'"},
		{"explicit", "distance_type = 'EUCLIDEAN', tree_depth = 3, num_branches = 2, num_leaves = 4"},
		{"uppercase_option", "DISTANCE_TYPE = 'DOT_PRODUCT'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmt, err := memefish.ParseDDL("", "CREATE VECTOR INDEX Vec ON Docs(Embedding) WHERE Embedding IS NOT NULL OPTIONS ("+tc.options+")")
			if err != nil {
				t.Fatal(err)
			}
			schema, err := FromDDLStatements([]ast.DDL{stmt})
			if err != nil {
				t.Fatal(err)
			}
			// Live GoogleSQL catalogs identify vector indexes as VECTOR.
			schema.Indexes[0].IndexType = "VECTOR"
			schema.IndexColumns[0].IndexType = "VECTOR"
			for _, o := range schema.IndexOptions {
				o.IndexType = "VECTOR"
			}
			for _, name := range []string{"system_optimized_tree_depth", "system_optimized_num_branches", "system_optimized_num_leaves"} {
				schema.IndexOptions = append(schema.IndexOptions, &infoschem.IndexOption{
					TableName: "Docs", IndexName: "Vec", IndexType: "VECTOR", OptionName: name, OptionType: "INT64", OptionValue: "2",
				})
			}
			ddls, err := schema.ToDDLStatements()
			if err != nil {
				t.Fatal(err)
			}
			if len(ddls) != 1 {
				t.Fatalf("got %d DDL statements, want 1", len(ddls))
			}
			if got, want := ddls[0].SQL(), stmt.SQL(); got != want {
				t.Errorf("reconstructed DDL = %s; want %s", got, want)
			}
		})
	}
}

func TestToVectorIndexesDDLRejectsMultipleKeyColumns(t *testing.T) {
	ordinal := func(value int64) *int64 { return &value }
	schema := &Schema{
		Indexes: []*infoschem.Index{
			{TableName: "Documents", IndexName: "DocEmbeddingIndexWithKeys", IndexType: "VECTOR_INDEX"},
		},
		IndexColumns: []*infoschem.IndexColumn{
			{TableName: "Documents", IndexName: "DocEmbeddingIndexWithKeys", ColumnName: "DocEmbedding", OrdinalPosition: ordinal(1)},
			{TableName: "Documents", IndexName: "DocEmbeddingIndexWithKeys", ColumnName: "DocName", OrdinalPosition: ordinal(2)},
			{TableName: "Documents", IndexName: "DocEmbeddingIndexWithKeys", ColumnName: "Author", OrdinalPosition: ordinal(3)},
		},
	}

	_, err := schema.ToDDLStatements()
	if err == nil || !strings.Contains(err.Error(), "has 3 key columns") {
		t.Fatalf("ToDDLStatements() error = %v, want multi-key fail-closed error", err)
	}
}

func TestVectorIndexDDLRequiresDistanceType(t *testing.T) {
	for _, tc := range []struct{ name, option string }{
		{"no_options", ""},
		{"effective_setting_only", "system_optimized_tree_depth"},
		{"explicit_tree_depth_only", "tree_depth"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			schema := &Schema{
				Indexes: []*infoschem.Index{{TableName: "Docs", IndexName: "Vec", IndexType: "VECTOR"}},
				IndexColumns: []*infoschem.IndexColumn{{
					TableName: "Docs", IndexName: "Vec", ColumnName: "Embedding", OrdinalPosition: new(int64(1)),
				}},
			}
			if tc.option != "" {
				schema.IndexOptions = []*infoschem.IndexOption{{
					TableName: "Docs", IndexName: "Vec", OptionName: tc.option, OptionType: "INT64", OptionValue: "2",
				}}
			}
			_, err := schema.ToDDLStatements()
			if err == nil || !strings.Contains(err.Error(), `vector index "Vec" has no distance_type`) {
				t.Fatalf("ToDDLStatements() error = %v, want missing distance_type for Vec", err)
			}
		})
	}
}

func TestToVectorIndexesDDLRejectsInvalidFilter(t *testing.T) {
	filter := "NOT VALID SQL ("
	ordinal := int64(1)
	schema := &Schema{
		Indexes: []*infoschem.Index{
			{TableName: "Documents", IndexName: "DocEmbeddingIndex", IndexType: "VECTOR_INDEX", Filter: &filter},
		},
		IndexColumns: []*infoschem.IndexColumn{
			{TableName: "Documents", IndexName: "DocEmbeddingIndex", ColumnName: "DocEmbedding", OrdinalPosition: &ordinal},
		},
	}

	_, err := schema.ToDDLStatements()
	if err == nil || !strings.Contains(err.Error(), "FILTER") {
		t.Fatalf("ToDDLStatements() error = %v, want FILTER parse error", err)
	}
}
