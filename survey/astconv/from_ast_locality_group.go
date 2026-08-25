package astconv

import (
	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func fromCreateLocalityGroup(s *Schema, clg *ast.CreateLocalityGroup) error {
	name := clg.Name.Name
	s.LocalityGroups = append(s.LocalityGroups, &infoschem.LocalityGroup{
		LocalityGroupName: name,
	})

	if clg.Options != nil {
		for _, opt := range clg.Options.Records {
			value := opt.Value.SQL()
			s.LocalityGroupOptions = append(s.LocalityGroupOptions, &infoschem.LocalityGroupOption{
				LocalityGroupName: name,
				OptionName:        opt.Name.Name,
				OptionValue:       &value,
				OptionType:        inferOptionType(opt.Value),
			})
		}
	}

	return nil
}
