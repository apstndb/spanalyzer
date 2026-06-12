package plancontract

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

// This file normalizes a raw google.spanner.v1.QueryPlan into the
// contract-evaluator input: normalized operators with operator families,
// operator edges, observed families, classification warnings, and the
// operator tree digest. It is independent of how the plan was obtained
// (Spanner, Spanner emulator, Spanner Omni, or a serialized plan).

// digest returns the hex-encoded SHA-256 of s.
func digest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func NormalizeOperators(plan *spannerpb.QueryPlan) []Operator {
	if plan == nil {
		return nil
	}
	operatorContexts := operatorContexts(plan)
	childrenByIndex := childrenByIndex(plan)
	familyByIndex := make(map[int32]string, len(plan.GetPlanNodes()))
	for _, node := range plan.GetPlanNodes() {
		familyByIndex[node.GetIndex()] = nodeOperatorFamily(node, operatorContexts[node.GetIndex()])
	}
	out := make([]Operator, 0, len(plan.GetPlanNodes()))
	for _, node := range plan.GetPlanNodes() {
		childIndexes := operatorChildIndexes(node)
		descendantIndexes := descendantIndexes(node.GetIndex(), childrenByIndex)
		subtreeFamilyCounts := subtreeFamilyCounts(node.GetIndex(), descendantIndexes, familyByIndex)
		out = append(out, Operator{
			Index:               node.GetIndex(),
			DisplayName:         node.GetDisplayName(),
			Family:              familyByIndex[node.GetIndex()],
			ExecutionMethod:     OperatorMetadataString(node, "execution_method"),
			IteratorType:        OperatorMetadataString(node, "iterator_type"),
			ScanMethod:          OperatorMetadataString(node, "scan_method"),
			ScanFormat:          OperatorMetadataString(node, "scan_format"),
			ScanType:            OperatorMetadataString(node, "scan_type"),
			ScanTarget:          nodeMetadataRawString(node, "scan_target"),
			SeekableKeySize:     nodeMetadataRawString(node, "seekable_key_size"),
			JoinType:            OperatorMetadataString(node, "join_type"),
			JoinConfiguration:   OperatorMetadataString(node, "join_configuration"),
			CallType:            OperatorMetadataString(node, "call_type"),
			DistributionTable:   nodeMetadataRawString(node, "distribution_table"),
			SubqueryClusterNode: nodeMetadataRawString(node, "subquery_cluster_node"),
			SpoolName:           nodeMetadataRawString(node, "spool_name"),
			FullScan:            nodeMetadataBool(node, "Full scan"),
			ChildIndexes:        childIndexes,
			DescendantIndexes:   descendantIndexes,
			SubtreeFamilyCounts: subtreeFamilyCounts,
		})
	}
	return out
}

func NormalizeOperatorEdges(plan *spannerpb.QueryPlan) []OperatorEdge {
	if plan == nil {
		return nil
	}
	edges := make([]OperatorEdge, 0)
	for _, node := range plan.GetPlanNodes() {
		for _, link := range node.GetChildLinks() {
			edges = append(edges, OperatorEdge{
				ParentIndex: node.GetIndex(),
				ChildIndex:  link.GetChildIndex(),
				Type:        link.GetType(),
				Variable:    link.GetVariable(),
			})
		}
	}
	return edges
}

func childrenByIndex(plan *spannerpb.QueryPlan) map[int32][]int32 {
	childrenByIndex := make(map[int32][]int32, len(plan.GetPlanNodes()))
	for _, node := range plan.GetPlanNodes() {
		childrenByIndex[node.GetIndex()] = operatorChildIndexes(node)
	}
	return childrenByIndex
}

func operatorChildIndexes(node *spannerpb.PlanNode) []int32 {
	childIndexes := make([]int32, 0, len(node.GetChildLinks()))
	for _, link := range node.GetChildLinks() {
		childIndexes = append(childIndexes, link.GetChildIndex())
	}
	return childIndexes
}

