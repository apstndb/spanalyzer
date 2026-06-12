package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	spanalyzer "github.com/apstndb/go-googlesql-spanner-poc"
	"github.com/apstndb/go-googlesql-spanner-poc/internal/plancontract"
	"github.com/apstndb/go-googlesql-spanner-poc/internal/querygen"
	"github.com/apstndb/spanemuboost"
	"github.com/apstndb/spannerplan/plantree/reference"
	"github.com/cloudspannerecosystem/memefish"
)

func runPlanReport(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("spanner-query-gen plan-report", flag.ContinueOnError)
	fs.SetOutput(flagOutput(args, stdout, stderr))
	configPath := fs.String("config", "spanner-query-gen.yaml", "query code generation config file")
	output := fs.String("output", "markdown", "report output format: markdown, yaml, or json")
	backend := fs.String("backend", "omni", "runtime backend: omni")
	renderFormat := fs.String("format", "current", "spannerplan tree format: current, traditional, or compact")
	wrapWidth := fs.Int("wrap-width", 0, "maximum rendered plan width; 0 disables wrapping")
	stable := fs.Bool("stable", false, "omit volatile metadata from report output")
	optimizerVersion := fs.String("optimizer-version", "", "Spanner optimizer version to pass to AnalyzeQuery")
	optimizerStatisticsPackage := fs.String("optimizer-statistics-package", "", "Spanner optimizer statistics package to pass to AnalyzeQuery")
	backendVersion := fs.String("backend-version", "", "backend version to record in backend_identity")
	backendImageDigest := fs.String("backend-image-digest", "", "backend container image digest to record in backend_identity, for example sha256:<64 hex chars>")
	requireTargets := fs.Bool("require-targets", false, "fail when no Spanner query targets are available")
	requireOptimizerPinning := fs.Bool("require-optimizer-pinning", false, "fail when optimizer version or statistics package is not pinned")
	contractsPath := fs.String("contracts", "", "experimental plan contracts YAML file")
	checkContracts := fs.Bool("check", false, "evaluate --contracts and fail when a contract is violated")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if strings.ToLower(strings.TrimSpace(*backend)) != "omni" {
		return fmt.Errorf("unsupported --backend %q; use omni", *backend)
	}
	if *checkContracts && strings.TrimSpace(*contractsPath) == "" {
		return fmt.Errorf("--check requires --contracts")
	}
	format, err := reference.ParseFormat(*renderFormat)
	if err != nil {
		return err
	}
	identity, err := planReportBackendIdentityFromFlags(strings.ToLower(strings.TrimSpace(*backend)), *backendVersion, *backendImageDigest)
	if err != nil {
		return err
	}

	config, err := readConfig(*configPath)
	if err != nil {
		return err
	}
	configDigest, err := planReportFileDigest(*configPath)
	if err != nil {
		return err
	}
	baseDir := filepath.Dir(*configPath)
	plan, err := querygen.BuildQueryCodegenPlan(config, baseDir)
	if err != nil {
		return err
	}
	report, err := buildPlanReport(context.Background(), config, plan, baseDir, planReportOptions{
		Backend:    strings.ToLower(strings.TrimSpace(*backend)),
		Format:     format,
		RenderMode: reference.RenderModePlan,
		WrapWidth:  *wrapWidth,
		Stable:     *stable,
		Identity:   identity,
		Optimizer: planReportOptimizerEnvironment{
			Version:           strings.TrimSpace(*optimizerVersion),
			StatisticsPackage: strings.TrimSpace(*optimizerStatisticsPackage),
		},
	})
	if err != nil {
		return err
	}
	report.Input.ConfigSHA256 = configDigest
	if strings.TrimSpace(*contractsPath) != "" {
		resolvedContractsPath := resolveOutputPath(baseDir, *contractsPath)
		contractDigest, err := planReportFileDigest(resolvedContractsPath)
		if err != nil {
			return err
		}
		contracts, err := readPlanContracts(resolvedContractsPath)
		if err != nil {
			return err
		}
		if err := applyPlanContracts(&report, contracts); err != nil {
			return err
		}
		report.Input.ContractFileSHA256 = contractDigest
		report.Input.ContractFilePath = planReportStableInputPath(baseDir, resolvedContractsPath, *contractsPath, report.Stable)
		if *checkContracts {
			report.ContractEvalMode = planContractEvaluationModeCheck
			addPlanReportContractCheckWarnings(&report)
		} else {
			report.ContractEvalMode = planContractEvaluationModeReportOnly
		}
	}
	if err := writePlanReport(stdout, *output, report); err != nil {
		return err
	}
	if *requireTargets && report.Status == planReportStatusNoTargets {
		return fmt.Errorf("plan-report found no Spanner query targets")
	}
	if *requireOptimizerPinning {
		if warnings := planReportOptimizerPinningWarnings(report); len(warnings) > 0 {
			return fmt.Errorf("optimizer pinning is required but not satisfied: %s", strings.Join(warnings, ", "))
		}
	}
	if *checkContracts {
		if failures := planReportContractCheckFailureCount(report); failures > 0 {
			return fmt.Errorf("plan contract check failed: %d failed or not evaluated contract(s)", failures)
		}
	}
	return nil
}

type planReportOptions struct {
	Backend    string
	Format     reference.Format
	RenderMode reference.RenderMode
	WrapWidth  int
	Stable     bool
	Identity   planReportBackendIdentity
	Optimizer  planReportOptimizerEnvironment
}

const (
	planReportStatusOK        = "ok"
	planReportStatusNoTargets = "no_targets"

	planContractStatusPass         = plancontract.StatusPass
	planContractStatusFail         = plancontract.StatusFail
	planContractStatusNotEvaluated = plancontract.StatusNotEvaluated

	planReportVersionV1Alpha        = "v1alpha-plan-report-v1"
	planContractFileVersionV1Alpha  = plancontract.FileVersionV1Alpha
	planContractEvaluatorVersionV1  = plancontract.EvaluatorVersionV1
	planContractStabilityNormalized = plancontract.StabilityNormalized
	planContractStabilityRawPlan    = plancontract.StabilityRawPlan

	planContractEvaluationModeNone       = plancontract.EvaluationModeNone
	planContractEvaluationModeReportOnly = plancontract.EvaluationModeReportOnly
	planContractEvaluationModeCheck      = plancontract.EvaluationModeCheck

	planContractFailureKindViolation             = plancontract.FailureKindViolation
	planContractFailureKindClassificationUnknown = plancontract.FailureKindClassificationUnknown
	planContractReasonTargetNotFound             = plancontract.ReasonTargetNotFound
	planContractReasonTargetError                = plancontract.ReasonTargetError
)

const (
	planContractIdentifierPattern = plancontract.IdentifierPattern
	planContractTargetIDPattern   = plancontract.TargetIDPattern
)

type planReport struct {
	ReportVersion       string                         `json:"report_version" yaml:"report_version"`
	Status              string                         `json:"status" yaml:"status"`
	Backend             string                         `json:"backend" yaml:"backend"`
	Input               planReportInput                `json:"input" yaml:"input"`
	PlanSource          planReportPlanSource           `json:"plan_source" yaml:"plan_source"`
	BackendIdentity     planReportBackendIdentity      `json:"backend_identity" yaml:"backend_identity"`
	Normalization       planReportNormalization        `json:"normalization" yaml:"normalization"`
	TargetSummary       planReportTargetSummary        `json:"target_summary" yaml:"target_summary"`
	Format              string                         `json:"format" yaml:"format"`
	RenderMode          string                         `json:"render_mode" yaml:"render_mode"`
	Stable              bool                           `json:"stable" yaml:"stable"`
	Queries             []planReportQuery              `json:"queries" yaml:"queries"`
	ContractEvalMode    string                         `json:"contract_evaluation_mode" yaml:"contract_evaluation_mode"`
	ContractFileVersion string                         `json:"contract_file_version,omitempty" yaml:"contract_file_version,omitempty"`
	ContractEvaluator   string                         `json:"contract_evaluator_version,omitempty" yaml:"contract_evaluator_version,omitempty"`
	ContractEvaluations []planContractEvaluation       `json:"contract_evaluations,omitempty" yaml:"contract_evaluations,omitempty"`
	ContractSummary     *planContractEvaluationSummary `json:"contract_summary,omitempty" yaml:"contract_summary,omitempty"`
	Warnings            []planReportDiagnostic         `json:"warnings" yaml:"warnings"`
	Optimizer           planReportOptimizer            `json:"optimizer" yaml:"optimizer"`
}

type planReportInput struct {
	ConfigSHA256       string `json:"config_sha256,omitempty" yaml:"config_sha256,omitempty"`
	ContractFileSHA256 string `json:"contract_file_sha256,omitempty" yaml:"contract_file_sha256,omitempty"`
	ContractFilePath   string `json:"contract_file_path,omitempty" yaml:"contract_file_path,omitempty"`
}

type planReportPlanSource struct {
	Backend    string `json:"backend" yaml:"backend"`
	API        string `json:"api" yaml:"api"`
	RenderTool string `json:"render_tool" yaml:"render_tool"`
}

type planReportBackendIdentity struct {
	Kind        string `json:"kind" yaml:"kind"`
	Version     string `json:"version" yaml:"version"`
	ImageDigest string `json:"image_digest" yaml:"image_digest"`
	Source      string `json:"source" yaml:"source"`
}

