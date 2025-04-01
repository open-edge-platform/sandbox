// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	"github.com/open-edge-platform/infra-core/api/test/utils"
)

func clearIDs() {
	utils.Instance1Request.HostID = nil
	utils.Instance1Request.OsID = nil
	utils.Site1Request.Region = nil
	utils.Host1Request.Site = nil
}

func TestTelemetryGroup_CreateGetDelete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	allLogsGroups, err := apiClient.GetTelemetryGroupsLogsWithResponse(
		ctx,
		&api.GetTelemetryGroupsLogsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, allLogsGroups.StatusCode())
	for _, logsGroups := range *allLogsGroups.JSON200.TelemetryLogsGroups {
		DeleteTelemetryLogsGroup(t, context.Background(), apiClient, *logsGroups.TelemetryLogsGroupId)
	}
	res1 := CreateTelemetryLogsGroup(t, ctx, apiClient, utils.TelemetryLogsGroup1Request)

	allMetricsGroups, err := apiClient.GetTelemetryGroupsMetricsWithResponse(
		ctx,
		&api.GetTelemetryGroupsMetricsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, allMetricsGroups.StatusCode())
	for _, metricsGroups := range *allMetricsGroups.JSON200.TelemetryMetricsGroups {
		DeleteTelemetryMetricsGroup(t, context.Background(), apiClient, *metricsGroups.TelemetryMetricsGroupId)
	}
	res2 := CreateTelemetryMetricsGroup(t, ctx, apiClient, utils.TelemetryMetricsGroup1Request)

	// Assert presence of telemetry resources
	allLogsGroups, err = apiClient.GetTelemetryGroupsLogsWithResponse(
		ctx,
		&api.GetTelemetryGroupsLogsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	fmt.Println(allLogsGroups.JSON200.TelemetryLogsGroups)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, allLogsGroups.StatusCode())
	assert.Len(t, *allLogsGroups.JSON200.TelemetryLogsGroups, 1)

	allMetricsGroups, err = apiClient.GetTelemetryGroupsMetricsWithResponse(
		ctx,
		&api.GetTelemetryGroupsMetricsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, allMetricsGroups.StatusCode())
	assert.Len(t, *allMetricsGroups.JSON200.TelemetryMetricsGroups, 1)

	logsGroup, err := apiClient.GetTelemetryGroupsLogsTelemetryLogsGroupIdWithResponse(
		ctx,
		*res1.JSON201.TelemetryLogsGroupId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, logsGroup.StatusCode())
	assert.Equal(t, res1.JSON201.Name, logsGroup.JSON200.Name)
	assert.Equal(t, res1.JSON201.Groups, logsGroup.JSON200.Groups)
	assert.Equal(t, res1.JSON201.CollectorKind, logsGroup.JSON200.CollectorKind)

	metricsGroup, err := apiClient.GetTelemetryGroupsMetricsTelemetryMetricsGroupIdWithResponse(
		ctx,
		*res2.JSON201.TelemetryMetricsGroupId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, metricsGroup.StatusCode())
	assert.Equal(t, res2.JSON201.Name, metricsGroup.JSON200.Name)
	assert.Equal(t, res2.JSON201.Groups, metricsGroup.JSON200.Groups)
	assert.Equal(t, res2.JSON201.CollectorKind, metricsGroup.JSON200.CollectorKind)

	// delete with auto-cleanup
}

func TestTelemetryLogsGroup_PostErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	testCases := map[string]struct {
		in                 api.TelemetryLogsGroup
		expectedHTTPStatus int
		valid              bool
	}{
		"Post_NoName_Status_BadRequest": {
			in: api.TelemetryLogsGroup{
				CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
				Groups:        []string{"test group"},
			},
			expectedHTTPStatus: http.StatusBadRequest,
		},
		"Post_NoCollectorKind_Status_BadRequest": {
			in: api.TelemetryLogsGroup{
				Name:   "Test Name",
				Groups: []string{"test group"},
			},
			expectedHTTPStatus: http.StatusBadRequest,
		},
		"Post_NoGroups_Status_BadRequest": {
			in: api.TelemetryLogsGroup{
				Name:          "Test Name",
				CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
			},
			expectedHTTPStatus: http.StatusBadRequest,
		},
	}

	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			resp, reqErr := apiClient.PostTelemetryGroupsLogsWithResponse(
				ctx,
				tc.in,
				AddJWTtoTheHeader,
				AddProjectIDtoTheHeader,
			)
			require.NoError(t, reqErr)
			assert.Equal(t, tc.expectedHTTPStatus, resp.StatusCode())
			fmt.Println(*resp.JSON400.Message)
		})
	}
}

func TestTelemetryMetricsGroup_PostErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	testCases := map[string]struct {
		in                 api.TelemetryMetricsGroup
		expectedHTTPStatus int
		valid              bool
	}{
		"Post_NoName_Status_BadRequest": {
			in: api.TelemetryMetricsGroup{
				CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
				Groups:        []string{"test group"},
			},
			expectedHTTPStatus: http.StatusBadRequest,
		},
		"Post_NoCollectorKind_Status_BadRequest": {
			in: api.TelemetryMetricsGroup{
				Name:   "Test Name",
				Groups: []string{"test group"},
			},
			expectedHTTPStatus: http.StatusBadRequest,
		},
		"Post_NoGroups_Status_BadRequest": {
			in: api.TelemetryMetricsGroup{
				Name:          "Test Name",
				CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
			},
			expectedHTTPStatus: http.StatusBadRequest,
		},
	}

	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			resp, reqErr := apiClient.PostTelemetryGroupsMetricsWithResponse(
				ctx,
				tc.in,
				AddJWTtoTheHeader,
				AddProjectIDtoTheHeader,
			)
			require.NoError(t, reqErr)
			assert.Equal(t, tc.expectedHTTPStatus, resp.StatusCode())
		})
	}
}

func TestTelemetryGroup_GetDeleteErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	testCases := map[string]struct {
		ID                 string
		expectedHTTPStatus int
		valid              bool
	}{
		"UnexistingID_Status_NotFound": {
			ID:                 "telemetrygroup-00000000",
			expectedHTTPStatus: http.StatusNotFound,
		},
		"InvalidID_Status_BadRequest": {
			ID:                 "telemetrygroup-XXXXXX",
			expectedHTTPStatus: http.StatusBadRequest,
		},
	}

	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			resp1, reqErr := apiClient.GetTelemetryGroupsLogsTelemetryLogsGroupIdWithResponse(
				ctx,
				tc.ID,
				AddJWTtoTheHeader,
				AddProjectIDtoTheHeader,
			)
			require.NoError(t, reqErr)
			assert.Equal(t, tc.expectedHTTPStatus, resp1.StatusCode())

			resp2, reqErr := apiClient.GetTelemetryGroupsMetricsTelemetryMetricsGroupIdWithResponse(
				ctx,
				tc.ID,
				AddJWTtoTheHeader,
				AddProjectIDtoTheHeader,
			)
			require.NoError(t, reqErr)
			assert.Equal(t, tc.expectedHTTPStatus, resp2.StatusCode())

			respDel1, reqErr := apiClient.DeleteTelemetryGroupsLogsTelemetryLogsGroupIdWithResponse(
				ctx,
				tc.ID,
				AddJWTtoTheHeader,
				AddProjectIDtoTheHeader,
			)
			require.NoError(t, reqErr)
			assert.Equal(t, tc.expectedHTTPStatus, respDel1.StatusCode())

			respDel2, reqErr := apiClient.DeleteTelemetryGroupsMetricsTelemetryMetricsGroupIdWithResponse(
				ctx,
				tc.ID,
				AddJWTtoTheHeader,
				AddProjectIDtoTheHeader,
			)
			require.NoError(t, reqErr)
			assert.Equal(t, tc.expectedHTTPStatus, respDel2.StatusCode())
		})
	}
}

