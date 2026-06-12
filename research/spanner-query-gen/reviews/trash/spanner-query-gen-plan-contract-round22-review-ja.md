# spanner-query-gen plan contract round 22 再レビュー

対象: 最新の `spanner-query-gen-plan-contract-round21-review-response-ja.md`、`README(37).md`、`DESIGN(37).md`、`PLAN_CONTRACTS(16).md`、`spanner-query-gen.plan-report.v1alpha.schema(20).json`、`spanner-query-gen.plan-contracts.v1alpha.schema(12).json`、`IMPLEMENTATION_STATUS(5).md`、および補助的に `spanner-hacks-operators-feedback-ja.md`。

前提: 実装コード、実行ログ、CI output は共有されていないため、ここでは public contract / schema / docs / 補助ドキュメントの整合性だけを見る。

## 結論

今回の更新はかなり良い。round 21 で指摘した主要な誤読ポイント、特に raw CEL input と serialized report の混同は、`plan` / `nodes` から `raw_plan` / `raw_nodes` への rename と `replayable_from_report` の追加によってかなり明確になった。

また、`contract_stability.reasons` の required + non-empty 化、`contract_summary.environment_warnings` の always-present 化、topology self-check、`matched_operator_indexes` の self-check、`backend_identity.source` の追加も、CI artifact としての説明力を上げている。大きな方向転換は不要。

今回さらに見るべき点は、ほぼ「最後の同期」と「schema で締められる invariant」の範囲に収まっている。

## 良くなった点

### 1. raw CEL input の命名が明確になった

`raw_plan` / `raw_nodes` への rename は正しい。serialized report の `queries[].plan` は human-readable rendered plan string であり、CEL evaluator の raw `spannerpb.QueryPlan` と同じ `plan` という名前にすると誤読されやすかった。旧 alias を残さない判断も v1alpha なら妥当。

`PLAN_CONTRACTS.md` でも、`raw_plan` は in-memory `spannerpb.QueryPlan`、`raw_nodes` は `raw_plan.plan_nodes` であり、serialized report fields ではないと説明されている。さらに `replayable_from_report: false` を出すことで、YAML/JSON report 単体で再評価できないことが artifact 上で読めるようになっている。

### 2. stability が review artifact として強くなった

`contract_stability` が `tier` / `reasons` / `check_recommended` / `replayable_from_report` を required にし、`reasons` を non-empty にしたのは良い。これにより、normalized contract、metadata-derived normalized contract、raw QueryPlan contract の差が、report consumer に常に見える。

### 3. topology / matched index self-check の追加はかなり重要

`normalized_operators[].index`、`operator_edges`、`child_indexes`、`descendant_indexes`、`subtree_family_counts`、`operator_family_counts`、`matched_operator_indexes` を出力直前に self-check する方針は妥当。これは schema だけでは守りづらい構造的 invariant なので、runtime self-check + regression test で固定するのが正しい。

### 4. backend identity の状態が前より読める

`backend_identity.source` の追加により、少なくとも identity 情報の取得経路が report に残るようになった。一方で `version` / `image_digest` はまだ `not_recorded` のまま、かつ warning も維持しているため、「完全な環境再現性がある」と誤解されにくい。この線引きは良い。

### 5. CEL defaults の machine-readable 化が進んだ

`normalization.cel_input_defaults.applies_to` を exact order で固定した点は良い。`operators[]` と `operator_edges[]` の optional string / boolean default を外部 evaluator が再現するための足場として十分に役立つ。

## 追加フィードバック

### P1. `PLAN_CONTRACT_CANDIDATES.md` に旧 CEL variable 名が残っている

最新の `PLAN_CONTRACTS.md` は `raw_plan` / `raw_nodes` に更新されているが、`PLAN_CONTRACT_CANDIDATES(12).md` の CEL Contracts section にはまだ次の旧表記が残っている。

```text
- raw `plan`
- raw `nodes`
```

これは補助ドキュメントとはいえ、plan contract surface の候補集なので、ここも `raw_plan` / `raw_nodes` に同期した方がよい。

修正案:

```text
- `raw_plan`
- `raw_nodes`
```

あわせて、`PLAN_CONTRACT_CANDIDATES.md` にも「raw CEL inputs are evaluator inputs, not serialized report fields」と短く入れると、`PLAN_CONTRACTS.md` との整合性が上がる。

### P1. `contract_stability` の tier と replayability / recommendation を schema でも coupling できる

現在の plan-report schema では `contract_stability` に `tier`、`check_recommended`、`replayable_from_report` が required になっており、これは良い。ただし schema 上は、たとえば次のような意味的に矛盾した artifact をまだ許し得るように見える。

```yaml
stability:
  tier: raw_query_plan
  check_recommended: true
  replayable_from_report: true
```

runtime が生成しないなら実害は小さいが、public artifact schema としては条件付きで締められる。

推奨 invariant:

```text
if tier == raw_query_plan:
  check_recommended == false
  replayable_from_report == false

if tier == normalized_operator:
  replayable_from_report == true
```

`normalized_operator` の `check_recommended` は現状 true で良さそうだが、将来 metadata-derived normalized fields をより慎重に扱いたいなら schema では固定せず runtime invariant に留めてもよい。少なくとも raw tier の `false / false` は schema で固定すると artifact consumer に親切。

### P1. top-level `warnings` も always-present にすると snapshot consumer に優しい

`contract_summary.environment_warnings` を required + `[]` always-present にしたのは良い。同じ考え方を top-level `warnings` にも適用してよいと思う。

現在の schema では top-level `warnings` は optional に見える。これは問題ではないが、snapshot consumer にとっては「警告なし」と「古い artifact / incomplete artifact で field がない」の区別がつきにくい。

