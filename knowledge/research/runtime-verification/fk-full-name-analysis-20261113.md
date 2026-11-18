---
type: Research Note
title: Managed and Omni FK full names, qualification and fingerprint hypotheses
description: FK backing-index name collisions across underscore-separated identifiers, managed and Omni observations, qualification controls and bounded hash-family rejection.
tags: [spanner, spanner-omni, foreign-keys, indexes, naming, fingerprint]
status: draft
sources:
  - id: official-how-to
    resource: https://docs.cloud.google.com/spanner/docs/foreign-keys/how-to
    title: Schema-scoped constraint names, generated names and metadata discovery
  - id: official-overview
    resource: https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#backing-indexes
    title: Service-managed backing indexes, sharing and ownership
  - id: official-metadata
    resource: https://docs.cloud.google.com/spanner/docs/information-schema
    title: Separate constraint, table and index schema/name fields
  - id: previous
    resource: fk-backing-index-predictability-20261112.md
    title: November 12 managed and cross-engine corpus and Emulator source
  - id: emulator-source
    resource: https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/d357ec1d2b08ae92ce1da4c5078de82fda0c07ab/backend/schema/updater/global_schema_names.cc
    title: Public Emulator name assembly and seeded fingerprint implementation
  - id: farmhash-source
    resource: https://github.com/google/farmhash/blob/84fead30a27d84830dd8edebe70db3c444cac5f6/src/farmhash.cc
    title: Fingerprint64 and HashLen17to32 used by the bounded symbolic model
  - id: observations
    resource: evidence/fk-name-analysis-20261113/observations.json
    title: Runtime manifests, structural controls, errors and document identities
  - id: controls
    resource: evidence/fk-name-analysis-20261113/prepare.py
    title: RPC-free preparation of pinned isolated reproductions
  - id: proof
    resource: evidence/fk-name-analysis-20261113/check_prefix.py
    title: Independently checked native controls and bounded symbolic exclusion
  - id: practical-follow-up
    resource: evidence/fk-name-analysis-20261113/practical-observations.json
    title: Natural UPPER_SNAKE_CASE order and shipping collision plus nullability control
  - id: lowercase-follow-up
    resource: evidence/fk-name-analysis-20261113/lowercase-observations.json
    title: The same lower_snake_case collision in both creation orders
  - id: managed-follow-up
    resource: evidence/fk-name-analysis-20261113/managed-observations.json
    title: Fresh managed lower_snake_case collision, physical names and verified cleanup
  - id: managed-controls
    resource: evidence/fk-name-analysis-20261113/managed-controls.json
    title: Exact eleven-step managed input including the optional inventory-query error
  - id: case-matrix
    resource: evidence/fk-name-analysis-20261113/case-matrix-observations.json
    title: Fresh uppercase and lowercase controls across managed, Omni and Emulator
  - id: case-matrix-controls
    resource: evidence/fk-name-analysis-20261113/case-matrix-controls.json
    title: Identical 32-step schema-only input for each of the three engines
---

# FK full-name analysis, November 13

The visible structure and schema qualification can be explained, but **the
managed/Omni 16-hex-digit suffix function is not identified**. Reusing the
Emulator's name algorithm would predict the wrong production names. The new
controls narrow plausible inputs and reject a stronger family than the two
concrete seeded/unseeded recipes rejected in the earlier investigation.

The retained November 12 managed corpus contains **18 distinct physical
index names**, not 23 distinct names. The 23 metadata result sets include
repeats and empty sets. Those 18 complete names match both then-tested Omni
releases. November 13 JST structural controls ran on pinned Omni LTS.2 and
r5.1, linux/arm64. A separate fresh managed follow-up reproduces the natural
lower_snake_case collision below, with the same physical name as Omni LTS.2.
It does not revalidate the entire earlier corpus or the other structural
controls on managed Spanner. A later uniform matrix below extends both
identifier cases to all three engines, including Emulator.

## Three identities to keep separate

| Object | Catalog identity | Observed example |
| --- | --- | --- |
| FK constraint | `CONSTRAINT_SCHEMA`, `CONSTRAINT_NAME` | `S.SameFk` |
| Referencing backing index | `TABLE_SCHEMA`, `INDEX_NAME` | `S.IDX_Same_Owner_N_FA7F931463D537A2` |
| Referenced primary/unique constraint | `UNIQUE_CONSTRAINT_SCHEMA`, `UNIQUE_CONSTRAINT_NAME` in REFERENTIAL_CONSTRAINTS | Discover independently; it is not the referencing FK name |

