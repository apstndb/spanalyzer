package main

import (
	"path/filepath"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestCurrentBundle(t *testing.T) {
	if err := run([]string{"--repo-root", "../..", "--gate", "all"}); err != nil {
		t.Fatal(err)
	}
}

func TestFindRepoRoot(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, start := range []string{
		root,
		filepath.Join(root, "tools"),
		filepath.Join(root, "tools", "okf-check"),
	} {
		t.Run(start, func(t *testing.T) {
			got, err := findRepoRoot(start)
			if err != nil {
				t.Fatal(err)
			}
			if got != root {
				t.Fatalf("findRepoRoot(%q) = %q, want %q", start, got, root)
			}
		})
	}
}

func TestSplitDocument(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantFront string
		wantBody  string
		wantFound bool
		wantError bool
	}{
		{
			name:      "concept",
			input:     "---\ntype: Reference\n---\n# Body\n",
			wantFront: "type: Reference",
			wantBody:  "# Body\n",
			wantFound: true,
		},
		{
			name:     "index without frontmatter",
			input:    "# Index\n",
			wantBody: "# Index\n",
		},
		{
			name:      "unterminated",
			input:     "---\ntype: Reference\n",
			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			front, body, found, err := splitDocument(tt.input)
			if (err != nil) != tt.wantError {
				t.Fatalf("splitDocument() error = %v, wantError %v", err, tt.wantError)
			}
			if tt.wantError {
				return
			}
			if front != tt.wantFront || body != tt.wantBody || found != tt.wantFound {
				t.Fatalf("splitDocument() = (%q, %q, %v), want (%q, %q, %v)", front, body, found, tt.wantFront, tt.wantBody, tt.wantFound)
			}
		})
	}
}

func TestValidateAssetPattern(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		wantError bool
	}{
		{name: "repository relative", pattern: "schemas/*.json"},
		{name: "empty", wantError: true},
		{name: "absolute", pattern: "/schemas/*.json", wantError: true},
		{name: "escapes repository", pattern: "../schemas/*.json", wantError: true},
		{name: "inside bundle", pattern: "knowledge/*.md", wantError: true},
		{name: "invalid glob", pattern: "schemas/[.json", wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAssetPattern(tt.pattern)
			if (err != nil) != tt.wantError {
				t.Fatalf("validateAssetPattern(%q) error = %v, wantError %v", tt.pattern, err, tt.wantError)
			}
		})
	}
}

func TestExplicitLocalResource(t *testing.T) {
	tests := []struct {
		resource string
		want     bool
	}{
		{resource: "../../README.md", want: true},
		{resource: "/concepts/example.md", want: true},
		{resource: "https://example.com/spec"},
		{resource: "all queries in project example"},
	}
	for _, tt := range tests {
		if got := explicitLocalResource(tt.resource); got != tt.want {
			t.Errorf("explicitLocalResource(%q) = %v, want %v", tt.resource, got, tt.want)
		}
	}
}

func TestLegacyResearchAllowlist(t *testing.T) {
	baseline, err := parseLegacyResearchMarkdown(legacyResearchBaseline)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(baseline); got != legacyResearchInitial {
		t.Fatalf("legacy baseline has %d paths, want %d", got, legacyResearchInitial)
	}
	allowed, err := parseLegacyResearchMarkdown(legacyResearchMarkdown)
	if err != nil {
		t.Fatal(err)
	}
	if unknown := setDifference(allowed, baseline); len(unknown) != 0 {
		t.Fatalf("legacy allowlist has paths outside baseline: %v", unknown)
	}
	empty, err := parseLegacyResearchMarkdown("# no active legacy bodies remain\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("empty active allowlist parsed as %v", empty)
	}
}

