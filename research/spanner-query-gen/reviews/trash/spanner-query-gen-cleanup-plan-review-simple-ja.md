# `spanner-query-gen` stable v1 cleanup plan への追加レビュー

対象: `spanner-query-gen stable v1 spec cleanup plan`
前提: これは実装コード・テストログのレビューではなく、共有された Markdown 計画に対する仕様レビューです。実装済みかどうかは検証していません。

## 総評

今回の cleanup plan はかなり良い方向です。特に、次の判断はそのまま採用してよいと思います。

- `config` / `plan` / `generated Go` を明確に分ける。
- `schemas` を `catalogs` に変える。
- `client: both` を廃止し、生成する surface を明示する。
- `queries[].sql | table | index | federated` の top-level union をやめ、query kind に寄せる。
- `federated` を `external_query` に改名し、Spanner external dataset とは分ける。
- `insert_or_update` を `upsert + conflict.strategy` に分解する。
- `update_mask` / `columns` / `methods` の曖昧な alias を stable v1 で捨てる。
- `rules.suppressions` を中央集約し、plan の diagnostic policy として扱う。
- heavy な projection matrix を通常 plan から外し、audit / verbose に逃がす。

大きな方向転換は不要です。ただし、**まだ canonical config example が重い**です。破壊的変更を許容するなら、stable v1 の README で見せる config はもっと短くできます。

一番の提案は次です。

> stable v1 の README では、全ユースケースを 1 つの YAML に詰め込まない。
> Spanner-only minimal example、`EXTERNAL_QUERY` example、external dataset example、DTO sharing example を分ける。

今の draft は、Spanner、BigQuery、`EXTERNAL_QUERY`、external dataset、access verification、projection policy、DTO cross-catalog sharing、rules suppression を 1 つの config に入れているため、仕様は整理されているのに見た目が重いです。

## さらに simple にするための提案

### 1. README の canonical example は Spanner-only にする

stable v1 の最初の例はこれくらいまで削った方がよいです。

```yaml
version: "1"

go:
  package: querydemo
  out: querydemo.sql.go

emit:
  spanner:
    query_methods: true
    mutations: true
    dml: true

catalogs:
- name: app
  kind: spanner
  ddl: schema/spanner.sql

queries:
- name: ListSingers
  catalog: app
  kind: table
  table: Singers
  result:
    cardinality: many
    struct: SingerRow

writes:
- name: UpdateSingerName
  catalog: app
  table: Singers
  operation: update
  input: SingerWrite
  key:
  - SingerId
  update:
    columns:
    - FirstName
    - LastName
  emit:
  - mutation
  - dml
```

これは計画案より少し flatten しています。もし実装上 `input.struct` や `key.columns` の方が扱いやすければ、それでも構いません。ただし README の最初の例では、`external dataset`、`rules.suppressions`、`dto.structs`、`verification` は出さない方がよいです。

### 2. `queries[].query.kind` は `queries[].kind` で十分かもしれない

cleanup plan は `query.kind` に寄せています。discriminated union にする判断は正しいです。ただ、`queries` 配下の各 item はすでに query なので、さらに `query:` block を挟む必要は薄いです。

現在案:

```yaml
queries:
- name: GetSinger
  catalog: app_spanner
  query:
    kind: sql
    sql: |
      SELECT ...
  result:
    cardinality: one
    struct: SingerRow
```

より単純な案:

```yaml
queries:
- name: GetSinger
  catalog: app_spanner
  kind: sql
  sql: |
    SELECT ...
  result:
    cardinality: one
    struct: SingerRow
```

`kind` を持つだけで、`sql` / `table` / `index` / `external_query` のどれとして解釈するかは十分に分かります。top-level exactly-one union の問題は、`kind` を足すだけでも解消できます。

ただし、`query:` block には「query-specific fields を一箇所に閉じ込める」という利点があります。config の見た目の単純さを優先するなら `kind` top-level、将来の拡張境界を優先するなら `query.kind` のまま、という判断です。私は README の読みやすさを優先して `queries[].kind` を推します。

### 3. `surfaces` より `emit` または `outputs` の方が利用者には分かりやすい

