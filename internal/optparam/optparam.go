// Package optparam models optional query parameters that drive dynamic SQL
// generation at codegen time.
//
// It is intentionally isolated from internal/querygen so the marker grammar,
// variant enumeration, and runtime builder generation remain reusable and easy
// to test apart from the v1alpha config shape.
//
// Optional shapes are modeled by per-segment markers in the SQL
// template:
//
//   - ModeRequired: parameter is always supplied; no SQL variation.
//
//   - ModeNullIsNull (marker /*?null_is_null:NAME*/ ... = @NAME ... /*?end*/):
//     the body must contain "= @NAME" which is rewritten to
//     "IS NOT DISTINCT FROM @NAME" at codegen time. A single SQL is
//     produced; the caller passes a nullable wrapper at runtime.
//
//   - ModeOmitWhenNull (marker /*?optional:NAME*/ ... /*?end*/):
//     when the caller leaves the *T pointer nil, the whole marker block
//     is removed from the SQL. Multiplies the variant count by 2.
//
//   - ModeOmitWhenEmpty (marker /*?empty:NAME*/ ... IN UNNEST(@NAME) ... /*?end*/):
//     when len([]T) is 0 the block is removed. Multiplies the variant
//     count by 2. SQL-wise indistinguishable from OmitWhenNull; the
//     difference is the runtime gating condition and Go type.
//
//   - ModeOrderByChoice (marker /*?orderby:NAME*/ <default> /*?end*/):
//     the body of the marker is replaced wholesale by one of the
//     declared Choices. Multiplies the variant count by len(Choices).
//
//   - ModeSQLChoice (marker /*?choice:NAME*/ <default> /*?end*/):
//     substitutes one declared SQL fragment, including correlated predicates
//     or a column selection. Only bindings referenced by the selected SQL
//     are supplied. Choices must all produce the same result row type.
//
// EnumerateVariants takes the Cartesian product across every kind of
// segment. VerifyVariants confirms every product point analyzes to the
// same row type. EmitGoBuilder generates a Go function that walks the
// segment list linearly at runtime and is byte-equal to whichever
// verified variant matches the call-site inputs.
package optparam

import (
	"fmt"
	"maps"
	"regexp"
	"sort"
	"strings"

	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/token"
)

// maxEnumeratedVariants bounds the codegen-time Cartesian product. Each
// variant is analyzed independently and retained in the generated plan, so an
// unbounded product can exhaust memory long before integer overflow occurs.
const maxEnumeratedVariants = 4096

// Mode describes how an optional parameter shapes the generated SQL.
type Mode int

const (
	ModeRequired Mode = iota
	ModeNullIsNull
	ModeOmitWhenNull
	ModeOmitWhenEmpty
	ModeOrderByChoice
	ModeSQLChoice
)

func (m Mode) String() string {
	switch m {
	case ModeRequired:
		return "required"
	case ModeNullIsNull:
		return "null_is_null"
	case ModeOmitWhenNull:
		return "omit_when_null"
	case ModeOmitWhenEmpty:
		return "omit_when_empty"
	case ModeOrderByChoice:
		return "orderby_choice"
	case ModeSQLChoice:
		return "sql_choice"
	default:
		return fmt.Sprintf("mode(%d)", int(m))
	}
}

// Param describes one named query parameter.
type Param struct {
	Name string
	// Type is the GoogleSQL type spec, e.g. "STRING", "INT64",
	// "ARRAY<STRING>". For choice selectors this is ignored.
	Type string
	Mode Mode
	// Choices is the set of allowed fragments for a choice selector,
	// keyed by the identifier the runtime caller selects. ModeOrderByChoice
	// values are full ORDER BY clauses; ModeSQLChoice values are SQL fragments.
	Choices map[string]string
	// Default is the choice key used when the caller does not specify
	// one. Must match a key in Choices.
	Default string
}

// SegmentKind classifies how a Segment contributes to the rendered SQL.
type SegmentKind int

