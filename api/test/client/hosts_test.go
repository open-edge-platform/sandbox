// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	"github.com/open-edge-platform/infra-core/api/test/utils"
)

func TestHostCustom(t *testing.T) {
	log.Info().Msgf("Begin compute host tests")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	r1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	assert.Equal(t, utils.Region1Name, *r1.JSON201.Name)

	utils.Site1Request.RegionId = r1.JSON201.RegionID
	s1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.Site2Request.RegionId = r1.JSON201.RegionID
	s2 := CreateSite(t, ctx, apiClient, utils.Site2Request)

	utils.Host1Request.SiteId = s1.JSON201.SiteID
	utils.Host2Request.SiteId = s1.JSON201.SiteID

	h1 := CreateHost(t, ctx, apiClient, utils.Host1Request)
	CreateHost(t, ctx, apiClient, utils.Host2Request)

	resHostH1, err := apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*h1.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resHostH1.StatusCode())
	assert.Equal(t, utils.Host1Name, resHostH1.JSON200.Name)
	assert.Equal(t, api.HOSTSTATEONBOARDED, *resHostH1.JSON200.DesiredState)

	// Change site of Host1 via PUT
	utils.Host1RequestUpdate.SiteId = s2.JSON201.SiteID
	h1Up, err := apiClient.PutComputeHostsHostIDWithResponse(
		ctx,
		*h1.JSON201.ResourceId,
		utils.Host1RequestUpdate,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, h1Up.StatusCode())

	// now check the filter, 1 host in site1, 1 host in site2
	site := s1.JSON201.SiteID
	res, err := apiClient.GetComputeWithResponse(
		ctx,
		&api.GetComputeParams{SiteID: site},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.Equal(t, 1, len(*res.JSON200.Hosts))

	site = s2.JSON201.SiteID
	res, err = apiClient.GetComputeWithResponse(
		ctx,
		&api.GetComputeParams{SiteID: site},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.Equal(t, 1, len(*res.JSON200.Hosts))

	resHostH1Up, err := apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*h1.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, utils.Host2Name, resHostH1Up.JSON200.Name)

	// Uses Puts to update Host site with empty string
	utils.Host1RequestUpdate.SiteId = &emptyString
	h1Up, err = apiClient.PutComputeHostsHostIDWithResponse(
		ctx,
		*h1.JSON201.ResourceId,
		utils.Host1RequestUpdate,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, h1Up.StatusCode())
	assert.Equal(t, api.HOSTSTATEONBOARDED, *h1Up.JSON200.DesiredState)

	// now check the filter
	site = s2.JSON201.SiteID
	res, err = apiClient.GetComputeWithResponse(
		ctx,
		&api.GetComputeParams{SiteID: site},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.Equal(t, 0, len(*res.JSON200.Hosts))

	resHostH1Up, err = apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*h1.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, utils.Host2Name, resHostH1Up.JSON200.Name)
	assert.Nil(t, resHostH1Up.JSON200.Site)

	// Uses Patch to update host1 site with s2 siteID
	utils.Host1RequestPatch.SiteId = s2.JSON201.SiteID
	h1Patch, err := apiClient.PatchComputeHostsHostIDWithResponse(
		ctx,
		*h1.JSON201.ResourceId,
		utils.Host1RequestPatch,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, h1Patch.StatusCode())

	resHostH1Patched, err := apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*h1.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, utils.Host3Name, resHostH1Patched.JSON200.Name)
	assert.Equal(t, *s2.JSON201.SiteID, *resHostH1Patched.JSON200.Site.ResourceId)

	// Uses Patch to update host1 site with s2 siteID
	utils.Host1RequestPatch.SiteId = &emptyString
	h1Patch, err = apiClient.PatchComputeHostsHostIDWithResponse(
		ctx,
		*h1.JSON201.ResourceId,
		utils.Host1RequestPatch,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, h1Patch.StatusCode())

	resHostH1Patched, err = apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*h1.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, utils.Host3Name, resHostH1Patched.JSON200.Name)
	assert.Nil(t, resHostH1Patched.JSON200.Site)
	assert.Equal(t, api.HOSTSTATEONBOARDED, *resHostH1Patched.JSON200.DesiredState)

	// Expect BadRequest errors in Patch/Put with emptyString wrong
	utils.Host1RequestUpdate.SiteId = &emptyStringWrong
	h1Up, err = apiClient.PutComputeHostsHostIDWithResponse(
		ctx,
		*h1.JSON201.ResourceId,
		utils.Host1RequestUpdate,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, h1Up.StatusCode())

	utils.Host1RequestPatch.SiteId = &emptyStringWrong
	h1Patch, err = apiClient.PatchComputeHostsHostIDWithResponse(
		ctx,
		*h1.JSON201.ResourceId,
		utils.Host1RequestPatch,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, h1Patch.StatusCode())

	// cleanup
	utils.Host1RequestPatch.Site = nil
	utils.Host1RequestUpdate.Site = nil

	log.Info().Msgf("End compute host tests")
}

