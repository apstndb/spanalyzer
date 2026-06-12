# Response to GPT-5.5 Pro Feedback

Thank you for the detailed design review. I agree with the main thrust of the
feedback: `spanner-query-gen` should stay narrow and should not become a general
ORM, query builder, or broad CRUD generator. The design has been revised to make
that product boundary explicit.

## Accepted and Reflected

The following points were incorporated into
`cmd/spanner-query-gen/DESIGN.md`.

### Product Boundary

The design now states that `spanner-query-gen` converts statically declared
GoogleSQL queries and Spanner writes into Go types and helpers without runtime
magic. It is explicitly not an ORM, a query builder, or a broad CRUD generator.

This became the guiding constraint for the roadmap.

### `client: both` Is Too Broad

The feedback correctly pointed out that `client: both` mixes too many concepts:
DTO type choices, Spanner decoding, Spanner mutation encoding, DML parameters,
BigQuery row loading, and future query methods.

The design now treats `client: both` as an early compact option and describes a
future split into:

- `gen`: Go rendering options
- `runtime`: Spanner and BigQuery runtime surfaces
- `dto`: DTO sharing and nullable wrapper policy

Cross-dialect DTO sharing is also called out as something that should become
explicit as the generator matures.

### Plan Model and `explain-plan` Moved Earlier

The roadmap now makes the plan model a Phase 1 concern. The plan should be
dumpable as YAML or JSON and should explain:

- expanded SQL for table and index shorthands
- result fields
- DTO merge decisions
- nullability confidence
- write column choices
- field provenance

This directly addresses the concern that shorthand expansion and DTO merging
must remain reviewable.

### Nullability Confidence and Provenance

The design now says the internal type model should capture nullability
confidence and field provenance. The documented confidence categories include:

- `non_null_by_schema`
- `non_null_by_expression`
- `nullable_by_schema`
- `nullable_by_join`
- `unknown`
- `user_required_override`

The generator should remain conservative, but `explain-plan` should make the
reason for nullable wrappers visible.

### Write Semantics Split

The roadmap and example config were revised so upsert-like operations can model
insert columns, update masks, and conflict strategy separately.

The design now treats `INSERT OR UPDATE`, `INSERT OR REPLACE`, and future
`ON CONFLICT` support as distinct semantic choices rather than overloading one
flat operation plus `update_mask`.

### DML and Mutation Policy

The design now keeps DML helpers and mutation helpers as separate API shapes and
adds a roadmap item for generated docs or lint rules that discourage mixing DML
and mutations in the same transaction.

### Validation Moved Earlier

`vet` is now Phase 2 rather than a late validation workflow. The first rule set
should be driven by this tool's specific risks:

- update masks
- cross-dialect unknown types
- duplicate generated names
- risky DTO sharing
- strict nullability policy where configured

### BigQuery Methods Deferred

The roadmap now limits early query methods to Spanner. BigQuery method
generation is kept as later work, after DTO loading and TableSchema metadata are
robust enough.

### Minimal SQL Annotations Only

The design now describes SQL file support as "sqlc-like minimal comments" rather
than sqlc compatibility. The expected initial form is:

```sql
-- name: GetSinger :one
SELECT SingerId, FirstName
FROM Singers
WHERE SingerId = @SingerId;
```

The design explicitly avoids promising sqlc macro compatibility.

### Type Overrides Moved Earlier Than Broad Model Generation

The roadmap now places type and field overrides before opt-in Spanner model
generation. This reflects the adoption risk around NUMERIC, JSON, BYTES,
TIMESTAMP, proto, enum, and BigQuery BIGNUMERIC mappings.

## Deliberately Not Adopted or Deferred

Some suggestions were intentionally not added as early roadmap commitments.

### Full sqlc Macro Compatibility

`sqlc.arg`, `sqlc.narg`, `sqlc.embed`, and `sqlc.slice` were not adopted as
planned features. GoogleSQL already has named parameters, and runtime SQL
expansion would weaken the "SQL is the contract" principle.

The design leaves room for GoogleSQL-native conveniences later, but not macro
compatibility as a goal.

### Broad yo-Style Model and CRUD Generation

Table models and index read helpers remain in the roadmap, but only as opt-in
Spanner model generation. Broad all-table model generation and CRUD generation
are intentionally not made central to the product.

### Template Customization and Plugin System

These remain possible later work, but they were not moved earlier. Exposing
templates or plugin APIs before the plan model stabilizes would freeze internal
details too soon.

### BigQuery Query Method Generation

BigQuery DTO and TableSchema generation remain important, but BigQuery execution
methods were deferred. BigQuery should not be forced into the same runtime
abstraction as Spanner.

### Concrete Transaction Runner Interface

The feedback's transaction-aware Spanner runner direction is good, but the
specific interface shape was not committed in the design. That should be decided
when Spanner query method generation starts and the generated method surface is
being implemented.

### Detailed SQLBoiler, Bob, ent, and xo Comparisons

Those comparisons were useful as background, but they were not added to the
design document. The document stays focused on sqlc, yo, and the boundary of
this tool.

### Generic `no-select-star` as an Early Rule

`no-select-star` was not included in the first vet rule set. `SELECT *` is not
always wrong in this tool, especially where shorthand or explicitly reviewed SQL
is used. Early rules should target this tool's higher-risk behavior first:
update masks, cross-dialect types, DTO sharing, and nullability.

## Resulting Roadmap Shape

The revised roadmap is now:

1. Stabilize the current generator and plan model, including `explain-plan`.
2. Add validation and clarify write semantics.
3. Add Spanner query methods.
4. Add minimal SQL files and annotations.
5. Add type and field overrides.
6. Add opt-in Spanner model generation.
7. Add later runtime and customization work, including BigQuery methods and
   possible plugin/template support.

This keeps the tool centered on its strongest niche: statically analyzed
GoogleSQL query and write declarations, with Spanner semantics and BigQuery
federation handled explicitly.
