# `spanner-query-gen` plan contract round 12 review

作成日: 2026-05-06

## 前提

このレビューは、共有された Markdown / JSON Schema / 補助ドキュメントから見える public contract のレビューです。実装コード、実行ログ、CI output は共有されていないため、`go test ./...` や追加済み unit test への言及は「開発側の内部作業ツリー上の主張」として扱い、ここでは外部レビュー可能な仕様・schema・docs の整合性だけを見ます。

今回確認した主なファイルは以下です。

- `spanner-query-gen-plan-contract-round11-review-response-ja.md`
- `spanner-query-gen.plan-report.v1alpha.schema(10).json`
- `spanner-query-gen.plan-contracts.v1alpha.schema(7).json`
- `PLAN_CONTRACTS(6).md`
- `PLAN_CONTRACT_CANDIDATES(6).md`
- `QUERY_EXECUTION_OPERATORS_OBSERVATIONS(5).md`
- `README(31).md`

## 総評

かなり良くなっています。round 11 で指摘した主な点、つまり `operator_family_counts` の zero-filled complete map 化、`matched_operator_indexes` の追加、`contracts[].name` 重複の diagnostic ID、`target_id` / `scope` / `backend` の canonical 化、`source` / `predefined` invariant の明文化は、方向として妥当です。

特に `operator_family_counts` を complete map にした判断は良いです。CEL で `operator_family_counts["hash_join"] == 0` のような式を書けることは、plan contract を CI で使う場合にかなり効きます。artifact size は増えますが、top-level map であれば許容できます。

また、`matched_operator_indexes` はとても良い追加です。`no_explicit_sort` や `no_hash_join` が fail したときに、どの `normalized_operators[].index` が原因なのかを report だけで追えるようになります。これは新しい contract surface を増やすよりも価値が高い改善です。

大きな方向転換は不要です。残る論点は、ほぼ artifact schema と CEL 入力の最後の厳密化です。

## P0: `subtree_family_counts` の map semantics がまだ弱い

今回、top-level の `operator_family_counts` は zero-filled complete map として整理されました。一方で、`PLAN_CONTRACTS.md` と `PLAN_CONTRACT_CANDIDATES.md` には次のような CEL example が残っています。

```cel
operators.exists(o,
  o.family == "hash_aggregate" &&
  o.subtree_family_counts["scan"] == 1)
```

これは便利ですが、現在の plan-report schema 上は `operators[].subtree_family_counts` が top-level `operator_family_counts` と同じ complete map として定義されているようには見えません。`operator_family_counts` については「全 known family を required にする」という contract ができていますが、per-operator の `subtree_family_counts` は optional field かつ sparse map に見えます。

この状態だと、top-level map は安全に direct indexing できる一方、subtree map は direct indexing してよいのかが曖昧です。CEL の map indexing が missing key に対してどう振る舞うかに contract が依存すると、せっかく zero-filled にした設計思想が半分だけ残ります。

おすすめは、次のどちらかです。

### 案 A: `subtree_family_counts` も complete map にする

public report field と CEL input の両方で、各 `operator.subtree_family_counts` も全 `operator_family` enum value を持つ complete map にします。operator 数 × family 数だけ artifact は増えますが、v1alpha の CI contract としては最も分かりやすいです。

その場合、`operator` schema でも以下を required に寄せるのが自然です。

```text
operators[].child_indexes
operators[].descendant_indexes
operators[].subtree_family_counts
```

`child_indexes` / `descendant_indexes` は空配列を許せばよく、`subtree_family_counts` は zero-filled complete map にします。

### 案 B: `subtree_family_counts` は sparse と明記し、example を変える

artifact size を優先して sparse map にするなら、docs の example から direct indexing を外すべきです。例えば、CEL 側に safe access がないなら、`operator_family_counts` だけを direct indexing example に残し、subtree example は削るか、別の normalized field が入るまで future candidate に戻します。

私の推奨は案 A です。すでに top-level counts を complete map にした以上、subtree だけ sparse にするより、少し artifact を大きくしてでも CEL 入力の直感を揃えた方がよいです。

## P1: `matched_operator_indexes` の emission rule を固定したい

`matched_operator_indexes` は非常に良い追加ですが、optional field としての扱いをもう少し固定した方がよいです。

おすすめは次です。

```text
rule: forbid_operator_family の場合:
  matched_operator_indexes は常に出す
  observed_count == 0 なら []
  observed_count > 0 なら family に一致した normalized_operators[].index を全件出す
  配列は昇順、重複なし

rule: cel の場合:
  v1alpha では matched_operator_indexes を出さない
```

schema description は「operator-family rule の matched indexes」となっているため、`rule: cel` でこの field が出ると意味が曖昧になります。将来、CEL rule が明示的に matched node を返す仕組みを入れるなら別ですが、v1alpha では `cel` に対して `matched_operator_indexes` を forbidden にするのが simple です。

また、`matched_operator_indexes` が optional のままだと、`observed_count > 0` なのに indexes が出ない artifact も schema 上は通ってしまいます。CI artifact としては、空配列を含めて常に出す方がレビューしやすいです。

## P1: `operators[]` の topology fields は schema で required にするか、docs で optional と書く

`PLAN_CONTRACTS.md` は normalized topology fields として以下を説明しています。

```text
operator_edges
operators[].child_indexes
operators[].descendant_indexes
operators[].subtree_family_counts
```

