---
type: Research Note
title: Mutation-count DDL and DML reference audit, November 10
description: A full GoogleSQL DDL/DML reference review identifies automatic-column index dependencies, conflict branches, cascade fanout, specialized indexes and schema-change boundaries for the preview write analyzer.
tags: [spanner, mutation-count, ddl, dml, write-analysis, coverage]
status: draft
sources:
  - id: ddl
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language
    title: Complete GoogleSQL DDL reference, retrieved November 10
  - id: dml
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax
    title: Complete GoogleSQL DML reference, retrieved November 10
  - id: limits
    resource: https://docs.cloud.google.com/spanner/quotas
    title: Mutation units, automatic columns, byte limits and API boundaries
  - id: comparison
    resource: https://docs.cloud.google.com/spanner/docs/dml-versus-mutations
    title: Statement versus commit enforcement and write visibility
  - id: practices
    resource: https://docs.cloud.google.com/spanner/docs/dml-best-practices
    title: Batching, last_statement and execution constraints
  - id: documents
    resource: ../runtime-verification/evidence/mutation-count-reference-audit-20261110/document-sources.json
    title: Full-document retrieval metadata and content hashes
  - id: coverage
    resource: ../runtime-verification/evidence/mutation-count-reference-audit-20261110/section-inventory.json
    title: Fence-aware inventory of every reference heading
  - id: managed
    resource: ../runtime-verification/queue-mutation-count-managed-20261110.md
    title: Separately authorized 319-attempt matrix and 43-attempt follow-up
  - id: implementation
    resource: ../../../mutation_count_model.go
    title: Conditional intervals, dependency closure and unknown new values
  - id: regressions
    resource: ../../../mutation_count_dependencies_test.go
    title: Source-backed automatic-column and unknown-expression regressions
  - id: validation
    resource: ../runtime-verification/evidence/mutation-count-reference-audit-20261110/validation.json
    title: Earlier local code and fixture hashes, validation commands and exit statuses
  - id: review-regressions
    resource: ../../../mutation_count_review_test.go
    title: Pre-PR contradictory-cardinality and cascade-overlap regressions
  - id: review-validation
    resource: ../runtime-verification/evidence/mutation-count-reference-audit-20261110/review-validation.json
    title: Post-review source hashes and local validation commands
  - id: expression-indexes
    resource: ../../observations/spanner-expression-indexes.md
    title: Existing environment-scoped expression-index and parser boundaries
---

# Mutation-count reference audit

The preview should retain **schema bounds, old-row assertions and same-input
measurements as separate layers**. Reading the references does not establish a
universal service write-price formula. The immediate corrections prevent false
zero index effects; specialized storage and unmodeled syntax retain uncertainty
or explicit rejection.

## Review coverage and provenance

The full GoogleSQL DDL and DML reference bodies were retrieved with `dkcli` and
read directly on November 10, 2026 JST. The retained inventory has **214 DDL
headings across 21 top-level families**, and **53 DML headings across 11
families**. Code-fence comments are excluded from heading counts. The DDL body
has 2,973 lines and the DML body 1,292 lines. Their reported update times are
October 28 and October 1 respectively; those are document metadata, not feature
launch dates. Every family has a disposition below. Quota notes, the API
comparison and DML best practices supply supporting boundaries.

This is a GoogleSQL write-analyzer audit. PostgreSQL references and the complete
set of separate graph/function language pages are outside this coverage. The
main DDL reference's absence of a statement family is not proof that the service
lacks it. Property-graph/view/function metadata alone does not identify a new
caller row write; any expression they enable still needs value/cardinality
analysis at the write target.

The new managed evidence is the separately owned 43-attempt queue follow-up.
Automatic-column fixes below are **source-backed and locally regression-tested**;
they were not extra live probes. No additional database was created for the
reference audit. Earlier ordinary-write and queue captures retain their own
runtime, schema and measurement boundaries.

## Alternatives and immediate changes

