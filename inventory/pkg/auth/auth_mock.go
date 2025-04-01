// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//

package auth

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"

	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/mocks"
)

//nolint:revive // auth.AuthServiceMockFactory is fine to be used by other packages (contains naming of the structure)
func AuthServiceMockFactory(
	t *testing.T,
	createShouldFail,
	getShouldFail,
	revokeShouldFail bool,
) func(ctx context.Context) (AuthService, error) {
	t.Helper()
	authMock := mocks.NewMockAuthService(gomock.NewController(t))

	if createShouldFail {
		authMock.EXPECT().CreateCredentialsWithUUID(gomock.Any(), gomock.Any(), gomock.Any()).
			Return("", "", inv_errors.Errorf("")).AnyTimes()
	} else {
		authMock.EXPECT().CreateCredentialsWithUUID(gomock.Any(), gomock.Any(), gomock.Any()).
			Return("", "", nil).AnyTimes()
	}

	if getShouldFail {
		authMock.EXPECT().GetCredentialsByUUID(gomock.Any(), gomock.Any(), gomock.Any()).
			Return("", "", inv_errors.Errorf("")).AnyTimes()
	} else {
		authMock.EXPECT().GetCredentialsByUUID(gomock.Any(), gomock.Any(), gomock.Any()).
			Return("", "", nil).AnyTimes()
	}

	if revokeShouldFail {
		authMock.EXPECT().RevokeCredentialsByUUID(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(inv_errors.Errorf("")).AnyTimes()
	} else {
		authMock.EXPECT().RevokeCredentialsByUUID(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil).AnyTimes()
	}

	authMock.EXPECT().Logout(gomock.Any()).Return().AnyTimes()

	return func(ctx context.Context) (AuthService, error) {
		return authMock, nil
	}
}
