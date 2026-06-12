# spanner-query-gen plan contract round 20 response

レビューありがとうございます。round 20 は、plan-report artifact の意味をさらに
machine-checkable にする指摘として扱いました。大きな方向転換はせず、P1 と
議論の余地が少ない P2 を実装・文書化しました。

## 反映したこと

### CEL failure kind を `violation` に固定

plan-report schema を更新し、`rule: cel` かつ `status: fail` の rule result では
`failure_kind: violation` のみを許容するようにしました。

また、CEL fail では `diagnostic_id` を禁止しました。これにより、
`classification_unknown` は predefined / direct `forbid_operator_family` 専用の
failure kind として読めるようになります。

runtime 側も既に CEL failure は `failure_kind: violation` だけを出す実装なので、
regression test で固定しました。

### `source: use/<name>` と `predefined: <name>` の runtime invariant を固定

schema で suffix equality まで表現することは避け、runtime invariant のままとしました。
ただし、既存の regression test を確認し、全 predefined contract について次を固定しています。

- result `source` は `use/<predefined>`
- result `predefined` は `<predefined>`
- direct `forbid[n]` と `cel` では `predefined` を出さない

これは round 20 の方針通り、schema を過度に複雑にせず artifact generator 側で
不整合を出さないことを固定する扱いです。

### summary count invariant を self-check

`contract_summary` と `contract_evaluations` の count invariant、および
`target_summary.included_count == planned + errors + skipped` は JSON Schema では
自然に表現しにくいため、`writePlanReport` の出力直前に self-check するようにしました。

不整合があれば YAML/JSON/Markdown artifact を書かず、generator bug として失敗します。
この挙動も regression test で固定しています。

### CEL stability detection を AST-based にした

追加で、CEL の stability 判定を単純な substring match から CEL AST の identifier /
select field ベースに変更しました。

これにより、文字列リテラルに `"nodes"`、`"execution_stats"`、`"scan_target"` などが
出るだけでは raw QueryPlan contract や PROFILE stats 参照、metadata-derived field
参照として扱われません。実際に変数や field として参照した場合だけ反映します。

## 維持した判断

### `cel_input_defaults` は compact form のまま

`optional_string_fields` / `optional_boolean_fields` を machine-readable に出す案は、
今回は採用していません。

外部 evaluator support を強く打ち出す段階では有用ですが、v1alpha では
`applies_to` exact set と docs の field list で十分だと判断しました。

### root-partitionable は candidate のまま

`ApproximateRootPartitionablePlanShape` は引き続き candidate / CEL example です。
predefined には昇格していません。

これは PLAN-shape approximation であり、`PartitionQuery` API acceptance probe では
ないためです。将来 predefined にする場合も
`approximate_root_partitionable_plan_shape` のような名前にする方針です。

### 新しい predefined contract は増やさない

`no_full_scan`、`require_seekable_scan`、`no_residual_condition`、`no_back_join` などは
候補として維持しますが、v1alpha の今は artifact semantics と fixture を固めることを
優先します。

特に `no_back_join` は schema-aware な index ownership / base table 判定が必要なので、
現時点では predefined にしません。

## 検証

```sh
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go run ./cmd/spanner-query-gen plan-report-schema --out schemas/spanner-query-gen.plan-report.v1alpha.schema.json
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go test ./cmd/spanner-query-gen ./internal/plancontract
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go test ./...
git diff --check
```
