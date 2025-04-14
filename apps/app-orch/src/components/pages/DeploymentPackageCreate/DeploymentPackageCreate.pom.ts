/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { CyPom } from "@orch-ui/tests";
import DeploymentPackageCreateEditPom from "../../organisms/deploymentPackages/DeploymentPackageCreateEdit/DeploymentPackageCreateEdit.pom";

const dataCySelectors = [] as const;
type Selectors = (typeof dataCySelectors)[number];

class DeploymentPackageCreatePom extends CyPom<Selectors> {
  deploymentPackageCreateEditPom: DeploymentPackageCreateEditPom;
  constructor(public rootCy = "deploymentPackageCreate") {
    super(rootCy, [...dataCySelectors]);
    this.deploymentPackageCreateEditPom = new DeploymentPackageCreateEditPom();
  }
}

export default DeploymentPackageCreatePom;
