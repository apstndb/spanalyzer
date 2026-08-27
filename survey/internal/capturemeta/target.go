// Package capturemeta defines target identities shared by observational
// catalog captures. It contains no catalog-specific query or surface logic.
package capturemeta

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const managedScope = "single_database"

var (
	sha256Pattern   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	imageTagPattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`)
	platformPattern = regexp.MustCompile(`^linux/(amd64|arm64)(/v[0-9]+)?$`)
)

// Target identifies either a point-in-time managed observation or a
// retrospectively reproducible container release.
type Target struct {
	Kind             string         `json:"kind"`
	ObservationScope string         `json:"observation_scope,omitempty"`
	Image            *ImageIdentity `json:"image,omitempty"`
}

// ImageIdentity identifies the platform-specific OCI manifest actually run.
type ImageIdentity struct {
	Family   string `json:"family"`
	Tag      string `json:"tag"`
	Digest   string `json:"digest"`
	Platform string `json:"platform"`
}

// ValidateTarget enforces the redacted target identity contract.
func ValidateTarget(target Target) error {
	switch target.Kind {
	case "managed":
		if target.ObservationScope != managedScope || target.Image != nil {
			return errors.New("managed capture must set observation_scope=single_database and omit image")
		}
	case "omni", "emulator":
		if target.ObservationScope != "" || target.Image == nil {
			return fmt.Errorf("%s capture must set image and omit observation_scope", target.Kind)
		}
		image := target.Image
		if image.Family == "" || strings.ContainsAny(image.Family, "@\n\r\t ") {
			return fmt.Errorf("capture image family = %q, want a non-empty repository without digest or whitespace", image.Family)
		}
		if !imageTagPattern.MatchString(image.Tag) {
			return fmt.Errorf("capture image tag = %q, want a canonical OCI tag", image.Tag)
		}
		if !strings.HasPrefix(image.Digest, "sha256:") || !sha256Pattern.MatchString(strings.TrimPrefix(image.Digest, "sha256:")) {
			return fmt.Errorf("capture image digest = %q, want sha256:<64 lowercase hex>", image.Digest)
		}
		if !platformPattern.MatchString(image.Platform) {
			return fmt.Errorf("capture image platform = %q, want normalized linux/amd64 or linux/arm64[/variant]", image.Platform)
		}
	default:
		return fmt.Errorf("capture target kind = %q, want managed, omni, or emulator", target.Kind)
	}
	return nil
}

// ExpectedPath returns the canonical evidence path for one target and surface.
func ExpectedPath(target Target, observedAt, surfaceSHA256 string) (string, error) {
	if err := ValidateTarget(target); err != nil {
		return "", err
	}
	if !sha256Pattern.MatchString(surfaceSHA256) {
		return "", fmt.Errorf("surface SHA-256 = %q, want 64 lowercase hexadecimal characters", surfaceSHA256)
	}
	shortHash := surfaceSHA256[:12]
	switch target.Kind {
	case "managed":
		observationTime, err := time.Parse(time.RFC3339Nano, observedAt)
		if err != nil {
			return "", fmt.Errorf("parse managed observed_at: %w", err)
		}
		stamp := observationTime.UTC().Format("20060102T150405Z")
		return filepath.ToSlash(filepath.Join("evidence", "managed", stamp+"-"+shortHash+".json")), nil
	case "omni", "emulator":
		digest := strings.TrimPrefix(target.Image.Digest, "sha256:")
		platform := strings.ReplaceAll(target.Image.Platform, "/", "-")
		return filepath.ToSlash(filepath.Join("evidence", target.Kind, digest, platform+"-"+shortHash+".json")), nil
	default:
		panic("ValidateTarget accepted an unknown target")
	}
}
