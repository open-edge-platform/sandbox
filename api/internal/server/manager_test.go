// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server_test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/api/internal/common"
	"github.com/open-edge-platform/infra-core/api/internal/dispatcher"
	"github.com/open-edge-platform/infra-core/api/internal/server"
	"github.com/open-edge-platform/infra-core/api/internal/types"
	handlers "github.com/open-edge-platform/infra-core/api/internal/worker/handlers/invhandlers"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
)

var (
	pgSize      = 10
	pgOffset    = 0
	showSites   = true
	showRegions = true
)

const (
	localhostAddress = "localhost:50051"
	shutdownTimeout  = 2
)

type testResponseWriter struct {
	name string
}

func (testResponseWriter) Header() http.Header {
	fmt.Println("testResponseWriter Header")
	return http.Header{}
}

func (testResponseWriter) Write([]byte) (int, error) {
	fmt.Println("testResponseWriter Write")
	return 0, nil
}

func (testResponseWriter) WriteHeader(statusCode int) {
	fmt.Printf("testResponseWriter WriteHeader: %d\n", statusCode)
}

func TestNewManagerDispatchWait(t *testing.T) {
	cfg := common.DefaultConfig()
	cfg.Inventory.Address = localhostAddress
	cfg.RestServer.Timeout = 1 * time.Second
	assert.NotEqual(t, cfg, nil)
	dispChan := make(chan bool, 1)
	termChan := make(chan bool, 1)
	wg := sync.WaitGroup{}
	disp := dispatcher.NewDispatcher(cfg, dispChan, termChan, &wg)
	assert.NotEqual(t, disp, nil)
	ctx := context.TODO()

	h, err := server.NewHandlers(disp.JobQueue, cfg.RestServer.Timeout)
	require.NoError(t, err)
	require.NotNil(t, h)

	// Tests job dispatched and response received with correct job ID
	jobOk := types.NewJob(
		ctx,
		types.List,
		types.Host,
		api.GetComputeHostsParams{},
		handlers.HostURLParams{},
	)
	go func() {
		resp := h.DispatchAndWait(jobOk)
		assert.NotEqual(t, resp, nil)
	}()

	respOK := &types.Response{
		Payload: types.Payload{Data: api.ProblemDetails{
			Message: nil,
		}},
		Status: 0,
		ID:     jobOk.ID,
	}
	jobOk.ResponseCh <- respOK

	// Tests job dispatched and context Done
	jobDone := types.NewJob(
		ctx,
		types.List,
		types.Host,
		api.GetComputeHostsParams{},
		handlers.HostURLParams{},
	)
	go func() {
		resp := h.DispatchAndWait(jobDone)
		assert.NotEqual(t, resp, nil)
	}()
	jobDone.Context.Done()

	// Tests job dispatched and response received with wrong job ID
	jobErr := types.NewJob(
		ctx,
		types.List,
		types.Host,
		api.GetComputeHostsParams{},
		handlers.HostURLParams{},
	)
	go func() {
		resp := h.DispatchAndWait(jobErr)
		assert.NotEqual(t, resp, nil)
	}()

	respErr := &types.Response{
		Payload: types.Payload{Data: api.ProblemDetails{
			Message: nil,
		}},
		Status: 0,
		ID:     jobOk.ID, // Wrong response ID
	}
	jobErr.ResponseCh <- respErr
}

