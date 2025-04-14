/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { siteOregonPortland } from "@orch-ui/utils";
import { Site } from "./Site";
import { SiteAtomPom } from "./Site.pom";

const pom = new SiteAtomPom();

describe("<Site/>", () => {
  beforeEach(() => {
    const viewHandler = cy.stub().as("viewHandler");
    cy.mount(<Site site={siteOregonPortland} viewHandler={viewHandler} />);
  });

  it("should render component", () => {
    pom.root.should("exist");
  });

  it("should call viewHandler prop from parent on site click", () => {
    pom.el.siteName.should("exist");
    pom.el.siteName.click();
    cy.get("@viewHandler").should("be.called");
  });

  describe("Should handle isSelectable prop", () => {
    it("to render radio button when isSelectable prop is set to true", () => {
      const viewHandler = cy.stub().as("viewHandler");
      const handleOnSiteSelected = cy.stub().as("handleOnSiteSelected");
      cy.mount(
        <Site
          site={siteOregonPortland}
          viewHandler={viewHandler}
          handleOnSiteSelected={handleOnSiteSelected}
          isSelectable
        />,
      );
      pom.el.selectSiteRadio.should("exist");
      pom.el.selectSiteRadio.click();
      cy.get("@handleOnSiteSelected").should("be.called");
    });

    it("to not render radio button when isSelectable prop is set to false", () => {
      const viewHandler = cy.stub().as("viewHandler");
      const handleOnSiteSelected = cy.stub().as("handleOnSiteSelected");
      cy.mount(
        <Site
          site={siteOregonPortland}
          viewHandler={viewHandler}
          handleOnSiteSelected={handleOnSiteSelected}
          isSelectable={false}
        />,
      );
      pom.el.selectSiteRadio.should("not.exist");
    });
  });
});
