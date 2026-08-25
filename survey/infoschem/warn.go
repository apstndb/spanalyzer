package infoschem

import (
	"log"
	"sort"
)

// WarnUnknownColumns logs warnings for columns or tables present in the live
// database that are not covered by AllTableMetas(). This helps detect schema
// drift when Spanner adds new INFORMATION_SCHEMA columns.
func WarnUnknownColumns(discovered DiscoveredColumns) {
	allMetas := AllTableMetas()

	metaNames := make(map[string]bool)
	metaCols := make(map[string]map[string]bool)
	for _, m := range allMetas {
		metaNames[m.Name] = true
		cols := make(map[string]bool)
		for _, c := range m.Columns {
			cols[c.Name] = true
		}
		metaCols[m.Name] = cols
	}

	for tableName, actualCols := range discovered {
		if !metaNames[tableName] {
			log.Printf("WARNING: Discovered unknown table %q in INFORMATION_SCHEMA (repo needs updating)", tableName)
			continue
		}

		expectedCols := metaCols[tableName]
		var unknownCols []string
		for colName := range actualCols {
			if !expectedCols[colName] {
				unknownCols = append(unknownCols, colName)
			}
		}
		if len(unknownCols) > 0 {
			sort.Strings(unknownCols)
			log.Printf("WARNING: Discovered unknown columns in table %q: %v (repo needs updating)", tableName, unknownCols)
		}
	}
}
