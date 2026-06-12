# `spanner-query-gen` plan contract round 17 review

作成日: 2026-05-06

## 前提

最新の response / README / DESIGN / `PLAN_CONTRACTS.md` / `PLAN_CONTRACT_CANDIDATES.md` / plan-report schema / operator observation notes を前提に再レビューする。

今回も実装コード、実行ログ、CI output は共有されていないため、ここでの判断は **public contract、schema、README/DESIGN、補助ドキュメントの整合性レビュー** に限定する。response 内の `go test ./...` などの記述は内部作業ツリー上の主張として扱い、外部から検証済みの事実とは扱わない。

## 総評

今回の更新はかなり良い。前回指摘した以下の論点は、仕様・schema・docs のいずれにもかなり反映されている。

- `queries[].status: error | skipped` で success-only plan fields を禁止する方針
- `target_summary.excluded` と `queries[].status: skipped` の語彙分離
- join-specific contract でも fallback `join` を `classification_unknown` として保守的に fail させる方針
- `source: use/<name>` と `predefined: <name>` の equality を runtime invariant として固定する方針
- `status: no_targets` の最小例
- root-partitionable 候補を display-name 列挙ではなく `subquery_cluster_node` metadata ベースに寄せる検討

特に、`PLAN_CONTRACTS.md` の Status Invariants はかなり読みやすくなっている。top-level `status` が artifact production status であり、target-level success は `target_summary.*` と `queries[].status` で見る、という分離は CI artifact として正しい。また、`queries[].status: error | skipped` では plan normalization fields が absent であり、成功 evidence として読んではいけない、と明記された点も良い。

大きな方向転換は不要。残る論点は、新しい predefined contract を増やすことではなく、**CEL input と serialized report の shape をさらに誤読しにくくすること**、および **root-partitionable 候補を PLAN-shape approximation として安全に閉じ込めること** だと思う。

## 追加フィードバック

### 1. `operator_edges` は `status: ok` で always present にした方がよい

現在の response では、`operator_edges` は success-only field だが、成功時でも edge が空なら serialized output で省略され得るため、`status: ok` の required field から外した、と説明されている。

ただ、v1alpha の plan-report はすでに CI artifact / CEL input として扱われているため、ここは **空配列でも always present** の方が downstream に優しい。

理由は次の通り。

- `PLAN_CONTRACTS.md` は normalized topology fields として `operator_edges` を説明している。
- `PLAN_CONTRACT_CANDIDATES.md` にも `operator_edges.exists(...)` のような CEL example がある。
- `operators[].child_indexes` / `operators[].descendant_indexes` / `operators[].subtree_family_counts` は topology view として required 化されている。
- `operator_edges` だけが optional だと、report consumer は `missing means empty` なのか `not computed` なのかを追加で解釈する必要がある。

したがって、可能なら schema を次の方向に寄せるのがよい。

```text
queries[].status: ok:
  required:
  - operator_edges

operator_edges:
  [] when the target has no child links or the normalized edge list is empty
```

もし artifact size / markdown readability のために省略したいなら、少なくとも `PLAN_CONTRACTS.md` に次のような文を入れてほしい。

```text
In serialized reports, operator_edges may be absent when the edge list is empty.
CEL input treats absent operator_edges as an empty list.
```

ただし、私は `operator_edges: []` を常に出す方が v1alpha artifact contract としてはきれいだと思う。

### 2. CEL input における optional metadata field の default semantics を明記した方がよい

root-partitionable 候補の CEL は次のような shape に寄っている。

```cel
operators.filter(o,
  o.subquery_cluster_node != "" &&
  o.call_type != "local")
```

これは良い方向だが、plan-report schema 上では `subquery_cluster_node` や `call_type` は optional string field に見える。serialized YAML/JSON に field が存在しない場合、CEL input で `o.subquery_cluster_node != ""` がどう評価されるかは明文化されていない。

ここはかなり重要。CEL input が Go struct 由来で optional string の zero value を `""` として見せるなら問題ないが、JSON/YAML map 由来で missing field として扱うなら expression が brittle になる可能性がある。

おすすめは、`PLAN_CONTRACTS.md` に次のような contract を入れること。

