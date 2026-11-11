package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	spanalyzer "github.com/apstndb/spanalyzer"
	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// decodeMutations accepts the wire representation, not Go SDK private fields.
// Unsupported types fail explicitly: this is not a complete service validator.
func decodeMutations(s *schemaModel, data []byte) ([]writeOperation, error) {
	var messages []json.RawMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("expected a JSON array of google.spanner.v1.Mutation: %w", err)
	}
	var out []writeOperation
	for n, raw := range messages {
		var m spannerpb.Mutation
		if err := protojson.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("mutation %d: %w", n, err)
		}
		ops, err := mutationOperations(s, &m)
		if err != nil {
			return nil, fmt.Errorf("mutation %d: %w", n, err)
		}
		out = append(out, ops...)
	}
	return out, nil
}

func mutationOperations(s *schemaModel, m *spannerpb.Mutation) ([]writeOperation, error) {
	if d := m.GetDelete(); d != nil {
		t := s.Tables[strings.ToLower(d.Table)]
		if t == nil {
			return nil, fmt.Errorf("unknown delete table %q", d.Table)
		}
		ks := d.KeySet
		if ks == nil || (!ks.All && len(ks.Keys)+len(ks.Ranges) == 0) {
			return nil, fmt.Errorf("delete requires a nonempty key set")
		}
		units := int64(len(ks.Keys) + len(ks.Ranges))
		if ks.All {
			units++
		}
		rows := uncertain(0, nil)
		fullKeys := !ks.All && len(ks.Ranges) == 0
		for _, key := range ks.Keys {
			if err := validateKey(t, key); err != nil {
				return nil, err
			}
			fullKeys = fullKeys && len(key.Values) == len(t.Table.PrimaryKey)
		}
		for _, r := range ks.Ranges {
			start := r.GetStartClosed()
			if start == nil {
				start = r.GetStartOpen()
			}
			end := r.GetEndClosed()
			if end == nil {
				end = r.GetEndOpen()
			}
			if start == nil || end == nil {
				return nil, fmt.Errorf("range requires both boundaries (empty boundary means unbounded)")
			}
			if err := validateKey(t, start); err != nil {
				return nil, err
			}
			if err := validateKey(t, end); err != nil {
				return nil, err
			}
		}
		if fullKeys {
			hi := int64(len(ks.Keys))
			rows = uncertain(0, &hi)
		}
		// Overlapping keys/ranges can coalesce. Only a single range/key has a
		// fixed base delete unit in this model; never multiply by matched rows.
		base := uncertain(1, &units)
		return []writeOperation{{Kind: "delete", Table: d.Table, Rows: rows, DeleteUnits: base, Channel: "mutation", Assumptions: []string{"delete range normalization and affected index entries require data"}}}, nil
	}
	var w *spannerpb.Mutation_Write
	kind := ""
	switch m.Operation.(type) {
	case *spannerpb.Mutation_Insert:
		w = m.GetInsert()
		kind = "insert"
	case *spannerpb.Mutation_Update:
		w = m.GetUpdate()
		kind = "update"
	case *spannerpb.Mutation_InsertOrUpdate:
		w = m.GetInsertOrUpdate()
		kind = "upsert"
	case *spannerpb.Mutation_Replace:
		w = m.GetReplace()
		kind = "replace"
	default:
		return nil, fmt.Errorf("unsupported or absent mutation operation")
	}
	if w == nil {
		return nil, fmt.Errorf("missing write")
	}
	t := s.Tables[strings.ToLower(w.Table)]
	if t == nil {
		return nil, fmt.Errorf("unknown writable table %q", w.Table)
	}
	columns := map[string]bool{}
	for _, name := range w.Columns {
		k := strings.ToLower(name)
		if columns[k] {
			return nil, fmt.Errorf("duplicate column %q", name)
		}
		columns[k] = true
		c, _ := t.Table.Column(name)
		if c == nil {
			return nil, fmt.Errorf("unknown column %q", name)
		}
		if _, ok := t.Generated[k]; ok || t.Virtual[k] {
			return nil, fmt.Errorf("cannot write generated column %q", name)
		}
	}
	for _, pk := range primaryKeys(t.Table) {
		k := strings.ToLower(pk)
		if !columns[k] && t.Defaults[k] == "" && t.Generated[k] == "" {
			return nil, fmt.Errorf("missing primary key %q", pk)
		}
	}
	if kind == "insert" || kind == "replace" {
		for _, c := range t.Table.Columns {
			k := strings.ToLower(c.Name)
			if c.NotNull && !columns[k] && t.Defaults[k] == "" && t.Generated[k] == "" && !t.Virtual[k] {
				return nil, fmt.Errorf("missing NOT NULL column %q", c.Name)
			}
		}
	}
	if len(w.Values) == 0 {
		return nil, fmt.Errorf("write contains no rows")
	}
	var out []writeOperation
	for row, values := range w.Values {
		if values == nil {
			return nil, fmt.Errorf("row %d: missing value list", row)
		}
		if len(values.Values) != len(w.Columns) {
			return nil, fmt.Errorf("row %d: got %d values for %d columns", row, len(values.Values), len(w.Columns))
		}
		op := writeOperation{Kind: kind, Table: w.Table, Columns: w.Columns, Rows: fixed(1), Values: map[string]interface{}{}, Channel: "mutation"}
		for i, name := range w.Columns {
			c, _ := t.Table.Column(name)
			v, err := protoValue(c.Type, values.Values[i], c.NotNull, c.AllowCommitTimestamp)
			if err != nil {
				return nil, fmt.Errorf("row %d column %s: %w", row, name, err)
			}
			op.Values[strings.ToLower(name)] = v
		}
		out = append(out, op)
	}
	return out, nil
}

