// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

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
	schedulev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/schedule/v1"
	schedule_cache "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client/cache/schedule"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

var (
	now      = int(time.Now().Unix())
	now30Min = now + 1800
	now1Hour = now + 3600
)

func BuildFmFromSingleSchedRequest(body api.SingleSchedule) []string {
	fm := []string{}
	fm = append(fm, "name")
	if body.TargetHostId != nil {
		fm = append(fm, "target_host")
	}
	if body.TargetSiteId != nil {
		fm = append(fm, "target_site")
	}
	if body.TargetRegionId != nil {
		fm = append(fm, "target_region")
	}
	if body.StartSeconds != 0 {
		fm = append(fm, "start_seconds")
	}
	if body.EndSeconds != nil {
		fm = append(fm, "end_seconds")
	}
	if body.ScheduleStatus != "" {
		fm = append(fm, "schedule_status")
	}
	return fm
}

func Test_SchedSingleHandler_Job_Error(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	scheduleCache := schedule_cache.NewScheduleCacheClient(
		inv_testing.TestClients[inv_testing.APIClient].GetTenantAwareInventoryClient())
	hScheduleCache, err := schedule_cache.NewHScheduleCacheClient(scheduleCache)
	require.NoError(t, err)
	h := handlers.NewHandlers(client, hScheduleCache)
	require.NotNil(t, h)

	// test List
	job := types.NewJob(
		ctxTest, BadOperation, types.SingleSched,
		nil, nil,
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))
}