The default schema is the empty string. Named-schema catalog rows return the
local name separately from the schema; do not interpret `INDEX_NAME` alone as
a database-wide identity. The how-to documents schema-scoped uniqueness and
generated unnamed FK constraints. It directs callers to discover those names;
it does not document a production fingerprint function.

For automatic default-schema constraint names, fresh Omni controls show the
shape `FK_<referencing table>_<referenced table>_<16 hex>_<sequence>`. Two
different FK column sets on the same long table reference P and receive the
same fingerprint with `_1` and `_2`. The index names for those column sets
are different. An automatic constraint name is therefore not an index name.

For indexes, the visible shape is a length-limited
`IDX_<local table>_<ordered key columns>[_U or _N]_<16 hex>`. U takes precedence
over N in the observed unique cases. This is an observed construction pattern,
not a public naming contract or a recovered suffix formula. In particular,
the Emulator-style candidate signature `S.IDX_Same_Owner_N` is a hypothesis
input, not a known proprietary implementation detail.

## What the fresh structural controls establish

| Changed input | Omni LTS.2 / r5.1 result | Bounded inference |
| --- | --- | --- |
| Different random databases | All 48 common outcomes and metadata agree; r5 also agrees | Database identity does not affect these tested names |
| Referenced P to Q, and FirstFk to SecondFk | RelationCase backing name unchanged | The combined change leaves this example unchanged; the earlier sharing control separately varied constraint names |
| Owner INT64 to STRING, retaining identifier/nullability | TypeCase backing name unchanged | The tested type change is not distinguished |
| Nullable PK Id ASC to DESC | Same DirectionCase name; metadata direction changes | The tested key direction is not distinguished |
| Referenced unique Alt nullable to NOT NULL | Same `IDX_UniqueParent_Alt_U_95ED9085F284257F`; NULL-filter flag changes | Unique naming does not distinguish this NULL-filter difference |
| CaseDemo/Owner to casedemo/owner | Suffix `E0B160AAE0445CCF` to `4272DC64F3E2B112` | Identifier case affects these names |
| S.Same to T.Same | Local visible prefix agrees; suffix `FA7F931463D537A2` versus `CA5457272264F14A` | Schema affects these names |
| Long column suffix A versus B beyond visible truncation | Distinct suffixes `75085515A05F2BED` / `3129FE930F70FAD1` | Hash input retains information absent from the truncated visible base |

The long names are 127 bytes, not exactly 128: truncating the visible base at
the identifier limit and removing a trailing separator can leave one byte
unused. This agrees with the public Emulator assembly rule; it is not a
promise that every generated name fills the limit.

Four owned runtimes hosted five random databases and executed **160 plan
steps**: two 48-step LTS repeats, a five-step reverse-order control, 50 r5
steps, and nine LTS qualification steps. Runtime closure and owned-container
absence were checked. Full routine logs are local; the compact receipt retains
metadata, relevant failures, identities and equality checks across runs.

## Reproduced full-name collision

Both definitions independently generate
`IDX_A_B_C_N_99E878FAF8057D7D`:

```sql
CREATE TABLE P(K INT64 NOT NULL) PRIMARY KEY(K);
CREATE TABLE A_B(
  Id INT64 NOT NULL, C INT64,
  CONSTRAINT ExplicitAB FOREIGN KEY(C) REFERENCES P(K)
) PRIMARY KEY(Id);
CREATE TABLE A(
  Id INT64 NOT NULL, B_C INT64,
  CONSTRAINT ExplicitA FOREIGN KEY(B_C) REFERENCES P(K)
) PRIMARY KEY(Id);
```

The first child can be created. Creating the second fails with
`FailedPrecondition` and an empty description. Reversing creation order in a
fresh LTS database reverses which child succeeds, with exactly the same full
index name. Distinct explicit constraint names rule out simply reusing a
constraint name. The original order also fails on r5.

This reproduces a naming-collision failure and supports a flattened underscore
signature interpretation: `A(B_C)` and `A_B(C)` both flatten to `A_B_C`, losing
the table/column boundary. The later uniform matrix reproduces the natural
lower_snake_case and UPPER_SNAKE_CASE failures on both managed and Omni.
Emulator accepts both definitions in both cases. The internal cause and hash input remain
inferred; no proprietary implementation was inspected. These artificial
identifiers were tested on Omni; the natural lowercase example below was
also tested on managed Spanner. No upstream issue was posted.

