package main

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"sort"
	"strings"

	spanalyzer "github.com/apstndb/spanalyzer"
	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
	"github.com/cloudspannerecosystem/memefish/token"
)

// This is an experimental comparison model, not the analyzer's public catalog.
// The row-type catalog intentionally omits facts needed by write accounting.
type schemaModel struct {
	Tables       map[string]*tableModel
	Indexes      map[string][]indexModel
	ForeignKeys  []foreignKeyModel
	SourceSHA256 string
	IndexSource  string
}
type tableModel struct {
	Table     *spanalyzer.Table
	Defaults  map[string]string
	Generated map[string]string
	Virtual   map[string]bool
	Unknown   []string
	Parent    string
	Cascade   bool
}
type indexModel struct {
	Name             string   `json:"name"`
	Table            string   `json:"table"`
	Keys             []string `json:"keys"`
	Stored           []string `json:"stored,omitempty"`
	NullFiltered     bool     `json:"null_filtered"`
	Managed          bool     `json:"managed"`
	ObservedUnique   *bool    `json:"observed_unique,omitempty"`
	PredicateUnknown bool     `json:"predicate_unknown,omitempty"`
	Constraints      []string `json:"inferred_constraint_selectors,omitempty"`
	Origin           string   `json:"origin"`
}
type foreignKeyModel struct {
	Name, Table, Referenced   string
	Columns, ReferenceColumns []string
	Enforced                  bool
	DeleteAction              string
}

// observedIndex is deliberately compatible with the retained local probe DTO.
type observedIndex struct {
	Table, Name, Type, State      string
	Unique, NullFiltered, Managed bool
	Columns, Stored               []string
}

