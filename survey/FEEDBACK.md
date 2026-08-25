# FEEDBACK — spanner-emulator-survey レビューと改善案

作成: 2026-07-03 / 対象コミット: **なし（後述: git 管理外）**

このドキュメントはリポジトリ全体を通読したレビューと改善案。改善案は「全面書き直しレベルの
根本的な提案」も含む。優先度順ではなく、影響度で並べたうえで各項に *Severity* を付す。

> 注: 執筆途中で中断しても失われないよう、確定した所見から順に追記していく方針で書いている。

---

## 0. 依頼された確認事項の結果

### 0-1. `CLAUDE.md` → `AGENTS.md` 参照構造 — OK ✅

- `CLAUDE.md` の中身は `@AGENTS.md` の 1 行のみ。プロジェクトコンテキストは `AGENTS.md` に集約
  され、`CLAUDE.md` はそれを取り込む形になっている。ユーザ global instruction の
  「新規探索プロジェクトでは `CLAUDE.md` は `@AGENTS.md` 参照にする」方針に一致。**問題なし。**

### 0-2. Open Issues / Open PRs の確認 — 対象リポジトリが GitHub 上に存在しない

- module パス `github.com/apstndb/spanner-emulator-survey` に対応する GitHub リポジトリは
  **存在しない**（`gh repo view apstndb/spanner-emulator-survey` → `Could not resolve to a
  Repository`）。`apstndb` の全リポジトリを検索しても該当なし。
