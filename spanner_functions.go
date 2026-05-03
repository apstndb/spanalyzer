package spanalyzer

import (
	"fmt"
	"strings"

	googlesql "github.com/goccy/go-googlesql"
)

func (a *Analyzer) addSpannerFunctionsToGoogleSQLCatalog() error {
	timestampType, err := a.typeFactory.GetTimestamp()
	if err != nil {
		return err
	}
	if err := a.addScalarFunction("PENDING_COMMIT_TIMESTAMP", timestampType, functionArgs()); err != nil {
		return err
	}

	int64Type, err := a.typeFactory.GetInt64()
	if err != nil {
		return err
	}

	boolType, err := a.typeFactory.GetBool()
	if err != nil {
		return err
	}
	floatType, err := a.typeFactory.GetFloat()
	if err != nil {
		return err
	}
	doubleType, err := a.typeFactory.GetDouble()
	if err != nil {
		return err
	}
	stringType, err := a.typeFactory.GetString()
	if err != nil {
		return err
	}
	bytesType, err := a.typeFactory.GetBytes()
	if err != nil {
		return err
	}
	jsonType, err := a.typeFactory.GetJson()
	if err != nil {
		return err
	}
	tokenlistType, err := a.typeFactory.GetTokenlist()
	if err != nil {
		return err
	}
	boolArrayType, err := a.typeFactory.MakeArrayType2(boolType)
	if err != nil {
		return err
	}
	int64ArrayType, err := a.typeFactory.MakeArrayType2(int64Type)
	if err != nil {
		return err
	}
	floatArrayType, err := a.typeFactory.MakeArrayType2(floatType)
	if err != nil {
		return err
	}
	doubleArrayType, err := a.typeFactory.MakeArrayType2(doubleType)
	if err != nil {
		return err
	}
	stringArrayType, err := a.typeFactory.MakeArrayType2(stringType)
	if err != nil {
		return err
	}
	bytesArrayType, err := a.typeFactory.MakeArrayType2(bytesType)
	if err != nil {
		return err
	}
	tokenlistArrayType, err := a.typeFactory.MakeArrayType2(tokenlistType)
	if err != nil {
		return err
	}
	categoryStructType, err := a.typeFactory.MakeStructType2([]*googlesql.StructField{
		{Type_: stringType},
		{Type_: stringType},
	})
	if err != nil {
		return err
	}
	namedCategoryStructType, err := a.typeFactory.MakeStructType2([]*googlesql.StructField{
		{Name: "label", Type_: stringType},
		{Name: "description", Type_: stringType},
	})
	if err != nil {
		return err
	}
	categoryStructArrayType, err := a.typeFactory.MakeArrayType2(categoryStructType)
	if err != nil {
		return err
	}
	namedCategoryStructArrayType, err := a.typeFactory.MakeArrayType2(namedCategoryStructType)
	if err != nil {
		return err
	}

	if err := a.addScalarFunction("BIT_REVERSE", int64Type,
		functionArgs(int64Type),
		functionArgs(int64Type, boolType),
	); err != nil {
		return err
	}
	if err := a.addScalarFunction("GET_NEXT_SEQUENCE_VALUE", int64Type,
		[]functionArgSpec{{sequence: true}},
		functionArgs(stringType),
	); err != nil {
		return err
	}
	if err := a.addScalarFunction("GET_INTERNAL_SEQUENCE_STATE", int64Type,
		[]functionArgSpec{{sequence: true}},
		functionArgs(stringType),
	); err != nil {
		return err
	}
	if err := a.addScalarFunction("GET_TABLE_COLUMN_IDENTITY_STATE", int64Type, functionArgs(stringType)); err != nil {
		return err
	}
	if err := a.addScalarFunction("AI_CLASSIFY", stringType,
		functionArgs(stringType, stringArrayType),
		functionArgs(stringType, categoryStructArrayType),
		functionArgs(stringType, namedCategoryStructArrayType),
	); err != nil {
		return err
	}
	if err := a.addScalarFunction("AI_IF", boolType, functionArgs(stringType)); err != nil {
		return err
	}
	if err := a.addScalarFunction("AI_SCORE", doubleType, functionArgs(stringType)); err != nil {
		return err
	}
	if err := a.addScalarFunction("DEBUG_TOKENLIST", stringType, functionArgs(tokenlistType)); err != nil {
		return err
	}
	if err := a.addScalarFunction("SCORE", doubleType,
		functionArgs(tokenlistType, stringType),
		functionArgs(tokenlistType, stringType, stringType),
		functionArgs(tokenlistType, stringType, stringType, stringType),
		functionArgs(tokenlistType, stringType, stringType, stringType, boolType),
		functionArgs(tokenlistType, stringType, stringType, stringType, boolType, stringType),
		functionArgs(tokenlistType, stringType, stringType, stringType, boolType, stringType, jsonType),
	); err != nil {
		return err
	}
	if err := a.addScalarFunction("SCORE_NGRAMS", doubleType,
		functionArgs(tokenlistType, stringType),
		functionArgs(tokenlistType, stringType, stringType),
		functionArgs(tokenlistType, stringType, stringType, stringType),
		functionArgs(tokenlistType, stringType, stringType, stringType, stringType),
	); err != nil {
		return err
	}
	if err := a.addScalarFunction("SEARCH", boolType,
		functionArgs(tokenlistType, stringType),
		functionArgs(tokenlistType, stringType, stringType),
		functionArgs(tokenlistType, stringType, stringType, stringType),
		functionArgs(tokenlistType, stringType, stringType, stringType, boolType),
		functionArgs(tokenlistType, stringType, stringType, stringType, boolType, stringType),
	); err != nil {
		return err
	}
	if err := a.addScalarFunction("SEARCH_NGRAMS", boolType,
		functionArgs(tokenlistType, stringType),
		functionArgs(tokenlistType, stringType, stringType),
		functionArgs(tokenlistType, stringType, stringType, int64Type),
		functionArgs(tokenlistType, stringType, stringType, int64Type, doubleType),
	); err != nil {
		return err
	}
	if err := a.addScalarFunction("SEARCH_SUBSTRING", boolType,
		functionArgs(tokenlistType, stringType),
		functionArgs(tokenlistType, stringType, stringType),
		functionArgs(tokenlistType, stringType, stringType, stringType),
	); err != nil {
		return err
	}
	if err := a.addScalarFunction("SNIPPET", jsonType,
		functionArgs(stringType, stringType),
		functionArgs(stringType, stringType, stringType),
		functionArgs(stringType, stringType, stringType, boolType),
		functionArgs(stringType, stringType, stringType, boolType, stringType),
		functionArgs(stringType, stringType, stringType, boolType, stringType, int64Type),
		functionArgs(stringType, stringType, stringType, boolType, stringType, int64Type, int64Type),
		functionArgs(stringType, stringType, stringType, boolType, stringType, int64Type, int64Type, stringType),
	); err != nil {
		return err
	}

	if err := a.addScalarFunction("TOKEN", tokenlistType,
		functionArgs(bytesType),
		functionArgs(bytesArrayType),
		functionArgs(stringType),
		functionArgs(stringArrayType),
	); err != nil {
		return err
	}
	if err := a.addScalarFunction("TOKENIZE_BOOL", tokenlistType,
		functionArgs(boolType),
		functionArgs(boolArrayType),
	); err != nil {
		return err
	}
	if err := a.addScalarFunction("TOKENIZE_FULLTEXT", tokenlistType,
		functionArgs(stringType),
		functionArgs(stringArrayType),
		functionArgs(stringType, stringType),
		functionArgs(stringArrayType, stringType),
		functionArgs(stringType, stringType, stringType),
		functionArgs(stringArrayType, stringType, stringType),
		functionArgs(stringType, stringType, stringType, stringType),
		functionArgs(stringArrayType, stringType, stringType, stringType),
		functionArgs(stringType, stringType, stringType, stringType, boolType),
		functionArgs(stringArrayType, stringType, stringType, stringType, boolType),
	); err != nil {
		return err
	}
	if err := a.addScalarFunction("TOKENIZE_JSON", tokenlistType, functionArgs(jsonType)); err != nil {
		return err
	}
	if err := a.addScalarFunction("TOKENIZE_NGRAMS", tokenlistType,
		functionArgs(stringType),
		functionArgs(stringArrayType),
		functionArgs(stringType, int64Type),
		functionArgs(stringArrayType, int64Type),
		functionArgs(stringType, int64Type, int64Type),
		functionArgs(stringArrayType, int64Type, int64Type),
		functionArgs(stringType, int64Type, int64Type, boolType),
		functionArgs(stringArrayType, int64Type, int64Type, boolType),
	); err != nil {
		return err
	}
	tokenizeNumberOverloads := tokenizeNumberFunctionArgs(int64Type, int64ArrayType, stringType, int64Type)
	tokenizeNumberOverloads = append(tokenizeNumberOverloads, tokenizeNumberFunctionArgs(floatType, floatArrayType, stringType, int64Type)...)
	tokenizeNumberOverloads = append(tokenizeNumberOverloads, tokenizeNumberFunctionArgs(doubleType, doubleArrayType, stringType, int64Type)...)
	if err := a.addScalarFunction("TOKENIZE_NUMBER", tokenlistType, tokenizeNumberOverloads...); err != nil {
		return err
	}
	if err := a.addScalarFunction("TOKENIZE_SUBSTRING", tokenlistType,
		functionArgs(stringType),
		functionArgs(stringArrayType),
		functionArgs(stringType, stringType),
		functionArgs(stringArrayType, stringType),
		functionArgs(stringType, stringType, int64Type),
		functionArgs(stringArrayType, stringType, int64Type),
		functionArgs(stringType, stringType, int64Type, int64Type),
		functionArgs(stringArrayType, stringType, int64Type, int64Type),
		functionArgs(stringType, stringType, int64Type, int64Type, stringArrayType),
		functionArgs(stringArrayType, stringType, int64Type, int64Type, stringArrayType),
		functionArgs(stringType, stringType, int64Type, int64Type, stringArrayType, stringType),
		functionArgs(stringArrayType, stringType, int64Type, int64Type, stringArrayType, stringType),
		functionArgs(stringType, stringType, int64Type, int64Type, stringArrayType, stringType, boolType),
		functionArgs(stringArrayType, stringType, int64Type, int64Type, stringArrayType, stringType, boolType),
		functionArgs(stringType, stringType, int64Type, int64Type, stringArrayType, stringType, boolType, boolType),
		functionArgs(stringArrayType, stringType, int64Type, int64Type, stringArrayType, stringType, boolType, boolType),
	); err != nil {
		return err
	}
	if err := a.addScalarFunction("TOKENLIST_CONCAT", tokenlistType,
		functionArgs(tokenlistArrayType),
		functionArgs(tokenlistType),
		functionArgs(tokenlistType, tokenlistType),
		functionArgs(tokenlistType, tokenlistType, tokenlistType),
	); err != nil {
		return err
	}
	return nil
}

