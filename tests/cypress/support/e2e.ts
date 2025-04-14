/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

// ***********************************************************
// This example support/e2e.js is processed and
// loaded automatically before your test files.
//
// This is a great place to put global configuration and
// behavior that modifies Cypress.
//
// You can change the location of this file or turn off
// automatically serving support files with the
// 'supportFile' configuration option.
//
// You can read more here:
// https://on.cypress.io/configuration
// ***********************************************************

// Import commands.js using ES2015 syntax:
import "cypress-mochawesome-reporter/register";
import "../../../apps/infra/src/remotes.ts";
import "./commands";
import "./network-logs";
// eslint-disable-next-line @typescript-eslint/no-var-requires
require("cypress-terminal-report/src/installLogsCollector")({
  collectTypes: [
    // "cons:log",
    // "cons:info",
    // "cons:warn",
    "cons:error",
    "cy:log",
    "cy:xhr",
    "cy:request",
    "cy:intercept",
    "cy:command",
  ],
});
// Alternatively you can use CommonJS syntax:
// require('./commands')
