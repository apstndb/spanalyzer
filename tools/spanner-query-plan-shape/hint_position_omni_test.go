//go:build integration && omni

package main

import (
	"testing"
)

func TestIntegrationHintPositionAuditOnOmni(t *testing.T) {
	clients := openOmniClients(t, hintPositionDDLs(t))
	runHintPositionAuditCasesWithOverrides(t, clients.Client, map[string]hintPositionExpectation{
		"hint-position/versioned/pipe-finish": hintPositionAccepted,
	})
}
