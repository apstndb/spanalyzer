// Command infoschema-capture records a redacted INFORMATION_SCHEMA surface
// observation from managed Spanner, Spanner Omni, or the Spanner Emulator.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/apstndb/spanalyzer/survey/internal/captureenv"
	"github.com/apstndb/spanalyzer/survey/internal/runtimepins"
)

const captureLabel = "io.github.apstndb.spanalyzer.infoschema-capture"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	checkMode := len(args) > 0 && args[0] == "check"
	if checkMode {
		args = args[1:]
	}
	flags := flag.NewFlagSet("infoschema-capture", flag.ContinueOnError)
	flags.SetOutput(stderr)
	target := flags.String("target", "", "capture target: managed, omni, or emulator")
	database := flags.String("database", os.Getenv("TEST_REAL_SPANNER_DATABASE"), "managed Spanner database resource (defaults to TEST_REAL_SPANNER_DATABASE; never retained)")
	image := flags.String("image", "", "container image override with an explicit descriptive tag and optional @sha256 digest pin; defaults to runtime_targets.json")
	repoRoot := flags.String("repo-root", "", "spanalyzer repository root (auto-detected when omitted)")
	output := flags.String("output", "", "canonical repository output path (stdout when omitted)")
	write := flags.Bool("write", false, "write to the canonical repository path")
	timeout := flags.Duration("timeout", 15*time.Minute, "overall capture timeout")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		writeDiagnostic(stderr, "unexpected positional arguments: %v\n", flags.Args())
		return 2
	}
	if *write && *output != "" {
		writeDiagnostic(stderr, "--write and --output are mutually exclusive\n")
		return 2
	}
	if checkMode && (*write || *output != "") {
		writeDiagnostic(stderr, "check does not accept --write or --output\n")
		return 2
	}
	if *target != "managed" && *target != "omni" && *target != "emulator" {
		writeDiagnostic(stderr, "--target must be managed, omni, or emulator\n")
		return 2
	}
	if *target == "managed" && *database == "" {
		writeDiagnostic(stderr, "--database or TEST_REAL_SPANNER_DATABASE is required for target managed\n")
		return 2
	}
	if err := validateExecutionEnvironment(); err != nil {
		writeDiagnostic(stderr, "%v\n", err)
		return 2
	}

	root, err := resolveRepoRoot(*repoRoot)
	if err != nil {
		writeDiagnostic(stderr, "%v\n", err)
		return 1
	}
	if *target != "managed" && *image == "" {
		*image, err = runtimepins.ImageForHost(root, *target)
		if err != nil {
			writeDiagnostic(stderr, "resolve pinned %s image: %v\n", *target, err)
			return 1
		}
	}
	producer, err := infoschem.ComputeProducerIdentity(root)
	if err != nil {
		writeDiagnostic(stderr, "compute producer identity: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	document, err := capture(ctx, *target, *database, *image, producer)
	if err != nil {
		writeDiagnostic(stderr, "capture INFORMATION_SCHEMA: %v\n", err)
		return 1
	}
	data, err := infoschem.EncodeCapture(document)
	if err != nil {
		writeDiagnostic(stderr, "encode INFORMATION_SCHEMA capture: %v\n", err)
		return 1
	}
	relative, err := infoschem.ExpectedCapturePath(document)
	if err != nil {
		writeDiagnostic(stderr, "derive INFORMATION_SCHEMA capture path: %v\n", err)
		return 1
	}
	canonicalOutput := filepath.Join(root, "survey", "infoschem", filepath.FromSlash(relative))
	if checkMode {
		report, err := compareRetainedCapture(root, document)
		if err != nil {
			writeDiagnostic(stderr, "compare retained INFORMATION_SCHEMA capture: %v\n", err)
			return 1
		}
		if err := writeComparisonReport(stdout, report); err != nil {
			writeDiagnostic(stderr, "write INFORMATION_SCHEMA comparison: %v\n", err)
			return 1
		}
		if report.MaterialChange {
			return 1
		}
		return 0
	}
	if !*write && *output == "" {
		if _, err := stdout.Write(data); err != nil {
			writeDiagnostic(stderr, "write capture: %v\n", err)
			return 1
		}
		writeDiagnostic(stderr, "canonical_path=%s\n", filepath.ToSlash(filepath.Join("survey", "infoschem", relative)))
		return 0
	}
	outputPath := canonicalOutput
	if *output != "" {
		outputPath, err = filepath.Abs(*output)
		if err != nil {
			writeDiagnostic(stderr, "resolve --output: %v\n", err)
			return 1
		}
	}
	wantPath, err := filepath.Abs(canonicalOutput)
	if err != nil {
		writeDiagnostic(stderr, "resolve canonical output: %v\n", err)
		return 1
	}
	if outputPath != wantPath {
		writeDiagnostic(stderr, "--output = %q, want canonical path %q\n", outputPath, wantPath)
		return 2
	}
	wrote, err := writeCapture(outputPath, data, document)
	if err != nil {
		writeDiagnostic(stderr, "write capture: %v\n", err)
		return 1
	}
	if !wrote {
		writeDiagnostic(stderr, "already retained equivalent observation %s\n", filepath.ToSlash(filepath.Join("survey", "infoschem", relative)))
		return 0
	}
	writeDiagnostic(stderr, "wrote %s\n", filepath.ToSlash(filepath.Join("survey", "infoschem", relative)))
	return 0
}

func capture(
	ctx context.Context,
	targetKind string,
	database string,
	imageReference string,
	producer infoschem.ProducerIdentity,
) (*infoschem.CaptureDocument, error) {
	environment, err := captureenv.Open(ctx, targetKind, database, imageReference, captureLabel)
	if err != nil {
		return nil, err
	}
	defer func() { _ = environment.Close() }()
	txn := newCaptureTransaction(environment.Client)
	defer txn.Close()
	return infoschem.CaptureFromTransaction(ctx, txn, environment.Target, producer)
}

func newCaptureTransaction(client *spanner.Client) *spanner.ReadOnlyTransaction {
	return client.ReadOnlyTransaction()
}

func splitTaggedImage(value string) (string, string, string, error) {
	return captureenv.SplitTaggedImage(value)
}

func validateExecutionEnvironment() error {
	return captureenv.ValidateExecutionEnvironment()
}

func resolveRepoRoot(explicit string) (string, error) {
	if explicit != "" {
		root, err := filepath.Abs(explicit)
		if err != nil {
			return "", fmt.Errorf("resolve repository root: %w", err)
		}
		return validateRepoRoot(root)
	}
	start, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	for directory := start; ; directory = filepath.Dir(directory) {
		if root, err := validateRepoRoot(directory); err == nil {
			return root, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("spanalyzer repository root not found from %q; pass --repo-root", start)
		}
	}
}

func validateRepoRoot(root string) (string, error) {
	for _, path := range []string{
		"go.work",
		filepath.Join("survey", "go.mod"),
		filepath.Join("survey", "infoschem", "capture-definition.v0alpha1.json"),
	} {
		info, err := os.Stat(filepath.Join(root, path))
		if err != nil || !info.Mode().IsRegular() {
			if err == nil {
				err = errors.New("not a regular file")
			}
			return "", fmt.Errorf("invalid repository root %q: %s: %w", root, path, err)
		}
	}
	return root, nil
}

func createAtomically(path string, data []byte) (bool, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".infoschema-capture-*.tmp")
	if err != nil {
		return false, err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return false, err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return false, err
	}
	if err := temporary.Close(); err != nil {
		return false, err
	}
	if err := os.Link(temporaryPath, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func writeCapture(path string, data []byte, document *infoschem.CaptureDocument) (bool, error) {
	wrote, err := createAtomically(path, data)
	if err != nil {
		return false, err
	}
	if wrote {
		return true, nil
	}
	existing, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if bytes.Equal(existing, data) {
		return false, nil
	}
	previous, err := infoschem.DecodeCapture(existing)
	if err != nil {
		return false, fmt.Errorf("decode existing canonical capture: %w", err)
	}
	if sameCanonicalCaptureIdentity(previous, document) {
		if previous.ProducerSourceSHA256 != document.ProducerSourceSHA256 || previous.InvocationSHA256 != document.InvocationSHA256 {
			return false, fmt.Errorf(
				"canonical capture path %q already exists with producer identity source=%s invocation=%s; current capture uses source=%s invocation=%s; review or remove the retained file before recapturing",
				filepath.ToSlash(path),
				previous.ProducerSourceSHA256,
				previous.InvocationSHA256,
				document.ProducerSourceSHA256,
				document.InvocationSHA256,
			)
		}
		// Preserve the first observation assigned to a canonical path. Container
		// paths key on runtime identity and surface; managed paths key on the
		// observation second and surface while the JSON retains the exact time.
		return false, nil
	}
	return false, errors.New("canonical capture path already exists with conflicting identity or surface")
}

func sameCanonicalCaptureIdentity(left, right *infoschem.CaptureDocument) bool {
	leftPath, leftErr := infoschem.ExpectedCapturePath(left)
	rightPath, rightErr := infoschem.ExpectedCapturePath(right)
	return leftErr == nil && rightErr == nil && leftPath == rightPath
}

func writeDiagnostic(writer io.Writer, format string, arguments ...any) {
	// Diagnostics are best-effort because run has already selected the exit
	// code and there is no second channel on which to report a stderr failure.
	_, _ = fmt.Fprintf(writer, format, arguments...)
}
