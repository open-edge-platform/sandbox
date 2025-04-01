// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package worker_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/open-edge-platform/infra-core/api/internal/common"
	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/internal/worker/handlers"
	inv_handlers "github.com/open-edge-platform/infra-core/api/internal/worker/handlers/invhandlers"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	"github.com/open-edge-platform/infra-core/api/test/utils"
	schedule_cache "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client/cache/schedule"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
)

const (
	UnexistentCode codes.Code      = 10000
	WrongResource  types.Resource  = "WrongResource"
	WrongOperation types.Operation = "WrongOp"

	authKey   = "authorization"
	sampleJWT = "bearer eirhjPHeoH.eufrgwiegr"
)

type MockHandlers struct {
	HandleCallCount int
	HandleCalls     []*types.Job
	HandleResponse  *types.Response
}

func (m *MockHandlers) Do(job *types.Job) (*types.Response, error) {
	m.HandleCallCount++
	m.HandleCalls = append(m.HandleCalls, job)
	return m.HandleResponse, nil
}

func TestHandler_HandleError(t *testing.T) {
	// use a single worker
	pool := make(chan chan types.Job, 1)

	cfg := common.DefaultConfig()
	assert.NotEqual(t, cfg, nil)
	cfg.Inventory.Retry = false

	mockClient := utils.NewTenantAwareMockInventoryServiceClient(
		utils.MockResponses{},
	)

	client := &clients.InventoryClientHandler{
		InvClient: mockClient.GetInventoryClient(),
	}

	scheduleCache := schedule_cache.NewScheduleCacheClient(mockClient)
	hScheduleCache, err := schedule_cache.NewHScheduleCacheClient(scheduleCache)
	require.NoError(t, err)

	h := handlers.NewHandlers(client, hScheduleCache)
	require.NotNil(t, h)

	worker.NewWorker(client, hScheduleCache, 1, pool)

	w := &worker.Worker{
		WorkerPool: pool,
		JobChannel: make(chan types.Job),
		Handlers:   h,
		Quit:       make(chan bool),
		Done:       make(chan bool, 1),
	}

	w.Start()

	ctx := context.WithValue(context.TODO(), authKey, sampleJWT)
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	// Test job with wrong resource type
	job := types.NewJob(ctx, types.Post, WrongResource, api.Site{}, inv_handlers.SiteURLParams{})

	// once the worker is ready send the job
	// NOTE that we need to read the worker out of the queue
	// otherwise the worker won't be able to register itself back once ready and will block
	jobChannel := <-pool
	jobChannel <- *job

	// wait for the worker to stop before checking the results to avoid race-conditions
	wg := sync.WaitGroup{}
	wg.Add(1)

	var res *types.Response
	go func(wg *sync.WaitGroup) {
		// wait for the job to complete
		res = <-job.ResponseCh
		wg.Done()
	}(&wg)
	wg.Wait()

	assert.Equal(t, http.StatusNotImplemented, res.Status)

	// Test job with wrong Operation
	job = types.NewJob(ctx, WrongOperation, types.Host, nil, nil)

	// once the worker is ready send the job
	// NOTE that we need to read the worker out of the queue
	// otherwise the worker won't be able to register itself back once ready and will block
	jobChannel = <-pool
	jobChannel <- *job

	// wait for the worker to stop before checking the results to avoid race-conditions
	wg = sync.WaitGroup{}
	wg.Add(1)

	go func(wg *sync.WaitGroup) {
		// shut down the worked
		w.Stop()
		// wait for the job to complete
		res = <-job.ResponseCh
		// wait for the worker to stop
		<-w.Done
		wg.Done()
	}(&wg)
	wg.Wait()

	assert.Equal(t, http.StatusNotImplemented, res.Status)
}

