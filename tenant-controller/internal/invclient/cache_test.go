// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invclient

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inventoryv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	providerv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/provider/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/client/cache"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/filters"
)

func TestInventoryClientCache(t *testing.T) {
	allProvidersFilter := &inventoryv1.ResourceFilter{
		Resource: &inventoryv1.Resource{Resource: &inventoryv1.Resource_Provider{}},
		Filter:   "",
	}

	allTenantProvidersFilter := &inventoryv1.ResourceFilter{
		Resource: &inventoryv1.Resource{Resource: &inventoryv1.Resource_Provider{}},
		Filter:   filters.NewBuilderWith(filters.ValEq("tenant_id", "foobar")).Build(),
	}

	providers := []*inventoryv1.Resource{{
		Resource: &inventoryv1.Resource_Provider{Provider: &providerv1.ProviderResource{}},
	}}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	icMock := new(inventoryClientMock)
	cachingClient := NewInventoryClientCache(icMock)
	sut, ok := cachingClient.(*inventoryClientNaiveCache)
	require.True(t, ok)

	require.Empty(t, sut.cacheByOperation, "just after initialization cache shall be empty")

	icMock.On("ListAll", ctx, allProvidersFilter).Return(providers, nil)
	rsp, err := cachingClient.ListAll(ctx, allProvidersFilter)

	require.NoError(t, err, "unexpected error during ListAll call")
	require.Len(t, rsp, 1)

	require.NotEmpty(t, sut.cacheByOperation)

	listAllCache, ok := sut.cacheByOperation["ListAll"]
	require.True(t, ok, "cache shall contain `ListAll` key")
	require.NotNil(t, listAllCache)
	require.NotEmpty(t, listAllCache)
	require.Len(t, listAllCache, 1)
	require.Contains(t, maps.Values(listAllCache), providers)

	icMock.On("ListAll", ctx, allProvidersFilter).
		Run(
			func(_ mock.Arguments) {
				require.Fail(t, "requested result shall be returned from the cache")
			}).Return(nil, nil)

	rsp, err = cachingClient.ListAll(ctx, allProvidersFilter)
	require.NoError(t, err, "unexpected error during ListAll call")
	require.Len(t, rsp, 1)

	icMock.On("ListAll", ctx, allTenantProvidersFilter).Return(providers, nil)
	_, err = cachingClient.ListAll(ctx, allTenantProvidersFilter)
	require.NoError(t, err)
	require.True(t, ok, "cache shall contain `ListAll` key")
	require.NotNil(t, listAllCache)
	require.NotEmpty(t, listAllCache)
	require.Len(t, listAllCache, 2)
	require.ElementsMatch(t, maps.Values(listAllCache), [][]*inventoryv1.Resource{providers, providers})
}

type inventoryClientMock struct {
	mock.Mock
}

func (i *inventoryClientMock) Close() error {
	args := i.Called()
	return args.Error(0)
}

func (i *inventoryClientMock) List(
	ctx context.Context, filter *inventoryv1.ResourceFilter,
) (*inventoryv1.ListResourcesResponse, error) {
	args := i.Called(ctx, filter)
	resp, ok := args.Get(0).(*inventoryv1.ListResourcesResponse)
	if !ok {
		return nil, errors.Errorf("unexpected type for ListResourcesResponse: %T", args.Get(0))
	}
	return resp, args.Error(1)
}

func (i *inventoryClientMock) ListAll(ctx context.Context, filter *inventoryv1.ResourceFilter) ([]*inventoryv1.Resource, error) {
	args := i.Called(ctx, filter)
	resource, ok := args.Get(0).([]*inventoryv1.Resource)
	if !ok {
		return nil, errors.Errorf("unexpected type for Resources: %T", args.Get(0))
	}
	return resource, args.Error(1)
}

func (i *inventoryClientMock) Find(
	ctx context.Context, filter *inventoryv1.ResourceFilter,
) (*inventoryv1.FindResourcesResponse, error) {
	args := i.Called(ctx, filter)
	resp, ok := args.Get(0).(*inventoryv1.FindResourcesResponse)
	if !ok {
		return nil, errors.Errorf("unexpected type for Resources: %T", args.Get(0))
	}
	return resp, args.Error(1)
}

func (i *inventoryClientMock) FindAll(
	ctx context.Context, filter *inventoryv1.ResourceFilter,
) ([]*client.ResourceTenantIDCarrier, error) {
	args := i.Called(ctx, filter)
	tenantIDCarrier, ok := args.Get(0).([]*client.ResourceTenantIDCarrier)
	if !ok {
		return nil, errors.Errorf("unexpected type for TenantIDCarrier: %T", args.Get(0))
	}
	return tenantIDCarrier, args.Error(1)
}

func (i *inventoryClientMock) Get(ctx context.Context, tenantID, id string) (*inventoryv1.GetResourceResponse, error) {
	args := i.Called(ctx, tenantID, id)
	resp, ok := args.Get(0).(*inventoryv1.GetResourceResponse)
	if !ok {
		return nil, errors.Errorf("unexpected type for Resource: %T", args.Get(0))
	}
	return resp, args.Error(1)
}

