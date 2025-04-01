// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"flag"
)

var (
	exporterAddress         = ":9101"
	exporterPath            = "/metrics"
	defaultInventoryAddress = "infra-inventory:50051"
)

type LogLevel struct {
	Tracing  bool
	TraceURL string
}

type OAM struct {
	Address string
}

type CollectorsConfig struct {
	Name                      CollectorName
	Address                   string
	CAPath, CertPath, KeyPath string
	EnableTracing             bool
}

// ExporterConfig establishes the fields needed for the instantiation of
// an exporter.
// Address and Path define the exporter endpoint from where KPIs can
// be pulled.
type ExporterConfig struct {
	Path       string
	Address    string
	Collectors []CollectorsConfig
}

type GlobalConfig struct {
	LogLevel       LogLevel
	ExporterConfig ExporterConfig
	OAMServer      OAM
}

func DefaultConfig() *GlobalConfig {
	return &GlobalConfig{
		LogLevel: LogLevel{
			Tracing:  false,
			TraceURL: "",
		},
		ExporterConfig: ExporterConfig{
			Path:    exporterPath,
			Address: exporterAddress,
			Collectors: []CollectorsConfig{
				{
					Name:    InventoryCollector,
					Address: defaultInventoryAddress,
				},
			},
		},
		OAMServer: OAM{
			Address: "",
		},
	}
}

func Config() (*GlobalConfig, error) {
	defaultCfg := DefaultConfig()

	expAddress := flag.String(
		"exporterAddress", exporterAddress,
		"Exporter address",
	)
	expPath := flag.String(
		"exporterPath", exporterPath,
		"Exporter Path",
	)
	inventoryAddress := flag.String(
		"inventoryAddress", defaultInventoryAddress,
		"Inventory address",
	)
	enableTracing := flag.Bool(
		"enableTracing", defaultCfg.LogLevel.Tracing,
		"Flag to enable tracing",
	)
	traceURL := flag.String(
		"traceURL", defaultCfg.LogLevel.TraceURL,
		"Tracing URL for OTLP protocol",
	)
	oamservaddr := flag.String(
		"oamservaddr", "",
		"The oam server address to serve on",
	)
	flag.Parse()

	return &GlobalConfig{
		LogLevel: LogLevel{
			Tracing:  *enableTracing,
			TraceURL: *traceURL,
		},
		ExporterConfig: ExporterConfig{
			Path:    *expPath,
			Address: *expAddress,
			Collectors: []CollectorsConfig{
				{
					Name:          InventoryCollector,
					Address:       *inventoryAddress,
					EnableTracing: *enableTracing,
				},
			},
		},
		OAMServer: OAM{
			Address: *oamservaddr,
		},
	}, nil
}
