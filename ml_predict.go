package spanalyzer

import (
	"fmt"
	"sort"
	"strings"

	googlesql "github.com/goccy/go-googlesql"
)

func (c *GoogleSQLCatalog) addMLPredict() error {
	if len(c.models) == 0 {
		return nil
	}
	fallback, err := c.firstRegisteredModel()
	if err != nil {
		return err
	}
	resultSchema, err := c.mlPredictResultSchema(fallback.model, nil)
	if err != nil {
		return err
	}
	resultArg, err := googlesql.NewFunctionArgumentTypeRelationWithSchema(resultSchema, true)
	if err != nil {
		return err
	}
	modelArg, err := googlesql.NewFunctionArgumentTypeAnyModel()
	if err != nil {
		return err
	}
	relationArg, err := googlesql.NewFunctionArgumentTypeAnyRelation()
	if err != nil {
		return err
	}
	optionsArg, err := googlesql.NewFunctionArgumentType4(googlesql.SignatureArgumentKindArgTypeAny3, 1)
	if err != nil {
		return err
	}
	sigOpts, err := googlesql.NewFunctionSignatureOptions()
	if err != nil {
		return err
	}
	twoArgSig, err := googlesql.NewFunctionSignature2(resultArg, []*googlesql.FunctionArgumentType{
		modelArg,
		relationArg,
	}, 0, sigOpts)
	if err != nil {
		return err
	}
	threeArgSig, err := googlesql.NewFunctionSignature2(resultArg, []*googlesql.FunctionArgumentType{
		modelArg,
		relationArg,
		optionsArg,
	}, 0, sigOpts)
	if err != nil {
		return err
	}
	signatures := []*googlesql.FunctionSignature{twoArgSig, threeArgSig}
	if err := c.addMLPredictTVF([]string{"ML", "PREDICT"}, signatures, fallback, c.SimpleCatalog); err != nil {
		return err
	}
	mlCatalog, _, err := simpleCatalogForObjectName(c.SimpleCatalog, c.simpleCatalogs, ObjectName{Parts: []string{"ML", "PREDICT"}})
	if err != nil {
		return err
	}
	if err := c.addMLPredictTVF([]string{"PREDICT"}, signatures, fallback, mlCatalog); err != nil {
		return err
	}
	safeMLCatalog, _, err := simpleCatalogForObjectName(c.SimpleCatalog, c.simpleCatalogs, ObjectName{Parts: []string{"SAFE", "ML", "PREDICT"}})
	if err != nil {
		return err
	}
	return c.addMLPredictTVF([]string{"PREDICT"}, signatures, fallback, safeMLCatalog)
}

func (c *GoogleSQLCatalog) addMLPredictTVF(namePath []string, signatures []*googlesql.FunctionSignature, fallback *registeredGoogleSQLModel, catalog *googlesql.SimpleCatalog) error {
	tvf, err := googlesql.NewTableValuedFunctionFromImpl(
		&mlPredictTVF{catalog: c, fallbackModel: fallback},
		namePath,
		"",
		signatures,
		&googlesql.TableValuedFunctionOptions{},
	)
	if err != nil {
		return err
	}
	return catalog.AddOwnedTableValuedFunction(tvf)
}

func (c *GoogleSQLCatalog) firstRegisteredModel() (*registeredGoogleSQLModel, error) {
	names := make([]string, 0, len(c.models))
	for name := range c.models {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("ML.PREDICT requires at least one model")
	}
	return c.models[names[0]], nil
}

func (c *GoogleSQLCatalog) mlPredictResultSchema(model *Model, relation *googlesql.TVFRelation) (*googlesql.TVFRelation, error) {
	columns := make([]*googlesql.TVFSchemaColumn, 0, len(model.Outputs)+len(model.Inputs))
	outputs, err := c.modelTVFSchemaColumns(model.Outputs)
	if err != nil {
		return nil, err
	}
	columns = append(columns, outputs...)
	outputNames := make(map[string]bool, len(outputs))
	for _, output := range outputs {
		outputNames[strings.ToLower(output.Name)] = true
	}
	if relation != nil {
		relationColumns, err := relation.Columns()
		if err != nil {
			return nil, err
		}
		for _, column := range relationColumns {
			if !outputNames[strings.ToLower(column.Name)] {
				columns = append(columns, column)
			}
		}
	} else {
		inputs, err := c.modelTVFSchemaColumns(model.Inputs)
		if err != nil {
			return nil, err
		}
		for _, column := range inputs {
			if !outputNames[strings.ToLower(column.Name)] {
				columns = append(columns, column)
			}
		}
	}
	return googlesql.NewTVFRelation(columns)
}

func (c *GoogleSQLCatalog) modelTVFSchemaColumns(columns []*ModelColumn) ([]*googlesql.TVFSchemaColumn, error) {
	out := make([]*googlesql.TVFSchemaColumn, 0, len(columns))
	for _, column := range columns {
		typ, err := c.TypeSpecToGoogleSQLType(column.Type)
		if err != nil {
			return nil, fmt.Errorf("model column %s: %w", column.Name, err)
		}
		out = append(out, &googlesql.TVFSchemaColumn{Name: column.Name, Type_: typ})
	}
	return out, nil
}

