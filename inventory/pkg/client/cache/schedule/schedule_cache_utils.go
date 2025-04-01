// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package schedule

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/adhocore/gronx"
	"google.golang.org/grpc/codes"

	compute_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	location_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/location/v1"
	schedulev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/schedule/v1"
	inverr "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/function"
)

// DefaultFilter is a filter accepting any schedule.
var DefaultFilter = new(Filters).Add(
	StandardFilter{
		desc: "All",
	})

// Filters struct represent set of filters applicable on collection of schedules.
type Filters struct {
	filters []Filter
	errs    []error
}

// String returns information about registered filters.
func (s *Filters) String() string {
	descriptions := collections.MapSlice[Filter, string](s.filters, func(filter Filter) string {
		return filter.GetDescription()
	})
	return fmt.Sprintf("Filters: %s", strings.Join(descriptions, ":"))
}

// Add register given filter f with description specified by name, end errors throw by filter builder.
func (s *Filters) Add(f Filter, errs ...error) *Filters {
	s.filters = append(s.filters, f)

	s.errs = append(
		s.errs,
		collections.Filter[error](errs, func(e error) bool {
			return e != nil
		})...)
	return s
}

// Evaluate evaluates registered filters, evaluation is executed base on conjunction (AND)
// returns true when:
// - all registered filters will return true
// - there is zero registered filters.
func (s *Filters) Evaluate(r *inv_v1.Resource) bool {
	matches := true
	for _, filter := range s.filters {
		if filter == nil || filter.GetFilterFunc() == nil {
			continue
		}
		matches = matches && filter.GetFilterFunc()(r)
	}

	return matches
}

func (s *Filters) Validate() error {
	if len(s.errs) > 0 {
		return inverr.Errorfc(codes.InvalidArgument, "cannot build concatenated filters: %v", errors.Join(s.errs...))
	}
	return nil
}

func (s *Filters) Size() int {
	return len(s.filters)
}

// Append appends the given filter to the current one. Also, errors are propagated to the current filter.
func (s *Filters) Append(otherFilters *Filters) {
	s.filters = append(s.filters, otherFilters.filters...)
	s.errs = append(s.errs, otherFilters.errs...)
}

type FilterFunc func(resource *inv_v1.Resource) bool

type Filter interface {
	GetFilterFunc() FilterFunc
	GetDescription() string
}

// StandardFilter is the generic filter containing a filter function and a description.
type StandardFilter struct {
	filter FilterFunc
	desc   string
}

// NewStandardFilter creates a new filter with the given filter function and description.
func NewStandardFilter(filter FilterFunc, desc string) *StandardFilter {
	return &StandardFilter{filter, desc}
}

func (sf StandardFilter) GetDescription() string {
	return sf.desc
}

func (sf StandardFilter) GetFilterFunc() FilterFunc {
	return sf.filter
}

// ResourceIDFilter is the specific filter matching on a Resource ID.
type ResourceIDFilter struct {
	StandardFilter
	resourceID string
}

func (idf ResourceIDFilter) GetFilterFunc() FilterFunc {
	return idf.filter
}

func (idf ResourceIDFilter) GetDescription() string {
	return idf.desc
}

func (idf ResourceIDFilter) GetResourceID() string {
	return idf.resourceID
}

func IsSingle(r *inv_v1.Resource) bool {
	return r.GetSingleschedule() != nil
}

func IsRepeated(r *inv_v1.Resource) bool {
	return r.GetRepeatedschedule() != nil
}

// HasNoTargets returns all schedules given the schedule kind and not having a target.
func HasNoTargets() Filter {
	return StandardFilter{
		func(r *inv_v1.Resource) bool {
			if IsSingle(r) && hasZeroTargets(r, extractSSR) {
				return true
			}
			if IsRepeated(r) && hasZeroTargets(r, extractRSR) {
				return true
			}
			return false
		},
		"hasNoTargets",
	}
}

// HasRegionID returns all schedules given the schedule kind and having as target the provided regionID.
func HasRegionID(regionID *string) Filter {
	if regionID == nil {
		return nil
	}
	return ResourceIDFilter{
		StandardFilter{
			func(r *inv_v1.Resource) bool {
				switch {
				case IsSingle(r):
					return hasRegionTarget(r, extractSSR, *regionID)
				case IsRepeated(r):
					return hasRegionTarget(r, extractRSR, *regionID)
				default:
					return false
				}
			},
			"hasRegionID",
		},
		*regionID,
	}
}

// HasNoRegion returns all schedules without a target Region.
func HasNoRegion() Filter {
	return StandardFilter{
		func(r *inv_v1.Resource) bool {
			switch {
			case IsSingle(r):
				return hasNoRegionTarget(r, extractSSR)
			case IsRepeated(r):
				return hasNoRegionTarget(r, extractRSR)
			default:
				return false
			}
		},
		"hasNoRegion",
	}
}

