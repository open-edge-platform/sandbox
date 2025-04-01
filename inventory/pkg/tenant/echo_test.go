// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tenant_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/tenant"
)

var tenantUUID = uuid.New().String()

// To create a request with a tenant header.
func createRequestWithTenantHeader(projectIDKey, projectID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set(projectIDKey, projectID)
	return req
}

func TestTenantInterceptor(t *testing.T) {
	// Create an Echo instance for testing.
	e := echo.New()

	tests := []struct {
		name           string
		request        *http.Request
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "No Tenant header and no allowed missing tenant info",
			request:        httptest.NewRequest(http.MethodGet, "/", http.NoBody),
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "missing Tenant header",
		},
		{
			name:           "Invalid Tenant header format",
			request:        createRequestWithTenantHeader("invalid_format", "invalid-uuid"),
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "wrong Tenant header definition",
		},
		{
			name:           "Tenant header with correct header but invalid uuid format",
			request:        createRequestWithTenantHeader(tenant.TenantKey, "invalid-uuid-format"),
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "wrong Tenant header uuid definition",
		},
		{
			name:           "Tenant header with correct header but invalid uuid format",
			request:        createRequestWithTenantHeader(tenant.TenantKey, ""),
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Expecting uuid value to be sent in tenantKey",
		},
		{
			name:           "Tenant header with correct key and valid uuid",
			request:        createRequestWithTenantHeader(tenant.TenantKey, tenantUUID),
			expectedStatus: http.StatusOK,
			expectedError:  "Tenant is valid, proceeding with processing",
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
			// Invoke interceptor.
			handler := tenant.TenantInterceptor(next)
			err := handler(c)
			if tt.expectedStatus == http.StatusOK {
				assert.NoError(t, err)
				// Makes sure the tenant ID was added to the context
				assert.Equal(t, tenantUUID, c.Request().Context().Value(tenant.CtxTenantIDKey))
			} else {
				var httpErr *echo.HTTPError
				if errors.As(err, &httpErr) {
					assert.Equal(t, http.StatusUnauthorized, httpErr.Code, "Expected an HTTP 401 Unauthorized error")
				} else {
					t.Errorf("Expected an echo.HTTPError, got %T", err)
				}
			}
		})
	}
}
