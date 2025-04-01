// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"

	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/endpointresource"
	netlinks "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/netlinkresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/utils/booleans"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	network_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/network/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

var netlinkCreationValidators = []resourceValidator[*network_v1.NetlinkResource]{
	protoValidator[*network_v1.NetlinkResource],
	doNotAcceptResourceID[*network_v1.NetlinkResource],
}

// enum state mapping.
func NetlinkEnumStateMap(fname string, eint int32) (ent.Value, error) {
	switch fname {
	case netlinks.FieldDesiredState:
		return netlinks.DesiredState(network_v1.NetlinkState_name[eint]), nil

	case netlinks.FieldCurrentState:
		return netlinks.CurrentState(network_v1.NetlinkState_name[eint]), nil

	default:
		zlog.InfraSec().InfraError("unknown Enum field %s", fname).Msg("")
		return nil, errors.Errorfc(codes.InvalidArgument, "unknown Enum field %s", fname)
	}
}

func (is *InvStore) CreateNetlink(ctx context.Context, in *network_v1.NetlinkResource) (*inv_v1.Resource, error) {
	if err := validate(in, netlinkCreationValidators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, netlinkCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("Netlink Created: %s, %s", res.GetNetlink().GetResourceId(), res)
	return res, nil
}

func netlinkCreator(in *network_v1.NetlinkResource) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_NETLINK)
		zlog.Debug().Msgf("CreateNetlink: %s", id)

		newEntity := tx.NetlinkResource.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, NetlinkEnumStateMap, nil); err != nil {
			return nil, err
		}

		// Look up the optional src ID for this netlink.
		if in.GetSrc() != nil {
			src, qerr := tx.EndpointResource.Query().
				Where(endpointresource.ResourceID(in.GetSrc().ResourceId)).
				Only(ctx)
			if qerr != nil {
				return nil, errors.Wrap(qerr)
			}
			mut.SetSrcID(src.ID)
		}
		// Look up the optional dst ID for this netlink.
		if in.GetDst() != nil {
			dst, qerr := tx.EndpointResource.Query().
				Where(endpointresource.ResourceID(in.GetDst().ResourceId)).
				Only(ctx)
			if qerr != nil {
				return nil, errors.Wrap(qerr)
			}
			mut.SetDstID(dst.ID)
		}

		if err := mut.SetField(netlinks.FieldResourceID, id); err != nil {
			return nil, errors.Wrap(err)
		}

		_, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, err := getNetlinkQuery(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		return util.WrapResource(entNetlinkResourceToProtoNetlinkResource(res))
	}
}

func (is *InvStore) GetNetlink(ctx context.Context, id string) (*inv_v1.Resource, error) {
	res, err := ExecuteInRoTxAndReturnSingle[ent.NetlinkResource](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.NetlinkResource, error) {
			return getNetlinkQuery(ctx, tx, id)
		})
	if err != nil {
		return nil, err
	}

	apiResource := entNetlinkResourceToProtoNetlinkResource(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_Netlink{Netlink: apiResource}}, nil
}

