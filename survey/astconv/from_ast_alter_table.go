package astconv

import (
	"fmt"

	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func fromAlterTable(s *Schema, alter *ast.AlterTable) error {
	tableSchema, tableName, err := schemaObjectName("altered table", alter.Name)
	if err != nil {
		return err
	}
	table := lookupSchemaTable(s, tableSchema, tableName)
	if table == nil {
		return fmt.Errorf("ALTER TABLE target %s has not been created", tableDisplayName(tableSchema, tableName))
	}
	if add, ok := alter.TableAlteration.(*ast.AddRowDeletionPolicy); ok {
		if table.TableType != "BASE TABLE" {
			return fmt.Errorf("ALTER TABLE target %s is not a base table", tableDisplayName(tableSchema, tableName))
		}
		if table.RowDeletionPolicyExpression != nil {
			return fmt.Errorf("ALTER TABLE target %s already has a row deletion policy", tableDisplayName(tableSchema, tableName))
		}
		expression, err := rowDeletionPolicyExpression(add.RowDeletionPolicy)
		if err != nil {
			return fmt.Errorf("ALTER TABLE target %s: %w", tableDisplayName(tableSchema, tableName), err)
		}
		table.RowDeletionPolicyExpression = new(expression)
		return nil
	}

	add, ok := alter.TableAlteration.(*ast.AddTableConstraint)
	if !ok {
		return fmt.Errorf("unsupported ALTER TABLE operation: %T", alter.TableAlteration)
	}
	if add.TableConstraint == nil {
		return fmt.Errorf("ALTER TABLE target %s has an empty ADD constraint", tableDisplayName(tableSchema, tableName))
	}
	if _, ok := add.TableConstraint.Constraint.(*ast.TablePrimaryKey); ok {
		return fmt.Errorf("unsupported ALTER TABLE ADD PRIMARY KEY on %s", tableDisplayName(tableSchema, tableName))
	}

	ordinal := 0
	for _, constraint := range s.TableConstraints {
		if constraint.TableSchema == tableSchema && constraint.TableName == tableName {
			ordinal++
		}
	}
	return fromTableConstraint(s, tableSchema, tableName, ordinal, add.TableConstraint)
}

func lookupSchemaTable(s *Schema, tableSchema, tableName string) *infoschem.Table {
	for _, table := range s.Tables {
		if table.TableSchema == tableSchema && table.TableName == tableName {
			return table
		}
	}
	return nil
}
