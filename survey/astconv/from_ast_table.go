package astconv

import (
	"fmt"
	"strings"

	"github.com/apstndb/spanner-emulator-survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
	"github.com/cloudspannerecosystem/memefish/token"
)

func fromCreateTable(s *Schema, ct *ast.CreateTable) error {
	if ct.Name == nil || len(ct.Name.Idents) == 0 {
		return fmt.Errorf("unsupported table with missing name")
	}
	tableSchema, tableName, err := schemaObjectName("table", ct.Name)
	if err != nil {
		return err
	}

	t := &infoschem.Table{
		TableSchema:  tableSchema,
		TableName:    tableName,
		TableType:    "BASE TABLE",
		SpannerState: strPtr("COMMITTED"),
	}

	// Parent table (interleave)
	if ct.Cluster != nil {
		if ct.Cluster.TableName == nil || len(ct.Cluster.TableName.Idents) == 0 {
			return fmt.Errorf("table %s has an interleave parent with no name", tableName)
		}
		parentSchema, parentName, err := schemaObjectName("interleave parent", ct.Cluster.TableName)
		if err != nil {
			return err
		}
		if parentSchema != tableSchema {
			return fmt.Errorf(
				"interleave parent %q must use the same schema as table %s",
				ct.Cluster.TableName.SQL(),
				tableDisplayName(tableSchema, tableName),
			)
		}
		t.ParentTableName = strPtr(parentName)
		switch ct.Cluster.OnDelete {
		case ast.OnDeleteCascade:
			t.OnDeleteAction = strPtr("CASCADE")
		case ast.OnDeleteNoAction:
			t.OnDeleteAction = strPtr("NO ACTION")
		}
		if ct.Cluster.Enforced {
			t.InterleaveType = strPtr("IN PARENT")
		} else {
			t.InterleaveType = strPtr("IN")
		}
	}

	// Row deletion policy
	if ct.RowDeletionPolicy != nil {
		rdp := ct.RowDeletionPolicy.RowDeletionPolicy
		expr := fmt.Sprintf("OLDER_THAN(%s, INTERVAL %s DAY)", rdp.ColumnName.SQL(), rdp.NumDays.SQL())
		t.RowDeletionPolicyExpression = strPtr(expr)
	}

	s.Tables = append(s.Tables, t)

	// Columns
	for i, colDef := range ct.Columns {
		col := fromColumnDef(tableSchema, tableName, colDef, int64(i+1))
		s.Columns = append(s.Columns, col)
		if colDef.PlacementKey != nil {
			s.PlacementKeyColumns = append(s.PlacementKeyColumns, &infoschem.PlacementKeyColumn{
				TableSchema: tableSchema,
				TableName:   tableName,
				ColumnName:  colDef.Name.Name,
			})
		}

		// Column options
		if colDef.Options != nil {
			for _, opt := range colDef.Options.Records {
				s.ColumnOptions = append(s.ColumnOptions, &infoschem.ColumnOption{
					TableSchema: tableSchema,
					TableName:   tableName,
					ColumnName:  colDef.Name.Name,
					OptionName:  opt.Name.Name,
					OptionType:  inferOptionType(opt.Value),
					OptionValue: opt.Value.SQL(),
				})
			}
		}
	}

	// Determine whether a primary key was declared, and its columns. An omitted
	// primary-key clause is semantically different from PRIMARY KEY (): Spanner
	// expands the former to a hidden generated rowid, while the latter remains a
	// zero-column key and permits only one row.
	primaryKeyDeclared := !ct.PrimaryKeyRparen.Invalid()
	pkCols := ct.PrimaryKeys
	for _, tc := range ct.TableConstraints {
		if tpk, ok := tc.Constraint.(*ast.TablePrimaryKey); ok {
			primaryKeyDeclared = true
			pkCols = tpk.Columns
			break
		}
	}

	if primaryKeyDeclared && len(pkCols) == 0 {
		// A live zero-column key has a TABLE_CONSTRAINTS row but no INDEXES,
		// INDEX_COLUMNS, or KEY_COLUMN_USAGE rows. Preserve that shape so the
		// reverse conversion can emit the representable table-constraint form.
		s.TableConstraints = append(s.TableConstraints, &infoschem.TableConstraint{
			ConstraintSchema:  tableSchema,
			ConstraintName:    "PK_" + tableName,
			TableSchema:       tableSchema,
			TableName:         tableName,
			ConstraintType:    "PRIMARY KEY",
			IsDeferrable:      "NO",
			InitiallyDeferred: "NO",
			Enforced:          "YES",
		})
	}

	if len(pkCols) > 0 {
		// Add the PRIMARY_KEY index entry only when the key has columns. Both
		// managed Spanner and Omni omit INDEXES for a zero-column primary key.
		s.Indexes = append(s.Indexes, &infoschem.Index{
			TableSchema: tableSchema,
			TableName:   tableName,
			IndexName:   "PRIMARY_KEY",
			IndexType:   "PRIMARY_KEY",
			IsUnique:    true,
			IndexState:  strPtr("READ_WRITE"),
		})
	}

	for i, pk := range pkCols {
		ordering := "ASC"
		if pk.Dir == ast.DirectionDesc {
			ordering = "DESC"
		}
		ordinal := int64(i + 1)
		s.IndexColumns = append(s.IndexColumns, &infoschem.IndexColumn{
			TableSchema:     tableSchema,
			TableName:       tableName,
			IndexName:       "PRIMARY_KEY",
			IndexType:       "PRIMARY_KEY",
			ColumnName:      pk.Name.Name,
			OrdinalPosition: &ordinal,
			ColumnOrdering:  strPtr(ordering),
		})
		s.KeyColumnUsage = append(s.KeyColumnUsage, &infoschem.KeyColumnUsage{
			ConstraintSchema: tableSchema,
			ConstraintName:   "PK_" + tableName,
			TableSchema:      tableSchema,
			TableName:        tableName,
			ColumnName:       pk.Name.Name,
			OrdinalPosition:  ordinal,
		})
	}

	// Table constraints
	for constraintOrdinal, tc := range ct.TableConstraints {
		if err := fromTableConstraint(s, tableSchema, tableName, constraintOrdinal, tc); err != nil {
			return err
		}
	}

	// Synonyms
	for _, syn := range ct.Synonyms {
		s.TableSynonyms = append(s.TableSynonyms, &infoschem.TableSynonym{
			SynonymSchema: tableSchema,
			SynonymName:   syn.Name.Name,
			TableSchema:   tableSchema,
			TableName:     tableName,
		})
	}

	// Table-level options (e.g. OPTIONS(locality_group='...'))
	if ct.Options != nil {
		for _, opt := range ct.Options.Records {
			s.TableOptions = append(s.TableOptions, &infoschem.TableOption{
				TableSchema: tableSchema,
				TableName:   tableName,
				OptionName:  opt.Name.Name,
				OptionType:  inferOptionType(opt.Value),
				OptionValue: opt.Value.SQL(),
			})
		}
	}

	return nil
}

