// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type RepeatedScheduleResource struct {
	ent.Schema
}

func (RepeatedScheduleResource) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.Enum("schedule_status").Optional().Values("SCHEDULE_STATUS_UNSPECIFIED", "SCHEDULE_STATUS_MAINTENANCE", "SCHEDULE_STATUS_SHIPPING", "SCHEDULE_STATUS_OS_UPDATE", "SCHEDULE_STATUS_FIRMWARE_UPDATE", "SCHEDULE_STATUS_CLUSTER_UPDATE"), field.String("name").Optional(), field.Uint32("duration_seconds").Optional(), field.String("cron_minutes"), field.String("cron_hours"), field.String("cron_day_month"), field.String("cron_month"), field.String("cron_day_week"), field.String("tenant_id").Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (RepeatedScheduleResource) Edges() []ent.Edge {
	return []ent.Edge{edge.To("target_site", SiteResource.Type).Unique(), edge.To("target_host", HostResource.Type).Unique(), edge.To("target_workload", WorkloadResource.Type).Unique(), edge.To("target_region", RegionResource.Type).Unique()}
}
func (RepeatedScheduleResource) Annotations() []schema.Annotation {
	return nil
}
func (RepeatedScheduleResource) Indexes() []ent.Index {
	return []ent.Index{index.Fields("tenant_id")}
}
