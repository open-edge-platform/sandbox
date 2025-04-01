// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type LocalAccountResource struct {
	ent.Schema
}

func (LocalAccountResource) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.String("username").Immutable(), field.String("ssh_key").Immutable(), field.String("tenant_id").Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (LocalAccountResource) Edges() []ent.Edge {
	return nil
}
func (LocalAccountResource) Annotations() []schema.Annotation {
	return nil
}
func (LocalAccountResource) Indexes() []ent.Index {
	return []ent.Index{index.Fields("username", "tenant_id").Unique(), index.Fields("tenant_id")}
}
