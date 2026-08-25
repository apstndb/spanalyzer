package astconv

import (
	"github.com/cloudspannerecosystem/memefish/ast"
)

func (s *Schema) toDatabaseDDL() ([]ast.DDL, error) {
	if len(s.DatabaseOptions) == 0 {
		return nil, nil
	}

	var defs []*ast.OptionsDef
	for _, opt := range s.DatabaseOptions {
		defs = append(defs, optionsDef(opt.OptionName, parseOptionValue(opt.OptionType, opt.OptionValue)))
	}

	if len(defs) == 0 {
		return nil, nil
	}

	// Find the database name from Schemata (CATALOG_NAME)
	dbName := ""
	for _, sch := range s.Schemata {
		if sch.CatalogName != "" {
			dbName = sch.CatalogName
			break
		}
	}

	// Skip if we can't determine the database name
	if dbName == "" {
		return nil, nil
	}

	return []ast.DDL{
		&ast.AlterDatabase{
			Name:    ident(dbName),
			Options: mkOptions(defs...),
		},
	}, nil
}
