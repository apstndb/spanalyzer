package spanalyzer

import (
	"fmt"
	"strings"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

type QueryCodegenPlanVetSuppression struct {
	Rule    string `json:"rule" yaml:"rule"`
	Reason  string `json:"reason" yaml:"reason"`
	Owner   string `json:"owner,omitempty" yaml:"owner,omitempty"`
	Expires string `json:"expires,omitempty" yaml:"expires,omitempty"`
}

type QueryCodegenPlanWarning struct {
	Rule        string `json:"rule" yaml:"rule"`
	Severity    string `json:"severity" yaml:"severity"`
	Message     string `json:"message" yaml:"message"`
	Remediation string `json:"remediation,omitempty" yaml:"remediation,omitempty"`
}

type QueryCodegenDiagnosticError struct {
	ID      string
	Stage   string
	Subject string
	Message string
	Err     error
}

func (e QueryCodegenDiagnosticError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.ID
}

func (e QueryCodegenDiagnosticError) Unwrap() error {
	return e.Err
}

func queryCodegenDiagnosticError(id, stage, subject, message string) error {
	return QueryCodegenDiagnosticError{
		ID:      id,
		Stage:   stage,
		Subject: subject,
		Message: message,
	}
}

func normalizeSpannerExternalDatasetAccess(access string) (string, error) {
	switch strings.ToLower(emptyDefault(access, "unknown")) {
	case "unknown":
		return "unknown", nil
	case "euc", "end_user_credentials":
		return "end_user_credentials", nil
	case "cloud_resource", "cloud_resource_connection":
		return "cloud_resource_connection", nil
	default:
		return "", fmt.Errorf("unsupported access %q; use unknown, euc/end_user_credentials, or cloud_resource/cloud_resource_connection", access)
	}
}

func validateSpannerExternalDatasetRelationRoles(sql string, bindings []BigQuerySpannerExternalDatasetBinding) error {
	for _, binding := range bindings {
		for _, table := range binding.ProjectedTables {
			role, ok := spannerExternalDatasetTableReferenceRole(sql, binding, table)
			if !ok || role == "select_source" {
				continue
			}
			id := "external-dataset-metadata-target-unsupported"
			if role == "dml_target" {
				id = "external-dataset-dml-target-unsupported"
			}
			return queryCodegenDiagnosticError(id, "bigquery_analysis", table.BigQueryTable, fmt.Sprintf("external dataset table %s is read-only and cannot be used as %s", table.BigQueryTable, role))
		}
		if target, ok := spannerExternalDatasetMetadataTarget(sql, binding); ok {
			return queryCodegenDiagnosticError("external-dataset-metadata-target-unsupported", "bigquery_analysis", target, fmt.Sprintf("external dataset path %s is read-only and cannot be used as metadata_target", target))
		}
	}
	return nil
}

func spannerExternalDatasetTableReferenceRole(sql string, binding BigQuerySpannerExternalDatasetBinding, table BigQuerySpannerExternalDatasetTable) (string, bool) {
	if !sqlReferencesExternalDatasetTable(sql, binding, table) {
		return "", false
	}
	normalizedSQL := normalizeSQLForTableReference(sql)
	names := externalDatasetTableReferenceNames(binding, table)
	for _, name := range names {
		normalizedName := normalizeSQLForTableReference(name)
		if normalizedName == "" {
			continue
		}
		if sqlTargetReference(normalizedSQL, "insert into", normalizedName) ||
			sqlTargetReference(normalizedSQL, "insert", normalizedName) ||
			sqlTargetReference(normalizedSQL, "update", normalizedName) ||
			sqlTargetReference(normalizedSQL, "merge into", normalizedName) ||
			sqlTargetReference(normalizedSQL, "merge", normalizedName) ||
			sqlTargetReference(normalizedSQL, "delete from", normalizedName) {
			return "dml_target", true
		}
		if sqlTargetReference(normalizedSQL, "truncate table", normalizedName) ||
			sqlTargetReference(normalizedSQL, "alter table", normalizedName) ||
			sqlTargetReference(normalizedSQL, "drop table", normalizedName) ||
			sqlTargetReference(normalizedSQL, "create table", normalizedName) ||
			sqlTargetReference(normalizedSQL, "create or replace table", normalizedName) {
			return "metadata_target", true
		}
	}
	return "select_source", true
}

func sqlTargetReference(sql, prefix, table string) bool {
	if !strings.HasPrefix(sql, prefix+" ") {
		return false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(sql, prefix))
	return rest == table ||
		strings.HasPrefix(rest, table+" ") ||
		strings.HasPrefix(rest, table+"(") ||
		strings.HasPrefix(rest, table+"\n")
}

func spannerExternalDatasetMetadataTarget(sql string, binding BigQuerySpannerExternalDatasetBinding) (string, bool) {
	normalizedSQL := normalizeSQLForTableReference(sql)
	prefixes := []string{
		"create or replace materialized view",
		"create materialized view",
		"create or replace view",
		"create or replace table",
		"create table",
		"create view",
		"alter table",
		"drop table",
		"drop view",
		"truncate table",
	}
	for _, prefix := range prefixes {
		if !strings.HasPrefix(normalizedSQL, prefix+" ") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(normalizedSQL, prefix))
		rest = strings.TrimPrefix(rest, "if not exists ")
		rest = strings.TrimPrefix(rest, "if exists ")
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			continue
		}
		target := strings.TrimRight(fields[0], ";")
		if spannerExternalDatasetPathMatches(target, binding) {
			return target, true
		}
	}
	return "", false
}

