// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"google.golang.org/grpc/codes"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/orch-library/go/pkg/auth"
	grpcauth "github.com/open-edge-platform/orch-library/go/pkg/grpc/auth"
)

const (
	// ContextMetadataBearerKeyLower metadata JWT token key.
	ContextMetadataBearerKeyLower = "bearer"
	// ContextMetadataBearerKeyCamel metadata JWT token key.
	ContextMetadataBearerKeyCamel = "Bearer"
	AllowMissingAuthClients       = "ALLOW_MISSING_AUTH_CLIENTS"
	clientKeyLower                = "client"
	clientKeyCamel                = "Client"
	authKey                       = "authorization"
)

// IsRequestAuthorized function validates the JWT token included in a context.
// It also starts the OPA instance and performs the RBAC authorization of the call.
func (p *Policy) IsRequestAuthorized(ctx context.Context, operation string) bool {
	// check if client is the one which is set to bypass authorization
	if CanBypassAuthN(ctx) {
		return true
	}

	// assuming that the client should not bypass authorization
	md, err := ExtractClaimsFromContext(ctx, true)
	if err != nil {
		return false
	}
	// performing RBAC authorization
	err = p.Verify(md, operation)
	return err == nil
}

// CanBypassAuthN checks if user can bypass AuthN (authentication + authorization) by
// checking the environmental variable.
func CanBypassAuthN(ctx context.Context) bool {
	niceMd := metautils.ExtractIncoming(ctx)
	acceptNoAuth := os.Getenv(AllowMissingAuthClients)
	if acceptNoAuth == "" {
		// no clients to bypass AuthN specified
		return false
	}
	allowedMissingClients := strings.Split(acceptNoAuth, ",")
	requestClient := niceMd.Get(clientKeyLower)
	if requestClient == "" {
		// re-try to read with the other key
		requestClient = niceMd.Get(clientKeyCamel)
		if requestClient == "" {
			// no client name specified in the context, AuthN should be performed
			return false
		}
	}
	var foundMissingAuthClient bool
	for _, amc := range allowedMissingClients {
		if strings.ToLower(requestClient) == strings.TrimSpace(strings.ToLower(amc)) {
			foundMissingAuthClient = true
			break
		}
	}
	if foundMissingAuthClient {
		zlog.Warn().Msgf("Allowing unauthenticated gRPC request from client: %s", niceMd.Get("client"))
		return true
	}

	zlog.Debug().Msgf("Client %s is not allowed to bypass authorization", requestClient)
	return false
}

func ExtractClaimsFromContext(ctx context.Context, validateJWT bool) (metautils.NiceMD, error) {
	niceMd := metautils.ExtractIncoming(ctx)

	// Extract token from metadata in the context
	tokenString1, err1 := grpc_auth.AuthFromMD(ctx, ContextMetadataBearerKeyLower)
	tokenString2, err2 := grpc_auth.AuthFromMD(ctx, ContextMetadataBearerKeyCamel)
	if err1 != nil && err2 != nil {
		// JWT is not found in the context
		zlog.InfraSec().InfraErr(err2).Msgf("Failed to extract JWT token from the context")
		return nil, err2
	}
	// JWT is found, extracting it
	var tokenString string
	if err1 == nil {
		tokenString = tokenString1
	}
	if err2 == nil {
		tokenString = tokenString2
	}
	authClaims, err := extractAuthClaims(tokenString, validateJWT)
	if err != nil {
		return nil, err
	}

	for k, v := range authClaims {
		err := grpcauth.HandleClaim(&niceMd, []string{k}, v)
		if err != nil {
			zlog.InfraSec().InfraErr(err).Msgf("Failed to handle claim in JWT token")
			return nil, err
		}
	}

	zlog.Debug().Msg("JWT token is valid, proceeding to RBAC")
	return niceMd, nil
}

func extractAuthClaims(tokenString string, validateJWT bool) (jwt.MapClaims, error) {
	var isMap bool
	var authClaims jwt.MapClaims
	if validateJWT {
		// Authenticate the jwt token
		jwtAuth := new(auth.JwtAuthenticator)
		authClaimsIf, err := jwtAuth.ParseAndValidate(tokenString)
		if err != nil {
			zlog.InfraSec().InfraErr(err).Msgf("Failed to parse and validate JWT token")
			return nil, err
		}
		authClaims, isMap = authClaimsIf.(jwt.MapClaims)
	} else {
		token, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
		if err != nil {
			zlog.InfraSec().InfraErr(err).Msgf("Failed to parse JWT token")
			return nil, err
		}
		authClaims, isMap = token.Claims.(jwt.MapClaims)
	}
	if !isMap {
		err := errors.Errorfc(codes.Internal, "error converting claims to a map")
		zlog.InfraSec().InfraErr(err).Msgf("Failed to convert claims into a map")
		return nil, err
	}
	return authClaims, nil
}

func AddClientNameToTheIncomingContext(ctx context.Context, clientName string) context.Context {
	niceMD := metautils.NiceMD{}
	niceMD.Add(clientKeyCamel, clientName)
	return niceMD.ToIncoming(ctx)
}

func AddClientNameToTheOutgoingContext(ctx context.Context, clientName string) context.Context {
	niceMD := metautils.NiceMD{}
	niceMD.Add(clientKeyCamel, clientName)
	return niceMD.ToOutgoing(ctx)
}

func AddJWTToTheOutgoingContext(ctx context.Context, jwtToken string) context.Context {
	niceMD := metautils.NiceMD{}
	niceMD.Add(authKey, "Bearer "+jwtToken)
	return niceMD.ToOutgoing(ctx)
}

func AddJWTToTheIncomingContext(ctx context.Context, jwtToken string) context.Context {
	niceMD := metautils.NiceMD{}
	niceMD.Add(authKey, "Bearer "+jwtToken)
	return niceMD.ToIncoming(ctx)
}
