// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server_test

import (
	"reflect"
	"testing"

	"google.golang.org/protobuf/proto"

	commonv1 "github.com/open-edge-platform/infra-core/apiv2/v2/internal/pbapi/resources/common/v1"
	computev1 "github.com/open-edge-platform/infra-core/apiv2/v2/internal/pbapi/resources/compute/v1"
	locationv1 "github.com/open-edge-platform/infra-core/apiv2/v2/internal/pbapi/resources/location/v1"
	statusv1 "github.com/open-edge-platform/infra-core/apiv2/v2/internal/pbapi/resources/status/v1"
	inv_computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_locationv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/location/v1"
	inv_statusv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/status/v1"
)

// Write an example of inventory resource with a Host resource filled with all fields.
var exampleInvHostResource = &inv_computev1.HostResource{
	ResourceId:   "host-12345678",
	Name:         "example-host",
	DesiredState: inv_computev1.HostState_HOST_STATE_REGISTERED,
	CurrentState: inv_computev1.HostState_HOST_STATE_ONBOARDED,
	Site: &inv_locationv1.SiteResource{
		ResourceId: "site-12345678",
	},
	Note:            "Example note",
	SerialNumber:    "SN12345678",
	Uuid:            "uuid-1234-5678-9012-3456",
	MemoryBytes:     16384,
	CpuModel:        "Intel Xeon",
	CpuSockets:      2,
	CpuCores:        16,
	CpuCapabilities: "capability1,capability2",
	CpuArchitecture: "x86_64",
	CpuThreads:      32,
	CpuTopology:     "topology-json",
	BmcKind:         inv_computev1.BaremetalControllerKind_BAREMETAL_CONTROLLER_KIND_IPMI,
	BmcIp:           "192.168.0.1",
	Hostname:        "example-hostname",
	ProductName:     "Example Product",
	BiosVersion:     "1.0.0",
	BiosReleaseDate: "2023-01-01",
	BiosVendor:      "Example Vendor",
	HostStorages: []*inv_computev1.HoststorageResource{
		{
			ResourceId:    "storage-12345678",
			Wwid:          "wwid-1234",
			Serial:        "serial-1234",
			Vendor:        "vendor-1234",
			Model:         "model-1234",
			CapacityBytes: 1024,
			DeviceName:    "sda",
		},
	},
	HostNics: []*inv_computev1.HostnicResource{
		{
			ResourceId:    "nic-12345678",
			DeviceName:    "eth0",
			PciIdentifier: "pci-1234",
			MacAddr:       "00:11:22:33:44:55",
			SriovEnabled:  true,
			SriovVfsNum:   8,
			SriovVfsTotal: 16,
			Features:      "feature1,feature2",
			Mtu:           1500,
			LinkState:     inv_computev1.NetworkInterfaceLinkState_NETWORK_INTERFACE_LINK_STATE_UP,
			BmcInterface:  true,
		},
	},
	HostUsbs: []*inv_computev1.HostusbResource{
		{
			ResourceId: "usb-12345678",
			Idvendor:   "vendor-1234",
			Idproduct:  "product-1234",
			Bus:        123,
			Addr:       123,
			Class:      "class-1234",
			Serial:     "serial-1234",
			DeviceName: "usb0",
		},
	},
	HostGpus: []*inv_computev1.HostgpuResource{
		{
			ResourceId:  "gpu-12345678",
			PciId:       "pci-1234",
			Product:     "product-1234",
			Vendor:      "vendor-1234",
			Description: "description-1234",
			DeviceName:  "gpu0",
			Features:    "feature1,feature2",
		},
	},
	Instance: &inv_computev1.InstanceResource{
		ResourceId:                  "instance-12345678",
		ProvisioningStatus:          "provisioned",
		ProvisioningStatusIndicator: inv_statusv1.StatusIndication_STATUS_INDICATION_IDLE,
		ProvisioningStatusTimestamp: 1234567890,
		UpdateStatus:                "updating",
		UpdateStatusIndicator:       inv_statusv1.StatusIndication_STATUS_INDICATION_IDLE,
		UpdateStatusTimestamp:       1234567890,
		InstanceStatus:              "running",
		InstanceStatusIndicator:     inv_statusv1.StatusIndication_STATUS_INDICATION_IDLE,
		InstanceStatusTimestamp:     1234567890,
	},
	Metadata:                    `[{"key":"key1","value":"value1"}]`,
	OnboardingStatus:            "onboarding",
	OnboardingStatusIndicator:   inv_statusv1.StatusIndication_STATUS_INDICATION_IDLE,
	OnboardingStatusTimestamp:   1234567890,
	RegistrationStatus:          "registered",
	RegistrationStatusIndicator: inv_statusv1.StatusIndication_STATUS_INDICATION_IDLE,
	RegistrationStatusTimestamp: 1234567890,
	HostStatus:                  "running",
	HostStatusIndicator:         inv_statusv1.StatusIndication_STATUS_INDICATION_IDLE,
	HostStatusTimestamp:         1234567890,
}

