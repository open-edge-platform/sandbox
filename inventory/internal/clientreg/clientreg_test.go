// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package clientreg_test

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/clientreg"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// Currently unused
	flag.String(
		"policyBundle",
		wd+"/../../out/policy_bundle.tar.gz",
		"Path of policy rego file",
	)
	flag.Parse()
	projectRoot := filepath.Dir(filepath.Dir(wd))

	policyPath := projectRoot + "/out"
	certPath := projectRoot + "/cert/certificates"
	migrationsDir := projectRoot + "/out"

	inv_testing.StartTestingEnvironment(policyPath, certPath, migrationsDir)
	run := m.Run() // run all tests
	inv_testing.StopTestingEnvironment()

	os.Exit(run)
}

func TestNewClientReg(t *testing.T) {
	cr := clientreg.NewClientReg(true)
	assert.NotNil(t, cr, "expected non-nil client reg")
	cr = clientreg.NewClientReg(false)
	assert.NotNil(t, cr, "expected non-nil client reg")
}

type mockGrpcStream struct {
	grpc.ServerStream
	Events []*inv_v1.SubscribeEventsResponse
}

func (stream *mockGrpcStream) Send(resp *inv_v1.SubscribeEventsResponse) error {
	stream.Events = append(stream.Events, resp)
	return nil
}

func TestClientReg_RegisterClient(t *testing.T) {
	tests := []struct {
		name       string
		clientInfo clientreg.ClientInfo
		valid      bool
	}{
		{name: "valid", clientInfo: clientreg.ClientInfo{
			Name:          "some name",
			Version:       "",
			ClientKind:    inv_v1.ClientKind_CLIENT_KIND_API,
			ResourceKinds: []inv_v1.ResourceKind{inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE},
			Stream:        &mockGrpcStream{},
		}, valid: true},
		{name: "missing name", clientInfo: clientreg.ClientInfo{
			Name:          "",
			Version:       "",
			ClientKind:    inv_v1.ClientKind_CLIENT_KIND_API,
			ResourceKinds: nil,
			Stream:        &mockGrpcStream{},
		}, valid: false},
		{name: "missing stream", clientInfo: clientreg.ClientInfo{
			Name:          "some name",
			Version:       "",
			ClientKind:    inv_v1.ClientKind_CLIENT_KIND_API,
			ResourceKinds: nil,
			Stream:        nil,
		}, valid: false},
		{name: "invalid kind", clientInfo: clientreg.ClientInfo{
			Name:          "some name",
			Version:       "",
			ClientKind:    inv_v1.ClientKind_CLIENT_KIND_UNSPECIFIED,
			ResourceKinds: nil,
			Stream:        &mockGrpcStream{},
		}, valid: false},
		{name: "duplicate resource kind", clientInfo: clientreg.ClientInfo{
			Name:       "some name",
			Version:    "",
			ClientKind: inv_v1.ClientKind_CLIENT_KIND_API,
			ResourceKinds: []inv_v1.ResourceKind{
				inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE,
				inv_v1.ResourceKind_RESOURCE_KIND_INSTANCE,
			},
			Stream: &mockGrpcStream{},
		}, valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := clientreg.NewClientReg(true)
			got, err := cr.RegisterClient(tt.clientInfo)
			if !tt.valid {
				if err == nil {
					t.Fatalf("RegisterClient(%v) error = %v, valid %v", tt.clientInfo, err, tt.valid)
				}
				return
			}

			id, err := uuid.Parse(got)
			require.NoErrorf(t, err, "RegisterClient(%v) did not return a valid UUID '%v'", tt.clientInfo, got)

			cr.ExitClient(id.String())
		})
	}
}

func TestClientReg_StreamNotify(t *testing.T) {
	cl1 := clientreg.ClientInfo{
		Name:          "client1",
		Version:       "",
		ClientKind:    inv_v1.ClientKind_CLIENT_KIND_API,
		ResourceKinds: []inv_v1.ResourceKind{inv_v1.ResourceKind_RESOURCE_KIND_HOST, inv_v1.ResourceKind_RESOURCE_KIND_SITE},
		Stream:        &mockGrpcStream{},
	}
	cl2 := clientreg.ClientInfo{
		Name:          "client2",
		Version:       "",
		ClientKind:    inv_v1.ClientKind_CLIENT_KIND_RESOURCE_MANAGER,
		ResourceKinds: []inv_v1.ResourceKind{inv_v1.ResourceKind_RESOURCE_KIND_HOST},
		Stream:        &mockGrpcStream{},
	}
	cr := clientreg.NewClientReg(true)
	id1, err := cr.RegisterClient(cl1)
	require.NoError(t, err)
	defer cr.ExitClient(id1)
	id2, err := cr.RegisterClient(cl2)
	require.NoError(t, err)
	defer cr.ExitClient(id2)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Send 1st notification from client 1 and check that only client 2 receives it.
	host, err := util.WrapResource(inv_testing.CreateHost(t, nil, nil))
	require.NoError(t, err)
	cr.StreamNotify(ctx, inv_v1.SubscribeEventsResponse_EVENT_KIND_CREATED, host, id1)
	grpcStream1, ok := cl1.Stream.(*mockGrpcStream)
	require.True(t, ok)
	require.Lenf(t, grpcStream1.Events, 0, "client 1 received notification from self")
	grpcStream2, ok := cl2.Stream.(*mockGrpcStream)
	require.True(t, ok)
	require.Lenf(t, grpcStream2.Events, 1, "client 2 did not receive event notification")
	require.Equal(t, host, grpcStream2.Events[0].Resource)
	require.Equal(t, host.GetHost().GetResourceId(), grpcStream2.Events[0].ResourceId)
	require.Equal(t, inv_v1.SubscribeEventsResponse_EVENT_KIND_CREATED, grpcStream2.Events[0].EventKind)
	grpcStream2.Events = []*inv_v1.SubscribeEventsResponse{}

	// Send 2nd notification from client 2 and check that no client receives it.
	osr, err := util.WrapResource(inv_testing.CreateOs(t))
	require.NoError(t, err)
	cr.StreamNotify(ctx, inv_v1.SubscribeEventsResponse_EVENT_KIND_UPDATED, osr, id2)
	grpcStream1, ok = cl1.Stream.(*mockGrpcStream)
	require.True(t, ok)
	require.Lenf(t, grpcStream1.Events, 0, "client 1 received notification for resource it did not subscribe to")
	grpcStream2, ok = cl2.Stream.(*mockGrpcStream)
	require.True(t, ok)
	require.Lenf(t, grpcStream2.Events, 0, "client 2 received notification for resource it did not subscribe to")

	// Send 3rd notification with an empty resource.
	res := &inv_v1.Resource{}
	cr.StreamNotify(ctx, inv_v1.SubscribeEventsResponse_EVENT_KIND_UPDATED, res, id2)
	grpcStream1, ok = cl1.Stream.(*mockGrpcStream)
	require.True(t, ok)
	require.Lenf(t, grpcStream1.Events, 0, "client 1 received notification for invalid resource")
	grpcStream2, ok = cl2.Stream.(*mockGrpcStream)
	require.True(t, ok)
	require.Lenf(t, grpcStream2.Events, 0, "client 2 received notification for invalid resource")
}
