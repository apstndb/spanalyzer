// Package plancontract evaluates structural Cloud Spanner query plan contracts.
//
// The package intentionally works on an already-built plan report projection:
// it does not parse DDL, analyze SQL with the GoogleSQL frontend, or start an
// emulator/Omni backend. Callers are responsible for collecting QueryPlan data
// and normalizing PlanNode operators before invoking this package.
package plancontract

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/goccy/go-yaml"
	"github.com/google/cel-go/cel"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	StatusPass         = "pass"
	StatusFail         = "fail"
	StatusNotEvaluated = "not_evaluated"

	FileVersionV1Alpha = "v1alpha-plan-contracts"
	EvaluatorVersionV1 = "v1"

	StabilityNormalized = "normalized_operator"
	StabilityRawPlan    = "raw_query_plan"

	EvaluationModeNone       = "none"
	EvaluationModeReportOnly = "report_only"
	EvaluationModeCheck      = "check"

	FailureKindViolation             = "violation"
	FailureKindClassificationUnknown = "classification_unknown"
	ReasonTargetNotFound             = "target_not_found"
	ReasonTargetError                = "target_error"
)

const (
	IdentifierPattern = `[A-Za-z_][A-Za-z0-9_]*`
	TargetIDPattern   = `^query/` + IdentifierPattern + `(#inner)?$`
)

var (
	identifierRegexp = regexp.MustCompile(`^` + IdentifierPattern + `$`)
	targetIDRegexp   = regexp.MustCompile(TargetIDPattern)
)

// File is the top-level plan contract YAML document.
type File struct {
	Version   string     `json:"version" yaml:"version"`
	Contracts []Contract `json:"contracts" yaml:"contracts"`
}

// Contract describes one user-authored plan contract.
type Contract struct {
	Name   string      `json:"name" yaml:"name"`
	Target string      `json:"target" yaml:"target"`
	Use    []string    `json:"use,omitempty" yaml:"use,omitempty"`
	Forbid []Predicate `json:"forbid,omitempty" yaml:"forbid,omitempty"`
	CEL    string      `json:"cel,omitempty" yaml:"cel,omitempty"`
}

// Predicate forbids an operator family above MaxCount.
type Predicate struct {
	OperatorFamily string `json:"operator_family" yaml:"operator_family"`
	MaxCount       int    `json:"max_count,omitempty" yaml:"max_count,omitempty"`
}

type resolvedPredicate struct {
	Predicate
	Rule       string
	Source     string
	Predefined string
}

const (
	ruleForbidOperatorFamily               = "forbid_operator_family"
	ruleForbidBlockingOperatorUnderLimit   = "forbid_blocking_operator_under_limit"
	ruleForbidFullScan                     = "forbid_full_scan"
	ruleForbidFullScanWithoutTimestamp     = "forbid_full_scan_without_timestamp_condition"
	ruleRequireTimestampCondition          = "require_timestamp_condition"
	predefinedNoFullScan                   = "no_full_scan"
	predefinedNoFullScanWithoutTimestamp   = "no_full_scan_without_timestamp_condition"
	predefinedRequireTimestampCondition    = "require_timestamp_condition"
	predefinedNoBlockingOperatorUnderLimit = "no_blocking_operator_under_limit"
)

// Report is the contract evaluator input projection.
type Report struct {
	Backend         string
	BackendIdentity BackendIdentity
	Optimizer       Optimizer
	Queries         []Query
}

// BackendIdentity identifies the backend that produced the plan.
type BackendIdentity struct {
	Kind        string `json:"kind" yaml:"kind"`
	Version     string `json:"version" yaml:"version"`
	ImageDigest string `json:"image_digest" yaml:"image_digest"`
	Source      string `json:"source" yaml:"source"`
}

// Optimizer records requested and effective optimizer settings.
type Optimizer struct {
	Requested OptimizerEnvironment `json:"requested" yaml:"requested"`
	Effective OptimizerEffective   `json:"effective" yaml:"effective"`
}

// OptimizerEnvironment is the requested optimizer environment.
type OptimizerEnvironment struct {
	Version           string `json:"version" yaml:"version"`
	StatisticsPackage string `json:"statistics_package" yaml:"statistics_package"`
}

// OptimizerEffective is the observed optimizer environment.
type OptimizerEffective struct {
	Version           string `json:"version" yaml:"version"`
	StatisticsPackage string `json:"statistics_package" yaml:"statistics_package"`
}

// Query is the per-query contract evaluator input projection.
type Query struct {
	TargetID             string
	Name                 string
	Catalog              string
	Scope                string
	Kind                 string
	Status               string
	SQLSHA256            string
	DDLSHA256            string
	OperatorTreeSHA256   string
	OperatorFamilies     []string
	OperatorFamilyCounts map[string]int
	NormalizedOperators  []Operator
	OperatorEdges        []OperatorEdge
	Error                string
	OptimizerNotPinned   bool
	PlanEnvironmentNotes []string
	ClassificationNotes  []Diagnostic
	RawPlan              *spannerpb.QueryPlan
}

// Operator is the normalized operator view used by stable predefined contracts.
type Operator struct {
	Index               int32          `json:"index" yaml:"index"`
	DisplayName         string         `json:"display_name" yaml:"display_name"`
	Family              string         `json:"family" yaml:"family"`
	ExecutionMethod     string         `json:"execution_method,omitempty" yaml:"execution_method,omitempty"`
	IteratorType        string         `json:"iterator_type,omitempty" yaml:"iterator_type,omitempty"`
	ScanMethod          string         `json:"scan_method,omitempty" yaml:"scan_method,omitempty"`
	ScanFormat          string         `json:"scan_format,omitempty" yaml:"scan_format,omitempty"`
	ScanType            string         `json:"scan_type,omitempty" yaml:"scan_type,omitempty"`
	ScanTarget          string         `json:"scan_target,omitempty" yaml:"scan_target,omitempty"`
	SeekableKeySize     string         `json:"seekable_key_size,omitempty" yaml:"seekable_key_size,omitempty"`
	JoinType            string         `json:"join_type,omitempty" yaml:"join_type,omitempty"`
	JoinConfiguration   string         `json:"join_configuration,omitempty" yaml:"join_configuration,omitempty"`
	CallType            string         `json:"call_type,omitempty" yaml:"call_type,omitempty"`
	DistributionTable   string         `json:"distribution_table,omitempty" yaml:"distribution_table,omitempty"`
	SubqueryClusterNode string         `json:"subquery_cluster_node,omitempty" yaml:"subquery_cluster_node,omitempty"`
	SpoolName           string         `json:"spool_name,omitempty" yaml:"spool_name,omitempty"`
	FullScan            bool           `json:"full_scan,omitempty" yaml:"full_scan,omitempty"`
	ChildIndexes        []int32        `json:"child_indexes" yaml:"child_indexes"`
	DescendantIndexes   []int32        `json:"descendant_indexes" yaml:"descendant_indexes"`
	SubtreeFamilyCounts map[string]int `json:"subtree_family_counts" yaml:"subtree_family_counts"`
}

