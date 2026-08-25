package astconv

import (
	"fmt"
	"strings"

	"github.com/cloudspannerecosystem/memefish/ast"
)

func (s *Schema) toLocalityGroupsDDL() ([]ast.DDL, error) {
	// Group options by locality group name. LocalityGroupOptions alone is the
	// INFORMATION_SCHEMA representation and can only imply groups with options.
	optsByGroup := make(map[string][]*ast.OptionsDef)
	groupNames := make(map[string]bool)
	orderedNames := make([]string, 0, len(s.LocalityGroups)+len(s.LocalityGroupOptions))
	for _, group := range s.LocalityGroups {
		if !groupNames[group.LocalityGroupName] {
			groupNames[group.LocalityGroupName] = true
			orderedNames = append(orderedNames, group.LocalityGroupName)
		}
	}
	for _, opt := range s.LocalityGroupOptions {
		if !groupNames[opt.LocalityGroupName] {
			groupNames[opt.LocalityGroupName] = true
			orderedNames = append(orderedNames, opt.LocalityGroupName)
		}
		if opt.OptionValue == nil {
			// INFORMATION_SCHEMA emits NULL for an inherited/unset locality
			// option. It describes the group but must not become OPTIONS (... =
			// NULL), which would change the canonical DDL contract.
			continue
		}
		if strings.EqualFold(opt.OptionName, "inflash") &&
			strings.EqualFold(strings.Trim(*opt.OptionValue, "'\""), "BOOL") {
			return nil, fmt.Errorf(
				"locality group %q has malformed emulator metadata inflash = BOOL",
				opt.LocalityGroupName,
			)
		}
		optsByGroup[opt.LocalityGroupName] = append(optsByGroup[opt.LocalityGroupName],
			optionsDef(opt.OptionName, parseOptionValue(opt.OptionType, *opt.OptionValue)))
	}

	var ddls []ast.DDL
	for _, name := range orderedNames {
		clg := &ast.CreateLocalityGroup{
			Name: ident(name),
		}
		if defs := optsByGroup[name]; len(defs) > 0 {
			clg.Options = mkOptions(defs...)
		}
		ddls = append(ddls, clg)
	}
	return ddls, nil
}
