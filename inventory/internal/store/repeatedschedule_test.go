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

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/regionresource"
	rsr "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/repeatedscheduleresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/siteresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/store"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	location_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/location/v1"
	schedule_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/schedule/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

func Test_Create_Get_Delete_Update_RepeatedSchedule(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site := inv_testing.CreateSite(t, nil, nil)
	host := inv_testing.CreateHost(t, site, nil)
	workload := inv_testing.CreateWorkload(t)
	ghost := &computev1.HostResource{
		ResourceId: "host-12345678",
	}
	gregion := &location_v1.RegionResource{
		ResourceId: "region-12345678",
	}
	gsite := &location_v1.SiteResource{
		ResourceId: "site-12345678",
	}
	gworkload := &computev1.WorkloadResource{
		ResourceId: "workload-12345678",
	}

	testcases := map[string]struct {
		in    *schedule_v1.RepeatedScheduleResource
		valid bool
	}{
		"CreateGoodRepeatedSchedule1": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:            "Test RepeatedSchedule 1",
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			valid: true,
		},
		"CreateGoodRepeatedSchedule2": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 2",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetHost{
					TargetHost: host,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			valid: true,
		},
		"CreateGoodRepeatedSchedule3": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 3",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetSite{
					TargetSite: site,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			valid: true,
		},
		"CreateGoodRepeatedSchedule4": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 4",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetWorkload{
					TargetWorkload: workload,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			valid: true,
		},
		"CreateGoodRepeatedSchedule5": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 5",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetRegion{
					TargetRegion: region,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			valid: true,
		},
		"CreateBadRepeatedScheduleWithResourceIdSet": {
			// This tests case verifies that create requests with a resource ID
			// already set are rejected.
			in: &schedule_v1.RepeatedScheduleResource{
				ResourceId:      "repeatedsche-12345678",
				Name:            "Test RepeatedSchedule 3",
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			valid: false,
		},
		"CreateBadTarget1": {
			// This tests case verifies that create requests with a ghost
			// host set is rejected.
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 1",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetHost{
					TargetHost: ghost,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			valid: false,
		},
		"CreateBadTarget2": {
			// This tests case verifies that create requests with a ghost
			// site set is rejected.
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 2",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetSite{
					TargetSite: gsite,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			valid: false,
		},
		"CreateBadTarget3": {
			// This tests case verifies that create requests with a ghost
			// workload set is rejected.
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 3",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetWorkload{
					TargetWorkload: gworkload,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			valid: false,
		},
		"CreateBadTarget4": {
			// This tests case verifies that create requests with a ghost region set is rejected.
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 4",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetRegion{
					TargetRegion: gregion,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			valid: false,
		},
		"CreateBadCronHou": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 7",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetSite{
					TargetSite: site,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(1),
				CronMinutes:     "3",
				CronHours:       "30",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			valid: false,
		},
		"CreateBadCronDayMonth": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 8",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetSite{
					TargetSite: site,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(1),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "32",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			valid: false,
		},
		"CreateBadCronMonth": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 9",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetSite{
					TargetSite: site,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(1),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "13",
				CronDayWeek:     "0",
			},
			valid: false,
		},
		"CreateBadCronDayWeek": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 10",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetSite{
					TargetSite: site,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(1),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "7",
			},
			valid: false,
		},
		"CreateBadCronValues": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 11",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetSite{
					TargetSite: site,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(1),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
			},
			valid: false,
		},
		"CreateBadMissingCronValues": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 11",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetSite{
					TargetSite: site,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(1),
			},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Repeatedschedule{Repeatedschedule: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			// create
			crepeatedScheduleResp, err := inv_testing.TestClients[inv_testing.APIClient].
				Create(ctx, createresreq)
			var resourceID string

			if err != nil {
				if tc.valid {
					t.Errorf("CreateRepeatedSchedule() failed: %s", err)
				}
			} else {
				resourceID = inv_testing.GetResourceIDOrFail(t, crepeatedScheduleResp)
				tc.in.ResourceId = resourceID // Update with created resource ID.
				tc.in.CreatedAt = crepeatedScheduleResp.GetRepeatedschedule().GetCreatedAt()
				tc.in.UpdatedAt = crepeatedScheduleResp.GetRepeatedschedule().GetUpdatedAt()
				assertSameResource(t, createresreq, crepeatedScheduleResp, nil)
				if !tc.valid {
					t.Errorf("CreateRepeatedSchedule() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				// get
				getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, resourceID)
				require.NoError(t, err, "GetRepeatedSchedule() failed")

				// verify data
				if eq, diff := inv_testing.ProtoEqualOrDiff(tc.in, getresp.GetResource().GetRepeatedschedule()); !eq {
					t.Errorf("GetRepeatedSchedule() data not equal: %v", diff)
				}

				// update
				updateresreq := &inv_v1.Resource{
					Resource: &inv_v1.Resource_Repeatedschedule{
						Repeatedschedule: &schedule_v1.RepeatedScheduleResource{
							Name:         "Updated Name",
							CronMinutes:  tc.in.CronMinutes,
							CronHours:    tc.in.CronHours,
							CronDayMonth: tc.in.CronDayMonth,
							CronMonth:    tc.in.CronMonth,
							CronDayWeek:  tc.in.CronDayWeek,
						},
					},
				}

				// update non-existent first
				upRes, err := inv_testing.TestClients[inv_testing.APIClient].Update(ctx,
					"repeatedsche-12345678",
					&fieldmaskpb.FieldMask{Paths: []string{rsr.FieldName}},
					updateresreq)
				require.Error(t, err)
				assert.Nil(t, upRes)

				fm := &fieldmaskpb.FieldMask{Paths: []string{
					rsr.FieldName, rsr.FieldCronMinutes, rsr.FieldCronHours,
					rsr.FieldCronDayMonth, rsr.FieldCronMonth, rsr.FieldCronDayWeek,
				}}
				upRes, err = inv_testing.TestClients[inv_testing.APIClient].Update(
					ctx,
					resourceID,
					fm,
					updateresreq,
				)
				if err != nil {
					t.Errorf("UpdateRepeatedSchedule() failed: %s", err)
				}

				// Validate returned resource
				assertSameResource(t, updateresreq, upRes, fm)

				// delete non-existent first
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, "repeatedsche-12345678")
				require.Error(t, err)

				// delete
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(
					ctx,
					resourceID,
				)
				if err != nil {
					t.Errorf("DeleteRepeatedSchedule() failed %s", err)
				}

				_, err = inv_testing.TestClients[inv_testing.APIClient].Get(ctx, resourceID)
				require.Error(t, err, "Failure - RepeatedSchedule was not deleted, but should be deleted")
			}
		})
	}
}

