package astconv

import (
	"fmt"

	"github.com/apstndb/spanner-emulator-survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

// Schema holds all INFORMATION_SCHEMA data needed for DDL reconstruction.
type Schema struct {
	Tables                 []*infoschem.Table
	Columns                []*infoschem.Column
	Indexes                []*infoschem.Index
	IndexColumns           []*infoschem.IndexColumn
	IndexOptions           []*infoschem.IndexOption
	TableConstraints       []*infoschem.TableConstraint
	CheckConstraints       []*infoschem.CheckConstraint
	ReferentialConstraints []*infoschem.ReferentialConstraint
	KeyColumnUsage         []*infoschem.KeyColumnUsage
	ConstraintColumnUsage  []*infoschem.ConstraintColumnUsage
	ConstraintTableUsage   []*infoschem.ConstraintTableUsage
	ColumnOptions          []*infoschem.ColumnOption
	ColumnColumnUsage      []*infoschem.ColumnColumnUsage
	Views                  []*infoschem.View
	ChangeStreams          []*infoschem.ChangeStream
	ChangeStreamTables     []*infoschem.ChangeStreamTable
	ChangeStreamColumns    []*infoschem.ChangeStreamColumn
	ChangeStreamOptions    []*infoschem.ChangeStreamOption
	Sequences              []*infoschem.Sequence
	SequenceOptions        []*infoschem.SequenceOption
	Models                 []*infoschem.Model
	ModelColumns           []*infoschem.ModelColumn
	ModelOptions           []*infoschem.ModelOption
	ModelColumnOptions     []*infoschem.ModelColumnOption
	PropertyGraphs         []*infoschem.PropertyGraph
	Placements             []*infoschem.Placement
	PlacementOptions       []*infoschem.PlacementOption
	Schemata               []*infoschem.Schema
	DatabaseOptions        []*infoschem.DatabaseOption
	SpannerStatistics      []*infoschem.SpannerStatistic
	Roles                  []*infoschem.Role
	RoleTableGrants        []*infoschem.RoleTableGrant
	RoleColumnGrants       []*infoschem.RoleColumnGrant
	RoleModelGrants        []*infoschem.RoleModelGrant
	RoleRoutineGrants      []*infoschem.RoleRoutineGrant
	RoleChangeStreamGrants []*infoschem.RoleChangeStreamGrant
	RoleGrantees           []*infoschem.RoleGrantee
	SchemaGrants           []*infoschem.SchemaGrant
	SequenceGrants         []*infoschem.SequenceGrant
	AllSchemaGrants        []*infoschem.AllSchemaGrant
	PlacementKeyColumns    []*infoschem.PlacementKeyColumn
	TableSynonyms          []*infoschem.TableSynonym
	Routines               []*infoschem.Routine
	RoutineOptions         []*infoschem.RoutineOption
	Parameters             []*infoschem.Parameter
	LocalityGroups         []*infoschem.LocalityGroup
	LocalityGroupOptions   []*infoschem.LocalityGroupOption
	TableOptions           []*infoschem.TableOption
	ProtoBundleTypes       []string // explicit proto type names; otherwise decoded from SCHEMATA.PROTO_BUNDLE
}

// ToDDLStatements converts all INFORMATION_SCHEMA data into DDL statements.
// The output order follows Spanner's convention:
//  1. CREATE PROTO BUNDLE
//  2. CREATE SCHEMA (named schemas)
//  3. ALTER DATABASE (database options)
//  4. CREATE TABLE (topologically sorted by parent-child)
//  5. CREATE INDEX / CREATE SEARCH INDEX
//  6. CREATE VECTOR INDEX
//  7. CREATE VIEW
//  8. CREATE CHANGE STREAM
//  9. CREATE SEQUENCE
//  10. CREATE MODEL
//  11. CREATE PROPERTY GRAPH
//  12. CREATE PLACEMENT
//  13. CREATE FUNCTION
//  14. CREATE LOCALITY GROUP
//  15. CREATE ROLE + GRANT
//  16. ALTER STATISTICS
func (s *Schema) ToDDLStatements() ([]ast.DDL, error) {
	var ddls []ast.DDL

	steps := []struct {
		name string
		fn   func() ([]ast.DDL, error)
	}{
		{"proto bundle", s.toProtoBundleDDL},
		{"named schemas", s.toSchemasDDL},
		{"database options", s.toDatabaseDDL},
		{"tables", s.toTablesDDL},
		{"indexes", s.toIndexesDDL},
		{"vector indexes", s.toVectorIndexesDDL},
		{"views", s.toViewsDDL},
		{"change streams", s.toChangeStreamsDDL},
		{"sequences", s.toSequencesDDL},
		{"models", s.toModelsDDL},
		{"property graphs", s.toGraphsDDL},
		{"placements", s.toPlacementsDDL},
		{"functions", s.toFunctionsDDL},
		{"locality groups", s.toLocalityGroupsDDL},
		{"roles & grants", s.toRolesDDL},
		{"statistics", s.toStatisticsDDL},
	}

	for _, step := range steps {
		out, err := step.fn()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", step.name, err)
		}
		ddls = append(ddls, out...)
	}

	return ddls, nil
}
