// Command optparam-plan-probe drives the internal/optparam PoC end-to-
// end against Spanner Omni: it enumerates every verified SQL variant
// for a hard-coded fixture and prints the per-variant query plan
// produced by AnalyzeQuery on the emulator.
//
// This is a developer probe, parallel to tools/spanner-query-plan-shape,
// not a public CLI surface.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/apstndb/spanemuboost"
	"github.com/apstndb/spannerplan/plantree/reference"

	"github.com/apstndb/spanalyzer/internal/optparam"
)

// Fixture bundles a DDL + template + param config for one named probe.
type Fixture struct {
	Name        string
	DDL         string
	Template    string
	Params      []optparam.Param
	SampleValue map[string]interface{}
}

const singersDDL = `
CREATE TABLE Singers (
  SingerId  INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName  STRING(MAX),
  Status    STRING(MAX),
) PRIMARY KEY (SingerId);

CREATE INDEX SingersByFirstName ON Singers(FirstName);
CREATE INDEX SingersByLastName  ON Singers(LastName);
CREATE INDEX SingersByStatus    ON Singers(Status);
`

var fixtures = []Fixture{
	{
		Name:     "omit_when_null:first_name+status",
		DDL:      singersDDL,
		Template: omitWhenNullTemplate,
		Params: []optparam.Param{
			{Name: "first_name", Type: "STRING", Mode: optparam.ModeOmitWhenNull},
			{Name: "status", Type: "STRING", Mode: optparam.ModeOmitWhenNull},
		},
		SampleValue: map[string]interface{}{
			"first_name": "Alice",
			"status":     "active",
		},
	},
	{
		Name:     "omit_when_empty:ids",
		DDL:      singersDDL,
		Template: omitWhenEmptyTemplate,
		Params: []optparam.Param{
			{Name: "ids", Type: "ARRAY<INT64>", Mode: optparam.ModeOmitWhenEmpty},
		},
		SampleValue: map[string]interface{}{
			"ids": []int64{1, 2, 3},
		},
	},
	{
		Name:     "orderby:sort",
		DDL:      singersDDL,
		Template: orderByTemplate,
		Params: []optparam.Param{
			{
				Name: "sort", Mode: optparam.ModeOrderByChoice,
				Choices: map[string]string{
					"id_asc":   "ORDER BY SingerId ASC",
					"name_asc": "ORDER BY LastName, FirstName",
				},
				Default: "id_asc",
			},
		},
	},
	{
		Name:     "null_is_null:status",
		DDL:      singersDDL,
		Template: nullIsNullTemplate,
		Params: []optparam.Param{
			{Name: "status", Type: "STRING", Mode: optparam.ModeNullIsNull},
		},
		SampleValue: map[string]interface{}{
			"status": spanner.NullString{StringVal: "active", Valid: true},
		},
	},
	// Three rewrites of `SELECT * FROM Singers WHERE FirstName = @firstName`.
	{
		Name:     "firstName/required (as written)",
		DDL:      singersDDL,
		Template: firstNameRequiredTemplate,
		Params: []optparam.Param{
			{Name: "firstName", Type: "STRING", Mode: optparam.ModeRequired},
		},
		SampleValue: map[string]interface{}{
			"firstName": "Alice",
		},
	},
	{
		Name:     "firstName/null_is_null",
		DDL:      singersDDL,
		Template: firstNameNullIsNullTemplate,
		Params: []optparam.Param{
			{Name: "firstName", Type: "STRING", Mode: optparam.ModeNullIsNull},
		},
		SampleValue: map[string]interface{}{
			"firstName": spanner.NullString{StringVal: "Alice", Valid: true},
		},
	},
	{
		Name:     "firstName/omit_when_null",
		DDL:      singersDDL,
		Template: firstNameOmitWhenNullTemplate,
		Params: []optparam.Param{
			{Name: "firstName", Type: "STRING", Mode: optparam.ModeOmitWhenNull},
		},
		SampleValue: map[string]interface{}{
			"firstName": "Alice",
		},
	},
	// Hand-rolled equivalent of IS NOT DISTINCT FROM. Worth comparing
	// against the null_is_null fixture above to see whether the planner
	// folds the OR into the same index-seek shape.
	{
		Name:     "firstName/manual_is_not_distinct_from",
		DDL:      singersDDL,
		Template: firstNameManualNullSafeTemplate,
		Params: []optparam.Param{
			{Name: "firstName", Type: "STRING", Mode: optparam.ModeRequired},
		},
		SampleValue: map[string]interface{}{
			"firstName": "Alice",
		},
	},
}

const omitWhenNullTemplate = `SELECT SingerId, FirstName, LastName, Status FROM Singers
WHERE TRUE
  /*?optional:first_name*/ AND FirstName = @first_name /*?end*/
  /*?optional:status*/ AND Status = @status /*?end*/
`

const omitWhenEmptyTemplate = `SELECT SingerId, FirstName FROM Singers
WHERE TRUE
  /*?empty:ids*/ AND SingerId IN UNNEST(@ids) /*?end*/
`

const orderByTemplate = `SELECT SingerId, FirstName, LastName FROM Singers
/*?orderby:sort*/ ORDER BY SingerId /*?end*/
LIMIT 100
`

const nullIsNullTemplate = `SELECT SingerId, Status FROM Singers
WHERE TRUE
  /*?null_is_null:status*/ AND Status = @status /*?end*/
`

