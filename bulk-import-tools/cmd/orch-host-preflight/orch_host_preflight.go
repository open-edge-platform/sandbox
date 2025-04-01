// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/open-edge-platform/infra-core/bulk-import-tools/info"
	"github.com/open-edge-platform/infra-core/bulk-import-tools/internal/files"
	"github.com/open-edge-platform/infra-core/bulk-import-tools/internal/validator"
)

const (
	NUMARGS = 2
)

func main() {
	// Define a flag to handle the --help option
	flag.Parse()

	// Check the command and call the appropriate function
	args := flag.Args()
	checkArgumentCount(args)

	switch args[0] {
	case "generate":
		if err := generateCSV(args[1]); err != nil {
			fmt.Printf("error: %v\n\n", err.Error())
			os.Exit(1)
		}
	case "check":
		if _, err := validator.CheckCSV(args[1]); err != nil {
			fmt.Printf("error: %v\n\n", err.Error())
			os.Exit(1)
		}
		fmt.Print("CSV validation successful\n\n")
	case "help":
		displayHelp()
	case "version":
		fmt.Printf("Version %s\n\n", info.Version)
	default:
		fmt.Printf("error: Unknown command '%s'\n\n", args[0])
		displayHelp()
		os.Exit(1)
	}
}

func checkArgumentCount(args []string) {
	if len(args) < 1 {
		displayHelp()
		os.Exit(1)
	}

	if (args[0] == "generate" || args[0] == "check") && len(args) < NUMARGS {
		fmt.Println("error: Filename required")
		displayHelp()
		os.Exit(1)
	}
}

// generateCSV creates a CSV file with the given filename.
func generateCSV(filename string) error {
	// The CSV generation logic
	fmt.Printf("Generating empty CSV template file: %s\n", filename)
	return files.CreateFile(filename)
}

// displayHelp prints the help information for the utility.
func displayHelp() {
	fmt.Printf("Create an empty template and scrutinize input CSV file for orch-host-bulk-import tool.\n\n")
	fmt.Printf("Usage: orch-host-preflight COMMAND\n\n")
	fmt.Println("Commands:")
	fmt.Println("\tgenerate <output.csv>  Generate a template CSV file with the given filename")
	fmt.Println("\tcheck <input.csv>      Check the contents of the given CSV file")
	fmt.Println("\tversion                Display version information")
	fmt.Printf("\thelp                   Display this help information\n\n")
}
