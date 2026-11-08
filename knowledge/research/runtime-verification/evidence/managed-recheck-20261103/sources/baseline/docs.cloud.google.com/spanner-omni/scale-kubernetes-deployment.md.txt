---
name: documents/docs.cloud.google.com/spanner-omni/scale-kubernetes-deployment
uri: https://docs.cloud.google.com/spanner-omni/scale-kubernetes-deployment
title: Scale a Kubernetes deployment
description: Learn how to scale Spanner Omni deployments on Kubernetes, including vertical resource grading, horizontal scaling of non-root servers, zone management, and disk volume expansion.
data_source: docs.cloud.google.com
---

Kubernetes environments let you scale your database resources dynamically as workload demands change. Use these scaling procedures if you deployed Spanner Omni using the Helm chart.

## Before you begin

Before you scale your Kubernetes deployment, you must do the following:

- [Create Spanner Omni on Kubernetes](https://docs.cloud.google.com/spanner-omni/deploy-on-kubernetes) .

- Download and install the [Spanner Omni CLI](https://docs.cloud.google.com/spanner-omni/cli-quickstart#step-1-download-install-cli) .

- Install [Helm](https://helm.sh/) and create a [Helm chart configuration](https://docs.cloud.google.com/spanner-omni/create-helm-configuration) .

As a best practice, we recommend scaling vertically up to at least 32 GB of memory per server before you add more servers to scale horizontally.

## Limitations

Scaling in Kubernetes has the following limitations:

- **Non-root servers only** : Horizontal scaling is supported for non-root instances. Root server scaling isn't supported.

- **StatefulSet storage constraints** : Because Kubernetes `volumeClaimTemplates` are immutable, you can't expand pod disks using a single `helm upgrade` command. Instead, scaling storage requires manual volume expansion steps.

## Scale vertically

To adjust the CPU or memory resources for your servers, update your configuration using a Helm upgrade:

```
helm upgrade spanner-omni HELM_CHART_PATH \
  --version VERSION \
  --reuse-values \
  --set resources.cpu=CPU_CORES \
  --set resources.memory=MEMORY_LIMIT \
  -n NAMESPACE
```

Replace the following:

- `HELM_CHART_PATH` : The path to your Helm chart, for example, `oci://us-docker.pkg.dev/spanner-omni/charts/spanner-omni` .
- `VERSION` : The version of the Helm chart, for example, `1.0.0` .
- `CPU_CORES` : The number of vCPU cores to assign to each server pod, for example, `8` .
- `MEMORY_LIMIT` : The RAM limit for each server pod, for example, `32Gi` .
- `NAMESPACE` : The Kubernetes namespace of the deployment, for example, `spanner-ns` .

## Scale horizontally

To scale horizontally, add more servers to your deployment. Horizontal scaling is supported for non-root servers.

### Add non-root servers

To add non-root servers, increase the replica count in the Helm chart configuration. You can scale all zones uniformly, or scale a specific zone.

### Scale uniformly

To scale each zone in the deployment to 15 servers, run the following command:

```
helm upgrade spanner-omni HELM_CHART_PATH \
  --version VERSION \
  --reuse-values \
  --set deployment.replicasPerZone=REPLICAS \
  -n NAMESPACE
```

Replace the following:

- `HELM_CHART_PATH` : The path to your Helm chart—for example, `oci://us-docker.pkg.dev/spanner-omni/charts/spanner-omni` .
- `VERSION` : The version of the Helm chart—for example, `1.0.0` .
- `REPLICAS` : The target number of server replicas per zone—for example, `15` .
- `NAMESPACE` : The Kubernetes namespace—for example, `spanner-ns` .

### Scale a specific zone

If your initial deployment configured different server counts for individual zones, you can target a single zone. For example, to increase the replicas of the first zone inside the first location to 15, run the following command:

```
helm upgrade spanner-omni HELM_CHART_PATH \
  --version VERSION \
  --reuse-values \
  --set locations[0].zones[0].replicas=REPLICAS \
  -n NAMESPACE
```

Replace `REPLICAS` with your target zone replica count—for example, `15` .

To verify that the new servers successfully joined the deployment, query the Spanner Omni CLI to list deployment servers or view your [Grafana dashboard](https://docs.cloud.google.com/spanner-omni/grafana-dashboards) .

```
spanner deployment servers list \
  --zone=ZONE \
  --deployment-endpoint=ENDPOINT
```

Replace the following:

- `ZONE` : The zone you want to list—for example, `us-central1-a` .
- `ENDPOINT` : The external endpoint of your deployment—for example, `${ENDPOINT}:15000` .

### Remove non-root servers

Scaling down servers requires additional steps because the system must safely relocate data partitions away from decommissioned servers. Because Kubernetes StatefulSets remove pods from highest to lowest index, you must target the highest-index non-root servers for removal first.

To scale down the number of servers, perform the following steps:

1.  List the servers in your zone to identify candidates for removal:

    ```
    spanner deployment servers list \
      --zone=ZONE \
      --deployment-endpoint=ENDPOINT
    ```

    Example output:

    ```
    NAME                                                          HOST                        PORT_BASE  ROOT  STATE
    zones/us-central1-a/servers/spanner-a-0.pod.spanner-ns:15000  spanner-a-0.pod.spanner-ns  15000      true  -
    zones/us-central1-a/servers/spanner-a-1.pod.spanner-ns:15000  spanner-a-1.pod.spanner-ns  15000      -     -
    ```

    Delete the decommissioned non-root server with the highest index (for example, `spanner-a-1.pod.spanner-ns:15000` ):

    ```
    spanner deployment servers delete SERVER_NAME \
      --zone=ZONE \
      --deployment-endpoint=ENDPOINT
    ```

    Replace `SERVER_NAME` with the server identifier—for example, `spanner-a-1.pod.spanner-ns:15000` .

    Check the server list until the target server is removed from the listing. After deletion, the server transitions to an unhealthy state in the system and is removed from the active service path.

2.  Scale down the Helm deployment by running a Helm upgrade command to match your target replica count. For example, to decrease the replicas per zone to `1` pod, run the following command:

    ```
    helm upgrade spanner-omni HELM_CHART_PATH \
      --version VERSION \
      --reuse-values \
      --set deployment.replicasPerZone=REPLICAS \
      -n NAMESPACE
    ```

    Replace `REPLICAS` with the updated replica count—for example, `1` .

3.  Delete the Kubernetes persistent volume claims (PVCs) associated with the removed pods. To prevent accidental data loss, Helm and Kubernetes don't automatically delete PVCs when scaling down a StatefulSet. Manually delete the PVCs to completely reclaim the storage:

    ```
    kubectl delete pvc LOGS_PVC DATA_PVC -n NAMESPACE
    ```

    For example, to delete the logs and data volumes for `spanner-a-1` in the namespace `spanner-ns` :

    ```
    kubectl delete pvc logs-volume-spanner-a-1 data-volume-spanner-a-1 -n spanner-ns
    ```

> **Note:** This procedure remains valid if a server has already been lost or removed out of order (for instance, if a pod was deleted and its associated PVC was cleared first). You can still invoke the `spanner deployment servers delete` command to safely purge the server from the Spanner Omni registry.

## Add a zone

Adding a new zone to your deployment increases availability and safeguards your database against single-zone outages.

For example, the following command initializes a running, single-zone deployment on Google Kubernetes Engine (GKE) in the `us-east1-b` zone of the `us` region:

```
helm upgrade --install spanner-omni HELM_CHART_PATH \
  --version VERSION \
  --set resources.cpu=2 \
  --set resources.memory=8Gi \
  --set global.platform=gke \
  --set-json 'locations=[{"name":"us","zones":[{"name":"us-east1-b","shortName":"east-b"}]}]' \
  -n NAMESPACE
```

Add the zone `us-east1-c` to this configuration by performing the following steps:

1.  Launch the pods in the new zone by running the `helm upgrade` command and passing an updated JSON block that includes the new location zone:

    ```
    helm upgrade spanner-omni HELM_CHART_PATH \
      --version VERSION \
      --reuse-values \
      --set-json 'locations=[{"name":"us","zones":[{"name":"us-east1-b","shortName":"east-b"},{"name":"us-east1-c","shortName":"east-c"}]}]' \
      -n NAMESPACE
    ```

2.  Add the new zone using the Spanner Omni CLI. Wait for the root server pods in the newly created zone to enter the `Running` and ready state. Then, execute the zone creation command:

    ```
    spanner deployment zones create NEW_ZONE \
      --location=LOCATION \
      --root-servers=ROOT_SERVERS_LIST \
      --deployment-endpoint=ENDPOINT
    ```

    Replace the following:

    - `NEW_ZONE` : The identifier of the zone to add—for example, `us-east1-c` .
    - `LOCATION` : The deployment location—for example, `us` .
    - `ROOT_SERVERS_LIST` : A comma-separated list of root server endpoints in the new zone—for example, `spanner-east-c-0.pod.spanner-ns:15000,spanner-east-c-1.pod.spanner-ns:15000,spanner-east-c-2.pod.spanner-ns:15000` .
    - `ENDPOINT` : The external deployment endpoint—for example, `${ENDPOINT}:15000` .

3.  Wait for the zone creation to complete. Replication of existing database schemas and tables to a new zone takes time. Monitor the progress of the zone synchronization by listing the deployment zones:

    ```
    spanner deployment zones list --deployment-endpoint=ENDPOINT
    ```

## Remove a zone

You can decommission an active zone from your multi-zone deployment to reduce resources or align with topology changes.

Remove the zone `us-east1-c` created in the previous section by performing the following steps:

1.  Delete the zone and initiate the zone tear-down inside Spanner Omni:

    ```
    spanner deployment zones delete ZONE --deployment-endpoint=ENDPOINT
    ```

    Replace `ZONE` with the zone to remove—for example, `us-east1-c` .

2.  Verify that the zone was removed. Run a list command and wait until the zone no longer appears in the output:

    ```
    spanner deployment zones list --deployment-endpoint=ENDPOINT
    ```

3.  Remove the servers from the Kubernetes cluster by running a helm upgrade command and passing an updated locations JSON block that excludes the removed zone:

    ```
    helm upgrade spanner-omni HELM_CHART_PATH \
      --version VERSION \
      --reuse-values \
      --set-json 'locations=[{"name":"us","zones":[{"name":"us-east1-b","shortName":"east-b"}]}]' \
      -n NAMESPACE
    ```

## Scale storage

Because Kubernetes `volumeClaimTemplates` are immutable, you cannot scale up pod storage capacities using the `helm upgrade` command directly. Instead, you must perform a manual volume expansion. For more information, see the [GKE StatefulSet Volume Expansion guide](https://docs.cloud.google.com/kubernetes-engine/docs/how-to/persistent-volumes/volume-expansion#managing-volume-expansions-in-statefulsets) .

To expand your disk storage, perform the following steps:

1.  Define the parameters for the volume expansion as environment variables in your terminal:

    ```
    NEW_SIZE="NEW_SIZE"
    NAMESPACE="NAMESPACE"
    RELEASE_NAME="spanner-omni"
    STATEFULSET_NAMES="STATEFULSET_NAME_1 STATEFULSET_NAME_2 STATEFULSET_NAME_3"
    HELM_CHART_PATH="HELM_CHART_PATH"
    VERSION="VERSION"
    ```

    Replace the following:

    - `NEW_SIZE` : The target storage capacity size—for example, `200Gi` .
    - `NAMESPACE` : The Kubernetes namespace—for example, `spanner-ns` .
    - `STATEFULSET_NAME_1` , `STATEFULSET_NAME_2` , ...: The names of the [StatefulSets](https://docs.cloud.google.com/kubernetes-engine/docs/concepts/statefulset) in your deployment, typically corresponding to the short names of your zones (for example, `spanner-east-b spanner-east-c` ).
    - `HELM_CHART_PATH` : The path to your Helm chart—for example, `oci://us-docker.pkg.dev/spanner-omni/charts/spanner-omni` .
    - `VERSION` : The version of the Helm chart—for example, `1.0.0` .

2.  Run the commands to patch the PVCs, delete the StatefulSets (leaving the backend pods intact), and upgrade the Helm deployment:

    ```
    # Patch all associated PVCs directly.
    for pvc in $(kubectl get pvc -n $NAMESPACE \
      -l app.kubernetes.io/instance=$RELEASE_NAME \
      -o name | grep "data-volume"); do
      kubectl patch $pvc -n $NAMESPACE -p "{\"spec\":{\"resources\":{\"requests\":{\"storage\":\"$NEW_SIZE\"}}}}"
    done

    # Delete the StatefulSet while leaving backend pods intact (orphan cascade).
    kubectl delete statefulset $STATEFULSET_NAMES -n $NAMESPACE --cascade=orphan

    # Run Helm upgrade to align the templates with the expanded size.
    helm upgrade $RELEASE_NAME $HELM_CHART_PATH \
      --version $VERSION \
      --reuse-values \
      --set storage.data.size=$NEW_SIZE \
      -n $NAMESPACE
    ```

## Next steps

- Learn how to [maintain a deployment](https://docs.cloud.google.com/spanner-omni/maintain-deployment) .
