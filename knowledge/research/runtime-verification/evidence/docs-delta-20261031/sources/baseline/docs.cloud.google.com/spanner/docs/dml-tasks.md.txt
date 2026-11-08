---
name: documents/docs.cloud.google.com/spanner/docs/dml-tasks
uri: https://docs.cloud.google.com/spanner/docs/dml-tasks
title: Insert, update, and delete data using data manipulation language (DML)
description: Manage data in Spanner using DML statements with client libraries, the {{dynamic_data.site_values.cloud_name_short}} console, and Google Cloud CLI.
data_source: docs.cloud.google.com
---

This page describes how to insert, update, and delete Spanner data using Data Manipulation Language (DML) statements. You can run DML statements using the [client libraries](https://docs.cloud.google.com/spanner/docs/dml-tasks#client-library-dml) , the [Google Cloud console](https://docs.cloud.google.com/spanner/docs/modify-data) , and the [`gcloud`](https://docs.cloud.google.com/spanner/docs/modify-gcloud) command-line tool. You can run [Partitioned DML](https://docs.cloud.google.com/spanner/docs/dml-partitioned) statements using the client libraries and the [`gcloud`](https://docs.cloud.google.com/spanner/docs/modify-gcloud) command-line tool.

For the complete DML syntax reference, see [Data Manipulation Language syntax](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax) for GoogleSQL-dialect databases or [PostgreSQL data manipulation language](https://docs.cloud.google.com/spanner/docs/reference/postgresql/dml-syntax) for PostgreSQL-dialect databases

> **Note:** To explore Spanner using a 90-day free trial instance, see [Create a Spanner free trial instance](https://docs.cloud.google.com/spanner/docs/free-trial-quickstart) .

## Use DML

DML supports `INSERT` , `UPDATE` , and `DELETE` statements in the Google Cloud console, Google Cloud CLI, and client libraries.

### Locking

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

### Concurrency

Spanner sequentially executes all the SQL statements ( `SELECT` , `INSERT` , `UPDATE` , and `DELETE` ) within a transaction. They are not executed concurrently. The only exception is that Spanner might execute multiple `SELECT` statements concurrently, because they are read-only operations.

### Transaction limits

A transaction that includes DML statements has the [same limits](https://docs.cloud.google.com/spanner/quotas#limits_for_creating_reading_updating_and_deleting_data) as any other transaction. If you have large-scale changes, consider using [Partitioned DML](https://docs.cloud.google.com/spanner/docs/dml-partitioned) .

The mutation limit is applied differently depending on whether you use DML statements or the Mutation API:

- **DML statements** : The limit of 80,000 mutations (including indexes) is applied per statement and is reset at the end of each statement. If a single DML statement exceeds this limit, it fails and returns a `BadUsage` error. However, a *transaction* can exceed 80,000 mutations by containing multiple DML statements, as long as each individual statement stays under that limit.

- **Mutation API** : Mutations using the Mutation API can only be applied as part of the commit operation, not in the middle of a transaction, unlike DML. The limit of 80,000 mutations (including indexes) is cumulative across all mutations in the commit. If a transaction uses mutations (either alone or mixed with DML), this cumulative limit applies to all mutations in the commit.

- **Transaction size** : All transactions are subject to a size limit of 100 MiB. For DML, if the statements in a transaction result in a transaction larger than 100 MiB, the statement that pushes it over the limit fails with a `BadUsage` error. Mutations performed using DML don't count towards the 80,000 limit across statements, but they do count towards this transaction size limit when committed.

### Run statements in the Google Cloud console

> **Note:** Spanner Studio (formerly labeled **Query** in the Google Cloud console) supports SQL, DML, and DDL operations in a single editor. For more information, see [Manage your data using the Google Cloud console](https://docs.cloud.google.com/spanner/docs/manage-data-using-console) .

Use the following steps to execute a DML statement in the Google Cloud console.

1.  Go to the Spanner **Instances** page.

2.  Select your project in the drop-down list in the toolbar.

3.  Click the name of the instance that contains your database to go to the **Instance details** page.

4.  In the **Overview** tab, click the name of your database. The **Database details** page appears.

5.  Click **Spanner Studio** .

6.  Enter a DML statement. For example, the following statement adds a new row to the `Singers` table.

    ```
    INSERT Singers (SingerId, FirstName, LastName)
    VALUES (1, 'Marc', 'Richards')
    ```

7.  Click **Run query** . The Google Cloud console displays the result.

> **Note:** When you click **Run query** , the transaction commits automatically.

### Execute statements with the Google Cloud CLI

To execute DML statements, use the `gcloud spanner databases execute-sql` command. The following example adds a new row to the `Singers` table.

```
gcloud spanner databases execute-sql example-db --instance=test-instance \
    --sql="INSERT Singers (SingerId, FirstName, LastName) VALUES (1, 'Marc', 'Richards')"
```

### Modify data using the client library

To execute DML statements using the client library:

- Create a [read-write transaction](https://docs.cloud.google.com/spanner/docs/transactions#read-write_transactions) .
- Call the client library method for DML execution and pass in the DML statement.
- Use the return value of the DML execution method to get the number of rows inserted, updated, or deleted.

The following code example inserts a new row into the `Singers` table.

### C++

You use the `ExecuteDml()` function to execute a DML statement.

```
void DmlStandardInsert(google::cloud::spanner::Client client) {
  using ::google::cloud::StatusOr;
  namespace spanner = ::google::cloud::spanner;
  std::int64_t rows_inserted;
  auto commit_result = client.Commit(
      [&client, &rows_inserted](
          spanner::Transaction txn) -> StatusOr<spanner::Mutations> {
        auto insert = client.ExecuteDml(
            std::move(txn),
            spanner::SqlStatement(
                "INSERT INTO Singers (SingerId, FirstName, LastName)"
                "  VALUES (10, 'Virginia', 'Watson')"));
        if (!insert) return std::move(insert).status();
        rows_inserted = insert->RowsModified();
        return spanner::Mutations{};
      });
  if (!commit_result) throw std::move(commit_result).status();
  std::cout << "Rows inserted: " << rows_inserted;
  std::cout << "Insert was successful [spanner_dml_standard_insert]\n";
}
```

### C#

You use the `ExecuteNonQueryAsync()` method to execute a DML statement.

```
using Google.Cloud.Spanner.Data;
using System;
using System.Threading.Tasks;

public class InsertUsingDmlCoreAsyncSample
{
    public async Task<int> InsertUsingDmlCoreAsync(string projectId, string instanceId, string databaseId)
    {
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";

        using var connection = new SpannerConnection(connectionString);
        await connection.OpenAsync();

        using var cmd = connection.CreateDmlCommand("INSERT Singers (SingerId, FirstName, LastName) VALUES (10, 'Virginia', 'Watson')");
        int rowCount = await cmd.ExecuteNonQueryAsync();

        Console.WriteLine($"{rowCount} row(s) inserted...");
        return rowCount;
    }
}
```

### Go

You use the `Update()` method to execute a DML statement.

```
import (
    "context"
    "fmt"
    "io"

    "cloud.google.com/go/spanner"
)

func insertUsingDML(w io.Writer, db string) error {
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    _, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        stmt := spanner.Statement{
            SQL: `INSERT Singers (SingerId, FirstName, LastName)
                    VALUES (10, 'Virginia', 'Watson')`,
        }
        rowCount, err := txn.Update(ctx, stmt)
        if err != nil {
            return err
        }
        fmt.Fprintf(w, "%d record(s) inserted.\n", rowCount)
        return nil
    })
    return err
}
```

### Java

You use the `executeUpdate()` method to execute a DML statement.

```
static void insertUsingDml(DatabaseClient dbClient) {
  dbClient
      .readWriteTransaction()
      .run(transaction -> {
        String sql =
            "INSERT INTO Singers (SingerId, FirstName, LastName) "
                + " VALUES (10, 'Virginia', 'Watson')";
        long rowCount = transaction.executeUpdate(Statement.of(sql));
        System.out.printf("%d record inserted.\n", rowCount);
        return null;
      });
}
```

### Node.js

You use the `runUpdate()` method to execute a DML statement.

```
// Imports the Google Cloud client library
const {Spanner} = require('@google-cloud/spanner');

/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';
// const databaseId = 'my-database';

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});

// Gets a reference to a Cloud Spanner instance and database
const instance = spanner.instance(instanceId);
const database = instance.database(databaseId);

database.runTransaction(async (err, transaction) => {
  if (err) {
    console.error(err);
    return;
  }
  try {
    const [rowCount] = await transaction.runUpdate({
      sql: 'INSERT Singers (SingerId, FirstName, LastName) VALUES (10, @firstName, @lastName)',
      params: {
        firstName: 'Virginia',
        lastName: 'Watson',
      },
    });

    console.log(
      `Successfully inserted ${rowCount} record into the Singers table.`,
    );

    await transaction.commit();
  } catch (err) {
    console.error('ERROR:', err);
  } finally {
    // Close the database when finished.
    database.close();
  }
});
```

### PHP

You use the `executeUpdate()` method to execute a DML statement.

```
use Google\Cloud\Spanner\SpannerClient;
use Google\Cloud\Spanner\Transaction;

/**
 * Inserts sample data into the given database with a DML statement.
 *
 * The database and table must already exist and can be created using
 * `create_database`.
 * Example:
 * ```
 * insert_data($instanceId, $databaseId);
 * ```
 *
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 */
function insert_data_with_dml(string $instanceId, string $databaseId): void
{
    $spanner = new SpannerClient();
    $instance = $spanner->instance($instanceId);
    $database = $instance->database($databaseId);

    $database->runTransaction(function (Transaction $t) {
        $rowCount = $t->executeUpdate(
            'INSERT Singers (SingerId, FirstName, LastName) '
            . " VALUES (10, 'Virginia', 'Watson')");
        $t->commit();
        printf('Inserted %d row(s).' . PHP_EOL, $rowCount);
    });
}
```

### Python

You use the `execute_update()` method to execute a DML statement.

```
# instance_id = "your-spanner-instance"
# database_id = "your-spanner-db-id"

spanner_client = spanner.Client()
instance = spanner_client.instance(instance_id)
database = instance.database(database_id)

def insert_singers(transaction):
    row_ct = transaction.execute_update(
        "INSERT INTO Singers (SingerId, FirstName, LastName) "
        " VALUES (10, 'Virginia', 'Watson')"
    )

    print("{} record(s) inserted.".format(row_ct))

database.run_in_transaction(insert_singers)
```

### Ruby

You use the `execute_update()` method to execute a DML statement.

```
# project_id  = "Your Google Cloud project ID"
# instance_id = "Your Spanner instance ID"
# database_id = "Your Spanner database ID"

require "google/cloud/spanner"

spanner   = Google::Cloud::Spanner.new project: project_id
client    = spanner.client instance_id, database_id
row_count = 0

client.transaction do |transaction|
  row_count = transaction.execute_update(
    "INSERT INTO Singers (SingerId, FirstName, LastName) VALUES (10, 'Virginia', 'Watson')"
  )
end

puts "#{row_count} record inserted."
```

The following code example updates the `MarketingBudget` column of the `Albums` table based on a `WHERE` clause.

### C++

```
void DmlStandardUpdate(google::cloud::spanner::Client client) {
  using ::google::cloud::StatusOr;
  namespace spanner = ::google::cloud::spanner;
  auto commit_result = client.Commit(
      [&client](spanner::Transaction txn) -> StatusOr<spanner::Mutations> {
        auto update = client.ExecuteDml(
            std::move(txn),
            spanner::SqlStatement(
                "UPDATE Albums SET MarketingBudget = MarketingBudget * 2"
                "  WHERE SingerId = 1 AND AlbumId = 1"));
        if (!update) return std::move(update).status();
        return spanner::Mutations{};
      });
  if (!commit_result) throw std::move(commit_result).status();
  std::cout << "Update was successful [spanner_dml_standard_update]\n";
}
```

### C#

```
using Google.Cloud.Spanner.Data;
using System;
using System.Threading.Tasks;

public class UpdateUsingDmlCoreAsyncSample
{
    public async Task<int> UpdateUsingDmlCoreAsync(string projectId, string instanceId, string databaseId)
    {
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";

        using var connection = new SpannerConnection(connectionString);
        await connection.OpenAsync();

        using var cmd = connection.CreateDmlCommand("UPDATE Albums SET MarketingBudget = MarketingBudget * 2 WHERE SingerId = 1 and AlbumId = 1");
        int rowCount = await cmd.ExecuteNonQueryAsync();

        Console.WriteLine($"{rowCount} row(s) updated...");
        return rowCount;
    }
}
```

### Go

```
import (
    "context"
    "fmt"
    "io"

    "cloud.google.com/go/spanner"
)

func updateUsingDML(w io.Writer, db string) error {
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    _, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        stmt := spanner.Statement{
            SQL: `UPDATE Albums
                SET MarketingBudget = MarketingBudget * 2
                WHERE SingerId = 1 and AlbumId = 1`,
        }
        rowCount, err := txn.Update(ctx, stmt)
        if err != nil {
            return err
        }
        fmt.Fprintf(w, "%d record(s) updated.\n", rowCount)
        return nil
    })
    return err
}
```

### Java

```
static void updateUsingDml(DatabaseClient dbClient) {
  dbClient
      .readWriteTransaction()
      .run(transaction -> {
        String sql =
            "UPDATE Albums "
                + "SET MarketingBudget = MarketingBudget * 2 "
                + "WHERE SingerId = 1 and AlbumId = 1";
        long rowCount = transaction.executeUpdate(Statement.of(sql));
        System.out.printf("%d record updated.\n", rowCount);
        return null;
      });
}
```

### Node.js

```
// Imports the Google Cloud client library
const {Spanner} = require('@google-cloud/spanner');

/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';
// const databaseId = 'my-database';

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});

// Gets a reference to a Cloud Spanner instance and database
const instance = spanner.instance(instanceId);
const database = instance.database(databaseId);

database.runTransaction(async (err, transaction) => {
  if (err) {
    console.error(err);
    return;
  }
  try {
    const [rowCount] = await transaction.runUpdate({
      sql: `UPDATE Albums SET MarketingBudget = MarketingBudget * 2
        WHERE SingerId = 1 and AlbumId = 1`,
    });

    console.log(`Successfully updated ${rowCount} record.`);
    await transaction.commit();
  } catch (err) {
    console.error('ERROR:', err);
  } finally {
    // Close the database when finished.
    database.close();
  }
});
```

### PHP

```
use Google\Cloud\Spanner\SpannerClient;
use Google\Cloud\Spanner\Transaction;

/**
 * Updates sample data in the database with a DML statement.
 *
 * This requires the `MarketingBudget` column which must be created before
 * running this sample. You can add the column by running the `add_column`
 * sample or by running this DDL statement against your database:
 *
 *     ALTER TABLE Albums ADD COLUMN MarketingBudget INT64
 *
 * Example:
 * ```
 * update_data($instanceId, $databaseId);
 * ```
 *
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 */
function update_data_with_dml(string $instanceId, string $databaseId): void
{
    $spanner = new SpannerClient();
    $instance = $spanner->instance($instanceId);
    $database = $instance->database($databaseId);

    $database->runTransaction(function (Transaction $t) {
        $rowCount = $t->executeUpdate(
            'UPDATE Albums '
            . 'SET MarketingBudget = MarketingBudget * 2 '
            . 'WHERE SingerId = 1 and AlbumId = 1');
        $t->commit();
        printf('Updated %d row(s).' . PHP_EOL, $rowCount);
    });
}
```

### Python

```
# instance_id = "your-spanner-instance"
# database_id = "your-spanner-db-id"

spanner_client = spanner.Client()
instance = spanner_client.instance(instance_id)
database = instance.database(database_id)

def update_albums(transaction):
    row_ct = transaction.execute_update(
        "UPDATE Albums "
        "SET MarketingBudget = MarketingBudget * 2 "
        "WHERE SingerId = 1 and AlbumId = 1"
    )

    print("{} record(s) updated.".format(row_ct))

database.run_in_transaction(update_albums)
```

### Ruby

```
# project_id  = "Your Google Cloud project ID"
# instance_id = "Your Spanner instance ID"
# database_id = "Your Spanner database ID"

require "google/cloud/spanner"

spanner = Google::Cloud::Spanner.new project: project_id
client  = spanner.client instance_id, database_id
row_count = 0

client.transaction do |transaction|
  row_count = transaction.execute_update(
    "UPDATE Albums
     SET MarketingBudget = MarketingBudget * 2
     WHERE SingerId = 1 and AlbumId = 1"
  )
end

puts "#{row_count} record updated."
```

The following code example deletes all the rows in the `Singers` table where the `FirstName` column is `Alice` .

### C++

```
void DmlStandardDelete(google::cloud::spanner::Client client) {
  using ::google::cloud::StatusOr;
  namespace spanner = ::google::cloud::spanner;
  auto commit_result = client.Commit([&client](spanner::Transaction txn)
                                         -> StatusOr<spanner::Mutations> {
    auto dele = client.ExecuteDml(
        std::move(txn),
        spanner::SqlStatement("DELETE FROM Singers WHERE FirstName = 'Alice'"));
    if (!dele) return std::move(dele).status();
    return spanner::Mutations{};
  });
  if (!commit_result) throw std::move(commit_result).status();
  std::cout << "Delete was successful [spanner_dml_standard_delete]\n";
}
```

### C#

```
using Google.Cloud.Spanner.Data;
using System;
using System.Threading.Tasks;

public class DeleteUsingDmlCoreAsyncSample
{
    public async Task<int> DeleteUsingDmlCoreAsync(string projectId, string instanceId, string databaseId)
    {
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";

        using var connection = new SpannerConnection(connectionString);
        await connection.OpenAsync();

        using var cmd = connection.CreateDmlCommand("DELETE FROM Singers WHERE FirstName = 'Alice'");
        int rowCount = await cmd.ExecuteNonQueryAsync();

        Console.WriteLine($"{rowCount} row(s) deleted...");
        return rowCount;
    }
}
```

### Go

```
import (
    "context"
    "fmt"
    "io"

    "cloud.google.com/go/spanner"
)

func deleteUsingDML(w io.Writer, db string) error {
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    _, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        stmt := spanner.Statement{SQL: `DELETE FROM Singers WHERE FirstName = 'Alice'`}
        rowCount, err := txn.Update(ctx, stmt)
        if err != nil {
            return err
        }
        fmt.Fprintf(w, "%d record(s) deleted.\n", rowCount)
        return nil
    })
    return err
}
```

### Java

```
static void deleteUsingDml(DatabaseClient dbClient) {
  dbClient
      .readWriteTransaction()
      .run(transaction -> {
        String sql = "DELETE FROM Singers WHERE FirstName = 'Alice'";
        long rowCount = transaction.executeUpdate(Statement.of(sql));
        System.out.printf("%d record deleted.\n", rowCount);
        return null;
      });
}
```

### Node.js

```
// Imports the Google Cloud client library
const {Spanner} = require('@google-cloud/spanner');

/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';
// const databaseId = 'my-database';

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});

// Gets a reference to a Cloud Spanner instance and database
const instance = spanner.instance(instanceId);
const database = instance.database(databaseId);

database.runTransaction(async (err, transaction) => {
  if (err) {
    console.error(err);
    return;
  }
  try {
    const [rowCount] = await transaction.runUpdate({
      sql: "DELETE FROM Singers WHERE FirstName = 'Alice'",
    });

    console.log(`Successfully deleted ${rowCount} record.`);
    await transaction.commit();
  } catch (err) {
    console.error('ERROR:', err);
  } finally {
    // Close the database when finished.
    database.close();
  }
});
```

### PHP

```
use Google\Cloud\Spanner\SpannerClient;
use Google\Cloud\Spanner\Transaction;

/**
 * Deletes sample data in the database with a DML statement.
 *
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 */
function delete_data_with_dml(string $instanceId, string $databaseId): void
{
    $spanner = new SpannerClient();
    $instance = $spanner->instance($instanceId);
    $database = $instance->database($databaseId);

    $database->runTransaction(function (Transaction $t) {
        $rowCount = $t->executeUpdate(
            "DELETE FROM Singers WHERE FirstName = 'Alice'");
        $t->commit();
        printf('Deleted %d row(s).' . PHP_EOL, $rowCount);
    });
}
```

### Python

```
# instance_id = "your-spanner-instance"
# database_id = "your-spanner-db-id"

spanner_client = spanner.Client()
instance = spanner_client.instance(instance_id)
database = instance.database(database_id)

def delete_singers(transaction):
    row_ct = transaction.execute_update(
        "DELETE FROM Singers WHERE FirstName = 'Alice'"
    )

    print("{} record(s) deleted.".format(row_ct))

database.run_in_transaction(delete_singers)
```

### Ruby

```
# project_id  = "Your Google Cloud project ID"
# instance_id = "Your Spanner instance ID"
# database_id = "Your Spanner database ID"

require "google/cloud/spanner"

spanner = Google::Cloud::Spanner.new project: project_id
client  = spanner.client instance_id, database_id
row_count = 0

client.transaction do |transaction|
  row_count = transaction.execute_update(
    "DELETE FROM Singers WHERE FirstName = 'Alice'"
  )
end

puts "#{row_count} record deleted."
```

The following example, for GoogleSQL-dialect databases only, uses a [`STRUCT` with bound parameters](https://docs.cloud.google.com/spanner/docs/structs#using_struct_objects_as_bound_parameters_in_sql_queries) to update the `LastName` in rows filtered by `FirstName` and `LastName` .

### GoogleSQL

### C++

```
void DmlStructs(google::cloud::spanner::Client client) {
  namespace spanner = ::google::cloud::spanner;
  std::int64_t rows_modified = 0;
  auto commit_result =
      client.Commit([&client, &rows_modified](spanner::Transaction const& txn)
                        -> google::cloud::StatusOr<spanner::Mutations> {
        auto singer_info = std::make_tuple("Marc", "Richards");
        auto sql = spanner::SqlStatement(
            "UPDATE Singers SET FirstName = 'Keith' WHERE "
            "STRUCT<FirstName String, LastName String>(FirstName, LastName) "
            "= @name",
            {{"name", spanner::Value(std::move(singer_info))}});
        auto dml_result = client.ExecuteDml(txn, std::move(sql));
        if (!dml_result) return std::move(dml_result).status();
        rows_modified = dml_result->RowsModified();
        return spanner::Mutations{};
      });
  if (!commit_result) throw std::move(commit_result).status();
  std::cout << rows_modified
            << " update was successful [spanner_dml_structs]\n";
}
```

### C#

```
using Google.Cloud.Spanner.Data;
using System;
using System.Threading.Tasks;

public class UpdateUsingDmlWithStructCoreAsyncSample
{
    public async Task<int> UpdateUsingDmlWithStructCoreAsync(string projectId, string instanceId, string databaseId)
    {
        var nameStruct = new SpannerStruct
        {
            { "FirstName", SpannerDbType.String, "Timothy" },
            { "LastName", SpannerDbType.String, "Campbell" }
        };
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";

        using var connection = new SpannerConnection(connectionString);
        await connection.OpenAsync();

        using var cmd = connection.CreateDmlCommand("UPDATE Singers SET LastName = 'Grant' WHERE STRUCT<FirstName STRING, LastName STRING>(FirstName, LastName) = @name");
        cmd.Parameters.Add("name", nameStruct.GetSpannerDbType(), nameStruct);
        int rowCount = await cmd.ExecuteNonQueryAsync();

        Console.WriteLine($"{rowCount} row(s) updated...");
        return rowCount;
    }
}
```

### Go

```
import (
    "context"
    "fmt"
    "io"

    "cloud.google.com/go/spanner"
)

func updateUsingDMLStruct(w io.Writer, db string) error {
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    _, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        type name struct {
            FirstName string
            LastName  string
        }
        var singerInfo = name{"Timothy", "Campbell"}

        stmt := spanner.Statement{
            SQL: `Update Singers Set LastName = 'Grant'
                WHERE STRUCT<FirstName String, LastName String>(Firstname, LastName) = @name`,
            Params: map[string]interface{}{"name": singerInfo},
        }
        rowCount, err := txn.Update(ctx, stmt)
        if err != nil {
            return err
        }
        fmt.Fprintf(w, "%d record(s) inserted.\n", rowCount)
        return nil
    })
    return err
}
```

### Java

```
static void updateUsingDmlWithStruct(DatabaseClient dbClient) {
  Struct name =
      Struct.newBuilder().set("FirstName").to("Timothy").set("LastName").to("Campbell").build();
  Statement s =
      Statement.newBuilder(
              "UPDATE Singers SET LastName = 'Grant' "
                  + "WHERE STRUCT<FirstName STRING, LastName STRING>(FirstName, LastName) "
                  + "= @name")
          .bind("name")
          .to(name)
          .build();
  dbClient
      .readWriteTransaction()
      .run(transaction -> {
        long rowCount = transaction.executeUpdate(s);
        System.out.printf("%d record updated.\n", rowCount);
        return null;
      });
}
```

### Node.js

```
// Imports the Google Cloud client library
const {Spanner} = require('@google-cloud/spanner');

const nameStruct = Spanner.struct({
  FirstName: 'Timothy',
  LastName: 'Campbell',
});

/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';
// const databaseId = 'my-database';

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});

// Gets a reference to a Cloud Spanner instance and database
const instance = spanner.instance(instanceId);
const database = instance.database(databaseId);

database.runTransaction(async (err, transaction) => {
  if (err) {
    console.error(err);
    return;
  }
  try {
    const [rowCount] = await transaction.runUpdate({
      sql: `UPDATE Singers SET LastName = 'Grant'
      WHERE STRUCT<FirstName STRING, LastName STRING>(FirstName, LastName) = @name`,
      params: {
        name: nameStruct,
      },
    });

    console.log(`Successfully updated ${rowCount} record.`);
    await transaction.commit();
  } catch (err) {
    console.error('ERROR:', err);
  } finally {
    // Close the database when finished.
    database.close();
  }
});
```

### PHP

```
use Google\Cloud\Spanner\SpannerClient;
use Google\Cloud\Spanner\Database;
use Google\Cloud\Spanner\Transaction;
use Google\Cloud\Spanner\StructType;
use Google\Cloud\Spanner\StructValue;

/**
 * Update data with a DML statement using Structs.
 *
 * The database and table must already exist and can be created using
 * `create_database`.
 * Example:
 * ```
 * insert_data($instanceId, $databaseId);
 * ```
 *
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 */
function update_data_with_dml_structs(string $instanceId, string $databaseId): void
{
    $spanner = new SpannerClient();
    $instance = $spanner->instance($instanceId);
    $database = $instance->database($databaseId);

    $database->runTransaction(function (Transaction $t) {
        $nameValue = (new StructValue)
            ->add('FirstName', 'Timothy')
            ->add('LastName', 'Campbell');
        $nameType = (new StructType)
            ->add('FirstName', Database::TYPE_STRING)
            ->add('LastName', Database::TYPE_STRING);

        $rowCount = $t->executeUpdate(
            "UPDATE Singers SET LastName = 'Grant' "
             . 'WHERE STRUCT<FirstName STRING, LastName STRING>(FirstName, LastName) '
             . '= @name',
            [
                'parameters' => [
                    'name' => $nameValue
                ],
                'types' => [
                    'name' => $nameType
                ]
            ]);
        $t->commit();
        printf('Updated %d row(s).' . PHP_EOL, $rowCount);
    });
}
```

### Python

```
# instance_id = "your-spanner-instance"
# database_id = "your-spanner-db-id"

spanner_client = spanner.Client()
instance = spanner_client.instance(instance_id)
database = instance.database(database_id)

record_type = param_types.Struct(
    [
        param_types.StructField("FirstName", param_types.STRING),
        param_types.StructField("LastName", param_types.STRING),
    ]
)
record_value = ("Timothy", "Campbell")

def write_with_struct(transaction):
    row_ct = transaction.execute_update(
        "UPDATE Singers SET LastName = 'Grant' "
        "WHERE STRUCT<FirstName STRING, LastName STRING>"
        "(FirstName, LastName) = @name",
        params={"name": record_value},
        param_types={"name": record_type},
    )
    print("{} record(s) updated.".format(row_ct))

database.run_in_transaction(write_with_struct)
```

### Ruby

```
# project_id  = "Your Google Cloud project ID"
# instance_id = "Your Spanner instance ID"
# database_id = "Your Spanner database ID"

require "google/cloud/spanner"

spanner = Google::Cloud::Spanner.new project: project_id
client  = spanner.client instance_id, database_id
row_count = 0
name_struct = { FirstName: "Timothy", LastName: "Campbell" }

client.transaction do |transaction|
  row_count = transaction.execute_update(
    "UPDATE Singers SET LastName = 'Grant'
     WHERE STRUCT<FirstName STRING, LastName STRING>(FirstName, LastName) = @name",
    params: { name: name_struct }
  )
end

puts "#{row_count} record updated."
```

### Modify data with the returning DML statements

The [`THEN RETURN` clause](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax#insert-and-then-return) (GoogleSQL-dialect databases) or [`RETURNING` clause](https://docs.cloud.google.com/spanner/docs/reference/postgresql/dml-syntax#insert-returning) (PostgreSQL-dialect databases) is intended for scenarios where you want to fetch data from modified rows. This is especially useful when you want to view unspecified values in the DML statements, default values, or generated columns.

To execute returning DML statements using the client library:

- Create a [read-write transaction](https://docs.cloud.google.com/spanner/docs/transactions#read-write_transactions) .
- Call the client library method for query execution and pass in the returning DML statement to obtain results.

The following code example inserts a new row into the `Singers` table, and it returns the generated column FullName of the inserted records.

### GoogleSQL

### C++

```
void InsertUsingDmlReturning(google::cloud::spanner::Client client) {
  // Insert records into SINGERS table and return the generated column
  // FullName of the inserted records using `THEN RETURN FullName`.
  auto commit = client.Commit(
      [&client](google::cloud::spanner::Transaction txn)
          -> google::cloud::StatusOr<google::cloud::spanner::Mutations> {
        auto sql = google::cloud::spanner::SqlStatement(R"""(
            INSERT INTO Singers (SingerId, FirstName, LastName)
              VALUES (12, 'Melissa', 'Garcia'),
                     (13, 'Russell', 'Morales'),
                     (14, 'Jacqueline', 'Long'),
                     (15, 'Dylan', 'Shaw')
              THEN RETURN FullName
        )""");
        using RowType = std::tuple<std::string>;
        auto rows = client.ExecuteQuery(std::move(txn), std::move(sql));
        // Note: This mutator might be re-run, or its effects discarded, so
        // changing non-transactional state (e.g., by producing output) is,
        // in general, not something to be imitated.
        for (auto& row : google::cloud::spanner::StreamOf<RowType>(rows)) {
          if (!row) return std::move(row).status();
          std::cout << "FullName: " << std::get<0>(*row) << "\n";
        }
        std::cout << "Inserted row(s) count: " << rows.RowsModified() << "\n";
        return google::cloud::spanner::Mutations{};
      });
  if (!commit) throw std::move(commit).status();
}
```

### C#

```
using Google.Cloud.Spanner.Data;
using System;
using System.Collections.Generic;
using System.Threading.Tasks;

public class InsertUsingDmlReturningAsyncSample
{
    public async Task<List<string>> InsertUsingDmlReturningAsync(string projectId, string instanceId, string databaseId)
    {
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";

        using var connection = new SpannerConnection(connectionString);
        await connection.OpenAsync();

        // Insert records into the SINGERS table and return the
        // generated column FullName of the inserted records using
        // 'THEN RETURN FullName'.
        // It is also possible to return all columns of all the
        // inserted records by using 'THEN RETURN *'.
        using var cmd = connection.CreateDmlCommand(
            @"INSERT INTO Singers(SingerId, FirstName, LastName) VALUES
            (6, 'Melissa', 'Garcia'), 
            (7, 'Russell', 'Morales'), 
            (8, 'Jacqueline', 'Long'), 
            (9, 'Dylan', 'Shaw') THEN RETURN FullName");

        var reader = await cmd.ExecuteReaderAsync();
        var insertedSingerNames = new List<string>();
        while (await reader.ReadAsync())
        {
            insertedSingerNames.Add(reader.GetFieldValue<string>("FullName"));
        }

        Console.WriteLine($"{insertedSingerNames.Count} row(s) inserted...");
        return insertedSingerNames;
    }
}
```

### Go

```
import (
    "context"
    "fmt"
    "io"

    "cloud.google.com/go/spanner"
    "google.golang.org/api/iterator"
)

func insertUsingDMLReturning(w io.Writer, db string) error {
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    // Insert records into the SINGERS table and returns the
    // generated column FullName of the inserted records using
    // 'THEN RETURN FullName'.
    // It is also possible to return all columns of all the
    // inserted records by using 'THEN RETURN *'.
    _, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        stmt := spanner.Statement{
            SQL: `INSERT INTO Singers (SingerId, FirstName, LastName)
                    VALUES (21, 'Melissa', 'Garcia'),
                           (22, 'Russell', 'Morales'),
                           (23, 'Jacqueline', 'Long'),
                           (24, 'Dylan', 'Shaw')
                    THEN RETURN FullName`,
        }
        iter := txn.Query(ctx, stmt)
        defer iter.Stop()
        for {
            row, err := iter.Next()
            if err == iterator.Done {
                break
            }
            if err != nil {
                return err
            }
            var fullName string
            if err := row.Columns(&fullName); err != nil {
                return err
            }
            fmt.Fprintf(w, "%s\n", fullName)
        }
        fmt.Fprintf(w, "%d record(s) inserted.\n", iter.RowCount)
        return nil
    })
    return err
}
```

### Java

```
import com.google.cloud.spanner.DatabaseClient;
import com.google.cloud.spanner.DatabaseId;
import com.google.cloud.spanner.ResultSet;
import com.google.cloud.spanner.Spanner;
import com.google.cloud.spanner.SpannerOptions;
import com.google.cloud.spanner.Statement;

public class InsertUsingDmlReturningSample {

  static void insertUsingDmlReturning() {
    // TODO(developer): Replace these variables before running the sample.
    final String projectId = "my-project";
    final String instanceId = "my-instance";
    final String databaseId = "my-database";
    insertUsingDmlReturning(projectId, instanceId, databaseId);
  }

  static void insertUsingDmlReturning(String projectId, String instanceId, String databaseId) {
    try (Spanner spanner =
        SpannerOptions.newBuilder()
            .setProjectId(projectId)
            .build()
            .getService()) {
      final DatabaseClient dbClient =
          spanner.getDatabaseClient(DatabaseId.of(projectId, instanceId, databaseId));
      // Insert records into the SINGERS table and returns the
      // generated column FullName of the inserted records using
      // ‘THEN RETURN FullName’.
      // It is also possible to return all columns of all the
      // inserted records by using ‘THEN RETURN *’.
      dbClient
          .readWriteTransaction()
          .run(
              transaction -> {
                String sql =
                    "INSERT INTO Singers (SingerId, FirstName, LastName) VALUES "
                        + "(12, 'Melissa', 'Garcia'), "
                        + "(13, 'Russell', 'Morales'), "
                        + "(14, 'Jacqueline', 'Long'), "
                        + "(15, 'Dylan', 'Shaw') THEN RETURN FullName";

                // readWriteTransaction.executeQuery(..) API should be used for executing
                // DML statements with RETURNING clause.
                try (ResultSet resultSet = transaction.executeQuery(Statement.of(sql))) {
                  while (resultSet.next()) {
                    System.out.println(resultSet.getString(0));
                  }
                  System.out.printf(
                      "Inserted row(s) count: %d\n", resultSet.getStats().getRowCountExact());
                }
                return null;
              });
    }
  }
}
```

### Node.js

```
// Imports the Google Cloud client library.
const {Spanner} = require('@google-cloud/spanner');

/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';
// const databaseId = 'my-database';

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});

function insertUsingDmlReturning(instanceId, databaseId) {
  // Gets a reference to a Cloud Spanner instance and database.
  const instance = spanner.instance(instanceId);
  const database = instance.database(databaseId);

  database.runTransaction(async (err, transaction) => {
    if (err) {
      console.error(err);
      return;
    }
    try {
      const [rows, stats] = await transaction.run({
        sql: 'INSERT Singers (SingerId, FirstName, LastName) VALUES (@id, @firstName, @lastName) THEN RETURN FullName',
        params: {
          id: 18,
          firstName: 'Virginia',
          lastName: 'Watson',
        },
      });

      const rowCount = Math.floor(stats[stats.rowCount]);
      console.log(
        `Successfully inserted ${rowCount} record into the Singers table.`,
      );
      rows.forEach(row => {
        console.log(row.toJSON().FullName);
      });

      await transaction.commit();
    } catch (err) {
      console.error('ERROR:', err);
    } finally {
      // Close the database when finished.
      database.close();
    }
  });
}
insertUsingDmlReturning(instanceId, databaseId);
```

### PHP

```
use Google\Cloud\Spanner\SpannerClient;

/**
 * Inserts sample data into the given database using DML returning.
 *
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 */
function insert_dml_returning(string $instanceId, string $databaseId): void
{
    $spanner = new SpannerClient();
    $instance = $spanner->instance($instanceId);
    $database = $instance->database($databaseId);

    // Insert records into SINGERS table and returns the generated column
    // FullName of the inserted records using ‘THEN RETURN FullName’. It is also
    // possible to return all columns of all the inserted records by using
    // ‘THEN RETURN *’.

    $sql = 'INSERT INTO Singers (SingerId, FirstName, LastName) '
        . "VALUES (12, 'Melissa', 'Garcia'), "
        . "(13, 'Russell', 'Morales'), "
        . "(14, 'Jacqueline', 'Long'), "
        . "(15, 'Dylan', 'Shaw') "
        . 'THEN RETURN FullName';

    $transaction = $database->transaction();
    $result = $transaction->execute($sql);
    foreach ($result->rows() as $row) {
        printf(
            '%s inserted.' . PHP_EOL,
            $row['FullName'],
        );
    }
    printf(
        'Inserted row(s) count: %d' . PHP_EOL,
        $result->stats()['rowCountExact']
    );
    $transaction->commit();
}
```

### Python

```
# instance_id = "your-spanner-instance"
# database_id = "your-spanner-db-id"

spanner_client = spanner.Client()
instance = spanner_client.instance(instance_id)
database = instance.database(database_id)

# Insert records into the SINGERS table and returns the
# generated column FullName of the inserted records using
# 'THEN RETURN FullName'.
# It is also possible to return all columns of all the
# inserted records by using 'THEN RETURN *'.
def insert_singers(transaction):
    results = transaction.execute_sql(
        "INSERT INTO Singers (SingerId, FirstName, LastName) VALUES "
        "(21, 'Luann', 'Chizoba'), "
        "(22, 'Denis', 'Patricio'), "
        "(23, 'Felxi', 'Ronan'), "
        "(24, 'Dominik', 'Martyna') "
        "THEN RETURN FullName"
    )
    for result in results:
        print("FullName: {}".format(*result))
    print("{} record(s) inserted.".format(results.stats.row_count_exact))

database.run_in_transaction(insert_singers)
```

### Ruby

```
require "google/cloud/spanner"

##
# This is a snippet for showcasing how to use DML return feature with insert
# operation.
#
# @param project_id  [String] The ID of the Google Cloud project.
# @param instance_id [String] The ID of the spanner instance.
# @param database_id [String] The ID of the database.
#
def spanner_insert_dml_returning project_id:, instance_id:, database_id:
  spanner = Google::Cloud::Spanner.new project: project_id
  client = spanner.client instance_id, database_id

  client.transaction do |transaction|
    # Insert records into the SINGERS table and returns the generated column
    # FullName of the inserted records using ‘THEN RETURN FullName’.
    # It is also possible to return all columns of all the inserted records
    # by using ‘THEN RETURN *’.
    results = transaction.execute_query "INSERT INTO Singers (SingerId, FirstName, LastName)
                                         VALUES (12, 'Melissa', 'Garcia'), (13, 'Russell', 'Morales'), (14, 'Jacqueline', 'Long'), (15, 'Dylan', 'Shaw')
                                         THEN RETURN FullName"
    results.rows.each do |row|
      puts "Inserted singers with FullName: #{row[:FullName]}"
    end
    puts "Inserted row(s) count: #{results.row_count}"
  end
end
```

### PostgreSQL

### C++

```
void InsertUsingDmlReturning(google::cloud::spanner::Client client) {
  // Insert records into SINGERS table and return the generated column
  // FullName of the inserted records using `RETURNING FullName`.
  auto commit = client.Commit(
      [&client](google::cloud::spanner::Transaction txn)
          -> google::cloud::StatusOr<google::cloud::spanner::Mutations> {
        auto sql = google::cloud::spanner::SqlStatement(R"""(
            INSERT INTO Singers (SingerId, FirstName, LastName)
                VALUES (12, 'Melissa', 'Garcia'),
                       (13, 'Russell', 'Morales'),
                       (14, 'Jacqueline', 'Long'),
                       (15, 'Dylan', 'Shaw')
                RETURNING FullName
        )""");
        using RowType = std::tuple<std::string>;
        auto rows = client.ExecuteQuery(std::move(txn), std::move(sql));
        for (auto& row : google::cloud::spanner::StreamOf<RowType>(rows)) {
          if (!row) return std::move(row).status();
          std::cout << "FullName: " << std::get<0>(*row) << "\n";
        }
        std::cout << "Inserted row(s) count: " << rows.RowsModified() << "\n";
        return google::cloud::spanner::Mutations{};
      });
  if (!commit) throw std::move(commit).status();
}
```

### C#

```
using Google.Cloud.Spanner.Data;
using System;
using System.Collections.Generic;
using System.Threading.Tasks;

public class InsertUsingDmlReturningAsyncPostgresSample
{
    public async Task<List<string>> InsertUsingDmlReturningAsyncPostgres(string projectId, string instanceId, string databaseId)
    {
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";

        using var connection = new SpannerConnection(connectionString);
        await connection.OpenAsync();

        // Insert records into SINGERS table and return the
        // generated column FullName of the inserted records
        // using 'RETURNING FullName'.
        // It is also possible to return all columns of all the
        // inserted records by using 'RETURNING *'.
        using var cmd = connection.CreateDmlCommand(
            @"INSERT INTO Singers(SingerId, FirstName, LastName) VALUES
            (6, 'Melissa', 'Garcia'), 
            (7, 'Russell', 'Morales'), 
            (8, 'Jacqueline', 'Long'), 
            (9, 'Dylan', 'Shaw') RETURNING FullName");

        var reader = await cmd.ExecuteReaderAsync();
        var insertedSingerNames = new List<string>();
        while (await reader.ReadAsync())
        {
            insertedSingerNames.Add(reader.GetFieldValue<string>("fullname"));
        }

        Console.WriteLine($"{insertedSingerNames.Count} row(s) inserted...");
        return insertedSingerNames;
    }
}
```

### Go

```
import (
    "context"
    "fmt"
    "io"

    "cloud.google.com/go/spanner"
    "google.golang.org/api/iterator"
)

func pgInsertUsingDMLReturning(w io.Writer, db string) error {
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    // Insert records into the SINGERS table and returns the
    // generated column FullName of the inserted records using
    // 'RETURNING FullName'.
    // It is also possible to return all columns of all the
    // inserted records by using 'RETURNING *'.
    _, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        stmt := spanner.Statement{
            SQL: `INSERT INTO Singers (SingerId, FirstName, LastName)
                    VALUES (21, 'Melissa', 'Garcia'),
                           (22, 'Russell', 'Morales'),
                           (23, 'Jacqueline', 'Long'),
                           (24, 'Dylan', 'Shaw')
                    RETURNING FullName`,
        }
        iter := txn.Query(ctx, stmt)
        defer iter.Stop()
        for {
            row, err := iter.Next()
            if err == iterator.Done {
                break
            }
            if err != nil {
                return err
            }
            var fullName string
            if err := row.Columns(&fullName); err != nil {
                return err
            }
            fmt.Fprintf(w, "%s\n", fullName)
        }
        fmt.Fprintf(w, "%d record(s) inserted.\n", iter.RowCount)
        return nil
    })
    return err
}
```

### Java

```
import com.google.cloud.spanner.DatabaseClient;
import com.google.cloud.spanner.DatabaseId;
import com.google.cloud.spanner.ResultSet;
import com.google.cloud.spanner.Spanner;
import com.google.cloud.spanner.SpannerOptions;
import com.google.cloud.spanner.Statement;

public class PgInsertUsingDmlReturningSample {

  static void insertUsingDmlReturning() {
    // TODO(developer): Replace these variables before running the sample.
    final String projectId = "my-project";
    final String instanceId = "my-instance";
    final String databaseId = "my-database";
    insertUsingDmlReturning(projectId, instanceId, databaseId);
  }

  static void insertUsingDmlReturning(String projectId, String instanceId, String databaseId) {
    try (Spanner spanner =
        SpannerOptions.newBuilder()
            .setProjectId(projectId)
            .build()
            .getService()) {
      final DatabaseClient dbClient =
          spanner.getDatabaseClient(DatabaseId.of(projectId, instanceId, databaseId));
      // Insert records into SINGERS table and returns the
      // generated column FullName of the inserted records
      // using ‘RETURNING FullName’.
      // It is also possible to return all columns of all the
      // inserted records by using ‘RETURNING *’.
      dbClient
          .readWriteTransaction()
          .run(
              transaction -> {
                String sql =
                    "INSERT INTO Singers (SingerId, FirstName, LastName) VALUES "
                        + "(12, 'Melissa', 'Garcia'), "
                        + "(13, 'Russell', 'Morales'), "
                        + "(14, 'Jacqueline', 'Long'), "
                        + "(15, 'Dylan', 'Shaw') RETURNING FullName";

                // readWriteTransaction.executeQuery(..) API should be used for executing
                // DML statements with RETURNING clause.
                try (ResultSet resultSet = transaction.executeQuery(Statement.of(sql))) {
                  while (resultSet.next()) {
                    System.out.println(resultSet.getString(0));
                  }
                  System.out.printf(
                      "Inserted row(s) count: %d\n", resultSet.getStats().getRowCountExact());
                }
                return null;
              });
    }
  }
}
```

### Node.js

```
// Imports the Google Cloud client library.
const {Spanner} = require('@google-cloud/spanner');

/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';
// const databaseId = 'my-database';

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});

function pgInsertUsingDmlReturning(instanceId, databaseId) {
  // Gets a reference to a Cloud Spanner instance and database.
  const instance = spanner.instance(instanceId);
  const database = instance.database(databaseId);

  database.runTransaction(async (err, transaction) => {
    if (err) {
      console.error(err);
      return;
    }
    try {
      const [rows, stats] = await transaction.run({
        sql: 'INSERT Into Singers (SingerId, FirstName, LastName) VALUES ($1, $2, $3) RETURNING FullName',
        params: {
          p1: 18,
          p2: 'Virginia',
          p3: 'Watson',
        },
      });

      const rowCount = Math.floor(stats[stats.rowCount]);
      console.log(
        `Successfully inserted ${rowCount} record into the Singers table.`,
      );
      rows.forEach(row => {
        console.log(row.toJSON().fullname);
      });

      await transaction.commit();
    } catch (err) {
      console.error('ERROR:', err);
    } finally {
      // Close the database when finished.
      database.close();
    }
  });
}
pgInsertUsingDmlReturning(instanceId, databaseId);
```

### PHP

```
use Google\Cloud\Spanner\SpannerClient;

/**
 * Inserts sample data into the given postgresql database using DML returning.
 *
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 */
function pg_insert_dml_returning(string $instanceId, string $databaseId): void
{
    $spanner = new SpannerClient();
    $instance = $spanner->instance($instanceId);
    $database = $instance->database($databaseId);

    // Insert records into SINGERS table and returns the generated column
    // FullName of the inserted records using ‘RETURNING FullName’. It is also
    // possible to return all columns of all the inserted records by using
    // ‘RETURNING *’.

    $sql = 'INSERT INTO Singers (Singerid, FirstName, LastName) '
      . "VALUES (12, 'Melissa', 'Garcia'), "
      . "(13, 'Russell', 'Morales'), "
      . "(14, 'Jacqueline', 'Long'), "
      . "(15, 'Dylan', 'Shaw') "
      . 'RETURNING FullName';

    $transaction = $database->transaction();
    $result = $transaction->execute($sql);
    foreach ($result->rows() as $row) {
        printf(
            '%s inserted.' . PHP_EOL,
            $row['fullname'],
        );
    }
    printf(
        'Inserted row(s) count: %d' . PHP_EOL,
        $result->stats()['rowCountExact']
    );
    $transaction->commit();
}
```

### Python

```
# instance_id = "your-spanner-instance"
# database_id = "your-spanner-db-id"

spanner_client = spanner.Client()
instance = spanner_client.instance(instance_id)
database = instance.database(database_id)

# Insert records into the SINGERS table and returns the
# generated column FullName of the inserted records using
# 'RETURNING FullName'.
# It is also possible to return all columns of all the
# inserted records by using 'RETURNING *'.
def insert_singers(transaction):
    results = transaction.execute_sql(
        "INSERT INTO Singers (SingerId, FirstName, LastName) VALUES "
        "(21, 'Luann', 'Chizoba'), "
        "(22, 'Denis', 'Patricio'), "
        "(23, 'Felxi', 'Ronan'), "
        "(24, 'Dominik', 'Martyna') "
        "RETURNING FullName"
    )
    for result in results:
        print("FullName: {}".format(*result))
    print("{} record(s) inserted.".format(results.stats.row_count_exact))

database.run_in_transaction(insert_singers)
```

### Ruby

```
require "google/cloud/spanner"

##
# This is a snippet for showcasing how to use DML return feature with insert
# operation in PostgreSql.
#
# @param project_id  [String] The ID of the Google Cloud project.
# @param instance_id [String] The ID of the spanner instance.
# @param database_id [String] The ID of the database.
#
def spanner_postgresql_insert_dml_returning project_id:, instance_id:, database_id:
  spanner = Google::Cloud::Spanner.new project: project_id
  client = spanner.client instance_id, database_id

  client.transaction do |transaction|
    # Insert records into SINGERS table and returns the generated column
    # FullName of the inserted records using ‘RETURNING FullName’.
    # It is also possible to return all columns of all the inserted
    # records by using ‘RETURNING *’.
    results = transaction.execute_query "INSERT INTO Singers (SingerId, FirstName, LastName)
                                         VALUES (12, 'Melissa', 'Garcia'), (13, 'Russell', 'Morales'), (14, 'Jacqueline', 'Long'), (15, 'Dylan', 'Shaw')
                                         RETURNING FullName"
    results.rows.each do |row|
      puts "Inserted singers with FullName: #{row[:fullname]}"
    end
    puts "Inserted row(s) count: #{results.row_count}"
  end
end
```

The following code example updates the `MarketingBudget` column of the `Albums` table based on a `WHERE` clause, and it returns the modified `MarketingBudget` column of the updated records.

### GoogleSQL

### C++

```
void UpdateUsingDmlReturning(google::cloud::spanner::Client client) {
  // Update MarketingBudget column for records satisfying a particular
  // condition and return the modified MarketingBudget column of the
  // updated records using `THEN RETURN MarketingBudget`.
  auto commit = client.Commit(
      [&client](google::cloud::spanner::Transaction txn)
          -> google::cloud::StatusOr<google::cloud::spanner::Mutations> {
        auto sql = google::cloud::spanner::SqlStatement(R"""(
            UPDATE Albums SET MarketingBudget = MarketingBudget * 2
              WHERE SingerId = 1 AND AlbumId = 1
              THEN RETURN MarketingBudget
        )""");
        using RowType = std::tuple<std::optional<std::int64_t>>;
        auto rows = client.ExecuteQuery(std::move(txn), std::move(sql));
        // Note: This mutator might be re-run, or its effects discarded, so
        // changing non-transactional state (e.g., by producing output) is,
        // in general, not something to be imitated.
        for (auto& row : google::cloud::spanner::StreamOf<RowType>(rows)) {
          if (!row) return std::move(row).status();
          std::cout << "MarketingBudget: ";
          if (std::get<0>(*row).has_value()) {
            std::cout << *std::get<0>(*row);
          } else {
            std::cout << "NULL";
          }
          std::cout << "\n";
        }
        std::cout << "Updated row(s) count: " << rows.RowsModified() << "\n";
        return google::cloud::spanner::Mutations{};
      });
  if (!commit) throw std::move(commit).status();
}
```

### C#

```
using Google.Cloud.Spanner.Data;
using System;
using System.Collections.Generic;
using System.Threading.Tasks;

public class UpdateUsingDmlReturningAsyncSample
{
    public async Task<List<long>> UpdateUsingDmlReturningAsync(string projectId, string instanceId, string databaseId)
    {
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";

        using var connection = new SpannerConnection(connectionString);
        await connection.OpenAsync();

        // Update MarketingBudget column for records satisfying
        // a particular condition and return the modified
        // MarketingBudget column of the updated records using
        // 'THEN RETURN MarketingBudget'.
        // It is also possible to return all columns of all the
        // updated records by using 'THEN RETURN *'.
        using var cmd = connection.CreateDmlCommand("UPDATE Albums SET MarketingBudget = MarketingBudget * 2 WHERE SingerId = 1 and AlbumId = 1 THEN RETURN MarketingBudget");
        var reader = await cmd.ExecuteReaderAsync();
        var updatedMarketingBudgets = new List<long>();
        while (await reader.ReadAsync())
        {
            updatedMarketingBudgets.Add(reader.GetFieldValue<long>("MarketingBudget"));
        }

        Console.WriteLine($"{updatedMarketingBudgets.Count} row(s) updated...");
        return updatedMarketingBudgets;
    }
}
```

### Go

```
import (
    "context"
    "fmt"
    "io"

    "cloud.google.com/go/spanner"
    "google.golang.org/api/iterator"
)

func updateUsingDMLReturning(w io.Writer, db string) error {
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    // Update MarketingBudget column for records satisfying
    // a particular condition and returns the modified
    // MarketingBudget column of the updated records using
    // 'THEN RETURN MarketingBudget'.
    // It is also possible to return all columns of all the
    // updated records by using 'THEN RETURN *'.
    _, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        stmt := spanner.Statement{
            SQL: `UPDATE Albums
                SET MarketingBudget = MarketingBudget * 2
                WHERE SingerId = 1 and AlbumId = 1
                THEN RETURN MarketingBudget`,
        }
        iter := txn.Query(ctx, stmt)
        defer iter.Stop()
        for {
            row, err := iter.Next()
            if err == iterator.Done {
                break
            }
            if err != nil {
                return err
            }
            var marketingBudget int64
            if err := row.Columns(&marketingBudget); err != nil {
                return err
            }
            fmt.Fprintf(w, "%d\n", marketingBudget)
        }
        fmt.Fprintf(w, "%d record(s) updated.\n", iter.RowCount)
        return nil
    })
    return err
}
```

### Java

```
import com.google.cloud.spanner.DatabaseClient;
import com.google.cloud.spanner.DatabaseId;
import com.google.cloud.spanner.ResultSet;
import com.google.cloud.spanner.Spanner;
import com.google.cloud.spanner.SpannerOptions;
import com.google.cloud.spanner.Statement;

public class UpdateUsingDmlReturningSample {

  static void updateUsingDmlReturning() {
    // TODO(developer): Replace these variables before running the sample.
    final String projectId = "my-project";
    final String instanceId = "my-instance";
    final String databaseId = "my-database";
    updateUsingDmlReturning(projectId, instanceId, databaseId);
  }

  static void updateUsingDmlReturning(String projectId, String instanceId, String databaseId) {
    try (Spanner spanner =
        SpannerOptions.newBuilder()
            .setProjectId(projectId)
            .build()
            .getService()) {
      final DatabaseClient dbClient =
          spanner.getDatabaseClient(DatabaseId.of(projectId, instanceId, databaseId));
      // Update MarketingBudget column for records satisfying
      // a particular condition and returns the modified
      // MarketingBudget column of the updated records using
      // ‘THEN RETURN MarketingBudget’.
      // It is also possible to return all columns of all the
      // updated records by using ‘THEN RETURN *’.
      dbClient
          .readWriteTransaction()
          .run(
              transaction -> {
                String sql =
                    "UPDATE Albums "
                        + "SET MarketingBudget = MarketingBudget * 2 "
                        + "WHERE SingerId = 1 and AlbumId = 1 "
                        + "THEN RETURN MarketingBudget";

                // readWriteTransaction.executeQuery(..) API should be used for executing
                // DML statements with RETURNING clause.
                try (ResultSet resultSet = transaction.executeQuery(Statement.of(sql))) {
                  while (resultSet.next()) {
                    System.out.printf("%d\n", resultSet.getLong(0));
                  }
                  System.out.printf(
                      "Updated row(s) count: %d\n", resultSet.getStats().getRowCountExact());
                }
                return null;
              });
    }
  }
}
```

### Node.js

```
// Imports the Google Cloud client library.
const {Spanner} = require('@google-cloud/spanner');

/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';
// const databaseId = 'my-database';

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});

function updateUsingDmlReturning(instanceId, databaseId) {
  // Gets a reference to a Cloud Spanner instance and database.
  const instance = spanner.instance(instanceId);
  const database = instance.database(databaseId);

  database.runTransaction(async (err, transaction) => {
    if (err) {
      console.error(err);
      return;
    }
    try {
      const [rows, stats] = await transaction.run({
        sql: 'UPDATE Albums SET MarketingBudget = 2000000 WHERE SingerId = 1 and AlbumId = 1 THEN RETURN MarketingBudget',
      });

      const rowCount = Math.floor(stats[stats.rowCount]);
      console.log(
        `Successfully updated ${rowCount} record into the Albums table.`,
      );
      rows.forEach(row => {
        console.log(row.toJSON().MarketingBudget);
      });

      await transaction.commit();
    } catch (err) {
      console.error('ERROR:', err);
    } finally {
      // Close the database when finished.
      database.close();
    }
  });
}
updateUsingDmlReturning(instanceId, databaseId);
```

### PHP

```
use Google\Cloud\Spanner\SpannerClient;

/**
 * Update the given database using DML returning.
 *
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 */
function update_dml_returning(string $instanceId, string $databaseId): void
{
    $spanner = new SpannerClient();
    $instance = $spanner->instance($instanceId);
    $database = $instance->database($databaseId);

    $transaction = $database->transaction();

    // Update MarketingBudget column for records satisfying a particular
    // condition and returns the modified MarketingBudget column of the updated
    // records using ‘THEN RETURN MarketingBudget’. It is also possible to return
    // all columns of all the updated records by using ‘THEN RETURN *’.

    $result = $transaction->execute(
        'UPDATE Albums '
        . 'SET MarketingBudget = MarketingBudget * 2 '
        . 'WHERE SingerId = 1 and AlbumId = 1 '
        . 'THEN RETURN MarketingBudget'
    );
    foreach ($result->rows() as $row) {
        printf('MarketingBudget: %s' . PHP_EOL, $row['MarketingBudget']);
    }
    printf(
        'Updated row(s) count: %d' . PHP_EOL,
        $result->stats()['rowCountExact']
    );
    $transaction->commit();
}
```

### Python

```
# instance_id = "your-spanner-instance"
# database_id = "your-spanner-db-id"

spanner_client = spanner.Client()
instance = spanner_client.instance(instance_id)
database = instance.database(database_id)

# Update MarketingBudget column for records satisfying
# a particular condition and returns the modified
# MarketingBudget column of the updated records using
# 'THEN RETURN MarketingBudget'.
# It is also possible to return all columns of all the
# updated records by using 'THEN RETURN *'.
def update_albums(transaction):
    results = transaction.execute_sql(
        "UPDATE Albums "
        "SET MarketingBudget = MarketingBudget * 2 "
        "WHERE SingerId = 1 and AlbumId = 1 "
        "THEN RETURN MarketingBudget"
    )
    for result in results:
        print("MarketingBudget: {}".format(*result))
    print("{} record(s) updated.".format(results.stats.row_count_exact))

database.run_in_transaction(update_albums)
```

### Ruby

```
require "google/cloud/spanner"

##
# This is a snippet for showcasing how to use DML return feature with update
# operation.
#
# @param project_id  [String] The ID of the Google Cloud project.
# @param instance_id [String] The ID of the spanner instance.
# @param database_id [String] The ID of the database.
#
def spanner_update_dml_returning project_id:, instance_id:, database_id:
  spanner = Google::Cloud::Spanner.new project: project_id
  client = spanner.client instance_id, database_id

  client.transaction do |transaction|
    # Update MarketingBudget column for records satisfying a particular
    # condition and returns the modified MarketingBudget column of the
    # updated records using ‘THEN RETURN MarketingBudget’.
    #
    # It is also possible to return all columns of all the updated records
    # by using ‘THEN RETURN *’.
    results = transaction.execute_query "UPDATE Albums SET MarketingBudget = MarketingBudget * 2
                                         WHERE SingerId = 1 and AlbumId = 1
                                         THEN RETURN MarketingBudget"
    results.rows.each do |row|
      puts "Updated Album with MarketingBudget: #{row[:MarketingBudget]}"
    end
    puts "Updated row(s) count: #{results.row_count}"
  end
end
```

### PostgreSQL

### C++

```
void UpdateUsingDmlReturning(google::cloud::spanner::Client client) {
  // Update MarketingBudget column for records satisfying a particular
  // condition and return the modified MarketingBudget column of the
  // updated records using `RETURNING MarketingBudget`.
  auto commit = client.Commit(
      [&client](google::cloud::spanner::Transaction txn)
          -> google::cloud::StatusOr<google::cloud::spanner::Mutations> {
        auto sql = google::cloud::spanner::SqlStatement(R"""(
            UPDATE Albums SET MarketingBudget = MarketingBudget * 2
              WHERE SingerId = 1 AND AlbumId = 1
              RETURNING MarketingBudget
        )""");
        using RowType = std::tuple<std::optional<std::int64_t>>;
        auto rows = client.ExecuteQuery(std::move(txn), std::move(sql));
        for (auto& row : google::cloud::spanner::StreamOf<RowType>(rows)) {
          if (!row) return std::move(row).status();
          std::cout << "MarketingBudget: ";
          if (std::get<0>(*row).has_value()) {
            std::cout << *std::get<0>(*row);
          } else {
            std::cout << "NULL";
          }
          std::cout << "\n";
        }
        std::cout << "Updated row(s) count: " << rows.RowsModified() << "\n";
        return google::cloud::spanner::Mutations{};
      });
  if (!commit) throw std::move(commit).status();
}
```

### C#

```
using Google.Cloud.Spanner.Data;
using System;
using System.Collections.Generic;
using System.Threading.Tasks;

public class UpdateUsingDmlReturningAsyncPostgresSample
{
    public async Task<List<long>> UpdateUsingDmlReturningAsyncPostgres(string projectId, string instanceId, string databaseId)
    {
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";

        using var connection = new SpannerConnection(connectionString);
        await connection.OpenAsync();

        // Update MarketingBudget column for records satisfying
        // a particular condition and return the modified
        // MarketingBudget column of the updated records using
        // 'RETURNING MarketingBudget'.
        // It is also possible to return all columns of all the
        // updated records by using 'RETURNING *'.
        using var cmd = connection.CreateDmlCommand("UPDATE Albums SET MarketingBudget = MarketingBudget * 2 WHERE SingerId = 14 and AlbumId = 20 RETURNING MarketingBudget");

        var reader = await cmd.ExecuteReaderAsync();
        var updatedMarketingBudgets = new List<long>();
        while (await reader.ReadAsync())
        {
            updatedMarketingBudgets.Add(reader.GetFieldValue<long>("marketingbudget"));
        }

        Console.WriteLine($"{updatedMarketingBudgets.Count} row(s) updated...");
        return updatedMarketingBudgets;
    }
}
```

### Go

```
import (
    "context"
    "fmt"
    "io"

    "cloud.google.com/go/spanner"
    "google.golang.org/api/iterator"
)

func pgUpdateUsingDMLReturning(w io.Writer, db string) error {
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    // Update MarketingBudget column for records satisfying
    // a particular condition and returns the modified
    // MarketingBudget column of the updated records using
    // 'RETURNING MarketingBudget'.
    // It is also possible to return all columns of all the
    // updated records by using 'RETURNING *'.
    _, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        stmt := spanner.Statement{
            SQL: `UPDATE Albums
                SET MarketingBudget = MarketingBudget * 2
                WHERE SingerId = 1 and AlbumId = 1
                RETURNING MarketingBudget`,
        }
        iter := txn.Query(ctx, stmt)
        defer iter.Stop()
        for {
            row, err := iter.Next()
            if err == iterator.Done {
                break
            }
            if err != nil {
                return err
            }
            var marketingBudget int64
            if err := row.Columns(&marketingBudget); err != nil {
                return err
            }
            fmt.Fprintf(w, "%d\n", marketingBudget)
        }
        fmt.Fprintf(w, "%d record(s) updated.\n", iter.RowCount)
        return nil
    })
    return err
}
```

### Java

```
import com.google.cloud.spanner.DatabaseClient;
import com.google.cloud.spanner.DatabaseId;
import com.google.cloud.spanner.ResultSet;
import com.google.cloud.spanner.Spanner;
import com.google.cloud.spanner.SpannerOptions;
import com.google.cloud.spanner.Statement;

public class PgUpdateUsingDmlReturningSample {

  static void updateUsingDmlReturning() {
    // TODO(developer): Replace these variables before running the sample.
    final String projectId = "my-project";
    final String instanceId = "my-instance";
    final String databaseId = "my-database";
    updateUsingDmlReturning(projectId, instanceId, databaseId);
  }

  static void updateUsingDmlReturning(String projectId, String instanceId, String databaseId) {
    try (Spanner spanner =
        SpannerOptions.newBuilder()
            .setProjectId(projectId)
            .build()
            .getService()) {
      final DatabaseClient dbClient =
          spanner.getDatabaseClient(DatabaseId.of(projectId, instanceId, databaseId));
      // Update MarketingBudget column for records satisfying
      // a particular condition and returns the modified
      // MarketingBudget column of the updated records using
      // ‘RETURNING MarketingBudget’.
      // It is also possible to return all columns of all the
      // updated records by using ‘RETURNING *’.
      dbClient
          .readWriteTransaction()
          .run(
              transaction -> {
                String sql =
                    "UPDATE Albums "
                        + "SET MarketingBudget = MarketingBudget * 2 "
                        + "WHERE SingerId = 1 and AlbumId = 1 "
                        + "RETURNING MarketingBudget";

                // readWriteTransaction.executeQuery(..) API should be used for executing
                // DML statements with RETURNING clause.
                try (ResultSet resultSet = transaction.executeQuery(Statement.of(sql))) {
                  while (resultSet.next()) {
                    System.out.printf("%d\n", resultSet.getLong(0));
                  }
                  System.out.printf(
                      "Updated row(s) count: %d\n", resultSet.getStats().getRowCountExact());
                }
                return null;
              });
    }
  }
}
```

### Node.js

```
// Imports the Google Cloud client library.
const {Spanner} = require('@google-cloud/spanner');

/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';
// const databaseId = 'my-database';

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});

function pgUpdateUsingDmlReturning(instanceId, databaseId) {
  // Gets a reference to a Cloud Spanner instance and database.
  const instance = spanner.instance(instanceId);
  const database = instance.database(databaseId);

  database.runTransaction(async (err, transaction) => {
    if (err) {
      console.error(err);
      return;
    }
    try {
      const [rows, stats] = await transaction.run({
        sql: 'UPDATE singers SET FirstName = $1, LastName = $2 WHERE singerid = $3 RETURNING FullName',
        params: {
          p1: 'Virginia1',
          p2: 'Watson1',
          p3: 18,
        },
      });

      const rowCount = Math.floor(stats[stats.rowCount]);
      console.log(
        `Successfully updated ${rowCount} record into the Singers table.`,
      );
      rows.forEach(row => {
        console.log(row.toJSON().fullname);
      });

      await transaction.commit();
    } catch (err) {
      console.error('ERROR:', err);
    } finally {
      // Close the database when finished.
      database.close();
    }
  });
}
pgUpdateUsingDmlReturning(instanceId, databaseId);
```

### PHP

```
use Google\Cloud\Spanner\SpannerClient;

/**
 * Update the given postgresql database using DML returning.
 *
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 */
function pg_update_dml_returning(string $instanceId, string $databaseId): void
{
    $spanner = new SpannerClient();
    $instance = $spanner->instance($instanceId);
    $database = $instance->database($databaseId);

    $transaction = $database->transaction();

    // Update MarketingBudget column for records satisfying a particular
    // condition and returns the modified MarketingBudget column of the updated
    // records using ‘RETURNING MarketingBudget’. It is also possible to return
    // all columns of all the updated records by using ‘RETURNING *’.

    $result = $transaction->execute(
        'UPDATE Albums '
        . 'SET MarketingBudget = MarketingBudget * 2 '
        . 'WHERE SingerId = 1 and AlbumId = 1 '
        . 'RETURNING MarketingBudget'
    );
    foreach ($result->rows() as $row) {
        printf('MarketingBudget: %s' . PHP_EOL, $row['marketingbudget']);
    }
    printf(
        'Updated row(s) count: %d' . PHP_EOL,
        $result->stats()['rowCountExact']
    );
    $transaction->commit();
}
```

### Python

```
# instance_id = "your-spanner-instance"
# database_id = "your-spanner-db-id"

spanner_client = spanner.Client()
instance = spanner_client.instance(instance_id)
database = instance.database(database_id)

# Update MarketingBudget column for records satisfying
# a particular condition and returns the modified
# MarketingBudget column of the updated records using
# 'RETURNING MarketingBudget'.
# It is also possible to return all columns of all the
# updated records by using 'RETURNING *'.
def update_albums(transaction):
    results = transaction.execute_sql(
        "UPDATE Albums "
        "SET MarketingBudget = MarketingBudget * 2 "
        "WHERE SingerId = 1 and AlbumId = 1 "
        "RETURNING MarketingBudget"
    )
    for result in results:
        print("MarketingBudget: {}".format(*result))
    print("{} record(s) updated.".format(results.stats.row_count_exact))

database.run_in_transaction(update_albums)
```

### Ruby

```
require "google/cloud/spanner"

##
# This is a snippet for showcasing how to use DML return feature with update
# operation in PostgreSql.
#
# @param project_id  [String] The ID of the Google Cloud project.
# @param instance_id [String] The ID of the spanner instance.
# @param database_id [String] The ID of the database.
#
def spanner_postgresql_update_dml_returning project_id:, instance_id:, database_id:
  spanner = Google::Cloud::Spanner.new project: project_id
  client = spanner.client instance_id, database_id

  client.transaction do |transaction|
    # Update MarketingBudget column for records satisfying a particular
    # condition and returns the modified MarketingBudget column of the
    # updated records using ‘RETURNING MarketingBudget’.
    # It is also possible to return all columns of all the updated records
    # by using ‘RETURNING *’.
    results = transaction.execute_query "UPDATE Albums SET MarketingBudget = MarketingBudget * 2
                                         WHERE SingerId = 1 and AlbumId = 1
                                         RETURNING MarketingBudget"
    results.rows.each do |row|
      puts "Updated Albums with MarketingBudget: #{row[:marketingbudget]}"
    end
    puts "Updated row(s) count: #{results.row_count}"
  end
end
```

The following code example deletes all the rows in the `Singers` table where the `FirstName` column is `Alice` , and it returns the `SingerId` and `FullName` column of the deleted records.

### GoogleSQL

### C++

```
void DeleteUsingDmlReturning(google::cloud::spanner::Client client) {
  // Delete records from SINGERS table satisfying a particular condition
  // and return the SingerId and FullName column of the deleted records
  // using `THEN RETURN SingerId, FullName'.
  auto commit = client.Commit(
      [&client](google::cloud::spanner::Transaction txn)
          -> google::cloud::StatusOr<google::cloud::spanner::Mutations> {
        auto sql = google::cloud::spanner::SqlStatement(R"""(
            DELETE FROM Singers
              WHERE FirstName = 'Alice'
              THEN RETURN SingerId, FullName
        )""");
        using RowType = std::tuple<std::int64_t, std::string>;
        auto rows = client.ExecuteQuery(std::move(txn), std::move(sql));
        // Note: This mutator might be re-run, or its effects discarded, so
        // changing non-transactional state (e.g., by producing output) is,
        // in general, not something to be imitated.
        for (auto& row : google::cloud::spanner::StreamOf<RowType>(rows)) {
          if (!row) return std::move(row).status();
          std::cout << "SingerId: " << std::get<0>(*row) << " ";
          std::cout << "FullName: " << std::get<1>(*row) << "\n";
        }
        std::cout << "Deleted row(s) count: " << rows.RowsModified() << "\n";
        return google::cloud::spanner::Mutations{};
      });
  if (!commit) throw std::move(commit).status();
}
```

### C#

```
using Google.Cloud.Spanner.Data;
using System;
using System.Collections.Generic;
using System.Threading;
using System.Threading.Tasks;

public class DeleteUsingDmlReturningAsyncSample
{
    public async Task<List<string>> DeleteUsingDmlReturningAsync(string projectId, string instanceId, string databaseId)
    {
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";

        using var connection = new SpannerConnection(connectionString);
        await connection.OpenAsync();

        // Delete records from SINGERS table satisfying a
        // particular condition and return the SingerId
        // and FullName column of the deleted records using
        // 'THEN RETURN SingerId, FullName'.
        // It is also possible to return all columns of all the
        // deleted records by using 'THEN RETURN *'.
        using var cmd = connection.CreateDmlCommand("DELETE FROM Singers WHERE FirstName = 'Alice' THEN RETURN SingerId, FullName");
        var reader = await cmd.ExecuteReaderAsync();
        var deletedSingerNames = new List<string>();
        while (await reader.ReadAsync())
        {
            deletedSingerNames.Add(reader.GetFieldValue<string>("FullName"));
        }

        Console.WriteLine($"{deletedSingerNames.Count} row(s) deleted...");
        return deletedSingerNames;
    }
}
```

### Go

```
import (
    "context"
    "fmt"
    "io"

    "cloud.google.com/go/spanner"
    "google.golang.org/api/iterator"
)

func deleteUsingDMLReturning(w io.Writer, db string) error {
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    // Delete records from SINGERS table satisfying a
    // particular condition and returns the SingerId
    // and FullName column of the deleted records using
    // 'THEN RETURN SingerId, FullName'.
    // It is also possible to return all columns of all the
    // deleted records by using 'THEN RETURN *'.
    _, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        stmt := spanner.Statement{
            SQL: `DELETE FROM Singers WHERE FirstName = 'Alice'
                    THEN RETURN SingerId, FullName`,
        }
        iter := txn.Query(ctx, stmt)
        defer iter.Stop()
        for {
            row, err := iter.Next()
            if err == iterator.Done {
                break
            }
            if err != nil {
                return err
            }
            var (
                singerID int64
                fullName string
            )
            if err := row.Columns(&singerID, &fullName); err != nil {
                return err
            }
            fmt.Fprintf(w, "%d %s\n", singerID, fullName)
        }
        fmt.Fprintf(w, "%d record(s) deleted.\n", iter.RowCount)
        return nil
    })
    return err
}
```

### Java

```
import com.google.cloud.spanner.DatabaseClient;
import com.google.cloud.spanner.DatabaseId;
import com.google.cloud.spanner.ResultSet;
import com.google.cloud.spanner.Spanner;
import com.google.cloud.spanner.SpannerOptions;
import com.google.cloud.spanner.Statement;

public class DeleteUsingDmlReturningSample {

  static void deleteUsingDmlReturningSample() {
    // TODO(developer): Replace these variables before running the sample.
    final String projectId = "my-project";
    final String instanceId = "my-instance";
    final String databaseId = "my-database";
    deleteUsingDmlReturningSample(projectId, instanceId, databaseId);
  }

  static void deleteUsingDmlReturningSample(
      String projectId, String instanceId, String databaseId) {
    try (Spanner spanner =
        SpannerOptions.newBuilder()
            .setProjectId(projectId)
            .build()
            .getService()) {
      final DatabaseClient dbClient =
          spanner.getDatabaseClient(DatabaseId.of(projectId, instanceId, databaseId));
      // Delete records from SINGERS table satisfying a
      // particular condition and returns the SingerId
      // and FullName column of the deleted records using
      // ‘THEN RETURN SingerId, FullName’.
      // It is also possible to return all columns of all the
      // deleted records by using ‘THEN RETURN *’.
      dbClient
          .readWriteTransaction()
          .run(
              transaction -> {
                String sql =
                    "DELETE FROM Singers WHERE FirstName = 'Alice' THEN RETURN SingerId, FullName";

                // readWriteTransaction.executeQuery(..) API should be used for executing
                // DML statements with RETURNING clause.
                try (ResultSet resultSet = transaction.executeQuery(Statement.of(sql))) {
                  while (resultSet.next()) {
                    System.out.printf("%d %s\n", resultSet.getLong(0), resultSet.getString(1));
                  }
                  System.out.printf(
                      "Deleted row(s) count: %d\n", resultSet.getStats().getRowCountExact());
                }
                return null;
              });
    }
  }
}
```

### Node.js

```
// Imports the Google Cloud client library.
const {Spanner} = require('@google-cloud/spanner');

/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';
// const databaseId = 'my-database';

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});

function deleteUsingDmlReturning(instanceId, databaseId) {
  // Gets a reference to a Cloud Spanner instance and database.
  const instance = spanner.instance(instanceId);
  const database = instance.database(databaseId);

  database.runTransaction(async (err, transaction) => {
    if (err) {
      console.error(err);
      return;
    }
    try {
      const [rows, stats] = await transaction.run({
        sql: 'DELETE FROM Singers WHERE SingerId = 18 THEN RETURN FullName',
      });

      const rowCount = Math.floor(stats[stats.rowCount]);
      console.log(
        `Successfully deleted ${rowCount} record from the Singers table.`,
      );
      rows.forEach(row => {
        console.log(row.toJSON().FullName);
      });

      await transaction.commit();
    } catch (err) {
      console.error('ERROR:', err);
    } finally {
      // Close the database when finished.
      database.close();
    }
  });
}
deleteUsingDmlReturning(instanceId, databaseId);
```

### PHP

```
use Google\Cloud\Spanner\SpannerClient;

/**
 * Delete data from the given database using DML returning.
 *
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 */
function delete_dml_returning(string $instanceId, string $databaseId): void
{
    $spanner = new SpannerClient();
    $instance = $spanner->instance($instanceId);
    $database = $instance->database($databaseId);

    $transaction = $database->transaction();

    // Delete records from SINGERS table satisfying a particular condition and
    // returns the SingerId and FullName column of the deleted records using
    // 'THEN RETURN SingerId, FullName'. It is also possible to return all columns
    //  of all the deleted records by using 'THEN RETURN *'.

    $result = $transaction->execute(
        "DELETE FROM Singers WHERE FirstName = 'Alice' "
        . 'THEN RETURN SingerId, FullName',
    );
    foreach ($result->rows() as $row) {
        printf(
            '%d %s.' . PHP_EOL,
            $row['SingerId'],
            $row['FullName']
        );
    }
    printf(
        'Deleted row(s) count: %d' . PHP_EOL,
        $result->stats()['rowCountExact']
    );
    $transaction->commit();
}
```

### Python

```
# instance_id = "your-spanner-instance"
# database_id = "your-spanner-db-id"

spanner_client = spanner.Client()
instance = spanner_client.instance(instance_id)
database = instance.database(database_id)

# Delete records from SINGERS table satisfying a
# particular condition and returns the SingerId
# and FullName column of the deleted records using
# 'THEN RETURN SingerId, FullName'.
# It is also possible to return all columns of all the
# deleted records by using 'THEN RETURN *'.
def delete_singers(transaction):
    results = transaction.execute_sql(
        "DELETE FROM Singers WHERE FirstName = 'David' "
        "THEN RETURN SingerId, FullName"
    )
    for result in results:
        print("SingerId: {}, FullName: {}".format(*result))
    print("{} record(s) deleted.".format(results.stats.row_count_exact))

database.run_in_transaction(delete_singers)
```

### Ruby

```
require "google/cloud/spanner"

##
# This is a snippet for showcasing how to use DML return feature with delete
# operation.
#
# @param project_id  [String] The ID of the Google Cloud project.
# @param instance_id [String] The ID of the spanner instance.
# @param database_id [String] The ID of the database.
#
def spanner_delete_dml_returning project_id:, instance_id:, database_id:
  spanner = Google::Cloud::Spanner.new project: project_id
  client = spanner.client instance_id, database_id

  client.transaction do |transaction|
    # Delete records from SINGERS table satisfying a particular condition and
    # returns the SingerId and FullName column of the deleted records using
    # ‘THEN RETURN SingerId, FullName’.
    # It is also possible to return all columns of all the deleted records
    # by using ‘THEN RETURN *’.
    results = transaction.execute_query "DELETE FROM Singers WHERE FirstName = 'Alice' THEN RETURN SingerId, FullName"
    results.rows.each do |row|
      puts "Deleted singer with SingerId: #{row[:SingerId]}, FullName: #{row[:FullName]}"
    end
    puts "Deleted row(s) count: #{results.row_count}"
  end
end
```

### PostgreSQL

### C++

```
void DeleteUsingDmlReturning(google::cloud::spanner::Client client) {
  // Delete records from SINGERS table satisfying a particular condition
  // and return the SingerId and FullName column of the deleted records
  // using `RETURNING SingerId, FullName'.
  auto commit = client.Commit(
      [&client](google::cloud::spanner::Transaction txn)
          -> google::cloud::StatusOr<google::cloud::spanner::Mutations> {
        auto sql = google::cloud::spanner::SqlStatement(R"""(
            DELETE FROM Singers
              WHERE FirstName = 'Alice'
              RETURNING SingerId, FullName
        )""");
        using RowType = std::tuple<std::int64_t, std::string>;
        auto rows = client.ExecuteQuery(std::move(txn), std::move(sql));
        for (auto& row : google::cloud::spanner::StreamOf<RowType>(rows)) {
          if (!row) return std::move(row).status();
          std::cout << "SingerId: " << std::get<0>(*row) << " ";
          std::cout << "FullName: " << std::get<1>(*row) << "\n";
        }
        std::cout << "Deleted row(s) count: " << rows.RowsModified() << "\n";
        return google::cloud::spanner::Mutations{};
      });
  if (!commit) throw std::move(commit).status();
}
```

### C#

```
using Google.Cloud.Spanner.Data;
using System;
using System.Collections.Generic;
using System.Threading.Tasks;

public class DeleteUsingDmlReturningAsyncPostgresSample
{
    public async Task<List<string>> DeleteUsingDmlReturningAsyncPostgres(string projectId, string instanceId, string databaseId)
    {
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";

        using var connection = new SpannerConnection(connectionString);
        await connection.OpenAsync();

        // Delete records from SINGERS table satisfying a
        // particular condition and return the SingerId
        // and FullName column of the deleted records using
        // 'RETURNING SingerId, FullName'.
        // It is also possible to return all columns of all the
        // deleted records by using 'RETURNING *'.
        using var cmd = connection.CreateDmlCommand("DELETE FROM Singers WHERE FirstName = 'Lata' RETURNING SingerId, FullName");
        var reader = await cmd.ExecuteReaderAsync();
        var deletedSingerNames = new List<string>();
        while (await reader.ReadAsync())
        {
            deletedSingerNames.Add(reader.GetFieldValue<string>("fullname"));
        }

        Console.WriteLine($"{deletedSingerNames.Count} row(s) deleted...");
        return deletedSingerNames;
    }
}
```

### Go

```
import (
    "context"
    "fmt"
    "io"

    "cloud.google.com/go/spanner"
    "google.golang.org/api/iterator"
)

func pgDeleteUsingDMLReturning(w io.Writer, db string) error {
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    // Delete records from SINGERS table satisfying a
    // particular condition and returns the SingerId
    // and FullName column of the deleted records using
    // 'RETURNING SingerId, FullName'.
    // It is also possible to return all columns of all the
    // deleted records by using 'RETURNING *'.
    _, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        stmt := spanner.Statement{
            SQL: `DELETE FROM Singers WHERE FirstName = 'Alice'
                    RETURNING SingerId, FullName`,
        }
        iter := txn.Query(ctx, stmt)
        defer iter.Stop()
        for {
            row, err := iter.Next()
            if err == iterator.Done {
                break
            }
            if err != nil {
                return err
            }
            var (
                singerID int64
                fullName string
            )
            if err := row.Columns(&singerID, &fullName); err != nil {
                return err
            }
            fmt.Fprintf(w, "%d %s\n", singerID, fullName)
        }
        fmt.Fprintf(w, "%d record(s) deleted.\n", iter.RowCount)
        return nil
    })
    return err
}
```

### Java

```
import com.google.cloud.spanner.DatabaseClient;
import com.google.cloud.spanner.DatabaseId;
import com.google.cloud.spanner.ResultSet;
import com.google.cloud.spanner.Spanner;
import com.google.cloud.spanner.SpannerOptions;
import com.google.cloud.spanner.Statement;

public class PgDeleteUsingDmlReturningSample {

  static void deleteUsingDmlReturningSample() {
    // TODO(developer): Replace these variables before running the sample.
    final String projectId = "my-project";
    final String instanceId = "my-instance";
    final String databaseId = "my-database";
    deleteUsingDmlReturningSample(projectId, instanceId, databaseId);
  }

  static void deleteUsingDmlReturningSample(
      String projectId, String instanceId, String databaseId) {
    try (Spanner spanner =
        SpannerOptions.newBuilder()
            .setProjectId(projectId)
            .build()
            .getService()) {
      final DatabaseClient dbClient =
          spanner.getDatabaseClient(DatabaseId.of(projectId, instanceId, databaseId));
      // Delete records from SINGERS table satisfying a
      // particular condition and returns the SingerId
      // and FullName column of the deleted records using
      // ‘RETURNING SingerId, FullName’.
      // It is also possible to return all columns of all the
      // deleted records by using ‘RETURNING *’.
      dbClient
          .readWriteTransaction()
          .run(
              transaction -> {
                String sql =
                    "DELETE FROM Singers WHERE FirstName = 'Alice' RETURNING SingerId, FullName";

                // readWriteTransaction.executeQuery(..) API should be used for executing
                // DML statements with RETURNING clause.
                try (ResultSet resultSet = transaction.executeQuery(Statement.of(sql))) {
                  while (resultSet.next()) {
                    System.out.printf("%d %s\n", resultSet.getLong(0), resultSet.getString(1));
                  }
                  System.out.printf(
                      "Deleted row(s) count: %d\n", resultSet.getStats().getRowCountExact());
                }
                return null;
              });
    }
  }
}
```

### Node.js

```
// Imports the Google Cloud client library.
const {Spanner} = require('@google-cloud/spanner');

/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';
// const databaseId = 'my-database';

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});