```text
CEL input uses normalized operator objects, not the serialized YAML object directly.
Optional string metadata fields such as subquery_cluster_node, call_type,
scan_target, scan_type, seekable_key_size, iterator_type, and join_type default
to the empty string in CEL when metadata is absent.
Optional boolean metadata fields default to false.
```

または、serialized report でも CEL input と同じように optional metadata fields を always present にする。ただし report が大きくなるので、前者の「CEL input default semantics を明記する」で十分だと思う。

この点を明記しないと、root-partitionable 候補だけでなく、`scan_target` / `full_scan` / `seekable_key_size` を使う future CEL examples でも同じ不安が出る。

### 3. root-partitionable は良い候補だが、名前と位置づけを保守的にした方がよい

`subquery_cluster_node` metadata を使う方向は、display-name 列挙よりかなり良い。observation note でも、SELECT 系 plan では `Distributed Union`、`Distributed Cross Apply`、`Push Broadcast Hash Join` などの distributed fragment signal として観測されており、Local Distributed Union は `call_type: Local` で除外する、という整理ができている。

一方で、docs でもすでに書かれている通り、これは `PartitionQuery` API の acceptance を直接確認するものではない。さらに response では、DML `Apply Mutations` は distributed operator に見えるが `subquery_cluster_node` を持たない counterexample として扱っている。

したがって、この候補は v1alpha では以下のような名前・位置づけに留めるのが安全。

```text
candidate name:
  root_partitionable_plan_shape
  or approximate_root_partitionable_plan_shape

not:
  require_root_partitionable
  root_partitionable
```

`PLAN_CONTRACT_CANDIDATES.md` ではすでに candidate 扱いなので、predefined contract に入れる必要はない。もし将来 predefined にするなら、`PartitionQuery` を実際に呼ぶ backend probe と PLAN-shape contract を別物として扱うべき。

```text
PLAN-shape contract:
  approximate_root_partitionable_plan_shape

API acceptance probe:
  partition_query_accepts
```

この分離は重要。PLAN-shape contract は review artifact として有用だが、API acceptance probe は live/backend behavior に近く、安定性・必要権限・環境依存性が違う。

### 4. `queries[].status: skipped` の `error` field は将来 `reason` / `message` に分ける余地を残したい

今回の response では、`error | skipped` のとき target identity と `error` が reliable、と整理されている。schema を単純に保つにはこれでよい。

ただし、`skipped` は必ずしも error ではない。たとえば non-Spanner source、BigQuery outer query、external dataset、target selector による対象外などは、failure というより intentional skip に近い。

v1alpha ですぐ変えなくてもよいが、将来は以下のように分けるとより読みやすい。

```yaml
status: skipped
reason: target.non_spanner_source
message: "BigQuery outer SQL is not a Spanner plan-report target."
```

今のまま進めるなら、`error` の description を少し緩めて、`Analysis or rendering error` ではなく `Target acquisition error or skip message` のようにするだけでも誤読は減る。

### 5. `excluded_target.reason` は pattern でよいが、現行値の example list は欲しい

`excluded_target.reason` を enum ではなく namespaced pattern にした判断は妥当。diagnostic ID と同様に、将来の追加で schema を壊しにくい。

一方、完全に pattern だけだと downstream consumer は実際にどんな reason が出るのか推測しにくい。enum に戻す必要はないが、`PLAN_CONTRACTS.md` に current examples を載せるとよい。

```text
Current reason examples:
- target.filtered_by_selector
- target.non_spanner_catalog
- target.missing_catalog
- target.external_dataset
```

実際の実装値に合わせてでよい。ここは machine contract ではなく docs の例で十分。

### 6. `classification_unknown` は良いが、CEL contract での扱いも一文あるとよい

aggregate / join の predefined contract と direct `forbid.operator_family` では、fallback `aggregate` / `join` があると `classification_unknown` で fail する方針になっている。これは非常に良い。

ただ、CEL contract ではユーザーが自由に `operator_family_counts["hash_join"] == 0` のように書けるため、fallback `join` を見落として pass させることもできる。

これは CEL の自由度として許容でよいが、docs に一文入れると親切。

```text
CEL contracts are evaluated literally. They do not automatically apply the
classification_unknown safeguards used by predefined and direct forbid rules.
If a CEL contract forbids a specific join or aggregate family and wants the same
conservative behavior, also check operator_family_counts["join"] or
operator_family_counts["aggregate"], or use predefined/direct forbid rules.
```

