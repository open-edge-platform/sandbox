/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { eim } from "@orch-ui/apis";
import { createSlice, PayloadAction } from "@reduxjs/toolkit";

export const hostStatusSliceName = "hostStatusList";

export interface _HostStatusRootState {
  [hostStatusSliceName]: HostStatusList;
}

export interface HostStatusState {
  hostId: string;
  status: eim.HostRead["hostStatus"] | eim.ScheduleStatus;
}
export interface HostStatusList {
  [hostId: string]: eim.HostRead["hostStatus"] | eim.ScheduleStatus;
}
const initialState: HostStatusList = {};

export const hostStatusList = createSlice({
  name: hostStatusSliceName,
  initialState,
  reducers: {
    setHostStatus(
      state: HostStatusList,
      action: PayloadAction<HostStatusState>,
    ) {
      state[action.payload.hostId] = action.payload.status;
    },
  },
});

export const selectHostStatus = (
  state: _HostStatusRootState,
  hostId: string,
) => {
  if (hostId in state.hostStatusList) return state.hostStatusList[hostId];
};

export const { setHostStatus } = hostStatusList.actions;

export const hostStatusReducer = hostStatusList.reducer;
