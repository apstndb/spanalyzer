# `spanner-query-gen` 追加フィードバック: 最新対応への返答と external dataset 方針

対象:

- `spanner-query-gen-additional-feedback-response-ja.md`
- `README(3).md`
- `DESIGN(3).md`

## 結論

今回の対応はかなり良いです。前回までの主要な懸念、つまり DML `replace` の停止、`update_mask` 省略の危険性、`insert_or_update` の insert branch validation、plan metadata、structured warnings、federation compatibility matrix、read/write field role の分離は、設計としてかなり健全に取り込まれています。

追加で大きな設計転換を求める必要はありません。今後伝えるべきことがあるとすれば、主に以下です。

1. `vet` / warnings / failure policy の「今できること」と「将来やること」を README と DESIGN でさらに明確化する。
2. `auto_all_non_key_columns` の表現を compact / structured config で揺らさない。
3. external dataset を将来サポートする意思があるなら、`EXTERNAL_QUERY` の拡張ではなく、**別の BigQuery catalog integration** として設計する。
4. `external_schemas` という名前は、将来 external dataset と紛らわしくなるため、structured config ではより具体的な名前に変える余地を残す。

## そのまま返せるコメント案

対応ありがとうございます。今回の変更はおおむね妥当です。特に、DML `replace` を mutation-only に戻したこと、`update_mask` を必須化したこと、`insert_or_update` の insert branch validation を入れたこと、plan に write column capability / server-side update effects / structured warnings を入れたことは、この generator が危険なコードを出さないための重要な前進だと思います。

追加で見るべき点は、大きくは external dataset と、現在の `vet` / warning policy の明確化です。

## 追加フィードバック

### 1. `vet` は「plan validation」と「rule engine」を分けて書いた方がよい

README では `vet` が同じ plan を使って validation すると書かれています。一方、レビュー回答では full vet rule engine、rule registry、severity-based fail policy はまだ deferred とされています。この差分は大きな矛盾ではありませんが、利用者から見ると「今の `vet` は何で失敗するのか」が少し曖昧です。

README か DESIGN に、現時点の `vet` を以下のように分けて書くとよいです。

```text
Current vet:
  - builds the same resolved plan as code generation
  - fails on planning errors
  - emits structured warnings
  - supports scoped suppressions for implemented warning rules

Future vet rule engine:
  - per-rule severity policy
  - --strict / --warnings-as-errors
  - project-level rule configuration
  - rule registry and custom rules
```

これにより、`structured warnings and suppressions are implemented` と `full rule engine is deferred` が両立します。

### 2. `auto_all_non_key_columns` の config 表現を固定する

README では compact config の例として次の形が出ています。

```yaml
update_mask: [auto_all_non_key_columns]
```

一方、DESIGN の structured config では次のような形です。

```yaml
update:
  mask: auto_all_non_key_columns
```

この違いは小さく見えますが、loader と plan schema の実装で曖昧になりやすいです。おすすめは、structured config では sentinel と column list を同じ型にしないことです。

```yaml
update:
  mask:
    columns:
    - FirstName
    - LastName
```

```yaml
update:
  mask:
    all_non_key_columns: true
```

または短くするなら、次のように `strategy` を分ける方が明確です。

```yaml
update:
  strategy: explicit_columns
  columns:
  - FirstName
  - LastName
```

```yaml
update:
  strategy: all_non_key_columns
```

`auto_all_non_key_columns` は便利ですが、文字列 sentinel を list に混ぜると、column name と衝突した場合の扱い、YAML 型、plan 表現、vet rule が複雑になります。特に Spanner は quoted identifier を許すので、将来的に sentinel と実 column 名の衝突を完全に無視できません。

### 3. README の compact example は validation 後も必ず valid にする

README の `SaveSinger` は compact `insert_or_update` の例として `update_mask: [FirstName]` を使っています。

```yaml
- name: SaveSinger
  operation: insert_or_update
  update_mask:
  - FirstName
```

今回の対応では、compact `insert_or_update` の insert value set は `primary key + update_mask` として扱い、`NOT NULL` かつ server-provided value がない required column が足りない場合は error になります。したがって、サンプル DDL の `Singers` に `LastName STRING(MAX) NOT NULL` のような列がある場合、この README example は invalid になります。

