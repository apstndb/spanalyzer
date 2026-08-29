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
			IndexState: strPtr("READ_WRITE"),
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
