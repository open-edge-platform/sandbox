<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->

# Inventory Mocks

The Inventory uses [mock](https://github.com/golang/mock) for mocking.
Do not use other mocking frameworks unless you have a good reason.
Do not write mocks manually; generate them instead.

Mock-generating annotation for the interface `AnyInterface` defined in the file `myinterfaces.go` would be:

```golang
    //go:generate mockgen -package mocks -destination=../mocks/myinterfaces_mock.go . AnyInterface
    type AnyInterface interface {

    }
```

The above mock generator can be triggered by the `mock-gen` Makefile target. The target directory for all generated
mocks is `pkg/mocks`; and the target package is `mocks`.
