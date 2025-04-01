// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package inventory_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

func TestMain(m *testing.M) {
	// Currently unused
	flag.String(
		"policyBundle",
		"/rego/policy_bundle.tar.gz",
		"Path of policy rego file",
	)
	flag.Parse()
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	projectRoot := filepath.Dir(filepath.Dir(wd))

	policyPath := projectRoot + "/out"
	certPath := projectRoot + "/cert/certificates"
	migrationsDir := projectRoot + "/out"

	inv_testing.StartTestingEnvironment(policyPath, certPath, migrationsDir)
	run := m.Run() // run all tests
	inv_testing.StopTestingEnvironment()

	os.Exit(run)
}