func descendantIndexes(root int32, childrenByIndex map[int32][]int32) []int32 {
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

func subtreeFamilyCounts(root int32, descendantIndexes []int32, familyByIndex map[int32]string) map[string]int {
	counts := ZeroOperatorFamilyCounts()
	add := func(index int32) {
		if family := familyByIndex[index]; family != "" {
			counts[family]++
		}
	}
	add(root)
	for _, index := range descendantIndexes {
		add(index)
	}
	AddDerivedOperatorFamilyCounts(counts)
	return counts
}

func ClassificationWarnings(operators []Operator) []Diagnostic {
	var warnings []Diagnostic
	for _, operator := range operators {
		if operator.Family == "unknown" {
			warnings = append(warnings, Diagnostic{
				ID:      "operator_family_unknown",
				Message: fmt.Sprintf("PlanNode %d with display name %q could not be classified into a known operator family.", operator.Index, operator.DisplayName),
			})
			continue
		}
		if operator.Family == "join" {
			warnings = append(warnings, Diagnostic{
				ID:      "join_family_unknown",
				Message: fmt.Sprintf("Join-like PlanNode %d with display name %q could not be classified into a specific join family.", operator.Index, operator.DisplayName),
			})
			continue
		}
		if normalizeOperatorName(operator.DisplayName) != "aggregate" || operator.Family != "aggregate" {
			continue
		}
		message := fmt.Sprintf("Aggregate PlanNode %d could not be classified as hash_aggregate or stream_aggregate.", operator.Index)
		if operator.IteratorType == "" {
			message += " iterator_type metadata is missing."
		} else {
			message += fmt.Sprintf(" iterator_type metadata is %q.", operator.IteratorType)
		}
		warnings = append(warnings, Diagnostic{
			ID:      "aggregate_iterator_type_unknown",
			Message: message,
		})
	}
	return warnings
}

func ObservedOperatorFamilies(operators []Operator) []string {
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

func OperatorTreeDigest(plan *spannerpb.QueryPlan) string {
	if plan == nil {
		return digest("")
	}
	operatorContexts := operatorContexts(plan)
	var b strings.Builder
	for _, node := range plan.GetPlanNodes() {
		fmt.Fprintf(&b, "%d|%s|%s|%s|%s|%s|%s|%t",
			node.GetIndex(),
			nodeOperatorFamily(node, operatorContexts[node.GetIndex()]),
			node.GetDisplayName(),
			OperatorMetadataString(node, "execution_method"),
			OperatorMetadataString(node, "iterator_type"),
			OperatorMetadataString(node, "scan_method"),
			OperatorMetadataString(node, "scan_type"),
			nodeMetadataBool(node, "Full scan"),
		)
		for _, link := range node.GetChildLinks() {
			fmt.Fprintf(&b, "|%s:%d", link.GetType(), link.GetChildIndex())
		}
		b.WriteByte('\n')
	}
	return digest(b.String())
}

// nodeOperatorFamily classifies one PlanNode into a normalized
// operator family. Display-name classification runs first so scalar-kind
// nodes with concrete operator families keep them: Array Subquery nodes are
// kind SCALAR on Spanner Omni and must still classify as array_subquery.
// Only otherwise-unclassified scalar-kind PlanNodes (Reference, Function,
// Constant, Parameter, and similar expression nodes) map to the dedicated
// "scalar" family; the "unknown" fallback stays reserved for unclassified
// relational operators.
func nodeOperatorFamily(node *spannerpb.PlanNode, context operatorContext) string {
	family := operatorFamilyWithContext(node, context)
	if family == "unknown" && node.GetKind() == spannerpb.PlanNode_SCALAR {
		return "scalar"
	}
	return family
}

type operatorContext struct {
	PushBroadcastInternalHashJoin    bool
	DistributedCrossApplyInternal    bool
	DistributedSemiApplyInternal     bool
	DistributedAntiSemiApplyInternal bool
}

func operatorFamilyWithContext(node *spannerpb.PlanNode, context operatorContext) string {
	displayName := normalizeOperatorName(node.GetDisplayName())
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
	return OperatorFamily(node)
}

func OperatorFamily(node *spannerpb.PlanNode) string {
	if node == nil {
		return "unknown"
	}
	displayName := normalizeOperatorName(node.GetDisplayName())
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
		return aggregateFamily(node)
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
		if nodeMetadataBool(node, "preserve_subquery_order") {
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
		// Spanner Omni emits the TVF name under the capitalized "Name"
		// metadata key; accept the lowercase spelling defensively.
		name := nodeMetadataRawString(node, "Name")
		if name == "" {
			name = nodeMetadataRawString(node, "name")
		}
		if normalizeOperatorName(name) == "search query conversion" {
			return "search_query_conversion_tvf"
		}
		return "unknown"
	case "unit relation":
		return "unit_relation"
	case "verifydeterminism":
		return "verify_determinism"
	default:
		description := normalizeOperatorName(node.GetShortRepresentation().GetDescription())
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

func operatorContexts(plan *spannerpb.QueryPlan) map[int32]operatorContext {
	contexts := map[int32]operatorContext{}
	if plan == nil {
		return contexts
	}
	nodesByIndex := make(map[int32]*spannerpb.PlanNode, len(plan.GetPlanNodes()))
	for _, node := range plan.GetPlanNodes() {
		nodesByIndex[node.GetIndex()] = node
	}
	for _, node := range plan.GetPlanNodes() {
		switch normalizeOperatorName(node.GetDisplayName()) {
		case "push broadcast hash join", "push broadcast hash join outer apply", "push broadcast hash join semi apply", "push broadcast hash join anti semi apply":
			for _, link := range node.GetChildLinks() {
				if !strings.EqualFold(strings.TrimSpace(link.GetType()), "Map") {
					continue
				}
				index, ok := findWrapperInternalOperator(nodesByIndex, link.GetChildIndex(), map[string]bool{"hash join": true})
				if !ok {
					continue
				}
				candidate := nodesByIndex[index]
				if !operatorConsumesBatchScan(nodesByIndex, candidate) {
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
				index, ok := findWrapperInternalOperator(nodesByIndex, link.GetChildIndex(), map[string]bool{
					"cross apply": true,
					"outer apply": true,
				})
				if !ok {
					continue
				}
				candidate := nodesByIndex[index]
				if !operatorConsumesBatchScan(nodesByIndex, candidate) {
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
				index, ok := findWrapperInternalOperator(nodesByIndex, link.GetChildIndex(), map[string]bool{
					"semi apply":      true,
					"anti-semi apply": true,
					"anti semi apply": true,
				})
				if !ok {
					continue
				}
				candidate := nodesByIndex[index]
				if !operatorConsumesBatchScan(nodesByIndex, candidate) {
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
				index, ok := findWrapperInternalOperator(nodesByIndex, link.GetChildIndex(), map[string]bool{
					"semi apply":      true,
					"anti-semi apply": true,
					"anti semi apply": true,
				})
				if !ok {
					continue
				}
				candidate := nodesByIndex[index]
				if !operatorConsumesBatchScan(nodesByIndex, candidate) {
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

func findWrapperInternalOperator(nodesByIndex map[int32]*spannerpb.PlanNode, start int32, targetNames map[string]bool) (int32, bool) {
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
		if targetNames[normalizeOperatorName(node.GetDisplayName())] {
			return start, true
		}
		children := relationalChildIndexes(nodesByIndex, node)
		if len(children) != 1 {
			return 0, false
		}
		start = children[0]
	}
}

func operatorConsumesBatchScan(nodesByIndex map[int32]*spannerpb.PlanNode, node *spannerpb.PlanNode) bool {
	for _, link := range node.GetChildLinks() {
		if subtreeHasBatchScan(nodesByIndex, link.GetChildIndex(), map[int32]bool{}) {
			return true
		}
	}
	return false
}

func aggregateFamily(node *spannerpb.PlanNode) string {
	switch nodeMetadataString(node, "iterator_type") {
	case "hash":
		return "hash_aggregate"
	case "stream":
		return "stream_aggregate"
	default:
		return "aggregate"
	}
}

func OperatorMetadataString(node *spannerpb.PlanNode, key string) string {
	value := nodeMetadataString(node, key)
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

func nodeMetadataString(node *spannerpb.PlanNode, key string) string {
	return strings.ToLower(nodeMetadataRawString(node, key))
}

func nodeMetadataRawString(node *spannerpb.PlanNode, key string) string {
	if node == nil || node.GetMetadata() == nil {
		return ""
	}
	value, ok := node.GetMetadata().AsMap()[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func nodeMetadataBool(node *spannerpb.PlanNode, key string) bool {
	switch nodeMetadataString(node, key) {
	case "true":
		return true
	default:
		return false
	}
}

func relationalChildIndexes(nodesByIndex map[int32]*spannerpb.PlanNode, node *spannerpb.PlanNode) []int32 {
	var out []int32
	for _, link := range node.GetChildLinks() {
		child, ok := nodesByIndex[link.GetChildIndex()]
		if !ok {
			continue
		}
		if isRelationalPlanNode(child) {
			out = append(out, child.GetIndex())
		}
	}
	return out
}

func subtreeHasBatchScan(nodesByIndex map[int32]*spannerpb.PlanNode, index int32, seen map[int32]bool) bool {
	if seen[index] {
		return false
	}
	seen[index] = true
	node, ok := nodesByIndex[index]
	if !ok {
		return false
	}
	if OperatorMetadataString(node, "scan_type") == "batch_scan" {
		return true
	}
	if OperatorMetadataString(node, "scan_method") == "batch" &&
		strings.HasPrefix(nodeMetadataRawString(node, "scan_target"), "$") {
		return true
	}
	for _, link := range node.GetChildLinks() {
		if subtreeHasBatchScan(nodesByIndex, link.GetChildIndex(), seen) {
			return true
		}
	}
	return false
}

func isRelationalPlanNode(node *spannerpb.PlanNode) bool {
	switch normalizeOperatorName(node.GetDisplayName()) {
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
