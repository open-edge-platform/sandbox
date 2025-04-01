// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package clients

import (
	"context"
	"sync"

	"google.golang.org/grpc"

	"github.com/open-edge-platform/infra-core/api/internal/common"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
)

var invNameAPI = "infra-api"

type InventoryClientHandler struct {
	InvClient client.InventoryClient
	wg        *sync.WaitGroup
}

// NewInventoryClientHandler creates a client for the Inventory Service.
func NewInventoryClientHandler(
	ctx context.Context,
	config *common.GlobalConfig,
) (*InventoryClientHandler, error) {
	var wg sync.WaitGroup
	insecureConnection := true
	eventsWatcher := make(chan *client.WatchEvents)

	dialOpts := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(client.TenantContextExtractingInterceptor()),
	}
	clientCfg := client.InventoryClientConfig{
		Name:    invNameAPI,
		Address: config.Inventory.Address,
		SecurityCfg: &client.SecurityConfig{
			Insecure: insecureConnection,
			CaPath:   config.Inventory.CAPath,
			CertPath: config.Inventory.CertPath,
			KeyPath:  config.Inventory.KeyPath,
		},
		Events:                    eventsWatcher,
		EnableRegisterRetry:       false,
		AbortOnUnknownClientError: true,
		ClientKind:                inv_v1.ClientKind_CLIENT_KIND_API,
		ResourceKinds:             []inv_v1.ResourceKind{},
		Wg:                        &wg,
		EnableTracing:             config.Traces.EnableTracing,
		EnableMetrics:             config.Inventory.EnableMetrics,
		DialOptions:               dialOpts,
	}

	invClient, err := client.NewInventoryClient(ctx, clientCfg)
	if err != nil {
		return nil, err
	}

	invHandler := &InventoryClientHandler{
		InvClient: invClient,
		wg:        &wg,
	}
	return invHandler, err
}
