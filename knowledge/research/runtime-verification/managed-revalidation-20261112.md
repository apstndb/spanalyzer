---
type: Research Note
title: Managed Spanner revalidation and parameterized views, November 12
description: Four documentation-mirror commits, the November 10 parameterized-views announcement, current managed behavior, and explicit harness corrections.
tags: [managed-spanner, documentation-diff, parameterized-views, secure-context, statistics, compression, runtime-verification]
status: draft
sources:
  - id: release-notes
    resource: https://docs.cloud.google.com/spanner/docs/release-notes
    title: Current official managed Spanner release notes
  - id: mirror
    resource: https://github.com/apstndb/spanner-docs-mirror/compare/fa8645afd32f27de73bd1fca7c29e89fbc027245...103e2193ab35b06695c2b518a9ae6a4c7d90116c
    title: Complete mirror range since the November 8 general investigation
  - id: audit
    resource: evidence/managed-recheck-20261112/documentation-audit.json
    title: Every new commit and changed documentation path with source hashes
  - id: assertions
    resource: evidence/managed-recheck-20261112/assertions.json
    title: Bounded runtime assertions and cleanup checks
  - id: readonly
    resource: evidence/managed-recheck-20261112/new-readonly.jsonl
    title: Secure context, compression and distribution probes
  - id: distributions
    resource: evidence/managed-recheck-20261112/distribution-frames.jsonl
    title: Nonempty aggregation and independently decoded Zstd frames
  - id: corrected-views
    resource: evidence/managed-recheck-20261112/google-view-repair.jsonl
    title: Qualified view predicates and original role-query retries in the same owned DB
  - id: owned-completion
    resource: evidence/managed-recheck-20261112/google-completion.jsonl
    title: Typed vectors, named-schema UDF prerequisites and index-state checks
  - id: udf-execution
    resource: evidence/managed-recheck-20261112/google-udf-check.jsonl
    title: UDF creation versus execution and context argument forwarding
  - id: lifecycle
    resource: evidence/managed-recheck-20261112/receipt.json
    title: Producer identities, exact observation scope and the three authorized owned database lifecycles
  - id: pg-native-controls
    resource: evidence/managed-recheck-20261112/pg-final.jsonl
    title: Separately authorized native PostgreSQL index and view-role controls
  - id: pg-native-assertions
    resource: evidence/managed-recheck-20261112/pg-final-assertions.json
    title: All 17 PostgreSQL controls, expected rejections and owned cleanup
  - id: view-overview
    resource: https://docs.cloud.google.com/spanner/docs/parameterized-views
    title: Parameterized views and documented limitations
  - id: view-tutorial
    resource: https://docs.cloud.google.com/spanner/docs/use-parameterized-views
    title: Parameterized-view tutorial and client examples
  - id: cast
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/conversion_functions#cast
    title: Formal CAST syntax
  - id: local-frontend
    resource: evidence/managed-recheck-20261112/local-analyzer-functions.json
    title: Current local analyzer function coverage, distinct from service support
  - id: live-html
    resource: evidence/managed-recheck-20261112/live-html-checks.json
    title: Direct current HTML checks for the CAST examples and system-role overview
---

# Managed verification on November 12, 2026

The code baseline is `8ba3f9539ed2cecc7820f92dde77bcaeb6c6a4e8`.
The general managed baseline is [November 8](managed-revalidation-20261108.md);
the later November 10 investigations primarily cover write accounting.
Observations use the existing dedicated managed profile, Go 1.26.11 and the
Spanner Go SDK v1.95.1. Connection and request identifiers are redacted.
The service does not expose a backend build identity through these probes:
timestamps identify this observation, not a product version or rollout date.

## Release notes and complete mirror comparison

Both the current official HTML and the full Developer Knowledge document list
the **November 10 parameterized-views GA announcement** as the latest entry. The preceding
release heading remains October 30. The mirror head is
`103e2193ab35b06695c2b518a9ae6a4c7d90116c`, captured November 11 at
18:01:37 UTC, or November 12 JST. A capture date does not change the announced
release date or establish when a feature reached this instance.

