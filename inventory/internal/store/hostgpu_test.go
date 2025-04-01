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

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostgpuresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/regionresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/siteresource"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

const (
	TestNameGPU    = "Test GPU"
	TestDescGPU    = "Hostgpu 1"
	TestPciGPU     = "0000:03:00.0"
	TestProductGPU = "some product name"
	TestVendorGPU  = "some vendor"
)

func Test_Create_Get_Delete_Hostgpu(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site := inv_testing.CreateSite(t, region, nil)
	provider := inv_testing.CreateProvider(t, "Test Provider1")
	host := inv_testing.CreateHost(t, site, provider)

	testcases := map[string]struct {
		in    *computev1.HostgpuResource
		valid bool
	}{
		"CreateGoodHostGPU": {
			in: &computev1.HostgpuResource{
				Description: TestDescGPU,
				Host:        host,
				DeviceName:  TestNameGPU,
				PciId:       TestPciGPU,
				Product:     TestProductGPU,
				Vendor:      TestVendorGPU,
				Features:    "a,b,c",
			},
			valid: true,
		},
		"CreateBadHostgpuWithResourceIdSet": {
			// This tests case verifies that create requests with a resource ID
			// already set are rejected.
			in: &computev1.HostgpuResource{
				ResourceId:  "hostgpu-12345678",
				Description: "Hostnic 2",
				Host:        host,
				DeviceName:  TestNameGPU,
				PciId:       TestPciGPU,
				Product:     TestProductGPU,
				Vendor:      TestVendorGPU,
			},
			valid: false,
		},
		"CreateBadHostgpuWithInvalidResourceIdSet": {
			// This tests case verifies that create requests with a resource ID
			// already set are rejected.
			in: &computev1.HostgpuResource{
				ResourceId:  "host-gpu-12345678",
				Description: "Hostnic 2",
				Host:        host,
				DeviceName:  TestNameGPU,
				PciId:       TestPciGPU,
				Product:     TestProductGPU,
				Vendor:      TestVendorGPU,
			},
			valid: false,
		},
		"CreateBadHostgpu_NoHostAssociated": {
			in:    &computev1.HostgpuResource{},
			valid: false,
		},
		"CreateBadHostgpuNonExistingHost": {
			// This tests case verifies that GPUs must point to a valid
			// existing host.
			in: &computev1.HostgpuResource{
				Host: &computev1.HostResource{
					ResourceId: "host-12345678",
				},
			},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Hostgpu{Hostgpu: tc.in},
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			resp, err := inv_testing.TestClients[inv_testing.APIClient].Create(
				ctx,
				createresreq,
			)
			var hostgpuResID string

			if err != nil {
				if tc.valid {
					t.Errorf("CreateHostgpu() failed: %s", err)
				}
			} else {
				hostgpuResID = resp.GetHostgpu().GetResourceId()
				tc.in.ResourceId = hostgpuResID // Update with created resource ID.
				tc.in.CreatedAt = resp.GetHostgpu().GetCreatedAt()
				tc.in.UpdatedAt = resp.GetHostgpu().GetUpdatedAt()
				assertSameResource(t, createresreq, resp, nil)
				if !tc.valid {
					t.Errorf("CreateHostgpu() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				// get non-existent first
				_, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, "hostgpu-12345678")
				require.Error(t, err)

				// get hostgpu
				getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, hostgpuResID)
				require.NoError(t, err, "GetHostgpu() failed")

				// verify data
				if eq, diff := inv_testing.ProtoEqualOrDiff(tc.in, getresp.GetResource().GetHostgpu()); !eq {
					t.Errorf("GetHostgpu() data not equal: %v", diff)
				}

				// delete non-existent first
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, "hostgpu-12345678")
				require.Error(t, err)

				// delete hostgpu from API
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, hostgpuResID)
				if err != nil {
					t.Errorf("DeleteHostgpu() failed: %s", err)
				}

				// get after complete Delete of hostgpu, should fail as Hostgpu is 2-phase deleted
				_, err = inv_testing.TestClients[inv_testing.RMClient].Get(ctx, hostgpuResID)
				require.Error(t, err, "Failure - Hostgpu was not deleted, but should be deleted")
			}
		})
	}
}

