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
	telemetryv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/telemetry/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

func Test_TelemetryLogsProfileHandler_Job_Error(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}

	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	ctx := context.TODO()
	job := types.NewJob(ctx, BadOperation, types.TelemetryLogsProfile, nil, inv_handlers.TelemetryLogsProfileURLParams{})
	_, err := h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))
}

//nolint:funlen // it is a test
func Test_TelemetryLogsProfileHandler_List(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	// test List with target instance
	job := types.NewJob(
		context.TODO(), types.List, types.TelemetryLogsProfile,
		api.GetTelemetryProfilesLogsParams{
			Offset:     &pgOffset,
			PageSize:   &pgSize,
			InstanceId: &utils.InstanceUnexistID,
		}, nil,
	)
	r, err := h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok := r.Payload.Data.(api.TelemetryLogsProfileList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 0, len(*listResources.TelemetryLogsProfiles))

	// test List with target site
	job = types.NewJob(
		context.TODO(), types.List, types.TelemetryLogsProfile,
		api.GetTelemetryProfilesLogsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
			SiteId:   &utils.SiteUnexistID,
		}, nil,
	)
	r, err = h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.TelemetryLogsProfileList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 0, len(*listResources.TelemetryLogsProfiles))

	// test List with target region
	job = types.NewJob(
		context.TODO(), types.List, types.TelemetryLogsProfile,
		api.GetTelemetryProfilesLogsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
			RegionId: &utils.RegionUnexistID,
		}, nil,
	)
	r, err = h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.TelemetryLogsProfileList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 0, len(*listResources.TelemetryLogsProfiles))

	telGroupRes := inv_testing.CreateTelemetryGroupLogs(t, true)

	osRes := inv_testing.CreateOs(t)
	hostRes := inv_testing.CreateHost(t, nil, nil)
	instRes := inv_testing.CreateInstance(t, hostRes, osRes)
	inv_testing.CreateTelemetryProfile(t, instRes, nil, nil, telGroupRes, true)

	siteRes := inv_testing.CreateSite(t, nil, nil)
	inv_testing.CreateTelemetryProfile(t, nil, siteRes, nil, telGroupRes, true)

	regionRes := inv_testing.CreateRegion(t, nil)
	inv_testing.CreateTelemetryProfile(t, nil, nil, regionRes, telGroupRes, true)

	// test List with target instance
	job = types.NewJob(
		context.TODO(), types.List, types.TelemetryLogsProfile,
		api.GetTelemetryProfilesLogsParams{
			Offset:     &pgOffset,
			PageSize:   &pgSize,
			InstanceId: &instRes.ResourceId,
		}, nil,
	)
	r, err = h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.TelemetryLogsProfileList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 1, len(*listResources.TelemetryLogsProfiles))

	// test List with target site
	job = types.NewJob(
		context.TODO(), types.List, types.TelemetryLogsProfile,
		api.GetTelemetryProfilesLogsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
			SiteId:   &siteRes.ResourceId,
		}, nil,
	)
	r, err = h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.TelemetryLogsProfileList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 1, len(*listResources.TelemetryLogsProfiles))

	// test List with target region
	job = types.NewJob(
		context.TODO(), types.List, types.TelemetryLogsProfile,
		api.GetTelemetryProfilesLogsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
			RegionId: &regionRes.ResourceId,
		}, nil,
	)
	r, err = h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.TelemetryLogsProfileList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 1, len(*listResources.TelemetryLogsProfiles))

	job = types.NewJob(
		context.TODO(), types.List, types.TelemetryLogsProfile,
		api.GetTelemetryProfilesLogsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
		}, nil,
	)
	r, err = h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.TelemetryLogsProfileList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 3, len(*listResources.TelemetryLogsProfiles))

	// test List error - wrong params
	job = types.NewJob(
		context.TODO(), types.List, types.TelemetryLogsProfile,
		api.GetComputeHostsParams{}, nil,
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	// test List error - all Instance ID, Site ID and Region ID set
	job = types.NewJob(
		context.TODO(), types.List, types.TelemetryLogsProfile,
		api.GetTelemetryProfilesLogsParams{
			Offset:     &pgOffset,
			PageSize:   &pgSize,
			InstanceId: &utils.InstanceUnexistID,
			SiteId:     &utils.SiteUnexistID,
			RegionId:   &utils.RegionUnexistID,
		}, nil,
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	// test List error - both Site ID and Region ID set
	job = types.NewJob(
		context.TODO(), types.List, types.TelemetryLogsProfile,
		api.GetTelemetryProfilesLogsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
			SiteId:   &utils.SiteUnexistID,
			RegionId: &utils.RegionUnexistID,
		}, nil,
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	// test List error - both Instance ID and Region ID set
	job = types.NewJob(
		context.TODO(), types.List, types.TelemetryLogsProfile,
		api.GetTelemetryProfilesLogsParams{
			Offset:     &pgOffset,
			PageSize:   &pgSize,
			InstanceId: &utils.InstanceUnexistID,
			RegionId:   &utils.RegionUnexistID,
		}, nil,
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	// test List error - both Instance ID and Site ID set
	job = types.NewJob(
		context.TODO(), types.List, types.TelemetryLogsProfile,
		api.GetTelemetryProfilesLogsParams{
			Offset:     &pgOffset,
			PageSize:   &pgSize,
			InstanceId: &utils.InstanceUnexistID,
			SiteId:     &utils.SiteUnexistID,
		}, nil,
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_TelemetryLogsProfileHandler_Post(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	telGroupRes := inv_testing.CreateTelemetryGroupLogs(t, true)

	osRes := inv_testing.CreateOs(t)
	hostRes := inv_testing.CreateHost(t, nil, nil)
	instRes := inv_testing.CreateInstance(t, hostRes, osRes)
	siteRes := inv_testing.CreateSite(t, nil, nil)
	regionRes := inv_testing.CreateRegion(t, nil)

	body := api.TelemetryLogsProfile{
		LogsGroupId:    telGroupRes.ResourceId,
		LogLevel:       api.TELEMETRYSEVERITYLEVELWARN,
		TargetInstance: &instRes.ResourceId,
	}
	job := types.NewJob(
		context.TODO(), types.Post, types.TelemetryLogsProfile,
		&body, inv_handlers.TelemetryLogsProfileURLParams{},
	)
	r, err := h.Do(job)
	require.Equal(t, nil, err)
	assert.Equal(t, http.StatusCreated, r.Status)
	gotRes1, ok := r.Payload.Data.(*api.TelemetryLogsProfile)
	assert.True(t, ok)

	body = api.TelemetryLogsProfile{
		LogsGroupId: telGroupRes.ResourceId,
		LogLevel:    api.TELEMETRYSEVERITYLEVELWARN,
		TargetSite:  &siteRes.ResourceId,
	}
	job = types.NewJob(
		context.TODO(), types.Post, types.TelemetryLogsProfile,
		&body, inv_handlers.TelemetryLogsProfileURLParams{},
	)
	r, err = h.Do(job)
	require.Equal(t, nil, err)
	assert.Equal(t, http.StatusCreated, r.Status)
	gotRes2, ok := r.Payload.Data.(*api.TelemetryLogsProfile)
	assert.True(t, ok)

	body = api.TelemetryLogsProfile{
		LogsGroupId:  telGroupRes.ResourceId,
		LogLevel:     api.TELEMETRYSEVERITYLEVELWARN,
		TargetRegion: &regionRes.ResourceId,
	}
	job = types.NewJob(
		context.TODO(), types.Post, types.TelemetryLogsProfile,
		&body, inv_handlers.TelemetryLogsProfileURLParams{},
	)
	r, err = h.Do(job)
	require.Equal(t, nil, err)
	assert.Equal(t, http.StatusCreated, r.Status)
	gotRes3, ok := r.Payload.Data.(*api.TelemetryLogsProfile)
	assert.True(t, ok)

	// Post error - wrong body request format
	job = types.NewJob(
		context.TODO(), types.Post, types.TelemetryLogsProfile,
		&api.Host{}, inv_handlers.TelemetryLogsProfileURLParams{},
	)
	_, err = h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	inv_testing.DeleteResource(t, *gotRes1.ProfileId)
	inv_testing.DeleteResource(t, *gotRes2.ProfileId)
	inv_testing.DeleteResource(t, *gotRes3.ProfileId)
}

func Test_TelemetryLogsProfileHandler_Put(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	targetSite := "site-12345678"
	bodyUpdate := api.TelemetryLogsProfile{
		LogLevel:   api.TELEMETRYSEVERITYLEVELWARN,
		TargetSite: &targetSite,
	}

	job := types.NewJob(
		context.TODO(),
		types.Put,
		types.TelemetryLogsProfile,
		&bodyUpdate,
		inv_handlers.TelemetryLogsProfileURLParams{TelemetryLogsProfileID: "telemetryprofile-12345678"},
	)
	_, err := h.Do(job)
	require.Error(t, err)

	telGroupRes := inv_testing.CreateTelemetryGroupLogs(t, true)
	siteRes := inv_testing.CreateSite(t, nil, nil)
	regRes := inv_testing.CreateRegion(t, nil)
	profileRes := inv_testing.CreateTelemetryProfile(t, nil, siteRes, nil, telGroupRes, true)

	bodyUpdate = api.TelemetryLogsProfile{
		LogLevel:     api.TELEMETRYSEVERITYLEVELWARN,
		TargetRegion: &regRes.ResourceId,
		LogsGroupId:  telGroupRes.ResourceId,
	}

	job = types.NewJob(
		context.TODO(),
		types.Put,
		types.TelemetryLogsProfile,
		&bodyUpdate,
		inv_handlers.TelemetryLogsProfileURLParams{TelemetryLogsProfileID: profileRes.ResourceId},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	// Test Put  error - wrong body
	job = types.NewJob(
		context.TODO(),
		types.Put,
		types.TelemetryLogsProfile,
		&api.Host{},
		inv_handlers.TelemetryLogsProfileURLParams{TelemetryLogsProfileID: "telemetryprofile-12345678"},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	// Test Put  error - wrong params
	job = types.NewJob(
		context.TODO(),
		types.Put,
		types.TelemetryLogsProfile,
		&bodyUpdate,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_TelemetryLogsProfileHandler_Patch(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)
	targetSite := "site-12345678"
	bodyUpdate := api.TelemetryLogsProfile{
		LogLevel:   api.TELEMETRYSEVERITYLEVELWARN,
		TargetSite: &targetSite,
	}

	job := types.NewJob(
		context.TODO(),
		types.Patch,
		types.TelemetryLogsProfile,
		&bodyUpdate,
		inv_handlers.TelemetryLogsProfileURLParams{TelemetryLogsProfileID: "telemetryprofile-12345678"},
	)
	_, err := h.Do(job)
	require.Error(t, err)

	telGroupRes := inv_testing.CreateTelemetryGroupLogs(t, true)
	siteRes := inv_testing.CreateSite(t, nil, nil)
	regRes := inv_testing.CreateRegion(t, nil)
	profileRes := inv_testing.CreateTelemetryProfile(t, nil, siteRes, nil, telGroupRes, true)

	bodyUpdate = api.TelemetryLogsProfile{
		LogLevel:     api.TELEMETRYSEVERITYLEVELDEBUG,
		TargetRegion: &regRes.ResourceId,
		TargetSite:   &utils.EmptyString,
		LogsGroupId:  telGroupRes.ResourceId,
	}

	job = types.NewJob(
		context.TODO(),
		types.Patch,
		types.TelemetryLogsProfile,
		&bodyUpdate,
		inv_handlers.TelemetryLogsProfileURLParams{TelemetryLogsProfileID: profileRes.ResourceId},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)
}

func Test_TelemetryLogsProfileHandler_PatchFieldMask(t *testing.T) {
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

	targetInstance := "inst-12345678"
	bodyUpdate := api.TelemetryLogsProfile{
		LogLevel:       api.TELEMETRYSEVERITYLEVELWARN,
		TargetInstance: &targetInstance,
		LogsGroupId:    "telemetrygroup-12345678",
	}

	job := types.NewJob(
		context.TODO(),
		types.Patch,
		types.TelemetryLogsProfile,
		&bodyUpdate,
		inv_handlers.TelemetryLogsProfileURLParams{TelemetryLogsProfileID: "telemetryprofile-12345678"},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	// test Patch FieldMask
	expectedPatchFieldMask := []string{
		telemetryv1.TelemetryProfileFieldLogLevel,
		telemetryv1.TelemetryProfileEdgeInstance,
		telemetryv1.TelemetryProfileEdgeGroup,
	}
	expectedFieldMask, err := fieldmaskpb.New(&telemetryv1.TelemetryProfile{}, expectedPatchFieldMask...)
	require.NoError(t, err)

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
	require.NoError(t, err)
}

func Test_TelemetryLogsProfileHandler_Get(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	job := types.NewJob(
		context.TODO(),
		types.Get,
		types.TelemetryLogsProfile,
		nil,
		inv_handlers.TelemetryLogsProfileURLParams{TelemetryLogsProfileID: "telemetryprofile-12345678"},
	)
	_, err := h.Do(job)
	assert.Error(t, err)

	telGroupRes := inv_testing.CreateTelemetryGroupLogs(t, true)
	siteRes := inv_testing.CreateSite(t, nil, nil)
	profileRes := inv_testing.CreateTelemetryProfile(t, nil, siteRes, nil, telGroupRes, true)

	job = types.NewJob(
		context.TODO(),
		types.Get,
		types.TelemetryLogsProfile,
		nil,
		inv_handlers.TelemetryLogsProfileURLParams{TelemetryLogsProfileID: profileRes.ResourceId},
	)
	r, err := h.Do(job)
	require.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, r.Status)
	gotRes, ok := r.Payload.Data.(*api.TelemetryLogsProfile)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, profileRes.ResourceId, *gotRes.ProfileId)
}

func Test_TelemetryLogsProfileHandler_Delete(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	job := types.NewJob(
		context.TODO(),
		types.Delete,
		types.TelemetryLogsProfile,
		nil,
		inv_handlers.TelemetryLogsProfileURLParams{TelemetryLogsProfileID: "telemetryprofile-12345678"},
	)
	_, err := h.Do(job)
	assert.Error(t, err)

	telGroupRes := inv_testing.CreateTelemetryGroupLogs(t, true)
	siteRes := inv_testing.CreateSite(t, nil, nil)
	profileRes := inv_testing.CreateTelemetryProfile(t, nil, siteRes, nil, telGroupRes, false)

	job = types.NewJob(
		context.TODO(),
		types.Delete,
		types.TelemetryLogsProfile,
		nil,
		inv_handlers.TelemetryLogsProfileURLParams{TelemetryLogsProfileID: profileRes.ResourceId},
	)
	r, err := h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusNoContent, r.Status)

	// Delete error - wrong params
	job = types.NewJob(
		context.TODO(),
		types.Delete,
		types.TelemetryLogsProfile,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_Inventory_TelemetryLogsProfile_Integration(t *testing.T) {
	telemetryProfile := &telemetryv1.TelemetryProfile{}
	validateInventoryIntegration(
		t,
		telemetryProfile,
		api.TelemetryLogsProfile{},
		inv_handlers.OpenAPITelemetryLogsProfileToProto,
		inv_handlers.OpenAPITelemetryLogsProfileToProtoExcluded,
		maps.Values(inv_handlers.OpenAPITelemetryLogsProfileToProto),
		true,
	)
}
