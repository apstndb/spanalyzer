// Package runtimepins reads the repository-owned container runtime contract.
// The loader is module-local because each nested module must survive its own
// dependency resolution; runtime_targets.json remains the single data source.
package runtimepins

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const registryFile = "runtime_targets.json"

var (
	digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	tagPattern    = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`)
)

type registry struct {
	SchemaVersion string   `json:"schema_version"`
	Targets       []target `json:"targets"`
}

type target struct {
	Kind         string     `json:"kind"`
	ImageFamily  string     `json:"image_family"`
	Tag          string     `json:"tag"`
	Platforms    []platform `json:"platforms"`
	Alternatives []version  `json:"alternatives,omitempty"`
}

type version struct {
	Tag       string     `json:"tag"`
	Platforms []platform `json:"platforms"`
}

type platform struct {
	Platform string `json:"platform"`
	Digest   string `json:"digest"`
}

// FindRepositoryRoot locates runtime_targets.json by walking upward from start.
func FindRepositoryRoot(start string) (string, error) {
	absolute, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve start directory: %w", err)
	}
	for directory := absolute; ; directory = filepath.Dir(directory) {
		info, err := os.Stat(filepath.Join(directory, registryFile))
		if err == nil && info.Mode().IsRegular() {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("%s not found from %q", registryFile, absolute)
		}
	}
}

// ImageForHost returns the exact platform manifest reference for kind.
func ImageForHost(repoRoot, kind string) (string, error) {
	return ImageForPlatform(repoRoot, kind, "linux/"+runtime.GOARCH)
}

// ImageForHostVersion selects a default or alternative version for the host.
// An empty tag selects the default; unregistered versions fail closed.
func ImageForHostVersion(repoRoot, kind, tag string) (string, error) {
	return ImageForVersion(repoRoot, kind, tag, "linux/"+runtime.GOARCH)
}

// ImageForPlatform returns image_family:tag@digest for one pinned target.
func ImageForPlatform(repoRoot, kind, wantedPlatform string) (string, error) {
	return ImageForVersion(repoRoot, kind, "", wantedPlatform)
}

// ImageForVersion returns a platform manifest from the same registry as the
// default pin. Explicit selection never changes the default for other callers.
func ImageForVersion(repoRoot, kind, tag, wantedPlatform string) (string, error) {
	file, err := os.Open(filepath.Join(repoRoot, registryFile))
	if err != nil {
		return "", fmt.Errorf("open runtime target registry: %w", err)
	}
	defer func() { _ = file.Close() }()
	return ImageFromReader(file, kind, tag, wantedPlatform)
}

// ImageFromReader validates a registry and selects one exact platform image.
func ImageFromReader(reader io.Reader, kind, tag, wantedPlatform string) (string, error) {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var document struct {
		Schema string `json:"$schema"`
		registry
	}
	if err := decoder.Decode(&document); err != nil {
		return "", fmt.Errorf("decode runtime target registry: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return "", err
	}
	if document.Schema != "schemas/runtime-targets.v0alpha1.schema.json" || document.SchemaVersion != "v0alpha1" {
		return "", errors.New("runtime target registry has an unsupported schema identity")
	}
	if len(document.Targets) != 2 {
		return "", fmt.Errorf("runtime target count = %d, want 2", len(document.Targets))
	}
	images := make(map[string]map[string]map[string]string, len(document.Targets))
	defaults := make(map[string]string, len(document.Targets))
	for _, candidate := range document.Targets {
		if _, duplicate := images[candidate.Kind]; duplicate {
			return "", fmt.Errorf("duplicate runtime target kind %q", candidate.Kind)
		}
		if candidate.Kind != "emulator" && candidate.Kind != "omni" {
			return "", fmt.Errorf("unsupported runtime target kind %q", candidate.Kind)
		}
		if candidate.ImageFamily == "" || strings.ContainsAny(candidate.ImageFamily, "@\n\r\t ") || !tagPattern.MatchString(candidate.Tag) {
			return "", fmt.Errorf("runtime target %q has an invalid image family or tag", candidate.Kind)
		}
		defaults[candidate.Kind] = candidate.Tag
		images[candidate.Kind] = map[string]map[string]string{}
		versions := append([]version{{Tag: candidate.Tag, Platforms: candidate.Platforms}}, candidate.Alternatives...)
		for _, current := range versions {
			if !tagPattern.MatchString(current.Tag) {
				return "", fmt.Errorf("runtime target %q has invalid version tag %q", candidate.Kind, current.Tag)
			}
			if _, duplicate := images[candidate.Kind][current.Tag]; duplicate {
				return "", fmt.Errorf("runtime target %q has duplicate version %q", candidate.Kind, current.Tag)
			}
			if len(current.Platforms) != 2 {
				return "", fmt.Errorf("runtime target %q version %q platform count = %d, want 2", candidate.Kind, current.Tag, len(current.Platforms))
			}
			platforms := make(map[string]string, len(current.Platforms))
			for _, pinned := range current.Platforms {
				if _, duplicate := platforms[pinned.Platform]; duplicate {
					return "", fmt.Errorf("runtime target %q has duplicate platform %q", candidate.Kind, pinned.Platform)
				}
				if pinned.Platform != "linux/amd64" && pinned.Platform != "linux/arm64" {
					return "", fmt.Errorf("runtime target %q has unsupported platform %q", candidate.Kind, pinned.Platform)
				}
				if !digestPattern.MatchString(pinned.Digest) {
					return "", fmt.Errorf("runtime target %q platform %q has invalid digest %q", candidate.Kind, pinned.Platform, pinned.Digest)
				}
				platforms[pinned.Platform] = candidate.ImageFamily + ":" + current.Tag + "@" + pinned.Digest
			}
			images[candidate.Kind][current.Tag] = platforms
		}
	}
	versions, ok := images[kind]
	if !ok {
		return "", fmt.Errorf("runtime target kind %q is not pinned", kind)
	}
	if tag == "" {
		tag = defaults[kind]
	}
	platforms, ok := versions[tag]
	if !ok {
		return "", fmt.Errorf("runtime target %q version %q is not pinned", kind, tag)
	}
	image, ok := platforms[wantedPlatform]
	if !ok {
		return "", fmt.Errorf("runtime target %q has no platform %q", kind, wantedPlatform)
	}
	return image, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("decode runtime target registry trailing data: %w", err)
	}
	return errors.New("runtime target registry has trailing JSON values")
}