//nolint:funlen // Testing only.
func TestNewManagerDelete(t *testing.T) {
	cfg := common.DefaultConfig()
	cfg.Inventory.Address = localhostAddress
	cfg.RestServer.Timeout = 1 * time.Second
	assert.NotEqual(t, cfg, nil)
	dispChan := make(chan bool, 1)
	termChan := make(chan bool, 1)
	wg := sync.WaitGroup{}
	disp := dispatcher.NewDispatcher(cfg, dispChan, termChan, &wg)
	assert.NotEqual(t, disp, nil)
	restMgrChan := make(chan bool, 1)
	man, err := server.NewManager(cfg, disp.JobQueue, restMgrChan, termChan, &wg)
	assert.NotEqual(t, man, nil)
	assert.NoError(t, err)

	h, err := server.NewHandlers(disp.JobQueue, cfg.RestServer.Timeout)
	require.NoError(t, err)
	require.NotNil(t, h)
	httpCtx := context.Background()

	r, err := http.NewRequestWithContext(httpCtx, http.MethodDelete, "test", http.NoBody)
	assert.NoError(t, err)
	w := testResponseWriter{name: "test"}

	ctx := echo.New().NewContext(r, w)

	assert.NotEqual(t, ctx, echo.Context(nil))
	err = h.DeleteComputeHostsHostID(ctx, "TestHostID")
	assert.NoError(t, err)

	err = h.DeleteRegionsRegionID(ctx, "TestRegion")
	assert.NoError(t, err)

	err = h.DeleteSitesSiteID(ctx, "TestSite")
	assert.NoError(t, err)

	err = h.DeleteOusOuID(ctx, "TestOU")
	assert.NoError(t, err)

	err = h.DeleteSchedulesSingleSingleScheduleID(ctx, "TestSingleSched")
	assert.NoError(t, err)

	err = h.DeleteSchedulesRepeatedRepeatedScheduleID(ctx, "TestRepeatedSched")
	assert.NoError(t, err)

	err = h.DeleteOSResourcesOSResourceID(ctx, "TestOS")
	assert.NoError(t, err)

	err = h.DeleteWorkloadsWorkloadID(ctx, "TestWorkload")
	assert.NoError(t, err)

	err = h.DeleteWorkloadMembersWorkloadMemberID(ctx, "TestWorkloadMember")
	assert.NoError(t, err)

	err = h.DeleteProvidersProviderID(ctx, "TestProvider")
	assert.NoError(t, err)

	err = h.DeleteInstancesInstanceID(ctx, "TestInstance-id")
	assert.NoError(t, err)

	err = h.DeleteTelemetryGroupsLogsTelemetryLogsGroupId(ctx, "TestTelemetrylogsGroupId")
	assert.NoError(t, err)

	err = h.DeleteTelemetryGroupsMetricsTelemetryMetricsGroupId(ctx, "TestTelemetryMetricsGroupId")
	assert.NoError(t, err)

	err = h.DeleteTelemetryProfilesLogsTelemetryLogsProfileId(ctx, "TelemetrylogsProfileId")
	assert.NoError(t, err)

	err = h.DeleteTelemetryProfilesMetricsTelemetryMetricsProfileId(ctx, "Test-metrics-telemetry-profile-id")
	assert.NoError(t, err)

	err = h.DeleteLocalAccountsLocalAccountID(ctx, "TestLocalAccountID")
	assert.NoError(t, err)
}

func TestNewManagerGetCompute(t *testing.T) {
	cfg := common.DefaultConfig()
	cfg.Inventory.Address = localhostAddress
	cfg.RestServer.Timeout = 1 * time.Second
	assert.NotEqual(t, cfg, nil)
	dispChan := make(chan bool, 1)
	termChan := make(chan bool, 1)
	wg := sync.WaitGroup{}
	disp := dispatcher.NewDispatcher(cfg, dispChan, termChan, &wg)
	assert.NotEqual(t, disp, nil)
	restMgrChan := make(chan bool)
	man, err := server.NewManager(cfg, disp.JobQueue, restMgrChan, termChan, &wg)
	assert.NotEqual(t, man, nil)
	assert.NoError(t, err)

	h, err := server.NewHandlers(disp.JobQueue, cfg.RestServer.Timeout)
	require.NoError(t, err)
	require.NotNil(t, h)
	httpCtx := context.Background()

	r, err := http.NewRequestWithContext(httpCtx, http.MethodDelete, "test", http.NoBody)
	assert.NoError(t, err)
	w := testResponseWriter{name: "test"}

	ctx := echo.New().NewContext(r, w)
	params := api.GetComputeParams{}
	err = h.GetCompute(ctx, params)
	assert.NoError(t, err)

	err = h.GetComputeHosts(ctx, api.GetComputeHostsParams{
		PageSize: &pgSize,
		Offset:   &pgOffset,
	})
	assert.NoError(t, err)

	err = h.GetComputeHostsHostID(ctx, "TestHostID")
	assert.NoError(t, err)

	summarySite := "site-12345678"
	err = h.GetComputeHostsSummary(ctx, api.GetComputeHostsSummaryParams{
		SiteID: &summarySite,
	})
	assert.NoError(t, err)
	err = h.GetInstances(ctx, api.GetInstancesParams{
		PageSize: &pgSize,
		Offset:   &pgOffset,
	})
	assert.NoError(t, err)

	err = h.GetInstancesInstanceID(ctx, "test-instance-id")
	assert.NoError(t, err)
}