func TestTelemetryProfile_CreateGetDelete(t *testing.T) {
	defer clearIDs()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	r1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	utils.Site1Request.RegionId = r1.JSON201.RegionID
	site1 := CreateSite(t, ctx, apiClient, utils.Site1Request)
	utils.Host1Request.SiteId = site1.JSON201.SiteID
	hostCreated1 := CreateHost(t, ctx, apiClient, utils.Host1Request)
	osCreated1 := CreateOS(t, ctx, apiClient, utils.OSResource1Request)
	utils.Instance1Request.HostID = hostCreated1.JSON201.ResourceId
	utils.Instance1Request.OsID = osCreated1.JSON201.OsResourceID
	inst1 := CreateInstance(t, ctx, apiClient, utils.Instance1Request)

	telemetryGroupMetrics1 := utils.TelemetryMetricsGroup1Request
	telemetryGroupMetrics2 := api.TelemetryMetricsGroup{
		Name:          "CPU Usage",
		CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
		Groups: []string{
			"cpu",
		},
	}

	logsGroup := CreateTelemetryLogsGroup(t, ctx, apiClient, utils.TelemetryLogsGroup1Request)
	metricsGroup1 := CreateTelemetryMetricsGroup(t, ctx, apiClient, telemetryGroupMetrics1)
	metricsGroup2 := CreateTelemetryMetricsGroup(t, ctx, apiClient, telemetryGroupMetrics2)

	TelemetryLogsProfilePerInstance := api.TelemetryLogsProfile{
		LogLevel:       api.TELEMETRYSEVERITYLEVELDEBUG,
		TargetInstance: inst1.JSON201.InstanceID,
		LogsGroupId:    *logsGroup.JSON201.TelemetryLogsGroupId,
	}
	TelemetryMetricsProfilePerSite := api.TelemetryMetricsProfile{
		MetricsInterval: 300,
		TargetSite:      site1.JSON201.SiteID,
		MetricsGroupId:  *metricsGroup1.JSON201.TelemetryMetricsGroupId,
	}
	TelemetryMetricsProfilePerRegion := api.TelemetryMetricsProfile{
		MetricsInterval: 300,
		TargetRegion:    r1.JSON201.RegionID,
		MetricsGroupId:  *metricsGroup2.JSON201.TelemetryMetricsGroupId,
	}

	res1 := CreateTelemetryLogsProfile(t, ctx, apiClient, TelemetryLogsProfilePerInstance)
	res1.JSON201.LogsGroup = logsGroup.JSON201
	res2 := CreateTelemetryMetricsProfile(t, ctx, apiClient, TelemetryMetricsProfilePerSite)
	res2.JSON201.MetricsGroup = metricsGroup1.JSON201
	res3 := CreateTelemetryMetricsProfile(t, ctx, apiClient, TelemetryMetricsProfilePerRegion)
	res3.JSON201.MetricsGroup = metricsGroup2.JSON201

	// Assert presence of telemetry resources
	allLogsProfiles, err := apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, allLogsProfiles.StatusCode())
	assert.Len(t, *allLogsProfiles.JSON200.TelemetryLogsProfiles, 1)

	allMetricsProfiles, err := apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, allMetricsProfiles.StatusCode())
	assert.Len(t, *allMetricsProfiles.JSON200.TelemetryMetricsProfiles, 2)

	res, err := apiClient.GetTelemetryProfilesLogsTelemetryLogsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.Equal(t, res1.JSON201.ProfileId, res.JSON200.ProfileId)
	assert.Equal(t, res1.JSON201.TargetInstance, res.JSON200.TargetInstance)
	assert.Equal(t, res1.JSON201.TargetSite, res.JSON200.TargetSite)
	assert.Equal(t, res1.JSON201.TargetRegion, res.JSON200.TargetRegion)
	assert.Equal(t, res1.JSON201.LogsGroupId, res.JSON200.LogsGroupId)
	assert.Equal(t, res1.JSON201.LogsGroup.TelemetryLogsGroupId, res.JSON200.LogsGroup.TelemetryLogsGroupId)
	assert.Equal(t, res1.JSON201.LogsGroup.Name, res.JSON200.LogsGroup.Name)
	assert.Equal(t, res1.JSON201.LogLevel, res.JSON200.LogLevel)

	for _, profile := range []*api.TelemetryMetricsProfile{res2.JSON201, res3.JSON201} {
		resp, err := apiClient.GetTelemetryProfilesMetricsTelemetryMetricsProfileIdWithResponse(
			ctx,
			*profile.ProfileId,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode())

		assert.Equal(t, profile.ProfileId, resp.JSON200.ProfileId)
		assert.Equal(t, res1.JSON201.TargetInstance, res.JSON200.TargetInstance)
		assert.Equal(t, res1.JSON201.TargetSite, res.JSON200.TargetSite)
		assert.Equal(t, res1.JSON201.TargetRegion, res.JSON200.TargetRegion)
		assert.Equal(t, profile.MetricsGroupId, resp.JSON200.MetricsGroupId)
		assert.Equal(t, profile.MetricsGroup.TelemetryMetricsGroupId, resp.JSON200.MetricsGroup.TelemetryMetricsGroupId)
		assert.Equal(t, profile.MetricsGroup.Name, resp.JSON200.MetricsGroup.Name)
		assert.Equal(t, profile.MetricsInterval, resp.JSON200.MetricsInterval)
	}
}

func TestTelemetryLogsProfile_UpdatePUT(t *testing.T) {
	defer clearIDs()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	logsGroup1 := CreateTelemetryLogsGroup(t, ctx, apiClient, utils.TelemetryLogsGroup1Request)
	logsGroup2 := CreateTelemetryLogsGroup(t, ctx, apiClient, api.TelemetryLogsGroup{
		Name:          "Kernel logs",
		CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
		Groups: []string{
			"kern",
		},
	})

	regionCreated1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	utils.Site1Request.RegionId = nil
	siteCreated1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	TelemetryLogsProfile := api.TelemetryLogsProfile{
		LogLevel:    api.TELEMETRYSEVERITYLEVELDEBUG,
		TargetSite:  siteCreated1.JSON201.SiteID,
		LogsGroupId: *logsGroup1.JSON201.TelemetryLogsGroupId,
	}
	res1 := CreateTelemetryLogsProfile(t, ctx, apiClient, TelemetryLogsProfile)
	res1.JSON201.LogsGroup = logsGroup1.JSON201

	// Assert presence of the telemetry profile
	TelemetryProfile1Get, err := apiClient.GetTelemetryProfilesLogsTelemetryLogsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, TelemetryProfile1Get.StatusCode())
	assert.Equal(t, TelemetryLogsProfile.LogLevel, TelemetryProfile1Get.JSON200.LogLevel)

	// re-assign telemetry profile from Site to Region
	TelemetryLogsProfile.TargetSite = &emptyString
	TelemetryLogsProfile.TargetRegion = regionCreated1.JSON201.RegionID
	telemetryLogsProfile1Update, err := apiClient.PutTelemetryProfilesLogsTelemetryLogsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryLogsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, telemetryLogsProfile1Update.StatusCode())
	assert.Equal(t, *TelemetryLogsProfile.TargetRegion, *telemetryLogsProfile1Update.JSON200.TargetRegion)

	TelemetryProfile1Get, err = apiClient.GetTelemetryProfilesLogsTelemetryLogsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, TelemetryProfile1Get.StatusCode())
	assert.Equal(t, TelemetryLogsProfile.LogLevel, TelemetryProfile1Get.JSON200.LogLevel)
	assert.Equal(t, TelemetryLogsProfile.LogsGroupId, TelemetryProfile1Get.JSON200.LogsGroupId)
	assert.Empty(t, TelemetryProfile1Get.JSON200.TargetSite)
	assert.Equal(t, *regionCreated1.JSON201.RegionID, *TelemetryProfile1Get.JSON200.TargetRegion)

	// change log level
	TelemetryLogsProfile.LogLevel = api.TELEMETRYSEVERITYLEVELINFO
	telemetryLogsProfile1Update, err = apiClient.PutTelemetryProfilesLogsTelemetryLogsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryLogsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, telemetryLogsProfile1Update.StatusCode())
	assert.Equal(t, api.TELEMETRYSEVERITYLEVELINFO, telemetryLogsProfile1Update.JSON200.LogLevel)

	TelemetryProfile1Get, err = apiClient.GetTelemetryProfilesLogsTelemetryLogsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, TelemetryProfile1Get.StatusCode())
	assert.Equal(t, api.TELEMETRYSEVERITYLEVELINFO, telemetryLogsProfile1Update.JSON200.LogLevel)

	// change the telemetry group
	TelemetryLogsProfile.LogsGroupId = *logsGroup2.JSON201.TelemetryLogsGroupId
	telemetryLogsProfile1Update, err = apiClient.PutTelemetryProfilesLogsTelemetryLogsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryLogsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, telemetryLogsProfile1Update.StatusCode())
	assert.Equal(t, *logsGroup2.JSON201.TelemetryLogsGroupId, telemetryLogsProfile1Update.JSON200.LogsGroupId)

	TelemetryProfile1Get, err = apiClient.GetTelemetryProfilesLogsTelemetryLogsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, TelemetryProfile1Get.StatusCode())
	assert.Equal(t, *logsGroup2.JSON201.TelemetryLogsGroupId, telemetryLogsProfile1Update.JSON200.LogsGroupId)

	// PUT with empty target relation
	TelemetryLogsProfile.TargetRegion = &emptyString
	telemetryLogsProfile1Update, err = apiClient.PutTelemetryProfilesLogsTelemetryLogsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryLogsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, telemetryLogsProfile1Update.StatusCode())

	// update to wrong type of telemetry group (logs profile cannot be associated with metrics group)
	metricsGroup := CreateTelemetryMetricsGroup(t, ctx, apiClient, utils.TelemetryMetricsGroup1Request)
	TelemetryLogsProfile.TargetRegion = regionCreated1.JSON201.RegionID
	TelemetryLogsProfile.LogsGroupId = *metricsGroup.JSON201.TelemetryMetricsGroupId
	telemetryLogsProfile1Update, err = apiClient.PutTelemetryProfilesLogsTelemetryLogsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryLogsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, telemetryLogsProfile1Update.StatusCode())
}

