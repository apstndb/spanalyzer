//go:build integration && omni

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apstndb/spanalyzer/internal/querygen"
	"github.com/apstndb/spannerplan/plantree/reference"
)

// seekabilityAnnotationDDL is the timestamp-ordered shard schema from
// research/spanner-query-plan-shape/TIMESTAMP_ORDERED_SHARD_QUERY_OBSERVATIONS.md.
// The index declares two key columns, so a fully discretized shard range
// renders as "seek 2/2" while a residual timestamp range renders as
// "seek 1/2".
const seekabilityAnnotationDDL = `
CREATE TABLE Foo (
  random_id STRING(22) NOT NULL,
  shard_id INT64 NOT NULL,
  timestamp_order TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
) PRIMARY KEY(random_id);

CREATE INDEX OrderIndexDesc ON Foo(shard_id, timestamp_order DESC);
`

const seekabilityAnnotationConfigYAML = `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
  ddl: schema.sql
queries:
- name: ShardRange
  catalog: app
  kind: sql
  sql: |-
    SELECT random_id, shard_id, timestamp_order
    FROM Foo@{FORCE_INDEX=OrderIndexDesc}
    WHERE shard_id BETWEEN 0 AND 9
      AND timestamp_order BETWEEN TIMESTAMP "2018-09-05T09:00:00Z" AND TIMESTAMP "2018-09-05T10:00:00Z"
    ORDER BY timestamp_order DESC
    LIMIT 100
  result: {struct: ShardRangeRow}
- name: PerShard
  catalog: app
  kind: sql
  sql: |-
    SELECT c.random_id, c.shard_id, c.timestamp_order
    FROM UNNEST(GENERATE_ARRAY(0, 9)) AS OneShard,
    UNNEST(ARRAY(
      SELECT AS STRUCT f.random_id, f.shard_id, f.timestamp_order
      FROM Foo@{FORCE_INDEX=OrderIndexDesc} AS f
      WHERE f.shard_id = OneShard
        AND f.timestamp_order BETWEEN TIMESTAMP "2018-09-05T09:00:00Z" AND TIMESTAMP "2018-09-05T10:00:00Z"
      ORDER BY timestamp_order DESC
      LIMIT 100
    )) AS c
    ORDER BY c.timestamp_order DESC
    LIMIT 100
  result: {struct: PerShardRow}
`

// TestIntegrationSeekabilityAnnotationOnOmni is the acceptance demo for the
// schema-aware seekability annotation: the same shard-range query renders
// "seekable_key_size: 2/2" under optimizer version 6 (interval
// discretization) and "seekable_key_size: 1/2" under version 8 (timestamp
// range demoted to a residual condition), while the per-shard rewrite stays
// fully seekable on both. The annotation replaces the metadata value in
// place rather than appending a duplicate row suffix; the denominator comes
// from the catalog DDL, which the plan alone cannot supply.
func TestIntegrationSeekabilityAnnotationOnOmni(t *testing.T) {
	if os.Getenv("SPANEMUBOOST_ENABLE_OMNI_TESTS") == "" {
		t.Skip("set SPANEMUBOOST_ENABLE_OMNI_TESTS=1 to run Spanner Omni tests")
	}
	querygenOmniIntegrationRequireRuntime(t)

	dir := t.TempDir()
	writeIntegrationTestFile(t, filepath.Join(dir, "schema.sql"), seekabilityAnnotationDDL)
	config, err := querygen.ParseQueryCodegenConfigYAML([]byte(seekabilityAnnotationConfigYAML))
	if err != nil {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v", err)
	}
	plan, err := querygen.BuildQueryCodegenPlan(config, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}

	runtime := querygenOmniRuntime(t)

	tests := []struct {
		optimizerVersion string
		wantShardRange   string
	}{
		{optimizerVersion: "6", wantShardRange: "(seekable_key_size: 2/2)"},
		{optimizerVersion: "8", wantShardRange: "(seekable_key_size: 1/2)"},
	}
	for _, test := range tests {
		test := test
		t.Run("optimizer_version="+test.optimizerVersion, func(t *testing.T) {
			report, err := buildPlanReportWithRuntime(t.Context(), config, plan, dir, planReportOptions{
				Backend:    "omni",
				Format:     reference.FormatCurrent,
				RenderMode: reference.RenderModePlan,
				Annotate:   planReportAnnotateOptions{Seekability: true},
				Optimizer:  planReportOptimizerEnvironment{Version: test.optimizerVersion},
			}, runtime)
			if err != nil {
				t.Fatalf("buildPlanReportWithRuntime() error = %v", err)
			}
			plans := map[string]string{}
			for _, query := range report.Queries {
				if query.Status != "ok" {
					t.Fatalf("query %s status = %q, error = %q", query.Name, query.Status, query.Error)
				}
				plans[query.Name] = query.Plan
			}
			if !strings.Contains(plans["ShardRange"], test.wantShardRange) {
				t.Errorf("ShardRange plan does not contain %q:\n%s", test.wantShardRange, plans["ShardRange"])
			}
			if want := "(seekable_key_size: 2/2)"; !strings.Contains(plans["PerShard"], want) {
				t.Errorf("PerShard plan does not contain %q:\n%s", want, plans["PerShard"])
			}
		})
	}
}
