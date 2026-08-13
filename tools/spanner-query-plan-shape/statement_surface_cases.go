package main

var statementSurfaceQueries = []queryCase{
	{
		Label: "statement-surface/call/cancel-literal",
		SQL:   `CALL cancel_query("spanalyzer-nonexistent-query-id")`,
	},
	{
		Label: "statement-surface/call/cancel-cast",
		SQL:   `CALL cancel_query(CAST("spanalyzer-nonexistent-query-id" AS STRING))`,
	},
	{
		Label: "statement-surface/call/compact-all",
		SQL:   `CALL compact_all()`,
	},
	{
		Label:  "statement-surface/call/cancel-parameter-error",
		SQL:    `CALL cancel_query(@query_id)`,
		Params: map[string]interface{}{"query_id": "spanalyzer-nonexistent-query-id"},
	},
	{
		Label: "statement-surface/call/cancel-expression-error",
		SQL:   `CALL cancel_query(CONCAT("spanalyzer-", "nonexistent-query-id"))`,
	},
	{
		Label: "statement-surface/call/unknown-procedure-error",
		SQL:   `CALL missing_spanalyzer_procedure()`,
	},
	{
		Label: "statement-surface/call/sql-optimizer-hint-error",
		SQL:   `@{OPTIMIZER_VERSION=1} CALL cancel_query("spanalyzer-nonexistent-query-id")`,
	},
	{
		Label: "statement-surface/routing/analyze-ddl-error",
		SQL:   `ANALYZE`,
	},
	{
		Label: "statement-surface/routing/start-batch-ddl-error",
		SQL:   `START BATCH DDL`,
	},
	{
		Label: "statement-surface/routing/execute-immediate-error",
		SQL:   `EXECUTE IMMEDIATE "SELECT 1"`,
	},
}