func TestNewManagerGetLocations(t *testing.T) {
	cfg := common.DefaultConfig()
	cfg.Inventory.Address = localhostAddress
	cfg.RestServer.Timeout = 1 * time.Second
	assert.NotEqual(t, cfg, nil)
	dispChan := make(chan bool, 1)
	termChan := make(chan bool, 1)
	wg := sync.WaitGroup{}
	disp := dispatcher.NewDispatcher(cfg, dispChan, termChan, &wg)
	assert.NotEqual(t, disp, nil)
	restMgrChan := make(chan bool)
	man, err := server.NewManager(cfg, disp.JobQueue, restMgrChan, termChan, &wg)
	assert.NotEqual(t, man, nil)
	assert.NoError(t, err)

	h, err := server.NewHandlers(disp.JobQueue, cfg.RestServer.Timeout)
	require.NoError(t, err)
	require.NotNil(t, h)
	httpCtx := context.Background()

	r, err := http.NewRequestWithContext(httpCtx, http.MethodDelete, "test", http.NoBody)
	assert.NoError(t, err)
	w := testResponseWriter{name: "test"}

	ctx := echo.New().NewContext(r, w)
	err = h.GetRegions(ctx, api.GetRegionsParams{
		PageSize: &pgSize,
		Offset:   &pgOffset,
	})
	assert.NoError(t, err)

	err = h.GetRegionsRegionID(ctx, "TestRegionID")
	assert.NoError(t, err)

	err = h.GetSites(ctx, api.GetSitesParams{
		PageSize: &pgSize,
		Offset:   &pgOffset,
	})
	assert.NoError(t, err)

	err = h.GetSitesSiteID(ctx, "TestSiteID")
	assert.NoError(t, err)

	err = h.GetSites(ctx, api.GetSitesParams{
		PageSize: &pgSize,
		Offset:   &pgOffset,
	})
	assert.NoError(t, err)

	locationsName := "name"
	err = h.GetLocations(ctx, api.GetLocationsParams{
		ShowSites:   &showSites,
		ShowRegions: &showRegions,
		Name:        &locationsName,
	})
	assert.NoError(t, err)
}

func TestNewManagerGetOUandSchedandOS(t *testing.T) {
	cfg := common.DefaultConfig()
	cfg.Inventory.Address = localhostAddress
	cfg.RestServer.Timeout = 1 * time.Second
	assert.NotEqual(t, cfg, nil)
	dispChan := make(chan bool, 1)
	termChan := make(chan bool, 1)
	wg := sync.WaitGroup{}
	disp := dispatcher.NewDispatcher(cfg, dispChan, termChan, &wg)
	assert.NotEqual(t, disp, nil)
	restMgrChan := make(chan bool)
	man, err := server.NewManager(cfg, disp.JobQueue, restMgrChan, termChan, &wg)
	assert.NotEqual(t, man, nil)
	assert.NoError(t, err)

	h, err := server.NewHandlers(disp.JobQueue, cfg.RestServer.Timeout)
	require.NoError(t, err)
	require.NotNil(t, h)
	httpCtx := context.Background()

	r, err := http.NewRequestWithContext(httpCtx, http.MethodDelete, "test", http.NoBody)
	assert.NoError(t, err)
	w := testResponseWriter{name: "test"}

	ctx := echo.New().NewContext(r, w)

	err = h.GetOus(ctx, api.GetOusParams{
		PageSize: &pgSize,
		Offset:   &pgOffset,
	})
	assert.NoError(t, err)

	err = h.GetOusOuID(ctx, "TestGetOUID")
	assert.NoError(t, err)

	err = h.GetSchedulesRepeated(ctx, api.GetSchedulesRepeatedParams{
		PageSize: &pgSize,
		Offset:   &pgOffset,
	})
	assert.NoError(t, err)

	err = h.GetSchedulesSingle(ctx, api.GetSchedulesSingleParams{
		PageSize: &pgSize,
		Offset:   &pgOffset,
	})
	assert.NoError(t, err)

	err = h.GetSchedulesRepeatedRepeatedScheduleID(ctx, "TestGetSchedRepeatedID")
	assert.NoError(t, err)

	err = h.GetSchedulesSingleSingleScheduleID(ctx, "TestGetSchedSingleID")
	assert.NoError(t, err)

	err = h.GetOSResources(ctx, api.GetOSResourcesParams{
		PageSize: &pgSize,
		Offset:   &pgOffset,
	})
	assert.NoError(t, err)

	err = h.GetOSResourcesOSResourceID(ctx, "TestOS")
	assert.NoError(t, err)

	err = h.GetSchedules(ctx, api.GetSchedulesParams{
		PageSize: &pgSize,
		Offset:   &pgOffset,
	})
	assert.NoError(t, err)
}

