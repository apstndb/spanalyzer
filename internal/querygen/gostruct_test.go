package querygen

import (
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

//go:embed testdata/spannerclient.mod
var generatedClientGoMod string

//go:embed testdata/spannerclient.sum
var generatedClientGoSum string

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
		"Scores   []spanner.NullFloat64",
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

func TestGenerateGoStructDuplicateNestedTypeFailsClosed(t *testing.T) {
	var firstErr string
	for i := 0; i < 5; i++ {
		code, err := generateGoStruct([]goResultField{
			{
				Name:   "info",
				Kind:   "STRUCT",
				Fields: []goResultField{{Name: "n", Kind: "STRING"}},
			},
			{
				Name:   "Info",
				Kind:   "STRUCT",
				Fields: []goResultField{{Name: "n", Kind: "STRING"}},
			},
		}, GoStructOptions{PackageName: "result", StructName: "Row", Target: GoStructTargetSpanner})
		if err == nil {
			t.Fatalf("generateGoStruct() error = nil, want nested type collision\n%s", code)
		}
		if code != "" {
			t.Fatalf("generateGoStruct() returned source on error:\n%s", code)
		}
		if !strings.Contains(err.Error(), "generated symbol RowInfo") {
			t.Fatalf("error = %v, want RowInfo", err)
		}
		if !strings.Contains(err.Error(), "field info") || !strings.Contains(err.Error(), "field Info") {
			t.Fatalf("error = %v, want dual field origins", err)
		}
		if firstErr == "" {
			firstErr = err.Error()
		} else if err.Error() != firstErr {
			t.Fatalf("generateGoStruct collision diagnostic is not deterministic:\nfirst: %s\nlater: %s", firstErr, err.Error())
		}
	}

	code, err := GenerateGoStructFromSpannerStructType(&spannerpb.StructType{
		Fields: []*spannerpb.StructType_Field{
			{
				Name: "info",
				Type: &spannerpb.Type{
					Code: spannerpb.TypeCode_STRUCT,
					StructType: &spannerpb.StructType{Fields: []*spannerpb.StructType_Field{
						{Name: "n", Type: &spannerpb.Type{Code: spannerpb.TypeCode_STRING}},
					}},
				},
			},
			{
				Name: "Info",
				Type: &spannerpb.Type{
					Code: spannerpb.TypeCode_STRUCT,
					StructType: &spannerpb.StructType{Fields: []*spannerpb.StructType_Field{
						{Name: "n", Type: &spannerpb.Type{Code: spannerpb.TypeCode_STRING}},
					}},
				},
			},
		},
	}, GoStructOptions{PackageName: "result", StructName: "Row", Target: GoStructTargetSpanner})
	if err == nil {
		t.Fatalf("GenerateGoStructFromSpannerStructType() error = nil, want nested type collision\n%s", code)
	}
	if code != "" {
		t.Fatalf("GenerateGoStructFromSpannerStructType() returned source on error:\n%s", code)
	}
	if err.Error() != firstErr {
		t.Fatalf("public Go-struct path error = %v, want %s", err, firstErr)
	}
}

func TestGenerateGoStructsWithExtraDuplicateRootAndNestedTypeFailsClosed(t *testing.T) {
	var firstErr string
	for i := 0; i < 5; i++ {
		code, err := generateGoStructsWithExtra([]namedGoStruct{
			{
				Name: "Row",
				Fields: []goResultField{{
					Name:   "info",
					Kind:   "STRUCT",
					Fields: []goResultField{{Name: "n", Kind: "STRING"}},
				}},
			},
			{Name: "RowInfo", Fields: []goResultField{{Name: "id", Kind: "INT64"}}},
		}, GoStructOptions{PackageName: "result", StructName: "QueryRow", Target: GoStructTargetSpanner}, nil, nil, "")
		if err == nil {
			t.Fatalf("generateGoStructsWithExtra() error = nil, want type collision\n%s", code)
		}
		if code != "" {
			t.Fatalf("generateGoStructsWithExtra() returned source on error:\n%s", code)
		}
		if !strings.Contains(err.Error(), "generated symbol RowInfo") {
			t.Fatalf("error = %v, want RowInfo", err)
		}
		if !strings.Contains(err.Error(), "generated struct Row nested struct RowInfo field info") || !strings.Contains(err.Error(), "generated struct RowInfo") {
			t.Fatalf("error = %v, want dual struct origins", err)
		}
		if firstErr == "" {
			firstErr = err.Error()
		} else if err.Error() != firstErr {
			t.Fatalf("generateGoStructsWithExtra collision diagnostic is not deterministic:\nfirst: %s\nlater: %s", firstErr, err.Error())
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

func TestGeneratedDefaultDTODecodesSpannerClientRow(t *testing.T) {
	code, err := generateGoStruct([]goResultField{
		{Name: "id", Kind: "INT64", Nullable: true},
		{Name: "missing", Kind: "INT64", Nullable: true},
		{Name: "f64", Kind: "FLOAT64", Nullable: true},
		{Name: "nan", Kind: "FLOAT64", Nullable: true},
		{Name: "posinf", Kind: "FLOAT64", Nullable: true},
		{Name: "neginf", Kind: "FLOAT64", Nullable: true},
		{Name: "f32", Kind: "FLOAT32", Nullable: true},
		{Name: "ok", Kind: "BOOL", Nullable: true},
		{Name: "name", Kind: "STRING", Nullable: true},
		{Name: "payload", Kind: "BYTES", Nullable: true},
		{Name: "ts", Kind: "TIMESTAMP", Nullable: true},
		{Name: "d", Kind: "DATE", Nullable: true},
		{Name: "tm", Kind: "TIME", Nullable: true},
		{Name: "dt", Kind: "DATETIME", Nullable: true},
		{Name: "num", Kind: "NUMERIC", Nullable: true},
	}, GoStructOptions{PackageName: "result", StructName: "Row", Target: GoStructTargetBoth})
	if err != nil {
		t.Fatalf("generateGoStruct() error = %v", err)
	}
	for _, want := range []string{"NullValue[int64]", "NullValue[[]byte]", "NullValue[time.Time]", "NullValue[civil.Date]", "NullValue[*big.Rat]", `spanner:"id"`} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated DTO missing %q:\n%s", want, code)
		}
	}

	genDir := t.TempDir()
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "go.mod"), generatedClientGoMod)
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "go.sum"), generatedClientGoSum)
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "generated.go"), code)
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "generated_test.go"), `package result

import (
	"bytes"
	"math"
	"math/big"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	sppb "cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestSpannerAndBigQuery(t *testing.T) {
	when := time.Date(2026, 9, 23, 1, 2, 3, 0, time.UTC)
	date := civil.Date{Year: 2026, Month: 9, Day: 23}
	clock := civil.Time{Hour: 1, Minute: 2, Second: 3}
	dateTime := civil.DateTime{Date: date, Time: clock}
	num := big.NewRat(3, 2)
	payload := []byte{1, 2, 3}
	row, err := spanner.NewRow(
		[]string{"id", "missing", "f64", "nan", "posinf", "neginf", "f32", "ok", "name", "payload", "ts", "d", "num"},
		[]interface{}{int64(42), (*string)(nil), 1.5, math.NaN(), math.Inf(1), math.Inf(-1), float32(1.25), true, "ok", payload, when, date, num},
	)
	if err != nil {
		t.Fatal(err)
	}
	var dst Row
	if err := row.ToStruct(&dst); err != nil {
		t.Fatal(err)
	}
	if !dst.Id.Valid || dst.Id.Value != 42 {
		t.Fatalf("Id = %+v", dst.Id)
	}
	if dst.Missing.Valid {
		t.Fatalf("Missing = %+v, want NULL", dst.Missing)
	}
	if !dst.F64.Valid || dst.F64.Value != 1.5 {
		t.Fatalf("F64 = %+v", dst.F64)
	}
	if !dst.Nan.Valid || !math.IsNaN(dst.Nan.Value) {
		t.Fatalf("Nan = %+v", dst.Nan)
	}
	if !dst.Posinf.Valid || !math.IsInf(dst.Posinf.Value, 1) {
		t.Fatalf("Posinf = %+v", dst.Posinf)
	}
	if !dst.Neginf.Valid || !math.IsInf(dst.Neginf.Value, -1) {
		t.Fatalf("Neginf = %+v", dst.Neginf)
	}
	if !dst.F32.Valid || dst.F32.Value != 1.25 {
		t.Fatalf("F32 = %+v", dst.F32)
	}
	if !dst.Ok.Valid || !dst.Ok.Value {
		t.Fatalf("Ok = %+v", dst.Ok)
	}
	if !dst.Name.Valid || dst.Name.Value != "ok" {
		t.Fatalf("Name = %+v", dst.Name)
	}
	if !dst.Payload.Valid || !bytes.Equal(dst.Payload.Value, payload) {
		t.Fatalf("Payload = %+v", dst.Payload)
	}
	if !dst.Ts.Valid || !dst.Ts.Value.Equal(when) {
		t.Fatalf("Ts = %+v", dst.Ts)
	}
	if !dst.D.Valid || dst.D.Value != date {
		t.Fatalf("D = %+v", dst.D)
	}
	// NewRow encodes civil.Time as a struct. TIME and DATETIME arrive on the
	// wire as strings, so use the same client entry point ToStruct uses.
	// Spanner v1.91.0 has no TIME or DATETIME type codes. Those values still
	// arrive at DecodeSpanner as strings, which STRING delivers.
	if err := (spanner.GenericColumnValue{Type: &sppb.Type{Code: sppb.TypeCode_STRING}, Value: structpb.NewStringValue("01:02:03")}).Decode(&dst.Tm); err != nil {
		t.Fatal(err)
	}
	if err := (spanner.GenericColumnValue{Type: &sppb.Type{Code: sppb.TypeCode_STRING}, Value: structpb.NewStringValue("2026-09-23t01:02:03")}).Decode(&dst.Dt); err != nil {
		t.Fatal(err)
	}
	if !dst.Tm.Valid || dst.Tm.Value != clock {
		t.Fatalf("Tm = %+v", dst.Tm)
	}
	if !dst.Dt.Valid || dst.Dt.Value != dateTime {
		t.Fatalf("Dt = %+v", dst.Dt)
	}
	if !dst.Num.Valid || dst.Num.Value.Cmp(num) != 0 {
		t.Fatalf("Num = %+v", dst.Num)
	}

	var loaded Row
	if err := loaded.Load([]bigquery.Value{int64(42), nil}, bigquery.Schema{{Name: "id"}, {Name: "missing"}}); err != nil {
		t.Fatal(err)
	}
	if !loaded.Id.Valid || loaded.Id.Value != 42 || loaded.Missing.Valid {
		t.Fatalf("BigQuery load = %+v", loaded)
	}
}
`)
	cmd := exec.Command("go", "test", ".")
	cmd.Dir = genDir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test generated package: %v\n%s\n--- generated ---\n%s", err, output, code)
	}
}

