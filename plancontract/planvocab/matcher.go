package planvocab

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/apstndb/spanalyzer/plancontract"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// VariablePresence selects whether a matching child link has a variable.
// Variable names are never part of a pattern or diagnostic.
type VariablePresence uint8

const (
	// VariableAny accepts child links with or without a variable.
	VariableAny VariablePresence = iota
	// VariableAbsent accepts only child links without a variable.
	VariableAbsent
	// VariablePresent accepts only child links with a variable.
	VariablePresent
)

// OperatorPattern requires all metadata and child-link conditions to hold on
// the same PlanNode. DisplayName and Family are optional individually, but at
// least one must be set. When both are set, both must match. DisplayName is
// exact and case-sensitive.
type OperatorPattern struct {
	DisplayName string
	Family      string
	Metadata    []MetadataRequirement
	ChildLinks  []ChildLinkRequirement
}

// MetadataRequirement matches one non-null metadata field. A nil Value checks
// only for presence; a non-nil Value uses typed protobuf equality.
type MetadataRequirement struct {
	Key   string
	Value *structpb.Value
}

// ChildLinkRequirement matches child-link entries on one operator. Kind
// KIND_UNSPECIFIED accepts dangling links and links of any target kind. Type is
// exact and case-sensitive, including the empty untyped value. MinCount zero
// means one.
type ChildLinkRequirement struct {
	Kind     spannerpb.PlanNode_Kind
	Type     string
	Variable VariablePresence
	MinCount int
}

// CandidateMismatch explains why an operator candidate did not satisfy a
// pattern. Reasons are deterministic and redact metadata values and child
// variable names.
type CandidateMismatch struct {
	NodeIndex int32    `json:"node_index"`
	Reasons   []string `json:"reasons"`
}

// MatchResult contains all matching nodes and redacted diagnostics for
// operator candidates that did not match.
type MatchResult struct {
	NodeIndexes []int32             `json:"node_indexes"`
	Mismatches  []CandidateMismatch `json:"mismatches"`
}

// HasMatches reports whether at least one PlanNode satisfied the pattern.
func (r MatchResult) HasMatches() bool {
	return len(r.NodeIndexes) != 0
}

// String returns a deterministic, redacted summary suitable for test errors.
func (r MatchResult) String() string {
	if r.HasMatches() {
		return fmt.Sprintf("matched plan nodes %v", r.NodeIndexes)
	}
	if len(r.Mismatches) == 0 {
		return "no operator candidates matched the pattern"
	}
	parts := make([]string, 0, len(r.Mismatches))
	for _, candidate := range r.Mismatches {
		parts = append(parts, fmt.Sprintf("node %d: %s", candidate.NodeIndex, strings.Join(candidate.Reasons, "; ")))
	}
	return "no plan node matched: " + strings.Join(parts, " | ")
}

// FindMatchingOperators finds every PlanNode that satisfies pattern. Family
// selection delegates entirely to plancontract.NormalizeOperators so
// planvocab does not maintain a second operator-family authority.
func FindMatchingOperators(plan *spannerpb.QueryPlan, pattern OperatorPattern) (MatchResult, error) {
	if err := validatePattern(pattern); err != nil {
		return MatchResult{}, err
	}
	result := MatchResult{
		NodeIndexes: []int32{},
		Mismatches:  []CandidateMismatch{},
	}
	if plan == nil {
		return result, nil
	}

	nodesByIndex := make(map[int32]*spannerpb.PlanNode, len(plan.GetPlanNodes()))
	for _, node := range plan.GetPlanNodes() {
		if node != nil {
			nodesByIndex[node.GetIndex()] = node
		}
	}
	familiesByIndex := make(map[int32]string, len(plan.GetPlanNodes()))
	for _, operator := range plancontract.NormalizeOperators(plan) {
		familiesByIndex[operator.Index] = operator.Family
	}
	for _, node := range sortedNodes(plan.GetPlanNodes()) {
		if pattern.DisplayName != "" && node.GetDisplayName() != pattern.DisplayName {
			continue
		}
		if pattern.Family != "" && familiesByIndex[node.GetIndex()] != pattern.Family {
			continue
		}
		reasons := matchNodeRequirements(node, nodesByIndex, pattern)
		if len(reasons) == 0 {
			result.NodeIndexes = append(result.NodeIndexes, node.GetIndex())
			continue
		}
		result.Mismatches = append(result.Mismatches, CandidateMismatch{
			NodeIndex: node.GetIndex(),
			Reasons:   reasons,
		})
	}
	return result, nil
}

