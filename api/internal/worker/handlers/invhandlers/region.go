// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers

import (
	"fmt"

	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	inventory "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	locationv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/location/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	inv_utils "github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

// OpenAPIRegionToProto maps OpenAPI fields name to Proto fields name.
// The key is derived from the json property respectively of the
// structs RegionTemplate defined in edge-infrastructure-manager-openapi-types.gen.go.
var OpenAPIRegionToProto = map[string]string{
	"metadata": locationv1.RegionResourceFieldMetadata,
	"name":     locationv1.RegionResourceFieldName,
	"parentId": locationv1.RegionResourceEdgeParentRegion,
}

// OpenAPIRegionToProtoExcluded defines exclusion rules as there are some fields
// defined in the OpenAPI spec that are not currently mapped to the proto
// fields.
var OpenAPIRegionToProtoExcluded = map[string]struct{}{
	"regionID":          {}, // regionID must not be set from the API
	"inheritedMetadata": {}, // inheritedMetadata must not be set from the API
	"parentRegion":      {}, // parentRegion must not be set from the API
	"resourceId":        {}, // resourceId must not be set from the API
	"totalSites":        {}, // totalSites is a readonly field
	"timestamps":        {}, // read-only field
}

func NewRegionHandler(invClient *clients.InventoryClientHandler) InventoryResource {
	return &regionHandler{invClient: invClient}
}

type regionHandler struct {
	invClient *clients.InventoryClientHandler
}

func (h *regionHandler) Create(job *types.Job) (*types.Payload, error) {
	body, err := castRegionAPI(&job.Payload)
	if err != nil {
		return nil, err
	}

	region, err := openapiToGrpcRegion(body)
	if err != nil {
		return nil, err
	}

	req := &inventory.Resource{
		Resource: &inventory.Resource_Region{
			Region: region,
		},
	}

	invResp, err := h.invClient.InvClient.Create(job.Context, req)
	if err != nil {
		return nil, err
	}

	createdRegion := invResp.GetRegion()
	obj := grpcToOpenAPIRegion(createdRegion, nil)
	return &types.Payload{Data: obj}, err
}

func (h *regionHandler) Get(job *types.Job) (*types.Payload, error) {
	req, err := regionResourceID(&job.Payload)
	if err != nil {
		return nil, err
	}

	invResp, err := h.invClient.InvClient.Get(job.Context, req)
	if err != nil {
		return nil, err
	}

	region, meta, err := CastToRegion(invResp)
	if err != nil {
		return nil, err
	}

	obj := grpcToOpenAPIRegion(region, meta)

	return &types.Payload{Data: obj}, nil
}

func (h *regionHandler) Update(job *types.Job) (*types.Payload, error) {
	resID, err := regionResourceID(&job.Payload)
	if err != nil {
		return nil, err
	}

	fm, err := regionFieldMask(&job.Payload, job.Operation)
	if err != nil {
		return nil, err
	}

	res, err := regionResource(&job.Payload)
	if err != nil {
		return nil, err
	}

	invResp, err := h.invClient.InvClient.Update(job.Context, resID, fm, res)
	if err != nil {
		return nil, err
	}

	updatedRegion := invResp.GetRegion()
	obj := grpcToOpenAPIRegion(updatedRegion, nil)
	obj.RegionID = &resID // to be removed
	obj.ResourceId = &resID

	return &types.Payload{Data: obj}, nil
}

func (h *regionHandler) Delete(job *types.Job) error {
	req, err := regionResourceID(&job.Payload)
	if err != nil {
		return err
	}

	_, err = h.invClient.InvClient.Delete(job.Context, req)
	if err != nil {
		return err
	}

	return nil
}

func showSitesPerRegion(payload *types.Payload) (bool, error) {
	var showTotalSites bool
	if payload.Data != nil {
		query, ok := payload.Data.(api.GetRegionsParams)
		if !ok {
			err := errors.Errorfc(codes.InvalidArgument,
				"GetRegionsParams incorrectly formatted: %T",
				payload.Data,
			)
			log.InfraErr(err).Msg("list operation")
			return false, err
		}
		if query.ShowTotalSites != nil {
			showTotalSites = *query.ShowTotalSites
		}
	}

	return showTotalSites, nil
}

func (h *regionHandler) getSitesPerRegion(job *types.Job, resp *inventory.ListResourcesResponse) (map[string]int, error) {
	sitesPerRegions := make(map[string]int, len(resp.GetResources()))

	showTotalSites, err := showSitesPerRegion(&job.Payload)
	if err != nil {
		return nil, err
	}

	if !showTotalSites || len(resp.GetResources()) == 0 {
		return sitesPerRegions, nil
	}

	regionIDs := []string{}
	for _, res := range resp.GetResources() {
		resID, errGet := inv_utils.GetResourceIDFromResource(res.GetResource())
		if errGet != nil {
			log.InfraErr(errGet).Msgf("resource %v has no resourceID", res)
			continue
		}
		regionIDs = append(regionIDs, resID)
	}

	request := &inventory.GetSitesPerRegionRequest{
		Filter: regionIDs,
	}

	response, err := h.invClient.InvClient.GetSitesPerRegion(job.Context, request)
	if err != nil {
		return nil, err
	}

	collections.MapSlice[*inventory.GetSitesPerRegionResponse_Node, int](
		response.GetRegions(),
		func(tN *inventory.GetSitesPerRegionResponse_Node) int {
			childSites := int(tN.GetChildSites())
			sitesPerRegions[tN.GetResourceId()] = childSites
			return childSites
		},
	)

	return sitesPerRegions, nil
}

func (h *regionHandler) List(job *types.Job) (*types.Payload, error) {
	filter, err := regionFilter(&job.Payload)
	if err != nil {
		return nil, err
	}

	resp, err := h.invClient.InvClient.List(job.Context, filter)
	if err != nil {
		return nil, err
	}

	sitesPerRegions, err := h.getSitesPerRegion(job, resp)
	if err != nil {
		return nil, err
	}

	regions := make([]api.Region, 0, len(resp.GetResources()))
	for _, res := range resp.GetResources() {
		region, meta, err := CastToRegion(res)
		if err != nil {
			return nil, err
		}

		obj := grpcToOpenAPIRegion(region, meta)
		if totalSites, hasTotalSites := sitesPerRegions[region.GetResourceId()]; hasTotalSites {
			obj.TotalSites = &totalSites
		}

		regions = append(regions, *obj)
	}

	hasNext := resp.GetHasNext()
	totalElems := int(resp.GetTotalElements())
	regionsList := api.RegionsList{
		Regions:       &regions,
		HasNext:       &hasNext,
		TotalElements: &totalElems,
	}

	payload := &types.Payload{Data: regionsList}
	return payload, nil
}

func castRegionAPI(payload *types.Payload) (*api.Region, error) {
	body, ok := payload.Data.(*api.Region)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument,
			"body format is not RegionRequest: %T",
			payload.Data,
		)
		log.InfraErr(err).Msgf("")
		return nil, err
	}
	return body, nil
}

