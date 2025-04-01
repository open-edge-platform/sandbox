// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/tenant-controller/internal/util"
)

func TestRetry_HappyPath(t *testing.T) {
	err := util.Retry(
		backoff.WithMaxRetries(
			backoff.NewConstantBackOff(time.Millisecond), 1,
		),
		func() error {
			return nil
		}, "")

	require.NoError(t, err)
}

func TestRetry_fnReturnsError(t *testing.T) {
	actualNumberOfRetries := uint64(0)
	requestedNumberOfRetries := uint64(5)
	expected := fmt.Errorf("error")
	err := util.Retry(
		backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Millisecond), requestedNumberOfRetries),
		func() error {
			actualNumberOfRetries++
			return expected
		}, "")

	require.ErrorIs(t, err, expected)
	require.Equal(t, requestedNumberOfRetries, actualNumberOfRetries-1) // all retries - first call
}

func TestRetryAndHandleError_fnReturnsError(t *testing.T) {
	actualNumberOfRetries := uint64(0)
	requestedNumberOfRetries := uint64(5)
	expected := fmt.Errorf("error")
	onErrorWasCalled := false
	err := util.RetryAndHandleError(
		backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Millisecond), requestedNumberOfRetries),
		func() error {
			actualNumberOfRetries++
			return expected
		},
		func(err error) error {
			onErrorWasCalled = true
			require.ErrorIs(t, err, expected)
			return nil
		}, "")

	require.NoError(t, err)
	require.Equal(t, requestedNumberOfRetries, actualNumberOfRetries-1) // all retries - first call
	require.True(t, onErrorWasCalled)
}

func TestRetryAndHandleError_HappyPath(t *testing.T) {
	err := util.RetryAndHandleError(
		backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Millisecond), 1),
		func() error { return nil },
		func(_ error) error {
			t.Fatal("onError function shall not be called")
			return nil
		}, "")

	require.NoError(t, err)
}
