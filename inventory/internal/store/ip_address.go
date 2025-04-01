// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"net/netip"

	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/ipaddressresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/utils/booleans"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	network_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/network/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

// ip_address.go  store logic for ipaddress

var ipAddressResourceCreationValidators = []resourceValidator[*network_v1.IPAddressResource]{
	protoValidator[*network_v1.IPAddressResource],
	validateIPAddressInput,
	doNotAcceptResourceID[*network_v1.IPAddressResource],
}

// enums mapping.
func IPAddressEnumsMap(fname string, eint int32) (ent.Value, error) {
	switch fname {
	case ipaddressresource.FieldDesiredState:
		return ipaddressresource.DesiredState(network_v1.IPAddressState_name[eint]), nil
	case ipaddressresource.FieldCurrentState:
		return ipaddressresource.CurrentState(network_v1.IPAddressState_name[eint]), nil
	case ipaddressresource.FieldConfigMethod:
		return ipaddressresource.ConfigMethod(network_v1.IPAddressConfigMethod_name[eint]), nil
	case ipaddressresource.FieldStatus:
		return ipaddressresource.Status(network_v1.IPAddressStatus_name[eint]), nil
	default:
		zlog.InfraSec().InfraError("unknown Enum field %s", fname).Msg("")
		return nil, errors.Errorfc(codes.InvalidArgument, "unknown Enum field %s", fname)
	}
}

func (is *InvStore) CreateIPAddress(ctx context.Context, in *network_v1.IPAddressResource) (*inv_v1.Resource, error) {
	if err := validate(in, ipAddressResourceCreationValidators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, ipAddressResourceCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("IpAddress Created: %s, %s", res.GetIpaddress().GetResourceId(), res)

	return res, nil
}

func ipAddressResourceCreator(in *network_v1.IPAddressResource) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_IPADDRESS)
		zlog.Debug().Msgf("CreateIPAddress: %s", id)

		newEntity := tx.IPAddressResource.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, IPAddressEnumsMap, nil); err != nil {
			return nil, err
		}

		// Look up the mandatory edges
		if err := setEdgeNicIDForMut(ctx, tx.Client(), mut, in.GetNic()); err != nil {
			return nil, err
		}

		// Set the resource_id field last.
		if err := mut.SetField(ipaddressresource.FieldResourceID, id); err != nil {
			return nil, errors.Wrap(err)
		}

		_, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, err := getIPAddressQuery(ctx, tx, id, false)
		if err != nil {
			return nil, err
		}
		return util.WrapResource(entIPAddressResourceToProtoIPAddressResource(res))
	}
}

func (is *InvStore) GetIPAddress(ctx context.Context, id string) (*inv_v1.Resource, error) {
	res, err := ExecuteInRoTxAndReturnSingle[ent.IPAddressResource](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.IPAddressResource, error) {
			return getIPAddressQuery(ctx, tx, id, true)
		})
	if err != nil {
		return nil, err
	}

	apiResource := entIPAddressResourceToProtoIPAddressResource(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_Ipaddress{Ipaddress: apiResource}}, nil
}

func getIPAddressQuery(ctx context.Context, tx *ent.Tx, resourceID string, loadNested bool) (*ent.IPAddressResource, error) {
	query := tx.IPAddressResource.Query().
		Where(ipaddressresource.ResourceID(resourceID))
	if loadNested {
		query.WithNic(func(hnq *ent.HostnicResourceQuery) {
			hnq.WithHost(func(hq *ent.HostResourceQuery) { // Populate the host of each nic
				hq.WithSite() // Populate the site of each host
			})
		})
	} else {
		query.WithNic()
	}
	entity, err := query.Only(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return entity, nil
}

//nolint:cyclop // high cyclomatic complexity due to hard-delete.
func (is *InvStore) UpdateIPAddress(
	ctx context.Context,
	id string,
	in *network_v1.IPAddressResource,
	fieldmask *fieldmaskpb.FieldMask,
) (*inv_v1.Resource, bool, error) {
	// validate input
	if err := validateIPAddressInput(in); err != nil {
		return nil, false, err
	}

	zlog.Debug().Msgf("UpdateIPAddress (%s): %v, fm: %v", id, in, fieldmask)

	res, hardDelete, err := ExecuteInTxAndReturnDouble[inv_v1.Resource, bool](is)(ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, *bool, error) {
			entity, err := tx.IPAddressResource.Query().
				Where(ipaddressresource.ResourceID(id)).
				Only(ctx)
			if err != nil {
				return nil, booleans.Pointer(false), errors.Wrap(err)
			}

			// hard delete - remove
			if isIPAddressHardDelete(fieldmask, entity, in) {
				zlog.Debug().Msgf("UpdateIPAddress Hard Delete: %s", id)

				// should be nil on success
				err = tx.IPAddressResource.DeleteOneID(entity.ID).Exec(ctx)
				if err != nil {
					return nil, booleans.Pointer(false), errors.Wrap(err)
				}

				var wrapped *inv_v1.Resource
				// Set current state to be consistent on the returned value on events and upon update.
				entity.CurrentState = ipaddressresource.CurrentStateIP_ADDRESS_STATE_DELETED
				wrapped, err = util.WrapResource(entIPAddressResourceToProtoIPAddressResource(entity))
				if err != nil {
					return nil, booleans.Pointer(false), err
				}
				return wrapped, booleans.Pointer(true), err
			}

			updateBuilder := tx.IPAddressResource.UpdateOneID(entity.ID)
			mut := updateBuilder.Mutation()

			// Look up the (new) referenced edge for this ipaddress.
			mut.ResetNic()
			err = setEdgeNicIDForMut(ctx, tx.Client(), mut, in.GetNic())
			if err != nil {
				return nil, booleans.Pointer(false), err
			}

			err = buildEntMutate(in, mut, IPAddressEnumsMap, fieldmask.GetPaths())
			if err != nil {
				return nil, booleans.Pointer(false), err
			}

			_, err = updateBuilder.Save(ctx)
			if err != nil {
				return nil, booleans.Pointer(false), errors.Wrap(err)
			}

			res, err := getIPAddressQuery(ctx, tx, id, false)
			if err != nil {
				return nil, booleans.Pointer(false), err
			}
			toBeReturned, err := util.WrapResource(entIPAddressResourceToProtoIPAddressResource(res))

			return toBeReturned, booleans.Pointer(false), errors.Wrap(err)
		},
	)
	if err != nil {
		return nil, false, err
	}

	return res, *hardDelete, err
}