func regionResource(payload *types.Payload) (*inventory.Resource, error) {
	body, err := castRegionAPI(payload)
	if err != nil {
		return nil, err
	}

	region, err := openapiToGrpcRegion(body)
	if err != nil {
		return nil, err
	}

	req := &inventory.Resource{
		Resource: &inventory.Resource_Region{
			Region: region,
		},
	}
	return req, nil
}

func regionFilter(payload *types.Payload) (*inventory.ResourceFilter, error) {
	req := &inventory.ResourceFilter{
		Resource: &inventory.Resource{Resource: &inventory.Resource_Region{Region: &locationv1.RegionResource{}}},
	}
	if payload.Data != nil {
		query, ok := payload.Data.(api.GetRegionsParams)
		if !ok {
			err := errors.Errorfc(codes.InvalidArgument,
				"GetRegionsParams incorrectly formatted: %T",
				payload.Data,
			)
			log.InfraErr(err).Msg("list operation")
			return nil, err
		}
		err := castRegionQueryList(&query, req)
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

func regionResourceID(payload *types.Payload) (string, error) {
	params, ok := payload.Params.(RegionURLParams)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument, "RegionURLParams incorrectly formatted: %T",
			payload.Data)
		log.InfraErr(err).Msg("could not parse job payload params")
		return "", err
	}
	return params.RegionID, nil
}

