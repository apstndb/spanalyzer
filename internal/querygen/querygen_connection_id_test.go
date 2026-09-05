package querygen

import (
	"path/filepath"
	"strings"
	"testing"
)

const duplicateExternalQueryConnectionID = "example-project.us.dup-connection"

func TestGenerateQueryCodeRejectsDuplicateExternalQueryConnectionIDs(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "int.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	writeTestFile(t, filepath.Join(dir, "string.sql"), `
CREATE TABLE Singers (
  SingerId STRING(MAX) NOT NULL
) PRIMARY KEY (SingerId);
`)
	intSchema := QueryCodegenSchema{Name: "singers_int", Dialect: "spanner", DDL: "int.sql"}
	stringSchema := QueryCodegenSchema{Name: "singers_string", Dialect: "spanner", DDL: "string.sql"}
	id := duplicateExternalQueryConnectionID
	tests := []struct {
		name    string
		config  QueryCodegenConfig
		origins []string
		sources []string
	}{
		{
			name: "legacy-only ExternalSchemas",
			config: duplicateConnectionConfig(intSchema, stringSchema, QueryCodegenSchema{
				Name:    "bigquery",
				Dialect: "bigquery",
				ExternalSchemas: []QueryCodegenExternalSchema{
					{Connection: id, Schema: "singers_int"},
					{Connection: id, Schema: "singers_string"},
				},
			}, federatedQuery("bigquery", id, "singers_int")),
			origins: []string{"schemas[2].external_schemas[0]", "schemas[2].external_schemas[1]"},
			sources: []string{"singers_int", "singers_string"},
		},
		{
			name: "current-only ExternalQueryConnections",
			config: duplicateConnectionConfig(intSchema, stringSchema, QueryCodegenSchema{
				Name:    "bigquery",
				Dialect: "bigquery",
				ExternalQueryConnections: []QueryCodegenExternalSchema{
					{Connection: id, SpannerSource: "singers_int"},
					{Connection: id, SpannerSource: "singers_string"},
				},
			}, federatedQuery("bigquery", id, "singers_int")),
			origins: []string{"schemas[2].external_query_connections[0]", "schemas[2].external_query_connections[1]"},
			sources: []string{"singers_int", "singers_string"},
		},
		{
			name: "combined legacy and current lists",
			config: duplicateConnectionConfig(intSchema, stringSchema, QueryCodegenSchema{
				Name:    "bigquery",
				Dialect: "bigquery",
				ExternalSchemas: []QueryCodegenExternalSchema{
					{Connection: id, Schema: "singers_int"},
				},
				ExternalQueryConnections: []QueryCodegenExternalSchema{
					{Connection: id, SpannerSource: "singers_string"},
				},
			}, federatedQuery("bigquery", id, "singers_int")),
			origins: []string{"schemas[2].external_schemas[0]", "schemas[2].external_query_connections[0]"},
			sources: []string{"singers_int", "singers_string"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertDuplicateConnectionIDError(t, tt.config, dir, id, tt.origins, tt.sources)
		})
	}
}

func TestGenerateQueryCodeDuplicateConnectionIDAsymmetricQueriesFailIdentically(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "int.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	writeTestFile(t, filepath.Join(dir, "string.sql"), `
CREATE TABLE Singers (
  SingerId STRING(MAX) NOT NULL
) PRIMARY KEY (SingerId);
`)
	id := duplicateExternalQueryConnectionID
	base := QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetSpanner,
		Schemas: []QueryCodegenSchema{
			{Name: "singers_int", Dialect: "spanner", DDL: "int.sql"},
			{Name: "singers_string", Dialect: "spanner", DDL: "string.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				ExternalQueryConnections: []QueryCodegenExternalSchema{
					{Connection: id, SpannerSource: "singers_int"},
					{Connection: id, SpannerSource: "singers_string"},
				},
			},
		},
	}
	first := base
	first.Queries = []QueryCodegenQuery{federatedQuery("bigquery", id, "singers_int")}
	second := base
	second.Queries = []QueryCodegenQuery{federatedQuery("bigquery", id, "singers_string")}

	firstErr := assertDuplicateConnectionIDError(t, first, dir, id,
		[]string{"schemas[2].external_query_connections[0]", "schemas[2].external_query_connections[1]"},
		[]string{"singers_int", "singers_string"})
	secondErr := assertDuplicateConnectionIDError(t, second, dir, id,
		[]string{"schemas[2].external_query_connections[0]", "schemas[2].external_query_connections[1]"},
		[]string{"singers_int", "singers_string"})
	if firstErr != secondErr {
		t.Fatalf("asymmetric queries returned different diagnostics:\nfirst: %s\nsecond: %s", firstErr, secondErr)
	}
}

func TestParseV1AlphaDuplicateExternalQueryConnectionIDs(t *testing.T) {
	_, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: singers_int
  kind: spanner
  ddl: int.sql
- name: singers_string
  kind: spanner
  ddl: string.sql
- name: analytics
  kind: bigquery
  bindings:
    external_query_connections:
    - name: conn_int
      id: example-project.us.dup-connection
      spanner_catalog: singers_int
    - name: conn_string
      id: example-project.us.dup-connection
      spanner_catalog: singers_string
queries:
- name: ExternalSingers
  catalog: analytics
  kind: external_query
  binding: conn_int
  inner_sql: SELECT SingerId FROM Singers
  result:
    struct: SingerRow
`))
	if err == nil {
		t.Fatal("ParseQueryCodegenConfigYAML() error = nil, want duplicate connection ID")
	}
	msg := err.Error()
	if !strings.Contains(msg, `duplicate BigQuery EXTERNAL_QUERY connection ID "example-project.us.dup-connection"`) {
		t.Fatalf("error = %v, want duplicate connection ID", err)
	}
	for _, want := range []string{
		"catalogs[2].bindings.external_query_connections[0] (conn_int)",
		"catalogs[2].bindings.external_query_connections[1] (conn_string)",
		"singers_int",
		"singers_string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error = %v, want %q", err, want)
		}
	}
}

