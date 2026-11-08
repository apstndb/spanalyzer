package spanalyzer

import (
	"fmt"
	"strings"

	googlesql "github.com/goccy/go-googlesql"
)

func (c *GoogleSQLCatalog) addModels() error {
	for _, model := range c.SpannerCatalog.Models {
		// Register as SimpleModel so ML.PREDICT can decode the named
		// argument back to a ModelNode. Go callback models surface as
		// ModelTrampoline and cannot be identified in Resolve.
		gsModel, err := c.newGoogleSQLModel(model)
		if err != nil {
			return err
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

func (c *GoogleSQLCatalog) newGoogleSQLModel(model *Model) (*googlesql.SimpleModel, error) {
	inputs, err := c.modelColumns(model.Name, model.Inputs)
	if err != nil {
		return nil, err
	}
	outputs, err := c.modelColumns(model.Name, model.Outputs)
	if err != nil {
		return nil, err
	}
	// The wrapper keeps the Go column handles alive through the registered
	// model. Let those handles own columns; native ownership would give both
	// the model and the Go handles responsibility for deleting them.
	gsModel, err := googlesql.NewSimpleModel(model.Name, inputs, outputs, false, -1)
	if err != nil {
		return nil, fmt.Errorf("model %s: %w", model.Name, err)
	}
	return gsModel, nil
}

func (c *GoogleSQLCatalog) modelColumns(modelName string, columns []*ModelColumn) ([]googlesql.Googlesql_ColumnNode, error) {
	// SimpleModel rejects duplicate column names in native code. Validate first
	// so malformed DDL or a caller-built Catalog returns a Go error instead.
	seen := make(map[string]bool, len(columns))
	for _, column := range columns {
		if column == nil || strings.TrimSpace(column.Name) == "" {
			return nil, fmt.Errorf("model %s has an empty column name", modelName)
		}
		key := strings.ToLower(column.Name)
		if seen[key] {
			return nil, fmt.Errorf("model %s has duplicate column %s", modelName, column.Name)
		}
		seen[key] = true
	}
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

type registeredGoogleSQLModel struct {
	model *Model
	node  googlesql.ModelNode
}
