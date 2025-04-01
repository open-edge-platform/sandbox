// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"slices"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/tenant"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/utils/booleans"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	tenantv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/tenant/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

var validators = []resourceValidator[*tenantv1.Tenant]{
	protoValidator[*tenantv1.Tenant],
	doNotAcceptResourceID[*tenantv1.Tenant],
}

func (is *InvStore) CreateTenant(ctx context.Context, in *tenantv1.Tenant) (*inv_v1.Resource, error) {
	if err := validate(in, validators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, tenantCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("Tenant Created: %s, %s", res.GetTenant().GetResourceId(), res)
	return res, nil
}

func (is *InvStore) GetTenant(ctx context.Context, id string) (*inv_v1.Resource, error) {
	res, err := ExecuteInRoTxAndReturnSingle[ent.Tenant](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.Tenant, error) {
			return getTenantQuery(ctx, tx, id)
		})
	if err != nil {
		return nil, err
	}

	apiResource := entTenantToProto(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	return util.WrapResource(apiResource)
}

func (is *InvStore) ListTenants(ctx context.Context, filter *inv_v1.ResourceFilter) ([]*inv_v1.GetResourceResponse, int, error) {
	tenants, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.Tenant, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.Tenant, *int, error) {
			tenants, total, err := filterTenants(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &tenants, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.Tenant, *inv_v1.GetResourceResponse](
		*tenants,
		func(t *ent.Tenant) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Tenant{
						Tenant: entTenantToProto(t),
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

func (is *InvStore) FilterTenants(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*cl.ResourceTenantIDCarrier, int, error,
) {
	tenants, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.Tenant, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]*ent.Tenant, *int, error) {
			tenant, total, err := filterTenants(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &tenant, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[*ent.Tenant, *cl.ResourceTenantIDCarrier](
		*tenants, func(c *ent.Tenant) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.TenantID, ResourceId: c.ResourceID}
		})

	return ids, *total, err
}

func (is *InvStore) UpdateTenant(
	ctx context.Context,
	id string,
	in *tenantv1.Tenant,
	fm *fieldmaskpb.FieldMask,
) (*inv_v1.Resource, bool, error) {
	zlog.Debug().Msgf("Update (%s): %v, fm: %v", id, in, fm)

	updated, isHardRemoval, err := ExecuteInTxAndReturnDouble[inv_v1.Resource, bool](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, *bool, error) {
			tenant, err := tx.Tenant.Query().Select().Where(tenant.ResourceID(id)).Only(ctx)
			if err != nil {
				return nil, booleans.Pointer(false), errors.Wrap(err)
			}

			if isTenantHardDelete(fm, tenant, in) {
				if e := hardDeleteTenant(ctx, tx, tenant); e != nil {
					return nil, booleans.Pointer(false), e
				}

				var res *inv_v1.Resource
				if res, err = util.WrapResource(entTenantToProto(tenant)); err != nil {
					return nil, booleans.Pointer(false), errors.Wrap(err)
				}
				return res, booleans.Pointer(true), nil
			}

			updateBuilder := tx.Tenant.UpdateOneID(tenant.ID)
			mut := updateBuilder.Mutation()

			err = buildEntMutate(in, mut, mapTenantEnums, fm.GetPaths())
			if err != nil {
				return nil, booleans.Pointer(false), err
			}

			_, err = updateBuilder.Save(ctx)
			if err != nil {
				return nil, booleans.Pointer(false), errors.Wrap(err)
			}

			res, err := getTenantQuery(ctx, tx, id)
			if err != nil {
				return nil, booleans.Pointer(false), err
			}
			toBeReturned, err := util.WrapResource(entTenantToProto(res))

			return toBeReturned, booleans.Pointer(false), errors.Wrap(err)
		},
	)
	if err != nil {
		return nil, false, err
	}

	return updated, *isHardRemoval, err
}

func (is *InvStore) SoftDeleteTenant(ctx context.Context, id string) (*inv_v1.Resource, error) {
	zlog.Debug().Msgf("SoftDeleteTenant Soft Delete: %s", id)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
			entity, err := tx.Tenant.Query().Where(tenant.ResourceID(id)).Only(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			_, err = tx.Tenant.UpdateOneID(entity.ID).
				SetDesiredState(tenant.DesiredStateTENANT_STATE_DELETED).Save(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			res, err := getTenantQuery(ctx, tx, id)
			if err != nil {
				return nil, err
			}
			return util.WrapResource(entTenantToProto(res))
		})

	return res, err
}

func getTenantQuery(ctx context.Context, tx *ent.Tx, resourceID string) (*ent.Tenant, error) {
	entity, err := tx.Tenant.Query().Where(tenant.ResourceID(resourceID)).Only(ctx)
	return entity, errors.Wrap(err)
}

func entTenantToProto(entity *ent.Tenant) *tenantv1.Tenant {
	if entity == nil {
		return nil
	}

	return &tenantv1.Tenant{
		ResourceId:       entity.ResourceID,
		CurrentState:     tenantv1.TenantState(tenantv1.TenantState_value[entity.CurrentState.String()]),
		DesiredState:     tenantv1.TenantState(tenantv1.TenantState_value[entity.DesiredState.String()]),
		WatcherOsmanager: entity.WatcherOsmanager,
		TenantId:         entity.TenantID,
		CreatedAt:        entity.CreatedAt,
		UpdatedAt:        entity.UpdatedAt,
	}
}

func tenantCreator(in *tenantv1.Tenant) func(context.Context, *ent.Tx) (*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_TENANT)
		zlog.Debug().Msgf("Tenant: %s", id)

		newEntity := tx.Tenant.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, mapTenantEnums, nil); err != nil {
			return nil, err
		}

		if err := mut.SetField(tenant.FieldResourceID, id); err != nil {
			return nil, errors.Wrap(err)
		}

		_, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, err := getTenantQuery(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		zlog.Debug().Msgf("Tenant Created: %s, %s", res.ResourceID, res)
		return util.WrapResource(entTenantToProto(res))
	}
}

