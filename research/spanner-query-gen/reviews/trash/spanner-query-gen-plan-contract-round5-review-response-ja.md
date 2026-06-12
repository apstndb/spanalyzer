# spanner-query-gen plan contract round 5 review response

作成日: 2026-05-06

## 概要

round 5 の指摘は概ね取り込みました。特に v1alpha では破壊的変更が許される前提なので、互換 alias や将来余地を残すためだけの CLI flag は削り、review artifact として明確な形に寄せました。

## 反映した内容

- `plan-report` の `--render-mode` flag は削除しました。
  - v1alpha は structural PLAN 専用です。
  - report artifact には引き続き `render_mode: PLAN` を出します。
- plan contract file は canonical `target` のみに寄せました。
  - `query` / `scope` alias は削除しました。
  - `target: query/Foo` または `target: query/Foo#inner` を使います。
- contract entry は exactly one mode にしました。
  - `use`
  - `forbid`
  - `cel`
  - これらの同時指定は contract-file validation error です。
- `backend` は contract entry から削除しました。
  - backend は `plan-report` 実行環境と report artifact の属性です。
- `forbid.operator_family` は既知の normalized operator family に限定しました。
  - schema enum にも反映しています。
- `contract_rule_result` schema は rule/status ごとの required fields を増やしました。
- `format` は report schema 上で `CURRENT` / `TRADITIONAL` / `COMPACT` enum にしました。
- plan-report / plan-contract schema は最新を `schemas/` に再生成しました。
- README / DESIGN の構成を少し整理し、現在実装との差分は `cmd/spanner-query-gen/IMPLEMENTATION_STATUS.md` に分けました。

## Push Broadcast Hash Join と Hash Join の扱い

最初に Spanner Omni の actual plan shape を確認しました。単純な `JOIN@{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN}` では、plan に次のような構造が現れます。

- `Push Broadcast Hash Join`
- その `Map` child 以下に `Serialize Result`
- さらにその下に implementation detail としての `Hash Join`
- その implementation `Hash Join` は pushed batch input 由来の `BatchScan` を消費する

したがって、単に `Hash Join` display name を見て `hash_join` と数えると、Push Broadcast Hash Join の内部実装まで `no_hash_join` に引っかかります。

一方で、`Map` subtree に含まれる `Hash Join` をすべて内部扱いするのも誤りです。複数段 JOIN では、broadcast side input の中に通常の Hash Join が含まれる可能性があります。

現在の v1alpha 実装では次の条件を満たす Hash Join だけを Push Broadcast 内部扱いにしています。

- `Push Broadcast Hash Join` の `Map` child から到達できる。
- その path が relational operator tree として分岐していない。
- 候補の `Hash Join` が pushed batch 由来の `BatchScan` input を消費している。

これにより、operator family は次のように分かれます。

- `hash_join`: standalone Hash Join
- `push_broadcast_hash_join`: Push Broadcast Hash Join wrapper
- `push_broadcast_hash_join_internal_hash_join`: Push Broadcast の内部実装 Hash Join

そのため `no_hash_join` は standalone `hash_join` を禁止します。`Push Broadcast Hash Join` 自体を禁止したい場合は `no_push_broadcast_hash_join` を使います。

なお、この分類は Spanner Omni の observed structural PLAN shape に基づく normalization です。完全な optimizer contract ではないため、optimizer version / statistics package の pinning と schema/report digest を併用して review artifact として扱う方針は変えていません。

## Distributed Cross Apply への同じ考え方

同様に `JOIN@{JOIN_METHOD=APPLY_JOIN}` の Spanner Omni plan では、次の構造を確認しました。

- `Distributed Cross Apply` wrapper
- その `Map` child 以下に `Serialize Result`
- さらにその下に implementation detail としての `Cross Apply`
- その implementation `Cross Apply` は pushed batch input 由来の `BatchScan` を消費する

これも Push Broadcast と同じ rule で分類し、次の family を追加しました。

- `apply_join`: standalone Cross Apply / Outer Apply
- `distributed_cross_apply`: Distributed Cross Apply wrapper
- `distributed_cross_apply_internal_apply`: Distributed Cross Apply の内部実装 Apply

predefined contract として `no_distributed_cross_apply` も追加しました。

## 開発用 plan shape probe

その場で作った `/private/tmp/spanner-query-plan-shape` は、再利用できるように repo の `tools/spanner-query-plan-shape` に移しました。

用途は public CLI ではなく、operator normalization や plan contract を変更する前に Spanner Omni の raw plan shape を確認するための developer tool です。

例:

```sh
go run ./tools/spanner-query-plan-shape
go run ./tools/spanner-query-plan-shape --case push_broadcast_hash_join
go run ./tools/spanner-query-plan-shape --ddl schema.sql --sql 'SELECT ...'
```

Docker と Spanner Omni image が必要です。

## 保留・今後の課題

- report には requested optimizer options と pinning 状態を出していますが、backend から見た effective optimizer version / source split はまだ未実装です。
- backend identity は `not_recorded` のままです。
- operator family catalog は今後の observed plan と contract need に応じて拡張します。
- plan contract は PLAN の structural shape を対象にし、PROFILE / execution stats は明示的に対象外です。
