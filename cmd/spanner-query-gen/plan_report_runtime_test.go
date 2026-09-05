package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apstndb/spanemuboost"
)

func TestSelectPlanReportRuntimeColdStartUsesExactPlatformPin(t *testing.T) {
	amd64 := "us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta@sha256:48631bc2f3999358368a81042c7974abb131e05c04d246206fb7b82eae41789f"
	arm64 := "us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta@sha256:ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741"
	tests := []struct {
		arch     string
		want     string
		wantHash string
	}{
		{arch: "amd64", want: amd64, wantHash: "sha256:48631bc2f3999358368a81042c7974abb131e05c04d246206fb7b82eae41789f"},
		{arch: "arm64", want: arm64, wantHash: "sha256:ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741"},
	}
	for _, tt := range tests {
		t.Run("linux/"+tt.arch, func(t *testing.T) {
			var constructed []string
			var pinCalls int
			selected, err := selectPlanReportRuntime(planReportOptions{Backend: "omni"}, t.TempDir(), planReportRuntimeDeps{
				InspectOmniEndpoint: func() (bool, error) { return false, nil },
				HostGOARCH:          tt.arch,
				FindRepositoryRoot:  func(string) (string, error) { return "/repo", nil },
				ImageForPlatform: func(repoRoot, kind, platform string) (string, error) {
					pinCalls++
					if repoRoot != "/repo" || kind != "omni" || platform != "linux/"+tt.arch {
						t.Fatalf("ImageForPlatform(%q, %q, %q)", repoRoot, kind, platform)
					}
					if platform == "linux/amd64" {
						return amd64, nil
					}
					return arm64, nil
				},
				NewAttachedRuntime: func() (spanemuboost.RuntimeHandle, error) {
					t.Fatal("attached constructor called")
					return nil, nil
				},
				NewColdStartRuntime: func(image string) (spanemuboost.RuntimeHandle, error) {
					constructed = append(constructed, image)
					return spanemuboost.NewLazyRuntime(spanemuboost.BackendOmni, spanemuboost.WithContainerImage(image)), nil
				},
			})
			if err != nil {
				t.Fatalf("selectPlanReportRuntime() error = %v", err)
			}
			if pinCalls != 1 {
				t.Fatalf("pin resolver calls = %d, want 1", pinCalls)
			}
			if got, want := constructed, []string{tt.want}; !equalStrings(got, want) {
				t.Fatalf("configured images = %v, want %v", got, want)
			}
			if selected.Image != tt.want {
				t.Fatalf("selected image = %q, want %q", selected.Image, tt.want)
			}
			if got, want := selected.Identity, (planReportBackendIdentity{
				Kind:        "omni",
				Version:     "2026.r2.1-beta",
				ImageDigest: tt.wantHash,
				Source:      planReportIdentitySourceRuntimeTargets,
			}); got != want {
				t.Fatalf("identity = %+v, want %+v", got, want)
			}
			if err := closePlanReportRuntime(selected.Runtime); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
		})
	}
}

