// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	"github.com/open-edge-platform/infra-core/api/test/utils"
)

func TestSchedRepeated_CreateGetDelete(t *testing.T) {
	log.Info().Msgf("Begin RepeatedSched tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	utils.Site1Request.RegionId = nil
	utils.Site1Request.OuId = nil
	siteCreated1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.RepeatedSchedule1Request.TargetSiteId = siteCreated1.JSON201.SiteID
	RepeatedSched1 := CreateSchedRepeated(t, ctx, apiClient, utils.RepeatedSchedule1Request)

	utils.RepeatedSchedule2Request.TargetSiteId = siteCreated1.JSON201.SiteID
	RepeatedSched2 := CreateSchedRepeated(t, ctx, apiClient, utils.RepeatedSchedule1Request)

	get1, err := apiClient.GetSchedulesRepeatedRepeatedScheduleIDWithResponse(
		ctx,
		*RepeatedSched1.JSON201.RepeatedScheduleID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, get1.StatusCode())
	assert.Equal(t, utils.SschedName1, *get1.JSON200.Name)

	get2, err := apiClient.GetSchedulesRepeatedRepeatedScheduleIDWithResponse(
		ctx,
		*RepeatedSched2.JSON201.RepeatedScheduleID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, get2.StatusCode())
	assert.Equal(t, utils.SschedName1, *get2.JSON200.Name)

	log.Info().Msgf("End RepeatedSchedule tests")
}

func TestSchedRepeated_CreateError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	utils.Site1Request.RegionId = nil
	utils.Site1Request.OuId = nil
	siteCreated1 := CreateSite(t, ctx, apiClient, utils.Site1Request)
	utils.Host1Request.SiteId = siteCreated1.JSON201.SiteID
	hostCreated1 := CreateHost(t, ctx, apiClient, utils.Host1Request)

	// Expected StatusUnprocessableEntity Error because of target site and host are set in Schedule
	utils.RepeatedScheduleError.TargetSiteId = siteCreated1.JSON201.SiteID
	utils.RepeatedScheduleError.TargetHostId = hostCreated1.JSON201.ResourceId

	sched, err := apiClient.PostSchedulesRepeatedWithResponse(
		ctx,
		utils.RepeatedScheduleError,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, sched.StatusCode())
}

