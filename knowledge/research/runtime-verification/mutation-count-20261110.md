---
type: Research Note
title: Mutation counts on Omni LTS.2 and Emulator, November 10
description: Eighty-five synthetic operations per engine distinguish index deltas, automatic-column API differences, buffered updates, missing statistics and unenforced local quota limits.
tags: [spanner-omni, emulator, mutation-count, dml, runtime-verification]
status: draft
sources:
  - id: receipt
    resource: evidence/mutation-count-20261110/receipt.json
    title: Image pins, execution windows, dependency scope and evidence hashes
  - id: omni
    resource: ../../../tools/mutation-count-lab/testdata/omni.json.gz
    title: Complete primary Omni response and wire-mutation capture
  - id: emulator
    resource: ../../../tools/mutation-count-lab/testdata/emulator.json.gz
    title: Complete primary Emulator response and wire-mutation capture
  - id: probe
    resource: ../../../tools/mutation-count-lab/probe/main.go
    title: Owned local runtime fixture, seeds and actual CommitStats requests
  - id: comparison
    resource: evidence/mutation-count-20261110/comparison-summary.json
    title: Derived model scores; raw observations remain authoritative
  - id: documents
    resource: evidence/mutation-count-20261110/document-sources.json
    title: Retrieved official-source metadata and content hashes
  - id: design
    resource: ../write-analysis/mutation-count-alternatives-20261110.md
    title: Implemented alternatives, public-service boundary and integration proposal
---

# Local mutation-count verification

Observed November 10, 2026 JST. This is first focused coverage of write
accounting on the selected images, not a new engine-version transition.
Both owned local containers complete 85 operation attempts: 84 successful
commits and one expected foreign-key violation. The retained JSON includes
the fixture DDL, before-state labels, DML row counts, captured wire mutations,
statuses and nullable CommitStats.[^omni][^emulator]

## Identity and method

| Runtime | Configured linux/arm64 image manifest |
| --- | --- |
| Spanner Omni `2026.r4-lts.2` | `sha256:b607b10bee3c1903f0a7f88f99bece67cb0dd9d9a028aee64226fc311bd78dae` |
| Emulator `1.5.59` | `sha256:969af6a9349948fbfb4a826c18e160e258a4005f7c759f8e7f2dcec369cccba8` |

The Go producer runs on darwin/arm64 with Spanner SDK `v1.95.1` and
spanemuboost `v0.4.10`. Host platform in the receipt is not the Linux container
platform. The image reference is configured execution identity, not a separate
server-version response. Optimizer version is not independently pinned or
observed in this write-only experiment.[^receipt]

The probe calls `spanemuboost.Run` to start an owned container, provisions a
random database in that runtime and seeds rows outside the measured
transaction. It uses `ReadWriteTransactionWithOptions` with
`ReturnCommitStats: true`; a unary interceptor records only protobuf Mutation
objects from the CommitRequest. DML strings and affected-row counts are
retained separately. No cloud project, managed endpoint, credential or
application payload is captured. The two final runtime container IDs are
absent from the post-run Docker inventory; unrelated containers are untouched.
Earlier setup/metadata-decoding attempts are local tooling failures, not engine
syntax findings. Only the final complete runs are used below.[^probe][^receipt]

## Base and index effects

Unless stated otherwise, tables have four columns `(Shard, Id, Active, Note)`,
PK `(Shard, Id)`. One-index fixtures index Active; the two-index fixture also
indexes Note. Stored fixtures add `STORING(Note)` to the Active index.

| Successful operation | Omni mutation_count |
| --- | ---: |
| Full insert, no / one / two ordinary indexes | 4 / 5 / 6 |
| Full insert, one `(Active,Note)` composite index | 5 |
| Full insert, one Active index storing Note | 6 |
| PK-only insert, two ordinary indexes / one NULL_FILTERED stored index | 4 / 2 |
| Change Note, no index / Active-only index / Note index | 3 / 3 / 5 |
| Write unchanged Note, Note index / Active index storing Note | 3 / 5 |
| Change Active, ordinary Active index / stored Active index | 5 / 6 |
| Write unchanged Active, either ordinary or stored Active index | 3 |
| NULL_FILTERED stored index, Active NULL to value / value to NULL | 5 / 4 |
| Existing-row full upsert or replace with both indexed values changed | 8 |
| Existing-row PK-only upsert / replace, two indexes | 2 / 6 |

The composite index is an important control: insertion adds one index-entry
unit, not one unit per declared index-key column. A changed indexed key removes
the old entry and inserts the new entry; an unchanged key can avoid that
maintenance. Writing a stored cell has a different observed accounting path
even when its value is unchanged. These statements are local observations,
not a latency, byte, lock or managed-service-cost model.[^omni]

## Automatic columns and API boundaries

The generated fixture adds `G AS (Id+1) STORED`. The default fixture gives Note
a constant default. The input PK has two columns.

| Insert fixture | Mutation API, full four explicit columns | Mutation API, PK only | DML, PK only |
| --- | ---: | ---: | ---: |
| No automatic columns | 4 | 2 | 2 |
| Default Note | 4 | 5 | 3 |
| Stored generated G | 7 | 5 | 3 |
| Stored G and default Note | 8 | 6 | 4 |

