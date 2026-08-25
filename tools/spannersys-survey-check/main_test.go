package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const repositorySourceCommit = "91908d001349f844aac070cc6518119c0e3c36c0"

func TestRunValidatesRepositoryManifest(t *testing.T) {
	if err := run([]string{"--repo-root", "../.."}); err != nil {
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
		filepath.Join(root, "tools", "spannersys-survey-check"),
	} {
		got, err := findRepoRoot(start)
		if err != nil {
			t.Fatalf("findRepoRoot(%q) error = %v", start, err)
		}
		if got != root {
			t.Fatalf("findRepoRoot(%q) = %q, want %q", start, got, root)
		}
	}
}

func TestReadAndValidateManifestInvokesInternalCommand(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, defaultManifestPath)
	data := repositoryManifest(t)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	runner := func(directory, name string, arguments ...string) ([]byte, error) {
		if directory != root || name != "go" {
			t.Fatalf("runner = dir %q command %q, want dir %q command go", directory, name, root)
		}
		want := []string{"-C", root, "run", "./internal/spannersysmanifest/checkcmd", "--manifest", path}
		if !reflect.DeepEqual(arguments, want) {
			t.Fatalf("runner arguments = %v, want %v", arguments, want)
		}
		return []byte(repositorySourceCommit + "\n"), nil
	}
	manifest, err := readAndValidateManifest(root, path, runner)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.SourceCommit != repositorySourceCommit {
		t.Fatalf("validated source commit = %q, want %q", manifest.SourceCommit, repositorySourceCommit)
	}
	if !reflect.DeepEqual(manifest.Bytes, data) {
		t.Fatal("validated manifest bytes differ from the input")
	}
}

func TestReadAndValidateManifestRejectsAdditionalOutput(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, defaultManifestPath)
	if err := os.WriteFile(path, repositoryManifest(t), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := func(_ string, _ string, _ ...string) ([]byte, error) {
		return []byte(repositorySourceCommit + "\nunexpected\n"), nil
	}
	_, err := readAndValidateManifest(root, path, runner)
	if err == nil || !strings.Contains(err.Error(), "validator output") {
		t.Fatalf("readAndValidateManifest() error = %v, want additional-output failure", err)
	}
}

func TestCompareSurveyCheckoutExactBytes(t *testing.T) {
	manifestBytes := repositoryManifest(t)
	surveyRoot := t.TempDir()
	wantCalls := [][]string{
		{"git", "status", "--porcelain=v1", "--untracked-files=all"},
		{"git", "rev-parse", "HEAD"},
		{"go", "run", "./cmd/spanner-sys-export", "--source-commit", repositorySourceCommit},
	}
	callIndex := 0
	runner := func(directory, name string, arguments ...string) ([]byte, error) {
		if directory != surveyRoot {
			t.Fatalf("runner directory = %q, want %q", directory, surveyRoot)
		}
		call := append([]string{name}, arguments...)
		if callIndex >= len(wantCalls) {
			t.Fatalf("unexpected runner call %d = %v", callIndex, call)
		}
		if !reflect.DeepEqual(call, wantCalls[callIndex]) {
			t.Fatalf("runner call %d = %v, want %v", callIndex, call, wantCalls[callIndex])
		}
		callIndex++
		switch name {
		case "git":
			if arguments[0] == "status" {
				return nil, nil
			}
			return []byte(repositorySourceCommit + "\n"), nil
		case "go":
			return manifestBytes, nil
		default:
			return nil, errors.New("unexpected command")
		}
	}
	if err := compareSurveyCheckout(manifestBytes, repositorySourceCommit, surveyRoot, runner); err != nil {
		t.Fatal(err)
	}
	if callIndex != len(wantCalls) {
		t.Fatalf("runner call count = %d, want %d", callIndex, len(wantCalls))
	}
}

func TestCompareSurveyCheckoutRejectsDirtyWorktree(t *testing.T) {
	manifestBytes := repositoryManifest(t)
	runner := func(_ string, _ string, _ ...string) ([]byte, error) {
		return []byte("?? scratch.txt\n"), nil
	}
	err := compareSurveyCheckout(manifestBytes, repositorySourceCommit, t.TempDir(), runner)
	if err == nil || !strings.Contains(err.Error(), "is dirty") {
		t.Fatalf("compareSurveyCheckout() error = %v, want dirty-worktree failure", err)
	}
}

func TestCompareSurveyCheckoutRejectsWrongCommit(t *testing.T) {
	manifestBytes := repositoryManifest(t)
	call := 0
	runner := func(_ string, _ string, _ ...string) ([]byte, error) {
		call++
		if call == 1 {
			return nil, nil
		}
		return []byte("0123456789012345678901234567890123456789\n"), nil
	}
	err := compareSurveyCheckout(manifestBytes, repositorySourceCommit, t.TempDir(), runner)
	if err == nil || !strings.Contains(err.Error(), "manifest pins") {
		t.Fatalf("compareSurveyCheckout() error = %v, want pinned-commit failure", err)
	}
}

func TestCompareSurveyCheckoutRejectsByteDifference(t *testing.T) {
	manifestBytes := repositoryManifest(t)
	call := 0
	runner := func(_ string, name string, _ ...string) ([]byte, error) {
		call++
		switch call {
		case 1:
			return nil, nil
		case 2:
			return []byte(repositorySourceCommit + "\n"), nil
		case 3:
			return append(append([]byte(nil), manifestBytes...), ' '), nil
		default:
			return nil, errors.New("unexpected call")
		}
	}
	err := compareSurveyCheckout(manifestBytes, repositorySourceCommit, t.TempDir(), runner)
	if err == nil || !strings.Contains(err.Error(), "export differs") {
		t.Fatalf("compareSurveyCheckout() error = %v, want exact-byte failure", err)
	}
}

func TestRunRejectsExplicitInvalidSurveyRoot(t *testing.T) {
	err := run([]string{"--repo-root", "../..", "--survey-root", filepath.Join(t.TempDir(), "missing")})
	if err == nil || !strings.Contains(err.Error(), "invalid --survey-root") {
		t.Fatalf("run() error = %v, want invalid --survey-root", err)
	}
}

func repositoryManifest(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("../../spanner_sys_manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}