func TestNewManagerGetWorkloadAndMember(t *testing.T) {
	cfg := common.DefaultConfig()
	cfg.Inventory.Address = localhostAddress
	cfg.RestServer.Timeout = 1 * time.Second
	assert.NotEqual(t, cfg, nil)
	dispChan := make(chan bool, 1)
	termChan := make(chan bool, 1)
	wg := sync.WaitGroup{}
	disp := dispatcher.NewDispatcher(cfg, dispChan, termChan, &wg)
	assert.NotEqual(t, disp, nil)
	restMgrChan := make(chan bool)
	man, err := server.NewManager(cfg, disp.JobQueue, restMgrChan, termChan, &wg)
	assert.NotEqual(t, man, nil)
	assert.NoError(t, err)

	h, err := server.NewHandlers(disp.JobQueue, cfg.RestServer.Timeout)
	require.NoError(t, err)
	require.NotNil(t, h)
	httpCtx := context.Background()

	r, err := http.NewRequestWithContext(httpCtx, http.MethodGet, "test", http.NoBody)
	assert.NoError(t, err)
	w := testResponseWriter{name: "test"}

	ctx := echo.New().NewContext(r, w)

	err = h.GetWorkloads(ctx, api.GetWorkloadsParams{
		PageSize: &pgSize,
		Offset:   &pgOffset,
	})
	assert.NoError(t, err)

	err = h.GetWorkloadsWorkloadID(ctx, "TestGetWorkload")
	assert.NoError(t, err)

	err = h.GetWorkloadMembers(ctx, api.GetWorkloadMembersParams{
		PageSize: &pgSize,
		Offset:   &pgOffset,
	})
	assert.NoError(t, err)

	err = h.GetWorkloadMembersWorkloadMemberID(ctx, "TestGetWorkloadMember")
	assert.NoError(t, err)
}

func TestNewManagerGetProviders(t *testing.T) {
	cfg := common.DefaultConfig()
	cfg.Inventory.Address = localhostAddress
	cfg.RestServer.Timeout = 1 * time.Second
	assert.NotEqual(t, cfg, nil)
	dispChan := make(chan bool, 1)
	termChan := make(chan bool, 1)
	wg := sync.WaitGroup{}
	disp := dispatcher.NewDispatcher(cfg, dispChan, termChan, &wg)
	assert.NotEqual(t, disp, nil)
	restMgrChan := make(chan bool)
	man, err := server.NewManager(cfg, disp.JobQueue, restMgrChan, termChan, &wg)
	assert.NotEqual(t, man, nil)
	assert.NoError(t, err)

	h, err := server.NewHandlers(disp.JobQueue, cfg.RestServer.Timeout)
	require.NoError(t, err)
	require.NotNil(t, h)
	httpCtx := context.Background()

	r, err := http.NewRequestWithContext(httpCtx, http.MethodGet, "test", http.NoBody)
	assert.Equal(t, err, nil)
	w := testResponseWriter{name: "test"}

	ctx := echo.New().NewContext(r, w)

	err = h.GetProviders(ctx, api.GetProvidersParams{
		PageSize: &pgSize,
		Offset:   &pgOffset,
	})
	assert.NoError(t, err)

	err = h.GetProvidersProviderID(ctx, "TestGetProvider")
	assert.NoError(t, err)
}

