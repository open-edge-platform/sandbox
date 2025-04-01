// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type RemoteAccessConfiguration struct {
	ent.Schema
}

func (RemoteAccessConfiguration) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.Uint64("expiration_timestamp").Immutable(), field.Uint32("local_port").Optional().Unique(), field.String("user").Optional(), field.Enum("current_state").Optional().Values("REMOTE_ACCESS_STATE_UNSPECIFIED", "REMOTE_ACCESS_STATE_DELETED", "REMOTE_ACCESS_STATE_ERROR", "REMOTE_ACCESS_STATE_ENABLED"), field.Enum("desired_state").Values("REMOTE_ACCESS_STATE_UNSPECIFIED", "REMOTE_ACCESS_STATE_DELETED", "REMOTE_ACCESS_STATE_ERROR", "REMOTE_ACCESS_STATE_ENABLED"), field.String("configuration_status").Optional(), field.Enum("configuration_status_indicator").Optional().Values("STATUS_INDICATION_UNSPECIFIED", "STATUS_INDICATION_ERROR", "STATUS_INDICATION_IN_PROGRESS", "STATUS_INDICATION_IDLE"), field.Uint64("configuration_status_timestamp").Optional(), field.String("tenant_id").Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (RemoteAccessConfiguration) Edges() []ent.Edge {
	return []ent.Edge{edge.To("instance", InstanceResource.Type).Required().Unique()}
}
func (RemoteAccessConfiguration) Annotations() []schema.Annotation {
	return nil
}
func (RemoteAccessConfiguration) Indexes() []ent.Index {
	return []ent.Index{index.Fields("tenant_id")}
}
