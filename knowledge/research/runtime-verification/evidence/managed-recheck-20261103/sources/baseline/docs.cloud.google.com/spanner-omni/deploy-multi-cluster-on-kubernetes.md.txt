---
name: documents/docs.cloud.google.com/spanner-omni/deploy-multi-cluster-on-kubernetes
uri: https://docs.cloud.google.com/spanner-omni/deploy-multi-cluster-on-kubernetes
title: Create a multi-cluster deployment for Spanner Omni on Kubernetes
description: Learn how to create a Spanner Omni deployment across multiple Kubernetes clusters.
data_source: docs.cloud.google.com
---

This document describes how to set up a Spanner Omni deployment across multiple Kubernetes clusters. You can deploy Spanner Omni with or without TLS encryption. If you use encryption, then Spanner Omni uses Transport Layer Security (TLS) 1.3 to encrypt and authenticate communication within the deployment and with its clients.

A deployment without TLS encryption has the following security risks:

- Anyone can access the deployment if they can reach its IP address.
- There is no network encryption between the client and the server or between the pods.

Because of these risks, avoid a deployment that's not configured with TLS encryption for production environments.

## Before you begin

To prepare for the deployment, complete these requirements:

- Create multiple Kubernetes clusters in each location for your deployment. Clusters can be zonal if you deploy in only one zone in that cluster. Spanner Omni supports Helm chart configuration for Google Kubernetes Engine (GKE) and Amazon Elastic Kubernetes Service (Amazon EKS) environments. Other environments might require custom configurations.

- Get access to the container image hosted in Artifact Registry.

