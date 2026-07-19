// Command planvocab-check checks raw QueryPlan envelopes with planvocab.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/apstndb/spanalyzer/plancontract/planvocab"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type rawPlanEnvelope struct {
	QueryLabel string          `json:"query_label"`
	Plan       json.RawMessage `json:"plan"`
	Error      string          `json:"error"`
}

type sourceInput struct {
	Name string
	Data []byte
}

type queryError struct {
	Source string `json:"source"`
	Label  string `json:"label"`
}

type findingRecord struct {
	Source  string            `json:"source"`
	Label   string            `json:"label"`
	Finding planvocab.Finding `json:"finding"`
}

type expectationDocument struct {
	Version string             `json:"version"`
	Queries []queryExpectation `json:"queries"`
}

type queryExpectation struct {
	Label    string                `json:"label"`
	Patterns []operatorExpectation `json:"patterns"`
}

type operatorExpectation struct {
	DisplayName string                 `json:"display_name"`
	Family      string                 `json:"family"`
	Metadata    []metadataExpectation  `json:"metadata"`
	ChildLinks  []childLinkExpectation `json:"child_links"`
}

type metadataExpectation struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type childLinkExpectation struct {
	Kind     string `json:"kind"`
	Type     string `json:"type"`
	Variable string `json:"variable"`
	MinCount int    `json:"min_count"`
}

type expectationFailure struct {
	Label string `json:"label"`
	// Pattern is a 1-based pattern ordinal. Zero means the failure applies to
	// the query label as a whole and is omitted from JSON output.
	Pattern int    `json:"pattern,omitempty"`
	Reason  string `json:"reason"`
}

type checkReport struct {
	Sources             int                  `json:"sources"`
	Plans               int                  `json:"plans"`
	QueryErrors         []queryError         `json:"query_errors"`
	Findings            []findingRecord      `json:"findings"`
	Expectations        int                  `json:"expectations"`
	ExpectationFailures []expectationFailure `json:"expectation_failures"`
}

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("planvocab-check", flag.ContinueOnError)
	allowQueryErrors := fs.Bool("allow-query-errors", false, "report source query errors without failing the vocabulary check")
	outputFormat := fs.String("format", "text", "output format: text or json")
	expectationPath := fs.String("expect", "", "positive expectation manifest keyed by query label")
	if err := fs.Parse(args); err != nil {
		return err
	}

	inputs, err := readInputs(fs.Args(), stdin)
	if err != nil {
		return err
	}
	expectations, err := readExpectations(*expectationPath)
	if err != nil {
		return err
	}
	report, err := inspectInputs(inputs, expectations)
	if err != nil {
		return err
	}
	if err := writeReport(stdout, report, *outputFormat); err != nil {
		return err
	}
	if len(report.Findings) != 0 {
		return fmt.Errorf("plan vocabulary check found %d finding(s)", len(report.Findings))
	}
	if len(report.ExpectationFailures) != 0 {
		return fmt.Errorf("plan vocabulary check found %d expectation failure(s)", len(report.ExpectationFailures))
	}
	if len(report.QueryErrors) != 0 && !*allowQueryErrors {
		return fmt.Errorf("plan source reported %d query error(s); pass --allow-query-errors to check the available plans", len(report.QueryErrors))
	}
	return nil
}

func readInputs(paths []string, stdin io.Reader) ([]sourceInput, error) {
	if len(paths) == 0 {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		return []sourceInput{{Name: "stdin", Data: data}}, nil
	}
	inputs := make([]sourceInput, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", path, err)
		}
		inputs = append(inputs, sourceInput{Name: filepath.ToSlash(path), Data: data})
	}
	return inputs, nil
}

