// Command planvocab-gen produces the embedded plan-vocabulary catalog from a
// reviewed structured source and stamps deterministic input provenance.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const generatedBy = "go run ./planvocab-gen"

var generatorInputPaths = []string{
	"schemas/spanalyzer.planvocab.v0alpha1.schema.json",
	"schemas/spanalyzer.planvocab-expectations.v0alpha1.schema.json",
	"tools/planvocab-gen/main.go",
}

type document struct {
	Info           catalogInfo    `json:"info"`
	CommonMetadata []metadataRule `json:"common_metadata"`
	Operators      []operatorRule `json:"operators"`
}

type fixtureSet struct {
	Version string        `json:"version"`
	Plans   []fixturePlan `json:"plans"`
}

type fixturePlan struct {
	Name string          `json:"name"`
	Plan json.RawMessage `json:"plan"`
}

type catalogInfo struct {
	Version       string         `json:"version"`
	Repository    string         `json:"repository"`
	Revision      string         `json:"revision"`
	Path          string         `json:"path"`
	Blob          string         `json:"blob"`
	LocalEvidence []string       `json:"local_evidence"`
	Compatibility string         `json:"compatibility"`
	GeneratedBy   string         `json:"generated_by,omitempty"`
	Inputs        []catalogInput `json:"inputs,omitempty"`
}

type catalogInput struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type metadataRule struct {
	Key    string   `json:"key"`
	Values []string `json:"values,omitempty"`
}

type operatorRule struct {
	Names      []string       `json:"names"`
	Kind       string         `json:"kind"`
	Metadata   []metadataRule `json:"metadata"`
	ChildLinks []childRule    `json:"child_links"`
}

