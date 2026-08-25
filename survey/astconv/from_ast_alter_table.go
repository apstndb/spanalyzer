package astconv

import (
	"fmt"

	"github.com/cloudspannerecosystem/memefish/ast"
)

func fromAlterTable(s *Schema, alter *ast.AlterTable) error {
	tableSchema, tableName, err := schemaObjectName("altered table", alter.Name)
	if err != nil {
		return err
	}
	if !schemaHasTable(s, tableSchema, tableName) {
		return fmt.Errorf("ALTER TABLE target %s has not been created", tableDisplayName(tableSchema, tableName))
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

func schemaHasTable(s *Schema, tableSchema, tableName string) bool {
	for _, table := range s.Tables {
		if table.TableSchema == tableSchema && table.TableName == tableName {
			return true
		}
	}
	return false
}
