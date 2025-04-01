// File updated by protoc-gen-ent.

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type SiteResource struct {
	ent.Schema
}

func (SiteResource) Fields() []ent.Field {
	return []ent.Field{field.String("resource_id").Unique(), field.String("name").Optional(), field.String("address").Optional(), field.Int32("site_lat").Optional(), field.Int32("site_lng").Optional(), field.String("dns_servers").Optional(), field.String("docker_registries").Optional(), field.String("metrics_endpoint").Optional(), field.String("http_proxy").Optional(), field.String("https_proxy").Optional(), field.String("ftp_proxy").Optional(), field.String("no_proxy").Optional(), field.String("metadata").Optional(), field.String("tenant_id").Immutable(), field.String("created_at").Immutable().SchemaType(map[string]string{"postgres": "TIMESTAMP"}), field.String("updated_at").SchemaType(map[string]string{"postgres": "TIMESTAMP"})}
}
func (SiteResource) Edges() []ent.Edge {
	return []ent.Edge{edge.To("region", RegionResource.Type).Unique(), edge.To("ou", OuResource.Type).Unique(), edge.To("provider", ProviderResource.Type).Unique()}
}
func (SiteResource) Annotations() []schema.Annotation {
	return nil
}
func (SiteResource) Indexes() []ent.Index {
	return []ent.Index{index.Fields("tenant_id")}
}
