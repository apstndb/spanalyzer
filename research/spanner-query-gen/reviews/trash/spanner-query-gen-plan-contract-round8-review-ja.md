# spanner-query-gen plan contract round 8 review

作成日: 2026-05-06

## 前提

今回も実装コード・実行ログ・テスト出力は共有されていないため、レビュー対象は添付された response / README / PLAN_CONTRACTS / PLAN_CONTRACT_CANDIDATES / schema から読み取れる **public contract と文書上の仕様** に限定します。本文で「反映された」「良い」と言う場合も、外部から検証できるのは文書・schema に現れている範囲です。

## 総評

今回の更新はかなり良いです。特に、次の点は前回の懸念をかなり閉じています。

- `plan-report` を main `spanner-query-gen.yaml` から分離したまま、optional / experimental な Omni-backed review workflow として維持している。
- README と `PLAN_CONTRACTS.md` に Spanner Omni execution-plan support の Preview / Pre-GA caveat を入れ、`plan-report` を development / testing / prototyping / review 用の artifact と明記している。
- `target` を `query/Foo` / `query/Foo#inner` の canonical target ID に寄せ、identifier grammar も schema と parser 側に寄せている。
- `contract_evaluation` を `pass` / `fail` / `not_evaluated` で分け、`not_evaluated` を operator violation と混ぜない形にした。
- `no_hash_join` / `no_standalone_hash_join`、`no_apply_join` / `no_standalone_apply_join` の分離により、名前と挙動の直感がだいぶ揃った。
- `optimizer.requested` と `optimizer.effective` を分離し、v1alpha では SQL statement hint 由来の pinning を推論しないと明記した。

ここまで来ると、大きな方向転換よりも **候補ドキュメントと schema/report artifact の細部をさらに同期する段階** だと思います。

## 追加フィードバック

### 1. `PLAN_CONTRACT_CANDIDATES.md` の “Current v1alpha Contract Surface” は schema と同期する

`PLAN_CONTRACT_CANDIDATES.md` は候補集・検討メモの位置づけですが、冒頭に “Current v1alpha Contract Surface” として現在実装 surface のように読める内容が載っています。ここで示されている `Known families` は、最新の plan-report / plan-contract schema の operator family enum より少ないように見えます。

たとえば observations では documentation examples から新たに `anti_semi_apply`, `array_subquery`, `array_unnest`, `bloom_filter_build`, `compute`, `compute_struct`, `create_batch`, `data_block_to_row`, `empty_relation`, `filter`, `key_range_accumulator`, `limit`, `random_id_assign`, `recursive_spool_scan`, `row_to_data_block`, `scalar_subquery`, `union_input` などが normalization に追加されたと整理されています。一方、候補集の `Known families` はこれらの多くを含んでいません。

対応案はどちらかです。

```text
A. PLAN_CONTRACT_CANDIDATES.md から “Known families” の完全列挙を削る
   -> 最新 enum は plan-contract schema / plan-report schema を参照、と書く。

B. “Known families” を schema から生成する
   -> 手書き drift を防ぐ。
```

候補集に operator family のサンプルを残す場合は、`Known families` ではなく `Example families` / `Common families` のように名前を変えるのが安全です。

### 2. `no_explicit_sort` の説明から substring matching の印象を消す

`PLAN_CONTRACT_CANDIDATES.md` の `no_explicit_sort` 説明に、`Sort`, `Sort Limit`, `Minor Sort`, and other normalized operator names containing `sort` という表現があります。

これは少し危険です。DESIGN 側はすでに、contract matching は rendered-plan snapshot や表示名文字列ではなく normalized operator family に対する semantic predicate で行う方針です。また `Distributed Merge Union` のような ordered-result operator を explicit sort と混同しない、という整理も入っています。

したがって、候補集側も次のように寄せる方がよいです。

```text
no_explicit_sort:
  forbids the normalized explicit_sort family.
  The current normalizer maps observed explicit sort operators such as Sort and Sort Limit to this family.
  This is not a substring match over display_name, and ordered-result operators such as distributed_merge_union remain separate families.
```

`Minor Sort` を含めるなら、実際に Spanner Omni / spannerplan の observed fixture にあるか、または “planned mapping” として扱うかを明記した方がよいです。

### 3. `contract_rule_result` に predefined source を残す

contract file は `use` に複数の predefined contract を書けます。

```yaml
contracts:
- name: OLTPPlan
  target: query/GetSinger
  use:
  - no_explicit_sort
  - no_hash_join
```

一方、plan-report schema の `contract_rule_result` は現在、`rule: forbid_operator_family | cel`、`operator_family`、`observed_count`、`max_count` などを中心にしています。このままだと、`use` が内部的に複数の forbid rule へ展開された時に、どの predefined contract 由来の result なのかが少し見えにくくなります。

report のレビュー性を上げるには、`contract_rule_result` に次のような任意 field を足すとよいです。

```yaml
results:
- rule: forbid_operator_family
  source: use/no_hash_join
  predefined: no_hash_join
  operator_family: push_broadcast_hash_join
  observed_count: 1
  max_count: 0
  status: fail
  failure_kind: violation
```

`source` または `predefined` のどちらか一つで十分です。`forbid` 直書きの場合は `source: forbid[0]` のようにしてもよいですが、まずは `use` 展開だけでも効果があります。

### 4. `contract_evaluation` は pass/fail 時に resolved target metadata を必須にしてよい

