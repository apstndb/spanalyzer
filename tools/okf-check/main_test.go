package main

import (
	"path/filepath"
	"testing"
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