サンプルは次のどちらかに寄せると安全です。

```yaml
update_mask:
- FirstName
- LastName
```

または、README に「この例では `LastName` は nullable または default/server-provided value を持つ」と注記します。

README の例は利用者が最初にコピーするため、強化した validation と常に整合していることが重要です。

### 4. `external_schemas` という名前は将来 external dataset と衝突しやすい

現状の `external_schemas` は、BigQuery `EXTERNAL_QUERY(connection, sql)` の connection ID を Spanner schema に対応付ける概念です。これは設計として正しいです。

ただし、将来 BigQuery Spanner external dataset もサポートしたくなるなら、`external_schemas` という名前は少し広すぎます。structured config では、できれば次のように意味を狭くした名前に移行する余地を残した方がよいです。

```yaml
schemas:
- name: analytics_bigquery
  dialect: bigquery
  external_query_connections:
  - connection: example-project.us.example-connection
    spanner_source: app_spanner
```

external dataset は別枠にします。

```yaml
schemas:
- name: analytics_bigquery
  dialect: bigquery
  spanner_external_datasets:
  - dataset: example-project.analytics_spanner
    spanner_source: app_spanner
```

つまり、`EXTERNAL_QUERY` と external dataset はどちらも Spanner federation ですが、config 上は別名にした方がよいです。

## external dataset を将来サポートしたい場合の方針

### まず結論

external dataset support は入れてよいと思います。ただし、`EXTERNAL_QUERY` support の延長として扱うべきではありません。

`EXTERNAL_QUERY` は query-level federation です。BigQuery SQL の中に connection ID と inner Spanner SQL があり、generator は inner SQL を Spanner catalog で解析し、その結果を BigQuery の temporary table として扱います。

一方、Spanner external dataset は dataset-level federation です。BigQuery 側では Spanner tables が普通の BigQuery tables のように見え、query は通常の BigQuery SQL として書かれます。inner SQL は存在しません。

したがって、将来サポートするなら、設計上の位置付けは次です。

> external dataset は `queries[].federated` の別記法ではなく、BigQuery analyzer に渡す catalog を拡張するための schema/catalog binding である。

### なぜ別扱いが必要か

`EXTERNAL_QUERY` と external dataset では、静的解析の単位が違います。

| 観点 | `EXTERNAL_QUERY` | Spanner external dataset |
| --- | --- | --- |
| Federation の単位 | query expression | BigQuery dataset |
| Spanner SQL | `inner_sql` として明示 | なし |
| BigQuery での見え方 | table-valued function の結果 | 通常の dataset tables のように見える |
| Generator の主な仕事 | inner SQL 解析、type conversion、outer SQL 解析 | external dataset table catalog の構築、BigQuery SQL 解析 |
| Ordering warning | inner `ORDER BY` と outer `ORDER BY` の関係を見る | 該当しない |
| Query priority | `EXTERNAL_QUERY` option として扱える | external dataset では固定的に扱われる |
| Write | Spanner inner SQL 側も generator では read query として扱う | external dataset tables は read-only |

そのため、external dataset を `federated.inner_sql` の variant に入れると、後で混乱します。

### 推奨 config 形

最初は BigQuery schema の catalog extension として表現するのがよいです。

```yaml
schemas:
- name: app_spanner
  dialect: spanner
  ddl: schema/spanner.sql
  proto_descriptors:
  - schema/order_descriptors.pb

- name: analytics_bigquery
  dialect: bigquery
  ddl: schema/bigquery.sql

  external_query_connections:
  - connection: example-project.us.example-connection
    spanner_source: app_spanner

  spanner_external_datasets:
  - dataset: example-project.analytics_spanner
    spanner_source: app_spanner
    access: unknown        # euc | cloud_resource | unknown
    unsupported_columns: warn_and_omit
    table_name_case: insensitive
```

query は通常の BigQuery SQL として書きます。

