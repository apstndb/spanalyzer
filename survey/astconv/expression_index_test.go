package astconv

import (
	"strings"
	"testing"

	"github.com/apstndb/spanalyzer/survey/infoschem"
)

func TestToDDLStatementsRejectsExpressionIndex(t *testing.T) {
	t.Parallel()

	ordinal := int64(1)
	ordering := "ASC"
	expression := "(JSON_VALUE(VenueData.address.city))"
	schema := &Schema{
		Indexes: []*infoschem.Index{{
			TableName:  "Venues",
			IndexName:  "VenuesByCity",
			IndexType:  "INDEX",
			IndexState: new("READ_WRITE"),
		}},
		IndexColumns: []*infoschem.IndexColumn{{
			TableName:       "Venues",
			IndexName:       "VenuesByCity",
			IndexType:       "INDEX",
			ColumnName:      "_ExpressionIndex_VenuesByCity_0",
			OrdinalPosition: &ordinal,
			ColumnOrdering:  &ordering,
			Expression:      &expression,
		}},
	}

	_, err := schema.ToDDLStatements()
	if err == nil {
		t.Fatal("ToDDLStatements() error = nil, want expression-index boundary")
	}
	for _, want := range []string{
		`unsupported expression index "VenuesByCity"`,
		`index key "_ExpressionIndex_VenuesByCity_0"`,
		`current memefish AST cannot represent`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ToDDLStatements() error = %q, want fragment %q", err, want)
		}
	}
}

func TestExpressionIndexWithoutKeyMetadataFailsClosed(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		schema := &Schema{
			Indexes: []*infoschem.Index{{TableName: "Venues", IndexName: "ByCity", IndexType: "INDEX"}},
			Columns: []*infoschem.Column{{TableName: "Venues", ColumnName: "_ExpressionIndex_ByCity_0", IsHidden: true, GenerationExpression: new("JSON_VALUE(Data.city)")}},
		}
		if mixed {
			schema.IndexColumns = []*infoschem.IndexColumn{{TableName: "Venues", IndexName: "ByCity", ColumnName: "Name", OrdinalPosition: new(int64(1)), ColumnOrdering: new("ASC")}}
		}
		_, err := schema.ToDDLStatements()
		if err == nil || !strings.Contains(err.Error(), "has no index-column metadata") {
			t.Fatalf("mixed=%t reconstruction error = %v", mixed, err)
		}
	}
}

func TestExplicitHiddenGeneratedKeyIsNotRejectedByNameAlone(t *testing.T) {
	schema := &Schema{
		Indexes:      []*infoschem.Index{{TableName: "Venues", IndexName: "ByCity", IndexType: "INDEX"}},
		Columns:      []*infoschem.Column{{TableName: "Venues", ColumnName: "_ExpressionIndex_ByCity_0", IsHidden: true, GenerationExpression: new("JSON_VALUE(Data.city)")}},
		IndexColumns: []*infoschem.IndexColumn{{TableName: "Venues", IndexName: "ByCity", ColumnName: "_ExpressionIndex_ByCity_0", OrdinalPosition: new(int64(1)), ColumnOrdering: new("ASC")}},
	}
	if _, err := schema.ToDDLStatements(); err != nil {
		t.Fatalf("explicit ordinary index key with complete metadata: %v", err)
	}
}
