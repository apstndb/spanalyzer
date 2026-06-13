# Feedback draft: `seekable_key_size` reports 0 for all-equality point seeks

Status: **draft, not delivered.** Self-contained reproduction below; verified
on both Cloud Spanner (DBaaS, 2026-06-13) and Spanner Omni `2026.r1-beta`.
Intended destination is a Google channel (docs "Send feedback" / public issue
tracker, Cloud Spanner component), not the spanner-hacks community repo.

## Summary

In query plans, the Filter Scan metadata field `seekable_key_size` reports
`0` for an all-equality seek on a full key prefix (a point lookup) — the same
value reported for a plain full scan — even though the scan carries a complete
`Seek Condition` over the whole prefix. The value `0` is therefore ambiguous
exactly where the field would be most useful (judging whether a query seeks
efficiently), the default `0` for a point seek is undocumented, and the
plan-metadata field itself is undocumented (only the same-named hint is).

## What the documentation says

1. Definition of "seekable condition"
   (query-operators-leaf.md, Filter Scan / scan properties):
   > The seekable condition applies if Spanner can determine a specific row to
   > access in the table. In general, this happens when the filter is on a
   > prefix of the primary key. For example, if the primary key consists of
   > `Col1` and `Col2`, then a `WHERE` clause that includes explicit values
   > for `Col1`, or `Col1` and `Col2` is seekable.

   So `WHERE Col1 = x AND Col2 = y` is, by definition, seekable on both keys.

2. `SEEKABLE_KEY_SIZE` table hint (reference/standard-sql/query-syntax.md):
   > Forces the seekable key size to be equal to the specified value. The
   > seekable key size is the length of the key (primary key or index key)
   > that's used in a seekable condition, while the rest of the key is used in
   > a residual condition.
   (Range `0` to `16`; requires `FORCE_INDEX`.)

Read together, for `WHERE Col1 = x AND Col2 = y` the seekable condition covers
both columns, so "the length of the key used in a seekable condition" is 2.

## Self-contained reproduction

```sql
CREATE TABLE SeekableKeySizeDemo (
  k1 INT64 NOT NULL,
  k2 INT64 NOT NULL,
  v  INT64,
) PRIMARY KEY (k1, k2);
```

Use `EXPLAIN` (or `AnalyzeQuery` in PLAN mode) and read the Filter Scan's
`seekable_key_size` and the scan's `Seek Condition` / `Residual Condition`.
`FORCE_INDEX=_BASE_TABLE` satisfies the `SEEKABLE_KEY_SIZE` hint's requirement
without needing a secondary index.

| Query | `Seek Condition` | Reported `seekable_key_size` |
| --- | --- | --- |
| `WHERE k1 = 1 AND k2 = 2` | `(k1 = 1) AND (k2 = 2)` | **`0`** |
| `WHERE k1 = 1 AND k2 > 2` | `(k1 = 1) AND (k2 > 2)` | `2` |
| `WHERE k1 BETWEEN 1 AND 5` | `k1` range | `1` |
| `WHERE v = 9` (no key predicate) | none (Full scan, residual `v = 9`) | `0` |

The first and last rows both report `0`, but the first is an optimal point
lookup (complete Seek Condition, no `Full scan`) and the last is a plain full
scan (`Full scan` flag, residual only, no Seek Condition).

Forcing the split on the point-lookup query (same table, `FORCE_INDEX=_BASE_TABLE`):

```sql
-- default: seekable_key_size = 0
SELECT v FROM SeekableKeySizeDemo@{FORCE_INDEX=_BASE_TABLE}
WHERE k1 = 1 AND k2 = 2;

-- @{... SEEKABLE_KEY_SIZE=1}: reports 1, Seek Condition still (k1=1 AND k2=2)
-- @{... SEEKABLE_KEY_SIZE=2}: reports 2, Seek Condition still (k1=1 AND k2=2)
-- @{... SEEKABLE_KEY_SIZE=3}: InvalidArgument (exceeds the 2-column key length)
```

Forcing `1` or `2` does not change the `Seek Condition` (it stays a complete
two-key point lookup) — only the reported number changes. So for a point seek
the reported value is a default split choice, not a description of how much of
the key is actually used to locate rows.

## The issues

1. **The default `0` for a point seek is undocumented and counterintuitive.**
   The documented definition says `WHERE k1 = x AND k2 = y` is seekable on both
   keys, but the reported default is `0`, not `2`. (`2` is achievable — it can
   be forced with the hint — so `0` is a default choice, not a structural
   limit.)
2. **`0` is overloaded.** The same value is reported for a complete point
   lookup (the most efficient access) and for a plain full scan. A reader using
   `seekable_key_size` to judge seek efficiency cannot tell them apart from the
   number alone; they must also inspect the `Seek Condition` child link and the
   `Full scan` flag.
3. **The plan-metadata field is undocumented.** The docs describe the
   `SEEKABLE_KEY_SIZE` *hint*; the reported `seekable_key_size` *field* and the
   meaning of its values (especially `0` for a point lookup) are not described.

A self-consistent reading of the values is that the field reports the
**range-extraction** prefix length the optimizer chose — the leading key
columns whose bounds form a non-degenerate range or enumeration — and that an
all-equality prefix is resolved as a single-point lookup with no range to
extract, hence `0` by default. If that is the intended semantics, it differs
from the documented "length of the key used in a seekable condition" and is
worth stating explicitly.

## Suggested resolution (no behaviour change assumed)

1. Document the `seekable_key_size` plan-metadata field and the exact quantity
   it reports, separately from the `SEEKABLE_KEY_SIZE` hint.
2. State that an all-equality key-prefix (point) seek reports `0` by default
   and why, so `0` is not misread as "no seek".
3. Reconcile the hint's "length of key in the seekable condition" wording with
   the reported field, or note that the field reports the chosen
   range-extraction split rather than the seekable-condition coverage.
4. Optionally, expose a signal that distinguishes a point lookup from a full
   scan without requiring callers to parse the `Seek Condition` text.

## Separate, less-understood observation (optional, verify before including)

On a **secondary index** seek (e.g. an index on `(SingerId, Duration)`),
forcing `SEEKABLE_KEY_SIZE=2` on an all-equality 2-key point is rejected with
`InvalidArgument`, whereas the same forced value is accepted on a base-table
seek of an equal-length key. The reason is unclear (secondary-index storage
keys append the base-table primary key, so the effective key length is larger
than the declared index columns). This is a separate question from the main
point above and should be confirmed and understood before including it; it is
not required for the core feedback.

## Notes

- Reproduced on Cloud Spanner DBaaS (2026-06-13, via `spanner-mycli EXPLAIN`,
  using the self-contained `SeekableKeySizeDemo` table above, since dropped)
  and on Spanner Omni `2026.r1-beta`. The behaviour is not Omni-specific.
- Framed as a documentation / diagnostic-clarity gap and a request for
  clarification, not a claim that the runtime behaviour is a bug. Reporting `0`
  for a point seek may be intentional internally; the issue is that it is
  undocumented and counterintuitive given the field name and the documented
  definitions.
