// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"

	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/workloadresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/utils/booleans"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

// workload.go  store information for workload

var workloadResourceCreationValidators = []resourceValidator[*computev1.WorkloadResource]{
	protoValidator[*computev1.WorkloadResource],
	doNotAcceptResourceID[*computev1.WorkloadResource],
}

// enum status mapping.
func WorkloadEnumStatusMap(fname string, eint int32) (ent.Value, error) {
	switch fname {
	case workloadresource.FieldDesiredState:
		return workloadresource.DesiredState(computev1.WorkloadState_name[eint]), nil
	case workloadresource.FieldCurrentState:
		return workloadresource.CurrentState(computev1.WorkloadState_name[eint]), nil
	case workloadresource.FieldKind:
		return workloadresource.Kind(computev1.WorkloadKind_name[eint]), nil
	default:
		zlog.InfraSec().InfraError("unknown Enum field %s", fname).Msg("")
		return nil, errors.Errorfc(codes.InvalidArgument, "unknown Enum field %s", fname)
	}
}

func (is *InvStore) CreateWorkload(ctx context.Context, in *computev1.WorkloadResource) (*inv_v1.Resource, error) {
	if err := validate(in, workloadResourceCreationValidators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, workloadResourceCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("Workload Created: %s, %s", res.GetWorkload().GetResourceId(), res)
	return res, nil
}

func workloadResourceCreator(in *computev1.WorkloadResource) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_WORKLOAD)
		zlog.Debug().Msgf("CreateWorkload: %s", id)

		newEntity := tx.WorkloadResource.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, WorkloadEnumStatusMap, nil); err != nil {
			return nil, err
		}

		if err := mut.SetField(workloadresource.FieldResourceID, id); err != nil {
			return nil, errors.Wrap(err)
		}

		_, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, err := getWorkloadQuery(ctx, tx, id, false)
		if err != nil {
			return nil, err
		}
		return util.WrapResource(entWorkloadResourceToProtoWorkloadResource(res))
	}
}

func (is *InvStore) GetWorkload(ctx context.Context, id string) (*inv_v1.Resource, error) {
	res, err := ExecuteInRoTxAndReturnSingle[ent.WorkloadResource](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.WorkloadResource, error) {
			return getWorkloadQuery(ctx, tx, id, true)
		})
	if err != nil {
		return nil, err
	}

	apiResource := entWorkloadResourceToProtoWorkloadResource(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_Workload{Workload: apiResource}}, nil
}

func getWorkloadQuery(ctx context.Context, tx *ent.Tx, resourceID string, nestedLoad bool) (*ent.WorkloadResource, error) {
	query := tx.WorkloadResource.Query().
		Where(workloadresource.ResourceID(resourceID))
	if nestedLoad {
		query.WithMembers(func(q *ent.WorkloadMemberQuery) {
			q.WithInstance() // Populate the instance of each member
		})
	} else {
		query.WithMembers()
	}
	entity, err := query.Only(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return entity, nil
}

func (is *InvStore) UpdateWorkload(
	ctx context.Context, id string, in *computev1.WorkloadResource, fieldmask *fieldmaskpb.FieldMask,
) (*inv_v1.Resource, bool, error) {
	zlog.Debug().Msgf("UpdateWorkload (%s): %v, fm: %v", id, in, fieldmask)

	res, hardDelete, err := ExecuteInTxAndReturnDouble[inv_v1.Resource, bool](is)(ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, *bool, error) {
			entity, err := tx.WorkloadResource.Query().
				Where(workloadresource.ResourceID(id)).
				WithMembers().
				Only(ctx)
			if err != nil {
				return nil, booleans.Pointer(false), errors.Wrap(err)
			}

			// hard delete - if both Desired and Current state are Deleted, remove
			if isWorkloadHardDelete(fieldmask, entity, in) {
				zlog.Debug().Msgf("UpdateWorkload Hard Delete: %s", id)

				err = deleteWorkloadWithConstraints(ctx, tx.Client(), entity.ID, entity.Edges.Members)
				if err != nil {
					return nil, booleans.Pointer(false), err
				}

				var wrapped *inv_v1.Resource
				// Set current state to be consistent on the returned value on events and upon update.
				entity.CurrentState = workloadresource.CurrentStateWORKLOAD_STATE_DELETED
				wrapped, err = util.WrapResource(entWorkloadResourceToProtoWorkloadResource(entity))
				if err != nil {
					return nil, booleans.Pointer(false), err
				}
				return wrapped, booleans.Pointer(true), nil
			}

			updateBuilder := tx.WorkloadResource.UpdateOneID(entity.ID)
			mut := updateBuilder.Mutation()

			err = buildEntMutate(in, mut, WorkloadEnumStatusMap, fieldmask.GetPaths())
			if err != nil {
				return nil, booleans.Pointer(false), err
			}

			_, err = updateBuilder.Save(ctx)
			if err != nil {
				return nil, booleans.Pointer(false), errors.Wrap(err)
			}

			res, err := getWorkloadQuery(ctx, tx, id, false)
			if err != nil {
				return nil, booleans.Pointer(false), err
			}
			toBeReturned, err := util.WrapResource(entWorkloadResourceToProtoWorkloadResource(res))
			return toBeReturned, booleans.Pointer(false), errors.Wrap(err)
		},
	)
	if err != nil {
		return nil, false, err
	}

	return res, *hardDelete, err
}

