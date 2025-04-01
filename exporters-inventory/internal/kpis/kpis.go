// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package kpis

import (
	"github.com/prometheus/client_golang/prometheus"
)

// KPI interface defines the methods that format the behavior
// of a kpi. It includes that a kpi must provide those methods
// in order to support its content to be exported to a particular
// TSDB.
type KPI interface {
	PrometheusFormat() ([]prometheus.Metric, error)
}

// Const definitions of kpis name and description.
// Name and description are used to define a particular KPI.
const (
	inventoryHostsMaintenanceName        = "host_schedule"
	inventoryHostsMaintenanceDescription = "The hosts maintenance"

	inventoryHostsStatusName        = "host_status"
	inventoryHostsStatusDescription = "The host status"

	inventoryHostsProvisioningStatusName        = "host_provisioning_status"
	inventoryHostsProvisioningStatusDescription = "The host provisioning status"

	inventoryHostsOnboardingStatusName        = "host_onboarding_status"
	inventoryHostsOnboardingStatusDescription = "The host onboarding status"

	inventoryHostsUpdateStatusName        = "host_update_status"
	inventoryHostsUpdateStatusDescription = "The host update status"

	inventoryHostsTotalProvisioningTimeName        = "host_total_provisioning_time"
	inventoryHostsTotalProvisioningTimeDescription = "The host total provisioning time"
)

// NewInventoryHostsStatus defines the factory implementation of a kpi
// InventoryHostsStatus having a well defined name and description.
func NewInventoryHostsStatus() *InventoryHostsStatus {
	return &InventoryHostsStatus{
		name:        inventoryHostsStatusName,
		description: inventoryHostsStatusDescription,
	}
}

// NewInventoryHostsSchedule defines the factory implementation of a kpi
// InventoryHostsMaintenance having a well defined name and description.
func NewInventoryHostsSchedule() *InventoryHostsSchedule {
	return &InventoryHostsSchedule{
		name:        inventoryHostsMaintenanceName,
		description: inventoryHostsMaintenanceDescription,
	}
}

// NewInventoryHostsOnboarding defines the factory implementation of a kpi
// InventoryHostsOnboarding having a well defined name and description.
func NewInventoryHostsOnboarding() *InventoryHostsOnboardingStatus {
	return &InventoryHostsOnboardingStatus{
		name:        inventoryHostsOnboardingStatusName,
		description: inventoryHostsOnboardingStatusDescription,
	}
}

// NewInventoryHostsProvisioning defines the factory implementation of a kpi
// InventoryHostsProvisioning having a well defined name and description.
func NewInventoryHostsProvisioning() *InventoryHostsProvisioningStatus {
	return &InventoryHostsProvisioningStatus{
		name:        inventoryHostsProvisioningStatusName,
		description: inventoryHostsProvisioningStatusDescription,
	}
}

// NewInventoryHostsUpdate defines the factory implementation of a kpi
// InventoryHostsUpdate having a well defined name and description.
func NewInventoryHostsUpdate() *InventoryHostsUpdateStatus {
	return &InventoryHostsUpdateStatus{
		name:        inventoryHostsUpdateStatusName,
		description: inventoryHostsUpdateStatusDescription,
	}
}

// NewInventoryHostsTotalProvisioningTime defines the factory implementation of a kpi
// InventoryHostsTotalProvisioningTime having a well defined name and description.
func NewInventoryHostsTotalProvisioningTime() *InventoryHostsTotalProvisoningTime {
	return &InventoryHostsTotalProvisoningTime{
		name:        inventoryHostsTotalProvisioningTimeName,
		description: inventoryHostsTotalProvisioningTimeDescription,
	}
}
