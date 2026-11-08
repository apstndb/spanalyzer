---
name: documents/docs.cloud.google.com/spanner/docs/foreign-keys/overview
uri: https://docs.cloud.google.com/spanner/docs/foreign-keys/overview
title: Foreign keys
description: Understand how foreign keys enforce referential integrity between tables in your Spanner database, ensuring data consistency.
data_source: docs.cloud.google.com
---

This document describes foreign keys in Spanner, and how you can use them to enforce referential integrity in your database. The following topics help you learn about foreign keys and how to use them:

- [Overview of foreign keys in Spanner](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#foreign-key-overview)
- [Types of foreign keys](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#types-of-foreign-keys)
- [Comparison of foreign key types](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#comparison-of-foreign-keys-types)
- [Choose which foreign key type to use](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#choose-foreign-key-type)
- [Use enforced foreign keys](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#use-enforced-foreign-keys)
- [Use informational foreign keys](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#use-informational-foreign-keys)
- [Backing indexes](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#backing-indexes)
- [Long-running schema changes](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#long-running-schema-changes)

## Overview of foreign keys in Spanner

Foreign keys define relationships between tables. You can use foreign keys to make sure that the data integrity of these relationships in Spanner is maintained.

Imagine you're a lead developer for an ecommerce business. You are designing a database to process customer orders. The database must store information about each order, customer, and product. Figure 1 illustrates the basic database structure for the application.

![Basic structure of the order processing database.](https://docs.cloud.google.com/static/spanner/docs/images/foreign-keys-order-processing-tables.svg)

**Figure 1.** Diagram of an order processing database

You define a `Customers` table to store customer information, an `Orders` table to track all orders, and a `Products` table to store information about each product.

Figure 1 also shows links between the tables that map to the following real-world relationships:

- A customer places an order.

- An order is placed for a product.

You decide that your database enforces the following rules to ensure that orders in your system are valid.

- You can't create an order for a customer that doesn't exist.

- A customer can't place an order for a product you don't carry.

When you enforce these rules, or *constraints* , you're maintaining the *referential integrity* of your data. When a database maintains referential integrity, all attempts to add invalid data, which would result in invalid links or references between data, fail. Referential integrity prevents user errors. By default, Spanner uses foreign keys to enforce referential integrity.

### Define referential integrity with foreign keys

The following examines the order processing example again, with more detail added to the design, as shown in Figure 2.

![Database schema with foreign keys](https://docs.cloud.google.com/static/spanner/docs/images/foreign-keys-example.svg)

**Figure 2.** Diagram of a database schema with foreign keys

The design now shows column names and types in each table. The `Orders` table also defines two foreign key relationships. `FK_CustomerOrder` expects that all rows in `Orders` have a valid `CustomerId` . The `FK_ProductOrder` foreign key expects that all `ProductId` values in the `Orders` table are valid. The following table maps these constraints back to the real-world rules that you want to enforce.

| Foreign Key Name | Constraint                                                  | Real-world description                  |
|------------------|-------------------------------------------------------------|-----------------------------------------|
| FK_CustomerOrder | Expects that all rows in `Orders` have a valid `CustomerId` | A valid customer places an order        |
| FK_ProductOrder  | Expects that all rows in `Orders` have a valid `ProductId`  | An order was placed for a valid product |

Spanner enforces constraints that are specified using [enforced foreign keys](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#enforced-foreign-keys) . This means that Spanner fails any transaction that attempts to insert or update a row in the `Orders` table that has a `CustomerId` or `ProductId` not found in the `Customers` and `Products` tables. It also fails transactions that attempt to update or delete rows in the `Customers` and `Products` tables that would invalidate the IDs in the `Orders` table. For more details about how Spanner validates constraints, refer to the [Transaction constraint validation](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#constraint_validation) section.

Unlike enforced foreign keys, Spanner doesn't validate constraints on [informational foreign keys](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#informational-foreign-keys) . This means that if you use an informational foreign key in this scenario, then a transaction that attempts to insert or update a row in the `Orders` table that has a `CustomerId` or `ProductId` not found in the `Customers` and `Products` tables isn't validated and the transaction doesn't fail. Also unlike enforced foreign keys, informational foreign keys are supported by GoogleSQL only, and not by PostgreSQL.

### Foreign key characteristics

The following is a list of characteristics of foreign keys in Spanner.

- The table that defines the foreign key is the *referencing* table, and the foreign key columns are the *referencing* columns.

- The foreign key references the *referenced* columns of the *referenced* table.

- As in the example, you can name each foreign key constraint. If you don't specify a name, Spanner generates a name for you. You can query the generated name from Spanner's [`INFORMATION_SCHEMA`](https://docs.cloud.google.com/spanner/docs/information-schema#referential-constraints) . Constraint names are scoped to the schema, along with the names for tables and indexes, and must be unique within the schema.

- The number of referencing and referenced columns must be the same. Order is important. For example, the first referencing column refers to the first referenced column and the second referencing column refers to the second referenced column.

- A referencing column and its referenced counterpart must be the same type. You must be able to index the columns.

- You can't create foreign keys on columns with the `allow_commit_timestamp=true` option.

- Array columns are not supported.

- JSON columns are not supported.

- A foreign key can reference columns of the same table (a *self-referencing* foreign key). An example is an `Employee` table with a `ManagerId` column that references the table's `EmployeeId` column.

- Foreign keys can also form circular relationships between tables where two tables reference each other, either directly or indirectly. The referenced table must exist before creating a foreign key. This means that at least one of the foreign keys must be added using the `ALTER TABLE` statement.

- The referenced keys must be unique. Spanner uses the `PRIMARY KEY` of the referenced table if the referenced columns for a foreign key match the referenced table's primary key columns. If Spanner can't use the referenced table's primary key, it creates a `UNIQUE NULL_FILTERED INDEX` over the referenced columns.

- Foreign keys don't use secondary indexes that you have created. Instead, they create their own backing indexes. Backing indexes are usable in query evaluations, including in explicit `force_index` directives. You can query the names of the backing indexes from Spanner's [`INFORMATION_SCHEMA`](https://docs.cloud.google.com/spanner/docs/information-schema#indexes) . For more information, see [Backing indexes](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#backing-indexes) .

## Types of foreign keys

There are two types of foreign keys, *enforced* and *informational* . Enforced foreign keys are the default and enforce referential integrity. Informational foreign keys don't enforce referential integrity and are best used to declare the intended logical data model for query optimization. For more details, see the following [enforced](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#enforced-foreign-keys) and [informational](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#informational-foreign-keys) foreign keys sections and the [comparison of foreign key types](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#comparison-of-foreign-keys-types) table.

### Enforced foreign keys

*Enforced* foreign keys, the default foreign key type in Spanner, enforce referential integrity. Because enforced foreign keys enforce referential integrity, they cause attempts to do the following to fail:

- Adding a row to a referencing table that has a foreign key value that doesn't exist in the referenced table fails.

- Deleting a row from a referenced table that's referenced by rows in the referencing table fails.

All PostgreSQL foreign keys are enforced. GoogleSQL foreign keys are enforced by default. Because foreign keys are enforced by default, using the `ENFORCED` keyword to specify that a GoogleSQL foreign key is enforced is optional.

### Informational foreign keys

*Informational* foreign keys are used to declare the intended logical data model for [query optimization](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#informational-fk-query-optimization) . While referenced table keys must be unique for informational foreign keys, the referential integrity isn't enforced. If you want to selectively validate referential integrity when you use informational foreign keys, then you need to manage validation logic on the client side. For more information, see [Use informational foreign keys](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#use-informational-foreign-keys) .

Use the `NOT ENFORCED` keyword to specify that a GoogleSQL foreign key is informational. PostgreSQL doesn't support informational foreign keys.

## Comparison of foreign key types

Both [enforced](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#enforced-foreign-keys) and [informational](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#informational-foreign-keys) have benefits. The following sections compare the two types of foreign keys and include some best practices.

### High-level foreign key differences

At a high level, the following are some of the differences between *enforced* and *informational* foreign keys:

- **Enforcement** . Enforced foreign keys validate and guarantee referential integrity on writes. Informational foreign keys don't validate or guarantee referential integrity.

- **Storage** . Enforced foreign keys might require additional storage for the backing index on the constrained table.

- **Write throughput** . Enforced foreign keys might incur more overhead in the write path than informational foreign keys.

- **Query optimization** . Both types of foreign keys can be used for query optimization. When the [optimizer is allowed to use informational foreign keys](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#informational-fk-query-optimization) , query results might not reflect the actual data if the data doesn't match the informational foreign key relationships (for example, if some constrained keys don't have matching referenced keys in the referenced table).

### Foreign key differences table

The following table lists detailed differences between enforced and informational foreign keys:

|                                                                                                                                                      | Enforced foreign keys                                                | Informational foreign keys                                                                                                                                                                                                                                                                                                                                 |
|------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Keywords                                                                                                                                             | `ENFORCED`                                                           | `NOT ENFORCED`                                                                                                                                                                                                                                                                                                                                             |
| Supported by GoogleSQL                                                                                                                               | Yes. Foreign keys in GoogleSQL are enforced by default.              | Yes.                                                                                                                                                                                                                                                                                                                                                       |
| Supported by PostgreSQL                                                                                                                              | Yes. Foreign keys in PostgreSQL can only be enforced.                | No.                                                                                                                                                                                                                                                                                                                                                        |
| Storage                                                                                                                                              | Enforced foreign keys require storage for up to two backing indexes. | Informational foreign keys require storage for up to one backing index.                                                                                                                                                                                                                                                                                    |
| Creates [backing indexes](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#backing-indexes) on referenced table columns when needed  | Yes.                                                                 | Yes.                                                                                                                                                                                                                                                                                                                                                       |
| Creates [backing indexes](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#backing-indexes) on referencing table columns when needed | Yes.                                                                 | No.                                                                                                                                                                                                                                                                                                                                                        |
| [Foreign key actions](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#how-to-define-foreign-key-action) support                     | Yes.                                                                 | No.                                                                                                                                                                                                                                                                                                                                                        |
| [Validates and enforces referential integrity](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#constraint_validation)               | Yes.                                                                 | No. Having no validation improves write performance, but can impact query results when [informational foreign keys are used for query optimization](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#informational-fk-query-optimization) . You can use client-side validation or an enforced foreign key to ensure referential integrity. |

## Choose which foreign key type to use

You can use the following guidelines to decide which foreign key type to use:

We recommend that you start with enforced foreign keys. Enforced foreign keys keep the data and the logical model consistent at all times. Enforced foreign keys are the recommended option unless they don't work for your use case.

We recommend that you consider informational foreign keys if each of the following is true:

- You want to use the logical data model described by informational foreign key in query optimization.

- Maintaining strict referential integrity is impractical or impacts performance significantly. The following are examples of when you might want to consider using an informational foreign key:

  - Your upstream data source follows an eventual-consistency model. In this case, updates made in the source system might not be reflected immediately in Spanner. Because updates might not be immediate, brief inconsistencies in foreign key relationships might occur.

  - Your data contains referenced rows that have a large number of referencing relationships. Updates to these rows can use a lot of resources because Spanner must validate or, in some cases, delete all rows that are related to maintaining referential integrity. In this scenario, updates might impact Spanner performance and slow down concurrent transactions.

- Your application can handle potential data inconsistencies and their impact on query results.

## Use informational foreign keys

The following topics are for informational foreign keys only. For topics that apply to both informational and enforced foreign keys, see the following:

- [Long-running schema changes](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#long-running-schema-changes)
- [Backing indexes](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#backing-indexes)

### Create a new table with an informational foreign key

You create and remove and [informational foreign keys](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#informational-foreign-keys) from your Spanner database using DDL statements. You add foreign keys to a new table with the [`CREATE TABLE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create_table) statement. Similarly, you can add or remove foreign keys from an existing table with the [`ALTER TABLE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter_table) statement.

The following example creates a new table with an informational foreign key using GoogleSQL. Informational foreign keys aren't supported by PostgreSQL.

### GoogleSQL

```
CREATE TABLE Customers (
  CustomerId INT64 NOT NULL,
  CustomerName STRING(MAX) NOT NULL,
) PRIMARY KEY(CustomerId);

CREATE TABLE Orders (
  OrderId INT64 NOT NULL,
  CustomerId INT64 NOT NULL,
  Quantity INT64 NOT NULL,
  ProductId INT64 NOT NULL,
  CONSTRAINT FK_CustomerOrder FOREIGN KEY (CustomerId)
   REFERENCES Customers (CustomerId) NOT ENFORCED
 ) PRIMARY KEY (OrderId);
```

### PostgreSQL

```
Not Supported
```

For more examples of how to create and manage foreign keys, see [Create and manage foreign key relationships](https://docs.cloud.google.com/spanner/docs/foreign-keys/how-to) . For more information about DDL statements, see the [DDL reference](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter_table) .

> **Note:** You can also add and remove foreign keys using the Google Cloud console. Select **edit as text** from the table creation interface, and enter the DDL to manage foreign keys for your table.

### Use informational foreign keys for query optimization

Both [enforced foreign keys](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#enforced-foreign-keys) and [informational foreign keys](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#informational-foreign-keys) can be used by the query optimizer to improve query performance. Using informational foreign keys lets you to take advantage of optimized query plans without the overhead of strict referential integrity enforcement.

If you enable the query optimizer to utilize informational foreign keys information, it's important to understand that the correctness of the optimization depends on having data that's consistent with the logical model described by the informational foreign keys. If inconsistencies exist, then query results might not reflect the actual data. An example of an inconsistency is when a value in constrained column doesn't have a matching value in a referenced column.

By default, the query optimizer uses `NOT ENFORCED` foreign keys. To change this, set the database option `use_unenforced_foreign_key_for_query_optimization` to false. The following is a GoogleSQL example that demonstrates this (informational foreign keys aren't available in PostgreSQL):

```
SET DATABASE OPTIONS (
    use_unenforced_foreign_key_for_query_optimization = false
);
```

The boolean query statement hint `@{use_unenforced_foreign_key}` overrides the database option on a per-query basis that controls whether the optimizer uses `NOT ENFORCED` foreign keys. Disabling this hint or the database option can be useful when troubleshooting unexpected query results. The following shows how to use `@{use_unenforced_foreign_key}` :

```
@{use_unenforced_foreign_key=false} SELECT Orders.CustomerId
    FROM Orders
    INNER JOIN Customers ON Customers.CustomerId = Orders.CustomerId;
```

## Use enforced foreign keys

The following topics are for enforced foreign keys only. For topics that apply to both informational and enforced foreign keys, see the following:

- [Long-running schema changes](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#long-running-schema-changes)
- [Backing indexes](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#backing-indexes)

### Create a new table with an enforced foreign key

You create and remove and [enforced](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#enforced-foreign-keys) foreign keys from your Spanner database using DDL. You add foreign keys to a new table with the [`CREATE TABLE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create_table) statement. Similarly, you add a foreign key to, or remove a foreign key from, an existing table with the [`ALTER TABLE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter_table) statement.

You create and remove foreign keys from your Spanner database using DDL. You add foreign keys to a new table with the [`CREATE TABLE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create_table) statement. Similarly, you add a foreign key to, or remove a foreign key from, an existing table with the [`ALTER TABLE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter_table) statement.

The following is an example of creating a new table with an enforced foreign key.

### GoogleSQL

```
CREATE TABLE Customers (
CustomerId INT64 NOT NULL,
CustomerName STRING(MAX) NOT NULL,
) PRIMARY KEY(CustomerId);

CREATE TABLE Orders (
OrderId INT64 NOT NULL,
CustomerId INT64 NOT NULL,
Quantity INT64 NOT NULL,
ProductId INT64 NOT NULL,
CONSTRAINT FK_CustomerOrder FOREIGN KEY (CustomerId)
  REFERENCES Customers (CustomerId) ENFORCED
) PRIMARY KEY (OrderId);
```

### PostgreSQL

```
CREATE TABLE Customers (
CustomerId bigint NOT NULL,
CustomerName character varying(1024) NOT NULL,
PRIMARY KEY(CustomerId)
);

CREATE TABLE Orders (
OrderId BIGINT NOT NULL,
CustomerId BIGINT NOT NULL,
Quantity BIGINT NOT NULL,
ProductId BIGINT NOT NULL,
CONSTRAINT FK_CustomerOrder FOREIGN KEY (CustomerId)
  REFERENCES Customers (CustomerId),
PRIMARY KEY (OrderId)
);
```

For more examples of how to create and manage foreign keys, see [Create and manage foreign key relationships](https://docs.cloud.google.com/spanner/docs/foreign-keys/how-to) .

> **Note:** You can also add and remove foreign keys using the Google Cloud console. Select **edit as text** from the table creation interface, and enter the DDL to manage foreign keys for your table.

### Foreign key actions

Foreign key actions can be defined on enforced foreign keys only.

Foreign key actions control what happens to the constrained column when the column it references is deleted or updated. Spanner supports the use of the `ON DELETE CASCADE` action. With the foreign key `ON DELETE CASCADE` action, when you delete a row that contains a referenced foreign key, all rows that reference that key are also deleted in the same transaction.

You can add a foreign key with an action when you create your database using DDL. Use the [`CREATE TABLE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create_table) statement to add foreign keys with an action to a new table. Similarly, you can use the [`ALTER TABLE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter_table) statement to add a foreign key action to an existing table or to remove a foreign key action. The following is an example of how to create a new table with a foreign key action.

### GoogleSQL

```
CREATE TABLE ShoppingCarts (
CartId INT64 NOT NULL,
CustomerId INT64 NOT NULL,
CustomerName STRING(MAX) NOT NULL,
CONSTRAINT FKShoppingCartsCustomers FOREIGN KEY(CustomerId, CustomerName)
  REFERENCES Customers(CustomerId, CustomerName) ON DELETE CASCADE,
) PRIMARY KEY(CartId);
```

### PostgreSQL

```
CREATE TABLE ShoppingCarts (
CartId bigint NOT NULL,
CustomerId bigint NOT NULL,
CustomerName character varying(1024) NOT NULL,
PRIMARY KEY(CartId),
CONSTRAINT fkshoppingcartscustomers FOREIGN KEY (CustomerId, CustomerName)
  REFERENCES Customers(CustomerId, CustomerName) ON DELETE CASCADE
);
```

The following is a list of characteristics of foreign key actions in Spanner.

- Foreign key actions are either `ON DELETE CASCADE` or `ON DELETE NO ACTION` .

- You can query the [`INFORMATION_SCHEMA`](https://docs.cloud.google.com/spanner/docs/information-schema#referential-constraints) to find foreign key constraints that have an action.

- Adding a foreign key action on an existing foreign key constraint isn't supported. You must add a new foreign key constraint with an action.

### Constraint validation

Constraint validation applies to enforced foreign keys only.

Spanner validates enforced foreign key constraints as a transaction is committed, or as the effects of writes are made visible to subsequent operations in the transaction.

A value inserted into the referencing column is matched against the values of the referenced table and referenced columns. Rows with `NULL` referencing values aren't checked, which means that you can add them to the referencing table.

Spanner validates all applicable enforced foreign key referential constraints when attempting to update data using either DML statements or an API. All pending changes are rolled back if any constraints are invalid.

Validation occurs immediately after each DML statement. For example, you must insert the referenced row before inserting its referencing rows. When using a mutation API, mutations are buffered until the transaction is committed. Enforced foreign key validation is deferred until the transaction is committed. In this case, it is permissible to insert the referencing rows first.

Each transaction is evaluated for modifications that affect enforced foreign key constraints. These evaluations might require additional requests to the server. Backing indexes also require additional processing time to evaluate transaction modifications and to maintain the indexes. Additional storage is also required for each index.

### Long-running delete cascade action

When you delete a row from a referenced table, Spanner must delete all rows in the referencing tables that reference the deleted row. This can lead to a cascading effect, where a single delete operation results in thousands of other delete operations. Adding a foreign key constraint with delete cascade action to a table or creating a table with foreign key constraints with delete cascade action can slow down delete operations.

### Mutation limit exceeded for foreign key delete cascade

Deleting a large number of records using a foreign key delete cascade can impact performance. This is because each deleted record invokes the deletion of all records related to it. If you need to delete a large number of records using a foreign key delete cascade, explicitly delete the rows from the child tables before deleting the row from the parent tables. This prevents the transaction from failing due to the mutation limit.

### Comparison of enforced foreign keys and table interleaving

Spanner's table interleaving is a good choice for many parent-child relationships where the child table's primary key includes the parent table's primary key columns. The co-location of child rows with their parent rows can significantly improve performance.

Foreign keys are a more general parent-child solution and address additional use cases. They're not limited to primary key columns, and tables can have multiple foreign key relationships, both as a parent in some relationships and a child in others. However, a foreign key relation does not imply co-location of the tables in the storage layer.

Consider an example that uses an `Orders` table that's defined as follows:

![Database schema with foreign keys](https://docs.cloud.google.com/static/spanner/docs/images/foreign-keys-example.svg)

**Figure 3.** Diagram of the database schema with enforced foreign keys

The design in Figure 3 has some limitations. For example, each order can contain only one order item.

Imagine that your customers want to be able to order more than one product per order. You can enhance your design by introducing an `OrderItems` table that contains an entry for each product the customer ordered. You can introduce another enforced foreign key to represent this new one-to-many relationship between `Orders` and `OrderItems` . However, you also know that you often want to run queries across orders and their respective order items. Because co-location of this data boosts performance, you would want to create the parent-child relationship using Spanner's table interleaving capability.

Here's how you define the `OrderItems` table, interleaved with `Orders` .

### GoogleSQL

```
CREATE TABLE Products (
ProductId INT64 NOT NULL,
Name STRING(256) NOT NULL,
Price FLOAT64
) PRIMARY KEY(ProductId);

CREATE TABLE OrderItems (
OrderId INT64 NOT NULL,
ProductId INT64 NOT NULL,
Quantity INT64 NOT NULL,
FOREIGN KEY (ProductId) REFERENCES Products (ProductId)
) PRIMARY KEY (OrderId, ProductId),
INTERLEAVE IN PARENT Orders ON DELETE CASCADE;
```

### PostgreSQL

```
CREATE TABLE Products (
ProductId BIGINT NOT NULL,
Name varchar(256) NOT NULL,
Price float8,
PRIMARY KEY(ProductId)
);

CREATE TABLE OrderItems (
OrderId BIGINT NOT NULL,
ProductId BIGINT NOT NULL,
Quantity BIGINT NOT NULL,
FOREIGN KEY (ProductId) REFERENCES Products (ProductId),
PRIMARY KEY (OrderId, ProductId)
) INTERLEAVE IN PARENT Orders ON DELETE CASCADE;
```

Figure 4 is a visual representation of the updated database schema as a result of introducing this new table, `OrderItems` , interleaved with `Orders` . Here you can also see the one-to-many relationship between those two tables.

![Database schema showing a one-to-many relationship between Orders and the new, interleaved, OrderItems table](https://docs.cloud.google.com/static/spanner/docs/images/foreign-keys-schema-with-interleaving.svg)

**Figure 4.** Addition of an interleaved OrderItems table

In this configuration, you can have multiple `OrderItems` entries in each order, and the `OrderItems` entries for each order are interleaved, and therefore co-located with the orders. Physically interleaving `Orders` and `OrderItems` in this way can improve performance, effectively pre-joining the tables and letting you access related rows together while minimizing disk accesses. For example, Spanner can perform joins by primary key locally, minimizing disk access and network traffic.

If the number of mutations in a transaction exceeds [80,000](https://docs.cloud.google.com/spanner/quotas#query-limits) , the transaction fails. Such large cascading deletes work well for tables with an "interleaved in parent" relationship, but not for tables with a foreign key relationship. If you have a foreign key relationship and you need to delete a large number of rows, you should explicitly delete the rows from the child tables first.

If you have a user table with a foreign key relationship to another table, and deleting a row from the referenced table triggers the deletion of millions of rows, you should design your schema with a delete cascade action with "interleaved in parent".

> **Note:** Google recommends that you choose to represent parent-child relationships either as interleaved tables or as foreign keys, but not both. Using both is redundant and might consume additional storage and computing resources in order to verify constraints and maintain [backing indexes](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#backing-indexes) .

#### Comparison table

The following table summarizes how enforced foreign keys and table interleaving compare. You can use this information to decide what is right for your design.

| Parent-child relationship type        | Table Interleaving                                                                                                                                                          | Enforced foreign keys                                                                                                                                                                                         |
|---------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Can use primary keys                  | Yes                                                                                                                                                                         | Yes                                                                                                                                                                                                           |
| Can use non-primary-key columns       | No                                                                                                                                                                          | Yes                                                                                                                                                                                                           |
| Number of parents supported           | 0 .. 1                                                                                                                                                                      | 0 .. N                                                                                                                                                                                                        |
| Stores parent and child data together | Yes                                                                                                                                                                         | No                                                                                                                                                                                                            |
| Supports cascade delete               | Yes                                                                                                                                                                         | Yes                                                                                                                                                                                                           |
| Null matching mode                    | Passes if all referencing values are not distinct from the referenced values. Null values are not distinct from null values; null values are distinct from non-null values. | Passes if any referencing values are null. Passes if all referencing values are non-null, and the referenced table has a row with values equal to the referencing values. Fails if no matching row was found. |
| Enforcement Timing                    | Per operation when using the mutation API. Per statement when using DML.                                                                                                    | Per transaction when using the mutation API. Per statement when using DML.                                                                                                                                    |
| Can be removed                        | No. Table interleaving can't be removed after it's created, unless you delete the whole child table.                                                                        | Yes                                                                                                                                                                                                           |

## Backing indexes

Foreign keys don't use user-created indexes. Instead, they create their own backing indexes. [Enforced](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#enforced-foreign-keys) and [informational](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#informational-foreign-keys) foreign keys create backing indexes differently in Spanner:

- For enforced foreign keys, Spanner can create up to two secondary backing indexes for each foreign key, one for the referencing columns, and a second for the referenced columns.

- For informational foreign keys, Spanner can create up to one backing index when needed for the referenced columns. Informational foreign keys don't create a backing index for the referencing columns.

For both enforced and informational foreign keys, a foreign key usually references the primary keys of the referenced table, so an index for the referenced table is typically not needed. Because of this, informational foreign keys typically have zero backing indexes. When needed, the backing index created for the *referenced* table is a `UNIQUE NULL_FILTERED` index. The creation of the foreign key fails if any existing data violates the index's uniqueness constraint.

Informational foreign keys don't have a backing index for the *referencing* table. For enforced foreign keys, the backing index for the referencing table is `NULL_FILTERED` .

If two or more foreign keys require the same backing index, Spanner creates a single index for each of them. The backing indexes are dropped when the foreign keys using them are dropped. You can't alter or drop the backing indexes.

Spanner uses the [information schema](https://docs.cloud.google.com/spanner/docs/information-schema) of each database to store metadata about backing indexes. Rows within [`INFORMATION_SCHEMA.INDEXES`](https://docs.cloud.google.com/spanner/docs/information-schema#indexes) that have a `SPANNER_IS_MANAGED` value of `true` describe backing indexes.

Outside of SQL queries that directly invoke the information schema, the Google Cloud console doesn't display any information about a database's backing indexes.

## Long-running schema changes

Adding an enforced foreign key to an existing table, or creating a new table with a foreign key, can lead to long-running operations. In the case of a new table, the table isn't writable until the long-running operation is complete.

The following table shows what happens in Spanner when an enforced and an informational foreign key is in a new or an existing table:

| Table type | Enforced foreign key                                                                                                                                                                                                                                           | Informational foreign key                                                                         |
|------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------|
| New        | Spanner backfills referenced indexes as needed for each foreign key.                                                                                                                                                                                           | Spanner backfills referenced indexes as needed for each foreign key.                              |
| Existing   | Spanner backfills the referencing and referenced indexes as needed. Spanner also validates existing data in the table to ensure that it complies with the referential integrity constraint of the foreign key. The schema change fails if any data is invalid. | Spanner backfills the referenced index as needed and doesn't validate existing data in the table. |

The following aren't supported:

- Adding a foreign key action to an existing enforced foreign key constraint.
- Changing the enforcement of an existing foreign key.

For both cases, we recommend that you instead do the following:

1.  Add a new constraint with the required action or enforcement.
2.  Drop the old constraint.

Adding a new constraint and dropping the old constraint prevents a *Long-running Alter Constraint Operation* issue. For example, suppose you want to add a `DELETE CASCADE` action on an existing foreign key. After you create the new foreign key with the `ON DELETE CASCADE` action, the effect of both constraints is a `DELETE CASCADE` action. Then you can drop the old constraint safely.

Dropping a constraint can lead to dropping the foreign key backing indexes if the indexes aren't used by other foreign key constraints. Because of this, if you drop the old constraint first, adding the same foreign key constraint with an action later might lead to long-running operations, such as backfilling indexes, validating unique index constraints, or validating foreign key referential constraints.

> **Note:** If you don't drop the older foreign key constraint, then extra foreign key constraints need validation which causes unnecessary overhead. Also, if a `UNIQUE` constraint violation prevents the creation of the referenced index, then adding the new constraint might fail.

You can query [`INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS.SPANNER_STATE`](https://docs.cloud.google.com/spanner/docs/information-schema#referential-constraints) to check foreign key creation state.

## What's next

- Learn about [Creating and managing foreign key relationships](https://docs.cloud.google.com/spanner/docs/foreign-keys/how-to) .

- Learn more about the [information schema](https://docs.cloud.google.com/spanner/docs/information-schema) .
