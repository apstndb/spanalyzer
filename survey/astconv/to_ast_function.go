package astconv

import (
	"fmt"
	"sort"
	"strings"

	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/apstndb/spanalyzer/survey/spannertype"
	"github.com/cloudspannerecosystem/memefish"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func preferredRoutineType(spannerType, dataType *string) string {
	if spannerType != nil && strings.TrimSpace(*spannerType) != "" {
		return strings.TrimSpace(*spannerType)
	}
	if dataType != nil && strings.TrimSpace(*dataType) != "" {
		return strings.TrimSpace(*dataType)
	}
	return ""
}

func routineOptionMetadata(options []*infoschem.RoutineOption, name string) string {
	for _, option := range options {
		if strings.EqualFold(option.OptionName, name) {
			value := strings.TrimSpace(option.OptionValue)
			if len(value) >= 2 &&
				((value[0] == '\'' && value[len(value)-1] == '\'') ||
					(value[0] == '"' && value[len(value)-1] == '"')) {
				value = value[1 : len(value)-1]
			}
			return strings.ToUpper(value)
		}
	}
	return ""
}

func isRoutineClauseMetadata(optionName string) bool {
	return strings.EqualFold(optionName, "LANGUAGE") || strings.EqualFold(optionName, "DETERMINISM")
}

func (s *Schema) toFunctionsDDL() ([]ast.DDL, error) {
	paramsByRoutine := groupBy(s.Parameters, func(p *infoschem.Parameter) string {
		return qualifiedTableKey(p.SpecificSchema, p.SpecificName)
	})
	optsByRoutine := groupBy(s.RoutineOptions, func(o *infoschem.RoutineOption) string {
		return qualifiedTableKey(o.SpecificSchema, o.SpecificName)
	})

	var ddls []ast.DDL
	for _, r := range s.Routines {
		if r.RoutineType != "FUNCTION" {
			continue
		}
		// Skip system schema routines, but never silently drop user DDL.
		if isSystemSchemaName(r.RoutineSchema) {
			continue
		}
		if r.RoutineSchema != r.SpecificSchema {
			return nil, fmt.Errorf(
				"function %q has inconsistent routine schema %q and specific schema %q",
				r.RoutineName,
				r.RoutineSchema,
				r.SpecificSchema,
			)
		}

		key := qualifiedTableKey(r.SpecificSchema, r.SpecificName)

		cf := &ast.CreateFunction{
			Name: schemaObjectPath(r.RoutineSchema, r.RoutineName),
		}

		// Parameters
		params := paramsByRoutine[key]
		sort.Slice(params, func(i, j int) bool {
			return params[i].OrdinalPosition < params[j].OrdinalPosition
		})
		for _, p := range params {
			if p.ParameterName == nil {
				continue
			}
			fp := &ast.FunctionParam{
				Name: ident(*p.ParameterName),
			}
			if typeText := preferredRoutineType(p.SpannerType, p.DataType); typeText != "" {
				schemaType, err := spannertype.ParseFunctionType(typeText)
				if err != nil {
					return nil, fmt.Errorf("function %s param %s type %q: %w", r.RoutineName, *p.ParameterName, typeText, err)
				}
				fp.Type = schemaType
			}
			if p.ParameterDefault != nil && *p.ParameterDefault != "" {
				expr, err := memefish.ParseExpr("", *p.ParameterDefault)
				if err != nil {
					return nil, fmt.Errorf("function %s param %s: parse default %q: %w", r.RoutineName, *p.ParameterName, *p.ParameterDefault, err)
				}
				fp.DefaultExpr = expr
			}
			cf.Params = append(cf.Params, fp)
		}

		// Return type
		if typeText := preferredRoutineType(r.SpannerType, r.DataType); typeText != "" {
			schemaType, err := spannertype.ParseFunctionType(typeText)
			if err != nil {
				return nil, fmt.Errorf("function %s return type %q: %w", r.RoutineName, typeText, err)
			}
			cf.ReturnType = schemaType
		}

		// SQL SECURITY
		switch r.SecurityType {
		case "INVOKER":
			cf.SqlSecurity = ast.SecurityTypeInvoker
		case "DEFINER":
			cf.SqlSecurity = ast.SecurityTypeDefiner
		}

		// Remote UDF metadata is retained on the routine model for AST
		// round-trips. Routine options are accepted as a compatibility fallback
		// for callers that materialize this metadata there.
		language := r.Language
		if language == "" {
			language = routineOptionMetadata(optsByRoutine[key], "LANGUAGE")
		}
		cf.Language = strings.ToUpper(language)
		determinism := r.Determinism
		if determinism == "" {
			determinism = routineOptionMetadata(optsByRoutine[key], "DETERMINISM")
		}
		cf.Determinism = ast.Determinism(strings.ToUpper(determinism))
		cf.Remote = r.Remote
		if !cf.Remote && cf.Language == "" &&
			(strings.EqualFold(r.RoutineBody, "REMOTE") || strings.EqualFold(r.RoutineBody, "EXTERNAL")) {
			return nil, fmt.Errorf(
				"function %s has %s routine body but no LANGUAGE/REMOTE metadata; refusing to guess remote-function syntax",
				r.RoutineName,
				r.RoutineBody,
			)
		}

		// Definition
		if r.RoutineDefinition != nil && *r.RoutineDefinition != "" {
			expr, err := memefish.ParseExpr("", *r.RoutineDefinition)
			if err != nil {
				return nil, fmt.Errorf("function %s: parse definition %q: %w", r.RoutineName, *r.RoutineDefinition, err)
			}
			cf.Definition = expr
		}

		// Options
		if opts := optsByRoutine[key]; len(opts) > 0 {
			var defs []*ast.OptionsDef
			for _, o := range opts {
				if isRoutineClauseMetadata(o.OptionName) {
					continue
				}
				defs = append(defs, optionsDef(o.OptionName, parseOptionValue(o.OptionType, o.OptionValue)))
			}
			cf.Options = mkOptions(defs...)
		}

		ddls = append(ddls, cf)
	}
	return ddls, nil
}