推奨:

```text
warnings: [] を always present にする
```

contract-specific warning は `contract_summary.environment_warnings`、artifact/general warning は top-level `warnings`、という分離を維持したまま、どちらも always-present にすると読みやすい。

### P1. `backend_identity.source` は良いが、CI 用途では次に version / digest capture が重要

`source: spanemuboost` を出すようになったのは前進。ただし `IMPLEMENTATION_STATUS.md` にもある通り、Omni version と image digest はまだ `not_recorded` のままなので、plan contract を CI gate として使う場合の再現性はまだ observation docs / container pinning 側に依存している。

次に優先するなら、新しい contract を増やすより、backend identity の自動記録または明示指定を優先したい。

候補:

```sh
spanner-query-gen plan-report \
  --backend-version 2026.r1-beta \
  --backend-image-digest sha256:...
```

または、`spanemuboost` runtime から安定取得できるなら自動記録の方が良い。

ただし v1alpha では、現在の `source + not_recorded + warning` でも十分 honest なので、これは次段階でよい。

### P2. `cel_input_defaults` は compact なままで良いが、将来 replay 需要が強ければ field list を machine-readable にする

現状は docs に optional string / boolean metadata field listが書かれており、schema には `optional_string: ""`、`optional_boolean: false`、`applies_to` がある。このバランスは v1alpha として妥当。

ただし、外部 evaluator が本当に serialized report だけから CEL を再評価するユースケースが増えるなら、次のような machine-readable field list を検討してもよい。

```yaml
normalization:
  cel_input_defaults:
    optional_string_fields:
      operators[]:
      - execution_method
      - iterator_type
      - scan_type
      - scan_target
      operator_edges[]:
      - type
      - variable
    optional_boolean_fields:
      operators[]:
      - full_scan
```

今すぐ入れる必要はない。v1alpha では現在の compact metadata の方が simple。

### P2. `spanner-hacks-operators-feedback` の `SpoolScan` mismatch は normalizer fixture に落とす価値がある

今回添付された `spanner-hacks-operators-feedback-ja.md` は、upstream `operators.md` との比較として、通常の repeated-CTE `SpoolScan` が `Scan.scan_type=SpoolScan` ではなく raw `PlanNode.display_name: SpoolScan` + `metadata.spool_name` として観測された、と指摘している。

plan-report 側ではすでに `spool_scan` family が存在するため、大きな仕様変更は不要。ただし normalizer の fixture として次を固定すると良い。

```text
raw display_name == "SpoolScan" -> family: spool_scan
metadata.spool_name があれば report metadata に残すか、少なくとも classification evidence に使う
```

もし別 plan family で `Scan.scan_type=SpoolScan` も存在し得るなら、それも許容して同じ `spool_scan` family に寄せると堅い。

## 今回はあえて提案しないこと

### raw QueryPlan を report に埋め込む機能

`raw_plan` / `raw_nodes` CEL が `replayable_from_report: false` になる問題を解決するために、`--include-raw-plan` のような option を考えることはできる。ただし、これは artifact size、stability、privacy、raw protobuf representation の互換性問題を増やす。

v1alpha では、raw QueryPlan CEL は exploratory / advanced として残し、normalized operator view を CI の主経路にする今の判断が良い。

### 新しい predefined contract の追加

`no_full_scan`、`require_seekable_scan`、`no_residual_condition`、`approximate_root_partitionable_plan_shape` などは魅力的だが、今はまだ候補集に置く方がよい。現在の plan-report contract は artifact semantics がかなり締まってきたので、次に増やすなら fixture と docs evidence が揃ったものからにするのが安全。

## そのまま返せる短いコメント案

> 反映内容はかなり良いです。特に `raw_plan` / `raw_nodes` への rename、`replayable_from_report`、`contract_stability.reasons` の required + non-empty 化、topology self-check、`backend_identity.source`、`environment_warnings` always-present は、CI artifact としての誤読をかなり減らしています。大きな方向転換は不要です。
>
> 追加で見るべき一番大きな点は、`PLAN_CONTRACT_CANDIDATES.md` にまだ旧 CEL variable 名の raw `plan` / raw `nodes` が残っていることです。補助ドキュメントですが、候補集も contract surface の一部として読まれやすいので、`raw_plan` / `raw_nodes` に同期してください。
>
> 次に、`contract_stability` は schema 上も `tier: raw_query_plan` なら `check_recommended: false` かつ `replayable_from_report: false`、`tier: normalized_operator` なら `replayable_from_report: true` という coupling を入れると、artifact schema としてより強くなります。
>
> また、`contract_summary.environment_warnings` と同様に、top-level `warnings` も `[]` always-present にすると snapshot consumer に優しいです。
>
> 最後に、今回添付された `spanner-hacks-operators-feedback` の `SpoolScan` raw representation mismatch は、plan-report の normalizer fixture に落とす価値があります。`display_name: SpoolScan` + `metadata.spool_name` を `spool_scan` family に分類する fixture を固定すると良いと思います。

## 最終判断

設計はかなり収束している。今回の round 21 対応で、raw QueryPlan CEL、replayability、stability、topology、backend identity の主要な誤読ポイントはかなり閉じた。

次にやるべきことは、新しい contract の追加ではなく、次の小さな同期と invariant 固定だと思う。

```text
1. PLAN_CONTRACT_CANDIDATES.md の raw plan/nodes 表記を raw_plan/raw_nodes に同期
2. contract_stability tier と replayability/check recommendation の schema coupling
3. top-level warnings の always-present 化
4. backend version/image digest capture の次段階検討
5. SpoolScan raw representation fixture の追加
```
