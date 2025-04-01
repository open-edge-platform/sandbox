// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/instanceresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/workloadmember"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/workloadresource"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

// workload_member.go  store information for workload member

var workloadMemberCreationValidators = []resourceValidator[*computev1.WorkloadMember]{
	protoValidator[*computev1.WorkloadMember],
	doNotAcceptResourceID[*computev1.WorkloadMember],
}

func WorkloadMemberEnumStatusMap(fname string, eint int32) (ent.Value, error) {
	switch fname {
	case workloadmember.FieldKind:
		return workloadmember.Kind(computev1.WorkloadMemberKind_name[eint]), nil
	default:
		zlog.InfraSec().InfraError("unknown Enum field %s", fname).Msg("")
		return nil, errors.Errorfc(codes.InvalidArgument, "unknown Enum field %s", fname)
	}
}

// checkInstanceWorkloadMemberEdgeIsUniqueGlobally verifies that the given Instance resourceID is pointed by a single
// workloadMember in the DB. Used after creating or updating a WorkloadMember to validate invariants.
func checkInstanceWorkloadMemberEdgeIsUniqueGlobally(ctx context.Context, client *ent.Client, instanceResourceID string) error {
	// Number of workload members that point to the same instance.
	numMembers, err := client.WorkloadMember.Query().
		Where(workloadmember.HasInstanceWith(
			instanceresource.ResourceIDEQ(instanceResourceID))).
		Count(ctx)
	if err != nil {
		return err
	}
	if numMembers > 1 {
		return errors.Errorfc(codes.AlreadyExists,
			"workload member referencing instance %v already exists", instanceResourceID)
	}

	return nil
}

func (is *InvStore) CreateWorkloadMember(ctx context.Context, in *computev1.WorkloadMember) (*inv_v1.Resource, error) {
	if err := validate(in, workloadMemberCreationValidators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, workloadMemberCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("WorkloadMember Created: %s, %s", res.GetWorkloadMember().GetResourceId(), res)
	return res, nil
}

func workloadMemberCreator(in *computev1.WorkloadMember) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_WORKLOAD_MEMBER)
		zlog.Debug().Msgf("CreateWorkloadMember: %s", id)

		newEntity := tx.WorkloadMember.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, WorkloadMemberEnumStatusMap, nil); err != nil {
			return nil, err
		}

		// Look up the mandatory edges
		if err := setEdgeWorkloadIDForMut(ctx, tx.Client(), mut, in.GetWorkload()); err != nil {
			return nil, err
		}
		if err := setEdgeInstanceIDForMut(ctx, tx.Client(), mut, in.GetInstance()); err != nil {
			return nil, err
		}

		if err := mut.SetField(workloadresource.FieldResourceID, id); err != nil {
			return nil, errors.Wrap(err)
		}

		_, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, err := getWorkloadMemberQuery(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		if err := checkInstanceWorkloadMemberEdgeIsUniqueGlobally(ctx, tx.Client(), res.Edges.Instance.ResourceID); err != nil {
			return nil, err
		}

		return util.WrapResource(entWorkloadMemberToProtoWorkloadMember(res))
	}
}

func (is *InvStore) GetWorkloadMember(ctx context.Context, id string) (*inv_v1.Resource, error) {
	res, err := ExecuteInRoTxAndReturnSingle[ent.WorkloadMember](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.WorkloadMember, error) {
			return getWorkloadMemberQuery(ctx, tx, id)
		})
	if err != nil {
		return nil, err
	}

	apiResource := entWorkloadMemberToProtoWorkloadMember(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_WorkloadMember{WorkloadMember: apiResource}}, nil
}

