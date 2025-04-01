// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers

import (
	"fmt"
	"strings"

	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	inventory "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	locationv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/location/v1"
	ouv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/ou/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

// ProtoSiteProxyFields projects the proxy fields of the Site resource.
// These fields correspond to the proxy fields name defined in the
// SiteResource proto message.
var ProtoSiteProxyFields = []string{
	locationv1.SiteResourceFieldFtpProxy,
	locationv1.SiteResourceFieldHttpProxy,
	locationv1.SiteResourceFieldHttpsProxy,
	locationv1.SiteResourceFieldNoProxy,
}

// OpenAPISiteToProto maps OpenAPI fields name to Proto fields name.
// The key is derived from the json property respectively of the
// structs SiteTemplate defined in edge-infrastructure-manager-openapi-types.gen.go.
var OpenAPISiteToProto = map[string]string{
	"dnsServers":       locationv1.SiteResourceFieldDnsServers,
	"dockerRegistries": locationv1.SiteResourceFieldDockerRegistries,
	"siteLat":          locationv1.SiteResourceFieldSiteLat,
	"siteLng":          locationv1.SiteResourceFieldSiteLng,
	"metadata":         locationv1.SiteResourceFieldMetadata,
	"metricsEndpoint":  locationv1.SiteResourceFieldMetricsEndpoint,
	"name":             locationv1.SiteResourceFieldName,
	"ouId":             locationv1.SiteResourceEdgeOu,
	"regionId":         locationv1.SiteResourceEdgeRegion,
	"ftpProxy":         ProtoSiteProxyFields[0],
	"httpProxy":        ProtoSiteProxyFields[1],
	"httpsProxy":       ProtoSiteProxyFields[2],
	"noProxy":          ProtoSiteProxyFields[3],
}

var OpenAPISiteToProtoExcluded = map[string]struct{}{
	"siteID":            {}, // siteID must not be set from the API
	"proxy":             {}, // proxy is a sub-object translated by itself
	"inheritedMetadata": {}, // inheritedMetadata must not be set from the API
	"resourceId":        {}, // resourceId must not be set from the API
	"region":            {}, // region must not be set from the API
	"ou":                {}, // ou must not be set from the API
	"provider":          {}, // provider must not be set from the API
	"timestamps":        {}, // read-only field
}

func NewSiteHandler(client *clients.InventoryClientHandler) InventoryResource {
	return &siteHandler{
		invClient: client,
	}
}

type siteHandler struct {
	invClient *clients.InventoryClientHandler
}

func (h *siteHandler) Create(job *types.Job) (*types.Payload, error) {
	body, err := castSiteAPI(&job.Payload)
	if err != nil {
		return nil, err
	}

	site, err := openapiToGrpcSite(body)
	if err != nil {
		return nil, err
	}

	req := &inventory.Resource{
		Resource: &inventory.Resource_Site{
			Site: site,
		},
	}

	invResp, err := h.invClient.InvClient.Create(job.Context, req)
	if err != nil {
		return nil, err
	}

	createdSite := invResp.GetSite()
	obj := grpcToOpenAPISite(createdSite, nil)

	return &types.Payload{Data: obj}, err
}

func (h *siteHandler) Get(job *types.Job) (*types.Payload, error) {
	req, err := siteResourceID(&job.Payload)
	if err != nil {
		return nil, err
	}

	invResp, err := h.invClient.InvClient.Get(job.Context, req)
	if err != nil {
		return nil, err
	}

	site, meta, err := CastToSite(invResp)
	if err != nil {
		return nil, err
	}

	obj := grpcToOpenAPISite(site, meta)

	return &types.Payload{Data: obj}, nil
}

func (h *siteHandler) Update(job *types.Job) (*types.Payload, error) {
	resID, err := siteResourceID(&job.Payload)
	if err != nil {
		return nil, err
	}

	fm, err := siteFieldMask(&job.Payload, job.Operation)
	if err != nil {
		return nil, err
	}

	res, err := siteResource(&job.Payload)
	if err != nil {
		return nil, err
	}

	invResp, err := h.invClient.InvClient.Update(job.Context, resID, fm, res)
	if err != nil {
		return nil, err
	}

	updatedSite := invResp.GetSite()
	obj := grpcToOpenAPISite(updatedSite, nil)
	obj.SiteID = &resID // to be removed
	obj.ResourceId = &resID

	return &types.Payload{Data: obj}, nil
}

func (h *siteHandler) Delete(job *types.Job) error {
	req, err := siteResourceID(&job.Payload)
	if err != nil {
		return err
	}

	_, err = h.invClient.InvClient.Delete(job.Context, req)
	if err != nil {
		return err
	}

	return nil
}

func (h *siteHandler) List(job *types.Job) (*types.Payload, error) {
	filter, err := siteFilter(&job.Payload)
	if err != nil {
		return nil, err
	}

	resp, err := h.invClient.InvClient.List(job.Context, filter)
	if err != nil {
		return nil, err
	}

	sites := make([]api.Site, 0, len(resp.GetResources()))
	for _, res := range resp.GetResources() {
		site, meta, err := CastToSite(res)
		if err != nil {
			return nil, err
		}

		obj := grpcToOpenAPISite(site, meta)
		sites = append(sites, *obj)
	}

	hasNext := resp.GetHasNext()
	totalElems := int(resp.GetTotalElements())
	sitesList := api.SitesList{
		Sites:         &sites,
		HasNext:       &hasNext,
		TotalElements: &totalElems,
	}

	payload := &types.Payload{Data: sitesList}
	return payload, nil
}

func castSiteAPI(payload *types.Payload) (*api.Site, error) {
	body, ok := payload.Data.(*api.Site)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument,
			"body format is not Site: %T",
			payload.Data,
		)
		log.InfraErr(err).Msgf("")
		return nil, err
	}
	return body, nil
}

