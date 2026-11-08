package runtimepins

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
)

//go:generate go run ./generate

// pins.generated.json is copied from the canonical root registry by go generate.
// The repository gate compares the bytes, so updates cannot silently diverge.
//
//go:embed pins.generated.json
var bundledRegistry []byte

// ImageFromRegistry selects the default image from an explicit registry file,
// or from the binary's bundled registry when path is empty. It does not search
// the working directory or a config's ancestors, so installed binaries and
// external configs use the same immutable default.
func ImageFromRegistry(path, kind, platform string) (string, error) {
	if path == "" {
		return ImageFromReader(bytes.NewReader(bundledRegistry), kind, "", platform)
	}
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open runtime target registry: %w", err)
	}
	defer func() { _ = file.Close() }()
	return ImageFromReader(file, kind, "", platform)
}
