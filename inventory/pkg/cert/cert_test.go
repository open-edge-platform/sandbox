// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cert_test

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/cert"
)

func TestMain(m *testing.M) {
	// Only needed to suppress the error
	flag.String(
		"policyBundle",
		"/rego/policy_bundle.tar.gz",
		"Path of policy rego file",
	)
	flag.Parse()

	run := m.Run() // run all tests
	os.Exit(run)
}

func Test_InvalidHandleCertPathsAndPools(t *testing.T) {
	_, err := cert.HandleCertPaths("", "", "", false)
	assert.Error(t, err)

	_, err = cert.GetCertPool("")
	assert.Error(t, err)
}
