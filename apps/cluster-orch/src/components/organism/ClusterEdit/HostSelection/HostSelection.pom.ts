/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { CyPom } from "@orch-ui/tests";
import ClusterEditAddNodesDrawerPom from "../../../atom/ClusterEditAddNodesDrawer/ClusterEditAddNodesDrawer.pom";
import ClusterEditNodeReviewPom from "../../../atom/ClusterEditNodeReview/ClusterEditNodeReview.pom";

const dataCySelectors = ["site", "region", "confirmBtn"] as const;
type Selectors = (typeof dataCySelectors)[number];

class HostSelectionPom extends CyPom<Selectors> {
  public clusterAddNodeDrawerPom: ClusterEditAddNodesDrawerPom;
  public clusterSelectedNodeReviewTablePom: ClusterEditNodeReviewPom;

  constructor(public rootCy: string = "hostSelection") {
    super(rootCy, [...dataCySelectors]);
    this.clusterSelectedNodeReviewTablePom = new ClusterEditNodeReviewPom();
    this.clusterAddNodeDrawerPom = new ClusterEditAddNodesDrawerPom();
  }

  getAddNodeDrawerBase() {
    return this.clusterAddNodeDrawerPom.root.find(".spark-drawer-base");
  }
}
export default HostSelectionPom;
