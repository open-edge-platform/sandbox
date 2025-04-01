// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"os"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

const (
	envENLokiURL = "EN_LOKI_URL"
)

var ENLokiURL = os.Getenv(envENLokiURL)

var zlog = logging.GetLogger("Env")

func MustGetEnv(key string) string {
	v, found := os.LookupEnv(key)
	if found && v != "" {
		zlog.Debug().Msgf("Found env var %s = %s", key, v)
		return v
	}

	zlog.Fatal().Msgf("Mandatory env var %s is not set or empty!", key)
	return ""
}

func MustEnsureRequired() {
	ENLokiURL = MustGetEnv(envENLokiURL)
}
