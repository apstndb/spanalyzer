// Command spanner-sys-export writes the deterministic SPANNER_SYS manifest
// for an explicitly identified spanner-emulator-survey commit.
package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/apstndb/spanner-emulator-survey/spannersys"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("spanner-sys-export", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sourceCommit := flags.String(
		"source-commit",
		"",
		"exact 40-hex spanner-emulator-survey commit containing this exporter",
	)
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
	if *sourceCommit == "" {
		writeDiagnostic(stderr, "--source-commit is required\n")
		return 2
	}
	if !validSourceCommit(*sourceCommit) {
		writeDiagnostic(stderr, "--source-commit must contain 40 lowercase hexadecimal characters\n")
		return 2
	}

	manifest, err := spannersys.ExportManifest(*sourceCommit)
	if err != nil {
		writeDiagnostic(stderr, "export SPANNER_SYS manifest: %v\n", err)
		return 1
	}
	if _, err := stdout.Write(manifest); err != nil {
		writeDiagnostic(stderr, "write SPANNER_SYS manifest: %v\n", err)
		return 1
	}
	return 0
}

func validSourceCommit(commit string) bool {
	if len(commit) != 40 {
		return false
	}
	_, err := hex.DecodeString(commit)
	return err == nil && commit == strings.ToLower(commit)
}

func writeDiagnostic(writer io.Writer, format string, arguments ...any) {
	// Diagnostics are best-effort because run has already selected the exit
	// code and there is no second channel on which to report a stderr failure.
	_, _ = fmt.Fprintf(writer, format, arguments...)
}
