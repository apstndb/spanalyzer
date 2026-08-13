# Unobserved Query Plan Probe Matrix (2026-07-10)

This note turns the remaining execution-plan gaps into concrete schema, data,
query, and hint combinations. "Unobserved" means absent from this repository's
captured Omni corpus and operator notes; it does not assume that Spanner must
emit a new operator family.

The matrix was built by comparing:

- the current `spanner-query-plan-shape` cases and compact metadata corpus,
- `apstndb/spanner-hacks` at `c7233fb`, especially `operators.md` and
  `seek-residual-conditions.md`, and
- current official Spanner operator, vector, JSON search, placement, ML, and
  graph documentation.

At that `spanner-hacks` revision, the MiniBatch trio and search-index machinery
are documented, but there are no vector-index or JSON-search plan
reproductions. Those feature-to-corpus gaps were therefore the highest-yield
Omni probes.

A useful probe changes one planner input at a time and keeps an explicit
control query. Raw JSON should be retained whenever the compact tree changes,
because a new metadata value or scalar child link can be important even when
all relational nodes remain in known families.

## Residual audit update (2026-08-13)

A fresh comparison against current official Spanner documentation and the
`google/googlesql` public rewriter registry closed several gaps from this
matrix:

- retained pinned-Omni and managed-Spanner n-gram fuzzy retrieval and ordinary `LIKE`, `STARTS_WITH`,
  `ENDS_WITH`, and `REGEXP_CONTAINS` search-index acceleration, including
  parameter and minimum-token-size controls;
- retained pinned-Omni and managed-Spanner reciprocal-rank fusion with ANN and
  text retrieval in one plan;
- retained covering single- and multi-facet search compositions;
- retained query enhancement, its two documented statement hints, and the
  documented generated-column `SOUNDEX` composition, with pinned Omni and
  managed PLAN evidence;
- retained managed PLAN lowering for `AI.CLASSIFY`, `AI.IF`, and `AI.SCORE`,
  while preserving the pinned-Omni capability error; and
- added the newly registered upstream `REWRITE_HOP_FUNCTION` to the rewriter
  completeness gate with the stable Spanner `HOP`-not-found boundary.

Three environment-dependent candidates below remain intentionally unexecuted:
Local Split Union needs an Enterprise Plus multi-region instance and a
user-created instance partition; `ML.PREDICT` needs a user-authorized remote
model endpoint and IAM setup; graph-algorithm `EXPORT DATA` needs scale-up mode
and an authorized Cloud Storage destination. Creating those external resources
would expand the authority of a PLAN audit, so their exact recipes remain here
rather than being reported as product rejections. The managed checks performed
for this update used only temporary, destination-redacted database objects and
removed them after capture.

The same upstream comparison found no additional current Spanner probe to
promote. Graph `INSERT` has grammar, AST, resolver, and builder machinery in
`google/googlesql`, but its own default feature test still classifies the
statement as unsupported. The newer macro-visibility and named `MODEL` and
`SEQUENCE` argument surfaces are generic GoogleSQL features without matching
Spanner documentation or runtime evidence. They therefore remain upstream-only
leads rather than entries in the Spanner PLAN matrix.

## Newly observed while constructing the matrix

### Full outer join metadata

Independent tables that cannot be reduced through interleaving or declared
referential containment expose metadata that the `Albums` / `Songs` join
matrix did not:

```sql
SELECT s.FirstName, c.ConcertDate
FROM Singers AS s
FULL JOIN@{JOIN_METHOD=HASH_JOIN} Concerts AS c
ON s.SingerId = c.SingerId;

SELECT s.FirstName, c.ConcertDate
FROM Singers AS s
FULL JOIN@{JOIN_METHOD=MERGE_JOIN} Concerts AS c
ON s.SingerId = c.SingerId;
```

