// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//nolint:testpackage // use the same pkg to test unexported functions
package metrics

import (
	"context"
	"flag"
	"net/http"
	"os"
	"testing"
	"time"

	grpc_prom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestParseOptions(t *testing.T) {
	t.Run("OnlyEndpoint", func(t *testing.T) {
		opts := parseOptions(WithEndpoint("testEndpoint"))
		assert.Equal(t, "testEndpoint", opts.endpoint)
		assert.Equal(t, MetricsAddressDefault, opts.listenAddress)
	})

	t.Run("OnlyAddress", func(t *testing.T) {
		opts := parseOptions(WithListenAddress("testListenAddress"))
		assert.Equal(t, defaultEndpoint, opts.endpoint)
		assert.Equal(t, "testListenAddress", opts.listenAddress)
	})

	t.Run("BothEndpointAndAddress", func(t *testing.T) {
		opts := parseOptions(WithEndpoint("testEndpoint"), WithListenAddress("testListenAddress"))
		assert.Equal(t, "testEndpoint", opts.endpoint)
		assert.Equal(t, "testListenAddress", opts.listenAddress)
	})
}

func TestStartMetricsExporter(t *testing.T) {
	srvMetrics := grpc_prom.NewServerMetrics()
	go StartMetricsExporter([]prometheus.Collector{srvMetrics})
	// Wait for the server to start
	time.Sleep(1 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8081/metrics", http.NoBody)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetServerMetricsWithLatency(t *testing.T) {
	metrics := GetServerMetricsWithLatency()
	assert.NotNil(t, metrics)
}
