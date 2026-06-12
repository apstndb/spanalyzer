# `spanner-query-gen` v1alpha 最新フィードバックへの回答

## 回答

レビューありがとうございます。今回の指摘はかなり妥当だったため、v1alpha の public YAML surface をさらに削る方向で反映しました。

特に、v1alpha は次の grammar に寄せます。

```text
version: v1alpha
go
emit
catalogs
queries
writes
rules.suppressions
```

`rules.severity`、write-level `emit`、raw SQL `EXTERNAL_QUERY(...)`、`connection_id` alias、external dataset `access.database_role` は v1alpha YAML から外します。`input` は scalar struct name、`key` は YAML sequence のみを canonical shape とします。

## 反映した点

### `upsert` semantics

採用します。

`operation: upsert` + `conflict.strategy: insert_or_update` は Spanner `INSERT OR UPDATE` に忠実に扱う必要があるため、v1alpha では次の invariant にします。

```text
insert.columns == key columns + update.columns
```

`insert.columns` に insert-only non-key column がある config は planning error にします。`update.columns` にある列が `insert.columns` にない場合も error にします。branch が異なる upsert は、将来の `ON CONFLICT DO UPDATE SET ...` strategy で扱う方針にします。

### cleanup plan / README / DESIGN の同期

採用します。

cleanup plan に残っていた古い grammar を同期しました。

- write-level `emit: [mutation, dml]` を削除。
- `methods` の移行先を write-level emit ではなく global `emit.spanner` に変更。
- external dataset config から `access.mode`、`access.database_role`、`verification.status: not_checked` を削除。
- `Federated...` example 名を `ExternalQuery...` に変更。
- minimal examples から不要な `result.cardinality` を省略。

### raw SQL `EXTERNAL_QUERY`

採用します。

v1alpha public YAML では raw SQL に含まれる `EXTERNAL_QUERY(...)` を拒否し、`kind: external_query` を canonical にします。literal-only extraction は、必要なら将来の migration helper として別扱いにします。

### `rules.severity`

採用します。

v1alpha の `rules` は suppressions-only にします。severity override は便利ですが、full rule engine や `--strict` と同じ policy layer に属するため、現時点では future work に戻します。

### `queries[].params`

当初は削る方針に寄せましたが、再検討して撤回します。

`queries[].params` は query method だけでなく analyzer 入力として必要です。特に analyzer が parameter type を推論できないケース、ARRAY / STRUCT parameter、BigQuery outer SQL と Spanner inner SQL の scope が分かれる `external_query` では、明示 parameter 型を渡せないと解析できない可能性があります。

したがって `queries[].params` は v1alpha MVP に残します。位置付けは「method signature の確定」ではなく「analyzer type input / override」です。

### write-level `emit`

採用します。

v1alpha では global `emit.spanner` のみで生成 surface を制御します。operation-specific incompatibility は planner が適用します。例えば `replace` は global `emit.spanner.dml: true` でも DML helper を出しません。

### schema / help 出力

追加提案として採用します。

レビュー可能性を上げるため、`spanner-query-gen config-schema --output json|yaml` を追加し、v1alpha config の JSON Schema を出力できるようにします。また README には root `--help` の出力を含めます。schema は canonical v1alpha shape を示すものなので、future/rejected fields は含めません。

さらに、最新 schema は `schemas/spanner-query-gen.v1alpha.schema.json` に保存し、command output と保存済み schema が一致することをテストで固定します。

## あえて残した点

`result.cardinality` 自体は schema には残します。ただし README の minimal example からは外し、query method semantics が必要な場面で使う optional field として扱います。

また、内部 Go 構造には historical な `Federated` 名がまだ残っています。public YAML、README、DESIGN、machine-readable diagnostic stage は `external_query` に寄せましたが、内部 rename は別途まとめて行う方が安全だと判断しました。

## 現在の v1alpha 方針

- `version: v1alpha` のみ受け付ける。
- `v1alpha` は v1 確定までは mutable preview。
- v1 確定後は、その時点の `v1alpha` を deprecated alias として固定する。
- 次の破壊的 preview は `v2alpha` など別 version にする。
- config schema と `explain-plan` をレビュー用 contract として扱う。
