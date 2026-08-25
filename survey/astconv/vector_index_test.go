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
