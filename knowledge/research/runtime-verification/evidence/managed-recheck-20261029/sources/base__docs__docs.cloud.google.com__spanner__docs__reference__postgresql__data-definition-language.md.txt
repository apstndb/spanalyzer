---
name: documents/docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language
uri: https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language
title: PostgreSQL data definition language
description: Learn the DDL syntax for PostgreSQL in Spanner, including CREATE, ALTER, and DROP statements for database objects.
data_source: docs.cloud.google.com
---

This page defines the syntax of the SQL data definition language (DDL) statements supported for PostgreSQL-dialect databases.

## Notations used in the syntax

- Square brackets `[ ]` indicate optional clauses.
- Curly braces `{ }` enclose a set of options.
- The vertical bar `|` indicates a logical OR.
- A comma followed by an ellipsis indicates that the preceding `item` can repeat in a comma-separated list.
  - `item [, ...]` indicates one or more items, and
  - `[item, ...]` indicates zero or more items.
- Purple-colored text, such as `item` , marks Spanner extensions to open source PostgreSQL.
- Parentheses `( )` indicate literal parentheses.
- A comma `,` indicates the literal comma.
- Angle brackets `<>` indicate literal angle brackets.
- Uppercase words, such as `INSERT` , are keywords.

## Names

Naming rules in PostgreSQL-dialect databases are the same as those used in [open source PostgreSQL](https://www.postgresql.org/docs/current/sql-syntax-lexical.html#SQL-SYNTAX-IDENTIFIERS) , except for the following:

- No two Spanner objects (schemas, tables, columns, indexes, views, role, constraints, or sequences) can be created with the same name, including names that differ only in capitalization. For example, the second statement in the following snippet fails because the table names differ only by case.

  ```
  CREATE TABLE MyTable (col1 INT64) PRIMARY KEY (col1);
  CREATE TABLE MYTABLE (col1 INT64) PRIMARY KEY (col1);
  ```

  The following snippet fails because two different objects have the same name:

  ```
  CREATE TABLE MyTable (col1 bigint PRIMARY KEY);
  CREATE SCHEMA MyTable;
  ```

- Table and schema object names are case-sensitive but not case preserving. Case reverts to lowercase in the database. As an example, consider the table `singers` created with the following statement.

  ```
  CREATE TABLE singers (
    singerid bigint NOT NULL PRIMARY KEY,
    firstname  character varying(1024),
    lastname   character varying(1024),
    singerinfo bytea,
    birthdate  date
  );
  ```

  The following command succeeds:

  ```
  CREATE INDEX singersbyfirstlastname ON singers(firstname, lastname)
  ```

  For the following table:

  ```
  CREATE TABLE mytable2 (
    col1 bigint PRIMARY KEY
  );
  ```

  The following queries all succeed:

  ```
  SELECT col1 FROM MyTable2 LIMIT 1;
  SELECT COL1 FROM MYTABLE2 LIMIT 1;
  SELECT COL1 FROM mytable2 LIMIT 1;
  INSERT INTO MYTABLE2 (col1) VALUES(1);
  ```

## SCHEMA statements

This section has information about `SCHEMA` statements.

### CREATE SCHEMA

Creates a new schema and assigns a name.

```
CREATE SCHEMA [schema_name]
```

#### Spanner differences from open source PostgreSQL

`schema_name`

- Contains a name for a schema. If not used, the default schema is used, which is the same schema referred to by using `public` .
- When querying data, use fully qualified names (FQNs) to specify objects that belong to a specific schema. FQNs combine the schema name and the object name to identify database objects. For example, `products.albums` for the `products` schema and `albums` table. For more information, see [Named schemas](https://docs.cloud.google.com/spanner/docs/schema-and-data-model#named-schemas) .

### DROP SCHEMA

Removes a named schema.

```
DROP SCHEMA schema_name [, ...]
```

#### Spanner differences from open source PostgreSQL

`schema_name`

- Contains the name of the schema that you want to drop.
- `CASCADE` is not supported.

## DATABASE statements

This section has information about `DATABASE` statements.

### CREATE DATABASE

Creates a new database and assigns an ID.

```
CREATE DATABASE name
```

### ALTER DATABASE

Changes the definition of a database.

```
ALTER DATABASE name SET configuration_parameter_def

ALTER DATABASE name RESET configuration_parameter

where the configuration_parameter_def is:

    {
        spanner.default_leader { TO | = } { 'region' | DEFAULT }
        | spanner.optimizer_version { TO | = } { 1 ... 9 | DEFAULT }
        | spanner.optimizer_statistics_package { TO | = } { 'package_name' | DEFAULT }
        | spanner.version_retention_period { TO | = } { 'duration' | DEFAULT }
        | spanner.default_sequence_kind { TO | = } { 'bit_reversed_positive' | DEFAULT }
        | spanner.default_time_zone { TO | = } { 'time_zone_name' | DEFAULT }
        | spanner.read_lease_regions { TO | = } { 'read_lease_region_name' | DEFAULT }
        | spanner.columnar_policy { TO | = } { 'columnar_policy' | DEFAULT }
    }

and the configuration_parameter is:

    {
        spanner.default_leader
        | spanner.optimizer_version
        | spanner.optimizer_statistics_package
        | spanner.version_retention_period
        | spanner.default_sequence_kind
        | spanner.default_time_zone
        | spanner.read_lease_regions
        | spanner.columnar_policy
    }
```

#### Spanner differences from open source PostgreSQL

`spanner.default_leader { TO | = } { ' `***`region`***` ' | DEFAULT }`

- This configuration parameter lets you specify the leader for your database. The only regions eligible to become the leader region for your database are the read-write regions in the [dual-region](https://docs.cloud.google.com/spanner/docs/instance-configurations#available-configurations-dual) or [multi-region](https://docs.cloud.google.com/spanner/docs/instance-configurations#available-configurations-multi-region) configuration. Use `DEFAULT` to choose the default leader region of the base instance configuration. For more information about leader regions and voting replicas, see [Replication](https://docs.cloud.google.com/spanner/docs/replication) .

`spanner.optimizer_version { TO | = } { 1 ... 9 | DEFAULT }`

- This configuration parameter lets you specify the query optimizer version to use. Use `DEFAULT` for the current default version, as listed in [Query optimizer](https://docs.cloud.google.com/spanner/docs/query-optimizer/overview) .

`spanner.optimizer_statistics_package { TO | = } { ' `***`package_name`***` ' | DEFAULT }`

- This configuration parameter lets you specify the query optimizer statistics package name to use. By default, this is the latest collected statistics package, but you can specify any available statistics package version. Use `DEFAULT` for the latest version. For more information, see [Query statistics package versioning](https://docs.cloud.google.com/spanner/docs/query-optimizer/manage-query-optimizer) .

`spanner.version_retention_period { TO | = } { ' `***`duration`***` ' | DEFAULT }`

- This configuration parameter lets you specify the period for which Spanner retains all versions of data and schema for the database. The duration must use the range `[1h, 7d]` and you can use days, hours, minutes, or seconds for the range. For example, the values `1d` , `24h` , `1440m` , and `86400s` are equivalent. Setting the value to `DEFAULT` resets the retention period to the default, which is 1 hour. You can use this option for point-in-time recovery. For more information, see [Point-in-time Recovery](https://docs.cloud.google.com/spanner/docs/pitr) .

`spanner.default_sequence_kind { TO | = } { ' `***`bit_reversed_positive`***` ' | DEFAULT }`

- This configuration parameter lets you specify the default sequence kind for your database. ***`bit_reversed_positive`*** is the only valid sequence kind. The `bit_reversed_positive` option specifies that the values generated by the sequence are of type `bigint` , are greater than zero, and aren't sequential. You don't need to specify a sequence type when using `default_sequence_kind` . When you use `default_sequence_kind` for a sequence or identity column, you can't change the sequence kind later. For more information, see [Primary key default values management](https://docs.cloud.google.com/spanner/docs/primary-key-default-value#serial-auto-increment) .

`spanner.default_time_zone { TO | = } { ' `***`time_zone_name`***` ' | DEFAULT }`

- This configuration parameter lets you specify the default time zone for your database. If set to `DEFAULT` , the system uses `America/Los_Angeles` . Specifying a time zone within a `DATE` or `TIMESTAMP` function overrides this setting. The `time_zone_name` must be a valid entry from the [IANA Time Zone Database](https://www.iana.org/time-zones) . This option can only be set on empty databases without any tables.

`spanner.read_lease_regions { TO | = } { ' `***`read_lease_region_name`***` ' | DEFAULT }`

- This configuration parameter sets the [read lease](https://docs.cloud.google.com/spanner/docs/read-lease) region for your database. By default, or when you set it to `DEFAULT` , the database doesn't use any read lease regions. If you set one or more read lease regions for your database, Spanner gives the right to serve reads locally to one or more non-leader, read-write, or read-only regions. This lets the non-leader regions directly serve strong reads and reduce strong read latency.

`spanner.columnar_policy { TO | = } { ' `***`columnar_policy`***` ' | DEFAULT }`

- This configuration parameter sets the [columnar policy](https://docs.cloud.google.com/spanner/docs/columnar-engine) for your database. By default, or when you set it to `DEFAULT` , no data will be written in columnar format unless enabled on a more specific schema object, such as a table or index. no data will be written in columnar format. For more information about how to configure columnar, see [Configure Spanner columnar engine](https://docs.cloud.google.com/spanner/docs/configure-columnar-engine#enable-columnar-engine-postgres) .

## PLACEMENT statements

> **Preview — [Geo-partitioning](https://docs.cloud.google.com/spanner/docs/geo-partitioning)**
>
> This feature is subject to the "Pre-GA Offerings Terms" in the General Service Terms section of the [Service Specific Terms](https://docs.cloud.google.com/terms/service-terms#1) . Pre-GA features are available "as is" and might have limited support. For more information, see the [launch stage descriptions](https://cloud.google.com/products/#product-launch-stages) .

This section has information about `PLACEMENT` statements.

### CREATE PLACEMENT

Use the `CREATE PLACEMENT` statement to define a placement to partition row data in your database. For more information, see [Geo-partitioning overview](https://docs.cloud.google.com/spanner/docs/geo-partitioning) .

#### Syntax

```
CREATE PLACEMENT placement_name WITH [ partition_def ]

where partition_def is:
    { ( instance_partition='partition_id' [, default_leader='leader_region_id' ]
        [, read_lease_regions = {'read_lease_region_name[, ... ]' | null } ] ) }
```

#### Description

`CREATE PLACEMENT` defines a new placement in the current database.

#### Parameters

***`placement_name`***

- The name of the placement.

***`partition_id`***

- The unique identifier of the user-created partition associated with the placement.

***`leader_region_id`***

- This optional parameter sets the default leader region for the partition. This parameter is similar to [setting the default leader](https://docs.cloud.google.com/spanner/docs/instance-configurations#configure-leader-region) at the database level, except it applies to only the partition.

`read_lease_regions { TO | = } { ' `***`read_lease_region_name`***` ' | DEFAULT }`

- This configuration parameter sets one or more [read lease](https://docs.cloud.google.com/spanner/docs/read-lease) regions for your placement. By default, or when you set it to `DEFAULT` , the placement doesn't use any read lease regions. If you set one or more read lease regions for your database, Spanner gives the right to serve reads locally to one or more non-leader, read-write, or read-only regions. This lets the non-leader regions directly serve strong reads and reduce strong read latency.

### DROP PLACEMENT

Use the `DROP PLACEMENT` statement to delete a placement.

#### Syntax

```
DROP PLACEMENT placement_name
```

#### Description

`DROP PLACEMENT` drops a placement.

#### Parameters

***`placement_name`***

- The name of the placement to drop.

## LOCALITY GROUP statements

This section has information about `LOCALITY GROUP` statements.

### CREATE LOCALITY GROUP

Use the `CREATE LOCALITY GROUP` statement to define a locality group to store some columns separately or to use tiered storage. For more information, see [Locality groups](https://docs.cloud.google.com/spanner/docs/schema-and-data-model#locality-groups) and [Tiered storage overview](https://docs.cloud.google.com/spanner/docs/tiered-storage) .

#### Syntax

```
CREATE LOCALITY GROUP locality_group_name [ storage_def ]

where storage_def is:
    { STORAGE '{ ssd | hdd }' [, SSD_TO_HDD_SPILL_TIMESPAN 'duration' ] }
```

#### Description

`CREATE LOCALITY GROUP` defines a new locality group in the current database.

#### Parameters

***`locality_group_name`***

- The name of the locality group.

***`storage_def`***

- Use ***`STORAGE`*** to define the storage type of the locality group. You can set the storage type as 'ssd' or 'hdd'.

- Use ***`SSD_TO_HDD_SPILL_TIMESPAN`*** to define the amount of time that data is stored in SSD storage before it moves to HDD storage. After the specified time passes, Spanner migrates the data to HDD storage during its normal compaction cycle, which typically occurs over the course of seven days from the specified time. The duration must be at least one hour ( `1h` ) and at most 365 days ( `365d` ) long. It can be specified in days, hours, minutes, or seconds. For example, the values `1d` , `24h` , `1440m` , and `86400s` are equivalent.

### ALTER LOCALITY GROUP

Use the `ALTER LOCALITY GROUP` statement to change the storage option or age-based policy of a locality group.

#### Syntax

```
ALTER LOCALITY GROUP locality_group_name [ storage_def ]

where storage_def is:
    { STORAGE '{ ssd | hdd }' SSD_TO_HDD_SPILL_TIMESPAN 'duration' }
```

#### Description

`ALTER LOCALITY GROUP` changes the storage option or age-based policy of a locality group. You can change these options together or individually.

#### Parameters

***`locality_group_name`***

- The name of the locality group.

***`storage_def`***

- Use ***`STORAGE`*** to define the new storage type of the locality group.

- Use the `SSD_TO_HDD_SPILL_TIMESPAN = 'duration'` option to set the new age-based policy of the locality group. The duration must be at least one hour ( `1h` ) and at most 365 days ( `365d` ) long. It can be specified in days, hours, minutes, or seconds. For example, the values `1d` , `24h` , `1440m` , and `86400s` are equivalent.

### DROP LOCALITY GROUP

Use the `DROP LOCALITY GROUP` statement to drop the locality group. You can't drop a locality group if it contains data. You must first move all data that's in the locality group to another locality group.

#### Syntax

```
DROP LOCALITY GROUP locality_group_name
```

#### Description

`DROP LOCALITY GROUP` drops the locality group.

## INDEX statements

This section has information about `INDEX` statements.

### CREATE INDEX

```
CREATE [ UNIQUE ] INDEX [ IF NOT EXISTS ] name ] ON table_name
    ( key_part [, ...] )
    [ USING method ( column_name [, ...] ) ]
    [ INCLUDE ( column_name [, ...] ) ]
    [ WITH ( index_option_list) ]
    [ LOCALITY GROUP locality_group_name ]
    [ COLUMNAR POLICY columnar_policy ]
    [ INTERLEAVE IN parent_table_name ]
    [ WHERE predicate ]

where key_part is:
    { column_name | ( expression )  } [ ASC | DESC ] [ NULLS { FIRST | LAST } ]

and predicate is:

    column_name IS NOT NULL
        [ AND column_name IS NOT NULL ] [ ... ]
```

#### Spanner differences from open source PostgreSQL

- `USING` and `WITH` are used to create [vector indexes](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#vector-indexes-ddl) only.

`[ INTERLEAVE IN parent_table_name ]`

- `INTERLEAVE IN` clause defines a table to interleave the index in (see [Index options](https://docs.cloud.google.com/spanner/docs/whitepapers/optimizing-schema-design#index-options) for more details). If T is the table into which the index is interleaved, then the primary key of T must be the key prefix of the index, with each key matching in type, sort order, and nullability. Matching by name is not required.

`[ WHERE `***`predicate`***` ]`

- The ***`predicate`*** can refer only to columns that are specified earlier in the `CREATE INDEX` statement, not to any column in the underlying table.

#### Vector indexes

The following parameters are supported for [vector indexes](https://docs.cloud.google.com/spanner/docs/vector-indexes) used in approximate nearest neighbor (ANN) search:

`[ USING `***`method`***` ( `***`column_name`***` [, ...] ) ]` (vector indexes only)

- The `method` specifies the index access method. Currently, Spanner only supports `scann` for vector indexes. Spanner uses the ScaNN algorithm to perform tree-quantization-based indexing of vectors.

`[ WITH ( `***`index_option_list`***` ) ]` (vector indexes only)

- The index option list specifies algorithmic configuration for the index. Currently, this is only supported for vector indexes that use the `scann` method. Specify the options in the following format: `NAME=VALUE, ...` .

  The following index options are supported for ScaNN indexes:

  | `Name`          | `Value`   | Details                                                                                                                                                                                                                                                                                                                                                                                                   |
  |-----------------|-----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
  | `distance_type` | `varchar` | Required. The distance metric used to build the vector index. This value can be [`COSINE`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions#mathematical) , [`DOT_PRODUCT`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions#mathematical) , or [`EUCLIDEAN`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/functions#mathematical) . |
  | `tree_depth`    | `int`     | The tree depth (level). This value can be either `2` or `3` . A tree with 2 levels only has leaves ( `num_leaves` ) as nodes. If the dataset has more than 100 million rows, then you can use a tree with 3 levels and add branches ( `num_branches` ) to further partition the dataset.                                                                                                                  |
  | `num_leaves`    | `bigint`  | The number of leaves (that is, potential partitions) for the vector data. You can designate `num_leaves` for trees with 2 or 3 levels. We recommend that the number of leaves is `number_of_rows_in_dataset/1000` .                                                                                                                                                                                       |
  | `num_branches`  | `bigint`  | The number of branches to further partition the vector data. You can only designate `num_branches` for trees with 3 levels. The number of branches must be fewer than the number of leaves. We recommend that the number of branches is between `1000` and `sqrt(number_of_rows_in_dataset)` .                                                                                                            |

### ALTER INDEX

Adds or removes a non-key column from an index.

```
ALTER INDEX index_name {ADD|DROP} INCLUDE COLUMN column_name
[ SET index_option ]

where index_option is:
    { LOCALITY GROUP 'locality_group_name'
    | COLUMNAR POLICY { 'columnar_policy' | NULL } }
```

### REINDEX INDEX

Performs an in-place reindex of a vector index.

```
REINDEX INDEX CONCURRENTLY index_name
```

#### Description

Performs an in-place reindex of a vector index with the same options to refresh its tree structure. Reindexing occurs in the background and allows read and write operations on the index to continue. You must specify `CONCURRENTLY` .

> **Note:** `REINDEX` is supported only for vector indexes.

### DROP INDEX

Removes a secondary index.

```
DROP INDEX [ IF EXISTS ] name
```

## SEARCH INDEX statements

This section has information about `SEARCH INDEX` statements.

### CREATE SEARCH INDEX

Use the `CREATE SEARCH INDEX` statement to define search indexes. For more information, see [Search indexes](https://docs.cloud.google.com/spanner/docs/full-text-search/search-indexes) .

#### Syntax

```
CREATE SEARCH INDEX index_name
ON table_name(tokenlist_column_name *)
  [INCLUDE(column_name [, ...])]
  [PARTITION BY column_name[, ...] 
  [ORDER BY column_name [ ASC | DESC ]
  [INTERLEAVE IN parent_table_name]
  [WHERE column_name IS NOT NULL [AND ...]]
  [WITH(search_index_options)])

where index_name is:
    {a—z|A—Z}[{a—z|A—Z|0—9|_}+]

and tokenlist_column is:
    column_name [, ...]

and search_index_options are:
    {sort_order_sharding = {true|false}}
```

#### Description

You can use `CREATE SEARCH INDEX` to create search indexes for `TOKENLIST` columns. Adding a search index on a column makes it more efficient to search data in the source column of the `TOKENLIST` .

#### Parameters

***`index_name`***

- The name to assign to the search index.

***`table_name`***

- The name of the table to be indexed for search.

***`tokenlist_column_name`***

- A list of `TOKENLIST` columns to be indexed for search.

`INCLUDE`

- Provides a mechanism for duplicating data from the table into the search index.

`PARTITION BY`

- Indicates a list of columns to partition the search index by. Partition columns subdivide the index into smaller units, one for each unique partition. Queries can only search within a single partition at a time. Queries against partitioned indexes are generally more efficient than queries against unpartitioned indexes because only splits from a single partition need to be read.

`ORDER BY`

- A list of `bigint` columns used to sort the rows within each partition of the search index. The column must be `NOT NULL` , or the index must define `WHERE IS NOT NULL` . This property can support at most one column.

`WHERE `***`column_name`***` IS NOT NULL`

- Rows that contain `NULL` in any of the columns listed in this clause don't get included in the index. The columns must be present in the `ORDER BY` or `INCLUDE` clause.

`INTERLEAVE IN`

- Similar to `INTERLEAVE IN` for secondary indexes, search indexes can be interleaved in an ancestor table of the base table. The primary reason to use interleaved search indexes is to colocate base table data with index data for small partitions.

- Interleaved search indexes have the following limitations:

  - Only sort-order sharded indexes can be interleaved.
  - Search indexes can only be created on top-level tables. They can't be interleaved with child tables.
  - Like interleaved tables and secondary indexes, the parent table's key must be a prefix of the partitioning columns of the interleaved search index.

`WITH`

- Specifies options to use when creating search indexes:

  - `sort_order_sharding` : Permits specifying sharding of the index. The default value is `false` in which case the index is uniformly sharded.

### ALTER SEARCH INDEX

Use the `ALTER SEARCH INDEX` statement to add or remove columns from the search indexes.

#### Syntax

```
ALTER SEARCH INDEX index_name {ADD|DROP} INCLUDE COLUMN column_name
```

#### Description

Add a `TOKENLIST` column into a search index or remove an existing `TOKENLIST` column from a search index. Use `INCLUDE COLUMN` to add or remove stored columns from a search index.

#### Parameters

***`index_name`***

- The name of the search index to alter.

***`column_name`***

- The name of the column to add or remove from the search index.

### DROP SEARCH INDEX

Removes a search index.

#### Syntax

```
DROP SEARCH INDEX [ IF EXISTS ] index_name
```

#### Description

Use the `DROP SEARCH INDEX` statement to drop a search index.

#### Parameters

`IF EXISTS`

- The `DROP` statement has no effect if the specified search index doesn't exist.

***`index_name`***

- The name of the search index to drop.

## SEQUENCE statements

This section describes `SEQUENCE` statements.

### CREATE SEQUENCE

Creates a sequence object with the specified attributes.

```
CREATE SEQUENCE [ IF NOT EXISTS ] sequence_name
[ sequence_kind ]
[ NO MINVALUE ]
[ NO MAXVALUE ]
[ SKIP RANGE skip_range_min skip_range_max ]
[ START COUNTER [ WITH ] start_with_counter ]
[ NO CYCLE ]
[ OWNED BY NONE ]
```

#### Spanner differences from open source PostgreSQL

Bit-reversed positive sequences don't support the following open source PostgreSQL `SEQUENCE` clauses:

- `[ AS `***`data_type`***` ]`
- `[ INCREMENT [ BY ] `***`increment`***` ]`
- `[ MINVALUE `***`minvalue`***` ] [ MAXVALUE `***`maxvalue`***` ]`
- `[ START [ WITH ] `***`start`***` ]`
- `[ CACHE `***`cache`***` ]`
- `[ CYCLE ]`

Spanner extends open source PostgreSQL with the following:

***`sequence_kind`***

- Inputs a string to indicate the type of sequence to use. At this time, ***`bit_reversed_positive`*** is the only valid sequence kind.

`[ SKIP RANGE `***`skip_range_min`***` `***`skip_range_max`***` ]`

- Restricts the sequence from generating values in that range. The skipped range is inclusive. Since bit-reversed positive sequences only generate positive values, setting a negative `SKIP RANGE` has no effect.

- `skip_range_min` and `skip_range_max` are both `bigint` value types. They both have a default value of null. The accepted values for `skip_range_min` is any value that is lesser than or equal to `skip_range_max` . The accepted values for `skip_range_max` is any value that is more than or equal to `skip_range_min` .

`[ START COUNTER [ WITH ] `***`start_with_counter`***` ]`

- Sets the next value for the internal sequence counter. For example, the next time that Spanner obtains a value from the bit-reversed sequence, it begins with `start_with_counter` . Spanner bit reverses this value before returning it to the client.

- `start_with_counter` is an `bigint` value type. The default value is `1` and it accepts positive `bigint` values.

- When the counter reaches the maximum in the `bigint` number space, the sequence no longer generate values. The sequence generator function, `nextval()` returns an error when it reaches the maximum number of values.

#### Examples

In the following example, you create a positive bit-reversed positive sequence. When you create a table, you can use `nextval` , the sequence generator function, as the default value of the primary key column, `SingerId` . Values the sequence generates are positive and bit-reversed.

```
CREATE SEQUENCE mysequence bit_reversed_positive START COUNTER WITH 50;

CREATE TABLE singers (
  singerid bigint DEFAULT nextval('mysequence'),
  name bigint,
  PRIMARY KEY (singerid)
);
```

### ALTER SEQUENCE

`ALTER SEQUENCE` makes changes to the specified sequence. Executing this statement doesn't affect values that the sequence previously generated. If the `ALTER SEQUENCE` statement doesn't include an option, the current value of the option remains the same.

After you execute `ALTER SEQUENCE` , the specified sequence uses the new schema options.

```
ALTER SEQUENCE [ IF EXISTS ] sequence_name
[ NO MINVALUE ]
[ NO MAXVALUE ]
[ SKIP RANGE skip_range_min skip_range_max ]
[ RESTART COUNTER [ WITH ] counter_restart ]
[ NO CYCLE ]
```

#### Spanner differences from open source PostgreSQL

Bit-reversed positive sequences don't support the following open source PostgreSQL `ALTER SEQUENCE` clauses:

- `[ AS `***`data_type`***` ]`
- `[ INCREMENT [ BY ] `***`increment`***` ]`
- `[ MINVALUE `***`minvalue`***` ]`
- `[ MAXVALUE `***`maxvalue`***` ]`
- `[ START [ WITH ] `***`start`***` ]`
- `[ RESTART [ WITH ] `***`restart`***` ]`
- `[ CACHE `***`cache`***` ]`
- `ALTER SEQUENCE [ IF EXISTS ] `***`name`***` SET { LOGGED | UNLOGGED }`
- `ALTER SEQUENCE [ IF EXISTS ] `***`name`***` OWNER TO { `***`new_owner`***` | CURRENT_ROLE | CURRENT_USER | SESSION_USER }`
- `ALTER SEQUENCE [ IF EXISTS ] `***`name`***` RENAME TO `***`new_name`***
- `ALTER SEQUENCE [ IF EXISTS ] `***`name`***` SET SCHEMA `***`new_schema`***

Spanner extends open source PostgreSQL with the following:

`[ SKIP RANGE `***`skip_range_min`***` `***`skip_range_max`***` ]`

- Restricts the sequence from generating values in the specified range. Since positive bit-reversed sequences only generate positive values, setting a negative `SKIP RANGE` has no effect.

`[ RESTART COUNTER [WITH] `***`counter_restart`*** \]

- Sets the current sequence counter to the user-specified value.

#### Examples

Alter a sequence to include an skipped range.

```
ALTER SEQUENCE mysequence SKIP RANGE 1 1234567;
```

Set the current sequence counter to 1000.

```
ALTER SEQUENCE mysequence RESTART COUNTER WITH 1000;
```

### DROP SEQUENCE

Drops a sequence.

#### Syntax

```
DROP SEQUENCE [IF EXISTS] sequence_name
```

#### Spanner differences from open source PostgreSQL

Bit-reversed positive sequences don't support the following open source PostgreSQL `DROP SEQUENCE` clauses:

- `[CASCADE]`
- `[RESTRICT]`

#### Description

`DROP SEQUENCE` drops a specific sequence. Spanner can't drop a sequence if its name appears in a sequence function used in a column default value or view.

## STATISTICS statements

This section has information about `STATISTICS` statements.

### ALTER STATISTICS

Sets optional attributes of a query optimizer statistics package.

#### Syntax

```
ALTER STATISTICS spanner."<package_name>"
    action

where package_name is:
    {a—z}[{a—z|0—9|_|-}+]{a—z|0—9}

and action is:
    SET OPTIONS ( options_def )

and options_def is:
    { allow_gc = { true | false } }
```

#### Description

`ALTER STATISTICS` sets optional attributes of a query optimizer statistics package.

`SET OPTIONS`

- Use this clause to set an option on the specified statistics package.

#### Parameters

***`package_name`***

- The name of an existing query optimizer statistics package whose attributes you want to alter.

  To fetch existing statistics packages:

  ```
  SELECT s.package_name AS package_name, s.allow_gc AS allow_gc
  FROM INFORMATION_SCHEMA.SPANNER_STATISTICS s;
  ```

***`options_def`***

- The `allow_gc = { true | false }` option lets you specify whether a given statistics package undergoes garbage collection. A package must be set as `allow_gc=false` if the package is used in a query hint. For more information, see [Garbage collection of statistics packages](https://docs.cloud.google.com/spanner/docs/query-optimizer/overview#statistics-gc) .

### ANALYZE

Starts a new query optimizer statistics package construction.

#### Syntax

```
ANALYZE
```

#### Description

`ANALYZE` starts a new query optimizer statistics package construction.

## TABLE statements

This section has information about `TABLE` statements.

### CREATE TABLE

Defines a new table.

```
CREATE TABLE [ IF NOT EXISTS ] table_name (
      {
        column_name data_type
        [ column_constraint [ ... ] ] | table_constraint
        | synonym_definition
      } [, ... ],
      [ PRIMARY KEY (column_name) ]
    )
    [ WITH ( table_options ) ]
    [ LOCALITY GROUP locality_group_name ]
    [ COLUMNAR POLICY columnar_policy ]
    [ INTERLEAVE IN [ PARENT ] parent_table_name ]
    [ ON DELETE ( CASCADE | NO ACTION ) ]
    [ TTL INTERVAL interval_spec ON timestamp_column_name ]

where column_constraint is:

    [ CONSTRAINT constraint_name ] {
        NOT NULL
        | NULL
        | CHECK ( expression )
        | DEFAULT expression [ ON UPDATE expression ]
        | GENERATED ALWAYS AS ( expression ) { STORED | VIRTUAL } [ HIDDEN ]
        | GENERATED BY DEFAULT AS IDENTITY [ ( sequence_option_clause ... ) ]
        | PRIMARY KEY
        | REFERENCES reftable ( refcolumn )
            [ ON DELETE {CASCADE | NO ACTION} ]
    }

and table_constraint is:

    [ CONSTRAINT constraint_name ] {
        CHECK ( expression )
        | PRIMARY KEY ( column_name [, ... ] )
        | FOREIGN KEY ( column_name [, ... ] )
        REFERENCES reftable ( refcolumn [, ... ] )
            [ ON DELETE {CASCADE | NO ACTION} ]
    }

and synonym_definition is:

    [ SYNONYM (synonym) ]

and sequence_option_clause is:
    { BIT_REVERSED_POSITIVE
    | NO MINVALUE
    | NO MAXVALUE
    | SKIP RANGE skip_range_min skip_range_max
    | START COUNTER [ WITH ] start_with_counter
    | NO CYCLE }

and table_options are:
    { type = 'fulltext_dictionary' |
      fulltext_dictionary_staleness = 'staleness' }
```

`PRIMARY KEY` `( `***`column_name`***` )`

In Spanner a primary key is required when creating a new table.

`HIDDEN`

Hides a column if it shouldn't appear in `SELECT *` statements. If the column is hidden, you can still select it using its name. For example, `SELECT Id, Name, ColHidden FROM TableWithHiddenColumn` .

The primary use case for `HIDDEN` columns is to omit `TOKENLIST` columns from a `SELECT *` statement.

`DEFAULT` ***`expression`***

- This clause sets a default value for the column.

- You can use a key or non-key column for a column with a default value..

- You can't create a column with a default value if it's a generated column.

- You can insert your own value into a column that has a default value, overriding the default value. You can also use `UPDATE ... SET` ***`column-name`*** `= DEFAULT` to reset a non-key column to its default value.

- A generated column or a check constraint can depend on a column with a default value.

- A column can only use `SPANNER.PENDING_COMMIT_TIMESTAMP()` as a default value if it has the `SPANNER.COMMIT_TIMESTAMP` type (this is the only default value allowed for this type).

- You can use a literal or any valid SQL expression that is assignable to the column data type as an ***`expression`*** , with the following properties and restrictions:

  - The expression can be nondeterministic.
  - The expression can't reference other columns.
  - The expression can't contain subqueries, query parameters, aggregates, or analytic functions.

`ON UPDATE` ***`expression`***

- This clause configures a column to automatically update its value whenever a row is modified. This is typically used to maintain "last updated" timestamps without requiring manual input in every `UPDATE` statement.

- The column is set to the result of the expression whenever an update occurs on any non-key column in the row.

- The expression is triggered even if the update statement sets a column to its current value (that is, no actual data change occurs).

- You can bypass the automated value by explicitly providing a value for the column within your [`UPDATE`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax#update-statement) or [`INSERT`](https://docs.cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax#insert-statement) statement.

- To use the `ON UPDATE` clause, the column must satisfy these conditions:

  - Must not be part of the table's `PRIMARY KEY` .
  - Must have a `DEFAULT` expression that is identical to the `ON UPDATE` expression.
  - Must be a commit timestamp column, and the expression must be `SPANNER.PENDING_COMMIT_TIMESTAMP()` .

`GENERATED BY DEFAULT AS IDENTITY [ (` ***`sequence_option_clause`*** `... )]`

- This clause auto-generates integer values for the column.
- `BIT_REVERSED_POSITIVE` is the only valid type.
- An identity column can be a key or non-key column.
- An identity column can't have a default value or be a generated column.
- You can insert your own value into an identity column. You can also reset a non-key column to use generated value by using `UPDATE ... SET` ***`column-name`*** `= DEFAULT` .
- A generated column or a check constraint can depend on an identity column.
- An identity column accepts the following option clauses:
  - `BIT_REVERSED_POSITIVE` indicates the type of identity column.
  - `SKIP RANGE` ***`skip_range_min`*** , ***`skip_range_max`*** allows the underlying sequence to skip the numbers in this range when calling `GET_NEXT_SEQUENCE_VALUE` . The skipped range is an integer value and inclusive. The accepted values for `skip_range_min` is any value that is less than or equal to `skip_range_max` . The accepted values for `skip_range_max` is any value that is greater than or equal to `skip_range_min` .
  - `START COUNTER WITH` ***`start_with_counter`*** is a positive `bigint` value that Spanner uses to set the next value for the internal sequence counter. For example, when Spanner obtains a value from the bit-reversed sequence, it begins with `start_with_counter` . Spanner bit reverses this value before returning it. The default value is `1` .

`GENERATED ALWAYS AS (` ***`expression`*** `) { STORED | VIRTUAL }`

- This clause creates a column as a *generated column* . Its value is defined as a function of other columns in the same row.

- You can use a literal or any valid SQL expression that is assignable to the column data type as an ***`expression`*** , with the following restrictions:

  - The expression can only reference columns in the same table.

  - The expression can't contain subqueries.

  - The expression can't contain nondeterministic functions such as `SPANNER.PENDING_COMMIT_TIMESTAMP()` , `CURRENT_DATE` , and `CURRENT_TIMESTAMP` .

  - You can't modify the expression of a generated column.

- The `STORED` attribute that follows the expression causes Spanner to store the result of the function along with other columns of the table. Subsequent updates to any of the referenced columns cause Spanner to re-evaluate and store the expression.

- The `VIRTUAL` attribute that follows the expression in Spanner doesn't store the result of the expression in the table.

- Spanner doesn't allow direct writes to generated columns.

- You can't use a commit timestamp column as a generated column, nor can any of the columns that the generated columns references.

- You can't change the data type of a STORED generated column, or of any columns that the generated column references.

- You can't drop a column the generated column references.

- The following rules apply when using generated key columns:

  - The generated primary key can't reference other generated columns.

  - The generated primary key can reference, at most, one non-key column.

  - The generated primary key can't depend on a non-key column with a `DEFAULT` clause.

  - The DML doesn't let you explicitly write to generated primary keys.

#### Spanner differences from open source PostgreSQL

Spanner might choose a different name for an anonymous constraint than would open source PostgreSQL. Therefore, if you depend on constraint names, use `CONSTRAINT `***`constraint_name`*** to specify them explicitly.

Spanner extends open source PostgreSQL with the following:

`INTERLEAVE IN PARENT `***`parent_table_name`***` [ ON DELETE ( CASCADE | NO ACTION ) ]`

- This clause defines a [child-to-parent](https://docs.cloud.google.com/spanner/docs/schema-and-data-model#parent-child) table relationship, which results in a physical interleaving of parent and child rows. The primary-key columns of a parent must positionally match, both in name and type, a prefix of the primary-key columns of any child. Adding rows to the child table fails if the corresponding parent row doesn't exist. The parent row can either already exist in the database or can be inserted before the insertion of the child rows in the same transaction.

- The optional `ON DELETE` clause defines the behavior of rows in a child table when a transaction attempts to delete the parent row. The supported options are:

  - `CASCADE` which deletes the child rows.

  - `NO ACTION` which doesn't delete the child rows. If deleting a parent would leave behind child rows, thus violating parent-child referential integrity, the transaction attempt fails.

  If you omit the `ON DELETE` clause, the behavior is that of `ON DELETE NO ACTION` .

`INTERLEAVE IN `***`parent_table_name`***

- `INTERLEAVE IN` defines the same parent-child relationship and physical interleaving of parent and child rows as `INTERLEAVE IN PARENT` , but the parent-child referential integrity constraint isn't enforced. Rows in the child table can be inserted before the corresponding rows in the parent table. Like with `IN PARENT` , the primary-key columns of a parent must positionally match, both in name and type, a prefix of the primary-key columns of any child.

`TTL INTERVAL `***`interval_spec`***` ON `***`timestamp_column_name`***

- This clause defines a [time to live (TTL)](https://docs.cloud.google.com/spanner/docs/ttl) policy on the table, which lets Spanner periodically delete data from the table.

  - *`interval_spec`* is the number of days past the timestamp in the `timestamp_column_name` in which Spanner marks the row for deletion. You must use a non-negative integer for the value and it must evaluate to a whole number of days. For example, `'3 days'` is allowed, but `'3 days - 2 minutes'` returns an error.

  - *`timestamp_column_name`* is the name of the timestamp column.

`LOCALITY GROUP `***`locality_group_name`***

- This clause defines a [locality group](https://docs.cloud.google.com/spanner/docs/schema-and-data-model#locality-groups) for the table, which determines its tiered storage policy.

  - *`locality_group_name`* is the name of the locality group.

`COLUMNAR POLICY `***`columnar_policy`***

- This clause defines a [columnar policy](https://docs.cloud.google.com/spanner/docs/columnar-engine) for the table. For more information about how to configure columnar, see [Configure Spanner columnar engine](https://docs.cloud.google.com/spanner/docs/configure-columnar-engine#enable-columnar-engine-postgres) .

`WITH ( `***`table_options`***` )`

- Use this clause to specify options for the table.

  - `type = 'fulltext_dictionary'` Use this option to define the table as a custom dictionary for full-text search. For more information, see [Search with query enhancement](https://docs.cloud.google.com/spanner/docs/full-text-search/search-query-enhancement#custom-dictionaries) .

  - `fulltext_dictionary_staleness = 'staleness'` Use this option to define the maximum staleness, as a string, for custom dictionary lookups. For more information, see [Dictionary lookup staleness](https://docs.cloud.google.com/spanner/docs/full-text-search/search-query-enhancement#dictionary-lookup-staleness) .

`[ SYNONYM ( `***`synonym`***` )]`

- Defines a [synonym](https://docs.cloud.google.com/spanner/docs/table-name-synonym#add-synonym) for a table, which is an additional name that an application can use to access the table. A table can have one synonym. You can only use a synonym in queries and DML. You can't use the synonym for DDL or schema changes. You can see the synonym in the DDL representation of the table.

### ALTER TABLE

Changes the definition of a table.

```
ALTER TABLE [ IF EXISTS ] [ ONLY ] name action

where action is one of:

    ADD SYNONYM synonym
    DROP SYNONYM synonym
    RENAME TO new_table_name
      [, ALTER TABLE [ IF EXISTS ] [ ONLY ] new_table_name RENAME TO new_table_name ...]
    RENAME WITH SYNONYM TO new_table_name
    SET LOCALITY GROUP locality_group_name
    SET COLUMNAR POLICY columnar_policy
    ADD [ COLUMN ] [ IF NOT EXISTS ] column_name data_type [ column_expression ]
    DROP [ COLUMN ] column_name
    ADD table_constraint
    DROP CONSTRAINT constraint_name [ RESTRICT | CASCADE ]
    ALTER COLUMN column_name
      {
        [ SET DATA ] TYPE data_type
        | { SET | DROP } NOT NULL
        | SET  DEFAULT expression
        | DROP DEFAULT
        | SET ON UPDATE expression
        | DROP ON UPDATE
        | SET {
                NO MINVALUE
                | NO MAXVALUE
                | { SKIP RANGE skip_range_min skip_range_max | NO SKIP RANGE }
                | NO CYCLE
                | LOCALITY GROUP locality_group_name
              }
        | RESTART COUNTER [ WITH ] counter_restart
      }
    ADD TTL INTERVAL interval_spec ON timestamp_column_name 
    ALTER TTL INTERVAL interval_spec ON timestamp_column_name 
    SET INTERLEAVE IN [ PARENT ] parent_table_name [ ON DELETE { CASCADE | NO ACTION } ]

and column_expression is:
    [ NOT NULL ] { DEFAULT expression [ ON UPDATE expression ]
    | GENERATED ALWAYS AS ( expression ) STORED
    | GENERATED BY DEFAULT AS IDENTITY [ ( sequence_option_clause ... ) ] }

and table_constraint is:

    [ CONSTRAINT constraint_name ] {
        CHECK ( expression ) |
        FOREIGN KEY ( column_name [, ... ] ) REFERENCES reftable ( refcolumn [, ... ] )
            [ ON DELETE {CASCADE | NO ACTION} ]
    }

and sequence_option_clause is:
    { BIT_REVERSED_POSITIVE
    | NO MINVALUE
    | NO MAXVALUE
    | SKIP RANGE skip_range_min skip_range_max
    | START COUNTER [ WITH ] start_with_counter
    | NO CYCLE }
```

- You can specify `NOT NULL` in an `ALTER TABLE...ADD [ COLUMN ]` statement if you specify `DEFAULT` ***`expression`*** or `GENERATED ALWAYS AS (` ***`expression`*** `) STORED` for the column.

- If you include `DEFAULT` ***`expression`*** or `GENERATED ALWAYS AS (` ***`expression`*** `) STORED` , Spanner evaluates the expression and backfills the computed value for existing rows. The backfill operation is asynchronous. This backfill operation only happens when Spanner issues an `ADD COLUMN` statement.

- `ALTER COLUMN` statements that you use to `SET` or `DROP` the default value or `ON UPDATE` value of an existing column don't affect existing rows. There is no backfill operation on `ALTER COLUMN` .

- The `DEFAULT` clause has restrictions. See the description of this clause in [`CREATE TABLE`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#postgresql-default-clause) .

- The `ON UPDATE` clause has restrictions. See the description of this clause in [`CREATE TABLE`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#postgresql-on-update) .

#### Spanner differences from open source PostgreSQL

- These Spanner restrictions apply when dropping a column:
  - You can't drop a column that a CHECK constraint references.
  - You can't drop a primary key column.
- Spanner might choose a different name for an anonymous constraint than open source PostgreSQL. Therefore, if you depend on constraint names, specify them explicitly using `CONSTRAINT `***`constraint_name`*** .
- The following Spanner restrictions apply when altering a column (using the `ALTER COLUMN` clause):
  - The statement must contain exactly two `ALTER COLUMN` clauses applied to the same column. One clause must alter (or keep as is) the column's data type and another clause must alter (or keep as is) the column's nullability. For example, if column `c1` is nullable and you want it to stay nullable after the execution of the `ALTER TABLE` statement, you need to add `ALTER COLUMN c1 DROP NOT NULL` . For example: `ALTER TABLE t1 ALTER COLUMN c1 TYPE VARCHAR(10), ALTER COLUMN c1 DROP NOT NULL;`
  - Spanner only supports operations that the [Supported schema updates](https://docs.cloud.google.com/spanner/docs/schema-updates#supported-updates) section describes.
- For all operations (ADD or DROP) except `ALTER COLUMN` , Spanner supports only a single operation per `ALTER TABLE` statement.

Spanner extends open source PostgreSQL with the following:

`ADD TTL INTERVAL` , `ALTER TTL INTERVAL`

- This clause defines or alters a [time to live (TTL)](https://docs.cloud.google.com/spanner/docs/ttl) policy on the table, which lets Spanner periodically delete data from the table.
- *`interval_spec`* is the number of days past the timestamp in the `timestamp_column_name` in which Spanner marks the row for deletion. You must use a non-negative integer for its value and it must evaluate to a whole number of days. For example, Spanner permits you to use `'3 days'` , but `'3 days - 2 minutes'` returns an error.
- *`timestamp_column_name`* is a column with the data type `TIMESTAMPTZ` . You need to create this column if it doesn't exist already. Columns with [commit timestamps](https://docs.cloud.google.com/spanner/docs/commit-timestamp) are valid, as are [generated columns](https://docs.cloud.google.com/spanner/docs/ttl/working-with-ttl#ttl_on_generated_columns) . However, you can't specify a generated column that references a commit timestamp column.

`ADD SYNONYM`

- Adds a synonym to a table to give it an alternate name. You can use the synonym for reads, writes, queries, and for use with DML. You can't use `ADD SYNONYM` with DDL, such as to create an index. A table can have one synonym. For more information, see [Add a synonym to a table](https://docs.cloud.google.com/spanner/docs/table-name-synonym#add-synonym) .

`DROP SYNONYM`

- Removes a synonym from a table. For more information, see [Remove a synonym from a table](https://docs.cloud.google.com/spanner/docs/table-name-synonym#remove-synonym) .

`RENAME TO`

- Renames a table without creating a synonym. In addition, you can concatenate multiple `ALTER TABLE RENAME TO` statements (delimited by a comma) to atomically rename multiple tables. For more information, see [Rename a table](https://docs.cloud.google.com/spanner/docs/table-name-synonym#rename-table) .

  For example, to change the names of multiple tables atomically, do the following:

  ```
  ALTER TABLE singers
      RENAME TO artists,
      ALTER TABLE albums
              RENAME TO recordings;
  ```

`RENAME WITH SYNONYM TO`

- Adds a synonym to a table so that when you rename the table, you can add the old table name to the synonym. This gives you time to update applications with the new table name while still allowing them to access the table with the old name. For more information, see [Rename a table and add a synonym](https://docs.cloud.google.com/spanner/docs/table-name-synonym#rename-add-synonym) .

`SET INTERLEAVE IN`

- `SET INTERLEAVE IN PARENT` migrates an interleaved table to use `IN PARENT` semantics, which require that the parent row exist for each child row. While executing this schema change, the child rows are validated to ensure there are no referential integrity violations. If there are, the schema change fails. If no `ON DELETE` clause is specified, `NO ACTION` is the default. Note that directly migrating from an `INTERLEAVE IN` table to `IN PARENT ON DELETE CASCADE` is not supported. This must be done in two steps. The first step is to migrate `INTERLEAVE IN` to `INTERLEAVE IN PARENT T [ON DELETE NO ACTION]` and the second step is to migrate to `INTERLEAVE IN PARENT T ON DELETE CASCADE` . If referential integrity validation fails, use a query like the following to identify missing parent rows.

  ```
      SELECT pk1, pk2 FROM child
      EXCEPT DISTINCT
      SELECT pk1, pk2 FROM parent;
  ```

- `SET INTERLEAVE IN` , like `SET INTERLEAVE IN PARENT` , migrates an `INTERLEAVE IN PARENT` interleaved table to `INTERLEAVE IN` , thus removing the parent-child enforcement between the two tables.

- The `ON DELETE` clause is only supported when migrating to `INTERLEAVE IN PARENT` .

### DROP TABLE

Removes a table.

```
DROP TABLE [ IF EXISTS ] name
```

#### Spanner differences from open source PostgreSQL

Spanner can't drop a table that has indexes. However, in open source PostgreSQL, when you drop a table, all related indexes are also dropped.

## VIEW statements

This section has information about `VIEW` statements.

### CREATE OR REPLACE VIEW

Defines a new view or replaces an existing view. If `CREATE VIEW` is used and the view already exists, the statement fails. Use `CREATE OR REPLACE VIEW` to replace the view or security type of a view. For more information, see [About views](https://docs.cloud.google.com/spanner/docs/views) .

```
{ CREATE | CREATE OR REPLACE } VIEW view_name
  SQL SECURITY { INVOKER | DEFINER }
  AS query
```

### DROP VIEW

Removes a view. Only the view is dropped; the objects that it references are not.

```
DROP VIEW name
```

## QUEUE statements

This section has information about the `CREATE QUEUE` , `ALTER QUEUE` , and `DROP QUEUE` statements.

### CREATE QUEUE

Defines a new [queue](https://docs.cloud.google.com/spanner/docs/queues/queues-overview) .

#### Syntax

```
CREATE QUEUE [ IF NOT EXISTS ] queue_name (
      {
        key_column_name data_type [ key_column_constraint [ ... ] ]
        | payload payload_type NOT NULL [ HIDDEN ]
        | table_constraint
      } [, ... ]
    )
    [ INTERLEAVE IN [ PARENT ] parent_table_name
    [ ON DELETE { CASCADE | NO ACTION } ] ]
    [ TTL INTERVAL interval_spec ON timestamp_column_name ]
    [ WITH ( queue_option [, ... ] ) ]

where key_column_constraint is:

    NOT NULL
    | PRIMARY KEY
    | GENERATED ALWAYS AS ( expression ) STORED
    | HIDDEN

and table_constraint is:

    PRIMARY KEY ( key_column_name [, ... ] )

and payload_type is:

    { bytea | text | varchar [ ( length ) ] | jsonb }

and queue_option is:
    { receive_mode = { 'PULL' }
    | disable_send = { true | false }
    | disable_delivery = { true | false }
    | locality_group = 'locality_group_name' }
```

#### Description

`CREATE QUEUE` defines a new queue for managing transactional messaging.

Every queue must define a primary key composed of one or more key columns, and exactly one `payload` column. All columns in the queue other than `payload` must be part of the primary key.

For more information, see [Spanner queues overview](https://docs.cloud.google.com/spanner/docs/queues/queues-overview) .

#### Parameters

`IF NOT EXISTS`

- If a queue exists with the same name, the `CREATE` statement has no effect and no error is generated.

***`queue_name`***

- The name of the queue to be created. For naming rules, see [Names](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#names) .
- Queues can't be created in a named schema.

***`key_column_name`***

- The name of a primary key column to be created. A queue can have multiple primary key columns. All columns in a queue other than `payload` must be part of the primary key. For naming rules, see [Names](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#names) .
- Column names can't begin with the `spanner_` prefix, which is reserved.

***`data_type`***

- The data type of the primary key column. For supported types, see [PostgreSQL data types](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-types) . Serial types ( `serial` , `bigserial` , `smallserial` ) are not supported for queues.

`payload`

- The required column that holds the message content. Every queue must have exactly one.
- Because PostgreSQL folds unquoted identifiers to lowercase, write the column as `payload` . If you quote the name, it must be either `"payload"` or `"Payload"` . Any other spelling, such as `"PAYLOAD"` , is rejected.
- The `payload` column must be declared `NOT NULL` , and it can't be part of the primary key.

***`payload_type`***

- The data type of the `payload` column. The supported types are `bytea` , `text` , `varchar` , and `jsonb` .

`NOT NULL`

- This column annotation specifies that the column is required for all mutations that insert a new row. It is required on the `payload` column.
- You cannot add a `NOT NULL` column to an existing queue.

`HIDDEN`

- This column annotation hides a column so that it doesn't appear in `SELECT *` statements. You can still select a hidden column by naming it explicitly.
- `HIDDEN` can be applied to primary key columns and to the `payload` column.
- You can't mark every column in a queue as `HIDDEN` .

`GENERATED ALWAYS AS ( `***`expression`***` ) STORED`

- This clause creates a column as a *stored generated column* , which is a column whose value is defined as a function of other columns in the same row. In queues, generated columns can only be defined on primary key columns, and must specify `STORED` .

`PRIMARY KEY`

- Every queue must have a primary key, composed of one or more columns of that queue.
- The primary key can be defined at the column level with `PRIMARY KEY` , or at the table level with `PRIMARY KEY ( `***`key_column_name`***` [, ... ] )` .
- Unlike GoogleSQL, the primary key is declared *inside* the parentheses of the column list. There is no trailing `PRIMARY KEY` clause after the closing parenthesis.

`INTERLEAVE IN [ PARENT ] `***`parent_table_name`***` [ ON DELETE { CASCADE | NO ACTION } ]`

- `INTERLEAVE IN PARENT` defines a child-to-parent table relationship, which results in a physical interleaving of parent and child rows. The primary-key columns of a parent must positionally match, both in name and type, a prefix of the primary-key columns of any child.
- The optional `ON DELETE` clause defines the behavior of rows in the child queue when a mutation attempts to delete the parent row. The supported options are `CASCADE` (child rows are deleted) and `NO ACTION` (child rows are not deleted).

`TTL INTERVAL `***`interval_spec`***` ON `***`timestamp_column_name`***

- Use this clause to set a [time to live (TTL)](https://docs.cloud.google.com/spanner/docs/ttl) policy for this queue. Only one TTL policy can exist on a queue at a time.

***`timestamp_column_name`***

- The name of a `timestamptz` column that is also specified in the `CREATE QUEUE` statement. Because every column other than `payload` must be part of the primary key, this is a primary key column.
- Naming a column of any other type returns an error.

***`interval_spec`***

- An interval literal that specifies how long after the timestamp in `timestamp_column_name` Spanner marks the row for deletion. For example, `'3 days'` .
- The interval must be non-negative. A negative interval returns the error `TTL interval must be greater than or equal to zero` .

`WITH ( `***`queue_option`***` [, ... ] )`

- A list of key-value pairs to configure the queue.
- `receive_mode` : Optional. Specifies how messages are retrieved from the queue. The default and only valid value is `'PULL'` (messages are retrieved using the queue `spanner.receive_queue_name()` TVF). This option can only be set at creation time.
- `disable_send` : When set to true, transactions that attempt to send messages to this queue will fail with an `INVALID_ARGUMENT` error. This is useful when you want to temporarily halt incoming message traffic or drain a queue before dropping it.
- `disable_delivery` : When set to true, message delivery to consumers is suspended. `spanner.receive_queue_name()` TVF calls are still allowed and won't fail, but no new messages will arrive. This is useful for pausing processing for remediation if you suspect queue consumers are causing downstream problems.
- `locality_group` : Stores the queue in the specified locality group. For more information, see [Locality groups](https://docs.cloud.google.com/spanner/docs/schema-and-data-model#locality-groups) .

#### Examples

Create a queue interleaved in a parent table:

```
CREATE QUEUE usertasks (
  userid     bigint NOT NULL,
  messageid  varchar(36) NOT NULL,
  payload    bytea NOT NULL,
  PRIMARY KEY (userid, messageid)
) INTERLEAVE IN PARENT users ON DELETE CASCADE;
```

Create a queue in a locality group, with sends disabled:

```
CREATE QUEUE auditqueue (
  eventid varchar(36) NOT NULL PRIMARY KEY,
  payload jsonb NOT NULL
) WITH (locality_group = 'archive', disable_send = true);
```

Create a queue with a TTL policy:

```
CREATE QUEUE eventqueue (
  eventid    varchar(36) NOT NULL,
  created_at timestamptz NOT NULL,
  payload    text NOT NULL,
  PRIMARY KEY (eventid, created_at)
) TTL INTERVAL '3 days' ON created_at;
```

### ALTER QUEUE

Changes the definition of a queue.

#### Syntax

```
ALTER QUEUE queue_name
    action

where action is:
    SET OPTIONS ( queue_option [, ... ] )

and queue_option is:
    { disable_send = { true | false | null }
    | disable_delivery = { true | false | null }
    | locality_group = { 'locality_group_name' | null } }
```

#### Description

`ALTER QUEUE` changes the definition of an existing queue. `SET OPTIONS` is the only supported action.

`SET OPTIONS ( `***`queue_option`***` [, ... ] )`

- A list of key-value pairs to configure the queue.
- `disable_send` : When set to true, transactions that attempt to send messages to this queue will fail with an `INVALID_ARGUMENT` error. This is useful when you want to temporarily halt incoming message traffic or drain a queue before dropping it. Set to `null` to reset the option to its default value ( `false` ).
- `disable_delivery` : When set to true, message delivery to consumers is suspended. `spanner.receive_queue_name()` TVF calls are still allowed and won't fail, but no new messages will arrive. This is useful for pausing processing for remediation if you suspect queue consumers are causing downstream problems. Set to `null` to reset the option to its default value ( `false` ).
- `locality_group` : Moves the queue to the specified locality group. Set to `null` to remove the queue from its locality group. For more information, see [Locality groups](https://docs.cloud.google.com/spanner/docs/schema-and-data-model#locality-groups) .

> **Note:** `receive_mode` can only be set when the queue is created. Attempting to change it returns the error `Queue option 'receive_mode' cannot be altered.`

In PostgreSQL-dialect databases, `ALTER QUEUE` can't add or remove columns, change the primary key, change the interleaving of a queue, or add, replace, or drop a TTL policy.

#### Parameters

***`queue_name`***

- The name of an existing queue to alter.

***`queue_option`***

- A key-value pair to configure the queue.

#### Example

```
ALTER QUEUE usertasks SET OPTIONS (disable_send = true);
```

### DROP QUEUE

Removes a queue.

#### Syntax

```
DROP QUEUE [ IF EXISTS ] queue_name
```

#### Description

Use the `DROP QUEUE` statement to remove a queue from the database. Unless the `IF EXISTS` clause is specified, the statement fails if the queue doesn't exist.

#### Parameters

`IF EXISTS`

- If the queue doesn't exist, the `DROP` statement has no effect and doesn't generate an error.

***`queue_name`***

- The name of the queue to drop.

## CHANGE STREAM statements

This section has information about `CHANGE STREAM` statements.

### CREATE CHANGE STREAM

Defines a new [change stream](https://docs.cloud.google.com/spanner/docs/change-streams) . For more information, see [Create a change stream](https://docs.cloud.google.com/spanner/docs/change-streams/manage#create) .

```
CREATE CHANGE STREAM [ IF NOT EXISTS ] change_stream_name
[ FOR { table_columns [, ... ] | ALL } ]
[ WITH ( configuration_parameter_def [, ... ] ) ]

where table_columns is:
    table_name [ ( [ column_name, ... ] ) ]

and configuration_parameter_def is:
    { retention_period = 'duration'
      | value_capture_type = { 'OLD_AND_NEW_VALUES'
      | 'NEW_ROW' |'NEW_VALUES'
      | 'NEW_ROW_AND_OLD_VALUES' }
      | exclude_ttl_deletes = { false | true }
      | exclude_insert = { false | true }
      | exclude_update = { false | true }
      | exclude_delete = { false | true } }
```

`FOR {` ***`table_columns`*** `[, ... ] | ALL }`

- The `FOR` clause defines the tables and columns that the change stream watches.

- You can specify a list of ***`table_columns`*** to watch, where ***`table_columns`*** can be either of the following:

  - ***`table_name`*** : This watches the entire table, including all of the future columns when they are added to this table.

  - ***`table_name`*** `( [` ***`column_name`*** `, ... ] )` : You can optionally specify a list of zero or more non-key columns following the table name. This watches only the primary key and the listed non-key columns of the table. With an empty list of non-key columns, ***`table_name`*** `()` watches only the primary key.

  > **Note:** The change stream always watches the primary key columns. You only need to list the non-key columns. Listing any primary key columns is not allowed.

- `ALL` lets you watch all tables and columns in the entire database, including all of the future tables and columns as soon as they are created.

- When you omit the `FOR` clause, the change stream watches nothing.

`WITH (` ***`configuration_parameter_def`*** `[, ... ] )`

- The `retention_period = 'duration'` configuration parameter lets you specify how long a change stream retains its data. For duration you must use the range `[1d, 30d]` which you can specify in days, hours, minutes, or seconds. For example, the values `1d` , `24h` , `1440m` , and `86400s` are equivalent. The default is 7 day. For details, see [Data retention](https://docs.cloud.google.com/spanner/docs/change-streams#data-retention) .

- The `value_capture_type` configuration parameter controls which values to capture for a changed row. It can be `OLD_AND_NEW_VALUES` (default), `NEW_VALUES` , `NEW_ROW` , OR `NEW_ROW_AND_OLD_VALUES` . For details, see [Value capture type](https://docs.cloud.google.com/spanner/docs/change-streams#value-capture-type) .

- The `exclude_ttl_deletes` configuration parameter lets you filter out [time to live based deletes](https://docs.cloud.google.com/spanner/docs/ttl#filter) from your change stream. When you set this filter, only future TTL-based deletes are removed. It can be set to `false` (default) or `true` . For more information, see [TTL-based deletes filter](https://docs.cloud.google.com/spanner/docs/change-streams#ttl-filter) .

- The `exclude_insert` configuration parameter lets you filter out all `INSERT` table modifications from your change stream. It can be set to `false` (default) or `true` . For more information, see [Table modification type filters](https://docs.cloud.google.com/spanner/docs/change-streams#mod-type-filter) .

- The `exclude_update` configuration parameter lets you filter out all `UPDATE` table modifications from your change stream. It can be set to `false` (default) or `true` . For more information, see [Table modification type filters](https://docs.cloud.google.com/spanner/docs/change-streams#mod-type-filter) .

- The `exclude_delete` configuration parameter lets you filter out all `DELETE` table modifications from your change stream. It can be set to `false` (default) or `true` . For more information, see [Table modification type filters](https://docs.cloud.google.com/spanner/docs/change-streams#mod-type-filter) .

### ALTER CHANGE STREAM

Changes the definition of a change stream. For more information, see [Modify a change stream](https://docs.cloud.google.com/spanner/docs/change-streams/manage#modify) .

```
ALTER CHANGE STREAM name action

where action is:
    { SET FOR { table_columns [, ... ] | ALL } |
      DROP FOR ALL |
      SET ( configuration_parameter_def [, ... ] ) |
      RESET ( configuration_parameter [, ... ] ) }

and table_columns is:
    table_name [ ( [ column_name, ... ] ) ]

and configuration_parameter_def is:
    { retention_period = { 'duration' | null } |
      value_capture_type = { 'OLD_AND_NEW_VALUES' | 'NEW_ROW' | 'NEW_VALUES' | 'NEW_ROW_AND_OLD_VALUES' | null } |
      exclude_ttl_deletes = { false | true | null } |
      exclude_insert = { false | true | null } |
      exclude_update = { false | true | null } |
      exclude_delete = { false | true | null } }

and configuration_parameter is:
    { retention_period | value_capture_type | exclude_ttl_deletes
    | exclude_insert | exclude_update | exclude_delete }
```

`SET FOR {` ***`table_columns`*** `[, ... ] | ALL }`

- Sets a new `FOR` clause to modify what the change stream watches, using the same syntax as [`CREATE CHANGE STREAM`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#for-clause) .

`DROP FOR ALL`

- [Suspends a change stream](https://docs.cloud.google.com/spanner/docs/change-streams/manage#suspend) to watch nothing.

`SET`

- Sets configuration parameters on the change stream (such as `retention_period` , `value_capture_type` , `exclude_ttl_deletes` , `exclude_insert` , `exclude_update` and `exclude_delete` ), using the same syntax as [`CREATE CHANGE STREAM`](https://docs.cloud.google.com/spanner/docs/reference/postgresql/data-definition-language#change-stream-options) .

- Setting an option to `null` is equivalent to setting it to the default value.

`RESET`

- Resets configuration parameters on the change stream (such as `retention_period` , `value_capture_type` , `exclude_ttl_deletes` , `exclude_insert` , `exclude_update` , and `exclude_delete` ) to the default values.

### DROP CHANGE STREAM

Removes a change stream and deletes its data change records.

```
DROP CHANGE STREAM [ IF EXISTS ] name
```

## ROLE statements

This section has information about `ROLE` statements.

### CREATE ROLE

Defines a new database role.

#### Syntax

```
CREATE ROLE database_role_name
```

#### Description

`CREATE ROLE` defines a new database role. Database roles are collections of [fine-grained access control](https://docs.cloud.google.com/spanner/docs/fgac-about) privileges. You can create only one role with this statement.

#### Parameters

***`database_role_name`***

- The name of the database role to create. Role names can't start with `pg_` . The role name `public` and role names starting with `spanner_` are reserved for [system roles](https://docs.cloud.google.com/spanner/docs/fgac-system-roles) .

#### Example

This example creates the database role `hr_manager` .

```
CREATE ROLE hr_manager;
```

### DROP ROLE

Drops a database role.

#### Syntax

```
DROP ROLE database_role_name
```

#### Description

`DROP ROLE` drops a database role. You can drop only one role with this statement.

You can't drop a database role if it has any privileges granted to it. All privileges granted to a database role must be revoked before the role can be dropped. You can drop a database role whether or not access to it is granted to IAM principals.

Dropping a role automatically revokes its membership in other roles and revokes the membership of its members.

You can't drop [system roles](https://docs.cloud.google.com/spanner/docs/fgac-system-roles) .

#### Parameters

***`database_role_name`***

- The name of the database role to drop.

#### Example

This example drops the database role `hr_manager` .

```
DROP ROLE hr_manager;
```

## GRANT and REVOKE statements

This section has information about `GRANT` and `REVOKE` statements.

### GRANT

Grants roles to database objects.

#### Syntax

```
GRANT { privilege_type [, ...] }
  ON {
       SCHEMA schema_name_list |
       [ TABLE ] table_list |
       [ TABLE ] view_list |
       CHANGE STREAM change_stream_list |
       SEQUENCE sequence_list |
       FUNCTION function_list
     }
  TO role_list

GRANT { SELECT | INSERT | UPDATE }(column_list) ON [TABLE] table_list TO role_list

GRANT role_list TO role_list

where privilege_type depends on the object type:
  FOR SCHEMAS: { USAGE }
  FOR TABLES: { SELECT | INSERT | UPDATE | DELETE }
  FOR VIEWS: { SELECT }
  FOR CHANGE STREAMS: { SELECT }
  FOR SEQUENCES: { SELECT | UPDATE }
  FOR FUNCTIONS: { EXECUTE }

and table_list is:
    table_name [, ...]

and column_list is:
    column_name [, ...]

and view_list is:
    view_name [, ...]

and change_stream_list is:
    change_stream_name [, ...]

and sequence_list is:
    sequence_name [, ...]

and function_list is:
    change_stream_read_function_name [, ...]

and schema_name_list is:
    schema_name [, ...]

and role_list is:
    database_role_name [, ...]
```

#### Description

For [fine-grained access control](https://docs.cloud.google.com/spanner/docs/fgac-about) , grants privileges on one or more tables, views, change streams, or change stream read functions to database roles. Also grants database roles to other database roles to create a database role hierarchy with inheritance. When granting `SELECT` , `INSERT` , or `UPDATE` on a table, optionally grants privileges on only a subset of table columns.

#### Parameters

***`table_name`***

- The name of an existing table.

***`column_name`***

- The name of an existing column in the specified table.

***`view_name`***

- The name of an existing view.

***`change_stream_name`***

- The name of an existing change stream.

***`change_stream_read_function_name`***

- The name of an existing read function for a change stream. For more information, see [Change stream read functions and query syntax](https://docs.cloud.google.com/spanner/docs/change-streams/details#change_stream_query_syntax) .

***`database_role_name`***

- The name of an existing database role.

#### Notes and restrictions

The following notes and restrictions apply when you grant privileges to database roles.

- Identifiers for database objects named in the `GRANT` statement must use the case that was specified when the object was created. For example, if you created a table with a name that is in all lowercase with a capitalized first letter, you must use that same case in the `GRANT` statement. For each change stream, PostgreSQL automatically creates a change stream read function with a name that consists of a prefix added to the change stream name, so ensure that you use the proper case for both the prefix and the change stream name. For more information about change stream read functions, see [Change stream query syntax](https://docs.cloud.google.com/spanner/docs/change-streams/details#change_stream_query_syntax) .

- When granting column-level privileges on multiple tables, each table must contain the named columns.

- If a table contains a column that is marked `NOT NULL` and has no default value, you can't insert into the table unless you have the `INSERT` privilege on that column.

- After granting `SELECT` on a change stream to a role, grant `EXECUTE` on the change stream's read function to that role. For information about change stream read functions, see [Change stream read functions and query syntax](https://docs.cloud.google.com/spanner/docs/change-streams/details#change_stream_query_syntax) .

- Granting `SELECT` on a table doesn't grant `SELECT` on the change stream that tracks it. You must make a separate grant for the change stream.

#### Examples

The following example grants `SELECT` on the `employees` table to the `hr_rep` role. Grantees of the `hr_rep` role can read all columns of `employees` .

```
GRANT SELECT ON TABLE employees TO hr_rep;
```

The next example is the same as the previous example, but with the optional `TABLE` keyword omitted.

`GRANT SELECT ON employees TO hr_rep;`

The next example grants `SELECT` on a subset of columns of the `contractors` table to the `hr_rep` role. Grantees of the `hr_rep` role can only read the named columns.

```
GRANT SELECT(name, address, phone) ON TABLE contractors TO hr_rep;
```

The next example mixes table-level and column-level grants. `hr_manager` can read all table columns, but can update only the `location` column.

```
GRANT SELECT, UPDATE(location) ON TABLE employees TO hr_manager;
```

The next example makes column-level grants on two tables. Both tables must contain the `name` , `level` , and `location` columns.

```
GRANT SELECT (name, level, location), UPDATE (location)
ON TABLE employees, contractors
TO hr_manager;
```

The next example grants `INSERT` on a subset of columns of the `employees` table.

```
GRANT INSERT(name, cost_center, location, manager) ON TABLE employees TO hr_manager;
```

The next example grants the database role `pii_access` to the roles `hr_manager` and `hr_director` . The `hr_manager` and `hr_director` roles are *members* of `pii_access` and inherit the privileges that were granted to `pii_access` . For more information, see [Database role hierarchies and inheritance](https://docs.cloud.google.com/spanner/docs/fgac-about#role_hierarchy) .

```
GRANT pii_access TO hr_manager, hr_director;
```

### REVOKE

Revokes privileges on one or more tables, views, change streams, or change stream read functions.

#### Syntax

```
REVOKE { privilege_type [, ...] }
  ON {
       SCHEMA schema_name_list |
       [ TABLE ] table_list |
       [ TABLE ] view_list |
       CHANGE STREAM change_stream_list |
       SEQUENCE sequence_list |
       FUNCTION function_list
     }
  FROM role_list

REVOKE { SELECT | INSERT | UPDATE }(column_list) ON [TABLE] table_list FROM role_list

REVOKE role_list FROM role_list

where privilege_type depends on the object type:
  FOR SCHEMAS: { USAGE }
  FOR TABLES: { SELECT | INSERT | UPDATE | DELETE }
  FOR VIEWS: { SELECT }
  FOR CHANGE STREAMS: { SELECT }
  FOR SEQUENCES: { SELECT | UPDATE }
  FOR FUNCTIONS: { EXECUTE }

and table_list is:
    table_name [, ...]

and column_list is:
    column_name [, ...]

and view_list is:
    view_name [, ...]

and change_stream_list is:
    change_stream_name [, ...]

and sequence_list is:
    sequence_name [, ...]

and function_list is:
    change_stream_read_function_name [, ...]

and role_list is:
    database_role_name [, ...]
```

#### Description

For [fine-grained access control](https://docs.cloud.google.com/spanner/docs/fgac-about) , revokes privileges on one or more tables, views, change streams, or change stream read functions from database roles. Also revokes database roles from other database roles. When revoking `SELECT` , `INSERT` , or `UPDATE` on a table, optionally revokes privileges on only a subset of table columns.

#### Parameters

***`table_name`***

- The name of an existing table.

***`column_name`***

- The name of an existing column in the previously specified table.

***`view_name`***

- The name of an existing view.

***`change_stream_name`***

- The name of an existing change stream.

***`change_stream_read_function_name`***

- The name of an existing read function for a change stream. For more information, see [Change stream read functions and query syntax](https://docs.cloud.google.com/spanner/docs/change-streams/details#change_stream_query_syntax) .

***`database_role_name`***

- The name of an existing database role.

#### Notes and restrictions

The following notes and restrictions apply when you revoke privileges from database roles.

- Identifiers for database objects named in the `REVOKE` statement must use the case that was specified when the object was created. For example, if you created a table with a name that is in all lowercase with a capitalized first letter, you must use that same case in the `REVOKE` statement. For each change stream, PostgreSQL automatically creates a change stream read function with a name that consists of a prefix added to the change stream name, so ensure that you use the proper case for both the prefix and the change stream name. For more information about change stream read functions, see [Change stream query syntax](https://docs.cloud.google.com/spanner/docs/change-streams/details#change_stream_query_syntax) .

- When revoking column-level privileges on multiple tables, each table must contain the named columns.

- A `REVOKE` statement at the column level has no effect if privileges were granted at the table level.

- After revoking `SELECT` on a change stream from a role, revoke `EXECUTE` on the change stream's read function from that role.

- Revoking `SELECT` on a change stream doesn't revoke any privileges on the table that it tracks.

#### Examples

The following example revokes `SELECT` on the `employees` table from the role `hr_rep` .

```
REVOKE SELECT ON TABLE employees FROM hr_rep;
```

The next example is the same as the previous example, but with the optional `TABLE` keyword omitted.

```
REVOKE SELECT ON employees FROM hr_rep;
```

The next example revokes `SELECT` on a subset of columns of the `contractors` table from the role `hr_rep` .

```
REVOKE SELECT(name, address, phone) ON TABLE contractors FROM hr_rep;
```

The next example shows revoking both table-level and column-level privileges in a single statement.

```
REVOKE SELECT, UPDATE(location) ON TABLE employees FROM hr_manager;
```

The next example revokes column-level grants on two tables. Both tables must contain the `name` , `level` , and `location` columns.

```
REVOKE
  SELECT (name, level, location),
  UPDATE (location)
ON TABLE employees, contractors
```

The next example revokes `INSERT` on a subset of columns.

```
REVOKE INSERT(name, cost_center, location, manager) ON TABLE employees FROM
hr_manager;
```

The following example revokes the database role `pii_access` from the `hr_manager` and `hr_director` database roles. The `hr_manager` and `hr_director` roles lose any privileges that they inherited from `pii_access` .

```
REVOKE pii_access FROM hr_manager, hr_director;
```
