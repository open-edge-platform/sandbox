// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package common_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-edge-platform/infra-core/apiv2/v2/internal/common"
)

func TestConfig(t *testing.T) {
	cfg, err := common.Config()
	assert.NoError(t, err)
	assert.NotEqual(t, cfg, nil)
}
