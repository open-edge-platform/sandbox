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

func Test_TelemetryMetricsGroupHandler_Job_Error(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}

	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	ctx := context.TODO()
	job := types.NewJob(ctx, BadOperation, types.TelemetryMetricsGroup, nil, inv_handlers.TelemetryMetricsGroupURLParams{})
	_, err := h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))
}

func Test_TelemetryMetricsGroupHandler_List(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	// test List
	job := types.NewJob(
		context.TODO(), types.List, types.TelemetryMetricsGroup,
		api.GetTelemetryGroupsMetricsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
		}, nil,
	)
	r, err := h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok := r.Payload.Data.(api.TelemetryMetricsGroupList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 0, len(*listResources.TelemetryMetricsGroups))

	inv_testing.CreateTelemetryGroupMetrics(t, true)

	job = types.NewJob(
		context.TODO(), types.List, types.TelemetryMetricsGroup,
		api.GetTelemetryGroupsMetricsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
		}, nil,
	)
	r, err = h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.TelemetryMetricsGroupList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 1, len(*listResources.TelemetryMetricsGroups))

	// test List error - wrong params
	job = types.NewJob(
		context.TODO(), types.List, types.TelemetryMetricsGroup,
		api.GetComputeHostsParams{}, nil,
	)
	_, err = h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_TelemetryMetricsGroupHandler_Post(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	body := api.TelemetryMetricsGroup{
		CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
		Groups:        []string{"net", "ethtool"},
		Name:          "NW Usage",
	}
	job := types.NewJob(
		context.TODO(), types.Post, types.TelemetryMetricsGroup,
		&body, inv_handlers.TelemetryMetricsGroupURLParams{},
	)
	r, err := h.Do(job)
	require.Equal(t, nil, err)
	assert.Equal(t, http.StatusCreated, r.Status)
	gotRes, ok := r.Payload.Data.(*api.TelemetryMetricsGroup)
	require.True(t, ok)
	assert.NotNil(t, gotRes)

	// Validate Post
	job = types.NewJob(
		context.TODO(),
		types.Get,
		types.TelemetryMetricsGroup,
		nil,
		inv_handlers.TelemetryMetricsGroupURLParams{TelemetryMetricsGroupID: *gotRes.TelemetryMetricsGroupId},
	)
	r, err = h.Do(job)
	require.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, r.Status)
	gotRes, ok = r.Payload.Data.(*api.TelemetryMetricsGroup)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, api.TELEMETRYCOLLECTORKINDHOST, gotRes.CollectorKind)

	// Post error - wrong body request format
	job = types.NewJob(
		context.TODO(), types.Post, types.TelemetryMetricsGroup,
		&api.Host{}, inv_handlers.TelemetryMetricsGroupURLParams{},
	)
	_, err = h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	inv_testing.DeleteResource(t, *gotRes.TelemetryMetricsGroupId)
}

func Test_TelemetryMetricsGroupHandler_Get(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	job := types.NewJob(
		context.TODO(),
		types.Get,
		types.TelemetryMetricsGroup,
		nil,
		inv_handlers.TelemetryMetricsGroupURLParams{TelemetryMetricsGroupID: "telemetrygroup-12345678"},
	)
	_, err := h.Do(job)
	assert.Error(t, err)

	telGroupRes := inv_testing.CreateTelemetryGroupMetrics(t, true)

	job = types.NewJob(
		context.TODO(),
		types.Get,
		types.TelemetryMetricsGroup,
		nil,
		inv_handlers.TelemetryMetricsGroupURLParams{TelemetryMetricsGroupID: telGroupRes.ResourceId},
	)
	r, err := h.Do(job)
	require.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, r.Status)
	gotRes, ok := r.Payload.Data.(*api.TelemetryMetricsGroup)
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

