// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/store"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
)

//nolint:funlen // long testing function due to hierarchy creation.
func Test_GetTreeHierarchy(t *testing.T) {
	// Create required Regions, OUs and Sites
	region0 := inv_testing.CreateRegion(t, nil)
	region1 := inv_testing.CreateRegion(t, region0)
	region2 := inv_testing.CreateRegion(t, region0)
	region3 := inv_testing.CreateRegion(t, region1)
	region4 := inv_testing.CreateRegion(t, region3)
	region5 := inv_testing.CreateRegion(t, region3)
	region6 := inv_testing.CreateRegion(t, nil)

	ou0 := inv_testing.CreateOu(t, nil)
	ou1 := inv_testing.CreateOu(t, ou0)
	ou2 := inv_testing.CreateOu(t, ou0)
	ou3 := inv_testing.CreateOu(t, ou1)
	ou4 := inv_testing.CreateOu(t, ou3)
	ou5 := inv_testing.CreateOu(t, ou3)
	ou6 := inv_testing.CreateOu(t, nil)

	site0 := inv_testing.CreateSite(t, region5, ou5)
	site1 := inv_testing.CreateSite(t, nil, ou4)
	site2 := inv_testing.CreateSite(t, region4, nil)
	site3 := inv_testing.CreateSite(t, region2, ou2)
	site4 := inv_testing.CreateSite(t, nil, nil)
	site5 := inv_testing.CreateSite(t, region6, ou6)

	inv_testing.CreateHost(t, site0, nil)
	inv_testing.CreateHost(t, site1, nil)
	inv_testing.CreateHost(t, site2, nil)
	host3 := inv_testing.CreateHost(t, site3, nil)
	host4 := inv_testing.CreateHost(t, site4, nil)
	host5 := inv_testing.CreateHost(t, site5, nil)
	type node struct {
		currentID string
		parents   []string
	}
	testcases := map[string]struct {
		request       *inv_v1.GetTreeHierarchyRequest
		checkOrder    bool
		expectedNodes []node
	}{
		"missingSite": {
			request: &inv_v1.GetTreeHierarchyRequest{
				Filter: []string{"site-00000000"},
			},
			checkOrder:    true,
			expectedNodes: []node{},
		},
		"missingHost": {
			request: &inv_v1.GetTreeHierarchyRequest{
				Filter: []string{"host-00000000"},
			},
			checkOrder:    true,
			expectedNodes: []node{},
		},
		"missingRegion": {
			request: &inv_v1.GetTreeHierarchyRequest{
				Filter: []string{"region-00000000"},
			},
			checkOrder:    true,
			expectedNodes: []node{},
		},
		"missingOU": {
			request: &inv_v1.GetTreeHierarchyRequest{
				Filter: []string{"ou-00000000"},
			},
			checkOrder:    true,
			expectedNodes: []node{},
		},
		"singleLevelRootToLeaf": {
			request: &inv_v1.GetTreeHierarchyRequest{
				Descending: false, // expected order is leaf to root
				Filter:     []string{host5.GetResourceId()},
			},
			checkOrder: true,
			expectedNodes: []node{
				{currentID: ou6.GetResourceId(), parents: []string{}},
				{currentID: region6.GetResourceId(), parents: []string{}},
				{currentID: site5.GetResourceId(), parents: []string{ou6.GetResourceId(), region6.GetResourceId()}},
				{currentID: host5.GetResourceId(), parents: []string{site5.GetResourceId()}},
			},
		},
		"singleLevelDescLeafToRoot": {
			request: &inv_v1.GetTreeHierarchyRequest{
				Descending: true, // expected order is root to leaf
				Filter:     []string{host5.GetResourceId()},
			},
			checkOrder: true,
			expectedNodes: []node{
				{currentID: host5.GetResourceId(), parents: []string{site5.GetResourceId()}},
				{currentID: site5.GetResourceId(), parents: []string{ou6.GetResourceId(), region6.GetResourceId()}},
				{currentID: ou6.GetResourceId(), parents: []string{}},
				{currentID: region6.GetResourceId(), parents: []string{}},
			},
		},
		"regions": {
			request: &inv_v1.GetTreeHierarchyRequest{
				Descending: false,
				Filter:     []string{region4.GetResourceId(), region2.GetResourceId()},
			},
			// We cannot check the order, it changes based on resource ID for nodes at the same depth
			expectedNodes: []node{
				{currentID: region4.GetResourceId(), parents: []string{region3.GetResourceId()}},
				{currentID: region3.GetResourceId(), parents: []string{region1.GetResourceId()}},
				{currentID: region1.GetResourceId(), parents: []string{region0.GetResourceId()}},
				{currentID: region2.GetResourceId(), parents: []string{region0.GetResourceId()}},
				{currentID: region0.GetResourceId(), parents: []string{}},
			},
		},
		"ous": {
			request: &inv_v1.GetTreeHierarchyRequest{
				Descending: false,
				Filter:     []string{ou4.GetResourceId(), ou2.GetResourceId()},
			},
			// We cannot check the order, it changes based on resource ID for nodes at the same depth
			expectedNodes: []node{
				{currentID: ou4.GetResourceId(), parents: []string{ou3.GetResourceId()}},
				{currentID: ou3.GetResourceId(), parents: []string{ou1.GetResourceId()}},
				{currentID: ou1.GetResourceId(), parents: []string{ou0.GetResourceId()}},
				{currentID: ou2.GetResourceId(), parents: []string{ou0.GetResourceId()}},
				{currentID: ou0.GetResourceId(), parents: []string{}},
			},
		},
		"sites": {
			request: &inv_v1.GetTreeHierarchyRequest{
				Filter: []string{site3.GetResourceId(), site4.GetResourceId(), site1.GetResourceId()},
			},
			// We cannot check the order, it changes based on resource ID for nodes at the same depth
			expectedNodes: []node{
				{currentID: site3.GetResourceId(), parents: []string{region2.GetResourceId(), ou2.GetResourceId()}},
				{currentID: region2.GetResourceId(), parents: []string{region0.GetResourceId()}},
				{currentID: region0.GetResourceId(), parents: []string{}},
				{currentID: ou2.GetResourceId(), parents: []string{ou0.GetResourceId()}},
				{currentID: ou0.GetResourceId(), parents: []string{}},
				{currentID: site4.GetResourceId(), parents: []string{}},
				{currentID: site1.GetResourceId(), parents: []string{ou4.GetResourceId()}},
				{currentID: ou4.GetResourceId(), parents: []string{ou3.GetResourceId()}},
				{currentID: ou3.GetResourceId(), parents: []string{ou1.GetResourceId()}},
				{currentID: ou1.GetResourceId(), parents: []string{ou0.GetResourceId()}},
			},
		},
		"hosts": {
			request: &inv_v1.GetTreeHierarchyRequest{
				Filter: []string{host5.GetResourceId(), host4.GetResourceId(), host3.GetResourceId()},
			},
			// We cannot check the order, it changes based on resource ID for nodes at the same depth
			expectedNodes: []node{
				{currentID: host5.GetResourceId(), parents: []string{site5.GetResourceId()}},
				{currentID: host4.GetResourceId(), parents: []string{site4.GetResourceId()}},
				{currentID: host3.GetResourceId(), parents: []string{site3.GetResourceId()}},
				{currentID: site5.GetResourceId(), parents: []string{ou6.GetResourceId(), region6.GetResourceId()}},
				{currentID: site4.GetResourceId(), parents: []string{}},
				{currentID: site3.GetResourceId(), parents: []string{ou2.GetResourceId(), region2.GetResourceId()}},
				{currentID: ou6.GetResourceId(), parents: []string{}},
				{currentID: ou2.GetResourceId(), parents: []string{ou0.GetResourceId()}},
				{currentID: ou0.GetResourceId(), parents: []string{}},
				{currentID: region6.GetResourceId(), parents: []string{}},
				{currentID: region2.GetResourceId(), parents: []string{region0.GetResourceId()}},
				{currentID: region0.GetResourceId(), parents: []string{}},
			},
		},

		"hostAndRegionAndOu": {
			request: &inv_v1.GetTreeHierarchyRequest{
				Filter: []string{host5.GetResourceId(), region2.GetResourceId(), region6.GetResourceId(), ou4.GetResourceId()},
			},
			// We cannot check the order, it changes based on resource ID for nodes at the same depth
			expectedNodes: []node{
				{currentID: host5.GetResourceId(), parents: []string{site5.GetResourceId()}},
				{currentID: site5.GetResourceId(), parents: []string{ou6.GetResourceId(), region6.GetResourceId()}},
				{currentID: ou6.GetResourceId(), parents: []string{}},
				{currentID: region6.GetResourceId(), parents: []string{}},
				{currentID: region2.GetResourceId(), parents: []string{region0.GetResourceId()}},
				{currentID: region0.GetResourceId(), parents: []string{}},
				{currentID: ou4.GetResourceId(), parents: []string{ou3.GetResourceId()}},
				{currentID: ou3.GetResourceId(), parents: []string{ou1.GetResourceId()}},
				{currentID: ou1.GetResourceId(), parents: []string{ou0.GetResourceId()}},
				{currentID: ou0.GetResourceId(), parents: []string{}},
			},
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			resp, err := inv_testing.TestClients[inv_testing.APIClient].GetTreeHierarchy(ctx, tc.request)
			require.NoError(t, err)
			nodes := collections.MapSlice[*inv_v1.GetTreeHierarchyResponse_TreeNode, node](
				resp,
				func(tN *inv_v1.GetTreeHierarchyResponse_TreeNode) node {
					return node{
						currentID: tN.GetCurrentNode().GetResourceId(),
						parents: collections.MapSlice[*inv_v1.GetTreeHierarchyResponse_Node, string](
							tN.GetParentNodes(),
							func(pN *inv_v1.GetTreeHierarchyResponse_Node) string {
								return pN.GetResourceId()
							}),
					}
				})
			if !tc.checkOrder {
				sort.Slice(nodes, func(i, j int) bool {
					return nodes[i].currentID > nodes[j].currentID
				})
				sort.Slice(tc.expectedNodes, func(i, j int) bool {
					return tc.expectedNodes[i].currentID > tc.expectedNodes[j].currentID
				})
			}
			collections.MapSlice[node, interface{}](nodes, func(n node) interface{} {
				sort.Strings(n.parents)
				return nil
			})
			collections.MapSlice[node, interface{}](tc.expectedNodes, func(n node) interface{} {
				sort.Strings(n.parents)
				return nil
			})
			assert.Equal(t, tc.expectedNodes, nodes)
		})
	}
}

