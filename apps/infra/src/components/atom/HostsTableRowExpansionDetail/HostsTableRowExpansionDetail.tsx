/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { eim, enhancedEimSlice } from "@orch-ui/apis";
import { Flex, TrustedCompute } from "@orch-ui/components";
import { getTrustedComputeCompatibility } from "@orch-ui/utils";
import { ScheduleMaintenanceStatusTag } from "../../molecules/ScheduleMaintenanceStatusTag/ScheduleMaintenanceStatusTag";
import { OsConfig } from "../OsConfig/OsConfig";
import "./HostsTableRowExpansionDetail.scss";
const dataCy = "hostsTableRowExpansionDetail";
interface HostsTableRowExpansionDetailProps {
  host: eim.HostRead;
}
const HostsTableRowExpansionDetail = ({
  host,
}: HostsTableRowExpansionDetailProps) => {
  const className = "hosts-table-row-expansion-detail";
  const cy = { "data-cy": dataCy };

  return (
    <div {...cy} className={className}>
      <Flex cols={[6, 6]}>
        <Flex cols={[3, 9]}>
          <b className={`${className}__label`}>Host ID</b>
          <div className={`${className}__content`} data-cy="hostName">
            <span>{host.name}</span>
            <ScheduleMaintenanceStatusTag
              targetEntity={
                "HostRead" as enhancedEimSlice.ScheduleMaintenanceTargetEntity
              }
              targetEntityType="host"
            />
          </div>
          <b className={`${className}__label`}>UUID</b>
          <div className={`${className}__content`} data-cy="uuid">
            {host.uuid}
          </div>
          <b className={`${className}__label`}>Processor</b>
          <div className={`${className}__content`} data-cy="cpuModel">
            {host.cpuModel}
          </div>
        </Flex>
        <Flex cols={[3, 9]}>
          <b className={`${className}__label`}>Latest Updates</b>
          <div className={`${className}__content`}>
            <OsConfig instance={host.instance} iconOnly />
          </div>
          <b className={`${className}__label`}>Trusted Compute</b>
          <div className={`${className}__content`} data-cy="trustedCompute">
            <TrustedCompute
              trustedComputeCompatible={getTrustedComputeCompatibility(host)}
            ></TrustedCompute>
          </div>
        </Flex>
      </Flex>
    </div>
  );
};

export default HostsTableRowExpansionDetail;