function pgDeleteUsingDmlReturning(instanceId, databaseId) {
  // Gets a reference to a Cloud Spanner instance and database.
  const instance = spanner.instance(instanceId);
  const database = instance.database(databaseId);

  database.runTransaction(async (err, transaction) => {
    if (err) {
      console.error(err);
      return;
    }
    try {
      const [rows, stats] = await transaction.run({
        sql: 'DELETE FROM Singers WHERE SingerId = 18 RETURNING FullName',
      });

      const rowCount = Math.floor(stats[stats.rowCount]);
      console.log(
        `Successfully deleted ${rowCount} record from the Singers table.`,
      );
      rows.forEach(row => {
        console.log(row.toJSON().fullname);
      });

      await transaction.commit();
    } catch (err) {
      console.error('ERROR:', err);
    } finally {
      // Close the database when finished.
      database.close();
    }
  });
}
pgDeleteUsingDmlReturning(instanceId, databaseId);
```

### PHP

```
use Google\Cloud\Spanner\SpannerClient;

/**
 * Delete data from the given postgresql database using DML returning.
 *
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 */
function pg_delete_dml_returning(string $instanceId, string $databaseId): void
{
    $spanner = new SpannerClient();
    $instance = $spanner->instance($instanceId);
    $database = $instance->database($databaseId);

    $transaction = $database->transaction();

    // Delete records from SINGERS table satisfying a particular condition and
    // returns the SingerId and FullName column of the deleted records using
    // ‘RETURNING SingerId, FullName’. It is also possible to return all columns
    //  of all the deleted records by using ‘RETURNING *’.

    $result = $transaction->execute(
        "DELETE FROM Singers WHERE FirstName = 'Alice' "
        . 'RETURNING SingerId, FullName',
    );
    foreach ($result->rows() as $row) {
        printf(
            '%d %s.' . PHP_EOL,
            $row['singerid'],
            $row['fullname']
        );
    }
    printf(
        'Deleted row(s) count: %d' . PHP_EOL,
        $result->stats()['rowCountExact']
    );
    $transaction->commit();
}
```

### Python

```
# instance_id = "your-spanner-instance"
# database_id = "your-spanner-db-id"

