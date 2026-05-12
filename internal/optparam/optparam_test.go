package optparam

import (
	"strings"
	"testing"
)

const singersDDL = `
CREATE TABLE Singers (
  SingerId   INT64 NOT NULL,
  FirstName  STRING(MAX),
  LastName   STRING(MAX),
  Status     STRING(MAX),
) PRIMARY KEY (SingerId);
`

func TestEnumerateVariants_TwoOptionalParams(t *testing.T) {
	sql := `SELECT SingerId, FirstName FROM Singers
WHERE TRUE
  /*?optional:first_name*/ AND FirstName = @first_name /*?end*/
  /*?optional:status*/ AND Status = @status /*?end*/
`
	params := []Param{
		{Name: "first_name", Type: "STRING", Mode: ModeOmitWhenNull},
		{Name: "status", Type: "STRING", Mode: ModeOmitWhenNull},
	}
	variants, err := EnumerateVariants(sql, params)
	if err != nil {
		t.Fatalf("EnumerateVariants: %v", err)
	}
	if got, want := len(variants), 4; got != want {
		t.Fatalf("variant count = %d, want %d", got, want)
	}
	wantKeys := []string{"(none)", "first_name", "first_name+status", "status"}
	for i, v := range variants {
		if v.Key() != wantKeys[i] {
			t.Errorf("variants[%d].Key = %q, want %q", i, v.Key(), wantKeys[i])
		}
		if strings.Contains(v.SQL, "/*?") {
			t.Errorf("variants[%d] still contains marker: %s", i, v.SQL)
		}
	}
	// Sanity-check the SQL each variant produced.
	allOff := variants[0].SQL
	if strings.Contains(allOff, "FirstName = @first_name") || strings.Contains(allOff, "Status = @status") {
		t.Errorf("(none) variant should drop both predicates, got: %s", allOff)
	}
	bothOn := variants[2].SQL
	if !strings.Contains(bothOn, "FirstName = @first_name") || !strings.Contains(bothOn, "Status = @status") {
		t.Errorf("first_name+status variant should keep both predicates, got: %s", bothOn)
	}
}

func TestEnumerateVariants_UnknownMarker(t *testing.T) {
	sql := `SELECT 1 /*?optional:nope*/ AND x = @nope /*?end*/`
	_, err := EnumerateVariants(sql, nil)
	if err == nil || !strings.Contains(err.Error(), "unknown param") {
		t.Fatalf("expected unknown-param error, got %v", err)
	}
}

func TestEnumerateVariants_WrongMode(t *testing.T) {
	sql := `SELECT 1 /*?optional:p*/ AND x = @p /*?end*/`
	_, err := EnumerateVariants(sql, []Param{{Name: "p", Type: "INT64", Mode: ModeNullIsNull}})
	if err == nil || !strings.Contains(err.Error(), "omit_when_null") {
		t.Fatalf("expected mode error, got %v", err)
	}
}

func TestEnumerateVariants_OmitWithoutMarker(t *testing.T) {
	sql := `SELECT 1`
	_, err := EnumerateVariants(sql, []Param{{Name: "p", Type: "INT64", Mode: ModeOmitWhenNull}})
	if err == nil || !strings.Contains(err.Error(), "no marker was found") {
		t.Fatalf("expected missing-marker error, got %v", err)
	}
}

func TestVerifyVariants_AllAgreeOnRowType(t *testing.T) {
	sql := `SELECT SingerId, FirstName FROM Singers
WHERE TRUE
  /*?optional:first_name*/ AND FirstName = @first_name /*?end*/
  /*?optional:status*/ AND Status = @status /*?end*/
`
	params := []Param{
		{Name: "first_name", Type: "STRING", Mode: ModeOmitWhenNull},
		{Name: "status", Type: "STRING", Mode: ModeOmitWhenNull},
	}
	result, err := VerifyVariants("singers.sql", singersDDL, sql, params)
	if err != nil {
		t.Fatalf("VerifyVariants: %v", err)
	}
	if got, want := len(result.Variants), 4; got != want {
		t.Fatalf("variant count = %d, want %d", got, want)
	}
	fields := result.RowType.GetFields()
	if len(fields) != 2 {
		t.Fatalf("row type fields = %d, want 2", len(fields))
	}
	if fields[0].GetName() != "SingerId" || fields[1].GetName() != "FirstName" {
		t.Errorf("unexpected field names: %s, %s", fields[0].GetName(), fields[1].GetName())
	}
}

func TestBuildPlanVariants_StableShape(t *testing.T) {
	sql := `SELECT SingerId, FirstName FROM Singers
WHERE TRUE
  /*?optional:first_name*/ AND FirstName = @first_name /*?end*/
  /*?optional:status*/ AND Status = @status /*?end*/
`
	params := []Param{
		{Name: "first_name", Type: "STRING", Mode: ModeOmitWhenNull},
		{Name: "status", Type: "STRING", Mode: ModeOmitWhenNull},
	}
	result, err := VerifyVariants("singers.sql", singersDDL, sql, params)
	if err != nil {
		t.Fatalf("VerifyVariants: %v", err)
	}
	entries := BuildPlanVariants(result)
	if len(entries) != 4 {
		t.Fatalf("plan variants = %d, want 4", len(entries))
	}
	seen := map[string]bool{}
	for _, e := range entries {
		if seen[e.SQLSHA256] {
			t.Errorf("duplicate SQL hash %s for label %s", e.SQLSHA256, e.Label)
		}
		seen[e.SQLSHA256] = true
		if e.Label == "" || e.SQL == "" || e.SQLSHA256 == "" {
			t.Errorf("incomplete entry: %+v", e)
		}
	}
	t.Log("\n" + FormatPlanVariants(entries))
}

func TestVerifyVariants_DivergentRowTypeIsRejected(t *testing.T) {
	// The "absent" variant projects only SingerId; the "present" variant
	// projects an extra column via a synthesized predicate that is removed
	// when the param is absent. Even though predicate omission alone cannot
	// change the SELECT list, this test deliberately abuses a marker block
	// that wraps SELECT items to confirm the cross-check catches divergence.
	sql := `SELECT SingerId /*?optional:also*/ , FirstName /*?end*/ FROM Singers
WHERE TRUE
  /*?optional:also*/ AND FirstName = @also /*?end*/
`
	params := []Param{
		{Name: "also", Type: "STRING", Mode: ModeOmitWhenNull},
	}
	_, err := VerifyVariants("singers.sql", singersDDL, sql, params)
	if err == nil || !strings.Contains(err.Error(), "row type mismatch") {
		t.Fatalf("expected row type mismatch error, got %v", err)
	}
}
