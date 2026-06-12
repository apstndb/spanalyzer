# `spanner-query-gen` 破壊的変更を許容した仕様整理案

作成日: 2026-05-04

## 前提

この提案は、ここまでの README / DESIGN / レビュー応答を踏まえた仕様整理案です。実装コードは共有されていないため、実装済み挙動の検証ではなく、**正式リリース前に破壊的変更を許容するなら、どのような stable v1 config / plan / CLI に整理すべきか**という設計提案として扱います。

現行文書はすでにかなり改善されています。特に、`client: both` が暫定 compact option であること、`gen` / `runtime` / `dto` への分離方針、`EXTERNAL_QUERY` と Spanner external dataset の分離、`replace` の mutation-only 化、`update_mask` 明示化、`explain-plan` / `vet` の中核化は良い方向です。

ただし、破壊的変更を許容できるなら、compact config を温存しながら structured config を追加するより、**stable v1 では最初から structured config のみを canonical form にする**方が仕様は大きく整理できます。

---

## 結論

推奨は次の 7 点です。

1. `schemas` を `catalogs` に置き換える。
2. `source` / `spanner_source` を `catalog` / `spanner_catalog` に統一する。
3. `client: both` を廃止し、`gen` / `surfaces` / `dto` に完全分離する。
4. `queries[].sql|table|index|federated` の exactly-one top-level union を廃止し、`queries[].query.kind` の discriminated union にする。
5. `federated` という名前をやめ、`external_query` と呼ぶ。Spanner external dataset は query kind ではなく BigQuery catalog binding のままにする。
6. `writes` は `operation: upsert` + `insert.columns` + `update.mask` + `conflict.strategy` に整理し、`insert_or_update` / `update_mask` / top-level `columns` を stable v1 から消す。
7. `vet.disable` を canonical にはせず、`rules.suppressions` に集約する。必要なら local shorthand は将来追加する。

最重要の考え方は、**config は user intent、plan は normalized semantics、generated Go は rendering result** という三層を混ぜないことです。

---

## なぜ `schemas` ではなく `catalogs` か

現行の `schemas` は、最初は Spanner / BigQuery DDL schema の置き場として自然でした。しかし現在は以下を同じ場所で扱っています。

- Spanner database schema from DDL
- BigQuery schema from DDL
- BigQuery `EXTERNAL_QUERY` connection mapping
- BigQuery Spanner external dataset projection
- access verification / cloud resource connection / role metadata

これはもはや単なる schema ではなく、**SQL analyzer が参照する catalog graph** です。したがって stable v1 では `schemas` ではなく `catalogs` にした方が、external dataset や将来の source expansion を自然に表現できます。

推奨形:

```yaml
catalogs:
- name: app_spanner
  kind: spanner_database
  dialect: googlesql
  ddl: schema/spanner.sql
  proto_descriptors:
  - schema/order_descriptors.pb

- name: analytics_bigquery
  kind: bigquery
  project: example-project
  ddl: schema/bigquery.sql
  bindings:
    external_queries:
    - name: app_conn
      connection_id: example-project.us.example-connection
      spanner_catalog: app_spanner

    spanner_external_datasets:
    - name: app_external_dataset
      dataset: analytics_spanner
      spanner_catalog: app_spanner
      spanner_database_uri: google-cloudspanner://reader@/projects/example-project/instances/app/databases/app
      location: US
      access:
        mode: cloud_resource_connection
        cloud_resource_connection_id: example-project.us.example-connection
        database_role: reader
        verification:
          status: not_checked
      projection:
        unsupported_columns: omit
        named_schema_tables: warn_and_omit
```

### 変更点

