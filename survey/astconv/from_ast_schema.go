package astconv

import (
	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func fromCreateSchema(s *Schema, cs *ast.CreateSchema) error {
	s.Schemata = append(s.Schemata, &infoschem.Schema{
		SchemaName: cs.Name.Name,
	})
	return nil
}
