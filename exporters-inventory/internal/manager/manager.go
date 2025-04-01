// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"github.com/onosproject/onos-lib-go/pkg/prom"

	"github.com/open-edge-platform/infra-core/exporters-inventory/internal/common"
	"github.com/open-edge-platform/infra-core/exporters-inventory/internal/exporter"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var log = logging.GetLogger("manager")

type Manager interface {
	Start() error
	Stop() error
}

func NewManager(cfg *common.GlobalConfig, mngrChan, termChan chan bool) (Manager, error) {
	log.Info().Msg("Creating collectors")
	collector, err := exporter.InitCollectorsPrometheus(cfg.ExporterConfig)
	if err != nil {
		log.InfraSec().InfraErr(err).Msgf("error registering collector %s", err)
		return nil, err
	}

	newExporter, err := exporter.NewPrometheusExporter(cfg.ExporterConfig, collector)
	if err != nil {
		return nil, err
	}

	return &manager{
		cfg:      cfg,
		mngrChan: mngrChan,
		termChan: termChan,
		exporter: newExporter,
	}, nil
}

type manager struct {
	cfg      *common.GlobalConfig
	mngrChan chan bool
	termChan chan bool
	exporter prom.Exporter
}

func (m *manager) Start() error {
	log.Info().Msg("Starting manager")
	m.mngrChan <- true
	if err := m.exporter.Run(); err != nil {
		log.InfraSec().InfraErr(err).Msg("exporter error")
		return err
	}
	return nil
}

func (m *manager) Stop() error {
	log.Info().Msg("Stopping manager")
	// TODO: Implement exporter (onos-lib-go) Stop() or support for termChan.
	return nil
}
