package astconv

import (
	"github.com/apstndb/spanner-emulator-survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func fromCreateVectorIndex(s *Schema, cvi *ast.CreateVectorIndex) error {
	indexName := cvi.Name.Name
	tableName := cvi.TableName.Name

	idx := &infoschem.Index{
		TableName:  tableName,
		IndexName:  indexName,
		IndexType:  "VECTOR_INDEX",
		IndexState: strPtr("READ_WRITE"),
	}
	if cvi.Where != nil {
		idx.Filter = strPtr(cvi.Where.Expr.SQL())
	}

	s.Indexes = append(s.Indexes, idx)

	// Indexed column
	ordinal := int64(1)
	s.IndexColumns = append(s.IndexColumns, &infoschem.IndexColumn{
		TableName:       tableName,
		IndexName:       indexName,
		IndexType:       "VECTOR_INDEX",
		ColumnName:      cvi.ColumnName.Name,
		OrdinalPosition: &ordinal,
		ColumnOrdering:  strPtr("ASC"),
	})

	// Storing columns
	if cvi.Storing != nil {
		for _, col := range cvi.Storing.Columns {
			s.IndexColumns = append(s.IndexColumns, &infoschem.IndexColumn{
				TableName:  tableName,
				IndexName:  indexName,
				IndexType:  "VECTOR_INDEX",
				ColumnName: col.Name,
			})
		}
	}

	// Options
	if cvi.Options != nil {
		for _, opt := range cvi.Options.Records {
			s.IndexOptions = append(s.IndexOptions, &infoschem.IndexOption{
				TableName:   tableName,
				IndexName:   indexName,
				IndexType:   "VECTOR_INDEX",
				OptionName:  opt.Name.Name,
				OptionType:  inferOptionType(opt.Value),
				OptionValue: opt.Value.SQL(),
			})
		}
	}

	return nil
}
