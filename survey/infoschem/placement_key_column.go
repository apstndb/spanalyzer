package infoschem

// PlacementKeyColumn identifies a PLACEMENT KEY column preserved from DDL.
//
// Production Spanner accepts PLACEMENT KEY, but INFORMATION_SCHEMA.COLUMNS
// does not expose a corresponding attribute. This struct is therefore used
// only for AST-originated round trips.
type PlacementKeyColumn struct {
	TableSchema string
	TableName   string
	ColumnName  string
}