| 現行 / compact | stable v1 推奨 | 理由 |
| --- | --- | --- |
| `schemas` | `catalogs` | analyzer の catalog graph を表すため。 |
| `dialect: spanner` | `kind: spanner_database`, `dialect: googlesql` | resource kind と SQL dialect を分ける。 |
| `source` | `catalog` | query/write の対象は schema ではなく catalog。 |
| `spanner_source` | `spanner_catalog` | external query / dataset binding の対象を明確化。 |
| `external_source` | `spanner_database_uri` | URI の意味を具体化。 |
| `connection` alias | `connection_id` or `cloud_resource_connection_id` | `EXTERNAL_QUERY` connection と external dataset access connection の混同を避ける。 |
| `access: euc` | `access.mode: end_user_credentials` | canonical value のみ残す。 |
| `access: cloud_resource` | `access.mode: cloud_resource_connection` | plan value と config value を揃える。 |

---

## `client: both` は廃止する

`client: both` は便利ですが、以下を一つの言葉に詰め込みすぎています。

- DTO nullable wrapper policy
- Spanner row decoding
- Spanner mutation encoding
- Spanner DML parameter generation
- BigQuery row loading
- BigQuery schema metadata generation
- future query methods

stable v1 では次のように分けるのがよいです。

```yaml
gen:
  go:
    package: querydemo
    out: querydemo.sql.go
    emit_sql_constants: true
    tags:
      json: true
      db: false
    nullable:
      kind: generated_wrapper
      wrapper: NullValue

surfaces:
  spanner:
    query_methods: true
    dml_statements: true
    mutations: true
  bigquery:
    row_loader: true
    table_schema: true
    query_methods: false

dto:
  sharing:
    default: same_catalog_only
    cross_catalog: explicit
    query_write: explicit
```

ここでは `runtime` より `surfaces` という名前を推奨します。理由は、これは実行時環境そのものではなく、**どの generated API surface を emit するか**を指定する設定だからです。

---

## query declaration は discriminated union にする

現行の `queries[]` は `sql` / `table` / `index` / `federated` の exactly-one field です。これは compact config では読みやすいですが、将来 `sql_file`、annotation、parameter override、BigQuery external query、query hints などが増えると top-level が散らかります。

stable v1 では `query.kind` に寄せるのがよいです。

```yaml
queries:
- name: GetSinger
  catalog: app_spanner
  query:
    kind: sql
    sql: |
      SELECT SingerId, FirstName, LastName
      FROM Singers
      WHERE SingerId = @SingerId
  params:
  - name: SingerId
    type: INT64
  result:
    cardinality: one
    struct: SingerRow
    required:
      policy: strict
      fields:
      - SingerId

- name: ListSingers
  catalog: app_spanner
  query:
    kind: table
    table: Singers
    columns: non_hidden
  result:
    cardinality: many
    struct: SingerRow

- name: FindAlbumsByTitle
  catalog: app_spanner
  query:
    kind: index
    index: AlbumsByTitle
    key_prefix:
    - AlbumTitle
  result:
    cardinality: many
    struct: AlbumIndexRow

- name: FederatedSingerIDs
  catalog: analytics_bigquery
  query:
    kind: external_query
    binding: app_conn
    inner_sql: |
      SELECT SingerId
      FROM Singers
    outer_sql: |
      SELECT *
      FROM __external__
  result:
    cardinality: many
    struct: SingerRow

- name: ExternalDatasetSingerNames
  catalog: analytics_bigquery
  query:
    kind: sql
    sql: |
      SELECT SingerId, FirstName
      FROM `example-project.analytics_spanner.Singers`
  result:
    cardinality: many
    struct: SingerRow
```

### `federated` ではなく `external_query`

`federated` という名前は曖昧です。BigQuery Spanner external dataset も federation の一種ですが、`EXTERNAL_QUERY` とは解析単位が違います。stable v1 では `query.kind: external_query` と呼ぶ方がよいです。

- `external_query`: BigQuery `EXTERNAL_QUERY(connection, inner_sql)` を生成・解析する query kind。
- `spanner_external_dataset`: BigQuery catalog binding。query kind ではない。通常の BigQuery SQL から参照される。