func TestHostSites(t *testing.T) {
	log.Info().Msgf("Begin compute host tests")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	r1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	assert.Equal(t, utils.Region1Name, *r1.JSON201.Name)

	utils.Site1Request.RegionId = r1.JSON201.RegionID
	s1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.Host1Request.SiteId = s1.JSON201.SiteID
	utils.Host2Request.SiteId = nil

	h1 := CreateHost(t, ctx, apiClient, utils.Host1Request)
	CreateHost(t, ctx, apiClient, utils.Host2Request)

	// now check the filter
	siteQuery := s1.JSON201.SiteID
	res, err := apiClient.GetComputeWithResponse(
		ctx,
		&api.GetComputeParams{SiteID: siteQuery},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.Equal(t, 1, len(*res.JSON200.Hosts))

	emptySite := "null"
	res, err = apiClient.GetComputeWithResponse(
		ctx,
		&api.GetComputeParams{
			SiteID: &emptySite,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.Equal(t, 1, len(*res.JSON200.Hosts))

	resHostH1, err := apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*h1.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resHostH1.StatusCode())
	assert.Equal(t, utils.Host1Name, resHostH1.JSON200.Name)

	log.Info().Msgf("End compute host tests")
}

func TestHostErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)
	if err != nil {
		t.Fatalf("new API client error %s", err.Error())
	}

	t.Run("Get_UnexistID_Status_StatusNotFound", func(t *testing.T) {
		resHost, err := apiClient.GetComputeHostsHostIDWithResponse(
			ctx,
			utils.HostUnexistID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resHost.StatusCode())
	})

	t.Run("Put_UnexistID_Status_NotFoundError", func(t *testing.T) {
		resHost, err := apiClient.PutComputeHostsHostIDWithResponse(
			ctx,
			utils.HostUnexistID,
			utils.Host1RequestPut,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resHost.StatusCode())
	})

	t.Run("Patch_UnexistID_Status_NotFoundError", func(t *testing.T) {
		resHost, err := apiClient.PatchComputeHostsHostIDWithResponse(
			ctx,
			utils.HostUnexistID,
			utils.Host1RequestPut,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resHost.StatusCode())
	})

	t.Run("Delete_UnexistID_Status_NotFoundError", func(t *testing.T) {
		resHost, err := apiClient.DeleteComputeHostsHostID(
			ctx,
			utils.HostUnexistID,
			api.DeleteComputeHostsHostIDJSONRequestBody{},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resHost.StatusCode)
	})

	t.Run("Get_WrongID_Status_StatusBadRequest", func(t *testing.T) {
		resHost, err := apiClient.GetComputeHostsHostIDWithResponse(
			ctx,
			utils.HostWrongID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resHost.StatusCode())
	})

	t.Run("Put_WrongID_Status_StatusBadRequest", func(t *testing.T) {
		resHost, err := apiClient.PutComputeHostsHostIDWithResponse(
			ctx,
			utils.HostWrongID,
			utils.Host1Request,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resHost.StatusCode())
	})

	t.Run("Patch_WrongID_Status_StatusBadRequest", func(t *testing.T) {
		resHost, err := apiClient.PatchComputeHostsHostIDWithResponse(
			ctx,
			utils.HostWrongID,
			utils.Host1Request,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resHost.StatusCode())
	})

	t.Run("Delete_WrongID_Status_StatusBadRequest", func(t *testing.T) {
		resHost, err := apiClient.DeleteComputeHostsHostID(
			ctx,
			utils.HostWrongID,
			api.DeleteComputeHostsHostIDJSONRequestBody{},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resHost.StatusCode)
	})

	t.Run("Post_NonPrintable_Status_StatusBadRequest", func(t *testing.T) {
		resHost, err := apiClient.PostComputeHosts(
			ctx,
			utils.HostNonPrintable,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resHost.StatusCode)
	})

	t.Run("Get_NonPrintable_Status_StatusBadRequest", func(t *testing.T) {
		resHost, err := apiClient.GetComputeHosts(
			ctx,
			&api.GetComputeHostsParams{
				Uuid: &utils.HostGUIDNonPrintable,
			},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resHost.StatusCode)
	})
}

