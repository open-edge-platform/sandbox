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
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/providerresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/regionresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/siteresource"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	location_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/location/v1"
	ou_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/ou/v1"
	providerv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/provider/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/filters"
)

func Test_Metadata_Inheritance_Site(t *testing.T) {
	// Create required Regions and OUs
	region0 := inv_testing.CreateRegion(t, nil)
	region1 := inv_testing.CreateRegionWithMeta(t, metaR1, nil)
	region2 := inv_testing.CreateRegionWithMeta(t, metaR2, region1)
	region3 := inv_testing.CreateRegionWithMeta(t, metaR3, region2)
	region4 := inv_testing.CreateRegion(t, region3)

	ou0 := inv_testing.CreateOu(t, nil)
	ou1 := inv_testing.CreateOuWithMeta(t, metaO1, nil)
	ou2 := inv_testing.CreateOuWithMeta(t, metaO2, ou1)
	ou3 := inv_testing.CreateOuWithMeta(t, metaO3, ou2)
	ou4 := inv_testing.CreateOu(t, ou3)
	ou5 := inv_testing.CreateOuWithMeta(t, metaO6, ou3)

	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	testcases := map[string]struct {
		in          *location_v1.SiteResource
		expPhyMeta  *string
		expLogiMeta *string
	}{
		"NoMetadataFromParent1": {
			in: &location_v1.SiteResource{
				Name:   "Test Site 1",
				Region: region0,
				Ou:     ou0,
			},
			expPhyMeta:  &emptyString,
			expLogiMeta: &emptyString,
		},
		"NoMetadataFromParent2": {
			in: &location_v1.SiteResource{
				Name: "Test Site 1",
			},
			expPhyMeta:  &emptyString,
			expLogiMeta: &emptyString,
		},
		"PhyMetadataFromParent": {
			in: &location_v1.SiteResource{
				Name:   "Test Site 1",
				Region: region4,
			},
			expPhyMeta:  &expPhyMeta1,
			expLogiMeta: nil,
		},
		"LogiMetadataFromParent": {
			in: &location_v1.SiteResource{
				Name: "Test Site 1",
				Ou:   ou4,
			},
			expPhyMeta:  nil,
			expLogiMeta: &expLogiMeta1,
		},
		"PhyMetadataOverrideFromParent1": {
			in: &location_v1.SiteResource{
				Name:   "Test Site 1",
				Ou:     ou4,
				Region: region4,
			},
			expPhyMeta:  &expPhyMeta1,
			expLogiMeta: &emptyString,
		},
		"PhyMetadataOverrideFromParent2": {
			in: &location_v1.SiteResource{
				Name:   "Test Site 1",
				Ou:     ou5,
				Region: region4,
			},
			expPhyMeta:  &expPhyMeta1,
			expLogiMeta: &metaO6,
		},
		"InheritMetadataFromOuParentAndLocal": {
			in: &location_v1.SiteResource{
				Name:     "Test Site 1",
				Ou:       ou4,
				Metadata: metaO5,
			},
			expPhyMeta:  nil,
			expLogiMeta: &expLogiMeta2,
		},
		"InheritMetadataFromRegionParentAndLocal": {
			in: &location_v1.SiteResource{
				Name:     "Test Site 1",
				Region:   region4,
				Metadata: metaR5,
			},
			expPhyMeta:  &expPhyMeta2,
			expLogiMeta: nil,
		},
		"InheritMetadataFromRegionAndOuParentAndLocal": {
			in: &location_v1.SiteResource{
				Name:     "Test Site 1",
				Region:   region4,
				Ou:       ou4,
				Metadata: metaR5,
			},
			expPhyMeta:  &expPhyMeta2,
			expLogiMeta: &emptyString,
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Site{Site: tc.in},
			}
			csiteResp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
			require.NoError(t, err, "CreateSite() failed")
			siteResID := inv_testing.GetResourceIDOrFail(t, csiteResp)
			t.Cleanup(func() { inv_testing.DeleteResource(t, siteResID) })

			t.Run("Verify by GET", func(t *testing.T) {
				getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, siteResID)
				require.NoError(t, err, "GetSite() failed")
				if tc.expPhyMeta != nil && !CompareMetadata(t, getresp.RenderedMetadata.PhyMetadata, *tc.expPhyMeta) {
					t.Errorf("Physical Metadata data not equal - want: %s, got: %s",
						*tc.expPhyMeta, getresp.RenderedMetadata.PhyMetadata,
					)
				}
				if tc.expLogiMeta != nil && !CompareMetadata(t, getresp.RenderedMetadata.LogiMetadata, *tc.expLogiMeta) {
					t.Errorf("Logical Metadata data not equal - want: %s, got: %s",
						*tc.expLogiMeta, getresp.RenderedMetadata.LogiMetadata,
					)
				}
			})

			t.Run("Verify by LIST", func(t *testing.T) {
				listResp, err := inv_testing.TestClients[inv_testing.APIClient].
					List(ctx, &inv_v1.ResourceFilter{
						Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Site{}},
						Filter: filters.NewBuilderWith(filters.ValEq("tenant_id", client.FakeTenantID)).
							And(filters.ValEq("resource_id", siteResID)).
							Build(),
					})

				require.NoError(t, err, "List(sites) failed")
				require.Len(t, listResp.Resources, 1)
				resource := listResp.GetResources()[0]
				if tc.expPhyMeta != nil && !CompareMetadata(t, resource.RenderedMetadata.PhyMetadata, *tc.expPhyMeta) {
					t.Errorf("Physical Metadata data not equal - want: %s, got: %s",
						*tc.expPhyMeta, resource.RenderedMetadata.PhyMetadata,
					)
				}
				if tc.expLogiMeta != nil && !CompareMetadata(t, resource.RenderedMetadata.LogiMetadata, *tc.expLogiMeta) {
					t.Errorf("Logical Metadata data not equal - want: %s, got: %s",
						*tc.expLogiMeta, resource.RenderedMetadata.LogiMetadata,
					)
				}
			})
		})
	}
}

