// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package perf

import (
	"flag"
	"net/http"
	_ "net/http/pprof" // Only imported for testing purposes.

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var zlog = logging.GetLogger("InfraPprof")

const (
	defaultPprofServerAddress = "0.0.0.0:6060"
	ServerAddress             = "pprofServerAddress"
	ServerAddressDescription  = "The endpoint address pprof to serve on. " +
		"It should have the following format <IP address>:<port>."
)

//nolint:gochecknoinits // Using init for defining flags is a valid exception.
func init() {
	flag.Func(
		ServerAddress,
		ServerAddressDescription,
		startPprofHTTPServer,
	)
}

func startPprofHTTPServer(address string) error {
	if address == "" {
		address = defaultPprofServerAddress
	}
	go func() {
		err := http.ListenAndServe(address, nil)
		zlog.InfraSec().Err(err).Msgf("failed to initialize pprof http server at %s", address)
	}()
	return nil
}
