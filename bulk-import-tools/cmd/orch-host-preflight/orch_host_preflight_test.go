// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"os/exec"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

var preflightBinPath = os.Getenv("PREFLIGHT_BIN_PATH")

func TestGenerateCSVCommand(t *testing.T) {
	out, err := exec.Command(preflightBinPath, "generate", "test.csv").CombinedOutput()
	require.NoError(t, err)
	require.Equal(t, string(out), "Generating empty CSV template file: test.csv\n")
}

func TestCheckCSVCommand(t *testing.T) {
	out, err := exec.Command(preflightBinPath, "check", "test.csv").CombinedOutput()
	require.NoError(t, err)
	require.Equal(t, string(out), "Checking CSV file: test.csv\nCSV validation successful\n\n")
}

func TestCheckCSVCommandFailure(t *testing.T) {
	_, err := exec.Command(preflightBinPath, "generate", "test_error.csv").CombinedOutput()
	require.NoError(t, err)

	f, err := os.OpenFile("test_error.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	require.NoError(t, err)

	_, err = f.WriteString("\nxx")
	require.NoError(t, err)
	f.Close()

	out, err := exec.Command(preflightBinPath, "check", "test_error.csv").CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(out), "Pre-flight check failed")

	re := regexp.MustCompile(`Generating error file: (\S+)`)
	matches := re.FindStringSubmatch(string(out))
	require.Equal(t, 2, len(matches))
	filename := matches[1]

	content, err := os.ReadFile(filename)

	require.NoError(t, err)
	require.Contains(t, string(content), "Invalid Serial number")
}

func TestVersionCommand(t *testing.T) {
	out, err := exec.Command(preflightBinPath, "version").CombinedOutput()
	require.NoError(t, err)
	require.Contains(t, string(out), "Version")
}

func TestHelpCommand(t *testing.T) {
	out, err := exec.Command(preflightBinPath, "help").CombinedOutput()
	require.NoError(t, err)
	require.Contains(t, string(out), "Usage")
}

func TestUsage(t *testing.T) {
	out, err := exec.Command(preflightBinPath).CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(out), "Usage")
}
