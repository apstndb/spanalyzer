# Handoff — spanner-emulator-survey

## Retained in spanalyzer (2026-08-25)

The final tracked source tree is retained as the independently testable
`github.com/apstndb/spanalyzer/survey` nested module. The unpublished legacy
repository's 27-commit history is intentionally not published. The strict
[`import-provenance.json`](import-provenance.json) maps legacy commit
`91908d001349f844aac070cc6518119c0e3c36c0` and tree
`34d63cf89aaf885cbfd8069e91c4ead707b048c8` to the exact initial spanalyzer
import commit and subtree.

Ignored agent/runtime directories, the local managed-database locator, and the
untracked memefish feedback scratch file were excluded. The scratch file's two
actionable upstream issues are closed. Once the integration commits are present
on remote `main`, the former local checkout is not required for builds, tests,
manifest regeneration, provenance checks, or retained knowledge and can be
deleted separately with explicit authority.

Last reviewed: 2026-08-25 / all repository-local release-note and live-parity
follow-ups are implemented and committed. The latest work covers named-schema
DDL, omitted and zero-column primary keys, rolling metadata columns, metadata
type/ordinal validation, nullable locality options, vector-index filters,
view-backed property graphs, omitted change-stream retention, generic options,
live `SPANNER_SYS` audit, UUID metadata, live proto/enum and function types,
search-index type normalization, empty graph-label properties, and an aggregate
managed canonical-DDL comparison.

The managed canonical set also proves that `GetDatabaseDdl` can retain an
`ALTER TABLE ... ADD CONSTRAINT` statement. AST-to-schema conversion accepts
check and foreign-key additions and folds them into reconstructed `CREATE TABLE`;
all other `ALTER TABLE` variants remain fail-closed.

Managed Spanner and Omni `2026.r2.1-beta` both passed the current
`SPANNER_SYS` audit at 50 advertised tables / 539 columns against the 51-table
package superset. The extra table is the officially documented Enterprise-only
`VECTOR_INDEX_STATS`, which neither tested target advertised. Managed Spanner,
Omni, and Emulator v1.5.56 agree on the UUID metadata contract: `DATA_TYPE` is SQL NULL,
`SPANNER_TYPE = UUID`, and `COLUMN_DEFAULT = NEW_UUID()`. Managed and Omni
expose custom dictionary/table option rows; index `columnar_policy` remains in
canonical DDL without an `INDEX_OPTIONS` row. Emulator v1.5.56 has the narrower
option surface recorded in the regression tests.

The latest managed canonical comparison completes successfully: 330 canonical
statements, one memefish parse error in the `CREATE TABLE` family, and 341
reconstructed statements. Parsed family counts match for change streams,
functions, regular/search indexes, property graphs, proto bundles, roles,
schemas, sequences, views, and grants. At the family-count layer, canonical
`ALTER DATABASE` and `ALTER TABLE ... ADD CONSTRAINT` have no generated ALTER
counterpart, while reconstruction adds eleven `ALTER STATISTICS` statements and
one locality group, placement, and table. `dd06e8a` preserves the added
constraint semantically by folding it into the generated `CREATE TABLE`.
Object identifiers are intentionally not retained or printed.

Primary-key omission remains environment-specific. Managed Spanner and Omni
create the documented hidden identity `rowid`; Emulator v1.5.56 rejects omission.
All three accept explicit `PRIMARY KEY ()` as a singleton table, with a small
metadata-shape difference that the converter handles. Nullable locality-group
option values are omitted during reconstruction, and malformed emulator
`inflash = 'BOOL'` metadata now fails closed.

Remaining work is external or ongoing maintenance: memefish lacks the AST shape
for additional vector-index keys, model privileges, named-schema property
graphs, and qualified targets for some identifier-only families; current
`INFORMATION_SCHEMA` cannot recover sequence/schema grants, placement keys,
optionless locality groups, or exact remote-function clauses. Omni commercial
licensing, standalone VM packaging, and backup migration remain outside this
container-backed schema-survey scope.

