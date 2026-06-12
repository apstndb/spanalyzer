# `spanner-query-gen` 最新 README / DESIGN に対する追加レビュー

作成日: 2026-05-04
対象: 最新の `README.md` / `DESIGN.md` を前提にしたゼロベースの再レビュー

## 1. 結論

最新の `DESIGN.md` は、前回レビューで重要だった論点をかなり正しく取り込んでいます。特に、次の判断はよく整理されています。

- `spanner-query-gen` を ORM / query builder / broad CRUD generator にしないという境界を明文化している。
- `client: both` を暫定的な compact option と見なし、将来は `gen` / `runtime` / `dto` に分ける方向を示している。
- plan model と `explain-plan` を Phase 1 に置いている。
- nullability confidence と field provenance を内部 plan に持たせる方針になっている。
- validation / `vet` を Phase 2 に前倒ししている。
- BigQuery query methods を後回しにし、まず Spanner query methods に集中している。
- sqlc macro compatibility を約束せず、sqlc-like minimal annotations に留めている。
- yo-style model generation を opt-in の後段機能として扱っている。

したがって、もう大きな方向転換は不要です。追加でフィードバックするなら、主眼は「設計哲学」ではなく、採用時に事故りやすい細部の明文化です。

優先度が高い順に見ると、残る主な論点は次の 10 点です。

1. DML の `replace` / `INSERT OR REPLACE` を最新仕様で再確認する。
2. `update_mask` 省略時に全非キー列を更新する挙動を、将来は明示 opt-in にする。
3. shared DTO における read result と write input の requiredness を分離する。
4. BigQuery federation の型互換 matrix と制限事項を DESIGN に明文化する。
5. `EXTERNAL_QUERY` の inner SQL を、review しやすい形で扱える config option を検討する。
6. SQL parameter inference / parameter override の責務境界を決める。
7. `result: one` の cardinality semantics を明確にする。
8. `vet` rule の抑制機構と reason を最初から設計する。
9. type override の解決順序と nullable custom type 戦略を明文化する。
10. Spanner index shorthand が base table primary key columns をどう扱うかを再確認する。

以下、具体的に説明します。

---

## 2. まず評価すべき改善点

### 2.1 Product boundary はかなり良くなった

最新 DESIGN は、`spanner-query-gen` を「静的に宣言された GoogleSQL queries と Spanner writes を Go types/helpers に変換する generator」と定義し、ORM ではない、query builder ではない、broad CRUD generator ではない、と明記しています。

これは非常に重要です。sqlc 的な query-driven generator と、yo 的な Spanner model/mutation helper の中間に位置するツールは、簡単に「全部入り ORM」へ膨らみます。今回の DESIGN は、その誘惑をかなり抑えられています。

この境界は今後も守るべきです。特に、以下の機能は便利でも product identity をぼかしやすいため、今のように後段・opt-in に置くのがよいです。

- broad table model generation
- template customization
- plugin system
- BigQuery query execution methods
- sqlc macro compatibility
- runtime query builder 的なオプション展開

### 2.2 `explain-plan` を中核にしたのは正しい

最新 DESIGN は、plan を YAML/JSON として dump できるようにし、それを debugging aid ではなく review mechanism として位置づけています。これは正しいです。

このツールでは、ユーザーが最も不安になるのは「生成された struct がなぜこうなったのか」「table/index shorthand が実際には何を SELECT したのか」「shared DTO の nullable field がどこから来たのか」です。

そのため `explain-plan` は単なる補助コマンドではなく、設計の中核です。Phase 1 に置かれたのは妥当です。

### 2.3 validation を早めたのも正しい

`vet` を Phase 2 に前倒ししたのも妥当です。sqlc は query annotations、vet、verify といった workflow を持っており、query generator が CI に入るためには「生成できる」だけでなく「安全でない宣言を検出できる」ことが重要です。

`spanner-query-gen` の場合、特に以下の rule は sqlc より重要です。

- update mask なしの Spanner update
- cross-dialect DTO sharing
- BigQuery federation で返せない Spanner type
- unproven required field
- duplicate generated names
- unsafe write receiver sharing

---

## 3. 追加フィードバック: P0 / すぐ確認したいもの

## 3.1 DML の `replace` / `INSERT OR REPLACE` は最新仕様で再確認するべき

