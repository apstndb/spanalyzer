---
name: documents/docs.cloud.google.com/spanner/docs/databases
uri: https://docs.cloud.google.com/spanner/docs/databases
title: Databases overview
description: Understand databases within a Spanner instance, including how to create, manage, and interact with them.
data_source: docs.cloud.google.com
---

This page describes Spanner databases and their properties.

## Overview

A Spanner database is a container for your tables, views, and indexes. Each Spanner instance can have multiple databases.

In the Spanner hierarchy, a database is a child of an [instance](https://docs.cloud.google.com/spanner/docs/instances) , and the parent of a [schema](https://docs.cloud.google.com/spanner/docs/schema-and-data-model) .

A database inherits properties from its parent instance, such as its configuration (regional or multi-region), the available [compute capacity](https://docs.cloud.google.com/spanner/docs/compute-capacity) , and the available storage.

Properties that are set on the database are:

- The [dialect](https://docs.cloud.google.com/spanner/docs/postgresql-interface#choosing_between_and) . Spanner supports GoogleSQL and PostgreSQL. All of the tables in a database must use the same dialect.
- An [IAM policy](https://docs.cloud.google.com/spanner/docs/grant-permissions#database-level_permissions) . The access rules defined by this policy are applied to the tables and data inside the database.
- The type of encryption key to use. The default is a Google-owned and Google-managed encryption key; a [customer-managed encryption key](https://docs.cloud.google.com/spanner/docs/cmek) is also supported.
- [Database deletion protection](https://docs.cloud.google.com/spanner/docs/prevent-database-deletion) to prevent accidental deletion of databases.
- Database policies such as the default leader region, query optimizer version, query optimizer statistics package version, and version retention period, can be set or modified with DDL statements. See the DDL reference for [GoogleSQL](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter-database) or [PostgreSQL](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#alter-database) .

Database limits, including the number of databases allowed per instance, and the number of tables allowed per database, are defined on the [Quotas and limits](https://docs.cloud.google.com/spanner/quotas) page.

## Considerations

Before creating a database, consider the implications of your schema design. Read [Schema design best practices](https://docs.cloud.google.com/spanner/docs/schema-design) , as well as the [Schema and data model](https://docs.cloud.google.com/spanner/docs/schema-and-data-model) document, particularly the section on [Designing for multi-tenancy](https://docs.cloud.google.com/spanner/docs/schema-and-data-model#multitenancy) .

## What's next

- [Create and manage databases](https://docs.cloud.google.com/spanner/docs/create-manage-databases)