// OperatorEdge records a PlanNode parent-child edge.
type OperatorEdge struct {
	ParentIndex int32  `json:"parent_index" yaml:"parent_index"`
	ChildIndex  int32  `json:"child_index" yaml:"child_index"`
	Type        string `json:"type,omitempty" yaml:"type,omitempty"`
	Variable    string `json:"variable,omitempty" yaml:"variable,omitempty"`
}

// Diagnostic records plan normalization or classification warnings.
type Diagnostic struct {
	ID      string `json:"id" yaml:"id"`
	Message string `json:"message" yaml:"message"`
}

// ApplyResult contains contract evaluation output fields.
type ApplyResult struct {
	FileVersion      string
	EvaluatorVersion string
	Evaluations      []Evaluation
	Summary          EvaluationSummary
}

// EvaluationSummary summarizes all contract evaluations.
type EvaluationSummary struct {
	Status              string   `json:"status" yaml:"status"`
	Contracts           int      `json:"contracts" yaml:"contracts"`
	Passed              int      `json:"passed" yaml:"passed"`
	Failed              int      `json:"failed" yaml:"failed"`
	NotEvaluated        int      `json:"not_evaluated" yaml:"not_evaluated"`
	EnvironmentWarnings []string `json:"environment_warnings" yaml:"environment_warnings"`
}

// Evaluation is the result for one Contract.
type Evaluation struct {
	Name      string       `json:"name" yaml:"name"`
	Query     string       `json:"query,omitempty" yaml:"query,omitempty"`
	TargetID  string       `json:"target_id" yaml:"target_id"`
	Scope     string       `json:"scope,omitempty" yaml:"scope,omitempty"`
	Status    string       `json:"status" yaml:"status"`
	Reason    string       `json:"reason,omitempty" yaml:"reason,omitempty"`
	Error     string       `json:"error,omitempty" yaml:"error,omitempty"`
	Stability Stability    `json:"stability" yaml:"stability"`
	Results   []RuleResult `json:"results,omitempty" yaml:"results,omitempty"`
}

// Stability classifies the robustness of a contract rule.
type Stability struct {
	Tier                 string   `json:"tier" yaml:"tier"`
	Reasons              []string `json:"reasons" yaml:"reasons"`
	CheckRecommended     bool     `json:"check_recommended" yaml:"check_recommended"`
	ReplayableFromReport bool     `json:"replayable_from_report" yaml:"replayable_from_report"`
}

// RuleResult is the result for one expanded contract rule.
type RuleResult struct {
	Rule                   string        `json:"rule,omitempty" yaml:"rule,omitempty"`
	Source                 string        `json:"source,omitempty" yaml:"source,omitempty"`
	Predefined             string        `json:"predefined,omitempty" yaml:"predefined,omitempty"`
	Expression             string        `json:"expression,omitempty" yaml:"expression,omitempty"`
	OperatorFamily         string        `json:"operator_family,omitempty" yaml:"operator_family,omitempty"`
	Status                 string        `json:"status" yaml:"status"`
	FailureKind            string        `json:"failure_kind,omitempty" yaml:"failure_kind,omitempty"`
	DiagnosticID           string        `json:"diagnostic_id,omitempty" yaml:"diagnostic_id,omitempty"`
	ObservedCount          int           `json:"observed_count,omitempty" yaml:"observed_count,omitempty"`
	MaxCount               int           `json:"max_count,omitempty" yaml:"max_count,omitempty"`
	MatchedOperatorIndexes *[]int32      `json:"matched_operator_indexes,omitempty" yaml:"matched_operator_indexes,omitempty"`
	Remediation            []Remediation `json:"remediation,omitempty" yaml:"remediation,omitempty"`
}

// Remediation describes a possible follow-up when a contract fails.
type Remediation struct {
	Kind       string `json:"kind" yaml:"kind"`
	AppliesTo  string `json:"applies_to,omitempty" yaml:"applies_to,omitempty"`
	Confidence string `json:"confidence,omitempty" yaml:"confidence,omitempty"`
	AutoFix    bool   `json:"auto_fix" yaml:"auto_fix"`
	Message    string `json:"message" yaml:"message"`
}

// ReadFile reads and validates a plan contract YAML file.
func ReadFile(path string) (File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, err
	}
	var contracts File
	if err := yaml.UnmarshalWithOptions(data, &contracts, yaml.DisallowUnknownField()); err != nil {
		return File{}, err
	}
	if strings.TrimSpace(contracts.Version) != FileVersionV1Alpha {
		return File{}, fmt.Errorf("unsupported plan contracts version %q; use version: %s", contracts.Version, FileVersionV1Alpha)
	}
	if len(contracts.Contracts) == 0 {
		return File{}, fmt.Errorf("plan contracts file must contain at least one contract")
	}
	seen := map[string]bool{}
	for _, contract := range contracts.Contracts {
		name := strings.TrimSpace(contract.Name)
		if name == "" {
			return File{}, fmt.Errorf("plan contract name is required")
		}
		if !identifierRegexp.MatchString(name) {
			return File{}, fmt.Errorf("plan contract name %q must match ^%s$", name, IdentifierPattern)
		}
		if seen[name] {
			return File{}, fmt.Errorf("plan_contract.duplicate_contract_name: duplicate plan contract name %q", name)
		}
		target := strings.TrimSpace(contract.Target)
		if target == "" {
			return File{}, fmt.Errorf("plan contract %s target is required", name)
		}
		if !targetIDRegexp.MatchString(target) {
			return File{}, fmt.Errorf("plan contract %s target %q must match %s", name, target, TargetIDPattern)
		}
		forbidSeen := map[string]bool{}
		for _, predicate := range contract.Forbid {
			family := strings.TrimSpace(predicate.OperatorFamily)
			if family == "" {
				continue
			}
			if forbidSeen[family] {
				return File{}, fmt.Errorf("plan_contract.duplicate_forbid_operator_family: plan contract %s has duplicate forbid.operator_family %q", name, family)
			}
			forbidSeen[family] = true
		}
		seen[name] = true
	}
	return contracts, nil
}

// Evaluate evaluates contracts against a plan report projection.
func Evaluate(report Report, contracts File) (ApplyResult, error) {
	evaluations := make([]Evaluation, 0, len(contracts.Contracts))
	for _, contract := range contracts.Contracts {
		evaluation, err := evaluateContract(report, contract)
		if err != nil {
			return ApplyResult{}, err
		}
		evaluations = append(evaluations, evaluation)
	}
	summary := Summarize(evaluations)
	summary.EnvironmentWarnings = EnvironmentWarnings(report)
	return ApplyResult{
		FileVersion:      contracts.Version,
		EvaluatorVersion: EvaluatorVersionV1,
		Evaluations:      evaluations,
		Summary:          summary,
	}, nil
}

