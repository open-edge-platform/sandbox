// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type SingleScheduleResource struct {
	ent.Schema
}

func (SingleScheduleResource) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.Enum("schedule_status").Optional().Values("SCHEDULE_STATUS_UNSPECIFIED", "SCHEDULE_STATUS_MAINTENANCE", "SCHEDULE_STATUS_SHIPPING", "SCHEDULE_STATUS_OS_UPDATE", "SCHEDULE_STATUS_FIRMWARE_UPDATE", "SCHEDULE_STATUS_CLUSTER_UPDATE"), field.String("name").Optional(), field.Uint64("start_seconds"), field.Uint64("end_seconds").Optional(), field.String("tenant_id").Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (SingleScheduleResource) Edges() []ent.Edge {
	return []ent.Edge{edge.To("target_site", SiteResource.Type).Unique(), edge.To("target_host", HostResource.Type).Unique(), edge.To("target_workload", WorkloadResource.Type).Unique(), edge.To("target_region", RegionResource.Type).Unique()}
}
func (SingleScheduleResource) Annotations() []schema.Annotation {
	return nil
}
func (SingleScheduleResource) Indexes() []ent.Index {
	return []ent.Index{index.Fields("tenant_id")}
}
