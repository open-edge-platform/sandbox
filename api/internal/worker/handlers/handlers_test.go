// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package handlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/internal/worker/handlers"
	inv_handlers "github.com/open-edge-platform/infra-core/api/internal/worker/handlers/invhandlers"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	"github.com/open-edge-platform/infra-core/api/test/utils"
	inventory "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
)

func TestHandler(t *testing.T) {
	mockClient := utils.NewMockInventoryServiceClient(
		utils.MockResponses{
			ListResourcesResponse: &inventory.ListResourcesResponse{
				Resources: []*inventory.GetResourceResponse{},
			},
		},
	)
	client := &clients.InventoryClientHandler{
		InvClient: mockClient,
	}

	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	ctx := context.TODO()
	job := types.NewJob(
		ctx,
		types.List,
		types.Host,
		api.GetComputeHostsParams{},
		inv_handlers.HostURLParams{},
	)
	r, err := h.Do(job)
	t.Logf("Returned response is: %v", r)
	assert.Equal(t, http.StatusOK, r.Status)
	assert.NoError(t, err)

	jobError := types.NewJob(
		ctx,
		types.List,
		"",
		api.GetComputeHostsParams{},
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(jobError)
	assert.Error(t, err)
}