func Test_FilterRepeatedSchedules(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site := inv_testing.CreateSite(t, nil, nil)
	host := inv_testing.CreateHost(t, site, nil)
	// Setting again the edge nilled by the helper
	repeatedSched1 := inv_testing.CreateRepeatedSchedule(t, host, nil, schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE)
	repeatedSched1.Relation = &schedule_v1.RepeatedScheduleResource_TargetHost{
		TargetHost: host,
	}
	repeatedSched2 := inv_testing.CreateRepeatedSchedule(t, nil, site, schedule_v1.ScheduleStatus_SCHEDULE_STATUS_SHIPPING)
	repeatedSched2.Relation = &schedule_v1.RepeatedScheduleResource_TargetSite{
		TargetSite: site,
	}
	repeatedSched3 := inv_testing.CreateRepeatedSchedule(t, nil, nil, schedule_v1.ScheduleStatus_SCHEDULE_STATUS_OS_UPDATE)
	repeatedSched4 := inv_testing.CreateRepeatedSchedule(t, nil, nil, schedule_v1.ScheduleStatus_SCHEDULE_STATUS_UNSPECIFIED)

	repeatedSched5 := inv_testing.CreateRepeatedSchedule(t, nil, nil,
		schedule_v1.ScheduleStatus_SCHEDULE_STATUS_SHIPPING, inv_testing.RSRRegion(region))
	repeatedSched5.Relation = &schedule_v1.RepeatedScheduleResource_TargetRegion{
		TargetRegion: region,
	}

	testcases := map[string]struct {
		in        *inv_v1.ResourceFilter
		resources []*schedule_v1.RepeatedScheduleResource
		valid     bool
	}{
		"NoFilter": {
			in: &inv_v1.ResourceFilter{},
			resources: []*schedule_v1.RepeatedScheduleResource{
				repeatedSched1,
				repeatedSched2,
				repeatedSched3,
				repeatedSched4,
				repeatedSched5,
			},
			valid: true,
		},
		"NoFilterOrderByResourceID": {
			in: &inv_v1.ResourceFilter{
				OrderBy: rsr.FieldResourceID,
			},
			resources: []*schedule_v1.RepeatedScheduleResource{
				repeatedSched1,
				repeatedSched2,
				repeatedSched3,
				repeatedSched4,
				repeatedSched5,
			},
			valid: true,
		},
		"FilterByEmptyResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = ""`, rsr.FieldResourceID),
			},
			valid: true,
		},
		"FilterByResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, rsr.FieldResourceID, repeatedSched1.ResourceId),
			},
			resources: []*schedule_v1.RepeatedScheduleResource{repeatedSched1},
			valid:     true,
		},
		"FilterStatus": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %s`, rsr.FieldScheduleStatus, schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE),
			},
			resources: []*schedule_v1.RepeatedScheduleResource{repeatedSched1},
			valid:     true,
		},
		"FilterHost": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, rsr.EdgeTargetHost, hostresource.FieldResourceID, host.GetResourceId()),
			},
			resources: []*schedule_v1.RepeatedScheduleResource{repeatedSched1},
			valid:     true,
		},
		"FilterSite": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, rsr.EdgeTargetSite, siteresource.FieldResourceID, site.GetResourceId()),
			},
			resources: []*schedule_v1.RepeatedScheduleResource{repeatedSched2},
			valid:     true,
		},
		"FilterRegion": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, rsr.EdgeTargetRegion, siteresource.FieldResourceID, region.GetResourceId()),
			},
			resources: []*schedule_v1.RepeatedScheduleResource{repeatedSched5},
			valid:     true,
		},
		"FilterRegionEmpty": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s)`, rsr.EdgeTargetRegion),
			},
			resources: []*schedule_v1.RepeatedScheduleResource{
				repeatedSched1,
				repeatedSched2,
				repeatedSched3,
				repeatedSched4,
			},
			valid: true,
		},
		"FilterSiteEmpty": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s)`, rsr.EdgeTargetSite),
			},
			resources: []*schedule_v1.RepeatedScheduleResource{
				repeatedSched1,
				repeatedSched3,
				repeatedSched4,
				repeatedSched5,
			},
			valid: true,
		},
		"FilterHostEmpty": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s)`, rsr.EdgeTargetHost),
			},
			resources: []*schedule_v1.RepeatedScheduleResource{
				repeatedSched2,
				repeatedSched3,
				repeatedSched4,
				repeatedSched5,
			},
			valid: true,
		},
		"FilterSchedStatEmpty": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = null`, rsr.FieldScheduleStatus),
			},
			resources: []*schedule_v1.RepeatedScheduleResource{repeatedSched4},
			valid:     true,
		},
		"FilterByHasSite": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s)`, rsr.EdgeTargetSite),
			},
			resources: []*schedule_v1.RepeatedScheduleResource{repeatedSched2},
			valid:     true,
		},
		"FilterBySiteID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, rsr.EdgeTargetSite, siteresource.FieldResourceID,
					site.GetResourceId()),
			},
			resources: []*schedule_v1.RepeatedScheduleResource{repeatedSched2},
			valid:     true,
		},
		"FilterByHasHostHasSite": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s.%s)`, rsr.EdgeTargetHost, hostresource.EdgeSite),
			},
			resources: []*schedule_v1.RepeatedScheduleResource{repeatedSched1},
			valid:     true,
		},
		"FilterByHasRegion": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s)`, rsr.EdgeTargetRegion),
			},
			resources: []*schedule_v1.RepeatedScheduleResource{repeatedSched5},
			valid:     true,
		},
		"FilterByRegionID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, rsr.EdgeTargetRegion, regionresource.FieldResourceID,
					region.GetResourceId()),
			},
			resources: []*schedule_v1.RepeatedScheduleResource{repeatedSched5},
			valid:     true,
		},
		"FilterLimit": {
			in: &inv_v1.ResourceFilter{
				Offset: 0,
				Limit:  5,
			},
			resources: []*schedule_v1.RepeatedScheduleResource{
				repeatedSched1,
				repeatedSched2,
				repeatedSched3,
				repeatedSched4,
				repeatedSched5,
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

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Repeatedschedule{}} // Set the resource kind
			findres, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("FilterRepeatedSchedules() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("FilterRepeatedSchedules() succeeded but should have failed")
				}
			}

			// only get/delete if valid test with non-zero returned value and hasn't failed, otherwise may segfault
			if !t.Failed() && tc.valid {
				if len(findres.Resources) != len(tc.resources) {
					t.Errorf("Expected to obtain %d Resource IDs, but obtained back %d Resource IDs",
						len(tc.resources), len(findres.Resources))
				}

				resIDs := inv_testing.GetSortedResourceIDSlice(tc.resources)
				inv_testing.SortHasResourceIDAndTenantID(findres.Resources)

				if !reflect.DeepEqual(resIDs, findres.Resources) {
					t.Errorf(
						"FilterRepeatedSchedules() failed - want: %s, got: %s",
						resIDs,
						findres.Resources,
					)
				}
			}

			listres, err := inv_testing.TestClients[inv_testing.APIClient].List(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("ListRepeatedSchedules() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("ListRepeatedSchedules() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				resources := make([]*schedule_v1.RepeatedScheduleResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetRepeatedschedule())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					repeatedSchedEdgesOnlyResourceID(expected)
					repeatedSchedEdgesOnlyResourceID(resources[i])

					fmt.Println("Expected:", expected)
					fmt.Println("Got:", resources[i])

					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, resources[i]); !eq {
						t.Errorf("ListRepeatedSchedules() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

type UpdateTCRSched struct {
	in           *schedule_v1.RepeatedScheduleResource
	fieldMask    *fieldmaskpb.FieldMask
	valid        bool
	expErrorCode codes.Code
}

func runUpdateTCForRSched(t *testing.T, testcases map[string]UpdateTCRSched, resourceID string) {
	t.Helper()

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			updateresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Repeatedschedule{Repeatedschedule: tc.in},
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

// Start with no target; new name; target host; update both site and host;
// integrity checks are also part of these tests
// NOTE tc are independent by the order of exec.
func Test_UpdateRepeatedSchedule1(t *testing.T) {
	site := inv_testing.CreateSite(t, nil, nil)
	host := inv_testing.CreateHost(t, site, nil)
	ghost := &computev1.HostResource{
		ResourceId: "host-12345678",
	}
	gsite := &location_v1.SiteResource{
		ResourceId: "site-12345678",
	}

	// create RepeatedSchedule to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Repeatedschedule{
			Repeatedschedule: &schedule_v1.RepeatedScheduleResource{
				Name:            "Test RepeatedSchedule 1",
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
		},
	}
	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	crepeatedScheduleResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	repeatedSchedResID := inv_testing.GetResourceIDOrFail(t, crepeatedScheduleResp)
	t.Cleanup(func() { inv_testing.DeleteResource(t, repeatedSchedResID) })

	testcases := map[string]UpdateTCRSched{
		"Update1": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:         "Updated Name",
				CronMinutes:  "3",
				CronHours:    "4",
				CronDayMonth: "5",
				CronMonth:    "6",
				CronDayWeek:  "0",
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					rsr.FieldName, rsr.FieldCronMinutes, rsr.FieldCronHours,
					rsr.FieldCronDayMonth, rsr.FieldCronMonth, rsr.FieldCronDayWeek,
				},
			},
			valid: true,
		},
		"Update2": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:         "Updated Name 2",
				CronMinutes:  "3",
				CronHours:    "4",
				CronDayMonth: "5",
				CronMonth:    "6",
				CronDayWeek:  "0",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetHost{
					TargetHost: host,
				},
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					rsr.EdgeTargetHost, rsr.FieldName, rsr.FieldCronMinutes,
					rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth, rsr.FieldCronDayWeek,
				},
			},
			valid: true,
		},
		"BadUpdate1": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Updated Name 3",
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"INVALID_FIELD"},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateNoFieldmask": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:            "Updated Name 4",
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_FIRMWARE_UPDATE,
				DurationSeconds: uint32(4),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"BadUpdateNotFoundHost": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:           "Updated Name 6",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_FIRMWARE_UPDATE,
				Relation: &schedule_v1.RepeatedScheduleResource_TargetHost{
					TargetHost: ghost,
				},
				DurationSeconds: uint32(4),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					rsr.FieldName, rsr.FieldScheduleStatus, rsr.EdgeTargetHost, rsr.FieldDurationSeconds,
					rsr.FieldCronMinutes, rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth, rsr.FieldCronDayWeek,
				},
			},
			valid:        false,
			expErrorCode: codes.NotFound,
		},
		"BadUpdateNotFoundSite": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:           "Updated Name 5",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_FIRMWARE_UPDATE,
				Relation: &schedule_v1.RepeatedScheduleResource_TargetSite{
					TargetSite: gsite,
				},
				DurationSeconds: uint32(4),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					rsr.FieldName, rsr.FieldScheduleStatus, rsr.EdgeTargetSite, rsr.FieldDurationSeconds,
					rsr.FieldCronMinutes, rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth, rsr.FieldCronDayWeek,
				},
			},
			valid:        false,
			expErrorCode: codes.NotFound,
		},
		"BadUpdateMissingCronFields": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:           "Updated Name 5",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_FIRMWARE_UPDATE,
				Relation: &schedule_v1.RepeatedScheduleResource_TargetSite{
					TargetSite: site,
				},
				DurationSeconds: uint32(4),
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					rsr.FieldName, rsr.FieldScheduleStatus, rsr.FieldDurationSeconds,
					rsr.FieldCronMinutes, rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth, rsr.FieldCronDayWeek,
				},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
	}
	runUpdateTCForRSched(t, testcases, repeatedSchedResID)
}