func TestHostList(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	r1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	assert.Equal(t, utils.Region1Name, *r1.JSON201.Name)

	utils.Site1Request.RegionId = r1.JSON201.RegionID
	s1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.Host1Request.SiteId = s1.JSON201.SiteID

	totalItems := 10
	pageId := 1
	pageSize := 4

	for id := 0; id < totalItems; id++ {
		h := GetHostRequestWithRandomUUID()
		CreateHost(t, ctx, apiClient, h)
	}

	resList, err := apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Offset:   &pageId,
			PageSize: &pageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.Hosts), pageSize)
	assert.Equal(t, true, *resList.JSON200.HasNext)

	// Use default page size (20)
	resList, err = apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, false, *resList.JSON200.HasNext)
}

func BenchmarkHostList(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	b.Cleanup(func() { cancel() })

	apiClient, err := GetAPIClient()
	assert.NoError(b, err)

	r1 := CreateRegion(b, ctx, apiClient, utils.Region1Request)
	assert.Equal(b, utils.Region1Name, *r1.JSON201.Name)
	b.Cleanup(func() { DeleteRegion(b, ctx, apiClient, *r1.JSON201.RegionID) })

	utils.Site1Request.RegionId = r1.JSON201.RegionID
	s1 := CreateSite(b, ctx, apiClient, utils.Site1Request)
	b.Cleanup(func() { DeleteSite(b, ctx, apiClient, *s1.JSON201.SiteID) })

	utils.Host1Request.SiteId = s1.JSON201.SiteID

	// this is the shakeup run
	benchmarkHosts(b, 5, apiClient, ctx)

	// Loop for different number of hosts.
	for _, i := range []int{10, 50, 100, 250} {
		b.Run(fmt.Sprintf("Hosts%d", i), func(b *testing.B) {
			benchmarkHosts(b, i, apiClient, ctx)
		})
	}
}

