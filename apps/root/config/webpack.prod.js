/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

const { merge } = require("webpack-merge");
const ModuleFederationPlugin = require("webpack/lib/container/ModuleFederationPlugin");
const commonConfig = require("./webpack.common");

const prodConfig = {
  mode: "production",
  devtool: false,
  module: {
    rules: [
      {
        test: /.*\.pom.tsx?$/,
        use: [{ loader: "ignore-loader" }],
      },
      {
        test: /.*\.cy.tsx?$/,
        use: [{ loader: "ignore-loader" }],
      },
      {
        test: /\.tsx?$/,
        exclude: /(node_modules)/,
        use: [
          {
            loader: "webpack-remove-code-blocks",
          },
        ],
      },
    ],
  },
  output: {
    filename: "[name].[contenthash].js",
    publicPath: "/",
    clean: true,
  },
  plugins: [
    new ModuleFederationPlugin({
      remotes: {
        AppOrchUI: "AppOrchUI@/mfe/applications/remoteEntry.js",
        EimUI: "EimUI@/mfe/infrastructure/remoteEntry.js",
        ClusterOrchUI: "ClusterOrchUI@/mfe/cluster-orch/remoteEntry.js",
        Admin: "Admin@/mfe/admin/remoteEntry.js",
      },
    }),
  ],
};

module.exports = merge(commonConfig, prodConfig);