func evaluateContract(report Report, contract Contract) (Evaluation, error) {
	if strings.TrimSpace(contract.Name) == "" {
		return Evaluation{}, fmt.Errorf("plan contract name is required")
	}
	baseEvaluation := Evaluation{
		Name:      contract.Name,
		TargetID:  strings.TrimSpace(contract.Target),
		Status:    StatusPass,
		Stability: stabilityFor(contract),
	}
	query, ok, err := findTarget(report, contract)
	if err != nil {
		return Evaluation{}, fmt.Errorf("plan contract %s: %w", contract.Name, err)
	}
	if !ok {
		baseEvaluation.Status = StatusNotEvaluated
		baseEvaluation.Reason = ReasonTargetNotFound
		return baseEvaluation, nil
	}
	if query.Status != "" && query.Status != "ok" {
		baseEvaluation.Query = query.Name
		baseEvaluation.TargetID = queryTargetID(query)
		baseEvaluation.Scope = query.Scope
		baseEvaluation.Status = StatusNotEvaluated
		baseEvaluation.Reason = ReasonTargetError
		baseEvaluation.Error = fmt.Sprintf("target is not an analyzed Spanner target: status=%s error=%q", query.Status, query.Error)
		return baseEvaluation, nil
	}
	predicates, err := predicates(contract)
	if err != nil {
		return Evaluation{}, fmt.Errorf("plan contract %s: %w", contract.Name, err)
	}
	evaluation := baseEvaluation
	evaluation.Query = query.Name
	evaluation.TargetID = queryTargetID(query)
	evaluation.Scope = query.Scope
	if expression := strings.TrimSpace(contract.CEL); expression != "" {
		if err := validateCELExpression(expression); err != nil {
			return Evaluation{}, fmt.Errorf("plan contract %s cel: %w", contract.Name, err)
		}
	}
	for _, predicate := range predicates {
		result, err := evaluatePredicate(query, predicate)
		if err != nil {
			return Evaluation{}, err
		}
		if result.Status == StatusFail {
			evaluation.Status = StatusFail
		}
		evaluation.Results = append(evaluation.Results, result)
	}
	if expression := strings.TrimSpace(contract.CEL); expression != "" {
		matched, err := evaluateCEL(report, query, expression)
		if err != nil {
			return Evaluation{}, fmt.Errorf("plan contract %s cel: %w", contract.Name, err)
		}
		result := RuleResult{
			Rule:       "cel",
			Source:     "cel",
			Expression: expression,
			Status:     StatusPass,
		}
		if !matched {
			result.Status = StatusFail
			result.FailureKind = FailureKindViolation
			result.Remediation = []Remediation{{
				Kind:       "cel_contract",
				AppliesTo:  "contract",
				Confidence: "low",
				Message:    "Review the CEL expression against the normalized and raw plan-report inputs, then adjust the query, hints, index design, or contract.",
			}}
			evaluation.Status = StatusFail
		}
		evaluation.Results = append(evaluation.Results, result)
	}
	return evaluation, nil
}

func evaluatePredicate(query Query, predicate resolvedPredicate) (RuleResult, error) {
	rule := predicate.Rule
	if rule == "" {
		rule = ruleForbidOperatorFamily
	}
	switch rule {
	case ruleForbidOperatorFamily:
		count := operatorFamilyCount(query, predicate.OperatorFamily)
		matchedOperatorIndexes := predicateMatchedOperatorIndexes(query, predicate.OperatorFamily)
		if matchedOperatorIndexes == nil {
			matchedOperatorIndexes = []int32{}
		}
		result := RuleResult{
			Rule:                   ruleForbidOperatorFamily,
			Source:                 predicate.Source,
			Predefined:             predicate.Predefined,
			OperatorFamily:         predicate.OperatorFamily,
			Status:                 StatusPass,
			ObservedCount:          count,
			MaxCount:               predicate.MaxCount,
			MatchedOperatorIndexes: &matchedOperatorIndexes,
		}
		if count > predicate.MaxCount {
			result.Status = StatusFail
			result.FailureKind = FailureKindViolation
			result.Remediation = remediationFor(predicate.OperatorFamily)
		}
		if predicateClassificationUnknown(query, predicate.OperatorFamily) {
			result.Status = StatusFail
			result.FailureKind = FailureKindClassificationUnknown
			result.DiagnosticID = classificationUnknownDiagnosticID(predicate.OperatorFamily)
			result.Remediation = append(result.Remediation, classificationUnknownRemediation(predicate.OperatorFamily)...)
		}
		return result, nil
	case ruleForbidBlockingOperatorUnderLimit:
		matchedOperatorIndexes := blockingOperatorUnderLimitIndexes(query)
		result := RuleResult{
			Rule:                   ruleForbidBlockingOperatorUnderLimit,
			Source:                 predicate.Source,
			Predefined:             predicate.Predefined,
			Status:                 StatusPass,
			ObservedCount:          len(matchedOperatorIndexes),
			MaxCount:               0,
			MatchedOperatorIndexes: &matchedOperatorIndexes,
		}
		if len(matchedOperatorIndexes) > 0 {
			result.Status = StatusFail
			result.FailureKind = FailureKindViolation
			result.Remediation = blockingOperatorUnderLimitRemediation()
		}
		return result, nil
	case ruleForbidFullScan:
		matchedOperatorIndexes := fullScanOperatorIndexes(query)
		result := RuleResult{
			Rule:                   ruleForbidFullScan,
			Source:                 predicate.Source,
			Predefined:             predicate.Predefined,
			Status:                 StatusPass,
			ObservedCount:          len(matchedOperatorIndexes),
			MaxCount:               0,
			MatchedOperatorIndexes: &matchedOperatorIndexes,
		}
		if len(matchedOperatorIndexes) > 0 {
			result.Status = StatusFail
			result.FailureKind = FailureKindViolation
			result.Remediation = fullScanRemediation()
		}
		return result, nil
	case ruleForbidFullScanWithoutTimestamp:
		matchedOperatorIndexes := fullScanWithoutTimestampConditionOperatorIndexes(query)
		result := RuleResult{
			Rule:                   ruleForbidFullScanWithoutTimestamp,
			Source:                 predicate.Source,
			Predefined:             predicate.Predefined,
			Status:                 StatusPass,
			ObservedCount:          len(matchedOperatorIndexes),
			MaxCount:               0,
			MatchedOperatorIndexes: &matchedOperatorIndexes,
		}
		if len(matchedOperatorIndexes) > 0 {
			result.Status = StatusFail
			result.FailureKind = FailureKindViolation
			result.Remediation = fullScanWithoutTimestampConditionRemediation()
		}
		return result, nil
	case ruleRequireTimestampCondition:
		matchedOperatorIndexes := timestampConditionOperatorIndexes(query)
		result := RuleResult{
			Rule:                   ruleRequireTimestampCondition,
			Source:                 predicate.Source,
			Predefined:             predicate.Predefined,
			Status:                 StatusPass,
			ObservedCount:          len(matchedOperatorIndexes),
			MaxCount:               0,
			MatchedOperatorIndexes: &matchedOperatorIndexes,
		}
		if len(matchedOperatorIndexes) == 0 {
			result.Status = StatusFail
			result.FailureKind = FailureKindViolation
			result.Remediation = timestampConditionRemediation()
		}
		return result, nil
	default:
		return RuleResult{}, fmt.Errorf("unsupported plan contract rule %q", rule)
	}
}

