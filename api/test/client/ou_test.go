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

func setupOuHierarchy(
	t *testing.T,
	ctx context.Context,
	apiClient *api.ClientWithResponses,
) (*api.OU, *api.OU, *api.OU) {
	ou1 := CreateOu(t, ctx, apiClient, utils.OU1Request)

	utils.OU2Request.ParentOu = ou1.JSON201.OuID
	ou2 := CreateOu(t, ctx, apiClient, utils.OU2Request)
	utils.OU2Request.ParentOu = nil

	utils.OU3Request.ParentOu = ou2.JSON201.OuID
	ou3 := CreateOu(t, ctx, apiClient, utils.OU3Request)
	utils.OU3Request.ParentOu = nil

	return ou1.JSON201, ou2.JSON201, ou3.JSON201
}

func TestOU_MetadataInheritance(t *testing.T) {
	log.Info().Msgf("Begin OU Meta Inheritance")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	ou1, ou2, ou3 := setupOuHierarchy(t, ctx, apiClient)

	get1, err := apiClient.GetOusOuIDWithResponse(ctx, *ou1.OuID, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, get1.StatusCode())
	assert.Nil(t, get1.JSON200.ParentOu)
	assert.Equal(t, utils.MetadataOU1, *get1.JSON200.Metadata)

	get2, err := apiClient.GetOusOuIDWithResponse(ctx, *ou2.OuID, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, get2.StatusCode())
	assert.Equal(t, *ou1.OuID, *get2.JSON200.ParentOu)
	assert.Equal(t, utils.MetadataOU2, *get2.JSON200.Metadata)
	assert.Equal(t, api.Metadata{}, *get2.JSON200.InheritedMetadata) // OU2 does not inherit metadata from OU1, same keys

	get3, err := apiClient.GetOusOuIDWithResponse(ctx, *ou3.OuID, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, get2.StatusCode())
	assert.Equal(t, *ou2.OuID, *get3.JSON200.ParentOu)
	assert.Equal(t, utils.MetadataOU3, *get3.JSON200.Metadata)
	assert.Equal(t, utils.MetadataOU3Rendered, *get3.JSON200.InheritedMetadata)

	// Checks if Put updates Parent to empty string
	utils.OU3Request.ParentOu = &emptyString
	ou3Update, err := apiClient.PutOusOuIDWithResponse(ctx, *ou3.OuID, utils.OU3Request, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, ou3Update.StatusCode())
	get3, err = apiClient.GetOusOuIDWithResponse(ctx, *ou3.OuID, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, get2.StatusCode())
	assert.Nil(t, get3.JSON200.ParentOu)
	assert.Equal(t, api.Metadata{}, *get3.JSON200.InheritedMetadata) // Verifies if no metadata is inherited.

	// Checks if Patch updates Parent with empty string
	utils.OU2Request.ParentOu = &emptyString
	ou2Update, err := apiClient.PatchOusOuIDWithResponse(ctx, *ou2.OuID, utils.OU2Request, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, ou2Update.StatusCode())
	get2, err = apiClient.GetOusOuIDWithResponse(ctx, *ou2.OuID, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, get2.StatusCode())
	assert.Nil(t, get2.JSON200.ParentOu)
	assert.Equal(t, api.Metadata{}, *get2.JSON200.InheritedMetadata) // Verifies if no metadata is inherited.

	// Checks if Put updates Parent to empty string with empty character
	utils.OU3Request.ParentOu = &emptyStringWrong
	ou3Update, err = apiClient.PutOusOuIDWithResponse(ctx, *ou3.OuID, utils.OU3Request, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, ou3Update.StatusCode())

	// Checks if Patch updates Parent with empty string with empty character
	utils.OU2Request.ParentOu = &emptyStringWrong
	ou2Update, err = apiClient.PatchOusOuIDWithResponse(ctx, *ou2.OuID, utils.OU2Request, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, ou2Update.StatusCode())

	utils.OU3Request.ParentOu = nil
	utils.OU2Request.ParentOu = nil
	utils.OU1Request.ParentOu = nil
}