func TestGeneratedSpannerArrayDecodesNullElements(t *testing.T) {
	code, err := GenerateGoStructFromSpannerStructType(&spannerpb.StructType{
		Fields: []*spannerpb.StructType_Field{{
			Name: "ids",
			Type: &spannerpb.Type{Code: spannerpb.TypeCode_ARRAY, ArrayElementType: &spannerpb.Type{Code: spannerpb.TypeCode_INT64}},
		}},
	}, GoStructOptions{PackageName: "result", StructName: "Row", Target: GoStructTargetSpanner})
	if err != nil {
		t.Fatalf("GenerateGoStructFromSpannerStructType() error = %v", err)
	}
	if !strings.Contains(code, "[]spanner.NullInt64") {
		t.Fatalf("generated array lost element nullability:\n%s", code)
	}
	genDir := t.TempDir()
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "generated.go"), code)
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "generated_test.go"), `package result

import (
	"testing"

	"cloud.google.com/go/spanner"
)

func TestNullElement(t *testing.T) {
	row, err := spanner.NewRow([]string{"ids"}, []interface{}{[]spanner.NullInt64{{Int64: 42, Valid: true}, {}}})
	if err != nil {
		t.Fatal(err)
	}
	var dst Row
	if err := row.ToStruct(&dst); err != nil {
		t.Fatal(err)
	}
	if len(dst.Ids) != 2 || !dst.Ids[0].Valid || dst.Ids[0].Int64 != 42 || dst.Ids[1].Valid {
		t.Fatalf("Ids = %+v", dst.Ids)
	}
}
`)
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "go.mod"), generatedClientGoMod)
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "go.sum"), generatedClientGoSum)
	cmd := exec.Command("go", "test", ".")
	cmd.Dir = genDir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test generated package: %v\n%s\n--- generated ---\n%s", err, output, code)
	}
}

