// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers_test

import (
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
	schedulev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/schedule/v1"
	schedule_cache "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client/cache/schedule"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

//nolint:cyclop // test only
func BuildFmFromRepeatedSchedRequest(body api.RepeatedSchedule) []string {
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
	if body.DurationSeconds != 0 {
		fm = append(fm, "duration_seconds")
	}
	if body.ScheduleStatus != "" {
		fm = append(fm, "schedule_status")
	}
	if body.CronDayMonth != "" {
		fm = append(fm, "cron_day_month")
	}
	if body.CronDayWeek != "" {
		fm = append(fm, "cron_day_week")
	}
	if body.CronHours != "" {
		fm = append(fm, "cron_hours")
	}
	if body.CronMinutes != "" {
		fm = append(fm, "cron_minutes")
	}
	if body.CronMonth != "" {
		fm = append(fm, "cron_month")
	}
	return fm
}

func Test_SchedRepeatedHandler_Job_Error(t *testing.T) {
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
		ctxTest, BadOperation, types.RepeatedSched,
		nil, nil,
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))
}

// check that we pass the expected filters to the inventory.
//
//nolint:funlen // it is a test
func Test_SchedRepeatedHandler_List(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	scheduleCache := schedule_cache.NewScheduleCacheClient(
		inv_testing.TestClients[inv_testing.APIClient].GetTenantAwareInventoryClient())
	hScheduleCache, err := schedule_cache.NewHScheduleCacheClient(scheduleCache)
	require.NoError(t, err)
	h := handlers.NewHandlers(client, hScheduleCache)
	require.NotNil(t, h)

	siteResource := inv_testing.CreateSite(t, nil, nil)
	inv_testing.CreateRepeatedSchedule(t, nil, siteResource,
		schedulev1.ScheduleStatus_SCHEDULE_STATUS_CLUSTER_UPDATE)

	hostResource := inv_testing.CreateHost(t, nil, nil)
	inv_testing.CreateRepeatedSchedule(t, hostResource, nil,
		schedulev1.ScheduleStatus_SCHEDULE_STATUS_CLUSTER_UPDATE)

	regionResource := inv_testing.CreateRegion(t, nil)
	inv_testing.CreateRepeatedSchedule(t, nil, nil,
		schedulev1.ScheduleStatus_SCHEDULE_STATUS_CLUSTER_UPDATE,
		inv_testing.RSRRegion(regionResource))

	inv_testing.CreateRepeatedSchedule(t, nil, nil, schedulev1.ScheduleStatus_SCHEDULE_STATUS_CLUSTER_UPDATE)

	inv_testing.CreateRepeatedSchedule(t, nil, nil, schedulev1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
		inv_testing.RSRDayWeek(fmt.Sprintf("%d", timeNow.Weekday())), inv_testing.RSRDayMonth("*"),
		inv_testing.RSRMonth("*"), inv_testing.RSRHours("*"), inv_testing.RSRMinutes("*"),
	)

	scheduleCache.LoadAllSchedulesFromInv()

	type tc struct {
		name                          string
		filter                        api.GetSchedulesRepeatedParams
		expectedNoOfReturnedSchedules int
	}

	tcs := []tc{
		{
			name: "By Site ID",
			filter: api.GetSchedulesRepeatedParams{
				Offset: &pgOffset, PageSize: &pgSize, SiteID: &utils.SiteUnexistID,
			},
			expectedNoOfReturnedSchedules: 0,
		},
		{
			name: "By Host ID",
			filter: api.GetSchedulesRepeatedParams{
				Offset: &pgOffset, PageSize: &pgSize, HostID: &utils.HostUnexistID,
			},
			expectedNoOfReturnedSchedules: 0,
		},
		{
			name: "By Region ID",
			filter: api.GetSchedulesRepeatedParams{
				Offset: &pgOffset, PageSize: &pgSize, RegionID: &utils.RegionUnexistID,
			},
			expectedNoOfReturnedSchedules: 0,
		},
		{
			name: "By Site ID",
			filter: api.GetSchedulesRepeatedParams{
				Offset: &pgOffset, PageSize: &pgSize, SiteID: &siteResource.ResourceId,
			},
			expectedNoOfReturnedSchedules: 1,
		},
		{
			name: "By Host ID",
			filter: api.GetSchedulesRepeatedParams{
				Offset: &pgOffset, PageSize: &pgSize, HostID: &hostResource.ResourceId,
			},
			expectedNoOfReturnedSchedules: 1,
		},
		{
			name: "By Region ID",
			filter: api.GetSchedulesRepeatedParams{
				Offset: &pgOffset, PageSize: &pgSize, RegionID: &regionResource.ResourceId,
			},
			expectedNoOfReturnedSchedules: 1,
		},
		{
			name:                          "All",
			filter:                        api.GetSchedulesRepeatedParams{Offset: &pgOffset, PageSize: &pgSize},
			expectedNoOfReturnedSchedules: 5,
		},
		{
			name: "By Null Region ID",
			filter: api.GetSchedulesRepeatedParams{
				Offset: &pgOffset, PageSize: &pgSize, RegionID: &nullString,
			},
			expectedNoOfReturnedSchedules: 4,
		},
		{
			name: "By Null Site ID",
			filter: api.GetSchedulesRepeatedParams{
				Offset: &pgOffset, PageSize: &pgSize, SiteID: &nullString,
			},
			expectedNoOfReturnedSchedules: 4,
		},
		{
			name: "By Null Host ID",
			filter: api.GetSchedulesRepeatedParams{
				Offset: &pgOffset, PageSize: &pgSize, HostID: &nullString,
			},
			expectedNoOfReturnedSchedules: 4,
		},
		{
			name: "By Null Region ID",
			filter: api.GetSchedulesRepeatedParams{
				Offset: &pgOffset, PageSize: &pgSize, RegionID: &nullString,
			},
			expectedNoOfReturnedSchedules: 4,
		},
		{
			name: "By Null Site ID & HostID",
			filter: api.GetSchedulesRepeatedParams{
				Offset: &pgOffset, PageSize: &pgSize, SiteID: &nullString, HostID: &nullString,
			},
			expectedNoOfReturnedSchedules: 3,
		},
		{
			name: "By Null Site ID & Region ID",
			filter: api.GetSchedulesRepeatedParams{
				Offset: &pgOffset, PageSize: &pgSize, SiteID: &nullString, RegionID: &nullString,
			},
			expectedNoOfReturnedSchedules: 3,
		},
		{
			name: "By Null Host ID & Region ID",
			filter: api.GetSchedulesRepeatedParams{
				Offset: &pgOffset, PageSize: &pgSize, HostID: &nullString, RegionID: &nullString,
			},
			expectedNoOfReturnedSchedules: 3,
		},
		{
			name: "All null",
			filter: api.GetSchedulesRepeatedParams{
				Offset: &pgOffset, PageSize: &pgSize, SiteID: &nullString, HostID: &nullString, RegionID: &nullString,
			},
			expectedNoOfReturnedSchedules: 2,
		},
		{
			name: "By Now Plus 30 Min",
			filter: api.GetSchedulesRepeatedParams{
				Offset: &pgOffset, PageSize: &pgSize, UnixEpoch: &timeNow30MinString,
			},
			expectedNoOfReturnedSchedules: 1,
		},
	}

	for idx, tc := range tcs {
		t.Run(fmt.Sprintf("%d, %s, expected no of schedules: %d",
			idx, tc.name, tc.expectedNoOfReturnedSchedules), func(t *testing.T) {
			job := types.NewJob(ctxTest, types.List, types.RepeatedSched, tc.filter, nil)
			r, err := h.Do(job)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, r.Status)
			listResources, ok := r.Payload.Data.(api.RepeatedSchedulesList)
			require.True(t, ok)
			assert.NotNil(t, listResources)
			require.Len(t, *listResources.RepeatedSchedules, tc.expectedNoOfReturnedSchedules)
		})
	}
}

