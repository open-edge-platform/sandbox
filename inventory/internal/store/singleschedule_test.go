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

	hosts "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/regionresource"
	ssr "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/singlescheduleresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/siteresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/store"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	locationv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/location/v1"
	schedule_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/schedule/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

// times for testing.
var (
	now   = uint64(time.Now().Unix())
	nowP1 = now - 3600   // one hour in past
	nowF1 = now + 3600   // one hour in future
	nowF2 = now + 7200   // two hours in future
	nowD1 = now + 86400  // one day in future
	nowD2 = now + 172800 // two days in future
)

func Test_Create_Get_Delete_Update_SingleSchedule(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site := inv_testing.CreateSite(t, nil, nil)
	host := inv_testing.CreateHost(t, site, nil)
	workload := inv_testing.CreateWorkload(t)

	ghost := &computev1.HostResource{
		ResourceId: "host-12345678",
	}
	gsite := &locationv1.SiteResource{
		ResourceId: "site-12345678",
	}
	gworkload := &computev1.WorkloadResource{
		ResourceId: "workload-12345678",
	}
	gregion := &locationv1.RegionResource{
		ResourceId: "region-12345678",
	}

	testcases := map[string]struct {
		in    *schedule_v1.SingleScheduleResource
		valid bool
	}{
		"CreateGoodSingleSchedule1": {
			in: &schedule_v1.SingleScheduleResource{
				Name:           "Test SingleSchedule 1",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				StartSeconds:   nowF1,
				EndSeconds:     nowF2,
			},
			valid: true,
		},
		"CreateGoodSingleSchedule2": {
			in: &schedule_v1.SingleScheduleResource{
				Name: "Test SingleSchedule 2",
				Relation: &schedule_v1.SingleScheduleResource_TargetHost{
					TargetHost: host,
				},
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				StartSeconds:   nowF1,
				EndSeconds:     nowF2,
			},
			valid: true,
		},
		"CreateGoodSingleSchedule3": {
			in: &schedule_v1.SingleScheduleResource{
				Name: "Test SingleSchedule 3",
				Relation: &schedule_v1.SingleScheduleResource_TargetSite{
					TargetSite: site,
				},
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				StartSeconds:   nowF1,
				EndSeconds:     nowF2,
			},
			valid: true,
		},
		"CreateGoodSingleSchedule4": {
			in: &schedule_v1.SingleScheduleResource{
				Name: "Test SingleSchedule 4",
				Relation: &schedule_v1.SingleScheduleResource_TargetWorkload{
					TargetWorkload: workload,
				},
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				StartSeconds:   nowF1,
				EndSeconds:     nowF2,
			},
			valid: true,
		},
		"CreateGoodSingleSchedule5": {
			in: &schedule_v1.SingleScheduleResource{
				Name: "Test SingleSchedule 5",
				Relation: &schedule_v1.SingleScheduleResource_TargetRegion{
					TargetRegion: region,
				},
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				StartSeconds:   nowF1,
				EndSeconds:     nowF2,
			},
			valid: true,
		},
		"CreateBadSingleScheduleWithResourceIdSet": {
			// This tests case verifies that create requests with a resource ID
			// already set are rejected.
			in: &schedule_v1.SingleScheduleResource{
				ResourceId:     "singlesche-12345678",
				Name:           "Test SingleSchedule 4",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_OS_UPDATE,
				StartSeconds:   nowF1,
				EndSeconds:     nowF2,
			},
			valid: false,
		},
		"CreateBadSingleScheduleWithInvalidResourceIdSet": {
			// This tests case verifies that create requests with a invalid resource ID
			// already set are rejected.
			in: &schedule_v1.SingleScheduleResource{
				ResourceId:     "single-sche-12345678",
				Name:           "Test SingleSchedule 4",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_OS_UPDATE,
				StartSeconds:   nowF1,
				EndSeconds:     nowF2,
			},
			valid: false,
		},
		"CreateBadTarget1": {
			// This tests case verifies that create requests with a ghost
			// host set is rejected.
			in: &schedule_v1.SingleScheduleResource{
				Name: "Test SingleSchedule 6",
				Relation: &schedule_v1.SingleScheduleResource_TargetHost{
					TargetHost: ghost,
				},
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_OS_UPDATE,
				StartSeconds:   nowF1,
				EndSeconds:     nowF2,
			},
			valid: false,
		},
		"CreateBadTarget2": {
			// This tests case verifies that create requests with a ghost
			// site set is rejected.
			in: &schedule_v1.SingleScheduleResource{
				Name: "Test SingleSchedule 7",
				Relation: &schedule_v1.SingleScheduleResource_TargetSite{
					TargetSite: gsite,
				},
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_OS_UPDATE,
				StartSeconds:   nowF1,
				EndSeconds:     nowF2,
			},
			valid: false,
		},
		"CreateBadTarget3": {
			// This tests case verifies that create requests with a ghost
			// workload set is rejected.
			in: &schedule_v1.SingleScheduleResource{
				Name: "Test SingleSchedule 7",
				Relation: &schedule_v1.SingleScheduleResource_TargetWorkload{
					TargetWorkload: gworkload,
				},
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_OS_UPDATE,
				StartSeconds:   nowF1,
				EndSeconds:     nowF2,
			},
			valid: false,
		},
		"CreateBadTarget5": {
			// This tests case verifies that create requests with a ghost region set is rejected.
			in: &schedule_v1.SingleScheduleResource{
				Name: "Test SingleSchedule 7",
				Relation: &schedule_v1.SingleScheduleResource_TargetRegion{
					TargetRegion: gregion,
				},
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_OS_UPDATE,
				StartSeconds:   nowF1,
				EndSeconds:     nowF2,
			},
			valid: false,
		},
		"CreateBadStartInPast": {
			// This tests case verifies that create requests with start before
			// current time are rejected
			in: &schedule_v1.SingleScheduleResource{
				Name:           "Test SingleSchedule CreateBadStart",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_OS_UPDATE,
				StartSeconds:   nowP1,
			},
			valid: false,
		},
		"CreateBadEnd": {
			// This tests case verifies that create requests with start
			// >= end are rejected.
			in: &schedule_v1.SingleScheduleResource{
				Name:           "Test SingleSchedule 8",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_OS_UPDATE,
				StartSeconds:   nowF1,
				EndSeconds:     1,
			},
			valid: false,
		},
		"CreateGoodStartNoEnd": {
			// This tests case verifies that create requests without end are ok.
			in: &schedule_v1.SingleScheduleResource{
				Name:           "Test SingleSchedule 9",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_OS_UPDATE,
				StartSeconds:   nowF1,
			},
			valid: true,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Singleschedule{Singleschedule: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			// create
			csingleScheduleResp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
			singleSchedResID := csingleScheduleResp.GetSingleschedule().GetResourceId()

			if err != nil {
				if tc.valid {
					t.Errorf("CreateSingleSchedule() failed: %s", err)
				}
			} else {
				tc.in.ResourceId = singleSchedResID // Update with created resource ID.
				tc.in.CreatedAt = csingleScheduleResp.GetSingleschedule().GetCreatedAt()
				tc.in.UpdatedAt = csingleScheduleResp.GetSingleschedule().GetUpdatedAt()
				assertSameResource(t, createresreq, csingleScheduleResp, nil)
				if !tc.valid {
					t.Errorf("CreateSingleSchedule() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				// get
				getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, singleSchedResID)
				require.NoError(t, err, "GetSingleSchedule() failed")

				// verify data
				if eq, diff := inv_testing.ProtoEqualOrDiff(tc.in, getresp.GetResource().GetSingleschedule()); !eq {
					t.Errorf("GetSingleSchedule() data not equal: %v", diff)
				}

				// update
				updateresreq := &inv_v1.Resource{
					Resource: &inv_v1.Resource_Singleschedule{
						Singleschedule: &schedule_v1.SingleScheduleResource{
							Name: "Updated Name",
						},
					},
				}

				fm := &fieldmaskpb.FieldMask{Paths: []string{ssr.FieldName}}
				// update non-existent first
				upRes, err := inv_testing.TestClients[inv_testing.APIClient].Update(ctx,
					"singlesche-12345678",
					fm,
					updateresreq)
				require.Error(t, err)
				assert.Nil(t, upRes)

				upRes, err = inv_testing.TestClients[inv_testing.APIClient].Update(
					ctx,
					singleSchedResID,
					fm,
					updateresreq,
				)
				if err != nil {
					t.Errorf("UpdateSingleSchedule() failed: %s", err)
				}

				// Validate returned resource
				assertSameResource(t, updateresreq, upRes, fm)

				// delete non-existent first
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, "singlesche-12345678")
				require.Error(t, err)

				// delete
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(
					ctx,
					singleSchedResID,
				)
				if err != nil {
					t.Errorf("DeleteSingleSchedule() failed %s", err)
				}

				_, err = inv_testing.TestClients[inv_testing.APIClient].Get(ctx, singleSchedResID)
				require.Error(t, err, "Failure - GetSingleSchedule was not deleted, but should be deleted")
			}
		})
	}
}