On Spanner Omni `2026.r1-beta` (image digest
`sha256:e98a088fa66d4a87dbb560d729bf21d998bb843f6018bd8dc118fe320e671886`),
the first query emits Hash Join metadata
`join_type=BUILD_PROBE_OUTER`; the second emits Merge Join metadata
`join_type=FULL_OUTER`. Both values were absent from the checked-in compact
metadata corpus. The hash shape appears directly in optimizer versions 7 and
8; versions 1 through 6 rewrite it as a Union All of outer and anti-semi join
branches. The merge value is stable across versions 1 through 8. These queries
now live in the `docs` built-in case.

### Vector index root and leaf scans

The `vector_search` built-in case produced two previously unseen raw scan
types on the same Omni image:

- `VectorIndexRootScan` over the vector index's internal depth-0 node table;
- `VectorIndexLeafScan` below a batch-driven key-range lookup.

The captured automatic-index plan had 80 nodes and `jq -c` single-line JSON
envelope SHA-256
`f78e06cf8de79d19665b169d2e8cab260e3a625c0d6937f7a910fa410a1edca3`.

The normalized spellings are `vector_index_root_scan` and
`vector_index_leaf_scan`. The useful controlled differences are:

| Query | Isolated observation |
| --- | --- |
| exact KNN with `_BASE_TABLE` | ordinary Table Scan plus Sort Limit control |
| ANN, automatic index | vector root/leaf scan path without a force hint |
| ANN plus extra-key equality | filter that can be evaluated through a vector index key |
| ANN plus stored-column predicate | leaf Filter Scan with Residual Condition |
| ANN projecting non-stored `Body` | additional order-preserving base-table back join |
| ANN over a filtered vector index | different internal node table and vector scan target |

The root scan has `Full scan: true`. That is a scan of vector centroids, not a
full base-table embedding scan. A literal `no_full_scan` contract will still
match it; vector-aware policies should use CEL to distinguish the normalized
scan types rather than treating the metadata bit as a latency conclusion.

### JSON search predicates

The `json_search` built-in case separates JSON search from Full Text Search.
On Omni `2026.r1-beta`, forced JSON containment and key-existence queries use
a SearchIndexScan and direct Search Predicate scalar links without the Search
Query Conversion TVF. Combining `SEARCH(TitleTokens, ...)` with
`JSON_CONTAINS(...)` restores that TVF and its VerifyDeterminism input.

The optimizer boundary is itself useful evidence: on this image, an unhinted
JSON containment query uses a base-table scan in optimizer versions 1 through
4 and the JSON search index in versions 5 through 8. Forced JSON search is
rejected in versions 1 through 4. The error text says the feature requires
version 6 or later even though version 5 succeeds, so this discrepancy needs a
Cloud Spanner DBaaS check before it is treated as a portable boundary.

Simple containment and simple key-existence plans differ only inside the
scalar Search Predicate's opaque `short_representation.description`. Their
relational topology and normalized metadata can therefore have the same
operator-tree digest. Do not parse the encoded token payload into a stable
contract; retain raw JSON/YAML when the distinction matters. Compound
predicates are structurally visible as a scalar Function with Search Predicate
descendants.

Keep both automatic-index and forced-index containment controls. On the live
default plan they shared the same relational shape, but the automatic case
retained a scalar `SPAN_JSON_CONTAINS_TO_SQUERY` Function while the forced case
constant-folded the JSON predicate into an opaque SQUERY token string.

## Runnable built-in matrix

```sh
go run ./tools/spanner-query-plan-shape \
  --case vector_search \
  --output compact-tree-metadata \
  --continue-on-error

go run ./tools/spanner-query-plan-shape \
  --case json_search \
  --output compact-tree-metadata \
  --continue-on-error
```

For optimizer boundaries, add `--optimizer-version-matrix`. For any changed
case, rerun with `--output json` and save the raw plan before changing the
normalizer.

The vector schema deliberately includes an embedding extra key, a stored
filter column, a hidden filtered-index predicate, and a non-stored payload.
The JSON schema includes JSON-only and mixed text-plus-JSON search indexes, a
stored projection, and a non-stored payload. The query sets include base-table
controls so that a generic Filter Scan or back join is not mistaken for a
feature-specific operator.

