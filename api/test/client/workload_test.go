// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	"github.com/open-edge-platform/infra-core/api/test/utils"
)

func assertSameMemberIDs(t *testing.T, expectedMembers, actualMembers []api.WorkloadMember) {
	t.Helper()

	assert.Equal(t, len(expectedMembers), len(actualMembers))

	expectedIDs := make([]string, 0, len(expectedMembers))
	for _, em := range expectedMembers {
		expectedIDs = append(expectedIDs, *em.Member.InstanceID)
	}

	actualIDs := make([]string, 0, len(actualMembers))
	for _, am := range actualMembers {
		actualIDs = append(actualIDs, *am.Member.InstanceID)
	}

	assert.ElementsMatch(t, expectedIDs, actualIDs)
}

func TestCluster_CreateGetDelete(t *testing.T) {
	log.Info().Msgf("Begin workload cluster tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	h1 := CreateHost(t, ctx, apiClient, GetHostRequestWithRandomUUID())
	h2 := CreateHost(t, ctx, apiClient, GetHostRequestWithRandomUUID())
	h3 := CreateHost(t, ctx, apiClient, GetHostRequestWithRandomUUID())
	os := CreateOS(t, ctx, apiClient, utils.OSResource1Request)

	utils.Instance1Request.OsID = os.JSON201.OsResourceID
	utils.Instance1Request.HostID = h1.JSON201.ResourceId
	i1 := CreateInstance(t, ctx, apiClient, utils.Instance1Request)
	i1ID := *i1.JSON201.InstanceID

	utils.Instance1Request.OsID = os.JSON201.OsResourceID
	utils.Instance1Request.HostID = h2.JSON201.ResourceId
	i2 := CreateInstance(t, ctx, apiClient, utils.Instance1Request)
	i2ID := *i2.JSON201.InstanceID

	utils.Instance1Request.OsID = os.JSON201.OsResourceID
	utils.Instance1Request.HostID = h3.JSON201.ResourceId
	i3 := CreateInstance(t, ctx, apiClient, utils.Instance1Request)
	i3ID := *i3.JSON201.InstanceID

	w1 := CreateWorkload(t, ctx, apiClient, utils.WorkloadCluster1Request)
	w1ID := *w1.JSON201.WorkloadId
	w2 := CreateWorkload(t, ctx, apiClient, utils.WorkloadCluster2Request)
	w2ID := *w2.JSON201.WorkloadId

	// Create workload member (associate workload to hosts)
	m1w1 := CreateWorkloadMember(t, ctx, apiClient, api.WorkloadMember{
		InstanceId: &i1ID,
		WorkloadId: &w1ID,
		Kind:       api.WORKLOADMEMBERKINDCLUSTERNODE,
	})
	m2w1 := CreateWorkloadMember(t, ctx, apiClient, api.WorkloadMember{
		InstanceId: &i2ID,
		WorkloadId: &w1ID,
		Kind:       api.WORKLOADMEMBERKINDCLUSTERNODE,
	})
	m1w2 := CreateWorkloadMember(t, ctx, apiClient, api.WorkloadMember{
		InstanceId: &i3ID,
		WorkloadId: &w2ID,
		Kind:       api.WORKLOADMEMBERKINDCLUSTERNODE,
	})

	// Assert presence of workload with expected members
	getw1, err := apiClient.GetWorkloadsWorkloadIDWithResponse(
		ctx,
		w1ID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, getw1.StatusCode())
	assert.Equal(t, utils.WorkloadName1, *getw1.JSON200.Name)
	assert.Equal(t, utils.WorkloadCluster1Request.Kind, getw1.JSON200.Kind)
	assert.NotNil(t, getw1.JSON200.Members)
	assertSameMemberIDs(t, *getw1.JSON200.Members, []api.WorkloadMember{*m1w1.JSON201, *m2w1.JSON201})

	getw2, err := apiClient.GetWorkloadsWorkloadIDWithResponse(
		ctx,
		w2ID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, getw2.StatusCode())
	assert.Equal(t, utils.WorkloadName2, *getw2.JSON200.Name)
	assert.Equal(t, utils.WorkloadCluster2Request.Kind, getw2.JSON200.Kind)
	assert.NotNil(t, getw2.JSON200.Members)
	assertSameMemberIDs(t, *getw2.JSON200.Members, []api.WorkloadMember{*m1w2.JSON201})

	// Assert presence of workload members with expected instance and workload
	getm1w1, err := apiClient.GetWorkloadMembersWorkloadMemberIDWithResponse(
		ctx,
		*m1w1.JSON201.WorkloadMemberId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, getm1w1.StatusCode())
	assert.Equal(t, w1ID, *getm1w1.JSON200.Workload.ResourceId)
	assert.Equal(t, i1ID, *getm1w1.JSON200.Member.InstanceID)

	getm2w1, err := apiClient.GetWorkloadMembersWorkloadMemberIDWithResponse(
		ctx,
		*m2w1.JSON201.WorkloadMemberId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, getm1w1.StatusCode())
	assert.Equal(t, w1ID, *getm2w1.JSON200.Workload.ResourceId)
	assert.Equal(t, i2ID, *getm2w1.JSON200.Member.InstanceID)

	getm1w2, err := apiClient.GetWorkloadMembersWorkloadMemberIDWithResponse(
		ctx,
		*m1w2.JSON201.WorkloadMemberId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, getm1w2.StatusCode())
	assert.Equal(t, w2ID, *getm1w2.JSON200.Workload.ResourceId)
	assert.Equal(t, i3ID, *getm1w2.JSON200.Member.InstanceID)

	clearInstanceIDs()

	log.Info().Msgf("End workload cluster tests")
}

func TestWorkload_UpdatePut(t *testing.T) {
	log.Info().Msgf("Begin Workload Update tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	w1 := CreateWorkload(t, ctx, apiClient, utils.WorkloadCluster1Request)

	w1Update, err := apiClient.PutWorkloadsWorkloadIDWithResponse(
		ctx,
		*w1.JSON201.WorkloadId,
		utils.WorkloadCluster2Request,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, w1Update.StatusCode())

	w1GetUp, err := apiClient.GetWorkloadsWorkloadIDWithResponse(
		ctx,
		*w1.JSON201.WorkloadId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, w1GetUp.StatusCode())
	assert.Equal(t, *utils.WorkloadCluster2Request.Name, *w1GetUp.JSON200.Name)
	assert.Equal(t, utils.WorkloadCluster2Request.Status, w1GetUp.JSON200.Status)

	log.Info().Msgf("End Workload Update tests")
}

func TestWorkload_UpdatePatch(t *testing.T) {
	log.Info().Msgf("Begin Workload Update tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	w1 := CreateWorkload(t, ctx, apiClient, utils.WorkloadCluster1Request)

	w1Update, err := apiClient.PatchWorkloadsWorkloadIDWithResponse(
		ctx,
		*w1.JSON201.WorkloadId,
		api.Workload{
			Kind:   api.WORKLOADKINDCLUSTER,
			Status: &utils.WorkloadStatus3,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, w1Update.StatusCode())

	w1GetUp, err := apiClient.GetWorkloadsWorkloadIDWithResponse(
		ctx,
		*w1.JSON201.WorkloadId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, w1GetUp.StatusCode())
	assert.Equal(t, *utils.WorkloadCluster1Request.Name, *w1GetUp.JSON200.Name)
	assert.Equal(t, utils.WorkloadStatus3, *w1GetUp.JSON200.Status)

	log.Info().Msgf("End workload Update tests")
}

func TestWorkload_Errors(t *testing.T) {
	log.Info().Msgf("Begin Workload Error tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)
	if err != nil {
		t.Fatalf("new API client error %s", err.Error())
	}

	t.Run("Post_NoKind_BadRequest", func(t *testing.T) {
		w1Up, err := apiClient.PostWorkloadsWithResponse(
			ctx,
			utils.WorkloadNoKind,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		log.Info().Msgf("Error No Kind %s", w1Up.Body)
		assert.Equal(t, http.StatusBadRequest, w1Up.StatusCode())
	})

	t.Run("Put_UnexistID_Status_NotFoundError", func(t *testing.T) {
		w1Up, err := apiClient.PutWorkloadsWorkloadIDWithResponse(
			ctx,
			utils.WorkloadUnexistID,
			utils.WorkloadCluster1Request,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, w1Up.StatusCode())
	})

	t.Run("Patch_UnexistID_Status_NotFoundError", func(t *testing.T) {
		os1Up, err := apiClient.PatchWorkloadsWorkloadIDWithResponse(
			ctx,
			utils.WorkloadUnexistID,
			utils.WorkloadCluster1Request,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, os1Up.StatusCode())
	})

	t.Run("Get_UnexistID_Status_NotFoundError", func(t *testing.T) {
		w1res, err := apiClient.GetWorkloadsWorkloadIDWithResponse(
			ctx,
			utils.WorkloadUnexistID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, w1res.StatusCode())
	})

	t.Run("Delete_UnexistID_Status_NotFoundError", func(t *testing.T) {
		resDelW, err := apiClient.DeleteWorkloadsWorkloadIDWithResponse(
			ctx,
			utils.WorkloadUnexistID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resDelW.StatusCode())
	})

	t.Run("Put_WrongID_Status_StatusBadRequest", func(t *testing.T) {
		w1Up, err := apiClient.PutWorkloadsWorkloadIDWithResponse(
			ctx,
			utils.WorkloadWrongID,
			utils.WorkloadCluster1Request,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, w1Up.StatusCode())
	})

	t.Run("Patch_WrongID_Status_StatusBadRequest", func(t *testing.T) {
		w1Up, err := apiClient.PatchWorkloadsWorkloadIDWithResponse(
			ctx,
			utils.WorkloadWrongID,
			utils.WorkloadCluster1Request,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, w1Up.StatusCode())
	})

	t.Run("Get_WrongID_Status_StatusBadRequest", func(t *testing.T) {
		w1res, err := apiClient.GetOSResourcesOSResourceIDWithResponse(
			ctx,
			utils.WorkloadWrongID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, w1res.StatusCode())
	})

	t.Run("Delete_WrongID_Status_StatusBadRequest", func(t *testing.T) {
		resDelW, err := apiClient.DeleteWorkloadsWorkloadIDWithResponse(
			ctx,
			utils.WorkloadWrongID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resDelW.StatusCode())
	})
	log.Info().Msgf("End Workload Error tests")
}

func TestWorkloadMember_Errors(t *testing.T) {
	log.Info().Msgf("Begin WorkloadMember Error tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)
	if err != nil {
		t.Fatalf("new API client error %s", err.Error())
	}

	h1 := CreateHost(t, ctx, apiClient, GetHostRequestWithRandomUUID())
	h1ID := *h1.JSON201.ResourceId
	w1 := CreateWorkload(t, ctx, apiClient, utils.WorkloadCluster1Request)
	w1ID := *w1.JSON201.WorkloadId

	t.Run("Post_NoKind_BadRequest", func(t *testing.T) {
		mUp, err := apiClient.PostWorkloadMembersWithResponse(
			ctx,
			api.WorkloadMember{
				WorkloadId: &w1ID,
				InstanceId: &h1ID,
			},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		log.Info().Msgf("Error No Kind %s", mUp.Body)
		assert.Equal(t, http.StatusBadRequest, mUp.StatusCode())
	})

	t.Run("Post_NoWorkloadID_BadRequest", func(t *testing.T) {
		mUp, err := apiClient.PostWorkloadMembersWithResponse(
			ctx,
			api.WorkloadMember{
				Kind:       api.WORKLOADMEMBERKINDCLUSTERNODE,
				InstanceId: &h1ID,
			},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		log.Info().Msgf("Error No Workload ID %s", mUp.Body)
		assert.Equal(t, http.StatusBadRequest, mUp.StatusCode())
	})

	t.Run("Post_NoHostID_BadRequest", func(t *testing.T) {
		mUp, err := apiClient.PostWorkloadMembersWithResponse(
			ctx,
			api.WorkloadMember{
				WorkloadId: &w1ID,
				Kind:       api.WORKLOADMEMBERKINDCLUSTERNODE,
			},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		log.Info().Msgf("Error No Host ID %s", mUp.Body)
		assert.Equal(t, http.StatusBadRequest, mUp.StatusCode())
	})

	t.Run("Get_UnexistID_Status_NotFoundError", func(t *testing.T) {
		mRes, err := apiClient.GetWorkloadMembersWorkloadMemberIDWithResponse(
			ctx,
			utils.WorkloadMemberUnexistID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, mRes.StatusCode())
	})

	t.Run("Delete_UnexistID_Status_NotFoundError", func(t *testing.T) {
		resDelM, err := apiClient.DeleteWorkloadMembersWorkloadMemberIDWithResponse(
			ctx,
			utils.WorkloadMemberUnexistID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resDelM.StatusCode())
	})

	t.Run("Get_WrongID_Status_StatusBadRequest", func(t *testing.T) {
		mRes, err := apiClient.GetWorkloadMembersWorkloadMemberIDWithResponse(
			ctx,
			utils.WorkloadMemberWrongID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, mRes.StatusCode())
	})

	t.Run("Delete_WrongID_Status_StatusBadRequest", func(t *testing.T) {
		resDelM, err := apiClient.DeleteWorkloadMembersWorkloadMemberIDWithResponse(
			ctx,
			utils.WorkloadMemberWrongID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resDelM.StatusCode())
	})
	log.Info().Msgf("End Workload Member Error tests")
}

func TestWorkloadList(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	totalItems := 10
	pageId := 1
	pageSize := 4

	for id := 0; id < totalItems; id++ {
		CreateWorkload(t, ctx, apiClient, utils.WorkloadCluster2Request)
	}

	// Checks if list resources return expected number of entries
	resList, err := apiClient.GetWorkloadsWithResponse(
		ctx,
		&api.GetWorkloadsParams{
			Offset:   &pageId,
			PageSize: &pageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.Workloads), pageSize)
	assert.Equal(t, true, *resList.JSON200.HasNext)

	resList, err = apiClient.GetWorkloadsWithResponse(
		ctx,
		&api.GetWorkloadsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, totalItems, len(*resList.JSON200.Workloads))
	assert.Equal(t, false, *resList.JSON200.HasNext)
}

func TestWorkloadMemberList(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	totalItems := 10
	pageId := 1
	pageSize := 4

	workload := CreateWorkload(t, ctx, apiClient, utils.WorkloadCluster1Request)
	os := CreateOS(t, ctx, apiClient, utils.OSResource1Request)

	for id := 0; id < totalItems; id++ {
		host := CreateHost(t, ctx, apiClient, GetHostRequestWithRandomUUID())

		utils.Instance1Request.OsID = os.JSON201.OsResourceID
		utils.Instance1Request.HostID = host.JSON201.ResourceId
		instance := CreateInstance(t, ctx, apiClient, utils.Instance1Request)

		CreateWorkloadMember(t, ctx, apiClient, api.WorkloadMember{
			InstanceId: instance.JSON201.InstanceID,
			WorkloadId: workload.JSON201.WorkloadId,
			Kind:       api.WORKLOADMEMBERKINDCLUSTERNODE,
		})
	}

	// Checks if list resources return expected number of entries
	resList, err := apiClient.GetWorkloadMembersWithResponse(
		ctx,
		&api.GetWorkloadMembersParams{
			Offset:   &pageId,
			PageSize: &pageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.WorkloadMembers), pageSize)
	assert.Equal(t, true, *resList.JSON200.HasNext)

	resList, err = apiClient.GetWorkloadMembersWithResponse(
		ctx,
		&api.GetWorkloadMembersParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, totalItems, len(*resList.JSON200.WorkloadMembers))
	assert.Equal(t, false, *resList.JSON200.HasNext)
}

func TestWorkloadList_ListEmpty(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	resList, err := apiClient.GetWorkloadsWithResponse(
		ctx,
		&api.GetWorkloadsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Empty(t, resList.JSON200.Workloads)
}

func TestWorkloadMemberList_ListEmpty(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	resList, err := apiClient.GetWorkloadMembersWithResponse(
		ctx,
		&api.GetWorkloadMembersParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Empty(t, resList.JSON200.WorkloadMembers)
}
