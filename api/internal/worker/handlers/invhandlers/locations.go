// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	inventory "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	locationv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/location/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

const (
	defaultOffset = 0
	defaultLimit  = 50
)

func NewLocationsHandler(invClient *clients.InventoryClientHandler) InventoryResource {
	return &locationsHandler{invClient: invClient}
}

type locationsHandler struct {
	invClient *clients.InventoryClientHandler
}

func (h *locationsHandler) Create(_ *types.Job) (*types.Payload, error) {
	err := errors.Errorfc(codes.Unimplemented, "unsupported endpoint for locations")
	return nil, err
}

func (h *locationsHandler) Get(_ *types.Job) (*types.Payload, error) {
	err := errors.Errorfc(codes.Unimplemented, "unsupported endpoint for locations")
	return nil, err
}

func (h *locationsHandler) Update(_ *types.Job) (*types.Payload, error) {
	err := errors.Errorfc(codes.Unimplemented, "unsupported endpoint for locations")
	return nil, err
}

func (h *locationsHandler) Delete(_ *types.Job) error {
	err := errors.Errorfc(codes.Unimplemented, "unsupported endpoint for locations")
	return err
}

func (h *locationsHandler) validateQueryParams(query api.GetLocationsParams) error {
	err := errors.Errorfc(codes.InvalidArgument, "GetLocationsParams incorrectly provided")
	if query.Name == nil {
		log.InfraErr(err).Msg("failed to validate GetLocationsParams," +
			"missing name parameter")
		return err
	}

	if query.ShowRegions == nil && query.ShowSites == nil {
		log.InfraErr(err).Msg("failed to validate GetLocationsParams," +
			"at least one of showSites or showRegions must be provided")
		return err
	}

	return nil
}