Every intervening commit and all 24 documentation paths were inspected. The
newly fetched Developer Knowledge copies of all 24 match their mirrored
bodies, ignoring trailing whitespace. Current live HTML was independently
checked for the release notes, two view pages and the system-role overview;
the CAST typo and missing reader-role entry persist there too. URL logs,
failure logs and synchronization metadata are accounted for separately.

| Commit, UTC capture date | Disposition |
| --- | --- |
| `a17f48f39452170f51ab9b8d3cf7b9282f214485`, November 8 | Synchronization metadata only. |
| `09d56518462a14ac31db4f4af1f244b83b7b122c`, November 9 | TTL documentation link fix and temporary removal of the schema-design page. |
| `e8560663f58b619f3d8f7defc1340ee7586e4d73`, November 10 | Restore the identical schema-design body; clarify vector dimensions, compression and custom-span statistics; update Java examples and formatting. |
| `103e2193ab35b06695c2b518a9ae6a4c7d90116c`, November 11 | Parameterized views and the GA entry; scorer creation cohorts; Studio workflow and CMEK re-encryption guidance. |

The schema-design removal/restoration has no net content change. It is not an
engine transition. Java examples now name 6.127.0; no Java client is installed
or exercised here. Organization-policy text changes the condition limit to
1,500 characters. CMEK text distinguishes the initial configuration update
from background re-encryption and retained backup keys. These administrative
and console changes are reviewed documentation, not tested SQL behavior;
no instance, policy, encryption key, remote model or console resource is changed.

## Existing managed surface

The 135 prior read-only cases retain their status classifications. Complete
metadata tuples, including raw types and ordinals, remain **48 tables / 310
columns** for INFORMATION_SCHEMA and **52 / 624** for SPANNER_SYS.
Three INFORMATION_SCHEMA observations, the SPANNER_SYS comparison and the
required drift/canonical tests pass. Both capture comparison reports say
`producer_changed_only`, with `material_change=false`; their producer hashes
changed, while observed surfaces did not. Existing canonical captures and
projection selectors are preserved.

The canonical DDL comparison is against the existing empty user schema,
not proof that every new view or DDL family round-trips. Previously retained
`RAND()` and selected zero-row `LOCK_STATS_TOTAL_10MINUTE` limitations persist.
Successful bounded statistics projections are separate from those failures.

There are 184 SDK read-only case records, including new controls, plus three
separate CLI syntax controls. Expected negative results remain explicit;
these counts do not mean every query returns OK.

## Secure context and parameterized views

The existing SDK already exposes `QueryOptions.ClientContext` and
`RequestOptions_ClientContext.SecureContext`; no dependency upgrade is needed
to send synthetic context values. These are request options, independent of
ordinary SQL query parameters.

`SECURE_CONTEXT('user_id')` returns STRING values supplied with the request.
An explicit NULL value returns SQL NULL; a missing key produces
`InvalidArgument`. Numeric/object values are rejected. A query parameter in
place of the function's literal key is rejected semantically, even though the
generic function-call syntax parses.

The corrected definer view uses qualified table references:

```sql
CREATE VIEW PVFilter SQL SECURITY DEFINER AS
SELECT r.Id, r.Tenant, r.Payload
FROM PVRows AS r
WHERE r.Tenant = SECURE_CONTEXT('tenant');
```

The view-only `pv_user` role receives IDs 1/3 for tenant `a`, ID 2 for tenant
`b`, and no rows for unknown or NULL tenants. A missing or incorrectly typed
context is rejected. An ordinary parameter and `OR TRUE` outside the view do
not bypass its tenant filter. Direct base-table reads and direct calls to
`SECURE_CONTEXT()` by that role are denied. A JSON string context also filters
correctly through `PARSE_JSON` and `JSON_VALUE`.

