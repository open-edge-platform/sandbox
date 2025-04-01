// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/ouresource"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	ou_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/ou/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/filters"
)

func Test_Metadata_Inheritance_Ou(t *testing.T) {
	// Create required OUs
	ou1 := inv_testing.CreateOuWithMeta(t, metaO1, nil)
	ou2 := inv_testing.CreateOuWithMeta(t, metaO2, ou1)
	ou3 := inv_testing.CreateOuWithMeta(t, metaO3, ou2)
	ou4 := inv_testing.CreateOu(t, ou3)
	ou5 := inv_testing.CreateOuWithMeta(t, metaO5, ou3)

	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	testcases := map[string]struct {
		in                string
		expLogiMeta       string
		expStandaloneMeta string
	}{
		"NoMetadataFromParent": {
			in:                ou4.ResourceId,
			expLogiMeta:       expLogiMeta1,
			expStandaloneMeta: "",
		},
		"InheritMetadataFromParentAndLocal": {
			in:                ou5.ResourceId,
			expLogiMeta:       expLogiMeta2,
			expStandaloneMeta: `[{"key":"key1-test", "value":"ou_key1_lvl4-test"}]`,
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			t.Run("Check with GET", func(t *testing.T) {
				getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, tc.in)
				require.NoError(t, err, "GetOu() failed")

				if getresp.RenderedMetadata == nil {
					t.Errorf("Get rendered Metadata failed")
					t.FailNow()
				}
				if !CompareMetadata(t, getresp.RenderedMetadata.LogiMetadata, tc.expLogiMeta) {
					t.Errorf("Logical Metadata data not equal - want: %s, got: %s",
						tc.expLogiMeta, getresp.RenderedMetadata.LogiMetadata,
					)
				}
				if !CompareMetadata(t, getresp.GetResource().GetOu().Metadata, tc.expStandaloneMeta) {
					t.Errorf("Standalone Metadata data not equal - want: %s, got: %s",
						tc.expStandaloneMeta, getresp.GetResource().GetOu().Metadata,
					)
				}
				if getresp.RenderedMetadata.PhyMetadata != "" {
					t.Errorf("OUs should not have Physical Metadata")
				}
			})
			t.Run("Check with LIST", func(t *testing.T) {
				listResp, err := inv_testing.TestClients[inv_testing.APIClient].
					List(ctx, &inv_v1.ResourceFilter{
						Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Ou{}},
						Filter: filters.NewBuilderWith(filters.ValEq("tenant_id", client.FakeTenantID)).
							Build(),
					})
				require.NoError(t, err)
				require.NotEmpty(t, listResp.GetResources())
				getResp := collections.Filter(listResp.GetResources(), func(v *inv_v1.GetResourceResponse) bool {
					return v.GetResource().GetOu().GetResourceId() == tc.in
				})[0]

				if getResp.RenderedMetadata == nil {
					t.Errorf("Get rendered Metadata failed")
					t.FailNow()
				}
				if !CompareMetadata(t, getResp.RenderedMetadata.LogiMetadata, tc.expLogiMeta) {
					t.Errorf("Logical Metadata data not equal - want: %s, got: %s",
						tc.expLogiMeta, getResp.RenderedMetadata.LogiMetadata,
					)
				}
				if !CompareMetadata(t, getResp.GetResource().GetOu().Metadata, tc.expStandaloneMeta) {
					t.Errorf("Standalone Metadata data not equal - want: %s, got: %s",
						tc.expStandaloneMeta, getResp.GetResource().GetOu().Metadata,
					)
				}
				if getResp.RenderedMetadata.PhyMetadata != "" {
					t.Errorf("OUs should not have Physical Metadata")
				}
			})
		})
	}
}