func TestGenerateQueryCodeDuplicateConnectionIDWinsOverMissingDDL(t *testing.T) {
	dir := t.TempDir()
	id := duplicateExternalQueryConnectionID
	cfg := QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetSpanner,
		Schemas: []QueryCodegenSchema{
			{Name: "missing_a", Dialect: "spanner", DDL: "missing_a.sql"},
			{Name: "missing_b", Dialect: "spanner", DDL: "missing_b.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				ExternalQueryConnections: []QueryCodegenExternalSchema{
					{Connection: id, SpannerSource: "missing_a"},
					{Connection: id, SpannerSource: "missing_b"},
				},
			},
		},
		Queries: []QueryCodegenQuery{federatedQuery("bigquery", id, "missing_a")},
	}
	msg := assertDuplicateConnectionIDError(t, cfg, dir, id,
		[]string{"schemas[2].external_query_connections[0]", "schemas[2].external_query_connections[1]"},
		[]string{"missing_a", "missing_b"})
	if strings.Contains(msg, "missing_a.sql") || strings.Contains(msg, "no such file") || strings.Contains(msg, "external schema") {
		t.Fatalf("duplicate-ID error reached DDL/analyzer work: %s", msg)
	}
}

func TestGenerateQueryCodeAllowsSameConnectionIDInIndependentCatalogs(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "int.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	writeTestFile(t, filepath.Join(dir, "string.sql"), `
CREATE TABLE Singers (
  SingerId STRING(MAX) NOT NULL
) PRIMARY KEY (SingerId);
`)
	id := duplicateExternalQueryConnectionID
	cfg := QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{
			{Name: "singers_int", Dialect: "spanner", DDL: "int.sql"},
			{Name: "singers_string", Dialect: "spanner", DDL: "string.sql"},
			{
				Name:    "bq_int",
				Dialect: "bigquery",
				ExternalQueryConnections: []QueryCodegenExternalSchema{{
					Connection:    id,
					SpannerSource: "singers_int",
				}},
			},
			{
				Name:    "bq_string",
				Dialect: "bigquery",
				ExternalQueryConnections: []QueryCodegenExternalSchema{{
					Connection:    id,
					SpannerSource: "singers_string",
				}},
			},
		},
		Queries: []QueryCodegenQuery{
			federatedQuery("bq_int", id, "singers_int"),
			{
				Name:    "StringSingers",
				Catalog: "bq_string",
				Federated: QueryCodegenFederatedQuery{
					Connection:    id,
					SpannerSource: "singers_string",
					InnerSQL:      "SELECT SingerId FROM Singers",
				},
				ResultStruct: "StringSingerRow",
			},
		},
	}
	code, err := GenerateQueryCode(cfg, dir)
	if err != nil {
		t.Fatalf("independent catalogs GenerateQueryCode() error = %v", err)
	}
	if !strings.Contains(code, "SingerId NullValue[int64]") || !strings.Contains(code, "SingerId NullValue[string]") {
		t.Fatalf("independent catalogs missing both mapped types:\n%s", code)
	}
	if _, err := BuildQueryCodegenPlan(cfg, dir); err != nil {
		t.Fatalf("independent catalogs BuildQueryCodegenPlan() error = %v", err)
	}
}

func TestGenerateQueryCodeMalformedExternalQueryConnectionIsNotDuplicate(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	id := duplicateExternalQueryConnectionID
	spanner := QueryCodegenSchema{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"}
	tests := []struct {
		name          string
		config        QueryCodegenConfig
		wantSubstring string
	}{
		{
			name:          "two current entries with duplicate ID and missing SpannerSource",
			wantSubstring: "require connection and spanner_source",
			config: QueryCodegenConfig{
				Package: "db",
				Client:  GoStructTargetBigQuery,
				Schemas: []QueryCodegenSchema{{
					Name:    "bigquery",
					Dialect: "bigquery",
					ExternalQueryConnections: []QueryCodegenExternalSchema{
						{Connection: id},
						{Connection: id},
					},
				}},
				Queries: []QueryCodegenQuery{{
					Name:    "ExternalSingers",
					Catalog: "bigquery",
					Federated: QueryCodegenFederatedQuery{
						Connection: id,
						InnerSQL:   "SELECT SingerId FROM Singers",
					},
					ResultStruct: "SingerRow",
				}},
			},
		},
		{
			name:          "two legacy entries with duplicate ID and missing Schema",
			wantSubstring: "require connection and spanner_source",
			config: QueryCodegenConfig{
				Package: "db",
				Client:  GoStructTargetBigQuery,
				Schemas: []QueryCodegenSchema{{
					Name:    "bigquery",
					Dialect: "bigquery",
					ExternalSchemas: []QueryCodegenExternalSchema{
						{Connection: id},
						{Connection: id},
					},
				}},
				Queries: []QueryCodegenQuery{{
					Name:    "ExternalSingers",
					Catalog: "bigquery",
					Federated: QueryCodegenFederatedQuery{
						Connection: id,
						InnerSQL:   "SELECT SingerId FROM Singers",
					},
					ResultStruct: "SingerRow",
				}},
			},
		},
		{
			name:          "valid mapping plus same-ID entry with missing source",
			wantSubstring: "require connection and spanner_source",
			config: QueryCodegenConfig{
				Package: "db",
				Client:  GoStructTargetSpanner,
				Schemas: []QueryCodegenSchema{
					spanner,
					{
						Name:    "bigquery",
						Dialect: "bigquery",
						ExternalQueryConnections: []QueryCodegenExternalSchema{
							{Connection: id, SpannerSource: "spanner"},
							{Connection: id},
						},
					},
				},
				Queries: []QueryCodegenQuery{federatedQuery("bigquery", id, "spanner")},
			},
		},
		{
			name:          "empty connection IDs",
			wantSubstring: "no external query connection entry",
			config: QueryCodegenConfig{
				Package: "db",
				Client:  GoStructTargetSpanner,
				Schemas: []QueryCodegenSchema{
					spanner,
					{
						Name:    "bigquery",
						Dialect: "bigquery",
						ExternalQueryConnections: []QueryCodegenExternalSchema{
							{Connection: "", SpannerSource: "spanner"},
							{Connection: "", SpannerSource: "spanner"},
						},
					},
				},
				Queries: []QueryCodegenQuery{federatedQuery("bigquery", "example-project.us.example-connection", "spanner")},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GenerateQueryCode(tt.config, dir)
			if err == nil {
				t.Fatal("GenerateQueryCode() error = nil, want existing required-field diagnostic")
			}
			if strings.Contains(err.Error(), "duplicate BigQuery EXTERNAL_QUERY connection ID") {
				t.Fatalf("malformed required field misclassified as duplicate: %v", err)
			}
			if !strings.Contains(err.Error(), tt.wantSubstring) {
				t.Fatalf("error = %v, want %q", err, tt.wantSubstring)
			}
		})
	}
}

func TestGenerateQueryCodeUnambiguousLegacyAndV1AlphaExternalQueryRemainValid(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "spanner.sql"), `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL
) PRIMARY KEY (SingerId);
`)
	legacy := QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetBoth,
		Schemas: []QueryCodegenSchema{
			{Name: "spanner", Dialect: "spanner", DDL: "spanner.sql"},
			{
				Name:    "bigquery",
				Dialect: "bigquery",
				ExternalSchemas: []QueryCodegenExternalSchema{{
					Connection: "example-project.us.example-connection",
					Schema:     "spanner",
				}},
			},
		},
		Queries: []QueryCodegenQuery{federatedQuery("bigquery", "example-project.us.example-connection", "spanner")},
	}
	if _, err := GenerateQueryCode(legacy, dir); err != nil {
		t.Fatalf("unambiguous legacy GenerateQueryCode() error = %v", err)
	}

	config, err := ParseQueryCodegenConfigYAML([]byte(`
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
  ddl: spanner.sql
