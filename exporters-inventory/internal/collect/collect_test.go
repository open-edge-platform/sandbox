// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package collect_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/exporters-inventory/internal/collect"
	"github.com/open-edge-platform/infra-core/exporters-inventory/internal/common"
	"github.com/open-edge-platform/infra-core/exporters-inventory/internal/kpis"
	sched_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/schedule/v1"
	schedule_cache "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client/cache/schedule"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

const (
	tenant1 = "11111111-1111-1111-1111-111111111111"
	tenant2 = "22222222-2222-2222-2222-222222222222"
)

func TestCollector_Collect(t *testing.T) {
	dao := inv_testing.NewInvResourceDAOOrFail(t)
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

	hostT1 := dao.CreateHost(t, tenant1)
	dao.CreateSingleSchedule(t, tenant1, inv_testing.SSRTargetHost(hostT1),
		inv_testing.SSRStatus(sched_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE))
	hostT2 := dao.CreateHost(t, tenant2)
	dao.CreateSingleSchedule(t, tenant2, inv_testing.SSRTargetHost(hostT2),
		inv_testing.SSRStatus(sched_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE))

	colkpis, err := invCollector.Collect()
	assert.NoError(t, err)
	assert.NotNil(t, colkpis)

	invCollector.Stop()
}

// FIXME what is really doing this test?
func TestExporter_KPIs(t *testing.T) {
	dao := inv_testing.NewInvResourceDAOOrFail(t)
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

	host1T1 := dao.CreateHost(t, tenant1)
	dao.CreateSingleSchedule(t, tenant1, inv_testing.SSRTargetHost(host1T1),
		inv_testing.SSRStatus(sched_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE))
	hostT2 := dao.CreateHost(t, tenant2)
	dao.CreateSingleSchedule(t, tenant2, inv_testing.SSRTargetHost(hostT2),
		inv_testing.SSRStatus(sched_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE))

	site := dao.CreateSite(t, tenant1)
	dao.CreateHost(t, tenant1, inv_testing.HostSite(site))
	dao.CreateRepeatedSchedule(t, tenant1, inv_testing.RSRTargetSite(site),
		inv_testing.RSRStatus(sched_v1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE))

	outKPIs := make(chan kpis.KPI)

	colkpis := []kpis.KPI{}
	go func(colkpis []kpis.KPI) {
		for newKPI := range outKPIs {
			colkpis = append(colkpis, newKPI)
		}
	}(colkpis)

	collect.KPIs(outKPIs, collectors)
	assert.NotNil(t, outKPIs)
	assert.NotNil(t, colkpis)

	time.Sleep(10 * time.Second)
	outKPIs = make(chan kpis.KPI)
	colkpis = []kpis.KPI{}
	go func(colkpis []kpis.KPI) {
		for newKPI := range outKPIs {
			colkpis = append(colkpis, newKPI)
		}
	}(colkpis)
	collect.KPIs(outKPIs, collectors)
	assert.NotNil(t, outKPIs)
	assert.NotNil(t, colkpis)

	invCollector.Stop()
}
