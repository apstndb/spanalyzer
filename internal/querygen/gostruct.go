package querygen

import (
	"bytes"
	"fmt"
	"go/format"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

type GoStructTarget string

const (
	GoStructTargetBigQuery GoStructTarget = "bigquery"
	GoStructTargetSpanner  GoStructTarget = "spanner"
	GoStructTargetBoth     GoStructTarget = "both"
)

type GoStructOptions struct {
	PackageName string
	StructName  string
	Target      GoStructTarget
}

func GenerateGoStructFromSpannerStructType(rowType *spannerpb.StructType, options GoStructOptions) (string, error) {
	if rowType == nil {
		return "", fmt.Errorf("nil Spanner struct type")
	}
	fields := make([]goResultField, 0, len(rowType.Fields))
	for _, field := range rowType.Fields {
		fields = append(fields, goResultFieldFromSpanner(field.Name, field.Type))
	}
	return generateGoStruct(fields, normalizeGoStructOptions(options, GoStructTargetSpanner))
}

func GenerateGoStructFromBigQueryTableSchema(schema *BigQueryTableSchema, options GoStructOptions) (string, error) {
	if schema == nil {
		return "", fmt.Errorf("nil BigQuery table schema")
	}
	fields := make([]goResultField, 0, len(schema.Fields))
	for _, field := range schema.Fields {
		fields = append(fields, goResultFieldFromBigQuery(field))
	}
	return generateGoStruct(fields, normalizeGoStructOptions(options, GoStructTargetBigQuery))
}

type goResultField struct {
	Name     string
	Kind     string
	Repeated bool
	Nullable bool
	Fields   []goResultField
}

type generatedStruct struct {
	Name   string
	Fields []generatedField
}

type namedGoStruct struct {
	Name   string
	Fields []goResultField
}

type generatedGoConst struct {
	Name  string
	Value string
}

type generatedField struct {
	Name       string
	Type       string
	Tags       map[string]string
	SourceName string
	LoadKind   string
	Repeated   bool
}

type goType struct {
	Expr    string
	Imports map[string]string
}

func normalizeGoStructOptions(options GoStructOptions, defaultTarget GoStructTarget) GoStructOptions {
	if options.PackageName == "" {
		options.PackageName = "main"
	}
	if options.StructName == "" {
		options.StructName = "QueryRow"
	}
	if options.Target == "" {
		options.Target = defaultTarget
	}
	return options
}

func goResultFieldFromSpanner(name string, typ *spannerpb.Type) goResultField {
	if typ == nil {
		return goResultField{Name: name, Kind: "UNKNOWN", Nullable: true}
	}
	field := goResultField{Name: name, Nullable: true}
	switch typ.Code {
	case spannerpb.TypeCode_BOOL:
		field.Kind = "BOOL"
	case spannerpb.TypeCode_INT64:
		field.Kind = "INT64"
	case spannerpb.TypeCode_FLOAT32:
		field.Kind = "FLOAT32"
	case spannerpb.TypeCode_FLOAT64:
		field.Kind = "FLOAT64"
	case spannerpb.TypeCode_TIMESTAMP:
		field.Kind = "TIMESTAMP"
	case spannerpb.TypeCode_DATE:
		field.Kind = "DATE"
	case spannerpb.TypeCode_STRING:
		field.Kind = "STRING"
	case spannerpb.TypeCode_BYTES:
		field.Kind = "BYTES"
	case spannerpb.TypeCode_NUMERIC:
		field.Kind = "NUMERIC"
	case spannerpb.TypeCode_JSON:
		field.Kind = "JSON"
	case spannerpb.TypeCode_UUID:
		field.Kind = "UUID"
	case spannerpb.TypeCode_PROTO:
		field.Kind = "PROTO"
	case spannerpb.TypeCode_ENUM:
		field.Kind = "ENUM"
	case spannerpb.TypeCode_ARRAY:
		elem := goResultFieldFromSpanner(name, typ.ArrayElementType)
		elem.Repeated = true
		elem.Nullable = false
		return elem
	case spannerpb.TypeCode_STRUCT:
		field.Kind = "STRUCT"
		for _, nested := range typ.GetStructType().GetFields() {
			field.Fields = append(field.Fields, goResultFieldFromSpanner(nested.Name, nested.Type))
		}
	default:
		field.Kind = typ.Code.String()
	}
	return field
}

func goResultFieldFromBigQuery(field *BigQueryTableFieldSchema) goResultField {
	if field == nil {
		return goResultField{Kind: "UNKNOWN", Nullable: true}
	}
	out := goResultField{
		Name:     field.Name,
		Kind:     strings.ToUpper(field.Type),
		Repeated: strings.EqualFold(field.Mode, "REPEATED"),
		Nullable: !strings.EqualFold(field.Mode, "REQUIRED") && !strings.EqualFold(field.Mode, "REPEATED"),
	}
	if out.Kind == "RECORD" {
		out.Kind = "STRUCT"
	}
	for _, nested := range field.Fields {
		out.Fields = append(out.Fields, goResultFieldFromBigQuery(nested))
	}
	return out
}

func generateGoStruct(fields []goResultField, options GoStructOptions) (string, error) {
	return generateGoStructsWithExtra([]namedGoStruct{{
		Name:   options.StructName,
		Fields: fields,
	}}, options, nil, nil, "")
}

func generateGoStructsWithExtra(structs []namedGoStruct, options GoStructOptions, constants []generatedGoConst, extraImports []string, extraCode string) (string, error) {
	if err := validateGoStructOptions(options); err != nil {
		return "", err
	}
	gen := &goStructGenerator{
		target:  options.Target,
		imports: map[string]string{},
		used:    map[string]bool{},
	}
	roots := make([]generatedStruct, 0, len(structs))
	for _, st := range structs {
		name := st.Name
		if name == "" {
			name = options.StructName
		}
		roots = append(roots, gen.buildStruct(exportedIdentifier(name, "QueryRow"), st.Fields))
	}
	if gen.needsBigQueryLoad {
		gen.imports["cloud.google.com/go/bigquery"] = ""
		gen.imports["fmt"] = ""
	}
	for _, path := range extraImports {
		gen.imports[path] = ""
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "package %s\n\n", packageIdentifier(options.PackageName))
	if len(gen.imports) > 0 {
		paths := make([]string, 0, len(gen.imports))
		for path := range gen.imports {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		b.WriteString("import (\n")
		for _, path := range paths {
			fmt.Fprintf(&b, "\t%q\n", path)
		}
		b.WriteString(")\n\n")
	}
	if len(constants) > 0 {
		writeGoConstants(&b, constants)
		b.WriteByte('\n')
	}
	for i, root := range roots {
		if i > 0 {
			b.WriteByte('\n')
		}
		writeGeneratedStruct(&b, root)
	}
	for _, nested := range gen.structs {
		b.WriteByte('\n')
		writeGeneratedStruct(&b, nested)
	}
	if gen.needsBigQueryLoad {
		allStructs := append(roots, gen.structs...)
		for _, st := range allStructs {
			b.WriteByte('\n')
			writeBigQueryLoadMethod(&b, st)
		}
	}
	if gen.needsNullValue {
		b.WriteByte('\n')
		writeNullValueSupport(&b)
	}
	if gen.needsAssignValue && !gen.needsNullValue {
		b.WriteByte('\n')
		writeAssignBigQueryValueSupport(&b)
	}
	if gen.needsValueSlice {
		b.WriteByte('\n')
		writeBigQueryValueSliceSupport(&b)
	}
	if extraCode != "" {
		if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n\n") {
			b.WriteByte('\n')
		}
		b.WriteString(extraCode)
		if !strings.HasSuffix(extraCode, "\n") {
			b.WriteByte('\n')
		}
	}
	formatted, err := format.Source(b.Bytes())
	if err != nil {
		return b.String(), fmt.Errorf("gofmt generated source: %w\n%s", err, b.String())
	}
	return string(formatted), nil
}

func writeGoConstants(b *bytes.Buffer, constants []generatedGoConst) {
	b.WriteString("const (\n")
	used := map[string]bool{}
	for i, constant := range constants {
		name := exportedIdentifier(constant.Name, fmt.Sprintf("Query%dSQL", i+1))
		if !strings.HasSuffix(name, "SQL") {
			name += "SQL"
		}
		name = uniqueIdentifier(name, used)
		fmt.Fprintf(b, "\t%s = %s\n", name, strconv.Quote(constant.Value))
	}
	b.WriteString(")\n")
}

type goStructGenerator struct {
	target            GoStructTarget
	imports           map[string]string
	structs           []generatedStruct
	used              map[string]bool
	needsBigQueryLoad bool
	needsNullValue    bool
	needsAssignValue  bool
	needsValueSlice   bool
}

func (g *goStructGenerator) buildStruct(name string, fields []goResultField) generatedStruct {
	st := generatedStruct{Name: g.uniqueTypeName(name)}
	usedFields := map[string]bool{}
	for i, field := range fields {
		fieldName := exportedIdentifier(field.Name, fmt.Sprintf("Field%d", i+1))
		nestedName := st.Name + fieldName
		for _, out := range g.generatedFields(field, fieldName, nestedName) {
			out.Name = uniqueIdentifier(out.Name, usedFields)
			usedFields[out.Name] = true
			st.Fields = append(st.Fields, out)
		}
	}
	return st
}

func (g *goStructGenerator) generatedFields(field goResultField, fieldName, nestedName string) []generatedField {
	switch g.target {
	case GoStructTargetBigQuery:
		typ := g.typeForClient(field, "bigquery", nestedName)
		return []generatedField{{Name: fieldName, Type: typ.Expr, Tags: map[string]string{"bigquery": field.Name}, SourceName: field.Name}}
	case GoStructTargetSpanner:
		typ := g.typeForClient(field, "spanner", nestedName)
		return []generatedField{{Name: fieldName, Type: typ.Expr, Tags: map[string]string{"spanner": field.Name}, SourceName: field.Name}}
	default:
		g.needsBigQueryLoad = true
		typ := g.typeForClient(field, "both", nestedName)
		loadKind := "value"
		if field.Repeated && isStructLikeGoResultField(field) {
			loadKind = "struct_slice"
		} else if field.Repeated && strings.HasPrefix(typ.Expr, "[]NullValue[") {
			loadKind = "nullable_slice"
		} else if field.Repeated {
			loadKind = "slice"
		} else if strings.HasPrefix(typ.Expr, "NullValue[") {
			loadKind = "nullable"
		} else if isStructLikeGoResultField(field) {
			loadKind = "struct"
		}
		if loadKind == "value" || loadKind == "slice" {
			g.needsAssignValue = true
		}
		if loadKind == "slice" {
			g.needsValueSlice = true
		}
		return []generatedField{{Name: fieldName, Type: typ.Expr, Tags: map[string]string{"bigquery": field.Name, "spanner": field.Name}, SourceName: field.Name, LoadKind: loadKind, Repeated: field.Repeated}}
	}
}

func isStructLikeGoResultField(field goResultField) bool {
	if field.Kind == "STRUCT" {
		return true
	}
	if field.Repeated {
		elem := field
		elem.Repeated = false
		return elem.Kind == "STRUCT"
	}
	return false
}

func (g *goStructGenerator) typeForClient(field goResultField, client, nestedName string) goType {
	if field.Repeated {
		elem := field
		elem.Repeated = false
		elem.Nullable = false
		typ := g.typeForClient(elem, client, nestedName)
		return goType{Expr: "[]" + typ.Expr}
	}
	if field.Kind == "STRUCT" {
		st := g.buildStruct(nestedName, field.Fields)
		g.structs = append(g.structs, st)
		if field.Nullable {
			return goType{Expr: "*" + st.Name}
		}
		return goType{Expr: st.Name}
	}
	if client == "both" {
		return g.commonType(field)
	}
	if client == "bigquery" {
		return g.bigQueryType(field)
	}
	return g.spannerType(field)
}

func (g *goStructGenerator) commonType(field goResultField) goType {
	switch field.Kind {
	case "BOOL", "BOOLEAN":
		if !field.Nullable {
			return goType{Expr: "bool"}
		}
		return g.nullValue("bool")
	case "INT64", "INTEGER":
		if !field.Nullable {
			return goType{Expr: "int64"}
		}
		return g.nullValue("int64")
	case "FLOAT32":
		if !field.Nullable {
			return goType{Expr: "float32"}
		}
		return g.nullValue("float32")
	case "FLOAT64", "FLOAT", "DOUBLE":
		if !field.Nullable {
			return goType{Expr: "float64"}
		}
		return g.nullValue("float64")
	case "STRING", "GEOGRAPHY":
		if !field.Nullable {
			return goType{Expr: "string"}
		}
		return g.nullValue("string")
	case "BYTES":
		if !field.Nullable {
			return goType{Expr: "[]byte"}
		}
		return g.nullValue("[]byte")
	case "TIMESTAMP":
		g.imports["time"] = ""
		if !field.Nullable {
			return goType{Expr: "time.Time"}
		}
		return g.nullValue("time.Time")
	case "DATE":
		g.imports["cloud.google.com/go/civil"] = ""
		if !field.Nullable {
			return goType{Expr: "civil.Date"}
		}
		return g.nullValue("civil.Date")
	case "TIME":
		g.imports["cloud.google.com/go/civil"] = ""
		if !field.Nullable {
			return goType{Expr: "civil.Time"}
		}
		return g.nullValue("civil.Time")
	case "DATETIME":
		g.imports["cloud.google.com/go/civil"] = ""
		if !field.Nullable {
			return goType{Expr: "civil.DateTime"}
		}
		return g.nullValue("civil.DateTime")
	case "NUMERIC", "BIGNUMERIC":
		g.imports["math/big"] = ""
		if !field.Nullable {
			return goType{Expr: "*big.Rat"}
		}
		return g.nullValue("*big.Rat")
	case "JSON":
		if !field.Nullable {
			return goType{Expr: "any"}
		}
		return g.nullValue("any")
	default:
		if !field.Nullable {
			return goType{Expr: "any"}
		}
		return g.nullValue("any")
	}
}

func (g *goStructGenerator) nullValue(expr string) goType {
	g.needsNullValue = true
	return goType{Expr: "NullValue[" + expr + "]"}
}

func (g *goStructGenerator) bigQueryType(field goResultField) goType {
	switch field.Kind {
	case "BOOL", "BOOLEAN":
		if field.Nullable {
			return g.imported("cloud.google.com/go/bigquery", "bigquery.NullBool")
		}
		return goType{Expr: "bool"}
	case "INT64", "INTEGER":
		if field.Nullable {
			return g.imported("cloud.google.com/go/bigquery", "bigquery.NullInt64")
		}
		return goType{Expr: "int64"}
	case "FLOAT32", "FLOAT64", "FLOAT", "DOUBLE":
		if field.Nullable {
			return g.imported("cloud.google.com/go/bigquery", "bigquery.NullFloat64")
		}
		return goType{Expr: "float64"}
	case "STRING":
		if field.Nullable {
			return g.imported("cloud.google.com/go/bigquery", "bigquery.NullString")
		}
		return goType{Expr: "string"}
	case "BYTES":
		return goType{Expr: "[]byte"}
	case "TIMESTAMP":
		if field.Nullable {
			return g.imported("cloud.google.com/go/bigquery", "bigquery.NullTimestamp")
		}
		return g.imported("time", "time.Time")
	case "DATE":
		if field.Nullable {
			return g.imported("cloud.google.com/go/bigquery", "bigquery.NullDate")
		}
		return g.imported("cloud.google.com/go/civil", "civil.Date")
	case "TIME":
		if field.Nullable {
			return g.imported("cloud.google.com/go/bigquery", "bigquery.NullTime")
		}
		return g.imported("cloud.google.com/go/civil", "civil.Time")
	case "DATETIME":
		if field.Nullable {
			return g.imported("cloud.google.com/go/bigquery", "bigquery.NullDateTime")
		}
		return g.imported("cloud.google.com/go/civil", "civil.DateTime")
	case "NUMERIC", "BIGNUMERIC":
		return g.imported("math/big", "*big.Rat")
	case "JSON":
		if field.Nullable {
			return g.imported("cloud.google.com/go/bigquery", "bigquery.NullJSON")
		}
		return goType{Expr: "string"}
	case "GEOGRAPHY":
		if field.Nullable {
			return g.imported("cloud.google.com/go/bigquery", "bigquery.NullGeography")
		}
		return goType{Expr: "string"}
	case "RANGE":
		return g.imported("cloud.google.com/go/bigquery", "bigquery.RangeValue")
	default:
		return goType{Expr: "any"}
	}
}

func (g *goStructGenerator) spannerType(field goResultField) goType {
	switch field.Kind {
	case "BOOL", "BOOLEAN":
		if field.Nullable {
			return g.imported("cloud.google.com/go/spanner", "spanner.NullBool")
		}
		return goType{Expr: "bool"}
	case "INT64", "INTEGER":
		if field.Nullable {
			return g.imported("cloud.google.com/go/spanner", "spanner.NullInt64")
		}
		return goType{Expr: "int64"}
	case "FLOAT32":
		if field.Nullable {
			return g.imported("cloud.google.com/go/spanner", "spanner.NullFloat32")
		}
		return goType{Expr: "float32"}
	case "FLOAT64", "FLOAT", "DOUBLE":
		if field.Nullable {
			return g.imported("cloud.google.com/go/spanner", "spanner.NullFloat64")
		}
		return goType{Expr: "float64"}
	case "STRING":
		if field.Nullable {
			return g.imported("cloud.google.com/go/spanner", "spanner.NullString")
		}
		return goType{Expr: "string"}
	case "BYTES":
		return goType{Expr: "[]byte"}
	case "TIMESTAMP":
		if field.Nullable {
			return g.imported("cloud.google.com/go/spanner", "spanner.NullTime")
		}
		return g.imported("time", "time.Time")
	case "DATE":
		if field.Nullable {
			return g.imported("cloud.google.com/go/spanner", "spanner.NullDate")
		}
		return g.imported("cloud.google.com/go/civil", "civil.Date")
	case "NUMERIC", "BIGNUMERIC":
		return g.imported("math/big", "*big.Rat")
	case "JSON":
		if field.Nullable {
			return g.imported("cloud.google.com/go/spanner", "spanner.NullJSON")
		}
		return goType{Expr: "any"}
	case "UUID":
		if field.Nullable {
			return g.imported("cloud.google.com/go/spanner", "spanner.NullUUID")
		}
		return goType{Expr: "string"}
	case "PROTO":
		return g.imported("cloud.google.com/go/spanner", "spanner.NullProtoMessage")
	case "ENUM":
		return g.imported("cloud.google.com/go/spanner", "spanner.NullProtoEnum")
	default:
		return goType{Expr: "any"}
	}
}

func (g *goStructGenerator) imported(path, expr string) goType {
	g.imports[path] = ""
	return goType{Expr: expr}
}

func (g *goStructGenerator) uniqueTypeName(name string) string {
	name = exportedIdentifier(name, "GeneratedStruct")
	return uniqueIdentifier(name, g.used)
}

func writeGeneratedStruct(b *bytes.Buffer, st generatedStruct) {
	fmt.Fprintf(b, "type %s struct {\n", st.Name)
	for _, field := range st.Fields {
		fmt.Fprintf(b, "\t%s %s", field.Name, field.Type)
		if len(field.Tags) > 0 {
			fmt.Fprintf(b, " `%s`", structTag(field.Tags))
		}
		b.WriteByte('\n')
	}
	b.WriteString("}\n")
}

func writeBigQueryLoadMethod(b *bytes.Buffer, st generatedStruct) {
	fmt.Fprintf(b, "func (r *%s) Load(values []bigquery.Value, schema bigquery.Schema) error {\n", st.Name)
	b.WriteString("\tif len(values) != len(schema) {\n")
	b.WriteString("\t\treturn fmt.Errorf(\"bigquery row has %d values for %d schema fields\", len(values), len(schema))\n")
	b.WriteString("\t}\n")
	b.WriteString("\tfor i, field := range schema {\n")
	b.WriteString("\t\tswitch field.Name {\n")
	for _, field := range st.Fields {
		if field.SourceName == "" || field.LoadKind == "" {
			continue
		}
		fmt.Fprintf(b, "\t\tcase %q:\n", field.SourceName)
		switch field.LoadKind {
		case "nullable":
			fmt.Fprintf(b, "\t\t\tif err := r.%s.LoadBigQuery(values[i]); err != nil {\n", field.Name)
			fmt.Fprintf(b, "\t\t\t\treturn fmt.Errorf(%q, err)\n", field.SourceName+": %w")
			b.WriteString("\t\t\t}\n")
		case "nullable_slice":
			fmt.Fprintf(b, "\t\t\tif err := loadBigQueryNullValueSlice(&r.%s, values[i]); err != nil {\n", field.Name)
			fmt.Fprintf(b, "\t\t\t\treturn fmt.Errorf(%q, err)\n", field.SourceName+": %w")
			b.WriteString("\t\t\t}\n")
		case "slice":
			fmt.Fprintf(b, "\t\t\tif err := loadBigQueryValueSlice(&r.%s, values[i]); err != nil {\n", field.Name)
			fmt.Fprintf(b, "\t\t\t\treturn fmt.Errorf(%q, err)\n", field.SourceName+": %w")
			b.WriteString("\t\t\t}\n")
		case "struct":
			writeBigQueryStructLoad(b, field)
		case "struct_slice":
			writeBigQueryStructSliceLoad(b, field)
		default:
			fmt.Fprintf(b, "\t\t\tif err := assignBigQueryValue(&r.%s, values[i]); err != nil {\n", field.Name)
			fmt.Fprintf(b, "\t\t\t\treturn fmt.Errorf(%q, err)\n", field.SourceName+": %w")
			b.WriteString("\t\t\t}\n")
		}
	}
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn nil\n")
	b.WriteString("}\n")
}

func writeBigQueryStructLoad(b *bytes.Buffer, field generatedField) {
	if strings.HasPrefix(field.Type, "*") {
		nestedType := strings.TrimPrefix(field.Type, "*")
		b.WriteString("\t\t\tif values[i] == nil {\n")
		fmt.Fprintf(b, "\t\t\t\tr.%s = nil\n", field.Name)
		b.WriteString("\t\t\t\tcontinue\n")
		b.WriteString("\t\t\t}\n")
		b.WriteString("\t\t\tnestedValues, ok := values[i].([]bigquery.Value)\n")
		b.WriteString("\t\t\tif !ok {\n")
		fmt.Fprintf(b, "\t\t\t\treturn fmt.Errorf(%q, values[i])\n", field.SourceName+": cannot decode %T")
		b.WriteString("\t\t\t}\n")
		fmt.Fprintf(b, "\t\t\tr.%s = &%s{}\n", field.Name, nestedType)
		fmt.Fprintf(b, "\t\t\tif err := r.%s.Load(nestedValues, field.Schema); err != nil {\n", field.Name)
		fmt.Fprintf(b, "\t\t\t\treturn fmt.Errorf(%q, err)\n", field.SourceName+": %w")
		b.WriteString("\t\t\t}\n")
		return
	}
	b.WriteString("\t\t\tnestedValues, ok := values[i].([]bigquery.Value)\n")
	b.WriteString("\t\t\tif !ok {\n")
	fmt.Fprintf(b, "\t\t\t\treturn fmt.Errorf(%q, values[i])\n", field.SourceName+": cannot decode %T")
	b.WriteString("\t\t\t}\n")
	fmt.Fprintf(b, "\t\t\tif err := r.%s.Load(nestedValues, field.Schema); err != nil {\n", field.Name)
	fmt.Fprintf(b, "\t\t\t\treturn fmt.Errorf(%q, err)\n", field.SourceName+": %w")
	b.WriteString("\t\t\t}\n")
}

func writeBigQueryStructSliceLoad(b *bytes.Buffer, field generatedField) {
	elemType := strings.TrimPrefix(field.Type, "[]")
	b.WriteString("\t\t\tif values[i] == nil {\n")
	fmt.Fprintf(b, "\t\t\t\tr.%s = nil\n", field.Name)
	b.WriteString("\t\t\t\tcontinue\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t\trecords, ok := values[i].([]bigquery.Value)\n")
	b.WriteString("\t\t\tif !ok {\n")
	fmt.Fprintf(b, "\t\t\t\treturn fmt.Errorf(%q, values[i])\n", field.SourceName+": cannot decode %T")
	b.WriteString("\t\t\t}\n")
	fmt.Fprintf(b, "\t\t\tr.%s = make([]%s, len(records))\n", field.Name, elemType)
	b.WriteString("\t\t\tfor j, record := range records {\n")
	b.WriteString("\t\t\t\tnestedValues, ok := record.([]bigquery.Value)\n")
	b.WriteString("\t\t\t\tif !ok {\n")
	fmt.Fprintf(b, "\t\t\t\t\treturn fmt.Errorf(%q, j, record)\n", field.SourceName+"[%d]: cannot decode %T")
	b.WriteString("\t\t\t\t}\n")
	fmt.Fprintf(b, "\t\t\t\tif err := r.%s[j].Load(nestedValues, field.Schema); err != nil {\n", field.Name)
	fmt.Fprintf(b, "\t\t\t\t\treturn fmt.Errorf(%q, j, err)\n", field.SourceName+"[%d]: %w")
	b.WriteString("\t\t\t\t}\n")
	b.WriteString("\t\t\t}\n")
}

func writeNullValueSupport(b *bytes.Buffer) {
	b.WriteString("type NullValue[T any] struct {\n")
	b.WriteString("\tValue T\n")
	b.WriteString("\tValid bool\n")
	b.WriteString("}\n\n")
	b.WriteString("func (n NullValue[T]) IsNull() bool {\n")
	b.WriteString("\treturn !n.Valid\n")
	b.WriteString("}\n\n")
	b.WriteString("func (n NullValue[T]) EncodeSpanner() (interface{}, error) {\n")
	b.WriteString("\tif !n.Valid {\n")
	b.WriteString("\t\treturn nil, nil\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn n.Value, nil\n")
	b.WriteString("}\n\n")
	b.WriteString("func (n *NullValue[T]) LoadBigQuery(value bigquery.Value) error {\n")
	b.WriteString("\treturn n.set(value)\n")
	b.WriteString("}\n\n")
	b.WriteString("func (n *NullValue[T]) DecodeSpanner(input interface{}) error {\n")
	b.WriteString("\treturn n.set(input)\n")
	b.WriteString("}\n\n")
	b.WriteString("func (n *NullValue[T]) set(value interface{}) error {\n")
	b.WriteString("\tif value == nil {\n")
	b.WriteString("\t\tvar zero T\n")
	b.WriteString("\t\tn.Value = zero\n")
	b.WriteString("\t\tn.Valid = false\n")
	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t}\n")
	b.WriteString("\ttyped, ok := value.(T)\n")
	b.WriteString("\tif !ok {\n")
	b.WriteString("\t\treturn fmt.Errorf(\"cannot decode %T\", value)\n")
	b.WriteString("\t}\n")
	b.WriteString("\tn.Value = typed\n")
	b.WriteString("\tn.Valid = true\n")
	b.WriteString("\treturn nil\n")
	b.WriteString("}\n\n")
	b.WriteString("func loadBigQueryNullValueSlice[T any](dst *[]NullValue[T], value bigquery.Value) error {\n")
	b.WriteString("\tif value == nil {\n")
	b.WriteString("\t\t*dst = nil\n")
	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t}\n")
	b.WriteString("\tvalues, ok := value.([]bigquery.Value)\n")
	b.WriteString("\tif !ok {\n")
	b.WriteString("\t\treturn fmt.Errorf(\"cannot decode %T\", value)\n")
	b.WriteString("\t}\n")
	b.WriteString("\tout := make([]NullValue[T], len(values))\n")
	b.WriteString("\tfor i, value := range values {\n")
	b.WriteString("\t\tif err := out[i].LoadBigQuery(value); err != nil {\n")
	b.WriteString("\t\t\treturn fmt.Errorf(\"[%d]: %w\", i, err)\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("\t*dst = out\n")
	b.WriteString("\treturn nil\n")
	b.WriteString("}\n\n")
	writeAssignBigQueryValueSupport(b)
}

