/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import { InternalError } from "@orch-ui/utils";
import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { ToastProps } from "@spark-design/react";
import {
  MessageBannerAlertState,
  ToastPosition,
  ToastState,
  ToastVisibility,
} from "@spark-design/tokens";

export interface MessageBannerState {
  showMessage?: boolean;
  messageTitle: string;
  messageBody: string;
  variant: MessageBannerAlertState;
}

interface NotificationsState {
  messageState: MessageBannerState;
  toastState: ToastProps;
  errorInfo?: InternalError;
}

const initialState: NotificationsState = {
  messageState: {
    showMessage: false,
    messageTitle: "",
    messageBody: "",
    variant: MessageBannerAlertState.Info,
  },
  toastState: {
    duration: 3 * 1000,
    canClose: true,
    position: ToastPosition.TopRight,
    visibility: ToastVisibility.Hide,
  },
};

export const notificationStatusList = createSlice({
  name: "notificationStatusList",
  initialState,
  reducers: {
    /* Trigger a message Banner disable after 15 seconds of show */
    disableMessageBanner(state: NotificationsState) {
      state.messageState = {
        ...state.messageState,
        showMessage: false,
      };
    },
    /** This will show a message Banner on top left whenever an action is triggered */
    showMessageNotification(
      state: NotificationsState,
      action: PayloadAction<MessageBannerState>,
    ) {
      // Show message Banner
      state.messageState = { ...action.payload, showMessage: true };
    },
    showToast(
      state: NotificationsState,
      action: PayloadAction<{
        message: string;
        state: ToastState;
      }>,
    ) {
      state.toastState.message = action.payload.message;
      state.toastState.state = action.payload.state;
      state.toastState.visibility = ToastVisibility.Show;
    },
    /** This will hide the toast notification */
    hideToast(state: NotificationsState) {
      state.toastState.visibility = ToastVisibility.Hide;
    },
    setErrorInfo(
      state: NotificationsState,
      action: PayloadAction<InternalError | undefined>,
    ) {
      state.errorInfo = action.payload;
    },
  },
});

export const {
  showMessageNotification,
  disableMessageBanner,
  showToast,
  hideToast,
  setErrorInfo,
} = notificationStatusList.actions;

export default notificationStatusList.reducer;