Thus the generated/default adjustment cannot be copied unqualified from one
API to the other. Full input to the default-only table does not pay the same
extra as input omitting that default. The combined generated/default fixture
has another local difference. The implementation records this as an
Omni-specific calibration and keeps the generic interval conditional; the
official documentation boundary is discussed in the design note. The tested
updates affect Active or Note; G depends only on immutable Id, so these cases
do not establish general generated-column update costs.[^omni][^design]

Two buffered mutations updating the same indexed row `a -> b -> a` produce
**6**. Two DML UPDATE statements performing the same transitions produce
**10** in cumulative CommitStats. Adding independent per-operation index
estimates would overcount the first case. This does not establish coalescing
rules for arbitrary mixed insert/update/replace/delete sequences.[^omni]

## Deletes and foreign-key indexes

| Case | Omni mutation_count |
| --- | ---: |
| Delete missing key, with zero/two/stored/NULL_FILTERED indexes | 1 in all four cases |
| Delete one range containing three rows, no / two ordinary indexes | 1 / 7 |
| Same three-row range, one stored index / one NULL_FILTERED stored index, all keys present | 4 / 4 |
| Delete one interleaved parent with three indexed child rows | 4 |
| Full insert with enforced FK to referenced PK | 5 |
| Same-column two enforced FKs sharing a backing index | 5 |
| Informational FK to referenced PK | 4 |

The range's base effect is one, and deleted stored index entries cost one each,
not their insertion cost. Interleaved cascade child base rows do not add one
unit per child in this case, but child secondary-index entries do.[^omni]

The metadata query observes **one**, not two, managed referencing indexes for
the two same-column constraints. An additional fixture referencing a non-PK
Alias column creates a managed referenced-side unique index and a managed
referencing-side index on both engines. The informational FK to the referenced
PK creates neither a referencing index nor an unnecessary referenced index.
Generated physical suffixes differ between Omni and Emulator. Constraint
aliases in the lab are inferred from unique managed key-prefix matches;
physical existence/columns are observed. Exact generated naming and arbitrary
constraint association are not inferred from these fixtures. This experiment
does not re-run FORCE_INDEX planning or prove the user-reported access_paths
metadata gap fixed in the ordinary PLAN collector.[^omni][^emulator]

## Statistics and limit controls

Omni returns CommitStats for all 84 successful operations. Emulator returns
no CommitStats for any of its 84 successful operations. The missing field is
retained as `null` with `stats_present: false`, rather than zero. Both engines
return `FailedPrecondition` for the invalid-reference control; failed writes
are excluded from numerical scores.[^omni][^emulator]

Both local engines accept **20,001 inserts of four columns**, a Mutation API
commit whose Omni statistic is **80,004**. Passing a local-container test does
not validate enforcement of the managed mutation limit. No managed write,
80,000-boundary enforcement test on managed Spanner, BatchWrite group test,
Partitioned DML test, commit-byte measurement, Java SDK test or latency
benchmark is performed here.[^omni][^emulator]

The [later managed observation](mutation-count-managed-20261110.md) retains
its own lifecycle, queue cases and primary capture. It rejects that 80,004-unit
Mutation API control. This follow-up does not rewrite the local observations
or fill the unexecuted DML/byte/BatchWrite boundaries of this experiment.

## Replay and interpretation

From `tools`:

```sh
go test ./mutation-count-lab
go run ./mutation-count-lab -compare mutation-count-lab/testdata/omni.json.gz
go run ./mutation-count-lab -compare mutation-count-lab/testdata/omni.json.gz -index-source ddl
go run ./mutation-count-lab -compare mutation-count-lab/testdata/emulator.json.gz
```

These commands read retained evidence without a database. To deliberately
start a new owned local runtime and perform writes:

```sh
go run ./mutation-count-lab/probe -backend omni -out /absolute/path/omni.json
go run ./mutation-count-lab/probe -backend emulator -out /absolute/path/emulator.json
```

The probe resolves current repository pins for a new run; the retained
receipts preserve this run's exact digests independently of later pin changes.
`-metadata-only` collects the same schema's index snapshot without the 85
measured operations. Tests check comparison coverage, NULL transitions,
automatic-column boundaries, same-key composition, statement aggregation,
wire validation, missing statistics, overflow and cascade uncertainty. Their
success validates this implementation against the retained calibration, not a
general public-service counting law.[^comparison][^design]

[^receipt]: [Execution receipt](evidence/mutation-count-20261110/receipt.json).
[^omni]: [Primary Omni capture](../../../tools/mutation-count-lab/testdata/omni.json.gz).
[^emulator]: [Primary Emulator capture](../../../tools/mutation-count-lab/testdata/emulator.json.gz).
[^probe]: [Probe source](../../../tools/mutation-count-lab/probe/main.go).
[^comparison]: [Derived comparison](evidence/mutation-count-20261110/comparison-summary.json).
[^design]: [Design alternatives and feedback dispositions](../write-analysis/mutation-count-alternatives-20261110.md).