func schemaFromDDL(ddl string) (*schemaModel, error) {
	c, err := spanalyzer.BuildSchemaCatalog("schema.sql", ddl)
	if err != nil {
		return nil, err
	}
	ddls, err := memefish.ParseDDLs("schema.sql", ddl)
	if err != nil {
		return nil, err
	}
	s := &schemaModel{Tables: map[string]*tableModel{}, Indexes: map[string][]indexModel{}, SourceSHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(ddl))), IndexSource: "ddl_declared_only"}
	for _, n := range ddls {
		switch n := n.(type) {
		case *ast.CreateTable:
			name := pathName(n.Name)
			t := c.Tables[name]
			if t == nil {
				continue
			}
			key := strings.ToLower(name)
			if s.Tables[key] != nil {
				return nil, fmt.Errorf("ambiguous table spelling %q", name)
			}
			m := &tableModel{Table: t, Defaults: map[string]string{}, Generated: map[string]string{}, Virtual: map[string]bool{}}
			for _, col := range n.Columns {
				k := strings.ToLower(col.Name.Name)
				switch v := col.DefaultSemantics.(type) {
				case *ast.ColumnDefaultExpr:
					m.Defaults[k] = v.Expr.SQL()
					if v.OnUpdate != nil {
						m.Unknown = append(m.Unknown, "on_update_expression")
					}
				case *ast.GeneratedColumnExpr:
					if v.Stored != token.InvalidPos {
						m.Generated[k] = v.Expr.SQL()
					} else {
						m.Virtual[k] = true
					}
				case *ast.IdentityColumn:
					m.Unknown = append(m.Unknown, "identity_generation")
				}
			}
			if n.Cluster != nil {
				m.Parent = pathName(n.Cluster.TableName)
				m.Cascade = n.Cluster.Enforced && strings.Contains(string(n.Cluster.OnDelete), "CASCADE")
			}
			for _, tc := range n.TableConstraints {
				if f, ok := tc.Constraint.(*ast.ForeignKey); ok {
					fk := foreignKeyModel{Table: name, Referenced: pathName(f.ReferenceTable), Columns: identNames(f.Columns), ReferenceColumns: identNames(f.ReferenceColumns), Enforced: !strings.Contains(string(f.Enforcement), "NOT"), DeleteAction: string(f.OnDelete)}
					if tc.Name != nil {
						fk.Name = tc.Name.Name
					} else {
						fk.Name = "<unnamed>"
					}
					s.ForeignKeys = append(s.ForeignKeys, fk)
				}
			}
			s.Tables[key] = m
		case *ast.CreateSearchIndex:
			// Applied below after all table declarations are available.
		case *ast.CreateVectorIndex:
		case *ast.AlterTable:
			// A general ALTER history needs a full supplemental schema state
			// machine. Do not silently trust only CREATE-time foreign keys.
			for _, t := range s.Tables {
				t.Unknown = append(t.Unknown, "alter_history_not_modeled")
			}
		}
	}
	for _, idx := range c.Indexes {
		i := indexModel{Name: idx.Name.String(), Table: idx.TableName.String(), Stored: append([]string(nil), idx.StoredColumns...), NullFiltered: idx.NullFiltered, Origin: "ddl"}
		for _, k := range idx.Keys {
			i.Keys = append(i.Keys, k.Name)
		}
		s.Indexes[strings.ToLower(i.Table)] = append(s.Indexes[strings.ToLower(i.Table)], i)
	}
	for _, n := range ddls {
		switch n := n.(type) {
		case *ast.CreateIndex:
			if n.Where != nil {
				key := strings.ToLower(pathName(n.TableName))
				for i := range s.Indexes[key] {
					if s.Indexes[key][i].Name == pathName(n.Name) {
						s.Indexes[key][i].PredicateUnknown = true
					}
				}
			}
			if n.InterleaveIn != nil {
				if t := s.Tables[strings.ToLower(pathName(n.TableName))]; t != nil {
					t.Unknown = append(t.Unknown, "interleaved_index_delete_policy")
				}
			}
		case *ast.CreateSearchIndex:
			if t := s.Tables[strings.ToLower(pathName(n.TableName))]; t != nil {
				t.Unknown = append(t.Unknown, "search_index_accounting")
			}
		case *ast.CreateVectorIndex:
			if t := s.Tables[strings.ToLower(n.TableName.Name)]; t != nil {
				t.Unknown = append(t.Unknown, "vector_index_accounting")
			}
		}
	}
	for k := range s.Indexes {
		sort.Slice(s.Indexes[k], func(i, j int) bool { return s.Indexes[k][i].Name < s.Indexes[k][j].Name })
	}
	return s, nil
}

