package astconv

import (
	"testing"

	"github.com/apstndb/spanner-emulator-survey/infoschem"
	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
	"github.com/cloudspannerecosystem/memefish/token"
)

func TestRoundtrip_SimpleTable(t *testing.T) {
	ddl := `CREATE TABLE Users (
  UserId INT64 NOT NULL,
  Name STRING(MAX),
  Email STRING(255) NOT NULL,
  Age INT64,
  CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
) PRIMARY KEY (UserId)`

	stmt, err := memefish.ParseDDL("", ddl)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}

	schema, err := FromDDLStatements([]ast.DDL{stmt})
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if len(schema.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(schema.Tables))
	}
	if schema.Tables[0].TableName != "Users" {
		t.Errorf("table name = %q, want %q", schema.Tables[0].TableName, "Users")
	}
	if len(schema.Columns) != 5 {
		t.Fatalf("expected 5 columns, got %d", len(schema.Columns))
	}

	// Re-convert to DDL
	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	// Should produce 1 CREATE TABLE
	var createTables int
	for _, d := range ddls {
		if _, ok := d.(*ast.CreateTable); ok {
			createTables++
		}
	}
	if createTables != 1 {
		t.Errorf("expected 1 CreateTable, got %d", createTables)
	}
}

func TestRoundtrip_InterleaveTable(t *testing.T) {
	ddls := []string{
		`CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  Name STRING(MAX),
) PRIMARY KEY (SingerId)`,
		`CREATE TABLE Albums (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  Title STRING(MAX),
) PRIMARY KEY (SingerId, AlbumId),
  INTERLEAVE IN PARENT Singers ON DELETE CASCADE`,
	}

	var stmts []ast.DDL
	for _, d := range ddls {
		stmt, err := memefish.ParseDDL("", d)
		if err != nil {
			t.Fatalf("ParseDDL: %v", err)
		}
		stmts = append(stmts, stmt)
	}

	schema, err := FromDDLStatements(stmts)
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if len(schema.Tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(schema.Tables))
	}

	// Albums should have parent
	var albums *ast.CreateTable
	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}
	for _, d := range reconDDLs {
		if ct, ok := d.(*ast.CreateTable); ok && ct.Name.Idents[0].Name == "Albums" {
			albums = ct
		}
	}
	if albums == nil {
		t.Fatal("Albums table not found in reconstructed DDL")
		return
	}
	if albums.Cluster == nil {
		t.Error("Albums should have Cluster (INTERLEAVE)")
	} else {
		parentName := leafName(albums.Cluster.TableName)
		if parentName != "Singers" {
			t.Errorf("parent = %q, want %q", parentName, "Singers")
		}
		if albums.Cluster.OnDelete != ast.OnDeleteCascade {
			t.Errorf("OnDelete = %q, want CASCADE", albums.Cluster.OnDelete)
		}
	}
}

func TestRoundtrip_Index(t *testing.T) {
	ddls := []string{
		`CREATE TABLE Users (
  UserId INT64 NOT NULL,
  Email STRING(MAX),
) PRIMARY KEY (UserId)`,
		`CREATE UNIQUE NULL_FILTERED INDEX UsersByEmail ON Users(Email DESC)`,
	}

	var stmts []ast.DDL
	for _, d := range ddls {
		stmt, err := memefish.ParseDDL("", d)
		if err != nil {
			t.Fatalf("ParseDDL: %v", err)
		}
		stmts = append(stmts, stmt)
	}

	schema, err := FromDDLStatements(stmts)
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	// Should have 2 indexes: PRIMARY_KEY + UsersByEmail
	if len(schema.Indexes) != 2 {
		t.Fatalf("expected 2 indexes, got %d", len(schema.Indexes))
	}

	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var foundIndex bool
	for _, d := range reconDDLs {
		if ci, ok := d.(*ast.CreateIndex); ok {
			foundIndex = true
			if ci.Name.Idents[0].Name != "UsersByEmail" {
				t.Errorf("index name = %q, want %q", ci.Name.Idents[0].Name, "UsersByEmail")
			}
			if !ci.Unique {
				t.Error("index should be UNIQUE")
			}
			if !ci.NullFiltered {
				t.Error("index should be NULL_FILTERED")
			}
		}
	}
	if !foundIndex {
		t.Error("UsersByEmail index not found in reconstructed DDL")
	}
}

