/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  enhancedEimSlice,
  omSlice as observabilityMonitor,
  tmSlice as tdmApi,
} from "@orch-ui/apis";
import {
  Action,
  combineReducers,
  configureStore,
  ThunkAction,
} from "@reduxjs/toolkit";
import { setupListeners } from "@reduxjs/toolkit/query";
import notificationStatusReducer from "./notifications";

const rootReducer = combineReducers({
  notificationStatusList: notificationStatusReducer,
  [enhancedEimSlice.miEnhancedApi.reducerPath]:
    enhancedEimSlice.miEnhancedApi.reducer,
  [observabilityMonitor.reducerPath]: observabilityMonitor.reducer,
  [tdmApi.reducerPath]: tdmApi.reducer,
});

export const setupStore = (preloadedState?: RootState) => {
  return configureStore({
    reducer: rootReducer,
    middleware: (getDefaultMiddleware) =>
      getDefaultMiddleware({
        serializableCheck: false,
        immutableCheck: false,
      })
        .concat(enhancedEimSlice.miEnhancedApi.middleware)
        .concat(observabilityMonitor.middleware)
        .concat(tdmApi.middleware),
    preloadedState,
  });
};

export const store = configureStore({
  reducer: rootReducer,
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware({
      serializableCheck: true,
      immutableCheck: true,
    })
      .concat(enhancedEimSlice.miEnhancedApi.middleware)
      .concat(observabilityMonitor.middleware)
      .concat(tdmApi.middleware),
});

setupListeners(store.dispatch);

// Infer the `RootState` and `AppDispatch` types from the store itself
export type RootState = ReturnType<typeof store.getState>;
// Inferred type: {posts: PostsState, comments: CommentsState, users: UsersState}
export type AppDispatch = typeof store.dispatch;

export type AppThunk<ReturnType = void> = ThunkAction<
  ReturnType,
  RootState,
  unknown,
  Action<string>
>;
