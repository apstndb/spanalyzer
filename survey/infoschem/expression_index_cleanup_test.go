package infoschem_test

import (
	"slices"
	"testing"
)

// expressionIndexCleanupStatements builds one UpdateDatabaseDdl batch that
// drops discovered secondary indexes first and the fixture table last.
func expressionIndexCleanupStatements(indexNames []string, tableName string) []string {
	statements := make([]string, 0, len(indexNames)+1)
	for _, indexName := range indexNames {
		statements = append(statements, "DROP INDEX "+indexName)
	}
	return append(statements, "DROP TABLE "+tableName)
}

func TestExpressionIndexCleanupStatements(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		indexNames []string
		tableName  string
		want       []string
	}{
		{
			name:      "table only when no secondary indexes were discovered",
			tableName: "ExpressionIndexProbe_1",
			want:      []string{"DROP TABLE ExpressionIndexProbe_1"},
		},
		{
			name:       "drop discovered indexes in given order then the table",
			indexNames: []string{"ExpressionIndexProbe_1ByCity", "ExpressionIndexProbe_1ByNameState", "ExpressionIndexProbe_1ByCityNoWrap"},
			tableName:  "ExpressionIndexProbe_1",
			want: []string{
				"DROP INDEX ExpressionIndexProbe_1ByCity",
				"DROP INDEX ExpressionIndexProbe_1ByNameState",
				"DROP INDEX ExpressionIndexProbe_1ByCityNoWrap",
				"DROP TABLE ExpressionIndexProbe_1",
			},
		},
		{
			name:       "include an unexpectedly accepted rejection-probe index",
			indexNames: []string{"ExpressionIndexProbe_1ByCity", "ExpressionIndexProbe_1ByCityDesc"},
			tableName:  "ExpressionIndexProbe_1",
			want: []string{
				"DROP INDEX ExpressionIndexProbe_1ByCity",
				"DROP INDEX ExpressionIndexProbe_1ByCityDesc",
				"DROP TABLE ExpressionIndexProbe_1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := expressionIndexCleanupStatements(tt.indexNames, tt.tableName)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("expressionIndexCleanupStatements(%q, %q) = %#v, want %#v", tt.indexNames, tt.tableName, got, tt.want)
			}
		})
	}
}
