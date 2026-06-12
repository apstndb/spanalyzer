# `spanner-query-gen` plan contract round 12 response

作成日: 2026-05-06

## 返答概要

round 12 の指摘は、report artifact と CEL 入力の public contract を締めるものとして妥当だと判断しました。今回の対応では、機能追加よりも artifact を読めば原因を追えることを優先し、schema / docs / runtime behavior の整合を強めました。

## 反映した内容

### `subtree_family_counts` を complete map に統一

`operator_family_counts` と同じ方針で、各 `operators[].subtree_family_counts` も known operator family をすべて含む complete map としました。これにより、CEL では次のような direct indexing を前提にできます。

```cel
operators.exists(o,
  o.family == "hash_aggregate" &&
  o.subtree_family_counts["scan"] == 1)
```

missing key の挙動に依存しないため、top-level counts と subtree counts の使い勝手が揃います。

### topology fields を schema-required に変更

`operators[].child_indexes`、`operators[].descendant_indexes`、`operators[].subtree_family_counts` は v1alpha の normalized plan surface として扱うことにし、plan report schema でも required にしました。

空の child / descendant は空配列で出力され、subtree counts は zero-filled complete map として出力されます。

### `matched_operator_indexes` の emission rule を固定

`rule: forbid_operator_family` の評価結果では、`matched_operator_indexes` を常に出力するようにしました。

- 一致する operator がない場合は `matched_operator_indexes: []`
- 一致する operator がある場合は、対応する `normalized_operators[].index` を昇順・重複なしで出力
- `rule: cel` では v1alpha では出力しない

schema でも、`forbid_operator_family` では required、`cel` では forbidden になるようにしました。

### `forbid[n]` の意味を明文化

`source: forbid[n]` の `n` は、該当 contract の `forbid` array における 0-based YAML order index と定義しました。正規化や展開後の内部 index ではなく、ユーザーが元 YAML に戻るための source location として扱います。

### `source` / `predefined` invariant の固定

次の invariant を維持しています。

- `source: use/<name>` の場合、`predefined: <name>`
- `source: forbid[n]` の場合、`predefined` は出力しない
- `source: cel` の場合、`predefined` は出力しない

schema では値域と presence / absence を制約し、`source use/<name>` と `predefined <name>` の equality は runtime / regression test 側で固定する方針です。per-name の JSON Schema `if/then` を大量生成するより、v1alpha ではこの分担の方が単純だと判断しました。

### `contract_summary` の derived counts を invariant として文書化

`contract_summary` について、次の関係を docs に明記しました。

- `contracts == len(contract_evaluations)`
- `passed == count(status == pass)`
- `failed == count(status == fail)`
- `not_evaluated == count(status == not_evaluated)`

これは schema で表現しにくいため、runtime behavior と regression test で固定します。

### Sort family を分離

ユーザーから共有された実行計画上、`Sort` と `Minor Sort` は最適化レビュー上の意味が異なるため、operator family を次のように分けました。

- `full_sort`: `Sort`, `Sort Limit`
- `minor_sort`: `Minor Sort`, `Minor Sort Limit`
- `explicit_sort`: `full_sort + minor_sort` の umbrella family

predefined contract も追加しました。

- `no_full_sort`
- `no_minor_sort`

既存の `no_explicit_sort` は、従来通り full sort と minor sort の両方を禁止する umbrella contract として残しています。

### Spool operator を既知 family に追加

CTE を複数箇所で参照するクエリで `SpoolBuild` と `SpoolScan` を確認できるため、known operator family に追加しました。

```sql
WITH CTE AS (SELECT 1 AS PK, "foo" AS col)
SELECT * FROM CTE c1 JOIN CTE c2 USING (PK);
```

plan-shape probe の再現ケースと `spanner-hacks` 向け feedback にも追加しました。

## 残した判断

### `source` / `predefined` equality は schema だけでは固定しない

round 12 の提案通り、JSON Schema だけで equality を完全表現する必要はないと判断しました。schema は artifact shape と値域を固定し、値同士の対応は runtime validation / regression test に寄せます。

### PROFILE / execution stats は対象外のまま

plan contract は引き続き PLAN-only とします。Spanner Omni の `EXPLAIN ANALYZE QUERY` や execution stats を使う PROFILE 系の contract は、v1alpha の対象外です。

## 確認した仕様面の状態

今回の更新後、外部レビュー可能な contract としては次の状態です。

- `operator_family_counts` は complete map
- `operators[].subtree_family_counts` も complete map
- `operators[].child_indexes` / `descendant_indexes` / `subtree_family_counts` は schema-required
- `matched_operator_indexes` は `forbid_operator_family` で schema-required
- `matched_operator_indexes` は `cel` で schema-forbidden
- `explicit_sort` は `full_sort + minor_sort` の umbrella family
- `no_full_sort` / `no_minor_sort` を predefined contract として利用可能
- `SpoolBuild` / `SpoolScan` は known family として count / CEL / report の対象

## 内部検証

開発側では次を確認しました。

```sh
go test ./cmd/spanner-query-gen
go test ./internal/plancontract
go test ./tools/spanner-query-plan-shape
go test ./...
git diff --check
```
