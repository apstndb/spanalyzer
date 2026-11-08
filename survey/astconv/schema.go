package astconv

import (
	"fmt"

	"github.com/apstndb/spanalyzer/survey/infoschem"
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
	// DatabaseName is the AST-facing database identity used by ALTER DATABASE.
	// It is never recoverable from INFORMATION_SCHEMA; the loader fills it from
	// the Spanner client's database resource name.
	DatabaseName           string
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
// Known object dependencies determine the output order, with the existing
// family/name order breaking ties. For example, sequences precede table defaults
// that use them, and SQL functions precede dependent views. Cycles are errors.
// References outside the supplied metadata remain external dependencies: this
// method also supports exporting a partial schema, not just an empty-DB restore.
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

	return orderDDLDependencies(ddls)
}