README は、DML operation として `replace` をサポートし、`replace` は `INSERT OR REPLACE` を emit すると説明しています。

ここは現時点で最も注意が必要です。

2026-05-04 時点で確認した Cloud Spanner GoogleSQL DML syntax では、top-level `INSERT` の grammar は概ね次の形です。

```sql
INSERT [[OR] IGNORE | UPDATE]
[INTO] table_name
  (column_name_1 [, ..., column_name_n])
  input [ASSERT_ROWS_MODIFIED number_rows] [return_clause]
```

同じ公式ページには `ON CONFLICT` syntax も示されていますが、top-level grammar には `OR REPLACE` が明示されていません。一方、Cloud Spanner Go client の Mutation API には `Replace` / `ReplaceMap` / `ReplaceStruct` があり、これは「既存行を削除してから insert し、明示されない値は NULL になる」動作として説明されています。

そのため、現状の README にある「DML replace emits `INSERT OR REPLACE`」は、少なくとも再確認が必要です。もし実際に生成して動作確認済みなら、その根拠となる Spanner version / syntax reference / integration test を README か DESIGN に明記した方がよいです。もし未確認なら、DML の `replace` は一旦 unsupported にするか、mutation-only operation として扱う方が安全です。

推奨する設計は次です。

```yaml
writes:
- name: ReplaceSinger
  operation: replace
  methods:
  - mutation
  dml:
    unsupported: true
    reason: "GoogleSQL DML replace syntax is not guaranteed"
```

または、operation を mutation semantics と DML semantics で明示的に分けます。

```yaml
writes:
- name: ReplaceSinger
  table: Singers
  mutation:
    operation: replace
  dml:
    operation: insert_or_update # or omitted / rejected
```

これは細かい話ではありません。`Replace` は `InsertOrUpdate` と異なり、指定しない列が NULL になるという強い semantics を持ちます。DML 側で同じ semantics を正しく表現できないなら、同名 operation で DML helper を出すべきではありません。

## 3.2 `update_mask` 省略時の default は将来危険

README は、`update` / `insert_or_update` で `update_mask` が省略された場合、non-hidden non-key columns を使うと説明しています。

これは現行実装の互換性としては理解できますが、DESIGN の「Spanner writes must be column-explicit」と緊張します。Spanner では更新対象列が lock scope や mutation count に影響するため、明示 column list はこのツールの中核価値です。

したがって、将来設計では次のようにした方がよいです。

```yaml
# 推奨: 明示 mask 必須
writes:
- name: UpdateSingerName
  operation: update
  update:
    mask:
    - FirstName
    - LastName
```

全非キー列更新をしたい場合も、暗黙 default ではなく明示 opt-in にします。

```yaml
writes:
- name: UpdateAllSingerColumns
  operation: update
  update:
    mask: auto_all_non_key_columns
```

また、Phase 2 の `vet` rule として以下を入れるとよいです。

```text
require-explicit-update-mask:
  default: warn in legacy config
  strict: error
```

`columns` を `update_mask` の alias として受け付ける現在の挙動も、README に「legacy compact syntax」と明記しておくと採用者が誤解しにくくなります。

## 3.3 shared DTO では read result と write input の安全基準を分ける

最新 DESIGN は、query results と write inputs が one Go struct を共有できると説明し、field provenance と nullability confidence を plan に持つ方針を示しています。これは良い方向です。

ただし、read result と write input は安全基準が違います。

query result の shared DTO では、ある query に field が欠けている場合に nullable に落とす、という判断は妥当です。一方、write input では key columns と touched columns が存在し、かつ write-safe な nullability を持つ必要があります。

たとえば、以下のようなケースは read DTO としては成立しても、write receiver としては危険です。

```yaml
queries:
- name: ListSingerNames
  table: Singers
  result_struct: Singer
  # SingerId, FirstName, LastName だけを持つ

writes:
- name: UpdateSingerStatus
  table: Singers
  input_struct: Singer
  update:
    mask:
    - Status
```

この場合、`Singer` が read DTO として生成済みだからといって、`Status` の write input として安全とは限りません。

plan model では、field ごとに単なる nullable/required だけでなく、役割を分けるとよいです。

