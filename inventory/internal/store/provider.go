// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	hosts "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostresource"
	instances "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/instanceresource"
	providers "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/providerresource"
	sites "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/siteresource"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	provider_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/provider/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

var providerResourceCreationValidators = []resourceValidator[*provider_v1.ProviderResource]{
	protoValidator[*provider_v1.ProviderResource],
	doNotAcceptResourceID[*provider_v1.ProviderResource],
}

// enum state mapping.
func ProviderEnumStateMap(fname string, eint int32) (ent.Value, error) {
	switch fname {
	case providers.FieldProviderKind:
		return providers.ProviderKind(provider_v1.ProviderKind_name[eint]), nil

	case providers.FieldProviderVendor:
		return providers.ProviderVendor(provider_v1.ProviderVendor_name[eint]), nil

	default:
		zlog.InfraSec().InfraError("unknown Enum field %s", fname).Msg("")
		return nil, errors.Errorfc(codes.InvalidArgument, "unknown Enum field %s", fname)
	}
}

func (is *InvStore) CreateProvider(ctx context.Context, in *provider_v1.ProviderResource) (*inv_v1.Resource, error) {
	if err := validate(in, providerResourceCreationValidators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, providerResourceCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("Provider Created: %s, %s", res.GetProvider().GetResourceId(), res)
	return res, nil
}

func providerResourceCreator(in *provider_v1.ProviderResource) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_PROVIDER)
		zlog.Debug().Msgf("CreateProvider: %s", id)

		newEntity := tx.ProviderResource.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, ProviderEnumStateMap, nil); err != nil {
			return nil, err
		}

		if err := mut.SetField(providers.FieldResourceID, id); err != nil {
			return nil, errors.Wrap(err)
		}

		_, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, err := getProviderQuery(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		return util.WrapResource(entProviderResourceToProtoProviderResource(res))
	}
}

func (is *InvStore) GetProvider(ctx context.Context, id string) (*inv_v1.Resource, error) {
	res, err := ExecuteInRoTxAndReturnSingle[ent.ProviderResource](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.ProviderResource, error) {
			return getProviderQuery(ctx, tx, id)
		})
	if err != nil {
		return nil, err
	}

	apiResource := entProviderResourceToProtoProviderResource(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_Provider{Provider: apiResource}}, nil
}

func getProviderQuery(ctx context.Context, tx *ent.Tx, resourceID string) (*ent.ProviderResource, error) {
	entity, err := tx.ProviderResource.Query().
		Where(providers.ResourceID(resourceID)).
		Only(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return entity, nil
}

func (is *InvStore) UpdateProvider(
	ctx context.Context, id string, in *provider_v1.ProviderResource, fieldmask *fieldmaskpb.FieldMask,
) (*inv_v1.Resource, error) {
	zlog.Debug().Msgf("UpdateProvider (%s): %v, fm: %v", id, in, fieldmask)
	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
			entity, err := tx.ProviderResource.Query().
				Select(providers.FieldID).
				Where(providers.ResourceID(id)).
				Only(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			updateBuilder := tx.ProviderResource.UpdateOneID(entity.ID)
			mut := updateBuilder.Mutation()

			err = buildEntMutate(in, mut, ProviderEnumStateMap, fieldmask.GetPaths())
			if err != nil {
				return nil, err
			}

			_, err = updateBuilder.Save(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			res, err := getProviderQuery(ctx, tx, id)
			if err != nil {
				return nil, err
			}
			return util.WrapResource(entProviderResourceToProtoProviderResource(res))
		},
	)
	if err != nil {
		return nil, err
	}

	return res, err
}

func (is *InvStore) DeleteProvider(ctx context.Context, id string) (*inv_v1.Resource, error) {
	// this is a "Hard Delete" as Providers don't have state
	zlog.Debug().Msgf("DeleteProvider Hard Delete: %s", id)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, deleteProvider(id))

	return res, err
}

