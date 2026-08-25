package astconv

import (
	"reflect"
	"testing"

	"github.com/apstndb/spanner-emulator-survey/infoschem"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestExtractProtoBundleTypes(t *testing.T) {
	raw := mustMarshalFileDescriptorSet(t, &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			{
				Package: proto.String("examples.shipping"),
				EnumType: []*descriptorpb.EnumDescriptorProto{
					{Name: proto.String("ShippingSpeed")},
				},
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: proto.String("Order"),
						EnumType: []*descriptorpb.EnumDescriptorProto{
							{Name: proto.String("Status")},
						},
						NestedType: []*descriptorpb.DescriptorProto{
							{Name: proto.String("Address")},
							{Name: proto.String("Item")},
						},
					},
				},
			},
		},
	})

	got, err := extractProtoBundleTypes(raw)
	if err != nil {
		t.Fatalf("extractProtoBundleTypes: %v", err)
	}

	want := []string{
		"examples.shipping.ShippingSpeed",
		"examples.shipping.Order",
		"examples.shipping.Order.Status",
		"examples.shipping.Order.Address",
		"examples.shipping.Order.Item",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("extractProtoBundleTypes() = %#v, want %#v", got, want)
	}
}

func TestToProtoBundleDDL_FromSchemataProtoBundle(t *testing.T) {
	raw := mustMarshalFileDescriptorSet(t, &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			{
				Package: proto.String("examples.shipping"),
				EnumType: []*descriptorpb.EnumDescriptorProto{
					{Name: proto.String("ShippingSpeed")},
				},
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: proto.String("Order"),
						EnumType: []*descriptorpb.EnumDescriptorProto{
							{Name: proto.String("Status")},
						},
						NestedType: []*descriptorpb.DescriptorProto{
							{Name: proto.String("Address")},
							{Name: proto.String("Item")},
						},
					},
				},
			},
		},
	})

	schema := &Schema{
		Schemata: []*infoschem.Schema{
			{SchemaName: "", ProtoBundle: raw},
		},
	}

	ddls, err := schema.toProtoBundleDDL()
	if err != nil {
		t.Fatalf("toProtoBundleDDL: %v", err)
	}
	if len(ddls) != 1 {
		t.Fatalf("len(ddls) = %d, want 1", len(ddls))
	}

	got := ddls[0].SQL()
	want := "CREATE PROTO BUNDLE (`examples.shipping.Order`, `examples.shipping.Order.Address`, `examples.shipping.Order.Item`, `examples.shipping.Order.Status`, examples.shipping.ShippingSpeed)"
	if got != want {
		t.Fatalf("SQL() = %q, want %q", got, want)
	}
}

func TestToProtoBundleDDL_ExplicitTypesOverrideDescriptor(t *testing.T) {
	schema := &Schema{
		ProtoBundleTypes: []string{"`examples.shipping.Order`"},
		Schemata: []*infoschem.Schema{
			{SchemaName: "", ProtoBundle: []byte("not-a-descriptor")},
		},
	}

	ddls, err := schema.toProtoBundleDDL()
	if err != nil {
		t.Fatalf("toProtoBundleDDL: %v", err)
	}
	if len(ddls) != 1 {
		t.Fatalf("len(ddls) = %d, want 1", len(ddls))
	}

	got := ddls[0].SQL()
	want := "CREATE PROTO BUNDLE (`examples.shipping.Order`)"
	if got != want {
		t.Fatalf("SQL() = %q, want %q", got, want)
	}
}

func TestToProtoBundleDDL_UnionsSchemataRows(t *testing.T) {
	defaultRaw := mustMarshalFileDescriptorSet(t, &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			{
				Package: proto.String("examples"),
				MessageType: []*descriptorpb.DescriptorProto{
					{Name: proto.String("Shared")},
					{Name: proto.String("DefaultOnly")},
				},
			},
		},
	})
	namedRaw := mustMarshalFileDescriptorSet(t, &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			{
				Package: proto.String("examples"),
				MessageType: []*descriptorpb.DescriptorProto{
					{Name: proto.String("Shared")},
					{Name: proto.String("NamedOnly")},
				},
			},
		},
	})
	schema := &Schema{
		Schemata: []*infoschem.Schema{
			{ProtoBundle: defaultRaw},
			{SchemaName: "app", ProtoBundle: namedRaw},
		},
	}

	ddls, err := schema.toProtoBundleDDL()
	if err != nil {
		t.Fatalf("toProtoBundleDDL: %v", err)
	}
	if len(ddls) != 1 {
		t.Fatalf("len(ddls) = %d, want 1", len(ddls))
	}
	got := ddls[0].SQL()
	want := "CREATE PROTO BUNDLE (examples.DefaultOnly, examples.NamedOnly, examples.Shared)"
	if got != want {
		t.Fatalf("SQL() = %q, want %q", got, want)
	}
}

func TestToProtoBundleDDL_InvalidDescriptor(t *testing.T) {
	schema := &Schema{
		Schemata: []*infoschem.Schema{
			{SchemaName: "app", ProtoBundle: []byte("not-a-descriptor")},
		},
	}

	_, err := schema.toProtoBundleDDL()
	if err == nil {
		t.Fatal("toProtoBundleDDL() error = nil, want non-nil")
	}
}

func mustMarshalFileDescriptorSet(t *testing.T, descriptors *descriptorpb.FileDescriptorSet) []byte {
	t.Helper()

	raw, err := proto.Marshal(descriptors)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}
	return raw
}