```text
field SingerId:
  roles:
  - decode_result
  - encode_key
  required_for:
  - update_key

field FirstName:
  roles:
  - decode_result
  - encode_update_value
  null_semantics: write_null_allowed

field Status:
  roles:
  - encode_update_value
  required_for:
  - UpdateSingerStatus.update.mask
```

そして `vet` では以下を検出します。

```text
write-input-missing-key
write-input-missing-updated-column
write-input-nullability-unsafe
read-write-dto-merge-changes-write-contract
```

shared DTO は便利ですが、write helper の receiver に使う時だけは query result より強い validation をかけるべきです。

## 3.4 BigQuery federation compatibility matrix を DESIGN に入れる

最新 DESIGN は「Spanner-only values that BigQuery cannot return を reject する」としています。これは正しいですが、もう少し具体化した方がよいです。

BigQuery federated query では、Spanner の結果は BigQuery GoogleSQL types に変換されます。公式 mapping では、たとえば以下のような注意があります。

| Spanner GoogleSQL type | BigQuery federation output | 推奨扱い |
|---|---:|---|
| `INT64` | `INT64` | ok |
| `STRING` | `STRING` | ok |
| `BOOL` | `BOOL` | ok |
| `BYTES` | `BYTES` | ok |
| `DATE` | `DATE` | ok |
| `FLOAT64` | `FLOAT64` | ok |
| `NUMERIC` | `NUMERIC` | ok。ただし precision / range を文書化 |
| `JSON` | `JSON` | ok。ただし runtime loader の扱いを明確化 |
| `ARRAY` | `ARRAY` | element type compatibility を検証 |
| `STRUCT` | unsupported | federation output として reject |
| `TIMESTAMP` | `TIMESTAMP` | ok。ただし nanoseconds truncated を plan に出す |
| `PROTO` / `ENUM` | Spanner scope only | BigQuery output では reject または explicit cast 必須 |

これを DESIGN に入れると、`vet`、`explain-plan`、type model の判断基準が揃います。

また、BigQuery federated query には read-only、unsupported type failure、unique connection count、pushdown、performance などの制約があります。すべてを generator が保証する必要はありませんが、`explain-plan` には少なくとも次を出せるとよいです。

```text
federated_query:
  outer_source: analytics_bigquery
  connection: example-project.us.example-connection
  inner_source: app_spanner
  inner_sql_read_only: true
  output_conversion:
    SingerId: INT64 -> INT64 ok
    UpdatedAt: TIMESTAMP -> TIMESTAMP ok, precision_loss: nanoseconds_truncated
    SingerStruct: STRUCT -> reject
```

## 3.5 `EXTERNAL_QUERY` の inner SQL は別 config で書けるようにするとよい

README の例では、BigQuery SQL の中に Spanner SQL が string literal として埋め込まれています。

```sql
SELECT *
FROM EXTERNAL_QUERY(
  'example-project.us.example-connection',
  '''SELECT SingerId FROM Singers'''
)
```

これは BigQuery の現実に沿っていますが、review と escaping の観点ではつらいです。`SQL is the contract` という原則に対して、inner Spanner SQL が文字列の中に埋もれてしまいます。

将来的には、raw SQL をサポートしつつ、宣言的に inner SQL を分けて書ける option があるとよいです。

```yaml
queries:
- name: FederatedSingerIDs
  source: analytics_bigquery
  external_query:
    connection: example-project.us.example-connection
    spanner_source: app_spanner
    inner_sql: |
      SELECT SingerId
      FROM Singers
  outer_sql: |
    SELECT *
    FROM __external__
  result_struct: SingerRow
```

または、より単純に次でもよいです。

```yaml
queries:
- name: FederatedSingerIDs
  source: analytics_bigquery
  federated:
    connection: example-project.us.example-connection
    sql: |
      SELECT SingerId
      FROM Singers
  result_struct: SingerRow
```

この形式なら、inner SQL を Spanner analyzer で解析し、生成時に BigQuery `EXTERNAL_QUERY` SQL を組み立てられます。raw BigQuery SQL をそのまま解析する機能は残すべきですが、ユーザー体験としては separated form の方がレビューしやすいです。

## 3.6 parameter inference / parameter override の責務を明確にする

Proposed Config Shape では、query に `params` が明示されています。

```yaml
params:
- name: SingerId
  type: INT64
```

一方、GoogleSQL には `@SingerId` のような named parameter があるため、理想的には analyzer から parameters を推論できます。

