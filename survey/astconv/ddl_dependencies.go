package astconv

import (
	"container/heap"
	"fmt"
	"maps"
	"strings"

	"github.com/cloudspannerecosystem/memefish/ast"
)

// DDL identities are separate namespaces. Joining identifier components with a
// NUL preserves the distinction between a qualified name and a quoted dotted
// identifier. Folding is for SQL reference lookup only; emitted names and proto
// type names retain their original spelling.
func ddlKey(kind string, names ...string) string {
	return kind + ":" + strings.ToLower(strings.Join(names, "\x00"))
}

func ddlPathKey(kind string, p *ast.Path) string {
	names := make([]string, len(p.Idents))
	for i, ident := range p.Idents {
		names[i] = ident.Name
	}
	return ddlKey(kind, names...)
}

func ddlIdentity(ddl ast.DDL) (key string, name *ast.Path) {
	switch n := ddl.(type) {
	case *ast.CreateProtoBundle:
		return "proto bundle", nil
	case *ast.CreateSchema:
		return ddlKey("schema", n.Name.Name), nil
	case *ast.CreateTable:
		return ddlPathKey("relation", n.Name), n.Name
	case *ast.CreateView:
		return ddlPathKey("relation", n.Name), n.Name
	case *ast.CreateSequence:
		return ddlPathKey("sequence", n.Name), n.Name
	case *ast.CreateFunction:
		return ddlPathKey("function", n.Name), n.Name
	case *ast.CreateModel:
		return ddlKey("model", n.Name.Name), nil
	case *ast.CreateIndex:
		return ddlPathKey("index", n.Name), n.Name
	case *ast.CreateSearchIndex:
		return ddlPathKey("index", n.Name), n.Name
	case *ast.CreateVectorIndex:
		return ddlKey("index", n.Name.Name), nil
	case *ast.CreateLocalityGroup:
		return ddlKey("locality group", n.Name.Name), nil
	case *ast.CreatePlacement:
		return ddlKey("placement", n.Name.Name), nil
	}
	return "", nil
}

func orderDDLDependencies(ddls []ast.DDL) ([]ast.DDL, error) {
	objects := make(map[string]int)
	names := make([]*ast.Path, len(ddls))
	labels := make([]string, len(ddls))
	for i, ddl := range ddls {
		key, name := ddlIdentity(ddl)
		names[i] = name
		labels[i] = strings.ReplaceAll(key, "\x00", ".")
		if key == "" {
			labels[i] = fmt.Sprintf("statement %d (%T)", i+1, ddl)
			continue
		}
		if _, exists := objects[key]; exists {
			return nil, fmt.Errorf("duplicate DDL object %s", labels[i])
		}
		objects[key] = i
	}
	// Synonyms resolve to the creating table, including when referenced by views.
	for i, ddl := range ddls {
		if table, ok := ddl.(*ast.CreateTable); ok {
			for _, synonym := range table.Synonyms {
				key := ddlKey("relation", synonym.Name.Name)
				if _, exists := objects[key]; exists {
					return nil, fmt.Errorf("duplicate DDL object %s", key)
				}
				objects[key] = i
			}
		}
	}
	remaining := make([]int, len(ddls))
	dependents := make([][]int, len(ddls))
	for i, ddl := range ddls {
		deps := make(map[int]bool)
		add := func(key string) {
			if dependency, ok := objects[key]; ok {
				deps[dependency] = true
			}
		}
		if name := names[i]; name != nil && len(name.Idents) > 1 {
			add(ddlKey("schema", name.Idents[0].Name))
		}
		if _, isProto := ddl.(*ast.CreateProtoBundle); !isProto {
			add("proto bundle")
		}
		ast.Walk(ddl, ddlDependencyVisitor{add: add, ctes: map[string]bool{}, aliases: map[string]bool{}})
		// A self-referencing FK is legal; a recursive view/function is not.
		if _, table := ddl.(*ast.CreateTable); table {
			delete(deps, i)
		}
		remaining[i] = len(deps)
		for dependency := range deps {
			dependents[dependency] = append(dependents[dependency], i)
		}
	}
	ready := &ddlReady{}
	for i, count := range remaining {
		if count == 0 {
			heap.Push(ready, i)
		}
	}
	ordered := make([]ast.DDL, 0, len(ddls))
	for ready.Len() > 0 {
		i := heap.Pop(ready).(int)
		ordered = append(ordered, ddls[i])
		for _, dependent := range dependents[i] {
			remaining[dependent]--
			if remaining[dependent] == 0 {
				heap.Push(ready, dependent)
			}
		}
	}
	if len(ordered) != len(ddls) {
		var blocked []string
		for i, count := range remaining {
			if count > 0 {
				blocked = append(blocked, labels[i])
			}
		}
		return nil, fmt.Errorf("DDL dependency cycle (including blocked dependents): %s", strings.Join(blocked, ", "))
	}
	return ordered, nil
}

type ddlReady []int