func stabilityFor(contract Contract) Stability {
	stability := Stability{
		Tier:                 StabilityNormalized,
		CheckRecommended:     true,
		ReplayableFromReport: true,
		Reasons:              []string{"contract uses the normalized plan-report view"},
	}
	expression := strings.TrimSpace(contract.CEL)
	if expression == "" {
		return stability
	}
	identifiers, parsed := referencedCELIdentifiers(expression)
	if parsed && (identifiers["raw_plan"] || identifiers["raw_nodes"]) {
		stability.Tier = StabilityRawPlan
		stability.CheckRecommended = false
		stability.ReplayableFromReport = false
		stability.Reasons = []string{
			"CEL expression references raw QueryPlan or PlanNode inputs",
			"raw QueryPlan CEL is not replayable from the serialized report alone",
		}
		return stability
	}
	if !parsed && (referencesIdentifier(expression, "raw_plan") || referencesIdentifier(expression, "raw_nodes")) {
		stability.Tier = StabilityRawPlan
		stability.CheckRecommended = false
		stability.ReplayableFromReport = false
		stability.Reasons = []string{
			"CEL expression references raw QueryPlan or PlanNode inputs",
			"raw QueryPlan CEL is not replayable from the serialized report alone",
		}
		return stability
	}
	if fields := referencedMetadataDerivedOperatorFields(expression, identifiers, parsed); len(fields) > 0 {
		stability.Reasons = append(stability.Reasons, "contract reads metadata-derived normalized fields: "+strings.Join(fields, ", "))
	}
	return stability
}

func referencedMetadataDerivedOperatorFields(expression string, identifiers map[string]bool, parsed bool) []string {
	fields := []string{
		"call_type",
		"distribution_table",
		"execution_method",
		"full_scan",
		"iterator_type",
		"join_configuration",
		"join_type",
		"scan_format",
		"scan_method",
		"scan_target",
		"scan_type",
		"seekable_key_size",
		"spool_name",
		"subquery_cluster_node",
	}
	var out []string
	for _, field := range fields {
		if parsed && identifiers[field] {
			out = append(out, field)
			continue
		}
		if !parsed && referencesIdentifier(expression, field) {
			out = append(out, field)
		}
	}
	return out
}

func validateCELExpression(expression string) error {
	identifiers, parsed := referencedCELIdentifiers(expression)
	if parsed && identifiers["execution_stats"] {
		return fmt.Errorf("execution_stats is not supported; plan contracts target structural PLAN output, not PROFILE execution statistics")
	}
	if !parsed && referencesIdentifier(strings.ToLower(expression), "execution_stats") {
		return fmt.Errorf("execution_stats is not supported; plan contracts target structural PLAN output, not PROFILE execution statistics")
	}
	return nil
}

func referencedCELIdentifiers(expression string) (map[string]bool, bool) {
	env, err := newCELEnv()
	if err != nil {
		return nil, false
	}
	ast, issues := env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, false
	}
	identifiers := map[string]bool{}
	checked, err := cel.AstToCheckedExpr(ast)
	if err != nil {
		return nil, false
	}
	collectCELIdentifiers(checked.GetExpr(), identifiers)
	return identifiers, true
}

func collectCELIdentifiers(expr *exprpb.Expr, identifiers map[string]bool) {
	if expr == nil {
		return
	}
	if ident := expr.GetIdentExpr(); ident != nil {
		identifiers[ident.GetName()] = true
		return
	}
	if selectExpr := expr.GetSelectExpr(); selectExpr != nil {
		identifiers[selectExpr.GetField()] = true
		collectCELIdentifiers(selectExpr.GetOperand(), identifiers)
		return
	}
	if call := expr.GetCallExpr(); call != nil {
		collectCELIdentifiers(call.GetTarget(), identifiers)
		for _, arg := range call.GetArgs() {
			collectCELIdentifiers(arg, identifiers)
		}
		return
	}
	if list := expr.GetListExpr(); list != nil {
		for _, elem := range list.GetElements() {
			collectCELIdentifiers(elem, identifiers)
		}
		return
	}
	if structExpr := expr.GetStructExpr(); structExpr != nil {
		for _, entry := range structExpr.GetEntries() {
			collectCELIdentifiers(entry.GetMapKey(), identifiers)
			collectCELIdentifiers(entry.GetValue(), identifiers)
		}
		return
	}
	if comprehension := expr.GetComprehensionExpr(); comprehension != nil {
		collectCELIdentifiers(comprehension.GetIterRange(), identifiers)
		collectCELIdentifiers(comprehension.GetAccuInit(), identifiers)
		collectCELIdentifiers(comprehension.GetLoopCondition(), identifiers)
		collectCELIdentifiers(comprehension.GetLoopStep(), identifiers)
		collectCELIdentifiers(comprehension.GetResult(), identifiers)
	}
}

func referencesIdentifier(expression, identifier string) bool {
	if identifier == "" {
		return false
	}
	for start := 0; start < len(expression); {
		idx := strings.Index(expression[start:], identifier)
		if idx < 0 {
			return false
		}
		pos := start + idx
		beforeOK := pos == 0 || !identifierByte(expression[pos-1])
		after := pos + len(identifier)
		afterOK := after == len(expression) || !identifierByte(expression[after])
		if beforeOK && afterOK {
			return true
		}
		start = after
	}
	return false
}

func identifierByte(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') ||
		b == '_'
}

func findTarget(report Report, contract Contract) (Query, bool, error) {
	target := strings.TrimSpace(contract.Target)
	if target == "" {
		return Query{}, false, fmt.Errorf("target is required")
	}
	query, ok := FindTarget(report, target)
	if !ok {
		return Query{}, false, nil
	}
	return query, true, nil
}

// FindTarget finds a query by canonical contract target ID.
func FindTarget(report Report, targetID string) (Query, bool) {
	for _, query := range report.Queries {
		if queryTargetID(query) == targetID {
			return query, true
		}
	}
	return Query{}, false
}

