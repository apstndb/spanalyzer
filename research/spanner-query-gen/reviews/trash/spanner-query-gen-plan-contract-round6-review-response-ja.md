# spanner-query-gen plan contract round 6 review response

作成日: 2026-05-06

## 概要

round 6 の指摘は、v1alpha の破壊的変更許容フェーズで締めるべき contract surface として概ね採用しました。今回の実装では、新機能を広げるよりも CI artifact としての曖昧さを減らす方向を優先しています。

## 反映した内容

- `no_hash_join` を直感に合わせて広義化しました。
  - `hash_join` と `push_broadcast_hash_join` wrapper を禁止します。
  - Push Broadcast Hash Join の内部実装 node である `push_broadcast_hash_join_internal_hash_join` は数えません。
  - standalone のみを禁止したい場合のために `no_standalone_hash_join` を追加しました。
- `no_apply_join` も同じ方針にしました。
  - `apply_join` と `distributed_cross_apply` wrapper を禁止します。
  - distributed wrapper の内部実装 node である `distributed_cross_apply_internal_apply` は数えません。
  - standalone のみを禁止したい場合のために `no_standalone_apply_join` を追加しました。
- canonical `target` の曖昧さを避けるため、v1alpha の public names を identifier に制限しました。
  - 対象は catalog / query / write / external_query_connection / spanner_external_dataset / plan contract name です。
  - pattern は `[A-Za-z_][A-Za-z0-9_]*` です。
  - これにより `query/Foo` と `query/Foo#inner` の escaping を導入せずに済みます。
- plan-report schema に operator family enum を共有する `$defs.operator_family` を追加しました。
  - `operator.family`
  - `query.operator_families[]`
  - `query.operator_family_counts` の `propertyNames`
  - `contract_rule_result.operator_family`
  - これらを同じ enum に寄せました。
- contract target が評価不能な場合を `fail` と分離しました。
  - `status: not_evaluated`
  - `reason: target_not_found`
  - `reason: target_error`
  - `--check` では violation と同様に non-zero exit の対象です。
  - report artifact 上では operator contract violation と target availability failure が混ざらないようにしました。
- optimizer field を requested / effective に分けました。
  - `optimizer.requested.version`
  - `optimizer.requested.statistics_package`
  - `optimizer.effective.version`
  - `optimizer.effective.statistics_package`
  - v1alpha では effective は `not_recorded` として明示します。
  - `--require-optimizer-pinning` は requested pinning の有無を判定する、という整理にしました。
- `stable` は plan-report schema の root required field にしました。
- plan-contract schema の exactly-one mode は `oneOf` 中心に寄せ、冗長な `if/then` は削りました。
- plan contract 詳細を `cmd/spanner-query-gen/PLAN_CONTRACTS.md` に分割し、README は最小例とリンク中心にしました。
- Spanner Omni execution-plan support が Preview / Pre-GA feature であり、development / testing / prototyping / demonstration 用とされている点を README と `PLAN_CONTRACTS.md` に明記しました。
- CEL で子孫条件を書きやすいように、normalized topology を plan-report に追加しました。
  - `operator_edges`
  - `operators[].child_indexes`
  - `operators[].descendant_indexes`
  - `operators[].subtree_family_counts`
- plan-report / plan-contract / config schema の checked-in JSON を再生成しました。
- README / DESIGN / IMPLEMENTATION_STATUS / PLAN_CONTRACT_CANDIDATES を実装後の contract surface に合わせて更新しました。

## CEL と子孫条件について

レビュー待ちの間に検討した点として、CEL だけで任意深さの子孫条件を書くのは現実的ではないと判断しています。

CEL の標準 macro で `exists` / `all` は使えますが、graph/tree を再帰的に辿る `descendants(node).exists(...)` のような形を YAML 側だけで自然に表現するのは困難です。Go 側で custom function を追加することは可能ですが、predicate を受ける汎用 descendant helper にすると UX と実装が重くなります。

したがって、次の方針が良いと考えています。

- plan-report 側で `operator_edges`, `descendant_indexes`, `subtree_family_counts` などを正規化して出す。
- CEL はその normalized topology の上に薄く書く。
- `no_back_join` のような contract は topology だけでなく schema catalog による index-to-base-table mapping も使う。

このうち `operator_edges`, `operators[].child_indexes`, `operators[].descendant_indexes`, `operators[].subtree_family_counts` は先に実装しました。`ancestor_indexes` や schema-aware な `base_table` はまだ未実装です。

特に `no_back_join` は、同じ subtree に index scan と table scan が存在するだけでは誤検出します。target index の base table と table scan target が一致することを確認する必要がある、という整理を `PLAN_CONTRACT_CANDIDATES.md` に明記しています。

## まだ反映していない内容

- observed plan shape の golden fixture 化はまだ実施していません。
  - `tools/spanner-query-plan-shape` はありますが、fixture として固定する作業は未実施です。
  - Push Broadcast / Distributed Cross Apply の normalization rule を今後変更する前に追加する価値があります。
- normalized topology に `ancestor_indexes` や relational-only edge view はまだ入れていません。
  - 必要になった時点で追加するのがよさそうです。
- effective optimizer version/source の実測はまだ未実装です。
  - field は分離しましたが、現時点では `not_recorded` です。

## 検証

- `go test ./cmd/spanner-query-gen`
- `go test ./...`

どちらもローカルで実行しています。
