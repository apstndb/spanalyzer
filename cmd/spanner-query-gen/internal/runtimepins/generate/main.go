// Command generate copies the repository's canonical runtime pin registry.
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	data, err := os.ReadFile("../../../../runtime_targets.json")
	if err != nil {
		return err
	}
	return os.WriteFile("pins.generated.json", data, 0o644)
}
