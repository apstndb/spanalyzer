package astconv

import (
	"fmt"
	"sort"

	"github.com/cloudspannerecosystem/memefish/ast"
	"github.com/cloudspannerecosystem/memefish/token"
)

func (s *Schema) toRolesDDL() ([]ast.DDL, error) {
	var ddls []ast.DDL

	for _, g := range s.RoleModelGrants {
		return nil, fmt.Errorf(
			"unsupported model grant %q ON MODEL %q TO ROLE %q: memefish v0.8.1 has no model-privilege AST node",
			g.PrivilegeType,
			g.ModelName,
			g.Grantee,
		)
	}

	// Build view name set for distinguishing GRANT ON TABLE vs ON VIEW
	viewNames := make(map[string]bool)
	for _, v := range s.Views {
		viewNames[qualifiedTableKey(v.TableSchema, v.TableName)] = true
	}

	// CREATE ROLE for non-system roles
	for _, r := range s.Roles {
		if r.IsSystem {
			continue
		}
		ddls = append(ddls, &ast.CreateRole{
			Name: ident(r.RoleName),
		})
	}

	// GRANT on tables and views
	type grantKey struct {
		Grantee     string
		TableSchema string
		TableName   string
	}
	grantsByKey := make(map[grantKey][]ast.TablePrivilege)
	tablePrivilegesByKey := make(map[grantKey]map[string]bool)
	var grantOrder []grantKey
	for _, g := range s.RoleTableGrants {
		var priv ast.TablePrivilege
		switch g.PrivilegeType {
		case "SELECT":
			priv = &ast.SelectPrivilege{}
		case "INSERT":
			priv = &ast.InsertPrivilege{}
		case "UPDATE":
			priv = &ast.UpdatePrivilege{}
		case "DELETE":
			priv = &ast.DeletePrivilege{}
		default:
			return nil, fmt.Errorf("invalid table privilege %q on %q for role %q", g.PrivilegeType, g.TableName, g.Grantee)
		}
		k := grantKey{
			Grantee:     g.Grantee,
			TableSchema: g.TableSchema,
			TableName:   g.TableName,
		}
		if _, exists := grantsByKey[k]; !exists {
			grantOrder = append(grantOrder, k)
		}
		if tablePrivilegesByKey[k] == nil {
			tablePrivilegesByKey[k] = make(map[string]bool)
		}
		if tablePrivilegesByKey[k][g.PrivilegeType] {
			continue
		}
		tablePrivilegesByKey[k][g.PrivilegeType] = true
		grantsByKey[k] = append(grantsByKey[k], priv)
	}
	for _, k := range grantOrder {
		privs := grantsByKey[k]
		targetName := tableDisplayName(k.TableSchema, k.TableName)
		targetPath := schemaObjectPath(k.TableSchema, k.TableName)
		if viewNames[qualifiedTableKey(k.TableSchema, k.TableName)] {
			if len(privs) != 1 || tablePrivilegeType(privs[0]) != "SELECT" {
				return nil, fmt.Errorf("unsupported non-SELECT privilege on view %q for role %q", targetName, k.Grantee)
			}
			// Views only get SELECT; emit as GRANT SELECT ON VIEW
			ddls = append(ddls, &ast.Grant{
				Privilege: &ast.SelectPrivilegeOnView{
					Names: []*ast.Path{targetPath},
				},
				Roles: []*ast.Ident{ident(k.Grantee)},
			})
		} else {
			ddls = append(ddls, &ast.Grant{
				Privilege: &ast.PrivilegeOnTable{
					Privileges: privs,
					Names:      []*ast.Path{targetPath},
				},
				Roles: []*ast.Ident{ident(k.Grantee)},
			})
		}
	}

	// GRANT column-scoped SELECT, INSERT, and UPDATE. ROLE_COLUMN_GRANTS also
	// reports columns inherited from a matching table-wide grant, so those rows
	// must be suppressed when the corresponding ROLE_TABLE_GRANTS row exists.
	type columnGrantKey struct {
		Grantee       string
		TableSchema   string
		TableName     string
		PrivilegeType string
	}
	columnsByKey := make(map[columnGrantKey]map[string]bool)
	var columnGrantOrder []columnGrantKey
	for _, g := range s.RoleColumnGrants {
		if g.PrivilegeType != "SELECT" && g.PrivilegeType != "INSERT" && g.PrivilegeType != "UPDATE" {
			return nil, fmt.Errorf(
				"invalid column privilege %q on %q.%q for role %q",
				g.PrivilegeType,
				tableDisplayName(g.TableSchema, g.TableName),
				g.ColumnName,
				g.Grantee,
			)
		}
		tableKey := grantKey{
			Grantee:     g.Grantee,
			TableSchema: g.TableSchema,
			TableName:   g.TableName,
		}
		if tablePrivilegesByKey[tableKey][g.PrivilegeType] {
			continue
		}
		if viewNames[qualifiedTableKey(g.TableSchema, g.TableName)] {
			return nil, fmt.Errorf(
				"unsupported column-scoped %s privilege on view %q for role %q: memefish v0.8.1 has no column-scoped view privilege AST node",
				g.PrivilegeType,
				tableDisplayName(g.TableSchema, g.TableName),
				g.Grantee,
			)
		}
		k := columnGrantKey{
			Grantee:       g.Grantee,
			TableSchema:   g.TableSchema,
			TableName:     g.TableName,
			PrivilegeType: g.PrivilegeType,
		}
		if columnsByKey[k] == nil {
			columnsByKey[k] = make(map[string]bool)
			columnGrantOrder = append(columnGrantOrder, k)
		}
		columnsByKey[k][g.ColumnName] = true
	}
	for _, k := range columnGrantOrder {
		columnNames := make([]string, 0, len(columnsByKey[k]))
		for columnName := range columnsByKey[k] {
			columnNames = append(columnNames, columnName)
		}
		sort.Strings(columnNames)
		columns := make([]*ast.Ident, 0, len(columnNames))
		for _, columnName := range columnNames {
			columns = append(columns, ident(columnName))
		}
		var priv ast.TablePrivilege
		switch k.PrivilegeType {
		case "SELECT":
			priv = &ast.SelectPrivilege{Columns: columns}
		case "INSERT":
			priv = &ast.InsertPrivilege{Columns: columns}
		case "UPDATE":
			priv = &ast.UpdatePrivilege{Columns: columns}
		}
		ddls = append(ddls, &ast.Grant{
			Privilege: &ast.PrivilegeOnTable{
				Privileges: []ast.TablePrivilege{priv},
				Names:      []*ast.Path{schemaObjectPath(k.TableSchema, k.TableName)},
			},
			Roles: []*ast.Ident{ident(k.Grantee)},
		})
	}

	// GRANT SELECT/UPDATE ON SEQUENCE. Group privileges by target without
	// combining different targets, which could otherwise widen permissions.
	type sequenceGrantKey struct {
		Grantee        string
		SequenceSchema string
		SequenceName   string
	}
	sequencePrivilegesByKey := make(map[sequenceGrantKey]map[string]bool)
	var sequenceGrantOrder []sequenceGrantKey
	for _, g := range s.SequenceGrants {
		if g.SequenceName == "" {
			return nil, fmt.Errorf("sequence grant for role %q has an empty target", g.Grantee)
		}
		if g.PrivilegeType != "SELECT" && g.PrivilegeType != "UPDATE" {
			return nil, fmt.Errorf("invalid sequence privilege %q on %q for role %q", g.PrivilegeType, g.SequenceName, g.Grantee)
		}
		k := sequenceGrantKey{
			Grantee:        g.Grantee,
			SequenceSchema: g.SequenceSchema,
			SequenceName:   g.SequenceName,
		}
		if sequencePrivilegesByKey[k] == nil {
			sequencePrivilegesByKey[k] = make(map[string]bool)
			sequenceGrantOrder = append(sequenceGrantOrder, k)
		}
		sequencePrivilegesByKey[k][g.PrivilegeType] = true
	}
	sort.Slice(sequenceGrantOrder, func(i, j int) bool {
		ai, aj := sequenceGrantOrder[i], sequenceGrantOrder[j]
		if ai.Grantee != aj.Grantee {
			return ai.Grantee < aj.Grantee
		}
		if ai.SequenceSchema != aj.SequenceSchema {
			return ai.SequenceSchema < aj.SequenceSchema
		}
		return ai.SequenceName < aj.SequenceName
	})
	for _, k := range sequenceGrantOrder {
		var privileges []ast.TablePrivilege
		if sequencePrivilegesByKey[k]["SELECT"] {
			privileges = append(privileges, &ast.SelectPrivilege{})
		}
		if sequencePrivilegesByKey[k]["UPDATE"] {
			privileges = append(privileges, &ast.UpdatePrivilege{})
		}
		target := path(k.SequenceName)
		if k.SequenceSchema != "" {
			target = path(k.SequenceSchema, k.SequenceName)
		}
		ddls = append(ddls, &ast.Grant{
			Privilege: &ast.PrivilegeOnSequence{
				Privileges: privileges,
				Names:      []*ast.Path{target},
			},
			Roles: []*ast.Ident{ident(k.Grantee)},
		})
	}

	// GRANT SELECT ON CHANGE STREAM
	for _, g := range s.RoleChangeStreamGrants {
		if g.PrivilegeType != "SELECT" {
			return nil, fmt.Errorf("invalid change stream privilege %q on %q for role %q", g.PrivilegeType, g.ChangeStreamName, g.Grantee)
		}
		ddls = append(ddls, &ast.Grant{
			Privilege: &ast.SelectPrivilegeOnChangeStream{
				Names: []*ast.Path{schemaObjectPath(g.ChangeStreamSchema, g.ChangeStreamName)},
			},
			Roles: []*ast.Ident{ident(g.Grantee)},
		})
	}

	// GRANT EXECUTE ON TABLE FUNCTION
	for _, g := range s.RoleRoutineGrants {
		if g.PrivilegeType != "EXECUTE" {
			return nil, fmt.Errorf("invalid routine privilege %q on %q for role %q", g.PrivilegeType, g.SpecificName, g.Grantee)
		}
		ddls = append(ddls, &ast.Grant{
			Privilege: &ast.ExecutePrivilegeOnTableFunction{
				Names: []*ast.Path{schemaObjectPath(g.SpecificSchema, g.SpecificName)},
			},
			Roles: []*ast.Ident{ident(g.Grantee)},
		})
	}

	// GRANT role membership
	for _, g := range s.RoleGrantees {
		if g.RoleName == g.Grantee {
			continue
		}
		ddls = append(ddls, &ast.Grant{
			Privilege: &ast.RolePrivilege{
				Names: []*ast.Ident{ident(g.RoleName)},
			},
			Roles: []*ast.Ident{ident(g.Grantee)},
		})
	}

	// GRANT USAGE ON SCHEMA
	type schemaGrantKey struct {
		Grantee   string
		IsDefault bool
	}
	schemaGrantsByKey := make(map[schemaGrantKey][]*ast.Path)
	var schemaGrantOrder []schemaGrantKey
	for _, g := range s.SchemaGrants {
		k := schemaGrantKey{Grantee: g.Grantee, IsDefault: g.IsDefault}
		if _, exists := schemaGrantsByKey[k]; !exists {
			schemaGrantOrder = append(schemaGrantOrder, k)
		}
		if !g.IsDefault {
			schemaGrantsByKey[k] = append(schemaGrantsByKey[k], path(g.SchemaName))
		}
	}
	for _, k := range schemaGrantOrder {
		priv := &ast.UsagePrivilegeOnSchema{Usage: token.InvalidPos, Default: token.InvalidPos}
		if k.IsDefault {
			priv.Default = token.Pos(1)
		} else {
			priv.Schemas = schemaGrantsByKey[k]
		}
		ddls = append(ddls, &ast.Grant{
			Privilege: priv,
			Roles:     []*ast.Ident{ident(k.Grantee)},
		})
	}

	// GRANT ... ON ALL ... IN SCHEMA
	type allSchemaGrantKey struct {
		Grantee    string
		SchemaName string
		ObjectType string
	}
	allSchemaGrantsByKey := make(map[allSchemaGrantKey][]ast.TablePrivilege)
	var allSchemaGrantOrder []allSchemaGrantKey
	for _, g := range s.AllSchemaGrants {
		k := allSchemaGrantKey{Grantee: g.Grantee, SchemaName: g.SchemaName, ObjectType: g.ObjectType}
		if _, exists := allSchemaGrantsByKey[k]; !exists {
			allSchemaGrantOrder = append(allSchemaGrantOrder, k)
		}
		switch g.ObjectType {
		case "TABLES":
			var priv ast.TablePrivilege
			switch g.PrivilegeType {
			case "SELECT":
				priv = &ast.SelectPrivilege{}
			case "INSERT":
				priv = &ast.InsertPrivilege{}
			case "UPDATE":
				priv = &ast.UpdatePrivilege{}
			case "DELETE":
				priv = &ast.DeletePrivilege{}
			default:
				return nil, fmt.Errorf("invalid privilege %q for ON ALL TABLES IN SCHEMA", g.PrivilegeType)
			}
			allSchemaGrantsByKey[k] = append(allSchemaGrantsByKey[k], priv)
		case "VIEWS", "CHANGE_STREAMS":
			if g.PrivilegeType != "SELECT" {
				return nil, fmt.Errorf("invalid privilege %q for ON ALL %s IN SCHEMA", g.PrivilegeType, g.ObjectType)
			}
		case "SEQUENCES":
			var priv ast.TablePrivilege
			switch g.PrivilegeType {
			case "SELECT":
				priv = &ast.SelectPrivilege{}
			case "UPDATE":
				priv = &ast.UpdatePrivilege{}
			default:
				return nil, fmt.Errorf("invalid privilege %q for ON ALL SEQUENCES IN SCHEMA", g.PrivilegeType)
			}
			allSchemaGrantsByKey[k] = append(allSchemaGrantsByKey[k], priv)
		default:
			return nil, fmt.Errorf("invalid object type %q for schema-wide grant", g.ObjectType)
		}
	}
	sort.Slice(allSchemaGrantOrder, func(i, j int) bool {
		ai, aj := allSchemaGrantOrder[i], allSchemaGrantOrder[j]
		if ai.Grantee != aj.Grantee {
			return ai.Grantee < aj.Grantee
		}
		if ai.SchemaName != aj.SchemaName {
			return ai.SchemaName < aj.SchemaName
		}
		return ai.ObjectType < aj.ObjectType
	})
	for _, k := range allSchemaGrantOrder {
		schemaPath := path(k.SchemaName)
		switch k.ObjectType {
		case "TABLES":
			ddls = append(ddls, &ast.Grant{
				Privilege: &ast.PrivilegeOnAllTablesInSchema{
					Privileges: allSchemaGrantsByKey[k],
					Schemas:    []*ast.Path{schemaPath},
				},
				Roles: []*ast.Ident{ident(k.Grantee)},
			})
		case "VIEWS":
			ddls = append(ddls, &ast.Grant{
				Privilege: &ast.SelectPrivilegeOnAllViewsInSchema{
					Schemas: []*ast.Path{schemaPath},
				},
				Roles: []*ast.Ident{ident(k.Grantee)},
			})
		case "CHANGE_STREAMS":
			ddls = append(ddls, &ast.Grant{
				Privilege: &ast.SelectPrivilegeOnAllChangeStreamsInSchema{
					Schemas: []*ast.Path{schemaPath},
				},
				Roles: []*ast.Ident{ident(k.Grantee)},
			})
		case "SEQUENCES":
			ddls = append(ddls, &ast.Grant{
				Privilege: &ast.PrivilegeOnAllSequencesInSchema{
					Privileges: allSchemaGrantsByKey[k],
					Schemas:    []*ast.Path{schemaPath},
				},
				Roles: []*ast.Ident{ident(k.Grantee)},
			})
		}
	}

	return ddls, nil
}
