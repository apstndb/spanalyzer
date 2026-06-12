# `spanner-query-gen` plan contract round 19 review

作成日: 2026-05-06

## 前提

今回も実装コード・実行ログ・CI output は共有されていないため、レビュー対象は添付された response / `PLAN_CONTRACTS.md` / `PLAN_CONTRACT_CANDIDATES.md` / plan-report schema / DESIGN から読み取れる public contract とドキュメント整合性です。

## 総評

今回の更新はかなり良いです。round 18 で指摘した「外部 evaluator が serialized report から同じ意味を再現できるか」という論点に対し、CEL input が serialized YAML/JSON そのものではなく normalized CEL input であることを明記し、`normalization.cel_input_defaults` も追加されています。また、root-partitionable candidate に `operator_family_counts["unknown"] == 0` を入れたこと、sort 系 contract と `ALLOW_DISTRIBUTED_MERGE` の関係を説明したこと、`scan_type` の normalized spelling を揃えたこと、`contracts[].name` uniqueness を runtime validation として明文化したことは、いずれも妥当です。

大きな方向転換は不要です。残りは、新しい predefined contract を増やすことではなく、**plan-report artifact schema が意味的に矛盾した report をどれだけ防げるか**、および **外部 replay のための machine-readable metadata をどこまで出すか** です。

## 良くなった点

### 1. CEL input default semantics が明確になった

`operators[]` / `operator_edges[]` の optional string fields は CEL 上 `""`、optional boolean fields は `false` として扱う、という整理は良いです。serialized report を直接 CEL 評価するのではなく、tool 内部の normalized CEL input を評価するという説明も重要です。

この方針により、JSON/YAML artifact を読む人は「serialized object に field がない」ことと「CEL 上の値」を混同しにくくなります。

### 2. root-partitionable candidate に `unknown == 0` が入った

`ApproximateRootPartitionablePlanShape` を candidate のままにし、`operator_family_counts["unknown"] == 0` を加えたのは良い判断です。未知の distributed operator が現れたときに、zero-distributed-fragment branch で silent pass する危険を下げられます。

この contract を `PartitionQuery` API の真の acceptance check として扱わず、PLAN-shape approximation として明記している点も維持でよいです。

### 3. `ALLOW_DISTRIBUTED_MERGE` を optimizer pinning と分けた

sort 系 contract が `ALLOW_DISTRIBUTED_MERGE` の影響を受け得ることを説明しつつ、v1alpha の `--require-optimizer-pinning` ではそれを optimizer pinning とは扱わない、という整理は妥当です。

`ALLOW_DISTRIBUTED_MERGE` は plan shape に効き得る hint であって、optimizer version/statistics package pinning とは別の軸です。この分離は CI contract の解釈を安全にします。

### 4. `contracts[].name` uniqueness を runtime validation にした

JSON Schema だけでは `contracts[].name` の一意性を自然に表現しにくいため、runtime validation で重複拒否するという判断は妥当です。`contract_evaluations[].name` を stable key として扱うなら、一意性を仕様として明文化するのは重要です。

## 追加フィードバック

### P0. `contract_rule_result.rule` と `source` の対応を schema でも締める

現在の plan-report schema はかなり厳密になっていますが、`contract_rule_result` については、`rule` と `source` の組み合わせにまだ意味的な抜けがありそうです。

ドキュメント上の意味は次のように読めます。

```text
rule: forbid_operator_family
  source: use/<predefined> または forbid[n]

rule: cel
  source: cel
```

しかし schema を見る限り、`source` は `use/<predefined> | forbid[n] | cel` の pattern であり、`rule: cel` と `source: use/no_hash_join`、または `rule: forbid_operator_family` と `source: cel` のような組み合わせを型だけでは排除しきれていないように見えます。

これは artifact consumer にとってかなり紛らわしいので、schema の conditional を次のように締めるのがよいです。

```text
if rule == cel:
  source must be cel
  predefined must be absent
  expression required
  operator_family / observed_count / max_count / matched_operator_indexes forbidden

if rule == forbid_operator_family:
  source must match use/<predefined> or forbid[n]
  expression forbidden
  operator_family / observed_count / max_count / matched_operator_indexes required
```

`source: use/<name>` と `predefined: <name>` の完全一致は runtime invariant でもよいですが、少なくとも `rule` と `source kind` の対応は schema で表現しやすく、public artifact contract として締める価値があります。

### P1. `normalization.cel_input_defaults.applies_to` は exact value にした方がよい

`normalization.cel_input_defaults` を report に追加したのは良いですが、schema 上は `applies_to` が enum array で、空配列や片方だけの配列も通り得る形に見えます。

v1alpha の意味が固定なら、次のどちらかに寄せるとよいです。

```yaml
normalization:
  cel_input_defaults:
    optional_string: ""
    optional_boolean: false
    applies_to:
    - operators[]
    - operator_edges[]
```

を exact set として schema で要求する。

または、より machine-readable にするなら、対象 field 名まで出す。

