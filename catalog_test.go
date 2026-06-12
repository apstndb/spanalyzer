package spanalyzer

import (
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

func TestBuildSchemaCatalogCreateAndAlterTable(t *testing.T) {
	const ddl = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(128),
  LastName STRING(MAX),
  Active BOOL,
  Tags ARRAY<STRING(MAX)>,
) PRIMARY KEY (SingerId);

ALTER TABLE Singers ADD COLUMN UpdatedAt TIMESTAMP;
ALTER TABLE Singers ALTER COLUMN FirstName STRING(256);
`
	catalog, err := BuildSchemaCatalog("schema.sql", ddl)
	if err != nil {
		t.Fatalf("BuildSchemaCatalog() error = %v", err)
	}
	table := catalog.Tables["Singers"]
	if table == nil {
		t.Fatalf("Singers table was not created")
	}
	if got, want := len(table.Columns), 6; got != want {
		t.Fatalf("len(table.Columns) = %d, want %d", got, want)
	}
	firstName, _ := table.Column("FirstName")
	if firstName == nil {
		t.Fatalf("FirstName column not found")
	}
	if firstName.Type.Code != spannerpb.TypeCode_STRING {
		t.Fatalf("FirstName type = %s, want STRING", firstName.Type.Code)
	}
	if firstName.Type.Length == nil || *firstName.Type.Length != 256 {
		t.Fatalf("FirstName length = %v, want 256", firstName.Type.Length)
	}
	lastName, _ := table.Column("LastName")
	if lastName == nil || !lastName.Type.Max {
		t.Fatalf("LastName MAX = %v, want true", lastName != nil && lastName.Type.Max)
	}
	tags, _ := table.Column("Tags")
	if tags == nil || tags.Type.Code != spannerpb.TypeCode_ARRAY || tags.Type.ArrayElement.Code != spannerpb.TypeCode_STRING {
		t.Fatalf("Tags type = %#v, want ARRAY<STRING>", tags)
	}
	if len(table.PrimaryKey) != 1 || table.PrimaryKey[0].Name != "SingerId" {
		t.Fatalf("primary key = %#v, want SingerId", table.PrimaryKey)
	}
}

func TestBuildSchemaCatalogTableSynonymAndRename(t *testing.T) {
	const ddl = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  SYNONYM (SingerAlias)
) PRIMARY KEY (SingerId);
ALTER TABLE Singers DROP SYNONYM SingerAlias;
ALTER TABLE Singers ADD SYNONYM SingerReadName;
ALTER TABLE Singers RENAME TO Artists, ADD SYNONYM Singers;
RENAME TABLE Artists TO Performers;
`
	catalog, err := BuildSchemaCatalog("schema.sql", ddl)
	if err != nil {
		t.Fatalf("BuildSchemaCatalog() error = %v", err)
	}
	if catalog.Tables["Singers"] != nil || catalog.Tables["Artists"] != nil {
		t.Fatalf("old table names should not remain: %#v", catalog.Tables)
	}
	table := catalog.Tables["Performers"]
	if table == nil {
		t.Fatalf("Performers table was not created")
	}
	if got, want := table.Name.String(), "Performers"; got != want {
		t.Fatalf("table.Name = %q, want %q", got, want)
	}
	if got, want := table.Synonyms, []string{"Singers"}; !sameStrings(got, want) {
		t.Fatalf("table.Synonyms = %#v, want %#v", got, want)
	}
}

func TestBuildSchemaCatalogIndexes(t *testing.T) {
	const ddl = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(1024),
  LastName STRING(1024),
) PRIMARY KEY (SingerId);
CREATE NULL_FILTERED INDEX SingersByFirstLastName ON Singers(FirstName, LastName);
`
	catalog, err := BuildSchemaCatalog("schema.sql", ddl)
	if err != nil {
		t.Fatalf("BuildSchemaCatalog() error = %v", err)
	}
	if catalog.Tables["Singers"] == nil {
		t.Fatalf("Singers table was not created")
	}
	index := catalog.Indexes["SingersByFirstLastName"]
	if index == nil {
		t.Fatalf("SingersByFirstLastName index was not created")
	}
	if !index.NullFiltered {
		t.Fatalf("index.NullFiltered = false, want true")
	}
}

func TestBuildSchemaCatalogSequences(t *testing.T) {
	const ddl = `