func TestTelemetryLogsProfile_UpdatePATCH(t *testing.T) {
	defer clearIDs()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	logsGroup1 := CreateTelemetryLogsGroup(t, ctx, apiClient, utils.TelemetryLogsGroup1Request)
	logsGroup2 := CreateTelemetryLogsGroup(t, ctx, apiClient, api.TelemetryLogsGroup{
		Name:          "Kernel logs",
		CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
		Groups: []string{
			"kern",
		},
	})

	siteCreated1 := CreateSite(t, ctx, apiClient, utils.Site1Request)
	regionCreated1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)

	TelemetryLogsProfile := api.TelemetryLogsProfile{
		LogLevel:    api.TELEMETRYSEVERITYLEVELDEBUG,
		TargetSite:  siteCreated1.JSON201.SiteID,
		LogsGroupId: *logsGroup1.JSON201.TelemetryLogsGroupId,
	}
	res1 := CreateTelemetryLogsProfile(t, ctx, apiClient, TelemetryLogsProfile)
	res1.JSON201.LogsGroup = logsGroup1.JSON201

	// Assert presence of the telemetry profile
	TelemetryProfile1Get, err := apiClient.GetTelemetryProfilesLogsTelemetryLogsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, TelemetryProfile1Get.StatusCode())
	assert.Equal(t, TelemetryLogsProfile.LogLevel, TelemetryProfile1Get.JSON200.LogLevel)

	// re-assign telemetry profile from Site to Region
	TelemetryLogsProfile.TargetSite = &emptyString
	TelemetryLogsProfile.TargetRegion = regionCreated1.JSON201.RegionID
	telemetryLogsProfile1Update, err := apiClient.PatchTelemetryProfilesLogsTelemetryLogsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryLogsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, telemetryLogsProfile1Update.StatusCode())
	assert.Equal(t, *TelemetryLogsProfile.TargetRegion, *telemetryLogsProfile1Update.JSON200.TargetRegion)

	TelemetryProfile1Get, err = apiClient.GetTelemetryProfilesLogsTelemetryLogsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, TelemetryProfile1Get.StatusCode())
	assert.Equal(t, TelemetryLogsProfile.LogLevel, TelemetryProfile1Get.JSON200.LogLevel)
	assert.Equal(t, TelemetryLogsProfile.LogsGroupId, TelemetryProfile1Get.JSON200.LogsGroupId)
	assert.Empty(t, TelemetryProfile1Get.JSON200.TargetSite)
	assert.Equal(t, *regionCreated1.JSON201.RegionID, *TelemetryProfile1Get.JSON200.TargetRegion)

	// change log level
	TelemetryLogsProfile.LogLevel = api.TELEMETRYSEVERITYLEVELINFO
	telemetryLogsProfile1Update, err = apiClient.PatchTelemetryProfilesLogsTelemetryLogsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryLogsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, telemetryLogsProfile1Update.StatusCode())
	assert.Equal(t, api.TELEMETRYSEVERITYLEVELINFO, telemetryLogsProfile1Update.JSON200.LogLevel)

	TelemetryProfile1Get, err = apiClient.GetTelemetryProfilesLogsTelemetryLogsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, TelemetryProfile1Get.StatusCode())
	assert.Equal(t, api.TELEMETRYSEVERITYLEVELINFO, telemetryLogsProfile1Update.JSON200.LogLevel)

	// change the telemetry group
	TelemetryLogsProfile.LogsGroupId = *logsGroup2.JSON201.TelemetryLogsGroupId
	telemetryLogsProfile1Update, err = apiClient.PatchTelemetryProfilesLogsTelemetryLogsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryLogsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, telemetryLogsProfile1Update.StatusCode())
	assert.Equal(t, *logsGroup2.JSON201.TelemetryLogsGroupId, telemetryLogsProfile1Update.JSON200.LogsGroupId)

	TelemetryProfile1Get, err = apiClient.GetTelemetryProfilesLogsTelemetryLogsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, TelemetryProfile1Get.StatusCode())
	assert.Equal(t, *logsGroup2.JSON201.TelemetryLogsGroupId, telemetryLogsProfile1Update.JSON200.LogsGroupId)

	// PUT with empty target relation
	TelemetryLogsProfile.TargetRegion = &emptyString
	telemetryLogsProfile1Update, err = apiClient.PatchTelemetryProfilesLogsTelemetryLogsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryLogsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, telemetryLogsProfile1Update.StatusCode())

	// update to wrong type of telemetry group (logs profile cannot be associated with metrics group)
	metricsGroup := CreateTelemetryMetricsGroup(t, ctx, apiClient, utils.TelemetryMetricsGroup1Request)
	TelemetryLogsProfile.TargetRegion = regionCreated1.JSON201.RegionID
	TelemetryLogsProfile.LogsGroupId = *metricsGroup.JSON201.TelemetryMetricsGroupId
	telemetryLogsProfile1Update, err = apiClient.PatchTelemetryProfilesLogsTelemetryLogsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryLogsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, telemetryLogsProfile1Update.StatusCode())
}

func TestTelemetryMetricsProfile_UpdatePUT(t *testing.T) {
	defer clearIDs()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	metricsGroup1 := CreateTelemetryMetricsGroup(t, ctx, apiClient, utils.TelemetryMetricsGroup1Request)
	metricsGroup2 := CreateTelemetryMetricsGroup(t, ctx, apiClient, api.TelemetryMetricsGroup{
		Name:          "NW Usage",
		CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
		Groups: []string{
			"net",
		},
	})

	siteCreated1 := CreateSite(t, ctx, apiClient, utils.Site1Request)
	regionCreated1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)

	TelemetryMetricsProfile := api.TelemetryMetricsProfile{
		MetricsInterval: 300,
		TargetSite:      siteCreated1.JSON201.SiteID,
		MetricsGroupId:  *metricsGroup1.JSON201.TelemetryMetricsGroupId,
	}
	res1 := CreateTelemetryMetricsProfile(t, ctx, apiClient, TelemetryMetricsProfile)
	res1.JSON201.MetricsGroup = metricsGroup1.JSON201

	// Assert presence of the telemetry profile
	TelemetryProfile1Get, err := apiClient.GetTelemetryProfilesMetricsTelemetryMetricsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, TelemetryProfile1Get.StatusCode())
	assert.Equal(t, TelemetryMetricsProfile.MetricsInterval, TelemetryProfile1Get.JSON200.MetricsInterval)

	// re-assign telemetry profile from Site to Region
	TelemetryMetricsProfile.TargetSite = &emptyString
	TelemetryMetricsProfile.TargetRegion = regionCreated1.JSON201.RegionID
	telemetryMetricsProfile1Update, err := apiClient.PutTelemetryProfilesMetricsTelemetryMetricsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryMetricsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, telemetryMetricsProfile1Update.StatusCode())
	assert.Equal(t, *TelemetryMetricsProfile.TargetRegion, *telemetryMetricsProfile1Update.JSON200.TargetRegion)

	TelemetryProfile1Get, err = apiClient.GetTelemetryProfilesMetricsTelemetryMetricsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, TelemetryProfile1Get.StatusCode())
	assert.Equal(t, TelemetryMetricsProfile.MetricsInterval, TelemetryProfile1Get.JSON200.MetricsInterval)
	assert.Equal(t, TelemetryMetricsProfile.MetricsGroupId, TelemetryProfile1Get.JSON200.MetricsGroupId)
	assert.Empty(t, TelemetryProfile1Get.JSON200.TargetSite)
	assert.Equal(t, *regionCreated1.JSON201.RegionID, *TelemetryProfile1Get.JSON200.TargetRegion)

	// change log level
	TelemetryMetricsProfile.MetricsInterval = 5
	telemetryMetricsProfile1Update, err = apiClient.PutTelemetryProfilesMetricsTelemetryMetricsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryMetricsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, telemetryMetricsProfile1Update.StatusCode())
	assert.Equal(t, 5, telemetryMetricsProfile1Update.JSON200.MetricsInterval)

	TelemetryProfile1Get, err = apiClient.GetTelemetryProfilesMetricsTelemetryMetricsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, TelemetryProfile1Get.StatusCode())
	assert.Equal(t, 5, telemetryMetricsProfile1Update.JSON200.MetricsInterval)

	// change the telemetry group
	TelemetryMetricsProfile.MetricsGroupId = *metricsGroup2.JSON201.TelemetryMetricsGroupId
	telemetryMetricsProfile1Update, err = apiClient.PutTelemetryProfilesMetricsTelemetryMetricsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryMetricsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, telemetryMetricsProfile1Update.StatusCode())
	assert.Equal(t, *metricsGroup2.JSON201.TelemetryMetricsGroupId, telemetryMetricsProfile1Update.JSON200.MetricsGroupId)

	TelemetryProfile1Get, err = apiClient.GetTelemetryProfilesMetricsTelemetryMetricsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, TelemetryProfile1Get.StatusCode())
	assert.Equal(t, *metricsGroup2.JSON201.TelemetryMetricsGroupId, telemetryMetricsProfile1Update.JSON200.MetricsGroupId)

	// PUT with empty target relation
	TelemetryMetricsProfile.TargetRegion = &emptyString
	telemetryMetricsProfile1Update, err = apiClient.PutTelemetryProfilesMetricsTelemetryMetricsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryMetricsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, telemetryMetricsProfile1Update.StatusCode())

	// update to wrong type of telemetry group (logs profile cannot be associated with metrics group)
	logsGroup := CreateTelemetryLogsGroup(t, ctx, apiClient, utils.TelemetryLogsGroup1Request)
	TelemetryMetricsProfile.TargetRegion = regionCreated1.JSON201.RegionID
	TelemetryMetricsProfile.MetricsGroupId = *logsGroup.JSON201.TelemetryLogsGroupId
	telemetryMetricsProfile1Update, err = apiClient.PutTelemetryProfilesMetricsTelemetryMetricsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryMetricsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, telemetryMetricsProfile1Update.StatusCode())
}

