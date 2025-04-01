// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"

	telemetryv1 "github.com/open-edge-platform/infra-core/apiv2/v2/internal/pbapi/resources/telemetry/v1"
	inv_computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_locationv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/location/v1"
	inv_telemetryv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/telemetry/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
)

func validateTelemetryProfileRelations(
	instanceID string,
	siteID string,
	regionID string,
	isRelationRequired bool,
) error {
	setCount := 0
	if instanceID != "" {
		setCount++
	}
	if siteID != "" {
		setCount++
	}
	if regionID != "" {
		setCount++
	}
	if isRelationRequired && setCount == 0 {
		return errors.Errorfc(codes.InvalidArgument,
			"No relation set. Target site, instance or region must be set.")
	}
	if setCount > 1 {
		return errors.Errorfc(codes.InvalidArgument,
			"More than 1 relation set. Only one of target site, instance or region can be set at the same time")
	}

	return nil
}

func validateAndSetTelemetryProfileRelations(
	profile *inv_telemetryv1.TelemetryProfile,
	instanceID string,
	siteID string,
	regionID string,
) error {
	if profile == nil {
		return errors.Errorf("telemetry profile is nil")
	}

	err := validateTelemetryProfileRelations(instanceID, siteID, regionID, true)
	if err != nil {
		return err
	}

	if instanceID != "" {
		profile.Relation = &inv_telemetryv1.TelemetryProfile_Instance{
			Instance: &inv_computev1.InstanceResource{
				ResourceId: instanceID,
			},
		}
	}

	if siteID != "" {
		profile.Relation = &inv_telemetryv1.TelemetryProfile_Site{
			Site: &inv_locationv1.SiteResource{
				ResourceId: siteID,
			},
		}
	}

	if regionID != "" {
		profile.Relation = &inv_telemetryv1.TelemetryProfile_Region{
			Region: &inv_locationv1.RegionResource{
				ResourceId: regionID,
			},
		}
	}

	return nil
}

func telemetryProfileFilter(
	kind telemetryv1.TelemetryResourceKind,
	instanceID string,
	siteID string,
	regionID string,
) string {
	filter := fmt.Sprintf("%s = %s", inv_telemetryv1.TelemetryProfileFieldKind,
		inv_telemetryv1.TelemetryResourceKind_name[int32(kind)])

	filters := make([]string, 0)
	if instanceID != "" {
		filters = append(filters, fmt.Sprintf("(has(%s) AND %s.%s = %q)",
			inv_telemetryv1.TelemetryProfileEdgeInstance,
			inv_telemetryv1.TelemetryProfileEdgeInstance,
			inv_computev1.InstanceResourceFieldResourceId,
			instanceID))
	}

	if siteID != "" {
		filters = append(filters, fmt.Sprintf("(has(%s) AND %s.%s = %q)",
			inv_telemetryv1.TelemetryProfileEdgeSite,
			inv_telemetryv1.TelemetryProfileEdgeSite,
			inv_locationv1.SiteResourceFieldResourceId,
			siteID))
	}

	if regionID != "" {
		filters = append(filters, fmt.Sprintf("(has(%s) AND %s.%s = %q)",
			inv_telemetryv1.TelemetryProfileEdgeRegion,
			inv_telemetryv1.TelemetryProfileEdgeRegion,
			inv_locationv1.RegionResourceFieldResourceId,
			regionID))
	}

	if len(filters) > 0 {
		filter += " AND (" + strings.Join(filters, " OR ") + ")"
	}

	return filter
}