func Test_StrongRelations_On_Delete_Site_Host(t *testing.T) {
	site1 := inv_testing.CreateSite(t, nil, nil)
	_ = inv_testing.CreateHost(t, site1, nil)

	err := inv_testing.DeleteResourceAndReturnError(t, site1.ResourceId)
	assertStrongRelationError(t, err, "the site has relations with host and cannot be deleted")
}

func Test_StrongRelations_On_Delete_Site_NetSeg(t *testing.T) {
	site := inv_testing.CreateSite(t, nil, nil)
	_ = inv_testing.CreateNetworkSegment(t, "good seg", site, 100)

	err := inv_testing.DeleteResourceAndReturnError(t, site.ResourceId)
	assertStrongRelationError(t, err, "constraint failed: ERROR: update or delete on table")
}

func Test_Create_Get_Delete_Update_Site(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	ou := inv_testing.CreateOu(t, nil)
	provider := inv_testing.CreateProvider(t, "Test")

	testcases := map[string]struct {
		in    *location_v1.SiteResource
		valid bool
	}{
		"CreateGoodSite": {
			in: &location_v1.SiteResource{
				Name:             "Test Site 1",
				Region:           region,
				Ou:               ou,
				SiteLat:          490033000, // Lat/Long are int32 fixed point, divide by 10000000 to get decimal notation
				SiteLng:          82414000,  // This location is Karlsruhe, DE per Wikipedia
				DnsServers:       []string{"10.10.10.53", "10.10.20.53"},
				DockerRegistries: []string{"https://registry.example.com"},
				MetricsEndpoint:  "https://metrics.example.com",
				HttpProxy:        "http://proxy.example.com:8080",
				HttpsProxy:       "https://proxy.example.com:8080",
				FtpProxy:         "ftp://proxy.example.com:8080",
				NoProxy:          "localhost",
				Metadata:         `[{"key":"cluster-name","value":""},{"key":"app-id","value":""}]`,
			},
			valid: true,
		},
		"CreateGoodSiteEmptyDnsServers": {
			in: &location_v1.SiteResource{
				Name:             "Test Site 2",
				DnsServers:       []string{},
				DockerRegistries: []string{},
			},
			valid: true,
		},
		"CreateGoodSiteWithProvider": {
			in: &location_v1.SiteResource{
				Name:             "Test Site 2",
				DnsServers:       []string{},
				DockerRegistries: []string{},
				Provider:         provider,
			},
			valid: true,
		},
		"CreateBadSiteWithResourceIdSet": {
			// This tests case verifies that create requests with a resource ID
			// already set are rejected.
			in: &location_v1.SiteResource{
				ResourceId:       "site-12345678",
				Name:             "Test Site 2",
				Region:           region,
				Ou:               ou,
				SiteLat:          490033000, // Lat/Long are int32 fixed point, divide by 10000000 to get decimal notation
				SiteLng:          82414000,  // This location is Karlsruhe, DE per Wikipedia
				DnsServers:       []string{"10.10.10.53", "10.10.20.53"},
				DockerRegistries: []string{"https://registry.example.com"},
				MetricsEndpoint:  "https://metrics.example.com",
				HttpProxy:        "http://proxy.example.com:8080",
				HttpsProxy:       "https://proxy.example.com:8080",
				FtpProxy:         "ftp://proxy.example.com:8080",
				NoProxy:          "localhost",
			},
			valid: false,
		},
		"CreateBadSiteNonExistingRegion": {
			// This tests case verifies that sites must point to a valid
			// existing region.
			in: &location_v1.SiteResource{
				Region: &location_v1.RegionResource{
					ResourceId: "region-12345678",
				},
			},
			valid: false,
		},
		"CreateBadSiteNonExistingOu": {
			// This tests case verifies that sites must point to a valid
			// existing ou.
			in: &location_v1.SiteResource{
				Ou: &ou_v1.OuResource{
					ResourceId: "ou-12345678",
				},
			},
			valid: false,
		},
		"CreateBadSiteNonExistingProvider": {
			// This tests case verifies that sites must point to a valid
			// existing provider.
			in: &location_v1.SiteResource{
				Provider: &providerv1.ProviderResource{
					ResourceId: "provider-12345678",
				},
			},
			valid: false,
		},
		"CreateBadSiteWithInvalidLat1": {
			in: &location_v1.SiteResource{
				Name:             "Test Site 1",
				SiteLat:          -900000001,
				SiteLng:          824140,
				DnsServers:       []string{"10.10.10.53", "10.10.20.53"},
				DockerRegistries: []string{"https://registry.example.com"},
				MetricsEndpoint:  "https://metrics.example.com",
				HttpProxy:        "http://proxy.example.com:8080",
				HttpsProxy:       "https://proxy.example.com:8080",
				FtpProxy:         "ftp://proxy.example.com:8080",
				NoProxy:          "localhost",
			},
			valid: false,
		},
		"CreateBadSiteWithInvalidLat2": {
			in: &location_v1.SiteResource{
				Name:             "Test Site 1",
				SiteLat:          900000001,
				SiteLng:          824140,
				DnsServers:       []string{"10.10.10.53", "10.10.20.53"},
				DockerRegistries: []string{"https://registry.example.com"},
				MetricsEndpoint:  "https://metrics.example.com",
				HttpProxy:        "http://proxy.example.com:8080",
				HttpsProxy:       "https://proxy.example.com:8080",
				FtpProxy:         "ftp://proxy.example.com:8080",
				NoProxy:          "localhost",
			},
			valid: false,
		},
		"CreateBadSiteWithInvalidLong1": {
			in: &location_v1.SiteResource{
				Name:             "Test Site 1",
				SiteLat:          4900330,
				SiteLng:          -1800000001,
				DnsServers:       []string{"10.10.10.53", "10.10.20.53"},
				DockerRegistries: []string{"https://registry.example.com"},
				MetricsEndpoint:  "https://metrics.example.com",
				HttpProxy:        "http://proxy.example.com:8080",
				HttpsProxy:       "https://proxy.example.com:8080",
				FtpProxy:         "ftp://proxy.example.com:8080",
				NoProxy:          "localhost",
			},
			valid: false,
		},
		"CreateBadSiteWithInvalidLong2": {
			in: &location_v1.SiteResource{
				Name:             "Test Site 1",
				SiteLat:          4900330,
				SiteLng:          1800000001,
				DnsServers:       []string{"10.10.10.53", "10.10.20.53"},
				DockerRegistries: []string{"https://registry.example.com"},
				MetricsEndpoint:  "https://metrics.example.com",
				HttpProxy:        "http://proxy.example.com:8080",
				HttpsProxy:       "https://proxy.example.com:8080",
				FtpProxy:         "ftp://proxy.example.com:8080",
				NoProxy:          "localhost",
			},
			valid: false,
		},
		"CreateBadSiteInvalidMetadata": {
			in: &location_v1.SiteResource{
				Name:             "Test Site 1",
				Region:           region,
				Ou:               ou,
				SiteLat:          490033000, // Lat/Long are int32 fixed point, divide by 10000000 to get decimal notation
				SiteLng:          82414000,  // This location is Karlsruhe, DE per Wikipedia
				DnsServers:       []string{"10.10.10.53", "10.10.20.53"},
				DockerRegistries: []string{"https://registry.example.com"},
				MetricsEndpoint:  "https://metrics.example.com",
				HttpProxy:        "http://proxy.example.com:8080",
				HttpsProxy:       "https://proxy.example.com:8080",
				FtpProxy:         "ftp://proxy.example.com:8080",
				NoProxy:          "localhost",
				Metadata:         metaDuplicatedKeys,
			},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Site{Site: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			// create
			csiteResp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
			siteResID := csiteResp.GetSite().GetResourceId()

			if err != nil {
				if tc.valid {
					t.Errorf("CreateSite() failed: %s", err)
				}
			} else {
				tc.in.ResourceId = siteResID // Update with created resource ID.
				tc.in.CreatedAt = csiteResp.GetSite().GetCreatedAt()
				tc.in.UpdatedAt = csiteResp.GetSite().GetUpdatedAt()
				assertSameResource(t, createresreq, csiteResp, nil)
				if !tc.valid {
					t.Errorf("CreateSite() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				// get non-existent first
				_, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, "site-12345678")
				require.Error(t, err)

				// get
				getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, siteResID)
				require.NoError(t, err, "GetSite() failed")

				// verify data
				if eq, diff := inv_testing.ProtoEqualOrDiff(tc.in, getresp.GetResource().GetSite()); !eq {
					t.Errorf("GetSite() data not equal: %v", diff)
				}

				// update
				updateresreq := &inv_v1.Resource{
					Resource: &inv_v1.Resource_Site{
						Site: &location_v1.SiteResource{
							Name: "Updated Name",
						},
					},
				}

				fm := &fieldmaskpb.FieldMask{Paths: []string{siteresource.FieldName}}
				upRes, err := inv_testing.TestClients[inv_testing.RMClient].Update(
					ctx,
					siteResID,
					fm,
					updateresreq,
				)
				if err != nil {
					t.Errorf("UpdateSite() failed: %s", err)
				}

				// Validate returned resource
				assertSameResource(t, updateresreq, upRes, fm)

				// delete non-existent first
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, "site-12345678")
				require.Error(t, err)

				// delete
				_, err = inv_testing.TestClients[inv_testing.RMClient].Delete(
					ctx,
					siteResID,
				)
				if err != nil {
					t.Errorf("DeleteSite() failed %s", err)
				}
			}
		})
	}
}