const firstNameRequiredTemplate = `SELECT * FROM Singers WHERE FirstName = @firstName
`

const firstNameNullIsNullTemplate = `SELECT * FROM Singers
WHERE /*?null_is_null:firstName*/ FirstName = @firstName /*?end*/
`

const firstNameOmitWhenNullTemplate = `SELECT * FROM Singers
WHERE TRUE
  /*?optional:firstName*/ AND FirstName = @firstName /*?end*/
`

// Verbatim from the user: hand-written null-safe equality without using
// IS NOT DISTINCT FROM. Worth feeding to the planner unchanged to see
// whether it recognizes the idiom.
const firstNameManualNullSafeTemplate = `SELECT * FROM Singers WHERE FirstName = @firstName OR (FirstName IS NULL AND @firstName IS NULL)
`

func main() {
	if err := run(os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(stdout io.Writer) error {
	timeout := flag.Duration("timeout", 5*time.Minute, "overall timeout for emulator + plan acquisition")
	only := flag.String("only", "", "if set, only run the fixture whose name matches this substring")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	runtime := spanemuboost.NewLazyRuntime(spanemuboost.BackendOmni)
	defer func() { _ = runtime.Close() }()

	ran := 0
	for _, fix := range fixtures {
		if *only != "" && !strings.Contains(fix.Name, *only) {
			continue
		}
		if err := runFixture(ctx, stdout, runtime, fix); err != nil {
			return fmt.Errorf("fixture %s: %w", fix.Name, err)
		}
		ran++
	}
	if ran == 0 {
		names := make([]string, 0, len(fixtures))
		for _, fix := range fixtures {
			names = append(names, fix.Name)
		}
		return fmt.Errorf("no fixture name contains %q; available fixtures: %s", *only, strings.Join(names, ", "))
	}
	return nil
}

func runFixture(ctx context.Context, stdout io.Writer, runtime *spanemuboost.LazyRuntime, fix Fixture) error {
	_, _ = fmt.Fprintf(stdout, "\n=== fixture %s ===\n", fix.Name)
	_, _ = fmt.Fprintf(stdout, "template:\n%s\n", strings.TrimRight(fix.Template, "\n"))

	// Verify variants with the in-process analyzer first. This is the
	// build-time guarantee; the probe then takes the same variants and
	// asks the emulator for actual execution plans.
	verify, err := optparam.VerifyVariants(fix.Name+".sql", fix.DDL, fix.Template, fix.Params)
	if err != nil {
		return fmt.Errorf("VerifyVariants: %w", err)
	}
	_, _ = fmt.Fprintf(stdout, "verified row type: %s\n", rowTypeString(verify.RowType))
	_, _ = fmt.Fprintf(stdout, "variants: %d\n", len(verify.Variants))

	// Boot the emulator with the fixture's DDL.
	clients, err := spanemuboost.OpenClients(ctx, runtime,
		spanemuboost.WithRandomDatabaseID(),
		spanemuboost.WithSetupDDLs(splitDDL(fix.DDL)),
	)
	if err != nil {
		return fmt.Errorf("OpenClients: %w", err)
	}
	defer func() { _ = clients.Close() }()

	for _, v := range verify.Variants {
		stmt, err := buildStatement(fix, v)
		if err != nil {
			return fmt.Errorf("variant %s: %w", v.Key(), err)
		}
		plan, err := clients.Client.Single().AnalyzeQuery(ctx, stmt)
		if err != nil {
			return fmt.Errorf("variant %s AnalyzeQuery: %w", v.Key(), err)
		}
		_, _ = fmt.Fprintf(stdout, "\n--- variant %s ---\n", v.Key())
		_, _ = fmt.Fprintln(stdout, strings.TrimSpace(v.SQL))
		rendered, err := reference.RenderTreeTable(plan.GetPlanNodes(), reference.RenderModePlan, reference.FormatCurrent, 0)
		if err != nil {
			return fmt.Errorf("variant %s render: %w", v.Key(), err)
		}
		_, _ = fmt.Fprintln(stdout, rendered)
	}
	return nil
}

func buildStatement(fix Fixture, v optparam.Variant) (spanner.Statement, error) {
	stmt := spanner.NewStatement(v.SQL)
	stmt.Params = map[string]interface{}{}

	for _, p := range fix.Params {
		switch p.Mode {
		case optparam.ModeOrderByChoice:
			// Resolved into literal SQL at codegen; no bind variable.
			continue
		case optparam.ModeOmitWhenNull, optparam.ModeOmitWhenEmpty:
			// Only bind when the predicate appears in this variant's SQL.
			if !contains(v.PresentParams, p.Name) {
				continue
			}
		case optparam.ModeNullIsNull, optparam.ModeRequired:
			// Always bound.
		}
		val, ok := fix.SampleValue[p.Name]
		if !ok {
			return spanner.Statement{}, fmt.Errorf("fixture missing SampleValue for param %q", p.Name)
		}
		stmt.Params[p.Name] = val
	}
	return stmt, nil
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func splitDDL(ddl string) []string {
	var out []string
	for _, s := range strings.Split(ddl, ";") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func rowTypeString(rt *spannerpb.StructType) string {
	var parts []string
	for _, f := range rt.GetFields() {
		parts = append(parts, fmt.Sprintf("%s:%s", f.GetName(), f.GetType().GetCode()))
	}
	return "(" + strings.Join(parts, ", ") + ")"
}
