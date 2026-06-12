# `spanner-query-gen` v1alpha 最新版再レビュー: `cardinality: exec` と custom DML / `THEN RETURN`

作成日: 2026-05-05

## 前提

このレビューは、添付された最新の回答文、README、DESIGN、JSON Schema を対象にした仕様レビューです。実装コード、実行ログ、テスト出力は共有されていないため、実装済みかどうかは検証していません。

今回の焦点は次の 2 点です。

1. 最新回答で示された方針と、README / DESIGN / schema が同期しているか。
2. `result.cardinality: exec` を v1alpha に残すべきか。特に custom DML と Spanner DML `THEN RETURN` のユースケースをどう扱うべきか。

## 総評

設計全体はかなり収束しています。`v1alpha` を mutable preview channel として扱うこと、config / plan / generated Go を分離すること、`config-schema` を public YAML contract として出すこと、`EXTERNAL_QUERY` と external dataset を分けること、global `emit` に寄せることはよい方向です。

ただし、今回の最新版ではまだ **回答文と README/schema が完全には同期していません**。開発側の回答では `conflict.strategy` と `result.cardinality: exec` を public YAML から削る方針が書かれていますが、添付 README と JSON Schema にはまだ残っています。

この状態だと、外部レビュアーやユーザーから見ると「どちらが本当の v1alpha contract なのか」が分かりません。まずは次のどちらかに決める必要があります。

- 方針 A: 回答文どおり、public YAML から `conflict` と `exec` を削る。
- 方針 B: README/schema どおり、v1alpha に `conflict` と `exec` を残す。ただし意味論をもっと厳密化する。

私の推奨は、`conflict` は削り、`exec` は **そのまま `queries[].result.cardinality` に残すのではなく、custom DML 用の別 surface として残すかどうかを判断する**、です。

## `conflict.strategy` について

`conflict.strategy` は削る方針でよいと思います。

現時点の `operation: upsert` が Spanner `INSERT OR UPDATE` にしか対応しないなら、public YAML に次を書かせる必要は薄いです。

```yaml
conflict:
  strategy: insert_or_update
```

将来 `ON CONFLICT DO UPDATE` を入れるときに、`conflict` block を再導入すれば十分です。今は plan にだけ正規化して出す方がシンプルです。

推奨 public YAML:

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

推奨 normalized plan:

```yaml
writes:
- name: UpsertSingerName
  operation: upsert
  conflict:
    strategy: insert_or_update
  column_set_invariant:
    insert_columns_equal_key_plus_update_columns: true
```

README/schema にはまだ `conflict.strategy` が残っているため、回答文の方針を採用するなら README example と JSON Schema から削るべきです。

## `cardinality: exec` は本当にユースケースがあるか

結論から言うと、**ユースケースはあります**。ただし、`THEN RETURN` と `exec` を同じものとして扱うのは避けるべきです。

### 1. row-count-only custom DML には `exec` 相当のユースケースがある

例えば次のような custom DML は、行を返さず、実行結果として row count / execution metadata が欲しいだけです。

```sql
UPDATE Singers
SET Archived = TRUE
WHERE LastSeenAt < @Cutoff;
```

```sql
DELETE FROM Sessions
WHERE ExpiresAt < @Now;
```

Go client の DML execution でも、`ReadWriteTransaction.Update` は affected row count を返す API です。つまり、row-returning query とは別の execution metadata surface が実際に存在します。

この意味で `exec` 相当のモードは確かに有用です。

ただし、現在の `queries` は「query DTOs と SQL constants を baseline output とする」設計で、`result.struct` も要求します。この形のまま `cardinality: exec` を残すと、**行を返さない statement なのに result struct が必須**という不自然な contract になります。

したがって、row-count-only custom DML を v1alpha で扱うなら、`queries[].result.cardinality: exec` ではなく、別 surface にした方がきれいです。

推奨例:

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

または、`queries` に残すとしても `kind: dml` を導入し、`result.struct` を forbidden にする必要があります。

```yaml
queries:
- name: ArchiveOldSingers
  catalog: app
  kind: dml
  sql: |
    UPDATE Singers
    SET Archived = TRUE
    WHERE LastSeenAt < @Cutoff
  result:
    cardinality: exec
```

ただし、`kind: dml` を入れるなら、それはもはや query DTO generator ではなく DML command declaration です。README 上も `queries` ではなく `commands` と呼ぶ方が自然です。

### 2. DML `THEN RETURN` は `exec` ではなく row-returning DML

Spanner DML `THEN RETURN` は、行を返す DML です。公式 syntax でも `INSERT` / `UPDATE` / `DELETE` に `return_clause` があり、`INSERT OR UPDATE ... THEN RETURN WITH ACTION` の例もあります。

