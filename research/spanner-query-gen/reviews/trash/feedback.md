以下は、アップロードされた README/DESIGN を主対象に、sqlc・yo・周辺ツールの公開ドキュメントも照合したレビューです。実装コード自体は読んでいないため、実装品質ではなく「設計方針・機能境界・ロードマップ」のレビューとして扱っています。 ￼  ￼

総評

spanner-query-gen の方向性はかなり良いです。特に、「ORM ではなく、SQL と schema を契約として残す」「Spanner の write 特性を update_mask / explicit column list として設計の中心に置く」「BigQuery EXTERNAL_QUERY による Spanner federation を first-class に扱う」という 3 点は、既存ツールとの差別化として成立しています。sqlc は query-first 型安全コード生成の成熟例ですが、公式サポート対象は MySQL / PostgreSQL / SQLite が中心で、Spanner / BigQuery は対象外です。 ￼ yo は Cloud Spanner 向け Go codegen として近いですが、DDL/Information Schema からの schema-first model 生成が中心で、query-driven generation や BigQuery federation までは主戦場ではありません。 ￼

一方で、現設計は「sqlc 的 query client」「yo 的 table/index model」「Spanner write helper」「BigQuery DTO」「federated query analyzer」を同時に取り込もうとしており、プロダクト境界が膨らみやすいです。いま最も重要なのは、機能を増やすことよりも、“このツールでしか価値が出ない核” を先に固定することです。私なら核を次のように定義します。

Spanner/BigQuery GoogleSQL の静的解析に基づき、Spanner semantics を壊さず、query result / write input / federated query DTO を生成する、GoogleSQL 専用の narrow code generator。

この核から外れる「一般 ORM 化」「過剰な template customization」「広範な CRUD 自動生成」は、少なくとも後回しがよいです。

良い設計判断

第一に、SQL を契約として扱う方針は正しいです。DESIGN は「SQL/schema が source of truth」「生成コードは小さく、予測可能で、review しやすい」と明記しており、これは sqlc の成功要因と同じ方向です。sqlc も、SQL を書いて型安全なインターフェースを生成する流れを中核にしています。 ￼  ￼

第二に、Spanner write を explicit column list / update_mask 中心にしているのは非常に良いです。Spanner では DML 実行時にアクセスした列に対して lock が関係し、lock statistics でも競合列が観測対象になります。 ￼ また mutation count や secondary index の影響も実運用上の制約であり、公式ドキュメントでも commit あたり mutation limit と secondary index の追加 mutation が説明されています。 ￼ したがって、DESIGN/README が「更新対象列は cosmetic optimization ではない」としているのは、Spanner 専用 generator として強い判断です。 ￼  ￼

第三に、planning と rendering の分離は必須であり、設計に入っているのは良いです。名前解決、field conflict、import、parameter、write helper の receiver field 名などを rendering 側で再推論しないという方針は、codegen の長期保守性に直結します。 ￼ ここはロードマップ上も Phase 1 の完了条件として扱うべきで、Query Methods より優先度を上げる価値があります。

第四に、BigQuery EXTERNAL_QUERY の connection ごとに Spanner schema scope を分ける設計は妥当です。BigQuery から Spanner に federated query を送るには EXTERNAL_QUERY を使い、接続は Spanner database role や権限にも関係します。 ￼ そのため、BigQuery outer query と Spanner inner query を同じ catalog で雑に解析するのではなく、connection ごとに Spanner analyzer を切る設計は正しいです。 ￼

最も大きいリスク

1. client: both と shared DTO が便利すぎて危険

client: both で NullValue[T] を使い、Spanner decoding / encoding と BigQuery loading を同一 struct で扱う方針は魅力的です。README/DESIGN でも nullable wrapper によって query result、DML parameter、mutation helper を共通 struct に寄せる意図が示されています。 ￼  ￼

ただし、ここは事故りやすいです。Spanner と BigQuery は同じ GoogleSQL 系でも、型境界、nullable semantics、STRUCT/ARRAY、JSON、PROTO/ENUM、timestamp/date handling、BigQuery load schema の期待値が完全には一致しません。DESIGN は「Spanner-only values that BigQuery cannot return を reject する」としており、この方針自体は良いのですが、shared DTO をデフォルト UX にすると、ユーザーは「共通化できるはず」と期待してしまいます。 ￼

