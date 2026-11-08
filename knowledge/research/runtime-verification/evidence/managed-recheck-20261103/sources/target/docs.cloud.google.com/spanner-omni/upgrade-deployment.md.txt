---
name: documents/docs.cloud.google.com/spanner-omni/upgrade-deployment
uri: https://docs.cloud.google.com/spanner-omni/upgrade-deployment
title: Upgrade a Spanner Omni deployment
description: Learn how to upgrade a Spanner Omni deployment to a later version using an asynchronous, phased rollout.
data_source: docs.cloud.google.com
---

This document describes how to upgrade a Spanner Omni deployment from an earlier version to a later version.

Spanner Omni uses an asynchronous, phased rollout state machine to help ensure safe upgrades without service disruption. Upgrading in phases lets you do the following:

- [Apply database schema updates](https://docs.cloud.google.com/spanner-omni/upgrade-deployment#prepare-schema) .
- [Verify binary compatibility](https://docs.cloud.google.com/spanner-omni/upgrade-deployment#update-container-image) during a rolling restart.
- [Roll back to the earlier version](https://docs.cloud.google.com/spanner-omni/upgrade-deployment#rollback-upgrade) if you detect errors before finalization.

## Supported upgrade paths

- You can update an LTS version to only the next LTS version. You can't skip a version when updating to a newer LTS version.

- You can skip a version when you upgrade from one short-term support (STS) version to another STS version if there isn't an LTS release between their release dates.

For more information, see [Version support policy](https://docs.cloud.google.com/spanner-omni/support#version-support-policy) .

## Upgrade workflow

The Spanner Omni upgrade process consists of multiple sequential phases:

1.  **Schema phase** : Prepares the deployment by running internal database schema migrations using the target binary version. Schema migrations can't be rolled back, but they are backward-compatible with the earlier binary version.
2.  **Binary phase** : Updates the server binary or container image across all servers in the deployment using a rolling restart. If you detect errors, you can roll back the binary or container image to the earlier version at any point before finalization begins.
3.  **Enable features phase** : Enables version-compatible features and starts after you upgrade the binary on all servers. This phase might not apply if the upgrade doesn't include version-compatible features, and it can require multiple rounds if features are interdependent. For optional phases that support rollback, you can initiate a rollback using the Spanner Omni CLI.
4.  **Finalize phase** : Finalizes the rollout, sealing the target version. After the finalize phase begins, rollbacks aren't possible.

## Before you begin

Before you upgrade your Spanner Omni deployment, ensure you meet the following prerequisites:

- You have a running Spanner Omni deployment. For more information, see [Create a deployment on Kubernetes](https://docs.cloud.google.com/spanner-omni/deploy-on-kubernetes) or [Create a deployment on VMs](https://docs.cloud.google.com/spanner-omni/deploy-on-vms) .
- You downloaded and installed the [Spanner Omni CLI](https://docs.cloud.google.com/spanner-omni/cli-quickstart#step-1-download-install-cli) .
- You have network access to all Spanner Omni internal ports (TCP 15000 to 15027) from the machine where you run the CLI commands, or network access to your deployment endpoint.
- If your deployment uses Transport Layer Security (TLS) or mutual TLS (mTLS) encryption, ensure valid certificates are available in your base directory under `BASE_DIR `` /tls` or mounted in your cluster.
- You identified your target release:
  - For VM deployments: download the target release package `spanner-omni-server- `` TARGET_VERSION `` .tar.gz` .
  - For Helm and Kubernetes deployments: identify the target container image in Artifact Registry, such as `us-docker.pkg.dev/spanner-omni/images/spanner-omni: `` TARGET_VERSION` .
- Administrators of Spanner Omni deployments should schedule maintenance windows for upgrades. Your organization should avoid deployments during upgrade maintenance windows.

## Step 1: Prepare the schema upgrade

To initiate the upgrade, prepare the rollout for the target version. In this phase, Spanner Omni runs internal database schema migrations while existing servers continue serving traffic.

Select the tab for your deployment environment:

### VM

In VM deployments, download and extract the target release package, and then use the extracted Spanner Omni CLI to initiate rollout preparation.

> **Important** : The target Spanner Omni release package includes both the `spanner` CLI and the `spanner_server` binary (typically under `bin/` ). Download and extract the target release package on the host before running the preparation phase.

1.  Sign in to a server in your deployment that has network access to all Spanner Omni internal ports (TCP 15000 to 15027).

2.  Download and extract the target release package:

    ```
    tar -xzf spanner-omni-server-TARGET_VERSION.tar.gz -C EXTRACT_DIR
    ```

    Replace the following:

    - `TARGET_VERSION` : The target version to upgrade to, for example, `2026.r5` .
    - `EXTRACT_DIR` : The directory where you extract the release package, for example, `/tmp/target_spanner/` .

3.  Initiate rollout preparation by running the `rollouts prepare` command from the extracted CLI:

    ```
    EXTRACT_DIR/bin/spanner deployment rollouts prepare \
      --target-server-binary=EXTRACT_DIR/bin/spanner_server \
      --root-server=ROOT_SERVERS \
      --base-dir=BASE_DIR
    ```

    Replace the following:

    - `EXTRACT_DIR` : The extraction directory containing the target `bin/spanner` CLI and `bin/spanner_server` binary.
    - `ROOT_SERVERS` : One root server endpoint or a comma-separated list of multiple root servers, for example, `localhost:15000` or `server1:15000,server2:15000,server3:15000` .
    - `BASE_DIR` : The base directory for Spanner Omni, for example, `/spanner` .
    - If your deployment uses TLS or mTLS encryption, append `--ca-certificate-file= `` CA_CERT_FILE` and `--client-certificate-directory= `` CERT_DIR` pointing to valid certificates.

### Helm

In Helm deployments, schema preparation depends on whether you run a multi-server or single-server deployment:

- **Multi-server deployments (High availability / Production)** : Helm automatically handles schema preparation. When you run `helm upgrade` in [Step 3: Update the binary or container image](https://docs.cloud.google.com/spanner-omni/upgrade-deployment#update-container-image) , the Helm chart triggers the `spanner-prepare-for-upgrade` pre-upgrade hook job to run schema migrations before updating any StatefulSets. Proceed directly to [Step 2: Verify the active rollout state](https://docs.cloud.google.com/spanner-omni/upgrade-deployment#verify-active-rollout) .

- **Single-server deployments ( `deployment.singleServer=true` )** : Because single-server mode binds internal services strictly to the loopback interface ( `127.0.0.1` ), network jobs can't reach them. Run the schema migration locally inside the running pod using an ephemeral debug container with the target container image:

  ```
  kubectl debug pod/POD_NAME -n NAMESPACE \
    --image=us-docker.pkg.dev/spanner-omni/images/spanner-omni:TARGET_VERSION \
    --container=upgrade-prepare -i \
    -- /google/spanner/bin/spanner_server prepare_for_upgrade --root_server=127.0.0.1
  ```

  Replace the following:

  - `POD_NAME` : The name of the server pod, for example, `spanner-a-0` .
  - `NAMESPACE` : The Kubernetes namespace of the deployment, for example, `spanner-ns` .
  - `TARGET_VERSION` : The target version to upgrade to, for example, `2026.r5` .

### Standalone Kubernetes

If you deploy Spanner Omni on Kubernetes without Helm, run a standalone Kubernetes batch job to execute schema migrations against the active root server using the target image.

1.  Create a file named `spanner-prepare-upgrade.yaml` with the following job manifest:

    ```
    apiVersion: batch/v1
    kind: Job
    metadata:
      namespace: NAMESPACE
      name: spanner-prepare-for-upgrade
    spec:
      # Fail fast on the first error to stop the rollout immediately.
      backoffLimit: 0
      template:
        metadata:
          namespace: NAMESPACE
        spec:
          restartPolicy: Never
          containers:
            - name: spanner-upgrade
              image: us-docker.pkg.dev/spanner-omni/images/spanner-omni:TARGET_VERSION
              command: ["/google/spanner/bin/spanner_server"]
              args:
                - "prepare_for_upgrade"
                - "--root_server=ROOT_SERVER_ENDPOINT"
              volumeMounts:
                - name: tls-certs
                  mountPath: "/spanner/tls"
                  readOnly: true
                - name: spanner-data
                  mountPath: /spanner
          volumes:
            - name: tls-certs
              secret:
                secretName: tls-certs
                optional: true
                defaultMode: 256
            - name: spanner-data
              emptyDir: {}
    ```

    Replace the following:

    - `NAMESPACE` : The Kubernetes namespace of the deployment, for example, `spanner-ns` .
    - `TARGET_VERSION` : The target version to upgrade to, for example, `2026.r5` .
    - `ROOT_SERVER_ENDPOINT` : The endpoint of an active root server pod, for example, `spanner-a-0.pod.spanner-ns` .

2.  Apply the manifest to run the preparation job:

    ```
    kubectl apply -f spanner-prepare-upgrade.yaml
    ```

## Step 2: Verify the active rollout state

After the preparation step completes, verify that the rollout was created and inspect its phase status.

> **Note** : In multi-server Helm deployments, Helm creates the rollout during `helm upgrade` in Step 3. You can verify the rollout state during or after running that step.

1.  List the active rollouts to retrieve the rollout ID:

    ```
    spanner deployment rollouts list \
      --deployment-endpoint=DEPLOYMENT_ENDPOINT
    ```

    Replace `DEPLOYMENT_ENDPOINT` with the endpoint of a server in your deployment, for example, `localhost:15000` or `spanner-a-0.pod.spanner-ns:15000` .

    The output is similar to the following:

    ```
    NAME                         STATE          TARGET_VERSION    START_TIME                     END_TIME
    rollouts/1788942172101727    IN_PROGRESS    2026.r3-beta      2026-09-09T08:22:52.101727Z    -
    ```

2.  Inspect the detailed rollout phase status:

    ```
    spanner deployment rollouts describe ROLLOUT_ID \
      --deployment-endpoint=DEPLOYMENT_ENDPOINT
    ```

    Replace the following:

    - `ROLLOUT_ID` : The numeric rollout ID, for example, `1788942172101727` .
    - `DEPLOYMENT_ENDPOINT` : The endpoint of a server in your deployment.

    The output is similar to the following:

    ```
    name: rollouts/1788942172101727
    phases:
        - name: rollouts/1788942172101727/phases/schema
          startTime: "2026-09-09T08:22:52.101727Z"
          state: SUCCEEDED
        - name: rollouts/1788942172101727/phases/binary
          startTime: "2026-09-09T08:23:29.585375Z"
          state: IN_PROGRESS
        - name: rollouts/1788942172101727/phases/finalize
          state: PENDING
    sourceVersion: 2026.r2-beta.3
    startTime: "2026-09-09T08:22:52.101727Z"
    state: IN_PROGRESS
    targetVersion: 2026.r3-beta
    ```

    Verify the following phase statuses before proceeding:

    - `schema` : Shows `SUCCEEDED` , indicating that internal database schema migrations completed.
    - `binary` : Shows `IN_PROGRESS` , indicating that the rollout engine is ready for binary updates.
    - `finalize` : Shows `PENDING` , awaiting completion of the binary phase.

## Step 3: Update the binary or container image

After the schema phase succeeds, update the running server binary or container image across all nodes in the deployment to the target version.

> **Tip** : If you observe errors or performance degradation, you can roll back the binary or container image to the earlier version at any point before finalization. For more information, see [Roll back an upgrade](https://docs.cloud.google.com/spanner-omni/upgrade-deployment#rollback-upgrade) .

Select the tab for your deployment environment:

### VM

In VM deployments, update the `spanner_server` binary across all VMs using a rolling restart:

- **Update one failure domain or zone at a time** : In multi-zone deployments, update servers in one zone and verify stability before updating the next zone. This ensures that the Paxos consensus group retains quorum.
- **Restart progressively** : Restart no more than 5% of servers simultaneously to maintain continuous query availability.
- **Verify server health** : Ensure that all restarted servers are healthy and have rejoined the cluster before updating the next failure domain.

### Helm

In Helm deployments, run `helm upgrade` while preserving your existing configuration values by passing `--reuse-values` or providing a values file:

- **Multi-server deployments (Production / HA)** :

  ```
  helm upgrade spanner-omni oci://us-docker.pkg.dev/spanner-omni/charts/spanner-omni \
    --version CHART_VERSION \
    -n NAMESPACE \
    --reuse-values \
    --set image.tag=TARGET_VERSION \
    --timeout 30m
  ```

  Helm automatically executes Phase 1 using the pre-upgrade hook, and then initiates a sequential, zone-by-zone rolling update of the StatefulSets ( `rollout.staggered: true` ).

- **Single-server deployments ( `deployment.singleServer=true` )** :

  Explicitly pass `--set skipPrepareUpgrade=true` so Helm skips the pre-upgrade hook job, because you already completed Phase 1 locally:

  ```
  helm upgrade spanner-omni oci://us-docker.pkg.dev/spanner-omni/charts/spanner-omni \
    --version CHART_VERSION \
    -n NAMESPACE \
    --reuse-values \
    --set skipPrepareUpgrade=true \
    --set image.tag=TARGET_VERSION \
    --timeout 30m
  ```

Replace the following:

- `CHART_VERSION` : The target Helm chart version, for example, `1.1.0` .
- `NAMESPACE` : The Kubernetes namespace of the deployment, for example, `spanner-ns` .
- `TARGET_VERSION` : The target container image tag, for example, `2026.r5` .

### Standalone Kubernetes

In custom Kubernetes deployments without Helm, update the container image in your StatefulSet or Deployment specifications to `TARGET_VERSION` .

Perform a rolling update across failure domains, updating one zone at a time and restarting no more than 5% of pods simultaneously to preserve quorum.

## Step 4: Verify binary phase progression

After you update all servers or pods to the target version, verify that the binary phase completes successfully.

> **Note** : After all servers in the deployment are updated, Spanner Omni tracks server health over an observation window of up to 15 minutes before marking the `binary` phase as `SUCCEEDED` .

Check the rollout phase status:

```
spanner deployment rollouts describe ROLLOUT_ID \
  --deployment-endpoint=DEPLOYMENT_ENDPOINT
```

Replace the following:

- `ROLLOUT_ID` : The numeric rollout ID, for example, `1788942172101727` .
- `DEPLOYMENT_ENDPOINT` : The endpoint of a server in your deployment.

The output is similar to the following:

```
name: rollouts/1788942172101727
phases:
    - name: rollouts/1788942172101727/phases/schema
      startTime: "2026-09-09T08:22:52.101727Z"
      state: SUCCEEDED
    - name: rollouts/1788942172101727/phases/binary
      startTime: "2026-09-09T08:23:29.585375Z"
      state: SUCCEEDED
    - name: rollouts/1788942172101727/phases/finalize
      state: PENDING
sourceVersion: 2026.r2-beta.3
startTime: "2026-09-09T08:22:52.101727Z"
state: IN_PROGRESS
targetVersion: 2026.r3-beta
```

Confirm that the `binary` phase status changes to `SUCCEEDED` . The overall rollout state remains `IN_PROGRESS` until all remaining phases complete. After `phases/binary` transitions to `SUCCEEDED` , the deployment is ready to proceed to the remaining phases.

## Step 5: Schedule and execute remaining phases

When the binary phase succeeds and you verify that your deployment is stable, schedule the remaining phases for the rollout up to the `finalize` phase. The `finalize` phase (the last phase of a rollout) seals the new version across the deployment. You can run this command from any machine that has network access to the Spanner Omni service by specifying `--deployment-endpoint` .

> **Caution** : Rolling back isn't possible after finalization begins. Finalize the rollout only after you observe the deployment for sufficient time and validate that it's healthy and stable under typical workloads. For more information, see [Roll back an upgrade](https://docs.cloud.google.com/spanner-omni/upgrade-deployment#rollback-upgrade) .

For each remaining phase in the `PENDING` state (such as `enable_features` if present, followed by `finalize` ), complete the following steps:

1.  Schedule the phase:

    ```
    spanner deployment rollouts phases schedule PHASE_NAME \
      --rollout=ROLLOUT_ID \
      --deployment-endpoint=DEPLOYMENT_ENDPOINT
    ```

    Replace the following:

    - `PHASE_NAME` : The name of the phase to schedule, for example, `finalize` or `enable_features` .
    - `ROLLOUT_ID` : The numeric rollout ID, for example, `1788942172101727` .
    - `DEPLOYMENT_ENDPOINT` : The endpoint of a server in your deployment.

    The output indicates that the scheduled phase is `IN_PROGRESS` :

    ```
    name: rollouts/1788942172101727
    phases:
        - name: rollouts/1788942172101727/phases/schema
          startTime: "2026-09-09T08:22:52.101727Z"
          state: SUCCEEDED
        - name: rollouts/1788942172101727/phases/binary
          startTime: "2026-09-09T08:23:29.585375Z"
          state: SUCCEEDED
        - name: rollouts/1788942172101727/phases/finalize
          state: IN_PROGRESS
    sourceVersion: 2026.r2-beta.3
    startTime: "2026-09-09T08:22:52.101727Z"
    state: IN_PROGRESS
    targetVersion: 2026.r3-beta
    ```

2.  Wait for the phase to succeed, and verify its status:

    ```
    spanner deployment rollouts describe ROLLOUT_ID \
      --deployment-endpoint=DEPLOYMENT_ENDPOINT
    ```

    Repeat these steps for each remaining phase until all phases up to and including `finalize` are complete.

    After the final phase ( `finalize` ) completes, all phases and the overall rollout state transition to `SUCCEEDED` :

    ```
    endTime: "2026-09-09T08:45:43.949702Z"
    name: rollouts/1788942172101727
    phases:
        - name: rollouts/1788942172101727/phases/schema
          startTime: "2026-09-09T08:22:52.101727Z"
          state: SUCCEEDED
        - name: rollouts/1788942172101727/phases/binary
          startTime: "2026-09-09T08:23:29.585375Z"
          state: SUCCEEDED
        - name: rollouts/1788942172101727/phases/finalize
          startTime: "2026-09-09T08:45:43.898111Z"
          state: SUCCEEDED
    sourceVersion: 2026.r2-beta.3
    startTime: "2026-09-09T08:22:52.101727Z"
    state: SUCCEEDED
    targetVersion: 2026.r3-beta
    ```

3.  Confirm the completed status in the rollouts list:

    ```
    spanner deployment rollouts list \
      --deployment-endpoint=DEPLOYMENT_ENDPOINT
    ```

    The output confirms that the rollout is complete:

    ```
    NAME                         STATE        TARGET_VERSION    START_TIME                     END_TIME
    rollouts/1788942172101727    SUCCEEDED    2026.r3-beta      2026-09-09T08:22:52.101727Z    2026-09-09T08:45:43.949702Z
    ```

## Roll back an upgrade

If you encounter issues during an upgrade, whether you can roll back the upgrade depends on its current rollout phase:

- **Schema phase** : can't be rolled back. Internal database schema migrations applied during preparation are forward-only and can't be reverted. However, schema migrations are backward-compatible with the source version, allowing earlier server binaries to continue operating normally.
- **Binary phase** : can be rolled back to the earlier version at any point before finalization begins. For more information, see [Roll back the binary or container image](https://docs.cloud.google.com/spanner-omni/upgrade-deployment#rollback-binary) .
- **Optional rollout phases** : for optional phases that support automated rollback (such as feature enablement), initiate the rollback using the Spanner Omni CLI.
- **Finalize phase** : can't be rolled back. After finalization begins, the target version is permanently sealed across the deployment and rollbacks aren't possible.

### Roll back the binary or container image

To roll back the binary phase before finalization begins, perform a rolling restart across all servers or pods to restore the earlier (source) version.

### VM

In VM deployments, roll back the `spanner_server` binary across all VMs:

1.  Deploy the earlier `spanner_server` binary version to your hosts.
2.  Restart servers progressively across failure domains, updating one zone at a time and restarting no more than 5% of servers simultaneously to preserve Paxos quorum.
3.  Verify that all restarted servers are healthy and have rejoined the cluster before proceeding to the next failure domain.

### Helm

In Helm deployments, update the container image tag back to the earlier version:

```
helm upgrade spanner-omni oci://us-docker.pkg.dev/spanner-omni/charts/spanner-omni \
  --version CHART_VERSION \
  -n NAMESPACE \
  --reuse-values \
  --set image.tag=SOURCE_VERSION \
  --timeout 30m
```

Replace the following:

- `CHART_VERSION` : The Helm chart version, for example, `1.1.0` .
- `NAMESPACE` : The Kubernetes namespace of the deployment, for example, `spanner-ns` .
- `SOURCE_VERSION` : The earlier container image tag to revert to, for example, `2026.r2-beta.3` .

### Standalone Kubernetes

In custom Kubernetes deployments without Helm, update the container image in your StatefulSet or Deployment specifications back to `SOURCE_VERSION` .

Perform a rolling update across failure domains, updating one zone at a time and restarting no more than 5% of pods simultaneously to preserve quorum.

### Roll back optional rollout phases

For optional rollout phases that support rollback (such as feature enablement phases), initiate the rollback by running the `rollouts rollback` command:

```
spanner deployment rollouts rollback ROLLOUT_ID \
  --deployment-endpoint=DEPLOYMENT_ENDPOINT
```

Replace the following:

- `ROLLOUT_ID` : The numeric rollout ID.
- `DEPLOYMENT_ENDPOINT` : The endpoint of a server in your deployment.

## What's next

- Learn how to [Maintain a deployment](https://docs.cloud.google.com/spanner-omni/maintain-deployment) .
- Learn how to [Scale a Kubernetes deployment](https://docs.cloud.google.com/spanner-omni/scale-kubernetes-deployment) or [Scale a VM deployment](https://docs.cloud.google.com/spanner-omni/scale-vm-deployment) .
- Learn about [Monitoring overview](https://docs.cloud.google.com/spanner-omni/monitoring-overview) .
