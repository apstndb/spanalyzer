---
name: documents/docs.cloud.google.com/spanner/docs/databoost/databoost-run-queries
uri: https://docs.cloud.google.com/spanner/docs/databoost/databoost-run-queries
title: Run federated queries with Data Boost
description: Run high-performance, serverless queries on Spanner using Spanner Data Boost from BigQuery or Dataflow.
data_source: docs.cloud.google.com
---

This page explains how to use Spanner Data Boost when you run federated queries from BigQuery to a Spanner database. With Data Boost, federated queries run with minimal impact to existing workloads on the provisioned Spanner instance. The Data Boost queries from BigQuery to a Spanner database can join BigQuery data with Spanner data.

You can run federated queries from BigQuery to Spanner using Data Boost with one of the following methods:

- [Run a federated Data Boost query](https://docs.cloud.google.com/spanner/docs/databoost/databoost-run-queries#run-data-boost-query) , which uses the `EXTERNAL_QUERY` federated query function.
- [Use Data Boost with external datasets](https://docs.cloud.google.com/spanner/docs/databoost/databoost-run-queries#run-data-boost-external-dataset) .

Spanner federation lets BigQuery query data residing in Spanner in real time, without copying or moving data. To learn more about Spanner federated queries, see [Spanner federated queries](https://docs.cloud.google.com/bigquery/docs/spanner-federated-queries) . To learn about Data Boost, see [Data Boost overview](https://docs.cloud.google.com/spanner/docs/databoost/databoost-overview) .

## Before you begin

Before you can run federated queries with Data Boost, you need to complete the following tasks:

- [Create a Spanner instance and database](https://docs.cloud.google.com/spanner/docs/databoost/databoost-run-queries#create-instance-database) .

- [Enable the BigQuery connection API](https://docs.cloud.google.com/spanner/docs/databoost/databoost-run-queries#enable-bqconnect-API) .

- [Grant the required IAM permissions](https://docs.cloud.google.com/spanner/docs/databoost/databoost-run-queries#grant-databoost-perms) .

### Create a Spanner instance and database

If you don't have a Spanner instance and database, follow the steps in [Create and query a database using the Google Cloud console](https://docs.cloud.google.com/spanner/docs/create-query-database-console) to create them.

### Enable the BigQuery connection API

The BigQuery connection API lets you manage BigQuery connections to external data sources such as a Spanner database.

- Enable the BigQuery connection API, if it is not already enabled.

  **Roles required to enable APIs**

  To enable APIs, you need the `serviceusage.services.enable` permission. If you created the project, then you likely already have this permission through the Owner role ( `roles/owner` ). Otherwise, you can get this permission through the Service Usage Admin role ( `roles/serviceusage.serviceUsageAdmin` ). [Learn how to grant roles](https://docs.cloud.google.com/iam/docs/granting-changing-revoking-access) .

For more information, see [BigQuery connection API](https://docs.cloud.google.com/bigquery/docs/reference/bigqueryconnection/rest) in the BigQuery documentation.

### Grant IAM permissions for Data Boost to principals

A principal must be granted the following permissions to run federated queries with Data Boost:

- `spanner.instances.get` - lets you get the configuration of an instance.
- `spanner.databases.useDataBoost` - lets you use the [Spanner Data Boost](https://docs.cloud.google.com/spanner/docs/databoost/databoost-overview) compute resources to process [partitioned queries](https://docs.cloud.google.com/spanner/docs/reads#read_data_in_parallel) .

For more information about Spanner permissions, see [Identity and Access Management (IAM) permissions](https://docs.cloud.google.com/spanner/docs/iam#permissions) .

To grant these required permissions, we recommend that you use the `Cloud Spanner Database Reader With DataBoost` ( `roles/spanner.databaseReaderWithDataBoost` ) IAM role. You can add that role to any principal that needs to run federated queries with Data Boost. To learn more about predefined roles in Spanner, see [Predefined roles](https://docs.cloud.google.com/spanner/docs/iam#roles) . To learn how to create a custom IAM role, see [Create a custom role](https://docs.cloud.google.com/iam/docs/creating-custom-roles#creating) .

## Run a federated Data Boost query

To run a Data Boost query from BigQuery to an external source, you need a BigQuery connection to the external source and the ID of the connection. When you run a federated Spanner query with Data Boost, the external source is a Spanner database. After you create your connection ID, it's used by BigQuery to run a Data Boost query of a Spanner database.

Use one of the following options to create a BigQuery connection ID, and then use the connection ID to run a Data Boost query from BigQuery:

1.  [Start in Spanner](https://docs.cloud.google.com/spanner/docs/databoost/databoost-run-queries#run-query-from-spanner) - Create the BigQuery external connection ID in the Spanner console. After your connection ID is created in the Spanner console, you're redirected to the BigQuery console to run a federated Data Boost query to a Spanner database.

2.  [Start in BigQuery](https://docs.cloud.google.com/spanner/docs/databoost/databoost-run-queries#run-query-from-bigquery) - Create the Data Boost external connection ID in the BigQuery console or using the `bq` command-line tool. After you create the connection ID, you stay in the BigQuery console to run a federated Data Boost query to a Spanner database.

### Start in Spanner to run a Data Boost query

To run a federated Data Boost query starting in the Spanner Studio, do the following:

1.  Go to the Spanner **Instances** page in the Google Cloud console.

    The console shows a list of your Spanner instances.

2.  Select a Spanner instance, and then select a database.

3.  On the **Database overview** page, in the navigation menu, click **Spanner Studio** .

4.  Click **View in BiqQuery** .

5.  In the **View in BigQuery** dialog, enter a connection ID.

    The connection ID is used to create a new BigQuery external connection to your Spanner database. You reference your external connection using the following pattern:

    ```
    PROJECT-ID.LOCATION.CONNECTION-ID
    ```

    An error occurs if the ID already exists.

6.  Fill in the rest of the dialog and do the following:

    - Select **Read data in parallel** .
    - Select **Use Spanner Data Boost** .

7.  Click **View in BigQuery** .

    BigQuery Studio opens with the following query:

    ```
    SELECT * FROM EXTERNAL_QUERY("PROJECT-ID.LOCATION.CONNECTION-ID", "SELECT * FROM INFORMATION_SCHEMA.TABLES;");
    ```

    You can replace this with your federated query. For example, you might make a query that's similar to the following example. This example makes a federated query from a table named `orders` in a Spanner database and joins the results with a BigQuery table named `mydataset.customers` .

    ```
    SELECT c.customer_id, c.name, rq.first_order_date
    FROM mydataset.customers AS c
    LEFT OUTER JOIN EXTERNAL_QUERY(
      'my-project.us.example-db',
      '''SELECT customer_id, MIN(order_date) AS first_order_date
      FROM orders
      GROUP BY customer_id''') AS rq
      ON rq.customer_id = c.customer_id
    GROUP BY c.customer_id, c.name, rq.first_order_date;
    ```

### Start in BigQuery to run a Data Boost query

To create an external data connection from BigQuery to a Spanner database and use that connection to run a federated Data Boost query from BigQuery, select one of the following options:

### Console

1.  Go to [Create Spanner connections](https://docs.cloud.google.com/bigquery/docs/connect-to-spanner#create-spanner-connection) in the BigQuery documentation and follow the instructions on the **Console** tab.

2.  In the **External data source** pane, do the following:

    - Select **Read data in parallel** .
    - Select **Use Spanner Data Boost** .

### bq

1.  Go to [Create Spanner connections](https://docs.cloud.google.com/bigquery/docs/connect-to-spanner#create-spanner-connection) in the BigQuery documentation and follow instructions on the **bq** \* tab.

2.  Set the following connection properties to `true` :

    - `useParallelism`
    - `useDataBoost`

The following example uses the [`bq mk`](https://docs.cloud.google.com/bigquery/docs/reference/bq-cli-reference#bq_mk) command to create a new connection named `my_connection` with the two required properties for Data Boost:

```
bq mk --connection --connection_type='CLOUD_SPANNER' --location='us' \
--properties='{"database":"projects/my-project/instances/my-instance/databases/my-database", "useParallelism":true, "useDataBoost": true}' my_connection
```

## Use Data Boost with external datasets

To run a Data Boost query from BigQuery to Spanner as an external source, you can create an [external dataset](https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets) (also known as a federated dataset) in BigQuery that's linked to an existing GoogleSQL or PostgreSQL database in Spanner.

### Use a `CLOUD_RESOURCE` connection

By default, Spanner external datasets use end-user credentials (EUC), which requires the users have direct access to their Spanner databases. Users can query these datasets if they have access granted in Spanner.

Optionally, Spanner external datasets can use a `CLOUD_RESOURCE` connection to interact with your Spanner database, so that you can provide a user access to Spanner data through BigQuery, without giving them direct access to the Spanner database. Because the service account from `CLOUD_RESOURCE` connection handles retrieving data from the Spanner, you only have to grant users access to the Spanner external dataset. This access delegation decouples access to the Spanner tables from external datasets and the direct access to the underlying Spanner tables. A Cloud resource connection associated with a service account is used to connect to the Spanner. Users can query these Spanner tables from external datasets even if they don't have access granted in Spanner.

Before you create Spanner external datasets with a `CLOUD_RESOURCE` connection, do the following:

#### Create a connection

You can create or use an existing [`CLOUD_RESOURCE` connection](https://docs.cloud.google.com/bigquery/docs/create-cloud-resource-connection) to connect to Spanner. Make sure to create the connection in the same [location](https://docs.cloud.google.com/bigquery/docs/locations) that you plan to create your Spanner external dataset.

Select one of the following options:

### Console

1.  Go to the **BigQuery** page.

2.  In the left pane, click explore **Explorer** :

    ![Highlighted button for the Explorer pane.](https://docs.cloud.google.com/static/bigquery/images/explorer-tab.png)

    If you don't see the left pane, click last_page **Expand left pane** to open the pane.

3.  In the **Explorer** pane, expand your project name, and then click **Connections** .

4.  On the **Connections** page, click **Create connection** .

5.  For **Connection type** , choose **Vertex AI remote models, remote functions, BigLake and Spanner (Cloud Resource)** .

6.  In the **Connection ID** field, enter a name for your connection.

7.  For **Location type** , select a location for your connection. The connection should be colocated with your other resources such as datasets.

8.  Click **Create connection** .

9.  Click **Go to connection** .

10. In the **Connection info** pane, copy the service account ID for use in a later step.

### SQL

Use the [`CREATE CONNECTION` statement](https://docs.cloud.google.com/bigquery/docs/reference/standard-sql/data-definition-language#create_connection_statement) :

1.  In the Google Cloud console, go to the **BigQuery** page.

2.  In the query editor, enter the following statement:

    ```
    CREATE CONNECTION [IF NOT EXISTS] `CONNECTION_NAME`
    OPTIONS (
      connection_type = "CLOUD_RESOURCE",
      friendly_name = "FRIENDLY_NAME",
      description = "DESCRIPTION"
      );
    ```

    Replace the following:

    - `CONNECTION_NAME` : the name of the connection in either the `PROJECT_ID `` . `` LOCATION `` . `` CONNECTION_ID` , `LOCATION `` . `` CONNECTION_ID` , or `CONNECTION_ID` format. If the project or location are omitted, then they are inferred from the project and location where the statement is run.
    - `FRIENDLY_NAME` (optional): a descriptive name for the connection.
    - `DESCRIPTION` (optional): a description of the connection.

3.  Click play_circle **Run** .

For more information about how to run queries, see [Run an interactive query](https://docs.cloud.google.com/bigquery/docs/running-queries#queries) .

### bq

1.  In a command-line environment, create a connection:

    ```
    bq mk --connection --location=REGION --project_id=PROJECT_ID \
        --connection_type=CLOUD_RESOURCE CONNECTION_ID
    ```

    The `--project_id` parameter overrides the default project.

    Replace the following:

    - `REGION` : your [connection region](https://docs.cloud.google.com/bigquery/docs/locations#supported_locations)
    - `PROJECT_ID` : your Google Cloud project ID
    - `CONNECTION_ID` : an ID for your connection

    When you create a connection resource, BigQuery creates a unique system service account and associates it with the connection.

    **Troubleshooting** : If you get the following connection error, [update the Google Cloud SDK](https://docs.cloud.google.com/sdk/docs/quickstart) :

    ```console
    Flags parsing error: flag --connection_type=CLOUD_RESOURCE: value should be one of...
    ```

2.  Retrieve and copy the service account ID for use in a later step:

    ```
    bq show --connection PROJECT_ID.REGION.CONNECTION_ID
    ```

    The output is similar to the following:

    ```console
    name                          properties
    1234.REGION.CONNECTION_ID     {"serviceAccountId": "connection-1234-9u56h9@gcp-sa-bigquery-condel.iam.gserviceaccount.com"}
    ```

### Python

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```python
import google.api_core.exceptions
from google.cloud import bigquery_connection_v1

client = bigquery_connection_v1.ConnectionServiceClient()


def create_connection(
    project_id: str,
    location: str,
    connection_id: str,
):
    """Creates a BigQuery connection to a Cloud Resource.

    Cloud Resource connection creates a service account which can then be
    granted access to other Google Cloud resources for federated queries.

    Args:
        project_id: The Google Cloud project ID.
        location: The location of the connection (for example, "us-central1").
        connection_id: The ID of the connection to create.
    """

    parent = client.common_location_path(project_id, location)

    connection = bigquery_connection_v1.Connection(
        friendly_name="Example Connection",
        description="A sample connection for a Cloud Resource.",
        cloud_resource=bigquery_connection_v1.CloudResourceProperties(),
    )

    try:
        created_connection = client.create_connection(
            parent=parent, connection_id=connection_id, connection=connection
        )
        print(f"Successfully created connection: {created_connection.name}")
        print(f"Friendly name: {created_connection.friendly_name}")
        print(
            f"Service Account: {created_connection.cloud_resource.service_account_id}"
        )

    except google.api_core.exceptions.AlreadyExists:
        print(f"Connection with ID '{connection_id}' already exists.")
        print("Please use a different connection ID.")
    except Exception as e:
        print(f"An unexpected error occurred while creating the connection: {e}")
```

### Node.js

To learn how to install and use the client library for Spanner, see [Spanner client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) .

To authenticate to Spanner, set up Application Default Credentials. For more information, see [Set up authentication for a local development environment](https://docs.cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) .

```javascript
const {ConnectionServiceClient} =
  require('@google-cloud/bigquery-connection').v1;
const {status} = require('@grpc/grpc-js');

const client = new ConnectionServiceClient();

/**
 * Creates a new BigQuery connection to a Cloud Resource.
 *
 * A Cloud Resource connection creates a service account that can be granted access
 * to other Google Cloud resources.
 *
 * @param {string} projectId The Google Cloud project ID. for example, 'example-project-id'
 * @param {string} location The location of the project to create the connection in. for example, 'us-central1'
 * @param {string} connectionId The ID of the connection to create. for example, 'example-connection-id'
 */
async function createConnection(projectId, location, connectionId) {
  const parent = client.locationPath(projectId, location);

  const connection = {
    friendlyName: 'Example Connection',
    description: 'A sample connection for a Cloud Resource',
    // The service account for this cloudResource will be created by the API.
    // Its ID will be available in the response.
    cloudResource: {},
  };

  const request = {
    parent,
    connectionId,
    connection,
  };

  try {
    const [response] = await client.createConnection(request);

    console.log(`Successfully created connection: ${response.name}`);
    console.log(`Friendly name: ${response.friendlyName}`);

    console.log(`Service Account: ${response.cloudResource.serviceAccountId}`);
  } catch (err) {
    if (err.code === status.ALREADY_EXISTS) {
      console.log(`Connection '${connectionId}' already exists.`);
    } else {
      console.error(`Error creating connection: ${err.message}`);
    }
  }
}
```

### Terraform

Use the [`google_bigquery_connection`](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_connection) resource.

> **Note:** To create BigQuery objects using Terraform, you must enable the [Cloud Resource Manager API](https://docs.cloud.google.com/resource-manager/reference/rest) .

To authenticate to BigQuery, set up Application Default Credentials. For more information, see [Set up authentication for client libraries](https://docs.cloud.google.com/bigquery/docs/authentication#client-libs) .

The following example creates a Cloud resource connection named `my_cloud_resource_connection` in the `US` region:

```terraform
# This queries the provider for project information.
data "google_project" "default" {}

# This creates a cloud resource connection in the US region named my_cloud_resource_connection.
# Note: The cloud resource nested object has only one output field - serviceAccountId.
resource "google_bigquery_connection" "default" {
  connection_id = "my_cloud_resource_connection"
  project       = data.google_project.default.project_id
  location      = "US"
  cloud_resource {}
}
```

To apply your Terraform configuration in a Google Cloud project, complete the steps in the following sections.

## Prepare Cloud Shell

1.  Launch [Cloud Shell](https://shell.cloud.google.com/) .

2.  Set the default Google Cloud project where you want to apply your Terraform configurations.

    You only need to run this command once per project, and you can run it in any directory.

    ```
    export GOOGLE_CLOUD_PROJECT=PROJECT_ID
    ```

    Environment variables are overridden if you set explicit values in the Terraform configuration file.

## Prepare the directory

Each Terraform configuration file must have its own directory (also called a *root module* ).

1.  In [Cloud Shell](https://shell.cloud.google.com/) , create a directory and a new file within that directory. The filename must have the `.tf` extension—for example `main.tf` . In this tutorial, the file is referred to as `main.tf` .

    ```
    mkdir DIRECTORY && cd DIRECTORY && touch main.tf
    ```

2.  If you are following a tutorial, you can copy the sample code in each section or step.

    Copy the sample code into the newly created `main.tf` .

    Optionally, copy the code from GitHub. This is recommended when the Terraform snippet is part of an end-to-end solution.

3.  Review and modify the sample parameters to apply to your environment.

4.  Save your changes.

5.  Initialize Terraform. You only need to do this once per directory.

    ```
    terraform init
    ```

    Optionally, to use the latest Google provider version, include the `-upgrade` option:

    ```
    terraform init -upgrade
    ```

## Apply the changes

1.  Review the configuration and verify that the resources that Terraform is going to create or update match your expectations:

    ```
    terraform plan
    ```

    Make corrections to the configuration as necessary.

2.  Apply the Terraform configuration by running the following command and entering `yes` at the prompt:

    ```
    terraform apply
    ```

    Wait until Terraform displays the "Apply complete!" message.

3.  [Open your Google Cloud project](https://console.cloud.google.com/) to view the results. In the Google Cloud console, navigate to your resources in the UI to make sure that Terraform has created or updated them.

> **Note:** Terraform samples typically assume that the required APIs are enabled in your Google Cloud project.

After you create the connection, open it, and in the **Connection info** pane, copy the service account ID. You need this ID when you configure permissions for the connection. When you create a connection resource, BigQuery creates a unique system service account and associates it with the connection.

#### Set up access

You must give the service account that is associated with the new connection read access to your Spanner instance or database. It is recommended to use the Cloud Spanner Database Reader with Data Boost ( `roles/spanner.databaseReaderWithDataBoost` ) predefined IAM role.

Follow these steps to grant access to database-level roles for the service account that you copied earlier from the connection:

1.  Go to the Spanner **Instances** page.

2.  Click the name of the instance that contains your database to go to the **Instance details** page.

3.  In the **Overview** tab, select the checkbox for your database.  
    The **Info panel** appears.

4.  Click **Add principal** .

5.  In the **Add principals** panel, in **New principals** , enter the service account ID that you copied earlier.

6.  In the **Select a role** field, select **Cloud Spanner Database Reader with DataBoost role** .

7.  Click **Save** .

### Create an external dataset

To create an external dataset, do the following:

### Console

1.  Open the BigQuery page in the Google Cloud console.

2.  In the left pane, click explore **Explorer** :

    ![Highlighted button for the Explorer pane.](https://docs.cloud.google.com/static/bigquery/images/explorer-tab.png)

    If you don't see the left pane, click last_page **Expand left pane** to open the pane.

3.  In the **Explorer** pane, select the project where you want to create the dataset.

4.  Click more_vert **View actions** , and then click **Create dataset** .

5.  On the **Create dataset** page, do the following:

    - For **Dataset ID** , enter a unique dataset name.

    - For **Location type** , choose a location for the dataset, such as `us-central1` or multiregion `us` . After you create a dataset, the location can't be changed.

    - For **External Dataset** , do the following:

      - Check the box next to **Link to an external dataset** .
      - For **External dataset type** , select `Spanner` .
      - For **External source** , enter the full identifier of your Spanner database in the following format: `projects/ `` PROJECT_ID `` /instances/ `` INSTANCE `` /databases/ `` DATABASE` . For example: `projects/my_project/instances/my_instance/databases/my_database` .
      - Optionally, for **Database role** enter the name of a Spanner database role. For more information read about database roles used for [creating Spanner Connections](https://docs.cloud.google.com/bigquery/docs/connect-to-spanner#create-spanner-connection)
      - Optionally, check the box next to **Use a Cloud Resource connection** to create the external dataset with a connection.

    - Leave the other default settings as they are.

6.  Click **Create dataset** .

### SQL

Use the [`CREATE EXTERNAL SCHEMA` data definition language (DDL) statement](https://docs.cloud.google.com/bigquery/docs/reference/standard-sql/data-definition-language#create_external_schema_statement) .

1.  In the Google Cloud console, go to the **BigQuery** page.

2.  In the query editor, enter the following statement:

    ```
    CREATE EXTERNAL SCHEMA DATASET_NAME
      OPTIONS (
        external_source = 'SPANNER_EXTERNAL_SOURCE',
        location = 'LOCATION');
    /*
      Alternatively, create with a connection:
    */
    CREATE EXTERNAL SCHEMA DATASET_NAME
      WITH CONNECTION PROJECT_ID.LOCATION.CONNECTION_NAME
      OPTIONS (
        external_source = 'SPANNER_EXTERNAL_SOURCE',
        location = 'LOCATION');
    ```

    Replace the following:

    - `DATASET_NAME` : the name of your new dataset in BigQuery.
    - `SPANNER_EXTERNAL_SOURCE` : the full, qualified Spanner database name, with a prefix identifying the source, in the following format: `google-cloudspanner://[ `` DATABASE_ROLE `` @]/projects/ `` PROJECT_ID `` /instances/ `` INSTANCE `` /databases/ `` DATABASE` . For example: `google-cloudspanner://admin@/projects/my_project/instances/my_instance/databases/my_database` or `google-cloudspanner:/projects/my_project/instances/my_instance/databases/my_database` .
    - `LOCATION` : the location of your new dataset in BigQuery, for example, `us-central1` . After you create a dataset, you can't change its location.
    - (Optional) `CONNECTION_NAME` : the name of your Cloud resource connection.

3.  Click play_circle **Run** .

For more information about how to run queries, see [Run an interactive query](https://docs.cloud.google.com/bigquery/docs/running-queries#queries) .

### bq

In a command-line environment, create an external dataset by using the [`bq mk` command](https://docs.cloud.google.com/bigquery/docs/reference/bq-cli-reference#mk-dataset) :

```
bq --location=LOCATION mk --dataset \
    --external_source SPANNER_EXTERNAL_SOURCE \
    DATASET_NAME
```

Alternatively, create with a connection:

```
bq --location=LOCATION mk --dataset \
    --external_source SPANNER_EXTERNAL_SOURCE \
    --connection_id PROJECT_ID.LOCATION.CONNECTION_NAME \
    DATASET_NAME
```

Replace the following:

- `LOCATION` : the location of your new dataset in BigQuery—for example, `us-central1` . After you create a dataset, you can't change its location. You can set a default location value by using the [`.bigqueryrc` file](https://docs.cloud.google.com/bigquery/docs/bq-command-line-tool#setting_default_values_for_command-line_flags) .
- `SPANNER_EXTERNAL_SOURCE` : the full, qualified Spanner database name, with a prefix identifying the source, in the following format: `google-cloudspanner://[ `` DATABASE_ROLE `` @]/projects/ `` PROJECT_ID `` /instances/ `` INSTANCE `` /databases/ `` DATABASE` . For example: `google-cloudspanner://admin@/projects/my_project/instances/my_instance/databases/my_database` or `google-cloudspanner:/projects/my_project/instances/my_instance/databases/my_database` .
- `DATASET_NAME` : the name of your new dataset in BigQuery. To create a dataset in a project other than your default project, add the project ID to the dataset name in the following format: `PROJECT_ID` : `DATASET_NAME` .
- (Optional) `CONNECTION_NAME` : the name of your Cloud resource connection.

### Terraform

Use the [`google_bigquery_dataset` resource](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_dataset#example-usage---bigquery-dataset-external-reference-aws-docs) .

> **Note:** To create BigQuery objects using Terraform, you must enable the [Cloud Resource Manager API](https://docs.cloud.google.com/resource-manager/reference/rest) .

To authenticate to BigQuery, set up Application Default Credentials. For more information, see [Set up authentication for client libraries](https://docs.cloud.google.com/bigquery/docs/authentication#client-libs) .

The following example creates a Spanner external dataset:

```terraform
resource "google_bigquery_dataset" "default" {
  dataset_id    = "my_external_dataset"
  friendly_name = "My external dataset"
  description   = "This is a test description."
  location      = "US"
  external_dataset_reference {
    # The full identifier of your Spanner database.
    external_source = "google-cloudspanner:/projects/my_project/instances/my_instance/databases/my_database"
    # Must be empty for a Spanner external dataset.
    connection = ""
  }
}
```

To apply your Terraform configuration in a Google Cloud project, complete the steps in the following sections.

## Prepare Cloud Shell

1.  Launch [Cloud Shell](https://shell.cloud.google.com/) .

2.  Set the default Google Cloud project where you want to apply your Terraform configurations.

    You only need to run this command once per project, and you can run it in any directory.

    ```
    export GOOGLE_CLOUD_PROJECT=PROJECT_ID
    ```

    Environment variables are overridden if you set explicit values in the Terraform configuration file.

## Prepare the directory

Each Terraform configuration file must have its own directory (also called a *root module* ).

1.  In [Cloud Shell](https://shell.cloud.google.com/) , create a directory and a new file within that directory. The filename must have the `.tf` extension—for example `main.tf` . In this tutorial, the file is referred to as `main.tf` .

    ```
    mkdir DIRECTORY && cd DIRECTORY && touch main.tf
    ```

2.  If you are following a tutorial, you can copy the sample code in each section or step.

    Copy the sample code into the newly created `main.tf` .

    Optionally, copy the code from GitHub. This is recommended when the Terraform snippet is part of an end-to-end solution.

3.  Review and modify the sample parameters to apply to your environment.

4.  Save your changes.

5.  Initialize Terraform. You only need to do this once per directory.

    ```
    terraform init
    ```

    Optionally, to use the latest Google provider version, include the `-upgrade` option:

    ```
    terraform init -upgrade
    ```

## Apply the changes

1.  Review the configuration and verify that the resources that Terraform is going to create or update match your expectations:

    ```
    terraform plan
    ```

    Make corrections to the configuration as necessary.

2.  Apply the Terraform configuration by running the following command and entering `yes` at the prompt:

    ```
    terraform apply
    ```

    Wait until Terraform displays the "Apply complete!" message.

3.  [Open your Google Cloud project](https://console.cloud.google.com/) to view the results. In the Google Cloud console, navigate to your resources in the UI to make sure that Terraform has created or updated them.

> **Note:** Terraform samples typically assume that the required APIs are enabled in your Google Cloud project.

### API

Call the [`datasets.insert` method](https://docs.cloud.google.com/bigquery/docs/reference/rest/v2/datasets/insert) with a defined [dataset resource](https://docs.cloud.google.com/bigquery/docs/reference/rest/v2/datasets) and [`externalDatasetReference` field](https://docs.cloud.google.com/bigquery/docs/reference/rest/v2/datasets#ExternalDatasetReference) for your Spanner database.

Note that names of the tables in the external datasets are case insensitive.

When you create the external datasets with a `CLOUD_RESOURCE` connection, you need to have the `bigquery.connections.delegate` permission (available from the BigQuery Connection Admin role) on the connection that is used by the external datasets.

### Create a non-incremental materialized view based on tables from an external dataset

Before you proceed, you must create the underlying Spanner external dataset using a [`CLOUD_RESOURCE` connection](https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets#use_a_cloud_resource_connection) .

You can create non-incremental materialized views that reference [Spanner external dataset tables](https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets) by using the `allow_non_incremental_definition` option. The following example uses a base Spanner external dataset table:

```
/*
  You must create the spanner_external_dataset with a CLOUD_RESOURCE connection.
*/
CREATE MATERIALIZED VIEW sample_dataset.sample_spanner_mv
  OPTIONS (
      enable_refresh = true, refresh_interval_minutes = 60,
      max_staleness = INTERVAL "24" HOUR,
        allow_non_incremental_definition = true)
AS
  SELECT COUNT(*) cnt FROM spanner_external_dataset.spanner_table;
```

Only BigQuery non-incremental materialized views can have [Spanner external dataset tables as base tables](https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets) . If a non-incremental materialized view's last refresh occurred outside the `max_staleness` interval, then the query reads the base Spanner external dataset tables. Learn more about BigQuery [non-incremental materialized views](https://docs.cloud.google.com/bigquery/docs/materialized-views-create#non-incremental) .

## What's next

- Learn about Data Boost in [Data Boost overview](https://docs.cloud.google.com/spanner/docs/databoost/databoost-overview)
- [Use Data Boost in your applications](https://docs.cloud.google.com/spanner/docs/databoost/databoost-applications)
- [Monitor Data Boost usage](https://docs.cloud.google.com/spanner/docs/databoost/databoost-monitor)
- [Monitor and manage Data Boost quota usage](https://docs.cloud.google.com/spanner/docs/databoost/databoost-quotas)
- Learn more about [Spanner external dataset](https://docs.cloud.google.com/bigquery/docs/spanner-external-datasets) .
- Learn more about [creating materialized views over Spanner external datasets](https://docs.cloud.google.com/bigquery/docs/materialized-views-create#spanner) .
