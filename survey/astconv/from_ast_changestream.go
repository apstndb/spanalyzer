package astconv

import (
	"github.com/apstndb/spanner-emulator-survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func fromCreateChangeStream(s *Schema, ccs *ast.CreateChangeStream) error {
	csName := ccs.Name.Name

	cs := &infoschem.ChangeStream{
		ChangeStreamName: csName,
	}

	switch f := ccs.For.(type) {
	case *ast.ChangeStreamForAll:
		cs.All = true
	case *ast.ChangeStreamForTables:
		for _, ft := range f.Tables {
			cst := &infoschem.ChangeStreamTable{
				ChangeStreamName: csName,
				TableName:        ft.TableName.Name,
				AllColumns:       len(ft.Columns) == 0,
			}
			s.ChangeStreamTables = append(s.ChangeStreamTables, cst)

			for _, col := range ft.Columns {
				s.ChangeStreamColumns = append(s.ChangeStreamColumns, &infoschem.ChangeStreamColumn{
					ChangeStreamName: csName,
					TableName:        ft.TableName.Name,
					ColumnName:       col.Name,
				})
			}
		}
	}

	if ccs.Options != nil {
		for _, opt := range ccs.Options.Records {
			s.ChangeStreamOptions = append(s.ChangeStreamOptions, &infoschem.ChangeStreamOption{
				ChangeStreamName: csName,
				OptionName:       opt.Name.Name,
				OptionType:       inferOptionType(opt.Value),
				OptionValue:      opt.Value.SQL(),
			})
		}
	}

	s.ChangeStreams = append(s.ChangeStreams, cs)
	return nil
}
