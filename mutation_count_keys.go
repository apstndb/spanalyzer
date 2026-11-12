package spanalyzer

import (
	"encoding/json"
	"strings"

	spannerpb "cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/cloudspannerecosystem/memefish/ast"
	"google.golang.org/protobuf/types/known/structpb"
)

// mutationDMLPredicateKey collects mandatory unqualified column = literal
// predicates. It deliberately does not evaluate predicates, infer cardinality,
// use old-row assertions as keys, or search inside OR/NOT/subqueries. An AND's
// remaining terms can only narrow the rows selected by these equalities.
func mutationDMLPredicateKey(where *ast.Where) map[string]interface{} {
	key := map[string]interface{}{}
	var visit func(ast.Expr)
	visit = func(expr ast.Expr) {
		switch expr := expr.(type) {
		case *ast.ParenExpr:
			visit(expr.Expr)
		case *ast.BinaryExpr:
			if string(expr.Op) == "AND" {
				visit(expr.Left)
				visit(expr.Right)
			} else if string(expr.Op) == "=" {
				for _, pair := range [][2]ast.Expr{{expr.Left, expr.Right}, {expr.Right, expr.Left}} {
					column, ok := pair[0].(*ast.Ident)
					value, literal := mutationLiteralValue(pair[1])
					if ok && literal {
						// Any one equality is a necessary condition, even if
						// another term contradicts it and no row can match.
						key[strings.ToLower(column.Name)] = value
					}
				}
			}
		}
	}
	if where != nil {
		visit(where.Expr)
	}
	return key
}

// mutationStageKeysDisjoint proves only an upper-bound exception to the
// relationship guard. FK/interleave effects, queues, and schema history remain
// outside this proof, even when the explicitly written row keys differ.
func mutationStageKeysDisjoint(s *mutationCountSchemaModel, dml, buffered []mutationCountOperation, related map[string]bool) bool {
	isolated := map[string]bool{}
	for name, table := range s.Tables {
		isolated[name] = !table.Queue && table.Parent == "" && len(table.Unknown) == 0
	}
	for _, table := range s.Tables {
		if table.Parent != "" {
			isolated[strings.ToLower(table.Parent)] = false
		}
	}
	for _, fk := range s.ForeignKeys {
		isolated[strings.ToLower(fk.Table)] = false
		isolated[strings.ToLower(fk.Referenced)] = false
	}
	keys := map[string]map[string]bool{}
	for _, op := range buffered {
		name := strings.ToLower(op.Table)
		if related[name] {
			if !isolated[name] {
				return false
			}
			keys[name] = map[string]bool{}
		}
	}
	for _, op := range dml {
		if tableKeys, needed := keys[strings.ToLower(op.Table)]; needed {
			key, ok := mutationStagePrimaryKey(s, op)
			if !ok {
				return false
			}
			tableKeys[key] = true
		}
	}
	for _, op := range buffered {
		if tableKeys, needed := keys[strings.ToLower(op.Table)]; needed {
			key, ok := mutationStagePrimaryKey(s, op)
			if !ok || len(tableKeys) == 0 || tableKeys[key] {
				return false
			}
		}
	}
	return true
}

func mutationStagePrimaryKey(s *mutationCountSchemaModel, op mutationCountOperation) (string, bool) {
	values := op.Values
	if op.Channel == "dml" && (op.Kind == "update" || op.Kind == "delete") {
		values = op.PredicateKey
	} else if op.Kind == "delete" && !op.DeleteKey {
		return "", false
	}
	table := s.Tables[strings.ToLower(op.Table)]
	pks := mutationPrimaryKeys(table.Table)
	if len(pks) == 0 {
		return "", false
	}
	key := make([]interface{}, 0, len(pks))
	for _, name := range pks {
		column, _ := table.Table.Column(name)
		value, known := values[strings.ToLower(name)]
		if !known || column == nil || column.Type == nil || table.Generated[strings.ToLower(name)] != "" {
			return "", false
		}
		// Limit equality to scalar encodings whose SQL literals and wire
		// values have matching semantics. In particular, do not compare raw
		// float JSON (-0/NaN), timestamp precision, or SQL bytes as base64.
		switch column.Type.Code {
		case spannerpb.TypeCode_INT64, spannerpb.TypeCode_STRING, spannerpb.TypeCode_BOOL, spannerpb.TypeCode_DATE:
		default:
			return "", false
		}
		wire, err := structpb.NewValue(value)
		if err != nil {
			return "", false
		}
		normalized, err := mutationProtoValue(column.Type, wire, column.NotNull, false)
		if err != nil {
			return "", false
		}
		key = append(key, normalized)
	}
	encoded, err := json.Marshal(key)
	return string(encoded), err == nil
}
