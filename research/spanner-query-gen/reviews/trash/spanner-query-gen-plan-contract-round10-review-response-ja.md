# spanner-query-gen plan contract round 10 review response

作成日: 2026-05-06

## 概要

round 10 の指摘は妥当だと判断しました。特に、plan-report artifact をレビュー可能な contract として扱うなら、`source` / `predefined` / status invariant / schema examples は曖昧なままにしない方がよいです。今回は新しい contract surface を広げず、既存 surface の意味と artifact schema を締める対応を優先しました。

## 反映した内容

- `PLAN_CONTRACTS.md` の minimal report outcomes を現行の `contract_evaluation` shape に同期しました。
  - `pass` / `fail` 例に `query`, `scope`, `stability`, `results` を含めました。
  - `not_evaluated` 例にも `stability` を含めました。
  - これらは full top-level `plan-report` ではなく contract-related excerpt であることを明記しました。
- `PLAN_CONTRACTS.md` の minimal report outcome snippets が古くならないように unit test を追加しました。
  - YAML として parse できること。
  - status ごとの必須 shape を満たすこと。
  - `use/<name>` result が `predefined` を持つこと。
- `contract_rule_result.source` の schema を厳密化しました。
  - `use/<predefined-name>`
  - `forbid[n]`
  - `cel`
  - 以外を受け付けない pattern にしました。
- `contract_rule_result.predefined` を free-form string ではなく predefined contract enum にしました。
- `forbid.max_count` の JSON Schema に `default: 0` を追加しました。
- predefined contract 名の一覧を helper に切り出し、schema と test が同じ名前集合を参照するようにしました。
- `source: use/<name>` と `predefined: <name>` が実装上対応していることを unit test で固定しました。
- direct `forbid[n]` と `cel` の result では `predefined` が空になることを unit test で固定しました。
- status invariant を unit test と docs に追加しました。
  - evaluation `pass` は全 rule result が `pass`。
  - evaluation `fail` は少なくとも一つの rule result が `fail`。
  - `not_evaluated` は result を持たない。
  - summary は全 evaluation pass の時だけ `pass`。
  - fail / not_evaluated が一つでもあれば summary は `fail`。
- `no_apply_join` の説明を明確化しました。
  - `Push Broadcast Hash Join Semi Apply` / `Anti Semi Apply` は `push_broadcast_hash_join` family なので、`no_apply_join` ではなく `no_hash_join` / `no_push_broadcast_hash_join` で検査します。
- `PLAN_CONTRACT_CANDIDATES.md` の predefined contract 説明を「one or more normalized operator families」に修正しました。
- `unknown` family の扱いを明文化しました。
  - classifier が既知 family に分類できない PlanNode は `unknown` に落とします。
  - `operator_family: unknown` を direct `forbid` で検査できます。
  - predefined contract は無関係な `unknown` の存在だけでは fail にしません。
- `unknown` family の direct forbid と predefined contract の境界を unit test で固定しました。
- normalizer の fallback を arbitrary snake-case family ではなく `unknown` に変更しました。
  - schema enum との整合性を優先しました。
  - unknown family に対する `classification_warnings` も追加しました。
- `QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md` の位置付けを明確化しました。
  - 観測 vocabulary の evidence であり、normative contract は fixture / generated schema / unit test である、と明記しました。

## 判断

`source` と `predefined` の一致は JSON Schema だけで完全に表すより、実装テストで固定する方が現在の実装には合っています。schema では許可される値域を狭め、実装テストで `use/<name>` と `<name>` の一致を確認する分担にしました。

`unknown` については、schema enum に `unknown` を持つ以上、未知 operator 名をそのまま snake-case family として出すより、`unknown` に正規化する方が report schema と artifact contract の整合性が高いと判断しました。

## 検証

- `GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go run ./cmd/spanner-query-gen plan-contract-schema --out schemas/spanner-query-gen.plan-contracts.v1alpha.schema.json`
- `GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go run ./cmd/spanner-query-gen plan-report-schema --out schemas/spanner-query-gen.plan-report.v1alpha.schema.json`
- `gofmt -w cmd/spanner-query-gen/config_schema.go cmd/spanner-query-gen/main_test.go cmd/spanner-query-gen/plan_report.go`
- `GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go test ./cmd/spanner-query-gen`
- `GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go test ./...`
- `git diff --check`

## 補足

今回は CEL macro や新しい predefined contract の追加には踏み込みませんでした。round 10 の焦点は artifact の契約性と docs/schema/test の同期であり、そこを先に固める方が v1alpha のレビュー容易性に効くと判断しています。
