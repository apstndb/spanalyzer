package infoschem

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// rejectedNamePatterns match the GoogleSQL name-resolution errors Spanner
// returns when a selected column does not exist in the owning view. They are
// applied only to an InvalidArgument rejection of a statement this package
// built, and a match is honored only for a column that was actually selected.
var rejectedNamePatterns = []*regexp.Regexp{
	regexp.MustCompile(`Unrecognized name: ([A-Za-z_][A-Za-z0-9_]*)`),
	regexp.MustCompile(`Name ([A-Za-z_][A-Za-z0-9_]*) not found inside`),
}

// ColumnsNamedByRejection returns, sorted, the candidate column names that an
// InvalidArgument rejection explicitly names. It returns nil for any other
// error or when no named column is a candidate. During a managed-service
// rollout each request can be served by a server with a different view of a
// system table, so an individual per-column probe can succeed while the
// multi-column statement is rejected; the rejection itself is then the only
// request-consistent evidence of which column is not queryable.
func ColumnsNamedByRejection(err error, candidates map[string]bool) []string {
	if err == nil || spanner.ErrCode(err) != codes.InvalidArgument || isClientClosedTransactionError(err) {
		return nil
	}
	message := status.Convert(err).Message()
	var named []string
	for _, pattern := range rejectedNamePatterns {
		for _, match := range pattern.FindAllStringSubmatch(message, -1) {
			if candidates[match[1]] && !slices.Contains(named, match[1]) {
				named = append(named, match[1])
			}
		}
	}
	slices.Sort(named)
	return named
}

// ExcludeUnqueryableColumns removes advertised-but-not-queryable columns of
// tableName from discovered[tableName] and returns their sorted names.
// When rejection is an InvalidArgument error that names one or more currently
// discovered columns, exactly those columns are excluded without probing.
// Otherwise every currently discovered column is probed with
// SELECT col FROM INFORMATION_SCHEMA.T LIMIT 0 in the same transaction and the
// columns whose probe fails with InvalidArgument or Unimplemented are excluded.
// Any other probe error, including a closed transaction, is propagated
// without mutating discovered.
func ExcludeUnqueryableColumns(
	ctx context.Context,
	txn *spanner.ReadOnlyTransaction,
	discovered DiscoveredColumns,
	tableName string,
	rejection error,
) ([]string, error) {
	if named := ColumnsNamedByRejection(rejection, discovered[tableName]); len(named) > 0 {
		for _, columnName := range named {
			delete(discovered[tableName], columnName)
		}
		return named, nil
	}
	return excludeUnqueryableColumns(discovered, tableName, func(table, column string) error {
		// table and column come from INFORMATION_SCHEMA.COLUMNS, so they are
		// safe to use as identifiers. LIMIT 0 forces name resolution without
		// reading user metadata rows.
		query := fmt.Sprintf(
			"SELECT `%s` FROM INFORMATION_SCHEMA.%s LIMIT 0",
			column,
			table,
		)
		iter := txn.Query(ctx, spanner.NewStatement(query))
		defer iter.Stop()
		_, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			return nil
		}
		return err
	})
}

func excludeUnqueryableColumns(
	discovered DiscoveredColumns,
	tableName string,
	probe func(table, column string) error,
) ([]string, error) {
	cols := discovered[tableName]
	if len(cols) == 0 {
		return nil, nil
	}

	var excluded []string
	for _, columnName := range slices.Sorted(maps.Keys(cols)) {
		if err := probe(tableName, columnName); err != nil {
			if isClientClosedTransactionError(err) {
				return nil, fmt.Errorf("probe %s.%s: %w", tableName, columnName, err)
			}
			switch spanner.ErrCode(err) {
			case codes.InvalidArgument, codes.Unimplemented:
				excluded = append(excluded, columnName)
			default:
				return nil, fmt.Errorf("probe %s.%s: %w", tableName, columnName, err)
			}
		}
	}
	for _, columnName := range excluded {
		delete(discovered[tableName], columnName)
	}
	return excluded, nil
}