spanner_client = spanner.Client()
instance = spanner_client.instance(instance_id)
database = instance.database(database_id)

# Delete records from SINGERS table satisfying a
# particular condition and returns the SingerId
# and FullName column of the deleted records using
# 'RETURNING SingerId, FullName'.
# It is also possible to return all columns of all the
# deleted records by using 'RETURNING *'.
def delete_singers(transaction):
    results = transaction.execute_sql(
        "DELETE FROM Singers WHERE FirstName = 'David' "
        "RETURNING SingerId, FullName"
    )
    for result in results:
        print("SingerId: {}, FullName: {}".format(*result))
    print("{} record(s) deleted.".format(results.stats.row_count_exact))

database.run_in_transaction(delete_singers)
```

### Ruby

```
require "google/cloud/spanner"

##
# This is a snippet for showcasing how to use DML return feature with delete
# operation in PostgreSql.
#
# @param project_id  [String] The ID of the Google Cloud project.
# @param instance_id [String] The ID of the spanner instance.
# @param database_id [String] The ID of the database.
#
def spanner_postgresql_delete_dml_returning project_id:, instance_id:, database_id:
  spanner = Google::Cloud::Spanner.new project: project_id
  client = spanner.client instance_id, database_id

  client.transaction do |transaction|
    # Delete records from SINGERS table satisfying a particular condition and
    # returns the SingerId and FullName column of the deleted records using
    # ‘RETURNING SingerId, FullName’.
    # It is also possible to return all columns of all the deleted records
    # by using ‘RETURNING *’.
    results = transaction.execute_query "DELETE FROM singers WHERE firstname = 'Alice' RETURNING SingerId, FullName"
    results.rows.each do |row|
      puts "Deleted singer with SingerId: #{row[:singerid]}, FullName: #{row[:fullname]}"
    end
    puts "Deleted row(s) count: #{results.row_count}"
  end