CREATE SEQUENCE Seq OPTIONS (sequence_kind = 'bit_reversed_positive');
ALTER SEQUENCE Seq SET OPTIONS (skip_range_min = 1, skip_range_max = 10);
CREATE SEQUENCE IF NOT EXISTS Seq OPTIONS (sequence_kind = 'bit_reversed_positive');
DROP SEQUENCE Seq;
`
	catalog, err := BuildSchemaCatalog("schema.sql", ddl)
	if err != nil {
		t.Fatalf("BuildSchemaCatalog() error = %v", err)
	}
	if len(catalog.Sequences) != 0 {
		t.Fatalf("sequences = %#v, want empty", catalog.Sequences)
	}
}

func TestBuildSchemaCatalogCreateModelEmulatorCompatibility(t *testing.T) {
	tests := []struct {
		name       string
		ddl        string
		wantModels []string
		wantErr    bool
		check      func(t *testing.T, catalog *Catalog)
	}{
		{
			name: "basic",
			ddl: `
CREATE MODEL m
INPUT (feature INT64)
OUTPUT (label STRING(MAX))
REMOTE OPTIONS (
  endpoint = '//aiplatform.googleapis.com/projects/aaa/locations/bbb/endpoints/ccc'
);
`,
			wantModels: []string{"m"},
			check: func(t *testing.T, catalog *Catalog) {
				model := catalog.Models["m"]
				if got, want := model.Inputs[0].Name, "feature"; got != want {
					t.Fatalf("input name = %q, want %q", got, want)
				}
				if got, want := model.Inputs[0].Type.Code, spannerpb.TypeCode_INT64; got != want {
					t.Fatalf("input type = %s, want %s", got, want)
				}
				if got, want := model.Outputs[0].Name, "label"; got != want {
					t.Fatalf("output name = %q, want %q", got, want)
				}
				if got, want := model.Outputs[0].Type.Code, spannerpb.TypeCode_STRING; got != want {
					t.Fatalf("output type = %s, want %s", got, want)
				}
			},
		},
		{
			name: "array and struct columns",
			ddl: `
CREATE MODEL m
INPUT (feature ARRAY<INT64>)
OUTPUT (label STRUCT<l STRING>)
REMOTE OPTIONS (
  endpoints = [
    '//aiplatform.googleapis.com/projects/aaa/locations/bbb/endpoints/ccc',
    '//aiplatform.googleapis.com/projects/aaa/locations/ddd/endpoints/eee'
  ],
  default_batch_size = 10
);
`,
			wantModels: []string{"m"},
			check: func(t *testing.T, catalog *Catalog) {
				model := catalog.Models["m"]
				if got, want := model.Inputs[0].Type.Code, spannerpb.TypeCode_ARRAY; got != want {
					t.Fatalf("input type = %s, want %s", got, want)
				}
				if got, want := model.Inputs[0].Type.ArrayElement.Code, spannerpb.TypeCode_INT64; got != want {
					t.Fatalf("input array element = %s, want %s", got, want)
				}
				if got, want := model.Outputs[0].Type.Code, spannerpb.TypeCode_STRUCT; got != want {
					t.Fatalf("output type = %s, want %s", got, want)
				}
				if got, want := model.Outputs[0].Type.StructFields[0].Name, "l"; got != want {
					t.Fatalf("output struct field = %q, want %q", got, want)
				}
			},
		},
		{
			name: "or replace",
			ddl: `
CREATE MODEL m
INPUT (feature INT64)
OUTPUT (label STRING(MAX))
REMOTE OPTIONS (endpoint = '//aiplatform.googleapis.com/projects/aaa/locations/bbb/endpoints/ccc');
CREATE OR REPLACE MODEL m
INPUT (f INT64)
OUTPUT (l STRING(MAX))
REMOTE OPTIONS (endpoint = '//aiplatform.googleapis.com/projects/aaa/locations/bbb/endpoints/ccc');
`,
			wantModels: []string{"m"},
			check: func(t *testing.T, catalog *Catalog) {
				model := catalog.Models["m"]
				if got, want := model.Inputs[0].Name, "f"; got != want {
					t.Fatalf("input name = %q, want %q", got, want)
				}
				if got, want := model.Outputs[0].Name, "l"; got != want {
					t.Fatalf("output name = %q, want %q", got, want)
				}
			},
		},
		{
			name: "if not exists emulator syntax unsupported by memefish",
			ddl: `
