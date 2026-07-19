package planvocab

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/apstndb/spanalyzer/plancontract"
	"google.golang.org/protobuf/types/known/structpb"
)

const logEvent = "spanner_query_plan_unknown_vocabulary"

const suppressionLogEvent = "spanner_query_plan_unknown_vocabulary_suppressed"

// DefaultMaxFingerprints bounds the per-observer deduplication set.
const DefaultMaxFingerprints = 1024

//go:embed catalog.json
var catalogData []byte

// CatalogInfo identifies the observational sources used by the embedded
// catalog. The catalog version is independent of the plancontract module
// version because its vocabulary can grow without changing the inspection API.
type CatalogInfo struct {
	Version       string         `json:"version"`
	Repository    string         `json:"repository"`
	Revision      string         `json:"revision"`
	Path          string         `json:"path"`
	Blob          string         `json:"blob"`
	LocalEvidence []string       `json:"local_evidence"`
	Compatibility string         `json:"compatibility"`
	GeneratedBy   string         `json:"generated_by"`
	Inputs        []CatalogInput `json:"inputs"`
}

// CatalogInput records the digest of one checked-in generation input.
type CatalogInput struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// MetadataShape is a redacted metadata component in an observed PlanNode
// shape. Identifier-like values are represented by their protobuf value kind.
type MetadataShape struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ChildLinkShape is a redacted child-link component in an observed PlanNode
// shape. Exact counts and variable names are omitted; Repeated records whether
// the same kind/type/variable-presence tuple occurred more than once.
type ChildLinkShape struct {
	Kind        string `json:"kind"`
	Type        string `json:"type"`
	HasVariable bool   `json:"has_variable"`
	Repeated    bool   `json:"repeated"`
}

// Finding describes one PlanNode whose operator vocabulary is not fully
// covered by the embedded catalog.
type Finding struct {
	NodeIndex             int32                     `json:"node_index"`
	DisplayName           string                    `json:"operator"`
	Kind                  string                    `json:"kind"`
	Fingerprint           string                    `json:"fingerprint"`
	Reasons               []string                  `json:"reasons"`
	Metadata              []MetadataShape           `json:"metadata"`
	ChildLinks            []ChildLinkShape          `json:"child_links"`
	UnknownMetadataKeys   []string                  `json:"unknown_metadata_keys,omitempty"`
	UnknownMetadataValues []MetadataShape           `json:"unknown_metadata_values,omitempty"`
	UnknownChildLinks     []ChildLinkShape          `json:"unknown_child_links,omitempty"`
	Diagnostics           []plancontract.Diagnostic `json:"diagnostics,omitempty"`
}

// ObserverOptions configures logging and the bounded fingerprint set.
type ObserverOptions struct {
	Logger          *slog.Logger
	MaxFingerprints int
}

// Observer logs each unknown fingerprint at most once until its bounded set is
// full. Further fingerprints are counted and one suppression record is logged.
// Observer is safe for concurrent use.
type Observer struct {
	logger            *slog.Logger
	maxFingerprints   int
	mu                sync.Mutex
	seen              map[string]struct{}
	suppressed        uint64
	suppressionLogged bool
}

type embeddedCatalog struct {
	Info           CatalogInfo    `json:"info"`
	CommonMetadata []metadataRule `json:"common_metadata"`
	Operators      []operatorRule `json:"operators"`
	byName         map[string]operatorRule
}

type operatorRule struct {
	Names      []string       `json:"names"`
	Kind       string         `json:"kind"`
	Metadata   []metadataRule `json:"metadata"`
	ChildLinks []childRule    `json:"child_links"`
}

type metadataRule struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

type childRule struct {
	Kind     string `json:"kind"`
	Type     string `json:"type"`
	Variable string `json:"variable"`
	Multiple *bool  `json:"multiple"`
}

var knownCatalog = mustLoadCatalog(catalogData)

// EmbeddedCatalogInfo returns a copy of the provenance and compatibility
// boundary recorded by the embedded catalog.
func EmbeddedCatalogInfo() CatalogInfo {
	info := knownCatalog.Info
	info.LocalEvidence = append([]string{}, info.LocalEvidence...)
	info.Inputs = append([]CatalogInput{}, info.Inputs...)
	return info
}

