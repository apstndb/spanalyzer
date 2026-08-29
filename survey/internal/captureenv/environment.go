// Package captureenv opens managed or container-backed Spanner clients while
// retaining the exact, redacted target identity used by capture documents.
package captureenv

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/apstndb/spanalyzer/survey/internal/capturemeta"
	"github.com/apstndb/spanemuboost"
	"github.com/distribution/reference"
	"github.com/moby/moby/client"
	"github.com/testcontainers/testcontainers-go"
)

// Environment owns one Spanner client and its backing runtime, if any.
type Environment struct {
	Client *spanner.Client
	Target capturemeta.Target
	close  func() error
}

// Close releases the client and any backing runtime.
func (environment *Environment) Close() error {
	if environment == nil || environment.close == nil {
		return nil
	}
	return environment.close()
}

// Open creates one target environment. labelKey must be unique to the capture
// command so the running container can be inspected without selecting an
// unrelated concurrent probe.
func Open(ctx context.Context, kind, database, imageReference, labelKey string) (*Environment, error) {
	if kind == "managed" {
		client, err := spanner.NewClient(ctx, database)
		if err != nil {
			return nil, fmt.Errorf("open managed Spanner client: %w", err)
		}
		return &Environment{
			Client: client,
			Target: capturemeta.Target{Kind: "managed", ObservationScope: "single_database"},
			close:  func() error { client.Close(); return nil },
		}, nil
	}
	if kind != "omni" && kind != "emulator" {
		return nil, fmt.Errorf("capture target kind = %q, want managed, omni, or emulator", kind)
	}

	family, tag, pinnedDigest, err := SplitTaggedImage(imageReference)
	if err != nil {
		return nil, err
	}
	labelValue, err := randomLabelValue()
	if err != nil {
		return nil, err
	}
	backend := spanemuboost.BackendOmni
	if kind == "emulator" {
		backend = spanemuboost.BackendEmulator
	}
	runtime, err := spanemuboost.RunWithClients(
		ctx,
		backend,
		spanemuboost.WithContainerImage(imageReference),
		spanemuboost.WithContainerCustomizers(testcontainers.WithLabels(map[string]string{
			labelKey: labelValue,
		})),
	)
	if err != nil {
		return nil, fmt.Errorf("start %s image %q: %w", kind, imageReference, err)
	}
	digest, platform, err := inspectRuntimeImage(ctx, labelKey, labelValue)
	if err != nil {
		_ = runtime.Close()
		return nil, err
	}
	if pinnedDigest != "" && digest != pinnedDigest {
		_ = runtime.Close()
		return nil, fmt.Errorf("running container manifest digest = %q, pinned image reference requires %q", digest, pinnedDigest)
	}
	target := capturemeta.Target{Kind: kind, Image: &capturemeta.ImageIdentity{
		Family: family, Tag: tag, Digest: digest, Platform: platform,
	}}
	if err := capturemeta.ValidateTarget(target); err != nil {
		_ = runtime.Close()
		return nil, err
	}
	return &Environment{Client: runtime.Client, Target: target, close: runtime.Close}, nil
}

// SplitTaggedImage returns the normalized family, descriptive tag, and
// optional pinned digest from a container image reference.
func SplitTaggedImage(value string) (string, string, string, error) {
	named, err := reference.ParseNormalizedNamed(value)
	if err != nil {
		return "", "", "", fmt.Errorf("parse --image %q: %w", value, err)
	}
	tagged, ok := named.(reference.Tagged)
	if !ok {
		return "", "", "", errors.New("--image must include an explicit descriptive tag")
	}
	pinnedDigest := ""
	if digested, ok := named.(reference.Digested); ok {
		pinnedDigest = digested.Digest().String()
	}
	return named.Name(), tagged.Tag(), pinnedDigest, nil
}

func inspectRuntimeImage(ctx context.Context, label, value string) (string, string, error) {
	docker, err := client.New(client.FromEnv)
	if err != nil {
		return "", "", fmt.Errorf("open Docker client: %w", err)
	}
	defer func() { _ = docker.Close() }()
	containers, err := docker.ContainerList(ctx, client.ContainerListOptions{
		All: true, Filters: client.Filters{}.Add("label", label+"="+value),
	})
	if err != nil {
		return "", "", fmt.Errorf("list labeled capture containers: %w", err)
	}
	if len(containers.Items) != 1 {
		return "", "", fmt.Errorf("capture label selected %d containers, want exactly 1", len(containers.Items))
	}
	inspection, err := docker.ContainerInspect(ctx, containers.Items[0].ID, client.ContainerInspectOptions{})
	if err != nil {
		return "", "", fmt.Errorf("inspect capture container: %w", err)
	}
	descriptor := inspection.Container.ImageManifestDescriptor
	if descriptor == nil || descriptor.Digest.String() == "" || descriptor.Platform == nil {
		return "", "", errors.New("docker inspect did not return a platform-specific image manifest descriptor")
	}
	platform := descriptor.Platform.OS + "/" + descriptor.Platform.Architecture
	if descriptor.Platform.Variant != "" {
		platform += "/" + descriptor.Platform.Variant
	}
	return descriptor.Digest.String(), platform, nil
}

func randomLabelValue() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate capture container label: %w", err)
	}
	return hex.EncodeToString(value), nil
}

// ValidateExecutionEnvironment enforces the producer-hash execution contract.
func ValidateExecutionEnvironment() error {
	if strings.TrimSpace(strings.ToLower(os.Getenv("GOWORK"))) != "off" {
		return errors.New("capture execution requires GOWORK=off so producer hashes close over the survey module graph")
	}
	if slices.Contains(strings.Fields(os.Getenv("GOFLAGS")), "-mod=readonly") {
		return nil
	}
	return errors.New("capture execution requires GOFLAGS=-mod=readonly")
}