func benchmarkHosts(b *testing.B, nHosts int,
	apiClient *api.ClientWithResponses, ctx context.Context,
) {
	b.Helper()

	// Emulate the request of the GUI
	pageId := 1
	pageSize := 100

	for id := 0; id < nHosts; id++ {
		postResp := CreateHost(b, ctx, apiClient, utils.Host1Request)
		b.Cleanup(func() { SoftDeleteHost(b, ctx, apiClient, *postResp.JSON201.ResourceId) })
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		resList, err := apiClient.GetComputeHostsWithResponse(
			ctx,
			&api.GetComputeHostsParams{
				Offset:   &pageId,
				PageSize: &pageSize,
			},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		assert.NoError(b, err)
		assert.Equal(b, http.StatusOK, resList.StatusCode())
	}
	b.StopTimer()
}

func TestHostListFilterMetadata(t *testing.T) {
	// initializing context
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	// creating host #1
	postResp := CreateHost(t, ctx, apiClient, utils.HostReqFilterMeta1)
	hID1 := *postResp.JSON201.ResourceId

	// creating host #2
	postResp = CreateHost(t, ctx, apiClient, utils.HostReqFilterMeta2)
	hID2 := *postResp.JSON201.ResourceId

	// obtaining host with Metadata Key=filtermetakey1 and Value=filtermetavalue1
	reqMetadata1 := []string{"filtermetakey1=filtermetavalue1"}
	resList, err := apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Metadata: &reqMetadata1,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.GreaterOrEqual(t, len(*resList.JSON200.Hosts), 2)
	assert.True(t, hostsContainsId(*resList.JSON200.Hosts, hID1))
	assert.True(t, hostsContainsId(*resList.JSON200.Hosts, hID2))

	// obtaining host with Metadata Key=filtermetakey2 and Value=filtermetavalue2
	reqMetadata2 := []string{"filtermetakey2=filtermetavalue2"}
	resList, err = apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Metadata: &reqMetadata2,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.GreaterOrEqual(t, len(*resList.JSON200.Hosts), 1)
	assert.True(t, hostsContainsId(*resList.JSON200.Hosts, hID1))
	assert.False(t, hostsContainsId(*resList.JSON200.Hosts, hID2))

	// obtaining host with Metadata Key=filtermetakey2 and Value=filtermetavalue2_mod
	reqMetadata3 := []string{"filtermetakey2=filtermetavalue2_mod"}
	resList, err = apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Metadata: &reqMetadata3,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.GreaterOrEqual(t, len(*resList.JSON200.Hosts), 1)
	assert.False(t, hostsContainsId(*resList.JSON200.Hosts, hID1))
	assert.True(t, hostsContainsId(*resList.JSON200.Hosts, hID2))

	// obtaining host with Metadata from Host1
	reqMetadataJoin := []string{"filtermetakey1=filtermetavalue1", "filtermetakey2=filtermetavalue2"}
	resList, err = apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Metadata: &reqMetadataJoin,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.GreaterOrEqual(t, len(*resList.JSON200.Hosts), 1)
	assert.True(t, hostsContainsId(*resList.JSON200.Hosts, hID1))

	// obtaining host with Metadata from Host2
	reqMetadataJoin = []string{"filtermetakey1=filtermetavalue1", "filtermetakey2=filtermetavalue2_mod"}
	resList, err = apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Metadata: &reqMetadataJoin,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.GreaterOrEqual(t, len(*resList.JSON200.Hosts), 1)
	assert.True(t, hostsContainsId(*resList.JSON200.Hosts, hID2))

	// Look for a host with wrong metadata
	reqMetadata4 := []string{"randomKey=randomValue"}
	resList, err = apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Metadata: &reqMetadata4,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Empty(t, resList.JSON200.Hosts)
}

func TestHostListFilterUUID(t *testing.T) {
	// initializing context
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	r1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	assert.Equal(t, utils.Region1Name, *r1.JSON201.Name)

	utils.Site1Request.RegionId = r1.JSON201.RegionID
	s1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.Site2Request.RegionId = r1.JSON201.RegionID
	s2 := CreateSite(t, ctx, apiClient, utils.Site2Request)

	utils.Host1Request.SiteId = s1.JSON201.SiteID
	utils.Host2Request.SiteId = s1.JSON201.SiteID

	// creating host #1
	CreateHost(t, ctx, apiClient, utils.Host1Request)

	// creating host #2
	CreateHost(t, ctx, apiClient, utils.Host2Request)

	metadata := &api.Metadata{
		{Key: "k", Value: "v"},
	}
	// creating host #3
	CreateHost(t, ctx, apiClient, api.Host{
		Name:     utils.Host3Name,
		SiteId:   s2.JSON201.SiteID,
		Metadata: metadata,
		Uuid:     &utils.Host3UUID,
	})

	// obtaining host with Device GUID#1
	guidFind1 := utils.Host1UUID1.String()
	resList, err := apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Uuid: &guidFind1,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, 1, len(*resList.JSON200.Hosts))
	assert.Equal(t, false, *resList.JSON200.HasNext)

	// obtaining host with Device GUID #2
	guidFind2 := utils.Host2UUID.String()
	resList, err = apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Uuid: &guidFind2,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.Hosts), 1)
	assert.Equal(t, false, *resList.JSON200.HasNext)

	largePageSize := 100
	// Look for all hosts
	resList, err = apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Uuid:     nil,
			PageSize: &largePageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.GreaterOrEqual(t, len(*resList.JSON200.Hosts), 3)
	assert.Equal(t, false, *resList.JSON200.HasNext)

	// Look for an unexistent host
	guidFindUnexists := utils.HostUUIDUnexists.String()
	resList, err = apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Uuid: &guidFindUnexists,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Empty(t, resList.JSON200.Hosts)

	// Look for a host with wrong UUID
	resList, err = apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Uuid: &utils.HostUUIDError,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resList.StatusCode())
}

