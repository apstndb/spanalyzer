package spannersys

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/civil"
)

// typeKind identifies the structural form of a SPANNER_TYPE descriptor.
type typeKind string

const (
	// typeKindScalar identifies a scalar Spanner type.
	typeKindScalar typeKind = "scalar"
	// typeKindArray identifies an ARRAY type.
	typeKindArray typeKind = "array"
	// typeKindStruct identifies a STRUCT type with ordered fields.
	typeKindStruct typeKind = "struct"
)

// scalarCode identifies a scalar Spanner type.
type scalarCode string

const (
	scalarBool      scalarCode = "BOOL"
	scalarInt64     scalarCode = "INT64"
	scalarFloat64   scalarCode = "FLOAT64"
	scalarStringMax scalarCode = "STRING(MAX)"
	scalarBytesMax  scalarCode = "BYTES(MAX)"
	scalarDate      scalarCode = "DATE"
	scalarTimestamp scalarCode = "TIMESTAMP"
)

// typeDescriptor is a structural Spanner type. Decoder pointer information is
// retained separately and never changes canonical SPANNER_TYPE rendering.
type typeDescriptor struct {
	Kind                   typeKind                `json:"kind"`
	Scalar                 scalarCode              `json:"scalar,omitempty"`
	Element                *typeDescriptor         `json:"element,omitempty"`
	ElementDecoderNullable bool                    `json:"element_decoder_nullable,omitempty"`
	Fields                 []structFieldDescriptor `json:"fields,omitempty"`
}

// structFieldDescriptor is one ordered field in a STRUCT type.
type structFieldDescriptor struct {
	Name            string         `json:"name"`
	Type            typeDescriptor `json:"type"`
	DecoderNullable bool           `json:"decoder_nullable"`
}

// columnDescriptor describes one registered SPANNER_SYS decoder column. Its
// position in tableDescriptor.Columns is declaration order, not an observed
// live ordinal; live ordinals belong to ColumnObservation.
type columnDescriptor struct {
	Name            string         `json:"name"`
	Type            typeDescriptor `json:"type"`
	DecoderNullable bool           `json:"decoder_nullable"`
}

// tableDescriptor describes one fully expanded SPANNER_SYS table decoder.
type tableDescriptor struct {
	Name    string             `json:"name"`
	Columns []columnDescriptor `json:"columns"`
}

var (
	boolType      = reflect.TypeFor[bool]()
	int64Type     = reflect.TypeFor[int64]()
	float64Type   = reflect.TypeFor[float64]()
	stringType    = reflect.TypeFor[string]()
	byteSliceType = reflect.TypeFor[[]byte]()
	timeType      = reflect.TypeFor[time.Time]()
	dateType      = reflect.TypeFor[civil.Date]()
)

// registryDescriptors returns a fresh, deterministic structural description
// of every table and column registered by this package.
func registryDescriptors() ([]tableDescriptor, error) {
	registry, err := tableRegistry()
	if err != nil {
		return nil, err
	}
	return descriptorsFromRegistry(registry)
}

func descriptorsFromRegistry(registry map[string]reflect.Type) ([]tableDescriptor, error) {
	tableNames := make([]string, 0, len(registry))
	for tableName := range registry {
		tableNames = append(tableNames, tableName)
	}
	sort.Strings(tableNames)

	tables := make([]tableDescriptor, 0, len(tableNames))
	for _, tableName := range tableNames {
		rowType := registry[tableName]
		if rowType == nil {
			return nil, fmt.Errorf("SPANNER_SYS.%s has a nil row type", tableName)
		}
		if rowType.Kind() != reflect.Struct {
			return nil, fmt.Errorf("SPANNER_SYS.%s row type %s is not a struct", tableName, rowType)
		}

		columns := make([]columnDescriptor, 0, rowType.NumField())
		seen := make(map[string]bool, rowType.NumField())
		for i := 0; i < rowType.NumField(); i++ {
			field := rowType.Field(i)
			if field.PkgPath != "" {
				return nil, fmt.Errorf("SPANNER_SYS.%s field %s is not exported", tableName, field.Name)
			}
			columnName := field.Tag.Get("spanner")
			if columnName == "" {
				return nil, fmt.Errorf("SPANNER_SYS.%s field %s has no spanner tag", tableName, field.Name)
			}
			if seen[columnName] {
				return nil, fmt.Errorf("SPANNER_SYS.%s has duplicate spanner tag %s", tableName, columnName)
			}
			seen[columnName] = true

			columnType, nullable, err := describeDecoderType(field.Type, make(map[reflect.Type]bool))
			if err != nil {
				return nil, fmt.Errorf("SPANNER_SYS.%s.%s: %w", tableName, columnName, err)
			}
			columns = append(columns, columnDescriptor{
				Name:            columnName,
				Type:            columnType,
				DecoderNullable: nullable,
			})
		}
		tables = append(tables, tableDescriptor{Name: tableName, Columns: columns})
	}
	return tables, nil
}