## DBaaS-only or data-dependent probes

### Local Split Union

This is the highest-confidence reproduction for the still-unobserved Local
Split Union operator, but it requires an Enterprise Plus multi-region instance
with a user-created instance partition. Omni rejects `CREATE PLACEMENT`.

```sql
CREATE PLACEMENT europeplacement
OPTIONS (instance_partition = "europe-partition");

CREATE TABLE PlacementSingers (
  SingerId INT64 NOT NULL,
  SingerName STRING(MAX) NOT NULL,
  BirthDate DATE,
  Location STRING(MAX) NOT NULL PLACEMENT KEY
) PRIMARY KEY (SingerId);
```

Seed at least one row in the placement. For a locality comparison, create a
second placement backed by another instance partition rather than inventing a
special default-placement name:

```sql
INSERT INTO PlacementSingers
  (SingerId, SingerName, BirthDate, Location)
VALUES
  (1, "Marc Richards", DATE "1970-09-03", "europeplacement");
```

The positive query is intentionally a projection-only scan:

```sql
SELECT BirthDate FROM PlacementSingers;
```

Capture a control table with the same columns but no `PLACEMENT KEY`. Expected
positive topology is Distributed Union -> Local Split Union -> Scan; compare
raw child-link types and locality metadata against the control.

Official sources: [Local Split Union](https://docs.cloud.google.com/spanner/docs/query-operators-unary#local-split-union),
[data placements](https://docs.cloud.google.com/spanner/docs/create-manage-data-placements).

### Generic TVF through `ML.PREDICT`

Change-stream and search-conversion TVFs are already observed. A remote model
is the documented route to the generic TVF shape shown in the operator docs:

```sql
CREATE MODEL GeminiPro
INPUT (prompt STRING(MAX))
OUTPUT (content STRING(MAX))
REMOTE
OPTIONS (
  endpoint = '//aiplatform.googleapis.com/projects/PROJECT/locations/LOCATION/publishers/google/models/gemini-pro',
  default_batch_size = 1
);
```

```sql
SELECT content
FROM ML.PREDICT(
  MODEL GeminiPro,
  (SELECT "Is 7 a prime number?" AS prompt),
  STRUCT(32 AS maxOutputTokens)
);
```

Use PLAN-only analysis so the endpoint is not invoked. Record whether the raw
TVF metadata contains `Name=ML.PREDICT`, a model name, or another value. Omni
currently rejects `CREATE MODEL`, so this requires DBaaS plus the model DDL and
IAM setup.

Official source: [Spanner ML prediction tutorial](https://docs.cloud.google.com/spanner/docs/ml-tutorial).

### MiniBatchAssign, MiniBatchKeyOrder, and RowCount

The known DBaaS plan uses a sorted non-covering index back join:

```sql
CREATE TABLE BatchSongs (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  TrackId INT64 NOT NULL,
  SongName STRING(MAX),
  Payload STRING(MAX),
) PRIMARY KEY (SingerId, AlbumId, TrackId);

CREATE INDEX BatchSongsBySongName ON BatchSongs(SongName);
```

Populate enough rows to create multiple base-table and index splits. Distribute
primary keys and `SongName` independently, and keep `Payload` wide enough that
the optimizer has a real reason to batch the non-covering lookup. Then run
`ANALYZE`, record the resulting package from
`INFORMATION_SCHEMA.SPANNER_STATISTICS`, and pin both optimizer version and
statistics package.

```sql
@{OPTIMIZER_VERSION=5, OPTIMIZER_STATISTICS_PACKAGE=PACKAGE_NAME}
SELECT *
FROM BatchSongs@{FORCE_INDEX=BatchSongsBySongName}
ORDER BY SongName DESC
LIMIT 1;
```

Run the following controlled axes rather than changing several together:

- row count: 1K, 100K, and 1M rows;
- limit: 1, 10, 100, and 1000;
- projection: index-only columns versus `Payload`;
- statement execution: default, `EXECUTION_METHOD=ROW`, and
  `EXECUTION_METHOD=BATCH`;
- additional parallelism: default, true, and false;
- optimizer version: 5 as the known positive, then 4, 6, 7, and 8;
- one fixed statistics package for every comparison.

The empty Omni database is a negative control, not evidence that these
operators no longer exist. `spanner-hacks` records the DBaaS optimizer-v5 plan
containing all three operators.

## Low-confidence Generate Relation probe

Superseded (2026-08-11): the retained set-operation matrix reproduced
`Generate Relation` under `INTERSECT ALL` and `EXCEPT ALL` on pinned Spanner
Omni `2026.r1-beta`. The graph-algorithm probe below remains a possible way to
explore another generator surface, but it is no longer needed to establish
that the operator is observable.

Constant SELECT, literal UNNEST, `GENERATE_ARRAY`, `GENERATE_DATE_ARRAY`, Union
All, and recursive CTEs have already produced Unit Relation, Array Unnest,
Union, or recursive spool operators instead. Repeating those variants is low
value.

The next distinct generator surface is the Preview graph-algorithm call, whose
result cardinality is generated by the algorithm rather than a relational
scan. Use a minimal directed property graph and test the PageRank call from the
official examples:

Capability update (2026-08-11): a retained plain
`GRAPH MusicGraph CALL PageRank() ...` analysis reached the runtime but
returned `Graph algorithm TVFs must be used in EXPORT DATA statements in
scale-up execution mode` at every optimizer version from 1 through 8. Thus a
plain graph-query PageRank plan is now an explicit negative control; the
`EXPORT DATA` form below remains the only meaningful QueryPlan candidate.

```sql
CREATE TABLE ProbeNodes (
  id INT64 NOT NULL
) PRIMARY KEY (id);

CREATE TABLE ProbeEdges (
  source_id INT64 NOT NULL,
  destination_id INT64 NOT NULL
) PRIMARY KEY (source_id, destination_id);

CREATE OR REPLACE PROPERTY GRAPH ProbeGraph
NODE TABLES (
  ProbeNodes AS ProbeNode
    KEY (id)
    LABEL Node PROPERTIES (id)
)
EDGE TABLES (
  ProbeEdges AS ProbeEdge
    KEY (source_id, destination_id)
    SOURCE KEY (source_id) REFERENCES ProbeNodes (id)
    DESTINATION KEY (destination_id) REFERENCES ProbeNodes (id)
    LABEL Edge PROPERTIES (source_id, destination_id)
);

INSERT INTO ProbeNodes (id) VALUES (1), (2), (3);
INSERT INTO ProbeEdges (source_id, destination_id)
VALUES (1, 2), (2, 3), (3, 1);

EXPORT DATA OPTIONS (
  uri = "gs://BUCKET/pagerank/*.csv",
  format = "csv"
) AS
GRAPH ProbeGraph
CALL PageRank(node_labels => ["Node"], edge_labels => ["Edge"])
YIELD node, score
RETURN node.id, score;
```

This is a hypothesis, not a claimed Generate Relation reproduction. First
establish whether `AnalyzeQuery` returns a standard QueryPlan for this statement
at all. If it does, compare PageRank with
`GRAPH ProbeGraph MATCH (n:Node) RETURN n.id` over the same schema and inspect
generic TVF or generator nodes. If it does not, remove this
candidate from QueryPlan coverage rather than inferring an operator from an
execution-only job.

Official source: [Spanner Graph algorithms](https://docs.cloud.google.com/spanner/docs/graph/algorithms).

## Capture checklist

For every positive or surprising negative result, retain:

1. exact DDL and seed-data generator digest;
2. exact SQL, parameter types, optimizer version, and statistics package;
3. backend identity and immutable image digest;
4. raw QueryPlan JSON, compact-tree-metadata, and normalization version;
5. operator display names, metadata keys and value types, and child-link
   type/variable pairs not present in the previous corpus;
6. the paired control query and its plan.

An operator-family-only summary is insufficient for unknown-unknown discovery:
both new findings in this pass first appeared as metadata values inside an
already-known `scan` or join family.