func TestRoundtrip_SearchIndexFilterAndOrder(t *testing.T) {
	ddls := []string{
		`CREATE TABLE Docs (
  DocId INT64 NOT NULL,
  Body STRING(MAX),
  BodyTokens TOKENLIST AS (TOKENIZE_FULLTEXT(Body)) HIDDEN,
  TenantId STRING(36),
  Rank INT64,
) PRIMARY KEY (DocId)`,
		`CREATE SEARCH INDEX DocsSearch ON Docs(BodyTokens)
  STORING (TenantId, Rank)
  PARTITION BY TenantId
  ORDER BY Rank DESC
  WHERE TenantId IS NOT NULL AND Rank IS NOT NULL`,
	}

	var stmts []ast.DDL
	for _, d := range ddls {
		stmt, err := memefish.ParseDDL("", d)
		if err != nil {
			t.Fatalf("ParseDDL: %v", err)
		}
		stmts = append(stmts, stmt)
	}

	schema, err := FromDDLStatements(stmts)
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	var searchIdx *infoschem.Index
	for _, idx := range schema.Indexes {
		if idx.IndexName == "DocsSearch" {
			searchIdx = idx
			break
		}
	}
	if searchIdx == nil {
		t.Fatal("DocsSearch index not found in schema")
	}
	if searchIdx.IndexType != "SEARCH" {
		t.Errorf("IndexType = %q, want live INFORMATION_SCHEMA value SEARCH", searchIdx.IndexType)
	}
	if got := searchIdx.Filter; got == nil || *got != "TenantId IS NOT NULL AND Rank IS NOT NULL" {
		t.Fatalf("Filter = %v, want %q", got, "TenantId IS NOT NULL AND Rank IS NOT NULL")
	}
	if len(searchIdx.SearchPartitionBy) != 1 || searchIdx.SearchPartitionBy[0] != "TenantId" {
		t.Fatalf("SearchPartitionBy = %v, want [TenantId]", searchIdx.SearchPartitionBy)
	}
	if len(searchIdx.SearchOrderBy) != 1 || searchIdx.SearchOrderBy[0] != "Rank DESC" {
		t.Fatalf("SearchOrderBy = %v, want [Rank DESC]", searchIdx.SearchOrderBy)
	}

	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var found bool
	for _, d := range reconDDLs {
		csi, ok := d.(*ast.CreateSearchIndex)
		if !ok || leafName(csi.Name) != "DocsSearch" {
			continue
		}
		found = true
		if csi.Where == nil || csi.Where.Expr.SQL() != "TenantId IS NOT NULL AND Rank IS NOT NULL" {
			t.Fatalf("Where = %v", csi.Where)
		}
		if csi.OrderBy == nil || len(csi.OrderBy.Items) != 1 || csi.OrderBy.Items[0].SQL() != "Rank DESC" {
			t.Fatalf("OrderBy = %v", csi.OrderBy)
		}
		if got := csi.SQL(); got != "CREATE SEARCH INDEX DocsSearch ON Docs(BodyTokens) STORING (TenantId, Rank) PARTITION BY TenantId ORDER BY Rank DESC WHERE TenantId IS NOT NULL AND Rank IS NOT NULL" {
			t.Fatalf("search index SQL = %q", got)
		}
	}
	if !found {
		t.Fatal("DocsSearch search index not found in reconstructed DDL")
	}
}

func TestRoundtrip_View(t *testing.T) {
	ddls := []string{
		`CREATE TABLE Users (
  UserId INT64 NOT NULL,
  Name STRING(MAX),
) PRIMARY KEY (UserId)`,
		`CREATE VIEW ActiveUsers SQL SECURITY INVOKER AS SELECT UserId, Name FROM Users`,
	}

	var stmts []ast.DDL
	for _, d := range ddls {
		stmt, err := memefish.ParseDDL("", d)
		if err != nil {
			t.Fatalf("ParseDDL: %v", err)
		}
		stmts = append(stmts, stmt)
	}

	schema, err := FromDDLStatements(stmts)
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if len(schema.Views) != 1 {
		t.Fatalf("expected 1 view, got %d", len(schema.Views))
	}

	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var foundView bool
	for _, d := range reconDDLs {
		if cv, ok := d.(*ast.CreateView); ok {
			foundView = true
			viewName := leafName(cv.Name)
			if viewName != "ActiveUsers" {
				t.Errorf("view name = %q, want %q", viewName, "ActiveUsers")
			}
			if cv.SecurityType != ast.SecurityTypeInvoker {
				t.Errorf("security type = %q, want INVOKER", cv.SecurityType)
			}
		}
	}
	if !foundView {
		t.Error("ActiveUsers view not found in reconstructed DDL")
	}
}