func getWorkloadMemberQuery(ctx context.Context, tx *ent.Tx, resourceID string) (*ent.WorkloadMember, error) {
	entity, err := tx.WorkloadMember.Query().
		Where(workloadmember.ResourceID(resourceID)).
		WithWorkload().
		WithInstance().
		Only(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return entity, nil
}

func (is *InvStore) UpdateWorkloadMember(
	ctx context.Context, id string, in *computev1.WorkloadMember, fieldmask *fieldmaskpb.FieldMask,
) (*inv_v1.Resource, error) {
	zlog.Debug().Msgf("UpdateWorkloadMember (%s): %v, fm: %v", id, in, fieldmask)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
			entity, err := tx.WorkloadMember.Query().
				Select(workloadmember.FieldID).
				Where(workloadmember.ResourceID(id)).
				Only(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			updateBuilder := tx.WorkloadMember.UpdateOneID(entity.ID)
			mut := updateBuilder.Mutation()

			// Look up the (new) referenced edges for this workload member.
			err = setRelationsForWorkloadMemberMutIfNeeded(ctx, tx.Client(), mut, in)
			if err != nil {
				return nil, err
			}

			err = buildEntMutate(in, mut, WorkloadMemberEnumStatusMap, fieldmask.GetPaths())
			if err != nil {
				return nil, err
			}

			_, err = updateBuilder.Save(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			res, err := getWorkloadMemberQuery(ctx, tx, id)
			if err != nil {
				return nil, err
			}
			if err := checkInstanceWorkloadMemberEdgeIsUniqueGlobally(
				ctx, tx.Client(), res.Edges.Instance.ResourceID); err != nil {
				return nil, err
			}

			return util.WrapResource(entWorkloadMemberToProtoWorkloadMember(res))
		})
	if err != nil {
		return nil, err
	}

	return res, err
}

func setRelationsForWorkloadMemberMutIfNeeded(
	ctx context.Context, client *ent.Client, mut *ent.WorkloadMemberMutation, in *computev1.WorkloadMember,
) error {
	mut.ResetWorkload()
	if err := setEdgeWorkloadIDForMut(ctx, client, mut, in.GetWorkload()); err != nil {
		return err
	}
	mut.ResetInstance()
	return setEdgeInstanceIDForMut(ctx, client, mut, in.GetInstance())
}

func (is *InvStore) DeleteWorkloadMember(ctx context.Context, id string) (*inv_v1.Resource, error) {
	// this is a "Hard Delete" as workload member don't have state to reconcile
	zlog.Debug().Msgf("DeleteWorkloadMember Hard Delete: %s", id)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, deleteWorkloadMember(id))

	return res, err
}

func deleteWorkloadMember(id string) func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		entity, err := tx.WorkloadMember.Query().Where(workloadmember.ResourceID(id)).Only(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		err = tx.WorkloadMember.DeleteOneID(entity.ID).Exec(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		return util.WrapResource(entWorkloadMemberToProtoWorkloadMember(entity))
	}
}

func (is *InvStore) DeleteWorkloadMembers(
	ctx context.Context, tenantID string, _ bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]
	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			collection, err := tx.WorkloadMember.Query().Where(workloadmember.TenantID(tenantID)).All(ctx)
			if err != nil {
				return err
			}
			if _, err := tx.WorkloadMember.Delete().Where(workloadmember.TenantID(tenantID)).Exec(ctx); err != nil {
				return err
			}
			for _, element := range collection {
				res, err := util.WrapResource(entWorkloadMemberToProtoWorkloadMember(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(HARD, res))
			}
			return nil
		})
	return deleted, txErr
}

func filterWorkloadMembers(ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter) (
	[]*ent.WorkloadMember, int, error,
) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_WORKLOAD_MEMBER, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[workloadmember.OrderOption](filter.GetOrderBy(), workloadmember.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.WorkloadMember.Query().
		Where(pred).
		Order(orderOpts...).
		WithWorkload().
		WithInstance().
		Offset(offset)

	// Limits number of query results if existent
	if limit != 0 {
		query = query.Limit(limit)
	}

	memberList, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.WorkloadMember.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	return memberList, total, nil
}

func (is *InvStore) ListWorkloadMember(
	ctx context.Context, filter *inv_v1.ResourceFilter,
) ([]*inv_v1.GetResourceResponse, int, error) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.WorkloadMember, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.WorkloadMember, *int, error) {
			resources, total, err := filterWorkloadMembers(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &resources, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.WorkloadMember, *inv_v1.GetResourceResponse](*resources,
		func(res *ent.WorkloadMember) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_WorkloadMember{
						WorkloadMember: entWorkloadMemberToProtoWorkloadMember(res),
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

func (is *InvStore) FilterWorkloadMember(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*cl.ResourceTenantIDCarrier, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.WorkloadMember, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]*ent.WorkloadMember, *int, error) {
			filtered, total, err := filterWorkloadMembers(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[*ent.WorkloadMember, *cl.ResourceTenantIDCarrier](
		*resources, func(c *ent.WorkloadMember) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.TenantID, ResourceId: c.ResourceID}
		})

	return ids, *total, err
}
