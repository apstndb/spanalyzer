# spanner-query-gen plan contract round 16 response

レビューありがとうございます。round 16 の指摘は、v1alpha の public artifact
contract を締めるものとして扱いました。今回も新しい predefined contract を増やす
より、schema/docs/runtime invariant の整合性を優先しています。

## 反映したこと

### `queries[].status: error | skipped` の schema 制約

選択肢 B を採用しました。`error | skipped` では target identity と `error` が
reliable であり、`sql`, `sql_sha256`, `ddl_sha256` は出力されている場合だけ
reliable という扱いです。

schema でも success-only plan fields を禁止しました。

```text
operator_tree_sha256
operator_families
operator_family_counts
normalized_operators
operator_edges
plan
classification_warnings
```

`operator_edges` は success-only field ですが、成功時でも edge が空なら serialized
output で省略され得ます。そのため `status: ok` の required field からは外し、
`error | skipped` では forbidden として扱っています。

### `target_summary.excluded` と `queries[].status: skipped`

用語を分けました。

```text
included target:
  appears in queries[]
  counted by target_summary.included_count
  status is ok | error | skipped

excluded target:
  does not appear in queries[]
  appears only in target_summary.excluded
  not counted by target_summary.included_count
```

非 Spanner source や source missing は `queries[]` に `status: skipped` として残し、
`target_summary.skipped` に数えます。`target_summary.excluded` には重複して入れない
ようにしました。

### join-specific contract の `classification_unknown`

aggregate と同じ考え方を join にも適用しました。

`join_family_unknown` warning がある場合、join-specific predefined contract または
direct `forbid.operator_family` は fail します。

```text
failure_kind: classification_unknown
diagnostic_id: plan.join_classification_unknown
matched_operator_indexes: [generic join node indexes]
```

generic fallback 自体を禁止する `forbid.operator_family: join` は通常の violation と
して扱います。これは「未分類 join を許さない」明示的な contract として使えます。

### `excluded_target.reason` pattern

`excluded_target.reason` は enum ではなく namespaced pattern にしました。

```text
^[a-z][a-z0-9_]*(\.[a-z0-9_]+)+$
```

理由は diagnostic ID と同様に、将来の追加に対して schema を頻繁に壊さずに済むため
です。

### `source: use/<name>` と `predefined: <name>` の equality

JSON Schema では値 domain を締め、`source: use/<name>` と
`predefined: <name>` の一致は runtime invariant として固定する方針を
`PLAN_CONTRACTS.md` に明記しました。

### `status: no_targets` の例

`no_targets` の最小例と、`--contracts` 指定時でも target が unavailable なら
contract は `not_evaluated` になることを docs に追加しました。

## 追加で反映したこと

### root-partitionable contract の distributed fragment 判定

レビュー後の追加検証で、SELECT 系 plan では `subquery_cluster_node` metadata が
distributed fragment 判定として使える見込みが強いことを確認しました。これに合わせて
root-partitionable の CEL 例を、operator name の列挙ではなく metadata ベースに寄せました。
この metadata は raw PlanNode から plan-report の `normalized_operators[]` にコピー
されるもので、`spannerplan reference` などの human-readable rendered plan では意図的
に非表示であることも docs に明記しました。

```cel
operators.filter(o,
  o.subquery_cluster_node != "" &&
  o.call_type != "local").size()
```

`Local Distributed Union` は `subquery_cluster_node` を持ちますが、`call_type: local`
なので除外します。これで将来 Spanner が distributed operator の display name を追加
しても、metadata が維持される限り contract を更新せず検出できます。

なお DML plan の `Apply Mutations` は distributed operator として見える一方で
`subquery_cluster_node` を持たない counterexample でした。root-partitionable の対象は
`PartitionQuery`/SELECT shape のレビューなので、現時点ではこの CEL 例の対象外として扱います。

## 反映しなかったこと

### `target_summary.excluded_count`

今回は追加していません。`target_summary.excluded` は always present であり、
`len(excluded)` で十分に取得できます。v1alpha schema はすでに
`included_count`, `planned`, `errors`, `skipped` の invariant を持つため、同じ情報を
重複する count field として増やすより、まずは語彙を絞る方を優先しました。

## 検証

```sh
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go run ./cmd/spanner-query-gen plan-report-schema --out schemas/spanner-query-gen.plan-report.v1alpha.schema.json
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go test ./cmd/spanner-query-gen ./internal/plancontract
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go test ./...
```
