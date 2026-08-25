package astconv

import (
	"fmt"

	"github.com/apstndb/spanner-emulator-survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
	"github.com/cloudspannerecosystem/memefish/token"
)

func (s *Schema) toChangeStreamsDDL() ([]ast.DDL, error) {
	// CreateChangeStream and ChangeStreamForTable use ast.Ident, so either a
	// stream or watched table schema would be silently discarded without this
	// validation.
	for _, cs := range s.ChangeStreams {
		if cs.ChangeStreamSchema != "" {
			return nil, fmt.Errorf("unsupported named-schema change stream %q", tableDisplayName(cs.ChangeStreamSchema, cs.ChangeStreamName))
		}
	}
	for _, table := range s.ChangeStreamTables {
		if table.ChangeStreamSchema != "" {
			return nil, fmt.Errorf("unsupported named-schema change stream %q", tableDisplayName(table.ChangeStreamSchema, table.ChangeStreamName))
		}
		if table.TableSchema != "" {
			return nil, fmt.Errorf("unsupported named-schema watched table %q for change stream %q", tableDisplayName(table.TableSchema, table.TableName), table.ChangeStreamName)
		}
	}
	for _, col := range s.ChangeStreamColumns {
		if col.ChangeStreamSchema != "" {
			return nil, fmt.Errorf("unsupported named-schema change stream %q", tableDisplayName(col.ChangeStreamSchema, col.ChangeStreamName))
		}
		if col.TableSchema != "" {
			return nil, fmt.Errorf("unsupported named-schema watched table %q for change stream %q", tableDisplayName(col.TableSchema, col.TableName), col.ChangeStreamName)
		}
	}
	for _, opt := range s.ChangeStreamOptions {
		if opt.ChangeStreamSchema != "" {
			return nil, fmt.Errorf("unsupported named-schema change stream %q", tableDisplayName(opt.ChangeStreamSchema, opt.ChangeStreamName))
		}
	}

	tablesByCS := groupBy(s.ChangeStreamTables, func(t *infoschem.ChangeStreamTable) string {
		return t.ChangeStreamSchema + "." + t.ChangeStreamName
	})
	colsByCSTable := groupBy(s.ChangeStreamColumns, func(c *infoschem.ChangeStreamColumn) string {
		return c.ChangeStreamSchema + "." + c.ChangeStreamName + "." + c.TableName
	})
	optsByCS := groupBy(s.ChangeStreamOptions, func(o *infoschem.ChangeStreamOption) string {
		return o.ChangeStreamSchema + "." + o.ChangeStreamName
	})

	var ddls []ast.DDL
	for _, cs := range s.ChangeStreams {
		key := cs.ChangeStreamSchema + "." + cs.ChangeStreamName

		ccs := &ast.CreateChangeStream{
			Name: ident(cs.ChangeStreamName),
		}

		if cs.All {
			ccs.For = &ast.ChangeStreamForAll{}
		} else if tables := tablesByCS[key]; len(tables) > 0 {
			var forTables []*ast.ChangeStreamForTable
			for _, t := range tables {
				ft := &ast.ChangeStreamForTable{
					TableName: ident(t.TableName),
				}
				if t.AllColumns {
					// No column list → suppress parentheses
					ft.Rparen = token.InvalidPos
				} else {
					colKey := cs.ChangeStreamSchema + "." + cs.ChangeStreamName + "." + t.TableName
					for _, c := range colsByCSTable[colKey] {
						ft.Columns = append(ft.Columns, ident(c.ColumnName))
					}
				}
				forTables = append(forTables, ft)
			}
			ccs.For = &ast.ChangeStreamForTables{
				Tables: forTables,
			}
		}

		if opts := optsByCS[key]; len(opts) > 0 {
			var defs []*ast.OptionsDef
			for _, o := range opts {
				defs = append(defs, optionsDef(o.OptionName, parseOptionValue(o.OptionType, o.OptionValue)))
			}
			ccs.Options = mkOptions(defs...)
		}

		ddls = append(ddls, ccs)
	}
	return ddls, nil
}
