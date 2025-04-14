/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { cm } from "@orch-ui/apis";
import {
  EChartBar,
  EChartBarSeries,
  EChartBarSeriesItem,
} from "@orch-ui/components";
import { EChartColorSetNames, SharedStorage } from "@orch-ui/utils";
import { useEffect, useState } from "react";

type IndicatorType = cm.ClusterInfo["providerStatus"] extends {
  indicator: infer I;
}
  ? I
  : undefined;
type ClusterStatusTotals = {
  [key in IndicatorType as string]: number;
};
type ClusterStatusColor = {
  [key in IndicatorType as string]: string;
};

const clusterStatusColor: ClusterStatusColor = {
  init: EChartColorSetNames.moss,
  creating: EChartColorSetNames.daisyShade1,
  reconciling: EChartColorSetNames.carbon,
};

const ClustersBarChart = () => {
  const projectName = SharedStorage.project?.name ?? "";
  const { data: clustersResponse } =
    cm.useGetV2ProjectsByProjectNameClustersQuery(
      {
        projectName,
      },
      {
        skip: !projectName,
      },
    );

  const [barSeries, setBarSeries] = useState<EChartBarSeries<number>>({
    data: new Map<string, EChartBarSeriesItem<number>[]>(),
    categories: [
      "init",
      "creating",
      "reconciling",
      "active",
      "removing",
      "inactive",
      "error",
    ],
  });

  const updateBarSeries = (totals: ClusterStatusTotals) => {
    const clusterSeries: EChartBarSeriesItem<number>[] = [];

    barSeries.categories.forEach((status: string) => {
      clusterSeries.push({
        name: status,
        value: totals[status],
        color: clusterStatusColor[status],
      });
    });

    //clear out old data
    barSeries.data.clear();
    const updatedBarSeries = { ...barSeries };
    updatedBarSeries.data.set("Clusters", clusterSeries);
    setBarSeries(updatedBarSeries);
  };

  useEffect(() => {
    if (!clustersResponse || !clustersResponse.clusters) return;

    let newTotals: ClusterStatusTotals = { Active: 0, Unkown: 0, Updating: 0 };
    newTotals = clustersResponse.clusters.reduce(
      (accumulator: ClusterStatusTotals, cluster: cm.ClusterInfo) => {
        accumulator[
          cluster.providerStatus?.indicator ?? "STATUS_INDICATION_UNSPECIFIED"
        ]++;
        return accumulator;
      },
      newTotals,
    );
    updateBarSeries(newTotals);
  }, [clustersResponse]);

  return (
    <EChartBar
      dataCy="clustersBarChart"
      width="100%"
      height="200px"
      showLabel={true}
      showTitle={false}
      series={barSeries}
    />
  );
};

export default ClustersBarChart;
