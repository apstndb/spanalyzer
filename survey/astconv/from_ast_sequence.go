package astconv

import (
	"fmt"

	"github.com/apstndb/spanner-emulator-survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func fromCreateSequence(s *Schema, cs *ast.CreateSequence) error {
	if cs.Name == nil || len(cs.Name.Idents) == 0 {
		return fmt.Errorf("unsupported sequence with missing name")
	}
	seqSchema, seqName, err := schemaObjectName("sequence", cs.Name)
	if err != nil {
		return err
	}

	seq := &infoschem.Sequence{
		Schema:   seqSchema,
		Name:     seqName,
		DataType: "INT64",
	}

	// Sequence params → SequenceOptions
	for _, p := range cs.Params {
		switch param := p.(type) {
		case *ast.BitReversedPositive:
			s.SequenceOptions = append(s.SequenceOptions, &infoschem.SequenceOption{
				Schema:      seqSchema,
				Name:        seqName,
				OptionName:  "sequence_kind",
				OptionType:  "STRING",
				OptionValue: "'bit_reversed_positive'",
			})
		case *ast.SkipRange:
			s.SequenceOptions = append(s.SequenceOptions, &infoschem.SequenceOption{
				Schema:      seqSchema,
				Name:        seqName,
				OptionName:  "skip_range_min",
				OptionType:  "INT64",
				OptionValue: param.Min.Value,
			})
			s.SequenceOptions = append(s.SequenceOptions, &infoschem.SequenceOption{
				Schema:      seqSchema,
				Name:        seqName,
				OptionName:  "skip_range_max",
				OptionType:  "INT64",
				OptionValue: param.Max.Value,
			})
		case *ast.StartCounterWith:
			s.SequenceOptions = append(s.SequenceOptions, &infoschem.SequenceOption{
				Schema:      seqSchema,
				Name:        seqName,
				OptionName:  "start_with_counter",
				OptionType:  "INT64",
				OptionValue: param.Counter.Value,
			})
		}
	}

	// Remaining options
	if cs.Options != nil {
		for _, opt := range cs.Options.Records {
			s.SequenceOptions = append(s.SequenceOptions, &infoschem.SequenceOption{
				Schema:      seqSchema,
				Name:        seqName,
				OptionName:  opt.Name.Name,
				OptionType:  inferOptionType(opt.Value),
				OptionValue: opt.Value.SQL(),
			})
		}
	}

	s.Sequences = append(s.Sequences, seq)
	return nil
}