func TestHostPower(t *testing.T) {
	log.Info().Msgf("Begin compute host power status tests")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	r1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	assert.Equal(t, utils.Region1Name, *r1.JSON201.Name)

	utils.Site1Request.RegionId = r1.JSON201.RegionID
	s1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.Host1Request.SiteId = s1.JSON201.SiteID
	h1 := CreateHost(t, ctx, apiClient, utils.Host1Request)

	// Get host status power state
	resHostH1, err := apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*h1.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resHostH1.StatusCode())
	assert.Equal(t, utils.Host1Name, resHostH1.JSON200.Name)
	// By default host current power state is Unspecified
	assert.Equal(
		t,
		api.POWERSTATEUNSPECIFIED,
		*resHostH1.JSON200.CurrentPowerState,
	)
	// By default host desired power state is ON
	assert.Equal(t, api.POWERSTATEON, *resHostH1.JSON200.DesiredPowerState)

	// Set host desired power on
	h1UpON, err := apiClient.PatchComputeHostsHostIDWithResponse(
		ctx,
		*h1.JSON201.ResourceId,
		utils.Host1RequestUpdatePowerON,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, h1UpON.StatusCode())

	resHostH1Up, err := apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*h1.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, api.POWERSTATEON, *resHostH1Up.JSON200.DesiredPowerState)

	// Set host desired power off
	h1UpOFF, err := apiClient.PatchComputeHostsHostIDWithResponse(
		ctx,
		*h1.JSON201.ResourceId,
		utils.Host1RequestUpdatePowerOFF,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, h1UpOFF.StatusCode())

	resHostH1Up, err = apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*h1.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, api.POWERSTATEOFF, *resHostH1Up.JSON200.DesiredPowerState)

	log.Info().Msgf("End compute host power status tests")
}

func TestHostInvalidate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	_, _, h4 := loadZTPTest(t, ctx, apiClient, &utils.Region1Request,
		&utils.Site1Request, &utils.Host4Request)
	require.NoError(t, err)

	_, err = apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*h4.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)

	note := "host is lost"
	_, err = apiClient.PutComputeHostsHostIDInvalidate(
		ctx,
		*h4.JSON201.ResourceId,
		api.PutComputeHostsHostIDInvalidateJSONRequestBody{
			Note: note,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)

	// TODO: wait for condition instead of sleep()
	time.Sleep(3 * time.Second)

	res, err := apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*h4.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, api.HOSTSTATEUNTRUSTED, *res.JSON200.CurrentState)
	assert.Equal(t, "Invalidated", *res.JSON200.HostStatus)
	assert.Equal(t, note, *res.JSON200.Note)
}

func loadZTPTest(t *testing.T, ctx context.Context, apiClient *api.ClientWithResponses,
	regionRequest *api.Region, siteRequest *api.Site, hostRequest *api.Host) (
	*api.PostRegionsResponse, *api.PostSitesResponse, *api.PostComputeHostsResponse,
) {
	reg := CreateRegion(t, ctx, apiClient, *regionRequest)
	assert.Equal(t, regionRequest.Name, reg.JSON201.Name)

	siteRequest.RegionId = reg.JSON201.RegionID
	sit := CreateSite(t, ctx, apiClient, *siteRequest)
	assert.Equal(t, siteRequest.Name, sit.JSON201.Name)

	// No site defined
	hos := CreateHost(t, ctx, apiClient, *hostRequest)

	resH, err := apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*hos.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resH.StatusCode())
	assert.Equal(t, hostRequest.Name, resH.JSON200.Name)
	assert.Equal(t, *hostRequest.Uuid, *resH.JSON200.Uuid)

	return reg, sit, hos
}

// Test main workflow for ZTP using PUT.
func TestHostZTPWithPut(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	_, s1, h4 := loadZTPTest(t, ctx, apiClient, &utils.Region1Request,
		&utils.Site1Request, &utils.Host4Request)
	require.NoError(t, err)

	// Simulate ZTP with PUT
	utils.Host4RequestPut.SiteId = s1.JSON201.SiteID
	h4Put, err := apiClient.PutComputeHostsHostIDWithResponse(
		ctx,
		*h4.JSON201.ResourceId,
		utils.Host4RequestPut,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, h4Put.StatusCode())

	// now check the filter
	UUID := utils.Host4UUID1.String()
	res, err := apiClient.GetComputeWithResponse(
		ctx,
		&api.GetComputeParams{Uuid: &UUID},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.Equal(t, 1, len(*res.JSON200.Hosts))

	resHostH4Put, err := apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*h4.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, utils.Host4UUID1, *resHostH4Put.JSON200.Uuid)
}

