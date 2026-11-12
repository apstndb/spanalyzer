package spanalyzer

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// UnmarshalJSON preserves the old probe's NullFiltered field as an input alias.
// A custom decoder must enforce unknown-field checks itself: an outer decoder's
// DisallowUnknownFields does not inspect the contents of UnmarshalJSON methods.
func (o *MutationCountIndexObservation) UnmarshalJSON(data []byte) error {
	type observation MutationCountIndexObservation
	var input struct {
		observation
		CanonicalNullFiltered *bool `json:"null_filtered"`
		LegacyNullFiltered    *bool `json:"NullFiltered"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return err
	}
	if input.CanonicalNullFiltered != nil && input.LegacyNullFiltered != nil && *input.CanonicalNullFiltered != *input.LegacyNullFiltered {
		return fmt.Errorf("conflicting null_filtered and NullFiltered index attributes")
	}
	if input.CanonicalNullFiltered != nil {
		input.observation.NullFiltered = *input.CanonicalNullFiltered
	} else if input.LegacyNullFiltered != nil {
		input.observation.NullFiltered = *input.LegacyNullFiltered
	}
	*o = MutationCountIndexObservation(input.observation)
	return nil
}
