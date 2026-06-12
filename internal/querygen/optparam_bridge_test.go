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
		"(none)":            true,
		"first_name":        true,
		"first_name+status": true,
		"status":            true,
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

// TestBuildQueryCodegenPlan_IndexKindNullIsNull confirms that
// optional: null_is_null on a kind: index query causes the generated
// SQL to use IS NOT DISTINCT FROM and that no Variants slice is
// emitted (single SQL, no shape multiplication).
func TestBuildQueryCodegenPlan_IndexKindNullIsNull(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId  INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName  STRING(MAX),
) PRIMARY KEY (SingerId);

CREATE INDEX SingersByLastName ON Singers(LastName);
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
- name: FindByLastName
  catalog: app
  kind: index
  index: SingersByLastName
  params:
  - name: LastName
    optional: null_is_null
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
	if !strings.Contains(q.SQL, "IS NOT DISTINCT FROM @LastName") {
		t.Errorf("generated SQL should use IS NOT DISTINCT FROM, got %q", q.SQL)
	}
	if strings.Contains(q.SQL, "LastName = @LastName") {
		t.Errorf("generated SQL should not contain `= @LastName`: %q", q.SQL)
	}
	if got := len(q.Variants); got != 0 {
		t.Errorf("Variants on single-SQL null_is_null query = %d, want 0", got)
	}
}

// TestBuildQueryCodegenPlan_IndexKindOmitWhenNull confirms that
// optional: omit_when_null on a kind: index key-prefix column fans
// the generated SQL out into 2 variants without the user authoring
// any marker.
func TestBuildQueryCodegenPlan_IndexKindOmitWhenNull(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId  INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName  STRING(MAX),
) PRIMARY KEY (SingerId);

CREATE INDEX SingersByLastName ON Singers(LastName);
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
- name: FindByLastName
  catalog: app
  kind: index
  index: SingersByLastName
  params:
  - name: LastName
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
	q := plan.Queries[0]
	if got, want := len(q.Variants), 2; got != want {
		t.Fatalf("variants = %d, want %d", got, want)
	}
	wantKeys := map[string]bool{"(none)": true, "LastName": true}
	for _, v := range q.Variants {
		if !wantKeys[v.Label] {
			t.Errorf("unexpected variant label %q", v.Label)
		}
	}
	if !strings.Contains(q.SQL, "AND `LastName` = @LastName") {
		t.Errorf("canonical SQL should keep the LastName predicate, got %q", q.SQL)
	}
}

// TestBuildQueryCodegenPlan_IndexKindOmitMiddleRejected confirms the
// contiguous-prefix rule: if column N is omittable, every column to
// its right must also be omittable.
func TestBuildQueryCodegenPlan_IndexKindOmitMiddleRejected(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId  INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName  STRING(MAX),
) PRIMARY KEY (SingerId);

CREATE INDEX SingersByLastFirst ON Singers(LastName, FirstName);
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
- name: Find
  catalog: app
  kind: index
  index: SingersByLastFirst
  params:
  - name: LastName
    optional: omit_when_null
  - name: FirstName
    # FirstName is required → invalid because LastName (earlier) is optional
  result:
    cardinality: many
    struct: SingerRow
