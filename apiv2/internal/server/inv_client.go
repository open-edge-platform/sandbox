// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"sync"

	"google.golang.org/grpc"

	"github.com/open-edge-platform/infra-core/apiv2/v2/internal/common"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	schedule_cache "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client/cache/schedule"
)

var invNameAPI = "infra-api"

// NewInventoryClient creates a client for the Inventory Service.
func NewInventoryClient(
	ctx context.Context,
	wg *sync.WaitGroup,
	config *common.GlobalConfig,
) (client.InventoryClient, error) {
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
		Wg:                        wg,
		EnableTracing:             config.Traces.EnableTracing,
		EnableMetrics:             config.Inventory.EnableMetrics,
		DialOptions:               dialOpts,
	}

	InvClient, err := client.NewInventoryClient(ctx, clientCfg)
	if err != nil {
		return nil, err
	}

	return InvClient, err
}

func NewInventoryHCacheClient(
	ctx context.Context,
	config *common.GlobalConfig,
) (*schedule_cache.HScheduleCacheClient, error) {
	scheduleCache, err := schedule_cache.NewScheduleCacheClientWithOptions(
		ctx,
		schedule_cache.WithInventoryAddress(config.Inventory.Address),
		schedule_cache.WithEnableTracing(config.Traces.EnableTracing),
	)
	if err != nil {
		zlog.InfraErr(err).Msg("Failed to create new inventory client for schedule cache")
		return nil, err
	}
	hScheduleCache, err := schedule_cache.NewHScheduleCacheClient(scheduleCache)
	if err != nil {
		zlog.InfraErr(err).Msg("Failed to create new inventory client for h schedule cache")
		return nil, err
	}
	return hScheduleCache, nil
}