func TestHandler_Handle(t *testing.T) {
	// use a single worker
	pool := make(chan chan types.Job, 1)

	h := MockHandlers{
		HandleResponse: &types.Response{
			Status: http.StatusOK,
		},
	}

	cfg := common.DefaultConfig()
	assert.NotEqual(t, cfg, nil)
	cfg.Inventory.Retry = false

	mockClient := utils.NewTenantAwareMockInventoryServiceClient(utils.MockResponses{})
	client := &clients.InventoryClientHandler{
		InvClient: mockClient.GetInventoryClient(),
	}
	scheduleCache := schedule_cache.NewScheduleCacheClient(mockClient)
	hScheduleCache, err := schedule_cache.NewHScheduleCacheClient(scheduleCache)
	require.NoError(t, err)

	worker.NewWorker(client, hScheduleCache, 1, pool)

	w := &worker.Worker{
		WorkerPool: pool,
		JobChannel: make(chan types.Job),
		Handlers:   &h,
		Quit:       make(chan bool),
		Done:       make(chan bool, 1),
	}

	w.Start()

	ctx := context.WithValue(context.TODO(), authKey, sampleJWT)
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	job := types.NewJob(ctx, types.List, types.Site, "test-data", "test-params")

	// once the worker is ready send the job
	// NOTE that we need to read the worker out of the queue
	// otherwise the worker won't be able to register itself back once ready and will block
	jobChannel := <-pool
	jobChannel <- *job

	// wait for the worker to stop before checking the results to avoid race-conditions
	wg := sync.WaitGroup{}
	wg.Add(1)

	var res *types.Response
	go func(wg *sync.WaitGroup) {
		// shut down the worked
		w.Stop()
		// wait for the job to complete
		res = <-job.ResponseCh
		// wait for the worker to stop
		<-w.Done
		wg.Done()
	}(&wg)
	wg.Wait()

	assert.Equal(t, 1, h.HandleCallCount)
	assert.Equal(t, types.List, h.HandleCalls[0].Operation)
	assert.Equal(t, types.Site, h.HandleCalls[0].Resource)
	assert.Equal(t, job.ID, h.HandleCalls[0].ID)
	assert.Equal(t, "test-data", h.HandleCalls[0].Payload.Data)
	assert.Equal(t, "test-params", h.HandleCalls[0].Payload.Params)

	assert.Equal(t, http.StatusOK, res.Status)
}

func TestWorker_Handle(t *testing.T) {
	assertInstance := assert.New(t)
	cfg := common.DefaultConfig()
	assertInstance.NotEqual(cfg, nil)
	// this test will fail undefinetely; we disable the retries
	cfg.Inventory.Address = "localhost:50051"
	cfg.Inventory.Retry = false
	pool := make(chan chan types.Job, 1)

	mockClient := utils.NewTenantAwareMockInventoryServiceClient(utils.MockResponses{})
	client := &clients.InventoryClientHandler{
		InvClient: mockClient.GetInventoryClient(),
	}

	scheduleCache := schedule_cache.NewScheduleCacheClient(mockClient)
	hScheduleCache, err := schedule_cache.NewHScheduleCacheClient(scheduleCache)
	require.NoError(t, err)

	w := worker.NewWorker(client, hScheduleCache, 1, pool)
	assertInstance.NotEqual(w, nil)
}

