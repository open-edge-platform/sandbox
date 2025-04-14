/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */
import { siteRestaurantOne } from "@orch-ui/utils";
import SelectSite from "./SelectSite";
import { SelectSiteForClusterPom } from "./SelectSite.pom";

const pom = new SelectSiteForClusterPom();
describe("<RegionSiteTree/>", () => {
  it("should render component", () => {
    cy.mount(
      <SelectSite
        selectedSite={siteRestaurantOne}
        onSelectedInheritedMeta={() => {}}
      />,
    );
    pom.root.should("exist");
  });
});
