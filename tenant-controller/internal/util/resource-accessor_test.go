// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	inventoryv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	providerv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/provider/v1"
	telemetryv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/telemetry/v1"
	inv_util "github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/tenant-controller/internal/util"
)

func TestSetResourceValue(t *testing.T) {
	data := []struct {
		resource *inventoryv1.Resource
	}{
		{
			resource: &inventoryv1.Resource{
				Resource: &inventoryv1.Resource_Provider{
					Provider: &providerv1.ProviderResource{},
				},
			},
		},
		{
			resource: &inventoryv1.Resource{
				Resource: &inventoryv1.Resource_TelemetryGroup{
					TelemetryGroup: &telemetryv1.TelemetryGroupResource{},
				},
			},
		},
	}
	for _, tc := range data {
		require.NoError(t, util.SetResourceValue(tc.resource, "tenant_id", "tid"))
		pm, uerr := inv_util.UnwrapResource[proto.Message](tc.resource)
		require.NoError(t, uerr)
		carrier, ok := pm.(interface{ GetTenantId() string })
		require.True(t, ok, "given resource does not expose `GetTenantId() string` function")
		require.Equal(t, "tid", carrier.GetTenantId())
	}
}

func TestSetResourceValue_ResourceIsNil(t *testing.T) {
	err := util.SetResourceValue(nil, "anyName", "anyValue")
	require.Error(t, err)
}

func TestSetResourceValue_ResourceHasNoResource(t *testing.T) {
	err := util.SetResourceValue(&inventoryv1.Resource{}, "anyName", "anyValue")
	require.Error(t, err)
}

func TestSetResourceValue_InvalidField(t *testing.T) {
	err := util.SetResourceValue(&inventoryv1.Resource{Resource: &inventoryv1.Resource_Host{}}, "anyInvalidField", "anyValue")
	require.Error(t, err)
}

func TestSetResourceValue_InvalidValue(t *testing.T) {
	err := util.SetResourceValue(&inventoryv1.Resource{Resource: &inventoryv1.Resource_Host{}}, "name", 1)
	require.Error(t, err)
}
