package astconv

import (
	"fmt"
	"sort"
	"strings"

	"github.com/apstndb/spanner-emulator-survey/infoschem"
	"github.com/apstndb/spanner-emulator-survey/spannertype"
	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
	"github.com/cloudspannerecosystem/memefish/token"
)

// The NUL prefix cannot be produced by a valid SQL identifier, so a real
// user-supplied constraint name can never be mistaken for an internal key.
const anonymousConstraintPrefix = "\x00ASTCONV_ANONYMOUS_"

func qualifiedTableKey(schema, name string) string {
	return schema + "\x00" + name
}

func qualifiedConstraintKey(schema, name string) string {
	return schema + "\x00" + name
}

func tableDisplayName(schema, name string) string {
	if schema == "" {
		return name
	}
	return schema + "." + name
}

func isSystemSchemaName(schema string) bool {
	return systemSchemas[strings.ToUpper(schema)]
}

func isAnonymousConstraintName(name string) bool {
	return name == "" || strings.HasPrefix(name, anonymousConstraintPrefix)
}

func constraintASTName(name string) *ast.Ident {
	if isAnonymousConstraintName(name) {
		return nil
	}
	return ident(name)
}

func anonymousConstraintName(tableName string, ordinal int) string {
	return fmt.Sprintf("%s%s_%d", anonymousConstraintPrefix, tableName, ordinal)
}

func (s *Schema) toTablesDDL() ([]ast.DDL, error) {
	// A synonym is emitted as an unqualified ast.Ident inside CREATE TABLE. It
	// inherits the table schema, so reject inconsistent metadata rather than
	// silently moving it between schemas.
	for _, syn := range s.TableSynonyms {
		if isSystemSchemaName(syn.TableSchema) {
			continue
		}
		if syn.SynonymSchema != syn.TableSchema {
			return nil, fmt.Errorf(
				"table synonym %q must use the same schema as target %q",
				tableDisplayName(syn.SynonymSchema, syn.SynonymName),
				tableDisplayName(syn.TableSchema, syn.TableName),
			)
		}
	}

	// Build lookup maps
	colsByTable := groupBy(s.Columns, func(c *infoschem.Column) string {
		return qualifiedTableKey(c.TableSchema, c.TableName)
	})
	optsByCol := groupBy2(s.ColumnOptions, func(o *infoschem.ColumnOption) (string, string) {
		return qualifiedTableKey(o.TableSchema, o.TableName), o.ColumnName
	})
	constraintsByTable := groupBy(s.TableConstraints, func(c *infoschem.TableConstraint) string {
		return qualifiedTableKey(c.TableSchema, c.TableName)
	})
	checksByName := groupBy(s.CheckConstraints, func(c *infoschem.CheckConstraint) string {
		return qualifiedConstraintKey(c.ConstraintSchema, c.ConstraintName)
	})
	refsByName := groupBy(s.ReferentialConstraints, func(r *infoschem.ReferentialConstraint) string {
		return qualifiedConstraintKey(r.ConstraintSchema, r.ConstraintName)
	})
	kcuByConstraint := groupBy(s.KeyColumnUsage, func(k *infoschem.KeyColumnUsage) string {
		return qualifiedConstraintKey(k.ConstraintSchema, k.ConstraintName)
	})
	ccuByConstraint := groupBy(s.ConstraintColumnUsage, func(c *infoschem.ConstraintColumnUsage) string {
		return qualifiedConstraintKey(c.ConstraintSchema, c.ConstraintName)
	})
	synonymsByTable := groupBy(s.TableSynonyms, func(syn *infoschem.TableSynonym) string {
		return qualifiedTableKey(syn.TableSchema, syn.TableName)
	})
	tableOptsByTable := groupBy(s.TableOptions, func(o *infoschem.TableOption) string {
		return qualifiedTableKey(o.TableSchema, o.TableName)
	})
	placementKeysByTable := make(map[string]map[string]bool)
	knownTables := make(map[string]bool, len(s.Tables))
	for _, table := range s.Tables {
		knownTables[qualifiedTableKey(table.TableSchema, table.TableName)] = true
	}
	knownColumns := make(map[string]bool, len(s.Columns))
	for _, column := range s.Columns {
		tableKey := qualifiedTableKey(column.TableSchema, column.TableName)
		knownColumns[tableKey+"\x00"+column.ColumnName] = true
	}
	for _, placementKey := range s.PlacementKeyColumns {
		tableKey := qualifiedTableKey(placementKey.TableSchema, placementKey.TableName)
		if !knownTables[tableKey] {
			return nil, fmt.Errorf(
				"PLACEMENT KEY metadata references missing table %s",
				placementKey.TableName,
			)
		}
		if !knownColumns[tableKey+"\x00"+placementKey.ColumnName] {
			return nil, fmt.Errorf(
				"PLACEMENT KEY metadata references missing column %s.%s",
				placementKey.TableName,
				placementKey.ColumnName,
			)
		}
		if placementKeysByTable[tableKey] == nil {
			placementKeysByTable[tableKey] = make(map[string]bool)
		}
		placementKeysByTable[tableKey][placementKey.ColumnName] = true
	}

	// PK index columns for each table
	pkIndexCols := make(map[string][]*infoschem.IndexColumn) // "schema.table" -> pk cols
	for _, ic := range s.IndexColumns {
		if ic.IndexName == "PRIMARY_KEY" {
			key := qualifiedTableKey(ic.TableSchema, ic.TableName)
			pkIndexCols[key] = append(pkIndexCols[key], ic)
		}
	}

	// Topological sort: interleave parents and referenced tables first.
	tables, err := topSortTables(
		s.Tables,
		s.TableConstraints,
		s.ConstraintTableUsage,
	)
	if err != nil {
		return nil, err
	}

	var ddls []ast.DDL
	for _, t := range tables {
		// Skip non-BASE TABLE (views are handled separately)
		if t.TableType != "BASE TABLE" {
			continue
		}
		// Skip only schemas managed by Spanner.
		if isSystemSchemaName(t.TableSchema) {
			continue
		}

		ddl, err := s.buildCreateTable(t, colsByTable, optsByCol, constraintsByTable,
			checksByName, refsByName, kcuByConstraint, ccuByConstraint, pkIndexCols,
			synonymsByTable, tableOptsByTable, placementKeysByTable)
		if err != nil {
			return nil, fmt.Errorf("table %s: %w", tableDisplayName(t.TableSchema, t.TableName), err)
		}
		ddls = append(ddls, ddl)
	}
	return ddls, nil
}

