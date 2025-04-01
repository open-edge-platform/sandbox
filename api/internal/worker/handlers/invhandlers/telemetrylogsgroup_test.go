// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/internal/worker/handlers"
	inv_handlers "github.com/open-edge-platform/infra-core/api/internal/worker/handlers/invhandlers"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	"github.com/open-edge-platform/infra-core/api/test/utils"
	telemetryv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/telemetry/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

func Test_TelemetryLogsGroupHandler_Job_Error(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}

	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	ctx := context.TODO()
	job := types.NewJob(ctx, BadOperation, types.TelemetryLogsGroup, nil, inv_handlers.TelemetryLogsGroupURLParams{})
	_, err := h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))
}

func Test_TelemetryLogsGroupHandler_List(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	// test List
	job := types.NewJob(
		context.TODO(), types.List, types.TelemetryLogsGroup,
		api.GetTelemetryGroupsLogsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
		}, nil,
	)
	r, err := h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok := r.Payload.Data.(api.TelemetryLogsGroupList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 0, len(*listResources.TelemetryLogsGroups))

	inv_testing.CreateTelemetryGroupLogs(t, true)

	job = types.NewJob(
		context.TODO(), types.List, types.TelemetryLogsGroup,
		api.GetTelemetryGroupsLogsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
		}, nil,
	)
	r, err = h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.TelemetryLogsGroupList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 1, len(*listResources.TelemetryLogsGroups))

	// test List error - wrong params
	job = types.NewJob(
		context.TODO(), types.List, types.TelemetryLogsGroup,
		api.GetComputeHostsParams{}, nil,
	)
	_, err = h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_TelemetryLogsGroupHandler_Post(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	body := api.TelemetryLogsGroup{
		CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
		Groups:        []string{"syslog"},
		Name:          "syslog",
	}
	job := types.NewJob(
		context.TODO(), types.Post, types.TelemetryLogsGroup,
		&body, inv_handlers.TelemetryLogsGroupURLParams{},
	)
	r, err := h.Do(job)
	require.Equal(t, nil, err)
	assert.Equal(t, http.StatusCreated, r.Status)
	gotRes, ok := r.Payload.Data.(*api.TelemetryLogsGroup)
	require.True(t, ok)
	assert.NotNil(t, gotRes)

	// Validate Post
	job = types.NewJob(
		context.TODO(),
		types.Get,
		types.TelemetryLogsGroup,
		nil,
		inv_handlers.TelemetryLogsGroupURLParams{TelemetryLogsGroupID: *gotRes.TelemetryLogsGroupId},
	)
	r, err = h.Do(job)
	require.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, r.Status)
	gotRes, ok = r.Payload.Data.(*api.TelemetryLogsGroup)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, api.TELEMETRYCOLLECTORKINDHOST, gotRes.CollectorKind)

	// Post error - wrong body request format
	job = types.NewJob(
		context.TODO(), types.Post, types.TelemetryLogsGroup,
		&api.Host{}, inv_handlers.TelemetryLogsGroupURLParams{},
	)
	_, err = h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_TelemetryLogsGroupHandler_Get(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	job := types.NewJob(
		context.TODO(),
		types.Get,
		types.TelemetryLogsGroup,
		nil,
		inv_handlers.TelemetryLogsGroupURLParams{TelemetryLogsGroupID: "telemetrygroup-12345678"},
	)
	_, err := h.Do(job)
	assert.Error(t, err)

	telGroupRes := inv_testing.CreateTelemetryGroupLogs(t, true)

	job = types.NewJob(
		context.TODO(),
		types.Get,
		types.TelemetryLogsGroup,
		nil,
		inv_handlers.TelemetryLogsGroupURLParams{TelemetryLogsGroupID: telGroupRes.ResourceId},
	)
	r, err := h.Do(job)
	require.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, r.Status)
	gotRes, ok := r.Payload.Data.(*api.TelemetryLogsGroup)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, telGroupRes.Name, gotRes.Name)

	// Get error - wrong params
	job = types.NewJob(
		context.TODO(),
		types.Get,
		types.TelemetryLogsGroup,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_TelemetryLogsGroupHandler_Delete(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	job := types.NewJob(
		context.TODO(),
		types.Delete,
		types.TelemetryLogsGroup,
		nil,
		inv_handlers.TelemetryLogsGroupURLParams{TelemetryLogsGroupID: "telemetrygroup-12345679"},
	)
	_, err := h.Do(job)
	require.Error(t, err)

	telGroupRes := inv_testing.CreateTelemetryGroupLogs(t, false)

	job = types.NewJob(
		context.TODO(),
		types.Delete,
		types.TelemetryLogsGroup,
		nil,
		inv_handlers.TelemetryLogsGroupURLParams{TelemetryLogsGroupID: telGroupRes.ResourceId},
	)
	r, err := h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusNoContent, r.Status)

	// Delete error - wrong params
	job = types.NewJob(
		context.TODO(),
		types.Delete,
		types.TelemetryLogsGroup,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_Inventory_TelemetryLogsGroup_Integration(t *testing.T) {
	// verify the projection of the constants to Proto first;
	// we build a map using the field names of the proto stored in the
	// ProtoOu* slices in internal/work/handlers/telemetrylogsgroup.go. Elements must
	// have a mapping key otherwise we throw an error if there is no
	// alignment with the proto in Inventory. Make sure to update these
	// two slices in internal/work/handlers/telemetrylogsgroup.go
	telemetryGroupResource := &telemetryv1.TelemetryGroupResource{}
	validateInventoryIntegration(
		t,
		telemetryGroupResource,
		api.TelemetryLogsGroup{},
		inv_handlers.OpenAPITelemetryLogsGroupResourceToProto,
		inv_handlers.OpenAPITelemetryLogsGroupToProtoExcluded,
		maps.Values(inv_handlers.OpenAPITelemetryLogsGroupResourceToProto),
		true,
	)
}

func Test_TelemetryLogsGroupHandler_InvMockClient_Errors(t *testing.T) {
	mockClient := utils.NewMockInventoryServiceClientError()
	client := &clients.InventoryClientHandler{
		InvClient: mockClient,
	}
	h := handlers.NewHandlers(client, nil)
	assert.NotNil(t, h)

	job := types.NewJob(
		context.TODO(), types.List, types.TelemetryLogsGroup,
		api.GetTelemetryGroupsLogsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
		}, nil,
	)
	_, err := h.Do(job)
	assert.NotEqual(t, nil, err)

	body := api.TelemetryLogsGroup{
		CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
		Groups:        []string{"syslog", "kmesg"},
		Name:          "System & Kernel Logs",
	}
	job = types.NewJob(
		context.TODO(), types.Post, types.TelemetryLogsGroup,
		&body, inv_handlers.TelemetryLogsGroupURLParams{},
	)
	_, err = h.Do(job)
	assert.NotEqual(t, nil, err)

	job = types.NewJob(
		context.TODO(),
		types.Put,
		types.TelemetryLogsGroup,
		&body,
		inv_handlers.TelemetryLogsGroupURLParams{TelemetryLogsGroupID: "telemetrygroup-12345678"},
	)
	_, err = h.Do(job)
	assert.NotEqual(t, nil, err)

	job = types.NewJob(
		context.TODO(),
		types.Get,
		types.TelemetryLogsGroup,
		nil,
		inv_handlers.TelemetryLogsGroupURLParams{TelemetryLogsGroupID: "telemetrygroup-12345678"},
	)
	_, err = h.Do(job)
	assert.NotEqual(t, nil, err)

	job = types.NewJob(
		context.TODO(),
		types.Delete,
		types.TelemetryLogsGroup,
		nil,
		inv_handlers.TelemetryLogsGroupURLParams{TelemetryLogsGroupID: "telemetrygroup-12345678"},
	)
	_, err = h.Do(job)
	assert.NotEqual(t, nil, err)
}
