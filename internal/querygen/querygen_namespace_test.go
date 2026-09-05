package querygen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateQueryCodePackageNamespaceCollisions(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	writeTestFile(t, filepath.Join(dir, "bigquery.sql"), `
CREATE TABLE mydataset.events (
  event_id INT64
);
`)

	spanner := QueryCodegenSchema{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}
	tests := []struct {
		name    string
		config  QueryCodegenConfig
		symbol  string
		origins []string
	}{
		{
			name: "query function versus another query All helper",
			config: QueryCodegenConfig{
				Package: "db",
				Client:  GoStructTargetSpanner,
				Schemas: []QueryCodegenSchema{spanner},
				Queries: []QueryCodegenQuery{
					{Name: "ListSingers", Catalog: "spanner", SQL: "SELECT SingerId FROM Singers", Result: "many", ResultStruct: "SingerRow"},
					{Name: "ListSingersAll", Catalog: "spanner", SQL: "SELECT SingerId FROM Singers", ResultStruct: "OtherRow"},
				},
			},
			symbol:  "ListSingersAll",
			origins: []string{"queries[0] (ListSingers) all helper", "queries[1] (ListSingersAll) function"},
		},
		{
			name: "exportedIdentifier folding",
			config: QueryCodegenConfig{
				Package: "db",
				Client:  GoStructTargetSpanner,
				Schemas: []QueryCodegenSchema{spanner},
				Queries: []QueryCodegenQuery{
					{Name: "GetSinger", Catalog: "spanner", SQL: "SELECT SingerId FROM Singers", ResultStruct: "SingerRow"},
					{Name: "get_singer", Catalog: "spanner", SQL: "SELECT SingerId FROM Singers", ResultStruct: "OtherRow"},
				},
			},
			symbol:  "GetSinger",
			origins: []string{"queries[0] (GetSinger) function", "queries[1] (get_singer) function"},
		},
		{
			name: "query function versus result struct",
			config: QueryCodegenConfig{
				Package: "db",
				Client:  GoStructTargetSpanner,
				Schemas: []QueryCodegenSchema{spanner},
				Queries: []QueryCodegenQuery{
					{Name: "SingerRow", Catalog: "spanner", SQL: "SELECT SingerId FROM Singers", ResultStruct: "SingerRow"},
				},
			},
			symbol:  "SingerRow",
			origins: []string{"queries[0] (SingerRow) function", "queries[0] (SingerRow) result struct"},
		},
		{
			name: "query result struct versus write DML constant",
			config: QueryCodegenConfig{
				Package: "db",
				Client:  GoStructTargetSpanner,
				Schemas: []QueryCodegenSchema{spanner},
				Queries: []QueryCodegenQuery{
					{Name: "GetSinger", Catalog: "spanner", SQL: "SELECT SingerId FROM Singers", ResultStruct: "SaveSingerDML"},
				},
				Writes: []QueryCodegenWrite{
					{Name: "SaveSinger", Catalog: "spanner", Table: "Singers", Methods: []string{"dml"}},
				},
			},
			symbol:  "SaveSingerDML",
			origins: []string{"queries[0] (GetSinger) result struct", "writes[0] (SaveSinger) dml constant"},
		},
		{
			name: "write DML constant versus query function",
			config: QueryCodegenConfig{
				Package: "db",
				Client:  GoStructTargetSpanner,
				Schemas: []QueryCodegenSchema{spanner},
				Queries: []QueryCodegenQuery{
					{Name: "SaveSingerDML", Catalog: "spanner", SQL: "SELECT SingerId FROM Singers", ResultStruct: "SingerRow"},
				},
				Writes: []QueryCodegenWrite{
					{Name: "SaveSinger", Catalog: "spanner", Table: "Singers", Methods: []string{"dml"}},
				},
			},
			symbol:  "SaveSingerDML",
			origins: []string{"queries[0] (SaveSingerDML) function", "writes[0] (SaveSinger) dml constant"},
		},
		{
			name: "query SQL constant versus query function",
			config: QueryCodegenConfig{
				Package: "db",
				Client:  GoStructTargetSpanner,
				Schemas: []QueryCodegenSchema{spanner},
				Queries: []QueryCodegenQuery{
					{Name: "GetSingerSQL", Catalog: "spanner", SQL: "SELECT SingerId FROM Singers", ResultStruct: "SingerRow"},
				},
			},
			symbol:  "GetSingerSQL",
			origins: []string{"queries[0] (GetSingerSQL) function", "queries[0] (GetSingerSQL) sql constant"},
		},
		{
			name: "optional builder versus another query function",
			config: QueryCodegenConfig{
				Package: "db",
				Client:  GoStructTargetSpanner,
				Schemas: []QueryCodegenSchema{spanner},
				Queries: []QueryCodegenQuery{
					{Name: "ListSingers", Catalog: "spanner", SQL: "SELECT SingerId FROM Singers WHERE TRUE /*?optional:name*/ AND FirstName = @name /*?end*/", Params: []QueryCodegenParam{{Name: "name", Type: "STRING", Optional: "omit_when_null"}}, ResultStruct: "SingerRow"},
					{Name: "BuildListSingersSQL", Catalog: "spanner", SQL: "SELECT SingerId FROM Singers", ResultStruct: "OtherRow"},
				},
			},
			symbol:  "BuildListSingersSQL",
			origins: []string{"queries[0] (ListSingers) sql builder", "queries[1] (BuildListSingersSQL) function"},
		},
		{
			name: "generated support versus query function",
			config: QueryCodegenConfig{
				Package: "db",
				Client:  GoStructTargetBoth,
				Schemas: []QueryCodegenSchema{spanner},
				Queries: []QueryCodegenQuery{
					{Name: "NullValue", Catalog: "spanner", SQL: "SELECT FirstName FROM Singers", ResultStruct: "SingerRow"},
				},
			},
			symbol:  "NullValue",
			origins: []string{"generated support NullValue", "queries[0] (NullValue) function"},
		},
		{
			name: "case folding collision",
			config: QueryCodegenConfig{
				Package: "db",
				Client:  GoStructTargetSpanner,
				Schemas: []QueryCodegenSchema{spanner},
				Queries: []QueryCodegenQuery{
					{Name: "ListEvents", Catalog: "spanner", SQL: "SELECT SingerId FROM Singers", Result: "many", ResultStruct: "EventRow"},
					{Name: "list_events", Catalog: "spanner", SQL: "SELECT SingerId FROM Singers", ResultStruct: "OtherRow"},
				},
			},
			symbol:  "ListEvents",
			origins: []string{"queries[0] (ListEvents) function", "queries[1] (list_events) function"},
		},
		{
			name: "optional params type versus result struct",
			config: QueryCodegenConfig{
				Package: "db",
				Client:  GoStructTargetSpanner,
				Schemas: []QueryCodegenSchema{spanner},
				Queries: []QueryCodegenQuery{
					{Name: "ListSingers", Catalog: "spanner", SQL: "SELECT SingerId FROM Singers WHERE TRUE /*?optional:name*/ AND FirstName = @name /*?end*/", Params: []QueryCodegenParam{{Name: "name", Type: "STRING", Optional: "omit_when_null"}}, ResultStruct: "SingerRow"},
					{Name: "GetSinger", Catalog: "spanner", SQL: "SELECT SingerId FROM Singers", ResultStruct: "ListSingersParams"},
				},
			},
			symbol:  "ListSingersParams",
			origins: []string{"queries[0] (ListSingers) optional params type", "queries[1] (GetSinger) result struct"},
		},
		{
			name: "SQL constant versus result struct",
			config: QueryCodegenConfig{
				Package: "db",
				Client:  GoStructTargetSpanner,
				Schemas: []QueryCodegenSchema{spanner},
				Queries: []QueryCodegenQuery{
					{Name: "GetSinger", Catalog: "spanner", SQL: "SELECT SingerId FROM Singers", ResultStruct: "GetSingerSQL"},
				},
			},
			symbol:  "GetSingerSQL",
			origins: []string{"queries[0] (GetSinger) result struct", "queries[0] (GetSinger) sql constant"},
		},
		{
			name: "write DML constant versus another DML constant",
			config: QueryCodegenConfig{
				Package: "db",
				Client:  GoStructTargetSpanner,
				Schemas: []QueryCodegenSchema{spanner},
				Writes: []QueryCodegenWrite{
					{Name: "SaveSinger", Catalog: "spanner", Table: "Singers", InputStruct: "SingerA", Methods: []string{"dml"}},
					{Name: "save_singer", Catalog: "spanner", Table: "Singers", InputStruct: "SingerB", Methods: []string{"dml"}},
				},
			},
			symbol:  "SaveSingerDML",
			origins: []string{"writes[0] (SaveSinger) dml constant", "writes[1] (save_singer) dml constant"},
		},
		{
			name: "SQL constant versus write input struct",
			config: QueryCodegenConfig{
				Package: "db",
				Client:  GoStructTargetSpanner,
				Schemas: []QueryCodegenSchema{spanner},
				Queries: []QueryCodegenQuery{
					{Name: "GetSinger", Catalog: "spanner", SQL: "SELECT SingerId FROM Singers", ResultStruct: "SingerRow"},
				},
				Writes: []QueryCodegenWrite{
					{Name: "SaveSinger", Catalog: "spanner", Table: "Singers", InputStruct: "GetSingerSQL", Methods: []string{"mutation"}},
				},
			},
			symbol:  "GetSingerSQL",
			origins: []string{"queries[0] (GetSinger) sql constant", "writes[0] (SaveSinger) input struct"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var firstErr string
			for i := 0; i < 5; i++ {
				code, err := GenerateQueryCode(tt.config, dir)
				if err == nil {
					t.Fatalf("GenerateQueryCode() error = nil, want symbol %s collision\n%s", tt.symbol, code)
				}
				if code != "" {
					t.Fatalf("GenerateQueryCode() returned source on error:\n%s", code)
				}
				if !strings.Contains(err.Error(), "generated symbol "+tt.symbol) {
					t.Fatalf("error = %v, want symbol %s", err, tt.symbol)
				}
				for _, origin := range tt.origins {
					if !strings.Contains(err.Error(), origin) {
						t.Fatalf("error = %v, want origin %q", err, origin)
					}
				}
				if firstErr == "" {
					firstErr = err.Error()
				} else if err.Error() != firstErr {
					t.Fatalf("collision diagnostic is not deterministic:\nfirst: %s\nlater: %s", firstErr, err.Error())
				}
			}
			_, planErr := BuildQueryCodegenPlan(tt.config, dir)
			if planErr == nil {
				t.Fatal("BuildQueryCodegenPlan() error = nil, want the same collision")
			}
			if planErr.Error() != firstErr {
				t.Fatalf("BuildQueryCodegenPlan() error = %v, want %s", planErr, firstErr)
			}
		})
	}
}

