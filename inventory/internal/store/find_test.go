// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

func Test_FindList_Errors(t *testing.T) {
	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	t.Run("Filter_MissingResource1", func(t *testing.T) {
		_, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx, &inv_v1.ResourceFilter{})
		require.Error(t, err)
		sts, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, sts.Code())
	})

	t.Run("List_MissingResource1", func(t *testing.T) {
		_, err := inv_testing.TestClients[inv_testing.APIClient].List(ctx, &inv_v1.ResourceFilter{})
		require.Error(t, err)
		sts, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, sts.Code())
	})

	t.Run("Filter_MissingResource2", func(t *testing.T) {
		_, err := inv_testing.TestClients[inv_testing.APIClient].Find(
			ctx,
			&inv_v1.ResourceFilter{
				Resource: &inv_v1.Resource{},
			},
		)
		require.Error(t, err)
		sts, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, sts.Code())
	})

	t.Run("List_MissingResource2", func(t *testing.T) {
		_, err := inv_testing.TestClients[inv_testing.APIClient].List(
			ctx,
			&inv_v1.ResourceFilter{
				Resource: &inv_v1.Resource{},
			},
		)
		require.Error(t, err)
		sts, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, sts.Code())
	})
}
