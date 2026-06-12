//go:build integration

package main

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	spanalyzer "github.com/apstndb/spanalyzer"
	"github.com/apstndb/spanalyzer/internal/querygen"
	"github.com/apstndb/spanemuboost"
	"github.com/cloudspannerecosystem/memefish"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
)

var querygenIntegrationRuntime = spanemuboost.NewLazyRuntime(
	spanemuboost.BackendEmulator,
	spanemuboost.EnableInstanceAutoConfigOnly(),
	spanemuboost.WithContainerCustomizers(testcontainers.WithConfigModifier(func(config *dcontainer.Config) {
		config.Cmd = []string{"./gateway_main", "--hostname", "0.0.0.0", "--disable_query_null_filtered_index_check"}
	})),
)

func TestMain(m *testing.M) {
	querygenIntegrationRuntime.TestMain(m)
}

const querygenIntegrationSchemaDDL = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName STRING(MAX),
  Rating INT64 NOT NULL,
  Active BOOL
) PRIMARY KEY (SingerId);

CREATE NULL_FILTERED INDEX SingersByFirstLastRating ON Singers(FirstName, LastName, Rating);

CREATE TABLE Albums (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  Title STRING(MAX)
) PRIMARY KEY (SingerId, AlbumId);
`

const querygenIntegrationConfigYAML = `
version: v1alpha
go:
  package: db
catalogs:
- name: app
  kind: spanner
  ddl: schema.sql
queries:
- name: ListSingers
  catalog: app
  kind: table
  table: Singers
  result:
    struct: SingerRow
- name: FindSingersByFirstName
  catalog: app
  kind: index
  index: SingersByFirstLastRating
  key_prefix:
  - FirstName
  result:
    struct: SingerIndexRow
- name: ActiveSingers
  catalog: app
  kind: sql
  sql: SELECT SingerId FROM Singers WHERE Active = @active
  params:
  - name: active
    type: BOOL
  result:
    struct: SingerIDRow
`

func TestIntegrationQueryCodegenGeneratedSpannerQueriesRunOnEmulator(t *testing.T) {
	querygenIntegrationRequireContainerRuntime(t)

	plan, ddl := querygenIntegrationBuildPlan(t)
	clients := spanemuboost.SetupClients(t, querygenIntegrationRuntime,
		spanemuboost.WithRandomDatabaseID(),
		spanemuboost.WithSetupDDLs(querygenIntegrationDDLs(t, "schema.sql", ddl)),
	)
	querygenIntegrationExecutePlan(t, clients.Client, plan, "Spanner emulator")
}

func querygenIntegrationBuildPlan(t testing.TB) (*querygen.QueryCodegenPlan, string) {
	t.Helper()
	_, plan, _, ddl := querygenIntegrationBuildFixture(t)
	return plan, ddl
}

func querygenIntegrationBuildFixture(t testing.TB) (querygen.QueryCodegenConfig, *querygen.QueryCodegenPlan, string, string) {
	t.Helper()
	dir := t.TempDir()
	writeIntegrationTestFile(t, filepath.Join(dir, "schema.sql"), querygenIntegrationSchemaDDL)
	config, err := querygen.ParseQueryCodegenConfigYAML([]byte(querygenIntegrationConfigYAML))
	if err != nil {
		t.Fatalf("ParseQueryCodegenConfigYAML() error = %v", err)
	}
	plan, err := querygen.BuildQueryCodegenPlan(config, dir)
	if err != nil {
		t.Fatalf("BuildQueryCodegenPlan() error = %v", err)
	}
	return config, plan, dir, querygenIntegrationSchemaDDL
}

func querygenIntegrationExecutePlan(t *testing.T, client *spanner.Client, plan *querygen.QueryCodegenPlan, backendName string) {
	t.Helper()
	for _, query := range plan.Queries {
		query := query
		t.Run(query.Name, func(t *testing.T) {
			stmt := spanner.NewStatement(query.SQL)
			stmt.Params = querygenIntegrationParamValues(t, query.Params)
			err := client.Single().Query(t.Context(), stmt).Do(func(*spanner.Row) error {
				return nil
			})
			if err != nil {
				t.Fatalf("query failed on %s:\nSQL: %s\nparams: %#v\nerror: %v", backendName, query.SQL, stmt.Params, err)
			}
		})
	}
}

func writeIntegrationTestFile(t testing.TB, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}

func querygenIntegrationDDLs(t testing.TB, path, ddlSQL string) []string {
	t.Helper()
	ddls, err := memefish.ParseDDLs(path, ddlSQL)
	if err != nil {
		t.Fatalf("memefish.ParseDDLs() error = %v", err)
	}
	out := make([]string, 0, len(ddls))
	for _, ddl := range ddls {
		out = append(out, ddl.SQL())
	}
	return out
}

func querygenIntegrationParamValues(t testing.TB, params []querygen.QueryCodegenParam) map[string]interface{} {
	t.Helper()
	if len(params) == 0 {
		return nil
	}
	values := make(map[string]interface{}, len(params))
	for _, param := range params {
		spec, err := spanalyzer.ParseTypeSpec("param", param.Type)
		if err != nil {
			t.Fatalf("ParseTypeSpec(%q) error = %v", param.Type, err)
		}
		values[param.Name] = querygenIntegrationParamValue(t, spec, param.Type)
	}
	return values
}

func querygenIntegrationParamValue(t testing.TB, spec *spanalyzer.TypeSpec, typeSQL string) interface{} {
	t.Helper()
	if spec == nil {
		t.Fatalf("nil TypeSpec for %s", typeSQL)
	}
	switch spec.Code {
	case spannerpb.TypeCode_BOOL:
		return true
	case spannerpb.TypeCode_INT64:
		return int64(1)
	case spannerpb.TypeCode_FLOAT32:
		return float32(math.Pi)
	case spannerpb.TypeCode_FLOAT64:
		return math.Pi
	case spannerpb.TypeCode_STRING:
		return "value"
	case spannerpb.TypeCode_BYTES:
		return []byte("value")
	case spannerpb.TypeCode_DATE:
		return civil.Date{Year: 2026, Month: time.May, Day: 6}
	case spannerpb.TypeCode_TIMESTAMP:
		return time.Unix(0, 0).UTC()
	default:
		t.Fatalf("unsupported integration test parameter type %s", strings.TrimSpace(typeSQL))
		return nil
	}
}

func querygenIntegrationRequireContainerRuntime(t testing.TB) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			querygenIntegrationContainerRuntimeUnavailable(t, r)
		}
	}()
	provider, err := testcontainers.ProviderDocker.GetProvider()
	if err != nil {
		querygenIntegrationContainerRuntimeUnavailable(t, err)
		return
	}
	if err := provider.Health(context.Background()); err != nil {
		querygenIntegrationContainerRuntimeUnavailable(t, err)
	}
}

func querygenIntegrationContainerRuntimeUnavailable(t testing.TB, reason interface{}) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Fatalf("container runtime is required for integration tests in CI: %v", reason)
	}
	t.Skipf("container runtime is not available: %v", reason)
}
