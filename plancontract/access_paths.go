package plancontract

import (
	"slices"
	"sort"
	"strconv"
	"strings"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

// AccessPathNormalizationVersion identifies the mutable normalized access-path
// contract. It does not imply that backend expression spellings are stable.
const AccessPathNormalizationVersion = "v1alpha.2"

// ScanTarget supplies schema facts without coupling this module to an analyzer
// or DDL parser. Unknown targets retain empty schema facts rather than guessing.
type ScanTarget struct {
	Name                        string
	Kind                        string
	BaseTable                   string
	KeyColumns                  []string
	StoredColumns               []string
	NullFiltered                bool
	Aliases                     []string
	IndexOrigin                 string
	Managed                     *bool
	InferredConstraintSelectors []string
}

// ScanCondition retains the immediate condition root and its backend expression.
// Columns are proven by Reference nodes bound by this scan's output links;
// unrelated join variables and text that merely resembles a column are omitted.
// Columns is a sorted, duplicate-free set, not a predicate or key order.
type ScanCondition struct {
	Index      int32    `json:"index" yaml:"index"`
	Expression string   `json:"expression" yaml:"expression"`
	Columns    []string `json:"columns" yaml:"columns"`
}

// AccessPath joins a Scan with its unique immediate Filter Scan wrapper.
// ConditionsComplete is false when multiple relational children or wrappers
// prevent an unambiguous pairing. Zero range-key metadata never establishes
// absence of a seek; HasSeekCondition is derived only from condition links.
// SeekColumns and ResidualColumns are sorted, duplicate-free sets. KeyColumns
// preserves declared key order. Test membership in SeekColumns when checking
// key prefixes; slice equality does not express that relationship.
type AccessPath struct {
	CatalogIndexName            string          `json:"catalog_index_name,omitempty" yaml:"catalog_index_name,omitempty"`
	IndexOrigin                 string          `json:"index_origin,omitempty" yaml:"index_origin,omitempty"`
	Managed                     *bool           `json:"managed,omitempty" yaml:"managed,omitempty"`
	InferredConstraintSelectors []string        `json:"inferred_constraint_selectors,omitempty" yaml:"inferred_constraint_selectors,omitempty"`
	ScanIndex                   int32           `json:"scan_index" yaml:"scan_index"`
	FilterIndex                 *int32          `json:"filter_index" yaml:"filter_index"`
	ScanTarget                  string          `json:"scan_target" yaml:"scan_target"`
	ScanType                    string          `json:"scan_type" yaml:"scan_type"`
	ScanTargetKind              string          `json:"scan_target_kind" yaml:"scan_target_kind"`
	BaseTable                   string          `json:"base_table" yaml:"base_table"`
	FullScan                    bool            `json:"full_scan" yaml:"full_scan"`
	ConditionsComplete          bool            `json:"conditions_complete" yaml:"conditions_complete"`
	HasSeekCondition            bool            `json:"has_seek_condition" yaml:"has_seek_condition"`
	HasResidualCondition        bool            `json:"has_residual_condition" yaml:"has_residual_condition"`
	HasTimestampCondition       bool            `json:"has_timestamp_condition" yaml:"has_timestamp_condition"`
	HasSearchCondition          bool            `json:"has_search_condition" yaml:"has_search_condition"`
	SeekConditions              []ScanCondition `json:"seek_conditions" yaml:"seek_conditions"`
	ResidualConditions          []ScanCondition `json:"residual_conditions" yaml:"residual_conditions"`
	TimestampConditions         []ScanCondition `json:"timestamp_conditions" yaml:"timestamp_conditions"`
	SearchConditions            []ScanCondition `json:"search_conditions" yaml:"search_conditions"`
	SeekColumns                 []string        `json:"seek_columns" yaml:"seek_columns"`
	ResidualColumns             []string        `json:"residual_columns" yaml:"residual_columns"`
	SeekableKeySize             string          `json:"seekable_key_size" yaml:"seekable_key_size"`
	RangeSeekableKeySize        *int64          `json:"range_seekable_key_size" yaml:"range_seekable_key_size"`
	DeclaredKeyCount            *int64          `json:"declared_key_count" yaml:"declared_key_count"`
	KeyColumns                  []string        `json:"key_columns" yaml:"key_columns"`
	StoredColumns               []string        `json:"stored_columns" yaml:"stored_columns"`
	NullFiltered                *bool           `json:"null_filtered" yaml:"null_filtered"`
}

// NormalizeAccessPaths retains condition evidence at its owning scan, without
// propagating conditions from joins or another scan's subtree. targets may be
// nil when schema information is unavailable. ANN root/leaf components retain
// their explicit scan kinds; a root centroid scan is not a base-table scan.
func NormalizeAccessPaths(plan *spannerpb.QueryPlan, targets []ScanTarget) []AccessPath {
	nodes := sortedPlanNodes(plan)
	byIndex := make(map[int32]*spannerpb.PlanNode, len(nodes))
	for _, node := range nodes {
		byIndex[node.GetIndex()] = node
	}
	wrappers := map[int32][]*spannerpb.PlanNode{}
	ambiguous := map[int32]bool{}
	markAmbiguous := func(root *spannerpb.PlanNode) {
		seen := map[int32]bool{}
		var visit func(*spannerpb.PlanNode)
		visit = func(node *spannerpb.PlanNode) {
			if node == nil || seen[node.GetIndex()] {
				return
			}
			seen[node.GetIndex()] = true
			if isAccessScan(node) {
				ambiguous[node.GetIndex()] = true
			}
			for _, link := range node.GetChildLinks() {
				child := byIndex[link.GetChildIndex()]
				if child != nil && (child.GetKind() == spannerpb.PlanNode_RELATIONAL || isAccessScan(child)) {
					visit(child)
				}
			}
		}
		visit(root)
	}

	for _, node := range nodes {
		if normalizeOperatorName(node.GetDisplayName()) != "filter scan" {
			continue
		}
		var inputs []*spannerpb.PlanNode
		for _, link := range node.GetChildLinks() {
			child := byIndex[link.GetChildIndex()]
			if child != nil && (child.GetKind() == spannerpb.PlanNode_RELATIONAL || isAccessScan(child)) {
				inputs = append(inputs, child)
			}
		}
		if len(inputs) == 1 && isAccessScan(inputs[0]) {
			wrappers[inputs[0].GetIndex()] = append(wrappers[inputs[0].GetIndex()], node)
		} else {
			for _, child := range inputs {
				markAmbiguous(child)
			}
		}
	}
	out := make([]AccessPath, 0)
	for _, scan := range nodes {
		if !isAccessScan(scan) {
			continue
		}
		path := AccessPath{
			ScanIndex: scan.GetIndex(), ScanTarget: nodeMetadataRawString(scan, "scan_target"),
			ScanType: OperatorMetadataString(scan, "scan_type"), SeekableKeySize: nodeMetadataRawString(scan, "seekable_key_size"), FullScan: nodeMetadataBool(scan, "Full scan"),
			ConditionsComplete: !ambiguous[scan.GetIndex()] && len(wrappers[scan.GetIndex()]) <= 1,
			ScanTargetKind:     "unknown", SeekConditions: []ScanCondition{}, ResidualConditions: []ScanCondition{},
			TimestampConditions: []ScanCondition{}, SearchConditions: []ScanCondition{},
			SeekColumns: []string{}, ResidualColumns: []string{}, KeyColumns: []string{}, StoredColumns: []string{},
		}
		switch path.ScanType {
		case "table_scan":
			path.ScanTargetKind, path.BaseTable = "table", path.ScanTarget
		case "index_scan":
			path.ScanTargetKind = "secondary_index"
		case "search_index_scan":
			path.ScanTargetKind = "search_index"
		case "vector_index_root_scan", "vector_index_leaf_scan":
			path.ScanTargetKind = strings.TrimSuffix(path.ScanType, "_scan")
		case "batch_scan":
			path.ScanTargetKind = "batch"
		}
		var target *ScanTarget
		for i := range targets {
			candidate := &targets[i]
			if (candidate.Name != path.ScanTarget && !slices.Contains(candidate.Aliases, path.ScanTarget)) || (path.ScanTargetKind != "unknown" && path.ScanTargetKind != "batch" && candidate.Kind != "" && candidate.Kind != path.ScanTargetKind) {
				continue
			}
			if target != nil {
				target = nil
				break
			}
			target = candidate
		}
		if target != nil {
			if target.Kind != "" {
				path.ScanTargetKind = target.Kind
			}
			if target.BaseTable != "" {
				path.BaseTable = target.BaseTable
			}
			path.KeyColumns = append(path.KeyColumns, target.KeyColumns...)
			path.StoredColumns = append(path.StoredColumns, target.StoredColumns...)
			if target.KeyColumns != nil {
				count := int64(len(target.KeyColumns))
				path.DeclaredKeyCount = &count
			}
			if target.Kind == "secondary_index" {
				// Existing callers need not opt into the supplemental index
				// provenance projection merely by supplying declared keys.
				if target.IndexOrigin != "" || target.Managed != nil || len(target.InferredConstraintSelectors) > 0 {
					path.CatalogIndexName = target.Name
					path.IndexOrigin = target.IndexOrigin
					path.InferredConstraintSelectors = slices.Clone(target.InferredConstraintSelectors)
					if target.Managed != nil {
						managed := *target.Managed
						path.Managed = &managed
					}
				}
				value := target.NullFiltered
				path.NullFiltered = &value
			}
		}
		owners := []*spannerpb.PlanNode{scan}
		if filter := wrappers[scan.GetIndex()]; len(filter) == 1 {
			index := filter[0].GetIndex()
			path.FilterIndex = &index
			if size := nodeMetadataRawString(filter[0], "seekable_key_size"); size != "" {
				path.SeekableKeySize = size
			}
			owners = append(owners, filter[0])
		}
		if size, err := strconv.ParseInt(path.SeekableKeySize, 10, 64); err == nil && size > 0 {
			path.RangeSeekableKeySize = &size
		}
		bindings := scanColumnBindings(scan, byIndex)
		for _, owner := range owners {
			for _, link := range owner.GetChildLinks() {
				var conditions *[]ScanCondition
				switch link.GetType() {
				case "Seek Condition":
					path.HasSeekCondition, conditions = true, &path.SeekConditions
				case "Residual Condition":
					path.HasResidualCondition, conditions = true, &path.ResidualConditions
				case "Timestamp Condition":
					path.HasTimestampCondition, conditions = true, &path.TimestampConditions
				case "Search Condition":
					path.HasSearchCondition, conditions = true, &path.SearchConditions
				default:
					continue
				}
				child := byIndex[link.GetChildIndex()]
				*conditions = append(*conditions, ScanCondition{Index: link.GetChildIndex(),
					Expression: child.GetShortRepresentation().GetDescription(),
					Columns:    conditionColumns(child, byIndex, bindings)})
			}
		}
		for _, group := range []*[]ScanCondition{&path.SeekConditions, &path.ResidualConditions, &path.TimestampConditions, &path.SearchConditions} {
			sort.SliceStable(*group, func(i, j int) bool { return (*group)[i].Index < (*group)[j].Index })
		}
		path.SeekColumns = combinedConditionColumns(path.SeekConditions)
		path.ResidualColumns = combinedConditionColumns(path.ResidualConditions)
		out = append(out, path)
	}
	return out
}

func isAccessScan(node *spannerpb.PlanNode) bool {
	if node == nil {
		return false
	}
	name := normalizeOperatorName(node.GetDisplayName())
	return name == "scan" || name == "table scan" || name == "index scan"
}

func scanColumnBindings(scan *spannerpb.PlanNode, nodes map[int32]*spannerpb.PlanNode) map[string]string {
	bindings := map[string]string{}
	for _, link := range scan.GetChildLinks() {
		child := nodes[link.GetChildIndex()]
		name := child.GetShortRepresentation().GetDescription()
		if link.GetVariable() != "" && child.GetDisplayName() == "Reference" && identifierRegexp.MatchString(name) {
			bindings[link.GetVariable()] = name
		}
	}
	return bindings
}

func conditionColumns(root *spannerpb.PlanNode, nodes map[int32]*spannerpb.PlanNode, bindings map[string]string) []string {
	columns := []string{}
	seen := map[int32]bool{}
	var visit func(*spannerpb.PlanNode)
	visit = func(node *spannerpb.PlanNode) {
		if node == nil || seen[node.GetIndex()] || node.GetKind() == spannerpb.PlanNode_RELATIONAL {
			return
		}
		seen[node.GetIndex()] = true
		if node.GetDisplayName() == "Reference" {
			if column, ok := bindings[strings.TrimPrefix(node.GetShortRepresentation().GetDescription(), "$")]; ok {
				columns = append(columns, column)
			}
		}
		for _, link := range node.GetChildLinks() {
			visit(nodes[link.GetChildIndex()])
		}
	}
	visit(root)
	slices.Sort(columns)
	return slices.Compact(columns)
}

func combinedConditionColumns(conditions []ScanCondition) []string {
	columns := []string{}
	for _, condition := range conditions {
		columns = append(columns, condition.Columns...)
	}
	slices.Sort(columns)
	return slices.Compact(columns)
}
