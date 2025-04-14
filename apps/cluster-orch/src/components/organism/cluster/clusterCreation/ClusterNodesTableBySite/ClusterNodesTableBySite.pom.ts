/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { TablePom } from "@orch-ui/components";
import { SiTablePom } from "@orch-ui/poms";
import { CyPom } from "@orch-ui/tests";
import ClusterNodeDetailsDrawerPom from "../../../../atom/ClusterNodeDetailsDrawer/ClusterNodeDetailsDrawer.pom";
import NodeRoleDropdownPom from "../../../../atom/NodeRoleDropdown/NodeRoleDropdown.pom";

const dataCySelectors = ["hostTableContainer", "rowSelectCheckbox"] as const;
type Selectors = (typeof dataCySelectors)[number];

class ClusterNodesSiteTablePom extends CyPom<Selectors> {
  public nodeRoleDropdown: NodeRoleDropdownPom;
  public nodeDetailsDrawer: ClusterNodeDetailsDrawerPom;
  public hostTable: TablePom;
  public hostTableUtils: SiTablePom;

  constructor(public rootCy: string = "clusterNodeTableBySite") {
    super(rootCy, [...dataCySelectors]);

    this.nodeDetailsDrawer = new ClusterNodeDetailsDrawerPom();
    this.nodeRoleDropdown = new NodeRoleDropdownPom();
    this.hostTable = new TablePom("hostTableContainer");
    this.hostTableUtils = new SiTablePom("hostTableContainer");
  }

  public getRowCheckboxByHostName(name: string) {
    return this.hostTableUtils
      .getRowBySearchText(name)
      .find("[data-cy='rowSelectCheckbox']");
  }
}
export default ClusterNodesSiteTablePom;