func predicates(contract Contract) ([]resolvedPredicate, error) {
	modes := 0
	if len(contract.Use) > 0 {
		modes++
	}
	if len(contract.Forbid) > 0 {
		modes++
	}
	if strings.TrimSpace(contract.CEL) != "" {
		modes++
	}
	if modes != 1 {
		return nil, fmt.Errorf("exactly one of use, forbid, or cel is required")
	}
	var predicates []resolvedPredicate
	for _, name := range contract.Use {
		predefined := strings.ToLower(strings.TrimSpace(name))
		source := "use/" + predefined
		appendPredefined := func(family string) {
			predicates = append(predicates, resolvedPredicate{
				Predicate:  Predicate{OperatorFamily: family},
				Rule:       ruleForbidOperatorFamily,
				Source:     source,
				Predefined: predefined,
			})
		}
		appendPredefinedRule := func(rule string) {
			predicates = append(predicates, resolvedPredicate{
				Rule:       rule,
				Source:     source,
				Predefined: predefined,
			})
		}
		switch predefined {
		case "no_explicit_sort":
			appendPredefined("explicit_sort")
		case "no_full_sort":
			appendPredefined("full_sort")
		case "no_minor_sort":
			appendPredefined("minor_sort")
		case "no_hash_join":
			appendPredefined("hash_join")
			appendPredefined("push_broadcast_hash_join")
		case "no_standalone_hash_join":
			appendPredefined("hash_join")
		case "no_push_broadcast_hash_join":
			appendPredefined("push_broadcast_hash_join")
		case "no_apply_join":
			appendPredefined("apply_join")
			appendPredefined("anti_semi_apply")
			appendPredefined("semi_apply")
			appendPredefined("distributed_cross_apply")
			appendPredefined("distributed_anti_semi_apply")
			appendPredefined("distributed_semi_apply")
		case "no_standalone_apply_join":
			appendPredefined("apply_join")
			appendPredefined("anti_semi_apply")
			appendPredefined("semi_apply")
		case "no_distributed_cross_apply":
			appendPredefined("distributed_cross_apply")
		case "no_merge_join":
			appendPredefined("merge_join")
		case "no_hash_aggregate":
			appendPredefined("hash_aggregate")
		case "no_stream_aggregate":
			appendPredefined("stream_aggregate")
		case predefinedNoFullScan:
			appendPredefinedRule(ruleForbidFullScan)
		case predefinedNoFullScanWithoutTimestamp:
			appendPredefinedRule(ruleForbidFullScanWithoutTimestamp)
		case predefinedRequireTimestampCondition:
			appendPredefinedRule(ruleRequireTimestampCondition)
		case predefinedNoBlockingOperatorUnderLimit:
			appendPredefinedRule(ruleForbidBlockingOperatorUnderLimit)
		default:
			return nil, fmt.Errorf("unsupported predefined plan contract %q", name)
		}
	}
	for i, predicate := range contract.Forbid {
		if strings.TrimSpace(predicate.OperatorFamily) == "" {
			return nil, fmt.Errorf("forbid.operator_family is required")
		}
		if predicate.MaxCount < 0 {
			return nil, fmt.Errorf("forbid.operator_family %q max_count must be >= 0", predicate.OperatorFamily)
		}
		predicate.OperatorFamily = strings.TrimSpace(predicate.OperatorFamily)
		if !KnownOperatorFamily(predicate.OperatorFamily) {
			return nil, fmt.Errorf("unsupported forbid.operator_family %q", predicate.OperatorFamily)
		}
		predicates = append(predicates, resolvedPredicate{
			Predicate: predicate,
			Rule:      ruleForbidOperatorFamily,
			Source:    fmt.Sprintf("forbid[%d]", i),
		})
	}
	return predicates, nil
}

func evaluateCEL(report Report, query Query, expression string) (bool, error) {
	env, err := newCELEnv()
	if err != nil {
		return false, err
	}
	ast, issues := env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return false, issues.Err()
	}
	program, err := env.Program(ast)
	if err != nil {
		return false, err
	}
	value, _, err := program.Eval(celActivation(report, query))
	if err != nil {
		return false, err
	}
	matched, ok := value.Value().(bool)
	if !ok {
		return false, fmt.Errorf("expression must evaluate to bool, got %s", value.Type())
	}
	return matched, nil
}

func newCELEnv() (*cel.Env, error) {
	return cel.NewEnv(
		cel.Types(
			&spannerpb.QueryPlan{},
			&spannerpb.PlanNode{},
			&spannerpb.PlanNode_ChildLink{},
			&spannerpb.PlanNode_ShortRepresentation{},
			&structpb.Struct{},
			&structpb.Value{},
			&structpb.ListValue{},
		),
		cel.Variable("backend", cel.StringType),
		cel.Variable("query", cel.DynType),
		cel.Variable("operators", cel.ListType(cel.DynType)),
		cel.Variable("operator_edges", cel.ListType(cel.DynType)),
		cel.Variable("operator_families", cel.ListType(cel.StringType)),
		cel.Variable("operator_family_counts", cel.MapType(cel.StringType, cel.IntType)),
		cel.Variable("optimizer", cel.DynType),
		cel.Variable("raw_plan", cel.ObjectType("google.spanner.v1.QueryPlan")),
		cel.Variable("raw_nodes", cel.ListType(cel.ObjectType("google.spanner.v1.PlanNode"))),
	)
}

func celActivation(report Report, query Query) map[string]interface{} {
	rawPlan := query.RawPlan
	if rawPlan == nil {
		rawPlan = &spannerpb.QueryPlan{}
	}
	return map[string]interface{}{
		"backend":                report.Backend,
		"query":                  celQuery(query),
		"operators":              celOperators(query.NormalizedOperators),
		"operator_edges":         celOperatorEdges(query.OperatorEdges),
		"operator_families":      query.OperatorFamilies,
		"operator_family_counts": OperatorFamilyCountsOrEmpty(query.OperatorFamilyCounts),
		"optimizer": map[string]interface{}{
			"requested": map[string]interface{}{
				"version":            report.Optimizer.Requested.Version,
				"statistics_package": report.Optimizer.Requested.StatisticsPackage,
			},
			"effective": map[string]interface{}{
				"version":            report.Optimizer.Effective.Version,
				"statistics_package": report.Optimizer.Effective.StatisticsPackage,
			},
		},
		"raw_plan":  rawPlan,
		"raw_nodes": rawPlan.GetPlanNodes(),
	}
}

func celQuery(query Query) map[string]interface{} {
	return map[string]interface{}{
		"target_id":               queryTargetID(query),
		"name":                    query.Name,
		"catalog":                 query.Catalog,
		"scope":                   query.Scope,
		"kind":                    query.Kind,
		"status":                  query.Status,
		"sql_sha256":              query.SQLSHA256,
		"ddl_sha256":              query.DDLSHA256,
		"operator_tree_sha256":    query.OperatorTreeSHA256,
		"operator_family_counts":  OperatorFamilyCountsOrEmpty(query.OperatorFamilyCounts),
		"optimizer_not_pinned":    query.OptimizerNotPinned,
		"plan_environment_notes":  query.PlanEnvironmentNotes,
		"classification_warnings": celDiagnostics(query.ClassificationNotes),
	}
}

func celOperators(operators []Operator) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(operators))
	for _, operator := range operators {
		out = append(out, map[string]interface{}{
			"index":                 operator.Index,
			"display_name":          operator.DisplayName,
			"family":                operator.Family,
			"execution_method":      operator.ExecutionMethod,
			"iterator_type":         operator.IteratorType,
			"scan_method":           operator.ScanMethod,
			"scan_format":           operator.ScanFormat,
			"scan_type":             operator.ScanType,
			"scan_target":           operator.ScanTarget,
			"seekable_key_size":     operator.SeekableKeySize,
			"join_type":             operator.JoinType,
			"join_configuration":    operator.JoinConfiguration,
			"call_type":             operator.CallType,
			"distribution_table":    operator.DistributionTable,
			"subquery_cluster_node": operator.SubqueryClusterNode,
			"spool_name":            operator.SpoolName,
			"full_scan":             operator.FullScan,
			"child_indexes":         operator.ChildIndexes,
			"descendant_indexes":    operator.DescendantIndexes,
			"subtree_family_counts": OperatorFamilyCountsOrEmpty(operator.SubtreeFamilyCounts),
		})
	}
	return out
}

func celOperatorEdges(edges []OperatorEdge) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(edges))
	for _, edge := range edges {
		out = append(out, map[string]interface{}{
			"parent_index": edge.ParentIndex,
			"child_index":  edge.ChildIndex,
			"type":         edge.Type,
			"variable":     edge.Variable,
		})
	}
	return out
}

func celDiagnostics(diagnostics []Diagnostic) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		out = append(out, map[string]interface{}{
			"id":      diagnostic.ID,
			"message": diagnostic.Message,
		})
	}
	return out
}

