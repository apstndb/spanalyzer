# `spanner-query-gen` 最新更新版への追加フィードバック

レビュー日: 2026-05-04
対象: `spanner-query-gen-latest-review-response.md`, `README(2).md`, `DESIGN(2).md`

## 結論

今回の更新で、前回レビューの大きな懸念はかなり解消されています。特に、DML `replace` を mutation-only にしたこと、`update_mask` 省略時の全列更新を長期 default にしないこと、read/write DTO role を plan に持つこと、BigQuery federation の型変換を matrix 化したこと、`explain-plan` を Phase 1 に置いたことは良い修正です。

追加でフィードバックするなら、設計思想の変更ではなく、**実装時に事故りやすい細部をさらに仕様化・テスト化すること**が中心です。優先度の高い順にまとめます。

---

## P0: 反映済み・実装済みと書いた内容は regression test で固定する

回答では、現在の compact 実装でも以下をすぐ直したと説明されています。

- `replace` の default は mutation helper のみ
- `methods: [dml]` for `replace` は error
- DML `INSERT OR REPLACE` の生成停止
- Spanner index shorthand が base table primary key columns を SELECT に含める
- README の説明更新

これは良いですが、ここはドキュメントだけでなく CI 上の regression test にしてください。特に DML `replace` は以前の仕様が危険だったため、将来の refactor で戻ると大きな事故になります。

推奨テストケース:

```text
writes.replace.default_methods
  input: operation: replace, methods omitted
  expect: mutation helper generated, no DML helper generated

writes.replace.explicit_dml_rejected
  input: operation: replace, methods: [dml]
  expect: config/planning error

writes.replace.no_insert_or_replace_sql
  input: operation: replace
  expect: generated output does not contain "INSERT OR REPLACE"

query.index.includes_base_pk
  input: secondary index on non-PK column
  expect: expanded SELECT includes index key columns, base table PK columns not already present, STORING columns

write.update_mask_omitted_is_marked_legacy
  input: update or insert_or_update without update_mask
  expect: plan records implicit/legacy auto-all behavior and emits warning, or test documents current behavior explicitly
```

理由: Cloud Spanner の Mutation API には `replace` が存在しますが、GoogleSQL DML の top-level `INSERT OR REPLACE` は公式 DML syntax 上では確認できません。一方で mutation `replace` は、既存行があれば削除して与えた列だけを insert し直し、明示されない値は `NULL` になるという強い意味を持ちます。また interleaved child table に `ON DELETE CASCADE` がある場合、parent row の replace は child rows も削除し得ます。

---

## P0: `insert_or_update` / upsert の insert branch validation を明文化する

現在の README の compact example では以下のように `insert_or_update` に `columns` だけが指定されています。

```yaml
writes:
- name: SaveSinger
  source: app_spanner
  table: Singers
  operation: insert_or_update
  input_struct: SingerWrite
  columns:
  - SingerId
  - FirstName
```

これは compact config としては理解できますが、ユーザーには「upsert では更新したい列だけを書けばよい」と読めます。実際には upsert は insert branch と update branch の両方を持つため、insert branch では `NOT NULL` / default / generated / commit timestamp / `ON UPDATE` などの列能力を見ないと安全に生成できません。

Cloud Spanner mutation `insertOrUpdate` では、行が既に存在して update される場合であっても、`insert` と同様に table のすべての `NOT NULL` columns に値が必要です。DML `INSERT OR UPDATE` も、primary key が見つからない場合には insert になるため、insert branch の required column coverage を plan/vet で検証すべきです。

提案:

```text
Plan validation for insert_or_update/upsert:
  insert_required_columns:
    - primary key columns
    - NOT NULL columns without default/generated/server value
  update_columns:
    - explicit update.mask columns only
  diagnostics:
    - missing required insert column => error
    - insert column is generated/non-writable => error
    - update mask includes key/generated/non-updatable column => error
```

structured config では、今の `operation: upsert` + `insert.columns` + `update.mask` + `conflict.strategy` の方向でよいです。ただし compact config でも、`columns` alias を update mask として扱うだけでなく、「insert branch に必要な列を満たしているか」を早めに検証した方が安全です。

---

## P0: Spanner column capability model を plan に入れる