Final verification on 2026-08-25: uncached `mise run test-all` passed, including
Emulator v1.5.56 and Omni integration tests; uncached managed drift passed with
50/539 advertised `SPANNER_SYS` metadata against the 51-table superset and 28
decoded sample rows; the explicit Omni drift gate passed with the same 50/539
surface and three decoded rows. The managed UUID fixture passed with cleanup,
and the read-only canonical comparison reproduced the 330/1/341 aggregate
above. `mise run lint` reported zero issues.

このリポジトリは Cloud Spanner の `INFORMATION_SCHEMA` / `SPANNER_SYS` を
Go struct 化し、memefish AST と双方向変換することで DDL を round-trip させる
プロジェクトです。設計の前提とアーキテクチャは `AGENTS.md` を参照してください。
本ドキュメントは「これまで何をしてきたか・今どこで止まっているか・次に何をすべきか」だけを述べます。

## 現在の状態（一行サマリ）

- latest published emulator は v1.5.56（linux/arm64 tested digest
  `sha256:18a56fd557011e50e1733a9232e8d17ec9bdd7e51f6cf7660f14c234479f4f36`）。
  metadata drift test もこの image を明示的に使う。
- rolling `INDEXES.SEARCH_UNNEST` / `INDEX_COLUMNS.EXPRESSION` は nullable
  superset fields として登録し、実際に query できる column だけを選ぶ probe を実装済み。
- `PARAMETERS` / `ROUTINES` の6件の ordinal mismatch は修正済みで、drift gate は
  name/type/ordinal を独立に検証する。
- real / Omni の `SPANNER_SYS` は 50 advertised tables / 539 columns で、
  query可能な sample rows の decode も成功。package superset は documented
  Enterprise-only `VECTOR_INDEX_STATS` を加えた51 tables。
- current real `GetDatabaseDdl` は330 statements / memefish parse 329、
  `LoadSchema` → `ToDDLStatements` は341 statementsまで完走する。
- locality option の SQL NULL は omission として扱い、emulator の malformed
  `inflash = 'BOOL'` は fail closed。nullable row による loader blocker は解消済み。
- memefish v0.8.1 に更新済み。新規 AST surface の `PLACEMENT KEY` と
  `GRANT` / `REVOKE ... ON [ALL] SEQUENCE(S)` を `astconv` に配線済み。
- `astconv` へ v0.8.0 の anonymous constraint style `PRIMARY KEY` と
  `GRANT ... ON ALL ... IN SCHEMA` を配線済み。
- named-schema table / regular and search index / view / sequence / scalar
  function と path-backed grant/revoke を双方向に配線済み。
- `SCHEMATA.PROTO_BUNDLE` は全 non-empty row の descriptor type を union /
  deduplicate / sort して database-scoped DDL を復元する。
- claude-code / codex のレビューで見つかった P1/P2 問題（model column panic、function silent loss、
  self-referential FK、空 privilege emission、LoadSchema snapshot consistency）を修正済み。
- 残課題は sequence/schema grant / placement key の live metadata 復元、
  live metadata だけでは clause を再構成できない remote UDF、
  optionless locality group の INFORMATION_SCHEMA 復元、エミュレータからは復元できない
  一部 options、および upstream AST/service boundary の外側にある family。

## 完了済みの作業

### モジュール基盤
- `go.mod` 初期化（`github.com/apstndb/spanner-emulator-survey`, Go 1.25.0）。
- 依存: memefish は v0.8.1 に固定。v0.8.1 では `PLACEMENT KEY` と sequence grant/revoke、
  v0.8.0 では anonymous constraint style `PRIMARY KEY`、
  `GRANT ... ON ALL ... IN SCHEMA`、`ParseSchemaType` の export などが追加された。
  v0.6.2 からの `*ast.Path` 化などの API 非互換は v0.7.0 で吸収済み。
- `cloud.google.com/go/spanner`, `cloud.google.com/go/civil`, `apstndb/spanemuboost` を採用。
- `mise` でタスク管理（`mise.toml`: `lint` / `build` / `test` / `test-drift-*` / `test-all` /
  `run-roundtrip`）。本番接続先などの秘匿値は gitignore 済みの `mise.local.toml` `[env]` に置く。

### spannertype/
- `ParseSchemaType(s) (ast.SchemaType, error)`: memefish v0.8.0 が export した
  `memefish.ParseSchemaType` を使う。逆方向は `ast.SchemaType.SQL()`。

