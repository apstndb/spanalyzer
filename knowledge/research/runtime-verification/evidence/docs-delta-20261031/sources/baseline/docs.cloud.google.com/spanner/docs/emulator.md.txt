---
name: documents/docs.cloud.google.com/spanner/docs/emulator
uri: https://docs.cloud.google.com/spanner/docs/emulator
title: Emulate Spanner locally
description: Develop and test applications locally using the Spanner emulator. Run it via the Google Cloud CLI or Docker.
data_source: docs.cloud.google.com
---

The [gcloud CLI](https://docs.cloud.google.com/sdk) provides a local, in-memory emulator to develop and test your applications. Because the emulator stores data only in memory, it loses all state, including data, schema, and configs, on restart. The emulator offers the same APIs as the Spanner production service and serves local development and testing, not production deployments.

The emulator supports both the GoogleSQL and PostgreSQL dialects. It supports all languages of the [client libraries](https://docs.cloud.google.com/spanner/docs/emulator#client-libraries) . You can also use the emulator with the [Google Cloud CLI](https://docs.cloud.google.com/sdk/gcloud) and [REST APIs](https://docs.cloud.google.com/spanner/docs/reference/rest) .

The emulator is also available as an open source project in [GitHub](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator) .

> **Note:** To explore Spanner using a 90-day free trial instance, see [Create a Spanner free trial instance](https://docs.cloud.google.com/spanner/docs/free-trial-quickstart) .

## Limitations and differences

The emulator doesn't support the following:

- TLS/HTTPS, authentication, Identity and Access Management (IAM), permissions, or roles.
- In the `PLAN` or `PROFILE` [query modes](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/QueryMode) , the query plan that is returned is empty.
- The [`ANALYZE` statement](https://docs.cloud.google.com/spanner/docs/query-optimizer/overview#construct-statistics-package) . The emulator accepts but ignores it.
- Any of the [audit logging](https://docs.cloud.google.com/spanner/docs/audit-logging) and monitoring tools.
- Database drop protection. The emulator accepts the `enable_drop_protection` field, but it allows databases to be dropped even if this property is enabled.

The emulator also differs from the Spanner production service in the following ways:

- Error messages might differ between the emulator and the production service.
- The emulator's performance and scalability don't compare to the production service.
- Read-write transactions and schema changes lock the entire database for exclusive access until completion.
- The emulator supports [Partitioned DML](https://docs.cloud.google.com/spanner/docs/dml-partitioned) and [`partitionQuery`](https://docs.cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases.sessions/partitionQuery) , but it does not verify that statements are [partitionable](https://docs.cloud.google.com/spanner/docs/dml-partitioned#partitionable-idempotent) . This means a partitioned DML or `partitionQuery` statement might run in the emulator, but fail in the production service with the non-partitionable statement error.

For a complete list of APIs and features that are supported, unsupported, and partially supported, see the [README](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/master/README.md) file in GitHub.

## Options for running the emulator

There are two common ways to run the emulator:

- [gcloud CLI](https://docs.cloud.google.com/spanner/docs/emulator#emulator-for-gcloud)
- [Docker](https://docs.cloud.google.com/spanner/docs/emulator#install-docker)

Choose the way that is appropriate for your application development and test workflow.

> **Note:** For more ways to run the emulator, see the [README](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/blob/master/README.md#quickstart) .

### Run the emulator using gcloud CLI

To run the emulator using the Google Cloud CLI:

1.  Install the `cloud-spanner-emulator` component:

    ```
    gcloud components install cloud-spanner-emulator
    ```

    If gcloud CLI is already installed, run the following command to ensure all of its components are updated:

    ```
    gcloud components update
    ```

2.  Start the emulator:

    ```
    gcloud emulators spanner start
    ```

    The emulator uses two local endpoints:

    - `localhost:9010` for gRPC requests
    - `localhost:9020` for REST requests

    > **Note:** When starting the emulator, you might see several warning messages about proto registration conflicts (such as `WARNING: proto: file "google/rpc/status.proto" is already registered` ) or log messages before initialization ( `WARNING: All log messages before absl::InitializeLog() is called are written to STDERR` ). These warnings are expected and can be safely ignored.

### Run the emulator using Docker

To run the emulator using Docker:

1.  Install [Docker](https://www.docker.com/products/docker-desktop) on your system and make it available on the system path.

2.  Get the latest emulator image:

    ```
    docker pull gcr.io/cloud-spanner-emulator/emulator
    ```

3.  Run the emulator in Docker:

    ```
    docker run -p 9010:9010 -p 9020:9020 gcr.io/cloud-spanner-emulator/emulator
    ```

    This command runs the emulator and maps the ports in the container to the same ports on your local host. The emulator uses two local endpoints: `localhost:9010` for gRPC requests and `localhost:9020` for REST requests.

    > **Note:** When starting the emulator, you might see several warning messages about proto registration conflicts (such as `WARNING: proto: file "google/rpc/status.proto" is already registered` ) or log messages before initialization ( `WARNING: All log messages before absl::InitializeLog() is called are written to STDERR` ). These warnings are expected and can be safely ignored.

## Configure gcloud CLI to use the emulator

To use the emulator with gcloud CLI, disable authentication and override the endpoint. Create a separate [gcloud CLI configuration](https://docs.cloud.google.com/sdk/docs/configurations) to switch quickly between the emulator and the production service.

1.  Create and activate an emulator configuration:

    ```
    gcloud config configurations create emulator
    gcloud config set auth/disable_credentials true
    gcloud config set project your-project-id
    gcloud config set api_endpoint_overrides/spanner http://localhost:9020/
    ```

    > **Note:** When you run `gcloud config set api_endpoint_overrides/spanner` , you might receive a warning that the property value is associated with a domain outside of the current config universe. Type `y` or `yes` to confirm the prompt and proceed.

2.  After configured, gcloud CLI sends your commands to the emulator instead of the production service. Verify this by creating an instance with the emulator's instance config:

    ```
    gcloud spanner instances create test-instance \
      --config=emulator-config --description="Test Instance" --nodes=1
    ```

### Switch configurations

To switch between the emulator and your default configuration, run:

```
# To switch to default (production) configuration:
gcloud config configurations activate default

# To switch back to emulator configuration:
gcloud config configurations activate emulator
```

## Use the client libraries with the emulator

You can use [supported versions](https://docs.cloud.google.com/spanner/docs/emulator#supported-versions) of the client libraries with the emulator by setting the `SPANNER_EMULATOR_HOST` environment variable. There are many ways to do this. For example:

> **Important:** If you are using C#, see the [additional instructions for C#](https://docs.cloud.google.com/spanner/docs/emulator#cs) .

### Linux/macOS

```
export SPANNER_EMULATOR_HOST=localhost:9010
```

### Windows

```
set SPANNER_EMULATOR_HOST=localhost:9010
```

Or with [gcloud env-init](https://docs.cloud.google.com/sdk/gcloud/reference/emulators/spanner/env-init) :

### Linux/macOS

```
$(gcloud emulators spanner env-init)
```

### Windows

```
gcloud emulators spanner env-init > set_vars.cmd && set_vars.cmd
```

When your application starts, the client library automatically checks for `SPANNER_EMULATOR_HOST` and connects to the emulator if it's running.

Once `SPANNER_EMULATOR_HOST` is set, you can test the emulator by following the Getting Started guides. Ignore the instructions related to project creation, authentication, and credentials since these aren't needed to use the emulator.

- [Getting Started in C++](https://docs.cloud.google.com/spanner/docs/getting-started/cpp)

- [Getting Started in C#](https://docs.cloud.google.com/spanner/docs/getting-started/csharp) . You must set connection string options. See [additional instructions for C#](https://docs.cloud.google.com/spanner/docs/emulator#cs) .

- [Getting Started in Go](https://docs.cloud.google.com/spanner/docs/getting-started/go)

- [Getting Started in Java](https://docs.cloud.google.com/spanner/docs/getting-started/java)

- [Getting Started in Node.js](https://docs.cloud.google.com/spanner/docs/getting-started/nodejs)

- [Getting Started in PHP](https://docs.cloud.google.com/spanner/docs/getting-started/php)

- [Getting Started in Python](https://docs.cloud.google.com/spanner/docs/getting-started/python)

- [Getting Started in Ruby](https://docs.cloud.google.com/spanner/docs/getting-started/ruby)

### Supported versions

The following table lists the versions of the [client libraries](https://docs.cloud.google.com/spanner/docs/reference/libraries) that support the emulator.

| Client library | Minimum version                                                        |
|----------------|------------------------------------------------------------------------|
| C++            | [v0.9.x+](https://github.com/googleapis/google-cloud-cpp/releases)     |
| C#             | [v3.1.0+](https://github.com/googleapis/google-cloud-dotnet/releases)  |
| Go             | [v1.5.0+](https://github.com/googleapis/google-cloud-go/releases)      |
| Java           | [v1.51.0+](https://github.com/googleapis/google-cloud-java/releases)   |
| Node.js        | [v4.5.0+](https://github.com/googleapis/google-cloud-node/releases)    |
| PHP            | [v1.25.0+](https://github.com/googleapis/google-cloud-php/releases)    |
| Python         | [v1.15.0+](https://github.com/googleapis/google-cloud-python/releases) |
| Ruby           | [v1.13.0+](https://github.com/googleapis/google-cloud-ruby/releases)   |

### Additional instructions for C

For the C# client library, specify the [`emulatordetection`](https://docs.cloud.google.com/dotnet/docs/reference/Google.Api.Gax/latest/Google.Api.Gax.EmulatorDetection) option in the [connection string](https://docs.cloud.google.com/dotnet/docs/reference/Google.Cloud.Spanner.Data/latest/connection_string) . Unlike the other client libraries, C# ignores the `SPANNER_EMULATOR_HOST` environment variable by default. The following example shows the connection string:

```
var builder = new SpannerConnectionStringBuilder
{
    DataSource = $"projects/{projectId}/instances/{instanceId}/databases/{databaseId}",
    EmulatorDetection = "EmulatorOnly"
};
```
