// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/internal/worker/handlers"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	"github.com/open-edge-platform/infra-core/api/test/utils"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

var (
	showSites    = true
	showRegions  = true
	locationName = "name"
)

func Test_locationsHandler_Job_Error(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	job := types.NewJob(
		context.TODO(), types.Post, types.Locations,
		nil, nil,
	)
	_, err := h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))

	job = types.NewJob(
		context.TODO(), types.Get, types.Locations,
		nil, nil,
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))

	job = types.NewJob(
		context.TODO(), types.Put, types.Locations,
		nil, nil,
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))

	job = types.NewJob(
		context.TODO(), types.Patch, types.Locations,
		nil, nil,
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))

	job = types.NewJob(
		context.TODO(), types.Delete, types.Locations,
		nil, nil,
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))

	// Error no params provided
	job = types.NewJob(
		context.TODO(), types.List, types.Locations,
		nil, nil,
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	// Error no name provided
	job = types.NewJob(
		context.TODO(), types.List, types.Locations,
		api.GetLocationsParams{
			ShowRegions: &showRegions,
		}, nil,
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	// Error no name provided
	job = types.NewJob(
		context.TODO(), types.List, types.Locations,
		api.GetLocationsParams{
			ShowSites: &showSites,
		}, nil,
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	// Error no flag provided
	job = types.NewJob(
		context.TODO(), types.List, types.Locations,
		api.GetLocationsParams{
			Name: &locationName,
		}, nil,
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

// check that we pass the expected filters to the inventory.
func Test_locationsHandler_List(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	ctx := context.TODO()

	// test List
	job := types.NewJob(
		ctx, types.List, types.Locations,
		api.GetLocationsParams{
			ShowRegions: &showRegions,
			ShowSites:   &showSites,
			Name:        &locationName,
		}, nil,
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	listResources, ok := r.Payload.Data.(api.LocationNodeList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 0, len(*listResources.Nodes))

	region1 := inv_testing.CreateRegion(t, nil)
	assert.NotNil(t, region1)
	region2 := inv_testing.CreateRegion(t, region1)
	assert.NotNil(t, region2)

	job = types.NewJob(
		ctx, types.List, types.Locations,
		api.GetLocationsParams{
			ShowRegions: &showRegions,
			ShowSites:   &showSites,
			Name:        &region1.Name,
		}, nil,
	)
	r, err = h.Do(job)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.LocationNodeList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 2, len(*listResources.Nodes))

	job = types.NewJob(
		ctx, types.List, types.Locations,
		api.GetLocationsParams{
			ShowRegions: &showRegions,
			ShowSites:   &showSites,
			Name:        &region2.Name,
		}, nil,
	)
	r, err = h.Do(job)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.LocationNodeList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 2, len(*listResources.Nodes))

	// test List error - wrong params
	job = types.NewJob(
		ctx, types.List, types.Locations,
		api.GetComputeHostsParams{}, nil,
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

// Test_locationsHandler_InvMockClient_Errors evaluates all
// Region handler methods with mock inventory client
// that returns errors.
func Test_locationsHandler_InvMockClient_Errors(t *testing.T) {
	mockClient := utils.NewMockInventoryServiceClientError()
	client := &clients.InventoryClientHandler{
		InvClient: mockClient,
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	ctx := context.TODO()

	job := types.NewJob(
		ctx, types.List, types.Locations,
		api.GetRegionsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
			Parent:   &utils.SiteUnexistID,
		}, nil,
	)
	_, err := h.Do(job)
	assert.Error(t, err)
}