type functionArgSpec struct {
	typ      googlesql.Googlesql_TypeNode
	sequence bool
}

func functionArgs(types ...googlesql.Googlesql_TypeNode) []functionArgSpec {
	args := make([]functionArgSpec, 0, len(types))
	for _, typ := range types {
		args = append(args, functionArgSpec{typ: typ})
	}
	return args
}

func (a *Analyzer) addScalarFunction(name string, resultType googlesql.Googlesql_TypeNode, overloads ...[]functionArgSpec) error {
	return a.addScalarFunctionAtPath([]string{name}, resultType, overloads...)
}

func (a *Analyzer) addScalarFunctionAtPath(namePath []string, resultType googlesql.Googlesql_TypeNode, overloads ...[]functionArgSpec) error {
	name := strings.Join(namePath, ".")
	resultArg, err := newFunctionArgumentType(resultType)
	if err != nil {
		return fmt.Errorf("function %s result: %w", name, err)
	}
	signatures := make([]*googlesql.FunctionSignature, 0, len(overloads))
	for _, args := range overloads {
		argTypes := make([]*googlesql.FunctionArgumentType, 0, len(args))
		for _, arg := range args {
			var argType *googlesql.FunctionArgumentType
			if arg.sequence {
				argType, err = googlesql.NewFunctionArgumentTypeAnySequence()
			} else {
				argType, err = newFunctionArgumentType(arg.typ)
			}
			if err != nil {
				return fmt.Errorf("function %s argument: %w", name, err)
			}
			argTypes = append(argTypes, argType)
		}
		signature, err := googlesql.NewFunctionSignature3(resultArg, argTypes, int64(googlesql.FunctionSignatureIdFnInvalidFunctionId))
		if err != nil {
			return fmt.Errorf("function %s signature: %w", name, err)
		}
		signatures = append(signatures, signature)
	}
	fn, err := googlesql.NewFunction(namePath, "Spanner", googlesql.FunctionEnums_ModeScalar, signatures, nil)
	if err != nil {
		return fmt.Errorf("function %s: %w", name, err)
	}
	return a.gsCatalog.AddFunction(fn)
}