改善案は、client: both を単なる boolean 的設定にしないことです。たとえば次のように分けた方が安全です。

runtime:
  spanner:
    decode: true
    dml_params: true
    mutations: true
  bigquery:
    load_schema: true
    query_methods: false
dto:
  sharing:
    default: compatible_only
    require_explicit_cross_dialect: true

特に cross-dialect DTO は、暗黙 merge ではなく shared_struct: Singer のような明示 opt-in に寄せるべきです。

2. nullability の保守性と UX がまだ弱い

README は「result fields are nullable by default」「required で non-null Go types を出せる」「strict は analyzer が証明できない場合に reject」としています。 ￼ DESIGN も nullability を conservative に扱う方針を示しています。 ￼ これは安全側ですが、すべてが nullable wrapper になると、生成コードの使い勝手が急速に悪化します。

ここは “安全だが不便” と “危険だが便利” の二択にしない方がよいです。おすすめは nullability confidence を plan に持つことです。

non_null_by_schema       -- table/index shorthand で NOT NULL column
non_null_by_expression   -- COUNT(*), literal, deterministic non-null expression
nullable_by_join         -- outer join, nullable column
unknown                  -- analyzer cannot prove
user_required_override   -- required override

そして生成時に、unknown を nullable にするだけでなく、--vet や explain-plan で「この field は unknown なので nullable にしました」と出すべきです。sqlc も sqlc.narg() で nullability を明示的に制御する escape hatch を持っています。 ￼

3. insert_or_update / replace / ON CONFLICT の意味をもっと分離すべき

README は insert_or_update が INSERT OR UPDATE、replace が INSERT OR REPLACE を emit し、ON CONFLICT は別モデル化すべきと書いています。 ￼ これは正しい方向です。Spanner GoogleSQL では INSERT OR UPDATE は primary key がなければ insert、あれば指定値で update し、update 時に指定しない column は unchanged になります。 ￼ 一方で ON CONFLICT DO UPDATE は conflict target を明示でき、unique constraint / primary key / unique index との関係を別に扱います。 ￼ PostgreSQL dialect 側の Spanner ON CONFLICT にはさらに制約があります。 ￼

設計上は、operation: insert_or_update に update_mask を足すだけでは不十分です。insert branch では NOT NULL column / DEFAULT / generated column / commit timestamp の扱いが問題になり、update branch では “触る column” が問題になります。したがって、以下のように分けるのが安全です。

writes:
- name: UpsertSingerName
  operation: upsert
  insert:
    columns: [SingerId, FirstName, LastName]
  update:
    mask: [FirstName, LastName]
  conflict:
    mode: insert_or_update   # or on_conflict
    target: primary_key      # or unique_index: ...

この形なら、将来 ON CONFLICT を入れても operation の意味が破綻しません。

4. DML と Mutation の同時生成は便利だが、transaction policy が必要

README は mutation helper と DML helper の両方を default で生成できるとしています。 ￼ Spanner 公式ドキュメントは、DML と mutations はどちらも data modification API だが、Read Your Writes や constraint checking のタイミングなどが異なり、同一 transaction で混在させないことを best practice としています。 ￼

そのため、単に両方を生成するだけではなく、生成コードの API 形状で混在を抑止した方がよいです。たとえば、DML helper は spanner.Statement を返すだけ、mutation helper は *spanner.Mutation を返すだけ、という現在方針はよいですが、さらに package comment / generated doc / optional lint rule で「同一 transaction 内で混ぜるな」を出すべきです。rules に no-mixed-dml-mutation を置けるようにするのも良いです。

5. table/index shorthand は「SQL is contract」と衝突しうる

table が SELECT に展開され、index が index keys + STORING columns に展開される仕様は便利です。README でも query は sql / table / index の exactly one とされています。 ￼ ただし、DESIGN の「SQL は見えて review 可能であるべき」という原則とは緊張関係があります。 ￼

対策として、table/index shorthand から展開された SQL は必ず generated constant として出すべきです。さらに spanner-query-gen explain-plan で以下を見られるようにすると、shorthand の透明性が保てます。

query: FindAlbumsByTitle
source: app_spanner
expanded_sql:
  SELECT AlbumTitle, SingerId, AlbumId, ...
  FROM Albums@{FORCE_INDEX=AlbumsByTitle}
  WHERE AlbumTitle = @AlbumTitle
