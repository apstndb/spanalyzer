# `spanner-query-gen` plan contract round 9 review

作成日: 2026-05-06

## 前提

レビュー対象は、今回共有された以下の Markdown / schema / 観測メモです。実装コード、実行ログ、CI の実出力は共有されていないため、ここでの評価は public contract、README / DESIGN、schema、補助ドキュメントの整合性に限定します。

今回の中心は `plan-report` / plan contract です。main `spanner-query-gen.yaml` の単純さを維持しつつ、Spanner Omni を使った optional な review artifact として plan contract を育てる方針は維持でよいです。

## 総評

今回の対応はかなり良いです。特に以下は前回レビューの懸念をよく閉じています。

- `PLAN_CONTRACT_CANDIDATES.md` を normative な contract surface ではなく候補集に戻し、operator family enum の source of truth を schema 側に寄せた点。
- `no_explicit_sort` を `display_name` の substring match ではなく normalized `explicit_sort` family への semantic matching と明記した点。
- `contract_rule_result` に `source` / `predefined` を追加し、`use: [no_hash_join]` のような predefined contract 展開結果を追えるようにした点。
- `contract_summary.status` を contract truth、process exit を `report_only` / `check` mode によって決まるものとして分けた点。
- `QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md` に Spanner Omni image digest、`spanemuboost` / `spannerplan` version、Go version、optimizer pinning 状態を入れた点。
- `IN` / `EXISTS` / `NOT IN` / `NOT EXISTS` の plan shape を観測し、`Semi Apply` / `Anti-Semi Apply` 系を apply/hash/push-broadcast predicate から漏れないようにした点。

特に最後の点は重要です。`EXISTS` が `Semi Apply`、`NOT EXISTS` が `Anti-Semi Apply` になり得ることを見落とすと、`no_apply_join` のような利用者が直感的に期待する contract が実際には穴だらけになります。今回の broad apply-family 化は妥当です。

## P0: schema / docs の同期をもう一度確認したい

### 1. 添付された schema が最新なら、operator family enum が response と矛盾しています

今回の response では、`semi_apply`、`distributed_semi_apply`、`distributed_semi_apply_internal_apply` を operator family に追加したと説明されています。また、`QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md` でも `Semi Apply` / `Distributed Semi Apply` / `Distributed Anti Semi Apply` が観測され、normalization impact に `semi_apply`、`distributed_semi_apply`、`distributed_semi_apply_internal_apply` が列挙されています。

ただし、こちらで見えている直近の `spanner-query-gen.plan-contracts.v1alpha.schema(4).json` / `spanner-query-gen.plan-report.v1alpha.schema(6).json` の `operator_family` enum には、少なくとも以下が見当たりませんでした。

```text
semi_apply
distributed_semi_apply
distributed_semi_apply_internal_apply
```

一方で `anti_semi_apply` は入っているように見えます。

今回のターンでは schema が新規添付されていないようなので、これは「手元にある schema が古いだけ」の可能性があります。ただし response は “checked-in schema files を再生成した” と書いているため、次のいずれかを確実にした方がよいです。

```text
- 最新の plan-contract schema / plan-report schema を共有する
- operator family enum に観測済み normalized family がすべて入っていることを確認する
- plan-contract schema と plan-report schema の operator_family enum が完全一致する test を追加する
- QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md の Normalization Impact に出る family が schema enum に存在する test を追加する
```

この同期は P0 です。plan contract は CI artifact なので、schema が stale だと downstream tooling が一番困ります。

### 2. `PLAN_CONTRACTS.md` も semi/anti apply 版に同期されているか確認したい

今回添付範囲では最新 `PLAN_CONTRACTS.md` は見えていないようです。手元にある `PLAN_CONTRACTS(1).md` では、`no_apply_join` が standalone `apply_join` と `distributed_cross_apply` wrapper を禁止する説明に見え、`semi_apply` / `anti_semi_apply` / `distributed_semi_apply` まではまだ反映されていないように読めます。

response の方針に合わせるなら、`PLAN_CONTRACTS.md` の predefined contract 説明は次のように明確にした方がよいです。

```text
no_apply_join:
  forbids standalone apply-family operators:
    - apply_join
    - semi_apply
    - anti_semi_apply
  and distributed apply-family wrappers:
    - distributed_cross_apply
    - distributed_semi_apply
    - distributed_anti_semi_apply, if modeled separately

no_standalone_apply_join:
  forbids only standalone apply-family operators:
    - apply_join
    - semi_apply
    - anti_semi_apply
```