func siteResource(payload *types.Payload) (*inventory.Resource, error) {
	body, err := castSiteAPI(payload)
	if err != nil {
		return nil, err
	}

	site, err := openapiToGrpcSite(body)
	if err != nil {
		return nil, err
	}

	req := &inventory.Resource{
		Resource: &inventory.Resource_Site{
			Site: site,
		},
	}
	return req, nil
}

func siteResourceID(payload *types.Payload) (string, error) {
	siteURL, ok := payload.Params.(SiteURLParams)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument, "URLParams incorrectly formatted: %T",
			payload.Params)
		log.InfraErr(err).Msg("could not parse job payload params")
		return "", err
	}
	return siteURL.SiteID, nil
}

func siteFieldMask(payload *types.Payload, operation types.Operation) (*fieldmaskpb.FieldMask, error) {
	body, ok := payload.Data.(*api.Site)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument,
			"body format is not Site: %T",
			payload.Data,
		)
		log.InfraErr(err).Msgf("")
		return nil, err
	}

	siteRes, err := siteResource(payload)
	if err != nil {
		return nil, err
	}
	var fieldmask *fieldmaskpb.FieldMask
	if operation == types.Patch {
		fieldmask, err = getSiteFieldmask(*body)
	} else {
		fieldmask, err = fieldmaskpb.New(siteRes.GetSite(), maps.Values(OpenAPISiteToProto)...)
	}
	if err != nil {
		log.InfraErr(err).Msgf("could not create fieldmask")
		return nil, errors.Wrap(err)
	}

	return fieldmask, nil
}

func siteFilter(payload *types.Payload) (*inventory.ResourceFilter, error) {
	req := &inventory.ResourceFilter{
		Resource: &inventory.Resource{Resource: &inventory.Resource_Site{Site: &locationv1.SiteResource{}}},
	}
	if payload.Data != nil {
		query, ok := payload.Data.(api.GetSitesParams)
		if !ok {
			err := errors.Errorfc(codes.InvalidArgument,
				"GetSiteParams incorrectly formatted: %T",
				payload.Data,
			)
			log.InfraErr(err).Msg("list operation")
			return nil, err
		}
		err := siteCastQueryList(&query, req)
		if err != nil {
			log.Debug().Msgf("error parsing query parameters in list operation: %s",
				err.Error())
			return nil, err
		}
	}

	if err := validator.ValidateMessage(req); err != nil {
		log.InfraSec().InfraErr(err).Msg("failed to validate query params")
		return nil, errors.Wrap(err)
	}
	return req, nil
}

