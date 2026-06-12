# spanner-query-gen plan contract round 22 response

レビューありがとうございます。round 22 は、設計の大きな変更ではなく、round 21 で
固めた plan-report / plan-contract artifact surface の最終同期と schema invariant
強化として扱いました。

## 反映したこと

### `PLAN_CONTRACT_CANDIDATES.md` の raw CEL input 名を同期

指摘通り、候補集側に旧表記の raw `plan` / raw `nodes` が残っていました。
`raw_plan` / `raw_nodes` に更新し、これらは CEL evaluator input であり serialized
report field ではないことを明記しました。

serialized report の `queries[].plan` は rendered human-readable plan string であり、
raw `spannerpb.QueryPlan` ではない、という整理を候補ドキュメント側にも合わせています。

### `contract_stability` の schema coupling を追加

plan-report schema で次の invariant を条件付きに固定しました。

- `tier: raw_query_plan`
  - `check_recommended: false`
  - `replayable_from_report: false`
- `tier: normalized_operator`
  - `replayable_from_report: true`

`normalized_operator.check_recommended` は現時点の runtime では true ですが、将来
metadata-derived normalized fields の扱いをさらに細分化する余地を残すため、schema
では固定していません。

### top-level `warnings` を always-present にした

`contract_summary.environment_warnings` と同様に、top-level `warnings` も report schema
の required field にしました。runtime report も警告なしの場合に `warnings: []` を
出すようにしています。

これにより、snapshot consumer が「警告なし」と「古い/incomplete artifact で field が
ない」を区別しやすくなります。

### `SpoolScan` raw representation を normalizer fixture にした

`display_name: SpoolScan` + `metadata.spool_name` の raw representation を fixture
として追加しました。これは `Scan.scan_type=SpoolScan` ではない representation として
固定し、normalized family は `spool_scan` になります。

あわせて、normalized operator view に `spool_name` を追加しました。`SpoolBuild` /
`SpoolScan` の対応を YAML/JSON report と CEL input の両方で確認しやすくするためです。

## 文書化したこと

- `PLAN_CONTRACT_CANDIDATES.md`
  - `raw_plan` / `raw_nodes` への同期。
  - raw CEL inputs と serialized report fields の違い。
- `PLAN_CONTRACTS.md`
  - top-level `warnings` が always-present であること。
  - `operators[].spool_name` が optional string metadata であること。
- `spanner-query-gen.plan-report.v1alpha.schema.json`
  - root required に `warnings` を追加。
  - `contract_stability` の tier-dependent coupling を追加。
  - `normalized_operators[].spool_name` を追加。

## 今回は defer したこと

### backend version / image digest capture

レビューの通り、CI gate としての再現性を上げる次の優先候補は backend identity の
自動記録または明示指定です。ただし現在の artifact は `source: spanemuboost` と
`version/image_digest: not_recorded`、および warning を出しており、v1alpha としては
honest な状態です。

今回は新しい contract を増やすより、schema invariant と normalizer fixture を優先しました。

### raw QueryPlan の report 埋め込み

`raw_plan` / `raw_nodes` contract を serialized report 単体で replay できるようにするための
raw QueryPlan 埋め込みは引き続き入れていません。artifact size、privacy、raw protobuf
表現の互換性を増やすため、v1alpha では normalized operator view を CI の主経路にします。

## 検証

```sh
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go run ./cmd/spanner-query-gen plan-report-schema --out schemas/spanner-query-gen.plan-report.v1alpha.schema.json
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go test ./cmd/spanner-query-gen -run 'TestRunPlanReportSchema|TestPlanReportWarningsAreSerializedWhenEmpty|TestPlanReportSchemaContractConditionals|TestPlanReportSchemaSeparatesConcreteAndDerivedOperatorFamilies|TestPlanReportSpoolScanRawRepresentationFixture|TestPlanReportSchemaFileIsCurrent' -count=1
```
