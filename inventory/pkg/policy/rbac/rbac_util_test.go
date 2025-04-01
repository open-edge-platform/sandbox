// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package rbac_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/policy/rbac"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

const (
	authKey    = "authorization"
	clientKey  = "Client"
	clientName = "CDNBoots"
)

func Test_RequestIsAuthorized(t *testing.T) {
	p, err := loadPolicyBundle(regoPath)
	require.NoError(t, err)

	t.Run("JWT with RO and RW roles", func(t *testing.T) {
		// creating a JWT with read and write roles
		_, jwtToken, err := inv_testing.CreateJWT(t, uuid.NewString())
		require.NoError(t, err)

		niceMD := metautils.NiceMD{}
		niceMD.Add(authKey, "Bearer "+jwtToken)
		ctx := niceMD.ToIncoming(context.Background())

		res := p.IsRequestAuthorized(ctx, rbac.PostKey)
		assert.True(t, res)

		res = p.IsRequestAuthorized(ctx, rbac.ListKey)
		assert.True(t, res)

		res = p.IsRequestAuthorized(ctx, "UnknownOp")
		assert.False(t, res)
	})

	t.Run("JWT with RO role", func(t *testing.T) {
		// creating a JWT with read only role
		_, jwtToken, err := inv_testing.CreateJWTWithReadRole(t, uuid.NewString())
		require.NoError(t, err)

		niceMD1 := metautils.NiceMD{}
		niceMD1.Add(authKey, "bearer "+jwtToken)
		ctx1 := niceMD1.ToIncoming(context.Background())

		res1 := p.IsRequestAuthorized(ctx1, rbac.PostKey)
		assert.False(t, res1)

		res1 = p.IsRequestAuthorized(ctx1, rbac.ListKey)
		assert.True(t, res1)

		res1 = p.IsRequestAuthorized(ctx1, "UnknownOp")
		assert.False(t, res1)
	})

	t.Run("JWT with RW role", func(t *testing.T) {
		// creating a JWT with read-write only role
		_, jwtToken, err := inv_testing.CreateJWTWithReadWriteRole(t, uuid.NewString())
		require.NoError(t, err)

		niceMD2 := metautils.NiceMD{}
		niceMD2.Add(authKey, "bearer "+jwtToken)
		ctx2 := niceMD2.ToIncoming(context.Background())

		res2 := p.IsRequestAuthorized(ctx2, rbac.PostKey)
		assert.True(t, res2)

		res2 = p.IsRequestAuthorized(ctx2, rbac.ListKey)
		assert.True(t, res2)

		res2 = p.IsRequestAuthorized(ctx2, "UnknownOp")
		assert.False(t, res2)
	})

	t.Run("EN's JWT with RO and RW roles", func(t *testing.T) {
		// creating EN's JWT with read and write roles
		_, enJwtToken, err := inv_testing.CreateENJWT(t, uuid.NewString())
		require.NoError(t, err)

		niceMD3 := metautils.NiceMD{}
		niceMD3.Add(authKey, "Bearer "+enJwtToken)
		ctx3 := niceMD3.ToIncoming(context.Background())

		res3 := p.IsRequestAuthorized(ctx3, rbac.PostKey)
		assert.True(t, res3)

		res3 = p.IsRequestAuthorized(ctx3, rbac.ListKey)
		assert.True(t, res3)

		res3 = p.IsRequestAuthorized(ctx3, "UnknownOp")
		assert.False(t, res3)
	})
}

