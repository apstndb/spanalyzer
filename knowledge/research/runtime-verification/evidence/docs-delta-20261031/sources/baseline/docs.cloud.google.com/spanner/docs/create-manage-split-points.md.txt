---
name: documents/docs.cloud.google.com/spanner/docs/create-manage-split-points
uri: https://docs.cloud.google.com/spanner/docs/create-manage-split-points
title: Create and manage split points
description: Manage split points in your Spanner database to pre-split your data and prepare for increased traffic.
data_source: docs.cloud.google.com
---

This page describes how you can create and manage split points in your database. You can create split points to pre-split your database to help prepare for an anticipated increase in traffic. For more information about pre-splitting, see [Pre-splitting overview](https://docs.cloud.google.com/spanner/docs/pre-splitting-overview) .

## Before you begin

- To get the permission that you need to create and manage split points, ask your administrator to grant you the [Cloud Spanner Database Admin](https://docs.cloud.google.com/iam/docs/roles-permissions/spanner#spanner.databaseAdmin) ( `roles/spanner.databaseAdmin` ) IAM role on your instance. For more information about granting roles, see [Manage access to projects, folders, and organizations](https://docs.cloud.google.com/iam/docs/granting-changing-revoking-access) .

  This predefined role contains the `spanner.databases.addSplitPoints` permission, which is required to create and manage split points.

  You might also be able to get this permission with [custom roles](https://docs.cloud.google.com/iam/docs/creating-custom-roles) or other [predefined roles](https://docs.cloud.google.com/iam/docs/roles-overview#predefined) .

- The gcloud CLI examples on this page make the following assumptions:

  - You've set up [gcloud CLI](https://docs.cloud.google.com/sdk/docs) for use with Spanner. If you're new to using gcloud CLI with Spanner, see [Create and query a database using the gcloud CLI](https://docs.cloud.google.com/spanner/docs/getting-started/gcloud) .

  - You have configured gcloud CLI with your project. For example:

    ```
    gcloud config set core/project PROJECT_ID
    ```

- Ensure your Spanner instance has enough nodes to support the anticipated increase in traffic. For more information, see [Pre-splitting overview](https://docs.cloud.google.com/spanner/docs/pre-splitting-overview#example-use-case) .

## Create split points

You can create split points using the Google Cloud CLI or the REST APIs.

### gcloud

If you want to create split points using the Google Cloud CLI, you need to create a file that contains all the splits and supply the path in the gcloud CLI command. The file must not exceed the API limit of 100 points per API request. For more information, see [Quotas and limits](https://docs.cloud.google.com/spanner/docs/pre-splitting-overview#quotas-limits) .

The file must use the following format to specify the split points:

```
  ObjectType ObjectName (SplitValue1, SplitValue2, ...)
  ObjectType ObjectName (IndexSplitValue1, ...) TableKey (TableSplitValue1, ...)
```

Replace the following variables when creating the file:

- ` ObjectType ` : the object type you want to add splits in. Valid values are `TABLE` and `INDEX` .
- ` ObjectName ` : the name of the database table or index.
- ` SplitValue1..N ` : the split point key values where you want to introduce the splits.
- ` IndexSplitValue1..N ` : the index key values for an index split point.
- ` TableSplitValue1..N ` : the table key values for an index split point.

Use the following rules when creating the split point values in the file:

- String values need to be in single quotes. For example, `'splitKeyPart'`
- Boolean values need to be either `true` or `false` .
- `INT64` and `NUMERIC` Spanner data type values need to be in single quotes. For example, `'123'` or `'99.99'` .
- All other number values need to be written without single quotes. For example, `1.287` .
- Timestamp values should be provided in the `'2020-06-18T17:24:53Z'` format in single quotes.
- Split values need to be surrounded by parentheses.
- The split keys value order must be the same as the primary key order.
- If the split value needs to have a comma, you must escape the comma using a \`\\\` character.
- For splitting indexes, you can either provide the index key or the entire index and the complete table key.
- You must always use the complete key when specifying the split point.

The following is a sample file that shows how split points are specified:

```
  TABLE Singers ('c32ca57a-786c-2268-09d4-95182a9930be')
  TABLE Singers ('bb98c7e2-8240-b780-346d-c5d63886594a')
  INDEX Order ('5b8bac71-0cb2-95e9-e1b0-89a027525460')
  TABLE Payment ('6cf41f21-2d77-318f-c504-816f0068db8b')
  INDEX Indx_A (2152120141932780000)
  TABLE TableD (0,'7ef9db22-d0e5-6041-8937-4bc6a7ef9db2')
  INDEX IndexXYZ ('8762203435012030000',NULL,NULL)
  INDEX IndexABC (0, '2020-06-18T17:24:53Z', '2020-06-18T17:24:53Z') TableKey (123,'ab\,c')
```

Before using any of the command data below, make the following replacements:

- ` SPLITS_FILE ` : the path to the splits file.
- ` INSTANCE_ID ` : the instance ID.
- ` DATABASE_ID ` : the database ID.
- ` EXPIRATION_DATE ` : (optional) the expiration date of the split points. Accepts a timestamp in the `'2020-06-18T17:24:53Z'` format.
- ` INITIATOR ` : (optional) the initiator of the split points.

Execute the following command:

#### Linux, macOS, or Cloud Shell

> **Note:** Ensure you have initialized the Google Cloud CLI with authentication and a project by running either [gcloud init](https://docs.cloud.google.com/sdk/gcloud/reference/init) ; or [gcloud auth login](https://docs.cloud.google.com/sdk/gcloud/reference/auth/login) and [gcloud config set project](https://docs.cloud.google.com/sdk/gcloud/reference/config/set) .

```
gcloud spanner databases splits add DATABASE_ID \
--splits-file=SPLITS_FILE \
--instance=INSTANCE_ID \
--split-expiration-date=EXPIRATION_DATE \
--initiator=INITIATOR
```

#### Windows (PowerShell)

> **Note:** Ensure you have initialized the Google Cloud CLI with authentication and a project by running either [gcloud init](https://docs.cloud.google.com/sdk/gcloud/reference/init) ; or [gcloud auth login](https://docs.cloud.google.com/sdk/gcloud/reference/auth/login) and [gcloud config set project](https://docs.cloud.google.com/sdk/gcloud/reference/config/set) .

```
gcloud spanner databases splits add DATABASE_ID `
--splits-file=SPLITS_FILE `
--instance=INSTANCE_ID `
--split-expiration-date=EXPIRATION_DATE `
--initiator=INITIATOR
```

#### Windows (cmd.exe)

> **Note:** Ensure you have initialized the Google Cloud CLI with authentication and a project by running either [gcloud init](https://docs.cloud.google.com/sdk/gcloud/reference/init) ; or [gcloud auth login](https://docs.cloud.google.com/sdk/gcloud/reference/auth/login) and [gcloud config set project](https://docs.cloud.google.com/sdk/gcloud/reference/config/set) .

```
gcloud spanner databases splits add DATABASE_ID ^
--splits-file=SPLITS_FILE ^
--instance=INSTANCE_ID ^
--split-expiration-date=EXPIRATION_DATE ^
--initiator=INITIATOR
```

### REST v1

You can use the [`projects.instances.databases.addSplitPoints`](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases/addSplitPoints) method to create split points.

Before using any of the request data, make the following replacements:

- ` PROJECT_ID ` : the project ID.
- ` INSTANCE_ID ` : the instance ID.
- ` DATABASE_ID ` : the database ID.

HTTP method and URL:

```
POST https://spanner.googleapis.com/v1/projects//instances//databases/:addSplitPoints
```

Request JSON body:

```
{
  "split_points": [
    {
      "table": "T1",
      "index": "T1_IDX",
      "expire_time": "2023-04-22T10:00:20.021Z",
      "keys": [
        {
          "key_parts": {
            "values": [
              3
            ]
          }
        },
        {
          "key_parts": {
            "values": [
              10
            ]
          }
        }
      ]
    },
    {
      "table": "T2",
      "expire_time": "2023-04-22T10:00:20.021Z",
      "keys": [
        {
          "key_parts": {
            "values": [
              50
            ]
          }
        }
      ]
    }
  ]
}
```

To send your request, expand one of these options:

#### curl (Linux, macOS, or Cloud Shell)

> **Note:** The following command assumes that you have logged in to the `gcloud` CLI with your user account by running [`gcloud init`](https://docs.cloud.google.com/sdk/gcloud/reference/init) or [`gcloud auth login`](https://docs.cloud.google.com/sdk/gcloud/reference/auth/login) , or by using [Cloud Shell](https://docs.cloud.google.com/shell/docs) , which automatically logs you into the `gcloud` CLI . You can check the currently active account by running [`gcloud auth list`](https://docs.cloud.google.com/sdk/gcloud/reference/auth/list) .

Save the request body in a file named `request.json` , and execute the following command:

```
curl -X POST \
     -H "Authorization: Bearer $(gcloud auth print-access-token)" \
     -H "Content-Type: application/json; charset=utf-8" \
     -d @request.json \
     "https://spanner.googleapis.com/v1/projects//instances//databases/:addSplitPoints"
```

#### PowerShell (Windows)

> **Note:** The following command assumes that you have logged in to the `gcloud` CLI with your user account by running [`gcloud init`](https://docs.cloud.google.com/sdk/gcloud/reference/init) or [`gcloud auth login`](https://docs.cloud.google.com/sdk/gcloud/reference/auth/login) . You can check the currently active account by running [`gcloud auth list`](https://docs.cloud.google.com/sdk/gcloud/reference/auth/list) .

Save the request body in a file named `request.json` , and execute the following command:

```
$cred = gcloud auth print-access-token
$headers = @{ "Authorization" = "Bearer $cred" }

Invoke-WebRequest `
    -Method POST `
    -Headers $headers `
    -ContentType: "application/json; charset=utf-8" `
    -InFile request.json `
    -Uri "https://spanner.googleapis.com/v1/projects//instances//databases/:addSplitPoints" | Select-Object -Expand Content
```

You should receive a successful status code (2xx) and an empty response.

### Client libraries

### Go

```go
import (
    "context"
    "fmt"
    "io"
    "time"

    database "cloud.google.com/go/spanner/admin/database/apiv1"
    "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
    adminpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
    "google.golang.org/protobuf/types/known/structpb"
    "google.golang.org/protobuf/types/known/timestamppb"
)

// Adds split points to table and index
// AddSplitPoins API - https://pkg.go.dev/cloud.google.com/go/spanner/admin/database/apiv1#DatabaseAdminClient.AddSplitPoints
func addSplitpoints(w io.Writer, dbName string) error {
    ctx := context.Background()

    dbAdminClient, err := database.NewDatabaseAdminClient(ctx)
    if err != nil {
        return err
    }
    defer dbAdminClient.Close()

    // Database is assumed to exist - https://cloud.google.com/spanner/docs/getting-started/go#create_a_database
    // Singers table is assumed to be present
    ddl := []string{
        "CREATE INDEX IF NOT EXISTS SingersByFirstLastName ON Singers(FirstName, LastName)",
    }
    op, err := dbAdminClient.UpdateDatabaseDdl(ctx, &adminpb.UpdateDatabaseDdlRequest{
        Database:   dbName,
        Statements: ddl,
    })

    if err != nil {
        return fmt.Errorf("addSplitPoints: waiting for UpdateDatabaseDdlRequest failed: %w", err)
    }

    // Wait for the UpdateDatabaseDdl operation to finish.
    if err := op.Wait(ctx); err != nil {
        return fmt.Errorf("addSplitPoints: waiting for UpdateDatabaseDdlRequest to finish failed: %w", err)
    }
    fmt.Fprintf(w, "Added indexes for Split testing\n")

    splitTableKey := databasepb.SplitPoints_Key{
        KeyParts: &structpb.ListValue{
            Values: []*structpb.Value{
                structpb.NewStringValue("42"),
            },
        },
    }

    splitForTable := databasepb.SplitPoints{
        Table: "Singers",
        Keys:  []*databasepb.SplitPoints_Key{&splitTableKey},
    }

    splitIndexKey := databasepb.SplitPoints_Key{
        KeyParts: &structpb.ListValue{
            Values: []*structpb.Value{
                structpb.NewStringValue("John"),
                structpb.NewStringValue("Doe"),
            },
        },
    }

    splitForindex := databasepb.SplitPoints{
        Index: "SingersByFirstLastName",
        Keys:  []*databasepb.SplitPoints_Key{&splitIndexKey},
    }

    splitIndexKeyWithTableKeyPart := databasepb.SplitPoints_Key{
        KeyParts: &structpb.ListValue{
            Values: []*structpb.Value{
                structpb.NewStringValue("38"),
            },
        },
    }

    splitIndexKeyWithIndexKeyPart := databasepb.SplitPoints_Key{
        KeyParts: &structpb.ListValue{
            Values: []*structpb.Value{
                structpb.NewStringValue("Jane"),
                structpb.NewStringValue("Doe"),
            },
        },
    }

    // the index key part is first and table keypart is second in the split definition
    splitForindexWithTableKey := databasepb.SplitPoints{
        Index: "SingersByFirstLastName",
        Keys:  []*databasepb.SplitPoints_Key{&splitIndexKeyWithIndexKeyPart, &splitIndexKeyWithTableKeyPart},
    }

    splitTableKeyWithExpire := databasepb.SplitPoints_Key{
        KeyParts: &structpb.ListValue{
            Values: []*structpb.Value{
                structpb.NewStringValue("30"),
            },
        },
    }

    splitForTableWithExpire := databasepb.SplitPoints{
        Table: "Singers",
        Keys:  []*databasepb.SplitPoints_Key{&splitTableKeyWithExpire},
        // A timestamp in the past means immediate expiration.
        // The maximum value can be 30 days in the future.
        // Defaults to 10 days in the future if not specified.
        //
        // Setting the expiration time to next day
        ExpireTime: timestamppb.New(time.Now().Add(24 * time.Hour)),
    }

    // Add split points to table and index
    req := databasepb.AddSplitPointsRequest{
        Database:    dbName,
        SplitPoints: []*databasepb.SplitPoints{&splitForTable, &splitForindex, &splitForindexWithTableKey, &splitForTableWithExpire},
    }

    res, err := dbAdminClient.AddSplitPoints(ctx, &req)
    if err != nil {
        return fmt.Errorf("addSplitpoints: failed to add split points: %w", err)
    }

    fmt.Fprintf(w, "Added split points %s", res)
    return nil
}
```

### Java

```java
import com.google.cloud.spanner.Spanner;
import com.google.cloud.spanner.SpannerException;
import com.google.cloud.spanner.SpannerOptions;
import com.google.cloud.spanner.admin.database.v1.DatabaseAdminClient;
import com.google.protobuf.ListValue;
import com.google.protobuf.Value;
import com.google.spanner.admin.database.v1.DatabaseName;
import com.google.spanner.admin.database.v1.SplitPoints;
import java.io.IOException;
import java.util.ArrayList;
import java.util.List;

public class DatabaseAddSplitPointsSample {

  /***
   * Assume DDL for the underlying database:
   * <pre>{@code
   * CREATE TABLE Singers (
   * SingerId INT64 NOT NULL,
   * FirstName STRING(1024),
   * LastName STRING(1024),
   *  SingerInfo BYTES(MAX),
   * ) PRIMARY KEY(SingerId);
   *
   *
   * CREATE INDEX SingersByFirstLastName ON Singers(FirstName, LastName);
   * }</pre>
   */

  static void addSplitPoints() throws IOException {
    // TODO(developer): Replace these variables before running the sample.
    String projectId = "my-project";
    String instanceId = "my-instance";
    String databaseId = "my-database";
    addSplitPoints(projectId, instanceId, databaseId);
  }

  static void addSplitPoints(String projectId, String instanceId, String databaseId)
      throws IOException {
    try (Spanner spanner =
            SpannerOptions.newBuilder().setProjectId(projectId).build().getService();
        DatabaseAdminClient databaseAdminClient = spanner.createDatabaseAdminClient()) {
      List<com.google.spanner.admin.database.v1.SplitPoints> splitPoints = new ArrayList<>();

      // table key
      com.google.spanner.admin.database.v1.SplitPoints splitPointForTable =
          SplitPoints.newBuilder()
              .setTable("Singers")
              .addKeys(
                  com.google.spanner.admin.database.v1.SplitPoints.Key.newBuilder()
                      .setKeyParts(
                          ListValue.newBuilder()
                              .addValues(Value.newBuilder().setStringValue("42").build())
                              .build()))
              .build();

      // index key without table key part
      com.google.spanner.admin.database.v1.SplitPoints splitPointForIndex =
          SplitPoints.newBuilder()
              .setIndex("SingersByFirstLastName")
              .addKeys(
                  com.google.spanner.admin.database.v1.SplitPoints.Key.newBuilder()
                      .setKeyParts(
                          ListValue.newBuilder()
                              .addValues(Value.newBuilder().setStringValue("John").build())
                              .addValues(Value.newBuilder().setStringValue("Doe").build())
                              .build()))
              .build();

      // index key with table key part, first key is the index key and second is the table key
      com.google.spanner.admin.database.v1.SplitPoints splitPointForIndexWitTableKey =
          SplitPoints.newBuilder()
              .setIndex("SingersByFirstLastName")
              .addKeys(
                  com.google.spanner.admin.database.v1.SplitPoints.Key.newBuilder()
                      .setKeyParts(
                          ListValue.newBuilder()
                              .addValues(Value.newBuilder().setStringValue("Jane").build())
                              .addValues(Value.newBuilder().setStringValue("Doe").build())
                              .build()))
              .addKeys(
                  com.google.spanner.admin.database.v1.SplitPoints.Key.newBuilder()
                      .setKeyParts(
                          ListValue.newBuilder()
                              .addValues(Value.newBuilder().setStringValue("38").build())
                              .build()))
              .build();

      splitPoints.add(splitPointForTable);
      splitPoints.add(splitPointForIndex);
      splitPoints.add(splitPointForIndexWitTableKey);
      databaseAdminClient.addSplitPoints(
          DatabaseName.of(projectId, instanceId, databaseId), splitPoints);

    } catch (Exception e) {
      // If the operation failed during execution, expose the cause.
      throw (SpannerException) e.getCause();
    }
  }
}
```

### Node.js

```javascript
// Import the Google Cloud client library for Spanner.
const {Spanner} = require('@google-cloud/spanner');

/**
 * TODO(developer): Uncomment these variables before running the sample.
 */
// const projectId = 'my-project-id';
// const instanceId = 'my-instance-id';
// const databaseId = 'my-database-id';

// Create a Spanner database admin client.
const spanner = new Spanner({projectId});
const databaseAdminClient = spanner.getDatabaseAdminClient();

try {
  // Add split points to table and index
  // first is a table level split that takes table primary key value
  // second is index level split with index key parts
  // third is index level split having index key part and table key part
  // Assume the following table and index structure
  // CREATE TABLE Singers (
  // SingerId INT64 NOT NULL,
  // FirstName STRING(1024),
  // LastName STRING(1024),
  // SingerInfo BYTES(MAX),
  // ) PRIMARY KEY(SingerId);
  //
  // CREATE INDEX SingersByFirstLastName ON Singers(FirstName, LastName);
  const request = [
    'CREATE INDEX IF NOT EXISTS SingersByFirstLastName ON Singers(FirstName, LastName)',
  ];

  const [operation] = await databaseAdminClient.updateDatabaseDdl({
    database: databaseAdminClient.databasePath(
      projectId,
      instanceId,
      databaseId,
    ),
    statements: request,
  });

  console.log('Waiting for operation to complete...');
  await operation.promise();

  console.log('Added the SingersByFirstLastName index.');

  databaseAdminClient.addSplitPoints({
    database: databaseAdminClient.databasePath(
      projectId,
      instanceId,
      databaseId,
    ),
    splitPoints: [
      {
        table: 'Singers',
        keys: [{keyParts: {values: [{stringValue: '42'}]}}],
      },
      {
        index: 'SingersByFirstLastName',
        keys: [
          {
            keyParts: {
              values: [{stringValue: 'John'}, {stringValue: 'Doe'}],
            },
          },
        ],
      },
      {
        index: 'SingersByFirstLastName',
        keys: [
          {
            keyParts: {
              values: [{stringValue: 'Jane'}, {stringValue: 'Doe'}],
            },
          },
          {keyParts: {values: [{stringValue: '38'}]}},
        ],
      },
    ],
  });
  console.log('Added Split Points');
} catch (err) {
  console.error('ERROR:', err);
}
```

### Python

```python
def add_split_points(instance_id, database_id):
    """Adds split points to table and index."""

    from google.cloud.spanner_admin_database_v1.types import spanner_database_admin

    spanner_client = spanner.Client()
    database_admin_api = spanner_client.database_admin_api

    request = spanner_database_admin.UpdateDatabaseDdlRequest(
        database=database_admin_api.database_path(
            spanner_client.project, instance_id, database_id
        ),
        statements=[
            "CREATE INDEX IF NOT EXISTS SingersByFirstLastName ON Singers(FirstName, LastName)"
        ],
    )

    operation = database_admin_api.update_database_ddl(request)

    print("Waiting for operation to complete...")
    operation.result(OPERATION_TIMEOUT_SECONDS)

    print("Added the SingersByFirstLastName index.")

    addSplitPointRequest = spanner_database_admin.AddSplitPointsRequest(
        database=database_admin_api.database_path(
            spanner_client.project, instance_id, database_id
        ),
        # Table split
        # Index split without table key part
        # Index split with table key part: first key is the index key and second the table key
        split_points=[
            spanner_database_admin.SplitPoints(
                table="Singers",
                keys=[
                    spanner_database_admin.SplitPoints.Key(
                        key_parts=struct_pb2.ListValue(
                            values=[struct_pb2.Value(string_value="42")]
                        )
                    )
                ],
            ),
            spanner_database_admin.SplitPoints(
                index="SingersByFirstLastName",
                keys=[
                    spanner_database_admin.SplitPoints.Key(
                        key_parts=struct_pb2.ListValue(
                            values=[
                                struct_pb2.Value(string_value="John"),
                                struct_pb2.Value(string_value="Doe"),
                            ]
                        )
                    )
                ],
            ),
            spanner_database_admin.SplitPoints(
                index="SingersByFirstLastName",
                keys=[
                    spanner_database_admin.SplitPoints.Key(
                        key_parts=struct_pb2.ListValue(
                            values=[
                                struct_pb2.Value(string_value="Jane"),
                                struct_pb2.Value(string_value="Doe"),
                            ]
                        )
                    ),
                    spanner_database_admin.SplitPoints.Key(
                        key_parts=struct_pb2.ListValue(
                            values=[struct_pb2.Value(string_value="38")]
                        )
                    ),
                ],
            ),
        ],
    )

    operation = database_admin_api.add_split_points(addSplitPointRequest)

    print("Added split points.")
```

### Possible error scenarios

The following scenarios can result in an error when creating split points:

- The index level split has an incorrect table name in the input.
- The table level split point has more than one key.
- The index level split point has more than two keys.
- The split points are defined on tables or indexes that aren't defined in the database schema.
- The request contains duplicate split points.

For information about quotas and limits, see [Quotas and limits](https://docs.cloud.google.com/spanner/quotas#split-point-limits) .

## View split points

You can view all the created split points on your database using the Google Cloud console or gcloud CLI:

### Console

To get the split point count by querying the `SPANNER_SYS.USER_SPLIT_POINTS` view in the Google Cloud console, do the following:

1.  Open the Spanner instances page.

2.  Select the names of the Spanner instance and the database that you want to query.

3.  Click **Spanner Studio** in the left navigation panel.

4.  Type the following query in the text field:

    ```
        SELECT * FROM SPANNER_SYS.USER_SPLIT_POINTS
    ```

5.  Click **Run query** .

A result similar to the following appears:

| TABLE_NAME | INDEX_NAME | INITIATOR              | SPLIT_KEY                                                                                                                      | EXPIRE_TIME                 |
|------------|------------|------------------------|--------------------------------------------------------------------------------------------------------------------------------|-----------------------------|
| T          |            | CloudAddSplitPointsAPI | T(90,153,4,2024-04-30T17:00:00-07:00,1,2024-05-01,a)                                                                           | 2025-03-06T09:58:58.007201Z |
| T          | T_IDX      | CloudAddSplitPointsAPI | Index: T_IDX on T, Index Key: (10), Primary Table Key: (\<begin\>,\<begin\>,\<begin\>,\<begin\>,\<begin\>,\<begin\>,\<begin\>) | 2025-03-08T07:33:23.861682Z |
| T          | T_IDX      | CloudAddSplitPointsAPI | Index: T_IDX on T, Index Key: (9091), Primary Table Key: (4567,123,4.2,2024-04-30T17:00:00-07:00,1,2024-05-01,a)               | 2025-03-08T07:35:25.990007Z |

### gcloud

Run the following gcloud CLI command to view split points in your database:

```
  gcloud spanner databases splits list DATABASE_ID \
  --instance INSTANCE_ID
```

Replace the following variables when running this command:

- ` INSTANCE_ID ` : the Spanner instance ID.
- ` DATABASE_ID ` : the Spanner database ID.

A response similar to the following appears:

```
  TABLE_NAME: T
  INDEX_NAME:
  INITIATOR: CloudAddSplitPointsAPI
  SPLIT_KEY: T(90,153,4,2024-04-30T17:00:00-07:00,1,2024-05-01,a)
  EXPIRE_TIME: 2025-03-06T09:58:58.007201Z

  TABLE_NAME: T
  INDEX_NAME: T_IDX
  INITIATOR: CloudAddSplitPointsAPI
  SPLIT_KEY: Index: T_IDX on T, Index Key: (10), Primary Table Key: (<begin>,<begin>,<begin>,<begin>,<begin>,<begin>,<begin>)
  EXPIRE_TIME: 2025-03-08T07:33:23.861682Z

  TABLE_NAME: T
  INDEX_NAME: T_IDX
  INITIATOR: CloudAddSplitPointsAPI
  SPLIT_KEY: Index: T_IDX on T, Index Key: (9091), Primary Table Key: (4567,123,4.2,2024-04-30T17:00:00-07:00,1,2024-05-01,a)
  EXPIRE_TIME: 2025-03-08T07:35:25.990007Z
```

## How to expire a split point

You can set an expiration time for each split point you create. For more information, see [Split point expiration](https://docs.cloud.google.com/spanner/docs/pre-splitting-overview#expire) . You can expire split points using the Google Cloud CLI or the REST APIs.

### gcloud

If you want to expire split points the Google Cloud CLI, you need to create a file that contains all the splits you want to expire and supply its path using the `splits-file` parameter in the gcloud CLI command. The file must not exceed the API limit of 100 points per API request. For more information, see [Quotas and limits](https://docs.cloud.google.com/spanner/quotas#split-point-limits) .

The file must use the following format to specify the split points:

```
  ObjectType ObjectName (SplitValue1, SplitValue2, ...)
  ObjectType ObjectName (IndexSplitValue1, ...) TableKey (TableSplitValue1, ...)
```

Replace the following variables when creating the file:

- ` ObjectType ` : the object type of the split you want to expire. Valid values are `TABLE` and `INDEX` .
- ` ObjectName ` : the name of the database table or index.
- ` SplitValue1..N ` : the split point key values you want to expire.
- ` IndexSplitValue1..N ` : the index key values for an index split point.
- ` TableSplitValue1..N ` : the table key values for an index split point.

Use the following rules when creating the split point values in the file:

- String values need to be in single quotes. For example, `'splitKeyPart'`
- Boolean values can be either `true` or `false` .
- `INT64` and `NUMERIC` Spanner data type values need to be in single quotes. For example, `'123'` or `'99.99'` .
- All other number values need to be written without single quotes. For example, `1.287` .
- Timestamp values should be provided in the `'2020-06-18T17:24:53Z'` format in single quotes.
- Split values need to be surrounded by parentheses.
- The split keys value order must be the same as the primary key order.
- If the split value needs to have a comma, you must escape the comma using a \`\\\` character.
- For splitting indexes, you can either provide the index key or the entire index and the complete table key.
- You must always use the complete key when specifying the split point.

The following is a sample file that shows how split points are specified:

```
  TABLE Singers ('c32ca57a-786c-2268-09d4-95182a9930be')
  TABLE Singers ('bb98c7e2-8240-b780-346d-c5d63886594a')
  INDEX Order ('5b8bac71-0cb2-95e9-e1b0-89a027525460')
  TABLE Payment ('6cf41f21-2d77-318f-c504-816f0068db8b')
  INDEX Indx_A (2152120141932780000)
  TABLE TableD (0,'7ef9db22-d0e5-6041-8937-4bc6a7ef9db2')
  INDEX IndexXYZ ('8762203435012030000',NULL,NULL)
  INDEX IndexABC (0, '2020-06-18T17:24:53Z', '2020-06-18T17:24:53Z') TableKey (123,'ab\,c')
```

Before using any of the command data below, make the following replacements:

- ` SPLITS_FILE ` : the path to the splits file.
- ` INSTANCE_ID ` : the instance ID.
- ` DATABASE_ID ` : the database ID.
- ` EXPIRATION_DATE ` : (optional) the expiration date of the split points. Accepts a timestamp in the `'2020-06-18T17:24:53Z'` format.
- ` INITIATOR ` : (optional) the initiator of the split points.

Execute the following command:

#### Linux, macOS, or Cloud Shell

> **Note:** Ensure you have initialized the Google Cloud CLI with authentication and a project by running either [gcloud init](https://docs.cloud.google.com/sdk/gcloud/reference/init) ; or [gcloud auth login](https://docs.cloud.google.com/sdk/gcloud/reference/auth/login) and [gcloud config set project](https://docs.cloud.google.com/sdk/gcloud/reference/config/set) .

```
gcloud spanner databases splits add DATABASE_ID \
--splits-file=SPLITS_FILE \
--instance=INSTANCE_ID \
--split-expiration-date=EXPIRATION_DATE \
--initiator=INITIATOR
```

#### Windows (PowerShell)

> **Note:** Ensure you have initialized the Google Cloud CLI with authentication and a project by running either [gcloud init](https://docs.cloud.google.com/sdk/gcloud/reference/init) ; or [gcloud auth login](https://docs.cloud.google.com/sdk/gcloud/reference/auth/login) and [gcloud config set project](https://docs.cloud.google.com/sdk/gcloud/reference/config/set) .

```
gcloud spanner databases splits add DATABASE_ID `
--splits-file=SPLITS_FILE `
--instance=INSTANCE_ID `
--split-expiration-date=EXPIRATION_DATE `
--initiator=INITIATOR
```

#### Windows (cmd.exe)

> **Note:** Ensure you have initialized the Google Cloud CLI with authentication and a project by running either [gcloud init](https://docs.cloud.google.com/sdk/gcloud/reference/init) ; or [gcloud auth login](https://docs.cloud.google.com/sdk/gcloud/reference/auth/login) and [gcloud config set project](https://docs.cloud.google.com/sdk/gcloud/reference/config/set) .

```
gcloud spanner databases splits add DATABASE_ID ^
--splits-file=SPLITS_FILE ^
--instance=INSTANCE_ID ^
--split-expiration-date=EXPIRATION_DATE ^
--initiator=INITIATOR
```

### REST v1

Before using any of the request data, make the following replacements:

- ` PROJECT_ID ` : the project ID.
- ` INSTANCE_ID ` : the instance ID.
- ` DATABASE_ID ` : the database ID.

HTTP method and URL:

```
POST https://spanner.googleapis.com/v1/projects//instances//databases/:addSplitPoints
```

Request JSON body:

```
{
  "split_points": [
    {
      "table": "T1",
      "index": "T1_IDX",
      "expire_time": "2023-04-22T10:00:20.021Z",
      "keys": [
        {
          "key_parts": {
            "values": [
              3
            ]
          }
        },
        {
          "key_parts": {
            "values": [
              10
            ]
          }
        }
      ]
    },
    {
      "table": "T2",
      "expire_time": "2023-04-22T10:00:20.021Z",
      "keys": [
        {
          "key_parts": {
            "values": [
              50
            ]
          }
        }
      ]
    }
  ]
}
```

To send your request, expand one of these options:

#### curl (Linux, macOS, or Cloud Shell)

> **Note:** The following command assumes that you have logged in to the `gcloud` CLI with your user account by running [`gcloud init`](https://docs.cloud.google.com/sdk/gcloud/reference/init) or [`gcloud auth login`](https://docs.cloud.google.com/sdk/gcloud/reference/auth/login) , or by using [Cloud Shell](https://docs.cloud.google.com/shell/docs) , which automatically logs you into the `gcloud` CLI . You can check the currently active account by running [`gcloud auth list`](https://docs.cloud.google.com/sdk/gcloud/reference/auth/list) .

Save the request body in a file named `request.json` , and execute the following command:

```
curl -X POST \
     -H "Authorization: Bearer $(gcloud auth print-access-token)" \
     -H "Content-Type: application/json; charset=utf-8" \
     -d @request.json \
     "https://spanner.googleapis.com/v1/projects//instances//databases/:addSplitPoints"
```

#### PowerShell (Windows)

> **Note:** The following command assumes that you have logged in to the `gcloud` CLI with your user account by running [`gcloud init`](https://docs.cloud.google.com/sdk/gcloud/reference/init) or [`gcloud auth login`](https://docs.cloud.google.com/sdk/gcloud/reference/auth/login) . You can check the currently active account by running [`gcloud auth list`](https://docs.cloud.google.com/sdk/gcloud/reference/auth/list) .

Save the request body in a file named `request.json` , and execute the following command:

```
$cred = gcloud auth print-access-token
$headers = @{ "Authorization" = "Bearer $cred" }

Invoke-WebRequest `
    -Method POST `
    -Headers $headers `
    -ContentType: "application/json; charset=utf-8" `
    -InFile request.json `
    -Uri "https://spanner.googleapis.com/v1/projects//instances//databases/:addSplitPoints" | Select-Object -Expand Content
```

You should receive a successful status code (2xx) and an empty response.

## What's next?

- [Pre-splitting overview](https://docs.cloud.google.com/spanner/docs/pre-splitting-overview)