`update_mask` と `columns` だけでは、Spanner の write semantics を十分に表現できません。DDL から各 column の capability を plan に正規化しておくべきです。

推奨する column capability:

```yaml
columns:
- name: LastUpdatedTime
  type: TIMESTAMP
  nullable: true
  roles:
    read: true
    insert_value: false        # or true if explicitly writable
    update_value: false        # depends on generated/on-update semantics
    key: false
  server_value:
    default: null
    on_update_expression: PENDING_COMMIT_TIMESTAMP()
    allow_commit_timestamp: true
  generated: false
  hidden: false
```

最低限、以下を区別してください。

- primary key column
- non-key writable column
- generated column
- hidden column
- default expression を持つ column
- commit timestamp / `PENDING_COMMIT_TIMESTAMP()` に関わる column
- `ON UPDATE` expression を持つ column
- insert branch では必要だが update branch では不要な column
- update mask に入れてはいけない column

特に DML `INSERT OR UPDATE` には、指定していない column は unchanged になる一方で、`ON UPDATE` expression を持つ column は、statement の column list に non-key column が含まれると自動的に更新されるという例外があります。これは「update_mask に列が入っていないから触らない」と説明すると不正確になります。plan には「明示的に送る列」と「副作用で変わり得る列」を分けて持たせるべきです。

```yaml
write_plan:
  explicit_update_columns:
  - Status
  server_side_update_effects:
  - column: LastUpdatedTime
    reason: on_update_expression_triggered_by_non_key_update
```

---

## P1: compact config から structured config への migration map を DESIGN に追加する

README は「current compact experimental config」であり、first stable release 前は互換性を約束しないと明記しています。これは良いです。ただし、README の例をコピーして使う人は多いため、compact と structured の対応表があると移行時の混乱を避けられます。

提案する migration map:

```text
Current compact config                 Structured config
--------------------------------------------------------------------------
client: both                            gen/runtime/dto sections
operation: insert_or_update             operation: upsert + conflict.strategy: insert_or_update
columns on insert_or_update             insert.columns and/or update.mask; do not overload
columns as update_mask alias             deprecated; use update.mask
omitted update_mask                      update.mask: auto_all_non_key_columns, explicit opt-in
replace + methods omitted                operation: replace + methods: [mutation]
source / schema alias                    source only; schema alias removed before stable
raw EXTERNAL_QUERY SQL                   federated.inner_sql + outer_sql where possible
required + required_policy               nullability confidence + required override rules
```

README の `SaveSinger` sample は、今のままだと非推奨予定の `columns` overload を推奨しているように見えます。次のどちらかにした方がよいです。

1. sample を `update_mask` に変える。
2. sample に `# compact legacy alias; structured config will split insert.columns/update.mask` という注釈を入れる。

---

## P1: mutation `replace` の危険な semantics を README/DESIGN にもう一段強く書く

DML `replace` を unsupported にしたのは正しいです。ただし mutation `replace` も安全な upsert ではありません。Cloud Spanner Mutation API の `replace` は、既存行があれば delete してから指定値を insert する semantics であり、明示されない values は `NULL` になります。interleaved child table に `ON DELETE CASCADE` がある場合には child rows も削除され得ます。

README の `replace` 説明に以下を足すことを推奨します。

```text
Mutation replace is not an update mask operation. If the row already exists,
Spanner deletes it and inserts the provided columns. Columns not explicitly
written become NULL, and replacing a parent row can delete interleaved child
rows when ON DELETE CASCADE is configured. Prefer insert_or_update/upsert when
preserving unspecified columns is desired.
```

この説明がないと、`replace` を「全列 upsert の強め版」程度に捉えられる危険があります。

---

## P1: DML helper / execution helper は Standard DML と Partitioned DML を分ける

現在の方針では DML helper は `spanner.Statement` を返し、Phase 3 で DML execution helpers を足す予定になっています。ここで Standard DML と Partitioned DML を同じ `result: exec` に混ぜない方がよいです。

提案:

```yaml
runtime:
  spanner:
    dml: true
    partitioned_dml: false
```

または query/write ごとに、

```yaml
writes:
- name: BulkExpireSessions
  operation: update
  dml:
    mode: partitioned
```

