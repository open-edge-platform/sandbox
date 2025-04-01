// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"errors"
	"net/http"
	"sync"

	grpc_prom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var zlog = logging.GetLogger("Metrics")

const (
	defaultEndpoint = "/metrics"

	EnableMetrics             = "enableMetrics"
	EnableMetricsDescription  = "Enable Prometheus metric exporter"
	MetricsAddress            = "metricsAddress"
	MetricsAddressDescription = "The Metrics server address to serve on. It should have the following format <IP address>:<port>."
	MetricsAddressDefault     = ":8081"
)

var (
	once               sync.Once
	cliMetricsInstance *grpc_prom.ClientMetrics
)

func WithEndpoint(endpoint string) Option {
	return func(o *Options) {
		o.endpoint = endpoint
	}
}

func WithListenAddress(listenAddress string) Option {
	return func(o *Options) {
		o.listenAddress = listenAddress
	}
}

type Options struct {
	listenAddress string
	endpoint      string
}

type Option func(*Options)

// parseOptions parses the given list of Option into an Options.
func parseOptions(options ...Option) *Options {
	opts := &Options{
		endpoint:      defaultEndpoint,
		listenAddress: MetricsAddressDefault,
	}
	for _, option := range options {
		option(opts)
	}
	return opts
}

// StartMetricsExporter start a metrics exporter server given the options and with the given metrics server definition.
func StartMetricsExporter(metrics []prometheus.Collector, options ...Option) {
	opts := parseOptions(options...)
	go func() {
		zlog.Info().Msgf("Start metrics exporter server on: %s", opts.listenAddress)
		reg := prometheus.NewRegistry()
		for _, collector := range metrics {
			reg.MustRegister(collector)
		}
		metrics := echo.New()
		metrics.GET(opts.endpoint, echoprometheus.NewHandlerWithConfig(echoprometheus.HandlerConfig{Gatherer: reg}))
		if metricsErr := metrics.Start(opts.listenAddress); metricsErr != nil &&
			!errors.Is(metricsErr, http.ErrServerClosed) {
			zlog.Fatal().Err(metricsErr).Msg("failed to start metrics server")
		}
	}()
}

// GetServerMetricsWithLatency returns a metrics server definition with latency histogram. Used all across Infra components
// to have a shared and consistent metrics definition.
func GetServerMetricsWithLatency() *grpc_prom.ServerMetrics {
	return grpc_prom.NewServerMetrics(
		grpc_prom.WithServerHandlingTimeHistogram(),
	)
}

// GetClientMetricsWithLatency returns a client metrics definition with latency histogram. Used all across Infra components
// to have a shared and consistent metrics definition.
func GetClientMetricsWithLatency() *grpc_prom.ClientMetrics {
	once.Do(func() {
		cliMetricsInstance = grpc_prom.NewClientMetrics(
			grpc_prom.WithClientHandlingTimeHistogram(),
		)
	})

	return cliMetricsInstance
}
