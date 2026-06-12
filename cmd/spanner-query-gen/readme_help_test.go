package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestREADMEHelpOutputsAreCurrent pins the CLI help text embedded in
// README.md to the actual --help output, the same way the checked-in JSON
// schemas are pinned, so flag changes cannot silently drift from the
// documentation.
func TestREADMEHelpOutputsAreCurrent(t *testing.T) {
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("os.ReadFile(README.md) error = %v", err)
	}
	document := string(readme)
	cases := []struct {
		marker string
		args   []string
	}{
		{"Current root help:", []string{"--help"}},
		{"Current `config-schema --help` output:", []string{"config-schema", "--help"}},
		{"Current `plan-report-schema --help` output:", []string{"plan-report-schema", "--help"}},
		{"Current `plan-contract-schema --help` output:", []string{"plan-contract-schema", "--help"}},
		{"Current `plan-report --help` output:", []string{"plan-report", "--help"}},
	}
	for _, c := range cases {
		c := c
		t.Run(c.marker, func(t *testing.T) {
			want := readmeFencedBlockAfter(t, document, c.marker)
			var stdout, stderr bytes.Buffer
			if err := run(c.args, &stdout, &stderr); err != nil {
				t.Fatalf("run(%v) error = %v", c.args, err)
			}
			got := normalizeHelpOutput(stdout.String())
			if got != want {
				t.Errorf("README block for %q is stale.\n--- README ---\n%s\n--- actual ---\n%s", c.marker, want, got)
			}
		})
	}
}

// readmeFencedBlockAfter returns the content of the first ```text fenced
// block after marker, without the fences.
func readmeFencedBlockAfter(t *testing.T, document, marker string) string {
	t.Helper()
	index := strings.Index(document, marker)
	if index < 0 {
		t.Fatalf("README.md does not contain marker %q", marker)
	}
	rest := document[index:]
	const fence = "```text\n"
	start := strings.Index(rest, fence)
	if start < 0 {
		t.Fatalf("README.md has no ```text block after marker %q", marker)
	}
	rest = rest[start+len(fence):]
	end := strings.Index(rest, "```")
	if end < 0 {
		t.Fatalf("README.md ```text block after marker %q is not closed", marker)
	}
	return strings.TrimRight(rest[:end], "\n")
}

// normalizeHelpOutput converts the flag package's "four spaces plus tab"
// usage indentation into the eight-space indentation used by the README so
// the comparison is whitespace-stable.
func normalizeHelpOutput(s string) string {
	s = strings.ReplaceAll(s, "    \t", "        ")
	return strings.TrimRight(s, "\n")
}