// Scope returns the canonical contract scope string.
func Scope(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "", "query":
		return ""
	case "inner", "external_query.inner":
		return "external_query.inner"
	default:
		return strings.TrimSpace(scope)
	}
}

// TargetID returns the canonical contract target ID for a query name and scope.
func TargetID(queryName, scope string) string {
	switch Scope(scope) {
	case "external_query.inner":
		return "query/" + queryName + "#inner"
	default:
		return "query/" + queryName
	}
}

func queryTargetID(query Query) string {
	if strings.TrimSpace(query.TargetID) != "" {
		return query.TargetID
	}
	return TargetID(query.Name, query.Scope)
}

func operatorFamilyCount(query Query, family string) int {
	count := 0
	for _, operator := range query.NormalizedOperators {
		if operatorFamilyMatches(operator.Family, family) {
			count++
		}
	}
	return count
}

func predicateMatchedOperatorIndexes(query Query, family string) []int32 {
	if predicateClassificationUnknown(query, family) {
		switch {
		case aggregateClassificationUnknownFamily(family):
			indexes := aggregateClassificationUnknownOperatorIndexes(query)
			if len(indexes) > 0 {
				return indexes
			}
		case joinClassificationUnknownFamily(family):
			indexes := joinClassificationUnknownOperatorIndexes(query)
			if len(indexes) > 0 {
				return indexes
			}
		}
	}
	var indexes []int32
	for _, operator := range query.NormalizedOperators {
		if operatorFamilyMatches(operator.Family, family) {
			indexes = append(indexes, operator.Index)
		}
	}
	return indexes
}

func operatorFamilyMatches(operatorFamily, want string) bool {
	if operatorFamily == want {
		return true
	}
	switch want {
	case "explicit_sort":
		return operatorFamily == "full_sort" || operatorFamily == "minor_sort"
	case "blocking_operator":
		return streamBlockingOperatorFamily(operatorFamily)
	default:
		return false
	}
}

func blockingOperatorUnderLimitIndexes(query Query) []int32 {
	operatorsByIndex := make(map[int32]Operator, len(query.NormalizedOperators))
	for _, operator := range query.NormalizedOperators {
		operatorsByIndex[operator.Index] = operator
	}
	seen := map[int32]bool{}
	for _, operator := range query.NormalizedOperators {
		if !limitOrSortLimitOperator(operator) {
			continue
		}
		for _, descendantIndex := range operator.DescendantIndexes {
			descendant, ok := operatorsByIndex[descendantIndex]
			if !ok || !streamBlockingOperatorFamily(descendant.Family) {
				continue
			}
			seen[descendant.Index] = true
		}
	}
	indexes := make([]int32, 0, len(seen))
	for index := range seen {
		indexes = append(indexes, index)
	}
	sort.Slice(indexes, func(i, j int) bool { return indexes[i] < indexes[j] })
	return indexes
}

func fullScanOperatorIndexes(query Query) []int32 {
	indexes := []int32{}
	for _, operator := range query.NormalizedOperators {
		if operator.Family == "scan" && operator.FullScan {
			indexes = append(indexes, operator.Index)
		}
	}
	sort.Slice(indexes, func(i, j int) bool { return indexes[i] < indexes[j] })
	return indexes
}

func fullScanWithoutTimestampConditionOperatorIndexes(query Query) []int32 {
	timestampConditionParents := map[int32]bool{}
	for _, edge := range query.OperatorEdges {
		if edge.Type == "Timestamp Condition" {
			timestampConditionParents[edge.ParentIndex] = true
		}
	}
	indexes := []int32{}
	for _, operator := range query.NormalizedOperators {
		if operator.Family == "scan" && operator.FullScan && !timestampConditionParents[operator.Index] {
			indexes = append(indexes, operator.Index)
		}
	}
	sort.Slice(indexes, func(i, j int) bool { return indexes[i] < indexes[j] })
	return indexes
}

func timestampConditionOperatorIndexes(query Query) []int32 {
	seen := map[int32]bool{}
	for _, edge := range query.OperatorEdges {
		if edge.Type == "Timestamp Condition" {
			seen[edge.ParentIndex] = true
		}
	}
	indexes := make([]int32, 0, len(seen))
	for index := range seen {
		indexes = append(indexes, index)
	}
	sort.Slice(indexes, func(i, j int) bool { return indexes[i] < indexes[j] })
	return indexes
}

func limitOrSortLimitOperator(operator Operator) bool {
	if operator.Family == "limit" {
		return true
	}
	return strings.HasSuffix(normalizeOperatorName(operator.DisplayName), "sort limit")
}

func streamBlockingOperatorFamily(family string) bool {
	switch family {
	case "aggregate",
		"bloom_filter_build",
		"blocking_operator",
		"full_sort",
		"hash_aggregate",
		"hash_join",
		"join",
		"push_broadcast_hash_join",
		"spool_build":
		return true
	default:
		return false
	}
}

func aggregateClassificationUnknownOperatorIndexes(query Query) []int32 {
	var indexes []int32
	for _, operator := range query.NormalizedOperators {
		if normalizeOperatorName(operator.DisplayName) == "aggregate" && operator.Family == "aggregate" {
			indexes = append(indexes, operator.Index)
		}
	}
	return indexes
}

func joinClassificationUnknownOperatorIndexes(query Query) []int32 {
	var indexes []int32
	for _, operator := range query.NormalizedOperators {
		if operator.Family == "join" {
			indexes = append(indexes, operator.Index)
		}
	}
	return indexes
}

func predicateClassificationUnknown(query Query, family string) bool {
	switch family {
	case "hash_aggregate", "stream_aggregate":
		return queryHasClassificationWarning(query, "aggregate_iterator_type_unknown")
	case "hash_join",
		"push_broadcast_hash_join",
		"apply_join",
		"semi_apply",
		"anti_semi_apply",
		"distributed_cross_apply",
		"distributed_semi_apply",
		"distributed_anti_semi_apply",
		"merge_join":
		return queryHasClassificationWarning(query, "join_family_unknown")
	default:
		return false
	}
}

func classificationUnknownDiagnosticID(family string) string {
	switch {
	case aggregateClassificationUnknownFamily(family):
		return "plan.aggregate_classification_unknown"
	case joinClassificationUnknownFamily(family):
		return "plan.join_classification_unknown"
	default:
		return "plan.operator_classification_unknown"
	}
}

func classificationUnknownRemediation(family string) []Remediation {
	switch {
	case aggregateClassificationUnknownFamily(family):
		return []Remediation{
			{
				Kind:       "operator_classification_unknown",
				AppliesTo:  "contract",
				Confidence: "medium",
				Message:    "This aggregate contract depends on hash/stream aggregate classification, but at least one Aggregate PlanNode has unknown iterator_type metadata.",
			},
		}
	case joinClassificationUnknownFamily(family):
		return []Remediation{
			{
				Kind:       "operator_classification_unknown",
				AppliesTo:  "contract",
				Confidence: "medium",
				Message:    "This join contract depends on a specific join-family classification, but at least one join-like PlanNode could not be classified.",
			},
		}
	default:
		return nil
	}
}