ただし最初は `standard` のみでよいです。Partitioned DML は主に bulk update/delete 用で、OLTP transaction helper と同じ UX にすると誤用されます。

DML と mutations の混在を避ける設計も、docs/lint だけでなく generated API の名前で分けるとよいです。

```go
func (v SingerUpdate) MutationUpdateSingerName() *spanner.Mutation
func (v SingerUpdate) StatementUpdateSingerName() spanner.Statement
```

Phase 3 で execution helper を追加するなら、DML execution path と mutation buffer path を同じ fluent API に載せない方が安全です。Spanner 公式ドキュメントでも、DML は read-your-writes を持つ一方、mutations は commit まで同一 transaction 内の SQL/DML から見えず、同一 transaction で混在させないことが best practice とされています。

---

## P1: federated query の `outer_sql` placeholder semantics を決める

DESIGN の structured federation config は良い方向です。

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

ただし `__external__` の意味を先に固定した方がよいです。

決めるべきこと:

- `__external__` は table expression placeholder なのか、CTE 名なのか。
- `outer_sql` で placeholder は exactly once なのか、複数回許すのか。
- alias は generator が付けるのか、ユーザーが `FROM __external__ AS e` と書くのか。
- `outer_sql` 省略時の default は `SELECT * FROM __external__` なのか。
- raw `EXTERNAL_QUERY` SQL は static string literal のみ解析対象にするのか。
- connection 引数が expression / parameter の場合は拒否するのか。
- inner Spanner parameters と outer BigQuery parameters の名前衝突をどう扱うか。

推奨は、最初はかなり狭くすることです。

```text
- outer_sql omitted => SELECT * FROM __external__
- __external__ must appear exactly once in FROM position
- generator renders WITH __external__ AS (SELECT * FROM EXTERNAL_QUERY(...))
- raw EXTERNAL_QUERY analysis supports only literal connection and literal SQL
- dynamic connection or dynamic SQL is unsupported
```

これにより、「SQL is the contract」を維持しつつ、複雑な BigQuery SQL rewriting を避けられます。

---

## P1: federated query では inner `ORDER BY` が BigQuery output order を保証しないことを vet する

BigQuery の `EXTERNAL_QUERY()` は、external query 内に `ORDER BY` があっても BigQuery 側の出力順を保証しません。したがって、`inner_sql` に `ORDER BY` があり、`outer_sql` に order がない場合は `vet` warning にした方がよいです。

例:

```yaml
rule: federated-inner-order-by-ignored
severity: warning
message: |
  EXTERNAL_QUERY does not preserve the inner query result ordering.
  Move ORDER BY to outer_sql if BigQuery result order matters.
```

`LIMIT 1` と `ORDER BY` の組み合わせは inner Spanner 側で「どの1行を返すか」を決めるため意味がありますが、複数行の順序を BigQuery result として期待するのは危険です。plan ではこの違いを説明できるとよいです。

---

## P1: BigQuery federation compatibility matrix に `handling severity` を足す

現在の matrix は良いですが、実装や vet に落とすなら、単なる `ok/warn/reject` だけでなく severity と remediation を持たせると使いやすいです。

```yaml
federation_type_decision:
  spanner_type: TIMESTAMP
  bigquery_type: TIMESTAMP
  status: warn
  rule: cross-dialect-timestamp-truncation
  remediation: "Cast or document that nanosecond precision is not required."

federation_type_decision:
  spanner_type: STRUCT
  bigquery_type: null
  status: reject
  rule: unsupported-federated-output-type
  remediation: "Project scalar fields or cast to JSON/STRING in inner_sql."
```

`ARRAY<T>` は recursive に判定してください。`ARRAY<INT64>` は ok ですが、`ARRAY<STRUCT<...>>` は element type が incompatible なので reject です。

また `PROTO` / `ENUM` は BigQuery の Spanner federation mapping table に出てこないため、現状の `unsupported unless explicitly cast` 方針でよいです。ただし plan では「明示 cast によって BigQuery-compatible scalar になった」ことを field provenance に残すべきです。

---

## P1: `result: one` / `maybe_one` の実装仕様を具体化する

DESIGN では `result: one` は exactly one row、`maybe_one` は zero allowed but more than one error と定義されました。これは良いです。次は generated method の実装方式と error taxonomy を決めてください。

