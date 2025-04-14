/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { CyPom } from "@orch-ui/tests";

const dataCy = "clusterPerformanceCard";
const dataCySelectors = [
  `${dataCy}Title`,
  `${dataCy}Body`,
  `${dataCy}Chart`,
] as const;
type Selectors = (typeof dataCySelectors)[number];

export class ClusterPerformanceCardPom extends CyPom<Selectors> {
  constructor(public rootCy: string = dataCy) {
    super(rootCy, [...dataCySelectors]);
  }
}
