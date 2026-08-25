package infoschem

import (
	"fmt"
	"strings"
)

// ColumnMeta describes a single column in an INFORMATION_SCHEMA table.
type ColumnMeta struct {
	Name            string
	SpannerType     string
	OrdinalPosition int
	// Rolling marks columns whose INFORMATION_SCHEMA.COLUMNS advertisement can
	// precede query support during a managed-service rollout. Discovery probes
	// these columns before including them in loader queries.
	Rolling bool
}

// TableMeta describes an INFORMATION_SCHEMA or SPANNER_SYS table.
type TableMeta struct {
	Schema  string // "INFORMATION_SCHEMA" or "SPANNER_SYS"
	Name    string // e.g. "COLUMNS"
	Columns []ColumnMeta
}

// DiscoveredColumns represents a set of available tables and columns in the database.
// Keys are table names, values are sets of column names.
type DiscoveredColumns map[string]map[string]bool

// Query generates a SELECT statement for this table targeting the given environment.
// Only columns that are present in the discovered columns are included.
func (m *TableMeta) Query(discovered DiscoveredColumns) (string, error) {
	availCols, tableExists := discovered[m.Name]
	if !tableExists {
		return "", fmt.Errorf("table %s.%s is not available", m.Schema, m.Name)
	}

	var names []string
	for _, c := range m.Columns {
		if availCols[c.Name] {
			names = append(names, "`"+c.Name+"`")
		}
	}
	if len(names) == 0 {
		return "", fmt.Errorf("table %s.%s has no columns available", m.Schema, m.Name)
	}
	return fmt.Sprintf(
		"SELECT %s FROM %s.%s",
		strings.Join(names, ", "),
		m.Schema,
		m.Name,
	), nil
}

// AllTableMetas returns metadata for all 48 INFORMATION_SCHEMA tables.
func AllTableMetas() []*TableMeta {
	metas := make([]*TableMeta, len(informationSchemaTables))
	for i, meta := range informationSchemaTables {
		metas[i] = cloneTableMeta(meta)
	}
	return metas
}

// TableMetaByName returns metadata for a specific INFORMATION_SCHEMA table.
func TableMetaByName(name string) (*TableMeta, bool) {
	for _, m := range informationSchemaTables {
		if m.Name == name {
			return cloneTableMeta(m), true
		}
	}
	return nil, false
}

func cloneTableMeta(meta *TableMeta) *TableMeta {
	return &TableMeta{
		Schema:  meta.Schema,
		Name:    meta.Name,
		Columns: append([]ColumnMeta{}, meta.Columns...),
	}
}

// col is a helper for concise ColumnMeta construction.
func col(name, typ string, ordinal int) ColumnMeta {
	return ColumnMeta{Name: name, SpannerType: typ, OrdinalPosition: ordinal}
}

func rollingCol(name, typ string, ordinal int) ColumnMeta {
	return ColumnMeta{Name: name, SpannerType: typ, OrdinalPosition: ordinal, Rolling: true}
}