推奨仕様:

```text
result: one
  zero rows       => ErrNoRows
  one row         => return row
  two or more     => ErrTooManyRows

result: maybe_one
  zero rows       => return nil, nil  または  (zero value, false, nil)
  one row         => return row
  two or more     => ErrTooManyRows
```

multiple rows を検出するには、iterator から最大2行を読む必要があります。`LIMIT 1` を勝手に付けると SQL contract を変えるため避けるべきです。

pointer/value の open question と関連しますが、`maybe_one` は pointer return の方が自然です。一方 `one` は value/pointer を config で選べます。

---

## P1: `explain-plan --format yaml/json` には schema version と source digest を入れる

`explain-plan` を stable YAML/JSON にする方針は良いです。CI/golden test に使うなら、machine-readable plan の schema を明確に versioning してください。

推奨 fields:

```yaml
plan_version: 1
schema_digests:
- source: app_spanner
  ddl_path: schema/spanner.sql
  sha256: ...
query_digests:
- name: GetSinger
  normalized_sql_sha256: ...
generator:
  name: spanner-query-gen
  version: 0.0.0-dev
  analyzer_version: ...
config:
  path: querygen.yaml
  sha256: ...
```

また、plan の安定性を保つため、以下を分けてください。

- human summary: review 用。多少変わってよい。
- stable plan JSON/YAML: CI/golden 用。field order と enum 値を固定。
- debug dump: internal troubleshooting 用。安定性を約束しない。

この分離は、Phase 1 の完了条件として非常に重要です。

---

## P2: type override は Phase 5 だとまだ遅い可能性がある

Type override を broad model generation より前に置いたのは良いです。ただし、Phase 5 は Spanner query methods と federated SQL UX の後になっています。実運用採用を考えると、最低限の scalar override は Phase 3 の前提または Phase 2.5 に前倒ししてもよいです。

理由:

- `NUMERIC` を `big.Rat` / decimal library / custom money type にしたいケースは多い。
- `JSON` を raw string / `[]byte` / custom struct にしたいケースが多い。
- `TIMESTAMP` precision policy は BigQuery federation と絡む。
- `BYTES` や proto/enum は application type に寄せたいケースがある。

提案:

```text
Phase 2.5: Minimal Overrides
  - scalar type override
  - table.column override
  - nullable custom type strategy
  - import path handling

Phase 5: Full Overrides
  - field rename
  - proto/enum identity override
  - advanced tags
  - broader custom contracts
```

Phase 3 の query methods までに minimal override があると、生成 API の採用障壁がかなり下がります。

---

## P2: generated name collision の rule をより具体化する

Phase 2 の `duplicate generated names` rule は良いです。さらに、どの collision を検出するかを具体化するとよいです。

検出すべき例:

```text
Column names:
  singer_id, SingerId, singerID => same Go field candidate
  type, package, var            => Go reserved/awkward names
  URL, Url, url                 => initialism policy conflict

Query names:
  GetSinger, get_singer, get-singer => same Go method candidate

Parameter names:
  @SingerId and @singer_id => same Go field candidate

Struct merge:
  same Go field name from different source column names
```

`explain-plan` には、最終 Go 名だけでなく original SQL name / source column name / query name を出してください。エラー時にユーザーが YAML 側で rename すべき場所を見つけやすくなります。

---

## P2: read/write shared DTO の validation は runtime surface ごとに contract matrix にする

DESIGN では read/write role を plan に持つようになっており、これはとても良いです。次は runtime surface ごとの contract を matrix にすると実装しやすくなります。

```yaml
field_runtime_contracts:
- field: FirstName
  roles:
  - decode_spanner_row
  - encode_spanner_dml_param
  - encode_spanner_mutation_value
  - load_bigquery_value
  type: NullValue[string]
  requirements:
    spanner_decoder: required
    spanner_encoder: required
    bigquery_loader: required_if_runtime_bigquery_dto
```

特に custom nullable type を許す場合、`runtime.bigquery.dto: true` と `runtime.spanner.mutations: true` では必要な interface が違います。`client: both` の将来分割は、この matrix に基づいて validation すると自然です。

---

## P2: Spanner external dataset は Non-Goal または Later Work に明記する