type planReportNormalization struct {
	OperatorTreeVersion          string                     `json:"operator_tree_version" yaml:"operator_tree_version"`
	OperatorFamilyMappingVersion string                     `json:"operator_family_mapping_version" yaml:"operator_family_mapping_version"`
	CELInputDefaults             planReportCELInputDefaults `json:"cel_input_defaults" yaml:"cel_input_defaults"`
}

type planReportCELInputDefaults struct {
	OptionalString  string   `json:"optional_string" yaml:"optional_string"`
	OptionalBoolean bool     `json:"optional_boolean" yaml:"optional_boolean"`
	AppliesTo       []string `json:"applies_to" yaml:"applies_to"`
}

type planReportTargetSummary struct {
	IncludedCount int                        `json:"included_count" yaml:"included_count"`
	Planned       int                        `json:"planned" yaml:"planned"`
	Errors        int                        `json:"errors" yaml:"errors"`
	Skipped       int                        `json:"skipped" yaml:"skipped"`
	Excluded      []planReportExcludedTarget `json:"excluded" yaml:"excluded"`
}

type planReportExcludedTarget struct {
	ID      string `json:"id" yaml:"id"`
	Query   string `json:"query" yaml:"query"`
	Catalog string `json:"catalog,omitempty" yaml:"catalog,omitempty"`
	Scope   string `json:"scope" yaml:"scope"`
	Reason  string `json:"reason" yaml:"reason"`
}

type planReportQuery struct {
	TargetID             string                   `json:"target_id" yaml:"target_id"`
	Name                 string                   `json:"name" yaml:"name"`
	Catalog              string                   `json:"catalog" yaml:"catalog"`
	Scope                string                   `json:"scope,omitempty" yaml:"scope,omitempty"`
	Kind                 string                   `json:"kind" yaml:"kind"`
	Status               string                   `json:"status" yaml:"status"`
	SQL                  string                   `json:"sql,omitempty" yaml:"sql,omitempty"`
	SQLSHA256            string                   `json:"sql_sha256,omitempty" yaml:"sql_sha256,omitempty"`
	DDLSHA256            string                   `json:"ddl_sha256,omitempty" yaml:"ddl_sha256,omitempty"`
	OperatorTreeSHA256   string                   `json:"operator_tree_sha256,omitempty" yaml:"operator_tree_sha256,omitempty"`
	OperatorFamilies     []string                 `json:"operator_families,omitempty" yaml:"operator_families,omitempty"`
	OperatorFamilyCounts map[string]int           `json:"operator_family_counts,omitempty" yaml:"operator_family_counts,omitempty"`
	NormalizedOperators  []planReportOperator     `json:"normalized_operators,omitempty" yaml:"normalized_operators,omitempty"`
	OperatorEdges        []planReportOperatorEdge `json:"operator_edges,omitempty" yaml:"operator_edges,omitempty"`
	Plan                 string                   `json:"plan,omitempty" yaml:"plan,omitempty"`
	Error                string                   `json:"error,omitempty" yaml:"error,omitempty"`
	OptimizerNotPinned   bool                     `json:"optimizer_not_pinned,omitempty" yaml:"optimizer_not_pinned,omitempty"`
	PlanEnvironmentNotes []string                 `json:"plan_environment_notes,omitempty" yaml:"plan_environment_notes,omitempty"`
	ClassificationNotes  []planReportDiagnostic   `json:"classification_warnings,omitempty" yaml:"classification_warnings,omitempty"`
	RawPlan              *spannerpb.QueryPlan     `json:"-" yaml:"-"`
}

type serializedPlanReportQuery struct {
	TargetID             string                 `json:"target_id" yaml:"target_id"`
	Name                 string                 `json:"name" yaml:"name"`
	Catalog              string                 `json:"catalog" yaml:"catalog"`
	Scope                string                 `json:"scope,omitempty" yaml:"scope,omitempty"`
	Kind                 string                 `json:"kind" yaml:"kind"`
	Status               string                 `json:"status" yaml:"status"`
	SQL                  string                 `json:"sql,omitempty" yaml:"sql,omitempty"`
	SQLSHA256            string                 `json:"sql_sha256,omitempty" yaml:"sql_sha256,omitempty"`
	DDLSHA256            string                 `json:"ddl_sha256,omitempty" yaml:"ddl_sha256,omitempty"`
	OperatorTreeSHA256   string                 `json:"operator_tree_sha256,omitempty" yaml:"operator_tree_sha256,omitempty"`
	OperatorFamilies     []string               `json:"operator_families,omitempty" yaml:"operator_families,omitempty"`
	OperatorFamilyCounts map[string]int         `json:"operator_family_counts,omitempty" yaml:"operator_family_counts,omitempty"`
	NormalizedOperators  []planReportOperator   `json:"normalized_operators,omitempty" yaml:"normalized_operators,omitempty"`
	OperatorEdges        interface{}            `json:"operator_edges,omitempty" yaml:"operator_edges,omitempty"`
	Plan                 string                 `json:"plan,omitempty" yaml:"plan,omitempty"`
	Error                string                 `json:"error,omitempty" yaml:"error,omitempty"`
	OptimizerNotPinned   bool                   `json:"optimizer_not_pinned,omitempty" yaml:"optimizer_not_pinned,omitempty"`
	PlanEnvironmentNotes []string               `json:"plan_environment_notes,omitempty" yaml:"plan_environment_notes,omitempty"`
	ClassificationNotes  []planReportDiagnostic `json:"classification_warnings,omitempty" yaml:"classification_warnings,omitempty"`
}

func (q planReportQuery) MarshalJSON() ([]byte, error) {
	return json.Marshal(q.serialized())
}

func (q planReportQuery) MarshalYAML() (interface{}, error) {
	return q.serialized(), nil
}

func (q planReportQuery) serialized() serializedPlanReportQuery {
	var operatorEdges interface{}
	if q.Status == "ok" {
		edges := q.OperatorEdges
		if edges == nil {
			edges = []planReportOperatorEdge{}
		}
		operatorEdges = edges
	}
	return serializedPlanReportQuery{
		TargetID:             q.TargetID,
		Name:                 q.Name,
		Catalog:              q.Catalog,
		Scope:                q.Scope,
		Kind:                 q.Kind,
		Status:               q.Status,
		SQL:                  q.SQL,
		SQLSHA256:            q.SQLSHA256,
		DDLSHA256:            q.DDLSHA256,
		OperatorTreeSHA256:   q.OperatorTreeSHA256,
		OperatorFamilies:     q.OperatorFamilies,
		OperatorFamilyCounts: q.OperatorFamilyCounts,
		NormalizedOperators:  q.NormalizedOperators,
		OperatorEdges:        operatorEdges,
		Plan:                 q.Plan,
		Error:                q.Error,
		OptimizerNotPinned:   q.OptimizerNotPinned,
		PlanEnvironmentNotes: q.PlanEnvironmentNotes,
		ClassificationNotes:  q.ClassificationNotes,
	}
}

type planReportOperator = plancontract.Operator
type planReportOperatorEdge = plancontract.OperatorEdge
type planReportDiagnostic = plancontract.Diagnostic
type planReportOptimizer = plancontract.Optimizer
type planReportOptimizerEnvironment = plancontract.OptimizerEnvironment
type planReportOptimizerEffective = plancontract.OptimizerEffective

func buildPlanReport(ctx context.Context, config querygen.QueryCodegenConfig, plan *querygen.QueryCodegenPlan, baseDir string, opts planReportOptions) (planReport, error) {
	runtime := spanemuboost.NewLazyRuntime(spanemuboost.BackendOmni)
	defer func() {
		_ = runtime.Close()
	}()
	return buildPlanReportWithRuntime(ctx, config, plan, baseDir, opts, runtime)
}

