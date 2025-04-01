// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/workloadmember"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/workloadresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/store"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/filters"
)

func Test_StrongRelations_On_Delete_Workload(t *testing.T) {
	os := inv_testing.CreateOs(t)
	instance := inv_testing.CreateInstance(t, nil, os)

	t.Run("ClusterWorkload", func(t *testing.T) {
		workload1 := inv_testing.CreateWorkload(t)
		inv_testing.CreateWorkloadMember(t, workload1, instance)

		err := inv_testing.DeleteResourceAndReturnError(t, workload1.ResourceId)
		assertStrongRelationError(t, err, "the workload has relations and cannot be deleted")
	})

	t.Run("ReconciliableWorkload", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		workload2 := &computev1.WorkloadResource{
			Kind:         computev1.WorkloadKind_WORKLOAD_KIND_DHCP,
			Name:         "Test workload 1",
			DesiredState: computev1.WorkloadState_WORKLOAD_STATE_PROVISIONED,
		}

		resp, err := inv_testing.GetClient(t, inv_testing.APIClient).Create(ctx,
			&inv_v1.Resource{
				Resource: &inv_v1.Resource_Workload{Workload: workload2},
			})
		require.NoError(t, err)
		workloadResID := inv_testing.GetResourceIDOrFail(t, resp)
		t.Cleanup(func() {
			inv_testing.HardDeleteWorkload(t, workloadResID, computev1.WorkloadKind_WORKLOAD_KIND_DHCP)
		})
		workload2.ResourceId = workloadResID
		inv_testing.CreateWorkloadMember(t, workload2, instance)

		err = inv_testing.HardDeleteWorkloadAndReturnError(t, workload2.ResourceId, computev1.WorkloadKind_WORKLOAD_KIND_DHCP)
		assertStrongRelationError(t, err, "the workload has relations and cannot be deleted")
	})
}

func Test_Workload_BackReferences_Read(t *testing.T) {
	os := inv_testing.CreateOs(t)
	instance := inv_testing.CreateInstance(t, nil, os)
	workload := inv_testing.CreateWorkload(t)
	workloadMember := inv_testing.CreateWorkloadMember(t, workload, instance)
	// Prepare the expected workload with required nested resources
	workloadMember.Instance = instance
	workload.Members = append(workload.Members, workloadMember)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Fetch the workload and check that all back-references are present.
	resp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, workload.ResourceId)
	require.NoError(t, err)
	eq, diff := inv_testing.ProtoEqualOrDiff(workload, resp.GetResource().GetWorkload())
	require.Truef(t, eq, "workload back-ref not equal %v", diff)
}