func (is *InvStore) DeleteIPAddress(_ context.Context, _ string) (*inv_v1.Resource, bool, error) {
	return nil, false, errors.Errorfc(codes.Unimplemented, "IPAddress softDelete not supported")
}

func (is *InvStore) DeleteIPAddresses(
	ctx context.Context, tenantID string, _ bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]
	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			all, err := tx.IPAddressResource.Query().Where(ipaddressresource.TenantID(tenantID)).All(ctx)
			if err != nil {
				return err
			}
			if _, err := tx.IPAddressResource.Delete().Where(ipaddressresource.TenantID(tenantID)).Exec(ctx); err != nil {
				return err
			}
			for _, element := range all {
				res, err := util.WrapResource(entIPAddressResourceToProtoIPAddressResource(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(HARD, res))
			}
			return nil
		})
	return deleted, txErr
}

func filterIPAddresses(
	ctx context.Context,
	client *ent.Client, filter *inv_v1.ResourceFilter,
) ([]*ent.IPAddressResource, int, error) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_IPADDRESS, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[ipaddressresource.OrderOption](filter.GetOrderBy(), ipaddressresource.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.IPAddressResource.Query().
		Where(pred).
		Order(orderOpts...).
		WithNic(func(hnq *ent.HostnicResourceQuery) {
			hnq.WithHost(func(hq *ent.HostResourceQuery) { // Populate the host of each nic
				hq.WithSite() // Populate the site of each host
			})
		}).
		Offset(offset)

	// Limits number of query results if existent
	if limit != 0 {
		query = query.Limit(limit)
	}
	ipsList, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.IPAddressResource.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	return ipsList, total, nil
}

func (is *InvStore) ListIPAddress(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*inv_v1.GetResourceResponse, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.IPAddressResource, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.IPAddressResource, *int, error) {
			filtered, total, err := filterIPAddresses(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.IPAddressResource, *inv_v1.GetResourceResponse](*resources,
		func(res *ent.IPAddressResource) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Ipaddress{
						Ipaddress: entIPAddressResourceToProtoIPAddressResource(res),
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

func (is *InvStore) FilterIPAddress(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*cl.ResourceTenantIDCarrier, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.IPAddressResource, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]*ent.IPAddressResource, *int, error) {
			filtered, total, err := filterIPAddresses(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[*ent.IPAddressResource, *cl.ResourceTenantIDCarrier](
		*resources, func(c *ent.IPAddressResource) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.TenantID, ResourceId: c.ResourceID}
		})

	return ids, *total, err
}

func validateIPAddressInput(in *network_v1.IPAddressResource) error {
	_, err := netip.ParsePrefix(in.Address)
	if err != nil && in.Address != "" {
		zlog.InfraSec().InfraError("%s is not a valid CIDR IPAddress", in.Address).Msg("")
		return errors.Errorfc(codes.InvalidArgument, "%s is not a valid CIDR IPAddress", in.Address)
	}
	return nil
}

func isIPAddressHardDelete(
	fieldmask *fieldmaskpb.FieldMask, ipaddressq *ent.IPAddressResource, in *network_v1.IPAddressResource,
) bool {
	// Discovered addresses do not have the desired state set. Nullable enum are retrieved as ""
	return slices.Contains(fieldmask.GetPaths(), ipaddressresource.FieldCurrentState) &&
		((ipaddressq.DesiredState == ipaddressresource.DesiredStateIP_ADDRESS_STATE_DELETED &&
			in.CurrentState == network_v1.IPAddressState_IP_ADDRESS_STATE_DELETED) ||
			(ipaddressq.DesiredState == "" &&
				in.CurrentState == network_v1.IPAddressState_IP_ADDRESS_STATE_DELETED))
}
