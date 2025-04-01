// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type HoststorageResource struct {
	ent.Schema
}

func (HoststorageResource) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.String("kind").Optional(), field.String("provider_status").Optional(), field.String("wwid").Optional(), field.String("serial").Optional(), field.String("vendor").Optional(), field.String("model").Optional(), field.Uint64("capacity_bytes").Optional(), field.String("device_name").Optional(), field.String("tenant_id").Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (HoststorageResource) Edges() []ent.Edge {
	return []ent.Edge{edge.To("host", HostResource.Type).Required().Unique()}
}
func (HoststorageResource) Annotations() []schema.Annotation {
	return nil
}
func (HoststorageResource) Indexes() []ent.Index {
	return []ent.Index{index.Fields("tenant_id")}
}