`operator_edges` は query status `ok` で required になっています。一方で、schema 上の `operator` required fields は `index` / `display_name` / `family` に見えます。このままだと、docs は topology fields を current CEL surface として紹介しているのに、schema はそれらの存在を保証していません。

v1alpha の public report contract としては、次のどちらかに寄せるべきです。

- topology fields を stable normalized view とするなら、`operator` schema で `child_indexes` / `descendant_indexes` / `subtree_family_counts` を required にする。
- まだ experimental なら、docs で “may be present” と書き、CEL examples から topology-dependent examples を外す。

今の方向性なら前者がよいと思います。plan contract はすでに optional / experimental workflow ですが、その中で出す report schema は CI artifact として強くしてよいです。

## P1: `source` / `predefined` equality は schema か test で固定する

`source` / `predefined` invariant の明文化は良いです。

```text
source: use/<name> なら predefined: <name>
source: forbid[n] または cel なら predefined は absent
```

ただし JSON Schema だけを見ると、`source: use/no_hash_join` と `predefined: no_explicit_sort` のような不一致を完全に排除するのは難しいです。現在は pattern と enum で値域をかなり狭めていますが、値同士の equality は runtime validation または generated per-name `if/then` が必要です。

これは必ずしも schema だけで解決しなくてよいです。むしろ次のような分担で十分です。

```text
JSON Schema:
  source / predefined の値域と presence/absence を制約する

runtime validation / golden tests:
  source use/<name> と predefined <name> の一致を保証する
```

ただし、`PLAN_CONTRACTS.md` に invariant を書いたのであれば、その invariant を固定する golden fixture は必須にした方がよいです。

## P1: `forbid[n]` の index semantics を明記する

Direct `forbid` entries は `source: forbid[0]` のように report されます。これは良いですが、`n` が何に対応するのかを明記した方がよいです。

おすすめの定義は次です。

```text
forbid[n] は、contract file の該当 contract に書かれた forbid array の 0-based YAML order index を指す。
正規化や expansion 後の index ではない。
```

`use/<name>` と同様、`source` はユーザーが元 YAML に戻るための field なので、元 YAML の順序に対応している方が自然です。

## P2: `operator_family_counts` の output ordering を固定すると snapshot review が楽になる

JSON object の順序は意味を持ちませんが、`--stable` output や Markdown/YAML artifact のレビューでは ordering が効きます。`operator_family_counts` が complete map になったので、出力順は normalizer registry order または alphabetic order に固定しておくと、golden diff がかなり読みやすくなります。

これは schema で表現する話ではなく、renderer / golden fixture の contract です。

## P2: `contract_summary` の derived counts は invariant として書いておくとよい

`contract_summary.status` と process exit の分離はすでに良く書けています。さらに、derived counts の invariant も明記しておくと CI reader に親切です。

```text
contract_summary.contracts == len(contract_evaluations)
contract_summary.passed == count(status == pass)
contract_summary.failed == count(status == fail)
contract_summary.not_evaluated == count(status == not_evaluated)
```

この invariant は schema では表現しにくいので、runtime validation / golden tests で固定するのがよいです。

## P2: candidate docs の CEL examples は “stable examples” と “experimental examples” を分ける

`PLAN_CONTRACT_CANDIDATES.md` は候補集としてかなり有用です。一方で、CEL examples に `subtree_family_counts` のような topology-dependent field が入っているため、ユーザーが「これは今すぐ安定して使える」と読む可能性があります。

`Current Surface Summary` の中では、次のように分けるとよいです。

```text
Stable normalized CEL examples:
  operator_family_counts[...]
  operators.exists(o, o.scan_target == ...)
  operators.all(o, !(o.family == "scan" && o.full_scan))

Topology examples:
  operators[].subtree_family_counts[...] など
  topology fields が schema-required / zero-filled になってから stable examples に昇格
```

もし `subtree_family_counts` を complete map + required にするなら、この分離は不要です。

## P2: Spanner Omni probe README の caveat は良い

`README(31).md` が `tools/` 配下の developer probe として、Spanner Omni execution-plan support の Preview / Pre-GA caveat を明記しているのは良いです。この wording は `plan-report` の optional workflow という位置づけと整合しています。

外部 docs 上も、Spanner Omni execution plans は Preview / Pre-GA として説明され、`EXPLAIN QUERY` と runtime statistics を含む `EXPLAIN ANALYZE QUERY` が区別されています。したがって、v1alpha plan contract が PLAN-only で PROFILE / execution stats を拒否する方針は維持でよいです。

## まとめ

今回の更新で plan contract surface はかなり安定してきました。大きな設計変更や predefined contract の追加は不要です。

次に詰めるべき優先順位は次です。

1. `subtree_family_counts` を complete map にするか、sparse と明記して CEL example を変える。
2. `operators[].child_indexes` / `descendant_indexes` / `subtree_family_counts` を schema-required にするか、docs 上で optional と明記する。
3. `matched_operator_indexes` を `forbid_operator_family` で常時出すか、少なくとも `observed_count > 0` で必須にする。
4. `matched_operator_indexes` を `rule: cel` では v1alpha で禁止する。
5. `source` / `predefined` equality と `forbid[n]` index semantics を runtime tests / golden fixtures で固定する。
6. `operator_family_counts` の renderer ordering と `contract_summary` derived counts の invariant を固定する。

この段階では、機能を増やすより、report artifact の “読めば原因を追える” 性質をさらに締めるのが一番価値があります。