func writeAssignBigQueryValueSupport(b *bytes.Buffer) {
	b.WriteString("func assignBigQueryValue[T any](dst *T, value bigquery.Value) error {\n")
	b.WriteString("\tif value == nil {\n")
	b.WriteString("\t\tvar zero T\n")
	b.WriteString("\t\t*dst = zero\n")
	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t}\n")
	b.WriteString("\ttyped, ok := value.(T)\n")
	b.WriteString("\tif !ok {\n")
	b.WriteString("\t\treturn fmt.Errorf(\"cannot decode %T\", value)\n")
	b.WriteString("\t}\n")
	b.WriteString("\t*dst = typed\n")
	b.WriteString("\treturn nil\n")
	b.WriteString("}\n")
}

func writeBigQueryValueSliceSupport(b *bytes.Buffer) {
	b.WriteString("func loadBigQueryValueSlice[T any](dst *[]T, value bigquery.Value) error {\n")
	b.WriteString("\tif value == nil {\n")
	b.WriteString("\t\t*dst = nil\n")
	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t}\n")
	b.WriteString("\tvalues, ok := value.([]bigquery.Value)\n")
	b.WriteString("\tif !ok {\n")
	b.WriteString("\t\treturn fmt.Errorf(\"cannot decode %T\", value)\n")
	b.WriteString("\t}\n")
	b.WriteString("\tout := make([]T, len(values))\n")
	b.WriteString("\tfor i, value := range values {\n")
	b.WriteString("\t\tif err := assignBigQueryValue(&out[i], value); err != nil {\n")
	b.WriteString("\t\t\treturn fmt.Errorf(\"[%d]: %w\", i, err)\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("\t*dst = out\n")
	b.WriteString("\treturn nil\n")
	b.WriteString("}\n")
}