// Start with target host; full update; target site; target workload, target host and target site; target host, site and workload;
// NOTE tc are independent by the order of exec.
func Test_UpdateRepeatedSchedule2(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site := inv_testing.CreateSite(t, nil, nil)
	host := inv_testing.CreateHost(t, site, nil)
	workload := inv_testing.CreateWorkload(t)

	// create RepeatedSchedule to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Repeatedschedule{
			Repeatedschedule: &schedule_v1.RepeatedScheduleResource{
				Name:           "Test RepeatedSchedule 1",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				Relation: &schedule_v1.RepeatedScheduleResource_TargetHost{
					TargetHost: host,
				},
				DurationSeconds: uint32(2),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
		},
	}
	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	crepeatedScheduleResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	repeatedSchedResID := inv_testing.GetResourceIDOrFail(t, crepeatedScheduleResp)
	t.Cleanup(func() { inv_testing.DeleteResource(t, repeatedSchedResID) })

	testcases := map[string]UpdateTCRSched{
		"Update1": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:           "Updated Name 1",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_FIRMWARE_UPDATE,
				Relation: &schedule_v1.RepeatedScheduleResource_TargetHost{
					TargetHost: host,
				},
				DurationSeconds: uint32(4),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			fieldMask: &fieldmaskpb.FieldMask{Paths: []string{
				rsr.FieldName, rsr.FieldScheduleStatus, rsr.EdgeTargetHost, rsr.FieldDurationSeconds,
				rsr.FieldCronMinutes, rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth, rsr.FieldCronDayWeek,
			}},
			valid: true,
		},
		"BadUpdate1": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Updated Name 2",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetSite{
					TargetSite: site,
				},
				CronMinutes:  "3",
				CronHours:    "4",
				CronDayMonth: "5",
				CronMonth:    "6",
				CronDayWeek:  "0",
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					rsr.EdgeTargetSite, rsr.FieldName, rsr.FieldCronMinutes,
					rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth, rsr.FieldCronDayWeek,
				},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"BadUpdate2": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Updated Name 3",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetWorkload{
					TargetWorkload: workload,
				},
				CronMinutes:  "3",
				CronHours:    "4",
				CronDayMonth: "5",
				CronMonth:    "6",
				CronDayWeek:  "0",
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					rsr.EdgeTargetWorkload, rsr.FieldName, rsr.FieldCronMinutes,
					rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth, rsr.FieldCronDayWeek,
				},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"BadUpdate3": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:         "BadUpdate3: add another target(region)",
				CronMinutes:  "3",
				CronHours:    "4",
				CronDayMonth: "5",
				CronMonth:    "6",
				CronDayWeek:  "0",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetRegion{
					TargetRegion: region,
				},
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					rsr.EdgeTargetRegion, rsr.FieldName, rsr.FieldCronMinutes,
					rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth, rsr.FieldCronDayWeek,
				},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
	}
	runUpdateTCForRSched(t, testcases, repeatedSchedResID)
}

