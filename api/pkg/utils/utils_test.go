// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"context"
	"testing"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/stretchr/testify/assert"

	"github.com/open-edge-platform/infra-core/api/pkg/utils"
)

const (
	sampleJWT = "weljrhSUfljbfAbfjk.ADFJgfbkafgLGFLFVvFvFLVVlv"
	authKey   = "authorization"
)

func TestAppendJWTtoContext(t *testing.T) {
	// creating context and adding a JWT
	ctx := context.WithValue(context.Background(), authKey, sampleJWT)
	jwt, ok := ctx.Value(authKey).(string)
	assert.True(t, ok)
	assert.Equal(t, jwt, sampleJWT)

	// appending to outgoing context
	updCtx, err := utils.AppendJWTtoContext(ctx)
	assert.Equal(t, nil, err)

	// retrieving back a JWT
	jwtStr := metautils.ExtractOutgoing(updCtx).Get(string(authKey))
	assert.Equal(t, sampleJWT, jwtStr)

	// Test AppendJWTtoContext when context has no JWT
	ctx = context.Background()
	_, err = utils.AppendJWTtoContext(ctx)
	assert.Equal(t, nil, err)
}
