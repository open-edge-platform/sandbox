// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/api/internal/server"
	"github.com/open-edge-platform/infra-core/api/test/utils"
)

func TestOptions_UnicodeChecker(t *testing.T) {
	httpCtx := context.Background()

	// Tests UnicodePrintableCharsCheckerMiddleware no error
	h := server.UnicodePrintableCharsCheckerMiddleware()(func(c echo.Context) error {
		c.Response().Write([]byte("test"))
		return nil
	})

	dataJSON := `{"name":"Jon Snow"}`
	r, err := http.NewRequestWithContext(httpCtx, http.MethodPost, "test", strings.NewReader(dataJSON))
	assert.NoError(t, err)
	w := testResponseWriter{name: "test"}
	ctx := echo.New().NewContext(r, w)
	err = h(ctx)
	require.NoError(t, err)

	// Tests UnicodePrintableCharsCheckerMiddleware error
	dataJSONError := utils.HostNameNonPrintable
	r, err = http.NewRequestWithContext(httpCtx, http.MethodPost, "test", strings.NewReader(dataJSONError))
	assert.NoError(t, err)
	ctx = echo.New().NewContext(r, w)
	err = h(ctx)
	require.Error(t, err)

	// Tests UnicodePrintableCharsCheckerMiddleware without body
	r, err = http.NewRequestWithContext(httpCtx, http.MethodPost, "test", http.NoBody)
	assert.NoError(t, err)
	ctx = echo.New().NewContext(r, w)
	err = h(ctx)
	require.NoError(t, err)
}