func Test_Create_Get_Delete_Update_Workload(t *testing.T) {
	testcases := map[string]struct {
		in    *computev1.WorkloadResource
		valid bool
	}{
		"CreateGoodWorkloadCluster": {
			in: &computev1.WorkloadResource{
				Kind:         computev1.WorkloadKind_WORKLOAD_KIND_CLUSTER,
				Name:         "Test workload 1",
				DesiredState: computev1.WorkloadState_WORKLOAD_STATE_PROVISIONED,
				Status:       "provisioning",
				Metadata:     metaO1, // Random metadata, we don't care about the content
				ExternalId:   uuid.NewString(),
			},
			valid: true,
		},
		"CreateGoodWorkloadDHCP": {
			in: &computev1.WorkloadResource{
				Kind:         computev1.WorkloadKind_WORKLOAD_KIND_DHCP,
				Name:         "Test workload 1",
				DesiredState: computev1.WorkloadState_WORKLOAD_STATE_PROVISIONED,
				Status:       "provisioning",
				Metadata:     metaO1, // Random metadata, we don't care about the content
			},
			valid: true,
		},
		"CreateBadWorkloadWithResourceIdSet": {
			// This tests case verifies that create requests with a resource ID
			// already set are rejected.
			in: &computev1.WorkloadResource{
				ResourceId:   "workload-12345678",
				Kind:         computev1.WorkloadKind_WORKLOAD_KIND_CLUSTER,
				Name:         "Test workload 2",
				DesiredState: computev1.WorkloadState_WORKLOAD_STATE_PROVISIONED,
				Metadata:     metaO1, // Random metadata, we don't care about the content
			},
			valid: false,
		},
		"CreateBadWorkloadWithDesiredState": {
			// This tests case verifies that a workload cannot be created with part of its required fields
			in: &computev1.WorkloadResource{
				DesiredState: computev1.WorkloadState_WORKLOAD_STATE_PROVISIONED,
			},
			valid: false,
		},
		"CreateBadWorkloadWithWrongMeta": {
			in: &computev1.WorkloadResource{
				Kind:         computev1.WorkloadKind_WORKLOAD_KIND_CLUSTER,
				Name:         "Test workload 1",
				DesiredState: computev1.WorkloadState_WORKLOAD_STATE_PROVISIONED,
				Metadata:     "WRONG METADATA",
			},
			valid: false,
		},
		"CreateBadWorkloadWithTooLongExternalID": {
			in: &computev1.WorkloadResource{
				Kind:         computev1.WorkloadKind_WORKLOAD_KIND_CLUSTER,
				Name:         "Test workload 1",
				DesiredState: computev1.WorkloadState_WORKLOAD_STATE_PROVISIONED,
				ExternalId:   strings.Repeat("A", 41),
			},
			valid: false,
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Workload{Workload: tc.in},
			}
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			// create
			cworkloadResp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
			workloadResID := cworkloadResp.GetWorkload().GetResourceId()

			if err != nil {
				if tc.valid {
					t.Errorf("CreateWorkload() failed: %s", err)
				}
			} else {
				tc.in.ResourceId = workloadResID // Update with created resource ID.
				tc.in.CreatedAt = cworkloadResp.GetWorkload().GetCreatedAt()
				tc.in.UpdatedAt = cworkloadResp.GetWorkload().GetUpdatedAt()
				assertSameResource(t, createresreq, cworkloadResp, nil)
				if !tc.valid {
					t.Errorf("CreateWorkload() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				// get non-existent first
				_, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, "workload-12345678")
				require.Error(t, err)

				// get
				getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, workloadResID)
				if err != nil {
					require.NoError(t, err, "GetWorkload() failed")
				}

				// verify data
				if eq, diff := inv_testing.ProtoEqualOrDiff(tc.in, getresp.GetResource().GetWorkload()); !eq {
					t.Errorf("GetWorkload() data not equal: %v", diff)
				}

				// update
				updateresreq := &inv_v1.Resource{
					Resource: &inv_v1.Resource_Workload{
						Workload: &computev1.WorkloadResource{
							Name: "Updated Name",
						},
					},
				}

				_, err = inv_testing.TestClients[inv_testing.APIClient].Update(ctx, workloadResID,
					&fieldmaskpb.FieldMask{Paths: []string{workloadresource.FieldName}}, updateresreq)
				if err != nil {
					t.Errorf("UpdateWorkload() failed: %s", err)
				}

				// delete non-existent first
				err = inv_testing.DeleteResourceAndReturnError(t, "workload-12345678")
				require.Error(t, err)

				// delete
				inv_testing.HardDeleteWorkload(t, workloadResID, tc.in.Kind)

				// get after complete Delete of workload
				_, err = inv_testing.TestClients[inv_testing.APIClient].Get(ctx, workloadResID)
				if err == nil {
					t.Errorf("Failure - Workload was not deleted, but should be deleted")
				}
			}
		})
	}
}