func (s *schemaModel) withObservedIndexes(indexes []observedIndex) (*schemaModel, error) {
	clone := *s
	clone.IndexSource = "observed_snapshot_requires_freshness_assumption"
	clone.Indexes = map[string][]indexModel{}
	seen := map[string]bool{}
	for _, m := range indexes {
		if m.Type == "PRIMARY_KEY" {
			continue
		}
		t := s.Tables[strings.ToLower(m.Table)]
		if t == nil {
			return nil, fmt.Errorf("snapshot index %s refers to table absent from DDL: %s", m.Name, m.Table)
		}
		if m.Type != "INDEX" || m.State != "READ_WRITE" {
			return nil, fmt.Errorf("unmodeled snapshot index %s: type=%s state=%s", m.Name, m.Type, m.State)
		}
		identity := strings.ToLower(m.Table + "." + m.Name)
		if seen[identity] {
			return nil, fmt.Errorf("duplicate snapshot index %s", m.Name)
		}
		seen[identity] = true
		for _, name := range append(slices.Clone(m.Columns), m.Stored...) {
			if col, _ := t.Table.Column(name); col == nil {
				return nil, fmt.Errorf("snapshot %s column %q absent from schema", m.Name, name)
			}
		}
		i := indexModel{Name: m.Name, Table: m.Table, Keys: slices.Clone(m.Columns), Stored: slices.Clone(m.Stored), NullFiltered: m.NullFiltered, Managed: m.Managed, ObservedUnique: &m.Unique, Origin: "observed_snapshot"}
		clone.Indexes[strings.ToLower(m.Table)] = append(clone.Indexes[strings.ToLower(m.Table)], i)
	}
	for key, declared := range s.Indexes {
		for _, d := range declared {
			found := false
			for n, i := range clone.Indexes[key] {
				if strings.EqualFold(d.Name, i.Name) {
					if !prefixEqual(i.Keys, d.Keys) || !equalNames(i.Stored, d.Stored) || i.NullFiltered != d.NullFiltered {
						return nil, fmt.Errorf("snapshot disagrees with declared index %s", d.Name)
					}
					found = true
					clone.Indexes[key][n].PredicateUnknown = d.PredicateUnknown
				}
			}
			if !found {
				return nil, fmt.Errorf("snapshot omits declared index %s", d.Name)
			}
		}
	}
	// Constraint selectors are semantic aliases, never guesses of the
	// generated hexadecimal suffix. Both FK sides and sharing are retained.
	for _, fk := range s.ForeignKeys {
		for _, role := range []struct {
			table   string
			keys    []string
			name    string
			enabled bool
		}{{fk.Table, fk.Columns, "referencing", fk.Enforced}, {fk.Referenced, fk.ReferenceColumns, "referenced", true}} {
			if !role.enabled {
				continue
			}
			key := strings.ToLower(role.table)
			var candidates []int
			for n, i := range clone.Indexes[key] {
				if i.Managed && prefixEqual(i.Keys, role.keys) && (role.name != "referenced" || i.ObservedUnique != nil && *i.ObservedUnique) {
					candidates = append(candidates, n)
				}
			}
			if len(candidates) == 1 {
				n := candidates[0]
				clone.Indexes[key][n].Constraints = append(clone.Indexes[key][n].Constraints, fk.Name+":"+role.name)
			}
		}
	}
	return &clone, nil
}

func (s *schemaModel) potentialManaged(table string) int64 {
	if s.IndexSource != "ddl_declared_only" {
		return 0
	}
	var n int64
	for _, fk := range s.ForeignKeys {
		if fk.Enforced && strings.EqualFold(fk.Table, table) {
			t := s.Tables[strings.ToLower(table)]
			if t == nil || !prefixEqual(primaryKeys(t.Table), fk.Columns) {
				n++
			}
		}
		if strings.EqualFold(fk.Referenced, table) {
			t := s.Tables[strings.ToLower(table)]
			if t == nil || !equalNames(primaryKeys(t.Table), fk.ReferenceColumns) {
				n++
			}
		}
	}
	// This is an upper bound on additional indexes, not one index per FK:
	// managed backing indexes may be shared, and key ordering can be rewritten.
	return n
}
func primaryKeys(t *spanalyzer.Table) []string {
	var out []string
	for _, p := range t.PrimaryKey {
		out = append(out, p.Name)
	}
	return out
}
func pathName(p *ast.Path) string {
	if p == nil {
		return ""
	}
	var parts []string
	for _, i := range p.Idents {
		parts = append(parts, i.Name)
	}
	return strings.Join(parts, ".")
}
func identNames(in []*ast.Ident) []string {
	var out []string
	for _, i := range in {
		out = append(out, i.Name)
	}
	return out
}
func prefixEqual(a, b []string) bool {
	if len(a) < len(b) {
		return false
	}
	return equalNames(a[:len(b)], b)
}
func equalNames(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !strings.EqualFold(a[i], b[i]) {
			return false
		}
	}
	return true
}

func (s *schemaModel) cascadeDescendants(parent string) []*tableModel {
	var out []*tableModel
	seen := map[string]bool{strings.ToLower(parent): true}
	queue := []string{parent}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		for key, t := range s.Tables {
			if !seen[key] && t.Cascade && strings.EqualFold(t.Parent, p) {
				seen[key] = true
				out = append(out, t)
				queue = append(queue, t.Table.Name.String())
			}
		}
	}
	return out
}
