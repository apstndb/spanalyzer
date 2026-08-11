package main

const googleSQLProtoSurfaceDDL = `
CREATE PROTO BUNDLE (
  ` + "`examples.shipping.Order`" + `,
  ` + "`examples.shipping.Order.Address`" + `,
  ` + "`examples.shipping.Order.Item`" + `,
  examples.user.User,
  examples.user.User.UserType
);

CREATE TABLE Orders (
  Id INT64 NOT NULL,
  OrderInfo ` + "`examples.shipping.Order`" + `,
) PRIMARY KEY(Id);

CREATE TABLE ProtoUsers (
  Id INT64 NOT NULL,
  UserInfo examples.user.User,
) PRIMARY KEY(Id);
`

// googleSQLProtoSurfaceQueries records the documented SELECT AS typename and
// proto-field query surfaces separately because their database setup requires
// a serialized FileDescriptorSet in addition to DDL.
var googleSQLProtoSurfaceQueries = []queryCase{
	{
		Label: "google-sql-proto-surface/accepted/new-map-constructor-field-access",
		SQL:   "SELECT order_value.order_number FROM (SELECT NEW `examples.shipping.Order` { order_number: CAST(Id AS STRING) date: Id } AS order_value FROM Orders)",
	},
	{
		Label: "google-sql-proto-surface/accepted/new-parenthesized-constructor-field-access",
		SQL:   "SELECT order_value.order_number FROM (SELECT NEW `examples.shipping.Order`(CAST(Id AS STRING) AS order_number, Id AS date) AS order_value FROM Orders)",
	},
	{
		Label: "google-sql-proto-surface/accepted/cast-string-to-proto-field-access",
		SQL:   "SELECT order_value.order_number FROM (SELECT CAST(CONCAT('order_number: \"', CAST(Id AS STRING), '\" date: ', CAST(Id AS STRING)) AS `examples.shipping.Order`) AS order_value FROM Orders)",
	},
	{
		Label: "google-sql-proto-surface/accepted/replace-proto-fields",
		SQL:   "SELECT order_value.order_number, order_value.shipping_address.country FROM (SELECT REPLACE_FIELDS(OrderInfo, CAST(Id AS STRING) AS order_number, \"CA\" AS shipping_address.country) AS order_value FROM Orders)",
	},
	{
		Label: "google-sql-proto-surface/accepted/select-as-proto-nested",
		SQL:   "SELECT order_value.order_number FROM (SELECT AS `examples.shipping.Order` CAST(Id AS STRING) AS order_number, Id AS date FROM Orders) AS order_value",
	},
	{
		Label: "google-sql-proto-surface/accepted/select-as-proto-distinct-nested",
		SQL:   "SELECT order_value.order_number FROM (SELECT DISTINCT AS `examples.shipping.Order` CAST(MOD(Id, 2) AS STRING) AS order_number, MOD(Id, 2) AS date FROM Orders) AS order_value",
	},
	{
		Label: "google-sql-proto-surface/accepted/proto-field-access",
		SQL:   "SELECT OrderInfo.order_number FROM Orders",
	},
	{
		Label: "google-sql-proto-surface/accepted/proto-nested-field-access",
		SQL:   "SELECT OrderInfo.shipping_address.country FROM Orders",
	},
	{
		Label: "google-sql-proto-surface/accepted/proto-presence-field-access",
		SQL:   "SELECT OrderInfo.has_order_number FROM Orders",
	},
	{
		Label: "google-sql-proto-surface/accepted/proto-enum-field-access",
		SQL:   "SELECT UserInfo.type FROM ProtoUsers",
	},
	{
		Label: "google-sql-proto-surface/unsupported/extract-field",
		SQL:   "SELECT EXTRACT(FIELD(order_number) FROM OrderInfo) AS order_number FROM Orders",
	},
	{
		Label: "google-sql-proto-surface/unsupported/extract-presence",
		SQL:   "SELECT EXTRACT(HAS(order_number) FROM OrderInfo) AS has_order_number FROM Orders",
	},
	{
		Label: "google-sql-proto-surface/unsupported/extract-raw-field",
		SQL:   "SELECT EXTRACT(RAW(order_number) FROM OrderInfo) AS raw_order_number FROM Orders",
	},
	{
		Label: "google-sql-proto-surface/unsupported/proto-default-if-null",
		SQL:   "SELECT PROTO_DEFAULT_IF_NULL(OrderInfo.order_number) AS order_number FROM Orders",
	},
	{
		Label: "google-sql-proto-surface/unsupported/filter-fields",
		SQL:   "SELECT filtered.order_number FROM (SELECT FILTER_FIELDS(OrderInfo, +order_number) AS filtered FROM Orders)",
	},
	{
		Label: "google-sql-proto-surface/unsupported/extract-oneof-case",
		SQL:   "SELECT EXTRACT(ONEOF_CASE(fulfillment) FROM OrderInfo) AS fulfillment_case FROM Orders",
	},
	{
		Label: "google-sql-proto-surface/accepted/proto-repeated-field-unnest",
		SQL:   "SELECT item.product_name FROM Orders AS o, UNNEST(o.OrderInfo.line_item) AS item",
	},
	{
		Label: "google-sql-proto-surface/unsupported/select-as-proto-top-level",
		SQL:   "SELECT AS `examples.shipping.Order` \"A-1\" AS order_number, 123 AS date",
	},
}
