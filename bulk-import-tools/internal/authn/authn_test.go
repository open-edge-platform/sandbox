// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package authn_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/bulk-import-tools/internal/authn"
)

// Mock server to simulate token server/keycloak.
func setupMockServer() *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request method is POST
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		// Check if the request content type is application/x-www-form-urlencoded
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			http.Error(w, "Invalid content type", http.StatusUnsupportedMediaType)
			return
		}

		// Read and parse the form data
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Error parsing form data", http.StatusBadRequest)
			return
		}

		// Check for required form fields
		username := r.FormValue("username")
		password := r.FormValue("password")
		grantType := r.FormValue("grant_type")
		clientID := r.FormValue("client_id")
		scope := r.FormValue("scope")

		if username == "" || password == "" || grantType == "" || clientID == "" || scope == "" {
			http.Error(w, "Missing form fields", http.StatusBadRequest)
			return
		}

		// Successful token response
		tokenResponse := struct {
			AccessToken string `json:"access_token"`
		}{
			AccessToken: "mocked_token",
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(tokenResponse)
		assert.NoError(nil, err)
	})

	return httptest.NewServer(handler)
}

func TestAuthenticate(t *testing.T) {
	// Set up a mock server
	mockServer := setupMockServer()
	defer mockServer.Close()

	// Set environment variables for testing
	t.Setenv("EDGEORCH_USER", "testuser")
	t.Setenv("EDGEORCH_PASSWORD", "testpassword")
	defer func() {
		// Clean up environment variables after test
		os.Unsetenv("EDGEORCH_USER")
		os.Unsetenv("EDGEORCH_PASSWORD")
	}()

	u, err := url.Parse(mockServer.URL)
	require.NoError(t, err)
	token, err := authn.Authenticate(context.Background(), u)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	assert.Equal(t, "mocked_token", token)
}

func TestAuthenticateNoScheme(t *testing.T) {
	// Set environment variables for testing
	t.Setenv("EDGEORCH_USER", "testuser")
	t.Setenv("EDGEORCH_PASSWORD", "testpassword")
	defer func() {
		// Clean up environment variables after the test
		os.Unsetenv("EDGEORCH_USER")
		os.Unsetenv("EDGEORCH_PASSWORD")
	}()

	// URL error : No scheme
	_, err := authn.Authenticate(context.Background(), &url.URL{Host: "keyclock.test.com"})
	assert.Error(t, err)
}

func TestAuthenticateWithUserInput(t *testing.T) {
	// Set up a mock server
	mockServer := setupMockServer()
	defer mockServer.Close()

	// Create a pipe to simulate user input
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("Error creating pipe: %v", err)
	}

	// Simulate user input by writing to the writer end of the pipe
	input := "testuser\ntestpassword\n"
	go func() {
		defer writer.Close()
		_, err1 := writer.WriteString(input)
		require.NoError(t, err1)
	}()

	// Temporarily replace os.Stdin with the reader end of the pipe
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }() // Restore original Stdin after the test
	os.Stdin = reader
	u, err := url.Parse(mockServer.URL)
	require.NoError(t, err)
	token, err := authn.Authenticate(context.Background(), u)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	assert.Equal(t, "mocked_token", token)
}
