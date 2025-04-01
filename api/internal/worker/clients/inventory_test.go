// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package clients_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/api/internal/common"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	inventory "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// internal/worker/clients
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(wd)))

	policyPath := projectRoot + "/out"
	migrationsDir := projectRoot + "/out"

	inv_testing.StartTestingEnvironment(policyPath, "", migrationsDir)
	run := m.Run() // run all tests
	inv_testing.StopTestingEnvironment()

	os.Exit(run)
}

func TestNewInventoryClientHandler(t *testing.T) {
	assertInstance := assert.New(t)
	cfg := common.DefaultConfig()
	assertInstance.NotEqual(cfg, nil)
	cfg.Inventory.Address = "localhost:50051"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	inv, err := clients.NewInventoryClientHandler(ctx, cfg)
	assertInstance.Equal(inv, (*clients.InventoryClientHandler)(nil))
	assertInstance.NotEqual(err, nil)
	req := inventory.GetResourceRequest{}
	req.Reset()
}

func TestClientConnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	clientHandler := &clients.InventoryClientHandler{
		InvClient: inv_testing.TestClients[inv_testing.APIClient],
	}
	_, err := clientHandler.InvClient.Get(ctx, "host-12345")
	require.Error(t, err, "Should return not found")
	require.True(t, inv_errors.IsNotFound(err))
}