func fromTableConstraint(
	s *Schema,
	tableSchema string,
	tableName string,
	constraintOrdinal int,
	tc *ast.TableConstraint,
) error {
	if tc == nil || tc.Constraint == nil {
		return fmt.Errorf("table %s has an empty constraint", tableDisplayName(tableSchema, tableName))
	}
	constraintName := ""
	if tc.Name != nil {
		constraintName = tc.Name.Name
	} else {
		// INFORMATION_SCHEMA has no table identity on CHECK_CONSTRAINTS or
		// REFERENTIAL_CONSTRAINTS. Keep a collision-free internal key so
		// multiple anonymous constraints can be associated on reconstruction;
		// to_ast_table recognizes this prefix and omits the name again.
		constraintName = anonymousConstraintName(tableName, constraintOrdinal)
	}

	switch c := tc.Constraint.(type) {
	case *ast.TablePrimaryKey:
		// CREATE TABLE handles primary keys before this helper. ALTER TABLE cannot
		// add a primary key in the supported managed canonical surface.
		return nil
	case *ast.Check:
		s.TableConstraints = append(s.TableConstraints, &infoschem.TableConstraint{
			ConstraintSchema:  tableSchema,
			ConstraintName:    constraintName,
			TableSchema:       tableSchema,
			TableName:         tableName,
			ConstraintType:    "CHECK",
			IsDeferrable:      "NO",
			InitiallyDeferred: "NO",
			Enforced:          "YES",
		})
		s.CheckConstraints = append(s.CheckConstraints, &infoschem.CheckConstraint{
			ConstraintSchema: tableSchema,
			ConstraintName:   constraintName,
			CheckClause:      c.Expr.SQL(),
			SpannerState:     strPtr("COMMITTED"),
		})
		return nil
	case *ast.ForeignKey:
		if c.ReferenceTable == nil || len(c.ReferenceTable.Idents) == 0 {
			return fmt.Errorf("foreign key %q on table %s has no referenced table", constraintDisplayName(constraintName), tableName)
		}
		refSchema, refTable, err := schemaObjectName("foreign-key reference", c.ReferenceTable)
		if err != nil {
			return err
		}

		enforced := "YES"
		if c.Enforcement == ast.NotEnforced {
			enforced = "NO"
		}
		s.TableConstraints = append(s.TableConstraints, &infoschem.TableConstraint{
			ConstraintSchema:  tableSchema,
			ConstraintName:    constraintName,
			TableSchema:       tableSchema,
			TableName:         tableName,
			ConstraintType:    "FOREIGN KEY",
			IsDeferrable:      "NO",
			InitiallyDeferred: "NO",
			Enforced:          enforced,
		})

		deleteRule := "NO ACTION"
		if c.OnDelete == ast.OnDeleteCascade {
			deleteRule = "CASCADE"
		}
		// The AST identifies the referenced columns directly but does not expose
		// the backing unique-constraint name. Use a collision-free internal
		// constraint and materialize its ordered KCU rows; guessing the parent
		// primary key would corrupt FKs backed by a unique secondary index.
		uniqueConstraintName := fmt.Sprintf(
			"%sREFERENCED_%s\x00%s_%d",
			anonymousConstraintPrefix,
			tableSchema,
			tableName,
			constraintOrdinal,
		)
		s.ReferentialConstraints = append(s.ReferentialConstraints, &infoschem.ReferentialConstraint{
			ConstraintSchema:       tableSchema,
			ConstraintName:         constraintName,
			UniqueConstraintSchema: refSchema,
			UniqueConstraintName:   uniqueConstraintName,
			MatchOption:            "SIMPLE",
			UpdateRule:             "NO ACTION",
			DeleteRule:             deleteRule,
			SpannerState:           strPtr("COMMITTED"),
		})

		for j, fkCol := range c.Columns {
			position := int64(j + 1)
			s.KeyColumnUsage = append(s.KeyColumnUsage, &infoschem.KeyColumnUsage{
				ConstraintSchema:           tableSchema,
				ConstraintName:             constraintName,
				TableSchema:                tableSchema,
				TableName:                  tableName,
				ColumnName:                 fkCol.Name,
				OrdinalPosition:            position,
				PositionInUniqueConstraint: &position,
			})
		}

		for j, refCol := range c.ReferenceColumns {
			position := int64(j + 1)
			s.ConstraintColumnUsage = append(s.ConstraintColumnUsage, &infoschem.ConstraintColumnUsage{
				TableSchema:      refSchema,
				TableName:        refTable,
				ColumnName:       refCol.Name,
				ConstraintSchema: tableSchema,
				ConstraintName:   constraintName,
			})
			s.KeyColumnUsage = append(s.KeyColumnUsage, &infoschem.KeyColumnUsage{
				ConstraintSchema: refSchema,
				ConstraintName:   uniqueConstraintName,
				TableSchema:      refSchema,
				TableName:        refTable,
				ColumnName:       refCol.Name,
				OrdinalPosition:  position,
			})
		}

		s.ConstraintTableUsage = append(
			s.ConstraintTableUsage,
			&infoschem.ConstraintTableUsage{
				TableSchema:      refSchema,
				TableName:        refTable,
				ConstraintSchema: tableSchema,
				ConstraintName:   constraintName,
			},
			&infoschem.ConstraintTableUsage{
				TableSchema:      refSchema,
				TableName:        refTable,
				ConstraintSchema: refSchema,
				ConstraintName:   uniqueConstraintName,
			},
		)
		return nil
	default:
		return fmt.Errorf("unsupported table constraint type: %T", tc.Constraint)
	}
}