end
```

### Read data written in the same transaction

Changes you make using DML statements are visible to subsequent statements in the same transaction. This is different from using [mutations](https://docs.cloud.google.com/spanner/docs/modify-mutation-api) , where changes are not visible until the transaction commits.

Spanner checks the constraints after every DML statement. This is different from using mutations, where Spanner buffers mutations in the client until commit and checks constraints at commit time. Evaluating the constraints after each statement allows Spanner to guarantee that the data that a DML statement returns is consistent with the schema.

The following example updates a row in the `Singers` table, then executes a `SELECT` statement to print the new values.

### C++

```
void DmlWriteThenRead(google::cloud::spanner::Client client) {
  namespace spanner = ::google::cloud::spanner;
  using ::google::cloud::StatusOr;

  auto commit_result = client.Commit(
      [&client](spanner::Transaction txn) -> StatusOr<spanner::Mutations> {
        auto insert = client.ExecuteDml(
            txn, spanner::SqlStatement(
                     "INSERT INTO Singers (SingerId, FirstName, LastName)"
                     "  VALUES (11, 'Timothy', 'Campbell')"));
        if (!insert) return std::move(insert).status();
        // Read newly inserted record.
        spanner::SqlStatement select(
            "SELECT FirstName, LastName FROM Singers where SingerId = 11");
        using RowType = std::tuple<std::string, std::string>;
        auto rows = client.ExecuteQuery(std::move(txn), std::move(select));
        // Note: This mutator might be re-run, or its effects discarded, so
        // changing non-transactional state (e.g., by producing output) is,
        // in general, not something to be imitated.
        for (auto const& row : spanner::StreamOf<RowType>(rows)) {
          if (!row) return std::move(row).status();
          std::cout << "FirstName: " << std::get<0>(*row) << "\t";
          std::cout << "LastName: " << std::get<1>(*row) << "\n";
        }
        return spanner::Mutations{};
      });
  if (!commit_result) throw std::move(commit_result).status();
  std::cout << "Write then read succeeded [spanner_dml_write_then_read]\n";
}
```

### C#

```
using Google.Cloud.Spanner.Data;
using System;
using System.Threading.Tasks;

