---
type: Research Note
title: Mutation-count analysis alternatives, November 10
description: Three implemented experimental strategies compare schema-only costs, conditional intervals and old-row calibration before choosing a spanalyzer integration boundary.
tags: [spanner, mutation-count, dml, write-analysis, design]
status: draft
sources:
  - id: core
    resource: ../../../mutation_count.go
    title: Preview core API retaining baseline, old-row and measurement refinements
  - id: generator
    resource: ../../../internal/querygen/mutation_count.go
    title: Non-executing configured-write projection into explain-plan
  - id: lab
    resource: ../../../tools/mutation-count-lab/
    title: Experimental non-networked accounting command and regression tests
  - id: runtime
    resource: ../runtime-verification/mutation-count-20261110.md
    title: Local Omni and Emulator observations with retained wire payloads
  - id: managed
    resource: ../runtime-verification/mutation-count-managed-20261110.md
    title: Managed CommitStats replay and operation-specific queue boundaries
  - id: quotas
    resource: https://docs.cloud.google.com/spanner/quotas
    title: Current mutation and commit-size limits, including notes 3 and 10
  - id: comparison
    resource: https://docs.cloud.google.com/spanner/docs/dml-versus-mutations
    title: Different constraint and quota enforcement boundaries
  - id: commit-statistics
    resource: https://docs.cloud.google.com/spanner/docs/commit-statistics
    title: CommitStats measurement and cumulative DML totals
  - id: best-practices
    resource: https://docs.cloud.google.com/spanner/docs/dml-best-practices
    title: Batching, predicates and transaction tradeoffs beyond mutation counts
  - id: foreign-keys
    resource: https://docs.cloud.google.com/spanner/docs/foreign-keys/overview
    title: Managed backing indexes, sharing, informational constraints and delete actions
  - id: defaults
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax
    title: DML DEFAULT examples and counted cells
  - id: batch-write
    resource: https://docs.cloud.google.com/spanner/docs/batch-write
    title: MutationGroup enforcement boundary
  - id: wire
    resource: https://docs.cloud.google.com/spanner/docs/reference/rest/v1/Mutation
    title: External mutation representation
  - id: article
    resource: https://zenn.dev/google_cloud_jp/articles/7aad16e9b5760d
    title: User-supplied DML and Mutation comparison article
---

# Mutation-count alternatives

The recommended direction is **conditional intervals as the default**, with
explicit old-row or runtime observations refining them. A fixed per-row scalar
is useful only for a sufficiently constrained operation. Three alternatives
are implemented in `tools/mutation-count-lab`; none is a production quota gate
or a frozen public API. Mutation API and ordinary DML are the main scope. DDL
provides schema facts rather than an estimate of schema-change backfill work.

The lab preserves the three calibration-era alternatives for comparison. The
root library and `spanner-analyzer --mode mutation_count` are the integrated
preview; later input validation, queue and dependency corrections live there.
The lab is not a second maintained implementation of every preview feature.

## Implemented comparison

| Strategy | Actual implementation | Main benefit | Limitation exposed by the experiment |
| --- | --- | --- | --- |
| `shape` | Written-column count plus one index entry and stored cells, multiplied by row count | Very small, falsifiable baseline for the proposed per-row surface | Misses old index deletions, range deletion, automatic cells and transaction composition |
| `bounds` | Lower/upper arithmetic, absent/present branches, index membership, cardinality, cascade and unsupported-effect diagnostics | Reports uncertainty without pretending to read existing rows | Some upper bounds are unknown; a finite interval still depends on schema and operation assumptions |
| `row-image` | Old-row facts, NULL transitions, changed keys, stored-cell writes and calibrated same-key buffered updates | Explains observed differences and narrows scenarios | Calibrated to Omni LTS.2 on these fixtures; it is not a general managed-Spanner predictor |

Each strategy can use either DDL-declared indexes or a complete observed index
snapshot. The latter includes foreign-key backing indexes. The same command
and normalized wire input exercise both providers; this is an implemented
comparison rather than a choice made from sketches alone.[^lab]

On the 84 successful Omni operations, with observed indexes:

