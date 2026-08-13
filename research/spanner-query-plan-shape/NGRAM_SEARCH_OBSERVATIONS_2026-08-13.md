# N-gram Search Plan Observations (2026-08-13)

This note records read-only `AnalyzeQuery(QueryMode=PLAN)` observations. It is
not a performance claim and does not establish result semantics.

## Sources and environment

- Official fuzzy-search documentation:
  <https://docs.cloud.google.com/spanner/docs/full-text-search/fuzzy-search>
- Official pattern-acceleration documentation:
  <https://docs.cloud.google.com/spanner/docs/full-text-search/pattern-matching-function-acceleration>
- Runtime: Spanner Omni `2026.r1-beta`, optimizer versions 1 through 8 and
  the unhinted default, on an empty generated fixture.
- Managed Spanner recheck: 2026-08-13, optimizer versions 1 through 8, using
  temporary destination-redacted table and index names. The objects were
  deleted after capture and their absence was verified.

The fixture deliberately uses two token columns. Ranked fuzzy search uses
`TOKENIZE_SUBSTRING(AlbumTitle, ...)` because `SCORE_NGRAMS` requires a direct
source-column reference. Ordinary pattern acceleration separately requires
`TOKENIZE_NGRAMS(LOWER(AlbumTitle), ...)`. Each search index stores
`AlbumTitle`, avoiding a base-table back join during exact predicate checking.

## Observed boundary

`SEARCH_NGRAMS`, `SCORE_NGRAMS`, and forced access to the n-gram search index
returned a version-capability error at optimizer versions 1 through 4. The
same queries returned plans at versions 5 through 8. The error text says the
feature requires version 6 or above, so the text and observed v5 acceptance
must be retained separately rather than treating the message as the actual
boundary.

The managed recheck independently produced the same effective boundary:
versions 1 through 4 rejected the fuzzy functions, forced n-gram search-index
access, and the scored hybrid-search control with the documented version-6
error text; versions 5 through 8 returned plans. The four `_BASE_TABLE`
pattern controls remained available at every version. All successful n-gram
candidate plans were byte-stable from v5 through v8, and each base control was
byte-stable from v1 through v8. Thus the v5 acceptance is not an Omni-only
artifact in this observation.

At versions 5 through 8 and at the default:

- fuzzy retrieval used `Scan{scan_type=SearchIndexScan}` on
  `NgramAlbumsFuzzyIndex` with a `Search Predicate` child;
- `SCORE_NGRAMS ... ORDER BY ... LIMIT` added one local `Sort Limit` and one
  global `Limit`, while remaining index-only;
- literal `LIKE`, `STARTS_WITH`, `ENDS_WITH`, and `REGEXP_CONTAINS` over the
  forced pattern index used a `Search Predicate` for candidate retrieval and
  a `Filter Scan` `Residual Condition` for exact checking;
- the byte-equivalent `_BASE_TABLE` controls used a table scan and residual
  condition, with no search predicate;
- a parameterized `LIKE` and a literal shorter than `ngram_size_min` still
  read the forced search index, but as `Full scan=true` with only a residual
  condition. They produced no `Search Predicate`, directly demonstrating the
  documented eligibility boundary.

The executable evidence is the `ngram_search` selector, its focused manifest,
and the v1-v8 Omni integration test. The raw discovery captures remain
temporary under `/tmp`; the repository retains the exact DDL, queries,
structural expectations, and absence assertions required to reproduce them.
The managed capture was also checked with the current plan-vocabulary catalog:
72 successful plans across the combined n-gram/hybrid run produced zero
unknown operator, metadata, or child-link findings. The raw managed file is
not retained because its per-request identifiers are transport evidence, not
part of the plan contract.
