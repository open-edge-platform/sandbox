// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"github.com/onosproject/onos-lib-go/pkg/prom"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/open-edge-platform/infra-core/exporters-inventory/internal/collect"
	"github.com/open-edge-platform/infra-core/exporters-inventory/internal/common"
	"github.com/open-edge-platform/infra-core/exporters-inventory/internal/kpis"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var log = logging.GetLogger("exporter")

// CollectorsPrometheus defines a prometheus collector
// for all collectors.
type CollectorsPrometheus struct {
	collectors []collect.Collector
}

// Retrieve implements the method needed for a Collector interface
// in a prometheus exporter. It retrieves all the kpis from
// CollectorsPrometheus and pass them to the ch channel using the
// prometheus.Metric format.
// The function collect.KPIs performs the collection of each collector
// list of KPIs, and aggregates them in miKPIs var.
func (c *CollectorsPrometheus) Retrieve(ch chan<- prometheus.Metric) error {
	outKPIs := make(chan kpis.KPI)
	go func() {
		collect.KPIs(outKPIs, c.collectors)
	}()

	for kpi := range outKPIs {
		promMetrics, err := kpi.PrometheusFormat()
		if err != nil {
			log.InfraSec().InfraErr(err).Msgf("kpi prometheus format error %s", err)
			return err
		}
		for _, m := range promMetrics {
			ch <- m
		}
	}
	log.Debug().Msg("Finished Retrieve")
	return nil
}

func (c *CollectorsPrometheus) SetCollectors(collectors []collect.Collector) {
	c.collectors = collectors
}

func (c *CollectorsPrometheus) Stop() {
	for _, col := range c.collectors {
		col.Stop()
	}
}

// Defines the set of collector used to extract KPIs for
// the prometheus exporter. Each collector implements the
// prom.Collector interface behavior via the method Collect.
func InitCollectorsPrometheus(config common.ExporterConfig) (prom.Collector, error) {
	collectors := []collect.Collector{}

	for _, collectorCfg := range config.Collectors {
		collector, err := collect.CreateCollector(collectorCfg)
		if err != nil {
			log.InfraSec().InfraErr(err).Msgf("%s not added to collectors", collectorCfg.Name)
			return nil, err
		}
		collectors = append(collectors, collector)
	}
	return &CollectorsPrometheus{
		collectors: collectors,
	}, nil
}

// PrometheusExporter uses Config to create an instance of a
// Prometheus exporter, registering all its collectors, which must
// implement the interface method Retrieve.
func NewPrometheusExporter(config common.ExporterConfig, collector prom.Collector) (prom.Exporter, error) {
	log.Info().Msg("Creating prometheus exporter")
	exporter := prom.NewExporter(config.Path, config.Address)
	err := exporter.RegisterCollector("infra", collector)
	if err != nil {
		log.InfraSec().InfraErr(err).Msg("error registering exporter collector")
		return nil, inv_errors.Wrap(err)
	}

	return exporter, nil
}
