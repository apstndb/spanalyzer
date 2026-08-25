package astconv

import (
	"fmt"

	"github.com/apstndb/spanner-emulator-survey/infoschem"
	"github.com/cloudspannerecosystem/memefish/ast"
)

func defaultSchemaObjectName(kind string, name *ast.Path) (string, error) {
	schemaName, objectName, err := schemaObjectName(kind, name)
	if err != nil {
		return "", err
	}
	if schemaName != "" {
		return "", fmt.Errorf("unsupported named-schema %s %q", kind, name.SQL())
	}
	return objectName, nil
}

func fromCreateRole(s *Schema, cr *ast.CreateRole) error {
	s.Roles = append(s.Roles, &infoschem.Role{
		RoleName: cr.Name.Name,
	})
	return nil
}

func fromRevoke(s *Schema, r *ast.Revoke) error {
	for _, role := range r.Roles {
		if err := revokePrivilege(s, r.Privilege, role.Name); err != nil {
			return err
		}
	}
	return nil
}

func tablePrivilegeType(priv ast.TablePrivilege) string {
	switch priv.(type) {
	case *ast.SelectPrivilege:
		return "SELECT"
	case *ast.InsertPrivilege:
		return "INSERT"
	case *ast.UpdatePrivilege:
		return "UPDATE"
	case *ast.DeletePrivilege:
		return "DELETE"
	}
	return ""
}

func tablePrivilegeColumns(priv ast.TablePrivilege) []*ast.Ident {
	switch p := priv.(type) {
	case *ast.SelectPrivilege:
		return p.Columns
	case *ast.InsertPrivilege:
		return p.Columns
	case *ast.UpdatePrivilege:
		return p.Columns
	default:
		return nil
	}
}

func sequencePrivilegeType(priv ast.TablePrivilege) (string, error) {
	privilegeType := tablePrivilegeType(priv)
	if privilegeType != "SELECT" && privilegeType != "UPDATE" {
		return "", fmt.Errorf("unsupported sequence privilege: %T", priv)
	}
	if len(tablePrivilegeColumns(priv)) != 0 {
		return "", fmt.Errorf("sequence privilege %s cannot contain a column list", privilegeType)
	}
	return privilegeType, nil
}

func sequenceGrantTarget(name *ast.Path) (schemaName, sequenceName string, err error) {
	if name == nil || len(name.Idents) == 0 {
		return "", "", fmt.Errorf("sequence grant target has no name")
	}
	switch len(name.Idents) {
	case 1:
		return "", name.Idents[0].Name, nil
	case 2:
		return name.Idents[0].Name, name.Idents[1].Name, nil
	default:
		return "", "", fmt.Errorf("unsupported sequence grant target %q", name.SQL())
	}
}

func addSequenceGrant(s *Schema, schemaName, sequenceName, privilegeType, grantee string) {
	for _, existing := range s.SequenceGrants {
		if existing.SequenceSchema == schemaName &&
			existing.SequenceName == sequenceName &&
			existing.PrivilegeType == privilegeType &&
			existing.Grantee == grantee {
			return
		}
	}
	s.SequenceGrants = append(s.SequenceGrants, &infoschem.SequenceGrant{
		SequenceSchema: schemaName,
		SequenceName:   sequenceName,
		PrivilegeType:  privilegeType,
		Grantee:        grantee,
	})
}

func removeSequenceGrant(s *Schema, schemaName, sequenceName, privilegeType, grantee string) {
	grants := s.SequenceGrants[:0]
	for _, grant := range s.SequenceGrants {
		if grant.SequenceSchema == schemaName &&
			grant.SequenceName == sequenceName &&
			grant.PrivilegeType == privilegeType &&
			grant.Grantee == grantee {
			continue
		}
		grants = append(grants, grant)
	}
	s.SequenceGrants = grants
}

func addRoleTableGrant(s *Schema, tableSchema, tableName, privilegeType, grantee string) {
	for _, existing := range s.RoleTableGrants {
		if existing.TableSchema == tableSchema &&
			existing.TableName == tableName &&
			existing.PrivilegeType == privilegeType &&
			existing.Grantee == grantee {
			return
		}
	}
	s.RoleTableGrants = append(s.RoleTableGrants, &infoschem.RoleTableGrant{
		TableSchema:   tableSchema,
		TableName:     tableName,
		PrivilegeType: privilegeType,
		Grantee:       grantee,
	})
}