// Start with no target; target site; reset site with put;
// NOTE tc are independent by the order of exec.
func Test_UpdateRepeatedSchedule3(t *testing.T) {
	site := inv_testing.CreateSite(t, nil, nil)

	// create RepeatedSchedule to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Repeatedschedule{
			Repeatedschedule: &schedule_v1.RepeatedScheduleResource{
				Name:            "Test RepeatedSchedule 1",
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
		},
	}
	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	crepeatedScheduleResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	repeatedSchedResID := inv_testing.GetResourceIDOrFail(t, crepeatedScheduleResp)
	t.Cleanup(func() { inv_testing.DeleteResource(t, repeatedSchedResID) })

	testcases := map[string]UpdateTCRSched{
		"Update1": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:         "Updated Name 1",
				CronMinutes:  "3",
				CronHours:    "4",
				CronDayMonth: "5",
				CronMonth:    "6",
				CronDayWeek:  "0",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetSite{
					TargetSite: site,
				},
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					rsr.EdgeTargetSite, rsr.FieldName, rsr.FieldCronMinutes,
					rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth, rsr.FieldCronDayWeek,
				},
			},
			valid: true,
		},
		"Update2": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:            "Updated Name 2",
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(4),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					rsr.FieldName, rsr.FieldScheduleStatus, rsr.FieldCronMinutes,
					rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth, rsr.FieldCronDayWeek,
				},
			},
			valid: true,
		},
	}
	runUpdateTCForRSched(t, testcases, repeatedSchedResID)
}

