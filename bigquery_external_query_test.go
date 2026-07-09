package spanalyzer

import (
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

func TestBigQueryTVFRelationFromSpannerRowTypeRejectsNilField(t *testing.T) {
	t.Parallel()

	_, err := bigQueryTVFRelationFromSpannerRowType(nil, &spannerpb.StructType{
		Fields: []*spannerpb.StructType_Field{nil},
	})
	if err == nil || !strings.Contains(err.Error(), "row field 0 is nil") {
		t.Fatalf("bigQueryTVFRelationFromSpannerRowType() error = %v, want indexed nil row field error", err)
	}
}