func addRoleColumnGrant(s *Schema, tableSchema, tableName, columnName, privilegeType, grantee string) {
	for _, existing := range s.RoleColumnGrants {
		if existing.TableSchema == tableSchema &&
			existing.TableName == tableName &&
			existing.ColumnName == columnName &&
			existing.PrivilegeType == privilegeType &&
			existing.Grantee == grantee {
			return
		}
	}
	s.RoleColumnGrants = append(s.RoleColumnGrants, &infoschem.RoleColumnGrant{
		TableSchema:   tableSchema,
		TableName:     tableName,
		ColumnName:    columnName,
		PrivilegeType: privilegeType,
		Grantee:       grantee,
	})
}

func addAllSchemaGrant(s *Schema, objectType, privilegeType, schemaName, grantee string) {
	for _, existing := range s.AllSchemaGrants {
		if existing.ObjectType == objectType &&
			existing.PrivilegeType == privilegeType &&
			existing.SchemaName == schemaName &&
			existing.Grantee == grantee {
			return
		}
	}
	s.AllSchemaGrants = append(s.AllSchemaGrants, &infoschem.AllSchemaGrant{
		ObjectType:    objectType,
		PrivilegeType: privilegeType,
		SchemaName:    schemaName,
		Grantee:       grantee,
	})
}

func removeRoleTableGrant(s *Schema, tableSchema, tableName, privilegeType, grantee string) {
	grants := s.RoleTableGrants[:0]
	for _, grant := range s.RoleTableGrants {
		if grant.TableSchema == tableSchema &&
			grant.TableName == tableName &&
			grant.PrivilegeType == privilegeType &&
			grant.Grantee == grantee {
			continue
		}
		grants = append(grants, grant)
	}
	s.RoleTableGrants = grants
}

func removeRoleColumnGrant(s *Schema, tableSchema, tableName, columnName, privilegeType, grantee string) {
	grants := s.RoleColumnGrants[:0]
	for _, grant := range s.RoleColumnGrants {
		if grant.TableSchema == tableSchema &&
			grant.TableName == tableName &&
			grant.ColumnName == columnName &&
			grant.PrivilegeType == privilegeType &&
			grant.Grantee == grantee {
			continue
		}
		grants = append(grants, grant)
	}
	s.RoleColumnGrants = grants
}

func removeAllSchemaGrant(s *Schema, objectType, privilegeType, schemaName, grantee string) {
	grants := s.AllSchemaGrants[:0]
	for _, grant := range s.AllSchemaGrants {
		if grant.ObjectType == objectType &&
			grant.PrivilegeType == privilegeType &&
			grant.SchemaName == schemaName &&
			grant.Grantee == grantee {
			continue
		}
		grants = append(grants, grant)
	}
	s.AllSchemaGrants = grants
}

func revokeTablePrivilege(
	s *Schema,
	tableSchema string,
	tableName string,
	priv ast.TablePrivilege,
	grantee string,
) error {
	privilegeType := tablePrivilegeType(priv)
	if privilegeType == "" {
		return fmt.Errorf("unsupported table privilege in REVOKE: %T", priv)
	}
	columns := tablePrivilegeColumns(priv)
	if len(columns) == 0 {
		removeRoleTableGrant(s, tableSchema, tableName, privilegeType, grantee)
		return nil
	}
	for _, column := range columns {
		removeRoleColumnGrant(s, tableSchema, tableName, column.Name, privilegeType, grantee)
	}
	return nil
}

func revokeAllSchemaPrivilege(s *Schema, objectType, schemaName string, priv ast.TablePrivilege, grantee string) error {
	privilegeType := tablePrivilegeType(priv)
	if privilegeType == "" {
		return fmt.Errorf("unsupported table privilege in REVOKE ON ALL TABLES IN SCHEMA: %T", priv)
	}
	removeAllSchemaGrant(s, objectType, privilegeType, schemaName, grantee)
	return nil
}

