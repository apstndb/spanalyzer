# spanner-query-gen plan contract round 9 review response

作成日: 2026-05-06

## 概要

round 9 の指摘は、plan contract の新機能追加よりも、schema / docs / normalizer / fixture の同期を締める内容として受け取りました。大きな方向転換は不要という判断に同意し、今回は v1alpha の public contract として drift しやすい箇所を優先して直しました。

## 反映した内容

- `Distributed Semi Apply` と `Distributed Anti Semi Apply` を別 family に分けました。
  - `distributed_semi_apply`
  - `distributed_anti_semi_apply`
  - `distributed_semi_apply_internal_apply`
  - `distributed_anti_semi_apply_internal_apply`
- `no_apply_join` は broad policy として、distributed semi / anti-semi wrapper の両方を禁止します。
  - `distributed_cross_apply`
  - `distributed_semi_apply`
  - `distributed_anti_semi_apply`
- `no_standalone_apply_join` は standalone apply-family のみを禁止します。
  - `apply_join`
  - `semi_apply`
  - `anti_semi_apply`
  - distributed wrapper 内部の implementation apply は standalone として数えません。
- plan-contract schema と plan-report schema の `operator_family` enum を再生成しました。
  - 両 schema の enum が完全一致する unit test を追加しました。
  - enum は plan-report normalizer registry から生成される前提を `PLAN_CONTRACTS.md` / `DESIGN.md` に明記しました。
- `QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md` の Normalization Impact と説明を更新しました。
  - `distributed_anti_semi_apply`
  - `distributed_anti_semi_apply_internal_apply`
  - 観測メモに列挙された normalized family が known family に存在することを unit test で固定しました。
- `PLAN_CONTRACTS.md` を semi/anti apply 版の semantics に同期しました。
  - `no_apply_join`
  - `no_standalone_apply_join`
  - distributed wrapper internal family
- `contract_rule_result.source` を schema 上 required にしました。
  - predefined expansion では `source: use/<name>` と `predefined: <name>` が必須です。
  - direct `forbid[n]` / `cel` result では `predefined` を禁止します。
  - CEL result には `source: cel` を出すようにしました。
- Push Broadcast / Semi Apply / Distributed Apply 系の file-based plan fixtures を追加しました。
  - standalone Hash Join
  - Push Broadcast Hash Join with internal Hash Join
  - branched Push Broadcast map path with real Hash Join
  - standalone Semi Apply / Anti-Semi Apply
  - Distributed Semi Apply with internal Semi Apply
  - Distributed Anti Semi Apply with internal Semi Apply
- fixture で contract semantics も固定しました。
  - `no_hash_join` は Push Broadcast wrapper を拒否します。
  - `no_standalone_hash_join` は Push Broadcast internal Hash Join を許可します。
  - branched map path にある本物の Hash Join は standalone `hash_join` のままです。
  - `no_apply_join` は standalone semi/anti と distributed semi/anti wrapper を拒否します。
  - `no_standalone_apply_join` は distributed wrapper 内部 apply を許可します。
- `tools/spanner-query-plan-shape/README.md` に Spanner Omni execution-plan support の Preview / Pre-GA caveat を追加しました。
- `PLAN_CONTRACTS.md` に report output の最小例を追加しました。
  - pass
  - report_only fail
  - target_not_found

## 判断

`distributed_anti_semi_apply` を `distributed_semi_apply` に fold する案ではなく、レビューで推奨された distinct family 方式を採用しました。理由は、operator family は report artifact の vocabulary であり、観測された operator 名に近い方が direct `forbid` や CEL を書く利用者に分かりやすいためです。

predefined contract は引き続き user-facing policy として broad に保っています。つまり、family は細かく分け、`no_apply_join` などの predefined contract が必要な family をまとめて禁止します。

## 検証

- `gofmt -w cmd/spanner-query-gen/plan_report.go cmd/spanner-query-gen/config_schema.go cmd/spanner-query-gen/main_test.go`
- `go run ./cmd/spanner-query-gen plan-report-schema --out schemas/spanner-query-gen.plan-report.v1alpha.schema.json`
- `go run ./cmd/spanner-query-gen plan-contract-schema --out schemas/spanner-query-gen.plan-contracts.v1alpha.schema.json`
- `go test ./cmd/spanner-query-gen`
- `go test -count=1 ./cmd/spanner-query-gen ./tools/spanner-query-plan-shape`
- `go test -count=1 ./...`

## 補足

今回の修正では新しい predefined contract は増やしていません。round 9 の結論どおり、今は contract surface を広げるよりも、既存 surface の schema 同期、normalizer vocabulary、fixture、artifact semantics を固める段階だと考えています。
