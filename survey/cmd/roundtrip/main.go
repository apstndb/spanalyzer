// Command roundtrip demonstrates the full pipeline:
//  1. Start a Spanner emulator via spanemuboost (testcontainers)
//  2. Create a database with sample DDL
//  3. Query INFORMATION_SCHEMA via astconv.LoadSchema
//  4. Convert to memefish AST → generate SQL
package main

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/apstndb/spanemuboost"
	"github.com/apstndb/spanner-emulator-survey/astconv"
)

var sampleDDLs = []string{
	`CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(1024),
  LastName STRING(1024),
  SingerInfo BYTES(MAX),
  BirthDate DATE,
) PRIMARY KEY (SingerId)`,

	`CREATE TABLE Albums (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  AlbumTitle STRING(MAX),
  MarketingBudget INT64,
) PRIMARY KEY (SingerId, AlbumId),
  INTERLEAVE IN PARENT Singers ON DELETE CASCADE`,

	`CREATE TABLE Songs (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  TrackId INT64 NOT NULL,
  SongName STRING(MAX),
  Duration INT64,
) PRIMARY KEY (SingerId, AlbumId, TrackId),
  INTERLEAVE IN PARENT Albums ON DELETE CASCADE`,

	`CREATE INDEX SongsBySongName ON Songs(SongName)`,

	`CREATE UNIQUE INDEX SongsBySingerAlbumSongNameDesc ON Songs(SingerId, AlbumId, SongName DESC), INTERLEAVE IN Albums`,

	`CREATE NULL_FILTERED INDEX AlbumsByAlbumTitle ON Albums(AlbumTitle) STORING (MarketingBudget)`,

	`CREATE TABLE Events (
  EventId INT64 NOT NULL,
  EventName STRING(MAX),
  StartTime TIMESTAMP NOT NULL,
  CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
) PRIMARY KEY (EventId)`,

	`CREATE VIEW SingerNames SQL SECURITY INVOKER AS SELECT Singers.SingerId, Singers.FirstName, Singers.LastName FROM Singers`,

	`CREATE CHANGE STREAM SingerChangeStream FOR Singers`,

	`CREATE SEQUENCE SingerIdSeq OPTIONS (sequence_kind = 'bit_reversed_positive')`,

	`CREATE TABLE Accounts (
  AccountId INT64 NOT NULL,
  Balance INT64 NOT NULL,
  CONSTRAINT BalancePositive CHECK (Balance >= 0),
) PRIMARY KEY (AccountId)`,
}

func main() {
	ctx := context.Background()

	fmt.Println("Starting Spanner emulator via spanemuboost...")
	env, err := spanemuboost.RunEmulatorWithClients(
		ctx,
		spanemuboost.WithSetupDDLs(sampleDDLs),
	)
	if err != nil {
		log.Fatalf("Failed to start emulator: %v", err)
	}
	defer func() { _ = env.Close() }()

	fmt.Printf("Emulator ready: %s\n", env.URI())
	fmt.Println()

	// Query INFORMATION_SCHEMA and build Schema
	schema, err := astconv.LoadSchema(ctx, env.Client)
	if err != nil {
		log.Fatalf("Failed to load schema: %v", err)
	}

	printSchemaSummary(schema)

	// Convert to DDL
	ddls, err := schema.ToDDLStatements()
	if err != nil {
		log.Fatalf("Failed to convert to DDL: %v", err)
	}

	fmt.Println("=== Generated DDL ===")
	fmt.Println()
	for _, d := range ddls {
		fmt.Println(d.SQL() + ";")
		fmt.Println()
	}
}

func printSchemaSummary(schema *astconv.Schema) {
	fmt.Println("=== INFORMATION_SCHEMA Summary ===")
	fmt.Printf("  Tables:               %d\n", len(schema.Tables))
	fmt.Printf("  Columns:              %d\n", len(schema.Columns))
	fmt.Printf("  Indexes:              %d\n", len(schema.Indexes))
	fmt.Printf("  Index Columns:        %d\n", len(schema.IndexColumns))
	fmt.Printf("  Table Constraints:    %d\n", len(schema.TableConstraints))
	fmt.Printf("  Check Constraints:    %d\n", len(schema.CheckConstraints))
	fmt.Printf("  Views:                %d\n", len(schema.Views))
	fmt.Printf("  Change Streams:       %d\n", len(schema.ChangeStreams))
	fmt.Printf("  Sequences:            %d\n", len(schema.Sequences))
	fmt.Printf("  Models:               %d\n", len(schema.Models))
	fmt.Printf("  Property Graphs:      %d\n", len(schema.PropertyGraphs))
	fmt.Printf("  Locality Group Opts:  %d\n", len(schema.LocalityGroupOptions))

	fmt.Println()
	fmt.Println("--- Tables ---")
	sort.Slice(schema.Tables, func(i, j int) bool {
		return schema.Tables[i].TableName < schema.Tables[j].TableName
	})
	for _, t := range schema.Tables {
		parent := ""
		if t.ParentTableName != nil && *t.ParentTableName != "" {
			parent = fmt.Sprintf(" (INTERLEAVE IN %s)", *t.ParentTableName)
		}
		fmt.Printf("  %s [%s]%s\n", t.TableName, t.TableType, parent)
	}
	fmt.Println()
}
