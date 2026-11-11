// mutation-count-lab compares experimental Spanner write-accounting models.
// It reads schema, wire mutations, DML, and retained synthetic observations;
// it never connects to a database. See the write-analysis Research Note for
// assumptions, calibration scope, and the separate opt-in local probe.
package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(args []string, out io.Writer) error {
	f := flag.NewFlagSet("mutation-count-lab", flag.ContinueOnError)
	ddlPath := f.String("ddl", "", "schema DDL file")
	mutPath := f.String("mutations", "", "JSON array of google.spanner.v1.Mutation (one commit)")
	dml := f.String("dml", "", "one GoogleSQL DML statement; UPDATE/DELETE affected rows default to unknown")
	factsPath := f.String("facts", "", "optional JSON array of row facts, one per normalized operation")
	strategy := f.String("strategy", "bounds", "shape, bounds, or row-image (experimental Omni LTS.2 calibration)")
	indexesPath := f.String("indexes", "", "complete probe receipt supplying observed indexes for the same DDL")
	receiptPath := f.String("compare", "", "compare all three models on a retained probe receipt (.json or .json.gz)")
	provider := f.String("index-source", "observed", "comparison schema provider: ddl or observed")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	var result interface{}
	if *receiptPath != "" {
		if *ddlPath != "" || *mutPath != "" || *dml != "" || *factsPath != "" || *indexesPath != "" {
			return fmt.Errorf("compare cannot be mixed with analysis inputs")
		}
		r, err := readReceipt(*receiptPath)
		if err != nil {
			return err
		}
		result, err = compareReceipt(r, *provider)
		if err != nil {
			return err
		}
	} else {
		if *ddlPath == "" || (*mutPath == "") == (*dml == "") {
			return fmt.Errorf("provide ddl and exactly one of mutations or dml")
		}
		data, err := os.ReadFile(*ddlPath)
		if err != nil {
			return err
		}
		s, err := schemaFromDDL(string(data))
		if err != nil {
			return err
		}
		if *indexesPath != "" {
			r, e := readReceipt(*indexesPath)
			if e != nil {
				return e
			}
			if r.Status != "complete" {
				return fmt.Errorf("index snapshot receipt is incomplete")
			}
			observed, e := schemaFromDDL(receiptDDL(r))
			if e != nil {
				return e
			}
			if s.SourceSHA256 != observed.SourceSHA256 {
				return fmt.Errorf("index snapshot DDL does not match input DDL")
			}
			s, err = s.withObservedIndexes(r.Indexes)
			if err != nil {
				return err
			}
		}
		var ops []writeOperation
		if *mutPath != "" {
			data, e := os.ReadFile(*mutPath)
			if e != nil {
				return e
			}
			ops, err = decodeMutations(s, data)
		} else {
			ops, err = dmlOperations(s, *dml)
		}
		if err != nil {
			return err
		}
		if *factsPath != "" {
			data, e := os.ReadFile(*factsPath)
			if e != nil {
				return e
			}
			var facts []rowFacts
			if err := json.Unmarshal(data, &facts); err != nil {
				return err
			}
			if len(facts) != len(ops) {
				return fmt.Errorf("facts count %d differs from normalized operations %d", len(facts), len(ops))
			}
			for n := range ops {
				if ops[n].Channel == "mutation" && ops[n].Kind != "delete" && facts[n].Rows != nil && *facts[n].Rows != 1 {
					return fmt.Errorf("wire write row cardinality is already known; rows fact must be 1")
				}
				if err := validateRowFacts(s, ops[n], &facts[n]); err != nil {
					return fmt.Errorf("facts %d: %w", n, err)
				}
				ops[n].Facts = facts[n]
			}
		}
		result, err = analyze(s, ops, *strategy)
		if err != nil {
			return err
		}
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

type probeObservation struct {
	ID           string            `json:"id"`
	Kind         string            `json:"kind"`
	Table        string            `json:"table"`
	State        string            `json:"before_state"`
	Status       string            `json:"status"`
	StatsPresent bool              `json:"stats_present"`
	Count        *int64            `json:"mutation_count"`
	Rows         []int64           `json:"dml_rows"`
	Mutations    []json.RawMessage `json:"mutations"`
	Statements   []string          `json:"statements"`
}
type probeReceipt struct {
	Status       string                                `json:"status"`
	Backend      string                                `json:"backend"`
	Image        string                                `json:"configured_image"`
	DDL          []string                              `json:"ddl"`
	Extra        []struct{ DDL, Status, Error string } `json:"extra_schema"`
	Indexes      []observedIndex                       `json:"indexes"`
	Observations []probeObservation                    `json:"observations"`
}

func readReceipt(path string) (probeReceipt, error) {
	f, err := os.Open(path)
	if err != nil {
		return probeReceipt{}, err
	}
	// Read-only handles: io.ReadAll below reports read/checksum failures;
	// closing the already-consumed input has no output to flush.
	defer func() { _ = f.Close() }()
	var in io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		z, e := gzip.NewReader(f)
		if e != nil {
			return probeReceipt{}, e
		}
		defer func() { _ = z.Close() }()
		in = z
	}
	data, err := io.ReadAll(in)
	if err != nil {
		return probeReceipt{}, err
	}
	var r probeReceipt
	err = json.Unmarshal(data, &r)
	return r, err
}
func receiptDDL(r probeReceipt) string {
	ddls := append([]string(nil), r.DDL...)
	for _, x := range r.Extra {
		if x.Status == "OK" {
			ddls = append(ddls, x.DDL)
		}
	}
	return strings.Join(ddls, ";\n") + ";\n"
}

type comparison struct {
	Backend       string                  `json:"backend"`
	Image         string                  `json:"image"`
	ReceiptStatus string                  `json:"receipt_status"`
	IndexSource   string                  `json:"index_source"`
	SchemaSHA256  string                  `json:"schema_sha256"`
	Scores        map[string]*score       `json:"scores"`
	Cases         []comparedCase          `json:"cases"`
	Indexes       map[string][]indexModel `json:"indexes"`
}
type score struct {
	Observed    int `json:"with_observed_stats"`
	Contains    int `json:"interval_contains_observation"`
	Exact       int `json:"exact_matches"`
	Outside     int `json:"outside_interval"`
	Unavailable int `json:"unavailable_stats"`
	Failed      int `json:"failed_operations"`
}
type comparedCase struct {
	ID           string              `json:"id"`
	Status       string              `json:"status"`
	Observed     *int64              `json:"observed_mutation_count"`
	StatsPresent bool                `json:"stats_present"`
	Estimates    map[string]analysis `json:"estimates,omitempty"`
}

func compareReceipt(r probeReceipt, provider string) (comparison, error) {
	c := comparison{Backend: r.Backend, Image: r.Image, ReceiptStatus: r.Status, Scores: map[string]*score{}}
	if r.Status != "complete" {
		return c, fmt.Errorf("refusing incomplete receipt")
	}
	s, err := schemaFromDDL(receiptDDL(r))
	if err != nil {
		return c, err
	}
	if provider == "observed" {
		s, err = s.withObservedIndexes(r.Indexes)
		if err != nil {
			return c, err
		}
	} else if provider != "ddl" {
		return c, fmt.Errorf("index-source must be ddl or observed")
	}
	c.SchemaSHA256 = s.SourceSHA256
	c.IndexSource = s.IndexSource
	c.Indexes = s.Indexes
	for _, strategy := range []string{"shape", "bounds", "row-image"} {
		c.Scores[strategy] = &score{}
	}
	for _, o := range r.Observations {
		row := comparedCase{ID: o.ID, Status: o.Status, Observed: o.Count, StatsPresent: o.StatsPresent}
		if o.Status != "OK" {
			for _, score := range c.Scores {
				score.Failed++
			}
			c.Cases = append(c.Cases, row)
			continue
		}
		ops, err := fixtureOperations(s, o)
		if err != nil {
			return c, fmt.Errorf("%s: %w", o.ID, err)
		}
		row.Estimates = map[string]analysis{}
		for _, strategy := range []string{"shape", "bounds", "row-image"} {
			a, err := analyze(s, ops, strategy)
			if err != nil {
				return c, fmt.Errorf("%s %s: %w", o.ID, strategy, err)
			}
			if len(a.Results) > 32 {
				a.Results = nil
			}
			row.Estimates[strategy] = a
			score := c.Scores[strategy]
			if !o.StatsPresent || o.Count == nil {
				score.Unavailable++
				continue
			}
			score.Observed++
			if a.Count.Lower <= *o.Count && (a.Count.Upper == nil || *a.Count.Upper >= *o.Count) {
				score.Contains++
				if a.Count.exact() {
					score.Exact++
				}
			} else {
				score.Outside++
			}
		}
		c.Cases = append(c.Cases, row)
	}
	return c, nil
}

// These facts come from the retained probe's deterministic seed code, never
// from CommitStats. This adapter is fixture-specific and is not a DB reader.
func fixtureOperations(s *schemaModel, o probeObservation) ([]writeOperation, error) {
	var ops []writeOperation
	if len(o.Statements) > 0 {
		oldActive := "a"
		for i, sql := range o.Statements {
			part, err := dmlOperations(s, sql)
			if err != nil {
				return nil, err
			}
			if len(part) != 1 || i >= len(o.Rows) {
				return nil, fmt.Errorf("fixture DML cardinality missing")
			}
			part[0].Facts.Rows = &o.Rows[i]
			part[0].Statement = i
			if part[0].Kind == "update" {
				part[0].Facts.Old = map[string]interface{}{"active": oldActive, "note": "n"}
				if v, ok := part[0].Values["active"].(string); ok {
					oldActive = v
				}
			}
			ops = append(ops, part...)
		}
	} else {
		data, err := json.Marshal(o.Mutations)
		if err != nil {
			return nil, err
		}
		ops, err = decodeMutations(s, data)
		if err != nil {
			return nil, err
		}
	}
	for n := range ops {
		op := &ops[n]
		yes, no := true, false
		op.Assumptions = append(op.Assumptions, "fixture seed facts supplied by retained probe source; no live database read")
		switch o.State {
		case "absent":
			op.Facts.Exists = &no
			if op.Kind == "delete" {
				v := int64(0)
				op.Facts.Rows = &v
			}
		case "present:a,n":
			if op.Channel != "dml" {
				op.Facts.Old = map[string]interface{}{"active": "a", "note": "n"}
			}
			op.Facts.Exists = &yes
		case "present=true":
			op.Facts.Exists = &yes
			op.Facts.Old = map[string]interface{}{"active": "old", "note": "old"}
		case "present=false", "20001 new rows", "referenced owner exists":
			op.Facts.Exists = &no
		case "active=NULL":
			op.Facts.Exists = &yes
			op.Facts.Old = map[string]interface{}{"active": nil, "note": "n"}
		case "active=a":
			op.Facts.Exists = &yes
			op.Facts.Old = map[string]interface{}{"active": "a", "note": "n"}
		case "3 existing indexable rows":
			v := int64(3)
			op.Facts.Rows = &v
			op.Facts.IndexRows = map[string]int64{}
			for _, i := range s.Indexes[strings.ToLower(op.Table)] {
				op.Facts.IndexRows[i.Name] = 3
			}
		case "parent +3 indexed children":
			v := int64(1)
			op.Facts.Rows = &v
			op.Facts.CascadeIndexRows = map[string]int64{"ChildNote": 3}
		default:
			return nil, fmt.Errorf("unrecognized fixture seed state %q", o.State)
		}
	}
	return ops, nil
}
