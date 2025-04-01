// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package oam_test

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/oam"
)

func validateGetServingStatus(t *testing.T, actual *grpc_health_v1.HealthCheckResponse_ServingStatus,
	expected grpc_health_v1.HealthCheckResponse_ServingStatus, err error,
) {
	t.Helper()

	switch {
	case err == nil && *actual == expected:
		// good one!
		return
	case err == nil && *actual != expected:
		t.Errorf("GetServingStatus() succeeded but status is %s", *actual)
	case err != nil:
		t.Errorf("GetServingStatus() failed: %s", err)
	default:
		t.Errorf("GetServingStatus() unhandled status/err %s/%s ", *actual, err)
	}
}

func validateWatchServingStatus(t *testing.T, stream grpc_health_v1.Health_WatchClient, err error) {
	t.Helper()

	switch {
	case err == nil && stream != nil:
		// get next event
		_, rcverr := stream.Recv()
		if rcverr != nil {
			if stat, ok := status.FromError(rcverr); ok && stat.Code() != codes.Unimplemented {
				t.Errorf("WatchServingStatus() should be %s but is %s", codes.Unimplemented, stat.Code())
			}
		} else {
			t.Errorf("WatchServingStatus() should be %s", codes.Unimplemented)
		}
	default:
		t.Errorf("WatchServingStatus() unhandled stream/err %s/%s ", stream, err)
	}
}

func Test_Not_Serving(t *testing.T) {
	toam := oam.NewTestOAM()
	toam.StartTestOAM()
	defer toam.StopTestOAM()

	testcases := map[string]struct {
		serviceID string
	}{
		"Global": {
			serviceID: "",
		},
		"Foo": {
			serviceID: "foo",
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			// initially is not ready
			servingStatus, err := oam.TestClient.GetServingStatus(ctx, tc.serviceID)
			validateGetServingStatus(t, servingStatus, grpc_health_v1.HealthCheckResponse_NOT_SERVING, err)
		})
	}
}

func Test_Serving(t *testing.T) {
	toam := oam.NewTestOAM()
	toam.StartTestOAM()
	defer toam.StopTestOAM()

	testcases := map[string]struct {
		serviceID string
	}{
		"Global": {
			serviceID: "",
		},
		"Foo": {
			serviceID: "foo",
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			// initially not serving
			servingStatus, err := oam.TestClient.GetServingStatus(ctx, tc.serviceID)
			validateGetServingStatus(t, servingStatus, grpc_health_v1.HealthCheckResponse_NOT_SERVING, err)

			// signal ready
			oam.TestReadyChan <- true

			servingStatus, err = oam.TestClient.GetServingStatus(ctx, tc.serviceID)
			validateGetServingStatus(t, servingStatus, grpc_health_v1.HealthCheckResponse_SERVING, err)

			// clean for the following tc
			oam.TestReadyChan <- false
		})
	}
}

func Test_Unimplemented_Watch(t *testing.T) {
	toam := oam.NewTestOAM()
	toam.StartTestOAM()
	defer toam.StopTestOAM()

	testcases := map[string]struct {
		serviceID string
	}{
		"Global": {
			serviceID: "",
		},
		"Foo": {
			serviceID: "foo",
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			stream, err := oam.TestClient.WatchServingStatus(ctx, tc.serviceID)
			validateWatchServingStatus(t, stream, err)
		})
	}
}
