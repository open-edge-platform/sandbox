// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package flags

import "flag"

const (
	ServerAddress            = "serverAddress"
	ServerAddressDescription = "The endpoint address of this component to serve on. " +
		"It should have the following format <IP address>:<port>."
	EnableAuditing            = "enableAuditing"
	EnableAuditingDescription = "Flag to enable audit logs for API calls."
)

var FlagDisableCredentialsManagement = flag.Bool("disableCredentialsManagement", false,
	"Disables credentials management for edge nodes. Should only be used for testing")
