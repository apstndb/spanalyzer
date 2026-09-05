package querygen

import (
	"fmt"
	"sort"
	"strings"
)

type packageNamespace struct {
	symbols map[string]string
}

func newPackageNamespace() *packageNamespace {
	return &packageNamespace{symbols: map[string]string{}}
}

func (ns *packageNamespace) claim(name, origin string) error {
	if name == "" {
		return fmt.Errorf("generated symbol origin %s has an empty name", origin)
	}
	if previous := ns.symbols[name]; previous != "" {
		if compatibleSharedStruct(previous, origin) {
			return nil
		}
		return fmt.Errorf("generated symbol %s is emitted by both %s and %s", name, previous, origin)
	}
	ns.symbols[name] = origin
	return nil
}

func compatibleSharedStruct(previous, origin string) bool {
	if isQueryResultStructOrigin(previous) && isQueryResultStructOrigin(origin) {
		return true
	}
	if isWriteInputStructOrigin(previous) && isWriteInputStructOrigin(origin) {
		return true
	}
	return (isQueryResultStructOrigin(previous) && isWriteInputStructOrigin(origin)) ||
		(isWriteInputStructOrigin(previous) && isQueryResultStructOrigin(origin))
}

func isQueryResultStructOrigin(origin string) bool {
	return strings.Contains(origin, " result struct") && !strings.Contains(origin, " nested struct ")
}

func isWriteInputStructOrigin(origin string) bool {
	return strings.Contains(origin, " input struct") && !strings.Contains(origin, " nested struct ")
}

func queryOrigin(spec resolvedQuerySpec, role string) string {
	return fmt.Sprintf("queries[%d] (%s) %s", spec.Index, spec.Name, role)
}

func writeOrigin(spec resolvedWriteSpec, role string) string {
	return fmt.Sprintf("writes[%d] (%s) %s", spec.Index, spec.Name, role)
}

func sortedFieldMapKeys(m map[string][]goResultField) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func emittedSQLConstantName(raw, fallback string) string {
	name := exportedIdentifier(raw, fallback)
	if !strings.HasSuffix(name, "SQL") {
		name += "SQL"
	}
	return name
}

func querySQLConstantNames(query QueryCodegenQuery) []string {
	if queryHasOptionalMarkers(query) {
		return nil
	}
	prefix := exportedIdentifier(query.Name, "Query")
	if !query.Federated.isZero() {
		return []string{
			emittedSQLConstantName(query.Name+"SpannerSQL", prefix+"SpannerSQL"),
			emittedSQLConstantName(query.Name+"BigQuerySQL", prefix+"BigQuerySQL"),
		}
	}
	return []string{emittedSQLConstantName(query.Name, prefix+"SQL")}
}

func queryPrimarySQLConstantName(query QueryCodegenQuery) string {
	names := querySQLConstantNames(query)
	if len(names) == 0 {
		return ""
	}
	return names[len(names)-1]
}

func newResolvedQuerySpec(schemas map[string]QueryCodegenSchema, query QueryCodegenQuery, structName string, index int) (resolvedQuerySpec, error) {
	sourceName, err := querySourceName(schemas, query)
	if err != nil {
		return resolvedQuerySpec{}, err
	}
	methodPrefix := exportedIdentifier(query.Name, "Query")
	spec := resolvedQuerySpec{
		Index:        index,
		Name:         query.Name,
		MethodPrefix: methodPrefix,
		ResultStruct: structName,
		ResultMode:   queryResultMode(query),
		Params:       query.Params,
		Dialect:      emptyDefault(schemas[sourceName].Dialect, "spanner"),
	}
	if queryHasOptionalMarkers(query) {
		spec.BuilderFunc = "Build" + methodPrefix + "SQL"
		spec.ParamsType = methodPrefix + "Params"
	} else {
		spec.ConstantNames = querySQLConstantNames(query)
		spec.SQLConstName = queryPrimarySQLConstantName(query)
	}
	return spec, nil
}

