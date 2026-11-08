---
type: Research Note
title: "Managed Spanner September release verification, 2026-09-22"
description: "Direct release-note retrieval and managed probes for automatic parameterization, graph options, PostgreSQL ILIKE and TABLESAMPLE, including a documented function mismatch."
tags: [spanner, managed, release-notes, graph, postgresql, query-plan]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: release
    resource: https://docs.cloud.google.com/spanner/docs/release-notes?hl=en
    title: "Spanner release notes, directly retrieved 2026-09-22"
  - id: hints
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#statement-hints
    title: "GoogleSQL statement hints"
  - id: pg
    resource: https://docs.cloud.google.com/spanner/docs/reference/postgresql/query-syntax
    title: "PostgreSQL statement hints and TABLESAMPLE"
  - id: functions
    resource: https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions
    title: "PostgreSQL function reference"
  - id: evidence
    resource: evidence/release-notes-20260922/receipt.json
    title: "Runtime, producer and documentation hashes"
---

# Managed Spanner September release verification

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

Observed on 2026-09-22 UTC through the Go SDK against managed GoogleSQL and
PostgreSQL databases. PROFILE reports optimizer version 9; a server build ID
is unavailable. The scope is the five September entries, ending September 18,
not an exhaustive historical release audit. No Emulator PLAN evidence is used.

## Documentation freshness

Direct full HTML retrieval includes the September 18 automatic parameterization
announcement. The same-day `dkcli get` response starts at September 17 and omits
that announcement. Use the direct page for this investigation. Full snapshots
are retained in the durable local directory named by the
[receipt](evidence/release-notes-20260922/receipt.json), with content hashes.

## Coverage

| Release date | Feature | Observation |
| --- | --- | --- |
| September 18 | Automatic SQL literal parameterization | Both dialects expose generated parameters for default/AUTO/ON, but not OFF; details below. |
| September 17 | Queues GA | Official announcement confirmed. Existing [September 18 runtime evidence](managed-spanner-queues-20260918.md) covers queue execution; not rerun here. |
| September 11 | Graph semantic options | CREATE succeeds; label/property description and synonyms survive metadata and GetDatabaseDdl. |
| September 11 | Graph key uniqueness validation | A join view fails creation with the default and true; false allows creation and MATCH. |
| September 3 | PostgreSQL case-insensitive matching | Four operator spellings work; the two documented function names fail, including with explicit text casts. |
| September 2 | PostgreSQL TABLESAMPLE | BERNOULLI and SPANNER.RESERVOIR work at tested boundaries; SYSTEM is explicitly unsupported, consistent with the reference. |

## Automatic parameterization

Each dialect used a two-row table with a primary key. The producer sent no
client parameters, changing the predicate literal from 1 to 2:

```sql
SELECT Id, Name FROM ReleaseProbe20260922 WHERE Id=1
```

The hint is `@{AUTO_PARAMETER='ON'}` in GoogleSQL and
`/*@AUTO_PARAMETER='ON'*/` in PostgreSQL. The reference defines AUTO as the
default and also accepts ON and OFF. Tests cover all three plus omission and
an invalid value.

- Default, AUTO and ON: `QueryStats.auto_parameters` contains
  `_auto_p0_int64_` with the corresponding value. OFF omits it.
- All valid modes return the correct row in a separate normal query.
- In GoogleSQL, the second default and AUTO PROFILE captures report
  `query_plan_cached=true` after changing the literal. This was not reported
  for the other captured pairs, including PostgreSQL. Cache use is an observed
  event, not a guarantee for every execution.
- Invalid values produce `InvalidArgument` in both dialects.

[GoogleSQL capture](evidence/release-notes-20260922/managed.jsonl) and
[PostgreSQL capture](evidence/release-notes-20260922/pg-managed.jsonl) retain
raw plans and statistics. This does not benchmark CPU or latency, cover DML,
secondary-index lookups or primary-key joins, or establish optimizer-version
boundaries. The sample only establishes primary-key SELECT behavior.

## Graph options and metadata

A view self-joins the two-row table on its primary key. Spanner cannot establish
the graph element key's uniqueness: default and explicit true return
`FailedPrecondition`. Explicit false succeeds, and MATCH returns count 2.
This is a schema-validation test, not evidence of correct behavior for actual
duplicate keys.

`PROPERTY_GRAPH_METADATA_JSON` includes a root `options` array with
`validate_element_key_uniqueness`, typed BOOL and SQL-text value FALSE or TRUE.
For semantic options, `labels[].options` and
`nodeTables[].propertyDefinitions[].options` contain description/synonyms.
Values are SQL text (for example `'People'` and `['person']`), not decoded
scalar/array JSON. GetDatabaseDdl retains the supplied semantic options.
The capture labels `graph-metadata`, `semantic-metadata`, and `semantic-export`
identify these records.

Source inspection of `survey/astconv/property_graph_metadata.go` shows no
options fields at these three locations. Its ordinary JSON decoder ignores
unknown fields, so preserving these options in reconstruction remains an
implementation gap. This is source-backed; a full reconstruction round trip
was not run here. Named labels, edges, and natural-language discoverability
quality were not tested. Ordinary column options retain their separate
[negative observations](managed-column-semantic-options-20260918.md).

## PostgreSQL matching and sampling

The initial combined ILIKE SELECT failed on function resolution; it cannot
establish any operator result. The
[isolated follow-up](evidence/release-notes-20260922/pg-followup.jsonl) establishes:

| Expression with inputs `Alice`, `a%` | Result |
| --- | --- |
| ILIKE / `~~*` | true |
| NOT ILIKE / `!~~*` | false |
| `pg.ilike(text,text)` | InvalidArgument: function does not exist |
| `pg.not_ilike(text,text)` | InvalidArgument: function does not exist |

The directly fetched function reference explicitly lists both `pg.*` signatures.
This is a documentation/runtime mismatch on the tested target; the cause
(rollout, documentation, or another limitation) is unresolved. Speculative
`spanner.*` aliases also fail. Unicode and NULL semantics are outside this probe.

For two input rows, BERNOULLI(0)/(100) returns 0/2 rows and
SPANNER.RESERVOIR(0)/(1)/(100) returns 0/1/2 rows. SYSTEM returns Unimplemented
with an error naming the two supported methods. Statistical uniformity was not
measured.

## Reproduction and cleanup

The receipt identifies original source and binary hashes, build/runtime identity,
and retained-source transformations. The three retained `.go.txt` producers
use environment-based target selection and direct SDK calls. Their historical
unused queue helpers are not part of this run. Target coordinates and credentials
are omitted. Both temporary PostgreSQL database lifecycles end with successful
DropDatabase; GoogleSQL graph/view/table drops succeed, followed by an empty
fixture-prefix TABLES query.

Run `python3 knowledge/research/runtime-verification/evidence/release-notes-20260922/verify.py`
for offline assertions over the retained captures. This checks the reported
observations and cleanup, without making new service requests.