ここは早めに設計を決めた方がよいです。おすすめは次です。

```text
1. SQL analyzer が parameter names と inferred types を抽出する。
2. config の params は source of truth ではなく override / disambiguation として扱う。
3. analyzer が型を証明できない parameter は explicit params を要求する。
4. ARRAY parameter、nullable parameter、STRUCT parameter は明示 config を許す。
5. DML params と query params は同じ plan model で扱う。
```

例:

```yaml
queries:
- name: FindSingers
  sql: |
    SELECT SingerId, FirstName
    FROM Singers
    WHERE LastName = @LastName
      AND SingerId IN UNNEST(@SingerIds)
  params:
  - name: SingerIds
    type: ARRAY<INT64>
```

`@LastName` は schema から推論できるかもしれませんが、`@SingerIds` は analyzer の能力によっては明示が必要です。

特に federation では、outer BigQuery parameters と inner Spanner parameters の境界が曖昧になりやすいです。`EXTERNAL_QUERY` の inner SQL が string literal なら、その中の `@param` は通常の outer query parameter と同じ扱いにできるとは限りません。ここは「サポートする」「reject する」「separated external_query config だけでサポートする」のどれかを明記すべきです。

---

## 4. 追加フィードバック: P1 / Phase 2〜3 までに決めたいもの

## 4.1 `result: one` の cardinality semantics を決める

最新 DESIGN は `result: one|many|exec` を Phase 3 に置いています。ここで重要なのは、pointer vs value だけでなく、cardinality semantics です。

`result: one` は、次のどれでしょうか。

1. 0 rows なら `nil, nil` / zero value。
2. 0 rows なら `ErrNotFound`。
3. 2 rows 以上なら最初の 1 row だけ返す。
4. 2 rows 以上なら error。
5. SQL 側に `LIMIT 1` がない場合は `vet` warning。

sqlc の `:one` は「1 row を返す想定」の annotation ですが、Spanner query methods として実装する場合、0 rows / multiple rows の UX を明確にしないと利用者ごとの期待が割れます。

おすすめは、`one` と `maybe_one` を分けることです。

```yaml
result: one        # exactly one; 0 rows and >1 rows are errors
result: maybe_one  # 0 rows allowed; >1 rows are errors
result: many
result: exec
```

ただし、初期実装でこれが重すぎるなら、最低限 README に次を明記するとよいです。

```text
result: one returns ErrNotFound on zero rows and returns an error if more than one row is observed.
```

また、Spanner DML には `THEN RETURN` があるため、`result: exec` が「row count だけ」なのか「DML returning result rows も扱う」のかも将来整理が必要です。今は `exec` を row-count only にし、`exec_returning: one|many` のような別概念を残すのが安全です。

## 4.2 `vet` rule suppression を最初から設計する

最新 DESIGN は `vet` を Phase 2 に置いています。これは良いですが、lint rule には必ず例外が必要です。

sqlc の `vet` には、query ごとに rule を無効化する annotation があります。`spanner-query-gen` でも同じ思想を取り込むべきです。ただし、単に disable するだけでなく reason を要求すると、CI review がしやすくなります。

SQL file annotation 例:

```sql
-- name: FederatedReviewedQuery :many
-- spanner-query-gen-vet-disable: cross-dialect-timestamp-truncation
-- spanner-query-gen-vet-reason: Reviewed; downstream uses microsecond precision only.
SELECT ...
```

YAML 例:

```yaml
queries:
- name: FederatedReviewedQuery
  vet:
    disable:
    - rule: cross-dialect-timestamp-truncation
      reason: "Reviewed; downstream uses microsecond precision only."
```

rule suppression がない `vet` は、BigQuery federation や shared DTO の正当な例外を扱えず、採用障壁になります。

## 4.3 type override の解決順序を明文化する

最新 DESIGN は type overrides を Phase 5 に置き、NUMERIC、JSON、BYTES、TIMESTAMP、proto、enum、BigQuery BIGNUMERIC を adoption blockers として扱っています。優先度として妥当です。

ただし、override は複雑化しやすいため、解決順序を DESIGN に入れるべきです。

推奨 order:

```text
1. query field override
2. write input field override
3. table.column override
4. named proto/enum override
5. dialect scalar type override
6. default scalar mapping
```