func readExpectations(path string) (expectationDocument, error) {
	if strings.TrimSpace(path) == "" {
		return expectationDocument{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return expectationDocument{}, fmt.Errorf("read expectation manifest %q: %w", path, err)
	}
	var document expectationDocument
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return expectationDocument{}, fmt.Errorf("decode expectation manifest %q: %w", path, err)
	}
	if document.Version != "v0alpha1" {
		return expectationDocument{}, fmt.Errorf("expectation manifest version = %q, want v0alpha1", document.Version)
	}
	seen := map[string]bool{}
	for _, query := range document.Queries {
		if strings.TrimSpace(query.Label) == "" || len(query.Patterns) == 0 {
			return expectationDocument{}, errors.New("expectation query requires a label and at least one pattern")
		}
		if seen[query.Label] {
			return expectationDocument{}, fmt.Errorf("duplicate expectation query label %q", query.Label)
		}
		seen[query.Label] = true
	}
	return document, nil
}

func inspectInputs(inputs []sourceInput, expectations expectationDocument) (checkReport, error) {
	report := checkReport{
		Sources:             len(inputs),
		QueryErrors:         []queryError{},
		Findings:            []findingRecord{},
		ExpectationFailures: []expectationFailure{},
	}
	expectationsByLabel := make(map[string][]operatorExpectation, len(expectations.Queries))
	for _, query := range expectations.Queries {
		expectationsByLabel[query.Label] = query.Patterns
		report.Expectations += len(query.Patterns)
	}
	seenLabels := map[string]bool{}
	for _, input := range inputs {
		envelopes, err := decodeEnvelopes(input.Data)
		if err != nil {
			return checkReport{}, fmt.Errorf("decode %s: %w", input.Name, err)
		}
		for i, envelope := range envelopes {
			label := strings.TrimSpace(envelope.QueryLabel)
			if label == "" {
				label = fallbackLabel(input.Name, i, len(envelopes))
			}
			seenLabels[label] = true
			if envelope.Error != "" {
				report.QueryErrors = append(report.QueryErrors, queryError{Source: input.Name, Label: label})
				if patterns := expectationsByLabel[label]; len(patterns) != 0 {
					report.ExpectationFailures = append(report.ExpectationFailures, expectationFailure{
						Label:  label,
						Reason: "query produced no plan",
					})
				}
				continue
			}
			if len(envelope.Plan) == 0 || bytes.Equal(bytes.TrimSpace(envelope.Plan), []byte("null")) {
				return checkReport{}, fmt.Errorf("%s:%s has neither plan nor query error", input.Name, label)
			}
			plan := &spannerpb.QueryPlan{}
			if err := protojson.Unmarshal(envelope.Plan, plan); err != nil {
				return checkReport{}, fmt.Errorf("decode plan %s:%s: %w", input.Name, label, err)
			}
			report.Plans++
			for _, finding := range planvocab.Inspect(plan) {
				report.Findings = append(report.Findings, findingRecord{
					Source:  input.Name,
					Label:   label,
					Finding: finding,
				})
			}
			for patternIndex, expectation := range expectationsByLabel[label] {
				pattern, err := expectation.pattern()
				if err != nil {
					return checkReport{}, fmt.Errorf("expectation %s pattern %d: %w", label, patternIndex+1, err)
				}
				result, err := planvocab.FindMatchingOperators(plan, pattern)
				if err != nil {
					return checkReport{}, fmt.Errorf("expectation %s pattern %d: %w", label, patternIndex+1, err)
				}
				if !result.HasMatches() {
					report.ExpectationFailures = append(report.ExpectationFailures, expectationFailure{
						Label:   label,
						Pattern: patternIndex + 1,
						Reason:  result.String(),
					})
				}
			}
		}
	}
	for _, query := range expectations.Queries {
		if !seenLabels[query.Label] {
			report.ExpectationFailures = append(report.ExpectationFailures, expectationFailure{
				Label:  query.Label,
				Reason: "query label was not present in the input",
			})
		}
	}
	return report, nil
}

