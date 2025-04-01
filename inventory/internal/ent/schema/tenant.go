// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Tenant struct {
	ent.Schema
}

func (Tenant) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.Enum("current_state").Optional().Values("TENANT_STATE_UNSPECIFIED", "TENANT_STATE_CREATED", "TENANT_STATE_DELETED"), field.Enum("desired_state").Values("TENANT_STATE_UNSPECIFIED", "TENANT_STATE_CREATED", "TENANT_STATE_DELETED"), field.Bool("watcher_osmanager").Optional(), field.String("tenant_id").Unique().Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (Tenant) Edges() []ent.Edge {
	return nil
}
func (Tenant) Annotations() []schema.Annotation {
	return nil
}
func (Tenant) Indexes() []ent.Index {
	return []ent.Index{index.Fields("tenant_id")}
}
