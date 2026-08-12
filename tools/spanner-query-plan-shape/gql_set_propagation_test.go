package main

import (
	"strings"
	"testing"
)

func TestLoadQueriesGQLSetPropagation(t *testing.T) {
	queries, err := loadQueries("gql_set_propagation", nil, nil)
	if err != nil {
		t.Fatalf("loadQueries(%q) error = %v", "gql_set_propagation", err)
	}
	if got, want := len(queries), 7; got != want {
		t.Fatalf("GQL set propagation query count = %d, want %d", got, want)
	}

	queriesByLabel := make(map[string]queryCase, len(queries))
	for _, query := range queries {
		if !strings.HasPrefix(query.Label, "gql-set-propagation/") {
			t.Errorf("query label %q has wrong prefix", query.Label)
		}
		if _, duplicate := queriesByLabel[query.Label]; duplicate {
			t.Errorf("duplicate query label %q", query.Label)
		}
		queriesByLabel[query.Label] = query
	}

	tests := []struct {
		name  string
		label string
		mode  string
	}{
		{name: "full", label: "gql-set-propagation/full", mode: "FULL UNION ALL"},
		{name: "full outer", label: "gql-set-propagation/full-outer", mode: "FULL OUTER UNION ALL"},
		{name: "outer", label: "gql-set-propagation/outer", mode: "OUTER UNION ALL"},
		{name: "inner", label: "gql-set-propagation/inner", mode: "INNER UNION ALL"},
		{name: "left", label: "gql-set-propagation/left", mode: "LEFT UNION ALL"},
		{name: "left outer", label: "gql-set-propagation/left-outer", mode: "LEFT OUTER UNION ALL"},
		{name: "strict control", label: "gql-set-propagation/strict-control", mode: "\nUNION ALL\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, ok := queriesByLabel[tt.label]
			if !ok {
				t.Fatalf("query %q is missing", tt.label)
			}
			if !strings.Contains(query.SQL, tt.mode) {
				t.Errorf("query %q does not contain %q: %s", tt.label, tt.mode, query.SQL)
			}
			for _, column := range []string{"left_only", "shared", "right_only"} {
				if !strings.Contains(query.SQL, column) {
					t.Errorf("query %q does not contain output column %q", tt.label, column)
				}
			}
		})
	}

	ddls, err := loadDDLs("gql_set_propagation", nil)
	if err != nil {
		t.Fatalf("loadDDLs(%q) error = %v", "gql_set_propagation", err)
	}
	if joined := strings.Join(ddls, "\n"); !strings.Contains(joined, "PROPERTY GRAPH MusicGraph") {
		t.Error("GQL set propagation schema is missing MusicGraph")
	}
}