func (s *Schema) buildCreateTable(
	t *infoschem.Table,
	colsByTable map[string][]*infoschem.Column,
	optsByCol map[string]map[string][]*infoschem.ColumnOption,
	constraintsByTable map[string][]*infoschem.TableConstraint,
	checksByName map[string][]*infoschem.CheckConstraint,
	refsByName map[string][]*infoschem.ReferentialConstraint,
	kcuByConstraint map[string][]*infoschem.KeyColumnUsage,
	ccuByConstraint map[string][]*infoschem.ConstraintColumnUsage,
	pkIndexCols map[string][]*infoschem.IndexColumn,
	synonymsByTable map[string][]*infoschem.TableSynonym,
	tableOptsByTable map[string][]*infoschem.TableOption,
	placementKeysByTable map[string]map[string]bool,
) (*ast.CreateTable, error) {
	tableKey := qualifiedTableKey(t.TableSchema, t.TableName)

	// Build columns
	cols := colsByTable[tableKey]
	sort.Slice(cols, func(i, j int) bool {
		return cols[i].OrdinalPosition < cols[j].OrdinalPosition
	})

	var columnDefs []*ast.ColumnDef
	for _, c := range cols {
		cd, err := buildColumnDef(c, optsByCol[tableKey], placementKeysByTable[tableKey][c.ColumnName])
		if err != nil {
			return nil, fmt.Errorf("column %s.%s: %w", t.TableName, c.ColumnName, err)
		}
		columnDefs = append(columnDefs, cd)
	}

	// Build table constraints (PRIMARY KEY, CHECK, FOREIGN KEY)
	var tableConstraints []*ast.TableConstraint
	primaryKeyDeclared := false
	anonymousCheckIndex := 0
	anonymousForeignKeyIndex := 0
	for _, tc := range constraintsByTable[tableKey] {
		switch tc.ConstraintType {
		case "PRIMARY KEY":
			// Non-empty primary keys are emitted from INDEX_COLUMNS below. A
			// zero-column key has no INDEX_COLUMNS rows, so retain the constraint
			// marker and emit the equivalent table-constraint syntax.
			primaryKeyDeclared = true
			continue
		case "CHECK":
			// Skip system-generated NOT NULL constraints (CK_IS_NOT_NULL_*)
			if strings.HasPrefix(tc.ConstraintName, "CK_IS_NOT_NULL_") {
				continue
			}
			checkRecords := checksByName[qualifiedConstraintKey(tc.ConstraintSchema, tc.ConstraintName)]
			if tc.ConstraintName == "" {
				checkRecords = checksByName[qualifiedConstraintKey(tc.ConstraintSchema, "")]
			}
			if len(checkRecords) == 0 {
				continue
			}
			checkIndex := 0
			if tc.ConstraintName == "" {
				checkIndex = anonymousCheckIndex
				anonymousCheckIndex++
			}
			if checkIndex >= len(checkRecords) {
				return nil, fmt.Errorf("missing metadata for anonymous CHECK constraint")
			}
			chk := checkRecords[checkIndex]
			expr, err := memefish.ParseExpr("", chk.CheckClause)
			if err != nil {
				return nil, fmt.Errorf(
					"check constraint %q on table %s: %w",
					constraintDisplayName(tc.ConstraintName),
					tableDisplayName(t.TableSchema, t.TableName),
					err,
				)
			}
			tableConstraints = append(tableConstraints, &ast.TableConstraint{
				Name: constraintASTName(tc.ConstraintName),
				Constraint: &ast.Check{
					Expr: expr,
				},
			})
		case "FOREIGN KEY":
			refRecords := refsByName[qualifiedConstraintKey(tc.ConstraintSchema, tc.ConstraintName)]
			if tc.ConstraintName == "" {
				refRecords = refsByName[qualifiedConstraintKey(tc.ConstraintSchema, "")]
			}
			if len(refRecords) == 0 {
				continue
			}
			refIndex := 0
			if tc.ConstraintName == "" {
				refIndex = anonymousForeignKeyIndex
				anonymousForeignKeyIndex++
			}
			if refIndex >= len(refRecords) {
				return nil, fmt.Errorf("missing metadata for anonymous FOREIGN KEY constraint")
			}
			ref := refRecords[refIndex]
			fkCols := append([]*infoschem.KeyColumnUsage(nil), kcuByConstraint[qualifiedConstraintKey(tc.ConstraintSchema, tc.ConstraintName)]...)
			sort.Slice(fkCols, func(i, j int) bool {
				return fkCols[i].OrdinalPosition < fkCols[j].OrdinalPosition
			})
			refSchema, refTableName, err := referencedTableForConstraint(tc, s.ConstraintTableUsage)
			if err != nil {
				return nil, err
			}
			if refTableName == "" {
				return nil, fmt.Errorf(
					"foreign key %q has no referenced table metadata",
					constraintDisplayName(tc.ConstraintName),
				)
			}
			refColumnNames, err := orderedReferencedColumnNames(
				fkCols,
				ccuByConstraint[qualifiedConstraintKey(tc.ConstraintSchema, tc.ConstraintName)],
				kcuByConstraint,
				ref,
				refSchema,
				refTableName,
			)
			if err != nil {
				return nil, fmt.Errorf("foreign key %q: %w", constraintDisplayName(tc.ConstraintName), err)
			}

			var fkColIdents []*ast.Ident
			for _, fc := range fkCols {
				fkColIdents = append(fkColIdents, ident(fc.ColumnName))
			}
			var refColIdents []*ast.Ident
			for _, refColumnName := range refColumnNames {
				refColIdents = append(refColIdents, ident(refColumnName))
			}

			fk := &ast.ForeignKey{
				Columns:          fkColIdents,
				ReferenceColumns: refColIdents,
			}
			switch ref.DeleteRule {
			case "CASCADE":
				fk.OnDelete = ast.OnDeleteCascade
			case "NO ACTION":
				fk.OnDelete = ast.OnDeleteNoAction
			}
			// Enforcement
			if tc.Enforced == "NO" {
				fk.Enforcement = ast.NotEnforced
			}

			refPath := []string{refTableName}
			if refSchema != "" {
				refPath = []string{refSchema, refTableName}
			}
			fk.ReferenceTable = path(refPath...)

			tableConstraints = append(tableConstraints, &ast.TableConstraint{
				Name:       constraintASTName(tc.ConstraintName),
				Constraint: fk,
			})
		}
	}

	// Build primary keys from INDEX_COLUMNS
	pkCols := pkIndexCols[tableKey]
	sort.Slice(pkCols, func(i, j int) bool {
		op1, op2 := int64(0), int64(0)
		if pkCols[i].OrdinalPosition != nil {
			op1 = *pkCols[i].OrdinalPosition
		}
		if pkCols[j].OrdinalPosition != nil {
			op2 = *pkCols[j].OrdinalPosition
		}
		return op1 < op2
	})
	var primaryKeys []*ast.IndexKey
	for _, pk := range pkCols {
		dir := dirFromString(ptrStr(pk.ColumnOrdering))
		primaryKeys = append(primaryKeys, indexKey(pk.ColumnName, dir))
	}
	if primaryKeyDeclared && len(primaryKeys) == 0 {
		// memefish v0.8.1 cannot serialize the trailing empty-key form because
		// CreateTable.SQL checks len(PrimaryKeys), but TablePrimaryKey.SQL can
		// emit the service-equivalent table-constraint form. Managed Spanner and
		// Omni 2026.r2.1-beta both accept it and canonicalize it to the trailing
		// PRIMARY KEY() form.
		tableConstraints = append(tableConstraints, &ast.TableConstraint{
			Constraint: &ast.TablePrimaryKey{},
		})
	}

	ct := &ast.CreateTable{
		Name:             schemaObjectPath(t.TableSchema, t.TableName),
		Columns:          columnDefs,
		TableConstraints: tableConstraints,
		PrimaryKeys:      primaryKeys,
		PrimaryKeyRparen: token.InvalidPos,
	}

	// Interleave (cluster)
	if t.ParentTableName != nil && *t.ParentTableName != "" {
		ct.Cluster = &ast.Cluster{
			TableName: schemaObjectPath(t.TableSchema, *t.ParentTableName),
			Enforced:  t.InterleaveType == nil || *t.InterleaveType == "IN PARENT",
		}
		if t.OnDeleteAction != nil {
			switch *t.OnDeleteAction {
			case "CASCADE":
				ct.Cluster.OnDelete = ast.OnDeleteCascade
			case "NO ACTION":
				ct.Cluster.OnDelete = ast.OnDeleteNoAction
			}
		}
	}

	// Row deletion policy
	if t.RowDeletionPolicyExpression != nil && *t.RowDeletionPolicyExpression != "" {
		rdp, err := parseRowDeletionPolicy(*t.RowDeletionPolicyExpression)
		if err != nil {
			return nil, fmt.Errorf(
				"row deletion policy expression %q: %w",
				*t.RowDeletionPolicyExpression,
				err,
			)
		}
		ct.RowDeletionPolicy = &ast.CreateRowDeletionPolicy{
			RowDeletionPolicy: rdp,
		}
	}

	// Synonyms
	for _, syn := range synonymsByTable[tableKey] {
		ct.Synonyms = append(ct.Synonyms, &ast.Synonym{
			Name: ident(syn.SynonymName),
		})
	}

	// Table-level options (e.g. OPTIONS(locality_group='...'))
	if opts := tableOptsByTable[tableKey]; len(opts) > 0 {
		var defs []*ast.OptionsDef
		for _, opt := range opts {
			defs = append(defs, optionsDef(opt.OptionName, parseOptionValue(opt.OptionType, opt.OptionValue)))
		}
		ct.Options = mkOptions(defs...)
	}

	return ct, nil
}