func buildPlanReportWithRuntime(ctx context.Context, config querygen.QueryCodegenConfig, plan *querygen.QueryCodegenPlan, baseDir string, opts planReportOptions, runtime spanemuboost.RuntimeHandle) (planReport, error) {
	report := planReport{
		ReportVersion:    planReportVersionV1Alpha,
		Status:           planReportStatusOK,
		Backend:          opts.Backend,
		Format:           string(opts.Format),
		RenderMode:       string(opts.RenderMode),
		Stable:           opts.Stable,
		ContractEvalMode: planContractEvaluationModeNone,
		PlanSource: planReportPlanSource{
			Backend:    opts.Backend,
			API:        "analyze_query",
			RenderTool: "spannerplan",
		},
		BackendIdentity: planReportBackendIdentity{
			Kind:        opts.Backend,
			Version:     "not_recorded",
			ImageDigest: "not_recorded",
			Source:      "spanemuboost",
		},
		Normalization: defaultPlanReportNormalization(),
		TargetSummary: planReportTargetSummary{
			Excluded: []planReportExcludedTarget{},
		},
		Warnings:  []planReportDiagnostic{},
		Optimizer: planReportResolvedOptimizer(opts.Optimizer),
	}
	if opts.Identity.Kind != "" {
		report.BackendIdentity = opts.Identity
	}
	schemas := queryCodegenSchemasByName(config.Schemas)
	configQueries := queryCodegenQueriesByName(config.Queries)
	clientsBySource := map[string]*spanemuboost.Clients{}
	defer func() {
		for _, clients := range clientsBySource {
			_ = clients.Close()
		}
	}()

	targets := 0
	for _, query := range plan.Queries {
		configQuery := configQueries[query.Name]
		catalog := query.Catalog
		sql := query.SQL
		params := query.Params
		scope := "query"
		if !planReportFederatedQueryIsZero(configQuery.Federated) {
			catalog = configQuery.Federated.SpannerSource
			sql = configQuery.Federated.InnerSQL
			params = planReportScopedParams(params, "inner")
			scope = "external_query.inner"
		}
		reportQuery := planReportQuery{
			TargetID:  planReportTargetID(query.Name, scope),
			Name:      query.Name,
			Catalog:   catalog,
			Scope:     scope,
			Kind:      query.Kind,
			Status:    "pending",
			SQL:       sql,
			SQLSHA256: planReportDigest(sql),
		}
		report.TargetSummary.IncludedCount++
		schema, ok := schemas[catalog]
		if !ok {
			reportQuery.Status = "skipped"
			reportQuery.Error = fmt.Sprintf("source catalog %q not found", catalog)
			report.TargetSummary.Skipped++
			report.Queries = append(report.Queries, reportQuery)
			continue
		}
		if schema.Dialect != "spanner" {
			reportQuery.Status = "skipped"
			reportQuery.Error = fmt.Sprintf("source catalog %q has dialect %q; plan-report currently supports Spanner GoogleSQL catalogs", catalog, schema.Dialect)
			report.TargetSummary.Skipped++
			report.Queries = append(report.Queries, reportQuery)
			continue
		}
		targets++
		ddlDigest, err := planReportDDLDigest(schema, baseDir)
		if err != nil {
			reportQuery.Status = "error"
			reportQuery.Error = err.Error()
			report.TargetSummary.Errors++
			report.Queries = append(report.Queries, reportQuery)
			continue
		}
		reportQuery.DDLSHA256 = ddlDigest
		reportQuery.OptimizerNotPinned = !planReportOptimizerEnvironmentPinned(report.Optimizer.Requested)
		if reportQuery.OptimizerNotPinned {
			reportQuery.PlanEnvironmentNotes = append(reportQuery.PlanEnvironmentNotes, "plan environment uses default Omni optimizer settings; optimizer version/statistics package are not pinned")
		}
		clients, ok := clientsBySource[catalog]
		if !ok {
			started, err := openPlanReportClients(ctx, runtime, schema, baseDir, report.Optimizer)
			if err != nil {
				return report, fmt.Errorf("open Omni database for catalog %s: %w", catalog, err)
			}
			clients = started
			clientsBySource[catalog] = clients
		}
		stmt := spanner.NewStatement(sql)
		values, err := planReportParamValues(params)
		if err != nil {
			reportQuery.Status = "error"
			reportQuery.Error = err.Error()
			report.TargetSummary.Errors++
			report.Queries = append(report.Queries, reportQuery)
			continue
		}
		stmt.Params = values
		queryPlan, err := clients.Client.Single().AnalyzeQuery(ctx, stmt)
		if err != nil {
			reportQuery.Status = "error"
			reportQuery.Error = err.Error()
			report.TargetSummary.Errors++
			report.Queries = append(report.Queries, reportQuery)
			continue
		}
		rendered, err := reference.RenderTreeTable(queryPlan.GetPlanNodes(), opts.RenderMode, opts.Format, opts.WrapWidth)
		if err != nil {
			reportQuery.Status = "error"
			reportQuery.Error = err.Error()
			report.TargetSummary.Errors++
			report.Queries = append(report.Queries, reportQuery)
			continue
		}
		reportQuery.NormalizedOperators = planReportOperators(queryPlan)
		reportQuery.OperatorEdges = planReportOperatorEdges(queryPlan)
		reportQuery.OperatorFamilies = planReportOperatorFamilies(reportQuery.NormalizedOperators)
		reportQuery.OperatorFamilyCounts = planReportOperatorFamilyCounts(reportQuery.NormalizedOperators)
		reportQuery.OperatorTreeSHA256 = planReportOperatorTreeDigest(queryPlan)
		reportQuery.ClassificationNotes = planReportClassificationWarnings(reportQuery.NormalizedOperators)
		reportQuery.RawPlan = queryPlan
		reportQuery.Status = "ok"
		reportQuery.Plan = strings.TrimRight(rendered, "\n")
		report.TargetSummary.Planned++
		report.Queries = append(report.Queries, reportQuery)
	}
	if targets == 0 {
		report.Status = planReportStatusNoTargets
		report.Warnings = append(report.Warnings, planReportDiagnostic{
			ID:      "plan-report-no-targets",
			Message: "no Spanner query targets were available for plan-report",
		})
	}
	return report, nil
}

func defaultPlanReportNormalization() planReportNormalization {
	return planReportNormalization{
		// Normalization identifiers follow the same mutable preview
		// philosophy as the v1alpha config: they may change in place while
		// the report format is pre-release, so they intentionally do not
		// promise stable v1/v2 digest comparability.
		OperatorTreeVersion:          "v1alpha",
		OperatorFamilyMappingVersion: "v1alpha",
		CELInputDefaults: planReportCELInputDefaults{
			OptionalString:  "",
			OptionalBoolean: false,
			AppliesTo:       []string{"operators[]", "operator_edges[]"},
		},
	}
}

func openPlanReportClients(ctx context.Context, runtime spanemuboost.RuntimeHandle, schema querygen.QueryCodegenSchema, baseDir string, optimizer planReportOptimizer) (*spanemuboost.Clients, error) {
	ddls, err := planReportDDLs(schema, baseDir)
	if err != nil {
		return nil, err
	}
	options := []spanemuboost.Option{
		spanemuboost.WithRandomDatabaseID(),
		spanemuboost.WithSetupDDLs(ddls),
	}
	if optimizer.Requested.Version != "not_pinned" || optimizer.Requested.StatisticsPackage != "not_pinned" {
		options = append(options, spanemuboost.WithClientConfig(spanner.ClientConfig{
			QueryOptions: spanner.QueryOptions{Options: &spannerpb.ExecuteSqlRequest_QueryOptions{
				OptimizerVersion:           planReportPinnedOptimizerValue(optimizer.Requested.Version),
				OptimizerStatisticsPackage: planReportPinnedOptimizerValue(optimizer.Requested.StatisticsPackage),
			}},
		}))
	}
	return spanemuboost.OpenClients(ctx, runtime, options...)
}

func planReportResolvedOptimizer(optimizer planReportOptimizerEnvironment) planReportOptimizer {
	return planReportOptimizer{
		Requested: planReportOptimizerEnvironment{
			Version:           planReportOptimizerValueOrNotPinned(optimizer.Version),
			StatisticsPackage: planReportOptimizerValueOrNotPinned(optimizer.StatisticsPackage),
		},
		Effective: planReportOptimizerEffective{
			Version:           "not_recorded",
			StatisticsPackage: "not_recorded",
		},
	}
}

func planReportBackendIdentityFromFlags(backend, version, imageDigest string) (planReportBackendIdentity, error) {
	version = strings.TrimSpace(version)
	imageDigest = strings.TrimSpace(imageDigest)
	if version == "" && imageDigest == "" {
		return planReportBackendIdentity{
			Kind:        backend,
			Version:     "not_recorded",
			ImageDigest: "not_recorded",
			Source:      "spanemuboost",
		}, nil
	}
	if imageDigest != "" {
		if err := validatePlanReportImageDigest(imageDigest); err != nil {
			return planReportBackendIdentity{}, err
		}
	}
	return planReportBackendIdentity{
		Kind:        backend,
		Version:     planReportValueOrNotRecorded(version),
		ImageDigest: planReportValueOrNotRecorded(imageDigest),
		Source:      "manual",
	}, nil
}

func validatePlanReportImageDigest(value string) error {
	const prefix = "sha256:"
	if !strings.HasPrefix(value, prefix) {
		return fmt.Errorf("--backend-image-digest must use sha256:<64 lowercase hex chars> format")
	}
	digest := strings.TrimPrefix(value, prefix)
	if len(digest) != 64 {
		return fmt.Errorf("--backend-image-digest must contain 64 lowercase hex characters after %s", prefix)
	}
	decoded, err := hex.DecodeString(digest)
	if err != nil || hex.EncodeToString(decoded) != digest {
		return fmt.Errorf("--backend-image-digest must contain 64 lowercase hex characters after %s", prefix)
	}
	return nil
}

func planReportValueOrNotRecorded(value string) string {
	if value == "" {
		return "not_recorded"
	}
	return value
}

func planReportOptimizerValueOrNotPinned(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "not_pinned"
	}
	return value
}

func planReportPinnedOptimizerValue(value string) string {
	if value == "not_pinned" {
		return ""
	}
	return value
}

func planReportOptimizerEnvironmentPinned(optimizer planReportOptimizerEnvironment) bool {
	return optimizer.Version != "not_pinned" && optimizer.StatisticsPackage != "not_pinned"
}

func planReportDDLs(schema querygen.QueryCodegenSchema, baseDir string) ([]string, error) {
	if strings.TrimSpace(schema.DDL) == "" {
		return nil, nil
	}
	path := resolveOutputPath(baseDir, schema.DDL)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ddls, err := memefish.ParseDDLs(path, string(data))
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(ddls))
	for _, ddl := range ddls {
		out = append(out, ddl.SQL())
	}
	return out, nil
}

func planReportDDLDigest(schema querygen.QueryCodegenSchema, baseDir string) (string, error) {
	if strings.TrimSpace(schema.DDL) == "" {
		return planReportDigest(""), nil
	}
	path := resolveOutputPath(baseDir, schema.DDL)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return planReportDigest(string(data)), nil
}

