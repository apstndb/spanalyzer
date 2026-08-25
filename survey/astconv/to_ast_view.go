package astconv

import (
	"fmt"

	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func (s *Schema) toViewsDDL() ([]ast.DDL, error) {
	var ddls []ast.DDL
	for _, v := range s.Views {
		// Skip system schema views, but never silently drop user DDL.
		if isSystemSchemaName(v.TableSchema) {
			continue
		}
		qs, err := memefish.ParseQuery("", v.ViewDefinition)
		if err != nil {
			return nil, fmt.Errorf("view %s: %w", tableDisplayName(v.TableSchema, v.TableName), err)
		}

		cv := &ast.CreateView{
			Name:  schemaObjectPath(v.TableSchema, v.TableName),
			Query: qs.Query,
		}

		switch v.SecurityType {
		case "INVOKER":
			cv.SecurityType = ast.SecurityTypeInvoker
		case "DEFINER":
			cv.SecurityType = ast.SecurityTypeDefiner
		default:
			// Default to INVOKER if security type is unknown or empty
			cv.SecurityType = ast.SecurityTypeInvoker
		}

		ddls = append(ddls, cv)
	}
	return ddls, nil
}