func buildColumnDef(c *infoschem.Column, colOpts map[string][]*infoschem.ColumnOption, placementKey bool) (*ast.ColumnDef, error) {
	schemaType, err := spannertype.ParseSchemaType(c.SpannerType)
	if err != nil {
		return nil, err
	}

	cd := &ast.ColumnDef{
		Name:    ident(c.ColumnName),
		Type:    schemaType,
		NotNull: c.IsNullable == "NO",
		Hidden:  token.InvalidPos, // default: not hidden
	}

	// Hidden
	if c.IsHidden {
		cd.Hidden = token.Pos(1)
	}
	if placementKey {
		cd.PlacementKey = &ast.PlacementKey{}
	}

	// Generated column
	if c.IsGenerated == "ALWAYS" && c.GenerationExpression != nil {
		expr, err := memefish.ParseExpr("", *c.GenerationExpression)
		if err != nil {
			return nil, fmt.Errorf("generation expression for %s: %w", c.ColumnName, err)
		}
		gen := &ast.GeneratedColumnExpr{
			Expr:   expr,
			Stored: token.InvalidPos,
		}
		if c.IsStored != nil && *c.IsStored == "YES" {
			gen.Stored = token.Pos(1)
		}
		cd.DefaultSemantics = gen
	}

	// DEFAULT / ON UPDATE expression
	if c.IsGenerated != "ALWAYS" {
		var defaultExpr *ast.ColumnDefaultExpr
		if c.ColumnDefault != nil && *c.ColumnDefault != "" {
			expr, err := memefish.ParseExpr("", *c.ColumnDefault)
			if err != nil {
				return nil, fmt.Errorf("default expression for %s: %w", c.ColumnName, err)
			}
			defaultExpr = &ast.ColumnDefaultExpr{
				Expr: expr,
			}
		}
		if c.OnUpdateExpression != nil && *c.OnUpdateExpression != "" {
			if defaultExpr == nil {
				return nil, fmt.Errorf("ON UPDATE expression for %s requires DEFAULT expression", c.ColumnName)
			}
			expr, err := memefish.ParseExpr("", *c.OnUpdateExpression)
			if err != nil {
				return nil, fmt.Errorf("ON UPDATE expression for %s: %w", c.ColumnName, err)
			}
			defaultExpr.OnUpdate = &ast.OnUpdate{
				Expr: expr,
			}
		}
		if defaultExpr != nil {
			cd.DefaultSemantics = defaultExpr
		}
	}

	// Identity column
	if c.IsIdentity != nil && *c.IsIdentity == "YES" {
		identity := &ast.IdentityColumn{
			Rparen: token.InvalidPos,
		}
		var params []ast.SequenceParam
		if strings.EqualFold(ptrStr(c.IdentityKind), "BIT_REVERSED_POSITIVE") ||
			strings.EqualFold(ptrStr(c.IdentityKind), "BIT_REVERSED_POSITIVE_SEQUENCE") {
			params = append(params, &ast.BitReversedPositive{
				BitReversedPositive: token.Pos(1),
			})
		}
		if c.IdentitySkipRangeMin != nil && c.IdentitySkipRangeMax != nil {
			params = append(params, &ast.SkipRange{
				Min: intval(*c.IdentitySkipRangeMin),
				Max: intval(*c.IdentitySkipRangeMax),
			})
		}
		if c.IdentityStartWithCounter != nil {
			params = append(params, &ast.StartCounterWith{
				Counter: intval(*c.IdentityStartWithCounter),
			})
		}
		if len(params) > 0 {
			identity.Params = params
			identity.Rparen = token.Pos(1)
		}
		cd.DefaultSemantics = identity
	}

	// AUTO_INCREMENT
	if c.ColumnDefault != nil && *c.ColumnDefault == "AUTO_INCREMENT" {
		cd.DefaultSemantics = &ast.AutoIncrement{}
	}

	// Column options
	if opts := colOpts[c.ColumnName]; len(opts) > 0 {
		var defs []*ast.OptionsDef
		for _, opt := range opts {
			defs = append(defs, optionsDef(opt.OptionName, parseOptionValue(opt.OptionType, opt.OptionValue)))
		}
		cd.Options = mkOptions(defs...)
	}

	return cd, nil
}

