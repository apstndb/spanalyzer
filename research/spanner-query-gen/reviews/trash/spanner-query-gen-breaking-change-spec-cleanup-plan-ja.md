# `spanner-query-gen` v1alpha spec cleanup plan

作成日: 2026-05-05

## 目的

`spanner-query-gen` は正式リリース前なので、互換性維持よりも仕様の単純化を優先できる。ここでは、既存ユースケースを損なわずに config / plan / CLI の複雑さを減らすための破壊的変更案をまとめる。

この文書は実装済み仕様ではなく、レビュー用の実装計画である。

## 基本方針

最重要の分離は次の 3 層。

- config: user intent を表す。人間が書く canonical input。
- plan: normalized semantics を表す。CI とレビューが見る machine-readable contract。
- generated Go: rendering result。plan から一方向に生成される成果物。

この 3 層を混ぜない。特に、review のために増えた plan field を config に逆流させない。
config は schema 外の未知 field を黙って無視しない。既知の移行対象 field は可能な限り移行先を示して拒否し、単なる typo や未設計 field は unsupported field として拒否する。

## 維持するユースケース

- Spanner GoogleSQL DDL から query result DTO を生成する。
- BigQuery DDL から query result DTO を生成する。
- BigQuery `EXTERNAL_QUERY` で connection ごとに別 Spanner catalog を解析する。
- BigQuery Spanner external dataset を BigQuery catalog binding として解析する。
- Spanner table / index shorthand から query SQL を展開する。
- 自動生成される Spanner table / index query は deterministic な `ORDER BY`
  をデフォルトで付け、必要な場合だけ `order_by: none` で opt out できる。
- 複数 query が互換な shape を返す場合に DTO を共有する。
- Spanner writes は mutation と DML の両方を生成できる。
- Spanner update は明示 update mask を持つ。
- `explain-plan` / `vet` を CI とレビューに使える。

## 採用する破壊的変更

### 1. `schemas` を `catalogs` に変更する

現行の `schemas` は DDL schema だけでなく、BigQuery external query mapping や Spanner external dataset binding も持つ。これは analyzer が参照する catalog graph なので、v1alpha では `catalogs` にする。

変更:

| 現行 | v1alpha |
| --- | --- |
| `schemas` | `catalogs` |
| `source` | `catalog` |
| `schema` alias | 廃止 |
| `spanner_source` | `spanner_catalog` |
| `external_source` | `spanner_database_uri` |
| `connection` alias | 廃止 |
| `external_schemas` alias | 廃止 |

### 2. `kind` と `dialect` を分ける

`dialect: spanner` のように resource kind と SQL dialect が混ざる形をやめる。

```yaml
catalogs:
- name: app_spanner
  kind: spanner
  dialect: googlesql
  ddl: schema/spanner.sql

- name: analytics_bigquery
  kind: bigquery
  dialect: googlesql
  project: example-project
  ddl: schema/bigquery.sql
```

`spanner_postgresql` は将来対応する場合も `kind: spanner`, `dialect: postgresql` とする。

### 3. `client: both` を廃止して `emit` にする

`client: both` は DTO shape、Spanner loader、BigQuery loader、mutation、DML、future query methods を混ぜている。v1alpha では生成 surface を `emit` で明示する。

最小形:

```yaml
go:
  package: querydemo
  out: querydemo.sql.go

emit:
  spanner:
    mutations: true
    dml: true
  bigquery:
    row_loader: true
    table_schema: true
```

`emit` は「どの generated API surface を出すか」を表す。README の最初の例では false flag を書かない。

### 4. query declaration を discriminated union にする

現行の `queries[].sql | table | index | federated` exactly-one top-level union は拡張時に散らかる。v1alpha では `queries[].kind` に寄せる。

```yaml
queries:
- name: GetSinger
  catalog: app_spanner
  kind: sql
  sql: |
    SELECT SingerId, FirstName
    FROM Singers
    WHERE SingerId = @SingerId
  result:
    cardinality: one
    struct: SingerRow
```

query kinds:

- `sql`
- `table`
- `index`
- `external_query`

`table` / `index` shorthand は、Spanner では `ORDER BY` なしの結果順序が
不定であることを避けるため、デフォルトで deterministic `ORDER BY` を付け
る。`order_by: key` は table では primary key、index では index key +
base table primary key の重複排除を意味する。最適化を優先したい場合は
query ごとに `order_by: none` を指定する。explicit `sql` と
`external_query` では SQL 本文が user intent なので `order_by` は使えな
い。

### 5. `federated` を `external_query` に変更する

`federated` は曖昧。BigQuery Spanner external dataset も federation の一種だが、`EXTERNAL_QUERY` とは解析単位が違う。

v1alpha では:

- `queries[].kind: external_query`: BigQuery `EXTERNAL_QUERY(connection, inner_sql)` を生成・解析する。
- `spanner_external_dataset`: BigQuery catalog binding。query kind ではない。

raw SQL 内の `EXTERNAL_QUERY(...)` は v1alpha では planning error にする。inner SQL と connection mapping を reviewable にするため、`query.kind: external_query` を使わせる。

### 6. result declaration を `result` block にまとめる

現行の `result`, `result_struct`, `required`, `required_policy` は同じ概念なので統合する。

```yaml
result:
  cardinality: one
  struct: SingerRow
  required:
    policy: strict
    fields:
    - SingerId
```

`queries[].params` は analyzer の parameter type input として v1alpha に残す。query method 用の signature 固定だけでなく、analyzer が型推論できない parameter、ARRAY / STRUCT parameter、BigQuery outer SQL と Spanner inner SQL の scope が分かれる `external_query` で必要になるため。

`params` は method signature customization ではなく analyzer input とする。regular `sql` / `table` / `index` query では `scope` を省略する。`external_query` では必要に応じて `scope: inner | outer` を指定し、同名 parameter が inner / outer の両方に現れうる場合は planning error にする。
`order_by` は generated `table` / `index` query 専用。Spanner での default
は `key`。`none` は ORDER BY を付けない明示 opt-out。

cardinality:

| value | semantics |
| --- | --- |
| `one` | exactly one row。0 rows は `ErrNoRows`、2+ rows は `ErrTooManyRows`。 |
| `maybe_one` | 0 or 1 row。2+ rows は `ErrTooManyRows`。 |
| `many` | 0+ rows。 |
| `exec` | row result ではなく execution result。 |

### 7. writes は branch semantics を明示する

現行の `operation: insert_or_update`, top-level `columns`, `update_mask`, `methods` は意味が混ざる。v1alpha では insert branch と update branch を分ける。ただし `conflict.strategy: insert_or_update` は Spanner `INSERT OR UPDATE` に忠実な範囲に制限し、`set(insert.columns)` は table primary key set + `set(update.columns)` と一致させる。duplicates は拒否し、rendering は deterministic key/update order に正規化する。

`key` は table primary key set を表す。省略時は DDL から推論する。partial primary key、unique index key、将来の conflict target は `key` では扱わない。

```yaml
writes:
- name: UpsertSingerName
  catalog: app_spanner
  table: Singers
  operation: upsert
  input: SingerWrite
  key:
  - SingerId
  insert:
    columns:
    - SingerId
    - FirstName
    - LastName
  update:
    columns:
    - FirstName
    - LastName
  conflict:
    strategy: insert_or_update
```

廃止:

| 廃止対象 | v1alpha |
| --- | --- |
| `operation: insert_or_update` | `operation: upsert` + `conflict.strategy: insert_or_update` |
| `update_mask` | `update.columns` |
| `update_mask: [auto_all_non_key_columns]` | `update.all_non_key_columns: true` |
| top-level `columns` | `insert.columns` or `update.columns` |
| `methods: [mutation, dml]` | global `emit.spanner` |

`replace` は mutation API の語彙として残す。ただし DML `replace` は生成しない。

### 8. `verification_hint` を廃止する

`verification_hint` と `verification_evidence` の分離は正しい方向だったが、v1alpha ではさらに単純化できる。

config では `not_checked` を書かせない。外部 evidence がある時だけ `access.verification_evidence` を書く。

```yaml
access:
  cloud_resource_connection_id: example-project.us.example-connection
```

外部 evidence がある場合:

```yaml
access:
  verification_evidence:
    status: verified
    verifier: terraform-plan
    checked_at: "2026-05-04T10:30:00Z"
```

`verified`, `mismatch`, `failed` は `verifier`, `checked_at` を必須にする。config に書かれた evidence は plan で `source: external_evidence` に正規化する。`source: live_probe` は generator-owned live verification が実装された時だけ plan に出す。

### 9. rules / suppressions を centralize する

現行の scoped `vet.disable` は便利だが、v1alpha の canonical config では `rules.suppressions` に集約する。`rules.severity` は full vet policy layer まで future work として扱う。

```yaml
rules:
  suppressions:
  - scope: catalog-binding/analytics_bigquery.app_dataset
    rule: external-dataset-access-unverified
    reason: access is validated by Terraform plan in CI
    owner: platform
    expires: "2026-12-31"
```

Local shorthand は v1alpha では入れない。必要なら後で sugar として追加し、plan では central shape に正規化する。

## External Dataset の単純化

Spanner external dataset は BigQuery catalog binding として固定する。

```yaml
catalogs:
- name: analytics_bigquery
  kind: bigquery
  project: example-project
  bindings:
    spanner_external_datasets:
    - name: app_dataset
      dataset: analytics_spanner
      spanner_catalog: app_spanner
      spanner_database_uri: google-cloudspanner://reader@/projects/example-project/instances/app/databases/app
      location: US
      access:
        cloud_resource_connection_id: example-project.us.example-connection
      projection:
        unsupported_columns: omit
        named_schema_tables: warn_and_omit
```

