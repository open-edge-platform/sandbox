// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type WorkloadMember struct {
	ent.Schema
}

func (WorkloadMember) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.Enum("kind").Values("WORKLOAD_MEMBER_KIND_UNSPECIFIED", "WORKLOAD_MEMBER_KIND_CLUSTER_NODE"), field.String("tenant_id").Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (WorkloadMember) Edges() []ent.Edge {
	return []ent.Edge{edge.To("workload", WorkloadResource.Type).Required().Unique(), edge.To("instance", InstanceResource.Type).Required().Unique()}
}
func (WorkloadMember) Annotations() []schema.Annotation {
	return nil
}
