package plancontract

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestContractValidationDoesNotDependOnTargetAvailability(t *testing.T) {
	tests := []struct {
		name     string
		contract Contract
		want     string
	}{
		{"no mode", Contract{}, "exactly one"},
		{"conflicting modes", Contract{Use: []string{"no_full_sort"}, CEL: "true"}, "exactly one"},
		{"unknown predefined", Contract{Use: []string{"not_a_contract"}}, "unsupported predefined"},
		{"empty family", Contract{Forbid: []Predicate{{}}}, "operator_family is required"},
		{"unknown family", Contract{Forbid: []Predicate{{OperatorFamily: "not_a_family"}}}, "unsupported forbid"},
		{"negative count", Contract{Forbid: []Predicate{{OperatorFamily: "full_sort", MaxCount: -1}}}, "max_count must be >= 0"},
		{"duplicate family", Contract{Forbid: []Predicate{{OperatorFamily: "full_sort"}, {OperatorFamily: " full_sort "}}}, "duplicate_forbid_operator_family"},
		{"invalid CEL", Contract{CEL: "true &&"}, "Syntax error"},
		{"unknown CEL variable", Contract{CEL: "unknown_plan_field == 1"}, "undeclared reference"},
		{"non boolean CEL", Contract{CEL: "42"}, "bool"},
		{"execution statistics", Contract{CEL: "raw_nodes.exists(n, n.execution_stats != null)"}, "execution_stats is not supported"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.contract
			c.Name, c.Target = "Invalid", "query/Q"
			file := File{Version: FileVersionV1Alpha, Contracts: []Contract{c}}
			if err := Validate(file); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Validate() error = %v, want %q", err, tt.want)
			}
			data, err := yaml.Marshal(file)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "contracts.yaml")
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := ReadFile(path); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("ReadFile() error = %v, want %q", err, tt.want)
			}
			for _, status := range []string{"missing", "error", "ok"} {
				t.Run(status, func(t *testing.T) {
					report := Report{}
					if status != "missing" {
						report.Queries = []Query{{Name: "Q", TargetID: "query/Q", Status: status}}
					}
					if _, err := Evaluate(report, file); err == nil || !strings.Contains(err.Error(), tt.want) {
						t.Errorf("Evaluate() error = %v, want %q", err, tt.want)
					}
				})
			}
		})
	}
}

func TestEvaluateValidatesDocumentIdentity(t *testing.T) {
	valid := Contract{Name: "Valid", Target: "query/Q", Use: []string{"no_full_sort"}}
	for _, file := range []File{
		{Contracts: []Contract{valid}},
		{Version: FileVersionV1Alpha},
		{Version: FileVersionV1Alpha, Contracts: []Contract{valid, valid}},
		{Version: FileVersionV1Alpha, Contracts: []Contract{{Name: "Invalid-Name", Target: "query/Q", CEL: "true"}}},
		{Version: FileVersionV1Alpha, Contracts: []Contract{{Name: "InvalidTarget", Target: "Q", CEL: "true"}}},
	} {
		if _, err := Evaluate(Report{}, file); err == nil {
			t.Errorf("Evaluate(%+v) accepted invalid document", file)
		}
	}
}

func TestEvaluateDynamicCELRequiresBooleanValue(t *testing.T) {
	file := File{Version: FileVersionV1Alpha, Contracts: []Contract{{Name: "Dynamic", Target: "query/Q", CEL: "query.name"}}}
	if err := Validate(file); err != nil {
		t.Fatalf("Validate() rejected dynamic result: %v", err)
	}
	if _, err := Evaluate(Report{Queries: []Query{{Name: "Q", TargetID: "query/Q", Status: "ok"}}}, file); err == nil || !strings.Contains(err.Error(), "bool") {
		t.Fatalf("Evaluate() error = %v, want boolean result error", err)
	}
}

func ExampleValidate() {
	contracts := File{
		Version:   FileVersionV1Alpha,
		Contracts: []Contract{{Name: "NoSort", Target: "query/ListSingers", Use: []string{"no_full_sort"}}},
	}
	// Validate before collecting any query plans or starting a backend.
	fmt.Println(Validate(contracts))
	// Output: <nil>
}

func TestEvaluateValidContractsWithUnavailableTargets(t *testing.T) {
	for _, c := range []Contract{{Use: []string{"no_full_sort"}}, {CEL: "operators.all(o, o.family != 'full_sort')"}} {
		c.Name, c.Target = "Valid", "query/Q"
		file := File{Version: FileVersionV1Alpha, Contracts: []Contract{c}}
		for _, status := range []string{"missing", "error"} {
			report := Report{}
			wantReason := ReasonTargetNotFound
			if status == "error" {
				report.Queries = []Query{{Name: "Q", TargetID: "query/Q", Status: status}}
				wantReason = ReasonTargetError
			}
			got, err := Evaluate(report, file)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Evaluations) != 1 || got.Evaluations[0].Status != StatusNotEvaluated || got.Evaluations[0].Reason != wantReason {
				t.Fatalf("Evaluate() = %+v, want not_evaluated/%s", got, wantReason)
			}
		}
	}
}