現在の schema は `contract_evaluation` で `name`, `target_id`, `status`, `stability` を required にし、`pass` / `fail` では `results` を required にしています。これはかなり良いです。

さらに締めるなら、`status: pass | fail` の時は `query` と `scope` も required にしてよいと思います。pass/fail できるということは target が解決済みなので、resolved target metadata は常に出せるはずです。

```text
status: pass | fail
  required: query, scope, results

status: not_evaluated + reason: target_not_found
  query/scope optional

status: not_evaluated + reason: target_error
  query/scope optional or present if resolution got that far
```

この形にすると downstream tool が `target_id` だけでなく resolved query name / scope を安定して参照できます。

### 5. `contract_summary.status` と process exit の関係を明文化する

README では、`--contracts` だけなら violations / unavailable targets を report に出すが exit code は 0、`--check` を付けると violations と `not_evaluated` が non-zero になる、と説明されています。これは良いです。

ただ、machine-readable report の `contract_summary.status: pass | fail` が **exit status なのか、contract truth なのか** は明文化しておくとさらに安全です。

おすすめは次です。

```text
contract_summary.status:
  pass if every configured contract was evaluated and passed.
  fail if any contract failed or was not_evaluated.

process exit:
  controlled by contract_evaluation_mode.
  report_only keeps exit code 0 even when contract_summary.status is fail.
  check exits non-zero when contract_summary.status is fail.
```

この説明を `PLAN_CONTRACTS.md` に入れておくと、CI 側が `--contracts` without `--check` の report を読んだ時にも迷いません。

### 6. `QUERY_EXECUTION_OPERATORS_OBSERVATIONS.md` には backend identity を入れる

observations document はとても有用です。特に、Spanner docs examples を `tools/spanner-query-plan-shape` で probe し、normalization enum に追加された operator family をまとめている点は良いです。

ただ、plan shape は Omni / optimizer / renderer / toolchain に依存し得るため、観測メモにはできれば次を入れてください。

```text
observed_at: 2026-05-06
spanner_omni_version: ... or not_recorded
spanner_omni_image_digest: sha256:... or not_recorded
spanemuboost_version_or_commit: ...
spannerplan_version_or_commit: ...
optimizer_version: ... or not_pinned
optimizer_statistics_package: ... or not_pinned
```

`plan-report` artifact 側では backend identity / digest / optimizer status を重視しているので、observations doc も同じ考え方に寄せると、normalization rule を後で見直す時の evidence として強くなります。

また、observations doc に `/private/tmp/...` のパスが残っています。developer note としては問題ありませんが、repo に置く文書なら `tmp/spanner-docs-best-practices-base.sql` や `<tmp>/spanner-docs-best-practices-base.sql` のように一般化すると読みやすいです。

### 7. golden fixture 化は次の最優先でよい

response でも、Push Broadcast / Distributed Cross Apply / Aggregate classification の observed plan shape golden fixture 化はまだと書かれています。これは次に優先してよいです。

特に fixture 化した方がよいのは次です。

```text
- explicit Sort / Sort Limit
- Distributed Merge Union が explicit_sort にならないこと
- Push Broadcast Hash Join wrapper + internal Hash Join
- standalone Hash Join
- Distributed Cross Apply wrapper + internal Apply
- standalone Apply / Cross Apply
- Hash Aggregate / Stream Aggregate / unknown aggregate classification
- docs examples 由来の newly added operator families
```

ここを固定できると、operator normalization を変更するときに “contract semantics が変わったのか、renderer 表示が変わっただけなのか” を切り分けやすくなります。

### 8. `PLAN_CONTRACTS.md` と `PLAN_CONTRACT_CANDIDATES.md` の役割をさらに分ける

今の README から `PLAN_CONTRACTS.md` に詳細を逃がした判断は良いです。次にやるなら、2 つの plan-contract 文書の役割をさらに明確にするとよいです。

```text
PLAN_CONTRACTS.md:
  current documented v1alpha surface only
  schemaと一致する grammar / predefined / report semantics

PLAN_CONTRACT_CANDIDATES.md:
  future ideas, prior art, non-normative observations
  current surface を再掲しない、または schema へのリンクに留める
```

`PLAN_CONTRACT_CANDIDATES.md` に “implemented today” や “Known families” の完全列挙があると、そこが README / schema と drift しやすくなります。候補集は設計メモとして価値が高いので、normative surface を持たせない方が長持ちします。

## 今回の結論

大きな設計変更は不要です。現在の方向性はかなり良いです。

次にやるべきことは、新しい predefined contract を増やすことではなく、次の整理だと思います。

1. `PLAN_CONTRACT_CANDIDATES.md` の current surface / known families の drift を消す。
2. `no_explicit_sort` の説明を substring matching ではなく normalized family matching に統一する。
3. `contract_rule_result` に predefined source を残す。
4. `contract_evaluation` pass/fail の resolved metadata をもう少し required にする。
5. `contract_summary.status` と process exit の意味を明文化する。
6. observations doc に backend identity / optimizer identity を入れる。
7. Push Broadcast / Distributed Cross Apply / Aggregate classification の golden fixture を作る。

この段階では、`plan-report` は optional / experimental でありながら、CI artifact としてはかなり形になっています。main v1alpha config の単純さを崩さず、opinionated な plan contract を別 surface として育てる方針は維持でよいと思います。