- Install and configure the [`kubectl` command-line tool](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/) and [Helm](https://helm.sh/docs/intro/install/) .

- If you set up the Kubernetes environment on vSphere virtualization platform machines, disable Time Stamp Counter (TSC) virtualization by adding `monitor_control.virtual_rdtsc = FALSE` to the virtual machine's `.vmx` configuration file. This helps TrueTime work correctly.

- Configure the Kubernetes cluster network so that pods in one cluster can connect with pods in another cluster. By default, Spanner Omni uses TCP ports 15000 to 15025 for internal communication. To enable communication between pods in these clusters, open these ports for traffic.

- Configure the cluster DNS to resolve a pod's hostname, such as `spanner-a-0.pod.spanner-ns-r1` , to its IP address. Spanner Omni requires hostname resolution for TCP connections between pods in different clusters.

## Prepare the Helm chart configuration

Create a Helm configuration. For more information, see [Create a Helm configuration](https://docs.cloud.google.com/spanner-omni/create-helm-configuration) .

The number of root servers per zone must be an odd number between one and nine, inclusive, to ensure quorum for consistency. If the number of servers is an even number, deployments might fail. When configuring your zones, designate servers as root servers. We recommend that you use one for development or testing and three for highly available production zones.

Because this is a multi-cluster deployment, ensure your Helm configuration YAML file includes the following:

```
# This is required for a multi-cluster deployment setup.

deployment:
  multiCluster: true
```

## Sample Helm chart configuration for multi-cluster deployment

The following is an example of a Helm chart that's configured for a Spanner Omni multi-cluster deployment. This configuration creates a deployment across three Google Cloud regions: `us-west1` , `us-west2` , and `us-west3` . The deployment is spread over five zones, with each zone representing a replica. The replicas in `us-west1` and `us-west2` are read-write, while the single replica in `us-west3` is a witness replica.

```
# The platform of the deployment
global:
  platform: gke

# This is required for a multi-cluster deployment setup.
deployment:
  multiCluster: true

# Locations and zones where clusters are created for the deployment
locations:
  - name: us-west1
    namespace: spanner-ns-usw1
    zones:
      - name: "us-west1-a"
        shortName: "a"
      - name: "us-west1-b"
        shortName: "b"
  - name: us-west2
    namespace: spanner-ns-usw2
    zones:
      - name: "us-west2-a"
        shortName: "a"
      - name: "us-west2-b"
        shortName: "b"
  - name: us-west3
    namespace: spanner-ns-usw3
    zones:
      - name: "us-west3-a"
        shortName: "a"
        replicaType: WITNESS

# Remaining configuration like storage, resources, isn't included in this sample.
```

## Configure `kubectl` to connect with multiple clusters

Before proceeding, create clusters with `kubectl` contexts. For example, in your [Helm configuration](https://docs.cloud.google.com/spanner-omni/create-helm-configuration) YAML file, you can name the contexts based on the location, such as `ctx-usw1` , `ctx-usw2` , and `ctx-usw3` .

## Configure TLS encryption

If you are setting up a deployment without encryption, skip to [Install a Helm chart for each cluster](https://docs.cloud.google.com/spanner-omni/deploy-multi-cluster-on-kubernetes#install-helm-cluster) .

To configure TLS encryption on a multi-cluster deployment, you must create a Certificate Authority (CA) and generate certificates for each cluster. For more information, see [Add TLS encryption to your Kubernetes deployment](https://docs.cloud.google.com/spanner-omni/deploy-encryption-kubernetes) .

### Generate the certificates

Create the CA and certificates for the server and API. The server and API certificates must include the hosts from all clusters.

#### Create the Spanner Omni server certificate

The Spanner Omni servers use server certificates to encrypt inter-server communication.

To create the server certificate, run the following. Replace ` SERVER_LIST ` with a comma-separated list of Spanner Omni server pod FQDNs or use wildcards.

```
SERVER_NAMES=*.pod.spanner-ns-usw1,*.pod.spanner-ns-usw2,*.pod.spanner-ns-usw3
Spanner Omni CLI certificates create-server --hostnames=${SERVER_NAMES} --ca-certificate-directory certs --output-directory certs
```

#### Create the API certificate

API certificates encrypt communication from systems interacting with the deployment.

To create the API certificate, run the following. Replace ` OMNI_ENDPOINT ` with the service endpoints for each cluster.

```
OMNI_ENDPOINT=spanner.spanner-ns-usw1,spanner.spanner-ns-usw2,spanner.spanner-ns-usw3
Spanner Omni CLI certificates create-server --filename-prefix=api --hostnames=${OMNI_ENDPOINT} --ca-certificate-directory certs --output-directory certs
```

### Push the certificates to each Kubernetes cluster

For each cluster, create the namespace and a generic secret that contains the certificates.

```
# Repeat for each context (for example, ctx-usw1, ctx-usw2, ctx-usw3)
# Replace NAMESPACE with the appropriate namespace for each region

kubectl create namespace <var>NAMESPACE</var> --context <var>CONTEXT</var>
kubectl create secret generic tls-certs \
  --from-file=ca.crt="certs/ca.crt" \
  --from-file=ca-api.crt="certs/ca-api.crt" \
  --from-file=server.crt="certs/server.crt" \
  --from-file=server.key="certs/server.key" \
  --from-file=api.crt="certs/api.crt" \
  --from-file=api.key="certs/api.key" \
  -n <var>NAMESPACE</var> \
  --context <var>CONTEXT</var>
```

## Install a Helm chart for each cluster

Create a Helm configuration. For more information, see [Create a Helm configuration](https://docs.cloud.google.com/spanner-omni/create-helm-configuration) .

For a multi-cluster deployment, apply the Helm chart configuration from your Helm configuration file to each Kubernetes cluster. In each command, target a specific location and its namespace to bind it to the Spanner Omni deployment. Apply the configuration to each location in the same order as listed in your Helm configuration file. When you apply the configuration to the final location, the system creates a deployment bootstrap job in that cluster.

Before you install Helm for each cluster, create the monitoring namespace on the cluster where you plan to host the observability stack:

```
kubectl create namespace monitoring --context ctx-usw1
```

### Helm chart install command examples

The following commands install a Helm chart on clusters. In each command, ` PATH_TO_HELM_CONFIG_FILE ` is the path to the Helm chart configuration YAML file that you created for your deployment.

#### Install a chart in `us-west1` with monitoring enabled

```
helm upgrade --install spanner-omni oci://us-docker.pkg.dev/spanner-omni/charts/spanner-omni --version 1.0.0 \
  -f PATH_TO_HELM_CONFIG_FILE \
  --namespace spanner-ns-usw1 \
  --set currentLocation=us-west1 \
  --set monitoring.enabled=true \
  --create-namespace \
  --kube-context ctx-usw1
```

#### Install a chart in `us-west2` without monitoring

```
helm upgrade --install spanner-omni oci://us-docker.pkg.dev/spanner-omni/charts/spanner-omni --version 1.0.0 \
  -f  PATH_TO_HELM_CONFIG_FILE \
  --namespace spanner-ns-usw2 \
  --set currentLocation=us-west2 \
  --create-namespace \
  --kube-context ctx-usw2
```

#### Install a chart in `us-west3` without monitoring

```
helm upgrade --install spanner-omni oci://us-docker.pkg.dev/spanner-omni/charts/spanner-omni --version 1.0.0 \
  -f  PATH_TO_HELM_CONFIG_FILE \
  --namespace spanner-ns-usw3 \
  --set currentLocation=us-west3 \
  --create-namespace \
  --kube-context ctx-usw3
```

### Follow the progress of your deployment

After you apply the Helm chart configuration, a bootstrap job starts in each Kubernetes cluster. After the bootstrap job starts in the last cluster, run the following command to follow the deployment process:

```
kubectl logs -n spanner-ns-usw3 -l app.kubernetes.io/component=bootstrap -f
```

If the logs indicate that the deployment can't reach all servers, ensure the DNS service on each [cluster is configured correctly](https://docs.cloud.google.com/spanner-omni/deploy-multi-cluster-on-kubernetes#config-cluster-dns) .

## Configure the cluster DNS service

If you are using an [external DNS](https://github.com/kubernetes-sigs/external-dns) with your Kubernetes cluster and it manages DNS entries for [headless services](https://kubernetes.io/docs/concepts/services-networking/service/#headless-services) , you can skip this step.

Spanner uses the pod hostname for all internal communication. Configure the DNS so that pods can resolve hostnames even when running in different Kubernetes clusters. To do this, expose the `kube-dns` service in each cluster so that other clusters can reach it by running the `dns-setup.sh` script.

This script deploys a load balancer service in each cluster for the `kube-dns` application, and updates the DNS configuration to point to the load balancer services. While `dns-setup.sh` configures `kube-dns` and `CoreDNS` across GKE and Amazon EKS, you might need to configure it for your environment.

To configure the cluster DNS service:

1.  If you haven't already, download the `dns-setup.sh` script from the `spanner-omni` Cloud Storage bucket.

2.  Run the `dns-setup.sh` script:

    ```
    dns-setup.sh -n CSV_NAME_SPACE_LIST CONTEXTS
    ```

    - Replace ` CSV_NAME_SPACE_LIST ` with a comma-separated list of your namespaces.

    - Replace ` CONTEXTS ` with a list of your contexts.

    The following is an example of using the `dns-setup.sh` script:

    ```
    dns-setup.sh -n spanner-ns-usw1,spanner-ns-usw2,spanner-ns-usw3 ctx-usw1 ctx-usw2 ctx-usw3
    ```

After running the script, check the logs from the deployment job. Messages indicate that deployment is progressing, and a `Deployment created successfully` message appears.

## Update the API certificate

If you are setting up deployment without TLS encryption, proceed to [Interact with Spanner Omni](https://docs.cloud.google.com/spanner-omni/deploy-multi-cluster-on-kubernetes#step-4-interact) .

To implement TLS encryption, you must update the API certificate with the external IP addresses or DNS names of the load balancers for each cluster. This ensures that clients can connect to the deployment over secure channels.

1.  Get the service details for each cluster:

    ```
    kubectl get service spanner -n spanner-ns-usw1 --context ctx-usw1
    kubectl get service spanner -n spanner-ns-usw2 --context ctx-usw2
    kubectl get service spanner -n spanner-ns-usw3 --context ctx-usw3
    ```

2.  Update the API certificate with the external IP addresses:

    ```
    # Replace <var>EXTERNAL_IP_USW1</var>, <var>EXTERNAL_IP_USW2</var>, and <var>EXTERNAL_IP_USW3</var>
    # with the actual external IP addresses or DNS names.

    OMNI_ENDPOINT=<var>EXTERNAL_IP_USW1</var>,<var>EXTERNAL_IP_USW2</var>,<var>EXTERNAL_IP_USW3</var>,spanner.spanner-ns.svc
    Spanner Omni CLI certificates update --filename_prefix=api --hostnames=${OMNI_ENDPOINT} --ca-certificate-directory certs --output_directory certs --overwrite
    ```

3.  Update the secrets in each Kubernetes cluster with the new API certificate:

    ```
    # Repeat for each context
    kubectl patch secret tls-certs -n spanner-ns-usw1 --context ctx-usw1 -p "{\"data\":{\"api.crt\":\"$(base64 -w 0 certs/api.crt)\"}}"
    kubectl patch secret tls-certs -n spanner-ns-usw2 --context ctx-usw2 -p "{\"data\":{\"api.crt\":\"$(base64 -w 0 certs/api.crt)\"}}"
    kubectl patch secret tls-certs -n spanner-ns-usw3 --context ctx-usw3 -p "{\"data\":{\"api.crt\":\"$(base64 -w 0 certs/api.crt)\"}}"
    ```

## Interact with Spanner Omni

Each cluster in a multi-cluster setup has one load balancer service. You can use any of the service's external IP addresses to interact with Spanner Omni. For writes and strong reads, use the address that acts as the leader region. For stale read requests, use the nearest region to your application to achieve optimal performance.

1.  Run the following command to get the service address:

    ```
    kubectl get service spanner -n spanner-ns-usw1 --context ctx-usw1
    ```

    The `EXTERNAL-IP:PORT` is the ` DEPLOYMENT_ENDPOINT ` for your deployment.

2.  If you haven't already, download the Spanner Omni CLI from the `spanner-omni` Cloud Storage bucket.

3.  If you created a deployment with TLS encryption, you must include the CA certificate with each command to establish an encrypted connection. If you enabled mTLS for clients, also include the client certificate directory.

    - `--ca-certificate-file=certs/ca-api.crt`
    - `--client-certificate-directory=clientcerts`

4.  Use the Spanner Omni CLI to create a GoogleSQL or PostgreSQL database and interact with it.

    ### GoogleSQL

    To create and interact with a GoogleSQL database, run the following:

    ```
    Spanner Omni CLI databases create DATABASE_NAME \
      --deployment-endpoint=dns:///DEPLOYMENT_ENDPOINT \
      --ca-certificate-file=certs/ca-api.crt
    Spanner Omni CLI sql --database=DATABASE_NAME \
      --deployment-endpoint=dns:///DEPLOYMENT_ENDPOINT \
      --ca-certificate-file=certs/ca-api.crt
    ```

    ### PostgreSQL

    To create and interact with a PostgreSQL database, run the following:

    ```
    Spanner Omni CLI databases create POSTGRESQL_DATABASE_NAME \
      --database_dialect POSTGRESQL \
      --deployment-endpoint=dns:///DEPLOYMENT_ENDPOINT \
      --ca-certificate-file=certs/ca-api.crt
    Spanner Omni CLI sql --database=POSTGRESQL_DATABASE_NAME \
      --deployment-endpoint=dns:///DEPLOYMENT_ENDPOINT \
      --ca-certificate-file=certs/ca-api.crt
    ```

    You can also interact with a PostgreSQL database by following the instructions in [Connect using PGAdapter](https://docs.cloud.google.com/spanner-omni/pgadapter) to configure PGAdapter and use PostgreSQL tools, such as `psql` , with your PostgreSQL-dialect databases.

## What's next

- Learn how to use [client libraries](https://docs.cloud.google.com/spanner-omni/client-library-overview) and [JDBC drivers](https://docs.cloud.google.com/spanner-omni/jdbc-driver) to connect your application to the deployment.