const (
	// SegFixed unconditionally emits Text.
	SegFixed SegmentKind = iota
	// SegOmitWhenNull emits Text iff the caller's *T pointer is non-nil.
	SegOmitWhenNull
	// SegOmitWhenEmpty emits Text iff len([]T) > 0.
	SegOmitWhenEmpty
	// SegOrderByChoice emits one of Choices keyed by the caller's
	// choice string. Text holds the default-choice body (used when
	// the SQL is interpreted without the framework).
	SegOrderByChoice
	// SegSQLChoice substitutes a closed SQL fragment and tracks its bindings.
	SegSQLChoice
)

// Segment is the unit of the parsed SQL template. The template is a flat
// slice of segments shared by EnumerateVariants (build-time verification)
// and EmitGoBuilder (runtime composition).
type Segment struct {
	Kind SegmentKind
	// Text is the segment body, with marker tokens already stripped.
	Text string
	// Param is the parameter that gates or selects the segment. Empty
	// for SegFixed.
	Param string
	// Choices is populated for choice segments with the same map as
	// Param.Choices, snapshotted at parse time.
	Choices map[string]string
	// Default is populated for choice segments with the default choice
	// key.
	Default string
}

// Variant is one concrete SQL produced by EnumerateVariants.
type Variant struct {
	SQL string
	// PresentParams names omit/empty params whose block is kept.
	PresentParams []string
	// AbsentParams names omit/empty params whose block is removed. For templates
	// with SQL choices it includes every real bind not referenced by this SQL.
	AbsentParams []string
	// ChoiceAssignments records the choice picked for each
	// choice selector, keyed by param name.
	ChoiceAssignments map[string]string
}

// Key returns a stable key suitable for indexing into a result map.
func (v Variant) Key() string {
	parts := append([]string(nil), v.PresentParams...)
	for name, choice := range v.ChoiceAssignments {
		parts = append(parts, name+"="+choice)
	}
	sort.Strings(parts)
	if len(parts) == 0 {
		return "(none)"
	}
	return strings.Join(parts, "+")
}

var markerRegexp = regexp.MustCompile(
	`(?s)/\*\?(optional|empty|null_is_null|orderby|choice):([A-Za-z_][A-Za-z0-9_]*)\*/(.*?)/\*\?end\*/`)

// SegmentTemplate splits sql at every recognized marker pair, validating
// each block against params. The returned slice preserves source order.
func SegmentTemplate(sql string, params []Param) ([]Segment, error) {
	byName := map[string]Param{}
	for _, p := range params {
		if p.Name == "" {
			return nil, fmt.Errorf("param name is required")
		}
		if _, dup := byName[p.Name]; dup {
			return nil, fmt.Errorf("duplicate param %q", p.Name)
		}
		byName[p.Name] = p
	}

	matches := markerRegexp.FindAllStringSubmatchIndex(sql, -1)
	var segments []Segment
	prev := 0
	seenBlock := map[string]bool{}
	for _, m := range matches {
		markerKind := sql[m[2]:m[3]]
		name := sql[m[4]:m[5]]
		body := sql[m[6]:m[7]]
		p, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("marker /*?%s:%s*/ references unknown param", markerKind, name)
		}
		expectedMode, segKind, err := markerExpectedMode(markerKind)
		if err != nil {
			return nil, err
		}
		if p.Mode != expectedMode {
			return nil, fmt.Errorf("marker /*?%s:%s*/ requires mode %s, got %s",
				markerKind, name, expectedMode, p.Mode)
		}
		seenBlock[name] = true
		if m[0] > prev {
			segments = append(segments, Segment{Kind: SegFixed, Text: sql[prev:m[0]]})
		}
		seg := Segment{Kind: segKind, Param: name}
		switch segKind {
		case SegOmitWhenNull, SegOmitWhenEmpty:
			seg.Text = body
		case SegOrderByChoice, SegSQLChoice:
			if len(p.Choices) == 0 {
				return nil, fmt.Errorf("param %q (%s): Choices is empty", name, markerKind)
			}
			if p.Default == "" {
				return nil, fmt.Errorf("param %q (%s): Default is required", name, markerKind)
			}
			if _, ok := p.Choices[p.Default]; !ok {
				return nil, fmt.Errorf("param %q (%s): Default %q is not in Choices", name, markerKind, p.Default)
			}
			seg.Text = body
			seg.Choices = maps.Clone(p.Choices)
			seg.Default = p.Default
		case SegFixed:
			// null_is_null rewrites the body and merges into a fixed
			// segment so it is emitted unconditionally.
			rewritten, err := rewriteNullIsNull(body, name)
			if err != nil {
				return nil, fmt.Errorf("param %q (null_is_null): %w", name, err)
			}
			seg.Param = ""
			seg.Text = rewritten
		}
		segments = append(segments, seg)
		prev = m[1]
	}
	if prev < len(sql) {
		segments = append(segments, Segment{Kind: SegFixed, Text: sql[prev:]})
	}

	for _, p := range params {
		switch p.Mode {
		case ModeOmitWhenNull, ModeOmitWhenEmpty, ModeOrderByChoice, ModeSQLChoice:
			// These modes change the SQL string at codegen time, so the
			// template must contain the marker for us to know where to
			// gate or substitute.
			if !seenBlock[p.Name] {
				return nil, fmt.Errorf("param %q has mode %s but no marker was found in the SQL", p.Name, p.Mode)
			}
		case ModeNullIsNull:
			// null_is_null is allowed with or without a marker. With a
			// marker (kind: sql) the body is rewritten in place; without
			// one (kind: index, kind: table) the rewrite was already
			// emitted by the SQL generator.
		}
	}
	if hasSQLChoice(segments) {
		if err := validateSQLChoiceBindings(segments, params); err != nil {
			return nil, err
		}
	}
	return segments, nil
}