### raw SQL 内の `EXTERNAL_QUERY` は原則禁止でよい

破壊的変更を許容するなら、`query.kind: sql` に含まれる raw `EXTERNAL_QUERY(...)` は default で planning error にしてよいです。inner SQL と connection mapping を reviewable にするため、`query.kind: external_query` を使わせるべきです。

例外を許す場合も、次のように明示 opt-in にするのが安全です。

```yaml
query:
  kind: sql
  sql: SELECT * FROM EXTERNAL_QUERY(...)
  allow_raw_external_query: true
```

ただし、この opt-in は first stable では不要かもしれません。

---

## result declaration を `result` block に集約する

現行の `result`, `result_struct`, `required`, `required_policy` は同じ概念に属します。stable v1 では `result` block にまとめると読みやすくなります。

```yaml
result:
  cardinality: one
  struct: SingerRow
  required:
    policy: strict
    fields:
    - SingerId
```

推奨 cardinality semantics:

| value | 意味 |
| --- | --- |
| `one` | exactly one row。0 rows は `ErrNoRows`、2+ rows は `ErrTooManyRows`。 |
| `maybe_one` | 0 or 1 row。2+ rows は `ErrTooManyRows`。 |
| `many` | 0+ rows。 |
| `exec` | row result ではなく execution result。DML / future query methods 用。 |

---

## DTO sharing は明示制に寄せる

現行の shared `result_struct` merge は便利ですが、Spanner / BigQuery / write input をまたぐと危険です。stable v1 では、同一 catalog 内の query result merge は許してもよいですが、cross-catalog または query/write role の共有は明示宣言に寄せるべきです。

推奨:

```yaml
dto:
  sharing:
    default: same_catalog_only
    cross_catalog: explicit
    query_write: explicit
  structs:
  - name: SingerRow
    allow:
      catalogs:
      - app_spanner
      - analytics_bigquery
      roles:
      - query_result
  - name: SingerWrite
    allow:
      catalogs:
      - app_spanner
      roles:
      - write_input
```

この方針なら、以下を安全に分離できます。

- BigQuery external dataset から読める BigQuery-visible columns
- underlying Spanner provenance
- Spanner write input requiredness
- nullable result field
- write-safe nullable semantics

特に write input は read result より厳しく扱うべきです。`query result` に存在しない write-only field を generator が黙って追加しない現行方針は維持しつつ、stable v1 では DTO sharing policy で明文化します。

---

## writes は branch semantics を必ず分ける

現行 compact config の `operation: insert_or_update` + `update_mask` は、実装しやすい一方で、insert branch と update branch の意味が混ざります。stable v1 では `operation: upsert` にし、conflict strategy と insert/update branch を分けます。

```yaml
writes:
- name: UpsertSingerName
  catalog: app_spanner
  table: Singers
  operation: upsert
  input:
    struct: SingerWrite
  key:
    columns:
    - SingerId
  insert:
    columns:
    - SingerId
    - FirstName
    - LastName
  update:
    mask:
      columns:
      - FirstName
      - LastName
  conflict:
    strategy: insert_or_update
  emit:
    mutation: true
    dml: true

- name: ReplaceSinger
  catalog: app_spanner
  table: Singers
  operation: replace
  input:
    struct: SingerReplace
  insert:
    columns:
    - SingerId
    - FirstName
    - LastName
  emit:
    mutation: true
    dml: false

- name: UpdateSingerName
  catalog: app_spanner
  table: Singers
  operation: update
  input:
    struct: SingerNameUpdate
  key:
    columns:
    - SingerId
  update:
    mask:
      columns:
      - FirstName
      - LastName
  emit:
    mutation: true
    dml: true
```

### なくすべき compact constructs

