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

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/internal/worker/handlers"
	inv_handlers "github.com/open-edge-platform/infra-core/api/internal/worker/handlers/invhandlers"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	"github.com/open-edge-platform/infra-core/api/test/utils"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

var (
	MemberTestInstance = "inst-12345678"
	MemberTestWorkload = "workload-12345678"
)

func Test_WorkloadMemberHandler_Query_Error(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	ctx := context.TODO()
	job := types.NewJob(ctx, BadOperation, types.WorkloadMember, nil, inv_handlers.WorkloadURLParams{})
	_, err := h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))
}

// check that we pass the expected filters to the inventory.
func Test_WorkloadMemberHandler_List(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	// test List
	job := types.NewJob(
		context.TODO(), types.List, types.WorkloadMember,
		api.GetWorkloadMembersParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
		}, nil,
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	listResources, ok := r.Payload.Data.(api.WorkloadMemberList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 0, len(*listResources.WorkloadMembers))
	osResource := inv_testing.CreateOs(t)
	hostResource := inv_testing.CreateHost(t, nil, nil)
	instanceResource := inv_testing.CreateInstance(t, hostResource, osResource)
	workloadResource := inv_testing.CreateWorkload(t)
	workloadmember1 := inv_testing.CreateWorkloadMember(t, workloadResource, instanceResource)

	job = types.NewJob(
		context.TODO(), types.List, types.WorkloadMember,
		api.GetWorkloadMembersParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
		}, nil,
	)
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.WorkloadMemberList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 1, len(*listResources.WorkloadMembers))

	filter := fmt.Sprintf("%s = %q", computev1.WorkloadMemberFieldResourceId, workloadmember1.GetResourceId())
	orderBy := computev1.WorkloadMemberFieldResourceId
	job = types.NewJob(
		context.TODO(), types.List, types.WorkloadMember,
		api.GetWorkloadMembersParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
			Filter:   &filter,
			OrderBy:  &orderBy,
		}, nil,
	)
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.WorkloadMemberList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 1, len(*listResources.WorkloadMembers))

	// test List error - wrong params
	job = types.NewJob(
		context.TODO(), types.List, types.WorkloadMember,
		api.GetComputeHostsParams{}, nil,
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_WorkloadMemberHandler_Post(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	osResource := inv_testing.CreateOs(t)
	hostResource := inv_testing.CreateHost(t, nil, nil)
	instanceResource := inv_testing.CreateInstance(t, hostResource, osResource)
	workloadResource := inv_testing.CreateWorkload(t)

	body := api.WorkloadMember{
		Kind:       api.WORKLOADMEMBERKINDCLUSTERNODE,
		InstanceId: &instanceResource.ResourceId,
		WorkloadId: &workloadResource.ResourceId,
	}
	job := types.NewJob(
		context.TODO(), types.Post, types.WorkloadMember,
		&body, inv_handlers.WorkloadMemberURLParams{},
	)
	r, err := h.Do(job)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusCreated, r.Status)
	gotRes, ok := r.Payload.Data.(*api.WorkloadMember)
	require.True(t, ok)

	// Validate Post changes
	job = types.NewJob(context.TODO(), types.Get, types.WorkloadMember, nil, inv_handlers.WorkloadMemberURLParams{
		WorkloadMemberID: *gotRes.WorkloadMemberId,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok = r.Payload.Data.(*api.WorkloadMember)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, api.WORKLOADMEMBERKINDCLUSTERNODE, gotRes.Kind)

	// Post error - wrong body request format
	job = types.NewJob(
		context.TODO(), types.Post, types.WorkloadMember,
		&api.Host{}, inv_handlers.WorkloadMemberURLParams{},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	inv_testing.DeleteResource(t, *gotRes.WorkloadMemberId)
}

func Test_WorkloadMemberHandler_Put_Patch_Unsupported(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	osResource := inv_testing.CreateOs(t)
	hostResource := inv_testing.CreateHost(t, nil, nil)
	instanceResource := inv_testing.CreateInstance(t, hostResource, osResource)
	workloadResource := inv_testing.CreateWorkload(t)
	workloadMemberResource := inv_testing.CreateWorkloadMember(t, workloadResource, instanceResource)

	bodyUpdate := api.WorkloadMember{
		Kind:       api.WORKLOADMEMBERKINDCLUSTERNODE,
		InstanceId: &MemberTestInstance,
		WorkloadId: &MemberTestWorkload,
	}

	// PUT unsupported
	job := types.NewJob(
		context.TODO(),
		types.Put,
		types.WorkloadMember,
		&bodyUpdate,
		inv_handlers.WorkloadMemberURLParams{WorkloadMemberID: workloadMemberResource.ResourceId},
	)
	_, err := h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))

	// PATCH unsupported
	job = types.NewJob(
		context.TODO(),
		types.Patch,
		types.WorkloadMember,
		&bodyUpdate,
		inv_handlers.WorkloadMemberURLParams{WorkloadMemberID: workloadMemberResource.ResourceId},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))
}

