package infoschem

import "cloud.google.com/go/spanner"

// All structs represent the real (production) Spanner superset of columns.
// Nullable columns use pointer types. Emulator-absent columns will be zero-valued
// when read via spanner.Row.ToStructLenient.

// ChangeStream represents INFORMATION_SCHEMA.CHANGE_STREAMS.
type ChangeStream struct {
	ChangeStreamCatalog string `spanner:"CHANGE_STREAM_CATALOG"`
	ChangeStreamSchema  string `spanner:"CHANGE_STREAM_SCHEMA"`
	ChangeStreamName    string `spanner:"CHANGE_STREAM_NAME"`
	All                 bool   `spanner:"ALL"`
}

// ChangeStreamColumn represents INFORMATION_SCHEMA.CHANGE_STREAM_COLUMNS.
type ChangeStreamColumn struct {
	ChangeStreamCatalog string `spanner:"CHANGE_STREAM_CATALOG"`
	ChangeStreamSchema  string `spanner:"CHANGE_STREAM_SCHEMA"`
	ChangeStreamName    string `spanner:"CHANGE_STREAM_NAME"`
	TableCatalog        string `spanner:"TABLE_CATALOG"`
	TableSchema         string `spanner:"TABLE_SCHEMA"`
	TableName           string `spanner:"TABLE_NAME"`
	ColumnName          string `spanner:"COLUMN_NAME"`
}

// ChangeStreamOption represents INFORMATION_SCHEMA.CHANGE_STREAM_OPTIONS.
type ChangeStreamOption struct {
	ChangeStreamCatalog string `spanner:"CHANGE_STREAM_CATALOG"`
	ChangeStreamSchema  string `spanner:"CHANGE_STREAM_SCHEMA"`
	ChangeStreamName    string `spanner:"CHANGE_STREAM_NAME"`
	OptionName          string `spanner:"OPTION_NAME"`
	OptionType          string `spanner:"OPTION_TYPE"`
	OptionValue         string `spanner:"OPTION_VALUE"`
}

// ChangeStreamPrivilege represents INFORMATION_SCHEMA.CHANGE_STREAM_PRIVILEGES (real only).
type ChangeStreamPrivilege struct {
	ChangeStreamCatalog string `spanner:"CHANGE_STREAM_CATALOG"`
	ChangeStreamSchema  string `spanner:"CHANGE_STREAM_SCHEMA"`
	ChangeStreamName    string `spanner:"CHANGE_STREAM_NAME"`
	PrivilegeType       string `spanner:"PRIVILEGE_TYPE"`
	Grantee             string `spanner:"GRANTEE"`
}

// ChangeStreamTable represents INFORMATION_SCHEMA.CHANGE_STREAM_TABLES.
type ChangeStreamTable struct {
	ChangeStreamCatalog string `spanner:"CHANGE_STREAM_CATALOG"`
	ChangeStreamSchema  string `spanner:"CHANGE_STREAM_SCHEMA"`
	ChangeStreamName    string `spanner:"CHANGE_STREAM_NAME"`
	TableCatalog        string `spanner:"TABLE_CATALOG"`
	TableSchema         string `spanner:"TABLE_SCHEMA"`
	TableName           string `spanner:"TABLE_NAME"`
	AllColumns          bool   `spanner:"ALL_COLUMNS"`
}

// CheckConstraint represents INFORMATION_SCHEMA.CHECK_CONSTRAINTS.
type CheckConstraint struct {
	ConstraintCatalog string  `spanner:"CONSTRAINT_CATALOG"`
	ConstraintSchema  string  `spanner:"CONSTRAINT_SCHEMA"`
	ConstraintName    string  `spanner:"CONSTRAINT_NAME"`
	CheckClause       string  `spanner:"CHECK_CLAUSE"`
	SpannerState      *string `spanner:"SPANNER_STATE"`
}

