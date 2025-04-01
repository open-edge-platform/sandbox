// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers

import (
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"

	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inventory "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	locationv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/location/v1"
	telemetryv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/telemetry/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
)

func castToTelemetryGroupResource(resp *inventory.GetResourceResponse) (
	*telemetryv1.TelemetryGroupResource, error,
) {
	if resp.GetResource().GetTelemetryGroup() != nil {
		return resp.GetResource().GetTelemetryGroup(), nil
	}

	err := errors.Errorfc(codes.Internal, "%s is not a TelemetryGroupResource", resp.GetResource())
	log.InfraErr(err).Msgf("could not cast inventory resource")
	return nil, err
}

func castToTelemetryProfile(resp *inventory.GetResourceResponse) (
	*telemetryv1.TelemetryProfile, error,
) {
	if resp.GetResource().GetTelemetryProfile() != nil {
		return resp.GetResource().GetTelemetryProfile(), nil
	}

	err := errors.Errorfc(codes.Internal, "%s is not a TelemetryLogsProfile", resp.GetResource())
	log.InfraErr(err).Msgf("could not cast inventory resource")
	return nil, err
}

func validateTelemetryProfileRelations(
	instanceID *string,
	siteID *string,
	regionID *string,
	isRelationRequired bool,
) error {
	setCount := 0
	if !isUnset(instanceID) {
		setCount++
	}
	if !isUnset(siteID) {
		setCount++
	}
	if !isUnset(regionID) {
		setCount++
	}
	if isRelationRequired && setCount == 0 {
		err := errors.Errorfc(codes.InvalidArgument,
			"No relation set. Target site, instance or region must be set.")
		return err
	}
	if setCount > 1 {
		err := errors.Errorfc(codes.InvalidArgument,
			"More than 1 relation set. Only one of target site, instance or region can be set at the same time")
		return err
	}

	return nil
}

func validateAndSetTelemetryProfileRelations(
	profile *telemetryv1.TelemetryProfile,
	instanceID *string,
	siteID *string,
	regionID *string,
) error {
	if profile == nil {
		return errors.Errorf("telemetry profile is nil")
	}

	err := validateTelemetryProfileRelations(instanceID, siteID, regionID, true)
	if err != nil {
		return err
	}

	if !isUnset(instanceID) {
		profile.Relation = &telemetryv1.TelemetryProfile_Instance{
			Instance: &computev1.InstanceResource{
				ResourceId: *instanceID,
			},
		}
	}

	if !isUnset(siteID) {
		profile.Relation = &telemetryv1.TelemetryProfile_Site{
			Site: &locationv1.SiteResource{
				ResourceId: *siteID,
			},
		}
	}

	if !isUnset(regionID) {
		profile.Relation = &telemetryv1.TelemetryProfile_Region{
			Region: &locationv1.RegionResource{
				ResourceId: *regionID,
			},
		}
	}

	return nil
}

func telemetryProfileFilter(
	kind telemetryv1.TelemetryResourceKind,
	instanceID *string,
	siteID *string,
	regionID *string,
) string {
	filter := fmt.Sprintf("%s = %s", telemetryv1.TelemetryProfileFieldKind,
		telemetryv1.TelemetryResourceKind_name[int32(kind)])

	filters := make([]string, 0)
	if instanceID != nil {
		filters = append(filters, fmt.Sprintf("(has(%s) AND %s.%s = %q)",
			telemetryv1.TelemetryProfileEdgeInstance,
			telemetryv1.TelemetryProfileEdgeInstance,
			computev1.InstanceResourceFieldResourceId,
			*instanceID))
	}

	if siteID != nil {
		filters = append(filters, fmt.Sprintf("(has(%s) AND %s.%s = %q)",
			telemetryv1.TelemetryProfileEdgeSite,
			telemetryv1.TelemetryProfileEdgeSite,
			locationv1.SiteResourceFieldResourceId,
			*siteID))
	}

	if regionID != nil {
		filters = append(filters, fmt.Sprintf("(has(%s) AND %s.%s = %q)",
			telemetryv1.TelemetryProfileEdgeRegion,
			telemetryv1.TelemetryProfileEdgeRegion,
			locationv1.RegionResourceFieldResourceId,
			*regionID))
	}

	if len(filters) > 0 {
		filter += " AND (" + strings.Join(filters, " OR ") + ")"
	}

	return filter
}