func (is *InvStore) DeleteWorkload(ctx context.Context, id string) (*inv_v1.Resource, bool, error) {
	zlog.Debug().Msgf("DeleteWorkload Soft Delete: %s", id)

	res, softDelete, err := ExecuteInTxAndReturnDouble[inv_v1.Resource, bool](is)(ctx, deleteWorkload(id))
	if err != nil {
		return nil, false, err
	}

	return res, *softDelete, err
}

func deleteWorkload(id string) func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, *bool, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, *bool, error) {
		entity, err := tx.WorkloadResource.Query().
			Where(workloadresource.ResourceID(id)).
			WithMembers().
			Only(ctx)
		if err != nil {
			return nil, booleans.Pointer(false), errors.Wrap(err)
		}

		softDelete := true
		switch entity.Kind {
		case workloadresource.KindWORKLOAD_KIND_CLUSTER:
			err = deleteWorkloadWithConstraints(ctx, tx.Client(), entity.ID, entity.Edges.Members)
			// this is "Hard Delete" directly deleting the resource
			softDelete = false
		default:
			// this is a "Soft Delete" - it only sets the Desired State to Deleted
			// Hard delete happens in Update, when both Desired and Current state are
			// both Deleted.
			_, err = tx.WorkloadResource.UpdateOneID(entity.ID).
				SetDesiredState(workloadresource.DesiredStateWORKLOAD_STATE_DELETED).
				Save(ctx)
			if err != nil {
				return nil, booleans.Pointer(false), errors.Wrap(err)
			}
			entity, err = getWorkloadQuery(ctx, tx, id, false)
		}
		if err != nil {
			return nil, booleans.Pointer(false), errors.Wrap(err)
		}

		wrapped, err := util.WrapResource(entWorkloadResourceToProtoWorkloadResource(entity))
		if err != nil {
			return nil, booleans.Pointer(false), err
		}

		return wrapped, &softDelete, nil
	}
}

