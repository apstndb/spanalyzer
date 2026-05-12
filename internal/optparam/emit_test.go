package optparam

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmitGoBuilder_FormatsAndContainsSegments(t *testing.T) {
	sql := singersTemplate
	params := singersParams()
	segments, err := SegmentTemplate(sql, params)
	if err != nil {
		t.Fatalf("SegmentTemplate: %v", err)
	}
	out, err := EmitGoBuilder(segments, params, BuilderOptions{
		Package:  "gen",
		FuncName: "BuildListSingers",
	})
	if err != nil {
		t.Fatalf("EmitGoBuilder: %v", err)
	}
	// Required surface. Whitespace inside the struct body is normalized
	// by gofmt, so check tokens individually rather than fixed prefixes.
	for _, needle := range []string{
		"package gen",
		"type BuildListSingersParams struct {",
		"FirstName",
		"*string",
		"Status",
		"func BuildListSingers(p BuildListSingersParams)",
		`if p.FirstName != nil {`,
		`if p.Status != nil {`,
		`args["first_name"] = *p.FirstName`,
		`args["status"] = *p.Status`,
	} {
		if !strings.Contains(out, needle) {
			t.Errorf("emitted source missing %q\n---\n%s", needle, out)
		}
	}
}

func TestVerifyBuilderRoundTrip_AllVariantsByteEqual(t *testing.T) {
	sql := singersTemplate
	params := singersParams()
	segments, err := SegmentTemplate(sql, params)
	if err != nil {
		t.Fatalf("SegmentTemplate: %v", err)
	}
	variants, err := EnumerateVariants(sql, params)
	if err != nil {
		t.Fatalf("EnumerateVariants: %v", err)
	}
	if err := VerifyBuilderRoundTrip(segments, variants); err != nil {
		t.Fatalf("VerifyBuilderRoundTrip: %v", err)
	}
}

// TestEmitGoBuilder_CompiledOutputMatchesVariants writes the emitted Go
// builder to a throwaway module, compiles it together with a generated
// driver that calls the builder with every (present/absent) combination,
// and asserts the runtime-composed SQL matches the verified variants
// byte-for-byte.
//
// This is the hybrid invariant in action: verification runs over all 2^k
// variants at codegen time, but the actual runtime call site only walks
// the linear segment list — and the two outputs must agree exactly.
func TestEmitGoBuilder_CompiledOutputMatchesVariants(t *testing.T) {
	sql := singersTemplate
	params := singersParams()
	segments, err := SegmentTemplate(sql, params)
	if err != nil {
		t.Fatalf("SegmentTemplate: %v", err)
	}
	variants, err := EnumerateVariants(sql, params)
	if err != nil {
		t.Fatalf("EnumerateVariants: %v", err)
	}
	src, err := EmitGoBuilder(segments, params, BuilderOptions{
		Package:  "main",
		FuncName: "Build",
	})
	if err != nil {
		t.Fatalf("EmitGoBuilder: %v", err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module poc\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "builder.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write builder.go: %v", err)
	}
	// Driver: enumerate the same 2^k presence sets and print each as a
	// label\tSQL line so the test can diff against EnumerateVariants.
	driver := `package main

import (
	"fmt"
)

func main() {
	cases := []BuildParams{
		{},
		{FirstName: strPtr("a")},
		{Status: strPtr("b")},
		{FirstName: strPtr("a"), Status: strPtr("b")},
	}
	for _, c := range cases {
		sql, _, label := Build(c)
		fmt.Printf("%s\x1f%s\x1e", label, sql)
	}
}

func strPtr(s string) *string { return &s }
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(driver), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run driver: %v\n--- builder.go ---\n%s\n--- output ---\n%s", err, src, output)
	}

	want := map[string]string{}
	for _, v := range variants {
		want[v.Key()] = v.SQL
	}
	records := strings.Split(strings.TrimRight(string(output), "\x1e"), "\x1e")
	if len(records) != len(variants) {
		t.Fatalf("driver produced %d records, want %d\n%q", len(records), len(variants), output)
	}
	for _, rec := range records {
		label, sql, ok := strings.Cut(rec, "\x1f")
		if !ok {
			t.Fatalf("malformed record %q", rec)
		}
		expected, found := want[label]
		if !found {
			t.Fatalf("driver emitted unknown variant label %q", label)
		}
		if sql != expected {
			t.Errorf("variant %s SQL diverged from verified variant:\n  runtime: %q\n  verified: %q",
				label, sql, expected)
		}
	}
}

const singersTemplate = `SELECT SingerId, FirstName FROM Singers
WHERE TRUE
  /*?optional:first_name*/ AND FirstName = @first_name /*?end*/
  /*?optional:status*/ AND Status = @status /*?end*/
`

func singersParams() []Param {
	return []Param{
		{Name: "first_name", Type: "STRING", Mode: ModeOmitWhenNull},
		{Name: "status", Type: "STRING", Mode: ModeOmitWhenNull},
	}
}
