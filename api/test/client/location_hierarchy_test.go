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

var (
	showRegions            = true
	showSites              = true
	emptyParentID          = ""
	commonSiteRegionSuffix = "-12345678"
	commonManySuffix       = "23456"

	regionPrefixName    = "state"
	subRegionPrefixName = "city"
	sitePrefixName      = "building"
	maxRegions          = 10
	maxSubRegions       = 10
	maxSites            = 10

	regionKind = api.RESOURCEKINDREGION
	siteKind   = api.RESOURCEKINDSITE
)

type testCase struct {
	name           string
	params         *api.GetLocationsParams
	expected       []api.LocationNode
	listedElements int // the expected length of the Nodes array inside the response
	totalElements  int // the expected response value of TotalElements
	outputElements int // the expected response value of outputElements
}

func setupRegionSiteHierarchy(
	t *testing.T,
	ctx context.Context,
	apiClient *api.ClientWithResponses,
) (*api.Region, *api.Region, *api.Region, *api.Site, *api.Site, *api.Site) {
	r1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)

	utils.Region2Request.ParentId = r1.JSON201.RegionID
	r2 := CreateRegion(t, ctx, apiClient, utils.Region2Request)
	utils.Region2Request.ParentId = nil

	utils.Region3Request.ParentId = r2.JSON201.RegionID
	r3 := CreateRegion(t, ctx, apiClient, utils.Region3Request)
	utils.Region3Request.ParentId = nil

	utils.Site1Request.RegionId = r1.JSON201.RegionID
	s1 := CreateSite(t, ctx, apiClient, utils.Site1Request)
	utils.Site1Request.RegionId = nil

	utils.Site2Request.RegionId = r2.JSON201.RegionID
	s2 := CreateSite(t, ctx, apiClient, utils.Site2Request)
	utils.Site2Request.Region = nil

	utils.Site2Request.RegionId = r2.JSON201.RegionID
	s3 := CreateSite(t, ctx, apiClient, utils.Site3Request)
	utils.Site2Request.Region = nil

	return r1.JSON201, r2.JSON201, r3.JSON201, s1.JSON201, s2.JSON201, s3.JSON201
}

func setupRegionSiteLargeHierarchy(
	t *testing.T,
	ctx context.Context,
	apiClient *api.ClientWithResponses,
) {
	for r := 0; r < maxRegions; r++ {
		regName := fmt.Sprintf("%s-%d", regionPrefixName, r)
		utils.Region1Request.Name = &regName
		utils.Region1Request.ParentId = nil
		r1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
		utils.Region1Request.Name = &utils.Region1Name

		for sr := 0; sr < maxSubRegions; sr++ {
			subregName := fmt.Sprintf("%s-%d-%d", subRegionPrefixName, r, sr)
			utils.Region2Request.Name = &subregName
			utils.Region2Request.ParentId = r1.JSON201.ResourceId
			r2 := CreateRegion(t, ctx, apiClient, utils.Region2Request)
			utils.Region2Request.ParentId = nil
			utils.Region2Request.Name = &utils.Region2Name

			for si := 0; si < maxSites; si++ {
				siteName := fmt.Sprintf("%s-%s-%d", subRegionPrefixName, sitePrefixName, si)
				utils.Site2Request.Name = &siteName
				utils.Site2Request.RegionId = r2.JSON201.RegionID
				CreateSite(t, ctx, apiClient, utils.Site2Request)
				utils.Site2Request.Region = nil
				utils.Site2Request.Name = &utils.Site2Name
			}
		}
	}
}