func TestTelemetryMetricsProfile_UpdatePATCH(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	metricsGroup1 := CreateTelemetryMetricsGroup(t, ctx, apiClient, utils.TelemetryMetricsGroup1Request)
	metricsGroup2 := CreateTelemetryMetricsGroup(t, ctx, apiClient, api.TelemetryMetricsGroup{
		Name:          "NW Usage",
		CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
		Groups: []string{
			"net",
		},
	})

	siteCreated1 := CreateSite(t, ctx, apiClient, utils.Site1Request)
	regionCreated1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	defer clearIDs()

	TelemetryMetricsProfile := api.TelemetryMetricsProfile{
		MetricsInterval: 300,
		TargetSite:      siteCreated1.JSON201.SiteID,
		MetricsGroupId:  *metricsGroup1.JSON201.TelemetryMetricsGroupId,
	}
	res1 := CreateTelemetryMetricsProfile(t, ctx, apiClient, TelemetryMetricsProfile)
	res1.JSON201.MetricsGroup = metricsGroup1.JSON201

	// Assert presence of the telemetry profile
	TelemetryProfile1Get, err := apiClient.GetTelemetryProfilesMetricsTelemetryMetricsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, TelemetryProfile1Get.StatusCode())
	assert.Equal(t, TelemetryMetricsProfile.MetricsInterval, TelemetryProfile1Get.JSON200.MetricsInterval)

	// re-assign telemetry profile from Site to Region
	TelemetryMetricsProfile.TargetSite = &emptyString
	TelemetryMetricsProfile.TargetRegion = regionCreated1.JSON201.RegionID
	telemetryMetricsProfile1Update, err := apiClient.PatchTelemetryProfilesMetricsTelemetryMetricsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryMetricsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, telemetryMetricsProfile1Update.StatusCode())
	assert.Equal(t, *TelemetryMetricsProfile.TargetRegion, *telemetryMetricsProfile1Update.JSON200.TargetRegion)

	TelemetryProfile1Get, err = apiClient.GetTelemetryProfilesMetricsTelemetryMetricsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, TelemetryProfile1Get.StatusCode())
	assert.Equal(t, TelemetryMetricsProfile.MetricsInterval, TelemetryProfile1Get.JSON200.MetricsInterval)
	assert.Equal(t, TelemetryMetricsProfile.MetricsGroupId, TelemetryProfile1Get.JSON200.MetricsGroupId)
	assert.Empty(t, TelemetryProfile1Get.JSON200.TargetSite)
	assert.Equal(t, *regionCreated1.JSON201.RegionID, *TelemetryProfile1Get.JSON200.TargetRegion)

	// change log level
	TelemetryMetricsProfile.MetricsInterval = 5
	telemetryMetricsProfile1Update, err = apiClient.PatchTelemetryProfilesMetricsTelemetryMetricsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryMetricsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, telemetryMetricsProfile1Update.StatusCode())
	assert.Equal(t, 5, telemetryMetricsProfile1Update.JSON200.MetricsInterval)

	TelemetryProfile1Get, err = apiClient.GetTelemetryProfilesMetricsTelemetryMetricsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, TelemetryProfile1Get.StatusCode())
	assert.Equal(t, 5, telemetryMetricsProfile1Update.JSON200.MetricsInterval)

	// change the telemetry group
	TelemetryMetricsProfile.MetricsGroupId = *metricsGroup2.JSON201.TelemetryMetricsGroupId
	telemetryMetricsProfile1Update, err = apiClient.PatchTelemetryProfilesMetricsTelemetryMetricsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryMetricsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, telemetryMetricsProfile1Update.StatusCode())
	assert.Equal(t, *metricsGroup2.JSON201.TelemetryMetricsGroupId, telemetryMetricsProfile1Update.JSON200.MetricsGroupId)

	TelemetryProfile1Get, err = apiClient.GetTelemetryProfilesMetricsTelemetryMetricsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, TelemetryProfile1Get.StatusCode())
	assert.Equal(t, *metricsGroup2.JSON201.TelemetryMetricsGroupId, telemetryMetricsProfile1Update.JSON200.MetricsGroupId)

	// PUT with empty target relation
	TelemetryMetricsProfile.TargetRegion = &emptyString
	telemetryMetricsProfile1Update, err = apiClient.PatchTelemetryProfilesMetricsTelemetryMetricsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryMetricsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, telemetryMetricsProfile1Update.StatusCode())

	// update to wrong type of telemetry group (logs profile cannot be associated with metrics group)
	logsGroup := CreateTelemetryLogsGroup(t, ctx, apiClient, utils.TelemetryLogsGroup1Request)
	TelemetryMetricsProfile.TargetRegion = regionCreated1.JSON201.RegionID
	TelemetryMetricsProfile.MetricsGroupId = *logsGroup.JSON201.TelemetryLogsGroupId
	telemetryMetricsProfile1Update, err = apiClient.PatchTelemetryProfilesMetricsTelemetryMetricsProfileIdWithResponse(
		ctx,
		*res1.JSON201.ProfileId,
		TelemetryMetricsProfile,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, telemetryMetricsProfile1Update.StatusCode())
}

func TestTelemetryGroupList_ListEmpty(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	resList1, err := apiClient.GetTelemetryGroupsLogsWithResponse(
		ctx,
		&api.GetTelemetryGroupsLogsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList1.StatusCode())
	assert.Empty(t, resList1.JSON200.TelemetryLogsGroups)

	resList2, err := apiClient.GetTelemetryGroupsMetricsWithResponse(
		ctx,
		&api.GetTelemetryGroupsMetricsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList2.StatusCode())
	assert.Empty(t, resList2.JSON200.TelemetryMetricsGroups)
}

func TestTelemetryProfileList_ListEmpty(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	resList1, err := apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList1.StatusCode())
	assert.Empty(t, resList1.JSON200.TelemetryLogsProfiles)

	resList2, err := apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList2.StatusCode())
	assert.Empty(t, resList2.JSON200.TelemetryMetricsProfiles)
}

func TestTelemetryLogsGroupList(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	totalItems := 10
	offset := 1
	pageSize := 4

	for id := 0; id < totalItems; id++ {
		CreateTelemetryLogsGroup(t, ctx, apiClient, api.TelemetryLogsGroup{
			CollectorKind: api.TELEMETRYCOLLECTORKINDCLUSTER,
			Groups:        []string{"test"},
			Name:          "Test Name",
		})
	}

	// Checks if list resources return expected number of entries
	resList, err := apiClient.GetTelemetryGroupsLogsWithResponse(
		ctx,
		&api.GetTelemetryGroupsLogsParams{
			Offset:   &offset,
			PageSize: &pageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.TelemetryLogsGroups), pageSize)
	assert.Equal(t, true, *resList.JSON200.HasNext)

	resList, err = apiClient.GetTelemetryGroupsLogsWithResponse(
		ctx,
		&api.GetTelemetryGroupsLogsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, totalItems, len(*resList.JSON200.TelemetryLogsGroups))
	assert.Equal(t, false, *resList.JSON200.HasNext)
}

func TestTelemetryMetricsGroupList(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	totalItems := 10
	offset := 1
	pageSize := 4

	for id := 0; id < totalItems; id++ {
		CreateTelemetryMetricsGroup(t, ctx, apiClient, api.TelemetryMetricsGroup{
			CollectorKind: api.TELEMETRYCOLLECTORKINDCLUSTER,
			Groups:        []string{"test"},
			Name:          "Test Name",
		})
	}

	// Checks if list resources return expected number of entries
	resList, err := apiClient.GetTelemetryGroupsMetricsWithResponse(
		ctx,
		&api.GetTelemetryGroupsMetricsParams{
			Offset:   &offset,
			PageSize: &pageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.TelemetryMetricsGroups), pageSize)
	assert.Equal(t, true, *resList.JSON200.HasNext)

	resList, err = apiClient.GetTelemetryGroupsMetricsWithResponse(
		ctx,
		&api.GetTelemetryGroupsMetricsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, totalItems, len(*resList.JSON200.TelemetryMetricsGroups))
	assert.Equal(t, false, *resList.JSON200.HasNext)
}

func TestTelemetryLogsProfileList(t *testing.T) {
	defer clearIDs()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	totalItems := 10
	offset := 1
	pageSize := 4

	group := CreateTelemetryLogsGroup(t, ctx, apiClient, api.TelemetryLogsGroup{
		CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
		Groups:        []string{"test"},
		Name:          "Test Name",
	})
	region1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	utils.Site1Request.RegionId = region1.JSON201.RegionID
	site1 := CreateSite(t, ctx, apiClient, utils.Site1Request)
	utils.Host1Request.SiteId = site1.JSON201.SiteID
	host := CreateHost(t, ctx, apiClient, utils.Host1Request)
	os := CreateOS(t, ctx, apiClient, utils.OSResource1Request)
	utils.Instance1Request.OsID = os.JSON201.OsResourceID
	utils.Instance1Request.HostID = host.JSON201.ResourceId
	instance := CreateInstance(t, ctx, apiClient, utils.Instance1Request)

	for id := 0; id < totalItems; id++ {
		CreateTelemetryLogsProfile(t, ctx, apiClient, api.TelemetryLogsProfile{
			LogsGroupId:    *group.JSON201.TelemetryLogsGroupId,
			LogLevel:       api.TELEMETRYSEVERITYLEVELWARN,
			TargetInstance: instance.JSON201.InstanceID,
		})
	}

	for id := 0; id < totalItems; id++ {
		CreateTelemetryLogsProfile(t, ctx, apiClient, api.TelemetryLogsProfile{
			LogsGroupId: *group.JSON201.TelemetryLogsGroupId,
			LogLevel:    api.TELEMETRYSEVERITYLEVELWARN,
			TargetSite:  site1.JSON201.SiteID,
		})
	}

	for id := 0; id < totalItems; id++ {
		CreateTelemetryLogsProfile(t, ctx, apiClient, api.TelemetryLogsProfile{
			LogsGroupId:  *group.JSON201.TelemetryLogsGroupId,
			LogLevel:     api.TELEMETRYSEVERITYLEVELWARN,
			TargetRegion: region1.JSON201.RegionID,
		})
	}

	// Checks if list resources return expected number of entries
	resList, err := apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			Offset:   &offset,
			PageSize: &pageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.TelemetryLogsProfiles), pageSize)
	assert.Equal(t, true, *resList.JSON200.HasNext)

	allPageSize := 30
	resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			PageSize: &allPageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, totalItems*3, len(*resList.JSON200.TelemetryLogsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	// check filters
	resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			InstanceId: instance.JSON201.InstanceID,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.TelemetryLogsProfiles), totalItems)
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			SiteId: site1.JSON201.SiteID,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.TelemetryLogsProfiles), totalItems)
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			RegionId: region1.JSON201.RegionID,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.TelemetryLogsProfiles), totalItems)
	assert.Equal(t, false, *resList.JSON200.HasNext)
}

