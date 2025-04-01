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
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/instanceresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/regionresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/siteresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/telemetrygroupresource"
	telemetryprofileres "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/telemetryprofile"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/workloadmember"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/store"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	location_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/location/v1"
	telemetry_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/telemetry/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

func Test_Create_Get_Delete_TelemetryProfile(t *testing.T) {
	parentRegion := inv_testing.CreateRegion(t, nil)
	region := inv_testing.CreateRegion(t, parentRegion)
	site := inv_testing.CreateSite(t, region, nil)
	os := inv_testing.CreateOs(t)
	h := inv_testing.CreateHost(t, site, nil)
	inst := inv_testing.CreateInstance(t, h, os)
	metricsGroup := inv_testing.CreateTelemetryGroupMetrics(t, true)
	logsGroup := inv_testing.CreateTelemetryGroupLogs(t, true)

	testcases := map[string]struct {
		in    *telemetry_v1.TelemetryProfile
		valid bool
	}{
		"CreateGoodTelemetryProfilePerInstance": {
			in: &telemetry_v1.TelemetryProfile{
				Relation: &telemetry_v1.TelemetryProfile_Instance{
					Instance: inst,
				},
				Group:           metricsGroup,
				Kind:            telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
				MetricsInterval: TestMetricInterval,
			},
			valid: true,
		},
		"CreateGoodTelemetryProfilePerSite": {
			in: &telemetry_v1.TelemetryProfile{
				Relation: &telemetry_v1.TelemetryProfile_Site{
					Site: site,
				},
				Group:           metricsGroup,
				Kind:            telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
				MetricsInterval: TestMetricInterval,
			},
			valid: true,
		},
		"CreateGoodTelemetryProfilePerRegion": {
			in: &telemetry_v1.TelemetryProfile{
				Relation: &telemetry_v1.TelemetryProfile_Region{
					Region: region,
				},
				Group:           metricsGroup,
				Kind:            telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
				MetricsInterval: TestMetricInterval,
			},
			valid: true,
		},
		"CreateGoodTelemetryProfilePerParentRegion": {
			in: &telemetry_v1.TelemetryProfile{
				Relation: &telemetry_v1.TelemetryProfile_Region{
					Region: parentRegion,
				},
				Group:    logsGroup,
				Kind:     telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS,
				LogLevel: telemetry_v1.SeverityLevel_SEVERITY_LEVEL_INFO,
			},
			valid: true,
		},
		"CreateBadEmptyTelemetryProfile": {
			in:    &telemetry_v1.TelemetryProfile{},
			valid: false,
		},
		"CreateBadTelemetryProfileWithResourceIdSet": {
			// This tests case verifies that create requests with a resource ID
			// already set are rejected.
			in: &telemetry_v1.TelemetryProfile{
				ResourceId: "telemetryprofile-12345678",
				Kind:       telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS,
				LogLevel:   telemetry_v1.SeverityLevel_SEVERITY_LEVEL_INFO,
				Relation: &telemetry_v1.TelemetryProfile_Instance{
					Instance: inst,
				},
			},
			valid: false,
		},
		"CreateBadTelemetryProfileWithInvalidResourceIdSet": {
			// This tests case verifies that create requests with a resource ID
			// already set are rejected.
			in: &telemetry_v1.TelemetryProfile{
				ResourceId: "telemetryp-12345678",
				Kind:       telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS,
				LogLevel:   telemetry_v1.SeverityLevel_SEVERITY_LEVEL_INFO,
			},
			valid: false,
		},
		"CreateBadTelemetryProfileInvalidMetricsInterval": {
			in: &telemetry_v1.TelemetryProfile{
				Relation: &telemetry_v1.TelemetryProfile_Instance{
					Instance: inst,
				},
				Group:           metricsGroup,
				Kind:            telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
				MetricsInterval: 0,
			},
			valid: false,
		},
		"CreateBadTelemetryProfileInvalidLogSeverity": {
			in: &telemetry_v1.TelemetryProfile{
				Relation: &telemetry_v1.TelemetryProfile_Region{
					Region: parentRegion,
				},
				Group:    logsGroup,
				Kind:     telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS,
				LogLevel: telemetry_v1.SeverityLevel_SEVERITY_LEVEL_UNSPECIFIED,
			},
			valid: false,
		},
		"CreateBadTelemetryProfileInvalidResourceKind": {
			in: &telemetry_v1.TelemetryProfile{
				Relation: &telemetry_v1.TelemetryProfile_Region{
					Region: parentRegion,
				},
				Group:    logsGroup,
				Kind:     telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_UNSPECIFIED,
				LogLevel: telemetry_v1.SeverityLevel_SEVERITY_LEVEL_UNSPECIFIED,
			},
			valid: false,
		},
		"CreateBadTelemetryProfileWithNonExistingInstance": {
			in: &telemetry_v1.TelemetryProfile{
				Relation: &telemetry_v1.TelemetryProfile_Instance{
					Instance: &computev1.InstanceResource{
						ResourceId: "inst-12345678",
					},
				},
				Kind:     telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS,
				LogLevel: telemetry_v1.SeverityLevel_SEVERITY_LEVEL_INFO,
			},
			valid: false,
		},
		"CreateBadTelemetryProfileWithNonExistingSite": {
			in: &telemetry_v1.TelemetryProfile{
				Relation: &telemetry_v1.TelemetryProfile_Site{
					Site: &location_v1.SiteResource{
						ResourceId: "site-12345678",
					},
				},
				Kind:     telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS,
				LogLevel: telemetry_v1.SeverityLevel_SEVERITY_LEVEL_INFO,
			},
			valid: false,
		},
		"CreateBadTelemetryProfileWithNonExistingRegion": {
			in: &telemetry_v1.TelemetryProfile{
				Relation: &telemetry_v1.TelemetryProfile_Region{
					Region: &location_v1.RegionResource{
						ResourceId: "region-12345678",
					},
				},
				Kind:     telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS,
				LogLevel: telemetry_v1.SeverityLevel_SEVERITY_LEVEL_INFO,
			},
			valid: false,
		},
		"CreateBadTelemetryProfileWithNonExistingGroup": {
			in: &telemetry_v1.TelemetryProfile{
				Relation: &telemetry_v1.TelemetryProfile_Region{
					Region: parentRegion,
				},
				Group: &telemetry_v1.TelemetryGroupResource{
					ResourceId: "telemetrygroup-12345678",
				},
				Kind:     telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS,
				LogLevel: telemetry_v1.SeverityLevel_SEVERITY_LEVEL_INFO,
			},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_TelemetryProfile{TelemetryProfile: tc.in},
			}
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			resp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
			telprofileResID := resp.GetTelemetryProfile().GetResourceId()

			if err != nil {
				if tc.valid {
					t.Errorf("CreateTelemetryProfile() failed: %s", err)
				}
			} else {
				tc.in.ResourceId = telprofileResID // Update with created resource ID.
				tc.in.CreatedAt = resp.GetTelemetryProfile().GetCreatedAt()
				tc.in.UpdatedAt = resp.GetTelemetryProfile().GetUpdatedAt()
				assertSameResource(t, createresreq, resp, nil)
				if !tc.valid {
					t.Errorf("CreateTelemetryProfile() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				// get
				getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, telprofileResID)
				if err != nil {
					require.NoError(t, err, "GetTelemetryProfile() failed")
				}

				// verify data
				if eq, diff := inv_testing.ProtoEqualOrDiff(tc.in, getresp.GetResource().GetTelemetryProfile()); !eq {
					t.Errorf("GetTelemetryProfile() data not equal: %v", diff)
				}

				// delete non-existent first
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, "telemetryprofile-12345678")
				require.Error(t, err)

				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(
					ctx,
					telprofileResID,
				)
				if err != nil {
					t.Errorf("DeleteTelemetryProfile() failed %s", err)
				}

				_, err = inv_testing.TestClients[inv_testing.APIClient].Get(ctx, telprofileResID)
				if err == nil {
					t.Errorf("Failure - TelemetryProfile was not deleted, but should be deleted")
				}
			}
		})
	}
}

