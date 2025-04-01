// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package configuration

import (
	"flag"
	"time"

	"github.com/cenkalti/backoff/v4"
)

const (
	AppName = "EdgeInfraTenantController"
)

var (
	//nolint:mnd // default value
	TenantTerminatorWaitUntilTickerInterval = flag.Duration(
		"tenantTerminatorWaitUntilTickerInterval", time.Second*10, "")

	//nolint:mnd // default value
	TenantTerminatorDefaultBackoffInitInterval = flag.Duration(
		"tenantTerminatorDefaultBackoffInitInterval", time.Second*2, "")

	//nolint:mnd // default value
	TenantTerminatorDefaultBackoffMultiplier = flag.Float64(
		"tenantTerminatorDefaultBackoffMultiplier", 1.5, "")

	//nolint:mnd // default value
	TenantTerminatorDefaultBackoffRandomizedFactor = flag.Float64(
		"tenantTerminatorDefaultBackoffRandomizedFactor", 0.1, "")

	//nolint:mnd // default value
	TenantTerminatorDefaultBackoffRetries = flag.Uint64(
		"tenantTerminatorDefaultBackoffRetries", 3, "")

	EnableHardDeletionFallback = flag.Bool(
		"enableHardDeletionFallbackForResourceManagedByRMs", false, "")

	ResourceSoftDeletionTimeout = flag.Duration(
		"resourceSoftDeletionTimeout", time.Minute, "")

	//nolint:mnd // default value
	WorkloadSoftDeletionTimeout = flag.Duration(
		"workloadSoftDeletionTimeout", time.Minute*5, "")

	DefaultBackoff = backoff.WithMaxRetries(
		backoff.NewExponentialBackOff(
			func(off *backoff.ExponentialBackOff) {
				off.InitialInterval = *TenantTerminatorDefaultBackoffInitInterval
				off.Multiplier = *TenantTerminatorDefaultBackoffMultiplier
				off.RandomizationFactor = *TenantTerminatorDefaultBackoffRandomizedFactor
			},
		), *TenantTerminatorDefaultBackoffRetries)

	//nolint:mnd //this is default value
	InvTimeout = flag.Duration("invTimeout", 10*time.Second,
		"default timeout (in seconds) for communication between TC and inventory")

	//nolint:mnd //this is default value
	DataModelTimeout = flag.Duration("dataModelTimeout", 5*time.Second,
		"default timeout (in seconds) for TC-DataModel communication")

	//nolint:mnd // default value
	TenantInitializationTimeout = flag.Duration(
		"tenantInitializationTimeout", time.Second*10, "")

	//nolint:mnd // default value
	TenantTerminationTimeout = flag.Duration(
		"tenantTerminationTimeout", time.Minute*10, "")
)