func aggregateClassificationUnknownFamily(family string) bool {
	switch family {
	case "hash_aggregate", "stream_aggregate":
		return true
	default:
		return false
	}
}

func joinClassificationUnknownFamily(family string) bool {
	switch family {
	case "hash_join",
		"push_broadcast_hash_join",
		"apply_join",
		"semi_apply",
		"anti_semi_apply",
		"distributed_cross_apply",
		"distributed_semi_apply",
		"distributed_anti_semi_apply",
		"merge_join":
		return true
	default:
		return false
	}
}

func queryHasClassificationWarning(query Query, id string) bool {
	for _, warning := range query.ClassificationNotes {
		if warning.ID == id {
			return true
		}
	}
	return false
}

func normalizeOperatorName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}

func remediationFor(family string) []Remediation {
	switch family {
	case "blocking_operator":
		return blockingOperatorUnderLimitRemediation()
	case "explicit_sort", "full_sort":
		return []Remediation{
			{
				Kind:       "config_change",
				AppliesTo:  "config",
				Confidence: "high",
				Message:    "For generated table/index shorthand, consider order_by: none if result order is not required.",
			},
			{
				Kind:       "index_design",
				AppliesTo:  "ddl",
				Confidence: "medium",
				Message:    "If ordering is required, consider an index whose key order satisfies the requested ORDER BY.",
			},
		}
	case "minor_sort":
		return []Remediation{
			{
				Kind:       "index_design",
				AppliesTo:  "ddl",
				Confidence: "medium",
				Message:    "If minor sort is not acceptable, consider an index whose full key order satisfies the requested ORDER BY or stream aggregate keys.",
			},
			{
				Kind:       "contract",
				AppliesTo:  "contract",
				Confidence: "medium",
				Message:    "If minor sort is acceptable for this query, prefer no_full_sort instead of no_explicit_sort.",
			},
		}
	case "hash_aggregate":
		return []Remediation{
			{
				Kind:       "sql_hint",
				AppliesTo:  "sql",
				Confidence: "medium",
				Message:    "If a stream aggregate is required, consider GROUP@{GROUP_METHOD=STREAM_GROUP} BY after reviewing the query plan.",
			},
			{
				Kind:       "index_design",
				AppliesTo:  "ddl",
				Confidence: "medium",
				Message:    "If stream aggregation is intended, consider an input order that satisfies the GROUP BY keys.",
			},
		}
	case "hash_join":
		return []Remediation{
			{
				Kind:       "sql_hint",
				AppliesTo:  "sql",
				Confidence: "medium",
				Message:    "If a hash join is not acceptable, review JOIN@{JOIN_METHOD=APPLY_JOIN} or JOIN@{JOIN_METHOD=MERGE_JOIN}; validate the resulting plan and latency before relying on it.",
			},
		}
	case "push_broadcast_hash_join":
		return []Remediation{
			{
				Kind:       "sql_hint",
				AppliesTo:  "sql",
				Confidence: "medium",
				Message:    "If push broadcast hash join is not acceptable, review JOIN@{JOIN_METHOD=HASH_JOIN}, APPLY_JOIN, or MERGE_JOIN and validate the resulting plan.",
			},
		}
	case "apply_join":
		return []Remediation{
			{
				Kind:       "sql_hint",
				AppliesTo:  "sql",
				Confidence: "medium",
				Message:    "If apply join is not acceptable, review JOIN@{JOIN_METHOD=HASH_JOIN} or MERGE_JOIN and validate the resulting plan.",
			},
		}
	case "distributed_cross_apply":
		return []Remediation{
			{
				Kind:       "sql_hint",
				AppliesTo:  "sql",
				Confidence: "medium",
				Message:    "If distributed cross apply is not acceptable, review JOIN@{JOIN_METHOD=HASH_JOIN}, PUSH_BROADCAST_HASH_JOIN, or MERGE_JOIN and validate the resulting plan.",
			},
		}
	case "merge_join":
		return []Remediation{
			{
				Kind:       "sql_hint",
				AppliesTo:  "sql",
				Confidence: "medium",
				Message:    "If merge join is not acceptable, review JOIN@{JOIN_METHOD=HASH_JOIN} or APPLY_JOIN; merge joins can introduce Sort operators when inputs are not already ordered.",
			},
		}
	case "stream_aggregate":
		return []Remediation{
			{
				Kind:       "sql_hint",
				AppliesTo:  "sql",
				Confidence: "medium",
				Message:    "If a hash aggregate is required, consider GROUP@{GROUP_METHOD=HASH_GROUP} BY after reviewing the query plan.",
			},
		}
	default:
		return nil
	}
}

func blockingOperatorUnderLimitRemediation() []Remediation {
	return []Remediation{
		{
			Kind:       "query_shape",
			AppliesTo:  "sql",
			Confidence: "medium",
			Message:    "Review blocking descendants under Limit or Sort Limit; prefer an order-preserving access path, stream aggregate, merge/apply join, or remove the LIMIT/ORDER BY requirement when it is not needed.",
		},
	}
}

func fullScanRemediation() []Remediation {
	return []Remediation{
		{
			Kind:       "index_design",
			AppliesTo:  "ddl",
			Confidence: "medium",
			Message:    "Review whether a table or secondary index key can satisfy the query predicates as Seek Conditions instead of a full scan.",
		},
		{
			Kind:       "query_shape",
			AppliesTo:  "sql",
			Confidence: "medium",
			Message:    "Check whether important filters are residual conditions; for sharded timestamp queries, prefer equality probes per shard when timestamp range seekability matters. For recent-data commit timestamp reads, a full scan with Timestamp Condition can still be intentional; consider require_timestamp_condition instead of no_full_scan for that case.",
		},
	}
}

func fullScanWithoutTimestampConditionRemediation() []Remediation {
	return []Remediation{
		{
			Kind:       "index_design",
			AppliesTo:  "ddl",
			Confidence: "medium",
			Message:    "Review whether a table or secondary index key can satisfy the query predicates as Seek Conditions, or whether this query should intentionally rely on commit timestamp predicate pushdown.",
		},
		{
			Kind:       "query_shape",
			AppliesTo:  "sql",
			Confidence: "medium",
			Message:    "If this is a recent-data commit timestamp read, make sure the predicate qualifies for Timestamp Condition; otherwise treat the full scan as unpruned.",
		},
	}
}

func timestampConditionRemediation() []Remediation {
	return []Remediation{
		{
			Kind:       "query_shape",
			AppliesTo:  "sql",
			Confidence: "medium",
			Message:    "Use a > or >= predicate on an allow_commit_timestamp column with a constant expression, and avoid OR in the qualifying predicate.",
		},
		{
			Kind:       "contract",
			AppliesTo:  "contract",
			Confidence: "medium",
			Message:    "Use this contract only for queries where commit timestamp predicate pushdown is expected; older-data or non-commit-timestamp filters may legitimately lack Timestamp Condition.",
		},
	}
}

