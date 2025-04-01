// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"reflect"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"

	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	tenantv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/tenant/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/tenant"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

const tenantIDFieldName = "TenantId"

var (
	fallbackDefaultTenantID = "00000000-0000-0000-0000-000000000000"
	errMissingTenantID      = errors.Errorfc(codes.InvalidArgument, "received request doesn't specify tenant ID")
)

// TenantContextExtractingInterceptor - intercept tenantID provided by incoming gRPC request ctx,
// and puts it into the message requests (setting TenantId field) and resources (in case of Create/Update).
//
//nolint:cyclop // complexity is 15
func TenantContextExtractingInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		tenantID, exists := tenant.GetTenantIDFromContext(ctx)
		if !exists {
			return errMissingTenantID
		}
		zlog.Debug().Msgf("TenantContextExtractingInterceptor: tenantID: %s", tenantID)
		switch v := req.(type) {
		case *inv_v1.CreateResourceRequest:
			v.TenantId = tenantID

			resource, err := util.UnwrapResource[proto.Message](v.GetResource())
			if err != nil {
				return err
			}

			err = setTenantIDField(tenantID, resource)
			if err != nil {
				return err
			}
		case *inv_v1.UpdateResourceRequest:
			v.TenantId = tenantID

			resource, err := util.UnwrapResource[proto.Message](v.GetResource())
			if err != nil {
				return err
			}

			err = setTenantIDField(tenantID, resource)
			if err != nil {
				return err
			}
		case *inv_v1.FindResourcesRequest:
			cfgFilterTenantID(tenantID, v.GetFilter())
		case *inv_v1.ListResourcesRequest:
			cfgFilterTenantID(tenantID, v.GetFilter())
		case *inv_v1.ListInheritedTelemetryProfilesRequest:
			v.TenantId = tenantID
		case *inv_v1.GetResourceRequest:
			v.TenantId = tenantID
		case *inv_v1.GetTreeHierarchyRequest:
			v.TenantId = tenantID
		case *inv_v1.GetSitesPerRegionRequest:
			v.TenantId = tenantID
		case *inv_v1.DeleteResourceRequest:
			v.TenantId = tenantID
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// TenantContextInsertingInterceptorTestingOnly - adds a fake tenant ID to the ctx.
// Must be used for testing purposes only.
func TenantContextInsertingInterceptorTestingOnly() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx = tenant.AddTenantIDToContext(ctx, fallbackDefaultTenantID)
		zlog.Debug().Msgf("TenantContextInsertingInterceptorTestingOnly: tenantID: %s", fallbackDefaultTenantID)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// setTenantIDField sets the "tenantIDFieldName" field in the given proto.Message to the provided value.
// It assumes that the "tenantIDFieldName" field is of type string.
func setTenantIDField(tenantID string, msg proto.Message) error {
	// Check if the message is nil
	if msg == nil {
		return fmt.Errorf("the proto message is nil")
	}

	// Get the reflection object for the proto message.
	msgValue := reflect.ValueOf(msg).Elem()

	// Get the "tenantIDFieldName" field from the message.
	tenantIDField := msgValue.FieldByName(tenantIDFieldName)
	if !tenantIDField.IsValid() {
		return fmt.Errorf("the proto message does not have a '%s' field", tenantIDFieldName)
	}

	// Check if the "tenantIDFieldName" field is of type string.
	if tenantIDField.Kind() != reflect.String {
		return fmt.Errorf("the '%s' field is not of type string", tenantIDFieldName)
	}

	// Set the value of the "tenantIDFieldName" field to the provided string.
	tenantIDField.SetString(tenantID)
	return nil
}

// cfgFilterTenantID adds a filter `tenant_id = ...` to the filter of the
// provided ResourceFilter. It checks if the filter can be concatenated or not.
func cfgFilterTenantID(tenantID string, resFilter *inv_v1.ResourceFilter) {
	// Formats the filter by tenant string using the tenant_id field
	filterByTenant := fmt.Sprintf(`%s = %q`, tenantv1.TenantFieldTenantId, tenantID)

	// Gets the filter string and concatenates or sets it.
	filter := resFilter.GetFilter()
	if filter != "" {
		newFilter := filterByTenant + " AND (" + filter + ")"
		resFilter.Filter = newFilter
	} else {
		resFilter.Filter = filterByTenant
	}
}
