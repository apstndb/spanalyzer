package spanalyzer

import (
	"strings"
	"testing"

	"cloud.google.com/go/spanner/apiv1/spannerpb"
)

func TestAnalyzerRowTypeForPropertyGraphDerivedPropertyTypes(t *testing.T) {
	const ddl = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName STRING(MAX),
) PRIMARY KEY (SingerId);

CREATE PROPERTY GRAPH MyGraph
  NODE TABLES (
    Singers
      LABEL Singer
      PROPERTIES (
        SingerId,
        CONCAT(FirstName, ' ', LastName) AS FullName,
        LENGTH(FirstName) AS FirstNameLength,
        [FirstName, LastName] AS Names
      )
  );
`
	analyzer, err := NewAnalyzerFromDDL("graph_derived.sql", ddl)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement("GRAPH MyGraph MATCH (n:Singer) RETURN n.SingerId, n.FullName, n.FirstNameLength, n.Names")
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 4; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "SingerId", spannerpb.TypeCode_INT64)
	assertField(t, rowType.Fields[1], "FullName", spannerpb.TypeCode_STRING)
	assertField(t, rowType.Fields[2], "FirstNameLength", spannerpb.TypeCode_INT64)
	if rowType.Fields[3].Name != "Names" || rowType.Fields[3].Type.GetCode() != spannerpb.TypeCode_ARRAY {
		t.Fatalf("Names = (%q, %s), want ARRAY", rowType.Fields[3].Name, rowType.Fields[3].Type.GetCode())
	}
	if rowType.Fields[3].Type.GetArrayElementType().GetCode() != spannerpb.TypeCode_STRING {
		t.Fatalf("Names element = %s, want STRING", rowType.Fields[3].Type.GetArrayElementType().GetCode())
	}
}

func TestAnalyzerRowTypeForPropertyGraphDerivedProtoAndEnum(t *testing.T) {
	const ddl = `
CREATE PROTO BUNDLE (
  ` + "`examples.shipping.Order`" + `,
  ` + "`examples.user.User`" + `,
  ` + "`examples.user.User.UserType`" + `
);
CREATE TABLE Orders (
  Id INT64 NOT NULL,
  OrderInfo ` + "`examples.shipping.Order`" + `,
  UserInfo ` + "`examples.user.User`" + `,
) PRIMARY KEY (Id);

CREATE PROPERTY GRAPH OrderGraph
  NODE TABLES (
    Orders
      LABEL OrderNode
      PROPERTIES (
        OrderInfo.order_number AS OrderNumber,
        IF(TRUE, OrderInfo, OrderInfo) AS OrderValue,
        UserInfo.type AS UserType
      )
  );
`
	analyzer, err := NewAnalyzerFromDDLWithProtoDescriptorFiles("graph_proto.sql", ddl, []string{
		"testdata/protos/order_descriptors.pb",
		"testdata/protos/complex/complex_descriptors.pb",
	})
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDLWithProtoDescriptorFiles() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement("GRAPH OrderGraph MATCH (n:OrderNode) RETURN n.OrderNumber, n.OrderValue, n.UserType")
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	if got, want := len(rowType.Fields), 3; got != want {
		t.Fatalf("len(rowType.Fields) = %d, want %d", got, want)
	}
	assertField(t, rowType.Fields[0], "OrderNumber", spannerpb.TypeCode_STRING)
	assertProtoField(t, rowType.Fields[1], "OrderValue", "examples.shipping.Order")
	assertEnumField(t, rowType.Fields[2], "UserType", "examples.user.User.UserType")
}

func TestNewAnalyzerFromDDLPropertyGraphDerivedExpressionError(t *testing.T) {
	const ddl = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
) PRIMARY KEY (SingerId);

CREATE PROPERTY GRAPH MyGraph
  NODE TABLES (
    Singers
      LABEL Singer
      PROPERTIES (MissingColumn AS Broken)
  );
`
	_, err := NewAnalyzerFromDDL("graph_bad.sql", ddl)
	if err == nil {
		t.Fatal("NewAnalyzerFromDDL() error = nil, want derived-expression diagnostic")
	}
	if !strings.Contains(err.Error(), "property graph MyGraph") || !strings.Contains(err.Error(), "property Broken") || !strings.Contains(err.Error(), "analyze derived expression") {
		t.Fatalf("error = %v, want graph/property/analyze context", err)
	}
}