public class WriteAndReadUsingDmlCoreAsyncSample
{
    public async Task<int> WriteAndReadUsingDmlCoreAsync(string projectId, string instanceId, string databaseId)
    {
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";

        using var connection = new SpannerConnection(connectionString);
        await connection.OpenAsync();

        using var createDmlCmd = connection.CreateDmlCommand(@"INSERT Singers (SingerId, FirstName, LastName) VALUES (11, 'Timothy', 'Campbell')");
        int rowCount = await createDmlCmd.ExecuteNonQueryAsync();
        Console.WriteLine($"{rowCount} row(s) inserted...");

        // Read newly inserted record.
        using var createSelectCmd = connection.CreateSelectCommand(@"SELECT FirstName, LastName FROM Singers WHERE SingerId = 11");
        using var reader = await createSelectCmd.ExecuteReaderAsync();
        while (await reader.ReadAsync())
        {
            Console.WriteLine($"{reader.GetFieldValue<string>("FirstName")}  {reader.GetFieldValue<string>("LastName")}");
        }
        return rowCount;
    }
}
```

### Go

```
import (
    "context"
    "fmt"
    "io"

    "cloud.google.com/go/spanner"
    "google.golang.org/api/iterator"
)

func writeAndReadUsingDML(w io.Writer, db string) error {
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    _, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        // Insert Record
        stmt := spanner.Statement{
            SQL: `INSERT Singers (SingerId, FirstName, LastName)
                VALUES (11, 'Timothy', 'Campbell')`,
        }
        rowCount, err := txn.Update(ctx, stmt)
        if err != nil {
            return err
        }
        fmt.Fprintf(w, "%d record(s) inserted.\n", rowCount)

        // Read newly inserted record
        stmt = spanner.Statement{SQL: `SELECT FirstName, LastName FROM Singers WHERE SingerId = 11`}
        iter := txn.Query(ctx, stmt)
        defer iter.Stop()

        for {
            row, err := iter.Next()
            if err == iterator.Done || err != nil {
                break
            }
            var firstName, lastName string
            if err := row.ColumnByName("FirstName", &firstName); err != nil {
                return err
            }
            if err := row.ColumnByName("LastName", &lastName); err != nil {
                return err
            }
            fmt.Fprintf(w, "Found record name with %s, %s", firstName, lastName)
        }
        return err
    })
    return err
}
```

### Java

```
static void writeAndReadUsingDml(DatabaseClient dbClient) {
  dbClient
      .readWriteTransaction()
      .run(transaction -> {
        // Insert record.
        String sql =
            "INSERT INTO Singers (SingerId, FirstName, LastName) "
                + " VALUES (11, 'Timothy', 'Campbell')";
        long rowCount = transaction.executeUpdate(Statement.of(sql));
        System.out.printf("%d record inserted.\n", rowCount);
        // Read newly inserted record.
        sql = "SELECT FirstName, LastName FROM Singers WHERE SingerId = 11";
        // We use a try-with-resource block to automatically release resources held by
        // ResultSet.
        try (ResultSet resultSet = transaction.executeQuery(Statement.of(sql))) {
          while (resultSet.next()) {
            System.out.printf(
                "%s %s\n",
                resultSet.getString("FirstName"), resultSet.getString("LastName"));
          }
        }
        return null;
      });
}
```

### Node.js

```
// Imports the Google Cloud client library
const {Spanner} = require('@google-cloud/spanner');