- したがって「このリポジトリ自身の Open Issues / Open PRs」は**存在しない**（レビュー対象なし）。
- 関連する upstream の追跡 issue は memefish 側の 2 件で、`UNSUPPORTED_DDL.md` / `TODO.md` に
  記載済み:
  - memefish [#335](https://github.com/cloudspannerecosystem/memefish/issues/335)
    (`GRANT ... ON ALL ... IN SCHEMA`)
  - memefish [#193](https://github.com/cloudspannerecosystem/memefish/issues/193)
    (`PLACEMENT KEY`)
  - named-schema property graph（issue 番号未記載 → **要: upstream issue 番号の明記**）

---

## 1. Critical: このプロジェクトは git 管理下にない  *(Severity: High / Process)*

- ルートに `.git` が無い（`git rev-parse` は `not a git repository`）。`.gitignore` は存在するのに
  リポジトリが初期化されていない。
- 数千行・複数回のハンドオフ・「双方向変換の網羅実装」というマイルストーンに達した規模の成果物が
  **バージョン管理されていない**。誤削除・巻き戻し不能・変更履歴喪失のリスクが常時ある。
- 改善案:
  1. 直ちに `git init` してコミット。
  2. ビルド成果物と一時物を除外（下記 §7 参照）。
  3. GitHub に push するなら Issue/PR ベースの追跡が可能になり、依頼された「Open Issues/PRs 運用」も
     初めて意味を持つ。少なくとも upstream ブロック項目（memefish #335 / #193 / named-schema graph）を
     ローカルの `TODO.md` からミラーする形で issue 化すると追跡しやすい。

---

## 2. 二重の手書きソース・オブ・トゥルース  *(Severity: High / Architecture)*

同じ「INFORMATION_SCHEMA の各カラム一覧」を **2 か所で手書き**している:

| 場所 | 形式 | 規模 |
|---|---|---|
| `infoschem/meta.go` `informationSchemaTables` | `col(name, type, ordinal)` レジストリ | **307 entries** |
| `infoschem/tables.go` の struct | `spanner:"COL"` タグ付き Go struct | **300 tags** |

- 両者は既に **307 vs 300 でズレている**（privilege 系など struct 未定義のものがある一方、
  レジストリ側にしか無い/その逆が生じうる）。追加時に片方だけ更新すると：
  - meta にあって struct に無い → クエリはされるが `WithLenient` で黙って捨てられる。
  - struct にあって meta に無い → **SELECT 対象に入らず永遠に読まれない**（サイレント欠落）。
- しかも `ColumnMeta.SpannerType` と `OrdinalPosition` は `col()` で**セットされるだけで一度も読まれ
  ない**（`Query()` は `Name` しか使わない）。307 行 × 2 フィールド分が実質デッドデータ。

**改善案（根本）**: レジストリを廃し、SELECT カラム一覧を **struct のリフレクションから生成**する。
`spanner:"..."` タグが既にカラム名の唯一の正なので、`reflect` で列挙すれば `meta.go` の大半（19 KB）が
消え、ドリフトの一クラスが構造的に消滅する。`TableMeta` を残すにしても、`Columns` は struct 型から
導出する 1 本化にできる。`SpannerType`/`OrdinalPosition` を残すなら「型ドリフト検出」に**実際に使う**
（drift テストで実カラム型と突合）ことで初めて存在意義が出る。今は使っていないので削除が妥当。

---

## 3. 「ライブ DB → Schema」ローダーが公開 API に無い  *(Severity: High / Architecture)*

- `AGENTS.md` が掲げる中核価値は「同じ `infoschem` 構造体を**ライブ DB を問い合わせて**populate し、
  `astconv` で DDL に戻す」こと。しかしそれを行う関数は**どこにも公開されていない**。
- 実体は `cmd/roundtrip/main.go` の非公開 `queryInformationSchema()` にハードコードされ、しかも
  `astconv.Schema` の ~44 スライス中 **29 テーブルしか読んでいない**（roles / grants / placements /
  routines / parameters / table_synonyms / proto 関連などが欠落）。demo としては動くが、ライブラリの
  目玉機能が「demo の中の私的関数」に閉じ込められている。
- drift テストも「カラム網羅の突合」しかせず、**ライブ DB から Schema を組んで DDL に戻す E2E は無い**。

**改善案（根本）**: `func LoadSchema(ctx, client) (*astconv.Schema, error)`（配置は `infoschem` か
新規 `schemaload` パッケージ）を公開し、`cmd/roundtrip` と drift テストの両方がこれを使う。ローダーは
§2 のリフレクション化と組み合わせれば「対象テーブルを列挙 → 各 struct 型へ `SelectAll`」の総当たりに
畳めるため、テーブル追加時に main.go の `targets` を手で足す必要も消える。

---

## 4. round-trip テストが浅い  *(Severity: Medium-High / Testing)*

- `roundtrip_test.go` の多くは `FromDDLStatements` → `ToDDLStatements` を回した後、
  「`CreateTable` が 1 個できた」「カラムが 5 個」といった**構造カウント**しか検証しない
  （例: `TestRoundtrip_SimpleTable`）。一部は `.SQL()` 完全一致で見ている（search index など）が、
  方針が不統一で、**入力 DDL と出力 DDL の正規化文字列一致**という round-trip の本丸を検証していない
  ケースが多い。カラム順・オプション欠落・別名生成などの退行を取りこぼす。

**改善案**:
1. テーブル駆動で `normalize(in) == normalize(ToDDL(FromDDL(in)))` を全サンプルに課す共通ヘルパを 1 本
   用意（`normalize` は memefish で再パース → `SQL()` 再出力、で空白/整形差を吸収）。
2. §3 のローダーが入ったら、emulator を立てて「実 INFORMATION_SCHEMA → Schema → DDL」を通す E2E を
   drift テストに追加。`cmd/roundtrip` を回すだけの手動確認から脱却する。
3. 既存の弱いカウント assert は上記共通ヘルパへ寄せて置換。

---

## 5. `ToDDLStatements` の 18 連ボイラープレート  *(Severity: Medium / Maintainability)*

- `astconv/schema.go` の `ToDDLStatements` は「変換関数を呼ぶ → err チェック → append」を 18 回ほぼ
  逐語コピーしている（proto → schema → db → tables → ... → statistics）。順序という重要情報が
  ボイラープレートに埋もれ、順序変更・段追加のたびに 5 行の定型を書く。
- 改善案: `[]func() ([]ast.DDL, error)`（各 `s.toXxxDDL`）のスライスを Spanner 出力順に並べ、for で回す。
  順序が 1 か所のリストとして可読・テスト可能になる。docstring の番号付き手順とコードのズレ
  （現状 docstring は Functions / Locality Group / Vector Index を書き落としている）も解消できる。

---

## 6. leaf-ident 抽出の重複  *(Severity: Low-Medium / Maintainability)*

- `x.Idents[len(x.Idents)-1].Name` という「`*ast.Path` の末尾識別子取り出し」が全 `from_ast_*.go` に
  散在（`AGENTS.md` でもわざわざ注意書きしているほど頻出）。off-by-one や nil パス時の panic 温床。
- 改善案: `helpers.go` に `func leafName(p *ast.Path) string`（nil/空を安全に扱う）を追加し、`path()` の
  対（構築 ↔ 取り出し）として全面置換。named-schema 対応（§8）を入れる際も、ここを 1 点にしておくと
  「末尾だけ取る」箇所を「スキーマ修飾を保つ」へ一括で切り替えやすい。

---

## 7. リポジトリ衛生  *(Severity: Medium / Hygiene)*

- **52 MB のビルド成果物 `roundtrip` がルートに commit 対象として置かれている**（`.gitignore` は
  `.tmp/` と `mise.local.toml` のみ）。バイナリはリポジトリに含めない。`.gitignore` に追加し削除。
- 日付付き schema-diff レポートが 3 本＋α ルート直下に堆積
  （`spanner-schema-diff-report-2026{0309,0429,0509}.md`）。`docs/` などに寄せると root が見通せる。
- `.tmp/` に 44 エントリ。ローカル作業残骸なら定期削除、共有すべきものがあるなら整理。

---

## 8. named-schema の扱いが不完全（かつ文書化が曖昧）  *(Severity: Medium / Correctness)*

- `fromCreateTable` は `ct.Name.Idents[len-1].Name` で**末尾のみ**採用し、`infoschem.Table.TableSchema`
  を一切セットしない。つまり `CREATE TABLE myschema.T (...)` は from 方向でスキーマ修飾を失う。
  property graph の named-schema 非対応（既知・upstream ブロック）と合わせ、「named schema でどこまで
  round-trip するか」の全体像が `UNSUPPORTED_DDL.md` に**部分的にしか**書かれていない。
- 改善案: named-schema 対応状況を DDL family 横断で表にまとめ（table / index / view / graph / grant …
  各々 default schema のみか named も可か）、コード側は §6 の `leafName` を「修飾を保つ path 変換」に
  差し替えられる形にしておく。

---

## 9. ドキュメントのドリフト  *(Severity: Low-Medium / Docs)*

`HANDOFF.md` / `AGENTS.md` と実態にいくつかズレ:

- `HANDOFF.md`: 「Go 1.24」「memefish は HEAD（`v0.6.3-...`）に固定」→ 実際は `go.mod` で
  **Go 1.25.0 / memefish v0.7.0**。`AGENTS.md` は v0.7.0 に更新済みだが HANDOFF は古い。
- drift テストは emulator を **1.5.55** に pin（`drift_test.go:29`）。一方 `AGENTS.md` / `HANDOFF.md` /
  schema-diff データは **1.5.53** 基準。どの版で較正済みかが 3 箇所でバラバラ。
- `AGENTS.md` の "Schema diff data" が指す `/tmp/spanner-schema-diff-20260509/...csv` は `/tmp` 配下で
  **揮発**（再起動で消える）。「source-of-truth」を `/tmp` に置くのは不適切。リポジトリ内（gitignore
  しない場所）へ移すか、再生成手順のみを正とする旨に書き換える。
- `HANDOFF.md` と `AGENTS.md` の「完了済み作業」節はかなり重複。HANDOFF は「今どこで止まっているか＋
  次の一手」に絞り、恒久情報は AGENTS へ寄せると二重メンテを避けられる。
- `ToDDLStatements` の docstring 手順（§5）と実コードの段が不一致。

---

## 10. Lint が現状 red — `test-all` が最初のゲートで落ちる  *(Severity: Medium / CI)*

- `golangci-lint run ./...` が 1 件失敗:
  `astconv/proto_bundle_emulator_test.go:72` — `context-as-argument`（revive）。
  `func assertProtoBundleSchemata(t *testing.T, ctx context.Context, ...)` の引数順を
  `(ctx context.Context, t *testing.T, ...)` に直すだけ。
- これがあるため `mise run test-all`（`depends = [lint, build, test]`）は**通らない**。CI を回す前提の
  タスク定義なのに現状レッド。まず緑化を。
- 補足: `.golangci.yml` は `version: "2"` だが `linters-settings:` / `formatters-settings:` という
  **v1 形式のトップレベルキー**を使っている（v2 は `linters.settings` / `formatters.settings`）。
  今は revive の `exported: disabled` が効いている様子（exported 警告が氾濫していない）ので実害は
  出ていないが、golangci-lint の将来版で無視され設定が飛ぶ懸念。v2 スキーマへ移行推奨。

---

## 11. 良い点（維持したい設計）

- `to_ast_<family>.go` / `from_ast_<family>.go` の対称ファイル分割は DDL family 単位で見通しがよい。
- **Dynamic Column Discovery**（`DiscoverColumns` + `Query(discovered)`）は Emulator/Omni/Real の差を
  ハードコードせず吸収する堅実な設計。§2 でレジストリを畳んでも、この実行時ディスカバリの発想は残す価値大。
- `helpers.go` の小さなコンストラクタ群（`ident`/`path`/`strval`/`mkOptions` 等）は読みやすい。
- 未対応機能を upstream issue とともに `UNSUPPORTED_DDL.md` に明示し、コード側にも参照を置く方針は良い。
- proto bundle を `FileDescriptorSet` から、property graph を metadata JSON から復元するアプローチは
  「INFORMATION_SCHEMA からの復元」という本来困難な方向をきちんと攻めている。

---

## 推奨着手順（インパクト × コスト）

1. **§1 git init + §7 バイナリ除外 + §10 lint 緑化** — 低コスト・即効。土台を安全にする。
2. **§3 公開ローダー + §4 E2E round-trip** — 中核価値を API と検証で裏づける。
3. **§2 レジストリのリフレクション化 / デッドフィールド削除** — ドリフト源の構造的除去（要リグレッション）。
4. **§5 変換ディスパッチの table 化 + §6 `leafName` 導入** — 保守性の底上げ。
5. **§8 named-schema 対応表 + §9 ドキュメント整合** — 仕様境界を明確化。
