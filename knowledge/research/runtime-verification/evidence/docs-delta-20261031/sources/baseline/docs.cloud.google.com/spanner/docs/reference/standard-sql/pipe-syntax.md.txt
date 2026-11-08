---
name: documents/docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax
uri: https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax
title: Pipe query syntax in GoogleSQL
description: A managed, mission-critical, globally consistent and scalable relational database service.
data_source: docs.cloud.google.com
---

Pipe query syntax is an extension to GoogleSQL that's simpler and more concise than [standard query syntax](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax) . Pipe syntax supports the same operations as standard syntax, and improves some areas of SQL query functionality and usability.

For more background and details on pipe syntax design, see the research paper [SQL Has Problems. We Can Fix Them: Pipe Syntax In SQL](https://research.google/pubs/sql-has-problems-we-can-fix-them-pipe-syntax-in-sql/) . For an introduction to pipe syntax, see [Work with pipe syntax](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax-guide) .

## Pipe syntax

Pipe syntax has the following key characteristics:

- Each pipe operator in pipe syntax consists of the pipe symbol, `|>` , an operator name, and any arguments:  
  `|> operator_name argument_list`
- Pipe operators can be added to the end of any valid query.
- Pipe syntax works anywhere standard syntax is supported: in queries, views, table-valued functions (TVFs), and other contexts.
- Pipe syntax can be mixed with standard syntax in the same query. For example, subqueries can use different syntax from the parent query.
- A pipe operator can see every alias that exists in the table preceding the pipe.
- A query can [start with a `FROM` clause](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#from_queries) , and pipe operators can optionally be added after the `FROM` clause.

### Query comparison

Consider the following table called `Produce` :

````
CREATE TABLE Produce(
  item STRING(MAX),
  sales INT64,
  category STRING(MAX),
) PRIMARY KEY (item, sales);

INSERT INTO Produce (item, sales, category) VALUES
  ('apples', 2, 'fruit'),
  ('carrots', 8, 'vegetable'),
  ('apples', 7, 'fruit'),
  ('bananas', 5, 'fruit');

SELECT * FROM Produce;

/*---------+-------+-----------+
 | item    | sales | category  |
 +---------+-------+-----------+
 | apples  | 2     | fruit     |
 | apples  | 7     | fruit     |
 | bananas | 5     | fruit     |
 | carrots | 8     | vegetable |
 +---------+-------+-----------*/
 ```

Compare the following equivalent queries that compute the number and total
amount of sales for each item in the `Produce` table:

**Standard syntax**

```googlesql
SELECT item, COUNT(*) AS num_items, SUM(sales) AS total_sales
FROM Produce
WHERE
  item != 'bananas'
  AND category IN ('fruit', 'nut')
GROUP BY item
ORDER BY item DESC;

/*--------+-----------+-------------+
 | item   | num_items | total_sales |
 +--------+-----------+-------------+
 | apples | 2         | 9           |
 +--------+-----------+-------------*/
````

**Pipe syntax**

```
FROM Produce
|> WHERE
    item != 'bananas'
    AND category IN ('fruit', 'nut')
|> AGGREGATE COUNT(*) AS num_items, SUM(sales) AS total_sales
   GROUP BY item
|> ORDER BY item DESC;

/*--------+-----------+-------------+
 | item   | num_items | total_sales |
 +--------+-----------+-------------+
 | apples | 2         | 9           |
 +--------+-----------+-------------*/
```

## Pipe operator semantics

Pipe operators have the following semantic behavior:

- Each pipe operator performs a self-contained operation.
- A pipe operator consumes the input table passed to it through the pipe symbol, `|>` , and produces a new table as output.
- A pipe operator can reference only columns from its immediate input table. Columns from earlier in the same query aren't visible. Inside subqueries, correlated references to outer columns are still allowed.

### Order preservation

The following operators preserve row order if the input table is ordered:

- [`SELECT`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#select_pipe_operator) (except when using `SELECT DISTINCT` or window functions)
- [`EXTEND`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#extend_pipe_operator) (except when using window functions)
- [`SET`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#set_pipe_operator) (except when using window functions)
- [`DROP`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#drop_pipe_operator)
- [`RENAME`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#rename_pipe_operator)
- [`AS`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#as_pipe_operator)
- [`LIMIT`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#limit_pipe_operator)

When you use these operators after an [`ORDER BY` operator](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#order_by_pipe_operator) , the result remains ordered. Additionally, if a [`LIMIT` operator](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#limit_pipe_operator) follows an order-preserving operator, the query computes the top rows based on that order.

## `FROM` queries

In pipe syntax, a query can start with a standard [`FROM` clause](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#from_clause) and use any standard `FROM` syntax, including tables, joins, subqueries, `UNNEST` operations, and table-valued functions (TVFs). Table aliases can be assigned to each input item using the [`AS alias` clause](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#using_aliases) .

A query with only a `FROM` clause, like `FROM table_name` , is allowed in pipe syntax and returns all rows from the table. For tables with columns, `FROM table_name` in pipe syntax is similar to [`SELECT * FROM table_name`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#select_star) in standard syntax.

**Examples**

The following queries use the [`Produce` table](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#query_comparison) :

```
FROM Produce;

/*---------+-------+-----------+
 | item    | sales | category  |
 +---------+-------+-----------+
 | apples  | 2     | fruit     |
 | apples  | 7     | fruit     |
 | bananas | 5     | fruit     |
 | carrots | 8     | vegetable |
 +---------+-------+-----------*/
```

```
-- Join tables in the FROM clause and then apply pipe operators.
FROM
  Produce AS p1
  JOIN Produce AS p2
    USING (item)
|> WHERE item = 'bananas'
|> SELECT p1.item, p2.sales;

/*---------+-------+
 | item    | sales |
 +---------+-------+
 | bananas | 5     |
 +---------+-------*/
```

## Pipe operators

GoogleSQL supports the following pipe operators. For operators that correspond or relate to similar operations in standard syntax, the operator descriptions highlight similarities and differences and link to more detailed documentation on the corresponding syntax.

### Pipe operator list

| Name                                                                                                                     | Summary                                                                                                                                                      |
|--------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [`SELECT`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#select_pipe_operator)           | Produces a new table with the listed columns.                                                                                                                |
| [`EXTEND`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#extend_pipe_operator)           | Propagates the existing table and adds computed columns.                                                                                                     |
| [`SET`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#set_pipe_operator)                 | Replaces the values of columns in the input table.                                                                                                           |
| [`DROP`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#drop_pipe_operator)               | Removes listed columns from the input table.                                                                                                                 |
| [`RENAME`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#rename_pipe_operator)           | Renames specified columns.                                                                                                                                   |
| [`AS`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#as_pipe_operator)                   | Introduces a table alias for the input table.                                                                                                                |
| [`WHERE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#where_pipe_operator)             | Filters the results of the input table.                                                                                                                      |
| [`AGGREGATE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#aggregate_pipe_operator)     | Performs aggregation on data across groups of rows or the full input table.                                                                                  |
| [`JOIN`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#join_pipe_operator)               | Joins rows from the input table with rows from a second table provided as an argument.                                                                       |
| [`ORDER BY`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#order_by_pipe_operator)       | Sorts results by a list of expressions.                                                                                                                      |
| [`LIMIT`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#limit_pipe_operator)             | Limits the number of rows to return in a query, with an optional `OFFSET` clause to skip over rows.                                                          |
| [`UNION`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#union_pipe_operator)             | Returns the combined results of the input queries to the left and right of the pipe operator.                                                                |
| [`INTERSECT`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#intersect_pipe_operator)     | Returns rows that are found in the results of both the input query to the left of the pipe operator and all input queries to the right of the pipe operator. |
| [`EXCEPT`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#except_pipe_operator)           | Returns rows from the input query to the left of the pipe operator that aren't present in any input queries to the right of the pipe operator.               |
| [`TABLESAMPLE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#tablesample_pipe_operator) | Selects a random sample of rows from the input table.                                                                                                        |

### `SELECT` pipe operator

```
|> SELECT expression [[AS] alias] [, ...]
```

**Description**

Produces a new table with the listed columns, similar to the outermost [`SELECT` clause](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#select_list) in a table subquery in standard syntax. The `SELECT` operator supports standard output modifiers like `SELECT AS STRUCT` and `SELECT DISTINCT` . The `SELECT` operator doesn't support aggregations or anonymization.

In pipe syntax, the `SELECT` operator in a query is optional. The `SELECT` operator can be used near the end of a query to specify the list of output columns. The final query result contains the columns returned from the last pipe operator. If the `SELECT` operator isn't used to select specific columns, the output includes the full row, similar to what the [`SELECT *` statement](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#select_star) in standard syntax produces.

In pipe syntax, the `SELECT` clause doesn't perform aggregation. Use the [`AGGREGATE` operator](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#aggregate_pipe_operator) instead.

For cases where `SELECT` would be used in standard syntax to rearrange columns, pipe syntax supports other operators:

- The [`EXTEND` operator](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#extend_pipe_operator) adds columns.
- The [`SET` operator](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#set_pipe_operator) updates the value of an existing column.
- The [`DROP` operator](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#drop_pipe_operator) removes columns.
- The [`RENAME` operator](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#rename_pipe_operator) renames columns.

**Examples**

```
FROM (SELECT 'apples' AS item, 2 AS sales)
|> SELECT item AS fruit_name;

/*------------+
 | fruit_name |
 +------------+
 | apples     |
 +------------*/
```

### `EXTEND` pipe operator

```
|> EXTEND expression [[AS] alias] [, ...]
```

**Description**

Propagates the existing table and adds computed columns, similar to [`SELECT *, new_column`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#select_star) in standard syntax.

**Examples**

```
(
  SELECT 'apples' AS item, 2 AS sales
  UNION ALL
  SELECT 'bananas' AS item, 8 AS sales
)
|> EXTEND item IN ('bananas', 'lemons') AS is_yellow;

/*---------+-------+------------+
 | item    | sales | is_yellow  |
 +---------+-------+------------+
 | apples  | 2     | FALSE      |
 | bananas | 8     | TRUE       |
 +---------+-------+------------*/
```

### `SET` pipe operator

```
|> SET column = expression [, ...]
```

**Description**

Replaces the value of a column in the input table, similar to [`SELECT * REPLACE (expression AS column)`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#select_replace) in standard syntax. Each referenced column must exist exactly once in the input table.

After a `SET` operation, the referenced top-level columns (like `x` ) are updated, but table aliases (like `t` ) still refer to the original row values. Therefore, `t.x` will still refer to the original value.

**Example**

```
(
  SELECT 1 AS x, 11 AS y
  UNION ALL
  SELECT 2 AS x, 22 AS y
)
|> SET x = x * x, y = 3;

/*---+---+
 | x | y |
 +---+---+
 | 1 | 3 |
 | 4 | 3 |
 +---+---*/
```

```
FROM (SELECT 2 AS x, 3 AS y) AS t
|> SET x = x * x, y = 8
|> SELECT t.x AS original_x, x, y;

/*------------+---+---+
 | original_x | x | y |
 +------------+---+---+
 | 2          | 4 | 8 |
 +------------+---+---*/
```

### `DROP` pipe operator

```
|> DROP column [, ...]
```

**Description**

Removes listed columns from the input table, similar to [`SELECT * EXCEPT (column)`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#select_except) in standard syntax. Each referenced column must exist at least once in the input table.

After a `DROP` operation, the referenced top-level columns (like `x` ) are removed, but table aliases (like `t` ) still refer to the original row values. Therefore, `t.x` will still refer to the original value.

**Example**

```
SELECT 'apples' AS item, 2 AS sales, 'fruit' AS category
|> DROP sales, category;

/*--------+
 | item   |
 +--------+
 | apples |
 +--------*/
```

```
FROM (SELECT 1 AS x, 2 AS y) AS t
|> DROP x
|> SELECT t.x AS original_x, y;

/*------------+---+
 | original_x | y |
 +------------+---+
 | 1          | 2 |
 +------------+---*/
```

### `RENAME` pipe operator

```
|> RENAME old_column_name [AS] new_column_name [, ...]
```

**Description**

Renames specified columns. Each column to be renamed must exist exactly once in the input table. The `RENAME` operator can't rename value table fields, pseudo-columns, range variables, or objects that aren't columns in the input table.

After a `RENAME` operation, the referenced top-level columns (like `x` ) are renamed, but table aliases (like `t` ) still refer to the original row values. Therefore, `t.x` will still refer to the original value.

**Example**

```
SELECT 1 AS x, 2 AS y, 3 AS z
|> AS t
|> RENAME y AS renamed_y
|> SELECT *, t.y AS t_y;

/*---+-----------+---+-----+
 | x | renamed_y | z | t_y |
 +---+-----------+---+-----+
 | 1 | 2         | 3 | 2   |
 +---+-----------+---+-----*/
```

### `AS` pipe operator

```
|> AS alias
```

**Description**

Introduces a table alias for the input table, similar to applying the [`AS alias` clause](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#using_aliases) on a table subquery in standard syntax. Any existing table aliases are removed and the new alias becomes the table alias for all columns in the row.

The `AS` operator can be useful after operators like [`SELECT`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#select_pipe_operator) , [`EXTEND`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#extend_pipe_operator) , or [`AGGREGATE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#aggregate_pipe_operator) that add columns but can't give table aliases to them. You can use the table alias to disambiguate columns after the `JOIN` operator.

**Example**

```
(
  SELECT "000123" AS id, "apples" AS item, 2 AS sales
  UNION ALL
  SELECT "000456" AS id, "bananas" AS item, 5 AS sales
) AS sales_table
|> AGGREGATE SUM(sales) AS total_sales GROUP BY id, item
-- AGGREGATE creates an output table, so the sales_table alias is now out of
-- scope. Add a t1 alias so the join can refer to its id column.
|> AS t1
|> JOIN (SELECT 456 AS id, "yellow" AS color) AS t2
   ON CAST(t1.id AS INT64) = t2.id
|> SELECT t2.id, total_sales, color;

/*-----+-------------+--------+
 | id  | total_sales | color  |
 +-----+-------------+--------+
 | 456 | 5           | yellow |
 +-----+-------------+--------*/
```

### `WHERE` pipe operator

```
|> WHERE boolean_expression
```

**Description**

Filters the results of the input table. The `WHERE` operator behaves the same as the [`WHERE` clause](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#where_clause) in standard syntax.

In pipe syntax, the `WHERE` operator also replaces the [`HAVING` clause](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#having_clause) in standard syntax. For example, after performing aggregation with the [`AGGREGATE` operator](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#aggregate_pipe_operator) , use the `WHERE` operator instead of the `HAVING` clause.

**Example**

```
(
  SELECT 'apples' AS item, 2 AS sales
  UNION ALL
  SELECT 'bananas' AS item, 5 AS sales
  UNION ALL
  SELECT 'carrots' AS item, 8 AS sales
)
|> WHERE sales >= 3;

/*---------+-------+
 | item    | sales |
 +---------+-------+
 | bananas | 5     |
 | carrots | 8     |
 +---------+-------*/
```

### `AGGREGATE` pipe operator

```
-- Full-table aggregation
|> AGGREGATE aggregate_expression [[AS] alias] [, ...]
```

```
-- Aggregation with grouping
|> AGGREGATE [aggregate_expression [[AS] alias] [, ...]]
   GROUP BY groupable_items [[AS] alias] [, ...]
```

```
-- Aggregation with grouping and shorthand ordering syntax
|> AGGREGATE [aggregate_expression [[AS] alias] [order_suffix] [, ...]]
   GROUP [AND ORDER] BY groupable_item [[AS] alias] [order_suffix] [, ...]

order_suffix: {ASC | DESC}
```

**Description**

Performs aggregation on data across grouped rows or an entire table. The `AGGREGATE` operator is similar to a query in standard syntax that contains a [`GROUP BY` clause](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#group_by_clause) or a `SELECT` list with [aggregate functions](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/aggregate_functions) or both. In pipe syntax, the `GROUP BY` clause is part of the `AGGREGATE` operator. Pipe syntax doesn't support a standalone `GROUP BY` operator.

Without the `GROUP BY` clause, the `AGGREGATE` operator performs full-table aggregation and produces one output row.

With the `GROUP BY` clause, the `AGGREGATE` operator performs aggregation with grouping, producing one row for each set of distinct values for the grouping expressions.

The `AGGREGATE` expression list corresponds to the aggregated expressions in a `SELECT` list in standard syntax. Each expression in the `AGGREGATE` list must include an aggregate function. Aggregate expressions can also include scalar expressions (for example, `sqrt(SUM(x*x))` ). Column aliases can be assigned using the [`AS` operator](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#as_pipe_operator) .

The `GROUP BY` clause in the `AGGREGATE` operator corresponds to the `GROUP BY` clause in standard syntax. Unlike in standard syntax, aliases can be assigned to `GROUP BY` items.

The output columns from the `AGGREGATE` operator include all grouping columns first, followed by all aggregate columns, using their assigned aliases as the column names.

Unlike in standard syntax, grouping expressions aren't repeated across `SELECT` and `GROUP BY` clauses. In pipe syntax, the grouping expressions are listed once, in the `GROUP BY` clause, and are automatically included as output columns for the `AGGREGATE` operator.

Because output columns are fully specified by the `AGGREGATE` operator, the `SELECT` operator isn't needed after the `AGGREGATE` operator unless you want to produce a list of columns different from the default.

**Standard syntax**

```
-- Aggregation in standard syntax
SELECT SUM(col1) AS total, col2, col3, col4...
FROM table1
GROUP BY col2, col3, col4...
```

**Pipe syntax**

```
-- The same aggregation in pipe syntax
FROM table1
|> AGGREGATE SUM(col1) AS total
   GROUP BY col2, col3, col4...
```

**Examples**

```
-- Full-table aggregation
(
  SELECT 'apples' AS item, 2 AS sales
  UNION ALL
  SELECT 'bananas' AS item, 5 AS sales
  UNION ALL
  SELECT 'apples' AS item, 7 AS sales
)
|> AGGREGATE COUNT(*) AS num_items, SUM(sales) AS total_sales;

/*-----------+-------------+
 | num_items | total_sales |
 +-----------+-------------+
 | 3         | 14          |
 +-----------+-------------*/
```

```
-- Aggregation with grouping
(
  SELECT 'apples' AS item, 2 AS sales
  UNION ALL
  SELECT 'bananas' AS item, 5 AS sales
  UNION ALL
  SELECT 'apples' AS item, 7 AS sales
)
|> AGGREGATE COUNT(*) AS num_items, SUM(sales) AS total_sales
   GROUP BY item;

/*---------+-----------+-------------+
 | item    | num_items | total_sales |
 +---------+-----------+-------------+
 | apples  | 2         | 9           |
 | bananas | 1         | 5           |
 +---------+-----------+-------------*/
```

#### Shorthand ordering syntax with `AGGREGATE`

The `AGGREGATE` operator supports a shorthand ordering syntax, which is equivalent to applying the [`ORDER BY` operator](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#order_by_pipe_operator) as part of the `AGGREGATE` operator without repeating the column list:

```
-- Aggregation with grouping and shorthand ordering syntax
|> AGGREGATE [aggregate_expression [[AS] alias] [order_suffix] [, ...]]
   GROUP [AND ORDER] BY groupable_item [[AS] alias] [order_suffix] [, ...]

order_suffix: {ASC | DESC}
```

The `GROUP AND ORDER BY` clause is equivalent to an `ORDER BY` clause on all `groupable_items` . By default, each `groupable_item` is sorted in ascending order with `NULL` values first. Other ordering suffixes like `DESC` can be used for other orders.

Without the `GROUP AND ORDER BY` clause, the `ASC` or `DESC` suffixes can be added on individual columns in the `GROUP BY` list or `AGGREGATE` list or both.

Adding these suffixes is equivalent to adding an `ORDER BY` clause that includes all of the suffixed columns with the suffixed grouping columns first, matching the left-to-right output column order.

**Examples**

Consider the following table called `Produce` :

```
/*---------+-------+-----------+
 | item    | sales | category  |
 +---------+-------+-----------+
 | apples  | 2     | fruit     |
 | carrots | 8     | vegetable |
 | apples  | 7     | fruit     |
 | bananas | 5     | fruit     |
 +---------+-------+-----------*/
```

The following two equivalent examples show you how to order by all grouping columns using the `GROUP AND ORDER BY` clause or a separate `ORDER BY` clause:

```
-- Order by all grouping columns using GROUP AND ORDER BY.
FROM Produce
|> AGGREGATE SUM(sales) AS total_sales
   GROUP AND ORDER BY category, item DESC;

/*-----------+---------+-------------+
 | category  | item    | total_sales |
 +-----------+---------+-------------+
 | fruit     | bananas | 5           |
 | fruit     | apples  | 9           |
 | vegetable | carrots | 8           |
 +-----------+---------+-------------*/
```

```
--Order by columns using ORDER BY after performing aggregation.
FROM Produce
|> AGGREGATE SUM(sales) AS total_sales
   GROUP BY category, item
|> ORDER BY category, item DESC;
```

You can add an ordering suffix to a column in the `AGGREGATE` list. Although the `AGGREGATE` list appears before the `GROUP BY` list in the query, ordering suffixes on columns in the `GROUP BY` list are applied first.

```
FROM Produce
|> AGGREGATE SUM(sales) AS total_sales ASC
   GROUP BY item, category DESC;

/*---------+-----------+-------------+
 | item    | category  | total_sales |
 +---------+-----------+-------------+
 | carrots | vegetable | 8           |
 | bananas | fruit     | 5           |
 | apples  | fruit     | 9           |
 +---------+-----------+-------------*/
```

The previous query is equivalent to the following:

```
-- Order by specified grouping and aggregate columns.
FROM Produce
|> AGGREGATE SUM(sales) AS total_sales
   GROUP BY item, category
|> ORDER BY category DESC, total_sales;
```

### `JOIN` pipe operator

```
|> [join_type] JOIN from_item [[AS] alias] [{on_clause | using_clause}]
```

**Description**

Joins rows from the input table with rows from a second table provided as an argument. The `JOIN` operator behaves the same as the [`JOIN` operation](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#join_types) in standard syntax. The input table is the left side of the join and the `JOIN` argument is the right side of the join. Standard join inputs are supported, including tables, subqueries, `UNNEST` operations, and table-valued function (TVF) calls. Standard join modifiers like `LEFT` , `INNER` , and `CROSS` are allowed before the `JOIN` keyword.

An alias can be assigned to the input table on the right side of the join, but not to the input table on the left side of the join. If an alias on the input table is needed, perhaps to disambiguate columns in an [`ON` expression](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#on_clause) , then an alias can be added using the [`AS` operator](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#as_pipe_operator) before the `JOIN` arguments.

**Example**

```
(
  SELECT 'apples' AS item, 2 AS sales
  UNION ALL
  SELECT 'bananas' AS item, 5 AS sales
)
|> AS produce_sales
|> LEFT JOIN
     (
       SELECT "apples" AS item, 123 AS id
     ) AS produce_data
   ON produce_sales.item = produce_data.item
|> SELECT produce_sales.item, sales, id;

/*---------+-------+------+
 | item    | sales | id   |
 +---------+-------+------+
 | apples  | 2     | 123  |
 | bananas | 5     | NULL |
 +---------+-------+------*/
```

### `ORDER BY` pipe operator

```
|> ORDER BY expression [sort_options] [, ...]
```

**Description**

Sorts results by a list of expressions. The `ORDER BY` operator behaves the same as the [`ORDER BY` clause](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#order_by_clause) in standard syntax. Suffixes like `DESC` are supported for customizing the ordering for each expression.

In pipe syntax, the [`AGGREGATE` operator](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#aggregate_pipe_operator) also supports [shorthand ordering suffixes](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax#shorthand_order_pipe_syntax) to apply `ORDER BY` behavior more concisely as part of aggregation.

**Example**

```
(
  SELECT 1 AS x
  UNION ALL
  SELECT 3 AS x
  UNION ALL
  SELECT 2 AS x
)
|> ORDER BY x DESC;

/*---+
 | x |
 +---+
 | 3 |
 | 2 |
 | 1 |
 +---*/
```

### `LIMIT` pipe operator

```
|> LIMIT count [OFFSET skip_rows]
```

**Description**

Limits the number of rows to return in a query, with an optional `OFFSET` clause to skip over rows. The `LIMIT` operator behaves the same as the [`LIMIT` and `OFFSET` clause](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#limit_and_offset_clause) in standard syntax.

**Examples**

```
(
  SELECT 'apples' AS item, 2 AS sales
  UNION ALL
  SELECT 'bananas' AS item, 5 AS sales
  UNION ALL
  SELECT 'carrots' AS item, 8 AS sales
)
|> ORDER BY item
|> LIMIT 1;

/*---------+-------+
 | item    | sales |
 +---------+-------+
 | apples  | 2     |
 +---------+-------*/
```

```
(
  SELECT 'apples' AS item, 2 AS sales
  UNION ALL
  SELECT 'bananas' AS item, 5 AS sales
  UNION ALL
  SELECT 'carrots' AS item, 8 AS sales
)
|> ORDER BY item
|> LIMIT 1 OFFSET 2;

/*---------+-------+
 | item    | sales |
 +---------+-------+
 | carrots | 8     |
 +---------+-------*/
```

### `UNION` pipe operator

```
query
|> UNION {ALL | DISTINCT} (query) [, (query), ...]
```

**Description**

Returns the combined results of the input queries to the left and right of the pipe operator. Columns are matched and rows are concatenated vertically.

The `UNION` pipe operator behaves the same as the [`UNION` set operator](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#union) in standard syntax. However, in pipe syntax, the `UNION` pipe operator can include multiple comma-separated queries without repeating the `UNION` syntax. Queries following the operator are enclosed in parentheses.

For example, compare the following equivalent queries:

```
-- Standard syntax
SELECT * FROM ...
UNION ALL
SELECT 1
UNION ALL
SELECT 2;

-- Pipe syntax
SELECT * FROM ...
|> UNION ALL
    (SELECT 1),
    (SELECT 2);
```

**Examples**

```
SELECT * FROM UNNEST(ARRAY<INT64>[1, 2, 3]) AS number
|> UNION ALL (SELECT 1);

/*--------+
 | number |
 +--------+
 | 1      |
 | 2      |
 | 3      |
 | 1      |
 +--------*/
```

```
SELECT * FROM UNNEST(ARRAY<INT64>[1, 2, 3]) AS number
|> UNION DISTINCT (SELECT 1);

/*--------+
 | number |
 +--------+
 | 1      |
 | 2      |
 | 3      |
 +--------*/
```

The following example shows multiple input queries to the right of the pipe operator:

```
SELECT * FROM UNNEST(ARRAY<INT64>[1, 2, 3]) AS number
|> UNION DISTINCT
    (SELECT 1),
    (SELECT 2);

/*--------+
 | number |
 +--------+
 | 1      |
 | 2      |
 | 3      |
 +--------*/
```

### `INTERSECT` pipe operator

```
query
|> INTERSECT {ALL | DISTINCT} (query) [, (query), ...]
```

**Description**

Returns rows that are found in the results of both the input query to the left of the pipe operator and all input queries to the right of the pipe operator.

The `INTERSECT` pipe operator behaves the same as the [`INTERSECT` set operator](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#intersect) in standard syntax. However, in pipe syntax, the `INTERSECT` pipe operator can include multiple comma-separated queries without repeating the `INTERSECT` syntax. Queries following the operator are enclosed in parentheses.

For example, compare the following equivalent queries:

```
-- Standard syntax
SELECT * FROM ...
INTERSECT ALL
SELECT 1
INTERSECT ALL
SELECT 2;

-- Pipe syntax
SELECT * FROM ...
|> INTERSECT ALL
    (SELECT 1),
    (SELECT 2);
```

**Examples**

```
SELECT * FROM UNNEST(ARRAY<INT64>[1, 2, 3, 3, 4]) AS number
|> INTERSECT ALL
    (SELECT * FROM UNNEST(ARRAY<INT64>[2, 3, 3, 5]) AS number);

/*--------+
 | number |
 +--------+
 | 2      |
 | 3      |
 | 3      |
 +--------*/
```

```
SELECT * FROM UNNEST(ARRAY<INT64>[1, 2, 3, 3, 4]) AS number
|> INTERSECT DISTINCT
    (SELECT * FROM UNNEST(ARRAY<INT64>[2, 3, 3, 5]) AS number);

/*--------+
 | number |
 +--------+
 | 2      |
 | 3      |
 +--------*/
```

The following example shows multiple input queries to the right of the pipe operator:

```
SELECT * FROM UNNEST(ARRAY<INT64>[1, 2, 3, 3, 4]) AS number
|> INTERSECT DISTINCT
    (SELECT * FROM UNNEST(ARRAY<INT64>[2, 3, 3, 5]) AS number),
    (SELECT * FROM UNNEST(ARRAY<INT64>[3, 3, 4, 5]) AS number);

/*--------+
 | number |
 +--------+
 | 3      |
 +--------*/
```

### `EXCEPT` pipe operator

```
query
|> EXCEPT {ALL | DISTINCT} (query) [, (query), ...]
```

**Description**

Returns rows from the input query to the left of the pipe operator that aren't present in any input queries to the right of the pipe operator.

The `EXCEPT` pipe operator behaves the same as the [`EXCEPT` set operator](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#except) in standard syntax. However, in pipe syntax, the `EXCEPT` pipe operator can include multiple comma-separated queries without repeating the `EXCEPT` syntax. Queries following the operator are enclosed in parentheses.

For example, compare the following equivalent queries:

```
-- Standard syntax
SELECT * FROM ...
EXCEPT ALL
SELECT 1
EXCEPT ALL
SELECT 2;

-- Pipe syntax
SELECT * FROM ...
|> EXCEPT ALL
    (SELECT 1),
    (SELECT 2);
```

Parentheses can be used to group set operations and control order of operations. In `EXCEPT` set operations, query results can vary depending on the operation grouping.

```
-- Default operation grouping
(
  SELECT * FROM ...
  EXCEPT ALL
  SELECT 1
)
EXCEPT ALL
SELECT 2;

-- Modified operation grouping
SELECT * FROM ...
EXCEPT ALL
(
  SELECT 1
  EXCEPT ALL
  SELECT 2
);

-- Same modified operation grouping in pipe syntax
SELECT * FROM ...
|> EXCEPT ALL
(
  SELECT 1
  |> EXCEPT ALL (SELECT 2)
);
```

**Examples**

```
SELECT * FROM UNNEST(ARRAY<INT64>[1, 2, 3, 3, 4]) AS number
|> EXCEPT ALL
    (SELECT * FROM UNNEST(ARRAY<INT64>[1, 2]) AS number)
|> ORDER BY number;

/*--------+
 | number |
 +--------+
 | 3      |
 | 3      |
 | 4      |
 +--------*/
```

```
SELECT * FROM UNNEST(ARRAY<INT64>[1, 2, 3, 3, 4]) AS number
|> EXCEPT DISTINCT
    (SELECT * FROM UNNEST(ARRAY<INT64>[1, 2]) AS number)
|> ORDER BY number;

/*--------+
 | number |
 +--------+
 | 3      |
 | 4      |
 +--------*/
```

The following example shows multiple input queries to the right of the pipe operator:

```
SELECT * FROM UNNEST(ARRAY<INT64>[1, 2, 3, 3, 4]) AS number
|> EXCEPT DISTINCT
    (SELECT * FROM UNNEST(ARRAY<INT64>[1, 2]) AS number),
    (SELECT * FROM UNNEST(ARRAY<INT64>[1, 4]) AS number);

/*--------+
 | number |
 +--------+
 | 3      |
 +--------*/
```

The following example groups the set operations to modify the order of operations. The first input query is used against the result of the last two queries instead of the values of the last two queries individually.

```
SELECT * FROM UNNEST(ARRAY<INT64>[1, 2, 3, 3, 4]) AS number
|> EXCEPT DISTINCT
(
  SELECT * FROM UNNEST(ARRAY<INT64>[1, 2]) AS number
  |> EXCEPT DISTINCT
      (SELECT * FROM UNNEST(ARRAY<INT64>[1, 4]) AS number)
) |> ORDER BY number;

/*--------+
 | number |
 +--------+
 | 1      |
 | 3      |
 | 4      |
 +--------*/
```

### `TABLESAMPLE` pipe operator

```
|> TABLESAMPLE sample_method (sample_size {PERCENT | ROWS}) [, ...]
```

**Description**

Selects a random sample of rows from the input table. The `TABLESAMPLE` pipe operator behaves the same as [`TABLESAMPLE` operator](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax#tablesample_operator) in standard syntax.

**Example**

The following example samples approximately 1% of data from a table called `LargeTable` :

```
FROM LargeTable
|> TABLESAMPLE SYSTEM (1 PERCENT);
```
