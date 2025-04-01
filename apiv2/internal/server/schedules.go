// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"

	"google.golang.org/grpc/codes"

	restv1 "github.com/open-edge-platform/infra-core/apiv2/v2/internal/pbapi/services/v1"
	schedule_cache "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client/cache/schedule"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/function"
)

func getSchedulesFilter(hostID, siteID, regionID *string) (
	schedFilters *schedule_cache.Filters, nonEmptyFilters int,
) {
	schedFilters = new(schedule_cache.Filters)
	nonEmptyFilters = 0
	if function.IsEmptyNullCase(regionID) {
		schedFilters.Add(schedule_cache.HasNoRegion())
	} else if function.IsNotEmptyNullCase(regionID) && isSet(regionID) {
		schedFilters.Add(schedule_cache.HasRegionID(regionID))
		nonEmptyFilters++
	}

	if function.IsEmptyNullCase(siteID) {
		schedFilters.Add(schedule_cache.HasNoSite())
	} else if function.IsNotEmptyNullCase(siteID) && isSet(siteID) {
		schedFilters.Add(schedule_cache.HasSiteID(siteID))
		nonEmptyFilters++
	}

	if function.IsEmptyNullCase(hostID) {
		schedFilters.Add(schedule_cache.HasNoHost())
	} else if function.IsNotEmptyNullCase(hostID) && isSet(hostID) {
		schedFilters.Add(schedule_cache.HasHostID(hostID))
		nonEmptyFilters++
	}
	return schedFilters, nonEmptyFilters
}

func parseSchedulesFilter(hostID, siteID, regionID, unixEpoch *string) (*schedule_cache.Filters, error) {
	schedFilters, nonEmptyFilters := getSchedulesFilter(hostID, siteID, regionID)
	// Cannot apply multiple filters
	if nonEmptyFilters > 1 {
		return nil,
			errors.Errorfc(codes.InvalidArgument, "Only one of TargetHost, TargetSite and TargetRegion can be specified")
	}
	// No filters
	if schedFilters.Size() == 0 {
		schedFilters.Add(schedule_cache.NewStandardFilter(nil, "All"))
	}

	// unixEpoch is optional string (empty can be defined as "")
	if isUnset(unixEpoch) {
		schedFilters.Add(schedule_cache.FilterByTS(nil))
	} else {
		schedFilters.Add(schedule_cache.FilterByTS(unixEpoch))
	}
	return schedFilters, nil
}

// Get a list of Schedules (single/schedule).
func (is *InventorygRPCServer) ListSchedules(
	ctx context.Context,
	req *restv1.ListSchedulesRequest,
) (*restv1.ListSchedulesResponse, error) {
	zlog.Debug().Msg("ListSchedules")

	singleReq := &restv1.ListSingleSchedulesRequest{
		HostId:    req.GetHostId(),
		SiteId:    req.GetSiteId(),
		RegionId:  req.GetRegionId(),
		UnixEpoch: req.GetUnixEpoch(),
		PageSize:  req.GetPageSize(),
		Offset:    req.GetOffset(),
	}
	zlog.Debug().Msgf("ListSingleSchedules %s", singleReq)
	singleSchedules, err := is.ListSingleSchedules(ctx, singleReq)
	if err != nil {
		return nil, err
	}

	repeatReq := &restv1.ListRepeatedSchedulesRequest{
		HostId:    req.GetHostId(),
		SiteId:    req.GetSiteId(),
		RegionId:  req.GetRegionId(),
		UnixEpoch: req.GetUnixEpoch(),
		PageSize:  req.GetPageSize(),
		Offset:    req.GetOffset(),
	}
	zlog.Debug().Msgf("ListRepeatedSchedules %s", repeatReq)
	repeatedSchedules, err := is.ListRepeatedSchedules(ctx, repeatReq)
	if err != nil {
		return nil, err
	}

	resp := &restv1.ListSchedulesResponse{
		SingleSchedules:   singleSchedules.GetSingleSchedules(),
		RepeatedSchedules: repeatedSchedules.GetRepeatedSchedules(),
		TotalElements:     singleSchedules.GetTotalElements() + repeatedSchedules.GetTotalElements(),
		HasNext:           singleSchedules.GetHasNext() || repeatedSchedules.GetHasNext(),
	}
	zlog.Debug().Msgf("Listed %s", resp)
	return resp, nil
}
