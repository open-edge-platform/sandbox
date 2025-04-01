// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	inv_handlers "github.com/open-edge-platform/infra-core/api/internal/worker/handlers/invhandlers"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	providerv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/provider/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/tenant"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

var (
	pgSize   = 10
	pgOffset = 0

	pgSizeWrong  = -10
	pgIndexWrong = -1

	emptyString = ""
	nullString  = "null"

	defaultRegionID      = "region-12345678"
	defaultOUID          = "ou-12345678"
	defaultOperationNote = "Test Operation Note"
	// For schedule tests.
	timeNow            = time.Now()
	timeNow30MinString = fmt.Sprint(timeNow.UTC().Unix() + 1800) // Now + 30 min.

	// see resources.go to understand the translation.
	Mtu           = "1500"
	Bmc           = false
	PciIdentifier = ""
	SriovEnabled  = false
	SriovVfsNum   = "0"
	SriovVfsTotal = "0"
	Condition     = computev1.HostComponentState_HOST_COMPONENT_STATE_EXISTS.String()
	MacAddr1      = "00:11:22:33:44:55"
	Name1         = "eth0"
	LinkState1    = api.LINKSTATEUP
	ConfigMode1   = api.IPADDRESSCONFIGMODEDYNAMIC
	IPAddress     = strfmt.CIDR("10.0.0.1/24")
	Status        = api.IPADDRESSSTATUSCONFIGURED
	StatusDetail  = "Specifically I am fine"
	IPAddresses1  = []api.IPAddress{
		{
			Address:      &IPAddress,
			ConfigMethod: &ConfigMode1,
			Status:       &Status,
			StatusDetail: &StatusDetail,
		},
	}
	MacAddr2     = "00:11:22:33:44:66"
	Name2        = "eth1"
	LinkState2   = api.LINKSTATEDOWN
	ConfigMode2  = api.IPADDRESSCONFIGMODEDYNAMIC
	IPAddresses2 = []api.IPAddress{}
)

const (
	BadOperation types.Operation = "BadOp"
)

var ctxTest = tenant.AddTenantIDToContext(context.TODO(), client.FakeTenantID)

// Starts all Inventory testing environment to test API inv handlers.
func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// Needed for filepath of current dir related to root where /out dir is placed
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(wd))))

	policyPath := projectRoot + "/out"
	migrationsDir := projectRoot + "/out"

	inv_testing.StartTestingEnvironment(policyPath, "", migrationsDir)

	run := m.Run() // run all tests
	inv_testing.StopTestingEnvironment()
	os.Exit(run)
}

func VerifyProvider(t *testing.T, got *api.Provider, expected *providerv1.ProviderResource) {
	t.Helper()

	require.NotNil(t, got)
	assert.Equal(t, got.Name, expected.Name)
	assert.Equal(t, *got.ApiCredentials, expected.ApiCredentials)
	assert.Equal(t, got.ApiEndpoint, expected.ApiEndpoint)
	assert.Equal(t, *got.Config, expected.Config)
	assert.Equal(t, got.ProviderKind, inv_handlers.GrpcProviderKindToOpenAPIProviderKind(expected.ProviderKind))
	assert.Equal(t, *got.ProviderVendor, inv_handlers.GrpcProviderVendorToOpenAPIProviderVendor(expected.ProviderVendor))
}

func TestGrpcToOpenAPITimestamps(t *testing.T) {
	timeZero := time.Unix(0, 0)
	t.Run("ValidTimestamp", func(t *testing.T) {
		createdAt := time.Now().UTC().Format(inv_handlers.ISO8601TimeFormat)
		createdAtParsed, err := time.Parse(inv_handlers.ISO8601TimeFormat, createdAt)
		require.NoError(t, err)
		updatedAt := time.Now().Add(time.Minute).UTC().Format(inv_handlers.ISO8601TimeFormat)
		updatedAtParsed, err := time.Parse(inv_handlers.ISO8601TimeFormat, updatedAt)
		require.NoError(t, err)

		host := &computev1.HostResource{
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}

		tsOpenAPI := inv_handlers.GrpcToOpenAPITimestamps(host)
		assert.Equal(t, createdAtParsed, *tsOpenAPI.CreatedAt)
		assert.Equal(t, updatedAtParsed, *tsOpenAPI.UpdatedAt)
	})
	t.Run("Nil", func(t *testing.T) {
		// Test for nil
		assert.Nil(t, inv_handlers.GrpcToOpenAPITimestamps(nil))
	})
	t.Run("NilCreatedAt", func(t *testing.T) {
		updatedAt := time.Now().Add(time.Minute).UTC().Format(inv_handlers.ISO8601TimeFormat)
		updatedAtParsed, err := time.Parse(inv_handlers.ISO8601TimeFormat, updatedAt)
		require.NoError(t, err)
		host := &computev1.HostResource{
			UpdatedAt: updatedAt,
		}

		tsOpenAPI := inv_handlers.GrpcToOpenAPITimestamps(host)
		assert.Equal(t, timeZero, *tsOpenAPI.CreatedAt)
		assert.Equal(t, updatedAtParsed, *tsOpenAPI.UpdatedAt)
	})
	t.Run("NilUpdatedAt", func(t *testing.T) {
		createdAt := time.Now().UTC().Format(inv_handlers.ISO8601TimeFormat)
		createdAtParsed, err := time.Parse(inv_handlers.ISO8601TimeFormat, createdAt)
		require.NoError(t, err)
		host := &computev1.HostResource{
			CreatedAt: createdAt,
		}

		tsOpenAPI := inv_handlers.GrpcToOpenAPITimestamps(host)
		assert.Equal(t, createdAtParsed, *tsOpenAPI.CreatedAt)
		assert.Equal(t, timeZero, *tsOpenAPI.UpdatedAt)
	})
}
