package astconv

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"github.com/apstndb/spanalyzer/survey/infoschem"
)

// LoadSchema queries the live database's INFORMATION_SCHEMA under a single
// ReadOnlyTransaction and populates an astconv.Schema. Using one transaction
// ensures the reconstructed schema is snapshot-consistent.
func LoadSchema(ctx context.Context, client *spanner.Client) (*Schema, error) {
	txn := client.ReadOnlyTransaction()
	defer txn.Close()

	discovered, err := infoschem.DiscoverColumnsWithTxn(ctx, txn)
	if err != nil {
		return nil, fmt.Errorf("discover columns: %w", err)
	}
	infoschem.WarnUnknownColumns(discovered)
	return loadSchemaWithTxn(ctx, txn, discovered)
}

// LoadSchemaFromDiscovered populates an astconv.Schema from a fresh discovery
// in the ReadOnlyTransaction used for all subsequent reads. The supplied map is
// retained for API compatibility only; it cannot safely describe this snapshot.
func LoadSchemaFromDiscovered(ctx context.Context, client *spanner.Client, discovered infoschem.DiscoveredColumns) (*Schema, error) {
	txn := client.ReadOnlyTransaction()
	defer txn.Close()

	effectiveDiscovered, err := infoschem.DiscoverColumnsWithTxn(ctx, txn)
	if err != nil {
		return nil, fmt.Errorf("discover columns in schema read transaction: %w", err)
	}
	infoschem.WarnUnknownColumns(effectiveDiscovered)
	return loadSchemaWithTxn(ctx, txn, effectiveDiscovered)
}

// loadSchemaWithTxn populates an astconv.Schema using the supplied transaction.
func loadSchemaWithTxn(ctx context.Context, txn *spanner.ReadOnlyTransaction, discovered infoschem.DiscoveredColumns) (*Schema, error) {
	schema := &Schema{}

	for _, meta := range infoschem.AllTableMetas() {
		availCols, tableExists := discovered[meta.Name]
		if !tableExists || len(availCols) == 0 {
			continue
		}

		destFn := tableDestinations[meta.Name]
		if destFn == nil {
			// Tables like *PRIVILEGES exist in INFORMATION_SCHEMA but do not
			// map to DDL statements; skip them.
			continue
		}

		query, err := meta.Query(discovered)
		if err != nil {
			return nil, fmt.Errorf("build query for %s: %w", meta.Name, err)
		}

		iter := txn.Query(ctx, spanner.NewStatement(query))
		if err := spanner.SelectAll(iter, destFn(schema), spanner.WithLenient()); err != nil {
			return nil, fmt.Errorf("query %s: %w", meta.Name, err)
		}
	}

	return schema, nil
}

// tableDestinations maps each INFORMATION_SCHEMA table that contributes to DDL
// reconstruction to the corresponding slice in astconv.Schema.
var tableDestinations = map[string]func(*Schema) any{
	"CHANGE_STREAMS":            func(s *Schema) any { return &s.ChangeStreams },
	"CHANGE_STREAM_COLUMNS":     func(s *Schema) any { return &s.ChangeStreamColumns },
	"CHANGE_STREAM_OPTIONS":     func(s *Schema) any { return &s.ChangeStreamOptions },
	"CHANGE_STREAM_TABLES":      func(s *Schema) any { return &s.ChangeStreamTables },
	"CHECK_CONSTRAINTS":         func(s *Schema) any { return &s.CheckConstraints },
	"COLUMNS":                   func(s *Schema) any { return &s.Columns },
	"COLUMN_COLUMN_USAGE":       func(s *Schema) any { return &s.ColumnColumnUsage },
	"COLUMN_OPTIONS":            func(s *Schema) any { return &s.ColumnOptions },
	"CONSTRAINT_COLUMN_USAGE":   func(s *Schema) any { return &s.ConstraintColumnUsage },
	"CONSTRAINT_TABLE_USAGE":    func(s *Schema) any { return &s.ConstraintTableUsage },
	"DATABASE_OPTIONS":          func(s *Schema) any { return &s.DatabaseOptions },
	"INDEXES":                   func(s *Schema) any { return &s.Indexes },
	"INDEX_COLUMNS":             func(s *Schema) any { return &s.IndexColumns },
	"INDEX_OPTIONS":             func(s *Schema) any { return &s.IndexOptions },
	"KEY_COLUMN_USAGE":          func(s *Schema) any { return &s.KeyColumnUsage },
	"LOCALITY_GROUP_OPTIONS":    func(s *Schema) any { return &s.LocalityGroupOptions },
	"MODELS":                    func(s *Schema) any { return &s.Models },
	"MODEL_COLUMNS":             func(s *Schema) any { return &s.ModelColumns },
	"MODEL_COLUMN_OPTIONS":      func(s *Schema) any { return &s.ModelColumnOptions },
	"MODEL_OPTIONS":             func(s *Schema) any { return &s.ModelOptions },
	"PARAMETERS":                func(s *Schema) any { return &s.Parameters },
	"PLACEMENTS":                func(s *Schema) any { return &s.Placements },
	"PLACEMENT_OPTIONS":         func(s *Schema) any { return &s.PlacementOptions },
	"PROPERTY_GRAPHS":           func(s *Schema) any { return &s.PropertyGraphs },
	"REFERENTIAL_CONSTRAINTS":   func(s *Schema) any { return &s.ReferentialConstraints },
	"ROLES":                     func(s *Schema) any { return &s.Roles },
	"ROLE_CHANGE_STREAM_GRANTS": func(s *Schema) any { return &s.RoleChangeStreamGrants },
	"ROLE_COLUMN_GRANTS":        func(s *Schema) any { return &s.RoleColumnGrants },
	"ROLE_GRANTEES":             func(s *Schema) any { return &s.RoleGrantees },
	"ROLE_MODEL_GRANTS":         func(s *Schema) any { return &s.RoleModelGrants },
	"ROLE_ROUTINE_GRANTS":       func(s *Schema) any { return &s.RoleRoutineGrants },
	"ROLE_TABLE_GRANTS":         func(s *Schema) any { return &s.RoleTableGrants },
	"ROUTINES":                  func(s *Schema) any { return &s.Routines },
	"ROUTINE_OPTIONS":           func(s *Schema) any { return &s.RoutineOptions },
	"SCHEMATA":                  func(s *Schema) any { return &s.Schemata },
	"SEQUENCES":                 func(s *Schema) any { return &s.Sequences },
	"SEQUENCE_OPTIONS":          func(s *Schema) any { return &s.SequenceOptions },
	"SPANNER_STATISTICS":        func(s *Schema) any { return &s.SpannerStatistics },
	"TABLES":                    func(s *Schema) any { return &s.Tables },
	"TABLE_CONSTRAINTS":         func(s *Schema) any { return &s.TableConstraints },
	"TABLE_OPTIONS":             func(s *Schema) any { return &s.TableOptions },
	"TABLE_SYNONYMS":            func(s *Schema) any { return &s.TableSynonyms },
	"VIEWS":                     func(s *Schema) any { return &s.Views },
}
