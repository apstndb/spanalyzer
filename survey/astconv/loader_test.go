package astconv

import (
	"context"
	"strings"
	"testing"

	"github.com/apstndb/spanemuboost"
	"github.com/cloudspannerecosystem/memefish/ast"
)

var loaderSampleDDLs = []string{
	"CREATE SCHEMA app",

	`CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(1024),
) PRIMARY KEY (SingerId)`,

	`CREATE TABLE Albums (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  AlbumTitle STRING(MAX),
) PRIMARY KEY (SingerId, AlbumId),
  INTERLEAVE IN PARENT Singers ON DELETE CASCADE`,

	`CREATE TABLE Songs (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  TrackId INT64 NOT NULL,
  SongName STRING(MAX),
) PRIMARY KEY (SingerId, AlbumId, TrackId),
  INTERLEAVE IN PARENT Albums ON DELETE CASCADE`,

	`CREATE INDEX AlbumsByAlbumTitle ON Albums(AlbumTitle)`,

	`CREATE VIEW SingerNames SQL SECURITY INVOKER AS SELECT Singers.SingerId, Singers.FirstName FROM Singers`,

	`CREATE CHANGE STREAM SingerChangeStream FOR Singers`,

	`CREATE SEQUENCE SingerIdSeq OPTIONS (sequence_kind = 'bit_reversed_positive')`,

	`CREATE TABLE Accounts (
  AccountId INT64 NOT NULL,
  Balance INT64 NOT NULL,
  CONSTRAINT BalancePositive CHECK (Balance >= 0),
) PRIMARY KEY (AccountId)`,

	`CREATE TABLE app.Parents (
  ParentId INT64 NOT NULL,
) PRIMARY KEY (ParentId)`,

	`CREATE TABLE app.Children (
  ParentId INT64 NOT NULL,
  Value STRING(MAX),
) PRIMARY KEY (ParentId),
  INTERLEAVE IN PARENT app.Parents ON DELETE CASCADE`,

	`CREATE INDEX app.ChildrenByValue ON app.Children(ParentId, Value)`,

	`CREATE VIEW app.ChildValues SQL SECURITY INVOKER
  AS SELECT Children.ParentId, Children.Value FROM app.Children`,

	`CREATE SEQUENCE app.ChildIdSeq
  OPTIONS (sequence_kind = 'bit_reversed_positive')`,
}