func queryCodegenSchemasByName(schemas []querygen.QueryCodegenSchema) map[string]querygen.QueryCodegenSchema {
	out := make(map[string]querygen.QueryCodegenSchema, len(schemas))
	for _, schema := range schemas {
		out[schema.Name] = schema
	}
	return out
}

func planReportTargetID(queryName, scope string) string {
	switch planContractScope(scope) {
	case "external_query.inner":
		return "query/" + queryName + "#inner"
	default:
		return "query/" + queryName
	}
}

func planReportQueryTargetID(query planReportQuery) string {
	if strings.TrimSpace(query.TargetID) != "" {
		return query.TargetID
	}
	return planReportTargetID(query.Name, query.Scope)
}

func planReportOperators(plan *spannerpb.QueryPlan) []planReportOperator {
	if plan == nil {
		return nil
	}
	operatorContexts := planReportOperatorContexts(plan)
	childrenByIndex := planReportChildrenByIndex(plan)
	familyByIndex := make(map[int32]string, len(plan.GetPlanNodes()))
	for _, node := range plan.GetPlanNodes() {
		familyByIndex[node.GetIndex()] = planReportNodeOperatorFamily(node, operatorContexts[node.GetIndex()])
	}
	out := make([]planReportOperator, 0, len(plan.GetPlanNodes()))
	for _, node := range plan.GetPlanNodes() {
		childIndexes := planReportOperatorChildIndexes(node)
		descendantIndexes := planReportDescendantIndexes(node.GetIndex(), childrenByIndex)
		subtreeFamilyCounts := planReportSubtreeFamilyCounts(node.GetIndex(), descendantIndexes, familyByIndex)
		out = append(out, planReportOperator{
			Index:               node.GetIndex(),
			DisplayName:         node.GetDisplayName(),
			Family:              familyByIndex[node.GetIndex()],
			ExecutionMethod:     planReportOperatorMetadataString(node, "execution_method"),
			IteratorType:        planReportOperatorMetadataString(node, "iterator_type"),
			ScanMethod:          planReportOperatorMetadataString(node, "scan_method"),
			ScanFormat:          planReportOperatorMetadataString(node, "scan_format"),
			ScanType:            planReportOperatorMetadataString(node, "scan_type"),
			ScanTarget:          planReportNodeMetadataRawString(node, "scan_target"),
			SeekableKeySize:     planReportNodeMetadataRawString(node, "seekable_key_size"),
			JoinType:            planReportOperatorMetadataString(node, "join_type"),
			JoinConfiguration:   planReportOperatorMetadataString(node, "join_configuration"),
			CallType:            planReportOperatorMetadataString(node, "call_type"),
			DistributionTable:   planReportNodeMetadataRawString(node, "distribution_table"),
			SubqueryClusterNode: planReportNodeMetadataRawString(node, "subquery_cluster_node"),
			SpoolName:           planReportNodeMetadataRawString(node, "spool_name"),
			FullScan:            planReportNodeMetadataBool(node, "Full scan"),
			ChildIndexes:        childIndexes,
			DescendantIndexes:   descendantIndexes,
			SubtreeFamilyCounts: subtreeFamilyCounts,
		})
	}
	return out
}

func planReportOperatorEdges(plan *spannerpb.QueryPlan) []planReportOperatorEdge {
	if plan == nil {
		return nil
	}
	edges := make([]planReportOperatorEdge, 0)
	for _, node := range plan.GetPlanNodes() {
		for _, link := range node.GetChildLinks() {
			edges = append(edges, planReportOperatorEdge{
				ParentIndex: node.GetIndex(),
				ChildIndex:  link.GetChildIndex(),
				Type:        link.GetType(),
				Variable:    link.GetVariable(),
			})
		}
	}
	return edges
}

func planReportChildrenByIndex(plan *spannerpb.QueryPlan) map[int32][]int32 {
	childrenByIndex := make(map[int32][]int32, len(plan.GetPlanNodes()))
	for _, node := range plan.GetPlanNodes() {
		childrenByIndex[node.GetIndex()] = planReportOperatorChildIndexes(node)
	}
	return childrenByIndex
}

func planReportOperatorChildIndexes(node *spannerpb.PlanNode) []int32 {
	childIndexes := make([]int32, 0, len(node.GetChildLinks()))
	for _, link := range node.GetChildLinks() {
		childIndexes = append(childIndexes, link.GetChildIndex())
	}
	return childIndexes
}

func planReportDescendantIndexes(root int32, childrenByIndex map[int32][]int32) []int32 {
	seen := map[int32]bool{root: true}
	out := make([]int32, 0)
	var walk func(int32)
	walk = func(parent int32) {
		for _, child := range childrenByIndex[parent] {
			if seen[child] {
				continue
			}
			seen[child] = true
			out = append(out, child)
			walk(child)
		}
	}
	walk(root)
	return out
}

func planReportSubtreeFamilyCounts(root int32, descendantIndexes []int32, familyByIndex map[int32]string) map[string]int {
	counts := planReportZeroOperatorFamilyCounts()
	add := func(index int32) {
		if family := familyByIndex[index]; family != "" {
			counts[family]++
		}
	}
	add(root)
	for _, index := range descendantIndexes {
		add(index)
	}
	plancontract.AddDerivedOperatorFamilyCounts(counts)
	return counts
}

func planReportClassificationWarnings(operators []planReportOperator) []planReportDiagnostic {
	var warnings []planReportDiagnostic
	for _, operator := range operators {
		if operator.Family == "unknown" {
			warnings = append(warnings, planReportDiagnostic{
				ID:      "operator_family_unknown",
				Message: fmt.Sprintf("PlanNode %d with display name %q could not be classified into a known operator family.", operator.Index, operator.DisplayName),
			})
			continue
		}
		if operator.Family == "join" {
			warnings = append(warnings, planReportDiagnostic{
				ID:      "join_family_unknown",
				Message: fmt.Sprintf("Join-like PlanNode %d with display name %q could not be classified into a specific join family.", operator.Index, operator.DisplayName),
			})
			continue
		}
		if planReportNormalizeOperatorName(operator.DisplayName) != "aggregate" || operator.Family != "aggregate" {
			continue
		}
		message := fmt.Sprintf("Aggregate PlanNode %d could not be classified as hash_aggregate or stream_aggregate.", operator.Index)
		if operator.IteratorType == "" {
			message += " iterator_type metadata is missing."
		} else {
			message += fmt.Sprintf(" iterator_type metadata is %q.", operator.IteratorType)
		}
		warnings = append(warnings, planReportDiagnostic{
			ID:      "aggregate_iterator_type_unknown",
			Message: message,
		})
	}
	return warnings
}

func planReportOperatorFamilies(operators []planReportOperator) []string {
	seen := map[string]bool{}
	for _, operator := range operators {
		if operator.Family != "" {
			seen[operator.Family] = true
		}
	}
	out := make([]string, 0, len(seen))
	for family := range seen {
		out = append(out, family)
	}
	sort.Strings(out)
	return out
}

func planReportOperatorFamilyCounts(operators []planReportOperator) map[string]int {
	return plancontract.OperatorFamilyCounts(operators)
}

func planReportOperatorFamilyCountsOrEmpty(counts map[string]int) map[string]int {
	return plancontract.OperatorFamilyCountsOrEmpty(counts)
}

func planReportZeroOperatorFamilyCounts() map[string]int {
	return plancontract.ZeroOperatorFamilyCounts()
}

func planReportKnownOperatorFamilies() []string {
	return plancontract.KnownOperatorFamilies()
}

func planReportConcreteOperatorFamilies() []string {
	return plancontract.ConcreteOperatorFamilies()
}

func planReportKnownOperatorFamily(family string) bool {
	return plancontract.KnownOperatorFamily(family)
}

func planContractPredefinedNames() []string {
	return plancontract.PredefinedNames()
}

func planReportOperatorTreeDigest(plan *spannerpb.QueryPlan) string {
	if plan == nil {
		return planReportDigest("")
	}
	operatorContexts := planReportOperatorContexts(plan)
	var b strings.Builder
	for _, node := range plan.GetPlanNodes() {
		fmt.Fprintf(&b, "%d|%s|%s|%s|%s|%s|%s|%t",
			node.GetIndex(),
			planReportNodeOperatorFamily(node, operatorContexts[node.GetIndex()]),
			node.GetDisplayName(),
			planReportOperatorMetadataString(node, "execution_method"),
			planReportOperatorMetadataString(node, "iterator_type"),
			planReportOperatorMetadataString(node, "scan_method"),
			planReportOperatorMetadataString(node, "scan_type"),
			planReportNodeMetadataBool(node, "Full scan"),
		)
		for _, link := range node.GetChildLinks() {
			fmt.Fprintf(&b, "|%s:%d", link.GetType(), link.GetChildIndex())
		}
		b.WriteByte('\n')
	}
	return planReportDigest(b.String())
}

// planReportNodeOperatorFamily classifies one PlanNode into a normalized
// operator family. Scalar-kind PlanNodes (Reference, Function, Constant,
// Parameter, and similar expression nodes) are not relational operators, so
// they map to the dedicated "scalar" family instead of competing with the
// "unknown" fallback reserved for unclassified relational operators.
func planReportNodeOperatorFamily(node *spannerpb.PlanNode, context planReportOperatorContext) string {
	if node.GetKind() == spannerpb.PlanNode_SCALAR {
		return "scalar"
	}
	return planReportOperatorFamilyWithContext(node, context)
}

