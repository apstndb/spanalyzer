# `spanner-query-gen` plan contract round 20 review

作成日: 2026-05-06

## 前提

最新の `spanner-query-gen-plan-contract-round19-review-response-ja.md`、`PLAN_CONTRACTS.md`、`PLAN_CONTRACT_CANDIDATES.md`、`DESIGN.md`、`spanner-query-gen.plan-report.v1alpha.schema.json`、`spanner-query-gen.plan-contracts.v1alpha.schema.json` を前提に再レビューする。

今回も実装コード・実行ログ・CI output は共有されていないため、ここでは **public contract / schema / docs / 補助ドキュメントの整合性レビュー** に限定する。`go test` 実行や fixture 追加の記述は、外部から検証済みの事実ではなく、開発側の内部作業ツリー上の主張として扱う。

## 結論

今回の更新はかなり良い。round 19 で指摘した以下の点は、v1alpha artifact contract としてかなり締まっている。

- `contract_rule_result.rule` と `source` の対応を schema で制約した。
- `normalization.cel_input_defaults.applies_to` を `operators[]` / `operator_edges[]` の exact set として扱うようにした。
- metadata-derived CEL の `stability.reasons` に、`call_type` / `subquery_cluster_node` のような具体的 field 名を出す方針にした。
- duplicate contract name の diagnostic ID を `plan_contract.duplicate_contract_name` に固定した。
- root-partitionable は引き続き candidate / CEL example に留め、predefined contract に昇格していない。
- `ALLOW_DISTRIBUTED_MERGE` を optimizer pinning とは別の plan-shape hint として扱い、SQL digest / optimizer matrix evidence と合わせて review する方針を維持している。

大きな方向転換は不要。残っている論点は、**schema でどこまで machine-checkable にするか** と、**外部 evaluator が serialized report から同じ意味を再現できるか** の細部に絞られる。

## 追加フィードバック

### 1. `rule: cel` では `failure_kind: classification_unknown` を禁止した方がよい

現在の docs は、CEL は literal evaluation であり、predefined / direct `forbid` の `classification_unknown` safeguard は自動では適用されない、と説明している。この説明は良い。

ただし、plan-report schema 上は次のような artifact が通り得るように見える。

```yaml
rule: cel
source: cel
status: fail
expression: "false"
failure_kind: classification_unknown
diagnostic_id: plan.foo
```

これは docs の意味と少しずれる。`classification_unknown` は、aggregate-family / join-family の direct or predefined `forbid_operator_family` が、fallback `aggregate` / `join` node の存在によって「特定 family が無いとは証明できない」と判断したときの failure kind として読むのが自然である。

したがって v1alpha では、schema で次のように締めるのがよい。

```text
if rule == cel and status == fail:
  failure_kind must be violation
  diagnostic_id must be absent
```

将来 CEL evaluator に unknown / indeterminate / type-error のような状態を入れたくなったら、`classification_unknown` を流用するのではなく、`cel_evaluation_error` や `cel_unknown` のような別 failure kind を追加する方が読みやすい。

優先度: **P1**

### 2. `source: use/<name>` と `predefined: <name>` の完全一致は runtime invariant でよいが、negative fixture を強めたい

round 19 response では、schema は value domain を制約し、`source: use/<name>` と `predefined: <name>` の完全一致は runtime invariant として扱う、と整理している。この判断は simple でよい。

ただし public artifact としては、次のような不整合を絶対に出さないことが重要になる。

```yaml
rule: forbid_operator_family
source: use/no_hash_join
predefined: no_merge_join
```

JSON Schema で `use/<name>` と `predefined` の suffix equality を完全に表すには、predefined name ごとの conditional を生成する必要があり、少し重い。v1alpha では runtime invariant のままでよいが、以下は固定した方がよい。

```text
- source starts with use/ なら predefined は必須
- predefined は source の suffix と完全一致
- source が forbid[n] または cel なら predefined は禁止
- 不一致 artifact を生成しない golden / regression test を持つ
```

優先度: **P2**

### 3. `cel_input_defaults` は良いが、外部 replay 用には field list を machine-readable にする余地がある

`applies_to` を `operators[]` / `operator_edges[]` の exact set にしたのは良い。docs でも、optional string fields は `""`、optional boolean fields は `false` と説明され、対象 field も列挙されている。

一方で、serialized report だけを受け取った外部 evaluator が CEL を replay する場合、現状では「どの field が optional string / optional boolean なのか」は docs を読む必要がある。artifact self-contained 性を少し上げるなら、次のような field を追加できる。

```yaml
normalization:
  cel_input_defaults:
    optional_string: ""
    optional_boolean: false
    applies_to:
    - operators[]
    - operator_edges[]
    optional_string_fields:
      operators[]:
      - execution_method
      - iterator_type
      - scan_method
      - scan_format
      - scan_type
      - scan_target
      - seekable_key_size
      - join_type
      - join_configuration
      - call_type
      - distribution_table
      - subquery_cluster_node
      operator_edges[]:
      - type
      - variable
    optional_boolean_fields:
      operators[]:
      - full_scan
```

ただし、これは artifact をやや重くする。現時点で外部 evaluator を強く支援する必要がないなら、今の compact form + docs で十分。採用するとしても P2 でよい。

優先度: **P2 / optional**

### 4. `contract_summary` と `contract_evaluations` の count invariant は schema ではなく runtime self-check でよい

`PLAN_CONTRACTS.md` では、次の invariant が明記されている。