func validatePattern(pattern OperatorPattern) error {
	if pattern.DisplayName == "" && pattern.Family == "" {
		return errors.New("operator pattern requires DisplayName or Family")
	}
	if pattern.Family != "" && !slices.Contains(plancontract.ConcreteOperatorFamilies(), pattern.Family) {
		return fmt.Errorf("operator pattern family %q is not a concrete plancontract family", pattern.Family)
	}
	for _, requirement := range pattern.Metadata {
		if requirement.Key == "" {
			return errors.New("metadata requirement has an empty key")
		}
		if requirement.Value != nil {
			if requirement.Value.GetKind() == nil {
				return fmt.Errorf("metadata requirement %q has an unset protobuf value", requirement.Key)
			}
			if _, isNull := requirement.Value.GetKind().(*structpb.Value_NullValue); isNull {
				return fmt.Errorf("metadata requirement %q cannot match a null value", requirement.Key)
			}
		}
	}
	for _, requirement := range pattern.ChildLinks {
		if _, known := spannerpb.PlanNode_Kind_name[int32(requirement.Kind)]; !known {
			return fmt.Errorf("child-link requirement has unknown PlanNode kind %d", requirement.Kind)
		}
		if requirement.Variable > VariablePresent {
			return fmt.Errorf("child-link requirement has invalid variable presence %d", requirement.Variable)
		}
		if requirement.MinCount < 0 {
			return fmt.Errorf("child-link requirement has negative MinCount %d", requirement.MinCount)
		}
	}
	return nil
}

func matchNodeRequirements(
	node *spannerpb.PlanNode,
	nodesByIndex map[int32]*spannerpb.PlanNode,
	pattern OperatorPattern,
) []string {
	reasons := make([]string, 0, len(pattern.Metadata)+len(pattern.ChildLinks))
	for _, requirement := range pattern.Metadata {
		value, found := node.GetMetadata().GetFields()[requirement.Key]
		if !found || value == nil || value.GetKind() == nil {
			reasons = append(reasons, fmt.Sprintf("metadata key %q is missing", requirement.Key))
			continue
		}
		if _, isNull := value.GetKind().(*structpb.Value_NullValue); isNull {
			reasons = append(reasons, fmt.Sprintf("metadata key %q is null", requirement.Key))
			continue
		}
		if requirement.Value != nil && !proto.Equal(value, requirement.Value) {
			reasons = append(reasons, fmt.Sprintf(
				"metadata key %q did not match (observed %s)",
				requirement.Key,
				safeMetadataDiagnosticValue(node, requirement.Key, value),
			))
		}
	}
	for _, requirement := range pattern.ChildLinks {
		count := matchingChildLinkCount(node, nodesByIndex, requirement)
		minimum := requirement.MinCount
		if minimum == 0 {
			minimum = 1
		}
		if count < minimum {
			reasons = append(reasons, fmt.Sprintf(
				"child link kind=%s type=%q variable=%s count=%d, want at least %d",
				childRequirementKind(requirement.Kind),
				requirement.Type,
				requirement.Variable,
				count,
				minimum,
			))
		}
	}
	sort.Strings(reasons)
	return reasons
}

func matchingChildLinkCount(
	node *spannerpb.PlanNode,
	nodesByIndex map[int32]*spannerpb.PlanNode,
	requirement ChildLinkRequirement,
) int {
	count := 0
	for _, link := range node.GetChildLinks() {
		if link == nil || link.GetType() != requirement.Type {
			continue
		}
		hasVariable := strings.TrimSpace(link.GetVariable()) != ""
		if requirement.Variable == VariableAbsent && hasVariable {
			continue
		}
		if requirement.Variable == VariablePresent && !hasVariable {
			continue
		}
		if requirement.Kind != spannerpb.PlanNode_KIND_UNSPECIFIED {
			child, found := nodesByIndex[link.GetChildIndex()]
			if !found || child.GetKind() != requirement.Kind {
				continue
			}
		}
		count++
	}
	return count
}

func safeMetadataDiagnosticValue(node *spannerpb.PlanNode, key string, value *structpb.Value) string {
	// Assertion diagnostics redact enum values that the catalog does not yet
	// know. Inspect intentionally preserves those normalized enum values so an
	// observer finding can identify vocabulary that needs catalog review.
	rule, knownOperator := knownCatalog.byName[normalizeToken(node.GetDisplayName())]
	metadataRule, knownKey := metadataRulesFor(rule, knownOperator)[key]
	if knownKey && len(metadataRule.Values) > 0 && knownMetadataValue(value, metadataRule.Values) {
		return normalizeMetadataValue(value)
	}
	return metadataValueKind(value)
}

func childRequirementKind(kind spannerpb.PlanNode_Kind) string {
	if kind == spannerpb.PlanNode_KIND_UNSPECIFIED {
		return "any"
	}
	return kind.String()
}

func (presence VariablePresence) String() string {
	switch presence {
	case VariableAny:
		return "any"
	case VariableAbsent:
		return "absent"
	case VariablePresent:
		return "present"
	default:
		return fmt.Sprintf("unknown(%d)", presence)
	}
}
