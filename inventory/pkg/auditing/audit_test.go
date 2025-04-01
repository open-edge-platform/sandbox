// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auditing_test

import (
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/auditing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

const (
	// SharedSecretKey environment variable name for shared secret key for signing a token.
	SharedSecretKey = "SHARED_SECRET_KEY"
	secretKey       = "randomSecretKey"
)

// Define this flag in order to call all tests with the same parameters.
var _ = flag.String(
	"policyBundle",
	"/rego/policy_bundle.tar.gz",
	"Path of policy rego file",
)

func TestAuditingMiddleware(t *testing.T) {
	// Create a buffer to capture log output
	buf := new(bytes.Buffer)
	writer := &logging.CustomWriter{Buf: buf}
	// injecting custom logger
	auditing.SetupLogger(logging.GetLoggerWithCustomWriter("Audit", writer))

	jwtStr, err := generateValidJWT(t)
	if err != nil {
		t.Errorf("Error signing token: %v", err)
	}

	e := echo.New()

	// Define a dummy handler
	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "{\"name\":\"Jon Snow\"}")
	}

	// Wrap the handler with the middleware
	wrappedHandler := auditing.RestEchoMiddleware(handler)

	tests := []struct {
		name       string
		apiKey     string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "Audit Log",
			wantStatus: http.StatusOK,
			wantBody:   "{\"name\":\"Jon Snow\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataJSON := `{"name":"Jon Snow"}`
			r := createRequestWithAuthHeader("Bearer", jwtStr, dataJSON)
			assert.Nil(t, err)
			rec := httptest.NewRecorder()
			c := e.NewContext(r, rec)

			// Execute the middleware
			if assert.NoError(t, wrappedHandler(c)) {
				capturedOutput := buf.String()
				assert.Equal(t, tt.wantStatus, rec.Code)
				assert.JSONEq(t, tt.wantBody, rec.Body.String())
				expectedAuditMessageContent := `"level":"info","component":"Audit","event":"auditmessage",` +
					`"operation":"POST","path":"/",`
				assert.Contains(t, capturedOutput, expectedAuditMessageContent)
				expectedIdentityContent := `"user":"testname","email":"test1@opennetworking.org"`
				assert.Contains(t, capturedOutput, expectedIdentityContent)
			}
		})
	}
}

// To create a request with an authorization header.
func createRequestWithAuthHeader(authScheme, authToken, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(auditing.AUTHORIZATION, fmt.Sprintf("%s %s", authScheme, authToken))
	return req
}

// To generates a valid JWT token for testing purposes.
func generateValidJWT(tb testing.TB) (jwtStr string, err error) {
	tb.Helper()
	tb.Setenv(SharedSecretKey, secretKey)
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		createCustomClaims())
	jwtStr, err = token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}
	return jwtStr, nil
}

type RealmAccess struct {
	Roles []string
}

type Account struct {
	Roles []string `json:"roles"`
}

type ResourceAccess struct {
	Account Account `json:"account"`
}
type TestCustomClaims struct {
	jwt.RegisteredClaims
	Name              string         `json:"name"`
	Email             string         `json:"email"`
	EmailVerified     bool           `json:"email_verified"`
	PreferredUsername string         `json:"preferred_username"`
	Groups            []string       `json:"groups"`
	Roles             []string       `json:"roles"`
	Foo               int            `json:"foo"`
	Foo32             int32          `json:"foo32"`
	RealmAccess       RealmAccess    `json:"realm_access"`
	ResourceAccess    ResourceAccess `json:"resource_access"`
}

func createCustomClaims() TestCustomClaims {
	now := time.Now()
	return TestCustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "http://dex:32000",
			Subject:   "Test_AuthenticationInterceptor",
			Audience:  []string{"testaudience"},
			ExpiresAt: &jwt.NumericDate{Time: now.Add(24 * time.Hour)},
			NotBefore: &jwt.NumericDate{Time: now},
			IssuedAt:  &jwt.NumericDate{Time: now},
			ID:        "",
		},
		Name:              "testname",
		Email:             "test1@opennetworking.org",
		EmailVerified:     true,
		PreferredUsername: "a user Name",
		Groups:            []string{"testGroup1", "testGroup2"},
		Roles:             []string{"testRole1", "testRole2"},
		Foo:               21,
		Foo32:             22,
		RealmAccess: RealmAccess{
			Roles: []string{
				"testRole1",
				"testRole2",
			},
		},
		ResourceAccess: ResourceAccess{
			Account: Account{
				Roles: []string{
					"testRole1",
					"testRole2",
				},
			},
		},
	}
}
