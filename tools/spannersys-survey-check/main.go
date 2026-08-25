// Command spannersys-survey-check validates the committed SPANNER_SYS
// manifest and optionally compares its exact bytes with the exporter at the
// pinned spanner-emulator-survey commit.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const defaultManifestPath = "spanner_sys_manifest.json"

type commandRunner func(directory, name string, arguments ...string) ([]byte, error)

type validatedManifest struct {
	Bytes        []byte
	SourceCommit string
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
	flags := flag.NewFlagSet("spannersys-survey-check", flag.ContinueOnError)
	repoRoot := flags.String("repo-root", "", "spanalyzer repository root (auto-detected when omitted)")
	manifestPath := flags.String("manifest", defaultManifestPath, "manifest path relative to the spanalyzer repository root")
	surveyRoot := flags.String("survey-root", "", "optional clean spanner-emulator-survey checkout to compare")
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
	manifest, err := readAndValidateManifest(root, filepath.Join(root, filepath.FromSlash(*manifestPath)), runner)
	if err != nil {
		return err
	}
	if *surveyRoot == "" {
		return nil
	}
	return compareSurveyCheckout(manifest.Bytes, manifest.SourceCommit, *surveyRoot, runner)
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
	return findRepoRoot(start)
}

func findRepoRoot(start string) (string, error) {
	directory, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	for {
		if regularFile(filepath.Join(directory, defaultManifestPath)) &&
			regularFile(filepath.Join(directory, "tools", "spannersys-survey-check", "main.go")) {
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

func readAndValidateManifest(root, path string, runner commandRunner) (*validatedManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read SPANNER_SYS manifest: %w", err)
	}
	commitOutput, err := runner(
		root,
		"go",
		"-C",
		root,
		"run",
		"./internal/spannersysmanifest/checkcmd",
		"--manifest",
		path,
	)
	if err != nil {
		return nil, fmt.Errorf("validate SPANNER_SYS manifest: %w", err)
	}
	if len(commitOutput) != 41 || commitOutput[40] != '\n' {
		return nil, fmt.Errorf("SPANNER_SYS manifest validator output = %q, want one lowercase 40-hex source commit followed by a newline", commitOutput)
	}
	commit := string(commitOutput[:40])
	decodedCommit, err := hex.DecodeString(commit)
	if err != nil || len(decodedCommit) != 20 || commit != strings.ToLower(commit) {
		return nil, fmt.Errorf("SPANNER_SYS manifest validator output = %q, want one lowercase 40-hex source commit followed by a newline", commitOutput)
	}
	validatedData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("re-read validated SPANNER_SYS manifest: %w", err)
	}
	if !bytes.Equal(validatedData, data) {
		return nil, errors.New("SPANNER_SYS manifest changed while it was being validated")
	}
	return &validatedManifest{Bytes: data, SourceCommit: commit}, nil
}

func compareSurveyCheckout(
	manifestBytes []byte,
	sourceCommit string,
	surveyRoot string,
	runner commandRunner,
) error {
	root, err := filepath.Abs(surveyRoot)
	if err != nil {
		return fmt.Errorf("resolve survey root: %w", err)
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		if err == nil {
			err = errors.New("not a directory")
		}
		return fmt.Errorf("invalid --survey-root %q: %w", root, err)
	}

	status, err := runner(root, "git", "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return fmt.Errorf("inspect survey worktree status in %q: %w", root, err)
	}
	if len(status) != 0 {
		return fmt.Errorf("survey checkout %q is dirty; exact producer comparison requires a clean worktree: %s", root, strings.TrimSpace(string(status)))
	}

	commitOutput, err := runner(root, "git", "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("resolve survey commit in %q: %w", root, err)
	}
	commit := strings.TrimSpace(string(commitOutput))
	if commit != sourceCommit {
		return fmt.Errorf("survey checkout commit = %q, manifest pins %q", commit, sourceCommit)
	}

	exported, err := runner(
		root,
		"go",
		"run",
		"./cmd/spanner-sys-export",
		"--source-commit",
		sourceCommit,
	)
	if err != nil {
		return fmt.Errorf("export SPANNER_SYS manifest from survey checkout %q: %w", root, err)
	}
	if !bytes.Equal(exported, manifestBytes) {
		return fmt.Errorf(
			"survey export differs from committed manifest: export SHA-256 %s (%d bytes), committed SHA-256 %s (%d bytes)",
			sha256Hex(exported),
			len(exported),
			sha256Hex(manifestBytes),
			len(manifestBytes),
		)
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

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