nullable custom type についても、以下を早めに決めるべきです。

```text
- non-null field uses CustomType
- nullable field defaults to NullValue[CustomType]
- user may specify CustomNullType
- CustomNullType must satisfy decoder/encoder contracts where runtime adapters require them
```

Go Spanner client には `spanner.NullString`、`spanner.NullTime`、`spanner.NullJSON`、`spanner.NullProtoMessage`、`spanner.NullProtoEnum` などがあります。独自 `NullValue[T]` を使う場合、これらの native null types とどう棲み分けるかも説明するとよいです。

## 4.4 index shorthand は base table primary key columns をどう扱うか明記する

README は、Spanner `index` query expansion について「index key columns followed by `STORING` columns」と説明しています。

ここは再確認した方がよいです。Cloud Spanner の secondary index は、index columns と STORING columns だけでなく、base table primary key columns も含みます。公式ドキュメントでも、secondary index に保存されるデータとして base table key columns が挙げられています。

したがって、`index` shorthand が「index-only read で安全に返せる列」を展開する意図なら、候補列は次の順序が自然です。

```text
index key columns
+ base table primary key columns not already included
+ STORING columns
```

ただし重複列は一度だけにします。

```text
AlbumTitle   # index key
SingerId     # base PK, if not already index key
AlbumId      # base PK
Marketing    # STORING
```

もし現在の実装が base PK を省いているなら、README に理由を書くべきです。たとえば「index shorthand は index definition に明示された columns だけを返す」という設計ならそれでもよいですが、Spanner の index read helper としては base PK を返せる方が使いやすいです。

また、null-filtered index / unique null-filtered index / interleaved index / descending key などは、`explain-plan` に最低限出るとよいです。

```text
index: AlbumsByTitle
base_table: Albums
selected_columns:
  - AlbumTitle: index_key
  - SingerId: base_primary_key
  - AlbumId: base_primary_key
  - MarketingBudget: storing
index_properties:
  null_filtered: false
  unique: false
  interleaved_in: null
```

## 4.5 `explain-plan` は human summary と machine JSON を分ける

`explain-plan` を早める判断は正しいですが、plan provenance を全部出すとノイズが大きくなります。

おすすめは 2 層に分けることです。

```sh
spanner-query-gen explain-plan --format summary
spanner-query-gen explain-plan --format json
spanner-query-gen explain-plan --format yaml
```

summary は review 用で、重要な差分だけを出します。

```text
Query: FindAlbumsByTitle
Expanded SQL:
  SELECT AlbumTitle, SingerId, AlbumId
  FROM Albums@{FORCE_INDEX=AlbumsByTitle}
  WHERE AlbumTitle = @AlbumTitle

DTO: AlbumIndexRow
  AlbumTitle: string, nullable_by_schema
  SingerId: int64, non_null_by_schema, base_primary_key
  AlbumId: int64, non_null_by_schema, base_primary_key

Warnings:
  none
```

JSON/YAML は CI や golden test 用に安定 schema として出します。人間向け output と機械向け output を混ぜない方が長期保守しやすいです。

---

## 5. 追加フィードバック: P2 / あると品質が上がるもの

## 5.1 README に「Current compact config」と「Future structured config」を分けて載せる

最新 DESIGN は future config をかなり整理しています。一方、README は現行実装の example として `client: both`、flat `operation: insert_or_update`、`columns` alias などを示しています。

これは current implementation の説明としては問題ありません。ただ、初見の利用者にはそれが「推奨される将来 API」に見える可能性があります。

README の例の直前に、以下のような短い注記を入れると誤解が減ります。

```text
The following example uses the current compact experimental config.
DESIGN.md describes the planned structured config with gen/runtime/dto sections.
Current compact keys such as client: both and columns-as-update-mask are expected
to remain as compatibility aliases while the tool is experimental.
```

さらに、migration table を DESIGN か README に置くとよいです。

| Current compact key | Future structured key | Migration policy |
|---|---|---|
| `client: both` | `runtime` + `dto.sharing` | accepted as alias in v1 |
| `columns` for update | `update.mask` | warn in strict mode |
| `operation: insert_or_update` | `operation: upsert` + `conflict.strategy` | accepted as legacy spelling |
| `schema` alias | `source` | accepted as compatibility alias |

## 5.2 generated constants は inner / outer SQL を分ける

