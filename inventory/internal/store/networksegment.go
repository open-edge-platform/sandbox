// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/networksegment"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/siteresource"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	network_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/network/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

var networkSegmentCreationValidators = []resourceValidator[*network_v1.NetworkSegment]{
	protoValidator[*network_v1.NetworkSegment],
	doNotAcceptResourceID[*network_v1.NetworkSegment],
}

func (is *InvStore) CreateNetworkSegment(ctx context.Context, in *network_v1.NetworkSegment) (*inv_v1.Resource, error) {
	if err := validate(in, networkSegmentCreationValidators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, networkSegmentCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("NetworkSegment Created: %s, %s", res.GetNetworkSegment().GetResourceId(), res)
	return res, nil
}

func networkSegmentCreator(in *network_v1.NetworkSegment) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_NETWORKSEGMENT)
		zlog.Debug().Msgf("CreateNetworkSegment: %s", id)

		newEntity := tx.NetworkSegment.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, EmptyEnumStateMap, nil); err != nil {
			return nil, err
		}

		// Look up the site ID for this network.
		err := setEdgeSiteIDForMut(ctx, tx.Client(), mut, in.GetSite())
		if err != nil {
			return nil, err
		}

		err = mut.SetField(networksegment.FieldResourceID, id)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		resSave, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		err = checkVlanIDUniqueness(ctx, tx.Client(), resSave.ResourceID)
		if err != nil {
			return nil, err
		}

		res, err := getNetworkSegmentQuery(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		return util.WrapResource(entNetworkSegmentToProtoNetworkSegmentResource(res))
	}
}

// One Site can have multiple Network Segments attached,
// we have to make sure that current Vlan ID is unique within a Site.
// Also, if we have a Vlan ID = 0 that must be the unique Vlan for the Site.
//
//nolint:cyclop,nolintlint // high cyclomatic complexity due to calling predicate-related functions (include switch statements).
func checkVlanIDUniqueness(ctx context.Context, client *ent.Client, netsegID string) error {
	// Querying the Network Segment, which is being updated, first
	netseg, err := client.NetworkSegment.Query().
		Where(networksegment.ResourceID(netsegID)).
		WithSite(). // eager-loading Site
		Only(ctx)
	if err != nil {
		zlog.InfraErr(err).Msgf("Couldn't query Network Segment (%s)", netsegID)
		return errors.Wrap(err)
	}

	// this is safe due to constraints on the Site being required (always not nil)
	siteID := netseg.Edges.Site.ResourceID

	// extracting all network segments attached to given Site different from the segment being updated
	netsegs, err := client.NetworkSegment.Query().
		Where(networksegment.HasSiteWith(siteresource.ResourceID(siteID))).
		Where(networksegment.Not(networksegment.ResourceID(netsegID))).
		All(ctx)
	if err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("")
		return errors.Wrap(err)
	}

	if len(netsegs) == 0 {
		// there are no other Vlans except this
		zlog.Debug().Msgf("Vlan ID uniqueness validation for Vlan ID succeeded: vlan_id=%d, site=%s",
			netseg.VlanID, siteID)
		return nil
	}
	// There are multiple Network Segments within a Site

	if netseg.VlanID == 0 {
		// Treating the case when VlanID is 0. It has to be the only VlanID within a Site, but
		// there are other VLANs which cannot coexist with default VLAN within a Site
		zlog.InfraSec().InfraError(
			"Vlan ID (%d) uniqueness validation has failed - not a single VlanID within a Site (%s), got %d segments",
			netseg.VlanID, siteID, len(netsegs))
		return errors.Errorfc(codes.InvalidArgument,
			"Vlan ID (%d) uniqueness validation has failed - not a single VlanID within a Site (%s), got %d segments",
			netseg.VlanID, siteID, len(netsegs))
	}

	// Treating the case with non-zero VlanID (i.e., updated Network Segment has non-zero Vlan ID).
	// Additionally, we should make sure that within a given Site, where updated Network Segment is,
	// do not exist any other VlanID equal to 0 (i.e., in this case, changed is not permitted) or
	// a VlanID that is a duplicate of this one.
	zlog.Debug().Msgf("Iterating over list of NetworkSegments (%d) attached to the Site (%s)",
		len(netsegs), siteID)
	for _, v := range netsegs {
		if v.VlanID == 0 {
			zlog.InfraSec().InfraError("VlanID=0 already exists within a Site (%s), can't set another VlanID (%d)",
				siteID, netseg.VlanID)
			return errors.Errorfc(codes.InvalidArgument,
				"VlanID=0 already exists within a Site (%s), can't create another VlanID (%d)",
				siteID, netseg.VlanID)
		}
		if v.VlanID == netseg.VlanID {
			zlog.InfraSec().InfraError("Found duplicated Vlan ID (%d) within a Site (%s), NetworkSegments are %s-%s",
				netseg.VlanID, siteID, netseg.ResourceID, v.ResourceID)
			return errors.Errorfc(codes.InvalidArgument, "VlanID (%d) is not unique within Site (%s)",
				netseg.VlanID, siteID)
		}
	}

	zlog.Debug().Msgf("Vlan ID uniqueness validation for Vlan ID (%d) at Site (%s) succeeded", netseg.VlanID, siteID)
	return nil
}