`surfaces` は設計者には正確ですが、利用者には少し抽象的です。計画案でも `writes[].emit` を使っているので、global でも `emit` に揃えると読みやすくなります。

```yaml
emit:
  spanner:
    query_methods: true
    mutations: true
    dml: true
  bigquery:
    row_loader: true
    table_schema: true
```

また、README では `query_methods: false` のような false flag を出さない方がよいです。stable v1 example は「何を出すか」だけを positive に書き、出さないものは省略でよいです。

### 4. `dialect: googlesql` は default にする

`kind` と `dialect` を分ける判断は正しいです。ただし BigQuery も Spanner も当面 GoogleSQL が主戦場なら、`dialect: googlesql` は default にしてよいです。

```yaml
catalogs:
- name: app
  kind: spanner
  ddl: schema/spanner.sql

- name: analytics
  kind: bigquery
  project: example-project
  ddl: schema/bigquery.sql
```

PostgreSQL dialect Spanner に対応する時だけ明示します。

```yaml
catalogs:
- name: app_pg
  kind: spanner
  dialect: postgresql
  ddl: schema/spanner_pg.sql
```

Open question の回答としては、`kind: spanner_database` までは不要だと思います。`kind: spanner` / `kind: bigquery` の方が短く、`ddl` や binding が database / dataset の詳細を表します。

### 5. `access.verification.status: not_checked` は config から消してよい

cleanup plan は `verification_hint` を消して `access.verification.status` に統一しています。この方向は前より単純ですが、`not_checked` は user intent というより plan result です。

config / plan / generated Go を分けるという基本方針から見ると、次の方がきれいです。

```yaml
access:
  cloud_resource_connection_id: example-project.us.example-connection
```

外部 evidence がある時だけ書きます。

```yaml
access:
  cloud_resource_connection_id: example-project.us.example-connection
  verification_evidence:
    verifier: terraform-plan
    checked_at: "2026-05-04T10:30:00Z"
```

plan 側で次のように正規化すれば十分です。

```yaml
access:
  verification:
    status: not_checked
    source: user_config
    independently_verified_by_generator: false
```

`not_checked` を毎回 config に書かせると、plan field が config に逆流しているように見えます。

### 6. external dataset config は重複を減らす

現行 draft では `spanner_database_uri` に role が含まれ、さらに `access.database_role` もあります。これは plan の整合性チェックには便利ですが、config では重複です。

より単純な config:

```yaml
bindings:
  spanner_external_datasets:
  - name: app_dataset
    dataset: analytics_spanner
    spanner_catalog: app
    spanner_database_uri: google-cloudspanner://reader@/projects/example-project/instances/app/databases/app
    location: US
    access:
      cloud_resource_connection_id: example-project.us.example-connection
```

plan 側で URI から database role を抽出して、`spanner.database_role: reader` として出せばよいです。もし URI に role がなく、別途 role を指定したいニーズが出たら、その時に `database_role` を追加すればよいです。

同様に、`access.mode: cloud_resource_connection` も今は不要に見えます。今サポートする access mode が Cloud Resource connection だけなら、`cloud_resource_connection_id` の存在で十分です。

### 7. external dataset の projection policy は安全 default に寄せる

README の最初の external dataset example では `projection` block も省略できるとよいです。stable v1 の default は安全側にして、たとえば次のようにできます。

```yaml
projection:
  unsupported_columns: reject
  named_schema_tables: reject
```

この default なら、example に policy を書かなくても危険な lossy projection にはなりません。`omit` や `warn_and_omit` は advanced example で示せば十分です。

```yaml
projection:
  unsupported_columns: omit
  named_schema_tables: warn_and_omit
```

### 8. write config は `mask` を config から消してもよい

Spanner semantics としては update mask が重要です。ただし config 上では `update` branch の `columns` がそのまま update mask なので、`mask.columns` まで深くしなくてもよいです。

現在案:

```yaml
update:
  mask:
    columns:
    - FirstName
    - LastName
```

より単純な案:

```yaml
update:
  columns:
  - FirstName
  - LastName
```