もし `distributed_anti_semi_apply` を `distributed_semi_apply` に fold するなら、その fold も明記してください。

### 3. `distributed_semi_apply` が anti-semi variant も含むのか、別 family にするのか決めたい

観測メモでは `Distributed Semi Apply` と `Distributed Anti Semi Apply` の両方が出ています。一方、normalization impact には `distributed_semi_apply` と `distributed_semi_apply_internal_apply` があり、`distributed_anti_semi_apply` はありません。

これは設計としてどちらでも可能ですが、名前の直感は重要です。

選択肢 A: distinct family にする。

```text
distributed_semi_apply
distributed_anti_semi_apply
distributed_semi_apply_internal_apply
distributed_anti_semi_apply_internal_apply
```

選択肢 B: broad family として fold する。

```text
distributed_semi_apply  # includes Distributed Semi Apply and Distributed Anti Semi Apply
distributed_semi_apply_internal_apply  # includes internal Semi/Anti-Semi implementation nodes
```

v1alpha で simple にするなら B でもよいですが、その場合は名前から anti-semi も含むことが読み取りづらいです。`semi_apply` と `anti_semi_apply` を standalone では分けているため、distributed だけ fold するなら理由を `PLAN_CONTRACTS.md` に書くのが安全です。

私の好みは A です。理由は、operator family は report artifact の vocabulary なので、観測名に近い方が後で CEL や direct `forbid` を書く利用者に優しいからです。predefined `no_apply_join` は両方をまとめて禁止すればよく、family 自体まで潰す必要はありません。

## P1: plan contract artifact と補助ドキュメントの安定性

### 4. `spanner-query-plan-shape` README にも Preview / Pre-GA caveat を短く入れる

`PLAN_CONTRACTS.md` に Spanner Omni execution-plan support の Preview / Pre-GA caveat を入れたのは良いです。同じ caveat は `tools/spanner-query-plan-shape` の README にも短く入れるとよいです。

この tool は public CLI ではなく developer probe ですが、README 単体で読まれる可能性があります。現在の tool README は Docker と Spanner Omni image requirement を説明していますが、Spanner Omni execution plans の Preview / Pre-GA 性質には触れていないように見えます。

提案文:

```text
This tool depends on Spanner Omni execution-plan support, which is Preview / Pre-GA in the official Spanner Omni documentation. Use it for design review, testing, prototyping, and normalization experiments, not as a production performance guarantee.
```

### 5. `contract_rule_result.source` は良いが、schema でも必須化したい

response では `contract_rule_result` に以下を追加したと説明されています。

```text
source: use/no_hash_join
predefined: no_hash_join
source: forbid[0]
```

これは良いです。predefined contract が複数 rule に展開されるとき、どの predefined rule 由来の violation なのかを追えるようになります。

ただし、schema 側でも次を固定した方がよいです。

```text
contract_rule_result.source: required
contract_rule_result.predefined: required when source starts with use/
contract_rule_result.predefined: forbidden for direct forbid/cel results
```

文字列 prefix を downstream が parse するのが嫌なら、将来はこうしてもよいです。

```yaml
source:
  kind: predefined
  name: no_hash_join
  expanded_index: 0
```

ただ、v1alpha の simple 方針なら、まずは `source` string + optional `predefined` で十分です。重要なのは report schema がそれを保証することです。

### 6. `no_hash_join` と push-broadcast internal hash join の fixture は最優先で固定したい

response では、file-based golden fixture はまだ未完了で、synthetic `spannerpb.QueryPlan` unit test と Omni probe で coverage を増やしたとあります。この判断は理解できます。

ただし、次に fixture 化するなら、最優先は Push Broadcast 系だと思います。理由は、`no_hash_join` が broad contract になった一方で、Push Broadcast wrapper 内部の implementation `Hash Join` は `push_broadcast_hash_join_internal_hash_join` として扱い、standalone `hash_join` とは分ける必要があるためです。

最低限、以下の fixture があると安心です。

```text
1. standalone Hash Join
   -> hash_join
   -> no_hash_join fails
   -> no_standalone_hash_join fails

2. Push Broadcast Hash Join with internal Hash Join
   -> push_broadcast_hash_join + push_broadcast_hash_join_internal_hash_join
   -> no_hash_join fails due to wrapper
   -> no_standalone_hash_join passes

3. real nested Hash Join under broadcast side input, not pushed-batch internal
   -> hash_join remains hash_join
   -> no_standalone_hash_join fails
```

