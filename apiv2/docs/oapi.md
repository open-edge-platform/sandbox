<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->

# Open API Edge Infrastructure Manager definitions

You can build and generate API files by:

```bash
  make generate
```

To visualize the API in a browser, run:

```bash
  make oapi-docs
```

Then open the file `api/openapi/openapi-static-doc.html` in a browser.

And finally, to compile the API into golang source code
(into folder ./pkg/api/$VERSION/),

```bash
  make go-dependency
  make build # or: $ make pkg/api/*/edge-infra-manager-openapi-*.gen.go
```
