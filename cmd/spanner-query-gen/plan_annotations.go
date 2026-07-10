package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	spanalyzer "github.com/apstndb/spanalyzer"
	"github.com/apstndb/spanalyzer/internal/querygen"
	"github.com/apstndb/spanalyzer/plancontract"
	queryplan "github.com/apstndb/spannerplan"
	"github.com/apstndb/spannerplan/plantree/reference"
)

// Schema-aware rendered-plan annotations. These enrich the human-readable
// plan text with information the QueryPlan alone does not carry (catalog key
// counts) or that is normalized outside the renderer (plancontract operator
// families). All schema and normalization knowledge stays in this module;
// spannerplan only exposes the two rendering hooks. The hooks divide by
// whether the information already renders as a metadata field:
//
//   - seekability enriches the existing seekable_key_size value in place
//     (queryplan.WithMetadataValueFunc), so "1" becomes "1/2" without
//     duplicating the field elsewhere on the row;
//   - operator families have no metadata counterpart, so they append as a
//     row annotation (reference.WithRowAnnotator).

type planReportAnnotateOptions struct {
	Seekability bool
	Families    bool
}

func (o planReportAnnotateOptions) Any() bool {
	return o.Seekability || o.Families
}

func parsePlanReportAnnotations(value string) (planReportAnnotateOptions, error) {
	var out planReportAnnotateOptions
	for _, part := range strings.Split(value, ",") {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "":
		case "seekability":
			out.Seekability = true
		case "families":
			out.Families = true
		default:
			return planReportAnnotateOptions{}, fmt.Errorf("unsupported --annotate value %q; supported values: seekability, families", strings.TrimSpace(part))
		}
	}
	return out, nil
}

// planReportSchemaKeyCounts builds the catalog from the schema's DDL file and
// returns its declared key column counts. A schema without DDL yields an
// empty map, so seekability annotations are silently absent rather than an
// error.
func planReportSchemaKeyCounts(schema querygen.QueryCodegenSchema, baseDir string) (map[string]int, error) {
	if strings.TrimSpace(schema.DDL) == "" {
		return map[string]int{}, nil
	}
	path := resolveOutputPath(baseDir, schema.DDL)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	catalog, err := spanalyzer.BuildSchemaCatalog(path, string(data))
	if err != nil {
		return nil, err
	}
	return planReportCatalogKeyCounts(catalog), nil
}

// planReportCatalogKeyCounts returns the number of declared key columns for
// every table and index in the catalog, keyed by object name as it appears in
// scan_target plan metadata. Secondary indexes count only their declared key
// parts: that is the schema surface users design against, and it matches the
// shard-seekability reading (an index on (ShardId, Timestamp) is "fully
// seekable" at 2/2). The storage key of a secondary index additionally
// appends base-table primary key columns, so seekable_key_size can in
// principle exceed this total for key-joining probes; the annotation renders
// the raw seekable_key_size unchanged in that case.
func planReportCatalogKeyCounts(catalog *spanalyzer.Catalog) map[string]int {
	if catalog == nil {
		return nil
	}
	counts := make(map[string]int, len(catalog.Tables)+len(catalog.Indexes))
	for name, table := range catalog.Tables {
		counts[name] = len(table.PrimaryKey)
	}
	for name, index := range catalog.Indexes {
		counts[name] = len(index.Keys)
	}
	return counts
}

// planReportRenderAnnotationOptions builds the spannerplan render options for
// the enabled annotations. All annotation values are precomputed into
// index-keyed maps, so the renderer callbacks only look up node indexes.
func planReportRenderAnnotationOptions(plan *spannerpb.QueryPlan, operators []planReportOperator, keyCounts map[string]int, annotate planReportAnnotateOptions) []reference.Option {
	var out []reference.Option
	if annotate.Seekability {
		seekValues := planReportSeekableKeyValues(plan, keyCounts)
		if len(seekValues) > 0 {
			out = append(out, reference.WithQueryPlanOptions(queryplan.WithMetadataValueFunc(func(node *spannerpb.PlanNode, key, value string) string {
				if key == "seekable_key_size" {
					if replacement, ok := seekValues[node.GetIndex()]; ok {
						return replacement
					}
				}
				return value
			})))
		}
	}
	if annotate.Families {
		familyAnnotations := planReportFamilyAnnotations(operators)
		out = append(out, reference.WithRowAnnotator(func(node *spannerpb.PlanNode) string {
			return familyAnnotations[node.GetIndex()]
		}))
	}
	return out
}