func TestSchedRepeated_UpdatePut(t *testing.T) {
	log.Info().Msgf("Begin RepeatedSchedule Update tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	utils.Site1Request.RegionId = nil
	utils.Site1Request.OuId = nil
	siteCreated1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.RepeatedSchedule1Request.TargetSiteId = siteCreated1.JSON201.SiteID
	RepeatedSched1 := CreateSchedRepeated(t, ctx, apiClient, utils.RepeatedSchedule1Request)

	RepeatedSchedule1Get, err := apiClient.GetSchedulesRepeatedRepeatedScheduleIDWithResponse(
		ctx,
		*RepeatedSched1.JSON201.RepeatedScheduleID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, RepeatedSchedule1Get.StatusCode())
	assert.Equal(t, utils.SschedName1, *RepeatedSchedule1Get.JSON200.Name)

	utils.SiteListRequest1.Region = nil
	utils.SiteListRequest1.Ou = nil
	siteCreated2 := CreateSite(t, ctx, apiClient, utils.SiteListRequest1)
	utils.RepeatedSchedule2Request.TargetSiteId = siteCreated2.JSON201.SiteID

	RepeatedSched1Update, err := apiClient.PutSchedulesRepeatedRepeatedScheduleIDWithResponse(
		ctx,
		*RepeatedSched1.JSON201.RepeatedScheduleID,
		utils.RepeatedSchedule2Request,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, RepeatedSched1Update.StatusCode())
	assert.Equal(t, utils.SschedName2, *RepeatedSched1Update.JSON200.Name)

	RepeatedSchedule1GetUp, err := apiClient.GetSchedulesRepeatedRepeatedScheduleIDWithResponse(
		ctx,
		*RepeatedSched1.JSON201.RepeatedScheduleID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, RepeatedSchedule1GetUp.StatusCode())
	assert.Equal(t, utils.SschedName2, *RepeatedSchedule1GetUp.JSON200.Name)
	assert.Equal(t, *siteCreated2.JSON201.SiteID, *RepeatedSchedule1GetUp.JSON200.TargetSite.ResourceId)
	assert.Equal(
		t,
		utils.RepeatedSchedule2Request.CronDayMonth,
		RepeatedSchedule1GetUp.JSON200.CronDayMonth,
	)

	// Uses PUT to set empty string to TargetSite and verifies it
	utils.RepeatedSchedule2Request.TargetSiteId = &emptyString
	RepeatedSched1Update, err = apiClient.PutSchedulesRepeatedRepeatedScheduleIDWithResponse(
		ctx,
		*RepeatedSched1.JSON201.RepeatedScheduleID,
		utils.RepeatedSchedule2Request,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, RepeatedSched1Update.StatusCode())
	assert.Equal(t, utils.SschedName2, *RepeatedSched1Update.JSON200.Name)

	RepeatedSchedule1GetUp, err = apiClient.GetSchedulesRepeatedRepeatedScheduleIDWithResponse(
		ctx,
		*RepeatedSched1.JSON201.RepeatedScheduleID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, RepeatedSchedule1GetUp.StatusCode())
	assert.Equal(t, utils.SschedName2, *RepeatedSchedule1GetUp.JSON200.Name)
	assert.Empty(t, RepeatedSchedule1GetUp.JSON200.TargetSite)

	// Uses PUT to set wrong empty string to TargetSite and verifies its BadRequest error
	utils.RepeatedSchedule2Request.TargetSiteId = &emptyStringWrong
	RepeatedSched1Update, err = apiClient.PutSchedulesRepeatedRepeatedScheduleIDWithResponse(
		ctx,
		*RepeatedSched1.JSON201.RepeatedScheduleID,
		utils.RepeatedSchedule2Request,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, RepeatedSched1Update.StatusCode())

	utils.RepeatedSchedule2Request.TargetSite = nil

	log.Info().Msgf("End RepeatedSchedule Update tests")
}

func TestSchedRepeated_UpdatePatch(t *testing.T) {
	log.Info().Msgf("Begin RepeatedSchedule Update tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	utils.Site1Request.RegionId = nil
	utils.Site1Request.OuId = nil
	siteCreated1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.RepeatedSchedule1Request.TargetSiteId = siteCreated1.JSON201.SiteID
	RepeatedSched1 := CreateSchedRepeated(t, ctx, apiClient, utils.RepeatedSchedule1Request)

	RepeatedSchedule1Get, err := apiClient.GetSchedulesRepeatedRepeatedScheduleIDWithResponse(
		ctx,
		*RepeatedSched1.JSON201.RepeatedScheduleID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, RepeatedSchedule1Get.StatusCode())
	assert.Equal(t, utils.SschedName1, *RepeatedSchedule1Get.JSON200.Name)

	utils.SiteListRequest1.Region = nil
	utils.SiteListRequest1.Ou = nil
	siteCreated2 := CreateSite(t, ctx, apiClient, utils.SiteListRequest1)

	utils.RepeatedSchedule2Request.TargetSiteId = siteCreated2.JSON201.SiteID
	RepeatedSched1Update, err := apiClient.PatchSchedulesRepeatedRepeatedScheduleIDWithResponse(
		ctx,
		*RepeatedSched1.JSON201.RepeatedScheduleID,
		utils.RepeatedSchedule2Request,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, RepeatedSched1Update.StatusCode())
	assert.Equal(t, utils.SschedName2, *RepeatedSched1Update.JSON200.Name)

	RepeatedSchedule1GetUp, err := apiClient.GetSchedulesRepeatedRepeatedScheduleIDWithResponse(
		ctx,
		*RepeatedSched1.JSON201.RepeatedScheduleID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, RepeatedSchedule1GetUp.StatusCode())
	assert.Equal(t, utils.SschedName2, *RepeatedSchedule1GetUp.JSON200.Name)
	assert.Equal(t, *siteCreated2.JSON201.SiteID, *RepeatedSchedule1GetUp.JSON200.TargetSite.ResourceId)
	assert.Equal(
		t,
		utils.RepeatedSchedule2Request.CronDayMonth,
		RepeatedSchedule1GetUp.JSON200.CronDayMonth,
	)

	// Uses PATCH to set empty string to TargetSite and verifies it
	utils.RepeatedSchedule2Request.TargetSiteId = &emptyString
	RepeatedSched1Update, err = apiClient.PatchSchedulesRepeatedRepeatedScheduleIDWithResponse(
		ctx,
		*RepeatedSched1.JSON201.RepeatedScheduleID,
		utils.RepeatedSchedule2Request,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, RepeatedSched1Update.StatusCode())
	assert.Equal(t, utils.SschedName2, *RepeatedSched1Update.JSON200.Name)

	RepeatedSchedule1GetUp, err = apiClient.GetSchedulesRepeatedRepeatedScheduleIDWithResponse(
		ctx,
		*RepeatedSched1.JSON201.RepeatedScheduleID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, RepeatedSchedule1GetUp.StatusCode())
	assert.Equal(t, utils.SschedName2, *RepeatedSchedule1GetUp.JSON200.Name)
	assert.Empty(t, RepeatedSchedule1GetUp.JSON200.TargetSite)
	assert.Equal(
		t,
		utils.RepeatedSchedule2Request.CronDayWeek,
		RepeatedSchedule1GetUp.JSON200.CronDayWeek,
	)

	// Uses PATCH to set wrong empty string to TargetSite and verifies its BadRequest error
	utils.RepeatedSchedule2Request.TargetSiteId = &emptyStringWrong
	RepeatedSched1Update, err = apiClient.PatchSchedulesRepeatedRepeatedScheduleIDWithResponse(
		ctx,
		*RepeatedSched1.JSON201.RepeatedScheduleID,
		utils.RepeatedSchedule2Request,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, RepeatedSched1Update.StatusCode())

	utils.RepeatedSchedule2Request.TargetSite = nil

	log.Info().Msgf("End RepeatedSchedule Update tests")
}

func TestSchedRepeated_Errors(t *testing.T) {
	log.Info().Msgf("Begin RepeatedSchedule Error tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)
	if err != nil {
		t.Fatalf("new API client error %s", err.Error())
	}

	utils.Site1Request.RegionId = nil
	utils.Site1Request.OuId = nil
	siteCreated1 := CreateSite(t, ctx, apiClient, utils.Site1Request)
	utils.RepeatedSchedule1Request.TargetSiteId = siteCreated1.JSON201.SiteID

	t.Run("Put_UnexistID_Status_NotFoundError", func(t *testing.T) {
		RepeatedSched1Up, err := apiClient.PutSchedulesRepeatedRepeatedScheduleIDWithResponse(
			ctx,
			utils.RepeatedScheduleUnexistID,
			utils.RepeatedSchedule1Request,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)

		log.Info().Msgf("error body %s", RepeatedSched1Up.Body)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, RepeatedSched1Up.StatusCode())
	})

	t.Run("Patch_UnexistID_Status_NotFoundError", func(t *testing.T) {
		RepeatedSched1Up, err := apiClient.PatchSchedulesRepeatedRepeatedScheduleIDWithResponse(
			ctx,
			utils.RepeatedScheduleUnexistID,
			utils.RepeatedSchedule1Request,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, RepeatedSched1Up.StatusCode())
	})

	t.Run("Get_UnexistID_Status_NotFoundError", func(t *testing.T) {
		s1res, err := apiClient.GetSchedulesRepeatedRepeatedScheduleIDWithResponse(
			ctx,
			utils.RepeatedScheduleUnexistID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, s1res.StatusCode())
	})

	t.Run("Delete_UnexistID_Status_NotFoundError", func(t *testing.T) {
		resDelSite, err := apiClient.DeleteSchedulesRepeatedRepeatedScheduleIDWithResponse(
			ctx,
			utils.RepeatedScheduleUnexistID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resDelSite.StatusCode())
	})

	t.Run("Put_WrongID_Status_StatusBadRequest", func(t *testing.T) {
		RepeatedSched1Up, err := apiClient.PutSchedulesRepeatedRepeatedScheduleIDWithResponse(
			ctx,
			utils.RepeatedScheduleWrongID,
			utils.RepeatedSchedule1Request,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, RepeatedSched1Up.StatusCode())
	})

	t.Run("Patch_WrongID_Status_StatusBadRequest", func(t *testing.T) {
		RepeatedSched1Up, err := apiClient.PatchSchedulesRepeatedRepeatedScheduleIDWithResponse(
			ctx,
			utils.RepeatedScheduleWrongID,
			utils.RepeatedSchedule1Request,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, RepeatedSched1Up.StatusCode())
	})

	t.Run("Get_WrongID_Status_StatusBadRequest", func(t *testing.T) {
		s1res, err := apiClient.GetSchedulesRepeatedRepeatedScheduleIDWithResponse(
			ctx,
			utils.RepeatedScheduleWrongID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, s1res.StatusCode())
	})

	t.Run("Delete_WrongID_Status_StatusBadRequest", func(t *testing.T) {
		resDelSite, err := apiClient.DeleteSchedulesRepeatedRepeatedScheduleIDWithResponse(
			ctx,
			utils.RepeatedScheduleWrongID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resDelSite.StatusCode())
	})

	// Verify partial updates
	utils.RepeatedSchedule1Request.TargetSiteId = siteCreated1.JSON201.SiteID
	RepeatedSched1 := CreateSchedRepeated(t, ctx, apiClient, utils.RepeatedSchedule1Request)

	t.Run("Put_WrongCron_StatusBadRequest", func(t *testing.T) {
		RepeatedSched1Up, err := apiClient.PutSchedulesRepeatedRepeatedScheduleIDWithResponse(
			ctx,
			*RepeatedSched1.JSON201.RepeatedScheduleID,
			utils.RepeatedMissingRequest,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, RepeatedSched1Up.StatusCode())
	})

	t.Run("Patch_WrongCron_StatusBadRequest", func(t *testing.T) {
		RepeatedSched1Up, err := apiClient.PatchSchedulesRepeatedRepeatedScheduleIDWithResponse(
			ctx,
			*RepeatedSched1.JSON201.RepeatedScheduleID,
			utils.RepeatedMissingRequest,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, RepeatedSched1Up.StatusCode())
	})

	log.Info().Msgf("End RepeatedSchedule Error tests")
}

func TestSchedRepeatedList(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	utils.Site1Request.RegionId = nil
	utils.Site1Request.OuId = nil
	siteCreated1 := CreateSite(t, ctx, apiClient, utils.Site1Request)
	utils.RepeatedSchedule1Request.TargetSiteId = siteCreated1.JSON201.SiteID

	totalItems := 10
	pageId := 1
	pageSize := 4

	for id := 0; id < totalItems; id++ {
		CreateSchedRepeated(t, ctx, apiClient, utils.RepeatedSchedule1Request)
	}

	// Checks if list resources return expected number of entries
	resList, err := apiClient.GetSchedulesRepeatedWithResponse(
		ctx,
		&api.GetSchedulesRepeatedParams{
			Offset:   &pageId,
			PageSize: &pageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.RepeatedSchedules), pageSize)
	assert.Equal(t, true, *resList.JSON200.HasNext)

	resList, err = apiClient.GetSchedulesRepeatedWithResponse(
		ctx,
		&api.GetSchedulesRepeatedParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, totalItems, len(*resList.JSON200.RepeatedSchedules))
	assert.Equal(t, false, *resList.JSON200.HasNext)
}

func TestSchedRepeatedListQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	utils.Site1Request.RegionId = nil
	utils.Site1Request.OuId = nil
	postRespSite1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.Site2Request.RegionId = nil
	utils.Site2Request.OuId = nil
	postRespSite2 := CreateSite(t, ctx, apiClient, utils.Site2Request)

	utils.RepeatedSchedule1Request.TargetSiteId = postRespSite1.JSON201.SiteID
	CreateSchedRepeated(t, ctx, apiClient, utils.RepeatedSchedule1Request)

	utils.RepeatedSchedule2Request.TargetSiteId = postRespSite2.JSON201.SiteID
	CreateSchedRepeated(t, ctx, apiClient, utils.RepeatedSchedule2Request)

	utils.RepeatedSchedule3Request.TargetSiteId = postRespSite2.JSON201.SiteID
	CreateSchedRepeated(t, ctx, apiClient, utils.RepeatedSchedule3Request)

	// Checks list of RepeatedSchedules with siteID 1
	resList, err := apiClient.GetSchedulesRepeatedWithResponse(
		ctx,
		&api.GetSchedulesRepeatedParams{
			SiteID: postRespSite1.JSON201.SiteID,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 1, len(*resList.JSON200.RepeatedSchedules))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	// Checks list of all RepeatedSchedules
	resList, err = apiClient.GetSchedulesRepeatedWithResponse(
		ctx,
		&api.GetSchedulesRepeatedParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 3, len(*resList.JSON200.RepeatedSchedules))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	// Checks list of RepeatedSchedules with SiteID 2
	resList, err = apiClient.GetSchedulesRepeatedWithResponse(
		ctx,
		&api.GetSchedulesRepeatedParams{
			SiteID: postRespSite2.JSON201.SiteID,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 2, len(*resList.JSON200.RepeatedSchedules))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetSchedulesRepeatedWithResponse(
		ctx,
		&api.GetSchedulesRepeatedParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, false, *resList.JSON200.HasNext)
}

func TestSchedRepeatedMaintenanceQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	region1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	region2 := CreateRegion(t, ctx, apiClient, utils.Region1Request)

	utils.Site1Request.RegionId = nil
	utils.Site1Request.OuId = nil
	utils.Site1Request.RegionId = region1.JSON201.RegionID
	site1 := CreateSite(t, ctx, apiClient, utils.Site1Request)
	utils.Site1Request.RegionId = nil

	utils.Site1Request.RegionId = region2.JSON201.RegionID
	site2 := CreateSite(t, ctx, apiClient, utils.Site1Request)
	utils.Site1Request.RegionId = nil

	utils.Host1Request.SiteId = site1.JSON201.SiteID
	host1 := CreateHost(t, ctx, apiClient, utils.Host1Request)
	utils.Host1Request.SiteId = nil

	host2 := CreateHost(t, ctx, apiClient, GetHostRequestWithRandomUUID())
	host3 := CreateHost(t, ctx, apiClient, GetHostRequestWithRandomUUID())

	utils.Host2Request.SiteId = site2.JSON201.SiteID
	host4 := CreateHost(t, ctx, apiClient, utils.Host2Request)
	utils.Host2Request.SiteId = nil

	utils.RepeatedScheduleAlwaysRequest.TargetSiteId = site1.JSON201.SiteID
	CreateSchedRepeated(t, ctx, apiClient, utils.RepeatedScheduleAlwaysRequest)
	utils.RepeatedScheduleAlwaysRequest.TargetSiteId = nil

	utils.RepeatedScheduleAlwaysRequest.TargetHostId = host2.JSON201.ResourceId
	CreateSchedRepeated(t, ctx, apiClient, utils.RepeatedScheduleAlwaysRequest)
	utils.RepeatedScheduleAlwaysRequest.TargetHostId = nil

	utils.RepeatedScheduleAlwaysRequest.TargetRegionId = region2.JSON201.ResourceId
	CreateSchedRepeated(t, ctx, apiClient, utils.RepeatedScheduleAlwaysRequest)
	utils.RepeatedScheduleAlwaysRequest.TargetRegionId = nil

	timestamp := time.Now()

	// Host1 should be in maintenance (it's in Site1, and we have maintenance window for it)
	AssertInMaintenance(t, ctx, apiClient, host1.JSON201.ResourceId, nil, nil, timestamp, 1, true)
	AssertInMaintenance(t, ctx, apiClient, nil, site1.JSON201.SiteID, nil, timestamp, 1, true)

	// Host2 should be in maintenance (it's directly in maintenance)
	AssertInMaintenance(t, ctx, apiClient, host2.JSON201.ResourceId, nil, nil, timestamp, 1, true)

	// Host3 should not be in maintenance
	AssertInMaintenance(t, ctx, apiClient, host3.JSON201.ResourceId, nil, nil, timestamp, 0, false)

	// Host4 should be in maintenance (it's in Region2, and we have maintenance window for it)
	AssertInMaintenance(t, ctx, apiClient, host4.JSON201.ResourceId, nil, nil, timestamp, 1, true)
	AssertInMaintenance(t, ctx, apiClient, nil, nil, region2.JSON201.RegionID, timestamp, 1, true)
}

func TestSchedRepeatedList_ListEmpty(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	resList, err := apiClient.GetSchedulesRepeatedWithResponse(
		ctx,
		&api.GetSchedulesRepeatedParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.NotNil(t, resList.JSON200.RepeatedSchedules)
	assert.Equal(t, 0, len(*resList.JSON200.RepeatedSchedules))
}

func TestSchedRepeated_cronjobValidationError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	utils.Site1Request.RegionId = nil
	utils.Site1Request.OuId = nil
	siteCreated1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.RepeatedScheduleCronReqErr.TargetSiteId = siteCreated1.JSON201.SiteID

	sched, err := apiClient.PostSchedulesRepeatedWithResponse(
		ctx,
		utils.RepeatedScheduleCronReqErr,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	assert.Equal(t, http.StatusBadRequest, sched.StatusCode())
	require.NoError(t, err)
}
