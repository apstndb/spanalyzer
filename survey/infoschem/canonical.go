package infoschem

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
	"github.com/cloudspannerecosystem/memefish/token"
)

// memefishConstraintColumnAllowlist is a temporary compatibility exception for
// one managed GetDatabaseDdl statement whose unquoted column spelling
// CONSTRAINT is parsed as a table-constraint keyword. Remove this entry when
// TestClassifyCanonicalDDL_ConstraintColumnParseClass passes against a
// memefish version that accepts that column-definition form. Do not open or
// post an external issue or PR without explicit user authorization.
const (
	memefishConstraintColumnAllowlistSHA256     = "26f18fe6c0fd5c08669ea7ef94193f92fc436d70643fcccfd011992fb7e3881c"
	memefishConstraintColumnAllowlistErrorClass = "punctuation_2c"
)

// CanonicalAllowlistEntry matches one parse failure by exact statement SHA-256
// and exact sanitized error class. Prefix matching is not allowed.
type CanonicalAllowlistEntry struct {
	SHA256     string
	ErrorClass string
}

// CanonicalParseFailure is a sanitized unexpected parse failure. It must not
// carry SQL, identifiers, literals, or parser source snippets.
type CanonicalParseFailure struct {
	Family     string
	SHA256     string
	ErrorClass string
}

// CanonicalClassification accounts for every canonical DDL statement.
type CanonicalClassification struct {
	Total          int
	Parsed         int
	Allowlisted    int
	ParsedFamilies map[string]int
	Unexpected     []CanonicalParseFailure
}

// DefaultCanonicalDDLAllowlist is the managed-gate exception list. Keep it
// empty once the memefish compatibility regression is fixed.
func DefaultCanonicalDDLAllowlist() []CanonicalAllowlistEntry {
	return []CanonicalAllowlistEntry{{
		SHA256:     memefishConstraintColumnAllowlistSHA256,
		ErrorClass: memefishConstraintColumnAllowlistErrorClass,
	}}
}

// ClassifyCanonicalDDL places each statement in exactly one of parsed,
// allowlisted, or unexpected. Allowlisted statements are not counted as parsed
// family data.
func ClassifyCanonicalDDL(statements []string, allowlist []CanonicalAllowlistEntry) CanonicalClassification {
	allowed := make(map[string]string, len(allowlist))
	for _, entry := range allowlist {
		allowed[entry.SHA256] = entry.ErrorClass
	}
	out := CanonicalClassification{
		Total:          len(statements),
		ParsedFamilies: map[string]int{},
	}
	for _, sql := range statements {
		sum := sha256.Sum256([]byte(sql))
		digest := hex.EncodeToString(sum[:])
		ddl, err := memefish.ParseDDL("", sql)
		if err == nil {
			family := ddlFamily(ddl)
			out.Parsed++
			out.ParsedFamilies[family]++
			continue
		}
		class := canonicalParseErrorClass(sql, err)
		family := unparsedDDLFamily(sql)
		if wantClass, ok := allowed[digest]; ok && wantClass == class {
			out.Allowlisted++
			continue
		}
		out.Unexpected = append(out.Unexpected, CanonicalParseFailure{
			Family:     family,
			SHA256:     digest,
			ErrorClass: class,
		})
	}
	return out
}

// DDLFamilyName returns the memefish AST type name without the *ast. prefix.
func DDLFamilyName(ddl ast.DDL) string {
	return ddlFamily(ddl)
}

func ddlFamily(ddl ast.DDL) string {
	return strings.TrimPrefix(reflect.TypeOf(ddl).String(), "*ast.")
}

func canonicalParseErrorClass(sql string, err error) string {
	first := firstMemefishError(err)
	if first == nil || first.Position == nil {
		return "parse_error"
	}
	return tokenClassAt(sql, first.Position.Pos)
}

func firstMemefishError(err error) *memefish.Error {
	var list memefish.MultiError
	if errors.As(err, &list) && len(list) > 0 {
		return list[0]
	}
	var one *memefish.Error
	if errors.As(err, &one) {
		return one
	}
	return nil
}

func tokenClassAt(sql string, pos token.Pos) string {
	if pos.Invalid() {
		return "unknown"
	}
	lex := &memefish.Lexer{File: &token.File{Buffer: sql}}
	for range len(sql) + 2 {
		if err := lex.NextToken(); err != nil {
			return "lex_error"
		}
		tok := lex.Token
		if tok.Kind == token.TokenEOF {
			return "eof"
		}
		if tok.Pos.Invalid() {
			continue
		}
		if pos >= tok.Pos && (tok.End.Invalid() || pos < tok.End) {
			return tokenKindClass(tok.Kind)
		}
		if pos < tok.Pos {
			return tokenKindClass(tok.Kind)
		}
	}
	return "unknown"
}