type childRule struct {
	Kind     string `json:"kind"`
	Type     string `json:"type"`
	Variable string `json:"variable"`
	Multiple *bool  `json:"multiple,omitempty"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("planvocab-gen", flag.ContinueOnError)
	repoRoot := fs.String("repo-root", "", "repository root (auto-detected when omitted)")
	sourcePath := fs.String("source", "plancontract/planvocab/catalog_source.json", "structured catalog source relative to repository root")
	outputPath := fs.String("output", "plancontract/planvocab/catalog.json", "generated catalog relative to repository root")
	fixturesOutputPath := fs.String("fixtures-output", "plancontract/planvocab/testdata/fixture_plans.json", "generated fixture mirror relative to repository root")
	check := fs.Bool("check", false, "fail if the generated catalog differs from the output file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	root, err := resolveRepoRoot(*repoRoot)
	if err != nil {
		return err
	}
	doc, err := readSource(filepath.Join(root, filepath.FromSlash(*sourcePath)))
	if err != nil {
		return err
	}
	if err := normalizeAndValidate(&doc); err != nil {
		return err
	}

	inputs := catalogInputPaths(*sourcePath, doc.Info.LocalEvidence)
	fixtures, err := filepath.Glob(filepath.Join(root, "cmd/spanner-query-gen/testdata/plan_fixtures/*.json"))
	if err != nil {
		return fmt.Errorf("glob plan fixtures: %w", err)
	}
	for _, fixture := range fixtures {
		relative, err := filepath.Rel(root, fixture)
		if err != nil {
			return fmt.Errorf("relativize fixture %q: %w", fixture, err)
		}
		inputs = append(inputs, filepath.ToSlash(relative))
	}
	sort.Strings(inputs)
	doc.Info.GeneratedBy = generatedBy
	doc.Info.Inputs, err = hashInputs(root, inputs)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal catalog: %w", err)
	}
	data = append(data, '\n')
	fixtureData, err := generateFixtureSet(fixtures)
	if err != nil {
		return err
	}
	output := filepath.Join(root, filepath.FromSlash(*outputPath))
	fixturesOutput := filepath.Join(root, filepath.FromSlash(*fixturesOutputPath))
	if *check {
		existing, err := os.ReadFile(output)
		if err != nil {
			return fmt.Errorf("read generated catalog: %w", err)
		}
		if !bytes.Equal(existing, data) {
			return errors.New("generated plan-vocabulary catalog is stale; run go run ./planvocab-gen from tools")
		}
		existingFixtures, err := os.ReadFile(fixturesOutput)
		if err != nil {
			return fmt.Errorf("read generated fixture mirror: %w", err)
		}
		if !bytes.Equal(existingFixtures, fixtureData) {
			return errors.New("generated plan-vocabulary fixture mirror is stale; run go run ./planvocab-gen from tools")
		}
		return nil
	}
	if err := os.WriteFile(output, data, 0o644); err != nil {
		return fmt.Errorf("write generated catalog: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(fixturesOutput), 0o755); err != nil {
		return fmt.Errorf("create fixture mirror directory: %w", err)
	}
	if err := os.WriteFile(fixturesOutput, fixtureData, 0o644); err != nil {
		return fmt.Errorf("write generated fixture mirror: %w", err)
	}
	return nil
}

// catalogInputPaths keeps the structured source's local_evidence list as the
// single authority for evidence provenance. Generator and schema inputs are
// added separately because they define the generated representation rather
// than authorize individual vocabulary entries.
func catalogInputPaths(sourcePath string, localEvidence []string) []string {
	seen := make(map[string]struct{}, 1+len(localEvidence)+len(generatorInputPaths))
	inputs := make([]string, 0, 1+len(localEvidence)+len(generatorInputPaths))
	add := func(path string) {
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		inputs = append(inputs, path)
	}
	add(sourcePath)
	for _, path := range localEvidence {
		add(path)
	}
	for _, path := range generatorInputPaths {
		add(path)
	}
	sort.Strings(inputs)
	return inputs
}

func resolveRepoRoot(explicit string) (string, error) {
	if explicit != "" {
		root, err := filepath.Abs(explicit)
		if err != nil {
			return "", fmt.Errorf("resolve repository root: %w", err)
		}
		return root, nil
	}
	start, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory for repository-root detection: %w", err)
	}
	return findRepoRoot(start)
}

func findRepoRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve repository-root search path: %w", err)
	}
	for {
		catalogSource := filepath.Join(dir, "plancontract", "planvocab", "catalog_source.json")
		generatorSource := filepath.Join(dir, "tools", "planvocab-gen", "main.go")
		if regularFile(catalogSource) && regularFile(generatorSource) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repository root not found from %q; pass --repo-root", start)
		}
		dir = parent
	}
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func generateFixtureSet(paths []string) ([]byte, error) {
	set := fixtureSet{
		Version: "v0alpha1",
		Plans:   make([]fixturePlan, 0, len(paths)),
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read plan fixture %q: %w", path, err)
		}
		var fixture struct {
			Plan json.RawMessage `json:"plan"`
		}
		if err := json.Unmarshal(data, &fixture); err != nil {
			return nil, fmt.Errorf("decode plan fixture %q: %w", path, err)
		}
		if len(fixture.Plan) == 0 {
			return nil, fmt.Errorf("plan fixture %q has no plan", path)
		}
		name := filepath.Base(path)
		name = name[:len(name)-len(filepath.Ext(name))]
		set.Plans = append(set.Plans, fixturePlan{Name: name, Plan: fixture.Plan})
	}
	sort.Slice(set.Plans, func(i, j int) bool {
		return set.Plans[i].Name < set.Plans[j].Name
	})
	data, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal fixture mirror: %w", err)
	}
	return append(data, '\n'), nil
}

func readSource(path string) (document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return document{}, fmt.Errorf("read catalog source: %w", err)
	}
	var doc document
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return document{}, fmt.Errorf("decode catalog source: %w", err)
	}
	return doc, nil
}

func normalizeAndValidate(doc *document) error {
	if doc.Info.Version == "" || doc.Info.Revision == "" || doc.Info.Blob == "" {
		return errors.New("catalog source lacks versioned spanner-hacks provenance")
	}
	if len(doc.Operators) == 0 {
		return errors.New("catalog source contains no operators")
	}
	seenNames := map[string]bool{}
	for i := range doc.Operators {
		operator := &doc.Operators[i]
		sort.Strings(operator.Names)
		if operator.Kind == "" || len(operator.Names) == 0 {
			return fmt.Errorf("operator %d lacks kind or names", i)
		}
		for _, name := range operator.Names {
			if seenNames[name] {
				return fmt.Errorf("duplicate operator name %q", name)
			}
			seenNames[name] = true
		}
		sort.Slice(operator.Metadata, func(i, j int) bool {
			return operator.Metadata[i].Key < operator.Metadata[j].Key
		})
		for i := range operator.Metadata {
			sort.Strings(operator.Metadata[i].Values)
		}
		sort.Slice(operator.ChildLinks, func(i, j int) bool {
			left := operator.ChildLinks[i]
			right := operator.ChildLinks[j]
			if left.Kind != right.Kind {
				return left.Kind < right.Kind
			}
			if left.Type != right.Type {
				return left.Type < right.Type
			}
			if left.Variable != right.Variable {
				return left.Variable < right.Variable
			}
			return multipleOrder(left.Multiple) < multipleOrder(right.Multiple)
		})
	}
	sort.Slice(doc.Operators, func(i, j int) bool {
		return doc.Operators[i].Names[0] < doc.Operators[j].Names[0]
	})
	sort.Slice(doc.CommonMetadata, func(i, j int) bool {
		return doc.CommonMetadata[i].Key < doc.CommonMetadata[j].Key
	})
	for i := range doc.CommonMetadata {
		sort.Strings(doc.CommonMetadata[i].Values)
	}
	doc.Info.LocalEvidence = append([]string{}, doc.Info.LocalEvidence...)
	sort.Strings(doc.Info.LocalEvidence)
	doc.Info.GeneratedBy = ""
	doc.Info.Inputs = nil
	return nil
}

func multipleOrder(value *bool) int {
	if value == nil {
		return 0
	}
	if !*value {
		return 1
	}
	return 2
}

func hashInputs(root string, paths []string) ([]catalogInput, error) {
	inputs := make([]catalogInput, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return nil, fmt.Errorf("read provenance input %q: %w", path, err)
		}
		sum := sha256.Sum256(data)
		inputs = append(inputs, catalogInput{
			Path:   filepath.ToSlash(path),
			SHA256: hex.EncodeToString(sum[:]),
		})
	}
	return inputs, nil
}