func TestGenerateQueryCodeWriteOnlyStructDoesNotReserveQuerySupport(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE T (
  Id INT64 NOT NULL,
  Name STRING(MAX)
) PRIMARY KEY (Id);
`)
	const configYAML = `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
  ddl: schema.sql
writes:
- name: InsertRow
  catalog: app
  table: T
  operation: insert
  input: NullValue
  insert:
    columns: [Id, Name]
`
	for _, tt := range []struct {
		name      string
		queries   string
		collision bool
	}{
		{name: "write only"},
		{
			name: "query without nullable support",
			queries: `
queries:
- name: GetID
  catalog: app
  kind: sql
  sql: SELECT Id FROM T
  result:
    struct: IDRow
    required:
      fields: [Id]
`,
		},
		{
			name: "query requires genuine support",
			queries: `
queries:
- name: GetName
  catalog: app
  kind: sql
  sql: SELECT Name FROM T
  result:
    struct: NameRow
`,
			collision: true,
		},
		{
			name: "shared query and write struct requires genuine support",
			queries: `
queries:
- name: GetRow
  catalog: app
  kind: sql
  sql: SELECT Id, Name FROM T
  result:
    struct: NullValue
`,
			collision: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseQueryCodegenConfigYAML([]byte(configYAML + tt.queries))
			if err != nil {
				t.Fatal(err)
			}
			code, genErr := GenerateQueryCode(config, dir)
			_, planErr := BuildQueryCodegenPlan(config, dir)
			if tt.collision {
				for _, err := range []error{genErr, planErr} {
					if err == nil || !strings.Contains(err.Error(), "generated support NullValue") {
						t.Fatalf("error = %v, want genuine NullValue support collision", err)
					}
				}
				if code != "" {
					t.Fatal("generation returned source on a collision")
				}
				return
			}
			if genErr != nil || planErr != nil {
				t.Fatalf("write-only DTO reserved query support: generate=%v plan=%v", genErr, planErr)
			}
			if strings.Count(code, "type NullValue struct") != 1 || strings.Contains(code, "type NullValue[") {
				t.Fatalf("want only the write DTO named NullValue:\n%s", code)
			}
			compileGeneratedPackage(t, code)
		})
	}
}

func TestGenerateQueryCodePackageNamespaceSharedStructAndMethodOverlapRemainValid(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	shared := QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetSpanner,
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Queries: []QueryCodegenQuery{{
			Name:         "GetSinger",
			Catalog:      "spanner",
			SQL:          "SELECT SingerId, FirstName FROM Singers WHERE SingerId = @id",
			Params:       []QueryCodegenParam{{Name: "id", Type: "INT64"}},
			Result:       "maybe_one",
			ResultStruct: "SingerRow",
		}},
		Writes: []QueryCodegenWrite{{
			Name:        "SaveSinger",
			Catalog:     "spanner",
			Table:       "Singers",
			InputStruct: "SingerRow",
			Methods:     []string{"mutation", "dml"},
		}},
	}
	code, err := GenerateQueryCode(shared, dir)
	if err != nil {
		t.Fatalf("shared struct GenerateQueryCode() error = %v", err)
	}
	if !strings.Contains(code, "type SingerRow struct") {
		t.Fatalf("shared struct missing SingerRow:\n%s", code)
	}
	if strings.Count(code, "type SingerRow struct") != 1 {
		t.Fatalf("shared struct emitted more than once:\n%s", code)
	}

	overlap := QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetSpanner,
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Queries: []QueryCodegenQuery{{
			Name:         "SaveSingerMutation",
			Catalog:      "spanner",
			SQL:          "SELECT SingerId FROM Singers",
			ResultStruct: "SingerRow",
		}},
		Writes: []QueryCodegenWrite{{
			Name:    "SaveSinger",
			Catalog: "spanner",
			Table:   "Singers",
			Methods: []string{"mutation"},
		}},
	}
	if _, err := GenerateQueryCode(overlap, dir); err != nil {
		t.Fatalf("method/function overlap GenerateQueryCode() error = %v", err)
	}

	sharedWrites := QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetSpanner,
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Writes: []QueryCodegenWrite{
			{
				Name:        "InsertSinger",
				Catalog:     "spanner",
				Table:       "Singers",
				InputStruct: "SingerRow",
				Methods:     []string{"mutation"},
			},
			{
				Name:        "UpsertSinger",
				Catalog:     "spanner",
				Table:       "Singers",
				InputStruct: "SingerRow",
				Methods:     []string{"dml"},
			},
		},
	}
	writeCode, err := GenerateQueryCode(sharedWrites, dir)
	if err != nil {
		t.Fatalf("shared write/write struct GenerateQueryCode() error = %v", err)
	}
	if strings.Count(writeCode, "type SingerRow struct") != 1 {
		t.Fatalf("shared write/write struct emitted SingerRow %d times:\n%s", strings.Count(writeCode, "type SingerRow struct"), writeCode)
	}
	compileGeneratedPackage(t, writeCode)
}

func TestGenerateQueryCodeWriteReceiverMethodNamespace(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	spanner := QueryCodegenSchema{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}
	sameReceiver := QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetSpanner,
		Schemas: []QueryCodegenSchema{spanner},
		Writes: []QueryCodegenWrite{
			{Name: "SaveSinger", Catalog: "spanner", Table: "Singers", InputStruct: "SingerRow", Methods: []string{"mutation"}},
			{Name: "save_singer", Catalog: "spanner", Table: "Singers", InputStruct: "SingerRow", Methods: []string{"mutation"}},
		},
	}
	var firstErr string
	for i := 0; i < 5; i++ {
		code, err := GenerateQueryCode(sameReceiver, dir)
		if err == nil {
			t.Fatalf("same-receiver mutation collision GenerateQueryCode() error = nil\n%s", code)
		}
		if code != "" {
			t.Fatalf("GenerateQueryCode() returned source on error:\n%s", code)
		}
		if !strings.Contains(err.Error(), "generated method SaveSingerMutation on SingerRow") {
			t.Fatalf("error = %v, want SaveSingerMutation on SingerRow", err)
		}
		for _, origin := range []string{"writes[0] (SaveSinger) mutation method", "writes[1] (save_singer) mutation method"} {
			if !strings.Contains(err.Error(), origin) {
				t.Fatalf("error = %v, want origin %q", err, origin)
			}
		}
		if firstErr == "" {
			firstErr = err.Error()
		} else if err.Error() != firstErr {
			t.Fatalf("receiver collision diagnostic is not deterministic:\nfirst: %s\nlater: %s", firstErr, err.Error())
		}
	}
	if _, planErr := BuildQueryCodegenPlan(sameReceiver, dir); planErr == nil || planErr.Error() != firstErr {
		t.Fatalf("BuildQueryCodegenPlan() error mismatch")
	}

	differentReceivers := QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetSpanner,
		Schemas: []QueryCodegenSchema{spanner},
		Writes: []QueryCodegenWrite{
			{Name: "SaveSinger", Catalog: "spanner", Table: "Singers", InputStruct: "SingerA", Methods: []string{"mutation"}},
			{Name: "save_singer", Catalog: "spanner", Table: "Singers", InputStruct: "SingerB", Methods: []string{"mutation"}},
		},
	}
	differentCode, err := GenerateQueryCode(differentReceivers, dir)
	if err != nil {
		t.Fatalf("different-receiver mutation GenerateQueryCode() error = %v", err)
	}
	if !strings.Contains(differentCode, "func (w *SingerA) SaveSingerMutation()") || !strings.Contains(differentCode, "func (w *SingerB) SaveSingerMutation()") {
		t.Fatalf("different-receiver methods missing:\n%s", differentCode)
	}
	compileGeneratedPackage(t, differentCode)

	distinctNames := QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetSpanner,
		Schemas: []QueryCodegenSchema{spanner},
		Writes: []QueryCodegenWrite{
			{Name: "SaveSinger", Catalog: "spanner", Table: "Singers", InputStruct: "SingerRow", Methods: []string{"mutation"}},
			{Name: "UpdateSinger", Catalog: "spanner", Table: "Singers", InputStruct: "SingerRow", Methods: []string{"mutation"}},
		},
	}
	distinctCode, err := GenerateQueryCode(distinctNames, dir)
	if err != nil {
		t.Fatalf("same-receiver distinct methods GenerateQueryCode() error = %v", err)
	}
	if !strings.Contains(distinctCode, "func (w *SingerRow) SaveSingerMutation()") || !strings.Contains(distinctCode, "func (w *SingerRow) UpdateSingerMutation()") {
		t.Fatalf("distinct same-receiver methods missing:\n%s", distinctCode)
	}
	compileGeneratedPackage(t, distinctCode)
}

func TestGenerateQueryCodePackageNamespaceDeterministicValidOutput(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	cfg := QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetSpanner,
		Schemas: []QueryCodegenSchema{{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"}},
		Queries: []QueryCodegenQuery{{
			Name:         "ListSingers",
			Catalog:      "spanner",
			SQL:          "SELECT SingerId, FirstName FROM Singers",
			Result:       "many",
			ResultStruct: "SingerRow",
		}},
	}
	first, err := GenerateQueryCode(cfg, dir)
	if err != nil {
		t.Fatalf("first GenerateQueryCode() error = %v", err)
	}
	second, err := GenerateQueryCode(cfg, dir)
	if err != nil {
		t.Fatalf("second GenerateQueryCode() error = %v", err)
	}
	if first != second {
		t.Fatal("GenerateQueryCode() output is not deterministic")
	}
}

func TestGenerateQueryCodeFederatedSQLConstantMatchesMethods(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	code, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBigQuery,
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{Name: "bigquery", Dialect: "bigquery", ExternalQueryConnections: []QueryCodegenExternalSchema{{
				Connection:    "example-project.us.example-connection",
				SpannerSource: "spanner",
			}}},
		},
		Queries: []QueryCodegenQuery{{
			Name:    "ExternalSingerIDs",
			Catalog: "bigquery",
			Federated: QueryCodegenFederatedQuery{
				Connection:    "example-project.us.example-connection",
				SpannerSource: "spanner",
				InnerSQL:      "SELECT SingerId FROM Singers",
			},
			Result:       "many",
			ResultStruct: "SingerRow",
		}},
	}, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode() error = %v", err)
	}
	for _, want := range []string{
		"ExternalSingerIDsSpannerSQL",
		"ExternalSingerIDsBigQuerySQL",
		"client.Query(ExternalSingerIDsBigQuerySQL)",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated federated code missing %q:\n%s", want, code)
		}
	}
	if strings.Contains(code, "client.Query(ExternalSingerIDsSQL)") {
		t.Fatalf("federated methods still reference missing ExternalSingerIDsSQL:\n%s", code)
	}
}

func TestGenerateQueryCodePackageNamespaceNestedStructCollision(t *testing.T) {
	fields := []goResultField{{
		Name: "info",
		Kind: "STRUCT",
		Fields: []goResultField{
			{Name: "name", Kind: "STRING"},
		},
	}}
	specs := []resolvedQuerySpec{{
		Index:        0,
		Name:         "GetRow",
		MethodPrefix: "GetRow",
		ResultStruct: "Row",
		ResultMode:   "one",
	}, {
		Index:        1,
		Name:         "GetInfo",
		MethodPrefix: "GetInfo",
		ResultStruct: "RowInfo",
		ResultMode:   "one",
	}}
	var firstErr string
	for i := 0; i < 5; i++ {
		err := validateGeneratedPackageNamespace(
			specs,
			nil,
			map[string][]goResultField{"Row": fields, "RowInfo": {{Name: "id", Kind: "INT64"}}},
			nil,
			GoStructTargetSpanner,
		)
		if err == nil {
			t.Fatal("validateGeneratedPackageNamespace() error = nil, want nested struct collision")
		}
		if !strings.Contains(err.Error(), "generated symbol RowInfo") || !strings.Contains(err.Error(), "nested struct RowInfo") || !strings.Contains(err.Error(), "queries[1] (GetInfo) result struct") {
			t.Fatalf("error = %v, want RowInfo nested vs result struct", err)
		}
		if !strings.Contains(err.Error(), "queries[0] (GetRow) result struct nested struct RowInfo") {
			t.Fatalf("error = %v, want nested vs result origins", err)
		}
		if firstErr == "" {
			firstErr = err.Error()
		} else if err.Error() != firstErr {
			t.Fatalf("nested collision diagnostic is not deterministic:\nfirst: %s\nlater: %s", firstErr, err.Error())
		}
	}
}

func TestParseV1AlphaCollidingConfigDoesNotReturnSource(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	config, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
  ddl: schema.sql
queries:
- name: SingerRow
  catalog: app
  kind: sql
  sql: SELECT SingerId FROM Singers
  result:
    struct: SingerRow
`))
	if err != nil {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v", err)
	}
	var firstErr string
	for i := 0; i < 5; i++ {
		code, genErr := GenerateQueryCode(config, dir)
		if genErr == nil {
			t.Fatalf("GenerateQueryCode() error = nil, want collision\n%s", code)
		}
		if code != "" {
			t.Fatalf("v1alpha GenerateQueryCode() returned source on error:\n%s", code)
		}
		if !strings.Contains(genErr.Error(), "generated symbol SingerRow") {
			t.Fatalf("error = %v, want SingerRow", genErr)
		}
		if !strings.Contains(genErr.Error(), "queries[0] (SingerRow) function") || !strings.Contains(genErr.Error(), "queries[0] (SingerRow) result struct") {
			t.Fatalf("error = %v, want function vs result struct origins", genErr)
		}
		if firstErr == "" {
			firstErr = genErr.Error()
		} else if genErr.Error() != firstErr {
			t.Fatalf("v1alpha collision diagnostic is not deterministic:\nfirst: %s\nlater: %s", firstErr, genErr.Error())
		}
	}
	if _, planErr := BuildQueryCodegenPlan(config, dir); planErr == nil || planErr.Error() != firstErr {
		t.Fatalf("BuildQueryCodegenPlan() error mismatch: %v want %s", planErr, firstErr)
	}
}