// EnvironmentWarnings returns warnings that affect contract interpretation.
func EnvironmentWarnings(report Report) []string {
	seen := map[string]bool{}
	var warnings []string
	add := func(warning string) {
		if !seen[warning] {
			seen[warning] = true
			warnings = append(warnings, warning)
		}
	}
	if strings.EqualFold(report.Optimizer.Requested.Version, "not_pinned") {
		add("optimizer_not_pinned")
	}
	if strings.EqualFold(report.Optimizer.Requested.StatisticsPackage, "not_pinned") {
		add("statistics_package_not_pinned")
	}
	if strings.EqualFold(report.BackendIdentity.Version, "not_recorded") || strings.EqualFold(report.BackendIdentity.ImageDigest, "not_recorded") {
		add("backend_identity_not_recorded")
	}
	for _, query := range report.Queries {
		if query.OptimizerNotPinned {
			add("query_optimizer_not_pinned")
		}
		if len(query.ClassificationNotes) > 0 {
			add("operator_classification_unknown")
		}
	}
	return warnings
}

// AddCheckWarnings mutates summary with warnings that only apply to --check.
func AddCheckWarnings(summary *EvaluationSummary, evaluations []Evaluation) {
	if summary == nil {
		return
	}
	for _, evaluation := range evaluations {
		if evaluation.Stability.Tier == StabilityRawPlan {
			summary.EnvironmentWarnings = appendUniqueString(summary.EnvironmentWarnings, "raw_query_plan_contract_used_in_check")
			return
		}
	}
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

// OptimizerPinningWarnings returns environment warnings related to optimizer pinning.
func OptimizerPinningWarnings(report Report) []string {
	allWarnings := EnvironmentWarnings(report)
	var warnings []string
	for _, warning := range allWarnings {
		switch warning {
		case "optimizer_not_pinned", "statistics_package_not_pinned", "query_optimizer_not_pinned":
			warnings = append(warnings, warning)
		}
	}
	return warnings
}

// Summarize summarizes contract evaluation results.
func Summarize(evaluations []Evaluation) EvaluationSummary {
	summary := EvaluationSummary{
		Status:              StatusPass,
		Contracts:           len(evaluations),
		EnvironmentWarnings: []string{},
	}
	for _, evaluation := range evaluations {
		switch evaluation.Status {
		case StatusFail:
			summary.Failed++
			summary.Status = StatusFail
		case StatusNotEvaluated:
			summary.NotEvaluated++
			summary.Status = StatusFail
		default:
			summary.Passed++
		}
	}
	return summary
}

// ViolationCount counts failed rule results.
func ViolationCount(evaluations []Evaluation) int {
	count := 0
	for _, evaluation := range evaluations {
		for _, result := range evaluation.Results {
			if result.Status == StatusFail {
				count++
			}
		}
	}
	return count
}

// CheckFailureCount counts failed or not-evaluated contracts.
func CheckFailureCount(evaluations []Evaluation) int {
	count := 0
	for _, evaluation := range evaluations {
		if evaluation.Status != StatusPass {
			count++
		}
	}
	return count
}

// OperatorFamilyCounts returns a complete zero-filled operator family count map.
func OperatorFamilyCounts(operators []Operator) map[string]int {
	counts := ZeroOperatorFamilyCounts()
	for _, operator := range operators {
		if operator.Family != "" {
			counts[operator.Family]++
		}
	}
	AddDerivedOperatorFamilyCounts(counts)
	return counts
}

// OperatorFamilyCountsOrEmpty returns a complete zero-filled copy of counts.
func OperatorFamilyCountsOrEmpty(counts map[string]int) map[string]int {
	out := ZeroOperatorFamilyCounts()
	for family, count := range counts {
		out[family] = count
	}
	AddDerivedOperatorFamilyCounts(out)
	return out
}

// AddDerivedOperatorFamilyCounts updates count-only umbrella families in counts.
func AddDerivedOperatorFamilyCounts(counts map[string]int) {
	counts["explicit_sort"] = counts["full_sort"] + counts["minor_sort"]
	blockingCount := 0
	for family, count := range counts {
		if family == "blocking_operator" {
			continue
		}
		if streamBlockingOperatorFamily(family) {
			blockingCount += count
		}
	}
	counts["blocking_operator"] = blockingCount
}

// ZeroOperatorFamilyCounts returns a complete zero-filled operator family map.
func ZeroOperatorFamilyCounts() map[string]int {
	counts := make(map[string]int, len(KnownOperatorFamilies()))
	for _, family := range KnownOperatorFamilies() {
		counts[family] = 0
	}
	return counts
}

// KnownOperatorFamilies returns all normalized operator families.
func KnownOperatorFamilies() []string {
	return []string{
		"aggregate",
		"anti_semi_apply",
		"apply_mutations",
		"array_subquery",
		"array_unnest",
		"bloom_filter_build",
		"blocking_operator",
		"change_stream_tvf",
		"compute",
		"compute_struct",
		"create_batch",
		"data_block_to_row",
		"apply_join",
		"distributed_anti_semi_apply",
		"distributed_anti_semi_apply_internal_apply",
		"distributed_merge_union",
		"distributed_cross_apply",
		"distributed_cross_apply_internal_apply",
		"distributed_semi_apply",
		"distributed_semi_apply_internal_apply",
		"distributed_union",
		"empty_relation",
		"explicit_sort",
		"filter",
		"filter_scan",
		"full_sort",
		"hash_aggregate",
		"hash_join",
		"join",
		"key_range_accumulator",
		"limit",
		"merge_join",
		"mini_batch_assign",
		"mini_batch_key_order",
		"minor_sort",
		"push_broadcast_hash_join",
		"push_broadcast_hash_join_internal_hash_join",
		"recursive_spool_scan",
		"random_id_assign",
		"recursive_union",
		"row_count",
		"row_to_data_block",
		"scan",
		"scalar_subquery",
		"search_predicate",
		"search_query_conversion_tvf",
		"serialize_result",
		"semi_apply",
		"spool_build",
		"spool_scan",
		"stream_aggregate",
		"union_all",
		"union_input",
		"unit_relation",
		"verify_determinism",
		"unknown",
	}
}

// ConcreteOperatorFamilies returns operator families that can appear directly
// in normalized_operators[].family. Derived umbrella families such as
// explicit_sort and blocking_operator are intentionally omitted.
func ConcreteOperatorFamilies() []string {
	out := make([]string, 0, len(KnownOperatorFamilies()))
	for _, family := range KnownOperatorFamilies() {
		if family == "explicit_sort" || family == "blocking_operator" {
			continue
		}
		out = append(out, family)
	}
	return out
}

// KnownOperatorFamily reports whether family is a supported normalized family.
func KnownOperatorFamily(family string) bool {
	for _, known := range KnownOperatorFamilies() {
		if family == known {
			return true
		}
	}
	return false
}

// PredefinedNames returns all predefined contract names.
func PredefinedNames() []string {
	return []string{
		"no_explicit_sort",
		"no_full_sort",
		"no_minor_sort",
		"no_hash_join",
		"no_standalone_hash_join",
		"no_push_broadcast_hash_join",
		"no_apply_join",
		"no_standalone_apply_join",
		"no_distributed_cross_apply",
		"no_merge_join",
		"no_hash_aggregate",
		"no_stream_aggregate",
		predefinedNoFullScan,
		predefinedNoFullScanWithoutTimestamp,
		predefinedRequireTimestampCondition,
		predefinedNoBlockingOperatorUnderLimit,
	}
}
