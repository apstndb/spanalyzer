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
package spanalyzer