CREATE MODEL m
INPUT (feature INT64)
OUTPUT (label STRING(MAX))
REMOTE OPTIONS (endpoint = '//aiplatform.googleapis.com/projects/aaa/locations/bbb/endpoints/ccc');
CREATE MODEL IF NOT EXISTS m
INPUT (f INT64)
OUTPUT (l STRING(MAX))
REMOTE OPTIONS (endpoint = '//aiplatform.googleapis.com/projects/aaa/locations/bbb/endpoints/ccc');
`,
			wantErr: true,
		},
		{
			name: "drop and recreate",
			ddl: `
CREATE MODEL m
INPUT (feature INT64)
OUTPUT (label STRING(MAX))
REMOTE OPTIONS (endpoint = '//aiplatform.googleapis.com/projects/aaa/locations/bbb/endpoints/ccc');
DROP MODEL m;
CREATE MODEL m
INPUT (f INT64)
OUTPUT (l STRING(MAX))
REMOTE OPTIONS (endpoint = '//aiplatform.googleapis.com/projects/aaa/locations/bbb/endpoints/ccc');
`,
			wantModels: []string{"m"},
			check: func(t *testing.T, catalog *Catalog) {
				model := catalog.Models["m"]
				if got, want := model.Inputs[0].Name, "f"; got != want {
					t.Fatalf("input name = %q, want %q", got, want)
				}
				if got, want := model.Outputs[0].Name, "l"; got != want {
					t.Fatalf("output name = %q, want %q", got, want)
				}
			},
		},
		{
			name: "multiple statements",
			ddl: `
CREATE MODEL m1
INPUT (feature INT64)
OUTPUT (label STRING(MAX))
REMOTE OPTIONS (endpoint = '//aiplatform.googleapis.com/projects/p1/locations/l1/endpoints/e1');
CREATE MODEL m2
INPUT (feature INT64)
OUTPUT (label STRING(MAX))
REMOTE OPTIONS (endpoint = '//aiplatform.googleapis.com/projects/p2/locations/l2/endpoints/e2');
CREATE MODEL m3
INPUT (feature INT64)
OUTPUT (label STRING(MAX))
REMOTE OPTIONS (endpoint = '//aiplatform.googleapis.com/projects/p3/locations/l3/endpoints/e3');
DROP MODEL m1;
ALTER MODEL m2 SET OPTIONS (
  endpoint = '//aiplatform.googleapis.com/projects/p2/locations/l2/endpoints/e2'
);
`,
			wantModels: []string{"m2", "m3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog, err := BuildSchemaCatalog("schema.sql", tt.ddl)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("BuildSchemaCatalog() succeeded, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("BuildSchemaCatalog() error = %v", err)
			}
			if got, want := len(catalog.Models), len(tt.wantModels); got != want {
				t.Fatalf("len(catalog.Models) = %d, want %d; models = %#v", got, want, catalog.Models)
			}
			for _, name := range tt.wantModels {
				if catalog.Models[name] == nil {
					t.Fatalf("model %s was not created", name)
				}
			}
			if tt.check != nil {
				tt.check(t, catalog)
			}
		})
	}
}

func TestBuildSchemaCatalogDropColumnTableAndView(t *testing.T) {
	const ddl = `
