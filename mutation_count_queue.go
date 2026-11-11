package spanalyzer

import (
	"fmt"
	"strings"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
	"github.com/cloudspannerecosystem/memefish/token"
)

// memefish v0.8.1 has no queue DDL AST. This write-only adapter recognizes
// statement-leading CREATE QUEUE tokens and reuses CREATE TABLE's shared
// column/key grammar. It never rewrites strings, comments or identifiers and
// does not register queue TVFs in the ordinary query analyzer. See the write
// analysis design's upstream constraints section before extending this grammar:
// knowledge/research/write-analysis/mutation-count-alternatives-20261110.md#upstream-constraints-and-workarounds.
func prepareMutationCountDDL(ddl string) (string, map[token.Pos]bool, error) {
	lexer := &memefish.Lexer{File: &token.File{Buffer: ddl}}
	buffer := []byte(ddl)
	starts := map[token.Pos]bool{}
	var prefix []token.Token
	for {
		if err := lexer.NextToken(); err != nil {
			return "", nil, err
		}
		t := lexer.Token
		if t.Kind == token.TokenEOF {
			break
		}
		if t.Kind == ";" {
			prefix = nil
			continue
		}
		if len(prefix) >= 2 {
			continue
		}
		prefix = append(prefix, t)
		if len(prefix) == 2 && strings.EqualFold(prefix[0].Raw, "CREATE") && strings.EqualFold(t.Raw, "QUEUE") {
			copy(buffer[int(t.Pos):int(t.End)], "TABLE")
			starts[prefix[0].Pos] = true
		}
	}
	return string(buffer), starts, nil
}

// AnalyzeQueueRead recognizes RECEIVE and RENEWLEASE queries without treating
// their internal effects as a caller Mutation API commit. These TVFs return no
// CommitStats; the accounting interval is explicitly unknown, not zero.
func (a *MutationCountAnalyzer) AnalyzeQueueRead(sql string) (MutationCountReport, error) {
	n, err := memefish.ParseQuery("queue.sql", sql)
	if err != nil {
		return MutationCountReport{}, err
	}
	var operations []MutationCountEstimate
	var lookupErr error
	ast.Inspect(n, func(n ast.Node) bool {
		call, ok := n.(*ast.TVFCallExpr)
		if !ok || len(call.Name.Idents) != 1 {
			return true
		}
		name := call.Name.Idents[0].Name
		for _, prefix := range []string{"RECEIVE_", "RENEWLEASE_"} {
			if len(name) > len(prefix) && strings.EqualFold(name[:len(prefix)], prefix) {
				queue := name[len(prefix):]
				t := a.schema.Tables[strings.ToLower(queue)]
				if t == nil || !t.Queue {
					lookupErr = fmt.Errorf("unknown queue %q", queue)
					return false
				}
				operations = append(operations, MutationCountEstimate{Strategy: "bounds", Table: t.Table.Name.String(), Operation: strings.TrimSuffix(strings.ToLower(prefix), "_"), Channel: "queue_tvf", Count: mutationUncertain(0, nil), Expression: "internal queue delivery/lease effects are not exposed through caller CommitStats", Classification: "unknown_upper", Reasons: []string{"queue_tvf_has_no_caller_commit_stats"}, IndexSource: a.schema.IndexSource, LimitScope: "not_caller_mutation_commit", LimitVerdict: "indeterminate"})
			}
		}
		return true
	})
	if lookupErr != nil {
		return MutationCountReport{}, lookupErr
	}
	if len(operations) == 0 {
		return MutationCountReport{}, fmt.Errorf("query has no recognized queue RECEIVE or RENEWLEASE call")
	}
	return MutationCountReport{Version: "spanalyzer.mutation-count/v0alpha1", SchemaSHA256: a.SchemaSHA256(), InputSHA256: digestMutationInput("queue_query", []byte(sql)), Scope: "queue_internal_queries", Bounds: MutationCountAnalysis{Strategy: "bounds", Count: mutationUncertain(0, nil), LimitScope: "not_caller_mutation_commit", LimitVerdict: "indeterminate", Results: operations, OperationCount: len(operations)}, Effective: mutationUncertain(0, nil), EffectiveBasis: "queue_query_effects_not_exposed_by_commit_stats"}, nil
}

func markMutationQueue(t *mutationCountTableModel) error {
	keys := mutationPrimaryKeys(t.Table)
	if len(keys) == 0 {
		return fmt.Errorf("queue primary key is required")
	}
	var payload *Column
	for _, c := range t.Table.Columns {
		if strings.EqualFold(c.Name, "Payload") {
			payload = c
			continue
		}
		if !mutationContainsFold(keys, c.Name) {
			return fmt.Errorf("queue column %s must be part of its primary key", c.Name)
		}
		if c.Hidden && c.Name == "rowid" {
			return fmt.Errorf("queue primary key must be explicit")
		}
	}
	if payload == nil || !payload.NotNull || mutationContainsFold(keys, payload.Name) {
		return fmt.Errorf("queue requires a non-key NOT NULL Payload column")
	}
	switch payload.Type.Code {
	case spannerpb.TypeCode_BYTES, spannerpb.TypeCode_STRING, spannerpb.TypeCode_JSON, spannerpb.TypeCode_PROTO:
	default:
		return fmt.Errorf("unsupported queue payload type %s", payload.Type.Code)
	}
	t.Queue = true
	t.Table.Columns = append(t.Table.Columns, &Column{Name: "DeliverTime", Type: &TypeSpec{Code: spannerpb.TypeCode_TIMESTAMP}, NotNull: true, DefaultSQL: "queue delivery time (implicit)"})
	// Internal queue bookkeeping is not advertised as ordinary secondary
	// indexes. Do not silently give it an ordinary-table upper bound.
	t.Unknown = append(t.Unknown, "queue_internal_accounting")
	return nil
}