plan ではもちろん `update_mask.columns` として出してよいです。config は user intent、plan は normalized semantics なので、config で `mask` という内部寄りの語彙を必ずしも使う必要はありません。

さらに短くするなら、次の形もありです。

```yaml
key:
- SingerId
update:
- FirstName
- LastName
```

ただしこれは将来 `update.returning` や `update.where` を足したくなった時に拡張しづらいので、私は `update.columns` くらいが良い妥協だと思います。

### 9. DTO sharing は README 最初の例から外す

cleanup plan の `dto.structs` は設計としては妥当ですが、canonical config に入れるとかなり重く見えます。

おすすめは次です。

- same catalog + same role の query result sharing は、同じ `result.struct` 名なら許可。
- cross-catalog sharing は明示 opt-in。
- query result と write input の共有も明示 opt-in。
- README の最初の例では `dto` block を出さない。

cross-catalog sharing を説明する advanced example だけ、次のように出せばよいです。

```yaml
dto:
  structs:
  - name: SingerRow
    allow:
      catalogs:
      - app
      - analytics
      roles:
      - query_result
```

それ以外の例では、Spanner query と BigQuery query で別 struct 名を使えば、DTO sharing policy を説明しなくて済みます。

### 10. raw SQL `EXTERNAL_QUERY(...)` は stable v1 では reject でよい

この点は cleanup plan の方針に同意です。`EXTERNAL_QUERY` を raw BigQuery SQL 内で許すと、inner SQL、connection mapping、outer SQL の provenance が見えにくくなります。

stable v1 は次に固定してよいです。

```yaml
kind: external_query
binding: app_conn
inner_sql: |
  SELECT SingerId FROM Singers
outer_sql: |
  SELECT * FROM __external__
```

どうしても raw `EXTERNAL_QUERY` を許す必要が後から出た場合は、後方互換で次のような escape hatch を追加できます。

```yaml
kind: sql
allow_external_query: true
sql: |
  SELECT * FROM EXTERNAL_QUERY(...)
```

しかし stable v1 の最初から入れる必要はありません。

### 11. CLI は `check` を subcommand にするか `generate --check` に寄せる

cleanup plan の CLI は分かりやすいです。

```sh
spanner-query-gen generate --config querygen.yaml
spanner-query-gen check --config querygen.yaml
spanner-query-gen explain-plan --config querygen.yaml --output yaml
spanner-query-gen vet --config querygen.yaml --output json
```

ただ、さらに単純化するなら `check` は `generate --check` に寄せてもよいです。

```sh
spanner-query-gen generate --config querygen.yaml
spanner-query-gen generate --config querygen.yaml --check
spanner-query-gen explain-plan --config querygen.yaml --output yaml
spanner-query-gen vet --config querygen.yaml --output json
```

`check` という subcommand は CI では読みやすいので残しても構いません。ここは好みの問題です。実装パスを減らしたいなら `generate --check`、CLI の意図を明確にしたいなら `check` です。

## Open questions への回答

### 1. `kind: spanner` か `kind: spanner_database` か

`kind: spanner` でよいと思います。短く、BigQuery と並べても自然です。

```yaml
kind: spanner
kind: bigquery
```

`spanner_database` にするなら `bigquery_dataset` も検討したくなり、全体が長くなります。config 名は短くし、plan 側で `resource_type: spanner_database` のように正規化すれば十分です。

### 2. `dialect: googlesql` は required か default か

default がよいです。GoogleSQL が標準なら、毎回書かせる必要はありません。

### 3. raw SQL `EXTERNAL_QUERY(...)` は reject か

reject でよいです。少なくとも stable v1 では `kind: external_query` を使わせる方が、plan / vet / explain-plan の品質を守りやすいです。

### 4. `projection_matrix` は通常 plan から外すか

外してよいです。通常 plan は semantic contract と diagnostic IDs に絞り、docs URL、docs checked date、matrix rows は `--audit` または `--verbose` に逃がすのがよいです。

### 5. `replace` は `operation: replace` のままか

`operation: replace` のまま、mutation-only として残すのがよいです。`operation: insert` に replacement semantics を混ぜると、むしろ危険です。

### 6. DTO cross-catalog sharing は global か query-level か

