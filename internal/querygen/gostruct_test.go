package querygen

import (
	"os"
	"os/exec"
	"path/filepath"
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

func TestGenerateGoStructFromBigQueryTableSchemaBothLoadsRepeatedPrimitives(t *testing.T) {
	code, err := GenerateGoStructFromBigQueryTableSchema(&BigQueryTableSchema{
		Fields: []*BigQueryTableFieldSchema{
			{Name: "numbers", Type: "INTEGER", Mode: "REPEATED"},
			{Name: "labels", Type: "STRING", Mode: "REPEATED"},
		},
	}, GoStructOptions{PackageName: "result", StructName: "ArrayRow", Target: GoStructTargetBoth})
	if err != nil {
		t.Fatalf("GenerateGoStructFromBigQueryTableSchema() error = %v", err)
	}

	dir := t.TempDir()
	writeGeneratedLoadTestFile(t, filepath.Join(dir, "go.mod"), `module generatedloadtest

go 1.22

require cloud.google.com/go/bigquery v0.0.0

replace cloud.google.com/go/bigquery => ./bigquerystub
`)
	writeGeneratedLoadTestFile(t, filepath.Join(dir, "generated.go"), code)
	writeGeneratedLoadTestFile(t, filepath.Join(dir, "generated_test.go"), `package result

import (
	"reflect"
	"strings"
	"testing"

	"cloud.google.com/go/bigquery"
)

func TestArrayRowLoad(t *testing.T) {
	schema := bigquery.Schema{
		{Name: "numbers"},
		{Name: "labels"},
	}
	var row ArrayRow
	if err := row.Load([]bigquery.Value{
		[]bigquery.Value{int64(1), int64(2)},
		[]bigquery.Value{"one", "two"},
	}, schema); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if want := []int64{1, 2}; !reflect.DeepEqual(row.Numbers, want) {
		t.Errorf("Numbers = %#v, want %#v", row.Numbers, want)
	}
	if want := []string{"one", "two"}; !reflect.DeepEqual(row.Labels, want) {
		t.Errorf("Labels = %#v, want %#v", row.Labels, want)
	}

	if err := row.Load([]bigquery.Value{
		[]bigquery.Value{},
		[]bigquery.Value{},
	}, schema); err != nil {
		t.Fatalf("Load(empty arrays) error = %v", err)
	}
	if row.Numbers == nil || row.Labels == nil {
		t.Fatalf("Load(empty arrays) = %#v, want non-nil empty slices", row)
	}

	if err := row.Load([]bigquery.Value{nil, nil}, schema); err != nil {
		t.Fatalf("Load(nil arrays) error = %v", err)
	}
	if row.Numbers != nil || row.Labels != nil {
		t.Fatalf("Load(nil arrays) = %#v, want nil slices", row)
	}

	row.Numbers = []int64{42}
	err := row.Load([]bigquery.Value{
		[]bigquery.Value{int64(1), "bad"},
		[]bigquery.Value{},
	}, schema)
	if err == nil || !strings.Contains(err.Error(), "numbers: [1]: cannot decode string") {
		t.Fatalf("Load(bad element) error = %v, want field and element index context", err)
	}
	if want := []int64{42}; !reflect.DeepEqual(row.Numbers, want) {
		t.Errorf("Numbers after failed Load = %#v, want unchanged %#v", row.Numbers, want)
	}
}
`)
	stubDir := filepath.Join(dir, "bigquerystub")
	if err := os.Mkdir(stubDir, 0o755); err != nil {
		t.Fatalf("Mkdir(bigquerystub) error = %v", err)
	}
	writeGeneratedLoadTestFile(t, filepath.Join(stubDir, "go.mod"), `module cloud.google.com/go/bigquery

go 1.22
`)
	writeGeneratedLoadTestFile(t, filepath.Join(stubDir, "bigquery.go"), `package bigquery

type Value interface{}

type Schema []*FieldSchema

type FieldSchema struct {
	Name   string
	Schema Schema
}
`)

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GOCACHE="+filepath.Join(dir, "gocache"),
		"GOTOOLCHAIN=local",
		"GOWORK=off",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test generated loader: %v\n--- generated.go ---\n%s\n--- output ---\n%s", err, code, output)
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

func writeGeneratedLoadTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}
