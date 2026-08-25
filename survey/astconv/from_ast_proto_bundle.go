package astconv

import (
	"github.com/cloudspannerecosystem/memefish/ast"
)

func fromCreateProtoBundle(s *Schema, cpb *ast.CreateProtoBundle) error {
	if cpb.Types == nil {
		return nil
	}
	for _, t := range cpb.Types.Types {
		s.ProtoBundleTypes = append(s.ProtoBundleTypes, t.SQL())
	}
	return nil
}
