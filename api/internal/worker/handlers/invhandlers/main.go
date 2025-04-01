// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers

import (
	"google.golang.org/grpc/codes"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	schedule_cache "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client/cache/schedule"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var log = logging.GetLogger("handlers")

// SupportedResourceTypes defines the implemented resource
// handlers available in inventory handlers.
var SupportedResourceTypes = map[types.Resource]struct{}{
	types.Host:                    {},
	types.Locations:               {},
	types.Region:                  {},
	types.Site:                    {},
	types.OU:                      {},
	types.SingleSched:             {},
	types.RepeatedSched:           {},
	types.OSResource:              {},
	types.Workload:                {},
	types.WorkloadMember:          {},
	types.Instance:                {},
	types.TelemetryLogsGroup:      {},
	types.TelemetryMetricsGroup:   {},
	types.TelemetryLogsProfile:    {},
	types.TelemetryMetricsProfile: {},
	types.Provider:                {},
	types.LocalAccount:            {},
}

// InventoryResource defines the set of methods
// that every inventory resource handler must
// implement.
type InventoryResource interface {
	Create(*types.Job) (*types.Payload, error)
	Get(*types.Job) (*types.Payload, error)
	Update(*types.Job) (*types.Payload, error)
	Delete(*types.Job) error
	List(*types.Job) (*types.Payload, error)
}

// NewInventoryResourceHandler instantiates a new inventory resource handler given the resource type.
//
//nolint:cyclop // high cyclomatic complexity because of the switch-case.
func NewInventoryResourceHandler(
	resType types.Resource,
	invClientHandler *clients.InventoryClientHandler,
	invHCacheClientHandler *schedule_cache.HScheduleCacheClient,
) (InventoryResource, error) {
	if _, ok := SupportedResourceTypes[resType]; !ok {
		log.InfraError("invalid inventory resource type %s", resType).
			Msg("could not instantiate inventory resource handler")
		return nil, errors.Errorfc(codes.Unimplemented, "invalid inventory resource type %s", resType)
	}

	switch resType {
	case types.Host:
		return NewHostHandler(invClientHandler), nil
	case types.Locations:
		return NewLocationsHandler(invClientHandler), nil
	case types.Region:
		return NewRegionHandler(invClientHandler), nil
	case types.Site:
		return NewSiteHandler(invClientHandler), nil
	case types.OU:
		return NewOUHandler(invClientHandler), nil
	case types.SingleSched:
		return NewSingleSchedHandler(invClientHandler, invHCacheClientHandler), nil
	case types.RepeatedSched:
		return NewRepeatedSchedHandler(invClientHandler, invHCacheClientHandler), nil
	case types.OSResource:
		return NewOSHandler(invClientHandler), nil
	case types.Workload:
		return NewWorkloadHandler(invClientHandler), nil
	case types.WorkloadMember:
		return NewWorkloadMemberHandler(invClientHandler), nil
	case types.Instance:
		return NewInstanceHandler(invClientHandler), nil
	case types.TelemetryLogsGroup:
		return NewTelemetryLogsGroupHandler(invClientHandler), nil
	case types.TelemetryMetricsGroup:
		return NewTelemetryMetricsGroupHandler(invClientHandler), nil
	case types.TelemetryLogsProfile:
		return NewTelemetryLogsProfileHandler(invClientHandler), nil
	case types.TelemetryMetricsProfile:
		return NewTelemetryMetricsProfileHandler(invClientHandler), nil
	case types.Provider:
		return NewProvider(invClientHandler), nil
	case types.LocalAccount:
		return NewLocalAccountHandler(invClientHandler), nil
	default:
		log.InfraError("invalid inventory resource type %s", resType).
			Msg("could not instantiate inventory resource handler")
		return nil, errors.Errorfc(codes.Unimplemented, "invalid inventory resource type %s", resType)
	}
}