result_fields:
  AlbumTitle: STRING NOT NULL by index key
  ...

sqlc / yo から取り込むべきもの、取り込まないべきもの

sqlc から取り込むべきもの

sqlc からは、query annotation、parameter struct、result mode、type override、rename、CI/vet/verify の考え方を取り込む価値があります。sqlc は -- name: <name> <command> 形式で :one / :many / :exec などを表現します。 ￼ v2 config では schema/query/gen/rules/plugin/overrides などが整理され、rules や plugin も config の first-class 要素になっています。 ￼ また sqlc vet は CEL rule で query lint を行い、sqlc verify は schema change が既存 query を壊すか検証する workflow を持っています。 ￼

取り込むなら、まずは -- name: GetSinger :one だけで十分です。sqlc.arg は GoogleSQL の named parameter @SingerId と重なるので優先度は低いです。sqlc.embed は JOIN 結果の struct composition として有用ですが、BigQuery/Spanner 両対応では型境界が複雑になるので後回しがよいです。sqlc.slice は runtime SQL expansion を伴い、DESIGN の “SQL is contract” と衝突しやすいため慎重に扱うべきです。 ￼

yo から取り込むべきもの

yo からは、Spanner table/index model、mutation helper、ignore tables/fields、build tags、custom type、template/module の経験を参考にできます。yo v2 は DDL file からの生成を recommended とし、Information Schema 生成は column ordering issue のため deprecated としています。 ￼ これは spanner-query-gen の「live database に依存しない」方針を後押しします。yo は table ごとに struct/metadata/methods を生成し、Spanner mutation methods や index read functions を出す設計です。 ￼

ただし、yo 的な broad model generation を早く入れすぎると、spanner-query-gen が “query generator” なのか “Spanner ORM generator” なのか曖昧になります。Model Generation は Phase 4 に置かれていますが、この順序は妥当です。 ￼ さらに言えば、Phase 4 でも「全 table model 生成」ではなく、models.include で明示された table だけに限定した方がよいです。

周辺ツールからの示唆

SQLBoiler は database-first ORM の代表例ですが、現在の README では maintenance mode と説明され、代替として Bob や sqlc が挙げられています。 ￼ Bob は database-first ORM / query generation / factory generation / relationships まで含む広い SQL toolkit で、hand-written SQL query からの code generation も特徴に含みます。 ￼ ent は schema-as-code / graph modeling / code-first の entity framework で、spanner-query-gen の思想とは逆側です。 ￼ xo は schema または custom query から Go types/functions を生成しますが、自身を ORM ではないと位置付けています。 ￼

この比較からの結論は明確です。spanner-query-gen は Bob/SQLBoiler/ent のような「アプリケーションの data access layer 全体を支配するツール」を目指さない方がよいです。sqlc/xo に近い “SQL-visible generator” に留まりつつ、yo から Spanner-specific write ergonomics を借りる、という境界が最も強いです。

ロードマップへの提案

現在の Roadmap は Phase 1 stabilization、Phase 2 query methods、Phase 3 SQL files/annotations、Phase 4 model generation、Phase 5 overrides、Phase 6 validation workflows です。 ￼ 私なら順序を少し変えます。

Phase 1 は planning layer 完成を最優先にするべきです。 ここで plan を JSON/YAML として dump できるようにしてください。explain-plan は Phase 6 ではなく Phase 1.5 に前倒しした方がよいです。理由は、table/index shorthand、nullability、DTO merge、write column selection のすべてが plan の透明性に依存するからです。

Phase 2 は Spanner query methods だけに限定するのがよいです。 BigQuery method generation は急がず、まず DTO + BigQuery schema metadata の生成に留めるべきです。DESIGN も BigQuery method は TableSchema-to-Go loading が十分 robust になってからとしています。 ￼

Phase 3 の SQL files/annotations は、sqlc compatibility ではなく “sqlc-like minimal syntax” と明記すべきです。 GoogleSQL は named parameters があるため、annotation は -- name: X :one で十分です。macro compatibility を掲げると、sqlc.arg / sqlc.narg / sqlc.embed / sqlc.slice の互換性期待が生まれます。公式 sqlc macros は driver-specific placeholder や runtime slice expansion を含むため、そのまま Spanner/BigQuery に輸入するのは危険です。 ￼