### Practical UPPER_SNAKE_CASE follow-up

The failure is not confined to short artificial identifiers. An order can
store its requested shipping address while a shipping record stores its
actual delivery address. These natural table/column combinations flatten to
the same name:

| Table | FK column | Primary key in the control |
| --- | --- | --- |
| SALES_ORDER_SHIPPING | ADDRESS_ID | SHIPMENT_ID |
| SALES_ORDER | SHIPPING_ADDRESS_ID | ORDER_ID |

Both address columns are INT64 NOT NULL and reference ADDRESS(ADDRESS_ID).
Both are non-PK columns and require a nonunique, non-NULL-filtered backing
index. Both independently generate the complete name
`IDX_SALES_ORDER_SHIPPING_ADDRESS_ID_C53AEFD47826CE09`.

Fresh Omni LTS.2 controls reproduce failure of the second CREATE TABLE in
both creation orders, despite distinct explicit FK constraint names. This
can block a schema deployment. Distinct constraint names do not avoid the
physical-name collision in these controls.

As a negative control, making only SALES_ORDER.SHIPPING_ADDRESS_ID nullable
creates a NULL-filtered index named
`IDX_SALES_ORDER_SHIPPING_ADDRESS_ID_N_A414555C5DC900A2`. Both tables then
succeed. This distinguishes the index-kind prerequisite; it is not a
recommendation to weaken data constraints as a workaround. A NOT NULL PK
that needs no extra referencing index also does not trigger this collision.

This additional run used three random databases in one owned LTS runtime,
with 15 steps. That initial uppercase run was Omni-only; the later uniform
matrix below extends these identifiers to managed and Emulator, while r5
remains untested for these concrete identifiers. Use `practical-forward`, `practical-reverse` and
`practical-different-nullability` preparation variants to reproduce them.

Lowercasing the same identifiers does not avoid the failure. A further
owned LTS.2 run uses two random databases and ten steps to compare
`sales_order_shipping(address_id)` with `sales_order(shipping_address_id)`.
Both independently generate
`IDX_sales_order_shipping_address_id_90E532EDA843ECD5`, and the second CREATE
TABLE fails in both creation orders. The suffix differs from the uppercase
case, while the collision within the lowercase case remains. Use
`practical-lower-forward` and `practical-lower-reverse` to reproduce this
separate Omni observation. The managed follow-up below tests the same
lowercase identifiers; r5 was not freshly probed with these identifiers.

### Natural lower_snake_case reproduction DDL

The requested shipping address belongs to an order; the actual address also
belongs to its shipping record. The two FK columns are outside their tables'
primary keys. The explicit constraint names are different.

Run the following statements separately in a disposable GoogleSQL database:

```sql
CREATE TABLE address (
  address_id INT64 NOT NULL
) PRIMARY KEY (address_id);

CREATE TABLE sales_order_shipping (
  shipment_id INT64 NOT NULL,
  address_id INT64 NOT NULL,
  CONSTRAINT shipping_address_fk
    FOREIGN KEY (address_id) REFERENCES address (address_id)
) PRIMARY KEY (shipment_id);

CREATE TABLE sales_order (
  order_id INT64 NOT NULL,
  shipping_address_id INT64 NOT NULL,
  CONSTRAINT order_address_fk
    FOREIGN KEY (shipping_address_id) REFERENCES address (address_id)
) PRIMARY KEY (order_id);
```

To test the reverse order, drop the successfully created sales_order_shipping
table, keep address, create sales_order first, then create sales_order_shipping.
Inspect the generated physical name after each successful child definition:

```sql
SELECT TABLE_SCHEMA, TABLE_NAME, INDEX_NAME, IS_UNIQUE, IS_NULL_FILTERED,
       SPANNER_IS_MANAGED
FROM INFORMATION_SCHEMA.INDEXES
WHERE INDEX_TYPE = 'INDEX'
ORDER BY TABLE_NAME, INDEX_NAME;
```

### Fresh managed follow-up

On **2026-11-13 01:41-01:44 UTC / 10:41-10:44 JST**, one isolated managed
GoogleSQL database reproduced both creation orders. The first physical
index was removed with its child table before testing the reverse order.

| Existing child | Next CREATE TABLE | Managed result |
| --- | --- | --- |
| sales_order_shipping | sales_order | FailedPrecondition, empty description |
| sales_order after dropping sales_order_shipping | sales_order_shipping | FailedPrecondition, empty description |

