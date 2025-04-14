/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { CyPom } from "@orch-ui/tests";
import SshHostsTablePom from "../../atoms/SshHostsTable/SshHostsTable.pom";

const dataCySelectors = [
  "sshKeyUsername",
  "sshPublicKey",
  "copySshButton",
  "cancelFooterBtn",
] as const;
type Selectors = (typeof dataCySelectors)[number];

class SshKeysViewDrawerPom extends CyPom<Selectors> {
  sshHostTablePom: SshHostsTablePom;
  constructor(public rootCy: string = "sshKeysViewDrawer") {
    super(rootCy, [...dataCySelectors]);
    this.sshHostTablePom = new SshHostsTablePom();
  }

  getDrawerBase() {
    return this.root.find(".spark-drawer-base");
  }

  getHeaderCloseButton() {
    return this.root.find("[data-testid='drawer-header-close-btn']");
  }
}
export default SshKeysViewDrawerPom;