func TestRoundtrip_ChangeStream(t *testing.T) {
	ddl := `CREATE CHANGE STREAM MyChangeStream FOR ALL OPTIONS (retention_period = '7d')`

	stmt, err := memefish.ParseDDL("", ddl)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}

	schema, err := FromDDLStatements([]ast.DDL{stmt})
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if len(schema.ChangeStreams) != 1 {
		t.Fatalf("expected 1 change stream, got %d", len(schema.ChangeStreams))
	}
	if !schema.ChangeStreams[0].All {
		t.Error("change stream should track ALL")
	}

	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var foundCS bool
	for _, d := range reconDDLs {
		if ccs, ok := d.(*ast.CreateChangeStream); ok {
			foundCS = true
			if ccs.Name.Name != "MyChangeStream" {
				t.Errorf("change stream name = %q, want %q", ccs.Name.Name, "MyChangeStream")
			}
			if _, ok := ccs.For.(*ast.ChangeStreamForAll); !ok {
				t.Errorf("change stream FOR type = %T, want ChangeStreamForAll", ccs.For)
			}
		}
	}
	if !foundCS {
		t.Error("change stream not found in reconstructed DDL")
	}
}

func TestRoundtrip_ColumnOnUpdate(t *testing.T) {
	ddl := `CREATE TABLE Users (
  UserId INT64 NOT NULL,
  UpdatedAt TIMESTAMP NOT NULL DEFAULT (PENDING_COMMIT_TIMESTAMP()) ON UPDATE (PENDING_COMMIT_TIMESTAMP()) OPTIONS (allow_commit_timestamp = true),
) PRIMARY KEY (UserId)`

	stmt, err := memefish.ParseDDL("", ddl)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}

	schema, err := FromDDLStatements([]ast.DDL{stmt})
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	var updatedAt *infoschem.Column
	for _, col := range schema.Columns {
		if col.TableName == "Users" && col.ColumnName == "UpdatedAt" {
			updatedAt = col
			break
		}
	}
	if updatedAt == nil {
		t.Fatal("UpdatedAt column not found in schema")
	}
	if updatedAt.ColumnDefault == nil || *updatedAt.ColumnDefault != "PENDING_COMMIT_TIMESTAMP()" {
		t.Fatalf("ColumnDefault = %v, want PENDING_COMMIT_TIMESTAMP()", updatedAt.ColumnDefault)
	}
	if updatedAt.OnUpdateExpression == nil || *updatedAt.OnUpdateExpression != "PENDING_COMMIT_TIMESTAMP()" {
		t.Fatalf("OnUpdateExpression = %v, want PENDING_COMMIT_TIMESTAMP()", updatedAt.OnUpdateExpression)
	}

	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var found bool
	var foundColumn bool
	for _, d := range reconDDLs {
		ct, ok := d.(*ast.CreateTable)
		if !ok {
			continue
		}
		if leafName(ct.Name) != "Users" {
			continue
		}
		found = true
		for _, col := range ct.Columns {
			if col.Name.Name != "UpdatedAt" {
				continue
			}
			foundColumn = true
			ds, ok := col.DefaultSemantics.(*ast.ColumnDefaultExpr)
			if !ok {
				t.Fatalf("DefaultSemantics = %T, want *ast.ColumnDefaultExpr", col.DefaultSemantics)
			}
			if ds.Expr.SQL() != "PENDING_COMMIT_TIMESTAMP()" {
				t.Fatalf("default expr = %q, want PENDING_COMMIT_TIMESTAMP()", ds.Expr.SQL())
			}
			if ds.OnUpdate == nil {
				t.Fatal("OnUpdate is nil")
			}
			if ds.OnUpdate.Expr.SQL() != "PENDING_COMMIT_TIMESTAMP()" {
				t.Fatalf("ON UPDATE expr = %q, want PENDING_COMMIT_TIMESTAMP()", ds.OnUpdate.Expr.SQL())
			}
			if got := col.SQL(); got != "UpdatedAt TIMESTAMP NOT NULL DEFAULT (PENDING_COMMIT_TIMESTAMP()) ON UPDATE (PENDING_COMMIT_TIMESTAMP()) OPTIONS (allow_commit_timestamp = true)" {
				t.Fatalf("column SQL = %q", got)
			}
		}
	}
	if !found {
		t.Fatal("Users table not found in reconstructed DDL")
	}
	if !foundColumn {
		t.Fatal("UpdatedAt column not found in reconstructed DDL")
	}
}

