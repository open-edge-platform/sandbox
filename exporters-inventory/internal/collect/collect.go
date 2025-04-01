// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package collect

import (
	"sync"

	"google.golang.org/grpc/codes"

	"github.com/open-edge-platform/infra-core/exporters-inventory/internal/common"
	"github.com/open-edge-platform/infra-core/exporters-inventory/internal/kpis"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var log = logging.GetLogger("collect")

// Collector defines an interface for Collectors to retrieve
// a list of kpis.KPI via the Collect method.
type Collector interface {
	Collect() ([]kpis.KPI, error)
	Stop()
}

// CreateCollector instantiates a new collector based on the
// name of the collector cfg specified.
func CreateCollector(cfg common.CollectorsConfig) (Collector, error) {
	log.Info().Msgf("Creating collector %s", cfg.Name)
	switch cfg.Name {
	case common.InventoryCollector:
		return NewInventoryCollector(cfg)
	default:
		return nil, inv_errors.Errorfc(codes.Unimplemented, "no collector found with name %s", cfg.Name)
	}
}

// KPIs retrieves the list of kpis.KPI from each Collector.
// It handles each collector error locally, logging the error.
// In any case, kpis.KPI list is returned, e.g., if one collector
// presents error or even if all collectors present errors.
// The channel outKPIs is passed as input, and must be read
// to unblock the execution of KPIS.
// After all KPIs are collected, outKPIs channel is closed.
func KPIs(outKPIs chan kpis.KPI, collectors []Collector) {
	wg := sync.WaitGroup{}
	wg.Add(len(collectors))
	for _, col := range collectors {
		go func(c Collector) {
			defer wg.Done()
			colKPIs, err := c.Collect()
			if err != nil {
				log.InfraErr(err).Msgf("collector KPIs Collect error: %s", err)
				return
			}

			log.Debug().Msgf("got collector kpis: %v", colKPIs)
			for _, colKPI := range colKPIs {
				outKPIs <- colKPI
			}
		}(col)
	}
	wg.Wait()
	close(outKPIs)

	log.Debug().Msgf("KPIs collected")
}