func markerExpectedMode(markerKind string) (Mode, SegmentKind, error) {
	switch markerKind {
	case "optional":
		return ModeOmitWhenNull, SegOmitWhenNull, nil
	case "empty":
		return ModeOmitWhenEmpty, SegOmitWhenEmpty, nil
	case "null_is_null":
		return ModeNullIsNull, SegFixed, nil
	case "orderby":
		return ModeOrderByChoice, SegOrderByChoice, nil
	case "choice":
		return ModeSQLChoice, SegSQLChoice, nil
	default:
		return ModeRequired, SegFixed, fmt.Errorf("unknown marker kind %q", markerKind)
	}
}

// rewriteNullIsNull replaces comparison tokens "= @name" with
// "IS NOT DISTINCT FROM @name". The memefish lexer decides which equals
// signs are code tokens, so strings, quoted identifiers, and comments,
// including hash comments, stay byte-for-byte. At least one real comparison
// is required.
func rewriteNullIsNull(body, name string) (string, error) {
	tokens, err := lexSQLFragment(body)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	last := 0
	replaced := false
	for i, tok := range tokens {
		if tok.Kind != "=" || i+1 >= len(tokens) || !sqlParamToken(tokens[i+1], name) {
			continue
		}
		start := int(tok.Pos)
		param := tokens[i+1]
		between := body[int(tok.End):int(param.Pos)]
		b.WriteString(body[last:start])
		if start > 0 && !isSQLSpace(body[start-1]) {
			b.WriteByte(' ')
		}
		b.WriteString("IS NOT DISTINCT FROM")
		if !containsSQLSpace(between) {
			b.WriteByte(' ')
		}
		b.WriteString(between)
		b.WriteString("@" + name)
		last = int(param.End)
		replaced = true
	}
	if !replaced {
		return "", fmt.Errorf("body %q does not contain `= @%s`", body, name)
	}
	b.WriteString(body[last:])
	return b.String(), nil
}

func lexSQLFragment(body string) ([]token.Token, error) {
	lexer := &memefish.Lexer{File: &token.File{Buffer: body}}
	var tokens []token.Token
	for {
		if err := lexer.NextToken(); err != nil {
			return nil, fmt.Errorf("SQL fragment: %w", err)
		}
		if lexer.Token.Kind == token.TokenBad {
			return nil, fmt.Errorf("SQL fragment: invalid SQL token at %d", lexer.Token.Pos)
		}
		if lexer.Token.Kind == token.TokenEOF {
			return tokens, nil
		}
		tokens = append(tokens, lexer.Token)
	}
}