func Test_StrongRelations_On_Delete_Ou(t *testing.T) {
	ou1 := inv_testing.CreateOu(t, nil)
	ou2 := inv_testing.CreateOu(t, ou1)
	_ = inv_testing.CreateSite(t, nil, ou2)
	// Adding this OU to verify delete works fine for OU without children.
	// Delete is done on cleanup automatically
	_ = inv_testing.CreateOu(t, ou1)

	err := inv_testing.DeleteResourceAndReturnError(t, ou1.ResourceId)
	assertStrongRelationError(t, err, "the ou has relations with ou and cannot be deleted")

	err = inv_testing.DeleteResourceAndReturnError(t, ou2.ResourceId)
	assertStrongRelationError(t, err, "the ou has relations with site and cannot be deleted")
}

func Test_Create_Get_Delete_Update_Ou(t *testing.T) {
	parentOu := inv_testing.CreateOu(t, nil)
	inv_testing.CreateOu(t, parentOu)

	testcases := map[string]struct {
		in    *ou_v1.OuResource
		valid bool
	}{
		"CreateGoodOu": {
			in: &ou_v1.OuResource{
				Name:     "Test OU 1",
				Metadata: `[{"key":"cluster-name","value":"test"},{"key":"app-id","value":"test2-value"}]`,
			},
			valid: true,
		},
		"CreateBadOuWithMetadata": {
			in: &ou_v1.OuResource{
				Name:     "Wrong OU",
				Metadata: "INVALID JSON",
			},
			valid: false,
		},
		"CreateBadOuWithResourceIdSet": {
			// This tests case verifies that create requests with a resource ID
			// already set are rejected.
			in: &ou_v1.OuResource{
				ResourceId: "ou-12345678",
				Name:       "Test OU 2",
			},
			valid: false,
		},
		"CreateBadOuWithInvalidResourceIdSet": {
			// This tests case verifies that create requests with a invalid resource ID
			// already set are rejected.
			in: &ou_v1.OuResource{
				ResourceId: "ou-test-12345678",
				Name:       "Test OU 2",
			},
			valid: false,
		},
		"CreateGoodOuWithParentOu": {
			in: &ou_v1.OuResource{
				Name:     "Test Ou 1",
				ParentOu: parentOu,
			},
			valid: true,
		},
		"CreateBadOuWithNonExistingParentOu": {
			in: &ou_v1.OuResource{
				Name: "Test Ou 1",
				ParentOu: &ou_v1.OuResource{
					ResourceId: "ou-12345678",
				},
			},
			valid: false,
		},
		"CreateBadOuInvalidMetadata": {
			in: &ou_v1.OuResource{
				Name:     "Test OU 1",
				Metadata: metaDuplicatedKeys,
			},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Ou{Ou: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			// create
			couResp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
			ouResID := couResp.GetOu().GetResourceId()

			if err != nil {
				if tc.valid {
					t.Errorf("CreateOu() failed: %s", err)
				}
			} else {
				tc.in.ResourceId = ouResID // Update with created resource ID.
				tc.in.CreatedAt = couResp.GetOu().GetCreatedAt()
				tc.in.UpdatedAt = couResp.GetOu().GetUpdatedAt()
				assertSameResource(t, createresreq, couResp, nil)
				if !tc.valid {
					t.Errorf("CreateOu() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				// get non-existent first
				_, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, "ou-12345678")
				require.Error(t, err)

				// get
				getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, ouResID)
				require.NoError(t, err, "GetOu() failed")

				// verify data
				if eq, diff := inv_testing.ProtoEqualOrDiff(tc.in, getresp.GetResource().GetOu()); !eq {
					t.Errorf("GetOu() data not equal: %v", diff)
				}

				// update
				updateresreq := &inv_v1.Resource{
					Resource: &inv_v1.Resource_Ou{
						Ou: &ou_v1.OuResource{
							Name: "Updated Name",
						},
					},
				}

				fieldMask := &fieldmaskpb.FieldMask{Paths: []string{ouresource.FieldName}}

				upRes, err := inv_testing.TestClients[inv_testing.RMClient].Update(
					ctx,
					ouResID,
					fieldMask,
					updateresreq,
				)
				if err != nil {
					t.Errorf("UpdateOu() failed: %s", err)
				}

				// Validate returned resource
				assertSameResource(t, updateresreq, upRes, fieldMask)

				// delete non-existent first
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, "ou-12345678")
				require.Error(t, err)

				// delete
				_, err = inv_testing.TestClients[inv_testing.RMClient].Delete(
					ctx,
					ouResID,
				)
				if err != nil {
					t.Errorf("DeleteOu() failed %s", err)
				}
			}
		})
	}
}

