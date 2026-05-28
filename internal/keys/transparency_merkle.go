/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package keys

import (
	"context"
	"encoding/hex"
	"fmt"
)

func (tl *TransparencyLog) rebuildMerkleTree(ctx context.Context) error {
	query := fmt.Sprintf(`SELECT entry_hash FROM %s ORDER BY id ASC`, tl.tableName)
	rows, err := tl.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var entryHash string
		if err := rows.Scan(&entryHash); err != nil {
			return err
		}
		if err := tl.addMerkleHash(entryHash); err != nil {
			return err
		}
	}

	return rows.Err()
}

func (tl *TransparencyLog) addMerkleHash(entryHash string) error {
	hashBytes, err := hex.DecodeString(entryHash)
	if err != nil {
		return fmt.Errorf("failed to decode transparency entry hash: %w", err)
	}
	tl.merkleTree.AddHash(hashBytes)
	return nil
}
