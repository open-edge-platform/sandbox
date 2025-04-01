// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"

	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	hoststorage "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hoststorageresource"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

var hoststorageResourceCreationValidators = []resourceValidator[*computev1.HoststorageResource]{
	protoValidator[*computev1.HoststorageResource],
	doNotAcceptResourceID[*computev1.HoststorageResource],
}

func (is *InvStore) CreateHoststorage(ctx context.Context, in *computev1.HoststorageResource) (*inv_v1.Resource, error) {
	if err := validate(in, hoststorageResourceCreationValidators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, hoststorageResourceCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("HostStorage Created: %s, %s", res.GetHoststorage().GetResourceId(), res)
	return res, nil
}

func hoststorageResourceCreator(in *computev1.HoststorageResource) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_HOSTSTORAGE)
		zlog.Debug().Msgf("CreateHoststorage: %s", id)

		newEntity := tx.HoststorageResource.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, EmptyEnumStateMap, nil); err != nil {
			return nil, err
		}
		// Look up the optional host ID for this storage.
		if err := setEdgeHostIDForMut(ctx, tx.Client(), mut, in.GetHost()); err != nil {
			return nil, err
		}

		// Set the resource_id field last.
		if err := mut.SetField(hoststorage.FieldResourceID, id); err != nil {
			return nil, errors.Wrap(err)
		}

		_, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, err := getHoststorageQuery(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		return util.WrapResource(entHostStorageResourceToProtoHostStorageResource(res))
	}
}

func (is *InvStore) GetHoststorage(ctx context.Context, id string) (*inv_v1.Resource, error) {
	res, err := ExecuteInRoTxAndReturnSingle[ent.HoststorageResource](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.HoststorageResource, error) {
			return getHoststorageQuery(ctx, tx, id)
		})
	if err != nil {
		return nil, err
	}

	apiResource := entHostStorageResourceToProtoHostStorageResource(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_Hoststorage{Hoststorage: apiResource}}, nil
}

func getHoststorageQuery(ctx context.Context, tx *ent.Tx, resourceID string) (*ent.HoststorageResource, error) {
	entity, err := tx.HoststorageResource.Query().
		Where(hoststorage.ResourceID(resourceID)).
		WithHost().
		Only(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return entity, nil
}

func (is *InvStore) UpdateHoststorage(
	ctx context.Context, id string, in *computev1.HoststorageResource, fieldmask *fieldmaskpb.FieldMask,
) (*inv_v1.Resource, error) {
	zlog.Debug().Msgf("UpdateHoststorage (%s): %v, fm: %v", id, in, fieldmask)

	return ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
			entity, err := tx.HoststorageResource.Query().
				Select(hoststorage.FieldID).
				Where(hoststorage.ResourceID(id)).
				Only(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			updateBuilder := tx.HoststorageResource.UpdateOneID(entity.ID)
			mut := updateBuilder.Mutation()

			// Look up the (new) referenced edges for this storage.
			err = setRelationsForHoststorageMutIfNeeded(ctx, tx.Client(), mut, in, fieldmask)
			if err != nil {
				return nil, err
			}

			err = buildEntMutate(in, mut, EmptyEnumStateMap, fieldmask.GetPaths())
			if err != nil {
				return nil, err
			}

			_, err = updateBuilder.Save(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			res, err := getHoststorageQuery(ctx, tx, id)
			if err != nil {
				return nil, err
			}
			toBeReturned, err := util.WrapResource(entHostStorageResourceToProtoHostStorageResource(res))

			return toBeReturned, errors.Wrap(err)
		},
	)
}

func (is *InvStore) DeleteHoststorage(ctx context.Context, id string) (*inv_v1.Resource, error) {
	// this is a "Hard Delete" as Hoststorages don't have state
	zlog.Debug().Msgf("DeleteHoststorage Hard Delete: %s", id)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, deleteHostStorage(id))

	return res, err
}

func deleteHostStorage(id string) func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		entity, err := tx.HoststorageResource.Query().
			Where(hoststorage.ResourceID(id)).
			Only(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		err = tx.HoststorageResource.DeleteOneID(entity.ID).Exec(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		return util.WrapResource(entHostStorageResourceToProtoHostStorageResource(entity))
	}
}

func (is *InvStore) DeleteHostStorages(
	ctx context.Context, tenantID string, _ bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]
	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			all, err := tx.HoststorageResource.Query().Where(hoststorage.TenantID(tenantID)).All(ctx)
			if err != nil {
				return err
			}
			if _, err := tx.HoststorageResource.Delete().Where(hoststorage.TenantID(tenantID)).Exec(ctx); err != nil {
				return err
			}
			for _, element := range all {
				res, err := util.WrapResource(entHostStorageResourceToProtoHostStorageResource(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(HARD, res))
			}
			return nil
		})
	return deleted, txErr
}

func filterHoststorages(ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter) (
	[]*ent.HoststorageResource,
	int,
	error,
) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_HOSTSTORAGE, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[hoststorage.OrderOption](filter.GetOrderBy(), hoststorage.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.HoststorageResource.Query().
		WithHost().
		Where(pred).
		Order(orderOpts...).
		Offset(offset)

	// Limits number of query results if existent
	if limit != 0 {
		query = query.Limit(limit)
	}

	hoststorageList, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.HoststorageResource.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	return hoststorageList, total, nil
}

func (is *InvStore) ListHoststorage(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*inv_v1.GetResourceResponse, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.HoststorageResource, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.HoststorageResource, *int, error) {
			filtered, total, err := filterHoststorages(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.HoststorageResource, *inv_v1.GetResourceResponse](*resources,
		func(res *ent.HoststorageResource) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Hoststorage{
						Hoststorage: entHostStorageResourceToProtoHostStorageResource(res),
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

func (is *InvStore) FilterHoststorage(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*cl.ResourceTenantIDCarrier, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.HoststorageResource, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]*ent.HoststorageResource, *int, error) {
			filtered, total, err := filterHoststorages(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[*ent.HoststorageResource, *cl.ResourceTenantIDCarrier](
		*resources, func(c *ent.HoststorageResource) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.TenantID, ResourceId: c.ResourceID}
		})

	return ids, *total, err
}

func setRelationsForHoststorageMutIfNeeded(
	ctx context.Context,
	client *ent.Client,
	mut *ent.HoststorageResourceMutation,
	in *computev1.HoststorageResource,
	fieldmask *fieldmaskpb.FieldMask,
) error {
	mut.ResetHost()
	if slices.Contains(fieldmask.GetPaths(), hoststorage.EdgeHost) {
		if err := setEdgeHostIDForMut(ctx, client, mut, in.GetHost()); err != nil {
			return err
		}
	}
	return nil
}
