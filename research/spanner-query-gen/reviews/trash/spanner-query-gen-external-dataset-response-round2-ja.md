# `spanner-query-gen` external dataset 対応後レビューへの追加返答

対象ファイル:

- `spanner-query-gen-external-dataset-feedback-response-ja.md`
- `README(4).md`
- `DESIGN(4).md`

## 結論

今回の対応は概ね良いです。特に、BigQuery Spanner external dataset を `EXTERNAL_QUERY` の別記法にせず、BigQuery catalog binding として扱う判断は正しいです。`external_query_connections` と `spanner_external_datasets` を分けた点、current `vet` と future rule engine の境界を明確化した点、structured config で `update.mask.columns` と `update.mask.all_non_key_columns` を分ける方針も妥当です。

追加でフィードバックするなら、今回から論点は「方針の正しさ」よりも、**external dataset を catalog に入れた時の名前解決・可視性・projection policy・plan 表現をどこまで明文化するか**に移っています。

## 良くなった点

- `EXTERNAL_QUERY` と external dataset を別物として扱っている。
- external dataset を `queries[].federated` に入れず、BigQuery SQL から普通の table reference として解析する方針になっている。
- generator が BigQuery dataset / connection / IAM / Terraform を作らないことを明示している。
- DDL-first workflow を維持し、live BigQuery introspection を source of truth にしない方針を保っている。
- external dataset の制約、特に default Spanner schema、read-only、unsupported columns、Data Boost / access delegation / EUC への注意が DESIGN に入り始めている。

この方向でよいです。

## 追加で直すべき点

### 1. README の `external_schemas` 参照が古い

README の `federated` 説明に、`spanner_source`, if set, must match the `external_schemas` mapping for the connection、という説明が残っています。

ただし最新 config では、推奨名は `external_query_connections` で、`external_schemas` は compact backward-compatible alias です。ここは次のように直した方がよいです。

```text
spanner_source, if set, must match the external_query_connections mapping
for the connection. external_schemas is accepted only as a compact legacy alias.
```

この手の古い alias 表記は、外部利用者が新旧 config を混在させる原因になります。

### 2. `outer_sql` の配置を固定する

DESIGN の例では `outer_sql` が `federated` block の外側にあります。

```yaml
queries:
- name: FederatedSingerIDs
  source: analytics_bigquery
  federated:
    connection: example-project.us.example-connection
    spanner_source: app_spanner
    inner_sql: |
      SELECT SingerId
      FROM Singers
  outer_sql: |
    SELECT *
    FROM __external__
```

一方で、対応文では `federated.outer_sql` のように読める説明になっています。ここは早めに固定した方がよいです。

私なら `outer_sql` は `federated` の内側に置きます。

```yaml
queries:
- name: FederatedSingerIDs
  source: analytics_bigquery
  federated:
    connection: example-project.us.example-connection
    spanner_source: app_spanner
    inner_sql: |
      SELECT SingerId
      FROM Singers
    outer_sql: |
      SELECT *
      FROM __external__
```

理由は単純で、`outer_sql` は raw BigQuery SQL query には意味を持たず、`federated` query の構成要素だからです。`queries[]` 直下に置くと、将来 `sql` / `federated` / `external_dataset` などが増えた時に field の所有者が曖昧になります。

### 3. external dataset の `dataset` は文字列だけでなく canonical reference として扱う

README では次のように短い dataset 名を使っています。

```yaml
spanner_external_datasets:
- dataset: analytics_spanner
  spanner_source: app_spanner
```

DESIGN の proposed config では project-qualified な名前も出ています。

```yaml
spanner_external_datasets:
- dataset: example-project.analytics_spanner
  spanner_source: app_spanner
```

BigQuery SQL 側では、同一 project の `dataset.table`、project-qualified の `project.dataset.table`、backtick が必要な project ID などが混ざります。ここを単なる string のままにすると、名前解決が曖昧になります。

推奨は、loader では string を受け付けても、plan では必ず canonical form に正規化することです。

```yaml
spanner_external_datasets:
- dataset: analytics_spanner
  project: example-project   # optional; omitted means BigQuery source default project
  spanner_source: app_spanner
```