func mapTenantEnums(fname string, eint int32) (ent.Value, error) {
	switch fname {
	case tenantv1.TenantFieldCurrentState:
		return tenant.CurrentState(tenantv1.TenantState_name[eint]), nil
	case tenantv1.TenantFieldDesiredState:
		return tenant.DesiredState(tenantv1.TenantState_name[eint]), nil
	default:
		zlog.InfraSec().InfraError("unknown Enum field %s", fname).Msg("")
		return nil, errors.Errorfc(codes.InvalidArgument, "unknown Enum field %s", fname)
	}
}

func filterTenants(
	ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter,
) ([]*ent.Tenant, int, error) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_TENANT, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[tenant.OrderOption](filter.GetOrderBy(), tenant.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.Tenant.Query().
		Where(pred).
		Order(orderOpts...).
		Offset(offset)

	// Limits number of query results if existent
	if limit != 0 {
		query = query.Limit(limit)
	}

	list, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.Tenant.Query().Where(pred).Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	return list, total, nil
}

func isTenantHardDelete(fm *fieldmaskpb.FieldMask, entity *ent.Tenant, resource *tenantv1.Tenant) bool {
	return slices.Contains(fm.GetPaths(), tenant.FieldCurrentState) &&
		entity.DesiredState == tenant.DesiredStateTENANT_STATE_DELETED &&
		resource != nil &&
		resource.CurrentState == tenantv1.TenantState_TENANT_STATE_DELETED
}

func hardDeleteTenant(ctx context.Context, tx *ent.Tx, entity *ent.Tenant) error {
	zlog.Debug().Msgf("hardDeleteTenant(ID: %s", entity.ResourceID)
	return errors.Wrap(tx.Tenant.DeleteOneID(entity.ID).Exec(ctx))
}