func TestLoadSchema_EmulatorRoundtrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping emulator integration test in short mode")
	}

	ctx := context.Background()
	const emulatorImage = "gcr.io/cloud-spanner-emulator/emulator:1.5.56"
	env, err := spanemuboost.RunWithClients(ctx, spanemuboost.BackendEmulator,
		spanemuboost.WithContainerImage(emulatorImage),
		spanemuboost.WithSetupDDLs(loaderSampleDDLs),
	)
	if err != nil {
		t.Fatalf("RunWithClients: %v", err)
	}
	defer func() { _ = env.Close() }()

	schema, err := LoadSchema(ctx, env.Client)
	if err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}

	var defaultTables int
	var namedTables int
	var namedChildParent string
	for _, tbl := range schema.Tables {
		if tbl.TableSchema == "" && tbl.TableType == "BASE TABLE" {
			defaultTables++
		}
		if tbl.TableSchema == "app" && tbl.TableType == "BASE TABLE" {
			namedTables++
			if tbl.TableName == "Children" && tbl.ParentTableName != nil {
				namedChildParent = *tbl.ParentTableName
			}
		}
	}
	if defaultTables != 4 {
		t.Errorf("default-schema tables = %d, want 4", defaultTables)
	}
	if namedTables != 2 {
		t.Errorf("app-schema tables = %d, want 2", namedTables)
	}
	if namedChildParent != "Parents" {
		t.Errorf("app.Children parent metadata = %q, want leaf name Parents", namedChildParent)
	}
	var namedIndexFound bool
	for _, index := range schema.Indexes {
		if index.TableSchema == "app" &&
			index.TableName == "Children" &&
			index.IndexName == "ChildrenByValue" {
			namedIndexFound = true
		}
	}
	if !namedIndexFound {
		t.Error("INDEXES metadata missing app.ChildrenByValue")
	}
	if len(schema.Views) != 2 {
		t.Errorf("Views = %d, want 2", len(schema.Views))
	}
	if len(schema.ChangeStreams) != 1 {
		t.Errorf("ChangeStreams = %d, want 1", len(schema.ChangeStreams))
	}
	if len(schema.ChangeStreamOptions) != 0 {
		t.Errorf("omitted retention produced ChangeStreamOptions = %#v", schema.ChangeStreamOptions)
	}
	if len(schema.Sequences) != 2 {
		t.Errorf("Sequences = %d, want 2", len(schema.Sequences))
	}

	ddls, err := schema.ToDDLStatements()
	if err != nil {
		t.Fatalf("ToDDLStatements: %v", err)
	}

	tables := make(map[string]*ast.CreateTable)
	indexes := make(map[string]*ast.CreateIndex)
	views := make(map[string]*ast.CreateView)
	sequences := make(map[string]*ast.CreateSequence)
	var changeStream *ast.CreateChangeStream
	for _, d := range ddls {
		switch ddl := d.(type) {
		case *ast.CreateTable:
			tables[ddl.Name.SQL()] = ddl
		case *ast.CreateIndex:
			indexes[ddl.Name.SQL()] = ddl
		case *ast.CreateView:
			views[ddl.Name.SQL()] = ddl
		case *ast.CreateChangeStream:
			changeStream = ddl
		case *ast.CreateSequence:
			sequences[ddl.Name.SQL()] = ddl
		}
	}

	singers := tables["Singers"]
	if singers == nil {
		t.Fatal("reconstructed DDL missing Singers table")
	}
	if len(singers.Columns) != 2 || singers.Columns[0].Name.Name != "SingerId" || singers.Columns[0].Type.SQL() != "INT64" || singers.Columns[1].Name.Name != "FirstName" || singers.Columns[1].Type.SQL() != "STRING(1024)" {
		t.Errorf("reconstructed Singers columns = %s", singers.SQL())
	}
	if len(singers.PrimaryKeys) != 1 || singers.PrimaryKeys[0].Name.Name != "SingerId" {
		t.Errorf("reconstructed Singers primary key = %s", singers.SQL())
	}

	for _, want := range []struct {
		name   string
		parent string
	}{
		{name: "Albums", parent: "Singers"},
		{name: "Songs", parent: "Albums"},
	} {
		table := tables[want.name]
		if table == nil || table.Cluster == nil || leafName(table.Cluster.TableName) != want.parent || !table.Cluster.Enforced || table.Cluster.OnDelete != ast.OnDeleteCascade {
			t.Errorf("reconstructed %s interleave = %#v, want enforced parent %s ON DELETE CASCADE", want.name, table, want.parent)
		}
	}

	accounts := tables["Accounts"]
	if accounts == nil || len(accounts.TableConstraints) != 1 || accounts.TableConstraints[0].Constraint == nil || !strings.Contains(accounts.TableConstraints[0].Constraint.SQL(), "Balance >= 0") {
		t.Errorf("reconstructed Accounts check constraint = %#v", accounts)
	}
	view := views["SingerNames"]
	if view == nil || view.Name == nil || leafName(view.Name) != "SingerNames" || !strings.Contains(view.Query.SQL(), "Singers.SingerId") {
		t.Errorf("reconstructed view = %#v", view)
	}
	if changeStream == nil {
		t.Fatal("reconstructed DDL missing change stream")
	}
	if changeStream.Options != nil {
		t.Errorf("omitted retention was materialized as %s", changeStream.Options.SQL())
	}
	forTables, ok := changeStream.For.(*ast.ChangeStreamForTables)
	if !ok || len(forTables.Tables) != 1 || forTables.Tables[0].TableName.Name != "Singers" || len(forTables.Tables[0].Columns) != 0 || !forTables.Tables[0].Rparen.Invalid() {
		t.Errorf("reconstructed change stream targets = %#v", changeStream.For)
	}
	sequence := sequences["SingerIdSeq"]
	if sequence == nil || leafName(sequence.Name) != "SingerIdSeq" || len(sequence.Params) != 1 {
		t.Errorf("reconstructed sequence = %#v", sequence)
	} else if _, ok := sequence.Params[0].(*ast.BitReversedPositive); !ok {
		t.Errorf("reconstructed sequence parameter = %T, want *ast.BitReversedPositive", sequence.Params[0])
	}

	namedChild := tables["app.Children"]
	if namedChild == nil ||
		namedChild.Cluster == nil ||
		namedChild.Cluster.TableName.SQL() != "app.Parents" ||
		!namedChild.Cluster.Enforced ||
		namedChild.Cluster.OnDelete != ast.OnDeleteCascade {
		t.Errorf("reconstructed app.Children interleave = %#v", namedChild)
	}
	namedIndex := indexes["app.ChildrenByValue"]
	if namedIndex == nil ||
		namedIndex.TableName.SQL() != "app.Children" ||
		namedIndex.InterleaveIn != nil {
		t.Errorf("reconstructed app.ChildrenByValue = %#v", namedIndex)
	}
	namedView := views["app.ChildValues"]
	if namedView == nil || !strings.Contains(namedView.Query.SQL(), "app.Children") {
		t.Errorf("reconstructed app.ChildValues = %#v", namedView)
	}
	namedSequence := sequences["app.ChildIdSeq"]
	if namedSequence == nil || len(namedSequence.Params) != 1 {
		t.Errorf("reconstructed app.ChildIdSeq = %#v", namedSequence)
	} else if _, ok := namedSequence.Params[0].(*ast.BitReversedPositive); !ok {
		t.Errorf("reconstructed app.ChildIdSeq parameter = %T, want *ast.BitReversedPositive", namedSequence.Params[0])
	}
}
