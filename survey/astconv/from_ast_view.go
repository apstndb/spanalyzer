package astconv

import (
	"fmt"

	"github.com/apstndb/spanner-emulator-survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func fromCreateView(s *Schema, cv *ast.CreateView) error {
	if cv.Name == nil || len(cv.Name.Idents) == 0 {
		return fmt.Errorf("view has no name")
	}
	viewSchema, viewName, err := schemaObjectName("view", cv.Name)
	if err != nil {
		return err
	}

	secType := string(cv.SecurityType)

	s.Views = append(s.Views, &infoschem.View{
		TableSchema:    viewSchema,
		TableName:      viewName,
		ViewDefinition: cv.Query.SQL(),
		SecurityType:   secType,
	})

	// INFORMATION_SCHEMA.COLUMNS exposes view output columns on a live
	// database. For AST-only round trips, recover the same minimal metadata from
	// a simple explicit SELECT list so property graphs can use a view as an
	// element table with implicit properties. If any output name is ambiguous
	// (for example SELECT * or an unaliased expression), leave the set empty so
	// graph conversion fails closed instead of inventing a column contract.
	if columnNames, ok := explicitViewColumnNames(cv.Query); ok {
		for i, columnName := range columnNames {
			s.Columns = append(s.Columns, &infoschem.Column{
				TableSchema:     viewSchema,
				TableName:       viewName,
				ColumnName:      columnName,
				OrdinalPosition: int64(i + 1),
			})
		}
	}

	// Also add to TABLES
	s.Tables = append(s.Tables, &infoschem.Table{
		TableSchema:  viewSchema,
		TableName:    viewName,
		TableType:    "VIEW",
		SpannerState: strPtr("COMMITTED"),
	})

	return nil
}

func explicitViewColumnNames(query ast.QueryExpr) ([]string, bool) {
	var selectQuery *ast.Select
	switch q := query.(type) {
	case *ast.Select:
		selectQuery = q
	case *ast.Query:
		if q.With != nil || len(q.PipeOperators) > 0 {
			return nil, false
		}
		return explicitViewColumnNames(q.Query)
	default:
		return nil, false
	}

	names := make([]string, 0, len(selectQuery.Results))
	for _, result := range selectQuery.Results {
		var name string
		switch item := result.(type) {
		case *ast.Alias:
			if item.As != nil && item.As.Alias != nil {
				name = item.As.Alias.Name
			}
		case *ast.ExprSelectItem:
			switch expr := item.Expr.(type) {
			case *ast.Ident:
				name = expr.Name
			case *ast.Path:
				name = leafName(expr)
			case *ast.SelectorExpr:
				if expr.Ident != nil {
					name = expr.Ident.Name
				}
			}
		}
		if name == "" {
			return nil, false
		}
		names = append(names, name)
	}
	return names, len(names) > 0
}
