package spanalyzer

import (
	"fmt"

	"github.com/cloudspannerecosystem/memefish/ast"
)

func (c *Catalog) applyCreatePropertyGraph(ddl *ast.CreatePropertyGraph) error {
	name := ddl.Name.Name
	if _, ok := c.PropertyGraphs[name]; ok {
		switch {
		case ddl.IfNotExists:
			return nil
		case !ddl.OrReplace:
			return fmt.Errorf("property graph %s already exists", name)
		}
	}
	graph, err := propertyGraphFromDDL(ddl)
	if err != nil {
		return err
	}
	c.PropertyGraphs[name] = graph
	return nil
}

func propertyGraphFromDDL(ddl *ast.CreatePropertyGraph) (*PropertyGraph, error) {
	graph := &PropertyGraph{
		Name:   ddl.Name.Name,
		RawSQL: ddl.SQL(),
	}
	if ddl.Content == nil || ddl.Content.NodeTables == nil || ddl.Content.NodeTables.Tables == nil {
		return nil, fmt.Errorf("property graph %s has no node tables", ddl.Name.Name)
	}
	for _, elem := range ddl.Content.NodeTables.Tables.Elements {
		node, err := graphElementFromAST(elem)
		if err != nil {
			return nil, fmt.Errorf("property graph %s node table %s: %w", graph.Name, elem.Name.Name, err)
		}
		graph.NodeTables = append(graph.NodeTables, node)
	}
	if ddl.Content.EdgeTables != nil && ddl.Content.EdgeTables.Tables != nil {
		for _, elem := range ddl.Content.EdgeTables.Tables.Elements {
			edge, err := graphElementFromAST(elem)
			if err != nil {
				return nil, fmt.Errorf("property graph %s edge table %s: %w", graph.Name, elem.Name.Name, err)
			}
			graph.EdgeTables = append(graph.EdgeTables, edge)
		}
	}
	return graph, nil
}

func graphElementFromAST(elem *ast.PropertyGraphElement) (*GraphElement, error) {
	out := &GraphElement{Name: elem.Name.Name}
	if elem.Alias != nil {
		out.Alias = elem.Alias.Name
	}
	switch keys := elem.Keys.(type) {
	case nil:
	case *ast.PropertyGraphNodeElementKey:
		out.KeyColumns = columnNames(keys.Key.Keys)
	case *ast.PropertyGraphEdgeElementKeys:
		if keys.Element != nil {
			out.KeyColumns = columnNames(keys.Element.Keys)
		}
		out.Source = graphEndpointFromSourceKey(keys.Source)
		out.Destination = graphEndpointFromDestinationKey(keys.Destination)
	default:
		return nil, fmt.Errorf("unsupported graph element keys %T", keys)
	}
	labels, propertiesSQL, err := graphLabelsAndProperties(elem.Properties)
	if err != nil {
		return nil, err
	}
	out.Labels = labels
	out.PropertiesSQL = propertiesSQL
	if elem.DynamicLabel != nil {
		out.DynamicLabel = elem.DynamicLabel.ColumnName.Name
	}
	if elem.DynamicProperties != nil {
		out.DynamicProperties = elem.DynamicProperties.ColumnName.Name
	}
	return out, nil
}

func graphEndpointFromSourceKey(key *ast.PropertyGraphSourceKey) *GraphEndpoint {
	if key == nil {
		return nil
	}
	return &GraphEndpoint{
		KeyColumns:       columnNames(key.Keys),
		ElementReference: key.ElementReference.Name,
		ReferenceColumns: columnNames(key.ReferenceColumns),
	}
}

func graphEndpointFromDestinationKey(key *ast.PropertyGraphDestinationKey) *GraphEndpoint {
	if key == nil {
		return nil
	}
	return &GraphEndpoint{
		KeyColumns:       columnNames(key.Keys),
		ElementReference: key.ElementReference.Name,
		ReferenceColumns: columnNames(key.ReferenceColumns),
	}
}

func graphLabelsAndProperties(node ast.PropertyGraphLabelsOrProperties) ([]*GraphLabelProperties, string, error) {
	switch node := node.(type) {
	case nil:
		return nil, "", nil
	case *ast.PropertyGraphSingleProperties:
		return nil, graphPropertiesSQL(node.Properties), nil
	case *ast.PropertyGraphLabelAndPropertiesList:
		labels := make([]*GraphLabelProperties, 0, len(node.LabelAndProperties))
		for _, item := range node.LabelAndProperties {
			label, err := graphLabelProperties(item)
			if err != nil {
				return nil, "", err
			}
			labels = append(labels, label)
		}
		return labels, "", nil
	default:
		return nil, "", fmt.Errorf("unsupported graph labels/properties %T", node)
	}
}

func graphLabelProperties(node *ast.PropertyGraphLabelAndProperties) (*GraphLabelProperties, error) {
	label := &GraphLabelProperties{PropertiesSQL: graphPropertiesSQL(node.Properties)}
	switch name := node.Label.(type) {
	case *ast.PropertyGraphElementLabelLabelName:
		label.Name = name.Name.Name
	case *ast.PropertyGraphElementLabelDefaultLabel:
		label.Default = true
	default:
		return nil, fmt.Errorf("unsupported graph label %T", name)
	}
	return label, nil
}

func graphPropertiesSQL(node ast.PropertyGraphElementProperties) string {
	if node == nil {
		return ""
	}
	return node.SQL()
}

func columnNames(list *ast.PropertyGraphColumnNameList) []string {
	if list == nil {
		return nil
	}
	out := make([]string, 0, len(list.ColumnNameList))
	for _, ident := range list.ColumnNameList {
		out = append(out, ident.Name)
	}
	return out
}
