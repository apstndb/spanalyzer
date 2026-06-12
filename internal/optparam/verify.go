package optparam

import (
	"fmt"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	spanalyzer "github.com/apstndb/spanalyzer"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

// VerifyResult is what VerifyVariants returns when every variant agrees on
// the result row type.
type VerifyResult struct {
	// Variants is the enumerated set (preserves the order returned by
	// EnumerateVariants).
	Variants []Variant
	// RowType is the agreed result row type. Pointer-shared across variants
	// since they were proven equal.
	RowType *spannerpb.StructType
}

// VerifyVariants enumerates SQL variants for the given query and runs the
// GoogleSQL analyzer against each one. The function fails if any variant
// fails to analyze or if two variants produce different result row types.
//
// The analyzer is rebuilt per variant because AnalyzerOptions track per-call
// state (parameter declarations). All declared params are added to every
// variant, even ones whose predicate block is omitted, so the analyzer can
// resolve identifiers without surprise.
func VerifyVariants(ddlPath, ddlSQL, sql string, params []Param) (*VerifyResult, error) {
	variants, err := EnumerateVariants(sql, params)
	if err != nil {
		return nil, err
	}
	if len(variants) == 0 {
		return nil, fmt.Errorf("no variants generated")
	}
	var first *spannerpb.StructType
	var firstKey string
	for i, v := range variants {
		analyzer, err := spanalyzer.NewAnalyzerFromDDL(ddlPath, ddlSQL)
		if err != nil {
			return nil, fmt.Errorf("variant %s: build analyzer: %w", v.Key(), err)
		}
		for _, p := range params {
			// ModeOrderByChoice never reaches the analyzer as a query
			// parameter: the choice is resolved into a literal SQL
			// fragment at codegen time.
			if p.Mode == ModeOrderByChoice {
				continue
			}
			spec, err := spanalyzer.ParseTypeSpec("param", p.Type)
			if err != nil {
				return nil, fmt.Errorf("variant %s param %s: %w", v.Key(), p.Name, err)
			}
			if err := analyzer.AddQueryParameter(p.Name, spec); err != nil {
				return nil, fmt.Errorf("variant %s param %s: %w", v.Key(), p.Name, err)
			}
		}
		rowType, err := analyzer.RowTypeForStatement(v.SQL)
		if err != nil {
			return nil, fmt.Errorf("variant %s: analyze: %w", v.Key(), err)
		}
		if i == 0 {
			first = rowType
			firstKey = v.Key()
			continue
		}
		if !proto.Equal(first, rowType) {
			return nil, fmt.Errorf(
				"row type mismatch between variant %s and %s:\n%s\nvs.\n%s",
				firstKey, v.Key(),
				prototext.Format(first), prototext.Format(rowType),
			)
		}
	}
	return &VerifyResult{Variants: variants, RowType: first}, nil
}
