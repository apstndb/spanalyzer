package optparam

import (
	"strings"
	"testing"
)

const albumsDDL = `
CREATE TABLE Singers (
  SingerId  INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName  STRING(MAX),
  Status    STRING(MAX),
) PRIMARY KEY (SingerId);

CREATE TABLE Albums (
  SingerId   INT64 NOT NULL,
  AlbumId    INT64 NOT NULL,
  Title      STRING(MAX),
  ReleaseTs  TIMESTAMP,
) PRIMARY KEY (SingerId, AlbumId);
`

// null_is_null

func TestNullIsNull_PredicateRewritten(t *testing.T) {
	sql := `SELECT SingerId FROM Singers
WHERE TRUE
  /*?null_is_null:status*/ AND Status = @status /*?end*/
`
	params := []Param{{Name: "status", Type: "STRING", Mode: ModeNullIsNull}}
	segs, err := SegmentTemplate(sql, params)
	if err != nil {
		t.Fatalf("SegmentTemplate: %v", err)
	}
	composed := ComposeVariant(segs, Presence{})
	if !strings.Contains(composed, "IS NOT DISTINCT FROM @status") {
		t.Errorf("expected predicate to be rewritten:\n%s", composed)
	}
	if strings.Contains(composed, "= @status") {
		t.Errorf("expected `= @status` to be replaced:\n%s", composed)
	}
}

func TestNullIsNull_AnalyzerAcceptsRewrittenSQL(t *testing.T) {
	sql := `SELECT SingerId FROM Singers
WHERE TRUE
  /*?null_is_null:status*/ AND Status = @status /*?end*/
`
	params := []Param{{Name: "status", Type: "STRING", Mode: ModeNullIsNull}}
	result, err := VerifyVariants("singers.sql", singersDDL, sql, params)
	if err != nil {
		t.Fatalf("VerifyVariants: %v", err)
	}
	if got := len(result.Variants); got != 1 {
		t.Fatalf("null_is_null should not multiply variants, got %d", got)
	}
}

func TestNullIsNull_MissingEqualsAtMarkerIsRejected(t *testing.T) {
	sql := `SELECT 1 /*?null_is_null:p*/ AND Status IS NULL /*?end*/`
	_, err := SegmentTemplate(sql, []Param{{Name: "p", Type: "STRING", Mode: ModeNullIsNull}})
	if err == nil || !strings.Contains(err.Error(), "= @p") {
		t.Fatalf("expected error mentioning missing `= @p`, got %v", err)
	}
}

// omit_when_empty (array IN UNNEST)

func TestOmitWhenEmpty_TwoVariantsAnalyzeIdentically(t *testing.T) {
	sql := `SELECT SingerId, FirstName FROM Singers
WHERE TRUE
  /*?empty:ids*/ AND SingerId IN UNNEST(@ids) /*?end*/
`
	params := []Param{{Name: "ids", Type: "ARRAY<INT64>", Mode: ModeOmitWhenEmpty}}
	result, err := VerifyVariants("singers.sql", singersDDL, sql, params)
	if err != nil {
		t.Fatalf("VerifyVariants: %v", err)
	}
	if got := len(result.Variants); got != 2 {
		t.Fatalf("expected 2 variants, got %d", got)
	}
	wantKeys := []string{"(none)", "ids"}
	for i, v := range result.Variants {
		if v.Key() != wantKeys[i] {
			t.Errorf("variant %d key %q want %q", i, v.Key(), wantKeys[i])
		}
	}
}

func TestOmitWhenEmpty_BuilderGatesOnLen(t *testing.T) {
	sql := `SELECT SingerId FROM Singers WHERE TRUE
  /*?empty:ids*/ AND SingerId IN UNNEST(@ids) /*?end*/`
	params := []Param{{Name: "ids", Type: "ARRAY<INT64>", Mode: ModeOmitWhenEmpty}}
	segs, err := SegmentTemplate(sql, params)
	if err != nil {
		t.Fatalf("SegmentTemplate: %v", err)
	}
	out, err := EmitGoBuilder(segs, params, BuilderOptions{Package: "gen", FuncName: "Build"})
	if err != nil {
		t.Fatalf("EmitGoBuilder: %v", err)
	}
	for _, needle := range []string{"Ids []int64", "if len(p.Ids) > 0 {"} {
		if !strings.Contains(out, needle) {
			t.Errorf("emitted source missing %q\n---\n%s", needle, out)
		}
	}
}

// orderby allowlist

func TestOrderBy_ChoiceVariants(t *testing.T) {
	sql := `SELECT SingerId, FirstName FROM Singers
/*?orderby:sort*/ ORDER BY SingerId /*?end*/
`
	params := []Param{{
		Name: "sort", Mode: ModeOrderByChoice,
		Choices: map[string]string{
			"id_asc":   "ORDER BY SingerId ASC",
			"name_asc": "ORDER BY LastName, FirstName",
		},
		Default: "id_asc",
	}}
	result, err := VerifyVariants("singers.sql", singersDDL, sql, params)
	if err != nil {
		t.Fatalf("VerifyVariants: %v", err)
	}
	if got, want := len(result.Variants), 2; got != want {
		t.Fatalf("variant count = %d, want %d", got, want)
	}
	wantKeys := map[string]bool{"sort=id_asc": true, "sort=name_asc": true}
	for _, v := range result.Variants {
		if !wantKeys[v.Key()] {
			t.Errorf("unexpected variant key %q", v.Key())
		}
		if !strings.Contains(v.SQL, "ORDER BY") {
			t.Errorf("variant %s missing ORDER BY: %q", v.Key(), v.SQL)
		}
	}
}