func Test_OuDepthDetection(t *testing.T) {
	fmUpdateParentOu := &fieldmaskpb.FieldMask{Paths: []string{ouresource.EdgeParentOu}}

	t.Run("DepthDetectionOnCreate", func(t *testing.T) {
		ou1 := inv_testing.CreateOu(t, nil)
		ou2 := inv_testing.CreateOu(t, ou1)
		ou3 := inv_testing.CreateOu(t, ou2)
		ou4 := inv_testing.CreateOu(t, ou3)
		ou5 := inv_testing.CreateOu(t, ou4)
		_, err := inv_testing.CreateOuAndReturnError(t, "", ou5)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, errors.ErrorToStringWithDetails(err), "depth")
	})

	t.Run("CycleDetectionOnUpdate", func(t *testing.T) {
		ou1 := inv_testing.CreateOu(t, nil)
		ou2 := inv_testing.CreateOu(t, ou1)

		// Try to update ou1 to point to ou2.
		ou2.ParentOu = nil // Delete ref to prevent proto cycle. Not committed to the DB.
		ou1.ParentOu = ou2
		updateresreq := &inv_v1.Resource{
			Resource: &inv_v1.Resource_Ou{
				Ou: ou1,
			},
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := inv_testing.TestClients[inv_testing.APIClient].
			Update(ctx, ou1.ResourceId, fmUpdateParentOu, updateresreq)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, errors.ErrorToStringWithDetails(err), "depth")
	})

	t.Run("CycleDetectionSelfOnUpdate", func(t *testing.T) {
		ou1 := inv_testing.CreateOu(t, nil)
		ou1.ParentOu = &ou_v1.OuResource{ResourceId: ou1.ResourceId}
		updateresreq := &inv_v1.Resource{
			Resource: &inv_v1.Resource_Ou{
				Ou: ou1,
			},
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := inv_testing.TestClients[inv_testing.APIClient].
			Update(ctx, ou1.ResourceId, fmUpdateParentOu, updateresreq)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, errors.ErrorToStringWithDetails(err), "depth")
	})

	t.Run("DepthDetectionOnUpdateHead", func(t *testing.T) {
		// Create a head ou, not connected to others yet.
		ou0 := inv_testing.CreateOu(t, nil)
		// Create a linear graph of 5 ous.
		ou1 := inv_testing.CreateOu(t, nil)
		ou2 := inv_testing.CreateOu(t, ou1)
		ou3 := inv_testing.CreateOu(t, ou2)
		ou4 := inv_testing.CreateOu(t, ou3)
		inv_testing.CreateOu(t, ou4)

		// Link the ous together by updating the current head. This should fail for exceeding the depth.
		ou1.ParentOu = ou0
		updateresreq := &inv_v1.Resource{
			Resource: &inv_v1.Resource_Ou{
				Ou: ou1,
			},
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := inv_testing.TestClients[inv_testing.APIClient].
			Update(ctx, ou1.ResourceId, fmUpdateParentOu, updateresreq)
		assert.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, errors.ErrorToStringWithDetails(err), "depth")
	})

	t.Run("DepthDetectionOnMergeLinearTrees", func(t *testing.T) {
		// Create two linear ou trees, both with 3 members each.
		ou1a := inv_testing.CreateOu(t, nil)
		ou2a := inv_testing.CreateOu(t, ou1a)
		ou3a := inv_testing.CreateOu(t, ou2a)

		ou1b := inv_testing.CreateOu(t, nil)
		ou2b := inv_testing.CreateOu(t, ou1b)
		inv_testing.CreateOu(t, ou2b)

		// Link the ous together by pointing one head to the tail of the other.
		// This should fail for exceeding the depth.
		ou1b.ParentOu = ou3a
		updateresreq := &inv_v1.Resource{
			Resource: &inv_v1.Resource_Ou{
				Ou: ou1b,
			},
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := inv_testing.TestClients[inv_testing.APIClient].
			Update(ctx, ou1b.ResourceId, fmUpdateParentOu, updateresreq)
		assert.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, errors.ErrorToStringWithDetails(err), "depth")
	})
}

