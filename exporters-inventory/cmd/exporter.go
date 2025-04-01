// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"os/signal"
	"sync"

	"github.com/open-edge-platform/infra-core/exporters-inventory/internal/common"
	"github.com/open-edge-platform/infra-core/exporters-inventory/internal/env"
	"github.com/open-edge-platform/infra-core/exporters-inventory/internal/manager"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/oam"
)

var log = logging.GetLogger("exporter")

var errFatal error

var (
	RepoURL   = "https://github.com/open-edge-platform/infra-core.git"
	Version   = "<unset>"
	Revision  = "<unset>"
	BuildDate = "<unset>"
)

func fatal(e error) {
	log.Fatal().Err(e).Msg("start error")
	errFatal = e
}

func printSummary() {
	log.Info().Msg("Starting Exporter")
	log.Info().
		Msgf("RepoURL: %s, Version: %s, Revision: %s, BuildDate: %s\n", RepoURL, Version, Revision, BuildDate)
}

func mainLoop(mngrChan, readyChan, termChan chan bool, quit chan os.Signal) {
	var mngrReady, exit bool
	for {
		select {
		case mngrReady = <-mngrChan:
			log.Info().Msg("Manager Ready")
		case <-quit:
			exit = true
		}
		// oam server was started
		if mngrReady && readyChan != nil {
			log.Info().Msg("Exporter Ready")
			readyChan <- true
		}
		// need to go
		if exit {
			log.Info().Msg("Exiting")
			close(termChan)
			break
		}
	}
}

func main() {
	printSummary()

	defer func() {
		if errFatal != nil {
			os.Exit(1)
		}
	}()

	env.MustEnsureRequired()
	cfg, err := common.Config()
	if err != nil {
		fatal(err)
	}

	// waitgroup so main will wait for oam server to exit cleanly
	wg := sync.WaitGroup{}
	termChan := make(chan bool)

	mgrChan := make(chan bool)
	mngr, err := manager.NewManager(cfg, mgrChan, termChan)
	if err != nil {
		fatal(err)
	}

	// channel to signal the readiness
	var readyChan chan bool
	if cfg.OAMServer.Address != "" {
		// Add oam grpc server
		wg.Add(1)

		readyChan = make(chan bool)
		go func() {
			if err := oam.StartOamGrpcServer(termChan, readyChan, &wg,
				cfg.OAMServer.Address, cfg.LogLevel.Tracing); err != nil {
				fatal(err)
			}
		}()
	}

	go func() {
		if err := mngr.Start(); err != nil {
			fatal(err)
		}
	}()
	defer func() {
		if err := mngr.Stop(); err != nil {
			fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	mainLoop(mgrChan, readyChan, termChan, quit)

	// wait until oam server terminate
	wg.Wait()
}