| Strategy | Exact matches | Observations inside interval | Outside interval |
| --- | ---: | ---: | ---: |
| `shape` | 45 | 45 | 39 |
| `bounds` | 48 | 84 | 0 |
| `row-image` | 84 | 84 | 0 |

With DDL alone the exact-match counts are 43, 46 and 82. This isolates the
benefit of reconciling the two tested referencing-side managed indexes. These
are **calibration results on the same cases used to develop the model**. They
are not held-out predictive accuracy. Emulator exposes no CommitStats for the
84 successful operations, so it contributes execution/metadata controls but
no numerical agreement score.[^runtime]

## Service boundary and what can be known

The current official pages distinguish 80,000 mutations per Mutation API
commit, per DML statement, and per BatchWrite MutationGroup. CommitStats for
DML is cumulative across the transaction, so a total above 80,000 is not itself
a violation. Both APIs remain subject to the separate 100 MiB transaction-size
limit. BatchWrite and Partitioned DML need their own adapters; this prototype
does not flatten them into one ordinary commit.[^quotas][^comparison][^commit-statistics][^batch-write]

| Operation/condition | Data-independent part | Data-dependent part |
| --- | --- | --- |
| Successful insert with fixed columns, no automatic cells, ordinary known indexes | Base written/PK cells and index insertion cells | NULL-filtered membership if values are unknown; constraints still determine success |
| Update with no relevant indexes or generated/on-update effects | Written targets plus identifying PK cells | Number of matched rows for DML |
| Indexed-key update or upsert | Written cells and an index-effect interval | Existing row, old key, old/new membership and whether the key actually changes |
| Replace | Supplied cells and replacement defaults | Existence and old index entries; omitted values change the resulting row |
| One Mutation API delete range | One base delete unit | Removed index entries, including indexed interleaved descendants |
| Foreign-key CASCADE or SET NULL | Declared relationships | Child fanout, key-range eligibility, downstream actions and index effects |
| Automatic columns | Declared expression/default metadata | API-specific accounting, expression dependencies, generated values and row existence |

Delete cost is not generally `per_row * rows`: the base range operation can be
one unit while secondary-index deletions grow with matched rows. Interleaved
cascade base deletion and foreign-key cascade also have different accounting
rules. Change streams do not consume mutation-count units, although their
bytes participate in the separate commit-size limit.[^quotas][^foreign-keys]

The quotas page's automatic-column formula and the DML syntax page's DEFAULT
examples cannot safely be implemented as a single unqualified row-price rule.
The local experiment additionally distinguishes Mutation API and DML inserts.
The runtime note preserves the exact cases. `bounds` leaves this policy
conditional; `row-image` labels its local calibration explicitly. The later
[managed capture](../runtime-verification/mutation-count-managed-20261110.md)
agrees on the shared automatic-column controls, without establishing a general
accounting rule for other expressions or schemas.[^quotas][^defaults][^runtime]

Same-key composition is another boundary: the observed buffered updates
`a -> b -> a` cost 6, while the two DML statements cost 10. The implementation
groups only the calibrated buffered-update case. Mixed repeated inserts,
replaces and deletes remain uncertain; mixed delete/write interactions have
an unknown total upper bound. DML rows are aggregated back into their actual
statement before checking its model limit.[^runtime]

## Schema and foreign-key alternatives

The root row-type catalog intentionally omits some write-accounting facts.
The experiment supplements it from memefish DDL: stored versus virtual
generation, defaults, foreign keys and interleaved cascade relationships.
Search/vector indexes, arbitrary ALTER histories, automatic on-update/identity
accounting and unevaluated generated values produce explicit unknowns. Generated
dependencies now propagate transitively through virtual/stored columns to
index keys and stored cells. Explicit ON UPDATE overrides bypass automation;
unknown written expressions never inherit their old value. This avoids
silently extending the row-type catalog into a complete physical-write model.

Three foreign-key approaches were considered:

1. **Predict generated physical names.** Rejected. Repeated identical suffixes
   do not establish a naming contract, and the observed engine suffixes differ.
2. **Synthesize one index per constraint.** Implemented only as an additional
   index-count upper bound. It must allow sharing and omitted backing indexes;
   informational constraints do not create referencing-side backing indexes.