### infoschem/
- 48 個の `INFORMATION_SCHEMA` テーブルに対応する Go struct（superset 形式）。
- `TableMeta` レジストリ + `DiscoverColumns(ctx, client)` による **Dynamic Column Discovery**。
  実行時に `INFORMATION_SCHEMA.COLUMNS` を読み、対象 DB に実在するカラムだけを SELECT するため、
  Emulator/Omni/Real の差をハードコードせず安全にクエリできる。registry 未定義のカラムを
  検知すると `cmd/roundtrip` と drift テストが WARNING を出す。
- 2026-05-09（emulator v1.5.53）の実測で `COLUMNS.ON_UPDATE_EXPRESSION` と `INDEXES.FILTER` が
  emulator にも入ったため、`meta.go` と関連テストを更新済み。
- 2026-07-03 に emulator を v1.5.55 に更新。v1.5.55 で `SCHEMATA.PROTO_BUNDLE` が復活したこと、
  および実機に `TABLE_OPTIONS` が存在することを確認し、`AllTableMetas()` に `TABLE_OPTIONS` を追加。

### spannersys/
- 51 個の known `SPANNER_SYS` テーブルを interval 共通化して ~25 struct に集約。
  tested real / Omni surface は50で、追加の `VECTOR_INDEX_STATS` は
  Enterprise-only の公式文書に基づく superset entry。
- ARRAY<STRUCT<...>> 用の共有 struct（`LatencyDistribution` 等）を `types.go` に。
- `ACTIVE_PARTITIONED_DMLS.TEXT_FINGERPRINT` and `PROGRESS` decode as strings,
  matching the managed and Omni metadata and expression-analysis evidence.
  Current official documentation still describes numeric types, so that
  disagreement remains documentation evidence rather than a decoder contract.
  `OPERATIONS_BY_TABLE.TABLE_NAME` and the locality-group table-size storage
  fields likewise follow the observed decode shape. Row-decoding tests cover
  these contracts, and interval helpers accept only the known values.
- `spannersys.Audit` retains each advertised table, column, raw `SPANNER_TYPE`,
  and ordinal tuple; compares names and relative declaration order; and checks
  query/decode for at most one row per table. Registry-only entries from a
  successful capture are reported separately as known absent. Managed and Omni
  both pass at 50 advertised tables / 539 columns with eight known-absent
  registry entries against the 51-table superset.
- The private `spannersys` descriptor extractor derives 51 tables / 547 columns
  and 14 canonical type shapes from the decoder structs. The renderer preserves exact
  nested field tags and `STRING(MAX)` spellings, unwraps pointers without
  changing structural types, and compares every advertised managed/Omni raw
  type without relying on the function-type parser.
- `spannersys.ExportManifest` combines that structural superset with two
  redacted, embedded 2026-08-25 captures. Managed Spanner and Omni
  `2026.r2.1-beta` agree on all 50 advertised tables / 539 columns; the eight
  registry-only columns remain `absent_both` and non-projecting. The exporter
  requires an explicit exact source commit, performs no Git lookup or live
  query, preserves current official-document conflicts as sidecar evidence,
  and hashes only required targets plus the graded table content. The survey
  repository intentionally does not check in a full manifest: a downstream
  consumer exports and pins it from the exact producer commit.

### astconv/
ファイル対応関係（`to_ast_*.go` = struct → AST、`from_ast_*.go` = AST → struct）:

