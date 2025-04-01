// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"

	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	inventoryv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
)

// Glossary:
// - standalone metadata: metadata belonging to the resource
// - inherited metadata: metadata belonging to the resources sitting in the upstream hierarchy of the resource
//    inherited metadata follow the rules for overlapping keys described in the documentation.
// - rendered metadata: metadata that applies to the resource, it's the merging of standalone and inherited metadata

var errUnexpectedTreeNode = errors.Errorfc(codes.Internal, "unexpected tree node type")

// Helper structure to pass around a resource and its own inherited meta.
type (
	resourceWithInheritedMeta[T ent.HostResource | ent.SiteResource | ent.RegionResource | ent.OuResource] struct {
		resource *T
		meta     inheritedMeta
	}
	hostWithInheritedMeta   resourceWithInheritedMeta[ent.HostResource]
	regionWithInheritedMeta resourceWithInheritedMeta[ent.RegionResource]
	ouWithInheritedMeta     resourceWithInheritedMeta[ent.OuResource]
	siteWithInheritedMeta   resourceWithInheritedMeta[ent.SiteResource]
)

// Helper structure to pack physical and logical metadata
// TODO: use more around the code.
type inheritedMeta struct {
	physical map[string]string
	logical  map[string]string
}

// Filter hosts by the given wantedMeta based on both standalone metadata
// and inherited metadata coming from both logical and physical hierarchy.
func filterHostsByMetadata(
	hostList map[int]*ent.HostResource,
	phyMeta map[int]map[string]string,
	logiMeta map[int]map[string]string,
	wantedMeta map[string]string,
) []*ent.HostResource {
	filteredHost := make([]*ent.HostResource, 0, len(hostList))
	for id, host := range hostList {
		stdMeta, err := ParseMetadata(host.Metadata)
		if err != nil {
			continue
		}
		// Any match on standalone or rendered metadata (both from logical or physical hierarchy).
		if matchMetadata(stdMeta, wantedMeta) ||
			matchMetadata(phyMeta[id], wantedMeta) ||
			matchMetadata(logiMeta[id], wantedMeta) {
			filteredHost = append(filteredHost, host)
		}
	}
	return filteredHost
}

func matchMetadata(metadata, wantedMeta map[string]string) bool {
	for key, val := range wantedMeta {
		if metadata[key] != val {
			return false
		}
	}
	return true
}

func getHostIDToHostMap(hosts []*ent.HostResource) map[int]*ent.HostResource {
	// Preallocate the map site
	hostIDs := make(map[int]*ent.HostResource, len(hosts))
	for _, h := range hosts {
		hostIDs[h.ID] = h
	}
	return hostIDs
}

// Optional tenantID parameter, in case of list/filter tenantID is not explicitly provided.
func getOusInheritedMeta(ctx context.Context, client *ent.Client, regionIDs []int, tenantIDs ...string) (
	logi map[int]map[string]string,
	err error,
) {
	isLeaf := func(t *treeNode) bool {
		return t.depth == 0
	}

	var tenantID *string
	if len(tenantIDs) > 0 {
		tenantID = &tenantIDs[0]
	}
	var tree []*treeNode
	tree, err = getHierarchyOus(ctx, client, descending, tenantID, regionIDs)
	if err != nil {
		return nil, err
	}
	logi = make(map[int]map[string]string)
	logiRenderedParent := make(map[int]map[string]string)
	ouIDs := make(map[int]interface{})
	for _, id := range regionIDs {
		ouIDs[id] = nil
	}
	// While traversing the tree from root to leaf, calculate the inherited metadata of the given ous
	for _, node := range tree {
		meta, err := ParseMetadata(node.metadata)
		if err != nil {
			return nil, err
		}
		switch node.nodeType {
		case inventoryv1.ResourceKind_RESOURCE_KIND_OU:
			if isLeaf(node) {
				handleMetaLeafNodeRegionOu(node, func(n *treeNode) *int { return n.ouParentID }, meta, logiRenderedParent, logi)
			} else {
				handleMetaTreeNodeRegionOu(node, func(n *treeNode) *int { return n.ouParentID }, meta, logiRenderedParent)
			}
		default:
			zlog.InfraSec().Err(errUnexpectedTreeNode).Send()
			return nil, errUnexpectedTreeNode
		}
	}
	return logi, nil
}

