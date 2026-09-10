package astconv

import (
	"context"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/apstndb/spanemuboost"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestProtoBundle_FromEmulatorSCHEMATA(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping emulator integration test in short mode")
	}

	ctx := context.Background()

	fds := exampleShippingFileDescriptorSet(t)

	env, err := spanemuboost.RunWithClients(ctx, spanemuboost.BackendEmulator,
		spanemuboost.WithContainerImage(pinnedEmulatorImage(t)),
		spanemuboost.WithSetupDDLs([]string{
			"CREATE SCHEMA app",
			"CREATE PROTO BUNDLE (`examples.shipping.Order`)",
		}),
		spanemuboost.WithSetupFileDescriptorSet(fds),
	)
	if err != nil {
		t.Fatalf("RunWithClients: %v", err)
	}
	defer func() { _ = env.Close() }()

	assertProtoBundleSchemata(ctx, t, env.Client)
}

func exampleShippingFileDescriptorSet(t *testing.T) *descriptorpb.FileDescriptorSet {
	t.Helper()

	return &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			{
				Name:    new("shipping.proto"),
				Syntax:  new("proto3"),
				Package: new("examples.shipping"),
				EnumType: []*descriptorpb.EnumDescriptorProto{
					{
						Name: new("ShippingSpeed"),
						Value: []*descriptorpb.EnumValueDescriptorProto{
							{Name: new("SHIPPING_SPEED_UNSPECIFIED"), Number: new(int32(0))},
						},
					},
				},
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: new("Order"),
						EnumType: []*descriptorpb.EnumDescriptorProto{
							{
								Name: new("Status"),
								Value: []*descriptorpb.EnumValueDescriptorProto{
									{Name: new("STATUS_UNSPECIFIED"), Number: new(int32(0))},
								},
							},
						},
						NestedType: []*descriptorpb.DescriptorProto{
							{Name: new("Address")},
						},
					},
				},
			},
		},
	}
}

func assertProtoBundleSchemata(ctx context.Context, t *testing.T, client *spanner.Client) {
	t.Helper()

	schema := &Schema{}
	if err := querySchemata(ctx, client, schema); err != nil {
		t.Fatalf("querySchemata: %v", err)
	}

	var protoBundle []byte
	var foundNamedSchema bool
	for _, s := range schema.Schemata {
		if s.SchemaName == "" && len(s.ProtoBundle) > 0 {
			protoBundle = s.ProtoBundle
		}
		if s.SchemaName == "app" {
			foundNamedSchema = true
		}
	}
	if len(protoBundle) == 0 {
		t.Fatal("SCHEMATA.PROTO_BUNDLE is empty for default schema")
	}
	if !foundNamedSchema {
		t.Fatal("SCHEMATA has no app row")
	}

	ddls, err := schema.toProtoBundleDDL()
	if err != nil {
		t.Fatalf("toProtoBundleDDL: %v", err)
	}
	if len(ddls) != 1 {
		t.Fatalf("expected 1 CREATE PROTO BUNDLE DDL, got %d", len(ddls))
	}

	got := ddls[0].SQL()
	want := "CREATE PROTO BUNDLE (`examples.shipping.Order`, `examples.shipping.Order.Address`, `examples.shipping.Order.Status`, examples.shipping.ShippingSpeed)"
	if got != want {
		t.Fatalf("SQL() = %q, want %q", got, want)
	}
}

func querySchemata(ctx context.Context, client *spanner.Client, schema *Schema) error {
	iter := client.Single().Query(ctx, spanner.Statement{SQL: "SELECT SCHEMA_NAME, PROTO_BUNDLE FROM INFORMATION_SCHEMA.SCHEMATA"})
	defer iter.Stop()
	return spanner.SelectAll(iter, &schema.Schemata, spanner.WithLenient())
}