// planReportFamilyAnnotations renders "{<family>}" for every classified
// operator, extended after a colon with the derived umbrella families the
// operator contributes to, in the lexicographic order
// DerivedOperatorFamiliesForOperator defines (for example
// "{full_sort: blocking_operator, explicit_sort}"). The colon makes the
// two-sorted structure self-describing: the left side is the single-valued
// concrete classification, the right side is the set of derived
// cross-cutting attributes, so readers do not need the position convention
// to tell them apart. By convention braces in rendered rows are reserved
// for these normalization labels, so no "family:" prefix is needed.
// Rendered trees only show relational rows, so entries for folded scalar
// nodes are simply never looked up.
func planReportFamilyAnnotations(operators []planReportOperator) map[int32]string {
	out := make(map[int32]string, len(operators))
	for _, operator := range operators {
		if operator.Family == "" {
			continue
		}
		label := operator.Family
		if derived := plancontract.DerivedOperatorFamiliesForOperator(operator); len(derived) > 0 {
			label += ": " + strings.Join(derived, ", ")
		}
		out[operator.Index] = "{" + label + "}"
	}
	return out
}

// planReportSeekableKeyValues computes the "k/N" replacement value for every
// node that carries seekable_key_size metadata, where N is the declared key
// column count of the scanned table or index from the catalog.
// seekable_key_size usually sits on a Filter Scan wrapper while scan_target
// sits on the scan node below it, so the target is resolved from the node's
// relational subtree. Nodes whose target is unknown or synthetic keep their
// original value.
func planReportSeekableKeyValues(plan *spannerpb.QueryPlan, keyCounts map[string]int) map[int32]string {
	nodes := plan.GetPlanNodes()
	nodesByIndex := make(map[int32]*spannerpb.PlanNode, len(nodes))
	for _, node := range nodes {
		nodesByIndex[node.GetIndex()] = node
	}
	out := map[int32]string{}
	for _, node := range nodes {
		size := planReportRawMetadataString(node, "seekable_key_size")
		if size == "" {
			continue
		}
		// seekable_key_size counts the key prefix length of a range-bounded
		// seek; pure point seeks (all-equality key conditions) and plain
		// full scans both report 0 (verified on Omni 2026.r1-beta across
		// literal/param point reads, prefix ranges, and full scans).
		// Rendering "0/N" would misread a perfect point seek as seeking
		// nothing, so the ambiguous 0 keeps its raw value.
		sizeValue, err := strconv.Atoi(size)
		if err != nil || sizeValue <= 0 {
			continue
		}
		target := planReportScanTargetForSubtree(node, nodesByIndex)
		// Synthetic targets such as $BatchPartition have no catalog entry.
		if target == "" || strings.HasPrefix(target, "$") {
			continue
		}
		total, ok := keyCounts[target]
		if !ok {
			continue
		}
		if sizeValue > total {
			continue
		}
		out[node.GetIndex()] = fmt.Sprintf("%d/%d", sizeValue, total)
	}
	return out
}

// planReportRawMetadataString reads node metadata without the lowercasing
// that plancontract.OperatorMetadataString applies: scan_target must keep its
// case to match catalog object names, and seekable_key_size is rendered
// verbatim.
func planReportRawMetadataString(node *spannerpb.PlanNode, key string) string {
	if node == nil || node.GetMetadata() == nil {
		return ""
	}
	value, ok := node.GetMetadata().GetFields()[key]
	if !ok || value == nil {
		return ""
	}
	raw := value.AsInterface()
	if raw == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(raw))
}

func planReportScanTargetForSubtree(node *spannerpb.PlanNode, nodesByIndex map[int32]*spannerpb.PlanNode) string {
	var targets []string
	visited := map[int32]bool{}
	var walk func(*spannerpb.PlanNode)
	walk = func(current *spannerpb.PlanNode) {
		if current == nil || visited[current.GetIndex()] {
			return
		}
		visited[current.GetIndex()] = true
		if target := planReportRawMetadataString(current, "scan_target"); target != "" {
			targets = append(targets, target)
		}
		for _, link := range current.GetChildLinks() {
			child := nodesByIndex[link.GetChildIndex()]
			if child != nil && child.GetKind() == spannerpb.PlanNode_RELATIONAL {
				walk(child)
			}
		}
	}
	walk(node)
	if len(targets) != 1 {
		// A Filter Scan normally wraps exactly one target-bearing Scan. Do not
		// attach a key count when a future or synthetic shape contains multiple
		// scans: choosing the first one would make a plausible but unsupported
		// k/N claim.
		return ""
	}
	return targets[0]
}