Generated Go should expose SQL constants という方針は良いです。federated query では、BigQuery outer SQL と Spanner inner SQL を別 constant として出すと review しやすくなります。

```go
const FederatedSingerIDsSpannerSQL = `
SELECT SingerId
FROM Singers
`

const FederatedSingerIDsBigQuerySQL = `
SELECT *
FROM EXTERNAL_QUERY(
  'example-project.us.example-connection',
  FederatedSingerIDsSpannerSQL
)
`
```

実際の BigQuery SQL では string literal escaping が必要になるかもしれませんが、生成コードの中では「元の inner SQL」と「escape 済み outer SQL」を分けて可視化する価値があります。

## 5.3 `required_policy` の default と CI profile を分ける

README は `required_policy` の default を `override` と説明しています。これは現行 UX としては使いやすいですが、CI では `strict` を使いたいチームが多いはずです。

config に profile を入れると、段階的導入がしやすくなります。

```yaml
required_policy: override
ci:
  required_policy: strict
```

または rule として表現します。

```yaml
rules:
- name: require-strict-required-policy
  packages:
  - db/prod
  severity: error
```

## 5.4 query method receiver は Spanner と BigQuery を絶対に混ぜない

最新 DESIGN は、Spanner query methods を先に出し、BigQuery methods は後回しにしています。これは正しいです。

将来の generated client は、Spanner と BigQuery の abstraction を無理に統合しない方がよいです。

```go
type SpannerQueries struct { ... }
type BigQueryDTOs struct { ... } // methodsなし、metadataだけでもよい
```

また、Spanner でも read-only transaction / read-write transaction / client single-use read の違いがあります。初期 query methods は以下のように transaction-aware interface を受ける形が良さそうです。

```go
type SpannerQuerier interface {
    Query(context.Context, spanner.Statement) *spanner.RowIterator
}
```

ただし、最初から過剰に抽象化せず、Phase 3 で実装しながら決めるという現 DESIGN の態度でよいです。

## 5.5 generated docs に DML / Mutation 混在 warning を入れる

DESIGN は DML helpers と mutation helpers を別 API shape にし、混在を discouraged にするとしています。良い方針です。

追加で、生成関数の doc comment に明示 warning を入れるとよいです。

```go
// StatementUpdateSingerName returns a Spanner DML statement.
// Do not mix DML statements and mutation writes in the same transaction unless
// you intentionally account for Spanner's DML-before-mutation execution order.
func (v SingerNameUpdate) StatementUpdateSingerName() spanner.Statement
```

Mutation helper 側にも同様に入れます。

```go
// MutationUpdateSingerName returns a Spanner mutation.
// Prefer either DML or mutations within a single transaction, not both.
func (v SingerNameUpdate) MutationUpdateSingerName() *spanner.Mutation
```

## 5.6 schema source は DDL-first をさらに強調してよい

最新 DESIGN は live database に依存しない primary workflow を non-goal/goal で表現しています。これはよいです。

yo の v2 documentation でも、Information Schema 由来の generation は column ordering issue のため deprecated とされ、DDL file が recommended とされています。

そのため、`spanner-query-gen` も次の文を DESIGN に追加してよいと思います。

```text
The canonical workflow is DDL-first and deterministic.
Live introspection, if added later, is an import/debug convenience, not the source of truth.
```

これは CI / reproducible generation / code review の観点で強いメッセージになります。

---

## 6. エージェントに返すなら、この文面がよい

以下は、そのまま開発エージェントに渡せるフィードバック文です。

