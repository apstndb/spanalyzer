# `spanner-query-gen` plan contract round 10 review

作成日: 2026-05-06

## 前提

今回も、実装コード・実行ログ・CI output は共有されていない前提で、添付された response / README / DESIGN / `PLAN_CONTRACTS.md` / `PLAN_CONTRACT_CANDIDATES.md` / `QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md` / JSON Schema から見える **public contract と文書整合性** をレビューします。

今回の更新は、大きな方向転換ではなく、前回指摘した schema / docs / normalizer / fixture の同期をかなり丁寧に進めたものとして評価します。

## 総評

全体として、かなり良い状態です。

特に次の点は、このまま進めてよいと思います。

- `Distributed Semi Apply` と `Distributed Anti Semi Apply` を distinct family に分けたこと。
- `no_apply_join` を broad policy、`no_standalone_apply_join` を narrow policy として分離したこと。
- `no_hash_join` と `no_standalone_hash_join` の意味を、Push Broadcast Hash Join の wrapper / internal implementation node と切り分けたこと。
- `operator_family` enum を plan-contract schema と plan-report schema で一致させる unit test を入れたこと。
- `contract_rule_result.source` / `predefined` を report artifact に出す方針にしたこと。
- file-based plan fixtures で Push Broadcast / Semi Apply / Distributed Apply 系の contract semantics を固定したこと。
- `tools/spanner-query-plan-shape` を public CLI ではなく developer probe と明示し、Spanner Omni execution-plan support の Preview / Pre-GA caveat を入れたこと。

このあたりは、`plan-report` を optional experimental workflow としつつ、CI/review artifact として使えるものに近づけるための正しい整理です。

## P0: `PLAN_CONTRACTS.md` の report example が schema-valid ではない

一番大きい残りの問題はこれです。

`PLAN_CONTRACTS.md` の “Minimal Report Outcomes” には pass / fail / target_not_found の例がありますが、現在の plan-report schema を見る限り、`contract_evaluation` には常に `stability` が required です。また `status: pass | fail` の場合は `query` / `scope` / `results` も required です。

しかし、`PLAN_CONTRACTS.md` の例では `contract_evaluations[]` に `stability` がなく、pass / fail 例でも `query` / `scope` がありません。

今の文書は “最小例” に見えるので、次のどちらかに寄せた方がよいです。

### 案 A: schema-valid な最小例にする

```yaml
contract_evaluation_mode: check
contract_summary:
  status: pass
  contracts: 1
  passed: 1
  failed: 0
  not_evaluated: 0
contract_evaluations:
- name: NoExplicitSort
  target_id: query/ScanSingerIDsFast
  query: ScanSingerIDsFast
  scope: query
  status: pass
  stability:
    tier: normalized_operator
    check_recommended: true
  results:
  - rule: forbid_operator_family
    source: use/no_explicit_sort
    predefined: no_explicit_sort
    operator_family: explicit_sort
    status: pass
    observed_count: 0
    max_count: 0
```

`not_evaluated` 例にも `stability` を入れる必要があります。

```yaml
contract_evaluations:
- name: MissingTarget
  target_id: query/MissingTarget
  status: not_evaluated
  reason: target_not_found
  stability:
    tier: normalized_operator
    check_recommended: true
```

### 案 B: 例を明示的に abridged にする

```text
The following snippets are abridged and omit required report fields such as
stability, query, and scope.
```

ただし、CI artifact としての信頼性を高めるなら、案 A の方がよいです。さらに、docs 内 YAML snippets を schema validate する test を追加できると、今後の drift を減らせます。

## P0: `contract_rule_result.source` は schema 上まだ緩い

今回、`contract_rule_result.source` を required にしたのは良いです。ただし、plan-report schema の current shape では、`source` 自体の property definition が `minLength` 程度に見えます。

そのため、次のような report fragment が schema 上は通り得ます。

```yaml
rule: forbid_operator_family
source: bogus
status: pass
operator_family: hash_join
observed_count: 0
max_count: 0
```

また、次のように `source` と `predefined` が矛盾していても schema 上は通り得ます。

```yaml
rule: forbid_operator_family
source: use/no_hash_join
predefined: no_standalone_hash_join
status: pass
operator_family: hash_join
observed_count: 0
max_count: 0
```

少なくとも次の制約を入れると、artifact schema としてかなり強くなります。

```json
"source": {
  "type": "string",
  "pattern": "^(use/(no_explicit_sort|no_hash_join|no_standalone_hash_join|no_push_broadcast_hash_join|no_apply_join|no_standalone_apply_join|no_distributed_cross_apply|no_merge_join|no_hash_aggregate|no_stream_aggregate)|forbid\\[[0-9]+\\]|cel)$"
}
```

さらに `predefined` も `minLength` ではなく predefined contract enum にするのがよいです。

```json
"predefined": {
  "enum": [
    "no_explicit_sort",
    "no_hash_join",
    "no_standalone_hash_join",
    "no_push_broadcast_hash_join",
    "no_apply_join",
    "no_standalone_apply_join",
    "no_distributed_cross_apply",
    "no_merge_join",
    "no_hash_aggregate",
    "no_stream_aggregate"
  ]
}
```

`source: use/<name>` と `predefined: <name>` が一致することまで JSON Schema で表すのは少し面倒ですが、少なくとも unit test で固定した方がよいです。

## P1: `no_apply_join` と Push Broadcast Hash Join Semi/Anti の関係を明文化する

