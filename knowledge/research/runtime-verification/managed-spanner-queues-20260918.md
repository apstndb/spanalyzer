---
type: Research Note
title: "Managed Spanner queues verification, 2026-09-18"
description: "Observed GoogleSQL queue DDL, catalog representation, receive and renewal TVFs, transactional enqueue/ack, delivery controls and cleanup on one managed database."
tags: [spanner, queues, managed, runtime, information-schema, transactions]
status: draft
sources:
  - id: history-import
    resource: evidence/history-import-20261108/receipt.json
    title: Historical publication provenance and original/retained byte hashes
  - id: overview
    resource: https://docs.cloud.google.com/spanner/docs/queues/queues-overview
    title: "Spanner queues overview, retrieved 2026-09-18"
  - id: usage
    resource: https://docs.cloud.google.com/spanner/docs/queues/queues-using
    title: "Use Spanner queues, retrieved directly 2026-09-18"
  - id: ddl
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create-queue
    title: "Queue DDL reference, retrieved directly 2026-09-18"
  - id: transactions
    resource: https://docs.cloud.google.com/spanner/docs/queues/queues-at-most-once
    title: "Exactly-once processing and at-most-once acknowledgment"
  - id: basic-evidence
    resource: evidence/queues-20260918/managed-r2.jsonl
    title: "Basic queue lifecycle and catalog observations"
  - id: schema-evidence
    resource: evidence/queues-20260918/schema.jsonl
    title: "DDL constraints, interleaving, controls and atomic acknowledgment"
  - id: lease-evidence
    resource: evidence/queues-20260918/lease-r2.jsonl
    title: "Controlled parameter-shape lease renewal observations"
  - id: plan-evidence
    resource: evidence/queues-20260918/plan.jsonl
    title: "Queue and TVF PLAN/PROFILE with composition restrictions"
  - id: catalog-evidence
    resource: evidence/queues-20260918/catalog.jsonl
    title: "Queue-present catalog surfaces and survey decode/reconstruction probe"
  - id: receipt
    resource: evidence/queues-20260918/receipt.json
    title: "Producer hashes, run status and retention transforms"
---

# Managed Spanner queues verification, 2026-09-18

Retained as historical research on November 8, 2026. Runtime and source claims
refer to the dated observations below; publication is not a rerun. The
[import receipt](evidence/history-import-20261108/receipt.json) distinguishes
original source hashes from retained publication bytes. For current selected
observations and projection policy, start at the [runtime index](index.md).

Queues executed successfully on the existing managed GoogleSQL test database.
This is a dated observation of one target, not a claim of fleet-wide rollout,
GA status, performance, or availability in every edition/region. The overview
lists Enterprise and Enterprise Plus editions.[^overview]

`dkcli get` of the overview returned `404 NOT_FOUND`; the official website
served the page. The usage and DDL pages were fetched directly as full HTML.
The retrieved release-notes page did not contain a queues announcement, so
this investigation does not assign a release date from those notes.
[Documentation retrieval metadata](evidence/queues-20260918/documentation.json)
records URLs and hashes; full HTML stays in the durable local task area.