func Test_Unique_Workload_external_ID(t *testing.T) {
	workload1 := inv_testing.CreateWorkload(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Workload with the same UUID
	workload2 := &computev1.WorkloadResource{
		Kind:         computev1.WorkloadKind_WORKLOAD_KIND_CLUSTER,
		DesiredState: computev1.WorkloadState_WORKLOAD_STATE_PROVISIONED,
		ExternalId:   workload1.ExternalId,
	}
	resp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx,
		&inv_v1.Resource{
			Resource: &inv_v1.Resource_Workload{Workload: workload2},
		})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))

	// Workload with empty UUID
	workload3 := &computev1.WorkloadResource{
		Kind:         computev1.WorkloadKind_WORKLOAD_KIND_CLUSTER,
		DesiredState: computev1.WorkloadState_WORKLOAD_STATE_PROVISIONED,
	}
	resp, err = inv_testing.TestClients[inv_testing.APIClient].Create(ctx,
		&inv_v1.Resource{
			Resource: &inv_v1.Resource_Workload{Workload: workload3},
		})
	require.NoError(t, err)
	require.NotNil(t, resp)
	respID := inv_testing.GetResourceIDOrFail(t, resp)
	t.Cleanup(func() { inv_testing.HardDeleteWorkload(t, respID, workload3.Kind) })

	// create workload for another tenant with external_id occupied but workload1
	anotherTenantWorkload := &computev1.WorkloadResource{
		Kind:         computev1.WorkloadKind_WORKLOAD_KIND_CLUSTER,
		DesiredState: computev1.WorkloadState_WORKLOAD_STATE_PROVISIONED,
		ExternalId:   workload1.ExternalId,
		TenantId:     tenantIDOne,
	}
	rsp, err := inv_testing.GetClient(t, inv_testing.APIClient).GetTenantAwareInventoryClient().
		Create(ctx, tenantIDOne, &inv_v1.Resource{
			Resource: &inv_v1.Resource_Workload{Workload: anotherTenantWorkload},
		})

	require.NoError(
		t, err, "multiple workloads with same external_id shall be allowed if workloads are related with different tenants")
	require.NotNil(t, rsp)
	rspID := inv_testing.GetResourceIDOrFail(t, rsp)

	t.Cleanup(
		func() {
			localCtx, localCancel := context.WithTimeout(context.Background(), time.Second)
			defer localCancel()
			_, deletionErr := inv_testing.GetClient(t, inv_testing.APIClient).
				GetTenantAwareInventoryClient().
				Delete(localCtx, tenantIDOne, rspID)

			if deletionErr != nil {
				require.NoError(t, err)
			}
		})
}

