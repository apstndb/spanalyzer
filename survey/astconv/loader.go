package astconv

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/apstndb/spanalyzer/survey/infoschem"
	"google.golang.org/grpc/codes"
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
	schema, err := loadSchemaWithTxn(ctx, txn, discovered)
	if err != nil {
		return nil, err
	}
	schema.DatabaseName = databaseIDFromResource(client.DatabaseName())
	return schema, nil
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
	schema, err := loadSchemaWithTxn(ctx, txn, effectiveDiscovered)
	if err != nil {
		return nil, err
	}
	schema.DatabaseName = databaseIDFromResource(client.DatabaseName())
	return schema, nil
}

// databaseIDFromResource returns the database ID from a Spanner database
// resource name such as projects/P/instances/I/databases/D. A bare ID is
// returned unchanged. Empty input yields "".
func databaseIDFromResource(resource string) string {
	resource = strings.Trim(strings.TrimSpace(resource), "/")
	if resource == "" {
		return ""
	}
	if i := strings.LastIndex(resource, "/"); i >= 0 {
		return resource[i+1:]
	}
	return resource
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

		if err := queryTableExcludingUnqueryable(ctx, txn, meta, discovered, query, destFn(schema)); err != nil {
			return nil, fmt.Errorf("query %s: %w", meta.Name, err)
		}
	}

	return schema, nil
}

// queryTableExcludingUnqueryable runs one INFORMATION_SCHEMA table query.
// Retry only known compatibility flags rejected by the service. Dropping
// arbitrary metadata could remove object identity or schema semantics while
// claiming a successful reconstruction; all other columns remain fail-closed.
func queryTableExcludingUnqueryable(
	ctx context.Context,
	txn *spanner.ReadOnlyTransaction,
	meta *infoschem.TableMeta,
	discovered infoschem.DiscoveredColumns,
	query string,
	dest any,
) error {
	var excluded []string
	for {
		iter := txn.Query(ctx, spanner.NewStatement(query))
		err := selectSchemaRowsAtomically(iter, dest)
		if err == nil {
			return nil
		}
		if spanner.ErrCode(err) != codes.InvalidArgument {
			return err
		}
		candidates := compatibilityFlagCandidates(meta.Name, discovered)
		if len(candidates[meta.Name]) == 0 {
			if len(excluded) > 0 {
				return fmt.Errorf("still rejected after excluding %s: %w", strings.Join(excluded, ", "), err)
			}
			return err
		}
		roundExcluded, probeErr := infoschem.ExcludeUnqueryableColumns(ctx, txn, candidates, meta.Name, err)
		if probeErr != nil {
			return probeErr
		}
		if len(roundExcluded) == 0 {
			return err
		}
		for _, column := range roundExcluded {
			delete(discovered[meta.Name], column)
		}
		excluded = append(excluded, roundExcluded...)
		log.Printf("WARNING: %s columns advertised but not queryable (rolling release): %s", meta.Name, strings.Join(roundExcluded, ", "))
		query, err = meta.Query(discovered)
		if err != nil {
			return err
		}
	}
}

// compatibilityFlagCandidates deliberately excludes required identity and
// semantic columns. IS_INHERITED was observed advertised before query support
// in these option views; absence has an existing decoder/reconstruction path.
func compatibilityFlagCandidates(table string, discovered infoschem.DiscoveredColumns) infoschem.DiscoveredColumns {
	candidates := infoschem.DiscoveredColumns{table: {}}
	if (table == "COLUMN_OPTIONS" || table == "LOCALITY_GROUP_OPTIONS") && discovered[table]["IS_INHERITED"] {
		candidates[table]["IS_INHERITED"] = true
	}
	return candidates
}

// schemaRowIterator matches the iterator contract consumed by SDK SelectAll.
type schemaRowIterator interface {
	Next() (*spanner.Row, error)
	Do(func(*spanner.Row) error) error
	Stop()
}

// SelectAll may append rows before returning an error. Keep failed attempts
// out of the schema so a compatibility retry cannot duplicate partial rows.
func selectSchemaRowsAtomically(rows schemaRowIterator, dest any) error {
	value := reflect.ValueOf(dest)
	if !value.IsValid() || value.Kind() != reflect.Pointer || value.IsNil() || value.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("schema destination must be a non-nil pointer to slice")
	}
	buffer := reflect.New(value.Elem().Type())
	if err := spanner.SelectAll(rows, buffer.Interface(), spanner.WithLenient()); err != nil {
		return err
	}
	value.Elem().Set(buffer.Elem())
	return nil
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
