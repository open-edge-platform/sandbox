// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type NetlinkResource struct {
	ent.Schema
}

func (NetlinkResource) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.String("kind").Optional(), field.String("name").Optional(), field.Enum("desired_state").Values("NETLINK_STATE_UNSPECIFIED", "NETLINK_STATE_DELETED", "NETLINK_STATE_ONLINE", "NETLINK_STATE_OFFLINE", "NETLINK_STATE_ERROR"), field.Enum("current_state").Optional().Values("NETLINK_STATE_UNSPECIFIED", "NETLINK_STATE_DELETED", "NETLINK_STATE_ONLINE", "NETLINK_STATE_OFFLINE", "NETLINK_STATE_ERROR"), field.String("provider_status").Optional(), field.String("tenant_id").Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (NetlinkResource) Edges() []ent.Edge {
	return []ent.Edge{edge.To("src", EndpointResource.Type).Unique(), edge.To("dst", EndpointResource.Type).Unique()}
}
func (NetlinkResource) Annotations() []schema.Annotation {
	return nil
}
func (NetlinkResource) Indexes() []ent.Index {
	return []ent.Index{index.Fields("tenant_id")}
}