type planReportOperatorContext struct {
	PushBroadcastInternalHashJoin    bool
	DistributedCrossApplyInternal    bool
	DistributedSemiApplyInternal     bool
	DistributedAntiSemiApplyInternal bool
}

func planReportOperatorFamilyWithContext(node *spannerpb.PlanNode, context planReportOperatorContext) string {
	displayName := planReportNormalizeOperatorName(node.GetDisplayName())
	if context.PushBroadcastInternalHashJoin && displayName == "hash join" {
		return "push_broadcast_hash_join_internal_hash_join"
	}
	if context.DistributedCrossApplyInternal && (displayName == "cross apply" || displayName == "outer apply") {
		return "distributed_cross_apply_internal_apply"
	}
	if context.DistributedSemiApplyInternal && (displayName == "semi apply" || displayName == "anti-semi apply" || displayName == "anti semi apply") {
		return "distributed_semi_apply_internal_apply"
	}
	if context.DistributedAntiSemiApplyInternal && (displayName == "semi apply" || displayName == "anti-semi apply" || displayName == "anti semi apply") {
		return "distributed_anti_semi_apply_internal_apply"
	}
	return planReportOperatorFamily(node)
}

func planReportOperatorFamily(node *spannerpb.PlanNode) string {
	if node == nil {
		return "unknown"
	}
	displayName := planReportNormalizeOperatorName(node.GetDisplayName())
	switch displayName {
	case "apply mutations":
		return "apply_mutations"
	case "sort", "sort limit", "local sort", "local sort limit":
		return "full_sort"
	case "minor sort", "minor sort limit", "local minor sort", "local minor sort limit":
		return "minor_sort"
	case "global limit", "limit", "local limit":
		return "limit"
	case "hash join":
		return "hash_join"
	case "push broadcast hash join", "push broadcast hash join outer apply", "push broadcast hash join semi apply", "push broadcast hash join anti semi apply":
		return "push_broadcast_hash_join"
	case "anti-semi apply", "anti semi apply":
		return "anti_semi_apply"
	case "semi apply":
		return "semi_apply"
	case "apply join", "cross apply", "outer apply":
		return "apply_join"
	case "distributed cross apply", "distributed outer apply", "distributed apply":
		return "distributed_cross_apply"
	case "distributed semi apply":
		return "distributed_semi_apply"
	case "distributed anti semi apply":
		return "distributed_anti_semi_apply"
	case "merge join":
		return "merge_join"
	case "aggregate":
		return planReportAggregateFamily(node)
	case "array subquery":
		return "array_subquery"
	case "array unnest":
		return "array_unnest"
	case "bloomfilterbuild":
		return "bloom_filter_build"
	case "changestream tvf":
		return "change_stream_tvf"
	case "compute":
		return "compute"
	case "compute struct":
		return "compute_struct"
	case "create batch":
		return "create_batch"
	case "datablocktorow":
		return "data_block_to_row"
	case "empty relation":
		return "empty_relation"
	case "filter":
		return "filter"
	case "keyrangeaccumulator":
		return "key_range_accumulator"
	case "minibatchassign":
		return "mini_batch_assign"
	case "minibatchkeyorder":
		return "mini_batch_key_order"
	case "random id assign":
		return "random_id_assign"
	case "recursive spool scan":
		return "recursive_spool_scan"
	case "rowcount":
		return "row_count"
	case "rowtodatablock":
		return "row_to_data_block"
	case "scalar subquery":
		return "scalar_subquery"
	case "search predicate":
		return "search_predicate"
	case "spoolbuild":
		return "spool_build"
	case "spoolscan":
		return "spool_scan"
	case "scan", "table scan", "index scan":
		return "scan"
	case "filter scan":
		return "filter_scan"
	case "serialize result":
		return "serialize_result"
	case "distributed union":
		if planReportNodeMetadataBool(node, "preserve_subquery_order") {
			return "distributed_merge_union"
		}
		return "distributed_union"
	case "distributed merge union":
		return "distributed_merge_union"
	case "union all":
		return "union_all"
	case "union input":
		return "union_input"
	case "recursive union":
		return "recursive_union"
	case "tvf":
		if planReportNormalizeOperatorName(planReportNodeMetadataRawString(node, "name")) == "search query conversion" {
			return "search_query_conversion_tvf"
		}
		return "unknown"
	case "unit relation":
		return "unit_relation"
	case "verifydeterminism":
		return "verify_determinism"
	default:
		description := planReportNormalizeOperatorName(node.GetShortRepresentation().GetDescription())
		switch {
		case strings.Contains(displayName, "hash aggregate"):
			return "hash_aggregate"
		case strings.Contains(displayName, "stream aggregate"):
			return "stream_aggregate"
		case strings.Contains(displayName, "push broadcast hash join"):
			return "push_broadcast_hash_join"
		case strings.Contains(displayName, "hash join"):
			return "hash_join"
		case strings.Contains(displayName, "apply"):
			return "apply_join"
		case strings.Contains(displayName, "merge join"):
			return "merge_join"
		case strings.Contains(displayName, "sort"):
			if strings.Contains(displayName, "minor sort") {
				return "minor_sort"
			}
			return "full_sort"
		case strings.Contains(displayName, "scan"):
			return "scan"
		case strings.Contains(displayName, "join"):
			return "join"
		case strings.Contains(displayName, "aggregate"):
			return "aggregate"
		case strings.Contains(description, "hash aggregate"):
			return "hash_aggregate"
		case strings.Contains(description, "stream aggregate"):
			return "stream_aggregate"
		default:
			return "unknown"
		}
	}
}

func planReportOperatorContexts(plan *spannerpb.QueryPlan) map[int32]planReportOperatorContext {
	contexts := map[int32]planReportOperatorContext{}
	if plan == nil {
		return contexts
	}
	nodesByIndex := make(map[int32]*spannerpb.PlanNode, len(plan.GetPlanNodes()))
	for _, node := range plan.GetPlanNodes() {
		nodesByIndex[node.GetIndex()] = node
	}
	for _, node := range plan.GetPlanNodes() {
		switch planReportNormalizeOperatorName(node.GetDisplayName()) {
		case "push broadcast hash join", "push broadcast hash join outer apply", "push broadcast hash join semi apply", "push broadcast hash join anti semi apply":
			for _, link := range node.GetChildLinks() {
				if !strings.EqualFold(strings.TrimSpace(link.GetType()), "Map") {
					continue
				}
				index, ok := planReportFindWrapperInternalOperator(nodesByIndex, link.GetChildIndex(), map[string]bool{"hash join": true})
				if !ok {
					continue
				}
				candidate := nodesByIndex[index]
				if !planReportOperatorConsumesBatchScan(nodesByIndex, candidate) {
					continue
				}
				context := contexts[index]
				context.PushBroadcastInternalHashJoin = true
				contexts[index] = context
			}
		case "distributed cross apply", "distributed outer apply", "distributed apply":
			for _, link := range node.GetChildLinks() {
				if !strings.EqualFold(strings.TrimSpace(link.GetType()), "Map") {
					continue
				}
				index, ok := planReportFindWrapperInternalOperator(nodesByIndex, link.GetChildIndex(), map[string]bool{
					"cross apply": true,
					"outer apply": true,
				})
				if !ok {
					continue
				}
				candidate := nodesByIndex[index]
				if !planReportOperatorConsumesBatchScan(nodesByIndex, candidate) {
					continue
				}
				context := contexts[index]
				context.DistributedCrossApplyInternal = true
				contexts[index] = context
			}
		case "distributed semi apply":
			for _, link := range node.GetChildLinks() {
				if !strings.EqualFold(strings.TrimSpace(link.GetType()), "Map") {
					continue
				}
				index, ok := planReportFindWrapperInternalOperator(nodesByIndex, link.GetChildIndex(), map[string]bool{
					"semi apply":      true,
					"anti-semi apply": true,
					"anti semi apply": true,
				})
				if !ok {
					continue
				}
				candidate := nodesByIndex[index]
				if !planReportOperatorConsumesBatchScan(nodesByIndex, candidate) {
					continue
				}
				context := contexts[index]
				context.DistributedSemiApplyInternal = true
				contexts[index] = context
			}
		case "distributed anti semi apply":
			for _, link := range node.GetChildLinks() {
				if !strings.EqualFold(strings.TrimSpace(link.GetType()), "Map") {
					continue
				}
				index, ok := planReportFindWrapperInternalOperator(nodesByIndex, link.GetChildIndex(), map[string]bool{
					"semi apply":      true,
					"anti-semi apply": true,
					"anti semi apply": true,
				})
				if !ok {
					continue
				}
				candidate := nodesByIndex[index]
				if !planReportOperatorConsumesBatchScan(nodesByIndex, candidate) {
					continue
				}
				context := contexts[index]
				context.DistributedAntiSemiApplyInternal = true
				contexts[index] = context
			}
		}
	}
	return contexts
}

func planReportFindWrapperInternalOperator(nodesByIndex map[int32]*spannerpb.PlanNode, start int32, targetNames map[string]bool) (int32, bool) {
	seen := map[int32]bool{}
	for {
		if seen[start] {
			return 0, false
		}
		seen[start] = true
		node, ok := nodesByIndex[start]
		if !ok {
			return 0, false
		}
		if targetNames[planReportNormalizeOperatorName(node.GetDisplayName())] {
			return start, true
		}
		children := planReportRelationalChildIndexes(nodesByIndex, node)
		if len(children) != 1 {
			return 0, false
		}
		start = children[0]
	}
}