func revokePrivilege(s *Schema, privilege ast.Privilege, grantee string) error {
	switch p := privilege.(type) {
	case *ast.PrivilegeOnTable:
		for _, tableName := range p.Names {
			schemaName, name, err := schemaObjectName("table grant target", tableName)
			if err != nil {
				return err
			}
			for _, priv := range p.Privileges {
				if err := revokeTablePrivilege(s, schemaName, name, priv, grantee); err != nil {
					return err
				}
			}
		}
	case *ast.PrivilegeOnAllTablesInSchema:
		for _, schemaName := range p.Schemas {
			name, err := defaultSchemaObjectName("schema", schemaName)
			if err != nil {
				return err
			}
			for _, priv := range p.Privileges {
				if err := revokeAllSchemaPrivilege(s, "TABLES", name, priv, grantee); err != nil {
					return err
				}
			}
		}
	case *ast.PrivilegeOnSequence:
		for _, sequenceName := range p.Names {
			schemaName, name, err := sequenceGrantTarget(sequenceName)
			if err != nil {
				return err
			}
			for _, priv := range p.Privileges {
				privilegeType, err := sequencePrivilegeType(priv)
				if err != nil {
					return err
				}
				removeSequenceGrant(s, schemaName, name, privilegeType, grantee)
			}
		}
	case *ast.PrivilegeOnAllSequencesInSchema:
		for _, schemaName := range p.Schemas {
			name, err := defaultSchemaObjectName("schema", schemaName)
			if err != nil {
				return err
			}
			for _, priv := range p.Privileges {
				privilegeType, err := sequencePrivilegeType(priv)
				if err != nil {
					return err
				}
				removeAllSchemaGrant(s, "SEQUENCES", privilegeType, name, grantee)
			}
		}
	case *ast.SelectPrivilegeOnChangeStream:
		for _, changeStreamName := range p.Names {
			schemaName, name, err := schemaObjectName("change stream grant target", changeStreamName)
			if err != nil {
				return err
			}
			grants := s.RoleChangeStreamGrants[:0]
			for _, grant := range s.RoleChangeStreamGrants {
				if grant.ChangeStreamSchema == schemaName &&
					grant.ChangeStreamName == name &&
					grant.PrivilegeType == "SELECT" && grant.Grantee == grantee {
					continue
				}
				grants = append(grants, grant)
			}
			s.RoleChangeStreamGrants = grants
		}
	case *ast.SelectPrivilegeOnAllChangeStreamsInSchema:
		for _, schemaName := range p.Schemas {
			name, err := defaultSchemaObjectName("schema", schemaName)
			if err != nil {
				return err
			}
			removeAllSchemaGrant(s, "CHANGE_STREAMS", "SELECT", name, grantee)
		}
	case *ast.SelectPrivilegeOnView:
		for _, viewName := range p.Names {
			schemaName, name, err := schemaObjectName("view grant target", viewName)
			if err != nil {
				return err
			}
			removeRoleTableGrant(s, schemaName, name, "SELECT", grantee)
		}
	case *ast.SelectPrivilegeOnAllViewsInSchema:
		for _, schemaName := range p.Schemas {
			name, err := defaultSchemaObjectName("schema", schemaName)
			if err != nil {
				return err
			}
			removeAllSchemaGrant(s, "VIEWS", "SELECT", name, grantee)
		}
	case *ast.ExecutePrivilegeOnTableFunction:
		for _, functionName := range p.Names {
			schemaName, name, err := schemaObjectName("table function grant target", functionName)
			if err != nil {
				return err
			}
			grants := s.RoleRoutineGrants[:0]
			for _, grant := range s.RoleRoutineGrants {
				if grant.SpecificSchema == schemaName &&
					grant.SpecificName == name &&
					grant.PrivilegeType == "EXECUTE" && grant.Grantee == grantee {
					continue
				}
				grants = append(grants, grant)
			}
			s.RoleRoutineGrants = grants
		}
	case *ast.RolePrivilege:
		for _, roleName := range p.Names {
			grants := s.RoleGrantees[:0]
			for _, grant := range s.RoleGrantees {
				if grant.RoleName == roleName.Name && grant.Grantee == grantee {
					continue
				}
				grants = append(grants, grant)
			}
			s.RoleGrantees = grants
		}
	case *ast.UsagePrivilegeOnSchema:
		grants := s.SchemaGrants[:0]
		for _, grant := range s.SchemaGrants {
			matches := grant.Grantee == grantee
			if !p.Default.Invalid() {
				matches = matches && grant.IsDefault
			} else {
				matches = matches && !grant.IsDefault
				schemaMatches := false
				for _, schemaName := range p.Schemas {
					name, err := defaultSchemaObjectName("schema", schemaName)
					if err != nil {
						return err
					}
					if grant.SchemaName == name {
						schemaMatches = true
						break
					}
				}
				matches = matches && schemaMatches
			}
			if matches {
				continue
			}
			grants = append(grants, grant)
		}
		s.SchemaGrants = grants
	default:
		return fmt.Errorf("unsupported REVOKE privilege: %T", privilege)
	}
	return nil
}