/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';
// const databaseId = 'my-database';

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});

// Gets a reference to a Cloud Spanner instance and database
const instance = spanner.instance(instanceId);
const database = instance.database(databaseId);

database.runTransaction(async (err, transaction) => {
  if (err) {
    console.error(err);
    return;
  }
  try {
    await transaction.runUpdate({
      sql: `INSERT Singers (SingerId, FirstName, LastName)
        VALUES (11, 'Timothy', 'Campbell')`,
    });

    const [rows] = await transaction.run({
      sql: 'SELECT FirstName, LastName FROM Singers',
    });
    rows.forEach(row => {
      const json = row.toJSON();
      console.log(`${json.FirstName} ${json.LastName}`);
    });

    await transaction.commit();
  } catch (err) {
    console.error('ERROR:', err);
  } finally {
    // Close the database when finished.
    database.close();
  }
});
```

### PHP

```
use Google\Cloud\Spanner\SpannerClient;
use Google\Cloud\Spanner\Transaction;

/**
 * Writes then reads data inside a Transaction with a DML statement.
 *
 * The database and table must already exist and can be created using
 * `create_database`.
 * Example:
 * ```
 * insert_data($instanceId, $databaseId);
 * ```
 *
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 */
function write_read_with_dml(string $instanceId, string $databaseId): void
{
    $spanner = new SpannerClient();
    $instance = $spanner->instance($instanceId);
    $database = $instance->database($databaseId);

    $database->runTransaction(function (Transaction $t) {
        $rowCount = $t->executeUpdate(
            'INSERT Singers (SingerId, FirstName, LastName) '
            . " VALUES (11, 'Timothy', 'Campbell')");

        printf('Inserted %d row(s).' . PHP_EOL, $rowCount);

        $results = $t->execute('SELECT FirstName, LastName FROM Singers WHERE SingerId = 11');

        foreach ($results as $row) {
            printf('%s %s' . PHP_EOL, $row['FirstName'], $row['LastName']);
        }

        $t->commit();
    });
}
```

### Python

```
# instance_id = "your-spanner-instance"
# database_id = "your-spanner-db-id"