func Test_GetTreeHierarchyNegative(t *testing.T) {
	testcases := map[string]struct {
		request    *inv_v1.GetTreeHierarchyRequest
		expErrCode codes.Code
	}{
		"wrongFilterID": {
			request: &inv_v1.GetTreeHierarchyRequest{
				Descending: false,
				Filter:     []string{"instance-12345678"},
			},
			expErrCode: codes.InvalidArgument,
		},
		"wrongMultipleFilterID": {
			request: &inv_v1.GetTreeHierarchyRequest{
				Descending: false,
				Filter:     []string{"host-12345678", "test-12345678", "instance-12345678"},
			},
			expErrCode: codes.InvalidArgument,
		},
		"wrongID": {
			request: &inv_v1.GetTreeHierarchyRequest{
				Descending: false,
				Filter:     []string{"testTestTest"},
			},
			expErrCode: codes.InvalidArgument,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			resp, err := inv_testing.TestClients[inv_testing.APIClient].GetTreeHierarchy(ctx, tc.request)
			require.Nil(t, resp)
			require.Error(t, err)
			gotStatus, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, tc.expErrCode, gotStatus.Code())
		})
	}
}

//nolint:funlen // long test function due to test cases
func Test_ValidateTreeHierarchyResponse(t *testing.T) {
	testcases := map[string]struct {
		response *inv_v1.GetTreeHierarchyResponse
		valid    bool
	}{
		"validParentHost": {
			response: &inv_v1.GetTreeHierarchyResponse{
				Tree: []*inv_v1.GetTreeHierarchyResponse_TreeNode{{
					CurrentNode: &inv_v1.GetTreeHierarchyResponse_Node{
						ResourceKind: inv_v1.ResourceKind_RESOURCE_KIND_HOST,
					},
					ParentNodes: []*inv_v1.GetTreeHierarchyResponse_Node{
						{
							ResourceKind: inv_v1.ResourceKind_RESOURCE_KIND_SITE,
						},
					},
				}},
			},
			valid: true,
		},
		"validParentSite": {
			response: &inv_v1.GetTreeHierarchyResponse{
				Tree: []*inv_v1.GetTreeHierarchyResponse_TreeNode{{
					CurrentNode: &inv_v1.GetTreeHierarchyResponse_Node{
						ResourceKind: inv_v1.ResourceKind_RESOURCE_KIND_SITE,
					},
					ParentNodes: []*inv_v1.GetTreeHierarchyResponse_Node{
						{ResourceKind: inv_v1.ResourceKind_RESOURCE_KIND_OU},
						{ResourceKind: inv_v1.ResourceKind_RESOURCE_KIND_REGION},
					},
				}},
			},
			valid: true,
		},
		"validParentRegion": {
			response: &inv_v1.GetTreeHierarchyResponse{
				Tree: []*inv_v1.GetTreeHierarchyResponse_TreeNode{{
					CurrentNode: &inv_v1.GetTreeHierarchyResponse_Node{
						ResourceKind: inv_v1.ResourceKind_RESOURCE_KIND_REGION,
					},
					ParentNodes: []*inv_v1.GetTreeHierarchyResponse_Node{
						{ResourceKind: inv_v1.ResourceKind_RESOURCE_KIND_REGION},
					},
				}},
			},
			valid: true,
		},
		"validParentOu": {
			response: &inv_v1.GetTreeHierarchyResponse{
				Tree: []*inv_v1.GetTreeHierarchyResponse_TreeNode{{
					CurrentNode: &inv_v1.GetTreeHierarchyResponse_Node{
						ResourceKind: inv_v1.ResourceKind_RESOURCE_KIND_OU,
					},
					ParentNodes: []*inv_v1.GetTreeHierarchyResponse_Node{
						{ResourceKind: inv_v1.ResourceKind_RESOURCE_KIND_OU},
					},
				}},
			},
			valid: true,
		},
		"invalidParentHost": {
			response: &inv_v1.GetTreeHierarchyResponse{
				Tree: []*inv_v1.GetTreeHierarchyResponse_TreeNode{{
					CurrentNode: &inv_v1.GetTreeHierarchyResponse_Node{
						ResourceKind: inv_v1.ResourceKind_RESOURCE_KIND_HOST,
					},
					ParentNodes: []*inv_v1.GetTreeHierarchyResponse_Node{
						{ResourceKind: inv_v1.ResourceKind_RESOURCE_KIND_REGION},
					},
				}},
			},
			valid: false,
		},
		"invalidParentSite": {
			response: &inv_v1.GetTreeHierarchyResponse{
				Tree: []*inv_v1.GetTreeHierarchyResponse_TreeNode{{
					CurrentNode: &inv_v1.GetTreeHierarchyResponse_Node{
						ResourceKind: inv_v1.ResourceKind_RESOURCE_KIND_SITE,
					},
					ParentNodes: []*inv_v1.GetTreeHierarchyResponse_Node{
						{ResourceKind: inv_v1.ResourceKind_RESOURCE_KIND_REGION},
						{ResourceKind: inv_v1.ResourceKind_RESOURCE_KIND_HOST},
					},
				}},
			},
			valid: false,
		},
		"invalidParentRegion": {
			response: &inv_v1.GetTreeHierarchyResponse{
				Tree: []*inv_v1.GetTreeHierarchyResponse_TreeNode{{
					CurrentNode: &inv_v1.GetTreeHierarchyResponse_Node{
						ResourceKind: inv_v1.ResourceKind_RESOURCE_KIND_REGION,
					},
					ParentNodes: []*inv_v1.GetTreeHierarchyResponse_Node{
						{ResourceKind: inv_v1.ResourceKind_RESOURCE_KIND_HOST},
					},
				}},
			},
			valid: false,
		},
		"invalidParentOu": {
			response: &inv_v1.GetTreeHierarchyResponse{
				Tree: []*inv_v1.GetTreeHierarchyResponse_TreeNode{{
					CurrentNode: &inv_v1.GetTreeHierarchyResponse_Node{
						ResourceKind: inv_v1.ResourceKind_RESOURCE_KIND_OU,
					},
					ParentNodes: []*inv_v1.GetTreeHierarchyResponse_Node{
						{ResourceKind: inv_v1.ResourceKind_RESOURCE_KIND_HOST},
					},
				}},
			},
			valid: false,
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			err := store.ValidateTreeHierarchyResponse(tc.response)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.ErrorContains(t, err, "invalid resource ID in the Tree hierarchy response")
			}
		})
	}
}