func (h *locationsHandler) getLocationIDs(
	ctx context.Context,
	query api.GetLocationsParams,
	resType inventory.ResourceKind,
) (locationIDs []string, totalElements int, err error) {
	var resource *inventory.Resource
	switch resType {
	case inventory.ResourceKind_RESOURCE_KIND_REGION:
		resource = &inventory.Resource{Resource: &inventory.Resource_Region{Region: &locationv1.RegionResource{}}}
	case inventory.ResourceKind_RESOURCE_KIND_SITE:
		resource = &inventory.Resource{Resource: &inventory.Resource_Site{Site: &locationv1.SiteResource{}}}
	default:
		err = errors.Errorfc(codes.Internal, "invalid resource type %v", resType)
		return nil, 0, errors.Wrap(err)
	}

	filterbyName := fmt.Sprintf(`%s = %q`, "name", *query.Name)
	filter := &inventory.ResourceFilter{
		Resource: resource,
		Filter:   filterbyName,
		Offset:   defaultOffset,
		Limit:    defaultLimit,
		OrderBy:  "name asc",
	}
	if err = validator.ValidateMessage(filter); err != nil {
		log.InfraSec().InfraErr(err).Msg("failed to validate query params")
		return nil, 0, errors.Wrap(err)
	}

	findResp, err := h.invClient.InvClient.Find(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	locationIDs = collections.MapSlice[*client.ResourceTenantIDCarrier, string](findResp.GetResources(),
		func(carrier *client.ResourceTenantIDCarrier) string {
			return carrier.GetResourceId()
		})
	totalElements = int(findResp.GetTotalElements())
	return locationIDs, totalElements, nil
}

func getLocationType(invKind inventory.ResourceKind) (*api.LocationType, error) {
	locMap := map[inventory.ResourceKind]api.LocationType{
		inventory.ResourceKind_RESOURCE_KIND_REGION: api.RESOURCEKINDREGION,
		inventory.ResourceKind_RESOURCE_KIND_SITE:   api.RESOURCEKINDSITE,
	}

	apiType, ok := locMap[invKind]
	if !ok {
		err := errors.Errorfc(codes.Internal, "invalid location type %v", invKind)
		return nil, errors.Wrap(err)
	}
	return &apiType, nil
}

func inventoryTreeToAPITree(
	treeResp []*inventory.GetTreeHierarchyResponse_TreeNode,
) (*[]api.LocationNode, error) {
	apiLocNodes := []api.LocationNode{}

	// Filters tree response (removes possible OU or Host resources in the response)
	locationNodes := []*inventory.GetTreeHierarchyResponse_TreeNode{}
	for _, node := range treeResp {
		if node.CurrentNode.ResourceKind == inventory.ResourceKind_RESOURCE_KIND_REGION ||
			node.CurrentNode.ResourceKind == inventory.ResourceKind_RESOURCE_KIND_SITE {
			locationNodes = append(locationNodes, node)
		}
	}

	for _, locNode := range locationNodes {
		resID := locNode.GetCurrentNode().GetResourceId()
		resName := locNode.GetName()
		resType, err := getLocationType(locNode.GetCurrentNode().GetResourceKind())
		if err != nil {
			return nil, err
		}

		// Filters parentIDs (removes possible OU resources)
		parentIDs := []string{}
		for _, parentNode := range locNode.GetParentNodes() {
			if parentNode.ResourceKind == inventory.ResourceKind_RESOURCE_KIND_REGION ||
				parentNode.ResourceKind == inventory.ResourceKind_RESOURCE_KIND_SITE {
				parentIDs = append(parentIDs, parentNode.GetResourceId())
			}
		}
		var parentID string
		// Region and Site have only one parentID, when existent.
		if len(parentIDs) > 0 {
			parentID = parentIDs[0]
		} else {
			log.Warn().Msgf("Invalid location parent IDs resource %s"+
				"has more than one parent of type region or site %v", resID, parentIDs)
		}
		locNode := api.LocationNode{
			ResourceId: &resID,
			Name:       &resName,
			ParentId:   &parentID,
			Type:       resType,
		}
		apiLocNodes = append(apiLocNodes, locNode)
	}

	return &apiLocNodes, nil
}

func (h *locationsHandler) getLocationTree(
	ctx context.Context,
	locationIDs []string,
) (api.LocationNodeList, error) {
	apiLocList := api.LocationNodeList{
		Nodes: &[]api.LocationNode{},
	}

	if len(locationIDs) == 0 {
		return apiLocList, nil
	}

	request := &inventory.GetTreeHierarchyRequest{
		Filter:     locationIDs,
		Descending: true,
	}
	treeResp, err := h.invClient.InvClient.GetTreeHierarchy(ctx, request)
	if err != nil {
		return apiLocList, err
	}

	apiLocNodes, err := inventoryTreeToAPITree(treeResp)
	if err != nil {
		return apiLocList, err
	}

	apiLocList.Nodes = apiLocNodes
	return apiLocList, nil
}

func (h *locationsHandler) getAllLocationIDs(
	ctx context.Context,
	query api.GetLocationsParams,
) (allIDs []string, totalElements, outputElements int, err error) {
	allIDs = []string{}
	totalElements = 0
	outputElements = 0
	if query.ShowRegions != nil && *query.ShowRegions {
		regionIDs, totalRegions, errReg := h.getLocationIDs(ctx, query, inventory.ResourceKind_RESOURCE_KIND_REGION)
		if errReg != nil {
			return nil, 0, 0, errReg
		}

		allIDs = append(allIDs, regionIDs...)
		totalElements += totalRegions
		outputElements += len(regionIDs)
	}

	if query.ShowSites != nil && *query.ShowSites {
		siteIDs, totalSites, errSit := h.getLocationIDs(ctx, query, inventory.ResourceKind_RESOURCE_KIND_SITE)
		if errSit != nil {
			return nil, 0, 0, errSit
		}
		allIDs = append(allIDs, siteIDs...)
		totalElements += totalSites
		outputElements += len(siteIDs)
	}

	return allIDs, totalElements, outputElements, nil
}

func (h *locationsHandler) List(job *types.Job) (*types.Payload, error) {
	query, ok := job.Payload.Data.(api.GetLocationsParams)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument,
			"GetLocationsParams incorrectly formatted: %T",
			job.Payload.Data,
		)
		log.InfraErr(err).Msg("list operation")
		return nil, err
	}

	err := h.validateQueryParams(query)
	if err != nil {
		return nil, err
	}

	allIDs, totalElements, outputElements, err := h.getAllLocationIDs(job.Context, query)
	if err != nil {
		return nil, err
	}

	locationsList, err := h.getLocationTree(job.Context, allIDs)
	if err != nil {
		return nil, err
	}

	locationsList.TotalElements = &totalElements
	locationsList.OutputElements = &outputElements
	payload := &types.Payload{Data: locationsList}
	return payload, nil
}