func describeDecoderType(
	t reflect.Type,
	structStack map[reflect.Type]bool,
) (typeDescriptor, bool, error) {
	unwrapped, nullable := unwrapPointer(t)
	descriptor, err := describeType(unwrapped, structStack)
	return descriptor, nullable, err
}

func describeType(t reflect.Type, structStack map[reflect.Type]bool) (typeDescriptor, error) {
	switch t {
	case boolType:
		return scalarDescriptor(scalarBool), nil
	case int64Type:
		return scalarDescriptor(scalarInt64), nil
	case float64Type:
		return scalarDescriptor(scalarFloat64), nil
	case stringType:
		return scalarDescriptor(scalarStringMax), nil
	case byteSliceType:
		return scalarDescriptor(scalarBytesMax), nil
	case dateType:
		return scalarDescriptor(scalarDate), nil
	case timeType:
		return scalarDescriptor(scalarTimestamp), nil
	}

	switch t.Kind() {
	case reflect.Slice:
		elementType, nullable := unwrapPointer(t.Elem())
		element, err := describeType(elementType, structStack)
		if err != nil {
			return typeDescriptor{}, fmt.Errorf("array element: %w", err)
		}
		if element.Kind == typeKindArray {
			return typeDescriptor{}, fmt.Errorf("nested arrays are not supported")
		}
		return typeDescriptor{
			Kind:                   typeKindArray,
			Element:                &element,
			ElementDecoderNullable: nullable,
		}, nil
	case reflect.Struct:
		if structStack[t] {
			return typeDescriptor{}, fmt.Errorf("recursive struct type %s is not supported", t)
		}
		structStack[t] = true
		defer delete(structStack, t)

		fields := make([]structFieldDescriptor, 0, t.NumField())
		seen := make(map[string]bool, t.NumField())
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" {
				return typeDescriptor{}, fmt.Errorf("struct %s field %s is not exported", t, field.Name)
			}
			fieldName := field.Tag.Get("spanner")
			if fieldName == "" {
				return typeDescriptor{}, fmt.Errorf("struct %s field %s has no spanner tag", t, field.Name)
			}
			if seen[fieldName] {
				return typeDescriptor{}, fmt.Errorf("struct %s has duplicate spanner tag %s", t, fieldName)
			}
			seen[fieldName] = true

			fieldType, nullable, err := describeDecoderType(field.Type, structStack)
			if err != nil {
				return typeDescriptor{}, fmt.Errorf("struct field %s: %w", fieldName, err)
			}
			fields = append(fields, structFieldDescriptor{
				Name:            fieldName,
				Type:            fieldType,
				DecoderNullable: nullable,
			})
		}
		if len(fields) == 0 {
			return typeDescriptor{}, fmt.Errorf("empty struct type %s is not supported", t)
		}
		return typeDescriptor{Kind: typeKindStruct, Fields: fields}, nil
	default:
		return typeDescriptor{}, fmt.Errorf("unsupported Go decoder type %s", t)
	}
}

func unwrapPointer(t reflect.Type) (reflect.Type, bool) {
	nullable := false
	for t.Kind() == reflect.Pointer {
		nullable = true
		t = t.Elem()
	}
	return t, nullable
}

func scalarDescriptor(code scalarCode) typeDescriptor {
	return typeDescriptor{Kind: typeKindScalar, Scalar: code}
}

