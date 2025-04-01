// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"

	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	hostnics "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostnicresource"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

var hostnicResourceCreationValidators = []resourceValidator[*computev1.HostnicResource]{
	protoValidator[*computev1.HostnicResource],
	doNotAcceptResourceID[*computev1.HostnicResource],
}

// enum state mapping.
func HostnicEnumStateMap(fname string, eint int32) (ent.Value, error) {
	switch fname {
	case hostnics.FieldLinkState:
		return hostnics.LinkState(computev1.NetworkInterfaceLinkState_name[eint]), nil
	default:
		zlog.InfraSec().InfraError("unknown Enum field %s", fname).Msg("")
		return nil, errors.Errorfc(codes.InvalidArgument, "unknown Enum field %s", fname)
	}
}

func (is *InvStore) CreateHostnic(ctx context.Context, in *computev1.HostnicResource) (*inv_v1.Resource, error) {
	if err := validate(in, hostnicResourceCreationValidators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, hostnicResourceCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("HostNic Created: %s, %s", res.GetHostnic().GetResourceId(), res)
	return res, nil
}

func hostnicResourceCreator(in *computev1.HostnicResource) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_HOSTNIC)
		zlog.Debug().Msgf("CreateHostnic: %s", id)

		newEntity := tx.HostnicResource.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, HostnicEnumStateMap, nil); err != nil {
			return nil, err
		}
		// Look up the optional host ID for this NIC.
		if err := setEdgeHostIDForMut(ctx, tx.Client(), mut, in.GetHost()); err != nil {
			return nil, err
		}

		// Set the resource_id field last.
		if err := mut.SetField(hostnics.FieldResourceID, id); err != nil {
			return nil, err
		}

		_, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, err := getHostnic(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		return util.WrapResource(entHostnicResourceToProtoHostnicResource(res))
	}
}

func (is *InvStore) GetHostnic(ctx context.Context, id string) (*inv_v1.Resource, error) {
	res, err := ExecuteInRoTxAndReturnSingle[ent.HostnicResource](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.HostnicResource, error) {
			return getHostnic(ctx, tx, id)
		})
	if err != nil {
		return nil, err
	}

	apiResource := entHostnicResourceToProtoHostnicResource(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_Hostnic{Hostnic: apiResource}}, nil
}

func getHostnic(ctx context.Context, tx *ent.Tx, resourceID string) (*ent.HostnicResource, error) {
	entity, err := tx.HostnicResource.Query().
		Where(hostnics.ResourceID(resourceID)).
		WithHost().
		Only(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return entity, nil
}

func (is *InvStore) UpdateHostnic(
	ctx context.Context, id string, in *computev1.HostnicResource, fieldmask *fieldmaskpb.FieldMask,
) (*inv_v1.Resource, error) {
	zlog.Debug().Msgf("UpdateHostnic (%s): %v, fm: %v", id, in, fieldmask)

	return ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
			entity, err := tx.HostnicResource.Query().
				Select(hostnics.FieldID).
				Where(hostnics.ResourceID(id)).
				Only(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			updateBuilder := tx.HostnicResource.UpdateOneID(entity.ID)
			mut := updateBuilder.Mutation()

			// Look up the (new) referenced edges for this NIC.
			err = setRelationsForHostnicMutIfNeeded(ctx, tx.Client(), mut, in, fieldmask)
			if err != nil {
				return nil, err
			}

			err = buildEntMutate(in, mut, HostnicEnumStateMap, fieldmask.GetPaths())
			if err != nil {
				return nil, err
			}

			_, err = updateBuilder.Save(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			res, err := getHostnic(ctx, tx, id)
			if err != nil {
				return nil, err
			}
			toBeReturned, err := util.WrapResource(entHostnicResourceToProtoHostnicResource(res))

			return toBeReturned, errors.Wrap(err)
		},
	)
}

func (is *InvStore) DeleteHostnic(ctx context.Context, id string) (*inv_v1.Resource, error) {
	// this is a "Hard Delete" as Hostnics don't have state
	zlog.Debug().Msgf("DeleteHostnic Hard Delete: %s", id)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, deleteHostnic(id))

	return res, err
}

func deleteHostnic(resourceID string) func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		entity, err := tx.HostnicResource.Query().
			Where(hostnics.ResourceID(resourceID)).
			Only(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		err = tx.HostnicResource.DeleteOneID(entity.ID).Exec(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		return util.WrapResource(entHostnicResourceToProtoHostnicResource(entity))
	}
}

func (is *InvStore) DeleteHostNICs(
	ctx context.Context, tenantID string, _ bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]
	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			all, err := tx.HostnicResource.Query().Where(hostnics.TenantID(tenantID)).All(ctx)
			if err != nil {
				return err
			}
			if _, err := tx.HostnicResource.Delete().Where(hostnics.TenantID(tenantID)).Exec(ctx); err != nil {
				return err
			}
			for _, element := range all {
				res, err := util.WrapResource(entHostnicResourceToProtoHostnicResource(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(HARD, res))
			}
			return nil
		})
	return deleted, txErr
}

func filterHostnics(ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter) ([]*ent.HostnicResource, int, error) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_HOSTNIC, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[hostnics.OrderOption](filter.GetOrderBy(), hostnics.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.HostnicResource.Query().
		WithHost().
		Where(pred).
		Order(orderOpts...).
		Offset(offset)

	// Limits number of query results if existent
	if limit != 0 {
		query = query.Limit(limit)
	}

	hostnicList, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.HostnicResource.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	return hostnicList, total, nil
}

func (is *InvStore) ListHostnics(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*inv_v1.GetResourceResponse, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.HostnicResource, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.HostnicResource, *int, error) {
			filtered, total, err := filterHostnics(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.HostnicResource, *inv_v1.GetResourceResponse](*resources,
		func(res *ent.HostnicResource) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Hostnic{
						Hostnic: entHostnicResourceToProtoHostnicResource(res),
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

func (is *InvStore) FilterHostnics(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*cl.ResourceTenantIDCarrier, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.HostnicResource, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]*ent.HostnicResource, *int, error) {
			filtered, total, err := filterHostnics(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[*ent.HostnicResource, *cl.ResourceTenantIDCarrier](
		*resources, func(c *ent.HostnicResource) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.TenantID, ResourceId: c.ResourceID}
		})

	return ids, *total, err
}

func setRelationsForHostnicMutIfNeeded(
	ctx context.Context,
	client *ent.Client,
	mut *ent.HostnicResourceMutation,
	in *computev1.HostnicResource,
	fieldmask *fieldmaskpb.FieldMask,
) error {
	mut.ResetHost()
	if slices.Contains(fieldmask.GetPaths(), hostnics.EdgeHost) {
		if err := setEdgeHostIDForMut(ctx, client, mut, in.GetHost()); err != nil {
			return err
		}
	}
	return nil
}

func getNicIDFromResourceID(
	ctx context.Context,
	client *ent.Client,
	nicRes *computev1.HostnicResource,
) (int, error) {
	nic, qerr := client.HostnicResource.Query().
		Where(hostnics.ResourceID(nicRes.ResourceId)).
		Only(ctx)
	if qerr != nil {
		return 0, errors.Wrap(qerr)
	}
	return nic.ID, nil
}
