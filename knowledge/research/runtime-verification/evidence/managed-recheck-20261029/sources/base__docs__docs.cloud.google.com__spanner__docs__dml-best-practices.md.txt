---
name: documents/docs.cloud.google.com/spanner/docs/dml-best-practices
uri: https://docs.cloud.google.com/spanner/docs/dml-best-practices
title: Data Manipulation Language best practices
description: Best practices for DML statements in Spanner to optimize performance and avoid lock contention.
data_source: docs.cloud.google.com
---

This page describes best practices for using data manipulation language (DML) and partitioned DML for GoogleSQL-dialect databases and PostgreSQL-dialect databases.

## Use a `WHERE` clause to reduce the scope of locks

You execute DML statements inside read-write transactions. When Spanner reads data, it acquires shared read locks on limited portions of the row ranges that you read. Specifically, it acquires these locks only on the columns you access. The locks can include data that does not satisfy the filter condition of the `WHERE` clause.

When Spanner modifies data using DML statements, it acquires exclusive locks on the specific data that you are modifying. In addition, it acquires shared locks in the same way as when you read data. If your request includes large row ranges, or an entire table, the shared locks might prevent other transactions from making progress in parallel.

To modify data as efficiently as possible, use a `WHERE` clause that enables Spanner to read only the necessary rows. You can achieve this goal with a filter on the primary key, or on the key of a secondary index. The `WHERE` clause limits the scope of the shared locks and enables Spanner to process the update more efficiently.

For example, suppose that one of the musicians in the `Singers` table changes their first name, and you need to update the name in your database. You could execute the following DML statement, but it forces Spanner to scan the entire table and acquires shared locks that cover the entire table. As a result, Spanner must read more data than necessary, and concurrent transactions cannot modify the data in parallel:

```
-- ANTI-PATTERN: SENDING AN UPDATE WITHOUT THE PRIMARY KEY COLUMN
-- IN THE WHERE CLAUSE

UPDATE Singers SET FirstName = "Marcel"
WHERE FirstName = "Marc" AND LastName = "Richards";
```

To make the update more efficient, include the `SingerId` column in the `WHERE` clause. The `SingerId` column is the only primary key column for the `Singers` table:

```
-- ANTI-PATTERN: SENDING AN UPDATE THAT MUST SCAN THE ENTIRE TABLE

UPDATE Singers SET FirstName = "Marcel"
WHERE FirstName = "Marc" AND LastName = "Richards"
```

If there is no index on `FirstName` or `LastName` , you need to scan the entire table to find the target singers. If you don't want to add a secondary index to make the update more efficient, then include the `SingerId` column in the `WHERE` clause.

The `SingerId` column is the only primary key column for the `Singers` table. To find it, run `SELECT` in a separate, read-only transaction prior to the update transaction:

```
  SELECT SingerId
  FROM Singers
  WHERE FirstName = "Marc" AND LastName = "Richards"

  -- Recommended: Including a seekable filter in the where clause

  UPDATE Singers SET FirstName = "Marcel"
  WHERE SingerId = 1;
```

## Avoid using DML statements and mutations in the same transaction

