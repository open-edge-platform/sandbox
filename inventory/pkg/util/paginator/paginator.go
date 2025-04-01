// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package paginator

import "github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"

var log = logging.GetLogger("Paginator")

func NewPaginator[T comparable](offset, limit int) Paginator[T] {
	return Paginator[T]{
		offset: offset,
		limit:  limit,
	}
}

type Paginator[T comparable] struct {
	offset, limit int
}

// utility function to apply the pagination on a slice of resources.
func (p Paginator[T]) Apply(resources []T) (res []T, hasNext bool, totLen int) {
	if len(resources) == 0 {
		log.Debug().Msgf("No resources found")
		return resources[:0], false, 0
	}
	// Apply offset and limit
	totalLen := len(resources)
	switch {
	case p.limit != 0 && p.offset+p.limit < len(resources):
		return resources[p.offset : p.offset+p.limit], true, totalLen
	case p.offset < len(resources):
		return resources[p.offset:], false, totalLen
	default:
		log.Debug().Msgf("No resources found")
		return resources[:0], false, totalLen
	}
}
