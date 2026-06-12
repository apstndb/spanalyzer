# `spanner-query-gen` plan contract round 13 review

作成日: 2026-05-06

## 前提

今回も、実装コード・実行ログ・CI output は共有されていない前提でレビューします。したがって、ここでの評価対象は、共有された response / README / DESIGN / schema / 補助ドキュメントから外部レビュー可能な public contract と、その文書間整合性です。

レビュー対象として確認した主なファイルは次です。

- `spanner-query-gen-plan-contract-round12-review-response-ja.md`
- `PLAN_CONTRACTS(7).md`
- `PLAN_CONTRACT_CANDIDATES(7).md`
- `QUERY_EXECUTION_OPERATORS_OBSERVATIONS(6).md`
- `spanner-query-gen.plan-contracts.v1alpha.schema(8).json`
- `spanner-query-gen.plan-report.v1alpha.schema(11).json`
- `IMPLEMENTATION_STATUS(3).md`
- `DESIGN(30).md`
- `README(32).md` ただしこれは main README ではなく `tools/spanner-query-plan-shape` 用 README に見えます。

## 総評

今回の対応はかなり良いです。round 12 で指摘した `subtree_family_counts` の complete map 化、topology fields の schema-required 化、`matched_operator_indexes` の emission rule 固定、`forbid[n]` の source semantics、`contract_summary` derived counts の invariant 化は、CI artifact としての読みやすさを大きく上げています。

特に、次の判断は妥当です。

- `operator_family_counts` と `operators[].subtree_family_counts` をどちらも zero-filled complete map に揃えたこと。
- `operators[].child_indexes` / `descendant_indexes` / `subtree_family_counts` を normalized plan surface として必須化したこと。
- `forbid_operator_family` では常に `matched_operator_indexes` を出し、`cel` では v1alpha では出さないとしたこと。
- `source: forbid[n]` を元 YAML の `forbid` array の 0-based index と明文化したこと。
- Sort family を `full_sort` / `minor_sort` に分けつつ、既存の broad policy として `explicit_sort` を残したこと。
- `PLAN_CONTRACTS.md` が、plan contracts を main config とは分離された optional PLAN-only workflow として説明し続けていること。

大きな方向転換は不要です。次に詰めるべきなのは、新しい predefined contract 追加ではなく、**virtual / umbrella family の semantics** と **rule-specific schema の締め** です。

## P0: `explicit_sort` の virtual family semantics を明文化した方がよい

今回、`Sort` / `Sort Limit` を `full_sort`、`Minor Sort` / `Minor Sort Limit` を `minor_sort`、その umbrella count を `explicit_sort` とする整理になっています。この方向自体は良いです。

ただし、ここで `explicit_sort` は **concrete operator family** なのか、**derived / virtual count family** なのかが少し曖昧です。

`normalized_operators[].family` は 1 node に 1 family を持つ形に見えます。その場合、実際の sort node は `full_sort` または `minor_sort` であり、同じ node が同時に `explicit_sort` という family を持つわけではないはずです。一方で、`operator_family_counts` と `subtree_family_counts` には `explicit_sort` を入れる設計になっています。

このため、次の invariant を `PLAN_CONTRACTS.md` と DESIGN に明記した方がよいです。

```text
full_sort and minor_sort are concrete operator families.
explicit_sort is a derived / virtual umbrella family.

operator_family_counts["explicit_sort"]
  == operator_family_counts["full_sort"]
   + operator_family_counts["minor_sort"]

For every operator o:
  o.subtree_family_counts["explicit_sort"]
    == o.subtree_family_counts["full_sort"]
     + o.subtree_family_counts["minor_sort"]
```

また、`operator_families` についても方針を固定した方がよいです。

- `operator_families` に virtual family を含めるのか。
- それとも `normalized_operators[].family` に実際に現れた concrete family だけを含めるのか。

現在の説明では `operator_families` は “actually observed” な compact list とされています。そうであれば、`full_sort` / `minor_sort` は入るが `explicit_sort` は入らない、というのが自然です。ただし `operator_family_counts["explicit_sort"] > 0` になるため、list と counts の関係が直感に反する可能性があります。

推奨は次です。

```text
operator_families lists concrete families observed in normalized_operators[].family.
Derived families such as explicit_sort may appear in *_family_counts but not in normalized_operators[].family.
```

もし `explicit_sort` も `operator_families` に入れるなら、それは “observed concrete families” ではなく “non-zero count families, including derived families” なので、そのように名前または説明を変えた方が安全です。

