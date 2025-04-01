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
	inventory "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	ou_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/ou/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

var ouName = "Testou"

func BuildFmFromOU(body api.OU) []string {
	fm := []string{}
	fm = append(fm, "name")
	if body.ParentOu != nil {
		fm = append(fm, "parent_ou")
	}
	if body.Metadata != nil {
		fm = append(fm, "metadata")
	}
	if body.OuKind != nil {
		fm = append(fm, "ou_kind")
	}
	return fm
}

func Test_OUHandler_Job_Error(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	// test List
	job := types.NewJob(
		context.TODO(), BadOperation, types.OU,
		nil, inv_handlers.HostURLParams{},
	)
	_, err := h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotImplemented, errors.ErrorToHTTPStatus(err))
}

// check that we pass the expected filters to the inventory.
func Test_OUHandler_List(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)
	ctx := context.TODO()

	// test List
	job := types.NewJob(
		ctx, types.List, types.OU,
		api.GetOusParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
			Parent:   &utils.OUUnexistID,
		}, nil,
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok := r.Payload.Data.(api.OUsList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 0, len(*listResources.OUs))

	ou1 := inv_testing.CreateOu(t, nil)

	job = types.NewJob(
		ctx, types.List, types.OU,
		api.GetOusParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
		}, nil,
	)
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.OUsList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 1, len(*listResources.OUs))

	filter := fmt.Sprintf("%s = %q", ou_v1.OuResourceFieldResourceId, ou1.GetResourceId())
	orderBy := ou_v1.OuResourceFieldResourceId
	job = types.NewJob(
		ctx, types.List, types.OU,
		api.GetOusParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
			Filter:   &filter,
			OrderBy:  &orderBy,
		}, nil,
	)
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)
	listResources, ok = r.Payload.Data.(api.OUsList)
	require.True(t, ok)
	assert.NotNil(t, listResources)
	assert.Equal(t, 1, len(*listResources.OUs))

	// test List error - wrong params format
	job = types.NewJob(
		ctx, types.List, types.OU,
		api.GetComputeHostsParams{}, nil,
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_OUHandler_Post(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	ctx := context.TODO()
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)
	metadata := api.Metadata{
		{
			Key:   "key",
			Value: "value",
		},
	}

	body := api.OU{
		Name:     ouName,
		Metadata: &metadata,
	}
	job := types.NewJob(
		ctx, types.Post, types.OU,
		&body, inv_handlers.OUURLParams{},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, r.Status)
	gotRes, ok := r.Payload.Data.(*api.OU)
	require.True(t, ok)

	// Validate Post changes
	job = types.NewJob(ctx, types.Get, types.OU, nil, inv_handlers.OUURLParams{
		OUID: *gotRes.OuID,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok = r.Payload.Data.(*api.OU)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, ouName, gotRes.Name)

	// Test Post Error - wrong body format
	job = types.NewJob(
		ctx, types.Post, types.OU,
		&api.Host{}, inv_handlers.OUURLParams{},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_OUHandler_Put(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}

	ouResource := inv_testing.CreateOu(t, nil)
	ctx := context.TODO()
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	metadata := api.Metadata{
		{
			Key:   "key",
			Value: "value",
		},
	}
	bodyUpdate := api.OU{
		Name:     ouName,
		Metadata: &metadata,
		ParentOu: &emptyString,
	}

	job := types.NewJob(
		ctx,
		types.Put,
		types.OU,
		&bodyUpdate,
		inv_handlers.OUURLParams{OUID: ouResource.ResourceId},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	// Validate Put changes
	job = types.NewJob(ctx, types.Get, types.OU, nil, inv_handlers.OUURLParams{
		OUID: ouResource.ResourceId,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.OU)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, ouName, gotRes.Name)

	// Test Put error - wrong body format
	job = types.NewJob(
		ctx,
		types.Put,
		types.OU,
		&api.Host{},
		inv_handlers.OUURLParams{OUID: ouResource.ResourceId},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))

	// Test Put error - wrong params format
	job = types.NewJob(
		ctx,
		types.Put,
		types.OU,
		&bodyUpdate,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_OUHandler_Patch(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	ouResource := inv_testing.CreateOu(t, nil)
	ctx := context.TODO()
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	metadata := api.Metadata{
		{
			Key:   "key",
			Value: "value",
		},
	}
	bodyUpdate := api.OU{
		Name:     ouName,
		Metadata: &metadata,
		ParentOu: &emptyString,
	}

	job := types.NewJob(
		ctx,
		types.Patch,
		types.OU,
		&bodyUpdate,
		inv_handlers.OUURLParams{OUID: ouResource.ResourceId},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	// Validate Patch changes
	job = types.NewJob(ctx, types.Get, types.OU, nil, inv_handlers.OUURLParams{
		OUID: ouResource.ResourceId,
	})
	r, err = h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.OU)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, ouName, gotRes.Name)
}

func Test_OUHandler_PatchFieldMask(t *testing.T) {
	mockClient := utils.NewMockInventoryServiceClient(
		utils.MockResponses{
			ListResourcesResponse: &inventory.ListResourcesResponse{
				Resources: []*inventory.GetResourceResponse{},
			},
			GetResourceResponse:    &inventory.GetResourceResponse{},
			UpdateResourceResponse: &inventory.Resource{},
		},
	)
	client := &clients.InventoryClientHandler{
		InvClient: mockClient,
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	metadata := api.Metadata{
		{
			Key:   "key",
			Value: "value",
		},
	}
	bodyUpdate := api.OU{
		Name:     "TestOUUpdate",
		Metadata: &metadata,
		ParentOu: &emptyString,
	}

	job := types.NewJob(
		context.TODO(),
		types.Patch,
		types.OU,
		&bodyUpdate,
		inv_handlers.OUURLParams{OUID: "ou-1234"},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.Status)

	// test Patch FieldMask
	expectedPatchFieldMask := BuildFmFromOU(bodyUpdate)
	ou := &ou_v1.OuResource{}
	expectedFieldMask, err := fieldmaskpb.New(ou, expectedPatchFieldMask...)
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

func Test_OUHandler_Get(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	ouResource := inv_testing.CreateOu(t, nil)

	ctx := context.TODO()
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	job := types.NewJob(
		ctx,
		types.Get,
		types.OU,
		nil,
		inv_handlers.OUURLParams{OUID: ouResource.ResourceId},
	)
	r, err := h.Do(job)
	require.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, r.Status)

	gotRes, ok := r.Payload.Data.(*api.OU)
	require.True(t, ok)
	assert.NotNil(t, gotRes)
	assert.Equal(t, ouResource.Name, gotRes.Name)

	// Get error - wrong params
	job = types.NewJob(
		ctx,
		types.Get,
		types.OU,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_OUHandler_Delete(t *testing.T) {
	client := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	ouResource := inv_testing.CreateOuNoCleaup(t, nil)
	ctx := context.TODO()
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	job := types.NewJob(
		ctx,
		types.Delete,
		types.OU,
		nil,
		inv_handlers.OUURLParams{OUID: ouResource.ResourceId},
	)
	r, err := h.Do(job)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.Status)

	// Validate Delete
	job = types.NewJob(
		ctx,
		types.Get,
		types.OU,
		nil,
		inv_handlers.OUURLParams{OUID: ouResource.ResourceId},
	)
	_, err = h.Do(job)
	require.Error(t, err)

	// Delete error - wrong params
	job = types.NewJob(
		ctx,
		types.Delete,
		types.OU,
		nil,
		inv_handlers.HostURLParams{},
	)
	_, err = h.Do(job)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, errors.ErrorToHTTPStatus(err))
}

func Test_Inventory_Ou_Integration(t *testing.T) {
	// verify the projection of the constants to Proto first;
	// we build a map using the field names of the proto stored in the
	// ProtoOu* slices in internal/work/handlers/ou.go. Elements must
	// have a mapping key otherwise we throw an error if there is no
	// alignment with OU proto in Inventory. Make sure to update these
	// two slices in internal/work/handlers/ou.go
	ouResource := &ou_v1.OuResource{}
	validateInventoryIntegration(
		t,
		ouResource,
		api.OU{},
		inv_handlers.OpenAPIOuToProto,
		inv_handlers.OpenAPIOUToProtoExcluded,
		maps.Values(inv_handlers.OpenAPIOuToProto),
		true,
	)
}

// Test_OUHandler_InvMockClient_Errors evaluates all
// OU handler methods with mock inventory client
// that returns errors.
func Test_OUHandler_InvMockClient_Errors(t *testing.T) {
	mockClient := utils.NewMockInventoryServiceClientError()
	client := &clients.InventoryClientHandler{
		InvClient: mockClient,
	}
	h := handlers.NewHandlers(client, nil)
	require.NotNil(t, h)

	job := types.NewJob(
		context.TODO(), types.List, types.OU,
		api.GetOusParams{
			Offset:   &pgOffset,
			PageSize: &pgSize,
			Parent:   &utils.OUUnexistID,
		}, nil,
	)
	_, err := h.Do(job)
	assert.Error(t, err)
	metadata := api.Metadata{
		{
			Key:   "key",
			Value: "value",
		},
	}
	body := api.OU{
		Name:     "Testou",
		Metadata: &metadata,
		ParentOu: &utils.OUUnexistID,
	}
	job = types.NewJob(
		context.TODO(), types.Post, types.OU,
		&body, inv_handlers.OUURLParams{},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	job = types.NewJob(
		context.TODO(),
		types.Put,
		types.OU,
		&body,
		inv_handlers.OUURLParams{OUID: "ou-1234"},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	job = types.NewJob(
		context.TODO(),
		types.Get,
		types.OU,
		nil,
		inv_handlers.OUURLParams{OUID: "ou-12345678"},
	)
	_, err = h.Do(job)
	assert.Error(t, err)

	job = types.NewJob(
		context.TODO(),
		types.Delete,
		types.OU,
		nil,
		inv_handlers.OUURLParams{OUID: "ou-1234"},
	)
	_, err = h.Do(job)
	assert.Error(t, err)
}