func Test_SchedRepeatedHandler_List_WrongParams(t *testing.T) {
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
		ctxTest, types.List, types.RepeatedSched,
		api.GetComputeHostsParams{}, nil,
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	// test List error - both Host ID and Site ID
	siteResource := inv_testing.CreateSite(t, nil, nil)
	hostResource := inv_testing.CreateHost(t, nil, nil)
	job = types.NewJob(
		ctxTest, types.List, types.RepeatedSched,
		api.GetSchedulesRepeatedParams{
			Offset: &pgOffset, PageSize: &pgSize, SiteID: &siteResource.ResourceId, HostID: &hostResource.ResourceId,
		}, nil,
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_SchedRepeatedHandler_Post_WithHost(t *testing.T) {
	h := createHandlersWithCache(t)

	hostResource := inv_testing.CreateHost(t, nil, nil)

	body := api.RepeatedSchedule{
		DurationSeconds: 1,
		TargetHostId:    &hostResource.ResourceId,
		ScheduleStatus:  api.SCHEDULESTATUSOSUPDATE,
		CronDayMonth:    "*",
		CronDayWeek:     "*",
		CronHours:       "*",
		CronMinutes:     "*",
		CronMonth:       "*",
	}
	job := types.NewJob(
		ctxTest, types.Post, types.RepeatedSched,
		&body, inv_handlers.RepeatedSchedURLParams{},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, r.Status)
	gotRes, ok := r.Payload.Data.(*api.RepeatedSchedule)
	require.True(t, ok)
	assert.NotNil(t, gotRes)

	// Validate Post changes
	job = types.NewJob(ctxTest, types.Get, types.RepeatedSched, nil, inv_handlers.RepeatedSchedURLParams{
		RepeatedSchedID: *gotRes.RepeatedScheduleID,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok = r.Payload.Data.(*api.RepeatedSchedule)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, hostResource.ResourceId, *gotRes.TargetHostId)

	body = api.RepeatedSchedule{
		DurationSeconds: 1,
		TargetHostId:    &utils.HostUnexistID,
		TargetSiteId:    &utils.SiteUnexistID,
		ScheduleStatus:  api.SCHEDULESTATUSOSUPDATE,
	}
	job = types.NewJob(
		ctxTest, types.Post, types.RepeatedSched,
		&body, inv_handlers.RepeatedSchedURLParams{},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	inv_testing.DeleteResource(t, *gotRes.RepeatedScheduleID)
}

func Test_SchedRepeatedHandler_Post_WithRegion(t *testing.T) {
	h := createHandlersWithCache(t)

	regionResource := inv_testing.CreateRegion(t, nil)

	body := api.RepeatedSchedule{
		DurationSeconds: 1,
		TargetRegionId:  &regionResource.ResourceId,
		ScheduleStatus:  api.SCHEDULESTATUSOSUPDATE,
		CronDayMonth:    "*",
		CronDayWeek:     "*",
		CronHours:       "*",
		CronMinutes:     "*",
		CronMonth:       "*",
	}
	job := types.NewJob(
		ctxTest, types.Post, types.RepeatedSched,
		&body, inv_handlers.RepeatedSchedURLParams{},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, r.Status)
	gotRes, ok := r.Payload.Data.(*api.RepeatedSchedule)
	require.True(t, ok)
	assert.NotNil(t, gotRes)

	// Validate Post changes
	job = types.NewJob(ctxTest, types.Get, types.RepeatedSched, nil, inv_handlers.RepeatedSchedURLParams{
		RepeatedSchedID: *gotRes.RepeatedScheduleID,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok = r.Payload.Data.(*api.RepeatedSchedule)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, regionResource.ResourceId, *gotRes.TargetRegionId)

	inv_testing.DeleteResource(t, *gotRes.RepeatedScheduleID)
}

func Test_SchedRepeatedHandler_Post_WrongBody(t *testing.T) {
	h := createHandlersWithCache(t)

	job := types.NewJob(
		ctxTest, types.Post, types.RepeatedSched,
		&api.Host{}, nil,
	)
	_, err := h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_SchedRepeatedHandler_Post_WithMultipleTargets(t *testing.T) {
	h := createHandlersWithCache(t)

	body := api.RepeatedSchedule{
		DurationSeconds: 1,
		TargetHostId:    &utils.HostUnexistID,
		TargetSiteId:    &utils.SiteUnexistID,
		ScheduleStatus:  api.SCHEDULESTATUSOSUPDATE,
	}
	job := types.NewJob(
		ctxTest, types.Post, types.RepeatedSched,
		&body, inv_handlers.RepeatedSchedURLParams{},
	)
	_, err := h.Do(job)
	assert.Error(t, err)
}

func createHandlersWithCache(t *testing.T) handlers.Handlers {
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

func Test_SchedRepeatedHandler_Put(t *testing.T) {
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
	repeatSchedResource := inv_testing.CreateRepeatedSchedule(
		t,
		hostResource,
		nil,
		schedulev1.ScheduleStatus_SCHEDULE_STATUS_CLUSTER_UPDATE,
	)

	bodyUpdate := api.RepeatedSchedule{
		DurationSeconds: 1,
		TargetHostId:    &hostResource.ResourceId,
		ScheduleStatus:  api.SCHEDULESTATUSOSUPDATE,
		CronDayMonth:    "*",
		CronDayWeek:     "*",
		CronHours:       "*",
		CronMinutes:     "*",
		CronMonth:       "*",
	}

	job := types.NewJob(
		ctxTest,
		types.Put,
		types.RepeatedSched,
		&bodyUpdate,
		inv_handlers.RepeatedSchedURLParams{RepeatedSchedID: repeatSchedResource.ResourceId},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	// Validate Put changes
	job = types.NewJob(ctxTest, types.Get, types.RepeatedSched, nil, inv_handlers.RepeatedSchedURLParams{
		RepeatedSchedID: repeatSchedResource.ResourceId,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.RepeatedSchedule)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, hostResource.ResourceId, *gotRes.TargetHostId)
	assert.Equal(t, api.SCHEDULESTATUSOSUPDATE, gotRes.ScheduleStatus)

	// Test Put error - wrong body
	job = types.NewJob(
		ctxTest,
		types.Put,
		types.RepeatedSched,
		&api.Host{},
		inv_handlers.RepeatedSchedURLParams{RepeatedSchedID: repeatSchedResource.ResourceId},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	// Test Put error - wrong params
	job = types.NewJob(
		ctxTest,
		types.Put,
		types.RepeatedSched,
		&bodyUpdate,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_SchedRepeatedHandler_Patch(t *testing.T) {
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
	repeatSchedResource := inv_testing.CreateRepeatedSchedule(
		t,
		hostResource,
		nil,
		schedulev1.ScheduleStatus_SCHEDULE_STATUS_CLUSTER_UPDATE,
	)

	bodyUpdate := api.RepeatedSchedule{
		DurationSeconds: 1,
		ScheduleStatus:  api.SCHEDULESTATUSOSUPDATE,
		CronDayMonth:    "*",
		CronDayWeek:     "*",
		CronHours:       "*",
		CronMinutes:     "*",
		CronMonth:       "*",
	}

	job := types.NewJob(
		ctxTest,
		types.Patch,
		types.RepeatedSched,
		&bodyUpdate,
		inv_handlers.RepeatedSchedURLParams{RepeatedSchedID: repeatSchedResource.ResourceId},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	// Validate Put changes
	job = types.NewJob(ctxTest, types.Get, types.RepeatedSched, nil, inv_handlers.RepeatedSchedURLParams{
		RepeatedSchedID: repeatSchedResource.ResourceId,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.RepeatedSchedule)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, hostResource.ResourceId, *gotRes.TargetHostId)
	assert.Equal(t, api.SCHEDULESTATUSOSUPDATE, gotRes.ScheduleStatus)
}

func Test_SchedRepeatedHandler_PatchFieldMask(t *testing.T) {
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

	bodyUpdate := api.RepeatedSchedule{
		Name:            &utils.SschedName1,
		DurationSeconds: 1,
		TargetHostId:    &utils.HostUnexistID,
		TargetSiteId:    &emptyString,
		ScheduleStatus:  api.SCHEDULESTATUSOSUPDATE,
		CronDayMonth:    "*",
		CronDayWeek:     "*",
		CronHours:       "*",
		CronMinutes:     "*",
		CronMonth:       "*",
	}

	job := types.NewJob(
		ctxTest,
		types.Patch,
		types.RepeatedSched,
		&bodyUpdate,
		inv_handlers.RepeatedSchedURLParams{RepeatedSchedID: "repeatedsche-12345678"},
	)
	r, err := h.Do(job)
	assert.Equal(t, http.StatusOK, r.Status)
	assert.NoError(t, err)

	// test Patch FieldMask
	expectedPatchFieldMask := BuildFmFromRepeatedSchedRequest(bodyUpdate)
	sched := &schedulev1.RepeatedScheduleResource{}
	expectedFieldMask, err := fieldmaskpb.New(sched, expectedPatchFieldMask...)
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
	assert.NoError(t, err)
}

func Test_SchedRepeatedHandler_Get(t *testing.T) {
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
		types.RepeatedSched,
		nil,
		inv_handlers.RepeatedSchedURLParams{RepeatedSchedID: "repeatedsche-12345678"},
	)
	_, err = h.Do(job)
	assert.NotEqual(t, nil, err)

	hostResource := inv_testing.CreateHost(t, nil, nil)
	repeatSchedResource := inv_testing.CreateRepeatedSchedule(
		t,
		hostResource,
		nil,
		schedulev1.ScheduleStatus_SCHEDULE_STATUS_CLUSTER_UPDATE,
	)
	scheduleCache.LoadAllSchedulesFromInv()

	job = types.NewJob(ctxTest, types.Get, types.RepeatedSched, nil, inv_handlers.RepeatedSchedURLParams{
		RepeatedSchedID: repeatSchedResource.ResourceId,
	})
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.RepeatedSchedule)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, hostResource.ResourceId, *gotRes.TargetHostId)

	// Get error - wrong params
	job = types.NewJob(
		ctxTest,
		types.Get,
		types.RepeatedSched,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_SchedRepeatedHandler_Delete(t *testing.T) {
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
	repeatSchedResource := inv_testing.CreateRepeatedScheduleNoCleaup(
		t,
		hostResource,
		nil,
		schedulev1.ScheduleStatus_SCHEDULE_STATUS_CLUSTER_UPDATE,
	)
	scheduleCache.LoadAllSchedulesFromInv()

	job := types.NewJob(
		ctxTest,
		types.Delete,
		types.RepeatedSched,
		nil,
		inv_handlers.RepeatedSchedURLParams{RepeatedSchedID: repeatSchedResource.ResourceId},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.Status)

	job = types.NewJob(ctxTest, types.Get, types.RepeatedSched, nil, inv_handlers.RepeatedSchedURLParams{
		RepeatedSchedID: repeatSchedResource.ResourceId,
	})
	_, err = h.Do(job)
	assert.Error(t, err)

	// Delete error - wrong params
	job = types.NewJob(
		ctxTest,
		types.Delete,
		types.RepeatedSched,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_Inventory_RepeatedSched_Integration(t *testing.T) {
	// verify the projection of the constants to Proto first;
	// we build a map using the field names of the proto stored in the
	// ProtoRepeatedSched* slices in internal/work/handlers/schedrepeat.go. Elements must
	// have a mapping key otherwise we throw an error if there is no
	// alignment with RepeatedScheduleResource proto in Inventory. Make sure to update these
	// two slices in internal/work/handlers/schedrepeat.go
	schedResource := &schedulev1.RepeatedScheduleResource{}
	validateInventoryIntegration(
		t,
		schedResource,
		api.RepeatedSchedule{},
		inv_handlers.OpenAPIRepeatedSchedToProto,
		inv_handlers.OpenAPIRepeatedSchedToProtoExcluded,
		maps.Values(inv_handlers.OpenAPIRepeatedSchedToProto),
		true,
	)
}

// Test_SchedRepeatedHandler_InvMockClient_Errors evaluates all
// Sched Repeated handler methods with mock inventory client
// that returns errors.
func Test_SchedRepeatedHandler_InvMockClient_Errors(t *testing.T) {
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

	body := api.RepeatedSchedule{
		DurationSeconds: 1,
		TargetHostId:    &utils.HostUnexistID,
		ScheduleStatus:  api.SCHEDULESTATUSOSUPDATE,
	}
	job := types.NewJob(
		ctxTest, types.Post, types.RepeatedSched,
		&body, inv_handlers.RepeatedSchedURLParams{},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	job = types.NewJob(
		ctxTest,
		types.Put,
		types.RepeatedSched,
		&body,
		inv_handlers.RepeatedSchedURLParams{RepeatedSchedID: "repeatedsche-12345678"},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	job = types.NewJob(
		ctxTest,
		types.Get,
		types.RepeatedSched,
		nil,
		inv_handlers.RepeatedSchedURLParams{RepeatedSchedID: "repeatedsche-12345678"},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	job = types.NewJob(
		ctxTest,
		types.Delete,
		types.RepeatedSched,
		nil,
		inv_handlers.RepeatedSchedURLParams{RepeatedSchedID: "repeatedsche-12345678"},
	)
	_, err = h.Do(job)
	assert.Error(t, err)
}