今回の observation では、`Push Broadcast Hash Join Semi Apply` と `Push Broadcast Hash Join Anti Semi Apply` が `push_broadcast_hash_join` family に fold される形になっています。一方、`no_apply_join` は standalone apply-family と distributed apply-family wrapper を禁止する policy です。

この整理自体は妥当ですが、ユーザーの直感とは少しズレる可能性があります。

たとえば、ユーザーが “semi apply 系を避けたい” という気持ちで `no_apply_join` を指定した場合、`Push Broadcast Hash Join Semi Apply` は名前に `Semi Apply` を含むものの、normalized family としては `push_broadcast_hash_join` なので、`no_apply_join` では止まりません。

これは悪い設計ではありません。むしろ wrapper family を優先するのは reasonable です。ただし、`PLAN_CONTRACTS.md` に次のような注意を追加した方が安全です。

```text
Push Broadcast Hash Join Semi Apply and Push Broadcast Hash Join Anti Semi Apply
are normalized as push_broadcast_hash_join, not as semi_apply / anti_semi_apply.
Therefore no_apply_join does not reject them. Use no_push_broadcast_hash_join
or no_hash_join when those wrappers should be rejected.
```

あるいは、意図別の表を入れると使いやすいです。

| Intent | Recommended contract |
|---|---|
| 明示 Sort を避けたい | `no_explicit_sort` |
| standalone hash join と push-broadcast hash wrapper を避けたい | `no_hash_join` |
| standalone hash join だけを避けたい | `no_standalone_hash_join` |
| standalone / distributed apply-family を避けたい | `no_apply_join` |
| Push Broadcast Hash Join wrapper も避けたい | `no_push_broadcast_hash_join` または `no_hash_join` |

## P1: `PLAN_CONTRACT_CANDIDATES.md` の predefined 説明を少し修正する

`PLAN_CONTRACT_CANDIDATES.md` では “Each one is a small alias for forbidding a normalized operator family” と説明されていますが、現状の `no_hash_join` や `no_apply_join` は 1 つの family ではなく、複数 family へ展開される policy です。

細かいですが、次のように直すと正確です。

```text
Each predefined contract is a small alias for forbidding one or more normalized
operator families.
```

これは、`source: use/<name>` / `predefined: <name>` を report に出す設計とも整合します。

## P1: `unknown` family の扱いを docs に入れる

plan-report schema の `operator_family` enum には `unknown` が含まれています。これは良い escape hatch ですが、contract behavior と reviewer UX を明示しておくとよいです。

おすすめは次です。

```text
unknown appears when the normalizer cannot classify a PlanNode display name.
It is included in operator_family_counts and classification_warnings. Predefined
contracts do not fail only because unknown is present, but users can explicitly
forbid unknown with a direct forbid rule if they want strict classification.
```

`classification_warnings` に必ず出るなら、`operator_family_counts.unknown > 0` を見るだけでなく、人間にも気づきやすくなります。

## P1: observation docs は fixture/source-of-truth との関係を明確にする

`QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md` はかなり有用です。Spanner Omni image digest、`spanemuboost`、`spannerplan`、Go version が記録されている点も良いです。

ただし、optimizer version が `not_pinned` なので、この文書は “stable proof” というより “observed vocabulary evidence” として扱うのが安全です。

おすすめは、文書冒頭に次のような位置づけを入れることです。

```text
This document records observed operator vocabulary and compact shapes for
normalizer design. Golden fixtures, schemas, and unit tests are the normative
contract for v1alpha behavior.
```

今の方針でも unit test / fixtures を追加しているとのことなので、この一文があると docs の読み方がはっきりします。

## P2: `forbid.max_count` に schema default を入れる

`PLAN_CONTRACTS.md` には `max_count` 省略時は `0` と説明されています。JSON Schema にも editor hint として `default: 0` を入れてよいと思います。

```json
"max_count": {
  "type": "integer",
  "minimum": 0,
  "default": 0
}
```

validation そのものには影響しませんが、public schema を editor integration に使う方針と相性が良いです。

## P2: contract status と rule result status の不変条件を docs/test にする

`contract_evaluation.status` と `contract_rule_result.status` の関係は schema だけでは完全には表しにくいです。

次の invariant は docs または tests で固定するとよいです。

```text
contract_evaluation.status == pass iff every rule result status is pass.
contract_evaluation.status == fail iff at least one rule result status is fail.
contract_evaluation.status == not_evaluated iff rule results are absent.
contract_summary.status == pass iff all contract evaluations are pass.
contract_summary.status == fail iff any contract evaluation is fail or not_evaluated.
```

`PLAN_CONTRACTS.md` には `contract_summary.status` と process exit の違いが既に書かれているので、この invariant を足すだけで十分です。

## 結論

大きな設計変更は不要です。今回の修正で、plan contract の主要な方向性はかなり固まっています。

次に優先すべきなのは、新しい predefined contract を増やすことではなく、次の小さな同期です。

1. `PLAN_CONTRACTS.md` の report examples を schema-valid にする。
2. `contract_rule_result.source` と `predefined` を schema 上もっと締める。
3. `no_apply_join` と `push_broadcast_hash_join` 系の関係を明文化する。
4. `PLAN_CONTRACT_CANDIDATES.md` の “one family” 表現を “one or more families” に直す。
5. `unknown` family と `classification_warnings` の扱いを docs に入れる。
6. observation docs を “observed vocabulary evidence” として位置づける。

この段階まで来ると、plan contract は “experimental だが review artifact として読める” ところまでかなり近づいています。
