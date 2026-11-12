---
type: Research Note
title: Mixed mutation-count upper bounds for disjoint primary keys
description: A conservative same-table key proof, alternatives considered, runnable regressions and pinned Omni and Emulator mixed-transaction observations.
tags: [spanner, mutation-count, dml, primary-key, omni, emulator, bounds]
status: draft
sources:
  - id: execution-order
    resource: https://docs.cloud.google.com/spanner/docs/dml-versus-mutations
    title: DML before buffered mutations and separate mutation limits
  - id: commit-statistics
    resource: https://docs.cloud.google.com/spanner/docs/commit-statistics
    title: Whole-transaction cumulative mutation counts
  - id: equality
    resource: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/operators
    title: Comparison semantics, NULL and codepoint string equality
  - id: initial-model
    resource: ../write-analysis/mixed-transactions-and-evidence-20261112.md
    title: Initial conservative relationship guard and retained managed replay
  - id: implementation
    resource: ../../../mutation_count_keys.go
    title: Mandatory literal equalities, typed keys and relationship guards
  - id: regressions
    resource: ../../../mutation_count_transaction_test.go
    title: Composite keys, normalization, partial keys and multi-stage guards
  - id: probe
    resource: ../../../tools/mutation-count-lab/mixed_transaction_integration_test.go
    title: Owned local runtimes, captured wire inputs and committed-key checks
  - id: observations
    resource: evidence/mixed-disjoint-keys-20261112/observations.json
    title: Compact dated counts, bounds, runtime identities and bootstrap failure
---

# Mixed mutation-count upper bounds for disjoint primary keys

The feedback was checked against the implementation derived from main
`439bddbb9fa99b0df1d4f2c39e62c6c5e2175f17`. Its original `c38773c` example
still lost both the buffered upper bound and the combined upper bound solely
because the two stages targeted the same table. The root library and analyzer
CLI now preserve the component upper sum when every related table passes a
complete disjoint-key proof. This remains a conditional experimental bound,
not an exact-count or quota-gate guarantee.

## Alternatives and scope

| Alternative | Benefit | Limitation and decision |
| --- | --- | --- |
| Always sum component uppers | Smallest implementation; accepts all predicates | Does not establish independence of FK/cascade/queue effects or stage-specific state; rejected |
| Keep every same-table total unknown | Preserves the original conservative guard | Discards useful finite bounds for explicitly distinct keys; replaced only in the proven subset |
| Prove complete key disjointness on isolated ordinary tables | Addresses the supplied example without interpreting general SQL | Selected; unknown keys and schema relationships retain the original fallback |
| Propagate old rows or execute predicates locally | Could cover expressions, parameters, ranges and cascades | Requires a substantially broader execution/state model; deferred |

For the selected case, the total is `[0, sum(component uppers)]`. The DML
statement and buffered-commit intervals and quota boundaries remain separate.
No exact additivity or lower-bound improvement is inferred. An unknown or
overflowing component upper remains unknown. A whole-transaction observation
can still narrow only the total.

The proof accepts literal INSERT keys, single full-key Mutation deletes,
ordinary explicit write keys, and mandatory unqualified `column = literal`
terms under AND in UPDATE/DELETE. Equality can be reversed; surrounding AND
filters can narrow the candidate row further. It compares the entire ordered
composite key, not a prefix or just the last column. INT64 wire encodings are
normalized before comparison, so `"01"` and SQL `1` cannot be declared disjoint.
The current scalar subset is INT64, STRING, BOOL and DATE; unsupported literal
forms or key types fail closed. Timestamp precision, SQL BYTES versus base64,
and float equality are not guessed from raw JSON.

