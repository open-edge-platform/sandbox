// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-core/tenant-controller/internal/invclient"
)

var ehlog = logging.GetLogger("event-handler")

func NewEventDispatcher(ic *invclient.TCInventoryClient, handlers ...IEventHandler) *Dispatcher {
	return &Dispatcher{
		ic:       ic,
		handlers: handlers,
	}
}

// Dispatcher dispatches events incoming from INV to all list of registered event handlers.
type Dispatcher struct {
	ic       *invclient.TCInventoryClient
	handlers []IEventHandler
}

func (e *Dispatcher) dispatch(we *client.WatchEvents) {
	for _, handler := range e.handlers {
		handler.HandleEvent(we)
	}
}

func (e *Dispatcher) Start(termChan chan bool) {
	// TODO: inventory controller should use the Reconciler.
	go func() {
		for {
			select {
			case we, ok := <-e.ic.Watcher:
				if !ok {
					ehlog.Fatal().Msgf("inventory notification channel has been closed, terminating event handler: %v", we)
				}
				ehlog.Debug().Msgf("inventory event received: %v", we)
				e.dispatch(we)
			case <-termChan:
				ehlog.Debug().Msg("Event dispatcher has been terminated")
				return
			}
		}
	}()
}

type IEventHandler interface {
	HandleEvent(we *client.WatchEvents)
}
