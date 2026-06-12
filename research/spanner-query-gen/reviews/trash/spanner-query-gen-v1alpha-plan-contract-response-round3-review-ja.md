# `spanner-query-gen` v1alpha plan contract round 3 review

対象: `spanner-query-gen-v1alpha-plan-contract-response-round2-response-ja.md`, `README(20).md`, `DESIGN(20).md`, `spanner-query-gen.v1alpha.schema(8).json`

前提: 実装コード、実行ログ、テスト出力は共有されていません。そのため、このレビューは **public contract / README / DESIGN / schema に見えている仕様のレビュー** です。実装済みかどうかは判断していません。

## 総評

今回の対応はかなり良いです。特に、`plan-report` を main `spanner-query-gen.yaml` に混ぜず、optional / experimental な外部 contract workflow に閉じ込めている点は、v1alpha core config を simple に保つ方針と整合しています。

また、以下は前回より明確になっています。

- `plan-report` は primary DDL-first generation path ではなく、Omni-backed review artifact である。
- plan contract は production performance guarantee ではなく、backend / DDL digest / SQL digest / optimizer settings / normalized operator tree に紐づく review contract である。
- `--contracts` without `--check` は contract を評価して report に含めるが、violation では exit code を変えない。
- `--check` は violation で non-zero にする。
- target selection は Spanner SQL に限定し、BigQuery outer SQL / external dataset bindings / writes は対象外にする。
- `kind: external_query` の `inner_sql` は Spanner SQL として `plan-report` 対象にする。
- report に `plan_source`, backend identity, normalization versions, target summary, digest fields を入れる。
- `Sort` / `Sort Limit` と `Distributed Merge Union` を区別しており、`no_explicit_sort` の意味がかなり健全になっている。

ここまで整理されていれば、`plan-report` はかなり差別化できる機能になります。一方で、今回の更新で **CEL と raw QueryPlan への露出** が増えたため、次は「何を安定 contract と呼べるか」をさらに分ける必要があります。

## P0: raw `QueryPlan` を CEL に渡すなら、contract stability tier を明示する

README / DESIGN は、CEL に normalized view だけでなく raw `spannerpb.QueryPlan` と `[]*spannerpb.PlanNode` を渡すとしています。これは表現力としては強いですが、CI contract としてはかなり危険です。

raw QueryPlan metadata は backend、optimizer version、statistics package、plan mode、API surface、将来の proto / metadata 変更で揺れやすいです。DESIGN でも raw metadata は揺れうると書かれていますが、CI で `--check` するユーザーは、CEL に書けるものをすぐ安定 contract と見なしてしまいます。

提案は、contract evaluation result に stability tier を必ず出すことです。

```yaml
contract_results:
- name: HashAggregatePlanShape
  status: pass
  stability:
    tier: raw_query_plan
    reason: cel expression references nodes/plan
    check_recommended: false
```

または contract file 側で明示させてもよいです。

```yaml
contracts:
- name: AdvancedRawPlanCheck
  query: SomeQuery
  backend: omni
  stability: raw_query_plan
  cel: |
    nodes.exists(n, n.display_name == "Aggregate" && n.metadata["iterator_type"] == "Hash")
```

さらに安全にするなら、`--check` で raw `plan` / `nodes` を参照する contract は、`allow_raw_query_plan: true` のような明示 opt-in がない限り warning または error にするのがよいです。

## P0: execution stats を contract に使う場合は、plan-mode と data-dependence を分離する

README / DESIGN は raw QueryPlan から execution stats も参照できるとしています。ただし、`plan-report` の通常目的は「データ分布にあまり依存しない plan shape contract」のはずです。実行統計は、まさに runtime / dataset / profile mode に依存します。

したがって、CEL に `execution_stats` 相当を見せるなら、次のどちらかに寄せるべきです。

### 案 A: v1alpha experimental contract では execution stats を CEL 入力から除外する

最も simple です。`render-mode: plan` の contract は structural plan だけを対象にし、profile stats は report artifact には出しても pass/fail contract には使わせない。

### 案 B: profile contract を別扱いにする

```yaml
contracts:
- name: NoLargeScanInProfile
  query: SomeQuery
  backend: omni
  mode: profile
  data_dependent: true
  cel: |
    operators.all(o, o.stats.rows < 10000)
```

この場合、report には以下を出すべきです。

```yaml
contract_results:
- name: NoLargeScanInProfile
  status: pass
  data_dependent: true
  plan_mode: profile
  stable_snapshot_recommended: false
```

`EXPLAIN ANALYZE` / profile 系は価値がありますが、今回の提案の核である「operator が含まれる / 含まれない contract」とは別物です。

## P0: predefined contract を増やすなら、classification confidence を持つ