func TestGeneratedBothTargetArrayDecodesNullElements(t *testing.T) {
	code, err := generateGoStruct([]goResultField{{
		Name:     "ids",
		Kind:     "INT64",
		Nullable: true,
		Repeated: true,
	}}, GoStructOptions{PackageName: "result", StructName: "Row", Target: GoStructTargetBoth})
	if err != nil {
		t.Fatalf("generateGoStruct() error = %v", err)
	}
	if !strings.Contains(code, "NullValueList[int64]") {
		t.Fatalf("both-target array lost element nullability:\n%s", code)
	}

	genDir := t.TempDir()
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "go.mod"), generatedClientGoMod)
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "go.sum"), generatedClientGoSum)
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "generated.go"), code)
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "generated_test.go"), `package result

import (
	"testing"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/bigquery"
)

func TestNullElement(t *testing.T) {
	element, err := spanner.NewRow([]string{"ids"}, []interface{}{[]spanner.NullInt64{{Int64: 42, Valid: true}, {}}})
	if err != nil {
		t.Fatal(err)
	}
	var withNull Row
	if err := element.ToStruct(&withNull); err != nil {
		t.Fatal(err)
	}
	if len(withNull.Ids) != 2 || !withNull.Ids[0].Valid || withNull.Ids[0].Value != 42 || withNull.Ids[1].Valid {
		t.Fatalf("NULL element = %+v", withNull.Ids)
	}
	empty, err := spanner.NewRow([]string{"ids"}, []interface{}{[]spanner.NullInt64{}})
	if err != nil {
		t.Fatal(err)
	}
	var emptyRow Row
	if err := empty.ToStruct(&emptyRow); err != nil {
		t.Fatal(err)
	}
	if emptyRow.Ids == nil || len(emptyRow.Ids) != 0 {
		t.Fatalf("empty array = %#v", emptyRow.Ids)
	}
	nullArray, err := spanner.NewRow([]string{"ids"}, []interface{}{[]spanner.NullInt64(nil)})
	if err != nil {
		t.Fatal(err)
	}
	var nullRow Row
	if err := nullArray.ToStruct(&nullRow); err != nil {
		t.Fatal(err)
	}
	if nullRow.Ids != nil {
		t.Fatalf("NULL array = %#v, want nil", nullRow.Ids)
	}
	var loaded Row
	if err := loaded.Load([]bigquery.Value{[]bigquery.Value{int64(7), nil}}, bigquery.Schema{{Name: "ids"}}); err != nil {
		t.Fatal(err)
	}
	if len(loaded.Ids) != 2 || !loaded.Ids[0].Valid || loaded.Ids[0].Value != 7 || loaded.Ids[1].Valid {
		t.Fatalf("BigQuery array = %#v", loaded.Ids)
	}
}
`)
	cmd := exec.Command("go", "test", ".")
	cmd.Dir = genDir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test generated package: %v\n%s\n--- generated ---\n%s", err, output, code)
	}
}

