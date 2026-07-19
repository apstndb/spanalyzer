// Package planvocab detects previously unobserved Cloud Spanner QueryPlan
// operator vocabulary.
//
// The embedded catalog records known associations between a PlanNode display
// name, selected metadata keys and values, and child-link shapes. Inspect is a
// pure function for callers that want to handle findings themselves. Observer
// adds process-local deduplication and structured slog output.
// FindMatchingOperators supports positive assertions about metadata and child
// links on the same operator node.
//
// The catalog is observational rather than a stable Spanner wire contract.
// Its source revision is available through EmbeddedCatalogInfo. Inspect assumes
// structurally sound PLAN output; PROFILE execution statistics are out of scope.
package planvocab