func TestRoundtrip_Sequence(t *testing.T) {
	ddl := `CREATE SEQUENCE MySeq OPTIONS (sequence_kind = 'bit_reversed_positive')`

	stmt, err := memefish.ParseDDL("", ddl)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}

	schema, err := FromDDLStatements([]ast.DDL{stmt})
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if len(schema.Sequences) != 1 {
		t.Fatalf("expected 1 sequence, got %d", len(schema.Sequences))
	}

	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var foundSeq bool
	for _, d := range reconDDLs {
		if cs, ok := d.(*ast.CreateSequence); ok {
			foundSeq = true
			seqName := leafName(cs.Name)
			if seqName != "MySeq" {
				t.Errorf("sequence name = %q, want %q", seqName, "MySeq")
			}
		}
	}
	if !foundSeq {
		t.Error("sequence not found in reconstructed DDL")
	}
}

func TestRoundtrip_TableOptions(t *testing.T) {
	ddl := `CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  Name STRING(MAX),
) PRIMARY KEY (SingerId), OPTIONS (locality_group = 'ssd_only')`

	stmt, err := memefish.ParseDDL("", ddl)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}

	schema, err := FromDDLStatements([]ast.DDL{stmt})
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	// Check that table options were extracted
	if len(schema.TableOptions) != 1 {
		t.Fatalf("expected 1 table option, got %d", len(schema.TableOptions))
	}
	opt := schema.TableOptions[0]
	if opt.TableName != "Singers" {
		t.Errorf("TableName = %q, want %q", opt.TableName, "Singers")
	}
	if opt.OptionName != "locality_group" {
		t.Errorf("OptionName = %q, want %q", opt.OptionName, "locality_group")
	}

	// Round-trip back to DDL
	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var found bool
	for _, d := range reconDDLs {
		ct, ok := d.(*ast.CreateTable)
		if !ok || leafName(ct.Name) != "Singers" {
			continue
		}
		found = true
		if ct.Options == nil {
			t.Fatal("CreateTable.Options is nil, expected OPTIONS(...)")
		}
		if len(ct.Options.Records) != 1 {
			t.Fatalf("expected 1 option record, got %d", len(ct.Options.Records))
		}
		if ct.Options.Records[0].Name.Name != "locality_group" {
			t.Errorf("option name = %q, want %q", ct.Options.Records[0].Name.Name, "locality_group")
		}
		// Verify the generated SQL contains OPTIONS
		sql := ct.SQL()
		if !contains(sql, "OPTIONS") || !contains(sql, "locality_group") {
			t.Errorf("SQL does not contain OPTIONS/locality_group: %s", sql)
		}
	}
	if !found {
		t.Fatal("Singers table not found in reconstructed DDL")
	}
}

func TestRoundtrip_TableOptionsWithColumnOptions(t *testing.T) {
	ddls := []string{
		`CREATE LOCALITY GROUP hdd_group OPTIONS (storage = 'hdd')`,
		`CREATE TABLE Documents (
  DocId INT64 NOT NULL,
  Content STRING(MAX),
  Archive BYTES(MAX) OPTIONS (locality_group = 'hdd_group'),
) PRIMARY KEY (DocId), OPTIONS (locality_group = 'ssd_group')`,
	}

	var stmts []ast.DDL
	for _, d := range ddls {
		stmt, err := memefish.ParseDDL("", d)
		if err != nil {
			t.Fatalf("ParseDDL(%q): %v", d, err)
		}
		stmts = append(stmts, stmt)
	}

	schema, err := FromDDLStatements(stmts)
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	// Check that table-level option exists
	if len(schema.TableOptions) != 1 {
		t.Fatalf("expected 1 table option, got %d", len(schema.TableOptions))
	}
	if schema.TableOptions[0].OptionName != "locality_group" {
		t.Errorf("table option name = %q, want %q", schema.TableOptions[0].OptionName, "locality_group")
	}

	// Check that column-level option exists for Archive column
	var archiveOpt *infoschem.ColumnOption
	for _, co := range schema.ColumnOptions {
		if co.ColumnName == "Archive" && co.OptionName == "locality_group" {
			archiveOpt = co
			break
		}
	}
	if archiveOpt == nil {
		t.Fatal("Archive column locality_group option not found")
	}

	// Check locality group option exists
	if len(schema.LocalityGroupOptions) != 1 {
		t.Fatalf("expected 1 locality group option, got %d", len(schema.LocalityGroupOptions))
	}

	// Round-trip
	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var foundTable bool
	var foundLG bool
	for _, d := range reconDDLs {
		switch dd := d.(type) {
		case *ast.CreateTable:
			if leafName(dd.Name) == "Documents" {
				foundTable = true
				if dd.Options == nil {
					t.Error("Documents table missing OPTIONS")
				}
			}
		case *ast.CreateLocalityGroup:
			if dd.Name.Name == "hdd_group" {
				foundLG = true
			}
		}
	}
	if !foundTable {
		t.Error("Documents table not found in reconstructed DDL")
	}
	if !foundLG {
		t.Error("hdd_group locality group not found in reconstructed DDL")
	}
}

