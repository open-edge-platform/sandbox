// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package schedule

import (
	"context"
	"time"

	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	schedule_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/schedule/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
)

const hierarchyTimeout = 8 * time.Second

type HScheduleCacheClient struct {
	scc *ScheduleCacheClient
}

// NewHScheduleCacheClientWithOptions creates a client for the Inventory Service.
func NewHScheduleCacheClientWithOptions(
	ctx context.Context,
	opts ...Option,
) (*HScheduleCacheClient, error) {
	invHandler, err := NewScheduleCacheClientWithOptions(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return NewHScheduleCacheClient(invHandler)
}

// NewHScheduleCacheClient creates a client that wraps an existing ScheduleCacheClient. Mainly for testing.
func NewHScheduleCacheClient(scc *ScheduleCacheClient) (*HScheduleCacheClient, error) {
	return &HScheduleCacheClient{
		scc: scc,
	}, nil
}

// GetAllSingleSchedules this is just a decorator for GetSingleSchedules, it specifies DefaultFilter filter.
func (hsc *HScheduleCacheClient) GetAllSingleSchedules(ctx context.Context, tenantID string, offset, limit int) (
	sScheds []*schedule_v1.SingleScheduleResource,
	hasNext bool,
	totLen int,
	err error,
) {
	return hsc.GetSingleSchedules(ctx, tenantID, offset, limit, DefaultFilter)
}

// GetSingleSchedules returns a list of single schedules (including the inherited ones) and pagination info.
// The schedules are filtered using the params.
func (hsc *HScheduleCacheClient) GetSingleSchedules(ctx context.Context, tenantID string, offset, limit int, filters *Filters) (
	sScheds []*schedule_v1.SingleScheduleResource,
	hasNext bool,
	totLen int,
	err error,
) {
	logC.Debug().Msgf("GetSingleSchedules(tenantID=%s, offset=%d, limit=%d, filters=%v)", tenantID, offset, limit, filters)

	newFilters, err := hsc.getHierarchyAndExtendFilters(ctx, tenantID, filters)
	if err != nil {
		return nil, false, 0, err
	}

	filtered, next, total, err := hsc.scc.GetSchedules(
		inv_v1.ResourceKind_RESOURCE_KIND_SINGLESCHEDULE, tenantID, newFilters, offset, limit)
	if err != nil {
		return nil, false, 0, err
	}

	sScheds = collections.MapSlice[*inv_v1.Resource, *schedule_v1.SingleScheduleResource](
		filtered,
		func(r *inv_v1.Resource) *schedule_v1.SingleScheduleResource {
			return r.GetSingleschedule()
		})

	return sScheds, next, total, nil
}

func (hsc *HScheduleCacheClient) getHierarchyAndExtendFilters(ctx context.Context, tenantID string, filters *Filters) (
	[]*Filters, error,
) {
	if filters == nil {
		return []*Filters{DefaultFilter}, nil
	}
	resourceID, err := getResourceID(*filters)
	if err != nil {
		return nil, err
	}
	if resourceID == nil {
		// No resource ID filters, we skip getting the hierarchy.
		return []*Filters{filters}, nil
	}

	hierarchyFilters, err := hsc.getHierarchyFilters(ctx, tenantID, *resourceID)
	if err != nil {
		return nil, err
	}

	// Filter out all non-resource ID filters
	otherFilters := getNonResourceIDFilters(filters)

	collections.ForEach[*Filters](hierarchyFilters, func(fltrs *Filters) {
		fltrs.Append(otherFilters)
	})
	return hierarchyFilters, nil
}

func (hsc *HScheduleCacheClient) getHierarchyFilters(ctx context.Context, tenantID, resourceID string) ([]*Filters, error) {
	ctx, cancel := context.WithTimeout(ctx, hierarchyTimeout)
	defer cancel()
	tree, err := hsc.scc.InvClient.GetTreeHierarchy(ctx, &inv_v1.GetTreeHierarchyRequest{
		Filter:     []string{resourceID},
		Descending: true,
		TenantId:   tenantID,
	})
	if err != nil {
		logC.Debug().Msgf("Failed to get tree hierarchy for resourceID=%v", resourceID)
		logC.InfraErr(err).Msgf("Failed to get tree hierarchy")
		return nil, err
	}

	newFilters := make([]*Filters, 0, len(tree))
	collections.ForEach[*inv_v1.GetTreeHierarchyResponse_TreeNode](
		tree,
		func(node *inv_v1.GetTreeHierarchyResponse_TreeNode) {
			resID := node.GetCurrentNode().GetResourceId()
			switch node.GetCurrentNode().GetResourceKind() {
			case inv_v1.ResourceKind_RESOURCE_KIND_HOST:
				newFilters = append(newFilters, new(Filters).Add(HasHostID(&resID)))
			case inv_v1.ResourceKind_RESOURCE_KIND_SITE:
				newFilters = append(newFilters, new(Filters).Add(HasSiteID(&resID)))
			case inv_v1.ResourceKind_RESOURCE_KIND_REGION:
				newFilters = append(newFilters, new(Filters).Add(HasRegionID(&resID)))
			case inv_v1.ResourceKind_RESOURCE_KIND_OU:
				logC.Debug().Msgf("Got OU in the tree Hierarchy, skip!")
			default:
				logC.Warn().Msgf("Unexpected resource kind while traversing the tree: %v",
					node.GetCurrentNode().GetResourceKind())
			}
		})
	return newFilters, nil
}

// GetAllRepeatedSchedules this is just a decorator for GetRepeatedSchedules, it specifies DefaultFilter filter.
func (hsc *HScheduleCacheClient) GetAllRepeatedSchedules(ctx context.Context, tenantID string, offset, limit int) (
	rScheds []*schedule_v1.RepeatedScheduleResource,
	hasNext bool,
	totLen int,
	err error,
) {
	return hsc.GetRepeatedSchedules(ctx, tenantID, offset, limit, DefaultFilter)
}

// GetRepeatedSchedules returns a list of single schedules (including the inherited ones) and pagination info.
// The schedules are filtered using the params.
func (hsc *HScheduleCacheClient) GetRepeatedSchedules(ctx context.Context, tenantID string, offset, limit int, filters *Filters) (
	rScheds []*schedule_v1.RepeatedScheduleResource,
	hasNext bool,
	totLen int,
	err error,
) {
	logC.Debug().Msgf("GetRepeatedSchedules(tenantID=%s, offset=%d, limit=%d, filters=%v)", tenantID, offset, limit, filters)

	newFilters, err := hsc.getHierarchyAndExtendFilters(ctx, tenantID, filters)
	if err != nil {
		return nil, false, 0, err
	}

	filtered, next, total, err := hsc.scc.GetSchedules(
		inv_v1.ResourceKind_RESOURCE_KIND_REPEATEDSCHEDULE, tenantID, newFilters, offset, limit)
	if err != nil {
		return nil, false, 0, err
	}

	rScheds = collections.MapSlice[*inv_v1.Resource, *schedule_v1.RepeatedScheduleResource](
		filtered,
		func(r *inv_v1.Resource) *schedule_v1.RepeatedScheduleResource {
			return r.GetRepeatedschedule()
		})

	return rScheds, next, total, nil
}

// InvalidateCache invalidates the item in the cache using the ID and the resource type.
func (hsc *HScheduleCacheClient) InvalidateCache(
	tenantID, resourceID string, invalidateKind inv_v1.SubscribeEventsResponse_EventKind,
) {
	hsc.scc.InvalidateCache(tenantID, resourceID, invalidateKind)
}

// GetSingleSchedule returns the single schedule from the cache using the ID provided as input.
func (hsc *HScheduleCacheClient) GetSingleSchedule(tenantID, resourceID string) (
	*schedule_v1.SingleScheduleResource, error,
) {
	return hsc.scc.GetSingleSchedule(tenantID, resourceID)
}

// GetRepeatedSchedule returns the repeated schedule from the cache using the ID provided as input.
func (hsc *HScheduleCacheClient) GetRepeatedSchedule(tenantID, resourceID string) (
	*schedule_v1.RepeatedScheduleResource, error,
) {
	return hsc.scc.GetRepeatedSchedule(tenantID, resourceID)
}

// Close terminates the connection with Inventory and gracefully shutdown the cache.
func (hsc *HScheduleCacheClient) Close() error {
	return hsc.scc.InvClient.Close()
}
