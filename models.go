package spanalyzer

import (
	"fmt"
	"strings"

	googlesql "github.com/goccy/go-googlesql"
)

func (c *GoogleSQLCatalog) addModels() error {
	for _, model := range c.SpannerCatalog.Models {
		impl, err := c.newGoogleSQLModel(model)
		if err != nil {
			return err
		}
		gsModel, err := googlesql.NewModelFromImpl(impl)
		if err != nil {
			return fmt.Errorf("model %s: %w", model.Name, err)
		}
		if err := c.SimpleCatalog.AddOwnedModel(gsModel); err != nil {
			return fmt.Errorf("model %s: %w", model.Name, err)
		}
		c.models[strings.ToLower(model.Name)] = &registeredGoogleSQLModel{
			model: model,
			node:  gsModel,
		}
	}
	return nil
}

func (c *GoogleSQLCatalog) newGoogleSQLModel(model *Model) (*googleSQLModel, error) {
	inputs, err := c.modelColumns(model.Name, model.Inputs)
	if err != nil {
		return nil, err
	}
	outputs, err := c.modelColumns(model.Name, model.Outputs)
	if err != nil {
		return nil, err
	}
	return &googleSQLModel{name: model.Name, inputs: inputs, outputs: outputs}, nil
}

func (c *GoogleSQLCatalog) modelColumns(modelName string, columns []*ModelColumn) ([]googlesql.Googlesql_ColumnNode, error) {
	out := make([]googlesql.Googlesql_ColumnNode, 0, len(columns))
	for _, column := range columns {
		typ, err := c.TypeSpecToGoogleSQLType(column.Type)
		if err != nil {
			return nil, fmt.Errorf("model %s column %s: %w", modelName, column.Name, err)
		}
		gsColumn, err := googlesql.NewSimpleColumn(modelName, column.Name, typ, false, true)
		if err != nil {
			return nil, fmt.Errorf("model %s column %s: %w", modelName, column.Name, err)
		}
		out = append(out, gsColumn)
	}
	return out, nil
}

type googleSQLModel struct {
	googlesql.ModelCallbackDefaults

	name    string
	inputs  []googlesql.Googlesql_ColumnNode
	outputs []googlesql.Googlesql_ColumnNode
}

type registeredGoogleSQLModel struct {
	model *Model
	node  googlesql.ModelNode
}

func (m *googleSQLModel) Name() (string, error) {
	return m.name, nil
}

func (m *googleSQLModel) FullName() (string, error) {
	return m.name, nil
}

func (m *googleSQLModel) NumInputs() (uint64, error) {
	return uint64(len(m.inputs)), nil
}

func (m *googleSQLModel) NumOutputs() (uint64, error) {
	return uint64(len(m.outputs)), nil
}

func (m *googleSQLModel) GetInput(i int32) (googlesql.Googlesql_ColumnNode, error) {
	return modelColumnAt(m.inputs, i)
}

func (m *googleSQLModel) GetOutput(i int32) (googlesql.Googlesql_ColumnNode, error) {
	return modelColumnAt(m.outputs, i)
}

func (m *googleSQLModel) FindInputByName(name string) (googlesql.Googlesql_ColumnNode, error) {
	return findModelColumn(m.inputs, name)
}

func (m *googleSQLModel) FindOutputByName(name string) (googlesql.Googlesql_ColumnNode, error) {
	return findModelColumn(m.outputs, name)
}

func modelColumnAt(columns []googlesql.Googlesql_ColumnNode, i int32) (googlesql.Googlesql_ColumnNode, error) {
	if i < 0 || int(i) >= len(columns) {
		return nil, nil
	}
	return columns[i], nil
}

func findModelColumn(columns []googlesql.Googlesql_ColumnNode, name string) (googlesql.Googlesql_ColumnNode, error) {
	for _, column := range columns {
		columnName, err := column.Name()
		if err != nil {
			return nil, err
		}
		if strings.EqualFold(columnName, name) {
			return column, nil
		}
	}
	return nil, nil
}