// check that we pass the expected filters to the inventory.
//
//nolint:funlen // it is a test
func Test_SchedSingleHandler_List(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	scheduleCache := schedule_cache.NewScheduleCacheClient(
		inv_testing.TestClients[inv_testing.APIClient].GetTenantAwareInventoryClient())
	hScheduleCache, err := schedule_cache.NewHScheduleCacheClient(scheduleCache)
	require.NoError(t, err)
	h := handlers.NewHandlers(client, hScheduleCache)
	require.NotNil(t, h)

	regionResource := inv_testing.CreateRegion(t, nil)
	inv_testing.CreateSingleSchedule(t, nil, nil, schedulev1.ScheduleStatus_SCHEDULE_STATUS_CLUSTER_UPDATE,
		inv_testing.SSRRegion(regionResource),
	)

	siteResource := inv_testing.CreateSite(t, nil, nil)
	inv_testing.CreateSingleSchedule(t, nil, siteResource, schedulev1.ScheduleStatus_SCHEDULE_STATUS_CLUSTER_UPDATE)

	hostResource := inv_testing.CreateHost(t, nil, nil)
	inv_testing.CreateSingleSchedule(t, hostResource, nil, schedulev1.ScheduleStatus_SCHEDULE_STATUS_CLUSTER_UPDATE)

	inv_testing.CreateSingleSchedule(t, nil, nil, schedulev1.ScheduleStatus_SCHEDULE_STATUS_CLUSTER_UPDATE)

	inv_testing.CreateSingleSchedule(t, nil, nil, schedulev1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
		//nolint:gosec // no overflow for a few billion years
		inv_testing.SSRStart(uint64(time.Now().Unix())+900), // Start: now + 15 min.
		inv_testing.SSREnd(0),
	)

	scheduleCache.LoadAllSchedulesFromInv()

	type tc struct {
		name                          string
		filter                        api.GetSchedulesSingleParams
		expectedNoOfReturnedSchedules int
	}

	tcs := []tc{
		{
			name: "By Site ID",
			filter: api.GetSchedulesSingleParams{
				Offset: &pgOffset, PageSize: &pgSize, SiteID: &utils.SiteUnexistID,
			},
			expectedNoOfReturnedSchedules: 0,
		},
		{
			name: "By Host ID",
			filter: api.GetSchedulesSingleParams{
				Offset: &pgOffset, PageSize: &pgSize, HostID: &utils.HostUnexistID,
			},
			expectedNoOfReturnedSchedules: 0,
		},
		{
			name: "By Region ID",
			filter: api.GetSchedulesSingleParams{
				Offset: &pgOffset, PageSize: &pgSize, RegionID: &utils.RegionUnexistID,
			},
			expectedNoOfReturnedSchedules: 0,
		},
		{
			name: "By Site ID",
			filter: api.GetSchedulesSingleParams{
				Offset: &pgOffset, PageSize: &pgSize, SiteID: &siteResource.ResourceId,
			},
			expectedNoOfReturnedSchedules: 1,
		},
		{
			name: "By Host ID",
			filter: api.GetSchedulesSingleParams{
				Offset: &pgOffset, PageSize: &pgSize, HostID: &hostResource.ResourceId,
			},
			expectedNoOfReturnedSchedules: 1,
		},
		{
			name: "By Region ID",
			filter: api.GetSchedulesSingleParams{
				Offset: &pgOffset, PageSize: &pgSize, RegionID: &regionResource.ResourceId,
			},
			expectedNoOfReturnedSchedules: 1,
		},
		{
			name:                          "All",
			filter:                        api.GetSchedulesSingleParams{Offset: &pgOffset, PageSize: &pgSize},
			expectedNoOfReturnedSchedules: 5,
		},
		{
			name: "By Null Region ID",
			filter: api.GetSchedulesSingleParams{
				Offset: &pgOffset, PageSize: &pgSize, RegionID: &nullString,
			},
			expectedNoOfReturnedSchedules: 4,
		},
		{
			name: "By Null Site ID",
			filter: api.GetSchedulesSingleParams{
				Offset: &pgOffset, PageSize: &pgSize, SiteID: &nullString,
			},
			expectedNoOfReturnedSchedules: 4,
		},
		{
			name: "By Null Host ID",
			filter: api.GetSchedulesSingleParams{
				Offset: &pgOffset, PageSize: &pgSize, HostID: &nullString,
			},
			expectedNoOfReturnedSchedules: 4,
		},
		{
			name: "By Null Region ID",
			filter: api.GetSchedulesSingleParams{
				Offset: &pgOffset, PageSize: &pgSize, RegionID: &nullString,
			},
			expectedNoOfReturnedSchedules: 4,
		},
		{
			name: "By Null Site ID & HostID",
			filter: api.GetSchedulesSingleParams{
				Offset: &pgOffset, PageSize: &pgSize, SiteID: &nullString, HostID: &nullString,
			},
			expectedNoOfReturnedSchedules: 3,
		},
		{
			name: "By Null Site ID & Region ID",
			filter: api.GetSchedulesSingleParams{
				Offset: &pgOffset, PageSize: &pgSize, SiteID: &nullString, RegionID: &nullString,
			},
			expectedNoOfReturnedSchedules: 3,
		},
		{
			name: "By Null Host ID & Region ID",
			filter: api.GetSchedulesSingleParams{
				Offset: &pgOffset, PageSize: &pgSize, HostID: &nullString, RegionID: &nullString,
			},
			expectedNoOfReturnedSchedules: 3,
		},
		{
			name: "All null",
			filter: api.GetSchedulesSingleParams{
				Offset: &pgOffset, PageSize: &pgSize, SiteID: &nullString, HostID: &nullString, RegionID: &nullString,
			},
			expectedNoOfReturnedSchedules: 2,
		},
		{
			name: "By Now Plus 30 Min",
			filter: api.GetSchedulesSingleParams{
				Offset: &pgOffset, PageSize: &pgSize, UnixEpoch: &timeNow30MinString,
			},
			expectedNoOfReturnedSchedules: 1,
		},
	}

	for idx, tc := range tcs {
		t.Run(fmt.Sprintf("%d %s, expected no of schedules: %d",
			idx, tc.name, tc.expectedNoOfReturnedSchedules), func(t *testing.T) {
			job := types.NewJob(ctxTest, types.List, types.SingleSched, tc.filter, nil)
			r, err := h.Do(job)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, r.Status)
			listResources, ok := r.Payload.Data.(api.SingleSchedulesList)
			require.True(t, ok)
			assert.NotNil(t, listResources)
			require.Len(t, *listResources.SingleSchedules, tc.expectedNoOfReturnedSchedules)
		})
	}
}

