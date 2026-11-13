// Package spanalyzer derives Cloud Spanner GoogleSQL query result row types
// from Spanner DDL and provides experimental analysis and write-impact APIs.
//
// The API is experimental; the v1 contract has not been frozen. GoogleSQL frontend
// means the analyzer/catalog library formerly named ZetaSQL; Spanner GoogleSQL is
// the database dialect. The parser and frontend are implementation details, though
// experimental low-level APIs expose some of their types.
//
// # Query analysis
//
// [NewAnalyzerFromDDL] constructs the convenience [Analyzer].
// [Analyzer.RowTypeForStatement] returns Spanner protobuf result metadata
// without querying a database. The
// example for [Analyzer.RowTypeForStatement] shows a complete DDL-to-row-type flow.
// Use lower-level catalogs, helpers, and converters to compose individual stages.
// GoogleSQL is initialized once per process through the frontend binding.
//
// # Composable components
//
// [BuildSchemaCatalog] parses Spanner DDL into this project's Spanner schema
// catalog.
//
// [BuildGoogleSQLCatalogFromSpannerCatalog] and [BuildGoogleSQLCatalogFromDDL]
// convert that schema into a GoogleSQL frontend catalog, analyzer options, and
// type factory.
//
// [BuildBigQueryGoogleSQLCatalogFromDDL] converts BigQuery DDL into a
// GoogleSQL frontend catalog.
//
// [GoogleSQLHelper] wraps parse, analyze, unparse, and resolved AST debug
// operations against that catalog.
//
// [RowTypeFromAnalyzerOutput], [RowTypeFromResolvedQuery], and
// [TypeFromAnalyzerOutput] convert GoogleSQL analyzer results into Cloud
// Spanner protobuf metadata.
//
// [BigQueryTableSchemaFromAnalyzerOutput],
// [BigQueryTableSchemaFromResolvedQuery], and
// [BigQueryTableFieldSchemaFromGoogleSQLType] convert GoogleSQL analyzer types
// into BigQuery REST TableSchema shaped metadata.
//
// [Analyzer] remains a convenience wrapper that wires these components together
// for the CLI-style row type use case.
//
// [BigQueryAnalyzer] does the same for BigQuery TableSchema output.
// Connection-specific Spanner analyzers can be attached with
// [BigQueryAnalyzer.SetExternalQueryAnalyzers] to infer EXTERNAL_QUERY result schemas.
//
// # Analysis boundaries
//
// PROTO BUNDLE support requires descriptor set files. DDL alone is not enough
// to analyze proto fields.
//
// Cloud Spanner and the [Cloud Spanner emulator] use the GoogleSQL frontend's
// native MakeProtoType and MakeEnumType APIs
// with descriptors from the active proto bundle. This project now loads the
// supplied descriptor set into the GoogleSQL frontend descriptor pool and uses
// those native proto and enum types when building the analyzer catalog.
//
// Proto and enum query outputs are converted back to Spanner row metadata when
// possible, including nested proto fields selected as values.
//
// Property graph DDL registers node and edge tables, labels, and direct column
// property definitions through the go-googlesql SimpleGraph* constructors.
// More advanced Spanner graph metadata, including arbitrary
// property expressions and dynamic labels/properties, is still limited.
//
// Some Spanner-specific functions are registered locally because they are not
// included in the default go-googlesql builtin function set. This
// includes commit timestamp, sequence, search, TOKENLIST, and AI helper
// functions needed for query analysis.
//
// ML.PREDICT is registered as an analyzer-only table-valued function. It
// models schema only, returns model output columns followed by non-duplicated
// input relation columns, and does not execute prediction logic. It supports
// ML.PREDICT, PREDICT, and SAFE.ML.PREDICT names.
//
// TOKENLIST is supported as an internal analysis type for search expressions,
// but Cloud Spanner result sets cannot return TOKENLIST, so the Cloud Spanner
// protobuf API has no TypeCode for it.
//
// Named arguments for locally registered Spanner functions are normalized before
// analysis because the current Go binding does not expose the GoogleSQL
// frontend's argument name setters.
//
// BigQuery mode currently registers ordinary BigQuery tables and views from DDL
// but does not model every DDL side effect. Dataset DDL, indexes, and drops are
// ignored because they do not change query result types in the current catalog
// model.
//
// BigQuery bigquery_type output is derived from resolved GoogleSQL types. It
// can represent repeated and nested fields, but query result nullability is not
// tracked, so non-repeated query output fields are emitted as NULLABLE.
//
// BigQuery-to-Spanner federated queries use the EXTERNAL_QUERY table-valued
// function. The current implementation identifies EXTERNAL_QUERY calls by
// literal connection and SQL arguments and delegates row type inference to the
// matching Spanner analyzer. It does not evaluate connection options,
// permissions, PostgreSQL-dialect Spanner SQL, or non-literal dynamic SQL
// expressions.
//
// # Mutation counts (preview)
//
// NewMutationCountAnalyzer constructs a non-executing write-impact catalog.
// MutationCountAnalyzer.AnalyzeMutationsJSON accepts a JSON array of Spanner
// wire mutations, including queue Send/Ack, and AnalyzeDML handles ordinary
// GoogleSQL writes. AnalyzeWriteShape reports configured writes without values.
// AnalyzeTransaction accepts ordered DML and buffered mutations together. Its
// whole-transaction observation refines only the total, while per-statement
// and buffered-commit quota boundaries stay separate. Disjoint complete primary
// keys on ordinary tables without FK/interleave relationships retain the sum
// of component upper bounds, with a zero combined lower bound. Other related
// writes retain unknown combined bounds. This does not assume exact additivity
// or infer UPDATE/DELETE cardinality from a predicate.
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
// DDL-only FK bounds retain potential backing indexes for nullable referencing
// primary keys and composite keys whose referenced order can change. A complete
// observed snapshot narrows those bounds and counts a shared physical index once.
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
//
// [Cloud Spanner emulator]: https://github.com/GoogleCloudPlatform/cloud-spanner-emulator
package spanalyzer
