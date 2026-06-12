# `spanner-query-gen` plan contract round 24 review

作成日: 2026-05-07

## 前提

今回も、実装コード・実行ログ・CI output はこちらには共有されていない前提で見ています。したがって、レビュー対象は最新 response、README / DESIGN、plan-contract / plan-report schema、PLAN_CONTRACTS / PLAN_CONTRACT_CANDIDATES などから読み取れる **public contract と文書上の整合性** です。

## 総評

今回の更新はかなり良いです。round 23 で指摘した「schema が外部レビュー対象として共有されていない」問題は解消され、最新の `spanner-query-gen.plan-report.v1alpha.schema.json` が添付されています。response でも、root required に `warnings` を追加したこと、`contract_stability` の tier-dependent conditional、`normalized_operators[].spool_name`、`SpoolScan` fixture、backend identity の manual override が明示されています。

特に良い点は以下です。

- top-level `warnings` と `contract_summary.environment_warnings` の役割分担を明文化したこと。
- `contract_stability.reasons` を non-empty にし、`tier: raw_query_plan` / `normalized_operator` と `replayable_from_report` の関係を schema 側でも締めたこと。
- `spool_name` を metadata-derived normalized field として扱い、CEL が読む場合は stability reason に出す方針にしたこと。
- backend version / image digest を自動取得できない場合に、まず manual override として artifact に残す設計にしたこと。
- `PLAN_CONTRACTS.md` の no-target 例を `warnings: []` always-present 方針と同期したこと。

ここまで来ると、大きな仕様変更よりも、artifact の provenance と future candidate の誤用防止を詰める段階だと思います。

## 追加フィードバック

### P1. `backend_identity.source: manual` の意味をもう少し固定した方がよい

`--backend-version` / `--backend-image-digest` の manual override は実用的です。`spanemuboost` が Omni version / image digest を stable API として公開していないなら、まず手動指定を許すのは妥当です。

ただし、`backend_identity.source` が単一 field だと、次のようなケースの意味が少し曖昧になります。

```yaml
backend_identity:
  kind: omni
  source: manual
  version: not_recorded
  image_digest: not_recorded
```

この形は避けたいです。schema または runtime self-check で、少なくとも次の invariant を固定するとよいです。

```text
if backend_identity.source == manual:
  version != not_recorded OR image_digest != not_recorded

if backend_identity.source == not_recorded:
  version == not_recorded AND image_digest == not_recorded
```

`source: spanemuboost` については、現状では `version` / `image_digest` が `not_recorded` のままでも許す判断でよいと思います。これは “spanemuboost 経由で backend identity を得ようとしたが、version / digest は取得不能” という意味に読めるからです。

さらに厳密にするなら、将来は field ごとに provenance を分ける案もあります。

```yaml
backend_identity:
  kind: omni
  version: 2026.r1-beta
  version_source: manual
  image_digest: not_recorded
  image_digest_source: not_recorded
```

ただし v1alpha では重いので、まずは single `source` + invariant で十分です。

### P1. `contract_evaluations` は contract evaluation mode では `minItems: 1` にしてよい

最新 plan-contract schema では `contracts` が `minItems: 1` です。`plan-report` 側でも、`contract_evaluation_mode: report_only | check` なら、contract file が存在し、少なくとも 1 つの contract があるはずです。

そのため、plan-report schema でも次を入れてよいと思います。

```text
if contract_evaluation_mode in [report_only, check]:
  contract_evaluations.minItems = 1
```

もちろん、`contract_summary.contracts == len(contract_evaluations)` は schema ではなく self-check でよいです。ただ、空配列だけは schema で弾けるので、public artifact contract として少し強くできます。

### P1. future spool consistency check は `spool_name != ""` guard を明示した方がよい

`spool_name` を normalized operator field として公開した判断は良いです。`PLAN_CONTRACT_CANDIDATES.md` に future spool consistency check を置くのも妥当です。

ただし、CEL input では optional string metadata の missing default が `""` なので、将来の spool consistency example では必ず空文字を除外する必要があります。

たとえば、候補としては次のような形が安全です。

```cel
operators
  .filter(o, o.family == "spool_scan")
  .all(scan,
    scan.spool_name != "" &&
    operators.exists(build,
      build.family == "spool_build" &&
      build.spool_name == scan.spool_name))
```

`spool_scan` があるのに `spool_name == ""` の場合は、silent pass ではなく fail させるべきです。これは `subquery_cluster_node` や `call_type` と同じく、metadata-derived normalized field を使う contract の典型例になるので、候補段階から guard を明記しておく価値があります。

### P1. manual backend identity は “evidence” ではなく “assertion” と呼ぶ方が安全

manual override は便利ですが、レビュー artifact としては「観測された事実」ではありません。ユーザーが CLI flag で与えた self-reported identity です。

したがって docs では、次のように書き分けると誤読を防げます。

```text
backend_identity.source: spanemuboost
  backend identity was obtained from the backend/probe layer where available.

backend_identity.source: manual
  backend identity was supplied by the caller and recorded as an assertion.

backend_identity.source: not_recorded
  backend identity was not recorded.
```

