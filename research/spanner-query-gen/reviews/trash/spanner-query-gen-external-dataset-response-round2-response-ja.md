# `spanner-query-gen` external dataset round 2 review response

## Summary

追加レビューありがとうございます。今回の指摘はほぼ採用します。

特に、BigQuery Spanner external dataset を `EXTERNAL_QUERY` の別記法にせず
BigQuery catalog binding として扱う方針は維持しつつ、catalog binding の
plan 表現をより明示的にします。external dataset support は利用者要件として
Current Scope に残します。

## 採用する点

### 古い alias 表記の整理

README に残っている `external_schemas` 主体の説明は修正します。
推奨名は `external_query_connections` とし、`external_schemas` は compact
legacy alias としてだけ触れます。

`columns` alias や `auto_all_non_key_columns` sentinel についても、pre-stable
期間だけの暫定形であり、structured config では削除または置換することを
roadmap に明示します。

### `outer_sql` の配置

`outer_sql` は `queries[].federated.outer_sql` に固定します。

raw BigQuery SQL query には意味を持たない field なので、`queries[]` 直下ではなく
`federated` block に所有させます。DESIGN の例もこの形に揃えます。

### external dataset reference の正規化

`spanner_external_datasets[].dataset` は loader では従来どおり string を
受け付けますが、plan では canonical BigQuery dataset reference を保持します。

追加で次を扱います。

- `schemas[].project` as the BigQuery source default project
- `spanner_external_datasets[].project` as an explicit project override
- `dataset` or `project.dataset` input
- project-qualified table path and unqualified table path aliasing when a default project is known

これにより、plan review では `project`, `dataset`, canonical `path` を見られるようにします。

### unsupported column と named schema の policy 化

`EXTERNAL_QUERY` output と external dataset projection は severity を分けます。

`EXTERNAL_QUERY` output で BigQuery に返せない型が出た場合は query failure として
error にします。一方、external dataset projection は BigQuery の実挙動に近い
`omit` を default にし、CI で厳密にしたい場合は `unsupported_columns: error` を
指定できるようにします。

Named Spanner schema についても、default は `warn_and_omit` とし、
`named_schema_policy: error` を opt-in で用意します。

### access / Data Boost metadata

generator が dataset、connection、IAM、Terraform を作らない方針は維持します。
その代わり、plan の catalog binding に次の metadata と warning を残します。

- access mode
- optional database role
- access verification status
- Data Boost access requirement
- read-only execution metadata

これは実インフラ状態を検証したという意味ではなく、DDL ベース projection の前提を
review できるようにするための metadata です。

### `vet` warning UX

現時点の `vet` は planning error で fail する検証コマンドですが、warning を見るために
常に `explain-plan` を別実行するのは CI UX として弱いです。

`vet` でも warning summary を stderr に出し、machine-readable output も用意します。
full rule engine、`--strict`、warnings-as-errors は引き続き future work とします。

## 一部調整する点

### external dataset unsupported columns の default

レビューでは default `error` も候補として挙げられていましたが、ここは default
`omit` にします。

理由は、BigQuery Spanner external dataset 自体が unsupported Spanner columns を
BigQuery 側で accessible にしない仕様なので、static catalog projection の default は
実サービスに寄せる方が自然だからです。

ただし silent omission は危険なので、plan warning は必ず出し、
`unsupported_columns: error` で CI を厳格化できるようにします。

### BigQuery から見えない key metadata

underlying Spanner primary key を DTO nullability や write planning の根拠に使う可能性は
ありますが、BigQuery catalog metadata として存在するものとは扱いません。

この provenance separation は設計文書に残します。実装上の field-level provenance は
plan model 全体の拡張と合わせて進めます。

## 今回すぐ実装する範囲

- README の古い `external_schemas` 説明を修正する。
- README/DESIGN の external dataset example を project-qualified quoted table path に寄せる。
- DESIGN の `outer_sql` example を `federated.outer_sql` に揃える。
- external dataset binding に canonical dataset reference、projection policy、access/Data Boost metadata を追加する。
- `unsupported_columns: omit|error` と `named_schema_policy: warn_and_omit|error` を追加する。
- default schema 以外の Spanner tables を warning付きで plan に出す。
- `vet` で warning summary と machine-readable output を出せるようにする。
- regression tests を追加する。

## 後続に残す範囲

- field-level provenance and nullability reason model
- BigQuery optimizer metadata と underlying Spanner metadata の分離を plan fields に反映すること
- full `vet` rule engine
- warnings-as-errors / `--strict`
- generated query method execution layer
- read-only DML rejection を analyzer-level diagnostic として体系化すること

## Closing

external dataset support はこのツールの DDL-first static analysis という境界と相性が良いと
判断しています。ただし、query text に明示的な `EXTERNAL_QUERY` が出ない分、plan の
catalog binding 表現がより重要になります。

今回の対応では、dataset reference、visible tables、omitted columns、named schema policy、
access assumption、Data Boost requirement を plan に残し、外部インフラを作らない
generator の責務境界を保ったまま利用できる形に寄せます。