func validateKey(t *tableModel, key *structpb.ListValue) error {
	if key == nil {
		return fmt.Errorf("missing key")
	}
	if len(key.Values) > len(t.Table.PrimaryKey) {
		return fmt.Errorf("key has more values than primary key")
	}
	for n, v := range key.Values {
		c, _ := t.Table.Column(t.Table.PrimaryKey[n].Name)
		if _, err := protoValue(c.Type, v, c.NotNull, false); err != nil {
			return fmt.Errorf("key column %s: %w", c.Name, err)
		}
	}
	return nil
}

func validateRowFacts(s *schemaModel, op writeOperation, f *rowFacts) error {
	t := s.Tables[strings.ToLower(op.Table)]
	if f.Rows != nil && *f.Rows < 0 {
		return fmt.Errorf("negative affected rows")
	}
	normalized := map[string]interface{}{}
	for name, value := range f.Old {
		c, _ := t.Table.Column(name)
		if c == nil {
			return fmt.Errorf("unknown old-row column %q", name)
		}
		p, err := structpb.NewValue(value)
		if err != nil {
			return err
		}
		v, err := protoValue(c.Type, p, c.NotNull, false)
		if err != nil {
			return fmt.Errorf("old-row column %s: %w", name, err)
		}
		key := strings.ToLower(name)
		if _, ok := normalized[key]; ok {
			return fmt.Errorf("duplicate old-row column %q", name)
		}
		normalized[key] = v
	}
	f.Old = normalized
	for _, counts := range []map[string]int64{f.IndexRows, f.CascadeIndexRows} {
		for name, n := range counts {
			if n < 0 {
				return fmt.Errorf("negative index rows for %s", name)
			}
		}
	}
	return nil
}

