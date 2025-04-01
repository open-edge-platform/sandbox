// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ProviderResource struct {
	ent.Schema
}

func (ProviderResource) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.Enum("provider_kind").Values("PROVIDER_KIND_UNSPECIFIED", "PROVIDER_KIND_BAREMETAL"), field.Enum("provider_vendor").Optional().Values("PROVIDER_VENDOR_UNSPECIFIED", "PROVIDER_VENDOR_LENOVO_LXCA", "PROVIDER_VENDOR_LENOVO_LOCA"), field.String("name"), field.String("api_endpoint"), field.String("api_credentials").Optional(), field.String("config").Optional(), field.String("tenant_id").Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (ProviderResource) Edges() []ent.Edge {
	return nil
}
func (ProviderResource) Annotations() []schema.Annotation {
	return nil
}
func (ProviderResource) Indexes() []ent.Index {
	return []ent.Index{index.Fields("name", "tenant_id").Unique(), index.Fields("tenant_id")}
}
