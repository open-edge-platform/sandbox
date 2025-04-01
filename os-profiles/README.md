# Edge Infrastructure Manager Operating System (OS) Profiles

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Contribute](#contribute)

## Overview

This folder includes all the officially supported Operating System (OS) profiles for the Edge Manageability framework.
The available profiles are automatically uploaded in the installation of the Edge Manageability Framework via the OS
Resource Manager. Further updates to these profiles are also automatically tracked and uploaded to deployed
orchestrators.
Currently, supported Operating Systems, with their profiles are:

- Canonical's Ubuntu 22.04 with 6.8 Kernel
- Intel's Edge Microvisor Toolkit v 3.0 based on Azure Linux, with 2 variantes, a non Real Time (`nonrt`) and
  Real time (`rt`) kernel.

For more information on OS Profiles please visit TODO LINK.
To learn how to manage OS profiles and manually push them to change elements or testing refer to the
[Manage OS profiles guide](docs/Manage_OS_profiles.md)

## Features

- Definition of Operating Systems available to be used during Edge Node deployment
- Support for different images and security features
- Profiles are Versioned for release mapping
- Support mutable and immutable Operating System
- Support custom OS dependencies and installation procedures via `Platform Bundle`
- Expose Security feature capabilities and known installed packages.

## Contribute

To learn how to contribute to the project, see the contributor's guide//TODOcontributors-guide-url]. The project will
accept contributions through Pull-Requests (PRs). PRs must be built successfully by the CI pipeline, pass linters
verifications and the unit tests.

There are convenience make targets to support activities in OS profiles, you can use `help` to see a list of makefile
targets. The following is a list of makefile targets that support developer activities:

- `lint` to run a list of linting targets

See the [docs](docs) for advanced development topics:

- [Manage OS profiles guide](docs/Manage_OS_profiles.md)

To learn more about internals and software architecture, see
//TODOEdge Infrastructure Manager developer documentation][inframanager-dev-guide-url.
