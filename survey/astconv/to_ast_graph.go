package astconv

import (
	"fmt"

	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func (s *Schema) toGraphsDDL() ([]ast.DDL, error) {
	var ddls []ast.DDL
	for _, pg := range s.PropertyGraphs {
		if pg.PropertyGraphSchema != "" {
			return nil, fmt.Errorf("unsupported named-schema property graph %q", tableDisplayName(pg.PropertyGraphSchema, pg.PropertyGraphName))
		}
		meta, err := decodePropertyGraphMetadata(pg.PropertyGraphMetadataJSON)
		if err != nil {
			return nil, fmt.Errorf("property graph %s: %w", pg.PropertyGraphName, err)
		}
		sql, err := buildPropertyGraphDDL(s, meta)
		if err != nil {
			return nil, fmt.Errorf("property graph %s: %w", pg.PropertyGraphName, err)
		}
		stmt, err := memefish.ParseDDL("", sql)
		if err != nil {
			return nil, fmt.Errorf("property graph %s: parse rebuilt SQL: %w", pg.PropertyGraphName, err)
		}
		cpg, ok := stmt.(*ast.CreatePropertyGraph)
		if !ok {
			return nil, fmt.Errorf("property graph %s: rebuilt SQL did not parse as CREATE PROPERTY GRAPH", pg.PropertyGraphName)
		}
		ddls = append(ddls, cpg)
	}
	return ddls, nil
}