func (expectation operatorExpectation) pattern() (planvocab.OperatorPattern, error) {
	pattern := planvocab.OperatorPattern{
		DisplayName: expectation.DisplayName,
		Family:      expectation.Family,
		Metadata:    make([]planvocab.MetadataRequirement, 0, len(expectation.Metadata)),
		ChildLinks:  make([]planvocab.ChildLinkRequirement, 0, len(expectation.ChildLinks)),
	}
	for _, metadata := range expectation.Metadata {
		requirement := planvocab.MetadataRequirement{Key: metadata.Key}
		if len(metadata.Value) != 0 {
			var value interface{}
			if err := json.Unmarshal(metadata.Value, &value); err != nil {
				return planvocab.OperatorPattern{}, fmt.Errorf("decode metadata %q value: %w", metadata.Key, err)
			}
			if value == nil {
				return planvocab.OperatorPattern{}, fmt.Errorf("metadata %q expectation cannot be null", metadata.Key)
			}
			protobufValue, err := structpb.NewValue(value)
			if err != nil {
				return planvocab.OperatorPattern{}, fmt.Errorf("convert metadata %q value: %w", metadata.Key, err)
			}
			requirement.Value = protobufValue
		}
		pattern.Metadata = append(pattern.Metadata, requirement)
	}
	for _, child := range expectation.ChildLinks {
		kind := spannerpb.PlanNode_KIND_UNSPECIFIED
		if child.Kind != "" {
			value, found := spannerpb.PlanNode_Kind_value[child.Kind]
			if !found {
				return planvocab.OperatorPattern{}, fmt.Errorf("unknown child kind %q", child.Kind)
			}
			kind = spannerpb.PlanNode_Kind(value)
		}
		variable := planvocab.VariableAny
		switch child.Variable {
		case "", "any":
		case "absent":
			variable = planvocab.VariableAbsent
		case "present":
			variable = planvocab.VariablePresent
		default:
			return planvocab.OperatorPattern{}, fmt.Errorf("unknown child variable presence %q", child.Variable)
		}
		pattern.ChildLinks = append(pattern.ChildLinks, planvocab.ChildLinkRequirement{
			Kind:     kind,
			Type:     child.Type,
			Variable: variable,
			MinCount: child.MinCount,
		})
	}
	return pattern, nil
}

func decodeEnvelopes(data []byte) ([]rawPlanEnvelope, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, errors.New("empty input")
	}
	if data[0] == '[' {
		var envelopes []rawPlanEnvelope
		if err := json.Unmarshal(data, &envelopes); err != nil {
			return nil, err
		}
		if len(envelopes) == 0 {
			return nil, errors.New("input contains no plan envelopes")
		}
		return envelopes, nil
	}
	var envelope rawPlanEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	return []rawPlanEnvelope{envelope}, nil
}

func fallbackLabel(source string, index, count int) string {
	base := filepath.Base(source)
	if count == 1 {
		return base
	}
	return fmt.Sprintf("%s#%d", base, index+1)
}

func writeReport(w io.Writer, report checkReport, format string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "text":
		if _, err := fmt.Fprintf(w, "sources: %d\nplans: %d\nquery_errors: %d\nfindings: %d\nexpectations: %d\nexpectation_failures: %d\n", report.Sources, report.Plans, len(report.QueryErrors), len(report.Findings), report.Expectations, len(report.ExpectationFailures)); err != nil {
			return err
		}
		for _, queryErr := range report.QueryErrors {
			if _, err := fmt.Fprintf(w, "query_error: %s:%s\n", queryErr.Source, queryErr.Label); err != nil {
				return err
			}
		}
		for _, finding := range report.Findings {
			if _, err := fmt.Fprintf(w, "finding: %s:%s node=%d operator=%q reasons=%s fingerprint=%s\n",
				finding.Source,
				finding.Label,
				finding.Finding.NodeIndex,
				finding.Finding.DisplayName,
				strings.Join(finding.Finding.Reasons, ","),
				finding.Finding.Fingerprint,
			); err != nil {
				return err
			}
		}
		for _, failure := range report.ExpectationFailures {
			if failure.Pattern == 0 {
				if _, err := fmt.Fprintf(w, "expectation_failure: %s reason=%s\n", failure.Label, failure.Reason); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprintf(w, "expectation_failure: %s pattern=%d reason=%s\n", failure.Label, failure.Pattern, failure.Reason); err != nil {
				return err
			}
		}
		return nil
	case "json":
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	default:
		return fmt.Errorf("unsupported --format %q; use text or json", format)
	}
}
