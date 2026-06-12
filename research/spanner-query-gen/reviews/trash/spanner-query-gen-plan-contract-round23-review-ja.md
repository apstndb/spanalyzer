# `spanner-query-gen` plan contract round 23 review

## 前提

今回も実装コード・実行ログ・CI output は共有されていないため、レビュー対象は public contract / schema / docs / 補助ドキュメントの整合性です。

確認した主な添付物は次です。

- `spanner-query-gen-plan-contract-round22-review-response-ja.md`
- `PLAN_CONTRACTS(17).md`
- `PLAN_CONTRACT_CANDIDATES(13).md`
- `README(38).md`, `README(39).md`
- 直近までに共有されていた `spanner-query-gen.plan-report.v1alpha.schema(20).json`

注意点として、round 22 response は plan-report schema の変更を述べていますが、今回の添付セットには更新後の plan-report schema が含まれていません。したがって schema に関する指摘は、**最新作業ツリーでは直っている可能性があるが、外部レビューではまだ検証できない点**として扱います。

## 総評

今回の方向性はかなり良いです。大きな設計変更ではなく、round 21 までに固めた plan-report / plan-contract artifact surface の最終同期として妥当です。

特に良い点は次です。

- raw CEL input を `raw_plan` / `raw_nodes` に統一し、serialized report の `queries[].plan` が raw `spannerpb.QueryPlan` ではなく rendered human-readable plan string であることを候補ドキュメント側にも反映した点。
- raw QueryPlan CEL は `replayable_from_report: false` であり、CI 向けには normalized operator view を推奨する線引きを維持した点。
- top-level `warnings` を always-present にする方針。
- `SpoolScan` の raw representation を fixture 化し、`spool_scan` family に正規化する方針。
- backend version / image digest capture を次の優先候補として認識しつつ、今回は artifact surface の tightening を優先した点。

この段階では、新しい predefined contract を増やすより、docs / schema / fixture / generated artifact の同期を詰める方が価値があります。

## P0: schema 更新が今回の添付から検証できない

round 22 response では、次の変更が反映済みと説明されています。

- root required に `warnings` を追加。
- `contract_stability` の tier-dependent coupling を schema に追加。
- `normalized_operators[].spool_name` を追加。

ただし、今回の添付には更新後の plan-report schema が含まれていないように見えます。直近までに共有されていた `spanner-query-gen.plan-report.v1alpha.schema(20).json` を見る限り、少なくとも以下はまだ確認できませんでした。

```text
root required に warnings がない
contract_stability に tier-dependent conditional が見えない
normalized_operators[].spool_name が operator schema にない
```

ローカル作業ツリーでは直っているなら問題ありませんが、外部レビュー可能性のため、次回は更新後の schema も添付するか、response に schema digest / generated schema path を明示するとよいです。

確認用には次のようなチェックを fixture 化できます。

```sh
jq '.required | index("warnings")' schemas/spanner-query-gen.plan-report.v1alpha.schema.json
jq '.$defs.operator.properties.spool_name' schemas/spanner-query-gen.plan-report.v1alpha.schema.json
jq '.$defs.contract_stability.allOf' schemas/spanner-query-gen.plan-report.v1alpha.schema.json
```

また、schema が generator output なら、`plan-report-schema --out ...` と保存済み schema の一致テストを round 22 の検証対象に含めるのが良いです。

## P0: no-target example と `warnings` always-present がまだ少し矛盾する

`PLAN_CONTRACTS.md` では、top-level `warnings` は always-present で、警告なしなら `warnings: []` と説明されています。この方針は良いです。

一方で、`Minimal no-target report` の例には `warnings: []` がありません。

```yaml
status: no_targets
target_summary:
  included_count: 0
  planned: 0
  errors: 0
  skipped: 0
  excluded: []
queries: []
contract_evaluation_mode: none
```

この例は「full artifact」ではなく「abridged snippet」だと言えば問題ありません。ただ、見出しが “minimal no-target report” なので、reader は schema-valid な最小 artifact と受け取りやすいです。

おすすめは、例に `warnings: []` を入れることです。

```yaml
status: no_targets
warnings: []
target_summary:
  included_count: 0
  planned: 0
  errors: 0
  skipped: 0
  excluded: []
queries: []
contract_evaluation_mode: none
```

もし他の required top-level fields も省略しているなら、見出しを “abridged no-target excerpt” に変えた方が安全です。

