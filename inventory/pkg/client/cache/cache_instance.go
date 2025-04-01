// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"google.golang.org/protobuf/proto"

	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
)

// storeInstanceResourceInfo caches the instance Resource info.
func (c *InventoryCache) storeInstanceResourceInfo(r *computev1.InstanceResource) {
	resID := r.ResourceId
	tenantID := r.GetTenantId()
	rCopy := proto.Clone(r)
	c.storeCacheEntry(tenantIDBasedKey{tenantID, resID}, rCopy)
}

func (c *InventoryCache) getInstanceResourceByID(tenantID, resourceID string) (*inv_v1.GetResourceResponse, bool) {
	// get resource from Cache.
	if val, ok := c.getCacheEntry(tenantIDBasedKey{tenantID, resourceID}); ok {
		// create resource rsp.
		if r, ok := val.(*computev1.InstanceResource); ok {
			zlog.Debug().Msgf("found host resource with resource ID <%v>", r.GetResourceId())
			// make copy and return.
			if rCopy, ok := proto.Clone(r).(*computev1.InstanceResource); ok {
				return &inv_v1.GetResourceResponse{
					Resource: &inv_v1.Resource{
						Resource: &inv_v1.Resource_Instance{
							Instance: rCopy,
						},
					},
				}, ok
			}
		}
	}
	return nil, false
}

func (c *InventoryCache) invalidateInstanceResourceInfo(r *computev1.InstanceResource) {
	// delete instance resource info from cache.
	c.deleteCacheEntry(tenantIDBasedKey{r.GetTenantId(), r.GetResourceId()})

	// delete instance resources 1st level relations from cache too
	// host relation
	if v := r.GetHost(); v != nil {
		c.deleteCacheEntry(tenantIDBasedKey{v.GetTenantId(), v.GetResourceId()})
	}
}