func newFunctionArgumentType(typ googlesql.Googlesql_TypeNode) (*googlesql.FunctionArgumentType, error) {
	opts, err := googlesql.NewFunctionArgumentTypeOptions()
	if err != nil {
		return nil, err
	}
	return googlesql.NewFunctionArgumentType(typ, opts, -1)
}

func tokenizeNumberFunctionArgs(scalarType, arrayType, stringType, int64Type googlesql.Googlesql_TypeNode) [][]functionArgSpec {
	return [][]functionArgSpec{
		functionArgs(scalarType),
		functionArgs(arrayType),
		functionArgs(scalarType, stringType),
		functionArgs(arrayType, stringType),
		functionArgs(scalarType, stringType, stringType),
		functionArgs(arrayType, stringType, stringType),
		functionArgs(scalarType, stringType, stringType, scalarType),
		functionArgs(arrayType, stringType, stringType, scalarType),
		functionArgs(scalarType, stringType, stringType, scalarType, scalarType),
		functionArgs(arrayType, stringType, stringType, scalarType, scalarType),
		functionArgs(scalarType, stringType, stringType, scalarType, scalarType, scalarType),
		functionArgs(arrayType, stringType, stringType, scalarType, scalarType, scalarType),
		functionArgs(scalarType, stringType, stringType, scalarType, scalarType, scalarType, int64Type),
		functionArgs(arrayType, stringType, stringType, scalarType, scalarType, scalarType, int64Type),
		functionArgs(scalarType, stringType, stringType, scalarType, scalarType, scalarType, int64Type, int64Type),
		functionArgs(arrayType, stringType, stringType, scalarType, scalarType, scalarType, int64Type, int64Type),
	}
}
