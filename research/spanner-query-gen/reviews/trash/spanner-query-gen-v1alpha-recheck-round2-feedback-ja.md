# `spanner-query-gen` v1alpha 再確認フィードバック

作成日: 2026-05-05

## 前提

このレビューは、最新の README / DESIGN / cleanup plan / `spanner-query-gen.v1alpha.schema.json` / 回答案を読んだうえでの仕様レビューです。実装コード、実行ログ、テスト出力は共有されていないため、ここでは「実装済みかどうか」は検証していません。対象は、外部レビュー可能な public contract、README/DESIGN の説明、JSON Schema の形、今後の計画の整合性です。

## 総評

かなり良くなっています。特に次の整理は、このまま進めてよいと思います。

- public YAML surface を `version / go / emit / catalogs / queries / writes / rules.suppressions` に絞ったこと。
- README の最初の例を Spanner-only にし、BigQuery、`EXTERNAL_QUERY`、external dataset、suppression を後段に分けたこと。
- `emit` の責務を「DTO/SQL constants を gate するもの」ではなく、「追加 runtime helper surface を gate するもの」と明記したこと。
- `kind: external_query` と BigQuery Spanner external dataset を分離したこと。
- `upsert` を Spanner `INSERT OR UPDATE` に忠実な set equality に寄せ、branch-different upsert を将来の `ON CONFLICT` に送ったこと。
- `config-schema` を public contract review / editor integration / CI 用に追加したこと。

大きな設計変更はもう不要です。ここからは、**「ドキュメントで約束している v1alpha grammar」と「JSON Schema / plan diagnostics が実際に拒否できる範囲」を同期する段階**だと思います。

## P0: JSON Schema を “canonical contract” と呼ぶなら、もう少し強くする

README では `config-schema` を public v1alpha YAML config の JSON Schema として位置付けています。これは良い方向ですが、現状の schema はかなり structural で、README / DESIGN が「v1alpha では拒否する」と説明している形の一部を schema 上は許します。

手元で schema ファイルだけを検査すると、少なくとも次のような config は JSON Schema validation 上は valid になります。これは実装の planner が拒否するかどうかとは別問題です。

```yaml
# kind: table なのに table がない
queries:
- name: Bad
  catalog: app
  kind: table
  result:
    struct: Row
```

```yaml
# kind: sql なのに table も同時に指定できる
queries:
- name: Bad
  catalog: app
  kind: sql
  sql: SELECT 1
  table: Singers
  result:
    struct: Row
```

```yaml
# external_query なのに inner_sql がない
queries:
- name: Bad
  catalog: analytics
  kind: external_query
  binding: app_conn
  result:
    struct: Row
```

```yaml
# update なのに update.columns も update.all_non_key_columns もない
writes:
- name: Bad
  catalog: app
  table: Singers
  operation: update
  input: Row
```

```yaml
# verification_evidence が空でも schema 上は通る
verification_evidence: {}
```

v1alpha の schema を「canonical public YAML schema」として出すなら、最低限以下は schema 側で表現した方がよいです。

### Query kind の discriminated union

```text
kind: sql
  require: sql
  forbid: table, index, binding, inner_sql, outer_sql, key_prefix

kind: table
  require: table
  forbid: sql, index, binding, inner_sql, outer_sql, key_prefix

kind: index
  require: index
  allow: key_prefix
  forbid: sql, table, binding, inner_sql, outer_sql

kind: external_query
  require: binding, inner_sql
  allow: outer_sql
  forbid: sql, table, index, key_prefix
```

`outer_sql` に exactly one `__external__` があるか、raw SQL に `EXTERNAL_QUERY(...)` が含まれるか、という lexical / SQL-level validation は planner diagnostics でよいです。ただし、kind-specific な required / forbidden fields は JSON Schema でかなり素直に書けます。

### Write operation の discriminated union