| Decision | Alternatives considered | Implemented choice and reason |
| --- | --- | --- |
| Generated/index dependencies | Evaluate SQL expressions; invalidate every index on every update; follow schema dependencies | Follow stored and virtual expressions to a fixed point. Mark dependent keys and stored cells as potentially affected, without assigning computed values. Unrelated indexes retain their narrower bounds. |
| Unknown explicit values | Carry forward the old value; reject nonliteral expressions; erase only the written new value | Preserve support for expression/shape inputs and erase stale new values. Old data does not prove the result of `UPPER(V)` or another expression. |
| ON UPDATE | Keep a table-wide unknown; count a fixed automatic cell; track its trigger and explicit override | Track non-key writes, including same-value writes, and the override. Automatic base accounting remains unknown; affected indexes no longer report an unjustified zero. |
| Omitted automatic values | Treat omitted columns uniformly as NULL; evaluate them; distinguish automatic metadata | Stored/virtual generated and identity values remain unknown. A NULL-filtered index can therefore have an entry even when its generated input was omitted. |
| Zero affected queue DML rows | Keep every queue count unknown; apply a universal `5 * rows` rule; narrow only proven zero cardinality | An asserted zero affected-row count narrows successful DML to zero. Nonzero queue costs remain observations. This rule does not apply to missing Ack or empty query results. |
| DELETE assertions rejected by the parser | Strip the clause; patch the dependency; retain the gap | Retain exact SQL and the two successful measurements as parser gaps. Removing the clause would change the observation's input identity and validation semantics. |

Regression tests cover a virtual index key, NULL-filtered insertion, automatic
`STORING`, stored/virtual transitive chains, identity omission, ON UPDATE with an
explicit override, unknown expression/shape values, and zero-row queue DML with
a missing-Ack negative control. The correction does not evaluate generated SQL
or claim that recalculation always changes an index value.

## DDL family dispositions

| Reference family | Relevance and current disposition |
| --- | --- |
| Notation, reserved keywords, names | Affect parsing and target binding. No independent mutation multiplier. Quoted and qualified identifiers must preserve identity; aliases are not extra rows. |
| Data types | Values constrain validity, NULL membership and bytes. The model counts explicitly written top-level cells; PROTO payloads are opaque base64. ARRAY elements and PROTO fields are not separately priced from their count alone. ENUM and other unsupported wire encodings fail explicitly. |
| SCHEMA | Namespace creation/removal does not supply a row-write count. Qualified tables are retained by the ordinary catalog; a full schema-history estimator is not provided. |
| DATABASE | Sequence defaults can affect generated keys. Time zones can affect expression results. Optimizer, leader, read-lease and retention options are not given speculative mutation surcharges. Columnar storage is a separate unverified physical-accounting dimension. |
| LOCALITY GROUP | Column/table/index placement and SSD/HDD migration may change storage work. No independent caller mutation surcharge was established; compaction is outside a caller commit estimate. |
| PLACEMENT | Placement keys and geo-partition choices affect execution/storage. No count difference is inferred from placement metadata alone; cross-placement writes were not measured. |
| PROTO BUNDLE | Descriptor/type changes are schema operations. A later PROTO write can affect derived generated/index values, but nested field edits need an adapter and descriptor-aware value analysis. |
| TABLE | CREATE supplies physical/automatic columns, keys, relationships and constraints. The detailed table below separates these effects. ALTER, DROP and RENAME are not a complete supplemental write-schema history machine. |
| INDEX | Ordinary keys and `STORING` are modeled. NULL_FILTERED membership can use old/new values. `WHERE IS NOT NULL` may reference stored columns, so it remains an unevaluated predicate. UNIQUE affects successful-write preconditions. Expression keys are documented but unsupported by pinned memefish's identifier-only IndexKey. Interleaved-index deletion policy remains unknown. |
| SEARCH INDEX | TOKENLIST derivation, partitions, ordering, stored cells and search options can introduce maintenance beyond an ordinary index entry. The current upper bound stays unknown; token count is not treated as a public mutation formula. |
| VIEW | View definitions/projections do not create ordinary stored rows. A view/subquery used by INSERT can determine source cardinality; the source is not executed by this analyzer. |
| QUEUE | Internal Send/Ack/deletion accounting differs from visible-column arithmetic. Generated queue keys, TTL, disable flags and ALTER histories remain incomplete. The managed matrices establish only their named operation and schema observations. |
| CHANGE STREAM | Does not add mutation units toward the quota, according to quota note 3. Capture/exclusion/retention options can affect separate bytes or data records. The 100 MiB size limit still includes change-stream bytes. |
| ROLE | Authorization metadata; may determine whether a request succeeds. No row-write mutation-price increment is inferred. |
| GRANT and REVOKE | Privilege and role-membership changes are not priced as ordinary row writes. Column-level access and sequence privileges are successful-operation preconditions. |
| SEQUENCE | Values can supply defaults/identity keys and conflict decisions. Sequence allocation/gaps are not derived from caller CommitStats alone; implicit identity and sequence-related effects stay unknown. |
| STATISTICS | ANALYZE and package retention concern optimizer metadata/background work, outside caller write accounting. |
| MODEL | Endpoint registration does not copy model rows into the database. Inference in a source/value expression can affect values or cardinality; it does not justify a new constant write price. |
| VECTOR INDEX | Tree maintenance, extra keys and `STORING` need dedicated accounting. Null exclusion and REBUILD do not turn this into an ordinary-index price; upper bounds stay unknown. |