3. **Reconcile observed physical indexes.** Implemented as the second provider.
   Check declared indexes against the snapshot, retain each physical index once,
   and attach constraint selectors when exactly one managed-index key prefix
   matches a semantic FK side. These selectors are explicitly **inferred**;
   this fixture does not prove the service's constraint-to-index association
   for every schema. Ambiguous matches receive no alias.[^foreign-keys][^runtime]

The supplemental DTO separates observed keys from DDL declarations. It does
not call every INFORMATION_SCHEMA key column a declared key: other runtimes
may expose implicit PK suffixes differently. A future PLAN adapter should
preserve origin, inference status and snapshot identity when enriching
`access_paths`. No generated index name or inferred alias is added to the
current plan-contract surface by this experiment.

## External writes and usage

From the `tools` module:

```sh
go run ./mutation-count-lab \
  -ddl mutation-count-lab/testdata/example.sql \
  -mutations mutation-count-lab/testdata/example-mutations.json \
  -strategy bounds

go run ./mutation-count-lab \
  -ddl mutation-count-lab/testdata/example.sql \
  -mutations mutation-count-lab/testdata/example-mutations.json \
  -facts mutation-count-lab/testdata/example-facts.json \
  -strategy row-image

go run ./mutation-count-lab \
  -ddl mutation-count-lab/testdata/example.sql \
  -dml "UPDATE Events SET Active='a' WHERE TRUE"

go run ./mutation-count-lab \
  -compare mutation-count-lab/testdata/omni.json.gz -index-source observed
```

The first example returns `[3,6]`; the supplied old image narrows the second
to 3 under the local model. The DML example has an unknown affected-row count,
not an assumed single row. `-facts` is an array parallel to normalized
operations. `old` follows the wire scalar representation (INT64 values are
decimal strings); `rows`, `index_rows` and `cascade_index_rows` are assertions
by the caller. An old image applied to multiple DML rows asserts a homogeneous
scenario, not a set of observed rows.

The input is a JSON **array** of `google.spanner.v1.Mutation` messages for one
commit, not a Go SDK reflection format. The validator checks supported wire
types, arity, tables/columns, primary-key availability, insert/replace required
columns, generated-column writes, lengths and commit-timestamp permission.
Unsupported types/operations fail explicitly. FK existence, uniqueness, CHECK
constraints, expression semantics and commit success remain service checks.
DML support is narrower: target-column checks and literal extraction for
INSERT VALUES, UPDATE and DELETE; complex expressions keep values unknown,
and conflict clauses/INSERT SELECT are rejected as outside the prototype.[^wire]

The probe demonstrates a unary gRPC interceptor capturing only
`CommitRequest.mutations`, without reaching into Go SDK private fields. A
production capture facility would additionally need opt-in redaction,
transaction-attempt identity, retry handling and aggregation boundaries. No
automatic recording of application data is installed by this change.

## Feedback disposition and integration order

| Requested surface | Alternatives considered | Implemented here / next boundary |
| --- | --- | --- |
| FK facts in `access_paths` and constraint-name contracts | Generated-name prediction; synthetic FK indexes; observed reconciliation | Reconciliation and inferred semantic selectors in the lab. Production PLAN enrichment needs a separately designed snapshot/origin contract. |
| Per-write estimate and maximum batch rows | Unconditional scalar; conservative interval; observed scalar | All three compared. A conditional homogeneous-row ceiling is emitted only when the interval has a finite positive per-row upper bound; bytes and contention are unchecked. |
| Execute synthetic writes during PLAN collection | Always write in `plan-report`; explicit opt-in mode; separate probe | Separate opt-in local `probe` command. Ordinary PLAN retrieval keeps its existing behavior. |
| CI mutation contract | `per_row <= n`; interval upper/lower; actual observed count | Interval and limit classifications implemented experimentally. Every result has `gate_eligible: false`; no existing config/schema contract is silently extended. |
| Non-generated writes | Reflect Go private fields; language-specific adapters; protobuf JSON | Protobuf JSON decoder/validator implemented, with supported-type limits. |
| Runtime capture | Go private reflection; commit interceptor; SDK-native Java accessors | Actual Go interceptor used by the retained probe. No Java client execution or production instrumentation claimed. |

