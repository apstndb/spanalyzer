package astconv

import (
	"fmt"
	"sort"

	"github.com/apstndb/spanner-emulator-survey/infoschem"
	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func (s *Schema) toVectorIndexesDDL() ([]ast.DDL, error) {
	idxCols := groupBy(s.IndexColumns, func(ic *infoschem.IndexColumn) string {
		return qualifiedIndexKey(ic.TableSchema, ic.TableName, ic.IndexName)
	})
	idxOpts := groupBy(s.IndexOptions, func(io *infoschem.IndexOption) string {
		return qualifiedIndexKey(io.TableSchema, io.TableName, io.IndexName)
	})

	var ddls []ast.DDL
	for _, idx := range s.Indexes {
		if idx.IndexType != "VECTOR_INDEX" {
			continue
		}
		if idx.SpannerIsManaged {
			continue
		}
		if isSystemSchemaName(idx.TableSchema) {
			continue
		}
		if idx.TableSchema != "" {
			return nil, fmt.Errorf("unsupported named-schema vector index %q on table %q", idx.IndexName, tableDisplayName(idx.TableSchema, idx.TableName))
		}

		key := qualifiedIndexKey(idx.TableSchema, idx.TableName, idx.IndexName)
		columns := idxCols[key]

		var keyColumns []*infoschem.IndexColumn
		for _, column := range columns {
			if column.OrdinalPosition != nil {
				keyColumns = append(keyColumns, column)
			}
		}
		sort.Slice(keyColumns, func(i, j int) bool {
			return *keyColumns[i].OrdinalPosition < *keyColumns[j].OrdinalPosition
		})
		if len(keyColumns) != 1 {
			return nil, fmt.Errorf(
				"vector index %q has %d key columns; memefish v0.8.1 can represent exactly one",
				idx.IndexName,
				len(keyColumns),
			)
		}

		cvi := &ast.CreateVectorIndex{
			Name:       ident(idx.IndexName),
			TableName:  ident(idx.TableName),
			ColumnName: ident(keyColumns[0].ColumnName),
		}

		// Storing columns
		var storing []*ast.Ident
		for _, column := range columns {
			if column.OrdinalPosition == nil {
				storing = append(storing, ident(column.ColumnName))
			}
		}
		if len(storing) > 0 {
			cvi.Storing = &ast.Storing{Columns: storing}
		}

		if idx.Filter != nil && *idx.Filter != "" {
			expr, err := memefish.ParseExpr("", *idx.Filter)
			if err != nil {
				return nil, fmt.Errorf("vector index %s FILTER: %w", idx.IndexName, err)
			}
			cvi.Where = &ast.Where{Expr: expr}
		}

		// Options
		if opts := idxOpts[key]; len(opts) > 0 {
			var defs []*ast.OptionsDef
			for _, o := range opts {
				defs = append(defs, optionsDef(o.OptionName, parseOptionValue(o.OptionType, o.OptionValue)))
			}
			cvi.Options = mkOptions(defs...)
		}

		ddls = append(ddls, cvi)
	}
	return ddls, nil
}