Both successful independent child definitions expose exactly
`IDX_sales_order_shipping_address_id_90E532EDA843ECD5`, matching Omni LTS.2.
All four index metadata queries succeed. After each failed CREATE, the index
metadata still describes only the previously created child's backing index.
Distinct constraint names do not avoid this managed failure either. This is
a reproduced schema-deployment problem; the precise internal naming algorithm
remains unknown.

The retained eleven-step input has eight successful steps, these two expected
DDL failures, and one unrelated optional final-inventory query error:
TABLE_STATE is not a column of the target's INFORMATION_SCHEMA.TABLES.
That erroneous query is preserved with its InvalidArgument result as a
fixture error, rather than a service defect. It runs after all collision
controls and is not needed for their conclusion. A table inventory should
select TABLE_NAME and TABLE_TYPE without TABLE_STATE; that corrected query
was not executed in this run.

The newly created database was dropped successfully; a subsequent GetDatabase
returned NotFound. No managed build version is exposed, so the receipt bounds
this result to the observation time and the recorded target identity hash.

### Uniform uppercase/lowercase matrix across three engines

The follow-up on **2026-11-13 02:45-02:49 UTC / 11:45-11:49 JST** uses the
same 32-step plan once per engine. Each engine receives one new database.
Within it, the uppercase case runs first, both children and the parent are
removed, then the lowercase case runs. Each case tests shipping first and
order first, removing the successfully created children between orders.
Conditional cleanup also handles an engine accepting both definitions.

| Engine | UPPER_SNAKE_CASE | lower_snake_case |
| --- | --- | --- |
| Managed Spanner, build version unavailable | Second child fails in both orders: FailedPrecondition, empty description | Second child fails in both orders: FailedPrecondition, empty description |
| Omni 2026.r4-lts.2, linux/arm64 | Same failure in both orders | Same failure in both orders |
| Emulator 1.5.59, linux/arm64 | Both child CREATE statements succeed in both orders | Both child CREATE statements succeed in both orders |

The natural pairs are SALES_ORDER_SHIPPING(ADDRESS_ID) versus
SALES_ORDER(SHIPPING_ADDRESS_ID), and their lowercase equivalents. This is
an underscore-separated identifier boundary issue, rather than a lowercase
naming issue. It does not imply that every underscore-containing schema
fails: these definitions create compatible extra FK backing indexes.

| Case | Managed and Omni complete name | Emulator complete name |
| --- | --- | --- |
| Uppercase | `IDX_SALES_ORDER_SHIPPING_ADDRESS_ID_C53AEFD47826CE09` | `IDX_SALES_ORDER_SHIPPING_ADDRESS_ID_1AFAA9CDAD6073AC` |
| Lowercase | `IDX_sales_order_shipping_address_id_90E532EDA843ECD5` | `IDX_sales_order_shipping_address_id_BF3CF24FB4130FD9` |

In all engines, each child independently exposes the same complete name as
the other child within its case. Managed and Omni metadata agree exactly.
On Emulator, the catalog still returned only the first child's index row
after both CREATE statements succeeded. The 32-step plan exercised DDL
and metadata reads. It did not exercise writes through either constraint.

The 96 planned steps comprise **80 successes, eight reproduced DDL failures,
and eight conditional cleanup skips**. All six post-case table-inventory
queries and all six post-case index queries succeed and return empty sets.
No optional TABLE_STATE query is used in this plan. The one new managed DB
is dropped and GetDatabase returns NotFound. Both newly owned local runtimes
close successfully; their specific container IDs are confirmed absent.
The image manifests and producer identities are retained in the compact
matrix receipt. Earlier observations remain separately dated above.

## A stronger rejected hash family

The public Emulator computes
`Fingerprint64("2695C62E33338E4B_" + signature)`. This exactly reproduces all
18 retained Emulator names, including schema placement and long-name cases,
but none of the corresponding managed/Omni suffixes.

The retained managed/Omni source fixtures are:

| Candidate complete signature | Managed and Omni suffix |
| --- | --- |
| `IDX_T_X_Y_U` | `026F836781A5FCAF` |
| `IDX_U_A_B_N` | `8FE1FF22505F93E5` |

For any common 17-byte prefix, these 11-byte signatures produce 28-byte
inputs. The first 20 bytes are common. Public FarmHash HashLen17to32 reduces
to `HashLen16(A + ROR(c,30), B + c, k2+56)`, with common unsigned 64-bit A/B
and known c derived from the final eight bytes. Allowing arbitrary A/B is a
superset of all possible common prefixes.

