# `spanner-query-gen` v1alpha 最新版への再確認フィードバック

作成日: 2026-05-05

## 前提

このレビューは、共有された Markdown / README / DESIGN / JSON Schema を対象にした仕様レビューです。実装コード、実行ログ、テスト出力は共有されていないため、「実装済みかどうか」は検証していません。ここでは、public config contract、README/DESIGN の整合性、JSON Schema の妥当性、将来の単純化余地に絞ります。

## 総評

かなり良くなっています。

特に良い点は次です。

- `version: v1alpha` を mutable preview として明確化した。
- public YAML surface を `go`, `emit`, `catalogs`, `queries`, `writes`, `rules.suppressions` にかなり絞った。
- README の最初の例が Spanner-only になり、BigQuery / `EXTERNAL_QUERY` / external dataset / suppressions が後段に分離された。
- `config-schema` を public contract として導入した。
- query DTO / SQL constants を baseline output とし、`emit` は runtime helper surface のみを制御する、と整理した。
- write-level `emit`、`rules.severity`、raw SQL `EXTERNAL_QUERY(...)`、external dataset `access.database_role` を v1alpha YAML から外した。
- `upsert` を Spanner `INSERT OR UPDATE` の実 semantics に合わせ、`insert.columns == key + update.columns` の invariant に寄せた。
- external dataset を `EXTERNAL_QUERY` shorthand ではなく BigQuery catalog binding として扱う方針が維持された。

大きな方向転換は不要です。今やるべきことは、**JSON Schema を public contract としてどこまで厳密にするか**、および **まだ残っている小さな重複や将来用フィールドを削れるか** の確認だと思います。

## P0: JSON Schema の役割を明確にする

README では `config-schema` が public v1alpha YAML config の JSON Schema を出すと説明されています。これは良いです。ただし、現状の schema は「public field set」はよく表していますが、「kind ごとの required/forbidden field」や「operation ごとの required/forbidden field」はあまり表していないように見えます。

たとえば、schema 上は次のような config も構文的には通り得ます。

```yaml
queries:
- name: Weird
  catalog: app
  kind: sql
  table: Singers
  outer_sql: SELECT * FROM __external__
  result:
    struct: WeirdRow
```

または、次のような write も schema だけでは弾きにくいです。

```yaml
writes:
- name: WeirdDelete
  catalog: app
  table: Singers
  operation: delete
  input: SingerWrite
  insert:
    columns: [SingerId, FirstName]
  update:
    columns: [LastName]
  conflict:
    strategy: insert_or_update
```

もちろん planner 側で semantic validation するのは正しいです。ただ、README が schema を review / editor integration / CI 用 contract として押し出すなら、最低限の discriminated-union 制約は schema に入れた方が、利用者の混乱が減ります。

おすすめは、すべての意味論を JSON Schema に詰め込むことではなく、次の「構文レベルの明らかな誤り」だけを schema で弾くことです。

### Query kind の最小 schema constraint

```text
kind: sql
  require: sql
  forbid: table, index, key_prefix, binding, inner_sql, outer_sql

kind: table
  require: table
  forbid: sql, index, binding, inner_sql, outer_sql

kind: index
  require: index
  allow: key_prefix
  forbid: sql, table, binding, inner_sql, outer_sql

kind: external_query
  require: binding, inner_sql
  allow: outer_sql
  forbid: sql, table, index, key_prefix
```

`outer_sql` 内の `__external__` が exactly once かどうか、placeholder が SQL identifier token かどうか、raw SQL `EXTERNAL_QUERY(...)` を検出するかどうかは planner validation のままでよいです。

### Write operation の最小 schema constraint

```text
operation: insert
  require: insert.columns
  forbid: update, conflict

operation: update
  require: update.columns or update.all_non_key_columns
  forbid: insert, conflict

operation: upsert
  require: insert.columns, update.columns
  allow: conflict only if kept in v1alpha

operation: replace
  require: insert.columns
  forbid: update, conflict

operation: delete
  forbid: insert, update, conflict
```

DDL から primary key を推論する、column が実在する、`insert.columns == key + update.columns` が成立する、NOT NULL / generated / hidden / ON UPDATE column をどう扱う、などは planner validation の責務でよいです。

