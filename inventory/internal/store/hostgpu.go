// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"

	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	hostgpus "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostgpuresource"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

var hostgpuResourceCreationValidators = []resourceValidator[*computev1.HostgpuResource]{
	protoValidator[*computev1.HostgpuResource],
	doNotAcceptResourceID[*computev1.HostgpuResource],
}

func (is *InvStore) CreateHostgpu(ctx context.Context, in *computev1.HostgpuResource) (*inv_v1.Resource, error) {
	if err := validate(in, hostgpuResourceCreationValidators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, hostgpuResourceCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("HostGpu Created: %s, %s", res.GetHostgpu().GetResourceId(), res)
	return res, nil
}

func hostgpuResourceCreator(in *computev1.HostgpuResource) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_HOSTGPU)
		zlog.Debug().Msgf("CreateHostgpu: %s", id)

		newEntity := tx.HostgpuResource.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, EmptyEnumStateMap, nil); err != nil {
			return nil, err
		}

		// Look up the optional host ID for this GPU.
		if err := setEdgeHostIDForMut(ctx, tx.Client(), mut, in.GetHost()); err != nil {
			return nil, err
		}

		// Set the resource_id field last.
		if err := mut.SetField(hostgpus.FieldResourceID, id); err != nil {
			return nil, errors.Wrap(err)
		}

		_, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, err := getHostgpuQuery(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		return util.WrapResource(entHostgpuResourceToProtoHostgpuResource(res))
	}
}

func (is *InvStore) GetHostgpu(ctx context.Context, id string) (*inv_v1.Resource, error) {
	res, err := ExecuteInRoTxAndReturnSingle[ent.HostgpuResource](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.HostgpuResource, error) {
			return getHostgpuQuery(ctx, tx, id)
		})
	if err != nil {
		return nil, err
	}

	apiResource := entHostgpuResourceToProtoHostgpuResource(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_Hostgpu{Hostgpu: apiResource}}, nil
}

func getHostgpuQuery(ctx context.Context, tx *ent.Tx, resourceID string) (*ent.HostgpuResource, error) {
	entity, err := tx.HostgpuResource.Query().
		Where(hostgpus.ResourceID(resourceID)).
		WithHost().
		Only(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return entity, nil
}

func (is *InvStore) UpdateHostgpu(
	ctx context.Context, id string, in *computev1.HostgpuResource, fieldmask *fieldmaskpb.FieldMask,
) (*inv_v1.Resource, error) {
	zlog.Debug().Msgf("UpdateHostgpu (%s): %v, fm: %v", id, in, fieldmask)

	return ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
			entity, err := tx.HostgpuResource.Query().
				Select(hostgpus.FieldID).
				Where(hostgpus.ResourceID(id)).
				Only(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			updateBuilder := tx.HostgpuResource.UpdateOneID(entity.ID)
			mut := updateBuilder.Mutation()

			// Look up the (new) referenced edges for this GPU.
			err = setRelationsForHostgpuMutIfNeeded(ctx, tx.Client(), mut, in, fieldmask)
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

			res, err := getHostgpuQuery(ctx, tx, id)
			if err != nil {
				return nil, err
			}
			toBeReturned, err := util.WrapResource(entHostgpuResourceToProtoHostgpuResource(res))

			return toBeReturned, errors.Wrap(err)
		},
	)
}

func (is *InvStore) DeleteHostgpu(ctx context.Context, id string) (*inv_v1.Resource, error) {
	// this is a "Hard Delete" as Hostgpus don't have state
	zlog.Debug().Msgf("DeleteHostgpu Hard Delete: %s", id)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, deleteHostGPU(id))

	return res, err
}

func deleteHostGPU(id string) func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		entity, err := tx.HostgpuResource.Query().
			Where(hostgpus.ResourceID(id)).
			Only(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		err = tx.HostgpuResource.DeleteOneID(entity.ID).Exec(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		return util.WrapResource(entHostgpuResourceToProtoHostgpuResource(entity))
	}
}

func (is *InvStore) DeleteHostGPUs(
	ctx context.Context, tenantID string, _ bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]
	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			all, err := tx.HostgpuResource.Query().Where(hostgpus.TenantID(tenantID)).All(ctx)
			if err != nil {
				return err
			}
			if _, err := tx.HostgpuResource.Delete().Where(hostgpus.TenantID(tenantID)).Exec(ctx); err != nil {
				return err
			}
			for _, element := range all {
				res, err := util.WrapResource(entHostgpuResourceToProtoHostgpuResource(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(HARD, res))
			}
			return nil
		})
	return deleted, txErr
}

func (is *InvStore) ListHostgpus(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*inv_v1.GetResourceResponse, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.HostgpuResource, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.HostgpuResource, *int, error) {
			filtered, total, err := filterHostgpus(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.HostgpuResource, *inv_v1.GetResourceResponse](*resources,
		func(res *ent.HostgpuResource) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Hostgpu{
						Hostgpu: entHostgpuResourceToProtoHostgpuResource(res),
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

func (is *InvStore) FilterHostgpus(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*cl.ResourceTenantIDCarrier, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.HostgpuResource, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]*ent.HostgpuResource, *int, error) {
			filtered, total, err := filterHostgpus(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[*ent.HostgpuResource, *cl.ResourceTenantIDCarrier](
		*resources, func(c *ent.HostgpuResource) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.TenantID, ResourceId: c.ResourceID}
		})

	return ids, *total, err
}

func filterHostgpus(ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter) ([]*ent.HostgpuResource, int, error) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_HOSTGPU, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[hostgpus.OrderOption](filter.GetOrderBy(), hostgpus.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.HostgpuResource.Query().
		WithHost().
		Where(pred).
		Order(orderOpts...).
		Offset(offset)

	// Limits number of query results if existent
	if limit != 0 {
		query = query.Limit(limit)
	}

	hostgpuList, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.HostgpuResource.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	return hostgpuList, total, nil
}

func setRelationsForHostgpuMutIfNeeded(
	ctx context.Context,
	client *ent.Client,
	mut *ent.HostgpuResourceMutation,
	in *computev1.HostgpuResource,
	fieldmask *fieldmaskpb.FieldMask,
) error {
	mut.ResetHost()
	if slices.Contains(fieldmask.GetPaths(), hostgpus.EdgeHost) {
		if err := setEdgeHostIDForMut(ctx, client, mut, in.GetHost()); err != nil {
			return err
		}
	}
	return nil
}
