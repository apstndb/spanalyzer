# `spanner-query-gen` v1alpha 最新状況への再レビュー

## 前提

今回も実装コード・実行ログ・テスト出力は共有されていないため、レビュー対象は次の公開仕様面に限定する。

- `spanner-query-gen-v1alpha-cardinality-exec-review-response-ja.md`
- `README(15).md`
- `DESIGN(15).md`
- `spanner-query-gen.v1alpha.schema(3).json`

結論として、方向性はかなり良い。特に、`queries` を SELECT-like / row-returning query DTO + SQL constant の surface として保ち、row-count-only DML や DML `THEN RETURN` を future `commands` surface に分ける判断は、v1alpha の小ささを保つうえで妥当である。

ただし、回答文で採用した方針と README / DESIGN / JSON Schema の反映がまだ同期しきっていない。次の作業は新しい設計論点を増やすことではなく、**採用済みの単純化方針を public contract に反映しきること**だと思う。

---

## 総評

最新回答は、前回の `cardinality: exec` 論点に対してかなり良い整理をしている。

- `exec` には実ユースケースがある。
- ただし `queries[].result.cardinality` に混ぜるべきではない。
- row-count-only DML と row-returning DML は SELECT-like query DTO generation とは別 surface として扱う。
- v1alpha では `commands` をまだ public contract に入れない。
- `THEN RETURN` は `exec` ではなく row-returning DML として、将来 `commands[].returning.cardinality` で扱う。

この整理は正しい。DML `THEN RETURN` は行を返す DML であり、row-count-only execution とは違う。したがって、`exec` を `one | maybe_one | many` と同じ `queries[].result.cardinality` に置くと、将来の method generation や DTO generation で混乱しやすい。

`commands` を今すぐ入れない判断も良い。custom DML を入れると、row count、returned rows、transaction API、Partitioned DML、DML statements vs execution helpers、Spanner-only vs BigQuery DML などの設計を一度に決める必要が出る。v1alpha の目的が config contract の単純化と query DTO / SQL constant generation の安定化なら、これは後回しでよい。

---

## P0: 回答文と README / DESIGN / JSON Schema の同期

最新回答では、`operation: upsert` の public YAML から `conflict.strategy` を削除し、normalized plan にだけ `conflict.strategy: insert_or_update` を出す方針になっている。また、`queries[].result.cardinality` から `exec` を削除し、custom DML / DML returning は future `commands` に送る方針になっている。

しかし、最新 README / DESIGN / schema にはまだ古い shape が残っている。

### 1. README にはまだ `conflict.strategy` が残っている

README の Writes example にはまだ次の block がある。

```yaml
conflict:
  strategy: insert_or_update
```

また、その後の説明も `For conflict.strategy: insert_or_update` から始まっている。

回答方針を採用するなら、README は次のようにするのがよい。

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

説明文はこう直す。

```text
For operation: upsert, v1alpha emits Spanner INSERT OR UPDATE semantics.
Spanner updates every non-key column listed in the INSERT OR UPDATE statement
when the row already exists. v1alpha therefore rejects insert-only non-key
columns.

The resolved plan records conflict_strategy: insert_or_update as normalized
semantics. Public v1alpha YAML does not expose a conflict block.
```

### 2. JSON Schema にはまだ `query_result.cardinality: exec` がある

schema の `query_result.cardinality` enum はまだ次のようになっている。

```json
["one", "maybe_one", "many", "exec"]
```

回答方針どおりなら、v1alpha public schema は次に絞るべき。

```json
["one", "maybe_one", "many"]
```

`exec` は future `commands[].result.mode: row_count` または similar に送る。

### 3. JSON Schema にはまだ `write.conflict` がある

schema にはまだ `write.properties.conflict.strategy.enum = ["insert_or_update"]` が残っている。回答方針を採用するなら、public YAML schema から `conflict` property を削除し、すべての write operation で `conflict` を unknown field として reject する方が一貫する。

将来 `ON CONFLICT` を入れる時に、`operation: upsert` と一緒に `conflict` を再導入すればよい。

### 4. DESIGN の Current Scope / Generated Go Shape にも `exec` と `conflict strategy` が残っている

DESIGN には、Current Scope と Query Client Methods 付近に `exec` が残っており、Write Helpers には「upsert should model insert columns, update masks, and conflict strategy separately」という表現が残っている。