func TestNewManagerGetLocalAccounts(t *testing.T) {
	cfg := common.DefaultConfig()
	cfg.Inventory.Address = localhostAddress
	cfg.RestServer.Timeout = 1 * time.Second
	assert.NotEqual(t, cfg, nil)
	dispChan := make(chan bool, 1)
	termChan := make(chan bool, 1)
	wg := sync.WaitGroup{}
	disp := dispatcher.NewDispatcher(cfg, dispChan, termChan, &wg)
	assert.NotEqual(t, disp, nil)
	restMgrChan := make(chan bool)
	man, err := server.NewManager(cfg, disp.JobQueue, restMgrChan, termChan, &wg)
	assert.NotEqual(t, man, nil)
	assert.NoError(t, err)

	h, err := server.NewHandlers(disp.JobQueue, cfg.RestServer.Timeout)
	require.NoError(t, err)
	require.NotNil(t, h)
	httpCtx := context.Background()

	r, err := http.NewRequestWithContext(httpCtx, http.MethodGet, "test", http.NoBody)
	assert.Equal(t, err, nil)
	w := testResponseWriter{name: "test"}

	ctx := echo.New().NewContext(r, w)

	err = h.GetLocalAccounts(ctx, api.GetLocalAccountsParams{
		PageSize: &pgSize,
		Offset:   &pgOffset,
	})
	assert.NoError(t, err)

	err = h.GetLocalAccountsLocalAccountID(ctx, "TestGetLocalAccount")
	assert.NoError(t, err)
}

//nolint:funlen // it is a test
func TestNewManagerPost(t *testing.T) {
	cfg := common.DefaultConfig()
	cfg.Inventory.Address = localhostAddress
	cfg.RestServer.Timeout = 1 * time.Second
	assert.NotEqual(t, cfg, nil)
	dispChan := make(chan bool, 1)
	termChan := make(chan bool, 1)
	wg := sync.WaitGroup{}
	disp := dispatcher.NewDispatcher(cfg, dispChan, termChan, &wg)
	assert.NotEqual(t, disp, nil)
	restMgrChan := make(chan bool, 1)
	man, err := server.NewManager(cfg, disp.JobQueue, restMgrChan, termChan, &wg)
	assert.NotEqual(t, man, nil)
	assert.NoError(t, err)

	h, err := server.NewHandlers(disp.JobQueue, cfg.RestServer.Timeout)
	require.NoError(t, err)
	require.NotNil(t, h)
	httpCtx := context.Background()

	r, err := http.NewRequestWithContext(httpCtx, http.MethodPost, "test", http.NoBody)
	assert.NoError(t, err)
	w := testResponseWriter{name: "test"}

	ctx := echo.New().NewContext(r, w)
	err = h.PostComputeHosts(ctx)
	assert.NoError(t, err)

	err = h.PostRegions(ctx)
	assert.NoError(t, err)

	err = h.PostSites(ctx)
	assert.NoError(t, err)

	err = h.PostOus(ctx)
	assert.NoError(t, err)

	err = h.PostSchedulesSingle(ctx)
	assert.NoError(t, err)

	err = h.PostSchedulesRepeated(ctx)
	assert.NoError(t, err)

	err = h.PostOSResources(ctx)
	assert.NoError(t, err)

	err = h.PostWorkloads(ctx)
	assert.NoError(t, err)

	err = h.PostWorkloadMembers(ctx)
	assert.NoError(t, err)

	err = h.PostProviders(ctx)
	assert.NoError(t, err)

	err = h.PostTelemetryGroupsLogs(ctx)
	assert.NoError(t, err)

	err = h.PostTelemetryGroupsMetrics(ctx)
	assert.NoError(t, err)

	err = h.PostTelemetryProfilesMetrics(ctx)
	assert.NoError(t, err)

	err = h.PostTelemetryProfilesLogs(ctx)
	assert.NoError(t, err)

	err = h.PostInstances(ctx)
	assert.NoError(t, err)

	err = h.PostComputeHostsRegister(ctx)
	assert.NoError(t, err)

	err = h.PostLocalAccounts(ctx)
	assert.NoError(t, err)
}