func planReportRelationalChildIndexes(nodesByIndex map[int32]*spannerpb.PlanNode, node *spannerpb.PlanNode) []int32 {
	var out []int32
	for _, link := range node.GetChildLinks() {
		child, ok := nodesByIndex[link.GetChildIndex()]
		if !ok {
			continue
		}
		if planReportIsRelationalPlanNode(child) {
			out = append(out, child.GetIndex())
		}
	}
	return out
}

func planReportIsRelationalPlanNode(node *spannerpb.PlanNode) bool {
	switch planReportNormalizeOperatorName(node.GetDisplayName()) {
	case "aggregate",
		"anti-semi apply",
		"array unnest",
		"apply join",
		"compute",
		"compute struct",
		"create batch",
		"cross apply",
		"datablocktorow",
		"distributed apply",
		"distributed cross apply",
		"distributed merge union",
		"distributed outer apply",
		"distributed semi apply",
		"distributed anti semi apply",
		"distributed union",
		"empty relation",
		"filter",
		"filter scan",
		"hash join",
		"index scan",
		"limit",
		"merge join",
		"outer apply",
		"push broadcast hash join",
		"push broadcast hash join outer apply",
		"push broadcast hash join semi apply",
		"push broadcast hash join anti semi apply",
		"recursive spool scan",
		"random id assign",
		"recursive union",
		"rowtodatablock",
		"scan",
		"serialize result",
		"semi apply",
		"sort",
		"sort limit",
		"spoolbuild",
		"spoolscan",
		"table scan",
		"union all",
		"unit relation":
		return true
	default:
		return false
	}
}

func planReportOperatorConsumesBatchScan(nodesByIndex map[int32]*spannerpb.PlanNode, node *spannerpb.PlanNode) bool {
	for _, link := range node.GetChildLinks() {
		if planReportSubtreeHasBatchScan(nodesByIndex, link.GetChildIndex(), map[int32]bool{}) {
			return true
		}
	}
	return false
}

func planReportSubtreeHasBatchScan(nodesByIndex map[int32]*spannerpb.PlanNode, index int32, seen map[int32]bool) bool {
	if seen[index] {
		return false
	}
	seen[index] = true
	node, ok := nodesByIndex[index]
	if !ok {
		return false
	}
	if planReportOperatorMetadataString(node, "scan_type") == "batch_scan" {
		return true
	}
	if planReportOperatorMetadataString(node, "scan_method") == "batch" &&
		strings.HasPrefix(planReportNodeMetadataRawString(node, "scan_target"), "$") {
		return true
	}
	for _, link := range node.GetChildLinks() {
		if planReportSubtreeHasBatchScan(nodesByIndex, link.GetChildIndex(), seen) {
			return true
		}
	}
	return false
}

func planReportAggregateFamily(node *spannerpb.PlanNode) string {
	switch planReportNodeMetadataString(node, "iterator_type") {
	case "hash":
		return "hash_aggregate"
	case "stream":
		return "stream_aggregate"
	default:
		return "aggregate"
	}
}

func planReportOperatorMetadataString(node *spannerpb.PlanNode, key string) string {
	value := planReportNodeMetadataString(node, key)
	switch key {
	case "scan_type":
		switch value {
		case "tablescan":
			return "table_scan"
		case "indexscan":
			return "index_scan"
		case "batchscan":
			return "batch_scan"
		case "searchindexscan":
			return "search_index_scan"
		}
	case "join_configuration":
		return strings.ToLower(strings.ReplaceAll(value, "-", "_"))
	}
	return value
}

func planReportNodeMetadataString(node *spannerpb.PlanNode, key string) string {
	return strings.ToLower(planReportNodeMetadataRawString(node, key))
}

func planReportNodeMetadataRawString(node *spannerpb.PlanNode, key string) string {
	if node == nil || node.GetMetadata() == nil {
		return ""
	}
	value, ok := node.GetMetadata().AsMap()[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func planReportNodeMetadataBool(node *spannerpb.PlanNode, key string) bool {
	switch planReportNodeMetadataString(node, key) {
	case "true":
		return true
	default:
		return false
	}
}

func planReportNormalizeOperatorName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}

func queryCodegenQueriesByName(queries []querygen.QueryCodegenQuery) map[string]querygen.QueryCodegenQuery {
	out := make(map[string]querygen.QueryCodegenQuery, len(queries))
	for _, query := range queries {
		out[query.Name] = query
	}
	return out
}

func planReportScopedParams(params []querygen.QueryCodegenParam, scope string) []querygen.QueryCodegenParam {
	var out []querygen.QueryCodegenParam
	for _, param := range params {
		paramScope := strings.ToLower(strings.TrimSpace(param.Scope))
		if paramScope == "" || paramScope == scope {
			out = append(out, param)
		}
	}
	return out
}

func planReportParamValues(params []querygen.QueryCodegenParam) (map[string]interface{}, error) {
	if len(params) == 0 {
		return nil, nil
	}
	values := make(map[string]interface{}, len(params))
	for _, param := range params {
		spec, err := spanalyzer.ParseTypeSpec("param", param.Type)
		if err != nil {
			return nil, fmt.Errorf("param %s: %w", param.Name, err)
		}
		value, err := planReportParamValue(spec)
		if err != nil {
			return nil, fmt.Errorf("param %s type %s: %w", param.Name, param.Type, err)
		}
		values[param.Name] = value
	}
	return values, nil
}

func planReportParamValue(spec *spanalyzer.TypeSpec) (interface{}, error) {
	if spec == nil {
		return nil, fmt.Errorf("nil type")
	}
	switch spec.Code {
	case spannerpb.TypeCode_BOOL:
		return true, nil
	case spannerpb.TypeCode_INT64:
		return int64(1), nil
	case spannerpb.TypeCode_FLOAT32:
		return float32(math.Pi), nil
	case spannerpb.TypeCode_FLOAT64:
		return math.Pi, nil
	case spannerpb.TypeCode_NUMERIC:
		return big.NewRat(1, 1), nil
	case spannerpb.TypeCode_STRING:
		return "value", nil
	case spannerpb.TypeCode_BYTES:
		return []byte("value"), nil
	case spannerpb.TypeCode_DATE:
		return civil.Date{Year: 2026, Month: time.May, Day: 6}, nil
	case spannerpb.TypeCode_TIMESTAMP:
		return time.Unix(0, 0).UTC(), nil
	case spannerpb.TypeCode_ARRAY:
		return planReportArrayParamValue(spec.ArrayElement)
	default:
		return nil, fmt.Errorf("unsupported plan-report parameter type %s", spec.Code)
	}
}

func planReportArrayParamValue(element *spanalyzer.TypeSpec) (interface{}, error) {
	if element == nil {
		return nil, fmt.Errorf("nil array element type")
	}
	switch element.Code {
	case spannerpb.TypeCode_BOOL:
		return []bool{true}, nil
	case spannerpb.TypeCode_INT64:
		return []int64{1}, nil
	case spannerpb.TypeCode_FLOAT32:
		return []float32{float32(math.Pi)}, nil
	case spannerpb.TypeCode_FLOAT64:
		return []float64{math.Pi}, nil
	case spannerpb.TypeCode_STRING:
		return []string{"value"}, nil
	case spannerpb.TypeCode_BYTES:
		return [][]byte{[]byte("value")}, nil
	case spannerpb.TypeCode_DATE:
		return []civil.Date{{Year: 2026, Month: time.May, Day: 6}}, nil
	case spannerpb.TypeCode_TIMESTAMP:
		return []time.Time{time.Unix(0, 0).UTC()}, nil
	default:
		return nil, fmt.Errorf("unsupported array element type %s", element.Code)
	}
}

func planReportDigest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func planReportFileDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return planReportDigest(string(data)), nil
}

func planReportStableInputPath(baseDir, resolvedPath, requestedPath string, stable bool) string {
	if stable {
		return ""
	}
	requestedPath = strings.TrimSpace(requestedPath)
	if requestedPath == "" {
		return ""
	}
	if !filepath.IsAbs(requestedPath) {
		return filepath.ToSlash(requestedPath)
	}
	relative, err := filepath.Rel(baseDir, resolvedPath)
	if err != nil {
		return ""
	}
	if relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return ""
	}
	return filepath.ToSlash(relative)
}

type planContractsFile = plancontract.File
type planContract = plancontract.Contract
type planContractPredicate = plancontract.Predicate
type planContractEvaluationSummary = plancontract.EvaluationSummary
type planContractEvaluation = plancontract.Evaluation
type planContractRuleResult = plancontract.RuleResult

func readPlanContracts(path string) (planContractsFile, error) {
	return plancontract.ReadFile(path)
}

func applyPlanContracts(report *planReport, contracts planContractsFile) error {
	if report == nil {
		return fmt.Errorf("nil plan report")
	}
	result, err := plancontract.Evaluate(planContractReport(*report), contracts)
	if err != nil {
		return err
	}
	report.ContractFileVersion = result.FileVersion
	report.ContractEvaluator = result.EvaluatorVersion
	if report.ContractEvalMode == "" || report.ContractEvalMode == planContractEvaluationModeNone {
		report.ContractEvalMode = planContractEvaluationModeReportOnly
	}
	report.ContractEvaluations = result.Evaluations
	report.ContractSummary = &result.Summary
	return nil
}

