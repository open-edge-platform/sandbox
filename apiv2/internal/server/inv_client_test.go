// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server_test

import (
	m_client "github.com/open-edge-platform/infra-core/apiv2/v2/mocks/m_client"
)

func newMockedInventoryTestClient() *m_client.MockInventoryClient {
	return &m_client.MockInventoryClient{}
}