func TestGeneratedBothTargetNullableStructArrayLoads(t *testing.T) {
	code, err := generateGoStruct([]goResultField{{
		Name: "records", Kind: "STRUCT", Repeated: true, Nullable: true,
		Fields: []goResultField{{Name: "id", Kind: "INT64", Nullable: true}},
	}}, GoStructOptions{PackageName: "result", StructName: "Row", Target: GoStructTargetBoth})
	if err != nil {
		t.Fatal(err)
	}
	genDir := t.TempDir()
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "go.mod"), generatedClientGoMod)
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "go.sum"), generatedClientGoSum)
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "generated.go"), code)
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "generated_test.go"), `package result

import (
	"testing"
	"cloud.google.com/go/bigquery"
)

func TestLoadStructArray(t *testing.T) {
	var dst Row
	values := []bigquery.Value{[]bigquery.Value{[]bigquery.Value{int64(7)}, nil}}
	schema := bigquery.Schema{{Name: "records", Schema: bigquery.Schema{{Name: "id"}}}}
	if err := dst.Load(values, schema); err != nil {
		t.Fatal(err)
	}
	if len(dst.Records) != 2 || dst.Records[0] == nil || !dst.Records[0].Id.Valid || dst.Records[0].Id.Value != 7 || dst.Records[1] != nil {
		t.Fatalf("records = %#v", dst.Records)
	}
}
`)
	cmd := exec.Command("go", "test", ".")
	cmd.Dir = genDir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated struct array: %v\n%s", err, output)
	}
}

func writeGeneratedLoadTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}
