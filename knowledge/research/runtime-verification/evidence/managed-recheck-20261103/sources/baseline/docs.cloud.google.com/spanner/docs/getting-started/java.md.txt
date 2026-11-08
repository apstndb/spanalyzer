---
name: documents/docs.cloud.google.com/spanner/docs/getting-started/java
uri: https://docs.cloud.google.com/spanner/docs/getting-started/java
title: Getting started with Spanner in Java
description: 'Get started with Spanner in Java: install the client library, connect to a database, and perform data operations.'
data_source: docs.cloud.google.com
---

## Objectives

This tutorial walks you through the following steps using the Spanner client library for Java:

- Create a Spanner instance and database.
- Write, read, and execute SQL queries on data in the database.
- Update the database schema.
- Update data using a read-write transaction.
- Add a secondary index to the database.
- Use the index to read and execute SQL queries on data.
- Retrieve data using a read-only transaction.

## Costs

This tutorial uses Spanner, which is a billable component of the Google Cloud. For information on the cost of using Spanner, see [Pricing](https://cloud.google.com/spanner/pricing) .

## Before you begin

Complete the steps described in [Set up](https://docs.cloud.google.com/spanner/docs/getting-started/set-up#set_up_a_project) , which cover creating and setting a default Google Cloud project, enabling billing, enabling the Cloud Spanner API, and setting up OAuth 2.0 to get authentication credentials to use the Cloud Spanner API.

In particular, make sure that you run [`gcloud auth application-default login`](https://docs.cloud.google.com/sdk/gcloud/reference/auth/application-default/login) to set up your local development environment with authentication credentials.

> **Note:** If you don't plan to keep the resources that you create in this tutorial, consider creating a new Google Cloud project instead of selecting an existing project. After you finish the tutorial, you can delete the project, removing all resources associated with the project.

## Prepare your local Java environment

1.  Install the following on your development machine if they are not already installed:

    - Java 8 JDK ( [download](http://openjdk.java.net/) ).
    - Maven 3 ( [download](https://maven.apache.org/download.cgi) ).

2.  Clone the sample app repository to your local machine:

    ```
    git clone https://github.com/googleapis/google-cloud-java.git
    ```

3.  Change to the directory that contains the Spanner sample code:

    ```
    cd google-cloud-java/java-spanner/samples/snippets
    ```

4.  Generate the sample JAR file:

    ```
    mvn clean package
    ```

## Create an instance

When you first use Spanner, you must create an instance, which is an allocation of resources that are used by Spanner databases. When you create an instance, you choose an *instance configuration* , which determines where your data is stored, and also the number of nodes to use, which determines the amount of serving and storage resources in your instance.

See [Create an instance](https://docs.cloud.google.com/spanner/docs/create-manage-instances#create-instance) to learn how to create a Spanner instance using any of the following methods. You can name your instance `test-instance` to use it with other topics in this document that reference an instance named `test-instance` .

- The Google Cloud CLI
- The Google Cloud console
- A client library (C++, C#, Go, Java, Node.js, PHP, Python, or Ruby)

## Look through sample files

The samples repository contains a sample that shows how to use Spanner with Java.

## Create a database

### GoogleSQL

```
java -jar target/spanner-snippets/spanner-google-cloud-samples.jar \
createdatabase test-instance example-db
```

### PostgreSQL

```
java -jar target/spanner-snippets/spanner-google-cloud-samples.jar \
createpgdatabase test-instance example-db
```

You should see:

```
Created database [example-db]
```

The following code creates a database and two tables in the database.

> **Note:** The subsequent code samples use these two tables. If you don't execute this code, then create the tables by using the Google Cloud console or the gcloud CLI. For more information, see the [example schema](https://docs.cloud.google.com/spanner/docs/schema-and-data-model#creating-interleaved-tables) .

### GoogleSQL

```java
static void createDatabase(DatabaseAdminClient dbAdminClient,
    InstanceName instanceName, String databaseId) {
  CreateDatabaseRequest createDatabaseRequest =
      CreateDatabaseRequest.newBuilder()
          .setCreateStatement("CREATE DATABASE `" + databaseId + "`")
          .setParent(instanceName.toString())
          .addAllExtraStatements(Arrays.asList(
              "CREATE TABLE Singers ("
                  + "  SingerId   INT64 NOT NULL,"
                  + "  FirstName  STRING(1024),"
                  + "  LastName   STRING(1024),"
                  + "  SingerInfo BYTES(MAX),"
                  + "  FullName STRING(2048) AS "
                  + "  (ARRAY_TO_STRING([FirstName, LastName], \" \")) STORED"
                  + ") PRIMARY KEY (SingerId)",
              "CREATE TABLE Albums ("
                  + "  SingerId     INT64 NOT NULL,"
                  + "  AlbumId      INT64 NOT NULL,"
                  + "  AlbumTitle   STRING(MAX)"
                  + ") PRIMARY KEY (SingerId, AlbumId),"
                  + "  INTERLEAVE IN PARENT Singers ON DELETE CASCADE")).build();
  try {
    // Initiate the request which returns an OperationFuture.
    com.google.spanner.admin.database.v1.Database db =
        dbAdminClient.createDatabaseAsync(createDatabaseRequest).get();
    System.out.println("Created database [" + db.getName() + "]");
  } catch (ExecutionException e) {
    // If the operation failed during execution, expose the cause.
    throw (SpannerException) e.getCause();
  } catch (InterruptedException e) {
    // Throw when a thread is waiting, sleeping, or otherwise occupied,
    // and the thread is interrupted, either before or during the activity.
    throw SpannerExceptionFactory.propagateInterrupt(e);
  }
}
```

### PostgreSQL

```java
static void createPostgreSqlDatabase(
    DatabaseAdminClient dbAdminClient, String projectId, String instanceId, String databaseId) {
  final CreateDatabaseRequest request =
      CreateDatabaseRequest.newBuilder()
          .setCreateStatement("CREATE DATABASE \"" + databaseId + "\"")
          .setParent(InstanceName.of(projectId, instanceId).toString())
          .setDatabaseDialect(DatabaseDialect.POSTGRESQL).build();

  try {
    // Initiate the request which returns an OperationFuture.
    Database db = dbAdminClient.createDatabaseAsync(request).get();
    System.out.println("Created database [" + db.getName() + "]");
  } catch (ExecutionException e) {
    // If the operation failed during execution, expose the cause.
    throw (SpannerException) e.getCause();
  } catch (InterruptedException e) {
    // Throw when a thread is waiting, sleeping, or otherwise occupied,
    // and the thread is interrupted, either before or during the activity.
    throw SpannerExceptionFactory.propagateInterrupt(e);
  }
}
static void createTableUsingDdl(DatabaseAdminClient dbAdminClient, DatabaseName databaseName) {
  try {
    // Initiate the request which returns an OperationFuture.
    dbAdminClient.updateDatabaseDdlAsync(
        databaseName,
        Arrays.asList(
            "CREATE TABLE Singers ("
                + "  SingerId   bigint NOT NULL,"
                + "  FirstName  character varying(1024),"
                + "  LastName   character varying(1024),"
                + "  SingerInfo bytea,"
                + "  FullName character varying(2048) GENERATED "
                + "  ALWAYS AS (FirstName || ' ' || LastName) STORED,"
                + "  PRIMARY KEY (SingerId)"
                + ")",
            "CREATE TABLE Albums ("
                + "  SingerId     bigint NOT NULL,"
                + "  AlbumId      bigint NOT NULL,"
                + "  AlbumTitle   character varying(1024),"
                + "  PRIMARY KEY (SingerId, AlbumId)"
                + ") INTERLEAVE IN PARENT Singers ON DELETE CASCADE")).get();
    System.out.println("Created Singers & Albums tables in database: [" + databaseName + "]");
  } catch (ExecutionException e) {
    // If the operation failed during execution, expose the cause.
    throw SpannerExceptionFactory.asSpannerException(e);
  } catch (InterruptedException e) {
    // Throw when a thread is waiting, sleeping, or otherwise occupied,
    // and the thread is interrupted, either before or during the activity.
    throw SpannerExceptionFactory.propagateInterrupt(e);
  }
}
```

The next step is to write data to your database.

## Create a database client

Before you can do reads or writes, you must create a [`DatabaseClient`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.DatabaseClient) . You can think of a `DatabaseClient` as a database connection: all of your interactions with Spanner must go through a `DatabaseClient` . Typically you create a `DatabaseClient` when your application starts up, then you re-use that `DatabaseClient` to read, write, and execute transactions.

```java
SpannerOptions options = SpannerOptions.newBuilder().build();
Spanner spanner = options.getService();
DatabaseAdminClient dbAdminClient = null;
try {
  DatabaseClient dbClient = spanner.getDatabaseClient(db);
  dbAdminClient = spanner.createDatabaseAdminClient();
} finally {
  if (dbAdminClient != null) {
    if (!dbAdminClient.isShutdown() || !dbAdminClient.isTerminated()) {
      dbAdminClient.close();
    }
  }
  spanner.close();
}
```

Each client uses resources in Spanner, so it is good practice to close unneeded clients by calling [`close()`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.Spanner#com_google_cloud_spanner_Spanner_close__) .

Read more in the [`DatabaseClient`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.DatabaseClient) Javadoc reference.

## Write data with DML

You can insert data using Data Manipulation Language (DML) in a read-write transaction.

You use the `executeUpdate()` method to execute a DML statement.

```java
static void writeUsingDml(DatabaseClient dbClient) {
  // Insert 4 singer records
  dbClient
      .readWriteTransaction()
      .run(transaction -> {
        String sql =
            "INSERT INTO Singers (SingerId, FirstName, LastName) VALUES "
                + "(12, 'Melissa', 'Garcia'), "
                + "(13, 'Russell', 'Morales'), "
                + "(14, 'Jacqueline', 'Long'), "
                + "(15, 'Dylan', 'Shaw')";
        long rowCount = transaction.executeUpdate(Statement.of(sql));
        System.out.printf("%d records inserted.\n", rowCount);
        return null;
      });
}
```

Run the sample using the `writeusingdml` argument.

```
java -jar target/spanner-snippets/spanner-google-cloud-samples.jar \
    writeusingdml test-instance example-db
```

You should see:

```
4 records inserted.
```

> **Note:** There are limits to commit size. See [CRUD limit](https://docs.cloud.google.com/spanner/quotas#limits-for) for more information.

## Write data with mutations

You can also insert data using [mutations](https://docs.cloud.google.com/spanner/docs/modify-mutation-api) .

You can write data using a [`Mutation`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.Mutation) object. A `Mutation` object is a container for mutation operations. A `Mutation` represents a sequence of inserts, updates, and deletes that Spanner applies atomically to different rows and tables in a Spanner database.

The [`newInsertBuilder()`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.Mutation#com_google_cloud_spanner_Mutation_newInsertBuilder_java_lang_String_) method in the `Mutation` class constructs an `INSERT` mutation, which inserts a new row in a table. If the row already exists, the write fails. Alternatively, you can use the [`newInsertOrUpdateBuilder`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.Mutation#com_google_cloud_spanner_Mutation_newInsertOrUpdateBuilder_java_lang_String_) method to construct an `INSERT_OR_UPDATE` mutation, which updates column values if the row already exists.

The [`write()`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.DatabaseClient#com_google_cloud_spanner_DatabaseClient_write_java_lang_Iterable_com_google_cloud_spanner_Mutation__) method in the `DatabaseClient` class writes the mutations. All mutations in a single batch are applied atomically.

This code shows how to write the data using mutations:

```java
static final List<Singer> SINGERS =
    Arrays.asList(
        new Singer(1, "Marc", "Richards"),
        new Singer(2, "Catalina", "Smith"),
        new Singer(3, "Alice", "Trentor"),
        new Singer(4, "Lea", "Martin"),
        new Singer(5, "David", "Lomond"));

static final List<Album> ALBUMS =
    Arrays.asList(
        new Album(1, 1, "Total Junk"),
        new Album(1, 2, "Go, Go, Go"),
        new Album(2, 1, "Green"),
        new Album(2, 2, "Forever Hold Your Peace"),
        new Album(2, 3, "Terrified"));
static void writeExampleData(DatabaseClient dbClient) {
  List<Mutation> mutations = new ArrayList<>();
  for (Singer singer : SINGERS) {
    mutations.add(
        Mutation.newInsertBuilder("Singers")
            .set("SingerId")
            .to(singer.singerId)
            .set("FirstName")
            .to(singer.firstName)
            .set("LastName")
            .to(singer.lastName)
            .build());
  }
  for (Album album : ALBUMS) {
    mutations.add(
        Mutation.newInsertBuilder("Albums")
            .set("SingerId")
            .to(album.singerId)
            .set("AlbumId")
            .to(album.albumId)
            .set("AlbumTitle")
            .to(album.albumTitle)
            .build());
  }
  dbClient.write(mutations);
}
```

Run the sample using the `write` argument.

```
java -jar target/spanner-snippets/spanner-google-cloud-samples.jar \
    write test-instance example-db
```

You should see the command run successfully.

> **Note:** There are limits to commit size. See [CRUD limit](https://docs.cloud.google.com/spanner/quotas#limits-for) for more information.

## Query data using SQL

Spanner supports a SQL interface for reading data, which you can access on the command line using the Google Cloud CLI or programmatically using the Spanner client library for Java.

### On the command line

Execute the following SQL statement to read the values of all columns from the `Albums` table:

### GoogleSQL

```
gcloud spanner databases execute-sql example-db --instance=test-instance \
    --sql='SELECT SingerId, AlbumId, AlbumTitle FROM Albums'
```

### GoogleSQL Pipe syntax

```
gcloud spanner databases execute-sql example-db --instance=test-instance \
    --sql='FROM Albums |> SELECT SingerId, AlbumId, AlbumTitle'
```

> **Note:** For the GoogleSQL reference, see [Standard SQL syntax in GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax) or [Pipe syntax in GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/pipe-syntax) ; for PostgreSQL reference, see [PostgreSQL lexical structure and syntax](https://docs.cloud.google.com/spanner/docs/reference/postgresql/lexical) .

The result shows:

```
SingerId AlbumId AlbumTitle
1        1       Total Junk
1        2       Go, Go, Go
2        1       Green
2        2       Forever Hold Your Peace
2        3       Terrified
```

### Use the Spanner client library for Java

In addition to executing a SQL statement on the command line, you can issue the same SQL statement programmatically using the Spanner client library for Java.

The following methods and classes are used to run the SQL query:

- The [`singleUse()`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.DatabaseClient#com_google_cloud_spanner_DatabaseClient_singleUse__) method in the `DatabaseClient` class: use this to read the value of one or more columns from one or more rows in a Spanner table. `singleUse()` returns a `ReadContext` object, which is used for running a read or SQL statement.
- The [`executeQuery()`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.ReadContext#com_google_cloud_spanner_ReadContext_executeQuery_com_google_cloud_spanner_Statement_com_google_cloud_spanner_Options_QueryOption____) method of the `ReadContext` class: use this method to execute a query against a database.
- The [`Statement`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.Statement) class: use this to construct a SQL string.
- The [`ResultSet`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.ResultSet) class: use this to access the data returned by a SQL statement or read call.

Here's how to issue the query and access the data:

```java
static void query(DatabaseClient dbClient) {
  try (ResultSet resultSet =
      dbClient
          .singleUse() // Execute a single read or query against Cloud Spanner.
          .executeQuery(Statement.of("SELECT SingerId, AlbumId, AlbumTitle FROM Albums"))) {
    while (resultSet.next()) {
      System.out.printf(
          "%d %d %s\n", resultSet.getLong(0), resultSet.getLong(1), resultSet.getString(2));
    }
  }
}
```

Run the sample using the `query` argument.

```
java -jar target/spanner-snippets/spanner-google-cloud-samples.jar \
    query test-instance example-db
```

You should see the following result:

```
1 1 Total Junk
1 2 Go, Go, Go
2 1 Green
2 2 Forever Hold Your Peace
2 3 Terrified
```

### Query using a SQL parameter

If your application has a frequently executed query, you can improve its performance by parameterizing it. The resulting parametric query can be cached and reused, which reduces compilation costs. For more information, see [Use query parameters to speed up frequently executed queries](https://docs.cloud.google.com/spanner/docs/sql-best-practices#query-parameters) .

Here is an example of using a parameter in the `WHERE` clause to query records containing a specific value for `LastName` .

### GoogleSQL

```java
static void queryWithParameter(DatabaseClient dbClient) {
  Statement statement =
      Statement.newBuilder(
              "SELECT SingerId, FirstName, LastName "
                  + "FROM Singers "
                  + "WHERE LastName = @lastName")
          .bind("lastName")
          .to("Garcia")
          .build();
  try (ResultSet resultSet = dbClient.singleUse().executeQuery(statement)) {
    while (resultSet.next()) {
      System.out.printf(
          "%d %s %s\n",
          resultSet.getLong("SingerId"),
          resultSet.getString("FirstName"),
          resultSet.getString("LastName"));
    }
  }
}
```

### PostgreSQL

```java
static void queryWithParameter(DatabaseClient dbClient) {
  Statement statement =
      Statement.newBuilder(
              "SELECT singerid AS \"SingerId\", "
                  + "firstname as \"FirstName\", lastname as \"LastName\" "
                  + "FROM Singers "
                  + "WHERE LastName = $1")
          .bind("p1")
          .to("Garcia")
          .build();
  try (ResultSet resultSet = dbClient.singleUse().executeQuery(statement)) {
    while (resultSet.next()) {
      System.out.printf(
          "%d %s %s\n",
          resultSet.getLong("SingerId"),
          resultSet.getString("FirstName"),
          resultSet.getString("LastName"));
    }
  }
}
```

Run the sample using the queryWithParameter argument.

```
java -jar target/spanner-snippets/spanner-google-cloud-samples.jar \
    querywithparameter test-instance example-db
```

You should see the following result:

```
12 Melissa Garcia
```

## Read data using the read API

In addition to Spanner's SQL interface, Spanner also supports a read interface.

Use the [`read()`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.ReadContext#com_google_cloud_spanner_ReadContext_read_java_lang_String_com_google_cloud_spanner_KeySet_java_lang_Iterable_java_lang_String__com_google_cloud_spanner_Options_ReadOption____) method of the `ReadContext` class to read rows from the database. Use a [`KeySet`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.KeySet) object to define a collection of keys and key ranges to read.

Here's how to read the data:

```java
static void read(DatabaseClient dbClient) {
  try (ResultSet resultSet =
      dbClient
          .singleUse()
          .read(
              "Albums",
              KeySet.all(), // Read all rows in a table.
              Arrays.asList("SingerId", "AlbumId", "AlbumTitle"))) {
    while (resultSet.next()) {
      System.out.printf(
          "%d %d %s\n", resultSet.getLong(0), resultSet.getLong(1), resultSet.getString(2));
    }
  }
}
```

Run the sample using the `read` argument.

```
java -jar target/spanner-snippets/spanner-google-cloud-samples.jar \
    read test-instance example-db
```

You should see output similar to:

```
1 1 Total Junk
1 2 Go, Go, Go
2 1 Green
2 2 Forever Hold Your Peace
2 3 Terrified
```

## Update the database schema

Assume you need to add a new column called `MarketingBudget` to the `Albums` table. Adding a new column to an existing table requires an update to your database schema. Spanner supports schema updates to a database while the database continues to serve traffic. Schema updates don't require taking the database offline and they don't lock entire tables or columns; you can continue writing data to the database during the schema update. Read more about supported schema updates and schema change performance in [Make schema updates](https://docs.cloud.google.com/spanner/docs/schema-updates) .

### Add a column

You can add a column on the command line using the Google Cloud CLI or programmatically using the Spanner client library for Java.

#### On the command line

Use the following [`ALTER TABLE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter_table) command to add the new column to the table:

### GoogleSQL

```
gcloud spanner databases ddl update example-db --instance=test-instance \
    --ddl='ALTER TABLE Albums ADD COLUMN MarketingBudget INT64'
```

### PostgreSQL

```
gcloud spanner databases ddl update example-db --instance=test-instance \
    --ddl='ALTER TABLE Albums ADD COLUMN MarketingBudget BIGINT'
```

You should see:

```
Schema updating...done.
```

#### Use the Spanner client library for Java

Use the [`updateDatabaseDdl()`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.DatabaseAdminClient#com_google_cloud_spanner_DatabaseAdminClient_updateDatabaseDdl_java_lang_String_java_lang_String_java_lang_Iterable_java_lang_String__java_lang_String_) method of the `DatabaseAdminClient` class to modify the schema:

### GoogleSQL

```java
static void addMarketingBudget(DatabaseAdminClient adminClient, DatabaseName databaseName) {
  try {
    // Initiate the request which returns an OperationFuture.
    adminClient.updateDatabaseDdlAsync(
        databaseName,
        Arrays.asList("ALTER TABLE Albums ADD COLUMN MarketingBudget INT64")).get();
    System.out.println("Added MarketingBudget column");
  } catch (ExecutionException e) {
    // If the operation failed during execution, expose the cause.
    throw (SpannerException) e.getCause();
  } catch (InterruptedException e) {
    // Throw when a thread is waiting, sleeping, or otherwise occupied,
    // and the thread is interrupted, either before or during the activity.
    throw SpannerExceptionFactory.propagateInterrupt(e);
  }
}
```

### PostgreSQL

```java
static void addMarketingBudget(DatabaseAdminClient adminClient, DatabaseName databaseName) {
  try {
    // Initiate the request which returns an OperationFuture.
    adminClient.updateDatabaseDdlAsync(
        databaseName,
        Arrays.asList("ALTER TABLE Albums ADD COLUMN MarketingBudget bigint")).get();
    System.out.println("Added MarketingBudget column");
  } catch (ExecutionException e) {
    // If the operation failed during execution, expose the cause.
    throw (SpannerException) e.getCause();
  } catch (InterruptedException e) {
    // Throw when a thread is waiting, sleeping, or otherwise occupied,
    // and the thread is interrupted, either before or during the activity.
    throw SpannerExceptionFactory.propagateInterrupt(e);
  }
}
```

Run the sample using the `addmarketingbudget` argument.

```
java -jar target/spanner-snippets/spanner-google-cloud-samples.jar \
    addmarketingbudget test-instance example-db
```

You should see:

```
Added MarketingBudget column.
```

### Write data to the new column

The following code writes data to the new column. It sets `MarketingBudget` to `100000` for the row keyed by `Albums(1, 1)` and to `500000` for the row keyed by `Albums(2, 2)` .

```java
static void update(DatabaseClient dbClient) {
  // Mutation can be used to update/insert/delete a single row in a table. Here we use
  // newUpdateBuilder to create update mutations.
  List<Mutation> mutations =
      Arrays.asList(
          Mutation.newUpdateBuilder("Albums")
              .set("SingerId")
              .to(1)
              .set("AlbumId")
              .to(1)
              .set("MarketingBudget")
              .to(100000)
              .build(),
          Mutation.newUpdateBuilder("Albums")
              .set("SingerId")
              .to(2)
              .set("AlbumId")
              .to(2)
              .set("MarketingBudget")
              .to(500000)
              .build());
  // This writes all the mutations to Cloud Spanner atomically.
  dbClient.write(mutations);
}
```

Run the sample using the `update` argument.

```
java -jar target/spanner-snippets/spanner-google-cloud-samples.jar \
    update test-instance example-db
```

You can also execute a SQL query or a read call to fetch the values that you just wrote.

Here's the code to execute the query:

### GoogleSQL

```java
static void queryMarketingBudget(DatabaseClient dbClient) {
  // Rows without an explicit value for MarketingBudget will have a MarketingBudget equal to
  // null. A try-with-resource block is used to automatically release resources held by
  // ResultSet.
  try (ResultSet resultSet =
      dbClient
          .singleUse()
          .executeQuery(Statement.of("SELECT SingerId, AlbumId, MarketingBudget FROM Albums"))) {
    while (resultSet.next()) {
      System.out.printf(
          "%d %d %s\n",
          resultSet.getLong("SingerId"),
          resultSet.getLong("AlbumId"),
          // We check that the value is non null. ResultSet getters can only be used to retrieve
          // non null values.
          resultSet.isNull("MarketingBudget") ? "NULL" : resultSet.getLong("MarketingBudget"));
    }
  }
}
```

### PostgreSQL

```java
static void queryMarketingBudget(DatabaseClient dbClient) {
  // Rows without an explicit value for MarketingBudget will have a MarketingBudget equal to
  // null. A try-with-resource block is used to automatically release resources held by
  // ResultSet.
  try (ResultSet resultSet =
      dbClient
          .singleUse()
          .executeQuery(Statement.of("SELECT singerid as \"SingerId\", "
              + "albumid as \"AlbumId\", marketingbudget as \"MarketingBudget\" "
              + "FROM Albums"))) {
    while (resultSet.next()) {
      System.out.printf(
          "%d %d %s\n",
          resultSet.getLong("SingerId"),
          resultSet.getLong("AlbumId"),
          // We check that the value is non null. ResultSet getters can only be used to retrieve
          // non null values.
          resultSet.isNull("MarketingBudget") ? "NULL" :
              resultSet.getLong("MarketingBudget"));
    }
  }
}
```

To execute this query, run the sample using the `querymarketingbudget` argument.

```
java -jar target/spanner-snippets/spanner-google-cloud-samples.jar \
    querymarketingbudget test-instance example-db
```

You should see:

```
1 1 100000
1 2 NULL
2 1 NULL
2 2 500000
2 3 NULL
```

## Update data

You can update data using DML in a read-write transaction.

You use the [`executeUpdate()`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.TransactionContext#com_google_cloud_spanner_TransactionContext_executeUpdate_com_google_cloud_spanner_Statement_com_google_cloud_spanner_Options_UpdateOption____) method to execute a DML statement.

### GoogleSQL

```java
static void writeWithTransactionUsingDml(DatabaseClient dbClient) {
  dbClient
      .readWriteTransaction()
      .run(transaction -> {
        // Transfer marketing budget from one album to another. We do it in a transaction to
        // ensure that the transfer is atomic.
        String sql1 =
            "SELECT MarketingBudget from Albums WHERE SingerId = 2 and AlbumId = 2";
        ResultSet resultSet = transaction.executeQuery(Statement.of(sql1));
        long album2Budget = 0;
        while (resultSet.next()) {
          album2Budget = resultSet.getLong("MarketingBudget");
        }
        // Transaction will only be committed if this condition still holds at the time of
        // commit. Otherwise it will be aborted and the callable will be rerun by the
        // client library.
        long transfer = 200000;
        if (album2Budget >= transfer) {
          String sql2 =
              "SELECT MarketingBudget from Albums WHERE SingerId = 1 and AlbumId = 1";
          ResultSet resultSet2 = transaction.executeQuery(Statement.of(sql2));
          long album1Budget = 0;
          while (resultSet2.next()) {
            album1Budget = resultSet2.getLong("MarketingBudget");
          }
          album1Budget += transfer;
          album2Budget -= transfer;
          Statement updateStatement =
              Statement.newBuilder(
                      "UPDATE Albums "
                          + "SET MarketingBudget = @AlbumBudget "
                          + "WHERE SingerId = 1 and AlbumId = 1")
                  .bind("AlbumBudget")
                  .to(album1Budget)
                  .build();
          transaction.executeUpdate(updateStatement);
          Statement updateStatement2 =
              Statement.newBuilder(
                      "UPDATE Albums "
                          + "SET MarketingBudget = @AlbumBudget "
                          + "WHERE SingerId = 2 and AlbumId = 2")
                  .bind("AlbumBudget")
                  .to(album2Budget)
                  .build();
          transaction.executeUpdate(updateStatement2);
        }
        return null;
      });
}
```

### PostgreSQL

```java
static void writeWithTransactionUsingDml(DatabaseClient dbClient) {
  dbClient
      .readWriteTransaction()
      .run(transaction -> {
        // Transfer marketing budget from one album to another. We do it in a transaction to
        // ensure that the transfer is atomic.
        String sql1 =
            "SELECT marketingbudget as \"MarketingBudget\" from Albums WHERE "
                + "SingerId = 2 and AlbumId = 2";
        ResultSet resultSet = transaction.executeQuery(Statement.of(sql1));
        long album2Budget = 0;
        while (resultSet.next()) {
          album2Budget = resultSet.getLong("MarketingBudget");
        }
        // Transaction will only be committed if this condition still holds at the time of
        // commit. Otherwise it will be aborted and the callable will be rerun by the
        // client library.
        long transfer = 200000;
        if (album2Budget >= transfer) {
          String sql2 =
              "SELECT marketingbudget as \"MarketingBudget\" from Albums WHERE "
                  + "SingerId = 1 and AlbumId = 1";
          ResultSet resultSet2 = transaction.executeQuery(Statement.of(sql2));
          long album1Budget = 0;
          while (resultSet2.next()) {
            album1Budget = resultSet2.getLong("MarketingBudget");
          }
          album1Budget += transfer;
          album2Budget -= transfer;
          Statement updateStatement =
              Statement.newBuilder(
                      "UPDATE Albums "
                          + "SET MarketingBudget = $1 "
                          + "WHERE SingerId = 1 and AlbumId = 1")
                  .bind("p1")
                  .to(album1Budget)
                  .build();
          transaction.executeUpdate(updateStatement);
          Statement updateStatement2 =
              Statement.newBuilder(
                      "UPDATE Albums "
                          + "SET MarketingBudget = $1 "
                          + "WHERE SingerId = 2 and AlbumId = 2")
                  .bind("p1")
                  .to(album2Budget)
                  .build();
          transaction.executeUpdate(updateStatement2);
        }
        return null;
      });
}
```

Run the sample using the `writewithtransactionusingdml` argument.

```
java -jar target/spanner-snippets/spanner-google-cloud-samples.jar \
    writewithtransactionusingdml test-instance example-db
```

> **Note:** You can also [update data using mutations](https://docs.cloud.google.com/spanner/docs/modify-mutation-api#updating_rows_in_a_table) .

## Use a secondary index

Suppose you wanted to fetch all rows of `Albums` that have `AlbumTitle` values in a certain range. You could read all values from the `AlbumTitle` column using a SQL statement or a read call, and then discard the rows that don't meet the criteria, but doing this full table scan is expensive, especially for tables with a lot of rows. Instead you can speed up the retrieval of rows when searching by non-primary key columns by creating a [secondary index](https://docs.cloud.google.com/spanner/docs/secondary-indexes) on the table.

Adding a secondary index to an existing table requires a schema update. Like other schema updates, Spanner supports adding an index while the database continues to serve traffic. Spanner automatically backfills the index with your existing data. Backfills might take a few minutes to complete, but you don't need to take the database offline or avoid writing to the indexed table during this process. For more details, see [Add a secondary index](https://docs.cloud.google.com/spanner/docs/secondary-indexes#adding_an_index) .

After you add a secondary index, Spanner automatically uses it for SQL queries that are likely to run faster with the index. If you use the read interface, you must specify the index that you want to use.

### Add a secondary index

You can add an index on the command line using the gcloud CLI or programmatically using the Spanner client library for Java.

#### On the command line

Use the following [`CREATE INDEX`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create_index) command to add an index to the database:

```
gcloud spanner databases ddl update example-db --instance=test-instance \
    --ddl='CREATE INDEX AlbumsByAlbumTitle ON Albums(AlbumTitle)'
```

You should see:

```
Schema updating...done.
```

#### Using the Spanner client library for Java

Use the [`updateDatabaseDdl()`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.DatabaseAdminClient#com_google_cloud_spanner_DatabaseAdminClient_updateDatabaseDdl_java_lang_String_java_lang_String_java_lang_Iterable_java_lang_String__java_lang_String_) method of the `DatabaseAdminClient` class to add an index:

```java
static void addIndex(DatabaseAdminClient adminClient, DatabaseName databaseName) {
  try {
    // Initiate the request which returns an OperationFuture.
    adminClient.updateDatabaseDdlAsync(
        databaseName,
        Arrays.asList("CREATE INDEX AlbumsByAlbumTitle ON Albums(AlbumTitle)")).get();
    System.out.println("Added AlbumsByAlbumTitle index");
  } catch (ExecutionException e) {
    // If the operation failed during execution, expose the cause.
    throw (SpannerException) e.getCause();
  } catch (InterruptedException e) {
    // Throw when a thread is waiting, sleeping, or otherwise occupied,
    // and the thread is interrupted, either before or during the activity.
    throw SpannerExceptionFactory.propagateInterrupt(e);
  }
}
```

Run the sample using the `addindex` argument.

```
java -jar target/spanner-snippets/spanner-google-cloud-samples.jar \
    addindex test-instance example-db
```

Adding an index can take a few minutes. After the index is added, you should see:

```
Added the AlbumsByAlbumTitle index.
```

### Read using the index

For SQL queries, Spanner automatically uses an appropriate index. In the read interface, you must specify the index in your request.

To use the index in the read interface, use the [`readUsingIndex()`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.ReadContext#com_google_cloud_spanner_ReadContext_readUsingIndex_java_lang_String_java_lang_String_com_google_cloud_spanner_KeySet_java_lang_Iterable_java_lang_String__com_google_cloud_spanner_Options_ReadOption____) method of the `ReadContext` class.

The following code fetches all `AlbumId` , and `AlbumTitle` columns from the `AlbumsByAlbumTitle` index.

```java
static void readUsingIndex(DatabaseClient dbClient) {
  try (ResultSet resultSet =
      dbClient
          .singleUse()
          .readUsingIndex(
              "Albums",
              "AlbumsByAlbumTitle",
              KeySet.all(),
              Arrays.asList("AlbumId", "AlbumTitle"))) {
    while (resultSet.next()) {
      System.out.printf("%d %s\n", resultSet.getLong(0), resultSet.getString(1));
    }
  }
}
```

Run the sample using the `readindex` argument.

```
java -jar target/spanner-snippets/spanner-google-cloud-samples.jar \
    readindex test-instance example-db
```

You should see:

```
2 Forever Hold Your Peace
2 Go, Go, Go
1 Green
3 Terrified
1 Total Junk
```

### Add an index for index-only reads

You might have noticed that the previous read example doesn't include reading the `MarketingBudget` column. This is because Spanner's read interface doesn't support the ability to join an index with a data table to look up values that are not stored in the index.

Create an alternate definition of `AlbumsByAlbumTitle` that stores a copy of `MarketingBudget` in the index.

#### On the command line

### GoogleSQL

```
gcloud spanner databases ddl update example-db --instance=test-instance \
    --ddl='CREATE INDEX AlbumsByAlbumTitle2 ON Albums(AlbumTitle) STORING (MarketingBudget)
```

### PostgreSQL

```
gcloud spanner databases ddl update example-db --instance=test-instance \
    --ddl='CREATE INDEX AlbumsByAlbumTitle2 ON Albums(AlbumTitle) INCLUDE (MarketingBudget)
```

Adding an index can take a few minutes. After the index is added, you should see:

```
Schema updating...done.
```

#### Using the Spanner client library for Java

Use the [`updateDatabaseDdl()`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.DatabaseAdminClient#com_google_cloud_spanner_DatabaseAdminClient_updateDatabaseDdl_java_lang_String_java_lang_String_java_lang_Iterable_java_lang_String__java_lang_String_) method of the `DatabaseAdminClient` class to add an index with a `STORING` clause for GoogleSQL and `INCLUDE` clause for PostgreSQL:

### GoogleSQL

```java
static void addStoringIndex(DatabaseAdminClient adminClient, DatabaseName databaseName) {
  try {
    // Initiate the request which returns an OperationFuture.
    adminClient.updateDatabaseDdlAsync(
        databaseName,
        Arrays.asList(
            "CREATE INDEX AlbumsByAlbumTitle2 ON Albums(AlbumTitle) "
                + "STORING (MarketingBudget)")).get();
    System.out.println("Added AlbumsByAlbumTitle2 index");
  } catch (ExecutionException e) {
    // If the operation failed during execution, expose the cause.
    throw (SpannerException) e.getCause();
  } catch (InterruptedException e) {
    // Throw when a thread is waiting, sleeping, or otherwise occupied,
    // and the thread is interrupted, either before or during the activity.
    throw SpannerExceptionFactory.propagateInterrupt(e);
  }
}
```

### PostgreSQL

```java
static void addStoringIndex(DatabaseAdminClient adminClient, DatabaseName databaseName) {
  try {
    // Initiate the request which returns an OperationFuture.
    adminClient.updateDatabaseDdlAsync(
        databaseName,
        Arrays.asList(
            "CREATE INDEX AlbumsByAlbumTitle2 ON Albums(AlbumTitle) "
                + "INCLUDE (MarketingBudget)")).get();
    System.out.println("Added AlbumsByAlbumTitle2 index");
  } catch (ExecutionException e) {
    // If the operation failed during execution, expose the cause.
    throw (SpannerException) e.getCause();
  } catch (InterruptedException e) {
    // Throw when a thread is waiting, sleeping, or otherwise occupied,
    // and the thread is interrupted, either before or during the activity.
    throw SpannerExceptionFactory.propagateInterrupt(e);
  }
}
```

Run the sample using the `addstoringindex` argument.

```
java -jar target/spanner-snippets/spanner-google-cloud-samples.jar \
    addstoringindex test-instance example-db
```

Adding an index can take a few minutes. After the index is added, you should see:

```
Added AlbumsByAlbumTitle2 index
```

Now you can execute a read that fetches all `AlbumId` , `AlbumTitle` , and `MarketingBudget` columns from the `AlbumsByAlbumTitle2` index:

```java
static void readStoringIndex(DatabaseClient dbClient) {
  // We can read MarketingBudget also from the index since it stores a copy of MarketingBudget.
  try (ResultSet resultSet =
      dbClient
          .singleUse()
          .readUsingIndex(
              "Albums",
              "AlbumsByAlbumTitle2",
              KeySet.all(),
              Arrays.asList("AlbumId", "AlbumTitle", "MarketingBudget"))) {
    while (resultSet.next()) {
      System.out.printf(
          "%d %s %s\n",
          resultSet.getLong(0),
          resultSet.getString(1),
          resultSet.isNull("MarketingBudget") ? "NULL" : resultSet.getLong("MarketingBudget"));
    }
  }
}
```

Run the sample using the `readstoringindex` argument.

```
java -jar target/spanner-snippets/spanner-google-cloud-samples.jar \
    readstoringindex test-instance example-db
```

You should see output similar to:

```
2 Forever Hold Your Peace 300000
2 Go, Go, Go NULL
1 Green NULL
3 Terrified NULL
1 Total Junk 300000
```

## Retrieve data using read-only transactions

Suppose you want to execute more than one read at the same timestamp. [Read-only transactions](https://docs.cloud.google.com/spanner/docs/transactions#read-only_transactions) observe a consistent prefix of the transaction commit history, so your application always gets consistent data. Use a [`ReadOnlyTransaction`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.ReadOnlyTransaction) object for executing read-only transactions. Use the [`readOnlyTransaction()`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.DatabaseClient#com_google_cloud_spanner_DatabaseClient_readOnlyTransaction__) method of the `DatabaseClient` class to get a `ReadOnlyTransaction` object.

The following shows how to run a query and perform a read in the same read-only transaction:

```java
static void readOnlyTransaction(DatabaseClient dbClient) {
  // ReadOnlyTransaction must be closed by calling close() on it to release resources held by it.
  // We use a try-with-resource block to automatically do so.
  try (ReadOnlyTransaction transaction = dbClient.readOnlyTransaction()) {
    try (ResultSet queryResultSet =
        transaction.executeQuery(
            Statement.of("SELECT SingerId, AlbumId, AlbumTitle FROM Albums"))) {
      while (queryResultSet.next()) {
        System.out.printf(
            "%d %d %s\n",
            queryResultSet.getLong(0), queryResultSet.getLong(1), queryResultSet.getString(2));
      }
    } // queryResultSet.close() is automatically called here
    try (ResultSet readResultSet =
        transaction.read(
          "Albums", KeySet.all(), Arrays.asList("SingerId", "AlbumId", "AlbumTitle"))) {
      while (readResultSet.next()) {
        System.out.printf(
            "%d %d %s\n",
            readResultSet.getLong(0), readResultSet.getLong(1), readResultSet.getString(2));
      }
    } // readResultSet.close() is automatically called here
  } // transaction.close() is automatically called here
}
```

Run the sample using the `readonlytransaction` argument.

```
java -jar target/spanner-snippets/spanner-google-cloud-samples.jar \
    readonlytransaction test-instance example-db
```

You should see output similar to:

```
2 2 Forever Hold Your Peace
1 2 Go, Go, Go
2 1 Green
2 3 Terrified
1 1 Total Junk
1 1 Total Junk
1 2 Go, Go, Go
2 1 Green
2 2 Forever Hold Your Peace
2 3 Terrified
```

## Cleanup

To avoid incurring additional charges to your Cloud Billing account for the resources used in this tutorial, drop the database and delete the instance that you created.

### Delete the database

If you delete an instance, all databases within it are automatically deleted. This step shows how to delete a database without deleting an instance (you would still incur charges for the instance).

#### On the command line

```
gcloud spanner databases delete example-db --instance=test-instance
```

#### Using the Google Cloud console

1.  Go to the **Spanner Instances** page in the Google Cloud console.

2.  Click the instance.

3.  Click the database that you want to delete.

4.  In the **Database details** page, click **Delete** .

5.  Confirm that you want to delete the database and click **Delete** .

### Delete the instance

Deleting an instance automatically drops all databases created in that instance.

#### On the command line

```
gcloud spanner instances delete test-instance
```

#### Using the Google Cloud console

1.  Go to the **Spanner Instances** page in the Google Cloud console.

2.  Click your instance.

3.  Click **Delete** .

4.  Confirm that you want to delete the instance and click **Delete** .

## What's next

- Try the [Spring Data module for Spanner](https://docs.cloud.google.com/spanner/docs/adding-spring) .

<!-- -->

- Learn how to [access Spanner with a virtual machine instance](https://docs.cloud.google.com/spanner/docs/configure-virtual-machine-instance) .

- Learn about authorization and authentication credentials in [Authenticate to Cloud services using client libraries](https://docs.cloud.google.com/docs/authentication/getting-started) .

- Learn more about Spanner [Schema design best practices](https://docs.cloud.google.com/spanner/docs/schema-design) .
