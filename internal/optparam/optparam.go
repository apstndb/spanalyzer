// Package optparam is a PoC for optional query parameters that drive dynamic
// SQL generation at codegen time.
//
// It is intentionally isolated from internal/querygen so the data model and
// algorithm can be validated independently before being promoted into the
// v1alpha config and the plan contract.
//
// Four optional shapes are modeled by per-segment markers in the SQL
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
// EnumerateVariants takes the Cartesian product across every kind of
// segment. VerifyVariants confirms every product point analyzes to the
// same row type. EmitGoBuilder generates a Go function that walks the
// segment list linearly at runtime and is byte-equal to whichever
// verified variant matches the call-site inputs.
package optparam

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Mode describes how an optional parameter shapes the generated SQL.
type Mode int

const (
	ModeRequired Mode = iota
	ModeNullIsNull
	ModeOmitWhenNull
	ModeOmitWhenEmpty
	ModeOrderByChoice
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
	default:
		return fmt.Sprintf("mode(%d)", int(m))
	}
}

// Param describes one named query parameter.
type Param struct {
	Name string
	// Type is the GoogleSQL type spec, e.g. "STRING", "INT64",
	// "ARRAY<STRING>". For ModeOrderByChoice this is ignored.
	Type string
	Mode Mode
	// Choices is the set of allowed ORDER BY clauses for an
	// ModeOrderByChoice param, keyed by an identifier the runtime
	// caller selects. Each value is a full ORDER BY ... clause.
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
	// Choices is populated for SegOrderByChoice with the same map as
	// Param.Choices, snapshotted at parse time.
	Choices map[string]string
	// Default is populated for SegOrderByChoice with the default choice
	// key.
	Default string
}

// Variant is one concrete SQL produced by EnumerateVariants.
type Variant struct {
	SQL string
	// PresentParams names omit/empty params whose block is kept.
	PresentParams []string
	// AbsentParams names omit/empty params whose block is removed.
	AbsentParams []string
	// ChoiceAssignments records the choice picked for each
	// ModeOrderByChoice param, keyed by param name.
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
	`(?s)/\*\?(optional|empty|null_is_null|orderby):([A-Za-z_][A-Za-z0-9_]*)\*/(.*?)/\*\?end\*/`)

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
		case SegOrderByChoice:
			if len(p.Choices) == 0 {
				return nil, fmt.Errorf("param %q (orderby): Choices is empty", name)
			}
			if p.Default == "" {
				return nil, fmt.Errorf("param %q (orderby): Default is required", name)
			}
			if _, ok := p.Choices[p.Default]; !ok {
				return nil, fmt.Errorf("param %q (orderby): Default %q is not in Choices", name, p.Default)
			}
			seg.Text = body
			seg.Choices = p.Choices
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
		case ModeOmitWhenNull, ModeOmitWhenEmpty, ModeOrderByChoice, ModeNullIsNull:
			if !seenBlock[p.Name] {
				return nil, fmt.Errorf("param %q has mode %s but no marker was found in the SQL", p.Name, p.Mode)
			}
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
	default:
		return ModeRequired, SegFixed, fmt.Errorf("unknown marker kind %q", markerKind)
	}
}

// rewriteNullIsNull replaces every "= @name" inside body with
// "IS NOT DISTINCT FROM @name". It is whitespace-tolerant around the "="
// token but requires at least one occurrence — otherwise the marker is
// almost certainly authored incorrectly.
func rewriteNullIsNull(body, name string) (string, error) {
	pattern := regexp.MustCompile(`(?i)=\s*@` + regexp.QuoteMeta(name) + `\b`)
	if !pattern.MatchString(body) {
		return "", fmt.Errorf("body %q does not contain `= @%s`", body, name)
	}
	return pattern.ReplaceAllString(body, "IS NOT DISTINCT FROM @"+name), nil
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
		case SegOrderByChoice:
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
			if segments[i].Kind == SegOrderByChoice && segments[i].Param == name {
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

	presenceTotal := 1 << len(gateable)
	choiceTotal := 1
	for _, ax := range choiceAxes {
		choiceTotal *= len(ax.keys)
	}

	variants := make([]Variant, 0, presenceTotal*choiceTotal)
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
			variants = append(variants, v)
		}
	}
	sort.SliceStable(variants, func(i, j int) bool {
		return variants[i].Key() < variants[j].Key()
	})
	return variants, nil
}

// segmentAxes returns the sorted names of (gateable, choiceable)
// segments. Gateable = omit/empty. Choiceable = orderby.
func segmentAxes(segments []Segment) (gateable, choiceable []string) {
	seenG, seenC := map[string]bool{}, map[string]bool{}
	for _, s := range segments {
		switch s.Kind {
		case SegOmitWhenNull, SegOmitWhenEmpty:
			if !seenG[s.Param] {
				seenG[s.Param] = true
				gateable = append(gateable, s.Param)
			}
		case SegOrderByChoice:
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
