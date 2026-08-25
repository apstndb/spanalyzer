// Package infoschem provides Go structs for INFORMATION_SCHEMA tables.
package infoschem

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// ColumnDiscoveryQuery is the canonical query used for retained
// INFORMATION_SCHEMA surface observations.
const ColumnDiscoveryQuery = `SELECT TABLE_NAME, COLUMN_NAME, SPANNER_TYPE, ORDINAL_POSITION
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_SCHEMA = 'INFORMATION_SCHEMA'
ORDER BY TABLE_NAME, ORDINAL_POSITION`

// DiscoverColumns queries INFORMATION_SCHEMA.COLUMNS to build a set of all
// available tables and columns in the current database environment.
func DiscoverColumns(ctx context.Context, client *spanner.Client) (DiscoveredColumns, error) {
	txn := client.ReadOnlyTransaction()
	defer txn.Close()
	return DiscoverColumnsWithTxn(ctx, txn)
}

// DiscoveredColumnMetadata is the column type and ordinal surface advertised by
// INFORMATION_SCHEMA.COLUMNS, keyed by table and column name.
type DiscoveredColumnMetadata map[string]map[string]ColumnMeta

// DiscoverColumnMetadata returns the complete advertised INFORMATION_SCHEMA
// column surface without applying loader-specific queryability filtering.
func DiscoverColumnMetadata(ctx context.Context, client *spanner.Client) (DiscoveredColumnMetadata, error) {
	return DiscoverColumnMetadataWithTxn(ctx, client.Single())
}

// DiscoverColumnsWithTxn is like DiscoverColumns but uses the supplied
// ReadOnlyTransaction, allowing the caller to keep column discovery in the same
// snapshot as subsequent reads.
func DiscoverColumnsWithTxn(ctx context.Context, txn *spanner.ReadOnlyTransaction) (DiscoveredColumns, error) {
	metadata, err := DiscoverColumnMetadataWithTxn(ctx, txn)
	if err != nil {
		return nil, err
	}
	discovered := columnNames(metadata)

	if err := filterQueryableRollingColumns(discovered, func(tableName, columnName string) error {
		// tableName and columnName come exclusively from the static registry, so
		// they are safe to use as identifiers. LIMIT 0 forces name resolution
		// without reading user metadata rows.
		query := fmt.Sprintf(
			"SELECT `%s` FROM INFORMATION_SCHEMA.%s LIMIT 0",
			columnName,
			tableName,
		)
		iter := txn.Query(ctx, spanner.NewStatement(query))
		defer iter.Stop()
		_, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			return nil
		}
		return err
	}); err != nil {
		return nil, err
	}

	return discovered, nil
}

// DiscoverColumnMetadataWithTxn is like DiscoverColumnMetadata but uses the
// supplied read-only transaction.
func DiscoverColumnMetadataWithTxn(
	ctx context.Context,
	txn *spanner.ReadOnlyTransaction,
) (DiscoveredColumnMetadata, error) {
	type columnRow struct {
		TableName       string `spanner:"TABLE_NAME"`
		ColumnName      string `spanner:"COLUMN_NAME"`
		SpannerType     string `spanner:"SPANNER_TYPE"`
		OrdinalPosition int64  `spanner:"ORDINAL_POSITION"`
	}

	iter := txn.Query(ctx, spanner.NewStatement(ColumnDiscoveryQuery))
	defer iter.Stop()

	var rows []columnRow
	if err := spanner.SelectAll(iter, &rows); err != nil {
		return nil, err
	}

	discovered := make(DiscoveredColumnMetadata)
	for _, r := range rows {
		if discovered[r.TableName] == nil {
			discovered[r.TableName] = make(map[string]ColumnMeta)
		}
		discovered[r.TableName][r.ColumnName] = ColumnMeta{
			Name:            r.ColumnName,
			SpannerType:     r.SpannerType,
			OrdinalPosition: int(r.OrdinalPosition),
		}
	}

	return discovered, nil
}

func columnNames(metadata DiscoveredColumnMetadata) DiscoveredColumns {
	discovered := make(DiscoveredColumns, len(metadata))
	for tableName, columns := range metadata {
		discovered[tableName] = make(map[string]bool, len(columns))
		for columnName := range columns {
			discovered[tableName][columnName] = true
		}
	}
	return discovered
}

func filterQueryableRollingColumns(
	discovered DiscoveredColumns,
	probe func(tableName, columnName string) error,
) error {
	for _, meta := range informationSchemaTables {
		for _, column := range meta.Columns {
			if !column.Rolling || !discovered[meta.Name][column.Name] {
				continue
			}

			if err := probe(meta.Name, column.Name); err != nil {
				if isClientClosedTransactionError(err) {
					return fmt.Errorf("probe %s.%s: %w", meta.Name, column.Name, err)
				}
				switch spanner.ErrCode(err) {
				case codes.InvalidArgument, codes.Unimplemented:
					// During a rolling backend release, COLUMNS can advertise a
					// column before the owning view accepts it. Exclude only that
					// explicitly marked column from subsequent loader queries.
					delete(discovered[meta.Name], column.Name)
					continue
				default:
					return fmt.Errorf("probe %s.%s: %w", meta.Name, column.Name, err)
				}
			}
		}
	}
	return nil
}