Update, 2026-09-22: the directly retrieved [official release notes](https://docs.cloud.google.com/spanner/docs/release-notes#September_17_2026) now confirm the September 17 Queues GA announcement. The earlier retrieval observation above is historical. See the [September release verification](managed-release-notes-20260922.md) for retrieval provenance.

## Runtime and method

- Observed on 2026-09-18 UTC, managed Spanner, GoogleSQL. The requests do not
  expose a service build ID. Target coordinates and credentials are omitted.
- Go 1.26.6, darwin/arm64; initial and plan probes use `cloud.google.com/go/spanner v1.91.0`. The later standalone survey catalog probe uses its pinned v1.88.0 client and memefish v0.8.1 with `GOWORK=off`.
- Queries and DML were sent through the Go client, bypassing local SQL parsers.
  The initial CLI DDL used `CLI_PARSE_MODE=NO_MEMEFISH` and exited successfully.
- Temporary names were `QueueProbe20260918A` through `G`, with C-prefixed
  validation objects and a parent table. No pre-existing application object
  was changed. Every created object was dropped and absence checked.
- Receive calls use strong reads, bounded client deadlines and explicit stream
  cancellation. Short `max_duration=>'3s'` probes test bounded behavior, not
  a throughput recommendation or an exact wall-clock deadline guarantee.
- SQL, result rows, service errors and timestamps are retained. Lease tokens
  are omitted or hashed; actual token values are passed in memory only.
  The independent verifier checks the retained observations, not the service.

## Observed behavior

| Area | Probe | Result |
| --- | --- | --- |
| Basic DDL | `CREATE QUEUE` / `DROP QUEUE` | Accepted and executed |
| Payload types | BYTES, JSON, STRING | Created and consumed successfully |
| Payload constraints | Missing, nullable, or INT64 Payload | `InvalidArgument` with the corresponding constraint |
| Column constraints | Additional non-key column | `InvalidArgument`; only keys plus Payload allowed |
| Catalog | `INFORMATION_SCHEMA.TABLES` | `TABLE_TYPE='QUEUE'` |
| Catalog | `INFORMATION_SCHEMA.COLUMNS` | Automatic `DeliverTime TIMESTAMP NOT NULL`; `COLUMN_DEFAULT` is NULL |
| Schema export | `GetDatabaseDdl` | Exports `CREATE QUEUE`; omits implicit DeliverTime |
| Routines | `INFORMATION_SCHEMA.ROUTINES` | RECEIVE and RENEWLEASE entries are `TABLE FUNCTION` |
| Options | `INFORMATION_SCHEMA.TABLE_OPTIONS` | Advertises receive_mode, disable_send and disable_delivery |
| Enqueue rollback | INSERT followed by intentional transaction error | No message persisted |
| Ordinary reads | SELECT from queue | Immediate and future messages are queryable as rows |
| Delivery | RECEIVE with one immediate and one future message | Immediate message received; future message absent from bounded receive, still stored |
| Read consistency | RECEIVE with exact staleness of one minute | Rejected: queue receive requires strong concurrency |
| Receive metadata | `SELECT *` | Keys, Payload, DeliverTime, lease expiration, STRING lease token and BOOL last-batch flag |
| Ack rollback | DELETE followed by intentional transaction error | Message remains |
| Ack commit | DELETE with `ASSERT_ROWS_MODIFIED 1` | Deletes exactly one message |
| Duplicate ack | Same asserted DELETE again | `OutOfRange`, zero rows modified |
| Default delivery | INSERT omitting DeliverTime | Accepted; service assigns a current delivery timestamp |
| Send control | `disable_send=true` | INSERT rejected; earlier parent INSERT in same transaction rolled back |
| Delivery control | `disable_delivery=true` | Receive returns no rows while queued message remains stored |
| Resume delivery | ALTER QUEUE sets disable_delivery to NULL | Previously queued JSON message received |
| Atomic processing | Increment parent counter and ack in one transaction | Both commit |
| Duplicate processing | Increment then failing asserted ack, error propagated | Increment rolls back; counter remains 1 |
| Interleaving | `INTERLEAVE IN PARENT ... ON DELETE CASCADE` | Accepted; deleting parent removes queued child message |

Basic results are in the [lifecycle capture](evidence/queues-20260918/managed-r2.jsonl);
constraints and transactional controls are in the
[schema capture](evidence/queues-20260918/schema.jsonl).

The NULL `COLUMN_DEFAULT` does not imply that omitting DeliverTime fails:
the successful INSERT and subsequent timestamp read establish otherwise.
No metadata table/column containing `QUEUE` was advertised by the bounded
INFORMATION_SCHEMA surface query; that does not negate the TABLES/ROUTINES
representation observed above.

## Lease renewal

Renewal succeeded when binding an entire ARRAY<STRING> parameter. The official
usage page's Go example constructs `lease_tokens=>[@token1]` instead; the
analogous scalar-inside-array form failed in these observations.[^usage]

| Token and binding | Observed result |
| --- | --- |
| Fresh received token, `lease_tokens=>[@token]`, scalar string parameter | `InvalidArgument`: invalid lease token |
| Same token immediately afterward, `lease_tokens=>@tokens`, `[]string{token}` | Success, matching old token, new token and later expiration |
| Returned new token, scalar-inside-array form | Same `InvalidArgument` |
| That same new token, whole-array parameter | Success, another new token and later expiration |

The controlled experiment decoded the token directly into a Go string, kept
the receive iterator open, used `max_duration=>'20m'`, and completed renewals
before expiration. The first typed token had about 9.9 seconds remaining.
Token hashes link both successful renewals without exposing the tokens.
This rules out an already-used or expired token and early client cancellation
as explanations for these particular failures. It does not establish the
server's implementation cause or behavior with other clients/SQL variants.

A working form on this target was:

```go
stmt := spanner.Statement{
    SQL: "SELECT * FROM RENEWLEASE_QueueName(lease_tokens=>@tokens)",
    Params: map[string]interface{}{"tokens": []string{token}},
}
```

The earlier experiment renewed via an array parameter and then reused the
original token in the scalar form; that sequence alone cannot isolate binding
shape, so the controlled fresh/new-token experiment above is the evidence.

Another documentation discrepancy: `lease_tokens=>['not-a-token']` returned
`InvalidArgument` on both basic runs. The usage page says unparseable tokens
return no row. Keep this observation scoped to this input and target; expired
or acknowledged-token behavior was not tested.

## Query plans and SQL composition

An additional managed run at 2026-09-18 02:54 UTC captured 11 PLAN requests
and 3 completed PROFILE requests on an empty temporary queue. All three
PROFILE responses reported optimizer version 9. The queue was dropped and
its absence verified afterward.[^plan-evidence]

Both RECEIVE and RENEWLEASE expose this relational tree (scalar children
omitted), in PLAN and in the bounded empty-input PROFILE:

```text
Serialize Result
└── TVF
    └── Unit Relation
```

Each relational node reports `execution_method=Row`. The TVF node's metadata
contains that execution method, without a function-name property. Its links
include `Referenced` scalar arguments and named `Output` column references.
The receive output metadata has keys/Payload/DeliverTime plus lease expiration,
STRING lease token and BOOL last-batch flag; renewal returns old/new STRING
tokens and TIMESTAMP expiration. The queue's internal receiver work is not
expanded into scan operators in these captures. This is not proof of local-only
execution or absence of distributed work behind the TVF.

Ordinary `SELECT * FROM QueueProbe20260918F` instead has:

```text
Distributed Union
└── Distributed Union (call_type=Local)
    └── Serialize Result
        └── Scan (scan_type=TableScan, Full scan=true)
```

The distribution table is `_Queue_Data_QueueProbe20260918F`, while the scan
target is the user-visible queue name. Ordinary queue row reads and receive
TVF calls are distinct plan surfaces.

| PLAN request | Result |
| --- | --- |
| RECEIVE `SELECT *` | TVF tree above |
| RECEIVE `SELECT Id, Payload` | Same relational tree, selected result metadata |
| RECEIVE with `WHERE Id=1` | `Unimplemented` |
| RECEIVE with `WHERE Payload='hello'` | `Unimplemented` |
| RECEIVE with `LIMIT 1` | `Unimplemented` |
| RECEIVE with `COUNT(*)` | `Unimplemented` |
| RECEIVE joined to its ordinary queue table | `Unimplemented` |
| RECEIVE inside a CTE, selected from outside | `Unimplemented` |
| RENEWLEASE with whole-array or scalar-inside-array parameter | Both produce the TVF tree |

All six composition failures say that queue TVFs must not be combined with
other SQL statements and prescribe a direct `SELECT ... FROM RECEIVE_...`
shape. These are observed PLAN failures, not inferred optimizer rewrites;
no execution result is claimed for the rejected shapes. RENEWLEASE composition
was not tested. Direct column projection succeeds, but arbitrary expressions
or other untested compositions are not covered.

The lease argument forms differ below the TVF: whole-array binding has a
scalar `Parameter` with type `array`; `[@token]` has an `Array Constructor`
whose child is a scalar `Parameter`. Both pass PLAN, including the latter
with an intentionally invalid token. PLAN success therefore does not validate
token contents or explain the runtime renewal discrepancy by itself.

PROFILE returned plans and statistics for ordinary queue SELECT, an empty
RECEIVE stream, and RENEWLEASE with an empty token array. These are completion
and visibility checks, not performance measurements. The `1s` receive argument
produced an observed 4.64-second server elapsed statistic, so it must not be
represented as a hard end-to-end one-second deadline.

## INFORMATION_SCHEMA / SPANNER_SYS integration

Read-only comparisons at 2026-09-18 03:10 UTC found no new column definitions
relative to the retained 2026-09-16 observations:

| Surface | Columns | Compared result |
| --- | --- | --- |
| INFORMATION_SCHEMA | 310 | Same table/column/type/ordinal tuples; three sequential observations requested |
| SPANNER_SYS | 587 | Same table/column/type/ordinal tuples |

Both tools reported `producer_changed_only`, not a material surface change:
the current producer code differs from the baseline producer. The
[INFORMATION_SCHEMA comparison](evidence/queues-20260918/is-check.json) and
[SPANNER_SYS comparison](evidence/queues-20260918/sys-check.json) retain both
identities. A second full tuple comparison while a queue existed matched the
same baselines exactly.[^catalog-evidence] This does not claim fleet-wide
absence of new columns or compare against a pre-queues service release.
The September 16 SPANNER_SYS distribution-column work is separate and must
not be attributed to queues from timing alone.

Observed queue-specific rows fit existing catalog definitions:

- TABLES: `TABLE_TYPE='QUEUE'`; COLUMNS: keys, Payload, automatic DeliverTime.
- TABLE_OPTIONS: receive_mode and disable_send in this fixture.
- INDEXES / INDEX_COLUMNS: the queue primary key is represented.
- ROUTINES: RECEIVE and RENEWLEASE have `ROUTINE_TYPE='TABLE FUNCTION'`,
  `ROUTINE_BODY='EXTERNAL'`, `SECURITY_TYPE='INVOKER'`, empty routine
  definitions and `TABLE<...>` strings in DATA_TYPE and SPANNER_TYPE.
- ROUTINE_OPTIONS: each generated TVF has `queue_name` pointing to its owner.
- PARAMETERS: only RENEWLEASE's `lease_tokens` appeared (`ARRAY<STRING(MAX)>`
  SPANNER_TYPE); RECEIVE's accepted arguments were not advertised by this query.

