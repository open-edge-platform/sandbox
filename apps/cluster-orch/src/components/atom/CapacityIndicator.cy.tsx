/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import CapacityIndicator from "./CapacityIndicator";

describe("<CapacityIndicator />", () => {
  describe("render CapacityIndicator", () => {
    it("should render CapacityIndicator", () => {
      cy.mount(<CapacityIndicator name="test indicator" percent={20} />);
      cy.contains("test indicator");
      cy.contains("20");
    });
  });
});