schema を軽く保ちたいなら、README に次の一文を足すだけでもよいです。

> `config-schema` validates the public YAML shape and kind/operation syntax. Catalog-dependent semantics such as column existence, key inference, unsupported type projection, and upsert column-set invariants are validated by `check`, `vet`, and `explain-plan` planning.

この線引きを明記すると、schema が「完全な semantic validator」だと誤解されにくくなります。

## P0: `verification_evidence` は schema でも “real evidence” にする

README は「real evidence がある時だけ `access.verification_evidence` を書く」と説明しており、これは良いです。一方で schema 上は `verification_evidence` の中身が required になっていないように見えます。

v1alpha では次の形に固定した方がよいです。

```json
"verification_evidence": {
  "type": "object",
  "additionalProperties": false,
  "required": ["status", "verifier", "checked_at"],
  "properties": {
    "status": { "enum": ["verified", "mismatch", "failed"] },
    "verifier": { "type": "string" },
    "checked_at": { "type": "string", "format": "date-time" }
  }
}
```

空の `verification_evidence: {}` を許すと、「未検証だが evidence block はある」という曖昧な状態が生まれます。これは、これまで整理してきた `not_checked` を config に書かせない方針と逆方向です。

同様に、`rules.suppressions[].expires` も README で `YYYY-MM-DD` としているので、schema では `format: date`、または厳しめにするなら `pattern: "^\\d{4}-\\d{2}-\\d{2}$"` を入れるとよいです。

## P1: public schema に残る alias / 重複語彙をさらに削る

v1alpha は canonical YAML surface を小さくする方針なので、schema に残る alias 的な enum は減らした方がよいです。

### `dialect`

schema では `googlesql`, `google_sql`, `google-sql`, `postgresql`, `postgres` が許容されています。README は「default は GoogleSQL、非 default dialect だけ書く」と説明しています。

canonical schema としては、次のどちらかで十分だと思います。

```yaml
# 省略時は googlesql
# 明示するなら canonical spelling のみ
dialect: postgresql
```

または、どうしても GoogleSQL を明示可能にするなら:

```text
dialect enum: googlesql | postgresql
```

`google_sql`, `google-sql`, `postgres` は loader の内部 alias として受け付けるにしても、public schema には載せない方が v1alpha contract は単純です。特に `config-schema` を editor integration に使うなら、schema には canonical spelling だけが出る方がよいです。

### `projection.named_schema_tables`

schema は `error`, `reject`, `omit`, `warn_and_omit` を許しているように見えます。README の説明に合わせるなら、v1alpha は次で足ります。

```text
unsupported_columns: error | omit
named_schema_tables: error | warn_and_omit
```

`error` と `reject` は意味が重複します。`named_schema_tables: omit` も、warning なしに lossy projection を許すなら、これまでの「lossy projection は reviewable にする」という方針と少しズレます。silent omit が本当に必要になるまでは、`warn_and_omit` だけで十分だと思います。

## P1: `conflict.strategy: insert_or_update` は v1alpha では削れるかもしれない

ここは設計判断ですが、さらに simple にするなら検討価値があります。

今の README は「`upsert` maps to Spanner `INSERT OR UPDATE` for now」と明記しています。v1alpha で唯一の upsert strategy が `insert_or_update` なら、次の `conflict` block は冗長です。

```yaml
conflict:
  strategy: insert_or_update
```

より単純な v1alpha config はこうです。

```yaml
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
```

そして normalized plan 側に次を出します。

```yaml
operation: upsert
conflict_strategy: insert_or_update
normalized_insert_columns: [SingerId, FirstName, LastName]
normalized_update_columns: [FirstName, LastName]
```

将来 `ON CONFLICT DO UPDATE` を入れる時に `conflict` block を追加してもよいです。v1alpha は mutable preview なので、現時点で将来用 block を public config に残す必要は薄いです。

ただし、`conflict.strategy` を残す判断も完全に間違いではありません。将来拡張の場所を先に見せるという利点があります。単純さを優先するなら削る、将来の構造を先に見せるなら残す、というトレードオフです。私は v1alpha では削る方を推します。

