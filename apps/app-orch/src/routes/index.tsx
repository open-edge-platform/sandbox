/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { AuthWrapper } from "@orch-ui/components";
import { Navigate, RouteObject, useRoutes } from "react-router-dom";
import Layout from "../components/templates/Layout";
import { childRoutes } from "./routes";

const baseUrl = process.env.NODE_ENV === "development" ? "/applications" : "/";

export const routes: RouteObject[] = [
  {
    path: "/",
    element: <Layout />,
    children: childRoutes,
  },
];

if (process.env.NODE_ENV === "development") {
  routes.unshift({ index: true, element: <Navigate to={baseUrl} /> });
}

export default () => {
  const Routes = useRoutes(routes);
  return <AuthWrapper>{Routes}</AuthWrapper>;
};
