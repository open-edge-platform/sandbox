/*
 * SPDX-FileCopyrightText: (C) 2024 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// ***********************************************
// This example commands.js shows you how to
// create various custom commands and overwrite
// existing commands.
//
// For more comprehensive examples of custom
// commands please read more here:
// https://on.cypress.io/custom-commands
// ***********************************************
//
//
// -- This is a parent command --
// Cypress.Commands.add('login', (email, password) => { ... })
//
//
// -- This is a child command --
// Cypress.Commands.add('drag', { prevSubject: 'element'}, (subject, options) => { ... })
//
//
// -- This is a dual command --
// Cypress.Commands.add('dismiss', { prevSubject: 'optional'}, (subject, options) => { ... })
//
//
// -- This will overwrite an existing command --
// Cypress.Commands.overwrite('visit', (originalFn, url, options) => { ... })

import * as path from "path";
import { LogFolder } from "./index";
import { IUser } from "./utilities";
import Loggable = Cypress.Loggable;
import Timeoutable = Cypress.Timeoutable;
import Shadow = Cypress.Shadow;
import Withinable = Cypress.Withinable;
import RequestOptions = Cypress.RequestOptions;

// Login with session
Cypress.Commands.add("login", ({ username, password }: IUser) => {
  // somehow the intel log cannot be loaded in an headless browser, we also don't care about it
  // just replace it with a local image
  cy.intercept("logo**.png", {
    fixture: "logo.png",
  }).as("intelLogo");

  cy.session(
    [username, password],
    () => {
      cy.visit("/");

      cy.url().should("include", "/openid-connect");
      cy.contains("Sign in");

      cy.get("#username").type(username);
      cy.get("#password").type(password);
      cy.get("#kc-login").click();
    },
    {
      validate() {
        cy.visit("/");
        cy.dataCy("profile", { timeout: 10 * 1000 }).click();
        cy.contains(username);
      },
    },
  );
});

Cypress.Commands.add(
  "dataCy",
  (value, options?: Partial<Loggable & Timeoutable & Withinable & Shadow>) => {
    return cy.get(`[data-cy=${value}]`, options);
  },
);

Cypress.Commands.add("execAndSaveOutput", (command: string, file: string) => {
  cy.exec(command, { failOnNonZeroExit: false }).then(({ stdout }) => {
    cy.writeFile(path.join(LogFolder, file), stdout);
  });
});

Cypress.Commands.add(
  "authenticatedRequest",
  ({ method = "GET", url, body }: Partial<RequestOptions>) => {
    cy.getAllSessionStorage({ log: true }).then((result: Cypress.Storable) => {
      expect(Cypress.config().baseUrl).not.to.be.null;
      const oidcStorage = `oidc.user:${Cypress.config().baseUrl!.replace("web-ui", "keycloak")}/realms/master:webui-client`;

      expect(result).not.to.be.null;
      expect(result).to.haveOwnProperty(Cypress.config().baseUrl!);
      expect(result![Cypress.config().baseUrl!]).to.haveOwnProperty(
        oidcStorage,
      );

      const token = JSON.parse(
        result![Cypress.config().baseUrl!][oidcStorage],
      ).id_token;
      Cypress.log({ message: `Token: ${JSON.stringify(token)}` });
      expect(token).not.to.be.null;

      return cy.request({
        method: method,
        url: `${Cypress.config().baseUrl!.replace("web-ui", "api")}${url}`,
        auth: {
          bearer: token,
        },
        body: body,
        failOnStatusCode: false,
      });
    });
  },
);

Cypress.Commands.add("currentProject", () => {
  cy.visit("/");
  cy.intercept("/v1/projects?member-role=true").as("getProjects");
  cy.url().should("contain", Cypress.config().baseUrl!);
  cy.contains("Dashboard").should("be.visible");
  cy.dataCy("projectSwitchText").should("not.have.text", "Select Projects");
  cy.wait("@getProjects");
  cy.wait(1000);
  return cy.getAllLocalStorage().then((result) => {
    expect(result).to.haveOwnProperty(Cypress.config().baseUrl!);
    expect(result[Cypress.config().baseUrl!]).to.haveOwnProperty("project");
    cy.log(`${result[Cypress.config().baseUrl!]["project"]}`);
    return cy.wrap<string>(
      JSON.parse(`${result[Cypress.config().baseUrl!]["project"]}`).name,
    );
  });
});

Cypress.Commands.add("waitForPageTransition", () => {
  // wait for the page to be loaded
  cy.dataCy("main").within(() => {
    cy.get(".page").should("have.class", "page-enter-done");
  });
});