// Column represents INFORMATION_SCHEMA.COLUMNS (real superset).
// Emulator v1.5.53+ exposes ON_UPDATE_EXPRESSION too, but still uses a
// different column order around IS_HIDDEN / GENERATION_EXPRESSION / IS_STORED.
type Column struct {
	TableCatalog             string  `spanner:"TABLE_CATALOG"`
	TableSchema              string  `spanner:"TABLE_SCHEMA"`
	TableName                string  `spanner:"TABLE_NAME"`
	ColumnName               string  `spanner:"COLUMN_NAME"`
	OrdinalPosition          int64   `spanner:"ORDINAL_POSITION"`
	ColumnDefault            *string `spanner:"COLUMN_DEFAULT"`
	DataType                 *string `spanner:"DATA_TYPE"`
	IsNullable               string  `spanner:"IS_NULLABLE"`
	SpannerType              string  `spanner:"SPANNER_TYPE"`
	IsGenerated              string  `spanner:"IS_GENERATED"`
	GenerationExpression     *string `spanner:"GENERATION_EXPRESSION"`
	IsStored                 *string `spanner:"IS_STORED"`
	IsHidden                 bool    `spanner:"IS_HIDDEN"`
	SpannerState             *string `spanner:"SPANNER_STATE"`
	IsIdentity               *string `spanner:"IS_IDENTITY"`
	IdentityGeneration       *string `spanner:"IDENTITY_GENERATION"`
	IdentityKind             *string `spanner:"IDENTITY_KIND"`
	IdentityStartWithCounter *string `spanner:"IDENTITY_START_WITH_COUNTER"`
	IdentitySkipRangeMin     *string `spanner:"IDENTITY_SKIP_RANGE_MIN"`
	IdentitySkipRangeMax     *string `spanner:"IDENTITY_SKIP_RANGE_MAX"`
	OnUpdateExpression       *string `spanner:"ON_UPDATE_EXPRESSION"`
}

// ColumnColumnUsage represents INFORMATION_SCHEMA.COLUMN_COLUMN_USAGE.
type ColumnColumnUsage struct {
	TableCatalog    string `spanner:"TABLE_CATALOG"`
	TableSchema     string `spanner:"TABLE_SCHEMA"`
	TableName       string `spanner:"TABLE_NAME"`
	ColumnName      string `spanner:"COLUMN_NAME"`
	DependentColumn string `spanner:"DEPENDENT_COLUMN"`
}

// ColumnOption represents INFORMATION_SCHEMA.COLUMN_OPTIONS.
type ColumnOption struct {
	TableCatalog string `spanner:"TABLE_CATALOG"`
	TableSchema  string `spanner:"TABLE_SCHEMA"`
	TableName    string `spanner:"TABLE_NAME"`
	ColumnName   string `spanner:"COLUMN_NAME"`
	OptionName   string `spanner:"OPTION_NAME"`
	OptionType   string `spanner:"OPTION_TYPE"`
	OptionValue  string `spanner:"OPTION_VALUE"`
}

// ColumnPrivilege represents INFORMATION_SCHEMA.COLUMN_PRIVILEGES (real only).
type ColumnPrivilege struct {
	TableCatalog  string `spanner:"TABLE_CATALOG"`
	TableSchema   string `spanner:"TABLE_SCHEMA"`
	TableName     string `spanner:"TABLE_NAME"`
	ColumnName    string `spanner:"COLUMN_NAME"`
	PrivilegeType string `spanner:"PRIVILEGE_TYPE"`
	Grantee       string `spanner:"GRANTEE"`
}

// ConstraintColumnUsage represents INFORMATION_SCHEMA.CONSTRAINT_COLUMN_USAGE.
type ConstraintColumnUsage struct {
	TableCatalog      string `spanner:"TABLE_CATALOG"`
	TableSchema       string `spanner:"TABLE_SCHEMA"`
	TableName         string `spanner:"TABLE_NAME"`
	ColumnName        string `spanner:"COLUMN_NAME"`
	ConstraintCatalog string `spanner:"CONSTRAINT_CATALOG"`
	ConstraintSchema  string `spanner:"CONSTRAINT_SCHEMA"`
	ConstraintName    string `spanner:"CONSTRAINT_NAME"`
}

// ConstraintTableUsage represents INFORMATION_SCHEMA.CONSTRAINT_TABLE_USAGE.
type ConstraintTableUsage struct {
	TableCatalog      string `spanner:"TABLE_CATALOG"`
	TableSchema       string `spanner:"TABLE_SCHEMA"`
	TableName         string `spanner:"TABLE_NAME"`
	ConstraintCatalog string `spanner:"CONSTRAINT_CATALOG"`
	ConstraintSchema  string `spanner:"CONSTRAINT_SCHEMA"`
	ConstraintName    string `spanner:"CONSTRAINT_NAME"`
}

