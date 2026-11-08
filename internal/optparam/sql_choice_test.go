package optparam

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const eventsDDL = `CREATE TABLE Events (
  Id INT64 NOT NULL, Shard INT64, CreatedAt TIMESTAMP,
  Active STRING(MAX), Pending STRING(MAX), Owner STRING(MAX)
) PRIMARY KEY (Id)`

const rangeTemplate = `SELECT Id FROM Events WHERE Shard = @shard
/*?choice:bounds*/ AND CreatedAt BETWEEN @start AND @end /*?end*/`

func rangeParams() []Param {
	return []Param{
		{Name: "shard", Type: "INT64"},
		{Name: "start", Type: "TIMESTAMP"},
		{Name: "end", Type: "TIMESTAMP"},
		{Name: "bounds", Mode: ModeSQLChoice, Default: "both", Choices: map[string]string{
			"both":  " AND CreatedAt BETWEEN @start AND @end",
			"start": " AND CreatedAt >= @START",
			"end":   " AND CreatedAt <= @end",
		}},
	}
}

func TestSQLChoiceCorrelatedRangeAndIndependentColumn(t *testing.T) {
	params := rangeParams()
	result, err := VerifyVariants("schema.sql", eventsDDL, rangeTemplate, params)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Variants) != 3 {
		t.Fatalf("range variants = %d, want exactly 3 (no unbounded branch)", len(result.Variants))
	}
	wantAbsent := map[string][]string{"both": nil, "start": {"end"}, "end": {"start"}}
	for _, v := range result.Variants {
		key := v.ChoiceAssignments["bounds"]
		if !reflect.DeepEqual(v.AbsentParams, wantAbsent[key]) || len(v.PresentParams) != 0 || v.Key() != "bounds="+key {
			t.Fatalf("unexpected range binding/label: %+v", v)
		}
	}
	params = append(params, Param{Name: "flag", Mode: ModeSQLChoice, Default: "active", Choices: map[string]string{
		"active": ` AND Active = "1"`, "pending": ` AND Pending = "1"`,
	}})
	sql := rangeTemplate + "\n/*?choice:flag*/ AND Active = \"1\" /*?end*/"
	result, err = VerifyVariants("schema.sql", eventsDDL, sql, params)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Variants) != 6 {
		t.Fatalf("range x column variants = %d, want 6", len(result.Variants))
	}
	first := BuildPlanVariants(result)
	second, err := EnumerateVariants(sql, params)
	if err != nil || !reflect.DeepEqual(first, BuildPlanVariants(&VerifyResult{Variants: second})) {
		t.Fatalf("variant labels/hashes changed: %v", err)
	}
	params = append(params, Param{Name: "owner", Type: "STRING", Mode: ModeOmitWhenNull})
	segments, err := SegmentTemplate(sql+" /*?optional:owner*/ AND Owner = @owner /*?end*/", params)
	if err != nil {
		t.Fatal(err)
	}
	variants, err := EnumerateVariants(sql+" /*?optional:owner*/ AND Owner = @owner /*?end*/", params)
	if err != nil || len(variants) != 12 {
		t.Fatalf("independent optional block: %d, %v", len(variants), err)
	}
	if err := VerifyBuilderRoundTrip(segments, variants); err != nil {
		t.Fatal(err)
	}
}