func Test_UpdateTelemetryProfile(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site := inv_testing.CreateSite(t, region, nil)
	os := inv_testing.CreateOs(t)
	host := inv_testing.CreateHost(t, site, nil)
	inst := inv_testing.CreateInstance(t, host, os)
	metricsGroup := inv_testing.CreateTelemetryGroupMetrics(t, true)
	logsGroup := inv_testing.CreateTelemetryGroupLogs(t, true)

	profile1 := inv_testing.CreateTelemetryProfile(t, inst, nil, nil, metricsGroup, true)
	profile2 := inv_testing.CreateTelemetryProfile(t, nil, site, nil, metricsGroup, true)
	profile3 := inv_testing.CreateTelemetryProfile(t, nil, nil, region, metricsGroup, true)

	testcases := map[string]struct {
		in           *telemetry_v1.TelemetryProfile
		resourceID   string
		fieldMask    *fieldmaskpb.FieldMask
		valid        bool
		expErrorCode codes.Code
	}{
		"UpdateMetricsInterval": {
			in: &telemetry_v1.TelemetryProfile{
				MetricsInterval: 1,
			},
			resourceID: profile1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{telemetryprofileres.FieldMetricsInterval},
			},
			valid: true,
		},
		"UpdateLogLevel": {
			in: &telemetry_v1.TelemetryProfile{
				LogLevel: telemetry_v1.SeverityLevel_SEVERITY_LEVEL_INFO,
			},
			resourceID: profile1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{telemetryprofileres.FieldLogLevel},
			},
			valid: true,
		},
		"UpdateRelation1": {
			in: &telemetry_v1.TelemetryProfile{
				Relation: &telemetry_v1.TelemetryProfile_Region{
					Region: region,
				},
			},
			resourceID: profile1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{telemetryprofileres.EdgeRegion},
			},
			valid: true,
		},
		"UpdateRelation2": {
			in: &telemetry_v1.TelemetryProfile{
				Relation: &telemetry_v1.TelemetryProfile_Instance{
					Instance: inst,
				},
			},
			resourceID: profile2.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{telemetryprofileres.EdgeInstance},
			},
			valid: true,
		},
		"UpdateRelation3": {
			in: &telemetry_v1.TelemetryProfile{
				Relation: &telemetry_v1.TelemetryProfile_Region{
					Region: region,
				},
			},
			resourceID: profile3.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{telemetryprofileres.EdgeRegion},
			},
			valid: true,
		},
		"UpdateGroupFail": {
			in: &telemetry_v1.TelemetryProfile{
				Group: logsGroup,
			},
			resourceID: profile1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{telemetryprofileres.EdgeGroup},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateResourceIDNotFound": {
			in: &telemetry_v1.TelemetryProfile{
				MetricsInterval: 1,
			},
			resourceID: "telemetryprofile-12345678",
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{telemetryprofileres.FieldMetricsInterval},
			},
			valid:        false,
			expErrorCode: codes.NotFound,
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			updateresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_TelemetryProfile{TelemetryProfile: tc.in},
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