func siteCastQueryList(
	query *api.GetSitesParams,
	req *inventory.ResourceFilter,
) error {
	site := &locationv1.SiteResource{}

	err := error(nil)
	req.Limit, req.Offset, err = parsePagination(
		query.PageSize,
		query.Offset,
	)
	if err != nil {
		return err
	}
	if query.OrderBy != nil {
		req.OrderBy = *query.OrderBy
	}

	if query.Filter != nil {
		req.Filter = *query.Filter
	} else {
		var clauses []string
		if query.RegionID != nil {
			if *query.RegionID != emptyNullCase {
				clauses = append(clauses, fmt.Sprintf("%s.%s = %q", locationv1.SiteResourceEdgeRegion,
					locationv1.RegionResourceFieldResourceId, *query.RegionID))
			} else {
				clauses = append(clauses, fmt.Sprintf("NOT has(%s)", locationv1.SiteResourceEdgeRegion))
			}
		}
		if query.OuID != nil {
			if *query.OuID != emptyNullCase {
				clauses = append(clauses, fmt.Sprintf("%s.%s = %q", locationv1.SiteResourceEdgeOu,
					ouv1.OuResourceFieldResourceId, *query.OuID))
			} else {
				clauses = append(clauses, fmt.Sprintf("NOT has(%s)", locationv1.SiteResourceEdgeOu))
			}
		}
		req.Filter = strings.Join(clauses, " AND ")
	}
	req.Resource.Resource = &inventory.Resource_Site{
		Site: site,
	}
	return nil
}

// helpers method to convert between API formats.
func CastToSite(resp *inventory.GetResourceResponse) (
	*locationv1.SiteResource, *inventory.GetResourceResponse_ResourceMetadata, error,
) {
	if resp.GetResource().GetSite() != nil {
		return resp.GetResource().GetSite(), resp.GetRenderedMetadata(), nil
	}
	err := errors.Errorfc(codes.Internal, "%s is not a SiteResource", resp.GetResource())
	log.InfraErr(err).Msgf("could not cast inventory resource")
	return nil, nil, err
}

func castSiteProxy(body *api.Site, site *locationv1.SiteResource) {
	if body.Proxy.HttpProxy != nil {
		site.HttpProxy = *body.Proxy.HttpProxy
	}
	if body.Proxy.HttpsProxy != nil {
		site.HttpsProxy = *body.Proxy.HttpsProxy
	}
	if body.Proxy.FtpProxy != nil {
		site.FtpProxy = *body.Proxy.FtpProxy
	}
	if body.Proxy.NoProxy != nil {
		site.NoProxy = *body.Proxy.NoProxy
	}
}

func getSiteFieldmask(body api.Site) (*fieldmaskpb.FieldMask, error) {
	var fieldList []string
	if body.Proxy != nil {
		fieldList = append(
			fieldList,
			getProtoFieldListFromOpenapiPointer(body.Proxy, OpenAPISiteToProto)...)
	}
	fieldList = append(
		fieldList,
		getProtoFieldListFromOpenapiValue(body, OpenAPISiteToProto)...)
	log.Debug().Msgf("Proto Valid Fields: %s", fieldList)
	return fieldmaskpb.New(&locationv1.SiteResource{}, fieldList...)
}

func openapiToGrpcSite(
	body *api.Site,
) (*locationv1.SiteResource, error) {
	metadata, metaErr := marshalMetadata(body.Metadata)
	if metaErr != nil {
		log.Debug().Msgf("marshal site metadata error: %s", metaErr.Error())
	}

	var siteName string
	if body.Name != nil {
		siteName = *body.Name
	}
	site := &locationv1.SiteResource{
		Name:     siteName,
		Metadata: metadata,
	}

	if !isUnset(body.RegionId) {
		site.Region = &locationv1.RegionResource{
			ResourceId: *body.RegionId,
		}
	}
	if !isUnset(body.OuId) {
		site.Ou = &ouv1.OuResource{
			ResourceId: *body.OuId,
		}
	}

	err := error(nil)
	site.SiteLat, site.SiteLng, err = openAPILatLongToGrpc(body.SiteLat, body.SiteLng)
	if err != nil {
		return nil, err
	}

	if body.DnsServers != nil {
		site.DnsServers = *body.DnsServers
	}

	if body.Proxy != nil {
		castSiteProxy(body, site)
	}
	err = validator.ValidateMessage(site)
	if err != nil {
		log.InfraSec().InfraErr(err).Msg("could not validate inventory resource")
		return nil, errors.Wrap(err)
	}
	return site, nil
}