func Test_ExtractClaimsFromContext(t *testing.T) {
	_, validJWT, err := inv_testing.CreateJWT(t, uuid.NewString())
	require.NoError(t, err)
	tests := []struct {
		name        string
		jwtToken    string
		expectErr   bool
		validateJwt bool
	}{
		{
			name:        "Valid JWT with validate",
			jwtToken:    validJWT,
			expectErr:   false,
			validateJwt: true,
		},
		{
			name:        "Valid JWT unverified",
			jwtToken:    validJWT,
			expectErr:   false,
			validateJwt: false,
		},
		{
			name:        "Invalid JWT with validate",
			jwtToken:    "invalid.jwt.token",
			expectErr:   true,
			validateJwt: true,
		},
		{
			name:        "Invalid JWT unverified",
			jwtToken:    "invalid.jwt.token",
			expectErr:   true,
			validateJwt: false,
		},
		{
			name:        "Missing JWT with validate",
			jwtToken:    "",
			expectErr:   true,
			validateJwt: true,
		},
		{
			name:        "Missing JWT unverified",
			jwtToken:    "",
			expectErr:   true,
			validateJwt: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			niceMD := metautils.NiceMD{}
			if tt.jwtToken != "" {
				niceMD.Add(authKey, "Bearer "+tt.jwtToken)
			}
			ctx := niceMD.ToIncoming(context.Background())

			res, err := rbac.ExtractClaimsFromContext(ctx, tt.validateJwt)
			if tt.expectErr {
				require.Error(t, err)
				assert.Nil(t, res)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, res)
				assert.NotNil(t, res.Get("realm_access/roles"))
			}
		})
	}
}

func Test_ClientCanBypassAuthN(t *testing.T) {
	p, err := loadPolicyBundle(regoPath)
	require.NoError(t, err)

	t.Setenv(rbac.AllowMissingAuthClients, clientName)

	niceMD := metautils.NiceMD{}
	ctx := niceMD.ToIncoming(context.Background())

	res := p.IsRequestAuthorized(ctx, rbac.PostKey)
	assert.False(t, res)

	niceMD.Add(clientKey, clientName)
	ctx = niceMD.ToIncoming(context.Background())

	res = p.IsRequestAuthorized(ctx, rbac.PostKey)
	assert.True(t, res)
}

func Test_AddClientNameToTheOutgoingContext(t *testing.T) {
	ctx := context.Background()
	ctx = rbac.AddClientNameToTheOutgoingContext(ctx, clientName)

	niceMD := metautils.ExtractOutgoing(ctx)
	retClientName := niceMD.Get(clientKey)
	assert.Equal(t, clientName, retClientName)
}

func Test_AddClientNameToTheIncomingContext(t *testing.T) {
	ctx := context.Background()
	ctx = rbac.AddClientNameToTheIncomingContext(ctx, clientName)

	niceMD := metautils.ExtractIncoming(ctx)
	retClientName := niceMD.Get(clientKey)
	assert.Equal(t, clientName, retClientName)
}

func Test_AddJWTToTheOutgoingContext(t *testing.T) {
	// creating a JWT with read and write roles
	_, jwtToken, err := inv_testing.CreateJWT(t, uuid.NewString())
	require.NoError(t, err)

	ctx := context.Background()
	ctx = rbac.AddJWTToTheOutgoingContext(ctx, jwtToken)

	niceMD := metautils.ExtractOutgoing(ctx)
	retAuth := niceMD.Get(authKey)
	retAuthTokens := strings.Split(retAuth, " ")
	require.Equal(t, 2, len(retAuthTokens))
	assert.Equal(t, jwtToken, retAuthTokens[1])
}

func Test_AddJWTToTheIncomingContext(t *testing.T) {
	// creating a JWT with read and write roles
	_, jwtToken, err := inv_testing.CreateJWT(t, uuid.NewString())
	require.NoError(t, err)

	ctx := context.Background()
	ctx = rbac.AddJWTToTheIncomingContext(ctx, jwtToken)

	niceMD := metautils.ExtractIncoming(ctx)
	retAuth := niceMD.Get(authKey)
	retAuthTokens := strings.Split(retAuth, " ")
	require.Equal(t, 2, len(retAuthTokens))
	assert.Equal(t, jwtToken, retAuthTokens[1])
}
