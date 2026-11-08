package main

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	spanalyzer "github.com/apstndb/spanalyzer"
	"github.com/apstndb/spanalyzer/internal/querygen"
	"github.com/apstndb/spanalyzer/plancontract"
	"github.com/apstndb/spanemuboost"
)

type planReportAnalyzer interface {
	AnalyzePlan(context.Context, spanner.Statement, bool) (*spannerpb.QueryPlan, error)
	Close() error
}

type planReportOpenAnalyzer func(context.Context, spanemuboost.RuntimeHandle, querygen.QueryCodegenSchema, string, *spanalyzer.ProtoDescriptorSet, planReportOptimizer) (planReportAnalyzer, error)

type spannerPlanReportAnalyzer struct {
	clients *spanemuboost.Clients
}

func openPlanReportAnalyzer(ctx context.Context, runtime spanemuboost.RuntimeHandle, schema querygen.QueryCodegenSchema, baseDir string, descriptors *spanalyzer.ProtoDescriptorSet, optimizer planReportOptimizer, dropCreatedDatabases bool) (planReportAnalyzer, error) {
	clients, err := openPlanReportClients(ctx, runtime, schema, baseDir, descriptors, optimizer, dropCreatedDatabases)
	if err != nil {
		return nil, err
	}
	return &spannerPlanReportAnalyzer{clients: clients}, nil
}

func (a *spannerPlanReportAnalyzer) Close() error { return a.clients.Close() }

var errPlanReportRollback = errors.New("rollback PLAN-only transaction")

func (a *spannerPlanReportAnalyzer) AnalyzePlan(ctx context.Context, stmt spanner.Statement, dml bool) (*spannerpb.QueryPlan, error) {
	if !dml {
		return a.clients.Client.Single().AnalyzeQuery(ctx, stmt)
	}
	var plan *spannerpb.QueryPlan
	_, err := a.clients.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var err error
		plan, err = txn.AnalyzeQuery(ctx, stmt)
		if err != nil {
			return err
		}
		// AnalyzeQuery is PLAN-only. Explicit rollback also prevents a future
		// collector change from silently committing transaction side effects.
		return errPlanReportRollback
	})
	if errors.Is(err, errPlanReportRollback) {
		err = nil
	}
	return plan, err
}

func planReportSchemaScanTargets(schema querygen.QueryCodegenSchema, baseDir string) ([]plancontract.ScanTarget, error) {
	ddls, err := planReportDDLs(schema, baseDir)
	if err != nil {
		return nil, err
	}
	catalog, err := spanalyzer.BuildSchemaCatalog(schema.Name, strings.Join(ddls, ";\n"))
	if err != nil {
		return nil, err
	}
	targets := make([]plancontract.ScanTarget, 0, len(catalog.Tables)+len(catalog.Indexes))
	for _, table := range catalog.Tables {
		target := plancontract.ScanTarget{Name: table.Name.String(), Kind: "table", BaseTable: table.Name.String(), KeyColumns: []string{}}
		for _, key := range table.PrimaryKey {
			target.KeyColumns = append(target.KeyColumns, key.Name)
		}
		targets = append(targets, target)
	}
	for _, index := range catalog.Indexes {
		target := plancontract.ScanTarget{Name: index.Name.String(), Kind: "secondary_index", BaseTable: index.TableName.String(), StoredColumns: index.StoredColumns, NullFiltered: index.NullFiltered, KeyColumns: []string{}}
		for _, key := range index.Keys {
			target.KeyColumns = append(target.KeyColumns, key.Name)
		}
		targets = append(targets, target)
	}
	return targets, nil
}
