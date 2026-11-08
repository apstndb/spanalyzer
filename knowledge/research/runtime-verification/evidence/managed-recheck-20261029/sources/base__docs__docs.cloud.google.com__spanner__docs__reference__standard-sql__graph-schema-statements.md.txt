---
name: documents/docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-schema-statements
uri: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-schema-statements
title: GQL schema statements
description: A managed, mission-critical, globally consistent and scalable relational database service.
data_source: docs.cloud.google.com
---

Graph Query Language (GQL) supports all GoogleSQL schema statements, including the following GQL-specific schema statements:

## Statement list

| Name                                                                                                                                            | Summary                   |
|-------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------|
| [`CREATE PROPERTY GRAPH` statement](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-schema-statements#gql_create_graph) | Creates a property graph. |
| [`DROP PROPERTY GRAPH` statement](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-schema-statements#gql_drop_graph)     | Deletes a property graph. |

## `CREATE PROPERTY GRAPH` statement

### Property graph definition

```
CREATE
  [ OR REPLACE ]
  PROPERTY GRAPH
  [ IF NOT EXISTS ]
  property_graph_name
  property_graph_content
  [ OPTIONS (key=value, ...) ];

property_graph_content:
  node_tables
  [ edge_tables ]

node_tables:
  NODE TABLES element_list

edge_tables:
  EDGE TABLES element_list

element_list:
  (element[, ...])
```

**Description**

Creates a property graph.

> **Note:** all GQL examples in the GQL reference use the [`FinGraph`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-schema-statements#fin_graph) property graph example. To set up this property graph, see [Set up and query Spanner Graph](https://cloud.google.com/spanner/docs/graph/set-up) .

**Definitions**

- `OR REPLACE` : Replaces any property graph with the same name if it exists. If the property graph doesn't exist, creates the property graph. Can't appear with `IF NOT EXISTS` .

- `IF NOT EXISTS` : If any property graph exists with the same name, the `CREATE` statement has no effect. Can't appear with `OR REPLACE` .

- `property_graph_name` : The name of the property graph. This name can be a path expression. This name must not conflict with the name of an existing table, view, or property graph.

- `property_graph_content` : Add the definitions for the nodes and edges in the property graph.

- `node_tables` : A collection of node definitions. A node definition defines a new type of node in the graph.

  The following example represents three node definitions: `Account` , `Customer` , and `GeoLocation` .

  ```
  NODE TABLES (
    Account,
    Customer
      LABEL Client
      PROPERTIES (cid, name),
    Location AS GeoLocation
      DEFAULT LABEL
      PROPERTIES ALL COLUMNS
  )
  ```

- `edge_tables` : A collection of edge definitions. An edge definition defines a new type of edge in the graph. An edge is directed and connects a source and a destination node.

  The following example represents two edge definitions: `Own` and `Transfer` .

  ```
  EDGE TABLES (
    Own
      SOURCE KEY (cid) REFERENCES Customer (cid)
      DESTINATION KEY (aid) REFERENCES Account
      NO PROPERTIES,
    Transfer
      SOURCE KEY (from_id) REFERENCES Account (aid)
      DESTINATION KEY (to_id) REFERENCES Account (aid)
      LABEL Transfer NO PROPERTIES
  )
  ```

- `element_list` : A list of element (node or edge) definitions.

- `element` : Refer to [Element definition](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-schema-statements#element_definition) for details.

- `OPTIONS` : A list of options for the property graph.

  For example, you can set the `validate_element_key_uniqueness` option to validate that the property graph's element keys are unique at schema creation time. Set this option to either `true` or `false` :

  - `true` (default): Spanner validates at schema creation time that the element keys for each node or edge table are unique. The `KEY` clause columns must satisfy one of the following requirements:

  - **Tables** : Backed by a primary key or a unique index on the underlying table.

  - **Views** : The view query conforms to supported view query patterns to guarantee key uniqueness.

  If uniqueness can't be verified, schema creation fails with an error.

  - `false` : Spanner doesn't validate element key uniqueness at schema creation time. This lets you define property graphs using columns that don't have an underlying uniqueness constraint. In this mode, you're responsible for ensuring that the `KEY` clause columns produce unique values for every node or edge row. If duplicate keys exist, queries might produce unexpected results or duplicate graph paths.

### Element definition

```
element:
  element_name
  [ AS element_alias ]
  element_keys
  [ { label_and_properties_list | element_properties } ]
  [ dynamic_label ]
  [ dynamic_properties ]

element_keys:
  { node_element_key | edge_element_keys }

node_element_key:
  [ element_key ]

edge_element_keys:
  [ element_key ]
  source_key
  destination_key

element_key:
  KEY column_name_list

source_key:
  SOURCE KEY edge_column_name_list
  REFERENCES element_alias_reference [ node_column_name_list ]

destination_key:
  DESTINATION KEY edge_column_name_list
  REFERENCES element_alias_reference [ node_column_name_list ]

edge_column_name_list:
  column_name_list

node_column_name_list:
  column_name_list

column_name_list:
  (column_name[, ...])
```

**Description**

Adds an element definition to the property graph. For example:

```
Customer
  LABEL Client
    PROPERTIES (cid, name)
```

In a graph, labels and properties are uniquely identified by their names. Labels and properties with the same name can appear in multiple node or edge definitions. However, labels and properties with the same name must follow these rules:

- Properties with the same name must have the same value type.
- Labels with the same name must expose the same set of properties.

**Definitions**

- `element_name` : The name of the input table from which elements are created.

- `element_alias` : An optional alias. You must use an alias if you use an input table for more than one element definition.

- `element_keys` : The key for a graph element. This uniquely identifies a graph element.

  - By default, the element key is the primary key of the input table.

  - Element keys can be explicitly defined with the `KEY` clause.

  - Columns with uniqueness constraints can be used as element keys.

- `node_element_key` : The element key for a node.

  ```
  KEY (item1_column, item2_column)
  ```

- `edge_element_keys` : The element key, source key, and destination key for an edge.

  ```
  KEY (item1_column, item2_column)
  SOURCE KEY (item1_column) REFERENCES item_node (item_node_column)
  DESTINATION KEY (item2_column) REFERENCES item_node (item_node_column)
  ```

- `element_key` : An optional key that identifies the node or edge element. If `element_key` isn't provided, then the primary key of the table is used.

  ```
  KEY (item1_column, item2_column)
  ```

- `source_key` : The key for the source node of the edge.

  ```
  SOURCE KEY (item1_column) REFERENCES item_node (item_node_column)
  ```

- `destination_key` : The key for the destination node of the edge.

  ```
  DESTINATION KEY (item2_column) REFERENCES item_node (item_node_column)
  ```

- `column_name_list` : One or more columns to assign to a key.

  In `column_name_list` , column names must be unique.

- Reference column name lists:

  - `node_column_name_list` : One or more columns referenced from the node tables.

  - `edge_column_name_list` : One or more columns referenced from the edge tables.

  Referenced columns must exist in the corresponding node or edge table.

  If `node_column_name_list` doesn't exist in `source_key` or `destination_key` , then the `element_keys` of the referenced node are used. In this case, the column order in the `element_keys` must match the column order in the `edge_column_name_list` .

- `element_alias_reference` : The alias of another element to reference.

- `label_and_properties_list` : The list of labels and properties to add to an element. For more information, see [Label and properties list definition](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-schema-statements#label_property_definition) .

- `dynamic_label` : The name of the column that holds dynamic label values. For more information, see the [Dynamic label definition](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-schema-statements#dynamic_label_definition) .

- `dynamic_properties` : The name of the column that holds dynamic properties values. For more information see the [Dynamic properties definition](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-schema-statements#dynamic_properties_definition) .

### Label and properties list definition

```
label_and_properties_list:
  label_and_properties[...]

label_and_properties:
  element_label
  [ element_properties ]

element_label:
  {
    LABEL label_name |
    DEFAULT LABEL [ options_clause ]
  }

options_clause:
  OPTIONS (
    [ description = description_string ]
    [, synonyms = synonym_array ]
  )
```

**Description**

Adds a list of labels and properties to an element.

**Definitions**

- `label_and_properties` : The label to add to the element and the properties exposed by that label. For example:

  ```
  LABEL Tourist PROPERTIES (home_city, home_country)
  ```

  When `label_and_properties` isn't specified, the following is applied implicitly:

  ```
  DEFAULT LABEL PROPERTIES ARE ALL COLUMNS
  ```

  A property must be unique in `label_and_properties` .

- `element_label` : Add a custom label or use the default label for the element. `label_name` must be unique in `element` .

  If you use `DEFAULT LABEL` , `label_name` is the same as `element_table_alias` .

- `element_properties` : The properties associated with a label. A property can't be used more than once for a specific label. For more information, see [Element properties definition](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-schema-statements#element_table_property_definition) .

- `description_string` : A string literal that describes the label to provide context and improve discoverability for natural language querying interfaces.

- `synonym_array` : An array of string literals that provides alternative names for the label to improve accuracy for natural language querying interfaces.

### Element properties definition

```
element_properties:
  {
    NO PROPERTIES |
    properties_are |
    derived_property_list
  }

properties_are:
  PROPERTIES [ ARE ] ALL COLUMNS [ EXCEPT column_name_list ]

column_name_list:
  (column_name[, ...])

derived_property_list:
  PROPERTIES (derived_property[, ...])

derived_property:
  value_expression [ AS property_name ] [ options_clause ]

options_clause:
  OPTIONS (
    [ description = description_string ]
    [, synonyms = synonym_array ]
  )
```

**Description**

Adds properties associated with a label.

**Definitions**

- `NO PROPERTIES` : The element doesn't have properties.

- `properties_are` : Define which columns to include as element properties.

  If you don't include this definition, all columns are included by default, and the following definition is applied implicitly:

  ```
  PROPERTIES ARE ALL COLUMNS
  ```

  In the following examples, all columns in a table are included as element properties:

  ```
  PROPERTIES ARE ALL COLUMNS
  ```

  ```
  PROPERTIES ALL COLUMNS
  ```

  In the following example, all columns in a table except for `home_city` and `home_country` are included as element properties:

  ```
  PROPERTIES ARE ALL COLUMNS EXCEPT (home_city, home_country)
  ```

- `column_name_list` : A list of columns to exclude as element properties.

  Column names in the `EXCEPT column_name_list` must be unique.

- `derived_property_list` : A list of element property definitions.

- `derived_property` : An expression that defines a property and can optionally reference the input table columns.

  In the following example, the `id` and `name` columns are included as properties. Additionally, the result of the `salary + bonus` expression are included as the `income` property:

  ```
  PROPERTIES (id, name, salary + bonus AS income)
  ```

  A derived property includes:

  - `value_expression` : An expression that can be represented by simple constructs such as column references and functions. Subqueries are excluded.

  - `AS property_name` : Alias to assign to the value expression. This is optional unless `value_expression` is a function.

  If `derived_property` has any column reference in `value_expression` , that column reference must refer to a column of the underlying table.

  If `derived_property` doesn't define `property_name` , `value_expression` must be a column reference and the implicit `property_name` is the column name.

- `description_string` : A string literal that describes the property to provide context and improve discoverability for natural language querying interfaces.

- `synonym_array` : An array of string literals that provides alternative names for the property to improve accuracy for natural language querying interfaces.

### Dynamic label definition

```
dynamic_label:
  DYNAMIC LABEL (dynamic_label_column_name)
```

**Description**

Specifies a column that holds dynamic label values.

**Definitions**

- `dynamic_label_column_name` : The name of the column that holds label values. The column must use the STRING data type.

  - As a graph element is mapped from a row of an element table, an element's dynamic label is the data that resides in the `dynamic_label_column_name` column.

  - There can be at most one node table and one edge table within a schema that supports dynamic labels.

  - The dynamic label values must be stored in lower-case. When you access them in queries, they are case-insensitive.

  - Both defined labels and a dynamic label can be applied to an element. If the names of a [defined label](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-schema-statements#label_property_definition) and dynamic label overlap, the defined label takes precedence over the dynamic one.

### Dynamic properties definition

```
dynamic_properties:
  DYNAMIC PROPERTIES (dynamic_properties_column_name)
```

**Description**

Specifies a column that holds dynamic properties values.

**Definitions**

- `dynamic_properties_column_name` : The name of the column that holds properties values. The column must be of JSON type.

  - As a graph element is mapped from a row of an element table, an element's dynamic properties are the data that resides in the `dynamic_properties_column_name` column.

  - Top-level JSON keys in the `dynamic_properties_column_name` column are mapped as dynamic properties.

  - The JSON key of each dynamic property must be stored in lower-case. When you access them in queries, they are case-insensitive.

  - Unlike dynamic labels, any number of nodes or edges within a schema can support dynamic properties.

  - Unlike the [Element properties definition](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-schema-statements#element_table_property_definition) , dynamic properties for an element are not exposed by a dynamic label and can evolve independently.

  - If the names of a defined property and dynamic property overlap, the defined property takes precedence over the dynamic one.

### `FinGraph` example

The following property graph, `FinGraph` , contains two node definitions ( `Account` and `Person` ) and two edge definitions ( `PersonOwnAccount` and `AccountTransferAccount` ).

> **Note:** all GQL examples in the GQL reference use the `FinGraph` property graph example. To set up this property graph, see [Set up and query Spanner Graph](https://cloud.google.com/spanner/docs/graph/set-up) .

```
CREATE OR REPLACE PROPERTY GRAPH FinGraph
  NODE TABLES (
    Account,
    Person
  )
  EDGE TABLES (
    PersonOwnAccount
      SOURCE KEY (id) REFERENCES Person (id)
      DESTINATION KEY (account_id) REFERENCES Account (id)
      LABEL Owns,
    AccountTransferAccount
      SOURCE KEY (id) REFERENCES Account (id)
      DESTINATION KEY (to_id) REFERENCES Account (id)
      LABEL Transfers
  );
```

Once the property graph is created, you can use it in [GQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-query-statements) queries. For example, the following query matches all nodes labeled `Person` and then returns the `name` values in the results.

```
GRAPH FinGraph
MATCH (p:Person)
RETURN p.name

/*---------+
 | name    |
 +---------+
 | Alex    |
 | Dana    |
 | Lee     |
 +---------*/
```

#### `FinGraph` with dynamic label and dynamic properties

The following property graph, `FinGraph` , contains a unified node and unified edge definition with dynamic label and dynamic properties to store all nodes and edges.

```
CREATE PROPERTY GRAPH FinGraph
  NODE TABLES (
    GraphNode
      DYNAMIC LABEL (label)
      DYNAMIC PROPERTIES (properties)
)
  EDGE TABLES (
    GraphEdge
      SOURCE KEY (id) REFERENCES GraphNode(id)
      DESTINATION KEY (dest_id) REFERENCES GraphNode(id)
      DYNAMIC LABEL (label)
      DYNAMIC PROPERTIES (properties)
);
```

Compared to the previous example, to add `Account` and `Person` nodes in a dynamic label model, insert entries into `GraphNode` with the label as `Account` or `Person` to indicate which node type that entry specifies. Dynamic properties must be added as JSON.

```
INSERT INTO GraphNode (id, label, properties)
VALUES (1, "person", JSON '{"name": "Alex", "age": 33}');
```

Similarly, inserting entries to `GraphEdge` with values like `PersonOwnAccount` and `AccountTransferAccount` for the `label` column creates edges.

## `DROP PROPERTY GRAPH` statement

```
DROP PROPERTY GRAPH [ IF EXISTS ] property_graph_name;
```

**Description**

Deletes a property graph.

**Definitions**

- `IF EXISTS` : If a property graph of the specified name doesn't exist, then the DROP statement has no effect and no error is generated.
- `property_graph_name` : The name of the property graph to drop.

**Example**

```
DROP PROPERTY GRAPH FinGraph;
```

## `INFORMATION SCHEMA`

Use the SQL [`INFORMATION_SCHEMA`](https://docs.cloud.google.com/spanner/docs/information-schema#property-graphs) to look up schemas created by the [`CREATE PROPERTY GRAPH` statement](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-schema-statements#gql_create_graph) .
