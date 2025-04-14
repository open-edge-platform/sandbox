/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { cyGet } from "@orch-ui/tests";
import { clusterOne } from "@orch-ui/utils";
import DeauthorizeNodeConfirmationDialog from "./DeauthorizeNodeConfirmationDialog";
import DeauthorizeNodeConfirmationDialogPom from "./DeauthorizeNodeConfirmationDialog.pom";

const pom = new DeauthorizeNodeConfirmationDialogPom();
describe("<DeauthorizeNodeConfirmationDialog/>", () => {
  describe("basic functionality", () => {
    beforeEach(() => {
      cy.mount(
        <DeauthorizeNodeConfirmationDialog
          name={clusterOne.name!}
          hostName="host-dh38bjw9"
          hostId="host-dh38bjw9"
          hostUuid="4c4c4544-0044-4210-8031-c2c04f305233"
          setDeauthorizeConfirmationOpen={() => {}}
          isDeauthConfirmationOpen
          deauthorizeHostFn={undefined}
          setErrorInfo={() => {}}
        />,
      );
    });
    it("should render component", () => {
      pom.root.should("exist");

      cy.get(".deauthorize-node-confirmation-dialog > p").should(
        "contain.text",
        "host-dh38bjw9",
      );

      pom.interceptApis([pom.api.putClusterNode]);
      cy.get(".spark-modal-footer").contains("Deauthorize").click();

      cy.get(`@${pom.api.putClusterNode}`)
        .its("request.url")
        .then((url) => {
          const match = url.match(`.v2.*clusters/${clusterOne.name}/nodes`);
          // TODO: Deauth flow to be updated with new API
          expect(match && match.length > 0).to.eq(true);
        });

      cy.get("#pathname").contains("/infrastructure/deauthorized-hosts");
    });
  });

  describe("calls deathorize host function", () => {
    it("after pressing the confirmation button", () => {
      cy.mount(
        <DeauthorizeNodeConfirmationDialog
          name={clusterOne.name!}
          hostName="host-dh38bjw9"
          hostId="host-dh38bjw9"
          hostUuid="4c4c4544-0044-4210-8031-c2c04f305233"
          setDeauthorizeConfirmationOpen={() => {}}
          isDeauthConfirmationOpen
          deauthorizeHostFn={cy.stub().as("deauthorizeHostStub")}
          setErrorInfo={() => {}}
        />,
      );
      cyGet("confirmBtn").click();
      cy.get("@deauthorizeHostStub").should("have.been.called");
    });
  });
});