このため、`THEN RETURN` は `exec` ではなく、row-returning statement として扱うべきです。

例:

```sql
INSERT INTO Singers (SingerId, FirstName, LastName)
VALUES (@SingerId, @FirstName, @LastName)
THEN RETURN SingerId;
```

これは `exec` ではなく、返却行の cardinality を持つものです。

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

`UPDATE ... THEN RETURN` や `DELETE ... THEN RETURN` は 0 行、1 行、多数行のいずれもあり得るため、`one` / `maybe_one` / `many` の意味論が必要です。

つまり、`THEN RETURN` の存在は「`exec` を残すべき理由」ではなく、むしろ **`exec` と `returning` を分けるべき理由** です。

## v1alpha での推奨判断

### 推奨 1: まだ custom DML execution surface を出さないなら、`exec` は削る

v1alpha の現在の説明では、query methods は reserved / rejected であり、query DTOs と SQL constants が baseline output です。この範囲に留めるなら、`exec` は削るのが最もシンプルです。

この場合の schema:

```json
"cardinality": {
  "enum": ["one", "maybe_one", "many"]
}
```

そして README にはこう書くのがよいです。

```text
result.cardinality is a plan annotation for future row-returning query methods.
Supported values in v1alpha are one, maybe_one, and many.
Row-count-only DML and DML THEN RETURN are future command surfaces and are not
modeled by queries[].result.cardinality.
```

### 推奨 2: custom DML を v1alpha に入れたいなら、`commands` を入れる

ユーザー要件として custom DML が明確にあるなら、`exec` を完全に消すより、`commands` を導入する方が筋がよいです。

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
    INSERT INTO Singers (FirstName, LastName)
    VALUES (@FirstName, @LastName)
    THEN RETURN SingerId
  returning:
    cardinality: one
    struct: SingerIDRow
```

この形なら、次の分離が明確になります。

| statement type | config | Go output |
| --- | --- | --- |
| SELECT-like query | `queries` | DTO + SQL constant, future query method |
| DML without rows | `commands[].result.mode: row_count` | SQL constant, future DML exec helper returning row count |
| DML with `THEN RETURN` | `commands[].returning` | DTO + SQL constant, future DML returning helper |
| structured table write | `writes` | mutation helper / DML statement helper |

この整理ができるなら、`exec` を v1alpha に残す価値があります。ただし、名前は `cardinality: exec` より `mode: row_count` の方が分かりやすいです。

## 私の最終推奨

現時点では、**v1alpha の `queries[].result.cardinality` から `exec` は削る**のがよいと思います。

理由は次の通りです。

1. `queries` は baseline として DTO / SQL constants を出す設計になっている。
2. `result.struct` が必須のまま `exec` を許すと、row-count-only DML と矛盾する。
3. `THEN RETURN` は `exec` ではなく row-returning DML であり、別設計が必要。
4. custom DML は確かに有用だが、`queries` に混ぜるより `commands` として入れた方がよい。
5. v1alpha では config を小さく保つ方針と整合する。

ただし、もし開発側に「v1alpha で custom DML SQL constants / DML exec helper まで提供する」という強い意図があるなら、`exec` を削らずに **`commands` block を v1alpha に追加する**方がよいです。その場合でも、`THEN RETURN` は `exec` ではなく `returning.cardinality` で扱ってください。

## JSON Schema への追加フィードバック

最新 schema は discriminated union がかなり良くなっていますが、回答文で「採用する」と書かれている制約の一部が、添付 schema にはまだ完全には反映されていません。

### 1. `conflict` がまだ残っている

回答文では `conflict.strategy` を public YAML から削る方針ですが、schema には `write.conflict` が残っています。README の upsert example にも `conflict.strategy` が残っています。

削るなら以下を徹底してください。

- schema の `write.properties.conflict` を削除。
- write operation union から `conflict` 関連の validation を削除。
- README の example と説明を `operation: upsert` 単独に変更。
- plan example にのみ `conflict.strategy: insert_or_update` を出す。

### 2. `result.cardinality` がまだ `exec` を許している

回答文では `exec` を削る方針ですが、schema はまだ `one | maybe_one | many | exec` です。README も `cardinality` を plan annotation として説明しているため、ここは決定が必要です。

- 削るなら `exec` を enum から削る。
- 残すなら `result.struct` required と矛盾しないように discriminated union にする。

### 3. `params[].scope` は schema 上まだ全 query kind で許可される

回答文では `params[].scope` を `kind: external_query` 専用にするとありますが、schema の `query_param` は全 query kind で `scope` を許可しています。

planner で弾くことはできますが、`config-schema` を editor / CI contract として出すなら、できる範囲で schema 側にも寄せたいです。

理想:

- `kind: sql/table/index` の `params.items` は `scope` forbidden。
- `kind: external_query` の `params.items` は `scope: inner|outer` allowed。

### 4. `all_non_key_columns: false` がまだ入り得る

回答文では `update.all_non_key_columns` を `const: true` にするとありますが、schema の property 自体は `type: boolean` です。

さらに、`operation: update` で `update.columns` と `update.all_non_key_columns: false` を同時指定すると、`columns` branch として schema を通過し得ます。

推奨:

```json
"all_non_key_columns": { "const": true }
```

かつ `update.columns` branch では `all_non_key_columns` を forbidden にしてください。

### 5. `minLength`, `minItems`, `uniqueItems` がまだ不足している

回答文では空文字、空配列、重複 column list を schema で弾くとありますが、添付 schema では多くの string / array にまだ `minLength`, `minItems`, `uniqueItems` がありません。

少なくとも以下は schema で弾くとよいです。

- `name`, `catalog`, `table`, `index`, `input`, `rule`, `reason`, `owner`, `sql`, `inner_sql`, `outer_sql`
- `insert.columns`, `update.columns`, `key`, `key_prefix`, `required.fields`, `proto_descriptors`
- `catalogs`, `queries`, `writes`, `rules.suppressions`

DDL 依存の column existence は planner でよいですが、空文字・空配列・重複は schema で十分扱えます。

## README/DESIGN への追加フィードバック

### README の `cardinality` 説明を custom DML と切り分ける

現在の README は `result.cardinality` を future query method semantics として説明しています。この説明自体は良いですが、custom DML や `THEN RETURN` を考えるなら、次の一文を足すと誤解が減ります。

```text
result.cardinality describes row-returning query results. It is not used for
row-count-only DML execution. DML THEN RETURN is row-returning DML and will be
modeled by a future command/returning surface rather than by cardinality: exec.
```

### DESIGN の `THEN RETURN` 方針は維持してよい

DESIGN には、DML `THEN RETURN` は `exec_returning: one|many` のように別扱いすべき、という趣旨が残っています。この方針は正しいです。

ただし、`exec_returning` という名前はやや曖昧です。最終的には `returning.cardinality` の方が分かりやすいと思います。

```yaml
returning:
  cardinality: one
  struct: SingerIDRow