| 廃止対象 | 理由 |
| --- | --- |
| `operation: insert_or_update` | operation と conflict strategy が混ざる。`operation: upsert` + `conflict.strategy` に分ける。 |
| `update_mask` | `update.mask.columns` に移す。 |
| `update_mask: [auto_all_non_key_columns]` | sentinel string と column list の混在を避ける。 |
| top-level `columns` | insert/replace columns なのか update mask なのか曖昧。 |
| `methods: [mutation, dml]` | `emit.mutation`, `emit.dml` の方が boolean feature として明確。 |

全列 update の opt-in は次の形にします。

```yaml
update:
  mask:
    all_non_key_columns: true
```

これは plan では `broad_update: true` として見えるようにし、default で warning にするのがよいです。

---

## external dataset は `catalog binding` として固定する

ここまでの議論どおり、BigQuery Spanner external dataset は `EXTERNAL_QUERY` の別記法ではありません。stable v1 でも query kind には入れず、BigQuery catalog binding として扱うのが正しいです。

推奨 binding:

```yaml
catalogs:
- name: analytics_bigquery
  kind: bigquery
  project: example-project
  bindings:
    spanner_external_datasets:
    - name: app_external_dataset
      dataset: analytics_spanner
      spanner_catalog: app_spanner
      spanner_database_uri: google-cloudspanner://reader@/projects/example-project/instances/app/databases/app
      location: US
      access:
        mode: cloud_resource_connection
        cloud_resource_connection_id: example-project.us.example-connection
        database_role: reader
        verification:
          status: not_checked
      projection:
        unsupported_columns: omit
        named_schema_tables: warn_and_omit
```

Plan では次を必ず分けます。

```yaml
catalog_bindings:
- kind: spanner_external_dataset
  binding: analytics_bigquery.app_external_dataset
  bigquery:
    dataset:
      project: example-project
      dataset: analytics_spanner
      sql_path: `example-project.analytics_spanner`
  spanner:
    catalog: app_spanner
    database_uri: google-cloudspanner://reader@/projects/example-project/instances/app/databases/app
    database_role: reader
  access:
    mode: cloud_resource_connection
    cloud_resource_connection_id: example-project.us.example-connection
    verification:
      status: not_checked
      source: user_hint
      independently_verified_by_generator: false
  projection:
    unsupported_columns: omit
    named_schema_tables: warn_and_omit
    projected_tables:
    - table: Singers
      visible: true
      omitted_columns:
      - SearchName
```

### external dataset で stable v1 に入れるべき invariant

- external dataset tables は read-only target。
- external dataset table への DML / metadata mutation は planning error。
- omitted columns / omitted named-schema tables は suppression しても plan から消さない。
- explicit omitted-column reference は error。
- `SELECT *` は BigQuery-visible columns のみへ展開し、`projection_loss` を記録する。
- explicit-column query でも relation provenance を残す。
- `access.verification.status: verified` は、`source: live_probe` でない限り generator による独立検証ではない。

---

## rules / vet は central policy にする

現行の scoped `vet.disable` は使いやすいですが、stable v1 では suppressions を centralize した方が CI で監査しやすいです。

```yaml
rules:
  defaults:
    severity: warning
  severity:
    require-explicit-update-mask: error
    broad-update-mask: warning
    cross-dialect-timestamp-truncation: warning
    external-dataset-access-unverified: warning
  suppressions:
  - scope: query/FederatedSingerIDs
    rule: cross-dialect-timestamp-truncation
    reason: downstream only uses microsecond precision
    owner: analytics-platform
    expires: "2026-12-31"
  - scope: catalog-binding/analytics_bigquery.app_external_dataset
    rule: external-dataset-access-unverified
    reason: checked by Terraform validation in CI
    owner: platform
```

Local suppressions を残したい場合も、canonical plan では必ず central shape に正規化します。

Diagnostic output は stable ID を必須にします。

```yaml
diagnostics:
- id: external-dataset-access-unverified
  stage: planning
  severity: warning
  suppressible: true
  suppressed: false
  scope: catalog-binding/analytics_bigquery.app_external_dataset
  message: External dataset access was not independently verified by the generator.
  remediation: Provide verification evidence or run a future live verification command.
```

