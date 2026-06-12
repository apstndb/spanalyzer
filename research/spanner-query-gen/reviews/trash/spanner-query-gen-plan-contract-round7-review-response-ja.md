# spanner-query-gen plan contract round 7 review response

作成日: 2026-05-06

## 概要

round 7 の指摘は大きな設計変更ではなく、v1alpha の public contract を CI artifact として締めるための指摘として受け取りました。今回の変更では、documentation の誤解防止、schema の status-specific validation、canonical target / identifier reference の lexical validation を中心に反映しました。

## 反映した内容

- README と `PLAN_CONTRACTS.md` に Spanner Omni execution-plan support の Preview / Pre-GA caveat を明記しました。
  - `plan-report` は review / testing / prototyping 用の artifact であり、production performance guarantee ではないという位置づけを維持しています。
- `PLAN_CONTRACT_CANDIDATES.md` に残っていた旧 `query:` grammar 例を canonical `name` + `target` grammar に直しました。
  - `target: query/Foo`
  - `target: query/Foo#inner`
- plan-contract schema の `target` を v1alpha の canonical target ID に限定しました。
  - pattern は `^query/[A-Za-z_][A-Za-z0-9_]*(#inner)?$` です。
  - `readPlanContracts` 側でも同じ pattern を検証するようにしました。
- main config schema の名前参照 field を `$defs.identifier` に寄せました。
  - `queries[].catalog`
  - `queries[].binding`
  - `writes[].catalog`
  - `catalogs[].bindings.external_query_connections[].spanner_catalog`
  - `catalogs[].bindings.spanner_external_datasets[].spanner_catalog`
  - YAML parse / normalize 側にも同じ lexical validation を追加しました。
- plan-report schema の `contract_evaluation` を `status` ごとに締めました。
  - `pass` / `fail` は `results` を要求し、`reason` / `error` を禁止します。
  - `not_evaluated` は `reason` を要求し、`results` を禁止します。
  - `reason: target_error` は `error` を要求します。
  - `reason: target_not_found` では `error` を出さない shape にしました。
- plan-report schema の `contract_rule_result` も `status` / `failure_kind` ごとに締めました。
  - `status: pass` では `failure_kind`, `diagnostic_id`, `remediation` を禁止します。
  - `status: fail` では `failure_kind` を要求します。
  - `failure_kind: classification_unknown` は `diagnostic_id` を要求します。
  - `failure_kind: violation` では `diagnostic_id` を禁止する方針にしました。
- `contract_summary` の count fields は zero value でも出力するようにしました。
  - schema 上は required なので、JSON/YAML artifact 側でも省略しない方が downstream tooling が扱いやすいためです。
- `--require-optimizer-pinning` の意味を README / `PLAN_CONTRACTS.md` に明記しました。
  - v1alpha では `plan-report` flags で requested optimizer options が指定されているかを見ます。
  - SQL statement hint から optimizer pinning を推論しません。
  - `optimizer.effective` は `not_recorded` です。
- checked-in schema files を再生成しました。

## まだ反映していない内容

- observed plan shape の golden fixture 化はまだです。
  - round 7 の指摘通り、Push Broadcast / Distributed Cross Apply / Aggregate classification は壊れやすいので早めに fixture 化する価値があります。
  - ただし今回は schema contract の締めを先に完了しました。
- `skipped` query の `error` を `reason` / `message` に分離する変更はまだ入れていません。
  - `contract_evaluation` 側は `not_evaluated.reason` を持つ shape にしました。
  - per-query skipped shape は次に report schema を大きく触る時に合わせるのがよさそうです。
- SQL statement hint を読んで `optimizer.statement_hint` や effective optimizer を report に出す処理は未実装です。
  - v1alpha では documentation で明示し、実装は増やしていません。

## 補足

`failure_kind: violation` の `diagnostic_id` は optional ではなく禁止にしました。通常の violation は `rule`, `operator_family`, `observed_count`, `max_count`, `remediation` で十分に機械判定できるためです。`diagnostic_id` は classification failure や analysis warning と紐づくケースに限定した方が downstream tooling 側の分岐が単純になります。

## 検証

- `gofmt -w cmd/spanner-query-gen/config_schema.go cmd/spanner-query-gen/plan_report.go cmd/spanner-query-gen/main_test.go querygen_config.go querygen_test.go`
- `go run ./cmd/spanner-query-gen config-schema --out schemas/spanner-query-gen.v1alpha.schema.json`
- `go run ./cmd/spanner-query-gen plan-report-schema --out schemas/spanner-query-gen.plan-report.v1alpha.schema.json`
- `go run ./cmd/spanner-query-gen plan-contract-schema --out schemas/spanner-query-gen.plan-contracts.v1alpha.schema.json`
- `go test ./cmd/spanner-query-gen -run 'TestRunConfigSchemaIdentifierReferences|TestRunPlanContractSchemaTargetPattern|TestPlanReportSchemaContractConditionals|TestReadPlanContractsRejectsInvalidTargetID|TestPlanReportContractTargetIDRejectsMissingTarget'`
- `go test . -run 'TestParseQueryCodegenConfigYAMLV1AlphaRejectsNonIdentifierReferences'`
- `go test ./...`