func Test_FilterSingleSchedules(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site := inv_testing.CreateSite(t, nil, nil)
	host := inv_testing.CreateHost(t, site, nil)
	// Setting again the edge nilled by the helper
	singleSched1 := inv_testing.CreateSingleSchedule(t, host, nil, schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE)
	singleSched1.Relation = &schedule_v1.SingleScheduleResource_TargetHost{
		TargetHost: host,
	}
	singleSched2 := inv_testing.CreateSingleSchedule(t, nil, site, schedule_v1.ScheduleStatus_SCHEDULE_STATUS_SHIPPING)
	singleSched2.Relation = &schedule_v1.SingleScheduleResource_TargetSite{
		TargetSite: site,
	}
	singleSched3 := inv_testing.CreateSingleSchedule(t, nil, nil, schedule_v1.ScheduleStatus_SCHEDULE_STATUS_OS_UPDATE)
	singleSched4 := inv_testing.CreateSingleSchedule(t, nil, nil, schedule_v1.ScheduleStatus_SCHEDULE_STATUS_UNSPECIFIED)

	singleSched5 := inv_testing.CreateSingleSchedule(t, nil, nil,
		schedule_v1.ScheduleStatus_SCHEDULE_STATUS_SHIPPING, inv_testing.SSRRegion(region))
	singleSched5.Relation = &schedule_v1.SingleScheduleResource_TargetRegion{
		TargetRegion: region,
	}
	testcases := map[string]struct {
		in        *inv_v1.ResourceFilter
		resources []*schedule_v1.SingleScheduleResource
		valid     bool
	}{
		"NoFilter": {
			in: &inv_v1.ResourceFilter{},
			resources: []*schedule_v1.SingleScheduleResource{
				singleSched1, singleSched2, singleSched3, singleSched4, singleSched5,
			},
			valid: true,
		},
		"NoFilterOrderByResourceID": {
			in: &inv_v1.ResourceFilter{
				OrderBy: ssr.FieldResourceID,
			},
			resources: []*schedule_v1.SingleScheduleResource{
				singleSched1, singleSched2, singleSched3, singleSched4, singleSched5,
			},
			valid: true,
		},
		"FilterByEmptyResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = ""`, ssr.FieldResourceID),
			},
			valid: true,
		},
		"FilterByResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, ssr.FieldResourceID, singleSched1.ResourceId),
			},
			resources: []*schedule_v1.SingleScheduleResource{singleSched1},
			valid:     true,
		},
		"FilterStatus": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %s`, ssr.FieldScheduleStatus, schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE),
			},
			resources: []*schedule_v1.SingleScheduleResource{singleSched1},
			valid:     true,
		},
		"FilterHost": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, ssr.EdgeTargetHost, hosts.FieldResourceID, host.GetResourceId()),
			},
			resources: []*schedule_v1.SingleScheduleResource{singleSched1},
			valid:     true,
		},
		"FilterSite": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, ssr.EdgeTargetSite, siteresource.FieldResourceID, site.GetResourceId()),
			},
			resources: []*schedule_v1.SingleScheduleResource{singleSched2},
			valid:     true,
		},
		"FilterRegion": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(
					`%s.%s = %q`, ssr.EdgeTargetRegion, siteresource.FieldResourceID, region.GetResourceId()),
			},
			resources: []*schedule_v1.SingleScheduleResource{singleSched5},
			valid:     true,
		},
		"FilterRegionEmpty": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s)`, ssr.EdgeTargetRegion),
			},
			resources: []*schedule_v1.SingleScheduleResource{singleSched1, singleSched2, singleSched3, singleSched4},
			valid:     true,
		},
		"FilterSiteEmpty": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s)`, ssr.EdgeTargetSite),
			},
			resources: []*schedule_v1.SingleScheduleResource{singleSched1, singleSched3, singleSched4, singleSched5},
			valid:     true,
		},
		"FilterHostEmpty": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s)`, ssr.EdgeTargetHost),
			},
			resources: []*schedule_v1.SingleScheduleResource{singleSched2, singleSched3, singleSched4, singleSched5},
			valid:     true,
		},
		"FilterStatusEmpty": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = null`, ssr.FieldScheduleStatus),
			},
			resources: []*schedule_v1.SingleScheduleResource{singleSched4},
			valid:     true,
		},
		"FilterByHasSite": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s)`, ssr.EdgeTargetSite),
			},
			resources: []*schedule_v1.SingleScheduleResource{singleSched2},
			valid:     true,
		},
		"FilterBySiteID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, ssr.EdgeTargetSite, siteresource.FieldResourceID,
					site.GetResourceId()),
			},
			resources: []*schedule_v1.SingleScheduleResource{singleSched2},
			valid:     true,
		},
		"FilterByHasRegion": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s)`, ssr.EdgeTargetRegion),
			},
			resources: []*schedule_v1.SingleScheduleResource{singleSched5},
			valid:     true,
		},
		"FilterByRegionID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, ssr.EdgeTargetRegion, siteresource.FieldResourceID,
					region.GetResourceId()),
			},
			resources: []*schedule_v1.SingleScheduleResource{singleSched5},
			valid:     true,
		},
		"FilterByHasHostHasSite": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s.%s)`, ssr.EdgeTargetHost, hosts.EdgeSite),
			},
			resources: []*schedule_v1.SingleScheduleResource{singleSched1},
			valid:     true,
		},
		"FilterLimit": {
			in: &inv_v1.ResourceFilter{
				Offset: 0,
				Limit:  5,
			},
			resources: []*schedule_v1.SingleScheduleResource{
				singleSched1, singleSched2, singleSched3, singleSched4, singleSched5,
			},
			valid: true,
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

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Singleschedule{}} // Set the resource kind
			findres, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("FilterSingleSchedules() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("FilterSingleSchedules() succeeded but should have failed")
				}
			}

			// only get/delete if valid test with non-zero returned response and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				if len(findres.Resources) != len(tc.resources) {
					t.Errorf("Expected to obtain %d Resource IDs, but obtained back %d Resource IDs",
						len(tc.resources), len(findres.Resources))
				}

				resIDs := inv_testing.GetSortedResourceIDSlice(tc.resources)
				inv_testing.SortHasResourceIDAndTenantID(findres.Resources)

				if !reflect.DeepEqual(resIDs, findres.Resources) {
					t.Errorf(
						"FilterSingleSchedules() failed - want: %s, got: %s",
						resIDs,
						findres.Resources,
					)
				}
			}

			listres, err := inv_testing.TestClients[inv_testing.APIClient].List(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("ListSingleSchedules() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("ListSingleSchedules() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				resources := make([]*schedule_v1.SingleScheduleResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetSingleschedule())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					singleSchedEdgesOnlyResourceID(expected)
					singleSchedEdgesOnlyResourceID(resources[i])

					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, resources[i]); !eq {
						t.Errorf("ListSingleSchedules() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

type UpdateTCSSched struct {
	in           *schedule_v1.SingleScheduleResource
	fieldMask    *fieldmaskpb.FieldMask
	valid        bool
	expErrorCode codes.Code
}

func runUpdateTCForSSched(t *testing.T, testcases map[string]UpdateTCSSched, resourceID string) {
	t.Helper()

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			updateresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Singleschedule{Singleschedule: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			upRes, err := inv_testing.TestClients[inv_testing.APIClient].Update(
				ctx,
				resourceID,
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
			getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, resourceID)
			require.NoError(t, err, "GetResource() failed")

			assertSameResource(t, updateresreq, getresp.GetResource(), tc.fieldMask)
		})
	}
}

// Start with no target; new name; target host; reset host with put;
// update both site and host; integrity checkes are also part of these tests
// NOTE tc are independent by the order of exec.
func Test_UpdateSingleSchedule1(t *testing.T) {
	site := inv_testing.CreateSite(t, nil, nil)
	host := inv_testing.CreateHost(t, site, nil)
	ghost := &computev1.HostResource{
		ResourceId: "host-12345678",
	}
	gsite := &locationv1.SiteResource{
		ResourceId: "site-12345678",
	}

	// create SingleSchedule to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Singleschedule{
			Singleschedule: &schedule_v1.SingleScheduleResource{
				Name:           "Test SingleSchedule 1",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				StartSeconds:   nowF1,
				EndSeconds:     nowF2,
			},
		},
	}
	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	csingleScheduleResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	singleSchedResID := csingleScheduleResp.GetSingleschedule().GetResourceId()
	t.Cleanup(func() { inv_testing.DeleteResource(t, singleSchedResID) })

	testcases := map[string]UpdateTCSSched{
		"Update1": {
			in: &schedule_v1.SingleScheduleResource{
				Name: "Updated Name",
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ssr.FieldName},
			},
			valid: true,
		},
		"Update2": {
			in: &schedule_v1.SingleScheduleResource{
				Name: "Updated Name 2",
				Relation: &schedule_v1.SingleScheduleResource_TargetHost{
					TargetHost: host,
				},
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ssr.EdgeTargetHost, ssr.FieldName},
			},
			valid: true,
		},
		"BadUpdate1": {
			in: &schedule_v1.SingleScheduleResource{
				Name: "Updated Name 3",
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"INVALID_FIELD"},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"BadUpdateNoFieldmask": {
			in: &schedule_v1.SingleScheduleResource{
				Name: "Updated Name 3",
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"Update3": {
			in: &schedule_v1.SingleScheduleResource{
				Name:           "Updated Name 4",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				StartSeconds:   nowD1,
				EndSeconds:     nowD2,
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ssr.FieldName, ssr.FieldScheduleStatus, ssr.FieldStartSeconds, ssr.FieldEndSeconds},
			},
			valid: true,
		},
		"BadUpdate2": {
			in: &schedule_v1.SingleScheduleResource{
				Name:           "Updated Name 6",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				StartSeconds:   nowD1,
				EndSeconds:     nowD1,
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ssr.FieldName, ssr.FieldScheduleStatus, ssr.FieldStartSeconds, ssr.FieldEndSeconds},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"BadUpdate3": {
			in: &schedule_v1.SingleScheduleResource{
				Name:           "Updated Name 7",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				Relation: &schedule_v1.SingleScheduleResource_TargetHost{
					TargetHost: ghost,
				},
				StartSeconds: nowD1,
				EndSeconds:   nowD2,
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					ssr.FieldName, ssr.FieldScheduleStatus, ssr.EdgeTargetHost,
					ssr.FieldStartSeconds, ssr.FieldEndSeconds,
				},
			},
			valid:        false,
			expErrorCode: codes.NotFound,
		},
		"BadUpdate4": {
			in: &schedule_v1.SingleScheduleResource{
				Name:           "Updated Name 9",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				Relation: &schedule_v1.SingleScheduleResource_TargetSite{
					TargetSite: gsite,
				},
				StartSeconds: nowD1,
				EndSeconds:   nowD2,
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					ssr.FieldName, ssr.FieldScheduleStatus, ssr.EdgeTargetSite,
					ssr.FieldStartSeconds, ssr.FieldEndSeconds,
				},
			},
			valid:        false,
			expErrorCode: codes.NotFound,
		},
		"BadUpdateStartInPast": {
			in: &schedule_v1.SingleScheduleResource{
				StartSeconds: nowP1,
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ssr.FieldStartSeconds},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
	}
	runUpdateTCForSSched(t, testcases, singleSchedResID)
}