func regionFieldMask(payload *types.Payload, operation types.Operation) (*fieldmaskpb.FieldMask, error) {
	body, ok := payload.Data.(*api.Region)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument,
			"body format is not RegionRequest: %T",
			payload.Data,
		)
		log.InfraErr(err).Msgf("")
		return nil, err
	}

	regionRes, err := regionResource(payload)
	if err != nil {
		return nil, err
	}

	var fieldmask *fieldmaskpb.FieldMask
	if operation == types.Patch {
		fieldmask, err = getRegionFieldmask(*body)
	} else {
		fieldmask, err = fieldmaskpb.New(regionRes.GetRegion(), maps.Values(OpenAPIRegionToProto)...)
	}
	if err != nil {
		log.InfraErr(err).Msgf("could not create fieldmask")
		return nil, errors.Wrap(err)
	}

	return fieldmask, nil
}

func castRegionQueryList(
	query *api.GetRegionsParams,
	req *inventory.ResourceFilter,
) error {
	region := &locationv1.RegionResource{}

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
	} else if query.Parent != nil {
		if *query.Parent != emptyNullCase {
			req.Filter = fmt.Sprintf("%s.%s = %q", locationv1.RegionResourceEdgeParentRegion,
				locationv1.RegionResourceFieldResourceId, *query.Parent)
		} else {
			req.Filter = fmt.Sprintf("NOT has(%s)", locationv1.RegionResourceEdgeParentRegion)
		}
	}
	req.Resource.Resource = &inventory.Resource_Region{
		Region: region,
	}
	return nil
}

// helpers method to convert between API formats.
func CastToRegion(resp *inventory.GetResourceResponse) (
	*locationv1.RegionResource, *inventory.GetResourceResponse_ResourceMetadata, error,
) {
	if resp.GetResource().GetRegion() != nil {
		return resp.GetResource().GetRegion(), resp.GetRenderedMetadata(), nil
	}
	err := errors.Errorfc(codes.Internal, "%s is not a RegionResource", resp.GetResource())
	log.InfraErr(err).Msgf("could not cast inventory resource")
	return nil, nil, err
}

func getRegionFieldmask(body api.Region) (*fieldmaskpb.FieldMask, error) {
	fieldList := getProtoFieldListFromOpenapiValue(body, OpenAPIRegionToProto)
	log.Debug().Msgf("Proto Valid Fields: %s", fieldList)
	return fieldmaskpb.New(&locationv1.RegionResource{}, fieldList...)
}

func openapiToGrpcRegion(body *api.Region) (*locationv1.RegionResource, error) {
	metadata, metaErr := marshalMetadata(body.Metadata)
	if metaErr != nil {
		log.Debug().Msgf("marshal region metadata error: %s", metaErr.Error())
	}

	// Name is not required in API nor in Inventory, setting empty
	var regionName string
	if body.Name != nil {
		regionName = *body.Name
	}

	region := &locationv1.RegionResource{
		Name:     regionName,
		Metadata: metadata,
	}

	if !isUnset(body.ParentId) {
		parentRegion := *body.ParentId
		region.ParentRegion = &locationv1.RegionResource{
			ResourceId: parentRegion,
		}
	}
	err := validator.ValidateMessage(region)
	if err != nil {
		log.InfraSec().InfraErr(err).Msg("could not validate inventory resource")
		return nil, errors.Wrap(err)
	}

	return region, nil
}

func grpcToOpenAPIRegion(
	region *locationv1.RegionResource,
	meta *inventory.GetResourceResponse_ResourceMetadata,
) *api.Region {
	var parentRegionID *string

	metadata, metaErr := unmarshalMetadata(region.GetMetadata())
	if metaErr != nil {
		log.Debug().Msgf("unmarshal region metadata error: %s", metaErr.Error())
	}

	parentRegion := region.GetParentRegion()
	if parentRegion != nil {
		parentRegionID = getPtr(parentRegion.GetResourceId())
	}

	resourceID := region.GetResourceId()
	regionName := region.GetName()
	obj := api.Region{
		RegionID:   &resourceID,
		Name:       &regionName,
		ParentId:   parentRegionID,
		Metadata:   metadata,
		ResourceId: &resourceID,
		Timestamps: GrpcToOpenAPITimestamps(region),
	}
	if parentRegion != nil {
		obj.ParentRegion = grpcToOpenAPIRegion(parentRegion, nil)
	}
	if meta != nil {
		obj.InheritedMetadata = &api.Metadata{}
		obj.InheritedMetadata, metaErr = unmarshalMetadata(meta.GetPhyMetadata())
		if metaErr != nil {
			log.Debug().Msgf("unmarshal region rendered location metadata error: %s", metaErr.Error())
		}
	}
	return &obj
}
