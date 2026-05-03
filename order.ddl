CREATE PROTO BUNDLE (
  examples.shipping.Order,
  examples.shipping.Order.Address,
  examples.shipping.Order.Item
);

CREATE TABLE Orders (
  Id INT64,
  OrderInfo `examples.shipping.Order`,
) PRIMARY KEY(Id);