The checked solver returns **UNSAT for the two retained managed/Omni pairs**.
Adding the fresh Omni collision pair `IDX_A_B_C_N -> 99E878FAF8057D7D` also
returns UNSAT. The modeled function matches an independent native FarmHash
implementation on 300 deterministic random binary-prefix controls. The
coordinator independently reran the calculation, not just the worker report.

Thus replacing the Emulator's 16-byte salt, even with arbitrary binary bytes
plus a separator, cannot explain these suffixes under this exact recipe.
This does not exclude a different prefix length, signature encoding,
fingerprint/hash function, or other seed-combination scheme. Finite scans of
common Farm/City, xxHash, Murmur, FNV and digest-derived candidates also found
no predictor; their exploratory reports remain task-local. Absence of a match
does not establish randomness or impossibility of prediction.

## Complete names in FORCE_INDEX

Both tested Omni releases accept these nullable-key-safe PLAN controls:

```sql
SELECT Owner FROM S.Same@{FORCE_INDEX=`S.SameFk`} WHERE Owner=1;
SELECT Owner FROM S.Same@{FORCE_INDEX=`S.IDX_Same_Owner_N_FA7F931463D537A2`}
WHERE Owner=1;
```

Unqualified SameFk and the unqualified physical index name are rejected in
this named-schema case. Bare `FORCE_INDEX=S.SameFk` is parsed as an expression
and fails name resolution; the complete hint value needs quoting. This fills
the earlier note's explicitly untested qualified-alias case for Omni only.
It does not newly establish managed behavior, and it does not make the
constraint alias a documented portable catalog identifier.

For durable identity, discover `(TABLE_SCHEMA, INDEX_NAME)` through
INFORMATION_SCHEMA.INDEXES and `(CONSTRAINT_SCHEMA, CONSTRAINT_NAME)` through
TABLE_CONSTRAINTS/REFERENTIAL_CONSTRAINTS. Keep them separate and quote the
complete selector when supplying a qualified name as a hint value. The
analyzer continues to use observed physical index metadata; no guessed
production-name predictor is added.


The [November 18 two-dialect comparison](fk-backing-index-resolution-20261118.md)
checks which physical index a constraint name selects in a query plan and
whether names recorded first on Omni predict managed results.

## Reproduction

Prepare without RPCs, using a fresh checkout-scoped scratch directory:

```sh
python3 knowledge/research/runtime-verification/evidence/fk-name-analysis-20261113/prepare.py \
  .tmp/fk-name-repro --variant full
```

The existing pinned producer can be built from the tools module and run with
an owned Omni backend, the appropriate digest from `runtime_targets.json`,
and the generated plan. Passing the same plan twice to one producer creates
two distinct random fixture databases in its owned container. `original`,
`reverse` and `qualified` variants reproduce the distinct retained scopes.
These commands do not authorize or create managed resources.

Prepare the exact managed follow-up input and producer without RPCs:

```sh
python3 knowledge/research/runtime-verification/evidence/fk-name-analysis-20261113/prepare.py \
  .tmp/fk-name-managed-repro --variant managed-lower
```

Build its generated managed.go from the tools module. For execution, supply
the explicitly selected verification instance through SPANNER_PROJECT_ID and
SPANNER_INSTANCE_ID, a fresh private SPANALYZER_FIXTURE_JOURNAL path, and the
guard value `SPANALYZER_OWNED_FIXTURE_APPROVAL=fk-name-managed-20261113` for a
currently authorized lifecycle. The producer creates one random database,
executes the eleven steps, drops that successfully created database, and
checks GetDatabase for absence. The preparation command itself performs no
database actions. The preserved optional TABLE_STATE query reproduces the
fixture error described above; it is not part of the minimal three-DDL case.

Use `--variant case-matrix` to reconstruct the later uniform 32-step plan
and its pinned local/managed producers. Run that identical plan on each
selected backend. Its managed producer uses the separate guard value
`SPANALYZER_OWNED_FIXTURE_APPROVAL=fk-case-matrix-20261113` for a currently
authorized one-DB lifecycle. Tables differing only in case are tested
sequentially, with empty schema/index checks before changing cases.

For the offline family rejection, create a task-local Python environment,
install the two pinned packages from the adjacent requirements.txt, and run
`check_prefix.py`. It performs no RPCs, requires UNSAT rather than a timeout,
and does not claim to test every possible production naming algorithm.
