// Command checkcmd validates one SPANNER_SYS manifest and prints its pinned
// source commit. It is internal glue for the tools module, not a public CLI.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/apstndb/spanalyzer/internal/spannersysmanifest"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("spannersys-manifest-check", flag.ContinueOnError)
	manifestPath := flags.String("manifest", "spanner_sys_manifest.json", "SPANNER_SYS manifest to validate")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	data, err := os.ReadFile(*manifestPath)
	if err != nil {
		return fmt.Errorf("read SPANNER_SYS manifest: %w", err)
	}
	document, err := spannersysmanifest.Decode(data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, document.Source.Commit)
	return err
}
