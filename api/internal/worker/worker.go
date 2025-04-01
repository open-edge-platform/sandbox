// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"fmt"
	"net/http"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/internal/worker/handlers"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	schedule_cache "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client/cache/schedule"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var log = logging.GetLogger("worker")

// Worker represents the worker that executes the job.
type Worker struct {
	id         int
	WorkerPool chan chan types.Job
	JobChannel chan types.Job
	// quit tells the worker to stop.
	Quit chan bool
	// Done is used to notify that the worker has stopped.
	Done     chan bool
	Handlers handlers.Handlers
}

func NewWorker(
	invClientHandler *clients.InventoryClientHandler,
	hScheduleCache *schedule_cache.HScheduleCacheClient,
	id int,
	workerPool chan chan types.Job,
) *Worker {
	wh := handlers.NewHandlers(invClientHandler, hScheduleCache)

	return &Worker{
		id:         id,
		WorkerPool: workerPool,
		JobChannel: make(chan types.Job),
		Handlers:   wh,
		Quit:       make(chan bool),
		// Done is buffered as it's not the worker problem if someone is listening on the other side
		Done: make(chan bool, 1),
	}
}

// Start method starts the run loop for the worker, listening for a quit channel in
// case we need to stop it.
func (w Worker) Start() {
	go func() {
		log.Debug().Int("id", w.id).Msg("Starting worker")
		for {
			// register the current worker into the worker queue.
			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:
				log.Info().
					Str("jobID", job.ID.String()).
					Str("Operation", string(job.Operation)).
					Str("Resource", string(job.Resource)).
					Int("workerId", w.id).
					Msg("Worker Receives Job")
				// Worker does the job
				res, err := w.Handlers.Do(&job)
				if err != nil {
					log.InfraErr(err).
						Str("jobID", job.ID.String()).
						Int("workerId", w.id).
						Str("Operation", string(job.Operation)).
						Str("Resource", string(job.Resource)).
						Str("Details", errors.ErrorToString(err)).
						Msg("Job response error")

					log.Debug().Msg(errors.ErrorToStringWithDetails(err))
					job.ResponseCh <- HandleError(job, err)
					// if job presents error worker should not end
					close(job.ResponseCh)
					continue
				}
				job.ResponseCh <- res
				log.Info().
					Str("jobID", job.ID.String()).
					Int("workerId", w.id).
					Str("Operation", string(job.Operation)).
					Str("Resource", string(job.Resource)).
					Str("Response", http.StatusText(res.Status)).
					Msg("Worker Job response sent")

				log.Debug().
					Str("data", fmt.Sprintf("%s", res.Payload.Data)).
					Msg("Worker Job response sent")

				close(job.ResponseCh)

			case <-w.Quit:
				log.Info().Int("workerId", w.id).Msg("Worker Job stopping")
				// we have received a signal to stop,
				// we close the jobChannel so that Dispatch can't send new messages to this worker
				// and we notify on the Done channel
				w.Done <- true
				close(w.JobChannel)
				return
			}
		}
	}()
}

// Stop signals the worker to stop listening for work requests.
func (w Worker) Stop() {
	go func() {
		w.Quit <- true
	}()
}

//nolint:cyclop // high cyclomatic complexity because of the switch-case.
func errorToStatusMessage(job types.Job, err error) string {
	switch {
	// DB-related errors, see https://www.postgresql.org/docs/current/errcodes-appendix.html for explanation.
	case job.Operation == types.Delete && errors.IsForeignKeyConstraintError(err):
		return fmt.Sprintf("%s is already in use in one or more resources and cannot be deleted.", job.Resource)

	case (job.Operation == types.Post || job.Operation == types.Put || job.Operation == types.Patch) &&
		errors.IsUniqueConstraintError(err):
		return fmt.Sprintf("One or more unique fields of %s cannot be set because they are already in use by other resources.",
			job.Resource)

	// Generic Inventory errors.
	case errors.IsNotFound(err):
		return fmt.Sprintf("%s resource was not found.", job.Resource)
	case errors.IsPermissionDenied(err):
		return fmt.Sprintf("Operation on %s is not allowed or %s missing required fields.",
			job.Resource, job.Resource)
	// InvalidArgument, Unauthenticated, AlreadyExists will be handled by default case.

	// generic DB error. We should not expose internal DB issues to external users.
	case errors.IsSQLError(err):
		return "SQL database error"
	default:
		return errors.ErrorToString(err)
	}
}

func HandleError(job types.Job, err error) *types.Response {
	errorStatus := errors.ErrorToHTTPStatus(err)
	errorMsg := errorToStatusMessage(job, err)

	return &types.Response{
		Payload: types.Payload{Data: api.ProblemDetails{
			Message: &errorMsg,
		}},
		Status: errorStatus,
		ID:     job.ID,
	}
}