// HasSiteID returns all schedules given the schedule kind and having as target the provided siteID.
func HasSiteID(siteID *string) Filter {
	if siteID == nil {
		return nil
	}
	return ResourceIDFilter{
		StandardFilter{
			func(r *inv_v1.Resource) bool {
				switch {
				case IsSingle(r):
					return hasSiteTarget(r, extractSSR, *siteID)
				case IsRepeated(r):
					return hasSiteTarget(r, extractRSR, *siteID)
				default:
					return false
				}
			},
			"hasSiteID",
		},
		*siteID,
	}
}

// HasNoSite returns all schedules without a target Site.
func HasNoSite() Filter {
	return StandardFilter{
		func(r *inv_v1.Resource) bool {
			switch {
			case IsSingle(r):
				return hasNoSiteTarget(r, extractSSR)
			case IsRepeated(r):
				return hasNoSiteTarget(r, extractRSR)
			default:
				return false
			}
		},
		"hasNoSite",
	}
}

// HasHostID returns all schedules given the schedule kind and having as target the provided hostID.
func HasHostID(hostID *string) Filter {
	if hostID == nil {
		return nil
	}
	return ResourceIDFilter{
		StandardFilter{
			func(r *inv_v1.Resource) bool {
				switch {
				case IsSingle(r):
					return hasHostTarget(r, extractSSR, *hostID)
				case IsRepeated(r):
					return hasHostTarget(r, extractRSR, *hostID)
				default:
					return false
				}
			},
			"hasHostID",
		},
		*hostID,
	}
}

// HasNoHost returns all schedules without target Host.
func HasNoHost() Filter {
	return StandardFilter{
		func(r *inv_v1.Resource) bool {
			switch {
			case IsSingle(r):
				return hasNoHostTarget(r, extractSSR)
			case IsRepeated(r):
				return hasNoHostTarget(r, extractRSR)
			default:
				return false
			}
		},
		"hasNoHost",
	}
}

// FilterByTS filters the given schedules based on the given timestamp if a valid timestamp is provided.
func FilterByTS(ts *string) (Filter, error) {
	if ts == nil {
		return StandardFilter{
			filter: func(_ *inv_v1.Resource) bool {
				return true
			},
		}, nil
	}

	timestamp, err := parseTimestamp(*ts)
	if err != nil {
		return nil, err
	}

	return StandardFilter{
		func(r *inv_v1.Resource) bool {
			switch {
			case IsSingle(r):
				singleSched := r.GetSingleschedule()
				if singleSched == nil {
					return false
				}
				return ssMatchesTimestamp(singleSched, timestamp)
			case IsRepeated(r):
				repeatedSched := r.GetRepeatedschedule()
				if repeatedSched == nil {
					return false
				}
				match := rsMatchesTimestamp(repeatedSched, timestamp)
				if !match {
					logC.Info().Msgf("FilterByTS: DROP: schedule(%v) does not match to ts(%s)", repeatedSched, timestamp)
				}
				return match
			}
			return false
		},
		"ByTimestamp",
	}, nil
}

// utility function to parse the timestamp.
func parseTimestamp(timestamp string) (time.Time, error) {
	timeInt64, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		logC.InfraErr(err).Msgf("parsing timestamp")
		return time.Time{}, inverr.Wrap(err)
	}
	return time.Unix(timeInt64, 0), nil
}

func ssMatchesTimestamp(ss *schedulev1.SingleScheduleResource, ts time.Time) bool {
	int64StartSeconds, err := util.Uint64ToInt64(ss.StartSeconds)
	if err != nil {
		logC.InfraErr(err).Msgf("parsing timestamp start seconds")
		return false
	}
	int64EndSeconds, err := util.Uint64ToInt64(ss.EndSeconds)
	if err != nil {
		logC.InfraErr(err).Msgf("parsing timestamp end seconds")
		return false
	}
	startTime := time.Unix(int64StartSeconds, 0)
	endTime := time.Unix(int64EndSeconds, 0)
	logC.Debug().Msgf("Single Sched: start=%v, end=%v, currentTime=%v", startTime, endTime, ts)
	if ts.After(startTime) || ts.Equal(startTime) {
		if endTime.Unix() == 0 || ts.Before(endTime) || ts.Equal(endTime) {
			return true
		}
	}
	return false
}

func rsMatchesTimestamp(rs *schedulev1.RepeatedScheduleResource, ts time.Time) bool {
	cron := fmt.Sprintf(
		"%s %s %s %s %s",
		rs.CronMinutes,
		rs.CronHours,
		rs.CronDayMonth,
		rs.CronMonth,
		rs.CronDayWeek,
	)
	prevStartTime, err := gronx.PrevTickBefore(cron, ts, false)
	if err != nil {
		logC.InfraErr(err).Msgf(
			"invalid cron for repeated schedule, skipping: repeatedSched=%v, cron=%s",
			rs,
			cron,
		)
		return false
	}
	prevEndTime := prevStartTime.Add(time.Duration(rs.DurationSeconds) * time.Second)
	logC.Debug().Msgf(
		"RepeatedSched: cron=%s, prevStar=%v, prevEnd=%v, currTime=%v",
		cron,
		prevStartTime,
		prevEndTime,
		ts,
	)
	if (ts.After(prevStartTime) || ts.Equal(prevStartTime)) &&
		(ts.Before(prevEndTime) || ts.Equal(prevEndTime)) {
		// the given time is within the schedule
		return true
	}
	return false
}

