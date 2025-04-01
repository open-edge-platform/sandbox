// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

// ParsePrometheusTextMetrics parses the raw metrics exported by a Prometheus exporter
// into a map struct.
func ParsePrometheusTextMetrics(metrics string) (map[string][]map[string]map[string]any, error) {
	parser := &expfmt.TextParser{}
	families, err := parser.TextToMetricFamilies(strings.NewReader(metrics))
	if err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	out := make(map[string][]map[string]map[string]any)

	for key, val := range families {
		family := out[key]

		for _, m := range val.GetMetric() {
			metric := make(map[string]any)
			for _, label := range m.GetLabel() {
				metric[label.GetName()] = label.GetValue()
			}
			switch val.GetType() {
			case dto.MetricType_COUNTER:
				metric["value"] = m.GetCounter().GetValue()
			case dto.MetricType_GAUGE:
				metric["value"] = m.GetGauge().GetValue()
			default:
				// fmt.Printf("unsupported type: %v", val.GetType())
			}
			family = append(family, map[string]map[string]any{
				val.GetName(): metric,
			})
		}

		out[key] = family
	}
	return out, nil
}

// CheckMaps verifies if a map (out) contains the keys and values
// of another map (in).
func CheckMaps(in map[string]string, out map[string]any) bool {
	for inK, inV := range in {
		outV, ok := out[inK]
		if !ok {
			return false
		}
		if outV != inV {
			return false
		}
	}
	return true
}

// ValidateMetrics verifies if the output of ParsePrometheusTextMetrics
// contains a metricName, with the given map of labels, and a specific value.
func ValidateMetrics(
	output map[string][]map[string]map[string]any,
	metricName string, labels map[string]string, value float64,
) bool {
	for outputName, outputMetrics := range output {
		if outputName == metricName {
			for _, outputMetric := range outputMetrics {
				metric := outputMetric[metricName]
				hasSameLabels := CheckMaps(labels, metric)
				metricValue := metric["value"]
				if hasSameLabels && metricValue == value {
					return true
				}
			}
		}
	}
	return false
}
