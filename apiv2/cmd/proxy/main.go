// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/open-edge-platform/infra-core/apiv2/v2/internal/common"
	"github.com/open-edge-platform/infra-core/apiv2/v2/internal/proxy"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/oam"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/tracing"
)

var log = logging.GetLogger("proxy-api")

var errFatal error

var (
	RepoURL   = "https://github.com/open-edge-platform/infra-core/apiv2/v2.git"
	Version   = "<unset>"
	Revision  = "<unset>"
	BuildDate = "<unset>"
)

var (
	wg          = sync.WaitGroup{}        // waitgroup so main will wait for all go routines to exit cleanly
	readyChan   = make(chan bool, 1)      // channel to signal the readiness.
	termChan    = make(chan bool, 1)      // channel to signal termination of main process.
	sigChan     = make(chan os.Signal, 1) // channel to handle any interrupt signals
	restMgrChan = make(chan bool)         // REST manager ready channel
)

const (
	waitTimeout = 10 * time.Second
)

func fatal(e error) {
	errFatal = e
	log.Fatal().Err(e).Msg("fatal error")
}

func printSummary() {
	log.Info().Msg("Starting Proxy API")
	log.Info().Msgf("RepoURL: %s, Version: %s, Revision: %s, BuildDate: %s\n",
		RepoURL, Version, Revision, BuildDate)
}

func traces(cfg *common.GlobalConfig) func(context.Context) error {
	cleanup, exportErr := tracing.NewTraceExporterHTTP(cfg.Traces.TraceURL, "rest-api", nil)
	if exportErr != nil {
		log.Err(exportErr).Msg("Error creating trace exporter")
	}
	if cleanup != nil {
		log.Info().Msg("Tracing enabled")
	} else {
		log.Info().Msg("Tracing disabled")
	}
	return cleanup
}

func setOAM(cfg *common.GlobalConfig, termChan, readyChan chan bool, wg *sync.WaitGroup) {
	if cfg.RestServer.OamServerAddr != "" {
		// Add oam grpc server
		wg.Add(1)
		go func() {
			if err := oam.StartOamGrpcServer(termChan, readyChan, wg,
				cfg.RestServer.OamServerAddr, cfg.Traces.EnableTracing); err != nil {
				fatal(err)
			}
		}()
	}
}

func mainLoop(restMgrChan, readyChan, termChan chan bool, quit chan os.Signal) {
	var restReady, exit bool
	for {
		select {
		case restReady = <-restMgrChan:
			log.Info().Msg("Proxy Ready")
		case <-quit:
			log.Info().Msg("Quit signal")
			exit = true
		}
		// need to go - takes precedence
		// otherwise mainLoop would get blocked trying to `readyChan <- true` again
		if exit {
			log.Info().Msg("Exiting")
			close(termChan)
			break
		}

		// oam server was started
		if restReady && readyChan != nil {
			log.Info().Msg("Ready")
			readyChan <- true
		}
	}
}

func main() {
	// Print a summary of the build
	printSummary()

	defer func() {
		if errFatal != nil {
			os.Exit(1)
		}
	}()

	cfg, err := common.Config()
	if err != nil {
		fatal(err)
	}

	if cfg.Traces.EnableTracing {
		cleanup := traces(cfg)
		if cleanup != nil {
			defer func() {
				cleanErr := cleanup(context.Background())
				if cleanErr != nil {
					log.Err(cleanErr).Msg("Error in tracing cleanup")
				}
			}()
		}
	}

	restMgr, err := proxy.NewManager(cfg, restMgrChan)
	if err != nil {
		fatal(err)
	}

	setOAM(cfg, termChan, readyChan, &wg)

	log.Info().Msg("Starting REST Manager")
	go func() {
		// Make sure to not catch err when restMgr stops
		if err := restMgr.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.InfraErr(err).Msg("failed to start REST Manager")
			fatal(err)
		}
	}()
	defer func() {
		log.Info().Msg("Stopping REST Manager")
		ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
		defer cancel()

		if err := restMgr.Stop(ctx); err != nil {
			log.InfraErr(err).Msg("failed to stop REST Manager")
			fatal(err)
		}
		log.Info().Msg("Stopped REST Manager")
	}()

	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	mainLoop(restMgrChan, readyChan, termChan, sigChan)

	// wait until oam server terminate
	wg.Wait()
	log.Info().Msgf("Shutdown Proxy API")
}
