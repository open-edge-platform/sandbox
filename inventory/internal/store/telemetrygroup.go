// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	telemetryres "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/telemetrygroupresource"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	telemetry_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/telemetry/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

// telemetrygroup.go  store information for TelemetryResource objects

var telemetryGroupCreationValidators = []resourceValidator[*telemetry_v1.TelemetryGroupResource]{
	protoValidator[*telemetry_v1.TelemetryGroupResource],
	doNotAcceptResourceID[*telemetry_v1.TelemetryGroupResource],
}

// enum status mapping.
func TelemetryGroupEnumStatusMap(fname string, eint int32) (ent.Value, error) {
	switch fname {
	case telemetryres.FieldKind:
		return telemetryres.Kind(telemetry_v1.TelemetryResourceKind_name[eint]), nil
	case telemetryres.FieldCollectorKind:
		return telemetryres.CollectorKind(telemetry_v1.CollectorKind_name[eint]), nil
	default:
		zlog.InfraSec().InfraError("unknown Enum field %s", fname).Msg("")
		return nil, errors.Errorfc(codes.InvalidArgument, "unknown Enum field %s", fname)
	}
}

func (is *InvStore) CreateTelemetryGroup(
	ctx context.Context,
	in *telemetry_v1.TelemetryGroupResource,
) (*inv_v1.Resource, error) {
	if err := validate(in, telemetryGroupCreationValidators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, telemetryGroupResourceCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("Telemetry Group Created: %s, %s", res.GetTelemetryGroup().GetResourceId(), res)
	return res, nil
}

func telemetryGroupResourceCreator(in *telemetry_v1.TelemetryGroupResource) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_TELEMETRY_GROUP)
		zlog.Debug().Msgf("CreateTelemetryGroup: %s", id)

		newEntity := tx.TelemetryGroupResource.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, TelemetryGroupEnumStatusMap, nil); err != nil {
			return nil, err
		}

		if err := mut.SetField(telemetryres.FieldResourceID, id); err != nil {
			return nil, errors.Wrap(err)
		}

		_, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, err := getTelemetryGroup(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		return util.WrapResource(entTelemetryGroupResourceToProtoTelemetryGroupResource(res))
	}
}

func (is *InvStore) GetTelemetryGroup(
	ctx context.Context, id string,
) (*inv_v1.Resource, error) {
	res, err := ExecuteInRoTxAndReturnSingle[ent.TelemetryGroupResource](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.TelemetryGroupResource, error) {
			return getTelemetryGroup(ctx, tx, id)
		})
	if err != nil {
		return nil, err
	}

	apiResource := entTelemetryGroupResourceToProtoTelemetryGroupResource(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_TelemetryGroup{TelemetryGroup: apiResource}}, nil
}

func getTelemetryGroup(ctx context.Context, tx *ent.Tx, resourceID string) (*ent.TelemetryGroupResource, error) {
	entity, err := tx.TelemetryGroupResource.Query().
		Where(telemetryres.ResourceID(resourceID)).
		WithProfiles().
		Only(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return entity, nil
}

func (is *InvStore) UpdateTelemetryGroup(
	ctx context.Context, id string, in *telemetry_v1.TelemetryGroupResource, fieldmask *fieldmaskpb.FieldMask,
) (*inv_v1.Resource, error) {
	zlog.Debug().Msgf("UpdateTelemetryGroup (%s): %v, fm: %v", id, in, fieldmask)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
			entity, err := tx.TelemetryGroupResource.Query().
				Select(telemetryres.FieldID).
				Where(telemetryres.ResourceID(id)).
				Only(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			updateBuilder := tx.TelemetryGroupResource.UpdateOneID(entity.ID)
			mut := updateBuilder.Mutation()

			err = buildEntMutate(in, mut, TelemetryGroupEnumStatusMap, fieldmask.GetPaths())
			if err != nil {
				return nil, err
			}

			_, err = updateBuilder.Save(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			res, err := getTelemetryGroup(ctx, tx, id)
			if err != nil {
				return nil, err
			}
			return util.WrapResource(entTelemetryGroupResourceToProtoTelemetryGroupResource(res))
		})
	if err != nil {
		return nil, err
	}

	return res, err
}

func (is *InvStore) DeleteTelemetryGroup(ctx context.Context, id string) (*inv_v1.Resource, error) {
	zlog.Debug().Msgf("DeleteTelemetryGroup: %s", id)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(
		ctx,
		deleteTelemetryGroup(id))

	return res, err
}

func deleteTelemetryGroup(id string) func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		entity, err := tx.TelemetryGroupResource.Query().
			Where(telemetryres.ResourceID(id)).
			Only(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		err = tx.TelemetryGroupResource.DeleteOneID(entity.ID).Exec(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		return util.WrapResource(entTelemetryGroupResourceToProtoTelemetryGroupResource(entity))
	}
}

func (is *InvStore) DeleteTelemetryGroups(
	ctx context.Context, tenantID string, _ bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]
	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			collection, err := tx.TelemetryGroupResource.Query().Where(telemetryres.TenantID(tenantID)).All(ctx)
			if err != nil {
				return err
			}
			if _, err := tx.TelemetryGroupResource.Delete().Where(telemetryres.TenantID(tenantID)).Exec(ctx); err != nil {
				return err
			}
			for _, element := range collection {
				res, err := util.WrapResource(entTelemetryGroupResourceToProtoTelemetryGroupResource(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(HARD, res))
			}
			return nil
		})
	return deleted, txErr
}

func filterTelemetryGroupResources(ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter) (
	[]*ent.TelemetryGroupResource,
	int,
	error,
) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_TELEMETRY_GROUP, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[telemetryres.OrderOption](filter.GetOrderBy(), telemetryres.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.TelemetryGroupResource.Query().
		Where(pred).
		Order(orderOpts...).
		WithProfiles().
		Offset(offset)

	// Limits number of query results if existent
	if limit != 0 {
		query = query.Limit(limit)
	}

	respList, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.TelemetryGroupResource.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	return respList, total, nil
}

func (is *InvStore) ListTelemetryGroup(
	ctx context.Context, filter *inv_v1.ResourceFilter,
) ([]*inv_v1.GetResourceResponse, int, error) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.TelemetryGroupResource, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.TelemetryGroupResource, *int, error) {
			resources, total, err := filterTelemetryGroupResources(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &resources, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.TelemetryGroupResource, *inv_v1.GetResourceResponse](*resources,
		func(res *ent.TelemetryGroupResource) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_TelemetryGroup{
						TelemetryGroup: entTelemetryGroupResourceToProtoTelemetryGroupResource(res),
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

func (is *InvStore) FilterTelemetryGroup(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*cl.ResourceTenantIDCarrier, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.TelemetryGroupResource, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]*ent.TelemetryGroupResource, *int, error) {
			filtered, total, err := filterTelemetryGroupResources(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[*ent.TelemetryGroupResource, *cl.ResourceTenantIDCarrier](
		*resources, func(c *ent.TelemetryGroupResource) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.TenantID, ResourceId: c.ResourceID}
		})

	return ids, *total, err
}

func getTelemetryGroupIDFromResourceID(
	ctx context.Context,
	client *ent.Client,
	telemetryGroupRes *telemetry_v1.TelemetryGroupResource,
) (int, error) {
	obj, qerr := client.TelemetryGroupResource.Query().
		Where(telemetryres.ResourceID(telemetryGroupRes.ResourceId)).
		Only(ctx)
	if qerr != nil {
		return 0, errors.Wrap(qerr)
	}
	return obj.ID, nil
}
