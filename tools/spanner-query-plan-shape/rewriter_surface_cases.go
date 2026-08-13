package main

const rewriterSurfaceDDL = docsDDL + `
CREATE VIEW SingerNames SQL SECURITY INVOKER AS
SELECT s.SingerId, s.FirstName AS Name FROM Singers AS s;

CREATE VIEW NamedSingerIds SQL SECURITY INVOKER AS
SELECT s.SingerId FROM SingerNames AS s WHERE s.Name IS NOT NULL;
`

// rewriterSurfaceQueries retains direct PLAN evidence for Spanner-supported
// surfaces and exact capability errors for public GoogleSQL rewrite surfaces
// that the pinned Spanner Omni runtime does not expose.
var rewriterSurfaceQueries = []queryCase{
	{Label: "rewriter-surface/accepted/array-first", SQL: `SELECT ARRAY_FIRST(TicketPrices) AS value FROM Concerts`},
	{Label: "rewriter-surface/accepted/array-last", SQL: `SELECT ARRAY_LAST(TicketPrices) AS value FROM Concerts`},
	{Label: "rewriter-surface/accepted/array-min", SQL: `SELECT ARRAY_MIN(TicketPrices) AS value FROM Concerts`},
	{Label: "rewriter-surface/accepted/array-max", SQL: `SELECT ARRAY_MAX(TicketPrices) AS value FROM Concerts`},
	{Label: "rewriter-surface/accepted/array-slice", SQL: `SELECT ARRAY_SLICE(TicketPrices, 1, 2) AS value FROM Concerts`},
	{Label: "rewriter-surface/accepted/array-filter-with-index", SQL: `SELECT ARRAY_FILTER(TicketPrices, (e, i) -> e > i) AS value FROM Concerts`},
	{Label: "rewriter-surface/accepted/array-transform-with-index", SQL: `SELECT ARRAY_TRANSFORM(TicketPrices, (e, i) -> e + i) AS value FROM Concerts`},
	{Label: "rewriter-surface/accepted/array-includes-value", SQL: `SELECT ARRAY_INCLUDES(TicketPrices, 25) AS value FROM Concerts`},
	{Label: "rewriter-surface/accepted/array-includes-lambda", SQL: `SELECT ARRAY_INCLUDES(TicketPrices, e -> e > 25) AS value FROM Concerts`},
	{Label: "rewriter-surface/accepted/array-includes-any", SQL: `SELECT ARRAY_INCLUDES_ANY(TicketPrices, [25, 50]) AS value FROM Concerts`},
	{Label: "rewriter-surface/accepted/array-includes-all", SQL: `SELECT ARRAY_INCLUDES_ALL(TicketPrices, [25, 50]) AS value FROM Concerts`},
	{Label: "rewriter-surface/accepted/dot-product", SQL: `SELECT DOT_PRODUCT(TicketPrices, TicketPrices) AS value FROM Concerts`},
	{Label: "rewriter-surface/accepted/array-concat-agg-order-limit", SQL: `SELECT ARRAY_CONCAT_AGG(TicketPrices ORDER BY VenueId LIMIT 2) AS value FROM Concerts`},
	{Label: "rewriter-surface/accepted/view", SQL: `SELECT * FROM SingerNames`},
	{Label: "rewriter-surface/accepted/view-control", SQL: `SELECT SingerId, FirstName AS Name FROM Singers`},
	{Label: "rewriter-surface/accepted/nested-view", SQL: `SELECT * FROM NamedSingerIds`},
	{Label: "rewriter-surface/accepted/nested-view-control", SQL: `SELECT SingerId FROM Singers WHERE FirstName IS NOT NULL`},
	{Label: "rewriter-surface/accepted/insert-values-multi-row", SQL: `INSERT INTO Singers (SingerId, FirstName, LastName) VALUES (1, 'A', 'B'), (2, 'C', 'D')`, PlanMode: planModeReadWrite},

	{Label: "rewriter-surface/unsupported/aggregation-threshold", SQL: `SELECT WITH AGGREGATION_THRESHOLD OPTIONS(threshold=2, privacy_unit_column=SingerId) SingerId, COUNT(*) FROM Singers GROUP BY SingerId`},
	{Label: "rewriter-surface/unsupported/anonymization", SQL: `SELECT WITH ANONYMIZATION OPTIONS(epsilon=1, k_threshold=2, anonymization_userid_column=SingerId) SingerId, ANON_COUNT(*) FROM Singers GROUP BY SingerId`},
	{Label: "rewriter-surface/unsupported/differential-privacy", SQL: `SELECT WITH DIFFERENTIAL_PRIVACY OPTIONS(epsilon=1, delta=0.01, privacy_unit_column=SingerId) SingerId, COUNT(*) FROM Singers GROUP BY SingerId`},
	{Label: "rewriter-surface/unsupported/flatten", SQL: `SELECT FLATTEN([1, 2]) AS value`},
	{Label: "rewriter-surface/unsupported/like-any", SQL: `SELECT FirstName LIKE ANY (SELECT AlbumTitle FROM Albums) FROM Singers`},
	{Label: "rewriter-surface/unsupported/quantified-comparison", SQL: `SELECT 1 = ANY (SELECT SingerId FROM Singers)`},
	{Label: "rewriter-surface/unsupported/nulliferror", SQL: `SELECT NULLIFERROR(1 / SingerId) FROM Singers`},
	{Label: "rewriter-surface/unsupported/typeof", SQL: `SELECT TYPEOF(SingerId) FROM Singers`},
	{Label: "rewriter-surface/unsupported/multiway-unnest", SQL: `SELECT * FROM UNNEST([1, 2], [3, 4])`},
	{Label: "rewriter-surface/unsupported/pipe-assert", SQL: `FROM Singers |> ASSERT SingerId > 0 |> SELECT SingerId`},
	{Label: "rewriter-surface/unsupported/pipe-describe", SQL: `FROM Singers |> DESCRIBE`},
	{Label: "rewriter-surface/unsupported/pipe-if", SQL: `FROM Singers |> IF true THEN (|> SELECT SingerId)`},
	{Label: "rewriter-surface/unsupported/hop", SQL: `SELECT * FROM HOP(TABLE Concerts, 'BeginTime', INTERVAL 1 HOUR, INTERVAL 15 MINUTE)`},
	{Label: "rewriter-surface/unsupported/tumble", SQL: `SELECT * FROM TUMBLE(TABLE Concerts, DESCRIPTOR(BeginTime), INTERVAL 1 HOUR)`},
	{Label: "rewriter-surface/unsupported/nested-array-update", SQL: `UPDATE Concerts c SET (UPDATE c.TicketPrices price SET price = price + 1 WHERE price > 0) WHERE VenueId = 1`, PlanMode: planModeReadWrite},
}

