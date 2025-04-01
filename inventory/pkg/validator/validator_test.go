// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package validator_test

import (
	"flag"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

var testHostResource = &computev1.HostResource{
	Name:         "for unit testing purposes",
	DesiredState: computev1.HostState_HOST_STATE_ONBOARDED,

	Site:     nil,
	Provider: nil,
	Note:     "some note",

	HardwareKind: "XDgen2",
	SerialNumber: "12345678",
	Uuid:         uuid.NewString(),
	MemoryBytes:  64 * util.Gigabyte,

	CpuModel:        "12th Gen Intel(R) Core(TM) i9-12900",
	CpuSockets:      1,
	CpuCores:        14,
	CpuCapabilities: "",
	CpuArchitecture: "x86_64",
	CpuThreads:      10,

	MgmtIp: "192.168.10.10",

	BmcKind:     computev1.BaremetalControllerKind_BAREMETAL_CONTROLLER_KIND_PDU,
	BmcIp:       "10.0.0.10",
	BmcUsername: "user",
	BmcPassword: "pass",
	PxeMac:      "90:49:fa:ff:ff:ff",

	Hostname: "testhost1",

	DesiredPowerState: computev1.PowerState_POWER_STATE_ON,
}

// Define this flag in order to call all tests with the same parameters.
var _ = flag.String(
	"policyBundle",
	"/rego/policy_bundle.tar.gz",
	"Path of policy rego file",
)

func TestMustInit(t *testing.T) {
	// empty messages
	validator.MustInit([]proto.Message{})

	validator.MustInit([]proto.Message{
		&computev1.HostResource{},
	})

	err := validator.ValidateMessage(testHostResource)
	require.NoError(t, err)
}

func TestValidateMessage(t *testing.T) {
	validator.MustInit([]proto.Message{
		&computev1.HostResource{},
	})

	// success cass
	err := validator.ValidateMessage(testHostResource)
	require.NoError(t, err)

	// error case
	testHostResource.Uuid = "wrong format"
	err = validator.ValidateMessage(testHostResource)
	require.Error(t, err)
}