// Start with target host; full update; target site; target workload; target host and target site; target host, site and workload;
// NOTE tc are independent by the order of exec.
func Test_UpdateSingleSchedule2(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site := inv_testing.CreateSite(t, nil, nil)
	host := inv_testing.CreateHost(t, site, nil)
	workload := inv_testing.CreateWorkload(t)

	// create SingleSchedule to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Singleschedule{
			Singleschedule: &schedule_v1.SingleScheduleResource{
				Name:           "Test SingleSchedule 1",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				Relation: &schedule_v1.SingleScheduleResource_TargetHost{
					TargetHost: host,
				},
				StartSeconds: nowF1,
				EndSeconds:   nowF2,
			},
		},
	}
	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	csingleScheduleResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	singleSchedResID := csingleScheduleResp.GetSingleschedule().GetResourceId()
	t.Cleanup(func() { inv_testing.DeleteResource(t, singleSchedResID) })

	testcases := map[string]UpdateTCSSched{
		"Update1": {
			in: &schedule_v1.SingleScheduleResource{
				Name:           "Updated Name 1",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				Relation: &schedule_v1.SingleScheduleResource_TargetHost{
					TargetHost: host,
				},
				StartSeconds: nowD1,
				EndSeconds:   nowD2,
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					ssr.FieldName, ssr.FieldScheduleStatus, ssr.EdgeTargetHost,
					ssr.FieldStartSeconds, ssr.FieldEndSeconds,
				},
			},
			valid: true,
		},
		"BadUpdate1": {
			in: &schedule_v1.SingleScheduleResource{
				Name: "Updated Name 2",
				Relation: &schedule_v1.SingleScheduleResource_TargetSite{
					TargetSite: site,
				},
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ssr.EdgeTargetSite, ssr.FieldName},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"BadUpdate2": {
			in: &schedule_v1.SingleScheduleResource{
				Name: "Updated Name 3",
				Relation: &schedule_v1.SingleScheduleResource_TargetWorkload{
					TargetWorkload: workload,
				},
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ssr.EdgeTargetWorkload, ssr.FieldName},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"BadUpdate3": {
			in: &schedule_v1.SingleScheduleResource{
				Name: "add another target(region)",
				Relation: &schedule_v1.SingleScheduleResource_TargetRegion{
					TargetRegion: region,
				},
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ssr.EdgeTargetRegion, ssr.FieldName},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
	}
	runUpdateTCForSSched(t, testcases, singleSchedResID)
}

// Start with no target; target site; reset site with put;
// NOTE tc are independent by the order of exec.
func Test_UpdateSingleSchedule3(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site := inv_testing.CreateSite(t, region, nil)

	// create SingleSchedule to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Singleschedule{
			Singleschedule: &schedule_v1.SingleScheduleResource{
				Name:           "Test SingleSchedule 1",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				StartSeconds:   nowF1,
				EndSeconds:     nowF2,
			},
		},
	}
	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	csingleScheduleResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	singleSchedResID := csingleScheduleResp.GetSingleschedule().GetResourceId()
	t.Cleanup(func() { inv_testing.DeleteResource(t, singleSchedResID) })

	testcases := map[string]UpdateTCSSched{
		"Update1": {
			in: &schedule_v1.SingleScheduleResource{
				Name: "Updated Name 1",
				Relation: &schedule_v1.SingleScheduleResource_TargetSite{
					TargetSite: site,
				},
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ssr.EdgeTargetSite, ssr.FieldName},
			},
			valid: true,
		},
		"Update2": {
			in: &schedule_v1.SingleScheduleResource{
				Name:           "Updated Name 2",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				StartSeconds:   nowD1,
				EndSeconds:     nowD2,
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ssr.FieldName, ssr.FieldScheduleStatus, ssr.FieldStartSeconds, ssr.FieldEndSeconds},
			},
			valid: true,
		},
	}
	runUpdateTCForSSched(t, testcases, singleSchedResID)
}

