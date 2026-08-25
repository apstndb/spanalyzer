package astconv

import (
	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func fromCreatePlacement(s *Schema, cp *ast.CreatePlacement) error {
	s.Placements = append(s.Placements, &infoschem.Placement{
		PlacementName: cp.Name.Name,
	})

	if cp.Options != nil {
		for _, opt := range cp.Options.Records {
			s.PlacementOptions = append(s.PlacementOptions, &infoschem.PlacementOption{
				PlacementName: cp.Name.Name,
				OptionName:    opt.Name.Name,
				OptionType:    inferOptionType(opt.Value),
				OptionValue:   opt.Value.SQL(),
			})
		}
	}

	return nil
}
