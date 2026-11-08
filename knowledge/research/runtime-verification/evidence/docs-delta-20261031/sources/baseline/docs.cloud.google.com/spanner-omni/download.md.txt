---
name: documents/docs.cloud.google.com/spanner-omni/download
uri: https://docs.cloud.google.com/spanner-omni/download
title: Download Spanner Omni
description: A downloadable, self-managed version of Spanner.
data_source: docs.cloud.google.com
---

> [Download Spanner Omni](https://docs.cloud.google.com/spanner-omni/download) to try it out at no charge. If you decide you want to use Spanner Omni for production use, [contact Google](https://cloud.google.com/consulting/spanner-omni) to learn about acquiring a license for the [commercial edition](https://docs.cloud.google.com/spanner-omni/editions-overview#commercial-edition) .

This document provides links and instructions to download Spanner Omni components, including container images, Helm charts, and standalone binaries.

## Container images

Artifact Registry hosts Spanner Omni container images at `us-docker.pkg.dev/spanner-omni/images/` .

The following container images are available:

- `spanner-omni` : includes Spanner Omni, the Spanner Omni CLI, and the Spanner Omni console.

- `spanner-omni-server` : includes Spanner Omni and the Spanner Omni CLI.

- `spanner-omni-ui` : includes only the Spanner Omni console.

Specify the exact version tag when pulling a container image.

### Container image versions

The following table lists the available container image versions.

| Version tag   | Release date       |
|---------------|--------------------|
| `2026.r4-lts` | September 30, 2026 |

For example, to download the `spanner-omni` image for the 2026.r4-lts release, run the following command:

```
docker pull us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r4-lts
```

## Helm charts

Helm charts for Spanner Omni deployments are hosted in Artifact Registry at `us-docker.pkg.dev/spanner-omni/charts` and are available as open source in the [Spanner Omni GitHub repository](https://github.com/GoogleCloudPlatform/spanner-omni) . These charts support various deployment topologies, from single-server to multi-cluster. For more information, see [Create a Helm chart configuration for Spanner Omni](https://docs.cloud.google.com/spanner-omni/create-helm-configuration) .

Specify the exact version tag when you download a chart.

### Helm chart versions

The following table lists the available Helm chart versions.

| Version tag | Release date       |
|-------------|--------------------|
| `1.0.0`     | September 30, 2026 |

For example, to download the Helm chart for version 1.0.0 from Artifact Registry, run the following command:

```
helm pull oci://us-docker.pkg.dev/spanner-omni/charts/spanner-omni --version 1.0.0
```

To clone the open-source Helm charts from GitHub, run the following command:

```
git clone https://github.com/GoogleCloudPlatform/spanner-omni.git
```

## Standalone binaries

The following Google Cloud bucket contains standalone binaries: `https://storage.googleapis.com/spanner-omni/` .

Each release is in a folder named after the version tag.

### Spanner Omni packages

The following table lists the available Spanner Omni server and component packages.

| Filename                                              | Description                                                                     |
|-------------------------------------------------------|---------------------------------------------------------------------------------|
| `spanner-omni-2026.r4-lts-linux-x86_64.tar.gz`        | Spanner Omni server, Spanner Omni CLI, and Spanner Omni console for Linux (x86) |
| `spanner-omni-server-2026.r4-lts-linux-x86_64.tar.gz` | Spanner Omni server and Spanner Omni CLI for Linux (x86)                        |
| `spanner-omni-ui-2026.r4-lts-linux-x86_64.tar.gz`     | Spanner Omni console for Linux (x86)                                            |

### Spanner Omni CLI binaries

The following table lists the available Spanner Omni CLI binaries.

| Filename                                            | Description                           |
|-----------------------------------------------------|---------------------------------------|
| `spanner-omni-cli-2026.r4-lts-darwin-arm.tar.gz`    | Spanner Omni CLI, Mac (M1, M2 and M3) |
| `spanner-omni-cli-2026.r4-lts-darwin-x86_64.tar.gz` | Spanner Omni CLI for Mac (x86)        |
| `spanner-omni-cli-2026.r4-lts-linux-arm.tar.gz`     | Spanner Omni CLI for Linux (ARM)      |
| `spanner-omni-cli-2026.r4-lts-linux-x86_64.tar.gz`  | Spanner Omni CLI for Linux (x86)      |

For example, to download the current version of the Spanner Omni CLI for Linux (x86), run the following command:

```
# Download CLI for Linux (x86)
curl -O https://storage.googleapis.com/spanner-omni/2026.r4-lts/spanner-omni-cli-2026.r4-lts-linux-x86_64.tar.gz
```
