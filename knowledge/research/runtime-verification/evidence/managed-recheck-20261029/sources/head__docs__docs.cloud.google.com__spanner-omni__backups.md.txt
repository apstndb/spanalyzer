---
name: documents/docs.cloud.google.com/spanner-omni/backups
uri: https://docs.cloud.google.com/spanner-omni/backups
title: Spanner Omni backups
description: A downloadable, self-managed version of Spanner.
data_source: docs.cloud.google.com
---

Spanner Omni lets you create full backups of databases on demand or by using a backup schedule. Backups store the entire data of a database. You can restore the backups when operator or application errors cause logical data corruption.

## Spanner Omni backup overview

Backups are highly available and can be retained for up to a year from their creation time. Each backup has an associated `createTime` and `versionTime` . The `createTime` is the timestamp when Spanner Omni starts creating the backup. The `versionTime` is the timestamp when the backup captures the database contents. The backup contains a consistent view of the database at the `versionTime` .

For on-demand backups, the `createTime` and `versionTime` are the same by default. If needed, you can specify an older `versionTime` when creating an on-demand backup if it's within the version retention period of the database.

For scheduled backups, the `versionTime` is the time you choose when you create the backup schedule. Spanner Omni starts creating the backup within four hours of the `versionTime` , so the `createTime` falls within this four-hour window. This is unlike on-demand backups, where Spanner Omni starts creating the backup when it receives the request.

For example, suppose you create a backup schedule with a frequency of `0 7 * * * UTC` (every day at 7:00 AM UTC). This means that for each backup, the `versionTime` is 7:00 AM UTC and the `createTime` is a timestamp within the four-hour window between 7:00 AM UTC and 11:00 AM UTC.

Spanner Omni backups can be restored only into Spanner Omni deployments. You can't restore a managed Spanner backup into Spanner Omni, or restore a Spanner Omni backup into managed Spanner. To transfer data between Spanner and Spanner Omni, use change streams or export and import Avro files. For more information, see [Replicate data from Spanner to Spanner Omni](https://docs.cloud.google.com/spanner-omni/replicate-data) and [Import and export data](https://docs.cloud.google.com/spanner-omni/import-export-data) .

## Key features

Spanner Omni backups provide data consistency, resilient external replication, and automated expiration.

- **Data consistency** : backups of a Spanner Omni database are transactionally and externally consistent at the `versionTime` of the backup.

- **Replication** : backup files are stored in an external storage system, outside of the Spanner Omni deployment.

- **Automatic expiration** : all backups have a user-specified expiration date that determines when it's deleted. Spanner Omni deletes expired backups asynchronously, so there might be a lag between when a backup expires and when it's actually deleted.

## External storage

External storage represents remote storage that is outside of the Spanner Omni deployment. You can configure Amazon Simple Storage Service (Amazon S3), Cloud Storage, or any Amazon S3-compatible storage as external storage. Spanner Omni stores the backup files in this external storage.

### External storage management

Manage external storage by creating, deleting, and listing storage locations for your backups.

#### Create external storage

To create Amazon S3 external storage, run the following command:

```
spanner external-storages create EXTERNAL_STORAGE_ID \
  --s3-bucket-name=BUCKET_NAME \
  --s3-region=AWS_REGION \
  --s3-assume-role-arn=ASSUME_ROLE_ARN
```

#### Create Cloud Storage external storage

> **Note:** Each deployment can have only one external storage, and Spanner Omni stores all of the deployment's backups in it. To use a different Cloud Storage bucket, you must first [delete the existing external storage](https://docs.cloud.google.com/spanner-omni/backups#delete-external-storage) .

Before you configure Cloud Storage external storage, ensure that you have these items:

- A target Cloud Storage bucket in your Google Cloud project.
- A Google Cloud service account granted the Storage Object Admin ( `roles/storage.objectAdmin` ) Identity and Access Management (IAM) role on the target bucket. For more information, see [IAM roles for Cloud Storage](https://docs.cloud.google.com/storage/docs/access-control/iam-roles) .
- Authentication credentials configured in your Spanner Omni environment, such as a service account key file or credentials exchanged through Workload Identity Federation. For more information about authenticating from external environments, see [Configure Workload Identity Federation with AWS VMs](https://docs.cloud.google.com/iam/docs/workload-identity-federation-with-other-clouds) and [Configure Workload Identity Federation with Kubernetes](https://docs.cloud.google.com/iam/docs/workload-identity-federation-with-kubernetes) .

To register your Cloud Storage bucket as external storage, run this command:

```
spanner external-storages create EXTERNAL_STORAGE_ID \
  --gcs-bucket-name=BUCKET_NAME
```

Replace the following:

- `EXTERNAL_STORAGE_ID` : a unique identifier for the external storage configuration, for example, `gcp-gcs-backup-target` .
- `BUCKET_NAME` : the name of your target Cloud Storage bucket.

#### Create Amazon S3-compatible external storage

To create Amazon S3-compatible external storage, run the following command:

```
spanner external-storages create EXTERNAL_STORAGE_ID \
    --s3-compatible-bucket-name=BUCKET_NAME \
    --s3-compatible-endpoint=ENDPOINT \
    --s3-compatible-credential-file-path=FILE
```

#### Delete external storage

To delete external storage, first ensure no existing or ongoing backups are present. Then, run the following command:

```
spanner external-storages delete EXTERNAL_STORAGE_ID
```

#### Describe external storage

To get information about external storage, run the following command:

```
spanner external-storages describe EXTERNAL_STORAGE_ID
```

#### List external storage

To get a list of external storage, run the following command:

```
spanner external-storages list
```

#### Backup descriptor

A backup descriptor represents the metadata and backup file paths of completed backups stored in an external storage.

```
spanner external-storages backup-descriptors list EXTERNAL_STORAGE_ID
```

## Backup information

When you create a backup, the backup metadata is stored in Spanner Omni and the backup files reside in the external storage.

A backup contains the following information from the database at the `versionTime` of the backup:

- A full backup contains all of the data.

- Schema information, including table names, fields, data types, secondary indexes, change streams, and the relationships between these entities.

- All database options that are set with the `ALTER DATABASE SET OPTIONS` command.

A Spanner Omni backup doesn't include the following information:

- Any modifications to the data or schema after the `versionTime` .

- IAM policies.

- Change stream data records. Although the change streams schema is stored, the change stream data should be streamed and consumed at about the same time as the changes it describes.

To help ensure external consistency of the backup, Spanner Omni pins the contents of the database at `versionTime` . This prevents the garbage collection system from removing the relevant data values for the duration of the backup operation.

### Backup management

To create backups, you need the following permissions. Ask your administrator to grant you the following IAM roles on a deployment:

| Action                                   | IAM Role                     |
|------------------------------------------|------------------------------|
| Create, view, update, and delete backups | `roles/spanner.backupAdmin`  |
| Create and view backups                  | `roles/spanner.backupWriter` |

#### Create backup

Create an on-demand backup.

```
spanner backups create BACKUP_NAME \
--database=DATABASE_ID \
--retention-period=RETENTION_PERIOD \
--async
```

#### Delete backup

Delete the backup metadata and files.

```
spanner backups delete BACKUP_NAME
```

#### Describe backup

Retrieve information about a backup.

```
spanner backups describe BACKUP_NAME
```

#### List backups

List existing Spanner Omni backups in the deployment.

```
spanner backups list
```

#### Update backup expiration

Update a backup's expiration date.

```
spanner backups update-metadata BACKUP_NAME \
--expiration-date=EXPIRATION_DATE
```

#### Import backup

To recover a database in another Spanner Omni deployment—for example, to recover between Spanner Omni deployments or if you delete the original deployment—you can import backups if the backup files remain in external storage:

> **Caution:** Importing a backup doesn't copy the backup files. The imported backup and the source backup reference the same files in external storage. When either backup is deleted or expires, Spanner Omni deletes the files, and the other backup can no longer be restored. If the original deployment still exists, don't delete the source backup or let it expire until the database that you restore in the new deployment is in the `READY` state.

1.  Create external storage in the new deployment that uses the same Amazon S3 or Cloud Storage bucket as the original deployment, for example:

    ```
    spanner external-storages create EXTERNAL_STORAGE_ID \
      --gcs-bucket-name=BUCKET_NAME
    ```

2.  List the backup descriptors in the external storage:

    ```
    spanner external-storages backup-descriptors list EXTERNAL_STORAGE_ID
    ```

3.  Import the backup metadata into the new deployment:

    ```
    spanner backups import BACKUP_NAME \
      --external-storage=EXTERNAL_STORAGE_ID \
      --backup-descriptor=BACKUP_DESCRIPTOR \
      --retention-period=RETENTION_PERIOD
    ```

    Replace the following:

    - `BACKUP_NAME` : a name for the imported backup.
    - `EXTERNAL_STORAGE_ID` : the ID of your external storage configuration.
    - `BACKUP_DESCRIPTOR` : the descriptor ID returned in the previous step.
    - `RETENTION_PERIOD` : the duration to retain the backup, for example, `24h` or `30d` .

After you import the backup metadata, you can restore the backup into a new database in the deployment. For more information, see [Restore a Spanner Omni backup](https://docs.cloud.google.com/spanner-omni/restores) .

## Backup schedules

Spanner Omni lets you schedule full backups for databases. You can specify how often Spanner Omni creates backups in the schedule.

A full backup schedule creates backups every 12 hours or more. Backups start within a 30-minute window of the scheduled time. You can have a maximum of four backup schedules per database.

> **Note:** Backup schedules for newly created databases take up to 24 hours to become active and start creating backups.

### Backup schedule management

To create and manage backup schedules, you need the following permissions. Ask your administrator to grant you the following IAM roles on the deployment:

- Create, view, update, and delete backup schedules: `roles/spanner.backupAdmin`

- Create and view backup schedules: `roles/spanner.backupWriter`

#### Create backup schedule

Create a new backup schedule for the Spanner Omni database.

```
spanner backup-schedules create SCHEDULE_ID \
  --database=DATABASE_ID \
  --retention-duration=RETENTION_DURATION \
  --cron="CRONTAB_EXPRESSION"
```

#### Get a backup schedule

Get information about a specific backup schedule.

```
spanner backup-schedules describe SCHEDULE_ID --database=DATABASE_ID
```

#### List backup schedules

List all backup schedules for a given database.

```
spanner backup-schedules list --database=DATABASE_ID
```

#### Update a backup schedule

Update the properties of an existing backup schedule.

```
spanner backup-schedules update SCHEDULE_ID \
  --database=DATABASE_ID \
  --retention-duration=RETENTION_DURATION \
  --cron="CRONTAB_EXPRESSION"
```

#### Delete a backup schedule

Delete a backup schedule from the database.

```
spanner backup-schedules delete SCHEDULE_ID --database=DATABASE_ID
```

#### Set IAM access control policy

Set the IAM access control policy for a backup schedule.

```
spanner backup-schedules set-iam-policy SCHEDULE_ID \
  --database=DATABASE_ID \
  policy.json
```

#### Get IAM access control policy

Get the IAM access control policy for a backup schedule.

```
spanner backup-schedules get-iam-policy SCHEDULE_ID --database=DATABASE_ID
```

## Comparing backup storage to database storage

The storage size of a backup can be smaller or larger than the source database's storage size at the time of creation.

A backup's storage can be smaller than the database's storage because a backup contains only one version of the data, while the live database can contain multiple versions due to ongoing operations. Differences in data format and compression can also result in a smaller backup size.

Conversely, a backup's storage can be larger than the database's storage, depending on the database's state and when you create the backup.

For similar reasons, when you restore a database from a backup, its storage can be larger than the database's storage. This might occur if a large set of data was deleted and compacted after the backup was created. Therefore, the backup size depends on when the backup was created and the subsequent operations on the database.

No formula predicts a backup's size relative to the live database. For a database with a high rate of writes (a *hot database* ), the backup is likely smaller than the live database. However, in some situations, the backup size can be larger.

## What's next

- [Restore a Spanner Omni backup](https://docs.cloud.google.com/spanner-omni/restores)
- [Replicate data from Spanner to Spanner Omni](https://docs.cloud.google.com/spanner-omni/replicate-data)