3 が重要です。すべての `Map` descendant Hash Join を internal 扱いしてしまうと、利用者が避けたい本物の Hash Join を見逃す可能性があります。DESIGN が “non-branching relational path from Map child and consuming a pushed BatchScan input” と条件を狭くしているのは良いので、その条件を fixture で守りたいです。

### 7. `no_apply_join` も internal implementation と user-visible apply を分ける fixture が欲しい

`no_apply_join` が `semi_apply` / `anti_semi_apply` まで広がったことで、こちらも fixture 価値が上がりました。

推奨 fixture:

```text
1. EXISTS -> Semi Apply
   -> semi_apply
   -> no_apply_join fails
   -> no_standalone_apply_join fails

2. NOT EXISTS -> Anti-Semi Apply
   -> anti_semi_apply
   -> no_apply_join fails
   -> no_standalone_apply_join fails

3. BATCH_MODE=TRUE -> Distributed Semi Apply with internal Semi Apply
   -> distributed_semi_apply + distributed_semi_apply_internal_apply
   -> no_apply_join fails due to wrapper
   -> no_standalone_apply_join passes
```

この fixture があると、今回追加した subquery join hint matrix が plan contract semantics に正しくつながっていることを外部 reviewer も理解しやすくなります。

### 8. 観測メモは “evidence” として良いが、normative contract ではないことを維持する

`QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md` はかなり有用です。環境情報、source pages、docs-derived examples、join matrix、subquery join hint matrix が入り、operator family normalization の根拠として読めます。

一方で、optimizer は `not_pinned` です。これは観測メモとしては問題ありませんが、golden fixture や CI gate の根拠に昇格する場合は扱いを分けた方がよいです。

```text
observations:
  - docs and Omni probe evidence
  - may be not optimizer-pinned
  - useful for discovering operator names and normalization candidates

fixtures:
  - optimizer/statistics pinned where possible
  - expected normalized family list fixed
  - used to guard plan-report schema / normalizer behavior
```

この分離は、Spanner Omni execution-plan support が Preview / Pre-GA であることとも相性が良いです。観測メモは “この環境で見えた shape” として残し、contract は “この normalized view への semantic predicate” として評価する、という境界が保てます。

## P2: さらに polished にするなら

### 9. `PLAN_CONTRACTS.md` に report output の最小例を 3 つ置く

CI 利用者向けには、説明だけでなく最小 output example が効きます。

```text
- pass example
- fail example
- not_evaluated target_not_found example
```

特に `contract_summary.status` と process exit を分けたので、`report_only` で summary が `fail` でも exit 0 になる例、`check` で non-zero になる例があると誤解が減ります。

### 10. `operator_family` enum の生成元を明示する

候補ドキュメントから手書き enum を消したのは良いです。さらに一歩進めるなら、schema の enum 生成元を README / DESIGN に一文で書くとよいです。

```text
The plan-contract and plan-report operator_family enums are generated from the same normalizer registry used by plan-report.
```

そのうえで、テスト名または expected behavior として次を固定します。

```text
- plan-contract schema operator_family enum == plan-report schema operator_family enum
- every normalized family emitted by observed probe cases is included in both schemas
- every predefined contract expands only to known operator families
```

## 結論

大きな方向転換は不要です。今回の更新で、plan contract はかなり “review artifact として使える” 形に近づいています。

次にやるべきことは、新しい predefined contract を増やすことではなく、次の同期と fixture 化です。

1. 最新 schema に `semi_apply` / distributed semi 系が本当に入っているか確認する。
2. `PLAN_CONTRACTS.md` を semi/anti apply 版の predefined semantics に同期する。
3. `distributed_semi_apply` が anti-semi variant も含むのか、別 family にするのか決める。
4. `tools/spanner-query-plan-shape` README に Preview / Pre-GA caveat を入れる。
5. Push Broadcast / Distributed Apply / Semi Apply 系の golden fixture を固定する。
6. `contract_rule_result.source` / `predefined` を schema でも保証する。

設計としては、`plan-report` を optional / experimental extension に閉じ込め、main config を汚さない方針を維持するのが正しいです。`IN` / `EXISTS` 系まで見たことで、operator-family normalization の実戦度が上がっています。あとは schema と docs がその新しい vocabulary に完全に追随していることを確認すればよい段階です。
