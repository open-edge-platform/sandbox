// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/open-edge-platform/infra-core/api/internal/auth"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/tenant"
)

const (
	// SharedSecretKey environment variable name for shared secret key for signing a token.
	SharedSecretKey = "SHARED_SECRET_KEY"
	secretKey       = "randomSecretKey"
	writeRole       = "im-rw"
	readRole        = "im-r"
)

var tenantUUID = uuid.New().String()

// To create a request with an authorization header.
func createRequestWithAuthHeader(authScheme, authToken string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", authScheme, authToken))
	return req
}

// To create a request with a User-Agent header.
func createRequestWithUserAgent(userAgent string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("User-Agent", userAgent)
	return req
}

// To generates a valid JWT token for testing purposes.
func generateValidJWT(tb testing.TB, tenantID string) (jwtStr string, err error) {
	tb.Helper()
	tenantWriteRole := tenantID + "_" + writeRole
	tenantReadRole := tenantID + "_" + readRole
	claims := &jwt.MapClaims{
		"iss": "https://keycloak.kind.internal/realms/master",
		"exp": time.Now().Add(time.Hour).Unix(),
		"typ": "Bearer",
		"realm_access": map[string]interface{}{
			"roles": []string{
				tenantWriteRole,
				tenantReadRole,
			},
		},
	}
	tb.Setenv(SharedSecretKey, secretKey)
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims)
	jwtStr, err = token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}
	return jwtStr, nil
}

//nolint:funlen // it is a test
func TestAuthenticationAuthorizationInterceptor(t *testing.T) {
	// Set up the environment variable for allowed missing auth clients.
	t.Setenv(auth.AllowMissingAuthClients, "test-client")
	defer os.Unsetenv(auth.AllowMissingAuthClients)

	// Set the RBAC policy path to a test file.
	testRbacPolicyPath := "../../rego/authz.rego"
	t.Setenv(auth.RbacPolicyEnvVar, testRbacPolicyPath)
	defer os.Unsetenv(auth.RbacPolicyEnvVar)

	jwtStrWithoutTenant, err := generateValidJWT(t, "")
	if err != nil {
		t.Errorf("Error signing token: %v", err)
	}
	jwtStrWithTenant, err := generateValidJWT(t, tenantUUID)
	if err != nil {
		t.Errorf("Error signing token: %v", err)
	}
	jwtStrInvalid, err := generateValidJWT(t, "abc")
	if err != nil {
		t.Errorf("Error signing token: %v", err)
	}
	// Create an Echo instance for testing.
	e := echo.New()

	tests := []struct {
		name               string
		request            *http.Request
		expectedStatus     int
		expectedError      string
		addTenantToContext bool
	}{
		{
			name:               "No Authorization header and no allowed missing auth client",
			request:            httptest.NewRequest(http.MethodGet, "/", http.NoBody),
			expectedStatus:     http.StatusUnauthorized,
			expectedError:      "missing Authorization header",
			addTenantToContext: true,
		},
		{
			name:               "Invalid Authorization header format",
			request:            createRequestWithAuthHeader("invalid_format", "token"),
			expectedStatus:     http.StatusUnauthorized,
			expectedError:      "wrong Authorization header definition",
			addTenantToContext: true,
		},
		{
			name:               "Authorization header with Bearer scheme but invalid JWT token",
			request:            createRequestWithAuthHeader("Bearer", "invalid-token"),
			expectedStatus:     http.StatusUnauthorized,
			expectedError:      "JWT token is invalid or expired",
			addTenantToContext: true,
		},
		{
			name:               "Authorization header with non-Bearer scheme",
			request:            createRequestWithAuthHeader("Basic", "token"),
			expectedStatus:     http.StatusUnauthorized,
			expectedError:      "Expecting \"Bearer\" Scheme to be sent",
			addTenantToContext: true,
		},
		{
			name:               "Authorization header with Bearer scheme with valid JWT token and no tenantID",
			request:            createRequestWithAuthHeader("Bearer", jwtStrWithoutTenant),
			expectedStatus:     http.StatusUnauthorized,
			expectedError:      "JWT token is valid, but tenantID was not passed in context",
			addTenantToContext: false,
		},
		{
			name:               "Authorization header with Bearer scheme with valid JWT token/tenantID but context invalid",
			request:            createRequestWithAuthHeader("Bearer", jwtStrWithTenant),
			expectedStatus:     http.StatusUnauthorized,
			expectedError:      "JWT token is valid, but tenantID was not passed in context",
			addTenantToContext: false,
		},
		{
			name:               "Authorization header with Bearer scheme with valid JWT token and tenantID",
			request:            createRequestWithAuthHeader("Bearer", jwtStrWithTenant),
			expectedStatus:     http.StatusOK,
			expectedError:      "JWT token is valid, proceeding with processing",
			addTenantToContext: true,
		},
		{
			name:               "Allowed missing auth client",
			request:            createRequestWithUserAgent("test-client"),
			expectedStatus:     http.StatusOK,
			addTenantToContext: true,
		},
		{
			name:               "Allowed missing auth client, but no tenantID in context",
			request:            createRequestWithUserAgent("test-client"),
			expectedStatus:     http.StatusOK,
			addTenantToContext: false,
		},
		{
			name:               "Authorization header with Bearer scheme with valid JWT without tenantID",
			request:            createRequestWithAuthHeader("Bearer", jwtStrInvalid),
			expectedStatus:     http.StatusForbidden,
			expectedError:      "JWT token is invalid, no tenantID in JWT roles",
			addTenantToContext: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a context with the request and recorder.
			c := e.NewContext(tt.request, httptest.NewRecorder())
			// Create a dummy next handler that returns OK status.
			next := func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			}
			if tt.addTenantToContext {
				c.SetRequest(
					c.Request().WithContext(
						tenant.AddTenantIDToContext(c.Request().Context(), tenantUUID),
					),
				)
			}
			// Invoke interceptor.
			handler := auth.AuthenticationAuthorizationInterceptor(next)
			err := handler(c)
			if tt.expectedStatus == http.StatusOK {
				assert.NoError(t, err)
			} else {
				var httpErr *echo.HTTPError
				if errors.As(err, &httpErr) {
					assert.Equal(t, tt.expectedStatus, httpErr.Code, "Expected an HTTP 401 Unauthorized error")
				} else {
					t.Errorf("Expected an echo.HTTPError, got %T", err)
				}
			}
		})
	}
}