// DatabaseOption represents INFORMATION_SCHEMA.DATABASE_OPTIONS.
type DatabaseOption struct {
	CatalogName string `spanner:"CATALOG_NAME"`
	SchemaName  string `spanner:"SCHEMA_NAME"`
	OptionName  string `spanner:"OPTION_NAME"`
	OptionType  string `spanner:"OPTION_TYPE"`
	OptionValue string `spanner:"OPTION_VALUE"`
}

// Index represents INFORMATION_SCHEMA.INDEXES (real superset).
// Emulator v1.5.53+ exposes FILTER, but SEARCH_PARTITION_BY, SEARCH_ORDER_BY,
// and SEARCH_UNNEST are still absent from Emulator v1.5.56.
type Index struct {
	TableCatalog      string   `spanner:"TABLE_CATALOG"`
	TableSchema       string   `spanner:"TABLE_SCHEMA"`
	TableName         string   `spanner:"TABLE_NAME"`
	IndexName         string   `spanner:"INDEX_NAME"`
	IndexType         string   `spanner:"INDEX_TYPE"`
	ParentTableName   string   `spanner:"PARENT_TABLE_NAME"`
	IsUnique          bool     `spanner:"IS_UNIQUE"`
	IsNullFiltered    bool     `spanner:"IS_NULL_FILTERED"`
	IndexState        *string  `spanner:"INDEX_STATE"`
	Filter            *string  `spanner:"FILTER"`
	SpannerIsManaged  bool     `spanner:"SPANNER_IS_MANAGED"`
	SearchPartitionBy []string `spanner:"SEARCH_PARTITION_BY"` // real only
	SearchOrderBy     []string `spanner:"SEARCH_ORDER_BY"`     // real only
	SearchUnnest      []string `spanner:"SEARCH_UNNEST"`       // real and newer Omni only
}

// IndexColumn represents INFORMATION_SCHEMA.INDEX_COLUMNS.
type IndexColumn struct {
	TableCatalog    string  `spanner:"TABLE_CATALOG"`
	TableSchema     string  `spanner:"TABLE_SCHEMA"`
	TableName       string  `spanner:"TABLE_NAME"`
	IndexName       string  `spanner:"INDEX_NAME"`
	IndexType       string  `spanner:"INDEX_TYPE"`
	ColumnName      string  `spanner:"COLUMN_NAME"`
	OrdinalPosition *int64  `spanner:"ORDINAL_POSITION"`
	ColumnOrdering  *string `spanner:"COLUMN_ORDERING"`
	IsNullable      *string `spanner:"IS_NULLABLE"`
	SpannerType     *string `spanner:"SPANNER_TYPE"`
	Expression      *string `spanner:"EXPRESSION"` // real and newer Omni only
}

// IndexOption represents INFORMATION_SCHEMA.INDEX_OPTIONS (real only).
type IndexOption struct {
	TableCatalog string `spanner:"TABLE_CATALOG"`
	TableSchema  string `spanner:"TABLE_SCHEMA"`
	TableName    string `spanner:"TABLE_NAME"`
	IndexName    string `spanner:"INDEX_NAME"`
	IndexType    string `spanner:"INDEX_TYPE"`
	OptionName   string `spanner:"OPTION_NAME"`
	OptionType   string `spanner:"OPTION_TYPE"`
	OptionValue  string `spanner:"OPTION_VALUE"`
}

// KeyColumnUsage represents INFORMATION_SCHEMA.KEY_COLUMN_USAGE.
type KeyColumnUsage struct {
	ConstraintCatalog          string `spanner:"CONSTRAINT_CATALOG"`
	ConstraintSchema           string `spanner:"CONSTRAINT_SCHEMA"`
	ConstraintName             string `spanner:"CONSTRAINT_NAME"`
	TableCatalog               string `spanner:"TABLE_CATALOG"`
	TableSchema                string `spanner:"TABLE_SCHEMA"`
	TableName                  string `spanner:"TABLE_NAME"`
	ColumnName                 string `spanner:"COLUMN_NAME"`
	OrdinalPosition            int64  `spanner:"ORDINAL_POSITION"`
	PositionInUniqueConstraint *int64 `spanner:"POSITION_IN_UNIQUE_CONSTRAINT"`
}

