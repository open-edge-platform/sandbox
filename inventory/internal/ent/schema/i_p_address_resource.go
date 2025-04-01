// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type IPAddressResource struct {
	ent.Schema
}

func (IPAddressResource) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.String("address").Optional(), field.Enum("desired_state").Optional().Values("IP_ADDRESS_STATE_UNSPECIFIED", "IP_ADDRESS_STATE_ERROR", "IP_ADDRESS_STATE_ASSIGNED", "IP_ADDRESS_STATE_CONFIGURED", "IP_ADDRESS_STATE_RELEASED", "IP_ADDRESS_STATE_DELETED"), field.Enum("current_state").Optional().Values("IP_ADDRESS_STATE_UNSPECIFIED", "IP_ADDRESS_STATE_ERROR", "IP_ADDRESS_STATE_ASSIGNED", "IP_ADDRESS_STATE_CONFIGURED", "IP_ADDRESS_STATE_RELEASED", "IP_ADDRESS_STATE_DELETED"), field.Enum("status").Optional().Values("IP_ADDRESS_STATUS_UNSPECIFIED", "IP_ADDRESS_STATUS_ASSIGNMENT_ERROR", "IP_ADDRESS_STATUS_ASSIGNED", "IP_ADDRESS_STATUS_CONFIGURATION_ERROR", "IP_ADDRESS_STATUS_CONFIGURED", "IP_ADDRESS_STATUS_RELEASED", "IP_ADDRESS_STATUS_ERROR"), field.String("status_detail").Optional(), field.Enum("config_method").Optional().Values("IP_ADDRESS_CONFIG_METHOD_UNSPECIFIED", "IP_ADDRESS_CONFIG_METHOD_STATIC", "IP_ADDRESS_CONFIG_METHOD_DYNAMIC"), field.String("tenant_id").Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (IPAddressResource) Edges() []ent.Edge {
	return []ent.Edge{edge.To("nic", HostnicResource.Type).Required().Unique()}
}
func (IPAddressResource) Annotations() []schema.Annotation {
	return nil
}
func (IPAddressResource) Indexes() []ent.Index {
	return []ent.Index{index.Fields("tenant_id")}
}
