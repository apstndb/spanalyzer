# spanner-query-gen plan contract round 21 review

対象: round 20 response、`PLAN_CONTRACTS.md`、`IMPLEMENTATION_STATUS.md`、最新 plan-report / plan-contract / main config schema。

前提: 実装コード・実行ログ・CI output は共有されていないため、このレビューは public contract / schema / docs / 補助ドキュメントの整合性レビューに限定する。

## 結論

大きな方向転換は不要。今回の更新はかなり良い。

特に、次の整理は v1alpha の CI artifact としての意味を強くしている。

- `rule: cel` の failure kind を `violation` に固定し、`classification_unknown` を predefined / direct `forbid_operator_family` 専用にしたこと。
- `source: use/<name>` と `predefined: <name>` の equality を runtime invariant として固定し、schema は値 domain と rule/source kind coupling に集中したこと。
- `contract_summary` / `target_summary` の count invariant を artifact 書き出し前 self-check に寄せたこと。
- CEL stability detection を substring matching ではなく AST identifier / select field ベースにしたこと。
- `cel_input_defaults` を compact に保ちつつ、v1alpha では docs + exact `applies_to` で説明する判断。

全体として、今は predefined contract を増やす段階ではなく、**report artifact を外部レビュー・CI で誤読しないための最後の締め込み**を続ける段階だと思う。

## 追加フィードバック

### 1. raw CEL variables の名前は `raw_plan` / `raw_nodes` にした方がよい

現在の `PLAN_CONTRACTS.md` では CEL variables として raw `plan` / raw `nodes` が説明されている。一方、plan-report schema の `queries[].plan` は human-readable rendered plan string であり、raw `spannerpb.QueryPlan` ではない。

これは、内部 evaluator では問題なくても、外部 reader / external evaluator には混乱しやすい。

v1alpha で破壊的変更がまだ許されるなら、CEL variable は次のようにした方が安全。

```text
plan  -> raw_plan
nodes -> raw_nodes
```

または、raw QueryPlan CEL を v1alpha ではさらに明確に experimental として、docs に次を入れる。

```text
CEL variables raw_plan/raw_nodes are evaluator inputs, not serialized report fields.
They are not replayable from the YAML/JSON report unless the evaluator is supplied
with the original raw QueryPlan input.
```

もし既存名 `plan` / `nodes` を維持するなら、最低限 `PLAN_CONTRACTS.md` の CEL Inputs に、`queries[].plan` との違いを明記した方がよい。

### 2. raw QueryPlan CEL の replayability を report に出した方がよい

normalized CEL input は serialized report からかなり再現しやすくなっている。一方、raw `spannerpb.QueryPlan` / raw PlanNode を読む CEL は、report YAML/JSON だけでは再評価できない可能性が高い。

`stability.tier: raw_query_plan` と `check_recommended: false` は良いが、CI artifact としてはもう一段だけ明示すると親切。

例:

```yaml
stability:
  tier: raw_query_plan
  check_recommended: false
  replayable_from_report: false
  reasons:
  - contract references raw QueryPlan input
```

`replayable_from_report` という field を増やさないなら、`stability.reasons` に固定文言として入れるだけでもよい。

### 3. `contract_stability.reasons` は required + `minItems: 1` にしてよい

plan-report schema では `contract_stability` が `tier` と `check_recommended` を required にしている一方、`reasons` は optional に見える。

しかし docs / examples では `reasons` が artifact の説明力を担っている。特に次の区別は、理由がないと downstream consumer が判断しにくい。

- normalized operator view だけを使う contract
- metadata-derived normalized fields を読む contract
- raw QueryPlan / PlanNode を読む contract
- check に推奨される contract / 推奨されない contract

したがって、v1alpha では `stability.reasons` を required にし、`minItems: 1` にしてよいと思う。

```json
"contract_stability": {
  "required": ["tier", "check_recommended", "reasons"],
  "properties": {
    "reasons": { "type": "array", "minItems": 1, ... }
  }
}
```