| DDL family | to_ast | from_ast |
|---|---|---|
| CREATE TABLE / 制約 / column options | `to_ast_table.go` | `from_ast_table.go` |
| CREATE INDEX / SEARCH INDEX | `to_ast_index.go` | `from_ast_index.go` |
| CREATE VECTOR INDEX | `to_ast_vector_index.go` | `from_ast_vector_index.go` |
| CREATE VIEW | `to_ast_view.go` | `from_ast_view.go` |
| CREATE CHANGE STREAM | `to_ast_changestream.go` | `from_ast_changestream.go` |
| CREATE SEQUENCE | `to_ast_sequence.go` | `from_ast_sequence.go` |
| CREATE MODEL | `to_ast_model.go` | `from_ast_model.go` |
| CREATE PROPERTY GRAPH | `to_ast_graph.go` | `from_ast_graph.go`（JSON ↔ AST） |
| CREATE PLACEMENT | `to_ast_placement.go` | `from_ast_placement.go` |
| CREATE SCHEMA | `to_ast_schema.go` | `from_ast_schema.go` |
| CREATE ROLE / GRANT / REVOKE | `to_ast_role.go` | `from_ast_role.go` |
| ALTER DATABASE | `to_ast_database.go` | `from_ast_database.go` |
| ALTER STATISTICS | `to_ast_statistics.go` | （from は no-op） |
| CREATE FUNCTION（SQL UDF / Remote UDF） | `to_ast_function.go` | `from_ast_function.go` |
| CREATE PROTO BUNDLE | `to_ast_proto_bundle.go` | `from_ast_proto_bundle.go` |
| CREATE LOCALITY GROUP | `to_ast_locality_group.go` | `from_ast_locality_group.go` |

### 個別の機能対応・修正（双方向）
- **memefish HEAD 追従**: `*ast.Path` 化した `CreateSearchIndex.Name/TableName`,
  `PrivilegeOnTable.Names`, `SelectPrivilegeOnChangeStream.Names` をヘルパ `path()` 経由に統一。
  末尾識別子は `helpers.go` の `leafName()` で取得するように整理済み。
- **`GRANT SELECT ON VIEW` 区別**: `ROLE_TABLE_GRANTS` の table/view を `Schema.Views` で判別し、
  view は `SelectPrivilegeOnView` を発行（`GetDatabaseDdl` 出力に一致）。
- **`ForeignKey.Enforcement`**: `Enforced == "NO"` を `ast.NotEnforced` に。
- **`AutoIncrement` / `IdentityColumn`**: `ColumnDefault == "AUTO_INCREMENT"` を `ast.AutoIncrement`、
  `IdentityColumn` の `SkipRange` / `StartCounterWith` / `BitReversedPositive` を双方向対応
  （param 無は `Rparen = token.InvalidPos`、有は `token.Pos(1)`）。
- **`CreateSequence` の Params 化**: `sequence_kind` / `skip_range_min,max` / `start_with_counter` を
  Params に展開、未知 option は `Options` に残す。
- **`CREATE PROTO BUNDLE` 復元**: `SCHEMATA.PROTO_BUNDLE` の `FileDescriptorSet` を deserialize し、
  top-level / nested message / enum の型名を復元。明示的 `Schema.ProtoBundleTypes` があればそちら優先。
- **`ON UPDATE (expr)`**: `COLUMNS.ON_UPDATE_EXPRESSION` を `CREATE TABLE` 列定義に双方向マップ。
- **`CREATE TABLE ... OPTIONS(locality_group=...)`**: AST round-trip 対応（`TableOption` struct +
  `Schema.TableOptions`）。INFORMATION_SCHEMA からの復元不可は「残課題」参照。
- **`CREATE PROPERTY GRAPH`（default schema）**: `PROPERTY_GRAPH_METADATA_JSON` ↔ AST を双方向対応。
- **search index metadata 配線**: `INDEXES.FILTER` ↔ `CreateSearchIndex.Where`、
  `INDEXES.SEARCH_ORDER_BY` ↔ `CreateSearchIndex.OrderBy`。
- **ラウンドトリップテスト拡充**: `CreateModel` / `CreateFunction` / `CreatePlacement` /
  `CreateLocalityGroup` / `CreateRole`+`Grant` / `CheckConstraint` / `ForeignKey` /
  `GeneratedColumn` / `RowDeletionPolicy` を追加。過程で `ROW DELETION POLICY` の
  AST→Schema 変換のプレフィックスバグを修正。
- **公開ローダー**: `astconv.LoadSchema(ctx, client)` を追加。`INFORMATION_SCHEMA` への
  クエリ・未知カラム警告・各種 struct へのロードを一箇所に集約。
- **cmd/roundtrip 簡潔化**: `cmd/roundtrip/main.go` から独自のクエリコードを削除し、
  `astconv.LoadSchema` を使うようにリファクタリング。
