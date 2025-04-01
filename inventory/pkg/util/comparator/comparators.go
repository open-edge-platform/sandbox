// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package comparator

import (
	"google.golang.org/protobuf/proto"

	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

var log = logging.GetLogger("comparator")

type resourceIDCarrier interface {
	GetResourceId() string
}

// ResourceIDComparator compares inventoryv1.Resource's. It relies on alphanumerical ascending order.
// In theory comparator works for any resources.
// It will accept resources which have no ResourceID - in such a case, resources w/o ResourceID will be placed at the end.
func ResourceIDAscComparator(toBeSorted []*inv_v1.Resource) (x []*inv_v1.Resource, less func(i int, j int) bool) {
	return toBeSorted, func(i, j int) bool {
		iv, err := util.UnwrapResource[proto.Message](toBeSorted[i])
		if err != nil {
			log.Warn().Msgf("cannot unwrap resource(%v)", toBeSorted[i])
			return false
		}
		left, ok := iv.(resourceIDCarrier)
		if !ok {
			log.Warn().Msgf("resource(%v) have no resourceID", toBeSorted[i])
			return false
		}
		jv, err := util.UnwrapResource[proto.Message](toBeSorted[j])
		if err != nil {
			log.Warn().Msgf("cannot unwrap resource(%v)", toBeSorted[j])
			return true
		}
		right, ok := jv.(resourceIDCarrier)
		if !ok {
			log.Warn().Msgf("resource(%v) have no resourceID", toBeSorted[j])
			return true
		}
		return left.GetResourceId() < right.GetResourceId()
	}
}
