// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package function

import "reflect"

const emptyNullCase = "null"

type Predicate[T any] func(v T) bool

func Not[T any](p Predicate[T]) Predicate[T] {
	return func(v T) bool {
		return !p(v)
	}
}

func IsNil(e any) bool {
	return e == nil || (reflect.ValueOf(e).Kind() == reflect.Ptr && reflect.ValueOf(e).IsNil())
}

func IsNotEmptyNullCase(v *string) bool {
	return v != nil && *v != emptyNullCase
}

func IsEmptyNullCase(v *string) bool {
	return v != nil && *v == emptyNullCase
}