spanner_client = spanner.Client()
instance = spanner_client.instance(instance_id)
database = instance.database(database_id)

def write_then_read(transaction):
    # Insert record.
    row_ct = transaction.execute_update(
        "INSERT INTO Singers (SingerId, FirstName, LastName) "
        " VALUES (11, 'Timothy', 'Campbell')"
    )
    print("{} record(s) inserted.".format(row_ct))

    # Read newly inserted record.
    results = transaction.execute_sql(
        "SELECT FirstName, LastName FROM Singers WHERE SingerId = 11"
    )
    for result in results:
        print("FirstName: {}, LastName: {}".format(*result))

database.run_in_transaction(write_then_read)
```

### Ruby

```
# project_id  = "Your Google Cloud project ID"
# instance_id = "Your Spanner instance ID"
# database_id = "Your Spanner database ID"

require "google/cloud/spanner"

spanner = Google::Cloud::Spanner.new project: project_id
client  = spanner.client instance_id, database_id
row_count = 0

client.transaction do |transaction|
  row_count = transaction.execute_update(
    "INSERT INTO Singers (SingerId, FirstName, LastName) VALUES (11, 'Timothy', 'Campbell')"
  )
  puts "#{row_count} record updated."
  transaction.execute("SELECT FirstName, LastName FROM Singers WHERE SingerId = 11").rows.each do |row|
    puts "#{row[:FirstName]} #{row[:LastName]}"
  end
end
```

### Get the query plan

You can [retrieve a query plan](https://docs.cloud.google.com/spanner/docs/sql-best-practices#how-execute-queries) using the Google Cloud console, the client libraries, and the `gcloud` command-line tool.

## Use Partitioned DML

[Partitioned DML](https://docs.cloud.google.com/spanner/docs/dml-partitioned) is designed for bulk updates and deletes, particularly periodic cleanup and backfilling.

### Execute statements with the Google Cloud CLI

To execute a Partitioned DML statement, use the `gcloud spanner databases execute-sql` command with the `--enable-partitioned-dml` option. The following example updates rows in the `Albums` table.

```
gcloud spanner databases execute-sql example-db \
    --instance=test-instance --enable-partitioned-dml \
    --sql='UPDATE Albums SET MarketingBudget = 0 WHERE MarketingBudget IS NULL'
```

> **Note:** Spanner does not support `--query-mode=PLAN` and `--query-mode=PROFILE` for Partitioned DML.

### Modify data using the client library

The following code example updates the `MarketingBudget` column of the `Albums` table.

### C++

You use the `ExecutePartitionedDml()` function to execute a Partitioned DML statement.

```
void DmlPartitionedUpdate(google::cloud::spanner::Client client) {
  namespace spanner = ::google::cloud::spanner;
  auto result = client.ExecutePartitionedDml(
      spanner::SqlStatement("UPDATE Albums SET MarketingBudget = 100000"
                            "  WHERE SingerId > 1"));
  if (!result) throw std::move(result).status();
  std::cout << "Updated at least " << result->row_count_lower_bound
            << " row(s) [spanner_dml_partitioned_update]\n";
}
```

### C#

You use the `ExecutePartitionedUpdateAsync()` method to execute a Partitioned DML statement.

```
using Google.Cloud.Spanner.Data;
using System;
using System.Threading.Tasks;

public class UpdateUsingPartitionedDmlCoreAsyncSample
{
    public async Task<long> UpdateUsingPartitionedDmlCoreAsync(string projectId, string instanceId, string databaseId)
    {
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";

        using var connection = new SpannerConnection(connectionString);
        await connection.OpenAsync();

        using var cmd = connection.CreateDmlCommand("UPDATE Albums SET MarketingBudget = 100000 WHERE SingerId > 1");
        long rowCount = await cmd.ExecutePartitionedUpdateAsync();

        Console.WriteLine($"{rowCount} row(s) updated...");
        return rowCount;
    }
}
```

### Go

You use the `PartitionedUpdate()` method to execute a Partitioned DML statement.

```
import (
    "context"
    "fmt"
    "io"

    "cloud.google.com/go/spanner"
)

func updateUsingPartitionedDML(w io.Writer, db string) error {
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    stmt := spanner.Statement{SQL: "UPDATE Albums SET MarketingBudget = 100000 WHERE SingerId > 1"}
    rowCount, err := client.PartitionedUpdate(ctx, stmt)
    if err != nil {
        return err
    }
    fmt.Fprintf(w, "%d record(s) updated.\n", rowCount)
    return nil
}
```

### Java

You use the `executePartitionedUpdate()` method to execute a Partitioned DML statement.

```
static void updateUsingPartitionedDml(DatabaseClient dbClient) {
  String sql = "UPDATE Albums SET MarketingBudget = 100000 WHERE SingerId > 1";
  long rowCount = dbClient.executePartitionedUpdate(Statement.of(sql));
  System.out.printf("%d records updated.\n", rowCount);
}
```

### Node.js

You use the `runPartitionedUpdate()` method to execute a Partitioned DML statement.

```
// Imports the Google Cloud client library
const {Spanner} = require('@google-cloud/spanner');

/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';
// const databaseId = 'my-database';

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});

// Gets a reference to a Cloud Spanner instance and database
const instance = spanner.instance(instanceId);
const database = instance.database(databaseId);

try {
  const [rowCount] = await database.runPartitionedUpdate({
    sql: 'UPDATE Albums SET MarketingBudget = 100000 WHERE SingerId > 1',
  });
  console.log(`Successfully updated ${rowCount} records.`);
} catch (err) {
  console.error('ERROR:', err);
} finally {
  // Close the database when finished.
  database.close();
}
```

### PHP

You use the `executePartitionedUpdate()` method to execute a Partitioned DML statement.

```
use Google\Cloud\Spanner\SpannerClient;

/**
 * Updates sample data in the database by partition with a DML statement.
 *
 * This updates the `MarketingBudget` column which must be created before
 * running this sample. You can add the column by running the `add_column`
 * sample or by running this DDL statement against your database:
 *
 *     ALTER TABLE Albums ADD COLUMN MarketingBudget INT64
 *
 * Example:
 * ```
 * update_data($instanceId, $databaseId);
 * ```
 *
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 */
function update_data_with_partitioned_dml(string $instanceId, string $databaseId): void
{
    $spanner = new SpannerClient();
    $instance = $spanner->instance($instanceId);
    $database = $instance->database($databaseId);

    $rowCount = $database->executePartitionedUpdate(
        'UPDATE Albums SET MarketingBudget = 100000 WHERE SingerId > 1'
    );

    printf('Updated %d row(s).' . PHP_EOL, $rowCount);
}
```

### Python

You use the `execute_partitioned_dml()` method to execute a Partitioned DML statement.

```
# instance_id = "your-spanner-instance"
# database_id = "your-spanner-db-id"