// Start with target site; full update; target host.
// NOTE tc are independent by the order of exec.
func Test_UpdateRepeatedSchedule4(t *testing.T) {
	site := inv_testing.CreateSite(t, nil, nil)
	host := inv_testing.CreateHost(t, site, nil)

	// create RepeatedSchedule to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Repeatedschedule{
			Repeatedschedule: &schedule_v1.RepeatedScheduleResource{
				Name:           "Test RepeatedSchedule 1",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				Relation: &schedule_v1.RepeatedScheduleResource_TargetSite{
					TargetSite: site,
				},
				DurationSeconds: uint32(2),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
		},
	}
	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	crepeatedScheduleResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	repeatedSchedResID := inv_testing.GetResourceIDOrFail(t, crepeatedScheduleResp)
	t.Cleanup(func() { inv_testing.DeleteResource(t, repeatedSchedResID) })

	testcases := map[string]UpdateTCRSched{
		"Update1": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:           "Updated Name 1",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				Relation: &schedule_v1.RepeatedScheduleResource_TargetSite{
					TargetSite: site,
				},
				DurationSeconds: uint32(4),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					rsr.FieldName, rsr.FieldScheduleStatus, rsr.EdgeTargetSite, rsr.FieldDurationSeconds,
					rsr.FieldCronMinutes, rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth, rsr.FieldCronDayWeek,
				},
			},
			valid: true,
		},
		"BadUpdate1": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:         "Updated Name 2",
				CronMinutes:  "3",
				CronHours:    "4",
				CronDayMonth: "5",
				CronMonth:    "6",
				CronDayWeek:  "0",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetHost{
					TargetHost: host,
				},
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					rsr.EdgeTargetHost, rsr.FieldName, rsr.FieldCronMinutes,
					rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth, rsr.FieldCronDayWeek,
				},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
	}
	runUpdateTCForRSched(t, testcases, repeatedSchedResID)
}