//nolint:funlen // long testing function due to hierarchy creation.
func Test_GetSitesPerRegion(t *testing.T) {
	// Create required Regions, OUs and Sites
	region0 := inv_testing.CreateRegion(t, nil)
	region1 := inv_testing.CreateRegion(t, region0)
	region2 := inv_testing.CreateRegion(t, region0)
	region3 := inv_testing.CreateRegion(t, region1)
	region4 := inv_testing.CreateRegion(t, region3)
	region5 := inv_testing.CreateRegion(t, region3)
	region6 := inv_testing.CreateRegion(t, nil)
	region7 := inv_testing.CreateRegion(t, nil)

	inv_testing.CreateSite(t, region2, nil)
	inv_testing.CreateSite(t, region4, nil)
	inv_testing.CreateSite(t, region5, nil)
	inv_testing.CreateSite(t, region6, nil)

	// Truth Table: regionID (parent of) -> total_sites (sites from...)
	// region0 (region1, region2) -> 3 sites (2-region1, 1-region2)
	// region1 (region3) -> 2 sites (2-region3)
	// region2 () -> 1 site (1-region2)
	// region3 (region4, region5) -> 2 sites (1-region4, 1-region5)
	// region4 () -> 1 site (region4)
	// region5 () -> 1 site (region5)
	// region6 () -> 1 site (region6)
	// region7 () -> 0 site ()

	type node struct {
		resourceID string
		childSites int
	}
	testcases := map[string]struct {
		request       *inv_v1.GetSitesPerRegionRequest
		expectedNodes []node
	}{
		"missingRegion": {
			request: &inv_v1.GetSitesPerRegionRequest{
				Filter: []string{"region-00000000"},
			},
			expectedNodes: []node{},
		},
		"allRegions": {
			request: &inv_v1.GetSitesPerRegionRequest{
				Filter: []string{
					region0.GetResourceId(),
					region1.GetResourceId(),
					region2.GetResourceId(),
					region3.GetResourceId(),
					region4.GetResourceId(),
					region5.GetResourceId(),
					region6.GetResourceId(),
					region7.GetResourceId(),
				},
			},
			expectedNodes: []node{
				{
					resourceID: region0.GetResourceId(),
					childSites: 3,
				},
				{
					resourceID: region1.GetResourceId(),
					childSites: 2,
				},
				{
					resourceID: region2.GetResourceId(),
					childSites: 1,
				},
				{
					resourceID: region3.GetResourceId(),
					childSites: 2,
				},
				{
					resourceID: region4.GetResourceId(),
					childSites: 1,
				},
				{
					resourceID: region5.GetResourceId(),
					childSites: 1,
				},
				{
					resourceID: region6.GetResourceId(),
					childSites: 1,
				},
				{
					resourceID: region7.GetResourceId(),
					childSites: 0,
				},
			},
		},
		"rootRegions": {
			request: &inv_v1.GetSitesPerRegionRequest{
				Filter: []string{
					region0.GetResourceId(),
					region6.GetResourceId(),
					region7.GetResourceId(),
				},
			},
			expectedNodes: []node{
				{
					resourceID: region0.GetResourceId(),
					childSites: 3,
				},
				{
					resourceID: region6.GetResourceId(),
					childSites: 1,
				},
				{
					resourceID: region7.GetResourceId(),
					childSites: 0,
				},
			},
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			resp, err := inv_testing.TestClients[inv_testing.APIClient].GetSitesPerRegion(ctx, tc.request)
			require.NoError(t, err)

			nodes := collections.MapSlice[*inv_v1.GetSitesPerRegionResponse_Node, node](
				resp.GetRegions(),
				func(tN *inv_v1.GetSitesPerRegionResponse_Node) node {
					return node{
						resourceID: tN.GetResourceId(),
						childSites: int(tN.GetChildSites()),
					}
				})

			sort.Slice(nodes, func(i, j int) bool {
				return nodes[i].resourceID < nodes[j].resourceID
			})
			sort.Slice(tc.expectedNodes, func(i, j int) bool {
				return tc.expectedNodes[i].resourceID < tc.expectedNodes[j].resourceID
			})
			assert.Equal(t, tc.expectedNodes, nodes)
		})
	}
}
