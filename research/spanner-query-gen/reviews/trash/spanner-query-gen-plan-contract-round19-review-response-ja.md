# spanner-query-gen plan contract round 19 response

レビューありがとうございます。round 19 は、新しい predefined contract を増やすよりも
plan-report artifact の public contract を締める指摘として扱いました。
大きな設計変更はせず、schema / runtime validation / docs / fixture を更新しました。

## 反映したこと

### `contract_rule_result.rule` と `source` の対応を schema で制約

plan-report schema の conditional を締めました。

- `rule: cel` は `source: cel` を要求し、`expression` を必須にする
- `rule: cel` では `predefined`, `operator_family`, `observed_count`,
  `max_count`, `matched_operator_indexes` を禁止する
- `rule: forbid_operator_family` は `source: use/<predefined>` または
  `source: forbid[n]` を要求する
- `rule: forbid_operator_family` では `operator_family`, `observed_count`,
  `max_count`, `matched_operator_indexes` を必須にし、`expression` を禁止する

`source: use/<name>` と `predefined: <name>` の完全一致は引き続き runtime invariant
として扱います。schema では少なくとも `rule` と source kind の矛盾を弾くように
しました。

### `normalization.cel_input_defaults.applies_to` を exact set に固定

`applies_to` は `operators[]` / `operator_edges[]` の enum array ですが、schema 上で
`minItems: 2`, `maxItems: 2`, `uniqueItems: true` にして、現在の v1alpha では両方を
必ず含む exact set として扱うようにしました。

field-level の machine-readable default list は今回は入れていません。docs には対象
field を列挙済みなので、v1alpha では `cel_input_defaults` の compact な形を維持します。

### metadata-derived CEL の stability reason を field-specific に変更

CEL expression が `subquery_cluster_node` や `call_type` など、raw PlanNode metadata
由来の normalized field を読む場合、report の `stability.reasons` に具体的な field 名を
出すようにしました。
判定は CEL AST の identifier / select field を見る形にし、単純な substring match は
避けました。これにより、文字列リテラルに `"nodes"` や `"execution_stats"` が出るだけで
raw plan contract や PROFILE stats 参照として扱われる誤判定を避けます。

root-partitionable candidate の fixture では次のようになります。

```yaml
stability:
  tier: normalized_operator
  reasons:
  - contract uses the normalized plan-report view
  - "contract reads metadata-derived normalized fields: call_type, subquery_cluster_node"
  check_recommended: true
```

`stability.tier` は `normalized_operator` のままです。ユーザーが raw `plan` / `nodes`
を直接参照しているわけではなく、normalizer が公開した normalized field を使っているためです。

### duplicate `contracts[].name` の diagnostic ID を固定

duplicate contract name の runtime validation は
`plan_contract.duplicate_contract_name` を返すようにしました。

これは direct `forbid` の重複 diagnostic
`plan_contract.duplicate_forbid_operator_family` と同じ名前空間に揃えたものです。
invalid fixture 相当の regression test も追加しています。

### `ALLOW_DISTRIBUTED_MERGE` 依存の注意と example を追加

`PLAN_CONTRACTS.md` に、sort 系 contract が statement hint の
`ALLOW_DISTRIBUTED_MERGE` に依存し得ることを短い SQL example 付きで追記しました。

`--require-optimizer-pinning` は requested optimizer version / statistics package だけを
確認し、statement-level plan-shape hints を optimizer pinning とは扱わない、という整理を
維持しています。hint に依存する contract では rendered SQL、`sql_sha256`、
optimizer matrix evidence を併せて確認する想定です。

## 維持した判断

### root-partitionable は candidate のまま

`ApproximateRootPartitionablePlanShape` は引き続き candidate / CEL example として扱い、
predefined contract には昇格していません。

これは PLAN-shape approximation であり、`PartitionQuery` API acceptance probe では
ないためです。将来 predefined にする場合も
`approximate_root_partitionable_plan_shape` のような保守的な名前にする方針を維持します。

### `stability.features` は追加しない

現時点では `stability.reasons` で十分にレビュー可能です。machine-readable feature list は、
外部 evaluator がそれを実際に使う段階で検討します。

### serialized report に optional field を全部明示出力しない

serialized YAML/JSON の冗長化は避け、CEL replay semantics は
`normalization.cel_input_defaults` と docs で明示する方針を維持しました。

## 検証

```sh
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go run ./cmd/spanner-query-gen plan-report-schema --out schemas/spanner-query-gen.plan-report.v1alpha.schema.json
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go test ./cmd/spanner-query-gen ./internal/plancontract
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go test ./...
git diff --check
```
