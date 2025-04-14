/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { cm } from "@orch-ui/apis";
import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { RootState } from "../../store";

const initialState: cm.NodeSpec[] = [];
export const nodesSpec = createSlice({
  name: "nodesSpec",
  initialState,
  reducers: {
    setNodesSpec(state: cm.NodeSpec[], action: PayloadAction<cm.NodeSpec[]>) {
      state = action.payload;
      return state;
    },

    clearNodesSpec(state: cm.NodeSpec[]) {
      state = initialState;
      return state;
    },
  },
});

export const getNodesSpec = (state: RootState) => state.nodesSpec;
export const getInitialNodes = () => initialState;

export const { clearNodesSpec, setNodesSpec } = nodesSpec.actions;

export default nodesSpec.reducer;