func spannerExternalDatasetPathMatches(path string, binding BigQuerySpannerExternalDatasetBinding) bool {
	path = normalizeSQLForTableReference(path)
	for _, datasetPath := range []string{binding.BigQueryDatasetRef.Dataset, binding.BigQueryDatasetRef.Path} {
		datasetPath = normalizeSQLForTableReference(datasetPath)
		if datasetPath == "" {
			continue
		}
		if path == datasetPath || strings.HasPrefix(path, datasetPath+".") {
			return true
		}
	}
	return false
}

func sqlReferencesExternalDatasetTable(sql string, binding BigQuerySpannerExternalDatasetBinding, table BigQuerySpannerExternalDatasetTable) bool {
	sql = normalizeSQLForTableReference(sql)
	for _, name := range externalDatasetTableReferenceNames(binding, table) {
		name = normalizeSQLForTableReference(name)
		if name == "" {
			continue
		}
		if strings.Contains(sql, name) {
			return true
		}
	}
	return false
}

func externalDatasetTableReferenceNames(binding BigQuerySpannerExternalDatasetBinding, table BigQuerySpannerExternalDatasetTable) []string {
	return []string{
		table.BigQueryTable,
		table.Name,
		binding.BigQueryDatasetRef.Dataset + "." + table.SpannerTable,
		binding.BigQueryDatasetRef.Path + "." + table.SpannerTable,
	}
}

func normalizeSQLForTableReference(sql string) string {
	var b strings.Builder
	for i := 0; i < len(sql); {
		switch {
		case sql[i] == '\'' || sql[i] == '"':
			b.WriteByte(' ')
			i = skipSQLQuotedString(sql, i, sql[i])
		case sql[i] == '`':
			i++
			for i < len(sql) {
				if sql[i] == '`' {
					i++
					break
				}
				b.WriteByte(sql[i])
				i++
			}
		case strings.HasPrefix(sql[i:], "--"):
			b.WriteByte(' ')
			i = skipSQLLineComment(sql, i)
		case strings.HasPrefix(sql[i:], "/*"):
			b.WriteByte(' ')
			i = skipSQLBlockComment(sql, i)
		default:
			b.WriteByte(sql[i])
			i++
		}
	}
	sql = strings.ToLower(b.String())
	fields := strings.Fields(sql)
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
}

func skipSQLQuotedString(sql string, start int, quote byte) int {
	i := start + 1
	for i < len(sql) {
		if sql[i] == quote {
			i++
			if i < len(sql) && sql[i] == quote {
				i++
				continue
			}
			return i
		}
		i++
	}
	return len(sql)
}

func skipSQLLineComment(sql string, start int) int {
	i := start
	for i < len(sql) && sql[i] != '\n' {
		i++
	}
	return i
}

func skipSQLBlockComment(sql string, start int) int {
	i := start + 2
	for i+1 < len(sql) {
		if sql[i] == '*' && sql[i+1] == '/' {
			return i + 2
		}
		i++
	}
	return len(sql)
}

func typeSpecSQL(spec *TypeSpec) (string, error) {
	if spec == nil {
		return "", fmt.Errorf("nil type")
	}
	if spec.Tokenlist {
		return "TOKENLIST", nil
	}
	switch spec.Code {
	case spannerpb.TypeCode_BOOL:
		return "BOOL", nil
	case spannerpb.TypeCode_INT64:
		return "INT64", nil
	case spannerpb.TypeCode_FLOAT64:
		return "FLOAT64", nil
	case spannerpb.TypeCode_FLOAT32:
		return "FLOAT32", nil
	case spannerpb.TypeCode_TIMESTAMP:
		return "TIMESTAMP", nil
	case spannerpb.TypeCode_DATE:
		return "DATE", nil
	case spannerpb.TypeCode_STRING:
		return "STRING", nil
	case spannerpb.TypeCode_BYTES:
		return "BYTES", nil
	case spannerpb.TypeCode_NUMERIC:
		return "NUMERIC", nil
	case spannerpb.TypeCode_JSON:
		return "JSON", nil
	case spannerpb.TypeCode_UUID:
		return "UUID", nil
	case spannerpb.TypeCode_INTERVAL:
		return "INTERVAL", nil
	case spannerpb.TypeCode_ARRAY:
		elem, err := typeSpecSQL(spec.ArrayElement)
		if err != nil {
			return "", err
		}
		return "ARRAY<" + elem + ">", nil
	case spannerpb.TypeCode_STRUCT:
		fields := make([]string, 0, len(spec.StructFields))
		for _, field := range spec.StructFields {
			fieldType, err := typeSpecSQL(field.Type)
			if err != nil {
				return "", err
			}
			if field.Name != "" {
				fieldType = quoteGoogleSQLIdent(field.Name) + " " + fieldType
			}
			fields = append(fields, fieldType)
		}
		return "STRUCT<" + strings.Join(fields, ", ") + ">", nil
	case spannerpb.TypeCode_PROTO:
		if spec.ProtoTypeFQN == "" {
			return "", fmt.Errorf("unnamed proto type")
		}
		return quoteGoogleSQLIdent(spec.ProtoTypeFQN), nil
	case spannerpb.TypeCode_ENUM:
		if spec.ProtoTypeFQN == "" {
			return "", fmt.Errorf("unnamed enum type")
		}
		return quoteGoogleSQLIdent(spec.ProtoTypeFQN), nil
	default:
		return "", fmt.Errorf("unsupported type code %s", spec.Code)
	}
}

func quoteGoogleSQLIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func emptyDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
