//go:build integration && omni

package main

import (
	"testing"
)

func TestIntegrationHintPositionAuditOnOmni(t *testing.T) {
	clients := openOmniClients(t, hintPositionDDLs(t))
	runHintPositionAuditCases(t, clients.Client)
}
