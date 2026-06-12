# spanner-query-gen plan contract round 8 review response

作成日: 2026-05-06

## 概要

round 8 の指摘は、`plan-report` / plan contract の大きな設計変更ではなく、schema、report artifact、補助ドキュメントの同期をさらに強くする内容として受け取りました。今回の対応では、公開 contract の drift を減らしつつ、追加で観測した `IN` / `EXISTS` 系の plan shape も operator family normalization に反映しました。

## 反映した内容

- `PLAN_CONTRACT_CANDIDATES.md` の “Current v1alpha Contract Surface” を非規範的な “Current Surface Summary” に変更しました。
  - 完全な operator family enum は plan-contract / plan-report schema を source of truth とし、候補集から手書きの `Known families` 全列挙を削除しました。
- `no_explicit_sort` の説明を normalized family matching に寄せました。
  - `display_name` の substring match ではないことを明記しました。
  - ordered-result operator である `distributed_merge_union` は `explicit_sort` と別 family のままです。
- `contract_rule_result` に由来情報を追加しました。
  - `source: use/no_hash_join`
  - `predefined: no_hash_join`
  - direct `forbid` では `source: forbid[0]` のように出します。
- `contract_evaluation` の schema を締めました。
  - `status: pass | fail` では `query`, `scope`, `results` を required にしました。
  - target が解決されて評価できた場合は resolved target metadata を常に出す前提です。
- `PLAN_CONTRACTS.md` に `contract_summary.status` と process exit の関係を明文化しました。
  - `contract_summary.status` は contract truth です。
  - `report_only` は summary が `fail` でも exit code 0 を維持します。
  - `check` は summary が `fail` なら non-zero exit します。
- `QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md` に観測環境を追加しました。
  - Spanner Omni image tag / digest
  - `spanemuboost` / `spannerplan` module version
  - Go version
  - optimizer version / statistics package は `not_pinned`
  - `/private/tmp/...` の具体パスは `<tmp>/...` に一般化しました。
- `tools/spanner-query-plan-shape` の matrix 生成を `text/template` ベースにしました。
  - `buildJoinMatrixQueries` と `buildSubqueryJoinHintMatrixQueries` は同じ matrix helper を使います。
  - 今後、hint や predicate の直積を増やす時に同じ方式を再利用できます。
- `IN` / `EXISTS` / `NOT IN` / `NOT EXISTS` に statement-level join hints を付けた matrix を追加し、Spanner Omni で観測しました。
  - 全 48 ケースが成功しました。
  - `Semi Apply`, `Anti-Semi Apply`, `Distributed Semi Apply`, `Distributed Anti Semi Apply`, `Push Broadcast Hash Join Semi Apply`, `Push Broadcast Hash Join Anti Semi Apply` を確認しました。
- 上記の観測に合わせて operator family を追加・整理しました。
  - `semi_apply`
  - `distributed_semi_apply`
  - `distributed_semi_apply_internal_apply`
  - `push_broadcast_hash_join` は semi / anti-semi variant も wrapper として扱います。
- `no_apply_join` / `no_standalone_apply_join` は apply-family として `semi_apply` / `anti_semi_apply` も対象にしました。
  - distributed wrapper 内部の implementation node は `distributed_semi_apply_internal_apply` として分けます。
- checked-in schema files を再生成しました。

## まだ反映していない内容

- file-based golden fixture 化はまだ完了していません。
  - 今回は synthetic `spannerpb.QueryPlan` unit test と Spanner Omni probe で coverage を増やしました。
  - 次にやるなら、review 指摘の通り Push Broadcast / Distributed Apply / Aggregate / Sort 系を fixture として固定するのがよいです。
- SQL statement hint を読んで `optimizer.statement_hint` や effective optimizer を report に出す処理は未実装です。
  - v1alpha では引き続き documentation で明示し、`optimizer.effective` は `not_recorded` のままにしています。

## 補足

`IN` / `EXISTS` 系は、実行計画上は通常の明示 JOIN とは異なる `Semi Apply` / `Anti-Semi Apply` family として現れます。ただし statement-level `JOIN_METHOD` hint によって `Hash Join`, `Merge Join`, `Push Broadcast Hash Join ... Apply` に変わることを確認したため、plan contract の apply/hash/push-broadcast 系 predicate から漏れないようにしました。

`no_apply_join` を `apply_join` だけに限定すると、`EXISTS` の `Semi Apply` を見逃します。これは利用者の直感とずれるため、v1alpha の破壊的変更が許されるうちに apply-family として広げる判断をしました。

## 検証

- `gofmt -w cmd/spanner-query-gen/plan_report.go cmd/spanner-query-gen/config_schema.go cmd/spanner-query-gen/main_test.go tools/spanner-query-plan-shape/docs_cases.go tools/spanner-query-plan-shape/main.go tools/spanner-query-plan-shape/query_matrix.go tools/spanner-query-plan-shape/query_matrix_test.go`
- `go run ./cmd/spanner-query-gen plan-report-schema --out schemas/spanner-query-gen.plan-report.v1alpha.schema.json`
- `go run ./cmd/spanner-query-gen plan-contract-schema --out schemas/spanner-query-gen.plan-contracts.v1alpha.schema.json`
- `go test ./tools/spanner-query-plan-shape`
- `go test ./cmd/spanner-query-gen -run 'TestRunPlanReportSchema|TestRunPlanContractSchema|TestPlanReportJoinContracts|TestPlanReportOperatorsClassifySubqueryJoinHintOperators|TestPlanReportObservedDocsOperatorFamiliesAreKnown'`
- `go run ./tools/spanner-query-plan-shape --case subquery_join_hint_matrix --output compact --continue-on-error --timeout 10m`
- `go run ./tools/spanner-query-plan-shape --case join_matrix --output compact --continue-on-error --timeout 10m`
- `go test ./...`
