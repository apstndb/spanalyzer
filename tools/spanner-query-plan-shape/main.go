package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/apstndb/spannerplan/plantree/reference"
	"github.com/goccy/go-yaml"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/apstndb/spanemuboost"
	"github.com/cloudspannerecosystem/memefish"
	"google.golang.org/protobuf/encoding/protojson"
)

const singersDDL = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName STRING(MAX),
  Rating INT64 NOT NULL,
  Active BOOL
) PRIMARY KEY (SingerId)
`

const albumsDDL = `
CREATE TABLE Albums (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  Title STRING(MAX)
) PRIMARY KEY (SingerId, AlbumId)
`

const pushBroadcastSQL = `
SELECT s.SingerId, a.AlbumId
FROM Singers s
JOIN@{JOIN_METHOD=PUSH_BROADCAST_HASH_JOIN} Albums a
ON s.SingerId = a.SingerId
`

const hashSQL = `
SELECT s.SingerId, a.AlbumId
FROM Singers s
JOIN@{JOIN_METHOD=HASH_JOIN} Albums a
ON s.SingerId = a.SingerId
`

type stringListFlag []string

func (f *stringListFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

type queryCase struct {
	Label    string
	SQL      string
	PlanMode planMode
}

type planMode string

const (
	planModeAuto      planMode = ""
	planModeReadOnly  planMode = "read_only"
	planModeReadWrite planMode = "read_write"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(args []string, stdout io.Writer) error {
	var ddlFiles stringListFlag
	var sqlTexts stringListFlag
	var sqlFiles stringListFlag
	fs := flag.NewFlagSet("spanner-query-plan-shape", flag.ContinueOnError)
	fs.Var(&ddlFiles, "ddl", "Spanner DDL file to load; may be repeated. Defaults to built-in Singers/Albums DDL")
	fs.Var(&sqlTexts, "sql", "SQL text to analyze; may be repeated. Overrides --case built-ins when present")
	fs.Var(&sqlFiles, "sql-file", "SQL file to analyze; may be repeated. Overrides --case built-ins when present")
	builtinCase := fs.String("case", "all", "built-in query case when --sql/--sql-file is omitted: all, docs, optimizer_gaps, optimizer_unhinted_candidates, cte, dml, tvf, lock_hints, full_text_search, function_hint, hint_matrix, statement_hint_query_matrix, join_matrix, subquery_join_hint_matrix, push_broadcast_hash_join, or hash_join")
	output := fs.String("output", "nodes", "output format: compact-dfs, compact-dfs-metadata, compact-tree, compact-tree-metadata, json, nodes, reference, summary, yaml, or legacy aliases compact/compact-metadata")
	compactTreeIndexes := fs.Bool("compact-tree-indexes", false, "include PlanNode indexes in compact-tree and compact-tree-metadata output")
	optimizerVersionMatrix := fs.Bool("optimizer-version-matrix", false, "expand each query with OPTIMIZER_VERSION statement hints for versions 1 through 8")
	optimizerVersionDiff := fs.Bool("optimizer-version-diff", false, "analyze each query with optimizer versions 1 through 8 and print only queries whose compact-tree-metadata shape changes")
	allowDistributedMergeMatrix := fs.Bool("allow-distributed-merge-matrix", false, "expand each query across ALLOW_DISTRIBUTED_MERGE unspecified, TRUE, and FALSE")
	continueOnError := fs.Bool("continue-on-error", false, "print errors and continue analyzing remaining queries")
	timeout := fs.Duration("timeout", 5*time.Minute, "maximum time to start Spanner Omni and analyze queries")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *optimizerVersionDiff && *optimizerVersionMatrix {
		return fmt.Errorf("--optimizer-version-diff already expands optimizer versions; do not combine it with --optimizer-version-matrix")
	}

	queries, err := loadQueries(*builtinCase, sqlTexts, sqlFiles)
	if err != nil {
		return err
	}
	if len(queries) == 0 {
		return fmt.Errorf("no queries to analyze")
	}
	if *allowDistributedMergeMatrix {
		queries = expandAllowDistributedMergeMatrix(queries)
	}
	if *optimizerVersionMatrix {
		queries = expandOptimizerVersionMatrix(queries)
	}
	ddls, err := loadDDLs(*builtinCase, ddlFiles)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	runtime := spanemuboost.NewLazyRuntime(spanemuboost.BackendOmni)
	defer func() {
		_ = runtime.Close()
	}()
	clients, err := spanemuboost.OpenClients(ctx, runtime,
		spanemuboost.WithRandomDatabaseID(),
		spanemuboost.WithSetupDDLs(ddls),
	)
	if err != nil {
		return err
	}
	defer func() {
		_ = clients.Close()
	}()

	if *optimizerVersionDiff {
		return printOptimizerVersionDiffs(ctx, stdout, clients.Client, queries)
	}
	if isRawPlanOutput(*output) {
		return printRawPlans(ctx, stdout, clients.Client, queries, *output, *continueOnError)
	}

	for i, query := range queries {
		if i > 0 {
			if err := writeln(stdout); err != nil {
				return err
			}
		}
		if err := printPlan(ctx, stdout, clients.Client, query, *output, *compactTreeIndexes); err != nil {
			if *continueOnError {
				if err := writef(stdout, "=== %s ===\n", query.Label); err != nil {
					return err
				}
				if err := writeln(stdout, strings.TrimSpace(query.SQL)); err != nil {
					return err
				}
				if err := writef(stdout, "error: %v\n", err); err != nil {
					return err
				}
				continue
			}
			return err
		}
	}
	return nil
}

func writef(w io.Writer, format string, args ...interface{}) error {
	_, err := fmt.Fprintf(w, format, args...)
	return err
}

func writeln(w io.Writer, args ...interface{}) error {
	_, err := fmt.Fprintln(w, args...)
	return err
}

func isRawPlanOutput(output string) bool {
	switch strings.ToLower(strings.TrimSpace(output)) {
	case "json", "yaml":
		return true
	default:
		return false
	}
}

func loadDDLs(builtinCase string, paths []string) ([]string, error) {
	if len(paths) == 0 {
		if strings.EqualFold(strings.TrimSpace(builtinCase), "dml") {
			return parseBuiltInDDLs("dml-schema.sql", dmlDDL)
		}
		if strings.EqualFold(strings.TrimSpace(builtinCase), "tvf") {
			return parseBuiltInDDLs("tvf-schema.sql", changeStreamTVFDDL)
		}
		if strings.EqualFold(strings.TrimSpace(builtinCase), "lock_hints") {
			return parseBuiltInDDLs("lock-hints-schema.sql", docsDDL)
		}
		if strings.EqualFold(strings.TrimSpace(builtinCase), "full_text_search") {
			return parseBuiltInDDLs("full-text-search-schema.sql", fullTextSearchDDL)
		}
		if strings.EqualFold(strings.TrimSpace(builtinCase), "optimizer_gaps") ||
			strings.EqualFold(strings.TrimSpace(builtinCase), "optimizer_unhinted_candidates") {
			return parseBuiltInDDLs("optimizer-gaps-schema.sql", optimizerGapsDDL)
		}
		if strings.EqualFold(strings.TrimSpace(builtinCase), "docs") ||
			strings.EqualFold(strings.TrimSpace(builtinCase), "cte") ||
			strings.EqualFold(strings.TrimSpace(builtinCase), "function_hint") ||
			strings.EqualFold(strings.TrimSpace(builtinCase), "hint_matrix") ||
			strings.EqualFold(strings.TrimSpace(builtinCase), "statement_hint_query_matrix") ||
			strings.EqualFold(strings.TrimSpace(builtinCase), "join_matrix") ||
			strings.EqualFold(strings.TrimSpace(builtinCase), "subquery_join_hint_matrix") {
			return parseBuiltInDDLs("docs-schema.sql", docsDDL)
		}
		return []string{singersDDL, albumsDDL}, nil
	}
	var out []string
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		ddls, err := memefish.ParseDDLs(path, string(data))
		if err != nil {
			return nil, err
		}
		for _, ddl := range ddls {
			out = append(out, ddl.SQL())
		}
	}
	return out, nil
}

func parseBuiltInDDLs(path, ddlSQL string) ([]string, error) {
	ddls, err := memefish.ParseDDLs(path, ddlSQL)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(ddls))
	for _, ddl := range ddls {
		out = append(out, ddl.SQL())
	}
	return out, nil
}

func loadQueries(builtinCase string, sqlTexts, sqlFiles []string) ([]queryCase, error) {
	var queries []queryCase
	for i, sql := range sqlTexts {
		queries = append(queries, queryCase{
			Label: fmt.Sprintf("custom_sql_%d", i+1),
			SQL:   sql,
		})
	}
	for _, path := range sqlFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		queries = append(queries, queryCase{
			Label: path,
			SQL:   string(data),
		})
	}
	if len(queries) > 0 {
		return queries, nil
	}

	switch strings.ToLower(strings.TrimSpace(builtinCase)) {
	case "all":
		return []queryCase{
			{Label: "PUSH_BROADCAST_HASH_JOIN", SQL: pushBroadcastSQL},
			{Label: "HASH_JOIN", SQL: hashSQL},
		}, nil
	case "docs":
		return docsQueries, nil
	case "optimizer_gaps":
		return optimizerGapQueries, nil
	case "optimizer_unhinted_candidates":
		return optimizerUnhintedCandidateQueries, nil
	case "cte":
		return cteQueries, nil
	case "dml":
		return dmlQueries, nil
	case "tvf":
		return tvfQueries, nil
	case "lock_hints":
		return lockHintQueries, nil
	case "full_text_search":
		return fullTextSearchQueries, nil
	case "function_hint":
		return functionHintQueries, nil
	case "hint_matrix":
		return hintMatrixQueries, nil
	case "statement_hint_query_matrix":
		return statementHintQueryMatrixQueries, nil
	case "join_matrix":
		return joinMatrixQueries, nil
	case "subquery_join_hint_matrix":
		return subqueryJoinHintMatrixQueries, nil
	case "push_broadcast_hash_join":
		return []queryCase{{Label: "PUSH_BROADCAST_HASH_JOIN", SQL: pushBroadcastSQL}}, nil
	case "hash_join":
		return []queryCase{{Label: "HASH_JOIN", SQL: hashSQL}}, nil
	default:
		return nil, fmt.Errorf("unsupported --case %q; use all, docs, optimizer_gaps, optimizer_unhinted_candidates, cte, dml, tvf, lock_hints, full_text_search, function_hint, hint_matrix, statement_hint_query_matrix, join_matrix, subquery_join_hint_matrix, push_broadcast_hash_join, or hash_join", builtinCase)
	}
}

func expandOptimizerVersionMatrix(queries []queryCase) []queryCase {
	out := make([]queryCase, 0, len(queries)*8)
	for _, query := range queries {
		for version := 1; version <= 8; version++ {
			out = append(out, queryCase{
				Label:    fmt.Sprintf("optimizer-version/v%d/%s", version, query.Label),
				SQL:      withOptimizerVersionStatementHint(query.SQL, version),
				PlanMode: query.PlanMode,
			})
		}
	}
	return out
}

func expandAllowDistributedMergeMatrix(queries []queryCase) []queryCase {
	out := make([]queryCase, 0, len(queries)*3)
	for _, query := range queries {
		out = append(out, queryCase{
			Label:    fmt.Sprintf("allow-distributed-merge/default/%s", query.Label),
			SQL:      query.SQL,
			PlanMode: query.PlanMode,
		})
		out = append(out, queryCase{
			Label:    fmt.Sprintf("allow-distributed-merge/true/%s", query.Label),
			SQL:      withStatementHintAssignment(query.SQL, "ALLOW_DISTRIBUTED_MERGE=TRUE"),
			PlanMode: query.PlanMode,
		})
		out = append(out, queryCase{
			Label:    fmt.Sprintf("allow-distributed-merge/false/%s", query.Label),
			SQL:      withStatementHintAssignment(query.SQL, "ALLOW_DISTRIBUTED_MERGE=FALSE"),
			PlanMode: query.PlanMode,
		})
	}
	return out
}

type optimizerVersionShape struct {
	version int
	shape   string
}

func printOptimizerVersionDiffs(ctx context.Context, stdout io.Writer, client *spanner.Client, queries []queryCase) error {
	var printed int
	for _, query := range queries {
		shapes := make([]optimizerVersionShape, 0, 8)
		for version := 1; version <= 8; version++ {
			versioned := queryCase{
				Label:    fmt.Sprintf("%s/v%d", query.Label, version),
				SQL:      withOptimizerVersionStatementHint(query.SQL, version),
				PlanMode: query.PlanMode,
			}
			plan, err := analyzePlan(ctx, client, versioned)
			shape := ""
			if err != nil {
				shape = "ERROR: " + err.Error()
			} else {
				shape = compactPlanTree(plan, true, false)
			}
			shapes = append(shapes, optimizerVersionShape{
				version: version,
				shape:   shape,
			})
		}
		if !optimizerVersionShapesChanged(shapes) {
			continue
		}
		if printed > 0 {
			if err := writeln(stdout); err != nil {
				return err
			}
		}
		if err := printOptimizerVersionShapeDiff(stdout, query, shapes); err != nil {
			return err
		}
		printed++
	}
	if printed == 0 {
		return writeln(stdout, "no optimizer-version-sensitive query shapes found")
	}
	return nil
}

func optimizerVersionShapesChanged(shapes []optimizerVersionShape) bool {
	if len(shapes) <= 1 {
		return false
	}
	first := shapes[0].shape
	for _, shape := range shapes[1:] {
		if shape.shape != first {
			return true
		}
	}
	return false
}

func printOptimizerVersionShapeDiff(stdout io.Writer, query queryCase, shapes []optimizerVersionShape) error {
	if err := writef(stdout, "=== %s ===\n", query.Label); err != nil {
		return err
	}
	if err := writeln(stdout, strings.TrimSpace(query.SQL)); err != nil {
		return err
	}
	for _, group := range groupOptimizerVersionShapes(shapes) {
		if err := writef(stdout, "%s: %s\n", group.label, group.shape); err != nil {
			return err
		}
	}
	return nil
}

type optimizerVersionShapeGroup struct {
	label string
	shape string
}

func groupOptimizerVersionShapes(shapes []optimizerVersionShape) []optimizerVersionShapeGroup {
	if len(shapes) == 0 {
		return nil
	}
	var groups []optimizerVersionShapeGroup
	start := shapes[0].version
	prev := shapes[0]
	for _, current := range shapes[1:] {
		if current.shape == prev.shape && current.version == prev.version+1 {
			prev = current
			continue
		}
		groups = append(groups, optimizerVersionShapeGroup{
			label: optimizerVersionRangeLabel(start, prev.version),
			shape: prev.shape,
		})
		start = current.version
		prev = current
	}
	groups = append(groups, optimizerVersionShapeGroup{
		label: optimizerVersionRangeLabel(start, prev.version),
		shape: prev.shape,
	})
	return groups
}

func optimizerVersionRangeLabel(start, end int) string {
	if start == end {
		return fmt.Sprintf("v%d", start)
	}
	return fmt.Sprintf("v%d-v%d", start, end)
}

func withOptimizerVersionStatementHint(sql string, version int) string {
	return withStatementHintAssignment(sql, fmt.Sprintf("OPTIMIZER_VERSION=%d", version))
}

func withStatementHintAssignment(sql, assignment string) string {
	return withStatementHintAssignments(sql, assignment)
}

func withStatementHintAssignments(sql string, assignments ...string) string {
	trimmed := strings.TrimLeft(sql, " \t\r\n")
	leading := sql[:len(sql)-len(trimmed)]
	nextAssignments := make([]string, 0, len(assignments))
	nextKeys := map[string]bool{}
	for _, assignment := range assignments {
		assignment = strings.TrimSpace(assignment)
		if assignment == "" {
			continue
		}
		nextAssignments = append(nextAssignments, assignment)
		nextKeys[strings.ToUpper(statementHintAssignmentKey(assignment))] = true
	}
	hint, rest, ok := splitLeadingStatementHint(trimmed)
	if !ok {
		return fmt.Sprintf("%s@{%s}\n%s", leading, strings.Join(nextAssignments, ", "), trimmed)
	}
	merged := append([]string{}, nextAssignments...)
	for _, existing := range strings.Split(hint, ",") {
		existing = strings.TrimSpace(existing)
		if existing == "" {
			continue
		}
		if nextKeys[strings.ToUpper(statementHintAssignmentKey(existing))] {
			continue
		}
		merged = append(merged, existing)
	}
	return fmt.Sprintf("%s@{%s}\n%s", leading, strings.Join(merged, ", "), strings.TrimLeft(rest, " \t\r\n"))
}

func statementHintAssignmentKey(assignment string) string {
	key := assignment
	if eq := strings.Index(key, "="); eq >= 0 {
		key = key[:eq]
	}
	return strings.TrimSpace(key)
}

func splitLeadingStatementHint(sql string) (hint string, rest string, ok bool) {
	if !strings.HasPrefix(sql, "@{") {
		return "", sql, false
	}
	end := strings.Index(sql, "}")
	if end < 0 {
		return "", sql, false
	}
	return sql[2:end], sql[end+1:], true
}

func printPlan(ctx context.Context, stdout io.Writer, client *spanner.Client, query queryCase, output string, compactTreeIndexes bool) error {
	plan, err := analyzePlan(ctx, client, query)
	if err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(output)) {
	case "nodes":
		return printPlanNodes(stdout, query, plan)
	case "json":
		return printPlanJSON(stdout, query, plan)
	case "yaml":
		return printPlanYAML(stdout, query, plan)
	case "compact", "compact-dfs":
		return printPlanCompactDFS(stdout, query, plan)
	case "compact-metadata", "compact-dfs-metadata":
		return printPlanCompactDFSMetadata(stdout, query, plan)
	case "compact-tree":
		return printPlanCompactTree(stdout, query, plan, false, compactTreeIndexes)
	case "compact-tree-metadata":
		return printPlanCompactTree(stdout, query, plan, true, compactTreeIndexes)
	case "summary":
		return printPlanSummary(stdout, query, plan)
	case "reference":
		return printPlanReference(stdout, query, plan)
	default:
		return fmt.Errorf("unsupported --output %q; use compact-dfs, compact-dfs-metadata, compact-tree, compact-tree-metadata, json, nodes, reference, summary, yaml, or legacy aliases compact/compact-metadata", output)
	}
}

func analyzePlan(ctx context.Context, client *spanner.Client, query queryCase) (*spannerpb.QueryPlan, error) {
	stmt := spanner.NewStatement(query.SQL)
	switch query.effectivePlanMode() {
	case planModeReadOnly:
		return client.Single().AnalyzeQuery(ctx, stmt)
	case planModeReadWrite:
		var plan *spannerpb.QueryPlan
		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			var err error
			plan, err = txn.AnalyzeQuery(ctx, stmt)
			return err
		})
		if err != nil {
			return nil, err
		}
		return plan, nil
	default:
		return nil, fmt.Errorf("unsupported plan mode %q for %s", query.effectivePlanMode(), query.Label)
	}
}

func (q queryCase) effectivePlanMode() planMode {
	if q.PlanMode != planModeAuto {
		return q.PlanMode
	}
	if isDMLStatement(q.SQL) {
		return planModeReadWrite
	}
	return planModeReadOnly
}

func isDMLStatement(sql string) bool {
	first := firstSQLKeyword(sql)
	return first == "INSERT" || first == "UPDATE" || first == "DELETE"
}

func firstSQLKeyword(sql string) string {
	trimmed := strings.TrimSpace(sql)
	for strings.HasPrefix(trimmed, "@{") {
		end := strings.Index(trimmed, "}")
		if end < 0 {
			break
		}
		trimmed = strings.TrimSpace(trimmed[end+1:])
	}
	for i, r := range trimmed {
		if r != '_' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
			return strings.ToUpper(trimmed[:i])
		}
	}
	return strings.ToUpper(trimmed)
}

func printPlanReference(stdout io.Writer, query queryCase, plan *spannerpb.QueryPlan) error {
	rendered, err := reference.RenderTreeTable(plan.GetPlanNodes(), reference.RenderModePlan, reference.FormatCurrent, 0)
	if err != nil {
		return err
	}
	if err := writef(stdout, "=== %s ===\n", query.Label); err != nil {
		return err
	}
	if err := writeln(stdout, strings.TrimSpace(query.SQL)); err != nil {
		return err
	}
	return writeln(stdout, rendered)
}

func printPlanNodes(stdout io.Writer, query queryCase, plan *spannerpb.QueryPlan) error {
	if err := writef(stdout, "=== %s ===\n", query.Label); err != nil {
		return err
	}
	if err := writeln(stdout, strings.TrimSpace(query.SQL)); err != nil {
		return err
	}
	for _, node := range plan.GetPlanNodes() {
		if err := writef(stdout, "node %d: %s\n", node.GetIndex(), node.GetDisplayName()); err != nil {
			return err
		}
		if short := node.GetShortRepresentation().GetDescription(); short != "" {
			if err := writef(stdout, "  short: %s\n", short); err != nil {
				return err
			}
		}
		if meta := node.GetMetadata(); meta != nil {
			data, err := json.MarshalIndent(meta.AsMap(), "  ", "  ")
			if err != nil {
				return err
			}
			if err := writef(stdout, "  metadata: %s\n", data); err != nil {
				return err
			}
		}
		for _, child := range node.GetChildLinks() {
			if err := writef(stdout, "  child: %s -> %d\n", child.GetType(), child.GetChildIndex()); err != nil {
				return err
			}
		}
	}
	return nil
}

type rawPlanEnvelope struct {
	QueryLabel string          `json:"query_label"`
	SQL        string          `json:"sql"`
	Plan       json.RawMessage `json:"plan,omitempty"`
	Error      string          `json:"error,omitempty"`
}

func printRawPlans(ctx context.Context, stdout io.Writer, client *spanner.Client, queries []queryCase, output string, continueOnError bool) error {
	envelopes := make([]rawPlanEnvelope, 0, len(queries))
	for _, query := range queries {
		plan, err := analyzePlan(ctx, client, query)
		if err != nil {
			if !continueOnError {
				return err
			}
			envelopes = append(envelopes, rawPlanEnvelope{
				QueryLabel: query.Label,
				SQL:        strings.TrimSpace(query.SQL),
				Error:      err.Error(),
			})
			continue
		}
		envelope, err := newRawPlanEnvelope(query, plan)
		if err != nil {
			return err
		}
		envelopes = append(envelopes, envelope)
	}
	data, err := marshalRawPlanEnvelopes(envelopes, output)
	if err != nil {
		return err
	}
	return writeln(stdout, string(data))
}

func printPlanJSON(stdout io.Writer, query queryCase, plan *spannerpb.QueryPlan) error {
	envelope, err := newRawPlanEnvelope(query, plan)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return err
	}
	return writeln(stdout, string(data))
}

func printPlanYAML(stdout io.Writer, query queryCase, plan *spannerpb.QueryPlan) error {
	envelope, err := newRawPlanEnvelope(query, plan)
	if err != nil {
		return err
	}
	data, err := marshalRawPlanEnvelopes([]rawPlanEnvelope{envelope}, "yaml")
	if err != nil {
		return err
	}
	return writeln(stdout, string(data))
}

func newRawPlanEnvelope(query queryCase, plan *spannerpb.QueryPlan) (rawPlanEnvelope, error) {
	planJSON, err := protojson.MarshalOptions{
		Multiline:     true,
		Indent:        "  ",
		UseProtoNames: true,
	}.Marshal(plan)
	if err != nil {
		return rawPlanEnvelope{}, err
	}
	return rawPlanEnvelope{
		QueryLabel: query.Label,
		SQL:        strings.TrimSpace(query.SQL),
		Plan:       json.RawMessage(planJSON),
	}, nil
}

func marshalRawPlanEnvelopes(envelopes []rawPlanEnvelope, output string) ([]byte, error) {
	jsonBytes, err := json.MarshalIndent(envelopes, "", "  ")
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(output)) {
	case "json":
		return jsonBytes, nil
	case "yaml":
		return yaml.JSONToYAML(jsonBytes)
	default:
		return nil, fmt.Errorf("unsupported raw plan output %q", output)
	}
}

func printPlanSummary(stdout io.Writer, query queryCase, plan *spannerpb.QueryPlan) error {
	if err := writef(stdout, "=== %s ===\n", query.Label); err != nil {
		return err
	}
	if err := writeln(stdout, strings.TrimSpace(query.SQL)); err != nil {
		return err
	}
	for _, node := range plan.GetPlanNodes() {
		if err := writef(stdout, "node %d: %s", node.GetIndex(), node.GetDisplayName()); err != nil {
			return err
		}
		if short := node.GetShortRepresentation().GetDescription(); short != "" {
			if err := writef(stdout, " | %s", short); err != nil {
				return err
			}
		}
		if err := writeln(stdout); err != nil {
			return err
		}
	}
	return nil
}

func printPlanCompactDFS(stdout io.Writer, query queryCase, plan *spannerpb.QueryPlan) error {
	return writef(stdout, "%s: %s\n", query.Label, strings.Join(compactPlanDFSOperators(plan, false), " > "))
}

func printPlanCompactDFSMetadata(stdout io.Writer, query queryCase, plan *spannerpb.QueryPlan) error {
	return writef(stdout, "%s: %s\n", query.Label, strings.Join(compactPlanDFSOperators(plan, true), " > "))
}

func compactPlanDFSOperators(plan *spannerpb.QueryPlan, includeMetadata bool) []string {
	if plan == nil {
		return nil
	}
	nodesByIndex := compactTreeNodesByIndex(plan)
	var operators []string
	seen := map[string]bool{}
	visited := map[int32]bool{}
	var walk func(node *spannerpb.PlanNode)
	walk = func(node *spannerpb.PlanNode) {
		if !compactTreeRootVisible(node) || visited[node.GetIndex()] {
			return
		}
		visited[node.GetIndex()] = true
		operator := compactTreeNodeLabel(node, nodesByIndex, includeMetadata, false)
		if !seen[operator] {
			seen[operator] = true
			operators = append(operators, operator)
		}
		for _, child := range compactTreeChildren(node, nodesByIndex) {
			walk(child.node)
		}
	}
	for _, root := range compactTreeRoots(plan, nodesByIndex) {
		walk(root)
	}
	return operators
}

func printPlanCompactTree(stdout io.Writer, query queryCase, plan *spannerpb.QueryPlan, includeMetadata, includeIndexes bool) error {
	return writef(stdout, "%s: %s\n", query.Label, compactPlanTree(plan, includeMetadata, includeIndexes))
}

func compactPlanTree(plan *spannerpb.QueryPlan, includeMetadata, includeIndexes bool) string {
	if plan == nil {
		return ""
	}
	nodesByIndex := compactTreeNodesByIndex(plan)
	roots := compactTreeRoots(plan, nodesByIndex)
	rendered := make([]string, 0, len(roots))
	for _, root := range roots {
		rendered = append(rendered, compactTreeRenderNode(root, nodesByIndex, includeMetadata, includeIndexes, map[int32]bool{}))
	}
	return strings.Join(rendered, " | ")
}

func compactTreeNodesByIndex(plan *spannerpb.QueryPlan) map[int32]*spannerpb.PlanNode {
	nodesByIndex := make(map[int32]*spannerpb.PlanNode, len(plan.GetPlanNodes()))
	for _, node := range plan.GetPlanNodes() {
		nodesByIndex[node.GetIndex()] = node
	}
	return nodesByIndex
}

func compactTreeRoots(plan *spannerpb.QueryPlan, nodesByIndex map[int32]*spannerpb.PlanNode) []*spannerpb.PlanNode {
	referenced := map[int32]bool{}
	for _, node := range plan.GetPlanNodes() {
		for _, link := range node.GetChildLinks() {
			if child := nodesByIndex[link.GetChildIndex()]; compactTreeRootVisible(child) {
				referenced[child.GetIndex()] = true
			}
		}
	}
	var roots []*spannerpb.PlanNode
	for _, node := range plan.GetPlanNodes() {
		if !compactTreeRootVisible(node) || referenced[node.GetIndex()] {
			continue
		}
		roots = append(roots, node)
	}
	if len(roots) == 0 {
		for _, node := range plan.GetPlanNodes() {
			if compactTreeRootVisible(node) {
				roots = append(roots, node)
				break
			}
		}
	}
	return roots
}

type compactTreeChild struct {
	linkType string
	node     *spannerpb.PlanNode
}

func compactTreeRenderNode(node *spannerpb.PlanNode, nodesByIndex map[int32]*spannerpb.PlanNode, includeMetadata, includeIndexes bool, path map[int32]bool) string {
	if node == nil {
		return ""
	}
	label := compactTreeNodeLabel(node, nodesByIndex, includeMetadata, includeIndexes)
	if path[node.GetIndex()] {
		return label + "..."
	}
	nextPath := make(map[int32]bool, len(path)+1)
	for index := range path {
		nextPath[index] = true
	}
	nextPath[node.GetIndex()] = true

	children := compactTreeChildren(node, nodesByIndex)
	if len(children) == 0 {
		return label
	}
	if len(children) == 1 {
		return label + " " + compactTreeLinkArrow(children[0].linkType) + " " + compactTreeRenderNode(children[0].node, nodesByIndex, includeMetadata, includeIndexes, nextPath)
	}
	rendered := make([]string, 0, len(children))
	for _, child := range children {
		childText := compactTreeRenderNode(child.node, nodesByIndex, includeMetadata, includeIndexes, nextPath)
		rendered = append(rendered, compactTreeLinkArrow(child.linkType)+" "+childText)
	}
	return label + "(" + strings.Join(rendered, ", ") + ")"
}

func compactTreeNodeLabel(node *spannerpb.PlanNode, nodesByIndex map[int32]*spannerpb.PlanNode, includeMetadata, includeIndexes bool) string {
	label := node.GetDisplayName()
	if includeMetadata {
		label = compactMetadataOperator(node, nodesByIndex)
	}
	if includeIndexes {
		label = fmt.Sprintf("%d:%s", node.GetIndex(), label)
	}
	return label
}

func compactTreeChildren(node *spannerpb.PlanNode, nodesByIndex map[int32]*spannerpb.PlanNode) []compactTreeChild {
	children := make([]compactTreeChild, 0, len(node.GetChildLinks()))
	for _, link := range node.GetChildLinks() {
		if !compactTreeLinkVisible(link, nodesByIndex) {
			continue
		}
		child := nodesByIndex[link.GetChildIndex()]
		children = append(children, compactTreeChild{
			linkType: strings.TrimSpace(link.GetType()),
			node:     child,
		})
	}
	return children
}

func compactTreeRootVisible(node *spannerpb.PlanNode) bool {
	if node == nil {
		return false
	}
	switch node.GetKind() {
	case spannerpb.PlanNode_RELATIONAL:
		return true
	case spannerpb.PlanNode_SCALAR:
		return false
	}
	switch node.GetDisplayName() {
	case "", "Array Constructor", "Constant", "Field", "Function", "Parameter", "Reference", "Search Predicate", "Struct Constructor":
		return false
	default:
		return true
	}
}

func compactTreeLinkVisible(link *spannerpb.PlanNode_ChildLink, nodesByIndex map[int32]*spannerpb.PlanNode) bool {
	if link == nil {
		return false
	}
	child := nodesByIndex[link.GetChildIndex()]
	if child == nil {
		return false
	}
	if strings.TrimSpace(link.GetType()) == "Scalar" {
		return true
	}
	return compactTreeRootVisible(child)
}

func compactTreeLinkArrow(linkType string) string {
	linkType = strings.TrimSpace(linkType)
	if linkType == "" {
		return "->"
	}
	return "-[" + linkType + "]->"
}

func compactMetadataOperator(node *spannerpb.PlanNode, nodesByIndex map[int32]*spannerpb.PlanNode) string {
	var annotations []string
	if meta := compactMetadataAnnotations(node); len(meta) > 0 {
		annotations = append(annotations, strings.Join(meta, ", "))
	}
	if scalarLinks := compactHiddenScalarChildAnnotations(node, nodesByIndex); len(scalarLinks) > 0 {
		annotations = append(annotations, strings.Join(scalarLinks, ", "))
	}
	if len(annotations) == 0 {
		return node.GetDisplayName()
	}
	return fmt.Sprintf("%s{%s}", node.GetDisplayName(), strings.Join(annotations, "; "))
}

func compactMetadataAnnotations(node *spannerpb.PlanNode) []string {
	meta := node.GetMetadata()
	if meta == nil {
		return nil
	}
	values := meta.AsMap()
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out []string
	for _, key := range keys {
		value := values[key]
		formatted, ok := formatCompactMetadataValue(value)
		if !ok {
			continue
		}
		out = append(out, compactMetadataName(key)+"="+formatted)
	}
	return out
}

func compactMetadataName(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, " ", "_")
	key = strings.ReplaceAll(key, "-", "_")
	return key
}

func formatCompactMetadataValue(value interface{}) (string, bool) {
	switch v := value.(type) {
	case bool:
		return fmt.Sprintf("%t", v), true
	case string:
		return v, true
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v)), true
		}
		return fmt.Sprintf("%g", v), true
	case nil:
		return "", false
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return "", false
		}
		return string(data), true
	}
}
