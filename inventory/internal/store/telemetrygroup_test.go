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
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	telemetryres "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/telemetrygroupresource"
	telemetryprofileres "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/telemetryprofile"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/store"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	telemetry_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/telemetry/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

const (
	TestTelemetryName  = "Telemetry Name"
	TestMetricInterval = 300
	TestMetricLatency  = 10
)

var TestMetricGroups = []string{
	"cpu",
	"memory",
}

var TestLogGroups = []string{
	"kmseg",
	"syslog",
}

func Test_Create_Get_Delete_TelemetryGroupResource(t *testing.T) {
	testcases := map[string]struct {
		in    *telemetry_v1.TelemetryGroupResource
		valid bool
	}{
		"CreateGoodTelemetryMetrics": {
			in: &telemetry_v1.TelemetryGroupResource{
				Name:          TestTelemetryName,
				Kind:          telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
				CollectorKind: telemetry_v1.CollectorKind_COLLECTOR_KIND_HOST,
				Groups:        TestMetricGroups,
			},
			valid: true,
		},
		"CreateGoodTelemetryLogs": {
			in: &telemetry_v1.TelemetryGroupResource{
				Name:          "Telemetry2",
				Kind:          telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS,
				CollectorKind: telemetry_v1.CollectorKind_COLLECTOR_KIND_HOST,
				Groups:        TestLogGroups,
			},
			valid: true,
		},
		"CreateBadTelemetryWithResourceIdSet": {
			in: &telemetry_v1.TelemetryGroupResource{
				ResourceId:    "telemetrygroup-12345678",
				Name:          TestTelemetryName,
				Kind:          telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
				CollectorKind: telemetry_v1.CollectorKind_COLLECTOR_KIND_HOST,
				Groups:        TestMetricGroups,
			},
			valid: false,
		},
		"CreateBadTelemetryWithBadResourceId": {
			in: &telemetry_v1.TelemetryGroupResource{
				ResourceId:    "xyz-12345678",
				Name:          TestTelemetryName,
				Kind:          telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
				CollectorKind: telemetry_v1.CollectorKind_COLLECTOR_KIND_HOST,
				Groups:        TestMetricGroups,
			},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_TelemetryGroup{TelemetryGroup: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			resp, err := inv_testing.TestClients[inv_testing.APIClient].
				Create(ctx, createresreq)
			telgroupResID := resp.GetTelemetryGroup().GetResourceId()

			if err != nil {
				if tc.valid {
					t.Errorf("CreateTelemetryGroup() failed: %s", err)
				}
			} else {
				tc.in.ResourceId = telgroupResID // Update with created resource ID.
				tc.in.CreatedAt = resp.GetTelemetryGroup().GetCreatedAt()
				tc.in.UpdatedAt = resp.GetTelemetryGroup().GetUpdatedAt()
				assertSameResource(t, createresreq, resp, nil)
				if !tc.valid {
					t.Errorf("CreateTelemetryGroup() succeeded but should have failed")
				}
			}

			if !t.Failed() && tc.valid {
				getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, telgroupResID)
				require.NoError(t, err, "GetTelemetryGroup() failed")

				if eq, diff := inv_testing.ProtoEqualOrDiff(tc.in, getresp.GetResource().GetTelemetryGroup()); !eq {
					t.Errorf("GetTelemetryGroup() data not equal: %v", diff)
				}

				updateresreq := &inv_v1.Resource{
					Resource: &inv_v1.Resource_TelemetryGroup{
						TelemetryGroup: &telemetry_v1.TelemetryGroupResource{
							Name: "Updated Name",
						},
					},
				}

				fm := &fieldmaskpb.FieldMask{Paths: []string{telemetryres.FieldName}}
				// update non-existent first
				upRes, err := inv_testing.TestClients[inv_testing.APIClient].Update(ctx,
					"telemetrygroup-12345678",
					fm,
					updateresreq)
				require.Error(t, err)
				assert.Nil(t, upRes)

				upRes, err = inv_testing.TestClients[inv_testing.APIClient].Update(
					ctx,
					telgroupResID,
					fm,
					updateresreq,
				)
				if err != nil {
					t.Errorf("UpdateTelemetryGroup() failed: %s", err)
				}

				// Validate returned resource
				assertSameResource(t, updateresreq, upRes, fm)

				// delete non-existent first
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, "telemetrygroup-12345678")
				require.Error(t, err)

				// delete
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(
					ctx,
					telgroupResID,
				)
				if err != nil {
					t.Errorf("DeleteTelemetryGroup() failed %s", err)
				}

				_, err = inv_testing.TestClients[inv_testing.APIClient].Get(ctx, telgroupResID)
				require.Error(t, err, "Failure - telemetry was not deleted, but should be deleted")
			}
		})
	}
}

