# `spanner-query-gen` plan contract round 14 response

作成日: 2026-05-06

## 返答概要

round 14 の指摘は、v1alpha の public artifact を CI で誤読されにくくするための tightening として妥当だと判断しました。新しい predefined contract は増やさず、silent pass 回避、target-level status の可視化、schema/docs の明確化を優先しました。

## 反映した内容

### direct aggregate forbid の classification unknown をテストで固定

実装上は direct `forbid.operator_family: hash_aggregate` / `stream_aggregate` でも `classification_unknown` fail になる経路が既にありました。ただし回帰防止が弱かったため、direct `forbid` で unknown `Aggregate` が silent pass しないことをテストで固定しました。

また、rule result の `diagnostic_id` はレビュー案に合わせて `plan.aggregate_classification_unknown` に変更しました。`matched_operator_indexes` は曖昧な `Aggregate` node index を返すこともテストしています。

### duplicate direct forbid を runtime validation で reject

同一 contract 内で同じ `forbid.operator_family` を複数回指定するケースは、semantic duplicate として reject するようにしました。

```text
plan_contract.duplicate_forbid_operator_family
```

これは schema の `uniqueItems` では表現しきれないため runtime validation の責務としています。

### target-level status counts を `target_summary` に追加

top-level `status` は artifact production status であり、query target が全て成功したという意味ではないため、`target_summary` に次の counts を追加しました。

- `planned`
- `errors`
- `skipped`

`included_count` は Spanner query target として plan acquisition 対象になった数を表し、`planned/errors/skipped` は `queries[].status` の集計として読めるようにしました。

### plan-report schema を tightening

plan-report schema に以下を反映しました。

- `target_summary.planned/errors/skipped` を required に追加。
- `query.name`、`contract_evaluation.name`、`contract_evaluation.query` を identifier pattern に制約。
- top-level `status` の description に、target-level success ではないことを明記。
- `queries[].status` の description に、`error/skipped` では input/error fields だけが信頼できることを明記。

### docs を更新

`PLAN_CONTRACTS.md` / `DESIGN.md` / `README.md` に以下を追記しました。

- direct aggregate-family rules でも classification unknown は fail する。
- `diagnostic_id: plan.aggregate_classification_unknown` と ambiguous Aggregate の `matched_operator_indexes` を返す。
- top-level status と target-level status は別物。
- `queries[].status: error | skipped` の plan normalization fields は successful plan evidence として扱わない。
- duplicate direct forbid は merge せず reject する。

## 追加で反映した function hint 観測

レビューとは別の直前指摘に対応して、`spanner-query-plan-shape` に `--case function_hint` を追加しました。Spanner docs の `DISABLE_INLINE` function hint を Spanner Omni で確認し、次を観測しました。

- default / `DISABLE_INLINE=FALSE`: `SHA512($SingerInfo)` が outer `SUBSTR` ごとに inline 展開される。
- `DISABLE_INLINE=TRUE`: `Compute` operator が追加され、`SHA512($SingerInfo)` を一度 `$x` として materialize し、outer expression が `$x` を参照する。

この差分は compact shape でも `Compute` として見えますが、scalar operator tree は `--output nodes` / `--output yaml` / `--output json` で確認しやすいです。`yaml` / `json` output は raw query plan protobuf を query label / SQL と一緒に出すようにしました。

## 反映しなかった内容

### 新しい predefined contract は追加しない

レビューの「今は増やさなくてよいもの」には同意します。今回は `no_distributed_semi_apply`、positive `require_stream_aggregate`、`no_back_join`、seekability rules、PROFILE / execution stats、remediation auto-fix は増やしていません。

### error/skipped の plan fields は schema で完全禁止しない

今回は Option B に寄せました。`status: error | skipped` でも SQL rendering や digest などの partial fields はレビュー上有用なため、schema description と docs で「input/error fields だけが信頼できる」と明記しました。

今後 stale plan fields が実際に混入するリスクが高いと分かった場合は、`planReportQuery` の serialization を status-aware にして Option A へ寄せる余地があります。

## 内部検証

開発側では次を確認しました。

```sh
go test ./cmd/spanner-query-gen ./internal/plancontract ./tools/spanner-query-plan-shape
go run ./cmd/spanner-query-gen plan-report-schema --out schemas/spanner-query-gen.plan-report.v1alpha.schema.json
go run ./tools/spanner-query-plan-shape --case function_hint --output compact
go run ./tools/spanner-query-plan-shape --case function_hint --output yaml
```
