// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package info

import "fmt"

var (
	version string
	commit  string
)

var (
	OHBulkImport = "Host Bulk Import Tool"
	OHPreflight  = "Host Pre-flight Tool"
	Version      = fmt.Sprintf("%s-%v", version, commit)
)
