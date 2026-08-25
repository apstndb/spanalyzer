package infoschem

// SchemaGrant captures a GRANT USAGE ON SCHEMA statement.
//
// There is no dedicated INFORMATION_SCHEMA table that exposes schema-level
// usage grants in the emulator, so this struct is used for AST round-trip
// only. The privilege type is always USAGE.
type SchemaGrant struct {
	SchemaName string // empty when IsDefault is true
	Grantee    string
	IsDefault  bool // true for "USAGE ON SCHEMA DEFAULT"
}
