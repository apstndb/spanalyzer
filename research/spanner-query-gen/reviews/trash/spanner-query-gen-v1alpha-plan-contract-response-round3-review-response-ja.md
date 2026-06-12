# `spanner-query-gen` v1alpha plan contract round 3 response

レビューありがとうございます。今回の指摘は、`plan-report --contracts --check`
を CI gate として使ったときに、どの contract が安定な plan-shape contract
なのかを report だけで判断できるようにするべき、という内容だと理解しました。
実装と公開ドキュメントの両方に反映しました。

## 反映した点

- report に version fields を追加しました。
  - `report_version: v1alpha-plan-report-v1`
  - `contract_file_version: v1alpha-plan-contracts`
  - `contract_evaluator_version: v1`
- report に canonical target ID を追加しました。
  - 通常 query: `query/ScanSingerIDsFast`
  - `external_query` inner SQL: `query/ExternalQuerySingerIDs#inner`
  - contract result にも resolved `target_id` を出すようにしました。
- contract file 側にも `target` を追加しました。
  - `target: query/ScanSingerIDsFast`
  - `target: query/ExternalQuerySingerIDs#inner`
  - `query` + `scope` は simple target 向け alias として残しています。
- `--require-optimizer-pinning` を追加しました。
  - `optimizer_not_pinned`
  - `statistics_package_not_pinned`
  - `query_optimizer_not_pinned`
  - これらを warning ではなく command error に昇格できます。
- `--optimizer-version` / `--optimizer-statistics-package` を追加しました。
  - Spanner query options として `AnalyzeQuery` に渡します。
  - これにより `--require-optimizer-pinning` を SQL hint なしでも満たせます。
- contract evaluation result に stability tier を追加しました。
  - normalized `operators` / `operator_families` だけを使う contract は
    `stability.tier: normalized_operator`
  - raw `plan` / `nodes` を参照する CEL contract は
    `stability.tier: raw_query_plan` かつ `check_recommended: false`
- `PROFILE` / `execution_stats` は v1alpha plan contract の対象外にしました。
  - `plan-report --render-mode=profile` は拒否します。
  - CEL expression が `execution_stats` を参照した場合も拒否します。
  - contract surface は structural PLAN output のみに固定します。
- aggregate classification の unknown handling を追加しました。
  - `Aggregate` node に `iterator_type: Hash` / `Stream` がない場合は
    `classification_warnings` に `aggregate_iterator_type_unknown` を出します。
  - `no_hash_aggregate` / `no_stream_aggregate` のように分類結果に依存する
    predefined contract は、unknown aggregate を黙って pass せず fail に寄せます。
- `contract_summary.environment_warnings` を拡張しました。
  - `backend_identity_not_recorded`
  - `operator_classification_unknown`
- remediation に `auto_fix: false` を明示するようにしました。
  - hint recommendation は violation-level の remediation に留め、SQL/config は自動変更しません。
- README の plan contract section は最小例中心に寄せました。
  - CEL / raw QueryPlan / aggregate classification などの詳細は DESIGN 側に寄せています。
- `plan-report-schema` を追加し、最新の plan-report output schema を
  `schemas/spanner-query-gen.plan-report.v1alpha.schema.json` に保存しました。
- `plan-contract-schema` を追加し、最新の plan contract file schema を
  `schemas/spanner-query-gen.plan-contracts.v1alpha.schema.json` に保存しました。
- `spannerplan` を `v0.1.8` に更新し、`cel-go` を `v0.28.0` に更新しました。
  - `go mod why -m github.com/apstndb/lox` は
    `main module does not need module github.com/apstndb/lox` になっています。

## 一部だけ採用した点

### contract status の三値化

`pass | fail | unknown` はまだ導入していません。

ただし、MVP としては「unknown classification に依存する predefined contract は
`--check` で fail に寄せる」挙動を入れました。これにより、
`no_hash_aggregate` が metadata 欠落により誤って pass するケースは避けています。

将来的に user-defined CEL も含めて unknown を表現したくなった場合は、
status 三値化を検討します。

### optimizer pinning policy

`optimizer_not_pinned` / `statistics_package_not_pinned` は引き続き
environment warning として出します。加えて、CLI flag として
`--optimizer-version`, `--optimizer-statistics-package`,
`--require-optimizer-pinning` を追加しました。

contract file 側の nested policy まではまだ入れていません。v1alpha の surface を
増やしすぎないため、まず CLI options, CLI gate, report warning に留めています。

### contract file の `target` field

contract file 側にも `target` を追加しました。

将来 `outer_sql` / writes / DML まで plan-report 対象が広がるなら、
`target: query/Foo#inner` を優先する形が安全だと考えています。
既存の `query` + `scope` は alias として残しています。

### backend identity の実値取得

`backend_identity.version` / `image_digest` はまだ `not_recorded` です。
ただし warning として `backend_identity_not_recorded` を出すようにしました。
将来 CI gate として強く使うなら、ここは取得できる実値に置き換えるべきです。

## 検証

- `go test -run 'TestRunPlanReportSchema|TestPlanReportSchemaFileIsCurrent|TestRunPlanReportRejectsProfileMode|TestWritePlanReportOutputs|TestPlanReportContracts|TestPlanReportCELContract|TestPlanReportCELRejectsExecutionStats|TestReadPlanContractsRejectsUnknownFields|TestPlanReportOperatorFamilyUsesDisplayName|TestPlanReportOperatorsIncludeStableMetadata|TestBuildPlanReportNoTargets' -count=1 ./cmd/spanner-query-gen`
- `go test -run 'TestRunHelp|TestRunPlanContractSchema|TestPlanContractSchemaFileIsCurrent|TestRunPlanReportHelp|TestPlanReportContractTargetID|TestPlanReportContractTargetIDRejectsMismatchedQuery|TestPlanReportOptimizerPinningWarnings' -count=1 ./cmd/spanner-query-gen`
- `go test -count=1 ./...`
- `mise x -- golangci-lint run --timeout=5m`
- `go test -count=1 -tags=integration -run '^TestIntegration' ./cmd/spanner-query-gen`
- `SPANEMUBOOST_ENABLE_OMNI_TESTS=1 go test -count=1 -v -tags='integration omni' -run '^TestIntegration.*Omni' ./cmd/spanner-query-gen`

## 現時点の結論

`plan-report` は optional Omni-backed review artifact として維持し、
main v1alpha config には plan contract を入れない方針を継続します。

その上で、今回の修正により以下の境界は明確になりました。

- stable CI contract の推奨入力は normalized operator view。
- raw QueryPlan / PlanNode CEL は可能だが `raw_query_plan` tier として扱う。
- PROFILE / execution stats は v1alpha plan contract の対象外。
- aggregate classification unknown は report に出し、依存する predefined contract は fail。

この状態なら、experimental でありながらも `--contracts --check` を使う人が
安定性を誤解しにくい形になったと思います。
