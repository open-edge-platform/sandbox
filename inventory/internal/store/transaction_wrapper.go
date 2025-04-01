// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
)

var (
	roTransactionFactory = func(tm TransactionManager) txCreator {
		return tm.startReadTransaction
	}
	rwTransactionFactory = func(tm TransactionManager) txCreator {
		return tm.startTransaction
	}
)

type TransactionManager interface {
	startTransaction(ctx context.Context) (*ent.Tx, error)
	startReadTransaction(ctx context.Context) (*ent.Tx, error)
	commitTransaction(tx *ent.Tx) error
	rollbackTransaction(tx *ent.Tx, err error) error
}

type NoResultReturningTxFn func(
	ctx context.Context,
	transactional func(ctx context.Context, tx *ent.Tx) error) error

type SingleResultReturningTxFn[R1 any] func(
	ctx context.Context,
	transactional func(ctx context.Context, tx *ent.Tx) (*R1, error)) (*R1, error)

type DoubleResultReturningTxFn[R1, R2 any] func(
	ctx context.Context,
	transactional func(ctx context.Context, tx *ent.Tx) (*R1, *R2, error)) (*R1, *R2, error)

type txCreator func(ctx context.Context) (*ent.Tx, error)

type txFactory func(manager TransactionManager) txCreator

func ExecuteInTx(tm TransactionManager) NoResultReturningTxFn {
	return withTx(tm, rwTransactionFactory)
}

func ExecuteInRoTx(tm TransactionManager) NoResultReturningTxFn {
	return withTx(tm, roTransactionFactory)
}

func ExecuteInTxAndReturnSingle[R1 any](tm TransactionManager) SingleResultReturningTxFn[R1] {
	return withTxAndRet[R1](tm, rwTransactionFactory)
}

func ExecuteInRoTxAndReturnSingle[R1 any](tm TransactionManager) SingleResultReturningTxFn[R1] {
	return withTxAndRet[R1](tm, roTransactionFactory)
}

func ExecuteInTxAndReturnDouble[R1, R2 any](tm TransactionManager) DoubleResultReturningTxFn[R1, R2] {
	return withTxAndRetR1R2[R1, R2](tm, rwTransactionFactory)
}

func ExecuteInRoTxAndReturnDouble[R1, R2 any](tm TransactionManager) DoubleResultReturningTxFn[R1, R2] {
	return withTxAndRetR1R2[R1, R2](tm, roTransactionFactory)
}

func withTx(tm TransactionManager, txFactory txFactory) NoResultReturningTxFn {
	return func(ctx context.Context, transactional func(ctx context.Context, tx *ent.Tx) error) error {
		tx, err := txFactory(tm)(ctx)
		if err != nil {
			return err
		}

		if err := transactional(ctx, tx); err != nil {
			return tm.rollbackTransaction(tx, err)
		}

		return tm.commitTransaction(tx)
	}
}

func withTxAndRet[R1 any](tm TransactionManager, txFactory txFactory) SingleResultReturningTxFn[R1] {
	return func(c context.Context, t func(ctx context.Context, tx *ent.Tx) (*R1, error)) (*R1, error) {
		var r1 *R1
		err := withTx(tm, txFactory)(
			c,
			func(ctx context.Context, tx *ent.Tx) error {
				t1, err := t(ctx, tx)
				if err != nil {
					return err
				}
				r1 = t1
				return nil
			},
		)
		return r1, err
	}
}

func withTxAndRetR1R2[R1, R2 any](tm TransactionManager, txFactory txFactory) DoubleResultReturningTxFn[R1, R2] {
	return func(c context.Context, t func(ctx context.Context, tx *ent.Tx) (*R1, *R2, error)) (*R1, *R2, error) {
		var r1 *R1
		var r2 *R2
		err := withTx(tm, txFactory)(
			c,
			func(ctx context.Context, tx *ent.Tx) error {
				t1, t2, err := t(ctx, tx)
				if err != nil {
					return err
				}
				r1 = t1
				r2 = t2
				return nil
			},
		)
		return r1, r2, err
	}
}