// Start with target site; full update; target host.
// NOTE tc are independent by the order of exec.
func Test_UpdateSingleSchedule4(t *testing.T) {
	site := inv_testing.CreateSite(t, nil, nil)
	host := inv_testing.CreateHost(t, site, nil)

	// create SingleSchedule to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Singleschedule{
			Singleschedule: &schedule_v1.SingleScheduleResource{
				Name:           "Test SingleSchedule 1",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				Relation: &schedule_v1.SingleScheduleResource_TargetSite{
					TargetSite: site,
				},
				StartSeconds: nowF1,
				EndSeconds:   nowF2,
			},
		},
	}
	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	csingleScheduleResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	singleSchedResID := csingleScheduleResp.GetSingleschedule().GetResourceId()
	t.Cleanup(func() { inv_testing.DeleteResource(t, singleSchedResID) })

	testcases := map[string]UpdateTCSSched{
		"Update1": {
			in: &schedule_v1.SingleScheduleResource{
				Name:           "Updated Name 1",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				Relation: &schedule_v1.SingleScheduleResource_TargetSite{
					TargetSite: site,
				},
				StartSeconds: nowD1,
				EndSeconds:   nowD2,
			},
			fieldMask: &fieldmaskpb.FieldMask{Paths: []string{
				ssr.FieldName, ssr.FieldScheduleStatus, ssr.EdgeTargetSite, ssr.FieldStartSeconds, ssr.FieldEndSeconds,
			}},
			valid: true,
		},
		"BadUpdate1": {
			in: &schedule_v1.SingleScheduleResource{
				Name: "Updated Name 2",
				Relation: &schedule_v1.SingleScheduleResource_TargetHost{
					TargetHost: host,
				},
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ssr.EdgeTargetHost, ssr.FieldName},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
	}
	runUpdateTCForSSched(t, testcases, singleSchedResID)
}

