// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package providerconfiguration

//nolint:tagliatelle // Renaming the json keys may effect while unmarshalling/marshaling.
type ProviderConfig struct {
	DefaultOs               string `json:"defaultOs"`
	AutoProvision           bool   `json:"autoProvision"`
	DefaultLocalAccount     string `json:"defaultLocalAccount"`
	OSSecurityFeatureEnable bool   `json:"osSecurityFeatureEnable"`
}

type LOCAProviderConfig struct {
	InstanceTpl string `json:"instance_tpl"`
	DNSDomain   string `json:"dns_domain"`
}