- **E2E ローダーテスト**: `astconv/loader_test.go` にエミュレータ起動 → `LoadSchema` →
  `ToDDLStatements` → 期待 DDL 含むことを検証するテストを追加。
- **named-schema 対応状況文書化**: `UNSUPPORTED_DDL.md` に DDL ファミリごとの
  named-schema 対応状況表を追加。
- **memefish v0.8.0 へ更新**: `go.mod` を v0.8.0 に更新。ビルド・テスト共に通過。
- **memefish v0.8.1 へ更新**: `ColumnDef.PlacementKey`、`PrivilegeOnSequence`、
  `PrivilegeOnAllSequencesInSchema` を AST-only state として双方向に配線。実 Spanner の
  INFORMATION_SCHEMA には placement-key 属性と role-sequence-grant table がないため、
  `LoadSchema` からの復元は引き続き不可。
- **anonymous constraint style `PRIMARY KEY`**: `ast.TablePrimaryKey` を `fromCreateTable` で
  処理し、primary key 情報が欠落しないように対応。
- **`GRANT ... ON ALL ... IN SCHEMA`**: `PrivilegeOnAllTablesInSchema` /
  `SelectPrivilegeOnAllChangeStreamsInSchema` / `SelectPrivilegeOnAllViewsInSchema` を
  `fromGrant` / `toRolesDDL` で扱えるようにし、`Schema.AllSchemaGrants` を追加。
- **`spannertype.ParseSchemaType` 簡素化**: memefish v0.8.0 で export された
  `memefish.ParseSchemaType` を使うように変更。
- **フィードバックメモ**: `MEMEFISH_FEEDBACK.md` に memefish への upstream フィードバック
  （temporary、コミット対象外）を作成。
- **レビュー後修正（claude-code / codex）**:
  - `to_ast_model.go` で `CreateModelColumn.DataType` を設定し panic を修正。
  - 同ファイルで捨てていた per-column model OPTIONS を復元。
  - `to_ast_function.go` で `ParseExpr` エラーを飲み込まないように修正。
  - `to_ast_table.go` / `from_ast_table.go` で self-referential FK の参照テーブル解決を修正。
  - `to_ast_role.go` で無効な `AllSchemaGrant` に対してエラーを返すように修正。
  - `LoadSchema` / `LoadSchemaFromDiscovered` を単一 `ReadOnlyTransaction` で実行するように修正。
  - 不要になった `fromCreateTablePK` とその `init()` スタブを削除。
  - `AGENTS.md` のバージョン表記を v0.8.0 / emulator 1.5.55 に更新。
  - 上記をカバーする回帰テストを追加。
- **ドキュメント**: `CLAUDE.md` を `AGENTS.md` にリネームし `CLAUDE.md` は `@AGENTS.md` 参照に
  （他エージェント対応）。
- **検証基盤**: `infoschem/drift_test.go` の `TestDrift_{Emulator,Omni,Real}TableMetas` で
  各環境の実カラムと `AllTableMetas()` を突合。`mise run test-drift-real` は
  `TEST_REAL_SPANNER_DATABASE`（`mise.local.toml`）を必要とし、未設定時に失敗する
  separate required gate。
- **今回の remediation**:
  - `ROLE_COLUMN_GRANTS` を使う column-scoped privileges と matching `REVOKE` を実装。
  - model grants と schema metadata を持つ named-schema family は silent loss ではなく明示 error に変更。
  - `LoadSchemaFromDiscovered` は caller map を信用せず、同一 read transaction で再 discovery。
  - locality option value の double quoting をなくし、AST 側は optionless locality group を保持。
  - metadata accessor は defensive copy にし、linter を 2.12.2 に固定。
- **2026-07-26 repository-local implementation**:
  - named-schema table / index / search index / view / sequence / scalar function を
    dependent metadata と qualified dependency ordering を含めて双方向対応。
  - table/view/sequence/change-stream/table-function grant/revoke の qualified target を保持。
  - named scalar function の実 Spanner acceptance と routine metadata schema を確認。
  - `PROTO_BUNDLE` の non-empty `SCHEMATA` row を union / deduplicate / sort。
