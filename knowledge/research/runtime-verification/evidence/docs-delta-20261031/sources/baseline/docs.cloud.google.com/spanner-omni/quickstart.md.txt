---
name: documents/docs.cloud.google.com/spanner-omni/quickstart
uri: https://docs.cloud.google.com/spanner-omni/quickstart
title: Spanner Omni quickstart
description: Get started with Spanner Omni by running a single-server container.
data_source: docs.cloud.google.com
---

> [Download Spanner Omni](https://docs.cloud.google.com/spanner-omni/download) to try it out at no charge. If you decide you want to use Spanner Omni for production use, [contact Google](https://cloud.google.com/consulting/spanner-omni) to learn about acquiring a license for the [commercial edition](https://docs.cloud.google.com/spanner-omni/editions-overview#commercial-edition) .

This quickstart explains how to get started with Spanner Omni and explore some of its features by running a single-server container.

## Run a single server container

To set up and run a Spanner Omni container, follow these steps:

1.  Create a Docker volume to store Spanner Omni data. Using a volume prevents data loss if the container is accidentally deleted.

    ```
    docker volume create spanner
    ```

2.  Start the Spanner Omni server:

    ```
    docker run -d --network host --name spanneromni -v "spanner:/spanner" us-docker.pkg.dev/spanner-omni/images/spanner-omni:2026.r4-lts start-single-server
    ```

    The `--network host` flag opens the ports for Spanner Omni on the host machine. The following table describes the ports used by Spanner Omni:

    | Port range         | Purpose                     |
    |--------------------|-----------------------------|
    | `15000` to `15025` | Spanner Omni servers        |
    | `15026`            | Spanner Omni console server |

3.  Ensure that the container is running:

    ```
    docker ps
    ```

4.  Create a sample database.

    Spanner Omni includes commands to create a sample database and populate it with data. The following command creates a sample database named `retail-sample` .

    ```
    docker exec -it spanneromni /google/spanner/bin/spanner databases create-sample-db retail --database-name=retail-sample
    ```

5.  Verify that the database was created:

    ```
    docker exec -it spanneromni /google/spanner/bin/spanner databases list
    ```

    The output shows that a database named `retail-sample` has been created.

6.  Check the schema of the database:

    ```
    docker exec -it spanneromni /google/spanner/bin/spanner databases ddl describe retail-sample
    ```

    The output shows the DDL for all tables, relationships, search indexes, vector indexes, and graphs. This multi-model schema represents an ecommerce environment that tracks the customer journey from browsing to checkout. It uses advanced features like full-text search indexes, vector embeddings, and a property graph layer.

7.  Connect to the database and open the SQL shell:

    ```
    docker exec -it spanneromni /google/spanner/bin/spanner sql --database=retail-sample
    ```

    The `spanner-cli>` prompt appears, letting you run queries.

8.  Explore the database tables:

    ```
    SHOW TABLES;
    ```

## Run queries to explore the dataset

The sample database includes data for products, users, orders, and payments. Use the following queries to explore the multi-model capabilities of Spanner Omni. The following are sample queries that demonstrate some of the functionality in Spanner Omni.

### Search for products by name

Search for products using full-text search. This query uses the `SEARCH()` function on the `Name_Tokens` column, which contains pre-processed tokens for efficient matching.

The following example searches for `phone` to find smartphones and telephones:

```
SELECT ProductID, Name, Description, PriceUSD
FROM Products
WHERE SEARCH(Name_Tokens, 'phone')
ORDER BY PriceUSD DESC
LIMIT 10;
```

### Search for products by description

Search for products based on specific characteristics or features mentioned in their descriptions.

The following example searches for `waterproof` across all product categories:

```
SELECT ProductID, Name, Description, PriceUSD
FROM Products
WHERE SEARCH(Description_Tokens, 'waterproof')
ORDER BY PriceUSD
LIMIT 10;
```

### Find similar products using vector search

Use vector similarity search to find products semantically similar to a reference product. This is useful for building recommendation engines. This query uses `COSINE_DISTANCE()` with `ProductEmbedding` vectors.

Replace `PRODUCT_ID` with the ID of the product you want recommendations for:

```
WITH
 ReferenceProduct AS (
   SELECT ProductEmbedding
   FROM Products
   WHERE ProductID = PRODUCT_ID
 )
SELECT p.ProductID, p.Name, p.Description, p.PriceUSD
FROM Products p, ReferenceProduct rp
WHERE p.ProductID != PRODUCT_ID AND p.ProductEmbedding IS NOT NULL
ORDER BY
 COSINE_DISTANCE(
   rp.ProductEmbedding,
   p.ProductEmbedding)
LIMIT 5;
```

### Retrieve user purchase history

Retrieve the complete purchase history for a specific user, including product details and historical pricing.

Replace `USER_ID` with the target user's ID:

```
SELECT o.OrderID, o.OrderDate, p.Name, p.Category, oi.Quantity, oi.PriceAtOrderUSD
FROM Orders o
JOIN OrderItems oi
 ON o.OrderID = oi.OrderID
JOIN Products p
 ON oi.ProductID = p.ProductID
WHERE o.UserID = USER_ID
ORDER BY o.OrderDate DESC;
```

### Find orders by status

Use full-text search to find orders by their status. This allows for flexible matching of terms like "shipped" or "shipping".

```
SELECT OrderID, UserID, OrderDate, TotalAmountUSD
FROM Orders
WHERE SEARCH(OrderStatus_Tokens, 'shipped')
ORDER BY OrderDate DESC
LIMIT 20;
```

### Identify high-value orders

Identify high-value orders within a specific date range for business analysis.

```
SELECT OrderID, UserID, OrderDate, TotalAmountUSD
FROM Orders
WHERE
 OrderDate BETWEEN 'START_DATE' AND 'END_DATE'
 AND TotalAmountUSD > 500
ORDER BY TotalAmountUSD DESC
LIMIT 10;
```

### Analyze sales performance by category

Calculate sales metrics, such as units sold and total revenue, aggregated by product category.

```
SELECT
 p.Category,
 COUNT(oi.OrderItemID) AS total_sales,
 SUM(oi.Quantity) AS units_sold,
 SUM(oi.PriceAtOrderUSD * oi.Quantity) AS revenue
FROM Products p
JOIN OrderItems oi
 ON p.ProductID = oi.ProductID
JOIN Orders o
 ON oi.OrderID = o.OrderID
WHERE o.OrderDate BETWEEN 'START_DATE' AND 'END_DATE'
GROUP BY p.Category
ORDER BY revenue DESC;
```

### Search for payments by method

Use full-text search to find payments performed using specific methods, such as "credit card".

```
SELECT p.PaymentID, p.OrderID, p.PaymentDate, p.AmountUSD, p.Status
FROM Payments p
WHERE SEARCH(p.PaymentMethod_Tokens, 'credit card')
ORDER BY p.PaymentDate DESC
LIMIT 20;
```

### Find users who purchased a product using graph query

Use graph query syntax to traverse relationships and find all users who purchased a specific product.

Replace `PRODUCT_ID` with the target product ID:

```
GRAPH ECommerceGraph
  MATCH (p:Products { ProductID: PRODUCT_ID })<-[:OrderItems]-(o:Orders)
  MATCH (u:Users)
  WHERE u.UserID = o.UserID
  RETURN DISTINCT u.UserID, u.Email;
```

## Explore the Spanner Omni console

Spanner Omni includes a web-based Spanner Omni console for monitoring and management. To start the Spanner Omni console, run the following command:

```
docker exec -it spanneromni /google/spanner/bin/omni-jre/bin/java -jar /google/spanner/bin/spanner-console
```

To access the Spanner Omni console, navigate to `http://localhost:15026` in your browser. The Spanner Omni console provides the following information:

- **[Overview](https://docs.cloud.google.com/spanner-omni/use-console#overview)** : Displays overall CPU utilization and the database version.

- **[Databases](https://docs.cloud.google.com/spanner-omni/use-console#databases)** : Lists all databases in your deployment.

- **[Backups](https://docs.cloud.google.com/spanner-omni/use-console#backups)** : Shows information about performed backups.

- **[System Insights](https://docs.cloud.google.com/spanner-omni/use-console#system-insights)** : Displays health metrics like CPU, memory, and storage utilization, as well as performance metrics like latency and throughput.

- **[Query insights](https://docs.cloud.google.com/spanner-omni/use-console#query-insights)** : Shows the top queries consuming CPU resources.

For more information, see [Use the Spanner Omni console](https://docs.cloud.google.com/spanner-omni/use-console) .

## What's next

- [Create a deployment on VMs](https://docs.cloud.google.com/spanner-omni/deploy-on-vms) .

- [Create a deployment on Kubernetes](https://docs.cloud.google.com/spanner-omni/deploy-on-kubernetes) .