func TestOU_CreateGetDelete(t *testing.T) {
	log.Info().Msgf("Begin OU tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	// -------- Create OUs ----------------
	ou1 := CreateOu(t, ctx, apiClient, utils.OU1Request)
	ou2 := CreateOu(t, ctx, apiClient, utils.OU2Request)

	utils.OU3Request.ParentOu = ou2.JSON201.OuID
	ou3 := CreateOu(t, ctx, apiClient, utils.OU3Request)

	utils.Site1Request.RegionId = nil
	utils.Site1Request.OuId = ou1.JSON201.OuID
	s1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.Site2Request.RegionId = nil
	utils.Site2Request.OuId = ou2.JSON201.OuID
	s2 := CreateSite(t, ctx, apiClient, utils.Site2Request)

	get1, err := apiClient.GetOusOuIDWithResponse(ctx, *ou1.JSON201.OuID, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, get1.StatusCode())
	assert.Nil(t, get1.JSON200.ParentOu)

	get2, err := apiClient.GetOusOuIDWithResponse(ctx, *ou2.JSON201.OuID, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, get2.StatusCode())
	assert.Nil(t, get2.JSON200.ParentOu)

	get3, err := apiClient.GetOusOuIDWithResponse(ctx, *ou3.JSON201.OuID, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, get3.StatusCode())
	assert.Equal(t, *ou2.JSON201.OuID, *get3.JSON200.ParentOu)

	s1res, err := apiClient.GetSitesSiteIDWithResponse(
		ctx,
		*s1.JSON201.SiteID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, s1res.StatusCode())
	assert.Equal(t, *ou1.JSON201.OuID, *s1res.JSON200.Ou.ResourceId)

	s2res, err := apiClient.GetSitesSiteIDWithResponse(
		ctx,
		*s2.JSON201.SiteID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, s2res.StatusCode())
	assert.Equal(t, *ou2.JSON201.OuID, *s2res.JSON200.Ou.ResourceId)

	log.Info().Msgf("End OU tests")
}

func TestOU_Update(t *testing.T) {
	log.Info().Msgf("Begin OU Update tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	ou1 := CreateOu(t, ctx, apiClient, utils.OU1Request)
	assert.Equal(t, utils.OU1Name, ou1.JSON201.Name)

	OU1Get, err := apiClient.GetOusOuIDWithResponse(ctx, *ou1.JSON201.OuID, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, OU1Get.StatusCode())
	assert.Equal(t, utils.OU1Name, OU1Get.JSON200.Name)

	ou1Update, err := apiClient.PutOusOuIDWithResponse(ctx, *ou1.JSON201.OuID, utils.OU2Request, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, ou1Update.StatusCode())
	assert.Equal(t, utils.OU2Name, ou1Update.JSON200.Name)

	OU1GetUp, err := apiClient.GetOusOuIDWithResponse(ctx, *ou1.JSON201.OuID, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, OU1GetUp.StatusCode())
	assert.Equal(t, utils.OU2Name, OU1GetUp.JSON200.Name)

	log.Info().Msgf("End OU Update tests")
}

func TestOU_SiteUpdate(t *testing.T) {
	log.Info().Msgf("Begin OU Site Update tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	ou1 := CreateOu(t, ctx, apiClient, utils.OU1Request)
	assert.Equal(t, utils.OU1Name, ou1.JSON201.Name)

	utils.Site1Request.Region = nil
	utils.Site1Request.OuId = ou1.JSON201.OuID
	s1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.Site1RequestUpdate.Region = nil
	utils.Site1RequestUpdate.OuId = ou1.JSON201.OuID
	s1Up, err := apiClient.PutSitesSiteIDWithResponse(
		ctx,
		*s1.JSON201.SiteID,
		utils.Site1RequestUpdate,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, s1Up.StatusCode())

	s1res, err := apiClient.GetSitesSiteIDWithResponse(
		ctx,
		*s1.JSON201.SiteID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, s1res.StatusCode())

	utils.Site1RequestUpdatePatch.Region = nil
	utils.Site1RequestUpdatePatch.OuId = ou1.JSON201.OuID
	s1UpPatch, err := apiClient.PatchSitesSiteIDWithResponse(
		ctx,
		*s1.JSON201.SiteID,
		utils.Site1RequestUpdatePatch,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, s1UpPatch.StatusCode())

	s1res, err = apiClient.GetSitesSiteIDWithResponse(
		ctx,
		*s1.JSON201.SiteID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, s1res.StatusCode())
	assert.Equal(t, utils.Site2Name, *s1res.JSON200.Name)

	log.Info().Msgf("End OU Site Update tests")
}

func TestOU_Errors(t *testing.T) {
	log.Info().Msgf("Begin OU Error tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)
	if err != nil {
		t.Fatalf("new API client error %s", err.Error())
	}

	t.Run("Put_UnexistID_Status_NotFoundError", func(t *testing.T) {
		ou1Up, err := apiClient.PutOusOuIDWithResponse(
			ctx,
			utils.OUUnexistID,
			utils.OU1Request,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, ou1Up.StatusCode())
	})

	t.Run("Patch_UnexistID_Status_NotFoundError", func(t *testing.T) {
		ou1Up, err := apiClient.PatchOusOuIDWithResponse(
			ctx,
			utils.OUUnexistID,
			utils.OU1Request,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, ou1Up.StatusCode())
	})

	t.Run("Get_UnexistID_Status_NotFoundError", func(t *testing.T) {
		s1res, err := apiClient.GetOusOuIDWithResponse(
			ctx,
			utils.OUUnexistID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, s1res.StatusCode())
	})

	t.Run("Delete_UnexistID_Status_NotFoundError", func(t *testing.T) {
		resDelSite, err := apiClient.DeleteOusOuIDWithResponse(
			ctx,
			utils.OUUnexistID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resDelSite.StatusCode())
	})

	t.Run("Put_WrongID_Status_StatusBadRequest", func(t *testing.T) {
		ou1Up, err := apiClient.PutOusOuIDWithResponse(
			ctx,
			utils.OUWrongID,
			utils.OU1Request,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, ou1Up.StatusCode())
	})

	t.Run("Patch_WrongID_Status_StatusBadRequest", func(t *testing.T) {
		ou1Up, err := apiClient.PatchOusOuIDWithResponse(
			ctx,
			utils.OUWrongID,
			utils.OU1Request,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, ou1Up.StatusCode())
	})

	t.Run("Get_WrongID_Status_StatusBadRequest", func(t *testing.T) {
		s1res, err := apiClient.GetOusOuIDWithResponse(
			ctx,
			utils.OUWrongID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, s1res.StatusCode())
	})

	t.Run("Delete_WrongID_Status_StatusBadRequest", func(t *testing.T) {
		resDelSite, err := apiClient.DeleteOusOuIDWithResponse(
			ctx,
			utils.OUWrongID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resDelSite.StatusCode())
	})

	log.Info().Msgf("End OU Error tests")
}

func TestOUList(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	totalItems := 10
	pageId := 1
	pageSize := 4

	for id := 0; id < totalItems; id++ {
		CreateOu(t, ctx, apiClient, utils.OU1Request)
	}

	// Checks if list resources return expected number of entries
	resList, err := apiClient.GetOusWithResponse(
		ctx,
		&api.GetOusParams{
			Offset:   &pageId,
			PageSize: &pageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.OUs), pageSize)
	assert.Equal(t, true, *resList.JSON200.HasNext)

	resList, err = apiClient.GetOusWithResponse(
		ctx,
		&api.GetOusParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, totalItems, len(*resList.JSON200.OUs))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetOusWithResponse(
		ctx,
		&api.GetOusParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	require.NotNil(t, resList)
}

func TestOUListQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	postResp1 := CreateOu(t, ctx, apiClient, utils.OU1Request)

	utils.OU2Request.ParentOu = postResp1.JSON201.OuID
	CreateOu(t, ctx, apiClient, utils.OU2Request)

	utils.OU3Request.ParentOu = postResp1.JSON201.OuID
	CreateOu(t, ctx, apiClient, utils.OU3Request)

	// Checks list of OUs with Parent OU ID
	resList, err := apiClient.GetOusWithResponse(
		ctx,
		&api.GetOusParams{
			Parent: postResp1.JSON201.OuID,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 2, len(*resList.JSON200.OUs))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	// Checks list of all OUs
	resList, err = apiClient.GetOusWithResponse(
		ctx,
		&api.GetOusParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 3, len(*resList.JSON200.OUs))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	// Checks list of OUs without Parent OU ID
	emptyParent := "null"
	resList, err = apiClient.GetOusWithResponse(
		ctx,
		&api.GetOusParams{
			Parent: &emptyParent,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 1, len(*resList.JSON200.OUs))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetOusWithResponse(
		ctx,
		&api.GetOusParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, false, *resList.JSON200.HasNext)

	_, err = apiClient.GetOusWithResponse(
		ctx,
		&api.GetOusParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
}

func TestOUList_ListEmpty(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	resList, err := apiClient.GetOusWithResponse(
		ctx,
		&api.GetOusParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Empty(t, resList.JSON200.OUs)
}