func deleteProvider(id string) func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		entity, qerr := tx.ProviderResource.Query().
			Where(providers.ResourceID(id)).
			Only(ctx)
		if qerr != nil {
			return nil, errors.Wrap(qerr)
		}

		// Error is already wrapped
		if err := verifyStrongRelations(ctx, tx, id); err != nil {
			return nil, err
		}

		if err := tx.ProviderResource.DeleteOneID(entity.ID).Exec(ctx); err != nil {
			return nil, errors.Wrap(err)
		}

		return util.WrapResource(entProviderResourceToProtoProviderResource(entity))
	}
}

func (is *InvStore) DeleteProviders(
	ctx context.Context, tenantID string, _ bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]
	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			collection, err := tx.ProviderResource.Query().Where(providers.TenantID(tenantID)).All(ctx)
			if err != nil {
				return err
			}
			if _, err := tx.ProviderResource.Delete().Where(providers.TenantID(tenantID)).Exec(ctx); err != nil {
				return err
			}
			for _, element := range collection {
				res, err := util.WrapResource(entProviderResourceToProtoProviderResource(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(HARD, res))
			}
			return nil
		})
	return deleted, txErr
}

func verifyStrongRelations(ctx context.Context, tx *ent.Tx, id string) error {
	// FIXME: this could be solved with a back-reference to hosts in the schema. Not done due to protobuf circular dep.
	// Query any child host
	_, err := tx.HostResource.Query().
		Where(hosts.HasProviderWith(providers.ResourceID(id))).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return errors.Wrap(err)
	}
	if err == nil {
		zlog.InfraSec().InfraError("the provider has a relation with host and cannot be deleted").Msg("")
		return errors.Errorfc(codes.FailedPrecondition,
			"the provider has a relation with host and cannot be deleted")
	}

	_, err = tx.InstanceResource.Query().
		Where(instances.HasProviderWith(providers.ResourceID(id))).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return errors.Wrap(err)
	}
	if err == nil {
		zlog.InfraSec().InfraError("the provider has a relation with instance and cannot be deleted").Msg("")
		return errors.Errorfc(codes.FailedPrecondition,
			"the provider has a relation with instance and cannot be deleted")
	}

	_, err = tx.SiteResource.Query().
		Where(sites.HasProviderWith(providers.ResourceID(id))).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return errors.Wrap(err)
	}
	if err == nil {
		zlog.InfraSec().InfraError("the provider has a relation with site and cannot be deleted").Msg("")
		return errors.Errorfc(codes.FailedPrecondition,
			"the provider has a relation with site and cannot be deleted")
	}

	return nil
}

func (is *InvStore) ListProviders(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*inv_v1.GetResourceResponse, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.ProviderResource, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.ProviderResource, *int, error) {
			filtered, total, err := filterProviders(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.ProviderResource, *inv_v1.GetResourceResponse](*resources,
		func(res *ent.ProviderResource) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Provider{
						Provider: entProviderResourceToProtoProviderResource(res),
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

func filterProviders(ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter) (
	[]*ent.ProviderResource, int, error,
) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_PROVIDER, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[providers.OrderOption](filter.GetOrderBy(), providers.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.ProviderResource.Query().
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
	total, err := client.ProviderResource.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	return provs, total, nil
}

func (is *InvStore) FilterProviders(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*cl.ResourceTenantIDCarrier, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.ProviderResource, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]*ent.ProviderResource, *int, error) {
			filtered, total, err := filterProviders(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[*ent.ProviderResource, *cl.ResourceTenantIDCarrier](
		*resources, func(c *ent.ProviderResource) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.TenantID, ResourceId: c.ResourceID}
		})

	return ids, *total, err
}

func getProviderIDFromResourceID(
	ctx context.Context, client *ent.Client, providerRes *provider_v1.ProviderResource,
) (int, error) {
	provider, qerr := client.ProviderResource.Query().
		Where(providers.ResourceID(providerRes.ResourceId)).
		Only(ctx)
	if qerr != nil {
		return 0, errors.Wrap(qerr)
	}
	return provider.ID, nil
}
