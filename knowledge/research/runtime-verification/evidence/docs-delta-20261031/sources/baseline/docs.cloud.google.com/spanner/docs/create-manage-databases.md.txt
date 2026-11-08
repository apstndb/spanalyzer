---
name: documents/docs.cloud.google.com/spanner/docs/create-manage-databases
uri: https://docs.cloud.google.com/spanner/docs/create-manage-databases
title: Create and manage databases
description: Manage Spanner databases using the {{dynamic_data.site_values.cloud_name_short}} console, Google Cloud CLI, client libraries, or REST APIs.
data_source: docs.cloud.google.com
---

This page describes how to create and manage Spanner [databases](https://docs.cloud.google.com/spanner/docs/databases) :

- Various methods to create a database
- Modify database options
- Delete a database

This page has information for both GoogleSQL-dialect databases and PostgreSQL-dialect databases. To learn how to update a database schema, see [Make schema updates](https://docs.cloud.google.com/spanner/docs/schema-updates) . For more information on creating an instance, see [Create and manage instances](https://docs.cloud.google.com/spanner/docs/create-manage-instances) . You can create a database in an existing instance in any of the following ways:

- **Create a database** : you can create a new database by selecting the SQL dialect and defining your schema.
- **Import your own data** : you can import a CSV, MySQL dump, or a PostgreSQL dump file into a new or existing database.
- **Import a Cloud SQL database** : you can import a Cloud SQL for MySQL database into Spanner.
- **Create a database with sample data** : you can populate a database using one of the available sample datasets to try out Spanner's capabilities.

## Create a database

You can create a new database in an existing instance. For GoogleSQL-dialect databases, you can define the database [schema](https://docs.cloud.google.com/spanner/docs/schema-and-data-model) either at the time of database creation, or after the database has been created. For PostgreSQL-dialect databases you must define the schema after creation.

Schemas are defined using the Database Definition Language, which is documented for [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language) and [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language) . Refer to the following links for more information about creating and updating schemas:

- [Schema and data model](https://docs.cloud.google.com/spanner/docs/schema-and-data-model)
- [Schema design best practices](https://docs.cloud.google.com/spanner/docs/schema-design)
- [Schema updates](https://docs.cloud.google.com/spanner/docs/schema-updates)

After you create your database, you can safeguard databases that are important to your applications and services by enabling database deletion protection. For more information, see [Prevent accidental database deletion](https://docs.cloud.google.com/spanner/docs/prevent-database-deletion) .

### Google Cloud console

1.  In the Google Cloud console, go to the **Spanner instances** page.

2.  Select the instance to create the database in.

3.  Click **Create database** .

4.  Enter the following values:

    - A **database name** to display in the Google Cloud console.
    - The **dialect** to use for this database.
    - For GoogleSQL-dialect databases, optionally provide a set of DDL statements that define your **schema** . Use the **DDL templates** to pre-fill common elements. If there are errors in your DDL statements, the Google Cloud console returns an error when you try to create the database.
    - Optionally, select a [customer-managed encryption key](https://docs.cloud.google.com/spanner/docs/cmek) to use for this database.

5.  Click **Create** to create the database.

### gcloud

Use the `gcloud spanner databases create` command.

````
```sh
gcloud spanner databases create DATABASE \
  --instance=INSTANCE \
  [--async] \
  [--database-dialect=DATABASE_DIALECT] \
  [--ddl=DDL] \
  [--ddl-file=DDL_FILE] \
  [--kms-key=KMS_KEY : --kms-keyring=KMS_KEYRING --kms-location=KMS_LOCATION --kms-project=KMS_PROJECT] \
  [GCLOUD_WIDE_FLAG …]
```
````

The following options are required:

`DATABASE`  
ID of the database or fully qualified identifier for the database. If specifying the fully qualified identifier, the `--instance` flag can be omitted.

`--instance= `` INSTANCE`  
The Spanner instance for the database.

The following options are optional:

`--async`  
Return immediately, without waiting for the operation in progress to complete.

`--database-dialect= `` DATABASE_DIALECT`  
The SQL dialect of the Spanner Database. Must be one of: `POSTGRESQL` , `GOOGLE_STANDARD_SQL` .

`--ddl= `` DDL`  
Semi-colon separated DDL (data definition language) statements to run inside the newly created database. If there is an error in any statement, the database is not created. This flag is ignored if `--ddl_file` is set. Not supported by PostgreSQL-dialect databases.

`--ddl-file= `` DDL_FILE`  
Path of a file that contains semicolon separated DDL (data definition language) statements to run inside the newly created database. If there is an error in any statement, the database is not created. If `--ddl_file` is set, `--ddl` is ignored. Not supported by PostgreSQL-dialect databases.

If you're specifying a [Cloud Key Management Service](https://cloud.google.com/security/products/security-key-management) key to use when creating the database, include the following options:

`--kms-key= `` KMS_KEY`  
ID of the key or fully qualified identifier for the key.

This flag must be specified if any of the other arguments in this group are specified. The other arguments could be omitted if the fully qualified identifier is provided.

`--kms-keyring= `` KMS_KEYRING`  
Cloud KMS key ring ID of the key.

`--kms-location= `` KMS_LOCATION`  
Google Cloud location for the key.

`--kms-project= `` KMS_PROJECT`  
Google Cloud project ID for the key.

### Client (GoogleSQL)

### C++

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```cpp
void CreateDatabase(google::cloud::spanner_admin::DatabaseAdminClient client,
                    std::string const& project_id,
                    std::string const& instance_id,
                    std::string const& database_id) {
  google::cloud::spanner::Database database(project_id, instance_id,
                                            database_id);
  google::spanner::admin::database::v1::CreateDatabaseRequest request;
  request.set_parent(database.instance().FullName());
  request.set_create_statement("CREATE DATABASE `" + database.database_id() +
                               "`");
  request.add_extra_statements(R"""(
      CREATE TABLE Singers (
          SingerId   INT64 NOT NULL,
          FirstName  STRING(1024),
          LastName   STRING(1024),
          SingerInfo BYTES(MAX),
          FullName   STRING(2049)
              AS (ARRAY_TO_STRING([FirstName, LastName], " ")) STORED
      ) PRIMARY KEY (SingerId))""");
  request.add_extra_statements(R"""(
      CREATE TABLE Albums (
          SingerId     INT64 NOT NULL,
          AlbumId      INT64 NOT NULL,
          AlbumTitle   STRING(MAX)
      ) PRIMARY KEY (SingerId, AlbumId),
          INTERLEAVE IN PARENT Singers ON DELETE CASCADE)""");
  auto db = client.CreateDatabase(request).get();
  if (!db) throw std::move(db).status();
  std::cout << "Database " << db->name() << " created.\n";
}
```

### Go

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```go
import (
    "context"
    "fmt"
    "io"
    "regexp"

    database "cloud.google.com/go/spanner/admin/database/apiv1"
    adminpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
)

func createDatabase(ctx context.Context, w io.Writer, db string) error {
    matches := regexp.MustCompile("^(.*)/databases/(.*)$").FindStringSubmatch(db)
    if matches == nil || len(matches) != 3 {
        return fmt.Errorf("Invalid database id %s", db)
    }

    adminClient, err := database.NewDatabaseAdminClient(ctx)
    if err != nil {
        return err
    }
    defer adminClient.Close()

    op, err := adminClient.CreateDatabase(ctx, &adminpb.CreateDatabaseRequest{
        Parent:          matches[1],
        CreateStatement: "CREATE DATABASE `" + matches[2] + "`",
        ExtraStatements: []string{
            `CREATE TABLE Singers (
                SingerId   INT64 NOT NULL,
                FirstName  STRING(1024),
                LastName   STRING(1024),
                SingerInfo BYTES(MAX),
                FullName   STRING(2048) AS (
                    ARRAY_TO_STRING([FirstName, LastName], " ")
                ) STORED
            ) PRIMARY KEY (SingerId)`,
            `CREATE TABLE Albums (
                SingerId     INT64 NOT NULL,
                AlbumId      INT64 NOT NULL,
                AlbumTitle   STRING(MAX)
            ) PRIMARY KEY (SingerId, AlbumId),
            INTERLEAVE IN PARENT Singers ON DELETE CASCADE`,
        },
    })
    if err != nil {
        return err
    }
    if _, err := op.Wait(ctx); err != nil {
        return err
    }
    fmt.Fprintf(w, "Created database [%s]\n", db)
    return nil
}
```

### Java

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```java
import com.google.cloud.spanner.SpannerException;
import com.google.cloud.spanner.SpannerExceptionFactory;
import com.google.cloud.spanner.admin.database.v1.DatabaseAdminClient;
import com.google.common.collect.ImmutableList;
import com.google.spanner.admin.database.v1.CreateDatabaseRequest;
import com.google.spanner.admin.database.v1.Database;
import java.io.IOException;
import java.util.concurrent.ExecutionException;

public class CreateDatabaseWithDefaultLeaderSample {

  static void createDatabaseWithDefaultLeader() throws IOException {
    // TODO(developer): Replace these variables before running the sample.
    final String instanceName = "projects/my-project/instances/my-instance-id";
    final String databaseId = "my-database-name";
    final String defaultLeader = "my-default-leader";
    createDatabaseWithDefaultLeader(instanceName, databaseId, defaultLeader);
  }

  static void createDatabaseWithDefaultLeader(String instanceName, String databaseId,
      String defaultLeader) throws IOException {
    try (DatabaseAdminClient databaseAdminClient = DatabaseAdminClient.create()) {
      Database createdDatabase =
          databaseAdminClient.createDatabaseAsync(
              CreateDatabaseRequest.newBuilder()
                  .setParent(instanceName)
                  .setCreateStatement("CREATE DATABASE `" + databaseId + "`")
                  .addAllExtraStatements(
                      ImmutableList.of("CREATE TABLE Singers ("
                              + "  SingerId   INT64 NOT NULL,"
                              + "  FirstName  STRING(1024),"
                              + "  LastName   STRING(1024),"
                              + "  SingerInfo BYTES(MAX)"
                              + ") PRIMARY KEY (SingerId)",
                          "CREATE TABLE Albums ("
                              + "  SingerId     INT64 NOT NULL,"
                              + "  AlbumId      INT64 NOT NULL,"
                              + "  AlbumTitle   STRING(MAX)"
                              + ") PRIMARY KEY (SingerId, AlbumId),"
                              + "  INTERLEAVE IN PARENT Singers ON DELETE CASCADE",
                          "ALTER DATABASE " + "`" + databaseId + "`"
                              + " SET OPTIONS ( default_leader = '" + defaultLeader + "' )"))
                  .build()).get();
      System.out.println("Created database [" + createdDatabase.getName() + "]");
      System.out.println("\tDefault leader: " + createdDatabase.getDefaultLeader());
    } catch (ExecutionException e) {
      // If the operation failed during execution, expose the cause.
      throw (SpannerException) e.getCause();
    } catch (InterruptedException e) {
      // Throw when a thread is waiting, sleeping, or otherwise occupied,
      // and the thread is interrupted, either before or during the activity.
      throw SpannerExceptionFactory.propagateInterrupt(e);
    }
  }
}
```

> **Note:** The old client library interface code samples for Java are archived in [GitHub](https://github.com/googleapis/google-cloud-java/tree/main/java-spanner/samples/snippets/src/main/java/com/example/spanner/admin/archived) .

### Node.js

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```javascript
/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance-id';
// const databaseId = 'my-database-id';
// const defaultLeader = 'my-default-leader'; example: 'asia-northeast1'

// Imports the Google Cloud client library
const {Spanner} = require('@google-cloud/spanner');

// creates a client
const spanner = new Spanner({
  projectId: projectId,
});

// Gets a reference to a Cloud Spanner Database Admin Client object
const databaseAdminClient = spanner.getDatabaseAdminClient();

async function createDatabaseWithDefaultLeader() {
  // Create a new database with an extra statement which will alter the
  // database after creation to set the default leader.
  console.log(
    `Creating database ${databaseAdminClient.databasePath(
      projectId,
      instanceId,
      databaseId,
    )}.`,
  );
  const createSingersTableStatement = `
    CREATE TABLE Singers (
      SingerId   INT64 NOT NULL,
      FirstName  STRING(1024),
      LastName   STRING(1024),
      SingerInfo BYTES(MAX)
    ) PRIMARY KEY (SingerId)`;
  const createAlbumsStatement = `
    CREATE TABLE Albums (
      SingerId     INT64 NOT NULL,
      AlbumId      INT64 NOT NULL,
      AlbumTitle   STRING(MAX)
    ) PRIMARY KEY (SingerId, AlbumId),
      INTERLEAVE IN PARENT Singers ON DELETE CASCADE`;

  // Default leader is one of the possible values in the leaderOptions field of the
  // instance config of the instance where the database is created.
  const setDefaultLeaderStatement = `
    ALTER DATABASE \`${databaseId}\`
    SET OPTIONS (default_leader = '${defaultLeader}')`;

  const [operation] = await databaseAdminClient.createDatabase({
    createStatement: 'CREATE DATABASE `' + databaseId + '`',
    extraStatements: [
      createSingersTableStatement,
      createAlbumsStatement,
      setDefaultLeaderStatement,
    ],
    parent: databaseAdminClient.instancePath(projectId, instanceId),
  });

  console.log(`Waiting for creation of ${databaseId} to complete...`);
  await operation.promise();
  console.log(
    `Created database ${databaseId} with default leader ${defaultLeader}.`,
  );
}
createDatabaseWithDefaultLeader();
```

> **Note:** The old client library interface code samples for Node.js are archived in [GitHub](https://github.com/googleapis/nodejs-spanner/tree/main/samples/archived) .

### PHP

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```php
use Google\Cloud\Spanner\Admin\Database\V1\Client\DatabaseAdminClient;
use Google\Cloud\Spanner\Admin\Database\V1\CreateDatabaseRequest;

/**
 * Creates a database and tables for sample data.
 * Example:
 * ```
 * create_database($instanceId, $databaseId);
 * ```
 *
 * @param string $projectId The Google Cloud project ID.
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 */
function create_database(string $projectId, string $instanceId, string $databaseId): void
{
    $databaseAdminClient = new DatabaseAdminClient();
    $instance = $databaseAdminClient->instanceName($projectId, $instanceId);

    $operation = $databaseAdminClient->createDatabase(
        new CreateDatabaseRequest([
            'parent' => $instance,
            'create_statement' => sprintf('CREATE DATABASE `%s`', $databaseId),
            'extra_statements' => [
                'CREATE TABLE Singers (' .
                'SingerId     INT64 NOT NULL,' .
                'FirstName    STRING(1024),' .
                'LastName     STRING(1024),' .
                'SingerInfo   BYTES(MAX),' .
                'FullName     STRING(2048) AS' .
                '(ARRAY_TO_STRING([FirstName, LastName], " ")) STORED' .
                ') PRIMARY KEY (SingerId)',
                'CREATE TABLE Albums (' .
                    'SingerId     INT64 NOT NULL,' .
                    'AlbumId      INT64 NOT NULL,' .
                    'AlbumTitle   STRING(MAX)' .
                ') PRIMARY KEY (SingerId, AlbumId),' .
                'INTERLEAVE IN PARENT Singers ON DELETE CASCADE'
            ]
        ])
    );

    print('Waiting for operation to complete...' . PHP_EOL);
    $operation->pollUntilComplete();

    printf('Created database %s on instance %s' . PHP_EOL,
        $databaseId, $instanceId);
}
```

> **Note:** The old client library interface code samples for PHP are archived in [GitHub](https://github.com/GoogleCloudPlatform/php-docs-samples/tree/main/spanner/src/admin/archived) .

### Python

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```python
def create_database(instance_id, database_id):
    """Creates a database and tables for sample data."""
    from google.cloud.spanner_admin_database_v1.types import spanner_database_admin

    spanner_client = spanner.Client()
    database_admin_api = spanner_client.database_admin_api

    request = spanner_database_admin.CreateDatabaseRequest(
        parent=database_admin_api.instance_path(spanner_client.project, instance_id),
        create_statement=f"CREATE DATABASE `{database_id}`",
        extra_statements=[
            """CREATE TABLE Singers (
            SingerId     INT64 NOT NULL,
            FirstName    STRING(1024),
            LastName     STRING(1024),
            SingerInfo   BYTES(MAX),
            FullName   STRING(2048) AS (
                ARRAY_TO_STRING([FirstName, LastName], " ")
            ) STORED
        ) PRIMARY KEY (SingerId)""",
            """CREATE TABLE Albums (
            SingerId     INT64 NOT NULL,
            AlbumId      INT64 NOT NULL,
            AlbumTitle   STRING(MAX)
        ) PRIMARY KEY (SingerId, AlbumId),
        INTERLEAVE IN PARENT Singers ON DELETE CASCADE""",
        ],
    )

    operation = database_admin_api.create_database(request=request)

    print("Waiting for operation to complete...")
    database = operation.result(OPERATION_TIMEOUT_SECONDS)

    print(
        "Created database {} on instance {}".format(
            database.name,
            database_admin_api.instance_path(spanner_client.project, instance_id),
        )
    )
```

> **Note:** The old client library interface code samples for Python are archived in [GitHub](https://github.com/googleapis/python-spanner/tree/main/samples/samples/archived) .

### Ruby

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```ruby
# project_id  = "Your Google Cloud project ID"
# instance_id = "Your Spanner instance ID"
# database_id = "Your Spanner database ID"

require "google/cloud/spanner"
require "google/cloud/spanner/admin/database"

database_admin_client = Google::Cloud::Spanner::Admin::Database.database_admin

instance_path = database_admin_client.instance_path project: project_id, instance: instance_id

job = database_admin_client.create_database parent: instance_path,
                                            create_statement: "CREATE DATABASE `#{database_id}`",
                                            extra_statements: [
                                              "CREATE TABLE Singers (
      SingerId     INT64 NOT NULL,
      FirstName    STRING(1024),
      LastName     STRING(1024),
      SingerInfo   BYTES(MAX)
    ) PRIMARY KEY (SingerId)",

                                              "CREATE TABLE Albums (
      SingerId     INT64 NOT NULL,
      AlbumId      INT64 NOT NULL,
      AlbumTitle   STRING(MAX)
    ) PRIMARY KEY (SingerId, AlbumId),
    INTERLEAVE IN PARENT Singers ON DELETE CASCADE"
                                            ]

puts "Waiting for create database operation to complete"

job.wait_until_done!

puts "Created database #{database_id} on instance #{instance_id}"
```

### Client (PostgreSQL)

### C++

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```cpp
void CreateDatabase(google::cloud::spanner_admin::DatabaseAdminClient client,
                    google::cloud::spanner::Database const& database) {
  google::spanner::admin::database::v1::CreateDatabaseRequest request;
  request.set_parent(database.instance().FullName());
  request.set_create_statement("CREATE DATABASE \"" + database.database_id() +
                               "\"");
  request.set_database_dialect(
      google::spanner::admin::database::v1::DatabaseDialect::POSTGRESQL);
  auto db = client.CreateDatabase(request).get();
  if (!db) throw std::move(db).status();
  std::cout << "Database " << db->name() << " created.\n";
}
```

### Go

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```go
import (
    "context"
    "fmt"
    "io"
    "regexp"

    database "cloud.google.com/go/spanner/admin/database/apiv1"
    adminpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
)

// pgCreateDatabase shows how to create a Spanner database that uses the
// PostgreSQL dialect.
func pgCreateDatabase(ctx context.Context, w io.Writer, db string) error {
    // db := "projects/my-project/instances/my-instance/databases/my-database"
    matches := regexp.MustCompile("^(.*)/databases/(.*)$").FindStringSubmatch(db)
    if matches == nil || len(matches) != 3 {
        return fmt.Errorf("invalid database id %v", db)
    }

    adminClient, err := database.NewDatabaseAdminClient(ctx)
    if err != nil {
        return err
    }
    defer adminClient.Close()

    // Databases with PostgreSQL dialect do not support extra DDL statements in the `CreateDatabase` call.
    req := &adminpb.CreateDatabaseRequest{
        Parent:          matches[1],
        DatabaseDialect: adminpb.DatabaseDialect_POSTGRESQL,
        // Note that PostgreSQL uses double quotes for quoting identifiers. This also
        // includes database names in the CREATE DATABASE statement.
        CreateStatement: `CREATE DATABASE "` + matches[2] + `"`,
    }
    opCreate, err := adminClient.CreateDatabase(ctx, req)
    if err != nil {
        return err
    }
    if _, err := opCreate.Wait(ctx); err != nil {
        return err
    }
    updateReq := &adminpb.UpdateDatabaseDdlRequest{
        Database: db,
        Statements: []string{
            `CREATE TABLE Singers (
                SingerId   bigint NOT NULL PRIMARY KEY,
                FirstName  varchar(1024),
                LastName   varchar(1024),
                SingerInfo bytea
            )`,
            `CREATE TABLE Albums (
                AlbumId      bigint NOT NULL,
                SingerId     bigint NOT NULL REFERENCES Singers (SingerId),
                AlbumTitle   text,
                PRIMARY KEY(SingerId, AlbumId)
            )`,
            `CREATE TABLE Venues (
                VenueId  bigint NOT NULL PRIMARY KEY,
                Name     varchar(1024) NOT NULL
            )`,
        },
    }
    opUpdate, err := adminClient.UpdateDatabaseDdl(ctx, updateReq)
    if err != nil {
        return err
    }
    if err := opUpdate.Wait(ctx); err != nil {
        return err
    }
    fmt.Fprintf(w, "Created Spanner PostgreSQL database [%v]\n", db)
    return nil
}
```

### Java

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

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

### Node.js

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```javascript
/**
 * TODO(developer): Uncomment these variables before running the sample.
 */
// const instanceId = 'my-instance';
// const databaseId = 'my-database';
// const projectId = 'my-project-id';

// Imports the Google Cloud client library
const {Spanner, protos} = require('@google-cloud/spanner');

// creates a client
const spanner = new Spanner({
  projectId: projectId,
});

const databaseAdminClient = spanner.getDatabaseAdminClient();

async function createPgDatabase() {
  // Creates a PostgreSQL database. PostgreSQL create requests may not contain any additional
  // DDL statements. We need to execute these separately after the database has been created.
  const [operationCreate] = await databaseAdminClient.createDatabase({
    createStatement: 'CREATE DATABASE "' + databaseId + '"',
    parent: databaseAdminClient.instancePath(projectId, instanceId),
    databaseDialect:
      protos.google.spanner.admin.database.v1.DatabaseDialect.POSTGRESQL,
  });

  console.log(`Waiting for operation on ${databaseId} to complete...`);
  await operationCreate.promise();
  const [metadata] = await databaseAdminClient.getDatabase({
    name: databaseAdminClient.databasePath(projectId, instanceId, databaseId),
  });
  console.log(
    `Created database ${databaseId} on instance ${instanceId} with dialect ${metadata.databaseDialect}.`,
  );

  // Create a couple of tables using a separate request. We must use PostgreSQL style DDL as the
  // database has been created with the PostgreSQL dialect.
  const statements = [
    `CREATE TABLE Singers 
      (SingerId   bigint NOT NULL,
      FirstName   varchar(1024),
      LastName    varchar(1024),
      SingerInfo  bytea,
      FullName    character varying(2048) GENERATED ALWAYS AS (FirstName || ' ' || LastName) STORED,
      PRIMARY KEY (SingerId)
      );
      CREATE TABLE Albums 
      (AlbumId    bigint NOT NULL,
      SingerId    bigint NOT NULL REFERENCES Singers (SingerId),
      AlbumTitle  text,
      PRIMARY KEY (AlbumId)
      );`,
  ];
  const [operationUpdateDDL] = await databaseAdminClient.updateDatabaseDdl({
    database: databaseAdminClient.databasePath(
      projectId,
      instanceId,
      databaseId,
    ),
    statements: [statements],
  });
  await operationUpdateDDL.promise();
  console.log('Updated schema');
}
createPgDatabase();
```

### PHP

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```php
use Google\Cloud\Spanner\Admin\Database\V1\Client\DatabaseAdminClient;
use Google\Cloud\Spanner\Admin\Database\V1\CreateDatabaseRequest;
use Google\Cloud\Spanner\Admin\Database\V1\DatabaseDialect;
use Google\Cloud\Spanner\Admin\Database\V1\GetDatabaseRequest;
use Google\Cloud\Spanner\Admin\Database\V1\UpdateDatabaseDdlRequest;

/**
 * Creates a database that uses Postgres dialect
 *
 * @param string $projectId The Google Cloud project ID.
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 */
function pg_create_database(string $projectId, string $instanceId, string $databaseId): void
{
    $databaseAdminClient = new DatabaseAdminClient();
    $instance = $databaseAdminClient->instanceName($projectId, $instanceId);
    $databaseName = $databaseAdminClient->databaseName($projectId, $instanceId, $databaseId);

    $table1Query = 'CREATE TABLE Singers (
        SingerId   bigint NOT NULL PRIMARY KEY,
        FirstName  varchar(1024),
        LastName   varchar(1024),
        SingerInfo bytea,
        FullName character varying(2048) GENERATED
        ALWAYS AS (FirstName || \' \' || LastName) STORED
    )';
    $table2Query = 'CREATE TABLE Albums (
        AlbumId      bigint NOT NULL,
        SingerId     bigint NOT NULL REFERENCES Singers (SingerId),
        AlbumTitle   text,
        PRIMARY KEY(SingerId, AlbumId)
    )';

    $operation = $databaseAdminClient->createDatabase(
        new CreateDatabaseRequest([
            'parent' => $instance,
            'create_statement' => sprintf('CREATE DATABASE "%s"', $databaseId),
            'extra_statements' => [],
            'database_dialect' => DatabaseDialect::POSTGRESQL
        ])
    );

    print('Waiting for operation to complete...' . PHP_EOL);
    $operation->pollUntilComplete();

    $request = new UpdateDatabaseDdlRequest([
        'database' => $databaseName,
        'statements' => [$table1Query, $table2Query]
    ]);

    $operation = $databaseAdminClient->updateDatabaseDdl($request);
    $operation->pollUntilComplete();

    $database = $databaseAdminClient->getDatabase(
        new GetDatabaseRequest(['name' => $databaseAdminClient->databaseName($projectId, $instanceId, $databaseId)])
    );
    $dialect = DatabaseDialect::name($database->getDatabaseDialect());

    printf('Created database %s with dialect %s on instance %s' . PHP_EOL,
        $databaseId, $dialect, $instanceId);
}
```

### Python

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```python
def create_database(instance_id, database_id):
    """Creates a PostgreSql database and tables for sample data."""

    from google.cloud.spanner_admin_database_v1.types import spanner_database_admin

    spanner_client = spanner.Client()
    database_admin_api = spanner_client.database_admin_api

    request = spanner_database_admin.CreateDatabaseRequest(
        parent=database_admin_api.instance_path(spanner_client.project, instance_id),
        create_statement=f'CREATE DATABASE "{database_id}"',
        database_dialect=DatabaseDialect.POSTGRESQL,
    )

    operation = database_admin_api.create_database(request=request)

    print("Waiting for operation to complete...")
    database = operation.result(OPERATION_TIMEOUT_SECONDS)

    create_table_using_ddl(database.name)
    print("Created database {} on instance {}".format(database_id, instance_id))


def create_table_using_ddl(database_name):
    from google.cloud.spanner_admin_database_v1.types import spanner_database_admin

    spanner_client = spanner.Client()
    request = spanner_database_admin.UpdateDatabaseDdlRequest(
        database=database_name,
        statements=[
            """CREATE TABLE Singers (
  SingerId   bigint NOT NULL,
  FirstName  character varying(1024),
  LastName   character varying(1024),
  SingerInfo bytea,
  FullName   character varying(2048)
    GENERATED ALWAYS AS (FirstName || ' ' || LastName) STORED,
  PRIMARY KEY (SingerId)
  )""",
            """CREATE TABLE Albums (
  SingerId     bigint NOT NULL,
  AlbumId      bigint NOT NULL,
  AlbumTitle   character varying(1024),
  PRIMARY KEY (SingerId, AlbumId)
  ) INTERLEAVE IN PARENT Singers ON DELETE CASCADE""",
        ],
    )
    operation = spanner_client.database_admin_api.update_database_ddl(request)
    operation.result(OPERATION_TIMEOUT_SECONDS)
```

### Ruby

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```ruby
require "google/cloud/spanner"
require "google/cloud/spanner/admin/database"

def postgresql_create_database project_id:, instance_id:, database_id:
  # project_id  = "Your Google Cloud project ID"
  # instance_id = "Your Spanner instance ID"
  # database_id = "Your Spanner database ID"

  database_admin_client = Google::Cloud::Spanner::Admin::Database.database_admin project: project_id

  instance_path = database_admin_client.instance_path project: project_id, instance: instance_id

  job = database_admin_client.create_database parent: instance_path,
                                              create_statement: "CREATE DATABASE \"#{database_id}\"",
                                              database_dialect: :POSTGRESQL

  puts "Waiting for create database operation to complete"

  job.wait_until_done!

  puts "Created database #{database_id} on instance #{instance_id}"
end
```

## Import your own data

You can import your own data into a Spanner database by using a CSV file, a MySQL dump file, or a PostgreSQL dump file. You can upload a local file using Cloud Storage or from a Cloud Storage bucket directly. Uploading a local file using Cloud Storage might incur charges.

If you choose to use a CSV file, you also need to upload a separate JSON file that contains the database schema.

### Google Cloud console

1.  In the Google Cloud console, go to the **Spanner Instances** page.

2.  Select the instance to create the database in.

3.  Click **Import my own data** .

4.  Enter the following values:

    - Select the **File type** .

    - Upload the file from your computer or select a Cloud Storage bucket path to the file.

    - (Optional) If you choose to use a CSV file, you also need to upload a separate JSON file that contains the database schema. The JSON file must use the following structure to define the schema:

      ```
      {
        "name": "COLUMN_NAME",
        "type": "TYPE",
        "notNull": NOT_NULL_VALUE,
        "primaryKeyOrder": PRIMARY_KEY_ORDER
      }
      ```

      Replace the following:

      - ` COLUMN_NAME ` : the name of the column in the table.

      - ` TYPE ` : the data type of the column.

      - (Optional) ` NOT_NULL_VALUE ` : whether the column can store null values or not. Valid inputs are `true` or `false` . Defaults to `false` .

      - (Optional): ` PRIMARY_KEY_ORDER ` : determines the primary key order. Set the value is set to `0` for a non-primary key column. Set the value to an integer, for example, `1` for a primary key column. Lower numbered columns appear earlier in a compound primary key.

      The CSV file expects a comma for the field delimiter and a new line for the line delimiter by default. For more information on using custom delimiters, see the [`gcloud alpha spanner databases import`](https://docs.cloud.google.com/sdk/gcloud/reference/alpha/spanner/databases/import) reference.

    - Select a new or existing database as the destination.

5.  Click **Import** .

6.  Spanner opens the Cloud Shell and populates a command that installs the [Spanner migration tool](https://googlecloudplatform.github.io/spanner-migration-tool/) and runs the [`gcloud alpha spanner databases import`](https://docs.cloud.google.com/sdk/gcloud/reference/alpha/spanner/databases/import) command. Press the `ENTER` key to import data into your database.x

## Import a database from Cloud SQL

If you use Cloud SQL for MySQL and want to evaluate Spanner, you can import data from your Cloud SQL for MySQL instance into Spanner. This feature converts the MySQL schema and generates a corresponding Spanner schema. Then, it loads data from the Cloud SQL for MySQL database into Spanner.

This functionality performs only a one-time bulk data load. It doesn't support capturing live data changes or fallback from Spanner to your source database.

For more information, see [Import from Cloud SQL to Spanner](https://docs.cloud.google.com/spanner/docs/import-database-cloudsql) .

## Use a sample dataset

To explore [Spanner Graph](https://docs.cloud.google.com/spanner/docs/graph/overview) , vector search, or the relational model, you can create a new database pre-populated with a sample dataset. Available sample datasets depend on the selected database dialect (GoogleSQL or PostgreSQL).

1.  In the Google Cloud console, go to the **Spanner instances** page.

2.  Select the instance to create the database in.

3.  Click **Explore datasets** .

4.  Select a dataset, depending on your database dialect:

    ### GoogleSQL

    - **Finance graph** : use this dataset to explore Spanner's [graph](https://docs.cloud.google.com/spanner/docs/graph/overview) and relational database features.
    - **Online banking** : use this dataset to explore Spanner's [full-text search](https://docs.cloud.google.com/spanner/docs/full-text-search) and [vector search](https://docs.cloud.google.com/spanner/docs/find-approximate-nearest-neighbors) features.
    - **Online gaming** : use this dataset to explore Spanner's relational database features.
    - **Retail** : use this dataset to explore Spanner's [graph](https://docs.cloud.google.com/spanner/docs/graph/overview) , [full-text search](https://docs.cloud.google.com/spanner/docs/full-text-search) , and [vector search](https://docs.cloud.google.com/spanner/docs/find-approximate-nearest-neighbors) features.

    ### PostgreSQL

    - **Online banking** : use this dataset to explore Spanner's [full-text search](https://docs.cloud.google.com/spanner/docs/full-text-search) features.
    - **Online gaming** : use this dataset to explore Spanner's relational database features.
    - **Retail** : use this dataset to explore Spanner's [full-text search](https://docs.cloud.google.com/spanner/docs/full-text-search) and [vector search](https://docs.cloud.google.com/spanner/docs/find-approximate-nearest-neighbors) features.

5.  Click **Create database** .

## Update database schema or options

You can update your database schema and options using DDL statements.

For example, to add a column to a table, use the following DDL statement:

### GoogleSQL

```
ALTER TABLE Songwriters ADD COLUMN Publisher STRING(10);
```

### PostgreSQL

```
ALTER TABLE Songwriters ADD COLUMN Publisher VARCHAR(10);
```

To update the query optimizer version, use the following DDL statement:

### GoogleSQL

```
ALTER DATABASE Music SET OPTIONS(optimizer_version=null);
```

### PostgreSQL

```
ALTER DATABASE DB-NAME SET spanner.optimizer_version TO DEFAULT;
```

For more information about supported options, refer to the `ALTER DATABASE` DDL reference for [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter-database) or [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#alter-database) .

For information about schema updates, see [Make schema updates](https://docs.cloud.google.com/spanner/docs/schema-updates) .

### Google Cloud console

1.  In the Google Cloud console, go to the **Spanner instances** page.

2.  Select the instance containing the database to alter.

3.  Select the database.

4.  Click **Spanner Studio** .

5.  Click add **New tab** or use the empty editor tab. Then, enter the DDL statements to apply.

6.  Click **Run** to apply the updates. If there are errors in your DDL, the Google Cloud console returns an error and the database is not altered.

### gcloud

To alter a database with the `gcloud` command-line tool, use `gcloud spanner databases ddl update` .

```
gcloud spanner databases ddl update \
(DATABASE : --instance=INSTANCE) \
[--async] \
[--ddl=DDL] \
[--ddl-file=DDL_FILE] \
```

Refer to the [`gcloud` reference](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/databases/ddl/update) for details about the available options.

Pass the database updates to the command with either the `--ddl` flag, or the `--ddl-file` flag. If a DDL file is specified, the `--ddl` flag is ignored.

Refer to the `ALTER DATABASE` DDL reference for [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter-database) or [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#alter-database) for the DDL statements to include.

### DDL

Refer to the `ALTER DATABASE` DDL reference for [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter-database) or [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#alter-database) for details.

## Check the progress of schema update operations

To check the progress of your schema update operations, select one of the following methods:

### Google Cloud console

1.  In the Spanner navigation menu, select the **Operations** tab. The **Operations** page shows a list of active running operations.

2.  Find the schema operation in the list. If it's still running, the progress bar in the **End time** column shows the percentage of the operation that is complete, as shown in the following image:

![Progress bar shows 98%](https://docs.cloud.google.com/static/spanner/docs/images/ddl-operation-progress.png)

### gcloud

Use [`gcloud spanner operations describe`](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/operations/describe) to check the progress of an operation.

1.  Get the operation ID:

    ```
    gcloud spanner operations list --instance=INSTANCE-NAME \
    --database=DATABASE-NAME --type=DATABASE_UPDATE_DDL
    ```

    Replace the following:

    - ` INSTANCE-NAME ` with the Spanner instance name.
    - ` DATABASE-NAME ` with the name of the database.

2.  Run `gcloud spanner operations describe` :

    ```
    gcloud spanner operations describe OPERATION_ID\
    --instance=INSTANCE-NAME \
    --database=DATABASE-NAME
    ```

    Replace the following:

    - ` OPERATION-ID ` : The operation ID of the operation that you want to check.
    - ` INSTANCE-NAME ` : The Spanner instance name.
    - ` DATABASE-NAME ` : The Spanner database name.

    The `progress` section in the output shows the percentage of the operation that's complete. The output looks similar to the following:

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

### REST v1

1.  Get the operation ID:

    ```
    gcloud spanner operations list --instance=INSTANCE-NAME \
    --database=DATABASE-NAME --type=DATABASE_UPDATE_DDL
    ```

    Replace the following:

    - ` INSTANCE-NAME ` with the Spanner instance name.
    - ` DATABASE-NAME ` with the database name.

2.  Check the progress for the operation.

    Before using any of the request data, make the following replacements:

    - ` PROJECT-ID ` : the project ID.
    - ` INSTANCE-ID ` : the instance ID.
    - ` DATABASE-ID ` : the database ID.
    - ` OPERATION-ID ` : the operation ID.

    HTTP method and URL:

    ```
    GET https://spanner.googleapis.com/v1/projects/PROJECT-ID/instances/INSTANCE-ID/databases/DATABASE-ID/operations/OPERATION-ID
    ```

    To send your request, expand one of these options:

    #### curl (Linux, macOS, or Cloud Shell)

    > **Note:** The following command assumes that you have logged in to the `gcloud` CLI with your user account by running [`gcloud init`](https://docs.cloud.google.com/sdk/gcloud/reference/init) or [`gcloud auth login`](https://docs.cloud.google.com/sdk/gcloud/reference/auth/login) , or by using [Cloud Shell](https://docs.cloud.google.com/shell/docs) , which automatically logs you into the `gcloud` CLI . You can check the currently active account by running [`gcloud auth list`](https://docs.cloud.google.com/sdk/gcloud/reference/auth/list) .

    Execute the following command:

    ```
    curl -X GET \
         -H "Authorization: Bearer $(gcloud auth print-access-token)" \
         "https://spanner.googleapis.com/v1/projects/PROJECT-ID/instances/INSTANCE-ID/databases/DATABASE-ID/operations/OPERATION-ID"
    ```

    #### PowerShell (Windows)

    > **Note:** The following command assumes that you have logged in to the `gcloud` CLI with your user account by running [`gcloud init`](https://docs.cloud.google.com/sdk/gcloud/reference/init) or [`gcloud auth login`](https://docs.cloud.google.com/sdk/gcloud/reference/auth/login) . You can check the currently active account by running [`gcloud auth list`](https://docs.cloud.google.com/sdk/gcloud/reference/auth/list) .

    Execute the following command:

    ```
    $cred = gcloud auth print-access-token
    $headers = @{ "Authorization" = "Bearer $cred" }

    Invoke-WebRequest `
        -Method GET `
        -Headers $headers `
        -Uri "https://spanner.googleapis.com/v1/projects/PROJECT-ID/instances/INSTANCE-ID/databases/DATABASE-ID/operations/OPERATION-ID" | Select-Object -Expand Content
    ```

    You should receive a JSON response similar to the following:

    ```
    {
    ...
        "progress": [
          {
            "progressPercent": 100,
            "startTime": "2023-05-27T00:52:27.366688Z",
            "endTime": "2023-05-27T00:52:30.184845Z"
          },
          {
            "progressPercent": 100,
            "startTime": "2023-05-27T00:52:30.184845Z",
            "endTime": "2023-05-27T00:52:40.750959Z"
          }
        ],
    ...
      "done": true,
      "response": {
        "@type": "type.googleapis.com/google.protobuf.Empty"
      }
    }
    ```

If the operation takes too long, you can cancel it. For more information, see [Cancel a long-running database operation](https://docs.cloud.google.com/spanner/docs/manage-and-observe-long-running-operations#cancel_a_long-running_database_operation) .

Spanner also automatically detects opportunities to apply [schema design best practices](https://docs.cloud.google.com/spanner/docs/schema-design) . If recommendations are available for a database, you can view them on the **Spanner Studio** page for that database. For more information, see [View schema design best practice recommendations](https://docs.cloud.google.com/spanner/docs/manage-data-using-console#recommend-schema-best) .

## Delete a database

Deleting a database permanently removes the database and all its data. Database deletion can't be undone. If [database deletion protection](https://docs.cloud.google.com/spanner/docs/prevent-database-deletion) is enabled on a database, you can't delete that database until you [disable its deletion protection](https://docs.cloud.google.com/spanner/docs/prevent-database-deletion#disable) .

Existing backups are **not** deleted when a database is deleted. For more information, see [Backup and restore](https://docs.cloud.google.com/spanner/docs/backup) .

### Google Cloud console

1.  In the Google Cloud console, go to the **Spanner instances** page.

2.  Select the instance containing the database to delete.

3.  Select the database.

4.  Click **Delete database** . A confirmation appears.

5.  Type the database name and click **Delete** .

### gcloud

To delete a database with the `gcloud` command-line tool, use `gcloud spanner databases delete` .

```
gcloud spanner databases delete \
  (DATABASE : --instance=INSTANCE)
```

The following options are required:

`DATABASE`  
ID of the database or fully qualified identifier for the database. If the fully qualified identifier is provided, the `--instance` flag should be omitted.

`--instance= `` INSTANCE`  
The Spanner instance for the database.

For more details refer to the [`gcloud` reference](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/databases/delete) .

### DDL

DDL does not support database deletion syntax.

## What's next

- [Create a database and load it with sample data](https://codelabs.developers.google.com/codelabs/cloud-spanner-first-db) .
- Learn more about [GoogleSQL DDL reference](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language) .
- Learn more about [PostgreSQL DDL reference](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language) .
- Learn how to [backup and restore a database](https://docs.cloud.google.com/spanner/docs/backup) .
- Learn how to [prevent accidental database deletion](https://docs.cloud.google.com/spanner/docs/prevent-database-deletion) .
- Learn how to [make schema updates](https://docs.cloud.google.com/spanner/docs/schema-updates) .
