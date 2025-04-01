// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"time"

	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/remoteaccessconfiguration"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/utils/booleans"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	remoteaccessv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/remoteaccess/v1"
	statusv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/status/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

const (
	inTimeOfRemoteAccess  = time.Minute * 10
	maxTimeOfRemoteAccess = time.Hour * 24
)

func (is *InvStore) GetRemoteAccessConfig(ctx context.Context, id string) (*inv_v1.Resource, error) {
	res, err := ExecuteInRoTxAndReturnSingle[ent.RemoteAccessConfiguration](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.RemoteAccessConfiguration, error) {
			return getRemoteAccessConfigQuery(ctx, tx, id)
		})
	if err != nil {
		return nil, err
	}

	apiResource := entRemoteAccessConfigurationToProto(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_RemoteAccess{RemoteAccess: apiResource}}, nil
}

func (is *InvStore) CreateRemoteAccessConfig(ctx context.Context,
	in *remoteaccessv1.RemoteAccessConfiguration,
) (*inv_v1.Resource, error) {
	if err := validateCreationRequest(in); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, remoteAccessConfigurationCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("Remote Access Config Created: %s, %s", res.GetRemoteAccess().GetResourceId(), res)
	return res, nil
}

func (is *InvStore) UpdateRemoteAccessConfig(
	ctx context.Context,
	id string,
	in *remoteaccessv1.RemoteAccessConfiguration,
	fm *fieldmaskpb.FieldMask,
) (*inv_v1.Resource, bool, error) {
	zlog.Debug().Msgf("Update (%s): %v, fm: %v", id, in, fm)

	updated, isHardRemoval, err := ExecuteInTxAndReturnDouble[inv_v1.Resource, bool](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, *bool, error) {
			rac, err := tx.RemoteAccessConfiguration.Query().
				Select().
				Where(remoteaccessconfiguration.ResourceID(id)).
				Only(ctx)
			if err != nil {
				return nil, booleans.Pointer(false), errors.Wrap(err)
			}

			if isRemoteAccessConfigurationHardDelete(fm, rac, in) {
				if e := hardDeleteRemoteAccessConfiguration(ctx, tx, rac); e != nil {
					return nil, booleans.Pointer(false), e
				}

				var res *inv_v1.Resource
				// Set current state to be consistent on the returned value on events and upon update.
				rac.CurrentState = remoteaccessconfiguration.CurrentStateREMOTE_ACCESS_STATE_DELETED
				if res, err = util.WrapResource(entRemoteAccessConfigurationToProto(rac)); err != nil {
					return nil, booleans.Pointer(false), errors.Wrap(err)
				}
				return res, booleans.Pointer(true), nil
			}

			updateBuilder := tx.RemoteAccessConfiguration.UpdateOneID(rac.ID)
			mut := updateBuilder.Mutation()

			err = buildEntMutate(in, mut, enumStateMap, fm.GetPaths())
			if err != nil {
				return nil, booleans.Pointer(false), err
			}

			_, err = updateBuilder.Save(ctx)
			if err != nil {
				return nil, booleans.Pointer(false), errors.Wrap(err)
			}

			res, err := getRemoteAccessConfigQuery(ctx, tx, id)
			if err != nil {
				return nil, booleans.Pointer(false), err
			}
			toBeReturned, err := util.WrapResource(entRemoteAccessConfigurationToProto(res))

			return toBeReturned, booleans.Pointer(false), errors.Wrap(err)
		},
	)
	if err != nil {
		return nil, false, err
	}

	return updated, *isHardRemoval, err
}

func (is *InvStore) SoftDeleteRemoteAccessConfig(ctx context.Context, id string) (*inv_v1.Resource, error) {
	zlog.Debug().Msgf("SoftDeleteRemoteAccessConfig Soft Delete: %s", id)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
			entity, err := tx.RemoteAccessConfiguration.Query().Where(remoteaccessconfiguration.ResourceID(id)).Only(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			_, err = tx.RemoteAccessConfiguration.UpdateOneID(entity.ID).
				SetDesiredState(remoteaccessconfiguration.DesiredStateREMOTE_ACCESS_STATE_DELETED).Save(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			res, err := getRemoteAccessConfigQuery(ctx, tx, id)
			if err != nil {
				return nil, err
			}
			return util.WrapResource(entRemoteAccessConfigurationToProto(res))
		})

	return res, err
}

func (is *InvStore) FilterRemoteAccessConfig(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*cl.ResourceTenantIDCarrier, int, error,
) {
	racs, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.RemoteAccessConfiguration, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]*ent.RemoteAccessConfiguration, *int, error) {
			racs, total, err := filterRemoteAccessConfiguration(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &racs, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[*ent.RemoteAccessConfiguration, *cl.ResourceTenantIDCarrier](
		*racs, func(c *ent.RemoteAccessConfiguration) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.TenantID, ResourceId: c.ResourceID}
		})

	return ids, *total, err
}