func sqlParamToken(tok token.Token, name string) bool {
	return tok.Kind == token.TokenParam && strings.EqualFold(tok.AsString, name)
}

func containsSQLSpace(s string) bool {
	for i := 0; i < len(s); i++ {
		if isSQLSpace(s[i]) {
			return true
		}
	}
	return false
}

func isSQLSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\f':
		return true
	default:
		return false
	}
}

// Presence carries the per-variant inputs to ComposeVariant.
type Presence struct {
	// Present[name] == true means the omit/empty block for `name` is kept.
	Present map[string]bool
	// Choices[name] is the choice key picked for the orderby segment
	// keyed by `name`.
	Choices map[string]string
}

// ComposeVariant assembles SQL from segments. The output is byte-identical
// to the verified variant EnumerateVariants would produce for the same
// inputs — this invariant is what the runtime composer in EmitGoBuilder
// relies on.
func ComposeVariant(segments []Segment, p Presence) string {
	var b strings.Builder
	for _, s := range segments {
		switch s.Kind {
		case SegFixed:
			b.WriteString(s.Text)
		case SegOmitWhenNull, SegOmitWhenEmpty:
			if p.Present[s.Param] {
				b.WriteString(s.Text)
			}
		case SegOrderByChoice, SegSQLChoice:
			key := p.Choices[s.Param]
			if key == "" {
				key = s.Default
			}
			b.WriteString(s.Choices[key])
		}
	}
	return b.String()
}

