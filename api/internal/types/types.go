// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"

	"github.com/google/uuid"
)

type Payload struct {
	// based on the Operation and the Resource we know the type of this interface,
	// we can then cast it to the appropriate value
	// on a Job.Payload it represents (and it is possibly nil):
	// - the body (for PUT and POST), eg: api.VirtualMachineRequest
	// - the query (for GET), eg: api.GetHardwareComputeParams
	// on a Response.Payload it represents the response data, eg: *[]api.VirtualMachine
	Data interface{}
	// this is a struct containing all the URL parameters for a specific resource, eg: handlers.VirtualMachineURLParams
	Params interface{}
}

func NewPayload(data, params interface{}) Payload {
	return Payload{
		Data:   data,
		Params: params,
	}
}

// Job represents the job to be run.
type Job struct {
	Context    context.Context
	Payload    Payload
	ResponseCh chan *Response
	Operation  Operation
	Action     Action
	Resource   Resource
	ID         uuid.UUID
}

func NewJob(
	ctx context.Context,
	op Operation,
	res Resource,
	data interface{},
	params interface{},
) *Job {
	payload := NewPayload(data, params)
	return &Job{
		Context:    ctx,
		Payload:    payload,
		ResponseCh: make(chan *Response, 1),
		Operation:  op,
		Resource:   res,
		ID:         uuid.New(),
	}
}

type Response struct {
	Payload Payload
	Status  int
	ID      uuid.UUID
}

type Operation string

const (
	Get    Operation = "GET"
	Post   Operation = "POST"
	Put    Operation = "PUT"
	Patch  Operation = "PATCH"
	Delete Operation = "DELETE"
	List   Operation = "LIST"
)

type Action uint8

const (
	ActionUnspecified Action = iota
	// HostActionRegister creates a host and sets it up to become registered.
	HostActionRegister
	// HostActionOnboard sets up a host to become registered.
	HostActionOnboard
	// HostActionInvalidate sets resource to untrusted. Used for Host only.
	HostActionInvalidate
	InstanceActionInvalidate
)

type Resource string

const (
	OU                      Resource = "OU"
	Locations               Resource = "Locations"
	Region                  Resource = "Region"
	Site                    Resource = "Site"
	Host                    Resource = "Host"
	RepeatedSched           Resource = "RepeatedSched"
	SingleSched             Resource = "SingleSched"
	OSResource              Resource = "OSResource"
	Workload                Resource = "Workload"
	WorkloadMember          Resource = "WorkloadMember"
	Instance                Resource = "Instance"
	TelemetryLogsGroup      Resource = "TelemetryLogsGroup"
	TelemetryMetricsGroup   Resource = "TelemetryMetricsGroup"
	TelemetryLogsProfile    Resource = "TelemetryLogsProfile"
	TelemetryMetricsProfile Resource = "TelemetryMetricsProfile"
	Provider                Resource = "Provider"
	LocalAccount            Resource = "LocalAccount"
)