## P1: Mutation `InsertOrUpdate` を捨てていないことを README に明記する

途中で確認した通り、mutation の `InsertOrUpdate` を活用しない、という方針ではありません。`operation: upsert` で mutation helper を emit する場合は、むしろ `spanner.InsertOrUpdate` / `InsertOrUpdateStruct` 系を使うのが自然です。

ただし、Mutation `insertOrUpdate` には重要な制約があります。

- 行が存在する場合、指定 column values は上書きされる。
- 明示されなかった column values は保持される。
- しかし、`insert` と同じく、すべての NOT NULL columns の値が必要。
- これは、行が既に存在して実際には update になる場合でも同じ。

そのため、Mutation `InsertOrUpdate` は「存在すれば少数 column だけ update、存在しなければ別の column set で insert」という branch-different upsert を自然には表せません。v1alpha の `insert.columns == key + update.columns` invariant は、DML `INSERT OR UPDATE` だけでなく mutation `InsertOrUpdate` に対しても安全な整理です。

README の Writes に短い注記を足すと誤解が減ります。

```text
When mutation helpers are enabled, `operation: upsert` emits a Spanner
InsertOrUpdate mutation helper. This does not allow branch-different column
sets: InsertOrUpdate uses one written column set, preserves unspecified columns
on existing rows, and still requires all NOT NULL insert columns to be supplied.
```

## P1: `result.cardinality` は v1alpha からさらに削れる可能性がある

README は「query method surfaces are reserved and rejected in v1alpha」と明記しています。一方で schema は `result.cardinality: one | maybe_one | many | exec` を受け付け、README も「plan に記録されるが DTO shape は変えない」と説明しています。

これは十分説明されているので許容できます。ただ、さらに simple にするなら、v1alpha では `cardinality` を public YAML から外してもよいです。

理由は、現時点で query methods が rejected なら、`cardinality` は利用者にとって「書けるが生成結果にほぼ影響しない field」に見えやすいからです。

選択肢は 2 つです。

### 案 A: v1alpha から外す

```yaml
result:
  struct: SingerRow
```

`one`, `maybe_one`, `exec` は query method surface を入れるタイミングで再導入します。これが一番 simple です。

### 案 B: 残すが “plan annotation only” として強く書く

今の README はかなり近いです。残すなら schema description も次のようにするとよいです。

```text
Plan annotation only in v1alpha. Does not change generated DTOs or emit query methods.
```

私は v1alpha の単純さを優先するなら案 A、将来の query file annotation との接続を早めに保ちたいなら案 B がよいと思います。

## P1: `result.required.policy` は enum にする

schema では `required.policy` が free string に見えます。DESIGN には `strict` が出ており、以前の README では `override` / `strict` のような語彙がありました。

v1alpha で受けるなら、schema では次に固定した方がよいです。

```json
"policy": {
  "enum": ["override", "strict"]
}
```

もし v1alpha で `policy` をまだ本格対応しないなら、`result.required.fields` だけにして `policy` は future work に戻す方が単純です。

## P1: top-level `catalogs` / `queries` / `writes` の requiredness を決める

schema の top-level required は `version` と `go` だけのように見えます。これは tooling 的には便利かもしれませんが、generator config としては `catalogs` も、少なくとも `queries` または `writes` のどちらかも必要に見えます。

おすすめはどちらかです。

### 案 A: schema でも実用 config を要求する

```text
required: version, go, catalogs
catalogs: minItems 1
anyOf:
  - required: queries
  - required: writes
queries/writes, if present: minItems 1
```

### 案 B: schema は空 generator config を許すが planner が warning/error

この場合は README に次を明記します。

```text
The JSON Schema allows partial files for editor support, but `generate`, `check`,
`explain-plan`, and `vet` require at least one declared query or write after
normalization.
```

v1alpha を CI contract として使うなら案 A がよいと思います。

## P2: `input` reuse の read/write safety を README にも短く出す

DESIGN では、shared DTO の read/write roles を分ける方針がかなり明確です。`decode_result`, `encode_key`, `encode_insert_value`, `encode_update_value` のような role tracking もよいです。