For a subsequent integration, expose a reusable write-impact catalog and
structured uncertainty first, then connect generator write plans to it.
Keep runtime measurements as separately named observations; do not substitute
one synthetic row for a universal upper bound. A future contract could check a
finite scenario upper bound and fail as **indeterminate** when it is unknown,
rather than pass a scalar that omitted effects. Default to schema-only analysis
with no database reads, adding explicit same-snapshot row facts when needed.
Managed verification, held-out schemas and a reviewed preview contract should
precede enabling an enforcing CI gate. Batch sizing must also respect bytes,
contention, retries and the application's atomicity requirements.[^best-practices]

The user-supplied article was read as comparison background. Its benchmark
measurements are not spanalyzer performance measurements or a substitute for
the current official API limits.[^article]

## Preview integration update

Following the comparative experiment, the selected interval model is now
implemented in the root library as `MutationCountAnalyzer`, with a mutable
`spanalyzer.mutation-count/v0alpha1` report. The original lab remains a
comparison reproducer. `bounds` retains schema/cardinality assumptions;
`old_rows` optionally narrows that interval, and a digest-qualified observation
can identify the measured count for the supplied input. A measurement outside
the model is retained as a disagreement rather than replacing the model or
passing a quota check. Snapshot consistency and measurement association remain
caller assertions, not execution attestations.

`spanner-analyzer --mode mutation_count` accepts `--mutations-file` or DML
through `--sql`, with optional fact, index-snapshot and observation files.
JSON and YAML share the same report shape. The ordinary generator's structured
write plans expose per-method `mutation_counts`; inability to account for an
otherwise valid write is explicit in `mutation_count_error`. Key-qualified DML
includes zero matched rows, and missing values are distinct from known NULL.
Each estimate exposes index effects, including known zero effects for an
unaffected index. Per-operation contributions need not sum to a coalesced
same-key buffered-write total.

The core recognizes queue `Send`/`Ack` wire messages and the implicit DeliverTime
column. Internal queue accounting retains an unknown interval until an
input-qualified measurement is supplied. RECEIVE and RENEWLEASE queries have
their own unknown scope because they expose no caller CommitStats. Ordinary
generator `writes` configuration does not gain Send/Ack SDK code generation.
The [managed follow-up](../runtime-verification/mutation-count-managed-20261110.md)
retains 197 successful CommitStats observations and a partial lease-flow
boundary. All measured counts fit the cardinality-qualified intervals, and
161 old-row refinements are exact. The one approved database is deleted and
absent after the run. Managed rejects the 80,004-mutation API control accepted
by the local engines. No managed measurement is inferred from local calibration.
The core rejects ordinary wire queue writes and DML queue UPDATE/UPSERT based
on the retained negative controls, while valid Send/Ack and INSERT/DELETE keep
their internal-accounting upper bounds unknown.

An update's coefficient depends on its target set and affected access paths.
For the preview example with primary key `Id` and an index on `K STORING(S)`,
one known existing row with a changed `Other`, `S`, or `K` value costs 2, 4,
or 5 units respectively under the model. All three inputs write the same
number of base columns. The difference is no index maintenance, a stored-cell
write, or old-entry deletion plus insertion with stored data. NULL filtering,
unchanged keys and generated dependencies can change those effects. The
runnable godoc example `AnalyzeMutationsJSON_indexEffects` fixes the assumptions;
it is not an additional managed measurement or a universal multiplier.

Pre-PR review also tightens caller assertions: supplied row counts must fit
the normalized input bounds and must not contradict singular existence facts.
Deleted-index assertions must also fit derived cardinality and known membership;
unconditional indexes must agree on matched-row counts, while NULL-filtered or
unevaluated predicates can retain conditional membership. A delete shape's
explicit bounds remain limited to its single full key.
Parent/child deletes with potentially overlapping cascades no longer add
independent old-row index removals into an unjustified lower bound. Their
independent upper estimate remains available while overlap is unresolved.

## Upstream constraints and workarounds