BigQuery から Spanner を query する方法は、`EXTERNAL_QUERY` だけでなく Spanner external dataset もあります。現在の tool は `EXTERNAL_QUERY` を first-class に扱う設計で、それ自体は良いです。ただし、ユーザーが「BigQuery Spanner federation 全体をサポートしている」と期待する可能性があります。

DESIGN の Non-Goals または Later Work に以下を追加すると境界が明確になります。

```text
Spanner external datasets are not supported initially. The first federation UX
focuses on EXTERNAL_QUERY because it exposes an explicit connection and inner SQL
that the generator can analyze. External dataset support may be considered later
as a separate BigQuery catalog integration feature.
```

---

## P2: `vet` suppression は reason に加えて scope と auditability を持たせる

`vet` suppression に reason 必須は良いです。さらに、CI 運用を想定すると以下もあると便利です。

```yaml
vet:
  disable:
  - rule: cross-dialect-timestamp-truncation
    reason: "Reviewed; downstream stores microseconds only."
    owner: "analytics-platform"
    expires: "2026-12-31"
```

`owner` / `expires` は必須でなくてもよいですが、plan には suppression を含めてください。`explain-plan --format json` に suppression が出れば、CI で「期限切れ suppression」を別途検出できます。

---

## 追加でそのまま返せるフィードバック文

> 反映内容はかなり良く、前回の P0 級の懸念はほぼ解消されています。追加で気になるのは、設計思想よりも実装時に事故りやすい細部です。
>
> まず、今回「すぐ直した」と書かれている DML `replace` 停止、`methods: [dml]` reject、index shorthand の base PK 追加は regression test で固定してください。特に `INSERT OR REPLACE` が将来の refactor で戻らないよう、生成出力に含まれないことをテストした方がよいです。
>
> 次に、`insert_or_update` / upsert の insert branch validation を明文化してください。Spanner mutation `insertOrUpdate` は、既存行の更新になる場合でも `insert` と同様にすべての `NOT NULL` columns が必要です。DML `INSERT OR UPDATE` も、行が存在しなければ insert になるため、update mask だけでは不十分です。plan で insert-required columns と update mask を分けて検証してください。
>
> また、Spanner column capability model を plan に入れるべきです。primary key、hidden、generated、default、commit timestamp、`ON UPDATE` expression、insertable/updatable を区別しないと、`auto_all_non_key_columns` や upsert helper が危険になります。特に DML `INSERT OR UPDATE` は未指定列を unchanged にしますが、`ON UPDATE` expression を持つ column は non-key column 更新時に server-side で変わり得るので、explicit update columns と server-side update effects を分けて plan に出すとよいです。
>
> README の compact example はまだ `insert_or_update` に `columns` を使っており、将来廃止予定の alias を推奨しているように見えます。compact→structured の migration map を DESIGN に追加し、README 例には legacy compact alias である注釈を入れるか、`update_mask` を使う例に変えた方がよいです。
>
> BigQuery federation では、`EXTERNAL_QUERY` の inner query に `ORDER BY` があっても BigQuery output order は保証されません。`inner_sql` に `ORDER BY` があり `outer_sql` に order がない場合は vet warning にしてください。また `__external__` placeholder の exactly-once / FROM-position / alias / default outer_sql semantics を先に固定した方がよいです。
>
> 最後に、`explain-plan --format yaml/json` は `plan_version`、DDL/config/query digest、generator/analyzer version を含めてください。human summary と stable machine plan を分ける方針は良いですが、CI に載せるなら plan schema versioning が必須です。

---

## 参考にした公式情報

- Cloud Spanner GoogleSQL DML syntax: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax
- Cloud Spanner Mutation REST API: https://docs.cloud.google.com/spanner/docs/reference/rest/v1/Mutation
- Cloud Spanner DML versus mutations: https://docs.cloud.google.com/spanner/docs/dml-versus-mutations
- Cloud Spanner secondary indexes: https://docs.cloud.google.com/spanner/docs/secondary-indexes
- BigQuery federated query functions: https://docs.cloud.google.com/bigquery/docs/reference/standard-sql/federated_query_functions
- BigQuery Spanner federated queries: https://docs.cloud.google.com/bigquery/docs/spanner-federated-queries
