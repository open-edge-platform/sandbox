// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auditing

import (
	"context"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/labstack/echo/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/orch-library/go/pkg/auth"
	grpcauth "github.com/open-edge-platform/orch-library/go/pkg/grpc/auth"
)

const (
	NAME               = "Name"
	EMAIL              = "Email"
	AUTHORIZATIONLOWER = "authorization"
	AUTHORIZATION      = "Authorization"
	NAMELOWER          = "name"
	EMAILLOWER         = "email"
	UNKNOWN            = "UNKNOWN"
	authPairLen        = 2
	bearer             = "bearer"
)

var zlog = logging.GetLogger("Audit")

// GrpcInterceptor unary interceptor function to audit the operation done via gRPC.
func GrpcInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	// Log the request
	createPreRequestAuditLog(ctx, info.FullMethod, req, "Operation request")

	// Call the handler to complete the normal execution of the method
	resp, err := handler(ctx, req)

	// Log the response and any error
	createPostRequestAuditLog(ctx, info.FullMethod, resp, err)

	return resp, err
}

// RestEchoMiddleware ECHO middleware to audit the operation done via REST.
func RestEchoMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO worth considering data when POST/PATCH
		usr, email := extractUserAndEmailfromJWT(c)
		zlog.InfraAuditEvent().InfraAuditOperation(c.Request().Method).InfraAuditPath(c.Request().URL.String()).
			InfraAuditUsr(usr).InfraAuditEmail(email).Info().
			Msgf("Northbound API Operation")
		return next(c)
	}
}

// createPreRequestAuditLog generate an audit log with user and operation done based on the existing gRPC call.
func createPreRequestAuditLog(ctx context.Context, fullMethod string, req interface{}, message string) {
	md, existing := metadata.FromIncomingContext(ctx)
	user := getValueFromMetadata(existing, md, NAME)
	email := getValueFromMetadata(existing, md, EMAIL)
	zlog.InfraAuditEvent().InfraAuditOperation(fullMethod).InfraAuditUsr(strings.Join(user, ",")).
		InfraAuditEmail(strings.Join(email, ",")).InfraAuditRequest(req).Info().Msgf(message+" %v", md)
}

// createPostRequestAuditLog generate an audit log with user and operation done based on result of the gRPC call.
func createPostRequestAuditLog(ctx context.Context, fullMethod string, resp interface{}, err error) {
	md, existing := metadata.FromIncomingContext(ctx)
	user := getValueFromMetadata(existing, md, NAME)
	email := getValueFromMetadata(existing, md, EMAIL)

	if err != nil {
		st, _ := status.FromError(err)
		zlog.InfraAuditEvent().InfraAuditOperation(fullMethod).InfraAuditUsr(strings.Join(user, ",")).
			InfraAuditEmail(strings.Join(email, ",")).InfraAuditResponse(resp).InfraAuditError(err).
			InfraAuditStatus(st.String()).Info().Msgf("Operation result %v", md)
	} else {
		zlog.InfraAuditEvent().InfraAuditOperation(fullMethod).InfraAuditUsr(strings.Join(user, ",")).
			InfraAuditEmail(strings.Join(email, ",")).InfraAuditResponse(resp).Info().Msgf("Operation result %v", md)
	}
}

func SetupLogger(logger logging.InfraLogger) {
	// Set up zerolog with the custom writer
	zlog = logger
}

// TODO ITEP-2566 when moving before authentication we can't use the context
// in either of the following methods. JWT is present but needs to be parsed, see lib go from app-orch and
// AuthenticationAuthorizationInterceptor from API.

// extracts user from metadata (gRPC).
// see auth/auth_test.go.
func getValueFromMetadata(existing bool, md metadata.MD, key string) []string {
	var value []string
	if existing && len(md.Get(key)) > 0 {
		value = md.Get(key)
	} else {
		value = append(value, key)
	}
	return value
}

// TODO need to refactor with common code from app-lib-go library
//
//nolint:cyclop // higher calculated cyclomatic complexity due to additional validation
func extractUserAndEmailfromJWT(c echo.Context) (string, string) {
	niceMd := metautils.ExtractIncoming(c.Request().Context())
	authHeader := c.Request().Header.Get(AUTHORIZATIONLOWER)
	var usr string
	var email string
	if authHeader == "" {
		// re-try if the extraction is case-sensitive
		authHeader = c.Request().Header.Get(AUTHORIZATION)
	}

	if authHeader == "" {
		return UNKNOWN, UNKNOWN
	}

	authPair := strings.Split(authHeader, " ")
	if len(authPair) != authPairLen {
		return UNKNOWN, UNKNOWN
	}

	// Extracting Authentication Scheme type and JWT
	authScheme := authPair[0]
	authToken := authPair[1]
	if !strings.EqualFold(authScheme, bearer) {
		return UNKNOWN, UNKNOWN
	}

	// Authenticate the jwt token
	jwtAuth := new(auth.JwtAuthenticator)
	authClaimsIf, err := jwtAuth.ParseAndValidate(authToken)
	if err != nil {
		return UNKNOWN, UNKNOWN
	}

	authClaims, isMap := authClaimsIf.(jwt.MapClaims)
	if !isMap {
		return UNKNOWN, UNKNOWN
	}
	for k, v := range authClaims {
		err = grpcauth.HandleClaim(&niceMd, []string{k}, v)
		if err != nil {
			return UNKNOWN, UNKNOWN
		}
	}
	if niceMd.Get(NAMELOWER) == "" {
		usr = UNKNOWN
	} else {
		usr = niceMd.Get(NAME)
	}

	if niceMd.Get(EMAILLOWER) == "" {
		email = UNKNOWN
	} else {
		email = niceMd.Get(EMAILLOWER)
	}

	return usr, email
}
