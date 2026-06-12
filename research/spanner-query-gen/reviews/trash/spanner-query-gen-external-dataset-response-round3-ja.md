# `spanner-query-gen` external dataset round 3 response

## Summary

今回の対応はかなり良いです。特に、BigQuery Spanner external dataset を `EXTERNAL_QUERY` の別記法にせず、BigQuery catalog binding として扱う方針を維持した点、`external_query_connections` と `spanner_external_datasets` を分離した点、`outer_sql` を `queries[].federated.outer_sql` に固定した点、`unsupported_columns` / `named_schema_policy` / access / Data Boost metadata を plan に載せる方針は妥当です。

追加で強く指摘するべき大きな設計転換はありません。ここからは、external dataset を実運用で安全に使うための **仕様固定・diagnostic・regression test** が中心です。

## 追加で反映してほしい点

### 1. Current Scope と Roadmap Phase 4.5 の関係を少し明確にする

最新 DESIGN では Current Scope に “BigQuery Spanner external dataset catalog bindings from Spanner DDL” が入り、Roadmap にも “Phase 4.5: BigQuery Spanner External Dataset Catalog Integration” が残っています。この両方が同時に存在すること自体は問題ありませんが、読む人によっては「もう実装済みなのか、Phase 4.5 で実装予定なのか」が少し曖昧に見えます。

おすすめは、Current Scope を次のように分けることです。

```text
Current implementation supports:
  - external dataset catalog binding declaration
  - static projection into BigQuery analyzer catalog
  - basic explain-plan / vet diagnostics

Planned external dataset enhancements:
  - richer field-level provenance
  - stricter vet rules
  - optional live verification
  - BigQuery execution methods, if added later
```

または Phase 4.5 を “Enhance BigQuery Spanner External Dataset Catalog Integration” に変えるだけでもよいです。

### 2. `unsupported_columns: omit` は良いが、query behavior を明文化する

`unsupported_columns: omit` を default にする判断には同意します。BigQuery external dataset の実サービスでも、BigQuery 側でサポートされない Spanner column は accessible にならないため、static catalog projection の default を実サービスに寄せるのは自然です。

ただし、`omit` の意味を次の 3 ケースで明確に分けてください。

```text
1. Projection build:
   unsupported column is omitted from projected BigQuery catalog.
   The plan emits a warning with source column, type, and remediation.

2. SELECT *:
   star expansion uses the projected BigQuery-visible columns only.
   The plan warns that some Spanner columns were omitted from the star result.

3. Explicit column reference:
   SELECT UnsupportedColumn FROM external_dataset.Table
   must be a planning error, not just a warning.
```

つまり、`unsupported_columns: omit` は「external dataset binding を作る段階では fatal にしない」という意味であり、「query が明示的に存在しない column を参照しても許す」という意味ではない、と固定した方が安全です。

同じ整理は `named_schema_policy: warn_and_omit` にも必要です。named schema の table を projection から omit するのはよいですが、user query がその table を明示参照した場合は unresolved table / unsupported table として planning error にするべきです。

### 3. table だけでなく projected column の case-folding collision も検証する

README では external dataset table names are case-insensitive として、Spanner table names が case-fold 後に衝突する場合は reject すると書かれています。この対応は良いです。

追加で、projected BigQuery table の column 名についても case-folding collision を検証してください。BigQuery は column names が case だけ違っても duplicate と見なします。したがって、Spanner 側で `Foo` と `foo` のような列が存在する場合、external dataset projection では BigQuery catalog として安全に表現できません。

推奨 diagnostic:

```text
external_dataset_column_name_collision
  dataset: example-project.analytics_spanner
  table: Singers
  columns:
    - Foo
    - foo
  reason: BigQuery column names are case-insensitive / duplicates after case-folding.
  remediation: Rename one Spanner column or exclude the table from external dataset projection.
```

将来 `STRUCT` / nested field を external dataset 側で扱う場合は、nested field 名にも同じ方針を適用できるようにしておくとよいです。

### 4. external dataset 用の type compatibility matrix も別に持つ

DESIGN には `EXTERNAL_QUERY` の Spanner-to-BigQuery matrix が入り、`STRUCT` error、`TIMESTAMP` truncation warning、`PROTO` / `ENUM` error などが整理されています。これは良いです。

ただし、external dataset projection は `EXTERNAL_QUERY` output conversion と severity が違います。したがって、同じ matrix を流用するのではなく、別 matrix を持つ方が読みやすくなります。

例:

```text
Spanner external dataset projection matrix

Spanner type        BigQuery projected column         Default behavior
BOOL                BOOL                              expose
INT64               INT64                             expose
FLOAT64             FLOAT64                           expose
NUMERIC             NUMERIC                           expose, precision notes in plan
STRING              STRING                            expose
BYTES               BYTES                             expose
DATE                DATE                              expose
TIMESTAMP           TIMESTAMP                         expose, nanosecond truncation warning if applicable
JSON                JSON                              expose, loader strategy must be explicit
ARRAY<T>            ARRAY<T>                          recurse on T
PROTO               not projected                     omit or error
ENUM                not projected                     omit or error
unsupported type    not projected                     omit or error
```

重要なのは、`EXTERNAL_QUERY` では unsupported output が query failure なので `error`、external dataset では unsupported column が BigQuery 側に出ないので default `omit`、という違いを plan 上でも維持することです。

### 5. read-only の意味を「target として書けない」に限定して表現する

DESIGN の “Keep external dataset tables read-only” は正しいですが、将来 BigQuery DML / query methods を扱う場合、少し精密に書いた方がよいです。

BigQuery external dataset table は data modification target にはできません。一方で、external dataset table を `SELECT` source として読み、その結果を通常の BigQuery table に保存したり、通常の BigQuery table への `MERGE` / `INSERT ... SELECT` の source に使うことは概念上あり得ます。

したがって analyzer rule は次のようにすると安全です。

```text
Reject:
  INSERT INTO external_dataset.Table ...
  UPDATE external_dataset.Table ...
  DELETE FROM external_dataset.Table ...
  MERGE external_dataset.Table AS target ...
  CREATE TABLE external_dataset.NewTable ...

Allow, if BigQuery execution methods are ever supported:
  INSERT INTO native_dataset.Table
  SELECT ... FROM external_dataset.Table

  MERGE native_dataset.Table AS target
  USING external_dataset.Table AS source
  ...
```

“read-only” を雑に “DML query に external dataset が出たら全部 reject” にすると、将来の BigQuery method surface で過剰拒否になります。

### 6. `access` / `access_verified` は plan metadata としてもう少し型を分ける

`access: unknown|euc|cloud_resource` と `access_verified` を plan metadata として載せる方針はよいです。ただし、`access_verified: false` は「検証して false だった」のか「検証していない」のかが読み手に伝わりにくいです。

おすすめは、将来 plan では tri-state にすることです。

```yaml
access:
  mode: cloud_resource        # unknown | euc | cloud_resource
  database_role: reader
  verification:
    status: not_checked       # not_checked | verified | mismatch | failed
    checked_at: null
  data_boost:
    required: true
    permission_verified: not_checked
```

このツールは live infrastructure を作らない・検証しない方針なので、default は `not_checked` が一番正直です。将来 optional live verification を入れる場合も、この shape なら自然に拡張できます。

### 7. 実 BigQuery external dataset の resource identity を任意 metadata として持てるようにする

今の `spanner_external_datasets` は `dataset` と `spanner_source` だけで static projection できます。ただ、plan review の観点では、実際にどの Spanner database に link される想定なのかが見えた方が安全です。

generator が Terraform や IAM を作らない方針は維持しつつ、任意 metadata として以下を持てるようにするとよいです。

```yaml
spanner_external_datasets:
- dataset: analytics_spanner
  spanner_source: app_spanner
  external_source: google-cloudspanner://reader@/projects/example/instances/prod/databases/app
  location: us
  access: cloud_resource
  connection: example-project.us.spanner-reader
```

この情報は codegen には必須ではありませんが、`explain-plan` で以下を確認できます。

```text
- projected DDL source: app_spanner
- expected external source: projects/example/instances/prod/databases/app
- expected BigQuery dataset: example-project.analytics_spanner
- expected location: us
- access mode: cloud_resource
- verification: not_checked
```

これにより、「DDL は dev DB 由来だが external dataset は prod DB を指していた」のような構成ミスを review で見つけやすくなります。live verification なしでも、少なくとも plan 上の意図は明確になります。

### 8. `database_role` がある場合、DDL-first catalog が過大推定する可能性を warning にする

external dataset に `database_role` を設定しても、generator は DDL から全 table / column を projection しています。これは DDL-first workflow として妥当ですが、実際の database role が table / column visibility を制限している場合、static projection は runtime visibility を過大推定する可能性があります。

そのため、`database_role` が設定され、かつ live verification をしていない場合は plan warning を出すとよいです。

