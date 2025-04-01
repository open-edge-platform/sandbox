// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"google.golang.org/protobuf/proto"

	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
)

// storeHostResourceInfo caches the host Resource info.
func (c *InventoryCache) storeHostResourceInfo(host *computev1.HostResource) {
	hostID := host.ResourceId
	tenantID := host.GetTenantId()
	hostCopy := proto.Clone(host)
	c.storeCacheEntry(tenantIDBasedKey{tenantID, hostID}, hostCopy)
}

func (c *InventoryCache) getHostResourceByID(tenantID, resourceID string) (*inv_v1.GetResourceResponse, bool) {
	// get resource from Cache.
	zlog.Debug().Msgf("get host resource with resource ID <%v>", resourceID)
	if val, ok := c.getCacheEntry(tenantIDBasedKey{tenantID, resourceID}); ok {
		// create resource rsp.
		if r, ok := val.(*computev1.HostResource); ok {
			zlog.Debug().Msgf("found host resource with resource ID <%v>", r.GetResourceId())
			// make copy and return.
			if rCopy, ok := proto.Clone(r).(*computev1.HostResource); ok {
				return &inv_v1.GetResourceResponse{
					Resource: &inv_v1.Resource{
						Resource: &inv_v1.Resource_Host{
							Host: rCopy,
						},
					},
				}, ok
			}
		}
	}
	return nil, false
}

func (c *InventoryCache) invalidateHostResourceInfo(hr *computev1.HostResource) {
	// delete host resource info from cache.
	c.deleteCacheEntry(tenantIDBasedKey{hr.GetTenantId(), hr.GetResourceId()})

	// later- delete host resources 1st level relations from cache too
}
