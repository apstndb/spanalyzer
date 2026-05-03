package spanalyzer

import (
	"fmt"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

func (t *TypeSpec) SpannerPB() (*spannerpb.Type, error) {
	if t == nil {
		return nil, fmt.Errorf("nil type spec")
	}
	if t.Tokenlist {
		return nil, fmt.Errorf("TOKENLIST cannot be represented as a Spanner protobuf type")
	}
	switch t.Code {
	case spannerpb.TypeCode_ARRAY:
		elem, err := t.ArrayElement.SpannerPB()
		if err != nil {
			return nil, err
		}
		return &spannerpb.Type{Code: spannerpb.TypeCode_ARRAY, ArrayElementType: elem}, nil
	case spannerpb.TypeCode_STRUCT:
		fields := make([]*spannerpb.StructType_Field, 0, len(t.StructFields))
		for _, field := range t.StructFields {
			fieldType, err := field.Type.SpannerPB()
			if err != nil {
				return nil, err
			}
			fields = append(fields, &spannerpb.StructType_Field{Name: field.Name, Type: fieldType})
		}
		return &spannerpb.Type{
			Code:       spannerpb.TypeCode_STRUCT,
			StructType: &spannerpb.StructType{Fields: fields},
		}, nil
	case spannerpb.TypeCode_PROTO, spannerpb.TypeCode_ENUM:
		return &spannerpb.Type{Code: t.Code, ProtoTypeFqn: t.ProtoTypeFQN}, nil
	default:
		return &spannerpb.Type{Code: t.Code}, nil
	}
}
