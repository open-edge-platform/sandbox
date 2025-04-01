// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/internal/worker/handlers"
	inv_handlers "github.com/open-edge-platform/infra-core/api/internal/worker/handlers/invhandlers"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	"github.com/open-edge-platform/infra-core/api/test/utils"
	inventory "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	location_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/location/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

var (
	regionName        = "Testregion"
	showTotalSites    = true
	filterHasNoParent = fmt.Sprintf(`NOT has(%s)`, "parent_region")
)

func BuildFmFromRegionRequest(body api.Region) []string {
	fm := []string{}
	fm = append(fm, "name")
	if body.ParentId != nil {
		fm = append(fm, "parent_region")
	}
	if body.Metadata != nil {
		fm = append(fm, "metadata")
	}
	return fm
}

func Test_regionHandler_Job_Error(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	job := types.NewJob(
		context.TODO(), BadOperation, types.Region,
		nil, nil,
	)
	_, err := h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))
}

// check that we pass the expected filters to the inventory.
func Test_regionHandler_List(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	ctx := context.TODO()

	// test List
	job := types.NewJob(
		ctx, types.List, types.Region,
		api.GetRegionsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
		}, nil,
	)
	r, err := h.Do(job)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	listResources, ok := r.Payload.Data.(api.RegionsList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 0, len(*listResources.Regions))

	region1 := inv_testing.CreateRegion(t, nil)

	job = types.NewJob(
		ctx, types.List, types.Region,
		api.GetRegionsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
		}, nil,
	)
	r, err = h.Do(job)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.RegionsList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 1, len(*listResources.Regions))

	filter := fmt.Sprintf("%s = %q", location_v1.RegionResourceFieldResourceId, region1.GetResourceId())
	orderBy := location_v1.RegionResourceFieldResourceId
	job = types.NewJob(
		ctx, types.List, types.Region,
		api.GetRegionsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
			Filter:   &filter,
			OrderBy:  &orderBy,
		}, nil,
	)
	r, err = h.Do(job)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.RegionsList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 1, len(*listResources.Regions))

	// test List error - wrong params
	job = types.NewJob(
		ctx, types.List, types.Region,
		api.GetComputeHostsParams{}, nil,
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func checkTotalSites(t *testing.T, truthTable map[string]int, regions *[]api.Region) {
	t.Helper()
	for _, region := range *regions {
		truthSites, hasShowSites := truthTable[*region.ResourceId]
		assert.True(t, hasShowSites)
		assert.Equal(t, truthSites, *region.TotalSites)
	}
}

func Test_regionHandler_ListTotalSites(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	ctx := context.TODO()

	region0 := inv_testing.CreateRegion(t, nil)
	region1 := inv_testing.CreateRegion(t, region0)
	region2 := inv_testing.CreateRegion(t, region0)
	region3 := inv_testing.CreateRegion(t, region1)
	region4 := inv_testing.CreateRegion(t, region3)
	region5 := inv_testing.CreateRegion(t, region3)
	region6 := inv_testing.CreateRegion(t, nil)
	region7 := inv_testing.CreateRegion(t, nil)

	inv_testing.CreateSite(t, region2, nil)
	inv_testing.CreateSite(t, region4, nil)
	inv_testing.CreateSite(t, region5, nil)
	inv_testing.CreateSite(t, region6, nil)

	// Truth Table: regionID (parent of) -> total_sites (sites from...)
	// region0 (region1, region2) -> 3 sites (2-region1, 1-region2)
	// region1 (region3) -> 2 sites (2-region3)
	// region2 () -> 1 site (1-region2)
	// region3 (region4, region5) -> 2 sites (1-region4, 1-region5)
	// region4 () -> 1 site (region4)
	// region5 () -> 1 site (region5)
	// region6 () -> 1 site (region6)
	// region7 () -> 0 site ()
	truthTable := map[string]int{
		region0.GetResourceId(): 3,
		region1.GetResourceId(): 2,
		region2.GetResourceId(): 1,
		region3.GetResourceId(): 2,
		region4.GetResourceId(): 1,
		region5.GetResourceId(): 1,
		region6.GetResourceId(): 1,
		region7.GetResourceId(): 0,
	}

	// List All regions
	job := types.NewJob(
		ctx, types.List, types.Region,
		api.GetRegionsParams{
			Offset:         &pgOffset,
			PageSize:       &pgSize,
			ShowTotalSites: &showTotalSites,
		}, nil,
	)
	r, err := h.Do(job)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok := r.Payload.Data.(api.RegionsList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 8, len(*listResources.Regions))
	checkTotalSites(t, truthTable, listResources.Regions)

	// List root regions
	job = types.NewJob(
		ctx, types.List, types.Region,
		api.GetRegionsParams{
			Offset:         &pgOffset,
			PageSize:       &pgSize,
			ShowTotalSites: &showTotalSites,
			Filter:         &filterHasNoParent,
		}, nil,
	)
	r, err = h.Do(job)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.RegionsList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 3, len(*listResources.Regions))
	checkTotalSites(t, truthTable, listResources.Regions)
}