plan 側は次のような形にするとよいです。

```yaml
catalog_bindings:
- kind: spanner_external_dataset
  bigquery_dataset:
    project: example-project
    dataset: analytics_spanner
  source_schema: app_spanner
  projected_tables:
  - bigquery_table: example-project.analytics_spanner.Singers
    spanner_table: Singers
```

BigQuery 外部 dataset では table names が case-insensitive と説明されているので、resolver も少なくとも projected BigQuery table reference については case-insensitive に寄せるべきです。

### 4. `EXTERNAL_QUERY` と external dataset で type compatibility の severity を分ける

今の DESIGN は Spanner-to-BigQuery compatibility matrix を `EXTERNAL_QUERY` に寄せて定義しており、external dataset 側にも近い制約を流用しています。方向はよいですが、severity は同じではありません。

`EXTERNAL_QUERY` の場合、unsupported type が inner SQL の output に出ると query failure です。したがって `STRUCT` / `PROTO` / `ENUM` などは output error として扱うのが自然です。

一方、external dataset の場合、公式制約では unsupported Spanner columns は BigQuery 側で accessible になりません。つまり、catalog projection 時に「列を omit する」か「projected table 自体を reject する」かを generator policy として決める必要があります。

したがって、matrix は context を分けるとよいです。

```yaml
conversion_contexts:
- context: external_query_output
  unsupported_column_policy: error
- context: spanner_external_dataset_projection
  unsupported_column_policy: omit_or_error_by_config
```

そして config で初期値を明示してください。

```yaml
spanner_external_datasets:
- dataset: analytics_spanner
  spanner_source: app_spanner
  unsupported_columns: omit   # or error
```

実運用では `omit` が BigQuery の挙動に近いですが、codegen/CI では「列が消える」方が怖いケースもあります。初期 default は `error` にして、`omit` を明示 opt-in にするのも良いです。

### 5. external dataset の access / role / visibility を plan に残す

DESIGN の proposed config に `access: unknown` が入り始めているのは良いです。ただ、ここはもう少し意味を固定した方がよいです。

external dataset は BigQuery から見ると ordinary table reference に近いですが、可視性は Spanner 側の database role、EUC、access delegation、connection service account の権限に依存します。generator が infrastructure を作らないなら、少なくとも plan に「この catalog projection は DDL ベースの仮定であり、実際の IAM/role 可視性は未検証」と出すべきです。

例:

```yaml
spanner_external_datasets:
- dataset: analytics_spanner
  spanner_source: app_spanner
  access:
    mode: unknown          # unknown | end_user_credentials | delegated_connection
    database_role: null
    verified: false
```

plan warning:

```yaml
warnings:
- rule: external-dataset-access-unverified
  severity: warning
  message: "Projected catalog is based on Spanner DDL. Actual BigQuery visibility depends on external dataset access mode and Spanner permissions."
```

Data Boost についても同様です。generator が権限を管理しないのは正しいですが、external dataset query は Data Boost を使うため、plan には `requires_databoost_access: true` くらいの metadata を残すとレビューしやすくなります。

### 6. default Spanner schema 以外の扱いを policy 化する

DESIGN には「only default Spanner schema tables are visible」とあります。これは重要です。

ただし、Spanner DDL に named schema が含まれている場合、generator がどうするかを明確にする必要があります。

候補は次です。

```yaml
spanner_external_datasets:
- dataset: analytics_spanner
  spanner_source: app_spanner
  named_schema_policy: warn_and_omit   # error | warn_and_omit
```

初期は `warn_and_omit` でよいと思います。理由は、external dataset が default schema の table だけを投影する仕様に近いからです。ただし CI で厳密に管理したいプロジェクト向けに `error` も用意するとよいです。

### 7. BigQuery から見えない Spanner key metadata をどう使うかを明確にする

external dataset では、Spanner の primary key / foreign key metadata は BigQuery 側に見えません。ただし generator は Spanner DDL を持っているため、underlying Spanner primary key を知っています。

ここで注意すべきなのは、BigQuery SQL の analyzer と generated DTO の根拠を混同しないことです。

