package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	spanalyzer "github.com/apstndb/spanalyzer"
	"github.com/goccy/go-yaml"
)

func TestRunMutationCountCLI(t *testing.T) {
	dir := t.TempDir()
	mut := filepath.Join(dir, "mutations.json")
	facts := filepath.Join(dir, "facts.json")
	if err := os.WriteFile(mut, []byte(`[{"update":{"table":"Events","columns":["Id","V"],"values":[["1","a"]]}}]`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(facts, []byte(`[{"exists":true,"old":{"V":"a"}}]`), 0600); err != nil {
		t.Fatal(err)
	}
	ddl := `CREATE TABLE Events(Id INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Id); CREATE INDEX ByV ON Events(V);`
	for _, format := range []string{"json", "yaml"} {
		out, err := runMutationCountCLI(ddl, "", format, mut, "", facts, "", "", "")
		if err != nil {
			t.Fatal(err)
		}
		b := []byte(out)
		if format == "yaml" {
			b, err = yaml.YAMLToJSON(b)
			if err != nil {
				t.Fatal(err)
			}
		}
		var r spanalyzer.MutationCountReport
		if err := json.Unmarshal(b, &r); err != nil {
			t.Fatal(err)
		}
		if r.Bounds.Count.Upper == nil || *r.Bounds.Count.Upper != 4 || r.Effective.Upper == nil || *r.Effective.Upper != 2 {
			t.Fatal(r)
		}
	}
	if _, err := runMutationCountCLI(ddl, "SELECT 1", "json", "", "", "", "", "", ""); err == nil {
		t.Fatal("ordinary SELECT is not a write estimate")
	}
	if _, err := runMutationCountCLI(`CREATE QUEUE Tasks(Id STRING(36) NOT NULL,Payload BYTES(MAX) NOT NULL) PRIMARY KEY(Id)`, "SELECT * FROM RECEIVE_Tasks(max_duration=>'1s')", "json", "", "", "", "", "", ""); err != nil {
		t.Fatal(err)
	}
}

func TestMutationCountCLIRejectsUnknownRefinementFields(t *testing.T) {
	dir := t.TempDir()
	mutations := filepath.Join(dir, "mutations.json")
	if err := os.WriteFile(mutations, []byte(`[{"update":{"table":"Events","columns":["Id","V"],"values":[["1","a"]]}}]`), 0600); err != nil {
		t.Fatal(err)
	}
	ddl := `CREATE TABLE Events(Id INT64 NOT NULL,V STRING(MAX)) PRIMARY KEY(Id); CREATE INDEX ByV ON Events(V);`
	for _, tc := range []struct {
		name, data string
	}{
		{"facts", `[{"exists":true,"old_row":{"V":"a"}}]`},
		{"indexes", `{"schema_sha256":"unused","indexes":[{"table":"Events","name":"ByV","columns":["V"],"manageed":true}]}`},
		{"observation", `{"successful":true,"stats_present":true,"mutationCount":2}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".json")
			if err := os.WriteFile(path, []byte(tc.data), 0600); err != nil {
				t.Fatal(err)
			}
			var facts, indexes, observation string
			switch tc.name {
			case "facts":
				facts = path
			case "indexes":
				indexes = path
			case "observation":
				observation = path
			}
			_, err := runMutationCountCLI(ddl, "", "json", mutations, "", facts, indexes, observation, "")
			if err == nil || !strings.Contains(err.Error(), "unknown field") || !strings.Contains(err.Error(), path) {
				t.Fatalf("expected a file-specific unknown-field error, got %v", err)
			}
		})
	}
}

func TestMutationCountTransactionCLI(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "mutation-count")
	ddl, err := os.ReadFile(filepath.Join(fixture, "transaction.sql"))
	if err != nil {
		t.Fatal(err)
	}
	transaction := filepath.Join(fixture, "transaction.json")
	out, err := runMutationCountCLI(string(ddl), "", "json", "", transaction, filepath.Join(fixture, "transaction-facts.json"), "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	var report spanalyzer.MutationCountReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatal(err)
	}
	if report.Scope != "mixed_dml_mutation_transaction" || report.Effective.Upper != nil || len(report.Bounds.Statements) != 1 || report.Bounds.BufferedCommit == nil {
		t.Fatal(report)
	}
	for _, tc := range []struct{ sql, mutations string }{{"SELECT 1", ""}, {"", transaction}} {
		if _, err := runMutationCountCLI(string(ddl), tc.sql, "json", tc.mutations, transaction, "", "", "", ""); err == nil {
			t.Fatal("mixed input sources must be rejected", tc)
		}
	}
	bad := filepath.Join(t.TempDir(), "transaction.json")
	if err := os.WriteFile(bad, []byte(`{"statements":["DELETE FROM Tasks"],"mutation":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runMutationCountCLI(string(ddl), "", "json", "", bad, "", "", "", ""); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatal("misspelled transaction field was not rejected", err)
	}
}
