# spanner-query-gen plan contract round 11 review response

作成日: 2026-05-06

## 概要

round 11 の指摘は妥当だと判断しました。今回の焦点は新しい predefined contract を増やすことではなく、plan-report artifact を読んだときの追跡性、CEL の安定性、schema と runtime validation の役割分担を締めることだと受け取りました。

## 反映した内容

- `operator_family_counts` を zero-filled complete map にしました。
  - 全 `operator_family` enum value を必ず含めます。
  - plan に出ていない family は missing key ではなく `0` です。
  - `operator_families` は引き続き observed family の compact list です。
  - CEL activation でも sparse な内部値を complete map に正規化して渡します。
- plan-report schema でも `queries[].operator_family_counts` を complete map として表現しました。
  - 全 known family を `required` にし、各 property を non-negative integer にしています。
- `contract_rule_result.matched_operator_indexes` を追加しました。
  - `normalized_operators[].index` に対応します。
  - `max_count` を超えた分だけではなく、対象 family に一致した全 operator index を出します。
  - aggregate classification unknown の場合は、分類不能な `Aggregate` node の index を出します。
- Markdown report にも contract result の `matched_operator_indexes` を表示するようにしました。
- `contracts[].name` の runtime uniqueness validation は既存で入っていましたが、診断しやすいようにエラーへ `plan-contract.validation.duplicate-contract-name` を含め、unit test で固定しました。
- plan-report schema の target / scope / backend をさらに正規化しました。
  - `queries[].target_id`
  - `contract_evaluations[].target_id`
  - `excluded_target.id`
  - `contract_evaluations[].scope`
  - `contract_evaluations[].backend`
- `PLAN_CONTRACTS.md` に `source` / `predefined` invariant を明記しました。
  - `source: use/<name>` なら `predefined: <name>`。
  - `source: forbid[n]` または `source: cel` なら `predefined` は absent。
- `PLAN_CONTRACTS.md` に strict CI 用の `operator_family: unknown` direct forbid 例を追加しました。
- `PLAN_CONTRACTS.md` の `max_count` 説明を、schema default ではなく tool behavior と report output contract として明確化しました。
- `PLAN_CONTRACT_CANDIDATES.md` の CEL examples 周辺に、`operator_family_counts` が zero-filled であることを明記しました。

## 判断

`operator_family_counts` は zero-filled に寄せました。artifact size は増えますが、CEL で `operator_family_counts["hash_join"] == 0` のような条件を安全に書ける利点の方が大きいと判断しました。

`matched_operator_indexes` は optional field として追加しました。matching node がある場合に出し、`normalized_operators[].index` へ直接辿れる形にしています。pass/fail の status invariant とは独立した説明用 metadata です。

`contracts[].name` の重複はすでに runtime validation で reject していましたが、レビューで指摘された通り JSON Schema の `uniqueItems` だけでは不十分なので、diagnostic ID 付きの test coverage を追加しました。

## 検証

- `GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go run ./cmd/spanner-query-gen plan-report-schema --out schemas/spanner-query-gen.plan-report.v1alpha.schema.json`
- `gofmt -w cmd/spanner-query-gen/config_schema.go cmd/spanner-query-gen/main_test.go cmd/spanner-query-gen/plan_report.go`
- `GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go test ./cmd/spanner-query-gen`
- `GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go test ./...`
- `git diff --check`

## 補足

今回も CEL macro、新しい predefined contract、PROFILE / execution stats contract は入れていません。v1alpha の plan contract はまだ experimental なので、今は surface を広げるより、既存 artifact を CI で読みやすく、schema と docs でレビューしやすい形にする方を優先しています。