var informationSchemaTables = []*TableMeta{
	{
		Schema: "INFORMATION_SCHEMA", Name: "CHANGE_STREAMS",
		Columns: []ColumnMeta{
			col("CHANGE_STREAM_CATALOG", "STRING(MAX)", 1),
			col("CHANGE_STREAM_SCHEMA", "STRING(MAX)", 2),
			col("CHANGE_STREAM_NAME", "STRING(MAX)", 3),
			col("ALL", "BOOL", 4),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "CHANGE_STREAM_COLUMNS",
		Columns: []ColumnMeta{
			col("CHANGE_STREAM_CATALOG", "STRING(MAX)", 1),
			col("CHANGE_STREAM_SCHEMA", "STRING(MAX)", 2),
			col("CHANGE_STREAM_NAME", "STRING(MAX)", 3),
			col("TABLE_CATALOG", "STRING(MAX)", 4),
			col("TABLE_SCHEMA", "STRING(MAX)", 5),
			col("TABLE_NAME", "STRING(MAX)", 6),
			col("COLUMN_NAME", "STRING(MAX)", 7),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "CHANGE_STREAM_OPTIONS",
		Columns: []ColumnMeta{
			col("CHANGE_STREAM_CATALOG", "STRING(MAX)", 1),
			col("CHANGE_STREAM_SCHEMA", "STRING(MAX)", 2),
			col("CHANGE_STREAM_NAME", "STRING(MAX)", 3),
			col("OPTION_NAME", "STRING(MAX)", 4),
			col("OPTION_TYPE", "STRING(MAX)", 5),
			col("OPTION_VALUE", "STRING(MAX)", 6),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "CHANGE_STREAM_PRIVILEGES",
		Columns: []ColumnMeta{
			col("CHANGE_STREAM_CATALOG", "STRING(MAX)", 1),
			col("CHANGE_STREAM_SCHEMA", "STRING(MAX)", 2),
			col("CHANGE_STREAM_NAME", "STRING(MAX)", 3),
			col("PRIVILEGE_TYPE", "STRING(MAX)", 4),
			col("GRANTEE", "STRING(MAX)", 5),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "CHANGE_STREAM_TABLES",
		Columns: []ColumnMeta{
			col("CHANGE_STREAM_CATALOG", "STRING(MAX)", 1),
			col("CHANGE_STREAM_SCHEMA", "STRING(MAX)", 2),
			col("CHANGE_STREAM_NAME", "STRING(MAX)", 3),
			col("TABLE_CATALOG", "STRING(MAX)", 4),
			col("TABLE_SCHEMA", "STRING(MAX)", 5),
			col("TABLE_NAME", "STRING(MAX)", 6),
			col("ALL_COLUMNS", "BOOL", 7),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "CHECK_CONSTRAINTS",
		Columns: []ColumnMeta{
			col("CONSTRAINT_CATALOG", "STRING(MAX)", 1),
			col("CONSTRAINT_SCHEMA", "STRING(MAX)", 2),
			col("CONSTRAINT_NAME", "STRING(MAX)", 3),
			col("CHECK_CLAUSE", "STRING(MAX)", 4),
			col("SPANNER_STATE", "STRING(MAX)", 5),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "COLUMNS",
		Columns: []ColumnMeta{
			col("TABLE_CATALOG", "STRING(MAX)", 1),
			col("TABLE_SCHEMA", "STRING(MAX)", 2),
			col("TABLE_NAME", "STRING(MAX)", 3),
			col("COLUMN_NAME", "STRING(MAX)", 4),
			col("ORDINAL_POSITION", "INT64", 5),
			col("COLUMN_DEFAULT", "STRING(MAX)", 6),
			col("DATA_TYPE", "STRING(MAX)", 7),
			col("IS_NULLABLE", "STRING(MAX)", 8),
			col("SPANNER_TYPE", "STRING(MAX)", 9),
			col("IS_GENERATED", "STRING(MAX)", 10),
			col("GENERATION_EXPRESSION", "STRING(MAX)", 11),
			col("IS_STORED", "STRING(MAX)", 12),
			col("IS_HIDDEN", "BOOL", 13),
			col("SPANNER_STATE", "STRING(MAX)", 14),
			col("IS_IDENTITY", "STRING(MAX)", 15),
			col("IDENTITY_GENERATION", "STRING(MAX)", 16),
			col("IDENTITY_KIND", "STRING(MAX)", 17),
			col("IDENTITY_START_WITH_COUNTER", "STRING(MAX)", 18),
			col("IDENTITY_SKIP_RANGE_MIN", "STRING(MAX)", 19),
			col("IDENTITY_SKIP_RANGE_MAX", "STRING(MAX)", 20),
			col("ON_UPDATE_EXPRESSION", "STRING(MAX)", 21),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "COLUMN_COLUMN_USAGE",
		Columns: []ColumnMeta{
			col("TABLE_CATALOG", "STRING(MAX)", 1),
			col("TABLE_SCHEMA", "STRING(MAX)", 2),
			col("TABLE_NAME", "STRING(MAX)", 3),
			col("COLUMN_NAME", "STRING(MAX)", 4),
			col("DEPENDENT_COLUMN", "STRING(MAX)", 5),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "COLUMN_OPTIONS",
		Columns: []ColumnMeta{
			col("TABLE_CATALOG", "STRING(MAX)", 1),
			col("TABLE_SCHEMA", "STRING(MAX)", 2),
			col("TABLE_NAME", "STRING(MAX)", 3),
			col("COLUMN_NAME", "STRING(MAX)", 4),
			col("OPTION_NAME", "STRING(MAX)", 5),
			col("OPTION_TYPE", "STRING(MAX)", 6),
			col("OPTION_VALUE", "STRING(MAX)", 7),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "COLUMN_PRIVILEGES",
		Columns: []ColumnMeta{
			col("TABLE_CATALOG", "STRING(MAX)", 1),
			col("TABLE_SCHEMA", "STRING(MAX)", 2),
			col("TABLE_NAME", "STRING(MAX)", 3),
			col("COLUMN_NAME", "STRING(MAX)", 4),
			col("PRIVILEGE_TYPE", "STRING(MAX)", 5),
			col("GRANTEE", "STRING(MAX)", 6),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "CONSTRAINT_COLUMN_USAGE",
		Columns: []ColumnMeta{
			col("TABLE_CATALOG", "STRING(MAX)", 1),
			col("TABLE_SCHEMA", "STRING(MAX)", 2),
			col("TABLE_NAME", "STRING(MAX)", 3),
			col("COLUMN_NAME", "STRING(MAX)", 4),
			col("CONSTRAINT_CATALOG", "STRING(MAX)", 5),
			col("CONSTRAINT_SCHEMA", "STRING(MAX)", 6),
			col("CONSTRAINT_NAME", "STRING(MAX)", 7),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "CONSTRAINT_TABLE_USAGE",
		Columns: []ColumnMeta{
			col("TABLE_CATALOG", "STRING(MAX)", 1),
			col("TABLE_SCHEMA", "STRING(MAX)", 2),
			col("TABLE_NAME", "STRING(MAX)", 3),
			col("CONSTRAINT_CATALOG", "STRING(MAX)", 4),
			col("CONSTRAINT_SCHEMA", "STRING(MAX)", 5),
			col("CONSTRAINT_NAME", "STRING(MAX)", 6),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "DATABASE_OPTIONS",
		Columns: []ColumnMeta{
			col("CATALOG_NAME", "STRING(MAX)", 1),
			col("SCHEMA_NAME", "STRING(MAX)", 2),
			col("OPTION_NAME", "STRING(MAX)", 3),
			col("OPTION_TYPE", "STRING(MAX)", 4),
			col("OPTION_VALUE", "STRING(MAX)", 5),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "INDEXES",
		Columns: []ColumnMeta{
			col("TABLE_CATALOG", "STRING(MAX)", 1),
			col("TABLE_SCHEMA", "STRING(MAX)", 2),
			col("TABLE_NAME", "STRING(MAX)", 3),
			col("INDEX_NAME", "STRING(MAX)", 4),
			col("INDEX_TYPE", "STRING(MAX)", 5),
			col("PARENT_TABLE_NAME", "STRING(MAX)", 6),
			col("IS_UNIQUE", "BOOL", 7),
			col("IS_NULL_FILTERED", "BOOL", 8),
			col("INDEX_STATE", "STRING(100)", 9),
			col("FILTER", "STRING(MAX)", 10),
			col("SPANNER_IS_MANAGED", "BOOL", 11),
			col("SEARCH_PARTITION_BY", "ARRAY<STRING(MAX)>", 12),
			col("SEARCH_ORDER_BY", "ARRAY<STRING(MAX)>", 13),
			rollingCol("SEARCH_UNNEST", "ARRAY<STRING(MAX)>", 14),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "INDEX_COLUMNS",
		Columns: []ColumnMeta{
			col("TABLE_CATALOG", "STRING(MAX)", 1),
			col("TABLE_SCHEMA", "STRING(MAX)", 2),
			col("TABLE_NAME", "STRING(MAX)", 3),
			col("INDEX_NAME", "STRING(MAX)", 4),
			col("INDEX_TYPE", "STRING(MAX)", 5),
			col("COLUMN_NAME", "STRING(MAX)", 6),
			col("ORDINAL_POSITION", "INT64", 7),
			col("COLUMN_ORDERING", "STRING(MAX)", 8),
			col("IS_NULLABLE", "STRING(MAX)", 9),
			col("SPANNER_TYPE", "STRING(MAX)", 10),
			rollingCol("EXPRESSION", "STRING(MAX)", 11),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "INDEX_OPTIONS",
		Columns: []ColumnMeta{
			col("TABLE_CATALOG", "STRING(MAX)", 1),
			col("TABLE_SCHEMA", "STRING(MAX)", 2),
			col("TABLE_NAME", "STRING(MAX)", 3),
			col("INDEX_NAME", "STRING(MAX)", 4),
			col("INDEX_TYPE", "STRING(MAX)", 5),
			col("OPTION_NAME", "STRING(MAX)", 6),
			col("OPTION_TYPE", "STRING(MAX)", 7),
			col("OPTION_VALUE", "STRING(MAX)", 8),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "KEY_COLUMN_USAGE",
		Columns: []ColumnMeta{
			col("CONSTRAINT_CATALOG", "STRING(MAX)", 1),
			col("CONSTRAINT_SCHEMA", "STRING(MAX)", 2),
			col("CONSTRAINT_NAME", "STRING(MAX)", 3),
			col("TABLE_CATALOG", "STRING(MAX)", 4),
			col("TABLE_SCHEMA", "STRING(MAX)", 5),
			col("TABLE_NAME", "STRING(MAX)", 6),
			col("COLUMN_NAME", "STRING(MAX)", 7),
			col("ORDINAL_POSITION", "INT64", 8),
			col("POSITION_IN_UNIQUE_CONSTRAINT", "INT64", 9),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "LOCALITY_GROUP_OPTIONS",
		Columns: []ColumnMeta{
			col("LOCALITY_GROUP_NAME", "STRING(MAX)", 1),
			col("OPTION_NAME", "STRING(MAX)", 2),
			col("OPTION_VALUE", "STRING(MAX)", 3),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "MODELS",
		Columns: []ColumnMeta{
			col("MODEL_CATALOG", "STRING(MAX)", 1),
			col("MODEL_SCHEMA", "STRING(MAX)", 2),
			col("MODEL_NAME", "STRING(MAX)", 3),
			col("IS_REMOTE", "BOOL", 4),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "MODEL_COLUMNS",
		Columns: []ColumnMeta{
			col("MODEL_CATALOG", "STRING(MAX)", 1),
			col("MODEL_SCHEMA", "STRING(MAX)", 2),
			col("MODEL_NAME", "STRING(MAX)", 3),
			col("COLUMN_KIND", "STRING(MAX)", 4),
			col("COLUMN_NAME", "STRING(MAX)", 5),
			col("ORDINAL_POSITION", "INT64", 6),
			col("DATA_TYPE", "STRING(MAX)", 7),
			col("IS_EXPLICIT", "BOOL", 8),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "MODEL_COLUMN_OPTIONS",
		Columns: []ColumnMeta{
			col("MODEL_CATALOG", "STRING(MAX)", 1),
			col("MODEL_SCHEMA", "STRING(MAX)", 2),
			col("MODEL_NAME", "STRING(MAX)", 3),
			col("COLUMN_KIND", "STRING(MAX)", 4),
			col("COLUMN_NAME", "STRING(MAX)", 5),
			col("OPTION_NAME", "STRING(MAX)", 6),
			col("OPTION_TYPE", "STRING(MAX)", 7),
			col("OPTION_VALUE", "STRING(MAX)", 8),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "MODEL_OPTIONS",
		Columns: []ColumnMeta{
			col("MODEL_CATALOG", "STRING(MAX)", 1),
			col("MODEL_SCHEMA", "STRING(MAX)", 2),
			col("MODEL_NAME", "STRING(MAX)", 3),
			col("OPTION_NAME", "STRING(MAX)", 4),
			col("OPTION_TYPE", "STRING(MAX)", 5),
			col("OPTION_VALUE", "STRING(MAX)", 6),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "MODEL_PRIVILEGES",
		Columns: []ColumnMeta{
			col("MODEL_CATALOG", "STRING(MAX)", 1),
			col("MODEL_SCHEMA", "STRING(MAX)", 2),
			col("MODEL_NAME", "STRING(MAX)", 3),
			col("PRIVILEGE_TYPE", "STRING(MAX)", 4),
			col("GRANTEE", "STRING(MAX)", 5),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "PARAMETERS",
		Columns: []ColumnMeta{
			col("SPECIFIC_CATALOG", "STRING(MAX)", 1),
			col("SPECIFIC_SCHEMA", "STRING(MAX)", 2),
			col("SPECIFIC_NAME", "STRING(MAX)", 3),
			col("ORDINAL_POSITION", "INT64", 4),
			col("PARAMETER_NAME", "STRING(MAX)", 5),
			col("DATA_TYPE", "STRING(MAX)", 6),
			col("PARAMETER_DEFAULT", "STRING(MAX)", 7),
			col("SPANNER_TYPE", "STRING(MAX)", 8),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "PLACEMENTS",
		Columns: []ColumnMeta{
			col("PLACEMENT_NAME", "STRING(MAX)", 1),
			col("IS_DEFAULT", "BOOL", 2),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "PLACEMENT_OPTIONS",
		Columns: []ColumnMeta{
			col("PLACEMENT_NAME", "STRING(MAX)", 1),
			col("OPTION_NAME", "STRING(MAX)", 2),
			col("OPTION_TYPE", "STRING(MAX)", 3),
			col("OPTION_VALUE", "STRING(MAX)", 4),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "PROPERTY_GRAPHS",
		Columns: []ColumnMeta{
			col("PROPERTY_GRAPH_CATALOG", "STRING(MAX)", 1),
			col("PROPERTY_GRAPH_SCHEMA", "STRING(MAX)", 2),
			col("PROPERTY_GRAPH_NAME", "STRING(MAX)", 3),
			col("PROPERTY_GRAPH_METADATA_JSON", "JSON", 4),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "REFERENTIAL_CONSTRAINTS",
		Columns: []ColumnMeta{
			col("CONSTRAINT_CATALOG", "STRING(MAX)", 1),
			col("CONSTRAINT_SCHEMA", "STRING(MAX)", 2),
			col("CONSTRAINT_NAME", "STRING(MAX)", 3),
			col("UNIQUE_CONSTRAINT_CATALOG", "STRING(MAX)", 4),
			col("UNIQUE_CONSTRAINT_SCHEMA", "STRING(MAX)", 5),
			col("UNIQUE_CONSTRAINT_NAME", "STRING(MAX)", 6),
			col("MATCH_OPTION", "STRING(MAX)", 7),
			col("UPDATE_RULE", "STRING(MAX)", 8),
			col("DELETE_RULE", "STRING(MAX)", 9),
			col("SPANNER_STATE", "STRING(MAX)", 10),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "ROLES",
		Columns: []ColumnMeta{
			col("ROLE_NAME", "STRING(MAX)", 1),
			col("IS_SYSTEM", "BOOL", 2),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "ROLE_CHANGE_STREAM_GRANTS",
		Columns: []ColumnMeta{
			col("CHANGE_STREAM_CATALOG", "STRING(MAX)", 1),
			col("CHANGE_STREAM_SCHEMA", "STRING(MAX)", 2),
			col("CHANGE_STREAM_NAME", "STRING(MAX)", 3),
			col("PRIVILEGE_TYPE", "STRING(MAX)", 4),
			col("GRANTEE", "STRING(MAX)", 5),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "ROLE_COLUMN_GRANTS",
		Columns: []ColumnMeta{
			col("TABLE_CATALOG", "STRING(MAX)", 1),
			col("TABLE_SCHEMA", "STRING(MAX)", 2),
			col("TABLE_NAME", "STRING(MAX)", 3),
			col("COLUMN_NAME", "STRING(MAX)", 4),
			col("PRIVILEGE_TYPE", "STRING(MAX)", 5),
			col("GRANTEE", "STRING(MAX)", 6),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "ROLE_GRANTEES",
		Columns: []ColumnMeta{
			col("ROLE_NAME", "STRING(MAX)", 1),
			col("GRANTEE", "STRING(MAX)", 2),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "ROLE_MODEL_GRANTS",
		Columns: []ColumnMeta{
			col("MODEL_CATALOG", "STRING(MAX)", 1),
			col("MODEL_SCHEMA", "STRING(MAX)", 2),
			col("MODEL_NAME", "STRING(MAX)", 3),
			col("PRIVILEGE_TYPE", "STRING(MAX)", 4),
			col("GRANTEE", "STRING(MAX)", 5),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "ROLE_ROUTINE_GRANTS",
		Columns: []ColumnMeta{
			col("SPECIFIC_CATALOG", "STRING(MAX)", 1),
			col("SPECIFIC_SCHEMA", "STRING(MAX)", 2),
			col("SPECIFIC_NAME", "STRING(MAX)", 3),
			col("PRIVILEGE_TYPE", "STRING(MAX)", 4),
			col("GRANTEE", "STRING(MAX)", 5),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "ROLE_TABLE_GRANTS",
		Columns: []ColumnMeta{
			col("TABLE_CATALOG", "STRING(MAX)", 1),
			col("TABLE_SCHEMA", "STRING(MAX)", 2),
			col("TABLE_NAME", "STRING(MAX)", 3),
			col("PRIVILEGE_TYPE", "STRING(MAX)", 4),
			col("GRANTEE", "STRING(MAX)", 5),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "ROUTINES",
		Columns: []ColumnMeta{
			col("SPECIFIC_CATALOG", "STRING(MAX)", 1),
			col("SPECIFIC_SCHEMA", "STRING(MAX)", 2),
			col("SPECIFIC_NAME", "STRING(MAX)", 3),
			col("ROUTINE_CATALOG", "STRING(MAX)", 4),
			col("ROUTINE_SCHEMA", "STRING(MAX)", 5),
			col("ROUTINE_NAME", "STRING(MAX)", 6),
			col("ROUTINE_TYPE", "STRING(MAX)", 7),
			col("DATA_TYPE", "STRING(MAX)", 8),
			col("ROUTINE_BODY", "STRING(MAX)", 9),
			col("ROUTINE_DEFINITION", "STRING(MAX)", 10),
			col("SECURITY_TYPE", "STRING(MAX)", 11),
			col("SPANNER_TYPE", "STRING(MAX)", 12),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "ROUTINE_OPTIONS",
		Columns: []ColumnMeta{
			col("SPECIFIC_CATALOG", "STRING(MAX)", 1),
			col("SPECIFIC_SCHEMA", "STRING(MAX)", 2),
			col("SPECIFIC_NAME", "STRING(MAX)", 3),
			col("OPTION_NAME", "STRING(MAX)", 4),
			col("OPTION_TYPE", "STRING(MAX)", 5),
			col("OPTION_VALUE", "STRING(MAX)", 6),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "ROUTINE_PRIVILEGES",
		Columns: []ColumnMeta{
			col("SPECIFIC_CATALOG", "STRING(MAX)", 1),
			col("SPECIFIC_SCHEMA", "STRING(MAX)", 2),
			col("SPECIFIC_NAME", "STRING(MAX)", 3),
			col("PRIVILEGE_TYPE", "STRING(MAX)", 4),
			col("GRANTEE", "STRING(MAX)", 5),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "SCHEMATA",
		Columns: []ColumnMeta{
			col("CATALOG_NAME", "STRING(MAX)", 1),
			col("SCHEMA_NAME", "STRING(MAX)", 2),
			col("EFFECTIVE_TIMESTAMP", "INT64", 3),
			col("PROTO_BUNDLE", "PROTO<proto2.FileDescriptorSet>", 4),
			col("SCHEMA_OWNER", "STRING(MAX)", 5),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "SEQUENCES",
		Columns: []ColumnMeta{
			col("CATALOG", "STRING(MAX)", 1),
			col("SCHEMA", "STRING(MAX)", 2),
			col("NAME", "STRING(MAX)", 3),
			col("DATA_TYPE", "STRING(MAX)", 4),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "SEQUENCE_OPTIONS",
		Columns: []ColumnMeta{
			col("CATALOG", "STRING(MAX)", 1),
			col("SCHEMA", "STRING(MAX)", 2),
			col("NAME", "STRING(MAX)", 3),
			col("OPTION_NAME", "STRING(MAX)", 4),
			col("OPTION_TYPE", "STRING(MAX)", 5),
			col("OPTION_VALUE", "STRING(MAX)", 6),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "SPANNER_STATISTICS",
		Columns: []ColumnMeta{
			col("CATALOG_NAME", "STRING(MAX)", 1),
			col("SCHEMA_NAME", "STRING(MAX)", 2),
			col("PACKAGE_NAME", "STRING(MAX)", 3),
			col("ALLOW_GC", "BOOL", 4),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "TABLES",
		Columns: []ColumnMeta{
			col("TABLE_CATALOG", "STRING(MAX)", 1),
			col("TABLE_SCHEMA", "STRING(MAX)", 2),
			col("TABLE_NAME", "STRING(MAX)", 3),
			col("PARENT_TABLE_NAME", "STRING(MAX)", 4),
			col("ON_DELETE_ACTION", "STRING(MAX)", 5),
			col("TABLE_TYPE", "STRING(32)", 6),
			col("SPANNER_STATE", "STRING(MAX)", 7),
			col("INTERLEAVE_TYPE", "STRING(MAX)", 8),
			col("ROW_DELETION_POLICY_EXPRESSION", "STRING(MAX)", 9),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "TABLE_OPTIONS",
		Columns: []ColumnMeta{
			col("TABLE_CATALOG", "STRING(MAX)", 1),
			col("TABLE_SCHEMA", "STRING(MAX)", 2),
			col("TABLE_NAME", "STRING(MAX)", 3),
			col("OPTION_NAME", "STRING(MAX)", 4),
			col("OPTION_TYPE", "STRING(MAX)", 5),
			col("OPTION_VALUE", "STRING(MAX)", 6),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "TABLE_CONSTRAINTS",
		Columns: []ColumnMeta{
			col("CONSTRAINT_CATALOG", "STRING(MAX)", 1),
			col("CONSTRAINT_SCHEMA", "STRING(MAX)", 2),
			col("CONSTRAINT_NAME", "STRING(MAX)", 3),
			col("TABLE_CATALOG", "STRING(MAX)", 4),
			col("TABLE_SCHEMA", "STRING(MAX)", 5),
			col("TABLE_NAME", "STRING(MAX)", 6),
			col("CONSTRAINT_TYPE", "STRING(MAX)", 7),
			col("IS_DEFERRABLE", "STRING(MAX)", 8),
			col("INITIALLY_DEFERRED", "STRING(MAX)", 9),
			col("ENFORCED", "STRING(MAX)", 10),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "TABLE_PRIVILEGES",
		Columns: []ColumnMeta{
			col("TABLE_CATALOG", "STRING(MAX)", 1),
			col("TABLE_SCHEMA", "STRING(MAX)", 2),
			col("TABLE_NAME", "STRING(MAX)", 3),
			col("PRIVILEGE_TYPE", "STRING(MAX)", 4),
			col("GRANTEE", "STRING(MAX)", 5),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "TABLE_SYNONYMS",
		Columns: []ColumnMeta{
			col("SYNONYM_CATALOG", "STRING(MAX)", 1),
			col("SYNONYM_SCHEMA", "STRING(MAX)", 2),
			col("SYNONYM_NAME", "STRING(MAX)", 3),
			col("TABLE_CATALOG", "STRING(MAX)", 4),
			col("TABLE_SCHEMA", "STRING(MAX)", 5),
			col("TABLE_NAME", "STRING(MAX)", 6),
		},
	},
	{
		Schema: "INFORMATION_SCHEMA", Name: "VIEWS",
		Columns: []ColumnMeta{
			col("TABLE_CATALOG", "STRING(MAX)", 1),
			col("TABLE_SCHEMA", "STRING(MAX)", 2),
			col("TABLE_NAME", "STRING(MAX)", 3),
			col("VIEW_DEFINITION", "STRING(MAX)", 4),
			col("SECURITY_TYPE", "STRING(MAX)", 5),
		},
	},
}