```text
operation: insert
  require: insert.columns
  forbid: update, conflict

operation: update
  require exactly one of:
    - update.columns
    - update.all_non_key_columns: true
  forbid: insert, conflict

operation: upsert
  require: insert.columns, update.columns, conflict.strategy
  forbid: update.all_non_key_columns unless intentionally supported
  planner validates: set(insert.columns) == primary_key_set + set(update.columns)

operation: replace
  require: insert.columns
  forbid: update, conflict
  planner rejects DML helper generation

operation: delete
  forbid: insert, update, conflict
```

`upsert` の set equality は DDL の primary key と照合するため planner validation でよいです。一方で、「update に update block がない」「insert に insert.columns がない」のような形は schema で落とした方が、`config-schema` を使う価値が上がります。

### Evidence / suppression metadata の required fields

README では verification evidence を「実 evidence がある時だけ書く」と説明しているので、空 object が valid になるのは避けたいです。

```text
verification_evidence:
  required: status, verifier, checked_at
  status: verified | mismatch | failed
  checked_at: format date-time
```

同様に、`rules.suppressions[].expires` は README で `YYYY-MM-DD` としているので、schema でも `format: date` または pattern を付けるのがよいです。`scope` も `^(query|write|catalog-binding)/` 程度は schema で縛れるはずです。

schema を強くしたくない場合は、README の `config-schema` 説明を次のように弱める必要があります。

```text
config-schema validates the structural shape of the public YAML config.
Kind-specific and DDL-dependent semantic invariants are enforced by check/vet/explain-plan.
```

ただ、今の方向性なら、v1alpha では schema を少し強くする方を推します。

## P0: alias value がまだ schema に残っている

field alias はかなり削れましたが、value alias が schema にまだ残っています。v1alpha を simple にするなら、ここも削った方が一貫します。

現在の schema では、たとえば以下が valid になります。

```yaml
dialect: google-sql
```

```yaml
projection:
  unsupported_columns: error
  named_schema_tables: omit
```

README/DESIGN は「unsafe aliases and compact fields can be removed」としているので、schema enum も canonical value だけにした方がよいです。

推奨形はどちらかです。

```text
dialect: googlesql | postgresql
```

外部 dataset projection は、個人的には `reject` を canonical にするのが読みやすいです。

```text
unsupported_columns: reject | omit
named_schema_tables: reject | warn_and_omit
```

`error` は diagnostic severity に見え、`reject` は projection policy に見えます。config に書く値としては `reject` の方が自然です。`error` を残すなら、README に「`error` は `reject` の alias」と明記する必要がありますが、v1alpha の目的が単純化なら alias は入れない方がよいです。

## P1: external dataset projection matrix の default severity が少し矛盾している

README は、external dataset の default projection policy を conservative とし、unsupported columns と named-schema tables は rejected と説明しています。DESIGN 本文も `unsupported_columns: error` / `named_schema_tables: error` が default と書いています。

一方で、DESIGN の projection matrix では named-schema table や unsupported column の default severity が「warning, or error with ...: error」のように読める箇所があります。ここは README と合わせて、次のようにした方がよいです。

```text
Default v1alpha:
  unsupported column -> planning error
  named-schema table -> planning error

With explicit lossy projection:
  unsupported_columns: omit -> omit + warning/provenance
  named_schema_tables: warn_and_omit -> omit + warning/provenance
```

BigQuery の Spanner external dataset では、default Spanner schema の table のみ accessible、unsupported type の column は BigQuery 側で accessible ではなく、DML や metadata mutation、`INFORMATION_SCHEMA`、Read API / Write API も unsupported です。したがって、generator の default を conservative にし、lossy behavior は明示 opt-in にする現在の方針は妥当です。

参考: Google Cloud documentation, “Create Spanner external datasets”, limitations and query behavior.

## P1: `status: not_checked` の `source: user_config` は少し紛らわしい

config から `verification_hint` と `not_checked` を外したのは良いです。一方、plan で DDL-only planning を次のように表す案は、少し誤解を生みます。

```yaml
verification:
  status: not_checked
  source: user_config
  independently_verified_by_generator: false
```