stable v1 では、global `dto.structs` を advanced feature として残し、README の最初の例からは外すのがよいです。

same-catalog query result sharing は同じ struct 名で許可し、cross-catalog / query-write sharing は `dto.structs` で明示、という分け方が単純です。

### 7. `migrate-config` は stable v1 前に必要か

不要だと思います。正式リリース前なら、compact config は破壊して構いません。

代わりに、README に migration table を出し、loader が旧 field を見た時に具体的な error を出すだけで十分です。

```text
unsupported field `schemas`; stable v1 uses `catalogs`
unsupported field `source`; stable v1 uses `catalog`
unsupported field `update_mask`; stable v1 uses `update.columns`
```

既存ユーザーが多くなってから `migrate-config` を追加しても間に合います。

## より単純な stable v1 grammar 案

破壊的変更を最大限使うなら、私は stable v1 を次の程度に絞ります。

```yaml
version: "1"

go:
  package: querydemo
  out: querydemo.sql.go

emit:
  spanner:
    query_methods: true
    mutations: true
    dml: true
  bigquery:
    row_loader: true
    table_schema: true

catalogs:
- name: app
  kind: spanner
  ddl: schema/spanner.sql

- name: analytics
  kind: bigquery
  project: example-project
  ddl: schema/bigquery.sql
  bindings:
    external_query_connections:
    - name: app_conn
      id: example-project.us.example-connection
      spanner_catalog: app

queries:
- name: GetSinger
  catalog: app
  kind: sql
  sql: |
    SELECT SingerId, FirstName
    FROM Singers
    WHERE SingerId = @SingerId
  result:
    cardinality: one
    struct: SingerRow

- name: FederatedSingerIDs
  catalog: analytics
  kind: external_query
  binding: app_conn
  inner_sql: |
    SELECT SingerId
    FROM Singers
  outer_sql: |
    SELECT * FROM __external__
  result:
    cardinality: many
    struct: FederatedSingerIDRow

writes:
- name: UpsertSingerName
  catalog: app
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
  emit:
  - mutation
  - dml
```

この案のポイントは、以下です。

- `dialect: googlesql` は省略。
- `surfaces` ではなく `emit`。
- false flag は書かない。
- `queries[].query.kind` ではなく `queries[].kind`。
- `input.struct` ではなく `input: SingerWrite`。
- `key.columns` ではなく `key: [...]`。
- `update.mask.columns` ではなく `update.columns`。
- DTO sharing は最初の例に出さない。
- external dataset は別 example にする。
- access verification は external evidence がある時だけ書く。

## 削らない方がよいもの

さらに単純にしたいとしても、次は削らない方がよいです。

- `catalogs` への rename。
- `external_query` と external dataset の分離。
- raw SQL `EXTERNAL_QUERY(...)` の原則 reject。
- write branch semantics、特に `insert` / `update` / `conflict` の分離。
- `replace` mutation-only。
- explicit update columns / update mask requirement。
- `explain-plan` の stable YAML/JSON。
- diagnostic ID の安定化。
- suppression が plan facts を消さないという方針。

ここは safety と reviewability の核なので、短くしすぎると元の問題が戻ります。

## 最終提案

cleanup plan は採用してよいです。ただし、stable v1 としては次の修正を入れるとさらに分かりやすくなります。

1. README の最初の config を Spanner-only minimal example にする。
2. `surfaces` は `emit` または `outputs` に改名する。
3. `dialect: googlesql` は default にする。
4. `queries[].query.kind` は `queries[].kind` に flatten するか再検討する。
5. `access.verification.status: not_checked` は config から省略し、plan で出す。
6. external dataset config では `database_role` / `access.mode` の重複を減らす。
7. `update.mask.columns` は `update.columns` にできないか検討する。
8. DTO sharing policy は advanced example に逃がす。
9. `migrate-config` は不要。旧 field に対する明確な error と migration table で十分。

この程度まで削ると、stable v1 の仕様はかなり読みやすくなります。計画案の安全性は保ちつつ、README を読んだ利用者が「これは重いフレームワークではなく、宣言した SQL / write を型安全にする generator だ」と理解しやすくなると思います。