> 最新 DESIGN/README を前提に再確認しました。前回の大きな論点はかなり正しく反映されています。product boundary、`client: both` の将来分割、plan model / `explain-plan` の Phase 1 化、validation の前倒し、BigQuery methods の deferred、minimal sqlc-like annotations という方向は妥当です。大きな方向転換は不要です。
>
> 追加で優先して確認してほしいのは DML の `replace` です。README は DML `replace` が `INSERT OR REPLACE` を emit すると説明していますが、2026-05-04 時点の Cloud Spanner GoogleSQL DML syntax では top-level `INSERT` grammar に `OR REPLACE` が明示されておらず、`INSERT [[OR] IGNORE | UPDATE]` と `ON CONFLICT` が中心です。一方、Go client の Mutation API には `Replace` があり、これは「既存行を削除して insert し、指定しない列は NULL になる」という semantics です。Mutation replace と DML replace を同列に扱うのは危険なので、実際にサポートされる syntax を integration test で確認するか、DML replace は unsupported / mutation-only にしてください。
>
> 次に、`update_mask` 省略時に全 non-hidden non-key columns を更新する current behavior は、将来は明示 opt-in にした方がよいです。このツールの中核価値は Spanner write を column-explicit にすることなので、Phase 2 の vet では `require-explicit-update-mask` を入れ、legacy config では warning、strict mode では error にするのがよいと思います。
>
> shared DTO については、read result と write input の validation を分けてください。query result の merge では missing fields を nullable にしてもよいですが、write helper の receiver として使う場合は keys と touched columns の存在、nullability、write-safe semantics を別途検証すべきです。plan には field role として `decode_result` / `encode_key` / `encode_insert_value` / `encode_update_value` のような情報を持たせると安全です。
>
> BigQuery federation については、Spanner to BigQuery compatibility matrix を DESIGN に入れてください。特に `STRUCT` unsupported、`TIMESTAMP` nanoseconds truncated、`NUMERIC` precision/range、`PROTO`/`ENUM` の reject/cast 方針は、plan / vet / explain-plan の共通判断材料にすべきです。
>
> `EXTERNAL_QUERY` の inner SQL が BigQuery string literal に埋もれると review しにくいため、raw SQL support は残しつつ、将来は `external_query` / `federated` config で connection と inner Spanner SQL を別に書ける形を検討してください。これにより inner SQL を Spanner analyzer で解析しやすく、generated constants も outer SQL と inner SQL に分けられます。
>
> そのほか、Phase 3 までに `result: one` の 0 rows / multiple rows semantics、parameter inference と explicit param override の責務、`vet` rule suppression with reason、type override resolution order、index shorthand に base table primary key columns を含めるかどうか、を明文化すると採用品質が上がります。

---

## 7. 優先順位まとめ

| 優先度 | フィードバック | 理由 |
|---|---|---|
| P0 | DML `replace` / `INSERT OR REPLACE` を再確認 | unsupported syntax を生成すると即破綻する |
| P0 | `update_mask` 省略 default を legacy 扱いにする | Spanner-specific value proposition と衝突する |
| P0 | read DTO と write input の validation 分離 | shared DTO で write bug が生まれやすい |
| P0 | BigQuery federation type matrix | cross-dialect DTO sharing の安全基準になる |
| P0 | `EXTERNAL_QUERY` inner SQL separated config | reviewability と analyzer boundary が改善する |
| P1 | parameter inference / override policy | query method generation の前提になる |
| P1 | `result: one` semantics | generated method UX の根幹 |
| P1 | `vet` suppression with reason | 現実の例外を扱うために必要 |
| P1 | type override resolution order | custom type 対応で破綻しやすい |
| P1 | index shorthand の base PK handling | Spanner index semantics と UX に関わる |
| P2 | README の current/future config 分離 | 初見ユーザーの誤解を減らす |
| P2 | human/machine explain-plan 分離 | 長期的な CI/golden test に有効 |
| P2 | generated docs の DML/Mutation warning | transaction misuse を減らす |

---

## 8. 参考情報

- sqlc documentation: https://docs.sqlc.dev/
- sqlc query annotations: https://docs.sqlc.dev/en/latest/reference/query-annotations.html
- sqlc vet: https://docs.sqlc.dev/en/latest/howto/vet.html
- sqlc database/language support: https://docs.sqlc.dev/en/stable/reference/language-support.html
- yo v2 documentation: https://pkg.go.dev/go.mercari.io/yo/v2
- Cloud Spanner DML syntax: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax
- Cloud Spanner DML vs mutations: https://docs.cloud.google.com/spanner/docs/dml-versus-mutations
- Cloud Spanner Go client: https://pkg.go.dev/cloud.google.com/go/spanner
- Cloud Spanner secondary indexes: https://docs.cloud.google.com/spanner/docs/secondary-indexes
- BigQuery federated queries overview: https://docs.cloud.google.com/bigquery/docs/federated-queries-intro
- BigQuery federated query functions and Spanner type mapping: https://docs.cloud.google.com/bigquery/docs/reference/standard-sql/federated_query_functions
