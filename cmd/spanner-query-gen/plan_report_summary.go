package main

import (
	spanalyzer "github.com/apstndb/spanalyzer"
	"github.com/apstndb/spanalyzer/plancontract"
)

const planReportSummaryVersionV1Alpha = "v1alpha-plan-summary-v1"

// Keep this projection explicit: adding evidence to a full report must not
// accidentally add bulky fields to summaries. Contracts are evaluated against
// the full in-memory report before this serialization-only projection.
type planReportSummary struct {
	ReportVersion          string                         `json:"report_version"`
	Status                 string                         `json:"status"`
	Backend                string                         `json:"backend"`
	CreatedDatabaseCleanup string                         `json:"created_database_cleanup,omitempty"`
	Input                  planReportInput                `json:"input"`
	PlanSource             planReportSummarySource        `json:"plan_source"`
	BackendIdentity        planReportBackendIdentity      `json:"backend_identity"`
	Normalization          planReportNormalization        `json:"normalization"`
	TargetSummary          planReportTargetSummary        `json:"target_summary"`
	Stable                 bool                           `json:"stable"`
	Queries                []planReportSummaryQuery       `json:"queries"`
	ContractEvalMode       string                         `json:"contract_evaluation_mode"`
	ContractFileVersion    string                         `json:"contract_file_version,omitempty"`
	ContractEvaluator      string                         `json:"contract_evaluator_version,omitempty"`
	ContractEvaluations    []planContractEvaluation       `json:"contract_evaluations,omitempty"`
	ContractSummary        *planContractEvaluationSummary `json:"contract_summary,omitempty"`
	Warnings               []planReportDiagnostic         `json:"warnings"`
	Optimizer              planReportOptimizer            `json:"optimizer"`
}

type planReportSummarySource struct {
	Backend string `json:"backend"`
	API     string `json:"api"`
}

type planReportSummaryQuery struct {
	MutationCounts        map[string]spanalyzer.MutationCountReport `json:"mutation_counts,omitempty"`
	MutationCountError    string                                    `json:"mutation_count_error,omitempty"`
	TargetID              string                                    `json:"target_id"`
	Name                  string                                    `json:"name"`
	Catalog               string                                    `json:"catalog"`
	Scope                 string                                    `json:"scope,omitempty"`
	Kind                  string                                    `json:"kind"`
	Status                string                                    `json:"status"`
	SQLSHA256             string                                    `json:"sql_sha256,omitempty"`
	DDLSHA256             string                                    `json:"ddl_sha256,omitempty"`
	ProtoDescriptorSHA256 string                                    `json:"proto_descriptor_sha256,omitempty"`
	OperatorTreeSHA256    string                                    `json:"operator_tree_sha256,omitempty"`
	AccessPaths           interface{}                               `json:"access_paths,omitempty"`
	Variant               *planReportSummaryVariant                 `json:"variant,omitempty"`
	Error                 string                                    `json:"error,omitempty"`
	OptimizerNotPinned    bool                                      `json:"optimizer_not_pinned,omitempty"`
	PlanEnvironmentNotes  []string                                  `json:"plan_environment_notes,omitempty"`
	ClassificationNotes   []planReportDiagnostic                    `json:"classification_warnings,omitempty"`
}

type planReportSummaryVariant struct {
	Label             string            `json:"label"`
	PresentParams     []string          `json:"present_params,omitempty"`
	AbsentParams      []string          `json:"absent_params,omitempty"`
	ChoiceAssignments map[string]string `json:"choice_assignments,omitempty"`
}

func summarizePlanReport(report planReport) planReportSummary {
	summary := planReportSummary{
		ReportVersion:          planReportSummaryVersionV1Alpha,
		Status:                 report.Status,
		Backend:                report.Backend,
		CreatedDatabaseCleanup: report.CreatedDatabaseCleanup,
		Input:                  report.Input,
		PlanSource:             planReportSummarySource{Backend: report.PlanSource.Backend, API: report.PlanSource.API},
		BackendIdentity:        report.BackendIdentity,
		Normalization:          report.Normalization,
		TargetSummary:          report.TargetSummary,
		Stable:                 report.Stable,
		Queries:                make([]planReportSummaryQuery, 0, len(report.Queries)),
		ContractEvalMode:       report.ContractEvalMode,
		ContractFileVersion:    report.ContractFileVersion,
		ContractEvaluator:      report.ContractEvaluator,
		ContractSummary:        report.ContractSummary,
		Warnings:               report.Warnings,
		Optimizer:              report.Optimizer,
	}
	for _, query := range report.Queries {
		q := planReportSummaryQuery{
			MutationCounts:        query.MutationCounts,
			MutationCountError:    query.MutationCountError,
			TargetID:              query.TargetID,
			Name:                  query.Name,
			Catalog:               query.Catalog,
			Scope:                 query.Scope,
			Kind:                  query.Kind,
			Status:                query.Status,
			SQLSHA256:             query.SQLSHA256,
			DDLSHA256:             query.DDLSHA256,
			ProtoDescriptorSHA256: query.ProtoDescriptorSHA256,
			OperatorTreeSHA256:    query.OperatorTreeSHA256,
			Error:                 query.Error,
			OptimizerNotPinned:    query.OptimizerNotPinned,
			PlanEnvironmentNotes:  query.PlanEnvironmentNotes,
			ClassificationNotes:   query.ClassificationNotes,
		}
		if query.Status == "ok" {
			paths := query.AccessPaths
			if paths == nil {
				paths = []plancontract.AccessPath{}
			}
			q.AccessPaths = paths
		}
		if query.Variant != nil {
			q.Variant = &planReportSummaryVariant{
				Label:             query.Variant.Label,
				PresentParams:     query.Variant.PresentParams,
				AbsentParams:      query.Variant.AbsentParams,
				ChoiceAssignments: query.Variant.ChoiceAssignments,
			}
		}
		summary.Queries = append(summary.Queries, q)
	}
	for _, evaluation := range report.ContractEvaluations {
		// Even normalized contracts may require the omitted operator graph.
		// Copy the reasons before appending so full-report replay metadata stays
		// unchanged when the same report is also serialized in full.
		evaluation.Stability.ReplayableFromReport = false
		evaluation.Stability.Reasons = append(append([]string(nil), evaluation.Stability.Reasons...), "summary_projection_omits_full_plan_evidence")
		summary.ContractEvaluations = append(summary.ContractEvaluations, evaluation)
	}
	return summary
}
