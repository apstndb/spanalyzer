package infoschem

// AllSchemaGrant captures a GRANT ... ON ALL <object_type> IN SCHEMA statement.
//
// There is no dedicated INFORMATION_SCHEMA table that exposes these schema-wide
// grants as a single row, so this struct is used for AST round-trip only.
// ObjectType is one of "TABLES", "VIEWS", "CHANGE_STREAMS", or "SEQUENCES".
type AllSchemaGrant struct {
	ObjectType    string // "TABLES", "VIEWS", "CHANGE_STREAMS", "SEQUENCES"
	PrivilegeType string // e.g. "SELECT", "INSERT", "UPDATE", "DELETE"
	SchemaName    string
	Grantee       string
}
