package infoschem

// TableOption represents a row in INFORMATION_SCHEMA.TABLE_OPTIONS.
// Table-level options (e.g. locality_group) from CREATE TABLE ... OPTIONS(...)
// are exposed here on real Spanner; the emulator does not yet expose this table.
type TableOption struct {
	TableCatalog string `spanner:"TABLE_CATALOG"`
	TableSchema  string `spanner:"TABLE_SCHEMA"`
	TableName    string `spanner:"TABLE_NAME"`
	OptionName   string `spanner:"OPTION_NAME"`
	OptionType   string `spanner:"OPTION_TYPE"`
	OptionValue  string `spanner:"OPTION_VALUE"`
}