- **2026-07-26 named-schema review remediation**:
  - same-leaf の child table が別 schema にある cross-schema FK で、内部 synthesized
    referenced-key identity が衝突する問題を child schema 込みの key に修正。
  - named parent/child interleave、non-interleaved regular index、view、sequence を
    emulator-backed `LoadSchema` → AST fixture に追加。
  - qualified regular-index `INTERLEAVE IN` を emulator v1.5.55 と実 Spanner の raw DDL
    で確認し、service は受理するが memefish v0.8.1 AST では表現できないと分類。
- **2026-08-24 to 2026-08-25 release-note follow-up**:
  - vector-index filter、view-backed property graph、change-stream default omission、
    generic options、UUID、primary-key omission/empty key を実装・実測。
  - rolling columns と metadata type/ordinal drift を loader gate に組み込み。
  - nullable locality options、live proto/enum types、search-index type、empty graph
    labels、function-specific types を修正し、managed canonical comparisonを完走。

## 残課題

### memefish v0.8.1 で AST 対応、live metadata からは復元不能

- **sequence grant**: 個別と `ON ALL SEQUENCES IN SCHEMA` の AST round-trip / REVOKE は対応。
  実 Spanner に専用 INFORMATION_SCHEMA table がないため `LoadSchema` では復元不能。
  `GetDatabaseDdl` は populated schema の wildcard grant を個別 sequence grant に展開し、
  empty schema では何も返さないため、原文の wildcard intent は live source から復元できない。
- **`PLACEMENT KEY`**: AST round-trip は対応。実 Spanner の `COLUMNS` に対応属性がないため
  `LoadSchema` では復元不能。

### memefish v0.8.0 で対応済み、`astconv` への配線も完了
- **anonymous constraint style `PRIMARY KEY`**: `fromCreateTable` が `ast.TablePrimaryKey` を処理。
- **`GRANT ... ON ALL ... IN SCHEMA`**: `fromGrant` / `toRolesDDL` が対応（#335 解消）。

### 部分対応・INFORMATION_SCHEMA から復元不能
- **`GRANT USAGE ON SCHEMA`**: AST round-trip は可能。`INFORMATION_SCHEMA` に対応するテーブルが
  エミュレータに存在しないため、INFORMATION_SCHEMA→DDL 方向の復元は未対応。
- **named-schema property graph**: memefish の graph AST が修飾名（`PROPERTY_GRAPH_SCHEMA` /
  `baseSchemaName` 非空）を受けられず、default schema 限定。
- **table-level `OPTIONS(locality_group=...)`**: 実機の `INFORMATION_SCHEMA.TABLE_OPTIONS` から復元可能になった。
  エミュレータはまだ `TABLE_OPTIONS` を出さないため、エミュレータ→DDL 方向は復元不可（AST round-trip は可能）。
  column-level `locality_group` は `COLUMN_OPTIONS` で取得可能。
- **named-schema の残存 family**: vector index / change stream creation / model /
  property graph / placement / locality group / statistics package は、現行 service の
  named-schema 対象外または未確認であり、memefish AST の object name も修飾名を保持できない。
  silent dequalification はせず明示 error のままにする。
- **model grants**: `ROLE_MODEL_GRANTS` は memefish AST node 不足のため明示 error。

## 次に着手するなら

1. service が named-schema 対象を追加した場合に限り、identifier-only AST family の
   upstream path support と repository conversion を再評価する。
2. memefish に model privilege node が追加されたら `ROLE_MODEL_GRANTS` を配線する。
3. schema/sequence grant、`PLACEMENT KEY`、optionless locality group、remote UDF の
   exact mode を公開する INFORMATION_SCHEMA surface が追加されたら `LoadSchema` に配線する。

## 検証方法

```sh
mise run test-all         # default local gate: lint + build + test（real drift の実行は保証しない）
mise run run-roundtrip     # E2E デモ: emulator → INFORMATION_SCHEMA → DDL
mise run test-drift-real   # separate required real-Spanner drift gate（env 未設定なら失敗）
mise run test-uuid-real    # managed UUID fixture（作成物は cleanup）
mise run test-canonical-real # read-only aggregate canonical comparison
```

