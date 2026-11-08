---
name: documents/docs.cloud.google.com/spanner/docs/fgac-models
uri: https://docs.cloud.google.com/spanner/docs/fgac-models
title: Fine-grained access control for models
description: Control access to Spanner models using FGAC and the EXECUTE privilege for machine learning functions.
data_source: docs.cloud.google.com
---

This page explains how [fine-grained access control](https://docs.cloud.google.com/spanner/docs/fgac-about) works with Spanner models.

For fine-grained access control users, you can control access to MODEL entities with the following privilege:

- Grant `EXECUTE` on the model to allow [machine learning functions](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/ml-functions) to use it.

  ### GoogleSQL

  ```
  GRANT EXECUTE ON MODEL MODEL_NAME TO ROLE ROLE_NAME;
  ```

## `INFORMATION_SCHEMA` views for models

The following views show the database roles and privileges information for models:

- GoogleSQL-dialect databases: [`INFORMATION_SCHEMA.MODEL_PRIVILEGES`](https://docs.cloud.google.com/spanner/docs/information-schema#model-privileges)

The rows in this view are filtered based on the current database role's privileges on models. This ensures that principals can view only the roles, privileges, and models that they have access to.

Row filtering also applies to the following model-related views:

### GoogleSQL

- [`INFORMATION_SCHEMA.MODELS`](https://docs.cloud.google.com/spanner/docs/information-schema#models)
- [`INFORMATION_SCHEMA.MODEL_OPTIONS`](https://docs.cloud.google.com/spanner/docs/information-schema#model-options)
- [`INFORMATION_SCHEMA.MODEL_COLUMNS`](https://docs.cloud.google.com/spanner/docs/information-schema#model-columns)
- [`INFORMATION_SCHEMA.MODEL_COLUMN_OPTIONS`](https://docs.cloud.google.com/spanner/docs/information-schema#model-column-options)

The system role `spanner_info_reader` and its members always see an unfiltered `INFORMATION_SCHEMA` .

## More information

- [About models](https://docs.cloud.google.com/spanner/docs/ml)
- [Create and manage models](https://docs.cloud.google.com/spanner/docs/ml-tutorial)
- [About fine-grained access control](https://docs.cloud.google.com/spanner/docs/fgac-about)
