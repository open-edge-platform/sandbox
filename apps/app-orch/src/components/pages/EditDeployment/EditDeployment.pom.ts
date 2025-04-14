/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { adm } from "@orch-ui/apis";
import { CyApiDetails, CyPom } from "@orch-ui/tests";
import { deploymentMinimal } from "@orch-ui/utils";

const dataCySelectors = [] as const;
type Selectors = (typeof dataCySelectors)[number];

type GetDeploymentDetailsSuccessApiAliases = "minimalDeploymentDetailsResponse";
type ApiAliases = GetDeploymentDetailsSuccessApiAliases;

const deploymentDetailsApiUrl =
  "**/v1/projects/**/appdeployment/deployments/**";

const getDeploymentDetailsSuccessEndpoints: CyApiDetails<
  GetDeploymentDetailsSuccessApiAliases,
  adm.DeploymentServiceGetDeploymentApiResponse
> = {
  minimalDeploymentDetailsResponse: {
    route: deploymentDetailsApiUrl,
    response: {
      deployment: deploymentMinimal,
    },
  },
};

class EditDeploymentPom extends CyPom<Selectors, ApiAliases> {
  constructor(public rootCy: string = "editDeployment") {
    super(rootCy, [...dataCySelectors], {
      ...getDeploymentDetailsSuccessEndpoints,
    });
  }
}
export default EditDeploymentPom;
