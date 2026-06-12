# `spanner-query-gen` external dataset round 6 feedback

## 前提

最新の round 5 response / README / DESIGN を前提に再レビューします。ただし、今回も実装コード・実行ログ・test output は共有されていないため、`Claimed implemented` や test 名は外部レビューで検証済みの事実ではなく、内部作業ツリー上の主張として扱います。

今回の返答で良かった点は、ここを明確に分けたことです。特に、test 名だけでなく expected diagnostic / expected plan shape / minimal output snippet を併記する方針は、Markdown だけのレビューとしてかなり良いです。

結論として、大きな設計転換は不要です。external dataset を `EXTERNAL_QUERY` の別記法ではなく BigQuery catalog binding として扱う方針、`cloud_resource_connection` と `EXTERNAL_QUERY` connection mapping を分ける方針、`access_verification` を observed fact ではなく evidence / attestation として扱う方針は、このまま維持してよいと思います。

ただし、ここから先は **CI で機械的に使える契約** として、いくつか詰めた方がよい点があります。

## 追加でフィードバックしたいこと

### 1. `diagnostics.id: planning-error` は外部レビュー・CI には粗すぎる

expected output の例で、location mismatch や database role conflict の `diagnostics.id` が `planning-error` になっています。これは人間向けには分かりますが、CI や downstream tool が分岐するには粗すぎます。

`planning-error` は category / stage として残してよいですが、`id` は具体的な rule / error identity にした方がよいです。

```yaml
diagnostics:
- id: external-dataset-connection-location-mismatch
  stage: external_dataset_binding
  severity: error
  suppressible: false
  suppressed: false
  subject: schemas.analytics_bigquery.spanner_external_datasets.analytics_spanner.access.cloud_resource_connection
  message: external dataset analytics_spanner: Cloud Resource connection location "eu" conflicts with dataset location "US"
```

同様に、少なくとも以下は専用 ID にすることを勧めます。

```text
external-dataset-connection-location-mismatch
external-dataset-database-role-conflict
external-dataset-external-source-unrecognized
external-dataset-access-unverified
external-dataset-unsupported-column-omitted
external-dataset-named-schema-table-omitted
external-dataset-dml-target-unsupported
federated-placeholder-invalid-position
federated-placeholder-missing
federated-placeholder-duplicate
```

ここはかなり重要です。`vet --output json/yaml` を CI 契約にするなら、利用者は message string ではなく diagnostic ID で判定したくなります。

### 2. plan の `location` は canonical 値に寄せた方がよい

現在の example では次のようになっています。

```yaml
location: us
location_metadata:
  configured: us
  canonical: US
```

この形でも読めますが、machine-readable plan の top-level `location` が canonical ではないと、plan consumer が毎回 `location_metadata.canonical` を見に行く必要があります。おすすめは、canonical value を主値にし、入力値は metadata に閉じ込める形です。

```yaml
location: US
location_metadata:
  configured: us
  canonical: US
  canonicalization: bigquery_location
```

regional location も同じです。

```yaml
location: us-central1
location_metadata:
  configured: us-central1
  canonical: us-central1
```

`cloud_resource_connection_metadata.parsed_location_canonical` と比較する側も、canonical values だけを比較できるようになります。

### 3. `external_source` parser は公式の no-role 形式も明記してほしい

recognized format を次のように書いている点は良いです。

```text
google-cloudspanner://[DATABASE_ROLE@]/projects/...
```

ただし、Google Cloud の公式ドキュメントの例では、role 付きの `google-cloudspanner://admin@/projects/...` と、role なしの `google-cloudspanner:/projects/...` の両方が示されています。したがって、parser の accepted grammar も明示した方が安全です。

提案:

```text
recognized external_source grammar:

1. google-cloudspanner://<database_role>@/projects/<project>/instances/<instance>/databases/<database>
2. google-cloudspanner:/projects/<project>/instances/<instance>/databases/<database>

All other schemes or slash forms are preserved as opaque metadata and produce
external-dataset-external-source-unrecognized.
```

もし `google-cloudspanner:///projects/...` のような別形も実装上受けるなら、それも accepted grammar に入れるべきです。逆に受けないなら、公式例との違いを README に明記した方がよいです。

### 4. `verification_hint` と `verification_evidence` の競合時 semantics を決める

`verification_hint` と `verification_evidence` を分けたのは良い判断です。ただし、両方が設定されたときの precedence / conflict handling がまだ曖昧です。