// LocalityGroupOption represents INFORMATION_SCHEMA.LOCALITY_GROUP_OPTIONS.
type LocalityGroupOption struct {
	LocalityGroupName string  `spanner:"LOCALITY_GROUP_NAME"`
	OptionName        string  `spanner:"OPTION_NAME"`
	OptionValue       *string `spanner:"OPTION_VALUE"`
	// OptionType is AST-facing metadata. INFORMATION_SCHEMA does not expose it,
	// so values loaded from the service leave it empty and are treated as strings.
	OptionType string `spanner:"-"`
}

// LocalityGroup represents a CREATE LOCALITY GROUP statement. INFORMATION_SCHEMA
// exposes only option rows, so live discovery cannot populate optionless groups.
type LocalityGroup struct {
	LocalityGroupName string
}

// Model represents INFORMATION_SCHEMA.MODELS.
type Model struct {
	ModelCatalog string `spanner:"MODEL_CATALOG"`
	ModelSchema  string `spanner:"MODEL_SCHEMA"`
	ModelName    string `spanner:"MODEL_NAME"`
	IsRemote     bool   `spanner:"IS_REMOTE"`
}

// ModelColumn represents INFORMATION_SCHEMA.MODEL_COLUMNS.
type ModelColumn struct {
	ModelCatalog    string `spanner:"MODEL_CATALOG"`
	ModelSchema     string `spanner:"MODEL_SCHEMA"`
	ModelName       string `spanner:"MODEL_NAME"`
	ColumnKind      string `spanner:"COLUMN_KIND"`
	ColumnName      string `spanner:"COLUMN_NAME"`
	OrdinalPosition int64  `spanner:"ORDINAL_POSITION"`
	DataType        string `spanner:"DATA_TYPE"`
	IsExplicit      bool   `spanner:"IS_EXPLICIT"`
}

// ModelColumnOption represents INFORMATION_SCHEMA.MODEL_COLUMN_OPTIONS.
type ModelColumnOption struct {
	ModelCatalog string `spanner:"MODEL_CATALOG"`
	ModelSchema  string `spanner:"MODEL_SCHEMA"`
	ModelName    string `spanner:"MODEL_NAME"`
	ColumnKind   string `spanner:"COLUMN_KIND"`
	ColumnName   string `spanner:"COLUMN_NAME"`
	OptionName   string `spanner:"OPTION_NAME"`
	OptionType   string `spanner:"OPTION_TYPE"`
	OptionValue  string `spanner:"OPTION_VALUE"`
}

// ModelOption represents INFORMATION_SCHEMA.MODEL_OPTIONS.
type ModelOption struct {
	ModelCatalog string `spanner:"MODEL_CATALOG"`
	ModelSchema  string `spanner:"MODEL_SCHEMA"`
	ModelName    string `spanner:"MODEL_NAME"`
	OptionName   string `spanner:"OPTION_NAME"`
	OptionType   string `spanner:"OPTION_TYPE"`
	OptionValue  string `spanner:"OPTION_VALUE"`
}

// ModelPrivilege represents INFORMATION_SCHEMA.MODEL_PRIVILEGES (real only).
type ModelPrivilege struct {
	ModelCatalog  string `spanner:"MODEL_CATALOG"`
	ModelSchema   string `spanner:"MODEL_SCHEMA"`
	ModelName     string `spanner:"MODEL_NAME"`
	PrivilegeType string `spanner:"PRIVILEGE_TYPE"`
	Grantee       string `spanner:"GRANTEE"`
}

// Parameter represents INFORMATION_SCHEMA.PARAMETERS (real only).
type Parameter struct {
	SpecificCatalog  string  `spanner:"SPECIFIC_CATALOG"`
	SpecificSchema   string  `spanner:"SPECIFIC_SCHEMA"`
	SpecificName     string  `spanner:"SPECIFIC_NAME"`
	OrdinalPosition  int64   `spanner:"ORDINAL_POSITION"`
	ParameterName    *string `spanner:"PARAMETER_NAME"`
	DataType         *string `spanner:"DATA_TYPE"`
	SpannerType      *string `spanner:"SPANNER_TYPE"`
	ParameterDefault *string `spanner:"PARAMETER_DEFAULT"`
}

