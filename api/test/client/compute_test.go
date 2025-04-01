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

func TestCompute(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	r1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	assert.Equal(t, utils.Region1Name, *r1.JSON201.Name)

	utils.Site1Request.RegionId = r1.JSON201.RegionID
	s1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	host1Request := GetHostRequestWithRandomUUID()
	host1Request.SiteId = s1.JSON201.SiteID
	host2Request := GetHostRequestWithRandomUUID()
	host2Request.SiteId = s1.JSON201.SiteID

	CreateHost(t, ctx, apiClient, host1Request)
	CreateHost(t, ctx, apiClient, host2Request)

	res, err := apiClient.GetComputeWithResponse(ctx, &api.GetComputeParams{}, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.LessOrEqual(t, 2, len(*res.JSON200.Hosts))

	// Cleanup done in create helper functions
}

func TestECM(t *testing.T) {
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

	host1Request := GetHostRequestWithRandomUUID()
	host1Request.SiteId = s1.JSON201.SiteID
	host2Request := GetHostRequestWithRandomUUID()
	host2Request.SiteId = s1.JSON201.SiteID

	h1 := CreateHost(t, ctx, apiClient, host1Request)
	CreateHost(t, ctx, apiClient, host2Request)

	nodeGuIDstr := host1Request.Uuid.String()

	ecmClient, err := GetAPIClient()
	require.NoError(t, err)

	resList, err := ecmClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Uuid: &nodeGuIDstr,
		},
		addEcmUserAgentToTheHeader,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.LessOrEqual(t, 1, len(*resList.JSON200.Hosts))

	// Uses Patch to update host site with s2 siteID
	host1PatchReq := api.Host{
		SiteId: s2.JSON201.SiteID,
	}
	h1Patch, err := ecmClient.PatchComputeHostsHostIDWithResponse(
		ctx,
		*h1.JSON201.ResourceId,
		host1PatchReq,
		addEcmUserAgentToTheHeader,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, h1Patch.StatusCode())
}

func TestComputeSummary(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	utils.Site1Request.RegionId = nil
	s1 := CreateSite(t, ctx, apiClient, utils.Site1Request)
	utils.Site2Request.RegionId = nil
	s2 := CreateSite(t, ctx, apiClient, utils.Site2Request)

	expectedTotalHost := 0
	expectedUnallocatedHost := 0

	hostsWithSiteAndMetaFromSite2 := 31
	hostsWithoutSiteWithMeta := 15
	hostsWithSiteFromSite1 := 20

	// Hosts without site
	for i := 1; i < 15; i++ {
		expectedTotalHost++
		expectedUnallocatedHost++
		hostRequest := GetHostRequestWithRandomUUID()
		CreateHost(t, ctx, apiClient, hostRequest)
	}

	// Hosts with Meta
	for i := 0; i < hostsWithoutSiteWithMeta; i++ {
		expectedTotalHost++
		expectedUnallocatedHost++
		hostRequest := GetHostRequestWithRandomUUID()
		hostRequest.Metadata = &utils.MetadataHost1
		CreateHost(t, ctx, apiClient, hostRequest)
	}

	// Hosts with site
	for i := 0; i < hostsWithSiteFromSite1; i++ {
		expectedTotalHost++
		hostRequest := GetHostRequestWithRandomUUID()
		hostRequest.SiteId = s1.JSON201.SiteID
		CreateHost(t, ctx, apiClient, hostRequest)
	}

	// Hosts with site and meta from site
	for i := 0; i < hostsWithSiteAndMetaFromSite2; i++ {
		expectedTotalHost++
		hostRequest := GetHostRequestWithRandomUUID()
		hostRequest.SiteId = s2.JSON201.SiteID
		hostRequest.Metadata = &utils.MetadataHost2
		CreateHost(t, ctx, apiClient, hostRequest)
	}

	// Total (all hosts)
	res, err := apiClient.GetComputeHostsSummaryWithResponse(ctx, &api.GetComputeHostsSummaryParams{}, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.Equal(t, expectedTotalHost, *res.JSON200.Total)
	assert.Equal(t, expectedUnallocatedHost, *res.JSON200.Unallocated)

	// Filter by metadata (inherited) `metadata='{"key":"examplekey3","value":"host2"}'`
	filter := fmt.Sprintf("metadata='{\"key\":\"%s\",\"value\":\"%s\"}'",
		utils.MetadataHost2[0].Key, utils.MetadataHost2[0].Value)
	assert.Equal(t, `metadata='{"key":"examplekey1","value":"host2"}'`, filter)
	res, err = apiClient.GetComputeHostsSummaryWithResponse(ctx, &api.GetComputeHostsSummaryParams{Filter: &filter}, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.Equal(t, hostsWithSiteAndMetaFromSite2, *res.JSON200.Total)
	assert.Zero(t, *res.JSON200.Unallocated)
	assert.Zero(t, *res.JSON200.Error)
	assert.Zero(t, *res.JSON200.Running)

	// Filter by metadata (standalone) `metadata='{"key":"examplekey3","value":"host2"}'`
	filter = fmt.Sprintf("metadata='{\"key\":\"%s\",\"value\":\"%s\"}'",
		utils.MetadataHost2[0].Key, utils.MetadataHost1[0].Value)
	assert.Equal(t, `metadata='{"key":"examplekey1","value":"host1"}'`, filter)
	res, err = apiClient.GetComputeHostsSummaryWithResponse(ctx, &api.GetComputeHostsSummaryParams{Filter: &filter}, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.Equal(t, hostsWithoutSiteWithMeta, *res.JSON200.Total)
	assert.Equal(t, hostsWithoutSiteWithMeta, *res.JSON200.Unallocated)
	assert.Zero(t, *res.JSON200.Error)
	assert.Zero(t, *res.JSON200.Running)

	// Filter by host's site-id
	filter = fmt.Sprintf("site.resourceId=\"%s\"", *s1.JSON201.SiteID)
	res, err = apiClient.GetComputeHostsSummaryWithResponse(ctx, &api.GetComputeHostsSummaryParams{Filter: &filter}, AddJWTtoTheHeader, AddProjectIDtoTheHeader)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode())
	assert.Equal(t, hostsWithSiteFromSite1, *res.JSON200.Total)
	assert.Zero(t, *res.JSON200.Unallocated)
	assert.Zero(t, *res.JSON200.Error)
	assert.Zero(t, *res.JSON200.Running)
	// Cleanup done in create helper functions
}
