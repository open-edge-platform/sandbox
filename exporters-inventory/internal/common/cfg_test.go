// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package common_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-edge-platform/infra-core/exporters-inventory/internal/common"
)

func TestConfig(t *testing.T) {
	cfg, err := common.Config()
	assert.Equal(t, err, nil)
	assert.NotEqual(t, cfg, nil)
}