func TestSQLChoiceTracksTokensAndRejectsInvalidBindings(t *testing.T) {
	for _, tc := range []struct {
		name, sql, fragment, want string
		mode                      Mode
	}{
		{name: "quoted references", sql: "SELECT '@missing', `@quoted` FROM (SELECT 1 AS `@quoted`) /*?choice:c*/ /*?end*/", fragment: " -- @missing\n # @c\n /* @missing */", mode: ModeRequired},
		{name: "undeclared branch bind", fragment: " + @missing", want: "undeclared bind parameter @missing"},
		{name: "selector bound", fragment: " + @c", want: "cannot be a bind parameter"},
		{name: "nullable bind in branch", fragment: " + @p", mode: ModeNullIsNull, want: "must be required"},
		{name: "omitted bind in branch", fragment: " + @p", mode: ModeOmitWhenNull, want: "must be required"},
		{name: "optional escapes block", sql: "SELECT @p /*?choice:c*/ /*?end*/ /*?optional:p*/ + @p /*?end*/", fragment: " + 1", mode: ModeOmitWhenNull, want: "must stay inside its own optional block"},
		{name: "invalid token", fragment: ` + '\xZZ'`, want: "SQL fragment"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sql := tc.sql
			if sql == "" {
				sql = "SELECT 1 /*?choice:c*/ /*?end*/"
				if tc.mode == ModeOmitWhenNull {
					sql += " /*?optional:p*/ + @p /*?end*/"
				}
			}
			params := []Param{{Name: "c", Mode: ModeSQLChoice, Default: "one", Choices: map[string]string{"one": tc.fragment}}, {Name: "p", Type: "INT64", Mode: tc.mode}}
			variants, err := EnumerateVariants(sql, params)
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("error = %v, want %q", err, tc.want)
				}
				return
			}
			if err != nil || len(variants) != 1 || !slices.Equal(variants[0].AbsentParams, []string{"p"}) {
				t.Fatalf("quoted tokens became bindings: %+v, %v", variants, err)
			}
		})
	}
}

func TestSQLChoiceRejectsRowTypeMismatchAndExcessiveProduct(t *testing.T) {
	params := []Param{{Name: "column", Mode: ModeSQLChoice, Default: "id", Choices: map[string]string{
		"id": "Id AS Value", "time": "CreatedAt AS Value",
	}}}
	if _, err := VerifyVariants("schema.sql", eventsDDL, "SELECT /*?choice:column*/ Id AS Value /*?end*/ FROM Events", params); err == nil || !strings.Contains(err.Error(), "row type mismatch") {
		t.Fatalf("row-type change = %v", err)
	}
	choices := map[string]string{}
	for i := range 65 {
		choices[fmt.Sprintf("c%d", i)] = " + 1"
	}
	params = []Param{{Name: "a", Mode: ModeSQLChoice, Default: "c0", Choices: choices}, {Name: "b", Mode: ModeSQLChoice, Default: "c0", Choices: choices}}
	if _, err := EnumerateVariants("SELECT 1 /*?choice:a*/ /*?end*/ /*?choice:b*/ /*?end*/", params); err == nil || !strings.Contains(err.Error(), "limit 4096") {
		t.Fatalf("unbounded Cartesian product = %v", err)
	}
}

func TestSQLChoiceGeneratedBuilderSelectedArgsAndDefault(t *testing.T) {
	params := rangeParams()
	segments, err := SegmentTemplate(rangeTemplate, params)
	if err != nil {
		t.Fatal(err)
	}
	src, err := EmitGoBuilder(segments, params, BuilderOptions{Package: "main", FuncName: "Build"})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	files := map[string]string{
		"go.mod":     "module sqlchoice\n\ngo 1.26\n",
		"builder.go": src,
		"main.go": `package main
import ("encoding/json"; "os"; "sort"; "time")
func main() {
  for _, choice := range []string{"both", "start", "end", ""} {
    p := BuildParams{Shard: 7, Bounds: choice, Start: time.Unix(1,0), End: time.Unix(2,0)}
    sql, args, label := Build(p)
    names := make([]string,0,len(args))
    for name, value := range args {
      names = append(names,name)
      if name=="start" && value != p.Start { panic("wrong start value") }
      if name=="end" && value != p.End { panic("wrong end value") }
    }
    sort.Strings(names)
    json.NewEncoder(os.Stdout).Encode(struct {SQL, Label string; Names []string}{sql,label,names})
  }
  func() {
    defer func() { if recover() == nil { panic("unknown choice accepted") } }()
    Build(BuildParams{Bounds: "unknown"})
  }()
}
`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated builder: %v\n%s\n%s", err, src, out)
	}
	variants, err := EnumerateVariants(rangeTemplate, params)
	if err != nil {
		t.Fatal(err)
	}
	wantSQL := map[string]string{}
	for _, v := range variants {
		wantSQL[v.Key()] = v.SQL
	}
	wantNames := [][]string{{"end", "shard", "start"}, {"shard", "start"}, {"end", "shard"}, {"end", "shard", "start"}}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 4 {
		t.Fatalf("builder output = %q", out)
	}
	for i, line := range lines {
		var got struct {
			SQL, Label string
			Names      []string
		}
		if err := json.Unmarshal([]byte(line), &got); err != nil {
			t.Fatal(err)
		}
		if got.SQL != wantSQL[got.Label] || !slices.Equal(got.Names, wantNames[i]) {
			t.Fatalf("builder[%d] = %+v, want names %v and verified SQL", i, got, wantNames[i])
		}
		if i == 3 && got.Label != "bounds=both" {
			t.Fatalf("default = %q", got.Label)
		}
	}
}

