package astconv

import (
	"github.com/apstndb/spanner-emulator-survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
	"github.com/cloudspannerecosystem/memefish/token"
)

func fromCreateModel(s *Schema, cm *ast.CreateModel) error {
	modelName := cm.Name.Name

	model := &infoschem.Model{
		ModelName: modelName,
		IsRemote:  cm.Remote != token.InvalidPos,
	}

	if cm.InputOutput != nil {
		for i, col := range cm.InputOutput.InputColumns {
			mc := &infoschem.ModelColumn{
				ModelName:       modelName,
				ColumnKind:      "INPUT",
				ColumnName:      col.Name.Name,
				OrdinalPosition: int64(i + 1),
			}
			if col.DataType != nil {
				mc.DataType = col.DataType.SQL()
			}
			s.ModelColumns = append(s.ModelColumns, mc)

			if col.Options != nil {
				for _, opt := range col.Options.Records {
					s.ModelColumnOptions = append(s.ModelColumnOptions, &infoschem.ModelColumnOption{
						ModelName:   modelName,
						ColumnKind:  "INPUT",
						ColumnName:  col.Name.Name,
						OptionName:  opt.Name.Name,
						OptionType:  inferOptionType(opt.Value),
						OptionValue: opt.Value.SQL(),
					})
				}
			}
		}
		for i, col := range cm.InputOutput.OutputColumns {
			mc := &infoschem.ModelColumn{
				ModelName:       modelName,
				ColumnKind:      "OUTPUT",
				ColumnName:      col.Name.Name,
				OrdinalPosition: int64(i + 1),
			}
			if col.DataType != nil {
				mc.DataType = col.DataType.SQL()
			}
			s.ModelColumns = append(s.ModelColumns, mc)

			if col.Options != nil {
				for _, opt := range col.Options.Records {
					s.ModelColumnOptions = append(s.ModelColumnOptions, &infoschem.ModelColumnOption{
						ModelName:   modelName,
						ColumnKind:  "OUTPUT",
						ColumnName:  col.Name.Name,
						OptionName:  opt.Name.Name,
						OptionType:  inferOptionType(opt.Value),
						OptionValue: opt.Value.SQL(),
					})
				}
			}
		}
	}

	if cm.Options != nil {
		for _, opt := range cm.Options.Records {
			s.ModelOptions = append(s.ModelOptions, &infoschem.ModelOption{
				ModelName:   modelName,
				OptionName:  opt.Name.Name,
				OptionType:  inferOptionType(opt.Value),
				OptionValue: opt.Value.SQL(),
			})
		}
	}

	s.Models = append(s.Models, model)
	return nil
}
