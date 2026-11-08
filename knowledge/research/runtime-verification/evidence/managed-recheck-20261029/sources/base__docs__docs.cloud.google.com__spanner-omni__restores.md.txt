---
name: documents/docs.cloud.google.com/spanner-omni/restores
uri: https://docs.cloud.google.com/spanner-omni/restores
title: Restore a Spanner Omni backup
description: A downloadable, self-managed version of Spanner.
data_source: docs.cloud.google.com
---

You can restore a backup of a Spanner Omni database into a new database. The restored database has all the data and schema information from the original database at the `versionTime` of the backup, including all database options you set with the `ALTER DATABASE SET OPTIONS` command.

The following items aren't included in a restored database:

- IAM permissions. You must apply appropriate IAM permissions after the restore completes.

- Internal data of any change streams.

- Time to live (TTL) defined by a row deletion policy. You must reconfigure these policies after the restore completes.

- Split points you create when pre-splitting a database.

## How restoration works

When you restore a Spanner Omni database, you must specify a source backup and a new target database. You cannot restore to an existing database.

The restore process provides high availability. You can restore the database provided that the majority quorum of the regions and zones in the target is available.

## Restoration states

A restored database transitions through three states, tracked by two long-running operations.

<table>
<colgroup>
<col style="width: 50%" />
<col style="width: 50%" />
</colgroup>
<thead>
<tr class="header">
<th>State</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td><code>CREATING</code></td>
<td>Spanner Omni begins restoring by creating a new database and mounting files from the backup. During this state, the restored database is not yet ready for use. Once complete, your database is ready to use.<br />
<br />
<strong>Note:</strong> Spanner Omni doesn't allow you to delete the backup while it restores. You can delete it after the restore completes and the database enters the <code>READY</code> state.</td>
</tr>
<tr class="even">
<td><code>READY_OPTIMIZING</code></td>
<td>After mounting the backup, Spanner Omni starts copying data into the new database while optimizing its stored size. Your database is ready for use during this process.<br />
<br />
<strong>Caveats:</strong>
<ul>
<li>Read latencies might be slightly higher than usual.</li>
<li>Storage metrics display the size of the new database, not the backup. Results might not reflect the total size of all your data during transfer.</li>
<li>You cannot delete the mounted backup during this state.</li>
</ul></td>
</tr>
<tr class="odd">
<td><code>READY</code></td>
<td>Once the copy-and-optimize operation completes, the database transitions to the <code>READY</code> state. The database is fully restored and no longer requires the backup.</td>
</tr>
</tbody>
</table>

To track progress during the `CREATING` state, you can query the long-running restore operation, which returns a `RestoreDatabaseMetadata` object. During the `READY_OPTIMIZING` state, the operation returns an `OptimizeRestoredDatabaseMetadata` object.

## Access control (IAM)

The following roles provide the permissions required for Spanner Omni restore operations:

| IAM role                     | Permissions                                                                             |
|------------------------------|-----------------------------------------------------------------------------------------|
| `roles/spanner.restoreAdmin` | Permission to restore from a backup.                                                    |
| `roles/spanner.admin`        | Full access to restore operations and all other Spanner Omni resources.                 |
| `owner`                      | Full access to restore operations.                                                      |
| `editor`                     | Full access to restore operations.                                                      |
| `viewer`                     | Permission to view restore operations. Cannot create, update, delete, or copy a backup. |

## Restore a database from a backup

To restore a database, use the `spanner databases restore` command:

```
spanner databases restore \
  --destination-database=RESTORE_DATABASE_NAME \
  --source-backup=BACKUP_NAME \
  --async
```