// Placement represents INFORMATION_SCHEMA.PLACEMENTS (real only).
type Placement struct {
	PlacementName string `spanner:"PLACEMENT_NAME"`
	IsDefault     bool   `spanner:"IS_DEFAULT"`
}

// PlacementOption represents INFORMATION_SCHEMA.PLACEMENT_OPTIONS (real only).
type PlacementOption struct {
	PlacementName string `spanner:"PLACEMENT_NAME"`
	OptionName    string `spanner:"OPTION_NAME"`
	OptionType    string `spanner:"OPTION_TYPE"`
	OptionValue   string `spanner:"OPTION_VALUE"`
}

// PropertyGraph represents INFORMATION_SCHEMA.PROPERTY_GRAPHS.
type PropertyGraph struct {
	PropertyGraphCatalog      string           `spanner:"PROPERTY_GRAPH_CATALOG"`
	PropertyGraphSchema       string           `spanner:"PROPERTY_GRAPH_SCHEMA"`
	PropertyGraphName         string           `spanner:"PROPERTY_GRAPH_NAME"`
	PropertyGraphMetadataJSON spanner.NullJSON `spanner:"PROPERTY_GRAPH_METADATA_JSON"`
}

// ReferentialConstraint represents INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS.
type ReferentialConstraint struct {
	ConstraintCatalog       string  `spanner:"CONSTRAINT_CATALOG"`
	ConstraintSchema        string  `spanner:"CONSTRAINT_SCHEMA"`
	ConstraintName          string  `spanner:"CONSTRAINT_NAME"`
	UniqueConstraintCatalog string  `spanner:"UNIQUE_CONSTRAINT_CATALOG"`
	UniqueConstraintSchema  string  `spanner:"UNIQUE_CONSTRAINT_SCHEMA"`
	UniqueConstraintName    string  `spanner:"UNIQUE_CONSTRAINT_NAME"`
	MatchOption             string  `spanner:"MATCH_OPTION"`
	UpdateRule              string  `spanner:"UPDATE_RULE"`
	DeleteRule              string  `spanner:"DELETE_RULE"`
	SpannerState            *string `spanner:"SPANNER_STATE"`
}

// Role represents INFORMATION_SCHEMA.ROLES (real only).
type Role struct {
	RoleName string `spanner:"ROLE_NAME"`
	IsSystem bool   `spanner:"IS_SYSTEM"`
}

// RoleChangeStreamGrant represents INFORMATION_SCHEMA.ROLE_CHANGE_STREAM_GRANTS (real only).
type RoleChangeStreamGrant struct {
	ChangeStreamCatalog string `spanner:"CHANGE_STREAM_CATALOG"`
	ChangeStreamSchema  string `spanner:"CHANGE_STREAM_SCHEMA"`
	ChangeStreamName    string `spanner:"CHANGE_STREAM_NAME"`
	PrivilegeType       string `spanner:"PRIVILEGE_TYPE"`
	Grantee             string `spanner:"GRANTEE"`
}

// RoleColumnGrant represents INFORMATION_SCHEMA.ROLE_COLUMN_GRANTS (real only).
type RoleColumnGrant struct {
	TableCatalog  string `spanner:"TABLE_CATALOG"`
	TableSchema   string `spanner:"TABLE_SCHEMA"`
	TableName     string `spanner:"TABLE_NAME"`
	ColumnName    string `spanner:"COLUMN_NAME"`
	PrivilegeType string `spanner:"PRIVILEGE_TYPE"`
	Grantee       string `spanner:"GRANTEE"`
}

// RoleGrantee represents INFORMATION_SCHEMA.ROLE_GRANTEES (real only).
type RoleGrantee struct {
	RoleName string `spanner:"ROLE_NAME"`
	Grantee  string `spanner:"GRANTEE"`
}

// RoleModelGrant represents INFORMATION_SCHEMA.ROLE_MODEL_GRANTS (real only).
type RoleModelGrant struct {
	ModelCatalog  string `spanner:"MODEL_CATALOG"`
	ModelSchema   string `spanner:"MODEL_SCHEMA"`
	ModelName     string `spanner:"MODEL_NAME"`
	PrivilegeType string `spanner:"PRIVILEGE_TYPE"`
	Grantee       string `spanner:"GRANTEE"`
}

