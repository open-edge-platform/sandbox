// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//go:build tools

package tools

// Blank imports to just include them in the mod file, these are required by ent gen
import (
	_ "github.com/mattn/go-runewidth"
	_ "github.com/olekukonko/tablewriter"
	_ "github.com/spf13/cobra"
	_ "golang.org/x/tools/cmd/goimports"
)