func TestGenerateQueryCodeValidPackagesCompile(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "schema.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX)
) PRIMARY KEY (SingerId);
`)
	writeTestFile(t, filepath.Join(dir, "bigquery.sql"), `
CREATE TABLE mydataset.events (
  event_id INT64
);
`)
	code, err := GenerateQueryCode(QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "schema.sql"},
			{Name: "warehouse", Dialect: "bigquery", DDL: "bigquery.sql"},
			{Name: "bqconn", Dialect: "bigquery", ExternalQueryConnections: []QueryCodegenExternalSchema{{
				Connection:    "example-project.us.example-connection",
				SpannerSource: "spanner",
			}}},
		},
		Queries: []QueryCodegenQuery{
			{Name: "ListSingers", Catalog: "spanner", SQL: "SELECT SingerId, FirstName FROM Singers", Result: "many", ResultStruct: "SingerRow"},
			{Name: "GetEvent", Catalog: "warehouse", SQL: "SELECT event_id FROM mydataset.events", Result: "one", ResultStruct: "EventRow"},
			{Name: "ListSingersOptional", Catalog: "spanner", SQL: "SELECT SingerId FROM Singers WHERE TRUE /*?optional:name*/ AND FirstName = @name /*?end*/", Params: []QueryCodegenParam{{Name: "name", Type: "STRING", Optional: "omit_when_null"}}, ResultStruct: "SingerIDRow"},
			{Name: "ExternalSingerIDs", Catalog: "bqconn", Federated: QueryCodegenFederatedQuery{Connection: "example-project.us.example-connection", SpannerSource: "spanner", InnerSQL: "SELECT SingerId FROM Singers"}, Result: "many", ResultStruct: "ExternalRow"},
		},
		Writes: []QueryCodegenWrite{{
			Name:    "SaveSinger",
			Catalog: "spanner",
			Table:   "Singers",
			Methods: []string{"mutation", "dml"},
		}},
	}, dir)
	if err != nil {
		t.Fatalf("GenerateQueryCode() error = %v", err)
	}
	compileGeneratedPackage(t, code)
}

func compileGeneratedPackage(t *testing.T, code string) {
	t.Helper()
	genDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(genDir, "bigquerystub"), 0o755); err != nil {
		t.Fatalf("mkdir bigquerystub: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(genDir, "spannerstub"), 0o755); err != nil {
		t.Fatalf("mkdir spannerstub: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(genDir, "googleapi", "iterator"), 0o755); err != nil {
		t.Fatalf("mkdir iterator: %v", err)
	}
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "go.mod"), `module generatednamespacetest