```text
contract_summary.contracts == len(contract_evaluations)
contract_summary.passed == count(contract_evaluations[].status == pass)
contract_summary.failed == count(contract_evaluations[].status == fail)
contract_summary.not_evaluated == count(contract_evaluations[].status == not_evaluated)
```

これは JSON Schema で表しにくいので、schema に無理に入れなくてよい。ただし CI artifact として重要なので、plan-report 生成時に self-check するか、golden fixture / unit test で固定した方がよい。

同様に、`target_summary.included_count == planned + errors + skipped` も runtime invariant として十分。ただし、report generator が内部的に一度 self-validate できると、artifact consumer の信頼性が上がる。

優先度: **P2**

### 5. `contract_rule_result` の `classification_unknown` は direct/predefined `forbid` 専用だと明文化できている

これは前回から良くなっている点。`PLAN_CONTRACTS.md` は、CEL は predefined/direct `forbid` と同じ `classification_unknown` safeguard を自動では持たないと説明している。さらに、join / aggregate fallback family の扱いも整理されている。

今後 `no_full_scan`、`require_seekable_scan`、`no_residual_condition` のような候補を追加する場合も、同じ原則を維持するとよい。

```text
- predefined/direct forbid: conservative failure semantics を持てる
- CEL: literal expression evaluation
- CEL で conservative semantics が欲しい場合は、fallback family や unknown を明示的に見る
```

優先度: **維持**

### 6. root-partitionable は candidate のままでよい

`ApproximateRootPartitionablePlanShape` は引き続き candidate に留める判断でよい。`subquery_cluster_node` / `call_type` を使う CEL は有用だが、これは PLAN-shape approximation であって `PartitionQuery` API acceptance そのものではない。

特に、現在の example が `operator_family_counts["unknown"] == 0` guard を持つのは良い。未知の distributed operator が現れたとき、zero-distributed-fragment branch で silent pass しない。

将来 predefined に昇格する場合も、名前は `approximate_root_partitionable_plan_shape` のように、近似であることを明示した方がよい。より authoritative なものは、別 backend probe として `partition_query_accepts` のように分けるべき。

優先度: **維持**

### 7. 新しい predefined contract を増やす段階ではまだない

`PLAN_CONTRACT_CANDIDATES.md` には、`no_full_scan`、`require_seekable_scan`、`no_residual_condition`、`no_back_join` などの候補が整理されている。これは良い候補集だが、v1alpha の今の状態では、追加 predefined を急がない方がよい。

理由は、現在の plan-report artifact contract がようやく安定しつつあるため。次に優先すべきなのは、候補を増やすことよりも、既存 predefined / direct / CEL の artifact semantics を fixture で固定すること。

特に `no_back_join` は schema-aware でないと誤検出しやすい。候補ドキュメントにも、index scan と table scan が同じ base table に属するかどうかを確認する必要があると書かれており、この判断は妥当。

優先度: **維持**

## そのまま送れる返答案

以下のように返すとよいと思います。

> round 19 の反映はかなり良いです。`rule/source` coupling、`cel_input_defaults.applies_to` の exact set、metadata-derived stability reason、duplicate contract name diagnostic、root-partitionable を candidate に留める判断はいずれも妥当です。大きな方向転換は不要です。
>
> 追加で一番気になるのは、`rule: cel` の failure kind です。docs では CEL は literal evaluation であり、predefined/direct `forbid` の `classification_unknown` safeguard は自動適用されないと説明されています。一方、schema 上は `rule: cel` + `failure_kind: classification_unknown` が通り得るように見えます。v1alpha では `rule: cel` の fail は `failure_kind: violation` のみに制限し、`classification_unknown` は `forbid_operator_family` 専用にした方が artifact contract として読みやすいです。将来 CEL evaluator に unknown / type-error を入れるなら、`cel_unknown` や `cel_evaluation_error` のような別 failure kind を追加する方がよいと思います。
>
> `source: use/<name>` と `predefined: <name>` の完全一致は、schema で無理に表さず runtime invariant のままでよいと思います。ただし、`source: use/no_hash_join` + `predefined: no_merge_join` のような不整合 artifact を生成しない negative fixture は固定してください。
>
> `cel_input_defaults` は compact で良いです。外部 evaluator が serialized report だけから CEL を replay する必要が強くなったら、optional string / boolean field list を machine-readable に出す余地がありますが、v1alpha では docs + `applies_to` exact set で十分だと思います。
>
> root-partitionable は引き続き candidate のままでよいです。`operator_family_counts["unknown"] == 0` guard を持つ PLAN-shape approximation として扱い、将来 predefined にする場合も `approximate_root_partitionable_plan_shape` のような保守的な名前にするのが安全です。authoritative な確認は `partition_query_accepts` のような別 backend probe に分けるべきです。

## 最終判断

現在の plan-contract 周辺は、かなり最終形に近い。今後の作業は、新しい predefined contract を増やすよりも、次の小さな締め込みを優先するのがよい。

1. `rule: cel` では `failure_kind: classification_unknown` を禁止する。
2. `source: use/<name>` と `predefined: <name>` の一致を runtime negative fixture で固定する。
3. `contract_summary` / `target_summary` の count invariant を self-check または test で固定する。
4. root-partitionable は candidate のまま維持する。
5. `no_back_join` など schema-aware contract は、catalog metadata を plan-report に入れるまで predefined 化しない。

全体としては、「main config は simple、plan contract は optional、artifact は CI で読める」という方針がかなりうまく守られている。
