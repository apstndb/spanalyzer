# `spanner-query-gen` v1alpha `cardinality: exec` review への回答

## 回答

レビューありがとうございます。今回の指摘は妥当だと判断します。

特に、`exec` にはユースケースがあるが、`queries[].result.cardinality` に混ぜる
べきではない、という整理に同意します。row-count-only custom DML と
row-returning DML は、SELECT-like query DTO generation とは別の surface として
扱う方がよいです。

## 採用する方針

### `conflict.strategy`

採用します。

`operation: upsert` の public YAML から `conflict.strategy` を削除します。
現時点では Spanner `INSERT OR UPDATE` 以外の strategy がないため、config に
唯一値を書かせる必要はありません。

`conflict.strategy: insert_or_update` は resolved plan の normalized annotation として
出します。将来 `ON CONFLICT` semantics を入れる段階で、必要なら public YAML に
`conflict` block を再導入します。

### `queries[].result.cardinality: exec`

採用します。

v1alpha の `queries[].result.cardinality` から `exec` を削除します。
`queries` は SELECT-like / row-returning query result DTO と SQL constant を扱う
surface とし、`result.cardinality` は row-returning result の cardinality だけを
表します。

v1alpha の enum は次に絞ります。

```text
one | maybe_one | many
```

README には次を明記します。

- `result.cardinality` は row-returning query result 用。
- row-count-only DML execution は `cardinality: exec` では扱わない。
- DML `THEN RETURN` は `exec` ではなく row-returning DML。
- custom DML / DML returning は future `commands` surface で扱う。

### custom DML

今回は v1alpha public contract には入れません。

custom DML のユースケースは認めます。特に次は実用上必要です。

- row-count-only DML:
  - `UPDATE ... WHERE ...`
  - `DELETE ... WHERE ...`
- row-returning DML:
  - `INSERT ... THEN RETURN ...`
  - `UPDATE ... THEN RETURN ...`
  - `DELETE ... THEN RETURN ...`
  - `INSERT OR UPDATE ... THEN RETURN WITH ACTION ...`

ただし、これは `queries` ではなく future `commands` として設計します。

想定する方向性は次です。

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

この分離により、`THEN RETURN` を `exec` と誤解せず、row-returning DML として
`returning.cardinality` で扱えます。

## schema / README / DESIGN の同期

指摘どおり、回答文と README / JSON Schema の同期がまだ不足しています。
次の作業として反映します。

1. schema から `write.conflict` を削除する。
2. README の upsert example から `conflict.strategy` を削除する。
3. plan example にのみ `conflict.strategy: insert_or_update` を出す。
4. schema の `result.cardinality` enum から `exec` を削除する。
5. README に custom DML / DML `THEN RETURN` は future `commands` surface で扱うと明記する。
6. `params[].scope` を `external_query` 専用にする。
7. `update.all_non_key_columns` を `const: true` にする。
8. string / array の `minLength`, `minItems`, `uniqueItems` を追加する。

## 保留する点

`commands` surface は今回すぐには入れません。

理由は、v1alpha の現在の目的が config contract の単純化と query DTO / SQL constant
generation の安定化にあるためです。custom DML は重要ですが、row-count-only DML と
`THEN RETURN` を含めると、method generation、transaction API、affected row count、
returning DTO、Spanner / BigQuery の適用範囲などを同時に設計する必要があります。

したがって、正式 v1 前に入れるとしても、`queries` に `exec` を混ぜるのではなく、
`commands` として別レビューにします。

## 現時点の結論

v1alpha では `queries[].result.cardinality` から `exec` を削除します。

`exec` 相当の row-count-only DML は必要ですが、future `commands[].result.mode:
row_count` として扱います。DML `THEN RETURN` は `exec` ではなく
`commands[].returning.cardinality` として扱います。
