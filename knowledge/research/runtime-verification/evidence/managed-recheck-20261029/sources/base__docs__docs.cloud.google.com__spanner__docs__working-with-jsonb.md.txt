---
name: documents/docs.cloud.google.com/spanner/docs/working-with-jsonb
uri: https://docs.cloud.google.com/spanner/docs/working-with-jsonb
title: Work with JSONB data
description: Learn how to work with the JSONB data type in Spanner for PostgreSQL, including creating tables, modifying data, and using indexes.
data_source: docs.cloud.google.com
---

This page describes how to work with the `JSONB` data type when using Spanner.

`JSONB` is a PostgreSQL data type used for holding semi-structured data in the Spanner PostgreSQL dialect. `JSONB` holds data in JavaScript Object Notation (JSON) format, which follows the specification described in [RFC 7159](https://datatracker.ietf.org/doc/rfc7159/) .

## Specifications

The Spanner `JSONB` data type stores a normalized representation of the input document. This implies the following:

- Quotation marks and whitespace characters are not preserved.

- Comments are not supported. Transactions or queries with comments fail.

- Object keys are sorted first by key length and then lexicographically by the equivalent object key length. If there are duplicate object keys, only the last one is preserved.

- Primitive types ( `string` , `boolean` , `number` , and `null` ) have their type and value preserved.
  - `string` type values are preserved exactly.
  - Trailing zeros are preserved. The output format for `number` type values does not use scientific notation.

- `JSONB` `null` values are treated as SQL non- `NULL` . For example:

  ```
  SELECT null::jsonb IS NULL;  -- Returns true
  SELECT 'null'::jsonb IS NULL; -- Returns false

  SELECT '{"a":null}'::jsonb -> 'a' IS NULL; -- Returns false
  SELECT '{"a":null}'::jsonb -> 'b' IS NULL; -- Returns true

  SELECT '{"a":null}'::jsonb -> 'a'; -- Returns a JSONB 'null'
  SELECT '{"a":null}'::jsonb -> 'b'; -- Returns a SQL NULL
  ```

- JSONB array element order is preserved.

### Restrictions

The following restrictions apply with Spanner `JSONB` :

- Arguments to the `to_jsonb` function can be only from the PostgreSQL data types that Spanner supports.
- Number type values can have 4,932 digits before the decimal point and 16,383 digits after the decimal point.
- The maximum permitted size of the normalized storage format is 10 MB.
- `JSONB` documents must be encoded in UTF-8. Transactions or queries with `JSONB` documents encoded in other formats return an error.

## Create a table with JSONB columns

You can add a `JSONB` column to a table when you create the table.

```
CREATE TABLE Venues (
 VenueId   BIGINT PRIMARY KEY,
 VenueName  VARCHAR(1024),
 VenueAddress VARCHAR(1024),
 VenueFeatures JSONB,
 DateOpened  TIMESTAMPTZ
);
```

A sample `VenueFeatures` `JSONB` object follows:

```
{
    "rating": 4.5,
    "capacity":"1500",
    "construction":"brick",
    "tags": [
        "multi-cuisine",
        "open-seating",
        "stage",
        "public address system"
    ]
}
```

## Add and remove JSONB columns from existing tables

You can add a `JSONB` column and drop it by using `ALTER` statements as follows:

```
ALTER TABLE Venues ADD COLUMN VenueDetails JSONB;
ALTER TABLE Venues DROP COLUMN VenueDetails;
```

The following sample shows how to add a `JSONB` column called `VenueDetails` to the `Venues` table using Spanner client libraries.

### C++

```cpp
void JsonbAddColumn(google::cloud::spanner_admin::DatabaseAdminClient client,
                    google::cloud::spanner::Database const& database) {
  std::vector<std::string> statements = {
      R"""(
        ALTER TABLE Venues
            ADD COLUMN VenueDetails JSONB
      )""",
  };
  auto metadata =
      client.UpdateDatabaseDdl(database.FullName(), statements).get();
  if (!metadata) throw std::move(metadata).status();
  std::cout << "Added JSONB column to table Venues in database "
            << database.FullName() << "\nNew DDL:\n"
            << metadata->DebugString();
}
```

### C#

```csharp
using Google.Cloud.Spanner.Data;
using System;
using System.Threading.Tasks;

public class AddJsonbColumnAsyncPostgresSample
{
    public async Task AddJsonbColumnAsyncPostgres(string projectId, string instanceId, string databaseId)
    {
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";
        string alterStatement = $"ALTER TABLE VenueDetails ADD COLUMN Details JSONB";

        using var connection = new SpannerConnection(connectionString);
        using var ddlCmd = connection.CreateDdlCommand(alterStatement);
        await ddlCmd.ExecuteNonQueryAsync();
        Console.WriteLine($"Added the JSONB column named Details to VenueDetails table.");
    }
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

// addJsonBColumn creates a column in the database of type JSONB
func addJsonBColumn(w io.Writer, db string) error {
    // db = `projects/<project>/instances/<instance-id>/database/<database-id>`
    matches := regexp.MustCompile("^(.*)/databases/(.*)$").FindStringSubmatch(db)
    if matches == nil || len(matches) != 3 {
        return fmt.Errorf("addJsonbColumn: invalid database id %s", db)
    }

    ctx := context.Background()
    adminClient, err := database.NewDatabaseAdminClient(ctx)
    if err != nil {
        return err
    }
    defer adminClient.Close()

    op, err := adminClient.UpdateDatabaseDdl(ctx, &adminpb.UpdateDatabaseDdlRequest{
        Database: db,
        Statements: []string{
            "ALTER TABLE Venues ADD COLUMN VenueDetails JSONB",
        },
    })
    if err != nil {
        return err
    }
    if err := op.Wait(ctx); err != nil {
        return err
    }
    fmt.Fprintf(w, "Added VenueDetails column\n")
    return nil
}
```

### Java

```java
import com.google.cloud.spanner.Spanner;
import com.google.cloud.spanner.SpannerOptions;
import com.google.cloud.spanner.admin.database.v1.DatabaseAdminClient;
import com.google.common.collect.ImmutableList;
import com.google.spanner.admin.database.v1.DatabaseName;
import java.util.concurrent.ExecutionException;

class AddJsonbColumnSample {

  static void addJsonbColumn() throws InterruptedException, ExecutionException {
    // TODO(developer): Replace these variables before running the sample.
    String projectId = "my-project";
    String instanceId = "my-instance";
    String databaseId = "my-database";

    addJsonbColumn(projectId, instanceId, databaseId);
  }

  static void addJsonbColumn(String projectId, String instanceId, String databaseId)
      throws InterruptedException, ExecutionException {
    try (Spanner spanner =
        SpannerOptions.newBuilder()
            .setProjectId(projectId)
            .build()
            .getService();
        DatabaseAdminClient databaseAdminClient = spanner.createDatabaseAdminClient()) {
      // JSONB datatype is only supported with PostgreSQL-dialect databases.
      // Wait for the operation to finish.
      // This will throw an ExecutionException if the operation fails.
      databaseAdminClient.updateDatabaseDdlAsync(
          DatabaseName.of(projectId, instanceId, databaseId),
          ImmutableList.of("ALTER TABLE Venues ADD COLUMN VenueDetails JSONB")).get();
      System.out.printf("Successfully added column `VenueDetails`%n");
    }
  }
}
```

> **Note:** The old client library interface code samples for Java are archived in [GitHub](https://github.com/googleapis/google-cloud-java/tree/main/java-spanner/samples/snippets/src/main/java/com/example/spanner/admin/archived) .

### Node.js

```javascript
/**
 * TODO(developer): Uncomment these variables before running the sample.
 */
// const instanceId = 'my-instance';
// const databaseId = 'my-database';
// const projectId = 'my-project-id';

// Imports the Google Cloud Spanner client library
const {Spanner} = require('@google-cloud/spanner');

// Instantiates a client
const spanner = new Spanner({
  projectId: projectId,
});

async function pgJsonbAddColumn() {
  // Gets a reference to a Cloud Spanner Database Admin Client object
  const databaseAdminClient = spanner.getDatabaseAdminClient();

  const request = ['ALTER TABLE Venues ADD COLUMN VenueDetails JSONB'];

  // Updates schema by adding a new table.
  const [operation] = await databaseAdminClient.updateDatabaseDdl({
    database: databaseAdminClient.databasePath(
      projectId,
      instanceId,
      databaseId,
    ),
    statements: request,
  });
  console.log(`Waiting for operation on ${databaseId} to complete...`);
  await operation.promise();
  console.log(
    `Added jsonb column to table venues to database ${databaseId}.`,
  );
}
pgJsonbAddColumn();
```

> **Note:** The old client library interface code samples for Node.js are archived in [GitHub](https://github.com/googleapis/nodejs-spanner/tree/main/samples/archived) .

### PHP

```php
use Google\Cloud\Spanner\Admin\Database\V1\Client\DatabaseAdminClient;
use Google\Cloud\Spanner\Admin\Database\V1\UpdateDatabaseDdlRequest;

/**
 * Add a JSONB column to a table present in a PG Spanner database.
 *
 * @param string $projectId The Google Cloud project ID.
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 * @param string $tableName The table in which the column needs to be added.
 */
function pg_add_jsonb_column(
    string $projectId,
    string $instanceId,
    string $databaseId,
    string $tableName = 'Venues'
): void {
    $databaseAdminClient = new DatabaseAdminClient();
    $databaseName = DatabaseAdminClient::databaseName($projectId, $instanceId, $databaseId);
    $statement = sprintf('ALTER TABLE %s ADD COLUMN VenueDetails JSONB', $tableName);
    $request = new UpdateDatabaseDdlRequest([
        'database' => $databaseName,
        'statements' => [$statement]
    ]);

    $operation = $databaseAdminClient->updateDatabaseDdl($request);

    print('Waiting for operation to complete...' . PHP_EOL);
    $operation->pollUntilComplete();

    print(sprintf('Added column VenueDetails on table %s.', $tableName) . PHP_EOL);
}
```

> **Note:** The old client library interface code samples for PHP are archived in [GitHub](https://github.com/GoogleCloudPlatform/php-docs-samples/tree/main/spanner/src/admin/archived) .

### Python

```python
def add_jsonb_column(instance_id, database_id):
    """
    Alters Venues tables in the database adding a JSONB column.
    You can create the table by running the `create_table_with_datatypes`
    sample or by running this DDL statement against your database:
    CREATE TABLE Venues (
      VenueId         BIGINT NOT NULL,
      VenueName       character varying(100),
      VenueInfo       BYTEA,
      Capacity        BIGINT,
      OutdoorVenue    BOOL,
      PopularityScore FLOAT8,
      Revenue         NUMERIC,
      LastUpdateTime  SPANNER.COMMIT_TIMESTAMP NOT NULL,
      PRIMARY KEY (VenueId))
    """
    # instance_id = "your-spanner-instance"
    # database_id = "your-spanner-db-id"

    from google.cloud.spanner_admin_database_v1.types import spanner_database_admin

    spanner_client = spanner.Client()
    database_admin_api = spanner_client.database_admin_api

    request = spanner_database_admin.UpdateDatabaseDdlRequest(
        database=database_admin_api.database_path(
            spanner_client.project, instance_id, database_id
        ),
        statements=["ALTER TABLE Venues ADD COLUMN VenueDetails JSONB"],
    )

    operation = database_admin_api.update_database_ddl(request)

    print("Waiting for operation to complete...")
    operation.result(OPERATION_TIMEOUT_SECONDS)

    print(
        'Altered table "Venues" on database {} on instance {}.'.format(
            database_id, instance_id
        )
    )
```

> **Note:** The old client library interface code samples for Python are archived in [GitHub](https://github.com/googleapis/python-spanner/tree/main/samples/samples/archived) .

### Ruby

```ruby
require "google/cloud/spanner"

def spanner_postgresql_jsonb_add_column project_id:, instance_id:, database_id:
  # project_id  = "Your Google Cloud project ID"
  # instance_id = "Your Spanner instance ID"
  # database_id = "Your Spanner database ID"


  # Show how to add JSONB column
  db_admin_client = Google::Cloud::Spanner::Admin::Database.database_admin project: project_id

  db_path = db_admin_client.database_path project: project_id,
                                          instance: instance_id,
                                          database: database_id

  add_column_query = "ALTER TABLE Venues ADD COLUMN VenueDetails JSONB"

  job = db_admin_client.update_database_ddl database: db_path,
                                            statements: [add_column_query]

  job.wait_until_done!

  if job.error?
    puts "Error while adding column. Code: #{job.error.code}. Message: #{job.error.message}"
    raise GRPC::BadStatus.new(job.error.code, job.error.message)
  end

  puts "Added Venues column to VenueDetails table in database #{database_id}"
end
```

## Modify JSONB data

You can modify a `JSONB` column just like any other column.

An example follows:

```
UPDATE Venues SET VenueFeatures = '{"rating": 4.5, "tags":["multi-cuisine", "open-seating"] }'
  WHERE VenueId = 1;
```

The following sample shows how to update `JSONB` data using Spanner client libraries.

### C++

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```cpp
void JsonbUpdateData(google::cloud::spanner::Client client) {
  auto venue19_details = google::cloud::spanner::JsonB(R"""(
        {"rating": 9, "open": true}
      )""");
  // PG.JSONB takes the last value in the case of duplicate keys.
  auto venue4_details = google::cloud::spanner::JsonB(R"""(
        [
          {"name": null, "available": true},
          {"name": "room 2", "available": false, "name": "room 3"},
          {
            "main hall": {
              "description": "this is the biggest space",
              "size": 200
            }
          }
        ]
      )""");
  auto venue42_details = google::cloud::spanner::JsonB(R"""(
        {
          "name": null,
          "open": {"Monday": true, "Tuesday": false},
          "tags": ["large", "airy"]
        }
      )""");
  auto update_venues = google::cloud::spanner::InsertOrUpdateMutationBuilder(
                           "Venues", {"VenueId", "VenueDetails"})
                           .EmplaceRow(19, venue19_details)
                           .EmplaceRow(4, venue4_details)
                           .EmplaceRow(42, venue42_details)
                           .Build();
  auto commit_result =
      client.Commit(google::cloud::spanner::Mutations{update_venues});
  if (!commit_result) throw std::move(commit_result).status();
  std::cout << "Updated data.\n";
}
```

### C#

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```csharp
using Google.Cloud.Spanner.Data;
using Newtonsoft.Json;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading.Tasks;

public class UpdateDataWithJsonbAsyncPostgresSample
{
    public async Task UpdateDataWithJsonbAsyncPostgres(string projectId, string instanceId, string databaseId)
    {
        List<VenueInformation> venueInformationList = new List<VenueInformation>
        {
            // If you are using .NET Core 3.1 or later, you can use System.Text.Json for serialization instead.
            new VenueInformation
            {
                VenueId = 19,
                Details = JsonConvert.SerializeObject(new
                {
                    rating = 9,
                    open = true,
                })
            },
            new VenueInformation
            {
                VenueId = 4,
                // In the case of repeated field names in the JSON, PostgreSQL JSONB will keep the value of the last field appearance.
                // For instance, in the following example, the value for name will be room 3.
                Details = @"
                {
                    ""name"": ""room 2"",
                    ""available"": false,
                    ""name"": ""room 3""
                }"
            },
            new VenueInformation
            {
                VenueId = 42,
                Details = JsonConvert.SerializeObject(new
                {
                    name = "Central Park",
                    open = new
                    {
                        Monday = true,
                        Tuesday = false,
                    },
                    tags = new string[] {"large", "airy" },
                }),
            },
        };
        // Create connection to Cloud Spanner.
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";
        using var connection = new SpannerConnection(connectionString);
        await connection.OpenAsync();

        await Task.WhenAll(venueInformationList.Select(venue =>
        {
            // Update rows in the Venues table.
            using var cmd = connection.CreateUpdateCommand("VenueInformation", new SpannerParameterCollection
            {
                { "VenueId", SpannerDbType.Int64, venue.VenueId },
                { "Details", SpannerDbType.PgJsonb, venue.Details }
            });
            return cmd.ExecuteNonQueryAsync();
        }));
        Console.WriteLine("Data updated.");
    }

    public struct VenueInformation
    {
        public int VenueId { get; set; }
        public string Details { get; set; }
    }
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

    "cloud.google.com/go/spanner"
)

// updateDataWithJsonBColumn updates database with JsonB type values
func updateDataWithJsonBColumn(w io.Writer, db string) error {
    // db = `projects/<project>/instances/<instance-id>/database/<database-id>`
    matches := regexp.MustCompile("^(.*)/databases/(.*)$").FindStringSubmatch(db)
    if matches == nil || len(matches) != 3 {
        return fmt.Errorf("updateDataWithJsonBColumn: invalid database id %s", db)
    }

    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    type VenueDetails struct {
        Name   spanner.NullString   `json:"name"`
        Rating spanner.NullFloat64  `json:"rating"`
        Open   interface{}          `json:"open"`
        Tags   []spanner.NullString `json:"tags"`
    }

    details_1 := spanner.PGJsonB{Value: []VenueDetails{
        {Name: spanner.NullString{StringVal: "room1", Valid: true}, Open: true},
        {Name: spanner.NullString{StringVal: "room2", Valid: true}, Open: false},
    }, Valid: true}
    details_2 := spanner.PGJsonB{Value: VenueDetails{
        Rating: spanner.NullFloat64{Float64: 9, Valid: true},
        Open:   true,
    }, Valid: true}

    details_3 := spanner.PGJsonB{Value: VenueDetails{
        Name: spanner.NullString{Valid: false},
        Open: map[string]bool{"monday": true, "tuesday": false},
        Tags: []spanner.NullString{{StringVal: "large", Valid: true}, {StringVal: "airy", Valid: true}},
    }, Valid: true}

    cols := []string{"VenueId", "VenueDetails"}
    _, err = client.Apply(ctx, []*spanner.Mutation{
        spanner.Update("Venues", cols, []interface{}{4, details_1}),
        spanner.Update("Venues", cols, []interface{}{19, details_2}),
        spanner.Update("Venues", cols, []interface{}{42, details_3}),
    })

    if err != nil {
        return err
    }
    fmt.Fprintf(w, "Updated data to VenueDetails column\n")

    return nil
}
```

### Java

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```java
import com.google.cloud.spanner.DatabaseClient;
import com.google.cloud.spanner.DatabaseId;
import com.google.cloud.spanner.Mutation;
import com.google.cloud.spanner.Spanner;
import com.google.cloud.spanner.SpannerOptions;
import com.google.cloud.spanner.Value;
import com.google.common.collect.ImmutableList;

class UpdateJsonbDataSample {

  static void updateJsonbData() {
    // TODO(developer): Replace these variables before running the sample.
    String projectId = "my-project";
    String instanceId = "my-instance";
    String databaseId = "my-database";

    try (Spanner spanner =
        SpannerOptions.newBuilder().setProjectId(projectId).build().getService()) {
      DatabaseClient client =
          spanner.getDatabaseClient(DatabaseId.of(projectId, instanceId, databaseId));
      updateJsonbData(client);
    }
  }

  static void updateJsonbData(DatabaseClient client) {
    // PG JSONB takes the last value in the case of duplicate keys.
    // PG JSONB sorts first by key length and then lexicographically with
    // equivalent key length.
    client.write(
        ImmutableList.of(
            Mutation.newInsertOrUpdateBuilder("Venues")
                .set("VenueId")
                .to(4L)
                .set("VenueDetails")
                .to(
                    Value.pgJsonb(
                        "[{\"name\":\"room 1\",\"open\":true,\"name\":\"room 3\"},"
                            + "{\"name\":\"room 2\",\"open\":false}]"))
                .build(),
            Mutation.newInsertOrUpdateBuilder("Venues")
                .set("VenueId")
                .to(19L)
                .set("VenueDetails")
                .to(Value.pgJsonb("{\"rating\":9,\"open\":true}"))
                .build(),
            Mutation.newInsertOrUpdateBuilder("Venues")
                .set("VenueId")
                .to(42L)
                .set("VenueDetails")
                .to(
                    Value.pgJsonb(
                        "{\"name\":null,"
                            + "\"open\":{\"Monday\":true,\"Tuesday\":false},"
                            + "\"tags\":[\"large\",\"airy\"]}"))
                .build()));
    System.out.println("Venues successfully updated");
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

// Imports the Google Cloud Spanner client library
const {Spanner} = require('@google-cloud/spanner');

// Instantiates a client
const spanner = new Spanner({
  projectId: projectId,
});

async function pgJsonbUpdateData() {
  // Gets a reference to a Cloud Spanner instance and database.
  const instance = spanner.instance(instanceId);
  const database = instance.database(databaseId);
  // Instantiate Spanner table objects.
  const venuesTable = database.table('venues');

  const data = [
    {
      VenueId: '19',
      VenueDetails: {rating: 9, open: true},
    },
    {
      VenueId: '4',
      // PG JSONB sorts first by key length and then lexicographically with equivalent key length
      // and takes the last value in the case of duplicate keys
      VenueDetails: `[
      {
        "name": null,
        "available": true
      },
      {
        "name": "room 2",
        "available": false,
        "name": "room 3"
      },
      {
        "main hall": {
          "description": "this is the biggest space",
          "size": 200
        }
      }
    ]`,
    },
    {
      VenueId: '42',
      VenueDetails: {
        name: null,
        open: {
          Monday: true,
          Tuesday: false,
        },
        tags: ['large', 'airy'],
      },
    },
  ];

  try {
    await venuesTable.update(data);
    console.log('Updated data.');
  } catch (err) {
    console.error('ERROR:', err);
  } finally {
    // Close the database when finished.
    await database.close();
  }
}
pgJsonbUpdateData();
```

### PHP

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```php
use Google\Cloud\Spanner\SpannerClient;

/**
 * Insert/update data in a JSONB column in a Postgres table.
 *
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 * @param string $tableName The table in which the data needs to be updated.
 */
function pg_jsonb_update_data(
    string $instanceId,
    string $databaseId,
    string $tableName = 'Venues'
): void {
    $spanner = new SpannerClient();
    $instance = $spanner->instance($instanceId);
    $database = $instance->database($databaseId);

    $database->insertOrUpdateBatch($tableName, [
        [
            'VenueId' => 1,
            'VenueDetails' => '{"rating": 9, "open": true}'
        ],
        [
            'VenueId' => 4,
            'VenueDetails' => '[
                {
                    "name": null,
                    "available": true
                },' .
                // PG JSONB sorts first by key length and then lexicographically with
                // equivalent key length and takes the last value in the case of duplicate keys
                '{
                    "name": "room 2",
                    "available": false,
                    "name": "room 3"
                },
                {
                    "main hall": {
                        "description": "this is the biggest space",
                        "size": 200
                    }
                }
            ]'
        ],
        [
            'VenueId' => 42,
            'VenueDetails' => $spanner->pgJsonb([
                'name' => null,
                'open' => [
                    'Monday' => true,
                    'Tuesday' => false
                ],
                'tags' => ['large', 'airy'],
            ])
        ]
    ]);

    print(sprintf('Inserted/updated 3 rows in table %s', $tableName) . PHP_EOL);
}
```

### Python

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```python
def update_data_with_jsonb(instance_id, database_id):
    """Updates Venues tables in the database with the JSONB
    column.
    This updates the `VenueDetails` column which must be created before
    running this sample. You can add the column by running the
    `add_jsonb_column` sample or by running this DDL statement
     against your database:
        ALTER TABLE Venues ADD COLUMN VenueDetails JSONB
    """
    # instance_id = "your-spanner-instance"
    # database_id = "your-spanner-db-id"

    spanner_client = spanner.Client()
    instance = spanner_client.instance(instance_id)
    database = instance.database(database_id)

    """
    PG JSONB takes the last value in the case of duplicate keys.
    PG JSONB sorts first by key length and then lexicographically with
    equivalent key length.
    """

    with database.batch() as batch:
        batch.update(
            table="Venues",
            columns=("VenueId", "VenueDetails"),
            values=[
                (
                    4,
                    JsonObject(
                        [
                            JsonObject({"name": None, "open": True}),
                            JsonObject({"name": "room 2", "open": False}),
                        ]
                    ),
                ),
                (19, JsonObject(rating=9, open=True)),
                (
                    42,
                    JsonObject(
                        {
                            "name": None,
                            "open": {"Monday": True, "Tuesday": False},
                            "tags": ["large", "airy"],
                        }
                    ),
                ),
            ],
        )

    print("Updated data.")
```

### Ruby

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```ruby
require "google/cloud/spanner"

def spanner_postgresql_jsonb_update_data project_id:, instance_id:, database_id:
  # project_id  = "Your Google Cloud project ID"
  # instance_id = "Your Spanner instance ID"
  # database_id = "Your Spanner database ID"

  # Insert JSONB data into table
  spanner = Google::Cloud::Spanner.new project: project_id
  client  = spanner.client instance_id, database_id

  data = [
    {
      VenueId: "19",
      VenueDetails: { rating: 9, open: true }
    },
    {
      VenueId: "4",
      VenueDetails: [
        {
          name: null,
          open: true
        },
        {
          name: "room 2",
          open: false
        },
        {
          main_hall: {
            description: "this is the biggest space",
            size: 200
          }
        }
      ]
    },
    {
      VenueId: "42",
      VenueDetails: {
        name: null,
        open: {
          Monday: true,
          Tuesday: false
        },
        tags: ["large", "airy"]
      }
    }
  ]

  client.upsert "Venues", data
  puts "Inserted data into Venues table"
end
```

## Index JSON data

You can accelerate querying JSONB data by using [secondary indexes](https://docs.cloud.google.com/spanner/docs/secondary-indexes) and [search indexes](https://docs.cloud.google.com/spanner/docs/full-text-search/search-indexes) with your JSONB data. Spanner doesn't support using JSONB type columns as keys in secondary indexes.

### Use secondary index

Secondary indexes are useful when filtering against scalar values within a JSONB document. To use secondary indexes with JSONB, create a [generated column](https://docs.cloud.google.com/spanner/docs/generated-column/how-to) that extracts the relevant scalar data and casts it to an appropriate SQL data type. You can then create a secondary index over this generated column. The index accelerates eligible queries that run against the generated column.

In the following example, you create a `VenuesByCapacity` index that the database uses to find the venues with capacities greater than 1000. Instead of checking every row, Spanner uses the index to locate the relevant rows, which improves query performance, especially for large tables.

```
ALTER TABLE Venues (
ADD COLUMN VenueCapacity BIGINT GENERATED ALWAYS AS ((VenueFeatures->>'capacity')::BIGINT) VIRTUAL,
DateOpened TIMESTAMPTZ
);

CREATE INDEX VenuesByCapacity ON Venues(VenueCapacity);

SELECT VenueName
FROM Venues
WHERE VenueCapacity > 1000;
```

### Use search indexes

> **Note:** This feature is available with the Spanner Enterprise edition and Enterprise Plus edition. For more information, see the [Spanner editions overview](https://docs.cloud.google.com/spanner/docs/editions-overview) .

Search indexes are useful when you query against JSONB documents that are dynamic or varied. Unlike secondary indexes, you can [create search indexes](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#create_search_index) over any JSONB document stored in a JSONB column. The search index automatically adapts to variations across JSON documents, between different rows, and over time.

In the following example, you create a `VenuesByVenueDetails` search index that the database uses to find the venues with specific details such as size and operating schedule. Instead of checking every row, Spanner uses the index to locate the relevant rows, which improves query performance, especially for large tables.

```
ALTER TABLE Venues
ADD COLUMN VenueDetails_Tokens spanner.tokenlist
  GENERATED ALWAYS AS (spanner.tokenize_jsonb(VenueDetails)) VIRTUAL HIDDEN;

CREATE SEARCH INDEX VenuesByVenueDetails
ON Venues (VenueDetails_Tokens);

SELECT VenueName
FROM Venues
WHERE VenueDetails @> '{"labels": ["large"], "open": {"Friday": true}}'::jsonb;

SELECT VenueName
FROM Venues
WHERE VenueDetails->'open'->'Friday' = 'true'::jsonb
  AND VenueDetails->'type' IN ('"stadium"'::jsonb, '"arena"'::jsonb);
```

For more information, see [JSON search indexes](https://docs.cloud.google.com/spanner/docs/full-text-search/json-indexes) .

## Query JSONB data

You can query `JSONB` columns based on the values of the underlying fields. The following example extracts `VenueId` and `VenueName` from `Venues` where `VenueFeatures` has a `rating` value greater than `3.5` .

```
SELECT VenueId, VenueName
FROM Venues
WHERE (VenueFeatures->>'rating')::FLOAT8 > 3.5;
```

The following sample shows how to query `JSONB` data using Spanner client libraries.

### C++

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```cpp
void JsonbQueryWithParameter(google::cloud::spanner::Client client) {
  auto sql = google::cloud::spanner::SqlStatement(
      "SELECT VenueId, VenueDetails FROM Venues"
      "  WHERE CAST(VenueDetails ->> 'rating' AS INTEGER) > $1",
      {{"p1", google::cloud::spanner::Value(2)}});
  using RowType =
      std::tuple<std::int64_t, std::optional<google::cloud::spanner::JsonB>>;
  auto rows = client.ExecuteQuery(std::move(sql));
  for (auto& row : google::cloud::spanner::StreamOf<RowType>(rows)) {
    if (!row) throw std::move(row).status();
    std::cout << "VenueId: " << std::get<0>(*row) << ", ";
    std::cout << "Details: " << std::string(std::get<1>(*row).value()) << "\n";
  }
}
```

### C#

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```csharp
using Google.Cloud.Spanner.Data;
using System.Collections.Generic;
using System.Threading.Tasks;

public class QueryJsonbDataUsingParameterAsyncPostgresSample
{
    public async Task<List<VenueInformation>> QueryJsonbDataUsingParameterAsyncPostgres(string projectId, string instanceId, string databaseId)
    {
        string connectionString = $"Data Source=projects/{projectId}/instances/{instanceId}/databases/{databaseId}";
        using var connection = new SpannerConnection(connectionString);
        // Get all the venues with a rating greater than 2.
        /* Details is a column of type JSONB. Some of the data persisted in the Details column has the following structure:
           [{
                 "name": "string",
                 "available": true,
                 "rating": int // This field is optional.
           }] */
        using var command = connection.CreateSelectCommand(
            "SELECT venueid, details FROM VenueInformation WHERE CAST(details ->> 'rating' AS INTEGER) > $1",
            new SpannerParameterCollection
            {
                { "p1", SpannerDbType.Int64, 2 }
            });
        var venues = new List<VenueInformation>();
        using var reader = await command.ExecuteReaderAsync();
        while (await reader.ReadAsync())
        {
            venues.Add(new VenueInformation
            {
                VenueId = reader.GetFieldValue<int>("venueid"),
                Details = reader.GetFieldValue<string>("details")
            });
        }
        return venues;
    }

    public struct VenueInformation
    {
        public int VenueId { get; set; }
        public string Details { get; set; }
    }
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

    "cloud.google.com/go/spanner"
    "google.golang.org/api/iterator"
)

// queryWithJsonBParameter queries data on the JSON type column of the database
func queryWithJsonBParameter(w io.Writer, db string) error {
    // db = `projects/<project>/instances/<instance-id>/database/<database-id>`
    matches := regexp.MustCompile("^(.*)/databases/(.*)$").FindStringSubmatch(db)
    if matches == nil || len(matches) != 3 {
        return fmt.Errorf("queryWithJsonBParameter: invalid database id %s", db)
    }
    ctx := context.Background()
    client, err := spanner.NewClient(ctx, db)
    if err != nil {
        return err
    }
    defer client.Close()

    type VenueDetails struct {
        Name   spanner.NullString   `json:"name"`
        Rating spanner.NullFloat64  `json:"rating"`
        Open   interface{}          `json:"open"`
        Tags   []spanner.NullString `json:"tags"`
    }

    stmt := spanner.Statement{
        SQL: `SELECT VenueId, VenueDetails FROM Venues WHERE CAST(venuedetails ->> 'rating' AS INTEGER) > $1`,
        Params: map[string]interface{}{
            "p1": 2,
        },
    }
    iter := client.Single().Query(ctx, stmt)
    defer iter.Stop()
    for {
        row, err := iter.Next()
        if err == iterator.Done {
            return nil
        }
        if err != nil {
            return err
        }
        var venueID int64
        var venueDetails spanner.PGJsonB
        if err := row.Columns(&venueID, &venueDetails); err != nil {
            return err
        }
        fmt.Fprintf(w, "The venue details for venue id %v is %v\n", venueID, venueDetails)
    }
}
```

### Java

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```java
import com.google.cloud.spanner.DatabaseClient;
import com.google.cloud.spanner.DatabaseId;
import com.google.cloud.spanner.ResultSet;
import com.google.cloud.spanner.Spanner;
import com.google.cloud.spanner.SpannerOptions;
import com.google.cloud.spanner.Statement;
import com.google.cloud.spanner.Value;

class QueryWithJsonbParameterSample {

  static void queryWithJsonbParameter() {
    // TODO(developer): Replace these variables before running the sample.
    String projectId = "my-project";
    String instanceId = "my-instance";
    String databaseId = "my-database";

    try (Spanner spanner =
        SpannerOptions.newBuilder().setProjectId(projectId).build().getService()) {
      DatabaseClient client =
          spanner.getDatabaseClient(DatabaseId.of(projectId, instanceId, databaseId));
      queryWithJsonbParameter(client);
    }
  }

  static void queryWithJsonbParameter(DatabaseClient client) {
    int rating = 2;
    Statement statement =
        Statement.newBuilder(
                "SELECT VenueId, VenueDetails\n"
                    + "FROM Venues\n"
                    + "WHERE CAST(venuedetails ->> 'rating' "
                    + "AS INTEGER) > $1")
            .bind("p1")
            .to(Value.int64(rating))
            .build();
    try (ResultSet resultSet = client.singleUse().executeQuery(statement)) {
      while (resultSet.next()) {
        System.out.printf(
            "VenueId: %s, VenueDetails: %s%n",
            resultSet.getLong("venueid"), resultSet.getPgJsonb("venuedetails"));
      }
    }
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

// Imports the Google Cloud Spanner client library
const {Spanner} = require('@google-cloud/spanner');

// Instantiates a client
const spanner = new Spanner({
  projectId: projectId,
});

async function pgJsonbDataType() {
  // Gets a reference to a Cloud Spanner instance and database.
  const instance = spanner.instance(instanceId);
  const database = instance.database(databaseId);

  const select_query = {
    sql: `SELECT venueid, venuedetails FROM Venues
        WHERE CAST(venuedetails ->> 'rating' AS INTEGER) > $1`,
    params: {
      p1: 2,
    },
    types: {
      p1: 'int64',
    },
    json: true,
  };

  // Queries row from the Venues table.
  try {
    const [rows] = await database.run(select_query);

    rows.forEach(row => {
      console.log(
        `VenueId: ${row.venueid}, Details: ${JSON.stringify(
          row.venuedetails,
        )}`,
      );
    });
  } finally {
    // Close the database when finished.
    await database.close();
  }
}
pgJsonbDataType();
```

### PHP

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```php
use Google\Cloud\Spanner\Database;
use Google\Cloud\Spanner\SpannerClient;

/**
 * Query data to a jsonb column in a PostgreSQL table.
 *
 * @param string $instanceId The Spanner instance ID.
 * @param string $databaseId The Spanner database ID.
 * @param string $tableName The table from which the data needs to be queried.
 */
function pg_jsonb_query_parameter(
    string $instanceId,
    string $databaseId,
    string $tableName = 'Venues'
): void {
    $spanner = new SpannerClient();
    $instance = $spanner->instance($instanceId);
    $database = $instance->database($databaseId);

    $results = $database->execute(
        sprintf('SELECT venueid, venuedetails FROM %s', $tableName) .
        " WHERE CAST(venuedetails ->> 'rating' AS INTEGER) > $1",
        [
        'parameters' => [
            'p1' => 2
        ],
        'types' => [
            'p1' => Database::TYPE_INT64
        ]
    ]);

    foreach ($results as $row) {
        printf('VenueId: %s, VenueDetails: %s' . PHP_EOL, $row['venueid'], $row['venuedetails']);
    }
}
```

### Python

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```python
def query_data_with_jsonb_parameter(instance_id, database_id):
    """Queries sample data using SQL with a JSONB parameter."""
    # instance_id = "your-spanner-instance"
    # database_id = "your-spanner-db-id"

    spanner_client = spanner.Client()
    instance = spanner_client.instance(instance_id)
    database = instance.database(database_id)

    param = {"p1": 2}
    param_type = {"p1": param_types.INT64}

    with database.snapshot() as snapshot:
        results = snapshot.execute_sql(
            "SELECT venueid, venuedetails FROM Venues"
            + " WHERE CAST(venuedetails ->> 'rating' AS INTEGER) > $1",
            params=param,
            param_types=param_type,
        )

        for row in results:
            print("VenueId: {}, VenueDetails: {}".format(*row))
```

### Ruby

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```ruby
require "google/cloud/spanner"

def spanner_postgresql_jsonb_query_parameter project_id:, instance_id:, database_id:
  # project_id  = "Your Google Cloud project ID"
  # instance_id = "Your Spanner instance ID"
  # database_id = "Your Spanner database ID"
  spanner = Google::Cloud::Spanner.new project: project_id
  client  = spanner.client instance_id, database_id

  sql_query = <<~QUERY
    SELECT venueid, venuedetails
    FROM Venues
    WHERE CAST(venuedetails ->> 'rating' AS INTEGER) > $1
  QUERY

  # pass parameterized query's params represented by position
  results = client.execute sql_query, params: { p1: 5 }
  puts results.rows.first

  # Read JSONB value from table
  results = client.read "Venues", [:VenueId, :VenueDetails], keys: 19
  puts results.rows.first
end
```

## Unsupported PostgreSQL JSONB features

The following open source PostgreSQL `JSONB` features aren't supported on Spanner `JSONB` :

- Ordering, comparison, and aggregation

- PrimaryKey and ForeignKey

- Indexing, including the GIN index. You can use a Spanner search index instead, which accelerates the same JSONB operations as a GIN index. For more information, see [Index JSON data](https://docs.cloud.google.com/spanner/docs/working-with-jsonb#index) .

- Altering a `JSONB` column to or from any other data type

- Using parameterized queries with untyped JSONB parameters in tools that use the PostgreSQL wire protocol

- Using coercion in the query engine. Unlike open source PostgreSQL, implicit coercion from `JSONB` to text isn't supported. You must use explicit casting from the `JSONB` type to match function signatures. For example:

  ```
      SELECT concat('abc'::text,  '{"key1":1}'::jsonb);  -- Returns error
      SELECT concat('abc'::text, CAST('{"key1":1}'::jsonb AS TEXT));  -- This works
      SELECT 3 + CAST('5'::jsonb AS INTEGER); -- This works
  ```
