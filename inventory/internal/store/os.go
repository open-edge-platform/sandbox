// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

// os.go  store information for OS objects

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	oss "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/operatingsystemresource"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	os_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

var osResourceCreationValidators = []resourceValidator[*os_v1.OperatingSystemResource]{
	protoValidator[*os_v1.OperatingSystemResource],
	validateOsProto,
	doNotAcceptResourceID[*os_v1.OperatingSystemResource],
}

func validateOsProto(in *os_v1.OperatingSystemResource) error {
	if in.GetOsType() == os_v1.OsType_OS_TYPE_UNSPECIFIED {
		return errors.Errorfc(codes.InvalidArgument, "OS type cannot be unspecified")
	}
	if in.GetOsProvider() == os_v1.OsProviderKind_OS_PROVIDER_KIND_UNSPECIFIED {
		return errors.Errorfc(codes.InvalidArgument, "OS provider cannot be unspecified")
	}

	return nil
}

// OsEnumStateMap maps proto enum fields to their Ent equivalents.
func OsEnumStateMap(fname string, eint int32) (ent.Value, error) {
	switch fname {
	case oss.FieldSecurityFeature:
		return oss.SecurityFeature(os_v1.SecurityFeature_name[eint]), nil
	case oss.FieldOsType:
		return oss.OsType(os_v1.OsType_name[eint]), nil
	case oss.FieldOsProvider:
		return oss.OsProvider(os_v1.OsProviderKind_name[eint]), nil
	default:
		zlog.InfraSec().InfraError("unknown Enum field %s", fname).Msg("")
		return nil, errors.Errorfc(codes.InvalidArgument, "unknown Enum field %s", fname)
	}
}

func (is *InvStore) CreateOs(ctx context.Context, in *os_v1.OperatingSystemResource) (*inv_v1.Resource, error) {
	if err := validate(in, osResourceCreationValidators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, osResourceCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("OS Created: %s, %s", res.GetOs().GetResourceId(), res)
	return res, nil
}

func osResourceCreator(in *os_v1.OperatingSystemResource) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_OS)
		zlog.Debug().Msgf("CreateOs: %s", id)

		newEntity := tx.OperatingSystemResource.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, OsEnumStateMap, nil); err != nil {
			return nil, err
		}

		if err := mut.SetField(oss.FieldResourceID, id); err != nil {
			return nil, errors.Wrap(err)
		}

		_, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, err := getOsQuery(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		return util.WrapResource(entOperatingSystemResourceToProtoOperatingSystemResource(res))
	}
}

func (is *InvStore) GetOs(ctx context.Context, id string) (*inv_v1.Resource, error) {
	res, err := ExecuteInRoTxAndReturnSingle[ent.OperatingSystemResource](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.OperatingSystemResource, error) {
			return getOsQuery(ctx, tx, id)
		})
	if err != nil {
		return nil, err
	}

	apiResource := entOperatingSystemResourceToProtoOperatingSystemResource(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_Os{Os: apiResource}}, nil
}

func getOsQuery(ctx context.Context, tx *ent.Tx, resourceID string) (*ent.OperatingSystemResource, error) {
	entity, err := tx.OperatingSystemResource.Query().
		Where(oss.ResourceID(resourceID)).
		Only(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return entity, nil
}

func (is *InvStore) UpdateOs(
	ctx context.Context,
	id string,
	in *os_v1.OperatingSystemResource,
	fieldmask *fieldmaskpb.FieldMask,
) (*inv_v1.Resource, error) {
	zlog.Debug().Msgf("UpdateOs (%s): %v, fm: %v", id, in, fieldmask)
	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
			entity, err := tx.OperatingSystemResource.Query().
				// we need to also retrieve immutable fields to do the check
				Select(oss.FieldID).
				Where(oss.ResourceID(id)).
				Only(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			updateBuilder := tx.OperatingSystemResource.UpdateOneID(entity.ID)
			mut := updateBuilder.Mutation()

			err = buildEntMutate(in, mut, OsEnumStateMap, fieldmask.GetPaths())
			if err != nil {
				return nil, err
			}

			_, err = updateBuilder.Save(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			res, err := getOsQuery(ctx, tx, id)
			if err != nil {
				return nil, err
			}
			return util.WrapResource(entOperatingSystemResourceToProtoOperatingSystemResource(res))
		})
	if err != nil {
		return nil, err
	}

	return res, err
}

func (is *InvStore) DeleteOs(ctx context.Context, id string) (*inv_v1.Resource, error) {
	// this is a "Hard Delete" as os don't have state to reconcile
	zlog.Debug().Msgf("Deleteos Hard Delete: %s", id)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, deleteOs(id))

	return res, err
}

func deleteOs(resourceID string) func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		entity, err := tx.OperatingSystemResource.Query().
			Where(oss.ResourceID(resourceID)).
			Only(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		err = tx.OperatingSystemResource.DeleteOneID(entity.ID).Exec(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		return util.WrapResource(entOperatingSystemResourceToProtoOperatingSystemResource(entity))
	}
}

func (is *InvStore) DeleteOSes(
	ctx context.Context, tenantID string, _ bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]
	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			collection, err := tx.OperatingSystemResource.Query().Where(oss.TenantID(tenantID)).All(ctx)
			if err != nil {
				return err
			}
			if _, err := tx.OperatingSystemResource.Delete().Where(oss.TenantID(tenantID)).Exec(ctx); err != nil {
				return err
			}
			for _, element := range collection {
				res, err := util.WrapResource(entOperatingSystemResourceToProtoOperatingSystemResource(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(HARD, res))
			}
			return nil
		})
	return deleted, txErr
}

func filterOss(ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter) (
	[]*ent.OperatingSystemResource, int, error,
) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_OS, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[oss.OrderOption](filter.GetOrderBy(), oss.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.OperatingSystemResource.Query().
		Where(pred).
		Order(orderOpts...).
		Offset(offset)

	// Limits number of query results if existent
	if limit != 0 {
		query = query.Limit(limit)
	}

	osList, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.OperatingSystemResource.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	return osList, total, nil
}

func (is *InvStore) ListOss(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*inv_v1.GetResourceResponse, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.OperatingSystemResource, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.OperatingSystemResource, *int, error) {
			resources, total, err := filterOss(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &resources, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.OperatingSystemResource, *inv_v1.GetResourceResponse](*resources,
		func(res *ent.OperatingSystemResource) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Os{
						Os: entOperatingSystemResourceToProtoOperatingSystemResource(res),
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

func (is *InvStore) FilterOss(ctx context.Context, filter *inv_v1.ResourceFilter) ([]*cl.ResourceTenantIDCarrier, int, error) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.OperatingSystemResource, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]*ent.OperatingSystemResource, *int, error) {
			filtered, total, err := filterOss(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[*ent.OperatingSystemResource, *cl.ResourceTenantIDCarrier](
		*resources, func(c *ent.OperatingSystemResource) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.TenantID, ResourceId: c.ResourceID}
		})

	return ids, *total, err
}

func getOSIDFromResourceID(ctx context.Context, client *ent.Client, osres *os_v1.OperatingSystemResource) (int, error) {
	os, qerr := client.OperatingSystemResource.Query().
		Where(oss.ResourceID(osres.ResourceId)).
		Only(ctx)
	if qerr != nil {
		return 0, errors.Wrap(qerr)
	}
	return os.ID, nil
}