例えば、Spanner primary key column は実データ上 non-null なので DTO nullability の判断材料にはできます。しかし BigQuery catalog metadata として primary key が見えるわけではないため、plan の provenance は分けるべきです。

```yaml
fields:
- name: SingerId
  type: INT64
  nullable: false
  nullability_reason: non_null_by_underlying_spanner_pk
  visible_in_bigquery_metadata: true
  bigquery_key_metadata_visible: false
```

この区別がないと、将来 BigQuery optimizer / method generation / schema metadata emission に進んだ時に「BigQuery に存在しない metadata を BigQuery catalog にあるものとして扱う」バグにつながります。

### 8. `vet` は warning を見せる UX まで決める

README は、current `vet` が resolved plan を構築し planning errors で失敗し、structured warnings と suppressions は `explain-plan` に出る、と説明しています。この説明は正確かもしれませんが、CI 用 command としては少し不便です。

`vet` を使うユーザーは、warning を見るために別途 `explain-plan` を実行する必要があるのか、それとも `vet` でも warning summary が出るのかを知りたいです。

おすすめは次です。

```sh
spanner-query-gen vet --config querygen.yaml
# planning errors: non-zero
# warnings: printed to stderr, exit 0

spanner-query-gen vet --config querygen.yaml --output json
# machine-readable warnings + suppressions
```

将来の full rule engine で `--strict` や warnings-as-errors を入れるとしても、現時点の `vet` が warning をどう表示するかは先に固定してよいと思います。

### 9. README example の `SaveSinger` が validation と矛盾しないことを明示する

README の compact example では `SaveSinger` が `insert_or_update` で `update_mask: [FirstName]` だけを持っています。

```yaml
writes:
- name: SaveSinger
  source: app_spanner
  table: Singers
  operation: insert_or_update
  input_struct: SingerWrite
  update_mask:
  - FirstName
```

現在の compact config では、`insert_or_update` の insert branch は primary key + `update_mask` columns を insert value set として扱う、と説明されています。したがって sample DDL の `Singers` に `LastName STRING(MAX) NOT NULL` のような required column があると、この example は validation error になります。

対応文では sample validity を維持すると書かれているので、おそらく実際の sample DDL では問題ないのだと思います。ただ、README 単体では読者が確認できないため、次のどちらかを入れると親切です。

```yaml
# This example assumes all non-key columns other than FirstName are nullable
# or have server-provided defaults.
```

または、より安全に:

```yaml
update_mask:
- FirstName
- LastName
```

実装の validation が強くなったので、README example も「安全側の例」に寄せるのがよいです。

### 10. compact alias の削除時期を roadmap に書く

正式リリース前なので互換性維持より危険な default を潰す、という方針には賛成です。

ただし、`external_schemas`、`columns` alias、`auto_all_non_key_columns` sentinel などは、README と DESIGN に残っている限り利用者が依存します。pre-stable でも migration intent は書いておくとよいです。

```text
Before v1.0:
- remove external_schemas alias in favor of external_query_connections
- remove columns alias for update-style writes
- replace update_mask: [auto_all_non_key_columns] with structured update.mask.all_non_key_columns
```

特に `auto_all_non_key_columns` は column name と衝突し得る sentinel なので、compact 期間中だけの暫定 escape hatch だと明示した方がよいです。

### 11. external dataset の table path quoting を統一する

README は次のように unquoted path を使っています。

```sql
SELECT SingerId, FirstName
FROM analytics_spanner.Singers
```

DESIGN の proposed config では project-qualified path を backtick で囲んでいます。

```sql
SELECT SingerId
FROM `example-project.analytics_spanner.Singers`
```

どちらも文脈によっては妥当ですが、README の main example では BigQuery の table path / quoting rules を誤読しやすいです。

おすすめは、README では project-qualified で backtick 付きの例に統一することです。

```sql
SELECT SingerId, FirstName
FROM `example-project.analytics_spanner.Singers`
```

また、config の `dataset` が unqualified の場合は、source BigQuery schema の default project に解決される、と plan に明示してください。

