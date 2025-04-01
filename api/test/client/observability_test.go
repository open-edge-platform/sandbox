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

func TestObservabilityClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	r1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)
	assert.Equal(t, utils.Region1Name, *r1.JSON201.Name)

	utils.Site1Request.RegionId = r1.JSON201.RegionID
	s1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.Site2Request.RegionId = r1.JSON201.RegionID
	_ = CreateSite(t, ctx, apiClient, utils.Site2Request)

	utils.Host1Request.SiteId = s1.JSON201.SiteID
	utils.Host2Request.SiteId = s1.JSON201.SiteID

	_ = CreateHost(t, ctx, apiClient, utils.Host1Request)
	CreateHost(t, ctx, apiClient, utils.Host2Request)

	nodeGuIDstr := utils.Host1UUID1.String()

	obsClient, err := GetAPIClient()
	require.NoError(t, err)

	resList, err := obsClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Uuid: &nodeGuIDstr,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.LessOrEqual(t, 1, len(*resList.JSON200.Hosts))

	schedList, err := obsClient.GetSchedulesWithResponse(
		ctx,
		&api.GetSchedulesParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, schedList.StatusCode())
	assert.NotNil(t, schedList.JSON200.SingleSchedules)
	assert.NotNil(t, schedList.JSON200.RepeatedSchedules)
}