type scheduleTargetCarrier interface {
	GetTargetHost() *compute_v1.HostResource
	GetTargetSite() *location_v1.SiteResource
	GetTargetRegion() *location_v1.RegionResource
	GetResourceId() string
}

func hasZeroTargets(res *inv_v1.Resource, getTargetCarrier func(*inv_v1.Resource) scheduleTargetCarrier) bool {
	stc := getTargetCarrier(res)
	if stc == nil {
		return false
	}
	targets := []any{stc.GetTargetHost(), stc.GetTargetSite(), stc.GetTargetRegion()}
	return len(collections.Filter(targets, function.IsNil)) == len(targets)
}

func hasRegionTarget(res *inv_v1.Resource, getTargetCarrier func(*inv_v1.Resource) scheduleTargetCarrier, regionID string) bool {
	stc := getTargetCarrier(res)
	if stc == nil {
		return false
	}
	if stc.GetTargetRegion() != nil && stc.GetTargetRegion().ResourceId == regionID {
		return true
	}
	if function.IsEmptyNullCase(&regionID) && stc.GetTargetRegion() == nil {
		return true
	}
	return false
}

func hasNoRegionTarget(res *inv_v1.Resource, getTargetCarrier func(*inv_v1.Resource) scheduleTargetCarrier) bool {
	stc := getTargetCarrier(res)
	if stc == nil {
		return false
	}
	if stc.GetTargetRegion() == nil {
		return true
	}
	return false
}

func hasSiteTarget(res *inv_v1.Resource, getTargetCarrier func(*inv_v1.Resource) scheduleTargetCarrier, siteID string) bool {
	stc := getTargetCarrier(res)
	if stc == nil {
		return false
	}
	if stc.GetTargetSite() != nil && stc.GetTargetSite().ResourceId == siteID {
		return true
	}
	if function.IsEmptyNullCase(&siteID) && stc.GetTargetSite() == nil {
		return true
	}
	return false
}

func hasNoSiteTarget(res *inv_v1.Resource, getTargetCarrier func(*inv_v1.Resource) scheduleTargetCarrier) bool {
	stc := getTargetCarrier(res)
	if stc == nil {
		return false
	}
	if stc.GetTargetSite() == nil {
		return true
	}
	return false
}

func hasHostTarget(res *inv_v1.Resource, getTargetCarrier func(*inv_v1.Resource) scheduleTargetCarrier, hostID string) bool {
	stc := getTargetCarrier(res)
	if stc == nil {
		return false
	}
	if stc.GetTargetHost() != nil && stc.GetTargetHost().ResourceId == hostID {
		return true
	}
	if function.IsEmptyNullCase(&hostID) && stc.GetTargetHost() == nil {
		return true
	}
	return false
}

func hasNoHostTarget(res *inv_v1.Resource, getTargetCarrier func(*inv_v1.Resource) scheduleTargetCarrier) bool {
	stc := getTargetCarrier(res)
	if stc == nil {
		return false
	}

	if stc.GetTargetHost() == nil {
		return true
	}
	return false
}

func extractSSR(res *inv_v1.Resource) scheduleTargetCarrier {
	return res.GetSingleschedule()
}

func extractRSR(res *inv_v1.Resource) scheduleTargetCarrier {
	return res.GetRepeatedschedule()
}

// getResourceID returns the resourceID from the given filters, we expect a single filter by resource ID to be present,
// otherwise an InvalidArgument error is thrown.
func getResourceID(filters Filters) (*string, error) {
	var resourceID *string
	for _, filter := range filters.filters {
		if idFilter, ok := filter.(ResourceIDFilter); ok {
			resID := idFilter.GetResourceID()
			if function.IsNotEmptyNullCase(&resID) {
				if resourceID != nil && *resourceID != "" {
					return nil, inverr.Errorfc(codes.InvalidArgument, "Unsupported filter")
				}
				resourceID = &resID
			}
		}
	}
	return resourceID, nil
}

// getNonResourceIDFilters removes from the filters all the ResourceID filters.
func getNonResourceIDFilters(filters *Filters) *Filters {
	newFilters := new(Filters)

	// Copy over all errors verbatim, we don't know anything about them
	newFilters.errs = make([]error, len(filters.errs))
	copy(newFilters.errs, filters.errs)

	for _, filter := range filters.filters {
		if _, ok := filter.(ResourceIDFilter); !ok {
			newFilters.filters = append(newFilters.filters, filter)
		}
	}
	return newFilters
}