// NewObserver creates an observer that emits structured warning records.
// A nil logger uses slog.Default.
func NewObserver(logger *slog.Logger) *Observer {
	return NewObserverWithOptions(ObserverOptions{Logger: logger})
}

// NewObserverWithOptions creates an observer with an explicit fingerprint
// bound. Non-positive bounds use DefaultMaxFingerprints.
func NewObserverWithOptions(options ObserverOptions) *Observer {
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}
	maxFingerprints := options.MaxFingerprints
	if maxFingerprints <= 0 {
		maxFingerprints = DefaultMaxFingerprints
	}
	return &Observer{
		logger:          logger,
		maxFingerprints: maxFingerprints,
		seen:            make(map[string]struct{}, maxFingerprints),
	}
}

// Observe inspects plan, logs each previously unseen unknown fingerprint, and
// returns all findings. Identifier-like metadata values and child variables are
// redacted before both fingerprinting and logging.
func (o *Observer) Observe(ctx context.Context, plan *spannerpb.QueryPlan) []Finding {
	findings := Inspect(plan)
	for _, finding := range findings {
		logFinding, logSuppression, suppressedCount := o.recordFingerprint(finding.Fingerprint)
		if logSuppression {
			o.logger.LogAttrs(
				ctx,
				slog.LevelWarn,
				"additional unknown Spanner query-plan vocabulary suppressed",
				slog.String("event", suppressionLogEvent),
				slog.Int("max_fingerprints", o.maxFingerprints),
				slog.Uint64("suppressed_count", suppressedCount),
			)
		}
		if !logFinding {
			continue
		}
		o.logger.LogAttrs(
			ctx,
			slog.LevelWarn,
			"unknown Spanner query-plan vocabulary",
			slog.String("event", logEvent),
			slog.Int64("node_index", int64(finding.NodeIndex)),
			slog.String("operator", finding.DisplayName),
			slog.String("kind", finding.Kind),
			slog.String("fingerprint", finding.Fingerprint),
			slog.Any("reasons", finding.Reasons),
			slog.Any("metadata", finding.Metadata),
			slog.Any("child_links", finding.ChildLinks),
			slog.Any("unknown_metadata_keys", finding.UnknownMetadataKeys),
			slog.Any("unknown_metadata_values", finding.UnknownMetadataValues),
			slog.Any("unknown_child_links", finding.UnknownChildLinks),
			slog.Any("diagnostics", finding.Diagnostics),
		)
	}
	return findings
}

// SuppressedCount returns the number of findings not added to the bounded
// fingerprint set after it became full.
func (o *Observer) SuppressedCount() uint64 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.suppressed
}

func (o *Observer) recordFingerprint(fingerprint string) (bool, bool, uint64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if _, duplicate := o.seen[fingerprint]; duplicate {
		return false, false, o.suppressed
	}
	if len(o.seen) < o.maxFingerprints {
		o.seen[fingerprint] = struct{}{}
		return true, false, o.suppressed
	}
	o.suppressed++
	logSuppression := !o.suppressionLogged
	o.suppressionLogged = true
	return false, logSuppression, o.suppressed
}

// Inspect returns one aggregated finding for each PlanNode that uses operator
// vocabulary outside the embedded catalog. It does not mutate plan or log.
func Inspect(plan *spannerpb.QueryPlan) []Finding {
	if plan == nil {
		return []Finding{}
	}

	nodesByIndex := make(map[int32]*spannerpb.PlanNode, len(plan.GetPlanNodes()))
	for _, node := range plan.GetPlanNodes() {
		if node != nil {
			nodesByIndex[node.GetIndex()] = node
		}
	}

	findings := []Finding{}
	operatorsByIndex := make(map[int32]plancontract.Operator, len(plan.GetPlanNodes()))
	for _, operator := range plancontract.NormalizeOperators(plan) {
		operatorsByIndex[operator.Index] = operator
	}
	for _, node := range sortedNodes(plan.GetPlanNodes()) {
		finding, unknown := inspectNode(
			node,
			nodesByIndex,
			operatorsByIndex[node.GetIndex()],
		)
		if unknown {
			findings = append(findings, finding)
		}
	}
	return findings
}

