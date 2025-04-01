// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type HostusbResource struct {
	ent.Schema
}

func (HostusbResource) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.String("kind").Optional(), field.String("owner_id").Optional(), field.String("idvendor").Optional(), field.String("idproduct").Optional(), field.Uint32("bus").Optional(), field.Uint32("addr").Optional(), field.String("class").Optional(), field.String("serial").Optional(), field.String("device_name").Optional(), field.String("tenant_id").Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (HostusbResource) Edges() []ent.Edge {
	return []ent.Edge{edge.To("host", HostResource.Type).Required().Unique()}
}
func (HostusbResource) Annotations() []schema.Annotation {
	return nil
}
func (HostusbResource) Indexes() []ent.Index {
	return []ent.Index{index.Fields("tenant_id")}
}