func TestNewManagerPut(t *testing.T) {
	cfg := common.DefaultConfig()
	cfg.Inventory.Address = localhostAddress
	cfg.RestServer.Timeout = 1 * time.Second
	assert.NotEqual(t, cfg, nil)
	dispChan := make(chan bool, 1)
	termChan := make(chan bool, 1)
	wg := sync.WaitGroup{}
	disp := dispatcher.NewDispatcher(cfg, dispChan, termChan, &wg)
	assert.NotEqual(t, disp, nil)
	restMgrChan := make(chan bool, 1)
	man, err := server.NewManager(cfg, disp.JobQueue, restMgrChan, termChan, &wg)
	assert.NotEqual(t, man, nil)
	assert.NoError(t, err)

	h, err := server.NewHandlers(disp.JobQueue, cfg.RestServer.Timeout)
	require.NoError(t, err)
	require.NotNil(t, h)
	httpCtx := context.Background()

	r, err := http.NewRequestWithContext(httpCtx, http.MethodDelete, "test", http.NoBody)
	assert.NoError(t, err)
	w := testResponseWriter{name: "test"}

	ctx := echo.New().NewContext(r, w)
	err = h.PutComputeHostsHostID(ctx, "TestHostID")
	assert.NoError(t, err)

	err = h.PutRegionsRegionID(ctx, "TestRegionID")
	assert.NoError(t, err)

	err = h.PutSitesSiteID(ctx, "TestSiteID")
	assert.NoError(t, err)

	err = h.PutOusOuID(ctx, "TestOUID")
	assert.NoError(t, err)

	err = h.PutSchedulesSingleSingleScheduleID(ctx, "TestSingleSchedID")
	assert.NoError(t, err)

	err = h.PutSchedulesRepeatedRepeatedScheduleID(ctx, "TestRepeatedSchedID")
	assert.NoError(t, err)

	err = h.PutOSResourcesOSResourceID(ctx, "TestOS")
	assert.NoError(t, err)

	err = h.PutWorkloadsWorkloadID(ctx, "TestWorkload")
	assert.NoError(t, err)

	err = h.PutComputeHostsHostIDInvalidate(ctx, "TestHostIdInvalidate")
	assert.NoError(t, err)

	err = h.PutTelemetryProfilesMetricsTelemetryMetricsProfileId(ctx, "TestTelemetryMetricsProfileId")
	assert.NoError(t, err)
}

func TestNewManagerPatch(t *testing.T) {
	cfg := common.DefaultConfig()
	cfg.Inventory.Address = localhostAddress
	cfg.RestServer.Timeout = 1 * time.Second
	assert.NotEqual(t, cfg, nil)
	dispChan := make(chan bool, 1)
	termChan := make(chan bool, 1)
	wg := sync.WaitGroup{}
	disp := dispatcher.NewDispatcher(cfg, dispChan, termChan, &wg)
	assert.NotEqual(t, disp, nil)
	restMgrChan := make(chan bool, 1)
	man, err := server.NewManager(cfg, disp.JobQueue, restMgrChan, termChan, &wg)
	assert.NotEqual(t, man, nil)
	assert.NoError(t, err)

	h, err := server.NewHandlers(disp.JobQueue, cfg.RestServer.Timeout)
	require.NoError(t, err)
	require.NotNil(t, h)
	httpCtx := context.Background()

	r, err := http.NewRequestWithContext(httpCtx, http.MethodPatch, "test", http.NoBody)
	assert.NoError(t, err)
	w := testResponseWriter{name: "test"}

	ctx := echo.New().NewContext(r, w)

	assert.NotEqual(t, ctx, echo.Context(nil))
	err = h.PatchComputeHostsHostID(ctx, "TestHostID")
	assert.NoError(t, err)

	err = h.PatchRegionsRegionID(ctx, "TestRegion")
	assert.NoError(t, err)

	err = h.PatchSitesSiteID(ctx, "TestSite")
	assert.NoError(t, err)

	err = h.PatchOusOuID(ctx, "TestOU")
	assert.NoError(t, err)

	err = h.PatchSchedulesSingleSingleScheduleID(ctx, "TestSingleSched")
	assert.NoError(t, err)

	err = h.PatchSchedulesRepeatedRepeatedScheduleID(ctx, "TestRepeatedSched")
	assert.NoError(t, err)

	err = h.PatchOSResourcesOSResourceID(ctx, "TestOS")
	assert.NoError(t, err)

	err = h.PatchWorkloadsWorkloadID(ctx, "TestWorkload")
	assert.NoError(t, err)

	err = h.PatchInstancesInstanceID(ctx, "TestInstanceID")
	assert.NoError(t, err)

	err = h.PatchTelemetryProfilesLogsTelemetryLogsProfileId(ctx, "TestTelemetryLogsProfileId")
	assert.NoError(t, err)

	err = h.PatchTelemetryProfilesMetricsTelemetryMetricsProfileId(ctx, "TestMetricsTelemetryProfileId")
	assert.NoError(t, err)

	err = h.PatchComputeHostsHostIDRegister(ctx, "TestHostID")
	assert.NoError(t, err)

	err = h.PatchComputeHostsHostIDOnboard(ctx, "TestHostID")
	assert.NoError(t, err)
}

