// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package rego

import (
	embed "embed"
)

//go:embed *.rego
var RegoFolder embed.FS