func (is *InvStore) ListRemoteAccessConfig(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*inv_v1.GetResourceResponse, int, error,
) {
	racs, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.RemoteAccessConfiguration, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.RemoteAccessConfiguration, *int, error) {
			racs, total, err := filterRemoteAccessConfiguration(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &racs, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.RemoteAccessConfiguration, *inv_v1.GetResourceResponse](*racs, racEnt2GetResourceResponse)
	if err := collections.FirstError[*inv_v1.GetResourceResponse](resps, validateProto[*inv_v1.GetResourceResponse]); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, 0, errors.Wrap(err)
	}

	return resps, *total, nil
}

func getRemoteAccessConfigQuery(ctx context.Context, tx *ent.Tx, resourceID string) (*ent.RemoteAccessConfiguration, error) {
	entity, err := tx.RemoteAccessConfiguration.Query().
		Where(remoteaccessconfiguration.ResourceID(resourceID)).
		WithInstance().
		Only(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return entity, nil
}

func hardDeleteRemoteAccessConfiguration(ctx context.Context, tx *ent.Tx, entity *ent.RemoteAccessConfiguration) error {
	zlog.Debug().Msgf("hardDeleteRemoteAccessConfiguration(ID: %s", entity.ResourceID)
	return errors.Wrap(tx.RemoteAccessConfiguration.DeleteOneID(entity.ID).Exec(ctx))
}

func racEnt2GetResourceResponse(rac *ent.RemoteAccessConfiguration) *inv_v1.GetResourceResponse {
	return &inv_v1.GetResourceResponse{
		Resource: &inv_v1.Resource{
			Resource: &inv_v1.Resource_RemoteAccess{
				RemoteAccess: entRemoteAccessConfigurationToProto(rac),
			},
		},
	}
}

func remoteAccessConfigurationCreator(in *remoteaccessv1.RemoteAccessConfiguration) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_RMT_ACCESS_CONF)
		zlog.Debug().Msgf("CreateRemoteAccessConfig: %s", id)

		newEntity := tx.RemoteAccessConfiguration.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, enumStateMap, nil); err != nil {
			return nil, err
		}

		if err := setEdgeInstanceIDForMut(ctx, tx.Client(), mut, in.GetInstance()); err != nil {
			return nil, err
		}

		if err := mut.SetField(remoteaccessconfiguration.FieldResourceID, id); err != nil {
			return nil, errors.Wrap(err)
		}

		_, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, err := getRemoteAccessConfigQuery(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		zlog.Debug().Msgf("Remote Access Config Created: %s, %s", res.ResourceID, res)
		return util.WrapResource(entRemoteAccessConfigurationToProto(res))
	}
}

func filterRemoteAccessConfiguration(ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter) (
	[]*ent.RemoteAccessConfiguration,
	int,
	error,
) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_RMT_ACCESS_CONF, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[remoteaccessconfiguration.OrderOption](
		filter.GetOrderBy(), remoteaccessconfiguration.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.RemoteAccessConfiguration.Query().
		WithInstance().
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
	total, err := client.RemoteAccessConfiguration.Query().Where(pred).Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	return list, total, nil
}

func enumStateMap(fname string, eint int32) (ent.Value, error) {
	switch fname {
	case remoteaccessv1.RemoteAccessConfigurationFieldConfigurationStatusIndicator:
		return remoteaccessconfiguration.ConfigurationStatusIndicator(statusv1.StatusIndication_name[eint]), nil
	case remoteaccessv1.RemoteAccessConfigurationFieldDesiredState:
		return remoteaccessconfiguration.DesiredState(remoteaccessv1.RemoteAccessState_name[eint]), nil
	case remoteaccessv1.RemoteAccessConfigurationFieldCurrentState:
		return remoteaccessconfiguration.CurrentState(remoteaccessv1.RemoteAccessState_name[eint]), nil
	default:
		zlog.InfraSec().InfraError("unknown Enum field %s", fname).Msg("")
		return nil, errors.Errorfc(codes.InvalidArgument, "unknown Enum field %s", fname)
	}
}

func isRemoteAccessConfigurationHardDelete(fm *fieldmaskpb.FieldMask, entity *ent.RemoteAccessConfiguration,
	resource *remoteaccessv1.RemoteAccessConfiguration,
) bool {
	return slices.Contains(fm.GetPaths(), remoteaccessconfiguration.FieldCurrentState) &&
		entity.DesiredState == remoteaccessconfiguration.DesiredStateREMOTE_ACCESS_STATE_DELETED &&
		resource.CurrentState == remoteaccessv1.RemoteAccessState_REMOTE_ACCESS_STATE_DELETED
}
