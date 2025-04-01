// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package secrets

import "context"

//go:generate mockgen -package mocks -destination=../mocks/secrets_mock.go . SecretsService
//nolint:revive // keep SecretsService name
type SecretsService interface {
	// ReadSecret reads a persistent secret under the given path and returns a stored object.
	// A consumer is responsible for parsing the returned object and converting it to an expected format.
	ReadSecret(ctx context.Context, path string) (map[string]interface{}, error)
	// WriteSecret write a persistent secret under the given path and returns the stored object.
	// A consumer is responsible for parsing the returned object and converting it to an expected format.
	WriteSecret(ctx context.Context, path string, secret map[string]interface{}) (map[string]interface{}, error)
	// Logout terminates a user session. Should be always invoked after all operations are done.
	Logout(ctx context.Context)
}

var SecretServiceFactory = newVaultService