- name: analytics
  kind: bigquery
  bindings:
    external_query_connections:
    - name: app_conn
      id: example-project.us.example-connection
      spanner_catalog: app
queries:
- name: ExternalSingerIDs
  catalog: analytics
  kind: external_query
  binding: app_conn
  inner_sql: SELECT SingerId FROM Singers
  result:
    struct: SingerRow
`))
	if err != nil {
		t.Fatalf("unambiguous v1alpha ParseQueryCodegenConfigYAML() error = %v", err)
	}
	if _, err := GenerateQueryCode(config, dir); err != nil {
		t.Fatalf("unambiguous v1alpha GenerateQueryCode() error = %v", err)
	}
}

func duplicateConnectionConfig(intSchema, stringSchema, bq QueryCodegenSchema, query QueryCodegenQuery) QueryCodegenConfig {
	return QueryCodegenConfig{
		Package: "db",
		Client:  GoStructTargetSpanner,
		Schemas: []QueryCodegenSchema{intSchema, stringSchema, bq},
		Queries: []QueryCodegenQuery{query},
	}
}

func federatedQuery(catalog, connection, spannerSource string) QueryCodegenQuery {
	return QueryCodegenQuery{
		Name:    "ExternalSingers",
		Catalog: catalog,
		Federated: QueryCodegenFederatedQuery{
			Connection:    connection,
			SpannerSource: spannerSource,
			InnerSQL:      "SELECT SingerId FROM Singers",
		},
		ResultStruct: "SingerRow",
	}
}

func assertDuplicateConnectionIDError(t *testing.T, cfg QueryCodegenConfig, dir, id string, origins, sources []string) string {
	t.Helper()
	var firstErr string
	for i := 0; i < 3; i++ {
		code, err := GenerateQueryCode(cfg, dir)
		if err == nil {
			t.Fatalf("GenerateQueryCode() error = nil, want duplicate connection ID\n%s", code)
		}
		if code != "" {
			t.Fatalf("GenerateQueryCode() returned source on error:\n%s", code)
		}
		msg := err.Error()
		if !strings.Contains(msg, `duplicate BigQuery EXTERNAL_QUERY connection ID "`+id+`"`) {
			t.Fatalf("error = %v, want duplicate connection ID %s", err, id)
		}
		for _, origin := range origins {
			if !strings.Contains(msg, origin) {
				t.Fatalf("error = %v, want origin %q", err, origin)
			}
		}
		for _, source := range sources {
			if !strings.Contains(msg, source) {
				t.Fatalf("error = %v, want Spanner catalog %q", err, source)
			}
		}
		if firstErr == "" {
			firstErr = msg
		} else if msg != firstErr {
			t.Fatalf("duplicate-ID diagnostic is not deterministic:\nfirst: %s\nlater: %s", firstErr, msg)
		}
	}
	plan, planErr := BuildQueryCodegenPlan(cfg, dir)
	if planErr == nil {
		t.Fatalf("BuildQueryCodegenPlan() error = nil, want duplicate connection ID: %+v", plan)
	}
	if planErr.Error() != firstErr {
		t.Fatalf("BuildQueryCodegenPlan() error = %v, want %s", planErr, firstErr)
	}
	return firstErr
}
