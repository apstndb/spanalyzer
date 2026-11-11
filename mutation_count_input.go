package spanalyzer

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// decodeMutationCountInput accepts the wire representation, not Go SDK private fields.
// Unsupported types fail explicitly: this is not a complete service validator.
func decodeMutationCountInput(s *mutationCountSchemaModel, data []byte) ([]mutationCountOperation, error) {
	var messages []json.RawMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("expected a JSON array of google.spanner.v1.Mutation: %w", err)
	}
	var out []mutationCountOperation
	for n, raw := range messages {
		var m spannerpb.Mutation
		if err := protojson.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("mutation %d: %w", n, err)
		}
		ops, err := mutationCountOperations(s, &m)
		if err != nil {
			return nil, fmt.Errorf("mutation %d: %w", n, err)
		}
		out = append(out, ops...)
	}
	return out, nil
}

func mutationCountOperations(s *mutationCountSchemaModel, m *spannerpb.Mutation) ([]mutationCountOperation, error) {
	if send := m.GetSend(); send != nil {
		t := s.Tables[strings.ToLower(send.Queue)]
		if t == nil || !t.Queue {
			return nil, fmt.Errorf("unknown queue %q", send.Queue)
		}
		if send.Key == nil || len(send.Key.Values) != len(t.Table.PrimaryKey) {
			return nil, fmt.Errorf("send requires a complete queue key")
		}
		if err := validateMutationKey(t, send.Key); err != nil {
			return nil, err
		}
		payload, _ := t.Table.Column("Payload")
		v, err := mutationProtoValue(payload.Type, send.Payload, true, false)
		if err != nil {
			return nil, fmt.Errorf("queue payload: %w", err)
		}
		op := mutationCountOperation{Kind: "insert", WireOperation: "send", Table: send.Queue, Rows: mutationFixed(1), Channel: "mutation", Values: map[string]interface{}{"payload": v}, Columns: append(mutationPrimaryKeys(t.Table), payload.Name)}
		for n, pk := range t.Table.PrimaryKey {
			c, _ := t.Table.Column(pk.Name)
			value, err := mutationProtoValue(c.Type, send.Key.Values[n], c.NotNull, false)
			if err != nil {
				return nil, err
			}
			op.Values[strings.ToLower(pk.Name)] = value
		}
		if send.DeliverTime != nil {
			if err := send.DeliverTime.CheckValid(); err != nil {
				return nil, err
			}
			op.Columns = append(op.Columns, "DeliverTime")
			op.Values["delivertime"] = send.DeliverTime.AsTime().UTC().Format(time.RFC3339Nano)
		}
		return []mutationCountOperation{op}, nil
	}
	if ack := m.GetAck(); ack != nil {
		t := s.Tables[strings.ToLower(ack.Queue)]
		if t == nil || !t.Queue {
			return nil, fmt.Errorf("unknown queue %q", ack.Queue)
		}
		if ack.Key == nil || len(ack.Key.Values) != len(t.Table.PrimaryKey) {
			return nil, fmt.Errorf("ack requires a complete queue key")
		}
		if err := validateMutationKey(t, ack.Key); err != nil {
			return nil, err
		}
		hi := int64(1)
		return []mutationCountOperation{{Kind: "delete", WireOperation: "ack", Table: ack.Queue, Rows: mutationUncertain(0, &hi), DeleteUnits: mutationUncertain(0, &hi), Channel: "mutation", Assumptions: []string{"Ack missing-row behavior depends on ignore_not_found; queue internal effects are not ordinary table deletes"}}}, nil
	}
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
		rows := mutationUncertain(0, nil)
		fullKeys := !ks.All && len(ks.Ranges) == 0
		for _, key := range ks.Keys {
			if err := validateMutationKey(t, key); err != nil {
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
			if err := validateMutationKey(t, start); err != nil {
				return nil, err
			}
			if err := validateMutationKey(t, end); err != nil {
				return nil, err
			}
		}
		if fullKeys {
			hi := int64(len(ks.Keys))
			rows = mutationUncertain(0, &hi)
		}
		// Overlapping keys/ranges can coalesce. Only a single range/key has a
		// mutationFixed base delete unit in this model; never mutationMultiply by matched rows.
		base := mutationUncertain(1, &units)
		op := mutationCountOperation{Kind: "delete", Table: d.Table, Rows: rows, DeleteUnits: base, Channel: "mutation", Assumptions: []string{"delete range normalization and affected index entries require data"}, DeleteKey: fullKeys && len(ks.Keys) == 1, Values: map[string]interface{}{}}
		if op.DeleteKey {
			for n, pk := range t.Table.PrimaryKey {
				c, _ := t.Table.Column(pk.Name)
				v, err := mutationProtoValue(c.Type, ks.Keys[0].Values[n], c.NotNull, false)
				if err != nil {
					return nil, err
				}
				op.Values[strings.ToLower(pk.Name)] = v
			}
		}
		return []mutationCountOperation{op}, nil
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
	for _, pk := range mutationPrimaryKeys(t.Table) {
		k := strings.ToLower(pk)
		if !columns[k] && t.Defaults[k] == "" && t.Generated[k] == "" {
			return nil, fmt.Errorf("missing primary key %q", pk)
		}
	}
	if kind == "insert" || kind == "replace" || kind == "upsert" {
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
	var out []mutationCountOperation
	for row, values := range w.Values {
		if values == nil {
			return nil, fmt.Errorf("row %d: missing value list", row)
		}
		if len(values.Values) != len(w.Columns) {
			return nil, fmt.Errorf("row %d: got %d values for %d columns", row, len(values.Values), len(w.Columns))
		}
		op := mutationCountOperation{Kind: kind, Table: w.Table, Columns: w.Columns, Rows: mutationFixed(1), Values: map[string]interface{}{}, Channel: "mutation"}
		for i, name := range w.Columns {
			c, _ := t.Table.Column(name)
			v, err := mutationProtoValue(c.Type, values.Values[i], c.NotNull, c.AllowCommitTimestamp)
			if err != nil {
				return nil, fmt.Errorf("row %d column %s: %w", row, name, err)
			}
			op.Values[strings.ToLower(name)] = v
		}
		out = append(out, op)
	}
	return out, nil
}

func validateMutationKey(t *mutationCountTableModel, key *structpb.ListValue) error {
	if key == nil {
		return fmt.Errorf("missing key")
	}
	if len(key.Values) > len(t.Table.PrimaryKey) {
		return fmt.Errorf("key has more values than primary key")
	}
	for n, v := range key.Values {
		c, _ := t.Table.Column(t.Table.PrimaryKey[n].Name)
		if _, err := mutationProtoValue(c.Type, v, c.NotNull, false); err != nil {
			return fmt.Errorf("key column %s: %w", c.Name, err)
		}
	}
	return nil
}

func validateMutationRowFacts(s *mutationCountSchemaModel, op mutationCountOperation, f *MutationCountRowFacts) error {
	t := s.Tables[strings.ToLower(op.Table)]
	if op.InsertSelect && (f.Exists != nil || len(f.Old) > 0) {
		return fmt.Errorf("INSERT SELECT is one cardinality group; supply Rows/IndexRows instead of a singular old row")
	}
	if f.Rows != nil && *f.Rows < 0 {
		return fmt.Errorf("negative affected rows")
	}
	if f.Rows != nil && (*f.Rows < op.Rows.Lower || op.Rows.Upper != nil && *f.Rows > *op.Rows.Upper) {
		return fmt.Errorf("affected-row fact contradicts normalized row bounds")
	}
	if f.Rows != nil && f.Exists != nil && (op.DeleteKey || op.Channel == "dml" && op.Kind == "update" && !*f.Exists) {
		want := int64(0)
		if *f.Exists {
			want = 1
		}
		if *f.Rows != want {
			return fmt.Errorf("existence contradicts affected-row fact")
		}
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
		v, err := mutationProtoValue(c.Type, p, c.NotNull, false)
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
	known := map[string]bool{}
	for _, i := range s.Indexes[strings.ToLower(op.Table)] {
		known[i.Name] = true
	}
	for name, n := range f.IndexRows {
		if !known[name] {
			return fmt.Errorf("unknown physical index fact %q", name)
		}
		if f.Rows != nil && n > *f.Rows {
			return fmt.Errorf("index rows exceed asserted matched rows for %s", name)
		}
	}
	children := map[string]bool{}
	for _, child := range s.cascadeDescendants(op.Table) {
		for _, i := range s.Indexes[strings.ToLower(child.Table.Name.String())] {
			children[i.Name] = true
		}
	}
	for name := range f.CascadeIndexRows {
		if !children[name] {
			return fmt.Errorf("unknown cascade index fact %q", name)
		}
	}
	for _, counts := range []map[string]int64{f.IndexRows, f.CascadeIndexRows} {
		for name, n := range counts {
			if n < 0 {
				return fmt.Errorf("negative index rows for %s", name)
			}
		}
	}
	return nil
}

func mutationProtoValue(t *TypeSpec, v *structpb.Value, notNull, commitTimestamp bool) (interface{}, error) {
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
	case spannerpb.TypeCode_JSON:
		x, ok := v.Kind.(*structpb.Value_StringValue)
		if !ok || !json.Valid([]byte(x.StringValue)) {
			return nil, fmt.Errorf("JSON requires an encoded JSON string")
		}
		return x.StringValue, nil
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
	case spannerpb.TypeCode_STRING, spannerpb.TypeCode_BYTES, spannerpb.TypeCode_PROTO, spannerpb.TypeCode_TIMESTAMP, spannerpb.TypeCode_DATE:
		x, ok := v.Kind.(*structpb.Value_StringValue)
		if !ok {
			return nil, fmt.Errorf("expected JSON string for %s", t.Code)
		}
		s := x.StringValue
		switch t.Code {
		case spannerpb.TypeCode_BYTES, spannerpb.TypeCode_PROTO:
			// PROTO has the same base64 wire envelope as BYTES. Count analysis
			// treats the payload as one opaque cell; descriptor-level validation
			// and successful decoding remain service/caller preconditions.
			b, e := base64.StdEncoding.DecodeString(s)
			if e != nil {
				return nil, fmt.Errorf("invalid base64")
			}
			if t.Length != nil && int64(len(b)) > *t.Length {
				return nil, fmt.Errorf("BYTES length exceeded")
			}
			s = base64.StdEncoding.EncodeToString(b)
		case spannerpb.TypeCode_STRING:
			if t.Length != nil && int64(utf8.RuneCountInString(s)) > *t.Length {
				return nil, fmt.Errorf("STRING length exceeded")
			}
		case spannerpb.TypeCode_TIMESTAMP:
			if s == "spanner.commit_timestamp()" {
				if !commitTimestamp {
					return nil, fmt.Errorf("commit timestamp not allowed")
				}
			} else {
				parsed, e := time.Parse(time.RFC3339Nano, s)
				if e != nil {
					return nil, fmt.Errorf("invalid timestamp")
				}
				s = parsed.UTC().Format(time.RFC3339Nano)
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
			n, e := mutationProtoValue(t.ArrayElement, a, false, false)
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

func mutationDMLOperations(s *mutationCountSchemaModel, sql string) ([]mutationCountOperation, error) {
	n, err := memefish.ParseDML("write.sql", sql)
	if err != nil {
		return nil, err
	}
	base := mutationCountOperation{Channel: "dml", Rows: mutationUncertain(0, nil), Values: map[string]interface{}{}, Assumptions: []string{"DML expression semantics and state constraints are validated by the service, not this parser"}}
	switch n := n.(type) {
	case *ast.Insert:
		if n.OnConflict != nil {
			return nil, fmt.Errorf("DML conflict clauses require a separate write model")
		}
		base.Kind = "insert"
		if mode := string(n.InsertOrType); mode != "" {
			switch {
			case strings.Contains(mode, "UPDATE"):
				base.Kind = "upsert"
			case strings.Contains(mode, "REPLACE"):
				base.Kind = "replace"
			case strings.Contains(mode, "IGNORE"):
				base.WireOperation = "insert_or_ignore"
			default:
				return nil, fmt.Errorf("unsupported INSERT conflict mode %s", mode)
			}
		}
		base.Table = mutationPathName(n.TableName)
		v, ok := n.Input.(*ast.ValuesInput)
		if !ok {
			if _, ok := n.Input.(*ast.SubQueryInput); !ok {
				return nil, fmt.Errorf("unsupported INSERT input")
			}
			base.InsertSelect = true
			base.Columns = mutationIdentNames(n.Columns)
			base.Assumptions = append(base.Assumptions, "INSERT SELECT source semantics/cardinality are not analyzed or executed")
			if err := checkMutationDMLColumns(s, []mutationCountOperation{base}); err != nil {
				return nil, err
			}
			return []mutationCountOperation{base}, nil
		}
		var out []mutationCountOperation
		for _, row := range v.Rows {
			if len(row.Exprs) != len(n.Columns) {
				return nil, fmt.Errorf("INSERT values/columns arity mismatch")
			}
			op := base
			op.Rows = mutationFixed(1)
			if op.WireOperation == "insert_or_ignore" {
				one := int64(1)
				op.Rows = mutationUncertain(0, &one)
			}
			op.Values = map[string]interface{}{}
			for i, e := range row.Exprs {
				if e.Default {
					continue
				}
				c := n.Columns[i].Name
				op.Columns = append(op.Columns, c)
				if value, ok := mutationLiteralValue(e.Expr); ok {
					op.Values[strings.ToLower(c)] = value
				}
			}
			out = append(out, op)
		}
		if err := checkMutationDMLColumns(s, out); err != nil {
			return nil, err
		}
		return out, nil
	case *ast.Update:
		base.Kind = "update"
		base.Table = mutationPathName(n.TableName)
		for _, item := range n.Updates {
			u, ok := item.(*ast.UpdateItemSetValue)
			if !ok || len(u.Path) != 1 {
				return nil, fmt.Errorf("nested or array UPDATE requires a separate model")
			}
			c := u.Path[0].Name
			base.Columns = append(base.Columns, c)
			if u.DefaultExpr.Default {
				if t := s.Tables[strings.ToLower(base.Table)]; t != nil {
					if v, ok := mutationConstantValue(t.Defaults[strings.ToLower(c)]); ok {
						base.Values[strings.ToLower(c)] = v
					}
				}
			} else if v, ok := mutationLiteralValue(u.DefaultExpr.Expr); ok {
				base.Values[strings.ToLower(c)] = v
			}
		}
	case *ast.Delete:
		base.Kind = "delete"
		base.Table = mutationPathName(n.TableName)
		base.DeleteUnits = mutationUncertain(0, nil)
	default:
		return nil, fmt.Errorf("DML kind is outside this prototype")
	}
	if err := checkMutationDMLColumns(s, []mutationCountOperation{base}); err != nil {
		return nil, err
	}
	return []mutationCountOperation{base}, nil
}
func checkMutationDMLColumns(s *mutationCountSchemaModel, ops []mutationCountOperation) error {
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
			if op.Kind == "update" && mutationContainsFold(mutationPrimaryKeys(t.Table), name) {
				return fmt.Errorf("DML cannot update a primary key")
			}
		}
	}
	return nil
}

func mutationConstantValue(sql string) (interface{}, bool) {
	n, e := memefish.ParseExpr("", sql)
	if e != nil {
		return nil, false
	}
	return mutationLiteralValue(n)
}
func mutationLiteralValue(n ast.Expr) (interface{}, bool) {
	switch n := n.(type) {
	case *ast.ParenExpr:
		return mutationLiteralValue(n.Expr)
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
			v, ok := mutationLiteralValue(n.Expr)
			if s, yes := v.(string); ok && yes {
				i, e := strconv.ParseInt("-"+s, 10, 64)
				return strconv.FormatInt(i, 10), e == nil
			}
		}
	}
	return nil, false
}
func mutationGeneratedDependency(sql string, written map[string]bool, pks []string) bool {
	n, err := memefish.ParseExpr("", sql)
	if err != nil {
		return true
	}
	depends := false
	ast.Inspect(n, func(n ast.Node) bool {
		if i, ok := n.(*ast.Ident); ok {
			depends = depends || (written[strings.ToLower(i.Name)] && !mutationContainsFold(pks, i.Name))
		}
		return true
	})
	return depends
}
