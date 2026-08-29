package astconv

import (
	"fmt"
	"sort"
	"strings"

	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func qualifiedIndexKey(schema, table, index string) string {
	return schema + "\x00" + table + "\x00" + index
}

func (s *Schema) toIndexesDDL() ([]ast.DDL, error) {
	idxCols := groupBy(s.IndexColumns, func(ic *infoschem.IndexColumn) string {
		return qualifiedIndexKey(ic.TableSchema, ic.TableName, ic.IndexName)
	})
	idxOpts := groupBy(s.IndexOptions, func(io *infoschem.IndexOption) string {
		return qualifiedIndexKey(io.TableSchema, io.TableName, io.IndexName)
	})

	var ddls []ast.DDL
	for _, idx := range s.Indexes {
		if idx.IndexName == "PRIMARY_KEY" {
			continue
		}
		if idx.SpannerIsManaged {
			continue
		}
		if idx.IndexType == "VECTOR_INDEX" {
			continue
		}
		if idx.IndexType != "INDEX" && idx.IndexType != "SEARCH" && idx.IndexType != "SEARCH_INDEX" {
			return nil, fmt.Errorf("unsupported index type %q for index %q", idx.IndexType, idx.IndexName)
		}
		// Skip system schema indexes, but never silently drop user DDL.
		if isSystemSchemaName(idx.TableSchema) {
			continue
		}
		// Spanner accepts a qualified index interleave target, but memefish
		// v0.8.1 cannot represent it. Emulator v1.5.55 resolves the only
		// representable, unqualified form in the default schema, so fail closed.
		// See the named-schema index boundary in UNSUPPORTED_DDL.md.
		isInterleaved := idx.ParentTableName != "" && idx.ParentTableName != idx.TableName
		if idx.TableSchema != "" && isInterleaved {
			return nil, fmt.Errorf(
				"unsupported named-schema interleaved index %q on table %q: interleave target %q cannot retain its schema",
				idx.IndexName,
				tableDisplayName(idx.TableSchema, idx.TableName),
				idx.ParentTableName,
			)
		}
		key := qualifiedIndexKey(idx.TableSchema, idx.TableName, idx.IndexName)
		cols := idxCols[key]
		opts := idxOpts[key]

		switch idx.IndexType {
		case "INDEX":
			ddl, err := buildCreateIndex(idx, cols, opts)
			if err != nil {
				return nil, err
			}
			ddls = append(ddls, ddl)
		case "SEARCH", "SEARCH_INDEX":
			ddl, err := buildCreateSearchIndex(idx, cols, opts)
			if err != nil {
				return nil, err
			}
			ddls = append(ddls, ddl)
		}
	}
	return ddls, nil
}

func buildCreateIndex(idx *infoschem.Index, cols []*infoschem.IndexColumn, opts []*infoschem.IndexOption) (*ast.CreateIndex, error) {
	var keys []*ast.IndexKey
	var storing []*ast.Ident

	sort.Slice(cols, func(i, j int) bool {
		oi, oj := int64(0), int64(0)
		if cols[i].OrdinalPosition != nil {
			oi = *cols[i].OrdinalPosition
		}
		if cols[j].OrdinalPosition != nil {
			oj = *cols[j].OrdinalPosition
		}
		return oi < oj
	})

	for _, c := range cols {
		if c.Expression != nil {
			return nil, fmt.Errorf(
				"unsupported expression index %q on table %q: index key %q has expression %q that the current memefish AST cannot represent",
				idx.IndexName,
				tableDisplayName(idx.TableSchema, idx.TableName),
				c.ColumnName,
				*c.Expression,
			)
		}
		if c.OrdinalPosition != nil && c.ColumnOrdering != nil {
			keys = append(keys, indexKey(c.ColumnName, dirFromString(*c.ColumnOrdering)))
		} else {
			storing = append(storing, ident(c.ColumnName))
		}
	}

	ci := &ast.CreateIndex{
		Name:         schemaObjectPath(idx.TableSchema, idx.IndexName),
		TableName:    schemaObjectPath(idx.TableSchema, idx.TableName),
		Keys:         keys,
		Unique:       idx.IsUnique,
		NullFiltered: idx.IsNullFiltered,
	}

	if len(storing) > 0 {
		ci.Storing = &ast.Storing{Columns: storing}
	}

	if idx.ParentTableName != "" && idx.ParentTableName != idx.TableName {
		ci.InterleaveIn = &ast.InterleaveIn{
			TableName: ident(idx.ParentTableName),
		}
	}

	if len(opts) > 0 {
		var defs []*ast.OptionsDef
		for _, o := range opts {
			defs = append(defs, optionsDef(o.OptionName, parseOptionValue(o.OptionType, o.OptionValue)))
		}
		ci.Options = mkOptions(defs...)
	}

	return ci, nil
}

func buildCreateSearchIndex(idx *infoschem.Index, cols []*infoschem.IndexColumn, opts []*infoschem.IndexOption) (*ast.CreateSearchIndex, error) {
	sort.Slice(cols, func(i, j int) bool {
		oi, oj := int64(0), int64(0)
		if cols[i].OrdinalPosition != nil {
			oi = *cols[i].OrdinalPosition
		}
		if cols[j].OrdinalPosition != nil {
			oj = *cols[j].OrdinalPosition
		}
		return oi < oj
	})

	var tokenListPart []*ast.Ident
	var storing []*ast.Ident
	for _, c := range cols {
		if c.OrdinalPosition != nil && c.ColumnOrdering != nil {
			tokenListPart = append(tokenListPart, ident(c.ColumnName))
		} else {
			storing = append(storing, ident(c.ColumnName))
		}
	}

	csi := &ast.CreateSearchIndex{
		Name:          schemaObjectPath(idx.TableSchema, idx.IndexName),
		TableName:     schemaObjectPath(idx.TableSchema, idx.TableName),
		TokenListPart: tokenListPart,
	}

	if len(storing) > 0 {
		csi.Storing = &ast.Storing{Columns: storing}
	}

	if len(idx.SearchPartitionBy) > 0 {
		for _, p := range idx.SearchPartitionBy {
			csi.PartitionColumns = append(csi.PartitionColumns, ident(p))
		}
	}

	if len(idx.SearchOrderBy) > 0 {
		orderBy, err := parseSearchIndexOrderBy(idx.SearchOrderBy)
		if err != nil {
			return nil, fmt.Errorf("search index %s ORDER BY: %w", idx.IndexName, err)
		}
		csi.OrderBy = orderBy
	}

	if idx.Filter != nil && *idx.Filter != "" {
		expr, err := memefish.ParseExpr("", *idx.Filter)
		if err != nil {
			return nil, fmt.Errorf("search index %s FILTER: %w", idx.IndexName, err)
		}
		csi.Where = &ast.Where{Expr: expr}
	}

	if idx.ParentTableName != "" && idx.ParentTableName != idx.TableName {
		csi.Interleave = &ast.InterleaveIn{
			TableName: ident(idx.ParentTableName),
		}
	}

	if len(opts) > 0 {
		var defs []*ast.OptionsDef
		for _, o := range opts {
			defs = append(defs, optionsDef(o.OptionName, parseOptionValue(o.OptionType, o.OptionValue)))
		}
		csi.Options = mkOptions(defs...)
	}

	return csi, nil
}

func parseSearchIndexOrderBy(items []string) (*ast.OrderBy, error) {
	stmt, err := memefish.ParseDDL("", fmt.Sprintf(
		"CREATE SEARCH INDEX _i ON _t (_c) ORDER BY %s",
		strings.Join(items, ", "),
	))
	if err != nil {
		return nil, err
	}
	csi, ok := stmt.(*ast.CreateSearchIndex)
	if !ok || csi.OrderBy == nil {
		return nil, fmt.Errorf("failed to parse search index ORDER BY")
	}
	return csi.OrderBy, nil
}
