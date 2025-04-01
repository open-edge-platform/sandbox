<!---
  SPDX-FileCopyrightText: (C) 2025 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->

# OAPI Edge Infrastructure Manager definitions

You can build and generate API files by:

```bash
  make oapi-bundle
  make generate-api
```

Notice, the `oapi-bundle` target uses the `sed` tool, in MacOS this might be problematic,
a solution can be found using gnu-sed in place of sed.

The oapi-bundle target compiles the file `edge-infra-manager-openapi.yaml` into the file
`edge-infra-manager-openapi-all.yaml`. Therefore, `edge-infra-manager-openapi-all.yaml` is a temporary file, if any
edit is to be done, make it in the schemas/paths referenced in the `edge-infra-manager-openapi.yaml` file.

To visualize the API in a browser, run:

```bash
  make oapi-docs
```

Then open the file `api/openapi/edge-infra-manager-openapi-static-doc.html` in a browser.

And finally, to compile the API into golang source code
(into folder ./pkg/api/$VERSION/),

```bash
  make go-dependency
  make build # or: $ make pkg/api/*/edge-infra-manager-openapi-*.gen.go
```

You can also run following target, which will include all the generation steps:

``` bash
  make generate
```