The existing survey Table, Routine and Parameter structs decoded the live rows
without errors. No new decoder fields or analyzer column-manifest entries are
needed for these observations. However, pre-fix `ToDDLStatements` returned
zero statements and no error for the decoded queue and its generated TVFs,
because it skipped non-BASE TABLE and non-FUNCTION objects. This silent loss
was reproduced both against retained runtime metadata and in a regression test.

The converter now reports an explicit unsupported-queue error; it preserves
the boundary until CREATE QUEUE AST/conversion support exists. The generated
TVFs are not independent scalar-function DDL. `survey/AGENTS.md` and
`survey/UNSUPPORTED_DDL.md` now document that distinction. Full queue DDL
round-trip and automatic generated-TVF registration in the analyzer remain
unimplemented; this change does not claim otherwise.

## Change stream comparison

A follow-up managed observation on 2026-09-18 created an empty change stream
(no FOR clause), inspected its metadata, requested a READ function PLAN, then
dropped the stream and verified its absence. The
[raw capture](evidence/queues-20260918/change-stream.jsonl) and
[producer](evidence/queues-20260918/change-stream.go.txt) are retained.

`READ_ChangeStreamProbe20260918H` **does appear in ROUTINES**, with
`ROUTINE_TYPE = TABLE FUNCTION`, `ROUTINE_BODY = EXTERNAL`,
`SECURITY_TYPE = INVOKER`, an empty definition, and a
`TABLE<ChangeRecord ARRAY<STRUCT<...>>>` result type. PARAMETERS contains
`start_timestamp`, `end_timestamp`, `partition_token`,
`heartbeat_milliseconds`, and `read_options` in positions 1 through 5.
ROUTINE_OPTIONS contains `change_stream_name` pointing to the owning stream.
The PLAN succeeds and includes `Serialize Result -> ChangeStream TVF`, unlike
the generic `TVF` display name observed for queue functions.