これは artifact size をほとんど増やさず、外部レビュー性を上げる。

### 4. topology self-check も summary self-check と同じ扱いにしたい

今回、summary count invariant を `writePlanReport` の出力直前 self-check にした判断は良い。同じ考え方で、plan topology も self-check した方がよい。

理由は、plan contract / CEL / `matched_operator_indexes` がすでに以下に強く依存しているため。

- `normalized_operators[].index`
- `operator_edges[].parent_index` / `child_index`
- `operators[].child_indexes`
- `operators[].descendant_indexes`
- `operators[].subtree_family_counts`
- `contract_rule_result.matched_operator_indexes`

推奨 invariant:

```text
for each query with status ok:
  normalized_operators[].index is unique
  every operator_edges[].parent_index and child_index exists in normalized_operators
  every child_indexes entry exists in normalized_operators
  every descendant_indexes entry exists in normalized_operators
  matched_operator_indexes in contract results refer to normalized_operators[].index
  subtree_family_counts is complete and matches the operator's normalized subtree
  operator_family_counts equals subtree count of the root or equivalent whole-tree count
```

JSON Schema では表現しにくいので、summary count と同じく runtime self-check + regression test が自然。

### 5. backend identity が `not_recorded` のままだと CI gate としては弱い

`IMPLEMENTATION_STATUS.md` では、plan report の backend identity は現状 `not_recorded` で、Omni image digest や tool versions は observation docs に手動記録する状態とされている。

これは v1alpha の documentation / prototype としては許容できる。ただし、plan contracts を CI gate として使い始めるなら、環境再現性の一番大きな穴になる。

優先順位としては、新しい predefined contract よりも先に、少なくとも以下を plan-report artifact に自動記録する方がよい。

```yaml
backend_identity:
  kind: omni
  version: ... or not_recorded
  image_digest: sha256:... or not_recorded
  source: spanemuboost | manual | not_recorded
```

今すぐ実装しないなら、`PLAN_CONTRACTS.md` の CI usage section に「backend identity が `not_recorded` の場合は observation docs / container pinning と併用する」と明記するとよい。

### 6. `cel_input_defaults.applies_to` は exact set だけでなく canonical order も固定したい

`applies_to` は schema 上、`operators[]` と `operator_edges[]` の exact set になっているように見える。これは良い。

ただし array なので、artifact snapshot としては order も固定した方が読みやすい。

```yaml
normalization:
  cel_input_defaults:
    applies_to:
    - operators[]
    - operator_edges[]
```

JSON Schema でも `prefixItems` で固定できる。schema を複雑にしたくないなら、runtime output order invariant として docs/test に固定するだけでも十分。

## 小さな確認点

### `contract_summary.environment_warnings`

`environment_warnings` は optional のように見える。CI artifact として安定させるなら、`contract_summary` に常に出して、警告なしは `[]` にしてもよい。これは必須ではないが、snapshot consumer には優しい。

### `contract_file_version` と schema digest

`contract_file_version` / `contract_evaluator_version` は良い。将来、external evaluator を強く打ち出す場合は、plan-contract schema digest / plan-report schema digest も report に出すと、artifact と validator の対応がさらに明確になる。ただし v1alpha では優先度は低い。

## まとめ

今回の更新で、round 20 の主要懸念はかなり閉じている。

次に詰めるなら、新しい plan contract を増やすより、次の順がよい。

1. raw CEL variable 名または replayability の明確化
2. `contract_stability.reasons` の required 化
3. plan topology self-check
4. backend identity の自動記録または CI docs caveat
5. `cel_input_defaults.applies_to` の canonical order 固定

特に 1 と 2 は、public artifact の読み間違いを減らす効果が大きい。現行の方向性自体は良く、v1alpha の plan contract は「experimental だが CI artifact として読める」段階にかなり近づいている。