維持する invariant:

- external dataset table は read-only target。
- external dataset table への DML / metadata mutation は planning error。
- omitted columns / omitted named-schema tables は suppression しても plan から消さない。
- explicit omitted-column reference は analyzer error。
- `SELECT *` は BigQuery-visible columns のみへ展開し、selected output impact を記録する。
- explicit-column query でも relation provenance を残す。
- `access.verification_evidence.status: verified` は generator による独立検証ではなく、plan では `source: external_evidence` として扱う。将来の `source: live_probe` だけが generator による独立検証を意味する。

## Plan schema の単純化

Plan は normalized semantics と CI contract に絞る。review のために増えた冗長 field は削る。

### Catalog binding plan

推奨 shape:

```yaml
catalog_bindings:
- kind: spanner_external_dataset
  name: analytics_bigquery.app_dataset
  bigquery:
    dataset:
      project: example-project
      dataset: analytics_spanner
      sql_path: example-project.analytics_spanner
    location: US
  spanner:
    catalog: app_spanner
    database_uri: google-cloudspanner://reader@/projects/example-project/instances/app/databases/app
    database_role: reader
  access:
    mode: cloud_resource_connection
    cloud_resource_connection_id: example-project.us.example-connection
    verification:
      status: not_checked
      independently_verified_by_generator: false
  projection:
    unsupported_columns: omit
    named_schema_tables: warn_and_omit
    projected_tables:
    - table: Singers
      sql_path: example-project.analytics_spanner.Singers
      visible: true
      omitted_columns:
      - SearchName
```

削る候補:

- top-level `bigquery_dataset` と `bigquery_dataset_ref` の二重持ち。
- top-level `location` と nested metadata の二重持ち。
- `projection_loss` と `projection.relation_has_omitted_columns` の二重持ち。
- top-level `omitted_columns` と `projection.omitted_columns` の二重持ち。
- `verification_hint` / `configured_hint`。
- 通常 plan の heavy `projection_matrix.rows`。

### Query relation plan

推奨 shape:

```yaml
queries:
- name: ExternalDatasetSingerNames
  relations:
  - sql_path: example-project.analytics_spanner.Singers
    source: spanner_external_dataset_projection
    role: select_source
    allowed: true
    projection:
      relation_has_omitted_columns: true
      selected_output_affected: false
      omitted_columns:
      - Singers.SearchName
```

`projection_loss` は `projection.relation_has_omitted_columns` から導けるので v1alpha では消す。

### Audit metadata

`projection_matrix` の source URL、docs checked date、matrix rows は通常 plan には重い。必要なら次のどちらかに分離する。

- `explain-plan --audit`
- `explain-plan --output yaml --verbose`

通常の stable plan は semantic fields と diagnostic IDs を優先する。

## Canonical v1alpha config draft

```yaml
version: v1alpha

go:
  package: querydemo
  out: querydemo.sql.go

emit:
  spanner:
    mutations: true
    dml: true
  bigquery:
    row_loader: true
    table_schema: true

catalogs:
- name: app_spanner
  kind: spanner
  ddl: schema/spanner.sql

- name: analytics_bigquery
  kind: bigquery
  project: example-project
  ddl: schema/bigquery.sql
  bindings:
    external_query_connections:
    - name: app_conn
      id: example-project.us.example-connection
      spanner_catalog: app_spanner

    spanner_external_datasets:
    - name: app_dataset
      dataset: analytics_spanner
      spanner_catalog: app_spanner
      spanner_database_uri: google-cloudspanner://reader@/projects/example-project/instances/app/databases/app
      location: US
      access:
        cloud_resource_connection_id: example-project.us.example-connection
      projection:
        unsupported_columns: omit
        named_schema_tables: warn_and_omit

queries:
- name: ListSingers
  catalog: app_spanner
  kind: table
  table: Singers
  result:
    struct: SingerRow

- name: ExternalQuerySingerIDs
  catalog: analytics_bigquery
  kind: external_query
  binding: app_conn
  inner_sql: |
    SELECT SingerId
    FROM Singers
  outer_sql: |
    SELECT *
    FROM __external__
  result:
    struct: SingerRow

- name: ExternalDatasetSingerNames
  catalog: analytics_bigquery
  kind: sql
  sql: |
    SELECT SingerId, FirstName
    FROM `example-project.analytics_spanner.Singers`
  result:
    struct: SingerRow

writes:
- name: UpdateSingerName
  catalog: app_spanner
  table: Singers
  operation: update
  input: SingerWrite
  key:
  - SingerId
  update:
    columns:
    - FirstName
    - LastName

rules:
  suppressions:
  - scope: catalog-binding/analytics_bigquery.app_dataset
    rule: external-dataset-access-unverified
    reason: access is validated by Terraform plan in CI
    owner: platform
    expires: "2026-12-31"
```

