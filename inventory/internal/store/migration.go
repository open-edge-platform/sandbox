// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"fmt"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
)

// UpdateDefaultTenantIDInTables updates the tenant_id in the specified tables where tenant_id equals '0'.
// It executes the update within a transaction to ensure atomicity, preventing partial updates in case of an error.
func (is *InvStore) UpdateDefaultTenantIDInTables(ctx context.Context, tables []string, tenantID string) error {
	zlog.Debug().Msgf("Populate tenant values in tables: %v", tables)
	err := ExecuteInTx(is)(ctx, func(ctx context.Context, tx *ent.Tx) error {
		for _, table := range tables {
			query := fmt.Sprintf("UPDATE %s SET tenant_id = $1 WHERE tenant_id = '0'", table)
			if _, err := tx.ExecContext(ctx, query, tenantID); err != nil {
				zlog.InfraSec().InfraErr(err).Msgf("Error updating tenant_id in table %s", table)
				return err
			}
		}
		return nil
	})
	if err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return err
	}
	return nil
}
