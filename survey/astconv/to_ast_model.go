package astconv

import (
	"fmt"
	"sort"

	"github.com/apstndb/spanner-emulator-survey/infoschem"
	"github.com/apstndb/spanner-emulator-survey/spannertype"
	"github.com/cloudspannerecosystem/memefish/ast"
	"github.com/cloudspannerecosystem/memefish/token"
)

func (s *Schema) toModelsDDL() ([]ast.DDL, error) {
	// CreateModel uses ast.Ident, so its name cannot retain a schema qualifier.
	for _, model := range s.Models {
		if model.ModelSchema != "" {
			return nil, fmt.Errorf("unsupported named-schema model %q", tableDisplayName(model.ModelSchema, model.ModelName))
		}
	}
	for _, col := range s.ModelColumns {
		if col.ModelSchema != "" {
			return nil, fmt.Errorf("unsupported named-schema model %q", tableDisplayName(col.ModelSchema, col.ModelName))
		}
	}
	for _, opt := range s.ModelOptions {
		if opt.ModelSchema != "" {
			return nil, fmt.Errorf("unsupported named-schema model %q", tableDisplayName(opt.ModelSchema, opt.ModelName))
		}
	}
	for _, opt := range s.ModelColumnOptions {
		if opt.ModelSchema != "" {
			return nil, fmt.Errorf("unsupported named-schema model %q", tableDisplayName(opt.ModelSchema, opt.ModelName))
		}
	}

	colsByModel := groupBy(s.ModelColumns, func(c *infoschem.ModelColumn) string {
		return c.ModelSchema + "." + c.ModelName
	})
	optsByModel := groupBy(s.ModelOptions, func(o *infoschem.ModelOption) string {
		return o.ModelSchema + "." + o.ModelName
	})
	colOptsByModel := groupBy(s.ModelColumnOptions, func(o *infoschem.ModelColumnOption) string {
		return o.ModelSchema + "." + o.ModelName + "." + o.ColumnKind + "." + o.ColumnName
	})
	_ = colOptsByModel // reserved for future use

	var ddls []ast.DDL
	for _, m := range s.Models {
		key := m.ModelSchema + "." + m.ModelName

		cm := &ast.CreateModel{
			Name: ident(m.ModelName),
		}

		if m.IsRemote {
			cm.Remote = token.Pos(1) // non-zero means REMOTE
		}

		// Input/Output columns
		cols := colsByModel[key]
		if len(cols) > 0 {
			sort.Slice(cols, func(i, j int) bool {
				if cols[i].ColumnKind != cols[j].ColumnKind {
					return cols[i].ColumnKind < cols[j].ColumnKind
				}
				return cols[i].OrdinalPosition < cols[j].OrdinalPosition
			})

			var inputCols, outputCols []*ast.CreateModelColumn
			for _, c := range cols {
				dataType, err := spannertype.ParseSchemaType(c.DataType)
				if err != nil {
					return nil, fmt.Errorf("model %s column %s: parse data type %q: %w", m.ModelName, c.ColumnName, c.DataType, err)
				}
				mc := &ast.CreateModelColumn{
					Name:     ident(c.ColumnName),
					DataType: dataType,
				}
				if colOpts := colOptsByModel[key+"."+c.ColumnKind+"."+c.ColumnName]; len(colOpts) > 0 {
					var defs []*ast.OptionsDef
					for _, o := range colOpts {
						defs = append(defs, optionsDef(o.OptionName, parseOptionValue(o.OptionType, o.OptionValue)))
					}
					mc.Options = mkOptions(defs...)
				}
				switch c.ColumnKind {
				case "INPUT":
					inputCols = append(inputCols, mc)
				case "OUTPUT":
					outputCols = append(outputCols, mc)
				}
			}
			if len(inputCols) > 0 || len(outputCols) > 0 {
				cm.InputOutput = &ast.CreateModelInputOutput{
					InputColumns:  inputCols,
					OutputColumns: outputCols,
				}
			}
		}

		// Options
		if opts := optsByModel[key]; len(opts) > 0 {
			var defs []*ast.OptionsDef
			for _, o := range opts {
				defs = append(defs, optionsDef(o.OptionName, parseOptionValue(o.OptionType, o.OptionValue)))
			}
			cm.Options = mkOptions(defs...)
		}

		ddls = append(ddls, cm)
	}
	return ddls, nil
}
