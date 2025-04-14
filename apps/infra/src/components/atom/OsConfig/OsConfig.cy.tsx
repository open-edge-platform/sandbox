/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { cyGet } from "@orch-ui/tests";
import { instanceOne, instanceTwo, osTb } from "@orch-ui/utils";
import { OsConfig } from "./OsConfig";
import { OsConfigPom } from "./OsConfig.pom";

const pom = new OsConfigPom();
describe("<OsConfig/>", () => {
  it("should render component", () => {
    cy.mount(<OsConfig />);
    pom.root.should("exist");
    pom.root.should("not.contain.text", osTb.name);
  });
  it("should render component", () => {
    cy.mount(<OsConfig instance={instanceOne} />);
    cyGet("osUpdate").should("not.exist");
  });
  it("should render component", () => {
    cy.mount(<OsConfig instance={instanceTwo} />);
    cyGet("osUpdate").should("exist");
    pom.root.should("contain.text", osTb.name);
  });
  it("should render icon when added", () => {
    cy.mount(<OsConfig instance={instanceTwo} iconOnly />);
    pom.el.icon.should("be.visible");
  });
});
