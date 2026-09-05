package infoschem

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

const constraintColumnDDL = `CREATE TABLE Probe (
  Id INT64 NOT NULL,
  CONSTRAINT BOOL,
  Payload STRING(MAX)
) PRIMARY KEY (Id)`

const quotedConstraintColumnDDL = "CREATE TABLE Probe (\n  Id INT64 NOT NULL,\n  `CONSTRAINT` BOOL,\n  Payload STRING(MAX)\n) PRIMARY KEY (Id)"

func TestClassifyCanonicalDDL_ConstraintColumnParseClass(t *testing.T) {
	got := ClassifyCanonicalDDL([]string{constraintColumnDDL}, nil)
	if got.Parsed != 0 || got.Allowlisted != 0 || len(got.Unexpected) != 1 {
		t.Fatalf("classification = parsed=%d allowlisted=%d unexpected=%d, want 0/0/1", got.Parsed, got.Allowlisted, len(got.Unexpected))
	}
	fail := got.Unexpected[0]
	if fail.Family != "CreateTable" {
		t.Errorf("family = %q, want CreateTable", fail.Family)
	}
	if fail.ErrorClass != "punctuation_2c" {
		t.Errorf("error_class = %q, want punctuation_2c", fail.ErrorClass)
	}
	wantSHA := sha256Hex(constraintColumnDDL)
	if fail.SHA256 != wantSHA {
		t.Errorf("sha256 = %q, want %q", fail.SHA256, wantSHA)
	}
	if fail.SHA256 == memefishConstraintColumnAllowlistSHA256 {
		t.Fatal("synthetic statement hashed to the managed allowlist key")
	}
	if strings.Contains(strings.ToLower(fail.Family+fail.SHA256+fail.ErrorClass), "probe") {
		t.Fatal("sanitized failure contains synthetic table identifier")
	}
}

func TestClassifyCanonicalDDL_UnparsedFamilyIgnoresIdentifierTokens(t *testing.T) {
	sql := "CREATE SecretObject ZzIdentifierLiteral_9f3c ('secret')"
	got := ClassifyCanonicalDDL([]string{sql}, nil)
	if len(got.Unexpected) != 1 {
		t.Fatalf("unexpected = %d, want 1", len(got.Unexpected))
	}
	fail := got.Unexpected[0]
	if fail.Family != "Create" {
		t.Errorf("family = %q, want Create", fail.Family)
	}
	joined := strings.ToLower(fail.Family + fail.SHA256 + fail.ErrorClass)
	for _, leak := range []string{"secretobject", "zzidentifierliteral", "secret"} {
		if strings.Contains(joined, leak) {
			t.Fatalf("sanitized fields contain %q: family=%q error_class=%q", leak, fail.Family, fail.ErrorClass)
		}
	}

	alterSQL := "ALTER UniqueThing_q1 SET OPTIONS (x = 1)"
	got = ClassifyCanonicalDDL([]string{alterSQL}, nil)
	if len(got.Unexpected) != 1 {
		t.Fatalf("ALTER unexpected = %d, want 1", len(got.Unexpected))
	}
	fail = got.Unexpected[0]
	if fail.Family != "Alter" {
		t.Errorf("ALTER family = %q, want Alter", fail.Family)
	}
	joined = strings.ToLower(fail.Family + fail.SHA256 + fail.ErrorClass)
	if strings.Contains(joined, "uniquething") {
		t.Fatalf("ALTER sanitized fields contain identifier: family=%q error_class=%q", fail.Family, fail.ErrorClass)
	}
}

func TestClassifyCanonicalDDL_QuotedConstraintColumnParses(t *testing.T) {
	got := ClassifyCanonicalDDL([]string{quotedConstraintColumnDDL}, nil)
	if got.Parsed != 1 || got.Allowlisted != 0 || len(got.Unexpected) != 0 {
		t.Fatalf("classification = parsed=%d allowlisted=%d unexpected=%d, want 1/0/0", got.Parsed, got.Allowlisted, len(got.Unexpected))
	}
	if got.ParsedFamilies["CreateTable"] != 1 {
		t.Fatalf("parsed families = %#v, want CreateTable=1", got.ParsedFamilies)
	}
}

func TestClassifyCanonicalDDL_ExactAllowlistMatch(t *testing.T) {
	allowlist := []CanonicalAllowlistEntry{{
		SHA256:     sha256Hex(constraintColumnDDL),
		ErrorClass: "punctuation_2c",
	}}
	got := ClassifyCanonicalDDL([]string{constraintColumnDDL}, allowlist)
	if got.Parsed != 0 || got.Allowlisted != 1 || len(got.Unexpected) != 0 {
		t.Fatalf("classification = parsed=%d allowlisted=%d unexpected=%d, want 0/1/0", got.Parsed, got.Allowlisted, len(got.Unexpected))
	}
	if got.Parsed+got.Allowlisted != got.Total {
		t.Fatalf("accounting parsed=%d allowlisted=%d total=%d", got.Parsed, got.Allowlisted, got.Total)
	}
	if len(got.ParsedFamilies) != 0 {
		t.Fatalf("allowlisted statement counted as parsed family data: %#v", got.ParsedFamilies)
	}
}

