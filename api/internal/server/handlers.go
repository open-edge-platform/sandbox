// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	api "github.com/open-edge-platform/infra-core/api/pkg/api/v0"
)

type RESTHandlers interface {
	api.ServerInterface
	DispatchAndWait(job *types.Job) types.Response
}

type restHandlers struct {
	jobCh   chan types.Job
	timeout time.Duration
}

func NewHandlers(jobCh chan types.Job, timeout time.Duration) (RESTHandlers, error) {
	return &restHandlers{
		jobCh:   jobCh,
		timeout: timeout,
	}, nil
}

func (r restHandlers) DispatchAndWait(job *types.Job) types.Response {
	var res types.Response
	// create a channel to track the job execution
	c := make(chan struct{})

	// now start listening for a response
	go func() {
		defer close(c)
		log.Trace().Str("jobId", job.ID.String()).Msg("Waiting for response")
		res = waitForResponse(job)
	}()

	// add the job to the executor queue
	log.Trace().Str("jobId", job.ID.String()).Msg("Adding Job to queue")
	r.jobCh <- *job

	// wait for a response (indicated via a channel close)
	<-c
	return res
}

func waitForResponse(job *types.Job) types.Response {
	// wait for a response
	// or the job ctx timeout and return the appropriate response
	select {
	case res := <-job.ResponseCh:
		if res.ID == job.ID {
			log.Trace().Str("jobId", res.ID.String()).Msg("Received response")
			// if res != nil && res.ID == job.ID {
			// we don't need this channel anymore
			// it was reserver for this response
			// close(job.ResponseCh)
			// close is done by the worker/sender
			return *res
		}
		// NOTE this should never happen, but if there is an ID mismatch log and error
		msgStatus := http.StatusInternalServerError
		msgDetail := http.StatusText(msgStatus)
		err := fmt.Errorf("received unexpected response from worker: %v", res)
		log.InfraErr(err).Str("jobId", job.ID.String()).Msgf("unexpected job response")
		return types.Response{
			Payload: types.Payload{
				Data: api.ProblemDetails{
					Message: &msgDetail,
				},
			},
			Status: msgStatus,
			ID:     res.ID,
		}
	case <-job.Context.Done():
		msgStatus := http.StatusRequestTimeout
		msgDetail := http.StatusText(msgStatus)
		log.InfraErr(job.Context.Err()).Str("jobId", job.ID.String()).Msgf("Response timeout")
		return types.Response{
			Payload: types.Payload{
				Data: api.ProblemDetails{
					Message: &msgDetail,
				},
			},
			Status: msgStatus,
			ID:     job.ID,
		}
	}
}

var _ RESTHandlers = &restHandlers{}

// CreateAndExecuteJob is a wrapper for DispatchAndWait function covering context creation and executing a worker's job.
func (r restHandlers) CreateAndExecuteJob(ctx echo.Context, op types.Operation, resource types.Resource,
	body, data, params interface{},
) (*types.Response, error) {
	if body != nil {
		// binding body to context
		if err := r.bind(ctx, body); err != nil {
			problemStatus := http.StatusBadRequest
			problemDetail := "Invalid request body"
			resBind := &types.Response{
				Payload: types.Payload{Data: api.ProblemDetails{
					Message: &problemDetail,
				}},
				Status: problemStatus,
			}
			return resBind, nil
		}
	}

	ctxJob, cancelJob := context.WithTimeout(ctx.Request().Context(), r.timeout)
	defer cancelJob()

	var job *types.Job
	if body != nil {
		job = types.NewJob(ctxJob, op, resource, body, params)
	} else {
		job = types.NewJob(ctxJob, op, resource, data, params)
	}
	// dispatching a job and waiting for execution
	res := r.DispatchAndWait(job)
	return &res, nil
}

// bind performs the binding of the ctx body data into the
// body interface{} format. ctx.Bind method returns http.StatusBadRequest
// in case of error.
func (r *restHandlers) bind(ctx echo.Context, body interface{}) error {
	// Create a custom JSON decoder that disallows unknown fields
	req := ctx.Request()
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()

	// Decode the request body into the provided struct
	if err := decoder.Decode(body); err != nil && !errors.Is(err, io.EOF) {
		log.InfraSec().InfraErr(err).Msgf("REST server could not bind HTTP body")
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body: "+err.Error())
	}
	return nil
}