// EnumerateVariants returns every product point: each on/off combination
// for omit/empty segments crossed with every choice for orderby segments.
func EnumerateVariants(sql string, params []Param) ([]Variant, error) {
	segments, err := SegmentTemplate(sql, params)
	if err != nil {
		return nil, err
	}
	gateable, choiceable := segmentAxes(segments)

	// Two-dimensional Cartesian product: 2^k presence combinations ×
	// product-of-len(choices) for each orderby param.
	type choiceAxis struct {
		param string
		keys  []string
	}
	var choiceAxes []choiceAxis
	for _, name := range choiceable {
		var seg *Segment
		for i := range segments {
			if (segments[i].Kind == SegOrderByChoice || segments[i].Kind == SegSQLChoice) && segments[i].Param == name {
				seg = &segments[i]
				break
			}
		}
		keys := make([]string, 0, len(seg.Choices))
		for k := range seg.Choices {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		choiceAxes = append(choiceAxes, choiceAxis{param: name, keys: keys})
	}

	presenceTotal := 1
	for range gateable {
		if presenceTotal > maxEnumeratedVariants/2 {
			return nil, variantLimitError()
		}
		presenceTotal *= 2
	}
	choiceTotal := 1
	for _, ax := range choiceAxes {
		if choiceTotal > maxEnumeratedVariants/len(ax.keys) {
			return nil, variantLimitError()
		}
		choiceTotal *= len(ax.keys)
	}
	if presenceTotal > maxEnumeratedVariants/choiceTotal {
		return nil, variantLimitError()
	}
	total := presenceTotal * choiceTotal

	variants := make([]Variant, 0, total)
	for mask := range presenceTotal {
		present := map[string]bool{}
		for i, name := range gateable {
			if mask&(1<<i) != 0 {
				present[name] = true
			}
		}
		// Enumerate every choice tuple via a mixed-radix counter.
		for c := range choiceTotal {
			choices := map[string]string{}
			rem := c
			for _, ax := range choiceAxes {
				n := len(ax.keys)
				choices[ax.param] = ax.keys[rem%n]
				rem /= n
			}
			v := Variant{
				SQL:               ComposeVariant(segments, Presence{Present: present, Choices: choices}),
				ChoiceAssignments: choices,
			}
			for _, name := range gateable {
				if present[name] {
					v.PresentParams = append(v.PresentParams, name)
				} else {
					v.AbsentParams = append(v.AbsentParams, name)
				}
			}
			if hasSQLChoice(segments) {
				bound, err := bindingNames(v.SQL, params)
				if err != nil {
					return nil, fmt.Errorf("variant %s: %w", v.Key(), err)
				}
				used := map[string]bool{}
				for _, name := range bound {
					used[name] = true
				}
				v.AbsentParams = nil
				for _, p := range params {
					if p.Mode != ModeOrderByChoice && p.Mode != ModeSQLChoice && !used[p.Name] {
						v.AbsentParams = append(v.AbsentParams, p.Name)
					}
				}
				sort.Strings(v.AbsentParams)
			}
			variants = append(variants, v)
		}
	}
	sort.SliceStable(variants, func(i, j int) bool {
		return variants[i].Key() < variants[j].Key()
	})
	return variants, nil
}

func variantLimitError() error {
	return fmt.Errorf("optional parameter variants exceed limit %d; reduce optional parameters or choices", maxEnumeratedVariants)
}

// segmentAxes returns the sorted names of (gateable, choiceable)
// segments. Gateable = omit/empty. Choiceable = orderby/SQL choice.
func segmentAxes(segments []Segment) (gateable, choiceable []string) {
	seenG, seenC := map[string]bool{}, map[string]bool{}
	for _, s := range segments {
		switch s.Kind {
		case SegOmitWhenNull, SegOmitWhenEmpty:
			if !seenG[s.Param] {
				seenG[s.Param] = true
				gateable = append(gateable, s.Param)
			}
		case SegOrderByChoice, SegSQLChoice:
			if !seenC[s.Param] {
				seenC[s.Param] = true
				choiceable = append(choiceable, s.Param)
			}
		}
	}
	sort.Strings(gateable)
	sort.Strings(choiceable)
	return
}

func hasSQLChoice(segments []Segment) bool {
	for _, seg := range segments {
		if seg.Kind == SegSQLChoice {
			return true
		}
	}
	return false
}

// bindingNames uses SQL tokens, not text substitution: comments, quoted
// identifiers and string literals containing @names never become bindings.
// Returned names use the configured spelling so argument maps stay consistent.
func bindingNames(sql string, params []Param) ([]string, error) {
	byName := make(map[string]Param, len(params))
	for _, p := range params {
		key := strings.ToLower(p.Name)
		if _, exists := byName[key]; exists {
			return nil, fmt.Errorf("duplicate bind/selector name %q", p.Name)
		}
		byName[key] = p
	}
	tokens, err := lexSQLFragment(sql)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var names []string
	for _, tok := range tokens {
		if tok.Kind != token.TokenParam {
			continue
		}
		p, ok := byName[strings.ToLower(tok.AsString)]
		if !ok {
			return nil, fmt.Errorf("undeclared bind parameter @%s", tok.AsString)
		}
		if p.Mode == ModeSQLChoice || p.Mode == ModeOrderByChoice {
			return nil, fmt.Errorf("choice selector %q cannot be a bind parameter", p.Name)
		}
		if !seen[p.Name] {
			seen[p.Name] = true
			names = append(names, p.Name)
		}
	}
	sort.Strings(names)
	return names, nil
}

func validateSQLChoiceBindings(segments []Segment, params []Param) error {
	byName := make(map[string]Param, len(params))
	for _, p := range params {
		byName[p.Name] = p
	}
	for _, seg := range segments {
		fragments := []string{seg.Text}
		if seg.Kind == SegSQLChoice || seg.Kind == SegOrderByChoice {
			fragments = nil
			for _, key := range sortedChoiceKeys(seg.Choices) {
				fragments = append(fragments, seg.Choices[key])
			}
		}
		for _, fragment := range fragments {
			names, err := bindingNames(fragment, params)
			if err != nil {
				return fmt.Errorf("segment %q: %w", seg.Param, err)
			}
			for _, name := range names {
				p := byName[name]
				if seg.Kind == SegSQLChoice && p.Mode != ModeRequired {
					return fmt.Errorf("SQL choice %q: bind parameter %q must be required; the choice controls its presence", seg.Param, name)
				}
				if (p.Mode == ModeOmitWhenNull || p.Mode == ModeOmitWhenEmpty) && seg.Param != name {
					return fmt.Errorf("bind parameter %q must stay inside its own optional block", name)
				}
			}
		}
	}
	return nil
}

func sortedChoiceKeys(choices map[string]string) []string {
	keys := make([]string, 0, len(choices))
	for key := range choices {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
