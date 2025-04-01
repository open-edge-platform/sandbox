// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invclient_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	grpc_status "google.golang.org/grpc/status"

	tenantv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/tenant/v1"
	tc_testing "github.com/open-edge-platform/infra-core/tenant-controller/internal/testing"
)

func TestCreateTenantResource(t *testing.T) {
	ic := tc_testing.CreateInvClient(t)

	anyTenant := uuid.NewString()
	anotherTenant := uuid.NewString()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	t.Run("Create Tenant", func(t *testing.T) {
		r, e := ic.CreateTenantResource(ctx, anyTenant)
		require.NoError(t, e)
		require.NotNil(t, r)
		require.Equal(t, anyTenant, r.GetTenant().GetTenantId())
	})

	t.Run("Create Another Tenant With Same TenantID", func(t *testing.T) {
		r, e := ic.CreateTenantResource(ctx, anyTenant)
		require.Error(t, e)
		require.Nil(t, r)
	})

	t.Run("Create Another Tenant", func(t *testing.T) {
		r, e := ic.CreateTenantResource(ctx, anotherTenant)
		require.NoError(t, e)
		require.Equal(t, anotherTenant, r.GetTenant().GetTenantId())
	})

	t.Run("Soft Delete Tenant", func(t *testing.T) {
		require.NoError(t, ic.DeleteTenantResource(ctx, anyTenant))

		tenant, err := ic.GetTenantResourceInstance(ctx, anyTenant)
		require.NoError(t, err)
		require.Equal(t, tenantv1.TenantState_TENANT_STATE_DELETED, tenant.GetDesiredState())
	})

	t.Run("Hard Delete Tenant", func(t *testing.T) {
		tenant, err := ic.GetTenantResourceInstance(ctx, anotherTenant)
		require.NoError(t, err)

		require.NoError(t, ic.DeleteTenantResource(ctx, anotherTenant))
		require.NoError(t, ic.HardDeleteTenantResource(ctx, anotherTenant, tenant.GetResourceId()))

		tenant, err = ic.GetTenantResourceInstance(ctx, anotherTenant)
		require.Error(t, err)
		require.Nil(t, tenant)
	})
}

func TestGetTenantResource(t *testing.T) {
	ic := tc_testing.CreateInvClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	t.Run("Get Existing Tenant", func(t *testing.T) {
		anyTenantID := uuid.NewString()
		r, e := ic.CreateTenantResource(ctx, anyTenantID)
		require.NoError(t, e)
		require.NotNil(t, r)
		require.Equal(t, anyTenantID, r.GetTenant().GetTenantId())

		tid, rid, err := ic.GetTenantResource(ctx, anyTenantID)
		require.NoError(t, err)
		require.NotEmpty(t, tid)
		require.NotEmpty(t, rid)
		require.Equal(t, anyTenantID, tid)
	})

	t.Run("Get Not Existing Tenant", func(t *testing.T) {
		anyTenantID := uuid.NewString()
		tid, rid, err := ic.GetTenantResource(ctx, anyTenantID)
		require.Error(t, err)
		require.Equal(t, codes.NotFound, grpc_status.Convert(err).Code())
		require.Empty(t, tid)
		require.Empty(t, rid)
	})
}
