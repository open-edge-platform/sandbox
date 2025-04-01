// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package dispatcher_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/api/internal/common"
	"github.com/open-edge-platform/infra-core/api/internal/dispatcher"
	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/test/utils"
	schedule_cache "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client/cache/schedule"
)

var invAddress = "bufconn"

func TestNewDispatcher(t *testing.T) {
	cfg := common.DefaultConfig()
	cfg.Inventory.Address = invAddress
	assert.NotEqual(t, cfg, nil)
	dispChan := make(chan bool, 1)
	termChan := make(chan bool, 1)
	wg := sync.WaitGroup{}
	disp := dispatcher.NewDispatcher(cfg, dispChan, termChan, &wg)
	assert.NotEqual(t, disp, nil)
	ctx := context.Background()

	j := types.NewJob(ctx, types.List, types.Site, "test", "test")
	disp.JobQueue <- *j

	wg.Add(1)
	go disp.Dispatch()
	close(termChan)
}

func TestNewDispatcherError(t *testing.T) {
	cfg := common.DefaultConfig()
	// this test will fail undefinetely; we disable the retries
	cfg.Inventory.Address = "localhost:50051"
	cfg.Inventory.Retry = false
	assert.NotEqual(t, cfg, nil)
	dispChan := make(chan bool, 1)
	termChan := make(chan bool, 1)
	wg := sync.WaitGroup{}
	disp := dispatcher.NewDispatcher(cfg, dispChan, termChan, &wg)
	assert.NotEqual(t, disp, nil)
	err := disp.Run()
	assert.Error(t, err)
}

func TestNewDispatcherErrorCache(t *testing.T) {
	cfg := common.DefaultConfig()
	// this test will fail undefinetely; we disable the retries
	cfg.Inventory.Address = "localhost:50051"
	// To trigger error from the schedule cache
	cfg.Worker.MaxWorkers = 0
	cfg.Inventory.Retry = false
	assert.NotEqual(t, cfg, nil)
	dispChan := make(chan bool, 1)
	termChan := make(chan bool, 1)
	wg := sync.WaitGroup{}
	disp := dispatcher.NewDispatcher(cfg, dispChan, termChan, &wg)
	assert.NotEqual(t, disp, nil)
	err := disp.Run()
	assert.Error(t, err)
}

func TestNewDispatcherOk(t *testing.T) {
	cfg := common.DefaultConfig()
	cfg.Inventory.Address = invAddress
	cfg.Inventory.Retry = false
	assert.NotEqual(t, cfg, nil)
	dispChan := make(chan bool, 1)
	termChan := make(chan bool, 1)
	wg := sync.WaitGroup{}
	disp := dispatcher.NewDispatcher(cfg, dispChan, termChan, &wg)
	assert.NotEqual(t, disp, nil)

	mockInvClient := utils.NewTenantAwareMockInventoryServiceClient(utils.MockResponses{})
	invClientHandler := &clients.InventoryClientHandler{
		InvClient: mockInvClient.GetInventoryClient(),
	}
	for i := 0; i < cfg.Worker.MaxWorkers; i++ {
		disp.InvClients = append(disp.InvClients, invClientHandler)
	}
	var err error
	scheduleCache := schedule_cache.NewScheduleCacheClient(mockInvClient)
	disp.HScheduleCache, err = schedule_cache.NewHScheduleCacheClient(scheduleCache)
	require.NoError(t, err)

	err = disp.Run()
	assert.NoError(t, err)
	close(termChan)
	wg.Wait()
}