func (is *InvStore) GetNetworkSegment(ctx context.Context, id string) (*inv_v1.Resource, error) {
	res, err := ExecuteInRoTxAndReturnSingle[ent.NetworkSegment](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.NetworkSegment, error) {
			return getNetworkSegmentQuery(ctx, tx, id)
		})
	if err != nil {
		return nil, err
	}

	apiResource := entNetworkSegmentToProtoNetworkSegmentResource(res)

	zlog.Debug().Msgf("Retrieved following NetworkSegment:\n%v", apiResource)

	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_NetworkSegment{NetworkSegment: apiResource}}, nil
}

func getNetworkSegmentQuery(ctx context.Context, tx *ent.Tx, resourceID string) (*ent.NetworkSegment, error) {
	entity, err := tx.NetworkSegment.Query().
		Where(networksegment.ResourceID(resourceID)).
		WithSite().
		Only(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return entity, nil
}

func (is *InvStore) UpdateNetworkSegment(
	ctx context.Context,
	id string,
	in *network_v1.NetworkSegment,
	fieldmask *fieldmaskpb.FieldMask,
) (*inv_v1.Resource, error) {
	if err := validator.ValidateMessage(in); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	zlog.Debug().Msgf("UpdateNetworkSegment (%s): %v, fm: %v", id, in, fieldmask)

	return ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
			entity, err := tx.NetworkSegment.Query().
				Select(networksegment.FieldID).WithSite().
				Where(networksegment.ResourceID(id)).
				Only(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			updateBuilder := tx.NetworkSegment.UpdateOneID(entity.ID)
			mut := updateBuilder.Mutation()

			// Look up the (new) referenced edges for this network.
			mut.ResetSite()
			err = setEdgeSiteIDForMut(ctx, tx.Client(), mut, in.GetSite())
			if err != nil {
				return nil, err
			}

			err = buildEntMutate(in, mut, EmptyEnumStateMap, fieldmask.GetPaths())
			if err != nil {
				return nil, err
			}

			resSave, err := updateBuilder.Save(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			err = checkVlanIDUniqueness(ctx, tx.Client(), resSave.ResourceID)
			if err != nil {
				return nil, err
			}

			res, err := getNetworkSegmentQuery(ctx, tx, id)
			if err != nil {
				return nil, err
			}
			toBeReturned, err := util.WrapResource(entNetworkSegmentToProtoNetworkSegmentResource(res))

			return toBeReturned, err
		},
	)
}

func (is *InvStore) DeleteNetworkSegment(ctx context.Context, id string) (*inv_v1.Resource, error) {
	// this is a "Hard Delete" as Network Segments don't have state
	zlog.Debug().Msgf("DeleteNetworkSegment Hard Delete: %s", id)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, deleteNetworkSegment(id))

	return res, err
}

func deleteNetworkSegment(id string) func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		entity, err := tx.NetworkSegment.Query().
			Where(networksegment.ResourceID(id)).
			Only(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		err = tx.NetworkSegment.DeleteOneID(entity.ID).Exec(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		return util.WrapResource(entNetworkSegmentToProtoNetworkSegmentResource(entity))
	}
}

func (is *InvStore) DeleteNetworkSegments(
	ctx context.Context, tenantID string, _ bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]
	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			collection, err := tx.NetworkSegment.Query().Where(networksegment.TenantID(tenantID)).All(ctx)
			if err != nil {
				return err
			}
			if _, err := tx.NetworkSegment.Delete().Where(networksegment.TenantID(tenantID)).Exec(ctx); err != nil {
				return err
			}
			for _, element := range collection {
				res, err := util.WrapResource(entNetworkSegmentToProtoNetworkSegmentResource(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(HARD, res))
			}
			return nil
		})
	return deleted, txErr
}

func (is *InvStore) ListNetworkSegments(
	ctx context.Context, filter *inv_v1.ResourceFilter,
) ([]*inv_v1.GetResourceResponse, int, error) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.NetworkSegment, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.NetworkSegment, *int, error) {
			resources, total, err := filterNetworkSegments(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &resources, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.NetworkSegment, *inv_v1.GetResourceResponse](*resources,
		func(res *ent.NetworkSegment) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_NetworkSegment{
						NetworkSegment: entNetworkSegmentToProtoNetworkSegmentResource(res),
					},
				},
			}
		})
	if err := collections.FirstError[*inv_v1.GetResourceResponse](resps, validateProto[*inv_v1.GetResourceResponse]); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, 0, errors.Wrap(err)
	}

	return resps, *total, nil
}

func filterNetworkSegments(
	ctx context.Context,
	client *ent.Client,
	filter *inv_v1.ResourceFilter,
) ([]*ent.NetworkSegment, int, error) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_NETWORKSEGMENT, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[networksegment.OrderOption](filter.GetOrderBy(), networksegment.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.NetworkSegment.Query().
		Where(pred).
		WithSite().
		Order(orderOpts...).
		Offset(offset)

	// Limits number of query results if existent
	if limit != 0 {
		query = query.Limit(limit)
	}

	netsegList, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.NetworkSegment.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	return netsegList, total, nil
}

func (is *InvStore) FilterNetworkSegments(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*cl.ResourceTenantIDCarrier, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.NetworkSegment, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]*ent.NetworkSegment, *int, error) {
			filtered, total, err := filterNetworkSegments(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[*ent.NetworkSegment, *cl.ResourceTenantIDCarrier](
		*resources, func(c *ent.NetworkSegment) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.TenantID, ResourceId: c.ResourceID}
		})

	return ids, *total, err
}
