package infoschem_test

import (
	"context"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	databasepb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"github.com/apstndb/spanalyzer/survey/astconv"
	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
	"google.golang.org/grpc/status"
)

// TestCanonicalDDL_RealSpanner verifies that the current managed database can
// pass through LoadSchema and ToDDLStatements. It deliberately reports only
// aggregate AST-family counts so database object identifiers never enter test
// output. Count equality is not required because INFORMATION_SCHEMA cannot
// recover every GetDatabaseDdl surface and can expose generated schema state.
func TestCanonicalDDL_RealSpanner(t *testing.T) {
	if os.Getenv("REQUIRE_REAL_SPANNER_CANONICAL") != "1" {
		t.Skip("skipping managed-Spanner canonical comparison; run mise run test-canonical-real")
	}
	databaseName := os.Getenv("TEST_REAL_SPANNER_DATABASE")
	if databaseName == "" {
		t.Fatal("TEST_REAL_SPANNER_DATABASE is required when REQUIRE_REAL_SPANNER_CANONICAL=1")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	admin, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		t.Fatalf("database.NewDatabaseAdminClient failed (code=%s)", status.Code(err))
	}
	defer func() {
		if err := admin.Close(); err != nil {
			t.Errorf("close managed-Spanner database admin client: %v", err)
		}
	}()
	canonical, err := admin.GetDatabaseDdl(ctx, &databasepb.GetDatabaseDdlRequest{Database: databaseName})
	if err != nil {
		t.Fatalf("GetDatabaseDdl failed (code=%s)", status.Code(err))
	}

	canonicalCounts := make(map[string]int)
	parseErrors := 0
	for _, sql := range canonical.Statements {
		ddl, err := memefish.ParseDDL("", sql)
		if err != nil {
			parseErrors++
			continue
		}
		canonicalCounts[ddlFamily(ddl)]++
	}

	client, err := spanner.NewClient(ctx, databaseName)
	if err != nil {
		t.Fatalf("spanner.NewClient failed (code=%s)", status.Code(err))
	}
	defer client.Close()
	schema, err := astconv.LoadSchema(ctx, client)
	if err != nil {
		t.Fatalf("LoadSchema failed (code=%s)", spanner.ErrCode(err))
	}
	generated, err := schema.ToDDLStatements()
	if err != nil {
		step := strings.SplitN(err.Error(), ":", 2)[0]
		t.Fatalf("ToDDLStatements failed in %s", step)
	}
	generatedCounts := make(map[string]int)
	for _, ddl := range generated {
		generatedCounts[ddlFamily(ddl)]++
	}

	families := make(map[string]bool)
	for name := range canonicalCounts {
		families[name] = true
	}
	for name := range generatedCounts {
		families[name] = true
	}
	names := make([]string, 0, len(families))
	for name := range families {
		names = append(names, name)
	}
	sort.Strings(names)

	t.Logf(
		"canonical DDL aggregate: canonical_total=%d canonical_parse_errors=%d generated_total=%d",
		len(canonical.Statements),
		parseErrors,
		len(generated),
	)
	for _, name := range names {
		t.Logf(
			"canonical DDL family: family=%s canonical=%d generated=%d delta=%+d",
			name,
			canonicalCounts[name],
			generatedCounts[name],
			generatedCounts[name]-canonicalCounts[name],
		)
	}
}

func ddlFamily(ddl ast.DDL) string {
	return strings.TrimPrefix(reflect.TypeOf(ddl).String(), "*ast.")
}