func planContractReport(report planReport) plancontract.Report {
	queries := make([]plancontract.Query, 0, len(report.Queries))
	for _, query := range report.Queries {
		queries = append(queries, plancontract.Query{
			TargetID:             planReportQueryTargetID(query),
			Name:                 query.Name,
			Catalog:              query.Catalog,
			Scope:                query.Scope,
			Kind:                 query.Kind,
			Status:               query.Status,
			SQLSHA256:            query.SQLSHA256,
			DDLSHA256:            query.DDLSHA256,
			OperatorTreeSHA256:   query.OperatorTreeSHA256,
			OperatorFamilies:     query.OperatorFamilies,
			OperatorFamilyCounts: query.OperatorFamilyCounts,
			NormalizedOperators:  query.NormalizedOperators,
			OperatorEdges:        query.OperatorEdges,
			Error:                query.Error,
			OptimizerNotPinned:   query.OptimizerNotPinned,
			PlanEnvironmentNotes: query.PlanEnvironmentNotes,
			ClassificationNotes:  query.ClassificationNotes,
			RawPlan:              query.RawPlan,
		})
	}
	return plancontract.Report{
		Backend: report.Backend,
		BackendIdentity: plancontract.BackendIdentity{
			Kind:        report.BackendIdentity.Kind,
			Version:     report.BackendIdentity.Version,
			ImageDigest: report.BackendIdentity.ImageDigest,
			Source:      report.BackendIdentity.Source,
		},
		Optimizer: report.Optimizer,
		Queries:   queries,
	}
}

func planContractScope(scope string) string {
	return plancontract.Scope(scope)
}

func addPlanReportContractCheckWarnings(report *planReport) {
	if report == nil || report.ContractSummary == nil {
		return
	}
	plancontract.AddCheckWarnings(report.ContractSummary, report.ContractEvaluations)
}

func planReportOptimizerPinningWarnings(report planReport) []string {
	return plancontract.OptimizerPinningWarnings(planContractReport(report))
}

func planReportContractViolationCount(report planReport) int {
	return plancontract.ViolationCount(report.ContractEvaluations)
}

func planReportContractCheckFailureCount(report planReport) int {
	return plancontract.CheckFailureCount(report.ContractEvaluations)
}

func writePlanReport(stdout io.Writer, output string, report planReport) error {
	if err := validatePlanReportInvariants(report); err != nil {
		return err
	}
	switch output {
	case "markdown", "md":
		return writePlanReportMarkdown(stdout, report)
	case "yaml":
		data, err := marshalYAMLViaJSON(report)
		if err != nil {
			return err
		}
		_, err = stdout.Write(data)
		return err
	case "json":
		data, err := marshalIndentedJSON(report)
		if err != nil {
			return err
		}
		_, err = stdout.Write(data)
		return err
	default:
		return fmt.Errorf("unsupported --output %q", output)
	}
}

func validatePlanReportInvariants(report planReport) error {
	if got, want := report.TargetSummary.Planned+report.TargetSummary.Errors+report.TargetSummary.Skipped, report.TargetSummary.IncludedCount; got != want {
		return fmt.Errorf("plan-report invariant violation: target_summary.planned + target_summary.errors + target_summary.skipped = %d, want target_summary.included_count %d", got, want)
	}
	if err := validatePlanReportBackendIdentity(report.BackendIdentity); err != nil {
		return err
	}
	switch report.ContractEvalMode {
	case planContractEvaluationModeNone:
		if len(report.ContractEvaluations) > 0 || report.ContractSummary != nil {
			return fmt.Errorf("plan-report invariant violation: contract_evaluation_mode none must not include contract evaluation fields")
		}
	case planContractEvaluationModeReportOnly, planContractEvaluationModeCheck:
		if len(report.ContractEvaluations) == 0 {
			return fmt.Errorf("plan-report invariant violation: contract_evaluation_mode %s requires at least one contract_evaluations entry", report.ContractEvalMode)
		}
	default:
		return fmt.Errorf("plan-report invariant violation: unsupported contract_evaluation_mode %q", report.ContractEvalMode)
	}
	for _, query := range report.Queries {
		if query.Status != "ok" {
			continue
		}
		if err := validatePlanReportQueryTopology(query); err != nil {
			return err
		}
	}
	if len(report.ContractEvaluations) > 0 && report.ContractSummary == nil {
		return fmt.Errorf("plan-report invariant violation: contract_evaluations present but contract_summary is absent")
	}
	if report.ContractSummary == nil {
		return nil
	}
	summary := report.ContractSummary
	if got, want := summary.Contracts, len(report.ContractEvaluations); got != want {
		return fmt.Errorf("plan-report invariant violation: contract_summary.contracts = %d, want len(contract_evaluations) %d", got, want)
	}
	var passed, failed, notEvaluated int
	for _, evaluation := range report.ContractEvaluations {
		switch evaluation.Status {
		case planContractStatusPass:
			passed++
		case planContractStatusFail:
			failed++
		case planContractStatusNotEvaluated:
			notEvaluated++
		}
		if err := validatePlanReportContractResultIndexes(report, evaluation); err != nil {
			return err
		}
	}
	if summary.Passed != passed {
		return fmt.Errorf("plan-report invariant violation: contract_summary.passed = %d, want %d", summary.Passed, passed)
	}
	if summary.Failed != failed {
		return fmt.Errorf("plan-report invariant violation: contract_summary.failed = %d, want %d", summary.Failed, failed)
	}
	if summary.NotEvaluated != notEvaluated {
		return fmt.Errorf("plan-report invariant violation: contract_summary.not_evaluated = %d, want %d", summary.NotEvaluated, notEvaluated)
	}
	wantStatus := planContractStatusPass
	if failed > 0 || notEvaluated > 0 {
		wantStatus = planContractStatusFail
	}
	if summary.Status != wantStatus {
		return fmt.Errorf("plan-report invariant violation: contract_summary.status = %q, want %q", summary.Status, wantStatus)
	}
	return nil
}

func validatePlanReportBackendIdentity(identity planReportBackendIdentity) error {
	versionNotRecorded := strings.EqualFold(identity.Version, "not_recorded")
	digestNotRecorded := strings.EqualFold(identity.ImageDigest, "not_recorded")
	switch identity.Source {
	case "spanemuboost":
		return nil
	case "manual":
		if versionNotRecorded && digestNotRecorded {
			return fmt.Errorf("plan-report invariant violation: backend_identity.source manual requires version or image_digest to be recorded")
		}
	case "not_recorded":
		if !versionNotRecorded || !digestNotRecorded {
			return fmt.Errorf("plan-report invariant violation: backend_identity.source not_recorded requires version and image_digest to be not_recorded")
		}
	default:
		return fmt.Errorf("plan-report invariant violation: unsupported backend_identity.source %q", identity.Source)
	}
	return nil
}

func validatePlanReportContractResultIndexes(report planReport, evaluation planContractEvaluation) error {
	query, ok := planReportFindQueryByTargetID(report.Queries, evaluation.TargetID)
	if !ok || query.Status != "ok" {
		return nil
	}
	operatorsByIndex := make(map[int32]bool, len(query.NormalizedOperators))
	for _, operator := range query.NormalizedOperators {
		operatorsByIndex[operator.Index] = true
	}
	for _, result := range evaluation.Results {
		if result.MatchedOperatorIndexes == nil {
			continue
		}
		for _, index := range *result.MatchedOperatorIndexes {
			if !operatorsByIndex[index] {
				return fmt.Errorf("plan-report invariant violation: contract %s matched_operator_indexes contains missing index %d for target %s", evaluation.Name, index, evaluation.TargetID)
			}
		}
	}
	return nil
}

func planReportFindQueryByTargetID(queries []planReportQuery, targetID string) (planReportQuery, bool) {
	for _, query := range queries {
		if planReportQueryTargetID(query) == targetID {
			return query, true
		}
	}
	return planReportQuery{}, false
}

func validatePlanReportQueryTopology(query planReportQuery) error {
	queryID := planReportQueryTargetID(query)
	operatorsByIndex := make(map[int32]planReportOperator, len(query.NormalizedOperators))
	for _, operator := range query.NormalizedOperators {
		if _, ok := operatorsByIndex[operator.Index]; ok {
			return fmt.Errorf("plan-report invariant violation: query %s normalized_operators index %d is duplicated", queryID, operator.Index)
		}
		operatorsByIndex[operator.Index] = operator
	}
	for _, edge := range query.OperatorEdges {
		if _, ok := operatorsByIndex[edge.ParentIndex]; !ok {
			return fmt.Errorf("plan-report invariant violation: query %s operator_edges parent_index %d is not in normalized_operators", queryID, edge.ParentIndex)
		}
		if _, ok := operatorsByIndex[edge.ChildIndex]; !ok {
			return fmt.Errorf("plan-report invariant violation: query %s operator_edges child_index %d is not in normalized_operators", queryID, edge.ChildIndex)
		}
	}
	for _, operator := range query.NormalizedOperators {
		for _, index := range operator.ChildIndexes {
			if _, ok := operatorsByIndex[index]; !ok {
				return fmt.Errorf("plan-report invariant violation: query %s operator %d child_indexes contains missing index %d", queryID, operator.Index, index)
			}
		}
		for _, index := range operator.DescendantIndexes {
			if _, ok := operatorsByIndex[index]; !ok {
				return fmt.Errorf("plan-report invariant violation: query %s operator %d descendant_indexes contains missing index %d", queryID, operator.Index, index)
			}
		}
		want := planReportExpectedSubtreeFamilyCounts(operator, operatorsByIndex)
		got := planReportOperatorFamilyCountsOrEmpty(operator.SubtreeFamilyCounts)
		if !reflect.DeepEqual(got, want) {
			return fmt.Errorf("plan-report invariant violation: query %s operator %d subtree_family_counts = %v, want %v", queryID, operator.Index, got, want)
		}
	}
	gotCounts := planReportOperatorFamilyCountsOrEmpty(query.OperatorFamilyCounts)
	wantCounts := planReportOperatorFamilyCounts(query.NormalizedOperators)
	if !reflect.DeepEqual(gotCounts, wantCounts) {
		return fmt.Errorf("plan-report invariant violation: query %s operator_family_counts = %v, want %v", queryID, gotCounts, wantCounts)
	}
	return nil
}