public YAML から削るなら、DESIGN は次のように分けるとよい。

```text
Current v1alpha config:
  - query result cardinality: one | maybe_one | many
  - custom DML and row-count-only execution are not part of v1alpha queries
  - upsert public config does not expose conflict

Resolved plan:
  - upsert records normalized conflict_strategy: insert_or_update

Future commands:
  - commands[].result.mode: row_count
  - commands[].returning.cardinality: one | maybe_one | many
  - future ON CONFLICT may reintroduce a conflict block
```

---

## P0: JSON Schema を public contract とするなら、今回の約束を schema に反映する

README は `config-schema` を editor integration / CI validation 用の reviewable contract と位置づけている。そうであれば、schema は planner の前段で弾けるものをできるだけ弾くべき。

最新 schema を直接確認した限り、次の強化がまだ不足している。

### 1. `minLength`, `minItems`, `uniqueItems`

回答文では `minLength`, `minItems`, `uniqueItems` を追加する方針になっているが、schema では多くの string / array にまだ入っていない。

最低限、次は schema で弾いてよい。

```text
name / catalog / table / input / rule / reason / owner: minLength: 1
insert.columns / update.columns / key / key_prefix / required.fields: minItems: 1, uniqueItems: true
queries / writes / catalogs: minItems: 1, uniqueItems は難しければ planner でもよい
```

DDL 依存の column existence や primary key inference は planner でよいが、空文字・空配列・重複 column は config schema の責務に寄せた方が、CI contract として使いやすい。

### 2. catalog kind-specific shape

回答文では catalog kind の discriminated union を採用するとしていたが、schema の `catalog` はまだ `kind: spanner` / `kind: bigquery` で許可 field を分けていないように見える。

v1alpha schema としては次を目指した方がよい。

```text
kind: spanner:
  allowed: name, kind, ddl, dialect, proto_descriptors
  forbidden: project, bindings

kind: bigquery:
  allowed: name, kind, ddl, dialect, project, bindings
  forbidden: proto_descriptors
```

`bindings` は BigQuery catalog 専用、`proto_descriptors` は Spanner catalog 専用とした方が、config の意図が明確になる。

### 3. `params[].scope` は external_query 専用にする

DESIGN/README の方針どおり、regular `sql` / `table` / `index` query では `params[].scope` を書けないようにした方がよい。

schema で完全に表すのが重ければ planner validation でもよいが、`query.kind` の oneOf 内で regular query の `params.items` に `not: { required: ["scope"] }` を入れる設計は検討に値する。

---

## P1: `commands` は future work のままで良いが、名前だけ先に安定させる

今は `commands` を v1alpha に入れない判断でよい。ただし DESIGN にはまだ `exec_returning: one|many` のような表現が残っている。これは `exec` という語を再利用しており、今回避けたかった曖昧さを少し戻してしまう。

future work の名前だけでも、次のようにしておく方がよい。

```yaml
commands:
- name: ArchiveOldSingers
  catalog: app
  kind: dml
  sql: |
    UPDATE Singers
    SET Archived = TRUE
    WHERE LastSeenAt < @Cutoff
  result:
    mode: row_count
```

```yaml
commands:
- name: InsertSingerReturningID
  catalog: app
  kind: dml
  sql: |
    INSERT INTO Singers (SingerId, FirstName, LastName)
    VALUES (@SingerId, @FirstName, @LastName)
    THEN RETURN SingerId
  returning:
    cardinality: one
    struct: SingerIDRow
```

この語彙なら、row-count-only DML と row-returning DML が明確に分かれる。`exec_returning` よりも、`result.mode: row_count` と `returning.cardinality` の方が読みやすい。

---

## P1: `outer_sql` optional の仕様を README と schema で合わせる

DESIGN には、`outer_sql` を省略した場合は `SELECT * FROM EXTERNAL_QUERY(...)` になるという説明がある。一方 README の `EXTERNAL_QUERY` セクションは、`outer_sql` must contain exactly one `__external__` token とだけ読めるため、schema 上で `outer_sql` が optional であることと少しズレて見える。

どちらでもよいが、v1alpha の simple さを優先するなら README に一文足すのがよい。

```text
If outer_sql is omitted, the generated BigQuery SQL is SELECT * FROM EXTERNAL_QUERY(...).
If outer_sql is supplied, it must contain exactly one __external__ placeholder.
```

