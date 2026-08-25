package spanalyzer

import (
	_ "embed"
	"fmt"
	"sync"

	"cloud.google.com/go/spanner/apiv1/spannerpb"

	"github.com/apstndb/spanalyzer/internal/spannersysmanifest"
)

const spannerSysName = "SPANNER_SYS"

type spannerSysColumn struct {
	name string
	typ  *TypeSpec
}

type spannerSysTable struct {
	name    string
	columns []spannerSysColumn
}

//go:embed spanner_sys_manifest.json
var embeddedSpannerSysManifest []byte

var (
	spannerSysOnce   sync.Once
	spannerSysTables []spannerSysTable
	spannerSysErr    error
)

func (c *Catalog) addSpannerSysTables() error {
	tables, err := loadSpannerSysTables()
	if err != nil {
		return err
	}
	for _, definition := range tables {
		table := &Table{
			Name:    ObjectName{Parts: []string{spannerSysName, definition.name}},
			Columns: make([]*Column, 0, len(definition.columns)),
		}
		for _, column := range definition.columns {
			table.Columns = append(table.Columns, &Column{
				Name: column.name,
				Type: column.typ,
			})
		}
		c.Tables[table.Name.String()] = table
	}
	return nil
}

func loadSpannerSysTables() ([]spannerSysTable, error) {
	spannerSysOnce.Do(func() {
		_, spannerSysTables, spannerSysErr = parseSpannerSysManifest(embeddedSpannerSysManifest)
	})
	if spannerSysErr != nil {
		return nil, spannerSysErr
	}
	return spannerSysTables, nil
}

func parseSpannerSysManifest(data []byte) (*spannersysmanifest.Document, []spannerSysTable, error) {
	manifest, err := spannersysmanifest.Decode(data)
	if err != nil {
		return nil, nil, err
	}
	tables := make([]spannerSysTable, 0, len(manifest.Tables))
	for _, manifestTable := range manifest.Tables {
		if !manifestTable.Project {
			continue
		}
		table := spannerSysTable{
			name:    manifestTable.Name,
			columns: make([]spannerSysColumn, 0, len(manifestTable.Columns)),
		}
		for _, manifestColumn := range manifestTable.Columns {
			if !manifestColumn.Project {
				continue
			}
			typ, err := spannerSysTypeSpec(manifestColumn.Type)
			if err != nil {
				return nil, nil, fmt.Errorf("SPANNER_SYS manifest column %s.%s: %w", manifestTable.Name, manifestColumn.Name, err)
			}
			table.columns = append(table.columns, spannerSysColumn{
				name: manifestColumn.Name,
				typ:  typ,
			})
		}
		if len(table.columns) == 0 {
			return nil, nil, fmt.Errorf("SPANNER_SYS manifest table %q has no projected columns", manifestTable.Name)
		}
		tables = append(tables, table)
	}
	return manifest, tables, nil
}

func spannerSysTypeSpec(descriptor spannersysmanifest.TypeDescriptor) (*TypeSpec, error) {
	switch descriptor.Kind {
	case "scalar":
		spec := &TypeSpec{}
		switch descriptor.Scalar {
		case "BOOL":
			spec.Code = spannerpb.TypeCode_BOOL
		case "INT64":
			spec.Code = spannerpb.TypeCode_INT64
		case "FLOAT64":
			spec.Code = spannerpb.TypeCode_FLOAT64
		case "STRING(MAX)":
			spec.Code = spannerpb.TypeCode_STRING
			spec.Max = true
		case "BYTES(MAX)":
			spec.Code = spannerpb.TypeCode_BYTES
			spec.Max = true
		case "DATE":
			spec.Code = spannerpb.TypeCode_DATE
		case "TIMESTAMP":
			spec.Code = spannerpb.TypeCode_TIMESTAMP
		default:
			return nil, fmt.Errorf("unsupported scalar code %q", descriptor.Scalar)
		}
		return spec, nil
	case "array":
		if descriptor.Element == nil {
			return nil, fmt.Errorf("array descriptor has no element")
		}
		element, err := spannerSysTypeSpec(*descriptor.Element)
		if err != nil {
			return nil, fmt.Errorf("array element: %w", err)
		}
		return &TypeSpec{Code: spannerpb.TypeCode_ARRAY, ArrayElement: element}, nil
	case "struct":
		fields := make([]StructField, 0, len(descriptor.Fields))
		for _, field := range descriptor.Fields {
			typ, err := spannerSysTypeSpec(field.Type)
			if err != nil {
				return nil, fmt.Errorf("struct field %s: %w", field.Name, err)
			}
			fields = append(fields, StructField{Name: field.Name, Type: typ})
		}
		return &TypeSpec{Code: spannerpb.TypeCode_STRUCT, StructFields: fields}, nil
	default:
		return nil, fmt.Errorf("unsupported descriptor kind %q", descriptor.Kind)
	}
}
