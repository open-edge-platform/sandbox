// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package datamodel

import (
	"testing"

	testutils "github.com/open-edge-platform/infra-core/tenant-controller/internal/testing"
)

func TestMain(m *testing.M) {
	testutils.InitTestEnvironment()(m)
}