func Test_FilterTelemetryGroupResource(t *testing.T) {
	telemetry1 := inv_testing.CreateTelemetryGroupMetrics(t, true)
	telemetry2 := inv_testing.CreateTelemetryGroupLogs(t, true)

	testcases := map[string]struct {
		in        *inv_v1.ResourceFilter
		resources []*telemetry_v1.TelemetryGroupResource
		valid     bool
	}{
		"NoFilter": {
			in:        &inv_v1.ResourceFilter{},
			resources: []*telemetry_v1.TelemetryGroupResource{telemetry1, telemetry2},
			valid:     true,
		},
		"NoFilterOrderByResourceId": {
			in: &inv_v1.ResourceFilter{
				OrderBy: telemetryres.FieldResourceID,
			},
			resources: []*telemetry_v1.TelemetryGroupResource{telemetry1, telemetry2},
			valid:     true,
		},
		"FilterByResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, telemetryres.FieldResourceID, telemetry1.ResourceId),
			},
			resources: []*telemetry_v1.TelemetryGroupResource{telemetry1},
			valid:     true,
		},
		"FilterMetricGroup": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, telemetryres.FieldGroups, telemetry1.Groups[0]),
			},
			resources: []*telemetry_v1.TelemetryGroupResource{telemetry1},
			valid:     true,
		},
		"FilterLogGroup": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, telemetryres.FieldGroups, telemetry2.Groups[0]),
			},
			resources: []*telemetry_v1.TelemetryGroupResource{telemetry2},
			valid:     true,
		},
		"FilterByHasProfiles": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s)`, telemetryres.EdgeProfiles),
			},
			resources: []*telemetry_v1.TelemetryGroupResource{},
			valid:     true,
		},
		"FilterByHasProfilesHasGroup": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s.%s)`, telemetryres.EdgeProfiles, telemetryprofileres.EdgeGroup),
			},
			resources: []*telemetry_v1.TelemetryGroupResource{},
			valid:     true,
		},
		"FilterLimit": {
			in: &inv_v1.ResourceFilter{
				Offset: 0,
				Limit:  2,
			},
			resources: []*telemetry_v1.TelemetryGroupResource{telemetry1, telemetry2},
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
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_TelemetryGroup{}} // Set the resource kind
			findres, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx, tc.in)
			if err != nil {
				if tc.valid {
					t.Errorf("FilterTelemetryGroup() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("FilterTelemetryGroup() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				resIDs := inv_testing.GetSortedResourceIDSlice(tc.resources)
				inv_testing.SortHasResourceIDAndTenantID(findres.Resources)

				if !reflect.DeepEqual(resIDs, findres.Resources) {
					t.Errorf(
						"FilterTelemetryGroup() failed - want: %s, got: %s",
						resIDs,
						findres.Resources,
					)
				}
			}

			listres, err := inv_testing.TestClients[inv_testing.APIClient].List(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("ListTelemetryGroup() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("ListTelemetryGroup() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				resources := make([]*telemetry_v1.TelemetryGroupResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetTelemetryGroup())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, resources[i]); !eq {
						t.Errorf("ListTelemetryGroup() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

func Test_TelemetryGroupEnumStatusMap(t *testing.T) {
	v, err := store.TelemetryGroupEnumStatusMap("invalid_input",
		int32(telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS))
	assert.Error(t, err)
	assert.Nil(t, v)
}

func TestTelemetryGroupLogsMTSanity(t *testing.T) {
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	suite.Run(t, &struct{ mt }{
		mt: mt{
			createResource: func(tenantID string) (string, *inv_v1.Resource) {
				tgl := dao.CreateTelemetryGroupLogs(t, tenantID, true)
				res, err := util.WrapResource(tgl)
				require.NoError(t, err)
				return tgl.GetResourceId(), res
			},
		},
	})
}

func TestTelemetryGroupMetricsMTSanity(t *testing.T) {
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	suite.Run(t, &struct{ mt }{
		mt: mt{
			createResource: func(tenantID string) (string, *inv_v1.Resource) {
				tgm := dao.CreateTelemetryGroupMetrics(t, tenantID, true)
				res, err := util.WrapResource(tgm)
				require.NoError(t, err)
				return tgm.GetResourceId(), res
			},
		},
	})
}

func TestDeleteResources_TelemetryGroups(t *testing.T) {
	suite.Run(t, &struct{ hardDeleteAllResourcesSuite }{
		hardDeleteAllResourcesSuite: hardDeleteAllResourcesSuite{
			createModel: func(dao *inv_testing.InvResourceDAO) (string, int) {
				tenantID := uuid.NewString()
				return tenantID, len([]any{
					dao.CreateTelemetryGroupLogs(t, tenantID, false),
					dao.CreateTelemetryGroupMetrics(t, tenantID, false),
				})
			},
			resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_TELEMETRY_GROUP,
		},
	})
}
