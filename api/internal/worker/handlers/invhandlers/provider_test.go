// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/internal/worker/handlers"
	inv_handlers "github.com/open-edge-platform/infra-core/api/internal/worker/handlers/invhandlers"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	"github.com/open-edge-platform/infra-core/api/test/utils"
	providerv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/provider/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

var (
	providerKinds  = []api.ProviderKind{api.PROVIDERKINDBAREMETAL}
	providerVendor = api.PROVIDERVENDORLENOVOLXCA
	name           = "SC LXCA"
	apiEndpoint    = "https://192.168.201.3/"
	apiCredentials = []string{"v1/lxca/user", "v1/lxca/password"}
	providerConfig = "foo"
)

func Test_ProviderHandler_Job_Error(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	ctx := context.TODO()
	job := types.NewJob(ctx, BadOperation, types.Provider, nil, inv_handlers.ProviderURLParams{})
	_, err := h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))
}

// check that we pass the expected filters to the inventory.
func Test_ProviderHandler_List(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	// test List
	job := types.NewJob(
		context.TODO(), types.List, types.Provider,
		api.GetProvidersParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
		}, nil,
	)
	r, err := h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusOK, r.Status)

	listResources, ok := r.Payload.Data.(api.ProviderList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 0, len(*listResources.Providers))

	provider1 := inv_testing.CreateProvider(t, name)

	job = types.NewJob(
		context.TODO(), types.List, types.Provider,
		api.GetProvidersParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
		}, nil,
	)
	r, err = h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.ProviderList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 1, len(*listResources.Providers))

	filter := fmt.Sprintf("%s = %q", providerv1.ProviderResourceFieldResourceId, provider1.GetResourceId())
	orderBy := providerv1.ProviderResourceFieldResourceId
	job = types.NewJob(
		context.TODO(), types.List, types.Provider,
		api.GetProvidersParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
			Filter:   &filter,
			OrderBy:  &orderBy,
		}, nil,
	)
	r, err = h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.ProviderList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 1, len(*listResources.Providers))

	// test List error - wrong params
	job = types.NewJob(
		context.TODO(), types.List, types.Provider,
		api.GetComputeHostsParams{}, nil,
	)
	_, err = h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_ProviderHandler_Post(t *testing.T) {
	for _, pk := range providerKinds {
		t.Run(string(pk), func(t *testing.T) {
			client := &clients.InventoryClientHandler{
				InvClient: inv_testing.TestClients[inv_testing.APIClient],
			}
			h := handlers.NewHandlers(client, nil)
			require.NotNil(t, h)
			body := api.Provider{
				ProviderKind:   pk,
				ProviderVendor: &providerVendor,
				Name:           name,
				ApiEndpoint:    apiEndpoint,
				ApiCredentials: &apiCredentials,
				Config:         &providerConfig,
			}
			job := types.NewJob(
				context.TODO(), types.Post, types.Provider,
				&body, inv_handlers.ProviderURLParams{},
			)
			r, err := h.Do(job)
			require.Equal(t, nil, err)
			assert.Equal(t, http.StatusCreated, r.Status)

			gotRes, ok := r.Payload.Data.(*api.Provider)
			require.True(t, ok)

			// Validate Post changes
			job = types.NewJob(context.TODO(), types.Get, types.Provider, nil, inv_handlers.ProviderURLParams{
				ProviderID: *gotRes.ProviderID,
			})
			r, err = h.Do(job)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, r.Status)

			gotRes, ok = r.Payload.Data.(*api.Provider)
			require.True(t, ok)
			assert.NotNil(t, gotRes)
			assert.Equal(t, name, gotRes.Name)

			// Post error - wrong body request format
			job = types.NewJob(
				context.TODO(), types.Post, types.Provider,
				&api.Host{}, inv_handlers.ProviderURLParams{},
			)
			_, err = h.Do(job)
			require.NotEqual(t, err, nil)
			assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

			inv_testing.DeleteResource(t, *gotRes.ProviderID)
		})
	}
}

func Test_ProviderHandler_Put_Patch_Unsupported(t *testing.T) {
	for _, pk := range providerKinds {
		client := &clients.InventoryClientHandler{
			InvClient: inv_testing.TestClients[inv_testing.APIClient],
		}
		h := handlers.NewHandlers(client, nil)
		require.NotNil(t, h)

		providerName := uuid.NewString()
		providerResource := inv_testing.CreateProvider(t, providerName)

		bodyUpdate := api.Provider{
			ProviderKind:   pk,
			ProviderVendor: &providerVendor,
			Name:           providerName,
		}

		// PUT unsupported
		job := types.NewJob(
			context.TODO(),
			types.Put,
			types.Provider,
			&bodyUpdate,
			inv_handlers.ProviderURLParams{ProviderID: providerResource.ResourceId},
		)
		_, err := h.Do(job)
		require.NotEqual(t, err, nil)
		assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))

		// PATCH unsupported
		job = types.NewJob(
			context.TODO(),
			types.Patch,
			types.Provider,
			&bodyUpdate,
			inv_handlers.ProviderURLParams{ProviderID: providerResource.ResourceId},
		)
		_, err = h.Do(job)
		require.NotEqual(t, err, nil)
		assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))
	}
}