func (h ddlReady) Len() int           { return len(h) }
func (h ddlReady) Less(i, j int) bool { return h[i] < h[j] }
func (h ddlReady) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *ddlReady) Push(x any)        { *h = append(*h, x.(int)) }
func (h *ddlReady) Pop() any {
	i := len(*h) - 1
	x := (*h)[i]
	*h = (*h)[:i]
	return x
}

type ddlDependencyVisitor struct {
	add     func(string)
	ctes    map[string]bool
	aliases map[string]bool
}

func (v ddlDependencyVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.Select:
		if n.From != nil {
			// FROM sources introduce range variables from left to right.
			// Visit them before the select expressions, without leaking their
			// bindings to sibling queries. A range-variable path is implicit
			// UNNEST even if a named schema relation has the same spelling.
			v.aliases = maps.Clone(v.aliases)
			ast.Walk(n.From, v)
			body := *n
			body.From = nil
			ast.Walk(&body, v)
			return nil
		}
	case *ast.Query:
		if n.With != nil {
			// CTEs see preceding CTEs, not themselves or later definitions. Do
			// not suppress a real table dependency in a CTE with the same name.
			v.ctes = maps.Clone(v.ctes)
			for _, cte := range n.With.CTEs {
				ast.Walk(cte.QueryExpr, v)
				v.ctes[strings.ToLower(cte.Name.Name)] = true
			}
			body := *n
			body.With = nil
			ast.Walk(&body, v)
			return nil
		}
	case *ast.SubQueryTableExpr:
		// The alias is introduced after its source has been visited. It must
		// not mask a dependency inside the source query itself.
		ast.Walk(n.Query, v)
		if n.Sample != nil {
			ast.Walk(n.Sample, v)
		}
		if n.As != nil {
			v.aliases[strings.ToLower(n.As.Alias.Name)] = true
		}
		return nil
	case *ast.Unnest:
		ast.Walk(n.Expr, v)
		if n.Hint != nil {
			ast.Walk(n.Hint, v)
		}
		if n.As != nil {
			v.aliases[strings.ToLower(n.As.Alias.Name)] = true
		}
		if n.WithOffset != nil {
			ast.Walk(n.WithOffset, v)
		}
		if n.Sample != nil {
			ast.Walk(n.Sample, v)
		}
		return nil
	case *ast.WithOffset:
		alias := "offset"
		if n.As != nil {
			alias = n.As.Alias.Name
		}
		v.aliases[strings.ToLower(alias)] = true
	case *ast.TableName:
		if !v.ctes[strings.ToLower(n.Table.Name)] {
			v.add(ddlKey("relation", n.Table.Name))
		}
		alias := n.Table.Name
		if n.As != nil {
			alias = n.As.Alias.Name
		}
		v.aliases[strings.ToLower(alias)] = true
	case *ast.PathTableExpr:
		// A path starting with a CTE is an implicit UNNEST, not a named
		// schema reference. Only match known relations for other paths.
		first := strings.ToLower(n.Path.Idents[0].Name)
		if !v.ctes[first] && !v.aliases[first] {
			v.add(ddlPathKey("relation", n.Path))
		}
		alias := n.Path.Idents[len(n.Path.Idents)-1].Name
		if n.As != nil {
			alias = n.As.Alias.Name
		}
		v.aliases[strings.ToLower(alias)] = true
	case *ast.CallExpr:
		v.add(ddlPathKey("function", n.Func))
	case *ast.TVFCallExpr:
		v.add(ddlPathKey("function", n.Name))
	case *ast.SequenceArg:
		switch expr := n.Expr.(type) {
		case *ast.Path:
			v.add(ddlPathKey("sequence", expr))
		case *ast.Ident:
			v.add(ddlKey("sequence", expr.Name))
		}
	case *ast.CreateIndex:
		v.add(ddlPathKey("relation", n.TableName))
	case *ast.CreateSearchIndex:
		v.add(ddlPathKey("relation", n.TableName))
	case *ast.CreateVectorIndex:
		v.add(ddlKey("relation", n.TableName.Name))
	case *ast.ForeignKey:
		v.add(ddlPathKey("relation", n.ReferenceTable))
	case *ast.Cluster:
		v.add(ddlPathKey("relation", n.TableName))
	case *ast.InterleaveIn:
		v.add(ddlKey("relation", n.TableName.Name))
	case *ast.ChangeStreamForTable:
		v.add(ddlKey("relation", n.TableName.Name))
	case *ast.PropertyGraphElement:
		v.add(ddlKey("relation", n.Name.Name))
	case *ast.OptionsDef:
		if value, ok := n.Value.(*ast.StringLiteral); ok {
			switch strings.ToLower(n.Name.Name) {
			case "locality_group", "default_locality_group":
				v.add(ddlKey("locality group", value.Value))
			case "placement":
				v.add(ddlKey("placement", value.Value))
			}
		}
	}
	return v
}

func (v ddlDependencyVisitor) VisitMany([]ast.Node) ast.Visitor { return v }
func (v ddlDependencyVisitor) Field(string) ast.Visitor         { return v }
func (v ddlDependencyVisitor) Index(int) ast.Visitor            { return v }