`source` は「検証の source」に見えるため、`not_checked` に `user_config` が付くと、何か user-provided evidence があるように読めます。より simple にするなら、どちらかに寄せたいです。

案 A: `not_checked` では `source` を省略する。

```yaml
verification:
  status: not_checked
  independently_verified_by_generator: false
```

案 B: 明示するなら `source: none` または `source: static_config_default` にする。

```yaml
verification:
  status: not_checked
  source: none
  independently_verified_by_generator: false
```

config-supplied evidence だけを `source: external_evidence` にし、future live probe だけを `source: live_probe` にする方が、レビュー時の意味が明確です。

## P1: `--stable` を出すなら plan の最小安定契約を先に書く

README は `explain-plan --stable` を snapshot-oriented output 用として紹介しています。これは便利ですが、`--stable` という名前は「machine-readable plan の安定性」をかなり強く連想させます。

v1alpha で出すなら、README か DESIGN に最小契約を書いた方がよいです。

```text
--stable keeps semantic fields and omits volatile audit fields only.
It does not guarantee that every plan field is stable across v1alpha releases.
The stable contract is plan_version + documented semantic fields.
```

また、`--stable --audit` の組み合わせも決めておくとよいです。

```text
--audit adds audit-only rows.
--stable removes volatile fields from both default and audit output.
```

これを曖昧にすると、ユーザーが v1alpha plan output 全体を golden file にして壊れやすくなります。

## P2: DESIGN / cleanup plan に小さな stale wording が残っている

最新 README / DESIGN / cleanup plan はだいぶ同期しましたが、まだいくつか小さな stale wording があります。

### `auto_all_non_key_columns`

v1alpha config では `update.all_non_key_columns: true` が canonical です。しかし DESIGN の説明や roadmap に `auto_all_non_key_columns` という旧 sentinel 名がまだ残っています。

変更案:

```text
before:
  auto_all_non_key_columns should appear in the plan...

after:
  update.all_non_key_columns should normalize to a plan field such as
  update.mode: all_non_key_columns or broad_update_opt_in: true.
```

旧名は migration table の「廃止対象」以外に出さない方がよいです。

### Roadmap の重複

Phase 1 に `Keep DML replace unsupported...` が重複しているように見えます。単純な重複なので削ってよいです。

### Phase 3 の `result` wording

DESIGN の Phase 3 に “Add `result: one|maybe_one|many|exec`” とありますが、v1alpha では `result.cardinality` はすでに plan に記録される説明になっています。

変更案:

```text
Activate cardinality semantics for generated query methods.
```

`result.cardinality` の config/plan presence と、generated method behavior は分けて書いた方が読みやすいです。

## P2: root-level schema の空 config 許容をどう扱うか決める

現在の schema は root required が `version` と `go` だけなので、`catalogs` / `queries` / `writes` がない config も schema 上は valid です。

これは意図的に「空 generator config も valid」にするなら問題ありません。ただし、普通の利用者には意味のない config なので、どちらかを明記した方がよいです。

```text
Option A:
  schema allows empty generation input, planner emits an empty output or warning.

Option B:
  schema requires at least one of queries or writes, and catalogs when either exists.
```

個人的には、v1alpha の CLI では Option B の方が親切です。ただし、schema を簡単に保ちたいなら Option A でもよく、その場合は `check` / `vet` の挙動だけ明記してください。

## P2: `params.type` の type syntax を少しだけ定義する

`queries[].params` を analyzer type input として残す判断は良いです。ただ、schema 上は `type: string` だけなので、README に一文あると使いやすくなります。

```text
params[].type is parsed as a GoogleSQL type expression such as INT64,
ARRAY<INT64>, STRUCT<...>, or the analyzer-supported subset documented in DESIGN.
Invalid type expressions are planning errors.
```

BigQuery outer scope と Spanner inner scope が分かれる `external_query` では、この field が重要になるので、plan には normalized type expression も残すとよいです。

