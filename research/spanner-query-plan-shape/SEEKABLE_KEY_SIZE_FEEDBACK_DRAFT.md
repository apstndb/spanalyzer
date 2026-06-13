# Feedback draft: `seekable_key_size` reports 0 for all-equality point seeks

Status: **draft, not delivered.** Scoped to Spanner Omni `2026.r1-beta`;
confirm on Cloud Spanner (DBaaS) before sending. Intended destination is a
Google channel (docs "Send feedback" / public issue tracker, Cloud Spanner
component), not the spanner-hacks community repo.

## Summary

The plan metadata field `seekable_key_size` reports `0` for an all-equality
seek on a full key prefix (a point lookup), which is the same value reported
for a plain full scan, even though the scan carries a complete `Seek
Condition`. This contradicts the value predicted by the documented
definitions, the field appears to be undocumented (only the same-named hint
is documented), and the hint cannot force the documented value. The result
is that `seekable_key_size` is ambiguous exactly where it would be most
useful — judging whether a query seeks efficiently.

## What the documentation says

1. `SEEKABLE_KEY_SIZE` table hint
   (reference/standard-sql/query-syntax.md, Hints section):
   > Forces the seekable key size to be equal to the specified value. The
   > seekable key size is the length of the key (primary key or index key)
   > that's used in a seekable condition, while the rest of the key is used in
   > a residual condition.
   (Range `0` to `16`; requires `FORCE_INDEX`.)

2. Definition of "seekable condition"
   (query-operators-leaf.md, Filter Scan / scan properties):
   > The seekable condition applies if Spanner can determine a specific row to
   > access in the table. In general, this happens when the filter is on a
   > prefix of the primary key. For example, if the primary key consists of
   > `Col1` and `Col2`, then a `WHERE` clause that includes explicit values
   > for `Col1`, or `Col1` and `Col2` is seekable.

Read together: for `WHERE Col1 = x AND Col2 = y` on a two-column key, the
seekable condition covers both columns, so "the length of the key used in a
seekable condition" is **2**. The expected `seekable_key_size` is therefore 2.

## What is observed (Omni 2026.r1-beta)

Index `OrderIndexDesc(shard_id, timestamp_order DESC)`, queries forced onto it
with `FORCE_INDEX=OrderIndexDesc`:

| `WHERE` | Seek Condition | Residual | Reported `seekable_key_size` |
| --- | --- | --- | --- |
| `shard_id = 0` | `shard_id = 0` | none | `0` |
| `shard_id = 0 AND timestamp_order = <ts>` | both keys | none | `0` |
| `shard_id = 0 AND timestamp_order > <ts>` | both keys | none | `2` |
| `shard_id BETWEEN 0 AND 9` | `shard_id` range | none | `1` |
| (no usable key predicate) | none (Full Scan) | — | `0` |

Hint round-trip on the all-equality 2-key point
(`shard_id = 0 AND timestamp_order = <ts>`, default reported value `0`):

| Hint | Result |
| --- | --- |
| `SEEKABLE_KEY_SIZE=1` | accepted; reports `1`; Seek Condition still covers both keys |
| `SEEKABLE_KEY_SIZE=2` | **rejected: `InvalidArgument` (empty message)** |

The same behaviour holds for primary-key tables (verified on a 3-column
`PRIMARY KEY` table) and is independent of interleaving.

## The discrepancies

1. **Predicted vs reported.** The documented definitions predict
   `seekable_key_size = 2` for the all-equality 2-key point, but the reported
   value is `0`.
2. **The documented value is unforceable.** `SEEKABLE_KEY_SIZE=2` on that same
   query is rejected with `InvalidArgument`, while `1` is accepted — so the
   value the documentation describes (`2`) cannot even be requested.
3. **The reported field is undocumented.** The docs define the
   `SEEKABLE_KEY_SIZE` *hint*; the `seekable_key_size` plan-metadata *field*
   and its values are not described, so there is no documented basis for
   interpreting a reported `0`.
4. **`0` is overloaded.** The same value `0` is reported for a complete point
   lookup (the most efficient access) and for a plain full scan (no seek at
   all). A reader using `seekable_key_size` to judge seek efficiency cannot
   distinguish the best case from a bad one without separately inspecting the
   `Seek Condition` child link and the `Full scan` flag.

A self-consistent reading of the observed values is that the reported field
counts the **range-extraction** prefix length (the leading key columns whose
bounds form a non-degenerate range or enumeration), and that an all-equality
prefix is resolved as a single-point lookup with no range to extract, hence
`0`. If that is the intended semantics, it differs from the documented
"length of the key used in a seekable condition" and is worth stating
explicitly.

## Reproduction

```sql
CREATE TABLE Foo (
  random_id STRING(22) NOT NULL,
  shard_id INT64 NOT NULL,
  timestamp_order TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
) PRIMARY KEY(random_id);
CREATE INDEX OrderIndexDesc ON Foo(shard_id, timestamp_order DESC);
```

```sql
-- reports seekable_key_size = 0 (complete 2-key Seek Condition, no residual)
SELECT random_id FROM Foo@{FORCE_INDEX=OrderIndexDesc}
WHERE shard_id = 0 AND timestamp_order = TIMESTAMP "2018-09-05T09:00:00Z";

-- rejected: InvalidArgument
SELECT random_id FROM Foo@{FORCE_INDEX=OrderIndexDesc, SEEKABLE_KEY_SIZE=2}
WHERE shard_id = 0 AND timestamp_order = TIMESTAMP "2018-09-05T09:00:00Z";

-- accepted: reports seekable_key_size = 2
SELECT random_id FROM Foo@{FORCE_INDEX=OrderIndexDesc}
WHERE shard_id = 0 AND timestamp_order > TIMESTAMP "2018-09-05T09:00:00Z";
```

Use `AnalyzeQuery` (PLAN mode) and read the Filter Scan's `seekable_key_size`
metadata and the scan's `Seek Condition` / `Residual Condition`.

## Suggested resolution (no behaviour change assumed)

1. Document the `seekable_key_size` plan-metadata field and the exact quantity
   it reports, separately from the `SEEKABLE_KEY_SIZE` hint.
2. Clarify the point-seek case: state that an all-equality key-prefix seek
   reports `0` (and why), so `0` is not read as "no seek".
3. Either reconcile the hint definition with the field (if they are meant to
   be the same quantity) or note that they differ, and document the valid
   forceable range for an all-equality point (observed: `0`/`1` accepted, the
   key length rejected).
4. Optionally, expose a signal that distinguishes a point lookup from a full
   scan without requiring callers to parse the Seek Condition text.

## Caveats before delivery

- All observations are on Spanner Omni `2026.r1-beta`. Confirm the same values
  and the `InvalidArgument` on `SEEKABLE_KEY_SIZE=2` on Cloud Spanner (DBaaS)
  before sending; scope the report to what is confirmed.
- This is framed as a documentation / diagnostic-clarity gap and a request for
  clarification, not a claim that the runtime behaviour is a bug. Reporting
  `0` for a point seek may be intentional internally; the issue is that it is
  undocumented and counterintuitive given the field name and the documented
  definitions.