// Start with no target; target site; target workload; reset target; target host;
// NOTE tc are independent by the order of exec.
func Test_UpdateRepeatedSchedule5(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site := inv_testing.CreateSite(t, nil, nil)
	host := inv_testing.CreateHost(t, site, nil)
	workload := inv_testing.CreateWorkload(t)

	// create RepeatedSchedule to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Repeatedschedule{
			Repeatedschedule: &schedule_v1.RepeatedScheduleResource{
				Name:            "Test RepeatedSchedule 1",
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
		},
	}
	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	crepeatedScheduleResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	repeatedSchedResID := inv_testing.GetResourceIDOrFail(t, crepeatedScheduleResp)
	t.Cleanup(func() { inv_testing.DeleteResource(t, repeatedSchedResID) })

	testcases := map[string]UpdateTCRSched{
		"Switch Target (nil->site)": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:           "Switch Target (nil->site)",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				Relation: &schedule_v1.RepeatedScheduleResource_TargetSite{
					TargetSite: site,
				},
				DurationSeconds: uint32(2),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			fieldMask: &fieldmaskpb.FieldMask{Paths: []string{
				rsr.FieldName, rsr.FieldScheduleStatus, rsr.EdgeTargetSite, rsr.EdgeTargetHost, rsr.EdgeTargetWorkload,
				rsr.FieldDurationSeconds, rsr.FieldCronMinutes, rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth,
				rsr.FieldCronDayWeek, rsr.EdgeTargetRegion,
			}},
			valid: true,
		},
		"Switch Target (site->host)": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:           "Switch Target (site->host)",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				Relation: &schedule_v1.RepeatedScheduleResource_TargetHost{
					TargetHost: host,
				},
				DurationSeconds: uint32(4),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			fieldMask: &fieldmaskpb.FieldMask{Paths: []string{
				rsr.FieldName, rsr.FieldScheduleStatus, rsr.EdgeTargetSite, rsr.EdgeTargetHost, rsr.EdgeTargetWorkload,
				rsr.FieldDurationSeconds, rsr.FieldCronMinutes, rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth,
				rsr.FieldCronDayWeek, rsr.EdgeTargetRegion,
			}},
			valid: true,
		},
		"Switch Target (host->workload)": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:           "Switch Target (host->workload)",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				Relation: &schedule_v1.RepeatedScheduleResource_TargetWorkload{
					TargetWorkload: workload,
				},
				DurationSeconds: uint32(4),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			fieldMask: &fieldmaskpb.FieldMask{Paths: []string{
				rsr.FieldName, rsr.FieldScheduleStatus, rsr.EdgeTargetSite, rsr.EdgeTargetHost, rsr.EdgeTargetWorkload,
				rsr.FieldDurationSeconds, rsr.FieldCronMinutes, rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth,
				rsr.FieldCronDayWeek, rsr.EdgeTargetRegion,
			}},
			valid: true,
		},
		"Switch Target (workload->region)": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:           "Switch Target (workload->region)",
				ScheduleStatus: schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				Relation: &schedule_v1.RepeatedScheduleResource_TargetRegion{
					TargetRegion: region,
				},
				DurationSeconds: uint32(4),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			fieldMask: &fieldmaskpb.FieldMask{Paths: []string{
				rsr.FieldName, rsr.FieldScheduleStatus, rsr.EdgeTargetSite, rsr.EdgeTargetHost, rsr.EdgeTargetWorkload,
				rsr.FieldDurationSeconds, rsr.FieldCronMinutes, rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth,
				rsr.FieldCronDayWeek, rsr.EdgeTargetRegion,
			}},
			valid: true,
		},
		"Reset Target": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:            "Reset Target",
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(4),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			fieldMask: &fieldmaskpb.FieldMask{Paths: []string{
				rsr.FieldName, rsr.FieldScheduleStatus, rsr.FieldDurationSeconds,
				rsr.FieldCronMinutes, rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth, rsr.FieldCronDayWeek,
			}},
			valid: true,
		},
	}
	runUpdateTCForRSched(t, testcases, repeatedSchedResID)
}

// Start with no target; target workload; reset workload with put;
// NOTE tc are independent by the order of exec.
func Test_UpdateRepeatedSchedule6(t *testing.T) {
	workload := inv_testing.CreateWorkload(t)

	// create RepeatedSchedule to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Repeatedschedule{
			Repeatedschedule: &schedule_v1.RepeatedScheduleResource{
				Name:            "Test RepeatedSchedule 1",
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
		},
	}
	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	crepeatedScheduleResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	repeatedSchedResID := inv_testing.GetResourceIDOrFail(t, crepeatedScheduleResp)
	t.Cleanup(func() { inv_testing.DeleteResource(t, repeatedSchedResID) })

	testcases := map[string]UpdateTCRSched{
		"Update1": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:         "Updated Name 1",
				CronMinutes:  "3",
				CronHours:    "4",
				CronDayMonth: "5",
				CronMonth:    "6",
				CronDayWeek:  "0",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetWorkload{
					TargetWorkload: workload,
				},
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					rsr.EdgeTargetWorkload, rsr.FieldName, rsr.FieldCronMinutes,
					rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth, rsr.FieldCronDayWeek,
				},
			},
			valid: true,
		},
		"Update2": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:            "Updated Name 2",
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(4),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			fieldMask: &fieldmaskpb.FieldMask{Paths: []string{
				rsr.FieldName, rsr.FieldScheduleStatus, rsr.FieldDurationSeconds,
				rsr.FieldCronMinutes, rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth, rsr.FieldCronDayWeek,
			}},
			valid: true,
		},
	}
	runUpdateTCForRSched(t, testcases, repeatedSchedResID)
}