func (is *InvStore) DeleteWorkloads(
	ctx context.Context, tenantID string, enforce bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]

	deletionStrategies := map[bool]func(ctx context.Context, tx *ent.Tx, tenantID string) (DeletionKind, int, error){
		true: func(ctx context.Context, tx *ent.Tx, tenantID string) (DeletionKind, int, error) {
			i, e := tx.WorkloadResource.Delete().Where(workloadresource.TenantID(tenantID)).Exec(ctx)
			return HARD, i, e
		},
		false: func(ctx context.Context, tx *ent.Tx, tenantID string) (DeletionKind, int, error) {
			i, e := tx.WorkloadResource.Update().
				Where(workloadresource.TenantID(tenantID)).
				SetDesiredState(workloadresource.DesiredStateWORKLOAD_STATE_DELETED).
				Save(ctx)
			return SOFT, i, e
		},
	}
	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			collection, err := tx.WorkloadResource.Query().Where(workloadresource.TenantID(tenantID)).All(ctx)
			if err != nil {
				return err
			}
			dk, noOfDeleted, err := deletionStrategies[enforce](ctx, tx, tenantID)
			if err != nil {
				return err
			}
			if noOfDeleted != len(collection) {
				return errors.Errorf(
					"Returned number of updated/delete workloads(%d) is different that number of retrieved hosts(%d)",
					noOfDeleted,
					len(collection))
			}
			if dk == SOFT {
				// because of performance reasons we do not want to fetch updated instance from DB,
				// and in the same time we want to have updated resource reported by the event.
				collections.ForEach(collection, func(i *ent.WorkloadResource) {
					i.DesiredState = workloadresource.DesiredStateWORKLOAD_STATE_DELETED
				})
			}
			for _, element := range collection {
				res, err := util.WrapResource(entWorkloadResourceToProtoWorkloadResource(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(dk, res))
			}

			return nil
		})
	return deleted, txErr
}

func filterWorkloads(ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter) (
	[]*ent.WorkloadResource, int, error,
) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_WORKLOAD, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[workloadresource.OrderOption](filter.GetOrderBy(), workloadresource.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.WorkloadResource.Query().
		Where(pred).
		WithMembers(func(q *ent.WorkloadMemberQuery) {
			q.WithInstance() // Populate the instance of each member
		}).
		Order(orderOpts...).
		Offset(offset)

	// Limits number of query results if existent
	if limit != 0 {
		query = query.Limit(limit)
	}

	workloadList, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.WorkloadResource.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	return workloadList, total, nil
}

func (is *InvStore) ListWorkload(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*inv_v1.GetResourceResponse, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.WorkloadResource, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.WorkloadResource, *int, error) {
			filtered, total, err := filterWorkloads(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.WorkloadResource, *inv_v1.GetResourceResponse](*resources,
		func(res *ent.WorkloadResource) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Workload{
						Workload: entWorkloadResourceToProtoWorkloadResource(res),
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

func (is *InvStore) FilterWorkload(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*cl.ResourceTenantIDCarrier, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.WorkloadResource, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]*ent.WorkloadResource, *int, error) {
			filtered, total, err := filterWorkloads(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[*ent.WorkloadResource, *cl.ResourceTenantIDCarrier](
		*resources, func(c *ent.WorkloadResource) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.TenantID, ResourceId: c.ResourceID}
		})

	return ids, *total, err
}

func getWorkloadIDFromResourceID(
	ctx context.Context,
	client *ent.Client,
	workloadRes *computev1.WorkloadResource,
) (int, error) {
	site, qerr := client.WorkloadResource.Query().
		Where(workloadresource.ResourceID(workloadRes.ResourceId)).
		Only(ctx)
	if qerr != nil {
		return 0, errors.Wrap(qerr)
	}
	return site.ID, nil
}

func deleteWorkloadWithConstraints(ctx context.Context, client *ent.Client, workloadID int, members []*ent.WorkloadMember) error {
	if len(members) != 0 {
		// The workload has members, we cannot delete
		zlog.InfraSec().InfraError("the workload has relations and cannot be deleted").Msg("")
		return errors.Errorfc(codes.FailedPrecondition, "the workload has relations and cannot be deleted")
	}

	// should be nil on success
	err := client.WorkloadResource.DeleteOneID(workloadID).Exec(ctx)
	if err != nil {
		return errors.Wrap(err)
	}
	return nil
}

func isWorkloadHardDelete(
	fieldmask *fieldmaskpb.FieldMask, workloadq *ent.WorkloadResource, in *computev1.WorkloadResource,
) bool {
	return slices.Contains(fieldmask.GetPaths(), workloadresource.FieldCurrentState) &&
		workloadq.DesiredState == workloadresource.DesiredStateWORKLOAD_STATE_DELETED &&
		in.CurrentState == computev1.WorkloadState_WORKLOAD_STATE_DELETED
}
