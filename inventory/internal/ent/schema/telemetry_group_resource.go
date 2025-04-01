// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type TelemetryGroupResource struct {
	ent.Schema
}

func (TelemetryGroupResource) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.String("name"), field.Enum("kind").Values("TELEMETRY_RESOURCE_KIND_UNSPECIFIED", "TELEMETRY_RESOURCE_KIND_METRICS", "TELEMETRY_RESOURCE_KIND_LOGS"), field.Enum("collector_kind").Values("COLLECTOR_KIND_UNSPECIFIED", "COLLECTOR_KIND_HOST", "COLLECTOR_KIND_CLUSTER"), field.String("groups"), field.String("tenant_id").Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (TelemetryGroupResource) Edges() []ent.Edge {
	return []ent.Edge{edge.From("profiles", TelemetryProfile.Type).Ref("group")}
}
func (TelemetryGroupResource) Annotations() []schema.Annotation {
	return nil
}
func (TelemetryGroupResource) Indexes() []ent.Index {
	return []ent.Index{index.Fields("tenant_id")}
}