// parseOptionValue converts an INFORMATION_SCHEMA option value to an ast.Expr.
func parseOptionValue(optionType, optionValue string) ast.Expr {
	switch {
	case optionValue == "NULL":
		return nullval()
	case strings.EqualFold(optionType, "BOOL"):
		return boolval(strings.EqualFold(optionValue, "TRUE"))
	case strings.EqualFold(optionType, "INT64"):
		return intval(optionValue)
	case strings.EqualFold(optionType, "FLOAT64"):
		return &ast.FloatLiteral{Value: optionValue}
	default:
		// STRING type - value is stored with quotes in INFORMATION_SCHEMA
		v := strings.Trim(optionValue, "'\"")
		return strval(v)
	}
}

// parseRowDeletionPolicy parses e.g. "OLDER_THAN(created_at, INTERVAL 30 DAY)"
func parseRowDeletionPolicy(expr string) (*ast.RowDeletionPolicy, error) {
	// Parse via synthetic DDL
	ddl := fmt.Sprintf("CREATE TABLE _t (_c INT64) PRIMARY KEY (), ROW DELETION POLICY (%s)", expr)
	stmt, err := memefish.ParseDDL("", ddl)
	if err != nil {
		return nil, err
	}
	ct, ok := stmt.(*ast.CreateTable)
	if !ok || ct.RowDeletionPolicy == nil {
		return nil, fmt.Errorf("failed to parse row deletion policy: %s", expr)
	}
	return ct.RowDeletionPolicy.RowDeletionPolicy, nil
}