```text
external_dataset_role_visibility_not_verified
  dataset: example-project.analytics_spanner
  database_role: reader
  reason: Static projection is based on DDL, not role-filtered live metadata.
  remediation: Use access_verified/live verification later, or provide an explicit allowlist.
```

将来の escape hatch として、role-based allowlist を config に持たせることも考えられます。

```yaml
spanner_external_datasets:
- dataset: analytics_spanner
  spanner_source: app_spanner
  database_role: reader
  visible_tables:
  - Singers
  - Albums
```

これは必須ではありませんが、database role を使う環境ではかなり有用です。

### 9. `vet` suppression の scope を schema / external dataset binding にも広げる

今の README の suppression example は `queries[]` に載っています。query-level warning には十分ですが、external dataset では schema-level / binding-level warning が増えます。

例えば以下は query ではなく binding の warning です。

```text
- unsupported Spanner column omitted
- named schema table omitted
- database role visibility not verified
- Data Boost permission not verified
- case-folding collision
```

そのため、suppression は query-level だけでなく schema binding-level にも置けるようにした方がよいです。

```yaml
schemas:
- name: analytics_bigquery
  dialect: bigquery
  spanner_external_datasets:
  - dataset: analytics_spanner
    spanner_source: app_spanner
    vet:
      disable:
      - rule: external-dataset-role-visibility-not-verified
        reason: Verified by Terraform policy in infra repo.
        owner: analytics-platform
        expires: "2026-12-31"
```

または structured config の `rules.suppressions` で selector を持たせてもよいです。

```yaml
rules:
  suppressions:
  - selector:
      schema: analytics_bigquery
      external_dataset: analytics_spanner
    rule: external-dataset-role-visibility-not-verified
    reason: Verified outside this repository.
    owner: analytics-platform
    expires: "2026-12-31"
```

### 10. `vet --output json|yaml` の exit code と stdout/stderr 契約を固定する

`vet` で warning summary と machine-readable output を出す対応は良いです。追加で、CI UX のために exit code と stream の契約を固定してください。

おすすめ:

```text
spanner-query-gen vet --output text
  stdout: empty or human summary
  stderr: diagnostics / warning summary
  exit 0: no planning errors
  exit 1: planning errors
  warnings do not affect exit until --strict or warnings-as-errors

spanner-query-gen vet --output json|yaml
  stdout: machine-readable diagnostics
  stderr: reserved for fatal CLI errors only
  exit 0: no planning errors
  exit 1: planning errors
```

machine-readable mode で warnings を stderr にも出すと CI parser が扱いづらくなるので、JSON/YAML の場合は stdout に集約するのがよいです。

### 11. `__external__` placeholder の置換フェーズを明記する

`federated.outer_sql` の placeholder を `__external__` に固定する方針はわかりやすいです。ただ、これは BigQuery SQL として parse する前に置換する placeholder なのか、仮想 table name として analyzer catalog に登録するのかを明記するとよいです。

推奨は「parse 前 replacement」です。

```text
`__external__` is a config-level placeholder, not a real BigQuery identifier.
The generator replaces it with the generated `EXTERNAL_QUERY(...)` table expression before BigQuery SQL analysis. It must appear exactly once in `federated.outer_sql`.
```

もし仮想 table として analyzer catalog に入れるなら、CTE や実 table 名との衝突をどう扱うかを明記する必要があります。単純な置換の方が、責務が狭くて良いです。

### 12. BigQuery source DDL と external dataset projection の catalog collision を定義する

BigQuery schema DDL と external dataset projection を同じ BigQuery catalog に入れるなら、次の衝突を定義する必要があります。

```text
- BigQuery DDL already defines `example-project.analytics_spanner.Singers`
- external dataset projection also defines `example-project.analytics_spanner.Singers`
```

この場合は silent override せず、planning error にした方が安全です。

```text
bigquery_catalog_table_collision
  table: example-project.analytics_spanner.Singers
  sources:
    - bigquery_schema_ddl: bigquery_schema.sql
    - spanner_external_dataset: analytics_spanner from app_spanner
  remediation: Use a different external dataset name or remove the duplicate BigQuery DDL table.
```

同様に、unqualified dataset alias を登録するときも、default project 由来の alias が既存 dataset と衝突しないかを検証してください。

### 13. Spanner PostgreSQL dialect external dataset は明示的に unsupported と書く

公式には BigQuery Spanner external dataset は GoogleSQL と PostgreSQL dialect の Spanner database に link できます。一方、この tool は Spanner GoogleSQL / BigQuery GoogleSQL の静的解析を核にしています。

