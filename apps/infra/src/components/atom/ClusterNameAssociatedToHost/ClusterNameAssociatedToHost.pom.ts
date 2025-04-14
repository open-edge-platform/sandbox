/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { eim } from "@orch-ui/apis";
import { CyApiDetails, CyPom } from "@orch-ui/tests";

const dataCySelectors = ["clusterLink", "notAssigned"] as const;
type Selectors = (typeof dataCySelectors)[number];

type ApiAliases = "getClusterName" | "getClusterNameEmpty";
const url = "**/compute/instances/**";

const endpoints: CyApiDetails<ApiAliases> = {
  getClusterName: {
    route: url,
    statusCode: 200,
    response: {
      workloadMembers: [
        {
          kind: "WORKLOAD_MEMBER_KIND_CLUSTER_NODE",
          workload: {
            name: "cluster-1",
          },
        },
      ],
    } as eim.GetV1ProjectsByProjectNameComputeInstancesAndInstanceIdApiResponse,
  },
  getClusterNameEmpty: {
    route: url,
    statusCode: 200,
    response: {
      workloadMembers: [],
    } as eim.GetV1ProjectsByProjectNameComputeInstancesAndInstanceIdApiResponse,
  },
};

class ClusterNameAssociatedToHostPom extends CyPom<Selectors, ApiAliases> {
  constructor(public rootCy: string = "clusterNameAssociatedToHost") {
    super(rootCy, [...dataCySelectors], endpoints);
  }
}
export default ClusterNameAssociatedToHostPom;