// Start with no target; target site; target workload; reset target; target host.
// NOTE tc are independent by the order of exec.
func Test_UpdateSingleSchedule5(t *testing.T) {
	site := inv_testing.CreateSite(t, nil, nil)
	host := inv_testing.CreateHost(t, site, nil)
	workload := inv_testing.CreateWorkload(t)

	// create SingleSchedule to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Singleschedule{
			Singleschedule: &schedule_v1.SingleScheduleResource{
				Name:           "Test SingleSchedule 1",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				StartSeconds:   nowF1,
				EndSeconds:     nowF2,
			},
		},
	}
	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	csingleScheduleResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	singleSchedResID := csingleScheduleResp.GetSingleschedule().GetResourceId()
	t.Cleanup(func() { inv_testing.DeleteResource(t, singleSchedResID) })

	testcases := map[string]UpdateTCSSched{
		"Update1": {
			in: &schedule_v1.SingleScheduleResource{
				Name:           "Updated Name 1",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				Relation: &schedule_v1.SingleScheduleResource_TargetSite{
					TargetSite: site,
				},
				StartSeconds: nowD1,
				EndSeconds:   nowD2,
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					ssr.FieldName, ssr.FieldScheduleStatus, ssr.EdgeTargetSite,
					ssr.FieldStartSeconds, ssr.FieldEndSeconds,
				},
			},
			valid: true,
		},
		"Update2": {
			in: &schedule_v1.SingleScheduleResource{
				Name:           "Updated Name 2",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				Relation: &schedule_v1.SingleScheduleResource_TargetHost{
					TargetHost: host,
				},
				StartSeconds: nowD1,
				EndSeconds:   nowD2,
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					ssr.FieldName, ssr.FieldScheduleStatus, ssr.EdgeTargetHost,
					ssr.EdgeTargetSite, ssr.EdgeTargetWorkload, ssr.FieldStartSeconds, ssr.FieldEndSeconds,
				},
			},
			valid: true,
		},
		"Update3": {
			in: &schedule_v1.SingleScheduleResource{
				Name:           "Updated Name 3",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				Relation: &schedule_v1.SingleScheduleResource_TargetWorkload{
					TargetWorkload: workload,
				},
				StartSeconds: nowD1,
				EndSeconds:   nowD2,
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					ssr.FieldName, ssr.FieldScheduleStatus, ssr.EdgeTargetHost, ssr.EdgeTargetSite,
					ssr.EdgeTargetWorkload, ssr.FieldStartSeconds, ssr.FieldEndSeconds,
				},
			},
			valid: true,
		},
		"Update4": {
			in: &schedule_v1.SingleScheduleResource{
				Name:           "Updated Name 3",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				StartSeconds:   nowD1,
				EndSeconds:     nowD2,
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					ssr.FieldName, ssr.FieldScheduleStatus, ssr.EdgeTargetHost,
					ssr.EdgeTargetSite, ssr.EdgeTargetWorkload, ssr.FieldStartSeconds, ssr.FieldEndSeconds,
				},
			},
			valid: true,
		},
	}
	runUpdateTCForSSched(t, testcases, singleSchedResID)
}