```yaml
normalization:
  cel_input_defaults:
    optional_string: ""
    optional_boolean: false
    fields:
      operators[]:
        optional_string:
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
        optional_boolean:
        - full_scan
      operator_edges[]:
        optional_string:
        - type
        - variable
```

後者は少し重いですが、外部 evaluator が serialized report から CEL を replay するならかなり親切です。simple に保つなら、少なくとも `applies_to` を exact set にするだけでもよいです。

### P1. `ApproximateRootPartitionablePlanShape` の name は今のまま保守的に維持する

現在の root-partitionable CEL は、`subquery_cluster_node` と `call_type != "local"` を使った PLAN-shape approximation としては良いです。ただし、これは引き続き `PartitionQuery` の acceptance を保証するものではありません。

そのため、将来 predefined に昇格する場合でも、名前は次のように保守的にした方がよいです。

```text
approximate_root_partitionable_plan_shape
```

または、さらに誤解を避けるなら、

```text
root_partitionability_not_disproved_by_plan_shape
```

のように “positive guarantee” ではなく “PLAN shape review” であることを名前に残すのも選択肢です。現時点では candidates に留める方針でよいです。

### P1. metadata-derived CEL の stability reason を report で確認できるようにする

DESIGN では、`subquery_cluster_node` や `call_type` のような raw PlanNode metadata を `normalized_operators[]` にコピーして使う normalized-tier CEL は、`normalized_operator` tier のままにしつつ、metadata-derived dependency を stability reason に入れる方針になっています。

これは良い設計です。次に固定するなら、root-partitionable candidate のように metadata-derived field を使う CEL contract の report snippet で、次のような `stability.reasons` を明示するとレビューしやすくなります。

```yaml
stability:
  tier: normalized_operator
  reasons:
  - contract uses the normalized plan-report view
  - contract reads metadata-derived normalized fields: subquery_cluster_node, call_type
  check_recommended: true
```

CEL expression の静的解析で field usage を取るのが難しいなら、最初は docs/golden fixture の expected output として固定するだけでも十分です。

### P2. `ALLOW_DISTRIBUTED_MERGE` 依存 contract は example を 1 つだけ置くと誤用が減る

`ALLOW_DISTRIBUTED_MERGE` を optimizer pinning と分けた説明は良いです。さらに、sort 系 contract で hint 依存の SQL を使う場合の example を 1 つ置くと、利用者が `--require-optimizer-pinning` に過剰な意味を期待しにくくなります。

たとえば、以下のような短い注意で十分です。

```text
When a sort contract depends on ALLOW_DISTRIBUTED_MERGE, keep the hint in the
rendered SQL and review sql_sha256 plus optimizer matrix evidence. The
--require-optimizer-pinning flag only checks requested optimizer version and
statistics package, not statement-level plan-shape hints.
```

現在の文面でもかなり伝わっていますが、CI 利用者向けには明示 example があると安全です。

### P2. `contracts[].name` uniqueness は invalid fixture も欲しい

runtime validation に寄せる方針は良いです。あとは次の invalid fixture を用意しておくと、将来 schema / parser / evaluator のどこかで一意性が漏れるのを防げます。

```yaml
version: v1alpha-plan-contracts
contracts:
- name: NoSort
  target: query/Foo
  use: [no_explicit_sort]
- name: NoSort
  target: query/Bar
  use: [no_hash_join]
```

期待 diagnostic は例えば次です。

```text
plan_contract.duplicate_contract_name
```

重複 `forbid.operator_family` と同様に、重複 `contracts[].name` も diagnostic ID を固定すると CI で扱いやすいです。

## 反映しなかった判断について

### `stability.features` を入れない判断は妥当

現時点では `stability.reasons` で十分だと思います。`features` のような machine-readable list は、使い道が明確になってから入れればよいです。

### serialized report の optional fields を全部明示出力しない判断も妥当

optional string / boolean fields をすべて `""` / `false` として serialized report に出すと artifact が冗長になります。今回のように `normalization.cel_input_defaults` と docs で replay semantics を明示する方が、v1alpha ではバランスが良いです。

ただし、外部 evaluator support を本気で打ち出すなら、将来的には default 対象 field list を machine-readable にする価値があります。

## 結論

今回の更新で、plan contract 周りはかなり収束しています。次に見るべきものは新しい predefined contract ではなく、以下です。

1. `contract_rule_result.rule` と `source` の schema-level coupling
2. `normalization.cel_input_defaults.applies_to` の exact-set 化、または default field list の machine-readable 化
3. root-partitionable candidate の metadata-derived stability reason の fixture 化
4. duplicate `contracts[].name` の diagnostic ID / invalid fixture
5. sort hint dependency の README / PLAN_CONTRACTS example

特に P0 は `rule` / `source` の不整合です。これは実装が正しいなら runtime では出ないかもしれませんが、plan-report schema を public contract として配る以上、schema だけで明らかに矛盾した artifact を弾けるようにしておくと、外部 consumer にとってかなり安心です。
