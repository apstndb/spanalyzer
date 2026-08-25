// Command survey-import-check verifies the immutable mapping from the
// unpublished survey repository snapshot to its initial spanalyzer subtree.
package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const defaultProvenancePath = "survey/import-provenance.json"

type commandRunner func(directory, name string, arguments ...string) ([]byte, error)

type provenance struct {
	SchemaVersion string       `json:"schema_version"`
	Source        source       `json:"source"`
	Destination   destination  `json:"destination"`
	ExcludedPaths []string     `json:"excluded_paths"`
	Dispositions  dispositions `json:"dispositions"`
}

type source struct {
	Kind             string `json:"kind"`
	Module           string `json:"module"`
	Commit           string `json:"commit"`
	Tree             string `json:"tree"`
	CommitCount      int    `json:"commit_count"`
	HistoryPublished bool   `json:"history_published"`
}

type destination struct {
	Repository   string `json:"repository"`
	Path         string `json:"path"`
	ImportCommit string `json:"import_commit"`
	ImportTree   string `json:"import_tree"`
}

type dispositions struct {
	StandaloneRepositoryPublished bool   `json:"standalone_repository_published"`
	UntrackedMemefishFeedback     string `json:"untracked_memefish_feedback"`
	LocalEnvironment              string `json:"local_environment"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	return runWithRunner(arguments, runCommand)
}

func runWithRunner(arguments []string, runner commandRunner) error {
	flags := flag.NewFlagSet("survey-import-check", flag.ContinueOnError)
	repoRoot := flags.String("repo-root", "", "spanalyzer repository root (auto-detected when omitted)")
	provenancePath := flags.String("provenance", defaultProvenancePath, "provenance path relative to the repository root")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}

	root, err := resolveRepoRoot(*repoRoot)
	if err != nil {
		return err
	}
	document, err := readProvenance(filepath.Join(root, filepath.FromSlash(*provenancePath)))
	if err != nil {
		return err
	}
	if err := validateProvenance(document); err != nil {
		return err
	}
	return verifyGitMapping(root, document, runner)
}

func resolveRepoRoot(explicit string) (string, error) {
	if explicit != "" {
		root, err := filepath.Abs(explicit)
		if err != nil {
			return "", fmt.Errorf("resolve repository root: %w", err)
		}
		return root, nil
	}
	start, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	directory, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	for {
		if regularFile(filepath.Join(directory, filepath.FromSlash(defaultProvenancePath))) &&
			regularFile(filepath.Join(directory, "go.work")) {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("spanalyzer repository root not found from %q; pass --repo-root", start)
		}
		directory = parent
	}
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func readProvenance(path string) (*provenance, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read survey import provenance: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var document provenance
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode survey import provenance: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("decode survey import provenance: multiple JSON values")
		}
		return nil, fmt.Errorf("decode survey import provenance trailer: %w", err)
	}
	return &document, nil
}

func validateProvenance(document *provenance) error {
	if document.SchemaVersion != "v0alpha1" {
		return fmt.Errorf("provenance schema_version = %q, want v0alpha1", document.SchemaVersion)
	}
	if document.Source.Kind != "unpublished_local_git" || document.Source.Module != "github.com/apstndb/spanner-emulator-survey" {
		return errors.New("provenance source identity is invalid")
	}
	if !validObjectID(document.Source.Commit) || !validObjectID(document.Source.Tree) {
		return errors.New("provenance source commit and tree must be lowercase 40-hex object IDs")
	}
	if document.Source.CommitCount != 27 || document.Source.HistoryPublished {
		return errors.New("provenance must record 27 unpublished source commits")
	}
	if document.Destination.Repository != "github.com/apstndb/spanalyzer" || document.Destination.Path != "survey" {
		return errors.New("provenance destination identity is invalid")
	}
	if !validObjectID(document.Destination.ImportCommit) || !validObjectID(document.Destination.ImportTree) {
		return errors.New("provenance import commit and tree must be lowercase 40-hex object IDs")
	}
	if document.Source.Tree != document.Destination.ImportTree {
		return errors.New("source tree and initial destination tree differ")
	}
	if document.Dispositions.StandaloneRepositoryPublished ||
		document.Dispositions.UntrackedMemefishFeedback != "retired_after_upstream_issues_193_and_385_closed" ||
		document.Dispositions.LocalEnvironment != "excluded" {
		return errors.New("provenance dispositions are invalid")
	}
	if len(document.ExcludedPaths) == 0 {
		return errors.New("provenance excluded_paths is empty")
	}
	for _, path := range document.ExcludedPaths {
		if path == "" || filepath.IsAbs(path) || strings.Contains(path, "..") {
			return fmt.Errorf("invalid excluded path %q", path)
		}
	}
	return nil
}

func validObjectID(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 20 && value == strings.ToLower(value)
}

func verifyGitMapping(root string, document *provenance, runner commandRunner) error {
	if _, err := runner(root, "git", "cat-file", "-e", document.Destination.ImportCommit+"^{commit}"); err != nil {
		return fmt.Errorf("verify import commit: %w", err)
	}
	if _, err := runner(root, "git", "merge-base", "--is-ancestor", document.Destination.ImportCommit, "HEAD"); err != nil {
		return fmt.Errorf("import commit is not an ancestor of HEAD: %w", err)
	}
	treeOutput, err := runner(root, "git", "rev-parse", document.Destination.ImportCommit+":"+document.Destination.Path)
	if err != nil {
		return fmt.Errorf("resolve imported subtree: %w", err)
	}
	if tree := strings.TrimSpace(string(treeOutput)); tree != document.Destination.ImportTree {
		return fmt.Errorf("initial imported subtree = %q, provenance records %q", tree, document.Destination.ImportTree)
	}
	arguments := []string{"ls-files", "--"}
	for _, path := range document.ExcludedPaths {
		arguments = append(arguments, filepath.ToSlash(filepath.Join(document.Destination.Path, path)))
	}
	tracked, err := runner(root, "git", arguments...)
	if err != nil {
		return fmt.Errorf("inspect excluded import paths: %w", err)
	}
	if len(bytes.TrimSpace(tracked)) != 0 {
		return fmt.Errorf("excluded survey paths are tracked: %s", strings.TrimSpace(string(tracked)))
	}
	return nil
}

func runCommand(directory, name string, arguments ...string) ([]byte, error) {
	command := exec.Command(name, arguments...)
	command.Dir = directory
	output, err := command.Output()
	if err == nil {
		return output, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && len(exitError.Stderr) != 0 {
		return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(arguments, " "), err, strings.TrimSpace(string(exitError.Stderr)))
	}
	return nil, fmt.Errorf("%s %s: %w", name, strings.Join(arguments, " "), err)
}
