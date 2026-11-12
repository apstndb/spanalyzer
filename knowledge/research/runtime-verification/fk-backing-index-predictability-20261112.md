---
type: Research Note
title: FK backing-index predictability, November 12
description: Official contracts, pinned Emulator source, cross-engine index metadata and write observations, and conservative mutation-count bounds.
tags: [spanner, spanner-omni, emulator, foreign-keys, indexes, mutation-count]
status: draft
sources:
  - id: overview
    resource: https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#backing-indexes
    title: FK backing-index ownership, sharing and enforcement
  - id: how-to
    resource: https://docs.cloud.google.com/spanner/docs/foreign-keys/how-to
    title: FK and backing-index INFORMATION_SCHEMA queries
  - id: information-schema
    resource: https://docs.cloud.google.com/spanner/docs/information-schema#indexes
    title: Physical index metadata and SPANNER_IS_MANAGED
  - id: omni-differences
    resource: https://docs.cloud.google.com/spanner-omni/differences
    title: Shared engine does not imply complete managed and Omni parity
  - id: documents
    resource: evidence/fk-backing-index-20261112/document-sources.json
    title: Full official document retrieval identities and hashes
  - id: source
    resource: evidence/fk-backing-index-20261112/source-audit.json
    title: Emulator v1.5.59 source pinned to d357ec1
  - id: receipt
    resource: evidence/fk-backing-index-20261112/receipt.json
    title: Independent runtime identities, input variants and lifecycle outcomes
  - id: plan
    resource: evidence/fk-backing-index-20261112/plan.json
    title: Eighteen FK families and 93 controlled steps
  - id: verifier
    resource: evidence/fk-backing-index-20261112/verify.py
    title: Offline checks of retained metadata, counts, rejections and identities
  - id: implementation
    resource: ../../../mutation_count_schema.go
    title: Conservative DDL-only FK index bounds
---

# FK backing-index predictability

FK index **requirements and shape can often be predicted from schema**, but a
complete physical catalog should come from `INFORMATION_SCHEMA`. A pinned
Emulator implementation exposes a deterministic generated-name recipe. The
reviewed official pages do not specify that recipe for managed Spanner or
Omni. Observed name agreement within a release or between two Omni releases
does not create a portable naming contract.

The practical model remains DDL bounds, then an observed physical index
snapshot, then old-row facts or a successful same-execution measurement. Do
not multiply the number of FK constraints by one universal coefficient: some
constraints need no index, several can share one, and a referenced index can
also affect writes to the parent table.

## Documented behavior

Full FK overview, how-to, INFORMATION_SCHEMA, GoogleSQL DDL and Omni differences
pages were retrieved with `dkcli` and read directly on November 12. Their
[identities and content hashes](evidence/fk-backing-index-20261112/document-sources.json)
remain retained; full downloads stay in task scratch.

- Enforced FKs may need a backing index on either side, up to two. Informational
  FKs do not create a referencing-side index and may need one referenced index.
- FK maintenance does not reuse an ordinary user-created index. Equal required
  backing indexes can be shared; dropping one of several constraints retains
  the shared index, while dropping its last owner removes it.
- A full referenced primary key generally avoids a referenced-side secondary
  index. A proper PK subset can still need a unique index.
- Backing indexes are service-managed. Query `SPANNER_IS_MANAGED`, key columns,
  uniqueness, NULL filtering and state instead of inferring them from a name.
- The overview permits using backing indexes in queries, including explicit
  `FORCE_INDEX`. It directs callers to discover names in INFORMATION_SCHEMA;
  it does not promise a hash algorithm or a constraint-name alias.

The documentation describes NULL-filtered backing indexes. Observed physical
metadata reports `IS_NULL_FILTERED=false` when every indexed key is NOT NULL,
and true when any is nullable. NULL filtering is redundant for the all-NOT-NULL
case; do not confuse this metadata optimization with different NULL semantics.

## Pinned implementation

The source audit resolves Emulator `v1.5.59` to
`d357ec1d2b08ae92ce1da4c5078de82fda0c07ab`.

