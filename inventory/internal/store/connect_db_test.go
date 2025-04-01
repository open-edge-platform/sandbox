// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/store"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

func TestConnectDb(t *testing.T) {
	dbURL := util.GetDBURL(util.LookupDBTestEnv())
	// Assumption is that migration are already run, so this function will pass correctly, otherwise it will fatal
	client := store.ConnectEntDB(dbURL, dbURL)
	require.NotNil(t, client)
}