func Test_SchedSingleHandler_List_WrongParams(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	scheduleCache := schedule_cache.NewScheduleCacheClient(
		inv_testing.TestClients[inv_testing.APIClient].GetTenantAwareInventoryClient())
	hScheduleCache, err := schedule_cache.NewHScheduleCacheClient(scheduleCache)
	require.NoError(t, err)
	h := handlers.NewHandlers(client, hScheduleCache)
	require.NotNil(t, h)

	job := types.NewJob(
		ctxTest, types.List, types.SingleSched,
		api.GetComputeHostsParams{}, nil,
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	// test List error - both Host ID and Site ID
	siteResource := inv_testing.CreateSite(t, nil, nil)
	hostResource := inv_testing.CreateHost(t, nil, nil)
	job = types.NewJob(
		ctxTest, types.List, types.SingleSched,
		api.GetSchedulesSingleParams{
			Offset: &pgOffset, PageSize: &pgSize, SiteID: &siteResource.ResourceId, HostID: &hostResource.ResourceId,
		}, nil,
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

//nolint:funlen // it is a test
func Test_SchedSingleHandler_Post(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	scheduleCache := schedule_cache.NewScheduleCacheClient(
		inv_testing.TestClients[inv_testing.APIClient].GetTenantAwareInventoryClient())
	hScheduleCache, err := schedule_cache.NewHScheduleCacheClient(scheduleCache)
	require.NoError(t, err)
	h := handlers.NewHandlers(client, hScheduleCache)
	require.NotNil(t, h)

	hostResource := inv_testing.CreateHost(t, nil, nil)
	siteResource := inv_testing.CreateSite(t, nil, nil)

	// Test Post with host ID set
	body := api.SingleSchedule{
		StartSeconds:   now30Min,
		EndSeconds:     &now1Hour,
		TargetHostId:   &hostResource.ResourceId,
		ScheduleStatus: api.SCHEDULESTATUSOSUPDATE,
	}
	job := types.NewJob(
		ctxTest, types.Post, types.SingleSched,
		&body, inv_handlers.SingleSchedURLParams{},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, r.Status)

	gotRes, ok := r.Payload.Data.(*api.SingleSchedule)
	require.True(t, ok)
	assert.NotNil(t, gotRes)

	// Validate Post changes
	job = types.NewJob(ctxTest, types.Get, types.SingleSched, nil, inv_handlers.SingleSchedURLParams{
		SingleSchedID: *gotRes.SingleScheduleID,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok = r.Payload.Data.(*api.SingleSchedule)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, hostResource.ResourceId, *gotRes.TargetHostId)

	inv_testing.DeleteResource(t, *gotRes.SingleScheduleID)

	// Test Post with site ID set
	body = api.SingleSchedule{
		StartSeconds:   now30Min,
		EndSeconds:     &now1Hour,
		TargetSiteId:   &siteResource.ResourceId,
		ScheduleStatus: api.SCHEDULESTATUSOSUPDATE,
	}
	job = types.NewJob(
		ctxTest, types.Post, types.SingleSched,
		&body, inv_handlers.SingleSchedURLParams{},
	)
	r, err = h.Do(job)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, r.Status)

	gotRes, ok = r.Payload.Data.(*api.SingleSchedule)
	require.True(t, ok)
	assert.NotNil(t, gotRes)

	// Validate Post changes
	job = types.NewJob(ctxTest, types.Get, types.SingleSched, nil, inv_handlers.SingleSchedURLParams{
		SingleSchedID: *gotRes.SingleScheduleID,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok = r.Payload.Data.(*api.SingleSchedule)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, siteResource.ResourceId, *gotRes.TargetSiteId)

	inv_testing.DeleteResource(t, *gotRes.SingleScheduleID)

	// Test Post with host and site IDs set
	body = api.SingleSchedule{
		StartSeconds:   now30Min,
		EndSeconds:     &now1Hour,
		TargetHostId:   &utils.HostUnexistID,
		TargetSiteId:   &utils.SiteUnexistID,
		ScheduleStatus: api.SCHEDULESTATUSOSUPDATE,
	}
	job = types.NewJob(
		ctxTest, types.Post, types.SingleSched,
		&body, inv_handlers.SingleSchedURLParams{},
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	// Test Post with start seconds bigger than end seconds
	body = api.SingleSchedule{
		StartSeconds:   now30Min,
		EndSeconds:     &now1Hour,
		TargetSiteId:   &utils.SiteUnexistID,
		ScheduleStatus: api.SCHEDULESTATUSOSUPDATE,
	}
	job = types.NewJob(
		ctxTest, types.Post, types.SingleSched,
		&body, inv_handlers.SingleSchedURLParams{},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	// Test Post error - wrong body request format
	job = types.NewJob(
		ctxTest, types.Post, types.SingleSched,
		&api.Host{}, nil,
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_SchedSingleHandler_PostWithRegion(t *testing.T) {
	h := createInventoryClientHandlers(t)

	regionResource := inv_testing.CreateRegion(t, nil)

	body := api.SingleSchedule{
		StartSeconds:   now30Min,
		EndSeconds:     &now1Hour,
		TargetRegionId: &regionResource.ResourceId,
		ScheduleStatus: api.SCHEDULESTATUSOSUPDATE,
	}
	job := types.NewJob(
		ctxTest, types.Post, types.SingleSched,
		&body, inv_handlers.SingleSchedURLParams{},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, r.Status)

	gotRes, ok := r.Payload.Data.(*api.SingleSchedule)
	require.True(t, ok)
	assert.NotNil(t, gotRes)

	// Validate Post changes
	job = types.NewJob(ctxTest, types.Get, types.SingleSched, nil, inv_handlers.SingleSchedURLParams{
		SingleSchedID: *gotRes.SingleScheduleID,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok = r.Payload.Data.(*api.SingleSchedule)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, regionResource.ResourceId, *gotRes.TargetRegionId)

	inv_testing.DeleteResource(t, *gotRes.SingleScheduleID)
}

func Test_SchedSingleHandler_Put(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	scheduleCache := schedule_cache.NewScheduleCacheClient(
		inv_testing.TestClients[inv_testing.APIClient].GetTenantAwareInventoryClient())
	hScheduleCache, err := schedule_cache.NewHScheduleCacheClient(scheduleCache)
	require.NoError(t, err)
	h := handlers.NewHandlers(client, hScheduleCache)
	require.NotNil(t, h)

	hostResource := inv_testing.CreateHost(t, nil, nil)
	singleSchedResource := inv_testing.CreateSingleSchedule(
		t,
		hostResource,
		nil,
		schedulev1.ScheduleStatus_SCHEDULE_STATUS_CLUSTER_UPDATE,
	)
	bodyUpdate := api.SingleSchedule{
		StartSeconds:   now30Min,
		TargetHostId:   &hostResource.ResourceId,
		ScheduleStatus: api.SCHEDULESTATUSOSUPDATE,
	}

	job := types.NewJob(
		ctxTest,
		types.Put,
		types.SingleSched,
		&bodyUpdate,
		inv_handlers.SingleSchedURLParams{SingleSchedID: singleSchedResource.ResourceId},
	)
	r, err := h.Do(job)
	assert.Equal(t, http.StatusOK, r.Status)
	assert.NoError(t, err)

	// Validate Put changes
	job = types.NewJob(ctxTest, types.Get, types.SingleSched, nil, inv_handlers.SingleSchedURLParams{
		SingleSchedID: singleSchedResource.ResourceId,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.SingleSchedule)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, hostResource.ResourceId, *gotRes.TargetHostId)
	assert.Equal(t, api.SCHEDULESTATUSOSUPDATE, gotRes.ScheduleStatus)

	// Test Put  error - wrong body
	job = types.NewJob(
		ctxTest,
		types.Put,
		types.SingleSched,
		&api.Host{},
		inv_handlers.SingleSchedURLParams{SingleSchedID: singleSchedResource.ResourceId},
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	// Test Post with start seconds bigger than end seconds
	endSec := 10
	bodyUpdate = api.SingleSchedule{
		StartSeconds:   20,
		EndSeconds:     &endSec,
		TargetSiteId:   &utils.SiteUnexistID,
		ScheduleStatus: api.SCHEDULESTATUSOSUPDATE,
	}
	job = types.NewJob(
		ctxTest, types.Post, types.SingleSched,
		&bodyUpdate, inv_handlers.SingleSchedURLParams{},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	// Test Put  error - wrong params
	job = types.NewJob(
		ctxTest,
		types.Put,
		types.SingleSched,
		&bodyUpdate,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_SchedSingleHandler_Patch(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	scheduleCache := schedule_cache.NewScheduleCacheClient(
		inv_testing.TestClients[inv_testing.APIClient].GetTenantAwareInventoryClient())
	hScheduleCache, err := schedule_cache.NewHScheduleCacheClient(scheduleCache)
	require.NoError(t, err)
	h := handlers.NewHandlers(client, hScheduleCache)
	assert.NotEqual(t, h, nil)

	hostResource := inv_testing.CreateHost(t, nil, nil)
	singleSchedResource := inv_testing.CreateSingleSchedule(
		t,
		hostResource,
		nil,
		schedulev1.ScheduleStatus_SCHEDULE_STATUS_CLUSTER_UPDATE,
	)
	bodyUpdate := api.SingleSchedule{
		StartSeconds:   now30Min,
		ScheduleStatus: api.SCHEDULESTATUSOSUPDATE,
	}

	job := types.NewJob(
		ctxTest,
		types.Patch,
		types.SingleSched,
		&bodyUpdate,
		inv_handlers.SingleSchedURLParams{SingleSchedID: singleSchedResource.ResourceId},
	)
	r, err := h.Do(job)
	assert.Equal(t, http.StatusOK, r.Status)
	assert.NoError(t, err)

	// Validate Patch changes
	job = types.NewJob(ctxTest, types.Get, types.SingleSched, nil, inv_handlers.SingleSchedURLParams{
		SingleSchedID: singleSchedResource.ResourceId,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.SingleSchedule)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, hostResource.ResourceId, *gotRes.TargetHostId)
	assert.Equal(t, api.SCHEDULESTATUSOSUPDATE, gotRes.ScheduleStatus)

	// Test Put  error - wrong body
	job = types.NewJob(
		ctxTest,
		types.Put,
		types.SingleSched,
		&api.Host{},
		inv_handlers.SingleSchedURLParams{SingleSchedID: singleSchedResource.ResourceId},
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_SchedSingleHandler_PatchFieldMask(t *testing.T) {
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
	scheduleCache := schedule_cache.NewScheduleCacheClient(
		inv_testing.TestClients[inv_testing.APIClient].GetTenantAwareInventoryClient())
	hScheduleCache, err := schedule_cache.NewHScheduleCacheClient(scheduleCache)
	require.NoError(t, err)
	h := handlers.NewHandlers(client, hScheduleCache)
	require.NotNil(t, h)

	bodyUpdate := api.SingleSchedule{
		Name:           &utils.SschedName1,
		StartSeconds:   now30Min,
		TargetSiteId:   &emptyString,
		ScheduleStatus: api.SCHEDULESTATUSOSUPDATE,
	}

	job := types.NewJob(
		ctxTest,
		types.Patch,
		types.SingleSched,
		&bodyUpdate,
		inv_handlers.SingleSchedURLParams{SingleSchedID: "singlesche-12345678"},
	)
	r, err := h.Do(job)
	assert.Equal(t, http.StatusOK, r.Status)
	assert.NoError(t, err)

	// test Patch FieldMask
	expectedPatchFieldMask := BuildFmFromSingleSchedRequest(bodyUpdate)
	sched := &schedulev1.SingleScheduleResource{}
	expectedFieldMask, err := fieldmaskpb.New(sched, expectedPatchFieldMask...)
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

func Test_SchedSingleHandler_Get(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	scheduleCache := schedule_cache.NewScheduleCacheClient(
		inv_testing.TestClients[inv_testing.APIClient].GetTenantAwareInventoryClient())
	hScheduleCache, err := schedule_cache.NewHScheduleCacheClient(scheduleCache)
	require.NoError(t, err)
	h := handlers.NewHandlers(client, hScheduleCache)
	require.NotNil(t, h)

	job := types.NewJob(
		ctxTest,
		types.Get,
		types.SingleSched,
		nil,
		inv_handlers.SingleSchedURLParams{SingleSchedID: "singlesche-12345678"},
	)
	_, err = h.Do(job)
	assert.NotEqual(t, nil, err)

	hostResource := inv_testing.CreateHost(t, nil, nil)
	singleSchedResource := inv_testing.CreateSingleSchedule(
		t,
		hostResource,
		nil,
		schedulev1.ScheduleStatus_SCHEDULE_STATUS_CLUSTER_UPDATE,
	)
	scheduleCache.LoadAllSchedulesFromInv()

	// Validate Get changes
	job = types.NewJob(ctxTest, types.Get, types.SingleSched, nil, inv_handlers.SingleSchedURLParams{
		SingleSchedID: singleSchedResource.ResourceId,
	})
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.SingleSchedule)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, hostResource.ResourceId, *gotRes.TargetHostId)

	// Get error - wrong params
	job = types.NewJob(
		ctxTest,
		types.Get,
		types.SingleSched,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_SchedSingleHandler_Delete(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	scheduleCache := schedule_cache.NewScheduleCacheClient(
		inv_testing.TestClients[inv_testing.APIClient].GetTenantAwareInventoryClient())
	hScheduleCache, err := schedule_cache.NewHScheduleCacheClient(scheduleCache)
	require.NoError(t, err)
	h := handlers.NewHandlers(client, hScheduleCache)
	require.NotNil(t, h)

	hostResource := inv_testing.CreateHost(t, nil, nil)
	singleSchedResource := inv_testing.CreateSingleScheduleNoCleanup(
		t,
		hostResource,
		nil,
		schedulev1.ScheduleStatus_SCHEDULE_STATUS_CLUSTER_UPDATE,
	)
	scheduleCache.LoadAllSchedulesFromInv()

	job := types.NewJob(
		ctxTest,
		types.Delete,
		types.SingleSched,
		nil,
		inv_handlers.SingleSchedURLParams{SingleSchedID: singleSchedResource.ResourceId},
	)
	r, err := h.Do(job)
	assert.Equal(t, http.StatusNoContent, r.Status)
	assert.NoError(t, err)

	// Validate Delete changes
	job = types.NewJob(ctxTest, types.Get, types.SingleSched, nil, inv_handlers.SingleSchedURLParams{
		SingleSchedID: singleSchedResource.ResourceId,
	})
	_, err = h.Do(job)
	assert.Error(t, err)

	// Delete error - wrong params
	job = types.NewJob(
		ctxTest,
		types.Delete,
		types.SingleSched,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_Inventory_SingleSched_Integration(t *testing.T) {
	// verify the projection of the constants to Proto first;
	// we build a map using the field names of the proto stored in the
	// ProtoSingleSched* slices in internal/work/handlers/schedsingle.go. Elements must
	// have a mapping key otherwise we throw an error if there is no
	// alignment with SingleScheduleResource proto in Inventory. Make sure to update these
	// two slices in internal/work/handlers/schedsingle.go
	schedResource := &schedulev1.SingleScheduleResource{}
	validateInventoryIntegration(
		t,
		schedResource,
		api.SingleSchedule{},
		inv_handlers.OpenAPISingleSchedToProto,
		inv_handlers.OpenAPISingleSchedToProtoExcluded,
		maps.Values(inv_handlers.OpenAPISingleSchedToProto),
		true,
	)
}

// Test_SchedSingleHandler_InvMockClient_Errors evaluates all
// Sched Single handler methods with mock inventory client
// that returns errors.
func Test_SchedSingleHandler_InvMockClient_Errors(t *testing.T) {
	mockClient := utils.NewMockInventoryServiceClientError()
	client := &clients.InventoryClientHandler{
		InvClient: mockClient,
	}
	scheduleCache := schedule_cache.NewScheduleCacheClient(
		inv_testing.TestClients[inv_testing.APIClient].GetTenantAwareInventoryClient())
	hScheduleCache, err := schedule_cache.NewHScheduleCacheClient(scheduleCache)
	require.NoError(t, err)
	h := handlers.NewHandlers(client, hScheduleCache)
	require.NotNil(t, h)

	// Post response error
	body := api.SingleSchedule{
		StartSeconds:   now30Min,
		TargetSiteId:   &utils.SiteUnexistID,
		ScheduleStatus: api.SCHEDULESTATUSOSUPDATE,
	}
	job := types.NewJob(
		ctxTest, types.Post, types.SingleSched,
		&body, inv_handlers.SingleSchedURLParams{},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	// Put response error
	job = types.NewJob(
		ctxTest,
		types.Put,
		types.SingleSched,
		&body,
		inv_handlers.SingleSchedURLParams{SingleSchedID: "singlesche-12345678"},
	)

	_, err = h.Do(job)
	assert.Error(t, err)

	// Get response error
	job = types.NewJob(
		ctxTest,
		types.Get,
		types.SingleSched,
		nil,
		inv_handlers.SingleSchedURLParams{SingleSchedID: "singlesche-12345678"},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	// Delete response error
	job = types.NewJob(
		ctxTest,
		types.Delete,
		types.SingleSched,
		nil,
		inv_handlers.SingleSchedURLParams{SingleSchedID: "singlesche-12345678"},
	)
	_, err = h.Do(job)
	assert.Error(t, err)
}

func createInventoryClientHandlers(t *testing.T) handlers.Handlers {
	t.Helper()
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	scheduleCache := schedule_cache.NewScheduleCacheClient(
		inv_testing.TestClients[inv_testing.APIClient].GetTenantAwareInventoryClient())
	hScheduleCache, err := schedule_cache.NewHScheduleCacheClient(scheduleCache)
	require.NoError(t, err)
	h := handlers.NewHandlers(client, hScheduleCache)
	require.NotNil(t, h)
	return h
}
