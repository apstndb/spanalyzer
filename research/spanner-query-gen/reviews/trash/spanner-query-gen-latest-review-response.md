# Response to Latest `spanner-query-gen` Review

レビューありがとうございます。大きな方向性については同意で、今回の指摘は「設計思想の変更」ではなく「採用時に事故りやすい細部を明文化するもの」として取り込みました。この repo はまだ正式リリース前なので、互換性維持よりも危険な default や曖昧な alias を早めに潰す方針にします。

## Reflected Changes

- DDL-first deterministic workflow を DESIGN に明記しました。将来 live introspection を追加しても、通常の生成 workflow の source of truth にはしません。
- DML `replace` は mutation-only として扱う方針に修正しました。Cloud Spanner Go client の mutation `Replace` は存在しますが、公式 DML grammar では top-level `INSERT OR REPLACE` を確認できないため、DML helper 生成は拒否します。
- `update_mask` 省略時の全 non-key column 更新は、長期 default として危険と判断しました。structured config では explicit update mask required、全列更新は `auto_all_non_key_columns` の明示 opt-in に寄せます。
- shared DTO の read result と write input を分けて検証する設計にしました。plan に `decode_result` / `encode_key` / `encode_insert_value` / `encode_update_value` のような field role を持たせます。
- BigQuery federation の Spanner-to-BigQuery type compatibility matrix を DESIGN に追加しました。特に `STRUCT` は federated output として reject、`TIMESTAMP` は nanosecond truncation warning、`PROTO` / `ENUM` は inner Spanner scope のみ有効という扱いにします。
- `EXTERNAL_QUERY` は raw BigQuery SQL を残しつつ、将来の `federated` / `external_query` config で inner Spanner SQL を別に書ける形を設計に追加しました。生成 constants も inner SQL と escaped outer SQL を分ける方針です。
- parameter は analyzer inference を基本とし、config `params` は override / disambiguation として扱う方針にしました。outer BigQuery params と inner Spanner params の境界も federation config 側で明示します。
- `result: one` は exactly one row、`maybe_one` は zero rows allowed、`many`、`exec` は row-count-only という cardinality semantics を DESIGN に追加しました。DML `THEN RETURN` は別概念として残します。
- `vet` suppression は最初から reason 必須にする設計にしました。
- type override の解決順序と nullable custom type strategy を DESIGN に追加しました。
- Spanner index shorthand は、index key columns に加えて base table primary key columns と `STORING` columns を選択する方針に修正しました。
- `explain-plan` は human summary と stable YAML/JSON output を分ける方針にしました。

## Immediate Implementation Adjustments

レビューの P0 のうち、現在の compact 実装でも危険だったものはすぐ直しました。

- `replace` write は default で mutation helper のみ生成し、`methods: [dml]` は error にしました。
- DML `INSERT OR REPLACE` の生成を停止しました。
- Spanner index shorthand が base table primary key columns を SELECT に含めるようにしました。
- `update` / `insert_or_update` の暗黙全 non-key column 更新を停止し、明示 `update_mask` を必須にしました。全 non-key column 更新は `update_mask: [auto_all_non_key_columns]` の明示 opt-in にしました。
- 既存 `result_struct` を write `input_struct` として再利用する場合、必要な key/value column が既に result DTO に存在する場合だけ許可し、write-only field を result DTO に黙って追加しないようにしました。
- `federated` query config を追加し、inner Spanner SQL を BigQuery string literal に埋め込まず config 上で review できるようにしました。生成 Go では inner Spanner SQL と outer BigQuery SQL の constants を分けます。
- 生成される mutation / DML helper に、同一 transaction での混在に注意する doc comment を追加しました。
- `spanner-query-gen explain-plan` を追加し、resolved generation plan を summary / YAML / JSON で出せるようにしました。table/index/federated SQL expansion、result fields、merged structs、write column choices を code rendering 前に確認できます。
- `spanner-query-gen vet` を追加し、同じ plan validation を code rendering なしで CI から実行できるようにしました。
- query `result` mode (`one` / `maybe_one` / `many` / `exec`) を config と plan に追加し、不正値を検証するようにしました。
- query params を plan に出すようにし、重複 param を reject します。index shorthand の自動 key params と同名の明示 params は override として扱います。
- `vet.disable` には `reason` を必須にしました。
- BigQuery `EXTERNAL_QUERY` rewrite で Spanner `STRUCT` output を reject するようにしました。公式 mapping と同様、`STRUCT` は Spanner federated query output として扱いません。
- separated `federated` config の `explain-plan` では、Spanner `TIMESTAMP` output が BigQuery 側で nanoseconds truncation を受ける warning を出します。
- README の current compact config 説明も、DML `replace` unsupported と index shorthand の base PK handling に合わせて更新しました。

## Deferred or Intentionally Not Expanded

- Broad CRUD generator / ORM / runtime query builder 方向には寄せません。
- BigQuery query execution methods は、DTO loading と TableSchema metadata が安定するまで後回しにします。
- sqlc macro compatibility は約束せず、GoogleSQL-native で reviewable な最小 annotation に留めます。
- DML `THEN RETURN` は `result: exec` に混ぜず、Spanner query method surface が固まった後に別扱いで設計します。
- Template customization / plugin system は、plan model が安定してから検討します。

## References Checked

- Cloud Spanner DML syntax: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax
- Cloud Spanner DML versus mutations: https://docs.cloud.google.com/spanner/docs/dml-versus-mutations
- Cloud Spanner secondary indexes: https://docs.cloud.google.com/spanner/docs/secondary-indexes
- BigQuery federated query functions and Spanner type mapping: https://docs.cloud.google.com/bigquery/docs/reference/standard-sql/federated_query_functions