A separate role explicitly granted `spanner_secure_context_reader` can call
the function directly but still cannot read the base table. A deliberately
projected context value can be returned by a definer view: view authors still
control what that view discloses. No claim is made that arbitrary view
definitions conceal every context value or form a security barrier.

An invoker view can be created, but the view-only caller's use of secure
context is denied. The definer requirement is demonstrated by the actual
restricted-role queries, not inferred solely from DDL acceptance.

The separately authorized PostgreSQL follow-up also verifies a definer view
using `spanner.secure_context('tenant')`. The view-only role receives ID 1
for tenant `a` and ID 2 for tenant `b`. Missing context is rejected, and direct
base-table and secure-context function reads are denied. The administrative
positive control still reads its supplied context value. The PostgreSQL DDL
export normalizes view text and emits its SELECT grant as `ON TABLE`; the
restricted-role query results provide the security evidence.

Generated-column and column-default uses are rejected during DDL. A
named-schema UDF containing the function can be created, but direct calls and
calls through a definer view fail with an internal
`secure_context_parameter_...` binding error. Creation alone does not prove
executable support. Passing `SECURE_CONTEXT('tenant')` as an argument to an
ordinary UDF succeeds. This gives a working interpretation of the overview's
UDF best practice while respecting its stated prohibition on calling the
function inside a UDF. The default-schema UDF rejection is a separate schema
prerequisite, not evidence of this secure-context limitation.

## Documentation defects and parser boundary

The new overview, creation example and tutorial use a comma-style cast:

```sql
-- Rejected by managed Spanner and strict memefish v0.8.1:
SELECT CAST(SECURE_CONTEXT('user_id'), INT64);

-- Accepted and executed with synthetic context user_id = '123':
SELECT CAST(SECURE_CONTEXT('user_id') AS INT64);
```

The formal Spanner CAST reference, pinned GoogleSQL grammar/AST/unparser and
strict local parser all require `AS`. The managed CLI probes bypass memefish;
the corrected form without a context reaches a missing-parameter error,
whereas the comma form is rejected by the service parser. Separate SDK
execution with context returns INT64 123. No parser relaxation is warranted.

The tutorial creates `checked_items` and `secure_checked_items`, but its
application examples query `MyUserData`/`myuserdata` and expect unrelated
columns. This is a source-backed example-consistency defect; those client
programs are not all executed. The separate system-role overview also omits
`spanner_secure_context_reader`, despite its mention on the new view pages and
the successful managed grant/control here. No upstream report is posted.

At this analyzer baseline, `SECURE_CONTEXT` and
`SPANNER_SYS.DISTRIBUTION_AGG` both produce local function-not-found errors.
The service accepts them. Compression analysis still succeeds. These are
local catalog gaps, separate from memefish's generic call/CAST grammar and
from the Go client's existing request-option support. Follow-up work should
register a STRING-returning secure-context function and a true aggregate with
its supported overloads; it must not model the aggregate as an ordinary scalar
or turn unsupported runtime restrictions into parser syntax rules.

## Compression and distribution aggregation

Managed accepts both positional and named compression levels in the documented
-5 through 22 range; -6 and 23 are rejected. A NULL level or NULL decompression
size limit returns NULL. Decompression size limits remain named-only, and a
four-byte limit rejects the five-byte payload.

Two independently decoded valid Zstd frames contain the same `hello` payload.
The frame with content size in its header decompresses to STRING/BYTES; the
streaming frame without that size is rejected. This matches the clarified
input restriction, rather than proving the latter frame is invalid Zstd.

`DISTRIBUTION_AGG` is exercised with a previously observed service-encoded
histogram, not only an empty statistics table. Two equal inputs yield COUNT=2;
one input plus NULL yields COUNT=1, with the same p99. All-NULL aggregation
returns NULL and a guard prevents the percentile call on that NULL result.
Malformed histogram JSON is rejected. A complete distribution ARRAY is not an
accepted argument, and a returned aggregate STRUCT must be flattened before
it is a result column. Current documented query/transaction shapes are tested
separately; empty live samples are not invented percentile measurements.

