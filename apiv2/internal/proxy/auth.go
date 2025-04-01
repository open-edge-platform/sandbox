// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/labstack/echo/v4"
	"google.golang.org/grpc/codes"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/policy/rbac"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/tenant"
	"github.com/open-edge-platform/orch-library/go/pkg/auth"
	grpcauth "github.com/open-edge-platform/orch-library/go/pkg/grpc/auth"
)

type contextKey string

// making it a constant to satisfy go-mnd linter.
const (
	authPairLen                        = 2
	authKey                 contextKey = "authorization"
	bearer                             = "bearer"
	rbacRules                          = "/rego/authz.rego"
	AllowMissingAuthClients            = "ALLOW_MISSING_AUTH_CLIENTS"
	RbacPolicyEnvVar                   = "RBAC_POLICY_PATH"
	roleProjectIDSeparator             = "_"
)

var tenantIDKey = "tenantid"

func validateAuthHeader(authHeader string) (
	authScheme string, authToken string, err error,
) {
	zlog.InfraSec().Debug().Msgf("parsing authorization header")
	authPair := strings.Split(authHeader, " ")
	if len(authPair) != authPairLen {
		err := fmt.Errorf("wrong Authorization header definition")
		zlog.InfraSec().InfraErr(err).
			Msgf("Expected to have 2 elements in authorization pair, got %d", len(authPair))
		return "", "", &echo.HTTPError{
			Code:    http.StatusUnauthorized,
			Message: http.StatusText(http.StatusUnauthorized),
		}
	}

	// Extracting Authentication Scheme type and JWT
	authScheme = authPair[0]
	authToken = authPair[1]

	zlog.InfraSec().Debug().Msgf("Verifying authentication scheme, it is %s", authScheme)
	if !strings.EqualFold(authScheme, bearer) {
		err := fmt.Errorf("wrong Authorization header definition. " +
			"Expecting \"Bearer\" Scheme to be sent")
		zlog.InfraSec().InfraErr(err).Msgf("A \"Bearer\" Authorization scheme was expected, got %s", authScheme)
		return "", "", &echo.HTTPError{
			Code:    http.StatusUnauthorized,
			Message: http.StatusText(http.StatusUnauthorized),
		}
	}
	return authScheme, authToken, nil
}

func validateJWT(authToken string) (jwt.Claims, error) {
	// verifying that JWT token is valid
	zlog.InfraSec().Debug().Msgf("validating JWT token")
	jwtAuth := new(auth.JwtAuthenticator)
	claims, err := jwtAuth.ParseAndValidate(authToken)
	if err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("JWT token is invalid or expired")
		return nil, &echo.HTTPError{
			Code:    http.StatusUnauthorized,
			Message: http.StatusText(http.StatusUnauthorized),
		}
	}
	zlog.InfraSec().Debug().Msgf("JWT token is valid, proceeding with processing")
	return claims, nil
}

func validateTenant(claims jwt.Claims, req *http.Request, p *rbac.Policy) error {
	// parsing claims -> prerequisite for Authorization
	claimsMap, isMap := claims.(jwt.MapClaims)
	if !isMap {
		err := fmt.Errorf("error converting claims to a map")
		zlog.InfraSec().InfraErr(err).Msgf("Failed to parse JWT claims")
		return &echo.HTTPError{
			Code:    http.StatusForbidden,
			Message: http.StatusText(http.StatusForbidden),
		}
	}

	niceMd := metautils.ExtractIncoming(req.Context())
	for k, v := range claimsMap {
		err := grpcauth.HandleClaim(&niceMd, []string{k}, v)
		if err != nil {
			zlog.InfraSec().InfraErr(err).Msgf("error handling claim")
			return &echo.HTTPError{
				Code:    http.StatusForbidden,
				Message: http.StatusText(http.StatusForbidden),
			}
		}
	}

	err := setTenantID(req.Context(), &niceMd)
	if err != nil {
		return &echo.HTTPError{
			Code:    http.StatusUnauthorized,
			Message: http.StatusText(http.StatusUnauthorized),
		}
	}
	zlog.Debug().Msgf("Request has tenantID, proceeding")
	// performing Authorization with OPA
	err = p.Verify(niceMd, req.Method)
	if err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("%v request can't be authorized", req.Method)
		return &echo.HTTPError{
			Code:    http.StatusForbidden,
			Message: http.StatusText(http.StatusForbidden),
		}
	}
	zlog.Debug().Msgf("Request is authorized, proceeding")

	return nil
}

// AuthenticationAuthorizationInterceptor performs REST call authentication (i.e., extracts JWT out of the call
// and checks that it is valid). This is necessary prerequisite for Role-Based Access Control (RBAC).
// It also authorizes the REST call.
func AuthenticationAuthorizationInterceptor(next echo.HandlerFunc) echo.HandlerFunc {
	// Retrieve the RBAC policy path from the environment variable or use a default value.
	rbacPolicyPath := os.Getenv(RbacPolicyEnvVar)
	if rbacPolicyPath == "" {
		rbacPolicyPath = rbacRules // Use the default constant if the environment variable is not set.
	}
	// starting OPA instance
	p, err := rbac.New(rbacPolicyPath)
	if err != nil {
		zlog.Fatal().Msgf("Can't upload RBAC policies to OPA package: %v", err)
		return nil
	}
	zlog.InfraSec().Debug().Msgf("OPA with RBAC policies is initialized")

	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("authorization")
		if authHeader == "" {
			err = fmt.Errorf("missing Authorization header")
			zlog.InfraSec().InfraErr(err).
				Msg("Expected to have authorization header in the request")
			return &echo.HTTPError{
				Code:    http.StatusUnauthorized,
				Message: http.StatusText(http.StatusUnauthorized),
			}
		}

		authScheme, authToken, err := validateAuthHeader(authHeader)
		if err != nil {
			return err
		}

		claims, err := validateJWT(authToken)
		if err != nil {
			return err
		}

		err = validateTenant(claims, c.Request(), p)
		if err != nil {
			return err
		}

		// including JWT token to the message metadata
		authValue := strings.ToLower(authScheme) + " " + authToken
		c.SetRequest(
			c.Request().WithContext(
				context.WithValue(c.Request().Context(),
					authKey,
					authValue)))
		return next(c)
	}
}

// setTenantID extracts a tenantID string from the provided context
// and adds it into the metadata md.
// It returns an Unauthenticated error if the tenantID is not provided in the context.
func setTenantID(ctx context.Context, md *metautils.NiceMD) error {
	tenantID, ok := tenant.GetTenantIDFromContext(ctx)
	if !ok {
		err := errors.Errorfc(codes.Unauthenticated, "TenantID not found in context")
		zlog.InfraSec().InfraErr(err).Msgf("failed tenantID JWT validation")
		return err
	}
	md.Set(tenantIDKey, tenantID)
	return nil
}