### 12. external dataset projection の golden tests を追加する

external dataset support を Current Scope に入れるなら、少なくとも次の regression tests は必要です。

- `spanner_external_datasets` が BigQuery catalog に projected table を追加する。
- default Spanner schema の table のみ projected される。
- named Spanner schema の table は policy に従って omit/error になる。
- unsupported Spanner columns は policy に従って omit/error になる。
- projected external dataset table への DML は rejected / warning になる。
- `INFORMATION_SCHEMA` reference は rejected / warning になる。
- project-qualified / unqualified / backticked table references が同じ canonical table に解決される。
- external dataset projection が `explain-plan` に catalog binding として出る。

これらは生成コードのテストというより、plan analyzer の golden test として固定するのがよいです。

## そのまま送れる返答案

> 対応ありがとうございます。今回の方向性はかなり良いです。特に、BigQuery Spanner external dataset を `EXTERNAL_QUERY` の別記法にせず、BigQuery catalog binding として扱う判断、`external_query_connections` と `spanner_external_datasets` を分ける判断、generator が dataset / connection / IAM / Terraform を作らず static catalog shape だけを model する判断には賛成です。
>
> 追加で直すなら、まず README の `federated` 説明に残っている `external_schemas` 参照を `external_query_connections` に直してください。`external_schemas` は legacy alias としてだけ触れるのがよいです。
>
> 次に、`outer_sql` の配置を固定してください。対応文では `federated.outer_sql` と読めますが、DESIGN の例では `outer_sql` が `federated` block の外側にあります。私は `outer_sql` を `federated` の内側に置く方を推奨します。raw `sql` query には意味がない field なので、query-level field にすると所有者が曖昧になります。
>
> external dataset については、`dataset` を単なる string としてではなく、plan で canonical BigQuery dataset reference に正規化してください。project-qualified / unqualified / backticked table path、default project、case-insensitive external dataset table names を resolver で明確に扱う必要があります。
>
> また、`EXTERNAL_QUERY` と external dataset では unsupported type の扱いが違います。`EXTERNAL_QUERY` output では unsupported type は query failure なので error でよいですが、external dataset では unsupported Spanner column は BigQuery 側で accessible にならないため、projection policy として `unsupported_columns: error | omit` を config / plan に持たせるとよいです。
>
> external dataset は database role、EUC、access delegation、connection service account、Data Boost permission の影響を受けます。generator が infrastructure を作らない方針は正しいですが、plan には `access.mode`, `database_role`, `verified`, `requires_databoost_access` のような metadata と warning を残してください。
>
> 最後に、external dataset support を Current Scope に入れるなら、projection の golden tests を追加してください。default schema のみ projected、named schema omit/error、unsupported columns omit/error、read-only DML rejection、`INFORMATION_SCHEMA` rejection、project-qualified path resolution、`explain-plan` catalog binding 出力を固定すると、この機能はかなり安全になります。

## 優先度

P0:

- README の `external_schemas` 表記を `external_query_connections` に修正する。
- `outer_sql` の config path を固定する。
- external dataset の dataset/table reference を canonicalize する。
- `EXTERNAL_QUERY` と external dataset projection で unsupported type/column policy を分ける。

P1:

- access / role / Data Boost metadata を plan に入れる。
- named Spanner schema policy を明記する。
- README example の `SaveSinger` が validation と矛盾しないことをコメントまたは例で保証する。
- compact alias / sentinel の削除時期を roadmap に入れる。

P2:

- `vet --output json` または warning summary UX を決める。
- external dataset projection の golden tests を体系化する。

## 最終判断

external dataset を将来対象にする判断は問題ありません。むしろ、BigQuery catalog integration として入れるなら、このツールの「静的に宣言された SQL/schema を分析する」方向性と相性が良いです。

ただし、external dataset は `EXTERNAL_QUERY` よりも query-level の明示性が低いぶん、catalog binding の plan 表現が重要になります。`dataset` 名、project、table path、可視列、unsupported columns、access assumptions、read-only status を `explain-plan` に出せるようにしておけば、機能を入れてもプロダクト境界は崩れません。
