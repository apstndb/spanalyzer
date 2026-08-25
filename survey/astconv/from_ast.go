// Package astconv provides bidirectional conversion between INFORMATION_SCHEMA and AST.
package astconv

import (
	"fmt"

	"github.com/cloudspannerecosystem/memefish/ast"
)

// FromDDLStatements converts DDL statements into a Schema.
// Fields that cannot be recovered from DDL (like TABLE_CATALOG, SPANNER_STATE)
// are set to sensible defaults.
func FromDDLStatements(ddls []ast.DDL) (*Schema, error) {
	s := &Schema{}
	for _, ddl := range ddls {
		if err := fromDDL(s, ddl); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func fromDDL(s *Schema, ddl ast.DDL) error {
	switch d := ddl.(type) {
	case *ast.CreateTable:
		return fromCreateTable(s, d)
	case *ast.AlterTable:
		return fromAlterTable(s, d)
	case *ast.CreateIndex:
		return fromCreateIndex(s, d)
	case *ast.CreateSearchIndex:
		return fromCreateSearchIndex(s, d)
	case *ast.CreateView:
		return fromCreateView(s, d)
	case *ast.CreateChangeStream:
		return fromCreateChangeStream(s, d)
	case *ast.CreateSequence:
		return fromCreateSequence(s, d)
	case *ast.CreateModel:
		return fromCreateModel(s, d)
	case *ast.CreatePropertyGraph:
		return fromCreatePropertyGraph(s, d)
	case *ast.CreatePlacement:
		return fromCreatePlacement(s, d)
	case *ast.CreateSchema:
		return fromCreateSchema(s, d)
	case *ast.CreateRole:
		return fromCreateRole(s, d)
	case *ast.Grant:
		return fromGrant(s, d)
	case *ast.Revoke:
		return fromRevoke(s, d)
	case *ast.CreateFunction:
		return fromCreateFunction(s, d)
	case *ast.CreateVectorIndex:
		return fromCreateVectorIndex(s, d)
	case *ast.CreateProtoBundle:
		return fromCreateProtoBundle(s, d)
	case *ast.CreateLocalityGroup:
		return fromCreateLocalityGroup(s, d)
	case *ast.AlterDatabase:
		return fromAlterDatabase(s, d)
	case *ast.AlterStatistics:
		return fromAlterStatistics(s, d)
	case *ast.CreateDatabase:
		// No-op: CREATE DATABASE doesn't contribute to INFORMATION_SCHEMA
		return nil
	default:
		return fmt.Errorf("unsupported DDL type: %T", ddl)
	}
}
