package infoschem

// SequenceGrant captures a GRANT ... ON SEQUENCE statement.
//
// Production Spanner accepts sequence grants and GetDatabaseDdl returns them,
// but no dedicated INFORMATION_SCHEMA table exposes the grant rows. This
// struct is therefore used only for AST-originated round trips.
type SequenceGrant struct {
	SequenceSchema string
	SequenceName   string
	PrivilegeType  string // "SELECT" or "UPDATE"
	Grantee        string
}
