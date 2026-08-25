package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "exports manifest",
			args:       []string{"--source-commit", strings.Repeat("a", 40)},
			wantCode:   0,
			wantStdout: `"schema_version": "v0alpha1"`,
		},
		{
			name:       "requires source commit",
			wantCode:   2,
			wantStderr: "--source-commit is required",
		},
		{
			name:       "rejects positional argument",
			args:       []string{"--source-commit", strings.Repeat("a", 40), "extra"},
			wantCode:   2,
			wantStderr: "unexpected positional arguments",
		},
		{
			name:       "rejects malformed commit",
			args:       []string{"--source-commit", "not-a-commit"},
			wantCode:   2,
			wantStderr: "must contain 40 lowercase hexadecimal characters",
		},
		{
			name:       "help",
			args:       []string{"--help"},
			wantCode:   0,
			wantStderr: "Usage of spanner-sys-export",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			gotCode := run(test.args, &stdout, &stderr)
			if gotCode != test.wantCode {
				t.Errorf("run() code = %d, want %d", gotCode, test.wantCode)
			}
			if !strings.Contains(stdout.String(), test.wantStdout) {
				t.Errorf("stdout = %q, want substring %q", stdout.String(), test.wantStdout)
			}
			if !strings.Contains(stderr.String(), test.wantStderr) {
				t.Errorf("stderr = %q, want substring %q", stderr.String(), test.wantStderr)
			}
		})
	}
}
