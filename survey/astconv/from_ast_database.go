package astconv

import (
	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func fromAlterDatabase(s *Schema, ad *ast.AlterDatabase) error {
	if ad.Name != nil && ad.Name.Name != "" {
		s.DatabaseName = ad.Name.Name
	}
	if ad.Options == nil {
		return nil
	}

	for _, opt := range ad.Options.Records {
		s.DatabaseOptions = append(s.DatabaseOptions, &infoschem.DatabaseOption{
			OptionName:  opt.Name.Name,
			OptionType:  inferOptionType(opt.Value),
			OptionValue: opt.Value.SQL(),
		})
	}
	return nil
}

func fromAlterStatistics(s *Schema, as *ast.AlterStatistics) error {
	stat := &infoschem.SpannerStatistic{
		PackageName: as.Name.Name,
	}
	if as.Options != nil {
		for _, opt := range as.Options.Records {
			if opt.Name.Name == "allow_gc" {
				if bl, ok := opt.Value.(*ast.BoolLiteral); ok {
					stat.AllowGC = bl.Value
				}
			}
		}
	}
	s.SpannerStatistics = append(s.SpannerStatistics, stat)
	return nil
}
