// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type NetworkSegment struct {
	ent.Schema
}

func (NetworkSegment) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.String("name").Optional(), field.Int32("vlan_id").Optional(), field.String("tenant_id").Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (NetworkSegment) Edges() []ent.Edge {
	return []ent.Edge{edge.To("site", SiteResource.Type).Required().Unique()}
}
func (NetworkSegment) Annotations() []schema.Annotation {
	return nil
}
