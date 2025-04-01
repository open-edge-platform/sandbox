// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server_test

import (
	"net/url"
	"testing"

	"github.com/open-edge-platform/infra-core/api/internal/server"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
)

// TestUtils_ValidateQuery tests the function ValidateQuery
// against possible use cases expected in the REST API.
// It includes test cases that should faild when a query
// parameter is passed in a request but no params were parsed/defined
// in the requested query struct.
func TestUtils_ValidateQuery(t *testing.T) {
	testCases := map[string]struct {
		InValues url.Values
		InQuery  interface{}
		Valid    bool
	}{
		"Test_QueryNil": {
			InValues: map[string][]string{},
			InQuery:  nil,
			Valid:    true,
		},
		"Test_QueryNil_Invalid": {
			InValues: map[string][]string{
				"inject_query": {},
			},
			InQuery: nil,
			Valid:   false,
		},
		"Test_Empty": {
			InValues: map[string][]string{},
			InQuery:  struct{}{},
			Valid:    true,
		},
		"Test_Empty_Invalid": {
			InValues: map[string][]string{
				"inject_query": {},
			},
			InQuery: struct{}{},
			Valid:   false,
		},
		"Test_GetComputeHostsParams": {
			InValues: map[string][]string{
				"offset":   {},
				"pageSize": {},
			},
			InQuery: api.GetComputeHostsParams{
				Offset:   &pgOffset,
				PageSize: &pgSize,
			},
			Valid: true,
		},
		"Test_GetComputeHostsParams_Invalid": {
			InValues: map[string][]string{
				"offset":       {},
				"pageSize":     {},
				"inject_query": {},
			},
			InQuery: api.GetComputeHostsParams{
				Offset:   &pgOffset,
				PageSize: &pgSize,
			},
			Valid: false,
		},
		"Test_GetComputeHostsParams_Empty": {
			InValues: map[string][]string{},
			InQuery:  api.GetComputeHostsParams{},
			Valid:    true,
		},
		"Test_GetComputeHostsParams_Empty_Invalid": {
			InValues: map[string][]string{
				"offset":   {},
				"pageSize": {},
			},
			InQuery: api.GetComputeHostsParams{},
			Valid:   false,
		},
		"Test_Query_Invalid": {
			InValues: map[string][]string{
				"offset":   {},
				"pageSize": {},
			},
			InQuery: 1,
			Valid:   false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := server.ValidateQuery(tc.InValues, tc.InQuery)

			if tc.Valid && err != nil {
				t.Errorf("invalid error, query and values should be validated")
			}

			if !tc.Valid && err == nil {
				t.Errorf("error should be provided for invalid request")
			}
		})
	}
}