// Test main workflow for ZTP using PATCH.
func TestHostZTPWithPatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	_, s1, h4 := loadZTPTest(t, ctx, apiClient, &utils.Region1Request,
		&utils.Site1Request, &utils.Host4Request)
	require.NoError(t, err)

	// Simulate ZTP with PATCH
	utils.Host4RequestPatch.SiteId = s1.JSON201.SiteID
	h4Patch, err := apiClient.PatchComputeHostsHostIDWithResponse(
		ctx,
		*h4.JSON201.ResourceId,
		utils.Host4RequestPatch,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, h4Patch.StatusCode())

	// now check the filter
	UUID := utils.Host4UUID1.String()
	res, err := apiClient.GetComputeWithResponse(
		ctx,
		&api.GetComputeParams{Uuid: &UUID},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.LessOrEqual(t, 1, len(*res.JSON200.Hosts))

	resHostH4Patch, err := apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*h4.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, utils.Host4UUID1, *resHostH4Patch.JSON200.Uuid)
}

func TestHostsSummary(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	r1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	assert.Equal(t, utils.Region1Name, *r1.JSON201.Name)

	utils.Site1Request.RegionId = r1.JSON201.RegionID
	s1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.Site2Request.RegionId = r1.JSON201.RegionID
	CreateSite(t, ctx, apiClient, utils.Site2Request)

	utils.Host1Request.SiteId = s1.JSON201.SiteID
	utils.Host2Request.SiteId = nil

	h1 := CreateHost(t, ctx, apiClient, utils.Host1Request)
	CreateHost(t, ctx, apiClient, utils.Host2Request)

	resHostH1, err := apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*h1.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resHostH1.StatusCode())
	assert.Equal(t, utils.Host1Name, resHostH1.JSON200.Name)

	resHostSummary, err := apiClient.GetComputeHostsSummaryWithResponse(
		ctx,
		&api.GetComputeHostsSummaryParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resHostSummary.StatusCode())
	assert.GreaterOrEqual(t, *resHostSummary.JSON200.Total, 2)
	if resHostSummary.JSON200.Error != nil {
		assert.GreaterOrEqual(t, *resHostSummary.JSON200.Error, 0)
	}
	if resHostSummary.JSON200.Running != nil {
		assert.GreaterOrEqual(t, *resHostSummary.JSON200.Running, 0)
	}
	assert.GreaterOrEqual(t, *resHostSummary.JSON200.Unallocated, 1)
}