func inspectNode(
	node *spannerpb.PlanNode,
	nodesByIndex map[int32]*spannerpb.PlanNode,
	operator plancontract.Operator,
) (Finding, bool) {
	finding := Finding{
		NodeIndex:   node.GetIndex(),
		DisplayName: strings.TrimSpace(node.GetDisplayName()),
		Kind:        node.GetKind().String(),
		Reasons:     []string{},
		Metadata:    []MetadataShape{},
		ChildLinks:  []ChildLinkShape{},
	}

	warnings := plancontract.ClassificationWarnings([]plancontract.Operator{operator})
	for _, warning := range warnings {
		finding.Reasons = append(finding.Reasons, warning.ID)
	}
	finding.Diagnostics = append([]plancontract.Diagnostic{}, warnings...)

	rule, knownOperator := knownCatalog.byName[normalizeToken(node.GetDisplayName())]
	if !knownOperator && len(warnings) == 0 {
		finding.Reasons = append(finding.Reasons, "catalog_entry_missing")
	}
	if knownOperator && !kindMatches(rule.Kind, node.GetKind()) {
		finding.Reasons = append(finding.Reasons, "unexpected_operator_kind")
	}

	metadataRules := metadataRulesFor(rule, knownOperator)
	inspectMetadata(node.GetMetadata(), metadataRules, knownOperator, &finding)
	inspectChildLinks(node, nodesByIndex, rule.ChildLinks, knownOperator, &finding)

	finding.Reasons = sortedUniqueStrings(finding.Reasons)
	finding.Metadata = sortedUniqueMetadata(finding.Metadata)
	finding.ChildLinks = aggregateChildLinks(finding.ChildLinks)
	finding.UnknownMetadataKeys = sortedUniqueStrings(finding.UnknownMetadataKeys)
	finding.UnknownMetadataValues = sortedUniqueMetadata(finding.UnknownMetadataValues)
	finding.UnknownChildLinks = aggregateChildLinks(finding.UnknownChildLinks)

	if len(finding.Reasons) == 0 {
		return Finding{}, false
	}
	finding.Fingerprint = findingFingerprint(finding)
	return finding, true
}

func inspectMetadata(
	metadata *structpb.Struct,
	rules map[string]metadataRule,
	knownOperator bool,
	finding *Finding,
) {
	if metadata == nil {
		return
	}
	keys := make([]string, 0, len(metadata.GetFields()))
	for key := range metadata.GetFields() {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := metadata.GetFields()[key]
		if value == nil || value.GetKind() == nil {
			continue
		}
		if _, isNull := value.GetKind().(*structpb.Value_NullValue); isNull {
			continue
		}
		rule, knownKey := rules[key]
		shape := MetadataShape{Key: key, Value: metadataShapeValue(value, rule, knownKey)}
		finding.Metadata = append(finding.Metadata, shape)
		if !knownOperator {
			continue
		}
		if !knownKey {
			finding.Reasons = append(finding.Reasons, "unknown_metadata_key")
			finding.UnknownMetadataKeys = append(finding.UnknownMetadataKeys, key)
			continue
		}
		if len(rule.Values) > 0 && !knownMetadataValue(value, rule.Values) {
			finding.Reasons = append(finding.Reasons, "unknown_metadata_value")
			finding.UnknownMetadataValues = append(finding.UnknownMetadataValues, shape)
		}
	}
}