// Start with no target; target region; reset region with put;
// NOTE tc are independent by the order of exec.
func Test_UpdateRepeatedSchedule7(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)

	// create RepeatedSchedule to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Repeatedschedule{
			Repeatedschedule: &schedule_v1.RepeatedScheduleResource{
				Name:            "Test RepeatedSchedule 1",
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
		},
	}
	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	crepeatedScheduleResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	repeatedSchedResID := inv_testing.GetResourceIDOrFail(t, crepeatedScheduleResp)
	t.Cleanup(func() { inv_testing.DeleteResource(t, repeatedSchedResID) })

	testcases := map[string]UpdateTCRSched{
		"SetTargetRegion": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:         "Set Target Region",
				CronMinutes:  "3",
				CronHours:    "4",
				CronDayMonth: "5",
				CronMonth:    "6",
				CronDayWeek:  "0",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetRegion{
					TargetRegion: region,
				},
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					rsr.EdgeTargetRegion, rsr.FieldName, rsr.FieldCronMinutes,
					rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth, rsr.FieldCronDayWeek,
				},
			},
			valid: true,
		},
		"ResetRegion": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:            "ResetRegion",
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(4),
				CronMinutes:     "3",
				CronHours:       "4",
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					rsr.FieldName, rsr.FieldScheduleStatus, rsr.FieldCronMinutes,
					rsr.FieldCronHours, rsr.FieldCronDayMonth, rsr.FieldCronMonth, rsr.FieldCronDayWeek,
				},
			},
			valid: true,
		},
	}
	runUpdateTCForRSched(t, testcases, repeatedSchedResID)
}

func repeatedSchedEdgesOnlyResourceID(expected *schedule_v1.RepeatedScheduleResource) {
	fmt.Println("Expected before conversion: ", expected)
	if expected.GetTargetHost() != nil {
		expected.Relation = &schedule_v1.RepeatedScheduleResource_TargetHost{
			TargetHost: &computev1.HostResource{ResourceId: expected.GetTargetHost().ResourceId},
		}
	}
	if expected.GetTargetSite() != nil {
		expected.Relation = &schedule_v1.RepeatedScheduleResource_TargetSite{
			TargetSite: &location_v1.SiteResource{ResourceId: expected.GetTargetSite().ResourceId},
		}
	}
	if expected.GetTargetWorkload() != nil {
		expected.Relation = &schedule_v1.RepeatedScheduleResource_TargetWorkload{
			TargetWorkload: &computev1.WorkloadResource{ResourceId: expected.GetTargetWorkload().ResourceId},
		}
	}
}