例えば次の config はどう扱うべきでしょうか。

```yaml
access:
  verification_hint: not_checked
  verification_evidence:
    status: verified
    source: external_evidence
    verifier: terraform-plan
    checked_at: "2026-05-04T10:30:00Z"
```

これは latest example と同じ方向ですが、`hint` と `evidence.status` が意味的に食い違って見える可能性があります。おすすめは、resolved status と configured hint を plan で分けることです。

```yaml
access_verification:
  resolved_status: verified
  source: external_evidence
  configured_hint: not_checked
  evidence:
    verifier: terraform-plan
    checked_at: "2026-05-04T10:30:00Z"
    volatile: true
  independently_verified_by_generator: false
```

また、明らかに矛盾する入力には warning か error を出すべきです。

```yaml
access:
  verification_hint: verified
  verification_evidence:
    status: failed
    source: external_evidence
```

この場合は、少なくとも warning が必要です。

```yaml
diagnostics:
- id: external-dataset-access-verification-conflict
  stage: external_dataset_binding
  severity: warning
  suppressible: true
  message: verification_hint "verified" conflicts with verification_evidence.status "failed"; evidence status is used in the resolved plan
```

### 5. stable plan output の方針を早めに仕様化した方がよい

`checked_at` を `volatile: true` とする方針は良いです。ただ、volatile になり得るものは `checked_at` だけではありません。

少なくとも以下は golden test / CI snapshot を壊しやすいです。

```text
checked_at
docs_last_checked
generator version / build metadata
generated_at
absolute local file paths
source file mtime, if included
```

そのため、`--stable` が未実装でも、先に仕様だけ置いた方がよいです。

```text
explain-plan --output yaml:
  includes volatile audit metadata

explain-plan --output yaml --stable:
  omits or normalizes volatile fields
  keeps semantic plan fields and diagnostic IDs
```

また、projection matrix の provenance は `docs_last_checked` だけでなく、公式ドキュメント自体の `last_updated` も持つとレビューしやすいです。

```yaml
projection_matrix:
  source: google_cloud_docs
  source_url: https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets
  docs_last_updated: "2026-05-01"
  docs_last_checked: "2026-05-04"
  generator_matrix_version: 1
```

`docs_last_checked` は generator 側の volatile audit metadata、`docs_last_updated` は参照した仕様の版、という位置付けにできます。

### 6. `projection_loss` は「relation が lossy」と「query result が affected」を分ける

explicit-column query でも relation provenance を残す方針は良いです。ただし、単に `projection_loss: true` だけだと、reviewer は「この query result に何か欠落があるのか」と誤解しやすいです。

例えば次の query は、underlying Spanner table に omitted column があっても、query 自体は BigQuery-visible columns だけを明示的に参照しています。

```sql
SELECT SingerId, FirstName
FROM analytics_spanner.Singers
```

この場合は、relation 自体は lossy projection 由来だが、selected output は directly affected ではない、と分けた方がよいです。

```yaml
relations:
- sql_path: analytics_spanner.Singers
  source: spanner_external_dataset_projection
  projection:
    relation_has_omitted_columns: true
    selected_output_affected: false
    omitted_columns:
    - Singers.SearchName
```

`SELECT *` の場合は次のようにできます。

```yaml
star_expansion:
  projection:
    relation_has_omitted_columns: true
    selected_output_affected: true
    omitted_columns:
    - Singers.SearchName
```

この分離により、plan facts は残しつつ、diagnostic severity を過剰に上げずに済みます。

### 7. hidden column omission は default warning ではなく info でもよい

external dataset projection matrix では `Hidden column` の default severity が warning になっています。これは安全側ではありますが、hidden columns が普通に存在する schema では warning noise になりやすいです。

`unsupported column such as PROTO / ENUM / STRUCT / TOKENLIST` は warning で良いと思います。一方、hidden column は BigQuery-visible catalog に出さないこと自体が自然なので、default は info に落としてもよいです。

```text
Hidden column: info by default
Unsupported visible Spanner column type: warning by default, or error with unsupported_columns: error
```

利用者が hidden column omission も fatal にしたい場合は、将来の rule policy で `external_dataset_hidden_columns: warning|error` を足せばよいです。

### 8. external dataset の source / target role analysis は AST role で明文化する