// contains checks if s contains substr (used in test assertions).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestRoundtrip_ForeignKey(t *testing.T) {
	ddls := []string{
		`CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
) PRIMARY KEY (SingerId)`,
		`CREATE TABLE Albums (
  AlbumId INT64 NOT NULL,
  SingerId INT64 NOT NULL,
  CONSTRAINT FK_Singer FOREIGN KEY (SingerId) REFERENCES Singers (SingerId),
) PRIMARY KEY (AlbumId)`,
	}

	var stmts []ast.DDL
	for _, d := range ddls {
		stmt, err := memefish.ParseDDL("", d)
		if err != nil {
			t.Fatalf("ParseDDL: %v", err)
		}
		stmts = append(stmts, stmt)
	}

	schema, err := FromDDLStatements(stmts)
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var foundFK bool
	for _, d := range reconDDLs {
		ct, ok := d.(*ast.CreateTable)
		if !ok || leafName(ct.Name) != "Albums" {
			continue
		}
		for _, tc := range ct.TableConstraints {
			if tc.Name != nil && tc.Name.Name == "FK_Singer" {
				if _, ok := tc.Constraint.(*ast.ForeignKey); ok {
					foundFK = true
				}
			}
		}
	}
	if !foundFK {
		t.Error("FK_Singer foreign key not found in reconstructed DDL")
	}
}

func TestRoundtrip_GeneratedColumn(t *testing.T) {
	ddl := `CREATE TABLE Items (
  ItemId INT64 NOT NULL,
  Price INT64,
  Tax INT64,
  Total INT64 AS (Price + Tax) STORED,
) PRIMARY KEY (ItemId)`

	stmt, err := memefish.ParseDDL("", ddl)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}

	schema, err := FromDDLStatements([]ast.DDL{stmt})
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	// Verify the generated column metadata
	var totalCol *infoschem.Column
	for _, col := range schema.Columns {
		if col.ColumnName == "Total" {
			totalCol = col
			break
		}
	}
	if totalCol == nil {
		t.Fatal("Total column not found")
	}
	if totalCol.IsGenerated != "ALWAYS" {
		t.Errorf("IsGenerated = %q, want ALWAYS", totalCol.IsGenerated)
	}
	if totalCol.GenerationExpression == nil || *totalCol.GenerationExpression != "Price + Tax" {
		t.Errorf("GenerationExpression = %v, want %q", totalCol.GenerationExpression, "Price + Tax")
	}

	// Round-trip
	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var found bool
	for _, d := range reconDDLs {
		ct, ok := d.(*ast.CreateTable)
		if !ok {
			continue
		}
		for _, col := range ct.Columns {
			if col.Name.Name == "Total" {
				found = true
				if _, ok := col.DefaultSemantics.(*ast.GeneratedColumnExpr); !ok {
					t.Errorf("Total column DefaultSemantics = %T, want *ast.GeneratedColumnExpr", col.DefaultSemantics)
				}
			}
		}
	}
	if !found {
		t.Error("Total generated column not found in reconstructed DDL")
	}
}