func TestAnalyzerRowTypeForPropertyGraphDuplicateColumnNamesStayTableScoped(t *testing.T) {
	const ddl = `
CREATE TABLE LeftSide (
  Id INT64 NOT NULL,
  Name STRING(MAX),
) PRIMARY KEY (Id);

CREATE TABLE RightSide (
  Id INT64 NOT NULL,
  Name INT64,
) PRIMARY KEY (Id);

CREATE PROPERTY GRAPH PairGraph
  NODE TABLES (
    LeftSide LABEL LeftNode PROPERTIES (CONCAT(Name, Name) AS Label),
    RightSide LABEL RightNode PROPERTIES (Name + 1 AS Quantity)
  );
`
	analyzer, err := NewAnalyzerFromDDL("graph_dup_cols.sql", ddl)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	leftType, err := analyzer.RowTypeForStatement("GRAPH PairGraph MATCH (n:LeftNode) RETURN n.Label")
	if err != nil {
		t.Fatalf("RowTypeForStatement(LeftNode) error = %v", err)
	}
	assertField(t, leftType.Fields[0], "Label", spannerpb.TypeCode_STRING)
	rightType, err := analyzer.RowTypeForStatement("GRAPH PairGraph MATCH (n:RightNode) RETURN n.Quantity")
	if err != nil {
		t.Fatalf("RowTypeForStatement(RightNode) error = %v", err)
	}
	assertField(t, rightType.Fields[0], "Quantity", spannerpb.TypeCode_INT64)
}

func TestNewAnalyzerFromDDLPropertyGraphCompatibleRepeatedProperty(t *testing.T) {
	const ddl = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName STRING(MAX),
) PRIMARY KEY (SingerId);

CREATE TABLE Fans (
  FanId INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName STRING(MAX),
) PRIMARY KEY (FanId);

CREATE PROPERTY GRAPH PeopleGraph
  NODE TABLES (
    Singers LABEL Singer PROPERTIES (CONCAT(FirstName, ' ', LastName) AS FullName),
    Fans LABEL Fan PROPERTIES (CONCAT(FirstName, ' ', LastName) AS FullName)
  );
`
	analyzer, err := NewAnalyzerFromDDL("graph_compat.sql", ddl)
	if err != nil {
		t.Fatalf("NewAnalyzerFromDDL() error = %v", err)
	}
	rowType, err := analyzer.RowTypeForStatement("GRAPH PeopleGraph MATCH (n:Singer) RETURN n.FullName")
	if err != nil {
		t.Fatalf("RowTypeForStatement() error = %v", err)
	}
	assertField(t, rowType.Fields[0], "FullName", spannerpb.TypeCode_STRING)
}

func TestNewAnalyzerFromDDLPropertyGraphIncompatibleRepeatedProperty(t *testing.T) {
	const ddl = `
CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(MAX),
  LastName STRING(MAX),
) PRIMARY KEY (SingerId);

CREATE TABLE Fans (
  FanId INT64 NOT NULL,
  FirstName STRING(MAX),
) PRIMARY KEY (FanId);

CREATE PROPERTY GRAPH PeopleGraph
  NODE TABLES (
    Singers LABEL Singer PROPERTIES (CONCAT(FirstName, ' ', LastName) AS FullName),
    Fans LABEL Fan PROPERTIES (LENGTH(FirstName) AS FullName)
  );
`
	_, err := NewAnalyzerFromDDL("graph_incompat.sql", ddl)
	if err == nil {
		t.Fatal("NewAnalyzerFromDDL() error = nil, want incompatible property types")
	}
	if !strings.Contains(err.Error(), "property graph PeopleGraph") || !strings.Contains(err.Error(), "property FullName") || !strings.Contains(err.Error(), "incompatible types") {
		t.Fatalf("error = %v, want incompatible FullName types", err)
	}
}