### Table features with potential fanout or automatic writes

| Feature | Data-independent information | Information still needed or boundary |
| --- | --- | --- |
| Explicit/implicit primary keys and HIDDEN | Schema identifies physical key cells; HIDDEN affects projection, not storage eligibility | Implicit identity is flagged unknown. A missing generated/default key is not generally handled by the strict wire adapter. |
| DEFAULT, UUID, identity and AUTO_INCREMENT | Declared automatic policy and explicitly supplied columns | Generated values, branch selection and API-specific accounting. AUTO_INCREMENT is rejected by the current ordinary catalog; no implicit support claim. |
| STORED generated columns | Expression graph and materialization metadata | Changed dependencies, generated values and calibrated charging policy. Transitive dependencies now propagate even through virtual columns. |
| Virtual generated columns | No base-row materialized cell in the model | They can be index keys or stored index values. Recalculation/membership remains conditional, rather than a zero index effect. |
| Generated primary keys | Their declaration and dependency restrictions | INSERT/UPSERT/REPLACE mutation key omission and UPDATE/DELETE key rules differ. The generic wire validator does not implement all these rules; do not infer full support from parsing the table. |
| ON UPDATE and commit timestamps | Trigger columns and explicit override | Automatic timestamp values/accounting are unknown. Same-value writes can still trigger automation; a known old timestamp cannot become the new timestamp by default. |
| CHECK, NOT NULL and UNIQUE | Successful-operation constraints | They can reject writes. Reads/validation costs do not automatically become mutation units. A failed transaction with absent stats is not a measured zero. |
| Foreign keys | Enforcement, both index sides and declared delete action | Reconcile physical backing-index sharing from an observed snapshot. CASCADE/SET NULL fanout and downstream updates remain unknown; two constraints are not necessarily two indexes. |
| INTERLEAVE IN PARENT versus IN | Parent existence enforcement and CASCADE declaration | Parent range deletion has different base accounting from individual index removals. Queue cascades have separate measurements. Non-enforced interleaving must not inherit CASCADE implicitly. |
| Row deletion policy / TTL | Background deletion policy and schema relationships | Not an inline surcharge on every foreground write. TTL execution, affected rows, cascading behavior and counts require a separate background scope. |
| Dictionary, locality and columnar options | Configuration and storage purpose | No formula for caller mutation-count deltas established here. Dictionary-table DML is still row DML; subsequent asynchronous search/storage maintenance is a separate question. |
| ALTER ADD/DROP/default/generated/FK/index changes | Final schema can change later write accounting | Existing-row backfills/validation and metadata-only changes are DDL operations. Prefer a canonical schema snapshot until supplemental ALTER histories are modeled. A rename/synonym must bind to the same physical target before counting. |

## DML family dispositions

