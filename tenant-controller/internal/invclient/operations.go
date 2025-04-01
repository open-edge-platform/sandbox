// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invclient

import (
	"context"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	tenantv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/tenant/v1"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	inv_util "github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/filters"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
	"github.com/open-edge-platform/infra-core/tenant-controller/internal/util"
)

func (c *TCInventoryClient) CreateTenantResource(
	ctx context.Context, tenantID string,
) (*inv_v1.Resource, error) {
	return c.CreateResource(ctx, tenantID, &inv_v1.Resource{
		Resource: &inv_v1.Resource_Tenant{
			Tenant: &tenantv1.Tenant{
				TenantId:         tenantID,
				DesiredState:     tenantv1.TenantState_TENANT_STATE_CREATED,
				WatcherOsmanager: false,
			},
		},
	})
}

func (c *TCInventoryClient) GetTenantResource(ctx context.Context, tenantID string) (tid, rid string, err error) {
	log.Debug().Msgf("GetTenantResource: tenantID=%s", tenantID)
	tenant, err := c.GetTenantResourceInstance(ctx, tenantID)
	if err != nil {
		return "", "", err
	}
	tenantID, resourceID := tenant.GetTenantId(), tenant.GetResourceId()
	return tenantID, resourceID, nil
}

func (c *TCInventoryClient) GetTenantResourceInstance(ctx context.Context, tenantID string) (*tenantv1.Tenant, error) {
	log.Debug().Msgf("GetTenantResourceInstance: tenantID=%s", tenantID)
	findResp, err := c.List(ctx, &inv_v1.ResourceFilter{
		Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Tenant{}},
		Filter:   filters.NewBuilderWith(filters.ValEq("tenant_id", tenantID)).Build(),
	})
	if err != nil {
		return nil, err
	}

	err = inv_util.CheckListOutputIsSingular(findResp.GetResources())
	if err != nil {
		if !inv_errors.IsNotFound(err) {
			log.InfraSec().InfraErr(err).Msgf("Expected one Tenant, received multiple: %s", findResp)
		}
		return nil, err
	}

	if err = validator.ValidateMessage(findResp); err != nil {
		log.InfraSec().InfraErr(err).Msg("Tenant returned failed validation")
		return nil, inv_errors.Wrap(err)
	}

	tenant := findResp.GetResources()[0].GetResource().GetTenant()
	tenantID, resourceID := tenant.GetTenantId(), tenant.GetResourceId()
	log.Debug().Msgf("Found tenant: TenantRes[tenantID:%s, resourceID: %s]", tenantID, resourceID)
	return tenant, nil
}

func (c *TCInventoryClient) UpdateTenantResource(
	ctx context.Context, fm *fieldmaskpb.FieldMask, tenant *tenantv1.Tenant,
) (*inv_v1.Resource, error) {
	log.Debug().Msgf("UpdateTenantResource: Tenant=%s, FM: %v", tenant, fm)
	rsp, err := c.Update(
		ctx,
		tenant.GetTenantId(),
		tenant.GetResourceId(),
		fm,
		&inv_v1.Resource{Resource: &inv_v1.Resource_Tenant{Tenant: tenant}},
	)
	return rsp, err
}

func (c *TCInventoryClient) HardDeleteTenantResource(ctx context.Context, tenantID, resourceID string) error {
	log.Debug().Msgf("HardDeleteTenantResource: tenantID=%s, resourceID=%s", tenantID, resourceID)
	_, err := c.UpdateTenantResource(
		ctx,
		&fieldmaskpb.FieldMask{Paths: []string{tenantv1.TenantFieldCurrentState}},
		&tenantv1.Tenant{ResourceId: resourceID, TenantId: tenantID, CurrentState: tenantv1.TenantState_TENANT_STATE_DELETED},
	)
	return err
}

func (c *TCInventoryClient) DeleteTenantResource(
	ctx context.Context, tenantID string,
) error {
	log.Debug().Msgf("DeleteTenantResource: tenantID=%s", tenantID)
	_, rid, err := c.GetTenantResource(ctx, tenantID)
	if err != nil {
		log.Err(err).Msgf("Error while getting tenant by tenantID: tenantID=%s", tenantID)
		return err
	}
	_, err = c.Delete(ctx, tenantID, rid)
	if err != nil {
		log.Err(err).Msgf("Error while deleting tenant: tenantID=%s, resourceID=%s", tenantID, rid)
	}
	return err
}

func (c *TCInventoryClient) CreateResource(
	ctx context.Context, tenantID string, resource *inv_v1.Resource,
) (*inv_v1.Resource, error) {
	log.Debug().Msgf("CreateResource: tenantID=%s, resource=%s", tenantID, resource.String())
	if err := util.SetResourceValue(resource, "tenant_id", tenantID); err != nil {
		return nil, err
	}
	log.Debug().Msgf("CreateResource - tenant set: tenantID=%s, resource=%v", resource, tenantID)
	if err := validator.ValidateMessage(resource); err != nil {
		return nil, err
	}
	rsp, err := c.Create(ctx, tenantID, resource)
	if err != nil {
		log.Info().Msgf("error has occurred during resource creation: %v", err)
		return nil, err
	}
	return rsp, nil
}
