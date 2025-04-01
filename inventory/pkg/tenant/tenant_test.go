// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tenant_test

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/tenant"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

const (
	testTenantID1     = "11111111-1111-1111-1111-111111111111"
	testTenantID2     = "22222222-2222-2222-2222-222222222222"
	testDefaultTenant = "00000000-0000-0000-0000-000000000000"
)

var expRoles = []string{
	"node-agent-readwrite-role", // TODO: deprecated, remove it
	"en-agent-rw",
}

func TestMain(m *testing.M) {
	// Only needed to suppress the error
	flag.String(
		"policyBundle",
		"/rego/policy_bundle.tar.gz",
		"Path of policy rego file",
	)
	flag.Parse()

	run := m.Run() // run all tests
	os.Exit(run)
}

func TestExtractTenantIDInterceptorEnforced(t *testing.T) {
	interceptorWithEnforcement := tenant.GetExtractTenantIDInterceptor(expRoles)

	t.Run("WithTenantID", func(t *testing.T) {
		ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background(), testTenantID1)
		testHandler := func(ctx context.Context, _ interface{}) (interface{}, error) {
			// Do assertion here on the actual context content
			assert.Equal(t, testTenantID1, ctx.Value(tenant.CtxTenantIDKey))
			return "", nil
		}

		_, err := interceptorWithEnforcement(ctx, nil, nil, testHandler)
		require.NoError(t, err)
	})

	t.Run("WithMultipleTenantIDEnforced", func(t *testing.T) {
		ctx := inv_testing.CreateIncomingContextWithENJWT(
			t, context.Background(), testTenantID1, testTenantID2)
		testHandler := func(_ context.Context, _ interface{}) (interface{}, error) {
			// Do assertion here on the actual context content
			t.Errorf("TenantID extraction should fail!")
			return "", nil
		}
		_, err := interceptorWithEnforcement(ctx, nil, nil, testHandler)
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("WithoutTenantIDEnforced", func(t *testing.T) {
		ctx := inv_testing.CreateIncomingContextWithENJWT(t, context.Background())
		testHandler := func(_ context.Context, _ interface{}) (interface{}, error) {
			// Do assertion here on the actual context content
			t.Errorf("Interceptor should block the request returning an error")
			return "", nil
		}
		testInfo := &grpc.UnaryServerInfo{
			FullMethod: "example",
		}
		_, err := interceptorWithEnforcement(ctx, nil, testInfo, testHandler)
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("ContextWithoutJWTEnforced", func(t *testing.T) {
		testHandler := func(_ context.Context, _ interface{}) (interface{}, error) {
			// Do assertion here on the actual context content
			t.Errorf("Interceptor should block the request returning an error")
			return "", nil
		}
		testInfo := &grpc.UnaryServerInfo{
			FullMethod: "example",
		}
		_, err := interceptorWithEnforcement(context.Background(), nil, testInfo, testHandler)
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})
}

func TestAddTenantIDToContext(t *testing.T) {
	ctx := tenant.AddTenantIDToContext(context.Background(), testTenantID1)
	assert.Equal(t, testTenantID1, ctx.Value(tenant.CtxTenantIDKey))
}

func TestGetTenantIDFromContext(t *testing.T) {
	t.Run("ValidTenantID", func(t *testing.T) {
		testCtx := context.WithValue(context.Background(), tenant.CtxTenantIDKey, testTenantID1)
		actualTID, valid := tenant.GetTenantIDFromContext(testCtx)
		require.True(t, valid)
		assert.Equal(t, testTenantID1, actualTID)
	})

	t.Run("EmptyTenantID", func(t *testing.T) {
		testCtx := context.WithValue(context.Background(), tenant.CtxTenantIDKey, "")
		actualTID, valid := tenant.GetTenantIDFromContext(testCtx)
		require.True(t, valid)
		assert.Empty(t, actualTID)
	})

	t.Run("WrongTypeTenantID", func(t *testing.T) {
		testCtx := context.WithValue(context.Background(), tenant.CtxTenantIDKey, 123)
		actualTID, valid := tenant.GetTenantIDFromContext(testCtx)
		require.False(t, valid)
		assert.Empty(t, actualTID)
	})
}

func TestGetRoles(t *testing.T) {
	aRoles := tenant.GetAgentsRole()
	assert.Len(t, aRoles, 1)

	oRoles := tenant.GetOnboardingRoles()
	assert.Len(t, oRoles, 2)
}
