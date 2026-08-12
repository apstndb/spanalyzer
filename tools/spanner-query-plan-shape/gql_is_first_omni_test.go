//go:build integration && omni

package main

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/apstndb/spanemuboost"
)

func TestIntegrationGQLISFirstPlacementSemanticsOnOmni(t *testing.T) {
	if os.Getenv("SPANEMUBOOST_ENABLE_OMNI_TESTS") == "" {
		t.Skip("set SPANEMUBOOST_ENABLE_OMNI_TESTS=1 to run Spanner Omni tests")
	}
	image := strings.TrimSpace(os.Getenv("SPANALYZER_OMNI_IMAGE"))
	if image == "" {
		t.Fatal("set SPANALYZER_OMNI_IMAGE to the pinned Spanner Omni image under test")
	}
	ddls, err := parseBuiltInDDLs("gql-is-first-schema.sql", docsDDL)
	if err != nil {
		t.Fatalf("parseBuiltInDDLs() error = %v", err)
	}
	runtime := spanemuboost.NewLazyRuntime(
		spanemuboost.BackendOmni,
		spanemuboost.WithContainerImage(image),
	)
	t.Cleanup(func() { _ = runtime.Close() })
	clients, err := spanemuboost.OpenClients(
		t.Context(),
		runtime,
		spanemuboost.WithRandomDatabaseID(),
		spanemuboost.WithSetupDDLs(ddls),
	)
	if err != nil {
		t.Fatalf("OpenClients() error = %v", err)
	}
	t.Cleanup(func() { _ = clients.Close() })

	mutations := make([]*spanner.Mutation, 0, 11)
	for _, singerID := range []int64{1, 2, 3, 4, 5} {
		mutations = append(mutations, spanner.Insert("Singers", []string{"SingerId"}, []any{singerID}))
	}
	for _, edge := range []struct {
		sourceID      int64
		destinationID int64
		albumTitle    string
	}{
		{sourceID: 1, destinationID: 3, albumTitle: "a"},
		{sourceID: 1, destinationID: 4, albumTitle: "z"},
		{sourceID: 2, destinationID: 3, albumTitle: "b"},
		{sourceID: 2, destinationID: 5, albumTitle: "y"},
		{sourceID: 3, destinationID: 4, albumTitle: "c"},
		{sourceID: 3, destinationID: 5, albumTitle: "x"},
	} {
		mutations = append(mutations, spanner.Insert(
			"Collaborations",
			[]string{"SingerId", "FeaturingSingerId", "AlbumTitle"},
			[]any{edge.sourceID, edge.destinationID, edge.albumTitle},
		))
	}
	if _, err := clients.Client.Apply(t.Context(), mutations); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	cases := queryCasesByLabel(t, gqlSurfaceQueries)
	returnCase := gqlISFirstQueryCase(t, cases, gqlSurfaceISFirstReturnLabel)
	filterCase := gqlISFirstQueryCase(t, cases, gqlSurfaceISFirstFilterLabel)
	edgeOneHopCase := gqlISFirstQueryCase(t, cases, gqlSurfaceISFirstEdgeOneHopLabel)
	edgeQuantifiedCase := gqlISFirstQueryCase(t, cases, gqlSurfaceISFirstQuantifiedLabel)
	nextControlCase := gqlISFirstQueryCase(t, cases, "gql-surface/linear/next-two-stage-traversal")
	nextOrderedCase := gqlISFirstQueryCase(t, cases, gqlSurfaceISFirstBeforeNextOrderedLabel)
	quantifiedControlCase := gqlISFirstQueryCase(t, cases, "gql-surface/search/all-bounded")

	versions := []int{0, 1, 2, 3, 4, 5, 6, 7, 8}
	for _, version := range versions {
		name := "default"
		if version != 0 {
			name = fmt.Sprintf("v%d", version)
		}
		t.Run(name, func(t *testing.T) {
			withVersion := func(query queryCase) string {
				if version == 0 {
					return query.SQL
				}
				return withOptimizerVersionStatementHint(query.SQL, version)
			}

			assertISFirstBoolRows(
				t,
				clients.Client,
				withVersion(returnCase),
				[]string{"1:3:false", "1:4:true", "2:3:false", "2:5:true", "3:4:false", "3:5:true"},
			)
			wantSelectedEdges := []string{"1:4", "2:5", "3:5"}
			assertISFirstPairRows(t, clients.Client, withVersion(filterCase), wantSelectedEdges)
			assertISFirstPairRows(t, clients.Client, withVersion(edgeOneHopCase), wantSelectedEdges)

			assertISFirstPairRows(
				t,
				clients.Client,
				withVersion(quantifiedControlCase),
				[]string{"1:3", "1:4", "1:4", "1:5", "2:3", "2:4", "2:5", "2:5", "3:4", "3:5"},
			)
			assertISFirstPairRows(t, clients.Client, withVersion(edgeQuantifiedCase), wantSelectedEdges)

			assertISFirstPairRows(
				t,
				clients.Client,
				withVersion(nextControlCase),
				[]string{"3:4", "3:4", "3:5", "3:5"},
			)
			assertISFirstTripletRows(
				t,
				clients.Client,
				withVersion(nextOrderedCase),
				[]string{"2:3:4", "2:3:5"},
			)
		})
	}
}

func gqlISFirstQueryCase(t *testing.T, cases map[string]queryCase, label string) queryCase {
	t.Helper()
	query, ok := cases[label]
	if !ok {
		t.Fatalf("GQL IS_FIRST query %q is missing", label)
	}
	return query
}

func assertISFirstBoolRows(t *testing.T, client *spanner.Client, sql string, want []string) {
	t.Helper()
	var got []string
	err := client.Single().Query(t.Context(), spanner.Statement{SQL: sql}).Do(func(row *spanner.Row) error {
		var sourceID, destinationID int64
		var selected bool
		if err := row.Columns(&sourceID, &destinationID, &selected); err != nil {
			return err
		}
		got = append(got, fmt.Sprintf("%d:%d:%t", sourceID, destinationID, selected))
		return nil
	})
	if err != nil {
		t.Fatalf("Query() error = %v\nSQL:\n%s", err, sql)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("result rows = %v, want %v\nSQL:\n%s", got, want, sql)
	}
}

func assertISFirstPairRows(t *testing.T, client *spanner.Client, sql string, want []string) {
	t.Helper()
	var got []string
	err := client.Single().Query(t.Context(), spanner.Statement{SQL: sql}).Do(func(row *spanner.Row) error {
		var sourceID, destinationID int64
		if err := row.Columns(&sourceID, &destinationID); err != nil {
			return err
		}
		got = append(got, fmt.Sprintf("%d:%d", sourceID, destinationID))
		return nil
	})
	if err != nil {
		t.Fatalf("Query() error = %v\nSQL:\n%s", err, sql)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("result rows = %v, want %v\nSQL:\n%s", got, want, sql)
	}
}

func assertISFirstTripletRows(t *testing.T, client *spanner.Client, sql string, want []string) {
	t.Helper()
	var got []string
	err := client.Single().Query(t.Context(), spanner.Statement{SQL: sql}).Do(func(row *spanner.Row) error {
		var sourceID, middleID, destinationID int64
		if err := row.Columns(&sourceID, &middleID, &destinationID); err != nil {
			return err
		}
		got = append(got, fmt.Sprintf("%d:%d:%d", sourceID, middleID, destinationID))
		return nil
	})
	if err != nil {
		t.Fatalf("Query() error = %v\nSQL:\n%s", err, sql)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("result rows = %v, want %v\nSQL:\n%s", got, want, sql)
	}
}