`planning-error` のような総称 ID は避け、各 error / warning に具体的 ID を持たせます。

---

## CLI も breaking change で整理する

stable v1 では subcommand を明示した方がよいです。

```sh
spanner-query-gen generate --config querygen.yaml
spanner-query-gen check --config querygen.yaml
spanner-query-gen explain-plan --config querygen.yaml --output yaml
spanner-query-gen vet --config querygen.yaml --output json
spanner-query-gen migrate-config --from compact --to v1 --config old.yaml --out new.yaml
```

推奨 stdout / stderr contract:

| command | stdout | stderr |
| --- | --- | --- |
| `generate --out -` | generated Go | diagnostics |
| `generate --out file.go` | empty | diagnostics |
| `check` | empty or machine report if `--output` | diagnostics |
| `explain-plan --output yaml/json` | plan | empty unless fatal CLI error |
| `vet --output yaml/json` | report | empty unless fatal CLI error |
| `vet` text | empty | warning / suppression summaries |

---

## stable v1 の full example

```yaml
version: "1"

gen:
  go:
    package: querydemo
    out: querydemo.sql.go
    emit_sql_constants: true
    tags:
      json: true
      db: false
    nullable:
      kind: generated_wrapper
      wrapper: NullValue

surfaces:
  spanner:
    query_methods: true
    dml_statements: true
    mutations: true
  bigquery:
    row_loader: true
    table_schema: true
    query_methods: false

dto:
  sharing:
    default: same_catalog_only
    cross_catalog: explicit
    query_write: explicit
  structs:
  - name: SingerRow
    allow:
      catalogs:
      - app_spanner
      - analytics_bigquery
      roles:
      - query_result
  - name: SingerWrite
    allow:
      catalogs:
      - app_spanner
      roles:
      - write_input

catalogs:
- name: app_spanner
  kind: spanner_database
  dialect: googlesql
  ddl: schema/spanner.sql

- name: analytics_bigquery
  kind: bigquery
  project: example-project
  ddl: schema/bigquery.sql
  bindings:
    external_queries:
    - name: app_conn
      connection_id: example-project.us.example-connection
      spanner_catalog: app_spanner

    spanner_external_datasets:
    - name: app_external_dataset
      dataset: analytics_spanner
      spanner_catalog: app_spanner
      spanner_database_uri: google-cloudspanner://reader@/projects/example-project/instances/app/databases/app
      location: US
      access:
        mode: cloud_resource_connection
        cloud_resource_connection_id: example-project.us.example-connection
        database_role: reader
        verification:
          status: not_checked
      projection:
        unsupported_columns: omit
        named_schema_tables: warn_and_omit

queries:
- name: GetLiteral
  catalog: app_spanner
  query:
    kind: sql
    sql: SELECT 1 AS value
  result:
    cardinality: one
    struct: LiteralRow
    required:
      policy: strict
      fields:
      - value

- name: ListSingers
  catalog: app_spanner
  query:
    kind: table
    table: Singers
    columns: non_hidden
  result:
    cardinality: many
    struct: SingerRow

- name: FederatedSingerIDs
  catalog: analytics_bigquery
  query:
    kind: external_query
    binding: app_conn
    inner_sql: |
      SELECT SingerId
      FROM Singers
    outer_sql: |
      SELECT *
      FROM __external__
  result:
    cardinality: many
    struct: SingerRow

- name: ExternalDatasetSingerNames
  catalog: analytics_bigquery
  query:
    kind: sql
    sql: |
      SELECT SingerId, FirstName
      FROM `example-project.analytics_spanner.Singers`
  result:
    cardinality: many
    struct: SingerRow

writes:
- name: UpsertSingerName
  catalog: app_spanner
  table: Singers
  operation: upsert
  input:
    struct: SingerWrite
  key:
    columns:
    - SingerId
  insert:
    columns:
    - SingerId
    - FirstName
    - LastName
  update:
    mask:
      columns:
      - FirstName
      - LastName
  conflict:
    strategy: insert_or_update
  emit:
    mutation: true
    dml: true

- name: UpdateSingerName
  catalog: app_spanner
  table: Singers
  operation: update
  input:
    struct: SingerWrite
  key:
    columns:
    - SingerId
  update:
    mask:
      columns:
      - FirstName
      - LastName
  emit:
    mutation: true
    dml: true

- name: ReplaceSinger
  catalog: app_spanner
  table: Singers
  operation: replace
  input:
    struct: SingerWrite
  insert:
    columns:
    - SingerId
    - FirstName
    - LastName
  emit:
    mutation: true
    dml: false

rules:
  severity:
    broad-update-mask: warning
    cross-dialect-timestamp-truncation: warning
    external-dataset-access-unverified: warning
  suppressions:
  - scope: catalog-binding/analytics_bigquery.app_external_dataset
    rule: external-dataset-access-unverified
    reason: access is validated by Terraform plan in CI
    owner: platform
    expires: "2026-12-31"
```