Therefore, the earlier statement that ROUTINES contains only UDFs was incorrect:
both change stream and queue generated TVFs are represented on this target.
This capture establishes current behavior, not when catalog exposure began.
The [official change stream documentation](https://docs.cloud.google.com/spanner/docs/change-streams/details)
describes automatic creation of the special READ function alongside the stream.
No change records were read; the query observation used PLAN only.

### Historical catalog evidence

The historical interpretation used here is **Change Streams -> UDFs -> Queues**:
ROUTINES already described change stream read functions in the earliest
documentation containing that section verified in this investigation; UDFs and
queue-generated functions extend the catalog's scope. This ordering is consistent
with the retained evidence and the user's historical account. ROUTINES must not
be described as originating as a UDF-only catalog, with Change Streams added
recently. Exact introduction dates are not needed for this interpretation and
are not asserted here.

The [2024-05-26 Wayback snapshot](https://web.archive.org/web/20240526142656/https://cloud.google.com/spanner/docs/information-schema#routines)
(page last updated 2024-05-23) explicitly describes ROUTINES as listing the
database's change stream read functions. ROUTINE_OPTIONS and PARAMETERS are
also described in terms of change stream read functions. Thus, this was a
documented Change Stream catalog surface years before the queue probe; the
previous UDF-only description was historically misleading as well as incorrect
today. The [retained section](evidence/queues-20260918/routines-history-20240526.html.txt)
and [provenance](evidence/queues-20260918/routines-history.json) preserve the evidence.

An earlier [Emulator commit dated 2023-07-31](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/commit/cc0e5b3b2a9fc57aadd5ec394a9ad6b17fbbb379)
adds ROUTINES, ROUTINE_OPTIONS and PARAMETERS column metadata. This corroborates
an older catalog surface but does not establish managed rollout or populated
rows on that date. The official [2022-05-27 release note](https://docs.cloud.google.com/spanner/docs/release-notes#May_27_2022)
announces Change Streams, without dating ROUTINES separately.

The checked 2022-08-11 Korean and 2023-01-29 Chinese Wayback snapshots include
CHANGE_STREAMS metadata but lack a ROUTINES section. Translation lag or missing
documentation prevents using this absence as evidence of service behavior.
These gaps do not contradict the Change Streams -> UDFs -> Queues ordering.
Neither the initial service rollout dates nor the first-ever documentation
publication date was established. The 2025-08-13 MySQL UDF library announcement
is not treated as the initial SQL UDF release date.

The archived ROUTINE_TYPE description says always FUNCTION, whereas the current
managed observation returns TABLE FUNCTION. Historical documentation wording
must not override the retained current runtime values.

## Reproduction and limits

The [receipt](evidence/queues-20260918/receipt.json) identifies the exact producer
sources and executables. The producer takes destination variables only from the
existing user-level managed verification profile. Do not paste resolved
coordinates into a command file or report. Review the hardcoded temporary
names before rerunning; use fresh names rather than immediately recreating a
queue with the same name, which has a documented receive-reset limitation.

The first run's metadata queries accidentally used unsupported
`CONTAINS_SUBSTR`; those errors are retained in `managed-initial.jsonl` but are
excluded from metadata conclusions. The corrected run uses `STRPOS`.
The initial CLI create's successful exit and subsequent catalog/DDL export
are retained; an empty CLI output alone is not treated as DDL proof.

Run the retained checks with:

```sh
python3 knowledge/research/runtime-verification/evidence/queues-20260918/verify.py
python3 knowledge/research/runtime-verification/evidence/queues-20260918/verify-plan.py
python3 knowledge/research/runtime-verification/evidence/queues-20260918/verify-catalog.py
```

Not tested: PostgreSQL, Emulator or Omni, local parser/analyzer support, SDK
Send/Ack helpers, actual delayed delivery after its scheduled time, redelivery
backoff, expiration/TTL execution, FGAC, named schemas, quotas, load or latency,
network-failure exactly-once guarantees. Counter rollback demonstrates the
specific transaction pattern, not universal exactly-once external effects.
All transaction errors were propagated out of the Go transaction callback;
catching an assertion error and committing other work is a different case.
