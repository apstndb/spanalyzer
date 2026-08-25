package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestContainsJapaneseOrCJK(t *testing.T) {
	for _, tt := range []struct {
		name string
		line string
		want bool
	}{
		{name: "English", line: "Managed Spanner evidence"},
		{name: "Hiragana", line: "ひらがな", want: true},
		{name: "Katakana", line: "カタカナ", want: true},
		{name: "Han", line: "実行計画", want: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsJapaneseOrCJK(tt.line); got != tt.want {
				t.Fatalf("containsJapaneseOrCJK(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestCheckMarkdownFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "english.md", "# English\n")
	writeFile(t, root, "source-title.md", "日本語 title "+allowNonEnglishMarker+"\n")
	writeFile(t, root, "mixed.md", "# English\n日本語 text\n")

	got, err := checkMarkdownFiles(root, []string{"english.md", "source-title.md", "mixed.md"})
	if err != nil {
		t.Fatal(err)
	}
	want := []violation{{path: "mixed.md", line: 2}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("checkMarkdownFiles() = %#v, want %#v", got, want)
	}
}

func TestCheckMarkdownFilesRejectsMissingTrackedPath(t *testing.T) {
	_, err := checkMarkdownFiles(t.TempDir(), []string{"missing.md"})
	if err == nil {
		t.Fatal("checkMarkdownFiles() succeeded for a missing tracked path")
	}
}

func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