func validateGeneratedPackageNamespace(querySpecs []resolvedQuerySpec, writeSpecs []resolvedWriteSpec, structs map[string][]goResultField, writeStructFields map[string][]goResultField, target GoStructTarget) error {
	ns := newPackageNamespace()
	// Only query-result structs use the configured client target. Write-only
	// structs are rendered separately with the Spanner target by emitWriteCode,
	// so their nullable fields never emit BigQuery support declarations. Shared
	// query/write structs already belong to structs and still reserve helpers.
	named := make([]namedGoStruct, 0, len(structs))
	for _, name := range sortedFieldMapKeys(structs) {
		named = append(named, namedGoStruct{Name: name, Fields: structs[name]})
	}
	for _, name := range activeSupportSymbols(target, querySpecs, named) {
		if err := ns.claim(name, "generated support "+name); err != nil {
			return err
		}
	}
	for _, spec := range querySpecs {
		if err := ns.claim(spec.MethodPrefix, queryOrigin(spec, "function")); err != nil {
			return err
		}
		if spec.ResultMode == "many" {
			if err := ns.claim(spec.MethodPrefix+"All", queryOrigin(spec, "all helper")); err != nil {
				return err
			}
		}
		if err := ns.claim(spec.ResultStruct, queryOrigin(spec, "result struct")); err != nil {
			return err
		}
		for _, constantName := range spec.ConstantNames {
			role := "sql constant"
			switch {
			case strings.HasSuffix(constantName, "SpannerSQL"):
				role = "spanner sql constant"
			case strings.HasSuffix(constantName, "BigQuerySQL"):
				role = "bigquery sql constant"
			}
			if err := ns.claim(constantName, queryOrigin(spec, role)); err != nil {
				return err
			}
		}
		if spec.ParamsType != "" {
			if err := ns.claim(spec.ParamsType, queryOrigin(spec, "optional params type")); err != nil {
				return err
			}
		}
		if spec.BuilderFunc != "" {
			if err := ns.claim(spec.BuilderFunc, queryOrigin(spec, "sql builder")); err != nil {
				return err
			}
		}
	}
	resultOwners := map[string]resolvedQuerySpec{}
	for _, spec := range querySpecs {
		if _, ok := resultOwners[spec.ResultStruct]; !ok {
			resultOwners[spec.ResultStruct] = spec
		}
	}
	for _, name := range sortedFieldMapKeys(structs) {
		origin := "result struct " + name
		if spec, ok := resultOwners[name]; ok {
			origin = queryOrigin(spec, "result struct")
		}
		if err := claimNestedStructs(ns, name, structs[name], origin); err != nil {
			return err
		}
	}
	for _, spec := range writeSpecs {
		if err := ns.claim(spec.InputStruct, writeOrigin(spec, "input struct")); err != nil {
			return err
		}
		for _, method := range spec.Methods {
			if method == "dml" {
				if err := ns.claim(spec.MethodPrefix+"DML", writeOrigin(spec, "dml constant")); err != nil {
					return err
				}
			}
		}
	}
	writeOwners := map[string]resolvedWriteSpec{}
	for _, spec := range writeSpecs {
		if _, ok := writeOwners[spec.InputStruct]; !ok {
			writeOwners[spec.InputStruct] = spec
		}
	}
	for _, name := range sortedFieldMapKeys(writeStructFields) {
		if _, shared := structs[name]; shared {
			continue
		}
		origin := "write input struct " + name
		if spec, ok := writeOwners[name]; ok {
			origin = writeOrigin(spec, "input struct")
		}
		if err := claimNestedStructs(ns, name, writeStructFields[name], origin); err != nil {
			return err
		}
	}
	return validateWriteReceiverMethods(writeSpecs)
}

type receiverMethodNamespace struct {
	methods map[string]string
}

func newReceiverMethodNamespace() *receiverMethodNamespace {
	return &receiverMethodNamespace{methods: map[string]string{}}
}

func receiverMethodKey(receiver, method string) string {
	return receiver + "\x00" + method
}

func (ns *receiverMethodNamespace) claim(receiver, method, origin string) error {
	if receiver == "" || method == "" {
		return fmt.Errorf("generated method origin %s has an empty receiver or name", origin)
	}
	key := receiverMethodKey(receiver, method)
	if previous := ns.methods[key]; previous != "" {
		return fmt.Errorf("generated method %s on %s is emitted by both %s and %s", method, receiver, previous, origin)
	}
	ns.methods[key] = origin
	return nil
}

func validateWriteReceiverMethods(writeSpecs []resolvedWriteSpec) error {
	ns := newReceiverMethodNamespace()
	for _, spec := range writeSpecs {
		for _, method := range spec.Methods {
			switch method {
			case "mutation":
				if err := ns.claim(spec.InputStruct, spec.MethodPrefix+"Mutation", writeOrigin(spec, "mutation method")); err != nil {
					return err
				}
			case "dml":
				if err := ns.claim(spec.InputStruct, spec.MethodPrefix+"DMLStatement", writeOrigin(spec, "dml statement method")); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func claimNestedStructs(ns *packageNamespace, parent string, fields []goResultField, origin string) error {
	for i, field := range fields {
		if !isStructLikeGoResultField(field) {
			continue
		}
		fieldName := exportedIdentifier(field.Name, fmt.Sprintf("Field%d", i+1))
		nested := parent + fieldName
		if err := ns.claim(nested, origin+" nested struct "+nested); err != nil {
			return err
		}
		elem := field
		elem.Repeated = false
		if err := claimNestedStructs(ns, nested, elem.Fields, origin); err != nil {
			return err
		}
	}
	return nil
}

func activeSupportSymbols(target GoStructTarget, querySpecs []resolvedQuerySpec, structs []namedGoStruct) []string {
	if target == "" {
		target = GoStructTargetBoth
	}
	var out []string
	if specsNeedSpannerQueryTransaction(querySpecs) {
		out = append(out, "SpannerQueryTransaction")
	}
	gen := &goStructGenerator{target: target, imports: map[string]string{}, usedOrigins: map[string]string{}}
	for _, st := range structs {
		name := st.Name
		if name == "" {
			name = "QueryRow"
		}
		exported := exportedIdentifier(name, "QueryRow")
		gen.buildStruct(exported, "generated support struct "+exported, st.Fields)
	}
	if gen.needsNullValue {
		out = append(out, "NullValue", "loadBigQueryNullValueSlice")
	}
	if gen.needsAssignValue || gen.needsNullValue {
		out = append(out, "assignBigQueryValue")
	}
	if gen.needsValueSlice {
		out = append(out, "loadBigQueryValueSlice")
	}
	return out
}
