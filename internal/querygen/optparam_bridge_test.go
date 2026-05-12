package querygen

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildQueryCodegenPlan_OmitWhenNullVariants exercises the
// optparam -> querygen integration end-to-end: a YAML config that
// declares two omit_when_null params plus matching SQL markers must
// produce a plan with four per-variant entries that all agree on row
// type.
func TestBuildQueryCodegenPlan_OmitWhenNullVariants(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId  INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName  STRING(MAX),
  Status    STRING(MAX),
) PRIMARY KEY (SingerId);
`)
	config, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
emit:
  spanner:
    mutations: true
catalogs:
- name: app
  kind: spanner
  ddl: spanner.sql
queries:
- name: ListSingers
  catalog: app
  kind: sql
  sql: |
    SELECT SingerId, FirstName FROM Singers
    WHERE TRUE
      /*?optional:first_name*/ AND FirstName = @first_name /*?end*/
      /*?optional:status*/ AND Status = @status /*?end*/
  params:
  - name: first_name
    type: STRING
    optional: omit_when_null
  - name: status
    type: STRING
    optional: omit_when_null
  result:
    cardinality: many
    struct: SingerRow
`))
	if err != nil {
		t.Fatalf("ParseQueryCodegenConfigYAML: %v", err)
	}
	plan, err := BuildQueryCodegenPlan(config, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan: %v", err)
	}
	if len(plan.Queries) != 1 {
		t.Fatalf("plan.Queries = %d, want 1", len(plan.Queries))
	}
	q := plan.Queries[0]
	if got, want := len(q.Variants), 4; got != want {
		t.Fatalf("variants = %d, want %d", got, want)
	}
	wantKeys := map[string]bool{
		"(none)":             true,
		"first_name":         true,
		"first_name+status":  true,
		"status":             true,
	}
	for _, v := range q.Variants {
		if !wantKeys[v.Label] {
			t.Errorf("unexpected variant label %q", v.Label)
		}
		if v.SQLSHA256 == "" || v.SQL == "" {
			t.Errorf("variant %s incomplete: %+v", v.Label, v)
		}
	}
	// Canonical (top-level) SQL should be the all-on variant.
	if !strings.Contains(q.SQL, "AND FirstName = @first_name") || !strings.Contains(q.SQL, "AND Status = @status") {
		t.Errorf("top-level SQL should be the all-on variant, got %q", q.SQL)
	}
	// Plan should still expose 4 result fields (SELECT FirstName, SingerId).
	if got, want := len(q.Fields), 2; got != want {
		t.Errorf("fields = %d, want %d", got, want)
	}
}

// TestBuildQueryCodegenPlan_NoOptionalIsBackwardsCompatible confirms
// that queries without any optional marker still produce a plan with
// no Variants slice — same shape as before this PoC.
func TestBuildQueryCodegenPlan_NoOptionalIsBackwardsCompatible(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId  INT64 NOT NULL,
  FirstName STRING(MAX),
) PRIMARY KEY (SingerId);
`)
	config, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
emit:
  spanner:
    mutations: true
catalogs:
- name: app
  kind: spanner
  ddl: spanner.sql
queries:
- name: ListSingers
  catalog: app
  kind: sql
  sql: SELECT SingerId, FirstName FROM Singers
  result:
    cardinality: many
    struct: SingerRow
`))
	if err != nil {
		t.Fatalf("ParseQueryCodegenConfigYAML: %v", err)
	}
	plan, err := BuildQueryCodegenPlan(config, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan: %v", err)
	}
	q := plan.Queries[0]
	if got := len(q.Variants); got != 0 {
		t.Errorf("Variants on plain query = %d, want 0", got)
	}
	if q.SQL != "SELECT SingerId, FirstName FROM Singers" {
		t.Errorf("plan SQL changed unexpectedly: %q", q.SQL)
	}
}

func TestBuildQueryCodegenPlan_OrderByChoice(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId  INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName  STRING(MAX),
) PRIMARY KEY (SingerId);
`)
	config, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
emit:
  spanner:
    mutations: true
catalogs:
- name: app
  kind: spanner
  ddl: spanner.sql
queries:
- name: ListSingers
  catalog: app
  kind: sql
  sql: |
    SELECT SingerId, FirstName FROM Singers
    /*?orderby:sort*/ ORDER BY SingerId /*?end*/
  params:
  - name: sort
    optional: orderby_choice
    default: id_asc
    choices:
      id_asc: "ORDER BY SingerId ASC"
      name_asc: "ORDER BY LastName, FirstName"
  result:
    cardinality: many
    struct: SingerRow
`))
	if err != nil {
		t.Fatalf("ParseQueryCodegenConfigYAML: %v", err)
	}
	plan, err := BuildQueryCodegenPlan(config, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan: %v", err)
	}
	q := plan.Queries[0]
	if got, want := len(q.Variants), 2; got != want {
		t.Fatalf("variants = %d, want %d", got, want)
	}
	for _, v := range q.Variants {
		if !strings.Contains(v.SQL, "ORDER BY") {
			t.Errorf("variant %s missing ORDER BY: %q", v.Label, v.SQL)
		}
		if v.ChoiceAssignments["sort"] == "" {
			t.Errorf("variant %s missing choice assignment", v.Label)
		}
	}
}
