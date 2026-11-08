---
name: documents/docs.cloud.google.com/spanner/docs/dc-integration
uri: https://docs.cloud.google.com/spanner/docs/dc-integration
title: Manage resources using Knowledge Catalog
description: Integrate Spanner with Knowledge Catalog to discover and manage your Spanner metadata.
data_source: docs.cloud.google.com
---

This document explains how to search for and manage your Spanner resources by using Knowledge Catalog.

Knowledge Catalog catalogs metadata about Spanner instances, databases, tables, columns, and views across Google Cloud, letting AI agents and analytics tools retrieve grounded, relevant context.

## About Knowledge Catalog and Spanner

Knowledge Catalog refers to the resources in Spanner and in other Google Cloud services as *assets* . In Knowledge Catalog, each Spanner asset, such as an instance, database, table, or view, is represented by a catalog *entry* that stores technical and business metadata ( *aspects* ) about the resource.

You can use Knowledge Catalog in the following workflows to discover and understand your Spanner assets:

- **Analysis** : understand dependencies between data assets and evaluate asset suitability for specific use cases.
- **Change management** : track structural modifications across databases.
- **Data movement** : trace pipeline origins and destinations.
- **Schema evolution** : monitor table and column definitions over time.
- **Contextual AI grounding** : provide a schema and business context to AI agents.

Knowledge Catalog automatically catalogs the following metadata for Spanner resources:

- Resource name and fully qualified name (FQN)
- Location (region)
- Creation and last modification timestamps
- Schema definitions (for tables and views)
- Column data types, nullability, and primary key indicators
- Descriptions and user labels

Knowledge Catalog metadata discovery is an automated process that scans connected data sources, such as Spanner, to identify data assets like tables, views, and databases. Then, Knowledge Catalog extracts technical metadata from the assets, such as schemas, descriptions, and locations.

Spanner syncs metadata to Knowledge Catalog at regular intervals, typically every six hours, so metadata changes made on Spanner assets can take approximately six hours to propagate.

## Before you begin

Before using Knowledge Catalog with Spanner, complete the following setup tasks:

1.  Enable the Dataplex API for the project.

### Required roles

To get the permissions that you need to manage Spanner assets in Knowledge Catalog, ask your administrator to grant you the following IAM roles on your project:

- View Spanner resources: [Cloud Spanner Viewer](https://docs.cloud.google.com/iam/docs/roles-permissions/spanner#spanner.viewer) ( `roles/spanner.viewer` )
- Search and view entries in Knowledge Catalog: [Dataplex Catalog Viewer](https://docs.cloud.google.com/iam/docs/roles-permissions/dataplex#dataplex.catalogViewer) ( `roles/dataplex.catalogViewer` )
- Create and manage aspect types:
  - [Dataplex Aspect Type Owner](https://docs.cloud.google.com/iam/docs/roles-permissions/dataplex#dataplex.aspectTypeOwner) ( `roles/dataplex.aspectTypeOwner` )
  - [Dataplex Catalog Admin](https://docs.cloud.google.com/iam/docs/roles-permissions/dataplex#dataplex.catalogAdmin) ( `roles/dataplex.catalogAdmin` )
- Attach or modify aspects:
  - [Dataplex Catalog Editor](https://docs.cloud.google.com/iam/docs/roles-permissions/dataplex#dataplex.catalogEditor) ( `roles/dataplex.catalogEditor` )
  - [Dataplex Entry and EntryLink Owner](https://docs.cloud.google.com/iam/docs/roles-permissions/dataplex#dataplex.entryOwner) ( `roles/dataplex.entryOwner` )
  - [Dataplex Aspect Type User](https://docs.cloud.google.com/iam/docs/roles-permissions/dataplex#dataplex.aspectTypeUser) ( `roles/dataplex.aspectTypeUser` )

For more information about granting roles, see [Manage access to projects, folders, and organizations](https://docs.cloud.google.com/iam/docs/granting-changing-revoking-access) .

These predefined roles contain the permissions required to manage Spanner assets in Knowledge Catalog. To see the exact permissions that are required, expand the **Required permissions** section:

#### Required permissions

The following permissions are required to manage Spanner assets in Knowledge Catalog:

- Create an aspect type: `dataplex.aspectTypes.create`
- Search for Spanner instances:
  - `dataplex.projects.search`
  - `spanner.instances.get`
- Search for Spanner databases, tables, and views:
  - `dataplex.projects.search`
  - `spanner.databases.get`
- View aspects on Spanner instances:
  - `dataplex.aspectTypes.get`
  - `spanner.instances.get`
- View aspects on Spanner databases, tables, and views:
  - `dataplex.aspectTypes.get`
  - `spanner.databases.get`
- Attach an aspect to a Spanner resource using an aspect type:
  - `dataplex.aspectTypes.use`
  - `dataplex.entries.update`

You might also be able to get these permissions with [custom roles](https://docs.cloud.google.com/iam/docs/creating-custom-roles) or other [predefined roles](https://docs.cloud.google.com/iam/docs/roles-overview#predefined) .

> **Note:** Knowledge Catalog doesn't support Spanner fine-grained access control.

## Search for Spanner assets

Knowledge Catalog supports semantic search, which helps you find resources using everyday language, eliminating the need for complex search syntax. For example, enter a query such as `List all Spanner tables related to sales` .

If you prefer, you can search for resources using specific keywords, filters, and a predicate-based search syntax.

To search for Spanner assets in Knowledge Catalog, follow these steps:

1.  In the Google Cloud console, go to the Knowledge Catalog **Search** page:

2.  In the **Filters** panel, under **Systems** , select **Spanner** .

    Knowledge Catalog displays all Spanner assets that you have access to in your Google Cloud projects.

3.  Use the search field to enter a natural language query, such as `List all Spanner tables related to sales` .

4.  Optional: Use the filtered search on the **Search** page. Select filters under **Type** , **Projects** , **Locations** , and **Aspect types** .

5.  Optional: In the search field, append a search parameter after `system=spanner` . Separate multiple parameters with spaces or use parentheses and the logical operators `AND` , `OR` , and `NOT` .

    For example, to view only databases, enter the following query:

    ```
    system=spanner type=database
    ```

    You can also use `type=service` to search for instances, or `instance_config= `` CONFIGURATION_NAME` to search for an instance configuration.

    For more query options, see [Search syntax for Knowledge Catalog](https://docs.cloud.google.com/knowledge-catalog/docs/search-syntax) .

6.  In the search results list, click the name of an asset to open its entry details page in Knowledge Catalog.

From the entry details page, you can do the following:

- To view asset summaries and add a description, go to the **Details** tab, and then in the **Overview** section, click **Add** .
- To view or attach business and operational metadata, go to the **Details** tab, and then in the **Aspects** section, click **Add** next to **Optional aspects** .
- To view member databases for an instance, click the **Entry list** tab, and then click **Show all children entries in search** .
- To inspect column data types, keys, and definitions for a table, click the **Schema** tab.

### Use case: Drill down from instance to columns

In this use case, you search for a Spanner instance and then drill down to one of the databases listed in the instance to view its tables and columns.

1.  In the Google Cloud console, go to the Knowledge Catalog **Search** page:

2.  In the **Filters** panel, under **Systems** , select **Spanner** .

3.  To view all Spanner instances in Knowledge Catalog, use the search field to enter a natural language query, such as `List all Spanner instances` , and press `Enter` .

4.  Select an instance name.

5.  On the **Spanner details** page, click the **Entry list** tab, and then click **Show all children entries in search** .

    Knowledge Catalog displays the databases in the instance. If there's no **Entry list** tab, return to the **Search** page and choose a different instance.

6.  Select a database name. On the **Spanner database details** page, click the **Entry list** tab, and then click **Show all children entries in search** .

    Knowledge Catalog displays the tables in the database.

7.  Select a table name, and then on the **Spanner table details** page, click **Schema** to see the table columns.

8.  Optional: To add an aspect to a column, select the column and click **Add aspect** .

## Enrich metadata with aspects

To enrich metadata and search for assets by custom context, you can create aspects based on aspect types (reusable templates) and attach them to your Spanner assets. Aspects let you add structured business metadata, such as data ownership, governance classifications, or retention policies.

For example, you can attach the following aspect to a column that contains a social security number, which is personally identifiable information (PII):

```
pii:true
pii_type:SSN
```

To attach an aspect to a Spanner asset, follow these steps:

1.  On the Knowledge Catalog **Search** page, click the asset that you want to enrich.
2.  On the entry details page, on the **Details** tab, go to the **Aspects** section.
3.  Next to **Optional aspects** , click **Add** .
4.  Search for and select the aspect type that you want to add, such as a custom schema for data governance or privacy.
5.  In the **Add aspect** window, specify the required aspect fields.
6.  Click **Save** .

For detailed instructions on creating aspect types and managing aspects, see [Manage aspects and enrich metadata](https://docs.cloud.google.com/knowledge-catalog/docs/enrich-entries-metadata) .

## Export and import aspects

When you [move a Spanner instance](https://docs.cloud.google.com/spanner/docs/move-instance) to a different instance configuration, the process deletes the entries and aspects associated with the source instance configuration in Knowledge Catalog. To preserve your custom aspects, you need to do the following:

1.  Export the aspects associated with the entries in the source instance configuration.
2.  Move the instance to the destination instance configuration.
3.  Import the aspects into the destination instance configuration.

### Export aspects from the source instance configuration

Before you move your instance, retrieve the aspects for your Spanner entries in the source instance configuration by using the Google Cloud CLI, the Dataplex API, or a metadata export job:

### gcloud

To retrieve the aspects for a Spanner entry in the `@spanner` system entry group, use the [`gcloud dataplex entries lookup`](https://docs.cloud.google.com/sdk/gcloud/reference/dataplex/entries/lookup) command:

```
gcloud dataplex entries lookup SOURCE_ENTRY_ID \
  --entry-group=@spanner \
  --location=LOCATION \
  --project=PROJECT_ID \
  --view=all
```

Replace the following:

- `SOURCE_ENTRY_ID` : the entry ID of the Spanner resource in the source instance configuration.

  How to find the entry ID

  To find the entry name or translate a [fully qualified name (FQN)](https://docs.cloud.google.com/knowledge-catalog/docs/fully-qualified-names#predefined-spanner) such as `spanner: `` PROJECT_ID `` . `` INSTANCE_CONFIG_ID `` . `` INSTANCE_ID `` . `` DATABASE_ID `` . `` TABLE_ID` , use the [`gcloud dataplex entries search`](https://docs.cloud.google.com/sdk/gcloud/reference/dataplex/entries/search) command.

- `LOCATION` : the region of the Spanner instance configuration, such as `us-central1` .

- `PROJECT_ID` : the ID of the Google Cloud project that contains the Spanner instance.

### REST API

To retrieve the aspects for a Spanner entry in the `@spanner` system entry group, send a `GET` request to the [`projects.locations.lookupEntry`](https://docs.cloud.google.com/knowledge-catalog/docs/reference/rest/v1/projects.locations/lookupEntry) endpoint.

```
curl -X GET \
-H "Authorization: Bearer ACCESS_TOKEN" \
-H "Accept: application/json" \
"https://dataplex.googleapis.com/v1/projects/PROJECT_ID/locations/LOCATION:lookupEntry?entry=projects/PROJECT_ID/locations/LOCATION/entryGroups/@spanner/entries/SOURCE_ENTRY_ID&view=ALL"
```

Replace the following:

- `ACCESS_TOKEN` : the access token that your application uses to authenticate to the service. For example, use `$(gcloud auth print-access-token)` .

- `PROJECT_ID` : the ID of the Google Cloud project that contains the Spanner instance.

- `LOCATION` : the region of the Spanner instance configuration, such as `us-central1` .

- `SOURCE_ENTRY_ID` : the entry ID of the Spanner resource in the source instance configuration.

  How to find the entry ID

  To find the entry name or translate a [fully qualified name (FQN)](https://docs.cloud.google.com/knowledge-catalog/docs/fully-qualified-names#predefined-spanner) such as `spanner: `` PROJECT_ID `` . `` INSTANCE_CONFIG_ID `` . `` INSTANCE_ID `` . `` DATABASE_ID `` . `` TABLE_ID` , use the [`gcloud dataplex entries search`](https://docs.cloud.google.com/sdk/gcloud/reference/dataplex/entries/search) command.

### Export job

To export metadata for multiple entries in a batch to Cloud Storage, run a [metadata export job](https://docs.cloud.google.com/knowledge-catalog/docs/export-metadata) .

### Move the instance to the destination instance configuration

Before you import aspects into the destination instance configuration, do the following:

1.  Move the Spanner instance to the destination instance configuration. Wait for Spanner metadata for the moved instance to sync to Knowledge Catalog, which can take approximately six hours.

2.  To verify that the sync is complete, search for the moved instance using one of the following methods:

    - On the Knowledge Catalog **Search** page, enter the following query:

      ```
      system=spanner instance_config=DESTINATION_CONFIG_NAME
      ```

    - Run the [`gcloud dataplex entries search`](https://docs.cloud.google.com/sdk/gcloud/reference/dataplex/entries/search) command:

      ```
      gcloud dataplex entries search "fqn=spanner:PROJECT_ID.DESTINATION_CONFIG_ID.INSTANCE_ID" \
          --project=PROJECT_ID
      ```

    When the command returns the entry for the destination instance configuration, the sync is complete.

### Import aspects into the destination instance configuration

After the metadata syncs, attach the copied aspects to the entries in the destination instance configuration:

### gcloud

To attach the copied aspects to the entry in the destination instance configuration, use the [`gcloud dataplex entries update-aspects`](https://docs.cloud.google.com/sdk/gcloud/reference/dataplex/entries/update-aspects) command:

```
gcloud dataplex entries update-aspects DESTINATION_ENTRY_ID \
  --entry-group=@spanner \
  --location=DESTINATION_LOCATION \
  --project=PROJECT_ID \
  --aspects=ASPECTS_FILE
```

Replace the following:

- `DESTINATION_ENTRY_ID` : the entry ID of the moved Spanner resource in the destination instance configuration.

- `DESTINATION_LOCATION` : the region of the destination Spanner instance configuration, such as `us-central1` .

- `PROJECT_ID` : the ID of the Google Cloud project that contains the Spanner instance.

- `ASPECTS_FILE` : the path to a JSON or YAML file that contains the aspects to add or update.

### REST API

To attach the exported aspects to the entry in the destination instance configuration, use the [`projects.locations.entryGroups.entries.patch`](https://docs.cloud.google.com/knowledge-catalog/docs/reference/rest/v1/projects.locations.entryGroups.entries/patch) REST API method, as shown in the following example:

```
curl -X PATCH \
-H "Authorization: Bearer ACCESS_TOKEN" \
-H "Accept: application/json" \
-H "Content-Type: application/json" \
-d '{"aspects":{"ASPECT_KEY":{"data":{"FIELD_NAME":"FIELD_VALUE"}}}}' \
"https://dataplex.googleapis.com/v1/projects/PROJECT_ID/locations/DESTINATION_LOCATION/entryGroups/@spanner/entries/DESTINATION_ENTRY_ID?updateMask=aspects"
```

Replace the following:

- `ACCESS_TOKEN` : the access token that your application uses to authenticate to the service. For example, use `$(gcloud auth print-access-token)` .

- `ASPECT_KEY` : the aspect key in the format `PROJECT_ID `` . `` ASPECT_LOCATION `` . `` ASPECT_TYPE_ID` for an entry-level aspect, or `PROJECT_ID `` . `` ASPECT_LOCATION `` . `` ASPECT_TYPE_ID `` @Schema. `` COLUMN_NAME` for a column-level aspect.

- `FIELD_NAME` and `FIELD_VALUE` : the field name and value defined by your aspect type.

- `PROJECT_ID` : the ID of the project that contains the Spanner instance.

- `DESTINATION_LOCATION` : the region of the destination Spanner instance configuration, such as `us-central1` .

- `DESTINATION_ENTRY_ID` : the entry ID of the moved Spanner resource in the destination instance configuration.

### Import job

To import aspects for multiple entries in batch from Cloud Storage, run a [metadata import job](https://docs.cloud.google.com/knowledge-catalog/docs/import-metadata) .

## What's next

- Learn more about [Knowledge Catalog](https://docs.cloud.google.com/knowledge-catalog/docs/introduction) .
- Learn more about [metadata management in Knowledge Catalog](https://docs.cloud.google.com/knowledge-catalog/docs/metadata-overview) .
- Learn how to [search for data assets in Knowledge Catalog](https://docs.cloud.google.com/knowledge-catalog/docs/search-assets) .
- Learn how to [manage aspects and enrich metadata](https://docs.cloud.google.com/knowledge-catalog/docs/enrich-entries-metadata) .
- Learn more about [Knowledge Catalog IAM roles](https://docs.cloud.google.com/knowledge-catalog/docs/iam-roles) and [permissions](https://docs.cloud.google.com/knowledge-catalog/docs/iam-permissions) .
- Explore [Fine-grained access control in Spanner](https://docs.cloud.google.com/spanner/docs/fgac-about) .