| Source boundary | Source-backed behavior |
| --- | --- |
| [Referenced-key ordering](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/d357ec1d2b08ae92ce1da4c5078de82fda0c07ab/backend/schema/updater/schema_updater.cc#L3113) | Normalize matched leading referenced PK columns first, then remaining referenced columns in table declaration order. Apply the same permutation to the referencing columns |
| [Referencing PK coverage](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/d357ec1d2b08ae92ce1da4c5078de82fda0c07ab/backend/schema/updater/schema_updater.cc#L3160) | Nullable FK columns require an index even when they form a PK prefix. Nonnullable columns also need the required normalized order to fit the PK |
| [Physical construction and reuse](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/d357ec1d2b08ae92ce1da4c5078de82fda0c07ab/backend/schema/updater/schema_updater.cc#L3203) | Set uniqueness for the referenced side; set NULL filtering from key nullability; reuse the generated index and attach another managing constraint. Copy applicable source-PK direction and NULL ordering |
| [Fingerprint](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/d357ec1d2b08ae92ce1da4c5078de82fda0c07ab/backend/schema/updater/global_schema_names.cc#L49) | FarmHash Fingerprint64 over the fixed seed `2695C62E33338E4B`, underscore and full signature; uppercase 16-digit hexadecimal suffix |
| [Managed index name](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/d357ec1d2b08ae92ce1da4c5078de82fda0c07ab/backend/schema/updater/global_schema_names.cc#L125) | Use table/schema and canonical column names with `U` for unique, otherwise `N` for NULL-filtered, otherwise no code. Hash before truncation. The FK constraint name is absent from the signature |

This permits exact-name prediction for this pinned Emulator source. It is
implementation evidence, not a managed-service specification. The exact
upstream `T(X,Y)` / `U(A,B)` fixture produced the expected Emulator names
`IDX_T_X_Y_U_5AD6E41B495C5BB9` and `IDX_U_A_B_N_8C11B65ACA7F01B9`.
Omni and managed produced different suffixes from Emulator for the same
definitions. All tested Omni and managed names match each other.

A separate read-only query on the existing managed verification DB evaluated
`FARM_FINGERPRINT` for two known signatures, with and without the Emulator
seed. Seeded results reproduce the Emulator suffixes, not the observed
managed suffixes; the unseeded candidates also differ. Thus this difference
is not evidence of incompatible SQL FarmHash implementations. These four
[control rows](evidence/fk-backing-index-20261112/fingerprint-readonly.jsonl)
rule out two concrete recipes; they do not prove that managed names are random
or impossible to predict. A different deterministic seed/algorithm remains
unverified. No additional managed DB or data write was used for this control.

Each managed/Omni matrix run records 11 successful default-schema FK-name
`FORCE_INDEX` plans on the referencing side. On named-schema `NS.C`, however,
the unqualified `NamedFk` selector is rejected despite an existing backing
index; a schema-qualified constraint selector was not tested. The same FK
constraint name is not accepted on the referenced parent, even when a parent
backing index exists. Emulator rejects every tested constraint-name selector
on both sides. A referencing hint can also fail the NULL-filter safety check
when the query does not exclude every nullable key. These are separate
rejection reasons; use the physical discovered name for a portable catalog
identity.

## Runtime matrix

The [receipt](evidence/fk-backing-index-20261112/receipt.json) records each
execution separately. Pinned local images are Emulator 1.5.59 and Omni
2026.r4-lts.2 / 2026.r5.1, all linux/arm64, with actual container manifest
evidence. The controlled GoogleSQL matrix covers nullability, full/prefix PKs,
ordinary indexes, shared FKs, informational FKs, referenced non-PK uniqueness,
column permutations/declaration order, named schemas, long names and the
upstream naming fixture. Constraints and table names are synthetic.

| Case | Observed result |
| --- | --- |
| Referencing non-PK column | One managed referencing index; NULL filtering follows key nullability |
| NOT NULL referencing PK or leading PK prefix, unchanged referenced order | No referencing secondary index |
| Nullable referencing PK | A managed NULL-filtered index remains necessary |
| Existing ordinary index on the same key | Ordinary and managed indexes coexist |
| Two equal enforced FKs | One physical index; remains after the first drop, disappears after the last, and is recreated with the same name under a different constraint name |
| Informational FK referencing the full parent PK | No secondary index on either side; the non-PK informational variant is documented/source-backed but not a separate runtime case here |
| Referenced non-PK / proper PK subset | A unique managed index appears on the referenced table |
| Reordered full referenced PK | No referenced secondary index, but the referencing key order can change |
| Referenced columns absent from the PK | Canonicalization follows referenced table declaration order, not simply the FK clause order |
| Drop/recreate the identical table | Same generated name within each tested runtime |
| Managed versus LTS versus r5 | All 23 physical metadata/name row sets match exactly in this matrix |
| Emulator versus Omni | Index shapes match in the matrix; generated suffixes differ |

The fresh managed run recorded all 93 steps in one explicitly authorized
GoogleSQL DB. All 24 DDL and 23 metadata steps succeeded; all 23 metadata row
sets, including generated names, match both Omni releases exactly. The owned
DB was dropped successfully and a subsequent GetDatabase returned NotFound.
The rolling service build is unavailable; this is one dated observation, not
fleet-wide or future naming evidence.

PostgreSQL identifier normalization, descending/NULLS key order and deliberate
collisions with pre-created generated names are outside the runtime matrix.
Source-level direction handling above is not an additional runtime claim.

## Write controls and unexpected failures

The first two Omni attempts stopped before SQL because the harness enabled
instance auto-configuration, which single-server Omni rejects. The corrected
producer enables database-only configuration for Omni. Both initial failures
and successful runtime closure are retained separately.

In the broad schema-first matrix, both Omni releases returned `Internal` while
seeding parents after all FK constraints were created; six dependent DML
steps remained `NOT_RUN`. A separate 17-step fixture seeds parents first and
isolates nullable-PK and three-column permutation controls. The following
table combines those Omni controls with the parent-first 93-step Emulator
and managed runs:

| Control | Emulator 1.5.59 | Omni LTS / r5 and managed |
| --- | --- | --- |
| Insert `(Id,V)` through nullable referencing PK | Succeeds; CommitStats absent | Succeeds, mutation count 3 |
| Same columns with NOT NULL PK | Succeeds; CommitStats absent | Succeeds, mutation count 2 |
| Three-cycle canonical child PK `(CC,CA,CB)` | Extra referencing index; valid insert succeeds | Same index shape; valid insert count 4 |
| Three-cycle inverse child PK `(CB,CC,CA)` | No referencing index; valid insert incorrectly fails FK validation | No referencing index; valid and invalid insert controls return `Internal` |

The permutation is `FOREIGN KEY(CA,CB,CC) REFERENCES ParentTriple(B,C,A)`.
The parent key is `(A,B,C)=(1,2,3)` and valid child values are `(2,3,1)`.
The pinned Emulator maps each column to a permutation value when checking PK
coverage; for a three-cycle that differs from the inverse mapping needed by
the normalized key order. The source and runtime observations support this
Emulator diagnosis; no patched-Emulator intervention was run. Similar
managed/Omni symptoms do not establish their proprietary implementation or
prove the same internal cause.

A separate five-step Omni LTS control confirms a second undercount trigger:
`FOREIGN KEY(X,Y) REFERENCES OrderedParent(B,A)` on child `PRIMARY KEY(X,Y)`
needs a backing index ordered `(Y,X)`. Inserting `(X,Y,V)` costs 4, not 3.
Rejected transactions and absent CommitStats are never recorded as count zero.

## Analyzer integration and alternatives

| Approach | Decision |
| --- | --- |
| Copy the Emulator generated-name algorithm into the portable model | Reject: it does not predict observed Omni suffixes and has no reviewed managed naming contract |
| Require a live connection for every estimate | Reject: retain an offline conservative DDL model, with explicit observation freshness assumptions |
| Treat every FK as a distinct index with an exact price | Reject: PK coverage and sharing invalidate that formula |
| Predict exact physical layouts from the pinned implementation | Useful for source diagnosis; avoid treating implementation quirks as a portable success/constraint contract |
| Preserve conservative bounds and reconcile actual physical metadata | Chosen: count actual shared indexes once; use physical names and flags from the supplied snapshot |

`mutation_count_schema.go` previously ruled out a referencing backing index
from the written FK-column PK prefix alone. It now also checks NOT NULL and,
for composite keys, requires an unchanged referenced PK prefix before ruling
out reordering. Other composite layouts preserve an extra-index upper bound
until an observed snapshot is supplied. This intentionally permits some wider
intervals rather than inferring exact canonicalization from one engine.

Regression tests cover DML and Mutation API inputs, nullable versus NOT NULL
PKs, reversed referenced keys and snapshot refinement. This does not simulate
FK constraint success or claim that every successful count is service-verified.
Existing snapshot alias linkage remains partial when physical canonical key
order differs from the declared FK order; physical index costing does not
depend on that alias being available.

The standalone CLI and tools modules pin published root commit
`7c70c0d080540df41f82acd231e36959c50ba441`. Both pass their tests with
`GOWORK=off`. An independent consumer without replacement directives resolves
that same module version and confirms nullable-PK `[2,3]`, NOT NULL `[2,2]`
and reordered-reference `[3,4]` bounds; this is a local consumer check, not
another service measurement.

## Reproduction and retained evidence

Run the [offline verifier](evidence/fk-backing-index-20261112/verify.py) from
the repository root. It checks the retained observations without contacting a
service. To prepare the reproducer, without RPCs:

```sh
python3 knowledge/research/runtime-verification/evidence/fk-backing-index-20261112/prepare.py \
  .tmp/fk-backing-index-repro --variant parent-first
```

Preparation retrieves immutable producer bodies from local Git and applies
small retained patches. Use the recorded dependency revision and hashes for
the same SDK/runtime adapter boundary; a later dependency set is a different
experiment. Build `local.go` from the `tools` module and pass backend, a
platform-pinned image from `runtime_targets.json`, and the prepared plan path.
The local producer owns and closes its container. `controls`, `ordering` and
`schema-first` variants preserve the distinct case orders used here.

The managed producer requires an explicit lifecycle token and an exclusive
private resource journal. Preparation grants no cloud-write authorization.
Only this task's newly created managed DB may be deleted; unrelated databases
and the instance are outside cleanup scope. Full logs, local locators and
opaque request IDs are not retained. Compact result rows, errors, CommitStats,
runtime/source identities and the synthetic inputs remain sufficient to
assess and rerun these claims.