// RoleRoutineGrant represents INFORMATION_SCHEMA.ROLE_ROUTINE_GRANTS (real only).
type RoleRoutineGrant struct {
	SpecificCatalog string `spanner:"SPECIFIC_CATALOG"`
	SpecificSchema  string `spanner:"SPECIFIC_SCHEMA"`
	SpecificName    string `spanner:"SPECIFIC_NAME"`
	PrivilegeType   string `spanner:"PRIVILEGE_TYPE"`
	Grantee         string `spanner:"GRANTEE"`
}

// RoleTableGrant represents INFORMATION_SCHEMA.ROLE_TABLE_GRANTS (real only).
type RoleTableGrant struct {
	TableCatalog  string `spanner:"TABLE_CATALOG"`
	TableSchema   string `spanner:"TABLE_SCHEMA"`
	TableName     string `spanner:"TABLE_NAME"`
	PrivilegeType string `spanner:"PRIVILEGE_TYPE"`
	Grantee       string `spanner:"GRANTEE"`
}

// Routine represents INFORMATION_SCHEMA.ROUTINES (real only).
type Routine struct {
	SpecificCatalog   string  `spanner:"SPECIFIC_CATALOG"`
	SpecificSchema    string  `spanner:"SPECIFIC_SCHEMA"`
	SpecificName      string  `spanner:"SPECIFIC_NAME"`
	RoutineCatalog    string  `spanner:"ROUTINE_CATALOG"`
	RoutineSchema     string  `spanner:"ROUTINE_SCHEMA"`
	RoutineName       string  `spanner:"ROUTINE_NAME"`
	RoutineType       string  `spanner:"ROUTINE_TYPE"`
	DataType          *string `spanner:"DATA_TYPE"`
	SpannerType       *string `spanner:"SPANNER_TYPE"`
	RoutineBody       string  `spanner:"ROUTINE_BODY"`
	RoutineDefinition *string `spanner:"ROUTINE_DEFINITION"`
	SecurityType      string  `spanner:"SECURITY_TYPE"`
	// Language, Determinism, and Remote are retained for AST round-trips.
	// They are not columns in INFORMATION_SCHEMA.ROUTINES.
	Language    string `spanner:"-"`
	Determinism string `spanner:"-"`
	Remote      bool   `spanner:"-"`
}

// RoutineOption represents INFORMATION_SCHEMA.ROUTINE_OPTIONS (real only).
type RoutineOption struct {
	SpecificCatalog string `spanner:"SPECIFIC_CATALOG"`
	SpecificSchema  string `spanner:"SPECIFIC_SCHEMA"`
	SpecificName    string `spanner:"SPECIFIC_NAME"`
	OptionName      string `spanner:"OPTION_NAME"`
	OptionType      string `spanner:"OPTION_TYPE"`
	OptionValue     string `spanner:"OPTION_VALUE"`
}

// RoutinePrivilege represents INFORMATION_SCHEMA.ROUTINE_PRIVILEGES (real only).
type RoutinePrivilege struct {
	SpecificCatalog string `spanner:"SPECIFIC_CATALOG"`
	SpecificSchema  string `spanner:"SPECIFIC_SCHEMA"`
	SpecificName    string `spanner:"SPECIFIC_NAME"`
	PrivilegeType   string `spanner:"PRIVILEGE_TYPE"`
	Grantee         string `spanner:"GRANTEE"`
}

// Schema represents INFORMATION_SCHEMA.SCHEMATA (real superset).
// Emulator lacks PROTO_BUNDLE.
type Schema struct {
	CatalogName        string  `spanner:"CATALOG_NAME"`
	SchemaName         string  `spanner:"SCHEMA_NAME"`
	EffectiveTimestamp *int64  `spanner:"EFFECTIVE_TIMESTAMP"`
	ProtoBundle        []byte  `spanner:"PROTO_BUNDLE"` // real only, PROTO<proto2.FileDescriptorSet>
	SchemaOwner        *string `spanner:"SCHEMA_OWNER"`
}

