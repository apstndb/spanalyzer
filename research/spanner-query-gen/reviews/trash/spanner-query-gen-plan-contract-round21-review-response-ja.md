# spanner-query-gen plan contract round 21 response

レビューありがとうございます。round 21 は、大きな仕様変更ではなく
plan-report / plan-contract artifact を外部 reviewer と CI consumer が同じ意味で
読めるようにする最終締め込みとして扱いました。v1alpha はまだ破壊的変更可能なので、
誤読を避ける効果が大きいものは互換性を残さず反映しています。

## 反映したこと

### raw CEL variable を `raw_plan` / `raw_nodes` に変更

指摘通り、serialized report の `queries[].plan` は human-readable rendered plan
string であり、CEL evaluator の raw `spannerpb.QueryPlan` と同じ名前にすると
外部 evaluator / reviewer が誤読しやすいと判断しました。

そのため v1alpha の破壊的変更として、CEL variable を次のように変更しました。

- `plan` -> `raw_plan`
- `nodes` -> `raw_nodes`

旧名の alias は残していません。v1alpha の public contract としては、曖昧な名前を
残すより、この段階で落とす方がよいと判断しました。

### raw QueryPlan CEL の replayability を artifact に出す

`contract_evaluations[].stability` に `replayable_from_report` を追加しました。

- normalized contract: `replayable_from_report: true`
- `raw_plan` / `raw_nodes` contract: `replayable_from_report: false`

raw QueryPlan CEL の `stability.reasons` には、
serialized report だけでは replay できないことも固定文言として出します。

### `contract_stability.reasons` を required + non-empty にした

plan-report schema の `contract_stability` は次を required にしました。

- `tier`
- `reasons`
- `check_recommended`
- `replayable_from_report`

`reasons` は `minItems: 1` です。これにより、normalized / metadata-derived /
raw QueryPlan のどれなのか、CI artifact から常に読み取れるようにしました。

### topology self-check を追加

`writePlanReport` の出力直前 self-check に、summary count invariant だけでなく
topology invariant も追加しました。

`queries[].status: ok` について、少なくとも次を検証します。

- `normalized_operators[].index` が unique
- `operator_edges[].parent_index` / `child_index` が既存 operator を指す
- `operators[].child_indexes[]` / `descendant_indexes[]` が既存 operator を指す
- `operators[].subtree_family_counts` がその operator の normalized subtree と一致する
- `queries[].operator_family_counts` が whole normalized operator list と一致する

また、contract result の `matched_operator_indexes` がある場合、それが target query の
`normalized_operators[].index` を指していることも self-check します。

### backend identity source を追加

`backend_identity.source` を plan-report schema / artifact に追加しました。
現時点では `source: spanemuboost` を出します。

Omni version と image digest はまだ `not_recorded` のままなので、
`backend_identity_not_recorded` warning は維持します。つまり、backend identity の
取得経路は見えるようにしつつ、version / digest が未記録である caveat は残します。

### `cel_input_defaults.applies_to` の canonical order を固定

schema で `prefixItems` を使い、次の順序を固定しました。

```yaml
normalization:
  cel_input_defaults:
    applies_to:
    - operators[]
    - operator_edges[]
```

runtime の default output もこの順序で固定しています。

### `contract_summary.environment_warnings` を always present にした

`contract_summary.environment_warnings` は required にし、警告なしの場合も `[]` を
出すようにしました。snapshot consumer が「警告なし」と「古い artifact で field が
ない」を区別しやすくするためです。

## 文書化したこと

- `PLAN_CONTRACTS.md`
  - `raw_plan` / `raw_nodes` と serialized `queries[].plan` の違い。
  - raw QueryPlan CEL が report 単体では replay できないこと。
  - topology / matched index / summary invariants。
  - `replayable_from_report` を含む minimal outcome snippets。
- `README.md`
  - backend identity source と `not_recorded` caveat。
  - raw CEL が `replayable_from_report: false` になること。
- `DESIGN.md`
  - CEL variable rename と replayability。
  - `backend_identity.source` の位置付け。
- `IMPLEMENTATION_STATUS.md`
  - 今回閉じた drift と、version / digest がまだ `not_recorded` である残課題。

## 維持した判断

### backend version / image digest の自動記録はまだ完全実装していない

`backend_identity.source` は追加しましたが、Omni image digest / tool versions を
完全に自動記録するところまでは今回入れていません。

理由は、現状の runtime surface から安定して取得できる identity と、研究メモで手動記録
している container digest / tool versions の境界をまだ分け切れていないためです。
ただし artifact には `source` と `not_recorded` を明示し、CI 利用時に
observation docs / container pinning と併用すべき状態であることは見えるようにしました。

### raw QueryPlan CEL は残す

raw QueryPlan CEL を v1alpha から削除する選択肢もありましたが、plan-shape 調査や
normalizer 追加前の検証には有用なので残しました。

ただし `raw_plan` / `raw_nodes` という名前、`replayable_from_report: false`、
`check_recommended: false`、`raw_query_plan_contract_used_in_check` warning により、
CI gate の主経路ではないことを artifact 上で明確化しました。

## 検証

```sh
GOCACHE=/private/tmp/spanner-analyzer-go-build go run ./cmd/spanner-query-gen plan-report-schema --out schemas/spanner-query-gen.plan-report.v1alpha.schema.json
GOCACHE=/private/tmp/spanner-analyzer-go-build go run ./cmd/spanner-query-gen plan-contract-schema --out schemas/spanner-query-gen.plan-contracts.v1alpha.schema.json
go test ./internal/plancontract ./cmd/spanner-query-gen
```
