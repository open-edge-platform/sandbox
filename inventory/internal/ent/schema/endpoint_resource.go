// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type EndpointResource struct {
	ent.Schema
}

func (EndpointResource) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.String("kind").Optional(), field.String("name").Optional(), field.String("tenant_id").Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (EndpointResource) Edges() []ent.Edge {
	return []ent.Edge{edge.To("host", HostResource.Type).Unique()}
}
func (EndpointResource) Annotations() []schema.Annotation {
	return nil
}
func (EndpointResource) Indexes() []ent.Index {
	return []ent.Index{index.Fields("tenant_id")}
}
