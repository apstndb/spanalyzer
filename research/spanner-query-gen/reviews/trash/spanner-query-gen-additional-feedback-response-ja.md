# Response to additional `spanner-query-gen` feedback

Review source: `spanner-query-gen-additional-feedback-after-response-ja.md`

## Summary

追加レビューの方向性には同意します。今回の指摘は大きな設計転換ではなく、
`spanner-query-gen` が将来危険なコードを生成しないための仕様固定と regression
test 強化が中心なので、議論の余地が小さいものは実装・文書化しました。

なお、このリポジトリはまだ正式リリース前なので、plan schema や compact config
の破壊的変更は許容しています。

## Applied now

### Regression tests for already claimed behavior

以下はテストで固定済みです。

- `replace` の default は mutation helper のみ。
- `methods: [dml]` for `replace` は planning error。
- generated output に `INSERT OR REPLACE` を出さない。
- Spanner index shorthand は index key、base table primary key、`STORING`
  columns を SELECT する。
- `update` / `insert_or_update` の `update_mask` 省略は error。
- `update_mask` に primary key を入れると error。

今回さらに、以下を追加しました。

- `insert_or_update` の insert branch が `NOT NULL` required column を満たさない
  場合は error。
- generated / hidden / primary key など non-writable column を update 対象にする
  と error。
- `ORDER BY ... LIMIT 1` は inner ordering warning の対象外であることをテスト。

### `insert_or_update` / upsert insert branch validation

compact config では、現時点では `insert_or_update` の insert value set を
`primary key + update_mask` として扱います。その上で、insert branch に必要な列を
検証するようにしました。

現在の判定:

- required insert columns は primary key と、`NOT NULL` かつ default/generated/hidden
  など server-provided value がない列。
- update columns は explicit `update_mask` の列のみ。
- update mask に primary key / generated / hidden column が入る場合は error。
- insert branch に required column が足りない場合は error。

structured config では、レビュー案どおり `insert.columns` と `update.mask` を分ける
方向で DESIGN に migration map を追加しました。

### Spanner column capability model

plan に write column capability を追加しました。

含めた情報:

- `nullable`
- `primary_key`
- `hidden`
- `insert_value`
- `update_value`
- `default_sql`
- `generated_sql`
- `on_update_sql`
- `allow_commit_timestamp`

また、`ON UPDATE` expression を持つ列については、明示 update columns とは別に
`server_side_update_effects` として plan に出すようにしました。これにより
「update_mask にないから絶対に変わらない」と誤読されるのを避けます。

### Machine-readable plan metadata

`explain-plan` の YAML/JSON に以下を追加しました。

- `plan_version`
- generator metadata
- schema DDL digests
- proto descriptor digests
- resolved SQL digest
- structured warnings
- vet suppressions with optional `owner` / `expires`
- write column capabilities
- server-side update effects

warnings は単なる文字列ではなく、`rule`, `severity`, `message`,
`remediation` を持つ構造にしました。

### Federated query placeholder and warnings

`federated.outer_sql` の placeholder semantics を狭く固定しました。

- omitted: `SELECT * FROM EXTERNAL_QUERY(...)`
- supplied: `__external__` must appear exactly once
- dynamic connection / dynamic SQL is out of static-analysis scope

また、`inner_sql` に `ORDER BY` があり、outer query に `ORDER BY` がない場合は
structured warning を出します。ただし `LIMIT 1` は inner ordering が意味を持つので
warning から除外します。

### Federation compatibility matrix

DESIGN の matrix を severity/remediation 付きに更新しました。

実装面では以下を反映済みです。

- `TIMESTAMP` output は nanosecond truncation warning。
- `ARRAY<T>` は element type を再帰的に評価。
- Spanner `STRUCT` output は BigQuery `EXTERNAL_QUERY` output として reject。
- `PROTO` / `ENUM` は Spanner inner scope のみ有効で、BigQuery output には出せない
  前提を維持。

### Replace semantics

README/DESIGN で mutation `replace` の危険な semantics を強調しました。

- mutation `Replace` は update mask ではない。
- 既存行があれば delete + insert。
- 明示されない columns は `NULL` になる。
- interleaved child table の `ON DELETE CASCADE` があると child rows も削除され得る。
- unspecified columns を残したい場合は `insert_or_update` / future structured upsert を
  使うべき。

### Standard DML vs Partitioned DML

DESIGN に、Standard DML と Partitioned DML は別 surface として扱うべきだと明記しました。
現時点では Standard DML helper のみを対象にし、Partitioned DML は将来の opt-in
execution helper として扱います。

### `result: one` / `maybe_one`

まだ method generation は実装していませんが、DESIGN に以下の error taxonomy を
追加しました。

- `ErrNoRows`
- `ErrTooManyRows`
- wrapped analyzer/client/decoding errors

`maybe_one` の戻り値形は pointer policy / value policy を config で決めるべきで、
query ごとに曖昧に推測しない方針にしました。

### Type override timing

レビューの指摘どおり、query methods の前に最小限の override が必要になりやすいので、
roadmap に Phase 2.5 を追加しました。

Phase 2.5 の対象:

- field rename
- scalar type override
- custom nullable type の Spanner/BigQuery runtime contract validation

proto/enum/nested field などの広い override は、より後の Phase 5 に残しました。

### Other design clarifications

以下も DESIGN に追加しました。

- compact config から structured config への migration map。
- generated name collision rules。
- read/write DTO runtime contract matrix。
- BigQuery Spanner external datasets は initial non-goal。
- vet suppression の optional `owner` / `expires` は plan に残す。

## Deferred intentionally

### Full structured write config

`insert.columns` / `update.mask` / `conflict.strategy` の structured config はまだ
実装していません。今回の compact config では validation を強化し、危険な upsert を
早めに error にするところまでに留めました。

理由: structured config は loader/schema migration/README examples/generated code の
まとまった変更になるため、現在の patch では plan と validation の事故防止を優先しました。

### Query method generation

`result: one` / `maybe_one` の actual generated method はまだ実装していません。
ただし、plan と DESIGN の contract は先に固定しました。

理由: method generation は Spanner client / transaction interface / pointer policy /
parameter struct inference を同時に決める必要があり、今回の review response の範囲より
大きいです。

### Full vet rule engine

structured warnings と suppressions は plan に入りましたが、rule registry や
severity-based fail policy はまだ実装していません。

理由: まず plan schema を機械可読にして、後続で `vet --strict` や per-rule policy を
載せる方が安全です。

### Partitioned DML execution helpers

DESIGN には入れましたが、実装はしていません。

理由: 現在の generated DML helper は `spanner.Statement` を返すだけなので、Partitioned
DML execution semantics を混ぜる段階ではありません。

## Disagreements or adjusted interpretations

大きな反論はありません。ただし、以下は少し狭く解釈しました。

- `outer_sql` の placeholder は、今は table-expression placeholder として exactly-once
  だけを実装しました。FROM-position や alias shape まで SQL parser で厳密検証するのは、
  BigQuery analyzer が最終的に検証するため現時点では過剰だと判断しました。
- type override は Phase 2.5 に前倒ししましたが、proto/enum/nested field まで一気に
  実装するのは deferred としました。
- warnings は plan に構造化しましたが、今は warning list であり、vet failure policy までは
  入れていません。

## Validation performed

- `go test ./...`
- `mise x -- golangci-lint run --timeout=5m`
- `git diff --check`
