// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package filters

import (
	"fmt"
	"strings"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
)

const singleSpace = " "

type Clause func() string

func ValEq(v1 string, v2 any) Clause {
	return func() string {
		switch v := v2.(type) {
		case string:
			return fmt.Sprintf("%s = %q", v1, v)
		default:
			return fmt.Sprintf("%s = %v", v1, v)
		}
	}
}

func ValNotEq(v1 string, v2 any) Clause {
	return func() string {
		switch v := v2.(type) {
		case string:
			return fmt.Sprintf("%s != %q", v1, v)
		default:
			return fmt.Sprintf("%s != %v", v1, v)
		}
	}
}

func ValDotValEq(v1, v2, v3 string) Clause {
	return func() string {
		return fmt.Sprintf("%s.%s = %q", v1, v2, v3)
	}
}

func ValDotValDotValEq(v1, v2, v3, v4 string) Clause {
	return func() string {
		return fmt.Sprintf("%s.%s.%s = %q", v1, v2, v3, v4)
	}
}

func NotHas(v string) Clause {
	return func() string {
		return fmt.Sprintf("NOT has(%s)", v)
	}
}

func NewBuilderWith(c Clause) IAndOr {
	return new(builder).startWith(c)
}

func NewBuilder() IAndOr {
	return new(builder)
}

type IAndOr interface {
	And(c Clause) IAndOr
	Or(c Clause) IAndOr
	IBuild
}

type IBuild interface {
	Build() string
}

func and() string {
	return "AND"
}

func or() string {
	return "OR"
}

type builder struct {
	clauses []Clause
}

func (b *builder) Build() string {
	return strings.Join(
		collections.MapSlice[Clause, string](
			b.clauses,
			func(c Clause) string {
				return c()
			},
		),
		singleSpace,
	)
}

func (b *builder) startWith(c Clause) IAndOr {
	b.clauses = append(b.clauses, c)
	return b
}

func (b *builder) And(c Clause) IAndOr {
	if len(b.clauses) > 0 {
		b.clauses = append(b.clauses, and, c)
	} else {
		b.clauses = append(b.clauses, c)
	}
	return b
}

func (b *builder) Or(c Clause) IAndOr {
	if len(b.clauses) > 0 {
		b.clauses = append(b.clauses, or, c)
	} else {
		b.clauses = append(b.clauses, c)
	}
	return b
}
