// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package dispatcher

import (
	"context"
	"sync"

	"github.com/open-edge-platform/infra-core/api/internal/common"
	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	schedule_cache "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client/cache/schedule"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var log = logging.GetLogger("dispatcher")

type Dispatcher struct {
	// A pool of workers channels that are registered with the dispatcher
	WorkerPool chan chan types.Job
	// An buffered channel that we can send work requests on.
	// It's important this is buffered so that we don't block the dispatcher
	// if workers are busy
	JobQueue chan types.Job
	cfg      *common.GlobalConfig
	workers  []*worker.Worker
	// quit tells the dispatcher to stop.
	Quit chan bool
	// Ready is used to notify that the dispatcher is ready
	Ready chan bool
	// Context to be used with inventory clients
	ctx context.Context
	// List of worker clients to inventory
	InvClients []*clients.InventoryClientHandler
	// Schedule Cache with related client to inventory
	HScheduleCache *schedule_cache.HScheduleCacheClient
	// Provides signal/wait for dispatcher to quit
	wg *sync.WaitGroup
}

func NewDispatcher(cfg *common.GlobalConfig, ready, quitChan chan bool, wg *sync.WaitGroup) *Dispatcher {
	pool := make(chan chan types.Job, cfg.Worker.MaxWorkers)
	queue := make(chan types.Job, cfg.Worker.MaxJobs)
	return &Dispatcher{
		ctx:        context.Background(),
		cfg:        cfg,
		JobQueue:   queue,
		WorkerPool: pool,
		workers:    []*worker.Worker{},
		InvClients: []*clients.InventoryClientHandler{},
		Ready:      ready,
		Quit:       quitChan,
		wg:         wg,
	}
}

func (d *Dispatcher) startInvClients() error {
	log.Info().Msg("Start Inventory Clients")

	for i := 0; i < d.cfg.Worker.MaxWorkers; i++ {
		invClientHandler, err := clients.NewInventoryClientHandler(d.ctx, d.cfg)
		if err != nil {
			d.InvClients = []*clients.InventoryClientHandler{}
			log.InfraErr(err).Msgf("Failed to create new inventory client for worker %d", i)
			return err
		}
		d.InvClients = append(d.InvClients, invClientHandler)
	}
	err := error(nil)
	scheduleCache, err := schedule_cache.NewScheduleCacheClientWithOptions(d.ctx,
		schedule_cache.WithInventoryAddress(d.cfg.Inventory.Address),
		schedule_cache.WithEnableTracing(d.cfg.Traces.EnableTracing),
	)
	if err != nil {
		d.HScheduleCache = nil
		log.InfraErr(err).Msg("Failed to create new inventory client for schedule cache")
		return err
	}
	d.HScheduleCache, err = schedule_cache.NewHScheduleCacheClient(scheduleCache)
	if err != nil {
		d.HScheduleCache = nil
		log.InfraErr(err).Msg("Failed to create new inventory client for h schedule cache")
		return err
	}
	return nil
}

func (d *Dispatcher) endInvClients() {
	log.Info().Msg("End Inventory Clients")

	for i := 0; i < len(d.InvClients); i++ {
		invClientHandler := d.InvClients[i]
		err := invClientHandler.InvClient.Close()
		if err != nil {
			log.InfraErr(err).Msgf("Failed to close inventory client %d", i)
		}
	}
	d.InvClients = []*clients.InventoryClientHandler{}
	if d.HScheduleCache != nil {
		err := d.HScheduleCache.Close()
		if err != nil {
			log.InfraErr(err).Msg("Failed to close inventory client for schedule cache")
		}
		d.HScheduleCache = nil
	}
}

func (d *Dispatcher) Run() error {
	log.Info().Msg("Dispatcher Run")

	if len(d.InvClients) == 0 {
		err := d.startInvClients()
		if err != nil {
			return err
		}
	}

	for i := 0; i < d.cfg.Worker.MaxWorkers; i++ {
		invClientHandler := d.InvClients[i]
		workerInstance := worker.NewWorker(invClientHandler, d.HScheduleCache, i, d.WorkerPool)
		workerInstance.Start()
		d.workers = append(d.workers, workerInstance)
	}

	go d.Dispatch()
	d.Ready <- true
	d.wg.Add(1)
	return nil
}

func (d *Dispatcher) Stop() {
	defer d.wg.Done()
	for _, worker := range d.workers {
		worker.Stop()
	}

	// Close inventory clients
	d.endInvClients()
	d.ctx.Done()
	log.Info().Msg("Dispatcher Stopped")
}

func (d *Dispatcher) Dispatch() {
	for {
		select {
		case job := <-d.JobQueue:
			// a job request has been received
			go func(job types.Job) {
				log.Trace().Str("jobId", job.ID.String()).Msg("Dispatching Job")
				// try to obtain a worker job channel that is available.
				// this will block until a worker is idle
				jobChannel := <-d.WorkerPool

				// checks if worker channel is not closed
				// i.e., if worker did not quit after added itself to WorkerPool
				// if jobchannel not open, put job back into queue and return
				if jobChannel == nil {
					d.JobQueue <- job
					return
				}
				// dispatch the job to the worker job channel
				jobChannel <- job
			}(job)

		case <-d.Quit:
			log.Info().Msg("Dispatcher stopping")
			// we have received a signal to stop,
			// and we notify on the Done channel
			d.Stop()
			return
		}
	}
}
