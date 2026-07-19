package planvocab

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/protobuf/encoding/protojson"
)

// knownFixtureGaps is an explicit, reviewed escape hatch for a raw fixture
// that intentionally lands before its vocabulary catalog update. Each entry is
// a redacted finding fingerprint. An empty map is the normal state.
var knownFixtureGaps = map[string][]string{}

func TestGeneratedPlanFixturesUseKnownVocabulary(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("testdata/fixture_plans.json")
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	var fixtures struct {
		Version string `json:"version"`
		Plans   []struct {
			Name string          `json:"name"`
			Plan json.RawMessage `json:"plan"`
		} `json:"plans"`
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if fixtures.Version != "v0alpha1" || len(fixtures.Plans) == 0 {
		t.Fatalf("fixture mirror header = version %q plans %d, want v0alpha1 and non-empty", fixtures.Version, len(fixtures.Plans))
	}
	for _, fixture := range fixtures.Plans {
		fixture := fixture
		t.Run(fixture.Name, func(t *testing.T) {
			t.Parallel()
			plan := &spannerpb.QueryPlan{}
			if err := protojson.Unmarshal(fixture.Plan, plan); err != nil {
				t.Fatalf("protojson.Unmarshal() error = %v", err)
			}
			findings := Inspect(plan)
			fingerprints := make([]string, 0, len(findings))
			for _, finding := range findings {
				fingerprints = append(fingerprints, finding.Fingerprint)
			}
			slices.Sort(fingerprints)
			want := append([]string{}, knownFixtureGaps[fixture.Name]...)
			slices.Sort(want)
			if !slices.Equal(fingerprints, want) {
				t.Fatalf(
					"plan vocabulary findings = %v, want reviewed gaps %v; findings=%+v; update catalog_source.json and regenerate, or add a reviewed temporary gap",
					fingerprints,
					want,
					findings,
				)
			}
		})
	}
}
