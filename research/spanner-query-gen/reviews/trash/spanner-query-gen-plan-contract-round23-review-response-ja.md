# spanner-query-gen plan contract round 23 response

レビューありがとうございます。round 23 は、round 22 で入れた artifact surface の変更が
外部レビュー可能な形で同期されているかを確認しつつ、残っていた docs/test の締め込みを
行いました。

## 反映したこと

### 最新 plan-report schema をレビュー対象として残す

round 22 response で述べた schema 変更は、保存済み schema に反映済みです。

- root required に `warnings` を追加。
- `contract_stability` に tier-dependent conditional を追加。
- `normalized_operators[].spool_name` を追加。

生成元と保存済み schema の一致は `TestPlanReportSchemaFileIsCurrent` で固定しています。
次回以降の外部レビューでは、更新後の
`schemas/spanner-query-gen.plan-report.v1alpha.schema.json` も共有対象に含めます。

### no-target 例を `warnings` always-present と同期

`PLAN_CONTRACTS.md` の no-target 例に `warnings: []` を追加しました。
また、他の required top-level fields を省略している snippet なので、見出しは
“minimal no-target report” ではなく “abridged no-target excerpt” に変更しました。

### top-level `warnings` と `environment_warnings` の役割を明記

`PLAN_CONTRACTS.md` に次の役割分担を追加しました。

- top-level `warnings`: artifact-wide production / completeness warnings
- `contract_summary.environment_warnings`: selected backend / optimizer /
  contract mode の下で contract interpretation に影響する warning

これで、snapshot consumer が top-level warning と contract-specific environment warning を
混同しにくくなります。

### `spool_name` を metadata-derived stability detection に追加

`operators[].spool_name` は raw PlanNode metadata を normalized operator view にコピーした
field なので、`call_type` や `subquery_cluster_node` と同じく metadata-derived normalized
field として扱います。

CEL が `spool_name` を読む場合、stability reason に次のような文言が出ます。

```yaml
stability:
  tier: normalized_operator
  reasons:
  - contract uses the normalized plan-report view
  - "contract reads metadata-derived normalized fields: call_type, spool_name, subquery_cluster_node"
  check_recommended: true
  replayable_from_report: true
```

### `SpoolScan` fixture 名を明確化

`display_name: SpoolScan` + `metadata.spool_name` の fixture は、意図が分かるように
`TestPlanReportSpoolScanDisplayNameWithSpoolNameMetadata` にしました。

### future spool consistency check を候補に追加

`PLAN_CONTRACT_CANDIDATES.md` に、`spool_build` / `spool_scan` /
`spool_name` を使った future spool consistency check のメモを追加しました。
これはまだ predefined contract にはせず、normalized CEL contract の候補として残します。

### backend identity の manual override を追加

`spanemuboost v0.4.0` は backend platform は公開していますが、Omni version / image digest
を stable API として公開していません。そのため、自動取得ではなくまず
`plan-report --backend-version` と `--backend-image-digest` を追加しました。

未指定の場合は従来通り `source: spanemuboost` かつ `version/image_digest:
not_recorded` です。明示指定された場合は `backend_identity.source: manual` として
artifact に記録します。

## レビュー archive の整理

最新の active exchange を round 23 に更新しました。round 22 以前の役割を終えた review /
response は `research/spanner-query-gen/reviews/trash/` に移動済みです。

## 検証

```sh
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go run ./cmd/spanner-query-gen plan-report-schema --out schemas/spanner-query-gen.plan-report.v1alpha.schema.json
GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go test ./internal/plancontract ./cmd/spanner-query-gen -run 'TestStabilityForUsesCELIdentifiersNotStringLiterals|TestRunPlanReportSchema|TestPlanReportWarningsAreSerializedWhenEmpty|TestPlanReportSchemaContractConditionals|TestPlanReportSchemaSeparatesConcreteAndDerivedOperatorFamilies|TestPlanReportSpoolScanDisplayNameWithSpoolNameMetadata|TestPlanReportSchemaFileIsCurrent|TestPlanReportBackendIdentityFromFlags|TestRunPlanReportHelp' -count=1
git diff --check
SPANNER_ANALYZER_GOOGLESQL_CACHE_DIR=/private/tmp/spanner-analyzer-googlesql-cache GOCACHE=/private/tmp/spanner-analyzer-go-build-cache go test ./...
```