func TestCheckLegacyResearchMembership(t *testing.T) {
	baseline := map[string]struct{}{
		"research/area/a.md": {},
		"research/area/b.md": {},
	}
	tests := []struct {
		name      string
		tracked   []string
		allowed   map[string]struct{}
		wantError bool
	}{
		{
			name:    "exact",
			tracked: []string{legacyResearchIndex, "research/area/a.md", "research/area/b.md"},
			allowed: baseline,
		},
		{
			name:    "first shrink",
			tracked: []string{legacyResearchIndex, "research/area/b.md"},
			allowed: map[string]struct{}{"research/area/b.md": {}},
		},
		{
			name:    "final shrink to zero",
			tracked: []string{legacyResearchIndex},
			allowed: map[string]struct{}{},
		},
		{
			name:      "unapproved nested note",
			tracked:   []string{legacyResearchIndex, "research/area/a.md", "research/area/b.md", "research/area/new.md"},
			allowed:   baseline,
			wantError: true,
		},
		{
			name:      "unapproved root note",
			tracked:   []string{legacyResearchIndex, "research/area/a.md", "research/area/b.md", "research/new.md"},
			allowed:   baseline,
			wantError: true,
		},
		{
			name:      "same count nested substitution",
			tracked:   []string{legacyResearchIndex, "research/area/a.md", "research/area/new.md"},
			allowed:   map[string]struct{}{"research/area/a.md": {}, "research/area/new.md": {}},
			wantError: true,
		},
		{
			name:      "same count root substitution",
			tracked:   []string{legacyResearchIndex, "research/area/a.md", "research/new.md"},
			allowed:   map[string]struct{}{"research/area/a.md": {}, "research/new.md": {}},
			wantError: true,
		},
		{
			name:      "post shrink regrowth outside baseline",
			tracked:   []string{legacyResearchIndex, "research/area/b.md", "research/area/new.md"},
			allowed:   map[string]struct{}{"research/area/b.md": {}, "research/area/new.md": {}},
			wantError: true,
		},
		{
			name:      "stale allowlist entry",
			tracked:   []string{legacyResearchIndex, "research/area/a.md"},
			allowed:   baseline,
			wantError: true,
		},
		{
			name:      "missing policy index",
			tracked:   []string{"research/area/a.md", "research/area/b.md"},
			allowed:   baseline,
			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkLegacyResearchMembership(tt.tracked, baseline, tt.allowed)
			if (err != nil) != tt.wantError {
				t.Fatalf("checkLegacyResearchMembership() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestCheckResearchNote(t *testing.T) {
	valid := frontMatter{
		Type:        researchNoteType,
		Title:       "A retained experiment",
		Description: "Records one bounded result.",
		Tags:        []string{"spanner"},
		Status:      "draft",
		Sources:     []source{{Resource: "scope descriptor"}},
	}
	if err := checkResearchNote(document{Path: "knowledge/research/query-plan/example.md", Front: valid}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		mutate func(*frontMatter)
	}{
		{name: "title", mutate: func(front *frontMatter) { front.Title = "" }},
		{name: "description", mutate: func(front *frontMatter) { front.Description = "" }},
		{name: "tags", mutate: func(front *frontMatter) { front.Tags = nil }},
		{name: "status", mutate: func(front *frontMatter) { front.Status = "" }},
		{name: "invalid status", mutate: func(front *frontMatter) { front.Status = "complete" }},
		{name: "sources", mutate: func(front *frontMatter) { front.Sources = nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			front := valid
			tt.mutate(&front)
			if err := checkResearchNote(document{Path: "knowledge/research/query-plan/example.md", Front: front}); err == nil {
				t.Fatal("checkResearchNote() succeeded, want error")
			}
		})
	}

	for _, doc := range []document{
		{Path: "knowledge/research/example.md", Front: valid},
		{Path: "knowledge/observations/example.md", Front: valid},
		{Path: "knowledge/research/query-plan/example.md", Front: frontMatter{Type: "Observation"}},
	} {
		if err := checkResearchNote(doc); err == nil {
			t.Errorf("checkResearchNote(%q) succeeded, want path/type error", doc.Path)
		}
	}
	if err := checkResearchNote(document{Path: "knowledge/research/query-plan/index.md"}); err != nil {
		t.Fatalf("reserved research index rejected: %v", err)
	}
}

func TestCheckOptionalTimestamp(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{name: "absent"},
		{name: "utc", value: "2026-08-24T00:00:00Z"},
		{name: "offset", value: "2026-08-24T09:00:00+09:00"},
		{name: "largest offset", value: "2026-08-24T00:00:00+23:59"},
		{name: "date only", value: "2026-08-24", wantError: true},
		{name: "no offset", value: "2026-08-24T00:00:00", wantError: true},
		{name: "single digit hour", value: "2026-08-24T0:00:00Z", wantError: true},
		{name: "comma fractional separator", value: "2026-08-24T00:00:00,1Z", wantError: true},
		{name: "positive hour 24", value: "2026-08-24T00:00:00+24:00", wantError: true},
		{name: "negative hour 24", value: "2026-08-24T00:00:00-24:00", wantError: true},
		{name: "positive minute 60", value: "2026-08-24T00:00:00+00:60", wantError: true},
		{name: "negative minute 60", value: "2026-08-24T00:00:00-00:60", wantError: true},
		{name: "leap second", value: "2026-12-31T23:59:60Z", wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkOptionalTimestamp("knowledge/example.md", "stale_after", tt.value)
			if (err != nil) != tt.wantError {
				t.Fatalf("checkOptionalTimestamp(%q) error = %v, wantError %v", tt.value, err, tt.wantError)
			}
		})
	}
}

func TestVerifiedEventsAcceptMappingAndList(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "mapping",
			input: "verified: { by: human:reviewer, at: 2026-08-24T00:00:00Z }\n",
			want:  1,
		},
		{
			name:  "list",
			input: "verified:\n  - { by: process:first, at: 2026-08-24T00:00:00Z }\n  - { by: human:second, at: 2026-08-24T01:00:00+01:00 }\n",
			want:  2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var front frontMatter
			if err := yaml.Unmarshal([]byte(tt.input), &front); err != nil {
				t.Fatal(err)
			}
			if got := len(front.Verified); got != tt.want {
				t.Fatalf("len(verified) = %d, want %d", got, tt.want)
			}
			if err := checkLifecycleAndTrustFields(document{Path: "knowledge/example.md", Front: front}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