func fromGrant(s *Schema, g *ast.Grant) error {
	for _, role := range g.Roles {
		switch p := g.Privilege.(type) {
		case *ast.PrivilegeOnTable:
			for _, tableName := range p.Names {
				schemaName, name, err := schemaObjectName("table grant target", tableName)
				if err != nil {
					return err
				}
				for _, priv := range p.Privileges {
					privType := tablePrivilegeType(priv)
					if privType == "" {
						return fmt.Errorf("unsupported table privilege in GRANT: %T", priv)
					}
					columns := tablePrivilegeColumns(priv)
					if len(columns) == 0 {
						addRoleTableGrant(s, schemaName, name, privType, role.Name)
						continue
					}
					for _, column := range columns {
						addRoleColumnGrant(s, schemaName, name, column.Name, privType, role.Name)
					}
				}
			}
		case *ast.PrivilegeOnAllTablesInSchema:
			for _, schemaName := range p.Schemas {
				name, err := defaultSchemaObjectName("schema", schemaName)
				if err != nil {
					return err
				}
				for _, priv := range p.Privileges {
					privType := tablePrivilegeType(priv)
					if privType == "" {
						return fmt.Errorf("unsupported table privilege in GRANT ON ALL TABLES IN SCHEMA: %T", priv)
					}
					addAllSchemaGrant(s, "TABLES", privType, name, role.Name)
				}
			}
		case *ast.PrivilegeOnSequence:
			for _, sequenceName := range p.Names {
				schemaName, name, err := sequenceGrantTarget(sequenceName)
				if err != nil {
					return err
				}
				for _, priv := range p.Privileges {
					privilegeType, err := sequencePrivilegeType(priv)
					if err != nil {
						return err
					}
					addSequenceGrant(s, schemaName, name, privilegeType, role.Name)
				}
			}
		case *ast.PrivilegeOnAllSequencesInSchema:
			for _, schemaName := range p.Schemas {
				name, err := defaultSchemaObjectName("schema", schemaName)
				if err != nil {
					return err
				}
				for _, priv := range p.Privileges {
					privilegeType, err := sequencePrivilegeType(priv)
					if err != nil {
						return err
					}
					addAllSchemaGrant(s, "SEQUENCES", privilegeType, name, role.Name)
				}
			}
		case *ast.SelectPrivilegeOnChangeStream:
			for _, csName := range p.Names {
				schemaName, name, err := schemaObjectName("change stream grant target", csName)
				if err != nil {
					return err
				}
				s.RoleChangeStreamGrants = append(s.RoleChangeStreamGrants, &infoschem.RoleChangeStreamGrant{
					ChangeStreamSchema: schemaName,
					ChangeStreamName:   name,
					PrivilegeType:      "SELECT",
					Grantee:            role.Name,
				})
			}
		case *ast.SelectPrivilegeOnAllChangeStreamsInSchema:
			for _, schemaName := range p.Schemas {
				name, err := defaultSchemaObjectName("schema", schemaName)
				if err != nil {
					return err
				}
				addAllSchemaGrant(s, "CHANGE_STREAMS", "SELECT", name, role.Name)
			}
		case *ast.SelectPrivilegeOnView:
			for _, viewName := range p.Names {
				schemaName, name, err := schemaObjectName("view grant target", viewName)
				if err != nil {
					return err
				}
				s.RoleTableGrants = append(s.RoleTableGrants, &infoschem.RoleTableGrant{
					TableSchema:   schemaName,
					TableName:     name,
					PrivilegeType: "SELECT",
					Grantee:       role.Name,
				})
			}
		case *ast.SelectPrivilegeOnAllViewsInSchema:
			for _, schemaName := range p.Schemas {
				name, err := defaultSchemaObjectName("schema", schemaName)
				if err != nil {
					return err
				}
				addAllSchemaGrant(s, "VIEWS", "SELECT", name, role.Name)
			}
		case *ast.ExecutePrivilegeOnTableFunction:
			for _, funcName := range p.Names {
				schemaName, name, err := schemaObjectName("table function grant target", funcName)
				if err != nil {
					return err
				}
				s.RoleRoutineGrants = append(s.RoleRoutineGrants, &infoschem.RoleRoutineGrant{
					SpecificSchema: schemaName,
					SpecificName:   name,
					PrivilegeType:  "EXECUTE",
					Grantee:        role.Name,
				})
			}
		case *ast.RolePrivilege:
			for _, roleName := range p.Names {
				s.RoleGrantees = append(s.RoleGrantees, &infoschem.RoleGrantee{
					RoleName: roleName.Name,
					Grantee:  role.Name,
				})
			}
		case *ast.UsagePrivilegeOnSchema:
			if !p.Default.Invalid() {
				s.SchemaGrants = append(s.SchemaGrants, &infoschem.SchemaGrant{
					Grantee:   role.Name,
					IsDefault: true,
				})
			} else {
				for _, schemaName := range p.Schemas {
					name, err := defaultSchemaObjectName("schema", schemaName)
					if err != nil {
						return err
					}
					s.SchemaGrants = append(s.SchemaGrants, &infoschem.SchemaGrant{
						SchemaName: name,
						Grantee:    role.Name,
					})
				}
			}
		default:
			return fmt.Errorf("unsupported GRANT privilege: %T", g.Privilege)
		}
	}
	return nil
}