README にはまだこの説明が少なめなので、Writes 付近に短く足すとよいです。

```text
`input` names a write input struct. It may reuse a query result struct only when
that struct already contains every key and value field required by the write and
each field is valid for the required Spanner encode role. The generator does not
silently add write-only fields to a query DTO.
```

これは、shared DTO が「便利な merge」から「意図せず write input になる」事故を避けるために重要です。

## P2: `EXTERNAL_QUERY` の `outer_sql` placeholder 制約は diagnostics ID を固定する

README は `outer_sql` が exactly one `__external__` token を含む必要がある、と説明しています。DESIGN では token として検出し、string literal / quoted identifier / comment 内は数えない、という説明もあります。

ここは schema ではなく planner でしか検証できません。したがって、diagnostic ID を固定しておくとよいです。

```text
external-query-placeholder-missing
external-query-placeholder-ambiguous
external-query-placeholder-invalid-position
raw-external-query-sql-rejected
```

このあたりは golden test にしやすく、後から CLI output を変えにくい部分なので、早めに名前を固定する価値があります。

## P2: `config-schema` の “canonical” と loader alias の関係を明文化する

仮に loader が一時的に `google_sql` などの alias を受け付ける場合でも、`config-schema` は canonical shape だけを出す、という方針を README に書くとよいです。

```text
`config-schema` describes the canonical public v1alpha YAML. The loader may
accept temporary migration aliases before v1alpha, but aliases are not part of
the schema and may be removed without notice.
```

今の方針では compact config は v1alpha 前に削るため、この文は不要かもしれません。ただ、schema に alias enum が残るなら、この線引きが必要です。

## 追加で送るならこの短文

> 最新版はかなり整理されています。大きな方向転換は不要です。次に見るべき主対象は JSON Schema です。README では `config-schema` を public v1alpha YAML contract として扱っているため、schema も最低限の discriminated union を表した方がよいと思います。具体的には、`queries[].kind` ごとの required/forbidden fields、`writes[].operation` ごとの required/forbidden fields、`verification_evidence` の required fields、`expires` / `checked_at` の date/date-time format を schema で固定してください。DDL 依存の column existence や upsert invariant は planner validation のままでよいです。
>
> さらに単純化するなら、v1alpha では `conflict.strategy: insert_or_update` を public YAML から外してもよいです。README がすでに `operation: upsert` は Spanner `INSERT OR UPDATE` に map すると書いているため、唯一の strategy を毎回書かせるのは冗長です。normalized plan には `conflict_strategy: insert_or_update` を出せば reviewability は保てます。将来 `ON CONFLICT` を入れる時に `conflict` block を追加できます。
>
> mutation の `InsertOrUpdate` は捨てているわけではありません。`operation: upsert` の mutation helper としては使うべきです。ただし、Mutation `insertOrUpdate` も one written column set の upsert であり、NOT NULL insert columns の値が必要です。そのため、insert と update で列集合を変えられる仕様にはしない、という現在の invariant は正しいです。この点は README の Writes に一文足すと誤解が減ります。
>
> 最後に、canonical schema に `google_sql` / `google-sql` / `postgres` のような alias、`error` / `reject` のような重複語彙、`named_schema_tables: omit` のような silent lossy projection が残っているなら、v1alpha では削ることを推します。schema は canonical spelling だけを示す方が、editor integration と CI contract として読みやすいです。

## 結論

ここまでの整理で、仕様の大枠はかなり収束しています。

残る重要論点は次の 5 つです。

1. JSON Schema を public contract とするなら、最低限の kind/operation 条件を入れる。
2. `verification_evidence` を schema 上も real evidence にする。
3. canonical schema から alias / 重複 enum を削る。
4. さらに simple にするなら `conflict.strategy` と `result.cardinality` を v1alpha から外すか、少なくとも “plan annotation only” を強く明記する。
5. Mutation `InsertOrUpdate` を upsert helper として使うが、branch-different upsert を許す根拠にはならない、と README に明記する。

実装レビューではないため、最終的には `config-schema --output json` の出力、`explain-plan --stable` の golden output、代表的な invalid config の diagnostics をセットで見せてもらえると、public contract としてかなりレビューしやすくなります。