// Write an example of API resource with a Host resource filled with all fields.
var exampleAPIHostResource = &computev1.HostResource{
	ResourceId:   "host-12345678",
	Name:         "example-host",
	DesiredState: computev1.HostState_HOST_STATE_REGISTERED,
	CurrentState: computev1.HostState_HOST_STATE_ONBOARDED,
	Site: &locationv1.SiteResource{
		ResourceId: "site-12345678",
	},
	Note:            "Example note",
	SerialNumber:    "SN12345678",
	Uuid:            "uuid-1234-5678-9012-3456",
	MemoryBytes:     "16384",
	CpuModel:        "Intel Xeon",
	CpuSockets:      2,
	CpuCores:        16,
	CpuCapabilities: "capability1,capability2",
	CpuArchitecture: "x86_64",
	CpuThreads:      32,
	CpuTopology:     "topology-json",
	BmcKind:         computev1.BaremetalControllerKind_BAREMETAL_CONTROLLER_KIND_IPMI,
	BmcIp:           "192.168.0.1",
	Hostname:        "example-hostname",
	ProductName:     "Example Product",
	BiosVersion:     "1.0.0",
	BiosReleaseDate: "2023-01-01",
	BiosVendor:      "Example Vendor",
	HostStorages: []*computev1.HoststorageResource{
		{
			ResourceId:    "storage-12345678",
			Wwid:          "wwid-1234",
			Serial:        "serial-1234",
			Vendor:        "vendor-1234",
			Model:         "model-1234",
			CapacityBytes: "1024",
			DeviceName:    "sda",
		},
	},
	HostNics: []*computev1.HostnicResource{
		{
			ResourceId:    "nic-12345678",
			DeviceName:    "eth0",
			PciIdentifier: "pci-1234",
			MacAddr:       "00:11:22:33:44:55",
			SriovEnabled:  true,
			SriovVfsNum:   8,
			SriovVfsTotal: 16,
			Features:      "feature1,feature2",
			Mtu:           1500,
			LinkState:     computev1.NetworkInterfaceLinkState_NETWORK_INTERFACE_LINK_STATE_UP,
			BmcInterface:  true,
		},
	},
	HostUsbs: []*computev1.HostusbResource{
		{
			ResourceId: "usb-12345678",
			Idvendor:   "vendor-1234",
			Idproduct:  "product-1234",
			Bus:        123,
			Addr:       123,
			Class:      "class-1234",
			Serial:     "serial-1234",
			DeviceName: "usb0",
		},
	},
	HostGpus: []*computev1.HostgpuResource{
		{
			ResourceId:  "gpu-12345678",
			PciId:       "pci-1234",
			Product:     "product-1234",
			Vendor:      "vendor-1234",
			Description: "description-1234",
			DeviceName:  "gpu0",
			Features:    "feature1,feature2",
		},
	},
	Instance: &computev1.InstanceResource{
		ResourceId:                  "instance-12345678",
		InstanceStatus:              "running",
		InstanceStatusIndicator:     statusv1.StatusIndication_STATUS_INDICATION_IDLE,
		InstanceStatusTimestamp:     "1234567890",
		ProvisioningStatus:          "provisioned",
		ProvisioningStatusIndicator: statusv1.StatusIndication_STATUS_INDICATION_IDLE,
		ProvisioningStatusTimestamp: "1234567890",
		UpdateStatus:                "updating",
		UpdateStatusIndicator:       statusv1.StatusIndication_STATUS_INDICATION_IDLE,
		UpdateStatusTimestamp:       "1234567890",
	},
	HostId:                      "host-12345678",
	SiteId:                      "site-12345678",
	Metadata:                    []*commonv1.MetadataItem{{Key: "key1", Value: "value1"}},
	OnboardingStatus:            "onboarding",
	OnboardingStatusIndicator:   statusv1.StatusIndication_STATUS_INDICATION_IDLE,
	OnboardingStatusTimestamp:   "1234567890",
	RegistrationStatus:          "registered",
	RegistrationStatusIndicator: statusv1.StatusIndication_STATUS_INDICATION_IDLE,
	RegistrationStatusTimestamp: "1234567890",
	HostStatus:                  "running",
	HostStatusIndicator:         statusv1.StatusIndication_STATUS_INDICATION_IDLE,
	HostStatusTimestamp:         "1234567890",
}

// compareProtoMessages compares two proto.Message parameters and checks if all the fields set in the first
// have the same value as in the second one.
//
//nolint:gocritic,errcheck // This function is used only for testing purposes.
func compareProtoMessages(t *testing.T, msg1, msg2 proto.Message) {
	t.Helper()
	v1 := reflect.ValueOf(msg1).Elem()
	v2 := reflect.ValueOf(msg2).Elem()

	for i := 0; i < v1.NumField(); i++ {
		field1 := v1.Field(i)
		field2 := v2.Field(i)

		if field1.IsZero() {
			continue
		}

		if field1.Kind() == reflect.Ptr &&
			field1.Type().Implements(reflect.TypeOf((*proto.Message)(nil)).Elem()) {
			// Compare messages recursively.
			compareProtoMessages(t, field1.Interface().(proto.Message), field2.Interface().(proto.Message))
		} else if field1.Kind() == reflect.Slice &&
			field1.Type().Elem().Implements(reflect.TypeOf((*proto.Message)(nil)).Elem()) {
			// Compare slices of messages recursively.
			for j := 0; j < field1.Len(); j++ {
				compareProtoMessages(t, field1.Index(j).Interface().(proto.Message),
					field2.Index(j).Interface().(proto.Message))
			}
		} else if !reflect.DeepEqual(field1.Interface(), field2.Interface()) {
			// Compare fields in the message.
			t.Errorf("Field %s: got %v, want %v", v1.Type().Field(i).Name, field2.Interface(), field1.Interface())
		}
	}
}