func getNetlinkQuery(ctx context.Context, tx *ent.Tx, resourceID string) (*ent.NetlinkResource, error) {
	entity, err := tx.NetlinkResource.Query().
		Where(netlinks.ResourceID(resourceID)).
		WithSrc().
		WithDst().
		Only(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return entity, nil
}

func (is *InvStore) UpdateNetlink(
	ctx context.Context, id string, in *network_v1.NetlinkResource, fieldmask *fieldmaskpb.FieldMask,
) (*inv_v1.Resource, bool, error) {
	zlog.Debug().Msgf("UpdateNetlink (%s): %v, fm: %v", id, in, fieldmask)

	res, hardDelete, err := ExecuteInTxAndReturnDouble[inv_v1.Resource, bool](is)(ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, *bool, error) {
			entity, err := tx.NetlinkResource.Query().
				Where(netlinks.ResourceID(id)).
				Only(ctx)
			if err != nil {
				return nil, booleans.Pointer(false), errors.Wrap(err)
			}

			// hard delete - if both Desired and Current state are Deleted, remove
			if isNetlinkHardDelete(fieldmask, entity, in) {
				zlog.Debug().Msgf("UpdateNetlink Hard Delete: %s", id)

				// should be nil on success
				err = tx.NetlinkResource.DeleteOneID(entity.ID).Exec(ctx)
				if err != nil {
					return nil, booleans.Pointer(false), errors.Wrap(err)
				}

				var wrapped *inv_v1.Resource
				// Set current state to be consistent on the returned value on events and upon update.
				entity.CurrentState = netlinks.CurrentStateNETLINK_STATE_DELETED
				wrapped, err = util.WrapResource(entNetlinkResourceToProtoNetlinkResource(entity))
				if err != nil {
					return nil, booleans.Pointer(false), err
				}
				return wrapped, booleans.Pointer(true), nil
			}

			updateBuilder := tx.NetlinkResource.UpdateOneID(entity.ID)
			mut := updateBuilder.Mutation()

			// Look up the (new) referenced edges for this netlink.
			err = setRelationsForNetlinkMutIfNeeded(ctx, tx.Client(), mut, in, fieldmask)
			if err != nil {
				return nil, booleans.Pointer(false), err
			}

			err = buildEntMutate(in, mut, NetlinkEnumStateMap, fieldmask.GetPaths())
			if err != nil {
				return nil, booleans.Pointer(false), err
			}

			_, err = updateBuilder.Save(ctx)
			if err != nil {
				return nil, booleans.Pointer(false), errors.Wrap(err)
			}

			res, err := getNetlinkQuery(ctx, tx, id)
			if err != nil {
				return nil, booleans.Pointer(false), err
			}
			toBeReturned, err := util.WrapResource(entNetlinkResourceToProtoNetlinkResource(res))

			return toBeReturned, booleans.Pointer(false), errors.Wrap(err)
		},
	)
	if err != nil {
		return nil, false, err
	}

	return res, *hardDelete, err
}

func (is *InvStore) DeleteNetlink(ctx context.Context, id string) (*inv_v1.Resource, bool, error) {
	// Hard delete happens in Update, when both Desired and Current state are
	// both Deleted.
	zlog.Debug().Msgf("DeleteNetlink Soft Delete: %s", id)

	res, isSoftDelete, err := ExecuteInTxAndReturnDouble[inv_v1.Resource, bool](is)(ctx, deleteNetLink(id))
	if err != nil {
		return nil, false, err
	}

	return res, *isSoftDelete, err
}

func deleteNetLink(resourceID string) func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, *bool, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, *bool, error) {
		entity, err := tx.NetlinkResource.Query().
			Where(netlinks.ResourceID(resourceID)).
			Only(ctx)
		if err != nil {
			return nil, booleans.Pointer(false), errors.Wrap(err)
		}

		_, err = tx.NetlinkResource.UpdateOneID(entity.ID).
			SetDesiredState(netlinks.DesiredStateNETLINK_STATE_DELETED).
			Save(ctx)
		if err != nil {
			return nil, booleans.Pointer(false), errors.Wrap(err)
		}

		res, err := getNetlinkQuery(ctx, tx, resourceID)
		if err != nil {
			return nil, booleans.Pointer(false), err
		}
		toBeReturned, err := util.WrapResource(entNetlinkResourceToProtoNetlinkResource(res))
		if err != nil {
			return nil, booleans.Pointer(false), err
		}

		return toBeReturned, booleans.Pointer(true), nil
	}
}

