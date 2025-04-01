// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tenant

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"google.golang.org/grpc/codes"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
)

// making it a constant to satisfy go-mnd linter.
const (
	TenantKey = "ActiveProjectID"
)

// extractProjectIDFromHeader extracts the tenant key from the
// HTTP header context and validates it against an uuid format.
func extractProjectIDFromHeader(c echo.Context) (string, error) {
	// If key is not found Get returns "". It is case insensitive
	projectID := c.Request().Header.Get(TenantKey)
	if projectID == "" {
		err := errors.Errorfc(
			codes.Unauthenticated,
			"key '%s' not found in HTTP header",
			TenantKey,
		)
		return "", err
	}

	_, err := uuid.Parse(projectID)
	if err != nil {
		err = errors.Errorfc(
			codes.Unauthenticated,
			"Rejected because failed to parse '%s' into uuid in HTTP header: rejected",
			TenantKey,
		)
		zlog.InfraSec().Err(err).Send()
		return "", err
	}
	return projectID, nil
}

func addTenantIDToEchoContext(c echo.Context, tenantID string) echo.Context {
	c.SetRequest(
		c.Request().WithContext(
			AddTenantIDToContext(c.Request().Context(), tenantID),
		),
	)
	return c
}

// TenantInterceptor returns an echo middleware to extract tenant id from HTTP header and provide it in the context.
// The middleware returns error only if the tenant id is not found, invalid or missing a JWT.
// This middleware should run after the AuthN middleware.
//
//nolint:revive // revice suggests name
func TenantInterceptor(next echo.HandlerFunc) echo.HandlerFunc {
	zlog.InfraSec().Debug().Msgf("TenantInterceptor is initialized")
	return func(c echo.Context) error {
		tenantID, err := extractProjectIDFromHeader(c)
		switch {
		case err != nil:
			err = errors.Errorfc(
				codes.Unauthenticated,
				"Rejected because failed to get '%s' in HTTP header: rejected",
				TenantKey,
			)
			zlog.InfraSec().Err(err).Send()
			return &echo.HTTPError{
				Code:    http.StatusUnauthorized,
				Message: http.StatusText(http.StatusUnauthorized),
			}
		case tenantID == "":
			err = errors.Errorfc(
				codes.Unauthenticated,
				"Rejected because missing '%s' projectID in HTTP header: rejected",
				TenantKey,
			)
			zlog.InfraSec().Err(err).Send()
			return &echo.HTTPError{
				Code:    http.StatusUnauthorized,
				Message: http.StatusText(http.StatusUnauthorized),
			}
		}
		// including tenantID to the message metadata
		c = addTenantIDToEchoContext(c, tenantID)
		zlog.Debug().Msgf("Request has authentication data, proceeding")
		return next(c)
	}
}
