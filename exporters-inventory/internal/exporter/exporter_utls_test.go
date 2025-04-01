// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package exporter_test

import (
	"os"
	"path/filepath"
	"testing"

	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

// Starts all Inventory and Host Manager requirements to test exporter API.
func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	projectRoot := filepath.Dir(filepath.Dir(wd))

	policyPath := projectRoot + "/out"
	migrationsDir := projectRoot + "/out"

	inv_testing.StartTestingEnvironment(policyPath, "", migrationsDir)
	run := m.Run() // run all tests
	inv_testing.StopTestingEnvironment()
	os.Exit(run)
}
