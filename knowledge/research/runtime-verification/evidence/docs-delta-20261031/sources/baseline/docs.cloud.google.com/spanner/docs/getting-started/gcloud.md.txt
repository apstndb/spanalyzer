---
name: documents/docs.cloud.google.com/spanner/docs/getting-started/gcloud
uri: https://docs.cloud.google.com/spanner/docs/getting-started/gcloud
title: Create and query a database using the Google Cloud CLI
description: 'Get started with Spanner using the Google Cloud CLI: create instances, databases, and run queries from the command line.'
data_source: docs.cloud.google.com
---

## Objectives

This tutorial walks you through the following steps using the [gcloud CLI](https://docs.cloud.google.com/sdk) :

- Create a Spanner instance, database, and schema
- Write data to the database and execute SQL queries that data
- Clean up by deleting the database and instance

The procedures on this page apply to both GoogleSQL-dialect databases and PostgreSQL-dialect databases.

For the complete Spanner `gcloud` reference, see [gcloud](https://docs.cloud.google.com/sdk/gcloud/reference/spanner) .

> **Note:** To explore Spanner using a 90-day free trial instance, see [Create a Spanner free trial instance](https://docs.cloud.google.com/spanner/docs/free-trial-quickstart) .

## Pricing

This tutorial uses Spanner, which is a billable component of Google Cloud. For information on the cost of using Spanner, see [Pricing](https://cloud.google.com/spanner/pricing) .

## Before you begin

Complete the steps described in [Install the gcloud CLI and set up the Cloud Spanner API](https://docs.cloud.google.com/spanner/docs/getting-started/set-up) , which covers creating and setting a default Google Cloud project, enabling billing, enabling the Cloud Spanner API, and setting up OAuth 2.0 to get authentication credentials to use the Cloud Spanner API.

In particular, ensure that you run [`gcloud auth application-default login`](https://docs.cloud.google.com/sdk/gcloud/reference/auth/application-default/login) to set up your local development environment with authentication credentials.

## Set a default project

If you haven't already done so, set the ID of a Google Cloud project as the default project for the Google Cloud CLI:

```
gcloud config set project PROJECT_ID
```

If you don't set the default project, you must pass `--project PROJECT_ID` to each of the commands below as the first argument to `gcloud` . For example:

```
gcloud --project=PROJECT_ID spanner instance-configs list
```

## Instances

When you first use Spanner, you must create an instance, which is an allocation of resources that are used by Spanner databases. When you create an instance, you choose where your data is stored and how much [compute capacity](https://docs.cloud.google.com/spanner/docs/compute-capacity) the instance has.

## Instances and instance configurations

To create an instance, you must select an instance configuration, which is like a blueprint for your instance that defines the geographic placement and replication of your Spanner data.

### List instance configurations

When you create an instance, you specify an *instance configuration* , which defines the geographic placement and replication of your databases in that instance. You can choose a regional configuration, which stores data in one region, a dual-region, which stores data in two regions in the same country, or a multi-region configuration, which distributes data across multiple regions. For more information, see the [Instances overview](https://docs.cloud.google.com/spanner/docs/instances) .

To see the set of instance configurations that are available for your project:

```
gcloud spanner instance-configs list
```

You should see a list of regional, dual-region, and multi-region configurations.

### Create an instance

To create an instance named `test-instance` with the display name `My Instance` using the regional instance configuration `regional-us-central1` with 1 nodes:

```
gcloud spanner instances create test-instance --config=regional-us-central1 \
    --description="My Instance" --nodes=1
```

In the command above, the instance name is set to `test-instance` and `--description` sets the display name of the instance. Both of these values must be unique within a Google Cloud Platform project.

> **Note:** Use the instance ID, not the display name, when referring to an instance in `gcloud` commands.

### Set the default instance

You can set the default instance that Spanner uses when you have not specified an instance in your command. To set the default instance:

```
gcloud config set spanner/instance test-instance
```

## Create a database

Create a database named `example-db` . The database dialect defaults to GoogleSQL.

### GoogleSQL

```
gcloud spanner databases create example-db
```

### PostgreSQL

```
gcloud spanner databases create example-db --database-dialect=POSTGRESQL
```

## Update the schema

Use Spanner's [Data Definition Language](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language) (DDL) to create, alter, or drop tables, and to create or drop indexes.

Let's create two tables:

### GoogleSQL

```
gcloud spanner databases ddl update example-db \
--ddl='CREATE TABLE Singers ( SingerId INT64 NOT NULL, FirstName STRING(1024), LastName STRING(1024), SingerInfo BYTES(MAX) ) PRIMARY KEY (SingerId)'

gcloud spanner databases ddl update example-db \
--ddl='CREATE TABLE Albums ( SingerId INT64 NOT NULL, AlbumId INT64 NOT NULL, AlbumTitle STRING(MAX)) PRIMARY KEY (SingerId, AlbumId), INTERLEAVE IN PARENT Singers ON DELETE CASCADE'
```

### PostgreSQL

```
gcloud spanner databases ddl update example-db \
--ddl='CREATE TABLE Singers ( SingerId bigint NOT NULL, FirstName varchar(1024), LastName varchar(1024), SingerInfo bytea, PRIMARY KEY (SingerId) )'

gcloud spanner databases ddl update example-db \
--ddl='CREATE TABLE Albums ( SingerId bigint NOT NULL, AlbumId bigint NOT NULL, AlbumTitle varchar, PRIMARY KEY (SingerId, AlbumId) ) INTERLEAVE IN PARENT Singers ON DELETE CASCADE'
```

To check the progress of the operation, use [`gcloud spanner operations describe`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/operations/describe) . This command requires the operation ID.

Get the operation ID:

```
gcloud spanner operations list --instance="test-instance" \
--database=DATABASE-NAME --type=DATABASE_UPDATE_DDL
```

Replace ` DATABASE-NAME ` with the name of the database.

Run `gcloud spanner operations describe` :

```
gcloud spanner operations describe \
  --instance="test-instance" \
  --database="example-db" \
  projects/PROJECT-NAME/instances/test-instance/databases/example-db/operations/OPERATION-ID
```

Replace the following:

- ` PROJECT-NAME ` : The project name.
- ` OPERATION-ID ` : The operation ID of the operation that you want to check.

The output looks similar to the following:

```
done: true
metadata:
...
progress:
- endTime: '2022-03-01T00:28:06.691403Z'
  progressPercent: 100
  startTime: '2022-03-01T00:28:04.221401Z'
- endTime: '2022-03-01T00:28:17.624588Z'
  startTime: '2022-03-01T00:28:06.691403Z'
  progressPercent: 100
...
```

## Write data

Let's add some sample data to our database

### GoogleSQL

```
gcloud spanner rows insert --database=example-db \
  --table=Singers \
  --data=SingerId=1,FirstName=Marc,LastName=Richards

gcloud spanner rows insert --database=example-db \
  --table=Singers \
  --data=SingerId=2,FirstName=Catalina,LastName=Smith

gcloud spanner rows insert --database=example-db \
  --table=Singers \
  --data=SingerId=3,FirstName=Alice,LastName=Trentor

gcloud spanner rows insert --database=example-db \
  --table=Albums \
  --data=SingerId=1,AlbumId=1,AlbumTitle="Total Junk"

gcloud spanner rows insert --database=example-db \
  --table=Albums \
  --data=SingerId=2,AlbumId=1,AlbumTitle="Green"

gcloud spanner rows insert --database=example-db \
  --table=Albums \
  --data=^:^SingerId=2:AlbumId=2:AlbumTitle="Go, Go, Go"
```

By default, a comma is used to delimit items in lists. In the last insert command, we specified a colon ( `^:^` ) as the delimiter so that we could use a comma in the album title.

### PostgreSQL

> **Note:** There is a known issue where the gcloud CLI can't look up table names when using mutations to write data to a PostgreSQL database. Therefore, you must use DML to insert data.

```
gcloud spanner databases execute-sql example-db \
  --sql="INSERT INTO Singers (SingerId, FirstName, LastName) VALUES (1, 'Marc', 'Richards')"

gcloud spanner databases execute-sql example-db \
  --sql="INSERT INTO Singers (SingerId, FirstName, LastName) VALUES (2, 'Catalina', 'Smith')"

gcloud spanner databases execute-sql example-db   \
  --sql="INSERT INTO Singers (SingerId, FirstName, LastName) VALUES (3, 'Alice', 'Trentor')"

gcloud spanner databases execute-sql example-db   \
  --sql="INSERT INTO Albums (SingerId, AlbumId, AlbumTitle) VALUES (1, 1, 'Total Junk')"

gcloud spanner databases execute-sql example-db   \
  --sql="INSERT INTO Albums (SingerId, AlbumId, AlbumTitle) VALUES (2, 1, 'Green')"

gcloud spanner databases execute-sql example-db   \
  --sql="INSERT INTO Albums (SingerId, AlbumId, AlbumTitle) VALUES (2, 2, 'Go, Go, Go')"
```

## Query data using SQL

Execute a query on the command line:

```
gcloud spanner databases execute-sql example-db \
    --sql='SELECT SingerId, AlbumId, AlbumTitle FROM Albums'
```

For the Spanner SQL reference, see [Query syntax for GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/query-syntax) or [Query syntax for PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/query-syntax) .

To see a list of flags you can use with the `execute-sql` command, see [gcloud spanner databases execute-sql](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/databases/execute-sql) .

## Cleanup

To avoid incurring additional charges to your Google Cloud account, delete the database and instance. Disabling the Cloud Billing API doesn't stop charges. When you delete an instance, all databases in the instance are deleted.

### Drop a database

To delete an existing instance:

```
gcloud spanner databases delete example-db
```

### Delete an instance

To delete an existing instance:

```
gcloud spanner instances delete test-instance
```

Note that deleting an instance also drops all of the databases in that instance. Deleting an instance is not reversible.