## P0: `matched_operator_indexes` と virtual family の関係を固定した方がよい

`matched_operator_indexes` は `forbid_operator_family` の結果に常に出るようになり、これは良い改善です。

ただし、`operator_family: explicit_sort` のような virtual family を forbid した場合、`matched_operator_indexes` が何を指すかを明文化してください。

推奨 semantics は次です。

```text
For concrete families:
  matched_operator_indexes contains indexes of operators whose family equals operator_family.

For derived umbrella families:
  matched_operator_indexes contains indexes of concrete operators contributing to the derived count.

For explicit_sort:
  matched_operator_indexes contains full_sort and minor_sort operator indexes.
```

これを固定しておくと、`no_explicit_sort` の fail report を見たときに、実際には `full_sort` と `minor_sort` のどちらが原因なのかを `normalized_operators[]` から追えます。

さらに読みやすくするなら、future work として `matched_operator_families` か `matched_operator_summaries` を検討してもよいです。ただし v1alpha では `matched_operator_indexes` + `normalized_operators[].family` で十分です。

## P0: `contract_rule_result` schema を rule kind ごとにもう少し締めたい

最新 plan-report schema は、`forbid_operator_family` で `operator_family` / `observed_count` / `max_count` / `matched_operator_indexes` を required にし、`cel` で `expression` を required かつ `matched_operator_indexes` を forbidden にしており、かなり良くなっています。

ただし、schema 上はまだ次のような余分な field が通り得るように見えます。

```yaml
# rule: cel なのに operator-family fields が混ざる
rule: cel
source: cel
status: pass
expression: operators.size() > 0
operator_family: hash_join
observed_count: 0
max_count: 0

# rule: forbid_operator_family なのに CEL expression が混ざる
rule: forbid_operator_family
source: forbid[0]
status: pass
operator_family: hash_join
observed_count: 0
max_count: 0
matched_operator_indexes: []
expression: operators.size() > 0
```

v1alpha の artifact を CI contract として使うなら、ここは shape を閉じた方が良いです。

推奨は次です。

```text
rule: forbid_operator_family
  required:
    operator_family
    observed_count
    max_count
    matched_operator_indexes
  forbidden:
    expression

rule: cel
  required:
    expression
  forbidden:
    operator_family
    observed_count
    max_count
    matched_operator_indexes
```

`remediation` は fail 時に advisory として残してよいです。`failure_kind` も fail 時 required のままでよいです。

## P1: Sort / Spool の観測 evidence を `QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md` に同期したい

round 12 response では、ユーザー共有の実行計画により `Sort` と `Minor Sort` の意味が異なるため `full_sort` / `minor_sort` を分けた、と説明されています。また、CTE を複数箇所で参照するクエリで `SpoolBuild` / `SpoolScan` を確認したため、known family に追加したと説明されています。

一方、最新の `QUERY_EXECUTION_OPERATORS_OBSERVATIONS(6).md` を見る限り、`Sort` / `Sort Limit` や `Recursive Spool Scan` は出ていますが、`Minor Sort`、`Minor Sort Limit`、`SpoolBuild`、`SpoolScan` の観測例が明確には載っていないように見えます。

これは実装上の問題とは限りません。response には「ユーザー共有の実行計画」や「CTE 再現ケース」と書かれているので、観測源が docs-derived probe ではないだけかもしれません。

ただ、`QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md` は vocabulary evidence として使われているため、次のどちらかを入れるとよいです。

```text
## Additional non-doc local probes

- Minor Sort / Minor Sort Limit
  - source: user-shared plan or local synthetic probe
  - compact shape: ...
  - normalized family: minor_sort

- SpoolBuild / SpoolScan
  - source: repeated CTE probe
  - SQL: WITH CTE AS (...)
  - compact shape: ...
  - normalized families: spool_build, spool_scan
```

または、まだ再現 fixture が安定していないなら、`observed externally; pending fixture` のように明記するだけでもよいです。

## P1: `explicit_sort` を direct `forbid` で使う例はよいが、冗長例は避けたい

`PLAN_CONTRACT_CANDIDATES.md` の direct forbid example では、`explicit_sort`、`full_sort`、`minor_sort` を同じ contract に並べています。

```yaml
forbid:
- operator_family: explicit_sort
- operator_family: full_sort
- operator_family: minor_sort
```

`explicit_sort` が `full_sort + minor_sort` の umbrella なら、この 3 つを同時に書くのは冗長です。候補ドキュメントでは “複数指定できる” ことを見せたいのだと思いますが、読者は「3 つを全部書くのが推奨なのか」と誤解する可能性があります。

