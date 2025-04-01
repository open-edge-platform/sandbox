// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type WorkloadResource struct {
	ent.Schema
}

func (WorkloadResource) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.Enum("kind").Values("WORKLOAD_KIND_UNSPECIFIED", "WORKLOAD_KIND_CLUSTER", "WORKLOAD_KIND_DHCP"), field.String("name").Optional(), field.String("external_id").Optional(), field.Enum("desired_state").Values("WORKLOAD_STATE_UNSPECIFIED", "WORKLOAD_STATE_ERROR", "WORKLOAD_STATE_DELETING", "WORKLOAD_STATE_DELETED", "WORKLOAD_STATE_PROVISIONED"), field.Enum("current_state").Optional().Values("WORKLOAD_STATE_UNSPECIFIED", "WORKLOAD_STATE_ERROR", "WORKLOAD_STATE_DELETING", "WORKLOAD_STATE_DELETED", "WORKLOAD_STATE_PROVISIONED"), field.String("status").Optional(), field.String("metadata").Optional(), field.String("tenant_id").Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (WorkloadResource) Edges() []ent.Edge {
	return []ent.Edge{edge.From("members", WorkloadMember.Type).Ref("workload")}
}
func (WorkloadResource) Annotations() []schema.Annotation {
	return nil
}
func (WorkloadResource) Indexes() []ent.Index {
	return []ent.Index{index.Fields("external_id", "tenant_id").Unique(), index.Fields("tenant_id")}
}