func tokenKindClass(kind token.TokenKind) string {
	raw := string(kind)
	if len(raw) == 1 {
		return fmt.Sprintf("punctuation_%02x", raw[0])
	}
	switch kind {
	case token.TokenIdent:
		return "ident"
	case token.TokenParam:
		return "param"
	case token.TokenInt:
		return "int"
	case token.TokenFloat:
		return "float"
	case token.TokenString:
		return "string"
	case token.TokenBytes:
		return "bytes"
	case token.TokenEOF:
		return "eof"
	case token.TokenBad:
		return "bad"
	default:
		return "token_other"
	}
}

var knownDDLKeywords = map[string]struct{}{
	"ALTER": {}, "BUNDLE": {}, "CHANGE": {}, "CREATE": {}, "DATABASE": {},
	"EXISTS": {}, "FUNCTION": {}, "GRAPH": {}, "GRANT": {}, "GROUP": {},
	"IF": {}, "INDEX": {}, "LOCALITY": {}, "MODEL": {}, "NOT": {},
	"OR": {}, "PLACEMENT": {}, "PROPERTY": {}, "PROTO": {}, "REPLACE": {},
	"REVOKE": {}, "ROLE": {}, "SCHEMA": {}, "SEARCH": {}, "SEQUENCE": {},
	"STATISTICS": {}, "STREAM": {}, "TABLE": {}, "VECTOR": {}, "VIEW": {},
}

var createFamilies = map[string]string{
	"TABLE":          "CreateTable",
	"INDEX":          "CreateIndex",
	"VIEW":           "CreateView",
	"SCHEMA":         "CreateSchema",
	"SEQUENCE":       "CreateSequence",
	"ROLE":           "CreateRole",
	"FUNCTION":       "CreateFunction",
	"MODEL":          "CreateModel",
	"DATABASE":       "CreateDatabase",
	"PLACEMENT":      "CreatePlacement",
	"CHANGE STREAM":  "CreateChangeStream",
	"SEARCH INDEX":   "CreateSearchIndex",
	"VECTOR INDEX":   "CreateVectorIndex",
	"PROPERTY GRAPH": "CreatePropertyGraph",
	"PROTO BUNDLE":   "CreateProtoBundle",
	"LOCALITY GROUP": "CreateLocalityGroup",
}

var alterFamilies = map[string]string{
	"TABLE":      "AlterTable",
	"DATABASE":   "AlterDatabase",
	"STATISTICS": "AlterStatistics",
	"INDEX":      "AlterIndex",
	"SEQUENCE":   "AlterSequence",
}

func unparsedDDLFamily(sql string) string {
	words := leadingKnownDDLKeywords(sql)
	if len(words) == 0 {
		return "Unknown"
	}
	switch words[0] {
	case "GRANT":
		return "Grant"
	case "REVOKE":
		return "Revoke"
	case "CREATE":
		return lookupKnownFamily("Create", createFamilies, words[1:])
	case "ALTER":
		return lookupKnownFamily("Alter", alterFamilies, words[1:])
	default:
		return "Unknown"
	}
}

func leadingKnownDDLKeywords(sql string) []string {
	lex := &memefish.Lexer{File: &token.File{Buffer: sql}}
	var words []string
	for range 12 {
		if err := lex.NextToken(); err != nil {
			break
		}
		if lex.Token.Kind == token.TokenEOF {
			break
		}
		word, ok := knownDDLKeyword(lex.Token)
		if !ok {
			continue
		}
		switch word {
		case "OR", "REPLACE", "IF", "NOT", "EXISTS":
			continue
		}
		words = append(words, word)
		if len(words) == 4 || (len(words) == 1 && (word == "GRANT" || word == "REVOKE")) {
			break
		}
	}
	return words
}

func knownDDLKeyword(tok token.Token) (string, bool) {
	var word string
	switch tok.Kind {
	case token.TokenIdent:
		word = strings.ToUpper(tok.Raw)
	default:
		raw := string(tok.Kind)
		for _, c := range raw {
			if c < 'A' || c > 'Z' {
				return "", false
			}
		}
		word = raw
	}
	if word == "" {
		return "", false
	}
	_, ok := knownDDLKeywords[word]
	return word, ok
}

func lookupKnownFamily(prefix string, families map[string]string, rest []string) string {
	for n := len(rest); n >= 1; n-- {
		if family, ok := families[strings.Join(rest[:n], " ")]; ok {
			return family
		}
	}
	return prefix
}