func constraintDisplayName(name string) string {
	if isAnonymousConstraintName(name) {
		return "<anonymous>"
	}
	return name
}

func referencedTableForConstraint(
	tc *infoschem.TableConstraint,
	usages []*infoschem.ConstraintTableUsage,
) (string, string, error) {
	// The FK's own CONSTRAINT_TABLE_USAGE row identifies its referenced table
	// without relying on a globally repeated unique-constraint name such as
	// PRIMARY_KEY.
	var schema, table string
	for _, usage := range usages {
		if usage.ConstraintSchema == tc.ConstraintSchema &&
			usage.ConstraintName == tc.ConstraintName &&
			usage.TableName != "" {
			if table != "" && (schema != usage.TableSchema || table != usage.TableName) {
				return "", "", fmt.Errorf(
					"foreign key %q has ambiguous referenced table metadata (%s and %s)",
					constraintDisplayName(tc.ConstraintName),
					tableDisplayName(schema, table),
					tableDisplayName(usage.TableSchema, usage.TableName),
				)
			}
			schema, table = usage.TableSchema, usage.TableName
		}
	}
	return schema, table, nil
}

func orderedReferencedColumnNames(
	fkCols []*infoschem.KeyColumnUsage,
	ccu []*infoschem.ConstraintColumnUsage,
	kcuByConstraint map[string][]*infoschem.KeyColumnUsage,
	ref *infoschem.ReferentialConstraint,
	refSchema, refTableName string,
) ([]string, error) {
	if len(fkCols) == 0 {
		return nil, fmt.Errorf("has no referencing column metadata")
	}
	uniqueColumns := kcuByConstraint[qualifiedConstraintKey(
		ref.UniqueConstraintSchema,
		ref.UniqueConstraintName,
	)]
	var referencedKeyColumns []*infoschem.KeyColumnUsage
	for _, column := range uniqueColumns {
		if column.TableSchema == refSchema && column.TableName == refTableName {
			referencedKeyColumns = append(referencedKeyColumns, column)
		}
	}
	sort.Slice(referencedKeyColumns, func(i, j int) bool {
		return referencedKeyColumns[i].OrdinalPosition < referencedKeyColumns[j].OrdinalPosition
	})

	if len(referencedKeyColumns) >= len(fkCols) && len(fkCols) > 0 {
		ordered := make([]string, len(fkCols))
		valid := true
		for i, fkColumn := range fkCols {
			position := fkColumn.PositionInUniqueConstraint
			if position == nil || *position < 1 || *position > int64(len(referencedKeyColumns)) {
				valid = false
				break
			}
			ordered[i] = referencedKeyColumns[*position-1].ColumnName
		}
		if valid {
			return ordered, nil
		}
	}

	// A single-column FK is unambiguous even in an older snapshot that lacks
	// the referenced unique-key KCU rows. Composite order cannot be inferred
	// from CONSTRAINT_COLUMN_USAGE because that table has no ordinal column.
	var fallback []string
	for _, column := range ccu {
		if column.TableSchema == refSchema && column.TableName == refTableName {
			fallback = append(fallback, column.ColumnName)
		}
	}
	if len(fkCols) == 1 && len(fallback) == 1 {
		return fallback, nil
	}
	return nil, fmt.Errorf(
		"cannot determine referenced column order for %d columns from available metadata",
		len(fkCols),
	)
}

