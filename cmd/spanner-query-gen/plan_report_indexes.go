package main

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	spanalyzer "github.com/apstndb/spanalyzer"
	"github.com/apstndb/spanalyzer/internal/querygen"
	"github.com/apstndb/spanalyzer/plancontract"
)

type planReportIndexObserver interface {
	ObserveIndexes(context.Context) ([]spanalyzer.MutationCountIndexObservation, error)
}

// ObserveIndexes reads one consistent snapshot of all regular indexes in the
// command-created database. Search/vector accounting remains explicitly unknown
// in the analyzer; this never invents a backing name from its hexadecimal suffix.
func (a *spannerPlanReportAnalyzer) ObserveIndexes(ctx context.Context) ([]spanalyzer.MutationCountIndexObservation, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	tx := a.clients.Client.ReadOnlyTransaction()
	defer tx.Close()
	var indexes []spanalyzer.MutationCountIndexObservation
	positions := map[string]int{}
	qualified := func(schema, name string) string {
		if schema == "" {
			return name
		}
		return schema + "." + name
	}
	err := tx.Query(ctx, spanner.NewStatement("SELECT TABLE_SCHEMA,TABLE_NAME,INDEX_NAME,INDEX_TYPE,INDEX_STATE,IS_UNIQUE,IS_NULL_FILTERED,SPANNER_IS_MANAGED FROM INFORMATION_SCHEMA.INDEXES WHERE INDEX_TYPE='INDEX' ORDER BY TABLE_SCHEMA,TABLE_NAME,INDEX_NAME")).Do(func(row *spanner.Row) error {
		var schema string
		var index spanalyzer.MutationCountIndexObservation
		if err := row.Columns(&schema, &index.Table, &index.Name, &index.Type, &index.State, &index.Unique, &index.NullFiltered, &index.Managed); err != nil {
			return err
		}
		positions[schema+"\x00"+index.Table+"\x00"+index.Name] = len(indexes)
		index.Table, index.Name = qualified(schema, index.Table), qualified(schema, index.Name)
		indexes = append(indexes, index)
		return nil
	})
	if err != nil {
		return nil, err
	}
	err = tx.Query(ctx, spanner.NewStatement("SELECT TABLE_SCHEMA,TABLE_NAME,INDEX_NAME,COLUMN_NAME,ORDINAL_POSITION FROM INFORMATION_SCHEMA.INDEX_COLUMNS ORDER BY TABLE_SCHEMA,TABLE_NAME,INDEX_NAME,ORDINAL_POSITION,COLUMN_NAME")).Do(func(row *spanner.Row) error {
		var schema, table, index, column string
		var position spanner.NullInt64
		if err := row.Columns(&schema, &table, &index, &column, &position); err != nil {
			return err
		}
		n, ok := positions[schema+"\x00"+table+"\x00"+index]
		if !ok {
			return nil
		}
		if position.Valid {
			indexes[n].Columns = append(indexes[n].Columns, column)
		} else {
			indexes[n].Stored = append(indexes[n].Stored, column)
		}
		return nil
	})
	return indexes, err
}

func planReportObservedIndexModel(ctx context.Context, analyzer planReportAnalyzer, schema querygen.QueryCodegenSchema, baseDir string, supplied *spanalyzer.MutationCountIndexSnapshot) (*spanalyzer.MutationCountAnalyzer, error) {
	model, err := querygen.MutationCountAnalyzerForSchema(schema, baseDir)
	if err != nil || model == nil {
		return model, err
	}
	observer, ok := analyzer.(planReportIndexObserver)
	if !ok {
		if supplied != nil {
			return model.WithObservedIndexes(supplied.SchemaSHA256, supplied.Indexes)
		}
		return model, nil
	}
	indexes, err := observer.ObserveIndexes(ctx)
	if err != nil {
		return nil, err
	}
	if supplied != nil {
		if supplied.SchemaSHA256 != model.SchemaSHA256() || !samePlanReportIndexes(supplied.Indexes, indexes) {
			return nil, fmt.Errorf("supplied index snapshot differs from this command-created database; omit --mutation-indexes-file to observe its generated names")
		}
	}
	return model.WithObservedIndexes(model.SchemaSHA256(), indexes)
}

func samePlanReportIndexes(a, b []spanalyzer.MutationCountIndexObservation) bool {
	normalize := func(in []spanalyzer.MutationCountIndexObservation) []spanalyzer.MutationCountIndexObservation {
		out := []spanalyzer.MutationCountIndexObservation{}
		for _, index := range in {
			if index.Type == "PRIMARY_KEY" {
				continue
			}
			index.Columns = slices.Clone(index.Columns)
			index.Stored = slices.Clone(index.Stored)
			if len(index.Stored) == 0 {
				index.Stored = nil
			}
			slices.Sort(index.Stored)
			out = append(out, index)
		}
		slices.SortFunc(out, func(a, b spanalyzer.MutationCountIndexObservation) int {
			return strings.Compare(a.Table+"\x00"+a.Name, b.Table+"\x00"+b.Name)
		})
		return out
	}
	return reflect.DeepEqual(normalize(a), normalize(b))
}

func planReportIndexScanTargets(indexes []spanalyzer.MutationCountIndex) []plancontract.ScanTarget {
	var targets []plancontract.ScanTarget
	for _, index := range indexes {
		managed := index.Managed
		target := plancontract.ScanTarget{Name: index.Name, Kind: "secondary_index", BaseTable: index.Table, KeyColumns: slices.Clone(index.Keys), StoredColumns: slices.Clone(index.Stored), NullFiltered: index.NullFiltered, Managed: &managed, IndexOrigin: index.Origin, InferredConstraintSelectors: slices.Clone(index.Constraints)}
		for _, selector := range index.Constraints {
			// These aliases are schema-derived, explicitly inferred associations;
			// an ambiguous physical target is left unresolved by the normalizer.
			constraint, ok := strings.CutSuffix(selector, ":referencing")
			if ok && !slices.Contains(target.Aliases, constraint) {
				target.Aliases = append(target.Aliases, constraint)
			}
		}
		targets = append(targets, target)
	}
	return targets
}
