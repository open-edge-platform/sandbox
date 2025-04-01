// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type HostgpuResource struct {
	ent.Schema
}

func (HostgpuResource) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.String("pci_id").Optional(), field.String("product").Optional(), field.String("vendor").Optional(), field.String("description").Optional(), field.String("device_name").Optional(), field.String("features").Optional(), field.String("tenant_id").Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (HostgpuResource) Edges() []ent.Edge {
	return []ent.Edge{edge.To("host", HostResource.Type).Required().Unique()}
}
func (HostgpuResource) Annotations() []schema.Annotation {
	return nil
}
func (HostgpuResource) Indexes() []ent.Index {
	return []ent.Index{index.Fields("tenant_id")}
}
