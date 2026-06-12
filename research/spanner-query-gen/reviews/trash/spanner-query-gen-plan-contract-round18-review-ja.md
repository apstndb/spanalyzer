# spanner-query-gen plan contract round 18 review

作成日: 2026-05-06

## 前提

今回も、実装コード・実行ログ・CI output そのものは共有されていない前提で、添付された response / README / DESIGN / schema / 補助ドキュメントから読み取れる public contract のレビューとして扱います。

結論として、設計はかなり収束しています。今回の更新で、`operator_edges` の成功時 always-present、CEL input の optional metadata default、root-partitionable の PLAN-shape approximation 化、CEL と `classification_unknown` の差分説明、metadata-derived field の stability reason は、いずれも良い整理です。

大きな方向転換は不要です。残るフィードバックは、新しい predefined contract を増やすことではなく、**外部 review artifact として第三者が同じ意味で読めるか**をさらに締めるものです。

## 良い点

### 1. `operator_edges` を成功時 always present に戻したのは正しい

`queries[].status: ok` では `operator_edges` が required で、edge がない場合も `operator_edges: []` を出す方針になりました。これは CEL examples と serialized artifact の見た目を揃えるので良いです。

前回懸念していた「topology fields を使う CEL examples があるのに、成功 report で `operator_edges` が absent になり得る」問題はほぼ解消されています。

### 2. root-partitionable を candidate / approximate に留めたのは良い

`ApproximateRootPartitionablePlanShape` という名前に寄せ、`PartitionQuery` API acceptance probe とは別物だと明記した判断は妥当です。

これは plan-shape review としては価値がありますが、実 API が partition を作れるかどうかの source of truth ではありません。その線引きを保っているのは良いです。

### 3. `classification_unknown` と CEL の差分を明記したのは良い

predefined/direct `forbid` は aggregate/join fallback family に対して保守的に `classification_unknown` で fail する。一方、CEL は literal evaluation であり、その safeguard は自動適用されない。この説明は重要です。

特に、`operator_family_counts["hash_join"] == 0` のような CEL を書いたユーザーが、generic `join` を見落として silent pass する可能性があるため、docs に「同じ保守性が必要なら `join` / `aggregate` も見る」と書いたのは良いです。

### 4. optimizer-version matrix は良い evidence になっている

`OPTIMIZER_VERSION_MATRIX_OBSERVATIONS.md` に、Spanner Omni image digest、`spanemuboost` / `spannerplan` version、optimizer statistics package、実行 command、version matrix、`ALLOW_DISTRIBUTED_MERGE` cross-check が入っています。

これは normative contract ではなく observed vocabulary / plan-shape evidence として扱うなら非常に有用です。

## 追加フィードバック

### P1. CEL input default semantics と serialized report の差分をもう一段だけ明確にする

今回の response では、CEL input は serialized YAML/JSON object そのものではなく normalized operator object を使い、absent string metadata は `""`、absent boolean metadata は `false` と説明されています。これは内部 evaluator としては良い方針です。

ただし、`plan-report --output json/yaml` を外部 CI artifact として読んだ第三者が、同じ CEL を自前 evaluator で再評価しようとした場合、serialized `normalized_operators[]` では optional fields が absent であり、内部 CEL input だけが default を持つ、という差分が残ります。

特に対象は operator metadata だけではありません。`operator_edges[]` も `type` / `variable` が optional field なので、`operator_edges.exists(e, e.parent_index == 0 && e.type == "Input")` のような CEL example に対して、`e.type` absent 時の default semantics を明示する必要があります。

最小修正案は、`PLAN_CONTRACTS.md` に次のような一文を追加することです。

```text
CEL evaluation is performed against the tool's normalized CEL input, not by
running CEL directly over the serialized report JSON/YAML. Optional string
fields in both operators[] and operator_edges[] default to "" in CEL; optional
boolean fields default to false. External evaluators that replay CEL from the
report must apply the same defaults.
```

より artifact として強くするなら、report 側に次のような metadata を出すのもありです。

```yaml
normalization:
  cel_input_defaults:
    optional_string: ""
    optional_boolean: false
    applies_to:
    - operators[]
    - operator_edges[]
```

さらに強くするなら serialized report 自体に empty string / false を明示的に出す方法もありますが、artifact がかなり冗長になります。今の simple 方針なら、まずは docs と `normalization.cel_input_defaults` 程度で十分です。

### P1. root-partitionable CEL は `unknown` family に対する保守性を足した方がよい

現在の approximate root-partitionable example は、`subquery_cluster_node != "" && call_type != "local"` を distributed-fragment signal として使っています。観測結果上は妥当ですが、将来未知の distributed operator が `subquery_cluster_node` を持たない、または normalizer が `unknown` に落とす場合、第二 branch の「non-local distributed metadata が 0 個」条件で pass する可能性があります。

candidate であり predefined ではないので大きな問題ではありません。ただ、CI contract example としては、次のように `unknown` を拒否する strict variant を示すと安全です。