2026-08-23 のlatest validationでは、GitHub release/APIからv1.5.56がlatest stable、
Go module resolutionからmemefish v0.8.1がlatestと再確認。local checkoutにはremoteが
なく、想定GitHub repository APIも404だったため、hosted mainとの比較は行っていない。

cache無効のrequired real drift gateは、新規`INDEXES.SEARCH_UNNEST`のためfailした。
expanded probeでは`INDEX_COLUMNS.EXPRESSION`も観測。前者は`ARRAY<STRING(MAX)>`,
nullable, ordinal 14、後者は`STRING(MAX)`, nullable, ordinal 11。12個の独立single-session
clientでは各columnのmetadata広告が9/12、query成功が5/12と8/12で一致せず、同一
read-only transactionでも`LIMIT 0`成功後のaggregateが`Unrecognized name`になるsampleが
あった。query可能時は`SEARCH_UNNEST` 329 rows中323 NULL/6 empty arrays、
`EXPRESSION` 598 rows全てNULL。公式Information Schema page（API update time
2026-08-15）とcurrent emulator master metadata CSVは両columnをまだ掲載/実装しない。

real locality NULLとloader failure、emulator v1.5.56の`storage = "hdd"`から
`inflash = "BOOL"`への誤復元は継続。v1.5.56 fixtureは28 tables / 198 columnsで、
new real columnsはない。接続先/object identifiersはartifactに保存していない。

2026-08-07 に configured real Spanner を再調査し、cache 無効の required real
drift gate が通過した。48 tables / 306 columns、未知/欠落 table/column 0、type mismatch
0。前回と同じく locality option は `storage = "ssd"` が2 rows、
`ssd_to_hdd_spill_timespan` は2 rowsとも SQL NULLで、column metadata 自体も
`IS_NULLABLE = YES` と宣言する。view は `INVOKER` 9 rows、change-stream option は0、
`GetDatabaseDdl` は328 statements / memefish parse 327で family countsも前回と同じ。
`LoadSchema` は同じ NULL decode error で停止した。

今回初めて ordinal まで比較し、`PARAMETERS.PARAMETER_DEFAULT` / `SPANNER_TYPE` と
`ROUTINES.ROUTINE_BODY` / `ROUTINE_DEFINITION` / `SECURITY_TYPE` / `SPANNER_TYPE` の
registry ordinal 6件が service と不一致と判明した。`PARAMETERS` 9 rows と `ROUTINES`
5 rows の direct name-tagged decode は成功したため、現在の loader failure ではない。
これらの registry entry は initial commit 由来で、8月5日の probe は ordinal を比較して
いなかったため、service の直近変更とは断定しない。接続先識別子は artifact に保存していない。

2026-08-05 に emulator image
`gcr.io/cloud-spanner-emulator/emulator:1.5.56`（linux/arm64、digest
`sha256:18a56fd557011e50e1733a9232e8d17ec9bdd7e51f6cf7660f14c234479f4f36`）と
configured real Spanner を同じ read-only harness で比較した。real drift gate は
Go test cache 無効で通過。emulator 28 tables / 198 columns と real 48 / 306 のうち、
共通 table の column set はすべて一致し、未知 registry entry はなかった。

v1.5.56 synthetic fixture の `GetDatabaseDdl` と `LoadSchema` 再生成はともに
4 statements / 同じ 4 families だった。差分は implicit primary-key ordering が
`ASC` へ明示化されるものと、locality group が `storage = "hdd"` から
`inflash = "BOOL"` へ変わるもの。後者は意味のある不一致なので、upstream defect か
compatibility mapping かを確認するまで pin は v1.5.55 のままにする。実 Spanner は
`GetDatabaseDdl` 328 statements（memefish v0.8.1 parse 327）だったが、
`ssd_to_hdd_spill_timespan` の 2 rows がともに SQL NULL で loader が停止したため、
real canonical-set comparison は未完了。接続先識別子は artifact に保存していない。

2026-07-26 の repository-local implementation 完了時点では、named-schema focused
regressions、`mise run test-all`（lint 0 issues / build / Docker-backed full
tests）、required real-Spanner drift、short race、vet、module verification、
formatting、diff hygiene がすべて通過。named-function/proto probe の一時
function/schema も cleanup 済み。