func Test_FilterOus(t *testing.T) {
	parentOu := inv_testing.CreateOu(t, nil)

	// Create Ous to find.
	createresreq1 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Ou{
			Ou: &ou_v1.OuResource{
				Name: "Test Ou 2",
			},
		},
	}

	createresreq2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Ou{
			Ou: &ou_v1.OuResource{
				ParentOu: parentOu,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cregResp1, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq1)
	ouResID1 := inv_testing.GetResourceIDOrFail(t, cregResp1)
	t.Cleanup(func() { inv_testing.DeleteResource(t, ouResID1) })
	cregResp2, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq2)
	ouResID2 := inv_testing.GetResourceIDOrFail(t, cregResp2)
	t.Cleanup(func() { inv_testing.DeleteResource(t, ouResID2) })

	expOu1 := cregResp1.GetOu()
	expOu1.ResourceId = ouResID1

	expOu2 := cregResp2.GetOu()
	expOu2.ResourceId = ouResID2

	testcases := map[string]struct {
		in        *inv_v1.ResourceFilter
		resources []*ou_v1.OuResource
		valid     bool
	}{
		"NoFilter": {
			in:        &inv_v1.ResourceFilter{},
			resources: []*ou_v1.OuResource{expOu1, expOu2, parentOu},
			valid:     true,
		},
		"NoFilterOrderByResourceID": {
			in: &inv_v1.ResourceFilter{
				OrderBy: ouresource.FieldResourceID,
			},
			resources: []*ou_v1.OuResource{expOu1, expOu2, parentOu},
			valid:     true,
		},
		"FilterByResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, ouresource.FieldResourceID, expOu1.ResourceId),
			},
			resources: []*ou_v1.OuResource{expOu1},
			valid:     true,
		},
		"FilterByEmptyResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = ""`, ouresource.FieldResourceID),
			},
			valid: true,
		},
		"FilterParentOu": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, ouresource.EdgeParentOu,
					ouresource.FieldResourceID, parentOu.GetResourceId()),
			},
			resources: []*ou_v1.OuResource{expOu2},
			valid:     true,
		},
		"FilterByHasParentOu": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s)`, ouresource.EdgeParentOu),
			},
			resources: []*ou_v1.OuResource{expOu2},
			valid:     true,
		},
		"FilterNoParentOu": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s)`, ouresource.EdgeParentOu),
			},
			resources: []*ou_v1.OuResource{expOu1, parentOu},
			valid:     true,
		},
		"FilterByHasParentOuHasParentOu": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s.%s)`, ouresource.EdgeParentOu, ouresource.EdgeParentOu),
			},
			resources: []*ou_v1.OuResource{},
			valid:     true,
		},
		"FilterWithOffsetLimit1": {
			in: &inv_v1.ResourceFilter{
				Offset: 5,
				Limit:  0,
			},
			valid: true,
		},
		"FilterWithOffsetLimit2": {
			in: &inv_v1.ResourceFilter{
				Offset: 0,
				Limit:  5,
			},
			resources: []*ou_v1.OuResource{expOu1, expOu2, parentOu},
			valid:     true,
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
			ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
			defer cancel()

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Ou{}} // Set the resource kind
			findres, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("FilterOus() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("FilterOus() succeeded but should have failed")
				}
			}

			// only get/delete if valid test with non-zero returned response and hasn't failed, otherwise may segfault
			if !t.Failed() && tc.valid {
				if len(findres.Resources) != len(tc.resources) {
					t.Errorf("Expected to obtain %d Resource IDs, but obtained back %d Resource IDs",
						len(tc.resources), len(findres.Resources))
				}

				resIDs := inv_testing.GetSortedResourceIDSlice(tc.resources)
				inv_testing.SortHasResourceIDAndTenantID(findres.Resources)

				if !reflect.DeepEqual(resIDs, findres.Resources) {
					t.Errorf(
						"FilterOus() failed - want: %s, got: %s",
						resIDs,
						findres.Resources,
					)
				}
			}

			listres, err := inv_testing.TestClients[inv_testing.APIClient].List(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("ListOus() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("ListOus() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				resources := make([]*ou_v1.OuResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetOu())
				}

				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					expCopy := *expected //nolint:govet // ok to copy lock in test
					ouEdgesOnlyResourceID(&expCopy)
					ouEdgesOnlyResourceID(resources[i])

					// Compare metadata separately
					assert.True(t, CompareMetadata(t, resources[i].Metadata, expCopy.Metadata))
					expCopy.Metadata = ""
					resources[i].Metadata = ""
					if eq, diff := inv_testing.ProtoEqualOrDiff(&expCopy, resources[i]); !eq {
						t.Errorf("ListOus() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

func Test_UpdateOu(t *testing.T) {
	// create Ou to update
	ou1 := inv_testing.CreateOu(t, nil)
	ou2 := inv_testing.CreateOu(t, nil)
	ou3 := inv_testing.CreateOu(t, nil)
	ou4 := inv_testing.CreateOu(t, ou1)
	ou5 := inv_testing.CreateOu(t, ou1)

	putOu := ou_v1.OuResource{
		ResourceId: ou5.ResourceId,
		Name:       "TEST",
		OuKind:     "TEST",
		ParentOu:   ou3,
		Metadata:   metaO1,
	}

	testcases := map[string]struct {
		in           *ou_v1.OuResource
		resourceID   string
		fieldMask    *fieldmaskpb.FieldMask
		valid        bool
		expErrorCode codes.Code
	}{
		"UpdatePut": {
			in:           &putOu,
			resourceID:   ou5.ResourceId,
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateName": {
			in: &ou_v1.OuResource{
				Name: "Updated Name",
			},
			resourceID: ou2.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ouresource.FieldName},
			},
			valid: true,
		},
		"UpdateResetName": {
			in:         &ou_v1.OuResource{},
			resourceID: ou2.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ouresource.FieldName},
			},
			valid: true,
		},
		"UpdateParentOu": {
			in: &ou_v1.OuResource{
				ParentOu: ou1,
			},
			resourceID: ou2.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ouresource.EdgeParentOu},
			},
			valid: true,
		},
		"UpdateResetParentOu": {
			in:         &ou_v1.OuResource{},
			resourceID: ou4.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ouresource.EdgeParentOu},
			},
			valid: true,
		},
		"UpdateNoFieldMask": {
			in: &ou_v1.OuResource{
				Name: "Updated Name 2",
			},
			resourceID:   ou2.ResourceId,
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateInvalidFieldMask1": {
			in: &ou_v1.OuResource{
				Name: "Updated Name 3",
			},
			resourceID: ou2.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"INVALID_FIELD"},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateInvalidMetadata": {
			in: &ou_v1.OuResource{
				Metadata: metaDuplicatedKeys,
			},
			resourceID: ou2.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ouresource.FieldMetadata},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateResourceIDNotFound": {
			in: &ou_v1.OuResource{
				Name: "Updated Name",
			},
			resourceID: "ou-12345678",
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ouresource.FieldName},
			},
			expErrorCode: codes.NotFound,
			valid:        false,
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			updateresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Ou{Ou: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			upRes, err := inv_testing.TestClients[inv_testing.APIClient].Update(ctx, tc.resourceID,
				tc.fieldMask, updateresreq)

			if !tc.valid {
				require.Errorf(t, err, "UpdateResource() succeeded but should have failed")
				assert.Equal(t, tc.expErrorCode, status.Code(err))
				assert.Nil(t, upRes)
				return
			}
			require.NoErrorf(t, err, "UpdateResource() failed: %s", err)

			// Validate returned resource
			assertSameResource(t, updateresreq, upRes, tc.fieldMask)

			// validate update via a get
			getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, tc.resourceID)
			require.NoError(t, err, "GetResource() failed")

			assertSameResource(t, updateresreq, getresp.GetResource(), tc.fieldMask)
		})
	}
}

func ouEdgesOnlyResourceID(expected *ou_v1.OuResource) {
	if expected.ParentOu != nil {
		expected.ParentOu = &ou_v1.OuResource{ResourceId: expected.ParentOu.ResourceId}
	}
}

func Test_NestedFilterOu(t *testing.T) {
	parentOu := inv_testing.CreateOu(t, nil)
	ou1 := inv_testing.CreateOu(t, parentOu)
	ou2 := inv_testing.CreateOu(t, ou1)
	cOu1 := *ou1 //nolint:govet // copying locks in test
	ou2.ParentOu = &cOu1
	ou1.ParentOu = parentOu

	testcases := map[string]struct {
		in                *inv_v1.ResourceFilter
		resources         []*ou_v1.OuResource
		valid             bool
		expectedCodeError codes.Code
	}{
		"FilterByParentParentOuID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s = %q`, ouresource.EdgeParentOu, ouresource.EdgeParentOu,
					ouresource.FieldResourceID, parentOu.GetResourceId()),
			},
			resources: []*ou_v1.OuResource{ou2},
			valid:     true,
		},
		"FilterByChildOuID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, ouresource.EdgeChildren,
					ouresource.FieldResourceID, ou1.GetResourceId()),
			},
			resources: []*ou_v1.OuResource{parentOu},
			valid:     true,
		},
		"FilterByHasChildren": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s)`, ouresource.EdgeChildren),
			},
			resources: []*ou_v1.OuResource{ou1, parentOu},
			valid:     true,
		},
		"FilterByNotHasChildren": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s)`, ouresource.EdgeChildren),
			},
			resources: []*ou_v1.OuResource{ou2},
			valid:     true,
		},
		"FailTooDeep": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s.%s.%s.%s = %q`, ouresource.EdgeParentOu,
					ouresource.EdgeParentOu, ouresource.EdgeParentOu, ouresource.EdgeParentOu,
					ouresource.EdgeParentOu, ouresource.FieldResourceID, parentOu.GetResourceId()),
			},
			valid:             false,
			expectedCodeError: codes.InvalidArgument,
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Ou{}} // Set the resource kind

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
						"FilterOus() failed - want: %s, got: %s",
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

				resources := make([]*ou_v1.OuResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetOu())
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

func TestOUMTSanity(t *testing.T) {
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	suite.Run(t, &struct{ mt }{
		mt: mt{
			createResource: func(tenantID string) (string, *inv_v1.Resource) {
				ou := dao.CreateOu(t, tenantID)
				res, err := util.WrapResource(ou)
				require.NoError(t, err)
				return ou.GetResourceId(), res
			},
		},
	})
}

func TestDeleteResources_OUs(t *testing.T) {
	suite.Run(t, &struct{ hardDeleteAllResourcesSuite }{
		hardDeleteAllResourcesSuite: hardDeleteAllResourcesSuite{
			createModel: func(dao *inv_testing.InvResourceDAO) (string, int) {
				tenantID := uuid.NewString()
				ou1 := dao.CreateOuNoCleanup(t, tenantID)
				ou2 := dao.CreateOuNoCleanup(t, tenantID, inv_testing.OuParent(ou1))
				ou3 := dao.CreateOuNoCleanup(t, tenantID, inv_testing.OuParent(ou2))
				ou4 := dao.CreateOuNoCleanup(t, tenantID, inv_testing.OuParent(ou3))
				ou5 := dao.CreateOuNoCleanup(t, tenantID, inv_testing.OuParent(ou2))
				return tenantID, len([]any{ou1, ou2, ou3, ou4, ou5})
			},
			resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_OU,
		},
	})
}
