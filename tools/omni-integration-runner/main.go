// Command omni-integration-runner starts one pinned Spanner Omni runtime and
// runs both live integration packages against that shared endpoint.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/apstndb/spanalyzer/tools/internal/runtimepins"
	"github.com/apstndb/spanemuboost"
)

type report struct {
	SchemaVersion   string       `json:"schema_version"`
	Status          string       `json:"status"`
	SourceHead      string       `json:"source_head"`
	SourceDirty     bool         `json:"source_dirty"`
	Image           string       `json:"image"`
	RuntimeStartMS  int64        `json:"runtime_start_ms"`
	TotalDurationMS int64        `json:"total_duration_ms"`
	Steps           []stepReport `json:"steps"`
}

type stepReport struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	DurationMS int64  `json:"duration_ms"`
}

type step struct {
	name    string
	workdir string
	args    []string
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("omni-integration-runner", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repoRootFlag := flags.String("repo-root", "", "spanalyzer repository root (auto-detected when omitted)")
	timeout := flags.Duration("timeout", 35*time.Minute, "overall runtime and integration timeout")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	repoRoot, err := resolveRepoRoot(*repoRootFlag)
	if err != nil {
		return err
	}
	image, err := runtimepins.ImageForHost(repoRoot, "omni")
	if err != nil {
		return fmt.Errorf("resolve pinned Spanner Omni image: %w", err)
	}
	head, dirty, err := gitSourceState(ctx, repoRoot)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	totalStart := time.Now()
	runtimeStart := time.Now()
	runtime, err := spanemuboost.Run(
		ctx,
		spanemuboost.BackendOmni,
		spanemuboost.WithContainerImage(image),
		spanemuboost.DisableAutoConfig(),
	)
	if err != nil {
		return fmt.Errorf("start pinned Spanner Omni runtime: %w", err)
	}
	runtimeStartDuration := time.Since(runtimeStart)
	closed := false
	defer func() {
		if !closed {
			_ = runtime.Close()
		}
	}()

	temporary, err := os.MkdirTemp("", "spanalyzer-omni-integration-")
	if err != nil {
		return fmt.Errorf("create endpoint directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(temporary) }()
	endpointPath := filepath.Join(temporary, "endpoint.json")
	endpoint, err := spanemuboost.EndpointFromRuntime(runtime)
	if err != nil {
		return fmt.Errorf("describe shared Spanner Omni runtime: %w", err)
	}
	endpoint.ManagedBy = "spanalyzer omni-integration-runner"
	endpoint.StartedAt = totalStart.UTC().Format(time.RFC3339)
	if err := spanemuboost.SaveEndpoint(endpointPath, endpoint); err != nil {
		return fmt.Errorf("write shared Spanner Omni endpoint: %w", err)
	}

	result := report{
		SchemaVersion:  "v0alpha1",
		Status:         "ok",
		SourceHead:     head,
		SourceDirty:    dirty,
		Image:          image,
		RuntimeStartMS: runtimeStartDuration.Milliseconds(),
	}
	steps := integrationSteps(repoRoot)
	for _, current := range steps {
		started := time.Now()
		err := runStep(ctx, current, endpointPath, image, stderr)
		entry := stepReport{Name: current.name, Status: "ok", DurationMS: time.Since(started).Milliseconds()}
		if err != nil {
			entry.Status = "failed"
			result.Status = "failed"
			result.Steps = append(result.Steps, entry)
			result.TotalDurationMS = time.Since(totalStart).Milliseconds()
			if encodeErr := writeReport(stdout, result); encodeErr != nil {
				return errors.Join(err, encodeErr)
			}
			return err
		}
		result.Steps = append(result.Steps, entry)
	}
	closeErr := runtime.Close()
	closed = true
	result.TotalDurationMS = time.Since(totalStart).Milliseconds()
	if closeErr != nil {
		result.Status = "failed"
		closeErr = fmt.Errorf("close shared Spanner Omni runtime: %w", closeErr)
	}
	if reportErr := writeReport(stdout, result); reportErr != nil {
		return errors.Join(closeErr, reportErr)
	}
	return closeErr
}

func integrationSteps(repoRoot string) []step {
	return []step{
		{
			name:    "plan-shape",
			workdir: filepath.Join(repoRoot, "tools"),
			args:    []string{"test", "-count=1", "-timeout=30m", "-tags=integration,omni", "-run", "^TestIntegration.*OnOmni$", "./spanner-query-plan-shape"},
		},
		{
			name:    "query-generator",
			workdir: filepath.Join(repoRoot, "cmd", "spanner-query-gen"),
			args:    []string{"test", "-count=1", "-timeout=30m", "-tags=integration,omni", "-run", "^TestIntegration.*OnOmni$", "./..."},
		},
	}
}

func runStep(ctx context.Context, current step, endpointPath, image string, stderr io.Writer) error {
	command := exec.CommandContext(ctx, "go", current.args...)
	command.Dir = current.workdir
	command.Stdout = stderr
	command.Stderr = stderr
	command.Env = append(os.Environ(),
		"SPANEMUBOOST_ENDPOINT_FILE="+endpointPath,
		"SPANEMUBOOST_ENABLE_OMNI_TESTS=1",
		"SPANALYZER_OMNI_IMAGE="+image,
	)
	if err := command.Run(); err != nil {
		return fmt.Errorf("run %s integration: %w", current.name, err)
	}
	return nil
}

func resolveRepoRoot(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		root, err := filepath.Abs(explicit)
		if err != nil {
			return "", fmt.Errorf("resolve repository root: %w", err)
		}
		return root, nil
	}
	return runtimepins.FindRepositoryRoot(".")
}

func gitSourceState(ctx context.Context, repoRoot string) (string, bool, error) {
	command := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	command.Dir = repoRoot
	data, err := command.Output()
	if err != nil {
		return "", false, fmt.Errorf("resolve source HEAD: %w", err)
	}
	status := exec.CommandContext(ctx, "git", "status", "--porcelain", "--untracked-files=normal")
	status.Dir = repoRoot
	statusData, err := status.Output()
	if err != nil {
		return "", false, fmt.Errorf("resolve source worktree status: %w", err)
	}
	return strings.TrimSpace(string(data)), len(bytes.TrimSpace(statusData)) > 0, nil
}

func writeReport(writer io.Writer, result report) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("write integration report: %w", err)
	}
	return nil
}
