package querygen

import (
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

func TestGenerateGoStructFromBigQueryTableSchemaBoth(t *testing.T) {
	code, err := GenerateGoStructFromBigQueryTableSchema(&BigQueryTableSchema{
		Fields: []*BigQueryTableFieldSchema{
			{Name: "user_id", Type: "INTEGER", Mode: "NULLABLE"},
			{Name: "payload", Type: "BYTES", Mode: "NULLABLE"},
			{Name: "amount", Type: "NUMERIC", Mode: "NULLABLE"},
			{
				Name: "profile",
				Type: "RECORD",
				Mode: "NULLABLE",
				Fields: []*BigQueryTableFieldSchema{
					{Name: "display_name", Type: "STRING", Mode: "NULLABLE"},
				},
			},
		},
	}, GoStructOptions{PackageName: "result", StructName: "OrderRow", Target: GoStructTargetBoth})
	if err != nil {
		t.Fatalf("GenerateGoStructFromBigQueryTableSchema() error = %v", err)
	}
	for _, want := range []string{
		"package result",
		"UserId  NullValue[int64]",
		`bigquery:"user_id" spanner:"user_id"`,
		"Payload NullValue[[]byte]",
		`bigquery:"payload" spanner:"payload"`,
		"Amount  NullValue[*big.Rat]",
		`bigquery:"amount" spanner:"amount"`,
		"Profile *OrderRowProfile",
		`bigquery:"profile" spanner:"profile"`,
		"DisplayName NullValue[string]",
		"func (r *OrderRow) Load(values []bigquery.Value, schema bigquery.Schema) error",
		"func (r *OrderRowProfile) Load(values []bigquery.Value, schema bigquery.Schema) error",
		"type NullValue[T any] struct",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated code missing %q:\n%s", want, code)
		}
	}
}

func TestGenerateGoStructFromSpannerStructTypeSpanner(t *testing.T) {
	code, err := GenerateGoStructFromSpannerStructType(&spannerpb.StructType{
		Fields: []*spannerpb.StructType_Field{
			{Name: "SingerId", Type: &spannerpb.Type{Code: spannerpb.TypeCode_INT64}},
			{Name: "Name", Type: &spannerpb.Type{Code: spannerpb.TypeCode_STRING}},
			{Name: "Scores", Type: &spannerpb.Type{Code: spannerpb.TypeCode_ARRAY, ArrayElementType: &spannerpb.Type{Code: spannerpb.TypeCode_FLOAT64}}},
		},
	}, GoStructOptions{PackageName: "result", StructName: "SingerRow", Target: GoStructTargetSpanner})
	if err != nil {
		t.Fatalf("GenerateGoStructFromSpannerStructType() error = %v", err)
	}
	for _, want := range []string{
		"package result",
		"SingerId spanner.NullInt64",
		`spanner:"SingerId"`,
		"Name     spanner.NullString",
		`spanner:"Name"`,
		"Scores   []float64",
		`spanner:"Scores"`,
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated code missing %q:\n%s", want, code)
		}
	}
}

func TestGenerateGoStructsWithExtraReportsFormatError(t *testing.T) {
	_, err := generateGoStructsWithExtra(
		nil,
		GoStructOptions{PackageName: "result", StructName: "Row", Target: GoStructTargetSpanner},
		nil,
		nil,
		"func broken( {\n",
	)
	if err == nil {
		t.Fatal("generateGoStructsWithExtra() error = nil, want gofmt error")
	}
	if !strings.Contains(err.Error(), "gofmt generated source") {
		t.Fatalf("generateGoStructsWithExtra() error = %v, want gofmt context", err)
	}
}
