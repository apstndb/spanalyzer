---
name: documents/docs.cloud.google.com/spanner/docs/use-pitr
uri: https://docs.cloud.google.com/spanner/docs/use-pitr
title: Recover data using point-in-time recovery (PITR)
description: Recover your Spanner data to a specific point in time using PITR for data protection and disaster recovery.
data_source: docs.cloud.google.com
---

This page describes how to use point-in-time recovery (PITR) to retain and recover data in Spanner for GoogleSQL-dialect databases and PostgreSQL-dialect databases.

To learn more, see [Point-in-time recovery](https://docs.cloud.google.com/spanner/docs/pitr) .

## Prerequisites

This guide uses the database and schema as defined in the [Spanner quickstart](https://docs.cloud.google.com/spanner/docs/create-query-database-console) . You can either run through the quickstart to create the database and schema, or modify the commands for use with your own database.

## Set the retention period

To set your database's retention period:

### Console

1.  Go to the Spanner Instances page in the Google Cloud console.

2.  Click the instance containing the database to open its **Overview** page.

3.  Click the database to open its **Overview** page.

4.  Select the **Backup/Restore** tab.

    ![Backup/Restore form in the console, showing the Version retention period field set to 1 hour and a pencil icon for editing this value.](https://docs.cloud.google.com/static/spanner/docs/images/pitr-retention-period-update.png)

5.  Click the pencil icon in the **Version retention period** field.

6.  Enter a quantity and unit of time for the retention period and then click **Update** .

### gcloud

Update the database's schema with the [`ALTER DATABASE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter-database) statement. For example:

```
gcloud spanner databases ddl update example-db \
    --instance=test-instance \
    --ddl="ALTER DATABASE \"example-db\" \
    SET spanner.version_retention_period = '7d';"
```

The following shows the output:

```
ALTER DATABASE "example-db" SET "spanner.version_retention_period" = '7d';
```

> **Note:** Double quotes (") are required if the database name includes special characters like a hyphen (-).

To view the retention period, get your database's DDL:

```
gcloud spanner databases ddl describe example-db \
    --instance=test-instance
```

The following shows the output:

```
ALTER DATABASE example-db SET OPTIONS (
  version_retention_period = '7d'
);
...
```

### Client libraries

### C#

```csharp
using Google.Cloud.Spanner.Data;
using System.Threading.Tasks;

public class CreateDatabaseWithRetentionPeriodAsyncSample
{
    public async Task CreateDatabaseWithRetentionPeriodAsync(string projectId, string instanceId, string databaseId)
    {
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}";

        using var connection = new SpannerConnection(connectionString);
        var versionRetentionPeriod = "7d";
        var createStatement = $"CREATE DATABASE `{databaseId}`";
        var alterStatement = @$"ALTER DATABASE `{databaseId}` SET OPTIONS
                   (version_retention_period = '{versionRetentionPeriod}')";
        // The retention period cannot be set as part of the CREATE DATABASE statement,
        // but can be set using an ALTER DATABASE statement directly after database creation.
        using var createDbCommand = connection.CreateDdlCommand(
            createStatement,
            alterStatement
        );
        await createDbCommand.ExecuteNonQueryAsync();
    }
}
```

### C++

```cpp
void CreateDatabaseWithVersionRetentionPeriod(
    google::cloud::spanner_admin::DatabaseAdminClient client,
    std::string const& project_id, std::string const& instance_id,
    std::string const& database_id) {
  google::cloud::spanner::Database database(project_id, instance_id,
                                            database_id);
  google::spanner::admin::database::v1::CreateDatabaseRequest request;
  request.set_parent(database.instance().FullName());
  request.set_create_statement("CREATE DATABASE `" + database.database_id() +
                               "`");
  request.add_extra_statements("ALTER DATABASE `" + database.database_id() +
                               "` " +
                               "SET OPTIONS (version_retention_period='2h')");
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

  auto ddl = client.GetDatabaseDdl(db->name());
  if (!ddl) throw std::move(ddl).status();
  std::cout << "Database DDL is:\n" << ddl->DebugString();
}
```

### Go

```go
import (
    "context"
    "fmt"
    "io"
    "regexp"

    database "cloud.google.com/go/spanner/admin/database/apiv1"
    adminpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
)

func createDatabaseWithRetentionPeriod(ctx context.Context, w io.Writer, db string) error {
    matches := regexp.MustCompile("^(.+)/databases/(.+)$").FindStringSubmatch(db)
    if matches == nil || len(matches) != 3 {
        return fmt.Errorf("createDatabaseWithRetentionPeriod: invalid database id %q", db)
    }

    adminClient, err := database.NewDatabaseAdminClient(ctx)
    if err != nil {
        return fmt.Errorf("createDatabaseWithRetentionPeriod.NewDatabaseAdminClient: %w", err)
    }
    defer adminClient.Close()

    // Create a database with a version retention period of 7 days.
    retentionPeriod := "7d"
    alterDatabase := fmt.Sprintf(
        "ALTER DATABASE `%s` SET OPTIONS (version_retention_period = '%s')",
        matches[2], retentionPeriod,
    )
    req := adminpb.CreateDatabaseRequest{
        Parent:          matches[1],
        CreateStatement: "CREATE DATABASE `" + matches[2] + "`",
        ExtraStatements: []string{
            `CREATE TABLE Singers (
                SingerId   INT64 NOT NULL,
                FirstName  STRING(1024),
                LastName   STRING(1024),
                SingerInfo BYTES(MAX)
            ) PRIMARY KEY (SingerId)`,
            `CREATE TABLE Albums (
                SingerId     INT64 NOT NULL,
                AlbumId      INT64 NOT NULL,
                AlbumTitle   STRING(MAX)
            ) PRIMARY KEY (SingerId, AlbumId),
            INTERLEAVE IN PARENT Singers ON DELETE CASCADE`,
            alterDatabase,
        },
    }
    op, err := adminClient.CreateDatabase(ctx, &req)
    if err != nil {
        return fmt.Errorf("createDatabaseWithRetentionPeriod.CreateDatabase: %w", err)
    }
    if _, err := op.Wait(ctx); err != nil {
        return fmt.Errorf("createDatabaseWithRetentionPeriod.Wait: %w", err)
    }
    fmt.Fprintf(w, "Created database [%s] with version retention period %q\n", db, retentionPeriod)
    return nil
}
```

### Java

```java
import com.google.cloud.spanner.Spanner;
import com.google.cloud.spanner.SpannerException;
import com.google.cloud.spanner.SpannerExceptionFactory;
import com.google.cloud.spanner.SpannerOptions;
import com.google.cloud.spanner.admin.database.v1.DatabaseAdminClient;
import com.google.common.collect.Lists;
import com.google.spanner.admin.database.v1.CreateDatabaseRequest;
import com.google.spanner.admin.database.v1.Database;
import com.google.spanner.admin.database.v1.InstanceName;
import java.util.concurrent.ExecutionException;

public class CreateDatabaseWithVersionRetentionPeriodSample {

  static void createDatabaseWithVersionRetentionPeriod() {
    // TODO(developer): Replace these variables before running the sample.
    String projectId = "my-project";
    String instanceId = "my-instance";
    String databaseId = "my-database";
    String versionRetentionPeriod = "7d";

    createDatabaseWithVersionRetentionPeriod(projectId, instanceId, databaseId,
        versionRetentionPeriod);
  }

  static void createDatabaseWithVersionRetentionPeriod(String projectId,
      String instanceId, String databaseId, String versionRetentionPeriod) {
    try (Spanner spanner =
        SpannerOptions.newBuilder().setProjectId(projectId).build().getService();
        DatabaseAdminClient databaseAdminClient = spanner.createDatabaseAdminClient()) {
      CreateDatabaseRequest request =
          CreateDatabaseRequest.newBuilder()
              .setParent(InstanceName.of(projectId, instanceId).toString())
              .setCreateStatement("CREATE DATABASE `" + databaseId + "`")
              .addAllExtraStatements(Lists.newArrayList("ALTER DATABASE " + "`" + databaseId + "`"
                  + " SET OPTIONS ( version_retention_period = '" + versionRetentionPeriod + "' )"))
              .build();
      Database database =
          databaseAdminClient.createDatabaseAsync(request).get();
      System.out.println("Created database [" + database.getName() + "]");
      System.out.println("\tVersion retention period: " + database.getVersionRetentionPeriod());
      System.out.println("\tEarliest version time: " + database.getEarliestVersionTime());
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

```javascript
// Imports the Google Cloud client library
const {Spanner} = require('@google-cloud/spanner');

/**
 * TODO(developer): Uncomment the following lines before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance';

// creates a client
const spanner = new Spanner({
  projectId: projectId,
});

// Gets a reference to a Cloud Spanner Database Admin Client object
const databaseAdminClient = spanner.getDatabaseAdminClient();

try {
  // Create a new database with an extra statement which will alter the
  // database after creation to set the version retention period.
  console.log(
    `Creating database ${databaseAdminClient.instancePath(
      projectId,
      instanceId,
    )}.`,
  );
  const versionRetentionStatement = `
    ALTER DATABASE \`${databaseId}\`
    SET OPTIONS (version_retention_period = '1d')`;

  const [operation] = await databaseAdminClient.createDatabase({
    createStatement: 'CREATE DATABASE `' + databaseId + '`',
    extraStatements: [versionRetentionStatement],
    parent: databaseAdminClient.instancePath(projectId, instanceId),
  });

  console.log(`Waiting for operation on ${databaseId} to complete...`);
  await operation.promise();
  console.log(`
      Created database ${databaseId} with version retention period.`);

  const [metadata] = await databaseAdminClient.getDatabase({
    name: databaseAdminClient.databasePath(projectId, instanceId, databaseId),
  });

  console.log(`Version retention period: ${metadata.versionRetentionPeriod}`);
  const milliseconds =
    parseInt(metadata.earliestVersionTime.seconds, 10) * 1000 +
    parseInt(metadata.earliestVersionTime.nanos, 10) / 1e6;
  const date = new Date(milliseconds);
  console.log(`Earliest version time: ${date.toString()}`);
} catch (err) {
  console.error('ERROR:', err);
}
```

> **Note:** The old client library interface code samples for Node.js are archived in [GitHub](https://github.com/googleapis/nodejs-spanner/tree/main/samples/archived) .

### PHP

```php
use Google\Cloud\Spanner\Admin\Database\V1\Client\DatabaseAdminClient;
use Google\Cloud\Spanner\Admin\Database\V1\CreateDatabaseRequest;
use Google\Cloud\Spanner\Admin\Database\V1\GetDatabaseRequest;

/**
 * Creates a database with data retention for Point In Time Restore.
 * Example:
 * ```
 * create_database_with_version_retention_period($projectId, $instanceId, $databaseId, $retentionPeriod);
 * ```
 *
 * @param string $projectId The Google Cloud project ID.
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 * @param string $retentionPeriod The data retention period for the database.
 */
function create_database_with_version_retention_period(
    string $projectId,
    string $instanceId,
    string $databaseId,
    string $retentionPeriod
): void {
    $databaseAdminClient = new DatabaseAdminClient();
    $instance = $databaseAdminClient->instanceName($projectId, $instanceId);
    $databaseFullName = $databaseAdminClient->databaseName($projectId, $instanceId, $databaseId);

    $operation = $databaseAdminClient->createDatabase(
        new CreateDatabaseRequest([
            'parent' => $instance,
            'create_statement' => sprintf('CREATE DATABASE `%s`', $databaseId),
            'extra_statements' => [
                'CREATE TABLE Singers (' .
                'SingerId     INT64 NOT NULL,' .
                'FirstName    STRING(1024),' .
                'LastName     STRING(1024),' .
                'SingerInfo   BYTES(MAX)' .
                ') PRIMARY KEY (SingerId)',
                'CREATE TABLE Albums (' .
                    'SingerId     INT64 NOT NULL,' .
                    'AlbumId      INT64 NOT NULL,' .
                    'AlbumTitle   STRING(MAX)' .
                ') PRIMARY KEY (SingerId, AlbumId),' .
                'INTERLEAVE IN PARENT Singers ON DELETE CASCADE',
                "ALTER DATABASE `$databaseId` SET OPTIONS(version_retention_period='$retentionPeriod')"
            ]
        ])
    );

    print('Waiting for operation to complete...' . PHP_EOL);
    $operation->pollUntilComplete();

    $request = new GetDatabaseRequest(['name' => $databaseFullName]);
    $databaseInfo = $databaseAdminClient->getDatabase($request);

    print(sprintf(
        'Database %s created with version retention period %s',
        $databaseInfo->getName(), $databaseInfo->getVersionRetentionPeriod()
    ) . PHP_EOL);
}
```

> **Note:** The old client library interface code samples for PHP are archived in [GitHub](https://github.com/GoogleCloudPlatform/php-docs-samples/tree/main/spanner/src/admin/archived) .

### Python

```python
def create_database_with_version_retention_period(
    instance_id, database_id, retention_period
):
    """Creates a database with a version retention period."""

    from google.cloud.spanner_admin_database_v1.types import spanner_database_admin

    spanner_client = spanner.Client()
    database_admin_api = spanner_client.database_admin_api
    ddl_statements = [
        "CREATE TABLE Singers ("
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
        "ALTER DATABASE `{}`"
        " SET OPTIONS (version_retention_period = '{}')".format(
            database_id, retention_period
        ),
    ]
    operation = database_admin_api.create_database(
        request=spanner_database_admin.CreateDatabaseRequest(
            parent=database_admin_api.instance_path(
                spanner_client.project, instance_id
            ),
            create_statement="CREATE DATABASE `{}`".format(database_id),
            extra_statements=ddl_statements,
        )
    )

    db = operation.result(30)
    print(
        "Database {} created with version retention period {} and earliest version time {}".format(
            db.name, db.version_retention_period, db.earliest_version_time
        )
    )

    database_admin_api.drop_database(
        spanner_database_admin.DropDatabaseRequest(database=db.name)
    )
```

> **Note:** The old client library interface code samples for Python are archived in [GitHub](https://github.com/googleapis/python-spanner/tree/main/samples/samples/archived) .

### Ruby

```ruby
# project_id  = "Your Google Cloud project ID"
# instance_id = "Your Spanner instance ID"
# database_id = "Your Spanner database ID"

require "google/cloud/spanner"
require "google/cloud/spanner/admin/database"

database_admin_client = Google::Cloud::Spanner::Admin::Database.database_admin

instance_path = database_admin_client.instance_path project: project_id, instance: instance_id

version_retention_period = "7d"

db_path = database_admin_client.database_path project: project_id,
                                              instance: instance_id,
                                              database: database_id

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
  INTERLEAVE IN PARENT Singers ON DELETE CASCADE",

                                              "ALTER DATABASE `#{database_id}`
    SET OPTIONS ( version_retention_period = '#{version_retention_period}' )"
                                            ]

puts "Waiting for create database operation to complete"

job.wait_until_done!
database = database_admin_client.get_database name: db_path

puts "Created database #{database_id} on instance #{instance_id}"
puts "\tVersion retention period: #{database.version_retention_period}"
puts "\tEarliest version time: #{database.earliest_version_time}"
```

Usage notes:

- The retention period must be between 1 hour and 7 days, and can be specified in days, hours, minutes, or seconds. For example, the values `1d` , `24h` , `1440m` , and `86400s` are equivalent.
- If you have [enabled logging](https://docs.cloud.google.com/logging/docs/audit/configure-data-access#config-console-enable) for the Spanner API in your project, the event is logged as [UpdateDatabaseDdl](https://docs.cloud.google.com/spanner/docs/audit-logging) and is visible in the [Logs Explorer](https://docs.cloud.google.com/logging/docs/view/logs-explorer-summary) .
- To revert to the default retention period of 1 hour, you can set the `version_retention_period` database option to `NULL` for GoogleSQL databases or `DEFAULT` for PostgreSQL databases.
- When you extend the retention period, the system doesn't backfill previous versions of data. For example, if you extend the retention period from one hour to 24 hours, then you must wait 23 hours for the system to accumulate old data before you can recover data from 24 hours in the past.

## Get the retention period and earliest version time

The [Database](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases#Database) resource includes two fields:

- [`version_retention_period`](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases#Database.FIELDS.version_retention_period) : the period in which Spanner retains all versions of data for the database.

- [`earliest_version_time`](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases#Database.FIELDS.earliest_version_time) : the earliest timestamp at which earlier versions of the data can be read from the database. This value is continuously updated by Spanner and becomes stale the moment it's queried. If you are using this value to recover data, make sure to account for the time from the moment when the value is queried to the moment when you initiate the recovery.

### Console

1.  Go to the Spanner Instances page in the Google Cloud console.

2.  Click the instance containing the database to open its **Overview** page.

3.  Click the database to open its **Overview** page.

4.  Select the **Backup/Restore** tab to open the **Backup/Restore** page and display the retention period.

    ![Backup/Restore form in the console, showing the Version retention period field set to 1 hour and a pencil icon for editing this value.](https://docs.cloud.google.com/static/spanner/docs/images/pitr-retention-period-update.png)

5.  Click **Create** to open the **Create a backup** page and display the earliest version time.

    ![Create backup form in the console with the Create backup from an earlier point in time option checked and showing the earliest version time.](https://docs.cloud.google.com/static/spanner/docs/images/pitr-backup.png)

### gcloud

You can get these fields by calling [describe databases](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/databases/describe) or [list databases](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/databases/list) . For example:

```
gcloud spanner databases describe example-db \
    --instance=test-instance
```

The following shows the output:

```
createTime: '2020-09-07T16:56:08.285140Z'
earliestVersionTime: '2020-10-07T16:56:08.285140Z'
name: projects/my-project/instances/test-instance/databases/example-db
state: READY
versionRetentionPeriod: 3d
```

## Recover a portion of your database

1.  Perform a [stale read](https://docs.cloud.google.com/spanner/docs/timestamp-bounds#exact_staleness) and specify the needed recovery timestamp. Make sure that the timestamp you specify is more recent than the database's `earliest_version_time.`

    ### gcloud

    Use [execute-sql](https://docs.cloud.google.com/sdk/gcloud/reference/spanner/databases/execute-sql) For example:

    ```
    gcloud spanner databases execute-sql example-db \
        --instance=test-instance \
        --read-timestamp=2020-09-11T10:19:36.010459-07:00 \
        --sql='SELECT * FROM SINGERS'
    ```

    ### Client libraries

    See [perform stale read](https://docs.cloud.google.com/spanner/docs/reads#perform-stale-read) .

2.  Store the results of the query. This is required because you can't write the results of the query back into the database in the same transaction. For small amounts of data, you can print to console or store in-memory. For larger amounts of data, you may need to write to a local file.

3.  Write the recovered data back to the table that needs to be recovered. For example:

    ### gcloud

    > **Note:** The `gcloud spanner rows update` command doesn't isn't supported for the PostgreSQL interface for Spanner.

    ```
    gcloud spanner rows update --instance=test-instance \
        --database=example-db --table=Singers \
        --data=SingerId=1,FirstName='Marc'
    ```

    For more information, see [updating data using gcloud](https://docs.cloud.google.com/spanner/docs/modify-gcloud) .

    ### Client libraries

    For more information, see [updating data using DML](https://docs.cloud.google.com/spanner/docs/dml-tasks) or [updating data using mutations](https://docs.cloud.google.com/spanner/docs/modify-mutation-api) .

    Optionally, if you want to do some analysis on the recovered data before writing back, you can manually create a temporay table in the same database, write the recovered data to this temporary table first, do the analysis, and then read the data you want to recover from this temporary table and write it to the table that needs to be recovered.

## Recover a specific table

If you need to recover a specific table and the chosen recovery point is more recent than the database's `earliest_version_time` , you can export the table from an earlier timestamp within the same database and then import it back.

The import process doesn't automatically delete rows that were added after the backup was created. Therefore, if you want the recovered table to exactly match its state at the chosen restore timestamp, you must manually delete the rows that were created after the restore timestamp.

1.  Export the specific table from the database specifying the `snapshotTime` parameter to the needed recovery timestamp. You can use the [Spanner to Avro](https://docs.cloud.google.com/dataflow/docs/guides/templates/provided-batch#cloud_spanner_to_gcs_avro) Dataflow template. For more information, see [Exporting data](https://docs.cloud.google.com/spanner/docs/export) .
2.  Import the exported table back into the database using the [Cloud Storage Avro to Spanner](https://docs.cloud.google.com/dataflow/docs/guides/templates/provided-batch#gcs_avro_to_cloud_spanner) Dataflow template. For more information, see [Importing data](https://docs.cloud.google.com/spanner/docs/import) .

## Recover an entire database

You can recover the entire database using either [Backup and Restore](https://docs.cloud.google.com/spanner/docs/backup) or [Import and Export](https://docs.cloud.google.com/spanner/docs/export) and specifying a recovery timestamp.

### Backup and restore

1.  Create a backup and set the [`version_time`](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.backups#Backup.FIELDS.version_time) to the needed recovery timestamp.

    ### Console

    1.  Go to the **Database details** page in the Cloud console.

    2.  In the **Backup/Restore** tab, click **Create** .

    3.  Check the **Create backup from an earlier point in time** box.

        ![Create backup form in the console with the Create backup from an earlier point in time option checked and showing the earliest version time.](https://docs.cloud.google.com/static/spanner/docs/images/pitr-backup.png)

    ### gcloud

    ```
    gcloud spanner backups create example-db-backup-1 \
        --instance=test-instance \
        --database=example-db \
        --retention-period=1y \
        --version-time=2021-01-22T01:10:35Z --async
    ```

    For more information, see [Create a backup using gcloud](https://docs.cloud.google.com/spanner/docs/backup/gcloud#create) .

    ### Client libraries

    ### C#

    ```csharp
    // Copyright 2020 Google Inc.
    //
    // Licensed under the Apache License, Version 2.0 (the "License");
    // you may not use this file except in compliance with the License.
    // You may obtain a copy of the License at
    //
    //     http://www.apache.org/licenses/LICENSE-2.0
    //
    // Unless required by applicable law or agreed to in writing, software
    // distributed under the License is distributed on an "AS IS" BASIS,
    // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    // See the License for the specific language governing permissions and
    // limitations under the License.


    using Google.Cloud.Spanner.Admin.Database.V1;
    using Google.Cloud.Spanner.Common.V1;
    using Google.LongRunning;
    using Google.Protobuf.WellKnownTypes;
    using System;

    public class CreateBackupSample
    {
        public Backup CreateBackup(string projectId, string instanceId, string databaseId, string backupId, DateTime versionTime)
        {
            // Create the DatabaseAdminClient instance.
            DatabaseAdminClient databaseAdminClient = DatabaseAdminClient.Create();

            // Initialize request parameters.
            Backup backup = new Backup
            {
                DatabaseAsDatabaseName = DatabaseName.FromProjectInstanceDatabase(projectId, instanceId, databaseId),
                ExpireTime = DateTime.UtcNow.AddDays(14).ToTimestamp(),
                VersionTime = versionTime.ToTimestamp(),
            };
            InstanceName instanceName = InstanceName.FromProjectInstance(projectId, instanceId);

            // Make the CreateBackup request.
            Operation<Backup, CreateBackupMetadata> response = databaseAdminClient.CreateBackup(instanceName, backup, backupId);

            Console.WriteLine("Waiting for the operation to finish.");

            // Poll until the returned long-running operation is complete.
            Operation<Backup, CreateBackupMetadata> completedResponse = response.PollUntilCompleted();

            if (completedResponse.IsFaulted)
            {
                Console.WriteLine($"Error while creating backup: {completedResponse.Exception}");
                throw completedResponse.Exception;
            }

            Console.WriteLine($"Backup created successfully.");

            // GetBackup to get more information about the created backup.
            BackupName backupName = BackupName.FromProjectInstanceBackup(projectId, instanceId, backupId);
            backup = databaseAdminClient.GetBackup(backupName);
            Console.WriteLine($"Backup {backup.Name} of size {backup.SizeBytes} bytes " +
                          $"was created at {backup.CreateTime} from {backup.Database} " +
                          $"and is in state {backup.State} " +
                          $"and has version time {backup.VersionTime}");
            return backup;
        }
    }
    ```

    ### C++

    ```cpp
    void CreateBackup(google::cloud::spanner_admin::DatabaseAdminClient client,
                      std::string const& project_id, std::string const& instance_id,
                      std::string const& database_id, std::string const& backup_id,
                      google::cloud::spanner::Timestamp expire_time,
                      google::cloud::spanner::Timestamp version_time) {
      google::cloud::spanner::Database database(project_id, instance_id,
                                                database_id);
      google::spanner::admin::database::v1::CreateBackupRequest request;
      request.set_parent(database.instance().FullName());
      request.set_backup_id(backup_id);
      request.mutable_backup()->set_database(database.FullName());
      *request.mutable_backup()->mutable_expire_time() =
          expire_time.get<google::protobuf::Timestamp>().value();
      *request.mutable_backup()->mutable_version_time() =
          version_time.get<google::protobuf::Timestamp>().value();
      auto backup = client.CreateBackup(request).get();
      if (!backup) throw std::move(backup).status();
      std::cout
          << "Backup " << backup->name() << " of " << backup->database()
          << " of size " << backup->size_bytes() << " bytes as of "
          << google::cloud::spanner::MakeTimestamp(backup->version_time()).value()
          << " was created at "
          << google::cloud::spanner::MakeTimestamp(backup->create_time()).value()
          << ".\n";
    }
    ```

    ### Go

    ```go
    import (
        "context"
        "fmt"
        "io"
        "regexp"
        "time"

        database "cloud.google.com/go/spanner/admin/database/apiv1"
        adminpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
        pbt "github.com/golang/protobuf/ptypes/timestamp"
    )

    func createBackup(ctx context.Context, w io.Writer, db, backupID string, versionTime time.Time) error {
        // versionTime := time.Now().AddDate(0, 0, -1) // one day ago
        matches := regexp.MustCompile("^(.+)/databases/(.+)$").FindStringSubmatch(db)
        if matches == nil || len(matches) != 3 {
            return fmt.Errorf("createBackup: invalid database id %q", db)
        }

        adminClient, err := database.NewDatabaseAdminClient(ctx)
        if err != nil {
            return fmt.Errorf("createBackup.NewDatabaseAdminClient: %w", err)
        }
        defer adminClient.Close()

        expireTime := time.Now().AddDate(0, 0, 14)
        // Create a backup.
        req := adminpb.CreateBackupRequest{
            Parent:   matches[1],
            BackupId: backupID,
            Backup: &adminpb.Backup{
                Database:    db,
                ExpireTime:  &pbt.Timestamp{Seconds: expireTime.Unix(), Nanos: int32(expireTime.Nanosecond())},
                VersionTime: &pbt.Timestamp{Seconds: versionTime.Unix(), Nanos: int32(versionTime.Nanosecond())},
            },
        }
        op, err := adminClient.CreateBackup(ctx, &req)
        if err != nil {
            return fmt.Errorf("createBackup.CreateBackup: %w", err)
        }
        // Wait for backup operation to complete.
        backup, err := op.Wait(ctx)
        if err != nil {
            return fmt.Errorf("createBackup.Wait: %w", err)
        }

        // Get the name, create time, version time and backup size.
        backupCreateTime := time.Unix(backup.CreateTime.Seconds, int64(backup.CreateTime.Nanos))
        backupVersionTime := time.Unix(backup.VersionTime.Seconds, int64(backup.VersionTime.Nanos))
        fmt.Fprintf(w,
            "Backup %s of size %d bytes was created at %s with version time %s\n",
            backup.Name,
            backup.SizeBytes,
            backupCreateTime.Format(time.RFC3339),
            backupVersionTime.Format(time.RFC3339))
        return nil
    }
    ```

    ### Java

    ```java
    static void createBackup(DatabaseAdminClient dbAdminClient, String projectId, String instanceId,
        String databaseId, String backupId, Timestamp versionTime) {
      // Set expire time to 14 days from now.
      Timestamp expireTime =
          Timestamp.newBuilder().setSeconds(TimeUnit.MILLISECONDS.toSeconds((
              System.currentTimeMillis() + TimeUnit.DAYS.toMillis(14)))).build();
      BackupName backupName = BackupName.of(projectId, instanceId, backupId);
      Backup backup = Backup.newBuilder()
          .setName(backupName.toString())
          .setDatabase(DatabaseName.of(projectId, instanceId, databaseId).toString())
          .setExpireTime(expireTime).setVersionTime(versionTime).build();

      // Initiate the request which returns an OperationFuture.
      System.out.println("Creating backup [" + backupId + "]...");
      try {
        // Wait for the backup operation to complete.
        backup = dbAdminClient.createBackupAsync(
            InstanceName.of(projectId, instanceId), backup, backupId).get();
        System.out.println("Created backup [" + backup.getName() + "]");
      } catch (ExecutionException e) {
        throw SpannerExceptionFactory.asSpannerException(e);
      } catch (InterruptedException e) {
        throw SpannerExceptionFactory.propagateInterrupt(e);
      }

      // Reload the metadata of the backup from the server.
      backup = dbAdminClient.getBackup(backup.getName());
      System.out.println(
          String.format(
              "Backup %s of size %d bytes was created at %s for version of database at %s",
              backup.getName(),
              backup.getSizeBytes(),
              java.time.OffsetDateTime.ofInstant(
                  Instant.ofEpochSecond(backup.getCreateTime().getSeconds(),
                      backup.getCreateTime().getNanos()), ZoneId.systemDefault()),
              java.time.OffsetDateTime.ofInstant(
                  Instant.ofEpochSecond(backup.getVersionTime().getSeconds(),
                      backup.getVersionTime().getNanos()), ZoneId.systemDefault()))
      );
    }
    ```

    > **Note:** The old client library interface code samples for Java are archived in [GitHub](https://github.com/googleapis/google-cloud-java/tree/main/java-spanner/samples/snippets/src/main/java/com/example/spanner/admin/archived) .

    ### Node.js

    ```javascript
    // Imports the Google Cloud client library and precise date library
    const {Spanner, protos} = require('@google-cloud/spanner');
    const {PreciseDate} = require('@google-cloud/precise-date');

    /**
     * TODO(developer): Uncomment the following lines before running the sample.
     */
    // const projectId = 'my-project-id';
    // const instanceId = 'my-instance';
    // const databaseId = 'my-database';
    // const backupId = 'my-backup';
    // const versionTime = Date.now() - 1000 * 60 * 60 * 24; // One day ago

    // Creates a client
    const spanner = new Spanner({
      projectId: projectId,
    });

    // Gets a reference to a Cloud Spanner Database Admin Client object
    const databaseAdminClient = spanner.getDatabaseAdminClient();

    // Creates a new backup of the database
    try {
      console.log(
        `Creating backup of database ${databaseAdminClient.databasePath(
          projectId,
          instanceId,
          databaseId,
        )}.`,
      );

      // Expire backup 14 days in the future
      const expireTime = Date.now() + 1000 * 60 * 60 * 24 * 14;

      // Create a backup of the state of the database at the current time.
      const [operation] = await databaseAdminClient.createBackup({
        parent: databaseAdminClient.instancePath(projectId, instanceId),
        backupId: backupId,
        backup: (protos.google.spanner.admin.database.v1.Backup = {
          database: databaseAdminClient.databasePath(
            projectId,
            instanceId,
            databaseId,
          ),
          expireTime: Spanner.timestamp(expireTime).toStruct(),
          versionTime: Spanner.timestamp(versionTime).toStruct(),
          name: databaseAdminClient.backupPath(projectId, instanceId, backupId),
        }),
      });

      console.log(
        `Waiting for backup ${databaseAdminClient.backupPath(
          projectId,
          instanceId,
          backupId,
        )} to complete...`,
      );
      await operation.promise();

      // Verify backup is ready
      const [backupInfo] = await databaseAdminClient.getBackup({
        name: databaseAdminClient.backupPath(projectId, instanceId, backupId),
      });
      if (backupInfo.state === 'READY') {
        console.log(
          `Backup ${backupInfo.name} of size ` +
            `${backupInfo.sizeBytes} bytes was created at ` +
            `${new PreciseDate(backupInfo.createTime).toISOString()} ` +
            'for version of database at ' +
            `${new PreciseDate(backupInfo.versionTime).toISOString()}`,
        );
      } else {
        console.error('ERROR: Backup is not ready.');
      }
    } catch (err) {
      console.error('ERROR:', err);
    } finally {
      // Close the spanner client when finished.
      // The databaseAdminClient does not require explicit closure. The closure of the Spanner client will automatically close the databaseAdminClient.
      spanner.close();
    }
    ```

    > **Note:** The old client library interface code samples for Node.js are archived in [GitHub](https://github.com/googleapis/nodejs-spanner/tree/main/samples/archived) .

    ### PHP

    ```php
    use Google\Cloud\Spanner\Admin\Database\V1\Backup;
    use Google\Cloud\Spanner\Admin\Database\V1\GetBackupRequest;
    use Google\Cloud\Spanner\Admin\Database\V1\Client\DatabaseAdminClient;
    use Google\Cloud\Spanner\Admin\Database\V1\CreateBackupRequest;
    use Google\Protobuf\Timestamp;

    /**
     * Create a backup.
     * Example:
     * ```
     * create_backup($projectId, $instanceId, $databaseId, $backupId, $versionTime);
     * ```
     *
     * @param string $projectId The Google Cloud project ID.
     * @param string $instanceId The Spanner instance ID.
     * @param string $databaseId The Spanner database ID.
     * @param string $backupId The Spanner backup ID.
     * @param string $versionTime The version of the database to backup. Read more
     * at https://cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.backups#Backup.FIELDS.version_time
     */
    function create_backup(
        string $projectId,
        string $instanceId,
        string $databaseId,
        string $backupId,
        string $versionTime = '-1hour'
    ): void {
        $databaseAdminClient = new DatabaseAdminClient();
        $databaseFullName = DatabaseAdminClient::databaseName($projectId, $instanceId, $databaseId);
        $instanceFullName = DatabaseAdminClient::instanceName($projectId, $instanceId);
        $timestamp = new Timestamp();
        $timestamp->setSeconds((new \DateTime($versionTime))->getTimestamp());
        $expireTime = new Timestamp();
        $expireTime->setSeconds((new \DateTime('+14 days'))->getTimestamp());
        $request = new CreateBackupRequest([
            'parent' => $instanceFullName,
            'backup_id' => $backupId,
            'backup' => new Backup([
                'database' => $databaseFullName,
                'expire_time' => $expireTime,
                'version_time' => $timestamp
            ])
        ]);

        $operation = $databaseAdminClient->createBackup($request);

        print('Waiting for operation to complete...' . PHP_EOL);
        $operation->pollUntilComplete();

        $request = new GetBackupRequest();
        $request->setName($databaseAdminClient->backupName($projectId, $instanceId, $backupId));
        $info = $databaseAdminClient->getBackup($request);
        printf(
            'Backup %s of size %d bytes was created at %d for version of database at %d' . PHP_EOL,
            basename($info->getName()),
            $info->getSizeBytes(),
            $info->getCreateTime()->getSeconds(),
            $info->getVersionTime()->getSeconds());
    }
    ```

    > **Note:** The old client library interface code samples for PHP are archived in [GitHub](https://github.com/GoogleCloudPlatform/php-docs-samples/tree/main/spanner/src/admin/archived) .

    ### Python

    ```python
    def create_backup(instance_id, database_id, backup_id, version_time):
        """Creates a backup for a database."""

        from google.cloud.spanner_admin_database_v1.types import backup as backup_pb

        spanner_client = spanner.Client()
        database_admin_api = spanner_client.database_admin_api

        # Create a backup
        expire_time = datetime.utcnow() + timedelta(days=14)

        request = backup_pb.CreateBackupRequest(
            parent=database_admin_api.instance_path(spanner_client.project, instance_id),
            backup_id=backup_id,
            backup=backup_pb.Backup(
                database=database_admin_api.database_path(
                    spanner_client.project, instance_id, database_id
                ),
                expire_time=expire_time,
                version_time=version_time,
            ),
        )

        operation = database_admin_api.create_backup(request)

        # Wait for backup operation to complete.
        backup = operation.result(2100)

        # Verify that the backup is ready.
        assert backup.state == backup_pb.Backup.State.READY

        print(
            "Backup {} of size {} bytes was created at {} for version of database at {}".format(
                backup.name, backup.size_bytes, backup.create_time, backup.version_time
            )
        )
    ```

    > **Note:** The old client library interface code samples for Python are archived in [GitHub](https://github.com/googleapis/python-spanner/tree/main/samples/samples/archived) .

    ### Ruby

    ```ruby
    # project_id  = "Your Google Cloud project ID"
    # instance_id = "Your Spanner instance ID"
    # database_id = "Your Spanner database ID"
    # backup_id = "Your Spanner backup ID"
    # version_time = Time.now - 60 * 60 * 24 # 1 day ago

    require "google/cloud/spanner"
    require "google/cloud/spanner/admin/database"

    database_admin_client = Google::Cloud::Spanner::Admin::Database.database_admin

    instance_path = database_admin_client.instance_path project: project_id, instance: instance_id
    db_path = database_admin_client.database_path project: project_id,
                                                  instance: instance_id,
                                                  database: database_id
    backup_path = database_admin_client.backup_path project: project_id,
                                                    instance: instance_id,
                                                    backup: backup_id
    expire_time = Time.now + (14 * 24 * 3600) # 14 days from now

    job = database_admin_client.create_backup parent: instance_path,
                                              backup_id: backup_id,
                                              backup: {
                                                database: db_path,
                                                  expire_time: expire_time,
                                                  version_time: version_time
                                              }

    puts "Backup operation in progress"

    job.wait_until_done!

    backup = database_admin_client.get_backup name: backup_path
    puts "Backup #{backup_id} of size #{backup.size_bytes} bytes was created at #{backup.create_time} for version of database at #{backup.version_time}"
    ```

2.  Restore from the backup to a new database. Note that Spanner preserves the retention period setting from the backup to the restored database.

    ### Console

    1.  Go to the **Instance details** page in the Cloud console.

    2.  In the **Backup/Restore** tab, select a backup and click **Restore** .

    ### gcloud

    ```
    gcloud spanner databases restore --async \
      --destination-instance=destination-instance --destination-database=example-db-restored \
      --source-instance=test-instance --source-backup=example-db-backup-1
    ```

    For more information, see [Restoring a database from a backup](https://docs.cloud.google.com/spanner/docs/backup/gcloud#restore) .

    ### Client libraries

    ### C#

    ```csharp
    using Google.Cloud.Spanner.Admin.Database.V1;
    using Google.Cloud.Spanner.Common.V1;
    using Google.LongRunning;
    using System;

    public class RestoreDatabaseSample
    {
        public RestoreInfo RestoreDatabase(string projectId, string instanceId, string databaseId, string backupId)
        {
            // Create the DatabaseAdminClient instance.
            DatabaseAdminClient databaseAdminClient = DatabaseAdminClient.Create();

            InstanceName parentAsInstanceName = InstanceName.FromProjectInstance(projectId, instanceId);
            BackupName backupAsBackupName = BackupName.FromProjectInstanceBackup(projectId, instanceId, backupId);

            // Make the RestoreDatabase request.
            Operation<Database, RestoreDatabaseMetadata> response = databaseAdminClient.RestoreDatabase(parentAsInstanceName, databaseId, backupAsBackupName);

            Console.WriteLine("Waiting for the operation to finish");

            // Poll until the returned long-running operation is complete.
            var completedResponse = response.PollUntilCompleted();

            if (completedResponse.IsFaulted)
            {
                Console.WriteLine($"Database Restore Failed: {completedResponse.Exception}");
                throw completedResponse.Exception;
            }

            RestoreInfo restoreInfo = completedResponse.Result.RestoreInfo;
            Console.WriteLine(
                $"Database {restoreInfo.BackupInfo.SourceDatabase} was restored " +
                $"to {databaseId} from backup {restoreInfo.BackupInfo.Backup} " +
                $"with version time {restoreInfo.BackupInfo.VersionTime}");

            return restoreInfo;
        }
    }
    ```

    ### C++

    ```cpp
    void RestoreDatabase(google::cloud::spanner_admin::DatabaseAdminClient client,
                         std::string const& project_id,
                         std::string const& instance_id,
                         std::string const& database_id,
                         std::string const& backup_id) {
      google::cloud::spanner::Database database(project_id, instance_id,
                                                database_id);
      google::cloud::spanner::Backup backup(database.instance(), backup_id);
      auto restored_db =
          client
              .RestoreDatabase(database.instance().FullName(),
                               database.database_id(), backup.FullName())
              .get();
      if (!restored_db) throw std::move(restored_db).status();
      std::cout << "Database";
      if (restored_db->restore_info().source_type() ==
          google::spanner::admin::database::v1::BACKUP) {
        auto const& backup_info = restored_db->restore_info().backup_info();
        std::cout << " " << backup_info.source_database() << " as of "
                  << google::cloud::spanner::MakeTimestamp(
                         backup_info.version_time())
                         .value();
      }
      std::cout << " restored to " << restored_db->name();
      std::cout << " from backup " << backup.FullName();
      std::cout << ".\n";
    }
    ```

    ### Go

    ```go
    import (
        "context"
        "fmt"
        "io"
        "regexp"

        database "cloud.google.com/go/spanner/admin/database/apiv1"
        adminpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
    )

    func restoreBackup(ctx context.Context, w io.Writer, db, backupID string) error {
        adminClient, err := database.NewDatabaseAdminClient(ctx)
        if err != nil {
            return err
        }
        defer adminClient.Close()

        matches := regexp.MustCompile("^(.*)/databases/(.*)$").FindStringSubmatch(db)
        if matches == nil || len(matches) != 3 {
            return fmt.Errorf("Invalid database id %s", db)
        }
        instanceName := matches[1]
        databaseID := matches[2]
        backupName := instanceName + "/backups/" + backupID

        // Start restoring backup to a new database.
        restoreOp, err := adminClient.RestoreDatabase(ctx, &adminpb.RestoreDatabaseRequest{
            Parent:     instanceName,
            DatabaseId: databaseID,
            Source: &adminpb.RestoreDatabaseRequest_Backup{
                Backup: backupName,
            },
        })
        if err != nil {
            return err
        }
        // Wait for restore operation to complete.
        dbObj, err := restoreOp.Wait(ctx)
        if err != nil {
            return err
        }
        // Newly created database has restore information.
        backupInfo := dbObj.RestoreInfo.GetBackupInfo()
        if backupInfo != nil {
            fmt.Fprintf(w, "Source database %s restored from backup %s\n", backupInfo.SourceDatabase, backupInfo.Backup)
        }

        return nil
    }
    ```

    ### Java

    ```java
    static void restoreBackup(
        DatabaseAdminClient dbAdminClient,
        String projectId,
        String instanceId,
        String backupId,
        String restoreToDatabaseId) {
      BackupName backupName = BackupName.of(projectId, instanceId, backupId);
      Backup backup = dbAdminClient.getBackup(backupName);
      // Initiate the request which returns an OperationFuture.
      System.out.println(String.format(
          "Restoring backup [%s] to database [%s]...", backup.getName(), restoreToDatabaseId));
      try {
        RestoreDatabaseRequest request =
            RestoreDatabaseRequest.newBuilder()
                .setParent(InstanceName.of(projectId, instanceId).toString())
                .setDatabaseId(restoreToDatabaseId)
                .setBackup(backupName.toString()).build();
        OperationFuture<com.google.spanner.admin.database.v1.Database, RestoreDatabaseMetadata> op =
            dbAdminClient.restoreDatabaseAsync(request);
        // Wait until the database has been restored.
        com.google.spanner.admin.database.v1.Database db = op.get();
        // Get the restore info.
        RestoreInfo restoreInfo = db.getRestoreInfo();
        BackupInfo backupInfo = restoreInfo.getBackupInfo();

        System.out.println(
            "Restored database ["
                + db.getName()
                + "] from ["
                + restoreInfo.getBackupInfo().getBackup()
                + "] with version time [" + backupInfo.getVersionTime() + "]");
      } catch (ExecutionException e) {
        throw SpannerExceptionFactory.newSpannerException(e.getCause());
      } catch (InterruptedException e) {
        throw SpannerExceptionFactory.propagateInterrupt(e);
      }
    }
    ```

    > **Note:** The old client library interface code samples for Java are archived in [GitHub](https://github.com/googleapis/google-cloud-java/tree/main/java-spanner/samples/snippets/src/main/java/com/example/spanner/admin/archived) .

    ### Node.js

    ```javascript
    // Imports the Google Cloud client library and precise date library
    const {Spanner} = require('@google-cloud/spanner');
    const {PreciseDate} = require('@google-cloud/precise-date');

    /**
     * TODO(developer): Uncomment the following lines before running the sample.
     */
    // const projectId = 'my-project-id';
    // const instanceId = 'my-instance';
    // const databaseId = 'my-database';
    // const backupId = 'my-backup';

    // Creates a client
    const spanner = new Spanner({
      projectId: projectId,
    });

    // Gets a reference to a Cloud Spanner Database Admin Client object
    const databaseAdminClient = spanner.getDatabaseAdminClient();

    // Restore the database
    console.log(
      `Restoring database ${databaseAdminClient.databasePath(
        projectId,
        instanceId,
        databaseId,
      )} from backup ${backupId}.`,
    );
    const [restoreOperation] = await databaseAdminClient.restoreDatabase({
      parent: databaseAdminClient.instancePath(projectId, instanceId),
      databaseId: databaseId,
      backup: databaseAdminClient.backupPath(projectId, instanceId, backupId),
    });

    // Wait for restore to complete
    console.log('Waiting for database restore to complete...');
    await restoreOperation.promise();

    console.log('Database restored from backup.');
    const [metadata] = await databaseAdminClient.getDatabase({
      name: databaseAdminClient.databasePath(projectId, instanceId, databaseId),
    });
    console.log(
      `Database ${metadata.restoreInfo.backupInfo.sourceDatabase} was restored ` +
        `to ${databaseId} from backup ${metadata.restoreInfo.backupInfo.backup} ` +
        'with version time ' +
        `${new PreciseDate(
          metadata.restoreInfo.backupInfo.versionTime,
        ).toISOString()}.`,
    );
    ```

    > **Note:** The old client library interface code samples for Node.js are archived in [GitHub](https://github.com/googleapis/nodejs-spanner/tree/main/samples/archived) .

    ### PHP

    ```php
    use Google\Cloud\Spanner\Admin\Database\V1\Client\DatabaseAdminClient;
    use Google\Cloud\Spanner\Admin\Database\V1\RestoreDatabaseRequest;

    /**
     * Restore a database from a backup.
     * Example:
     * ```
     * restore_backup($projectId, $instanceId, $databaseId, $backupId);
     * ```
     * @param string $projectId The Google Cloud project ID.
     * @param string $instanceId The Spanner instance ID.
     * @param string $databaseId The Spanner database ID.
     * @param string $backupId The Spanner backup ID.
     */
    function restore_backup(
        string $projectId,
        string $instanceId,
        string $databaseId,
        string $backupId
    ): void {
        $databaseAdminClient = new DatabaseAdminClient();

        $backupName = DatabaseAdminClient::backupName($projectId, $instanceId, $backupId);
        $instanceName = DatabaseAdminClient::instanceName($projectId, $instanceId);

        $request = new RestoreDatabaseRequest([
            'parent' => $instanceName,
            'database_id' => $databaseId,
            'backup' => $backupName
        ]);

        $operationResponse = $databaseAdminClient->restoreDatabase($request);
        $operationResponse->pollUntilComplete();

        $database = $operationResponse->operationSucceeded() ? $operationResponse->getResult() : null;
        $restoreInfo = $database->getRestoreInfo();
        $backupInfo = $restoreInfo->getBackupInfo();
        $sourceDatabase = $backupInfo->getSourceDatabase();
        $sourceBackup = $backupInfo->getBackup();
        $versionTime = $backupInfo->getVersionTime()->getSeconds();
        printf(
            'Database %s restored from backup %s with version time %s' . PHP_EOL,
            $sourceDatabase, $sourceBackup, $versionTime
        );
    }
    ```

    > **Note:** The old client library interface code samples for PHP are archived in [GitHub](https://github.com/GoogleCloudPlatform/php-docs-samples/tree/main/spanner/src/admin/archived) .

    ### Python

    ```python
    def restore_database(instance_id, new_database_id, backup_id):
        """Restores a database from a backup."""
        from google.cloud.spanner_admin_database_v1 import RestoreDatabaseRequest

        spanner_client = spanner.Client()
        database_admin_api = spanner_client.database_admin_api

        # Start restoring an existing backup to a new database.
        request = RestoreDatabaseRequest(
            parent=database_admin_api.instance_path(spanner_client.project, instance_id),
            database_id=new_database_id,
            backup=database_admin_api.backup_path(
                spanner_client.project, instance_id, backup_id
            ),
        )
        operation = database_admin_api.restore_database(request)

        # Wait for restore operation to complete.
        db = operation.result(1600)

        # Newly created database has restore information.
        restore_info = db.restore_info
        print(
            "Database {} restored to {} from backup {} with version time {}.".format(
                restore_info.backup_info.source_database,
                new_database_id,
                restore_info.backup_info.backup,
                restore_info.backup_info.version_time,
            )
        )
    ```

    > **Note:** The old client library interface code samples for Python are archived in [GitHub](https://github.com/googleapis/python-spanner/tree/main/samples/samples/archived) .

    ### Ruby

    ```ruby
    # project_id  = "Your Google Cloud project ID"
    # instance_id = "Your Spanner instance ID"
    # database_id = "Your Spanner database ID of where to restore"
    # backup_id = "Your Spanner backup ID"

    require "google/cloud/spanner"
    require "google/cloud/spanner/admin/database"

    database_admin_client = Google::Cloud::Spanner::Admin::Database.database_admin

    instance_path = database_admin_client.instance_path project: project_id, instance: instance_id

    db_path = database_admin_client.database_path project: project_id,
                                                  instance: instance_id,
                                                  database: database_id

    backup_path = database_admin_client.backup_path project: project_id,
                                                    instance: instance_id,
                                                    backup: backup_id

    job = database_admin_client.restore_database parent: instance_path,
                                                 database_id: database_id,
                                                 backup: backup_path

    puts "Waiting for restore backup operation to complete"

    job.wait_until_done!

    database = database_admin_client.get_database name: db_path
    restore_info = database.restore_info
    puts "Database #{restore_info.backup_info.source_database} was restored to #{database_id} from backup #{restore_info.backup_info.backup} with version time #{restore_info.backup_info.version_time}"
    ```

### Import and export

1.  Export the database, specifying the `snapshotTime` parameter to the needed recovery timestamp.

    ### Console

    1.  Go to the **Instance details** page in the Cloud console.

    2.  In the **Import/Export** tab, click **Export** .

    3.  Check the **Export database from an earlier point in time** box.

        ![Export data form in the console, showing options for selecting a storage bucket and database.](https://docs.cloud.google.com/static/spanner/docs/images/pitr-export.png)

    For detailed instructions, see [export a database](https://docs.cloud.google.com/spanner/docs/export#exporting_a_database) .

    ### gcloud

    Use the [Spanner to Avro](https://docs.cloud.google.com/dataflow/docs/guides/templates/provided-batch#cloud_spanner_to_gcs_avro) Dataflow template to export the database.

    ```
    gcloud dataflow jobs run JOB_NAME \
       --gcs-location='gs://cloud-spanner-point-in-time-recovery/Import Export Template/export/templates/Cloud_Spanner_to_GCS_Avro'
       --region=DATAFLOW_REGION \
       --parameters='instanceId=test-instance,databaseId=example-db,outputDir=YOUR_GCS_DIRECTORY,snapshotTime=2020-09-01T23:59:40.125245Z'
    ```

    Usage notes:

- You can track the progress of your import and export jobs in the [Dataflow Console](https://console.cloud.google.com/dataflow/jobs) .
- Spanner guarantees that the exported data is externally and transactionally consistent at the specified timestamp.
- Specify the timestamp in [RFC 3339 format](https://developers.google.com/protocol-buffers/docs/reference/google.protobuf#google.protobuf.Timestamp) . For example, 2020-09-01T23:59:30.234233Z.
- Make sure that the timestamp you specify is more recent than the database's `earliest_version_time` . If data no longer exists at the specified timestamp, you get an error.

1.  Import to a new database.

    ### Console

    1.  Go to the **Instance details** page in the Cloud console.

    2.  In the **Import/Export** tab, click **Import** .

    For detailed instructions, see [Importing Spanner Avro Files](https://docs.cloud.google.com/spanner/docs/import) .

    ### gcloud

    Use the [Cloud Storage Avro to Spanner](https://docs.cloud.google.com/dataflow/docs/guides/templates/provided-batch#gcs_avro_to_cloud_spanner) Dataflow template to import the Avro files.

    ```
    gcloud dataflow jobs run JOB_NAME \
       --gcs-location='gs://cloud-spanner-point-in-time-recovery/Import Export Template/import/templates/GCS_Avro_to_Cloud_Spanner' \
       --region=DATAFLOW_REGION \
       --staging-location=YOUR_GCS_STAGING_LOCATION \
       --parameters='instanceId=test-instance,databaseId=example-db,inputDir=YOUR_GCS_DIRECTORY'
    ```

## Estimate the storage increase

Before increasing a database's version retention period, you can estimate the expected increase in database storage utilization by totaling the transaction bytes for the needed period of time. For example the following query calculates the number of GiB written in the past 7 days (168h) by reading from the [transaction statistics](https://docs.cloud.google.com/spanner/docs/introspection/transaction-statistics) tables.

### GoogleSQL

```
SELECT
  SUM(bytes_per_hour) / (1024 * 1024 * 1024 ) as GiB
FROM (
  SELECT
    ((commit_attempt_count - commit_failed_precondition_count - commit_abort_count) * avg_bytes)
    AS bytes_per_hour, interval_end
  FROM
    spanner_sys.txn_stats_total_hour
  ORDER BY
    interval_end DESC
  LIMIT
    168);
```

### PostgreSQL

```
SELECT
  bph /  (1024 * 1024 * 1024 ) as GiB
FROM (
  SELECT
    SUM(bytes_per_hour) as bph
  FROM (
    SELECT
      ((commit_attempt_count - commit_failed_precondition_count - commit_abort_count) * avg_bytes)
      AS bytes_per_hour, interval_end
    FROM
      spanner_sys.txn_stats_total_hour
    ORDER BY
      interval_end DESC
    LIMIT
      168)
sub1) sub2;
```

Note that the query provides a rough estimate and can be inaccurate for a few reasons:

- The query doesn't account for the [timestamp](https://docs.cloud.google.com/spanner/docs/true-time-external-consistency#timestamps_and_multi-version_concurrency_control_mvcc) that must be stored for each version of old data. If your database consists of many small [data types](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-types#allowable_types) , the query may underestimate the storage increase.
- The query includes all write operations, but only update operations create previous versions of data. If your workload includes a lot of insert operations, the query may overestimate the storage increase.
