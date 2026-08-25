package astconv

import (
	"github.com/cloudspannerecosystem/memefish/ast"
)

// systemSchemas are schemas managed by Spanner itself and should not appear in DDL output.
var systemSchemas = map[string]bool{
	"INFORMATION_SCHEMA": true,
	"SPANNER_SYS":        true,
}

func (s *Schema) toSchemasDDL() ([]ast.DDL, error) {
	var ddls []ast.DDL
	for _, sch := range s.Schemata {
		// Skip default schema (empty name) and system schemas
		if sch.SchemaName == "" || systemSchemas[sch.SchemaName] {
			continue
		}
		ddls = append(ddls, &ast.CreateSchema{
			Name: ident(sch.SchemaName),
		})
	}
	return ddls, nil
}