func structTag(tags map[string]string) string {
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf(`%s:"%s"`, key, tags[key]))
	}
	return strings.Join(parts, " ")
}

func validateGoStructOptions(options GoStructOptions) error {
	switch options.Target {
	case GoStructTargetBigQuery, GoStructTargetSpanner, GoStructTargetBoth:
		return nil
	default:
		return fmt.Errorf("unsupported Go struct target %q", options.Target)
	}
}

var nonIdent = regexp.MustCompile(`[^A-Za-z0-9_]+`)

func exportedIdentifier(value, fallback string) string {
	value = nonIdent.ReplaceAllString(value, "_")
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == '_' })
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		r, size := utf8.DecodeRuneInString(part)
		if r == utf8.RuneError {
			continue
		}
		b.WriteRune(unicode.ToUpper(r))
		b.WriteString(part[size:])
	}
	out := b.String()
	if out == "" {
		out = fallback
	}
	r, _ := utf8.DecodeRuneInString(out)
	if !unicode.IsLetter(r) {
		out = fallback + out
	}
	if isGoKeyword(out) {
		out += "Value"
	}
	return out
}

func packageIdentifier(value string) string {
	value = strings.ToLower(nonIdent.ReplaceAllString(value, "_"))
	value = strings.Trim(value, "_")
	if value == "" || isGoKeyword(value) {
		return "main"
	}
	r, _ := utf8.DecodeRuneInString(value)
	if !unicode.IsLetter(r) && r != '_' {
		return "main"
	}
	return value
}

func uniqueIdentifier(name string, used map[string]bool) string {
	base := name
	for i := 2; used[name]; i++ {
		name = fmt.Sprintf("%s%d", base, i)
	}
	used[name] = true
	return name
}

func isGoKeyword(value string) bool {
	switch value {
	case "break", "default", "func", "interface", "select", "case", "defer", "go", "map", "struct",
		"chan", "else", "goto", "package", "switch", "const", "fallthrough", "if", "range",
		"type", "continue", "for", "import", "return", "var":
		return true
	default:
		return false
	}
}