これにより、stable CI には predefined/direct forbid を推奨し、CEL は exploratory / query-specific という今の方針がさらに明確になる。

### 7. `subquery_cluster_node` を raw-metadata-derived field として stability reason に反映できるとよい

root-partitionable 候補は normalized operator view を使っているが、信号自体は raw PlanNode metadata 由来で、human-readable `spannerplan reference` には出ないと説明されている。この整理は良い。

ただ、contract stability としては単純な operator family count より少し fragile かもしれない。可能なら、CEL が `subquery_cluster_node` のような raw-metadata-derived field を参照した場合、`stability.reasons` に次のような説明を足すと report だけで判断しやすい。

```yaml
stability:
  tier: normalized_operator
  reasons:
  - contract uses normalized operator view
  - contract references raw PlanNode metadata exposed through normalized_operators[].subquery_cluster_node
```

tier は `normalized_operator` のままでよい。`raw_query_plan` tier に落とす必要はないと思う。理由は、ユーザーが raw `plan` / `nodes` を直接参照しているわけではなく、normalizer が露出した normalized field を使っているから。ただし metadata-derived であることは reason に出た方がよい。

## そのまま送れる返答案

以下のように返すとよいと思う。

> 今回の更新はかなり良いです。前回指摘した `error | skipped` の success-only plan fields 禁止、`target_summary.excluded` と `queries[].status: skipped` の語彙分離、join-specific contract の `classification_unknown`、`source: use/<name>` と `predefined` の runtime invariant、`status: no_targets` の例は、CI artifact としてかなり読みやすくなっています。
>
> 大きな方向転換は不要です。追加で一番気になるのは、`operator_edges` が `status: ok` でも optional になっている点です。`PLAN_CONTRACTS.md` では `operator_edges` を normalized topology field として説明しており、CEL example でも使うので、成功時は空配列でも `operator_edges: []` を常に出した方が downstream consumer に優しいと思います。省略を続けるなら、serialized report では absent になり得るが CEL input では empty list として扱う、という semantics を明記してください。
>
> 次に、root-partitionable 候補の CEL が `o.subquery_cluster_node != "" && o.call_type != "local"` のような optional metadata field を使っています。schema 上は `subquery_cluster_node` / `call_type` が optional に見えるので、CEL input では absent string metadata が empty string として見えるのか、あるいは serialized report と同じ missing field なのかを明文化した方がよいです。`scan_target` や `seekable_key_size` を使う future CEL examples にも同じ論点が出ます。
>
> root-partitionable の方向性自体は良いですが、これは `PartitionQuery` API acceptance を直接検証するものではなく PLAN-shape approximation です。v1alpha では `root_partitionable_plan_shape` または `approximate_root_partitionable_plan_shape` のような candidate name に留め、将来 `PartitionQuery` を実際に呼ぶ backend probe とは別 contract として扱うのが安全だと思います。
>
> さらに、aggregate / join の `classification_unknown` 保護は predefined/direct forbid では良い状態ですが、CEL contract では自動では効かないはずです。docs に「CEL は literal evaluation なので、同じ conservative behavior が必要なら direct forbid/predefined を使うか、`operator_family_counts["join"]` / `operator_family_counts["aggregate"]` も明示的に確認する」と一文入れると誤用を減らせます。
>
> あとは `skipped` で `error` field を使う naming が少しだけ気になります。今すぐ変える必要はありませんが、将来的には `reason` / `message` に分けると、intentional skip と actual error の区別が clearer になります。

## 結論

現時点で新しい predefined contract を増やす必要はない。plan contract surface はすでに十分に豊かになっているので、次は以下を優先するのがよい。

1. `operator_edges` の always-present 化、または absent-as-empty semantics の明文化
2. CEL input における optional metadata field の default semantics 明文化
3. root-partitionable を PLAN-shape approximation として candidate に閉じ込める
4. CEL と predefined/direct forbid の `classification_unknown` semantics 差分の明文化
5. `skipped` の `error` field naming の将来整理

この段階では、v1alpha の plan-report artifact はかなり外部レビュー可能な contract に近づいている。残りは機能追加より、artifact consumer が誤読しにくい小さな仕様固定だと思う。
