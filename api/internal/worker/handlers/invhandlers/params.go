// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers

import "github.com/open-edge-platform/infra-core/api/internal/types"

type SiteURLParams struct {
	SiteID string
}

type SingleSchedURLParams struct {
	SingleSchedID string
}

type RepeatedSchedURLParams struct {
	RepeatedSchedID string
}

type RegionURLParams struct {
	RegionID string
}

type OUURLParams struct {
	OUID string
}

type OSResourceURLParams struct {
	OSResourceID string
}

type HostURLParams struct {
	HostID string
	Action types.Action
}

type WorkloadURLParams struct {
	WorkloadID string
}

type WorkloadMemberURLParams struct {
	WorkloadMemberID string
}

type InstanceURLParams struct {
	InstanceID string
	Action     types.Action
}

type TelemetryLogsGroupURLParams struct {
	TelemetryLogsGroupID string
}

type TelemetryMetricsGroupURLParams struct {
	TelemetryMetricsGroupID string
}

type TelemetryLogsProfileURLParams struct {
	TelemetryLogsProfileID string
}

type TelemetryMetricsProfileURLParams struct {
	TelemetryMetricsProfileID string
}

type ProviderURLParams struct {
	ProviderID string
}

type LocalAccountURLParams struct {
	LocalAccountID string
}