// Test bad updates.
func Test_UpdateSingleSchedule6(t *testing.T) {
	// create SingleSchedule to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Singleschedule{
			Singleschedule: &schedule_v1.SingleScheduleResource{
				Name:           "Test SingleSchedule 1",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				StartSeconds:   nowF1,
				EndSeconds:     nowF2,
			},
		},
	}
	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	csingleScheduleResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	singleSchedResID := csingleScheduleResp.GetSingleschedule().GetResourceId()
	t.Cleanup(func() { inv_testing.DeleteResource(t, singleSchedResID) })

	testcases := map[string]UpdateTCSSched{
		// Try update with end == start
		"BadUpdate1": {
			in: &schedule_v1.SingleScheduleResource{
				EndSeconds: nowF1,
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ssr.FieldEndSeconds},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		// Try update with start == end
		"BadUpdate2": {
			in: &schedule_v1.SingleScheduleResource{
				StartSeconds: nowF2,
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ssr.FieldStartSeconds},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
	}
	runUpdateTCForSSched(t, testcases, singleSchedResID)
}

// Start with no target; target workload; reset workload with put;
// NOTE tc are independent by the order of exec.
func Test_UpdateSingleSchedule7(t *testing.T) {
	workload := inv_testing.CreateWorkload(t)

	// create SingleSchedule to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Singleschedule{
			Singleschedule: &schedule_v1.SingleScheduleResource{
				Name:           "Test SingleSchedule 1",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				StartSeconds:   nowF1,
				EndSeconds:     nowF2,
			},
		},
	}
	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	csingleScheduleResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	singleSchedResID := csingleScheduleResp.GetSingleschedule().GetResourceId()
	t.Cleanup(func() { inv_testing.DeleteResource(t, singleSchedResID) })

	testcases := map[string]UpdateTCSSched{
		"Update1": {
			in: &schedule_v1.SingleScheduleResource{
				Name: "Updated Name 1",
				Relation: &schedule_v1.SingleScheduleResource_TargetWorkload{
					TargetWorkload: workload,
				},
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ssr.EdgeTargetWorkload, ssr.FieldName},
			},
			valid: true,
		},
		"Update2": {
			in: &schedule_v1.SingleScheduleResource{
				Name:           "Updated Name 2",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				StartSeconds:   nowD1,
				EndSeconds:     nowD2,
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ssr.FieldName, ssr.FieldScheduleStatus, ssr.FieldStartSeconds, ssr.FieldEndSeconds},
			},
			valid: true,
		},
	}
	runUpdateTCForSSched(t, testcases, singleSchedResID)
}

