// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"encoding/json"
	"fmt"
	"net/url"

	"golang.org/x/exp/slices"
)

// toMapKeys translates a struct into a map[string]interface{}
// and then extracts the keys of such map to return them
// as a slice of strings.
// It is useful to get all the names of the fields of an
// struct.
func toMapKeys(p interface{}) ([]string, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys, nil
}

// queryChecker validates if the keys contained in the url.Values
// parameter correspond to the list of fields contained in the
// expected list of strings parameter.
func queryChecker(values url.Values, expected []string) bool {
	if len(expected) > 0 {
		for k := range values {
			if !slices.Contains(expected, k) {
				return false
			}
		}
	}

	if len(expected) == 0 && len(values) > 0 {
		return false
	}

	return true
}

// ValidateQuery performs the validation of an expected set of fields
// contained in a query struct against the provided query parameters
// of an HTTP request.
// If the query expects values and those match the provided values,
// the validation returns nil, otherwise it resturns an error.
// It uses ToMapKeys function to get the names of the fields of a
// query struct, and then uses QueryChecker to validate those fields
// against the provided query values.
func ValidateQuery(values url.Values, query interface{}) error {
	expected := []string{}
	var err error

	if query != nil {
		expected, err = toMapKeys(query)
		if err != nil {
			log.InfraSec().InfraError("invalid query format").Msg("")
			return fmt.Errorf("invalid query format")
		}
	}

	if !queryChecker(values, expected) {
		log.InfraSec().InfraError("invalid query format").Msg("")
		return fmt.Errorf("invalid query format")
	}

	return nil
}
