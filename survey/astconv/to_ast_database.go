package astconv

import (
	"fmt"
	"strings"

	"github.com/cloudspannerecosystem/memefish/ast"
)

// nonCanonicalDatabaseOptions are DATABASE_OPTIONS keys that GetDatabaseDdl is
// observed not to emit:
//   - database_dialect: read-only dialect marker; never DDL-settable
//   - enable_key_visualizer: observed 2026-09-09 — a managed database with
//     DATABASE_OPTIONS.enable_key_visualizer = TRUE returned an ALTER DATABASE
//     statement containing only default_sequence_kind
var nonCanonicalDatabaseOptions = map[string]struct{}{
	"database_dialect":      {},
	"enable_key_visualizer": {},
}

func (s *Schema) toDatabaseDDL() ([]ast.DDL, error) {
	var defs []*ast.OptionsDef
	for _, opt := range s.DatabaseOptions {
		if isNonCanonicalDatabaseOption(opt.OptionName) {
			continue
		}
		defs = append(defs, optionsDef(opt.OptionName, parseOptionValue(opt.OptionType, opt.OptionValue)))
	}

	if len(defs) == 0 {
		return nil, nil
	}

	dbName := s.DatabaseName
	if dbName == "" {
		for _, sch := range s.Schemata {
			if sch.CatalogName != "" {
				dbName = sch.CatalogName
				break
			}
		}
	}
	if dbName == "" {
		return nil, fmt.Errorf("ALTER DATABASE options are set but database name is unknown")
	}

	return []ast.DDL{
		&ast.AlterDatabase{
			Name:    ident(dbName),
			Options: mkOptions(defs...),
		},
	}, nil
}

func isNonCanonicalDatabaseOption(name string) bool {
	_, ok := nonCanonicalDatabaseOptions[strings.ToLower(name)]
	return ok
}
