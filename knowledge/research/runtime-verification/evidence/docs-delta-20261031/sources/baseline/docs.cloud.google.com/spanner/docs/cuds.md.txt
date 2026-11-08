---
name: documents/docs.cloud.google.com/spanner/docs/cuds
uri: https://docs.cloud.google.com/spanner/docs/cuds
title: Committed use discounts
description: Reduce your Spanner costs by purchasing CUDs for predictable workloads.
data_source: docs.cloud.google.com
---

This document describes spend-based committed use discounts (CUDs) for Spanner.

Committed use discounts (CUDs) for Spanner provide discounted prices in exchange for your commitment to continuously spend a minimum hourly amount on Spanner capacity for a year or longer.

Spanner spend-based CUDs are ideal when your spending on Spanner capacity involves a predictable minimum that you can commit to for at least a year.

## Spanner CUD pricing

Spanner CUDs offer two levels of discounts, depending on the commitment period:

- **20% discount** : You get this by committing to a 1-year term. For the duration of your term, you pay the Spanner CUD 1-year price (consumption model ID 558C-892D-2291) as your committed hourly spend amount.
- **40% discount** : You get this by committing to a 3-year term. For the duration of your term, you pay the Spanner CUD 3-year price (consumption model ID 38C3-A961-A68B) as your committed hourly spend amount.

