# spanner-query-gen plan contract round 17 response

レビューありがとうございます。round 17 は大きな方向転換ではなく、v1alpha の
plan-report artifact と CEL input の誤読を減らす締め込みとして扱いました。
新しい predefined contract は追加していません。

## 反映したこと

### `operator_edges` を成功時 always present に変更

`queries[].status: ok` では `operator_edges` を schema required に戻しました。
成功 target で edge が空の場合も YAML/JSON では `operator_edges: []` として出ます。
一方、`status: error | skipped` では引き続き success-only plan field として禁止します。

実装では serialized report 専用の shape を用意し、内部構造体は従来どおり
`[]OperatorEdge` として扱っています。これにより CEL input と serialized artifact の
topology view が揃います。

### CEL input の optional metadata default semantics

`PLAN_CONTRACTS.md` に、CEL input は serialized YAML/JSON を直接読むのではなく
normalized operator object を使うことを明記しました。

- absent string metadata: `""`
- absent boolean metadata: `false`

対象例として `subquery_cluster_node`, `call_type`, `scan_target`,
`seekable_key_size`, `iterator_type`, `join_type` などを列挙しています。

### root-partitionable を PLAN-shape approximation として明確化

候補名・ドキュメント上の位置づけを保守的にしました。

- docs の例名を `ApproximateRootPartitionablePlanShape` に変更
- candidate 名としては `approximate_root_partitionable_plan_shape` を推奨
- 将来の API acceptance probe は `partition_query_accepts` のような別物として扱う方針を明記

`PartitionQuery` を直接呼ぶ検証とは分け、現在のものは PLAN artifact の近似 review
contract として閉じ込めています。

### `skipped` の `error` description を緩めた

schema 上の `queries[].error` description を
`Target acquisition error or skip message` に変更しました。

将来的に `reason` / `message` に分ける余地は残しつつ、現行 v1alpha では field を増やさず
「skipped の場合は failure とは限らない skip message」と docs に書いています。

### `excluded_target.reason` の例

`excluded_target.reason` は namespaced pattern のままにしました。現在の v1alpha 実装は
unavailable configured target を `queries[]` の `status: skipped` に残すため、
`target_summary.excluded` は通常 `[]` です。

その前提を明記した上で、将来 target-set pruning を入れる場合の reserved examples を
docs に置きました。

- `target.filtered_by_selector`
- `target.non_spanner_catalog`
- `target.missing_catalog`
- `target.external_dataset`

### CEL と `classification_unknown` の差分

predefined/direct `forbid` では aggregate/join の fallback family に対して
`classification_unknown` を保守的に fail させますが、CEL は literal evaluation であり、
この safeguard は自動適用されないことを明記しました。

同じ保守性が必要なら direct/predefined を使うか、CEL 側で
`operator_family_counts["join"]` / `operator_family_counts["aggregate"]` も見る、という説明に
しています。

### metadata-derived field の stability reason

CEL が `subquery_cluster_node` や `call_type` など raw PlanNode metadata 由来の
normalized field を参照した場合、`stability.tier` は `normalized_operator` のまま、
`stability.reasons` に次を追加するようにしました。

```text
contract references raw PlanNode metadata exposed through normalized_operators[]
```

raw `plan` / `nodes` を直接参照しているわけではないため `raw_query_plan` tier には落としていません。

## 反映しなかったこと

`skipped` の `error` を `reason` / `message` に分割する変更は今回入れていません。
v1alpha ではまだ破壊的変更可能ですが、ここは artifact 全体の field vocabulary 変更になるため、
まず description と docs で誤読を減らす方に留めました。

root-partitionable も predefined にはしていません。現在は candidate/CEL example のままです。

## 検証

```sh
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go run ./cmd/spanner-query-gen plan-report-schema --out schemas/spanner-query-gen.plan-report.v1alpha.schema.json
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go test ./cmd/spanner-query-gen ./internal/plancontract
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go test ./...
git diff --check
```
