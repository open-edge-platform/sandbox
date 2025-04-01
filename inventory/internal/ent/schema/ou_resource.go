// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type OuResource struct {
	ent.Schema
}

func (OuResource) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.String("name").Optional(), field.String("ou_kind").Optional(), field.String("metadata").Optional(), field.String("tenant_id").Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (OuResource) Edges() []ent.Edge {
	return []ent.Edge{edge.To("parent_ou", OuResource.Type).Unique(), edge.From("children", OuResource.Type).Ref("parent_ou")}
}
func (OuResource) Annotations() []schema.Annotation {
	return nil
}
func (OuResource) Indexes() []ent.Index {
	return []ent.Index{index.Fields("tenant_id")}
}
