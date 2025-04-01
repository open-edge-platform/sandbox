// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/internal/worker/handlers"
	inv_handlers "github.com/open-edge-platform/infra-core/api/internal/worker/handlers/invhandlers"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	localaccountv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/localaccount/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

var (
	username = "testuser"
	sshKey   = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILtu+7Pdtj6ihyFynecnd+155AdxqvHhMRxvxdcQ8/D/ test-user@example.com"
)

func Test_LocalAccountHandler_Job_Error(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	ctx := context.TODO()
	job := types.NewJob(ctx, BadOperation, types.LocalAccount, nil, inv_handlers.LocalAccountURLParams{})
	_, err := h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))
}

// check that we pass the expected filters to the inventory.
func Test_LocalAccountHandler_List(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	// test List
	job := types.NewJob(
		context.TODO(), types.List, types.LocalAccount,
		api.GetLocalAccountsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
		}, nil,
	)
	r, err := h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusOK, r.Status)

	listResources, ok := r.Payload.Data.(api.LocalAccountList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 0, len(*listResources.LocalAccounts))

	localAccount1 := inv_testing.CreateLocalAccount(t, username, sshKey)

	job = types.NewJob(
		context.TODO(), types.List, types.LocalAccount,
		api.GetLocalAccountsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
		}, nil,
	)
	r, err = h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.LocalAccountList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 1, len(*listResources.LocalAccounts))

	filter := fmt.Sprintf("%s = %q", localaccountv1.LocalAccountResourceFieldResourceId, localAccount1.GetResourceId())
	orderBy := localaccountv1.LocalAccountResourceFieldResourceId
	job = types.NewJob(
		context.TODO(), types.List, types.LocalAccount,
		api.GetLocalAccountsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
			Filter:   &filter,
			OrderBy:  &orderBy,
		}, nil,
	)
	r, err = h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.LocalAccountList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 1, len(*listResources.LocalAccounts))

	// test List error - wrong params
	job = types.NewJob(
		context.TODO(), types.List, types.LocalAccount,
		api.GetComputeHostsParams{}, nil,
	)
	_, err = h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_LocalAccountHandler_Post(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)
	body := api.LocalAccount{
		Username: username,
		SshKey:   sshKey,
	}
	job := types.NewJob(
		context.TODO(), types.Post, types.LocalAccount,
		&body, inv_handlers.LocalAccountURLParams{},
	)
	r, err := h.Do(job)
	require.Equal(t, nil, err)
	assert.Equal(t, http.StatusCreated, r.Status)

	gotRes, ok := r.Payload.Data.(*api.LocalAccount)
	require.True(t, ok)

	// Validate Post changes
	job = types.NewJob(context.TODO(), types.Get, types.LocalAccount, nil, inv_handlers.LocalAccountURLParams{
		LocalAccountID: *gotRes.ResourceId,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok = r.Payload.Data.(*api.LocalAccount)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, username, gotRes.Username)

	// Post error - wrong body request format
	job = types.NewJob(
		context.TODO(), types.Post, types.LocalAccount,
		&api.Provider{}, inv_handlers.LocalAccountURLParams{},
	)
	_, err = h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	inv_testing.DeleteResource(t, *gotRes.ResourceId)
}

func Test_LocalAccountHandler_Put_Patch_Unsupported(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}

	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	localAccountResource := inv_testing.CreateLocalAccount(t, username, sshKey)

	bodyUpdate := api.LocalAccount{
		Username: username,
		SshKey:   sshKey,
	}

	// PUT unsupported
	job := types.NewJob(
		context.TODO(),
		types.Put,
		types.LocalAccount,
		&bodyUpdate,
		inv_handlers.LocalAccountURLParams{LocalAccountID: localAccountResource.ResourceId},
	)
	_, err := h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))

	// PATCH unsupported
	job = types.NewJob(
		context.TODO(),
		types.Patch,
		types.LocalAccount,
		&bodyUpdate,
		inv_handlers.LocalAccountURLParams{LocalAccountID: localAccountResource.ResourceId},
	)
	_, err = h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))
}

func Test_LocalAccountHandler_Get(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	job := types.NewJob(
		context.TODO(),
		types.Get,
		types.LocalAccount,
		nil,
		inv_handlers.LocalAccountURLParams{LocalAccountID: "localaccount-12345678"},
	)
	_, err := h.Do(job)
	assert.NotEqual(t, err, nil)

	localAccountResource := inv_testing.CreateLocalAccount(t, username, sshKey)

	job = types.NewJob(
		context.TODO(),
		types.Get,
		types.LocalAccount,
		nil,
		inv_handlers.LocalAccountURLParams{LocalAccountID: localAccountResource.ResourceId},
	)
	r, err := h.Do(job)
	require.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.LocalAccount)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, localAccountResource.Username, gotRes.Username)

	// Get error - wrong params
	job = types.NewJob(
		context.TODO(),
		types.Get,
		types.LocalAccount,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_LocalAccountHandler_Delete(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	localAccountResource := inv_testing.CreateLocalAccountNoCleanup(t, username, sshKey)

	job := types.NewJob(
		context.TODO(),
		types.Delete,
		types.LocalAccount,
		nil,
		inv_handlers.LocalAccountURLParams{LocalAccountID: localAccountResource.ResourceId},
	)
	r, err := h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusNoContent, r.Status)

	// Validate Delete changes
	job = types.NewJob(context.TODO(), types.Get, types.LocalAccount, nil, inv_handlers.LocalAccountURLParams{
		LocalAccountID: localAccountResource.ResourceId,
	})
	_, err = h.Do(job)
	assert.NotEqual(t, err, nil)

	// Delete error - wrong params
	job = types.NewJob(
		context.TODO(),
		types.Delete,
		types.LocalAccount,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}
