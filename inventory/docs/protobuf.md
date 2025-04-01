<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->

# Working with Protobuf

All resources in the Inventory are modeled using the [Protocol Buffer (protobuf)](https://protobuf.dev/) serialization
format that is language independent, and then presented to other services using gRPC.

See Inventory API definitions in the [api](../api/) folder.

The protobuf format is the canonical format for data in Edge Infrastructure Manager - other representations (such as in
the REST API) are secondary adaptations to the gRPC API provided by the Inventory.

Protobuf file structure and naming follows the
[buf style guide](https://docs.buf.build/best-practices/style-guide), and is enforced by `buf format` and `buf lint`.

Code and docs are generated from protobuf files using the buf tool. See the `buf-*`
targets in the Makefile.

Validation of the contents of messages beyond the basic types is provided using
[protovalidate](protovalidate.md).

Buf can also lint and reformat proto files - if the `buf-lint` target fails,
fix any errors and reformat with `buf format -w`.

Buf also generates documentation on the proto files, in docs/api.

`buf-update` can be used to update the buf modules. But this will apply for the modules that are fetched using the
Buf Schema Registry.