type mlPredictTVF struct {
	googlesql.TableValuedFunctionCallbackDefaults

	catalog       *GoogleSQLCatalog
	fallbackModel *registeredGoogleSQLModel
}

func (f *mlPredictTVF) Resolve(
	_ *googlesql.AnalyzerOptions,
	actualArguments []*googlesql.TVFInputArgumentType,
	concreteSignature *googlesql.FunctionSignature,
	_ googlesql.CatalogNode,
	typeFactory *googlesql.TypeFactory,
) (*googlesql.TVFSignature, error) {
	model := f.fallbackModel
	relation, err := relationArgument(actualArguments)
	if err != nil {
		return nil, err
	}
	if actualModel, err := f.modelArgument(actualArguments); err != nil {
		return nil, err
	} else if actualModel != nil {
		model = actualModel
	}
	resultSchema, err := f.catalog.mlPredictResultSchema(model.model, relation)
	if err != nil {
		return nil, err
	}
	inputs, err := f.inputArguments(actualArguments, concreteSignature, model, relation, typeFactory)
	if err != nil {
		return nil, err
	}
	return googlesql.NewTVFSignature(inputs, resultSchema, nil)
}

func (f *mlPredictTVF) modelArgument(args []*googlesql.TVFInputArgumentType) (*registeredGoogleSQLModel, error) {
	if len(args) == 0 {
		return nil, nil
	}
	isModel, err := args[0].IsModel()
	if err != nil || !isModel {
		return nil, err
	}
	modelArg, err := args[0].Model()
	if err != nil {
		return nil, err
	}
	modelNode, err := modelArg.Model()
	if err != nil {
		// go-googlesql can currently surface Go callback models as ModelTrampoline,
		// which cannot be decoded back to a ModelNode. The catalog fallback still
		// preserves Spanner's schema-only ML.PREDICT behavior for registered models.
		return nil, nil
	}
	name, err := modelNode.Name()
	if err != nil {
		return nil, err
	}
	model := f.catalog.models[strings.ToLower(name)]
	if model == nil {
		return nil, fmt.Errorf("model %s is not registered", name)
	}
	return model, nil
}

func relationArgument(args []*googlesql.TVFInputArgumentType) (*googlesql.TVFRelation, error) {
	if len(args) < 2 {
		return nil, nil
	}
	isRelation, err := args[1].IsRelation()
	if err != nil || !isRelation {
		return nil, err
	}
	return args[1].Relation()
}

func (f *mlPredictTVF) inputArguments(args []*googlesql.TVFInputArgumentType, concreteSignature *googlesql.FunctionSignature, model *registeredGoogleSQLModel, relation *googlesql.TVFRelation, typeFactory *googlesql.TypeFactory) ([]*googlesql.TVFInputArgumentType, error) {
	if len(args) > 0 {
		return args, nil
	}
	modelArg, err := googlesql.NewTVFModelArgument(model.node)
	if err != nil {
		return nil, err
	}
	modelInput, err := googlesql.NewTVFInputArgumentType2(modelArg)
	if err != nil {
		return nil, err
	}
	if relation == nil {
		relation, err = f.catalog.modelInputRelation(model.model)
		if err != nil {
			return nil, err
		}
	}
	relationInput, err := googlesql.NewTVFInputArgumentType(relation)
	if err != nil {
		return nil, err
	}
	inputs := []*googlesql.TVFInputArgumentType{modelInput, relationInput}
	count, err := concreteArgumentCount(concreteSignature)
	if err != nil {
		return nil, err
	}
	if count >= 3 {
		optionsInput, err := mlPredictOptionsInput(typeFactory)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, optionsInput)
	}
	return inputs, nil
}

func (c *GoogleSQLCatalog) modelInputRelation(model *Model) (*googlesql.TVFRelation, error) {
	inputs, err := c.modelTVFSchemaColumns(model.Inputs)
	if err != nil {
		return nil, err
	}
	return googlesql.NewTVFRelation(inputs)
}

func concreteArgumentCount(signature *googlesql.FunctionSignature) (int32, error) {
	if signature == nil {
		return 2, nil
	}
	hasConcreteArguments, err := signature.HasConcreteArguments()
	if err != nil {
		return 0, err
	}
	if !hasConcreteArguments {
		return 2, nil
	}
	return signature.NumConcreteArguments()
}

func mlPredictOptionsInput(typeFactory *googlesql.TypeFactory) (*googlesql.TVFInputArgumentType, error) {
	optionsType, err := typeFactory.MakeStructType2(nil)
	if err != nil {
		return nil, err
	}
	optionsArg, err := googlesql.NewInputArgumentType5(optionsType, false, false)
	if err != nil {
		return nil, err
	}
	return googlesql.NewTVFInputArgumentType6(optionsArg)
}