// Start with no target; target region; reset region with put;
// NOTE tc are independent by the order of exec.
func Test_UpdateSingleSchedule8(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)

	// create SingleSchedule to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Singleschedule{
			Singleschedule: &schedule_v1.SingleScheduleResource{
				Name:           "Test SingleSchedule 1",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				StartSeconds:   nowF1,
				EndSeconds:     nowF2,
			},
		},
	}
	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	csingleScheduleResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	singleSchedResID := csingleScheduleResp.GetSingleschedule().GetResourceId()
	t.Cleanup(func() { inv_testing.DeleteResource(t, singleSchedResID) })

	testcases := map[string]UpdateTCSSched{
		"Update1": {
			in: &schedule_v1.SingleScheduleResource{
				Name: "Updated Name 1",
				Relation: &schedule_v1.SingleScheduleResource_TargetRegion{
					TargetRegion: region,
				},
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ssr.EdgeTargetRegion, ssr.FieldName},
			},
			valid: true,
		},
		"Update2": {
			in: &schedule_v1.SingleScheduleResource{
				Name:           "Updated Name 2",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				StartSeconds:   nowD1,
				EndSeconds:     nowD2,
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ssr.FieldName, ssr.FieldScheduleStatus, ssr.FieldStartSeconds, ssr.FieldEndSeconds},
			},
			valid: true,
		},
	}
	runUpdateTCForSSched(t, testcases, singleSchedResID)
}

// Start with target region; full update; target region.
// NOTE tc are independent by the order of exec.
func Test_UpdateSingleSchedule9(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site := inv_testing.CreateSite(t, region, nil)
	host := inv_testing.CreateHost(t, site, nil)

	// create SingleSchedule to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Singleschedule{
			Singleschedule: &schedule_v1.SingleScheduleResource{
				Name:           "Test SingleSchedule 1",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				Relation: &schedule_v1.SingleScheduleResource_TargetRegion{
					TargetRegion: region,
				},
				StartSeconds: nowF1,
				EndSeconds:   nowF2,
			},
		},
	}
	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	csingleScheduleResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	singleSchedResID := csingleScheduleResp.GetSingleschedule().GetResourceId()
	t.Cleanup(func() { inv_testing.DeleteResource(t, singleSchedResID) })

	testcases := map[string]UpdateTCSSched{
		"Update1": {
			in: &schedule_v1.SingleScheduleResource{
				Name:           "Updated Name 1",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				Relation: &schedule_v1.SingleScheduleResource_TargetRegion{
					TargetRegion: region,
				},
				StartSeconds: nowD1,
				EndSeconds:   nowD2,
			},
			fieldMask: &fieldmaskpb.FieldMask{Paths: []string{
				ssr.FieldName, ssr.FieldScheduleStatus, ssr.EdgeTargetRegion, ssr.FieldStartSeconds, ssr.FieldEndSeconds,
			}},
			valid: true,
		},
		"BadUpdate1": {
			in: &schedule_v1.SingleScheduleResource{
				Name: "Updated Name 2",
				Relation: &schedule_v1.SingleScheduleResource_TargetHost{
					TargetHost: host,
				},
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ssr.EdgeTargetHost, ssr.FieldName},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
	}
	runUpdateTCForSSched(t, testcases, singleSchedResID)
}

