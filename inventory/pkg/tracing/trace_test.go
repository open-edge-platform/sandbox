// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tracing_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/tracing"
)

var (
	tracingAddress  = ""
	tracingService  = "test"
	tracingSpanName = "span"
	tracingAttribs  = map[string]string{
		"key": "value",
	}
)

func TestTracing(t *testing.T) {
	ctx := context.Background()

	shutdownTraceHTTP, err := tracing.NewTraceExporterHTTP(tracingAddress, tracingService, tracingAttribs)
	require.NoError(t, err)
	err = shutdownTraceHTTP(ctx)
	require.NoError(t, err)

	shutdownTraceHTTP, err = tracing.NewTraceExporterHTTP(tracingAddress, tracingService, nil)
	require.NoError(t, err)
	err = shutdownTraceHTTP(ctx)
	require.NoError(t, err)

	// exporterGRPC does not return error on connection, as connection is only established in the first gRPC call.
	shutdownTraceGRPC, err := tracing.NewTraceExporterGRPC(tracingAddress, tracingService, tracingAttribs)
	require.NoError(t, err)
	err = shutdownTraceGRPC(ctx)
	require.NoError(t, err)

	shutdownTraceGRPC, err = tracing.NewTraceExporterGRPC(tracingAddress, tracingService, nil)
	require.NoError(t, err)
	err = shutdownTraceGRPC(ctx)
	require.NoError(t, err)

	tracing.StartTrace(ctx, tracingService, tracingSpanName)
	defer tracing.StopTrace(ctx)

	optsClient := []grpc.DialOption{}
	optsClient = tracing.EnableGrpcClientTracing(optsClient)
	assert.NotNil(t, optsClient)

	optsServer := []grpc.ServerOption{}
	optsServer = tracing.EnableGrpcServerTracing(optsServer)
	assert.NotNil(t, optsServer)
}