func planReportExpectedSubtreeFamilyCounts(operator planReportOperator, operatorsByIndex map[int32]planReportOperator) map[string]int {
	counts := planReportZeroOperatorFamilyCounts()
	add := func(index int32) {
		if subtreeOperator, ok := operatorsByIndex[index]; ok && subtreeOperator.Family != "" {
			counts[subtreeOperator.Family]++
		}
	}
	add(operator.Index)
	for _, index := range operator.DescendantIndexes {
		add(index)
	}
	counts["explicit_sort"] += counts["full_sort"] + counts["minor_sort"]
	return counts
}

func writePlanReportMarkdown(w io.Writer, report planReport) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Spanner Query Plan Report\n\n")
	fmt.Fprintf(&b, "- Report version: `%s`\n", report.ReportVersion)
	fmt.Fprintf(&b, "- Status: `%s`\n", report.Status)
	fmt.Fprintf(&b, "- Backend: `%s`\n", report.Backend)
	if report.Input.ConfigSHA256 != "" {
		fmt.Fprintf(&b, "- Config SHA-256: `%s`\n", report.Input.ConfigSHA256)
	}
	if report.Input.ContractFileSHA256 != "" {
		fmt.Fprintf(&b, "- Contract file SHA-256: `%s`\n", report.Input.ContractFileSHA256)
	}
	if report.PlanSource.API != "" {
		fmt.Fprintf(&b, "- Plan source: `%s` via `%s`\n", report.PlanSource.API, report.PlanSource.RenderTool)
	}
	fmt.Fprintf(&b, "- Format: `%s`\n", strings.ToLower(report.Format))
	fmt.Fprintf(&b, "- Render mode: `%s`\n", strings.ToLower(report.RenderMode))
	fmt.Fprintf(&b, "- Contract evaluation mode: `%s`\n", report.ContractEvalMode)
	fmt.Fprintf(&b, "- Included targets: `%d`\n\n", report.TargetSummary.IncludedCount)
	if len(report.Warnings) > 0 {
		fmt.Fprintf(&b, "## Warnings\n\n")
		for _, warning := range report.Warnings {
			fmt.Fprintf(&b, "- `%s`: %s\n", warning.ID, warning.Message)
		}
		b.WriteByte('\n')
	}
	if len(report.TargetSummary.Excluded) > 0 {
		fmt.Fprintf(&b, "## Excluded Targets\n\n")
		for _, excluded := range report.TargetSummary.Excluded {
			fmt.Fprintf(&b, "- `%s`: `%s`\n", excluded.Query, excluded.Reason)
		}
		b.WriteByte('\n')
	}
	for _, query := range report.Queries {
		fmt.Fprintf(&b, "## %s\n\n", query.Name)
		fmt.Fprintf(&b, "- Target ID: `%s`\n", planReportQueryTargetID(query))
		fmt.Fprintf(&b, "- Catalog: `%s`\n", query.Catalog)
		if query.Scope != "" {
			fmt.Fprintf(&b, "- Scope: `%s`\n", query.Scope)
		}
		fmt.Fprintf(&b, "- Kind: `%s`\n", query.Kind)
		fmt.Fprintf(&b, "- Status: `%s`\n", query.Status)
		fmt.Fprintf(&b, "- SQL SHA-256: `%s`\n", query.SQLSHA256)
		if query.DDLSHA256 != "" {
			fmt.Fprintf(&b, "- DDL SHA-256: `%s`\n", query.DDLSHA256)
		}
		if query.OperatorTreeSHA256 != "" {
			fmt.Fprintf(&b, "- Operator tree SHA-256: `%s`\n", query.OperatorTreeSHA256)
		}
		if len(query.OperatorFamilies) > 0 {
			fmt.Fprintf(&b, "- Operator families: `%s`\n", strings.Join(query.OperatorFamilies, "`, `"))
		}
		if counts := planReportFormatOperatorFamilyCounts(query.OperatorFamilyCounts); counts != "" {
			fmt.Fprintf(&b, "- Operator family counts: `%s`\n", counts)
		}
		if len(query.ClassificationNotes) > 0 {
			for _, warning := range query.ClassificationNotes {
				fmt.Fprintf(&b, "- Classification warning `%s`: %s\n", warning.ID, warning.Message)
			}
		}
		b.WriteByte('\n')
		fmt.Fprintf(&b, "```sql\n%s\n```\n\n", strings.TrimSpace(query.SQL))
		if query.Error != "" {
			fmt.Fprintf(&b, "```text\n%s\n```\n\n", query.Error)
			continue
		}
		fmt.Fprintf(&b, "```text\n%s\n```\n\n", query.Plan)
	}
	if len(report.ContractEvaluations) > 0 {
		fmt.Fprintf(&b, "## Contract Evaluations\n\n")
		fmt.Fprintf(&b, "- Status: `%s`\n", report.ContractSummary.Status)
		fmt.Fprintf(&b, "- Passed: `%d`\n", report.ContractSummary.Passed)
		fmt.Fprintf(&b, "- Failed: `%d`\n", report.ContractSummary.Failed)
		if report.ContractSummary.NotEvaluated > 0 {
			fmt.Fprintf(&b, "- Not evaluated: `%d`\n", report.ContractSummary.NotEvaluated)
		}
		if len(report.ContractSummary.EnvironmentWarnings) > 0 {
			fmt.Fprintf(&b, "- Environment warnings: `%s`\n", strings.Join(report.ContractSummary.EnvironmentWarnings, "`, `"))
		}
		b.WriteByte('\n')
		for _, evaluation := range report.ContractEvaluations {
			fmt.Fprintf(&b, "### %s\n\n", evaluation.Name)
			if evaluation.Query != "" {
				fmt.Fprintf(&b, "- Query: `%s`\n", evaluation.Query)
			}
			fmt.Fprintf(&b, "- Target ID: `%s`\n", evaluation.TargetID)
			if evaluation.Scope != "" {
				fmt.Fprintf(&b, "- Scope: `%s`\n", evaluation.Scope)
			}
			fmt.Fprintf(&b, "- Status: `%s`\n", evaluation.Status)
			if evaluation.Reason != "" {
				fmt.Fprintf(&b, "- Reason: `%s`\n", evaluation.Reason)
			}
			if evaluation.Error != "" {
				fmt.Fprintf(&b, "- Error: `%s`\n", evaluation.Error)
			}
			fmt.Fprintf(&b, "- Stability: `%s` check_recommended=%t replayable_from_report=%t\n", evaluation.Stability.Tier, evaluation.Stability.CheckRecommended, evaluation.Stability.ReplayableFromReport)
			for _, result := range evaluation.Results {
				label := result.OperatorFamily
				if label == "" {
					label = result.Rule
				}
				fmt.Fprintf(&b, "- `%s`: `%s` observed=%d max=%d", label, result.Status, result.ObservedCount, result.MaxCount)
				if result.FailureKind != "" {
					fmt.Fprintf(&b, " failure_kind=`%s`", result.FailureKind)
				}
				if result.DiagnosticID != "" {
					fmt.Fprintf(&b, " diagnostic_id=`%s`", result.DiagnosticID)
				}
				if result.MatchedOperatorIndexes != nil {
					fmt.Fprintf(&b, " matched_operator_indexes=`%s`", planReportFormatInt32List(*result.MatchedOperatorIndexes))
				}
				b.WriteByte('\n')
				for _, remediation := range result.Remediation {
					fmt.Fprintf(&b, "  - remediation `%s` (%s, confidence=%s, auto_fix=%t): %s\n", remediation.Kind, remediation.AppliesTo, remediation.Confidence, remediation.AutoFix, remediation.Message)
				}
			}
			b.WriteByte('\n')
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// planReportFormatOperatorFamilyCounts renders observed (non-zero) operator
// family counts for the human-readable Markdown report. Machine-readable
// YAML/JSON output keeps the complete zero-filled map for CEL replay.
func planReportFormatOperatorFamilyCounts(counts map[string]int) string {
	families := make([]string, 0, len(counts))
	for family := range counts {
		if counts[family] > 0 {
			families = append(families, family)
		}
	}
	sort.Strings(families)
	parts := make([]string, 0, len(families))
	for _, family := range families {
		parts = append(parts, fmt.Sprintf("%s=%d", family, counts[family]))
	}
	return strings.Join(parts, "`, `")
}

func planReportFormatInt32List(values []int32) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprint(value))
	}
	return strings.Join(parts, ",")
}

func planReportFederatedQueryIsZero(q querygen.QueryCodegenFederatedQuery) bool {
	return q.Connection == "" && q.SpannerSource == "" && q.InnerSQL == "" && q.OuterSQL == ""
}
