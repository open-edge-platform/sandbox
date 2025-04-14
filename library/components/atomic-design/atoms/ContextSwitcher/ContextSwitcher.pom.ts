/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { CyPom } from "@orch-ui/tests";

const dataCySelectors = ["all"] as const;
type Selectors = (typeof dataCySelectors)[number];

export class ContextSwitcherPom extends CyPom<Selectors> {
  constructor(public rootCy: string = "contextSwitcher") {
    super(rootCy, [...dataCySelectors]);
  }

  getTabButton(name: string) {
    return this.root.find(`[data-cy='${name}']`).first();
  }

  getActiveTab() {
    return this.root.find(".active").first();
  }
}
