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

var siteName = "Testsite"

func BuildFmFromSite(body api.Site) []string {
	fm := []string{}
	fm = append(fm, "name")
	if body.SiteLat != nil {
		fm = append(fm, "site_lat")
	}
	if body.SiteLng != nil {
		fm = append(fm, "site_lng")
	}
	if body.DnsServers != nil {
		fm = append(fm, "dns_servers")
	}
	if body.DockerRegistries != nil {
		fm = append(fm, "docker_registries")
	}
	if body.MetricsEndpoint != nil {
		fm = append(fm, "metrics_endpoint")
	}
	if body.Metadata != nil {
		fm = append(fm, "metadata")
	}
	if body.RegionId != nil {
		fm = append(fm, "region")
	}
	if body.OuId != nil {
		fm = append(fm, "ou")
	}
	return fm
}

func Test_siteHandler_Job_Error(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}

	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	ctx := context.TODO()
	job := types.NewJob(ctx, BadOperation, types.Site, nil, inv_handlers.SiteURLParams{})
	_, err := h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))
}

//nolint:funlen // it is a test
func Test_siteHandler_List(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	ctx := context.TODO()
	job := types.NewJob(ctx, types.List, types.Site, nil, api.GetSitesParams{})
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	// Verify that nothing is returned without any existing sites.
	job = types.NewJob(ctx, types.List, types.Site, api.GetSitesParams{
		Offset:   &pgOffset,
		PageSize: &pgSize,
	}, inv_handlers.SiteURLParams{})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok := r.Payload.Data.(api.SitesList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Len(t, *listResources.Sites, 0)

	region1 := inv_testing.CreateRegion(t, nil)
	site1 := inv_testing.CreateSite(t, region1, nil)

	// Verify that the new site is returned on LIST all.
	job = types.NewJob(ctx, types.List, types.Site, api.GetSitesParams{
		Offset:   &pgOffset,
		PageSize: &pgSize,
	}, inv_handlers.SiteURLParams{})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.SitesList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Len(t, *listResources.Sites, 1)
	assert.Equal(t, site1.GetResourceId(), *(*listResources.Sites)[0].ResourceId)

	// Verify that the new site is returned on Get for SiteID.
	job = types.NewJob(ctx, types.List, types.Site, api.GetSitesParams{
		Offset:   &pgOffset,
		PageSize: &pgSize,
	}, inv_handlers.SiteURLParams{SiteID: site1.GetResourceId()})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.SitesList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Len(t, *listResources.Sites, 1)
	assert.Equal(t, site1.GetResourceId(), *(*listResources.Sites)[0].ResourceId)

	// Verify that the new site is returned on LIST with legacy regionID filter parameter.
	regionID := region1.GetResourceId()
	job = types.NewJob(ctx, types.List, types.Site, api.GetSitesParams{
		Offset:   &pgOffset,
		PageSize: &pgSize,
		RegionID: &regionID,
	}, inv_handlers.SiteURLParams{})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.SitesList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Len(t, *listResources.Sites, 1)
	assert.Equal(t, site1.GetResourceId(), *(*listResources.Sites)[0].ResourceId)

	// Verify that the new site is returned on LIST with filter.
	filter := fmt.Sprintf("%s = %q", location_v1.SiteResourceFieldResourceId, site1.GetResourceId())
	orderBy := location_v1.SiteResourceFieldResourceId
	job = types.NewJob(ctx, types.List, types.Site, api.GetSitesParams{
		Offset:   &pgOffset,
		PageSize: &pgSize,
		Filter:   &filter,
		OrderBy:  &orderBy,
	}, inv_handlers.SiteURLParams{})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.SitesList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Len(t, *listResources.Sites, 1)
	assert.Equal(t, site1.GetResourceId(), *(*listResources.Sites)[0].ResourceId)

	// Verify that wrong offset and page size args are rejected.
	job = types.NewJob(ctx, types.List, types.Site, api.GetSitesParams{
		Offset:   &pgIndexWrong,
		PageSize: &pgSizeWrong,
	}, inv_handlers.SiteURLParams{})
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
	job = types.NewJob(ctx, types.List, types.Site, api.GetComputeHostsParams{}, inv_handlers.SiteURLParams{})
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_siteHandler_Post(t *testing.T) {
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
	proxyStr := "proxy"
	body := api.Site{
		Name:     &siteName,
		Metadata: &metadata,
		Proxy: &api.Proxy{
			FtpProxy:   &proxyStr,
			HttpProxy:  &proxyStr,
			HttpsProxy: &proxyStr,
			NoProxy:    &proxyStr,
		},
	}

	ctx := context.TODO()
	job := types.NewJob(
		ctx,
		types.Post,
		types.Site,
		&body,
		inv_handlers.SiteURLParams{},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, r.Status)
	gotRes, ok := r.Payload.Data.(*api.Site)
	assert.True(t, ok)

	// Validate Post changes
	job = types.NewJob(ctx, types.Get, types.Site, nil, inv_handlers.SiteURLParams{
		SiteID: *gotRes.SiteID,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok = r.Payload.Data.(*api.Site)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, siteName, *gotRes.Name)

	// Post error - wrong body format
	job = types.NewJob(
		ctx,
		types.Post,
		types.Site,
		&api.Host{},
		inv_handlers.SiteURLParams{SiteID: "site-12345678"},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_siteHandler_Get(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	siteResource := inv_testing.CreateSite(t, nil, nil)
	ctx := context.TODO()
	job := types.NewJob(ctx, types.Get, types.Site, nil, inv_handlers.SiteURLParams{
		SiteID: siteResource.ResourceId,
	})
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.Site)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, siteResource.Name, *gotRes.Name)

	// verify provider
	provider := inv_testing.CreateProvider(t, "TEST")
	siteResourceWithProvider := inv_testing.CreateSiteWithArgs(t, "TEST", 0, 0, "", nil, nil, provider)
	job = types.NewJob(ctx, types.Get, types.Site, nil, inv_handlers.SiteURLParams{
		SiteID: siteResourceWithProvider.ResourceId,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok = r.Payload.Data.(*api.Site)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, siteResourceWithProvider.Name, *gotRes.Name)
	VerifyProvider(t, gotRes.Provider, provider)

	// Get error - wrong params
	job = types.NewJob(ctx, types.Get, types.Site, nil, inv_handlers.HostURLParams{})
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_siteHandler_Put(t *testing.T) {
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
	body := api.Site{
		Name:     &siteName,
		Metadata: &metadata,
		OuId:     &emptyString,
		RegionId: &emptyString,
	}

	siteResource := inv_testing.CreateSite(t, nil, nil)

	ctx := context.TODO()
	job := types.NewJob(ctx, types.Put, types.Site, &body, inv_handlers.SiteURLParams{
		SiteID: siteResource.ResourceId,
	})
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	// Validate Put changes
	job = types.NewJob(ctx, types.Get, types.Site, nil, inv_handlers.SiteURLParams{
		SiteID: siteResource.ResourceId,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.Site)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, siteName, *gotRes.Name)

	job = types.NewJob(context.TODO(), types.Put, types.Site, nil, inv_handlers.SiteURLParams{
		SiteID: siteResource.ResourceId,
	})
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	job = types.NewJob(context.TODO(), types.Put, types.Site, &body, inv_handlers.HostURLParams{})
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_siteHandler_Patch(t *testing.T) {
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
	body := api.Site{
		Name:     &siteName,
		Metadata: &metadata,
		OuId:     &emptyString,
		RegionId: &emptyString,
	}
	siteResource := inv_testing.CreateSite(t, nil, nil)

	ctx := context.TODO()
	job := types.NewJob(ctx, types.Patch, types.Site, &body, inv_handlers.SiteURLParams{SiteID: siteResource.ResourceId})
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	// Validate Patch changes
	job = types.NewJob(ctx, types.Get, types.Site, nil, inv_handlers.SiteURLParams{
		SiteID: siteResource.ResourceId,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.Site)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, siteName, *gotRes.Name)

	job = types.NewJob(context.TODO(), types.Patch, types.Site, nil, inv_handlers.SiteURLParams{SiteID: siteResource.ResourceId})
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_siteHandler_PatchFieldMask(t *testing.T) {
	mockClient := utils.NewMockInventoryServiceClient(
		utils.MockResponses{
			ListResourcesResponse: &inventory.ListResourcesResponse{
				Resources: []*inventory.GetResourceResponse{},
			},
			GetResourceResponse: &inventory.GetResourceResponse{},
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
	body := api.Site{
		Name:     &siteName,
		Metadata: &metadata,
		OuId:     &emptyString,
		RegionId: &emptyString,
	}

	ctx := context.TODO()
	job := types.NewJob(ctx, types.Patch, types.Site, &body, inv_handlers.SiteURLParams{SiteID: "site-12345678"})
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	// test Patch FieldMask
	expectedPatchFieldMask := BuildFmFromSite(body)
	site := &location_v1.SiteResource{}
	expectedFieldMask, err := fieldmaskpb.New(site, expectedPatchFieldMask...)
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

func Test_siteHandler_Delete(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	siteResource := inv_testing.CreateSiteNoCleanup(t, nil, nil)

	ctx := context.TODO()
	job := types.NewJob(ctx, types.Delete, types.Site, nil, inv_handlers.SiteURLParams{SiteID: siteResource.ResourceId})
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.Status)

	// Validate Delete changes
	job = types.NewJob(ctx, types.Get, types.Site, nil, inv_handlers.SiteURLParams{
		SiteID: siteResource.ResourceId,
	})
	_, err = h.Do(job)
	assert.Error(t, err)

	// Delete error - wrong params
	job = types.NewJob(ctx, types.Delete, types.Site, nil, inv_handlers.HostURLParams{})
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_Inventory_Site_Integration(t *testing.T) {
	// verify the projection of the constants to Proto first;
	// we build a map using the field names of the proto stored in the
	// ProtoSite* slices in internal/work/handlers/site.go. Elements must
	// have a mapping key otherwise we throw an error if there is no
	// alignment with Site proto in Inventory. Make sure to update these
	// two slices in internal/work/handlers/site.go
	siteResource := &location_v1.SiteResource{}
	validateInventoryIntegration(t, siteResource, api.Site{}, inv_handlers.OpenAPISiteToProto,
		inv_handlers.OpenAPISiteToProtoExcluded, maps.Values(inv_handlers.OpenAPISiteToProto), false)
	validateInventoryIntegration(
		t,
		siteResource,
		api.Proxy{},
		inv_handlers.OpenAPISiteToProto,
		map[string]struct{}{},
		inv_handlers.ProtoSiteProxyFields,
		true,
	)
}

// Test_siteHandler_InvMockClient_Errors evaluates all
// Site handler methods with mock inventory client
// that returns errors.
func Test_siteHandler_InvMockClient_Errors(t *testing.T) {
	mockClient := utils.NewMockInventoryServiceClientError()
	client := &clients.InventoryClientHandler{
		InvClient: mockClient,
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	ctx := context.TODO()

	// List response error
	listParams := api.GetSitesParams{
		Offset:   &pgOffset,
		PageSize: &pgSize,
		RegionID: &defaultRegionID,
		OuID:     &defaultOUID,
	}
	job := types.NewJob(ctx, types.List, types.Site, listParams, inv_handlers.SiteURLParams{SiteID: "site-12345678"})
	_, err := h.Do(job)
	assert.Error(t, err)

	// Post response error
	metadata := api.Metadata{
		{
			Key:   "key",
			Value: "value",
		},
	}
	proxyStr := "proxy"
	body := api.Site{
		Name:     &siteName,
		Metadata: &metadata,
		Proxy: &api.Proxy{
			FtpProxy:   &proxyStr,
			HttpProxy:  &proxyStr,
			HttpsProxy: &proxyStr,
			NoProxy:    &proxyStr,
		},
	}

	job = types.NewJob(
		ctx,
		types.Post,
		types.Site,
		&body,
		inv_handlers.SiteURLParams{SiteID: "site-12345678"},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	// Put response error
	job = types.NewJob(
		ctx,
		types.Put,
		types.Site,
		&body,
		inv_handlers.SiteURLParams{SiteID: "site-12345678"},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	// Get response error
	job = types.NewJob(ctx, types.Get, types.Site, nil, inv_handlers.SiteURLParams{SiteID: "site-12345678"})
	_, err = h.Do(job)
	assert.Error(t, err)

	// Delete response error
	job = types.NewJob(ctx, types.Delete, types.Site, nil, inv_handlers.SiteURLParams{SiteID: "site-12345678"})
	_, err = h.Do(job)
	assert.Error(t, err)
}