func TestClassifyCanonicalDDL_HashMismatchIsUnexpected(t *testing.T) {
	allowlist := []CanonicalAllowlistEntry{{
		SHA256:     strings.Repeat("0", 64),
		ErrorClass: "punctuation_2c",
	}}
	got := ClassifyCanonicalDDL([]string{constraintColumnDDL}, allowlist)
	if got.Allowlisted != 0 || len(got.Unexpected) != 1 {
		t.Fatalf("classification = allowlisted=%d unexpected=%d, want 0/1", got.Allowlisted, len(got.Unexpected))
	}
}

func TestClassifyCanonicalDDL_ErrorClassMismatchIsUnexpected(t *testing.T) {
	allowlist := []CanonicalAllowlistEntry{{
		SHA256:     sha256Hex(constraintColumnDDL),
		ErrorClass: "ident",
	}}
	got := ClassifyCanonicalDDL([]string{constraintColumnDDL}, allowlist)
	if got.Allowlisted != 0 || len(got.Unexpected) != 1 {
		t.Fatalf("classification = allowlisted=%d unexpected=%d, want 0/1", got.Allowlisted, len(got.Unexpected))
	}
	if got.Unexpected[0].ErrorClass != "punctuation_2c" {
		t.Fatalf("error_class = %q, want punctuation_2c", got.Unexpected[0].ErrorClass)
	}
}

func TestClassifyCanonicalDDL_UnexpectedParseFailure(t *testing.T) {
	sql := "CREATE TABLE"
	got := ClassifyCanonicalDDL([]string{sql}, nil)
	if got.Parsed != 0 || got.Allowlisted != 0 || len(got.Unexpected) != 1 {
		t.Fatalf("classification = parsed=%d allowlisted=%d unexpected=%d, want 0/0/1", got.Parsed, got.Allowlisted, len(got.Unexpected))
	}
	fail := got.Unexpected[0]
	if fail.Family != "CreateTable" {
		t.Errorf("family = %q, want CreateTable", fail.Family)
	}
	if fail.SHA256 != sha256Hex(sql) {
		t.Errorf("sha256 = %q, want statement digest", fail.SHA256)
	}
	if fail.ErrorClass == "" || strings.Contains(fail.ErrorClass, " ") {
		t.Errorf("error_class = %q, want a stable class without source text", fail.ErrorClass)
	}
}

func TestClassifyCanonicalDDL_FullAccounting(t *testing.T) {
	valid := `CREATE TABLE Ok (
  Id INT64 NOT NULL
) PRIMARY KEY (Id)`
	allowlist := []CanonicalAllowlistEntry{{
		SHA256:     sha256Hex(constraintColumnDDL),
		ErrorClass: "punctuation_2c",
	}}
	got := ClassifyCanonicalDDL([]string{valid, constraintColumnDDL}, allowlist)
	if got.Total != 2 || got.Parsed != 1 || got.Allowlisted != 1 || len(got.Unexpected) != 0 {
		t.Fatalf("classification = total=%d parsed=%d allowlisted=%d unexpected=%d, want 2/1/1/0", got.Total, got.Parsed, got.Allowlisted, len(got.Unexpected))
	}
	if got.Parsed+got.Allowlisted != got.Total {
		t.Fatalf("accounting parsed=%d allowlisted=%d total=%d", got.Parsed, got.Allowlisted, got.Total)
	}
	if got.ParsedFamilies["CreateTable"] != 1 {
		t.Fatalf("parsed families = %#v, want only the successful statement", got.ParsedFamilies)
	}
}

func TestDefaultCanonicalDDLAllowlistIsExact(t *testing.T) {
	list := DefaultCanonicalDDLAllowlist()
	if len(list) != 1 {
		t.Fatalf("allowlist len = %d, want 1", len(list))
	}
	if list[0].SHA256 != memefishConstraintColumnAllowlistSHA256 {
		t.Fatalf("allowlist sha256 = %q, want managed key", list[0].SHA256)
	}
	if list[0].ErrorClass != memefishConstraintColumnAllowlistErrorClass {
		t.Fatalf("allowlist error_class = %q, want punctuation_2c", list[0].ErrorClass)
	}
	if len(list[0].SHA256) != 64 {
		t.Fatalf("allowlist sha256 length = %d, want full digest", len(list[0].SHA256))
	}
}

func sha256Hex(sql string) string {
	sum := sha256.Sum256([]byte(sql))
	return hex.EncodeToString(sum[:])
}
