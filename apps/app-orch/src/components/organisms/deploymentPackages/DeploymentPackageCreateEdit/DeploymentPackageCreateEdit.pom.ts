/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { catalog } from "@orch-ui/apis";
import { CyApiDetails, CyPom, defaultActiveProject } from "@orch-ui/tests";
import { CompositeApplicationList as DeploymentPackageList } from "@orch-ui/utils";
import ApplicationTablePom from "../../applications/ApplicationTable/ApplicationTable.pom";
import DeploymentPackageProfileFormPom from "../../profiles/DeploymentPackageProfileForm/DeploymentPackageProfileForm.pom";
import DeploymentPackageCreateEditReviewPom from "../DeploymentPackageCreateEditReview/DeploymentPackageCreateEditReview.pom";
import DeploymentPackageGeneralInfoFormPom from "../DeploymentPackageGeneralInfoForm/DeploymentPackageGeneralInfoForm.pom";

const dataCySelectors = [
  "dpCreateEditStepper",
  "submitButton",
  "cancelBtn",
  "step0NextBtn",
  "step1NextBtn",
  "step2NextBtn",
] as const;
type Selectors = (typeof dataCySelectors)[number];

type ApiAliases = "deploymentPackageCreate" | "deploymentPackageLoad";

const project = defaultActiveProject.name;
const deploymentPackageApiUrl = `**/v3/projects/${project}/catalog/deployment_packages`;

export const deploymentPackageApis: CyApiDetails<ApiAliases> = {
  deploymentPackageCreate: {
    route: deploymentPackageApiUrl,
    method: "POST",
    response: { statusCode: 201 },
  },
  deploymentPackageLoad: {
    route: `${deploymentPackageApiUrl}/**`,
    response: {
      deploymentPackage: DeploymentPackageList[0],
    } as catalog.GetDeploymentPackageResponse,
  },
};

class DeploymentPackageCreateEditPom extends CyPom<Selectors, ApiAliases> {
  public deploymentPackageGeneralInfoFormPom: DeploymentPackageGeneralInfoFormPom; // Step 1 pom
  public appTablePom: ApplicationTablePom; // Step 2 pom
  public deploymentPackageProfilePom: DeploymentPackageProfileFormPom; // Step 3 pom
  public deploymentPackageReviewPom: DeploymentPackageCreateEditReviewPom; // Step 4 pom

  constructor(public rootCy = "deploymentPackageCreateEdit") {
    super(rootCy, [...dataCySelectors], deploymentPackageApis);
    this.deploymentPackageGeneralInfoFormPom =
      new DeploymentPackageGeneralInfoFormPom();
    this.appTablePom = new ApplicationTablePom();
    this.deploymentPackageProfilePom = new DeploymentPackageProfileFormPom();
    this.deploymentPackageReviewPom =
      new DeploymentPackageCreateEditReviewPom();
  }
  public clickNextOnStep(stepIdx: 0 | 1 | 2) {
    return this.el[`step${stepIdx}NextBtn`].click();
  }
  public clickToCancel() {
    this.el.cancelBtn.click();
  }

  public fillDeploymentPackageCreateEditForm(
    deploymentPackage: Partial<catalog.DeploymentPackage>,
    applicationNamesForSelections: string[],
  ) {
    // General Info Flow
    this.deploymentPackageGeneralInfoFormPom.fillGeneralInfoForm(
      deploymentPackage,
    );
    this.clickNextOnStep(0);

    // Application Selection Flow
    applicationNamesForSelections.forEach((applicationNames) => {
      this.appTablePom.getCheckBoxBySearchText(applicationNames).click();
    });
    this.clickNextOnStep(1);

    // Deployment Package Profile flow
    const generatedProfileCy =
      this.deploymentPackageProfilePom.profileList.getProfileEntryByProfileName(
        "Deployment Profile 1",
      );
    generatedProfileCy.should("contain.text", "System generated profile");
    generatedProfileCy
      .find(".spark-badge-text")
      .should("contain.text", "Default");
    this.clickNextOnStep(2);
  }
}
export default DeploymentPackageCreateEditPom;
