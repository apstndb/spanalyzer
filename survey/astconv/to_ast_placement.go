package astconv

import (
	"github.com/apstndb/spanner-emulator-survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func (s *Schema) toPlacementsDDL() ([]ast.DDL, error) {
	optsByPlacement := groupBy(s.PlacementOptions, func(o *infoschem.PlacementOption) string {
		return o.PlacementName
	})

	var ddls []ast.DDL
	for _, p := range s.Placements {
		cp := &ast.CreatePlacement{
			Name: ident(p.PlacementName),
		}

		if opts := optsByPlacement[p.PlacementName]; len(opts) > 0 {
			var defs []*ast.OptionsDef
			for _, o := range opts {
				defs = append(defs, optionsDef(o.OptionName, parseOptionValue(o.OptionType, o.OptionValue)))
			}
			cp.Options = mkOptions(defs...)
		}

		ddls = append(ddls, cp)
	}
	return ddls, nil
}
