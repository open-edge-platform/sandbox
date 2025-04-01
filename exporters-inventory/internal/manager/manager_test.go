// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package manager_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-edge-platform/infra-core/exporters-inventory/internal/common"
	"github.com/open-edge-platform/infra-core/exporters-inventory/internal/manager"
)

func TestManager_New(t *testing.T) {
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

	termChan := make(chan bool)
	readyChan := make(chan bool)
	mngr, err := manager.NewManager(&cfg, readyChan, termChan)
	assert.Error(t, err)
	assert.Nil(t, mngr)
}
