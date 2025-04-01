// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var log = logging.GetLogger("tc-utils")

func Retry(bo backoff.BackOff, fn func() error, msg string, args ...any) error {
	err := backoff.RetryNotify(
		fn,
		bo,
		func(err error, duration time.Duration) {
			log.Debug().Msgf("%s: %s, retrying after %s", fmt.Sprintf(msg, args...), err, duration)
		})
	if err != nil {
		log.Err(err).Msgf(msg, args...)
		return err
	}
	return nil
}

func RetryAndHandleError(bo backoff.BackOff, fn func() error, onErrorFn func(err error) error, msg string, args ...any) error {
	err := backoff.RetryNotify(
		fn,
		bo,
		func(err error, duration time.Duration) {
			log.Debug().Msgf("%s: %s, retrying after %s", fmt.Sprintf(msg, args...), err, duration)
		})
	if err != nil {
		log.Err(err).Msgf(msg, args...)
		return onErrorFn(err)
	}
	return nil
}
