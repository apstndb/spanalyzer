package astconv

import (
	"strings"

	"github.com/apstndb/spanalyzer/survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func (s *Schema) toSequencesDDL() ([]ast.DDL, error) {
	optsBySeq := groupBy(s.SequenceOptions, func(o *infoschem.SequenceOption) string {
		return qualifiedTableKey(o.Schema, o.Name)
	})

	var ddls []ast.DDL
	for _, seq := range s.Sequences {
		key := qualifiedTableKey(seq.Schema, seq.Name)

		cs := &ast.CreateSequence{
			Name: schemaObjectPath(seq.Schema, seq.Name),
		}

		// Build sequence params and options from SEQUENCE_OPTIONS
		if opts := optsBySeq[key]; len(opts) > 0 {
			var defs []*ast.OptionsDef
			var skipMin, skipMax string
			var hasSkipRange bool
			for _, o := range opts {
				switch o.OptionName {
				case "sequence_kind":
					v := strings.Trim(o.OptionValue, "'\"")
					if strings.EqualFold(v, "bit_reversed_positive") {
						cs.Params = append(cs.Params, &ast.BitReversedPositive{})
					}
				case "skip_range_min":
					skipMin = strings.Trim(o.OptionValue, "'\"")
					hasSkipRange = true
				case "skip_range_max":
					skipMax = strings.Trim(o.OptionValue, "'\"")
					hasSkipRange = true
				case "start_with_counter":
					v := strings.Trim(o.OptionValue, "'\"")
					cs.Params = append(cs.Params, &ast.StartCounterWith{
						Counter: intval(v),
					})
				default:
					defs = append(defs, optionsDef(o.OptionName, parseOptionValue(o.OptionType, o.OptionValue)))
				}
			}
			if hasSkipRange && skipMin != "" && skipMax != "" {
				cs.Params = append(cs.Params, &ast.SkipRange{
					Min: intval(skipMin),
					Max: intval(skipMax),
				})
			}
			cs.Options = mkOptions(defs...)
		}

		ddls = append(ddls, cs)
	}
	return ddls, nil
}