// Optional tenantID parameter, in case of list/filter tenantID is not explicitly provided.
func getRegionsInheritedMeta(ctx context.Context, client *ent.Client, regionIDs []int, tenantIDs ...string) (
	phy map[int]map[string]string,
	err error,
) {
	var tenantID *string
	if len(tenantIDs) > 0 {
		tenantID = &tenantIDs[0]
	}
	var tree []*treeNode
	tree, err = getHierarchyRegions(ctx, client, descending, tenantID, regionIDs)
	if err != nil {
		return nil, err
	}
	phy = make(map[int]map[string]string)
	phyRenderedParent := make(map[int]map[string]string)
	regIDs := make(map[int]interface{})
	for _, id := range regionIDs {
		regIDs[id] = nil
	}

	isLeaf := func(t *treeNode) bool {
		return t.depth == 0
	}

	// While traversing the tree from root to leaf, calculate the inherited metadata of the given regions
	for _, node := range tree {
		meta, err := ParseMetadata(node.metadata)
		if err != nil {
			return nil, err
		}
		switch node.nodeType {
		case inventoryv1.ResourceKind_RESOURCE_KIND_REGION:
			if isLeaf(node) {
				handleMetaLeafNodeRegionOu(
					node, func(n *treeNode) *int { return n.regionParentID }, meta, phyRenderedParent, phy)
			} else {
				handleMetaTreeNodeRegionOu(node, func(n *treeNode) *int { return n.regionParentID }, meta, phyRenderedParent)
			}
		default:
			zlog.InfraSec().Err(errUnexpectedTreeNode).Send()
			return nil, errUnexpectedTreeNode
		}
	}
	return phy, nil
}

// Optional tenantID parameter, in case of list/filter tenantID is not explicitly provided.
func getSitesInheritedMeta(ctx context.Context, client *ent.Client, siteIDs []int, tenantIDs ...string) (
	phy map[int]map[string]string,
	logi map[int]map[string]string,
	err error,
) {
	var tenantID *string
	if len(tenantIDs) > 0 {
		tenantID = &tenantIDs[0]
	}
	var tree []*treeNode
	tree, err = getHierarchySites(ctx, client, descending, tenantID, siteIDs)
	if err != nil {
		return nil, nil, err
	}
	phy = make(map[int]map[string]string)
	logi = make(map[int]map[string]string)
	phyRenderedParent := make(map[int]map[string]string)
	logiRenderedParent := make(map[int]map[string]string)
	siteMap := make(map[int]inheritedMeta)
	// While traversing the tree from root to leaf, calculate the inherited metadata of the given sites
	for _, node := range tree {
		meta, err := ParseMetadata(node.metadata)
		if err != nil {
			return nil, nil, err
		}
		switch node.nodeType {
		case inventoryv1.ResourceKind_RESOURCE_KIND_REGION:
			handleMetaTreeNodeRegionOu(node, func(n *treeNode) *int { return n.regionParentID }, meta, phyRenderedParent)
		case inventoryv1.ResourceKind_RESOURCE_KIND_OU:
			handleMetaTreeNodeRegionOu(node, func(n *treeNode) *int { return n.ouParentID }, meta, logiRenderedParent)
		case inventoryv1.ResourceKind_RESOURCE_KIND_SITE:
			handleMetaLeafNodeSite(node, meta, phyRenderedParent, logiRenderedParent, siteMap, phy, logi)
		default:
			zlog.InfraSec().Err(errUnexpectedTreeNode).Msgf("nodeType=%v", node.nodeType)
			return nil, nil, errUnexpectedTreeNode
		}
	}
	return phy, logi, nil
}

// TODO: return map of inheritedMeta.
// Optional tenantID parameter, in case of list/filter tenantID is not explicitly provided.
func getHostsInheritedMeta(ctx context.Context, client *ent.Client, hostIDs []int, tenantIDs ...string) (
	phy map[int]map[string]string,
	logi map[int]map[string]string,
	err error,
) {
	var tenantID *string
	if len(tenantIDs) > 0 {
		tenantID = &tenantIDs[0]
	}
	var tree []*treeNode
	tree, err = getHierarchyHosts(ctx, client, descending, tenantID, hostIDs)
	if err != nil {
		return nil, nil, err
	}
	phy = make(map[int]map[string]string)
	logi = make(map[int]map[string]string)
	parentRegRendered := make(map[int]map[string]string)
	parentOuRendered := make(map[int]map[string]string)
	parentSiteRendered := make(map[int]inheritedMeta)
	// While traversing the tree from root to leaf, calculate the inherited metadata of the given hosts
	for _, node := range tree {
		meta, err := ParseMetadata(node.metadata)
		if err != nil {
			return nil, nil, err
		}
		switch node.nodeType {
		case inventoryv1.ResourceKind_RESOURCE_KIND_REGION:
			handleMetaTreeNodeRegionOu(node, func(t *treeNode) *int { return t.regionParentID }, meta, parentRegRendered)
		case inventoryv1.ResourceKind_RESOURCE_KIND_OU:
			handleMetaTreeNodeRegionOu(node, func(t *treeNode) *int { return t.ouParentID }, meta, parentOuRendered)
		case inventoryv1.ResourceKind_RESOURCE_KIND_SITE:
			handleMetaTreeNodeSite(node, meta, parentRegRendered, parentOuRendered, parentSiteRendered)
		case inventoryv1.ResourceKind_RESOURCE_KIND_HOST:
			handleMetaLeafNodeHost(node, meta, parentSiteRendered, phy, logi)
		default:
			zlog.InfraSec().Err(errUnexpectedTreeNode).Send()
			return nil, nil, errUnexpectedTreeNode
		}
	}
	return phy, logi, nil
}