```yaml
queries:
- name: JoinNativeBQAndSpannerExternalDataset
  source: analytics_bigquery
  sql: |
    SELECT
      c.CustomerId,
      s.SingerId
    FROM `example-project.analytics.customers` AS c
    JOIN `example-project.analytics_spanner.Singers` AS s
      ON s.SingerId = c.FavoriteSingerId
  result: many
  result_struct: CustomerSingerRow
```

ここで generator は `analytics_spanner.Singers` を `app_spanner` DDL から BigQuery-visible table として投影します。

### plan に出すべき情報

external dataset をサポートするなら、plan に以下を出すべきです。

```yaml
catalog_bindings:
- kind: spanner_external_dataset
  bigquery_dataset: example-project.analytics_spanner
  spanner_source: app_spanner
  access: unknown
  execution:
    data_boost: forced
    query_priority: medium
    writable: false
  limitations:
  - default_spanner_schema_only
  - primary_and_foreign_keys_not_visible_to_bigquery
  - unsupported_spanner_columns_not_visible
  - no_dml
  - no_read_api_or_write_api
  - no_information_schema
  visible_tables:
  - name: Singers
    source_table: Singers
    name_matching: case_insensitive
    columns:
    - name: SingerId
      spanner_type: INT64
      bigquery_type: INT64
      visible: true
    - name: SomeProtoColumn
      spanner_type: PROTO
      visible: false
      reason: unsupported_bigquery_external_dataset_column
```

重要なのは、external dataset の table schema を「BigQuery から見える投影」として扱うことです。Spanner DDL には存在しても、BigQuery external dataset 側に見えない columns は query catalog から除外するか、strict mode では error にします。

### DDL-first 方針との整合

external dataset は BigQuery 側で tables が自動的に見えるため、live BigQuery introspection を使いたくなります。しかし、この tool の強みは DDL-first deterministic workflow です。したがって、通常 workflow は次にすべきです。

1. Spanner DDL を source of truth として読む。
2. BigQuery external dataset rules に基づき、BigQuery-visible catalog projection を作る。
3. BigQuery query をその projected catalog で解析する。
4. 必要なら optional verification として BigQuery API / `bq show` の結果と突き合わせる。

live introspection を primary workflow にすると、CI が credential / project / dataset state に依存してしまいます。これは現 DESIGN の DDL-first 方針と衝突します。

### external dataset 用の vet rules

最初に入れるなら、次の rules が有用です。

```text
external-dataset-named-spanner-schema
  Only default Spanner schema is visible through current BigQuery external dataset rules.

external-dataset-unsupported-column
  A Spanner column exists in DDL but is not visible in BigQuery external dataset projection.

external-dataset-write-attempt
  DML / metadata changes against external dataset tables are not supported.

external-dataset-information-schema
  INFORMATION_SCHEMA views are not supported for Spanner external datasets.

external-dataset-data-boost-permission-note
  Queries use Data Boost by default and require corresponding permissions.

external-dataset-keys-not-visible
  BigQuery does not expose Spanner primary/foreign keys for external dataset tables; generator may know them from DDL, but BigQuery query planning must not rely on them.

external-dataset-case-insensitive-table-name
  External dataset table names are case-insensitive; warn on ambiguous Spanner table names after case-folding.
```

これらは generator の安全性にも、利用者の運用理解にも効きます。

### phase placement

Roadmap に入れるなら、Phase 4 の後、BigQuery query method generation の前がよいです。

```text
Phase 4.5: BigQuery Spanner External Dataset Catalog Integration

- Keep EXTERNAL_QUERY support as the first federation UX.
- Add external dataset catalog bindings as a separate BigQuery schema integration.
- Build BigQuery-visible table projections from Spanner DDL.
- Reject or omit unsupported Spanner columns according to config.
- Keep external dataset tables read-only.
- Add explain-plan output for projected external dataset tables.
- Add vet rules for unsupported columns, named schemas, writes, INFORMATION_SCHEMA, and access/execution caveats.
- Do not generate BigQuery execution methods yet.
```

これなら、外部データセット対応は product boundary を壊しません。あくまで BigQuery catalog を増やすだけで、ORM や runtime query builder にはなりません。