type rewriterRetention string

const (
	rewriterDirectPlan    rewriterRetention = "direct-plan"
	rewriterDirectError   rewriterRetention = "direct-error"
	rewriterExistingPlan  rewriterRetention = "existing-plan"
	rewriterExistingError rewriterRetention = "existing-error"
	rewriterNotExposed    rewriterRetention = "not-exposed"
)

type rewriterCoverageEntry struct {
	Name           string
	Retention      rewriterRetention
	EvidenceLabels []string
	Note           string
}

// registeredRewriterCoverage enumerates every rewriter registered by
// RegisterBuiltinRewriters() at google/googlesql revision
// 1f8aa333f4d6353cd3a64471fc83121df72df3f7. A registered rewriter is not
// necessarily a user-visible Spanner syntax feature: several operate only on
// internal resolved nodes or on language features that Spanner does not expose.
var registeredRewriterCoverage = []rewriterCoverageEntry{
	{Name: "REWRITE_AGGREGATION_THRESHOLD", Retention: rewriterDirectError, EvidenceLabels: []string{"rewriter-surface/unsupported/aggregation-threshold"}},
	{Name: "REWRITE_ANONYMIZATION", Retention: rewriterDirectError, EvidenceLabels: []string{"rewriter-surface/unsupported/anonymization", "rewriter-surface/unsupported/differential-privacy"}},
	{Name: "REWRITE_BUILTIN_FUNCTION_INLINER", Retention: rewriterDirectPlan, EvidenceLabels: []string{"rewriter-surface/accepted/array-first", "rewriter-surface/accepted/array-last", "rewriter-surface/accepted/array-min", "rewriter-surface/accepted/array-max", "rewriter-surface/accepted/array-slice", "rewriter-surface/accepted/array-filter-with-index", "rewriter-surface/accepted/array-transform-with-index", "rewriter-surface/accepted/array-includes-value", "rewriter-surface/accepted/array-includes-lambda", "rewriter-surface/accepted/array-includes-any", "rewriter-surface/accepted/array-includes-all", "rewriter-surface/accepted/dot-product"}},
	{Name: "REWRITE_FLATTEN", Retention: rewriterDirectError, EvidenceLabels: []string{"rewriter-surface/unsupported/flatten"}},
	{Name: "REWRITE_GENERALIZED_QUERY_STMT", Retention: rewriterExistingError, EvidenceLabels: []string{"google-sql-surface/unsupported/core-pipe"}},
	{Name: "REWRITE_GROUPING_SET", Retention: rewriterExistingError, EvidenceLabels: []string{"google-sql-surface/unsupported/grouping-sets", "google-sql-surface/unsupported/rollup", "google-sql-surface/unsupported/cube"}},
	{Name: "REWRITE_HOP_FUNCTION", Retention: rewriterDirectError, EvidenceLabels: []string{"rewriter-surface/unsupported/hop"}, Note: "Managed Spanner recognizes the TVF position but reports that HOP is not exposed."},
	{Name: "REWRITE_INLINE_SQL_FUNCTIONS", Retention: rewriterNotExposed, Note: "Spanner does not expose SQL scalar-function definitions."},
	{Name: "REWRITE_INLINE_SQL_TVFS", Retention: rewriterNotExposed, Note: "Spanner does not expose SQL table-function definitions."},
	{Name: "REWRITE_INLINE_SQL_UDAS", Retention: rewriterNotExposed, Note: "Spanner does not expose SQL aggregate-function definitions."},
	{Name: "REWRITE_INLINE_SQL_VIEWS", Retention: rewriterDirectPlan, EvidenceLabels: []string{"rewriter-surface/accepted/view", "rewriter-surface/accepted/view-control", "rewriter-surface/accepted/nested-view", "rewriter-surface/accepted/nested-view-control"}},
	{Name: "REWRITE_INSERT_DML_VALUES", Retention: rewriterDirectPlan, EvidenceLabels: []string{"rewriter-surface/accepted/insert-values-multi-row"}},
	{Name: "REWRITE_IS_FIRST_IS_LAST_FUNCTION", Retention: rewriterExistingPlan, EvidenceLabels: []string{"gql-surface/analytic/is-first"}, Note: "Spanner documents IS_FIRST; IS_LAST is not exposed."},
	{Name: "REWRITE_LIKE_ANY_ALL", Retention: rewriterDirectError, EvidenceLabels: []string{"rewriter-surface/unsupported/like-any"}},
	{Name: "REWRITE_MATCH_RECOGNIZE_FUNCTION", Retention: rewriterExistingError, EvidenceLabels: []string{"google-sql-surface/unsupported/match-recognize"}},
	{Name: "REWRITE_MEASURE_TYPE", Retention: rewriterNotExposed, Note: "Spanner does not expose GoogleSQL measure definitions or AGG over MEASURE values."},
	{Name: "REWRITE_MULTIWAY_UNNEST", Retention: rewriterDirectError, EvidenceLabels: []string{"rewriter-surface/unsupported/multiway-unnest"}},
	{Name: "REWRITE_NULLIFERROR_FUNCTION", Retention: rewriterDirectError, EvidenceLabels: []string{"rewriter-surface/unsupported/nulliferror"}},
	{Name: "REWRITE_ORDER_BY_AND_LIMIT_IN_AGGREGATE", Retention: rewriterDirectPlan, EvidenceLabels: []string{"rewriter-surface/accepted/array-concat-agg-order-limit", "google-sql-surface/accepted/array-agg-order-limit"}},
	{Name: "REWRITE_PIPE_ASSERT", Retention: rewriterDirectError, EvidenceLabels: []string{"rewriter-surface/unsupported/pipe-assert"}},
	{Name: "REWRITE_PIPE_DESCRIBE", Retention: rewriterDirectError, EvidenceLabels: []string{"rewriter-surface/unsupported/pipe-describe"}},
	{Name: "REWRITE_PIPE_IF", Retention: rewriterDirectError, EvidenceLabels: []string{"rewriter-surface/unsupported/pipe-if"}},
	{Name: "REWRITE_PIVOT", Retention: rewriterExistingError, EvidenceLabels: []string{"google-sql-surface/unsupported/pivot"}},
	{Name: "REWRITE_PROTO_MAP_FNS", Retention: rewriterNotExposed, Note: "Spanner proto support does not expose the GoogleSQL MAP type or proto-map functions."},
	{Name: "REWRITE_QUANTIFIED_COMPARISONS", Retention: rewriterDirectError, EvidenceLabels: []string{"rewriter-surface/unsupported/quantified-comparison"}},
	{Name: "REWRITE_ROW_TYPE", Retention: rewriterExistingPlan, EvidenceLabels: []string{"google-sql-surface/accepted/select-as-value-subquery", "google-sql-surface/accepted/unnest-array-of-struct"}, Note: "Retains Spanner value-table and row-valued table-scan surfaces; PLAN does not identify the frontend rewrite by name."},
	{Name: "REWRITE_SUBPIPELINE_STMT", Retention: rewriterDirectError, EvidenceLabels: []string{"rewriter-surface/unsupported/pipe-if"}},
	{Name: "REWRITE_TUMBLE_FUNCTION", Retention: rewriterDirectError, EvidenceLabels: []string{"rewriter-surface/unsupported/tumble"}},
	{Name: "REWRITE_TYPEOF_FUNCTION", Retention: rewriterDirectError, EvidenceLabels: []string{"rewriter-surface/unsupported/typeof"}},
	{Name: "REWRITE_UNPIVOT", Retention: rewriterExistingError, EvidenceLabels: []string{"google-sql-surface/unsupported/unpivot"}},
	{Name: "REWRITE_UPDATE_CONSTRUCTOR", Retention: rewriterDirectError, EvidenceLabels: []string{"rewriter-surface/unsupported/nested-array-update"}, Note: "The pinned runtime rejects the documented nested-array UPDATE surface before returning a plan."},
	{Name: "REWRITE_VARIADIC_FUNCTION_SIGNATURE_EXPANDER", Retention: rewriterNotExposed, Note: "The pinned expander targets MAP functions, and Spanner does not expose the MAP type."},
	{Name: "REWRITE_WITH_EXPR", Retention: rewriterExistingPlan, EvidenceLabels: []string{"google-sql-surface/accepted/with-expression"}},
}
