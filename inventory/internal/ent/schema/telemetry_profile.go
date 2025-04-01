// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type TelemetryProfile struct {
	ent.Schema
}

func (TelemetryProfile) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.Enum("kind").Values("TELEMETRY_RESOURCE_KIND_UNSPECIFIED", "TELEMETRY_RESOURCE_KIND_METRICS", "TELEMETRY_RESOURCE_KIND_LOGS"), field.Uint32("metrics_interval").Optional(), field.Enum("log_level").Optional().Values("SEVERITY_LEVEL_UNSPECIFIED", "SEVERITY_LEVEL_CRITICAL", "SEVERITY_LEVEL_ERROR", "SEVERITY_LEVEL_WARN", "SEVERITY_LEVEL_INFO", "SEVERITY_LEVEL_DEBUG"), field.String("tenant_id").Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (TelemetryProfile) Edges() []ent.Edge {
	return []ent.Edge{edge.To("region", RegionResource.Type).Unique(), edge.To("site", SiteResource.Type).Unique(), edge.To("instance", InstanceResource.Type).Unique(), edge.To("group", TelemetryGroupResource.Type).Required().Unique()}
}
func (TelemetryProfile) Annotations() []schema.Annotation {
	return nil
}
func (TelemetryProfile) Indexes() []ent.Index {
	return []ent.Index{index.Fields("tenant_id")}
}
