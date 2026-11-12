// Package spanalyzer derives Cloud Spanner GoogleSQL query result row types
// from Spanner DDL.
//
// BuildSchemaCatalog parses DDL into the package's schema model. Analyzer is
// the convenience API for constructing a GoogleSQL frontend catalog and
// converting analyzed statement or expression types into Spanner protobuf
// metadata. Lower-level catalog, helper, and conversion APIs are available for
// callers that need to compose those steps themselves.
//
// The API is experimental; the v1 contract has not been frozen.
//
// # Mutation counts (preview)
//
// NewMutationCountAnalyzer constructs a non-executing write-impact catalog.
// MutationCountAnalyzer.AnalyzeMutationsJSON accepts a JSON array of Spanner
// wire mutations, including queue Send/Ack, and AnalyzeDML handles ordinary
// GoogleSQL writes. AnalyzeWriteShape reports configured writes without values.
// AnalyzeTransaction accepts ordered DML and buffered mutations together. Its
// whole-transaction observation refines only the total, while per-statement
// and buffered-commit quota boundaries stay separate. Related writes retain
// unknown combined bounds rather than assuming cross-stage additivity.
// INSERT OR IGNORE retains affected-row uncertainty; INSERT SELECT accepts
// cardinality facts without executing or semantically analyzing its source.
// Zero affected-row facts can narrow successful DML to zero, including queues;
// missing Ack and zero-row queue queries have separate internal accounting.
// Every MutationCountReport preserves the schema interval, optional old-row
// assertions, and an optional input-qualified CommitStats observation separately.
// Unknown affected rows or internal queue effects retain a nil upper bound.
// AnalyzeQueueRead explicitly reports RECEIVE/RENEWLEASE effects as unknown;
// their query responses are not caller CommitStats and do not establish zero.
//
// This mutable v0alpha1 projection estimates mutation units, not bytes, latency,
// contention or commit success. DML statements have separate mutation limits;
// their cumulative transaction statistic is not a per-statement measurement.
// An observed execution never supplies a universal price for future writes.
// Old images and index snapshots require caller-provided consistency/freshness.
// MutationCountIndexSnapshot binds a complete regular-index snapshot to the
// exact DDL digest. IndexCatalog exposes independent copies of the same index
// facts used by accounting, including physical managed names and explicitly
// inferred constraint selectors, for plan collectors and other consumers.
// It never synthesizes generated names or performs metadata RPCs.
// Generated-column dependencies propagate to indexes, including virtual columns.
// Expressions and automatic timestamp values are not evaluated, so old images
// cannot establish their new values. Unsupported syntax fails explicitly.
//
// An update's cost depends on the indexes affected by its target columns,
// including generated dependencies. Index-key changes can remove an old entry
// and insert a new one; STORING writes and NULL-filtered membership have their
// own effects. Updating unrelated columns need not maintain an index. Therefore
// a table-wide index count is not a constant multiplier for every update.
// See the AnalyzeMutationsJSON_indexEffects example for a partial-update
// comparison under explicitly supplied old-row assertions.
package spanalyzer
