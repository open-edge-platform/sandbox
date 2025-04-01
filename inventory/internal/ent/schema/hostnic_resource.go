// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type HostnicResource struct {
	ent.Schema
}

func (HostnicResource) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.String("kind").Optional(), field.String("provider_status").Optional(), field.String("device_name").Optional(), field.String("pci_identifier").Optional(), field.String("mac_addr").Optional(), field.Bool("sriov_enabled").Optional(), field.Uint32("sriov_vfs_num").Optional(), field.Uint32("sriov_vfs_total").Optional(), field.String("peer_name").Optional(), field.String("peer_description").Optional(), field.String("peer_mac").Optional(), field.String("peer_mgmt_ip").Optional(), field.String("peer_port").Optional(), field.String("supported_link_mode").Optional(), field.String("advertising_link_mode").Optional(), field.Uint64("current_speed_bps").Optional(), field.String("current_duplex").Optional(), field.String("features").Optional(), field.Uint32("mtu").Optional(), field.Enum("link_state").Optional().Values("NETWORK_INTERFACE_LINK_STATE_UNSPECIFIED", "NETWORK_INTERFACE_LINK_STATE_UP", "NETWORK_INTERFACE_LINK_STATE_DOWN"), field.Bool("bmc_interface").Optional(), field.String("tenant_id").Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (HostnicResource) Edges() []ent.Edge {
	return []ent.Edge{edge.To("host", HostResource.Type).Required().Unique()}
}
func (HostnicResource) Annotations() []schema.Annotation {
	return nil
}
func (HostnicResource) Indexes() []ent.Index {
	return []ent.Index{index.Fields("tenant_id")}
}