func Test_FilterWorkload(t *testing.T) {
	workload1 := inv_testing.CreateWorkload(t)
	workload2 := inv_testing.CreateWorkload(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// Workload without UUID
	workload3 := &computev1.WorkloadResource{
		Kind:         computev1.WorkloadKind_WORKLOAD_KIND_CLUSTER,
		Name:         "for unit testing purposes",
		DesiredState: computev1.WorkloadState_WORKLOAD_STATE_PROVISIONED,
		Status:       "provisioned",
		Metadata:     metaR1,
	}
	resp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx,
		&inv_v1.Resource{
			Resource: &inv_v1.Resource_Workload{Workload: workload3},
		})
	require.NoError(t, err)
	workload3 = resp.GetWorkload()
	t.Cleanup(func() { inv_testing.HardDeleteWorkload(t, workload3.ResourceId, workload3.Kind) })

	testcases := map[string]struct {
		in        *inv_v1.ResourceFilter
		resources []*computev1.WorkloadResource
		valid     bool
	}{
		"NoFilter": {
			in:        &inv_v1.ResourceFilter{},
			resources: []*computev1.WorkloadResource{workload1, workload2, workload3},
			valid:     true,
		},
		"NoFilterOrderByResourceID": {
			in: &inv_v1.ResourceFilter{
				OrderBy: workloadresource.FieldResourceID,
			},
			resources: []*computev1.WorkloadResource{workload1, workload2, workload3},
			valid:     true,
		},
		"FilterByResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, workloadresource.FieldResourceID, workload1.ResourceId),
			},
			resources: []*computev1.WorkloadResource{workload1},
			valid:     true,
		},
		"FilterExternalID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, workloadresource.FieldExternalID, workload1.ExternalId),
			},
			resources: []*computev1.WorkloadResource{workload1},
			valid:     true,
		},
		"FilterNoExternalID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, workloadresource.FieldExternalID, ""),
			},
			resources: []*computev1.WorkloadResource{workload3},
			valid:     true,
		},
		"FilterByEmptyResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = ""`, workloadresource.FieldResourceID),
			},
			valid: true,
		},
		"FilterByHasMembers": {
			in: &inv_v1.ResourceFilter{
				Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Workload{}},
				Filter:   fmt.Sprintf(`has(%s)`, workloadresource.EdgeMembers),
			},
			resources: []*computev1.WorkloadResource{},
			valid:     true,
		},
		"FilterLimit": {
			in: &inv_v1.ResourceFilter{
				Offset: 0,
				Limit:  3,
			},
			resources: []*computev1.WorkloadResource{workload1, workload2, workload3},
			valid:     true,
		},
		"FilterWithOffsetLimit": {
			in: &inv_v1.ResourceFilter{
				Offset: 5,
				Limit:  0,
			},
			valid: true,
		},
		"FilterInvalidEdge": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s)`, "invalid_edge"),
			},
			valid: false,
		},
		"FilterInvalidField": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, "invalid_field", "some-value"),
			},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Workload{}} // Set the resource kind
			findres, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("FilterWorkload() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("FilterWorkload() succeeded but should have failed")
				}
			}

			// only check if valid test with non-zero returned response and hasn't failed, otherwise may segfault
			if !t.Failed() && tc.valid {
				if len(findres.Resources) != len(tc.resources) {
					t.Errorf("Expected to obtain %d Resource IDs, but obtained back %d Resource IDs",
						len(tc.resources), len(findres.Resources))
				}

				resIDs := inv_testing.GetSortedResourceIDSlice(tc.resources)
				inv_testing.SortHasResourceIDAndTenantID(findres.Resources)

				if !reflect.DeepEqual(resIDs, findres.Resources) {
					t.Errorf(
						"FilterWorkload() failed - want: %s, got: %s",
						resIDs,
						findres.Resources,
					)
				}
			}

			listres, err := inv_testing.TestClients[inv_testing.APIClient].List(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("ListWorkload() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("ListWorkload() succeeded but should have failed")
				}
			}

			// only check if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				resources := make([]*computev1.WorkloadResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetWorkload())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, resources[i]); !eq {
						t.Errorf("ListWorkload() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

func Test_UpdateWorkload(t *testing.T) {
	// create workload to update
	workload1 := inv_testing.CreateWorkload(t)

	putAllFieldsWorkload := computev1.WorkloadResource{
		ResourceId:   workload1.ResourceId,
		Kind:         computev1.WorkloadKind_WORKLOAD_KIND_CLUSTER,
		Name:         "TEST",
		DesiredState: computev1.WorkloadState_WORKLOAD_STATE_DELETED,
		Status:       "unspecified",
		Metadata:     metaO3,
		ExternalId:   uuid.NewString(),
	}
	testcases := map[string]struct {
		in           *computev1.WorkloadResource
		resourceID   string
		fieldMask    *fieldmaskpb.FieldMask
		valid        bool
		expErrorCode codes.Code
	}{
		"UpdatePut": {
			in:           &putAllFieldsWorkload,
			resourceID:   workload1.ResourceId,
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateName": {
			in: &computev1.WorkloadResource{
				Name: "Updated Name",
			},
			resourceID: workload1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{workloadresource.FieldName},
			},
			valid: true,
		},
		"UpdateExternalID": {
			in: &computev1.WorkloadResource{
				ExternalId: uuid.NewString(),
			},
			resourceID: workload1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{workloadresource.FieldExternalID},
			},
			valid: true,
		},
		"UpdateMultipleFields": {
			in: &computev1.WorkloadResource{
				Name:         "Updated Name 2",
				DesiredState: computev1.WorkloadState_WORKLOAD_STATE_ERROR,
			},
			resourceID: workload1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{workloadresource.FieldName, workloadresource.FieldDesiredState},
			},
			valid: true,
		},
		"UpdateNoFieldMask": {
			in: &computev1.WorkloadResource{
				Kind:         computev1.WorkloadKind_WORKLOAD_KIND_CLUSTER,
				DesiredState: computev1.WorkloadState_WORKLOAD_STATE_DELETING,
				Status:       "error",
				ExternalId:   uuid.NewString(),
			},
			resourceID:   workload1.ResourceId,
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateInvalidFieldMask1": {
			in: &computev1.WorkloadResource{
				Name: "Updated Name 5",
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"INVALID_FIELD"},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateResourceIDNotFound": {
			in: &computev1.WorkloadResource{
				Name: "Updated Name",
			},
			resourceID: "workload-12345678",
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{workloadresource.FieldName},
			},
			valid:        false,
			expErrorCode: codes.NotFound,
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			updateresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Workload{Workload: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			upRes, err := inv_testing.TestClients[inv_testing.APIClient].Update(
				ctx, tc.resourceID, tc.fieldMask, updateresreq)

			if !tc.valid {
				require.Errorf(t, err, "UpdateResource() succeeded but should have failed")
				assert.Equal(t, tc.expErrorCode, status.Code(err))
				assert.Nil(t, upRes)
				return
			}
			require.NoErrorf(t, err, "UpdateResource() failed: %s", err)

			// Validate returned resource
			assertSameResource(t, updateresreq, upRes, tc.fieldMask)

			if err != nil {
				if tc.valid {
					t.Errorf("UpdateResource() failed: %s", err)
				}
				// Useless to check if the value was updated if the Update was unsuccessful
				return
			} else if !tc.valid {
				t.Errorf("UpdateResource() succeeded but should have failed")
				return
			}

			// validate update via a get
			getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, tc.resourceID)
			require.NoError(t, err, "GetWorkload() failed")
			assertSameResource(t, updateresreq, getresp.GetResource(), tc.fieldMask)
		})
	}
}

// Test_NestedEagerLoading verifies that the eager loading of the host associated to the member is provided when
// querying workloads (via Get or List).
func Test_NestedEagerLoading(t *testing.T) {
	os := inv_testing.CreateOs(t)
	workload := inv_testing.CreateWorkload(t)
	instance := inv_testing.CreateInstance(t, nil, os)
	inv_testing.CreateWorkloadMember(t, workload, instance)

	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	getRes, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, workload.ResourceId)
	require.NoError(t, err)

	getW := getRes.GetResource().GetWorkload()
	if eq, diff := inv_testing.ProtoEqualOrDiff(instance, getW.Members[0].Instance); !eq {
		t.Errorf("Nested host eager loaded not equal via get: %v", diff)
	}

	listRes, err := inv_testing.TestClients[inv_testing.APIClient].List(ctx, &inv_v1.ResourceFilter{
		Resource: &inv_v1.Resource{
			Resource: &inv_v1.Resource_Workload{},
		},
	})
	require.NoError(t, err)

	listW := listRes.Resources[0].GetResource().GetWorkload()
	if eq, diff := inv_testing.ProtoEqualOrDiff(instance, listW.Members[0].Instance); !eq {
		t.Errorf("Nested host eager loaded not equal via list: %v", diff)
	}
}

func Test_Events_DeleteWorkload(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workloadCluster := &computev1.WorkloadResource{
		Kind:         computev1.WorkloadKind_WORKLOAD_KIND_CLUSTER,
		Name:         "testing",
		DesiredState: computev1.WorkloadState_WORKLOAD_STATE_PROVISIONED,
	}
	respCluster, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx,
		&inv_v1.Resource{
			Resource: &inv_v1.Resource_Workload{Workload: workloadCluster},
		})
	require.NoError(t, err)
	wClusterResID := inv_testing.GetResourceIDOrFail(t, respCluster)

	workloadDhcp := &computev1.WorkloadResource{
		Kind:         computev1.WorkloadKind_WORKLOAD_KIND_DHCP,
		Name:         "testing",
		DesiredState: computev1.WorkloadState_WORKLOAD_STATE_PROVISIONED,
	}
	respDhcp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx,
		&inv_v1.Resource{
			Resource: &inv_v1.Resource_Workload{Workload: workloadDhcp},
		})
	require.NoError(t, err)
	wDhcpResID := inv_testing.GetResourceIDOrFail(t, respDhcp)

	// Empty the channel
	time.Sleep(1 * time.Second)
	for len(inv_testing.TestClientsEvents[inv_testing.RMClient]) > 0 {
		<-inv_testing.TestClientsEvents[inv_testing.RMClient]
	}
	for len(inv_testing.TestClientsEvents[inv_testing.APIClient]) > 0 {
		<-inv_testing.TestClientsEvents[inv_testing.APIClient]
	}

	_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, wClusterResID)
	require.NoError(t, err)
	assertReceiveEvent(
		t,
		inv_testing.TestClientsEvents[inv_testing.RMClient],
		inv_v1.SubscribeEventsResponse_EVENT_KIND_DELETED,
		inv_v1.ResourceKind_RESOURCE_KIND_WORKLOAD,
		wClusterResID,
	)

	_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, wDhcpResID)
	require.NoError(t, err)
	assertReceiveEvent(
		t,
		inv_testing.TestClientsEvents[inv_testing.RMClient],
		inv_v1.SubscribeEventsResponse_EVENT_KIND_UPDATED,
		inv_v1.ResourceKind_RESOURCE_KIND_WORKLOAD,
		wDhcpResID,
	)

	_, err = inv_testing.TestClients[inv_testing.RMClient].Update(ctx,
		wDhcpResID,
		&fieldmaskpb.FieldMask{
			Paths: []string{workloadresource.FieldCurrentState},
		},
		&inv_v1.Resource{
			Resource: &inv_v1.Resource_Workload{
				Workload: &computev1.WorkloadResource{CurrentState: computev1.WorkloadState_WORKLOAD_STATE_DELETED},
			},
		},
	)
	require.NoError(t, err)
	assertReceiveEvent(
		t,
		inv_testing.TestClientsEvents[inv_testing.APIClient],
		inv_v1.SubscribeEventsResponse_EVENT_KIND_DELETED,
		inv_v1.ResourceKind_RESOURCE_KIND_WORKLOAD,
		wDhcpResID,
	)
}

func Test_FilterNestedWorkloadResource(t *testing.T) {
	os := inv_testing.CreateOs(t)
	workload1 := inv_testing.CreateWorkload(t)
	host1 := inv_testing.CreateHost(t, nil, nil)
	instance1 := inv_testing.CreateInstance(t, host1, os)
	member1 := inv_testing.CreateWorkloadMember(t, workload1, instance1)
	member1.Instance = instance1
	workload1.Members = append(workload1.Members, member1)

	workload2 := inv_testing.CreateWorkload(t)
	instance2 := inv_testing.CreateInstance(t, nil, os)
	member2 := inv_testing.CreateWorkloadMember(t, workload2, instance2)
	member2.Instance = instance2
	workload2.Members = append(workload2.Members, member2)

	workload3 := inv_testing.CreateWorkload(t)

	testcases := map[string]struct {
		in                *inv_v1.ResourceFilter
		resources         []*computev1.WorkloadResource
		valid             bool
		expectedCodeError codes.Code
	}{
		"FilterByWorkloadMemberID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, workloadresource.EdgeMembers,
					workloadmember.FieldResourceID, member1.GetResourceId()),
			},
			resources: []*computev1.WorkloadResource{workload1},
			valid:     true,
		},
		"FilterByHasWorkloadMembers": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s)`, workloadresource.EdgeMembers),
			},
			resources: []*computev1.WorkloadResource{workload1, workload2},
			valid:     true,
		},
		"FilterByNotHasWorkloadMembers": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s)`, workloadresource.EdgeMembers),
			},
			resources: []*computev1.WorkloadResource{workload3},
			valid:     true,
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Workload{}} // Set the resource kind

			// Test FIND
			findres, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx, tc.in)
			if !tc.valid {
				require.Error(t, err)
				assert.Equal(t, tc.expectedCodeError, status.Code(err))
			} else {
				require.NoError(t, err)

				resIDs := inv_testing.GetSortedResourceIDSlice(tc.resources)
				inv_testing.SortHasResourceIDAndTenantID(findres.Resources)

				if !reflect.DeepEqual(resIDs, findres.Resources) {
					t.Errorf(
						"FilterInstances() failed - want: %s, got: %s",
						resIDs,
						findres.Resources,
					)
				}
			}

			// Test LIST
			listres, err := inv_testing.TestClients[inv_testing.APIClient].List(ctx, tc.in)
			if !tc.valid {
				require.Error(t, err)
				assert.Equal(t, tc.expectedCodeError, status.Code(err))
			} else {
				require.NoError(t, err)

				resources := make([]*computev1.WorkloadResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetWorkload())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, resources[i]); !eq {
						t.Errorf("ListInstances() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

func Test_WorkloadEnumStatusMap(t *testing.T) {
	v, err := store.WorkloadEnumStatusMap(workloadresource.FieldCurrentState,
		int32(computev1.WorkloadState_WORKLOAD_STATE_ERROR))
	val0, ok0 := v.(workloadresource.CurrentState)
	assert.Nil(t, err)
	assert.True(t, ok0)
	assert.Equal(t, "WORKLOAD_STATE_ERROR", val0.String())

	v, err = store.WorkloadEnumStatusMap("invalid_input",
		int32(computev1.WorkloadState_WORKLOAD_STATE_ERROR))
	assert.Error(t, err)
	assert.Nil(t, v)
}

func TestWorkloadMTSanity(t *testing.T) {
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	suite.Run(t, &struct{ mt }{
		mt: mt{
			createResource: func(tenantID string) (string, *inv_v1.Resource) {
				workload := dao.CreateWorkload(t, tenantID)
				res, err := util.WrapResource(workload)
				require.NoError(t, err)
				return workload.GetResourceId(), res
			},
		},
	})
}

func TestSoftDeleteResources_Workloads(t *testing.T) {
	suite.Run(t, &softDeleteAllResourcesSuite{
		createModel: func(dao *inv_testing.InvResourceDAO) (string, int) {
			tenantID := uuid.NewString()
			return tenantID, len([]any{
				dao.CreateWorkloadNoCleanup(t, tenantID, inv_testing.WorkloadKind(computev1.WorkloadKind_WORKLOAD_KIND_DHCP)),
				dao.CreateWorkloadNoCleanup(t, tenantID, inv_testing.WorkloadKind(computev1.WorkloadKind_WORKLOAD_KIND_DHCP)),
			})
		},
		resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_WORKLOAD,
		deletedClause: filters.ValEq(
			computev1.WorkloadResourceFieldDesiredState, computev1.WorkloadState_WORKLOAD_STATE_DELETED),
		notDeletedClause: filters.ValNotEq(
			computev1.WorkloadResourceFieldDesiredState, computev1.WorkloadState_WORKLOAD_STATE_DELETED),
	})
}

func TestHardDeleteResources_Workloads(t *testing.T) {
	suite.Run(t, &hardDeleteAllResourcesSuite{
		createModel: func(dao *inv_testing.InvResourceDAO) (string, int) {
			tenantID := uuid.NewString()
			return tenantID, len(
				[]any{
					dao.CreateWorkloadNoCleanup(t, tenantID,
						inv_testing.WorkloadKind(computev1.WorkloadKind_WORKLOAD_KIND_CLUSTER)),
					dao.CreateWorkloadNoCleanup(t, tenantID,
						inv_testing.WorkloadKind(computev1.WorkloadKind_WORKLOAD_KIND_CLUSTER)),
					dao.CreateWorkloadNoCleanup(t, tenantID,
						inv_testing.WorkloadKind(computev1.WorkloadKind_WORKLOAD_KIND_DHCP)),
				},
			)
		},
		resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_WORKLOAD,
	})
}