`))
	if err != nil {
		t.Fatalf("ParseQueryCodegenConfigYAML: %v", err)
	}
	_, err = BuildQueryCodegenPlan(config, dir)
	if err == nil || !strings.Contains(err.Error(), "key-prefix seeks need contiguous predicates") {
		t.Fatalf("expected contiguous-prefix error, got %v", err)
	}
}

// TestBuildQueryCodegenPlan_IndexKindOrderByChoice confirms that
// optional: orderby_choice on a kind: index param replaces the
// default-key ORDER BY clause with a marker that fans out to one
// variant per choice.
func TestBuildQueryCodegenPlan_IndexKindOrderByChoice(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId  INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName  STRING(MAX),
) PRIMARY KEY (SingerId);

CREATE INDEX SingersByLastName ON Singers(LastName);
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
- name: ListByLastName
  catalog: app
  kind: index
  index: SingersByLastName
  params:
  - name: sort
    optional: orderby_choice
    default: name_asc
    choices:
      name_asc: "ORDER BY LastName ASC, FirstName ASC"
      name_desc: "ORDER BY LastName DESC, FirstName DESC"
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
	wantKeys := map[string]bool{"sort=name_asc": true, "sort=name_desc": true}
	for _, v := range q.Variants {
		if !wantKeys[v.Label] {
			t.Errorf("unexpected variant label %q", v.Label)
		}
		if !strings.Contains(v.SQL, "ORDER BY") {
			t.Errorf("variant %s missing ORDER BY: %q", v.Label, v.SQL)
		}
	}
}

// TestBuildQueryCodegenPlan_TableKindKeyPrefix confirms that
// kind: table accepts a key_prefix and applies the same per-param
// optional semantics as kind: index.
func TestBuildQueryCodegenPlan_TableKindKeyPrefix(t *testing.T) {
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
- name: GetById
  catalog: app
  kind: table
  table: Singers
  key_prefix: [SingerId]
  params:
  - name: SingerId
    optional: null_is_null
  result:
    cardinality: maybe_one
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
	if !strings.Contains(q.SQL, "IS NOT DISTINCT FROM @SingerId") {
		t.Errorf("expected IS NOT DISTINCT FROM, got %q", q.SQL)
	}
	if len(q.Variants) != 0 {
		t.Errorf("Variants for null_is_null single-SQL query = %d, want 0", len(q.Variants))
	}
}

func TestBuildQueryCodegenPlan_TableKindOrderByChoice(t *testing.T) {
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
  kind: table
  table: Singers
  params:
  - name: sort
    optional: orderby_choice
    default: id_asc
    choices:
      id_asc: "ORDER BY SingerId ASC"
      name_asc: "ORDER BY LastName ASC, FirstName ASC"
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
	wantKeys := map[string]bool{"sort=id_asc": true, "sort=name_asc": true}
	for _, v := range q.Variants {
		if !wantKeys[v.Label] {
			t.Errorf("unexpected variant label %q", v.Label)
		}
		if !strings.Contains(v.SQL, "ORDER BY") {
			t.Errorf("variant %s missing ORDER BY: %q", v.Label, v.SQL)
		}
	}
}

func TestBuildQueryCodegenPlan_TableKindOmitWhenNull(t *testing.T) {
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
- name: ListMaybeById
  catalog: app
  kind: table
  table: Singers
  key_prefix: [SingerId]
  params:
  - name: SingerId
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
	q := plan.Queries[0]
	if got, want := len(q.Variants), 2; got != want {
		t.Fatalf("variants = %d, want %d", got, want)
	}
	wantKeys := map[string]bool{"(none)": true, "SingerId": true}
	for _, v := range q.Variants {
		if !wantKeys[v.Label] {
			t.Errorf("unexpected variant label %q", v.Label)
		}
	}
}

func TestGenerateQueryCodeOptionalTableUsesBuilder(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
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
- name: ListMaybeById
  catalog: app
  kind: table
  table: Singers
  key_prefix: [SingerId]
  params:
  - name: SingerId
    optional: omit_when_null
  result:
    cardinality: many
    struct: SingerRow
`))
	if err != nil {
		t.Fatalf("ParseQueryCodegenConfigYAML: %v", err)
	}
	code, err := GenerateQueryCode(config, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode: %v", err)
	}
	for _, want := range []string{
		"type ListMaybeByIdParams struct {",
		"SingerId *int64",
		"func BuildListMaybeByIdSQL(p ListMaybeByIdParams)",
		"func ListMaybeById(ctx context.Context, tx *spanner.ReadOnlyTransaction, params ListMaybeByIdParams)",
		"return tx.Query(ctx, spanner.Statement{SQL: sql, Params: args})",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated code missing %q:\n%s", want, code)
		}
	}
	if strings.Contains(code, "\tListMaybeByIdSQL = ") {
		t.Fatalf("optional query should use a builder instead of a single SQL constant:\n%s", code)
	}
}

func TestGenerateQueryCodeOptionalSQLBuilderIncludesRequiredParams(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  Status STRING(MAX),
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
- name: ListByStatus
  catalog: app
  kind: sql
  sql: |
    SELECT SingerId, Status FROM Singers
    WHERE Status = @status
    /*?optional:SingerId*/ AND SingerId = @SingerId /*?end*/
  params:
  - name: status
    type: STRING
  - name: SingerId
    type: INT64
    optional: omit_when_null
  result:
    cardinality: many
    struct: SingerRow
`))
	if err != nil {
		t.Fatalf("ParseQueryCodegenConfigYAML: %v", err)
	}
	code, err := GenerateQueryCode(config, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode: %v", err)
	}
	for _, want := range []string{
		"type ListByStatusParams struct {",
		"Status   string",
		"SingerId *int64",
		`args["status"] = p.Status`,
		`args["SingerId"] = *p.SingerId`,
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated code missing %q:\n%s", want, code)
		}
	}
}

func TestBuildQueryCodegenPlanRejectsInvalidOrderByChoiceKey(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
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
  kind: table
  table: Singers
  params:
  - name: sort
    optional: orderby_choice
    default: name-asc
    choices:
      name-asc: "ORDER BY SingerId"
  result:
    cardinality: many
    struct: SingerRow
`))
	if err != nil {
		t.Fatalf("ParseQueryCodegenConfigYAML: %v", err)
	}
	_, err = BuildQueryCodegenPlan(config, dir)
	if err == nil || !strings.Contains(err.Error(), `choice key "name-asc" must match`) {
		t.Fatalf("expected invalid choice key error, got %v", err)
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