go 1.22

require (
	cloud.google.com/go/bigquery v0.0.0
	cloud.google.com/go/spanner v0.0.0
	google.golang.org/api v0.0.0
)

replace cloud.google.com/go/bigquery => ./bigquerystub

replace cloud.google.com/go/spanner => ./spannerstub

replace google.golang.org/api => ./googleapi
`)
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "generated.go"), code)
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "bigquerystub", "go.mod"), "module cloud.google.com/go/bigquery\n\ngo 1.22\n")
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "bigquerystub", "bigquery.go"), `package bigquery

import "context"

type Value interface{}
type NullInt64 struct{ Int64 int64; Valid bool }
type QueryParameter struct{ Name string; Value interface{} }
type Client struct{}
func (c *Client) Query(sql string) *Query { return &Query{} }
type Query struct{ Parameters []QueryParameter }
func (q *Query) Read(ctx context.Context) (*RowIterator, error) { return &RowIterator{}, nil }
type RowIterator struct{}
func (it *RowIterator) Next(dst interface{}) error { return nil }
type Schema []*FieldSchema
type FieldSchema struct{ Name string; Schema Schema }
`)
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "spannerstub", "go.mod"), "module cloud.google.com/go/spanner\n\ngo 1.22\n")
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "spannerstub", "spanner.go"), `package spanner

