package spanalyzer

import (
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

func TestTypeSpecFromSpannerPBRejectsNilStructField(t *testing.T) {
	t.Parallel()

	_, err := TypeSpecFromSpannerPB(&spannerpb.Type{
		Code: spannerpb.TypeCode_STRUCT,
		StructType: &spannerpb.StructType{Fields: []*spannerpb.StructType_Field{
			{Name: "id", Type: &spannerpb.Type{Code: spannerpb.TypeCode_INT64}},
			nil,
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "struct field 1 is nil") {
		t.Fatalf("TypeSpecFromSpannerPB() error = %v, want indexed nil struct field error", err)
	}
}