Spanner buffers insertions, updates, and deletions performed using DML statements on the server-side, and the results are visible to subsequent SQL and DML statements within the same transaction. This behavior is different from the [Mutation API](https://docs.cloud.google.com/spanner/docs/modify-mutation-api) , where Spanner buffers the mutations on the client-side and sends the mutations server-side as part of the commit operation. As a result, mutations in the commit request aren't visible to SQL or DML statements within the same transaction.

Avoid using both DML statements and mutations in the same transaction. If you use both in the same transaction, you need to account for the order of execution in your client library code. If a transaction contains both DML statements and mutations in the same request, Spanner executes the DML statements before the mutations.

For operations that are only supported using mutations, you might want to combine DML statements and mutations in the same transaction—for example, [`insert_or_update`](https://docs.cloud.google.com/spanner/docs/reference/rpc/google.spanner.v1#google.spanner.v1.Mutation) .

If you use both, the buffer writes only at the very end of the transaction.

## Use the `PENDING_COMMIT_TIMESTAMP` function to write commit timestamps

### GoogleSQL

You use the [`PENDING_COMMIT_TIMESTAMP`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/timestamp_functions#pending_commit_timestamp) function to write the commit timestamp in a DML statement. Spanner selects the commit timestamp when the transaction commits.

> **Note:** After you call the `PENDING_COMMIT_TIMESTAMP` function, the table and any derived index is unreadable to any future SQL statements in the transaction. Because of this, the change stream can't extract the previous value for the column that has a pending commit timestamp, if the coloumn is modified again later in the same transaction. You must write commit timestamps as the last statement in a transaction to prevent the possibility of trying to read the table. If you try to read the table, then Spanner produces an error.

### PostgreSQL

You use the `SPANNER.PENDING_COMMIT_TIMESTAMP()` function to write the commit timestamp in a DML statement. Spanner selects the commit timestamp when the transaction commits.

> **Note:** After you call the `SPANNER.PENDING_COMMIT_TIMESTAMP()` function, the table and any derived index is unreadable to any subsequent SQL statements in the transaction. You must write commit timestamps as the last statement in a transaction to prevent the possibility of trying to read the table. If you try to read the table, then Spanner returns an error.

## Partitioned DML and date and timestamp functions

Partitioned DML uses one or more transactions that might run and commit at different times. If you use the [date](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/date_functions) or [timestamp](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/timestamp_functions) functions, the modified rows might contain different values.

## Improve latency with batch DML

To reduce latency, use [batch DML](https://docs.cloud.google.com/spanner/docs/dml-tasks#use-batch) to send multiple DML statements to Spanner within a single client-server round trip.

Batch DML can apply optimizations to groups of statements within a batch to enable faster and more efficient data updates.

- **Execute writes with a single request**

  Spanner automatically optimizes contiguous groups of similar `INSERT` , `UPDATE` , or `DELETE` batched statements that have different parameter values, if they don't violate data dependencies.

  For example, consider a scenario where you want to insert a large set of new rows into a table called `Albums` . To let Spanner optimize all the required `INSERT` statements into a single, efficient server-side action, begin by writing an appropriate DML statement that uses SQL query parameters:

  ```
  INSERT INTO Albums (SingerId, AlbumId, AlbumTitle) VALUES (@Singer, @Album, @Title);
  ```

  Then, send Spanner a DML batch that invokes this statement repeatedly and contiguously, with the repetitions differing only in the values you bind to the statement's three query parameters. Spanner optimizes these structurally identical DML statements into a single server-side operation before executing it.

- **Execute writes in parallel**

  Spanner automatically optimizes contiguous groups of DML statements by executing in parallel when doing so doesn't violate data dependencies. This optimization brings performance benefits to a wider set of batched DML statements because it can apply to a mix of DML statement types ( `INSERT` , `UPDATE` and `DELETE` ) and to both parameterized or non-parameterized DML statements.

  For example, our sample schema has the tables `Singers` , `Albums` , and `Accounts` . `Albums` is interleaved within `Singers` and stores information about albums for `Singers` . The following contiguous group of statements writes new rows to multiple tables and doesn't have complex data dependencies.

  ```
  INSERT INTO Singers (SingerId, Name) VALUES(1, "John Doe");
  INSERT INTO Singers (SingerId, Name) VALUES(2, "Marcel Richards");
  INSERT INTO Albums(SingerId, AlbumId, AlbumTitle) VALUES (1, 10001, "Album 1");
  INSERT INTO Albums(SingerId, AlbumId, AlbumTitle) VALUES (1, 10002, "Album 2");
  INSERT INTO Albums(SingerId, AlbumId, AlbumTitle) VALUES (2, 10001, "Album 1");
  UPDATE Accounts SET Balance = 100 WHERE AccountId = @AccountId;
  ```

  Spanner optimizes this group of DML statements by executing the statements in parallel. The writes are applied in order of the statements in the batch and maintains batch DML semantics if a statement fails during execution.

### Enable client-side batching in JDBC

For Java applications using a Spanner-supported [JDBC driver](https://docs.cloud.google.com/spanner/docs/jdbc-drivers) , you can reduce latency by enabling client-side DML batching. The JDBC driver has a [connection property](https://github.com/googleapis/java-spanner-jdbc/blob/main/documentation/connection_properties.md) called `auto_batch_dml` that, when enabled, buffers DML statements on the client and sends them to Spanner as a single batch. This can reduce the number of round trips to the server and improve overall performance.

By default, `auto_batch_dml` is set to `false` . You can enable it by setting it to `true` in your JDBC connection string.

For example:

```
String url = "jdbc:cloudspanner:/projects/my-project/instances/my-instance/databases/my-database;auto_batch_dml=true";
try (Connection connection = DriverManager.getConnection(url)) {
    // Include your DML statements for batching here
}
```

With this connection property enabled, Spanner sends buffered DML statements as a batch when a non-DML statement is executed or when the current transaction is committed. This property only applies to read-write transactions; DML statements in autocommit mode are executed directly.

By default, the update count for buffered DML statements is set to `1` . You can change this by setting the `auto_batch_dml_update_count` connection variable to a different value. For more information, see [JDBC supported connection properties](https://github.com/googleapis/java-spanner-jdbc/blob/main/documentation/connection_properties.md) .

## Use the `last_statement` option to reduce DML latency

When the last statement in a read-write transaction is a DML statement, you can use the `last_statement` query option to reduce latency. This option is available in the [`executeSql`](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases.sessions/executeSql) and [`executeStreamingSql`](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases.sessions/executeStreamingSql) query APIs.

Using this option defers some validation steps, such as unique constraint validation, until the transaction is committed. When using `last_statement` , subsequent operations, such as reads, queries, and DML, in the same transaction are rejected. This option isn't compatible with mutations. If you include mutations in the same transaction, Spanner returns an error.

The `last_statement` option is supported in the following client libraries:

- Go in version 1.77.0 or later
- Java in version 2.27.0 or later
- Python in version 3.53.0 or later
- PGAdapter in version 0.45.0 or later

It is supported and enabled by default when using the autocommit mode in the following drivers:

- [JDBC driver](https://docs.cloud.google.com/spanner/docs/use-oss-jdbc#use_a_transaction_in_autocommit_mode_to_add_rows) in version 6.87.0 or later

- [Go database/sql driver](https://docs.cloud.google.com/spanner/docs/use-golang-database-sql) in version 1.11.2 or later

- [Python dbapi driver](https://docs.cloud.google.com/python/docs/reference/spanner/latest#connection-api) in version 3.53.0 or later

### Go

### GoogleSQL

```
import (
    "context"
    "fmt"
    "io"

    "cloud.google.com/go/spanner"
)

// Updates a row while also setting the update DML as the last
// statement.
func updateDmlWithLastStatement(w io.Writer, db string) error {
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    _, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        // other statements for the transaction if any.

        updateStmt := spanner.Statement{
            SQL: `UPDATE Singers SET LastName = 'Doe' WHERE SingerId = 54213`,
        }
        opts := spanner.QueryOptions{LastStatement: true}
        updateRowCount, err := txn.UpdateWithOptions(ctx, updateStmt, opts)
        if err != nil {
            return err
        }
        fmt.Fprintf(w, "%d record(s) updated.\n", updateRowCount)
        return nil
    })
    if err != nil {
        return err
    }

    return nil
}
```

### PostgreSQL

```
import (
    "context"
    "fmt"
    "io"

    "cloud.google.com/go/spanner"
)

// Updates a row while also setting the update DML as the last
// statement.
func pgUpdateDmlWithLastStatement(w io.Writer, db string) error {
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    _, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        // other statements for the transaction if any.

        updateStmt := spanner.Statement{
            SQL: `UPDATE Singers SET LastName = 'Doe' WHERE SingerId = 54214`,
        }
        opts := spanner.QueryOptions{LastStatement: true}
        updateRowCount, err := txn.UpdateWithOptions(ctx, updateStmt, opts)
        if err != nil {
            return err
        }
        fmt.Fprintf(w, "%d record(s) updated.\n", updateRowCount)
        return nil
    })
    if err != nil {
        return err
    }

    return nil
}
```

### Java

### GoogleSQL

```
static void UpdateUsingLastStatement(DatabaseClient client) {
    client
        .readWriteTransaction()
        .run(
            transaction -> {
            // other statements for the transaction if any

            // Pass in the `lastStatement` option to the last DML statement of the transaction.
            transaction.executeUpdate(
                Statement.of(
                    "UPDATE Singers SET Singers.LastName = 'Doe' WHERE SingerId = 54213\n"),
                Options.lastStatement());
            System.out.println("Singer last name updated.");

            return null;
            });
}
```

### PostgreSQL

```
static void UpdateUsingLastStatement(DatabaseClient client) {
    client
        .readWriteTransaction()
        .run(
            transaction -> {
            // other statements for the transaction if any.

            // Pass in the `lastStatement` option to the last DML statement of the transaction.
            transaction.executeUpdate(
                Statement.of("UPDATE Singers SET LastName = 'Doe' WHERE SingerId = 54214\n"),
                Options.lastStatement());
            System.out.println("Singer last name updated.");

            return null;
            });
}
```

### Python

### GoogleSQL

```
def dml_last_statement_option(instance_id, database_id):
"""Updates using DML where the update set the last statement option."""
# [START spanner_dml_last_statement]
# instance_id = "your-spanner-instance"
# database_id = "your-spanner-db-id"

spanner_client = spanner.Client()
instance = spanner_client.instance(instance_id)
database = instance.database(database_id)

def update_singers(transaction):
    # other statements for the transaction if any.

    update_row_ct = transaction.execute_update(
        "UPDATE Singers SET LastName = 'Doe' WHERE SingerId = 54213",
        last_statement=True)

    print("{} record(s) updated.".format(update_row_ct))

database.run_in_transaction(update_singers)
```

### PostgreSQL

```
def dml_last_statement_option(instance_id, database_id):
"""Updates using DML where the update set the last statement option."""
# instance_id = "your-spanner-instance"
# database_id = "your-spanner-db-id"

spanner_client = spanner.Client()
instance = spanner_client.instance(instance_id)
database = instance.database(database_id)

def update_singers(transaction):
    # other statements for the transaction if any.

    update_row_ct = transaction.execute_update(
        "UPDATE Singers SET LastName = 'Doe' WHERE SingerId = 54214",
        last_statement=True)

    print("{} record(s) updated.".format(update_row_ct))

database.run_in_transaction(update_singers)
```
