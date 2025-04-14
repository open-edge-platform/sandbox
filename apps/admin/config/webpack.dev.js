/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */
const ModuleFederationPlugin = require("webpack/lib/container/ModuleFederationPlugin");
const TsconfigPathsPlugin = require("tsconfig-paths-webpack-plugin");
const { merge } = require("webpack-merge");
const commonConfig = require("./webpack.common");
const openBrowser = require("react-dev-utils/openBrowser");

const mode = "development";
const devConfig = {
  mode: mode,
  devtool: "source-map",
  module: {
    rules: [
      {
        test: /.*\.pom.(ts|tsx)?$/,
        use: [{ loader: "ignore-loader" }],
      },
      {
        test: /.*\.cy.(ts|tsx)?$/,
        use: [{ loader: "ignore-loader" }],
      },
    ],
  },
  resolve: {
    plugins: [new TsconfigPathsPlugin({ configFile: "tsconfig.dev.json" })],
  },
  output: {
    publicPath: process.env.REACT_LP_REMOTE_EP
      ? `http://${process.env.REACT_LP_REMOTE_EP}:8084/`
      : "http://localhost:8084/",
  },
  devServer: {
    port: 8084,
    historyApiFallback: true,
    hot: false,
    open: false,
    watchFiles: ["src/**/*.tsx", "src/**/*.ts", "public/**/*"],
    headers: {
      "Access-Control-Allow-Origin": "*",
      "Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, PATCH, OPTIONS",
      "Access-Control-Allow-Headers":
        "X-Requested-With, content-type, Authorization",
    },
    onListening: function (devServer) {
      if (!devServer) {
        throw new Error("webpack-dev-server is not defined");
      }
      const port = devServer.server.address().port;
      openBrowser(`http://localhost:${port}`);
    },
  },
  optimization: {
    nodeEnv: mode,
  },
  plugins: [
    new ModuleFederationPlugin({
      remotes: {
        AppOrchUI: `AppOrchUI@http://localhost:8081/remoteEntry.js`,
        ClusterOrchUI: `ClusterOrchUI@http://localhost:8083/remoteEntry.js`,
        EimUI: `EimUI@http://localhost:8082/remoteEntry.js`,
      },
    }),
  ],
};

module.exports = merge(commonConfig, devConfig);