```cel
operator_family_counts["unknown"] == 0 &&
(
  (
    operators.exists(o,
      o.index == 0 &&
      o.family == "distributed_union" &&
      o.subquery_cluster_node != "" &&
      o.call_type != "local") &&
    operators.filter(o,
      o.subquery_cluster_node != "" &&
      o.call_type != "local").size() == 1
  ) ||
  (
    operators.filter(o,
      o.subquery_cluster_node != "" &&
      o.call_type != "local").size() == 0
  )
)
```

もし `unknown` を blanket reject したくないなら、docs に「この example は unknown distributed operator を検出する authoritative rule ではない」と一文足すだけでもよいです。

### P1. sort 系 contract は `ALLOW_DISTRIBUTED_MERGE` の影響を受けることを docs に短く書く

optimizer-version matrix で、`ALLOW_DISTRIBUTED_MERGE=FALSE` により sort / sorted-merge related な plan shape が変わることが観測されています。これは `no_explicit_sort` / `no_full_sort` / `no_minor_sort` のような sort contract に直接関係します。

現在、`--require-optimizer-pinning` は optimizer version / statistics package の requested pinning を見る設計で、SQL statement hint 由来の pinning は推論しない、という整理になっています。この方針自体は良いです。

追加で、`PLAN_CONTRACTS.md` の optimizer section に次のような注記があるとよいです。

```text
Sort-related plan contracts can also be sensitive to statement hints such as
ALLOW_DISTRIBUTED_MERGE. v1alpha does not model that hint as optimizer pinning;
when a contract depends on it, make the hint visible in the rendered SQL and
review the SQL digest / optimizer matrix evidence together.
```

これは新機能ではなく、観測ドキュメントを contract 利用時の注意に接続するだけです。

### P2. `PLAN_CONTRACT_CANDIDATES.md` に残る `scan_type` の表記ゆれを直す

候補ドキュメントの Full Text Search examples は `scan_type == "search_index_scan"` / `"table_scan"` になっており、observations と一致しています。一方、後半の “Require A Specific Index Scan And No Table Scan” では `scan_type == "IndexScan"` / `"TableScan"` が残っています。

これは non-normative candidates document なので重大ではありませんが、normalized metadata の spelling を docs 全体で揃えた方がよいです。

修正案:

```cel
operators.exists(o,
  o.scan_target == "SongsBySongNameStoring" &&
  o.scan_type == "index_scan") &&
operators.all(o, o.scan_type != "table_scan")
```

もし `index_scan` ではなく別の normalized spelling を採用しているなら、その spelling を `PLAN_CONTRACTS.md` の metadata section か observations に明示してください。今の observations では `SearchIndexScan` が `search_index_scan` に正規化されることは明記されていますが、通常 index/table scan の normalized value も examples で揃えると誤読が減ります。

### P2. metadata-derived normalized fields の stability reason は良いが、将来 machine-readable にしてもよい

今回、`subquery_cluster_node` や `call_type` のような raw PlanNode metadata 由来の normalized field を CEL が参照した場合、`stability.tier` は `normalized_operator` のままにしつつ、`stability.reasons` に

```text
contract references raw PlanNode metadata exposed through normalized_operators[]
```

を追加する方針になっています。これは simple で良いです。

将来、外部 CI が stability をより機械的に扱うようになったら、次のような machine-readable field があると便利かもしれません。

```yaml
stability:
  tier: normalized_operator
  features:
  - normalized_operator_family
  - normalized_operator_metadata
```

ただし、これは v1alpha で急ぐ必要はありません。今は `reasons` で十分です。

### P2. contract name uniqueness を docs に一文だけ足す

contract file schema は `uniqueItems` で完全重複 object しか弾けないため、`contracts[].name` の重複は runtime validation になるはずです。すでに direct `forbid.operator_family` duplicate は runtime diagnostic として説明されています。

同じ粒度で、`contracts[].name` duplicate も runtime validation の対象だと一文書いておくと、report consumer が `contract_evaluations[].name` を key として使うときに安心できます。

```text
Contract names must be unique within one contract file. Runtime validation
rejects duplicate names because report consumers may use contract_evaluations[].name
as a stable key.
```

## これ以上 simple にするなら

現時点で削るべき大きな機能は見当たりません。`plan-report --contracts` は main config と分離され、PROFILE/runtime stats も除外され、predefined set も広げすぎていません。

ただし README の本体は、引き続き最小例だけに留めるのがよいです。root-partitionable CEL、optimizer-version matrix、hint-specific observations は `PLAN_CONTRACTS.md` / `PLAN_CONTRACT_CANDIDATES.md` / observations documents に置く現在の分離がちょうど良いです。

## まとめ

今回の更新はかなり良いです。残りは、次の小さな同期だけで十分だと思います。

1. CEL default semantics を `operators[]` だけでなく `operator_edges[]` にも明示する。
2. 外部 evaluator が serialized report から CEL を replay する場合の default application を明記する。
3. root-partitionable candidate に `unknown == 0` の strict variant、または unknown limitation を足す。
4. sort contracts と `ALLOW_DISTRIBUTED_MERGE` の関係を optimizer section に一文足す。
5. `PLAN_CONTRACT_CANDIDATES.md` の `scan_type` casing を normalized spelling に揃える。
6. `contracts[].name` uniqueness を runtime invariant として docs に書く。

いずれも大きな設計変更ではなく、v1alpha の artifact を第三者が誤読しないための最後の整備です。
