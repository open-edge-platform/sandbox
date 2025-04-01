// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/flags"
	"github.com/open-edge-platform/infra-core/tenant-controller/internal/invclient"
)

func TestResMgr_InvClient(t *testing.T) {
	rc := make(chan bool, 1)
	tc := make(chan bool, 1)
	wg := sync.WaitGroup{}
	sc := GetSecurityConfig()

	_, err := invclient.NewInventoryClientWithOptions(
		rc,
		tc,
		&wg,
		sc,
		invclient.WithInventoryAddress("localhost:50051"),
		invclient.WithEnableTracing(false),
	)
	require.Error(t, err)

	wg.Wait()
}

func TestResMgr_Startup(t *testing.T) {
	StartupSummary()

	traceFunc := SetupTracing("http://trace")

	SetupOamServerAndSetReady(false, "localhost:2379")

	err := traceFunc(context.Background())

	require.Nil(t, err)
}

func Test_getInitResourceProviders(t *testing.T) {
	t.Run("load existing files", func(t *testing.T) {
		flags.FlagDisableCredentialsManagement = &([]bool{true}[0])
		initResourcesDefinitionPath = &([]string{"../configuration/default/resources.json"}[0])
		lenovoResourcesDefinitionPath = &([]string{"../configuration/default/resources-lenovo.json"}[0])
		res, err := getInitResourceProviders()
		require.NoError(t, err)
		require.Len(t, res, 2)
	})

	t.Run("try to load not existing files", func(t *testing.T) {
		flags.FlagDisableCredentialsManagement = &([]bool{true}[0])
		initResourcesDefinitionPath = &([]string{"foo"}[0])
		lenovoResourcesDefinitionPath = &([]string{"bar"}[0])
		res, err := getInitResourceProviders()
		require.Error(t, err)
		require.Empty(t, res)
	})
}