//nolint:funlen // it's a table-driven test
func Test_handleError(t *testing.T) {
	type args struct {
		err       error
		operation types.Operation
		resource  types.Resource
	}
	tests := []struct {
		name        string
		args        args
		want        int
		wantMessage string
	}{
		{
			name:        "unknown",
			args:        args{err: status.Error(codes.Internal, "test-error")},
			want:        http.StatusInternalServerError,
			wantMessage: "test-error",
		},
		{
			name: "unknown",
			args: args{err: status.Error(codes.Unknown, "test-error")},
			want: http.StatusInternalServerError,
		},
		{
			name: "canceled",
			args: args{err: status.Error(codes.Canceled, "test-error")},
			want: http.StatusNotAcceptable,
		},
		{
			name: "not-found",
			args: args{err: status.Error(codes.NotFound, "test-error")},
			want: http.StatusNotFound,
		},
		{
			name: "invalid-arg",
			args: args{err: status.Error(codes.InvalidArgument, "test-error")},
			want: http.StatusUnprocessableEntity,
		},
		{
			name: "deadline",
			args: args{err: status.Error(codes.DeadlineExceeded, "test-error")},
			want: http.StatusRequestTimeout,
		},
		{
			name: "AlreadyExists",
			args: args{err: status.Error(codes.AlreadyExists, "test-error")},
			want: http.StatusConflict,
		},
		{
			name: "PermissionDenied",
			args: args{err: status.Error(codes.PermissionDenied, "test-error")},
			want: http.StatusForbidden,
		},
		{
			name: "ResourceExhausted",
			args: args{err: status.Error(codes.ResourceExhausted, "test-error")},
			want: http.StatusTooManyRequests,
		},
		{
			name: "FailedPrecondition",
			args: args{err: status.Error(codes.FailedPrecondition, "test-error")},
			want: http.StatusPreconditionFailed,
		},
		{
			name: "Aborted",
			args: args{err: status.Error(codes.Aborted, "test-error")},
			want: http.StatusInternalServerError,
		},
		{
			name: "OutOfRange",
			args: args{err: status.Error(codes.OutOfRange, "test-error")},
			want: http.StatusUnprocessableEntity,
		},
		{
			name: "Unimplemented",
			args: args{err: status.Error(codes.Unimplemented, "test-error")},
			want: http.StatusNotImplemented,
		},
		{
			name: "Unavailable",
			args: args{err: status.Error(codes.Unavailable, "test-error")},
			want: http.StatusServiceUnavailable,
		},
		{
			name: "Unauthenticated",
			args: args{err: status.Error(codes.Unauthenticated, "test-error")},
			want: http.StatusUnauthorized,
		},
		{
			name: "Unexistent",
			args: args{err: status.Error(UnexistentCode, "test-error")},
			want: http.StatusInternalServerError,
		},
		{
			name: "ForeignKeyConstraint_OS",
			args: args{
				err:       errors.Errorfc(codes.FailedPrecondition, "violates foreign key constraint"),
				resource:  types.OSResource,
				operation: types.Delete,
			},
			want:        http.StatusPreconditionFailed,
			wantMessage: "OSResource is already in use in one or more resources and cannot be deleted.",
		},
		{
			name: "ForeignKeyConstraint_Host",
			args: args{
				err:       errors.Errorfc(codes.FailedPrecondition, "violates foreign key constraint"),
				resource:  types.Host,
				operation: types.Delete,
			},
			want:        http.StatusPreconditionFailed,
			wantMessage: "Host is already in use in one or more resources and cannot be deleted.",
		},
		{
			name: "UniqueKeyConstraint_Put",
			args: args{
				err:       errors.Errorfc(codes.FailedPrecondition, "violates unique constraint"),
				resource:  types.Host,
				operation: types.Put,
			},
			want:        http.StatusPreconditionFailed,
			wantMessage: "One or more unique fields of Host cannot be set because they are already in use by other resources.",
		},
		{
			name: "UniqueKeyConstraint_Patch",
			args: args{
				err:       errors.Errorfc(codes.FailedPrecondition, "violates unique constraint"),
				resource:  types.Host,
				operation: types.Patch,
			},
			want:        http.StatusPreconditionFailed,
			wantMessage: "One or more unique fields of Host cannot be set because they are already in use by other resources.",
		},
		{
			name: "UniqueKeyConstraint_Post",
			args: args{
				err:       errors.Errorfc(codes.FailedPrecondition, "violates unique constraint"),
				resource:  types.Host,
				operation: types.Post,
			},
			want:        http.StatusPreconditionFailed,
			wantMessage: "One or more unique fields of Host cannot be set because they are already in use by other resources.",
		},
		{
			name: "NotFound",
			args: args{
				err:       errors.Errorfc(codes.NotFound, ""),
				resource:  types.Host,
				operation: types.Get,
			},
			want:        http.StatusNotFound,
			wantMessage: "Host resource was not found.",
		},
		{
			name: "PermissionDenied",
			args: args{
				err:       errors.Errorfc(codes.PermissionDenied, ""),
				resource:  types.Host,
				operation: types.Get,
			},
			want:        http.StatusForbidden,
			wantMessage: "Operation on Host is not allowed or Host missing required fields.",
		},
		{
			name: "Other SQL error",
			args: args{
				err:       errors.Errorfc(codes.Internal, "SQLSTATE"),
				resource:  types.Host,
				operation: types.Get,
			},
			want:        http.StatusInternalServerError,
			wantMessage: "SQL database error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testID := uuid.New()
			res := worker.HandleError(types.Job{
				ID:        testID,
				Resource:  tt.args.resource,
				Operation: tt.args.operation,
			}, tt.args.err)
			assert.Equalf(t, testID, res.ID, "handleError jobID mismatch")
			assert.Equalf(t, tt.want, res.Status, "handleError reported wrong http error")
			if tt.wantMessage != "" {
				errorDetails, ok := res.Payload.Data.(api.ProblemDetails)
				require.True(t, ok)
				require.NotNil(t, errorDetails.Message)
				assert.Equal(t, tt.wantMessage, *errorDetails.Message)
			}
		})
	}
}