func Test_WorkloadMemberHandler_Get(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	job := types.NewJob(
		context.TODO(),
		types.Get,
		types.WorkloadMember,
		nil,
		inv_handlers.WorkloadMemberURLParams{WorkloadMemberID: "workloadmember-12345678"},
	)
	_, err := h.Do(job)
	assert.Error(t, err)

	osResource := inv_testing.CreateOs(t)
	hostResource := inv_testing.CreateHost(t, nil, nil)
	instanceResource := inv_testing.CreateInstance(t, hostResource, osResource)
	workloadResource := inv_testing.CreateWorkload(t)
	workloadMemberResource := inv_testing.CreateWorkloadMember(t, workloadResource, instanceResource)

	job = types.NewJob(context.TODO(), types.Get, types.WorkloadMember, nil, inv_handlers.WorkloadMemberURLParams{
		WorkloadMemberID: workloadMemberResource.ResourceId,
	})
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.WorkloadMember)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, workloadResource.ResourceId, *gotRes.Workload.ResourceId)

	// Get error - wrong params
	job = types.NewJob(
		context.TODO(),
		types.Get,
		types.WorkloadMember,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_WorkloadMemberHandler_Delete(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	osResource := inv_testing.CreateOs(t)
	hostResource := inv_testing.CreateHost(t, nil, nil)
	instanceResource := inv_testing.CreateInstance(t, hostResource, osResource)
	workloadResource := inv_testing.CreateWorkload(t)
	workloadMemberResource := inv_testing.CreateWorkloadMemberNoCleanup(t, workloadResource, instanceResource)

	job := types.NewJob(
		context.TODO(),
		types.Delete,
		types.WorkloadMember,
		nil,
		inv_handlers.WorkloadMemberURLParams{WorkloadMemberID: workloadMemberResource.ResourceId},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.Status)

	// Validate Delete changes
	job = types.NewJob(context.TODO(), types.Get, types.WorkloadMember, nil, inv_handlers.WorkloadMemberURLParams{
		WorkloadMemberID: workloadMemberResource.ResourceId,
	})
	_, err = h.Do(job)
	assert.Error(t, err)

	// Delete error - wrong params
	job = types.NewJob(
		context.TODO(),
		types.Delete,
		types.WorkloadMember,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_Inventory_WorkloadMember_Integration(t *testing.T) {
	// verify the projection of the constants to Proto first;
	// we build a map using the field names of the proto stored in the
	// ProtoOu* slices in internal/work/handlers/workload_member.go. Elements must
	// have a mapping key otherwise we throw an error if there is no
	// alignment with WorkloadMember proto in Inventory. Make sure to update these
	// two slices in internal/work/handlers/workload_member.go
	member := &computev1.WorkloadMember{}
	validateInventoryIntegration(
		t,
		member,
		api.WorkloadMember{},
		inv_handlers.OpenAPIWorkloadMemberToProto,
		inv_handlers.OpenAPIWorkloadMemberToProtoExcluded,
		maps.Values(inv_handlers.OpenAPIWorkloadMemberToProto),
		true,
	)
}

// Test_WorkloadMemberHandler_InvMockClient_Errors evaluates all
// Workload member handler methods with mock inventory client
// that returns errors.
func Test_WorkloadMemberHandler_InvMockClient_Errors(t *testing.T) {
	mockClient := utils.NewMockInventoryServiceClientError()
	client := &clients.InventoryClientHandler{
		InvClient: mockClient,
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	job := types.NewJob(
		context.TODO(), types.List, types.WorkloadMember,
		api.GetWorkloadMembersParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
		}, nil,
	)
	_, err := h.Do(job)
	assert.Error(t, err)

	body := api.WorkloadMember{
		Kind:       api.WORKLOADMEMBERKINDCLUSTERNODE,
		InstanceId: &MemberTestInstance,
		WorkloadId: &MemberTestWorkload,
	}
	job = types.NewJob(
		context.TODO(), types.Post, types.WorkloadMember,
		&body, inv_handlers.WorkloadURLParams{},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	job = types.NewJob(
		context.TODO(),
		types.Put,
		types.WorkloadMember,
		&body,
		inv_handlers.WorkloadMemberURLParams{WorkloadMemberID: "workloadmember-12345678"},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	job = types.NewJob(
		context.TODO(),
		types.Get,
		types.WorkloadMember,
		nil,
		inv_handlers.WorkloadMemberURLParams{WorkloadMemberID: "workloadmember-12345678"},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	job = types.NewJob(
		context.TODO(),
		types.Delete,
		types.WorkloadMember,
		nil,
		inv_handlers.WorkloadMemberURLParams{WorkloadMemberID: "workloadmember-12345678"},
	)
	_, err = h.Do(job)
	assert.Error(t, err)
}