func Test_ProviderHandler_Get(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	job := types.NewJob(
		context.TODO(),
		types.Get,
		types.Provider,
		nil,
		inv_handlers.ProviderURLParams{ProviderID: "provider-12345678"},
	)
	_, err := h.Do(job)
	assert.NotEqual(t, err, nil)

	providerResource := inv_testing.CreateProvider(t, name)

	job = types.NewJob(
		context.TODO(),
		types.Get,
		types.Provider,
		nil,
		inv_handlers.ProviderURLParams{ProviderID: providerResource.ResourceId},
	)
	r, err := h.Do(job)
	require.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.Provider)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, providerResource.Name, gotRes.Name)

	// Get error - wrong params
	job = types.NewJob(
		context.TODO(),
		types.Get,
		types.Provider,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_ProviderHandler_Delete(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	providerResource := inv_testing.CreateProviderbNoCleanup(t, name)

	job := types.NewJob(
		context.TODO(),
		types.Delete,
		types.Provider,
		nil,
		inv_handlers.ProviderURLParams{ProviderID: providerResource.ResourceId},
	)
	r, err := h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusNoContent, r.Status)

	// Validate Delete changes
	job = types.NewJob(context.TODO(), types.Get, types.Provider, nil, inv_handlers.ProviderURLParams{
		ProviderID: providerResource.ResourceId,
	})
	_, err = h.Do(job)
	assert.NotEqual(t, err, nil)

	// Delete error - wrong params
	job = types.NewJob(
		context.TODO(),
		types.Delete,
		types.Provider,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_Inventory_Provider_Integration(t *testing.T) {
	// verify the projection of the constants to Proto first;
	// we build a map using the field names of the proto stored in the
	// ProtoProvider* slices in internal/work/handlers/provider.go. Elements must
	// have a mapping key otherwise we throw an error if there is no
	// alignment with Provider proto in Inventory. Make sure to update these
	// two slices in internal/work/handlers/provider.go
	providerResource := &providerv1.ProviderResource{}
	validateInventoryIntegration(
		t,
		providerResource,
		api.Provider{},
		inv_handlers.OpenAPIProviderToProto,
		inv_handlers.OpenAPIProviderToProtoExcluded,
		maps.Values(inv_handlers.OpenAPIProviderToProto),
		true,
	)
}

// Test_ProviderHandler_InvMockClient_Errors evaluates all
// Provider handler methods with mock inventory client
// that returns errors.
func Test_ProviderHandler_InvMockClient_Errors(t *testing.T) {
	for _, pk := range providerKinds {
		mockClient := utils.NewMockInventoryServiceClientError()
		client := &clients.InventoryClientHandler{
			InvClient: mockClient,
		}
		h := handlers.NewHandlers(client, nil)
		require.NotNil(t, h)

		job := types.NewJob(
			context.TODO(), types.List, types.Provider,
			api.GetProvidersParams{
				Offset:   &pgOffset,
				PageSize: &pgSize,
			}, nil,
		)
		_, err := h.Do(job)
		assert.NotEqual(t, err, nil)

		body := api.Provider{
			ProviderKind:   pk,
			ProviderVendor: &providerVendor,
			Name:           name,
			ApiEndpoint:    apiEndpoint,
			ApiCredentials: &apiCredentials,
			Config:         &providerConfig,
		}
		job = types.NewJob(
			context.TODO(), types.Post, types.Provider,
			&body, inv_handlers.ProviderURLParams{},
		)
		_, err = h.Do(job)
		assert.NotEqual(t, err, nil)

		job = types.NewJob(
			context.TODO(),
			types.Put,
			types.Provider,
			&body,
			inv_handlers.ProviderURLParams{ProviderID: "provider-12345678"},
		)
		_, err = h.Do(job)
		assert.NotEqual(t, err, nil)

		job = types.NewJob(
			context.TODO(),
			types.Get,
			types.Provider,
			nil,
			inv_handlers.ProviderURLParams{ProviderID: "provider-12345678"},
		)
		_, err = h.Do(job)
		assert.NotEqual(t, err, nil)

		job = types.NewJob(
			context.TODO(),
			types.Delete,
			types.Provider,
			nil,
			inv_handlers.ProviderURLParams{ProviderID: "provider-12345678"},
		)
		_, err = h.Do(job)
		assert.NotEqual(t, err, nil)
	}
}