この区別は、以前 external dataset の access verification で “observed fact” と “attestation” を分けた方がよいと議論した点と同じです。

### P2. `spool_name` stability reason fixture は field-specific に固定した方がよい

response の例では、`call_type`, `spool_name`, `subquery_cluster_node` が stability reason に並んでいます。この方向は良いです。

追加で、`spool_name` だけを読む CEL fixture を 1 つ用意すると、field-specific stability detection の regression として強くなります。

```cel
operators.exists(o, o.family == "spool_scan" && o.spool_name != "")
```

期待する stability reason は、余計な field を含まず、たとえば次のようになるのが理想です。

```yaml
reasons:
- contract uses the normalized plan-report view
- "contract reads metadata-derived normalized fields: spool_name"
```

AST-based stability detection を入れているので、こういう最小 fixture はかなり価値があります。

### P2. `contract_stability` の tier coupling は良い。`check_recommended` も将来は policy 化できる

最新 schema では、`tier: raw_query_plan` なら `check_recommended: false` / `replayable_from_report: false`、`tier: normalized_operator` なら `replayable_from_report: true` という条件が入っています。これはかなり良いです。

今は `normalized_operator` の `check_recommended` を常に `true` に固定していないように見えますが、これは許容できます。将来、normalized view でも特定の metadata-derived field を読む contract を低 confidence にしたくなる可能性があるからです。

したがって、ここは現状維持でよいと思います。

### P2. `PLAN_CONTRACT_CANDIDATES.md` の raw plan 説明に canonical variable names をもう少し足す

今回、旧 `plan` / `nodes` の raw CEL variable 名はかなり整理されています。ただ、候補集ではまだ “raw `spannerpb.QueryPlan` and `spannerpb.PlanNode` objects” のような説明が中心です。

ここに一文だけ足すと、読者が `queries[].plan` の rendered string と混同しにくくなります。

```text
Raw CEL variables are named `raw_plan` and `raw_nodes`; they are evaluator inputs and are not replayable from the serialized YAML/JSON report.
```

`PLAN_CONTRACTS.md` ではすでに説明されていますが、候補集だけ読んだ人向けに同じ線引きを置く価値があります。

### P2. schema digest / generated schema path を response に残すと外部レビューが楽になる

今回の response では、`TestPlanReportSchemaFileIsCurrent` により生成元と保存済み schema の一致を固定していると説明されています。これは良いです。

さらに、今後の response では次のような短い情報を入れると、外部レビューで “この schema が最新か” を追いやすくなります。

```text
schema files updated:
- schemas/spanner-query-gen.plan-report.v1alpha.schema.json sha256: ...
- schemas/spanner-query-gen.plan-contracts.v1alpha.schema.json sha256: ...
```

これは必須ではありませんが、レビュー archive が長くなってきたので、最新 artifact の同一性を確認しやすくなります。

## これ以上やらなくてよいこと

今の段階では、以下は急がなくてよいと思います。

- 新しい predefined contract の追加。
- `spool_consistency` の predefined 化。
- `backend_identity` を複雑な per-field provenance object にすること。
- raw QueryPlan contract の完全な external replay support。
- PROFILE / execution stats contract。

特に plan contract はかなり opinionated な extension なので、今は “optional Omni-backed structural PLAN review artifact” として閉じ込める方針を維持するのが良いです。

## そのまま送れる短い返答案

> 今回の更新はかなり良いです。round 23 で気にしていた schema 共有の問題は解消され、root `warnings`、`contract_stability` の tier-dependent conditional、`normalized_operators[].spool_name` が外部レビュー可能になっています。
>
> 追加で見るなら、まず `backend_identity.source: manual` の invariant を固定した方がよいです。`source: manual` なら `version` または `image_digest` の少なくとも片方は `not_recorded` ではない、`source: not_recorded` なら両方 `not_recorded`、という self-check か schema 条件を入れると artifact として読みやすくなります。また manual は観測事実ではなく caller assertion なので、その wording も docs に入れると安全です。
>
> 次に、`contract_evaluation_mode: report_only | check` では `contract_evaluations` を `minItems: 1` にしてよいと思います。contract file schema は `contracts.minItems: 1` なので、空の `contract_evaluations` は artifact として不自然です。
>
> `spool_name` については、future spool consistency candidate に `spool_name != ""` guard を明記してください。CEL input では optional string metadata の missing default が `""` なので、`spool_scan` があるのに `spool_name` が空のケースを silent pass させない例にした方がよいです。
>
> それ以外は大きな方向転換不要です。次は predefined contract を増やすより、manual backend identity の provenance、contract evaluation minItems、spool candidate の guard、schema digest の記録あたりを締めるのが良いと思います。

## 結論

設計はかなり収束しています。今回の更新で、前回までの主な懸念だった “schema 変更が本当に共有されているか” は解消されています。

残る論点は、機能拡張ではなく artifact provenance と candidate docs の誤用防止です。特に `backend_identity.source: manual` の意味、contract evaluation mode と空 evaluation の扱い、`spool_name` を使う CEL candidate の missing metadata guard を締めれば、v1alpha の plan-report / plan-contract surface はかなりレビューしやすい状態になります。
