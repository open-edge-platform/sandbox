// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"time"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/flags"
)

const (
	defaultTimeout = 3 * time.Second
)

// AuthService implements the authorization service to create or revoke EN credentials.
// Remember to call auth.Init() at the very beginning.
//
//go:generate mockgen -package mocks -destination=../mocks/auth_mock.go . AuthService
type AuthService interface { //nolint:revive // Need this interface name for more readable.
	// CreateCredentialsWithUUID creates EN credentials based on UUID.
	// The credentials can be further used by edge node agents.
	CreateCredentialsWithUUID(ctx context.Context, tenantID, uuid string) (string, string, error)
	// GetCredentialsByUUID obtains EN credentials based on UUID.
	GetCredentialsByUUID(ctx context.Context, tenantID, uuid string) (string, string, error)
	// RevokeCredentialsByUUID revokes EN credentials based on UUID.
	RevokeCredentialsByUUID(ctx context.Context, tenantID, uuid string) error

	// Logout closes the session with authorization service.
	// Should always be invoked after all operations in a session are done.
	Logout(ctx context.Context)
}

// Init bootstraps the auth service library. Must be called after secretprovider.Init().
func Init() error {
	if *flags.FlagDisableCredentialsManagement {
		zlog.Warn().Msgf("disableCredentialsManagement flag is set to true, " +
			"skip auth initialization")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Note that this function only creates auth service and logs out immediately.
	// The assumption is that AuthServiceFactory will perform all necessary initializations.
	authService, err := AuthServiceFactory(ctx)
	if err != nil {
		return err
	}
	defer authService.Logout(ctx)

	return nil
}
