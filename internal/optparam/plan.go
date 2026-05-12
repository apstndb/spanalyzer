package optparam

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// PlanQueryVariant is the per-variant plan-contract entry. It is shaped to be
// emitted into QueryCodegenPlanQuery.Variants once this PoC is promoted into
// internal/querygen so downstream tools (e.g. spanner-query-plan-shape) can
// produce one execution plan per SQL variant.
type PlanQueryVariant struct {
	// Label is a stable identifier for the variant. It is the same value as
	// Variant.Key (alphabetized concatenation of present OmitWhenNull
	// params, joined with '+', or "(none)" when every block is omitted).
	Label string `json:"label" yaml:"label"`
	// SQL is the rewritten statement.
	SQL string `json:"sql" yaml:"sql"`
	// SQLSHA256 is the SHA-256 of SQL, hex-encoded, so plan contracts can
	// pin per-variant execution plans without re-running the generator.
	SQLSHA256 string `json:"sql_sha256" yaml:"sql_sha256"`
	// PresentParams lists the OmitWhenNull params kept in this variant.
	PresentParams []string `json:"present_params,omitempty" yaml:"present_params,omitempty"`
	// AbsentParams lists the OmitWhenNull params dropped in this variant.
	AbsentParams []string `json:"absent_params,omitempty" yaml:"absent_params,omitempty"`
}

// BuildPlanVariants turns a VerifyResult into plan-contract entries. It does
// not interpret the row type; that is shared across every variant and lives
// at the parent QueryCodegenPlanQuery level.
func BuildPlanVariants(result *VerifyResult) []PlanQueryVariant {
	if result == nil {
		return nil
	}
	out := make([]PlanQueryVariant, 0, len(result.Variants))
	for _, v := range result.Variants {
		out = append(out, PlanQueryVariant{
			Label:         v.Key(),
			SQL:           v.SQL,
			SQLSHA256:     sha256Hex(v.SQL),
			PresentParams: append([]string(nil), v.PresentParams...),
			AbsentParams:  append([]string(nil), v.AbsentParams...),
		})
	}
	return out
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// FormatPlanVariants renders a human-readable summary of the plan-contract
// entries; useful for the PoC test output and for confirming what would land
// in the plan contract.
func FormatPlanVariants(entries []PlanQueryVariant) string {
	var s string
	for i, e := range entries {
		s += fmt.Sprintf("variant[%d] %s (sha256=%s)\n  present=%v\n  absent=%v\n  sql=%q\n",
			i, e.Label, e.SQLSHA256[:12], e.PresentParams, e.AbsentParams, e.SQL)
	}
	return s
}