func Test_FilterNestedRepeatedSchedules(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site := inv_testing.CreateSite(t, region, nil)
	host := inv_testing.CreateHost(t, site, nil)
	// Setting again the edge nilled by the helper
	repeatedSched1 := inv_testing.CreateRepeatedSchedule(t, host, nil, schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE)
	repeatedSched1.Relation = &schedule_v1.RepeatedScheduleResource_TargetHost{
		TargetHost: host,
	}
	repeatedSched2 := inv_testing.CreateRepeatedSchedule(t, nil, site, schedule_v1.ScheduleStatus_SCHEDULE_STATUS_SHIPPING)
	repeatedSched2.Relation = &schedule_v1.RepeatedScheduleResource_TargetSite{
		TargetSite: site,
	}

	testcases := map[string]struct {
		in                *inv_v1.ResourceFilter
		resources         []*schedule_v1.RepeatedScheduleResource
		valid             bool
		expectedCodeError codes.Code
	}{
		"FilterByHostSiteID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s = %q`, rsr.EdgeTargetHost, hostresource.EdgeSite,
					siteresource.FieldResourceID, site.GetResourceId()),
			},
			resources: []*schedule_v1.RepeatedScheduleResource{repeatedSched1},
			valid:     true,
		},
		"FilterBySiteRegionID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s = %q`, rsr.EdgeTargetSite, siteresource.EdgeRegion,
					regionresource.FieldResourceID, region.GetResourceId()),
			},
			resources: []*schedule_v1.RepeatedScheduleResource{repeatedSched2},
			valid:     true,
		},
		"FailTooDeep": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s.%s.%s.%s = %q`, rsr.EdgeTargetSite, siteresource.EdgeRegion,
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

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Repeatedschedule{}} // Set the resource kind

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
						"FilterRepeatedSchedules() failed - want: %s, got: %s",
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

				resources := make([]*schedule_v1.RepeatedScheduleResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetRepeatedschedule())
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

func Test_Validate_RepeatedSchedule(t *testing.T) {
	site := inv_testing.CreateSite(t, nil, nil)
	host := inv_testing.CreateHost(t, site, nil)

	testcases := map[string]struct {
		in    *schedule_v1.RepeatedScheduleResource
		valid bool
	}{
		"ValidCronFieldsRepeatedSchedule1": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:            "Test RepeatedSchedule 1",
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "*",
				CronHours:       "*",
				CronDayMonth:    "*",
				CronMonth:       "*",
				CronDayWeek:     "*",
			},
			valid: true,
		},
		"ValidCronFieldsRepeatedSchedule2": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:            "Test RepeatedSchedule 2",
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "5",
				CronHours:       "1,2,3", // specific time intervals
				CronDayMonth:    "*",
				CronMonth:       "*",
				CronDayWeek:     "*",
			},
			valid: true,
		},
		"ValidCronFieldsRepeatedSchedule3": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:            "Test RepeatedSchedule 3",
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "0",
				CronHours:       "0",
				CronDayMonth:    "1",
				CronMonth:       "*",
				CronDayWeek:     "*",
			},
			valid: true,
		},
		"ValidCronFieldsRepeatedSchedule4": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:            "Test RepeatedSchedule 4",
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "0",
				CronHours:       "0",
				CronDayMonth:    "*",
				CronMonth:       "0",
				CronDayWeek:     "0",
			},
			valid: true,
		},
		"InValidCronFieldsRepeatedSchedule": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:            "Test RepeatedSchedule 5",
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "/5", // Every x mins interval not supported
				CronHours:       "*",
				CronDayMonth:    "*",
				CronMonth:       "*",
				CronDayWeek:     "*",
			},
			valid: false,
		},
		"InValidCronFieldsRepeatedSchedule1": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:            "Test RepeatedSchedule 6",
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "5-", // invalid
				CronHours:       "*",
				CronDayMonth:    "*",
				CronMonth:       "*",
				CronDayWeek:     "*",
			},
			valid: false,
		},
		"InValidCronFieldsRepeatedSchedule2": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name:            "Test RepeatedSchedule 7",
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "5",
				CronHours:       "2-3", // From specific duration not suppoerted
				CronDayMonth:    "*",
				CronMonth:       "*",
				CronDayWeek:     "*",
			},
			valid: false,
		},
		"InvalidCronMinFieldsRepeatedSchedule": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 8",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetHost{
					TargetHost: host,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "60", // invalid
				CronHours:       "2",
				CronDayMonth:    "*",
				CronMonth:       "*",
				CronDayWeek:     "*",
			},
			valid: false,
		},
		"InvalidCronHourFieldsRepeatedSchedule2": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 9",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetSite{
					TargetSite: site,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "3",
				CronHours:       "25", // invalid
				CronDayMonth:    "5",
				CronMonth:       "6",
				CronDayWeek:     "0",
			},
			valid: false,
		},
		"InvalidCronDomFieldsRepeatedSchedule": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 10",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetHost{
					TargetHost: host,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "59",
				CronHours:       "24",
				CronDayMonth:    "32", // invalid
				CronMonth:       "*",
				CronDayWeek:     "*",
			},
			valid: false,
		},
		"InvalidCronMonthFieldsRepeatedSchedule": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 11",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetHost{
					TargetHost: host,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "59",
				CronHours:       "24",
				CronDayMonth:    "31",
				CronMonth:       "13", // invalid
				CronDayWeek:     "*",
			},
			valid: false,
		},
		"InvalidCrondayweekFieldsRepeatedSchedule": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 12",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetHost{
					TargetHost: host,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(2),
				CronMinutes:     "59",
				CronHours:       "24",
				CronDayMonth:    "*",
				CronMonth:       "*",
				CronDayWeek:     "7", // invalid
			},
			valid: false,
		},
		"CreateMissingCronValues": {
			in: &schedule_v1.RepeatedScheduleResource{
				Name: "Test RepeatedSchedule 13",
				Relation: &schedule_v1.RepeatedScheduleResource_TargetSite{
					TargetSite: site,
				},
				ScheduleStatus:  schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
				DurationSeconds: uint32(1),
			},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Repeatedschedule{Repeatedschedule: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			// Create RepeatedSchedule to validate cron fields
			rsp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
			if err != nil {
				if !tc.valid {
					assert.Error(t, err)
				}
				return
			}
			t.Cleanup(func() {
				inv_testing.DeleteResource(t, inv_testing.GetResourceIDOrFail(t, rsp))
			})
		})
	}
}

func Test_RepeatedScheduleEnumStatusMap(t *testing.T) {
	v, err := store.RepeatedScheduleEnumStatusMap("invalid_input",
		int32(schedule_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE))
	assert.Error(t, err)
	assert.Nil(t, v)
}

func TestRepeatedScheduleMTSanity(t *testing.T) {
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	suite.Run(t, &struct{ mt }{
		mt: mt{
			createResource: func(tenantID string) (string, *inv_v1.Resource) {
				rs := dao.CreateRepeatedSchedule(t, tenantID)
				res, err := util.WrapResource(rs)
				require.NoError(t, err)
				return rs.GetResourceId(), res
			},
		},
	})
}

func TestDeleteResources_RepeatedSchedules(t *testing.T) {
	suite.Run(t, &struct{ hardDeleteAllResourcesSuite }{
		hardDeleteAllResourcesSuite: hardDeleteAllResourcesSuite{
			createModel: func(dao *inv_testing.InvResourceDAO) (string, int) {
				tenantID := uuid.NewString()
				region := dao.CreateRegion(t, tenantID)
				site := dao.CreateSite(t, tenantID)
				host := dao.CreateHost(t, tenantID)
				workload := dao.CreateWorkload(t, tenantID)
				return tenantID, len([]any{
					dao.CreateRepeatedScheduleNoCleanup(t, tenantID, inv_testing.RSRRegion(region)),
					dao.CreateRepeatedScheduleNoCleanup(t, tenantID, inv_testing.RSRTargetSite(site)),
					dao.CreateRepeatedScheduleNoCleanup(t, tenantID, inv_testing.RSRTargetHost(host)),
					dao.CreateRepeatedScheduleNoCleanup(t, tenantID, inv_testing.RSRTargetWorkload(workload)),
				})
			},
			resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_REPEATEDSCHEDULE,
		},
	})
}
