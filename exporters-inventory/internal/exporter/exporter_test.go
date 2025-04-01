// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package exporter_test

import (
	"context"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/exporters-inventory/internal/collect"
	"github.com/open-edge-platform/infra-core/exporters-inventory/internal/common"
	"github.com/open-edge-platform/infra-core/exporters-inventory/internal/exporter"
	sched_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/schedule/v1"
	schedule_cache "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client/cache/schedule"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

const metricsBufferSize = 50

var log = logging.GetLogger("test-collect")

func TestExporter_New(t *testing.T) {
	cfg := common.GlobalConfig{
		LogLevel: common.LogLevel{
			Tracing:  false,
			TraceURL: "",
		},
		ExporterConfig: common.ExporterConfig{
			Path:    "/metrics",
			Address: ":19101",
			Collectors: []common.CollectorsConfig{
				{
					Name:    common.InventoryCollector,
					Address: "bufconn",
				},
			},
		},
		OAMServer: common.OAM{
			Address: "",
		},
	}

	collector, err := exporter.InitCollectorsPrometheus(cfg.ExporterConfig)
	assert.Error(t, err)

	exp, err := exporter.NewPrometheusExporter(cfg.ExporterConfig, collector)
	assert.NoError(t, err)
	assert.NotNil(t, exp)
}

func TestExporter_Retrieve(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())

	invClient := inv_testing.TestClients[inv_testing.RMClient].GetTenantAwareInventoryClient()
	invEventsWatcher := inv_testing.TestClientsEvents[inv_testing.RMClient]

	chanTerm := make(chan bool)
	var wg sync.WaitGroup

	scheduleCache := schedule_cache.NewScheduleCacheClient(invClient)
	hScheduleCache, err := schedule_cache.NewHScheduleCacheClient(scheduleCache)
	require.NoError(t, err)
	invCollectorCache := collect.NewInvCollectorCache(invClient, chanTerm, &wg, invEventsWatcher)

	invCollector := &collect.InventoryCollector{
		Name:            common.InventoryCollector,
		Address:         "",
		Cancel:          cancel,
		CollectorClient: invCollectorCache,
		HScheduleCache:  hScheduleCache,
	}

	collectors := []collect.Collector{invCollector}

	exp := exporter.CollectorsPrometheus{}
	exp.SetCollectors(collectors)

	host := inv_testing.CreateHost(t, nil, nil)
	inv_testing.CreateSingleSchedule(t, host, nil, sched_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE)

	ch := make(chan<- prometheus.Metric, metricsBufferSize)
	err = exp.Retrieve(ch)
	assert.NoError(t, err)

	log.Info().Msg("stopping exporter")

	exp.Stop()
}