例は次のように分けた方が分かりやすいです。

```yaml
# broad policy: reject both full and minor sorts
forbid:
- operator_family: explicit_sort
```

```yaml
# narrower policy: reject only full/global sorts
forbid:
- operator_family: full_sort
```

```yaml
# specialized policy: reject minor sorts too
forbid:
- operator_family: minor_sort
```

## P1: `no_apply_join` の broad/narrow hierarchy は良いが、narrow predefined の穴を明記したい

現在の `no_apply_join` は standalone apply-family operators と distributed apply-family wrappers を禁止する broad policy になっています。これは直感的です。

一方で、narrow predefined は `no_distributed_cross_apply` だけがあり、`distributed_semi_apply` / `distributed_anti_semi_apply` 専用の predefined はありません。これは v1alpha で predefined を増やしすぎない判断として妥当です。

ただし、docs には次の一文を入れるとよいです。

```text
There are no dedicated no_distributed_semi_apply or no_distributed_anti_semi_apply predefined contracts in v1alpha. Use direct forbid.operator_family for those narrower policies.
```

例:

```yaml
contracts:
- name: NoDistributedSemiApplyOnly
  target: query/Foo
  forbid:
  - operator_family: distributed_semi_apply
  - operator_family: distributed_anti_semi_apply
```

これにより、predefined set を増やさずにユーザーの逃げ道を明確にできます。

## P1: `IMPLEMENTATION_STATUS.md` の “intentionally small” は言い換えてよい

`IMPLEMENTATION_STATUS(3).md` には、normalized operator-family catalog は intentionally small であり、real plans / contracts に応じて grow するとあります。

初期方針としては正しいですが、現在の schema enum はすでにかなり広く、さらに `explicit_sort` のような derived family も入っています。そのため “small” という語はやや実態とずれてきています。

言い換え案です。

```text
The normalized operator-family catalog is bounded, registry-driven, and evidence-based. It grows only when observed plans, fixtures, or contract use cases require a new family.
```

これなら、今後 family が増えても方針がぶれません。

## P1: backend identity の `not_recorded` と observations の environment evidence を接続したい

`IMPLEMENTATION_STATUS.md` では、plan report の backend identity は現在 Omni version / image digest を `not_recorded` としている、と説明されています。一方、`QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md` には Spanner Omni image digest、spanemuboost、spannerplan、Go version などがしっかり記録されています。

この分離は理解できます。observations は developer probe の evidence で、plan-report artifact は別の workflow です。

ただ、読者が混乱しないよう、次のように明記するとよいです。

```text
Developer observation documents may record the Omni image digest and tool versions manually.
The public plan-report artifact currently records backend.version and backend.image_digest as not_recorded until the plan-report backend source split is wired into the report.
```

## P2: `source` / `predefined` equality を runtime test に寄せる判断は妥当

`source: use/<name>` と `predefined: <name>` の equality を JSON Schema だけで完全表現しない判断は妥当です。大量の `if/then` を schema に入れるより、schema は値域と presence / absence を固定し、equality は runtime validation / regression test で固定する方が読みやすいです。

この点はこれ以上厳しくしなくてよいと思います。

## P2: PROFILE / execution stats を対象外にし続ける判断も妥当

PLAN-only の境界は維持でよいです。`EXPLAIN ANALYZE` / PROFILE / rows scanned / latency / CPU を入れると、data distribution と runtime environment の影響が一気に強くなり、今の structural review contract とは別物になります。

現時点では、PLAN-only、optional Omni-backed、production performance guarantee ではない、という線引きを維持するのが正しいです。

## まとめ

今回の状態はかなり良いです。大きな仕様変更は不要です。

次にやるなら、優先順位は次です。

1. `explicit_sort` を virtual / derived family として明文化する。
2. `operator_family_counts` / `subtree_family_counts` / `operator_families` / `normalized_operators[].family` の関係を固定する。
3. `matched_operator_indexes` が virtual family で何を返すかを明記する。
4. `contract_rule_result` schema で `forbid_operator_family` と `cel` の余分な field を相互に禁止する。
5. `Minor Sort` / `SpoolBuild` / `SpoolScan` の観測 evidence を observations document に同期する。
6. direct forbid examples から `explicit_sort + full_sort + minor_sort` の冗長な同時指定を外す。
7. narrow distributed semi/anti apply policy は predefined 追加ではなく direct forbid で書ける、と docs に明記する。

ここまで締めれば、plan contract 周りは v1alpha public artifact としてかなりレビューしやすい状態になると思います。