// canonicalSpannerType renders a structural descriptor using the raw spelling
// returned by INFORMATION_SCHEMA.COLUMNS.SPANNER_TYPE.
func canonicalSpannerType(descriptor typeDescriptor) (string, error) {
	switch descriptor.Kind {
	case typeKindScalar:
		if descriptor.Element != nil || len(descriptor.Fields) != 0 || descriptor.ElementDecoderNullable {
			return "", fmt.Errorf("scalar descriptor contains non-scalar fields")
		}
		switch descriptor.Scalar {
		case scalarBool, scalarInt64, scalarFloat64, scalarStringMax,
			scalarBytesMax, scalarDate, scalarTimestamp:
			return string(descriptor.Scalar), nil
		default:
			return "", fmt.Errorf("unsupported scalar code %q", descriptor.Scalar)
		}
	case typeKindArray:
		if descriptor.Scalar != "" || descriptor.Element == nil || len(descriptor.Fields) != 0 {
			return "", fmt.Errorf("malformed array descriptor")
		}
		if descriptor.Element.Kind == typeKindArray {
			return "", fmt.Errorf("nested arrays are not supported")
		}
		element, err := canonicalSpannerType(*descriptor.Element)
		if err != nil {
			return "", fmt.Errorf("array element: %w", err)
		}
		return "ARRAY<" + element + ">", nil
	case typeKindStruct:
		if descriptor.Scalar != "" || descriptor.Element != nil || descriptor.ElementDecoderNullable || len(descriptor.Fields) == 0 {
			return "", fmt.Errorf("malformed struct descriptor")
		}
		fields := make([]string, 0, len(descriptor.Fields))
		seen := make(map[string]bool, len(descriptor.Fields))
		for _, field := range descriptor.Fields {
			if field.Name == "" {
				return "", fmt.Errorf("struct field has an empty name")
			}
			if seen[field.Name] {
				return "", fmt.Errorf("duplicate struct field %s", field.Name)
			}
			seen[field.Name] = true
			fieldType, err := canonicalSpannerType(field.Type)
			if err != nil {
				return "", fmt.Errorf("struct field %s: %w", field.Name, err)
			}
			fields = append(fields, field.Name+" "+fieldType)
		}
		return "STRUCT<" + strings.Join(fields, ", ") + ">", nil
	default:
		return "", fmt.Errorf("unsupported descriptor kind %q", descriptor.Kind)
	}
}

// compareLiveTypes compares raw live SPANNER_TYPE observations with registry
// descriptors. Missing observations are skipped; availability is evaluated by
// the separate known-absence and required-target policies.
func compareLiveTypes(
	descriptors []tableDescriptor,
	observations []ColumnObservation,
) ([]ColumnTypeMismatch, error) {
	expected := make(map[string]map[string]string, len(descriptors))
	for _, table := range descriptors {
		if table.Name == "" {
			return nil, fmt.Errorf("descriptor contains an empty table name")
		}
		if expected[table.Name] != nil {
			return nil, fmt.Errorf("duplicate table descriptor %s", table.Name)
		}
		expected[table.Name] = make(map[string]string, len(table.Columns))
		for _, column := range table.Columns {
			if column.Name == "" {
				return nil, fmt.Errorf("table %s contains an empty column name", table.Name)
			}
			if _, ok := expected[table.Name][column.Name]; ok {
				return nil, fmt.Errorf("table %s has duplicate column descriptor %s", table.Name, column.Name)
			}
			canonical, err := canonicalSpannerType(column.Type)
			if err != nil {
				return nil, fmt.Errorf("table %s column %s: %w", table.Name, column.Name, err)
			}
			expected[table.Name][column.Name] = canonical
		}
	}

	var mismatches []ColumnTypeMismatch
	for _, observation := range observations {
		table, ok := expected[observation.TableName]
		if !ok {
			continue
		}
		expectedType, ok := table[observation.ColumnName]
		if !ok || observation.SpannerType == expectedType {
			continue
		}
		mismatches = append(mismatches, ColumnTypeMismatch{
			TableName:    observation.TableName,
			ColumnName:   observation.ColumnName,
			ObservedType: observation.SpannerType,
			ExpectedType: expectedType,
		})
	}
	sort.Slice(mismatches, func(i, j int) bool {
		if mismatches[i].TableName != mismatches[j].TableName {
			return mismatches[i].TableName < mismatches[j].TableName
		}
		return mismatches[i].ColumnName < mismatches[j].ColumnName
	})
	return mismatches, nil
}