func TestSQLChoiceBuilderUsesCompleteSQLCommentContext(t *testing.T) {
	// A fragment-local lexer sees @value and @guard in later segments, but
	// the complete statement comments them out in the comment branch. The
	// repeated choice marker is one axis, including in the emitted label.
	sql := `SELECT 1 /*?choice:tail*/ /*?end*/ + @value /*?optional:guard*/ + @guard /*?end*/ /*?choice:tail*/ /*?end*/`
	params := []Param{
		{Name: "tail", Mode: ModeSQLChoice, Default: "comment", Choices: map[string]string{"comment": " -- comment", "value": " + 0"}},
		{Name: "value", Type: "INT64"},
		{Name: "guard", Type: "INT64", Mode: ModeOmitWhenNull},
	}
	verified, err := VerifyVariants("schema.sql", "", sql, params)
	if err != nil {
		t.Fatal(err)
	}
	segments, err := SegmentTemplate(sql, params)
	if err != nil {
		t.Fatal(err)
	}
	src, err := EmitGoBuilder(segments, params, BuilderOptions{Package: "main", FuncName: "Build"})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for name, content := range map[string]string{
		"go.mod":     "module comments\n\ngo 1.26\n",
		"builder.go": src,
		"main.go": `package main
import ("encoding/json"; "os"; "sort")
func main() {
  n := int64(3)
  for _, p := range []BuildParams{{Value: 2}, {Value: 2, Guard: &n}, {Tail: "value", Value: 2}, {Tail: "value", Value: 2, Guard: &n}} {
    sql, args, label := Build(p)
    names := make([]string,0,len(args)); for name := range args { names=append(names,name) }; sort.Strings(names)
    json.NewEncoder(os.Stdout).Encode(struct {SQL, Label string; Names []string}{sql,label,names})
  }
}
`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled comment-boundary builder: %v\n%s\n%s", err, src, out)
	}
	want := map[string]Variant{}
	for _, v := range verified.Variants {
		want[v.Key()] = v
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != len(want) {
		t.Fatalf("outputs = %q", out)
	}
	for _, line := range lines {
		var got struct {
			SQL, Label string
			Names      []string
		}
		if err := json.Unmarshal([]byte(line), &got); err != nil {
			t.Fatal(err)
		}
		v, ok := want[got.Label]
		if !ok || v.SQL != got.SQL {
			t.Fatalf("label/SQL differs from verification: %+v", got)
		}
		var names []string
		for _, name := range []string{"guard", "value"} {
			if !slices.Contains(v.AbsentParams, name) {
				names = append(names, name)
			}
		}
		if !slices.Equal(names, got.Names) {
			t.Fatalf("%s bindings = %v, want verified %v", got.Label, got.Names, names)
		}
	}
}
