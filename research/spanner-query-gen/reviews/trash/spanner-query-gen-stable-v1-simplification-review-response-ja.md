# `spanner-query-gen` stable v1 simplification review への回答

## 回答

レビューありがとうございます。今回の指摘は採用します。

特に、`version: "1"` はまだ正式な v1 互換性を約束しているように見えるため、公開 YAML config の version は `v1alpha` に変更します。これにより、`spanner-query-gen` の config は「v1 に向けた alpha surface であり、正式 v1 までは破壊的変更を許容する」ものとして扱います。

実装・文書では次の方針に寄せます。

- YAML config は `version: v1alpha` のみ受け付ける。
- Go 側の公開 config struct 名も `QueryCodegenV1Alpha...` に寄せ、安定 v1 API のように見せない。
- README / DESIGN / cleanup plan の “stable v1” 表現を `v1alpha` に置き換える。
- `v1alpha` は pre-release であり、初回 stable v1 までは互換性維持より仕様整理を優先できることを README と DESIGN に明記する。
- 旧 compact config は引き続き拒否し、`schemas` / `source` / `update_mask` などは具体的な移行先を示す diagnostic にする。

## 取り込んだレビュー項目

### CLI 表現の同期

採用します。README と DESIGN は `generate` / `check` / `explain-plan` / `vet` の subcommand 前提に統一します。no-subcommand generation は v1alpha では残しません。

### “current implementation supports” の表現

採用します。DESIGN では “Current documented working-tree surface” のように、外部レビュー可能な contract と実装意図を混ぜすぎない表現へ寄せます。

### `federated` 語彙の削除

採用します。public config では `kind: external_query` に統一し、Roadmap も “External Query UX” に改めます。内部構造に historical な名前が残る場合でも、README / DESIGN の public grammar では `federated` を出しません。

### `query_methods`

採用します。query methods はまだ stable に見せるべき surface ではないため、README の minimal config から外します。現時点の最小例は確実に扱える DTO / mutation / DML helper に寄せます。

### global `emit` と write-level `emit`

採用します。semantics は intersection として明文化します。

```text
generated write surfaces = global emit.spanner mutation/dml surfaces intersected with writes[].emit
```

write-level `emit` が省略された場合は global `emit.spanner` に従います。write-level `emit` は boolean map に統一し、list shorthand は v1alpha では拒否します。

### `projection.named_schema_tables`

採用します。public config の語彙は `projection.named_schema_tables` に統一し、`named_schema_policy` は内部や移行説明以外では出さないようにします。

### shorthand sugar を増やさない

採用します。`result: SingerRow` や `update: [FirstName]` のような shorthand は v1alpha には入れません。単純化は grammar の短縮ではなく、段階的 README と normalized plan で実現します。

## 追加判断

`v1alpha` は `v1alpha1` ではなく、まずは `v1alpha` とします。

理由は、まだ正式リリース前で alpha 内の互換性も固定したくないためです。正式 v1 が固まるまでは `v1alpha` を mutable preview channel として扱い、破壊的変更を許容します。

正式 v1 が固まった時点では、次の移行方針にします。

- canonical version として `version: v1` を追加する。
- その時点の `v1alpha` は固定し、`v1` と同じ schema / normalizer への deprecated alias として残す。
- `v1alpha` alias は以後意味を変えない。次の破壊的 preview が必要なら `v2alpha` など別 version を使う。
- 移行期間中は `v1alpha` 入力に warning を出し、`v1` への移行を促す。
- `v1alpha` と `v1` が同じ normalized plan になることを golden test で固定する。

つまり、`v1alpha` は v1 確定前は「v1 に向けた alpha channel」、v1 確定後は「v1 への移行用 alias」として扱います。alpha の中で互換性を段階的に固定したくなった時点で、`v1alpha2` や `v1beta1` のような versioning を検討します。

## 残る方針

当面の public grammar は次に絞ります。

```text
version: v1alpha
go
emit
catalogs
queries
writes
rules
```

`dto.structs`、`overrides`、`models`、`queries_file`、BigQuery query methods、Partitioned DML、`migrate-config` は v1alpha の最小 public surface には入れず、DESIGN の future work として扱います。