func TestTelemetryMetricsProfileList(t *testing.T) {
	defer clearIDs()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	totalItems := 10
	pffset := 1
	pageSize := 4

	group := CreateTelemetryMetricsGroup(t, ctx, apiClient, api.TelemetryMetricsGroup{
		CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
		Groups:        []string{"test"},
		Name:          "Test Name",
	})
	region1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	utils.Site1Request.RegionId = region1.JSON201.RegionID
	site1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.Host1Request.SiteId = site1.JSON201.SiteID
	hostUUID := uuid.New()
	host := CreateHost(t, ctx, apiClient, api.Host{
		Name:     utils.Host1Request.Name,
		Metadata: utils.Host1Request.Metadata,
		Uuid:     &hostUUID,
	})

	os := CreateOS(t, ctx, apiClient, utils.OSResource1Request)
	utils.Instance1Request.OsID = os.JSON201.OsResourceID
	utils.Instance1Request.HostID = host.JSON201.ResourceId
	instance := CreateInstance(t, ctx, apiClient, utils.Instance1Request)

	for id := 0; id < totalItems; id++ {
		CreateTelemetryMetricsProfile(t, ctx, apiClient, api.TelemetryMetricsProfile{
			MetricsGroupId:  *group.JSON201.TelemetryMetricsGroupId,
			MetricsInterval: 300,
			TargetInstance:  instance.JSON201.InstanceID,
		})
	}

	for id := 0; id < totalItems; id++ {
		CreateTelemetryMetricsProfile(t, ctx, apiClient, api.TelemetryMetricsProfile{
			MetricsGroupId:  *group.JSON201.TelemetryMetricsGroupId,
			MetricsInterval: 300,
			TargetSite:      site1.JSON201.SiteID,
		})
	}

	for id := 0; id < totalItems; id++ {
		CreateTelemetryMetricsProfile(t, ctx, apiClient, api.TelemetryMetricsProfile{
			MetricsGroupId:  *group.JSON201.TelemetryMetricsGroupId,
			MetricsInterval: 300,
			TargetRegion:    region1.JSON201.RegionID,
		})
	}

	// Checks if list resources return expected number of entries
	resList, err := apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			Offset:   &pffset,
			PageSize: &pageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.TelemetryMetricsProfiles), pageSize)
	assert.Equal(t, true, *resList.JSON200.HasNext)

	allPageSize := 30
	resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			PageSize: &allPageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, totalItems*3, len(*resList.JSON200.TelemetryMetricsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	// check filters
	resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			InstanceId: instance.JSON201.InstanceID,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.TelemetryMetricsProfiles), totalItems)
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			SiteId: site1.JSON201.SiteID,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.TelemetryMetricsProfiles), totalItems)
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			RegionId: region1.JSON201.RegionID,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.TelemetryMetricsProfiles), totalItems)
	assert.Equal(t, false, *resList.JSON200.HasNext)
}