```

## 開発側へのそのまま送れる返答案

> 最新版はかなり収束していますが、回答文と README/schema がまだ同期していません。回答文では `conflict.strategy` と `result.cardinality: exec` を public YAML から削る方針ですが、README の upsert example と JSON Schema にはまだ `conflict` が残っており、schema の `result.cardinality` もまだ `exec` を許しています。まずこのどちらを v1alpha contract とするかを決めてください。
>
> `cardinality: exec` については、確かに row-count-only custom DML というユースケースはあります。ただし、`THEN RETURN` は `exec` ではなく row-returning DML です。Spanner DML の `THEN RETURN` は INSERT/UPDATE/DELETE/INSERT OR UPDATE の結果行を返すため、`one | maybe_one | many` の returning semantics で扱うべきです。したがって、`exec` を `queries[].result.cardinality` に残すのではなく、custom DML を v1alpha で扱うなら `commands` のような別 surface を導入し、row-count-only は `result.mode: row_count`、`THEN RETURN` は `returning.cardinality` として分けるのがよいと思います。
>
> v1alpha を simple に保つなら、現時点では `queries[].result.cardinality` から `exec` を削り、custom DML / DML returning は future `commands` に戻すのが一番きれいです。逆に、v1alpha で custom DML SQL constants や DML exec helper を本当に出すなら、`exec` を削らずに `commands` block を正式に入れるべきです。その場合でも `THEN RETURN` を `exec` にしないでください。
>
> schema については、`params[].scope` を external_query 専用にする、`all_non_key_columns` を `const: true` にする、`minLength` / `minItems` / `uniqueItems` を追加する、といった回答文で採用済みの制約がまだ完全には反映されていないように見えます。`config-schema` を public contract とするなら、ここを次の同期ポイントにするとよいです。

## 結論

`cardinality: exec` には確かなユースケースがあります。しかし、それは主に **row-count-only custom DML** のためのものであり、`THEN RETURN` のためではありません。

そのため、最も整理された仕様は次のどちらかです。

1. v1alpha では custom DML を扱わない。`queries[].result.cardinality` から `exec` を削る。
2. v1alpha で custom DML を扱う。`commands` を導入し、row-count-only と returning DML を分ける。

私は、現在の「simple v1alpha」方針を優先するなら 1 を推します。custom DML 需要がすでに強いなら 2 も十分ありですが、その場合は `queries` に `exec` を混ぜるのではなく、`commands` として明示的に設計した方が長期的にきれいです。