---

## 実装順序案

破壊的変更を許容するなら、実装順は次がよいです。

### Step 1: normalized internal config を先に作る

compact config loader と structured config loader の両方を作るのではなく、まず stable v1 config だけを parse し、すぐ normalized plan input に変換します。

```text
YAML config -> normalized config -> plan -> render
```

ここで aliases は持たせません。

### Step 2: `migrate-config` を optional にする

互換性を runtime に残すより、one-shot migration command に寄せます。

```sh
spanner-query-gen migrate-config --from compact --to v1 old.yaml > querygen.yaml
```

migration は best-effort でよく、曖昧な箇所は `TODO` comment または diagnostic を出します。

### Step 3: plan golden を stable v1 の契約にする

実装コードがなくても外部レビューできるように、次を golden として固定します。

- config input
- expected diagnostics
- expected normalized plan excerpt
- expected generated Go excerpt

### Step 4: compact support を消す

正式リリース前なら、`client`, `source`, `external_schemas`, `update_mask`, `insert_or_update` などの legacy compact constructs は loader で warning するのではなく error にしてよいです。

---

## 仕様整理後に残る判断事項

以下はまだ決める必要があります。

1. `schemas` -> `catalogs` の rename を本当に行うか。
2. `runtime` ではなく `surfaces` という名前にするか。
3. raw BigQuery SQL 内の `EXTERNAL_QUERY` を stable v1 で禁止するか。
4. DTO cross-catalog sharing を declaration 必須にするか、それとも query-level opt-in にするか。
5. suppressions を central `rules.suppressions` のみにするか、local shorthand も許すか。
6. `operation: replace` を残すか、`operation: insert` + `mode: replace_existing` のように mutation semantics をさらに明示するか。

私の推奨は、1〜5 はすべて強めに整理し、6 は `operation: replace` を残す、です。`replace` は mutation API の語彙として十分定着している一方、raw `EXTERNAL_QUERY` や compact aliases は安定後に残すほど仕様負債になります。

---

## 最終提案

stable v1 を切る前に、README / DESIGN を次の構成に組み替えるのがよいです。

```text
README.md
  - What it is / is not
  - CLI usage
  - Minimal stable v1 config
  - Query kinds
  - Write kinds
  - External query vs external dataset
  - explain-plan / vet

DESIGN.md
  - Product boundary
  - Catalog model
  - Query model
  - Write model
  - DTO model
  - Plan model
  - Diagnostics / rules
  - External dataset semantics
  - Generated Go surface
  - Roadmap
  - Deprecated compact config migration notes
```

これにより、今ある良い設計判断を維持しながら、`client: both`、`schemas` の過負荷、`federated` の曖昧さ、`insert_or_update` の branch 混在、`vet.disable` の散在を一気に整理できます。