同日の named-schema review remediation 後は、focused regressions、emulator
drift、required real-Spanner drift、short race、vet、module verification、
formatting、diff hygiene が通過。`mise run test-all` は lint / build と Omni
以外の tests が通過したが、Omni image `2026.r1-beta.2` の `spanner_server` が
readiness 前に `SIGABRT` を繰り返し、full gate と Omni 単体の計 3 回で timeout
したため、現 snapshot の Omni gate は runtime 要因で未通過。

2026-07-28 に configured real Spanner との比較を cache 無効で再実行。
required real drift gate は通過し、read-only `DiscoverColumns` と
`AllTableMetas` の双方向比較も service / registry ともに 48 tables /
306 columns で完全一致した。unknown service table/column と
registry-only table/column はすべて 0。接続先識別子は artifact に保持していない。

ただし row-level の `LoadSchema` 比較では新しい実機差分を検出した。
`LOCALITY_GROUP_OPTIONS` の `ssd_to_hdd_spill_timespan` は観測した 2 行とも
`OPTION_VALUE = NULL` で、`LocalityGroupOption.OptionValue string` への decode が
`InvalidArgument` になり、`LoadSchema` → `ToDDLStatements` と
`GetDatabaseDdl` の canonical set 比較は未完了。`storage` の 2 行は non-NULL。
公式 INFORMATION_SCHEMA 文書は option value を説明するが nullability を明記して
いないため、NULL を「未設定 option row」として省略するか、DDL の `NULL` 値として
保持するかを実装前に確定する必要がある。`GetDatabaseDdl` は 328 statements、
memefish v0.8.1 で parse できたものは 327 statements。

2026-07-23 の memefish v0.8.1 更新後は、targeted AST round-trip tests、`mise run test-all`
（lint / build / full tests）、short race、vet、module verification、required real-Spanner
drift を実行してすべて通過。real probe の一時 role / sequence / schema は cleanup 済みで、
`PLACEMENT KEY` table は作成前の semantic/configuration validation で拒否されたため残存しない。

`cmd/roundtrip/main.go` の `sampleDDLs` は round-trip 確認用に増やしてよい。代表的な DDL
（INTERLEAVE, INDEX, VIEW, CHANGE STREAM, SEQUENCE, FK with CHECK, NULL_FILTERED INDEX,
SQL SECURITY INVOKER, allow_commit_timestamp）は既に含む。

## 見落としやすい落とし穴（再掲）

- **`token.Pos(0)` は valid**。「位置不明」は `token.InvalidPos = -1`。
  Optional な Pos フィールド（`Rparen`, `Hidden`, `Stored`, etc.）は `token.InvalidPos` で初期化する。
- **`ast.SchemaType` ≠ `ast.Type`**。`memefish.ParseType` は `ast.Type` を返す。
  カラム型は必ず `spannertype.ParseSchemaType` 経由。
- **memefish は未知の型名を NamedType として受理する**ので「invalid 型」を投げてもエラーにならない。
  parse 通過 = 妥当、ではない。
- **Spanner に `CREATE PROCEDURE` は存在しない**。`INFORMATION_SCHEMA.ROUTINES` は UDF を
  procedure として登録しているだけ。`CALL` は system procedure 専用。
- **SQL UDF は GA だが未ドキュメント**。MySQL UDF library（80 関数）で実運用されている。
  Remote UDF は Preview だがドキュメントあり。両方とも `ast.CreateFunction` で対応。

## 参照ファイル

- アーキテクチャ・コーディング規約: `AGENTS.md`
- 未対応機能リスト: `UNSUPPORTED_DDL.md`
- タスク: `TODO.md`
- スキーマ差分レポート: `docs/spanner-schema-diff-report-20260509.md`（過去版: 20260429 / 20260309）
- 差分元データ: `/tmp/spanner-schema-diff-20260509/{emulator-v1.5.55,real}.csv`
- memefish v0.8.1 ソース: `~/go/pkg/mod/github.com/cloudspannerecosystem/memefish@v0.8.1/ast/`
  （API 差分の比較用）
- 参考 struct パターン: `~/work/apstndb/spanneropttools/schema/types.go`