CREATE TABLE Albums (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  Title STRING(MAX),
) PRIMARY KEY (SingerId, AlbumId);
ALTER TABLE Albums DROP COLUMN Title;
CREATE VIEW AlbumIds SQL SECURITY INVOKER AS SELECT SingerId, AlbumId FROM Albums;
DROP VIEW AlbumIds;
DROP TABLE Albums;
`
	catalog, err := BuildSchemaCatalog("schema.sql", ddl)
	if err != nil {
		t.Fatalf("BuildSchemaCatalog() error = %v", err)
	}
	if got := userTableCount(catalog); got != 0 {
		t.Fatalf("user table count = %d, want 0; tables = %#v", got, catalog.Tables)
	}
	if len(catalog.Views) != 0 {
		t.Fatalf("views = %#v, want empty", catalog.Views)
	}
}

func userTableCount(catalog *Catalog) int {
	count := 0
	for name := range catalog.Tables {
		if strings.HasPrefix(name, informationSchemaName+".") || strings.HasPrefix(name, spannerSysName+".") {
			continue
		}
		count++
	}
	return count
}

func TestBuildSchemaCatalogCreateOrReplaceView(t *testing.T) {
	const ddl = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName STRING(MAX),
) PRIMARY KEY (SingerId);
CREATE VIEW SingerNames SQL SECURITY INVOKER AS SELECT FirstName FROM Singers;
CREATE OR REPLACE VIEW SingerNames SQL SECURITY INVOKER AS SELECT LastName FROM Singers;
`
	catalog, err := BuildSchemaCatalog("schema.sql", ddl)
	if err != nil {
		t.Fatalf("BuildSchemaCatalog() error = %v", err)
	}
	if got, want := len(catalog.ViewOrder), 1; got != want {
		t.Fatalf("len(catalog.ViewOrder) = %d, want %d", got, want)
	}
	view := catalog.Views["SingerNames"]
	if view == nil {
		t.Fatalf("SingerNames view not found")
	}
	if got, want := view.Query, "SELECT LastName FROM Singers"; got != want {
		t.Fatalf("view.Query = %q, want %q", got, want)
	}
}

func TestBuildSchemaCatalogCreateOrReplacePropertyGraph(t *testing.T) {
	const ddl = `
CREATE TABLE Account (
  id INT64 NOT NULL,
  name STRING(MAX),
) PRIMARY KEY (id);
CREATE TABLE Person (
  id INT64 NOT NULL,
  name STRING(MAX),
) PRIMARY KEY (id);
CREATE TABLE PersonOwnAccount (
  id INT64 NOT NULL,
  account_id INT64 NOT NULL,
) PRIMARY KEY (id, account_id);

CREATE PROPERTY GRAPH IF NOT EXISTS FinGraph
  NODE TABLES (
    Account,
    Person
  )
  EDGE TABLES (
    PersonOwnAccount
      SOURCE KEY (id) REFERENCES Person (id)
      DESTINATION KEY (account_id) REFERENCES Account (id)
      LABEL Owns
  );

CREATE OR REPLACE PROPERTY GRAPH FinGraph
  NODE TABLES (
    Account LABEL AccountNode PROPERTIES ARE ALL COLUMNS,
    Person AS People KEY (id) LABEL PersonNode PROPERTIES ARE ALL COLUMNS EXCEPT (id)
  );
`
	catalog, err := BuildSchemaCatalog("schema.sql", ddl)
	if err != nil {
		t.Fatalf("BuildSchemaCatalog() error = %v", err)
	}
	graph := catalog.PropertyGraphs["FinGraph"]
	if graph == nil {
		t.Fatalf("FinGraph property graph not found")
	}
	if got, want := len(graph.NodeTables), 2; got != want {
		t.Fatalf("len(graph.NodeTables) = %d, want %d", got, want)
	}
	if got, want := len(graph.EdgeTables), 0; got != want {
		t.Fatalf("len(graph.EdgeTables) = %d, want %d", got, want)
	}
	account := graph.NodeTables[0]
	if got, want := account.Labels[0].Name, "AccountNode"; got != want {
		t.Fatalf("Account label = %q, want %q", got, want)
	}
	people := graph.NodeTables[1]
	if got, want := people.Alias, "People"; got != want {
		t.Fatalf("Person alias = %q, want %q", got, want)
	}
	if got, want := people.KeyColumns, []string{"id"}; !sameStrings(got, want) {
		t.Fatalf("Person key columns = %#v, want %#v", got, want)
	}
	if got, want := people.Labels[0].PropertiesSQL, "PROPERTIES ARE ALL COLUMNS EXCEPT (id)"; got != want {
		t.Fatalf("Person properties = %q, want %q", got, want)
	}
}