そのため、DESIGN の external dataset section に次を足すと誤解が減ります。

```text
Spanner PostgreSQL-dialect external datasets are not supported by this generator yet.
`spanner_external_datasets[].spanner_source` must refer to a Spanner GoogleSQL schema. PostgreSQL support requires a separate dialect analyzer and type mapping policy.
```

これを書いておかないと、「BigQuery external dataset 公式機能として PostgreSQL Spanner も対応しているなら、この generator も DDL projection できるはず」という期待が生まれます。

## 優先度

P0 として仕様固定してほしいもの:

1. `unsupported_columns: omit` での explicit column reference は planning error。
2. `named_schema_policy: warn_and_omit` での explicit table reference は planning error。
3. projected column name の case-folding collision 検証。
4. BigQuery DDL catalog と external dataset projection の table collision 検証。
5. `vet --output json|yaml` の stdout/stderr/exit code 契約。

P1 として入れるとかなり良いもの:

1. external dataset 用 type compatibility matrix。
2. `database_role` visibility not verified warning。
3. binding-level `vet` suppression。
4. optional `external_source` / `location` / `connection` metadata。
5. `__external__` placeholder replacement phase の明文化。

P2 でよいもの:

1. role-based allowlist。
2. optional live BigQuery / Spanner verification。
3. BigQuery execution methods で external dataset tables を source として使う DML の扱い。
4. Spanner PostgreSQL dialect external dataset support。

## そのまま送れる返答案

> 今回の対応は概ね良いです。external dataset を `EXTERNAL_QUERY` の別記法にせず、BigQuery catalog binding として扱う方針、`external_query_connections` と `spanner_external_datasets` の分離、`outer_sql` の配置固定、projection policy / access / Data Boost metadata を plan に載せる方針には同意します。
>
> 追加で仕様固定してほしいのは、まず `unsupported_columns: omit` の query behavior です。projection build では omit + warning でよいですが、`SELECT *` は BigQuery-visible columns のみへ展開し omitted column warning を出し、明示的に omitted column を参照した場合は planning error にしてください。`named_schema_policy: warn_and_omit` も同様に、projection から omit することと、明示参照を許すことは分けるべきです。
>
> 次に、external dataset table names だけでなく projected column names の case-folding collision も検証してください。BigQuery は case だけ違う duplicate column names を許さないため、Spanner 側の `Foo` / `foo` のような列は projected BigQuery catalog で error にする必要があります。
>
> また、BigQuery schema DDL と external dataset projection を同じ analyzer catalog に入れるなら、同じ canonical table path を二重に定義した場合は planning error にしてください。silent override は避けるべきです。
>
> `vet` については、query-level suppression だけでなく、external dataset binding-level warning を抑制できる場所も必要です。unsupported column omitted、database role visibility not verified、Data Boost permission not checked などは query ではなく schema binding の warning なので、`schemas[].spanner_external_datasets[].vet.disable` か structured `rules.suppressions` with selector を設計してください。
>
> 最後に、`access_verified: false` は「検証して false」なのか「未検証」なのか曖昧なので、plan では `verification.status: not_checked|verified|mismatch|failed` のような tri-state に寄せるのが安全です。generator が live infra を作らない方針なら default は `not_checked` が最も正直です。

## Closing

ここまで来ると、設計上の大きな懸念はかなり解消されています。今後のレビューでは、機能追加よりも、`explain-plan` / `vet` / regression tests で「実サービスと static projection のズレをどう見える化するか」を中心に見るのがよさそうです。

external dataset は `EXTERNAL_QUERY` より query text から実体が見えにくいので、catalog binding の plan 表現がこの機能の信頼性そのものになります。今回の方向性は正しいので、あとは omitted columns、role visibility、case-folding、catalog collision、read-only target detection を丁寧に固定してください。

## References

- Latest response: `/mnt/data/spanner-query-gen-external-dataset-response-round2-response-ja.md`
- Latest README: `/mnt/data/README(5).md`
- Latest DESIGN: `/mnt/data/DESIGN(5).md`
- Google Cloud: Create Spanner external datasets, last updated 2026-05-01 UTC: https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets
- Google Cloud: Federated query functions / `EXTERNAL_QUERY`, last updated 2026-05-01 UTC: https://docs.cloud.google.com/bigquery/docs/reference/standard-sql/federated_query_functions
- Google Cloud: BigQuery schema column naming rules: https://docs.cloud.google.com/bigquery/docs/schemas