func TestOrderBy_BuilderSwitch(t *testing.T) {
	sql := `SELECT SingerId FROM Singers /*?orderby:sort*/ ORDER BY SingerId /*?end*/`
	params := []Param{{
		Name: "sort", Mode: ModeOrderByChoice,
		Choices: map[string]string{
			"id_asc":   "ORDER BY SingerId ASC",
			"name_asc": "ORDER BY LastName, FirstName",
		},
		Default: "id_asc",
	}}
	segs, err := SegmentTemplate(sql, params)
	if err != nil {
		t.Fatalf("SegmentTemplate: %v", err)
	}
	out, err := EmitGoBuilder(segs, params, BuilderOptions{Package: "gen", FuncName: "Build"})
	if err != nil {
		t.Fatalf("EmitGoBuilder: %v", err)
	}
	for _, needle := range []string{
		"Sort string",
		`case "id_asc":`,
		`case "name_asc":`,
		`ORDER BY LastName, FirstName`,
	} {
		if !strings.Contains(out, needle) {
			t.Errorf("emitted source missing %q\n---\n%s", needle, out)
		}
	}
}

// Cross-product: omit + orderby.

func TestCrossProduct_OmitTimesOrderBy(t *testing.T) {
	sql := `SELECT SingerId FROM Singers
WHERE TRUE
  /*?optional:status*/ AND Status = @status /*?end*/
/*?orderby:sort*/ ORDER BY SingerId /*?end*/
`
	params := []Param{
		{Name: "status", Type: "STRING", Mode: ModeOmitWhenNull},
		{
			Name: "sort", Mode: ModeOrderByChoice,
			Choices: map[string]string{
				"id_asc":   "ORDER BY SingerId ASC",
				"name_asc": "ORDER BY LastName, FirstName",
			},
			Default: "id_asc",
		},
	}
	result, err := VerifyVariants("singers.sql", singersDDL, sql, params)
	if err != nil {
		t.Fatalf("VerifyVariants: %v", err)
	}
	if got, want := len(result.Variants), 4; got != want {
		t.Fatalf("variant count = %d, want %d (2 omit * 2 sort)", got, want)
	}
	want := map[string]bool{
		"sort=id_asc":          true,
		"sort=name_asc":        true,
		"sort=id_asc+status":   true,
		"sort=name_asc+status": true,
	}
	for _, v := range result.Variants {
		if !want[v.Key()] {
			t.Errorf("unexpected variant key %q", v.Key())
		}
	}
}

// Range pair (since / until).

func TestRangePair_BothOptional(t *testing.T) {
	sql := `SELECT AlbumId, Title FROM Albums
WHERE TRUE
  /*?optional:since*/ AND ReleaseTs >= @since /*?end*/
  /*?optional:until*/ AND ReleaseTs <  @until /*?end*/
`
	params := []Param{
		{Name: "since", Type: "TIMESTAMP", Mode: ModeOmitWhenNull},
		{Name: "until", Type: "TIMESTAMP", Mode: ModeOmitWhenNull},
	}
	// TIMESTAMP has no Go pointer mapping in the PoC, so we exercise
	// only the analyzer + variant layer (not the Go emitter).
	result, err := VerifyVariants("albums.sql", albumsDDL, sql, params)
	if err != nil {
		t.Fatalf("VerifyVariants: %v", err)
	}
	if got, want := len(result.Variants), 4; got != want {
		t.Fatalf("variant count = %d, want %d", got, want)
	}
}

// LIMIT fixture (existing INT64 pointer mapping).

func TestLimit_OptionalInt64(t *testing.T) {
	sql := `SELECT SingerId, FirstName FROM Singers
WHERE TRUE
  /*?optional:limit*/ LIMIT @limit /*?end*/
`
	params := []Param{{Name: "limit", Type: "INT64", Mode: ModeOmitWhenNull}}
	result, err := VerifyVariants("singers.sql", singersDDL, sql, params)
	if err != nil {
		t.Fatalf("VerifyVariants: %v", err)
	}
	if got, want := len(result.Variants), 2; got != want {
		t.Fatalf("variant count = %d, want %d", got, want)
	}
	segs, err := SegmentTemplate(sql, params)
	if err != nil {
		t.Fatalf("SegmentTemplate: %v", err)
	}
	out, err := EmitGoBuilder(segs, params, BuilderOptions{Package: "gen", FuncName: "Build"})
	if err != nil {
		t.Fatalf("EmitGoBuilder: %v", err)
	}
	for _, needle := range []string{"Limit *int64", "if p.Limit != nil"} {
		if !strings.Contains(out, needle) {
			t.Errorf("emitted source missing %q\n---\n%s", needle, out)
		}
	}
	if err := VerifyBuilderRoundTrip(segs, result.Variants); err != nil {
		t.Errorf("VerifyBuilderRoundTrip: %v", err)
	}
}
