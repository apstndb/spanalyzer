package astconv

import (
	"fmt"

	"github.com/cloudspannerecosystem/memefish/ast"
)

// ident creates an ast.Ident from a string.
func ident(name string) *ast.Ident {
	return &ast.Ident{Name: name}
}

// path creates an ast.Path from one or more name segments.
func path(names ...string) *ast.Path {
	if len(names) == 0 {
		return nil
	}
	idents := make([]*ast.Ident, len(names))
	for i, n := range names {
		idents[i] = ident(n)
	}
	return &ast.Path{Idents: idents}
}

// schemaObjectName splits an unqualified or schema-qualified object path.
func schemaObjectName(kind string, name *ast.Path) (schemaName, objectName string, err error) {
	if name == nil || len(name.Idents) == 0 {
		return "", "", fmt.Errorf("%s has no name", kind)
	}
	switch len(name.Idents) {
	case 1:
		return "", name.Idents[0].Name, nil
	case 2:
		return name.Idents[0].Name, name.Idents[1].Name, nil
	default:
		return "", "", fmt.Errorf("unsupported qualified %s %q", kind, name.SQL())
	}
}

// schemaObjectPath creates an unqualified or schema-qualified object path.
func schemaObjectPath(schemaName, objectName string) *ast.Path {
	if schemaName == "" {
		return path(objectName)
	}
	return path(schemaName, objectName)
}

// leafName returns the last identifier of an ast.Path, or "" for nil/empty paths.
func leafName(p *ast.Path) string {
	if p == nil || len(p.Idents) == 0 {
		return ""
	}
	return p.Idents[len(p.Idents)-1].Name
}

// strval creates an ast.StringLiteral.
func strval(s string) *ast.StringLiteral {
	return &ast.StringLiteral{Value: s}
}

// intval creates an ast.IntLiteral.
func intval(s string) *ast.IntLiteral {
	return &ast.IntLiteral{Value: s}
}

// boolval creates a BoolLiteral.
func boolval(b bool) *ast.BoolLiteral {
	return &ast.BoolLiteral{Value: b}
}

// nullval creates a NullLiteral.
func nullval() *ast.NullLiteral {
	return &ast.NullLiteral{}
}

// optionsDef creates an ast.OptionsDef with the given name and value.
func optionsDef(name string, value ast.Expr) *ast.OptionsDef {
	return &ast.OptionsDef{
		Name:  ident(name),
		Value: value,
	}
}

// mkOptions creates an ast.Options from a list of OptionsDefs.
// Returns nil if defs is empty.
func mkOptions(defs ...*ast.OptionsDef) *ast.Options {
	if len(defs) == 0 {
		return nil
	}
	return &ast.Options{
		Records: defs,
	}
}

// indexKey creates an ast.IndexKey.
func indexKey(name string, dir ast.Direction) *ast.IndexKey {
	return &ast.IndexKey{
		Name: ident(name),
		Dir:  dir,
	}
}

// dirFromString converts a COLUMN_ORDERING string to ast.Direction.
func dirFromString(s string) ast.Direction {
	switch s {
	case "ASC":
		return ast.DirectionAsc
	case "DESC":
		return ast.DirectionDesc
	default:
		return ""
	}
}

// ptrStr dereferences a *string, returning "" if nil.
func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// strPtr returns a pointer to a string.
func strPtr(s string) *string {
	return &s
}

// int64Ptr returns a pointer to an int64.
func int64Ptr(i int64) *int64 {
	return &i
}