func protoValue(t *spanalyzer.TypeSpec, v *structpb.Value, notNull, commitTimestamp bool) (interface{}, error) {
	if v == nil || v.Kind == nil {
		return nil, fmt.Errorf("missing value")
	}
	if _, ok := v.Kind.(*structpb.Value_NullValue); ok {
		if notNull {
			return nil, fmt.Errorf("NULL for NOT NULL column")
		}
		return nil, nil
	}
	switch t.Code {
	case spannerpb.TypeCode_INT64:
		x, ok := v.Kind.(*structpb.Value_StringValue)
		if !ok {
			return nil, fmt.Errorf("INT64 must use a decimal JSON string")
		}
		n, e := strconv.ParseInt(x.StringValue, 10, 64)
		if e != nil {
			return nil, fmt.Errorf("invalid INT64: %w", e)
		}
		return strconv.FormatInt(n, 10), nil
	case spannerpb.TypeCode_STRING, spannerpb.TypeCode_BYTES, spannerpb.TypeCode_TIMESTAMP, spannerpb.TypeCode_DATE:
		x, ok := v.Kind.(*structpb.Value_StringValue)
		if !ok {
			return nil, fmt.Errorf("expected JSON string for %s", t.Code)
		}
		s := x.StringValue
		switch t.Code {
		case spannerpb.TypeCode_BYTES:
			b, e := base64.StdEncoding.DecodeString(s)
			if e != nil {
				return nil, fmt.Errorf("invalid base64")
			}
			if t.Length != nil && int64(len(b)) > *t.Length {
				return nil, fmt.Errorf("BYTES length exceeded")
			}
		case spannerpb.TypeCode_STRING:
			if t.Length != nil && int64(utf8.RuneCountInString(s)) > *t.Length {
				return nil, fmt.Errorf("STRING length exceeded")
			}
		case spannerpb.TypeCode_TIMESTAMP:
			if s == "spanner.commit_timestamp()" {
				if !commitTimestamp {
					return nil, fmt.Errorf("commit timestamp not allowed")
				}
			} else if _, e := time.Parse(time.RFC3339Nano, s); e != nil {
				return nil, fmt.Errorf("invalid timestamp")
			}
		case spannerpb.TypeCode_DATE:
			if _, e := time.Parse("2006-01-02", s); e != nil {
				return nil, fmt.Errorf("invalid date")
			}
		}
		return s, nil
	case spannerpb.TypeCode_BOOL:
		x, ok := v.Kind.(*structpb.Value_BoolValue)
		if !ok {
			return nil, fmt.Errorf("expected BOOL")
		}
		return x.BoolValue, nil
	case spannerpb.TypeCode_FLOAT32, spannerpb.TypeCode_FLOAT64:
		if x, ok := v.Kind.(*structpb.Value_NumberValue); ok {
			return x.NumberValue, nil
		}
		if x, ok := v.Kind.(*structpb.Value_StringValue); ok && (x.StringValue == "NaN" || x.StringValue == "Infinity" || x.StringValue == "-Infinity") {
			return x.StringValue, nil
		}
		return nil, fmt.Errorf("expected floating-point number")
	case spannerpb.TypeCode_ARRAY:
		x, ok := v.Kind.(*structpb.Value_ListValue)
		if !ok {
			return nil, fmt.Errorf("expected ARRAY")
		}
		var out []interface{}
		for _, a := range x.ListValue.Values {
			n, e := protoValue(t.ArrayElement, a, false, false)
			if e != nil {
				return nil, e
			}
			out = append(out, n)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("type %s is outside this prototype's validator", t.Code)
	}
}

func dmlOperations(s *schemaModel, sql string) ([]writeOperation, error) {
	n, err := memefish.ParseDML("write.sql", sql)
	if err != nil {
		return nil, err
	}
	base := writeOperation{Channel: "dml", Rows: uncertain(0, nil), Values: map[string]interface{}{}, Assumptions: []string{"DML expression semantics and state constraints are validated by the service, not this parser"}}
	switch n := n.(type) {
	case *ast.Insert:
		if n.OnConflict != nil || string(n.InsertOrType) != "" {
			return nil, fmt.Errorf("DML conflict clauses require a separate write model")
		}
		base.Kind = "insert"
		base.Table = pathName(n.TableName)
		v, ok := n.Input.(*ast.ValuesInput)
		if !ok {
			return nil, fmt.Errorf("INSERT SELECT cardinality is not implemented in this prototype")
		}
		var out []writeOperation
		for _, row := range v.Rows {
			if len(row.Exprs) != len(n.Columns) {
				return nil, fmt.Errorf("INSERT values/columns arity mismatch")
			}
			op := base
			op.Rows = fixed(1)
			op.Values = map[string]interface{}{}
			for i, e := range row.Exprs {
				if e.Default {
					continue
				}
				c := n.Columns[i].Name
				op.Columns = append(op.Columns, c)
				if value, ok := literalValue(e.Expr); ok {
					op.Values[strings.ToLower(c)] = value
				}
			}
			out = append(out, op)
		}
		if err := checkDMLColumns(s, out); err != nil {
			return nil, err
		}
		return out, nil
	case *ast.Update:
		base.Kind = "update"
		base.Table = pathName(n.TableName)
		for _, item := range n.Updates {
			u, ok := item.(*ast.UpdateItemSetValue)
			if !ok || len(u.Path) != 1 {
				return nil, fmt.Errorf("nested or array UPDATE requires a separate model")
			}
			c := u.Path[0].Name
			base.Columns = append(base.Columns, c)
			if u.DefaultExpr.Default {
				if t := s.Tables[strings.ToLower(base.Table)]; t != nil {
					if v, ok := constantValue(t.Defaults[strings.ToLower(c)]); ok {
						base.Values[strings.ToLower(c)] = v
					}
				}
			} else if v, ok := literalValue(u.DefaultExpr.Expr); ok {
				base.Values[strings.ToLower(c)] = v
			}
		}
	case *ast.Delete:
		base.Kind = "delete"
		base.Table = pathName(n.TableName)
		base.DeleteUnits = uncertain(0, nil)
	default:
		return nil, fmt.Errorf("DML kind is outside this prototype")
	}
	if err := checkDMLColumns(s, []writeOperation{base}); err != nil {
		return nil, err
	}
	return []writeOperation{base}, nil
}
func checkDMLColumns(s *schemaModel, ops []writeOperation) error {
	for _, op := range ops {
		t := s.Tables[strings.ToLower(op.Table)]
		if t == nil {
			return fmt.Errorf("unknown table %q", op.Table)
		}
		for _, name := range op.Columns {
			c, _ := t.Table.Column(name)
			if c == nil {
				return fmt.Errorf("unknown column %q", name)
			}
			k := strings.ToLower(name)
			if t.Generated[k] != "" || t.Virtual[k] {
				return fmt.Errorf("generated DML target %q", name)
			}
			if op.Kind == "update" && containsFold(primaryKeys(t.Table), name) {
				return fmt.Errorf("DML cannot update a primary key")
			}
		}
	}
	return nil
}

func constantValue(sql string) (interface{}, bool) {
	n, e := memefish.ParseExpr("", sql)
	if e != nil {
		return nil, false
	}
	return literalValue(n)
}
func literalValue(n ast.Expr) (interface{}, bool) {
	switch n := n.(type) {
	case *ast.ParenExpr:
		return literalValue(n.Expr)
	case *ast.StringLiteral:
		return n.Value, true
	case *ast.IntLiteral:
		v, e := strconv.ParseInt(n.Value, n.Base, 64)
		return strconv.FormatInt(v, 10), e == nil
	case *ast.BoolLiteral:
		return n.Value, true
	case *ast.NullLiteral:
		return nil, true
	case *ast.UnaryExpr:
		if string(n.Op) == "-" {
			v, ok := literalValue(n.Expr)
			if s, yes := v.(string); ok && yes {
				i, e := strconv.ParseInt("-"+s, 10, 64)
				return strconv.FormatInt(i, 10), e == nil
			}
		}
	}
	return nil, false
}
func generatedDependsOnWritten(sql string, written map[string]bool, pks []string) bool {
	n, err := memefish.ParseExpr("", sql)
	if err != nil {
		return true
	}
	depends := false
	ast.Inspect(n, func(n ast.Node) bool {
		if i, ok := n.(*ast.Ident); ok {
			depends = depends || (written[strings.ToLower(i.Name)] && !containsFold(pks, i.Name))
		}
		return true
	})
	return depends
}
