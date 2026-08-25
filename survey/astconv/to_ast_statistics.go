package astconv

import (
	"fmt"

	"github.com/cloudspannerecosystem/memefish/ast"
)

func (s *Schema) toStatisticsDDL() ([]ast.DDL, error) {
	var ddls []ast.DDL
	for _, stat := range s.SpannerStatistics {
		if stat.SchemaName != "" {
			return nil, fmt.Errorf("unsupported named-schema statistics package %q", tableDisplayName(stat.SchemaName, stat.PackageName))
		}
		ddls = append(ddls, &ast.AlterStatistics{
			Name: ident(stat.PackageName),
			Options: mkOptions(
				optionsDef("allow_gc", boolval(stat.AllowGC)),
			),
		})
	}
	return ddls, nil
}
