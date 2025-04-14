/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { cm } from "@orch-ui/apis";
import { CyApiDetails, CyPom } from "@orch-ui/tests";
import { clusterOne } from "@orch-ui/utils";

const dataCySelectors = ["name", "status", "hosts", "link"] as const;
type Selectors = (typeof dataCySelectors)[number];

type ApiAliases = "cluster" | "clusterMocked" | "cluster500";

const route = "**v2/**/clusters/**";

const endpoints: CyApiDetails<ApiAliases, cm.ClusterDetailInfo> = {
  cluster: {
    route,
  },
  clusterMocked: {
    route,
    statusCode: 200,
    response: clusterOne,
  },
  cluster500: {
    route,
    statusCode: 500,
    response: undefined,
  },
};

class ClusterSummaryPom extends CyPom<Selectors, ApiAliases> {
  constructor(public rootCy: string = "clusterSummary") {
    super(rootCy, [...dataCySelectors], endpoints);
  }
}
export default ClusterSummaryPom;
