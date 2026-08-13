package main

import (
	"strings"
	"testing"
)

func TestAIPlanCases(t *testing.T) {
	queries, err := loadQueries("ai_plan", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(queries), 6; got != want {
		t.Fatalf("AI PLAN query count = %d, want %d", got, want)
	}
	seen := make(map[string]string, len(queries))
	for _, query := range queries {
		if _, duplicate := seen[query.Label]; duplicate {
			t.Errorf("duplicate AI PLAN label %q", query.Label)
		}
		seen[query.Label] = query.SQL
	}
	for label, function := range map[string]string{
		"ai-plan/classify-projection": "AI.CLASSIFY(",
		"ai-plan/if-filter":           "AI.IF(",
		"ai-plan/score-order-limit":   "AI.SCORE(",
	} {
		if !strings.Contains(seen[label], function) {
			t.Errorf("%s missing %s", label, function)
		}
	}
}
