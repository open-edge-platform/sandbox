/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { CyPom } from "@orch-ui/tests";
import { TreeExpanderPom } from "../TreeExpander/TreeExpander.pom";

const dataCySelectors = ["content"] as const;
type Selectors = (typeof dataCySelectors)[number];

export class TreeBranchPom extends CyPom<Selectors> {
  public treeExpander = new TreeExpanderPom();
  constructor(public rootCy: string = "treeBranch") {
    super(rootCy, [...dataCySelectors]);
  }
}