func Test_regionHandler_Post(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)
	metadata := api.Metadata{
		{
			Key:   "key",
			Value: "value",
		},
	}

	body := api.Region{
		Name:     &regionName,
		Metadata: &metadata,
	}
	ctx := context.TODO()
	job := types.NewJob(
		ctx, types.Post, types.Region,
		&body, inv_handlers.RegionURLParams{},
	)
	r, err := h.Do(job)
	assert.Equal(t, http.StatusCreated, r.Status)
	assert.NoError(t, err)

	gotRes, ok := r.Payload.Data.(*api.Region)
	assert.True(t, ok)

	// Validate Post changes
	job = types.NewJob(ctx, types.Get, types.Region, nil, inv_handlers.RegionURLParams{
		RegionID: *gotRes.RegionID,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok = r.Payload.Data.(*api.Region)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, regionName, *gotRes.Name)

	// Test Post error - wrong body request format
	job = types.NewJob(
		ctx, types.Post, types.Region,
		&api.Host{}, nil,
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_regionHandler_Put(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	regionResource := inv_testing.CreateRegion(t, nil)

	metadata := api.Metadata{
		{
			Key:   "key",
			Value: "value",
		},
	}
	bodyUpdate := api.Region{
		Name:     &regionName,
		Metadata: &metadata,
	}

	bodyEmpty := api.Region{}

	ctx := context.TODO()
	job := types.NewJob(
		ctx,
		types.Put,
		types.Region,
		&bodyUpdate,
		inv_handlers.RegionURLParams{RegionID: regionResource.ResourceId},
	)
	r, err := h.Do(job)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	// Validate Put changes
	job = types.NewJob(ctx, types.Get, types.Region, nil, inv_handlers.RegionURLParams{
		RegionID: regionResource.ResourceId,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.Region)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, regionName, *gotRes.Name)

	// Test Put error - wrong body
	job = types.NewJob(
		ctx,
		types.Put,
		types.Region,
		&api.Host{},
		inv_handlers.RegionURLParams{RegionID: regionResource.ResourceId},
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	// Test Put error - wrong params
	job = types.NewJob(
		ctx,
		types.Put,
		types.Region,
		&bodyUpdate,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	// Test Put error - wrong params
	job = types.NewJob(
		ctx,
		types.Put,
		types.Region,
		&bodyEmpty,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_regionHandler_Patch(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	regionResource := inv_testing.CreateRegion(t, nil)

	metadata := api.Metadata{
		{
			Key:   "key",
			Value: "value",
		},
	}
	bodyUpdate := api.Region{
		Name:     &regionName,
		Metadata: &metadata,
	}

	ctx := context.TODO()
	job := types.NewJob(
		ctx,
		types.Patch,
		types.Region,
		&bodyUpdate,
		inv_handlers.RegionURLParams{RegionID: regionResource.ResourceId},
	)
	r, err := h.Do(job)
	assert.Equal(t, http.StatusOK, r.Status)
	assert.NoError(t, err)

	// Validate Put changes
	job = types.NewJob(ctx, types.Get, types.Region, nil, inv_handlers.RegionURLParams{
		RegionID: regionResource.ResourceId,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.Region)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, regionName, *gotRes.Name)
}

func Test_regionHandler_PatchFieldMask(t *testing.T) {
	mockClient := utils.NewMockInventoryServiceClient(
		utils.MockResponses{
			ListResourcesResponse: &inventory.ListResourcesResponse{
				Resources: []*inventory.GetResourceResponse{},
			},
			GetResourceResponse:    &inventory.GetResourceResponse{},
			UpdateResourceResponse: &inventory.Resource{},
		},
	)
	client := &clients.InventoryClientHandler{
		InvClient: mockClient,
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	metadata := api.Metadata{
		{
			Key:   "key",
			Value: "value",
		},
	}
	bodyUpdate := api.Region{
		Name:     &regionName,
		Metadata: &metadata,
		ParentId: &emptyString,
	}

	ctx := context.TODO()
	job := types.NewJob(
		ctx,
		types.Patch,
		types.Region,
		&bodyUpdate,
		inv_handlers.RegionURLParams{RegionID: "region-1234"},
	)
	r, err := h.Do(job)
	assert.Equal(t, http.StatusOK, r.Status)
	assert.NoError(t, err)

	// test Patch FieldMask
	expectedPatchFieldMask := BuildFmFromRegionRequest(bodyUpdate)
	region := &location_v1.RegionResource{}
	expectedFieldMask, err := fieldmaskpb.New(region, expectedPatchFieldMask...)
	assert.NoError(t, err)

	if mockClient.LastUpdateResourceRequestFieldMask != nil {
		mockClient.LastUpdateResourceRequestFieldMask.Normalize()
		expectedFieldMask.Normalize()
		if !proto.Equal(expectedFieldMask, mockClient.LastUpdateResourceRequestFieldMask) {
			err = fmt.Errorf(
				"FieldMask is incorrectly constructed, expected: %s got: %s",
				expectedFieldMask.Paths,
				mockClient.LastUpdateResourceRequestFieldMask.Paths,
			)
		}
	} else {
		err = fmt.Errorf("no request in Mock Inventory")
	}
	assert.NoError(t, err)
}

func Test_regionHandler_Get(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}

	regionResource := inv_testing.CreateRegion(t, nil)

	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	ctx := context.TODO()
	job := types.NewJob(
		ctx,
		types.Get,
		types.Region,
		nil,
		inv_handlers.RegionURLParams{RegionID: regionResource.ResourceId},
	)
	r, err := h.Do(job)
	assert.Equal(t, http.StatusOK, r.Status)
	assert.NoError(t, err)

	// Get error - wrong params
	job = types.NewJob(
		ctx,
		types.Get,
		types.Region,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_regionHandler_Delete(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}

	regionResource := inv_testing.CreateRegionNoCleanup(t, nil)

	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	ctx := context.TODO()
	job := types.NewJob(
		ctx,
		types.Delete,
		types.Region,
		nil,
		inv_handlers.RegionURLParams{RegionID: regionResource.ResourceId},
	)
	r, err := h.Do(job)
	assert.Equal(t, http.StatusNoContent, r.Status)
	assert.NoError(t, err)

	// Validate delete
	job = types.NewJob(
		ctx,
		types.Get,
		types.Region,
		nil,
		inv_handlers.RegionURLParams{RegionID: regionResource.ResourceId},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	// Delete error - wrong params
	job = types.NewJob(
		ctx,
		types.Delete,
		types.Region,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_Inventory_Region_Integration(t *testing.T) {
	// verify the projection of the constants to Proto first;
	// we build a map using the field names of the proto stored in the
	// ProtoRegion* slices in internal/work/handlers/region.go. Elements must
	// have a mapping key otherwise we throw an error if there is no
	// alignment with Region proto in Inventory. Make sure to update these
	// two slices in internal/work/handlers/region.go
	regionResource := &location_v1.RegionResource{}
	validateInventoryIntegration(
		t,
		regionResource,
		api.Region{},
		inv_handlers.OpenAPIRegionToProto,
		inv_handlers.OpenAPIRegionToProtoExcluded,
		maps.Values(inv_handlers.OpenAPIRegionToProto),
		true,
	)
}

// Test_regionHandler_InvMockClient_Errors evaluates all
// Region handler methods with mock inventory client
// that returns errors.
func Test_regionHandler_InvMockClient_Errors(t *testing.T) {
	mockClient := utils.NewMockInventoryServiceClientError()
	client := &clients.InventoryClientHandler{
		InvClient: mockClient,
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	ctx := context.TODO()

	job := types.NewJob(
		ctx, types.List, types.Region,
		api.GetRegionsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
			Parent:   &utils.SiteUnexistID,
		}, nil,
	)
	_, err := h.Do(job)
	assert.Error(t, err)

	metadata := api.Metadata{
		{
			Key:   "key",
			Value: "value",
		},
	}
	body := api.Region{
		Name:     &regionName,
		Metadata: &metadata,
		ParentId: &utils.RegionUnexistID,
	}
	job = types.NewJob(
		ctx, types.Post, types.Region,
		&body, inv_handlers.RegionURLParams{},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	job = types.NewJob(
		ctx,
		types.Put,
		types.Region,
		&body,
		inv_handlers.RegionURLParams{RegionID: "region-12345678"},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	job = types.NewJob(
		ctx,
		types.Get,
		types.Region,
		nil,
		inv_handlers.RegionURLParams{RegionID: "region-1234"},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	job = types.NewJob(
		ctx,
		types.Delete,
		types.Region,
		nil,
		inv_handlers.RegionURLParams{RegionID: "region-1234"},
	)
	_, err = h.Do(job)
	assert.Error(t, err)
}