func TestHostRegister(t *testing.T) {
	log.Info().Msgf("Begin compute host register tests")
	var registeredHosts []*api.PostComputeHostsRegisterResponse

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	// register host using UUID & SN
	registeredHost1, err := apiClient.PostComputeHostsRegisterWithResponse(
		ctx,
		utils.HostRegister,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	registeredHosts = append(registeredHosts, registeredHost1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, registeredHost1.StatusCode())
	assert.Equal(t, *utils.HostRegister.Uuid, *registeredHost1.JSON201.Uuid)
	assert.Equal(t, api.HOSTSTATEREGISTERED, *registeredHost1.JSON201.DesiredState)

	// change registered host name - via Patch
	resHostRegisterPatch, err := apiClient.PatchComputeHostsHostIDRegisterWithResponse(
		ctx,
		*registeredHost1.JSON201.ResourceId,
		utils.HostRegisterPatch,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resHostRegisterPatch.StatusCode())

	// get the patched host and verify name is updated
	resHostGet, err := apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*registeredHost1.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, *utils.HostRegisterPatch.Name, resHostGet.JSON200.Name)

	// change name & autoOnboard=true for registered host - via Patch
	resHostRegisterPatch, err = apiClient.PatchComputeHostsHostIDRegisterWithResponse(
		ctx,
		*registeredHost1.JSON201.ResourceId,
		api.PatchComputeHostsHostIDRegisterJSONRequestBody{
			Name:        &utils.Host2bName,
			AutoOnboard: &utils.AutoOnboardTrue,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resHostRegisterPatch.StatusCode())

	// get the patched host and verify desiredState is updated
	resHostGet, err = apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*registeredHost1.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resHostGet.StatusCode())
	assert.Equal(t, utils.Host2bName, resHostGet.JSON200.Name)
	assert.Equal(t, api.HOSTSTATEONBOARDED, *resHostGet.JSON200.DesiredState)

	// change autoOnboard=false only for registered host - via Patch
	resHostRegisterPatch, err = apiClient.PatchComputeHostsHostIDRegisterWithResponse(
		ctx,
		*registeredHost1.JSON201.ResourceId,
		api.PatchComputeHostsHostIDRegisterJSONRequestBody{
			AutoOnboard: &utils.AutoOnboardFalse,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resHostRegisterPatch.StatusCode())

	// get the patched host and verify desiredState is updated
	resHostGet, err = apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*registeredHost1.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resHostGet.StatusCode())
	assert.Equal(t, api.HOSTSTATEREGISTERED, *resHostGet.JSON200.DesiredState)

	// register host with autoOnboard=true
	registeredHost2, err := apiClient.PostComputeHostsRegisterWithResponse(
		ctx,
		utils.HostRegisterAutoOnboard,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	registeredHosts = append(registeredHosts, registeredHost2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, registeredHost2.StatusCode())
	assert.Equal(t, api.HOSTSTATEONBOARDED, *registeredHost2.JSON201.DesiredState)

	// change name & autoOnboard=false for registered host - via Patch
	resHostRegisterPatch, err = apiClient.PatchComputeHostsHostIDRegisterWithResponse(
		ctx,
		*registeredHost2.JSON201.ResourceId,
		api.PatchComputeHostsHostIDRegisterJSONRequestBody{
			Name:        &utils.Host1Name,
			AutoOnboard: &utils.AutoOnboardFalse,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resHostRegisterPatch.StatusCode())

	// get the patched host and verify desiredState is updated
	resHostGet, err = apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*registeredHost2.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resHostGet.StatusCode())
	assert.Equal(t, utils.Host1Name, resHostGet.JSON200.Name)
	assert.Equal(t, api.HOSTSTATEREGISTERED, *resHostGet.JSON200.DesiredState)

	// register host using UUID only
	registeredHost3, err := apiClient.PostComputeHostsRegisterWithResponse(
		ctx,
		api.HostRegisterInfo{
			Uuid: &utils.Host5UUID,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	registeredHosts = append(registeredHosts, registeredHost3)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, registeredHost3.StatusCode())
	assert.Equal(t, utils.Host5UUID, *registeredHost3.JSON201.Uuid)
	assert.Equal(t, api.HOSTSTATEREGISTERED, *registeredHost3.JSON201.DesiredState)

	// register host using SN only
	registeredHost4, err := apiClient.PostComputeHostsRegisterWithResponse(
		ctx,
		api.HostRegisterInfo{
			SerialNumber: &utils.HostSerialNumber3,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	registeredHosts = append(registeredHosts, registeredHost4)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, registeredHost4.StatusCode())
	assert.Equal(t, utils.HostSerialNumber3, *registeredHost4.JSON201.SerialNumber)
	assert.Equal(t, api.HOSTSTATEREGISTERED, *registeredHost4.JSON201.DesiredState)

	// invalid register command - no UUID, no SN
	resHostRegisterInv, err := apiClient.PostComputeHostsRegisterWithResponse(
		ctx,
		api.HostRegisterInfo{Name: &utils.Host4Name},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, resHostRegisterInv.StatusCode())

	// delete the registered hosts
	for _, host := range registeredHosts {
		resHost, err := apiClient.DeleteComputeHostsHostID(
			ctx,
			*host.JSON201.ResourceId,
			api.DeleteComputeHostsHostIDJSONRequestBody{},
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resHost.StatusCode)
	}

	log.Info().Msgf("End compute host register tests")
}

func TestHostOnboard(t *testing.T) {
	log.Info().Msgf("Begin compute host onboard tests")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	// register host using UUID & SN
	registeredHost, err := apiClient.PostComputeHostsRegisterWithResponse(
		ctx,
		utils.HostRegister,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, registeredHost.StatusCode())
	assert.Equal(t, *utils.HostRegister.Uuid, *registeredHost.JSON201.Uuid)
	assert.Equal(t, api.HOSTSTATEREGISTERED, *registeredHost.JSON201.DesiredState)

	// onboard host
	resOnboard, err := apiClient.PatchComputeHostsHostIDOnboardWithResponse(
		ctx,
		*registeredHost.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resOnboard.StatusCode())

	// get the onboarded host and verify the desiredState is updated
	onboardedHost, err := apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		*registeredHost.JSON201.ResourceId,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, onboardedHost.StatusCode())
	assert.Equal(t, api.HOSTSTATEONBOARDED, *onboardedHost.JSON200.DesiredState)

	// delete the onboarded host
	resHost, err := apiClient.DeleteComputeHostsHostID(
		ctx,
		*registeredHost.JSON201.ResourceId,
		api.DeleteComputeHostsHostIDJSONRequestBody{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resHost.StatusCode)

	log.Info().Msgf("End compute host onboard tests")
}
