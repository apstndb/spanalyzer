package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/apstndb/spanalyzer/cmd/spanner-query-gen/internal/runtimepins"
	"github.com/apstndb/spanemuboost"
)

type planReportRuntimeSelection struct {
	Runtime  spanemuboost.RuntimeHandle
	Close    bool
	Identity planReportBackendIdentity
	Image    string
}

type planReportRuntimeDeps struct {
	InspectOmniEndpoint func() (attached bool, err error)
	HostGOARCH          string
	FindRepositoryRoot  func(start string) (string, error)
	ImageForPlatform    func(repoRoot, kind, platform string) (string, error)
	NewAttachedRuntime  func() (spanemuboost.RuntimeHandle, error)
	NewColdStartRuntime func(image string) (spanemuboost.RuntimeHandle, error)
}

func defaultPlanReportRuntimeDeps() planReportRuntimeDeps {
	return planReportRuntimeDeps{
		InspectOmniEndpoint: inspectOmniEndpointAttached,
		HostGOARCH:          runtime.GOARCH,
		FindRepositoryRoot:  runtimepins.FindRepositoryRoot,
		ImageForPlatform:    runtimepins.ImageForPlatform,
		NewAttachedRuntime: func() (spanemuboost.RuntimeHandle, error) {
			return spanemuboost.NewLazyRuntimeFromEnvOrStart(spanemuboost.BackendOmni)
		},
		NewColdStartRuntime: func(image string) (spanemuboost.RuntimeHandle, error) {
			return spanemuboost.NewLazyRuntime(spanemuboost.BackendOmni, spanemuboost.WithContainerImage(image)), nil
		},
	}
}

func inspectOmniEndpointAttached() (bool, error) {
	if path := strings.TrimSpace(os.Getenv("SPANEMUBOOST_ENDPOINT_FILE")); path != "" {
		endpoint, err := spanemuboost.ReadEndpointFile(path)
		if err != nil {
			return false, fmt.Errorf("load attached Omni endpoint: %w", err)
		}
		return endpoint.Backend == spanemuboost.BackendOmni, nil
	}
	if strings.TrimSpace(os.Getenv("SPANEMUBOOST_OMNI_URI")) == "" {
		return false, nil
	}
	if _, err := spanemuboost.LoadEndpointForBackend(spanemuboost.BackendOmni); err != nil {
		return false, fmt.Errorf("load attached Omni endpoint: %w", err)
	}
	return true, nil
}

func selectPlanReportRuntime(opts planReportOptions, startDir string, deps planReportRuntimeDeps) (planReportRuntimeSelection, error) {
	backend := emptyDefault(opts.Backend, "omni")
	if deps.InspectOmniEndpoint != nil {
		attached, err := deps.InspectOmniEndpoint()
		if err != nil {
			return planReportRuntimeSelection{}, err
		}
		if attached {
			if deps.NewAttachedRuntime == nil {
				return planReportRuntimeSelection{}, fmt.Errorf("attached Omni runtime constructor is required")
			}
			runtimeHandle, err := deps.NewAttachedRuntime()
			if err != nil {
				return planReportRuntimeSelection{}, err
			}
			return planReportRuntimeSelection{
				Runtime:  runtimeHandle,
				Close:    true,
				Identity: attachedPlanReportIdentity(backend, opts.Identity),
			}, nil
		}
	}
	arch := strings.TrimSpace(deps.HostGOARCH)
	if arch == "" {
		arch = runtime.GOARCH
	}
	platform := "linux/" + arch
	if deps.FindRepositoryRoot == nil || deps.ImageForPlatform == nil {
		return planReportRuntimeSelection{}, fmt.Errorf("resolve Omni runtime pin for %s: pin resolver is required", platform)
	}
	if startDir == "" {
		startDir = "."
	}
	root, err := deps.FindRepositoryRoot(startDir)
	if err != nil {
		return planReportRuntimeSelection{}, fmt.Errorf("resolve Omni runtime pin for %s: %w", platform, err)
	}
	image, err := deps.ImageForPlatform(root, "omni", platform)
	if err != nil {
		return planReportRuntimeSelection{}, fmt.Errorf("resolve Omni runtime pin for %s: %w", platform, err)
	}
	tag, digest, err := parsePinnedContainerImage(image)
	if err != nil {
		return planReportRuntimeSelection{}, fmt.Errorf("parse Omni runtime pin for %s: %w", platform, err)
	}
	if err := rejectConflictingColdStartAssertion(opts.Identity, tag, digest); err != nil {
		return planReportRuntimeSelection{}, err
	}
	if deps.NewColdStartRuntime == nil {
		return planReportRuntimeSelection{}, fmt.Errorf("cold-start Omni runtime constructor is required")
	}
	runtimeHandle, err := deps.NewColdStartRuntime(image)
	if err != nil {
		return planReportRuntimeSelection{}, err
	}
	return planReportRuntimeSelection{
		Runtime: runtimeHandle,
		Close:   true,
		Image:   image,
		Identity: planReportBackendIdentity{
			Kind:        backend,
			Version:     tag,
			ImageDigest: digest,
			Source:      planReportIdentitySourceRuntimeTargets,
		},
	}, nil
}

func attachedPlanReportIdentity(backend string, assertion planReportBackendIdentity) planReportBackendIdentity {
	if assertion.Source == planReportIdentitySourceManual {
		assertion.Kind = emptyDefault(assertion.Kind, backend)
		return assertion
	}
	return planReportBackendIdentity{
		Kind:        backend,
		Version:     "not_recorded",
		ImageDigest: "not_recorded",
		Source:      planReportIdentitySourceExternalUnverified,
	}
}

func rejectConflictingColdStartAssertion(assertion planReportBackendIdentity, tag, digest string) error {
	if assertion.Source != planReportIdentitySourceManual {
		return nil
	}
	if assertion.Version != "not_recorded" && assertion.Version != tag {
		return fmt.Errorf("--backend-version %q does not match configured Omni pin tag %q", assertion.Version, tag)
	}
	if assertion.ImageDigest != "not_recorded" && assertion.ImageDigest != digest {
		return fmt.Errorf("--backend-image-digest %q does not match configured Omni pin digest %q", assertion.ImageDigest, digest)
	}
	return nil
}

func parsePinnedContainerImage(image string) (tag, digest string, err error) {
	image = strings.TrimSpace(image)
	at := strings.LastIndex(image, "@")
	if at <= 0 || at == len(image)-1 {
		return "", "", fmt.Errorf("pinned image %q is not image:tag@sha256:<digest>", image)
	}
	nameTag := image[:at]
	digest = image[at+1:]
	if err := validatePlanReportImageDigest(digest); err != nil {
		return "", "", fmt.Errorf("pinned image digest: %w", err)
	}
	colon := strings.LastIndex(nameTag, ":")
	if colon <= 0 || colon == len(nameTag)-1 {
		return "", "", fmt.Errorf("pinned image %q is missing a tag", image)
	}
	tag = nameTag[colon+1:]
	if tag == "" || strings.Contains(tag, "/") {
		return "", "", fmt.Errorf("pinned image %q has an invalid tag", image)
	}
	return tag, digest, nil
}

func emptyDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func closePlanReportRuntime(runtime spanemuboost.RuntimeHandle) error {
	if closer, ok := runtime.(interface{ Close() error }); ok && closer != nil {
		return closer.Close()
	}
	return nil
}
