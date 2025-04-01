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
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/internal/worker/handlers"
	inv_handlers "github.com/open-edge-platform/infra-core/api/internal/worker/handlers/invhandlers"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	"github.com/open-edge-platform/infra-core/api/test/utils"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inventory "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

var (
	WorkloadTestName   = "Test Workload"
	WorkloadTestStatus = "Test Status"
	WorkloadTestKind   = api.WORKLOADKINDCLUSTER
)

func BuildFmFromWorkloadRequest(body api.Workload) []string {
	fm := []string{}
	fm = append(fm, "kind")
	if body.WorkloadId != nil {
		fm = append(fm, "workload_id")
	}
	if body.Name != nil {
		fm = append(fm, "name")
	}
	if body.Status != nil {
		fm = append(fm, "status")
	}
	return fm
}

func Test_WorkloadHandler_Job_Error(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	ctx := context.TODO()
	job := types.NewJob(ctx, BadOperation, types.Workload, nil, inv_handlers.WorkloadURLParams{})
	_, err := h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))
}

// check that we pass the expected filters to the inventory.
func Test_WorkloadHandler_List(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	// test List
	job := types.NewJob(
		context.TODO(), types.List, types.Workload,
		api.GetWorkloadsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
			Kind:     &WorkloadTestKind,
		}, nil,
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	listResources, ok := r.Payload.Data.(api.WorkloadList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 0, len(*listResources.Workloads))

	workload1 := inv_testing.CreateWorkload(t)

	job = types.NewJob(
		context.TODO(), types.List, types.Workload,
		api.GetWorkloadsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
		}, nil,
	)
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.WorkloadList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 1, len(*listResources.Workloads))

	filter := fmt.Sprintf("%s = %q", computev1.WorkloadResourceFieldResourceId, workload1.GetResourceId())
	orderBy := computev1.WorkloadResourceFieldResourceId
	job = types.NewJob(
		context.TODO(), types.List, types.Workload,
		api.GetWorkloadsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
			Filter:   &filter,
			OrderBy:  &orderBy,
		}, nil,
	)
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.WorkloadList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 1, len(*listResources.Workloads))

	// test List error - wrong params
	job = types.NewJob(
		context.TODO(), types.List, types.Workload,
		api.GetComputeHostsParams{}, nil,
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_WorkloadHandler_Post(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)
	body := api.Workload{
		Kind:   api.WORKLOADKINDCLUSTER,
		Name:   &WorkloadTestName,
		Status: &WorkloadTestStatus,
	}
	job := types.NewJob(
		context.TODO(), types.Post, types.Workload,
		&body, inv_handlers.WorkloadURLParams{},
	)
	r, err := h.Do(job)
	require.Equal(t, nil, err)
	assert.Equal(t, http.StatusCreated, r.Status)
	gotRes, ok := r.Payload.Data.(*api.Workload)
	require.True(t, ok)

	// Validate Post changes
	job = types.NewJob(context.TODO(), types.Get, types.Workload, nil, inv_handlers.WorkloadURLParams{
		WorkloadID: *gotRes.WorkloadId,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok = r.Payload.Data.(*api.Workload)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, WorkloadTestName, *gotRes.Name)

	// Post error - wrong body request format
	job = types.NewJob(
		context.TODO(), types.Post, types.Workload,
		&api.Host{}, inv_handlers.WorkloadURLParams{},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	inv_testing.DeleteResource(t, *gotRes.WorkloadId)
}

func Test_WorkloadHandler_Put(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	workloadResource := inv_testing.CreateWorkload(t)

	bodyUpdate := api.Workload{
		Kind:   api.WORKLOADKINDCLUSTER,
		Name:   &WorkloadTestName,
		Status: &WorkloadTestStatus,
	}

	job := types.NewJob(
		context.TODO(),
		types.Put,
		types.Workload,
		&bodyUpdate,
		inv_handlers.WorkloadURLParams{WorkloadID: workloadResource.ResourceId},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	// Validate Put changes
	job = types.NewJob(context.TODO(), types.Get, types.Workload, nil, inv_handlers.WorkloadURLParams{
		WorkloadID: workloadResource.ResourceId,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.Workload)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, WorkloadTestName, *gotRes.Name)

	// Update error - wrong body format
	job = types.NewJob(
		context.TODO(),
		types.Put,
		types.Workload,
		&api.Host{},
		inv_handlers.WorkloadURLParams{WorkloadID: workloadResource.ResourceId},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	// Update error - wrong params
	job = types.NewJob(
		context.TODO(),
		types.Put,
		types.OSResource,
		&bodyUpdate,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_WorkloadHandler_Patch(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	assert.NotEqual(t, h, nil)

	workloadResource := inv_testing.CreateWorkload(t)

	bodyUpdate := api.Workload{
		Kind:   api.WORKLOADKINDCLUSTER,
		Name:   &WorkloadTestName,
		Status: &WorkloadTestStatus,
	}

	job := types.NewJob(
		context.TODO(),
		types.Patch,
		types.Workload,
		&bodyUpdate,
		inv_handlers.WorkloadURLParams{WorkloadID: workloadResource.ResourceId},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	// Validate Patch changes
	job = types.NewJob(context.TODO(), types.Get, types.Workload, nil, inv_handlers.WorkloadURLParams{
		WorkloadID: workloadResource.ResourceId,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.Workload)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, WorkloadTestName, *gotRes.Name)
}

func Test_WorkloadHandler_PatchFieldMask(t *testing.T) {
	mockClient := utils.NewMockInventoryServiceClient(
		utils.MockResponses{
			ListResourcesResponse: &inventory.ListResourcesResponse{
				Resources: []*inventory.GetResourceResponse{},
			},
			GetResourceResponse: &inventory.GetResourceResponse{},
			UpdateResourceResponse: &inventory.Resource{
				Resource: &inventory.Resource_Workload{
					Workload: &computev1.WorkloadResource{},
				},
			},
		},
	)
	client := &clients.InventoryClientHandler{
		InvClient: mockClient,
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	bodyUpdate := api.Workload{
		Kind:   api.WORKLOADKINDCLUSTER,
		Name:   &WorkloadTestName,
		Status: &WorkloadTestStatus,
	}

	job := types.NewJob(
		context.TODO(),
		types.Patch,
		types.Workload,
		&bodyUpdate,
		inv_handlers.WorkloadURLParams{WorkloadID: "workload-12345678"},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	// test Patch FieldMask
	expectedPatchFieldMask := BuildFmFromWorkloadRequest(bodyUpdate)
	workload := &computev1.WorkloadResource{}
	expectedFieldMask, err := fieldmaskpb.New(workload, expectedPatchFieldMask...)
	require.NoError(t, err)

	if mockClient.LastUpdateResourceRequestFieldMask != nil {
		mockClient.LastUpdateResourceRequestFieldMask.Normalize()
		expectedFieldMask.Normalize()
		if !proto.Equal(expectedFieldMask, mockClient.LastUpdateResourceRequestFieldMask) {
			err = fmt.Errorf(
				"FieldMask is incorrectly constructed, expected: %s got: %s",
				expectedFieldMask.Paths,
				mockClient.LastUpdateResourceRequestFieldMask.Paths,
			)
		}
	} else {
		err = fmt.Errorf("no request in Mock Inventory")
	}
	assert.NoError(t, err)
}

func Test_WorkloadHandler_Get(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	workloadResource := inv_testing.CreateWorkload(t)

	job := types.NewJob(
		context.TODO(),
		types.Get,
		types.Workload,
		nil,
		inv_handlers.WorkloadURLParams{WorkloadID: workloadResource.ResourceId},
	)
	r, err := h.Do(job)
	require.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.Workload)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, workloadResource.Name, *gotRes.Name)

	// Get error - wrong params
	job = types.NewJob(
		context.TODO(),
		types.Get,
		types.Workload,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_WorkloadHandler_Delete(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)
	workloadResource := inv_testing.CreateWorkloadNoCleanup(t)

	job := types.NewJob(
		context.TODO(),
		types.Delete,
		types.Workload,
		nil,
		inv_handlers.WorkloadURLParams{WorkloadID: workloadResource.ResourceId},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.Status)

	// Delete error - wrong params
	job = types.NewJob(
		context.TODO(),
		types.Delete,
		types.Workload,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_Inventory_Workload_Integration(t *testing.T) {
	// verify the projection of the constants to Proto first;
	// we build a map using the field names of the proto stored in the
	// ProtoOu* slices in internal/work/handlers/workload.go. Elements must
	// have a mapping key otherwise we throw an error if there is no
	// alignment with OU proto in Inventory. Make sure to update these
	// two slices in internal/work/handlers/workload.go
	workloadResource := &computev1.WorkloadResource{}
	validateInventoryIntegration(
		t,
		workloadResource,
		api.Workload{},
		inv_handlers.OpenAPIWorkloadToProto,
		inv_handlers.OpenAPIWorkloadToProtoExcluded,
		maps.Values(inv_handlers.OpenAPIWorkloadToProto),
		true,
	)
}

// Test_WorkloadHandler_InvMockClient_Errors evaluates all
// Workload handler methods with mock inventory client
// that returns errors.
func Test_WorkloadHandler_InvMockClient_Errors(t *testing.T) {
	mockClient := utils.NewMockInventoryServiceClientError()
	client := &clients.InventoryClientHandler{
		InvClient: mockClient,
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	job := types.NewJob(
		context.TODO(), types.List, types.Workload,
		api.GetWorkloadsParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
			Kind:     &WorkloadTestKind,
		}, nil,
	)
	_, err := h.Do(job)
	assert.Error(t, err)

	body := api.Workload{
		Kind:   api.WORKLOADKINDCLUSTER,
		Name:   &WorkloadTestName,
		Status: &WorkloadTestStatus,
	}
	job = types.NewJob(
		context.TODO(), types.Post, types.Workload,
		&body, inv_handlers.WorkloadURLParams{},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	job = types.NewJob(
		context.TODO(),
		types.Put,
		types.Workload,
		&body,
		inv_handlers.WorkloadURLParams{WorkloadID: "workload-12345678"},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	job = types.NewJob(
		context.TODO(),
		types.Get,
		types.Workload,
		nil,
		inv_handlers.WorkloadURLParams{WorkloadID: "workload-12345678"},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	job = types.NewJob(
		context.TODO(),
		types.Delete,
		types.Workload,
		nil,
		inv_handlers.WorkloadURLParams{WorkloadID: "workload-12345678"},
	)
	_, err = h.Do(job)
	assert.Error(t, err)
}
