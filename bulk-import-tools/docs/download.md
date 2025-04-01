<!---
   SPDX-FileCopyrightText: (C) 2025 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
-->

# Downloading Released Tools

As part of the Continuous Integration pipeline, both the tools `orch-host-preflight` and `orch-host-bulk-import` are
pushed to release registries. Both the artifacts are available in OCI (Open Container Registry) compliant registries
and it's recommended to use `oras` client to interact with them. So, ensure you have `oras` available if you want to
download these tools. Follow instructions in [public documentation](https://oras.land/docs/installation) to install
`oras` if not already done.

1. **orch-host-preflight** is made available in the public AWS ECR. This can be pulled without any credentials using
a command like below -

   ```bash
     oras pull [NEW_PUBLIC_ECR_REGISTRY]/orch-host-preflight:0.1.0
   ```

2. **orch-host-bulk-import** is however available only in Release Service which is access controlled.
Credentials should be needed to pull this tool. You can choose to login to the registry and then pull or a one-shot
pull without login.

Note that we have identity token based login enabled on Release Service. Hence obtain the token before proceeding with
either of the options.

   ```bash
     RS_AT=eyJ... // Identity Token obtained from https://[RELEASE_SERVICE_URL]/oauth/login in any browser
   ```

Pull with login :

  ```bash
     echo "${RS_AT}" | oras login [RELEASE_SERVICE_URL] --password-stdin
     oras pull [RELEASE_SERVICE_URL]/[PATH_TO_ORCH_INFRA]/file/orch-host-bulk-import:0.1.0
  ```

Pull without login :

  ```bash
     echo "${RS_AT}" | oras pull [RELEASE_SERVICE_URL]/[PATH_TO_ORCH_INFRA]/file/orch-host-bulk-import:0.1.0 --password-stdin
  ```