func Test_TelemetryMetricsGroupHandler_Delete(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	job := types.NewJob(
		context.TODO(),
		types.Delete,
		types.TelemetryMetricsGroup,
		nil,
		inv_handlers.TelemetryMetricsGroupURLParams{TelemetryMetricsGroupID: "telemetrygroup-12345679"},
	)
	_, err := h.Do(job)
	assert.Error(t, err)

	telGroupRes := inv_testing.CreateTelemetryGroupMetrics(t, false)

	job = types.NewJob(
		context.TODO(),
		types.Delete,
		types.TelemetryMetricsGroup,
		nil,
		inv_handlers.TelemetryMetricsGroupURLParams{TelemetryMetricsGroupID: telGroupRes.ResourceId},
	)
	r, err := h.Do(job)
	require.Equal(t, err, nil)
	assert.Equal(t, http.StatusNoContent, r.Status)

	// Delete error - wrong params
	job = types.NewJob(
		context.TODO(),
		types.Delete,
		types.TelemetryMetricsGroup,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	require.NotEqual(t, err, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_Inventory_TelemetryMetricsGroup_Integration(t *testing.T) {
	// verify the projection of the constants to Proto first;
	// we build a map using the field names of the proto stored in the
	// ProtoOu* slices in internal/work/handlers/telemetrymetricsgroup.go. Elements must
	// have a mapping key otherwise we throw an error if there is no
	// alignment with the proto in Inventory. Make sure to update these
	// two slices in internal/work/handlers/telemetrymetricsgroup.go
	telemetryGroupResource := &telemetryv1.TelemetryGroupResource{}
	validateInventoryIntegration(
		t,
		telemetryGroupResource,
		api.TelemetryMetricsGroup{},
		inv_handlers.OpenAPITelemetryMetricsGroupResourceToProto,
		inv_handlers.OpenAPITelemetryMetricsGroupToProtoExcluded,
		maps.Values(inv_handlers.OpenAPITelemetryMetricsGroupResourceToProto),
		true,
	)
}

func Test_TelemetryMetricsGroupHandler_InvMockClient_Errors(t *testing.T) {
	mockClient := utils.NewMockInventoryServiceClientError()
	client := &clients.InventoryClientHandler{
		InvClient: mockClient,
	}
	h := handlers.NewHandlers(client, nil)
	assert.NotNil(t, h)

	job := types.NewJob(
		context.TODO(), types.List, types.TelemetryMetricsGroup,
		api.GetTelemetryGroupsMetricsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
		}, nil,
	)
	_, err := h.Do(job)
	assert.NotEqual(t, nil, err)

	body := api.TelemetryMetricsGroup{
		CollectorKind: api.TELEMETRYCOLLECTORKINDHOST,
		Groups:        []string{"cpu", "mem"},
		Name:          "HW Usage",
	}
	job = types.NewJob(
		context.TODO(), types.Post, types.TelemetryMetricsGroup,
		&body, inv_handlers.TelemetryMetricsGroupURLParams{},
	)
	_, err = h.Do(job)
	assert.NotEqual(t, nil, err)

	job = types.NewJob(
		context.TODO(),
		types.Put,
		types.TelemetryMetricsGroup,
		&body,
		inv_handlers.TelemetryMetricsGroupURLParams{TelemetryMetricsGroupID: "telemetrygroup-12345678"},
	)
	_, err = h.Do(job)
	assert.NotEqual(t, nil, err)

	job = types.NewJob(
		context.TODO(),
		types.Get,
		types.TelemetryMetricsGroup,
		nil,
		inv_handlers.TelemetryMetricsGroupURLParams{TelemetryMetricsGroupID: "telemetrygroup-12345678"},
	)
	_, err = h.Do(job)
	assert.NotEqual(t, nil, err)

	job = types.NewJob(
		context.TODO(),
		types.Delete,
		types.TelemetryMetricsGroup,
		nil,
		inv_handlers.TelemetryMetricsGroupURLParams{TelemetryMetricsGroupID: "telemetrygroup-12345678"},
	)
	_, err = h.Do(job)
	assert.NotEqual(t, nil, err)
}