spanner_client = spanner.Client()
instance = spanner_client.instance(instance_id)
database = instance.database(database_id)

row_ct = database.execute_partitioned_dml(
    "UPDATE Albums SET MarketingBudget = 100000 WHERE SingerId > 1"
)

print("{} records updated.".format(row_ct))
```

### Ruby

You use the `execute_partitioned_update()` method to execute a Partitioned DML statement.

```
# project_id  = "Your Google Cloud project ID"
# instance_id = "Your Spanner instance ID"
# database_id = "Your Spanner database ID"

require "google/cloud/spanner"

spanner = Google::Cloud::Spanner.new project: project_id
client  = spanner.client instance_id, database_id

row_count = client.execute_partition_update(
  "UPDATE Albums SET MarketingBudget = 100000 WHERE SingerId > 1"
)

puts "#{row_count} records updated."
```

The following code example deletes rows from the `Singers` table, based on the `SingerId` column.

### C++

```
void DmlPartitionedDelete(google::cloud::spanner::Client client) {
  namespace spanner = ::google::cloud::spanner;
  auto result = client.ExecutePartitionedDml(
      spanner::SqlStatement("DELETE FROM Singers WHERE SingerId > 10"));
  if (!result) throw std::move(result).status();
  std::cout << "Deleted at least " << result->row_count_lower_bound
            << " row(s) [spanner_dml_partitioned_delete]\n";
}
```

### C#

```
using Google.Cloud.Spanner.Data;
using System;
using System.Threading.Tasks;

public class DeleteUsingPartitionedDmlCoreAsyncSample
{
    public async Task<long> DeleteUsingPartitionedDmlCoreAsync(string projectId, string instanceId, string databaseId)
    {
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";

        using var connection = new SpannerConnection(connectionString);
        await connection.OpenAsync();

        using var cmd = connection.CreateDmlCommand("DELETE FROM Singers WHERE SingerId > 10");
        long rowCount = await cmd.ExecutePartitionedUpdateAsync();

        Console.WriteLine($"{rowCount} row(s) deleted...");
        return rowCount;
    }
}
```

### Go

```
import (
    "context"
    "fmt"
    "io"

    "cloud.google.com/go/spanner"
)

func deleteUsingPartitionedDML(w io.Writer, db string) error {
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    stmt := spanner.Statement{SQL: "DELETE FROM Singers WHERE SingerId > 10"}
    rowCount, err := client.PartitionedUpdate(ctx, stmt)
    if err != nil {
        return err

    }
    fmt.Fprintf(w, "%d record(s) deleted.", rowCount)
    return nil
}
```

### Java

```
static void deleteUsingPartitionedDml(DatabaseClient dbClient) {
  String sql = "DELETE FROM Singers WHERE SingerId > 10";
  long rowCount = dbClient.executePartitionedUpdate(Statement.of(sql));
  System.out.printf("%d records deleted.\n", rowCount);
}
```

### Node.js

```
// Imports the Google Cloud client library
const {Spanner} = require('@google-cloud/spanner');

/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';
// const databaseId = 'my-database';

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});

// Gets a reference to a Cloud Spanner instance and database
const instance = spanner.instance(instanceId);
const database = instance.database(databaseId);

try {
  const [rowCount] = await database.runPartitionedUpdate({
    sql: 'DELETE FROM Singers WHERE SingerId > 10',
  });
  console.log(`Successfully deleted ${rowCount} records.`);
} catch (err) {
  console.error('ERROR:', err);
} finally {
  // Close the database when finished.
  database.close();
}
```

### PHP

```
use Google\Cloud\Spanner\SpannerClient;

/**
 * Delete sample data in the database by partition with a DML statement.
 *
 * This updates the `MarketingBudget` column which must be created before
 * running this sample. You can add the column by running the `add_column`
 * sample or by running this DDL statement against your database:
 *
 *     ALTER TABLE Albums ADD COLUMN MarketingBudget INT64
 *
 * Example:
 * ```
 * update_data($instanceId, $databaseId);
 * ```
 *
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 */
function delete_data_with_partitioned_dml(string $instanceId, string $databaseId): void
{
    $spanner = new SpannerClient();
    $instance = $spanner->instance($instanceId);
    $database = $instance->database($databaseId);

    $rowCount = $database->executePartitionedUpdate(
        'DELETE FROM Singers WHERE SingerId > 10'
    );

    printf('Deleted %d row(s).' . PHP_EOL, $rowCount);
}
```

### Python

```
# instance_id = "your-spanner-instance"
# database_id = "your-spanner-db-id"
spanner_client = spanner.Client()
instance = spanner_client.instance(instance_id)
database = instance.database(database_id)

row_ct = database.execute_partitioned_dml("DELETE FROM Singers WHERE SingerId > 10")

print("{} record(s) deleted.".format(row_ct))
```

### Ruby

```
# project_id  = "Your Google Cloud project ID"
# instance_id = "Your Spanner instance ID"
# database_id = "Your Spanner database ID"

require "google/cloud/spanner"

spanner = Google::Cloud::Spanner.new project: project_id
client  = spanner.client instance_id, database_id

row_count = client.execute_partition_update(
  "DELETE FROM Singers WHERE SingerId > 10"
)

puts "#{row_count} records deleted."
```

## Use batch DML

If you need to avoid the extra latency incurred from multiple serial requests, use batch DML to send multiple `INSERT` , `UPDATE` , or `DELETE` statements in a single transaction:

### C++

Use the `ExecuteBatchDml()` function to execute a list of DML statements.

```
void DmlBatchUpdate(google::cloud::spanner::Client client) {
  namespace spanner = ::google::cloud::spanner;

  auto commit_result =
      client.Commit([&client](spanner::Transaction const& txn)
                        -> google::cloud::StatusOr<spanner::Mutations> {
        std::vector<spanner::SqlStatement> statements = {
            spanner::SqlStatement("INSERT INTO Albums"
                                  " (SingerId, AlbumId, AlbumTitle,"
                                  " MarketingBudget)"
                                  " VALUES (1, 3, 'Test Album Title', 10000)"),
            spanner::SqlStatement("UPDATE Albums"
                                  " SET MarketingBudget = MarketingBudget * 2"
                                  "  WHERE SingerId = 1 and AlbumId = 3")};
        auto result = client.ExecuteBatchDml(txn, statements);
        if (!result) return std::move(result).status();
        // Note: This mutator might be re-run, or its effects discarded, so
        // changing non-transactional state (e.g., by producing output) is,
        // in general, not something to be imitated.
        for (std::size_t i = 0; i < result->stats.size(); ++i) {
          std::cout << result->stats[i].row_count << " rows affected"
                    << " for the statement " << (i + 1) << ".\n";
        }
        // Batch operations may have partial failures, in which case
        // ExecuteBatchDml returns with success, but the application should
        // verify that all statements completed successfully
        if (!result->status.ok()) return result->status;
        return spanner::Mutations{};
      });
  if (!commit_result) throw std::move(commit_result).status();
  std::cout << "Update was successful [spanner_dml_batch_update]\n";
}
```

### C#

Use the `connection.CreateBatchDmlCommand()` method to create your batch command, use the `Add` method to add DML statements, and execute the statements with the `ExecuteNonQueryAsync()` method.

```
using Google.Cloud.Spanner.Data;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading.Tasks;

public class UpdateUsingBatchDmlCoreAsyncSample
{
    public async Task<int> UpdateUsingBatchDmlCoreAsync(string projectId, string instanceId, string databaseId)
    {
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";

        using var connection = new SpannerConnection(connectionString);
        await connection.OpenAsync();

        SpannerBatchCommand cmd = connection.CreateBatchDmlCommand();

        cmd.Add("INSERT INTO Albums (SingerId, AlbumId, AlbumTitle, MarketingBudget) VALUES (1, 3, 'Test Album Title', 10000)");

        cmd.Add("UPDATE Albums SET MarketingBudget = MarketingBudget * 2 WHERE SingerId = 1 and AlbumId = 3");

        IEnumerable<long> affectedRows = await cmd.ExecuteNonQueryAsync();

        Console.WriteLine($"Executed {affectedRows.Count()} " + "SQL statements using Batch DML.");
        return affectedRows.Count();
    }
}
```

### Go

Use the `BatchUpdate()` method to execute an array of DML `Statement` objects.

```
import (
    "context"
    "fmt"
    "io"

    "cloud.google.com/go/spanner"
)

func updateUsingBatchDML(w io.Writer, db string) error {
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    _, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        stmts := []spanner.Statement{
            {SQL: `INSERT INTO Albums
                (SingerId, AlbumId, AlbumTitle, MarketingBudget)
                VALUES (1, 3, 'Test Album Title', 10000)`},
            {SQL: `UPDATE Albums
                SET MarketingBudget = MarketingBudget * 2
                WHERE SingerId = 1 and AlbumId = 3`},
        }
        rowCounts, err := txn.BatchUpdate(ctx, stmts)
        if err != nil {
            return err
        }
        fmt.Fprintf(w, "Executed %d SQL statements using Batch DML.\n", len(rowCounts))
        return nil
    })
    return err
}
```

### Java

Use the `transaction.batchUpdate()` method to execute an `ArrayList` of multiple DML `Statement` objects.

```
static void updateUsingBatchDml(DatabaseClient dbClient) {
  dbClient
      .readWriteTransaction()
      .run(transaction -> {
        List<Statement> stmts = new ArrayList<Statement>();
        String sql =
            "INSERT INTO Albums "
                + "(SingerId, AlbumId, AlbumTitle, MarketingBudget) "
                + "VALUES (1, 3, 'Test Album Title', 10000) ";
        stmts.add(Statement.of(sql));
        sql =
            "UPDATE Albums "
                + "SET MarketingBudget = MarketingBudget * 2 "
                + "WHERE SingerId = 1 and AlbumId = 3";
        stmts.add(Statement.of(sql));
        long[] rowCounts;
        try {
          rowCounts = transaction.batchUpdate(stmts);
        } catch (SpannerBatchUpdateException e) {
          rowCounts = e.getUpdateCounts();
        }
        for (int i = 0; i < rowCounts.length; i++) {
          System.out.printf("%d record updated by stmt %d.\n", rowCounts[i], i);
        }
        return null;
      });
}
```

### Node.js

Use `transaction.batchUpdate()` to execute a list of DML statements.

```
// Imports the Google Cloud client library
const {Spanner} = require('@google-cloud/spanner');

/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';
// const databaseId = 'my-database';

// Creates a client
const spanner = new Spanner({
  projectId: projectId,
});

// Gets a reference to a Cloud Spanner instance and database
const instance = spanner.instance(instanceId);
const database = instance.database(databaseId);

const insert = {
  sql: `INSERT INTO Albums (SingerId, AlbumId, AlbumTitle, MarketingBudget)
    VALUES (1, 3, "Test Album Title", 10000)`,
};

const update = {
  sql: `UPDATE Albums SET MarketingBudget = MarketingBudget * 2
    WHERE SingerId = 1 and AlbumId = 3`,
};

const dmlStatements = [insert, update];

try {
  await database.runTransactionAsync(async transaction => {
    const [rowCounts] = await transaction.batchUpdate(dmlStatements);
    await transaction.commit();
    console.log(
      `Successfully executed ${rowCounts.length} SQL statements using Batch DML.`,
    );
  });
} catch (err) {
  console.error('ERROR:', err);
  throw err;
} finally {
  // Close the database when finished.
  database.close();
}
```

### PHP

Use `executeUpdateBatch()` to create a list of DML statements, then use `commit()` to execute the statements.

```
use Google\Cloud\Spanner\SpannerClient;
use Google\Cloud\Spanner\Transaction;

/**
 * Updates sample data in the database with Batch DML.
 *
 * This requires the `MarketingBudget` column which must be created before
 * running this sample. You can add the column by running the `add_column`
 * sample or by running this DDL statement against your database:
 *
 *     ALTER TABLE Albums ADD COLUMN MarketingBudget INT64
 *
 * Example:
 * ```
 * update_data_with_batch_dml($instanceId, $databaseId);
 * ```
 *
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 */
function update_data_with_batch_dml(string $instanceId, string $databaseId): void
{
    $spanner = new SpannerClient();
    $instance = $spanner->instance($instanceId);
    $database = $instance->database($databaseId);

    $batchDmlResult = $database->runTransaction(function (Transaction $t) {
        $result = $t->executeUpdateBatch([
            [
                'sql' => 'INSERT INTO Albums '
                . '(SingerId, AlbumId, AlbumTitle, MarketingBudget) '
                . "VALUES (1, 3, 'Test Album Title', 10000)"
            ],
            [
                'sql' => 'UPDATE Albums '
                . 'SET MarketingBudget = MarketingBudget * 2 '
                . 'WHERE SingerId = 1 and AlbumId = 3'
            ],
        ]);
        $t->commit();
        $rowCounts = count($result->rowCounts());
        printf('Executed %s SQL statements using Batch DML.' . PHP_EOL,
            $rowCounts);
    });
}
```

### Python

Use `transaction.batch_update()` to execute multiple DML statement strings.

```
from google.rpc.code_pb2 import OK

# instance_id = "your-spanner-instance"
# database_id = "your-spanner-db-id"

spanner_client = spanner.Client()
instance = spanner_client.instance(instance_id)
database = instance.database(database_id)

insert_statement = (
    "INSERT INTO Albums "
    "(SingerId, AlbumId, AlbumTitle, MarketingBudget) "
    "VALUES (1, 3, 'Test Album Title', 10000)"
)

update_statement = (
    "UPDATE Albums "
    "SET MarketingBudget = MarketingBudget * 2 "
    "WHERE SingerId = 1 and AlbumId = 3"
)

def update_albums(transaction):
    status, row_cts = transaction.batch_update([insert_statement, update_statement])

    if status.code != OK:
        # Do handling here.
        # Note: the exception will still be raised when
        # `commit` is called by `run_in_transaction`.
        return

    print("Executed {} SQL statements using Batch DML.".format(len(row_cts)))

database.run_in_transaction(update_albums)
```

### Ruby

Use `transaction.batch_update` to execute multiple DML statement strings.

```
# project_id  = "Your Google Cloud project ID"
# instance_id = "Your Spanner instance ID"
# database_id = "Your Spanner database ID"

require "google/cloud/spanner"

spanner = Google::Cloud::Spanner.new project: project_id
client  = spanner.client instance_id, database_id

row_counts = nil
client.transaction do |transaction|
  row_counts = transaction.batch_update do |b|
    b.batch_update(
      "INSERT INTO Albums " \
      "(SingerId, AlbumId, AlbumTitle, MarketingBudget) " \
      "VALUES (1, 3, 'Test Album Title', 10000)"
    )
    b.batch_update(
      "UPDATE Albums " \
      "SET MarketingBudget = MarketingBudget * 2 " \
      "WHERE SingerId = 1 and AlbumId = 3"
    )
  end
end

statement_count = row_counts.count

puts "Executed #{statement_count} SQL statements using Batch DML."
```
