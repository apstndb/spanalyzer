package astconv

import (
	"fmt"

	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func fromCreateIndex(s *Schema, ci *ast.CreateIndex) error {
	indexSchema, indexName, err := schemaObjectName("index", ci.Name)
	if err != nil {
		return err
	}
	tableSchema, tableName, err := schemaObjectName("index table", ci.TableName)
	if err != nil {
		return err
	}
	if indexSchema != tableSchema {
		return fmt.Errorf(
			"index %q and table %q must use the same schema",
			ci.Name.SQL(),
			ci.TableName.SQL(),
		)
	}
	// Spanner accepts a qualified target, but memefish v0.8.1 models it as an
	// unqualified Ident. An unqualified target resolves in the default schema
	// on emulator v1.5.55, so retaining it here would change the DDL's meaning.
	// See the named-schema index boundary in UNSUPPORTED_DDL.md.
	if indexSchema != "" && ci.InterleaveIn != nil {
		return fmt.Errorf(
			"unsupported named-schema interleaved index %q: interleave target %q cannot retain its schema",
			ci.Name.SQL(),
			ci.InterleaveIn.TableName.SQL(),
		)
	}

	idx := &infoschem.Index{
		TableSchema:    tableSchema,
		TableName:      tableName,
		IndexName:      indexName,
		IndexType:      "INDEX",
		IsUnique:       ci.Unique,
		IsNullFiltered: ci.NullFiltered,
		IndexState:     strPtr("READ_WRITE"),
	}

	if ci.InterleaveIn != nil {
		idx.ParentTableName = ci.InterleaveIn.TableName.Name
	}

	if ci.Options != nil {
		for _, opt := range ci.Options.Records {
			s.IndexOptions = append(s.IndexOptions, &infoschem.IndexOption{
				TableSchema: tableSchema,
				TableName:   tableName,
				IndexName:   indexName,
				IndexType:   "INDEX",
				OptionName:  opt.Name.Name,
				OptionType:  inferOptionType(opt.Value),
				OptionValue: opt.Value.SQL(),
			})
		}
	}

	s.Indexes = append(s.Indexes, idx)

	// Key columns
	for i, key := range ci.Keys {
		ordering := "ASC"
		if key.Dir == ast.DirectionDesc {
			ordering = "DESC"
		}
		ordinal := int64(i + 1)
		s.IndexColumns = append(s.IndexColumns, &infoschem.IndexColumn{
			TableSchema:     tableSchema,
			TableName:       tableName,
			IndexName:       indexName,
			IndexType:       "INDEX",
			ColumnName:      key.Name.Name,
			OrdinalPosition: &ordinal,
			ColumnOrdering:  strPtr(ordering),
		})
	}

	// Storing columns
	if ci.Storing != nil {
		for _, col := range ci.Storing.Columns {
			s.IndexColumns = append(s.IndexColumns, &infoschem.IndexColumn{
				TableSchema: tableSchema,
				TableName:   tableName,
				IndexName:   indexName,
				IndexType:   "INDEX",
				ColumnName:  col.Name,
			})
		}
	}

	return nil
}

func fromCreateSearchIndex(s *Schema, csi *ast.CreateSearchIndex) error {
	indexSchema, indexName, err := schemaObjectName("search index", csi.Name)
	if err != nil {
		return err
	}
	tableSchema, tableName, err := schemaObjectName("search index table", csi.TableName)
	if err != nil {
		return err
	}
	if indexSchema != tableSchema {
		return fmt.Errorf(
			"search index %q and table %q must use the same schema",
			csi.Name.SQL(),
			csi.TableName.SQL(),
		)
	}
	// InterleaveIn has the same unqualified-Ident limitation as a regular
	// index. Fail closed until memefish can retain the target schema.
	if indexSchema != "" && csi.Interleave != nil {
		return fmt.Errorf(
			"unsupported named-schema interleaved search index %q: interleave target %q cannot retain its schema",
			csi.Name.SQL(),
			csi.Interleave.TableName.SQL(),
		)
	}

	idx := &infoschem.Index{
		TableSchema: tableSchema,
		TableName:   tableName,
		IndexName:   indexName,
		IndexType:   "SEARCH",
		IndexState:  strPtr("READ_WRITE"),
	}

	if csi.Interleave != nil {
		idx.ParentTableName = csi.Interleave.TableName.Name
	}

	if len(csi.PartitionColumns) > 0 {
		for _, p := range csi.PartitionColumns {
			idx.SearchPartitionBy = append(idx.SearchPartitionBy, p.Name)
		}
	}

	if csi.OrderBy != nil {
		for _, item := range csi.OrderBy.Items {
			idx.SearchOrderBy = append(idx.SearchOrderBy, item.SQL())
		}
	}

	if csi.Where != nil {
		idx.Filter = strPtr(csi.Where.Expr.SQL())
	}

	if csi.Options != nil {
		for _, opt := range csi.Options.Records {
			s.IndexOptions = append(s.IndexOptions, &infoschem.IndexOption{
				TableSchema: tableSchema,
				TableName:   tableName,
				IndexName:   indexName,
				IndexType:   "SEARCH",
				OptionName:  opt.Name.Name,
				OptionType:  inferOptionType(opt.Value),
				OptionValue: opt.Value.SQL(),
			})
		}
	}

	s.Indexes = append(s.Indexes, idx)

	// Token list part columns
	for i, tlp := range csi.TokenListPart {
		ordinal := int64(i + 1)
		s.IndexColumns = append(s.IndexColumns, &infoschem.IndexColumn{
			TableSchema:     tableSchema,
			TableName:       tableName,
			IndexName:       indexName,
			IndexType:       "SEARCH",
			ColumnName:      tlp.Name,
			OrdinalPosition: &ordinal,
			ColumnOrdering:  strPtr("ASC"),
		})
	}

	// Storing columns
	if csi.Storing != nil {
		for _, col := range csi.Storing.Columns {
			s.IndexColumns = append(s.IndexColumns, &infoschem.IndexColumn{
				TableSchema: tableSchema,
				TableName:   tableName,
				IndexName:   indexName,
				IndexType:   "SEARCH",
				ColumnName:  col.Name,
			})
		}
	}

	return nil
}