func openAPILatLongToGrpc(oapiLat, oapiLng *int) (lat, lng int32, err error) {
	if oapiLat != nil {
		lat, err = util.IntToInt32(*oapiLat)
		if err != nil {
			return 0, 0, err
		}
	}
	if oapiLng != nil {
		lng, err = util.IntToInt32(*oapiLng)
		if err != nil {
			return 0, 0, err
		}
	}
	return lat, lng, err
}

func grpcToOpenAPISite(
	site *locationv1.SiteResource,
	meta *inventory.GetResourceResponse_ResourceMetadata,
) *api.Site {
	name := site.GetName()
	resourceID := site.GetResourceId()
	// Safe casts, int32 to int
	siteLat := int(site.GetSiteLat())
	siteLng := int(site.GetSiteLng())
	DNSServers := site.GetDnsServers()
	dockerRegistries := site.GetDockerRegistries()
	endpointMetrics := site.GetMetricsEndpoint()
	proxyHTTPProxy := site.GetHttpProxy()
	proxyHTTPSProxy := site.GetHttpProxy()
	proxyFtpProxy := site.GetFtpProxy()
	proxyNoProxy := site.GetNoProxy()

	metadata, metaErr := unmarshalMetadata(site.GetMetadata())
	if metaErr != nil {
		log.Debug().Msgf("unmarshal site metadata error: %s", metaErr.Error())
	}

	var regionID *string
	var ouID *string

	siteRegion := site.GetRegion()
	siteOU := site.GetOu()
	siteProvider := site.GetProvider()

	if siteRegion != nil {
		regionID = getPtr(siteRegion.GetResourceId())
	}
	if siteOU != nil {
		ouID = getPtr(siteOU.GetResourceId())
	}

	var proxy api.Proxy

	obj := api.Site{
		SiteID:           &resourceID,
		Name:             &name,
		OuId:             ouID,
		RegionId:         regionID,
		SiteLat:          &siteLat,
		SiteLng:          &siteLng,
		DnsServers:       &DNSServers,
		DockerRegistries: &dockerRegistries,
		Proxy:            &proxy,
		MetricsEndpoint:  &endpointMetrics,
		Metadata:         metadata,
		ResourceId:       &resourceID,
		Timestamps:       GrpcToOpenAPITimestamps(site),
	}

	if siteRegion != nil {
		obj.Region = grpcToOpenAPIRegion(siteRegion, nil)
	}
	if siteOU != nil {
		obj.Ou = grpcToOpenAPIOU(siteOU, nil)
	}
	if siteProvider != nil {
		obj.Provider = GrpcProviderToOpenAPIProvider(siteProvider)
	}

	obj.Proxy.HttpProxy = &proxyHTTPProxy
	obj.Proxy.HttpsProxy = &proxyHTTPSProxy
	obj.Proxy.FtpProxy = &proxyFtpProxy
	obj.Proxy.NoProxy = &proxyNoProxy

	if meta != nil {
		obj.InheritedMetadata = &api.MetadataJoin{}
		obj.InheritedMetadata.Location, metaErr = unmarshalMetadata(meta.GetPhyMetadata())
		if metaErr != nil {
			log.Debug().Msgf("unmarshal site rendered location metadata error: %s", metaErr.Error())
		}

		obj.InheritedMetadata.Ou, metaErr = unmarshalMetadata(meta.GetLogiMetadata())
		if metaErr != nil {
			log.Debug().Msgf("unmarshal site rendered OU metadata error: %s", metaErr.Error())
		}
	}
	return &obj
}