func (i *inventoryClientMock) Create(
	ctx context.Context, tenantID string, res *inventoryv1.Resource,
) (*inventoryv1.Resource, error) {
	args := i.Called(ctx, tenantID, res)
	resource, ok := args.Get(0).(*inventoryv1.Resource)
	if !ok {
		return nil, errors.Errorf("unexpected type for Resource: %T", args.Get(0))
	}
	return resource, args.Error(1)
}

func (i *inventoryClientMock) Update(
	ctx context.Context, tenantID, id string, fm *fieldmaskpb.FieldMask, res *inventoryv1.Resource,
) (*inventoryv1.Resource, error) {
	args := i.Called(ctx, tenantID, id, fm, res)
	resource, ok := args.Get(0).(*inventoryv1.Resource)
	if !ok {
		return nil, errors.Errorf("unexpected type for Resource: %T", args.Get(0))
	}
	return resource, args.Error(1)
}

func (i *inventoryClientMock) Delete(ctx context.Context, tenantID, id string) (*inventoryv1.DeleteResourceResponse, error) {
	args := i.Called(ctx, tenantID, id)
	resp, ok := args.Get(0).(*inventoryv1.DeleteResourceResponse)
	if !ok {
		return nil, errors.Errorf("unexpected type for DeleteResourceResponse: %T", args.Get(0))
	}
	return resp, args.Error(1)
}

func (i *inventoryClientMock) UpdateSubscriptions(ctx context.Context, tenantID string, kinds []inventoryv1.ResourceKind) error {
	args := i.Called(ctx, tenantID, kinds)
	return args.Error(0)
}

func (i *inventoryClientMock) ListInheritedTelemetryProfiles(
	ctx context.Context, tenantID string, inheritBy *inventoryv1.ListInheritedTelemetryProfilesRequest_InheritBy,
	filter string, orderBy string, limit, offset uint32,
) (*inventoryv1.ListInheritedTelemetryProfilesResponse, error) {
	args := i.Called(ctx, tenantID, inheritBy, filter, orderBy, limit, offset)
	resp, ok := args.Get(0).(*inventoryv1.ListInheritedTelemetryProfilesResponse)
	if !ok {
		return nil, errors.Errorf("unexpected type for ListInheritedTelemetryProfilesResponse: %T", args.Get(0))
	}
	return resp, args.Error(1)
}

func (i *inventoryClientMock) GetHostByUUID(ctx context.Context, tenantID, uuid string) (*computev1.HostResource, error) {
	args := i.Called(ctx, tenantID, uuid)
	host, ok := args.Get(0).(*computev1.HostResource)
	if !ok {
		return nil, errors.Errorf("unexpected type for HostResource: %T", args.Get(0))
	}
	return host, args.Error(1)
}

func (i *inventoryClientMock) GetTreeHierarchy(
	ctx context.Context, request *inventoryv1.GetTreeHierarchyRequest,
) ([]*inventoryv1.GetTreeHierarchyResponse_TreeNode, error) {
	args := i.Called(ctx, request)
	resp, ok := args.Get(0).([]*inventoryv1.GetTreeHierarchyResponse_TreeNode)
	if !ok {
		return nil, errors.Errorf("unexpected type for GetTreeHierarchyResponse_TreeNode: %T", args.Get(0))
	}
	return resp, args.Error(1)
}

func (i *inventoryClientMock) GetSitesPerRegion(
	ctx context.Context, request *inventoryv1.GetSitesPerRegionRequest,
) (*inventoryv1.GetSitesPerRegionResponse, error) {
	args := i.Called(ctx, request)
	resp, ok := args.Get(0).(*inventoryv1.GetSitesPerRegionResponse)
	if !ok {
		return nil, errors.Errorf("unexpected type for GetSitesPerRegionResponse: %T", args.Get(0))
	}
	return resp, args.Error(1)
}

func (i *inventoryClientMock) TestingOnlySetClient(isc inventoryv1.InventoryServiceClient) {
	_ = i.Called(isc)
}

func (i *inventoryClientMock) TestGetClientCache() *cache.InventoryCache {
	args := i.Called()
	cacheInv, ok := args.Get(0).(*cache.InventoryCache)
	if !ok {
		log.Error().Msgf("unexpected type for InventoryCache: %T", args.Get(0))
		return nil
	}
	return cacheInv
}

func (i *inventoryClientMock) TestGetClientCacheUUID() *cache.InventoryCache {
	args := i.Called()
	cacheInv, ok := args.Get(0).(*cache.InventoryCache)
	if !ok {
		log.Error().Msgf("unexpected type for InventoryCache: %T", args.Get(0))
		return nil
	}
	return cacheInv
}

func (i *inventoryClientMock) DeleteAllResources(
	ctx context.Context, tenantID string, kind inventoryv1.ResourceKind, enforce bool,
) error {
	args := i.Called(ctx, tenantID, kind, enforce)
	return args.Error(0)
}