// Sequence represents INFORMATION_SCHEMA.SEQUENCES.
type Sequence struct {
	Catalog  string `spanner:"CATALOG"`
	Schema   string `spanner:"SCHEMA"`
	Name     string `spanner:"NAME"`
	DataType string `spanner:"DATA_TYPE"`
}

// SequenceOption represents INFORMATION_SCHEMA.SEQUENCE_OPTIONS.
type SequenceOption struct {
	Catalog     string `spanner:"CATALOG"`
	Schema      string `spanner:"SCHEMA"`
	Name        string `spanner:"NAME"`
	OptionName  string `spanner:"OPTION_NAME"`
	OptionType  string `spanner:"OPTION_TYPE"`
	OptionValue string `spanner:"OPTION_VALUE"`
}

// SpannerStatistic represents INFORMATION_SCHEMA.SPANNER_STATISTICS.
type SpannerStatistic struct {
	CatalogName string `spanner:"CATALOG_NAME"`
	SchemaName  string `spanner:"SCHEMA_NAME"`
	PackageName string `spanner:"PACKAGE_NAME"`
	AllowGC     bool   `spanner:"ALLOW_GC"`
}

// Table represents INFORMATION_SCHEMA.TABLES.
type Table struct {
	TableCatalog                string  `spanner:"TABLE_CATALOG"`
	TableSchema                 string  `spanner:"TABLE_SCHEMA"`
	TableName                   string  `spanner:"TABLE_NAME"`
	ParentTableName             *string `spanner:"PARENT_TABLE_NAME"`
	OnDeleteAction              *string `spanner:"ON_DELETE_ACTION"`
	TableType                   string  `spanner:"TABLE_TYPE"`
	SpannerState                *string `spanner:"SPANNER_STATE"`
	InterleaveType              *string `spanner:"INTERLEAVE_TYPE"`
	RowDeletionPolicyExpression *string `spanner:"ROW_DELETION_POLICY_EXPRESSION"`
}

// TableConstraint represents INFORMATION_SCHEMA.TABLE_CONSTRAINTS.
type TableConstraint struct {
	ConstraintCatalog string `spanner:"CONSTRAINT_CATALOG"`
	ConstraintSchema  string `spanner:"CONSTRAINT_SCHEMA"`
	ConstraintName    string `spanner:"CONSTRAINT_NAME"`
	TableCatalog      string `spanner:"TABLE_CATALOG"`
	TableSchema       string `spanner:"TABLE_SCHEMA"`
	TableName         string `spanner:"TABLE_NAME"`
	ConstraintType    string `spanner:"CONSTRAINT_TYPE"`
	IsDeferrable      string `spanner:"IS_DEFERRABLE"`
	InitiallyDeferred string `spanner:"INITIALLY_DEFERRED"`
	Enforced          string `spanner:"ENFORCED"`
}

// TablePrivilege represents INFORMATION_SCHEMA.TABLE_PRIVILEGES (real only).
type TablePrivilege struct {
	TableCatalog  string `spanner:"TABLE_CATALOG"`
	TableSchema   string `spanner:"TABLE_SCHEMA"`
	TableName     string `spanner:"TABLE_NAME"`
	PrivilegeType string `spanner:"PRIVILEGE_TYPE"`
	Grantee       string `spanner:"GRANTEE"`
}

// TableSynonym represents INFORMATION_SCHEMA.TABLE_SYNONYMS (real only).
type TableSynonym struct {
	SynonymCatalog string `spanner:"SYNONYM_CATALOG"`
	SynonymSchema  string `spanner:"SYNONYM_SCHEMA"`
	SynonymName    string `spanner:"SYNONYM_NAME"`
	TableCatalog   string `spanner:"TABLE_CATALOG"`
	TableSchema    string `spanner:"TABLE_SCHEMA"`
	TableName      string `spanner:"TABLE_NAME"`
}

// View represents INFORMATION_SCHEMA.VIEWS.
type View struct {
	TableCatalog   string `spanner:"TABLE_CATALOG"`
	TableSchema    string `spanner:"TABLE_SCHEMA"`
	TableName      string `spanner:"TABLE_NAME"`
	ViewDefinition string `spanner:"VIEW_DEFINITION"`
	SecurityType   string `spanner:"SECURITY_TYPE"`
}