func fromColumnDef(tableSchema, tableName string, cd *ast.ColumnDef, ordinal int64) *infoschem.Column {
	spannerType := cd.Type.SQL()
	dataType := simplifyDataType(spannerType)

	isNullable := "YES"
	if cd.NotNull {
		isNullable = "NO"
	}

	col := &infoschem.Column{
		TableSchema:     tableSchema,
		TableName:       tableName,
		ColumnName:      cd.Name.Name,
		OrdinalPosition: ordinal,
		DataType:        strPtr(dataType),
		IsNullable:      isNullable,
		SpannerType:     spannerType,
		IsGenerated:     "NEVER",
		SpannerState:    strPtr("COMMITTED"),
	}

	// Hidden
	if cd.Hidden != token.InvalidPos {
		col.IsHidden = true
	}

	// Default semantics
	switch ds := cd.DefaultSemantics.(type) {
	case *ast.ColumnDefaultExpr:
		col.ColumnDefault = strPtr(ds.Expr.SQL())
		if ds.OnUpdate != nil {
			col.OnUpdateExpression = strPtr(ds.OnUpdate.Expr.SQL())
		}
	case *ast.GeneratedColumnExpr:
		col.IsGenerated = "ALWAYS"
		expr := ds.Expr.SQL()
		col.GenerationExpression = strPtr(expr)
		if ds.Stored != token.InvalidPos {
			col.IsStored = strPtr("YES")
		} else {
			col.IsStored = strPtr("NO")
		}
	case *ast.IdentityColumn:
		col.IsIdentity = strPtr("YES")
		col.IdentityGeneration = strPtr("BY DEFAULT")
		for _, p := range ds.Params {
			switch param := p.(type) {
			case *ast.SkipRange:
				col.IdentitySkipRangeMin = strPtr(param.Min.Value)
				col.IdentitySkipRangeMax = strPtr(param.Max.Value)
			case *ast.StartCounterWith:
				col.IdentityStartWithCounter = strPtr(param.Counter.Value)
			case *ast.BitReversedPositive:
				col.IdentityKind = strPtr("BIT_REVERSED_POSITIVE")
			}
		}
	case *ast.AutoIncrement:
		col.ColumnDefault = strPtr("AUTO_INCREMENT")
	}

	return col
}

// simplifyDataType converts SPANNER_TYPE to simplified DATA_TYPE.
// e.g. "STRING(MAX)" -> "STRING", "ARRAY<INT64>" -> "ARRAY"
func simplifyDataType(spannerType string) string {
	s := strings.ToUpper(spannerType)
	if strings.HasPrefix(s, "ARRAY") {
		return "ARRAY"
	}
	if before, _, found := strings.Cut(s, "("); found {
		return before
	}
	return s
}

// inferOptionType infers the option type from an AST expression.
func inferOptionType(expr ast.Expr) string {
	switch expr.(type) {
	case *ast.BoolLiteral:
		return "BOOL"
	case *ast.IntLiteral:
		return "INT64"
	case *ast.FloatLiteral:
		return "FLOAT64"
	case *ast.NullLiteral:
		return "STRING"
	default:
		return "STRING"
	}
}