func TestTelemetryMetricsProfileListInherited(t *testing.T) {
	defer clearIDs()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	group := CreateTelemetryMetricsGroup(t, ctx, apiClient, api.TelemetryMetricsGroup{
		CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
		Groups:        []string{"test"},
		Name:          "Test Name",
	})
	os := CreateOS(t, ctx, apiClient, utils.OSResource1Request)
	region1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	utils.Site1Request.RegionId = region1.JSON201.RegionID
	site1Region1 := CreateSite(t, ctx, apiClient, utils.Site1Request)
	utils.Site2Request.RegionId = region1.JSON201.RegionID
	site2Region1 := CreateSite(t, ctx, apiClient, utils.Site2Request)
	parentRegion2Name := "Parent Region 2"
	parentRegion2 := CreateRegion(t, ctx, apiClient, api.Region{
		Name: &parentRegion2Name,
	})
	region2Name := "Region 2"
	region2 := CreateRegion(t, ctx, apiClient, api.Region{
		Name:     &region2Name,
		ParentId: parentRegion2.JSON201.RegionID,
	})
	site1Region2Name := "Site 1 Region 2"
	site1Region2 := CreateSite(t, ctx, apiClient, api.Site{
		Name:     &site1Region2Name,
		RegionId: region2.JSON201.RegionID,
	})
	// 3 Instances in Site 1 of Region 1
	site1Region1Instances := make([]*api.Instance, 0)
	kindMetal := api.INSTANCEKINDMETAL
	for i := 0; i < 3; i++ {
		hostUuid := uuid.New()
		host := CreateHost(t, ctx, apiClient, api.Host{
			Name:   fmt.Sprintf("Host %d S1R1", i),
			SiteId: site1Region1.JSON201.SiteID,
			Uuid:   &hostUuid,
		})
		instName := fmt.Sprintf("Site 1 Region 1 - Instance %d", i)
		inst := CreateInstance(t, ctx, apiClient, api.Instance{
			HostID: host.JSON201.ResourceId,
			OsID:   os.JSON201.OsResourceID,
			Kind:   &kindMetal,
			Name:   &instName,
		})
		site1Region1Instances = append(site1Region1Instances, inst.JSON201)
	}

	// 3 Instances in Site 2 of Region 1
	site2Region1Instances := make([]*api.Instance, 0)
	for i := 0; i < 3; i++ {
		hostUuid := uuid.New()
		host := CreateHost(t, ctx, apiClient, api.Host{
			Name:   fmt.Sprintf("Host %d S2R1", i),
			SiteId: site2Region1.JSON201.SiteID,
			Uuid:   &hostUuid,
		})
		instName := fmt.Sprintf("Site 2 Region 1 - Instance %d", i)
		inst := CreateInstance(t, ctx, apiClient, api.Instance{
			HostID: host.JSON201.ResourceId,
			OsID:   os.JSON201.OsResourceID,
			Kind:   &kindMetal,
			Name:   &instName,
		})
		site2Region1Instances = append(site2Region1Instances, inst.JSON201)
	}

	// 1 Instance in Site 1 of Region 2
	site1Region2Instances := make([]*api.Instance, 0)
	for i := 0; i < 1; i++ {
		hostUuid := uuid.New()
		host := CreateHost(t, ctx, apiClient, api.Host{
			Name:   fmt.Sprintf("Host %d S1R2", i),
			SiteId: site1Region2.JSON201.SiteID,
			Uuid:   &hostUuid,
		})
		instName := fmt.Sprintf("Site 1 Region 2 - Instance %d", i)
		inst := CreateInstance(t, ctx, apiClient, api.Instance{
			HostID: host.JSON201.ResourceId,
			OsID:   os.JSON201.OsResourceID,
			Kind:   &kindMetal,
			Name:   &instName,
		})
		site1Region2Instances = append(site1Region2Instances, inst.JSON201)
	}

	// Region 1 - 3 Telemetry Metrics Profiles
	for id := 0; id < 3; id++ {
		CreateTelemetryMetricsProfile(t, ctx, apiClient, api.TelemetryMetricsProfile{
			MetricsGroupId:  *group.JSON201.TelemetryMetricsGroupId,
			MetricsInterval: 300,
			TargetRegion:    region1.JSON201.RegionID,
		})
	}

	// Region 2 - 1 Telemetry Metrics Profile
	CreateTelemetryMetricsProfile(t, ctx, apiClient, api.TelemetryMetricsProfile{
		MetricsGroupId:  *group.JSON201.TelemetryMetricsGroupId,
		MetricsInterval: 300,
		TargetRegion:    region2.JSON201.RegionID,
	})

	// Parent Region 2 - 2 Telemetry Metrics Profiles
	for id := 0; id < 2; id++ {
		CreateTelemetryMetricsProfile(t, ctx, apiClient, api.TelemetryMetricsProfile{
			MetricsGroupId:  *group.JSON201.TelemetryMetricsGroupId,
			MetricsInterval: 300,
			TargetRegion:    parentRegion2.JSON201.RegionID,
		})
	}

	// Site 1 Region 1 - no Telemetry Metrics Profile

	// Site 2 Region 1 - 2 Telemetry Metrics Profiles
	for id := 0; id < 2; id++ {
		CreateTelemetryMetricsProfile(t, ctx, apiClient, api.TelemetryMetricsProfile{
			MetricsGroupId:  *group.JSON201.TelemetryMetricsGroupId,
			MetricsInterval: 300,
			TargetSite:      site2Region1.JSON201.SiteID,
		})
	}

	// Site 1 Region 2 - 1 Telemetry Metrics Profile
	CreateTelemetryMetricsProfile(t, ctx, apiClient, api.TelemetryMetricsProfile{
		MetricsGroupId:  *group.JSON201.TelemetryMetricsGroupId,
		MetricsInterval: 300,
		TargetSite:      site1Region2.JSON201.SiteID,
	})

	// Site 1 Region 1 - 1 Telemetry Profile per Instance
	for _, inst := range site1Region1Instances {
		CreateTelemetryMetricsProfile(t, ctx, apiClient, api.TelemetryMetricsProfile{
			MetricsGroupId:  *group.JSON201.TelemetryMetricsGroupId,
			MetricsInterval: 300,
			TargetInstance:  inst.InstanceID,
		})
	}

	// Site 2 Region 1 - No Telemetry Profiles for any Instance

	// Site 1 Region 2 - 1 Telemetry Profile per Instance
	for _, inst := range site1Region2Instances {
		CreateTelemetryMetricsProfile(t, ctx, apiClient, api.TelemetryMetricsProfile{
			MetricsGroupId:  *group.JSON201.TelemetryMetricsGroupId,
			MetricsInterval: 300,
			TargetInstance:  inst.InstanceID,
		})
	}

	offset := 1
	pageSize := 4

	// list all telemetry profiles (no filtering)
	resList, err := apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			Offset:   &offset,
			PageSize: &pageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.TelemetryMetricsProfiles), pageSize)
	assert.Equal(t, true, *resList.JSON200.HasNext)

	allPageSize := 100
	resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			PageSize: &allPageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 13, len(*resList.JSON200.TelemetryMetricsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	showInherited := true
	// render for Instances in Site 1 Region 1
	for _, inst := range site1Region1Instances {
		resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
			ctx,
			&api.GetTelemetryProfilesMetricsParams{
				InstanceId:    inst.InstanceID,
				ShowInherited: &showInherited,
			},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resList.StatusCode())
		assert.Equal(t, 4, // 1 for Instance + 0 for Site + 3 for Region 1 (no parent regions)
			len(*resList.JSON200.TelemetryMetricsProfiles))
		assert.Equal(t, false, *resList.JSON200.HasNext)

		// no inheritance
		resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
			ctx,
			&api.GetTelemetryProfilesMetricsParams{
				InstanceId: inst.InstanceID,
			},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resList.StatusCode())

		assert.Equal(t, 1, // 1 for Instance
			len(*resList.JSON200.TelemetryMetricsProfiles))
		assert.Equal(t, false, *resList.JSON200.HasNext)
	}

	// render for Instances in Site 2 Region 1
	for _, inst := range site2Region1Instances {
		resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
			ctx,
			&api.GetTelemetryProfilesMetricsParams{
				InstanceId:    inst.InstanceID,
				ShowInherited: &showInherited,
			},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resList.StatusCode())
		expectedItems := 5 // 0 for Instance + 2 for Site + 3 for Region (no parent regions)
		assert.Equal(t, expectedItems, len(*resList.JSON200.TelemetryMetricsProfiles))
		assert.Equal(t, false, *resList.JSON200.HasNext)

		// no inheritance
		resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
			ctx,
			&api.GetTelemetryProfilesMetricsParams{
				InstanceId: inst.InstanceID,
			},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resList.StatusCode())
		assert.Equal(t, 0, // 0 for Instance
			len(*resList.JSON200.TelemetryMetricsProfiles))
		assert.Equal(t, false, *resList.JSON200.HasNext)
	}

	// render for Instances in Site 1 Region 2
	for _, inst := range site1Region2Instances {
		resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
			ctx,
			&api.GetTelemetryProfilesMetricsParams{
				InstanceId:    inst.InstanceID,
				ShowInherited: &showInherited,
			},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resList.StatusCode())
		assert.Equal(t, 5, // 1 for Instance + 1 for Site + 1 for Region + 2 from Parent Region 2
			len(*resList.JSON200.TelemetryMetricsProfiles))
		assert.Equal(t, false, *resList.JSON200.HasNext)

		// no inheritance
		resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
			ctx,
			&api.GetTelemetryProfilesMetricsParams{
				InstanceId: inst.InstanceID,
			},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resList.StatusCode())
		assert.Equal(t, 1, // 1 for Instance
			len(*resList.JSON200.TelemetryMetricsProfiles))
		assert.Equal(t, false, *resList.JSON200.HasNext)
	}

	// render for Site 1 Region 1
	resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			SiteId:        site1Region1.JSON201.SiteID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 3, // 0 for Site + 3 for Region 1 (no parent regions)
		len(*resList.JSON200.TelemetryMetricsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	// render for Site 2 Region 1
	resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			SiteId:        site2Region1.JSON201.SiteID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 5, // 2 for Site + 3 for Region 1 (no parent regions)
		len(*resList.JSON200.TelemetryMetricsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	// render for Site 1 Region 2
	resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			SiteId:        site1Region2.JSON201.SiteID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 4, // 1 for Site + 1 for Region 2 + 2 for parent region
		len(*resList.JSON200.TelemetryMetricsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	// render for Region 1
	resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			RegionId:      region1.JSON201.RegionID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 3, //  3 for Region 1 (no parent regions)
		len(*resList.JSON200.TelemetryMetricsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	// render for Region 2
	resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			RegionId:      region2.JSON201.RegionID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 3, //  1 for Region 2 + 2 for parent region
		len(*resList.JSON200.TelemetryMetricsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)
}

func TestTelemetryMetricsProfileListInheritedNestingLimit(t *testing.T) {
	defer clearIDs()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	group := CreateTelemetryMetricsGroup(t, ctx, apiClient, api.TelemetryMetricsGroup{
		CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
		Groups:        []string{"test"},
		Name:          "Test Name",
	})
	os := CreateOS(t, ctx, apiClient, utils.OSResource1Request)

	regionLevel5Name := "Region 5"
	regionLevel5 := CreateRegion(t, ctx, apiClient, api.Region{
		Name: &regionLevel5Name,
	})

	regionLevel4Name := "Region 4"
	regionLevel4 := CreateRegion(t, ctx, apiClient, api.Region{
		Name:     &regionLevel4Name,
		ParentId: regionLevel5.JSON201.RegionID,
	})

	regionLevel3Name := "Region 3"
	regionLevel3 := CreateRegion(t, ctx, apiClient, api.Region{
		Name:     &regionLevel3Name,
		ParentId: regionLevel4.JSON201.RegionID,
	})

	regionLevel2Name := "Region 2"
	regionLevel2 := CreateRegion(t, ctx, apiClient, api.Region{
		Name:     &regionLevel2Name,
		ParentId: regionLevel3.JSON201.RegionID,
	})

	regionLevel1Name := "Region 1"
	regionLevel1 := CreateRegion(t, ctx, apiClient, api.Region{
		Name:     &regionLevel1Name,
		ParentId: regionLevel2.JSON201.RegionID,
	})

	utils.Site1Request.RegionId = regionLevel1.JSON201.RegionID
	site := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.Host1Request.SiteId = site.JSON201.SiteID
	host := CreateHost(t, ctx, apiClient, utils.Host1Request)

	utils.Instance1Request.OsID = os.JSON201.OsResourceID
	utils.Instance1Request.HostID = host.JSON201.ResourceId
	instance := CreateInstance(t, ctx, apiClient, utils.Instance1Request)

	// profile per instance
	CreateTelemetryMetricsProfile(t, ctx, apiClient, api.TelemetryMetricsProfile{
		MetricsGroupId:  *group.JSON201.TelemetryMetricsGroupId,
		MetricsInterval: 300,
		TargetInstance:  instance.JSON201.InstanceID,
	})
	// profile per site
	CreateTelemetryMetricsProfile(t, ctx, apiClient, api.TelemetryMetricsProfile{
		MetricsGroupId:  *group.JSON201.TelemetryMetricsGroupId,
		MetricsInterval: 300,
		TargetSite:      site.JSON201.SiteID,
	})
	// profile per region level 1
	CreateTelemetryMetricsProfile(t, ctx, apiClient, api.TelemetryMetricsProfile{
		MetricsGroupId:  *group.JSON201.TelemetryMetricsGroupId,
		MetricsInterval: 300,
		TargetRegion:    regionLevel1.JSON201.RegionID,
	})
	// profile per region level 3
	CreateTelemetryMetricsProfile(t, ctx, apiClient, api.TelemetryMetricsProfile{
		MetricsGroupId:  *group.JSON201.TelemetryMetricsGroupId,
		MetricsInterval: 300,
		TargetRegion:    regionLevel3.JSON201.RegionID,
	})
	// profile per region level 5
	CreateTelemetryMetricsProfile(t, ctx, apiClient, api.TelemetryMetricsProfile{
		MetricsGroupId:  *group.JSON201.TelemetryMetricsGroupId,
		MetricsInterval: 300,
		TargetRegion:    regionLevel5.JSON201.RegionID,
	})

	allPageSize := 100
	resList, err := apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			PageSize: &allPageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 5, len(*resList.JSON200.TelemetryMetricsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	showInherited := true
	resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			InstanceId:    instance.JSON201.InstanceID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 5, // 1 for Instance + 1 for Site + 1 for Region Level 1 + 1 for Region Level 3 + 1 for Region Level 5
		len(*resList.JSON200.TelemetryMetricsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			SiteId:        site.JSON201.SiteID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 4, // 1 for Site + 1 for Region Level 1 + 1 for Region Level 3 + 1 for Region Level 5
		len(*resList.JSON200.TelemetryMetricsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			RegionId:      regionLevel1.JSON201.RegionID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 3, // 1 for Region Level 1 + 1 for Region Level 3 + 1 for Region Level 5
		len(*resList.JSON200.TelemetryMetricsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			RegionId:      regionLevel4.JSON201.RegionID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 1, // 1 for Region Level 5
		len(*resList.JSON200.TelemetryMetricsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)
}

func TestTelemetryMetricsProfileListInheritedNoParents(t *testing.T) {
	defer clearIDs()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	group := CreateTelemetryMetricsGroup(t, ctx, apiClient, api.TelemetryMetricsGroup{
		CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
		Groups:        []string{"test"},
		Name:          "Test Name",
	})
	os := CreateOS(t, ctx, apiClient, utils.OSResource1Request)

	region2Name := "Region 2"
	region2 := CreateRegion(t, ctx, apiClient, api.Region{
		Name: &region2Name,
	})

	region1Name := "Region 1"
	region1 := CreateRegion(t, ctx, apiClient, api.Region{
		Name: &region1Name,
	})

	utils.Site1Request.RegionId = nil
	site := CreateSite(t, ctx, apiClient, utils.Site1Request)
	utils.Host1Request.SiteId = nil
	host := CreateHost(t, ctx, apiClient, utils.Host1Request)

	utils.Instance1Request.OsID = os.JSON201.OsResourceID
	utils.Instance1Request.HostID = host.JSON201.ResourceId
	instance := CreateInstance(t, ctx, apiClient, utils.Instance1Request)

	// profile per instance
	CreateTelemetryMetricsProfile(t, ctx, apiClient, api.TelemetryMetricsProfile{
		MetricsGroupId:  *group.JSON201.TelemetryMetricsGroupId,
		MetricsInterval: 300,
		TargetInstance:  instance.JSON201.InstanceID,
	})
	// profile per site
	CreateTelemetryMetricsProfile(t, ctx, apiClient, api.TelemetryMetricsProfile{
		MetricsGroupId:  *group.JSON201.TelemetryMetricsGroupId,
		MetricsInterval: 300,
		TargetSite:      site.JSON201.SiteID,
	})
	// profile per region 1
	CreateTelemetryMetricsProfile(t, ctx, apiClient, api.TelemetryMetricsProfile{
		MetricsGroupId:  *group.JSON201.TelemetryMetricsGroupId,
		MetricsInterval: 300,
		TargetRegion:    region1.JSON201.RegionID,
	})
	// profile per region 2
	CreateTelemetryMetricsProfile(t, ctx, apiClient, api.TelemetryMetricsProfile{
		MetricsGroupId:  *group.JSON201.TelemetryMetricsGroupId,
		MetricsInterval: 300,
		TargetRegion:    region2.JSON201.RegionID,
	})

	allPageSize := 100
	resList, err := apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			PageSize: &allPageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 4, len(*resList.JSON200.TelemetryMetricsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	showInherited := true
	resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			InstanceId:    instance.JSON201.InstanceID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 1, // 1 for Instance, no parent relations
		len(*resList.JSON200.TelemetryMetricsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			SiteId:        site.JSON201.SiteID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 1, // 1 for Site, no parent relations
		len(*resList.JSON200.TelemetryMetricsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetTelemetryProfilesMetricsWithResponse(
		ctx,
		&api.GetTelemetryProfilesMetricsParams{
			RegionId:      region1.JSON201.RegionID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 1, // 1 for Region, no parents
		len(*resList.JSON200.TelemetryMetricsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)
}

func TestTelemetryLogsProfileListInherited(t *testing.T) {
	defer clearIDs()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	group := CreateTelemetryLogsGroup(t, ctx, apiClient, api.TelemetryLogsGroup{
		CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
		Groups:        []string{"test"},
		Name:          "Test Name",
	})
	os := CreateOS(t, ctx, apiClient, utils.OSResource1Request)
	region1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	utils.Site1Request.RegionId = region1.JSON201.RegionID
	site1Region1 := CreateSite(t, ctx, apiClient, utils.Site1Request)
	utils.Site2Request.RegionId = region1.JSON201.RegionID
	site2Region1 := CreateSite(t, ctx, apiClient, utils.Site2Request)
	parentRegion2Name := "Parent Region 2"
	parentRegion2 := CreateRegion(t, ctx, apiClient, api.Region{
		Name: &parentRegion2Name,
	})
	region2Name := "Region 2"
	region2 := CreateRegion(t, ctx, apiClient, api.Region{
		Name:     &region2Name,
		ParentId: parentRegion2.JSON201.RegionID,
	})
	site1Region2Name := "Site 1 Region 2"
	site1Region2 := CreateSite(t, ctx, apiClient, api.Site{
		Name:     &site1Region2Name,
		RegionId: region2.JSON201.RegionID,
	})
	// 3 Instances in Site 1 of Region 1
	site1Region1Instances := make([]*api.Instance, 0)
	kindMetal := api.INSTANCEKINDMETAL
	for i := 0; i < 3; i++ {
		hostUuid := uuid.New()
		host := CreateHost(t, ctx, apiClient, api.Host{
			Name:   fmt.Sprintf("Host %d S1R1", i),
			SiteId: site1Region1.JSON201.SiteID,
			Uuid:   &hostUuid,
		})
		instName := fmt.Sprintf("Site 1 Region 1 - Instance %d", i)
		inst := CreateInstance(t, ctx, apiClient, api.Instance{
			HostID: host.JSON201.ResourceId,
			OsID:   os.JSON201.OsResourceID,
			Kind:   &kindMetal,
			Name:   &instName,
		})
		site1Region1Instances = append(site1Region1Instances, inst.JSON201)
	}

	// 3 Instances in Site 2 of Region 1
	site2Region1Instances := make([]*api.Instance, 0)
	for i := 0; i < 3; i++ {
		hostUuid := uuid.New()
		host := CreateHost(t, ctx, apiClient, api.Host{
			Name:   fmt.Sprintf("Host %d S2R1", i),
			SiteId: site2Region1.JSON201.SiteID,
			Uuid:   &hostUuid,
		})
		instName := fmt.Sprintf("Site 2 Region 1 - Instance %d", i)
		inst := CreateInstance(t, ctx, apiClient, api.Instance{
			HostID: host.JSON201.ResourceId,
			OsID:   os.JSON201.OsResourceID,
			Kind:   &kindMetal,
			Name:   &instName,
		})
		site2Region1Instances = append(site2Region1Instances, inst.JSON201)
	}

	// 1 Instance in Site 1 of Region 2
	site1Region2Instances := make([]*api.Instance, 0)
	for i := 0; i < 1; i++ {
		hostUuid := uuid.New()
		host := CreateHost(t, ctx, apiClient, api.Host{
			Name:   fmt.Sprintf("Host %d S1R2", i),
			SiteId: site1Region2.JSON201.SiteID,
			Uuid:   &hostUuid,
		})
		instName := fmt.Sprintf("Site 1 Region 2 - Instance %d", i)
		inst := CreateInstance(t, ctx, apiClient, api.Instance{
			HostID: host.JSON201.ResourceId,
			OsID:   os.JSON201.OsResourceID,
			Kind:   &kindMetal,
			Name:   &instName,
		})
		site1Region2Instances = append(site1Region2Instances, inst.JSON201)
	}

	// Region 1 - 3 Telemetry Logs Profiles
	for id := 0; id < 3; id++ {
		CreateTelemetryLogsProfile(t, ctx, apiClient, api.TelemetryLogsProfile{
			LogsGroupId:  *group.JSON201.TelemetryLogsGroupId,
			LogLevel:     api.TELEMETRYSEVERITYLEVELWARN,
			TargetRegion: region1.JSON201.RegionID,
		})
	}

	// Region 2 - 1 Telemetry Logs Profile
	CreateTelemetryLogsProfile(t, ctx, apiClient, api.TelemetryLogsProfile{
		LogsGroupId:  *group.JSON201.TelemetryLogsGroupId,
		LogLevel:     api.TELEMETRYSEVERITYLEVELWARN,
		TargetRegion: region2.JSON201.RegionID,
	})

	// Parent Region 2 - 2 Telemetry Logs Profiles
	for id := 0; id < 2; id++ {
		CreateTelemetryLogsProfile(t, ctx, apiClient, api.TelemetryLogsProfile{
			LogsGroupId:  *group.JSON201.TelemetryLogsGroupId,
			LogLevel:     api.TELEMETRYSEVERITYLEVELWARN,
			TargetRegion: parentRegion2.JSON201.RegionID,
		})
	}

	// Site 1 Region 1 - no Telemetry Logs Profile

	// Site 2 Region 1 - 2 Telemetry Logs Profiles
	for id := 0; id < 2; id++ {
		CreateTelemetryLogsProfile(t, ctx, apiClient, api.TelemetryLogsProfile{
			LogsGroupId: *group.JSON201.TelemetryLogsGroupId,
			LogLevel:    api.TELEMETRYSEVERITYLEVELWARN,
			TargetSite:  site2Region1.JSON201.SiteID,
		})
	}

	// Site 1 Region 2 - 1 Telemetry Logs Profile
	CreateTelemetryLogsProfile(t, ctx, apiClient, api.TelemetryLogsProfile{
		LogsGroupId: *group.JSON201.TelemetryLogsGroupId,
		LogLevel:    api.TELEMETRYSEVERITYLEVELWARN,
		TargetSite:  site1Region2.JSON201.SiteID,
	})

	// Site 1 Region 1 - 1 Telemetry Profile per Instance
	for _, inst := range site1Region1Instances {
		CreateTelemetryLogsProfile(t, ctx, apiClient, api.TelemetryLogsProfile{
			LogsGroupId:    *group.JSON201.TelemetryLogsGroupId,
			LogLevel:       api.TELEMETRYSEVERITYLEVELWARN,
			TargetInstance: inst.InstanceID,
		})
	}

	// Site 2 Region 1 - No Telemetry Profiles for any Instance

	// Site 1 Region 2 - 1 Telemetry Profile per Instance
	for _, inst := range site1Region2Instances {
		CreateTelemetryLogsProfile(t, ctx, apiClient, api.TelemetryLogsProfile{
			LogsGroupId:    *group.JSON201.TelemetryLogsGroupId,
			LogLevel:       api.TELEMETRYSEVERITYLEVELWARN,
			TargetInstance: inst.InstanceID,
		})
	}

	offset := 1
	pageSize := 4

	// list all telemetry profiles (no filtering)
	resList, err := apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			Offset:   &offset,
			PageSize: &pageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.TelemetryLogsProfiles), pageSize)
	assert.Equal(t, true, *resList.JSON200.HasNext)

	allPageSize := 100
	resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			PageSize: &allPageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 13, len(*resList.JSON200.TelemetryLogsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	showInherited := true
	// render for Instances in Site 1 Region 1
	for _, inst := range site1Region1Instances {
		resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
			ctx,
			&api.GetTelemetryProfilesLogsParams{
				InstanceId:    inst.InstanceID,
				ShowInherited: &showInherited,
			},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resList.StatusCode())
		assert.Equal(t, 4, // 1 for Instance + 0 for Site + 3 for Region 1 (no parent regions)
			len(*resList.JSON200.TelemetryLogsProfiles))
		assert.Equal(t, false, *resList.JSON200.HasNext)

		// no inheritance
		resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
			ctx,
			&api.GetTelemetryProfilesLogsParams{
				InstanceId: inst.InstanceID,
			},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resList.StatusCode())

		assert.Equal(t, 1, // 1 for Instance
			len(*resList.JSON200.TelemetryLogsProfiles))
		assert.Equal(t, false, *resList.JSON200.HasNext)
	}

	// render for Instances in Site 2 Region 1
	for _, inst := range site2Region1Instances {
		resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
			ctx,
			&api.GetTelemetryProfilesLogsParams{
				InstanceId:    inst.InstanceID,
				ShowInherited: &showInherited,
			},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resList.StatusCode())
		expectedItems := 5 // 0 for Instance + 2 for Site + 3 for Region (no parent regions)
		assert.Equal(t, expectedItems, len(*resList.JSON200.TelemetryLogsProfiles))
		assert.Equal(t, false, *resList.JSON200.HasNext)

		// no inheritance
		resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
			ctx,
			&api.GetTelemetryProfilesLogsParams{
				InstanceId: inst.InstanceID,
			},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resList.StatusCode())
		assert.Equal(t, 0, // 0 for Instance
			len(*resList.JSON200.TelemetryLogsProfiles))
		assert.Equal(t, false, *resList.JSON200.HasNext)
	}

	// render for Instances in Site 1 Region 2
	for _, inst := range site1Region2Instances {
		resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
			ctx,
			&api.GetTelemetryProfilesLogsParams{
				InstanceId:    inst.InstanceID,
				ShowInherited: &showInherited,
			},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resList.StatusCode())
		assert.Equal(t, 5, // 1 for Instance + 1 for Site + 1 for Region + 2 from Parent Region 2
			len(*resList.JSON200.TelemetryLogsProfiles))
		assert.Equal(t, false, *resList.JSON200.HasNext)

		// no inheritance
		resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
			ctx,
			&api.GetTelemetryProfilesLogsParams{
				InstanceId: inst.InstanceID,
			},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resList.StatusCode())
		assert.Equal(t, 1, // 1 for Instance
			len(*resList.JSON200.TelemetryLogsProfiles))
		assert.Equal(t, false, *resList.JSON200.HasNext)
	}

	// render for Site 1 Region 1
	resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			SiteId:        site1Region1.JSON201.SiteID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 3, // 0 for Site + 3 for Region 1 (no parent regions)
		len(*resList.JSON200.TelemetryLogsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	// render for Site 2 Region 1
	resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			SiteId:        site2Region1.JSON201.SiteID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 5, // 2 for Site + 3 for Region 1 (no parent regions)
		len(*resList.JSON200.TelemetryLogsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	// render for Site 1 Region 2
	resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			SiteId:        site1Region2.JSON201.SiteID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 4, // 1 for Site + 1 for Region 2 + 2 for parent region
		len(*resList.JSON200.TelemetryLogsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	// render for Region 1
	resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			RegionId:      region1.JSON201.RegionID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 3, //  3 for Region 1 (no parent regions)
		len(*resList.JSON200.TelemetryLogsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	// render for Region 2
	resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			RegionId:      region2.JSON201.RegionID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 3, //  1 for Region 2 + 2 for parent region
		len(*resList.JSON200.TelemetryLogsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)
}

func TestTelemetryMetricsLogsListInheritedNestingLimit(t *testing.T) {
	defer clearIDs()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	group := CreateTelemetryLogsGroup(t, ctx, apiClient, api.TelemetryLogsGroup{
		CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
		Groups:        []string{"test"},
		Name:          "Test Name",
	})
	os := CreateOS(t, ctx, apiClient, utils.OSResource1Request)

	regionLevel5Name := "Region 5"
	regionLevel5 := CreateRegion(t, ctx, apiClient, api.Region{
		Name: &regionLevel5Name,
	})

	regionLevel4Name := "Region 4"
	regionLevel4 := CreateRegion(t, ctx, apiClient, api.Region{
		Name:     &regionLevel4Name,
		ParentId: regionLevel5.JSON201.RegionID,
	})

	regionLevel3Name := "Region 3"
	regionLevel3 := CreateRegion(t, ctx, apiClient, api.Region{
		Name:     &regionLevel3Name,
		ParentId: regionLevel4.JSON201.RegionID,
	})

	regionLevel2Name := "Region 2"
	regionLevel2 := CreateRegion(t, ctx, apiClient, api.Region{
		Name:     &regionLevel2Name,
		ParentId: regionLevel3.JSON201.RegionID,
	})

	regionLevel1Name := "Region 1"
	regionLevel1 := CreateRegion(t, ctx, apiClient, api.Region{
		Name:     &regionLevel1Name,
		ParentId: regionLevel2.JSON201.RegionID,
	})

	utils.Site1Request.RegionId = regionLevel1.JSON201.RegionID
	site := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.Host1Request.SiteId = site.JSON201.SiteID
	host := CreateHost(t, ctx, apiClient, utils.Host1Request)

	utils.Instance1Request.OsID = os.JSON201.OsResourceID
	utils.Instance1Request.HostID = host.JSON201.ResourceId
	instance := CreateInstance(t, ctx, apiClient, utils.Instance1Request)

	// profile per instance
	CreateTelemetryLogsProfile(t, ctx, apiClient, api.TelemetryLogsProfile{
		LogsGroupId:    *group.JSON201.TelemetryLogsGroupId,
		LogLevel:       api.TELEMETRYSEVERITYLEVELWARN,
		TargetInstance: instance.JSON201.InstanceID,
	})
	// profile per site
	CreateTelemetryLogsProfile(t, ctx, apiClient, api.TelemetryLogsProfile{
		LogsGroupId: *group.JSON201.TelemetryLogsGroupId,
		LogLevel:    api.TELEMETRYSEVERITYLEVELWARN,
		TargetSite:  site.JSON201.SiteID,
	})
	// profile per region level 1
	CreateTelemetryLogsProfile(t, ctx, apiClient, api.TelemetryLogsProfile{
		LogsGroupId:  *group.JSON201.TelemetryLogsGroupId,
		LogLevel:     api.TELEMETRYSEVERITYLEVELWARN,
		TargetRegion: regionLevel1.JSON201.RegionID,
	})
	// profile per region level 3
	CreateTelemetryLogsProfile(t, ctx, apiClient, api.TelemetryLogsProfile{
		LogsGroupId:  *group.JSON201.TelemetryLogsGroupId,
		LogLevel:     api.TELEMETRYSEVERITYLEVELWARN,
		TargetRegion: regionLevel3.JSON201.RegionID,
	})
	// profile per region level 5
	CreateTelemetryLogsProfile(t, ctx, apiClient, api.TelemetryLogsProfile{
		LogsGroupId:  *group.JSON201.TelemetryLogsGroupId,
		LogLevel:     api.TELEMETRYSEVERITYLEVELWARN,
		TargetRegion: regionLevel5.JSON201.RegionID,
	})

	allPageSize := 100
	resList, err := apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			PageSize: &allPageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 5, len(*resList.JSON200.TelemetryLogsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	showInherited := true
	resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			InstanceId:    instance.JSON201.InstanceID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 5, // 1 for Instance + 1 for Site + 1 for Region Level 1 + 1 for Region Level 3 + 1 for Region Level 5
		len(*resList.JSON200.TelemetryLogsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			SiteId:        site.JSON201.SiteID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 4, // 1 for Site + 1 for Region Level 1 + 1 for Region Level 3 + 1 for Region Level 5
		len(*resList.JSON200.TelemetryLogsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			RegionId:      regionLevel1.JSON201.RegionID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 3, // 1 for Region Level 1 + 1 for Region Level 3 + 1 for Region Level 5
		len(*resList.JSON200.TelemetryLogsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			RegionId:      regionLevel4.JSON201.RegionID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 1, // 1 for Region Level 5
		len(*resList.JSON200.TelemetryLogsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)
}

func TestTelemetryLogsProfileListInheritedNoParents(t *testing.T) {
	defer clearIDs()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	group := CreateTelemetryLogsGroup(t, ctx, apiClient, api.TelemetryLogsGroup{
		CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
		Groups:        []string{"test"},
		Name:          "Test Name",
	})
	os := CreateOS(t, ctx, apiClient, utils.OSResource1Request)

	region2Name := "Region 2"
	region2 := CreateRegion(t, ctx, apiClient, api.Region{
		Name: &region2Name,
	})

	region1Name := "Region 1"
	region1 := CreateRegion(t, ctx, apiClient, api.Region{
		Name: &region1Name,
	})

	utils.Site1Request.RegionId = nil
	site := CreateSite(t, ctx, apiClient, utils.Site1Request)
	utils.Host1Request.SiteId = nil
	host := CreateHost(t, ctx, apiClient, utils.Host1Request)

	utils.Instance1Request.OsID = os.JSON201.OsResourceID
	utils.Instance1Request.HostID = host.JSON201.ResourceId
	instance := CreateInstance(t, ctx, apiClient, utils.Instance1Request)

	// profile per instance
	CreateTelemetryLogsProfile(t, ctx, apiClient, api.TelemetryLogsProfile{
		LogsGroupId:    *group.JSON201.TelemetryLogsGroupId,
		LogLevel:       api.TELEMETRYSEVERITYLEVELWARN,
		TargetInstance: instance.JSON201.InstanceID,
	})
	// profile per site
	CreateTelemetryLogsProfile(t, ctx, apiClient, api.TelemetryLogsProfile{
		LogsGroupId: *group.JSON201.TelemetryLogsGroupId,
		LogLevel:    api.TELEMETRYSEVERITYLEVELWARN,
		TargetSite:  site.JSON201.SiteID,
	})
	// profile per region 1
	CreateTelemetryLogsProfile(t, ctx, apiClient, api.TelemetryLogsProfile{
		LogsGroupId:  *group.JSON201.TelemetryLogsGroupId,
		LogLevel:     api.TELEMETRYSEVERITYLEVELWARN,
		TargetRegion: region1.JSON201.RegionID,
	})
	// profile per region 2
	CreateTelemetryLogsProfile(t, ctx, apiClient, api.TelemetryLogsProfile{
		LogsGroupId:  *group.JSON201.TelemetryLogsGroupId,
		LogLevel:     api.TELEMETRYSEVERITYLEVELWARN,
		TargetRegion: region2.JSON201.RegionID,
	})

	allPageSize := 100
	resList, err := apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			PageSize: &allPageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 4, len(*resList.JSON200.TelemetryLogsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	showInherited := true
	resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			InstanceId:    instance.JSON201.InstanceID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 1, // 1 for Instance, no parent relations
		len(*resList.JSON200.TelemetryLogsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			SiteId:        site.JSON201.SiteID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 1, // 1 for Site, no parent relations
		len(*resList.JSON200.TelemetryLogsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetTelemetryProfilesLogsWithResponse(
		ctx,
		&api.GetTelemetryProfilesLogsParams{
			RegionId:      region1.JSON201.RegionID,
			ShowInherited: &showInherited,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 1, // 1 for Region, no parents
		len(*resList.JSON200.TelemetryLogsProfiles))
	assert.Equal(t, false, *resList.JSON200.HasNext)
}