The later [queue matrix](../runtime-verification/queue-mutation-count-managed-20261110.md)
adds 270 CommitStats measurements: Send is 5 across the tested key/payload
variants, Ack of a missing message with ignore_not_found is 1, and a buffered
Send/Ack pair can cost 3. Queue deletes and cascades have distinct observed
paths. The default internal-accounting upper bound stays unknown; exact-input
observations can refine it. PROTO values use a validated base64 envelope,
INSERT OR IGNORE has affected-row uncertainty, and INSERT SELECT accepts
cardinality facts without executing the source query. Combined DML/buffered
mutation inputs still require a separate scope and remain explicit replay gaps.

The separately approved **43-attempt follow-up is complete**, adding 32 measured
commits. OR IGNORE and SELECT inputs agree with 5 per actually inserted queue
row; all-existing inputs expose zero. Six ON CONFLICT controls and two RW
RENEWLEASE controls reject. Thirty measured inputs replay through the core;
two successful DELETE assertion inputs retain explicit pinned-parser gaps.
The additional DB is deleted and confirmed absent. The default model narrows
zero affected DML rows to zero, without turning the observed nonzero price into
a universal queue rule. The [full reference audit](mutation-count-reference-audit-20261110.md)
records every GoogleSQL DDL/DML family, source-backed dependency corrections,
alternatives and remaining adapters/accounting measurements.

- The pinned memefish v0.8.1 has no queue DDL AST. The write-count catalog uses
  a token-aware, statement-leading CREATE QUEUE adapter to reuse the common
  column/key grammar. It preserves strings, comments and identifiers, checks
  queue Payload/key structure and adds the implicit DeliverTime column. This
  does not add queue TVF registration to the general row-type analyzer or
  implement arbitrary ALTER/DROP QUEUE histories. The workaround is confined
  to [mutation_count_queue.go](../../../mutation_count_queue.go).
- Queue bookkeeping is not exposed as a complete ordinary index-price model.
  [Queue documentation](https://docs.cloud.google.com/spanner/docs/queues/queues-using)
  describes transactional send/ack and read TVFs; it does not supply a general
  public mutation-price formula. Unknown effects are preserved. The managed
  probe records DDL/index snapshots per phase rather than attributing a later
  queue to an earlier commit's schema.
- Pinned memefish v0.8.1 cannot represent expression index keys or a trailing
  DELETE ASSERT_ROWS_MODIFIED clause. The current DDL/DML references and
  [expression-index observations](../../observations/spanner-expression-indexes.md)
  identify the former boundary; the managed follow-up executes positive and
  negative DELETE assertions. Exact-input replay leaves the unsupported
  successful assertions unmodeled instead of stripping clauses. No dependency
  fork or upstream parser implementation is introduced here.
- Nested CLI modules consume immutable root versions independently when
  `GOWORK=off`; workspace execution uses the current local planner. Standalone
  build compatibility is checked separately from new-feature workspace tests.
  This local integration does not advance a published dependency or freeze
  an enforcing mutation-count contract.

[^lab]: [Experimental implementation](../../../tools/mutation-count-lab/).
[^runtime]: [Local runtime verification](../runtime-verification/mutation-count-20261110.md).
[^quotas]: [Spanner quotas and limits](https://docs.cloud.google.com/spanner/quotas), retrieved November 10, 2026 JST.
[^comparison]: [DML versus mutations](https://docs.cloud.google.com/spanner/docs/dml-versus-mutations), retrieved November 10, 2026 JST.
[^commit-statistics]: [Commit statistics](https://docs.cloud.google.com/spanner/docs/commit-statistics), retrieved November 10, 2026 JST.
[^best-practices]: [DML best practices](https://docs.cloud.google.com/spanner/docs/dml-best-practices), retrieved November 10, 2026 JST.
[^foreign-keys]: [Foreign-key overview](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview), retrieved November 10, 2026 JST.
[^defaults]: [DML syntax](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax), retrieved November 10, 2026 JST.
[^batch-write]: [Batch writes](https://docs.cloud.google.com/spanner/docs/batch-write), retrieved November 10, 2026 JST.
[^wire]: [Mutation wire representation](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/Mutation).
[^article]: [DML and Mutation comparison article](https://zenn.dev/google_cloud_jp/articles/7aad16e9b5760d), retrieved November 10, 2026 JST.