import "context"

type RowIterator struct{}
func (it *RowIterator) Next() (*Row, error) { return nil, nil }
func (it *RowIterator) Stop() {}
type Row struct{}
func (r *Row) ToStruct(dst interface{}) error { return nil }
type Statement struct {
	SQL    string
	Params map[string]interface{}
}
type Mutation struct{}
type NullString struct{ StringVal string; Valid bool }
func Insert(table string, cols []string, vals []interface{}) *Mutation { return &Mutation{} }
func InsertOrUpdate(table string, cols []string, vals []interface{}) *Mutation { return &Mutation{} }
type ReadWriteTransaction struct{}
func (tx *ReadWriteTransaction) Query(ctx context.Context, stmt Statement) *RowIterator { return &RowIterator{} }
func (tx *ReadWriteTransaction) Update(ctx context.Context, stmt Statement) (int64, error) { return 0, nil }
`)
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "googleapi", "go.mod"), "module google.golang.org/api\n\ngo 1.22\n")
	writeGeneratedLoadTestFile(t, filepath.Join(genDir, "googleapi", "iterator", "iterator.go"), `package iterator

import "errors"

var Done = errors.New("no more items in iterator")
`)
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = genDir
	cmd.Env = append(os.Environ(), "GOCACHE="+filepath.Join(genDir, "gocache"), "GOTOOLCHAIN=local", "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test generated package: %v\n%s\n--- generated ---\n%s", err, output, code)
	}
}