func TestLocation_Hierarchy(t *testing.T) {
	log.Info().Msgf("Begin TestLocation_Hierarchy")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	r1, r2, r3, s1, s2, s3 := setupRegionSiteHierarchy(t, ctx, apiClient)

	testCases := []testCase{
		{
			name: "Test root regions",
			params: &api.GetLocationsParams{
				Name:        &utils.Region1Name,
				ShowRegions: &showRegions,
			},
			expected: []api.LocationNode{
				{
					ResourceId: r1.ResourceId,
					Name:       r1.Name,
					ParentId:   &emptyParentID,
					Type:       &regionKind,
				},
			},
			totalElements:  1,
			outputElements: 1,
		},
		{
			name: "Test mid regions tree: looks for r2 -> gets [r1,r2]",
			params: &api.GetLocationsParams{
				Name:        &utils.Region2Name,
				ShowRegions: &showRegions,
			},
			expected: []api.LocationNode{
				{
					ResourceId: r2.ResourceId,
					Name:       r2.Name,
					ParentId:   r1.ResourceId,
					Type:       &regionKind,
				},
				{
					ResourceId: r1.ResourceId,
					Name:       r1.Name,
					ParentId:   &emptyParentID,
					Type:       &regionKind,
				},
			},
			totalElements:  1,
			outputElements: 1,
		},
		{
			name: "Test mid regions tree: looks for r3 -> gets [r1,r2,r3]",
			params: &api.GetLocationsParams{
				Name:        &utils.Region3Name,
				ShowRegions: &showRegions,
			},
			expected: []api.LocationNode{
				{
					ResourceId: r3.ResourceId,
					Name:       r3.Name,
					ParentId:   r2.ResourceId,
					Type:       &regionKind,
				},
				{
					ResourceId: r2.ResourceId,
					Name:       r2.Name,
					ParentId:   r1.ResourceId,
					Type:       &regionKind,
				},
				{
					ResourceId: r1.ResourceId,
					Name:       r1.Name,
					ParentId:   &emptyParentID,
					Type:       &regionKind,
				},
			},
			totalElements:  1,
			outputElements: 1,
		},
		{
			name: "Test mid sites tree: looks for s1 -> gets [r1,s1]",
			params: &api.GetLocationsParams{
				Name:      &utils.Site1Name,
				ShowSites: &showSites,
			},
			expected: []api.LocationNode{
				{
					ResourceId: s1.ResourceId,
					Name:       s1.Name,
					ParentId:   r1.ResourceId,
					Type:       &siteKind,
				},
				{
					ResourceId: r1.ResourceId,
					Name:       r1.Name,
					ParentId:   &emptyParentID,
					Type:       &regionKind,
				},
			},
			totalElements:  1,
			outputElements: 1,
		},
		{
			name: "Test mid sites tree: looks for s2 -> gets [r1,r2,s2]",
			params: &api.GetLocationsParams{
				Name:      &utils.Site2Name,
				ShowSites: &showSites,
			},
			expected: []api.LocationNode{
				{
					ResourceId: s2.ResourceId,
					Name:       s2.Name,
					ParentId:   r2.ResourceId,
					Type:       &siteKind,
				},
				{
					ResourceId: r2.ResourceId,
					Name:       r2.Name,
					ParentId:   r1.ResourceId,
					Type:       &regionKind,
				},
				{
					ResourceId: r1.ResourceId,
					Name:       r1.Name,
					ParentId:   &emptyParentID,
					Type:       &regionKind,
				},
			},
			totalElements:  1,
			outputElements: 1,
		},
		{
			name: "Test site and region tree: looks for common name -> gets [r1,s1] and totalElements 2",
			params: &api.GetLocationsParams{
				Name:        &commonSiteRegionSuffix,
				ShowSites:   &showSites,
				ShowRegions: &showRegions,
			},
			expected: []api.LocationNode{
				{
					ResourceId: s1.ResourceId,
					Name:       s1.Name,
					ParentId:   r1.ResourceId,
					Type:       &siteKind,
				},
				{
					ResourceId: r1.ResourceId,
					Name:       r1.Name,
					ParentId:   &emptyParentID,
					Type:       &regionKind,
				},
			},
			totalElements:  2,
			outputElements: 2,
		},
		{
			name: "Test site and region tree: looks for common name -> gets [r1,r2,s1,s2,s3] and totalElements 5",
			params: &api.GetLocationsParams{
				Name:        &commonManySuffix,
				ShowSites:   &showSites,
				ShowRegions: &showRegions,
			},
			expected: []api.LocationNode{
				{
					ResourceId: s1.ResourceId,
					Name:       s1.Name,
					ParentId:   r1.ResourceId,
					Type:       &siteKind,
				},
				{
					ResourceId: s2.ResourceId,
					Name:       s2.Name,
					ParentId:   r2.ResourceId,
					Type:       &siteKind,
				},
				{
					ResourceId: s3.ResourceId,
					Name:       s3.Name,
					ParentId:   &emptyParentID,
					Type:       &siteKind,
				},
				{
					ResourceId: r2.ResourceId,
					Name:       r2.Name,
					ParentId:   r1.ResourceId,
					Type:       &regionKind,
				},
				{
					ResourceId: r1.ResourceId,
					Name:       r1.Name,
					ParentId:   &emptyParentID,
					Type:       &regionKind,
				},
			},
			totalElements:  5,
			outputElements: 5,
		},
		{
			name: "Test leaf sites",
			params: &api.GetLocationsParams{
				Name:      &utils.Site3Name,
				ShowSites: &showSites,
			},
			expected: []api.LocationNode{
				{
					ResourceId: s3.ResourceId,
					Name:       s3.Name,
					ParentId:   &emptyParentID,
					Type:       &siteKind,
				},
			},
			totalElements:  1,
			outputElements: 1,
		},
		{
			name: "Test empty/unknown site",
			params: &api.GetLocationsParams{
				Name:      &utils.SiteUnexistID,
				ShowSites: &showSites,
			},
			expected:       []api.LocationNode{},
			totalElements:  0,
			outputElements: 0,
		},
		{
			name: "Test empty/unknown region",
			params: &api.GetLocationsParams{
				Name:        &utils.RegionUnexistID,
				ShowRegions: &showRegions,
			},
			expected:       []api.LocationNode{},
			totalElements:  0,
			outputElements: 0,
		},
	}

	for _, tcase := range testCases {
		t.Run(tcase.name, func(t *testing.T) {
			getlocResponse, err := apiClient.GetLocationsWithResponse(ctx, tcase.params, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
			require.NoError(t, err)
			respStatusCode := getlocResponse.StatusCode()
			require.Equal(t, http.StatusOK, respStatusCode)
			assert.EqualValues(t, tcase.expected, *getlocResponse.JSON200.Nodes)
			assert.Equal(t, tcase.totalElements, *getlocResponse.JSON200.TotalElements)
			assert.Equal(t, tcase.outputElements, *getlocResponse.JSON200.OutputElements)
		})
	}
}

func TestLocation_LargeHierarchy(t *testing.T) {
	log.Info().Msgf("Begin TestLocation_Hierarchy")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout*4)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	setupRegionSiteLargeHierarchy(t, ctx, apiClient)
	testCases := []testCase{
		{
			name: "Test root regions",
			params: &api.GetLocationsParams{
				Name:        &regionPrefixName,
				ShowRegions: &showRegions,
			},

			totalElements:  10,
			outputElements: 10,
			listedElements: 10,
		},
		{
			name: "Test sub regions",
			params: &api.GetLocationsParams{
				Name:        &subRegionPrefixName,
				ShowRegions: &showRegions,
			},

			totalElements:  100,
			outputElements: 50,
			listedElements: 55,
		},
		{
			name: "Test sites",
			params: &api.GetLocationsParams{
				Name:      &sitePrefixName,
				ShowSites: &showSites,
			},

			totalElements:  1000,
			outputElements: 50,
			listedElements: 105, // It returns 5 root regions, 50 sub regions (10 per root), and 1 site in each returned subregion
		},
		{
			name: "Test subregions and sites - contain the same prefix",
			params: &api.GetLocationsParams{
				Name:        &subRegionPrefixName,
				ShowSites:   &showSites,
				ShowRegions: &showRegions,
			},

			totalElements:  1100,
			outputElements: 100,
			listedElements: 105, // It returns 5 root regions, 50 sub regions (10 per root), and 1 site in each returned subregion
		},
	}
	for _, tcase := range testCases {
		t.Run(tcase.name, func(t *testing.T) {
			getlocResponse, err := apiClient.GetLocationsWithResponse(ctx, tcase.params, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
			require.NoError(t, err)
			respStatusCode := getlocResponse.StatusCode()
			require.Equal(t, http.StatusOK, respStatusCode)
			assert.Equal(t, tcase.totalElements, *getlocResponse.JSON200.TotalElements)
			assert.Equal(t, tcase.outputElements, *getlocResponse.JSON200.OutputElements)
			assert.Equal(t, tcase.listedElements, len(*getlocResponse.JSON200.Nodes))
		})
	}
}