func Test_FilterTelemetryProfile(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site1 := inv_testing.CreateSite(t, region, nil)
	site2 := inv_testing.CreateSite(t, region, nil)
	os := inv_testing.CreateOs(t)
	h1 := inv_testing.CreateHost(t, site1, nil)
	h2 := inv_testing.CreateHost(t, site1, nil)
	inst1 := inv_testing.CreateInstance(t, h1, os)
	inst2 := inv_testing.CreateInstance(t, h2, os)

	metricsGroup := inv_testing.CreateTelemetryGroupMetrics(t, true)

	profilePerInstance1 := inv_testing.CreateTelemetryProfile(t, inst1, nil, nil, metricsGroup, true)
	profilePerInstance1.Relation = &telemetry_v1.TelemetryProfile_Instance{Instance: inst1}
	profilePerInstance1.Group = metricsGroup
	profilePerInstance2 := inv_testing.CreateTelemetryProfile(t, inst1, nil, nil, metricsGroup, true)
	profilePerInstance2.Relation = &telemetry_v1.TelemetryProfile_Instance{Instance: inst1}
	profilePerInstance2.Group = metricsGroup
	profilePerInstance3 := inv_testing.CreateTelemetryProfile(t, inst2, nil, nil, metricsGroup, true)
	profilePerInstance3.Relation = &telemetry_v1.TelemetryProfile_Instance{Instance: inst2}
	profilePerInstance3.Group = metricsGroup

	profilePerSite1 := inv_testing.CreateTelemetryProfile(t, nil, site1, nil, metricsGroup, true)
	profilePerSite1.Relation = &telemetry_v1.TelemetryProfile_Site{Site: site1}
	profilePerSite1.Group = metricsGroup
	profilePerSite2 := inv_testing.CreateTelemetryProfile(t, nil, site1, nil, metricsGroup, true)
	profilePerSite2.Relation = &telemetry_v1.TelemetryProfile_Site{Site: site1}
	profilePerSite2.Group = metricsGroup
	profilePerSite3 := inv_testing.CreateTelemetryProfile(t, nil, site2, nil, metricsGroup, true)
	profilePerSite3.Relation = &telemetry_v1.TelemetryProfile_Site{Site: site2}
	profilePerSite3.Group = metricsGroup

	profilePerRegion1 := inv_testing.CreateTelemetryProfile(t, nil, nil, region, metricsGroup, true)
	profilePerRegion1.Relation = &telemetry_v1.TelemetryProfile_Region{Region: region}
	profilePerRegion1.Group = metricsGroup
	profilePerRegion2 := inv_testing.CreateTelemetryProfile(t, nil, nil, region, metricsGroup, true)
	profilePerRegion2.Group = metricsGroup
	profilePerRegion2.Relation = &telemetry_v1.TelemetryProfile_Region{Region: region}

	testcases := map[string]struct {
		in        *inv_v1.ResourceFilter
		resources []*telemetry_v1.TelemetryProfile
		valid     bool
	}{
		"NoFilterAll": {
			in: &inv_v1.ResourceFilter{},
			resources: []*telemetry_v1.TelemetryProfile{
				profilePerInstance1, profilePerInstance2, profilePerInstance3,
				profilePerSite1, profilePerSite2, profilePerSite3,
				profilePerRegion1, profilePerRegion2,
			},
			valid: true,
		},
		"NoFilterOrderByResourceId": {
			in: &inv_v1.ResourceFilter{
				OrderBy: telemetryprofileres.FieldResourceID,
			},
			resources: []*telemetry_v1.TelemetryProfile{
				profilePerInstance1, profilePerInstance2, profilePerInstance3,
				profilePerSite1, profilePerSite2, profilePerSite3,
				profilePerRegion1, profilePerRegion2,
			},
			valid: true,
		},
		"NoFilterAllPerInstance": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf("has(%s)", telemetryprofileres.EdgeInstance),
			},
			resources: []*telemetry_v1.TelemetryProfile{
				profilePerInstance1, profilePerInstance2, profilePerInstance3,
			},
			valid: true,
		},
		"NoFilterAllPerSite": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf("has(%s)", telemetryprofileres.EdgeSite),
			},
			resources: []*telemetry_v1.TelemetryProfile{
				profilePerSite1, profilePerSite2, profilePerSite3,
			},
			valid: true,
		},
		"NoFilterAllPerRegion": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf("has(%s)", telemetryprofileres.EdgeRegion),
			},
			resources: []*telemetry_v1.TelemetryProfile{
				profilePerRegion1, profilePerRegion2,
			},
			valid: true,
		},
		"FilterByResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, telemetryprofileres.FieldResourceID, profilePerInstance1.ResourceId),
			},
			resources: []*telemetry_v1.TelemetryProfile{profilePerInstance1},
			valid:     true,
		},
		"FilterByRegionID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, telemetryprofileres.EdgeRegion, regionresource.FieldResourceID,
					region.GetResourceId()),
			},
			resources: []*telemetry_v1.TelemetryProfile{profilePerRegion1, profilePerRegion2},
			valid:     true,
		},
		"FilterBySiteID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, telemetryprofileres.EdgeSite, siteresource.FieldResourceID,
					site1.GetResourceId()),
			},
			resources: []*telemetry_v1.TelemetryProfile{profilePerSite1, profilePerSite2},
			valid:     true,
		},
		"FilterByInstanceID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, telemetryprofileres.EdgeInstance, instanceresource.FieldResourceID,
					inst1.GetResourceId()),
			},
			resources: []*telemetry_v1.TelemetryProfile{profilePerInstance1, profilePerInstance2},
			valid:     true,
		},
		"FilterByGroup": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, telemetryprofileres.EdgeGroup, telemetrygroupresource.FieldResourceID,
					metricsGroup.GetResourceId()),
			},
			resources: []*telemetry_v1.TelemetryProfile{
				profilePerInstance1, profilePerInstance2, profilePerInstance3,
				profilePerSite1, profilePerSite2, profilePerSite3,
				profilePerRegion1, profilePerRegion2,
			},
			valid: true,
		},
		"FilterByHasGroup": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s)`, telemetryprofileres.EdgeGroup),
			},
			resources: []*telemetry_v1.TelemetryProfile{
				profilePerInstance1, profilePerInstance2, profilePerInstance3,
				profilePerSite1, profilePerSite2, profilePerSite3,
				profilePerRegion1, profilePerRegion2,
			},
			valid: true,
		},
		"FilterByHasGroupHasProfiles": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s.%s)`, telemetryprofileres.EdgeGroup, telemetrygroupresource.EdgeProfiles),
			},
			resources: []*telemetry_v1.TelemetryProfile{
				profilePerInstance1, profilePerInstance2, profilePerInstance3,
				profilePerSite1, profilePerSite2, profilePerSite3,
				profilePerRegion1, profilePerRegion2,
			},
			valid: true,
		},
		"FilterLimit": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf("has(%s)", telemetryprofileres.EdgeInstance),
				Offset: 0,
				Limit:  3,
			},
			resources: []*telemetry_v1.TelemetryProfile{profilePerInstance1, profilePerInstance2, profilePerInstance3},
			valid:     true,
		},
		"FilterWithOffsetLimit": {
			in: &inv_v1.ResourceFilter{
				Offset: 10,
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

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_TelemetryProfile{}} // Set the resource kind
			findres, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("FilterTelemetryProfile() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("FilterTelemetryProfile() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				resIDs := inv_testing.GetSortedResourceIDSlice(tc.resources)
				inv_testing.SortHasResourceIDAndTenantID(findres.Resources)

				if !reflect.DeepEqual(resIDs, findres.Resources) {
					t.Errorf(
						"FilterTelemetryProfile() failed - want: %s, got: %s",
						resIDs,
						findres.Resources,
					)
				}
			}

			listres, err := inv_testing.TestClients[inv_testing.APIClient].List(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("ListTelemetryProfile() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("ListTelemetryProfile() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				resources := make([]*telemetry_v1.TelemetryProfile, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetTelemetryProfile())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, resources[i]); !eq {
						t.Errorf("ListTelemetryProfile() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

func Test_FilterNestedTelemetryProfile(t *testing.T) {
	os := inv_testing.CreateOs(t)
	region := inv_testing.CreateRegion(t, nil)
	site1 := inv_testing.CreateSite(t, region, nil)
	host1 := inv_testing.CreateHost(t, nil, nil)
	instance1 := inv_testing.CreateInstance(t, host1, os)

	group := inv_testing.CreateTelemetryGroupLogs(t, true)

	profilePerInstance := inv_testing.CreateTelemetryProfile(t, instance1, nil, nil, group, true)
	profilePerInstance.Relation = &telemetry_v1.TelemetryProfile_Instance{
		Instance: instance1,
	}
	profilePerInstance.Group = group

	profilePerSite := inv_testing.CreateTelemetryProfile(t, nil, site1, nil, group, true)
	profilePerSite.Relation = &telemetry_v1.TelemetryProfile_Site{
		Site: site1,
	}
	profilePerSite.Group = group

	testcases := map[string]struct {
		in                *inv_v1.ResourceFilter
		resources         []*telemetry_v1.TelemetryProfile
		valid             bool
		expectedCodeError codes.Code
	}{
		"FilterPerInstanceConfigByHostID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s = %q`, telemetryprofileres.EdgeInstance, instanceresource.EdgeHost,
					hostresource.FieldResourceID, host1.GetResourceId()),
			},
			resources: []*telemetry_v1.TelemetryProfile{profilePerInstance},
			valid:     true,
		},
		"FilterPerSiteConfigByRegionID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s = %q`, telemetryprofileres.EdgeSite, siteresource.EdgeRegion,
					regionresource.FieldResourceID, region.GetResourceId()),
			},
			resources: []*telemetry_v1.TelemetryProfile{profilePerSite},
			valid:     true,
		},
		"FailTooDeep": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s.%s.%s.%s = %q`, workloadmember.EdgeInstance,
					instanceresource.EdgeHost, hostresource.EdgeSite, siteresource.EdgeRegion,
					regionresource.EdgeParentRegion, regionresource.FieldResourceID, region.GetResourceId()),
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

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_TelemetryProfile{}} // Set the resource kind

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
						"Filter() failed - want: %s, got: %s",
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

				resources := make([]*telemetry_v1.TelemetryProfile, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetTelemetryProfile())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, resources[i]); !eq {
						t.Errorf("List() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

func Test_ListInheritedTelemetryProfiles(t *testing.T) {
	region1 := inv_testing.CreateRegion(t, nil)
	region2 := inv_testing.CreateRegion(t, region1)
	region3 := inv_testing.CreateRegion(t, region2)
	region4 := inv_testing.CreateRegion(t, region3)
	region5 := inv_testing.CreateRegion(t, region4)
	// Corner cases without telemetry profiles defined for the current resource
	region6 := inv_testing.CreateRegion(t, region3)
	region7 := inv_testing.CreateRegion(t, nil)
	// Site1 has no region hierarchy
	site1 := inv_testing.CreateSite(t, nil, nil)
	site2 := inv_testing.CreateSite(t, region4, nil)
	site3 := inv_testing.CreateSite(t, region5, nil)
	// Corner cases without telemetry profiles defined for the current resource
	site4 := inv_testing.CreateSite(t, region5, nil)
	site5 := inv_testing.CreateSite(t, nil, nil)

	// h1 has no site-region hierarchy
	h1 := inv_testing.CreateHost(t, nil, nil)
	// h2 has only site in the hierarchy
	h2 := inv_testing.CreateHost(t, site1, nil)
	h3 := inv_testing.CreateHost(t, site2, nil)
	h4 := inv_testing.CreateHost(t, site3, nil)
	// Corner cases without telemetry profiles defined for the current resource
	h5 := inv_testing.CreateHost(t, site3, nil)
	h6 := inv_testing.CreateHost(t, nil, nil)

	os := inv_testing.CreateOs(t)
	inst1 := inv_testing.CreateInstance(t, h1, os)
	inst2 := inv_testing.CreateInstance(t, h2, os)
	inst3 := inv_testing.CreateInstance(t, h3, os)
	inst4 := inv_testing.CreateInstance(t, h4, os)
	// Corner cases without telemetry profiles defined for the current resource
	inst5 := inv_testing.CreateInstance(t, h5, os)
	inst6 := inv_testing.CreateInstance(t, h6, os)

	metricsGroup := inv_testing.CreateTelemetryGroupMetrics(t, true)

	profile1PerInstance1 := inv_testing.CreateTelemetryProfile(t, inst1, nil, nil, metricsGroup, true)
	profile1PerInstance1.Relation = &telemetry_v1.TelemetryProfile_Instance{Instance: inst1}
	profile1PerInstance1.Group = metricsGroup
	profile2PerInstance1 := inv_testing.CreateTelemetryProfile(t, inst1, nil, nil, metricsGroup, true)
	profile2PerInstance1.Relation = &telemetry_v1.TelemetryProfile_Instance{Instance: inst1}
	profile2PerInstance1.Group = metricsGroup
	profilePerInstance2 := inv_testing.CreateTelemetryProfile(t, inst2, nil, nil, metricsGroup, true)
	profilePerInstance2.Relation = &telemetry_v1.TelemetryProfile_Instance{Instance: inst2}
	profilePerInstance2.Group = metricsGroup
	profilePerInstance3 := inv_testing.CreateTelemetryProfile(t, inst3, nil, nil, metricsGroup, true)
	profilePerInstance3.Relation = &telemetry_v1.TelemetryProfile_Instance{Instance: inst3}
	profilePerInstance3.Group = metricsGroup
	profilePerInstance4 := inv_testing.CreateTelemetryProfile(t, inst4, nil, nil, metricsGroup, true)
	profilePerInstance4.Relation = &telemetry_v1.TelemetryProfile_Instance{Instance: inst4}
	profilePerInstance4.Group = metricsGroup

	profilePerSite1 := inv_testing.CreateTelemetryProfile(t, nil, site1, nil, metricsGroup, true)
	profilePerSite1.Relation = &telemetry_v1.TelemetryProfile_Site{Site: site1}
	profilePerSite1.Group = metricsGroup
	profile1PerSite2 := inv_testing.CreateTelemetryProfile(t, nil, site2, nil, metricsGroup, true)
	profile1PerSite2.Relation = &telemetry_v1.TelemetryProfile_Site{Site: site2}
	profile1PerSite2.Group = metricsGroup
	profile2PerSite2 := inv_testing.CreateTelemetryProfile(t, nil, site2, nil, metricsGroup, true)
	profile2PerSite2.Relation = &telemetry_v1.TelemetryProfile_Site{Site: site2}
	profile2PerSite2.Group = metricsGroup
	profilePerSite3 := inv_testing.CreateTelemetryProfile(t, nil, site3, nil, metricsGroup, true)
	profilePerSite3.Relation = &telemetry_v1.TelemetryProfile_Site{Site: site3}
	profilePerSite3.Group = metricsGroup

	profilePerRegion1 := inv_testing.CreateTelemetryProfile(t, nil, nil, region1, metricsGroup, true)
	profilePerRegion1.Relation = &telemetry_v1.TelemetryProfile_Region{Region: region1}
	profilePerRegion1.Group = metricsGroup
	profile1PerRegion2 := inv_testing.CreateTelemetryProfile(t, nil, nil, region2, metricsGroup, true)
	profile1PerRegion2.Group = metricsGroup
	profile1PerRegion2.Relation = &telemetry_v1.TelemetryProfile_Region{Region: region2}
	profile2PerRegion2 := inv_testing.CreateTelemetryProfile(t, nil, nil, region2, metricsGroup, true)
	profile2PerRegion2.Group = metricsGroup
	profile2PerRegion2.Relation = &telemetry_v1.TelemetryProfile_Region{Region: region2}
	profile3PerRegion2 := inv_testing.CreateTelemetryProfile(t, nil, nil, region2, metricsGroup, true)
	profile3PerRegion2.Group = metricsGroup
	profile3PerRegion2.Relation = &telemetry_v1.TelemetryProfile_Region{Region: region2}
	profilePerRegion3 := inv_testing.CreateTelemetryProfile(t, nil, nil, region3, metricsGroup, true)
	profilePerRegion3.Group = metricsGroup
	profilePerRegion3.Relation = &telemetry_v1.TelemetryProfile_Region{Region: region3}
	profilePerRegion4 := inv_testing.CreateTelemetryProfile(t, nil, nil, region4, metricsGroup, true)
	profilePerRegion4.Group = metricsGroup
	profilePerRegion4.Relation = &telemetry_v1.TelemetryProfile_Region{Region: region4}
	profile1PerRegion5 := inv_testing.CreateTelemetryProfile(t, nil, nil, region5, metricsGroup, true)
	profile1PerRegion5.Group = metricsGroup
	profile1PerRegion5.Relation = &telemetry_v1.TelemetryProfile_Region{Region: region5}
	profile2PerRegion5 := inv_testing.CreateTelemetryProfile(t, nil, nil, region5, metricsGroup, true)
	profile2PerRegion5.Group = metricsGroup
	profile2PerRegion5.Relation = &telemetry_v1.TelemetryProfile_Region{Region: region5}

	testcases := map[string]struct {
		renderBy  *inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy
		resources []*telemetry_v1.TelemetryProfile
		valid     bool
	}{
		"ByInstance1ID": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_InstanceId{InstanceId: inst1.ResourceId},
			},
			resources: []*telemetry_v1.TelemetryProfile{profile1PerInstance1, profile2PerInstance1},
			valid:     true,
		},
		"ByInstance2ID": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_InstanceId{InstanceId: inst2.ResourceId},
			},
			resources: []*telemetry_v1.TelemetryProfile{
				profilePerInstance2, profilePerSite1,
			},
			valid: true,
		},
		"ByInstance3ID": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_InstanceId{InstanceId: inst3.ResourceId},
			},
			resources: []*telemetry_v1.TelemetryProfile{
				profilePerInstance3,
				profile1PerSite2, profile2PerSite2,
				profilePerRegion4,
				profilePerRegion3,
				profile1PerRegion2, profile2PerRegion2, profile3PerRegion2,
				profilePerRegion1,
			},
			valid: true,
		},
		"ByInstance4ID": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_InstanceId{InstanceId: inst4.ResourceId},
			},
			resources: []*telemetry_v1.TelemetryProfile{
				profilePerInstance4,
				profilePerSite3,
				profile1PerRegion5, profile2PerRegion5,
				profilePerRegion4,
				profilePerRegion3,
				profile1PerRegion2, profile2PerRegion2, profile3PerRegion2,
				profilePerRegion1,
			},
			valid: true,
		},
		"ByInstance5ID": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_InstanceId{InstanceId: inst5.ResourceId},
			},
			resources: []*telemetry_v1.TelemetryProfile{
				profilePerSite3,
				profile1PerRegion5, profile2PerRegion5,
				profilePerRegion4,
				profilePerRegion3,
				profile1PerRegion2, profile2PerRegion2, profile3PerRegion2,
				profilePerRegion1,
			},
			valid: true,
		},
		"ByInstance6ID": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_InstanceId{InstanceId: inst6.ResourceId},
			},
			resources: []*telemetry_v1.TelemetryProfile{},
			valid:     true,
		},
		"BySite1ID": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_SiteId{SiteId: site1.ResourceId},
			},
			resources: []*telemetry_v1.TelemetryProfile{profilePerSite1},
			valid:     true,
		},
		"BySite2ID": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_SiteId{SiteId: site2.ResourceId},
			},
			resources: []*telemetry_v1.TelemetryProfile{
				profile1PerSite2, profile2PerSite2,
				profilePerRegion4,
				profilePerRegion3,
				profile1PerRegion2, profile2PerRegion2, profile3PerRegion2,
				profilePerRegion1,
			},
			valid: true,
		},
		"BySite3ID": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_SiteId{SiteId: site3.ResourceId},
			},
			resources: []*telemetry_v1.TelemetryProfile{
				profilePerSite3,
				profile1PerRegion5, profile2PerRegion5,
				profilePerRegion4,
				profilePerRegion3,
				profile1PerRegion2, profile2PerRegion2, profile3PerRegion2,
				profilePerRegion1,
			},
			valid: true,
		},
		"BySite4ID": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_SiteId{SiteId: site4.ResourceId},
			},
			resources: []*telemetry_v1.TelemetryProfile{
				profile1PerRegion5, profile2PerRegion5,
				profilePerRegion4,
				profilePerRegion3,
				profile1PerRegion2, profile2PerRegion2, profile3PerRegion2,
				profilePerRegion1,
			},
			valid: true,
		},
		"BySite5ID": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_SiteId{SiteId: site5.ResourceId},
			},
			resources: []*telemetry_v1.TelemetryProfile{},
			valid:     true,
		},
		"ByRegion1ID": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_RegionId{RegionId: region1.ResourceId},
			},
			resources: []*telemetry_v1.TelemetryProfile{profilePerRegion1},
			valid:     true,
		},
		"ByRegion2ID": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_RegionId{RegionId: region2.ResourceId},
			},
			resources: []*telemetry_v1.TelemetryProfile{
				profilePerRegion1,
				profile1PerRegion2, profile2PerRegion2, profile3PerRegion2,
			},
			valid: true,
		},
		"ByRegion3ID": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_RegionId{RegionId: region3.ResourceId},
			},
			resources: []*telemetry_v1.TelemetryProfile{
				profilePerRegion1,
				profile1PerRegion2, profile2PerRegion2, profile3PerRegion2,
				profilePerRegion3,
			},
			valid: true,
		},
		"ByRegion4ID": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_RegionId{RegionId: region4.ResourceId},
			},
			resources: []*telemetry_v1.TelemetryProfile{
				profilePerRegion1,
				profile1PerRegion2, profile2PerRegion2, profile3PerRegion2, profilePerRegion3,
				profilePerRegion4,
			},
			valid: true,
		},
		"ByRegion5ID": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_RegionId{RegionId: region5.ResourceId},
			},
			resources: []*telemetry_v1.TelemetryProfile{
				profilePerRegion1,
				profile1PerRegion2, profile2PerRegion2, profile3PerRegion2,
				profilePerRegion3,
				profilePerRegion4,
				profile1PerRegion5, profile2PerRegion5,
			},
			valid: true,
		},
		"ByRegion6ID": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_RegionId{RegionId: region6.ResourceId},
			},
			resources: []*telemetry_v1.TelemetryProfile{
				profilePerRegion1,
				profile1PerRegion2, profile2PerRegion2, profile3PerRegion2,
				profilePerRegion3,
			},
			valid: true,
		},
		"ByRegion7ID": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_RegionId{RegionId: region7.ResourceId},
			},
			resources: []*telemetry_v1.TelemetryProfile{},
			valid:     true,
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			resp, err := inv_testing.TestClients[inv_testing.APIClient].
				ListInheritedTelemetryProfiles(ctx, tc.renderBy, "", "", 20, 0)

			if err != nil {
				if tc.valid {
					t.Errorf("ListInheritedTelemetry() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("ListInheritedTelemetry() succeeded but should have failed")
				}
			}
			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				findRes := resp.GetTelemetryProfiles()
				inv_testing.OrderByResourceID(findRes)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, findRes[i]); !eq {
						t.Errorf("ListInheritedTelemetry() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

func Test_ListInheritedTelemetryProfilesPaginate(t *testing.T) {
	region1 := inv_testing.CreateRegion(t, nil)
	region2 := inv_testing.CreateRegion(t, region1)
	region3 := inv_testing.CreateRegion(t, region2)
	region4 := inv_testing.CreateRegion(t, region3)
	region5 := inv_testing.CreateRegion(t, region4)
	site1 := inv_testing.CreateSite(t, region5, nil)
	h1 := inv_testing.CreateHost(t, site1, nil)
	os := inv_testing.CreateOs(t)
	inst1 := inv_testing.CreateInstance(t, h1, os)

	metricsGroup := inv_testing.CreateTelemetryGroupMetrics(t, true)
	logsGroup := inv_testing.CreateTelemetryGroupLogs(t, true)

	var metricsPerInstance1 []*telemetry_v1.TelemetryProfile
	for i := 0; i < 10; i++ {
		profilePerInstance := inv_testing.CreateTelemetryProfile(t, inst1, nil, nil, metricsGroup, true)
		profilePerInstance.Relation = &telemetry_v1.TelemetryProfile_Instance{Instance: inst1}
		profilePerInstance.Group = metricsGroup
		metricsPerInstance1 = append(metricsPerInstance1, profilePerInstance)
	}

	var logsPerRegion4 []*telemetry_v1.TelemetryProfile
	for i := 0; i < 10; i++ {
		profilePerRegion := inv_testing.CreateTelemetryProfile(t, nil, nil, region4, logsGroup, true)
		profilePerRegion.Relation = &telemetry_v1.TelemetryProfile_Region{Region: region4}
		profilePerRegion.Group = logsGroup
		logsPerRegion4 = append(logsPerRegion4, profilePerRegion)
	}
	inv_testing.OrderByResourceID(metricsPerInstance1)
	inv_testing.OrderByResourceID(logsPerRegion4)
	allProfiles := metricsPerInstance1
	allProfiles = append(allProfiles, logsPerRegion4...)
	inv_testing.OrderByResourceID(allProfiles)

	testcases := map[string]struct {
		renderBy *inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy
		orderBy  string
		filter   string
		limit    uint32
		offset   uint32
		expRes   []*telemetry_v1.TelemetryProfile
	}{
		"Paginate1": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_InstanceId{InstanceId: inst1.ResourceId},
			},
			orderBy: telemetryprofileres.FieldResourceID,
			filter:  "",
			limit:   5,
			offset:  0,
			expRes:  allProfiles[0:5],
		},
		"Paginate2": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_InstanceId{InstanceId: inst1.ResourceId},
			},
			orderBy: telemetryprofileres.FieldResourceID,
			filter:  "",
			limit:   10,
			offset:  5,
			expRes:  allProfiles[5:15],
		},
		"Paginate3": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_InstanceId{InstanceId: inst1.ResourceId},
			},
			orderBy: telemetryprofileres.FieldResourceID,
			filter:  "",
			limit:   10,
			offset:  20,
		},
		"Paginate4": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_InstanceId{InstanceId: inst1.ResourceId},
			},
			orderBy: telemetryprofileres.FieldResourceID,
			filter:  "",
			limit:   100,
			offset:  0,
			expRes:  allProfiles,
		},
		"FilterByProfileKindMetrics": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_InstanceId{InstanceId: inst1.ResourceId},
			},
			orderBy: telemetryprofileres.FieldResourceID,
			filter: fmt.Sprintf("%s = %s",
				telemetryprofileres.FieldKind, telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS),
			limit:  100,
			offset: 0,
			expRes: metricsPerInstance1,
		},
		"FilterByProfileKindLogs": {
			renderBy: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_InstanceId{InstanceId: inst1.ResourceId},
			},
			orderBy: telemetryprofileres.FieldResourceID,
			filter: fmt.Sprintf("%s = %s",
				telemetryprofileres.FieldKind, telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS),
			limit:  100,
			offset: 0,
			expRes: logsPerRegion4,
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			resp, err := inv_testing.TestClients[inv_testing.APIClient].
				ListInheritedTelemetryProfiles(ctx, tc.renderBy, tc.filter, tc.orderBy, tc.limit, tc.offset)
			require.NoError(t, err)
			findRes := resp.GetTelemetryProfiles()
			require.Equal(t, len(tc.expRes), len(findRes))
			inv_testing.OrderByResourceID(findRes)
			for i, expected := range tc.expRes {
				if eq, diff := inv_testing.ProtoEqualOrDiff(expected, findRes[i]); !eq {
					t.Errorf("ListInheritedTelemetryProfiles() data not equal: %v", diff)
				}
			}
		})
	}
}

func Test_TelemetryProfileEnumStatusMap(t *testing.T) {
	v, err := store.TelemetryProfileEnumStatusMap("invalid_input",
		int32(telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS))
	assert.Error(t, err)
	assert.Nil(t, v)
}

func TestTelemetryProfileMTSanity(t *testing.T) {
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	suite.Run(t, &struct{ mt }{
		mt: mt{
			createResource: func(tenantID string) (string, *inv_v1.Resource) {
				group := dao.CreateTelemetryGroupMetrics(t, tenantID, true)
				region := dao.CreateRegion(t, tenantID)
				tp := dao.CreateTelemetryProfile(t, tenantID, inv_testing.TelemetryProfileTarget(region), group, true)
				res, err := util.WrapResource(tp)
				require.NoError(t, err)
				return tp.GetResourceId(), res
			},
		},
	})
}

func TestMultitenancySanity_ListInheritedTelemetryProfile(t *testing.T) {
	T1 := uuid.NewString()
	T2 := uuid.NewString()

	testLogGroups := []string{
		"kmseg",
		"syslog",
	}

	region1Tenant1 := &location_v1.RegionResource{
		Name:     "Test Region 1",
		Metadata: `[{"key":"cluster-name","value":"test"}]`,
		TenantId: T1,
	}

	createRegion1Tenant1Request := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Region{
			Region: region1Tenant1,
		},
	}

	tg1Tenant1 := &telemetry_v1.TelemetryGroupResource{
		Name:          "Test TG 1",
		TenantId:      T1,
		Kind:          telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS,
		CollectorKind: telemetry_v1.CollectorKind_COLLECTOR_KIND_HOST,
		Groups:        testLogGroups,
	}

	createTg1Tenant1Request := &inv_v1.Resource{
		Resource: &inv_v1.Resource_TelemetryGroup{
			TelemetryGroup: tg1Tenant1,
		},
	}

	createTp1Tenant1Request := &inv_v1.Resource{
		Resource: &inv_v1.Resource_TelemetryProfile{
			TelemetryProfile: &telemetry_v1.TelemetryProfile{
				TenantId: T1,
				Kind:     telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS,
				LogLevel: telemetry_v1.SeverityLevel_SEVERITY_LEVEL_DEBUG,
				Relation: &telemetry_v1.TelemetryProfile_Region{
					Region: region1Tenant1,
				},
				Group: tg1Tenant1,
			},
		},
	}

	region1Tenant2 := &location_v1.RegionResource{
		Name:     "Test Region 1",
		Metadata: `[{"key":"cluster-name","value":"test"},{"key":"app-id","value":"test2-value"}]`,
		TenantId: T2,
	}
	createRegion1Tenant2Request := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Region{
			Region: region1Tenant2,
		},
	}

	tg1Tenant2 := &telemetry_v1.TelemetryGroupResource{
		Name:          "Test TG 1",
		TenantId:      T2,
		Kind:          telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS,
		CollectorKind: telemetry_v1.CollectorKind_COLLECTOR_KIND_HOST,
		Groups:        testLogGroups,
	}
	createTg1Tenant2Request := &inv_v1.Resource{
		Resource: &inv_v1.Resource_TelemetryGroup{
			TelemetryGroup: tg1Tenant2,
		},
	}

	createTp1Tenant2Request := &inv_v1.Resource{
		Resource: &inv_v1.Resource_TelemetryProfile{
			TelemetryProfile: &telemetry_v1.TelemetryProfile{
				TenantId: T2,
				Kind:     telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS,
				LogLevel: telemetry_v1.SeverityLevel_SEVERITY_LEVEL_DEBUG,
				Relation: &telemetry_v1.TelemetryProfile_Region{
					Region: region1Tenant2,
				},
				Group: tg1Tenant2,
			},
		},
	}

	tenantAwareClient := inv_testing.NewInvResourceDAOOrFail(t).GetAPIClient()

	reg1Tenant1CreateResp, err := tenantAwareClient.Create(
		context.TODO(), createRegion1Tenant1Request.GetRegion().TenantId, createRegion1Tenant1Request)
	require.NoError(t, err)
	reg1Tenant1ID := inv_testing.GetResourceIDOrFail(t, reg1Tenant1CreateResp)
	t.Cleanup(func() {
		_, derr := tenantAwareClient.Delete(
			context.TODO(), createRegion1Tenant1Request.GetRegion().TenantId, reg1Tenant1ID)
		require.NoError(t, derr)
	})
	region1Tenant1.ResourceId = reg1Tenant1ID

	reg1Tenant2CreateResp, err := tenantAwareClient.Create(
		context.TODO(), createRegion1Tenant2Request.GetRegion().TenantId, createRegion1Tenant2Request)
	require.NoError(t, err)
	reg1Tenant2ID := inv_testing.GetResourceIDOrFail(t, reg1Tenant2CreateResp)
	t.Cleanup(func() {
		_, derr := tenantAwareClient.Delete(
			context.TODO(), createRegion1Tenant2Request.GetRegion().TenantId, reg1Tenant2ID)
		require.NoError(t, derr)
	})
	region1Tenant2.ResourceId = reg1Tenant2ID

	tg1Tenant1CreateResp, err := tenantAwareClient.Create(
		context.TODO(), createTg1Tenant1Request.GetTelemetryGroup().GetTenantId(), createTg1Tenant1Request)
	require.NoError(t, err)
	tg1Tenant1ID := inv_testing.GetResourceIDOrFail(t, tg1Tenant1CreateResp)
	t.Cleanup(func() {
		_, derr := tenantAwareClient.Delete(
			context.TODO(), createTg1Tenant1Request.GetTelemetryGroup().GetTenantId(), tg1Tenant1ID)
		require.NoError(t, derr)
	})
	tg1Tenant1.ResourceId = tg1Tenant1ID

	tg1Tenant2CreateResp, err := tenantAwareClient.Create(
		context.TODO(), createTg1Tenant2Request.GetTelemetryGroup().GetTenantId(), createTg1Tenant2Request)
	require.NoError(t, err)
	tg1Tenant2ID := inv_testing.GetResourceIDOrFail(t, tg1Tenant2CreateResp)
	t.Cleanup(func() {
		_, derr := tenantAwareClient.Delete(
			context.TODO(), createTg1Tenant2Request.GetTelemetryGroup().GetTenantId(), tg1Tenant2ID)
		require.NoError(t, derr)
	})
	tg1Tenant2.ResourceId = tg1Tenant2ID

	tp1Tenant1CreateResp, err := tenantAwareClient.Create(context.TODO(),
		createTp1Tenant1Request.GetTelemetryProfile().GetTenantId(), createTp1Tenant1Request)
	require.NoError(t, err)
	tp1Tenant1ID := inv_testing.GetResourceIDOrFail(t, tp1Tenant1CreateResp)
	t.Cleanup(func() {
		_, derr := tenantAwareClient.Delete(
			context.TODO(), createTp1Tenant1Request.GetTelemetryProfile().GetTenantId(), tp1Tenant1ID)
		require.NoError(t, derr)
	})
	tp1Tenant1ResID := tp1Tenant1ID

	tp1Tenant2CreateResp, err := tenantAwareClient.Create(context.TODO(),
		createTp1Tenant2Request.GetTelemetryProfile().GetTenantId(), createTp1Tenant2Request)
	require.NoError(t, err)
	tp1Tenant2ID := inv_testing.GetResourceIDOrFail(t, tp1Tenant2CreateResp)
	t.Cleanup(func() {
		_, derr := tenantAwareClient.Delete(
			context.TODO(), createTp1Tenant2Request.GetTelemetryProfile().GetTenantId(), tp1Tenant2ID)
		require.NoError(t, derr)
	})
	tp1Tenant2ResID := tp1Tenant2ID

	filterTpTenant1 := &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
		Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_RegionId{
			RegionId: region1Tenant1.GetResourceId(),
		},
	}
	inheritedTenant1Response, err := tenantAwareClient.ListInheritedTelemetryProfiles(
		context.TODO(), T1, filterTpTenant1, "", "", 100, 0)
	require.NoError(t, err)
	tpsTenant1 := inheritedTenant1Response.GetTelemetryProfiles()
	require.Len(t, tpsTenant1, 1)
	assert.Equal(t, tp1Tenant1ResID, tpsTenant1[0].GetResourceId())
	assert.Equal(t, T1, tpsTenant1[0].GetTenantId())

	filterTpTenant2 := &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy{
		Id: &inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_RegionId{
			RegionId: region1Tenant2.GetResourceId(),
		},
	}
	inheritedTenant2Response, err := tenantAwareClient.ListInheritedTelemetryProfiles(
		context.TODO(), T2, filterTpTenant2, "", "", 100, 0)
	require.NoError(t, err)
	tpsTenant2 := inheritedTenant2Response.GetTelemetryProfiles()
	require.Len(t, tpsTenant2, 1)
	assert.Equal(t, tp1Tenant2ResID, tpsTenant2[0].GetResourceId())
	assert.Equal(t, T2, tpsTenant2[0].GetTenantId())

	// Wrong tenant
	wrongTenant, err := tenantAwareClient.ListInheritedTelemetryProfiles(
		context.TODO(), T1, filterTpTenant2, "", "", 100, 0)
	require.NoError(t, err)
	assert.Len(t, wrongTenant.TelemetryProfiles, 0)
}

func TestDeleteResources_TelemetryProfiles(t *testing.T) {
	suite.Run(t, &struct{ hardDeleteAllResourcesSuite }{
		hardDeleteAllResourcesSuite: hardDeleteAllResourcesSuite{
			createModel: func(dao *inv_testing.InvResourceDAO) (string, int) {
				tenantID := uuid.NewString()
				site := dao.CreateSite(t, tenantID)
				region := dao.CreateRegion(t, tenantID)
				tg1 := dao.CreateTelemetryGroupLogs(t, tenantID, true)
				tg2 := dao.CreateTelemetryGroupMetrics(t, tenantID, true)
				return tenantID, len([]any{
					dao.CreateTelemetryProfile(t, tenantID, inv_testing.TelemetryProfileTarget(site), tg1, false),
					dao.CreateTelemetryProfile(t, tenantID, inv_testing.TelemetryProfileTarget(region), tg2, false),
				})
			},
			resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_TELEMETRY_PROFILE,
		},
	})
}