## Owned fixtures, corrections and boundaries

The user authorized one GoogleSQL and one PostgreSQL disposable DB for the
prepared 235 steps (152/83). Only this run's successfully created databases
may be deleted. Exclusive lifecycle markers and approved input hashes prevent
accidental replacement lifecycles. The base database is read-only throughout.

The initial GoogleSQL view predicates lacked required qualification in strict
name-resolution mode. Initial vector literals inferred FLOAT64, and a cast
between FLOAT64 and FLOAT32 arrays is not supported. The corrected controls
use `ARRAY<FLOAT32>[...]`. A named schema and ordinary UDF positive control
separate a schema prerequisite from the secure-context restriction.

All corrections reuse the same owned GoogleSQL DB. The original owner was
paused while supplemental producers, which contain no create/drop API calls,
verified the unique current-run prefix/time/schema binding and then the same
database-identity hash. The owner wait expired during that deliberate pause;
its `DeadlineExceeded` is a harness interruption, not an engine regression.
The original wrapper stopped on its transport guard. Completed corrections
and index `READ_WRITE` state are retained separately, and the owner then
deleted the GoogleSQL DB and independently observed `NotFound`.

The initial PostgreSQL fixture records all 83 steps and is also deleted with
`NotFound` verified. Two scorer-default-dependent cases remain blocked by the
known unsupported `spanner.score_version` database option. Explicit scorer
versions and six-row pagination still work. The complete three-argument queue
receiver repeats the earlier repaired NULL/default and explicit-20 behavior;
omitting the batch size still fails resolution.

PostgreSQL `spanner.secure_context('tenant')` returns the supplied string;
the unqualified function does not resolve. Both documented statistics examples
execute, including populated query and transaction percentile rows in this
fixture. Its annotated/unannotated vector storage and KNN controls succeed.

The two initial PostgreSQL index probes mistakenly use GoogleSQL's
`CREATE VECTOR INDEX` and fail at the `VECTOR` keyword. They do not test
PostgreSQL index support. A separately authorized additional PostgreSQL DB
completes all 17 corrected controls. The native positive form succeeds:

```sql
CREATE INDEX FinalAnnotatedIndex ON FinalAnnotated
USING scann(e) WITH (distance_type='EUCLIDEAN')
WHERE e IS NOT NULL;
```

The equivalent index on an unannotated column fails with `FailedPrecondition`
requiring `vector_length`. View-only-role filtering also succeeds as described
above. The additional producer exits 0, and its DB is deleted with `NotFound`
verified. All three authorized DB lifecycles are complete; no current owned DB
remains. Original wrong-dialect results are preserved alongside this correction.

The completed investigation has 482 passing bounded assertions, including 38
for the additional 17-step fixture and its lifecycle. Expected rejections,
initial blocked cases, same-DB corrections and the deliberate owner-pause
timeout retain their own evidence. This count does not turn every initial
statement into a successful or attempted API call.

In the corrected GoogleSQL fixture, with the annotated index already present,
length 3 is accepted and length 2 is rejected; an unannotated column stores
both lengths. Both execute the bounded KNN positive control. The annotated
index is `READ_WRITE`; creating one without the dimension annotation fails.
PostgreSQL independently rejects a length-2 write to the annotated column
before any valid index is created. Longer-than-annotated vectors are not
tested. These are compatibility controls, not a KNN latency or ANN recall
benchmark.

The scorer documentation clarifies creation cohorts. Both initial new DBs'
unpinned scores match explicit version 2 and differ from version 1. This tests
the post-April cohort, but cannot establish historical pre-April defaults, an
April rollout boundary, or an actual default migration.
Original errors, blocked cases and correction results remain distinct; no
all-green assertion is inferred from a producer exiting successfully.