func TestNewManagerGetTelemetryGroupsLogs(t *testing.T) {
	cfg := common.DefaultConfig()
	cfg.Inventory.Address = localhostAddress
	cfg.RestServer.Timeout = 1 * time.Second
	assert.NotEqual(t, cfg, nil)
	dispChan := make(chan bool, 1)
	termChan := make(chan bool, 1)
	wg := sync.WaitGroup{}
	disp := dispatcher.NewDispatcher(cfg, dispChan, termChan, &wg)
	assert.NotEqual(t, disp, nil)
	restMgrChan := make(chan bool)
	man, err := server.NewManager(cfg, disp.JobQueue, restMgrChan, termChan, &wg)
	assert.NotEqual(t, man, nil)
	assert.NoError(t, err)

	h, err := server.NewHandlers(disp.JobQueue, cfg.RestServer.Timeout)
	require.NoError(t, err)
	require.NotNil(t, h)
	httpCtx := context.Background()

	r, err := http.NewRequestWithContext(httpCtx, http.MethodGet, "test", http.NoBody)
	assert.Equal(t, err, nil)
	w := testResponseWriter{name: "test"}

	ctx := echo.New().NewContext(r, w)

	err = h.GetTelemetryGroupsLogs(ctx, api.GetTelemetryGroupsLogsParams{
		PageSize: &pgSize,
		Offset:   &pgOffset,
	})
	assert.NoError(t, err)

	err = h.GetTelemetryProfilesMetrics(ctx, api.GetTelemetryProfilesMetricsParams{
		PageSize: &pgSize,
		Offset:   &pgOffset,
	})
	assert.NoError(t, err)

	err = h.GetTelemetryGroupsLogsTelemetryLogsGroupId(ctx, "test-telemetry-group-id")
	assert.NoError(t, err)
	err = h.GetTelemetryProfilesLogs(ctx, api.GetTelemetryProfilesLogsParams{
		PageSize: &pgSize,
		Offset:   &pgOffset,
	})
	assert.NoError(t, err)
	err = h.GetTelemetryGroupsMetrics(ctx, api.GetTelemetryGroupsMetricsParams{
		PageSize: &pgSize,
		Offset:   &pgOffset,
	})
	assert.NoError(t, err)
	err = h.GetTelemetryGroupsMetricsTelemetryMetricsGroupId(ctx, "test-telemetry-metrics-group-id")
	assert.NoError(t, err)
	err = h.GetTelemetryProfilesLogsTelemetryLogsProfileId(ctx, "test-telemetry-log-profile-id")
	assert.NoError(t, err)
	err = h.GetTelemetryProfilesMetricsTelemetryMetricsProfileId(ctx, "test-telemetry-metrics-profile-id")
	assert.NoError(t, err)
}