README / DESIGN では「external dataset tables are read-only targets」「source としては将来許容し得る」と整理されています。この方向は正しいです。Google Cloud docs でも、external dataset の table は通常の BigQuery table と同じように query できる一方、data modification operations は unsupported です。また、query results は BigQuery table として保存したり、join / merge に使ったりできるとされています。

したがって、将来 BigQuery execution surface を入れるなら、単に “DML allowed / disallowed” ではなく、AST 上の role を分けるべきです。

```text
Allowed:
  SELECT source relation
  CTAS source relation into native BigQuery table
  INSERT INTO native_table SELECT ... FROM external_dataset.table
  MERGE native_table USING external_dataset.table
  CREATE VIEW native_dataset.view AS SELECT ... FROM external_dataset.table

Rejected:
  INSERT INTO external_dataset.table
  UPDATE external_dataset.table
  DELETE FROM external_dataset.table
  MERGE external_dataset.table USING ...
  TRUNCATE TABLE external_dataset.table
  CREATE TABLE external_dataset.new_table
  CREATE VIEW external_dataset.view
  CREATE MATERIALIZED VIEW external_dataset.mv
  ALTER / DROP metadata target under external dataset
```

Plan には relation role を残すとよいです。

```yaml
relations:
- sql_path: analytics_spanner.Singers
  source: spanner_external_dataset_projection
  role: select_source
  writable_target: false
```

DML target の場合は suppressible warning ではなく planning error です。

```yaml
diagnostics:
- id: external-dataset-dml-target-unsupported
  stage: bigquery_analysis
  severity: error
  suppressible: false
  subject: query.CopyBackToExternalDataset
```

### 9. projection matrix の各行に `evidence_kind` を持たせる

`projection_matrix.source: google_cloud_docs` は良いですが、matrix の全項目が同じ根拠ではありません。

例えば、次のように分かれます。

```text
Documented by Google Cloud docs:
  - default schema tables only
  - keys not visible to BigQuery
  - unsupported columns not accessible
  - DML / metadata mutation unsupported
  - INFORMATION_SCHEMA unsupported
  - Read API / Write API unsupported
  - Data Boost default

Generator safety policy:
  - case-folded table collision is planning error
  - case-folded column collision is planning error
  - relation provenance is always retained
  - vet.disable does not remove projection facts

Currently inferred / to be verified:
  - exact runtime loader caveats for external dataset NUMERIC / TIMESTAMP
```

そのため、matrix row ごとに根拠種別を持つと、将来 docs が変わったときにも追いやすいです。

```yaml
projection_matrix:
  rows:
  - object: named_schema_table
    behavior: omit
    default_severity: warning
    evidence_kind: documented
    source_url: https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets
  - object: case_folded_column_collision
    behavior: reject
    default_severity: error
    evidence_kind: generator_safety_policy
```

### 10. PostgreSQL-dialect external dataset の unsupported diagnostic を具体化する

公式ドキュメントでは、Spanner external dataset は GoogleSQL または PostgreSQL database にリンクできるとされています。一方、この generator は Spanner GoogleSQL DDL projection に限定する方針です。この割り切りは妥当です。

ただし、README / DESIGN には、どう検出し、どう失敗するかを少し足すとよいです。

```yaml
diagnostics:
- id: external-dataset-postgresql-dialect-unsupported
  stage: external_dataset_projection
  severity: error
  suppressible: false
  message: external dataset analytics_spanner uses Spanner PostgreSQL dialect, but spanner-query-gen currently projects only Spanner GoogleSQL DDL
```

`spanner_source` が明示的に GoogleSQL DDL catalog を指している場合は問題ありません。将来 live_probe で actual database dialect が PostgreSQL と分かった場合は、config / live mismatch として別 diagnostic にできます。

### 11. `Claimed implemented` 表には「外部レビュー用 bundle」を追加してほしい

今回の response は、実装未共有の前提を正しく扱っています。次の段階では、test 名に加えて、外部 reviewer が仕様として読める bundle を各 behavior に 1 つずつ付けるとさらに良いです。

```text
Behavior: explicit omitted column reference is analyzer error

External review bundle:
  - minimal config snippet
  - query snippet
  - command: spanner-query-gen vet --config omitted-column.yaml --output json
  - expected exit code: 1
  - expected stdout JSON diagnostic ID: external-dataset-omitted-column-reference
  - expected stderr: empty
  - expected plan field, if any
```

