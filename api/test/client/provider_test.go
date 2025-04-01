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

const (
	numdefaultProviders = 1
)

func TestProvider_CreateGetDelete(t *testing.T) {
	log.Info().Msgf("Begin CreateGetDelete Provider tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	provider1 := CreateProvider(t, ctx, apiClient, utils.Provider1Request)
	provider2 := CreateProvider(t, ctx, apiClient, utils.Provider2Request)
	provider3 := CreateProvider(t, ctx, apiClient, utils.Provider3Request)

	get1, err := apiClient.GetProvidersProviderIDWithResponse(
		ctx,
		*provider1.JSON201.ProviderID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, get1.StatusCode())
	assert.Equal(t, utils.ProviderName1, get1.JSON200.Name)
	assert.Equal(t, *utils.Provider1Request.Config, *get1.JSON200.Config)

	get2, err := apiClient.GetProvidersProviderIDWithResponse(
		ctx,
		*provider2.JSON201.ProviderID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, get2.StatusCode())
	assert.Equal(t, utils.ProviderName2, get2.JSON200.Name)

	get3, err := apiClient.GetProvidersProviderIDWithResponse(
		ctx,
		*provider3.JSON201.ProviderID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, get3.StatusCode())
	assert.Equal(t, utils.ProviderName3, get3.JSON200.Name)

	log.Info().Msgf("End CreateGetDelete Provider tests")
}

func TestProvider_Errors(t *testing.T) {
	log.Info().Msgf("Begin Errors Provider tests")
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)
	if err != nil {
		t.Fatalf("new API client error %s", err.Error())
	}

	t.Run("Post_NoKind_BadRequest", func(t *testing.T) {
		provider, err := apiClient.PostProvidersWithResponse(
			ctx,
			utils.ProviderNoKind,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		log.Info().Msgf("Error Kind %s", provider.Body)
		assert.Equal(t, http.StatusBadRequest, provider.StatusCode())
	})

	t.Run("Post_NoName_BadRequest", func(t *testing.T) {
		provider, err := apiClient.PostProvidersWithResponse(
			ctx,
			utils.ProviderNoName,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		log.Info().Msgf("Error Name %s", provider.Body)
		assert.Equal(t, http.StatusBadRequest, provider.StatusCode())
	})

	t.Run("Post_NoApiEndpoint_UnprocessableEntity", func(t *testing.T) {
		provider, err := apiClient.PostProvidersWithResponse(
			ctx,
			utils.ProviderNoApiEndpoint,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		log.Info().Msgf("Error ApiEndpoint %s", provider.Body)
		assert.Equal(t, http.StatusUnprocessableEntity, provider.StatusCode())
	})

	t.Run("Post_BadApiCredentials_BadRequest", func(t *testing.T) {
		provider, err := apiClient.PostProvidersWithResponse(
			ctx,
			utils.ProviderBadCredentials,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		log.Info().Msgf("Error ApiCredentials %s", provider.Body)
		assert.Equal(t, http.StatusBadRequest, provider.StatusCode())
	})

	t.Run("Get_UnexistID_NotFound", func(t *testing.T) {
		provider, err := apiClient.GetProvidersProviderIDWithResponse(
			ctx,
			utils.ProviderUnexistID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, provider.StatusCode())
	})

	t.Run("Delete_UnexistID_NotFound", func(t *testing.T) {
		provider, err := apiClient.DeleteProvidersProviderIDWithResponse(
			ctx,
			utils.ProviderUnexistID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, provider.StatusCode())
	})

	t.Run("Get_WrongID_BadRequest", func(t *testing.T) {
		provider, err := apiClient.GetProvidersProviderIDWithResponse(
			ctx,
			utils.ProviderWrongID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, provider.StatusCode())
	})

	t.Run("Delete_WrongID_BadRequest", func(t *testing.T) {
		provider, err := apiClient.DeleteProvidersProviderIDWithResponse(
			ctx,
			utils.ProviderWrongID,
			AddJWTtoTheHeader,
			AddProjectIDtoTheHeader,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, provider.StatusCode())
	})
	log.Info().Msgf("End Errors Provider tests")
}

func TestProviderList(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	totalItems := 10
	offset := 0
	pageSize := 4

	name := "provider"
	for id := 0; id < totalItems; id++ {
		// Generate sequentialnames
		utils.Provider1Request.Name = fmt.Sprintf("%s%d", name, id)
		CreateProvider(t, ctx, apiClient, utils.Provider1Request)
	}

	// Checks if list resources return expected number of entries
	resList, err := apiClient.GetProvidersWithResponse(
		ctx,
		&api.GetProvidersParams{
			Offset:   &offset,
			PageSize: &pageSize,
		},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, len(*resList.JSON200.Providers), pageSize)
	assert.Equal(t, true, *resList.JSON200.HasNext)

	resList, err = apiClient.GetProvidersWithResponse(
		ctx,
		&api.GetProvidersParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, totalItems+numdefaultProviders, len(*resList.JSON200.Providers))
	assert.Equal(t, false, *resList.JSON200.HasNext)
}

func TestProviderList_ListEmpty(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	apiClient, err := GetAPIClient()
	require.NoError(t, err)

	resList, err := apiClient.GetProvidersWithResponse(
		ctx,
		&api.GetProvidersParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())
	assert.Equal(t, numdefaultProviders, len(*resList.JSON200.Providers))
}
