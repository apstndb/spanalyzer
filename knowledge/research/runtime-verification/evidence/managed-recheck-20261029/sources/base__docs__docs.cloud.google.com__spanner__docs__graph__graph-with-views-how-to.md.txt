---
name: documents/docs.cloud.google.com/spanner/docs/graph/graph-with-views-how-to
uri: https://docs.cloud.google.com/spanner/docs/graph/graph-with-views-how-to
title: Create a graph from a SQL view
description: A managed, mission-critical, globally consistent and scalable relational database service.
data_source: docs.cloud.google.com
---

> **Note:** This feature is available with the Spanner Enterprise edition and Enterprise Plus edition. For more information, see the [Spanner editions overview](https://docs.cloud.google.com/spanner/docs/editions-overview) .

Learn how to create a graph using [SQL views](https://docs.cloud.google.com/spanner/docs/views) . This document provides step-by-step instructions and code examples for defining views and using them to define node and edge tables. Explore [examples](https://docs.cloud.google.com/spanner/docs/graph/graph-with-views-how-to#use-cases-graphs-views) with sample code that demonstrate use cases for creating graphs with views. To learn more about using views to create a property graph, including benefits and considerations, see [Overview of graphs created from SQL views](https://docs.cloud.google.com/spanner/docs/graph/graph-with-views-overview) .

## Before you begin

To create a graph, you must:

1.  Ensure your Spanner Graph environment is [set up](https://docs.cloud.google.com/spanner/docs/graph/set-up#before-you-begin) .

2.  Familiarize yourself with [how Spanner Graph schemas work](https://docs.cloud.google.com/spanner/docs/graph/schema-overview) .

## Create a graph using views

To create a graph using views:

1.  Define views for your graph. By default, your views must follow one of the [supported view patterns](https://docs.cloud.google.com/spanner/docs/graph/graph-with-views-overview#graph-view-requirements) to ensure element uniqueness. If your views use queries that don't follow these patterns, you can disable key uniqueness validation in step 4. For more information, see [Create a view](https://docs.cloud.google.com/spanner/docs/create-manage-views#create_a_view) .

2.  Use your views in the `NODE TABLES` and `EDGE TABLES` clauses of the [`CREATE PROPERTY GRAPH`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/graph-schema-statements#property_graph_definition) statement to create a graph.

3.  Include the `KEY` clause in the `CREATE PROPERTY GRAPH` statement. The `KEY` clause specifies the columns from the source view that uniquely identify each graph element.

4.  Optional: If your views use arbitrary SQL queries that don't follow the supported uniqueness patterns, add the `OPTIONS (validate_element_key_uniqueness = false)` clause to the `CREATE PROPERTY GRAPH` statement. Ensure that the `KEY` clause columns produce unique values for every node or edge row:

    ```
    CREATE PROPERTY GRAPH MyGraph
      NODE TABLES (
        CustomerViewTrusted KEY(id)
      ) OPTIONS (validate_element_key_uniqueness = false);
    ```

    For more information, see [Disabled key validation (any SQL query)](https://docs.cloud.google.com/spanner/docs/graph/graph-with-views-overview#disabled-key-validation) .

## Example: Create a graph using views

This example creates the following views over `Customer` and `Account` tables: `AsiaCustomer` , `AsiaBankAccount` , and `AsiaAccountsOwnership` . Then, the example uses these views to create the following in a graph:

- Create the `Customer` node table using the `AsiaCustomer` view.

- Create the `Account` node table using the `AsiaBankAccount` view.

- Create the `Owns` edge table using the `AsiaAccountsOwnership` view. This edge connects `Customer` nodes with `Account` nodes.

### Step 1: Create the tables

First, create the data tables. The following code creates the `Customer` and `Account` tables.

```
CREATE TABLE Customer (
  customer_id INT64 NOT NULL,
  name STRING(MAX),
  address_continent STRING(MAX),
  address_country STRING(MAX),
) PRIMARY KEY(customer_id);

CREATE TABLE Account (
  account_id INT64 NOT NULL,
  customer_id INT64 NOT NULL,
  account_type STRING(MAX),
  balance INT64,
  create_time TIMESTAMP,
  address_continent STRING(MAX),
  address_country STRING(MAX),
  CONSTRAINT FK_CustomerId FOREIGN KEY (customer_id)
    REFERENCES Customer (customer_id)
) PRIMARY KEY(account_id);
```

### Step 2: Create the views

Next, create views to transform or filter data from the tables. These views filter the tables to include only customers and accounts in Asia. Unless `validate_element_key_uniqueness` is set to `false` , views used to create graph elements must [follow supported patterns to ensure that rows in the view are unique](https://docs.cloud.google.com/spanner/docs/graph/graph-with-views-overview#graph-view-requirements) .

```
-- View for 'Customer' nodes, filtered for Asia
CREATE VIEW AsiaCustomer
  SQL SECURITY INVOKER AS
    SELECT customer.customer_id, customer.name
    FROM Customer customer
    WHERE LOWER(customer.address_continent) = "asia";

-- View for 'Account' nodes, filtered for Asia.
CREATE VIEW AsiaBankAccount
  SQL SECURITY INVOKER AS
    SELECT account.account_id, account.balance, account.account_type, account.create_time
    FROM Account account
    WHERE LOWER(account.address_continent) = "asia";

-- View for 'Owns' edges, connecting customers to accounts in Asia.
CREATE VIEW AsiaAccountsOwnership
  SQL SECURITY INVOKER AS
    SELECT account.customer_id, account.account_id
    FROM Account account
    WHERE LOWER(account.address_continent) = "asia";
```

### Step 3: Create the property graph

Now, create the `AsiaFinGraph` using the views you created. The `CREATE PROPERTY GRAPH` statement includes the `KEY` clause for each graph element definition to specify columns that uniquely identify the graph elements.

```
CREATE PROPERTY GRAPH AsiaFinGraph
  NODE TABLES (
    AsiaCustomer AS Customer KEY(customer_id),
    AsiaBankAccount AS Account KEY(account_id)
  )
  EDGE TABLES (
    AsiaAccountsOwnership AS Owns
      KEY(customer_id, account_id)
      SOURCE KEY (customer_id) REFERENCES Customer (customer_id)
      DESTINATION KEY (account_id) REFERENCES Account (account_id)
  );
```

## Use cases examples

SQL views offer benefits over using tables for property graph elements. The following examples demonstrate some use cases for defining graph elements with views instead of tables.

### Example: Enforce fine-grained graph data access control

To enforce row-level security on your graph data, define your node or edge tables using [definer's rights views](https://docs.cloud.google.com/spanner/docs/views#definer) . The view exposes a permitted subset of the underlying data to the graph

For example, to restrict graph access to only employees in an engineering cost center, you can create an `EngineerEmployeeView` view and grant `SELECT` permissions on the view to an `engineering_data_reader` role using the `GRANT` clause.

When you define a graph node table using this view, users running graph queries with the `engineering_data_reader` role can see only the rows filtered by the view, which include engineering employees.

```
-- The table containing all employee data.
CREATE TABLE Employee (
  id INT64 NOT NULL,
  cost_center STRING(MAX),
  job_title STRING(MAX),
  office STRING(MAX)
) PRIMARY KEY (id);

-- The definer's rights view that filters for engineering employees.
CREATE VIEW EngineerEmployeeView SQL SECURITY DEFINER AS
  SELECT e.id, e.cost_center, e.job_title, e.office
  FROM Employee e
  WHERE LOWER(e.cost_center) = "engineering";

-- The role that is granted to read the view.
CREATE ROLE engineering_data_reader;
GRANT SELECT ON VIEW EngineerEmployeeView TO ROLE engineering_data_reader;

-- The graph that uses definer's rights view.
CREATE PROPERTY GRAPH EngineeringGraph
  NODE TABLES (
    EngineerEmployeeView KEY(id)
  );
```

### Example: Model derived graph elements

You can use views to define graph elements that require data transformations. A key benefit is that the view defines the transformation, so you don't need to maintain a separate table for the derived data.

For example, you can `UNNEST` data from an `ARRAY` column (or an array field within a `JSON` column) to model multiple edge relationships from a single row.

In the following supply chain schema example, a `Parts` table stores a list of sub-components in a `dependent_parts` array. A view can use the `UNNEST` operator to transform each element of that array into distinct rows. This view can then serve as an edge table, letting you model a `PartDependsOnPart` edge to represent dependency relationships between parts.

```
-- Parts table with an ARRAY of dependent parts.
CREATE TABLE Parts (
  part_id INT64 NOT NULL,
  dependent_parts ARRAY<INT64>
) PRIMARY KEY (part_id);

-- A view that unnests the dependent_parts array.
-- GROUP BY ensures uniqueness for the graph element KEY.
CREATE VIEW PartDependsOnPart SQL SECURITY INVOKER AS
  SELECT p.part_id, dependent_part_id
  FROM Parts AS p,
    UNNEST(p.dependent_parts) AS dependent_part_id
  GROUP BY p.part_id, dependent_part_id;

-- Graph modeling the part dependency relationship.
CREATE PROPERTY GRAPH SupplyChainGraph
  NODE TABLES (
    Parts
  )
  EDGE TABLES (
    PartDependsOnPart KEY (part_id, dependent_part_id)
      SOURCE KEY (part_id) REFERENCES Parts(part_id)
      DESTINATION KEY (dependent_part_id) REFERENCES Parts(part_id)
  );
```

### Example: Schemaless data transition

[Schemaless data management](https://docs.cloud.google.com/spanner/docs/graph/manage-schemaless-data) lets you create a flexible graph definition without predefined node and edge types. While schemaless data management provides flexibility, you might need to transition to a more formal structure as your data becomes more defined. A more formal structure exposes the graph's node and edge relationships, labels, and properties in the schema, which reduces the need for manual data exploration to understand the graph schema.

You can use views to formalize the node and edge types without migrating your underlying data. For example, you can transition from a typical schemaless model that uses canonical `GraphNode` and `GraphEdge` tables. To do this, you create views that extract the data from your schemaless tables:

1.  Define a view for each node and edge type you want to formalize (for example, `Person` or `WorksFor` ). In the view, filter the data by its label (for example, `WHERE n_label = "person"` ) and cast the properties from the JSON column to specific data types (for example, `STRING(prop.name) AS name` ).

2.  Define a new property graph where `NODE TABLES` and `EDGE TABLES` reference the typed views you just created.

A schemaless graph provides better performance than a formalized graph for some queries (for example, a quantified path pattern with multiple edge types). If formalized metadata is important for your use case, then you can use views to transition from a schemaless graph to a typed schema. You can also choose to use a schemaless graph for some use cases and a typed schema graph for other use cases. For more information, see [Choose a schema design based on graph queries](https://docs.cloud.google.com/spanner/docs/graph/best-practices-designing-schema#choose-design-based-on-queries) .

The following example demonstrates the workflow for transitioning from a schemaless to a formalized graph in four steps:

1.  Define the canonical `GraphNode` and `GraphEdge` tables for the schemaless model.

2.  Create an initial, flexible graph on those schemaless tables.

3.  Define typed views ( `Person` , `Company` , `WorksFor` ) that extract and formalize the data from the schemaless tables.

4.  Create the final, strongly-typed graph that uses these views as its node and edge tables.

```
-- 1. Create the canonical tables for a schemaless model.
CREATE TABLE GraphNode (
  id INT64 NOT NULL,
  label STRING(MAX) NOT NULL,
  properties JSON
) PRIMARY KEY (id);

CREATE TABLE GraphEdge (
  id INT64 NOT NULL,
  dest_id INT64 NOT NULL,
  edge_id INT64 NOT NULL,
  label STRING(MAX) NOT NULL,
  properties JSON
) PRIMARY KEY (id, dest_id, edge_id),
  INTERLEAVE IN PARENT GraphNode;

-- 2. Define a schemaless graph.
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

-- 3. Define typed views that extract and formalize the data.
--    Convert JSON fields to primitive types (for example, INT64, STRING) to
--    ensure type safety.
CREATE VIEW Person SQL SECURITY INVOKER AS
  SELECT n.id, STRING(n.properties.name) AS name, INT64(n.properties.age) AS age
  FROM GraphNode n WHERE n.label = "person";

CREATE VIEW Company SQL SECURITY INVOKER AS
  SELECT n.id, STRING(n.properties.name) AS company_name, BOOL(n.properties.is_public) AS is_public
  FROM GraphNode n WHERE n.label = "company";

CREATE VIEW WorksFor SQL SECURITY INVOKER AS
  SELECT e.id AS person_id, e.dest_id AS company_id, e.edge_id AS edge_id, STRING(e.properties.since) AS since
  FROM GraphEdge e
  WHERE e.label = "worksfor";

-- 4. Create the final, formalized graph from the typed views.
CREATE PROPERTY GRAPH typed_formalized_graph
  NODE TABLES (
    Person KEY(id)
      PROPERTIES (name, age),
    Company KEY(id)
      PROPERTIES (company_name, is_public)
  )
  EDGE TABLES(
    WorksFor KEY(person_id, company_id, edge_id)
      SOURCE KEY (person_id) REFERENCES Person(id)
      DESTINATION KEY (company_id) REFERENCES Company(id)
      PROPERTIES (since)
  );
```

## What's next

- Learn more about [element key uniqueness requirements](https://docs.cloud.google.com/spanner/docs/graph/graph-with-views-overview#graph-view-requirements) .

- Learn about the [Spanner Graph schema](https://docs.cloud.google.com/spanner/docs/graph/schema-overview) .

- Learn about [best practices for designing a Spanner Graph schema](https://docs.cloud.google.com/spanner/docs/graph/best-practices-designing-schema) .