`no_explicit_sort` だけでなく、`no_hash_join`, `no_push_broadcast_hash_join`, `no_apply_join`, `no_merge_join`, `no_hash_aggregate`, `no_stream_aggregate` まで experimental predefined に広げた判断は理解できます。main config に入れていないので、core surface の複雑化は防げています。

ただし、aggregate 系は特に注意が必要です。DESIGN は `Aggregate` node を `iterator_type: Hash` / `Stream` で分類するとしていますが、この metadata が存在しない、または想定外の値になる可能性があります。そのときに `no_hash_aggregate` が単に pass すると、review contract として危険です。

提案は、operator classification に confidence / unknown を持つことです。

```yaml
operators:
- id: 3
  display_name: Aggregate
  family: aggregate
  subfamily: unknown
  classification:
    source: metadata.iterator_type
    confidence: unknown
    reason: iterator_type missing
```

そして contract result は三値にするのが理想です。

```yaml
status: pass | fail | unknown
```

MVP で三値が重いなら、少なくとも `contract_summary.environment_warnings` か `classification_warnings` に出し、`--check` 時の default を明記してください。私なら、predefined contract が unknown classification に依存する場合は `--check` では fail に寄せます。

## P1: optimizer pinning warning を contract policy に昇格できるようにする

`optimizer_not_pinned`, `statistics_package_not_pinned`, `query_optimizer_not_pinned` を environment warning として report に出す方針は良いです。

ただし、plan contract を CI gate として使うチームは、「optimizer が pin されていないなら operator contract は信用しない」という運用にしたくなるはずです。warning だけだと、contract は pass しているが実は環境が緩い、という状態が埋もれます。

main YAML には入れず、contract file 側にだけ policy を置くとよいです。

```yaml
version: v1alpha-plan-contracts

environment:
  optimizer_pinning: require
  statistics_package_pinning: warn

contracts:
- name: SingerIndexLookupPlan
  query: ScanSingerIDsFast
  backend: omni
  use:
  - no_explicit_sort
```

またはもっと simple に、CLI flag でもよいです。

```sh
spanner-query-gen plan-report \
  --contracts plan-contracts.yaml \
  --check \
  --require-optimizer-pinning
```

## P1: `external_query` inner SQL の target ID を canonical にする

`kind: external_query` の `inner_sql` を plan-report 対象に含める判断は正しいです。ただ、contract file の `query: ExternalQuerySingerIDs` だけでは、将来 `outer_sql` や DML command も plan-report 対象になった時に曖昧になります。

report には canonical target ID を出すべきです。

```yaml
targets:
- id: query/ScanSingerIDsFast
  query: ScanSingerIDsFast
  scope: spanner_query
- id: query/ExternalQuerySingerIDs#inner
  query: ExternalQuerySingerIDs
  scope: external_query_inner
```

contract file も、将来は `target` を優先する方が安全です。

```yaml
contracts:
- name: ExternalInnerPlan
  target: query/ExternalQuerySingerIDs#inner
  backend: omni
  use:
  - no_explicit_sort
```

MVP では `query` alias を残してもよいですが、report には canonical target ID を必ず出してください。

## P1: README の contract section は少し advanced すぎる

README の plan contract section は、`no_explicit_sort`、CEL、raw protobuf、hash aggregate、direct forbid rule、target selection まで一気に説明しています。機能としては良いのですが、README の simple 方針から見ると少し重いです。

おすすめは次です。

README では最小例だけを出す。

```yaml
version: v1alpha-plan-contracts

contracts:
- name: SingerIndexLookupPlan
  query: ScanSingerIDsFast
  backend: omni
  use:
  - no_explicit_sort
```

CEL / raw QueryPlan / aggregate split / direct forbid は DESIGN か `docs/plan-contracts.md` に逃がす。

README には一文だけ残す。

```text
Advanced CEL contracts and raw QueryPlan access are experimental and documented in DESIGN.md.
Prefer predefined contracts or the normalized operators view for CI.
```

この方が、main config を simple にした努力が README 上でも伝わりやすいです。

## P1: direct `forbid` rule の shape を README か schema comment に入れる

README は「Direct `forbid` rules count normalized operator nodes」と説明していますが、実際の YAML shape が見えません。`use` と `cel` の例はあるので、`forbid` も 1 つだけ例を置くと、誤解が減ります。

```yaml
contracts:
- name: NoExplicitSort
  query: ScanSingerIDsFast
  backend: omni
  forbid:
  - operator_family: explicit_sort
    max_count: 0
```

もし `use: [no_explicit_sort]` が基本推奨なら、direct rule は README から消して DESIGN だけでもよいです。

## P1: contract report schema version を出す