| Reference family | Relevance and current disposition |
| --- | --- |
| Example tables and notation | Examples introduce defaults, ON UPDATE, interleaved cascades and compound values. They are fixtures/grammar, not an independent charging rule. |
| INSERT statement and examples | VALUES cardinality is known before conflicts; SELECT/UNNEST/subqueries need supplied affected-row facts. Omitted/default values and generated keys can add automatic effects. OR IGNORE depends on actually inserted rows. OR UPDATE has absent/present branches. |
| ON CONFLICT within INSERT | A unique-index target, optional update predicate, EXCLUDED defaults/generated values and a different UPDATE target set need a separate branch adapter. The core explicitly rejects this surface. Queue DO NOTHING/DO UPDATE were rejected in all six live controls; they cannot be normalized as OR IGNORE. |
| ASSERT_ROWS_MODIFIED within INSERT | A success/failure assertion is not a multiplier and does not replace actual cardinality semantics. The service also accepted the tested DELETE form, which is missing from the current documented DELETE grammar and pinned parser. Two successful DELETE measurements remain outside replay. |
| DELETE statement and examples | WHERE/subqueries determine matched rows, while physical range coalescing and index entry deletion determine units. Empty DML can be zero; a missing Mutation API delete/Ack is a different operation. Interleaved and FK cascades need separate fanout facts. |
| UPDATE statement and examples | The SET target set and identifying keys establish the base shape. Values are assigned simultaneously; unknown expressions cannot inherit old values. DEFAULT and ON UPDATE have their own automatic effects. Whole-array assignments target one top-level column in the model, not a row per array element. |
| Bound STRUCT parameters | Parameters/predicates can select many or no rows. The parser-only adapter does not bind parameters or execute predicates, so cardinality remains conditional. |
| Commit timestamps | Explicit or automatic timestamp writes can change indexed keys. Value resolution occurs at commit; later read restrictions affect whether a transaction can succeed, not an assumed extra mutation constant. |
| Update fields in protocol buffers | Qualified field paths, repeated fields, nested DELETE/UPDATE/INSERT and multiple-field edits need top-level target normalization and shared-cell accounting. They remain explicitly unsupported rather than counted as independent table rows or writes per nested element. |

THEN RETURN, WITH ACTION, EXCEPT and projection REPLACE change returned data;
projection REPLACE is not Mutation API Replace. The return clause is not a new
row write. FORCE_INDEX/PDML parallelism hints, predicates and source queries can
change access paths or execution; they are not independent write multipliers.
Old-row and cardinality assertions must describe the same transaction state,
including preceding DML, rather than a stale pre-transaction snapshot.

## Priority and remaining investigation

1. **Correctness fixes implemented:** automatic/transitive index dependencies,
   unknown new values, omitted automatic values and conditional zero-row DML.
   Their tests do not promote this mutable preview to an enforcing quota gate.
2. **Next parser/input adapters:** generated/default primary-key omission,
   expression indexes, complete conflict branches, DELETE assertions and nested
   PROTO/ARRAY updates. Require syntax/AST design and focused runtime controls;
   preserve explicit rejection until then. No upstream parser patch is claimed.
3. **Next accounting measurements:** generated keys, ON UPDATE, FK SET NULL and
   cascades across multiple levels, predicate indexes, search/vector indexes,
   queue generated keys and columnar/locality/placement variants. Use one
   schema/input/scope at a time; no new live lifecycle is authorized by this list.
4. **Separate scopes:** DDL backfill/validation, background TTL, sequence state,
   queue receive/renew, transaction bytes and lock/latency work. BatchWrite has a
   mutation-group limit; PDML has independent transactions and lower-bound row
   statistics. Cumulative DML CommitStats cannot stand in for each statement.

Batch DML/JDBC batching preserves dependencies but may change grouping. The
`last_statement` option defers validation and excludes later operations or
buffered mutations; this is a successful-execution constraint, not a fixed
discount. The official DEFAULT examples and automatic-column quota formula
still warrant API-specific conditional modeling rather than one global scalar.

## Local validation

The full root race suite, workspace query-generator consumer tests and
standalone tools tests pass. The five-module pre-push gate passes module tidy,
standalone builds, lint and govulncheck. The first sandbox attempt cannot fetch
the public vulnerability database; the host-network retry completes. No called
vulnerabilities are reported; module-level advisories remain distinct from
reachable-code findings. OKF conformance/quality/assets/publication and the
Markdown language check pass at that snapshot, which was uncommitted when the
receipt was captured.

## Pre-PR correctness review

The later review identifies two locally reproduced model defects and corrects
them before publication:

- A single VALUES row could accept an impossible affected-row assertion,
  including zero for an ordinary successful INSERT; a full-key delete could
  silently overwrite a contradictory existence/cardinality pair. The core now
  rejects those conflicting assertions before estimating units.
- Explicit parent and child deletes could sum old-row index removals even when
  a preceding cascade had already removed the same entry. Potential overlap now
  clears that additive lower bound and retains the independent upper estimate.
  This is conservative interaction handling, not a newly measured cascade price.

The runnable partial-update godoc example also makes the index coefficient
explicit: for a one-column PK and `K STORING(S)`, changed unrelated/stored/key
columns have modeled totals 2/4/5 with the declared old values. Source-backed
corrections and examples are separate from the retained managed observations.
The validation receipts describe their original local snapshots; subsequent
publication and review revisions are recorded by Git history.