func TestBuildSchemaCatalogDropPropertyGraph(t *testing.T) {
	const ddl = `
CREATE TABLE Account (
  id INT64 NOT NULL,
) PRIMARY KEY (id);
CREATE PROPERTY GRAPH FinGraph NODE TABLES (Account);
DROP PROPERTY GRAPH FinGraph;
`
	catalog, err := BuildSchemaCatalog("schema.sql", ddl)
	if err != nil {
		t.Fatalf("BuildSchemaCatalog() error = %v", err)
	}
	if len(catalog.PropertyGraphs) != 0 {
		t.Fatalf("property graphs = %#v, want empty", catalog.PropertyGraphs)
	}
}

func TestBuildSchemaCatalogProtoBundle(t *testing.T) {
	const ddl = `
CREATE PROTO BUNDLE (
  ` + "`examples.shipping.Order`" + `,
  ` + "`examples.shipping.Order.Address`" + `
);
ALTER PROTO BUNDLE INSERT (` + "`examples.shipping.Order.Item`" + `);
ALTER PROTO BUNDLE DELETE (` + "`examples.shipping.Order.Address`" + `);
`
	catalog, err := BuildSchemaCatalog("schema.sql", ddl)
	if err != nil {
		t.Fatalf("BuildSchemaCatalog() error = %v", err)
	}
	if !catalog.ProtoTypes["examples.shipping.Order"] {
		t.Fatalf("Order proto type not found")
	}
	if !catalog.ProtoTypes["examples.shipping.Order.Item"] {
		t.Fatalf("Order.Item proto type not found")
	}
	if catalog.ProtoTypes["examples.shipping.Order.Address"] {
		t.Fatalf("Order.Address proto type was not deleted")
	}
}

func TestBuildSchemaCatalogDropProtoBundle(t *testing.T) {
	const ddl = `
CREATE PROTO BUNDLE (` + "`examples.shipping.Order`" + `);
DROP PROTO BUNDLE;
`
	catalog, err := BuildSchemaCatalog("schema.sql", ddl)
	if err != nil {
		t.Fatalf("BuildSchemaCatalog() error = %v", err)
	}
	if len(catalog.ProtoTypes) != 0 {
		t.Fatalf("proto types = %#v, want empty", catalog.ProtoTypes)
	}
}

func TestTypeSpecSpannerPB(t *testing.T) {
	spec := &TypeSpec{
		Code: spannerpb.TypeCode_ARRAY,
		ArrayElement: &TypeSpec{
			Code: spannerpb.TypeCode_STRUCT,
			StructFields: []StructField{
				{Name: "Name", Type: &TypeSpec{Code: spannerpb.TypeCode_STRING}},
			},
		},
	}
	got, err := spec.SpannerPB()
	if err != nil {
		t.Fatalf("SpannerPB() error = %v", err)
	}
	if got.Code != spannerpb.TypeCode_ARRAY {
		t.Fatalf("Code = %s, want ARRAY", got.Code)
	}
	if got.ArrayElementType.GetStructType().GetFields()[0].Name != "Name" {
		t.Fatalf("field name = %q, want Name", got.ArrayElementType.GetStructType().GetFields()[0].Name)
	}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestBuildSchemaCatalog_IgnoredObjects(t *testing.T) {
	const ddl = `
CREATE CHANGE STREAM MyChangeStream FOR ALL;
ALTER DATABASE MyDatabase SET OPTIONS (
  version_retention_period = '7d'
);
CREATE ROLE MyRole;
GRANT SELECT ON TABLE MyTable TO ROLE MyRole;
CREATE TABLE SearchAlbums (
  AlbumId STRING(MAX) NOT NULL,
  AlbumTitle STRING(MAX),
  AlbumTitle_Tokens TOKENLIST AS (TOKENIZE_FULLTEXT(AlbumTitle)) HIDDEN
) PRIMARY KEY(AlbumId);
CREATE SEARCH INDEX SearchAlbumsTitleIndex ON SearchAlbums(AlbumTitle_Tokens);
ALTER SEARCH INDEX SearchAlbumsTitleIndex ADD STORED COLUMN AlbumTitle;
DROP SEARCH INDEX SearchAlbumsTitleIndex;
`
	// BuildSchemaCatalog should not fail on these, even if they are not fully modeled
	// because memefish supports them.
	// Note: memefish might not support all of these, let's verify.
	_, err := BuildSchemaCatalog("ignored.sql", ddl)
	if err != nil {
		t.Fatalf("BuildSchemaCatalog() error = %v", err)
	}
}