report には `normalization.operator_tree_version` と `normalization.operator_family_mapping_version` が入る方針ですが、contract evaluation 自体の version もあった方がよいです。

```yaml
report_version: v1alpha-plan-report-v1
contract_evaluator_version: v1
contract_file_version: v1alpha-plan-contracts
```

特に CEL input shape は今後変わりやすいので、`operators` view の schema version と contract evaluator version を分けて記録してください。

## P2: `backend_identity.version: not_recorded` はよいが、将来の必須化方針を残す

現時点で `backend_identity.version: not_recorded` / `image_digest: not_recorded` を入れるのは、field shape を先に固定する意味で良いです。ただし、plan contract が CI gate になると、backend image digest がない report は再現性が弱いです。

将来の hardening として次を残すとよいです。

```yaml
backend_identity:
  kind: omni
  version:
    value: not_recorded
    required_for_check: false
  image_digest:
    value: not_recorded
    required_for_check: false
```

または `environment_warnings` に `backend_identity_not_recorded` を入れてもよいです。

## P2: hint recommendation の confidence は、contract ごとに出す

remediation に `applies_to` と `confidence` を入れる方針は良いです。さらに、recommendation は target-level ではなく violation-level に紐づけるのがよいです。

```yaml
violations:
- contract: NoExplicitSort
  operator_family: explicit_sort
  remediation:
  - message: Consider order_by: none for generated shorthand if ordering is not required.
    applies_to: query/ScanSingerIDsFast
    confidence: medium
    auto_fix: false
```

`auto_fix: false` を明示すると、SQL is contract の方針と衝突しないことがより分かりやすいです。

## 追加で確認したい小さな点

- `--stable` で contract evaluation result の volatile fields も落ちるか。例えば raw plan text、profile stats、backend image local ID など。
- `operator_tree_sha256` は normalized operator tree に基づくので、`--format`, `--wrap-width`, rendered plan text に依存しないと明記されているか。
- `plan_source.api: analyze_query` と README の `render-mode: profile` の関係。`profile` が実際に stats acquisition を伴うなら、`plan_source.api` は `analyze_query` だけでは足りず、`plan_mode: profile` を明確にする必要がある。
- `GROUP@{GROUP_METHOD=...}` / `JOIN@{JOIN_METHOD=...}` による integration test は良いが、それは “hint が効くこと” の test であり、すべての production query で family classification が安定することの証明ではない、と docs に一文入れておく。

## そのまま送れる返答案

以下のように返すとよいと思います。

> 今回の反映はかなり良いです。`plan-report` を main v1alpha config に入れず、外部 contract file + optional Omni-backed workflow に閉じ込めている点は、core config を simple に保つ方針と合っています。`plan_source`, backend identity, normalization versions, target summary, digest definitions, `--contracts` without `--check` semantics も、review artifact としてかなり良くなっています。
>
> 追加で一番大きい懸念は、CEL に raw `spannerpb.QueryPlan` / `PlanNode` / execution stats を渡すことで、CI contract としての安定性境界が曖昧になることです。normalized `operators` view を推奨する方針は良いですが、raw `plan` / `nodes` を参照する contract は `stability: raw_query_plan` や `uses_raw_plan: true` を report に出し、`--check` で使う場合は明示 opt-in にした方がよいです。
>
> また、execution stats は plan shape contract とは別扱いにしてください。`EXPLAIN ANALYZE` / profile 系は dataset/runtime dependent なので、v1alpha experimental contract では pass/fail に使わせないか、`mode: profile` + `data_dependent: true` のように明示するのが安全です。
>
> predefined contract を `no_hash_join` や aggregate 系まで広げる判断は、main config に入れていないので許容できます。ただし aggregate の `hash_aggregate` / `stream_aggregate` 分類は `iterator_type` metadata に依存するため、metadata が欠落した場合の `unknown` state を plan/report に出してください。`no_hash_aggregate` が unknown aggregate を黙って pass すると contract として危険です。
>
> README は少し advanced になってきたので、最初の plan-contract example は `no_explicit_sort` だけにし、CEL / raw QueryPlan / direct forbid / aggregate classification は DESIGN または別 docs に逃がすと、simple 方針がより伝わります。

## 結論

設計の大枠は良いです。`plan-report` は `spanner-query-gen` の差別化になり得ますし、main YAML を汚さない今の分離は維持すべきです。

次に詰めるべきなのは、機能追加ではなく次の 4 点です。

1. raw QueryPlan / raw PlanNode を使う contract の stability tier
2. execution stats / profile mode を structural plan contract と分けること
3. operator classification unknown の扱い
4. README の最小例と advanced docs の分離

この 4 点が整理できれば、`plan-report --contracts --check` は experimental でありながら、CI で使う人が誤解しにくい contract になると思います。