Phase 5 の overrides は Phase 2 より前に一部だけ入れてもよいです。 特に Spanner NUMERIC、JSON、BYTES、TIMESTAMP、proto/enum、BigQuery BIGNUMERIC などは、現実のプロジェクトで default mapping だけだと採用障壁になります。sqlc でも type overrides と field renames は Go codegen の重要機能です。 ￼ ただし template customization は plan model が固まるまで待つべきです。

Phase 6 の validation workflows はもっと早く始めるべきです。 最初から --check だけでなく、vet の骨格を入れてください。sqlc の vet は CEL rules で query lint を行うため、spanner-query-gen でも no-select-star, no-mixed-dml-mutation, require-update-mask, no-cross-dialect-unknown-type, require-required-policy-strict-in-prod のようなルールが自然です。 ￼

具体的な設計改善案

config を「生成対象」と「実行 runtime」で分ける

現状の client: both は短くて便利ですが、意味が広すぎます。以下のように分けると将来の破綻が少ないです。

version: "1"
package: db
out: db/query_gen.go
gen:
  go:
    json_tags: true
    db_tags: false
    result_pointers: false
    nullable: generated_wrapper
runtime:
  spanner:
    queries: true
    dml: true
    mutations: true
  bigquery:
    dto: true
    table_schema: true
    methods: false

writes は insert/update/delete/upsert の column semantics を分離する

writes:
- name: UpsertSingerName
  source: app_spanner
  table: Singers
  operation: upsert
  input_struct: SingerNameWrite
  keys:
  - SingerId
  insert:
    columns:
    - SingerId
    - FirstName
    - LastName
  update:
    mask:
    - FirstName
    - LastName
  conflict:
    strategy: insert_or_update

この形なら mutation helper と DML helper の両方で意味が揃いますし、ON CONFLICT に移行する場合も conflict.strategy: on_conflict と conflict.target を足せます。

DTO merge は「暗黙」ではなく「説明可能」にする

result_struct の merge は便利ですが、merge conflict は codegen で最も理解しづらい失敗になります。生成 plan に、どの query/write がどの field に寄与したかを持つべきです。

struct SingerRow
  SingerId: INT64 required
    from:
      - query ListSingers column SingerId non_null_by_schema
      - query FederatedSingerIDs column SingerId unknown_cross_dialect
  FirstName: STRING nullable
    from:
      - query ListSingers column FirstName nullable_by_schema
      - write UpdateSingerName input FirstName required_by_update_mask

これにより、「missing fields become nullable」という設計判断が review 可能になります。DESIGN の merge rule は妥当ですが、ユーザーに見える説明が必要です。 ￼

generated code の public API は最初から控えめにする

最初の query method は Spanner のみに限定し、receiver は *spanner.Client 固定よりも transaction-aware にした方がよいです。たとえば、read-only transaction / read-write transaction / client のいずれでも使える抽象を検討してください。

type SpannerQueryRunner interface {
    Single() *spanner.Row
    Query(context.Context, spanner.Statement) *spanner.RowIterator
}

ただし、この interface が無理に BigQuery まで抽象化し始めたら悪手です。BigQuery は別 runtime として扱うべきです。

採用優先度

今すぐ優先すべきなのは、以下です。

1. plan model の完成と explain-plan
2. nullability confidence / DTO merge provenance
3. write semantics の分離: insert columns / update mask / conflict strategy
4. Spanner query methods only
5. minimal SQL file annotations
6. type overrides の最小セット
7. vet rules の骨格

後回しでよいのは、BigQuery query method、template customization、広範な model generation、sqlc macro compatibility、plugin system です。

最終判断

このツールは、単なる「sqlc for Spanner」よりも良い位置を取れます。強みは、Spanner の write semantics と BigQuery federation を同じ静的解析・DTO 生成の中で扱える点です。逆に危険なのは、sqlc・yo・ORM 系ツールの良いところをすべて取り込もうとして、境界がぼやけることです。

設計方針としては、次の一文を明文化するとよいと思います。

spanner-query-gen は GoogleSQL/Spanner/BigQuery の静的に宣言された query と write を、実行時 magic なしで Go 型へ落とす generator であり、ORM・query builder・汎用 CRUD generator ではない。

この制約を守る限り、現在の方向性はかなり有望です。