これは実装証明ではありませんが、Markdown だけでも仕様契約として十分にレビューできます。

## 細かい文言調整

### `access: cloud_resource_connection` は `access.mode` と混同しやすい

expected plan snippet に以下があります。

```yaml
access: cloud_resource_connection
cloud_resource_connection: example-project.us.example-connection
```

README では `access.mode` の canonical values が説明されているので、plan でも `access_mode` にした方が混乱が少ないです。

```yaml
access_mode: cloud_resource_connection
cloud_resource_connection: example-project.us.example-connection
```

または block 化します。

```yaml
access:
  mode: cloud_resource_connection
  cloud_resource_connection: example-project.us.example-connection
```

### `database_role_source: config_and_external_source` は分解できるとよい

`database_role_source: config_and_external_source` は読みやすいですが、diff / tooling では少し扱いづらいです。

```yaml
database_role:
  value: reader
  sources:
  - config
  - external_source
  consistency: matched
```

role mismatch 時も同じ shape で表現できます。

```yaml
database_role:
  config: writer
  external_source: reader
  consistency: conflict
```

### `vet.disable` は warning presentation だけでなく diagnostic status を残す

「projection facts は消さない」という方針は非常に良いです。加えて、suppressed diagnostic 自体も plan / vet JSON に残すべきです。

```yaml
diagnostics:
- id: external-dataset-access-unverified
  severity: warning
  suppressible: true
  suppressed: true
  suppression:
    scope: schemas.analytics_bigquery.spanner_external_datasets.analytics_spanner
    reason: Static CI verifies external dataset access outside this generator.
    owner: analytics-platform
    expires: "2026-12-31"
```

これは既に近い方針になっていますが、`diagnostic policy にだけ作用する` という説明と合わせて明文化しておくとよいです。

## 参考にした外部仕様

2026-05-04 時点で Google Cloud docs を確認すると、Spanner external dataset は BigQuery dataset-level federation として説明され、BigQuery から Spanner tables を直接 query できます。一方で、external dataset table への data modification operations は unsupported で、Data Boost が default かつ変更不可です。また、default Spanner schema の table のみ accessible、Spanner primary / foreign key metadata は BigQuery 側に visible ではない、unsupported Spanner column は BigQuery 側で accessible ではない、`INFORMATION_SCHEMA` / Read API / Write API も unsupported とされています。

`EXTERNAL_QUERY` については、inner query の `ORDER BY` は BigQuery output order を保証せず、Spanner `STRUCT` は federated query output として unsupported、Spanner `TIMESTAMP` は nanoseconds truncated、unsupported type は query failure になります。

Spanner DML については、`INSERT OR UPDATE` は指定列のみを update し、未指定列は unchanged ですが、`ON UPDATE` expression column は例外的に server-side effect を持ちます。また、DML と mutations は Read Your Writes / constraint checking / transaction ordering が異なるため、混在を避ける方針は引き続き妥当です。

## 結論

今回の response はかなり良く、設計の大枠に対する追加の反対はありません。次に詰めるべきなのは、機能追加ではなく **診断 ID、plan schema、volatile metadata、verification evidence、relation role** の安定化です。

特に優先順位は次です。

1. `diagnostics.id` を具体化し、`planning-error` を generic category に下げる。
2. top-level `location` を canonical value にする。
3. official `external_source` no-role form を accepted grammar に入れる。
4. `verification_hint` / `verification_evidence` の precedence と conflict handling を決める。
5. `--stable` plan output の volatile field policy を仕様化する。
6. `projection_loss` を relation-level と selected-output-level に分ける。
7. external dataset DML / metadata target rejection を AST role で定義する。
8. projection matrix rows に documented / generator policy / inferred の evidence kind を持たせる。

実装が共有されていない前提では、これらを「実装済み」と主張する必要はありません。README / DESIGN / expected output の契約として明記されていれば、外部レビューとしては十分に前進しています。


## Sources checked

- Google Cloud: Create Spanner external datasets — https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets
- Google Cloud: Federated query functions / `EXTERNAL_QUERY` — https://docs.cloud.google.com/bigquery/docs/reference/standard-sql/federated_query_functions
- Google Cloud Spanner: GoogleSQL DML syntax — https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax
- Google Cloud Spanner: Compare DML and mutations — https://docs.cloud.google.com/spanner/docs/dml-versus-mutations