func TestRoundtrip_RowDeletionPolicy(t *testing.T) {
	ddl := `CREATE TABLE Events (
  EventId INT64 NOT NULL,
  CreatedAt TIMESTAMP NOT NULL,
) PRIMARY KEY (EventId), ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 30 DAY))`

	stmt, err := memefish.ParseDDL("", ddl)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}

	schema, err := FromDDLStatements([]ast.DDL{stmt})
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	// Verify row deletion policy in table metadata
	if len(schema.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(schema.Tables))
	}
	if schema.Tables[0].RowDeletionPolicyExpression == nil {
		t.Fatal("RowDeletionPolicyExpression is nil")
	}

	// Round-trip
	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var found bool
	for _, d := range reconDDLs {
		ct, ok := d.(*ast.CreateTable)
		if !ok {
			continue
		}
		if leafName(ct.Name) == "Events" {
			found = true
			if ct.RowDeletionPolicy == nil {
				t.Error("RowDeletionPolicy is nil in reconstructed DDL")
			}
		}
	}
	if !found {
		t.Error("Events table not found in reconstructed DDL")
	}
}

func TestRoundtrip_RoleAndGrant(t *testing.T) {
	ddls := []string{
		`CREATE TABLE MyTable (
  Id INT64 NOT NULL,
) PRIMARY KEY (Id)`,
		`CREATE ROLE my_reader`,
		`GRANT SELECT ON TABLE MyTable TO ROLE my_reader`,
	}

	var stmts []ast.DDL
	for _, d := range ddls {
		stmt, err := memefish.ParseDDL("", d)
		if err != nil {
			t.Fatalf("ParseDDL(%q): %v", d, err)
		}
		stmts = append(stmts, stmt)
	}

	schema, err := FromDDLStatements(stmts)
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	// Check role was created
	if len(schema.Roles) != 1 || schema.Roles[0].RoleName != "my_reader" {
		t.Fatalf("expected 1 role named my_reader, got %+v", schema.Roles)
	}

	// Check grant was recorded
	if len(schema.RoleTableGrants) != 1 {
		t.Fatalf("expected 1 role table grant, got %d", len(schema.RoleTableGrants))
	}
	rtg := schema.RoleTableGrants[0]
	if rtg.Grantee != "my_reader" || rtg.PrivilegeType != "SELECT" || rtg.TableName != "MyTable" {
		t.Errorf("grant = %+v, want Grantee=my_reader, PrivilegeType=SELECT, TableName=MyTable", rtg)
	}

	// Round-trip
	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var foundRole, foundGrant bool
	for _, d := range reconDDLs {
		switch dd := d.(type) {
		case *ast.CreateRole:
			if dd.Name.Name == "my_reader" {
				foundRole = true
			}
		case *ast.Grant:
			foundGrant = true
		}
	}
	if !foundRole {
		t.Error("my_reader role not found in reconstructed DDL")
	}
	if !foundGrant {
		t.Error("GRANT not found in reconstructed DDL")
	}
}

func TestRoundtrip_SchemaGrant(t *testing.T) {
	ddls := []string{
		`CREATE ROLE my_reader`,
		`GRANT USAGE ON SCHEMA DEFAULT TO ROLE my_reader`,
		`GRANT USAGE ON SCHEMA my_schema, your_schema TO ROLE my_writer`,
	}

	var stmts []ast.DDL
	for _, d := range ddls {
		stmt, err := memefish.ParseDDL("", d)
		if err != nil {
			t.Fatalf("ParseDDL(%q): %v", d, err)
		}
		stmts = append(stmts, stmt)
	}

	schema, err := FromDDLStatements(stmts)
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if len(schema.SchemaGrants) != 3 {
		t.Fatalf("expected 3 schema grants, got %d", len(schema.SchemaGrants))
	}

	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var foundDefault, foundNamed bool
	for _, d := range reconDDLs {
		g, ok := d.(*ast.Grant)
		if !ok {
			continue
		}
		u, ok := g.Privilege.(*ast.UsagePrivilegeOnSchema)
		if !ok {
			continue
		}
		if !u.Default.Invalid() {
			foundDefault = true
		}
		if len(u.Schemas) == 2 {
			foundNamed = true
		}
	}
	if !foundDefault {
		t.Error("GRANT USAGE ON SCHEMA DEFAULT not found in reconstructed DDL")
	}
	if !foundNamed {
		t.Error("GRANT USAGE ON SCHEMA my_schema, your_schema not found in reconstructed DDL")
	}
}