func Test_FilterSites(t *testing.T) {
	region1 := inv_testing.CreateRegion(t, nil)
	region2 := inv_testing.CreateRegion(t, nil)
	ou1 := inv_testing.CreateOu(t, nil)
	site3 := inv_testing.CreateSite(t, nil, nil)
	provider := inv_testing.CreateProvider(t, "TEST")

	createresreq1 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Site{
			Site: &location_v1.SiteResource{
				Name:             "Test Site 2",
				Region:           region1,
				Ou:               ou1,
				SiteLat:          4900330, // Lat/Long are int32 fixed point, divide by 100000 to get decimal notation
				SiteLng:          824140,  // This location is Karlsruhe, DE per Wikipedia
				DnsServers:       []string{"10.10.10.53", "10.10.20.53"},
				DockerRegistries: []string{"https://registry.example.com"},
				MetricsEndpoint:  "https://metrics.example.com",
				HttpProxy:        "http://proxy.example.com:8080",
				HttpsProxy:       "http://proxy.example.com:8080",
				FtpProxy:         "http://proxy.example.com:8080",
				NoProxy:          "localhost",
				Metadata:         `[{"key":"cluster-name","value":""},{"key":"app-id","value":""}]`,
			},
		},
	}

	createresreq2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Site{
			Site: &location_v1.SiteResource{
				Name:             "Test Site 3",
				Region:           region2,
				SiteLat:          4900330, // Lat/Long are int32 fixed point, divide by 100000 to get decimal notation
				SiteLng:          824140,  // This location is Karlsruhe, DE per Wikipedia
				DnsServers:       []string{"10.20.10.53", "10.20.20.53"},
				DockerRegistries: []string{"https://registry.example.com"},
				MetricsEndpoint:  "https://metrics.example.com",
				Address:          "aabbccddee",
			},
		},
	}

	createresreq3 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Site{
			Site: &location_v1.SiteResource{
				Name:     "Test Site 4",
				Provider: provider,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	csiteResp1, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq1)
	siteResID1 := inv_testing.GetResourceIDOrFail(t, csiteResp1)
	csiteResp2, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq2)
	siteResID2 := inv_testing.GetResourceIDOrFail(t, csiteResp2)
	csiteResp3, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq3)
	siteResID3 := inv_testing.GetResourceIDOrFail(t, csiteResp3)
	t.Cleanup(func() { inv_testing.DeleteResource(t, siteResID1) })
	t.Cleanup(func() { inv_testing.DeleteResource(t, siteResID2) })
	t.Cleanup(func() { inv_testing.DeleteResource(t, siteResID3) })

	expSite1 := csiteResp1.GetSite()
	expSite1.ResourceId = siteResID1

	expSite2 := csiteResp2.GetSite()
	expSite2.ResourceId = siteResID2

	expSite3 := csiteResp3.GetSite()
	expSite3.ResourceId = siteResID3

	testcases := map[string]struct {
		in                    *inv_v1.ResourceFilter
		resources             []*location_v1.SiteResource
		valid                 bool
		mismatchFilterContent bool
	}{
		"NoFilter": {
			in:        &inv_v1.ResourceFilter{},
			resources: []*location_v1.SiteResource{expSite1, expSite2, site3, expSite3},
			valid:     true,
		},
		"NoFilterOrderByResourceID": {
			in: &inv_v1.ResourceFilter{
				OrderBy: siteresource.FieldResourceID,
			},
			resources: []*location_v1.SiteResource{expSite1, expSite2, site3, expSite3},
			valid:     true,
		},
		"FilterByEmptyResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = ""`, siteresource.FieldResourceID),
			},
			valid: true,
		},
		"FilterByResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, siteresource.FieldResourceID, expSite1.ResourceId),
			},
			resources: []*location_v1.SiteResource{expSite1},
			valid:     true,
		},
		"FilterByDnsPipeConcatStringField": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, siteresource.FieldDNSServers, expSite1.DnsServers[0]),
			},
			resources: []*location_v1.SiteResource{expSite1},
			valid:     true,
		},
		"FilterByDnsWildcardPipeConcatStringField": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, siteresource.FieldDNSServers, "*.10.53"),
			},
			resources: []*location_v1.SiteResource{expSite1, expSite2},
			valid:     true,
		},
		"FilterRegion": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, siteresource.EdgeRegion,
					regionresource.FieldResourceID, region1.GetResourceId()),
			},
			resources: []*location_v1.SiteResource{expSite1},
			valid:     true,
		},
		"FilterByHasRegion": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s)`, siteresource.EdgeRegion),
			},
			resources: []*location_v1.SiteResource{expSite1, expSite2},
			valid:     true,
		},
		"FilterByOuID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, siteresource.EdgeOu, ouresource.FieldResourceID,
					ou1.GetResourceId()),
			},
			resources: []*location_v1.SiteResource{expSite1},
			valid:     true,
		},
		"FilterByProviderID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, siteresource.EdgeProvider, providerresource.FieldResourceID,
					provider.GetResourceId()),
			},
			resources: []*location_v1.SiteResource{expSite3},
			valid:     true,
		},
		"FilterByHasRegionHasParentRegion": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s.%s)`, siteresource.EdgeRegion, regionresource.EdgeParentRegion),
			},
			resources: []*location_v1.SiteResource{},
			valid:     true,
		},
		"FilterEmptyRegion": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s)`, siteresource.EdgeRegion),
			},
			resources: []*location_v1.SiteResource{site3, expSite3},
			valid:     true,
		},
		"FilterAddress": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, siteresource.FieldAddress, "aabbccddee"),
			},
			resources: []*location_v1.SiteResource{expSite2},
			valid:     true,
		},
		"FilterEmptyAddress": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, siteresource.FieldAddress, ""),
			},
			resources: []*location_v1.SiteResource{expSite1, site3, expSite3},
			valid:     true,
		},
		"FilterMetadata": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = '%s' AND %s = '%s'`,
					siteresource.FieldMetadata, `{"key":"cluster-name","value":""}`,
					siteresource.FieldMetadata, `{"key":"app-id","value":""}`),
			},
			resources: []*location_v1.SiteResource{expSite1},
			valid:     true,
		},
		"FilterByMetadata": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = '%s'`, siteresource.FieldMetadata, `{"key":"cluster-name","value":""}`),
			},
			resources: []*location_v1.SiteResource{expSite1},
			valid:     true,
		},
		"FilterEmptyOu": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s)`, siteresource.EdgeOu),
			},
			resources: []*location_v1.SiteResource{expSite2, site3, expSite3},
			valid:     true,
		},
		"FilterEmptyProvider": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s)`, siteresource.EdgeProvider),
			},
			resources: []*location_v1.SiteResource{expSite1, expSite2, site3},
			valid:     true,
		},
		"FilterOffsetLimitOk": {
			in: &inv_v1.ResourceFilter{
				Offset: 0,
				Limit:  0,
			},
			resources: []*location_v1.SiteResource{expSite1, expSite2, site3, expSite3},
			valid:     true,
		},
		"FilterOffsetLimitMin": {
			in: &inv_v1.ResourceFilter{
				Offset: 1,
				Limit:  1,
			},
			resources:             []*location_v1.SiteResource{expSite1},
			valid:                 true,
			mismatchFilterContent: true,
		},
		"FilterOffsetLimitMax": {
			in: &inv_v1.ResourceFilter{
				Offset: 0,
				Limit:  4,
			},
			resources: []*location_v1.SiteResource{expSite1, expSite2, site3, expSite3},
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

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Site{}} // Set the resource kind
			findres, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("FilterSites() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("FilterSites() succeeded but should have failed")
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

				if !tc.mismatchFilterContent {
					if !reflect.DeepEqual(resIDs, findres.Resources) {
						t.Errorf(
							"FilterSites() failed - want: %s, got: %s",
							resIDs,
							findres.Resources,
						)
					}
				}

				if len(resIDs) != len(findres.Resources) {
					t.Errorf(
						"FilterSites() failed - want: %d, got: %d filtered elements",
						len(resIDs),
						len(findres.Resources),
					)
				}
				if tc.in.Limit != 0 {
					if int(tc.in.Limit) != len(findres.Resources) {
						t.Errorf(
							"FilterSites() failed - want: %d, got: %d limit filtered elements",
							len(resIDs),
							len(findres.Resources),
						)
					}
				}
			}

			listres, err := inv_testing.TestClients[inv_testing.APIClient].List(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("ListSites() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("ListSites() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				resources := make([]*location_v1.SiteResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetSite())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				if len(resources) != len(tc.resources) {
					t.Errorf(
						"ListSites() failed - want: %d, got: %d filtered elements",
						len(resources),
						len(findres.Resources),
					)
				}
				if tc.in.Limit != 0 {
					if int(tc.in.Limit) != len(resources) {
						t.Errorf(
							"ListSites() failed - want: %d, got: %d limit filtered elements",
							int(tc.in.Limit),
							len(resources),
						)
					}
				}
				for i, expected := range tc.resources {
					expCopy := *expected //nolint:govet // ok to copy lock in test
					siteEdgesOnlyResourceID(&expCopy)
					siteEdgesOnlyResourceID(resources[i])

					if !tc.mismatchFilterContent {
						// Compare metadata separately
						assert.True(t, CompareMetadata(t, resources[i].Metadata, expCopy.Metadata))
						expCopy.Metadata = ""
						resources[i].Metadata = ""
						if eq, diff := inv_testing.ProtoEqualOrDiff(&expCopy, resources[i]); !eq {
							t.Errorf("ListSites() data not equal: %v", diff)
						}
					}
				}
			}
		})
	}
}

func Test_UpdateSite(t *testing.T) {
	region1 := inv_testing.CreateRegion(t, nil)
	region2 := inv_testing.CreateRegion(t, nil)
	ou1 := inv_testing.CreateOu(t, nil)
	ou2 := inv_testing.CreateOu(t, nil)
	provider := inv_testing.CreateProvider(t, "TEST")
	site1 := inv_testing.CreateSite(t, region1, ou1)
	// Site2 is mostly used to reset fields
	site2 := inv_testing.CreateSiteWithArgs(t, "TEST", 0, 0, "", region1, ou1, provider)
	// Site3 mostly used for PUT-style update
	site3 := inv_testing.CreateSite(t, region1, ou1)

	putSite := location_v1.SiteResource{
		ResourceId:       site3.ResourceId,
		Name:             "Updated Name",
		Region:           region2,
		Ou:               ou2,
		Address:          "test",
		SiteLat:          800000000,
		SiteLng:          1700000000,
		DnsServers:       []string{"192.168.1.1"},
		DockerRegistries: []string{"test1", "test2"},
		MetricsEndpoint:  "test",
		HttpProxy:        "http://test.intel.com",
		HttpsProxy:       "https://test.intel.com",
		FtpProxy:         "ftp://test.intel.com",
		NoProxy:          "notest.intel.com",
		Metadata:         metaHost1,
	}

	testcases := map[string]struct {
		in           *location_v1.SiteResource
		resourceID   string
		fieldMask    *fieldmaskpb.FieldMask
		valid        bool
		expErrorCode codes.Code
	}{
		"UpdatePut": {
			in:           &putSite,
			resourceID:   site3.ResourceId,
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateName1": {
			in: &location_v1.SiteResource{
				Name: "Updated Name",
			},
			resourceID: site1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{siteresource.FieldName},
			},
			valid: true,
		},
		"UpdateName2": {
			in: &location_v1.SiteResource{
				Name: "Updated Name 2",
			},
			resourceID: site1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{siteresource.FieldName},
			},
			valid: true,
		},
		"UpdateParentRegion": {
			in: &location_v1.SiteResource{
				Region: region2,
			},
			resourceID: site1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{siteresource.EdgeRegion},
			},
			valid: true,
		},
		"UpdateResetParentRegion": {
			in:         &location_v1.SiteResource{},
			resourceID: site2.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{siteresource.EdgeRegion},
			},
			valid: true,
		},
		"UpdateParentOu": {
			in: &location_v1.SiteResource{
				Ou: ou1,
			},
			resourceID: site1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{siteresource.EdgeOu},
			},
			valid: true,
		},
		"UpdateResetParentOu": {
			in:         &location_v1.SiteResource{},
			resourceID: site2.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{siteresource.EdgeOu},
			},
			valid: true,
		},
		"UpdateProvider": {
			in: &location_v1.SiteResource{
				Provider: provider,
			},
			resourceID: site1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{siteresource.EdgeProvider},
			},
			valid: true,
		},
		"UpdateResetProvider": {
			in:         &location_v1.SiteResource{},
			resourceID: site2.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{siteresource.EdgeProvider},
			},
			valid: true,
		},
		"UpdateNoFieldMask": {
			in: &location_v1.SiteResource{
				Name: "Updated Name 4",
			},
			resourceID:   site1.ResourceId,
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateInvalidFieldMask1": {
			in: &location_v1.SiteResource{
				Name: "Updated Name 5",
			},
			resourceID: site1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"INVALID_FIELD"},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateInvalidMetadata": {
			in: &location_v1.SiteResource{
				Metadata: metaDuplicatedKeys,
			},
			resourceID: site1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{siteresource.FieldMetadata},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateResourceIDNotFound": {
			in: &location_v1.SiteResource{
				Name: "Updated Name",
			},
			resourceID: "site-12345678",
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{siteresource.FieldName},
			},
			valid:        false,
			expErrorCode: codes.NotFound,
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			updateresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Site{Site: tc.in},
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

func siteEdgesOnlyResourceID(expected *location_v1.SiteResource) {
	if expected.Ou != nil {
		expected.Ou = &ou_v1.OuResource{ResourceId: expected.Ou.ResourceId}
	}
	if expected.Region != nil {
		expected.Region = &location_v1.RegionResource{ResourceId: expected.Region.ResourceId}
	}
	if expected.Provider != nil {
		expected.Provider = &providerv1.ProviderResource{ResourceId: expected.Provider.ResourceId}
	}
}

func Test_FilterNestedSite(t *testing.T) {
	parentRegion := inv_testing.CreateRegion(t, nil)
	region1 := inv_testing.CreateRegion(t, parentRegion)
	site1 := inv_testing.CreateSite(t, region1, nil)
	site1.Region = region1
	parentOu := inv_testing.CreateOu(t, nil)
	ou1 := inv_testing.CreateOu(t, parentOu)
	site2 := inv_testing.CreateSite(t, nil, ou1)
	site2.Ou = ou1

	testcases := map[string]struct {
		in                *inv_v1.ResourceFilter
		resources         []*location_v1.SiteResource
		valid             bool
		expectedCodeError codes.Code
	}{
		"FilterByParentRegionID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s = %q`, siteresource.EdgeRegion, regionresource.EdgeParentRegion,
					regionresource.FieldResourceID, parentRegion.GetResourceId()),
			},
			resources: []*location_v1.SiteResource{site1},
			valid:     true,
		},
		"FilterByParentOuID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s = %q`, siteresource.EdgeOu, ouresource.EdgeParentOu,
					ouresource.FieldResourceID, parentOu.GetResourceId()),
			},
			resources: []*location_v1.SiteResource{site2},
			valid:     true,
		},
		"FailTooDeep": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s.%s.%s.%s = %q`, siteresource.EdgeOu, ouresource.EdgeParentOu,
					ouresource.EdgeParentOu, ouresource.EdgeParentOu, ouresource.EdgeParentOu,
					ouresource.EdgeParentOu, parentRegion.GetResourceId()),
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

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Site{}} // Set the resource kind

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
						"FilterSites() failed - want: %s, got: %s",
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

				resources := make([]*location_v1.SiteResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetSite())
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

func TestSiteMTSanity(t *testing.T) {
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	suite.Run(t, &struct{ mt }{
		mt: mt{
			createResource: func(tenantID string) (string, *inv_v1.Resource) {
				site := dao.CreateSite(t, tenantID)
				res, err := util.WrapResource(site)
				require.NoError(t, err)
				return site.GetResourceId(), res
			},
		},
	})
}

func TestDeleteResources_Sites(t *testing.T) {
	suite.Run(t, &struct{ hardDeleteAllResourcesSuite }{
		hardDeleteAllResourcesSuite: hardDeleteAllResourcesSuite{
			createModel: func(dao *inv_testing.InvResourceDAO) (string, int) {
				tenantID := uuid.NewString()
				return tenantID, len([]any{dao.CreateSiteNoCleanup(t, tenantID), dao.CreateSiteNoCleanup(t, tenantID)})
			},
			resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_SITE,
		},
	})
}
