// Command markdown-language-check rejects Japanese and CJK text in tracked
// Markdown while allowing narrowly annotated source text.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

const allowNonEnglishMarker = "<!-- language-check: allow-non-english -->"

type violation struct {
	path string
	line int
}

func main() {
	repoRoot := flag.String("repo-root", ".", "repository root")
	flag.Parse()

	if err := run(*repoRoot); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}

	paths, err := trackedMarkdown(absRoot)
	if err != nil {
		return err
	}
	violations, err := checkMarkdownFiles(absRoot, paths)
	if err != nil {
		return err
	}
	if len(violations) == 0 {
		return nil
	}

	var message strings.Builder
	message.WriteString("tracked Markdown contains unapproved Japanese or CJK text:")
	for _, item := range violations {
		fmt.Fprintf(&message, "\n  %s:%d", item.path, item.line)
	}
	message.WriteString("\nTranslate the text or add the narrow same-line marker ")
	message.WriteString(allowNonEnglishMarker)
	message.WriteString(" for an intentional source title or quotation.")
	return errors.New(message.String())
}

func trackedMarkdown(root string) ([]string, error) {
	output, err := exec.Command("git", "-C", root, "ls-files", "-z", "--", "*.md").Output()
	if err != nil {
		return nil, fmt.Errorf("list tracked Markdown: %w", err)
	}

	var paths []string
	for _, name := range strings.Split(string(output), "\x00") {
		if name != "" {
			paths = append(paths, name)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func checkMarkdownFiles(root string, paths []string) ([]violation, error) {
	var violations []violation
	for _, name := range paths {
		file, err := os.Open(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", name, err)
		}

		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for lineNumber := 1; scanner.Scan(); lineNumber++ {
			line := scanner.Text()
			if containsJapaneseOrCJK(line) && !strings.Contains(line, allowNonEnglishMarker) {
				violations = append(violations, violation{path: name, line: lineNumber})
			}
		}
		scanErr := scanner.Err()
		closeErr := file.Close()
		if scanErr != nil {
			return nil, fmt.Errorf("scan %s: %w", name, scanErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close %s: %w", name, closeErr)
		}
	}
	return violations, nil
}

func containsJapaneseOrCJK(line string) bool {
	for _, r := range line {
		if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana) {
			return true
		}
	}
	return false
}
