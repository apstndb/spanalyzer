---
name: documents/docs.cloud.google.com/spanner-omni/deploy-on-kubernetes
uri: https://docs.cloud.google.com/spanner-omni/deploy-on-kubernetes
title: Create a deployment on Kubernetes
description: Deploy Spanner Omni on Kubernetes. Configure single-server or regional deployments and understand security risks.
data_source: docs.cloud.google.com
---

This document explains how create a Spanner Omni deployment on Kubernetes. This deployment isn't encrypted. If you want to quickly set up a test or proof-of-concept environment to evaluate Spanner Omni, then creating a deployment without encryption is the fastest way to get started because it doesn't require you to configure mTLS or other security measures. However, because of security risks, such as unencrypted network traffic and open access, this configuration isn't recommended for production environments. You can choose between a single-server or a regional deployment across multiple zones.

> **Caution:** Unencrypted deployments are intended for testing only. In production environments, use encrypted deployments. For more information, see [Add TLS encryption to your Kubernetes deployment](https://docs.cloud.google.com/spanner-omni/deploy-encryption-kubernetes) .

## Before you begin

Before deploying Spanner Omni, ensure that your environment meets the following requirements:

- Create a Kubernetes cluster. The configuration supports Google Kubernetes Engine (GKE) and Amazon Elastic Kubernetes Service (Amazon EKS). You might need to customize the configuration to work in other environments.

- Ensure the Kubernetes cluster can access the Artifact Registry artifact that hosts the Spanner Omni container.

- Install and configure the [`kubectl` command line tool](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/) and [Helm](https://helm.sh/) .

- If you set up the Kubernetes environment on vSphere virtualization platform machines, disable Time Stamp Counter (TSC) virtualization by adding `monitor_control.virtual_rdtsc = FALSE` to the virtual machine's `.vmx` configuration file. This helps TrueTime work correctly.

- Verify your environment meets [Spanner Omni system requirements](https://docs.cloud.google.com/spanner-omni/system-requirements) .

- Choose a [topology for your deployment](https://docs.cloud.google.com/spanner-omni/overview#deployment-topologies) .

## Prepare the Helm configuration

Create a Helm configuration. For more information, see [Create a Helm configuration](https://docs.cloud.google.com/spanner-omni/create-helm-configuration) .

## Create the deployment

Install the Helm chart with the specific overrides. The following are sample commands for the most common deployments:

### Example 1: Run Spanner Omni on a single server on GKE with the monitoring stack

To run Spanner Omni on a single server on GKE with the monitoring stack, run the following command:

```
  kubectl create ns monitoring
  helm upgrade --install spanner-omni oci://us-docker.pkg.dev/spanner-omni/charts/spanner-omni --version 1.0.0 \
    --set global.platform=gke \
    --set deployment.singleServer=true \
    --set monitoring.enabled=true \
    --namespace spanner-ns \
    --create-namespace
```

> **Note:** Update the platform to `eks` to deploy Spanner Omni in Amazon EKS.

### Example 2: Run Spanner Omni on multiple servers in GKE

To run Spanner Omni on multiple servers in a single zone ( `us-central1-a` ) in GKE, run the following command:

```
  kubectl create ns monitoring
  helm upgrade --install spanner-omni oci://us-docker.pkg.dev/spanner-omni/charts/spanner-omni --version 1.0.0 \
    --set global.platform=gke \
    --set deployment.replicasPerZone=5 \
    --set deployment.rootServersPerZone=3 \
    --set-json 'locations=[{"name":"us-central1","zones":[{"name":"us-central1-a","shortName":"a"}]}]' \
    --set monitoring.enabled=true \
    --namespace spanner-ns \
    --create-namespace
```

> **Note:** To deploy in Amazon EKS, Update `global.platform` to `eks` and `locations` to the AWS zone where your Amazon EKS cluster is created.

The number of root servers per zone must be an odd number between one and nine, inclusive, to ensure quorum for consistency. If the number of servers is an even number, deployments might fail. When configuring your zones, designate servers as root servers. We recommend that you use one for development or testing and three for highly available production zones.

### Example 3: Highly available regional deployment

This deployment keeps three copies of data, allowing Spanner Omni to continue working even if a zone experiences an outage. To create this deployment, run the following command:

```
  kubectl create ns monitoring
  helm upgrade --install spanner-omni oci://us-docker.pkg.dev/spanner-omni/charts/spanner-omni --version 1.0.0 \
    --set global.platform=gke \
    --set-json 'locations=[{"name":"us-central1","zones":[{"name":"us-central1-a","shortName":"a"},{"name":"us-central1-c","shortName":"b"},{"name":"us-central1-d","shortName":"c"}]}]' \
    --set monitoring.enabled=true \
    --namespace spanner-ns \
    --create-namespace
```

## Check the status of the pods

To check the status of the pods, run the following command:

```
  kubectl get pods --watch --namespace spanner-ns
```

Example output:

```
  NAME          READY   STATUS    RESTARTS   AGE
  spanner-a-0   1/1     Running   0          4m
  spanner-a-1   1/1     Running   0          4m
  spanner-a-2   1/1     Running   0          4m
  spanner-a-3   1/1     Running   0          4m
  spanner-a-4   1/1     Running   0          4m
  spanner-b-0   1/1     Running   0          4m
  spanner-b-1   1/1     Running   0          4m
  spanner-b-2   1/1     Running   0          4m
  spanner-b-3   1/1     Running   0          4m
  spanner-b-4   1/1     Running   0          4m
  spanner-c-0   1/1     Running   0          4m
  spanner-c-1   1/1     Running   0          4m
  spanner-c-2   1/1     Running   0          4m
  spanner-c-3   1/1     Running   0          4m
  spanner-c-4   1/1     Running   0          4m
```

## Interact with Spanner Omni

After the pods are running, you can connect to your deployment and interact with it using the Spanner Omni CLI.

1.  Run the following command to get the service address:

    ```
    kubectl get service spanner -n spanner-ns
    ```

    The `EXTERNAL-IP:PORT` is the ` DEPLOYMENT_ENDPOINT ` for your deployment.

2.  If you haven't already, download the Spanner Omni CLI from the `spanner-omni` Cloud Storage bucket.

3.  Use the Spanner Omni CLI to create a GoogleSQL or PostgreSQL database and interact with it.

    ### GoogleSQL

    To create and interact with a GoogleSQL database, run the following:

    ```
    spanner databases create DATABASE_NAME --deployment-endpoint DEPLOYMENT_ENDPOINT
    spanner sql --database=DATABASE_NAME --deployment-endpoint DEPLOYMENT_ENDPOINT
    ```

    ### PostgreSQL

    To create and interact with a PostgreSQL database, run the following:

    ````
     spanner databases create POSTGRESQL_DATABASE_NAME --database_dialect POSTGRESQL --deployment-endpoint DEPLOYMENT_ENDPOINT
     spanner sql --database=POSTGRESQL_DATABASE_NAME --deployment-endpoint DEPLOYMENT_ENDPOINT
     ```

    You can also interact with a PostgreSQL database by following
    the instructions in [Connect using PGAdapter](/spanner-omni/pgadapter)
    to configure PGAdapter and use PostgreSQL tools, such as
    `psql`, with your PostgreSQL-dialect databases.
    ````

## Observe the deployment (Optional)

You can set up Spanner Omni with `monitoring.enabled=true` to configure Prometheus to ingest metrics that Spanner Omni exports. This helps you analyze and debug issues with your deployment. For more information see:

- [Use Prometheus alerts to monitor Spanner Omni](https://docs.cloud.google.com/spanner-omni/prometheus-alerts) .

- [Use Grafana dashboards to monitor Spanner Omni](https://docs.cloud.google.com/spanner-omni/grafana-dashboards) .

To get the service details, run the following commands:

```
  # Prometheus service details. Default port is 9090.
  kubectl get service prometheus-service -n monitoring

  # Grafana service details. Default port is 3000.
  kubectl get service grafana -n monitoring
```

## What's next

- [Add TLS encryption to your Kubernetes deployment](https://docs.cloud.google.com/spanner-omni/deploy-encryption-kubernetes) .
