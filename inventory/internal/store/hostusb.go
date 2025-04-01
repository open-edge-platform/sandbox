// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"

	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	hostusb "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostusbresource"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

var hostusbResourceCreationValidators = []resourceValidator[*computev1.HostusbResource]{
	protoValidator[*computev1.HostusbResource],
	doNotAcceptResourceID[*computev1.HostusbResource],
}

func (is *InvStore) CreateHostusb(ctx context.Context, in *computev1.HostusbResource) (*inv_v1.Resource, error) {
	if err := validate(in, hostusbResourceCreationValidators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, hostusbResourceCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("HostUsb Created: %s, %s", res.GetHostusb().GetResourceId(), res)
	return res, err
}

func hostusbResourceCreator(in *computev1.HostusbResource) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_HOSTUSB)
		zlog.Debug().Msgf("CreateHostusb: %s", id)

		newEntity := tx.HostusbResource.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, EmptyEnumStateMap, nil); err != nil {
			return nil, err
		}

		// Look up the optional host ID.
		if err := setEdgeHostIDForMut(ctx, tx.Client(), mut, in.GetHost()); err != nil {
			return nil, err
		}

		if err := mut.SetField(hostusb.FieldResourceID, id); err != nil {
			return nil, errors.Wrap(err)
		}

		_, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, err := getHostusbQuery(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		return util.WrapResource(entHostusbResourceToProtoHostusbResource(res))
	}
}

func (is *InvStore) GetHostusb(ctx context.Context, id string) (*inv_v1.Resource, error) {
	res, err := ExecuteInRoTxAndReturnSingle[ent.HostusbResource](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.HostusbResource, error) {
			return getHostusbQuery(ctx, tx, id)
		})
	if err != nil {
		return nil, err
	}

	apiResource := entHostusbResourceToProtoHostusbResource(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_Hostusb{Hostusb: apiResource}}, nil
}

func getHostusbQuery(ctx context.Context, tx *ent.Tx, resourceID string) (*ent.HostusbResource, error) {
	entity, err := tx.HostusbResource.Query().
		Where(hostusb.ResourceID(resourceID)).
		WithHost().
		Only(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return entity, nil
}

func (is *InvStore) UpdateHostusb(
	ctx context.Context, id string, in *computev1.HostusbResource, fieldmask *fieldmaskpb.FieldMask,
) (*inv_v1.Resource, error) {
	zlog.Debug().Msgf("UpdateHostusb (%s): %v, fm: %v", id, in, fieldmask)

	return ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
			entity, err := tx.HostusbResource.Query().
				Select(hostusb.FieldID).
				Where(hostusb.ResourceID(id)).
				Only(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			updateBuilder := tx.HostusbResource.UpdateOneID(entity.ID)
			mut := updateBuilder.Mutation()

			// Look up the (new) referenced edges for this USB.
			mut.ResetHost()
			if slices.Contains(fieldmask.GetPaths(), hostusb.EdgeHost) {
				err = setEdgeHostIDForMut(ctx, tx.Client(), mut, in.GetHost())
				if err != nil {
					return nil, err
				}
			}

			err = buildEntMutate(in, mut, EmptyEnumStateMap, fieldmask.GetPaths())
			if err != nil {
				return nil, err
			}

			_, err = updateBuilder.Save(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			res, err := getHostusbQuery(ctx, tx, id)
			if err != nil {
				return nil, err
			}
			toBeReturned, err := util.WrapResource(entHostusbResourceToProtoHostusbResource(res))

			return toBeReturned, errors.Wrap(err)
		},
	)
}

func (is *InvStore) DeleteHostusb(ctx context.Context, id string) (*inv_v1.Resource, error) {
	// this is a "Hard Delete" as Hostusbs don't have state
	zlog.Debug().Msgf("DeleteHostusb Hard Delete: %s", id)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, deleteHostUSB(id))

	return res, err
}

func deleteHostUSB(resourceID string) func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		entity, err := tx.HostusbResource.Query().
			Where(hostusb.ResourceID(resourceID)).
			Only(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		err = tx.HostusbResource.DeleteOneID(entity.ID).Exec(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		return util.WrapResource(entHostusbResourceToProtoHostusbResource(entity))
	}
}

func (is *InvStore) DeleteHostUSBs(
	ctx context.Context, tenantID string, _ bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]
	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			all, err := tx.HostusbResource.Query().Where(hostusb.TenantID(tenantID)).All(ctx)
			if err != nil {
				return err
			}
			if _, err := tx.HostusbResource.Delete().Where(hostusb.TenantID(tenantID)).Exec(ctx); err != nil {
				return err
			}
			for _, element := range all {
				res, err := util.WrapResource(entHostusbResourceToProtoHostusbResource(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(HARD, res))
			}
			return nil
		})
	return deleted, txErr
}

func filterHostusbs(ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter) ([]*ent.HostusbResource, int, error) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_HOSTUSB, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[hostusb.OrderOption](filter.GetOrderBy(), hostusb.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.HostusbResource.Query().
		WithHost().
		Where(pred).
		Order(orderOpts...).
		Offset(offset)

	// Limits number of query results if existent
	if limit != 0 {
		query = query.Limit(limit)
	}
	hostusbList, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.HostusbResource.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	return hostusbList, total, nil
}

func (is *InvStore) ListHostusb(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*inv_v1.GetResourceResponse, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.HostusbResource, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.HostusbResource, *int, error) {
			filtered, total, err := filterHostusbs(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.HostusbResource, *inv_v1.GetResourceResponse](*resources,
		func(res *ent.HostusbResource) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Hostusb{
						Hostusb: entHostusbResourceToProtoHostusbResource(res),
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

func (is *InvStore) FilterHostusb(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*cl.ResourceTenantIDCarrier, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.HostusbResource, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]*ent.HostusbResource, *int, error) {
			filtered, total, err := filterHostusbs(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[*ent.HostusbResource, *cl.ResourceTenantIDCarrier](
		*resources, func(c *ent.HostusbResource) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.TenantID, ResourceId: c.ResourceID}
		})

	return ids, *total, err
}