// generated from https://www.scottbrady91.com/tools/jwt - expires 12 Jul 2030.
//
//nolint:gosec // Sample Token used in Tests
const SampleGeneratedTokenRsa = `eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImtpZCI6IjEyOGEzOWVkYTU0MDU3YTIzN2M2NmZjODFhZmNmN
jI1In0.eyJpc3MiOiJodHRwOi8vZGV4OjMyMDAwIiwic3ViIjoiQ2lRd09HRTROamcwWWkxa1lqZzRMVFJpTnpNdE9UQmhPUzB6WTJReE5qWXhaalUwTmp
nU0JXeHZZMkZzIiwiYXVkIjoiZWRnZS1pYWFzIiwiZXhwIjoxOTA5OTM4NDQ3LCJpYXQiOjE1OTQ1Nzg0NDcsIm5vbmNlIjoiY3pSdk5raExVWGxYVUhOR
GFqbDJjR0pRYzFWTlJXdE9jREZrTG5CdGR5MWxSUzFZZDNGM1VFRjROSGRRIiwiYXRfaGFzaCI6Ill3MUNTNVJnb2FUVVlDVGVrcVJMQWciLCJlbWFpbCI
6ImVkZ2UtaWFhc0BleGFtcGxlLm9yZyIsImVtYWlsX3ZlcmlmaWVkIjp0cnVlLCJuYW1lIjoiZWRnZS1pYWFzIn0.UZKq6X7u7EWPwIGa0582bfAwxQfYy
LhHdLzYE93MHSx7hKEInZRlBkMMnY_uvgdzvnF7f8WawK8WNFizW3tlLc0NTv-FX-Dx5alHmDpu3Ht7qHifkkG2iZwQbZeQjEKiT1YDP5Sb_ZU5PPTxmaM
2R-spzZuvYrm05EhK-0edGT2ZZ7DaltUPPF07CS9meS0yP3Bh-HvWrgYmh3havW54x-BnB2LlvMl95WDbm7xQHRmFl9IH2RsfscfmEC7MuriqNaI6aev1Q
-x40dgydd2Wvre06ikYiF1CHhQNm1qkzYwT96ZD7Qot_HOFFDGUOAIGb94a2eWdxGc9jgBGK1-EJ0-6U6qYLjfxEKtJ7QXirUcRVDpaF_SmTWm2sgpSNtr
Bgvn1gENoy6mnC5oQeCCh7oeQBrSfBLSpmiG1MH2LplguXGt9_32aEAxNN8VaQR_mAQAVEF7l0cpeXCdPUAv1QVWYXNXTG1fUAt4a_0hojorgDZWrw8GqA
KzDolFruHiz`

func TestNewManagerPutAuthenticatedRsa(t *testing.T) {
	cfg := common.DefaultConfig()
	cfg.Inventory.Address = localhostAddress
	cfg.RestServer.Timeout = 1 * time.Second
	cfg.RestServer.Authentication = true
	assert.NotEqual(t, cfg, nil)
	dispChan := make(chan bool, 1)
	termChan := make(chan bool, 1)
	wg := sync.WaitGroup{}
	disp := dispatcher.NewDispatcher(cfg, dispChan, termChan, &wg)
	assert.NotEqual(t, disp, nil)
	restMgrChan := make(chan bool, 1)
	man, err := server.NewManager(cfg, disp.JobQueue, restMgrChan, termChan, &wg)
	assert.NotEqual(t, man, nil)
	assert.NoError(t, err)

	h, err := server.NewHandlers(disp.JobQueue, cfg.RestServer.Timeout)
	require.NoError(t, err)
	require.NotNil(t, h)
	httpCtx := context.Background()

	r, err := http.NewRequestWithContext(httpCtx, http.MethodDelete, "test", http.NoBody)
	assert.NoError(t, err)
	w := testResponseWriter{name: "test"}

	// adding JWT to the context
	r.Header.Add("authorization", "Bearer "+SampleGeneratedTokenRsa)

	ctx := echo.New().NewContext(r, w)
	err = h.PutComputeHostsHostID(ctx, "TestHostID")
	assert.NoError(t, err)

	err = h.PutRegionsRegionID(ctx, "TestRegionID")
	assert.NoError(t, err)

	err = h.PutSitesSiteID(ctx, "TestSiteID")
	assert.NoError(t, err)
}