func TestRoundtrip_LocalityGroup(t *testing.T) {
	ddl := `CREATE LOCALITY GROUP cold_storage OPTIONS (storage = 'hdd', ssd_to_hdd_spill_timespan = '1d')`

	stmt, err := memefish.ParseDDL("", ddl)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}

	schema, err := FromDDLStatements([]ast.DDL{stmt})
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if len(schema.LocalityGroupOptions) != 2 {
		t.Fatalf("expected 2 locality group options, got %d", len(schema.LocalityGroupOptions))
	}

	// Round-trip
	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var found bool
	for _, d := range reconDDLs {
		if clg, ok := d.(*ast.CreateLocalityGroup); ok {
			if clg.Name.Name == "cold_storage" {
				found = true
				if clg.Options == nil || len(clg.Options.Records) != 2 {
					t.Errorf("expected 2 options, got %v", clg.Options)
				}
			}
		}
	}
	if !found {
		t.Error("cold_storage locality group not found in reconstructed DDL")
	}
}

func TestRoundtrip_CheckConstraint(t *testing.T) {
	ddl := `CREATE TABLE Accounts (
  AccountId INT64 NOT NULL,
  Balance INT64 NOT NULL,
  CONSTRAINT BalancePositive CHECK (Balance >= 0),
) PRIMARY KEY (AccountId)`

	stmt, err := memefish.ParseDDL("", ddl)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}

	schema, err := FromDDLStatements([]ast.DDL{stmt})
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	// Find the explicit check constraint (not NOT NULL auto-generated ones)
	var checkFound bool
	for _, cc := range schema.CheckConstraints {
		if cc.ConstraintName == "BalancePositive" {
			checkFound = true
			if cc.CheckClause != "Balance >= 0" {
				t.Errorf("CheckClause = %q, want %q", cc.CheckClause, "Balance >= 0")
			}
		}
	}
	if !checkFound {
		t.Fatal("BalancePositive check constraint not found")
	}

	// Round-trip
	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var foundConstraint bool
	for _, d := range reconDDLs {
		ct, ok := d.(*ast.CreateTable)
		if !ok {
			continue
		}
		for _, tc := range ct.TableConstraints {
			if tc.Name != nil && tc.Name.Name == "BalancePositive" {
				if _, ok := tc.Constraint.(*ast.Check); ok {
					foundConstraint = true
				}
			}
		}
	}
	if !foundConstraint {
		t.Error("BalancePositive check constraint not found in reconstructed DDL")
	}
}

func TestRoundtrip_Model(t *testing.T) {
	ddl := `CREATE MODEL MyModel
INPUT(feature1 INT64, feature2 FLOAT64)
OUTPUT(label STRING(MAX))
REMOTE OPTIONS (endpoint = 'https://example.com/predict')`

	stmt, err := memefish.ParseDDL("", ddl)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}

	schema, err := FromDDLStatements([]ast.DDL{stmt})
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if len(schema.Models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(schema.Models))
	}
	if schema.Models[0].ModelName != "MyModel" {
		t.Errorf("ModelName = %q, want MyModel", schema.Models[0].ModelName)
	}
	if !schema.Models[0].IsRemote {
		t.Errorf("IsRemote = %v, want true", schema.Models[0].IsRemote)
	}

	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var found bool
	for _, d := range reconDDLs {
		if cm, ok := d.(*ast.CreateModel); ok {
			if cm.Name.Name == "MyModel" {
				found = true
				if cm.Remote == token.InvalidPos {
					t.Error("Remote flag not set in reconstructed DDL")
				}
				if cm.InputOutput == nil {
					t.Error("InputOutput is nil")
				} else {
					if len(cm.InputOutput.InputColumns) != 2 {
						t.Error("Input columns missing or incorrect")
					}
					if len(cm.InputOutput.OutputColumns) != 1 {
						t.Error("Output columns missing or incorrect")
					}
				}
				if cm.Options == nil || len(cm.Options.Records) != 1 {
					t.Error("Model options missing")
				}
			}
		}
	}
	if !found {
		t.Error("MyModel not found in reconstructed DDL")
	}
}

