package astconv

import (
	"cloud.google.com/go/spanner"
	"github.com/apstndb/spanner-emulator-survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func fromCreatePropertyGraph(s *Schema, cpg *ast.CreatePropertyGraph) error {
	meta, err := newPropertyGraphMetadataBuilder(s).fromAST(cpg)
	if err != nil {
		return err
	}
	s.PropertyGraphs = append(s.PropertyGraphs, &infoschem.PropertyGraph{
		PropertyGraphName: cpg.Name.Name,
		PropertyGraphMetadataJSON: spanner.NullJSON{
			Value: meta,
			Valid: true,
		},
	})
	return nil
}