// topSortTables sorts tables so interleave parents and FK targets come first.
func topSortTables(
	tables []*infoschem.Table,
	constraints []*infoschem.TableConstraint,
	constraintTableUsage []*infoschem.ConstraintTableUsage,
) ([]*infoschem.Table, error) {
	byName := make(map[string]*infoschem.Table, len(tables))
	for _, t := range tables {
		key := qualifiedTableKey(t.TableSchema, t.TableName)
		if _, exists := byName[key]; exists {
			return nil, fmt.Errorf("duplicate table identity %q", tableDisplayName(t.TableSchema, t.TableName))
		}
		byName[key] = t
	}

	dependencies := make(map[string][]string, len(tables))
	for _, t := range tables {
		childKey := qualifiedTableKey(t.TableSchema, t.TableName)
		if t.ParentTableName != nil && *t.ParentTableName != "" {
			parentKey := qualifiedTableKey(t.TableSchema, *t.ParentTableName)
			if _, ok := byName[parentKey]; ok && parentKey != childKey {
				dependencies[childKey] = append(dependencies[childKey], parentKey)
			}
		}
	}
	for _, tc := range constraints {
		if tc.ConstraintType != "FOREIGN KEY" {
			continue
		}
		childKey := qualifiedTableKey(tc.TableSchema, tc.TableName)
		refSchema, refTableName, err := referencedTableForConstraint(tc, constraintTableUsage)
		if err != nil {
			return nil, err
		}
		refKey := qualifiedTableKey(refSchema, refTableName)
		if _, ok := byName[refKey]; ok && refKey != childKey {
			dependencies[childKey] = append(dependencies[childKey], refKey)
		}
	}

	color := make(map[string]uint8, len(tables))
	var stack []string
	result := make([]*infoschem.Table, 0, len(tables))
	var visit func(string) error
	visit = func(key string) error {
		switch color[key] {
		case 1:
			cycleStart := 0
			for i, stackKey := range stack {
				if stackKey == key {
					cycleStart = i
					break
				}
			}
			cycle := append([]string(nil), stack[cycleStart:]...)
			cycle = append(cycle, key)
			parts := make([]string, 0, len(cycle))
			for _, cycleKey := range cycle {
				table := byName[cycleKey]
				parts = append(parts, tableDisplayName(table.TableSchema, table.TableName))
			}
			return fmt.Errorf("table dependency cycle: %s", strings.Join(parts, " -> "))
		case 2:
			return nil
		}

		color[key] = 1
		stack = append(stack, key)
		for _, dependency := range dependencies[key] {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		stack = stack[:len(stack)-1]
		color[key] = 2
		result = append(result, byName[key])
		return nil
	}

	for _, t := range tables {
		if err := visit(qualifiedTableKey(t.TableSchema, t.TableName)); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// groupBy groups a slice by a key function.
func groupBy[T any](items []T, key func(T) string) map[string][]T {
	m := make(map[string][]T)
	for _, item := range items {
		k := key(item)
		m[k] = append(m[k], item)
	}
	return m
}

// groupBy2 groups by primary key, then secondary key.
func groupBy2[T any](items []T, keys func(T) (string, string)) map[string]map[string][]T {
	m := make(map[string]map[string][]T)
	for _, item := range items {
		k1, k2 := keys(item)
		if m[k1] == nil {
			m[k1] = make(map[string][]T)
		}
		m[k1][k2] = append(m[k1][k2], item)
	}
	return m
}