func singleSchedEdgesOnlyResourceID(expected *schedule_v1.SingleScheduleResource) {
	if expected.GetTargetHost() != nil {
		expected.Relation = &schedule_v1.SingleScheduleResource_TargetHost{
			TargetHost: &computev1.HostResource{ResourceId: expected.GetTargetHost().ResourceId},
		}
	}
	if expected.GetTargetSite() != nil {
		expected.Relation = &schedule_v1.SingleScheduleResource_TargetSite{
			TargetSite: &locationv1.SiteResource{ResourceId: expected.GetTargetSite().ResourceId},
		}
	}
	if expected.GetTargetWorkload() != nil {
		expected.Relation = &schedule_v1.SingleScheduleResource_TargetWorkload{
			TargetWorkload: &computev1.WorkloadResource{ResourceId: expected.GetTargetWorkload().ResourceId},
		}
	}
}

func Test_NestedSingleSchedules(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site := inv_testing.CreateSite(t, region, nil)
	host := inv_testing.CreateHost(t, site, nil)
	// Setting again the edge nilled by the helper
	singleSched1 := inv_testing.CreateSingleSchedule(t, host, nil, schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE)
	singleSched1.Relation = &schedule_v1.SingleScheduleResource_TargetHost{
		TargetHost: host,
	}
	singleSched2 := inv_testing.CreateSingleSchedule(t, nil, site, schedule_v1.ScheduleStatus_SCHEDULE_STATUS_SHIPPING)
	singleSched2.Relation = &schedule_v1.SingleScheduleResource_TargetSite{
		TargetSite: site,
	}

	testcases := map[string]struct {
		in                *inv_v1.ResourceFilter
		resources         []*schedule_v1.SingleScheduleResource
		valid             bool
		expectedCodeError codes.Code
	}{
		"FilterByHostSiteID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s = %q`, ssr.EdgeTargetHost, hosts.EdgeSite,
					siteresource.FieldResourceID, site.GetResourceId()),
			},
			resources: []*schedule_v1.SingleScheduleResource{singleSched1},
			valid:     true,
		},
		"FilterBySiteRegionID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s = %q`, ssr.EdgeTargetSite, siteresource.EdgeRegion,
					regionresource.FieldResourceID, region.GetResourceId()),
			},
			resources: []*schedule_v1.SingleScheduleResource{singleSched2},
			valid:     true,
		},
		"FailTooDeep": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s.%s.%s.%s = %q`, ssr.EdgeTargetSite, siteresource.EdgeRegion,
					regionresource.EdgeParentRegion, regionresource.EdgeParentRegion, regionresource.EdgeParentRegion,
					regionresource.FieldResourceID, region.GetResourceId()),
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

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Singleschedule{}} // Set the resource kind

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
						"FilterSingleSchedules() failed - want: %s, got: %s",
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

				resources := make([]*schedule_v1.SingleScheduleResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetSingleschedule())
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

func Test_SingleScheduleEnumStatusMap(t *testing.T) {
	v, err := store.SingleScheduleEnumStatusMap("invalid_input",
		int32(schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE))
	assert.Error(t, err)
	assert.Nil(t, v)
}

func TestSingleScheduleMTSanity(t *testing.T) {
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	suite.Run(t, &struct{ mt }{
		mt: mt{
			createResource: func(tenantID string) (string, *inv_v1.Resource) {
				rs := dao.CreateSingleSchedule(t, tenantID)
				res, err := util.WrapResource(rs)
				require.NoError(t, err)
				return rs.GetResourceId(), res
			},
		},
	})
}

func TestDeleteResources_SingleSchedules(t *testing.T) {
	suite.Run(t, &struct{ hardDeleteAllResourcesSuite }{
		hardDeleteAllResourcesSuite: hardDeleteAllResourcesSuite{
			createModel: func(dao *inv_testing.InvResourceDAO) (string, int) {
				tenantID := uuid.NewString()
				region := dao.CreateRegion(t, tenantID)
				site := dao.CreateSite(t, tenantID)
				host := dao.CreateHost(t, tenantID)
				workload := dao.CreateWorkload(t, tenantID)
				return tenantID, len([]any{
					dao.CreateSingleScheduleNoCleanup(t, tenantID, inv_testing.SSRRegion(region)),
					dao.CreateSingleScheduleNoCleanup(t, tenantID, inv_testing.SSRTargetSite(site)),
					dao.CreateSingleScheduleNoCleanup(t, tenantID, inv_testing.SSRTargetHost(host)),
					dao.CreateSingleScheduleNoCleanup(t, tenantID, inv_testing.SSRTargetWorkload(workload)),
				})
			},
			resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_SINGLESCHEDULE,
		},
	})
}