### Non-Goal の文言を少し変える

今の DESIGN には external dataset が initial non-goal として書かれており、これは正しいです。ただ、将来対応の意思があるなら、次のように書くとより自然です。

```text
Do not support BigQuery Spanner external datasets initially.
The first federation UX focuses on EXTERNAL_QUERY because it exposes an explicit connection and inner SQL that the generator can analyze.
If added later, external dataset support should be a separate BigQuery catalog integration, not a federated-query shorthand.
```

この一文を足すだけで、将来の設計方向がかなり明確になります。

## external dataset support でやらない方がよいこと

### 1. `queries[].federated` に `mode: external_dataset` を足さない

これは避けた方がよいです。external dataset は query-level ではなく dataset-level の機能なので、query config に押し込むと、同じ dataset を使う複数 query で重複します。

### 2. BigQuery external dataset の作成を generator の責務にしない

external dataset の作成には project、location、IAM、connection、access delegation などのインフラ設定が関係します。`spanner-query-gen` が BigQuery dataset を作る CLI になると product boundary が広がりすぎます。

必要なら、生成ではなく docs / Terraform example / `vet` preflight note に留めるべきです。

### 3. external dataset を使えば Spanner DDL が不要になる、とはしない

BigQuery external dataset 側では Spanner tables が見えますが、この generator の primary workflow は DDL-first であるべきです。external dataset metadata を source of truth にすると、Spanner primary key、foreign key、generated columns、hidden columns、write capability など、Spanner-specific な解析情報が失われやすくなります。

### 4. PostgreSQL dialect Spanner external dataset を暗黙サポートしない

BigQuery Spanner external dataset は GoogleSQL database と PostgreSQL database にリンクできますが、この tool は現状 GoogleSQL/Spanner を中心にしています。PostgreSQL dialect Spanner まで広げるなら、別 dialect として明示的に扱うべきです。

```yaml
spanner_external_datasets:
- dataset: example-project.pg_spanner_ext
  spanner_source: app_spanner_pg
  spanner_dialect: postgresql
```

初期対応では GoogleSQL Spanner のみに限定するのが安全です。

## 追加で小さく確認したい点

### `result: exec` と write helper の関係

`result: exec` は query result cardinality として入っていますが、writes から生成される DML helper も execution metadata を返す可能性があります。将来の method generation で、query entry の `result: exec` と write entry の DML execution helper が混ざらないよう、plan 上では次のように分けた方がよいです。

```text
query.result_mode: one | maybe_one | many | exec
write.execution_mode: statement_only | execute_standard_dml | execute_partitioned_dml
```

DML `THEN RETURN` は既に別概念として扱う方針なので、この分離と相性が良いです。

### `vet` suppression の期限切れ behavior

`owner` / `expires` を plan に残すのは良いです。さらに、`expires` が過去日付のときの behavior を決めると CI で使いやすくなります。

```text
- expired suppression is warning by default
- --strict treats expired suppression as error
- invalid date format is always error
```

### plan diagnostics に source span を入れる

machine-readable plan が CI golden test に使われるなら、warning / error に config path と SQL location を持たせるとよいです。

```yaml
warnings:
- rule: cross-dialect-timestamp-truncation
  severity: warning
  source:
    config_path: queries[3].federated.inner_sql
    line: 12
    column: 8
  remediation: Cast or format the value if nanosecond precision matters.
```

これがあると、`vet` を実運用に入れたときの修正体験がかなり良くなります。

## 最終評価

現時点の方向はかなり良いです。特に、危険な default を stable release 前に潰す方針は正しいです。

external dataset については、サポート意思があるなら今すぐ実装する必要はありません。ただし、今のうちに DESIGN の Non-Goal / Roadmap / config naming に「将来入れるなら別 BigQuery catalog integration」と書いておく価値があります。

一番避けるべきなのは、external dataset を `EXTERNAL_QUERY` の単なる shorthand として扱うことです。そうではなく、BigQuery schema catalog に Spanner-derived external tables を追加する仕組みとして設計すれば、現在の DDL-first / SQL-is-contract / narrow generator という方針を壊さずに拡張できます。
