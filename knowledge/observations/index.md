# Observations

- [GoogleSQL hint placement inventory](googlesql-hint-placement.md) -
  normalized generic hint placements in a pinned upstream GoogleSQL grammar,
  separated from frontend, runtime-syntax, and plan-effect evidence.
- [Spanner set operations](spanner-set-operations.md) - SQL-standard
  multiplicity semantics and environment-scoped Spanner plan observations for
  `UNION`, `INTERSECT`, and `EXCEPT` with `ALL` and `DISTINCT`.
- [SPANNER_SYS live-primary catalog](spanner-sys-live-primary-catalog.md) -
  pinned managed and Omni metadata, fail-closed projection policy, and the
  replacement of the handwritten analyzer registry.
- [INFORMATION_SCHEMA managed-primary catalog](information-schema-managed-primary-catalog.md) -
  point-in-time managed selection, digest-pinned Omni and Emulator comparison
  captures, and rolling-column queryability.