## P1: `spool_name` は metadata-derived stability reason の対象に入れる

`SpoolBuild` / `SpoolScan` の対応を見るために `operators[].spool_name` を公開する方針は良いです。これは raw QueryPlan metadata を normalized operator view にコピーした field なので、`subquery_cluster_node` や `call_type` と同じく **metadata-derived normalized field** として扱うべきです。

つまり、CEL が `spool_name` を読む場合、report の stability reason には例えば次のように出るのが自然です。

```yaml
stability:
  tier: normalized_operator
  reasons:
  - contract uses the normalized plan-report view
  - "contract reads metadata-derived normalized fields: spool_name"
  check_recommended: true
  replayable_from_report: true
```

`PLAN_CONTRACTS.md` の optional string metadata list に `spool_name` が追加されているのは良いですが、stability detection の field list / tests にも入っていることを確認したいです。

## P1: `contract_stability` の coupling は schema と runtime test の両方で固定する

round 22 response の方針どおり、次の coupling は schema でも runtime test でも固定する価値があります。

```text
tier: raw_query_plan
  check_recommended: false
  replayable_from_report: false

tier: normalized_operator
  replayable_from_report: true
  check_recommended: runtime policy; v1alpha では true だが schema 固定しない
```

`normalized_operator.check_recommended` を schema で固定しない判断は妥当です。metadata-derived normalized fields の扱いを将来細分化できるためです。

ただし `raw_query_plan` の `check_recommended: false` / `replayable_from_report: false` は、public artifact の安全性に直結するため schema で固定した方がよいです。

## P1: `warnings` が diagnostic array なら docs で粒度を説明する

Top-level `warnings` を always-present にするのは良いです。`contract_summary.environment_warnings` と役割が近いため、次の区別を `PLAN_CONTRACTS.md` に一文だけ足すとさらに読みやすくなります。

```text
warnings: artifact-wide production / completeness warnings
contract_summary.environment_warnings: warnings that affect contract interpretation under the selected backend / optimizer / contract mode
```

たとえば backend identity が `not_recorded` の場合、それが artifact-wide warning なのか contract environment warning なのかが reader にとって少し曖昧になります。両方に出す必要はありませんが、どちらに出すかを固定しておくと snapshot consumer に優しいです。

## P2: README が複数スコープに見えるので、添付・外部レビューでは path を明示するとよい

今回の添付には `README(38).md` と `README(39).md` があり、片方は `spanner-query-plan-shape` tool README、もう片方は root / library 風の README に見えます。実リポジトリ上では path が違うはずなので問題ではないと思いますが、外部レビューではどちらがどの path なのかが分かりづらいです。

次回以降、添付名や response の “Updated files” に以下のように path を書くとレビューしやすくなります。

```text
README.md
tools/spanner-query-plan-shape/README.md
research/spanner-query-plan-shape/PLAN_CONTRACTS.md
```

これは設計上の問題ではなく、レビュー時の traceability の問題です。

## P2: `SpoolScan` fixture は `spanner-hacks` feedback と implementation tests の橋渡しにする

`display_name: SpoolScan` + `metadata.spool_name` を `spool_scan` family にする fixture は良いです。さらに強くするなら、fixture 名に raw representation の意図を含めるとよいです。

```text
TestPlanReportSpoolScanDisplayNameWithSpoolNameMetadata
```

また、`SpoolBuild` / `SpoolScan` の対応を使う将来 contract は、単なる family count ではなく `spool_name` で対応関係を見る可能性が高いので、候補ドキュメントに “future spool consistency check” として一段落だけメモしておくと発展しやすいです。

## 今回の結論

大きな方向転換は不要です。round 22 の内容は、v1alpha plan contract surface の最終 polish として妥当です。

次にやるべきことは、新機能追加ではなく次の同期確認です。

1. 更新後 plan-report schema を外部レビュー可能にする。
2. root `warnings` required、`contract_stability` coupling、`operators[].spool_name` を schema で確認する。
3. no-target example に `warnings: []` を入れるか、abridged snippet と明記する。
4. `spool_name` を metadata-derived stability detection の対象として fixture 化する。
5. top-level `warnings` と `contract_summary.environment_warnings` の役割分担を一文で固定する。

これで plan-report artifact はかなり machine-checkable かつ reviewer-friendly な状態に近づいていると思います。