func (is *InvStore) DeleteNetLinks(
	ctx context.Context, tenantID string, enforce bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	deletionStrategies := map[bool]func(ctx context.Context, tx *ent.Tx, tenantID string) (DeletionKind, int, error){
		true: func(ctx context.Context, tx *ent.Tx, tenantID string) (DeletionKind, int, error) {
			i, e := tx.NetlinkResource.Delete().Where(netlinks.TenantID(tenantID)).Exec(ctx)
			return HARD, i, e
		},
		false: func(ctx context.Context, tx *ent.Tx, tenantID string) (DeletionKind, int, error) {
			i, e := tx.NetlinkResource.Update().
				Where(netlinks.TenantID(tenantID)).
				SetDesiredState(netlinks.DesiredStateNETLINK_STATE_DELETED).
				Save(ctx)
			return SOFT, i, e
		},
	}
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]

	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			collection, err := tx.NetlinkResource.Query().Where(netlinks.TenantID(tenantID)).All(ctx)
			if err != nil {
				return err
			}

			dk, noOfDeleted, err := deletionStrategies[enforce](ctx, tx, tenantID)
			if err != nil {
				return err
			}
			if noOfDeleted != len(collection) {
				return errors.Errorf(
					"Returned number of updated/delete netlinks(%d) is different that number of retrieved %d",
					noOfDeleted,
					len(collection))
			}
			if dk == SOFT {
				// because of performance reasons we do not want to fetch updated instance from DB
				collections.ForEach(collection, func(i *ent.NetlinkResource) {
					i.DesiredState = netlinks.DesiredStateNETLINK_STATE_DELETED
				})
			}
			for _, element := range collection {
				res, err := util.WrapResource(entNetlinkResourceToProtoNetlinkResource(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(dk, res))
			}

			return nil
		})
	return deleted, txErr
}

func (is *InvStore) ListNetlinks(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*inv_v1.GetResourceResponse, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.NetlinkResource, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.NetlinkResource, *int, error) {
			resources, total, err := filterNetlinks(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &resources, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.NetlinkResource, *inv_v1.GetResourceResponse](*resources,
		func(res *ent.NetlinkResource) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Netlink{
						Netlink: entNetlinkResourceToProtoNetlinkResource(res),
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

func filterNetlinks(ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter) ([]*ent.NetlinkResource, int, error) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_NETLINK, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[netlinks.OrderOption](filter.GetOrderBy(), netlinks.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.NetlinkResource.Query().
		WithSrc().
		WithDst().
		Where(pred).
		Order(orderOpts...).
		Offset(offset)

	// Limits number of query results if existent
	if limit != 0 {
		query = query.Limit(limit)
	}

	netlinkList, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.NetlinkResource.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	return netlinkList, total, nil
}

func (is *InvStore) FilterNetlinks(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*cl.ResourceTenantIDCarrier, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.NetlinkResource, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]*ent.NetlinkResource, *int, error) {
			filtered, total, err := filterNetlinks(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[*ent.NetlinkResource, *cl.ResourceTenantIDCarrier](
		*resources, func(c *ent.NetlinkResource) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.TenantID, ResourceId: c.ResourceID}
		})

	return ids, *total, err
}

func setRelationsForNetlinkMutIfNeeded(
	ctx context.Context,
	client *ent.Client,
	mut *ent.NetlinkResourceMutation,
	in *network_v1.NetlinkResource,
	fieldmask *fieldmaskpb.FieldMask,
) error {
	mut.ResetSrc()
	if in.GetSrc() != nil && slices.Contains(fieldmask.GetPaths(), netlinks.EdgeSrc) {
		src, queryErr := client.EndpointResource.Query().
			Where(endpointresource.ResourceID(in.GetSrc().ResourceId)).
			Only(ctx)
		if queryErr != nil {
			return errors.Wrap(queryErr)
		}
		mut.SetSrcID(src.ID)
	}
	mut.ResetDst()
	if in.GetDst() != nil && slices.Contains(fieldmask.GetPaths(), netlinks.EdgeDst) {
		dst, queryErr := client.EndpointResource.Query().
			Where(endpointresource.ResourceID(in.GetDst().ResourceId)).
			Only(ctx)
		if queryErr != nil {
			return errors.Wrap(queryErr)
		}
		mut.SetDstID(dst.ID)
	}
	return nil
}

func isNetlinkHardDelete(fieldmask *fieldmaskpb.FieldMask, netlinkq *ent.NetlinkResource, in *network_v1.NetlinkResource) bool {
	return slices.Contains(fieldmask.GetPaths(), netlinks.FieldCurrentState) &&
		netlinkq.DesiredState == netlinks.DesiredStateNETLINK_STATE_DELETED &&
		in.CurrentState == network_v1.NetlinkState_NETLINK_STATE_DELETED
}