これで schema の optional `outer_sql` と documentation が揃う。

---

## P1: `conflict.strategy` を削るなら、upsert の説明は “statement column set” に寄せる

`conflict.strategy` という config block を削るなら、upsert 説明の主語は `conflict.strategy` ではなく `operation: upsert` にするのが自然。

おすすめの説明は次。

```text
operation: upsert emits Spanner INSERT OR UPDATE semantics in v1alpha.
The DML statement has one column list. When the row exists, Spanner updates
non-key columns in that statement column list and leaves unspecified columns
unchanged. Therefore v1alpha requires insert.columns to equal key plus
update.columns.
```

Mutation helper 側の説明も同じ列集合の話に寄せる。

```text
The mutation helper uses Spanner InsertOrUpdate with the same written column set.
It preserves unspecified columns on existing rows, but still requires all
insert-time NOT NULL values to be supplied.
```

この説明なら、DML と mutation の差を隠さず、branch-different column set ができない理由も伝わる。

---

## P2: 実装・テスト主張は README ではなく DESIGN/Implementation Notes に寄せてもよい

README には emulator integration test / Omni optional check へのかなり具体的な説明が入っている。プロジェクト内の README としては有用だが、外部レビューでは実装コードやテスト出力が共有されていないため、検証済み事実としては扱えない。

もし README を public UX docs として軽くしたいなら、詳細は DESIGN の implementation notes に寄せ、README では一文に留めてもよい。

```text
Generated Spanner shorthand SQL is covered by generator-specific integration tests.
```

ただし、これは必須の修正ではない。実際にリポジトリにテストがあるなら README に残してもよい。

---

## `cardinality: exec` についての最終整理

ユーザー視点では、`cardinality: exec` に対応するユースケースは確かにある。

- affected row count を返す custom DML
- `UPDATE ... WHERE ...`
- `DELETE ... WHERE ...`
- `INSERT OR UPDATE ...` の custom DML

しかし、それは `queries` の cardinality ではない。

`queries` は、今の設計では query result DTO と SQL constant を baseline output として生成する surface である。ここに `exec` を置くと、`result.struct` 必須の構造と矛盾しやすい。また、`THEN RETURN` は行を返すので `exec` ではなく returning result として扱うべき。

したがって、今回の採用方針どおりでよい。

```text
v1alpha:
  queries[].result.cardinality = one | maybe_one | many

future:
  commands[].result.mode = row_count
  commands[].returning.cardinality = one | maybe_one | many
```

v1alpha に custom DML を本当に入れたい場合でも、`queries[].result.cardinality: exec` を戻すのではなく、最小 `commands` surface を別に入れる方がよい。

---

## 優先順位

次にやるべきことは次の順だと思う。

1. README / DESIGN / schema から public YAML `conflict` block を削除する。
2. schema の `query_result.cardinality` enum から `exec` を削除する。
3. DESIGN の Current Scope / Query Client Methods から public v1alpha の `exec` を削り、future `commands` に移す。
4. schema に `minLength`, `minItems`, `uniqueItems` を入れる。
5. catalog kind-specific schema を入れる。
6. `params[].scope` を `external_query` 専用にする。
7. README の `outer_sql` optional/default 説明を schema と同期する。
8. DESIGN の `exec_returning` という仮名を `commands[].returning.cardinality` に寄せる。

---

## 結論

大きな設計変更は不要。最新回答の方向性は良い。

ただし、今の状態は **「採用方針は正しいが、README / DESIGN / JSON Schema がまだ一世代前の surface を一部含んでいる」** という段階に見える。

v1alpha を simple にするなら、次の canonical public YAML に完全に寄せるのがよい。

```text
version: v1alpha
go
emit
catalogs
queries    # row-returning query DTO / SQL constants only
writes     # declared Spanner write helpers only
rules.suppressions
```

そして、次は public YAML ではなく resolved plan / future surface に置く。

```text
conflict.strategy: insert_or_update   -> resolved plan annotation
cardinality: exec                     -> future commands[].result.mode: row_count
DML THEN RETURN                       -> future commands[].returning.cardinality
ON CONFLICT                           -> future commands conflict block
```

これで v1alpha の config がかなり読みやすくなり、将来の DML execution surface も無理なく追加できると思う。
