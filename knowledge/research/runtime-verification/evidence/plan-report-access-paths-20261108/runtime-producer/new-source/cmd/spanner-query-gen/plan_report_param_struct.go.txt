package main

import (
	"fmt"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	spanalyzer "github.com/apstndb/spanalyzer"
	"google.golang.org/protobuf/types/known/structpb"
)

func planReportGenericParamValue(spec *spanalyzer.TypeSpec) (spanner.GenericColumnValue, error) {
	typ, err := spec.SpannerPB()
	if err != nil {
		return spanner.GenericColumnValue{}, err
	}
	value, err := planReportEncodedParamValue(spec)
	if err != nil {
		return spanner.GenericColumnValue{}, err
	}
	// The SDK encodes GenericColumnValue directly. []GenericColumnValue is
	// explicitly unsupported, so an ARRAY<STRUCT> is encoded as one typed value.
	return spanner.GenericColumnValue{Type: typ, Value: value}, nil
}

func planReportEncodedParamValue(spec *spanalyzer.TypeSpec) (*structpb.Value, error) {
	if spec == nil {
		return nil, fmt.Errorf("nil parameter type")
	}
	switch spec.Code {
	case spannerpb.TypeCode_STRUCT:
		values := make([]*structpb.Value, 0, len(spec.StructFields))
		for _, field := range spec.StructFields {
			value, err := planReportEncodedParamValue(field.Type)
			if err != nil {
				return nil, fmt.Errorf("STRUCT field %q: %w", field.Name, err)
			}
			values = append(values, value)
		}
		return structpb.NewListValue(&structpb.ListValue{Values: values}), nil
	case spannerpb.TypeCode_ARRAY:
		if spec.ArrayElement == nil || spec.ArrayElement.Code == spannerpb.TypeCode_ARRAY {
			return nil, fmt.Errorf("ARRAY requires a non-ARRAY element type")
		}
		value, err := planReportEncodedParamValue(spec.ArrayElement)
		if err != nil {
			return nil, err
		}
		return structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{value}}), nil
	case spannerpb.TypeCode_BOOL:
		return structpb.NewBoolValue(true), nil
	case spannerpb.TypeCode_INT64, spannerpb.TypeCode_NUMERIC:
		return structpb.NewStringValue("1"), nil
	case spannerpb.TypeCode_FLOAT32, spannerpb.TypeCode_FLOAT64:
		return structpb.NewNumberValue(1), nil
	case spannerpb.TypeCode_STRING:
		return structpb.NewStringValue("value"), nil
	case spannerpb.TypeCode_BYTES:
		return structpb.NewStringValue("dmFsdWU="), nil
	case spannerpb.TypeCode_DATE:
		return structpb.NewStringValue("2026-05-06"), nil
	case spannerpb.TypeCode_TIMESTAMP:
		return structpb.NewStringValue("1970-01-01T00:00:00Z"), nil
	case spannerpb.TypeCode_JSON:
		return structpb.NewStringValue(`{"value":1}`), nil
	default:
		return nil, fmt.Errorf("unsupported plan-report parameter type %s", spec.Code)
	}
}