func inspectChildLinks(
	node *spannerpb.PlanNode,
	nodesByIndex map[int32]*spannerpb.PlanNode,
	rules []childRule,
	knownOperator bool,
	finding *Finding,
) {
	observed := []ChildLinkShape{}
	for _, link := range node.GetChildLinks() {
		if link == nil {
			continue
		}
		child, found := nodesByIndex[link.GetChildIndex()]
		if !found {
			continue
		}
		observed = append(observed, ChildLinkShape{
			Kind:        child.GetKind().String(),
			Type:        strings.TrimSpace(link.GetType()),
			HasVariable: strings.TrimSpace(link.GetVariable()) != "",
		})
	}
	finding.ChildLinks = aggregateChildLinks(observed)
	if !knownOperator {
		return
	}
	for _, shape := range finding.ChildLinks {
		rule, known := matchChildLink(shape, rules)
		if !known {
			finding.Reasons = append(finding.Reasons, "unknown_child_link")
			finding.UnknownChildLinks = append(finding.UnknownChildLinks, shape)
			continue
		}
		if shape.Repeated && rule.Multiple != nil && !*rule.Multiple {
			finding.Reasons = append(finding.Reasons, "unexpected_child_link_multiplicity")
			finding.UnknownChildLinks = append(finding.UnknownChildLinks, shape)
		}
	}
}

func metadataRulesFor(rule operatorRule, knownOperator bool) map[string]metadataRule {
	rules := make(map[string]metadataRule, len(knownCatalog.CommonMetadata)+len(rule.Metadata))
	for _, metadata := range knownCatalog.CommonMetadata {
		rules[metadata.Key] = metadata
	}
	if knownOperator {
		for _, metadata := range rule.Metadata {
			rules[metadata.Key] = metadata
		}
	}
	return rules
}

func metadataShapeValue(value *structpb.Value, rule metadataRule, knownKey bool) string {
	if knownKey && len(rule.Values) > 0 {
		return normalizeMetadataValue(value)
	}
	return metadataValueKind(value)
}

func knownMetadataValue(value *structpb.Value, known []string) bool {
	actual := normalizeMetadataValue(value)
	for _, candidate := range known {
		if actual == normalizeToken(candidate) {
			return true
		}
	}
	return false
}

func normalizeMetadataValue(value *structpb.Value) string {
	switch kind := value.GetKind().(type) {
	case *structpb.Value_StringValue:
		return normalizeToken(kind.StringValue)
	case *structpb.Value_NumberValue:
		return strconv.FormatFloat(kind.NumberValue, 'g', -1, 64)
	case *structpb.Value_BoolValue:
		return strconv.FormatBool(kind.BoolValue)
	case *structpb.Value_NullValue:
		return "null"
	case *structpb.Value_ListValue:
		return "list"
	case *structpb.Value_StructValue:
		return "object"
	default:
		return "unknown"
	}
}

func metadataValueKind(value *structpb.Value) string {
	switch value.GetKind().(type) {
	case *structpb.Value_StringValue:
		return "string"
	case *structpb.Value_NumberValue:
		return "number"
	case *structpb.Value_BoolValue:
		return "bool"
	case *structpb.Value_NullValue:
		return "null"
	case *structpb.Value_ListValue:
		return "list"
	case *structpb.Value_StructValue:
		return "object"
	default:
		return "unknown"
	}
}

func matchChildLink(shape ChildLinkShape, rules []childRule) (childRule, bool) {
	for _, rule := range rules {
		kindUnspecified := shape.Kind == spannerpb.PlanNode_KIND_UNSPECIFIED.String()
		kindMatches := kindUnspecified || strings.EqualFold(shape.Kind, rule.Kind)
		typeMatches := matchRuleToken(shape.Type, rule.Type)
		variableMatches := rule.Variable == "any" || (rule.Variable == "present") == shape.HasVariable
		if kindMatches && typeMatches && variableMatches {
			return rule, true
		}
	}
	return childRule{}, false
}

func matchRuleToken(actual, rule string) bool {
	actual = normalizeToken(actual)
	rule = normalizeToken(rule)
	if strings.HasSuffix(rule, "*") {
		return strings.HasPrefix(actual, strings.TrimSuffix(rule, "*"))
	}
	return actual == rule
}

func kindMatches(known string, actual spannerpb.PlanNode_Kind) bool {
	return actual == spannerpb.PlanNode_KIND_UNSPECIFIED || strings.EqualFold(known, actual.String())
}