// handleMetaTreeNodeRegionOu handles the node of the tree for a region or OU, and calculate the rendered metadata
// for that node, given the one of the parent in the hierarchy and its standalone metadata.
func handleMetaTreeNodeRegionOu(
	node *treeNode,
	getParentID func(*treeNode) *int,
	standaloneMeta map[string]string,
	parentRenderedMap map[int]map[string]string,
) {
	parentID := getParentID(node)
	if parentID == nil && standaloneMeta != nil {
		parentRenderedMap[node.ID] = standaloneMeta
	} else {
		// How much expensive are these ops? should we optimize them?
		if parentMeta, ok := parentRenderedMap[*parentID]; ok {
			phyRenderedMeta := maps.Clone(parentMeta)
			maps.Copy(phyRenderedMeta, standaloneMeta)
			parentRenderedMap[node.ID] = phyRenderedMeta
		}
	}
}

// handleMetaTreeNodeSite handles the node of the tree for a site, and calculate the rendered metadata for that node,
// given the metadata of the parent region and OU in the hierarchy and its standalone metadata.
func handleMetaTreeNodeSite(
	node *treeNode,
	standaloneMeta map[string]string,
	parentRenderedRegMap map[int]map[string]string,
	parentRenderedOuMap map[int]map[string]string,
	siteRenderedMap map[int]inheritedMeta,
) {
	ouParentID := node.ouParentID
	regParentID := node.regionParentID
	var logiRenderedMeta map[string]string
	phyRenderedMeta := standaloneMeta

	if regParentID != nil {
		if regMeta, ok := parentRenderedRegMap[*regParentID]; ok {
			phyRenderedMeta = maps.Clone(regMeta)
			// merge standalone site metadata with physical hierarchy metadata
			maps.Copy(phyRenderedMeta, standaloneMeta)
		}
	}
	if ouParentID != nil {
		if ouMeta, ok := parentRenderedOuMap[*ouParentID]; ok {
			logiRenderedMeta = maps.Clone(ouMeta)
			// remove phy inherited metadata from the logical one
			mapsDifference(logiRenderedMeta, phyRenderedMeta)
		}
	}
	siteRenderedMap[node.ID] = inheritedMeta{phyRenderedMeta, logiRenderedMeta}
}

func calculateInheritedMetadata(
	nodeID int,
	standaloneMeta map[string]string,
	parentMeta map[string]string,
	inheritedMetaMap map[int]map[string]string,
) {
	inhMeta := maps.Clone(parentMeta)
	mapsDifference(inhMeta, standaloneMeta)
	inheritedMetaMap[nodeID] = inhMeta
}

// handleMetaLeafNodeRegionOu handles the leaf node in the tree for region and OU.
// Calculates the inherited metadata of the node given the hierarchy by removing standalone metadata from the rendered
// metadata of the parent region or OU.
func handleMetaLeafNodeRegionOu(
	node *treeNode,
	getParentID func(*treeNode) *int,
	standaloneMeta map[string]string,
	parentRenderedMap map[int]map[string]string,
	inheritedMetaMap map[int]map[string]string,
) {
	parentID := getParentID(node)
	if parentID != nil {
		if parentMeta, ok := parentRenderedMap[*parentID]; ok {
			calculateInheritedMetadata(node.ID, standaloneMeta, parentMeta, inheritedMetaMap)
		}
	}
}

// handleMetaLeafNodeSite handles the leaf node in the tree for site.
// Calculates the inherited metadata of the node given the hierarchy by removing standalone metadata from the rendered
// metadata of the parent region and OU.
func handleMetaLeafNodeSite(
	node *treeNode,
	standaloneMeta map[string]string,
	parentRegionRenderedMap map[int]map[string]string,
	parentOuRenderedMap map[int]map[string]string,
	siteRenderedMap map[int]inheritedMeta,
	phyInheritedMetaMap map[int]map[string]string,
	logiInheritedMetaMap map[int]map[string]string,
) {
	handleMetaTreeNodeSite(node, standaloneMeta, parentRegionRenderedMap, parentOuRenderedMap, siteRenderedMap)
	siteID := node.ID
	// Remove standalone meta also from phy meta
	calculateInheritedMetadata(node.ID, standaloneMeta, siteRenderedMap[node.ID].physical, phyInheritedMetaMap)
	// Logical metadata are already rendered properly
	logiInheritedMetaMap[siteID] = siteRenderedMap[node.ID].logical
}

// handleMetaLeafNodeSite handles the leaf node in the tree for host.
// Calculates the inherited metadata of the node given the hierarchy by removing standalone metadata from the rendered
// metadata of the parent site.
func handleMetaLeafNodeHost(
	node *treeNode,
	standaloneMeta map[string]string,
	parentRenderedMap map[int]inheritedMeta,
	phyInheritedMetaMap map[int]map[string]string,
	logiInheritedMetaMap map[int]map[string]string,
) {
	siteParentID := node.siteParentID
	if siteParentID != nil {
		if parentMeta, ok := parentRenderedMap[*siteParentID]; ok {
			calculateInheritedMetadata(node.ID, standaloneMeta, parentMeta.physical, phyInheritedMetaMap)
			calculateInheritedMetadata(node.ID, standaloneMeta, parentMeta.logical, logiInheritedMetaMap)
		}
	}
}