func TestRoundtrip_Function(t *testing.T) {
	ddl := `CREATE FUNCTION MyFunc(a INT64, b STRING(MAX)) RETURNS INT64 SQL SECURITY INVOKER AS (a + CAST(b AS INT64))`

	stmt, err := memefish.ParseDDL("", ddl)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}

	schema, err := FromDDLStatements([]ast.DDL{stmt})
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if len(schema.Routines) != 1 {
		t.Fatalf("expected 1 routine, got %d", len(schema.Routines))
	}
	if schema.Routines[0].RoutineName != "MyFunc" {
		t.Errorf("RoutineName = %q, want MyFunc", schema.Routines[0].RoutineName)
	}
	if schema.Routines[0].RoutineBody != "SQL" {
		t.Errorf("RoutineBody = %q, want SQL", schema.Routines[0].RoutineBody)
	}

	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var found bool
	for _, d := range reconDDLs {
		if cf, ok := d.(*ast.CreateFunction); ok {
			if leafName(cf.Name) == "MyFunc" {
				found = true
				if len(cf.Params) != 2 {
					t.Errorf("expected 2 params, got %d", len(cf.Params))
				}
				if cf.SqlSecurity != ast.SecurityTypeInvoker {
					t.Errorf("SqlSecurity = %q, want INVOKER", cf.SqlSecurity)
				}
			}
		}
	}
	if !found {
		t.Error("MyFunc not found in reconstructed DDL")
	}
}

func TestRoundtrip_Placement(t *testing.T) {
	ddl := `CREATE PLACEMENT regional_placement OPTIONS (default_leader = 'us-east1')`

	stmt, err := memefish.ParseDDL("", ddl)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}

	schema, err := FromDDLStatements([]ast.DDL{stmt})
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if len(schema.Placements) != 1 {
		t.Fatalf("expected 1 placement, got %d", len(schema.Placements))
	}
	if schema.Placements[0].PlacementName != "regional_placement" {
		t.Errorf("PlacementName = %q, want regional_placement", schema.Placements[0].PlacementName)
	}

	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var found bool
	for _, d := range reconDDLs {
		if cp, ok := d.(*ast.CreatePlacement); ok {
			if cp.Name.Name == "regional_placement" {
				found = true
				if cp.Options == nil || len(cp.Options.Records) != 1 {
					t.Error("Placement options missing")
				}
			}
		}
	}
	if !found {
		t.Error("regional_placement not found in reconstructed DDL")
	}
}

func TestRoundtrip_AlterDatabase(t *testing.T) {
	ddl := `ALTER DATABASE my_db SET OPTIONS (optimizer_version = 4)`

	stmt, err := memefish.ParseDDL("", ddl)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}

	schema, err := FromDDLStatements([]ast.DDL{stmt})
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if len(schema.DatabaseOptions) != 1 {
		t.Fatalf("expected 1 database option, got %d", len(schema.DatabaseOptions))
	}
	if schema.DatabaseOptions[0].OptionName != "optimizer_version" {
		t.Errorf("OptionName = %q, want optimizer_version", schema.DatabaseOptions[0].OptionName)
	}

	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var found bool
	for _, d := range reconDDLs {
		if ad, ok := d.(*ast.AlterDatabase); ok {
			if ad.Name.Name == "my_db" {
				found = true
				if ad.Options == nil || len(ad.Options.Records) != 1 {
					t.Error("Database options missing")
				}
			}
		}
	}
	if !found {
		t.Error("ALTER DATABASE my_db not found in reconstructed DDL")
	}
}

func TestRoundtrip_AlterStatistics(t *testing.T) {
	ddl := `ALTER STATISTICS my_stats SET OPTIONS (allow_gc = false)`

	stmt, err := memefish.ParseDDL("", ddl)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}

	schema, err := FromDDLStatements([]ast.DDL{stmt})
	if err != nil {
		t.Fatalf("FromDDLStatements: %v", err)
	}

	if len(schema.SpannerStatistics) != 1 {
		t.Fatalf("expected 1 statistic, got %d", len(schema.SpannerStatistics))
	}
	if schema.SpannerStatistics[0].PackageName != "my_stats" {
		t.Errorf("PackageName = %q, want my_stats", schema.SpannerStatistics[0].PackageName)
	}
	if schema.SpannerStatistics[0].AllowGC != false {
		t.Errorf("AllowGC = %v, want false", schema.SpannerStatistics[0].AllowGC)
	}

	reconDDLs, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	var found bool
	for _, d := range reconDDLs {
		if as, ok := d.(*ast.AlterStatistics); ok {
			if as.Name.Name == "my_stats" {
				found = true
				if as.Options == nil || len(as.Options.Records) != 1 {
					t.Error("Statistics options missing")
				}
			}
		}
	}
	if !found {
		t.Error("ALTER STATISTICS my_stats not found in reconstructed DDL")
	}
}