func Test_UpdateHostgpu(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site := inv_testing.CreateSite(t, region, nil)
	provider := inv_testing.CreateProvider(t, "Test Provider1")
	host := inv_testing.CreateHost(t, site, provider)

	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Hostgpu{
			Hostgpu: &computev1.HostgpuResource{
				DeviceName:  TestNameGPU,
				Host:        host,
				Description: TestDescGPU,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cvmResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	hostgpuResID := inv_testing.GetResourceIDOrFail(t, cvmResp)
	t.Cleanup(func() { inv_testing.DeleteResource(t, hostgpuResID) })

	testcases := map[string]struct {
		in           *computev1.HostgpuResource
		resourceID   string
		fieldMask    *fieldmaskpb.FieldMask
		valid        bool
		expErrorCode codes.Code
	}{
		"Update1": {
			in: &computev1.HostgpuResource{
				DeviceName: TestNameGPU,
				PciId:      TestPciGPU,
			},
			resourceID: hostgpuResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hostgpuresource.FieldDeviceName, hostgpuresource.FieldPciID},
			},
			valid: true,
		},
		"Update2": {
			in: &computev1.HostgpuResource{
				DeviceName: TestNameGPU,
				PciId:      TestPciGPU,
			},
			resourceID: hostgpuResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hostgpuresource.FieldDeviceName, hostgpuresource.FieldPciID, hostgpuresource.FieldDescription},
			},
			valid: true,
		},
		"Update3": {
			in: &computev1.HostgpuResource{
				DeviceName: TestNameGPU,
			},
			resourceID: hostgpuResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hostgpuresource.FieldDescription},
			},
			valid: true,
		},
		"Update4": {
			in: &computev1.HostgpuResource{
				PciId: "00:00.1",
			},
			resourceID: hostgpuResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hostgpuresource.FieldPciID},
			},
			valid: true,
		},
		"Update5": {
			in: &computev1.HostgpuResource{
				Vendor: "new vendor",
			},
			resourceID: hostgpuResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hostgpuresource.FieldVendor},
			},
			valid: true,
		},
		"Update6": {
			in: &computev1.HostgpuResource{
				Product: "new product",
			},
			resourceID: hostgpuResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hostgpuresource.FieldProduct},
			},
			valid: true,
		},
		"UpdateNoFieldMask": {
			in: &computev1.HostgpuResource{
				DeviceName: TestNameGPU,
				PciId:      TestPciGPU,
				Host:       host,
			},
			resourceID:   hostgpuResID,
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateInvalidFieldMask": {
			in: &computev1.HostgpuResource{
				DeviceName: TestNameGPU,
				PciId:      TestPciGPU,
			},
			resourceID: hostgpuResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"jklljkjlklkj"},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateFieldMaskNonClearableField": {
			in: &computev1.HostgpuResource{
				ResourceId: "proj-fb123457",
			},
			resourceID: hostgpuResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"resource"},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateResourceIDNotFound": {
			in: &computev1.HostgpuResource{
				DeviceName: TestNameGPU,
				PciId:      TestPciGPU,
			},
			resourceID: "hostgpu-12345678",
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hostgpuresource.FieldDeviceName, hostgpuresource.FieldPciID},
			},
			valid:        false,
			expErrorCode: codes.NotFound,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			updateresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Hostgpu{Hostgpu: tc.in},
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			upRes, err := inv_testing.TestClients[inv_testing.RMClient].Update(
				ctx,
				tc.resourceID,
				tc.fieldMask,
				updateresreq,
			)

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

