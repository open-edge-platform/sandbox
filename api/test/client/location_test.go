// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	"github.com/open-edge-platform/infra-core/api/test/utils"
)

func setupRegionHierarchy(
	t *testing.T,
	ctx context.Context,
	apiClient *api.ClientWithResponses,
) (*api.Region, *api.Region, *api.Region) {
	r1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)

	utils.Region2Request.ParentId = r1.JSON201.RegionID
	r2 := CreateRegion(t, ctx, apiClient, utils.Region2Request)
	utils.Region2Request.ParentId = nil

	utils.Region3Request.ParentId = r2.JSON201.RegionID
	r3 := CreateRegion(t, ctx, apiClient, utils.Region3Request)
	utils.Region3Request.ParentId = nil

	return r1.JSON201, r2.JSON201, r3.JSON201
}

func TestLocation_Metadata(t *testing.T) {
	log.Info().Msgf("Begin Location Metadata Validation OK/NOK tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	region, err := apiClient.PostRegionsWithResponse(ctx, utils.Region1RequestMetadataNOK, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	assert.Equal(t, http.StatusBadRequest, region.StatusCode())
	require.NoError(t, err)

	region, err = apiClient.PostRegionsWithResponse(ctx, utils.Region1RequestMetadataOK, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	assert.Equal(t, http.StatusCreated, region.StatusCode())
	require.NoError(t, err)

	t.Cleanup(func() { DeleteRegion(t, context.Background(), apiClient, *region.JSON201.RegionID) })
}

func TestLocation_MetadataInheritance(t *testing.T) {
	log.Info().Msgf("Begin Location Meta Inheritance tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	r1, r2, r3 := setupRegionHierarchy(t, ctx, apiClient)

	_, ou2, ou3 := setupOuHierarchy(t, ctx, apiClient)

	utils.Site1Request.RegionId = r3.RegionID
	utils.Site1Request.OuId = ou3.OuID
	s1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.Site2Request.RegionId = r2.RegionID
	utils.Site2Request.OuId = ou2.OuID
	s2 := CreateSite(t, ctx, apiClient, utils.Site2Request)
	utils.Site2Request.Region = nil
	utils.Site2Request.Ou = nil

	getr1, err := apiClient.GetRegionsRegionIDWithResponse(ctx, *r1.RegionID, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, getr1.StatusCode())
	assert.Nil(t, getr1.JSON200.ParentId)
	assert.Equal(t, utils.MetadataR1, *getr1.JSON200.Metadata)
	assert.Equal(t, api.Metadata{}, *getr1.JSON200.InheritedMetadata)

	getr2, err := apiClient.GetRegionsRegionIDWithResponse(ctx, *r2.RegionID, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, getr2.StatusCode())
	assert.Equal(t, *r1.RegionID, *getr2.JSON200.ParentId)
	assert.Equal(t, utils.MetadataR2, *getr2.JSON200.Metadata)
	assert.Equal(t, api.Metadata{}, *getr2.JSON200.InheritedMetadata)

	getr3, err := apiClient.GetRegionsRegionIDWithResponse(ctx, *r3.RegionID, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, getr3.StatusCode())
	assert.Equal(t, *r2.RegionID, *getr3.JSON200.ParentId)
	assert.Equal(t, utils.MetadataR3, *getr3.JSON200.Metadata)
	assert.Equal(t, utils.MetadataR3Inherited, *getr3.JSON200.InheritedMetadata)

	gets1, err := apiClient.GetSitesSiteIDWithResponse(ctx, *s1.JSON201.SiteID, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, gets1.StatusCode())
	assert.Equal(t, *r3.RegionID, *gets1.JSON200.Region.ResourceId)
	assert.Equal(t, *ou3.OuID, *gets1.JSON200.Ou.ResourceId)
	assert.Equal(t, 2, len(*gets1.JSON200.InheritedMetadata.Location))
	log.Info().Msgf("%s", *gets1.JSON200.InheritedMetadata.Location)
	assert.True(
		t,
		ListMetadataContains(*gets1.JSON200.InheritedMetadata.Location, "examplekey2", "r2"),
	)
	assert.True(
		t,
		ListMetadataContains(*gets1.JSON200.InheritedMetadata.Location, "examplekey", "r3"),
	)
	assert.Equal(
		t,
		api.Metadata{{Key: "examplekey3", Value: "ou3"}},
		*gets1.JSON200.InheritedMetadata.Ou,
	)

	gets2, err := apiClient.GetSitesSiteIDWithResponse(ctx, *s2.JSON201.SiteID, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, gets2.StatusCode())
	assert.Equal(t, *r2.RegionID, *gets2.JSON200.Region.ResourceId)
	assert.Equal(t, *ou2.OuID, *gets2.JSON200.Ou.ResourceId)
	assert.Equal(
		t,
		api.Metadata{{Key: "examplekey", Value: "r2"}},
		*gets2.JSON200.InheritedMetadata.Location,
	)
	assert.Equal(t, api.Metadata{}, *gets2.JSON200.InheritedMetadata.Ou)
}

func TestLocation_CreateGetDelete(t *testing.T) {
	log.Info().Msgf("Begin Location RegionSite tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	r1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	r2 := CreateRegion(t, ctx, apiClient, utils.Region2Request)

	utils.Site1Request.RegionId = nil
	utils.Site1Request.OuId = nil
	s1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.Site2Request.RegionId = nil
	utils.Site2Request.OuId = nil
	s2 := CreateSite(t, ctx, apiClient, utils.Site2Request)

	sites1, err := apiClient.GetRegionsRegionIDWithResponse(
		ctx,
		*r1.JSON201.RegionID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, sites1.StatusCode())

	sites2, err := apiClient.GetRegionsRegionIDWithResponse(
		ctx,
		*r2.JSON201.RegionID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, sites2.StatusCode())

	s1res, err := apiClient.GetSitesSiteIDWithResponse(
		ctx,
		*s1.JSON201.SiteID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, s1res.StatusCode())

	s2res, err := apiClient.GetSitesSiteIDWithResponse(
		ctx,
		*s2.JSON201.SiteID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, s2res.StatusCode())

	log.Info().Msgf("End Location RegionSite tests")
}

func TestLocation_RegionUpdate(t *testing.T) {
	log.Info().Msgf("Begin Location Region Update tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	r1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	assert.Equal(t, utils.Region1Name, *r1.JSON201.Name)

	r2 := CreateRegion(t, ctx, apiClient, utils.Region2Request)
	assert.Equal(t, utils.Region2Name, *r2.JSON201.Name)

	region1Get, err := apiClient.GetRegionsRegionIDWithResponse(
		ctx,
		*r1.JSON201.RegionID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, region1Get.StatusCode())
	assert.Equal(t, utils.Region1Name, *region1Get.JSON200.Name)

	r1Update, err := apiClient.PutRegionsRegionIDWithResponse(
		ctx,
		*r1.JSON201.RegionID,
		utils.Region2Request,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r1Update.StatusCode())
	assert.Equal(t, utils.Region2Name, *r1Update.JSON200.Name)

	region1GetUp, err := apiClient.GetRegionsRegionIDWithResponse(
		ctx,
		*r1.JSON201.RegionID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, region1GetUp.StatusCode())
	assert.Equal(t, utils.Region2Name, *region1GetUp.JSON200.Name)

	// Updates using Put r1 Parent with r2 regionID
	utils.Region1Request.ParentId = r2.JSON201.RegionID
	r1Update, err = apiClient.PutRegionsRegionIDWithResponse(
		ctx,
		*r1.JSON201.RegionID,
		utils.Region1Request,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r1Update.StatusCode())

	// Gets r1 and checks Parent equals to r2 regionID
	region1GetUp, err = apiClient.GetRegionsRegionIDWithResponse(
		ctx,
		*r1.JSON201.RegionID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, region1GetUp.StatusCode())
	assert.Equal(t, utils.Region1Name, *region1GetUp.JSON200.Name)
	assert.Equal(t, *r2.JSON201.RegionID, *region1GetUp.JSON200.ParentId)

	// Updates using Put r1 Parent with empty string
	utils.Region1Request.ParentId = &emptyString
	r1Update, err = apiClient.PutRegionsRegionIDWithResponse(
		ctx,
		*r1.JSON201.RegionID,
		utils.Region1Request,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r1Update.StatusCode())

	// Gets r1 and checks Parent equals to empty string
	region1GetUp, err = apiClient.GetRegionsRegionIDWithResponse(
		ctx,
		*r1.JSON201.RegionID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, region1GetUp.StatusCode())
	assert.Equal(t, utils.Region1Name, *region1GetUp.JSON200.Name)
	assert.Nil(t, region1GetUp.JSON200.ParentId)

	// Updates using Patch r1 Parent with r2 regionID
	utils.Region1Request.ParentId = r2.JSON201.RegionID
	r1UpdatePatch, err := apiClient.PatchRegionsRegionIDWithResponse(
		ctx,
		*r1.JSON201.RegionID,
		utils.Region1Request,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r1UpdatePatch.StatusCode())

	// Gets r1 and checks Parent equals to r2 regionID
	region1GetUp, err = apiClient.GetRegionsRegionIDWithResponse(
		ctx,
		*r1.JSON201.RegionID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, region1GetUp.StatusCode())
	assert.Equal(t, utils.Region1Name, *region1GetUp.JSON200.Name)
	assert.Equal(t, *r2.JSON201.RegionID, *region1GetUp.JSON200.ParentId)

	// Updates using Patch r1 Parent with empty string
	utils.Region1Request.ParentId = &emptyString
	r1UpdatePatch, err = apiClient.PatchRegionsRegionIDWithResponse(
		ctx,
		*r1.JSON201.RegionID,
		utils.Region1Request,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r1UpdatePatch.StatusCode())

	// Gets r1 and checks Parent equals to empty string
	region1GetUp, err = apiClient.GetRegionsRegionIDWithResponse(
		ctx,
		*r1.JSON201.RegionID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, region1GetUp.StatusCode())
	assert.Equal(t, utils.Region1Name, *region1GetUp.JSON200.Name)
	assert.Nil(t, region1GetUp.JSON200.ParentId)

	// Check for BadReqeuest error in case Parent contains empty character in Put
	utils.Region1Request.ParentId = &emptyStringWrong
	r1Update, err = apiClient.PutRegionsRegionIDWithResponse(
		ctx,
		*r1.JSON201.RegionID,
		utils.Region1Request,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, r1Update.StatusCode())

	// Check for BadReqeuest error in case Parent contains empty character in Patch
	utils.Region1Request.ParentId = &emptyStringWrong
	r1UpdatePatch, err = apiClient.PatchRegionsRegionIDWithResponse(
		ctx,
		*r1.JSON201.RegionID,
		utils.Region1Request,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, r1UpdatePatch.StatusCode())

	// Cleanup
	utils.Region1Request.ParentId = nil
	log.Info().Msgf("End Location Region Update tests")
}

func TestLocation_SiteUpdate(t *testing.T) {
	log.Info().Msgf("Begin Location Site Update tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	r1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	assert.Equal(t, utils.Region1Name, *r1.JSON201.Name)

	r2 := CreateRegion(t, ctx, apiClient, utils.Region2Request)
	assert.Equal(t, utils.Region2Name, *r2.JSON201.Name)

	utils.Site1Request.RegionId = r1.JSON201.RegionID
	s1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	ou1 := CreateOu(t, ctx, apiClient, utils.OU1Request)

	// Updates site using Put, sets Region to r1 regionID and verifies it
	utils.Site1RequestUpdate.RegionId = r1.JSON201.RegionID
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
	assert.Equal(t, *r1.JSON201.RegionID, *s1res.JSON200.Region.ResourceId)

	// Updates site using Put, sets Region to emptyString and verifies it
	utils.Site1RequestUpdate.RegionId = &emptyString
	s1Up, err = apiClient.PutSitesSiteIDWithResponse(
		ctx,
		*s1.JSON201.SiteID,
		utils.Site1RequestUpdate,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, s1Up.StatusCode())

	s1res, err = apiClient.GetSitesSiteIDWithResponse(
		ctx,
		*s1.JSON201.SiteID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, s1res.StatusCode())
	assert.Nil(t, s1res.JSON200.Region)

	// Updates site using Patch, sets Region to r2 regionID and verifies it
	utils.Site1RequestUpdatePatch.RegionId = r2.JSON201.RegionID
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
	assert.Equal(t, *r2.JSON201.RegionID, *s1res.JSON200.Region.ResourceId)

	// Updates site using Patch, sets Region to emptyString and verifies it
	utils.Site1RequestUpdatePatch.RegionId = &emptyString
	s1UpPatch, err = apiClient.PatchSitesSiteIDWithResponse(
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
	assert.Nil(t, s1res.JSON200.Region)

	// Updates site using Put, sets Ou to ou1 and verifies it
	utils.Site1RequestUpdate.OuId = ou1.JSON201.OuID
	s1Up, err = apiClient.PutSitesSiteIDWithResponse(
		ctx,
		*s1.JSON201.SiteID,
		utils.Site1RequestUpdate,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, s1Up.StatusCode())

	s1res, err = apiClient.GetSitesSiteIDWithResponse(
		ctx,
		*s1.JSON201.SiteID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, s1res.StatusCode())
	assert.Equal(t, *ou1.JSON201.OuID, *s1res.JSON200.Ou.ResourceId)

	// Updates site using Put, sets Ou to emptyString and verifies it
	utils.Site1RequestUpdate.OuId = &emptyString
	s1Up, err = apiClient.PutSitesSiteIDWithResponse(
		ctx,
		*s1.JSON201.SiteID,
		utils.Site1RequestUpdate,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, s1Up.StatusCode())

	s1res, err = apiClient.GetSitesSiteIDWithResponse(
		ctx,
		*s1.JSON201.SiteID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, s1res.StatusCode())
	assert.Nil(t, s1res.JSON200.Ou)

	// Updates site using Patch, sets Ou to ou1 and verifies it
	utils.Site1RequestUpdatePatch.OuId = ou1.JSON201.OuID
	s1UpPatch, err = apiClient.PatchSitesSiteIDWithResponse(
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
	assert.Equal(t, *ou1.JSON201.OuID, *s1res.JSON200.Ou.ResourceId)

	// Updates site using Patch, sets Ou to emptyString and verifies it
	utils.Site1RequestUpdatePatch.OuId = &emptyString
	s1UpPatch, err = apiClient.PatchSitesSiteIDWithResponse(
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
	assert.Nil(t, s1res.JSON200.Ou)

	// Updates site using Put and Patch, sets Ou to wrong emptyString and verifies
	// expected error BadRequest
	utils.Site1RequestUpdate.OuId = &emptyStringWrong
	s1Up, err = apiClient.PutSitesSiteIDWithResponse(
		ctx,
		*s1.JSON201.SiteID,
		utils.Site1RequestUpdate,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, s1Up.StatusCode())

	utils.Site1RequestUpdatePatch.OuId = &emptyStringWrong
	s1UpPatch, err = apiClient.PatchSitesSiteIDWithResponse(
		ctx,
		*s1.JSON201.SiteID,
		utils.Site1RequestUpdatePatch,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, s1UpPatch.StatusCode())

	// Sets OU in Update resources OU to nil
	utils.Site1RequestUpdatePatch.OuId = nil
	utils.Site1RequestUpdate.OuId = nil

	// Updates site using Put and Patch, sets Region to wrong emptyString and verifies
	// expected error BadRequest
	utils.Site1RequestUpdate.RegionId = &emptyStringWrong
	s1Up, err = apiClient.PutSitesSiteIDWithResponse(
		ctx,
		*s1.JSON201.SiteID,
		utils.Site1RequestUpdate,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, s1Up.StatusCode())

	utils.Site1RequestUpdatePatch.RegionId = &emptyStringWrong
	s1UpPatch, err = apiClient.PatchSitesSiteIDWithResponse(
		ctx,
		*s1.JSON201.SiteID,
		utils.Site1RequestUpdatePatch,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, s1UpPatch.StatusCode())

	// Sets region in Update resources OU to nil
	utils.Site1RequestUpdatePatch.RegionId = nil
	utils.Site1RequestUpdate.RegionId = nil

	log.Info().Msgf("End Location Site Update tests")
}

func TestLocation_RegionErrors(t *testing.T) {
	log.Info().Msgf("Begin Location Region Error tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)
	if err != nil {
		t.Fatalf("new API client error %s", err.Error())
	}

	t.Run("Put_UnexistID_Status_NotFoundError", func(t *testing.T) {
		r1Up, err := apiClient.PutRegionsRegionIDWithResponse(
			ctx,
			utils.RegionUnexistID,
			utils.Region1Request,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, r1Up.StatusCode())
	})

	t.Run("Patch_UnexistID_Status_NotFoundError", func(t *testing.T) {
		r1Up, err := apiClient.PatchRegionsRegionIDWithResponse(
			ctx,
			utils.RegionUnexistID,
			utils.Region1Request,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, r1Up.StatusCode())
	})

	t.Run("Get_UnexistID_Status_NotFoundError", func(t *testing.T) {
		s1res, err := apiClient.GetRegionsRegionIDWithResponse(
			ctx,
			utils.RegionUnexistID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, s1res.StatusCode())
	})

	t.Run("Delete_UnexistID_Status_NotFoundError", func(t *testing.T) {
		resDelSite, err := apiClient.DeleteRegionsRegionIDWithResponse(
			ctx,
			utils.RegionUnexistID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resDelSite.StatusCode())
	})

	t.Run("Put_WrongID_Status_StatusUnprocessableEntity", func(t *testing.T) {
		r1Up, err := apiClient.PutRegionsRegionIDWithResponse(
			ctx,
			utils.RegionWrongID,
			utils.Region1Request,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, r1Up.StatusCode())
	})

	t.Run("Patch_WrongID_Status_StatusUnprocessableEntity", func(t *testing.T) {
		r1Up, err := apiClient.PatchRegionsRegionIDWithResponse(
			ctx,
			utils.RegionWrongID,
			utils.Region1Request,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, r1Up.StatusCode())
	})

	t.Run("Get_WrongID_Status_StatusUnprocessableEntity", func(t *testing.T) {
		s1res, err := apiClient.GetRegionsRegionIDWithResponse(
			ctx,
			utils.RegionWrongID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, s1res.StatusCode())
	})

	t.Run("Delete_WrongID_Status_StatusUnprocessableEntity", func(t *testing.T) {
		resDelSite, err := apiClient.DeleteRegionsRegionIDWithResponse(
			ctx,
			utils.RegionWrongID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resDelSite.StatusCode())
	})

	log.Info().Msgf("End Location Region Error tests")
}

func TestLocation_SiteErrors(t *testing.T) {
	log.Info().Msgf("Begin Location Site Error tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	t.Run("Put_UnexistID_Status_NotFoundError", func(t *testing.T) {
		s1Up, err := apiClient.PutSitesSiteIDWithResponse(
			ctx,
			utils.SiteUnexistID,
			utils.Site1RequestUpdate,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, s1Up.StatusCode())
	})

	t.Run("Get_UnexistID_Status_NotFoundError", func(t *testing.T) {
		s1res, err := apiClient.GetSitesSiteIDWithResponse(
			ctx,
			utils.SiteUnexistID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, s1res.StatusCode())
	})

	t.Run("Delete_UnexistID_Status_NotFoundError", func(t *testing.T) {
		resDelSite, err := apiClient.DeleteSitesSiteIDWithResponse(
			ctx,
			utils.SiteUnexistID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resDelSite.StatusCode())
	})

	t.Run("Put_WrongID_Status_StatusUnprocessableEntity", func(t *testing.T) {
		s1Up, err := apiClient.PutSitesSiteIDWithResponse(
			ctx,
			utils.SiteWrongID,
			utils.Site1RequestUpdate,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, s1Up.StatusCode())
	})

	t.Run("Get_WrongID_Status_StatusUnprocessableEntity", func(t *testing.T) {
		s1res, err := apiClient.GetSitesSiteIDWithResponse(
			ctx,
			utils.SiteWrongID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, s1res.StatusCode())
	})

	t.Run("Delete_WrongID_Status_StatusUnprocessableEntity", func(t *testing.T) {
		resDelSite, err := apiClient.DeleteSitesSiteIDWithResponse(
			ctx,
			utils.SiteWrongID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resDelSite.StatusCode())
	})
	log.Info().Msgf("End Location Site Error tests")
}

func TestRegionList(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	totalItems := 10
	pageId := 1
	pageSize := 4

	for id := 0; id < totalItems; id++ {
		CreateRegion(t, ctx, apiClient, utils.Region1Request)
	}

	// Checks if list resources return expected number of entries
	resList, err := apiClient.GetRegionsWithResponse(
		ctx,
		&api.GetRegionsParams{
			Offset:   &pageId,
			PageSize: &pageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.Regions), pageSize)
	assert.Equal(t, true, *resList.JSON200.HasNext)

	resList, err = apiClient.GetRegionsWithResponse(
		ctx,
		&api.GetRegionsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, totalItems, len(*resList.JSON200.Regions))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetRegionsWithResponse(
		ctx,
		&api.GetRegionsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	require.NotNil(t, resList)
}

func TestLocation_RegionListQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	postResp1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)

	utils.Region2Request.ParentId = postResp1.JSON201.RegionID
	CreateRegion(t, ctx, apiClient, utils.Region2Request)

	utils.Region3Request.ParentId = postResp1.JSON201.RegionID
	CreateRegion(t, ctx, apiClient, utils.Region3Request)

	// Checks Regions with Parent Region ID
	resList, err := apiClient.GetRegionsWithResponse(
		ctx,
		&api.GetRegionsParams{
			Parent: postResp1.JSON201.RegionID,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 2, len(*resList.JSON200.Regions))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	// Checks all Regions
	resList, err = apiClient.GetRegionsWithResponse(
		ctx,
		&api.GetRegionsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 3, len(*resList.JSON200.Regions))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	// Checks Regions without Parent Region ID
	emptyParent := "null"
	resList, err = apiClient.GetRegionsWithResponse(
		ctx,
		&api.GetRegionsParams{
			Parent: &emptyParent,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 1, len(*resList.JSON200.Regions))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetRegionsWithResponse(
		ctx,
		&api.GetRegionsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, false, *resList.JSON200.HasNext)

	resList, err = apiClient.GetRegionsWithResponse(
		ctx,
		&api.GetRegionsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	require.NotNil(t, resList)
}

func TestLocation_SiteList(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	totalItems := 10
	pageId := 1
	pageSize := 4

	for id := 0; id < totalItems; id++ {
		CreateSite(t, ctx, apiClient, utils.SiteListRequest)
	}

	// Checks if list resources return expected number of entries
	resSiteList, err := apiClient.GetSitesWithResponse(
		ctx,
		&api.GetSitesParams{
			Offset:   &pageId,
			PageSize: &pageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resSiteList.StatusCode())
	assert.Equal(t, len(*resSiteList.JSON200.Sites), pageSize)
	assert.Equal(t, true, *resSiteList.JSON200.HasNext)

	resSiteList, err = apiClient.GetSitesWithResponse(
		ctx,
		&api.GetSitesParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resSiteList.StatusCode())
	assert.Equal(t, totalItems, len(*resSiteList.JSON200.Sites))
	assert.Equal(t, false, *resSiteList.JSON200.HasNext)
}

func TestLocation_SiteListQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	postRespRegion := CreateRegion(t, ctx, apiClient, utils.Region1Request)

	postRespOU := CreateOu(t, ctx, apiClient, utils.OU1Request)

	CreateSite(t, ctx, apiClient, utils.SiteListRequest1)

	utils.SiteListRequest2.OuId = postRespOU.JSON201.OuID
	CreateSite(t, ctx, apiClient, utils.SiteListRequest2)

	utils.SiteListRequest3.RegionId = postRespRegion.JSON201.RegionID
	CreateSite(t, ctx, apiClient, utils.SiteListRequest3)

	// Checks query to all sites
	resSiteList, err := apiClient.GetSitesWithResponse(
		ctx,
		&api.GetSitesParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resSiteList.StatusCode())
	assert.Equal(t, 3, len(*resSiteList.JSON200.Sites))
	assert.Equal(t, false, *resSiteList.JSON200.HasNext)

	// Checks query to sites with region ID
	resSiteList, err = apiClient.GetSitesWithResponse(
		ctx,
		&api.GetSitesParams{
			RegionID: postRespRegion.JSON201.RegionID,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resSiteList.StatusCode())
	assert.Equal(t, 1, len(*resSiteList.JSON200.Sites))
	assert.Equal(t, false, *resSiteList.JSON200.HasNext)

	// Checks query to sites without region ID
	emptyRegion := "null"
	resSiteList, err = apiClient.GetSitesWithResponse(
		ctx,
		&api.GetSitesParams{
			RegionID: &emptyRegion,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resSiteList.StatusCode())
	assert.Equal(t, 2, len(*resSiteList.JSON200.Sites))
	assert.Equal(t, false, *resSiteList.JSON200.HasNext)

	// Checks query to sites with OU ID
	resSiteList, err = apiClient.GetSitesWithResponse(
		ctx,
		&api.GetSitesParams{
			OuID: postRespOU.JSON201.OuID,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resSiteList.StatusCode())
	assert.Equal(t, 1, len(*resSiteList.JSON200.Sites))
	assert.Equal(t, false, *resSiteList.JSON200.HasNext)

	// Checks query to sites without OU ID
	emptyOU := "null"
	resSiteList, err = apiClient.GetSitesWithResponse(
		ctx,
		&api.GetSitesParams{
			OuID: &emptyOU,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resSiteList.StatusCode())
	assert.Equal(t, 2, len(*resSiteList.JSON200.Sites))
	assert.Equal(t, false, *resSiteList.JSON200.HasNext)

	resSiteList, err = apiClient.GetSitesWithResponse(
		ctx,
		&api.GetSitesParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resSiteList.StatusCode())
	assert.Equal(t, false, *resSiteList.JSON200.HasNext)
}

func TestLocation_ListEmpty(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	resRegionList, err := apiClient.GetRegionsWithResponse(
		ctx,
		&api.GetRegionsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resRegionList.StatusCode())
	assert.Empty(t, resRegionList.JSON200.Regions)

	resSiteList, err := apiClient.GetSitesWithResponse(
		ctx,
		&api.GetSitesParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resSiteList.StatusCode())
	assert.Empty(t, resSiteList.JSON200.Sites)
}

func TestLocation_Filter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	// create regions
	r1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	utils.Region2Request.ParentId = nil
	r2 := CreateRegion(t, ctx, apiClient, utils.Region2Request)

	// create sites with region
	s1req := utils.Site1Request
	s1req.RegionId = r1.JSON201.ResourceId
	s1req.RegionId = r1.JSON201.ResourceId
	s1 := CreateSite(t, ctx, apiClient, s1req)

	s2req := utils.Site2Request
	s2req.RegionId = r2.JSON201.ResourceId
	s2 := CreateSite(t, ctx, apiClient, s2req)

	// filter- site->region->resource-id
	filter := fmt.Sprintf("region.resourceId=\"%s\"", *r1.JSON201.ResourceId)
	sites1, err := apiClient.GetSitesWithResponse(
		ctx,
		&api.GetSitesParams{Filter: &filter},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, sites1.StatusCode())
	assert.Equal(t, 1, *sites1.JSON200.TotalElements)
	assert.Equal(t, *s1.JSON201.Region.ResourceId, *r1.JSON201.ResourceId)

	// filter- site->region->resource-id
	filter = fmt.Sprintf("region.resourceId=\"%s\"", *r2.JSON201.ResourceId)
	sites2, err := apiClient.GetSitesWithResponse(
		ctx,
		&api.GetSitesParams{Filter: &filter},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, sites2.StatusCode())
	assert.Equal(t, 1, *sites2.JSON200.TotalElements)
	assert.Equal(t, *s2.JSON201.Region.ResourceId, *r2.JSON201.ResourceId)

	// filter- region with ShotTotalSites: region1 and region2 has not parent and 1 site each
	filter = fmt.Sprintf(`NOT has(%s)`, "parent_region")
	regions, err := apiClient.GetRegionsWithResponse(
		ctx,
		&api.GetRegionsParams{
			ShowTotalSites: &showSites,
			Filter:         &filter,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, sites2.StatusCode())
	assert.Equal(t, 2, *regions.JSON200.TotalElements)
	assert.Equal(t, 2, len(*regions.JSON200.Regions))
	region1, region2 := (*regions.JSON200.Regions)[0], (*regions.JSON200.Regions)[1]
	assert.Equal(t, 1, *region1.TotalSites)
	assert.Equal(t, 1, *region2.TotalSites)
}

func TestLocation_FilterSites(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	// create regions
	r1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	utils.Region2Request.ParentId = nil
	r2 := CreateRegion(t, ctx, apiClient, utils.Region2Request)

	// create sites with region
	s1req := utils.Site1Request
	s1req.RegionId = r1.JSON201.ResourceId
	s1req.RegionId = r1.JSON201.ResourceId
	CreateSite(t, ctx, apiClient, s1req)

	s2req := utils.Site2Request
	s2req.RegionId = r2.JSON201.ResourceId
	CreateSite(t, ctx, apiClient, s2req)

	orderByResourceID := "resource_id asc"
	orderByRegion := "site_resource_region desc, resource_id"
	orderByWrong := "resource_?"

	type testCase struct {
		name            string
		filter          string
		orderby         *string
		amountResources int
		fail            bool
	}

	testCasesSites := []testCase{
		{
			name:            "test sites: no resource_id",
			filter:          fmt.Sprintf(`%s = ""`, "resource_id"),
			amountResources: 0,
			fail:            false,
		},
		{
			name:            "test sites: no region with parent_region",
			filter:          fmt.Sprintf(`has(%s.%s)`, "region", "parent_region"),
			amountResources: 0,
			fail:            false,
		},
		{
			name:            "test sites: no site with dns address",
			filter:          fmt.Sprintf(`%s = %q`, "dns_servers", "*.10.53"),
			amountResources: 0,
			fail:            false,
		},
		{
			name:            "test sites: sites with dns address",
			filter:          fmt.Sprintf(`%s = %q`, "dns_servers", "*.10.10"),
			orderby:         &orderByResourceID,
			amountResources: 2,
			fail:            false,
		},
		{
			name:            "test sites: sites with no region",
			filter:          fmt.Sprintf(`NOT has(%s)`, "region"),
			amountResources: 0,
			fail:            false,
		},
		{
			name:            "test sites: sites with non existing metadata",
			filter:          fmt.Sprintf(`%s = '%s'`, "metadata", `{"key":"cluster-name","value":""}`),
			amountResources: 0,
			fail:            false,
		},
		{
			name:            "test sites: sites with existing metadata - site2",
			filter:          fmt.Sprintf(`%s = '%s'`, "metadata", `{"key":"examplekey2","value":"site1"}`),
			orderby:         &orderByResourceID,
			amountResources: 1,
			fail:            false,
		},
		{
			name:            "test sites: sites with existing metadata - site2",
			filter:          fmt.Sprintf(`%s = '%s'`, "metadata", `{"key":"examplekey2","value":"site1"}`),
			orderby:         &orderByRegion,
			amountResources: 1,
			fail:            false,
		},
		{
			name:            "test sites: sites with bad metadata value",
			filter:          fmt.Sprintf(`%s = '%s'`, "metadata", `{"key":"??","value":"site1"}`),
			amountResources: 0,
			fail:            true,
		},
		{
			name:            "test sites: sites with bad orderby value",
			filter:          fmt.Sprintf(`%s = '%s'`, "metadata", `{"key":"examplekey2","value":"site1"}`),
			orderby:         &orderByWrong,
			amountResources: 0,
			fail:            true,
		},
	}

	for _, tc := range testCasesSites {
		t.Run(tc.name, func(t *testing.T) {
			sites, err := apiClient.GetSitesWithResponse(
				ctx,
				&api.GetSitesParams{
					Filter:  &tc.filter,
					OrderBy: tc.orderby,
				},
				AddJWTtoTheHeader,
				AddProjectIDtoTheHeader,
			)
			require.NoError(t, err)

			if !tc.fail {
				require.Equal(t, http.StatusOK, sites.StatusCode())
				assert.Equal(t, tc.amountResources, *sites.JSON200.TotalElements)
				assert.Equal(t, tc.amountResources, len(*sites.JSON200.Sites))
			} else {
				require.NotEqual(t, http.StatusOK, sites.StatusCode())
			}
		})
	}
}
