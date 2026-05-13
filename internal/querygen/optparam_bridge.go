package querygen

import (
	"fmt"
	"strings"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	"github.com/apstndb/go-googlesql-spanner-poc/internal/optparam"
)

// analyzeCodegenQuerySpannerVariants enumerates SQL variants for a query
// that uses one or more optional markers, runs the GoogleSQL analyzer on
// each variant, confirms they all agree on the result row type, and
// returns the row-type fields together with the per-variant plan-contract
// entries.
//
// The function is the codegen-time bridge between internal/optparam and
// internal/querygen. The marker grammar lives in internal/optparam; this
// function only translates the QueryCodegenParam list into the optparam
// param shape and ferries results back.
func analyzeCodegenQuerySpannerVariants(ddlPath, ddl string, schema QueryCodegenSchema, query QueryCodegenQuery, baseDir string) ([]goResultField, []QueryCodegenPlanQueryVariant, error) {
	opParams, err := toOptParamParams(query.Params)
	if err != nil {
		return nil, nil, err
	}
	segments, err := optparam.SegmentTemplate(query.SQL, opParams)
	if err != nil {
		return nil, nil, err
	}
	variants, err := optparam.EnumerateVariants(query.SQL, opParams)
	if err != nil {
		return nil, nil, err
	}
	if err := optparam.VerifyBuilderRoundTrip(segments, variants); err != nil {
		return nil, nil, err
	}

	protoPaths := resolveCodegenPaths(baseDir, schema.ProtoDescriptorFiles)
	var (
		firstRowType *spannerpb.StructType
		firstLabel   string
	)
	for _, v := range variants {
		analyzer, err := NewAnalyzerFromDDLWithProtoDescriptorFiles(ddlPath, ddl, protoPaths)
		if err != nil {
			return nil, nil, fmt.Errorf("variant %s: build analyzer: %w", v.Key(), err)
		}
		for _, p := range query.Params {
			if optParamMode(p) == optparam.ModeOrderByChoice {
				continue
			}
			if err := addSpannerQueryParameter(analyzer, query.Name, p); err != nil {
				return nil, nil, fmt.Errorf("variant %s: %w", v.Key(), err)
			}
		}
		rowType, err := analyzer.RowTypeForStatement(v.SQL)
		if err != nil {
			return nil, nil, fmt.Errorf("variant %s: %w", v.Key(), err)
		}
		if firstRowType == nil {
			firstRowType = rowType
			firstLabel = v.Key()
			continue
		}
		if !proto.Equal(firstRowType, rowType) {
			return nil, nil, fmt.Errorf("row type mismatch between variant %s and %s:\n%s\nvs.\n%s",
				firstLabel, v.Key(), prototext.Format(firstRowType), prototext.Format(rowType))
		}
	}

	fields := make([]goResultField, 0, len(firstRowType.GetFields()))
	for _, f := range firstRowType.GetFields() {
		fields = append(fields, goResultFieldFromSpanner(f.Name, f.Type))
	}
	return fields, toPlanQueryVariants(variants), nil
}

func emitQueryGoBuilder(query QueryCodegenQuery, builderFunc, paramsType string) (string, []string, error) {
	opParams, err := toOptParamParams(query.Params)
	if err != nil {
		return "", nil, err
	}
	segments, err := optparam.SegmentTemplate(query.SQL, opParams)
	if err != nil {
		return "", nil, err
	}
	code, err := optparam.EmitGoBuilder(segments, opParams, optparam.BuilderOptions{
		FuncName:       builderFunc,
		ParamsTypeName: paramsType,
		Fragment:       true,
	})
	if err != nil {
		return "", nil, err
	}
	imports := []string{"sort", "strings"}
	for _, p := range opParams {
		if p.Mode == optparam.ModeNullIsNull {
			imports = append(imports, "cloud.google.com/go/spanner")
			break
		}
	}
	return code, imports, nil
}

func toOptParamParams(params []QueryCodegenParam) ([]optparam.Param, error) {
	out := make([]optparam.Param, 0, len(params))
	for _, p := range params {
		mode := optParamMode(p)
		if mode == optparam.ModeRequired && strings.TrimSpace(p.Optional) != "" && strings.ToLower(strings.TrimSpace(p.Optional)) != "required" {
			return nil, fmt.Errorf("param %q: unsupported optional %q", p.Name, p.Optional)
		}
		out = append(out, optparam.Param{
			Name:    p.Name,
			Type:    p.Type,
			Mode:    mode,
			Choices: p.Choices,
			Default: p.Default,
		})
	}
	return out, nil
}

func optParamMode(p QueryCodegenParam) optparam.Mode {
	switch strings.ToLower(strings.TrimSpace(p.Optional)) {
	case "null_is_null":
		return optparam.ModeNullIsNull
	case "omit_when_null":
		return optparam.ModeOmitWhenNull
	case "omit_when_empty":
		return optparam.ModeOmitWhenEmpty
	case "orderby_choice":
		return optparam.ModeOrderByChoice
	default:
		return optparam.ModeRequired
	}
}

func toPlanQueryVariants(variants []optparam.Variant) []QueryCodegenPlanQueryVariant {
	if len(variants) == 0 {
		return nil
	}
	planVariants := optparam.BuildPlanVariants(&optparam.VerifyResult{Variants: variants})
	out := make([]QueryCodegenPlanQueryVariant, 0, len(planVariants))
	for _, v := range planVariants {
		out = append(out, QueryCodegenPlanQueryVariant{
			Label:             v.Label,
			SQL:               v.SQL,
			SQLSHA256:         v.SQLSHA256,
			PresentParams:     v.PresentParams,
			AbsentParams:      v.AbsentParams,
			ChoiceAssignments: cloneChoices(variants, v.Label),
		})
	}
	return out
}

// pickCanonicalVariant returns the variant whose label aggregates the
// most present params (ties broken alphabetically). The picked variant
// becomes the plan's top-level SQL / SQLSHA256 so single-SQL consumers
// keep seeing the fully-expanded shape.
func pickCanonicalVariant(variants []QueryCodegenPlanQueryVariant) QueryCodegenPlanQueryVariant {
	best := variants[0]
	for _, v := range variants[1:] {
		if len(v.PresentParams) > len(best.PresentParams) ||
			(len(v.PresentParams) == len(best.PresentParams) && v.Label < best.Label) {
			best = v
		}
	}
	return best
}

func cloneChoices(variants []optparam.Variant, label string) map[string]string {
	for _, v := range variants {
		if v.Key() != label {
			continue
		}
		if len(v.ChoiceAssignments) == 0 {
			return nil
		}
		out := make(map[string]string, len(v.ChoiceAssignments))
		for k, val := range v.ChoiceAssignments {
			out[k] = val
		}
		return out
	}
	return nil
}
