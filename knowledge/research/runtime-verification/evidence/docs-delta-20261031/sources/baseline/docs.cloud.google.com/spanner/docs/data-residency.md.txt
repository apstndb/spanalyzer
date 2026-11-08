---
name: documents/docs.cloud.google.com/spanner/docs/data-residency
uri: https://docs.cloud.google.com/spanner/docs/data-residency
title: Data residency overview
description: Configure data residency in Spanner to ensure your data is stored and processed within the geographic areas you specify.
data_source: docs.cloud.google.com
---

This page describes data residency for Spanner.

Spanner meets data residency compliance and regulatory requirements by letting you to specify the geographic locations (regions) where Spanner data is stored.

The following definitions apply to this page:

- A *location* is a Google Cloud region or multi-region as listed on the [Google Cloud locations page](https://cloud.google.com/about/locations) .

- The term *your data* is equivalent to the meaning of the term "Customer Data" in the *Data Location* item in the Google Cloud [General Service Terms](https://cloud.google.com/terms/service-terms) .

## Data residency commitments

Data residency commitments in Spanner differ for databases that don't use [geo-partitioning](https://docs.cloud.google.com/spanner/docs/geo-partitioning) versus databases that do use geo-partitioning.

### Databases that don't use geo-partitioning

For databases that don't use geo-partitioning, Spanner provides data residency commitments at the database level according to the [Google Cloud Terms of Service](https://cloud.google.com/terms) .

### Databases that use geo-partitioning

For databases that use geo-partitioning, Spanner provides data residency commitments at the [placement](https://docs.cloud.google.com/spanner/docs/create-manage-data-placements) level. For each placement, you can select a specific region or multi-region location as listed on the [Google Cloud locations page](https://cloud.google.com/about/locations) . Spanner stores your data at rest only within the selected region or multi-region with the following limitations:

- A small subset of primary keys, indexed column values, and foreign key column values (for both placement and non-placement tables) are used as [split boundaries](https://docs.cloud.google.com/spanner/docs/schema-and-data-model#database-splits) , which might be stored in the default placement.
- Statistics and observability information for key ranges used for the [key visualizer](https://docs.cloud.google.com/spanner/docs/key-visualizer/getting-started) , keys experiencing high [lock contention](https://docs.cloud.google.com/spanner/docs/introspection/lock-statistics) , and [query statistics](https://docs.cloud.google.com/spanner/docs/introspection/query-statistics) are stored in the default placement.
- Column-level [statistics](https://docs.cloud.google.com/spanner/docs/query-optimizer/overview#statistics-packages) used for query optimization are stored in the default placement.

The following are by design:

- Placement table primary keys are used for routing traffic and might be stored in the default placement. If this is a concern, consider using [UUIDs](https://docs.cloud.google.com/spanner/docs/primary-key-default-value#universally_unique_identifier_uuid) , or other keys that aren't in scope for data residency.
- [Interleaved indexes](https://docs.cloud.google.com/spanner/docs/secondary-indexes#indexes_and_interleaving) inherit placement from the parent row. Global indexes (including keys and storing values) are placed in the default placement.
- [Foreign keys backing indexes](https://docs.cloud.google.com/spanner/docs/foreign-keys/overview#backing-indexes) are placed in the default placement.
- If you change the placement key for a row, the data move happens asynchronously. It might take hours to move the row to the new location. Even after the data is available and served from the new location, deletion of data from the old location is subject to the [Google Cloud data deletion process](https://docs.cloud.google.com/docs/security/deletion) .

## Data residency encryption

By default, Spanner [encrypts customer data at rest](https://docs.cloud.google.com/docs/security/encryption/default-encryption) . Spanner handles encryption for you without any additional actions on your part. This option is called *Google default encryption* . By default, Google uses encryption keys from the same location as where your data resides.

If you want to control your encryption keys, then you can use *customer-managed encryption keys (CMEKs)* in [Cloud KMS](https://docs.cloud.google.com/kms/docs) with CMEK-integrated services including Spanner. When using CMEK, you must select keys in the same location as where your data resides. For more information, see [Customer-managed encryption keys (CMEK) overview](https://docs.cloud.google.com/spanner/docs/cmek) .
