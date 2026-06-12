# spanner-query-gen plan contract round 15 response

レビューありがとうございます。round 15 の指摘は、新しい contract を増やすより
artifact contract を締める段階のものとして扱いました。破壊的変更はまだ許容される
前提なので、曖昧さが残る public field は整理しています。

## 反映したこと

### `target_summary` の invariant

`target_summary.included_count` を `queries[]` に出る target 数として定義し直し、
次の invariant を docs と tests で固定しました。

```text
target_summary.included_count
  == target_summary.planned
   + target_summary.errors
   + target_summary.skipped
```

`target_summary.excluded` は always present にしました。除外/skip がない場合も
`[]` として出ます。`excluded_target` は canonical artifact として `id`,
`query`, `scope`, `reason` を required にしています。

### `queries[].status: error | skipped` の partial-field semantics

`PLAN_CONTRACTS.md`, `DESIGN.md`, `README.md`, schema description に、reliable な
partial field を明記しました。

`error | skipped` で reliable とみなすのは、target/query identity、SQL と SQL
digest、DDL resolution 済みの場合の DDL digest、そして `error` です。
`operator_tree_sha256`, `operator_family_counts`, `normalized_operators`,
`operator_edges`, `plan` は successful plan evidence ではない、と明記しました。

実装テストでも、Analyze 前に失敗した `error` query と `skipped` query に stale な
plan normalization fields が混入しないことを固定しました。

### `aggregate` / `join` generic family の意味

`aggregate` と `join` は derived umbrella ではなく、generic fallback concrete
family として明文化しました。

- `aggregate`: `Aggregate` PlanNode だが `hash_aggregate` /
  `stream_aggregate` に分類できない fallback
- `join`: join-like PlanNode だが `hash_join`, `merge_join`, `apply_join`,
  `push_broadcast_hash_join` などに分類できない fallback

したがって `aggregate` は `hash_aggregate + stream_aggregate` の別名ではなく、
direct `forbid.operator_family: aggregate` は fallback aggregate node だけを対象に
します。

### `contract_evaluation.backend` の削除

`contract_evaluation.backend` は top-level `backend` / `plan_source.backend` と
重複していたため、public report schema/output から削除しました。CEL の
`backend` input は report-level backend として引き続き残しています。

### `diagnostic_id` pattern

`contract_rule_result.diagnostic_id` に軽い pattern を付けました。

```text
^[a-z][a-z0-9_]*(\.[a-z0-9_]+)+$
```

### function hint observation

`DISABLE_INLINE` function hint の観測は、すでに
`QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md` の `Function Hint Shapes` に同期済み
です。今回新しい predefined contract にはしていません。

## 反映しなかったこと

`queries[].status: skipped` の `error` field を `reason` / `message` に rename する
案は今回は見送りました。`not_evaluated.reason` との整理まで含むと artifact 全体の
語彙変更になるため、まずは docs で `error` が acquisition/skip message であることを
明確化する方に寄せています。

新しい predefined contract、positive `require_*`、PROFILE/runtime statistics、
CEL macro、main config への plan contract 統合も追加していません。レビュー方針通り、
v1alpha の public artifact を締めることを優先しました。

## 検証

```sh
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go test ./cmd/spanner-query-gen ./internal/plancontract
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go test ./...
```
