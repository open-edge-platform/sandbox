<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->

# Inventory Exporter internals

## Packages

The exporter is composed of 5 internal packages:

- kpis: defines the metrics to be exported and their parser into a prometheus format.
- collect: contains the mechanisms to collect metrics from multiple targets, each one of them
  abstracted as a collector. A collector performs the implementation of a `Collector` interface and
  returns a list of KPIs, as specified by the kpis package.
- exporter: implements the instantiation of the prometheus exporter with multiple collectors,
  from where metrics are collected each time the exporter `Retrieve` method is called.
  An exporter defines the address and path from where the prometheus endpoint can be used
  to pull the exporter metrics via an HTTP interface.
- common: defines the overall configuration scheme used to instantiate the exporter and
  its collectors.
- manager: handles the instantiation of exporter and its start/stop functionalities.

## Design and Workflow

The exporter has a simple design, it contains a set of collectors, each collector
has its own way of obtaining measurements, which are translated into a prometheus metrics format
using the kpis package. Every time prometheus scrapes the Inventory exporter (exporter package)
it retrieves metrics from all the collectors (from the collect package) in a prometheus
format (done by kpis package).

In the case of Edge Infrastructure Manager, there is only one collector named inventory. It has a client to the
Inventory component and maintains a cache of Host and Schedule resources. It manages the cache
by periodically (every 10s) pulling info from Inventory as well as updating the cache on a per subscribed event
basis. The collection of metrics takes place using the local inventory cache.
The inventory collector returns the `host_status` and `host_schedule` metrics, where maintenance
calculation is done using the reference time the metrics are collected.

## ToDo

The cache implementation of resources in the inventory collector will be moved out of exporter,
once the inventory client has a cache implementation and there exists a schedule service to perform
the calculation of maintenance status of host resources.
