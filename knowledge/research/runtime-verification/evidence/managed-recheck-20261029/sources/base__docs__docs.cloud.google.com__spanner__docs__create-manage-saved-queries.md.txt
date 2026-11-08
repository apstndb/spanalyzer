---
name: documents/docs.cloud.google.com/spanner/docs/create-manage-saved-queries
uri: https://docs.cloud.google.com/spanner/docs/create-manage-saved-queries
title: Create and manage saved queries
description: Manage saved queries in Spanner to streamline your workflow and improve collaboration.
data_source: docs.cloud.google.com
---

> **Preview**
>
> This feature is subject to the "Pre-GA Offerings Terms" in the General Service Terms section of the [Service Specific Terms](https://docs.cloud.google.com/terms/service-terms#1) . Pre-GA features are available "as is" and might have limited support. For more information, see the [launch stage descriptions](https://cloud.google.com/products/#product-launch-stages) .

This page describes how to create and manage saved queries. When you write SQL scripts in Spanner Studio, you can save and manage those SQL scripts. For more information, see [Saved queries overview](https://docs.cloud.google.com/spanner/docs/saved-queries) .

The saved queries capability is available only in the console, and not using APIs.

## Required roles

To get the permissions that you need to use saved queries, ask your administrator to grant you the following Identity and Access Management (IAM) roles on the project:

- To create, edit, and delete saved queries: [Studio Query User](https://docs.cloud.google.com/iam/docs/roles-permissions/databasesconsole) ( `roles/databasesconsole.studioQueryUser` )
- To manage all saved queries in a project, including access to the **Saved queries** page: [Studio Query Admin](https://docs.cloud.google.com/iam/docs/roles-permissions/databasesconsole) ( `roles/databasesconsole.studioQueryAdmin` )

For more information about granting roles, see [Manage access to projects, folders, and organizations](https://docs.cloud.google.com/iam/docs/granting-changing-revoking-access) .

These predefined roles contain the permissions required to use saved queries. To learn more about required permissions, see [Required permissions](https://docs.cloud.google.com/spanner/docs/create-manage-saved-queries#required-permissions) .

### Required permissions

To create, view, modify, and delete saved queries, you need the following IAM permissions at the project level:

<table>
<colgroup>
<col style="width: 50%" />
<col style="width: 50%" />
</colgroup>
<thead>
<tr class="header">
<th><strong>Action</strong></th>
<th><strong>Required IAM permissions</strong></th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>Create saved queries</td>
<td><ul>
<li><code>databasesconsole.studioQueries.create</code></li>
</ul></td>
</tr>
<tr class="even">
<td>View saved queries</td>
<td><ul>
<li><code>databasesconsole.studioQueries.search</code></li>
<li><code>databasesconsole.locations.get</code></li>
<li><code>databasesconsole.locations.list</code></li>
</ul></td>
</tr>
<tr class="odd">
<td>Modify saved queries</td>
<td><ul>
<li><code>databasesconsole.studioQueries.update</code></li>
</ul></td>
</tr>
<tr class="even">
<td>Delete saved queries</td>
<td><ul>
<li><code>databasesconsole.studioQueries.delete</code></li>
</ul></td>
</tr>
<tr class="odd">
<td>Administer saved queries<br />
(Only for use by administrators)</td>
<td><ul>
<li><code>databasesConsole.studioQueries.list</code></li>
</ul>
<ul>
<li><code>databasesconsole.locations.list</code></li>
</ul></td>
</tr>
</tbody>
</table>

> **Note:** Having access to saved queries doesn't automatically mean that you have permissions to run the query.

You can also get these permissions using [custom roles](https://docs.cloud.google.com/iam/docs/creating-custom-roles) or other [predefined roles](https://docs.cloud.google.com/iam/docs/roles-overview#predefined) .

The [Studio Query User](https://docs.cloud.google.com/iam/docs/roles-permissions/databasesconsole) ( `roles/databasesconsole.studioQueryUser` ) role provides create, view, and write saved queries access. This role meets the needs of most users.

Grant the [Studio Query Admin](https://docs.cloud.google.com/iam/docs/roles-permissions/databasesconsole) ( `roles/databasesconsole.studioQueryAdmin` ) role only to administrators. For more information about Spanner IAM, see [IAM overview](https://docs.cloud.google.com/spanner/docs/iam) .

## Create a saved query

To create a saved query, follow these steps:

1.  Go to the Spanner **Instances** page in the Google Cloud console.  
2.  Select the instance in which you want to create a saved query.
3.  Select the database in which you want to create a saved query.
4.  In the navigation menu, click **Spanner Studio** .
5.  On the **Spanner Studio** page, click add **New tab** or use the empty editor tab.
6.  In the query editor, enter a SQL query.
7.  Click **Save** .
8.  In the **Save query** dialog, enter a name for the saved query.
9.  Click **Save** .

## Open and run a saved query

To open a saved query as a new query, follow these steps:

1.  In the **Explorer** pane on the **Spanner Studio** page, navigate to the **Queries** section.
2.  Click a saved query to open it in a new editor tab. Optionally, you can click more_vert **View actions** next to a saved query. Then, click **Open query** to open it in a new editor tab.
3.  Click **Run** .

## Update a saved query

To update an existing saved query, follow these steps:

1.  In the **Explorer** pane on the **Spanner Studio** page, navigate to the **Queries** section.
2.  Click a saved query to open it in a new editor tab. Optionally, you can click more_vert **View actions** next to a saved query. Then, click **Open query** to open it in a new editor tab.
3.  Modify the query.
4.  To save the modified query, click **Save** .

## View and manage a list of all saved queries

To view a list of all saved queries in your project, follow these steps:

1.  Go to the Spanner **Instances** page in the Google Cloud console.  
2.  Select any instance. Because a saved query is a child of a project, as long as you have the required role, you can view all saved queries in the project from any instance or database.

<!-- -->

3.  Select any database.

4.  In the **Explorer** pane on the **Spanner Studio** page, navigate to the **Queries** section.

5.  Click more_vert **View actions** next to a saved query. Then, click **Manage queries** .

    The **Saved queries** page opens. This page lists all the saved queries in this project, including saved queries for other Google Cloud products.

You can search, filter, view, and delete queries on the **Saved queries** page. You can't edit an existing query on the **Saved queries** page.

## Delete a saved query

You can delete a saved query from the **Spanner Studio** page or on the **Saved queries** page in the Google Cloud console.

1.  In the **Explorer** pane on the **Spanner Studio** page, navigate to the **Queries** section.
2.  Click more_vert **View actions** next to the saved query that you want to delete. Then, to delete the saved query, click **Delete query** .
3.  In the **Delete query** dialog, click **Delete** .

You can also delete a saved query on the **Saved queries** page. To delete a saved query on the **Saved queries** page, follow these steps:

1.  Navigate to the [**Saved queries**](https://docs.cloud.google.com/spanner/docs/create-manage-saved-queries#view-and-manage) page.
2.  Click more_vert **View actions** next to the saved query that you want to delete.
3.  Click **Delete query** to delete the saved query.
4.  In the **Delete query** dialog, click **Delete** .

## What's next

- Learn about [saved queries](https://docs.cloud.google.com/spanner/docs/saved-queries) .