When you purchase a commitment, you agree to pay a fixed hourly fee for a one or three-year term. Your monthly invoice shows usage charges using the CUD [consumption model](https://docs.cloud.google.com/billing/docs/resources/multiprice-cuds) prices for usage that falls within your commitment. You're charged \$1 for \$1 worth of commitment fees, and a corresponding credit applies so that the commitment fee is offset for any utilized portion of your commitment. For a full example, see [An example Spanner CUD](https://docs.cloud.google.com/spanner/docs/cuds#example) .

For any unused portion of your commitment, the fee applies. The result is that you pay the flat commitment fee every hour, whether you use the services or not, but commitment fees are then credited back to you for the used portions within the commitment amount.

Any expenditure beyond the commitment gets billed at the on-demand rate. As your usage grows, you can purchase additional commitments to receive discounts on increased expenditures not covered by previous commitments.

The CUD discount applies to any eligible usage in projects associated with the Cloud Billing account.

If the on-demand rates change after you purchase a commitment, your commitment fee doesn't change.

## Resources eligible for Spanner CUDs

Spanner CUDs automatically apply to your spending on Spanner compute capacity (as measured in nodes or processing units) across projects. This flexibility helps you achieve a high utilization rate of your commitment across regions and projects without manual intervention, saving you time and money.

Spanner CUDs don't apply to your spending on Spanner storage, backup, Data Boost, or outbound data transfer.

For a list of applicable SKUs, see [Spanner CUD Eligible SKUs](https://cloud.google.com/skus/sku-groups/cloud-spanner-cud-eligible-skus) .

## Purchase a Spanner CUD

> After your purchase a CUD, you can't cancel your commitment. Make sure the size and duration of your commitment aligns with both your historical and your expected minimum expenditure on Spanner capacity. For more information, see [Canceling commitments](https://docs.cloud.google.com/docs/cuds-spend-based#canceling_commitments) .

To purchase or manage Spanner committed use discounts for your Cloud Billing account, follow the instructions at [Purchasing spend-based commitments](https://docs.cloud.google.com/docs/cuds-spend-based#purchasing) .

## An example Spanner CUD

In this example, we calculate your expected on-demand usage cost and use that information to determine the savings offered by a one or three-year commitment. For this example, we assume that you use Spanner instances in two different regions, allocating 10 nodes in `us-central1` , and 20 nodes in `us-west2` .

> **Note:** 1-year and 3-year commitments provide different discounts.

### Calculate the on-demand cost

Start by calculating your non-discounted cost for one month (730 hours). Find your desired Spanner SKUs in the [pricing page](https://cloud.google.com/spanner/pricing) and use the drop-down to select the regions. For the calculations, use the prices you see in the three columns, *Default* price (on-demand), 1-year CUD, and 3-year CUD prices:

- Node cost for `us-central1` : \$0.9 per node per hour on-demand cost \* 10 nodes = \$9.00 per hour.
- Node cost for `us-west2` : \$1.08 per node per hour on-demand cost \* 20 nodes = \$21.60 per hour.
- On-demand costs across all regions: \$9.00 + \$21.60 = \$30.60 per hour.
- Total monthly on-demand cost: \$30.60 per hour \* 730 hours in a month = \$22,338 per month.

### Calculate the 1-year commitment cost and discount

Use the [Spanner pricing page](https://cloud.google.com/spanner/pricing) to get the hourly cost for a 1-year commitment for the regions and number of nodes, then perform the following calculations:

- Node cost for `us-central1` : 10 nodes \* \$0.72 per node per hour 1-year cost = \$7.20 per hour.
- Node cost for `us-west2` : 20 nodes \* \$0.864 per node per hour 1-year cost = \$17.28 per hour.
- Total 1-year commitment hourly cost: \$7.20 + \$17.28 = \$24.48 per hour.
- Total monthly 1-year commitment cost: \$24.48 per hour \* 730 hours in a month = \$17,870.40 per month.

Based on the results, you can make a commitment amount of `$24.48` per hour to achieve the 20% discount and see the total savings over 1 year:

- Total 1-year committment savings: On-demand cost per month \$22,338 - 1-year commitment cost per month \$17,870.40 = \$4467.60 \* 12 months = **\$53,611.20** .

### Calculate the 3-year commitment cost and discount

Use the [Spanner pricing page](https://cloud.google.com/spanner/pricing) to get the hourly cost for a 3-year commitment for the regions and number of nodes, then perform the following calculations:

- Node cost for `us-central1` : 10 nodes \* \$0.54 per node per hour 3-year cost = \$5.40 per hour.
- Node cost for `us-west2` : 20 nodes \* \$0.648 per node per hour 3-year cost = \$12.96 per hour.
- Total 3-year commitment hourly cost: \$5.40 + \$12.96 = \$18.36 per hour.
- Total monthly 3-year commitment cost: \$18.36 per hour \* 730 hours in a month = \$13,402.80 per month.

Based on the results, you can make a commitment amount of `$18.36` per hour to achieve the 40% discount and see the total savings over 3 years:

- Total 3-year committment savings: On-demand cost per month \$22,338 - 3-year commitment cost per month \$13,402.80 = \$8935.20 \* 36 months = **\$321,667.20** .

## Recommendations for choosing a commitment

When considering the purchase of Spanner CUDs, keep in mind the following:

**Projects** : Determine the consistent baseline expenditure per project while calculating total commitment. Consider that production loads usually run 100% of the time, while development or staging environments might run intermittently.

**Resources** : If you frequently scale your resources up or down, consider purchasing CUDs only for their baseline predictable usage. If you have instances that you run only for bursts or brief durations, exclude them from your calculations.

Your commitment fee applies to every hour during the term of the commitment, regardless of actual usage. Choose your CUD's commitment amount carefully, based on both your historical Spanner usage and your future expectations. As long as your use of Spanner capacity stays higher than your committed expenditure level, you will enjoy the maximum possible discount for the length of that commitment.

## What's next

- Learn more about [Spanner pricing](https://cloud.google.com/spanner/pricing) .

- Learn more about [Google Cloud spend-based CUDs](https://docs.cloud.google.com/docs/cuds) .

- Learn more about CUD [consumption models](https://docs.cloud.google.com/billing/docs/resources/multiprice-cuds) .

- Learn how to [view your CUD reports](https://docs.cloud.google.com/billing/docs/how-to/cud-analysis) .

- Understand savings with [cost breakdown reports](https://docs.cloud.google.com/billing/docs/how-to/cost-breakdown) .

- See [the list of Spanner SKUs](https://cloud.google.com/skus/sku-groups/cloud-spanner-cud-eligible-skus) that you can use with Spanner CUDs.