func Test_FilterHostgpus(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site1 := inv_testing.CreateSite(t, region, nil)
	site2 := inv_testing.CreateSite(t, region, nil)
	provider1 := inv_testing.CreateProvider(t, "Test Provider1")
	provider2 := inv_testing.CreateProvider(t, "Test Provider2")
	host1 := inv_testing.CreateHost(t, site1, provider1)
	host2 := inv_testing.CreateHost(t, site2, provider2)
	host3 := inv_testing.CreateHost(t, nil, nil)

	createresreq1 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Hostgpu{
			Hostgpu: &computev1.HostgpuResource{
				Description: "Hostgpu 3",
				Host:        host1,
				DeviceName:  TestNameGPU,
			},
		},
	}

	createresreq2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Hostgpu{
			Hostgpu: &computev1.HostgpuResource{
				Description: "Hostgpu 4",
				Host:        host2,
				DeviceName:  TestNameGPU,
			},
		},
	}

	createresreqEmpty := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Hostgpu{
			Hostgpu: &computev1.HostgpuResource{
				Host: host3,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	chostgpuResp1, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq1)
	hostgpuResID1 := inv_testing.GetResourceIDOrFail(t, chostgpuResp1)
	chostgpuResp2, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq2)
	hostgpuResID2 := inv_testing.GetResourceIDOrFail(t, chostgpuResp2)
	chostgpuEmpty, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreqEmpty)
	hostgpuResIDEmpty := inv_testing.GetResourceIDOrFail(t, chostgpuEmpty)
	t.Cleanup(func() { inv_testing.DeleteResource(t, hostgpuResID1) })
	t.Cleanup(func() { inv_testing.DeleteResource(t, hostgpuResID2) })
	t.Cleanup(func() { inv_testing.DeleteResource(t, hostgpuResIDEmpty) })

	expHostgpu1 := createresreq1.GetHostgpu()
	expHostgpu1.ResourceId = hostgpuResID1

	expHostgpu2 := createresreq2.GetHostgpu()
	expHostgpu2.ResourceId = hostgpuResID2

	expHostgpuEmpty := createresreqEmpty.GetHostgpu()
	expHostgpuEmpty.ResourceId = hostgpuResIDEmpty

	testcases := map[string]struct {
		in        *inv_v1.ResourceFilter
		resources []*computev1.HostgpuResource
		valid     bool
	}{
		"NoFilter": {
			in:        &inv_v1.ResourceFilter{},
			resources: []*computev1.HostgpuResource{expHostgpu1, expHostgpu2, expHostgpuEmpty},
			valid:     true,
		},
		"NoFilterOrderByResourceID": {
			in: &inv_v1.ResourceFilter{
				OrderBy: hostgpuresource.FieldResourceID,
			},
			resources: []*computev1.HostgpuResource{expHostgpu1, expHostgpu2, expHostgpuEmpty},
			valid:     true,
		},
		"FilterByEmptyResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = ""`, hostgpuresource.FieldResourceID),
			},
			resources: []*computev1.HostgpuResource{},
			valid:     true,
		},
		"FilterByResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, hostgpuresource.FieldResourceID, expHostgpu1.ResourceId),
			},
			resources: []*computev1.HostgpuResource{expHostgpu1},
			valid:     true,
		},
		"FilterByHostID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, hostgpuresource.EdgeHost,
					hostresource.FieldResourceID, host2.GetResourceId()),
			},
			resources: []*computev1.HostgpuResource{expHostgpu2},
			valid:     true,
		},
		"FilterHostEmpty": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s)`, hostgpuresource.EdgeHost),
			},
			valid: true, // HostGpu must have a Host
		},
		"FilterByHasHost": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s)`, hostgpuresource.EdgeHost),
			},
			resources: []*computev1.HostgpuResource{expHostgpu1, expHostgpu2, expHostgpuEmpty},
			valid:     true,
		},
		"FilterByHasHostHasSite": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s.%s)`, hostgpuresource.EdgeHost, hostresource.EdgeSite),
			},
			resources: []*computev1.HostgpuResource{expHostgpu1, expHostgpu2},
			valid:     true,
		},
		"FilterDeviceName": {
			in: &inv_v1.ResourceFilter{
				Filter:   fmt.Sprintf(`%s = %q`, hostgpuresource.FieldDeviceName, TestNameGPU),
				Resource: &inv_v1.Resource{},
			},
			resources: []*computev1.HostgpuResource{expHostgpu1, expHostgpu2},
			valid:     true,
		},
		"FilterNameEmpty": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, hostgpuresource.FieldDeviceName, ""),
			},
			resources: []*computev1.HostgpuResource{expHostgpuEmpty},
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
			resources: []*computev1.HostgpuResource{expHostgpu1, expHostgpu2, expHostgpuEmpty},
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
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Hostgpu{}} // Set the resource kind
			findres, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("FilterHostgpu() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("FilterHostgpu() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				if len(findres.Resources) != len(tc.resources) {
					t.Errorf("Expected to obtain %d Resource IDs, but obtained back %d Resource IDs",
						len(tc.resources), len(findres.Resources))
				}

				resIDs := inv_testing.GetSortedResourceIDSlice(tc.resources)
				inv_testing.SortHasResourceIDAndTenantID(findres.Resources)

				if !reflect.DeepEqual(resIDs, findres.Resources) {
					t.Errorf(
						"FilterHostgpu() failed - want: %s, got: %s",
						resIDs,
						findres.Resources,
					)
				}
			}

			listres, err := inv_testing.TestClients[inv_testing.APIClient].List(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("ListHostgpu() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("ListHostgpu() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				resources := make([]*computev1.HostgpuResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetHostgpu())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					hostGpuEdgesOnlyResourceID(expected)
					hostGpuEdgesOnlyResourceID(resources[i])

					// Skip check of CreatedAt and UpdatedAt.
					resources[i].CreatedAt = expected.CreatedAt
					resources[i].UpdatedAt = expected.UpdatedAt
					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, resources[i]); !eq {
						t.Errorf("ListHostgpu() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

func hostGpuEdgesOnlyResourceID(expected *computev1.HostgpuResource) {
	if expected.Host != nil {
		expected.Host = &computev1.HostResource{ResourceId: expected.Host.ResourceId}
	}
}

func Test_NestedFilterHostgpu(t *testing.T) {
	region1 := inv_testing.CreateRegion(t, nil)
	site1 := inv_testing.CreateSite(t, region1, nil)
	host1 := inv_testing.CreateHost(t, site1, nil)
	host2 := inv_testing.CreateHost(t, nil, nil)

	hostGpu1 := inv_testing.CreatHostGPU(t, host1)
	hostGpu1.Host = host1
	hostGpu2 := inv_testing.CreatHostGPU(t, host2)
	hostGpu2.Host = host2

	testcases := map[string]struct {
		in                *inv_v1.ResourceFilter
		resources         []*computev1.HostgpuResource
		valid             bool
		expectedCodeError codes.Code
	}{
		"FilterBySiteID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s = %q`, hostgpuresource.EdgeHost, hostresource.EdgeSite,
					siteresource.FieldResourceID, site1.GetResourceId()),
			},
			resources: []*computev1.HostgpuResource{hostGpu1},
			valid:     true,
		},
		"FilterBySiteEmpty": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s.%s)`, hostgpuresource.EdgeHost, hostresource.EdgeSite),
			},
			resources: []*computev1.HostgpuResource{hostGpu2},
			valid:     true,
		},
		"FilterByHasSite": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s.%s)`, hostgpuresource.EdgeHost, hostresource.EdgeSite),
			},
			resources: []*computev1.HostgpuResource{hostGpu1},
			valid:     true,
		},
		"FilterByRegionID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s.%s = %q`, hostgpuresource.EdgeHost, hostresource.EdgeSite,
					siteresource.EdgeRegion, regionresource.FieldResourceID, region1.GetResourceId()),
			},
			resources: []*computev1.HostgpuResource{hostGpu1},
			valid:     true,
		},
		"FailTooDeep": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s.%s.%s.%s = %q`, hostgpuresource.EdgeHost, hostresource.EdgeSite,
					siteresource.EdgeRegion, regionresource.EdgeParentRegion, regionresource.EdgeParentRegion,
					regionresource.FieldResourceID, region1.GetResourceId()),
			},
			valid:             false,
			expectedCodeError: codes.InvalidArgument,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Hostgpu{}} // Set the resource kind

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
						"FilterGpus() failed - want: %s, got: %s",
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

				resources := make([]*computev1.HostgpuResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetHostgpu())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, resources[i]); !eq {
						t.Errorf("ListGpus() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

func Test_StrongRelations_On_Delete_HostGpu(t *testing.T) {
	host := inv_testing.CreateHost(t, nil, nil)
	inv_testing.CreatHostGPU(t, host)

	err := inv_testing.HardDeleteHostAndReturnError(t, host.ResourceId)
	assertStrongRelationError(t, err, "violates foreign key constraint")
}

func TestHostGPUMTSanity(t *testing.T) {
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	suite.Run(t, &struct{ mt }{
		mt: mt{
			createResource: func(tenantID string) (string, *inv_v1.Resource) {
				host := dao.CreateHost(t, tenantID)
				hostGPU := dao.CreateHostGPU(t, tenantID, host)
				res, err := util.WrapResource(hostGPU)
				require.NoError(t, err)
				return hostGPU.GetResourceId(), res
			},
		},
	})
}

func TestDeleteResources_HostGPUs(t *testing.T) {
	suite.Run(t, &struct{ hardDeleteAllResourcesSuite }{
		hardDeleteAllResourcesSuite: hardDeleteAllResourcesSuite{
			createModel: func(dao *inv_testing.InvResourceDAO) (string, int) {
				tenantID := uuid.NewString()
				host := dao.CreateHost(t, tenantID)
				return tenantID, len(
					[]any{
						dao.CreateHostGPUNoCleanup(t, tenantID, host),
						dao.CreateHostGPUNoCleanup(t, tenantID, host),
						dao.CreateHostGPUNoCleanup(t, tenantID, host),
					})
			},
			resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_HOSTGPU,
		},
	})
}