func findingFingerprint(finding Finding) string {
	payload := struct {
		Operator   string           `json:"operator"`
		Kind       string           `json:"kind"`
		Reasons    []string         `json:"reasons"`
		Metadata   []MetadataShape  `json:"metadata"`
		ChildLinks []ChildLinkShape `json:"child_links"`
	}{
		Operator:   normalizeToken(finding.DisplayName),
		Kind:       strings.ToLower(finding.Kind),
		Reasons:    finding.Reasons,
		Metadata:   finding.Metadata,
		ChildLinks: finding.ChildLinks,
	}
	data, err := json.Marshal(payload)
	// payload contains only strings and slices, so json.Marshal cannot fail;
	// keep the branch as a guard if fields with fallible encodings are added.
	if err != nil {
		panic(fmt.Sprintf("marshal plan-vocabulary fingerprint: %v", err))
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func mustLoadCatalog(data []byte) embeddedCatalog {
	var catalog embeddedCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		panic(fmt.Sprintf("load embedded plan-vocabulary catalog: %v", err))
	}
	missingProvenance := catalog.Info.GeneratedBy == "" || len(catalog.Info.Inputs) == 0
	if catalog.Info.Version == "" || len(catalog.Operators) == 0 || missingProvenance {
		panic("load embedded plan-vocabulary catalog: missing version, provenance, or operators")
	}
	catalog.byName = make(map[string]operatorRule)
	for _, operator := range catalog.Operators {
		if operator.Kind == "" || len(operator.Names) == 0 {
			panic("load embedded plan-vocabulary catalog: operator missing kind or names")
		}
		for _, child := range operator.ChildLinks {
			switch child.Variable {
			case "absent", "present", "any":
			default:
				panic(fmt.Sprintf("load embedded plan-vocabulary catalog: invalid variable policy %q", child.Variable))
			}
		}
		for _, name := range operator.Names {
			key := normalizeToken(name)
			if _, duplicate := catalog.byName[key]; duplicate {
				panic(fmt.Sprintf("load embedded plan-vocabulary catalog: duplicate operator %q", name))
			}
			catalog.byName[key] = operator
		}
	}
	return catalog
}

func normalizeToken(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func sortedNodes(nodes []*spannerpb.PlanNode) []*spannerpb.PlanNode {
	result := make([]*spannerpb.PlanNode, 0, len(nodes))
	for _, node := range nodes {
		if node != nil {
			result = append(result, node)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].GetIndex() < result[j].GetIndex()
	})
	return result
}

func sortedUniqueStrings(values []string) []string {
	result := append([]string{}, values...)
	sort.Strings(result)
	return compactStrings(result)
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}

func sortedUniqueMetadata(values []MetadataShape) []MetadataShape {
	result := append([]MetadataShape{}, values...)
	sort.Slice(result, func(i, j int) bool {
		if result[i].Key != result[j].Key {
			return result[i].Key < result[j].Key
		}
		return result[i].Value < result[j].Value
	})
	if len(result) == 0 {
		return []MetadataShape{}
	}
	compacted := result[:1]
	for _, value := range result[1:] {
		if value != compacted[len(compacted)-1] {
			compacted = append(compacted, value)
		}
	}
	return compacted
}

func aggregateChildLinks(values []ChildLinkShape) []ChildLinkShape {
	result := append([]ChildLinkShape{}, values...)
	sort.Slice(result, func(i, j int) bool {
		if result[i].Kind != result[j].Kind {
			return result[i].Kind < result[j].Kind
		}
		if result[i].Type != result[j].Type {
			return result[i].Type < result[j].Type
		}
		return !result[i].HasVariable && result[j].HasVariable
	})
	if len(result) == 0 {
		return []ChildLinkShape{}
	}
	compacted := result[:1]
	for _, value := range result[1:] {
		last := &compacted[len(compacted)-1]
		if sameChildLinkBase(value, *last) {
			last.Repeated = true
			continue
		}
		compacted = append(compacted, value)
	}
	return compacted
}

func sameChildLinkBase(left, right ChildLinkShape) bool {
	return left.Kind == right.Kind && left.Type == right.Type && left.HasVariable == right.HasVariable
}
