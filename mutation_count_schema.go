package spanalyzer

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
	"github.com/cloudspannerecosystem/memefish/token"
)

// The ordinary row-type catalog intentionally omits some write-accounting
// facts. This supplemental model reuses its tables/types and retains those facts.
type mutationCountSchemaModel struct {
	Tables       map[string]*mutationCountTableModel
	Indexes      map[string][]MutationCountIndex
	ForeignKeys  []mutationCountForeignKey
	SourceSHA256 string
	IndexSource  string
}
type mutationCountTableModel struct {
	Queue       bool
	Table       *Table
	Defaults    map[string]string
	Generated   map[string]string // Stored expressions contribute base-row cells.
	Expressions map[string]string // All generated expressions can affect indexes.
	OnUpdate    map[string]bool
	Virtual     map[string]bool
	Unknown     []string
	Parent      string
	Cascade     bool
}

// MutationCountIndex preserves declared or observed index facts and their origin.
// Constraints are inferred unique-prefix aliases, not observed associations.
type MutationCountIndex struct {
	Name             string   `json:"name" yaml:"name"`
	Table            string   `json:"table" yaml:"table"`
	Keys             []string `json:"keys" yaml:"keys"`
	Stored           []string `json:"stored,omitempty" yaml:"stored,omitempty"`
	NullFiltered     bool     `json:"null_filtered" yaml:"null_filtered"`
	Managed          bool     `json:"managed" yaml:"managed"`
	ObservedUnique   *bool    `json:"observed_unique,omitempty" yaml:"observed_unique,omitempty"`
	PredicateUnknown bool     `json:"predicate_unknown,omitempty" yaml:"predicate_unknown,omitempty"`
	Constraints      []string `json:"inferred_constraint_selectors,omitempty" yaml:"inferred_constraint_selectors,omitempty"`
	Origin           string   `json:"origin" yaml:"origin"`
}
type mutationCountForeignKey struct {
	Name, Table, Referenced   string
	Columns, ReferenceColumns []string
	Enforced                  bool
	DeleteAction              string
}

// MutationCountIndexObservation uses canonical snake_case JSON fields while
// retaining compatibility with the earlier probe DTO's NullFiltered spelling.
type MutationCountIndexObservation struct {
	Table        string   `json:"table" yaml:"table"`
	Name         string   `json:"name" yaml:"name"`
	Type         string   `json:"type" yaml:"type"`
	State        string   `json:"state" yaml:"state"`
	Unique       bool     `json:"unique" yaml:"unique"`
	NullFiltered bool     `json:"null_filtered" yaml:"null_filtered"`
	Managed      bool     `json:"managed" yaml:"managed"`
	Columns      []string `json:"columns" yaml:"columns"`
	Stored       []string `json:"stored" yaml:"stored"`
}