## CLI cleanup

v1alpha CLI:

```sh
spanner-query-gen generate --config querygen.yaml
spanner-query-gen check --config querygen.yaml
spanner-query-gen explain-plan --config querygen.yaml --output yaml
spanner-query-gen vet --config querygen.yaml --output json
spanner-query-gen config-schema --output json
spanner-query-gen config-schema --out schemas/spanner-query-gen.v1alpha.schema.json
```

Current no-subcommand generation can be removed before v1alpha.

stdout / stderr contract:

| command | stdout | stderr |
| --- | --- | --- |
| `generate --out -` | generated Go | diagnostics |
| `generate --out file.go` | empty | diagnostics |
| `check` | empty by default | diagnostics |
| `explain-plan --output yaml/json` | plan | empty unless fatal CLI error |
| `vet --output yaml/json` | report | empty unless fatal CLI error |
| `vet` text | empty | warning / suppression summaries |
| `config-schema --output yaml/json` | v1alpha JSON Schema | empty unless fatal CLI error |

## Implementation plan

### Step 1: Add v1alpha config structs

Add new config structs for:

- `QueryCodegenV1AlphaConfig`
- `CatalogConfig`
- `QueryConfig`
- `WriteConfig`
- `RulesConfig`

Do not add compact aliases to these structs.

### Step 2: Normalize v1alpha config into current internal planning model

Initially, map v1alpha config into the existing internal structures so rendering work can continue.

This allows a smaller migration:

```text
v1alpha YAML -> v1alpha config -> normalized internal model -> existing plan -> renderer
```

### Step 3: Update plan schema

Once v1alpha config is accepted, simplify the plan:

- group catalog binding fields under `bigquery`, `spanner`, `access`, `projection`;
- remove duplicate projection fields;
- remove `verification_hint` / `configured_hint`;
- move heavy docs matrix to audit/verbose output;
- keep diagnostic IDs stable.

### Step 4: Remove compact config loader

Before v1alpha:

- reject `schemas`;
- reject `source`;
- reject `spanner_source`;
- reject `federated`;
- reject `client`;
- reject `update_mask`;
- reject `methods`;
- reject `vet.disable`;
- reject raw SQL `EXTERNAL_QUERY(...)`; any future literal-only migration helper
  should be separate from the v1alpha public grammar.
- reject unknown v1alpha YAML fields instead of silently ignoring schema drift or
  typos.

### Step 5: Update README / DESIGN

After the v1 structs and plan are implemented:

- make README show only v1alpha config;
- move old compact examples to a migration note or remove them;
- update DESIGN around catalog/query/write/DTO/diagnostic models.

## Resolved / Deferred

Resolved:

- `kind: spanner` stays; do not rename it to `spanner_database`.
- `dialect: googlesql` is the default for Spanner and BigQuery catalogs.
- Raw SQL `EXTERNAL_QUERY(...)` is rejected in v1alpha; any literal-only extraction belongs to a future migration helper, not the public grammar.
- `projection_matrix` rows are audit-only by default and exposed by `explain-plan --audit`.
- Generated Spanner `table` / `index` queries add deterministic key ordering by
  default; `order_by: none` is the explicit optimization opt-out.
- `EXTERNAL_QUERY` `inner_sql` is never automatically ordered because inner
  ordering can prevent root-partitionable Spanner execution and does not
  guarantee final BigQuery output order.
- `replace` remains `operation: replace` and is mutation-only unless verified DML support is added later.
- v1alpha has no write-level `emit`.
- `rules.severity` is future vet policy; v1alpha rules are suppressions-only.
- compact config can break before v1alpha; no `migrate-config` command is required unless real user demand appears.
- Unknown v1alpha YAML fields are rejected; config-schema is the reviewable
  public contract.

Deferred:

- DTO cross-catalog sharing declarations beyond per-query `result.struct`.
- full rule severity policy, strict mode, and warnings-as-errors.
- query methods and cardinality-sensitive generated APIs.
- BigQuery execution method generation.
- richer conflict targets such as unique indexes or `ON CONFLICT`.

## Recommendation

Adopt the breaking cleanup. The compact config is already showing its limits, and there is no stable compatibility promise yet.

The highest-value simplifications are:

1. `schemas` -> `catalogs`.
2. `query.kind` discriminated union.
3. `federated` -> `external_query`.
4. `result` block.
5. structured write branches.
6. removal of `verification_hint`.
7. central `rules.suppressions`.
8. reduced, normalized plan shape.

These changes preserve the current use cases while giving the project a simpler v1alpha surface.
