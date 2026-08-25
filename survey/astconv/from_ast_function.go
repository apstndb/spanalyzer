package astconv

import (
	"fmt"
	"strings"

	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func fromCreateFunction(s *Schema, cf *ast.CreateFunction) error {
	if cf.Name == nil || len(cf.Name.Idents) == 0 {
		return fmt.Errorf("function has no name")
	}
	funcSchema, funcName, err := schemaObjectName("function", cf.Name)
	if err != nil {
		return err
	}

	r := &infoschem.Routine{
		SpecificSchema: funcSchema,
		SpecificName:   funcName,
		RoutineSchema:  funcSchema,
		RoutineName:    funcName,
		RoutineType:    "FUNCTION",
		Language:       cf.Language,
		Determinism:    string(cf.Determinism),
		Remote:         cf.Remote,
	}

	if cf.ReturnType != nil {
		dt := cf.ReturnType.SQL()
		r.DataType = &dt
		r.SpannerType = &dt
	}

	switch cf.SqlSecurity {
	case ast.SecurityTypeInvoker:
		r.SecurityType = "INVOKER"
	case ast.SecurityTypeDefiner:
		r.SecurityType = "DEFINER"
	}

	if cf.Remote || strings.EqualFold(cf.Language, "REMOTE") {
		r.RoutineBody = "REMOTE"
	} else {
		r.RoutineBody = "SQL"
	}

	if cf.Definition != nil {
		def := cf.Definition.SQL()
		r.RoutineDefinition = &def
	}

	s.Routines = append(s.Routines, r)

	// Parameters
	for i, p := range cf.Params {
		param := &infoschem.Parameter{
			SpecificSchema:  funcSchema,
			SpecificName:    funcName,
			OrdinalPosition: int64(i + 1),
		}
		if p.Name != nil {
			param.ParameterName = strPtr(p.Name.Name)
		}
		if p.Type != nil {
			dt := p.Type.SQL()
			param.DataType = &dt
			param.SpannerType = &dt
		}
		if p.DefaultExpr != nil {
			def := p.DefaultExpr.SQL()
			param.ParameterDefault = &def
		}
		s.Parameters = append(s.Parameters, param)
	}

	// Options
	if cf.Options != nil {
		for _, opt := range cf.Options.Records {
			s.RoutineOptions = append(s.RoutineOptions, &infoschem.RoutineOption{
				SpecificSchema: funcSchema,
				SpecificName:   funcName,
				OptionName:     opt.Name.Name,
				OptionType:     inferOptionType(opt.Value),
				OptionValue:    opt.Value.SQL(),
			})
		}
	}

	return nil
}
