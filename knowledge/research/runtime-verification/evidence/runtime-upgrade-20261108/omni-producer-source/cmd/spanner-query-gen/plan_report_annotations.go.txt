package main

import (
	"fmt"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/apstndb/spanalyzer/plancontract"
	"github.com/apstndb/spannerplan/plantree/reference"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func renderPlanReportTree(plan *spannerpb.QueryPlan, paths []plancontract.AccessPath, opts planReportOptions) (string, error) {
	if len(plan.GetPlanNodes()) == 0 {
		return "", fmt.Errorf("AnalyzeQuery returned no plan nodes")
	}
	renderPlan := plan
	if opts.AnnotateSeekability {
		// Stock spannerplan already renders metadata values. Enrich a private
		// clone, so annotations require no unpublished renderer hooks and cannot
		// affect normalization, contract evaluation, or the raw plan digest.
		renderPlan = proto.Clone(plan).(*spannerpb.QueryPlan)
		values := map[int32]string{}
		nodes := map[int32]*spannerpb.PlanNode{}
		for _, node := range renderPlan.GetPlanNodes() {
			nodes[node.GetIndex()] = node
		}
		for _, path := range paths {
			if !path.ConditionsComplete || path.RangeSeekableKeySize == nil || path.DeclaredKeyCount == nil {
				continue
			}
			k, n := *path.RangeSeekableKeySize, *path.DeclaredKeyCount
			if k > 0 && n > 0 && k <= n {
				owners := []int32{path.ScanIndex}
				if path.FilterIndex != nil {
					owners = append(owners, *path.FilterIndex)
				}
				for _, index := range owners {
					if planReportOperatorMetadataString(nodes[index], "seekable_key_size") == path.SeekableKeySize {
						values[index] = fmt.Sprintf("%d/%d", k, n)
					}
				}
			}
		}
		for _, node := range renderPlan.GetPlanNodes() {
			if value, ok := values[node.GetIndex()]; ok && node.GetMetadata() != nil {
				node.Metadata.Fields["seekable_key_size"] = structpb.NewStringValue(value)
			}
		}
	}
	return reference.RenderTreeTable(renderPlan.GetPlanNodes(), opts.RenderMode, opts.Format, opts.WrapWidth)
}
