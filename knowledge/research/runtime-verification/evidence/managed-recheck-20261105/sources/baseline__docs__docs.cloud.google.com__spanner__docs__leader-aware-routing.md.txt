---
name: documents/docs.cloud.google.com/spanner/docs/leader-aware-routing
uri: https://docs.cloud.google.com/spanner/docs/leader-aware-routing
title: Leader-aware routing
description: Use leader-aware routing in Spanner to direct write requests and strong read requests to the leader replica, reducing latency.
data_source: docs.cloud.google.com
---

This page describes leader-aware routing in Spanner and how to use it. Spanner uses leader-aware routing to dynamically route read-write transactions in dual-region and multi-region instance configurations to reduce latency and improve performance in your database. Leader-aware routing is enabled by default.

## Spanner routing for read-write transactions

Spanner replicates data to provide additional data availability and geographic locality. In Spanner [dual-region and multi-region instance configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations) , one region in the dual-region and multi-region instance configuration is designated as the leader region and it contains the [leader replicas](https://docs.cloud.google.com/spanner/docs/replication#role-of-replicas) of the database. When you use a dual-region or multi-region instance configuration and your client issues a read-write transaction to your database from a non-leader region, the write is always processed in the leader region and then sent back to the non-leader region. Therefore, read-write transactions committed from a non-leader region require multiple round trips to the leader replica to be committed successfully.

Leader-aware routing is a mechanism that improves latency for read-write transactions by intelligently routing these transactions. If leader-aware routing is enabled, even if the write doesn't originate from the leader region, the session creation requests are routed to the leader region to align the [Spanner API frontend](https://docs.cloud.google.com/spanner/docs/latency-points#spanner-api-requests) with the leader region. This routing mechanism improves latency for read-write transactions by reducing the number of network round trips required between the non-leader region (where the client application is located) and the leader region to two.

![Screenshot Spanner routing with leader-aware-routing enabled.](https://docs.cloud.google.com/static/spanner/docs/images/leader-aware-routing-enabled.png) **Figure 1.** Example of Spanner routing with leader-aware-routing enabled.

If leader-aware routing is disabled, the client application first routes the request to a Spanner API frontend service within the client application region (non-leader region). Then, from the Spanner API frontend in the client application region, three or more round trips are made to the Spanner server (SpanServer) in the leader region to commit the write, increasing latency. These additional round trips are needed to support secondary indexes, constraint checks, and read your writes.

![Screenshot Spanner routing with leader-aware-routing disabled.](https://docs.cloud.google.com/static/spanner/docs/images/leader-aware-routing-disabled.png) **Figure 2.** Example of Spanner routing with leader-aware-routing disabled.

## Use cases

As a result of using leader-aware routing, the following use cases benefits from lower latency:

- **Bulk updates** : Executing Dataflow imports or running background changes (for example, [batch DMLs](https://docs.cloud.google.com/spanner/docs/samples/spanner-dml-batch-update) ) from a non-leader region.
- **Disaster tolerance and increased availability** : Deploying client applications in both leader and non-leader regions to tolerate regional outages while initiating writes from non-leader regions.
- **Global application** : Deploying client applications globally with widespread region locations that commit data.

## Limitations

If your client application is deployed outside of the leader region and you write values without reading the data ("blind writes"), you might observe latency regression if leader-aware routing is enabled. This is because when leader-aware routing is enabled, there are two inter-region round trips ( `beginTransaction` and the `commit` request) between the client application in the non-leader region and the Spanner API frontend in the leader region. However, with leader-aware routing disabled, writes without reads only require one inter-region round trip for the `commit` request ( `beginTransaction` is processed in the local Spanner API frontend). For example, if you bulk load data into a newly created table, the transactions are unlikely to read data from the table. If you frequently commit write operations without reading it in your application, you might want to consider disabling leader-aware routing. For more information, see [Disable leader-aware routing](https://docs.cloud.google.com/spanner/docs/leader-aware-routing#disable) .

## Use leader-aware routing

Leader-aware routing is enabled by default in the Spanner client libraries.

We recommend processing your read-write requests with leader-aware routing enabled. You can disable it to compare performance differences.

### Enable leader-aware routing

You can use the Spanner client libraries to enable leader-aware routing manually.

### C++

Use the [`RouteToLeaderOption`](https://docs.cloud.google.com/cpp/docs/reference/spanner/latest/structgoogle_1_1cloud_1_1spanner_1_1RouteToLeaderOption) structure to configure your client application with leader-aware routing enabled:

```
void RouteToLeaderOption(std::string const& project_id, std::string const& instance_id,
              std::string const& database_id) {
namespace spanner = ::google::cloud::spanner;

// Create a client with RouteToLeaderOption enabled.
auto client = spanner::Client(
  spanner::MakeConnection(
      spanner::Database(project_id, instance_id, database_id)),
  google::cloud::Options{}.set<spanner::RouteToLeaderOption>(
      spanner::true));
```

### C#

Use [`EnableLeaderRouting`](https://docs.cloud.google.com/dotnet/docs/reference/Google.Cloud.Spanner.Data/latest/Google.Cloud.Spanner.Data.SpannerConnectionStringBuilder#Google_Cloud_Spanner_Data_SpannerConnectionStringBuilder_EnableLeaderRouting) to configure your client application with leader-aware routing enabled:

```
// Create a client with leader-aware routing enabled.
SpannerConnectionStringBuilder builder = new
SpannerConnectionStringBuilder();
Builder.EnableLeaderRouting = true;
```

### Go

Use [`ClientConfig`](https://docs.cloud.google.com/go/docs/reference/cloud.google.com/go/spanner/latest#cloud_google_com_go_spanner_ClientConfig) to configure your client application with leader-aware routing enabled:

```
type ClientConfig struct {
    // DisableRouteToLeader specifies if all the requests of type read-write
    // and PDML need to be routed to the leader region.
    // Default: false
    DisableRouteToLeader false
}
```

### Java

Use [`SpannerOptions.Builder`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.SpannerOptions.Builder#com_google_cloud_spanner_SpannerOptions_Builder_enableLeaderAwareRouting__) to configure your client application with leader-aware routing enabled:

```
SpannerOptions options = SpannerOptions.newBuilder().enableLeaderAwareRouting.build();
Spanner spanner = options.getService();
String instance = "my-instance";
String database = "my-database";
```

### Node.js

Use [`SpannerOptions`](https://docs.cloud.google.com/nodejs/docs/reference/spanner/latest/spanner/spanneroptions#_google_cloud_spanner_SpannerOptions_routeToLeaderEnabled_member) to configure your client application with leader-aware routing enabled:

```
// Instantiates a client with routeToLeaderEnabled enabled
const spanner = new Spanner({
projectId: projectId,
routeToLeaderEnabled: true;
});
```

### PHP

Use `routeToLeader` to configure your client application with leader-aware routing enabled:

```
// Instantiates a client with leader-aware routing enabled
use Google\Cloud\Spanner\SpannerClient;

$routeToLeader = true;
$spanner = new SpannerClient($routeToLeader);
```

### Python

Use [`route_to_leader_enabled`](https://docs.cloud.google.com/python/docs/reference/spanner/latest/google.cloud.spanner_v1.client.Client#google_cloud_spanner_v1_client_Client_route_to_leader_enabled) to configure your client application with leader-aware routing enabled:

```
spanner_client = spanner.Client(
route_to_leader_enabled=true
)
instance = spanner_client.instance(instance_id)
database = instance.database(database_id)
```

### Ruby

Use [`self.new`](https://docs.cloud.google.com/ruby/docs/reference/google-cloud-spanner/latest/Google-Cloud-Spanner#Google__Cloud__Spanner_new_class_) to configure your client application with leader-aware routing enabled:

```
def self.new(project_id: nil, credentials: nil, scope: nil, timeout: nil,
     endpoint: nil, project: nil, keyfile: nil, emulator_host: nil,
    lib_name: nil, lib_version: nil, enable_leader_aware_routing: true) ->
    Google::Cloud::Spanner::Project
```

### Disable leader-aware routing

You can use the Spanner client libraries to disable leader-aware routing.

### C++

Use the [`RouteToLeaderOption`](https://docs.cloud.google.com/cpp/docs/reference/spanner/latest/structgoogle_1_1cloud_1_1spanner_1_1RouteToLeaderOption) structure to configure your client application with leader-aware routing disabled:

```
void RouteToLeaderOption(std::string const& project_id, std::string const& instance_id,
              std::string const& database_id) {
namespace spanner = ::google::cloud::spanner;

// Create a client with RouteToLeaderOption disabled.
auto client = spanner::Client(
  spanner::MakeConnection(
      spanner::Database(project_id, instance_id, database_id)),
  google::cloud::Options{}.set<spanner::RouteToLeaderOption>(
      spanner::false));
```

### C#

Use [`EnableLeaderRouting`](https://docs.cloud.google.com/dotnet/docs/reference/Google.Cloud.Spanner.Data/latest/Google.Cloud.Spanner.Data.SpannerConnectionStringBuilder#Google_Cloud_Spanner_Data_SpannerConnectionStringBuilder_EnableLeaderRouting) to configure your client application with leader-aware routing disabled:

```
// Create a client with leader-aware routing disabled.
SpannerConnectionStringBuilder builder = new
SpannerConnectionStringBuilder();
Builder.EnableLeaderRouting = false;
```

### Go

Use [`ClientConfig`](https://docs.cloud.google.com/go/docs/reference/cloud.google.com/go/spanner/latest#cloud_google_com_go_spanner_ClientConfig) to configure your client application with leader-aware routing disabled:

```
type ClientConfig struct {
    // DisableRouteToLeader specifies if all the requests of type read-write
    // and PDML need to be routed to the leader region.
    // Default: false
    DisableRouteToLeader true
}
```

### Java

Use [`SpannerOptions.Builder`](https://docs.cloud.google.com/java/docs/reference/google-cloud-spanner/latest/com.google.cloud.spanner.SpannerOptions.Builder#com_google_cloud_spanner_SpannerOptions_Builder_disableLeaderAwareRouting__) to create a connection to a Spanner database with leader aware routing disabled:

```
SpannerOptions options = SpannerOptions.newBuilder().disableLeaderAwareRouting.build();
Spanner spanner = options.getService();
String instance = "my-instance";
String database = "my-database";
```

### Node.js

Use [`SpannerOptions`](https://docs.cloud.google.com/nodejs/docs/reference/spanner/latest/spanner/spanneroptions#_google_cloud_spanner_SpannerOptions_routeToLeaderEnabled_member) to configure your client application with leader-aware routing disabled:

```
// Instantiates a client with routeToLeaderEnabled disabled
const spanner = new Spanner({
projectId: projectId,
routeToLeaderEnabled: false;
});
```

### PHP

Use `routeToLeader` to configure your client application with leader-aware routing disabled:

```
// Instantiates a client with leader-aware routing disabled
use Google\Cloud\Spanner\SpannerClient;

$routeToLeader = false;
$spanner = new SpannerClient($routeToLeader);
```

### Python

Use [`route_to_leader_enabled`](https://docs.cloud.google.com/python/docs/reference/spanner/latest/google.cloud.spanner_v1.client.Client#google_cloud_spanner_v1_client_Client_route_to_leader_enabled) to configure your client application with leader-aware routing disabled:

```
spanner_client = spanner.Client(
route_to_leader_enabled=false
)
instance = spanner_client.instance(instance_id)
database = instance.database(database_id)
```

### Ruby

Use [`self.new`](https://docs.cloud.google.com/ruby/docs/reference/google-cloud-spanner/latest/Google-Cloud-Spanner#Google__Cloud__Spanner_new_class_) to configure your client application with leader-aware routing disabled:

```
def self.new(project_id: nil, credentials: nil, scope: nil, timeout: nil,
     endpoint: nil, project: nil, keyfile: nil, emulator_host: nil,
    lib_name: nil, lib_version: nil, enable_leader_aware_routing: false) ->
    Google::Cloud::Spanner::Project
```

## What's next

- Learn about [Regional, dual-region, and multi-region configurations](https://docs.cloud.google.com/spanner/docs/instance-configurations) .
- Learn about [Replication](https://docs.cloud.google.com/spanner/docs/replication) .
- Learn about how to [Modify the leader region of a database](https://docs.cloud.google.com/spanner/docs/modifying-leader-region) .