## P2: diagnostic IDs を schema/README と結び付ける

`rules.suppressions` を central にしたので、rule IDs が public API になります。README は例として `external-dataset-access-unverified` を出していますが、最低限の built-in diagnostic ID 一覧を DESIGN に置くとよいです。

特に v1alpha では以下の ID は contract 化してよいと思います。

```text
query-kind-field-mismatch
external-query-raw-sql-unsupported
external-query-placeholder-invalid
write-operation-field-mismatch
write-update-columns-required
write-upsert-column-set-mismatch
write-replace-dml-unsupported
external-dataset-access-unverified
external-dataset-dml-target-unsupported
external-dataset-metadata-target-unsupported
external-dataset-lossy-projection
suppression-scope-invalid
suppression-expired   # future policy, if implemented later
```

`rules.suppressions` は ID が安定して初めて CI で使いやすくなります。

## これ以上 simple にしない方がよい点

以下は、さらに短く見せるために削らない方がよいです。

- `catalogs`: external query と external dataset を扱う以上、`schemas` より正しい。
- `kind: external_query`: raw SQL の `EXTERNAL_QUERY(...)` 抽出に戻すと、inner/outer scope、parameter scope、ORDER BY warning、type conversion matrix が見えにくくなる。
- `update.columns`: Spanner では touched columns が重要なので、ここを top-level `columns` に戻すべきではない。
- `rules.suppressions`: warning を抑制しても plan facts を消さない、という方針はかなり良い。
- external dataset を catalog binding として扱う方針: `EXTERNAL_QUERY` とは解析単位が違うので、今の分離が正しい。

## そのまま送れる返答案

> 最新版はかなり良くなっています。大きな設計変更はもう不要だと思います。`version / go / emit / catalogs / queries / writes / rules.suppressions` の public surface、Spanner-only first example、`emit` の責務整理、`kind: external_query` と external dataset の分離、`upsert` の set equality はこのまま進めてよいです。
>
> 追加で一番重要なのは、`config-schema` を public v1alpha schema として出すなら、README/DESIGN の grammar と同じ制約を JSON Schema でもある程度表現することです。現状の schema は structural なので、`kind: table` なのに `table` がない query、`kind: sql` なのに `table` も同時にある query、`external_query` なのに `inner_sql` がない query、`operation: update` なのに `update.columns` がない write、空の `verification_evidence` などが schema 上は通ります。planner が拒否するなら実行時には安全かもしれませんが、`config-schema` を CI / editor integration / public contract として出すなら、query kind と write operation の discriminated union、verification evidence の required fields、suppression scope/date format は schema 側にも入れた方がよいです。
>
> もう一点、value alias がまだ残っています。`dialect: google-sql` や projection policy の `error` / `reject` / `omit` の重複は、v1alpha の単純化方針と少しずれます。canonical value だけにして、たとえば `dialect: googlesql | postgresql`、`unsupported_columns: reject | omit`、`named_schema_tables: reject | warn_and_omit` に絞るのがよいです。
>
> external dataset projection の default severity も README と DESIGN matrix を同期してください。README は default conservative reject と説明しているので、matrix も “default: planning error; explicit lossy projection: omit + warning/provenance” と書いた方が明確です。
>
> 最後に、`status: not_checked` の plan source は `user_config` より `none` または source omitted の方が誤解が少ないです。config-supplied evidence だけを `external_evidence`、future live probe だけを `live_probe` とする方が review しやすいと思います。

## 参考

- Cloud Spanner GoogleSQL DML syntax: `INSERT OR UPDATE` updates specified columns and leaves unspecified columns unchanged; `ON UPDATE` columns are a special case.
  - https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax
- BigQuery Spanner external datasets: external dataset tables are queried like BigQuery tables, DML is unsupported, Data Boost is used by default, only default-schema Spanner tables are accessible, unsupported columns are not accessible on BigQuery side, and metadata mutation / `INFORMATION_SCHEMA` / Read API / Write API are unsupported.
  - https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets
