package spanalyzer

import (
	"fmt"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	googlesql "github.com/goccy/go-googlesql"
)

func typeSpecToGoogleSQLType(tf *googlesql.TypeFactory, spec *TypeSpec) (googlesql.Googlesql_TypeNode, error) {
	return typeSpecToGoogleSQLTypeWithProto(tf, spec, nil)
}

func (a *Analyzer) typeSpecToGoogleSQLType(spec *TypeSpec) (googlesql.Googlesql_TypeNode, error) {
	return typeSpecToGoogleSQLTypeWithProto(a.typeFactory, spec, a.catalog)
}

func typeSpecToGoogleSQLTypeWithProto(tf *googlesql.TypeFactory, spec *TypeSpec, catalog *Catalog) (googlesql.Googlesql_TypeNode, error) {
	if spec == nil {
		return nil, fmt.Errorf("nil type spec")
	}
	if spec.Tokenlist {
		return tf.GetTokenlist()
	}
	switch spec.Code {
	case spannerpb.TypeCode_BOOL:
		return tf.GetBool()
	case spannerpb.TypeCode_INT64:
		return tf.GetInt64()
	case spannerpb.TypeCode_FLOAT32:
		return tf.GetFloat()
	case spannerpb.TypeCode_FLOAT64:
		return tf.GetDouble()
	case spannerpb.TypeCode_TIMESTAMP:
		return tf.GetTimestamp()
	case spannerpb.TypeCode_DATE:
		return tf.GetDate()
	case spannerpb.TypeCode_STRING:
		return tf.GetString()
	case spannerpb.TypeCode_BYTES:
		return tf.GetBytes()
	case spannerpb.TypeCode_NUMERIC:
		return tf.GetNumeric()
	case spannerpb.TypeCode_JSON:
		return tf.GetJson()
	case spannerpb.TypeCode_INTERVAL:
		return tf.GetInterval()
	case spannerpb.TypeCode_UUID:
		return tf.GetUuid()
	case spannerpb.TypeCode_ARRAY:
		elem, err := typeSpecToGoogleSQLTypeWithProto(tf, spec.ArrayElement, catalog)
		if err != nil {
			return nil, err
		}
		return tf.MakeArrayType2(elem)
	case spannerpb.TypeCode_STRUCT:
		fields := make([]*googlesql.StructField, 0, len(spec.StructFields))
		for _, field := range spec.StructFields {
			fieldType, err := typeSpecToGoogleSQLTypeWithProto(tf, field.Type, catalog)
			if err != nil {
				return nil, err
			}
			fields = append(fields, &googlesql.StructField{Name: field.Name, Type_: fieldType})
		}
		return tf.MakeStructType2(fields)
	case spannerpb.TypeCode_PROTO:
		if catalog == nil {
			return nil, fmt.Errorf("proto type %s requires a proto descriptor set", spec.ProtoTypeFQN)
		}
		shadow, err := catalog.protoShadowStructType(spec.ProtoTypeFQN)
		if err != nil {
			return nil, err
		}
		return typeSpecToGoogleSQLTypeWithProto(tf, shadow, catalog)
	case spannerpb.TypeCode_ENUM:
		// go-googlesql v0.1.0 does not expose enum type construction. INT64
		// keeps expressions analyzable; direct enum column outputs are mapped
		// back to Spanner ENUM row metadata after analysis.
		return tf.GetInt64()
	default:
		return nil, fmt.Errorf("unsupported Spanner type code %s", spec.Code)
	}
}

func googleSQLTypeToSpannerPB(t googlesql.Googlesql_TypeNode) (*spannerpb.Type, error) {
	if t == nil {
		return nil, fmt.Errorf("nil GoogleSQL type")
	}
	kind, err := t.Kind()
	if err != nil {
		return nil, err
	}
	switch kind {
	case googlesql.TypeKindTypeBool:
		return scalarPB(spannerpb.TypeCode_BOOL), nil
	case googlesql.TypeKindTypeInt64:
		return scalarPB(spannerpb.TypeCode_INT64), nil
	case googlesql.TypeKindTypeFloat:
		return scalarPB(spannerpb.TypeCode_FLOAT32), nil
	case googlesql.TypeKindTypeDouble:
		return scalarPB(spannerpb.TypeCode_FLOAT64), nil
	case googlesql.TypeKindTypeTimestamp:
		return scalarPB(spannerpb.TypeCode_TIMESTAMP), nil
	case googlesql.TypeKindTypeDate:
		return scalarPB(spannerpb.TypeCode_DATE), nil
	case googlesql.TypeKindTypeString:
		return scalarPB(spannerpb.TypeCode_STRING), nil
	case googlesql.TypeKindTypeBytes:
		return scalarPB(spannerpb.TypeCode_BYTES), nil
	case googlesql.TypeKindTypeNumeric:
		return scalarPB(spannerpb.TypeCode_NUMERIC), nil
	case googlesql.TypeKindTypeJson:
		return scalarPB(spannerpb.TypeCode_JSON), nil
	case googlesql.TypeKindTypeInterval:
		return scalarPB(spannerpb.TypeCode_INTERVAL), nil
	case googlesql.TypeKindTypeUuid:
		return scalarPB(spannerpb.TypeCode_UUID), nil
	case googlesql.TypeKindTypeTokenlist:
		return nil, fmt.Errorf("TOKENLIST cannot be represented as a Spanner protobuf row type")
	case googlesql.TypeKindTypeArray:
		arrayType, err := t.AsArray()
		if err != nil {
			return nil, err
		}
		elemType, err := arrayType.ElementType()
		if err != nil {
			return nil, err
		}
		elemPB, err := googleSQLTypeToSpannerPB(elemType)
		if err != nil {
			return nil, err
		}
		return &spannerpb.Type{Code: spannerpb.TypeCode_ARRAY, ArrayElementType: elemPB}, nil
	case googlesql.TypeKindTypeStruct:
		structType, err := t.AsStruct()
		if err != nil {
			return nil, err
		}
		fields, err := structType.Fields()
		if err != nil {
			return nil, err
		}
		pbFields := make([]*spannerpb.StructType_Field, 0, len(fields))
		for _, field := range fields {
			fieldPB, err := googleSQLTypeToSpannerPB(field.Type_)
			if err != nil {
				return nil, err
			}
			pbFields = append(pbFields, &spannerpb.StructType_Field{Name: field.Name, Type: fieldPB})
		}
		return &spannerpb.Type{
			Code:       spannerpb.TypeCode_STRUCT,
			StructType: &spannerpb.StructType{Fields: pbFields},
		}, nil
	case googlesql.TypeKindTypeProto:
		protoType, err := t.AsProto()
		if err != nil {
			return nil, err
		}
		fqn, err := protoType.TypeName()
		if err != nil {
			return nil, err
		}
		return &spannerpb.Type{Code: spannerpb.TypeCode_PROTO, ProtoTypeFqn: fqn}, nil
	case googlesql.TypeKindTypeEnum:
		enumType, err := t.AsEnum()
		if err != nil {
			return nil, err
		}
		fqn, err := enumType.TypeName()
		if err != nil {
			return nil, err
		}
		return &spannerpb.Type{Code: spannerpb.TypeCode_ENUM, ProtoTypeFqn: fqn}, nil
	default:
		debug, _ := t.DebugString(false)
		return nil, fmt.Errorf("unsupported GoogleSQL type kind %s (%s)", kind, debug)
	}
}

func scalarPB(code spannerpb.TypeCode) *spannerpb.Type {
	return &spannerpb.Type{Code: code}
}
