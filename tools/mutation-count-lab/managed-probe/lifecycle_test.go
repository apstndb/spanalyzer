package main

import (
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestManagedTargetRefusesEmulatorOverride(t *testing.T) {
	t.Setenv("SPANNER_EMULATOR_HOST", "localhost:9010")
	t.Setenv("SPANNER_PROJECT_ID", "test-project")
	t.Setenv("SPANNER_INSTANCE_ID", "test-instance")
	if _, err := managedTarget(); err == nil {
		t.Fatal("managed fixture must not silently run on emulator")
	}
}

func TestLeaseErrorRemovesLiveToken(t *testing.T) {
	s := redactLeaseError(status.Error(codes.InvalidArgument, "bad token live-secret-token"), "live-secret-token")
	if strings.Contains(s, "live-secret-token") || !strings.Contains(s, "<lease-token>") {
		t.Fatal("lease token redaction failed:", s)
	}
}
func TestRedactManagedEvidence(t *testing.T) {
	t.Setenv("SPANNER_PROJECT_ID", "private-project")
	t.Setenv("SPANNER_INSTANCE_ID", "private-instance")
	t.Setenv("SPANNER_DATABASE_ID", "private-base")
	s := redact(`database projects/private-project/instances/private-instance/databases/mutation1110-0123abcd failed; mutation1110-0123abcd; requestID = "secret-request"`)
	for _, v := range []string{"private-project", "private-instance", "0123abcd", "secret-request"} {
		if strings.Contains(s, v) {
			t.Fatal("retained identifier:", s)
		}
	}
}
