---
name: documents/docs.cloud.google.com/spanner-omni/setup
uri: https://docs.cloud.google.com/spanner-omni/setup
title: Set up Spanner Omni
description: Learn how to set up Spanner Omni using a TAR file or a Docker container.
data_source: docs.cloud.google.com
---

> [Download Spanner Omni](https://docs.cloud.google.com/spanner-omni/download) to try it out at no charge. If you decide you want to use Spanner Omni for production use, [contact Google](https://cloud.google.com/consulting/spanner-omni) to learn about acquiring a license for the [commercial edition](https://docs.cloud.google.com/spanner-omni/editions-overview#commercial-edition) .

This page describes how to start Spanner Omni in single-server mode. You can either install it using a TAR file or run it as a container using Docker.

## Before you begin

Before you set up your Spanner Omni instance, ensure you meet the following requirements:

- Verify your machine meets the [system requirements](https://docs.cloud.google.com/spanner-omni/system-requirements) .

- Ensure you have `sudo` access in your environment to set up the Spanner Omni binary.

- Ensure [Docker](https://docs.docker.com/get-docker/) is installed on your machine for container-based installations.

## Run using Docker

We recommend storing Spanner data in a [Docker volume](https://docs.docker.com/storage/volumes/) to ensure data persistence if you delete the container.

1.  Create a Docker volume:

    ```
    docker volume create spanner
    ```

2.  Start the Spanner Omni server container. Replace ` VERSION_TAG ` with the Spanner Omni version you want to use. The current version is `2026.r4-lts` .

    ```
    docker run -d --network host \
        --name spanneromni \
        -v "spanner:/spanner" \
        us-docker.pkg.dev/spanner-omni/images/spanner-omni:VERSION_TAG \
        start-single-server
    ```

    The `--network host` flag maps the Spanner Omni ports to the host machine.

3.  Verify that the container is running:

    ```
    docker ps
    ```

    Check the **STATUS** field in the output to ensure the container is healthy.

### Interact with the containerized server

You can use `docker exec` to run Spanner Omni CLI commands inside the container:

1.  Create a database:

    ```
    docker exec -it spanneromni /google/spanner/bin/spanner databases create DATABASE_NAME
    ```

2.  Open the SQL shell:

    ```
    docker exec -it spanneromni /google/spanner/bin/spanner sql --database=DATABASE_NAME
    ```

3.  List databases:

    ```
    docker exec -it spanneromni /google/spanner/bin/spanner databases list
    ```