Queues, FK relationships on either side, interleaved parents/children and
unmodeled schema history are excluded. Every related table and DML statement
must pass; a nonoverlapping first statement does not excuse an overlapping
later statement. Parameters, qualified identifiers, ranges, OR/NOT predicates,
INSERT SELECT and unresolved/default keys do not supply a complete proof.
Old-row or cardinality assertions never supply a missing predicate key.
UPDATE/DELETE cardinality remains an explicit fact; a complete equality is
used for nonoverlap only and does not silently assert that a row exists.

## Runnable feedback example

```sh
go run ./cmd/spanner-analyzer --mode mutation_count \
  --ddl testdata/mutation-count/transaction-disjoint.sql \
  --mutation-transaction-file testdata/mutation-count/transaction-disjoint.json \
  --mutation-facts-file testdata/mutation-count/transaction-disjoint-facts.json \
  --output json
```

The fixture updates `Events(Shard=1, Id=2)` and inserts `(1,3)` with only its
two primary-key columns. One affected DML row and its old Note are explicit
assertions. The DML interval is `[3,3]`, the buffered interval `[2,2]`, and the
combined interval `[0,5]`, with reason
`mixed_stage_disjoint_primary_keys_upper_bound`. Omitting the row-count fact
still leaves the DML and total upper bounds unknown.

## Local observations, November 12

The opt-in probe uses the host-platform manifest from `runtime_targets.json`,
captures actual SDK CommitRequest mutations, seeds outside measured commits,
and checks committed primary-key sets. It creates an owned container and a
random fixture database, then closes the clients and removes the container.
It never attaches to an endpoint or uses managed credentials.

| Case | Omni LTS.2 CommitStats | Modeled combined interval | Emulator 1.5.59 |
| --- | ---: | --- | --- |
| Feedback UPDATE plus disjoint key-only Insert | 5 | `[0,5]` | Commit and final keys succeed; statistics absent |
| UPDATE plus disjoint Update / Upsert / Replace | 6 each | `[0,6]` each | Same final keys; statistics absent |
| UPDATE plus disjoint Delete | 4 | `[0,4]` | Same final keys; statistics absent |
| DML INSERT plus disjoint Insert | 4 | `[0,4]` | Same final keys; statistics absent |
| DML DELETE plus disjoint Insert | 3 | `[0,3]` | Same final keys; statistics absent |
| Same primary key in both stages | 6 | `[0,unknown]` | Same final keys; statistics absent |
| Parameterized / unresolved composite-key component | 5 each | `[0,unknown]` each | Same final keys; statistics absent |
| DML releases a unique index key, buffered Insert reuses it on another PK | 9 | `[0,9]` | Same final keys; statistics absent |

Both corrected local runs pass all 11 cases. Emulator's missing CommitStats
provide no numerical agreement or zero-count evidence. An initial Emulator
bootstrap failed before any case because the probe redundantly attempted to
create the already bootstrapped instance. The corrected probe creates only
the fixture database; the failure and successful retry are distinct in the
compact receipt. Owned containers were removed after each run.

Reproduce from the repository root with the current workspace enabled:

```sh
export DOCKER_HOST="$(docker context inspect --format '{{.Endpoints.docker.Host}}')"
export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
unset SPANNER_EMULATOR_HOST
SPANALYZER_MIXED_MUTATION_BACKEND=omni \
  go -C tools test -tags=integration -count=1 -timeout=7m -v \
  -run '^TestIntegrationMixedMutationPrimaryKeys$' ./mutation-count-lab
SPANALYZER_MIXED_MUTATION_BACKEND=emulator \
  go -C tools test -tags=integration -count=1 -timeout=7m -v \
  -run '^TestIntegrationMixedMutationPrimaryKeys$' ./mutation-count-lab
```

The source-backed execution order and local measurements support this bounded
implementation; they do not establish a new managed-service measurement or
universal cross-stage formula. The earlier managed queue captures continue to
replay offline, with same-key queue interactions still unknown before an exact
whole-transaction observation. Full downloaded documents and routine test logs
remain in task scratch; the receipt keeps only identities and observations.
