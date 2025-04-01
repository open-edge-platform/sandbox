// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"

	"google.golang.org/grpc/codes"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	instances "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/instanceresource"
	localaccounts "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/localaccountresource"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	localaccount_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/localaccount/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

var localAccountResourceCreationValidators = []resourceValidator[*localaccount_v1.LocalAccountResource]{
	protoValidator[*localaccount_v1.LocalAccountResource],
	doNotAcceptResourceID[*localaccount_v1.LocalAccountResource],
}

func (is *InvStore) CreateLocalAccount(ctx context.Context, in *localaccount_v1.LocalAccountResource) (*inv_v1.Resource, error) {
	if err := validate(in, localAccountResourceCreationValidators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, localAccountResourceCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("LocalAccount Created: %s, %s", res.GetLocalAccount().GetResourceId(), res)
	return res, nil
}

func localAccountResourceCreator(in *localaccount_v1.LocalAccountResource) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_LOCALACCOUNT)
		zlog.Debug().Msgf("LocalAccount: %s", id)

		newEntity := tx.LocalAccountResource.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, EmptyEnumStateMap, nil); err != nil {
			return nil, err
		}

		if err := mut.SetField(localaccounts.FieldResourceID, id); err != nil {
			return nil, errors.Wrap(err)
		}

		_, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, err := getLocalAccountQuery(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		return util.WrapResource(entLocalAccountResourceToProtoLocalAccountResource(res))
	}
}

func getLocalAccountQuery(ctx context.Context, tx *ent.Tx, resourceID string) (*ent.LocalAccountResource, error) {
	entity, err := tx.LocalAccountResource.Query().
		Where(localaccounts.ResourceID(resourceID)).
		Only(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return entity, nil
}

func (is *InvStore) GetLocalAccount(ctx context.Context, id string) (*inv_v1.Resource, error) {
	res, err := ExecuteInRoTxAndReturnSingle[ent.LocalAccountResource](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.LocalAccountResource, error) {
			return getLocalAccountQuery(ctx, tx, id)
		})
	if err != nil {
		return nil, err
	}

	apiResource := entLocalAccountResourceToProtoLocalAccountResource(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_LocalAccount{LocalAccount: apiResource}}, nil
}

func (is *InvStore) DeleteLocalAccount(ctx context.Context, id string) (*inv_v1.Resource, error) {
	zlog.Debug().Msgf("DeleteLocalAccount Delete: %s", id)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, deleteLocalAccount(id))

	return res, err
}

func deleteLocalAccount(id string) func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		entity, qerr := tx.LocalAccountResource.Query().
			Where(localaccounts.ResourceID(id)).
			Only(ctx)
		if qerr != nil {
			return nil, errors.Wrap(qerr)
		}

		// Error is already wrapped
		if err := verifyLocalAccountStrongRelations(ctx, tx, id); err != nil {
			return nil, err
		}

		if err := tx.LocalAccountResource.DeleteOneID(entity.ID).Exec(ctx); err != nil {
			return nil, errors.Wrap(err)
		}

		return util.WrapResource(entLocalAccountResourceToProtoLocalAccountResource(entity))
	}
}

func verifyLocalAccountStrongRelations(ctx context.Context, tx *ent.Tx, id string) error {
	_, err := tx.InstanceResource.Query().
		Where(instances.HasLocalaccountWith(localaccounts.ResourceID(id))).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return errors.Wrap(err)
	}

	if err == nil {
		zlog.InfraSec().InfraError("the localaccount has a relation with instance and cannot be deleted").Msg("")
		return errors.Errorfc(codes.FailedPrecondition,
			"the localaccount has a relation with instance and cannot be deleted")
	}

	return nil
}

func (is *InvStore) DeleteLocalAccounts(
	ctx context.Context, tenantID string, _ bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]
	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			collection, err := tx.LocalAccountResource.Query().Where(localaccounts.TenantID(tenantID)).All(ctx)
			if err != nil {
				return err
			}
			if _, err := tx.LocalAccountResource.Delete().Where(localaccounts.TenantID(tenantID)).Exec(ctx); err != nil {
				return err
			}
			for _, element := range collection {
				res, err := util.WrapResource(entLocalAccountResourceToProtoLocalAccountResource(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(HARD, res))
			}
			return nil
		})
	return deleted, txErr
}

func (is *InvStore) ListLocalAccounts(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*inv_v1.GetResourceResponse, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.LocalAccountResource, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.LocalAccountResource, *int, error) {
			filtered, total, err := filterLocalAccounts(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.LocalAccountResource, *inv_v1.GetResourceResponse](*resources,
		func(res *ent.LocalAccountResource) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_LocalAccount{
						LocalAccount: entLocalAccountResourceToProtoLocalAccountResource(res),
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

func filterLocalAccounts(ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter) (
	[]*ent.LocalAccountResource, int, error,
) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_LOCALACCOUNT, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[localaccounts.OrderOption](filter.GetOrderBy(), localaccounts.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.LocalAccountResource.Query().
		Where(pred).
		Order(orderOpts...).
		Offset(offset)

	// Limits number of query results if existent
	if limit != 0 {
		query = query.Limit(limit)
	}

	provs, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.LocalAccountResource.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	return provs, total, nil
}

func (is *InvStore) FilterLocalAccounts(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*cl.ResourceTenantIDCarrier, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.LocalAccountResource, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]*ent.LocalAccountResource, *int, error) {
			filtered, total, err := filterLocalAccounts(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[*ent.LocalAccountResource, *cl.ResourceTenantIDCarrier](
		*resources, func(c *ent.LocalAccountResource) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.TenantID, ResourceId: c.ResourceID}
		})

	return ids, *total, err
}

func getLocalAccountIDFromResourceID(
	ctx context.Context, client *ent.Client, localAccountRes *localaccount_v1.LocalAccountResource,
) (int, error) {
	localAccount, qerr := client.LocalAccountResource.Query().
		Where(localaccounts.ResourceID(localAccountRes.ResourceId)).
		Only(ctx)
	if qerr != nil {
		return 0, errors.Wrap(qerr)
	}
	return localAccount.ID, nil
}