func newMutationCountSchema(ddl string) (*mutationCountSchemaModel, error) {
	adapted, queues, err := prepareMutationCountDDL(ddl)
	if err != nil {
		return nil, err
	}
	c, err := BuildSchemaCatalog("schema.sql", adapted)
	if err != nil {
		return nil, err
	}
	ddls, err := memefish.ParseDDLs("schema.sql", adapted)
	if err != nil {
		return nil, err
	}
	s := &mutationCountSchemaModel{Tables: map[string]*mutationCountTableModel{}, Indexes: map[string][]MutationCountIndex{}, SourceSHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(ddl))), IndexSource: "ddl_declared_only"}
	for _, n := range ddls {
		switch n := n.(type) {
		case *ast.CreateTable:
			name := mutationPathName(n.Name)
			t := c.Tables[name]
			if t == nil {
				continue
			}
			key := strings.ToLower(name)
			if s.Tables[key] != nil {
				return nil, fmt.Errorf("ambiguous table spelling %q", name)
			}
			m := &mutationCountTableModel{Table: t, Defaults: map[string]string{}, Generated: map[string]string{}, Expressions: map[string]string{}, OnUpdate: map[string]bool{}, Virtual: map[string]bool{}}
			for _, col := range t.Columns {
				if col.Hidden && col.PrimaryKey && strings.Contains(col.GeneratedSQL, "IDENTITY") {
					m.Unknown = append(m.Unknown, "implicit_identity_primary_key")
				}
			}
			if queues[n.Create] {
				if err := markMutationQueue(m); err != nil {
					return nil, fmt.Errorf("queue %s: %w", name, err)
				}
			}
			for _, col := range n.Columns {
				k := strings.ToLower(col.Name.Name)
				switch v := col.DefaultSemantics.(type) {
				case *ast.ColumnDefaultExpr:
					m.Defaults[k] = v.Expr.SQL()
					if v.OnUpdate != nil {
						m.OnUpdate[k] = true
					}
				case *ast.GeneratedColumnExpr:
					m.Expressions[k] = v.Expr.SQL()
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
				m.Parent = mutationPathName(n.Cluster.TableName)
				m.Cascade = n.Cluster.Enforced && strings.Contains(string(n.Cluster.OnDelete), "CASCADE")
			}
			for _, tc := range n.TableConstraints {
				if f, ok := tc.Constraint.(*ast.ForeignKey); ok {
					fk := mutationCountForeignKey{Table: name, Referenced: mutationPathName(f.ReferenceTable), Columns: mutationIdentNames(f.Columns), ReferenceColumns: mutationIdentNames(f.ReferenceColumns), Enforced: !strings.Contains(string(f.Enforcement), "NOT"), DeleteAction: string(f.OnDelete)}
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
		i := MutationCountIndex{Name: idx.Name.String(), Table: idx.TableName.String(), Stored: append([]string(nil), idx.StoredColumns...), NullFiltered: idx.NullFiltered, Origin: "ddl"}
		for _, k := range idx.Keys {
			i.Keys = append(i.Keys, k.Name)
		}
		s.Indexes[strings.ToLower(i.Table)] = append(s.Indexes[strings.ToLower(i.Table)], i)
	}
	for _, n := range ddls {
		switch n := n.(type) {
		case *ast.CreateIndex:
			if n.Where != nil {
				key := strings.ToLower(mutationPathName(n.TableName))
				for i := range s.Indexes[key] {
					if s.Indexes[key][i].Name == mutationPathName(n.Name) {
						s.Indexes[key][i].PredicateUnknown = true
					}
				}
			}
			if n.InterleaveIn != nil {
				if t := s.Tables[strings.ToLower(mutationPathName(n.TableName))]; t != nil {
					t.Unknown = append(t.Unknown, "interleaved_index_delete_policy")
				}
			}
		case *ast.CreateSearchIndex:
			if t := s.Tables[strings.ToLower(mutationPathName(n.TableName))]; t != nil {
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

func (s *mutationCountSchemaModel) withObservedIndexes(indexes []MutationCountIndexObservation) (*mutationCountSchemaModel, error) {
	clone := *s
	clone.IndexSource = "observed_snapshot_requires_freshness_assumption"
	clone.Indexes = map[string][]MutationCountIndex{}
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
		i := MutationCountIndex{Name: m.Name, Table: m.Table, Keys: slices.Clone(m.Columns), Stored: slices.Clone(m.Stored), NullFiltered: m.NullFiltered, Managed: m.Managed, ObservedUnique: &m.Unique, Origin: "observed_snapshot"}
		clone.Indexes[strings.ToLower(m.Table)] = append(clone.Indexes[strings.ToLower(m.Table)], i)
	}
	for key, declared := range s.Indexes {
		for _, d := range declared {
			found := false
			for n, i := range clone.Indexes[key] {
				if strings.EqualFold(d.Name, i.Name) {
					if !mutationPrefixEqual(i.Keys, d.Keys) || !mutationNameSetEqual(i.Stored, d.Stored) || i.NullFiltered != d.NullFiltered {
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
		if fk.Name == "<unnamed>" {
			continue
		}
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
				if i.Managed && mutationPrefixEqual(i.Keys, role.keys) && (role.name != "referenced" || i.ObservedUnique != nil && *i.ObservedUnique) {
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

func (s *mutationCountSchemaModel) potentialManaged(table string) int64 {
	if s.IndexSource != "ddl_declared_only" {
		return 0
	}
	var n int64
	for _, fk := range s.ForeignKeys {
		if fk.Enforced && strings.EqualFold(fk.Table, table) {
			t := s.Tables[strings.ToLower(table)]
			covered := t != nil && mutationPrefixEqual(mutationPrimaryKeys(t.Table), fk.Columns)
			if covered {
				for _, name := range fk.Columns {
					col, _ := t.Table.Column(name)
					if col == nil || !col.NotNull {
						covered = false
						break
					}
				}
			}
			// A nullable PK prefix still needs a NULL_FILTERED backing index.
			// Composite referenced keys can also reorder the referencing keys;
			// only rule that out for an unchanged referenced PK prefix. Other
			// layouts retain the upper bound until an index snapshot is supplied.
			// See knowledge/research/runtime-verification/
			// fk-backing-index-predictability-20261112.md for the runtime evidence.
			if covered && len(fk.Columns) > 1 {
				referenced := s.Tables[strings.ToLower(fk.Referenced)]
				covered = referenced != nil && mutationPrefixEqual(mutationPrimaryKeys(referenced.Table), fk.ReferenceColumns)
			}
			if !covered {
				n++
			}
		}
		if strings.EqualFold(fk.Referenced, table) {
			t := s.Tables[strings.ToLower(table)]
			if t == nil || !mutationEqualNames(mutationPrimaryKeys(t.Table), fk.ReferenceColumns) {
				n++
			}
		}
	}
	// This is an upper bound on additional indexes, not one index per FK:
	// managed backing indexes may be shared, and key ordering can be rewritten.
	return n
}
func mutationPrimaryKeys(t *Table) []string {
	var out []string
	for _, p := range t.PrimaryKey {
		out = append(out, p.Name)
	}
	return out
}
func mutationPathName(p *ast.Path) string {
	if p == nil {
		return ""
	}
	var parts []string
	for _, i := range p.Idents {
		parts = append(parts, i.Name)
	}
	return strings.Join(parts, ".")
}
func mutationIdentNames(in []*ast.Ident) []string {
	var out []string
	for _, i := range in {
		out = append(out, i.Name)
	}
	return out
}
func mutationPrefixEqual(a, b []string) bool {
	if len(a) < len(b) {
		return false
	}
	return mutationEqualNames(a[:len(b)], b)
}
func mutationEqualNames(a, b []string) bool {
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

func mutationNameSetEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	left := make([]string, len(a))
	right := make([]string, len(b))
	for i := range a {
		left[i] = strings.ToLower(a[i])
		right[i] = strings.ToLower(b[i])
	}
	slices.Sort(left)
	slices.Sort(right)
	return slices.Equal(left, right)
}

func (s *mutationCountSchemaModel) cascadeDescendants(parent string) []*mutationCountTableModel {
	var out []*mutationCountTableModel
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