func TestSelectPlanReportRuntimeMissingPinDoesNotConstructRuntime(t *testing.T) {
	var constructed int
	_, err := selectPlanReportRuntime(planReportOptions{Backend: "omni"}, ".", planReportRuntimeDeps{
		InspectOmniEndpoint: func() (bool, error) { return false, nil },
		HostGOARCH:          "arm64",
		FindRepositoryRoot:  func(string) (string, error) { return "/repo", nil },
		ImageForPlatform: func(string, string, string) (string, error) {
			return "", errors.New("runtime target \"omni\" has no platform \"linux/arm64\"")
		},
		NewColdStartRuntime: func(string) (spanemuboost.RuntimeHandle, error) {
			constructed++
			return nil, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "Omni") || !strings.Contains(err.Error(), "linux/arm64") {
		t.Fatalf("error = %v, want Omni linux/arm64 pin context", err)
	}
	if constructed != 0 {
		t.Fatalf("cold-start constructor calls = %d, want 0", constructed)
	}
}

func TestSelectPlanReportRuntimeAttachedNeverResolvesPin(t *testing.T) {
	digest := "sha256:" + strings.Repeat("a", 64)
	tests := []struct {
		name      string
		assertion planReportBackendIdentity
		want      planReportBackendIdentity
	}{
		{
			name: "manual assertion",
			assertion: planReportBackendIdentity{
				Kind:        "omni",
				Version:     "2026.r1-beta",
				ImageDigest: digest,
				Source:      planReportIdentitySourceManual,
			},
			want: planReportBackendIdentity{
				Kind:        "omni",
				Version:     "2026.r1-beta",
				ImageDigest: digest,
				Source:      planReportIdentitySourceManual,
			},
		},
		{
			name: "external unverified",
			want: planReportBackendIdentity{
				Kind:        "omni",
				Version:     "not_recorded",
				ImageDigest: "not_recorded",
				Source:      planReportIdentitySourceExternalUnverified,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pinCalls, attachedCalls, coldCalls int
			selected, err := selectPlanReportRuntime(planReportOptions{Backend: "omni", Identity: tt.assertion}, ".", planReportRuntimeDeps{
				InspectOmniEndpoint: func() (bool, error) { return true, nil },
				FindRepositoryRoot: func(string) (string, error) {
					pinCalls++
					return "", errors.New("pin resolver must not run")
				},
				ImageForPlatform: func(string, string, string) (string, error) {
					pinCalls++
					return "", errors.New("pin resolver must not run")
				},
				NewAttachedRuntime: func() (spanemuboost.RuntimeHandle, error) {
					attachedCalls++
					return spanemuboost.NewLazyRuntime(spanemuboost.BackendOmni), nil
				},
				NewColdStartRuntime: func(string) (spanemuboost.RuntimeHandle, error) {
					coldCalls++
					return nil, nil
				},
			})
			if err != nil {
				t.Fatalf("selectPlanReportRuntime() error = %v", err)
			}
			if pinCalls != 0 || coldCalls != 0 || attachedCalls != 1 {
				t.Fatalf("pin=%d attached=%d cold=%d", pinCalls, attachedCalls, coldCalls)
			}
			if selected.Identity != tt.want {
				t.Fatalf("identity = %+v, want %+v", selected.Identity, tt.want)
			}
			if err := closePlanReportRuntime(selected.Runtime); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
		})
	}
}

func TestSelectPlanReportRuntimeEmulatorEndpointFileIsNotAttachedOmni(t *testing.T) {
	endpointPath := filepath.Join(t.TempDir(), "endpoint.json")
	if err := os.WriteFile(endpointPath, []byte(`{
  "backend": "emulator",
  "uri": "127.0.0.1:9010",
  "project_id": "test-project",
  "instance_id": "test-instance"
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SPANEMUBOOST_ENDPOINT_FILE", endpointPath)
	t.Setenv("SPANEMUBOOST_OMNI_URI", "")
	t.Setenv("SPANEMUBOOST_EMULATOR_URI", "")

	image := "us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta@sha256:" + strings.Repeat("a", 64)
	deps := defaultPlanReportRuntimeDeps()
	var attachedCalls, coldCalls int
	deps.HostGOARCH = "arm64"
	deps.FindRepositoryRoot = func(string) (string, error) { return "/repo", nil }
	deps.ImageForPlatform = func(string, string, string) (string, error) { return image, nil }
	deps.NewAttachedRuntime = func() (spanemuboost.RuntimeHandle, error) {
		attachedCalls++
		return spanemuboost.NewLazyRuntime(spanemuboost.BackendOmni), nil
	}
	deps.NewColdStartRuntime = func(string) (spanemuboost.RuntimeHandle, error) {
		coldCalls++
		return spanemuboost.NewLazyRuntime(spanemuboost.BackendOmni), nil
	}

	selected, err := selectPlanReportRuntime(planReportOptions{Backend: "omni"}, ".", deps)
	if err != nil {
		t.Fatalf("selectPlanReportRuntime() error = %v", err)
	}
	if attachedCalls != 0 || coldCalls != 1 {
		t.Fatalf("attached calls = %d, cold calls = %d; emulator endpoint file must not attach Omni", attachedCalls, coldCalls)
	}
	if selected.Identity.Source != planReportIdentitySourceRuntimeTargets {
		t.Fatalf("identity source = %q, want %q", selected.Identity.Source, planReportIdentitySourceRuntimeTargets)
	}
}

func TestSelectPlanReportRuntimeMalformedEndpointFileFailsClosed(t *testing.T) {
	endpointPath := filepath.Join(t.TempDir(), "endpoint.json")
	if err := os.WriteFile(endpointPath, []byte(`{`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SPANEMUBOOST_ENDPOINT_FILE", endpointPath)
	t.Setenv("SPANEMUBOOST_OMNI_URI", "")
	t.Setenv("SPANEMUBOOST_EMULATOR_URI", "")

	var attachedCalls, coldCalls int
	deps := defaultPlanReportRuntimeDeps()
	deps.NewAttachedRuntime = func() (spanemuboost.RuntimeHandle, error) {
		attachedCalls++
		return nil, nil
	}
	deps.NewColdStartRuntime = func(string) (spanemuboost.RuntimeHandle, error) {
		coldCalls++
		return nil, nil
	}
	_, err := selectPlanReportRuntime(planReportOptions{Backend: "omni"}, ".", deps)
	if err == nil {
		t.Fatal("selectPlanReportRuntime() error = nil, want malformed endpoint failure")
	}
	if attachedCalls != 0 || coldCalls != 0 {
		t.Fatalf("attached=%d cold=%d, want no constructor after malformed endpoint", attachedCalls, coldCalls)
	}
}

func TestSelectPlanReportRuntimeRejectsConflictingColdStartAssertion(t *testing.T) {
	image := "us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta@sha256:ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741"
	var constructed int
	_, err := selectPlanReportRuntime(planReportOptions{
		Backend: "omni",
		Identity: planReportBackendIdentity{
			Kind:        "omni",
			Version:     "other-tag",
			ImageDigest: "not_recorded",
			Source:      planReportIdentitySourceManual,
		},
	}, ".", planReportRuntimeDeps{
		InspectOmniEndpoint: func() (bool, error) { return false, nil },
		HostGOARCH:          "arm64",
		FindRepositoryRoot:  func(string) (string, error) { return "/repo", nil },
		ImageForPlatform:    func(string, string, string) (string, error) { return image, nil },
		NewColdStartRuntime: func(string) (spanemuboost.RuntimeHandle, error) {
			constructed++
			return nil, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "other-tag") || !strings.Contains(err.Error(), "2026.r2.1-beta") {
		t.Fatalf("error = %v, want conflicting tag diagnostic", err)
	}
	if constructed != 0 {
		t.Fatalf("constructor calls = %d, want 0", constructed)
	}
}

func TestSelectPlanReportRuntimeMatchingColdStartAssertionKeepsConfiguredIdentity(t *testing.T) {
	image := "us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta@sha256:ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741"
	selected, err := selectPlanReportRuntime(planReportOptions{
		Backend: "omni",
		Identity: planReportBackendIdentity{
			Kind:        "omni",
			Version:     "2026.r2.1-beta",
			ImageDigest: "sha256:ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741",
			Source:      planReportIdentitySourceManual,
		},
	}, ".", planReportRuntimeDeps{
		InspectOmniEndpoint: func() (bool, error) { return false, nil },
		HostGOARCH:          "arm64",
		FindRepositoryRoot:  func(string) (string, error) { return "/repo", nil },
		ImageForPlatform:    func(string, string, string) (string, error) { return image, nil },
		NewColdStartRuntime: func(string) (spanemuboost.RuntimeHandle, error) {
			return spanemuboost.NewLazyRuntime(spanemuboost.BackendOmni), nil
		},
	})
	if err != nil {
		t.Fatalf("selectPlanReportRuntime() error = %v", err)
	}
	if selected.Identity.Source != planReportIdentitySourceRuntimeTargets || selected.Identity.Version != "2026.r2.1-beta" {
		t.Fatalf("identity = %+v, want configured runtime_targets", selected.Identity)
	}
}

func TestParsePinnedContainerImage(t *testing.T) {
	tag, digest, err := parsePinnedContainerImage("us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r2.1-beta@sha256:ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741")
	if err != nil {
		t.Fatalf("parsePinnedContainerImage() error = %v", err)
	}
	if tag != "2026.r2.1-beta" {
		t.Fatalf("tag = %q", tag)
	}
	if digest != "sha256:ed31d9ee72eeee69cac78566eb3a6e72ee389b26234735f0ef449774cc006741" {
		t.Fatalf("digest = %q", digest)
	}
	if _, _, err := parsePinnedContainerImage("missing-digest"); err == nil {
		t.Fatal("parsePinnedContainerImage() error = nil, want missing digest")
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
